package fakeinfo

import (
	"strings"
	"testing"
)

func TestUAPools_AllLoaded(t *testing.T) {
	tests := []struct {
		kind UAPoolKind
		min  int
	}{
		{UAKindAndroid, 1000}, // C# 1625
		{UAKindIOS, 1500},     // C# 2216
		{UAKindRequest, 1000}, // C# 1625
	}
	for _, tt := range tests {
		if got := UAPoolSize(tt.kind); got < tt.min {
			t.Errorf("UA pool %s size = %d, expected >= %d", tt.kind, got, tt.min)
		}
	}
}

func TestRandomUAFromPool_AndroidFormat(t *testing.T) {
	ua := RandomUAFromPool(UAKindAndroid)
	if ua == "" {
		t.Fatal("RandomUAFromPool(android) empty")
	}
	// C# FB4A UA bắt đầu bằng "[FBAN/FB4A"
	if !strings.HasPrefix(ua, "[FBAN/FB4A") {
		t.Errorf("Android UA không đúng format FB4A: %q", truncForTest(ua, 80))
	}
}

func TestRandomUAFromPool_IOSFormat(t *testing.T) {
	ua := RandomUAFromPool(UAKindIOS)
	if ua == "" {
		t.Fatal("RandomUAFromPool(ios) empty")
	}
	// iOS UA bắt đầu bằng "Mozilla/5.0 (iPhone"
	if !strings.HasPrefix(ua, "Mozilla/5.0 (iPhone") {
		t.Errorf("iOS UA không đúng format: %q", truncForTest(ua, 80))
	}
	if !strings.Contains(ua, "FBAN/FBIOS") {
		t.Errorf("iOS UA thiếu FBAN/FBIOS: %q", truncForTest(ua, 80))
	}
}

func TestRandomUAFromPool_Randomness(t *testing.T) {
	seen := make(map[string]struct{})
	for i := 0; i < 50; i++ {
		seen[RandomUAFromPool(UAKindAndroid)] = struct{}{}
	}
	// Với pool >1000 UA, 50 lần random phải cho ít nhất 10 UA khác nhau
	if len(seen) < 10 {
		t.Errorf("random UA quá ít biến thể: %d unique in 50 calls", len(seen))
	}
}

func TestUAOverrideActive_DefaultFalse(t *testing.T) {
	// Pool embed default không có override nếu user chưa tạo Config/UserAgent/
	// (test chạy trong cwd của package — Config/UserAgent/ không tồn tại)
	for _, kind := range []UAPoolKind{UAKindAndroid, UAKindIOS, UAKindRequest} {
		if UAPoolOverrideActive(kind) {
			t.Errorf("kind %s: override active but should be false by default in test env", kind)
		}
	}
}

func TestUAOverridePath(t *testing.T) {
	cases := map[UAPoolKind]string{
		UAKindAndroid: "Config/UserAgent/Android_UG.txt",
		UAKindIOS:     "Config/UserAgent/iOS_UG.txt",
		UAKindRequest: "Config/UserAgent/Request_UG.txt",
	}
	for kind, want := range cases {
		got := UAOverridePath(kind)
		// Filesystem separator may be "\" on Windows — so compare via path join semantics
		if !strings.HasSuffix(got, strings.ReplaceAll(want, "/", string([]byte{got[len("Config")]}))) &&
			got != want {
			// Soft compare: chấp nhận cả "/" và "\\"
			gotNorm := strings.ReplaceAll(got, "\\", "/")
			if gotNorm != want {
				t.Errorf("UAOverridePath(%s) = %q, want %q", kind, got, want)
			}
		}
	}
}

func truncForTest(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
