// Package s273 — Facebook FB4A v273 verify (add email + confirm OTP).
//
// Khác hoàn toàn s5xx Bloks/GraphQL:
//   - Endpoint: POST b-api.facebook.com/method/user.editregistrationcontactpoint (add email)
//              POST b-api.facebook.com/method/user.confirmcontactpoint (submit OTP)
//   - friendly-name: editRegistrationContactpoint + confirmContactpoint
//   - Body form-urlencoded thuần có thêm field method= (khác s399 không có)
//   - Auth: OAuth <EAA token> trong header Authorization
//   - Thiết bị: Vivo V2242A, Android 9, FBAV/273.0.0.39.123
package s273

import (
	"context"
	"fmt"
	"strings"
	"time"

	"HVRIns/internal/email"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/verify/verifybase"
)

// ─── Verifier ────────────────────────────────────────────────────────────────

type Verifier struct{}

func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	return verifyAccount(ctx, session, cfg, outputPath, onStatus)
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformS273, func() instagram.Verifier {
		return &Verifier{}
	})
	instagram.RegisterPlatformVerifyUA(instagram.PlatformS273, RandomUA)
}

const tag = "[S273 Verify]"

// ─── Orchestration ───────────────────────────────────────────────────────────

func verifyAccount(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, _ string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(session.UID, msg)
		}
	}

	if session.Token == "" {
		notify(tag + " ERROR: Missing access token!")
		return &instagram.VerifyResult{Status: "error", Message: "Missing access token"}
	}
	if session.UID == "" {
		notify(tag + " ERROR: Missing UID!")
		return &instagram.VerifyResult{Status: "error", Message: "Missing UID"}
	}

	notify(fmt.Sprintf("%s Starting... UID=%s Token=%s...",
		tag, session.UID, session.Token[:verifybase.Mmin(len(session.Token), 20)]))

	// HTTP client
	client, err := verifybase.CreateClient(session.Proxy)
	if err != nil {
		notify(fmt.Sprintf("%s Client ERROR: %v", tag, err))
		return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Create client: %v", err)}
	}
	defer client.CloseIdleConnections()

	// UA: build từ s273 pool (4 combo device cố định) + country pool theo IP (locale/carrier).
	// Giữ UA cũ nếu đã thuộc pool, ngược lại random 1 combo mới khớp country IP.
	verifyUA := session.UserAgent
	if !IsPoolUA(verifyUA) {
		country := verifybase.CountryFromPhone(session.Phone)
		verifyUA = RandomUA(country)
		label := country
		if label == "" {
			label = "default-pool"
		}
		notify(fmt.Sprintf("%s UA built from s273 device pool (country=%s)", tag, label))
	}

	// Locale + country code
	locale := "en_US"
	if cfg.DeepFakeLocale {
		if l := verifybase.ExtractLocaleFromCookie(session.Cookie); l != "" {
			locale = l
		}
	}
	countryCode := strings.ToUpper(verifybase.CountryFromPhone(session.Phone))
	if countryCode == "" {
		countryCode = "VN"
	}

	// SIM HNI cho headers (default MobiFone VN — giống pattern s399)
	simHNI := "45201"

	// ─── Email service ───────────────────────────────────────────────────────
	customUsername := ""
	if cfg.FmUserTmpMail {
		login := session.Phone
		if login == "" {
			login = session.UID
		}
		customUsername = email.CreateUsernameFromLogin(login)
	}
	proxyOverride := ""
	if email.IsRentMailProvider(cfg.MailProvider) {
		if cfg.UseProxyGmail {
			proxyOverride = email.PickRentMailProxy()
		}
	} else if cfg.UseProxyTempMail {
		proxyOverride = email.PickTempMailProxy()
	}

	notify(fmt.Sprintf("%s Creating email [provider=%s]...", tag, cfg.MailProvider))
	emailSvc, err := email.New(email.Options{
		Provider:                cfg.MailProvider,
		ProxyStr:                session.Proxy,
		ProxyOverride:           proxyOverride,
		CustomUsername:          customUsername,
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
		notify(fmt.Sprintf("%s Email service ERROR: %v", tag, err))
		return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Email service: %v", err), UserAgent: verifyUA}
	}
	defer emailSvc.Close()
	notify(tag + " Email service OK")

	// TempMail reuse từ register (nếu có)
	var tempEmail string
	reuseMail := false
	if session.Email != "" && session.EmailMeta != "" && email.RestoreIfPossible(emailSvc, session.EmailMeta) {
		tempEmail = session.Email
		reuseMail = true
		notify(fmt.Sprintf("%s [REUSE] Reuse mail from register: %s (skip CreateEmail+AddEmail)", tag, tempEmail))
		if cfg.OnEmailCreated != nil {
			cfg.OnEmailCreated(tempEmail)
		}
	} else {
		tempEmail, err = verifybase.RetryCreateEmail(ctx, emailSvc, notify)
		if tempEmail != "" && cfg.OnEmailCreated != nil {
			cfg.OnEmailCreated(tempEmail)
		}
		if err != nil {
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Create email: %v", err), UserAgent: verifyUA}
		}
		notify(fmt.Sprintf("%s Email: %s", tag, tempEmail))
	}

	// ─── Step 1: Add email (POST /method/user.editregistrationcontactpoint) ──
	if !reuseMail {
		notify(fmt.Sprintf("%s Adding email %s...", tag, tempEmail))
		addBody := buildAddEmailBody(tempEmail, locale, countryCode)
		addHeaders := buildVerifyHeaders(verifyUA, session.Token, simHNI, addEmailFriendly)

		addCtx, addCancel := context.WithTimeout(ctx, 30*time.Second)
		addResp, err := verifybase.DoPost(addCtx, client, addEmailURL, addBody, addHeaders)
		addCancel()
		if err != nil {
			notify(fmt.Sprintf("%s Add email HTTP error: %v", tag, err))
			email.ReleaseIfPossible(ctx, emailSvc)
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Add email: %v", err), Email: tempEmail, UserAgent: verifyUA}
		}

		// Response v273: "true" (bare boolean) hoặc {"result":true} = success.
		// Old REST API b-api trả về bare "true", KHÔNG phải JSON object.
		addRespTrim := strings.TrimSpace(addResp)
		isAddSuccess := addRespTrim == "true" ||
			strings.Contains(addResp, `"result":true`) ||
			strings.Contains(addResp, `"result": true`)
		if isAddSuccess {
			notify(tag + " Email added OK — waiting for OTP...")
		} else {
			short := verifybase.SummarizeFBError(addResp)
			notify(tag + " Add email failed: " + short)
			email.ReleaseIfPossible(ctx, emailSvc)
			return &instagram.VerifyResult{Status: "error", Message: "Add email failed: " + short, Email: tempEmail, UserAgent: verifyUA}
		}
	} else {
		notify(tag + " Skip AddEmail (reuse mail) — waiting for OTP...")
	}

	// ─── Step 2: Wait OTP ────────────────────────────────────────────────────
	waitSec := cfg.TimeDelaySendCode
	if waitSec <= 0 {
		waitSec = 30
	}
	pollMs := cfg.WaitMailMs
	if pollMs <= 0 {
		pollMs = 5000
	}
	maxRetry := waitSec * 1000 / pollMs
	if maxRetry < 1 {
		maxRetry = 1
	}
	notify(fmt.Sprintf("%s Waiting for OTP... [mail: %s]", tag, emailSvc.GetEmail()))
	stopHB := verifybase.StartOTPHeartbeat(ctx, notify, 5*time.Second, tag, emailSvc.GetEmail())
	code, err := emailSvc.WaitForCode(ctx, maxRetry, pollMs)
	stopHB()
	if err != nil {
		if !cfg.SendAgainCode {
			email.ReleaseIfPossible(ctx, emailSvc)
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("OTP timeout: %v", err), Email: tempEmail, UserAgent: verifyUA}
		}
		notify(tag + " OTP timeout — resending code via /method/user.sendconfirmationcode...")
		resendBody := buildResendBody(tempEmail, locale, countryCode)
		resendHeaders := buildVerifyHeaders(verifyUA, session.Token, simHNI, resendFriendly)
		verifybase.DoPost(ctx, client, resendURL, resendBody, resendHeaders) //nolint:errcheck

		notify(tag + " Resent — waiting again...")
		stopHB2 := verifybase.StartOTPHeartbeat(ctx, notify, 5*time.Second, tag, emailSvc.GetEmail())
		code, err = emailSvc.WaitForCode(ctx, maxRetry, pollMs)
		stopHB2()
		if err != nil {
			email.ReleaseIfPossible(ctx, emailSvc)
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("OTP timeout after resend: %v", err), Email: tempEmail, UserAgent: verifyUA}
		}
	}
	notify(fmt.Sprintf("%s OTP: %s", tag, code))

	if cfg.DelayConfirmEmail > 0 {
		notify(fmt.Sprintf("%s Waiting %ds before confirm...", tag, cfg.DelayConfirmEmail))
		select {
		case <-ctx.Done():
			return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail, UserAgent: verifyUA}
		case <-time.After(time.Duration(cfg.DelayConfirmEmail) * time.Second):
		}
	}

	// ─── Step 3: Confirm code (POST /method/user.confirmcontactpoint) ─────────
	notify(fmt.Sprintf("%s Submitting OTP %s → confirm email...", tag, code))
	confirmBody := buildConfirmCodeBody(tempEmail, code, locale, countryCode)
	confirmHeaders := buildVerifyHeaders(verifyUA, session.Token, simHNI, confirmFriendly)

	const confirmMaxRetries = 3
	confirmOK := false
	for attempt := 1; attempt <= confirmMaxRetries; attempt++ {
		confirmResp, confirmErr := verifybase.DoPost(ctx, client, confirmURL, confirmBody, confirmHeaders)
		if confirmErr != nil {
			notify(fmt.Sprintf("%s Confirm HTTP error (attempt %d/%d): %v", tag, attempt, confirmMaxRetries, confirmErr))
			if attempt < confirmMaxRetries {
				select {
				case <-ctx.Done():
					return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail, UserAgent: verifyUA}
				case <-time.After(2 * time.Second):
				}
				continue
			}
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Confirm: %v", confirmErr), Email: tempEmail, UserAgent: verifyUA}
		}

		// Response v273 success: chỉ chấp nhận positive marker rõ ràng:
		//   - bare "true" (REST API trả bool)
		//   - {"result":true} (JSON object)
		// Mọi response khác → KHÔNG coi là success (trước đây dùng negative check
		// "vắng lỗi = success" → ghi nhầm acc chưa verify vào SuccessVerify_No2FA.txt).
		respLow := strings.ToLower(confirmResp)
		respTrim := strings.TrimSpace(confirmResp)

		// Checkpoint: cả 2 format (modern Bloks inject bởi DoPost + old REST native)
		isCheckpoint := strings.Contains(respLow, "checkpointed") ||
			strings.Contains(respLow, `"code":459`) ||
			strings.Contains(respLow, `"error_code":459`)
		if isCheckpoint {
			return &instagram.VerifyResult{Status: "Die", Message: "Checkpoint after confirm", Email: tempEmail, UserAgent: verifyUA}
		}

		// Positive success check — strict: phải có "true" hoặc "result":true
		isSuccess := respTrim == "true" ||
			respTrim == `"true"` ||
			strings.Contains(respLow, `"result":true`) ||
			strings.Contains(respLow, `"result": true`)

		if isSuccess {
			notify(tag + " Email confirmed! (response: " + truncateForLog(respTrim, 120) + ")")
			confirmOK = true
			break
		}

		// Không match success rõ ràng → fail. Log response để user debug được.
		short := verifybase.SummarizeFBError(confirmResp)
		if short == "" || short == confirmResp {
			short = "Response không phải positive success: " + truncateForLog(respTrim, 200)
		}
		notify(fmt.Sprintf("%s Confirm KHÔNG xác nhận được (attempt %d/%d): %s", tag, attempt, confirmMaxRetries, short))
		if attempt < confirmMaxRetries {
			select {
			case <-ctx.Done():
				return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail, UserAgent: verifyUA}
			case <-time.After(2 * time.Second):
			}
			continue
		}
		return &instagram.VerifyResult{Status: "error", Message: "Confirm failed: " + short, Email: tempEmail, UserAgent: verifyUA}
	}
	if !confirmOK {
		return &instagram.VerifyResult{Status: "error", Message: "Confirm failed after retries", Email: tempEmail, UserAgent: verifyUA}
	}

	// ─── Step 3.5: Post-confirm verify (Graph /me?fields=id,email) ─────────────
	// Catch 2 trường hợp confirm response không phát hiện được:
	//   1. Token checkpointed NGAY SAU confirm (error 459) — acc đã Die, không phải Live
	//   2. Confirm response lạ (HTTP 200 nhưng email không thực sự được set)
	// Đây là check NHẸ (1 GET request), luôn chạy không phụ thuộc cfg.CheckLiveDie.
	notify(tag + " Post-confirm check: verify email is actually attached...")
	emailStatus, emailDetail := postConfirmCheckEmail(ctx, client, session.Token)
	switch emailStatus {
	case "DIE":
		notify(fmt.Sprintf("%s ⚠ Token checkpointed NGAY SAU confirm — Die. %s", tag, emailDetail))
		return &instagram.VerifyResult{Status: "Die", Message: "Token checkpointed after confirm: " + emailDetail, Email: tempEmail, UserAgent: verifyUA}
	case "NO_EMAIL":
		notify(fmt.Sprintf("%s ⚠ Confirm trả OK nhưng FB không thấy email attached — confirm silent fail. %s", tag, emailDetail))
		return &instagram.VerifyResult{Status: "error", Message: "Email not attached after confirm (silent fail): " + emailDetail, Email: tempEmail, UserAgent: verifyUA}
	case "OK":
		notify(fmt.Sprintf("%s ✓ Email confirmed attached on FB (%s)", tag, emailDetail))
	default:
		// NETWORK / UNKNOWN — không fail flow, đi tiếp vào live/die check.
		notify(fmt.Sprintf("%s Post-confirm check không quyết định được: %s — đi tiếp", tag, emailDetail))
	}

	// ─── Step 4: Live/Die check ───────────────────────────────────────────────
	checkDelay := cfg.TimeDelayCheck
	if checkDelay <= 0 {
		checkDelay = 5
	}
	status := "Live"
	if cfg.CheckLiveDie {
		checkInterval := 5
		notify(fmt.Sprintf("%s Checking live/die every %ds for %ds total (UID=%s)...", tag, checkInterval, checkDelay, session.UID))
		elapsed := 0
		for elapsed < checkDelay {
			wait := checkInterval
			if elapsed+wait > checkDelay {
				wait = checkDelay - elapsed
			}
			select {
			case <-time.After(time.Duration(wait) * time.Second):
			case <-ctx.Done():
				return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail, UserAgent: verifyUA}
			}
			elapsed += wait
			notify(fmt.Sprintf("%s [%ds/%ds] Checking live/die UID=%s...", tag, elapsed, checkDelay, session.UID))
			s := verifybase.CheckLiveDieByPicture(ctx, verifyUA, session.UID)
			if s == "Die" {
				status = "Die"
				break
			}
		}
		notify(fmt.Sprintf("%s Check result: %s", tag, status))
	} else {
		notify(fmt.Sprintf("%s Wait %ds (check live disabled)...", tag, checkDelay))
		select {
		case <-time.After(time.Duration(checkDelay) * time.Second):
		case <-ctx.Done():
			return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail, UserAgent: verifyUA}
		}
	}

	notify(fmt.Sprintf("%s Done: %s — %s", tag, status, tempEmail))
	return &instagram.VerifyResult{
		Success:   status == "Live",
		Status:    status,
		Message:   fmt.Sprintf("%s — Email: %s", status, tempEmail),
		Email:     tempEmail,
		UserAgent: verifyUA,
	}
}
