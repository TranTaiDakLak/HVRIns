// Package cookie — lưu trữ cookie initial và datr pool chuẩn trong Config/Cookie/.
//
// Port từ C# (FacebookLogoutSessionUtils + PathSingleton):
//   - cookie_initial.txt   — nguồn input do user paste vào, đọc khi start register
//   - datr_pool.txt        — tự tích lũy mỗi datr mới thu được từ reg thành công
//
// Khác với C# chỉ tự ghi 1 file datr.txt cạnh exe, Go gom cả 2 file vào Config/Cookie/
// để mọi dữ liệu persistent nằm trong một thư mục duy nhất, dễ backup và đồng bộ.
//
// Thread-safe: mutex toàn cục + in-memory set để dedup nhanh không cần đọc lại file.
package cookie

import (
	"bufio"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Embedded default data — ship sẵn khi build, không cần user upload datr.
// Khi user chạy lần đầu trên máy mới → app tự extract ra Config/Cookie/.
//
//go:embed embedded/cookie_initial.txt
var embeddedCookieInitial []byte

//go:embed embedded/datr_pool.txt
var embeddedDatrPool []byte

const (
	// DefaultDir là thư mục mặc định chứa cookie files, tương đối so với working dir (giống C# ./config/cookie_initial/).
	DefaultDir = "Config/Cookie"

	// InitialFilename là tên file input user paste cookie vào.
	InitialFilename = "cookie_initial.txt"

	// PoolFilename là tên file tự tích lũy datr từ reg thành công.
	PoolFilename = "datr_pool.txt"
)

// reDatr match giá trị datr trong cookie string (format FB: `datr=xxx;`).
var reDatr = regexp.MustCompile(`datr=([A-Za-z0-9_-]+)`)

var (
	mu       sync.Mutex
	savedSet = make(map[string]struct{}) // dedup in-memory, tránh đọc lại file mỗi lần append
	loadedDB bool                        // đã load savedSet từ file chưa
)

// DefaultInitialPath trả về đường dẫn mặc định của cookie_initial.txt.
func DefaultInitialPath() string {
	return filepath.Join(DefaultDir, InitialFilename)
}

// DefaultPoolPath trả về đường dẫn mặc định của datr_pool.txt.
func DefaultPoolPath() string {
	return filepath.Join(DefaultDir, PoolFilename)
}

// EnsureDir tạo thư mục Config/Cookie/ nếu chưa tồn tại.
func EnsureDir() error {
	if err := os.MkdirAll(DefaultDir, 0755); err != nil {
		return fmt.Errorf("cookie: create dir %q: %w", DefaultDir, err)
	}
	return nil
}

// ExtractDatr tìm giá trị datr từ 1 dòng text (cookie string hoặc account line).
// Trả về "" nếu không có datr.
func ExtractDatr(line string) string {
	m := reDatr.FindStringSubmatch(line)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// LoadInitial đọc cookie_initial.txt trả về danh sách dòng non-empty.
// Trả về nil slice nếu file chưa tồn tại (không error).
// path rỗng → dùng DefaultInitialPath(), tự seed embedded data nếu chưa có
// để user không cần paste datr — app ship sẵn pool.
func LoadInitial(path string) []string {
	if path == "" {
		path = DefaultInitialPath()
		SeedInitialIfMissing(path)
	}
	return readLines(path)
}

// SeedInitialIfMissing tạo cookie_initial.txt với data embedded nếu chưa có.
// Best-effort — lỗi bỏ qua. Gọi từ LoadInitial hoặc GetDefaultCookiePaths.
func SeedInitialIfMissing(path string) {
	if _, err := os.Stat(path); err == nil {
		return // file đã có — không overwrite
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	content := embeddedCookieInitial
	if len(content) == 0 {
		content = []byte("# Paste each datr value or full cookie line here (one per line)\n")
	}
	_ = os.WriteFile(path, content, 0600)
}

// SeedPoolIfMissing tạo datr_pool.txt với data embedded nếu chưa có.
func SeedPoolIfMissing(path string) {
	if _, err := os.Stat(path); err == nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	if len(embeddedDatrPool) > 0 {
		_ = os.WriteFile(path, embeddedDatrPool, 0600)
	}
}

// LoadPool đọc datr_pool.txt và đồng thời warm savedSet cho dedup.
// Trả về nil slice nếu file chưa tồn tại.
func LoadPool(path string) []string {
	if path == "" {
		path = DefaultInitialPath()
	}
	lines := readLines(path)

	mu.Lock()
	defer mu.Unlock()
	for _, l := range lines {
		d := ExtractDatr(l)
		if d == "" {
			// file datr_pool.txt có thể chứa raw datr value (không có prefix)
			d = strings.TrimSpace(l)
		}
		if d != "" {
			savedSet[d] = struct{}{}
		}
	}
	loadedDB = true
	return lines
}

// AppendDatr ghi 1 datr mới vào datr_pool.txt nếu chưa có.
// Đảm bảo: dedup in-memory, atomic append, idempotent.
// datr rỗng → no-op (không error).
// path rỗng → dùng DefaultPoolPath().
func AppendDatr(path, datr string) error {
	datr = strings.TrimSpace(datr)
	if datr == "" || strings.HasPrefix(datr, "_") || strings.HasPrefix(datr, "-") {
		return nil
	}
	if path == "" {
		path = DefaultInitialPath()
	}

	mu.Lock()
	defer mu.Unlock()

	// Lazy load file vào savedSet lần đầu để không duplicate với entry đã có trên disk.
	if !loadedDB {
		if f, err := os.Open(path); err == nil {
			sc := bufio.NewScanner(f)
			sc.Buffer(make([]byte, 64*1024), 1024*1024)
			for sc.Scan() {
				d := ExtractDatr(sc.Text())
				if d == "" {
					d = strings.TrimSpace(sc.Text())
				}
				if d != "" {
					savedSet[d] = struct{}{}
				}
			}
			_ = f.Close()
		}
		loadedDB = true
	}

	if _, dup := savedSet[datr]; dup {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("cookie: create dir for pool: %w", err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("cookie: open pool file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(datr + "\n"); err != nil {
		return fmt.Errorf("cookie: write datr: %w", err)
	}
	savedSet[datr] = struct{}{}
	return nil
}

// PoolSize trả về số datr đang có trong pool (đã dedup).
// Chỉ chính xác sau khi LoadPool() hoặc AppendDatr() đã warm cache.
// RemoveDatr deletes datr from a pool/initial text file.
// It matches both raw datr lines and full cookie lines containing datr=...
// path empty uses DefaultInitialPath(). Missing file and empty datr are no-op.
func RemoveDatr(path, datr string) error {
	datr = strings.TrimSpace(datr)
	if datr == "" {
		return nil
	}
	if path == "" {
		path = DefaultInitialPath()
	}

	mu.Lock()
	defer mu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			delete(savedSet, datr)
			return nil
		}
		return fmt.Errorf("cookie: open pool file for remove: %w", err)
	}

	var kept []string
	changed := false
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		d := ExtractDatr(line)
		if d == "" {
			d = strings.TrimSpace(line)
		}
		if d == datr {
			changed = true
			continue
		}
		kept = append(kept, line)
	}
	closeErr := f.Close()
	if err := sc.Err(); err != nil {
		return fmt.Errorf("cookie: scan pool file for remove: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("cookie: close pool file for remove: %w", closeErr)
	}

	delete(savedSet, datr)
	if !changed {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("cookie: create dir for remove: %w", err)
	}
	data := ""
	if len(kept) > 0 {
		data = strings.Join(kept, "\n") + "\n"
	}
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		return fmt.Errorf("cookie: rewrite pool file after remove: %w", err)
	}
	return nil
}

func PoolSize() int {
	mu.Lock()
	defer mu.Unlock()
	return len(savedSet)
}

type datrFileOpKind uint8

const (
	datrFileAppend datrFileOpKind = iota + 1
	datrFileRemove
)

type datrFileOp struct {
	kind datrFileOpKind
	path string
	datr string // plain datr value — dùng cho dedup
	line string // dòng thực ghi ra file; nếu rỗng thì dùng datr
}

type datrFileState struct {
	lines   []string
	datrs   map[string]struct{}
	removes map[string]struct{}
	dirty   bool
}

// DatrFileQueue batches datr append/remove operations onto disk.
type DatrFileQueue struct {
	paths         []string
	flushInterval time.Duration
	ops           chan datrFileOp
	done          chan struct{}
	wg            sync.WaitGroup
	once          sync.Once
}

func NewDatrFileQueue(paths []string, flushInterval time.Duration) *DatrFileQueue {
	if flushInterval <= 0 {
		flushInterval = 1500 * time.Millisecond
	}
	seen := make(map[string]struct{}, len(paths)+1)
	normalized := make([]string, 0, len(paths)+1)
	if len(paths) == 0 {
		paths = []string{DefaultInitialPath()}
	}
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		normalized = append(normalized, p)
	}
	q := &DatrFileQueue{
		paths:         normalized,
		flushInterval: flushInterval,
		ops:           make(chan datrFileOp, 8192),
		done:          make(chan struct{}),
	}
	q.wg.Add(1)
	go q.run()
	return q
}

func (q *DatrFileQueue) AppendDatr(path, datr string) {
	if q == nil {
		return
	}
	datr = strings.TrimSpace(datr)
	if datr == "" || strings.HasPrefix(datr, "_") || strings.HasPrefix(datr, "-") {
		return
	}
	if strings.TrimSpace(path) == "" {
		path = DefaultInitialPath()
	}
	q.enqueue(datrFileOp{kind: datrFileAppend, path: path, datr: datr})
}

// AppendDatrToPool ghi datr trực tiếp vào file Pool bằng O_APPEND.
// Không đi qua queue — Pool file là append-only nên không cần batch/dedup phức tạp.
// Concurrent-safe: kernel đảm bảo atomic append cho write nhỏ trên cùng 1 file.
func AppendDatrToPool(path, datr string) {
	datr = strings.TrimSpace(datr)
	if datr == "" || strings.HasPrefix(datr, "_") || strings.HasPrefix(datr, "-") {
		return
	}
	if strings.TrimSpace(path) == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		slog.Error("[Pool] mkdir failed", "path", path, "err", err)
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("[Pool] open failed", "path", path, "err", err)
		return
	}
	defer f.Close()
	if _, err = fmt.Fprintf(f, "%s\n", datr); err != nil {
		slog.Error("[Pool] write failed", "path", path, "err", err)
	}
}

