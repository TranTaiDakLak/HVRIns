// Package proxy — Tạo HTTP client với proxy
// Mapping từ WeBM CreateProxyClient()
package proxy

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	// reSessionProxy: username đã có sid-XXXX-t-NN → thay XXXX bằng session mới
	// Ví dụ: y3b758257-region-US-sid-ebpjpuhepz-t-15
	reSessionProxy = regexp.MustCompile(`(?i)(.*-sid-)([a-z0-9]+)(-t-\d+)$`)

	// reRegionProxy: username có -region-XXX nhưng chưa có sid → thêm -sid-XXXX-t-15
	// Ví dụ: y3b758257-region-US, wkiPYKNH2YRh-region-Random
	reRegionProxy = regexp.MustCompile(`(?i)-region-\w+$`)

	// reZoneProxy: username có -zone-XXX (711proxy, v.v.) → thêm -session-XXXX
	// Ví dụ: USER255485-zone-custom
	reZoneProxy = regexp.MustCompile(`(?i)-zone-\w+$`)

	// reSSIDProxy: NiceProxy format — username có -ssid-XXXXXXXXXX → thay ssid mới
	// Ví dụ: soialvin_Y4uV-ssid-n1nlmUkcxa
	reSSIDProxy = regexp.MustCompile(`(?i)(.*-ssid-)([a-zA-Z0-9]+)$`)

	// reProxyShareSession: ProxyShare format — username có _session-XXXXXXXXXX (+ optional _life-N)
	// Ví dụ: ps-k3b0n3zt2nnu_area-US_session-2HNUFU64BE_life-5
	// Replace session ID mới → proxyshare trả về IP khác
	reProxyShareSession = regexp.MustCompile(`(?i)(_session-)([A-Za-z0-9]+)`)

	// Domain-based session patterns — dùng để strip session cũ trước khi inject mới
	reLunaSession  = regexp.MustCompile(`(?i)-sessid-[a-z0-9]+-t-\d+$`)           // lunaproxy
	reAbcSession   = regexp.MustCompile(`(?i)-session-[a-z0-9]+-sessTime-\d+$`)   // abcproxy/lightningproxies/pyproxy
	reSmartSession = regexp.MustCompile(`(?i)-sessionduration-\d+$`)               // smartproxy
)

// RenderSessionIfIsProxyServer — Mapping từ C# RenderSessionIfIsProxyServer()
// Ba trường hợp:
//  1. Username đã có sid-XXXX-t-NN → thay session ID mới
//  2. Username có -region-XXX (bất kỳ) → thêm -sid-XXXX-t-15
//  3. Username có -zone-XXX (711proxy) → thêm -session-XXXX
//
// Mỗi lần gọi → session ID mới → IP khác nhau (rotating proxy)
func RenderSessionIfIsProxyServer(proxyStr string) string {
	if proxyStr == "" {
		return proxyStr
	}

	// Strip scheme (http/https/socks*) để làm việc với format thuần
	noScheme := proxyStr
	scheme := ""
	for _, s := range []string{"http://", "https://", "socks5://", "socks5h://", "socks4://", "socks4a://", "socks://"} {
		if strings.HasPrefix(strings.ToLower(noScheme), s) {
			scheme = noScheme[:len(s)]
			noScheme = noScheme[len(s):]
			break
		}
	}

	// Format user:pass@host:port — rotate session trong credPart rồi ghép lại
	if atIdx := strings.LastIndex(noScheme, "@"); atIdx > 0 {
		credPart := noScheme[:atIdx]
		hostPart := noScheme[atIdx+1:]
		host := strings.SplitN(hostPart, ":", 2)[0]
		credParts := strings.SplitN(credPart, ":", 2)
		if len(credParts) >= 1 {
			user := credParts[0]
			// Ưu tiên domain-based (lunaproxy, abcproxy, v.v.), fallback username regex
			newUser, matched := rotateSessionForHost(user, host)
			if !matched {
				newUser = rotateSessionInUser(user)
			}
			if newUser == user {
				return proxyStr // không match pattern → static proxy
			}
			if len(credParts) == 2 {
				return scheme + newUser + ":" + credParts[1] + "@" + hostPart
			}
			return scheme + newUser + "@" + hostPart
		}
	}

	parts := strings.SplitN(noScheme, ":", 4)
	if len(parts) < 4 {
		return proxyStr
	}

	// Detect format: user:pass:host:port vs host:port:user:pass
	// Nếu parts[3] là port (số) và parts[1] không phải số → user:pass:host:port
	userIdx := 2 // default: host:port:user:pass
	hostIdx := 0
	if isPort(parts[3]) && !isPort(parts[1]) {
		userIdx = 0 // user:pass:host:port (NiceProxy)
		hostIdx = 2
	}

	host := parts[hostIdx]
	user := parts[userIdx]
	// Ưu tiên domain-based (lunaproxy, abcproxy, v.v.), fallback username regex
	newUser, matched := rotateSessionForHost(user, host)
	if !matched {
		newUser = rotateSessionInUser(user)
	}
	if newUser == user {
		return proxyStr // proxy tĩnh (không match pattern)
	}
	parts[userIdx] = newUser

	return strings.Join(parts, ":")
}

