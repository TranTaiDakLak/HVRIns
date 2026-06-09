// steps.go — Verify steps B1-B5 + ChangeLanguage + Resend
// EXACT headers/endpoints/referers từ WeBM WemakeFacebook.Func.12.Verify.cs
package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/instagram"
	"HVRIns/internal/proxy"

	"github.com/google/uuid"
)

// defaultChromeDesktopUA — UA Chrome Desktop dùng khi session UA hoàn toàn không hợp lệ.
// Pool WebMobile (Config/UserAgent/WebChrome_UA.txt) là nguồn UA chính cho m.facebook.com.
// Default này chỉ là fallback cuối cùng khi pool rỗng + session UA trống.
const defaultChromeDesktopUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36"

// Regex parse các phần của UA để build sec-ch-ua-* headers consistent.
var (
	chromeMajorVerRe   = regexp.MustCompile(`Chrome/(\d+)`)
	edgeVerRe          = regexp.MustCompile(`Edg/(\d+)`)
	androidVerRe       = regexp.MustCompile(`Android (\d+(?:\.\d+)?(?:\.\d+)?)`)
	windowsNTVerRe     = regexp.MustCompile(`Windows NT (\d+\.\d+)`)
	macOSVerRe         = regexp.MustCompile(`Mac OS X (\d+[._]\d+(?:[._]\d+)?)`)
	deviceModelRe      = regexp.MustCompile(`(?:Android \d+(?:\.\d+)*(?:\.\d+)*;\s*)([^)]+?)(?:\s+Build/|\))`)
)

// resolveWebUA — pass-through; UA đã được chọn từ pickUAForVerifyPlatform (pool WebMobile).
// Chỉ fallback Chrome Desktop nếu session UA hoàn toàn rỗng.
func resolveWebUA(sessionUA string) string {
	sessionUA = strings.TrimSpace(sessionUA)
	if sessionUA == "" {
		return defaultChromeDesktopUA
	}
	return sessionUA
}

// chromeMajor parse Chrome major version từ UA, fallback "134" nếu không tìm thấy.
func chromeMajor(ua string) string {
	if m := chromeMajorVerRe.FindStringSubmatch(ua); len(m) > 1 {
		return m[1]
	}
	return "134"
}

// secChUaHints — kết quả parse UA cho sec-ch-ua-* headers.
// Build động theo UA thật để FB không detect mismatch.
type secChUaHints struct {
	BrandList       string // sec-ch-ua: "Chromium";v="134", "Not:A-Brand";v="24", "Google Chrome";v="134"
	BrandFullList   string // sec-ch-ua-full-version-list
	Mobile          string // sec-ch-ua-mobile: "?0" (desktop) | "?1" (mobile)
	Platform        string // sec-ch-ua-platform: "Windows" | "Android" | "macOS" | "iOS" | "Linux"
	PlatformVersion string // sec-ch-ua-platform-version: "10.0.0" | "13.0.0" | ...
	Model           string // sec-ch-ua-model: "" desktop | "Pixel 7" mobile
}

