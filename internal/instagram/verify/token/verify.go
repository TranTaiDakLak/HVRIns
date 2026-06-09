// Package token — Facebook Token API verify (port C# FacebookVerifyAPIToken).
// Flow:
//   1. Yêu cầu session.Token (EAA... access_token) sẵn — KHÔNG tự login bằng uid/password.
//   2. Tạo temp email qua email.Service.
//   3. AddEmail: POST api.facebook.com/method/user.editregistrationcontactpoint.
//   4. Poll OTP từ email.
//   5. ConfirmEmail: POST api.facebook.com/method/user.confirmcontactpoint.
//   6. Optional CheckLiveDie qua graph.facebook.com/{uid}/picture.
package token

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"HVRIns/internal/email"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/instagram/verify/verifybase"
	"HVRIns/internal/proxy"
)

const (
	addEmailURL     = "https://api.facebook.com/method/user.editregistrationcontactpoint"
	confirmEmailURL = "https://api.facebook.com/method/user.confirmcontactpoint"
	graphPictureURL = "https://graph.facebook.com/%s/picture?type=normal&redirect=false"
)

// defaultFallbackUA trả UA từ Android pool — tránh hardcode.
func defaultFallbackUA() string {
	if ua := fakeinfo.RandomUAFromPool(fakeinfo.UAKindAndroid); ua != "" {
		return ua
	}
	return fakeinfo.BuildAndroidUAWithOpts(fakeinfo.RandomDeviceProfile(), "en_US", "", "", "", false, false)
}

// Verifier implements instagram.Verifier via Facebook Graph/Method API với access_token.
type Verifier struct{}

