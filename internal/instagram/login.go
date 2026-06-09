// login.go — Facebook Cookie Mobile login
// Mapping từ WeBM LoginFacebookWithCookieMobile() + GetDataLogin() + ParseTokens()
// Fix: dùng CookieJar để capture cookies mới từ Facebook (giống WeBM HttpClientRequest)
package instagram

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/httpx"
)

// unescapeHTML xử lý các escape sequences trong HTML response từ Facebook.
//
// s là chuỗi HTML thô trả về từ Facebook — thường chứa các ký tự bị encode dạng JSON string
// do Facebook nhúng dữ liệu vào trong thẻ <script> hoặc trả về qua Bloks API.
//
// Thực hiện 3 bước unescape theo thứ tự:
//  1. \uXXXX → ký tự Unicode tương ứng (ví dụ: \u0022 → ")
//  2. \" → " (dấu ngoặc kép được JSON-escape)
//  3. \/ → / (dấu gạch chéo được JSON-escape, phổ biến trong URL)
//
// Mapping từ WeBM: Regex.Unescape(source).
func unescapeHTML(s string) string {
	// Unescape \uXXXX sequences
	re := regexp.MustCompile(`\\u([0-9a-fA-F]{4})`)
	s = re.ReplaceAllStringFunc(s, func(match string) string {
		var r rune
		fmt.Sscanf(match, `\u%04x`, &r)
		return string(r)
	})
	// Unescape \" → "
	s = strings.ReplaceAll(s, `\"`, `"`)
	// Unescape \/ → /
	s = strings.ReplaceAll(s, `\/`, `/`)
	return s
}

// ParseTokens extract các session token từ HTML response của Facebook và ghi vào Session.
//
// s là con trỏ Session sẽ được ghi trực tiếp — các field FbDtsg, Jazoest, Lsd, Hsi,
// SpinT, SpinR, Rev, S, Dyn được cập nhật tại chỗ sau khi parse xong.
//
// html là chuỗi HTML/JSON thô nhận từ response của Facebook (m.facebook.com hoặc www.facebook.com).
// Hàm gọi unescapeHTML trước để xử lý các ký tự escape trước khi áp dụng regex.
//
// fb_dtsg là CSRF token quan trọng nhất, cần cho mọi API call tiếp theo. Facebook
// thay đổi cách nhúng token qua nhiều phiên bản nên hàm thử 4 fallback patterns:
//  1. name="fb_dtsg" value="..." — form field truyền thống
//  2. DTSGInitialData",[],{"token":"..." — Relay store format
//  3. {"dtsg":{"token":"..." — GraphQL response format
//  4. initDtsg":"..." — Bloks API format
//
// jazoest và lsd cũng được extract với fallback tương tự.
// Nếu jazoest rỗng, dùng default "21966".
// Nếu lsd rỗng, dùng default "P2XQp3VtEtx1ajSpA8Wbgw".
// Nếu hsi, SpinT, SpinR rỗng, dùng các default value hardcode từ WeBM.
//
// Mapping từ WeBM WeSocial.cs ParseTokens() lines 1331-1420.
func ParseTokens(s *Session, html string) {
	// WeBM: try { source = Regex.Unescape(source); } catch { }
	html = unescapeHTML(html)
	// fb_dtsg — 4 fallback patterns
	s.FbDtsg = extractFirst(html,
		`name="fb_dtsg" value="([^"]+)"`,
		`DTSGInitialData",\[\],\{"token":"([^"]+)"`,
		`\{"dtsg":\{"token":"([^"]+)"`,
		`initDtsg":"([^"]+)"`,
	)

	// jazoest
	s.Jazoest = extractFirst(html,
		`name="jazoest" value="(\d+)"`,
		`jazoest=(\d+)`,
	)
	if s.Jazoest == "" {
		s.Jazoest = "21966"
	}

	// lsd
	s.Lsd = extractFirst(html,
		`LSD",\[\],\{"token":"([^"]+)"`,
	)
	if s.Lsd == "" {
		s.Lsd = "P2XQp3VtEtx1ajSpA8Wbgw"
	}

	// __hsi
	s.Hsi = extractFirst(html, `"hsi":"([^"]+)"`)
	if s.Hsi == "" {
		s.Hsi = "7052297439888351100-0"
	}

	// __spin_t, __spin_r
	s.SpinT = extractFirst(html, `"__spin_t":(\d+)`)
	if s.SpinT == "" {
		s.SpinT = "1723909047"
	}
	s.SpinR = extractFirst(html, `"__spin_r":(\d+)`)
	if s.SpinR == "" {
		s.SpinR = "1015763339"
	}

	// __rev — WeBM không parse __rev riêng, dùng __spin_r làm __rev
	s.Rev = s.SpinR

	// __s — session hash (extract từ HTML hoặc để rỗng)
	s.S = extractFirst(html,
		`"__s":"([^"]+)"`,
		`__s=([^&"]+)`,
	)
	// __s có thể rỗng — WeBM cũng cho phép rỗng

	// __dyn
	s.Dyn = extractFirst(html, `"__dyn":"([^"]+)"`)
	// __dyn thường rỗng trong body verify
}

