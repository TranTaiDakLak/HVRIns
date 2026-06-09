// verify.go — Web platform Verifier + VerifyAccount orchestrator + live/die check.
//
// File này gộp 4 file cũ:
//   - verify.go (cũ)     → Verifier interface + init()
//   - types.go           → type aliases (VerifyConfig/VerifyResult)
//   - verify_verify.go   → VerifyAccount + createTempEmail + runB1toB5 + truncStr
//   - check.go           → CheckLiveDieAccount + SaveAccountToFolder + liveDieClient
//
// Package web — Facebook Web platform verify (m.facebook.com endpoints).
// Mapping từ WeBM WemakeFacebook.Func.12.Verify.cs lines 279-955.
package web

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"HVRIns/internal/email"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/verify/verifybase"
	"HVRIns/internal/proxy"
)

// ─── Type aliases ────────────────────────────────────────────────────────────

type (
	VerifyConfig = instagram.VerifyConfig
	VerifyResult = instagram.VerifyResult
)

// Config alias → instagram.VerifyConfig.
type Config = VerifyConfig

// Result alias → instagram.VerifyResult.
type Result = VerifyResult

// VerifyStatusCallback callback để cập nhật trạng thái realtime lên UI.
type VerifyStatusCallback func(accountUID string, message string)

// maxRetryB1B5 = 0 — không retry trong verify, để scheduler.go quản lý retry.
const maxRetryB1B5 = 0

// ─── Verifier interface + init ──────────────────────────────────────────────

// Verifier implements instagram.Verifier cho nền tảng web (m.facebook.com).
type Verifier struct{}

// Verify thực hiện toàn bộ flow verify cho 1 account.
// Implements instagram.Verifier interface.
func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	var cb VerifyStatusCallback
	if onStatus != nil {
		cb = onStatus
	}
	r := VerifyAccount(ctx, session, cfg, outputPath, cb)
	return r
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformWeb, func() instagram.Verifier {
		return &Verifier{}
	})
}

// ─── VerifyAccount orchestrator ──────────────────────────────────────────────
//
// Mapping từ WeBM WemakeFacebook.Func.12.Verify.cs VerifyAccount() lines 279-496.