func (q *DatrFileQueue) RemoveDatrEverywhere(datr string) {
	if q == nil {
		return
	}
	datr = strings.TrimSpace(datr)
	if datr == "" {
		return
	}
	q.enqueue(datrFileOp{kind: datrFileRemove, datr: datr})
}

func (q *DatrFileQueue) Close() {
	if q == nil {
		return
	}
	q.once.Do(func() {
		close(q.done)
		q.wg.Wait()
	})
}

func (q *DatrFileQueue) enqueue(op datrFileOp) {
	select {
	case q.ops <- op:
	case <-q.done:
	}
}

func (q *DatrFileQueue) run() {
	defer q.wg.Done()
	ticker := time.NewTicker(q.flushInterval)
	defer ticker.Stop()

	states := make(map[string]*datrFileState)
	for {
		select {
		case op := <-q.ops:
			q.applyOp(states, op)
		case <-ticker.C:
			q.flushDirty(states)
		case <-q.done:
			for {
				select {
				case op := <-q.ops:
					q.applyOp(states, op)
				default:
					q.flushDirty(states)
					return
				}
			}
		}
	}
}

func (q *DatrFileQueue) applyOp(states map[string]*datrFileState, op datrFileOp) {
	switch op.kind {
	case datrFileAppend:
		st := q.stateFor(states, op.path)
		if _, exists := st.datrs[op.datr]; exists {
			return
		}
		line := op.line
		if line == "" {
			line = op.datr
		}
		st.lines = append(st.lines, line)
		st.datrs[op.datr] = struct{}{}
		delete(st.removes, op.datr)
		st.dirty = true
	case datrFileRemove:
		for _, path := range q.paths {
			st := q.stateFor(states, path)
			if _, exists := st.datrs[op.datr]; !exists {
				continue
			}
			delete(st.datrs, op.datr)
			st.removes[op.datr] = struct{}{}
			st.dirty = true
		}
	}
}