// createClientWithCookieJar tạo http.Client kèm CookieJar — tương đương WeBM HttpClientRequest có CookieContainer.
//
// proxyStr là chuỗi proxy của account, định dạng ip:port hoặc ip:port:user:pass.
// Truyền chuỗi rỗng nếu không dùng proxy.
//
// initialCookies là chuỗi cookie ban đầu dạng "name1=value1; name2=value2" cần import vào jar
// trước khi bắt đầu request. Truyền chuỗi rỗng nếu không có cookie khởi tạo.
//
// targetURL là URL gốc để gắn cookies vào jar (thường là "https://m.facebook.com").
// Jar dùng URL này để xác định domain scope khi SetCookies.
//
// CookieJar tự động capture mọi Set-Cookie header từ response của Facebook và lưu lại
// để sử dụng cho các request tiếp theo trong cùng session — đây là cơ chế cốt lõi
// giúp session token (c_user, xs, fr, ...) được cập nhật sau mỗi redirect.
//
// DisableKeepAlives=true vì mỗi account dùng một client riêng biệt, không chia sẻ
// connection pool. Giữ keep-alive sẽ tốn resource mà không có lợi ích gì.
func createClientWithCookieJar(proxyStr string, initialCookies string, targetURL string) (*http.Client, *cookiejar.Jar) {
	jar, _ := cookiejar.New(nil)

	// Import cookies ban đầu vào jar
	if initialCookies != "" {
		u, _ := url.Parse(targetURL)
		if u != nil {
			var cookies []*http.Cookie
			for _, pair := range strings.Split(initialCookies, ";") {
				pair = strings.TrimSpace(pair)
				if pair == "" {
					continue
				}
				parts := strings.SplitN(pair, "=", 2)
				if len(parts) == 2 {
					cookies = append(cookies, &http.Cookie{
						Name:  strings.TrimSpace(parts[0]),
						Value: strings.TrimSpace(parts[1]),
					})
				}
			}
			jar.SetCookies(u, cookies)
		}
	}

	transport := &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		TLSHandshakeTimeout:   8 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		// Keep-alive cho phép Step1 và Step2 trong cùng login call tái dùng 1 kết nối TCP+TLS
		// thay vì tạo 2 kết nối riêng. Client này chỉ tồn tại trong 1 lần gọi LoginWithCookieMobile.
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: 2,
	}

	// Set proxy nếu có
	proxyStr = strings.TrimSpace(proxyStr)
	if proxyStr != "" {
		proxyURL := parseProxyURL(proxyStr)
		if proxyURL != nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	client := &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   20 * time.Second,
	}

	return client, jar
}