// EnsureStickySession đảm bảo proxy có sticky session.
// Nếu username đã có session params → rotate (giống RenderSessionIfIsProxyServer).
// Nếu là username thường (không có params) → tự động inject -session-RANDOM vào username,
// giúp provider như IPRocket pin về 1 IP mà không cần user phải nhập thủ công.
func EnsureStickySession(proxyStr string) string {
	if proxyStr == "" {
		return proxyStr
	}
	// Thử rotate — nếu có session params sẽ trả về string mới khác proxyStr
	rendered := RenderSessionIfIsProxyServer(proxyStr)
	if rendered != proxyStr {
		return rendered
	}
	// Plain proxy → inject -session-RANDOM
	return injectSession(proxyStr)
}

// injectSession thêm -session-RANDOM vào phần username của proxy string.
func injectSession(proxyStr string) string {
	noScheme := proxyStr
	scheme := ""
	for _, s := range []string{"http://", "https://", "socks5://", "socks5h://", "socks4://", "socks4a://", "socks://"} {
		if strings.HasPrefix(strings.ToLower(noScheme), s) {
			scheme = noScheme[:len(s)]
			noScheme = noScheme[len(s):]
			break
		}
	}

	suffix := "-session-" + randomSessionID(10)

	// Format user:pass@host:port
	if atIdx := strings.LastIndex(noScheme, "@"); atIdx > 0 {
		credPart := noScheme[:atIdx]
		hostPart := noScheme[atIdx+1:]
		credParts := strings.SplitN(credPart, ":", 2)
		if len(credParts) == 2 {
			return scheme + credParts[0] + suffix + ":" + credParts[1] + "@" + hostPart
		}
		return scheme + credPart + suffix + "@" + hostPart
	}

	// Format host:port:user:pass hoặc user:pass:host:port
	parts := strings.SplitN(noScheme, ":", 4)
	if len(parts) < 4 {
		return proxyStr
	}
	userIdx := 2
	if isPort(parts[3]) && !isPort(parts[1]) {
		userIdx = 0
	}
	parts[userIdx] += suffix
	return scheme + strings.Join(parts, ":")
}

// rotateSessionForHost rotate session dựa trên domain proxy host — ưu tiên hơn username regex.
// Mapping từ C# RenderSessionIfIsProxyServer() domain checks.
// Trả về (newUser, true) nếu host khớp provider, (user, false) nếu không.
func rotateSessionForHost(user, host string) (string, bool) {
	newSess := randomSessionID(10)
	const sesstime = "15"
	h := strings.ToLower(host)
	switch {
	case strings.Contains(h, "lunaproxy"):
		base := reLunaSession.ReplaceAllString(user, "")
		return base + "-sessid-" + newSess + "-t-" + sesstime, true
	case strings.Contains(h, "abcproxy"),
		strings.Contains(h, "lightningproxies"),
		strings.Contains(h, "pyproxy"):
		base := reAbcSession.ReplaceAllString(user, "")
		return base + "-session-" + newSess + "-sessTime-" + sesstime, true
	case strings.Contains(h, "arxlabs"):
		if reSessionProxy.MatchString(user) {
			return reSessionProxy.ReplaceAllString(user, "${1}"+newSess+"${3}"), true
		}
		return user + "-sid-" + newSess + "-t-" + sesstime, true
	case strings.Contains(h, "smartproxy"):
		base := reSmartSession.ReplaceAllString(user, "")
		return base + "-sessionduration-" + sesstime, true
	}
	return "", false
}