// parseSecChUaHints — phân tích UA → build hints cho sec-ch-ua-* headers.
// Match đúng FB checker logic: UA mobile thì hints mobile, UA Windows thì hints Windows.
func parseSecChUaHints(ua string) secChUaHints {
	major := chromeMajor(ua)
	hints := secChUaHints{
		BrandList:       fmt.Sprintf(`"Chromium";v="%s", "Not:A-Brand";v="24", "Google Chrome";v="%s"`, major, major),
		BrandFullList:   fmt.Sprintf(`"Chromium";v="%s.0.0.0", "Not:A-Brand";v="24.0.0.0", "Google Chrome";v="%s.0.0.0"`, major, major),
		Mobile:          "?0",
		Platform:        `"Windows"`,
		PlatformVersion: `"10.0.0"`,
		Model:           `""`,
	}

	// Edge variant: brand list khác (Microsoft Edge thay Google Chrome).
	if m := edgeVerRe.FindStringSubmatch(ua); len(m) > 1 {
		edgeMajor := m[1]
		hints.BrandList = fmt.Sprintf(`"Chromium";v="%s", "Not:A-Brand";v="24", "Microsoft Edge";v="%s"`, major, edgeMajor)
		hints.BrandFullList = fmt.Sprintf(`"Chromium";v="%s.0.0.0", "Not:A-Brand";v="24.0.0.0", "Microsoft Edge";v="%s.0.0.0"`, major, edgeMajor)
	}

	// Mobile detection — quan trọng nhất, FB check key này đầu tiên.
	if strings.Contains(ua, "Mobile") {
		hints.Mobile = "?1"
	}

	// Platform detection theo thứ tự ưu tiên.
	switch {
	case strings.Contains(ua, "Android"):
		hints.Platform = `"Android"`
		if m := androidVerRe.FindStringSubmatch(ua); len(m) > 1 {
			hints.PlatformVersion = `"` + m[1] + `"`
		} else {
			hints.PlatformVersion = `"13.0.0"`
		}
		// Device model parse từ UA pattern: "Android 11; Pixel 7 Build/..."
		if dm := deviceModelRe.FindStringSubmatch(ua); len(dm) > 1 {
			model := strings.TrimSpace(dm[1])
			// Strip locale tag (vd: "ko-kr; LG-L160L" → "LG-L160L")
			if idx := strings.LastIndex(model, "; "); idx >= 0 {
				model = strings.TrimSpace(model[idx+2:])
			}
			hints.Model = `"` + model + `"`
		}
	case strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad"):
		hints.Platform = `"iOS"`
		hints.PlatformVersion = `"17.0.0"`
		hints.Mobile = "?1"
	case strings.Contains(ua, "Macintosh") || strings.Contains(ua, "Mac OS X"):
		hints.Platform = `"macOS"`
		if m := macOSVerRe.FindStringSubmatch(ua); len(m) > 1 {
			// "10_15_7" → "10.15.7"
			ver := strings.ReplaceAll(m[1], "_", ".")
			hints.PlatformVersion = `"` + ver + `"`
		} else {
			hints.PlatformVersion = `"14.0.0"`
		}
	case strings.Contains(ua, "Windows"):
		hints.Platform = `"Windows"`
		// Windows NT 10.0 dùng cho cả Win 10 và Win 11.
		// Phân biệt qua sec-ch-ua-platform-version: 10.0.0 = Win 10, 15.0.0 = Win 11.
		if windowsNTVerRe.MatchString(ua) {
			hints.PlatformVersion = `"10.0.0"`
		}
	case strings.Contains(ua, "Linux"):
		hints.Platform = `"Linux"`
		hints.PlatformVersion = `"6.5.0"`
	}

	return hints
}

// generateUUID sinh UUID v4 ngẫu nhiên dùng làm waterfallId cho mỗi bước verify.
// waterfallId là identifier duy nhất cho một "luồng" yêu cầu verify — Facebook dùng để
// track trạng thái session từ B1 đến B5. Mỗi lần retry phải tạo waterfallId mới.
func generateUUID() string {
	return uuid.New().String()
}

// setVerifyHeaders gán toàn bộ HTTP headers cho request verify B1-B5.
// EXACT headers từ WeBM WemakeFacebook.Func.12.Verify.cs lines 296-313.
//
// req:     HTTP request đang được chuẩn bị — headers sẽ được set trực tiếp vào đây.
// s:       session Facebook — lấy UserAgent và Cookie từ đây.
// referer: URL trang nguồn, khác nhau giữa B1 và B2-B5 (xem constants.go).
func setVerifyHeaders(req *http.Request, s *instagram.Session, referer string) {
	ua := resolveWebUA(s.UserAgent)
	hints := parseSecChUaHints(ua)

	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("origin", "https://m.facebook.com")
	req.Header.Set("priority", "u=1, i")
	if referer != "" {
		req.Header.Set("referer", referer)
	}
	req.Header.Set("sec-ch-prefers-color-scheme", "light")
	// sec-ch-ua-* build động theo UA → tránh FB phát hiện UA-vs-headers mismatch.
	req.Header.Set("sec-ch-ua", hints.BrandList)
	req.Header.Set("sec-ch-ua-full-version-list", hints.BrandFullList)
	req.Header.Set("sec-ch-ua-mobile", hints.Mobile)
	req.Header.Set("sec-ch-ua-model", hints.Model)
	req.Header.Set("sec-ch-ua-platform", hints.Platform)
	req.Header.Set("sec-ch-ua-platform-version", hints.PlatformVersion)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("user-agent", ua)
	// WeBM: UseCookies=false, set Cookie header trực tiếp (line 21, 313)
	req.Header.Set("Cookie", s.Cookie)
}

