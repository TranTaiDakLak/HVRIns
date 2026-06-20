package result

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Writer is a thread-safe helper that appends or upserts lines into result files
// within a session output directory.  All methods are no-ops when Root() == "".
type Writer struct {
	root string
	mu   sync.Mutex
}

// NewWriter creates a Writer rooted at dir.
func NewWriter(dir string) *Writer {
	return &Writer{root: dir}
}

// Root returns the session output directory.
func (w *Writer) Root() string {
	if w == nil {
		return ""
	}
	return w.root
}

// Append creates the directory if needed, then appends line (plus newline) to
// root/filename.
func (w *Writer) Append(filename, line string) error {
	if w == nil || w.root == "" {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := os.MkdirAll(w.root, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(w.root, filename), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, line)
	return err
}

// UpsertUID adds line to root/filename, replacing any existing entry whose first
// pipe-delimited field (UID) matches.  If no match is found the line is appended.
func (w *Writer) UpsertUID(filename, line string) error {
	if w == nil || w.root == "" {
		return nil
	}
	uid := uidFromLine(line)
	if uid == "" {
		return w.Append(filename, line)
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := os.MkdirAll(w.root, 0o755); err != nil {
		return err
	}
	path := filepath.Join(w.root, filename)
	existing, _ := readLines(path)
	replaced := false
	for i, l := range existing {
		if uidFromLine(l) == uid {
			existing[i] = line
			replaced = true
			break
		}
	}
	if !replaced {
		existing = append(existing, line)
	}
	return writeLines(path, existing)
}

// uidFromLine returns the first pipe-delimited field (the UID) from a result line.
func uidFromLine(line string) string {
	if idx := strings.IndexByte(line, '|'); idx > 0 {
		return line[:idx]
	}
	return ""
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if t := sc.Text(); t != "" {
			lines = append(lines, t)
		}
	}
	return lines, sc.Err()
}

func writeLines(path string, lines []string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	bw := bufio.NewWriter(f)
	for _, l := range lines {
		if _, err := fmt.Fprintln(bw, l); err != nil {
			return err
		}
	}
	return bw.Flush()
}
