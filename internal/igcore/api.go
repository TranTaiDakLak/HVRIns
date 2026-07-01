// api.go — exported public API for the igcore package.
// All other files use unexported names; this file bridges to callers.
package igcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
)

// ErrThrottled is the exported throttle sentinel.
var ErrThrottled = errThrottled

// chromeMajorVersionFromUA extract "120"/"124"/"133" từ chuỗi UA "...Chrome/120.0.0.0...".
// Dùng để đồng bộ sec-ch-ua với UA/TLS profile đã chọn ngẫu nhiên — fallback "133"
// nếu không parse được (không nên xảy ra vì UA luôn do chromeCheckProfiles sinh ra).
func chromeMajorVersionFromUA(ua string) string {
	const marker = "Chrome/"
	i := strings.Index(ua, marker)
	if i == -1 {
		return "133"
	}
	rest := ua[i+len(marker):]
	end := strings.IndexByte(rest, '.')
	if end == -1 {
		return "133"
	}
	return rest[:end]
}

// EncryptPassword wraps encryptPasswordInstagram for external callers.
func EncryptPassword(password, pubKeyB64, keyIDStr string) (string, error) {
	return encryptPasswordInstagram(password, pubKeyB64, keyIDStr)
}

// ParseRegContext extracts the reg_context blob from a Bloks response.
func ParseRegContext(resp string) string { return parseRegContext(resp) }

// ParseIGSession extracts cookies from a create.account response.
func ParseIGSession(resp string) IGSession { return parseIGSession(resp) }

// ParseConfirmationCode extracts the 8-char confirmation token from a Bloks response.
func ParseConfirmationCode(resp string) string { return parseConfirmationCode(resp) }

// SharedDevicePool — aged device pool (mid/datr/ig_did) inject trước reg IG iOS.
// Set từ app_register.go trước khi batch bắt đầu, nil sau khi batch xong.
var SharedDevicePool *DevicePool

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

// ClassifyLiveResponse phân loại response của accounts/current_user theo
// taxonomy của instagrapi. Trả: live | suspended | checkpoint | die | unknown.
// Exported để các package khác (igandroid, ...) tái dùng cùng logic.
func ClassifyLiveResponse(statusCode int, body string) string {
	return classifyLiveResponse(statusCode, body)
}

// BuildFullCookieStr rebuilds the FullCookie string from all populated fields
// of an IGSession. Useful after enriching a session with jar-sourced cookies.
func BuildFullCookieStr(s IGSession) string {
	var parts []string
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
	return strings.Join(parts, ";")
}