// setLangHeaders gán HTTP headers cho request đổi ngôn ngữ.
// EXACT headers từ WeBM ChangeLanguageV2 lines 196-209.
//
// req: HTTP request đang được chuẩn bị.
// s:   session Facebook — lấy UserAgent, Lsd, Cookie từ đây.
func setLangHeaders(req *http.Request, s *instagram.Session) {
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "vi,en;q=0.9")
	req.Header.Set("origin", "https://m.facebook.com")
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("referer", RefererLang)
	req.Header.Set("sec-ch-prefers-color-scheme", "dark")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("user-agent", resolveWebUA(s.UserAgent))
	req.Header.Set("x-asbd-id", "359341")
	req.Header.Set("x-fb-lsd", s.Lsd)
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("x-response-format", "JSONStream")
	req.Header.Set("Cookie", s.Cookie)
}

// doPost gửi POST request đến Facebook endpoint, trả về response text và status code.
// Tái dùng s.Client nếu đã có (tạo 1 lần ở VerifyAccount), ngược lại tạo client tạm thời.
//
// ctx:      context để cancel/timeout — truyền từ runner, thường có deadline 20-30s.
// s:        session Facebook — lấy s.Client (shared) hoặc s.Proxy để tạo client mới.
// endpoint: URL đích, VD: EndpointB1, EndpointB2... (xem constants.go).
// body:     form-urlencoded body string đã được build bởi body builder tương ứng.
// referer:  HTTP Referer header, khác nhau theo từng bước B1-B5.
func doPost(ctx context.Context, s *instagram.Session, endpoint, body, referer string) (string, int, error) {
	// Retry strategy: lên tới 3 lần khi gặp EOF / forcibly closed / connection reset
	// (transient TCP-level error từ FB / proxy). Mỗi retry lần thứ 2+ dùng client MỚI
	// (không reuse shared client) để tránh stale TCP connection bị FB drop.
	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		client := s.Client
		if client == nil || attempt > 1 {
			// Lần đầu: dùng shared client. Retry: tạo client mới để skip stale conn.
			client = proxy.CreateClient(s.Proxy, 20*time.Second)
			if attempt == maxAttempts {
				defer client.CloseIdleConnections()
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(body))
		if err != nil {
			return "", 0, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		setVerifyHeaders(req, s, referer)

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			// Retry chỉ khi lỗi là TCP-level (EOF, RST, forcibly closed)
			if isTransientNetErr(err) && attempt < maxAttempts {
				select {
				case <-ctx.Done():
					return "", 0, ctx.Err()
				case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
				}
				continue
			}
			return "", 0, err
		}
		defer resp.Body.Close()

		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		return string(respBody), resp.StatusCode, nil
	}
	return "", 0, lastErr
}

// isTransientNetErr xác định lỗi có nên retry không.
// EOF / forcibly closed / connection reset / broken pipe → transient, retry.
// Context cancel / 4xx → KHÔNG retry.
func isTransientNetErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "eof") ||
		strings.Contains(msg, "forcibly closed") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "wsarecv") ||
		strings.Contains(msg, "connection refused")
}

// ChangeLanguageV2 đổi ngôn ngữ tài khoản sang en_US trước khi bắt đầu verify.
// EXACT mapping từ WeBM lines 187-245.
// Phải gọi trước SelectMail (B1) để đảm bảo các response từ Facebook là tiếng Anh.
//
// ctx: context để cancel/timeout request.
// s:   session Facebook — cần Cookie, UID, FbDtsg, Jazoest, Lsd, Dyn, UserAgent, Proxy.
func ChangeLanguageV2(ctx context.Context, s *instagram.Session) error {
	client := s.Client
	if client == nil {
		client = proxy.CreateClient(s.Proxy, 20*time.Second)
		defer client.CloseIdleConnections()
	}

	// Form data — exact từ WeBM lines 213-228
	body := fmt.Sprintf("loc=en_US&ref=m_touch_locale_selector&should_redirect=false&fb_dtsg=%s&jazoest=%s&lsd=%s&__dyn=%s&__csr=&__hsdp=&__hblp=&__sjsp=&__req=3&__fmt=1&__a=1&__user=%s",
		s.FbDtsg, s.Jazoest, s.Lsd, s.Dyn, s.UID)

	req, err := http.NewRequestWithContext(ctx, "POST", EndpointLang, strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	setLangHeaders(req, s)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // drain để transport tái sử dụng TCP connection

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ChangeLanguage HTTP %d", resp.StatusCode)
	}
	return nil
}