// Verify performs email verification using Token API.
func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(session.UID, msg)
		}
	}

	if session.Token == "" {
		notify("[Token] ERROR: Missing access_token (session.Token)")
		return &instagram.VerifyResult{Status: "error", Message: "Token API verify yêu cầu access_token — session.Token rỗng"}
	}
	if session.UID == "" {
		notify("[Token] ERROR: Missing UID")
		return &instagram.VerifyResult{Status: "error", Message: "Missing UID"}
	}

	notify(fmt.Sprintf("[Token] Starting... UID=%s | MailProvider=%s", session.UID, cfg.MailProvider))

	// Advanced options parity với WebAndroid
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
		Provider:                cfg.MailProvider,
		ProxyStr:                session.Proxy,
		ProxyOverride:           proxyOverride,
		CustomUsername:          customUsername,
		OnStatus:                notify,
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
		notify(fmt.Sprintf("[Token] Email service ERROR: %v", err))
		return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Email service: %v", err)}
	}
	defer emailSvc.Close()

	// TempMail reuse: nếu reg đã tạo mail tạm + lưu creds → Restore + skip
	// CreateEmail + skip AddEmail. Xem comment chi tiết ở s23/steps.go.
	var tempEmail string
	reuseMail := false
	if session.Email != "" && session.EmailMeta != "" && email.RestoreIfPossible(emailSvc, session.EmailMeta) {
		tempEmail = session.Email
		reuseMail = true
		notify(fmt.Sprintf("[Token] Reuse mail từ register: %s (skip CreateEmail+AddEmail)", tempEmail))
		if cfg.OnEmailCreated != nil {
			cfg.OnEmailCreated(tempEmail)
		}
	} else {
		var err error
		tempEmail, err = emailSvc.CreateEmail(ctx)
		if err != nil || tempEmail == "" {
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Create email: %v", err)}
		}
		// Emit email lên UI ngay khi tạo xong → cột EMAIL/PHONE hiện realtime
		if cfg.OnEmailCreated != nil {
			cfg.OnEmailCreated(tempEmail)
		}
		notify(fmt.Sprintf("[Token] Email: %s (provider=%s)", tempEmail, cfg.MailProvider))
	}

	client := proxy.CreateClient(session.Proxy, 30*time.Second)
	locale := "en_US"
	if cfg.DeepFakeLocale {
		locale = "vi_VN" // fallback khi DeepFakeLocale bật — Session không có Locale field
	}

	// === Step 2: AddEmail qua Graph method API ===
	// TempMail reuse: skip nếu mail đã được dùng làm contactpoint khi reg.
	if !reuseMail {
		notify(fmt.Sprintf("[Token] Add email %s...", tempEmail))
		if err := addEmail(ctx, client, session.Token, tempEmail, locale, session.UserAgent); err != nil {
			notify(fmt.Sprintf("[Token] Add email FAIL: %v", err))
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Add email: %v", err), Email: tempEmail}
		}
		notify("[Token] Email added — waiting OTP...")
	} else {
		notify("[Token] Skip AddEmail (reuse mail từ reg) — chờ OTP từ inbox...")
	}

	// === Step 3: Chờ OTP ===
	waitSec := cfg.TimeDelaySendCode
	if waitSec <= 0 {
		waitSec = 30
	}
	maxRetry := waitSec * 1000 / 3000
	if maxRetry < 1 {
		maxRetry = 1
	}

	stopHB := verifybase.StartOTPHeartbeat(ctx, notify, 5*time.Second, "[Token]", emailSvc.GetEmail())
	code, err := emailSvc.WaitForCode(ctx, maxRetry, 3000)
	stopHB()
	if err != nil {
		return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("OTP timeout: %v", err), Email: tempEmail}
	}
	notify(fmt.Sprintf("[Token] OTP: %s", code))

	// Delay giữa nhận code và confirm
	if cfg.DelayConfirmEmail > 0 {
		notify(fmt.Sprintf("[Token] Chờ %ds trước confirm...", cfg.DelayConfirmEmail))
		select {
		case <-ctx.Done():
			return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail}
		case <-time.After(time.Duration(cfg.DelayConfirmEmail) * time.Second):
		}
	}

	// === Step 4: ConfirmEmail ===
	notify(fmt.Sprintf("[Token] Nhập OTP %s → confirm email...", code))
	if err := confirmCode(ctx, client, session.Token, tempEmail, code, locale, session.UserAgent); err != nil {
		notify(fmt.Sprintf("[Token] Confirm ERROR: %v", err))
		if strings.Contains(err.Error(), "checkpoint") {
			return &instagram.VerifyResult{Status: "Die", Message: "Checkpoint after confirm", Email: tempEmail}
		}
		return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Confirm: %v", err), Email: tempEmail}
	}
	notify("[Token] Email confirmed!")

	// === Step 5: CheckLiveDie — periodic every 5s, bail early on die ===
	checkDelay := cfg.TimeDelayCheck
	if checkDelay <= 0 {
		checkDelay = 5
	}
	status := "Live"
	if cfg.CheckLiveDie {
		checkInterval := 5
		notify(fmt.Sprintf("[Token] Checking live/die every %ds for %ds total...", checkInterval, checkDelay))
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
			notify(fmt.Sprintf("[Token] [%ds/%ds] Checking live/die UID=%s...", elapsed, checkDelay, session.UID))
			// Combined check — token /me catch checkpoint NGAY (picture endpoint delay 30-60p).
			s := verifybase.CheckLiveDieCombined(ctx, session.UserAgent, session.UID, session.Token)
			if s == "Die" {
				status = "Die"
				break
			}
		}
		notify(fmt.Sprintf("[Token] Check result: %s", status))
	} else {
		notify(fmt.Sprintf("[Token] Wait %ds (check live disabled)...", checkDelay))
		select {
		case <-ctx.Done():
			return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail}
		case <-time.After(time.Duration(checkDelay) * time.Second):
		}
	}

	notify(fmt.Sprintf("[Token] Done: %s — %s", status, tempEmail))
	return &instagram.VerifyResult{
		Success: status == "Live",
		Status:  status,
		Message: fmt.Sprintf("%s — Email: %s", status, tempEmail),
		Email:   tempEmail,
	}
}

