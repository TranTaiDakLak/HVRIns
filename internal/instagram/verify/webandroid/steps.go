// steps.go — Web Android verify HTTP layer
// Mapping từ C#: FacebookVerifyWebAndroidAPI
// Flow: GET changeemail → POST setemail → POST confirmation_cliff
package webandroid

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/proxy"
)

const (
	changeEmailURL = "https://m.facebook.com/changeemail"
	setEmailURL    = "https://m.facebook.com/setemail"

	// x-referer cookie values — exact từ C# FacebookVerifyWebAndroidAPI
	xRefererChangeEmail = "eyJyIjoiL2NvbmZpcm1lbWFpbC5waHA%2FbmV4dD1odHRwcyUzQSUyRiUyRm0uZmFjZWJvb2suY29tJTJGJTNGZGVvaWElM0QxJnNvZnQ9aGprIiwiaCI6Ii9jb25maXJtZW1haWwucGhwP25leHQ9aHR0cHMlM0ElMkYlMkZtLmZhY2Vib29rLmNvbSUyRiUzRmRlb2lhJTNEMSZzb2Z0PWhqayIsInMiOiJtIn0%3D"
	xRefererConfirmCode = "eyJyIjoiL2NvbmZpcm1lbWFpbC5waHA%2FZW1haWxfY2hhbmdlZCZzb2Z0PWhqayIsImgiOiIvY29uZmlybWVtYWlsLnBocD9lbWFpbF9jaGFuZ2VkJnNvZnQ9aGprIiwicyI6Im0ifQ%3D%3D"

	// Default Chrome Android UA khi session.UserAgent không phải Chrome mobile.
	// CHANGED 2026-05: Chrome 134 → 141 (stable hiện tại). UA cũ → FB nghi ngờ vì
	// Chrome auto-update trong < 6 tuần, version > 6 tháng = unrealistic.
	defaultChromeAndroidUA = "Mozilla/5.0 (Linux; Android 15; SM-S931B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.7390.107 Mobile Safari/537.36"
)

// chromeUA trả về UA phù hợp cho Chrome Android request.
// Webandroid verify gọi m.facebook.com qua flow Chrome Mobile — UA BẮT BUỘC
// phải là Chrome Mobile, ngược lại sec-ch-ua-mobile và navigator fingerprint
// sẽ mismatch → FB phát hiện.
//
// Logic:
//   - Chrome Mobile (có "Chrome" + "Mobile") → dùng nguyên
//   - Otherwise → generate random Chrome Android UA mới
//
// CHANGED 2026-05: thay vì fallback về hardcoded defaultChromeAndroidUA (mọi VER
// đều cùng 1 UA → FB detect ngay), giờ random Chrome Android UA mỗi lần.
// Caller (Verify entry) đã normalize session.UserAgent → chromeUA() chỉ là safety net.
func chromeUA(sessionUA string) string {
	if strings.Contains(sessionUA, "Chrome") && strings.Contains(sessionUA, "Mobile") {
		return sessionUA
	}
	// Safety fallback: generate random Chrome Android UA thay vì hardcoded.
	// Bình thường KHÔNG đi qua đây vì verify.go:Verify() đã set session.UserAgent
	// = RandomUA() từ đầu. Đây chỉ là safety cho case gọi trực tiếp addEmail/confirmEmail.
	return RandomUA("")
}

// parseChromeInfo extract Chrome version và Android device info từ UA string.
// Trả về (chromeVerMajor, chromeVerFull, androidVer, deviceModel).
// Dùng để build sec-ch-ua headers.
func parseChromeInfo(ua string) (major, full, androidVer, deviceModel string) {
	// Chrome/134.0.6998.135
	if m := regexp.MustCompile(`Chrome/([\d.]+)`).FindStringSubmatch(ua); len(m) > 1 {
		full = m[1]
		if parts := strings.Split(full, "."); len(parts) > 0 {
			major = parts[0]
		}
	}
	if major == "" {
		major = "134"
		full = "134.0.6998.135"
	}
	// Android 14; Pixel 8
	if m := regexp.MustCompile(`Android (\d+); ([^)]+)`).FindStringSubmatch(ua); len(m) > 2 {
		androidVer = m[1]
		deviceModel = strings.TrimSpace(m[2])
	}
	if androidVer == "" {
		androidVer = "14"
		deviceModel = "Pixel 8"
	}
	return
}

