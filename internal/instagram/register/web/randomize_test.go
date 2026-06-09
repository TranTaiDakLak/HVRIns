package web

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

// TestRandomVNPhone_E164Format — verify VN phone gen ra E.164 với prefix "+84".
//
// Đây là regression guard cho fix: trước đây trả "0xxxxxxxxx" (local), giờ
// phải trả "+84xxxxxxxxx" để nhất quán với các country khác (US, Chile, PH...)
// và FB reg API expect E.164 → tăng tỉ lệ register success.
func TestRandomVNPhone_E164Format(t *testing.T) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 100; i++ {
		got := randomVNPhone(r)
		if !strings.HasPrefix(got, "+84") {
			t.Errorf("phone=%q must start with +84", got)
		}
		// E.164 VN: "+84" + 9 digits = 12 chars total.
		if len(got) != 12 {
			t.Errorf("phone=%q len=%d, want 12 (+84 + 9 digits)", got, len(got))
		}
		// Sau "+84" phải toàn digit.
		rest := got[3:]
		for _, c := range rest {
			if c < '0' || c > '9' {
				t.Errorf("phone=%q has non-digit %q after +84", got, c)
				break
			}
		}
	}
}

// TestGeneratePhoneByCountry_VN — VN qua GeneratePhoneByCountry phải dùng E.164.
func TestGeneratePhoneByCountry_VN(t *testing.T) {
	for i := 0; i < 20; i++ {
		got := GeneratePhoneByCountry("vn")
		if !strings.HasPrefix(got, "+84") {
			t.Errorf("GeneratePhoneByCountry(vn)=%q must start with +84", got)
		}
	}
}

// TestRandomVNPhone_PrefixMatchesCarrier — đầu số sau "+84" phải khớp với
// 1 trong các carrier VN hợp lệ (strip "0" đầu của vnPrefixes).
func TestRandomVNPhone_PrefixMatchesCarrier(t *testing.T) {
	r := rand.New(rand.NewSource(42))
	validPrefixesNoZero := make(map[string]bool)
	for _, p := range vnPrefixes {
		// vnPrefixes lưu "0xx", strip "0" → "xx".
		if len(p) >= 2 {
			validPrefixesNoZero[p[1:]] = true
		}
	}

	for i := 0; i < 50; i++ {
		got := randomVNPhone(r)
		if len(got) < 5 {
			t.Fatalf("phone=%q too short", got)
		}
		// "+84" (3) + 2 digits prefix = first 5 chars; check chars 3-4.
		twoDigitPrefix := got[3:5]
		if !validPrefixesNoZero[twoDigitPrefix] {
			t.Errorf("phone=%q prefix %q not in valid VN carrier list", got, twoDigitPrefix)
		}
	}
}
