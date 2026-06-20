package result

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCounter_IncrAndGet(t *testing.T) {
	c := NewCounter()
	c.Incr("VN")
	c.Incr("VN")
	c.Incr("US")

	if got := c.Get("VN"); got != 2 {
		t.Errorf("VN count = %d, want 2", got)
	}
	if got := c.Get("US"); got != 1 {
		t.Errorf("US count = %d, want 1", got)
	}
	if got := c.Get("ZZ"); got != 0 {
		t.Errorf("ZZ count = %d, want 0", got)
	}
}

func TestCounter_Add(t *testing.T) {
	c := NewCounter()
	c.Add("VN", 10)
	c.Add("VN", 5)
	if got := c.Get("VN"); got != 15 {
		t.Errorf("Add: got %d, want 15", got)
	}
}

func TestCounter_EmptyKey(t *testing.T) {
	c := NewCounter()
	c.Incr("")
	c.Incr("  ")
	if c.Size() != 0 {
		t.Errorf("empty key should be skipped, size=%d", c.Size())
	}
}

func TestCounter_DumpSorted(t *testing.T) {
	c := NewCounter()
	c.Add("VN", 100)
	c.Add("US", 50)
	c.Add("UK", 100) // same count as VN → alphabetical order

	got := c.DumpSorted()
	want := "UK|100\nVN|100\nUS|50\n"
	if got != want {
		t.Errorf("DumpSorted:\ngot  %q\nwant %q", got, want)
	}
}

func TestCounter_DumpSorted_Empty(t *testing.T) {
	c := NewCounter()
	if got := c.DumpSorted(); got != "" {
		t.Errorf("empty counter should dump empty string, got %q", got)
	}
}

func TestCounter_Concurrent(t *testing.T) {
	c := NewCounter()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Incr("VN")
		}()
	}
	wg.Wait()
	if got := c.Get("VN"); got != 100 {
		t.Errorf("concurrent: got %d, want 100", got)
	}
}

func TestCounter_Reset(t *testing.T) {
	c := NewCounter()
	c.Add("VN", 50)
	c.Reset()
	if c.Size() != 0 {
		t.Errorf("Reset: size=%d, want 0", c.Size())
	}
}

func TestCounter_NilSafe(t *testing.T) {
	var c *Counter
	c.Incr("VN")     // should not panic
	c.Add("VN", 10)  // should not panic
	if c.Get("VN") != 0 {
		t.Error("nil counter should return 0")
	}
	if c.Size() != 0 {
		t.Error("nil counter size = 0")
	}
	if c.DumpSorted() != "" {
		t.Error("nil counter dump empty")
	}
	c.Reset() // no-op
}

// ── CounterSet tests ─────────────────────────────────────────────────────────

func TestCounterSet_FlushWritesFiles(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)
	cs := NewCounterSet(w)

	cs.FbAppVersion.Incr("554.0.0.57.70|918990560")
	cs.FbAppVersion.Incr("556.1.0.63.64|942217461")

	if err := cs.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	assertFileContains(t, filepath.Join(dir, FileFbAppVersionSuccess), "554.0.0.57.70|918990560|1")
	assertFileContains(t, filepath.Join(dir, FileFbAppVersionSuccess), "556.1.0.63.64|942217461|1")
}

func TestCounterSet_AutoSave(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)
	cs := NewCounterSet(w)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cs.Start(ctx, time.Second) // min interval 1s (clamped)

	cs.FbAppVersion.Incr("556.1.0.63.64|942217461")
	time.Sleep(1100 * time.Millisecond) // đợi 1 tick

	assertFileContains(t, filepath.Join(dir, FileFbAppVersionSuccess), "556.1.0.63.64|942217461|1")

	cs.Stop() // flush lần cuối
}

func TestCounterSet_StopFlushFinal(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)
	cs := NewCounterSet(w)

	cs.Start(context.Background(), 10*time.Second) // interval dài để không có tick nào chạy
	cs.FbAppVersion.Incr("554.0.0.57.70|918990560")
	cs.Stop() // phải flush lần cuối

	assertFileContains(t, filepath.Join(dir, FileFbAppVersionSuccess), "554.0.0.57.70|918990560|1")
}

func TestCounterSet_StopIdempotent(t *testing.T) {
	cs := NewCounterSet(NewWriter(t.TempDir()))
	cs.Start(context.Background(), time.Second)
	cs.Stop()
	cs.Stop() // no panic
}

func TestCounterSet_StartIdempotent(t *testing.T) {
	cs := NewCounterSet(NewWriter(t.TempDir()))
	cs.Start(context.Background(), time.Second)
	cs.Start(context.Background(), time.Second) // no-op, không tạo goroutine dup
	cs.Stop()
}

