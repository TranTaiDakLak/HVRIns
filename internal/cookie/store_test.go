package cookie

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestExtractDatr(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain cookie", "c_user=123;datr=abc123_XY-Z;xs=foo", "abc123_XY-Z"},
		{"no datr", "c_user=123;xs=foo", ""},
		{"trailing datr", "datr=ONLY", "ONLY"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractDatr(tt.in); got != tt.want {
				t.Errorf("ExtractDatr(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestAppendDatr_Dedup(t *testing.T) {
	ResetForTest()
	dir := t.TempDir()
	path := filepath.Join(dir, "datr_pool.txt")

	if err := AppendDatr(path, "aaa"); err != nil {
		t.Fatalf("append 1: %v", err)
	}
	if err := AppendDatr(path, "bbb"); err != nil {
		t.Fatalf("append 2: %v", err)
	}
	if err := AppendDatr(path, "aaa"); err != nil { // dup
		t.Fatalf("append dup: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 unique lines, got %d: %v", len(lines), lines)
	}
}

func TestAppendDatr_Empty(t *testing.T) {
	ResetForTest()
	dir := t.TempDir()
	path := filepath.Join(dir, "datr_pool.txt")

	if err := AppendDatr(path, ""); err != nil {
		t.Fatalf("append empty: %v", err)
	}
	if err := AppendDatr(path, "   "); err != nil {
		t.Fatalf("append whitespace: %v", err)
	}

	// File không được tạo khi không có gì ghi
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("file should not exist after empty appends")
	}
}

func TestAppendDatr_LoadsExistingFileOnFirstCall(t *testing.T) {
	ResetForTest()
	dir := t.TempDir()
	path := filepath.Join(dir, "datr_pool.txt")

	// Pre-populate file trực tiếp (giả lập app chạy lại sau restart)
	if err := os.WriteFile(path, []byte("pre1\npre2\n"), 0644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Try append dup với giá trị đã có trong file → không được duplicate
	if err := AppendDatr(path, "pre1"); err != nil {
		t.Fatalf("append: %v", err)
	}
	if err := AppendDatr(path, "new1"); err != nil {
		t.Fatalf("append new: %v", err)
	}

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines (pre1, pre2, new1), got %d: %v", len(lines), lines)
	}
}

func TestLoadInitial_MissingFile(t *testing.T) {
	if lines := LoadInitial(filepath.Join(t.TempDir(), "nope.txt")); lines != nil {
		t.Errorf("expected nil for missing file, got %v", lines)
	}
}

func TestLoadInitial_SkipEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ci.txt")
	_ = os.WriteFile(path, []byte("line1\n\n  \nline2\n"), 0644)

	lines := LoadInitial(path)
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d: %v", len(lines), lines)
	}
}

func TestLoadPool_WarmsCache(t *testing.T) {
	ResetForTest()
	dir := t.TempDir()
	path := filepath.Join(dir, "datr_pool.txt")
	_ = os.WriteFile(path, []byte("loaded1\nloaded2\n"), 0644)

	lines := LoadPool(path)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Append dup với giá trị đã có trong file → skip
	if err := AppendDatr(path, "loaded1"); err != nil {
		t.Fatalf("append: %v", err)
	}

	data, _ := os.ReadFile(path)
	count := len(strings.Split(strings.TrimSpace(string(data)), "\n"))
	if count != 2 {
		t.Errorf("dedup after LoadPool failed: %d lines", count)
	}
}

func TestAppendDatr_Concurrent(t *testing.T) {
	ResetForTest()
	dir := t.TempDir()
	path := filepath.Join(dir, "datr_pool.txt")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = AppendDatr(path, "same")
		}(i)
	}
	wg.Wait()

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Errorf("concurrent dedup failed: expected 1 line, got %d", len(lines))
	}
}

func TestRemoveDatr_RawAndCookieLines(t *testing.T) {
	ResetForTest()
	dir := t.TempDir()
	path := filepath.Join(dir, "datr_pool.txt")
	initial := strings.Join([]string{
		"raw1",
		"c_user=1; datr=cookie1; xs=x",
		"raw2",
		"c_user=2;datr=cookie2;xs=y",
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(initial), 0600); err != nil {
		t.Fatal(err)
	}

	if err := RemoveDatr(path, "raw1"); err != nil {
		t.Fatal(err)
	}
	if err := RemoveDatr(path, "cookie2"); err != nil {
		t.Fatal(err)
	}

	gotBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	if strings.Contains(got, "raw1") || strings.Contains(got, "cookie2") {
		t.Fatalf("removed datr still present: %q", got)
	}
	if !strings.Contains(got, "cookie1") || !strings.Contains(got, "raw2") {
		t.Fatalf("kept datr missing: %q", got)
	}
}

func TestDatrFileQueue_BatchAppendRemove(t *testing.T) {
	dir := t.TempDir()
	poolPath := filepath.Join(dir, "datr_pool.txt")
	initialPath := filepath.Join(dir, "cookie_initial.txt")

	if err := os.WriteFile(poolPath, []byte("old1\nold2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(initialPath, []byte("datr=old1;\nkeep\n"), 0600); err != nil {
		t.Fatal(err)
	}

	q := NewDatrFileQueue([]string{poolPath, initialPath}, time.Hour)
	q.AppendDatr(poolPath, "new1")
	q.AppendDatr(poolPath, "new1")
	q.RemoveDatrEverywhere("old1")
	q.Close()

	poolBytes, err := os.ReadFile(poolPath)
	if err != nil {
		t.Fatal(err)
	}
	pool := string(poolBytes)
	if strings.Contains(pool, "old1") {
		t.Fatalf("old1 still present in pool: %q", pool)
	}
	if strings.Count(pool, "new1") != 1 {
		t.Fatalf("new1 should be appended once: %q", pool)
	}

	initialBytes, err := os.ReadFile(initialPath)
	if err != nil {
		t.Fatal(err)
	}
	initial := string(initialBytes)
	if strings.Contains(initial, "old1") {
		t.Fatalf("old1 still present in initial: %q", initial)
	}
	if !strings.Contains(initial, "keep") {
		t.Fatalf("kept line missing from initial: %q", initial)
	}
}