func (q *DatrFileQueue) stateFor(states map[string]*datrFileState, path string) *datrFileState {
	path = strings.TrimSpace(path)
	if path == "" {
		path = DefaultInitialPath()
	}
	if st := states[path]; st != nil {
		return st
	}
	st := &datrFileState{datrs: make(map[string]struct{}), removes: make(map[string]struct{})}
	st.lines = readLines(path)
	for _, line := range st.lines {
		d := ExtractDatr(line)
		if d == "" {
			d = strings.TrimSpace(line)
			// strip timestamp suffix "datr|YYYY-MM-DD HH:MM:SS"
			if idx := strings.IndexByte(d, '|'); idx != -1 {
				d = d[:idx]
			}
		}
		if d != "" {
			st.datrs[d] = struct{}{}
		}
	}
	states[path] = st
	return st
}

func (q *DatrFileQueue) flushDirty(states map[string]*datrFileState) {
	for path, st := range states {
		if st == nil || !st.dirty {
			continue
		}
		if len(st.removes) > 0 {
			st.lines = filterDatrLinesSet(st.lines, st.removes)
			clear(st.removes)
		}
		if err := writeDatrLines(path, st.lines); err == nil {
			st.dirty = false
		} else {
			slog.Error("[DatrFileQueue] write failed", "path", path, "err", err)
		}
	}
}

