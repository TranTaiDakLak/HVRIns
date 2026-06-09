// Package verifybase contains shared helpers for all verify variant packages.
// Variant-specific body builders, header builders and constants live in the
// individual sXXX packages; the common orchestration lives in RunVerify.
package verifybase

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	mrand "math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/google/uuid"

	"HVRIns/internal/email"
	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

// ─── OTP heartbeat ──────────────────────────────────────────────────────────────

// StartOTPHeartbeat phát status định kỳ trong lúc CHỜ OTP (WaitForCode block im lặng,
// poll mail + đợi FB gửi code không emit gì → UI nhìn như treo). Mục đích: cột HOẠT ĐỘNG
// luôn nhúc nhích, hiển thị số giây đã chờ.
//
// every: chu kỳ phát (vd 5s). tag: prefix log ("[s562]"...). mail: địa chỉ đang chờ OTP.
// Trả về stop func — PHẢI gọi sau khi WaitForCode trả về để dừng goroutine heartbeat.
func StartOTPHeartbeat(ctx context.Context, notify func(string), every time.Duration, tag, mail string) func() {
	if notify == nil || every <= 0 {
		return func() {}
	}
	hbCtx, cancel := context.WithCancel(ctx)
	go func() {
		start := time.Now()
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-ticker.C:
				notify(fmt.Sprintf("%s ⏳ Đang đọc mail chờ OTP... (%ds) [%s]",
					tag, int(time.Since(start).Seconds()), mail))
			}
		}
	}()
	return cancel
}

// ─── HTTP client ──────────────────────────────────────────────────────────────

// CreateClient creates a tls-client for verify requests with optional proxy.
func CreateClient(proxyStr string) (tls_client.HttpClient, error) {
	return createClientWithProfile(proxyStr, profiles.Okhttp4Android13)
}

// CreateIOSClient creates a tls-client with Safari iOS TLS fingerprint for iOS verify.
func CreateIOSClient(proxyStr string) (tls_client.HttpClient, error) {
	return createClientWithProfile(proxyStr, profiles.Safari_IOS_15_6)
}

func createClientWithProfile(proxyStr string, profile profiles.ClientProfile) (tls_client.HttpClient, error) {
	jar := tls_client.NewCookieJar()
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profile),
		tls_client.WithCookieJar(jar),
		tls_client.WithInsecureSkipVerify(),
		tls_client.WithNotFollowRedirects(),
	}
	if proxyStr != "" {
		if f := proxy.FormatProxyURL(proxyStr); f != "" {
			opts = append(opts, tls_client.WithProxyUrl(f))
		}
	}
	return tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
}

// DoPost sends a gzip-compressed POST with the given ordered headers and returns the response body.
func DoPost(ctx context.Context, client tls_client.HttpClient, targetURL, body string, headers [][2]string) (string, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(body)); err != nil {
		return "", fmt.Errorf("gzip write: %v", err)
	}
	if err := gz.Close(); err != nil {
		return "", fmt.Errorf("gzip close: %v", err)
	}

	req, err := fhttp.NewRequestWithContext(ctx, "POST", targetURL, &buf)
	if err != nil {
		return "", err
	}

	order := make([]string, 0, len(headers))
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
		order = append(order, kv[0])
	}
	req.Header[fhttp.HeaderOrderKey] = order

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	data, err := httpx.ReadBody(resp.Body, 512*1024)
	respStr := string(data)

	// Detect checkpoint via response header — port C# FacebookCheckpointDetectorUtils.
	if integrity := resp.Header.Get("X-Fb-Integrity-Required"); integrity != "" {
		if strings.Contains(strings.ToLower(integrity), "checkpoint") {
			respStr = `{"error":{"code":459,"message":"checkpointed"}}` + respStr
		}
	}
	if resp.Header.Get("X-Fb-Integrity-Requires-Login") != "" {
		respStr = `{"error":{"message":"checkpointed"}}` + respStr
	}

	if resp.StatusCode >= 400 {
		return respStr, fmt.Errorf("HTTP %d: %s", resp.StatusCode, respStr[:Mmin(len(respStr), 300)])
	}
	return respStr, err
}