// webState chứa tokens extract được từ trang changeemail — dùng cho cả addEmail và confirmEmail.
type webState struct {
	oldEmail    string
	regInstance string
	dtsg        string
	jazoest     string
	lsd         string
	encA        string // encrypted param (__a)
	confirmDtsg string // từ MPageLoadClientMetrics.init — dùng cho confirmEmail nếu có
	confirmJazo string
}

// createClient tạo standard net/http client qua proxy.
// Timeout 15s: proxy chậm (iprocket ~8s) vẫn đủ; proxy tốt (proxyshare ~2s) không bị chờ lãng phí.
func createClient(proxyStr string) *http.Client {
	return proxy.CreateClient(proxyStr, 15*time.Second)
}

// doGet thực hiện GET request, trả về (body, finalURL, error).
// finalURL là URL sau khi redirect — dùng để detect checkpoint.
// Body prepend sentinel nếu response header có `X-Fb-Integrity-Required: checkpoint`
// hoặc `X-Fb-Integrity-Requires-Login` — match C# FacebookCheckpointDetectorUtils.
func doGet(ctx context.Context, client *http.Client, targetURL string, headers http.Header) (body, finalURL string, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", "", err
	}
	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 3<<20))
	body = prependIntegritySentinel(resp.Header, string(b))
	return body, resp.Request.URL.String(), nil
}

// doPost thực hiện POST form-urlencoded request.
// Body prepend sentinel nếu response header chứa integrity markers.
func doPost(ctx context.Context, client *http.Client, targetURL, formBody string, headers http.Header) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, strings.NewReader(formBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 3<<20))
	return prependIntegritySentinel(resp.Header, string(b)), nil
}

// prependIntegritySentinel — port C# FacebookCheckpointDetectorUtils.IsCheckpointRequired.
// FB trả body rỗng/redirect + header X-Fb-Integrity-Required: checkpoint khi checkpoint.
// Prepend sentinel để isCheckpointResponse strings.Contains() bắt được.
func prependIntegritySentinel(h http.Header, body string) string {
	integrity := h.Get("X-Fb-Integrity-Required")
	if integrity != "" && strings.Contains(strings.ToLower(integrity), "checkpoint") {
		return `{"error":{"code":459,"message":"checkpointed"}}` + body
	}
	if h.Get("X-Fb-Integrity-Requires-Login") != "" {
		return `{"error":{"message":"checkpointed"}}` + body
	}
	if h.Get("X-Fb-Integrity-Enrollment") != "" {
		return `{"error":{"message":"checkpointed"}}` + body
	}
	return body
}

// navHeaders — Chrome Android navigation headers (dùng cho GET changeemail).
// Mapping từ C#: PerpectChromeAndroidNavHeadersFormat2
func navHeaders(ua string) http.Header {
	major, full, androidVer, deviceModel := parseChromeInfo(ua)
	h := make(http.Header)
	h.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	h.Set("upgrade-insecure-requests", "1")
	h.Set("sec-fetch-site", "none")
	h.Set("sec-fetch-mode", "navigate")
	h.Set("sec-fetch-user", "?1")
	h.Set("sec-fetch-dest", "document")
	h.Set("dpr", "2")
	h.Set("viewport-width", "393")
	h.Set("sec-ch-ua", fmt.Sprintf(`"Chromium";v="%s", "Google Chrome";v="%s", "Not-A.Brand";v="99"`, major, major))
	h.Set("sec-ch-ua-mobile", "?1")
	h.Set("sec-ch-ua-platform", `"Android"`)
	h.Set("sec-ch-ua-platform-version", fmt.Sprintf(`"%s"`, androidVer))
	h.Set("sec-ch-ua-model", fmt.Sprintf(`"%s"`, deviceModel))
	h.Set("sec-ch-ua-full-version-list", fmt.Sprintf(`"Chromium";v="%s", "Google Chrome";v="%s", "Not-A.Brand";v="99.0.0.0"`, full, full))
	h.Set("sec-ch-prefers-color-scheme", "light")
	h.Set("accept-language", "en-US,en;q=0.9,vi-VN;q=0.8,vi;q=0.7")
	h.Set("priority", "u=0, i")
	h.Set("user-agent", ua)
	return h
}

