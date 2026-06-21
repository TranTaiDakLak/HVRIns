// helpers_test.go — S07-D1-T001: white-box test cho các helper THUẦN trong internal/app.
// Bỏ qua hàm dùng a.ctx / Wails runtime / network — chỉ test hàm thuần (package-level).
// Hàm bỏ: resetRegStats, recordRegOutcome, resetVerifyStats, recordVerifyOutcome,
//         popAccountFromFolder, popFromFile, startFolderWatcher, và tất cả (a *App) method
//         cần Wails ctx.
package app

import (
	"testing"
)

// ─── isGUID ──────────────────────────────────────────────────────────────────

func TestIsGUID(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		// Valid GUID
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF", true},
		{"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", true},
		{"00000000-0000-0000-0000-000000000000", true},
		// Wrong length
		{"", false},
		{"550e8400-e29b-41d4-a716-44665544000", false},  // 35 ký tự
		{"550e8400-e29b-41d4-a716-4466554400000", false}, // 37 ký tự
		// Dash ở vị trí sai
		{"550e8400-e29b41d4-a716-446655440000", false},
		// Ký tự ngoài hex
		{"550e8400-e29b-41d4-a716-44665544000g", false},
		{"550e8400-e29b-41d4-a716-44665544000 ", false},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := isGUID(tt.s); got != tt.want {
				t.Errorf("isGUID(%q) = %v; want %v", tt.s, got, tt.want)
			}
		})
	}
}

// ─── isAlphaNumeric ──────────────────────────────────────────────────────────

func TestIsAlphaNumeric(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"abc", true},
		{"ABC", true},
		{"123", true},
		{"abc123", true},
		{"ABC123xyz", true},
		{"", true}, // không có ký tự sai → true (range không chạy)
		{"abc@", false},
		{"abc-def", false},
		{"abc def", false},
		{"abc.def", false},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := isAlphaNumeric(tt.s); got != tt.want {
				t.Errorf("isAlphaNumeric(%q) = %v; want %v", tt.s, got, tt.want)
			}
		})
	}
}

// ─── hasLetterAndDigit ───────────────────────────────────────────────────────

func TestHasLetterAndDigit(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"abc123", true},
		{"A1bcde2", true},
		{"a1", true},
		// Chỉ chữ
		{"abcdef", false},
		{"ABCDEF", false},
		// Chỉ số
		{"123456", false},
		// Rỗng
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := hasLetterAndDigit(tt.s); got != tt.want {
				t.Errorf("hasLetterAndDigit(%q) = %v; want %v", tt.s, got, tt.want)
			}
		})
	}
}

// ─── isAllDigits ─────────────────────────────────────────────────────────────

func TestIsAllDigits(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"0", true},
		{"123456789", true},
		{"0000000", true},
		// Rỗng → false (len=0)
		{"", false},
		// Có chữ hoặc ký tự đặc biệt
		{"12a456", false},
		{"123 456", false},
		{"+84901234567", false},
		{"12.34", false},
	}
	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			if got := isAllDigits(tt.s); got != tt.want {
				t.Errorf("isAllDigits(%q) = %v; want %v", tt.s, got, tt.want)
			}
		})
	}
}

// ─── extractCUserFromCookie ──────────────────────────────────────────────────

