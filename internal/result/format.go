// format.go — format chuỗi data theo pipe-delimited của C# FMain.GetSaveData().
//
// C# 2 format chính:
//
//	isFromReg=true  → uid|pass|cookie|token|time|country
//	isFromReg=false → uid|pass|2fa|cookie|token|email|fullname|time|country
//
// Thêm format cho TKQC:
//   - |Tạo TKQC OK - Kích Tut Lưỡng Tính Ok - Live Ads|{currencyCode}
package result

import (
	"encoding/base64"
	"strings"
	"time"
)

// RegData chứa field cần thiết cho Reg output.
// UID, Password, Cookie, Token phải có; Country có thể rỗng.
type RegData struct {
	UID string
	// Username: nếu có → dùng làm field ĐẦU thay cho UID (account IG identify bằng @handle;
	// UID số vẫn còn trong Cookie qua ds_user_id nên không mất). Rỗng → fallback UID.
	Username string
	Password string
	Cookie   string
	Token    string
	// Email: nếu có, được chèn giữa Token và Time → "uid|pass|cookie|token|email|time|country".
	// Nếu rỗng, format giữ nguyên 6 field: "uid|pass|cookie|token|time|country".
	Email   string
	Country string
	// IsNVR: true → append "|NVR" ở cuối (C# SuccessReg ghi "...country|NVR"
	// để phân biệt account "Not Verified Yet").
	IsNVR bool
	// EmailMeta — TempMail provider creds (base64-encoded JSON) để verify Restore.
	// Empty cho mode Phone/Mail (giả). Append làm cột cuối với prefix "MM:" để
	// loader cũ skip (split "|" → cột này nằm sau NVR, parser legacy access
	// fields[0..7] vẫn work).
	EmailMeta string
}

// FormatReg build chuỗi "uid|pass|cookie|token[|email]|time|country[|NVR][|MM:meta]".
// Email được thêm chỉ khi không rỗng.
// EmailMeta được append làm cột cuối với prefix "MM:" + base64 (chỉ khi có).
// time: nếu nil dùng time.Now(), format "dd-MM-yyyy HH:mm:ss" (giống C#).
func FormatReg(d RegData, now *time.Time) string {
	t := time.Now()
	if now != nil {
		t = *now
	}
	// Field đầu: ưu tiên Username (@handle IG), fallback UID nếu không có.
	lead := d.UID
	if d.Username != "" {
		lead = d.Username
	}
	parts := []string{
		lead,
		d.Password,
		d.Cookie,
		d.Token,
	}
	if d.Email != "" {
		parts = append(parts, d.Email)
	}
	parts = append(parts, t.Format("02-01-2006 15:04:05"), d.Country)
	out := joinNonNil(parts)
	if d.IsNVR {
		out += "|NVR"
	}
	if d.EmailMeta != "" {
		// Base64 encode để tránh ký tự "|" trong JSON meta phá format.
		out += "|MM:" + base64.StdEncoding.EncodeToString([]byte(d.EmailMeta))
	}
	return out
}

// ParseEmailMetaFromLine — extract EmailMeta từ saved reg/verify line.
// Trả "" nếu không có cột MM: hoặc decode fail.
//
// Pattern: split "|" → tìm cột bắt đầu "MM:" → strip prefix → base64 decode.
// Backwards-compat: line không có cột "MM:" → trả "" → verify fall back về
// CreateEmail mới (existing behavior).
func ParseEmailMetaFromLine(line string) string {
	parts := strings.Split(line, "|")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if !strings.HasPrefix(p, "MM:") {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(p[3:])
		if err != nil {
			return ""
		}
		return string(raw)
	}
	return ""
}

// VerifyData chứa field cho Verify output (có thêm 2FA + Email + FullName so với Reg).
type VerifyData struct {
	UID      string
	Password string
	TwoFA    string
	Cookie   string
	Token    string
	Email    string
	FullName string
	Country  string
	// HasCalledOpenAds: true → append "|Tạo TKQC OK - Kích Tut Lưỡng Tính Ok - Live Ads|{CurrencyCode}"
	// Port trực tiếp C# GetSaveData() khi hasCalled_openads=true.
	HasCalledOpenAds bool
	CurrencyCode     string
}

// FormatVerify build chuỗi:
//
//	"uid|pass|2fa|cookie|token|email|fullname|time|country"
//
// + "|Tạo TKQC OK - Kích Tut Lưỡng Tính Ok - Live Ads|currency" nếu HasCalledOpenAds.
func FormatVerify(d VerifyData, now *time.Time) string {
	t := time.Now()
	if now != nil {
		t = *now
	}
	parts := []string{
		d.UID,
		d.Password,
		d.TwoFA,
		d.Cookie,
		d.Token,
		d.Email,
		d.FullName,
		t.Format("02-01-2006 15:04:05"),
		d.Country,
	}
	out := joinNonNil(parts)
	if d.HasCalledOpenAds {
		out += "|Tạo TKQC OK - Kích Tut Lưỡng Tính Ok - Live Ads|" + d.CurrencyCode
	}
	return out
}

// joinNonNil nối các phần bằng "|" — giữ empty string (không skip) để field position
// nhất quán cho upsert/parser đếm field đúng.
func joinNonNil(parts []string) string {
	// Không dùng strings.Join trực tiếp để sau này có thể thêm escape logic
	// nếu field chứa ký tự "|" (hiếm, nhưng cookie có thể).
	return strings.Join(parts, "|")
}
