// http.go — Web Android HTTP transport + request headers + session pool.
//
// File này gộp 3 file cũ:
//   - httpclient.go  → session (tls-client Chrome_120) + cookie helpers + seedCookies
//   - headers.go     → buildNavHeaders + buildPostHeaders + sec-ch-ua* device headers
//   - sessionpool.go → SessionPool + SharedSessionPool (keep HTTP session per proxy)
//
// Dùng bogdanfinn/tls-client với Chrome_120 profile (Chrome browser TLS fingerprint).
// Cookie jar được chia sẻ giữa GET và POST trong cùng một session.
package webandroid

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"

	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/instagram/register/android"
	"HVRIns/internal/proxy"
)

// SharedPool là partitioned datr pool dùng chung — set từ app.go trước khi chạy reg.
// Cùng type với s23/android để tái sử dụng RecordResult/GetStats tracking.
var SharedPool *android.PartitionedDatrPool

// ─── Session (Chrome_120 TLS + cookie jar) ───────────────────────────────────

// session giữ HTTP client + cookie jar dùng chung cho cả flow.
type session struct {
	client   tls_client.HttpClient
	finalURL string // URL sau redirect (httpResponse.Target trong C#)
	proxy    string // proxy string gốc — cần giữ để dùng cho Android token fetch sau reg
}

// newSession tạo client mới với Chrome_120 profile + cookie jar tự động.
func newSession(proxyStr string) (*session, error) {
	jar := tls_client.NewCookieJar()
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_120),
		tls_client.WithCookieJar(jar),
		tls_client.WithInsecureSkipVerify(),
	}
	if proxyStr != "" {
		if formatted := proxy.FormatProxyURL(proxyStr); formatted != "" {
			opts = append(opts, tls_client.WithProxyUrl(formatted))
		}
	}
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return nil, fmt.Errorf("create chrome tls client: %w", err)
	}
	return &session{client: c, proxy: proxyStr}, nil
}

// proxyStr trả về proxy đã dùng cho session — giữ để fetch Android token sau reg.
func (s *session) proxyStr() string { return s.proxy }

// get thực hiện GET request — follow redirects tự động, lưu finalURL sau redirect.
func (s *session) get(ctx context.Context, targetURL string, headers [][2]string) (string, error) {
	req, err := fhttp.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("create GET request: %w", err)
	}
	applyHeaders(req, headers)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.Request != nil && resp.Request.URL != nil {
		s.finalURL = resp.Request.URL.String()
	} else {
		s.finalURL = targetURL
	}

	data, err := io.ReadAll(resp.Body)
	return string(data), err
}

// post thực hiện POST request.
func (s *session) post(ctx context.Context, targetURL, body string, headers [][2]string) (string, error) {
	req, err := fhttp.NewRequestWithContext(ctx, "POST", targetURL, strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create POST request: %w", err)
	}
	applyHeaders(req, headers)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.Request != nil && resp.Request.URL != nil {
		s.finalURL = resp.Request.URL.String()
	} else {
		s.finalURL = targetURL
	}

	data, err := io.ReadAll(resp.Body)
	return string(data), err
}