// CheckLiveByCookie kiểm tra trạng thái account bằng cookie của chính account đó.
// Gọi GET i.instagram.com/api/v1/accounts/current_user/?edit=true với Safari iOS TLS.
// Chính xác hơn CheckLiveByCheckerCookie vì test trực tiếp session cookie.
// Trả: "live" | "checkpoint" | "suspended" | "die" | "unknown".
func CheckLiveByCookie(ctx context.Context, cookie, userAgent, proxyStr string) string {
	if cookie == "" {
		return "unknown"
	}
	csrf := cookieField(decodeCookieValues(cookie), "csrftoken")
	if userAgent == "" {
		userAgent = "Instagram 319.0.0.34.109 (iPhone14,3; iOS 15_8_3; en_US; en-US; scale=3.00; 1284x2778; 545155814)"
	}
	sess, err := newIGSession(proxyStr)
	if err != nil {
		return "unknown"
	}
	defer sess.client.CloseIdleConnections() // tránh leak conn + readLoop goroutine mỗi lần check
	req, err := fhttp.NewRequestWithContext(ctx, "GET",
		igHost+"/api/v1/accounts/current_user/?edit=true", nil)
	if err != nil {
		return "unknown"
	}
	req.Header[fhttp.HeaderOrderKey] = []string{
		"cookie", "user-agent", "x-ig-app-id", "x-csrftoken", "accept", "accept-encoding",
	}
	req.Header.Set("cookie", cookie)
	req.Header.Set("user-agent", userAgent)
	req.Header.Set("x-ig-app-id", igAppID)
	req.Header.Set("x-csrftoken", csrf)
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-encoding", "zstd")

	resp, err := sess.client.Do(req)
	if err != nil {
		return "unknown"
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	body := sess.decode(resp.Header.Get("Content-Encoding"), raw)
	return classifyLiveResponse(resp.StatusCode, body)
}

// CheckLiveByBearer kiểm tra account còn sống bằng Bearer token (IGT:2:...) qua
// GET current_user. Dùng cho account SPC (con) — con chỉ có sessionid/bearer,
// KHÔNG có username thật để dùng CheckLiveByUsername, và cookie-only check không đủ
// auth cho IG bearer account. Header android khớp ngữ cảnh gốc của bearer (SPC).
// Trả: "live" | "die" | "suspended" | "checkpoint" | "unknown".
func CheckLiveByBearer(ctx context.Context, bearer, proxyStr string) string {
	bearer = strings.TrimSpace(bearer)
	if bearer == "" {
		return "unknown"
	}
	if !strings.HasPrefix(bearer, "Bearer ") {
		bearer = "Bearer " + bearer
	}
	sess, err := newIGSession(proxyStr)
	if err != nil {
		return "unknown"
	}
	defer sess.client.CloseIdleConnections()
	req, err := fhttp.NewRequestWithContext(ctx, "GET",
		igHost+"/api/v1/accounts/current_user/?edit=true", nil)
	if err != nil {
		return "unknown"
	}
	req.Header[fhttp.HeaderOrderKey] = []string{
		"authorization", "user-agent", "x-ig-app-id", "accept", "accept-encoding",
	}
	req.Header.Set("authorization", bearer)
	req.Header.Set("user-agent", "Instagram 421.0.0.51.66 Android (35/15; 450dpi; 1080x2400; samsung; SM-G996B; t2s; exynos2100; en_GB; 909555893)")
	req.Header.Set("x-ig-app-id", "567067343352427")
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-encoding", "zstd")

	resp, err := sess.client.Do(req)
	if err != nil {
		return "unknown"
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	body := sess.decode(resp.Header.Get("Content-Encoding"), raw)
	return classifyLiveResponse(resp.StatusCode, body)
}

// CheckLiveByCheckerCookie kiểm tra username có tồn tại không bằng cách dùng
// cookie của 1 account checker (không phải cookie của nick cần check).
// Gọi GET /api/v1/users/web_profile_info/?username=X với Chrome TLS fingerprint.
// Trả: "live" | "die" | "unknown".
func CheckLiveByCheckerCookie(ctx context.Context, checkerCookie, username, proxyStr string) string {
	if checkerCookie == "" || username == "" {
		return "unknown"
	}
	decoded := decodeCookieValues(checkerCookie)
	csrf := cookieField(decoded, "csrftoken")

	sess, err := newChromeCheckSession(proxyStr)
	if err != nil {
		return "unknown"
	}
	defer sess.client.CloseIdleConnections()

	endpoint := "https://www.instagram.com/api/v1/users/web_profile_info/?username=" + username
	req, err := fhttp.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "unknown"
	}
	req.Header[fhttp.HeaderOrderKey] = []string{
		"accept", "accept-language", "cookie", "dpr", "referer",
		"sec-ch-prefers-color-scheme", "sec-ch-ua", "sec-ch-ua-mobile",
		"sec-ch-ua-platform", "sec-fetch-dest", "sec-fetch-mode",
		"sec-fetch-site", "user-agent", "viewport-width",
		"x-asbd-id", "x-csrftoken", "x-ig-app-id", "x-requested-with",
	}
	// sec-ch-ua PHẢI khớp version Chrome thật của sess.checkUA (đồng bộ với TLS
	// ClientProfile đã chọn ở newChromeCheckSession) — tránh mismatch UA/TLS/sec-ch-ua.
	chromeVer := chromeMajorVersionFromUA(sess.checkUA)
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "vi-VN,vi;q=0.9,en-US;q=0.6,en;q=0.5")
	req.Header.Set("cookie", decoded)
	req.Header.Set("dpr", "1")
	req.Header.Set("referer", "https://www.instagram.com/"+username+"/")
	req.Header.Set("sec-ch-prefers-color-scheme", "light")
	req.Header.Set("sec-ch-ua", fmt.Sprintf(`"Google Chrome";v="%s", "Chromium";v="%s", "Not(A:Brand";v="99"`, chromeVer, chromeVer))
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("user-agent", sess.checkUA)
	req.Header.Set("viewport-width", "887")
	req.Header.Set("x-asbd-id", "359341")
	req.Header.Set("x-csrftoken", csrf)
	req.Header.Set("x-ig-app-id", "936619743392459")
	req.Header.Set("x-requested-with", "XMLHttpRequest")

	resp, err := sess.client.Do(req)
	if err != nil {
		return "unknown"
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	body := sess.decode(resp.Header.Get("Content-Encoding"), raw)

	if resp.StatusCode == 429 {
		return "unknown"
	}
	if resp.StatusCode == 404 {
		return "die"
	}

	// {"data":{"user":null},...}  → die
	// {"data":{"user":{"id":...}} → live
	var result struct {
		Data struct {
			User *json.RawMessage `json:"user"`
		} `json:"data"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(body), &result); err == nil {
		if result.Data.User == nil || string(*result.Data.User) == "null" {
			return "die"
		}
		return "live"
	}
	return "unknown"
}

// CheckLiveByUsername kiểm tra tài khoản IG còn live bằng cách đọc HTML title
// của trang profile công khai. Không cần cookie — dùng anonymous Chrome request.
// Chuẩn nhất cho check sau reg vì kiểm tra profile thực tế, không phải session.
// Trả: "live" | "die" | "unknown".
func CheckLiveByUsername(ctx context.Context, username, proxyStr string) string {
	if username == "" {
		return "unknown"
	}
	sess, err := newChromeCheckSession(proxyStr)
	if err != nil {
		return "unknown"
	}
	defer sess.client.CloseIdleConnections()
	req, err := fhttp.NewRequestWithContext(ctx, "GET",
		"https://www.instagram.com/"+username+"/", nil)
	if err != nil {
		return "unknown"
	}
	req.Header[fhttp.HeaderOrderKey] = []string{
		"user-agent", "accept", "accept-language", "accept-encoding",
		"sec-fetch-dest", "sec-fetch-mode", "sec-fetch-site", "sec-fetch-user",
	}
	// UA khớp với TLS ClientProfile ngẫu nhiên đã chọn (xem newChromeCheckSession)
	// — tránh lặp UA/TLS giống hệt nhau hàng nghìn lần khi check-live volume lớn.
	req.Header.Set("user-agent", sess.checkUA)
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("accept-language", "vi-VN,vi;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("accept-encoding", "gzip, deflate, br")
	req.Header.Set("sec-fetch-dest", "document")
	req.Header.Set("sec-fetch-mode", "navigate")
	req.Header.Set("sec-fetch-site", "none")
	req.Header.Set("sec-fetch-user", "?1")

	resp, err := sess.client.Do(req)
	if err != nil {
		return "unknown"
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 110*1024))
	html := sess.decode(resp.Header.Get("Content-Encoding"), raw)

	start := strings.Index(html, "<title>")
	if start == -1 {
		return "unknown"
	}
	ts := start + len("<title>")
	end := strings.Index(html[ts:], "</title>")
	if end == -1 {
		return "unknown"
	}
	title := strings.TrimSpace(html[ts : ts+end])
	if title == "" || title == "Instagram" {
		return "die"
	}
	// Catch "Sorry, this page isn't available." và "Page Not Found" dạng die
	lower := strings.ToLower(title)
	if strings.Contains(lower, "isn't available") || strings.Contains(lower, "page not found") {
		return "die"
	}
	return "live"
}

// prefetchCSRFToken lấy csrftoken từ www.instagram.com/accounts/login/
// bằng cách đọc Set-Cookie header — không cần login.
func prefetchCSRFToken(ctx context.Context, sess *igSession) string {
	req, err := fhttp.NewRequestWithContext(ctx, "GET",
		"https://www.instagram.com/accounts/login/", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Cookie", "ig_cb=1")
	req.Header.Set("User-Agent", igUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := sess.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))

	for _, cookie := range resp.Cookies() {
		if cookie.Name == "csrftoken" && cookie.Value != "" {
			return cookie.Value
		}
	}
	// fallback: parse Set-Cookie header thủ công
	for _, h := range resp.Header["Set-Cookie"] {
		for _, part := range strings.Split(h, ";") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "csrftoken=") {
				return strings.TrimPrefix(part, "csrftoken=")
			}
		}
	}
	return ""
}

// decodeCookieValues decode URL-encoded cookie values (sessionid có %3A → :).
func decodeCookieValues(cookieStr string) string {
	parts := strings.Split(cookieStr, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		idx := strings.IndexByte(p, '=')
		if idx < 0 {
			out = append(out, p)
			continue
		}
		key, val := p[:idx], p[idx+1:]
		if dec, err := url.QueryUnescape(val); err == nil {
			val = dec
		}
		out = append(out, key+"="+val)
	}
	return strings.Join(out, "; ")
}

// cookieField lấy value của field trong cookie string.
func cookieField(cookieStr, field string) string {
	prefix := field + "="
	for _, p := range strings.Split(cookieStr, ";") {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, prefix) {
			return strings.TrimPrefix(p, prefix)
		}
	}
	return ""
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
