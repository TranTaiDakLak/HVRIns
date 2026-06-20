// result_test.go — S05-D1-T003: khóa hành vi FormatReg/FormatVerify/UpsertUID/
// ParseEmailMetaFromLine để đổi format/parser sau này không vỡ âm thầm.
//
// Test gốc (store_test.go) đã có TestFormatReg/TestFormatVerify_*/TestUpsertByUID_*;
// file này BỔ SUNG (tên riêng, không trùng): khóa thứ tự field tường minh theo VỊ TRÍ,
// round-trip EmailMeta (writer ↔ reader), và đặc biệt ParseEmailMetaFromLine (gap chưa có).
package result

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"
)

// fixedTime — timestamp cố định để format ổn định (tránh phụ thuộc time.Now()).
func fixedTime() time.Time { return time.Date(2026, 4, 18, 10, 30, 15, 0, time.UTC) }

const fixedStamp = "18-04-2026 10:30:15" // = fixedTime().Format("02-01-2006 15:04:05")

// ─── FormatReg: khóa thứ tự field theo vị trí ────────────────────────────────

func TestFormatReg_FieldOrderLock(t *testing.T) {
	tm := fixedTime()

	// 6 field khi KHÔNG có email: uid|pass|cookie|token|time|country
	got := FormatReg(RegData{
		UID: "100", Password: "pw", Cookie: "ck", Token: "tk", Country: "VN",
	}, &tm)
	want := "100|pw|ck|tk|" + fixedStamp + "|VN"
	if got != want {
		t.Errorf("no-email:\n got=%q\nwant=%q", got, want)
	}

	// 7 field khi CÓ email: email chèn giữa token và time
	got = FormatReg(RegData{
		UID: "100", Password: "pw", Cookie: "ck", Token: "tk",
		Email: "a@b.com", Country: "VN",
	}, &tm)
	want = "100|pw|ck|tk|a@b.com|" + fixedStamp + "|VN"
	if got != want {
		t.Errorf("with-email:\n got=%q\nwant=%q", got, want)
	}

	// IsNVR → append "|NVR" SAU country
	got = FormatReg(RegData{
		UID: "100", Password: "pw", Cookie: "ck", Token: "tk", Country: "VN", IsNVR: true,
	}, &tm)
	want = "100|pw|ck|tk|" + fixedStamp + "|VN|NVR"
	if got != want {
		t.Errorf("nvr:\n got=%q\nwant=%q", got, want)
	}
}

// ─── FormatVerify: khóa thứ tự field 9 cột + suffix ads ───────────────────────

func TestFormatVerify_FieldOrderLock(t *testing.T) {
	tm := fixedTime()

	got := FormatVerify(VerifyData{
		UID: "100", Password: "pw", TwoFA: "2fa", Cookie: "ck", Token: "tk",
		Email: "a@b.com", FullName: "John Doe", Country: "VN",
	}, &tm)
	want := "100|pw|2fa|ck|tk|a@b.com|John Doe|" + fixedStamp + "|VN"
	if got != want {
		t.Errorf("verify base:\n got=%q\nwant=%q", got, want)
	}

	// HasCalledOpenAds → append marker TKQC + currency
	got = FormatVerify(VerifyData{
		UID: "100", Password: "pw", TwoFA: "2fa", Cookie: "ck", Token: "tk",
		Email: "a@b.com", FullName: "John Doe", Country: "VN",
		HasCalledOpenAds: true, CurrencyCode: "USD",
	}, &tm)
	if !strings.HasPrefix(got, want+"|") {
		t.Errorf("ads: phải giữ nguyên 9 field base trước marker; got=%q", got)
	}
	if !strings.HasSuffix(got, "|USD") {
		t.Errorf("ads: phải kết bằng |USD; got=%q", got)
	}
	if !strings.Contains(got, "Live Ads") {
		t.Errorf("ads: thiếu marker TKQC; got=%q", got)
	}
}

// ─── ParseEmailMetaFromLine: GAP chưa có test gốc — khóa kỹ ───────────────────