// rotateSessionInUser rotate session ID trong username theo pattern provider.
// Trả về username mới nếu match pattern, nếu không trả về nguyên user (proxy tĩnh).
func rotateSessionInUser(user string) string {
	newSession := randomSessionID(10)
	switch {
	case reProxyShareSession.MatchString(user):
		// ProxyShare: ps-xxx_area-US_session-XXXX_life-5 → replace session value
		return reProxyShareSession.ReplaceAllString(user, "${1}"+strings.ToUpper(newSession))
	case reSSIDProxy.MatchString(user):
		// NiceProxy: soialvin_Y4uV-ssid-n1nlmUkcxa → replace ssid
		return reSSIDProxy.ReplaceAllString(user, "${1}"+newSession)
	case reSessionProxy.MatchString(user):
		// đã có sid-XXXX-t-NN → thay session ID
		return reSessionProxy.ReplaceAllString(user, "${1}"+newSession+"${3}")
	case reRegionProxy.MatchString(user):
		// có -region-XXX → thêm -sid-XXXX-t-15
		return user + "-sid-" + newSession + "-t-15"
	case reZoneProxy.MatchString(user):
		// có -zone-XXX (711proxy) → thêm -session-XXXX
		return user + "-session-" + newSession
	}
	return user
}

const sessionChars = "abcdefghijklmnopqrstuvwxyz0123456789"

func randomSessionID(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	b := make([]byte, n)
	for i := range b {
		b[i] = sessionChars[r.Intn(len(sessionChars))]
	}
	return string(b)
}

// ProxyAPIClient — shared client dùng cho tất cả proxy API calls (minproxy, netproxy, proxyfarm, shoplike, tinsoft)
// Tránh tạo Transport mới mỗi lần gọi API — tiết kiệm memory + TLS handshake overhead khi 100+ luồng
var ProxyAPIClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	},
}