// parseProxyURL chuyển đổi chuỗi proxy thô thành *url.URL để dùng với http.Transport.
//
// proxy chấp nhận 2 định dạng:
//   - ip:port — proxy không xác thực (ví dụ: "1.2.3.4:8080")
//   - ip:port:user:pass — proxy có xác thực basic auth (ví dụ: "1.2.3.4:8080:user:pass")
//
// Scheme http:// hoặc https:// nếu có sẽ bị tách bỏ trước khi parse.
// Trả về nil nếu chuỗi proxy không hợp lệ.
//
// Hàm này là bản duplicate từ proxy package để tránh circular import —
// login package không được import proxy package vì proxy package có thể import lại facebook.
func parseProxyURL(proxy string) *url.URL {
	noScheme := strings.Replace(proxy, "https://", "", 1)
	noScheme = strings.Replace(noScheme, "http://", "", 1)
	parts := strings.Split(noScheme, ":")

	var host, port, user, pass string
	switch {
	case len(parts) >= 4:
		host, port, user, pass = parts[0], parts[1], parts[2], parts[3]
	case len(parts) >= 2:
		host, port = parts[0], parts[1]
	default:
		host, port = noScheme, "80"
	}

	proxyURLStr := fmt.Sprintf("http://%s:%s", host, port)
	if user != "" {
		proxyURLStr = fmt.Sprintf("http://%s:%s@%s:%s", user, pass, host, port)
	}
	u, _ := url.Parse(proxyURLStr)
	return u
}

// extractCookiesFromJar lấy toàn bộ cookies từ jar và trả về dạng chuỗi "name=value; ...".
//
// jar là CookieJar đang chứa cookies được Facebook set qua Set-Cookie header.
//
// Hàm kiểm tra 3 URL domain khác nhau vì Facebook set cookie với các domain scope khác nhau:
//   - https://m.facebook.com/ — cookies từ mobile site
//   - https://www.facebook.com/ — cookies từ desktop site
//   - https://.facebook.com/ — cookies shared cho toàn bộ subdomain
//
// Ba URL phải được check riêng vì CookieJar phân biệt domain theo RFC 6265 —
// cookie set cho ".facebook.com" không tự động xuất hiện khi query "m.facebook.com".
// Map seen dùng để dedup khi cùng tên cookie xuất hiện ở nhiều domain, ưu tiên giá trị cuối.
func extractCookiesFromJar(jar *cookiejar.Jar) string {
	urls := []*url.URL{
		{Scheme: "https", Host: "m.facebook.com", Path: "/"},
		{Scheme: "https", Host: "www.facebook.com", Path: "/"},
		{Scheme: "https", Host: ".facebook.com", Path: "/"},
	}

	seen := make(map[string]string)
	for _, u := range urls {
		for _, c := range jar.Cookies(u) {
			seen[c.Name] = c.Value
		}
	}

	var parts []string
	for name, value := range seen {
		parts = append(parts, name+"="+value)
	}
	return strings.Join(parts, "; ")
}