func filterDatrLines(lines []string, datr string) []string {
	out := lines[:0]
	for _, line := range lines {
		d := ExtractDatr(line)
		if d == "" {
			d = strings.TrimSpace(line)
		}
		if d == datr {
			continue
		}
		out = append(out, line)
	}
	return out
}

func filterDatrLinesSet(lines []string, removes map[string]struct{}) []string {
	out := lines[:0]
	for _, line := range lines {
		d := ExtractDatr(line)
		if d == "" {
			d = strings.TrimSpace(line)
		}
		if _, remove := removes[d]; remove {
			continue
		}
		out = append(out, line)
	}
	return out
}

func writeDatrLines(path string, lines []string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("cookie: create dir for queued write: %w", err)
	}
	data := ""
	if len(lines) > 0 {
		data = strings.Join(lines, "\n") + "\n"
	}
	return os.WriteFile(path, []byte(data), 0600)
}

// NewRunPoolPath trả về đường dẫn file pool cho lần chạy hiện tại.
// Pattern: {dir}/Pool{YYYYMMDD}_{N}.txt — N tự tăng dựa trên file đã có trong dir.
// dir rỗng → dùng DefaultDir.
func NewRunPoolPath(dir string) string {
	if dir == "" {
		dir = DefaultDir
	}
	today := time.Now().Format("20060102")
	prefix := "Pool" + today + "_"

	entries, _ := os.ReadDir(dir)
	maxN := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".txt") {
			continue
		}
		inner := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".txt")
		if n, err := strconv.Atoi(inner); err == nil && n > maxN {
			maxN = n
		}
	}
	return filepath.Join(dir, fmt.Sprintf("Pool%s_%d.txt", today, maxN+1))
}

// ResetForTest clears dedup cache — chỉ dùng trong unit test.
func ResetForTest() {
	mu.Lock()
	defer mu.Unlock()
	savedSet = make(map[string]struct{})
	loadedDB = false
}

// readLines đọc file thành []string, bỏ dòng rỗng. File không tồn tại → nil.
func readLines(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}