// postHeaders — Chrome Android POST headers (dùng cho POST setemail và confirmation_cliff).
// Mapping từ C#: PerpectChromeAndroidPostHeadersFormat2
func postHeaders(ua, referer, xRefererCookie string) http.Header {
	major, full, androidVer, deviceModel := parseChromeInfo(ua)
	h := make(http.Header)
	h.Set("accept", "*/*")
	h.Set("sec-fetch-site", "same-origin")
	h.Set("sec-fetch-mode", "cors")
	h.Set("sec-fetch-dest", "empty")
	h.Set("referer", referer)
	h.Set("origin", "https://m.facebook.com")
	h.Set("sec-ch-ua", fmt.Sprintf(`"Chromium";v="%s", "Google Chrome";v="%s", "Not-A.Brand";v="99"`, major, major))
	h.Set("sec-ch-ua-mobile", "?1")
	h.Set("sec-ch-ua-platform", `"Android"`)
	h.Set("sec-ch-ua-platform-version", fmt.Sprintf(`"%s"`, androidVer))
	h.Set("sec-ch-ua-model", fmt.Sprintf(`"%s"`, deviceModel))
	h.Set("sec-ch-ua-full-version-list", fmt.Sprintf(`"Chromium";v="%s", "Google Chrome";v="%s", "Not-A.Brand";v="99.0.0.0"`, full, full))
	h.Set("sec-ch-prefers-color-scheme", "light")
	h.Set("accept-language", "en-US,en;q=0.9,vi-VN;q=0.8,vi;q=0.7")
	h.Set("priority", "u=1, i")
	h.Set("X-Response-Format", "JSONStream")
	h.Set("X-Requested-With", "XMLHttpRequest")
	h.Set("user-agent", ua)
	// x-referer cookie — append vào Cookie header
	h.Set("Cookie", "") // placeholder, set lúc addEmail/confirmEmail call
	_ = xRefererCookie   // set bởi caller vào Cookie
	return h
}

// addEmail thực hiện GET changeemail → extract tokens → POST setemail.
// Trả về *webState chứa các token cần cho bước confirmEmail.
func addEmail(ctx context.Context, proxyStr, cookie, uid, ua, newEmail string) (*webState, error) {
	return addEmailWithNotify(ctx, proxyStr, cookie, uid, ua, newEmail, nil)
}