// ─── Email helpers ────────────────────────────────────────────────────────────

// RetryCreateEmail tries to create an email address up to 3 times.
func RetryCreateEmail(ctx context.Context, svc email.Service, notify func(string)) (string, error) {
	for i := 1; i <= 3; i++ {
		if ctx.Err() != nil {
			return "", fmt.Errorf("cancelled")
		}
		if i > 1 {
			notify(fmt.Sprintf("[Mail] Retry %d/3...", i))
			select {
			case <-time.After(2 * time.Second):
			case <-ctx.Done():
				return "", fmt.Errorf("cancelled")
			}
		}
		addr, err := svc.CreateEmail(ctx)
		if err == nil && addr != "" {
			return addr, nil
		}
		if err != nil {
			notify(fmt.Sprintf("[Mail] Error: %v", err))
		}
	}
	return "", fmt.Errorf("create email failed after 3 retries")
}

// ─── Cookie helpers ───────────────────────────────────────────────────────────

// ExtractDatrFromCookieStr extracts datr value from "datr=xxx; sb=yyy" format.
func ExtractDatrFromCookieStr(cookieStr string) string {
	for _, part := range strings.Split(cookieStr, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "datr=") {
			return strings.TrimPrefix(part, "datr=")
		}
	}
	return ""
}

// InjectVerifyCookies injects datr, sb, fr from account cookie string into the verify client.
func InjectVerifyCookies(client tls_client.HttpClient, cookieStr string) {
	if cookieStr == "" {
		return
	}
	allow := map[string]bool{"datr": true, "sb": true, "fr": true}
	fbURL, _ := url.Parse("https://b-graph.facebook.com")
	var cookies []*fhttp.Cookie
	for _, part := range strings.Split(cookieStr, ";") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		name := strings.TrimSpace(kv[0])
		if allow[name] {
			cookies = append(cookies, &fhttp.Cookie{Name: name, Value: strings.TrimSpace(kv[1])})
		}
	}
	if len(cookies) > 0 {
		client.GetCookieJar().SetCookies(fbURL, cookies)
	}
}

// ExtractLocaleFromCookie finds cookie `locale=xx_YY` in the cookie string.
func ExtractLocaleFromCookie(cookieStr string) string {
	for _, part := range strings.Split(cookieStr, ";") {
		part = strings.TrimSpace(part)
		if kv := strings.SplitN(part, "=", 2); len(kv) == 2 && strings.TrimSpace(kv[0]) == "locale" {
			return strings.TrimSpace(kv[1])
		}
	}
	return ""
}

// ─── Live/Die check ───────────────────────────────────────────────────────────

var tokenCheckClient = &http.Client{Timeout: 15 * time.Second}

// defaultFallbackUA returns a UA from the Android pool — avoids hardcoding.
func defaultFallbackUA() string {
	if ua := fakeinfo.RandomUAFromPool(fakeinfo.UAKindAndroid); ua != "" {
		return ua
	}
	return fakeinfo.BuildAndroidUAWithOpts(fakeinfo.RandomDeviceProfile(), "en_US", "", "", "", false, false)
}

// pictureCheckClient — client KHÔNG follow redirect, dùng cho check live/die qua picture endpoint.
// Logic: GET /picture?type=normal trả 302 Location → check Location header.
//   - Location chứa "C5yt7Cqf3zU.jpg" (default avatar) → Die
//   - Location chứa "scontent.*.fbcdn.net" (real CDN) → Live
var pictureCheckClient = &http.Client{
	Timeout: 15 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse // KHÔNG follow redirect — giữ 302 response
	},
}