// VerifyAccount thực hiện toàn bộ quy trình verify cho 1 account.
// Bao gồm: đổi ngôn ngữ → tạo email tạm → chạy B1-B5 → kiểm tra live/die → lưu kết quả.
//
// ctx: context để hủy giữa chừng (user dừng tool, timeout...).
// session: phiên đăng nhập Facebook — phải có đầy đủ: UID, FbDtsg, Jazoest, Lsd, Datr, Jar, Proxy, Phone.
//
//	Thiếu FbDtsg hoặc Jazoest sẽ trả về lỗi ngay.
//
// cfg: cấu hình verify — mail provider, delay OTP, thư mục output, API keys, v.v.
// outputPath: thư mục ghi kết quả Live.txt/Die.txt. Nếu truyền "" thì fallback về cfg.OutputPath.
//
//	Truyền outputPath trực tiếp cho phép mỗi batch ghi vào thư mục riêng
//	mà không cần sửa cfg dùng chung.
//
// onStatus: callback nhận cập nhật trạng thái realtime (prefix [B1], [Mail], [CheckLiveDie], ...).
//
//	Có thể là nil nếu không cần UI update.
func VerifyAccount(ctx context.Context, session *instagram.Session, cfg *Config, outputPath string, onStatus VerifyStatusCallback) *Result {
	uid := session.UID
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(uid, msg)
		}
	}

	// Tạo 1 shared HTTP client, tái dùng cho ChangeLanguage + B1-B5 + Resend
	// Tránh tạo mới 7+ client/TLS handshake riêng lẻ mỗi luồng verify
	sharedClient := proxy.CreateClient(session.Proxy, 20*time.Second)
	defer func() {
		sharedClient.CloseIdleConnections()
		session.Client = nil // clear để tránh reuse sau khi VerifyAccount return
	}()
	session.Client = sharedClient

	notify(fmt.Sprintf("[WebVerify] Bắt đầu — MailProvider=%s", cfg.MailProvider))

	// Phase 1: Đổi ngôn ngữ sang en_US (chỉ 1 lần)
	notify("[ChangeLanguage] Đổi ngôn ngữ sang en_US...")
	if err := ChangeLanguageV2(ctx, session); err != nil {
		notify(fmt.Sprintf("[ChangeLanguage] Thất bại: %v", err))
	}

	// Kiểm tra session tokens (chỉ 1 lần)
	if session.FbDtsg == "" || session.Jazoest == "" {
		notify(fmt.Sprintf("[ERROR] Thiếu tokens! fb_dtsg='%s' jazoest='%s' lsd='%s' datr='%s'",
			session.FbDtsg, session.Jazoest, session.Lsd, session.Datr))
		return &Result{Success: false, Message: "Thiếu session tokens (fb_dtsg/jazoest) — login có thể thất bại"}
	}

	notify(fmt.Sprintf("[Tokens] fb_dtsg=%s... jazoest=%s datr=%s... phone=%s hasJar=%v",
		truncStr(session.FbDtsg, 15), session.Jazoest, truncStr(session.Datr, 8), session.Phone, session.Jar != nil))

	// Advanced options: FmUserTmpMail + UseProxyTempMail (port C# MainFormUISettings)
	customUsername := ""
	if cfg.FmUserTmpMail {
		login := session.Phone
		if login == "" {
			login = session.UID
		}
		customUsername = email.CreateUsernameFromLogin(login)
	}
	// Proxy override cho mail reading:
	//  - Rent provider hỗ trợ proxy (zeus-x, muamail, unlimitmail) → pick từ proxy_rentmail.txt
	//  - Temp provider → pick từ proxy_tempmail.txt
	proxyOverride := ""
	if email.IsRentMailProvider(cfg.MailProvider) {
		if cfg.UseProxyGmail {
			proxyOverride = email.PickRentMailProxy()
		}
	} else if cfg.UseProxyTempMail {
		proxyOverride = email.PickTempMailProxy()
	}

	// Tạo email service 1 lần
	emailSvc, err := email.New(email.Options{
		Provider:                cfg.MailProvider,
		ProxyOverride:           proxyOverride,
		CustomUsername:          customUsername,
		ProxyStr:                session.Proxy,
		OnStatus:                func(msg string) { notify(msg) },
		Pool:                    cfg.EmailPool,
		ZeusXApiKey:             cfg.ZeusXApiKey,
		ZeusXAccountCode:        cfg.ZeusXAccountCode,
		DvfbApiKey:              cfg.DvfbApiKey,
		DvfbAccountType:         cfg.DvfbAccountType,
		Store1sApiKey:           cfg.Store1sApiKey,
		Store1sProductID:        cfg.Store1sProductID,
		Mail30sApiKey:           cfg.Mail30sApiKey,
		Mail30sProductSlug:      cfg.Mail30sProductSlug,
		TempMailLolApiKey:       cfg.TempMailLolApiKey,
		TempMailDomain:          cfg.TempMailDomain,
		MuaMailApiKey:           cfg.MuaMailApiKey,
		MuaMailProductID:        cfg.MuaMailProductID,
		UnlimitMailApiKey:       cfg.UnlimitMailApiKey,
		UnlimitMailProductID:    cfg.UnlimitMailProductID,
		SptMailApiKey:           cfg.SptMailApiKey,
		SptMailServiceCode:      cfg.SptMailServiceCode,
		EmailAPIInfoApiKey:      cfg.EmailAPIInfoApiKey,
		EmailAPIInfoProductCode: cfg.EmailAPIInfoProductCode,
		OtpCheapApiKey:          cfg.OtpCheapApiKey,
		OtpCheapServiceID:       cfg.OtpCheapServiceID,
		ShopGmail9999ApiKey:     cfg.ShopGmail9999ApiKey,
		ShopGmail9999Service:    cfg.ShopGmail9999Service,
		RentGmailApiKey:         cfg.RentGmailApiKey,
		RentGmailPlatform:       cfg.RentGmailPlatform,
		OtpCodesSmsApiKey:       cfg.OtpCodesSmsApiKey,
		OtpCodesSmsServiceID:    cfg.OtpCodesSmsServiceID,
		WmemailApiKey:           cfg.WmemailApiKey,
		WmemailCommodity:        cfg.WmemailCommodity,
		PriyoEmailApiKey:        cfg.PriyoEmailApiKey,
		OTPHotmailPriority:      cfg.OTPHotmailPriority,
		TempMailToken:           cfg.TempMailToken,
	})
	if err != nil {
		return &Result{Success: false, Message: fmt.Sprintf("email.New: %v", err)}
	}
	defer emailSvc.Close()

	// Retry B1-B5: tối đa 3 lần (1 + 2 retry)
	// Chỉ tạo email mới khi: chưa có email / tạo email thất bại / không nhận được OTP
	var lastErr string
	var tempEmail string
	needNewEmail := true // lần đầu luôn cần tạo email

	// TempMail reuse: nếu reg đã tạo mail tạm + lưu creds vào session.EmailMeta,
	// Restore + skip CreateEmail loop. Web verify B1-B5 vẫn chạy (FB web flow
	// có UI navigation phức tạp; B1-B3 attempts to add email mới có thể bị FB
	// reject vì email đã linked, hoặc duplicate-detection accept lại — tùy phiên
	// FB. Reuse mode tối thiểu tiết kiệm provider cost.
	if session.Email != "" && session.EmailMeta != "" && email.RestoreIfPossible(emailSvc, session.EmailMeta) {
		tempEmail = session.Email
		needNewEmail = false
		notify(fmt.Sprintf("[Mail] Reuse mail từ register: %s (skip CreateEmail)", tempEmail))
		if cfg.OnEmailCreated != nil {
			cfg.OnEmailCreated(tempEmail)
		}
	}

	for attempt := 1; attempt <= maxRetryB1B5+1; attempt++ {
		select {
		case <-ctx.Done():
			return &Result{Success: false, Message: "Đã dừng"}
		default:
		}

		if attempt > 1 {
			notify(fmt.Sprintf("[Retry] Thử lại lần %d/%d...", attempt-1, maxRetryB1B5))
		}

		// Chỉ tạo email mới khi cần (lần đầu / email fail / OTP fail)
		if needNewEmail {
			newEmail, err := createTempEmail(ctx, emailSvc, notify, cfg.OnEmailCreated)
			if err != nil {
				lastErr = err.Error()
				needNewEmail = true // lần sau vẫn cần tạo mới
				continue
			}
			tempEmail = newEmail
		} else {
			notify(fmt.Sprintf("[Mail] Dùng lại email: %s", tempEmail))
		}

		// Chạy B1 → B5
		failedAt, err := runB1toB5(ctx, session, cfg, emailSvc, notify, tempEmail)
		if err == nil {
			// Thành công
			lastErr = ""
			break
		}

		lastErr = err.Error()
		notify(fmt.Sprintf("[Lần %d/%d] Thất bại: %s", attempt, maxRetryB1B5+1, lastErr))

		// Chỉ tạo email mới nếu: tạo email fail hoặc không nhận được OTP
		// Nếu B1/B2/B3/B4/B5 API fail → dùng lại email cũ
		needNewEmail = (failedAt == "email" || failedAt == "otp")
	}

	if lastErr != "" {
		return &Result{Success: false, Message: lastErr, Email: tempEmail}
	}

	// Check Live/Die — periodic every 5s, bail early on die
	checkDelaySec := cfg.TimeDelayCheck
	checkInterval := 5
	var status string
	if checkDelaySec > 0 {
		notify(fmt.Sprintf("[CheckLiveDie] Kiểm tra live/die mỗi %ds, tổng %ds...", checkInterval, checkDelaySec))
		status = "Live"
		elapsed := 0
		for elapsed < checkDelaySec {
			wait := checkInterval
			if elapsed+wait > checkDelaySec {
				wait = checkDelaySec - elapsed
			}
			select {
			case <-ctx.Done():
				return &Result{Success: true, Message: "Verify OK (chưa check live/die)", Email: tempEmail}
			case <-time.After(time.Duration(wait) * time.Second):
			}
			elapsed += wait
			notify(fmt.Sprintf("[CheckLiveDie] [%ds/%ds] Kiểm tra...", elapsed, checkDelaySec))
			// Combined check — token /me (catch checkpoint ngay) + picture (fallback).
			s := verifybase.CheckLiveDieCombined(ctx, session.UserAgent, session.UID, session.Token)
			if s == "Die" {
				status = "Die"
				break
			}
		}
		notify(fmt.Sprintf("[CheckLiveDie] Kết quả: %s", status))
	} else {
		// Combined check — token /me (catch checkpoint ngay) + picture (fallback).
		// Web (MFB) thường không có Token sẵn → fallback picture; nếu REG có cung cấp
		// session.Token thì sẽ catch checkpoint ngay lập tức.
		status = verifybase.CheckLiveDieCombined(ctx, session.UserAgent, session.UID, session.Token)
		notify(fmt.Sprintf("[CheckLiveDie] Kết quả: %s", status))
	}

	// POST-VERIFY PENDING CHECK 2026-05-18 — chống FALSE POSITIVE cho api mfb.
	// Picture endpoint vẫn trả ảnh kể cả account ở pending/checkpoint state.
	// Gọi thêm GET m.facebook.com/ với cookie account → nếu redirect /confirmemail.php
	// hoặc /checkpoint/ → account VẪN PENDING → demote "Live" → "Die".
	if status == "Live" && session.Cookie != "" {
		if pendingURL := detectPendingOrCheckpointWeb(ctx, session.Proxy, session.Cookie, session.UserAgent); pendingURL != "" {
			notify(fmt.Sprintf("[WebVerify] FALSE POSITIVE — account vẫn ở %s → demote Live → Die", pendingURL))
			status = "Die"
		}
	}

	savePath := outputPath
	if savePath == "" {
		savePath = cfg.OutputPath
	}
	if savePath != "" {
		SaveAccountToFolder(savePath, status, session.InputAccount, tempEmail)
		emailInfo := ""
		if tempEmail != "" {
			emailInfo = " | mail: " + tempEmail
		}
		notify(fmt.Sprintf("[Save] Đã lưu %s%s", status, emailInfo))
	}

	return &Result{
		Success: true,
		Message: fmt.Sprintf("%s — Email: %s", status, tempEmail),
		Status:  status,
		Email:   tempEmail,
	}
}

