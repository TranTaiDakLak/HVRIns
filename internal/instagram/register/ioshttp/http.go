// http.go — iOS HTTP (MFB) transport + request headers + session pool.
//
// File này gộp 3 file cũ:
//   - httpclient.go → session struct + newSession/get/post/cookies + applyHeaders
//   - headers.go    → buildNavHeaders + buildPostHeaders (iPhone iOS Mobile Safari)
//   - sessionpool.go → SessionPool + SharedSessionPool (keep HTTP session per proxy)
//
// Closest Go equivalent to C#'s LeafxNetLibWrapper (Leaf.xNet):
//   - Uses bogdanfinn/fhttp STANDALONE (NOT tls-client)
//   - Standard TLS — NO browser fingerprint impersonation (same as Leaf.xNet)
//   - HTTP/1.1 only (same as Leaf.xNet)
//   - Lowercase headers preserved via direct map assignment (same as Leaf.xNet)
//   - Cookie jar + proxy support
//
// WHY fhttp standalone:
//
//	bogdanfinn/tls-client impersonates Safari/Chrome TLS → Facebook detects as bot.
//	fhttp standalone uses standard Go TLS (like Leaf.xNet uses standard .NET TLS)
//	→ Facebook cannot match against known browser fingerprint database → higher trust.
package ioshttp

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/fhttp/cookiejar"
	tls "github.com/bogdanfinn/utls"

	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/instagram/register/android"
	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

// SharedPool là partitioned datr pool dùng chung — set từ app.go trước khi chạy reg.
var SharedPool *android.PartitionedDatrPool

// ─── Session (1 client = 1 account = 1 cookie jar) ───────────────────────────
//
// Maps to C#: LeafxNetLibWrapper — one instance per registration attempt.

// session holds the HTTP client + cookie jar shared across the entire flow.
type session struct {
	client   *fhttp.Client
	jar      *cookiejar.Jar
	finalURL string // Final URL after redirect chain (C#: httpResponse.Target)
}

// newSession creates a fhttp client with standard TLS (no fingerprint impersonation).
// Matches C#: LeafxNetLibWrapper with FollowRedirects=true, standard .NET TLS, HTTP/1.1.
func newSession(proxyStr string) (*session, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}

	transport := &fhttp.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		ForceAttemptHTTP2:   false, // HTTP/1.1 only — matches Leaf.xNet
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     30 * time.Second,
	}

	// Proxy support
	proxyStr = strings.TrimSpace(proxyStr)
	if proxyStr != "" {
		proxyURL := proxy.FormatProxyURL(proxyStr)
		if proxyURL != "" {
			u, parseErr := url.Parse(proxyURL)
			if parseErr == nil {
				transport.Proxy = fhttp.ProxyURL(u)
			}
		}
	}

	client := &fhttp.Client{
		Jar:       jar,
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &session{client: client, jar: jar}, nil
}

// get performs a GET request — follows redirects automatically.
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

	data, err := httpx.ReadBody(resp.Body, 1<<20)
	return string(data), err
}

// post performs a POST request — follows redirects automatically.
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

	data, err := httpx.ReadBody(resp.Body, 1<<20)
	return string(data), err
}

// getCookiesStr returns all cookies from jar for facebook.com + m.facebook.com.
func (s *session) getCookiesStr() string {
	seen := map[string]bool{}
	parts := make([]string, 0)
	for _, rawURL := range []string{"https://m.facebook.com", "https://facebook.com"} {
		u, _ := url.Parse(rawURL)
		for _, c := range s.jar.Cookies(u) {
			if !seen[c.Name] {
				seen[c.Name] = true
				parts = append(parts, c.Name+"="+c.Value)
			}
		}
	}
	return strings.Join(parts, ";")
}

// addCookie adds a single cookie to the jar.
func (s *session) addCookie(name, value string) {
	u, _ := url.Parse("https://m.facebook.com")
	s.jar.SetCookies(u, []*fhttp.Cookie{
		{Name: name, Value: value, Path: "/", Domain: ".facebook.com"},
	})
}

// seedFromParsed applies a parsed Seed to the session.
func (s *session) seedFromParsed(seed Seed) {
	switch seed.Mode {
	case SeedModeDatrOnly:
		if seed.Datr != "" {
			s.addCookie("datr", seed.Datr)
		}
	case SeedModeFullCookie:
		s.seedCookieString(seed.CookieString)
	case SeedModeInitialAccount:
		if seed.CookieString != "" {
			s.seedCookieString(seed.CookieString)
		} else if seed.Datr != "" {
			s.addCookie("datr", seed.Datr)
		}
	}
}

// seedCookieString parses and adds cookies from a semicolon-separated string.
func (s *session) seedCookieString(cookieStr string) {
	for _, pair := range strings.Split(cookieStr, ";") {
		pair = strings.TrimSpace(pair)
		if pair == "" || strings.Contains(pair, "c_user") || strings.Contains(pair, "locale=") {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			s.addCookie(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
		}
	}
}

// applyHeaders sets headers on request preserving EXACT case (lowercase).
// Direct map assignment bypasses fhttp's canonicalization.
// HeaderOrderKey controls wire order — critical for Facebook detection.
func applyHeaders(req *fhttp.Request, headers [][2]string) {
	order := make([]string, 0, len(headers))
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
		order = append(order, kv[0])
	}
	req.Header[fhttp.HeaderOrderKey] = order
}

// ─── Headers (iPhone iOS Mobile Safari) ──────────────────────────────────────
//
// PORT CHÍNH XÁC từ C#:
//
//	PerpectMfbNavHeadersFormat3(referer, origin) — cho GET + update-nonce POST
//	PerpectMfbPostHeadersFormat3(referer, origin) — cho register POST
//
// Khác với WebAndroid: KHÔNG có sec-ch-ua*, dpr, viewport-width (iOS-specific).