// seedCookies — PORT CHÍNH XÁC từ C# WebAndroid Register() lines 72-96.
// Hỗ trợ 3 format MachineId:
//
//  1. "datr_value"                   → add cookie datr=value
//  2. "datr=xxx;sb=yyy;fr=zzz"      → parse full string, add tất cả (trừ c_user, locale)
//  3. "datr=xxx;sb=yyy|uid|pass"    → tách phần chứa "datr=", parse và add
func seedCookies(sess *session, machineID string) {
	if machineID == "" {
		return
	}
	if strings.Contains(machineID, "datr=") {
		// Full cookie string — C#: httpRequest.AddCookies(cleaned_cookie_str, "facebook.com")
		cookieStr := machineID
		if strings.Contains(machineID, "|") {
			// C#: accountInfo.MachineId.Split('|').Where(x => x.Contains("datr=")).First()
			for _, part := range strings.Split(machineID, "|") {
				if strings.Contains(part, "datr=") {
					cookieStr = part
					break
				}
			}
		}
		// C#: ckinitial.Split(';').Where(x => !IsNullOrEmpty && !Contains("c_user") && !Contains("locale="))
		for _, pair := range strings.Split(cookieStr, ";") {
			pair = strings.TrimSpace(pair)
			if pair == "" || strings.Contains(pair, "c_user") || strings.Contains(pair, "locale=") {
				continue
			}
			kv := strings.SplitN(pair, "=", 2)
			if len(kv) == 2 {
				sess.addCookie(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
			}
		}
	} else {
		// Simple datr value — C#: httpRequest.AddCookie(new Cookie { Name="datr", Value=machineId })
		sess.addCookie("datr", machineID)
	}
}

// getCookiesStr lấy tất cả cookie từ jar cho domain facebook.com + m.facebook.com.
// Mapping từ C#: httpRequest.GetCookies("https://facebook.com").
func (s *session) getCookiesStr() string {
	seen := map[string]bool{}
	parts := make([]string, 0)
	for _, rawURL := range []string{"https://m.facebook.com", "https://facebook.com"} {
		u, _ := url.Parse(rawURL)
		for _, c := range s.client.GetCookies(u) {
			if !seen[c.Name] {
				seen[c.Name] = true
				parts = append(parts, c.Name+"="+c.Value)
			}
		}
	}
	return strings.Join(parts, ";")
}

// getCookiesFBOrder trả cookie theo thứ tự FIX của C# WebAndroid:
//
//	datr={datr};sb={sb};c_user={c_user};xs={xs};fr={fr};pas={pas}
//
// Port C# Register L228. `xs` được double URL-decode (C# L226).
// Các cookie vắng mặt bị bỏ qua (giữ format sạch, không có `x=;` trống).
func (s *session) getCookiesFBOrder() string {
	vals := map[string]string{}
	// C# GetCookies("https://facebook.com") — jar trả cookie của domain .facebook.com
	// Go cookie jar: check cả m.facebook.com + facebook.com để union tất cả.
	for _, rawURL := range []string{"https://facebook.com", "https://m.facebook.com"} {
		u, _ := url.Parse(rawURL)
		for _, c := range s.client.GetCookies(u) {
			if _, ok := vals[c.Name]; !ok {
				vals[c.Name] = c.Value
			}
		}
	}

	// xs double URL-decode — C# WebUtility.UrlDecode(WebUtility.UrlDecode(xs))
	if xs, ok := vals["xs"]; ok {
		if d1, err := url.QueryUnescape(xs); err == nil {
			if d2, err := url.QueryUnescape(d1); err == nil {
				vals["xs"] = d2
			} else {
				vals["xs"] = d1
			}
		}
	}

	// Build theo thứ tự cố định C#. Bỏ qua key có value rỗng.
	order := []string{"datr", "sb", "c_user", "xs", "fr", "pas"}
	parts := make([]string, 0, len(order))
	for _, k := range order {
		if v := vals[k]; v != "" {
			parts = append(parts, k+"="+v)
		}
	}
	return strings.Join(parts, ";")
}

// addCookie thêm cookie vào jar trước khi gửi request.
// Dùng Domain ".facebook.com" để cookie được gửi tới m.facebook.com.
func (s *session) addCookie(name, value string) {
	u, _ := url.Parse("https://m.facebook.com")
	s.client.SetCookies(u, []*fhttp.Cookie{
		{Name: name, Value: value, Path: "/", Domain: ".facebook.com"},
	})
}

// clearCookies xoá cookies facebook.com cho reg kế. Tránh leak c_user/xs
// của account vừa reg sang account tiếp theo khi WorkerContext reuse session.
// Cũng reset finalURL để Referer computed lại từ đầu.
func (s *session) clearCookies() {
	urls := []string{
		"https://m.facebook.com",
		"https://facebook.com",
		"https://www.facebook.com",
	}
	for _, rawURL := range urls {
		u, _ := url.Parse(rawURL)
		existing := s.client.GetCookies(u)
		if len(existing) == 0 {
			continue
		}
		expired := make([]*fhttp.Cookie, 0, len(existing))
		for _, c := range existing {
			expired = append(expired, &fhttp.Cookie{Name: c.Name, Value: "", Path: "/", Domain: c.Domain, MaxAge: -1})
		}
		s.client.SetCookies(u, expired)
	}
	s.finalURL = ""
}

// applyHeaders set headers vào request (giữ nguyên case).
func applyHeaders(req *fhttp.Request, headers [][2]string) {
	order := make([]string, 0, len(headers))
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
		order = append(order, kv[0])
	}
	req.Header[fhttp.HeaderOrderKey] = order
}