// createTempEmail tạo một địa chỉ email tạm thời, retry tối đa 3 lần nếu thất bại.
// Dừng sớm nếu ctx bị hủy giữa các lần thử.
// onEmailCreated: optional callback — gọi khi tạo email thành công để UI update realtime.
func createTempEmail(ctx context.Context, emailSvc email.Service, notify func(string), onEmailCreated func(string)) (string, error) {
	var tempEmail string
	var emailErr error
	for i := 1; i <= 3; i++ {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("Đã dừng")
		default:
		}
		notify(fmt.Sprintf("[Mail] Tạo email lần %d/3...", i))
		tempEmail, emailErr = emailSvc.CreateEmail(ctx)
		if tempEmail != "" {
			// Emit email lên UI ngay khi tạo xong → cột EMAIL/PHONE hiện realtime
			if onEmailCreated != nil {
				onEmailCreated(tempEmail)
			}
			notify(fmt.Sprintf("[Mail] Tạo thành công: %s", tempEmail))
			return tempEmail, nil
		}
		notify(fmt.Sprintf("[Mail] Thất bại lần %d: %v", i, emailErr))
	}
	return "", fmt.Errorf("Tạo email thất bại sau 3 lần")
}

// runB1toB5 chạy toàn bộ waterfall xác thực email: B1 → B2 → B3 → B4 → chờ OTP → B5.
//
// Trả về (failedAt, error):
//   - failedAt = ""      — thành công toàn bộ waterfall
//   - failedAt = "api"   — lỗi ở một trong các bước B1/B2/B3/B4/B5 (API Facebook từ chối)
//   - failedAt = "otp"   — poll OTP hết thời gian, không nhận được mã (cần email mới lần sau)
//   - failedAt = "email" — hiện không dùng trong runB1toB5, được đặt bởi caller khi createTempEmail fail
func runB1toB5(ctx context.Context, session *instagram.Session, cfg *Config, emailSvc email.Service, notify func(string), tempEmail string) (string, error) {
	waterfallId := generateUUID()

	// B1
	notify(fmt.Sprintf("[B1] SelectMail cho %s...", session.UID))
	if err := SelectMail(ctx, session, waterfallId); err != nil {
		return "api", fmt.Errorf("B1 SelectMail thất bại: %s", err)
	}
	notify("[B1] OK")

	// B2
	select {
	case <-ctx.Done():
		return "api", fmt.Errorf("Đã dừng")
	default:
	}
	notify("[B2] ChangeEmail...")
	if err := ChangeEmail(ctx, session, waterfallId); err != nil {
		return "api", fmt.Errorf("B2 ChangeEmail thất bại: %s", err)
	}
	notify("[B2] OK")

	// B3
	notify("[B3] SubmitEmail...")
	b3Resp, b3Err := SubmitEmail(ctx, session, waterfallId, tempEmail)
	if b3Resp != "" {
		notify(fmt.Sprintf("[B3] Response: %s", b3Resp))
	}
	if b3Err != nil {
		return "api", fmt.Errorf("B3 SubmitEmail thất bại: %s", b3Err)
	}
	notify("[B3] OK — Facebook đã gửi OTP tới " + tempEmail)

	// B4
	notify("[B4] LoadConfirmation...")
	b4Resp, b4Err := LoadConfirmation(ctx, session, waterfallId, tempEmail)
	if b4Resp != "" {
		notify(fmt.Sprintf("[B4] Response: %s", b4Resp))
	}
	if b4Err != nil {
		return "api", fmt.Errorf("B4 LoadConfirmation thất bại: %s", b4Err)
	}
	notify("[B4] OK")

	// Poll OTP — cho phép resend theo cfg.MaxResend, cap tối đa 2 lần để tránh spam FB.
	// Port C# MainFormUISettings.TrySendCode (default 2).
	maxResend := cfg.MaxResend
	if maxResend < 0 {
		maxResend = 0
	}
	if maxResend > 2 {
		maxResend = 2 // cap cứng — resend quá nhiều FB sẽ block
	}
	pollMs := cfg.WaitMailMs
	if pollMs <= 0 {
		pollMs = 5000 // default 5000ms giống WeBM — Facebook thường mất 30-50s gửi mail
	}
	waitSec := cfg.TimeDelaySendCode
	if waitSec <= 0 {
		waitSec = 30
	}
	pollRetry := waitSec * 1000 / pollMs
	if pollRetry < 1 {
		pollRetry = 1
	}
	var otpCode string

	for resendAttempt := 0; resendAttempt <= maxResend; resendAttempt++ {
		select {
		case <-ctx.Done():
			return "otp", fmt.Errorf("Đã dừng")
		default:
		}

		notify(fmt.Sprintf("[Mail] Chờ OTP từ %s — lượt chờ %d/%d...", tempEmail, resendAttempt+1, maxResend+1))
		stopHB := verifybase.StartOTPHeartbeat(ctx, notify, 5*time.Second, "[Mail]", tempEmail)
		code, pollErr := emailSvc.WaitForCode(ctx, pollRetry, pollMs)
		stopHB()
		if pollErr == nil && code != "" {
			otpCode = code
			notify(fmt.Sprintf("[Mail] OTP: %s", otpCode))
			break
		}
		notify(fmt.Sprintf("[Mail] Poll kết thúc lượt %d: %v", resendAttempt+1, pollErr))

		if resendAttempt < maxResend {
			notify(fmt.Sprintf("[Mail] Chưa nhận được OTP, gửi lại mã (%d/%d)...", resendAttempt+1, maxResend))
			if err := ResendConfirmationCode(ctx, session, waterfallId, tempEmail); err != nil {
				notify(fmt.Sprintf("[Resend] Thất bại: %v", err))
			}
		}
	}

	if otpCode == "" {
		return "otp", fmt.Errorf("Không nhận được OTP từ %s", tempEmail)
	}

	// B5
	select {
	case <-ctx.Done():
		return "api", fmt.Errorf("Đã dừng")
	default:
	}
	notify(fmt.Sprintf("[B5] ConfirmOTP [%s]...", otpCode))
	if err := ConfirmOTP(ctx, session, waterfallId, tempEmail, otpCode); err != nil {
		return "api", fmt.Errorf("B5 ConfirmOTP thất bại: %s", err)
	}
	notify("[B5] Verify thành công!")

	return "", nil
}

