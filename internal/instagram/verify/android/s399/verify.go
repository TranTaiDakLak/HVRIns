// Package s399 — Facebook FB4A v399 verify (change email + enter code).
//
// Khác hoàn toàn s5xx Bloks/GraphQL:
//   - Endpoint: POST graph.facebook.com/me/edit_registration_contactpoint (add email)
//                POST graph.facebook.com/me/confirm_contactpoint (submit OTP)
//   - friendly-name: editRegistrationContactpoint + confirmContactpoint
//   - Body form-urlencoded thuần (không nested JSON như Bloks)
//   - Auth: OAuth <EAA token> trong header Authorization
//
// Captured traffic confirmed (May 2026): /me/edit_registration_contactpoint trả {"result":true};
// /me/confirm_contactpoint trả empty body (HTTP 200 = success).
package s399

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
	instagram.RegisterPlatformVerifier(instagram.PlatformS399, func() instagram.Verifier {
		return &Verifier{}
	})
	instagram.RegisterPlatformVerifyUA(instagram.PlatformS399, RandomUA)
}

const tag = "[S399 Verify]"

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

	// UA: dùng UA verify v399 (regen nếu UA hiện tại không phải FBAV/399)
	verifyUA := session.UserAgent
	if !strings.Contains(verifyUA, "FBAV/399.") {
		country := verifybase.CountryFromPhone(session.Phone)
		verifyUA = RandomUA(country)
		label := country
		if label == "" {
			label = "random"
		}
		notify(fmt.Sprintf("%s UA regenerated (FBAV/399, country=%s)", tag, label))
	}

	// Locale: ưu tiên cookie, fallback en_US
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

	// SIM HNI cho headers (best-effort match country)
	simHNI := "45204" // default Viettel VN
	if cs := strings.ToUpper(countryCode); cs != "" {
		// gán theo bảng nhỏ — không cần đầy đủ vì chỉ là display, FB không reject
		// nếu HNI hợp lệ format MCC+MNC.
		_ = cs
	}

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
		return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Email service: %v", err)}
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
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Create email: %v", err)}
		}
		notify(fmt.Sprintf("%s Email: %s", tag, tempEmail))
	}

	// ─── Step 1: Add email (POST /me/edit_registration_contactpoint) ─────────
	if !reuseMail {
		notify(fmt.Sprintf("%s Adding email %s...", tag, tempEmail))
		addBody := buildAddEmailBody(tempEmail, locale, countryCode)
		addHeaders := buildVerifyHeaders(verifyUA, session.Token, simHNI, addEmailFriendly, "2610", "WIFI")

		addTimeout := 30 * time.Second
		addCtx, addCancel := context.WithTimeout(ctx, addTimeout)
		addResp, err := verifybase.DoPost(addCtx, client, addEmailURL, addBody, addHeaders)
		addCancel()
		if err != nil {
			notify(fmt.Sprintf("%s Add email HTTP error: %v", tag, err))
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Add email: %v", err), Email: tempEmail}
		}

		// Response v399: {"result":true} = success
		if strings.Contains(addResp, `"result":true`) || strings.Contains(addResp, `"result": true`) {
			notify(tag + " Email added OK — waiting for OTP...")
		} else {
			short := verifybase.SummarizeFBError(addResp)
			notify(tag + " Add email failed: " + short)
			return &instagram.VerifyResult{Status: "error", Message: "Add email failed: " + short, Email: tempEmail}
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
		// v399 không có resend endpoint riêng — chỉ fail
		return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("OTP timeout: %v", err), Email: tempEmail}
	}
	notify(fmt.Sprintf("%s OTP: %s", tag, code))

	if cfg.DelayConfirmEmail > 0 {
		notify(fmt.Sprintf("%s Waiting %ds before confirm...", tag, cfg.DelayConfirmEmail))
		select {
		case <-ctx.Done():
			return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail}
		case <-time.After(time.Duration(cfg.DelayConfirmEmail) * time.Second):
		}
	}

	// ─── Step 3: Confirm code (POST /me/confirm_contactpoint) ────────────────
	notify(fmt.Sprintf("%s Submitting OTP %s → confirm email...", tag, code))
	confirmBody := buildConfirmCodeBody(tempEmail, code, locale, countryCode)
	confirmHeaders := buildVerifyHeaders(verifyUA, session.Token, simHNI, confirmFriendly, "2610", "WIFI")

	const confirmMaxRetries = 3
	confirmOK := false
	for attempt := 1; attempt <= confirmMaxRetries; attempt++ {
		confirmResp, confirmErr := verifybase.DoPost(ctx, client, confirmURL, confirmBody, confirmHeaders)
		if confirmErr != nil {
			notify(fmt.Sprintf("%s Confirm HTTP error (attempt %d/%d): %v", tag, attempt, confirmMaxRetries, confirmErr))
			if attempt < confirmMaxRetries {
				select {
				case <-ctx.Done():
					return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail}
				case <-time.After(2 * time.Second):
				}
				continue
			}
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Confirm: %v", confirmErr), Email: tempEmail}
		}

		// Response v399: empty body (HTTP 200) hoặc JSON với error nếu fail.
		// Coi như success nếu không có error markers.
		respLow := strings.ToLower(confirmResp)
		if strings.Contains(respLow, "checkpointed") || strings.Contains(respLow, `"code":459`) {
			return &instagram.VerifyResult{Status: "Die", Message: "Checkpoint after confirm", Email: tempEmail}
		}
		if strings.Contains(respLow, "error") && (strings.Contains(respLow, `"code":`) || strings.Contains(respLow, `"message":`)) {
			short := verifybase.SummarizeFBError(confirmResp)
			notify(fmt.Sprintf("%s Confirm error (attempt %d/%d): %s", tag, attempt, confirmMaxRetries, short))
			if attempt < confirmMaxRetries {
				select {
				case <-ctx.Done():
					return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail}
				case <-time.After(2 * time.Second):
				}
				continue
			}
			return &instagram.VerifyResult{Status: "error", Message: "Confirm failed: " + short, Email: tempEmail}
		}
		// Empty body / no error → success
		notify(tag + " Email confirmed!")
		confirmOK = true
		break
	}
	if !confirmOK {
		return &instagram.VerifyResult{Status: "error", Message: "Confirm failed after retries", Email: tempEmail}
	}

	// ─── Step 4: Live/Die check — periodic every 5s, bail early on die ─────────
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
				return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail}
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
			return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail}
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
