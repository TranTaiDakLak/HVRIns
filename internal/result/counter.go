package result

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// StringCounter is a thread-safe frequency map (string key → int count) used to
// track per-session metrics such as Facebook app versions across goroutines.
type StringCounter struct {
	mu     sync.Mutex
	counts map[string]int
}

// Incr increments the count for key by 1.
func (c *StringCounter) Incr(key string) {
	if key == "" {
		return
	}
	c.mu.Lock()
	if c.counts == nil {
		c.counts = make(map[string]int)
	}
	c.counts[key]++
	c.mu.Unlock()
}

// snapshot returns a sorted copy of the current counts (safe to read without lock).
func (c *StringCounter) snapshot() map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	snap := make(map[string]int, len(c.counts))
	for k, v := range c.counts {
		snap[k] = v
	}
	return snap
}

// CounterSet groups the per-session frequency counters and flushes them to the
// session result directory on a ticker so the operator can watch progress in real time.
type CounterSet struct {
	writer       *Writer
	FbAppVersion *StringCounter
	done         chan struct{}
	once         sync.Once
}

// NewCounterSet creates a CounterSet backed by w.
func NewCounterSet(w *Writer) *CounterSet {
	return &CounterSet{
		writer:       w,
		FbAppVersion: &StringCounter{counts: make(map[string]int)},
		done:         make(chan struct{}),
	}
}

// Start launches a background goroutine that flushes all counters every interval.
// The goroutine exits when ctx is cancelled or Stop() is called.
func (cs *CounterSet) Start(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cs.flush()
			case <-ctx.Done():
				cs.flush()
				return
			case <-cs.done:
				cs.flush()
				return
			}
		}
	}()
}

// Stop signals the flush goroutine to exit (idempotent).
func (cs *CounterSet) Stop() {
	cs.once.Do(func() { close(cs.done) })
}

// flush writes all counter snapshots to their result files.
func (cs *CounterSet) flush() {
	if cs.writer == nil || cs.writer.Root() == "" {
		return
	}
	cs.flushCounter("FbAppVersion.txt", cs.FbAppVersion)
}

func (cs *CounterSet) flushCounter(filename string, sc *StringCounter) {
	snap := sc.snapshot()
	if len(snap) == 0 {
		return
	}
	keys := make([]string, 0, len(snap))
	for k := range snap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("%s: %d\n", k, snap[k]))
	}
	path := filepath.Join(cs.writer.Root(), filename)
	_ = os.WriteFile(path, []byte(sb.String()), 0o644)
}