// addEmailWithNotify như addEmail nhưng nhận callback để log từng bước (GET/POST).
// Dùng tls-client (Chrome JA3) — net/http (Go default TLS) bị FB anti-bot RST connection
// ngay ở TLS handshake (lỗi EOF / forcibly closed). Bắt buộc dùng tls-client.
func addEmailWithNotify(ctx context.Context, proxyStr, cookie, uid, ua, newEmail string, notify func(string)) (*webState, error) {
	logf := func(format string, args ...interface{}) {
		if notify != nil {
			notify(fmt.Sprintf(format, args...))
		}
	}
	ua = chromeUA(ua)
	chromeMajor := extractChromeMajor(ua)

	client, err := createTLSClient(proxyStr, chromeMajor)
	if err != nil {
		return nil, fmt.Errorf("create TLS client: %w", err)
	}
	defer client.CloseIdleConnections()

	// Seed account cookies vào jar (cần thiết cho m.facebook.com auth state)
	seedCookieStringTLS(client, cookie)

	// === Step 1: GET changeemail ===
	logf("[WebAndroid] GET /settings/email/change...")
	getH := navHeaders(ua)
	// KHÔNG set Cookie header — tls-client tự gửi từ jar (đã seed ở trên)

	// Retry 3 lần khi proxy drop / FB RST.
	var body, finalURL string
	for attempt := 1; attempt <= 3; attempt++ {
		body, finalURL, err = doGetTLS(ctx, client, changeEmailURL, getH, nil)
		if err == nil {
			break
		}
		if !isTransientTLSErr(err) || attempt == 3 {
			return nil, fmt.Errorf("GET changeemail: %w", err)
		}
		client.CloseIdleConnections()
		client, err = createTLSClient(proxyStr, chromeMajor)
		if err != nil {
			return nil, fmt.Errorf("retry create TLS client: %w", err)
		}
		seedCookieStringTLS(client, cookie)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
		}
	}
	if isLogoutURL(finalURL) {
		return nil, fmt.Errorf("checkpoint/logout redirect: %s", finalURL)
	}
	if body == "" {
		return nil, fmt.Errorf("GET changeemail: empty response")
	}

	// Chờ 1 giây như C#: Task.Delay(1s)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(time.Second):
	}

	// === Extract tokens từ HTML ===
	// Multi-pattern fallback (giống register/webandroid) — FB đổi format JSON theo
	// thời gian: "dtsg" → "DTSGInitData" / "DTSGInitialData", form input fallback.
	state := &webState{
		oldEmail:    reExtract(body, `name="old_email" value="(.*?)"`),
		regInstance: reExtract(body, `name="reg_instance" value="(.*?)"`),
		dtsg: firstMatch(body,
			`(?:DTSG|dtsg)(?:Init(?:ial)?Data)?"\s*:\s*\{\s*"token"\s*:\s*"([^"]+)"`,
			`dtsg":\{"token":"(.*?)"`,
			`name="fb_dtsg"\s+value="([^"]+)"`,
		),
		jazoest: firstMatch(body,
			`name="jazoest"\s+value="(\d+)"`,
			`jazoest",\s*"(\d+)"`,
			`"jazoest"\s*:\s*"?(\d+)"?`,
		),
		lsd: firstMatch(body,
			`LSD".*?token":"(.*?)"`,
			`name="lsd"\s+value="([^"]+)"`,
		),
		encA: reExtract(body, `encrypted":"(.*?)"`),
	}

	// MPageLoadClientMetrics.init — optional, dùng confirm_email_dtsg nếu có
	if re, err2 := regexp.Compile(`MPageLoadClientMetrics\.init\.\(\"(.*?)\", \"\", \"jazoest\", \"(.*?)\"`); err2 == nil {
		if m := re.FindStringSubmatch(body); len(m) > 2 {
			state.confirmDtsg = m[1]
			state.confirmJazo = m[2]
		}
	}

	if state.dtsg == "" {
		return nil, fmt.Errorf("addEmail: cannot extract fb_dtsg from changeemail page (checkpoint?)")
	}

	// === Step 2: POST setemail ===
	postBody := buildAddEmailBody(state, uid, newEmail)
	postH := make(http.Header)
	postH.Set("accept", "*/*")
	postH.Set("sec-fetch-site", "same-origin")
	postH.Set("sec-fetch-mode", "cors")
	postH.Set("sec-fetch-dest", "empty")
	postH.Set("referer", changeEmailURL)
	postH.Set("origin", "https://m.facebook.com")
	major, full, androidVer, deviceModel := parseChromeInfo(ua)
	postH.Set("sec-ch-ua", fmt.Sprintf(`"Chromium";v="%s", "Google Chrome";v="%s", "Not-A.Brand";v="99"`, major, major))
	postH.Set("sec-ch-ua-mobile", "?1")
	postH.Set("sec-ch-ua-platform", `"Android"`)
	postH.Set("sec-ch-ua-platform-version", fmt.Sprintf(`"%s"`, androidVer))
	postH.Set("sec-ch-ua-model", fmt.Sprintf(`"%s"`, deviceModel))
	postH.Set("sec-ch-ua-full-version-list", fmt.Sprintf(`"Chromium";v="%s", "Google Chrome";v="%s", "Not-A.Brand";v="99.0.0.0"`, full, full))
	postH.Set("sec-ch-prefers-color-scheme", "light")
	postH.Set("accept-language", "en-US,en;q=0.9,vi-VN;q=0.8,vi;q=0.7")
	postH.Set("priority", "u=1, i")
	postH.Set("X-Response-Format", "JSONStream")
	postH.Set("X-Requested-With", "XMLHttpRequest")
	postH.Set("user-agent", ua)
	// x-referer cookie thêm vào jar (account cookies đã seed ở đầu function)
	addCookieTLS(client, "x-referer", xRefererChangeEmail)

	logf("[WebAndroid] POST /settings/email/setemail (email=%s)...", newEmail)
	// Retry 3 lần khi proxy drop / FB RST (EOF / forcibly closed).
	var resp string
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = doPostTLS(ctx, client, setEmailURL, postBody, postH, nil)
		if err == nil {
			break
		}
		if !isTransientTLSErr(err) || attempt == 3 {
			return nil, fmt.Errorf("POST setemail: %w", err)
		}
		client.CloseIdleConnections()
		client, err = createTLSClient(proxyStr, chromeMajor)
		if err != nil {
			return nil, fmt.Errorf("retry create TLS client: %w", err)
		}
		seedCookieStringTLS(client, cookie)
		addCookieTLS(client, "x-referer", xRefererChangeEmail)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
		}
	}

	// Parse kết quả — port C# Regex.Unescape: strip TẤT CẢ backslashes trước `/`.
	// Response thực tế từ FB có dạng "MPageController.forceLoad(\"\\/caa\\/reg\\/...")"
	// — JSON nested escape: `\\/` (3 chars) sau khi parse JS code becomes `/`.
	// strings.ReplaceAll(resp, `\/`, `/`) cũ KHÔNG xử lý `\\/` đúng (chỉ thay
	// trailing `\/`, để lại 1 backslash) → check `caa/reg/confirmation` fail.
	respNorm := unescapeJSONSlashes(resp)
	snippet := resp
	if len(snippet) > 300 {
		snippet = snippet[:300]
	}

	// Check checkpoint/block TRƯỚC khi check success — tránh false-positive khi
	// body error chứa chuỗi "confirmemail" trong URL redirect.
	if isCheckpointResponse(resp) {
		return nil, fmt.Errorf("setemail blocked/checkpoint | resp[:300]: %s", snippet)
	}
	// Match C# FacebookVerifyWebAndroidAPI.AddEmailMobile (line 155):
	//   resultapi.Contains("/confirmemail.php?email_changed")
	//   || Regex.Unescape(resultapi).Contains("caa\\/reg\\/confirmation")
	// MPageController.forceLoad("\/caa/reg/confirmation/...") = success indicator.
	if strings.Contains(respNorm, "/confirmemail.php?email_changed") ||
		strings.Contains(respNorm, "caa/reg/confirmation") ||
		strings.Contains(respNorm, "MPageController") && strings.Contains(respNorm, "/caa/reg/") {
		return state, nil
	}
	// C# IsBlockedActionGraphql check (line 159)
	if strings.Contains(resp, "sentry_block_data") || strings.Contains(resp, `"severity":"CRITICAL"`) {
		return nil, fmt.Errorf("setemail blocked (sentry/critical) | resp[:300]: %s", snippet)
	}
	return nil, fmt.Errorf("setemail unexpected response | resp[:300]: %s", snippet)
}

