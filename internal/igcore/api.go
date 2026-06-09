// api.go — exported public API for the igcore package.
// All other files use unexported names; this file bridges to callers.
package igcore

import (
	"context"
	"errors"
	"io"
	"net/url"
	"regexp"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
)

// ErrThrottled is the exported throttle sentinel.
var ErrThrottled = errThrottled

// InjectAgedDevice set cookie datr/mid/ig_did aged vào cookie jar của session
// TRƯỚC khi reg → IG thấy "thiết bị có lịch sử". Mid cũng được set vào profile
// để dùng làm X-MID header (override mid tươi từ qe/sync).
func (s *Session) InjectAgedDevice(p *Profile, dev *AgedDevice) {
	if dev == nil {
		return
	}
	if dev.Mid != "" {
		p.MachineID = dev.Mid // dùng mid aged làm X-MID
	}
	u, err := url.Parse("https://i.instagram.com")
	if err != nil {
		return
	}
	var cookies []*fhttp.Cookie
	if dev.Datr != "" {
		cookies = append(cookies, &fhttp.Cookie{Name: "datr", Value: dev.Datr, Domain: ".instagram.com", Path: "/"})
	}
	if dev.Mid != "" {
		cookies = append(cookies, &fhttp.Cookie{Name: "mid", Value: dev.Mid, Domain: ".instagram.com", Path: "/"})
	}
	if dev.IgDID != "" {
		cookies = append(cookies, &fhttp.Cookie{Name: "ig_did", Value: dev.IgDID, Domain: ".instagram.com", Path: "/"})
	}
	if len(cookies) > 0 {
		s.client.SetCookies(u, cookies)
	}
}

// IsThrottled reports whether err is a throttle error.
func IsThrottled(err error) bool {
	return errors.Is(err, errThrottled)
}

// RotateSession rotates the session token in a proxy string.
func RotateSession(proxyStr string) string {
	return rotateSessionProxy(proxyStr)
}

// Session wraps igSession for external callers.
type Session = igSession

// NewIGSession creates a new TLS session with the given proxy.
func NewIGSession(proxyStr string) (*Session, error) {
	return newIGSession(proxyStr)
}

// Profile wraps igProfile for external callers.
type Profile = igProfile

// NewProfile creates a fresh random iOS device fingerprint profile.
func NewProfile() *Profile {
	return newRandomProfile()
}

// NewProfileForCountry tạo profile iOS với locale khớp với country code của proxy.
// VD: country="VN" → locale="vi_VN", country="US" → locale="en_US".
func NewProfileForCountry(countryCode string) *Profile {
	locale := CountryToLocale(countryCode)
	return newRandomProfileWithLocale(locale)
}

// CheckProxyCountry gửi request qua proxy để phát hiện country code của IP.
// Returns ISO 2-letter code ("VN", "US", "TH"...) hoặc "" nếu không xác định được.
func (s *Session) CheckProxyCountry(ctx context.Context) string {
	_, country := s.CheckProxyIPCountry(ctx)
	return country
}

var ipFieldRe = regexp.MustCompile(`"query"\s*:\s*"([0-9.]+)"`)
var ccFieldRe = regexp.MustCompile(`"countryCode"\s*:\s*"([A-Z]{2})"`)