func TestCounterSet_NilWriter(t *testing.T) {
	cs := NewCounterSet(nil)
	cs.FbAppVersion.Incr("556.1.0.63.64|942217461")
	if err := cs.Flush(); err != nil {
		t.Errorf("Flush with nil writer should be no-op, got err=%v", err)
	}
}

// ── Dispatch tests ───────────────────────────────────────────────────────────

func TestDispatchVerifyDetails_DieCheckpoint(t *testing.T) {
	// saveVerifyOutcome tự ghi DieAfterVerify.txt — dispatch chỉ trả supplementary files.
	got := DispatchVerifyDetails("die", "account has checkpoint", "100|pass|...")
	foundDieMain := false
	foundCheckpoint := false
	for _, d := range got {
		if d.File == FileDieAfterVerify {
			foundDieMain = true
		}
		if d.File == VerifyFailedFile("Checkpoint") {
			foundCheckpoint = true
		}
	}
	if foundDieMain {
		t.Error("dispatch should NOT return DieAfterVerify.txt — saveVerifyOutcome handles it")
	}
	if !foundCheckpoint {
		t.Error("missing VerifyFailed_Checkpoint.txt")
	}
}

func TestDispatchVerifyDetails_UnknownCantGetCode(t *testing.T) {
	// saveVerifyOutcome tự ghi Unknown.txt — dispatch chỉ trả supplementary files.
	got := DispatchVerifyDetails("unknown", "china mail can't get code", "100|pass")
	var files []string
	for _, d := range got {
		files = append(files, d.File)
	}
	joined := strings.Join(files, ",")
	if strings.Contains(joined, FileUnknownErrorCheckLiveDieApi) {
		t.Errorf("dispatch should NOT return Unknown.txt — saveVerifyOutcome handles it, got %v", files)
	}
	if !strings.Contains(joined, FileChinaMailCantGetCode) {
		t.Errorf("missing ChinaMail file: %v", files)
	}
}

func TestDispatchVerifyDetails_LiveEmpty(t *testing.T) {
	got := DispatchVerifyDetails("live", "success", "100|pass")
	if len(got) != 0 {
		t.Errorf("live should return no extra dispatch, got %d", len(got))
	}
}

func TestDetectVerifyFailCode(t *testing.T) {
	tests := map[string]string{
		"cantgetcode from server":  "CantGetCode",
		"createtempmail failed":    "CreateTempMailFailed",
		"AddMailFailed":            "AddMailFailed",
		"unknown":                  "",
		"ConfirmMailFailed retry":  "ConfirmMailFailed",
	}
	for input, want := range tests {
		got := detectVerifyFailCode(strings.ToLower(input))
		if got != want {
			t.Errorf("detectVerifyFailCode(%q) = %q, want %q", input, got, want)
		}
	}
}

// ── ErrorLog tests ───────────────────────────────────────────────────────────

func TestWriter_LogError(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)
	if err := w.LogError("worker slot=3", os.ErrNotExist); err != nil {
		t.Fatalf("LogError: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, FileErrorData))
	s := string(data)
	if !strings.Contains(s, "worker slot=3") {
		t.Errorf("errordata missing context: %q", s)
	}
	if !strings.Contains(s, "err:") {
		t.Errorf("errordata missing err tag: %q", s)
	}
}

func TestWriter_RecordPanic(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)

	// Simulate panic recover — recover() phải gọi trực tiếp trong defer function
	func() {
		defer func() {
			if r := recover(); r != nil {
				w.RecordPanic(r, "worker test")
			}
		}()
		panic("oops")
	}()

	data, _ := os.ReadFile(filepath.Join(dir, FileErrorData))
	s := string(data)
	if !strings.Contains(s, "PANIC worker test") {
		t.Errorf("errordata missing panic header: %q", s)
	}
	if !strings.Contains(s, "oops") {
		t.Errorf("errordata missing recover value: %q", s)
	}
	if !strings.Contains(s, "stack:") {
		t.Errorf("errordata missing stack tag: %q", s)
	}
}

func TestWriter_RecordPanic_NilRecovered(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)

	w.RecordPanic(nil, "not panicking") // no-op

	// File không được tạo khi không có panic
	if _, err := os.Stat(filepath.Join(dir, FileErrorData)); !os.IsNotExist(err) {
		t.Error("errordata.txt should not exist when RecordPanic(nil)")
	}
}

// ── helper ───────────────────────────────────────────────────────────────────

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Errorf("file %s does not contain %q, got %q", filepath.Base(path), want, data)
	}
}