// truncStr cắt ngắn chuỗi s về tối đa n ký tự đầu.
// Không thêm "..." — chỉ cắt thô, đủ để nhận dạng trong log.
func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// detectPendingOrCheckpointWeb — GET m.facebook.com/ với cookie account.
// Nếu FB redirect về /confirmemail.php hoặc /checkpoint/ → account vẫn pending/checkpoint
// (verify chưa thực sự hoàn tất). Trả về string mô tả state để log.
// Trả "" nếu account healthy (không redirect vào endpoint nghi ngờ).
//
// Dùng client through proxy của session (tránh leak IP). Timeout 10s — nhanh + không block.
func detectPendingOrCheckpointWeb(ctx context.Context, proxyStr, cookie, ua string) string {
	if cookie == "" {
		return ""
	}
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client := proxy.CreateClient(proxyStr, 10*time.Second)
	defer client.CloseIdleConnections()

	if ua == "" {
		ua = defaultChromeDesktopUA
	}

	req, err := http.NewRequestWithContext(checkCtx, "GET", "https://m.facebook.com/", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Cookie", cookie)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9,vi-VN;q=0.8,vi;q=0.7")

	resp, err := client.Do(req)
	if err != nil {
		return "" // network error → không demote (giữ Live), CheckLive đã làm việc rồi
	}
	defer resp.Body.Close()

	finalURL := ""
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	lowURL := strings.ToLower(finalURL)
	switch {
	case strings.Contains(lowURL, "/confirmemail.php"):
		return "confirmemail.php (pending email confirmation)"
	case strings.Contains(lowURL, "/checkpoint/"):
		return "checkpoint (account locked)"
	// login.php KHÔNG dùng để demote — có thể là session expire hoặc cookie thiếu
	// navigator cookies, không liên quan đến verify thành công hay không.
	}
	return ""
}