// confirmEmail thực hiện POST confirmation_cliff với OTP code.
// Dùng tls-client (Chrome JA3) — net/http bị FB RST connection.
func confirmEmail(ctx context.Context, proxyStr, cookie, uid, ua, emailAddr, code string, state *webState) error {
	ua = chromeUA(ua)
	chromeMajor := extractChromeMajor(ua)

	client, err := createTLSClient(proxyStr, chromeMajor)
	if err != nil {
		return fmt.Errorf("create TLS client: %w", err)
	}
	defer client.CloseIdleConnections()

	// Seed account cookies + x-referer cookie vào jar
	seedCookieStringTLS(client, cookie)
	addCookieTLS(client, "x-referer", xRefererConfirmCode)

	// Chọn dtsg/jazoest: ưu tiên confirmDtsg nếu có (MPageLoadClientMetrics)
	dtsg := state.confirmDtsg
	jazo := state.confirmJazo
	if dtsg == "" {
		dtsg = state.dtsg
	}
	if jazo == "" {
		jazo = state.jazoest
	}

	confirmBody := buildConfirmBody(dtsg, jazo, state.lsd, state.encA, uid)

	confirmURL := fmt.Sprintf(
		"https://m.facebook.com/confirmation_cliff/?contact=%s&type=submit&is_soft_cliff=false&medium=email&code=%s",
		url.QueryEscape(emailAddr), code,
	)

	h := make(http.Header)
	h.Set("accept", "*/*")
	h.Set("sec-fetch-site", "same-origin")
	h.Set("sec-fetch-mode", "cors")
	h.Set("sec-fetch-dest", "empty")
	h.Set("referer", changeEmailURL)
	h.Set("origin", "https://m.facebook.com")
	major, full, androidVer, deviceModel := parseChromeInfo(ua)
	h.Set("sec-ch-ua", fmt.Sprintf(`"Chromium";v="%s", "Google Chrome";v="%s", "Not-A.Brand";v="99"`, major, major))
	h.Set("sec-ch-ua-mobile", "?1")
	h.Set("sec-ch-ua-platform", `"Android"`)
	h.Set("sec-ch-ua-platform-version", fmt.Sprintf(`"%s"`, androidVer))
	h.Set("sec-ch-ua-model", fmt.Sprintf(`"%s"`, deviceModel))
	h.Set("sec-ch-ua-full-version-list", fmt.Sprintf(`"Chromium";v="%s", "Google Chrome";v="%s", "Not-A.Brand";v="99.0.0.0"`, full, full))
	h.Set("sec-ch-prefers-color-scheme", "light")
	h.Set("accept-language", "en-US,en;q=0.9,vi-VN;q=0.8,vi;q=0.7")
	h.Set("priority", "u=1, i")
	h.Set("X-Response-Format", "JSONStream")
	h.Set("X-Requested-With", "XMLHttpRequest")
	h.Set("user-agent", ua)
	// KHÔNG set Cookie header — tls-client tự gửi từ jar (seed ở đầu function)

	// Retry 3 lần khi gặp lỗi TCP transient (proxy drop, EOF, forcibly closed).
	// Lần 2+ tạo client MỚI với cookie jar mới — tránh stale TCP từ proxy chết.
	var resp string
	for attempt := 1; attempt <= 3; attempt++ {
		resp, err = doPostTLS(ctx, client, confirmURL, confirmBody, h, nil)
		if err == nil {
			break
		}
		if !isTransientTLSErr(err) || attempt == 3 {
			return fmt.Errorf("POST confirmation_cliff: %w", err)
		}
		// Recreate client + reseed cookies cho retry (TCP cũ đã chết)
		client.CloseIdleConnections()
		client, err = createTLSClient(proxyStr, chromeMajor)
		if err != nil {
			return fmt.Errorf("retry create TLS client: %w", err)
		}
		seedCookieStringTLS(client, cookie)
		addCookieTLS(client, "x-referer", xRefererConfirmCode)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
		}
	}
	defer client.CloseIdleConnections()

	// Normalize JSON escape — port C# Regex.Unescape, xử lý cả `\/` và `\\/`.
	respNorm := unescapeJSONSlashes(resp)
	confirmSnippet := resp
	if len(confirmSnippet) > 300 {
		confirmSnippet = confirmSnippet[:300]
	}

	// Check checkpoint TRƯỚC success để tránh false-positive khi error URL chứa
	// keyword "confirmed_account" / "fb://feed".
	if isCheckpointResponse(resp) {
		return fmt.Errorf("confirm blocked/checkpoint | resp[:300]: %s", confirmSnippet)
	}

	// NEGATIVE check 2026-05-18 — nếu response VẪN còn là confirmation form page
	// (FB reject OTP, hiển thị lại trang nhập code) → REJECT bất kể có positive marker.
	// Lý do: response page có thể chứa link "/home.php?confirmed_account" trong nav/script
	// → trigger positive marker false → app ghi nhầm Live trong khi user vẫn ở OTP form.
	lcResp := strings.ToLower(respNorm)
	confirmFormMarkers := []string{
		"enter the confirmation code",       // "Enter the confirmation code" page heading
		"enter confirmation code",           // variant
		"5-digit code we sent",              // "To confirm your account, enter the 5-digit code we sent to..."
		"didn't get the code",               // "I didn't get the code" button
		"didn&#039;t get the code",          // HTML-escaped variant
		"didnt get the code",                // sanitized variant
		`name="code"`,                       // form input field
		`placeholder="confirmation code"`,   // input placeholder
		"/confirmation_cliff/?contact=",     // form action URL → still on OTP page
		"wrong_code",                        // FB error keyword khi OTP sai
		"invalid_code",
		"code_expired",
	}
	for _, marker := range confirmFormMarkers {
		if strings.Contains(lcResp, marker) {
			return fmt.Errorf("confirm STILL on OTP form (FB rejected OTP) | marker=%q | resp[:300]: %s", marker, confirmSnippet)
		}
	}

	// POSITIVE check — port từ C# FacebookVerifyWebAndroidAPI.ConfirmEmailMobile.
	// Chỉ accept khi response CHỨA marker AND đã pass negative check ở trên.
	if strings.Contains(respNorm, `uri":"/home.php`) ||
		strings.Contains(respNorm, "confirmed_account") ||
		strings.Contains(respNorm, `uri":"fb://feed`) {
		return nil
	}
	return fmt.Errorf("confirm unexpected (no specific success marker) | resp[:300]: %s", confirmSnippet)
}