// LoginWithCookieMobile đăng nhập Facebook bằng cookie trên m.facebook.com và cập nhật Session.
//
// ctx dùng để kiểm soát timeout/cancellation cho toàn bộ quá trình đăng nhập.
// Caller nên truyền context có timeout hợp lý (ví dụ: 30-60 giây).
//
// s là con trỏ Session chứa thông tin đầu vào và sẽ được cập nhật khi thành công.
// Các field bắt buộc phải có trước khi gọi:
//   - Cookie: chuỗi cookie Facebook dạng "c_user=...; xs=...; ..." (bắt buộc, không được rỗng)
//   - Proxy: proxy của account, định dạng ip:port hoặc ip:port:user:pass (rỗng nếu không dùng)
//   - UserAgent: User-Agent string cho request (rỗng sẽ dùng DefaultUserAgent)
//
// Luồng thực hiện:
//  1. GET m.facebook.com/login/ không kèm cookie → parse session tokens (ParseTokens) và datr
//  2. Import cookie của account vào CookieJar
//  3. Vòng lặp tối đa 5 lần: GET m.facebook.com/login với cookie →
//     kiểm tra redirect URL (có "/login" → cookie chết), parse tokens,
//     fallback sang www.facebook.com/ nếu fb_dtsg rỗng,
//     fallback tiếp sang m.facebook.com/help nếu vẫn rỗng
//  4. Kiểm tra c_user trong cookie để xác nhận đăng nhập thành công
//
// Khi thành công, Session được cập nhật: Cookie (mới nhất từ jar), UID, Datr, FbDtsg, v.v.
// c_user trong cookie phải khớp với s.UID nếu UID đã có sẵn — để phát hiện cookie sai account.
//
// Mapping từ WeBM Login.WWW.cs LoginFacebookWithCookieMobile() + GetDataLogin() + GetSession().
func LoginWithCookieMobile(ctx context.Context, s *Session) (*LoginResult, error) {
	if s.Cookie == "" {
		return &LoginResult{Success: false, Message: "Không có cookie"}, nil
	}

	// WeBM dùng 1 HttpClientRequest xuyên suốt (CookieContainer chung)
	client, jar := createClientWithCookieJar(s.Proxy, "", "https://m.facebook.com")
	// Task 5: client + transport tạo per-call → close idle khi func return để
	// fhttp/net.Transport free TCP+TLS buffer ngay (trước đây để GC dọn sau N giây).
	// Chạy 100 worker × verify retry → tích lũy nhanh.
	defer client.CloseIdleConnections()

	// === Step 1: GetDataLogin — GET m.facebook.com/login/ KHÔNG cookie ===
	// WeBM: GetDataLogin(_request, f, "https://m.facebook.com")
	// → _request.GetContentResponse(GET, "https://m.facebook.com/login/", autoAddCookieHeader: false)
	req, err := http.NewRequestWithContext(ctx, "GET", "https://m.facebook.com/login/", nil)
	if err != nil {
		return &LoginResult{Success: false, Message: err.Error()}, err
	}
	setMobileHeaders(req, s.UserAgent, "")

	resp, err := client.Do(req)
	if err != nil {
		return &LoginResult{Success: false, Message: "Không kết nối được Facebook"}, err
	}
	body, _ := httpx.ReadBody(resp.Body, 512*1024)
	resp.Body.Close()
	html := string(body)

	// WeBM: GetSession(_request, f, responseStr) → ParseTokens
	ParseTokens(s, html)

	// WeBM: lấy datr từ CookieContainer
	extractDatrFromJar(s, jar)

	// === Step 2: Import cookies + validate (WeBM loop max 5 lần) ===
	// WeBM: import cookie pairs vào _request.AddCookie() (cùng CookieContainer)
	importCookiesToJar(jar, s.Cookie, "https://m.facebook.com")

	for attempt := 0; attempt < 2; attempt++ {
		// WeBM: _request.GetContent(Method.GET, "https://m.facebook.com/login")
		req2, err := http.NewRequestWithContext(ctx, "GET", "https://m.facebook.com/login", nil)
		if err != nil {
			return &LoginResult{Success: false, Message: err.Error()}, err
		}
		setMobileHeaders(req2, s.UserAgent, "")

		resp2, err := client.Do(req2)
		if err != nil {
			continue
		}
		body2, _ := httpx.ReadBody(resp2.Body, 512*1024)
		resp2.Body.Close()
		html2 := string(body2)

		// WeBM: string responseLink = _request.Url (final URL sau redirect)
		finalURL := resp2.Request.URL.String()

		// WeBM: if responseLink.Contains("/login") → cookie die
		if strings.Contains(finalURL, "/login") {
			return &LoginResult{Success: false, Message: "Cookie không hợp lệ, không thể đăng nhập!"}, nil
		}

		// WeBM: GetSession(_request, f, f.responseString) → ParseTokens
		// → nếu fb_dtsg rỗng → GET www.facebook.com/ → ParseTokens
		// → nếu vẫn rỗng → GET m.facebook.com/help → ParseTokens
		ParseTokens(s, html2)

		if s.FbDtsg == "" {
			// Fallback 1: GET www.facebook.com/
			desktopHTML := fetchURL(ctx, client, "https://www.facebook.com/", s.UserAgent)
			if desktopHTML != "" {
				ParseTokens(s, desktopHTML)
			}
		}
		if s.FbDtsg == "" {
			// Fallback 2: GET m.facebook.com/help
			helpHTML := fetchURL(ctx, client, "https://m.facebook.com/help", s.UserAgent)
			if helpHTML != "" {
				ParseTokens(s, helpHTML)
			}
		}

		// WeBM: f.Cookie = _request.GetAllCookiesAsStringForFacebook()
		updatedCookies := extractCookiesFromJar(jar)
		if updatedCookies != "" && strings.Contains(updatedCookies, "c_user") {
			s.Cookie = updatedCookies
		}

		// Extract datr từ jar
		extractDatrFromJar(s, jar)

		// WeBM: check c_user từ cookie
		cUser := extractCUser(s.Cookie)
		if cUser != "" {
			s.UID = strings.TrimSpace(s.UID)
			if s.UID == "" || cUser == s.UID {
				if s.UID == "" {
					s.UID = cUser
				}
				return &LoginResult{
					Success: true,
					Message: "Đăng nhập bằng cookie thành công",
					Session: s,
				}, nil
			}
		}
	}

	return &LoginResult{Success: false, Message: "Đăng nhập bằng cookie thất bại"}, nil
}

