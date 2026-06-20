package app

import (
	"testing"
	"time"
)

// TestIsVerifiableAccountFile bảo đảm popAccountFromFolder chỉ đọc SuccessReg*.txt
// — tránh bug cũ: verify bị feed Phone/Email/UA/Blocked khiến Die.txt lẫn data rác.
func TestIsVerifiableAccountFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		// WHITELIST — file chứa account cookie đầy đủ
		{"SuccessReg.txt", true},
		{"SuccessReg_Android.txt", true},
		{"SuccessReg_S23.txt", true},

		// BLACKLIST — file không phải account verify-able
		{"SuccessNVR_Phone.txt", false},
		{"SuccessNVR_Email.txt", false},
		{"SuccessRegNVR_UG.txt", false},     // new: single file (không suffix)
		{"SuccessRegNVR_UG_s23.txt", false}, // legacy: vẫn excluded vì HasPrefix
		{"SuccessRegNVR_UG_ApiAndroid.txt", false},
		{"Blocked.txt", false},
		{"Checkpoint.txt", false},
		{"UnknownBlockType.txt", false},
		{"Live.txt", false},
		{"Die.txt", false},
		{"Unknown.txt", false},
		{"SuccessVerify.txt", false},
		{"SuccessVerify_No2FA.txt", false},
		{"DieAfterVerify.txt", false},
		{"CountrySuccess.txt", false},
		{"FbAppVersisonSuccess.txt", false},
		{"FbLocalesSuccess.txt", false},
		{"errordata.txt", false},
		{"RemainData.txt", false},

		// Edge cases
		{"", false},
		{"random.log", false},
		{"SuccessReg", false}, // không có .txt
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isVerifiableAccountFile(tt.name); got != tt.want {
				t.Errorf("isVerifiableAccountFile(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestEffectiveRegMode_RotatesContinuously(t *testing.T) {
	start := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)
	cfg := InteractionConfig{
		RegMode:                   "Phone",
		RegModeRotate:             true,
		RegModeRotatePhoneMinutes: 2,
		RegModeRotateMailMinutes:  3,
	}

	tests := []struct {
		at   time.Duration
		want string
	}{
		{0, "Phone"},
		{2 * time.Minute, "Mail"},
		{5 * time.Minute, "Phone"},
		{7 * time.Minute, "Mail"},
		{10 * time.Minute, "Phone"},
	}
	for _, tt := range tests {
		if got := effectiveRegMode(cfg, start, start.Add(tt.at)); got != tt.want {
			t.Fatalf("effectiveRegMode at %s = %s, want %s", tt.at, got, tt.want)
		}
	}
}

func TestEffectiveRegMode_StartsFromMail(t *testing.T) {
	start := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)
	cfg := InteractionConfig{
		RegMode:                   "Mail",
		RegModeRotate:             true,
		RegModeRotatePhoneMinutes: 2,
		RegModeRotateMailMinutes:  3,
	}

	if got := effectiveRegMode(cfg, start, start.Add(2*time.Minute)); got != "Mail" {
		t.Fatalf("before mail duration = %s, want Mail", got)
	}
	if got := effectiveRegMode(cfg, start, start.Add(3*time.Minute)); got != "Phone" {
		t.Fatalf("after mail duration = %s, want Phone", got)
	}
	if got := effectiveRegMode(cfg, start, start.Add(5*time.Minute)); got != "Mail" {
		t.Fatalf("after full cycle = %s, want Mail", got)
	}
}