// ─── Headers (Chrome Android browser, sec-ch-ua* mobile) ─────────────────────
//
// PORT CHÍNH XÁC từ C#:
//
//	PerpectChromeAndroidNavHeadersFormat2(referer, origin, account) — cho GET requests
//	PerpectChromeAndroidPostHeadersFormat2(referer, origin, account) — cho POST request
//	ChromeVersionAndDeviceInfoHeader(account) — sec-ch-ua* headers

// buildNavHeaders tạo headers cho GET m.facebook.com/.
// PORT CHÍNH XÁC: PerpectChromeAndroidNavHeadersFormat2(referer, origin, accountInfo).
// Header order C#: Accept → upgrade → sec-fetch-* → dpr → viewport → referer → origin → chromeDevice → accept-language → priority.
func buildNavHeaders(prof fakeinfo.ChromeAndroidProfile, referer, origin string) [][2]string {
	h := [][2]string{
		{"Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		{"upgrade-insecure-requests", "1"},
		{"sec-fetch-site", "none"},
		{"sec-fetch-mode", "navigate"},
		{"sec-fetch-user", "?1"},
		{"sec-fetch-dest", "document"},
		{"dpr", prof.Dpr},
		{"viewport-width", prof.ViewportWidth},
	}
	if referer != "" {
		h = append(h, [2]string{"referer", referer})
	}
	if origin != "" {
		h = append(h, [2]string{"origin", origin})
	}
	h = append(h, chromeDeviceHeaders(prof)...)
	h = append(h,
		[2]string{"accept-language", "en-US,en;q=0.9,vi-VN;q=0.8,vi;q=0.7"},
		[2]string{"priority", "u=0, i"},
		[2]string{"User-Agent", prof.UserAgent},
	)
	return h
}

// buildPostHeaders tạo headers cho POST /async/wbloks/fetch/.
// PORT CHÍNH XÁC: PerpectChromeAndroidPostHeadersFormat2(referer, origin, accountInfo).
// Header order C#: accept → sec-fetch-* → referer → origin → chromeDevice → accept-language → priority.
func buildPostHeaders(prof fakeinfo.ChromeAndroidProfile, referer, origin string) [][2]string {
	h := [][2]string{
		{"accept", "*/*"},
		{"sec-fetch-site", "same-origin"},
		{"sec-fetch-mode", "cors"},
		{"sec-fetch-dest", "empty"},
	}
	if referer != "" {
		h = append(h, [2]string{"referer", referer})
	}
	if origin != "" {
		h = append(h, [2]string{"origin", origin})
	}
	h = append(h, chromeDeviceHeaders(prof)...)
	h = append(h,
		[2]string{"accept-language", "en-US,en;q=0.9,vi-VN;q=0.8,vi;q=0.7"},
		[2]string{"priority", "u=1, i"},
		[2]string{"User-Agent", prof.UserAgent},
		[2]string{"Content-Type", "application/x-www-form-urlencoded"},
	)
	return h
}

// chromeDeviceHeaders tạo sec-ch-ua* headers từ ChromeAndroidProfile.
// PORT CHÍNH XÁC: ChromeVersionAndDeviceInfoHeader(account).
func chromeDeviceHeaders(prof fakeinfo.ChromeAndroidProfile) [][2]string {
	secCHUA := fmt.Sprintf(`"Chromium";v="%s", "Google Chrome";v="%s", "Not-A.Brand";v="99"`,
		prof.ChromeVersion, prof.ChromeVersion)
	secCHUAFull := fmt.Sprintf(`"Chromium";v="%s", "Google Chrome";v="%s", "Not-A.Brand";v="99.0.0.0"`,
		prof.ChromeVersionFull, prof.ChromeVersionFull)

	h := [][2]string{
		{"sec-ch-ua", secCHUA},
		{"sec-ch-ua-mobile", "?1"},
	}
	if prof.AndroidModel != "" {
		h = append(h,
			[2]string{"sec-ch-ua-platform", `"Android"`},
			[2]string{"sec-ch-ua-platform-version", fmt.Sprintf(`"%s"`, prof.AndroidOsVersion)},
			[2]string{"sec-ch-ua-model", fmt.Sprintf(`"%s"`, prof.AndroidModel)},
		)
	}
	h = append(h,
		[2]string{"sec-ch-ua-full-version-list", secCHUAFull},
		[2]string{"sec-ch-prefers-color-scheme", "light"},
	)
	return h
}

// ─── SessionPool (keep HTTP session across regs per proxy) ───────────────────
//
// PORT từ C#: RegisterWithKeepHttpSession — cùng proxy dùng lại session cũ.

// SharedSessionPool là pool session dùng chung — set từ app.go trước khi chạy reg.
var SharedSessionPool *SessionPool

// SessionPool quản lý session per proxy key.
type SessionPool struct {
	mu       sync.Mutex
	sessions map[string]*session
}

// NewSessionPool tạo pool mới.
func NewSessionPool() *SessionPool {
	return &SessionPool{
		sessions: make(map[string]*session),
	}
}

// Acquire lấy session cho proxy, trả (session, isFirst).
func (p *SessionPool) Acquire(proxyStr string) (*session, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := proxyStr
	if key == "" {
		key = "__direct__"
	}
	if existing, ok := p.sessions[key]; ok {
		return existing, false
	}
	return nil, true
}

// Store lưu session vào pool.
func (p *SessionPool) Store(proxyStr string, sess *session) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := proxyStr
	if key == "" {
		key = "__direct__"
	}
	p.sessions[key] = sess
}

