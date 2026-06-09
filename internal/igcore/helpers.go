// helpers.go — USDID, time, regex parser cho response Bloks.
package igcore

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

func nowUnix() int64 { return time.Now().Unix() }

// genUSDID — X-Meta-USDID: "{UUID-upper}.{unix}.{base64url ECDSA-P256 sig}".
func genUSDID(p *igProfile) string {
	// Dùng device-id (đã uppercase UUID) làm phần đầu, giống capture.
	id := p.DeviceID
	ts := fmt.Sprintf("%d", nowUnix())
	payload := id + "." + ts
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return payload + ".err"
	}
	h := sha256.Sum256([]byte(payload))
	sig, err := ecdsa.SignASN1(rand.Reader, key, h[:])
	if err != nil {
		return payload + ".err"
	}
	return payload + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// ─── Response parsers ──────────────────────────────────────────────────────

// reRegContext — bắt reg_context trong response Bloks (đã strip backslash).
// Pattern trong response: "reg_context" "<value ...|regm>" — value rất dài, kết thúc bằng |regm.
var (
	reRegContext = regexp.MustCompile(`reg_context"[^"]*"([A-Za-z0-9_\-]{30,}\|?[a-z]*)"`)
	reErrorMsg   = regexp.MustCompile(`(?:error_message|"#")"\s*"([^"]{4,300})"`)
)