// SelectMail thực hiện B1: yêu cầu Facebook gửi code xác thực về email.
// EXACT mapping từ WeBM Verify.cs lines 289-330.
// WeBM: _requestFb.GetContent đã bị comment, chỉ dùng client.SendAsync.
//
// ctx:         context để cancel/timeout request.
// s:           session Facebook — cần Cookie, UID, FbDtsg, Jazoest, Lsd, Dyn, FullName, Proxy.
// waterfallId: UUID v4 dùng để định danh luồng verify — tạo 1 lần ở verify.go, dùng xuyên suốt B1-B5.
func SelectMail(ctx context.Context, s *instagram.Session, waterfallId string) error {
	firstName, lastName := splitName(s.FullName)
	body := vrfy_buildB1Body(s, waterfallId, firstName, lastName)
	respText, status, err := doPost(ctx, s, EndpointB1, body, RefererB1)
	if err != nil {
		return err
	}
	if status >= 400 {
		snippet := respText
		if len(snippet) > 500 {
			snippet = snippet[:500]
		}
		return fmt.Errorf("B1 HTTP %d — %s", status, snippet)
	}
	if err := checkBloksBodyError(respText, "B1"); err != nil {
		return err
	}
	return nil
}

// ChangeEmail thực hiện B2: chuyển phương thức xác thực sang email.
// EXACT mapping từ WeBM Verify.cs lines 502-557.
//
// ctx:         context để cancel/timeout request.
// s:           session Facebook — cần Cookie, UID, FbDtsg, Jazoest, Lsd, Dyn, FullName, Proxy.
// waterfallId: UUID v4 định danh luồng verify — phải cùng giá trị đã dùng ở B1.
func ChangeEmail(ctx context.Context, s *instagram.Session, waterfallId string) error {
	firstName, lastName := splitName(s.FullName)
	body := vrfy_buildB2Body(s, waterfallId, firstName, lastName)
	respText, status, err := doPost(ctx, s, EndpointB2, body, RefererB2)
	if err != nil {
		return err
	}
	if status >= 400 {
		snippet := respText
		if len(snippet) > 500 {
			snippet = snippet[:500]
		}
		return fmt.Errorf("B2 HTTP %d — %s", status, snippet)
	}
	if err := checkBloksBodyError(respText, "B2"); err != nil {
		return err
	}
	return nil
}

// SubmitEmail thực hiện B3: gửi địa chỉ email đích để Facebook dùng làm nơi nhận code.
// EXACT mapping từ WeBM Verify.cs lines 563-619.
// Trả về responseSnippet (tối đa 300 ký tự) để verify.go ghi log.
//
// ctx:         context để cancel/timeout request.
// s:           session Facebook — cần Cookie, UID, FbDtsg, Jazoest, Lsd, Dyn, FullName, Proxy.
// waterfallId: UUID v4 định danh luồng verify — phải cùng giá trị đã dùng ở B1-B2.
// email:       địa chỉ email sẽ nhận code OTP (từ mail provider đã mua/lấy trước đó).
func SubmitEmail(ctx context.Context, s *instagram.Session, waterfallId, email string) (string, error) {
	firstName, lastName := splitName(s.FullName)
	eventRequestId := generateUUID()
	body := vrfy_buildB3Body(s, waterfallId, firstName, lastName, email, eventRequestId)
	respText3, status, err := doPost(ctx, s, EndpointB3, body, RefererB2)
	if err != nil {
		return "", err
	}
	snippet := respText3
	if len(snippet) > 300 {
		snippet = snippet[:300]
	}
	if status >= 400 {
		return snippet, fmt.Errorf("B3 HTTP %d — %s", status, snippet)
	}
	if err := checkBloksBodyError(respText3, "B3"); err != nil {
		return snippet, err
	}
	return snippet, nil
}