// extractDatrFromJar lấy giá trị cookie datr từ jar và ghi vào s.Datr.
//
// s là Session sẽ được cập nhật field Datr khi tìm thấy.
//
// jar là CookieJar hiện tại của session.
//
// datr là "device authentication token" — Facebook dùng để định danh thiết bị/browser.
// Cookie này được set bởi m.facebook.com khi load trang lần đầu (trước khi có cookie account).
// datr cần được gửi kèm trong các API request tiếp theo để Facebook nhận diện đây là
// request từ một trình duyệt hợp lệ, tránh bị block do thiếu device fingerprint.
// Chỉ lấy từ m.facebook.com vì đây là domain được dùng trong toàn bộ mobile login flow.
func extractDatrFromJar(s *Session, jar *cookiejar.Jar) {
	for _, c := range jar.Cookies(&url.URL{Scheme: "https", Host: "m.facebook.com", Path: "/"}) {
		if strings.EqualFold(c.Name, "datr") {
			s.Datr = c.Value
			return
		}
	}
}

// importCookiesToJar parse chuỗi cookie và import vào CookieJar cho domain của targetURL.
//
// jar là CookieJar sẽ nhận các cookie được import vào.
//
// cookieStr là chuỗi cookie dạng "name1=value1; name2=value2; ..." — thường là cookie
// lấy từ database/file của account. Chuỗi rỗng hoặc pair thiếu dấu "=" sẽ bị bỏ qua.
//
// targetURL xác định domain scope khi SetCookies (ví dụ: "https://m.facebook.com").
// Tất cả cookie được import sẽ gắn với domain này và sẽ được gửi kèm khi request tới cùng domain.
//
// Tương đương WeBM _request.AddCookie() — WeBM duyệt từng cookie pair và thêm vào CookieContainer.
func importCookiesToJar(jar *cookiejar.Jar, cookieStr, targetURL string) {
	u, _ := url.Parse(targetURL)
	if u == nil {
		return
	}
	var cookies []*http.Cookie
	for _, pair := range strings.Split(cookieStr, ";") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			cookies = append(cookies, &http.Cookie{
				Name:  strings.TrimSpace(parts[0]),
				Value: strings.TrimSpace(parts[1]),
			})
		}
	}
	jar.SetCookies(u, cookies)
}

