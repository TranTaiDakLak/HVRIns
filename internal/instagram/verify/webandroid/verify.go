// Package webandroid — Facebook Web Android Chrome verify
// Mapping từ C#: FacebookVerifyWebAndroidAPI
// Flow: GET m.facebook.com/changeemail → POST setemail → wait OTP → POST confirmation_cliff
package webandroid

import (
	"context"
	"fmt"
	"strings"
	"time"

	"HVRIns/internal/email"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/instagram/verify/verifybase"
)

// RandomUA exposes a random Chrome Android UA — đăng ký với verify registry
// để pickUAForVerifyPlatform() có thể fallback sang đây thay vì hardcode default.
// Mỗi VER account dùng UA Chrome Android random từ pool ~116k combinations
// (2190 devices × 53 Chrome versions × 5 OS versions × random viewport/dpr).
func RandomUA(countryCode string) string {
	_ = countryCode // WebAndroid không cần country-specific UA (FB không kiểm tra FBCR cho Chrome)
	return fakeinfo.RandomChromeAndroidProfile().UserAgent
}

// isValidChromeMobileUA kiểm tra UA có phải Chrome Mobile format hay không.
// Webandroid endpoint /m.facebook.com BẮT BUỘC UA Chrome Mobile, không phải FB4A.
func isValidChromeMobileUA(ua string) bool {
	return strings.Contains(ua, "Chrome") && strings.Contains(ua, "Mobile")
}

// Verifier implements instagram.Verifier for the Web Android Chrome platform.
type Verifier struct{}

