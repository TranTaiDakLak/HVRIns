package fakeinfo

import (
	"testing"
)

func TestPhoneCodes_LoadedFullDataset(t *testing.T) {
	if got := len(phoneList); got < 200 {
		t.Errorf("phoneList = %d entries, expected >= 200 (C# 221)", got)
	}
}

func TestLookupPhoneCode_MajorCountries(t *testing.T) {
	tests := []struct {
		cc   string
		want string // expected prefix
	}{
		{"VN", "+84"},
		{"US", "+1"},
		{"GB", "+44"},
		{"JP", "+81"},
		{"DE", "+49"},
		{"TH", "+66"},
	}
	for _, tt := range tests {
		p, ok := LookupPhoneCode(tt.cc)
		if !ok {
			t.Errorf("LookupPhoneCode(%q): không tìm thấy", tt.cc)
			continue
		}
		if p.PhoneCode != tt.want {
			t.Errorf("LookupPhoneCode(%q): got %q, want %q", tt.cc, p.PhoneCode, tt.want)
		}
	}
}

func TestLookupPhoneCode_CaseInsensitive(t *testing.T) {
	p1, _ := LookupPhoneCode("vn")
	p2, _ := LookupPhoneCode("VN")
	if p1.PhoneCode != p2.PhoneCode || p1.PhoneCode == "" {
		t.Errorf("case-insensitive fail: vn=%q VN=%q", p1.PhoneCode, p2.PhoneCode)
	}
}

func TestLookupPhoneCode_NotFound(t *testing.T) {
	if _, ok := LookupPhoneCode("ZZ"); ok {
		t.Error("expected not found for ZZ")
	}
}

func TestPhoneCodeFor(t *testing.T) {
	if got := PhoneCodeFor("VN"); got != "+84" {
		t.Errorf("PhoneCodeFor(VN) = %q, want +84", got)
	}
	if got := PhoneCodeFor("XXX"); got != "" {
		t.Errorf("PhoneCodeFor(XXX) should be empty, got %q", got)
	}
}

func TestFindCountryByPhonePrefix(t *testing.T) {
	tests := []struct {
		phone string
		want  string // expected country code
	}{
		{"+84912345678", "VN"},
		{"84912345678", "VN"}, // no + prefix, should still work
		{"+15551234567", "US"},
		{"+442012345678", "GB"},
	}
	for _, tt := range tests {
		p, ok := FindCountryByPhonePrefix(tt.phone)
		if !ok {
			t.Errorf("FindCountryByPhonePrefix(%q): not found", tt.phone)
			continue
		}
		if p.CountryCode != tt.want {
			t.Errorf("FindCountryByPhonePrefix(%q): got %q, want %q", tt.phone, p.CountryCode, tt.want)
		}
	}
}

func TestPhoneCountries_ReturnsCopy(t *testing.T) {
	snapshot := PhoneCountries()
	if len(snapshot) == 0 {
		t.Fatal("phoneCountries empty")
	}
	orig := snapshot[0].Name
	snapshot[0].Name = "MUTATED"
	if PhoneCountries()[0].Name == "MUTATED" {
		t.Error("caller mutation leaked into phoneList")
	}
	_ = orig
}