// ─── Form data builders ───────────────────────────────────────────────────────

// buildAddEmailBody — POST setemail form data.
// Mapping CHÍNH XÁC từ C# FacebookApiFormDataBuilder.AddEmailChromeAndroidFormData:
//   string.Join("&", formdata.Select(fdt => $"{fdt.Key}={fdt.Value}"))
// → KHÔNG URL-encode + giữ NGUYÊN thứ tự insertion (C# Dictionary giữ order).
// url.Values.Encode() sort alphabetical + URL-encode → mismatch FB fingerprint.
func buildAddEmailBody(state *webState, uid, newEmail string) string {
	pairs := [][2]string{
		{"next", ""},
		{"old_email", state.oldEmail},
		{"reg_instance", state.regInstance},
		{"new", newEmail},
		{"submit", "Add"},
		{"fb_dtsg", state.dtsg},
		{"jazoest", state.jazoest},
		{"lsd", state.lsd},
		{"__dyn", ""},
		{"__csr", ""},
		{"__hsdp", ""},
		{"__hblpi", ""},
		{"__req", "2"},
		{"__fmt", "1"},
		// `__a` raw từ page (có thể rỗng) — C# dùng trực tiếp, không fallback "1".
		{"__a", state.encA},
		{"__user", uid},
	}
	return joinFormPairsRaw(pairs)
}