// Verify thực hiện xác minh email cho account Facebook qua Web Android Chrome flow.
// Input: session cần có Cookie + UID (token optional cho CheckLiveDie).
func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(session.UID, msg)
		}
	}

	if session.Cookie == "" {
		notify("[WebAndroid] ERROR: Missing cookie")
		return &instagram.VerifyResult{Status: "error", Message: "Missing cookie"}
	}

	// Normalize session.UserAgent — WebAndroid bắt buộc Chrome Mobile UA cho mọi request.
	// Nếu pickUAForVerifyPlatform fallback "" hoặc gửi UA Android (FB4A) sai loại,
	// thì generate random Chrome Android UA tại đây để toàn bộ flow (addEmail +
	// confirmEmail + Enable2FA) dùng CÙNG 1 UA — tránh fingerprint inconsistency
	// giữa các request trong cùng verify session.
	if !isValidChromeMobileUA(session.UserAgent) {
		newUA := RandomUA("")
		notify(fmt.Sprintf("[WebAndroid] UA invalid/empty — generated random Chrome UA: %.60s...", newUA))
		session.UserAgent = newUA
	}

	notify(fmt.Sprintf("[WebAndroid] Starting... UID=%s | MailProvider=%s",
		session.UID, cfg.MailProvider))

	// Advanced options: FmUserTmpMail + UseProxyTempMail
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

	// === Step 1: Tạo email ===
	emailSvc, err := email.New(email.Options{
		Provider:           cfg.MailProvider,
		ProxyStr:           session.Proxy,
		ProxyOverride:      proxyOverride,
		CustomUsername:     customUsername,
		OnStatus:           func(msg string) { notify(msg) },
		Pool:               cfg.EmailPool,
		ZeusXApiKey:        cfg.ZeusXApiKey,
		ZeusXAccountCode:   cfg.ZeusXAccountCode,
		DvfbApiKey:         cfg.DvfbApiKey,
		DvfbAccountType:    cfg.DvfbAccountType,
		Store1sApiKey:      cfg.Store1sApiKey,
		Store1sProductID:   cfg.Store1sProductID,
		Mail30sApiKey:      cfg.Mail30sApiKey,
		Mail30sProductSlug: cfg.Mail30sProductSlug,
		TempMailLolApiKey:    cfg.TempMailLolApiKey,
		TempMailDomain:       cfg.TempMailDomain,
		MuaMailApiKey:        cfg.MuaMailApiKey,
		MuaMailProductID:     cfg.MuaMailProductID,
		UnlimitMailApiKey:    cfg.UnlimitMailApiKey,
		UnlimitMailProductID: cfg.UnlimitMailProductID,
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
		TempMailToken:           cfg.TempMailToken,
	})
	if err != nil {
		notify(fmt.Sprintf("[WebAndroid] Email service ERROR: %v", err))
		return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Email service: %v", err)}
	}
	defer emailSvc.Close()

	// TempMail reuse: nếu reg đã tạo mail tạm + lưu creds → Restore + skip
	// CreateEmail. AddEmail step (line ~113) vẫn chạy vì cần `state` cho
	// confirmEmail downstream — FB webandroid API có thể accept duplicate
	// add cho cùng email đã linked, hoặc trả error → fall back xử lý qua
	// existing error handling.
	var tempEmail string
	if session.Email != "" && session.EmailMeta != "" && email.RestoreIfPossible(emailSvc, session.EmailMeta) {
		tempEmail = session.Email
		notify(fmt.Sprintf("[WebAndroid] Reuse mail từ register: %s (skip CreateEmail)", tempEmail))
		if cfg.OnEmailCreated != nil {
			cfg.OnEmailCreated(tempEmail)
		}
	} else {
		var err error
		tempEmail, err = retryCreateEmail(ctx, emailSvc, notify)
		if err != nil {
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Create email: %v", err)}
		}
		// Emit email lên UI ngay khi tạo xong → cột EMAIL/PHONE hiện realtime
		if cfg.OnEmailCreated != nil {
			cfg.OnEmailCreated(tempEmail)
		}
		notify(fmt.Sprintf("[WebAndroid] Email: %s (provider=%s)", tempEmail, cfg.MailProvider))
	}

	// === Step 2: addEmail — GET changeemail + POST setemail ===
	notify(fmt.Sprintf("[WebAndroid] Add email %s [provider=%s]...", tempEmail, cfg.MailProvider))
	state, err := addEmailWithNotify(ctx, session.Proxy, session.Cookie, session.UID, session.UserAgent, tempEmail, notify)
	if err != nil {
		notify(fmt.Sprintf("[WebAndroid] Add email FAIL: %v", err))
		return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Add email: %v", err), Email: tempEmail}
	}
	notify("[WebAndroid] Email added — waiting OTP...")

	// === Step 3: Chờ OTP ===
	waitSec := cfg.TimeDelaySendCode
	if waitSec <= 0 {
		waitSec = 30
	}
	maxRetry := waitSec * 1000 / 3000
	if maxRetry < 1 {
		maxRetry = 1
	}

	stopHB := verifybase.StartOTPHeartbeat(ctx, notify, 5*time.Second, "[WebAndroid]", emailSvc.GetEmail())
	code, err := emailSvc.WaitForCode(ctx, maxRetry, 3000)
	stopHB()
	if err != nil {
		if !cfg.SendAgainCode {
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("OTP timeout: %v", err), Email: tempEmail}
		}
		notify("[WebAndroid] OTP timeout — no resend available for WebAndroid flow")
		return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("OTP timeout: %v", err), Email: tempEmail}
	}
	notify(fmt.Sprintf("[WebAndroid] OTP: %s", code))

	// Delay giữa nhận code và confirm
	if cfg.DelayConfirmEmail > 0 {
		notify(fmt.Sprintf("[WebAndroid] Chờ %ds trước confirm...", cfg.DelayConfirmEmail))
		select {
		case <-ctx.Done():
			return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail}
		case <-time.After(time.Duration(cfg.DelayConfirmEmail) * time.Second):
		}
	}

	// === Step 4: Confirm code ===
	notify(fmt.Sprintf("[WebAndroid] Nhập OTP %s → confirm email...", code))
	if err := confirmEmail(ctx, session.Proxy, session.Cookie, session.UID, session.UserAgent, tempEmail, code, state); err != nil {
		notify(fmt.Sprintf("[WebAndroid] Confirm ERROR: %v", err))
		if contains(err.Error(), "checkpoint") {
			return &instagram.VerifyResult{Status: "Die", Message: "Checkpoint after confirm", Email: tempEmail}
		}
		return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Confirm: %v", err), Email: tempEmail}
	}
	notify("[WebAndroid] Email confirmed!")

	// === Enable 2FA via AccountsCenter Chrome Android API ===
	// Port C# FacebookSecurityFeatureAPI.TurnOnTwofactor — bật 2FA TOTP.
	//   - If FB triggers reauth → sends OTP to confirmed email → emailSvc retrieves it
	var twoFAKey string
	if cfg.Enable2FA {
		notify("[WebAndroid] Enabling 2FA via AccountsCenter...")
		emailOTPFn := func(maskedEmail string, _ int) string {
			notify(fmt.Sprintf("[WebAndroid] 2FA reauth — waiting OTP for %s...", maskedEmail))
			c, e := emailSvc.WaitForCode(ctx, 3, 3000) // cap 3 lần poll
			if e != nil {
				notify(fmt.Sprintf("[WebAndroid] 2FA reauth OTP timeout: %v", e))
				return ""
			}
			return c
		}
		var err2fa error
		twoFAKey, err2fa = Enable2FA(ctx,
			session.Proxy, session.Cookie, session.UID, session.UserAgent,
			session.Password, tempEmail,
			emailOTPFn, notify)
		if err2fa != nil {
			// 2FA failure is non-fatal — account is already verified
			notify(fmt.Sprintf("[WebAndroid] 2FA enable failed (non-fatal): %v", err2fa))
		} else {
			notify(fmt.Sprintf("[WebAndroid] 2FA enabled — secret: %s", twoFAKey))
		}
	}

	// === Step 5: CheckLiveDie — periodic every 5s, bail early on die ===
	checkDelay := cfg.TimeDelayCheck
	if checkDelay <= 0 {
		checkDelay = 5
	}
	status := "Live"
	if cfg.CheckLiveDie {
		checkInterval := 5
		notify(fmt.Sprintf("[WebAndroid] Checking live/die every %ds for %ds total...", checkInterval, checkDelay))
		elapsed := 0
		for elapsed < checkDelay {
			wait := checkInterval
			if elapsed+wait > checkDelay {
				wait = checkDelay - elapsed
			}
			select {
			case <-ctx.Done():
				return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail}
			case <-time.After(time.Duration(wait) * time.Second):
			}
			elapsed += wait
			notify(fmt.Sprintf("[WebAndroid] [%ds/%ds] Checking live/die UID=%s...", elapsed, checkDelay, session.UID))
			s := verifybase.CheckLiveDieCombined(ctx, session.UserAgent, session.UID, session.Token)
			if s == "Die" {
				status = "Die"
				break
			}
		}
		notify(fmt.Sprintf("[WebAndroid] Check result: %s", status))

		// POST-VERIFY PENDING CHECK 2026-05-18 — chống FALSE POSITIVE.
		// Picture endpoint vẫn trả ảnh kể cả account ở pending/checkpoint state.
		// Gọi thêm GET m.facebook.com/ với cookie account → nếu redirect /confirmemail.php
		// hoặc /checkpoint/ → account VẪN PENDING → demote "Live" → "Die".
		if status == "Live" && session.Cookie != "" {
			if pendingURL := detectPendingOrCheckpoint(ctx, session.Proxy, session.Cookie, session.UserAgent); pendingURL != "" {
				notify(fmt.Sprintf("[WebAndroid] FALSE POSITIVE detected — account vẫn ở %s → demote Live → Die", pendingURL))
				status = "Die"
			}
		}
	} else {
		notify(fmt.Sprintf("[WebAndroid] Wait %ds (check live disabled)...", checkDelay))
		select {
		case <-ctx.Done():
			return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail}
		case <-time.After(time.Duration(checkDelay) * time.Second):
		}
	}

	msg := fmt.Sprintf("%s — Email: %s", status, tempEmail)
	if twoFAKey != "" {
		msg = fmt.Sprintf("%s — Email: %s — 2FA: %s", status, tempEmail, twoFAKey)
	}
	notify(fmt.Sprintf("[WebAndroid] Done: %s — %s", status, tempEmail))
	return &instagram.VerifyResult{
		Success: status == "Live",
		Status:  status,
		Message: msg,
		Email:   tempEmail,
		TwoFA:   twoFAKey,
	}
}