// LoadConfirmation thực hiện B4: tải trang nhập code xác nhận.
// EXACT mapping từ WeBM Verify.cs lines 626-685.
// WeBM B4 dùng EMAIL reg_info (contactpoint=email, device_id=datr), __req=6, __hs=20539.
// Trả về responseSnippet (tối đa 300 ký tự) để verify.go ghi log.
//
// ctx:         context để cancel/timeout request.
// s:           session Facebook — cần Cookie, UID, FbDtsg, Jazoest, Lsd, Dyn, Datr, FullName, Proxy.
// waterfallId: UUID v4 định danh luồng verify — phải cùng giá trị đã dùng ở B1-B3.
// email:       địa chỉ email đã submit ở B3 — dùng để build contactpoint param.
func LoadConfirmation(ctx context.Context, s *instagram.Session, waterfallId, email string) (string, error) {
	firstName, lastName := splitName(s.FullName)
	emailUser, emailDomain := splitEmail(email)
	body := vrfy_buildB4Body(s, waterfallId, firstName, lastName, emailUser, emailDomain)
	respText4, status, err := doPost(ctx, s, EndpointB4, body, RefererB2)
	if err != nil {
		return "", err
	}
	snippet := respText4
	if len(snippet) > 300 {
		snippet = snippet[:300]
	}
	if status >= 400 {
		return snippet, fmt.Errorf("B4 HTTP %d — %s", status, snippet)
	}
	if err := checkBloksBodyError(respText4, "B4"); err != nil {
		return snippet, err
	}
	return snippet, nil
}

// checkBloksBodyError — detect FB silent reject (HTTP 200 OK nhưng body có error).
// FB Bloks response format: `for (;;);{"__ar":1,"error":<code>,"errorSummary":"...",...}`.
// Khi response có "error":<non-zero> hoặc "errorSummary":<non-empty> → FB đã reject silent.
// Không catch ở step nào → flow tiếp tục → false positive ở B5 / CheckLive.
//
// Catch ở mỗi step B1-B4 → return error → KHÔNG continue → KHÔNG ghi SuccessVerify.
func checkBloksBodyError(respText, stepTag string) error {
	if respText == "" {
		return nil
	}
	// Bloks payload format: "for (;;);{...}" — strip prefix nếu có để parse dễ hơn.
	body := respText
	if strings.HasPrefix(body, "for (;;);") {
		body = body[9:]
	}
	lc := strings.ToLower(body)
	// Pattern 1: "error":<số khác 0> — FB error code
	// Vd: "error":1357004 hoặc "error":1357027
	if i := strings.Index(lc, `"error":`); i >= 0 {
		// Skip "error":null hoặc "error":0
		rest := lc[i+8:]
		rest = strings.TrimLeft(rest, " ")
		if !strings.HasPrefix(rest, "null") && !strings.HasPrefix(rest, "0,") && !strings.HasPrefix(rest, "0}") && !strings.HasPrefix(rest, "false") {
			// Extract error code + summary cho log
			snippet := body
			if len(snippet) > 300 {
				snippet = snippet[:300]
			}
			summary := extractBetween(body, `"errorSummary":"`, `"`)
			if summary == "" {
				summary = extractBetween(body, `errorSummary":"`, `"`)
			}
			return fmt.Errorf("%s FB silent reject (errorSummary=%q) body[:300]=%s", stepTag, summary, snippet)
		}
	}
	// Pattern 2: "errorSummary":"..." (non-empty) — backup nếu "error" pattern không catch
	if strings.Contains(lc, `"errorsummary":"sorry`) ||
		strings.Contains(lc, `"errorsummary":"something went wrong`) {
		snippet := body
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return fmt.Errorf("%s FB error 'Sorry, something went wrong' body[:300]=%s", stepTag, snippet)
	}
	return nil
}