// ─── Live/Die check + save result ────────────────────────────────────────────
//
// Mapping từ WeBM lines 885-955.

var saveLock sync.Mutex

// liveDieClient — shared client cho CheckLiveDieAccount, tránh tạo Transport mới mỗi lần gọi.
var liveDieClient = &http.Client{
	Timeout: 8 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	},
}

// CheckLiveDieAccount kiểm tra trạng thái tài khoản Facebook qua Graph API.
// Mapping từ WeBM CheckLiveDieAccount() lines 885-910.
//
// Gọi endpoint picture của Graph API không cần access token — Facebook trả ảnh
// nếu account còn sống, trả ảnh default hoặc thiếu field "height" nếu đã die/bị khóa.
//
// ctx: context để hủy request nếu user dừng hoặc timeout.
// uid: Facebook User ID (dạng số nguyên, ví dụ "100012345678").
//
// Trả về:
//   - "Live"    — account còn hoạt động (response có field "height")
//   - "Die"     — account die hoặc bị khóa (ảnh default /C5yt7Cqf3zU.jpg hoặc không có "height")
//   - "Unknown" — lỗi network, timeout, hoặc body rỗng
func CheckLiveDieAccount(ctx context.Context, uid string) string {
	url := "https://graph.facebook.com/" + uid + "/picture?type=normal&redirect=false"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "Unknown"
	}

	resp, err := liveDieClient.Do(req)
	if err != nil {
		return "Unknown"
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "Unknown"
	}

	ret := string(body)
	if ret == "" {
		return "Unknown"
	}

	// /C5yt7Cqf3zU.jpg = default dead account image
	// Không có "height" = không có ảnh = die
	if strings.Contains(ret, "/C5yt7Cqf3zU.jpg") || !strings.Contains(ret, "height") {
		return "Die"
	}

	return "Live"
}