// CreateClient tạo *http.Client với proxy từ string format:
//   - "" (no proxy)
//   - "ip:port"
//   - "ip:port:user:pass"
//   - "http://user:pass@ip:port"
//
// Mapping từ WeBM WemakeFacebook.Func.12.Verify.cs CreateProxyClient() lines 17-83
//
// **Transport reuse**: Transport được cache per-proxy-string qua transport_pool.go
// (cap 500, LRU evict, idle TTL 5 phút). Giảm RAM leak khi chạy 24/7 — thay vì mỗi
// CreateClient tạo Transport mới (mỗi cái ~200KB buffer + TLS cache), pool reuse tối đa.
//
// Rollback: set env `TRANSPORT_POOL_DISABLED=1` → fallback tạo Transport mỗi call.
func CreateClient(proxyStr string, timeout time.Duration) *http.Client {
	proxyStr = strings.TrimSpace(proxyStr)
	transport := getOrCreateTransport(proxyStr)

	if timeout == 0 {
		timeout = 60 * time.Second
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		// Không tự động follow redirect cho một số case
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

// FormatProxyURL chuyển proxy string sang full URL format cho bogdanfinn/tls-client.
// Trả về "" nếu proxyStr rỗng hoặc không parse được.
func FormatProxyURL(proxyStr string) string {
	if proxyStr == "" {
		return ""
	}
	u := parseProxy(proxyStr)
	if u == nil {
		return ""
	}
	return u.String()
}

// parseProxy parse proxy string thành *url.URL
// Hỗ trợ các format:
//   - ip:port                               (không auth)
//   - ip:port:user                          (user-only)
//   - ip:port:user:pass                     (default)
//   - user:pass:ip:port                     (NiceProxy)
//   - user:pass@ip:port                     (không scheme — nhiều nguồn dùng)
//   - http://user:pass@ip:port              (URL full)
//   - https://user:pass@ip:port
//   - socks5://user:pass@ip:port            (SOCKS5)
//   - socks5h://user:pass@ip:port           (SOCKS5 + DNS remote)
//   - socks4://ip:port                      (SOCKS4)
func parseProxy(proxy string) *url.URL {
	proxy = strings.TrimSpace(proxy)
	if proxy == "" {
		return nil
	}

	// Nếu đã có scheme (http/https/socks*), parse trực tiếp qua url.Parse
	lower := strings.ToLower(proxy)
	for _, scheme := range []string{"http://", "https://", "socks5://", "socks5h://", "socks4://", "socks4a://", "socks://"} {
		if strings.HasPrefix(lower, scheme) {
			u, err := url.Parse(proxy)
			if err != nil {
				return nil
			}
			return u
		}
	}

	// user:pass@host:port (không scheme) — detect bằng dấu "@"
	// Note: một số provider dùng dấu "@" trong username → kiểm tra phần sau "@" có host:port
	if atIdx := strings.LastIndex(proxy, "@"); atIdx > 0 {
		credPart := proxy[:atIdx]
		hostPart := proxy[atIdx+1:]
		hostParts := strings.SplitN(hostPart, ":", 2)
		if len(hostParts) == 2 && isPort(hostParts[1]) {
			credParts := strings.SplitN(credPart, ":", 2)
			u := &url.URL{
				Scheme: "http",
				Host:   fmt.Sprintf("%s:%s", hostParts[0], hostParts[1]),
			}
			if len(credParts) == 2 {
				u.User = url.UserPassword(credParts[0], credParts[1])
			} else {
				u.User = url.User(credParts[0])
			}
			return u
		}
	}

	// ip:port:user:pass — tách bằng cách giới hạn 4 phần từ trái
	// Không dùng Split vì password có thể chứa dấu ":"
	parts := strings.SplitN(proxy, ":", 4)

	var host, port, user, pass string

	switch len(parts) {
	case 4:
		// Detect format: kiểm tra parts[1] và parts[3] xem cái nào là port (số)
		// user:pass:host:port — parts[3] là số → format này
		// host:port:user:pass — parts[1] là số → format này
		if isPort(parts[3]) && !isPort(parts[1]) {
			// user:pass:host:port
			user = parts[0]
			pass = parts[1]
			host = parts[2]
			port = parts[3]
		} else {
			// host:port:user:pass (default)
			host = parts[0]
			port = parts[1]
			user = parts[2]
			pass = parts[3]
		}
	case 3:
		// ip:port:user (một số provider dùng username-only, không có pass)
		host = parts[0]
		port = parts[1]
		user = parts[2]
	case 2:
		// ip:port (không cần auth)
		host = parts[0]
		port = parts[1]
	default:
		host = proxy
		port = "80"
	}

	u := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", host, port),
	}
	if user != "" {
		u.User = url.UserPassword(user, pass)
	}
	return u
}

// ShortDisplay trả về "host:port" để hiển thị UI an toàn khi CheckIP fail.
// Tránh lộ credentials (user:pass hoặc session token dài của proxyshare) ở cột IP CHẠY.
//
// Input formats:
//   - "host:port"                                → "host:port"
//   - "host:port:user:pass"                      → "host:port"
//   - "user:pass@host:port" / "http://user:pass@host:port" → "host:port"
//   - proxyshare session: "host:port:ps-...long-session-token...:pass" → "host:port"
func ShortDisplay(proxyStr string) string {
	s := strings.TrimSpace(proxyStr)
	if s == "" {
		return ""
	}
	// URL format: http://user:pass@host:port
	if u := parseProxy(s); u != nil && u.Host != "" {
		return u.Host
	}
	// Legacy format: host:port:user:pass hoặc host:port
	parts := strings.Split(s, ":")
	if len(parts) >= 2 && isPort(parts[1]) {
		return parts[0] + ":" + parts[1]
	}
	return s
}

// isPort kiểm tra chuỗi có phải port hợp lệ (số 1-65535)
func isPort(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	if len(s) == 0 || len(s) > 5 {
		return false
	}
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n >= 1 && n <= 65535
}