// buildNavHeaders tạo headers cho GET requests + update-nonce POST.
// PORT CHÍNH XÁC: PerpectMfbNavHeadersFormat3(referer, origin).
// Header order khớp C#: Accept → upgrade → sec-fetch-* → referer → origin → accept-language → priority.
// User-Agent KHÔNG nằm trong C# Nav headers (set qua DefaultRequestHeaders) — Go thêm cuối.
func buildNavHeaders(prof fakeinfo.IPhoneProfile, referer, origin string) [][2]string {
	h := [][2]string{
		{"Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		{"upgrade-insecure-requests", "1"},
		{"sec-fetch-site", "none"},
		{"sec-fetch-mode", "navigate"},
		{"sec-fetch-user", "?1"},
		{"sec-fetch-dest", "document"},
	}
	if referer != "" {
		h = append(h, [2]string{"referer", referer})
	}
	if origin != "" {
		h = append(h, [2]string{"origin", origin})
	}
	h = append(h,
		[2]string{"accept-language", "en-US,en;q=0.9,vi-VN;q=0.8,vi;q=0.7"},
		[2]string{"priority", "u=0, i"},
		[2]string{"User-Agent", prof.UserAgent},
	)
	return h
}

// buildPostHeaders tạo headers cho POST /async/wbloks/fetch/ và /reg/submit/.
// PORT CHÍNH XÁC: PerpectMfbPostHeadersFormat3(referer, origin).
// C# KHÔNG set User-Agent/Content-Type ở đây (set qua DefaultRequestHeaders/StringContent).
// Go thêm cuối để đảm bảo chúng được gửi.
func buildPostHeaders(prof fakeinfo.IPhoneProfile, referer, origin string) [][2]string {
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
	h = append(h,
		[2]string{"accept-language", "en-US,en;q=0.9,vi-VN;q=0.8,vi;q=0.7"},
		[2]string{"priority", "u=1, i"},
		[2]string{"User-Agent", prof.UserAgent},
		[2]string{"Content-Type", "application/x-www-form-urlencoded"},
	)
	return h
}

// ─── SessionPool (keep HTTP session across regs per proxy) ───────────────────
//
// PORT từ C#: RegisterWithKeepHttpSession — cùng proxy dùng lại session cũ.
// Lần 1 (isFirst=true): tạo session → mồi → register.
// Lần 2+ (isFirst=false): dùng lại session → register trực tiếp (cookies tin cậy).

// SharedSessionPool là pool session dùng chung — set từ app.go trước khi chạy reg.
var SharedSessionPool *SessionPool

// SessionPool quản lý session per proxy key.
type SessionPool struct {
	mu       sync.Mutex
	sessions map[string]*session
	maxUsage int // số lần dùng tối đa per session, 0 = unlimited
}

// NewSessionPool tạo pool mới.
func NewSessionPool(maxUsage int) *SessionPool {
	return &SessionPool{
		sessions: make(map[string]*session),
		maxUsage: maxUsage,
	}
}

// Acquire lấy session cho proxy, trả (session, isFirst).
// isFirst=true nếu session mới tạo hoặc chưa có trong pool.
func (p *SessionPool) Acquire(proxyStr string) (*session, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := proxyStr
	if key == "" {
		key = "__direct__"
	}

	if existing, ok := p.sessions[key]; ok {
		return existing, false // isFirst=false — reuse session
	}
	return nil, true // isFirst=true — caller sẽ tạo session mới
}

// Store lưu session vào pool sau khi reg thành công.
func (p *SessionPool) Store(proxyStr string, sess *session) {
	p.mu.Lock()
	defer p.mu.Unlock()

	key := proxyStr
	if key == "" {
		key = "__direct__"
	}
	p.sessions[key] = sess
}

// Remove xóa session khỏi pool (khi session lỗi/hết hạn).
// Đồng thời close idle TCP/TLS connection để giải phóng native buffer ngay —
// nếu không, idle conn còn đọng trong Transport pool đến khi GC dọn object.
// Lưu ý: chỉ close IDLE connection; request đang chạy KHÔNG bị abort.
func (p *SessionPool) Remove(proxyStr string) {
	p.mu.Lock()
	key := proxyStr
	if key == "" {
		key = "__direct__"
	}
	sess := p.sessions[key]
	delete(p.sessions, key)
	p.mu.Unlock()

	if sess != nil && sess.client != nil {
		sess.client.CloseIdleConnections()
	}
}

// CloseIdleConnsAll close idle TCP/TLS connection của TẤT CẢ session trong pool
// MÀ KHÔNG remove session khỏi map. Khác với CloseAll() (remove all sessions).
//
// Dùng cho periodic cleanup trong run dài (12h+): mỗi 10-15 phút gọi 1 lần để
// giải phóng idle conn buffer (native, ngoài Go heap → GC không động tới)
// trong khi vẫn giữ session object + cookie jar để reuse cho proxy đó.
//
// SAFE: chỉ close IDLE conn (không có request đang chạy); active request không
// bị abort. Lần dùng tiếp theo của session sẽ tự re-establish TCP nếu cần.
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

// CloseAll đóng toàn bộ session trong pool — gọi khi register run kết thúc
// để giải phóng native HTTP buffer / idle TCP / TLS state.
// Pool tiếp tục dùng được sau CloseAll (sessions map được reset rỗng).
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

// Size trả về số session đang giữ trong pool (cho debug/log).
func (p *SessionPool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.sessions)
}