// addEmail gọi POST user.editregistrationcontactpoint.
// Payload: access_token, contactpoint=email, locale, format=json.
func addEmail(ctx context.Context, client *http.Client, token, email, locale, userAgent string) error {
	form := url.Values{}
	form.Set("access_token", token)
	form.Set("contactpoint", email)
	form.Set("contactpoint_type", "email")
	form.Set("locale", locale)
	form.Set("format", "json")

	req, _ := http.NewRequestWithContext(ctx, "POST", addEmailURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	} else {
		req.Header.Set("User-Agent", defaultFallbackUA())
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	return classifyAddEmailResp(resp, string(body))
}

// confirmCode gọi POST user.confirmcontactpoint với OTP.
func confirmCode(ctx context.Context, client *http.Client, token, email, code, locale, userAgent string) error {
	form := url.Values{}
	form.Set("access_token", token)
	form.Set("contactpoint", email)
	form.Set("contactpoint_type", "email")
	form.Set("code", code)
	form.Set("locale", locale)
	form.Set("format", "json")

	req, _ := http.NewRequestWithContext(ctx, "POST", confirmEmailURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	} else {
		req.Header.Set("User-Agent", defaultFallbackUA())
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	return classifyConfirmResp(resp, string(body))
}

// classifyAddEmailResp map response → error. "true" response = success.
func classifyAddEmailResp(resp *http.Response, body string) error {
	if isCheckpointHeader(resp) {
		return fmt.Errorf("checkpoint")
	}
	if strings.Contains(body, "Invalid OAuth 2.0 Access Token") {
		return fmt.Errorf("checkpoint: token invalid/die")
	}
	if strings.Contains(body, "email is already being used") ||
		strings.Contains(body, "The email address you entered is already") {
		return fmt.Errorf("email đã được dùng")
	}
	if strings.Contains(body, "Email Disabled") ||
		strings.Contains(body, "That email can") ||
		strings.Contains(body, "You have entered an invalid email") {
		return fmt.Errorf("email không hợp lệ / bị block")
	}
	// Parse JSON để check root-level `true` thực sự — tránh false positive từ chuỗi
	// "true" lọt trong field khác (ví dụ {"error": true, ...} bị classify nhầm là success).
	trimmed := strings.TrimSpace(body)
	if trimmed == "true" || trimmed == `"true"` {
		return nil
	}
	var bv bool
	if err := json.Unmarshal([]byte(trimmed), &bv); err == nil && bv {
		return nil
	}
	if body == "" {
		return fmt.Errorf("empty response — http %d", resp.StatusCode)
	}
	return fmt.Errorf("unknown addEmail response: %.200s", body)
}

// classifyConfirmResp map response → error.
func classifyConfirmResp(resp *http.Response, body string) error {
	if isCheckpointHeader(resp) {
		return fmt.Errorf("checkpoint")
	}
	lower := strings.ToLower(body)
	if strings.Contains(body, "Invalid OAuth 2.0 Access Token") {
		return fmt.Errorf("checkpoint: token invalid/die")
	}
	if strings.Contains(body, "Incorrect confirmation code") ||
		strings.Contains(body, "error_code\":3301") {
		return fmt.Errorf("wrong code")
	}
	if strings.Contains(lower, "true") {
		return nil
	}
	if body == "" {
		return fmt.Errorf("empty response — http %d", resp.StatusCode)
	}
	return fmt.Errorf("unknown confirm response: %.200s", body)
}

// isCheckpointHeader kiểm tra header X-Fb-Blocking-Checkpoint hoặc x-fb-integrity-required: checkpoint.
func isCheckpointHeader(resp *http.Response) bool {
	if resp == nil {
		return false
	}
	if resp.Header.Get("X-Fb-Blocking-Checkpoint") != "" {
		return true
	}
	if strings.EqualFold(resp.Header.Get("x-fb-integrity-required"), "checkpoint") {
		return true
	}
	return false
}

// checkLiveDiePicture dùng graph.facebook.com/{uid}/picture để xác định trạng thái account.
//
// Live    — response có field "height"
// Die     — response chứa avatar default `/C5yt7Cqf3zU.jpg` HOẶC không có "height"
// Unknown — lỗi mạng / timeout
func checkLiveDiePicture(ctx context.Context, client *http.Client, uid, ua string) string {
	if uid == "" {
		return "Unknown"
	}
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(graphPictureURL, uid), nil)
	if err != nil {
		return "Unknown"
	}
	if strings.TrimSpace(ua) == "" {
		ua = defaultFallbackUA()
	}
	req.Header.Set("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		return "Unknown"
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	text := string(body)
	if text == "" {
		return "Unknown"
	}
	if strings.Contains(text, "/C5yt7Cqf3zU.jpg") || !strings.Contains(text, "height") {
		return "Die"
	}
	return "Live"
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformToken, func() instagram.Verifier {
		return &Verifier{}
	})
}