// fetchURL thực hiện GET request đến targetURL và trả về body dạng string.
//
// ctx dùng để kiểm soát timeout/cancellation, được truyền thẳng từ caller (LoginWithCookieMobile).
//
// client là http.Client đang được dùng trong session hiện tại — quan trọng vì client
// mang theo CookieJar, giúp cookies được gửi kèm và Set-Cookie từ response được capture tự động.
//
// targetURL là URL cần GET, thường là "https://www.facebook.com/" hoặc "https://m.facebook.com/help"
// dùng làm fallback khi parse tokens từ trang chính thất bại.
//
// userAgent được set vào header User-Agent — cần nhất quán với các request trước để tránh
// Facebook detect session bất thường do UA thay đổi giữa các request.
//
// Trả về chuỗi rỗng nếu request lỗi — caller tự kiểm tra trước khi dùng kết quả.
func fetchURL(ctx context.Context, client *http.Client, targetURL, userAgent string) string {
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return ""
	}
	setMobileHeaders(req, userAgent, "")
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 512*1024)
	return string(body)
}

// setMobileHeaders gắn các HTTP header cần thiết để giả lập browser mobile truy cập Facebook.
//
// req là *http.Request sẽ được set header trực tiếp — hàm không tạo request mới.
//
// userAgent là chuỗi User-Agent của mobile browser (ví dụ: Chrome trên iOS).
// Nếu rỗng, dùng DefaultUserAgent(). Cần nhất quán xuyên suốt session.
//
// cookie là chuỗi cookie cần gắn vào header Cookie. Truyền rỗng nếu muốn để
// CookieJar tự xử lý (trường hợp request qua client.Do thay vì gắn tay).
//
// Dùng mobile headers vì LoginWithCookieMobile target m.facebook.com — Facebook
// phân biệt mobile/desktop traffic qua User-Agent và các Sec-Fetch-* headers.
// Request thiếu mobile signature có thể bị redirect về desktop hoặc bị reject.
func setMobileHeaders(req *http.Request, userAgent, cookie string) {
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	} else {
		req.Header.Set("User-Agent", DefaultUserAgent())
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
}

// extractFirst thử lần lượt từng regex pattern và trả về nội dung capture group đầu tiên tìm thấy.
//
// html là chuỗi HTML hoặc JSON nguồn cần tìm kiếm — thường là response body từ Facebook
// sau khi đã qua unescapeHTML.
//
// patterns là danh sách các regex pattern theo thứ tự ưu tiên. Mỗi pattern phải chứa
// ít nhất 1 capture group (dấu ngoặc đơn) — hàm lấy match[1] (group đầu tiên).
// Patterns được thử tuần tự, dừng lại và trả về ngay khi có match khác rỗng.
// Nếu không có pattern nào match, trả về chuỗi rỗng "".
//
// Pattern regex lỗi (compile thất bại) sẽ bị bỏ qua và thử pattern tiếp theo.
//
// Dùng cho ParseTokens để extract fb_dtsg, jazoest, lsd, hsi, spin_t, spin_r
// từ nhiều định dạng response khác nhau (HTML form, Relay store, GraphQL, Bloks API).
func extractFirst(html string, patterns ...string) string {
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		match := re.FindStringSubmatch(html)
		if len(match) >= 2 && match[1] != "" {
			return match[1]
		}
	}
	return ""
}

// extractCUser lấy giá trị cookie c_user từ chuỗi cookie của session.
//
// cookie là chuỗi cookie dạng "name1=value1; name2=value2; c_user=123456789".
//
// c_user là cookie chứa Facebook User ID (UID) của tài khoản đang đăng nhập.
// Sự có mặt của c_user trong cookie string là dấu hiệu chắc chắn nhất cho thấy
// session đang active và đăng nhập thành công — Facebook chỉ set c_user sau khi
// xác thực cookie hợp lệ. Cookie die hoặc bị revoke sẽ không có c_user.
//
// Trả về chuỗi rỗng "" nếu c_user không tồn tại trong cookie string.
func extractCUser(cookie string) string {
	for _, pair := range strings.Split(cookie, ";") {
		pair = strings.TrimSpace(pair)
		if strings.HasPrefix(pair, "c_user=") {
			return strings.TrimPrefix(pair, "c_user=")
		}
	}
	return ""
}

// DefaultUserAgent UA mặc định cho mobile Facebook
func DefaultUserAgent() string {
	return "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/134.0.1911.158 Mobile/15E148 Safari/604.1"
}