func TestParseEmailMetaFromLine(t *testing.T) {
	tm := fixedTime()
	// Dựng line thật có MM: bằng chính FormatReg để không hardcode base64.
	meta := `{"provider":"zeus","user":"x|y","pass":"p@ss"}` // CHỨA "|" — chứng minh base64 bảo vệ
	lineWithMeta := FormatReg(RegData{
		UID: "100", Password: "pw", Cookie: "ck", Token: "tk",
		Country: "VN", EmailMeta: meta,
	}, &tm)

	tests := []struct {
		name string
		line string
		want string
	}{
		{"suffix MM trên full reg line", lineWithMeta, meta},
		{"không có cột MM", "100|pw|ck|tk|" + fixedStamp + "|VN", ""},
		{"line rỗng", "", ""},
		{"MM base64 hỏng → rỗng", "100|pw|MM:@@@khong-phai-base64@@@", ""},
		{"chỉ mỗi token MM hợp lệ", "MM:" + b64("hello"), "hello"},
		{"MM có khoảng trắng quanh token", "uid|  MM:" + b64("trim") + "  ", "trim"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := ParseEmailMetaFromLine(tc.line); got != tc.want {
				t.Errorf("ParseEmailMetaFromLine(%q) = %q; want %q", tc.line, got, tc.want)
			}
		})
	}
}

// b64 — helper encode để dựng input test mà không hardcode chuỗi base64.
func b64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// ─── Round-trip: FormatReg ghi EmailMeta → ParseEmailMetaFromLine đọc lại đúng ─

func TestFormatReg_EmailMetaRoundTrip(t *testing.T) {
	tm := fixedTime()
	for _, meta := range []string{
		`{"provider":"dvfb","u":"abc","p":"x|y|z"}`, // nhiều "|" trong JSON
		"simple-no-special",
		`{"emoji":"🎯","unicode":"tiếng việt"}`,
	} {
		line := FormatReg(RegData{
			UID: "999", Password: "pw", Cookie: "ck", Token: "tk",
			Country: "US", EmailMeta: meta,
		}, &tm)
		// Cột MM: KHÔNG được phá vị trí 7 field đầu (parser legacy đọc fields[0..6]).
		if got := ParseEmailMetaFromLine(line); got != meta {
			t.Errorf("round-trip fail:\n meta in=%q\nmeta out=%q\nline=%q", meta, got, line)
		}
		// EmailMeta rỗng → KHÔNG có cột MM:
		lineNoMeta := FormatReg(RegData{UID: "999", Password: "pw", Cookie: "ck", Token: "tk", Country: "US"}, &tm)
		if strings.Contains(lineNoMeta, "MM:") {
			t.Errorf("EmailMeta rỗng không được sinh cột MM:; line=%q", lineNoMeta)
		}
	}
}

// ─── UpsertUID: append UID mới, replace UID trùng (dedupe theo field[0]) ───────

func TestUpsertUID_BehaviorLock(t *testing.T) {
	w := NewWriter(t.TempDir())
	const file = "Die.txt"

	mustUpsert := func(line string) {
		t.Helper()
		if err := w.UpsertUID(file, line); err != nil {
			t.Fatalf("UpsertUID(%q): %v", line, err)
		}
	}

	mustUpsert("100|pw|v1") // UID mới → append
	mustUpsert("200|pw|v1") // UID khác → append
	mustUpsert("100|pw|v2") // UID 100 trùng → REPLACE (không thêm dòng)

	data, err := os.ReadFile(w.Path(file))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	lines := splitNonEmpty(string(data))
	if len(lines) != 2 {
		t.Fatalf("kỳ vọng 2 dòng (UID 100 replace, không nhân đôi), got %d: %v", len(lines), lines)
	}
	// UID 100 phải là v2 (đã replace), UID 200 còn nguyên.
	var got100, got200 string
	for _, ln := range lines {
		switch {
		case strings.HasPrefix(ln, "100|"):
			got100 = ln
		case strings.HasPrefix(ln, "200|"):
			got200 = ln
		}
	}
	if got100 != "100|pw|v2" {
		t.Errorf("UID 100 phải = v2 sau upsert, got %q", got100)
	}
	if got200 != "200|pw|v1" {
		t.Errorf("UID 200 phải còn nguyên, got %q", got200)
	}
}

func splitNonEmpty(s string) []string {
	var out []string
	for _, ln := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(ln) != "" {
			out = append(out, ln)
		}
	}
	return out
}
