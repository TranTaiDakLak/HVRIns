package result

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAppendLine_LazyMkdirAndAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "SuccessVerify.txt")

	if err := AppendLine(path, "uid1|pass1|..."); err != nil {
		t.Fatalf("append 1: %v", err)
	}
	if err := AppendLine(path, "uid2|pass2|..."); err != nil {
		t.Fatalf("append 2: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "uid1|pass1|...\nuid2|pass2|...\n"
	if got := string(data); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAppendLine_EmptyPath(t *testing.T) {
	if err := AppendLine("", "foo"); err == nil {
		t.Error("expected error for empty path")
	}
}

func TestUpsertByUID_AddNewThenUpdate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "DieAfterVerify.txt")

	// Lần đầu: thêm mới
	if err := UpsertByUID(path, "100|pass|cookie|v1"); err != nil {
		t.Fatalf("upsert new: %v", err)
	}
	if err := UpsertByUID(path, "200|pass|cookie|v2"); err != nil {
		t.Fatalf("upsert new 2: %v", err)
	}

	// Upsert cùng UID 100 → update dòng, không thêm
	if err := UpsertByUID(path, "100|pass|cookie|UPDATED"); err != nil {
		t.Fatalf("upsert update: %v", err)
	}

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "100|pass|cookie|UPDATED" {
		t.Errorf("line[0] not updated: %q", lines[0])
	}
	if lines[1] != "200|pass|cookie|v2" {
		t.Errorf("line[1] changed: %q", lines[1])
	}
}

func TestUpsertByUID_NoUIDFallsBackToAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")

	// Content không có UID (trim empty) → rơi về AppendLine
	if err := UpsertByUID(path, ""); err != nil {
		t.Fatalf("upsert empty: %v", err)
	}
	if err := UpsertByUID(path, "line2"); err != nil {
		t.Fatalf("upsert line2: %v", err)
	}
}

func TestOverwrite_ReplacesContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CountrySuccess.txt")

	if err := AppendLine(path, "VN|100"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := Overwrite(path, "VN|150\nUS|80\n"); err != nil {
		t.Fatalf("overwrite: %v", err)
	}

	data, _ := os.ReadFile(path)
	want := "VN|150\nUS|80\n"
	if string(data) != want {
		t.Errorf("got %q, want %q", data, want)
	}
}

func TestFirstField(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"100|pass|cookie", "100"},
		{"  100  |pass", "100"},
		{"100", "100"},
		{"", ""},
		{"|", ""}, // empty UID
	}
	for _, tt := range tests {
		if got := firstField(tt.in); got != tt.want {
			t.Errorf("firstField(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestAppendLine_Concurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Live.txt")

	var wg sync.WaitGroup
	N := 100
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = AppendLine(path, "uid"+stringOfInt(n))
		}(i)
	}
	wg.Wait()

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != N {
		t.Errorf("expected %d lines, got %d (có thể bị overlap ghi do mất lock)", N, len(lines))
	}
}

func TestUpsertByUID_Concurrent_SameUID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "DieAfterVerify.txt")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = UpsertByUID(path, "999|die|attempt"+stringOfInt(n))
		}(i)
	}
	wg.Wait()

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line (UID=999 upsert), got %d: %v", len(lines), lines)
	}
}

// ── Writer tests ─────────────────────────────────────────────────────────────

func TestWriter_Path(t *testing.T) {
	w := NewWriter("/base/result_x")
	got := w.Path(FileSuccessVerify)
	want := filepath.Join("/base/result_x", "SuccessVerify.txt")
	if got != want {
		t.Errorf("Path(FileSuccessVerify) = %q, want %q", got, want)
	}
}

func TestWriter_AppendAndPath(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)
	if err := w.Append(FileSuccessVerify, "uid|pass"); err != nil {
		t.Fatalf("append: %v", err)
	}
	data, _ := os.ReadFile(w.Path(FileSuccessVerify))
	if string(data) != "uid|pass\n" {
		t.Errorf("got %q", data)
	}
}

func TestDynamicFilename(t *testing.T) {
	tests := map[string]string{
		"VerifyFailed_Checkpoint.txt":     VerifyFailedFile("Checkpoint"),
		"LoginFbFailed_Error.txt":         LoginFbFailedFile("Error"),
		"Cfem_LoginFbFailed_WrongCode.txt": CfemLoginFbFailedFile("WrongCode"),
		"LoginGmail_Captcha.txt":          LoginGmailFile("Captcha"),
		"SuccessVerifyUG.txt":             SuccessVerifyUGFile("ApiAndroid"),
	}
	for want, got := range tests {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

func TestSanitizeFilename(t *testing.T) {
	if got := sanitizeFilename("bad/name:foo"); got != "bad_name_foo" {
		t.Errorf("sanitize: got %q", got)
	}
}

// ── Format tests ─────────────────────────────────────────────────────────────

func TestFormatReg(t *testing.T) {
	fixedTime := time.Date(2026, 4, 18, 10, 30, 15, 0, time.UTC)
	d := RegData{
		UID: "100", Password: "abc", Cookie: "c_user=100;xs=foo", Token: "EAA...",
		Country: "VN", IsNVR: true,
	}
	got := FormatReg(d, &fixedTime)
	want := "100|abc|c_user=100;xs=foo|EAA...|18-04-2026 10:30:15|VN|NVR"
	if got != want {
		t.Errorf("FormatReg:\ngot  %q\nwant %q", got, want)
	}
}

func TestFormatVerify_Basic(t *testing.T) {
	fixedTime := time.Date(2026, 4, 18, 10, 30, 15, 0, time.UTC)
	d := VerifyData{
		UID: "100", Password: "abc", TwoFA: "SECRET2FA",
		Cookie: "c_user=100", Token: "EAA",
		Email: "a@b.com", FullName: "John Doe", Country: "VN",
	}
	got := FormatVerify(d, &fixedTime)
	want := "100|abc|SECRET2FA|c_user=100|EAA|a@b.com|John Doe|18-04-2026 10:30:15|VN"
	if got != want {
		t.Errorf("FormatVerify:\ngot  %q\nwant %q", got, want)
	}
}

func TestFormatVerify_WithAds(t *testing.T) {
	d := VerifyData{
		UID: "100", HasCalledOpenAds: true, CurrencyCode: "VND",
	}
	got := FormatVerify(d, nil)
	if !strings.Contains(got, "Tạo TKQC OK - Kích Tut Lưỡng Tính Ok - Live Ads|VND") {
		t.Errorf("missing ads suffix: %q", got)
	}
}

// ── Helper ───────────────────────────────────────────────────────────────────

// stringOfInt: tránh import strconv trong test — tiny helper.
func stringOfInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