// CheckLiveDieByToken kiểm tra account qua /me với access token.
// CHÍNH XÁC HƠN picture check vì FB invalidate token NGAY KHI checkpoint
// (picture endpoint delay 30-60 phút mới catch up).
//
// Returns:
//   - "Live" — response có {"id":"...","name":"..."} (token còn work, account active)
//   - "Die"  — response có "OAuthException" / "checkpoint" / "error"
//   - "Unknown" — network error / token rỗng / không xác định
func CheckLiveDieByToken(ctx context.Context, ua, token string) string {
	if strings.TrimSpace(token) == "" {
		return "Unknown"
	}
	url := "https://graph.facebook.com/me?fields=id,name&access_token=" + token
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "Unknown"
	}
	// Dùng ĐÚNG UA của account đang verify (iOS → FBIOS, Android → FB4A, web → UA web).
	// Trước đây hardcode UA Windows Chrome → FB ghi session "/me" thành Windows Desktop,
	// lệch hẳn dấu vân tay thiết bị (vd account verify iOS lại hiện "Nơi đăng nhập" là Windows).
	// Fallback chỉ khi ua rỗng (hiếm) — UA mobile trung tính, KHÔNG ép platform nào.
	if strings.TrimSpace(ua) == "" {
		ua = "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390"
	}
	req.Header.Set("User-Agent", ua)
	resp, err := tokenCheckClient.Do(req)
	if err != nil {
		return "Unknown"
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	ret := string(body)
	if ret == "" {
		return "Unknown"
	}
	low := strings.ToLower(ret)
	// Die markers — checkpoint / token invalid / disabled
	if strings.Contains(low, "oauthexception") ||
		strings.Contains(low, "checkpoint") ||
		strings.Contains(low, "session is invalid") ||
		strings.Contains(low, "session has expired") ||
		strings.Contains(low, "account is disabled") ||
		strings.Contains(low, "account has been disabled") {
		return "Die"
	}
	// Live marker — response có "id" + "name" field
	if strings.Contains(ret, `"id"`) && strings.Contains(ret, `"name"`) {
		return "Live"
	}
	// Generic error fallback
	if strings.Contains(low, `"error"`) {
		return "Die"
	}
	return "Unknown"
}

// CheckLiveDieCombined dùng cả 2 method: token check (gold standard) + picture check.
// Logic priority:
//  1. Có token → ưu tiên token check (catch checkpoint NGAY LẬP TỨC)
//  2. Token check "Die" → return Die (token invalid = chắc chắn die)
//  3. Token check "Live" → return Live (token work = chắc chắn live)
//  4. Token check "Unknown" hoặc không có token → fallback picture check
//
// Khắc phục false positive: picture endpoint delay 30-60 phút mới reflect checkpoint,
// trong khi token check catch ngay.
func CheckLiveDieCombined(ctx context.Context, ua, uid, token string) string {
	if token != "" {
		if status := CheckLiveDieByToken(ctx, ua, token); status != "Unknown" {
			return status
		}
	}
	// Fallback picture check
	return CheckLiveDieByPicture(ctx, ua, uid)
}

// CheckLiveDiePictureFirst — đảo thứ tự: ƯU TIÊN picture check trước, token check sau.
// Dùng cho iOS verify (inject qua Spec.CheckLiveDieFunc).
// Logic:
//  1. Có uid → picture check trước; trả về nếu ra Live/Die rõ ràng.
//  2. Picture "Unknown" hoặc không có uid → fallback token check.
//
// LƯU Ý: picture endpoint trễ 30-60 phút mới reflect checkpoint, nên picture-first
// có thể báo Live cho account vừa bị checkpoint (token check sẽ bắt ngay). Đây là
// đánh đổi có chủ đích theo yêu cầu — token vẫn là lưới fallback.
func CheckLiveDiePictureFirst(ctx context.Context, ua, uid, token string) string {
	if uid != "" {
		if status := CheckLiveDieByPicture(ctx, ua, uid); status != "Unknown" {
			return status
		}
	}
	// Fallback token check
	if token != "" {
		return CheckLiveDieByToken(ctx, ua, token)
	}
	return "Unknown"
}