// parseRegContext bóc reg_context server cấp từ response (để mang sang step kế).
// Trả "" nếu không thấy.
func parseRegContext(resp string) string {
	clean := strings.ReplaceAll(resp, `\`, "")
	// Tìm cụm: "reg_context") (dkc "<value>") — value thường ngay sau reg_context trong keys/values.
	// Fallback: regex bắt blob dài kết thúc |regm.
	if m := regexp.MustCompile(`([A-Za-z0-9_\-]{200,}\|regm)`).FindStringSubmatch(clean); len(m) > 1 {
		return m[1]
	}
	if m := reRegContext.FindStringSubmatch(clean); len(m) > 1 {
		return m[1]
	}
	return ""
}

// hasMarker kiểm tra response chứa chuỗi marker (case-insensitive, sau strip backslash).
func hasMarker(resp, marker string) bool {
	clean := strings.ToLower(strings.ReplaceAll(resp, `\`, ""))
	return strings.Contains(clean, strings.ToLower(marker))
}

// extractError lấy 1 đoạn error_message dễ đọc (decode \uXXXX nếu có).
func extractError(resp string) string {
	clean := strings.ReplaceAll(resp, `\\`, `\`)
	if m := regexp.MustCompile(`error_message[\\"]*\s*[\\"]*([^\\"]{4,200})`).FindStringSubmatch(clean); len(m) > 1 {
		return decodeUnicode(strings.TrimSpace(m[1]))
	}
	return ""
}

// rotateSessionProxy đổi IP bằng cách thêm -session-<rand> vào username proxy.
// sessionCounter đảm bảo mỗi session token là duy nhất tuyệt đối,
// kể cả khi nhiều luồng gọi cùng nanosecond.
var sessionCounter uint64

// uniqueSessionToken sinh token duy nhất: counter + crypto random.
// Format alphanumeric an toàn cho mọi proxy provider.
func uniqueSessionToken() string {
	n := atomic.AddUint64(&sessionCounter, 1)
	b := make([]byte, 5)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%d%x", n, b)
}

// rotateSessionProxy inject session token mới vào username để lấy IP khác.
// "host:port:user:pass" → "host:port:user-session-<tok>:pass".
// Nếu user đã có "-session-" cũ thì thay token mới (không nối chồng).
func rotateSessionProxy(proxyStr string) string {
	parts := strings.SplitN(proxyStr, ":", 4)
	if len(parts) != 4 {
		return proxyStr
	}
	host, port, user, pass := parts[0], parts[1], parts[2], parts[3]

	// Bỏ token session cũ nếu có (tránh user-session-X-session-Y).
	if idx := strings.Index(user, "-session-"); idx >= 0 {
		user = user[:idx]
	}

	tok := uniqueSessionToken()
	return host + ":" + port + ":" + user + "-session-" + tok + ":" + pass
}

// ─── Debug dump ────────────────────────────────────────────────────────────

var debugDir = os.Getenv("IGDEBUG_DIR")

func dumpDir() string { return debugDir }

func writeDebug(dir, name, content string) {
	_ = os.MkdirAll(dir, 0755)
	_ = os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
}

// extractInvalidReason bóc lý do thật sau USER_REGISTER_INVALID_EMAIL trong action Bloks.
// Trả "" nếu không tìm thấy lý do cụ thể (→ coi như throttle trá hình).
func extractInvalidReason(resp string) string {
	clean := strings.ReplaceAll(resp, `\`, "")
	low := strings.ToLower(clean)
	switch {
	case strings.Contains(low, "throttl"):
		return "" // throttle → retry
	case strings.Contains(low, "already") || strings.Contains(low, "taken") || strings.Contains(low, "in use") || strings.Contains(low, "đã được") || strings.Contains(low, "existing"):
		return "EMAIL_ĐÃ_DÙNG"
	case strings.Contains(low, "invalid") && (strings.Contains(low, "domain") || strings.Contains(low, "format")):
		return "EMAIL_DOMAIN_INVALID"
	}
	// Bóc đoạn error_message gần USER_REGISTER_INVALID_EMAIL.
	if i := strings.Index(clean, "USER_REGISTER_INVALID_EMAIL"); i >= 0 {
		seg := clean[i:]
		if len(seg) > 400 {
			seg = seg[:400]
		}
		if m := regexp.MustCompile(`error_message[":\s]*"?([^"]{6,150})`).FindStringSubmatch(seg); len(m) > 1 {
			return decodeUnicode(m[1])
		}
	}
	return ""
}

// IGSession — session cookies sau khi tạo account thành công.
type IGSession struct {
	UID        string
	SessionID  string
	CSRFToken  string
	DSUserID   string
	Datr       string
	IgDID      string
	Mid        string
	Rur        string
	FullCookie string // full cookie string sẵn dùng
}

// parseIGSession bóc cookies từ create.account response.
// Response có nhiều lớp escape: `\\\\\\\\u00253A` → strip `\` → `u00253A` → decode → `%3A`.
func parseIGSession(resp string) IGSession {
	clean := strings.ReplaceAll(resp, `\`, "")
	clean = regexp.MustCompile(`u([0-9a-fA-F]{4})`).ReplaceAllStringFunc(clean, func(m string) string {
		var r rune
		fmt.Sscanf(m[1:], "%04x", &r)
		return string(r)
	})

	extract := func(pattern string) string {
		if m := regexp.MustCompile(pattern).FindStringSubmatch(clean); len(m) > 1 {
			return m[1]
		}
		return ""
	}

	var s IGSession
	s.SessionID = extract(`sessionid=([A-Za-z0-9%:._\-]{20,})`)
	s.CSRFToken = extract(`csrftoken=([A-Za-z0-9]{20,})`)
	s.DSUserID = extract(`ds_user_id=(\d{8,})`)
	s.Datr = extract(`datr=([A-Za-z0-9_\-]{10,})`)
	s.IgDID = extract(`ig_did=([A-Z0-9\-]{30,})`)
	s.Mid = extract(`mid=([A-Za-z0-9_\-]{10,})`)
	s.Rur = extract(`rur=([A-Za-z0-9,_:\-]{5,})`)

	if s.DSUserID != "" {
		s.UID = s.DSUserID
	}

	// Build full cookie string
	parts := []string{}
	if s.CSRFToken != "" {
		parts = append(parts, "csrftoken="+s.CSRFToken)
	}
	if s.Datr != "" {
		parts = append(parts, "datr="+s.Datr)
	}
	if s.IgDID != "" {
		parts = append(parts, "ig_did="+s.IgDID)
	}
	if s.Mid != "" {
		parts = append(parts, "mid="+s.Mid)
	}
	if s.Rur != "" {
		parts = append(parts, "rur="+s.Rur)
	}
	if s.DSUserID != "" {
		parts = append(parts, "ds_user_id="+s.DSUserID)
	}
	if s.SessionID != "" {
		parts = append(parts, "sessionid="+s.SessionID)
	}
	s.FullCookie = strings.Join(parts, ";")

	return s
}

// parseConfirmationCode bóc confirmation_code token từ confirmOTP response.
// Token dạng base64 8 chars (vd "nLbK4VOJ") — IG cấp sau xác nhận OTP thành công.
func parseConfirmationCode(resp string) string {
	clean := strings.ReplaceAll(resp, `\`, "")
	if m := regexp.MustCompile(`confirmation_code[":\s]*"([A-Za-z0-9+/]{6,12})"`).FindStringSubmatch(clean); len(m) > 1 {
		return m[1]
	}
	return ""
}

func decodeUnicode(s string) string {
	re := regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)
	return re.ReplaceAllStringFunc(s, func(m string) string {
		var r rune
		fmt.Sscanf(m[2:], "%04x", &r)
		return string(r)
	})
}