// buildConfirmBody — POST confirmation_cliff form data.
// Mapping CHÍNH XÁC từ C# FacebookApiFormDataBuilder.ConfirmEmailRequestFormData.
func buildConfirmBody(dtsg, jazoest, lsd, encA, uid string) string {
	pairs := [][2]string{
		{"fb_dtsg", dtsg},
		{"jazoest", jazoest},
		{"lsd", lsd},
		{"__dyn", ""},
		{"__csr", ""},
		{"__req", "3"},
		{"__fmt", "1"},
		{"__a", encA},
		{"__user", uid},
	}
	return joinFormPairsRaw(pairs)
}

// joinFormPairsRaw join key=value pairs với "&" KHÔNG URL-encode value.
// Match C# string.Join("&", dict.Select(fdt => $"{fdt.Key}={fdt.Value}")).
// Lý do KHÔNG encode: dtsg/__a/lsd là token sạch (alphanumeric + base64 chars),
// `new` là email (FB accept @ raw), `submit` = "Add" (no special). Encode sẽ
// thay đổi byte-level form data và có thể khiến FB phát hiện non-browser request.
func joinFormPairsRaw(pairs [][2]string) string {
	var b strings.Builder
	for i, kv := range pairs {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(kv[0])
		b.WriteByte('=')
		b.WriteString(kv[1])
	}
	return b.String()
}

// ─── Detection helpers ────────────────────────────────────────────────────────

var logoutURLKeys = []string{
	"/login", "/checkpoint", "login.php", "checkpoint/?next",
}