// detectPendingOrCheckpoint — GET m.facebook.com/ với cookie account qua tls-client
// Dùng tls-client (Chrome JA3) — match với flow chính. net/http bị FB RST.
//
// Chỉ demote Live → Die khi: confirmemail.php (email chưa confirm) hoặc /checkpoint/
// (account bị lock). login.php KHÔNG dùng để demote — session expire/missing nav
// cookies không liên quan đến verify thành công hay không.
func detectPendingOrCheckpoint(ctx context.Context, proxyStr, cookie, ua string) string {
	if cookie == "" {
		return ""
	}
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if ua == "" {
		ua = defaultChromeAndroidUA
	}
	chromeMajor := extractChromeMajor(ua)
	client, err := createTLSClient(proxyStr, chromeMajor)
	if err != nil {
		return "" // không tạo được client → không demote
	}
	defer client.CloseIdleConnections()

	// Seed cookie vào jar
	seedCookieStringTLS(client, cookie)

	h := navHeaders(ua)
	_, finalURL, err := doGetTLS(checkCtx, client, "https://m.facebook.com/", h, nil)
	if err != nil {
		return "" // network error → không demote
	}

	lowURL := strings.ToLower(finalURL)
	switch {
	case strings.Contains(lowURL, "/confirmemail.php"):
		return "confirmemail.php (pending email confirmation)"
	case strings.Contains(lowURL, "/checkpoint/"):
		return "checkpoint (account locked)"
	// login.php KHÔNG dùng để demote — không liên quan đến verify thành công.
	}
	return ""
}

// retryCreateEmail thử tạo email tối đa 3 lần.
func retryCreateEmail(ctx context.Context, svc email.Service, notify func(string)) (string, error) {
	for i := 1; i <= 3; i++ {
		if ctx.Err() != nil {
			return "", fmt.Errorf("cancelled")
		}
		if i > 1 {
			notify(fmt.Sprintf("[Mail] Retry %d/3...", i))
			select {
			case <-ctx.Done():
				return "", fmt.Errorf("cancelled")
			case <-time.After(2 * time.Second):
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

func contains(s, sub string) bool {
	return len(s) > 0 && len(sub) > 0 && (s == sub || len(s) >= len(sub) && func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}())
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformWebAndroid, func() instagram.Verifier {
		return &Verifier{}
	})
}