// ConfirmOTP thực hiện B5: submit code OTP để hoàn tất xác thực email.
// EXACT mapping từ WeBM Verify.cs lines 691-793.
// Kiểm tra error patterns trong response (errorsummary, wrong_code, code_expired...).
//
// ctx:         context để cancel/timeout request.
// s:           session Facebook — cần Cookie, UID, FbDtsg, Jazoest, Lsd, Dyn, Datr, FullName, Proxy.
// waterfallId: UUID v4 định danh luồng verify — phải cùng giá trị đã dùng ở B1-B4.
// email:       địa chỉ email đã dùng ở B3-B4 — dùng để build referer và body.
// code:        code OTP 6 chữ số nhận từ email provider (ZeusX, Mail30s, Store1s, DongVanFB...).
func ConfirmOTP(ctx context.Context, s *instagram.Session, waterfallId, email, code string) error {
	firstName, lastName := splitName(s.FullName)
	emailUser, emailDomain := splitEmail(email)
	body := vrfy_buildB5Body(s, waterfallId, firstName, lastName, emailUser, emailDomain, code)
	ref := refererB5(emailUser, emailDomain)

	respText, status, err := doPost(ctx, s, EndpointB5, body, ref)
	if err != nil {
		return err
	}

	if status >= 400 {
		snippet := respText
		if len(snippet) > 500 {
			snippet = snippet[:500]
		}
		return fmt.Errorf("B5 HTTP %d — %s", status, snippet)
	}

	// Error pattern check — exact từ WeBM lines 746-758
	respLower := strings.ToLower(respText)
	if strings.Contains(respLower, "errorsummary") ||
		strings.Contains(respLower, "wrong_code") ||
		strings.Contains(respLower, "code_expired") ||
		strings.Contains(respLower, "invalid_code") ||
		strings.Contains(respLower, "you can't make this change at the moment") {
		// Extract title giống WeBM: StringHelper.GetValue(responseText, ", \"title\", \"", "\",")
		title := extractBetween(respText, `, "title", "`, `",`)
		if title == "" {
			title = extractBetween(respText, `"errorSummary":"`, `"`)
		}
		if title == "" {
			title = "unknown"
		}
		return fmt.Errorf("OTP error [%s]", title)
	}

	return nil
}

// ResendConfirmationCode yêu cầu Facebook gửi lại code OTP khi code cũ hết hạn hoặc không đến.
// EXACT mapping từ WeBM Verify.cs lines 817-876.
//
// ctx:         context để cancel/timeout request.
// s:           session Facebook — cần Cookie, UID, FbDtsg, Jazoest, Lsd, Dyn, Datr, FullName, Proxy.
// waterfallId: UUID v4 định danh luồng verify — phải cùng giá trị đã dùng ở B1-B4.
// email:       địa chỉ email cần gửi lại code — phải là email đã submit ở B3.
func ResendConfirmationCode(ctx context.Context, s *instagram.Session, waterfallId, email string) error {
	firstName, lastName := splitName(s.FullName)
	emailUser, emailDomain := splitEmail(email)
	body := vrfy_buildResendBody(s, waterfallId, firstName, lastName, emailUser, emailDomain)
	ref := refererB5(emailUser, emailDomain)

	_, status, err := doPost(ctx, s, EndpointResend, body, ref)
	if err != nil {
		return err
	}
	if status >= 400 {
		return fmt.Errorf("Resend HTTP %d", status)
	}
	return nil
}

// extractBetween tìm và trả về chuỗi nằm giữa start và end trong s.
// s: chuỗi nguồn (thường là JSON response từ Facebook).
// start: chuỗi mốc bắt đầu (ví dụ `, "title", "`).
// end: chuỗi mốc kết thúc (ví dụ `",`).
// Trả về "" nếu không tìm thấy start hoặc end.
// Dùng để extract error title từ Bloks response B5 (giống WeBM StringHelper.GetValue).
func extractBetween(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	s = s[i+len(start):]
	j := strings.Index(s, end)
	if j < 0 {
		return ""
	}
	return s[:j]
}

// splitName tách họ tên đầy đủ thành firstName (phần trước dấu cách đầu tiên) và lastName (phần còn lại).
// fullName: tên đầy đủ của account, ví dụ "Văn Hải Nguyễn".
// Trả về (firstName="Văn", lastName="Hải Nguyễn").
// Nếu không có dấu cách: trả về (fullName, "").
// Dùng trong tất cả B1-B5 body builder để điền first_name/last_name.
func splitName(fullName string) (string, string) {
	if i := strings.Index(fullName, " "); i > 0 {
		return fullName[:i], fullName[i+1:]
	}
	return fullName, ""
}

// splitEmail tách email thành user (phần trước @) và domain (phần sau @).
// email: địa chỉ email, ví dụ "example@hotmail.com".
// Trả về (user="example", domain="hotmail.com").
// Nếu không có @: trả về (email, "").
// Dùng trong B3/B4/B5/Resend body builder để điền email_user và email_domain riêng biệt
// (Facebook API yêu cầu tách, không nhận email nguyên vẹn).
func splitEmail(email string) (string, string) {
	if i := strings.Index(email, "@"); i > 0 {
		return email[:i], email[i+1:]
	}
	return email, ""
}

// min trả về giá trị nhỏ hơn giữa a và b.
// a, b: hai số nguyên cần so sánh.
// Dùng trong body builder để tính snippet length an toàn.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