func isLogoutURL(u string) bool {
	lower := strings.ToLower(u)
	for _, k := range logoutURLKeys {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

// checkpointPatterns — chỉ dùng pattern thật sự xác định checkpoint/block.
// KHÔNG dùng "checkpoint" chung chung vì nhiều URL FB redirect bình thường
// cũng chứa chuỗi đó (vd: /confirmation_cliff/?next=... có thể chứa checkpoint param).
//
// Port C# FacebookVerifyWebAndroidAPI.IsBlockedActionGraphql + IsCheckpointRequired:
//   - "sentry_block_data" — FB backend trả khi action bị sentry block
//   - "severity":"CRITICAL" — GraphQL error marker
//   - "checkpointed" — body có marker này (từ prependIntegritySentinel)
var checkpointPatterns = []string{
	"blocked_action",
	"you can't use this feature",
	"we've blocked",
	"code\":459",
	"\"code\":459",
	"\"code\": 459", // variant có space
	"/checkpoint/?next",
	"/checkpoint/",          // ADDED 2026-05-18: catch /checkpoint/<id>/?next=... patterns
	"facebook.com/checkpoint", // ADDED: backup pattern khi URL có domain
	"confirm you're human",  // ADDED: "Confirm you're human to use your account" page
	"confirm you are human",
	"login_attempt_failed",
	"your account has been",
	"sentry_block_data",         // C# IsBlockedActionGraphql
	"\"severity\":\"critical\"", // C# IsBlockedActionGraphql (đã toLower → match "critical")
	"checkpointed",              // sentinel từ response header detection
}

func isCheckpointResponse(body string) bool {
	lower := strings.ToLower(body)
	for _, p := range checkpointPatterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// ─── CheckLiveDie ─────────────────────────────────────────────────────────────

// checkLiveDie kiểm tra account qua Graph API /picture endpoint.
//
// QUAN TRỌNG: dùng client qua proxy + honor ctx (tránh leak IP máy thật).
//
// Live — response có field "height" HOẶC không check được (network error/timeout).
//        Lý do default Live khi inconclusive: account đã pass addEmail + confirmEmail
//        thành công ⇒ đã verified ở FB. CheckLiveDie chỉ là secondary check để
//        confirm account chưa bị disable ngay sau verify. Network lỗi không có
//        nghĩa account die — chỉ nghĩa không gọi được Graph API lúc đó.
// Die  — response chứa avatar default `/C5yt7Cqf3zU.jpg` HOẶC không có "height".
//
// CHANGED 2026-05: bỏ status "Unknown" — hiển thị chỉ Live/Die cho UI thân thiện
// (user request). Inconclusive cases → optimistic "Live" (verify đã success).
func checkLiveDie(ctx context.Context, client *http.Client, token, uid string) string {
	_ = token // không dùng — Picture API không cần token
	if uid == "" {
		return "Live" // verify đã success, UID rỗng = bug khác → optimistic Live
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://graph.facebook.com/"+uid+"/picture?type=normal&redirect=false", nil)
	if err != nil {
		return "Live" // request build error → fallback optimistic
	}
	resp, err := client.Do(req)
	if err != nil {
		return "Live" // network/timeout → giả định account live (verify đã pass)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	ret := string(b)
	if ret == "" {
		return "Live" // empty body → giả định Live (server có response nhưng không trả picture)
	}
	if strings.Contains(ret, "/C5yt7Cqf3zU.jpg") || !strings.Contains(ret, "height") {
		return "Die"
	}
	return "Live"
}

// ─── Misc ─────────────────────────────────────────────────────────────────────

// reExtract tìm group[1] của regex pattern trong src.
func reExtract(src, pattern string) string {
	m := regexp.MustCompile(pattern).FindStringSubmatch(src)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// firstMatch chạy lần lượt patterns, trả group[1] đầu tiên non-empty.
// Cho phép multi-pattern fallback khi FB đổi format JSON theo thời gian.
func firstMatch(src string, patterns ...string) string {
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			continue
		}
		m := re.FindStringSubmatch(src)
		if len(m) > 1 && m[1] != "" {
			return m[1]
		}
	}
	return ""
}

// unescapeJSONSlashes equivalent C# Regex.Unescape() cho path slashes:
// strip TẤT CẢ backslashes liên tiếp đứng trước `/` (bất kể 1, 2, 3 backslash...).
// Cần thiết vì FB response có nested JSON escape:
//   - 1 lớp: `\/` (regex/JSON escape thường)
//   - 2 lớp: `\\/` (JSON-in-JSON như MPageController.forceLoad("\/..."))
//   - 3 lớp: `\\\/` (rare, triple-nested)
// `strings.ReplaceAll(s, `\/`, `/`)` cũ KHÔNG xử lý đúng vì sau khi thay
// `\/` → `/` trong `\\/`, còn lại 1 backslash trước `/`.
var reJSONSlashEscape = regexp.MustCompile(`\\+/`)

func unescapeJSONSlashes(s string) string {
	return reJSONSlashEscape.ReplaceAllString(s, "/")
}

func mminWA(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// isTransientTLSErr xác định lỗi TCP-level cần retry.
// EOF / forcibly closed / connection reset / broken pipe → transient (proxy drop, FB RST).
// Context cancel / HTTP error → KHÔNG retry.
func isTransientTLSErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "eof") ||
		strings.Contains(msg, "forcibly closed") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "wsarecv") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "i/o timeout")
}