// SaveAccountToFolder lưu thông tin account vào file tương ứng theo trạng thái.
// Mapping từ WeBM SaveAccountToFolder() lines 916-955.
// Thread-safe: dùng saveLock mutex, an toàn khi nhiều goroutine ghi đồng thời.
//
// outputPath: thư mục đích — sẽ được tạo tự động nếu chưa tồn tại.
// status: trạng thái account — "die" → Die.txt, default → Unknown.txt.
// inputAccount: chuỗi account gốc (dạng user|pass|... hoặc cookie string).
//
//	Dấu "|NVR" sẽ bị loại bỏ trước khi ghi. Nếu rỗng thì fallback ghi email.
//
// email: fallback khi inputAccount rỗng.
//
// Lưu ý: Live accounts KHÔNG ghi ở đây — saveVerifyOutcome (app.go) chịu trách nhiệm
// ghi SuccessVerify_No2FA.txt qua OnAccountDone, bao gồm email đã thêm thành công.
// Nếu ghi cả hai nơi sẽ gây trùng dữ liệu (2 dòng cho 1 account).
func SaveAccountToFolder(outputPath, status, inputAccount, email string) {
	status = strings.TrimSpace(status)
	// Live do saveVerifyOutcome (app.go → OnAccountDone) ghi vào SuccessVerify_No2FA.txt.
	// SaveAccountToFolder chỉ xử lý Die và Unknown để tránh ghi trùng.
	var fileName string
	switch strings.ToLower(status) {
	case "live":
		return
	case "die":
		fileName = "Die.txt"
	default:
		fileName = "Unknown.txt"
	}

	// Tạo thư mục nếu chưa có
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return
	}
	filePath := filepath.Join(outputPath, fileName)

	// Chuẩn bị dòng ghi
	line := strings.ReplaceAll(inputAccount, "|NVR", "")
	if line == "" {
		line = email
	}

	// Thread-safe write
	saveLock.Lock()
	defer saveLock.Unlock()

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	if _, err := f.WriteString(line + "\n"); err != nil {
		slog.Warn("SaveAccountToFolder: ghi file thất bại", "file", filePath, "err", err)
	}
}