// Remove xóa session.
func (p *SessionPool) Remove(proxyStr string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := proxyStr
	if key == "" {
		key = "__direct__"
	}
	delete(p.sessions, key)
}

// CloseIdleConnsAll close idle TCP/TLS connection của TẤT CẢ session trong pool.
// Restored từ HEAD để app.go cleanup compile được.
func (p *SessionPool) CloseIdleConnsAll() int {
	p.mu.Lock()
	sessions := make([]*session, 0, len(p.sessions))
	for _, s := range p.sessions {
		sessions = append(sessions, s)
	}
	p.mu.Unlock()

	n := 0
	for _, sess := range sessions {
		if sess != nil && sess.client != nil {
			sess.client.CloseIdleConnections()
			n++
		}
	}
	return n
}

// CloseAll đóng toàn bộ session trong pool — gọi khi register run kết thúc.
func (p *SessionPool) CloseAll() int {
	p.mu.Lock()
	sessions := p.sessions
	p.sessions = make(map[string]*session)
	p.mu.Unlock()

	n := 0
	for _, sess := range sessions {
		if sess != nil && sess.client != nil {
			sess.client.CloseIdleConnections()
			n++
		}
	}
	return n
}

// Size trả về số session đang giữ trong pool.
func (p *SessionPool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.sessions)
}