// CheckLiveDieByPicture checks account status via the Graph API /picture endpoint.
// Returns "Live", "Die", or "Unknown".
//
// Method: GET https://graph.facebook.com/{uid}/picture?type=normal (Chrome browser headers).
// FB trả 302 Location → check Location header:
//   - Location chứa "C5yt7Cqf3zU.jpg" (default silhouette avatar) → Die
//   - Location chứa "scontent.*.fbcdn.net" (real photo CDN) → Live
//
// Fallback: nếu response không phải 302 → fall back đọc body JSON với redirect=false.
//
// LƯU Ý: Picture endpoint có delay 30-60 phút mới reflect checkpoint.
// Để check chính xác hơn cho account vừa verify, dùng CheckLiveDieCombined với token.
func CheckLiveDieByPicture(ctx context.Context, ua, uid string) string {
	if uid == "" {
		return "Unknown"
	}

	if strings.TrimSpace(ua) == "" {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36"
	}

	// === Method 1: 302 Location check (Chrome browser style) ===
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://graph.facebook.com/"+uid+"/picture?type=normal", nil)
	if err != nil {
		return "Unknown"
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Referer", "https://dongvanfb.net/")
	req.Header.Set("sec-ch-ua", `"Chromium";v="148", "Google Chrome";v="148", "Not/A)Brand";v="99"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)

	resp, err := pictureCheckClient.Do(req)
	if err != nil {
		return "Unknown"
	}
	defer resp.Body.Close()

	// 302 redirect → check Location header
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		if location != "" {
			if strings.Contains(location, "/C5yt7Cqf3zU.jpg") {
				return "Die"
			}
			// scontent.*.fbcdn.net = real CDN → có ảnh thật → Live
			if strings.Contains(location, "scontent.") && strings.Contains(location, "fbcdn.net") {
				return "Live"
			}
			// URL lạ khác → optimistic Live
			return "Live"
		}
	}

	// === Method 2 (fallback): JSON body parse với redirect=false ===
	req2, err := http.NewRequestWithContext(ctx, "GET",
		"https://graph.facebook.com/"+uid+"/picture?type=normal&redirect=false", nil)
	if err != nil {
		return "Unknown"
	}
	req2.Header.Set("User-Agent", ua)
	resp2, err := tokenCheckClient.Do(req2)
	if err != nil {
		return "Unknown"
	}
	defer resp2.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp2.Body, 4096))
	ret := string(body)
	if ret == "" {
		return "Unknown"
	}
	if strings.Contains(ret, "/C5yt7Cqf3zU.jpg") || !strings.Contains(ret, "height") {
		return "Die"
	}
	return "Live"
}

// ─── Phone / Name / SIM helpers ───────────────────────────────────────────────

// CountryFromPhone maps phone prefix → ISO country code for SIM selection.
func CountryFromPhone(phone string) string {
	if phone == "" {
		return ""
	}
	if p, ok := fakeinfo.FindCountryByPhonePrefix(phone); ok {
		return p.CountryCode
	}
	return ""
}

// SplitFullName splits "First Last" into (first, last).
func SplitFullName(fullName string) (string, string) {
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return "User", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

// PickCountryCarrierLocale selects carrier and locale by ISO country code.
func PickCountryCarrierLocale(countryCode string) (locale, carrier string) {
	if countryCode == "" {
		return "", ""
	}
	sim := fakeinfo.RandomSimProfile(countryCode)
	if sim.OperatorName != "" {
		carrier = sim.OperatorName
	}
	if l := fakeinfo.LocaleFromCountry(countryCode); l != "" {
		locale = l
	}
	return
}

// ─── JSON / crypto helpers ────────────────────────────────────────────────────

// MustJSON marshals v to JSON string, panics on error.
func MustJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("json.Marshal: %v", err))
	}
	return string(b)
}

// GenUSDID generates a signed USDID token.
func GenUSDID() string {
	id := uuid.New().String()
	ts := fmt.Sprintf("%d", time.Now().Unix())
	payload := id + "." + ts
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return payload
	}
	hash := sha256.Sum256([]byte(payload))
	sig, _ := ecdsa.SignASN1(rand.Reader, key, hash[:])
	return payload + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// GenPinnedConnUUID generates a base64-encoded pinned connection UUID.
func GenPinnedConnUUID() string {
	return base64.StdEncoding.EncodeToString([]byte(uuid.New().String()[:16]))
}

// GenPinnedDeviceGroup generates a random device group string (1000-9999).
func GenPinnedDeviceGroup() string {
	return fmt.Sprintf("%d", 1000+mrand.Intn(9000))
}

// GenPinnedTaLoggingID generates a graphql:<uuid> logging ID.
func GenPinnedTaLoggingID() string {
	return "graphql:" + uuid.New().String()
}

// ─── Network helpers ──────────────────────────────────────────────────────────

// NetworkBSSID returns a random WiFi BSSID consistent with S23 networkSig header.
func NetworkBSSID() string {
	return fmt.Sprintf("7C:7B:68:E7:06:%02X", 50+mrand.Intn(10))
}

// ─── Error formatting ─────────────────────────────────────────────────────────

// reUnicodeEsc matches "\uXXXX" escape sequence (BMP code point in hex).
var reUnicodeEsc = regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)

// DecodeUnicodeEscapes thay tất cả "\uXXXX" trong chuỗi bằng ký tự thật.
// Dùng cho log FB error response — error_msg thường chứa Thai/Arabic/CJK escape thành \uXXXX.
//
// Ví dụ: `{"error_msg":"รหัส..."}` → `{"error_msg":"รหัส..."}`
//
// Chỉ xử lý BMP (1 surrogate). Surrogate pair (emoji) không được decode — giữ nguyên \uXXXX
// (đủ tốt cho log error FB vì hiếm khi chứa emoji).
func DecodeUnicodeEscapes(s string) string {
	if !strings.Contains(s, `\u`) {
		return s
	}
	return reUnicodeEsc.ReplaceAllStringFunc(s, func(m string) string {
		if len(m) != 6 {
			return m
		}
		n, err := strconv.ParseInt(m[2:], 16, 32)
		if err != nil {
			return m
		}
		return string(rune(n))
	})
}

// SummarizeFBError extracts a short error description from an FB API response.
// Tự động decode \uXXXX escape trước khi match pattern + truncate.
func SummarizeFBError(resp string) string {
	if resp == "" {
		return "no response"
	}
	resp = DecodeUnicodeEscapes(resp)
	low := strings.ToLower(resp)
	switch {
	case strings.Contains(low, "field_exception"):
		return "FB server error (field_exception)"
	case strings.Contains(low, "checkpoint") || strings.Contains(low, "is_checkpointed"):
		return "checkpoint required"
	case strings.Contains(low, "session is invalid") || strings.Contains(low, "session has expired"):
		return "session invalid/expired"
	case strings.Contains(low, "account is currently disabled") || strings.Contains(low, "account has been disabled"):
		return "account disabled"
	case strings.Contains(low, "fb_bloks_action") && strings.Contains(low, "errors"):
		return "FB bloks error"
	case strings.Contains(low, "rate limit") || strings.Contains(low, "too many requests"):
		return "rate limited"
	}
	// Truncate theo RUNE (không phải byte) để không cắt giữa multi-byte UTF-8 char (Thai, CJK, ...).
	runes := []rune(resp)
	if len(runes) > 80 {
		return string(runes[:80]) + "..."
	}
	return resp
}

// ─── Misc ─────────────────────────────────────────────────────────────────────

// Mmin returns the smaller of a and b.
func Mmin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