func TestExtractCUserFromCookie(t *testing.T) {
	tests := []struct {
		cookie string
		want   string
	}{
		// c_user ở giữa cookie
		{"sb=xx; c_user=100123456789; xs=yy; fr=zz", "100123456789"},
		// c_user ở đầu
		{"c_user=987654321; xs=abc", "987654321"},
		// c_user ở cuối (không có dấu ; sau)
		{"sb=xx; c_user=111222333", "111222333"},
		// Không có c_user
		{"sb=xx; xs=yy; fr=zz", ""},
		// Cookie rỗng
		{"", ""},
		// Cookie chứa ds_user_id nhưng không có c_user
		{"ds_user_id=123; sessionid=abc", ""},
	}
	for _, tt := range tests {
		t.Run(tt.cookie[:min(len(tt.cookie), 40)], func(t *testing.T) {
			if got := extractCUserFromCookie(tt.cookie); got != tt.want {
				t.Errorf("extractCUserFromCookie(%q)\n got=%q\nwant=%q", tt.cookie, got, tt.want)
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── extractFBAV ─────────────────────────────────────────────────────────────

func TestExtractFBAV(t *testing.T) {
	tests := []struct {
		ua   string
		want string
	}{
		// User-agent thực tế: FBAV/version; ở giữa chuỗi
		{"[FBAN/FB4A;FBAV/448.0.0.35.109;FBBV/534222895;FBDM", "448.0.0.35.109"},
		// Version kết thúc bằng ] (bracket format)
		{"[FBAN/FB4A;FBAV/300.0.0.0.0]", "300.0.0.0.0"},
		// FBAV ở đầu chuỗi, kết thúc bởi ;
		{"FBAV/123.4.5;rest", "123.4.5"},
		// Không có FBAV
		{"Mozilla/5.0 (Linux; Android 10) AppleWebKit", ""},
		// Rỗng
		{"", ""},
		// FBAV tại cuối chuỗi, không có ; hay ]
		{"FBAV/500.0.0.0.1", "500.0.0.0.1"},
	}
	for _, tt := range tests {
		ua := tt.ua
		if len(ua) > 40 {
			ua = ua[:40]
		}
		t.Run(ua, func(t *testing.T) {
			if got := extractFBAV(tt.ua); got != tt.want {
				t.Errorf("extractFBAV(%q) = %q; want %q", tt.ua, got, tt.want)
			}
		})
	}
}

// ─── verifyPlatformDisplayName ───────────────────────────────────────────────

func TestVerifyPlatformDisplayName(t *testing.T) {
	tests := []struct {
		internal string
		want     string
	}{
		{"web", "api mfb"},
		{"webandroid", "api web andr"},
		{"s23", "api android"},
		{"android", "api token"},
		// Unknown platform → trả về chính nó
		{"custom_platform", "custom_platform"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.internal, func(t *testing.T) {
			if got := verifyPlatformDisplayName(tt.internal); got != tt.want {
				t.Errorf("verifyPlatformDisplayName(%q) = %q; want %q", tt.internal, got, tt.want)
			}
		})
	}
}

// ─── autoDetectAccount ───────────────────────────────────────────────────────

func TestAutoDetectAccount_BasicFields(t *testing.T) {
	// uid|pass|cookie(c_user=)|token(EAA)
	cookie := "sb=x; c_user=100000001; xs=y"
	token := "EAAabcdefghij123456789"
	line := "100000001|mypassword|" + cookie + "|" + token

	acc := autoDetectAccount(line)
	if acc.UID != "100000001" {
		t.Errorf("UID = %q; want 100000001", acc.UID)
	}
	if acc.Password != "mypassword" {
		t.Errorf("Password = %q; want mypassword", acc.Password)
	}
	if acc.Cookie != cookie {
		t.Errorf("Cookie = %q; want %q", acc.Cookie, cookie)
	}
	if acc.Token != token {
		t.Errorf("Token = %q; want %q", acc.Token, token)
	}
}

func TestAutoDetectAccount_With2FA(t *testing.T) {
	// uid|pass|2FA(32 hex alphanumeric mix)|cookie|token
	twofa := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4" // 32 hex alphanumeric
	cookie := "c_user=200000002; xs=z"
	token := "EAAtoken200"
	line := "200000002|pw2|" + twofa + "|" + cookie + "|" + token

	acc := autoDetectAccount(line)
	if acc.Twofa != twofa {
		t.Errorf("Twofa = %q; want %q", acc.Twofa, twofa)
	}
	if acc.UID != "200000002" {
		t.Errorf("UID = %q; want 200000002", acc.UID)
	}
}

func TestAutoDetectAccount_WithEmail(t *testing.T) {
	cookie := "c_user=300000003; xs=z"
	token := "EAAtoken300"
	line := "300000003|pw3|" + cookie + "|" + token + "|user@example.com"

	acc := autoDetectAccount(line)
	if acc.Email != "user@example.com" {
		t.Errorf("Email = %q; want user@example.com", acc.Email)
	}
}

func TestAutoDetectAccount_CookieOnlyExtractsUID(t *testing.T) {
	// Không có UID tường minh ở field 0, chỉ có cookie chứa c_user=
	cookie := "sb=x; c_user=444000444; xs=y"
	line := "|pw4|" + cookie + "|EAAtoken444"
	// field 0 là "" → UID từ cookie
	acc := autoDetectAccount(line)
	if acc.UID == "" {
		t.Errorf("UID phải được extract từ cookie c_user=; got empty")
	}
}

func TestAutoDetectAccount_PhoneField(t *testing.T) {
	cookie := "c_user=500000005; xs=y"
	token := "EAAtoken500"
	// Số điện thoại 10 chữ số → nhận diện là Phone
	line := "500000005|pw5|" + cookie + "|" + token + "|0901234567"

	acc := autoDetectAccount(line)
	if acc.Phone != "0901234567" {
		t.Errorf("Phone = %q; want 0901234567", acc.Phone)
	}
}

func TestAutoDetectAccount_SRNAndSCUID(t *testing.T) {
	cookie := "c_user=600000006; xs=y"
	token := "EAAtoken600"
	line := "600000006|pw6|" + cookie + "|" + token + "|SRN:mysrnonce|SCUID:mysessioncuid"

	acc := autoDetectAccount(line)
	if acc.Srnonce != "mysrnonce" {
		t.Errorf("Srnonce = %q; want mysrnonce", acc.Srnonce)
	}
	if acc.SessionlessCryptedUID != "mysessioncuid" {
		t.Errorf("SessionlessCryptedUID = %q; want mysessioncuid", acc.SessionlessCryptedUID)
	}
}

func TestAutoDetectAccount_EmptyLine(t *testing.T) {
	acc := autoDetectAccount("")
	if acc.UID != "" || acc.Password != "" || acc.Cookie != "" {
		t.Errorf("empty line should yield zero Account; got uid=%q pw=%q cookie=%q", acc.UID, acc.Password, acc.Cookie)
	}
}

func TestAutoDetectAccount_GUIDGoesToNote(t *testing.T) {
	guid := "550e8400-e29b-41d4-a716-446655440000"
	cookie := "c_user=700000007; xs=y"
	token := "EAAtoken700"
	line := "700000007|pw7|" + guid + "|" + cookie + "|" + token

	acc := autoDetectAccount(line)
	if acc.Note != guid {
		t.Errorf("Note = %q; want %q (GUID phải lưu vào Note)", acc.Note, guid)
	}
}