// CheckProxyIPCountry trả về cả IP thật và country code của proxy.
// VD: ("103.241.43.12", "VN"). Trả ("","") nếu lỗi.
func (s *Session) CheckProxyIPCountry(ctx context.Context) (ip, country string) {
	req, err := fhttp.NewRequestWithContext(ctx, "GET",
		"http://ip-api.com/json/?fields=query,countryCode", nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	str := string(body)

	if m := ipFieldRe.FindStringSubmatch(str); len(m) > 1 {
		ip = m[1]
	}
	if m := ccFieldRe.FindStringSubmatch(str); len(m) > 1 {
		country = m[1]
	}
	return ip, country
}

// QeSync exposes the qe/sync step.
func (s *Session) QeSync(ctx context.Context, p *Profile) (keyID, pubKey, xmid string, err error) {
	return s.qeSync(ctx, p)
}

// Engine is the exported registration engine.
type Engine struct {
	inner *engine
}

// NewEngine creates an Engine wrapping the inner engine.
func NewEngine(sess *Session, p *Profile, keyID, pubKey, proxyStr string, log func(string, ...any)) *Engine {
	return &Engine{inner: &engine{
		sess:     sess,
		p:        p,
		log:      log,
		keyID:    keyID,
		pubKey:   pubKey,
		proxyStr: proxyStr,
	}}
}

// Log returns the engine's log function for reuse.
func (e *Engine) Log() func(string, ...any) { return e.inner.log }

// Aymh runs the aymh step.
func (e *Engine) Aymh(ctx context.Context) error { return e.inner.aymh(ctx) }

// SubmitEmail submits the email address to Instagram.
func (e *Engine) SubmitEmail(ctx context.Context, addr string) error {
	return e.inner.submitEmail(ctx, addr)
}

// ConfirmOTP confirms the OTP code.
func (e *Engine) ConfirmOTP(ctx context.Context, addr, otp string) error {
	return e.inner.confirmOTP(ctx, addr, otp)
}

// SetPassword sets the account password.
func (e *Engine) SetPassword(ctx context.Context, addr, password string) error {
	return e.inner.setPassword(ctx, addr, password)
}

// SetBirthday sets a random birthday.
func (e *Engine) SetBirthday(ctx context.Context, addr string) error {
	return e.inner.setBirthday(ctx, addr)
}

// SetNameIG sets the full name.
func (e *Engine) SetNameIG(ctx context.Context, addr, name string) error {
	return e.inner.setNameIG(ctx, addr, name)
}

// SetUsername sets the username.
func (e *Engine) SetUsername(ctx context.Context, addr, username string) error {
	return e.inner.setUsername(ctx, addr, username)
}

// AcceptTOS accepts the Terms of Service.
func (e *Engine) AcceptTOS(ctx context.Context, addr string) error {
	return e.inner.acceptTOS(ctx, addr)
}

// CreateAccount runs the final account creation step.
func (e *Engine) CreateAccount(ctx context.Context, addr, username, name string) error {
	return e.inner.createAccount(ctx, addr, username, name)
}

// Session returns the IGSession cookies after successful account creation.
func (e *Engine) Session() IGSession {
	return e.inner.Session
}

// CheckLive calls IG API to determine account status after registration.
// Returns: "live" | "suspended" | "checkpoint" | "unknown"
//
// Detection logic based on IG response patterns:
//   - {"status":"ok", "user":{...}}                               → live
//   - challenge_required + url contains "accounts/suspended"      → suspended (hard ban)
//   - challenge_required + lock:true + url has other path         → suspended (ban variant)
//   - challenge_required + lock:false                             → checkpoint (verify needed)
//   - login_required / not_authorized                             → session invalid → unknown
func (e *Engine) CheckLive(ctx context.Context) string {
	if e.inner.Session.SessionID == "" {
		return "unknown"
	}

	ua := igUserAgent
	if e.inner.p != nil && e.inner.p.UserAgent != "" {
		ua = e.inner.p.UserAgent // dùng UA khớp device đã reg
	}

	// Build cookie string from captured session
	var cookieParts []string
	if e.inner.Session.FullCookie != "" {
		cookieParts = append(cookieParts, e.inner.Session.FullCookie)
	} else {
		if e.inner.Session.SessionID != "" {
			cookieParts = append(cookieParts, "sessionid="+e.inner.Session.SessionID)
		}
		if e.inner.Session.CSRFToken != "" {
			cookieParts = append(cookieParts, "csrftoken="+e.inner.Session.CSRFToken)
		}
		if e.inner.Session.DSUserID != "" {
			cookieParts = append(cookieParts, "ds_user_id="+e.inner.Session.DSUserID)
		}
	}
	cookieStr := strings.Join(cookieParts, "; ")

	req, err := fhttp.NewRequestWithContext(ctx, "GET",
		"https://i.instagram.com/api/v1/accounts/current_user/?edit=true", nil)
	if err != nil {
		return "unknown"
	}
	req.Header.Set("Cookie", cookieStr)
	req.Header.Set("X-CSRFToken", e.inner.Session.CSRFToken)
	req.Header.Set("X-IG-App-ID", igAppID)
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "*/*")

	resp, err := e.inner.sess.client.Do(req)
	if err != nil {
		return "unknown"
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	s := string(body)
	code := resp.StatusCode

	// classifyLiveResponse tách logic để test được.
	return classifyLiveResponse(code, s)
}

// classifyLiveResponse phân loại response của accounts/current_user theo
// taxonomy của instagrapi. Trả: live | suspended | checkpoint | die | unknown.
func classifyLiveResponse(statusCode int, body string) string {
	// ── 429 / throttle: KHÔNG phải verdict, để unknown để retry ───────────────
	if statusCode == 429 || strings.Contains(body, "Please wait a few minutes") {
		return "unknown"
	}

	// ── SUSPENDED: ban policy ──────────────────────────────────────────────────
	if strings.Contains(body, "accounts/suspended") {
		return "suspended"
	}

	// ── challenge_required: phân biệt suspended vs checkpoint ───────────────────
	if strings.Contains(body, "challenge_required") {
		// lock:true + suspended URL đã bắt ở trên. lock:true mà path khác → ban variant.
		if strings.Contains(body, `"lock":true`) {
			return "suspended"
		}
		// challenge khác (verify email/phone) → account còn sống, cần verify
		return "checkpoint"
	}

	// ── feedback_required: action block — account VẪN LIVE, chỉ bị hạn chế ──────
	if strings.Contains(body, "feedback_required") {
		return "live"
	}

	// ── login_required / 401 / 403: session chết ───────────────────────────────
	if statusCode == 401 || statusCode == 403 ||
		strings.Contains(body, "login_required") ||
		strings.Contains(body, "not_authorized") {
		return "die"
	}

	// ── sentry_block: account bị flag automation (coi như die) ──────────────────
	if strings.Contains(body, "sentry_block") {
		return "die"
	}

	// ── LIVE: trả về user data ──────────────────────────────────────────────────
	if strings.Contains(body, `"status":"ok"`) ||
		strings.Contains(body, `"logged_in_user"`) ||
		strings.Contains(body, `"user":{`) {
		return "live"
	}

	return "unknown"
}
