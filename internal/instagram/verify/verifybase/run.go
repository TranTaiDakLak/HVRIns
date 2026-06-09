package verifybase

import (
	"context"
	"fmt"
	mrand "math/rand"
	"net/url"
	"strings"
	"time"

	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/google/uuid"

	"HVRIns/internal/email"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	webreg "HVRIns/internal/instagram/register/web"
)

// Spec carries all variant-specific configuration for RunVerify.
// Variants that share identical logic in a section leave the function field nil
// and the shared default is used.
type Spec struct {
	// Tag is the log prefix, e.g. "[S555 Verify]".
	Tag string

	// DocID is the client_doc_id sent in the form body.
	DocID string

	// BloksVer is the bloks_versioning_id used in body and nt_context.
	BloksVer string

	// StylesID is the styles_id in nt_context (default "6100e7e89411ccf67ace027cedecd84f").
	StylesID string

	// IsPushOn is the is_push_on value in nt_context (default true).
	IsPushOn bool

	// AddEmailTimeout overrides the HTTP timeout for the AddEmail request.
	// Zero means use 30s.
	AddEmailTimeout time.Duration

	// FixUA is called with the session UserAgent and phone country.
	// It should return a corrected UA and a non-empty log message when the UA
	// was regenerated. Return ("", "") to skip validation.
	FixUA func(ua, phone string) (correctedUA, logMsg string)

	// BuildHeaders builds the ordered HTTP headers for a single request.
	// friendlyName is one of the three friendly name constants.
	// withZeroState=true for addEmail and confirm, false for resend.
	BuildHeaders func(sc *SessionCtx, friendlyName string, withZeroState bool) [][2]string

	// BuildAddEmailBody builds the URL-encoded form body for the AddEmail request.
	BuildAddEmailBody func(spec *Spec, emailAddr, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string

	// BuildConfirmBody builds the URL-encoded form body for the Confirm request.
	BuildConfirmBody func(spec *Spec, emailAddr, code, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string

	// BuildResendBody builds the URL-encoded form body for the Resend request.
	BuildResendBody func(spec *Spec, emailAddr, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string

	// Enable2FA performs the variant-specific 2FA enable flow after email confirm.
	// Return ("", nil) to skip. Return ("", non-nil) for non-fatal failure.
	Enable2FA func(ctx context.Context, session *instagram.Session, uid, machineID, deviceID string, emailOTPFn func(string, int) string, notify func(string)) (string, error)

	// PostConfirm is called after email is confirmed (and after 2FA) while status=="Live".
	// Variants can use it to call addinfo or other post-confirm steps.
	// Return nil to continue normally.
	PostConfirm func(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, notify func(string))

	// SetupSessionCtx is called after the SessionCtx is created and InitPinnedHeaders is called.
	// Legacy variants (s23/s555/s556/s557) use this to populate AppnetSID/AppnetNID/BaseTid.
	// Leave nil for s558/s559 which don't use these fields.
	SetupSessionCtx func(sc *SessionCtx)

	// CreateClient overrides the HTTP client factory. Nil → uses verifybase.CreateClient (Android OkHttp).
	// iOS variants set this to verifybase.CreateIOSClient to get Safari TLS fingerprint.
	CreateClient func(proxyStr string) (tls_client.HttpClient, error)

	// GraphEndpoint overrides the base URL for AddEmail and Confirm requests.
	// Empty → defaults to BgraphURL ("https://b-graph.facebook.com/graphql").
	// iOS variants set this to GraphURL ("https://graph.facebook.com/graphql").
	GraphEndpoint string

	// MachineIDFunc overrides machineID computation. nil → uses datr or hardcoded default.
	// iOS variants use this to generate a 24-char base64url iOS-format machineID.
	MachineIDFunc func(datr string) string

	// CheckAddEmailSuccess overrides add-email response success detection.
	// nil → default logic (isSuccess/isExplicitError/isBloksAction in RunVerify).
	// Messenger (appmessv3): Bloks response always has "error_message" field → triggers
	// false isExplicitError. Override to: success = has fb_bloks_action AND no real error.
	CheckAddEmailSuccess func(resp string) bool

	// CheckConfirmSuccess reports whether a confirm response represents success.
	// nil → default: strings.Contains(resp, "confirmation_success").
	// iOS overrides this: success = fb_bloks_action present AND confirmation_failure absent.
	CheckConfirmSuccess func(resp string) bool

	// CheckLiveDieFunc override hàm check live/die sau confirm.
	// nil → default CheckLiveDieCombined (token-first). iOS set
	// CheckLiveDiePictureFirst để ưu tiên picture check trước token.
	CheckLiveDieFunc func(ctx context.Context, ua, uid, token string) string

	// iOS-specific session tokens — truyền từ reg partial result sang verify confirm.
	// Srnonce: server_params.srnonce trong confirm body.
	// SessionlessCryptedUID: server_params.sessionless_crypted_user_id trong confirm body.
	// CloudTrustToken: X-Cloud-Trust-Token header + cloud_trust_token trong body.
	Srnonce               string
	SessionlessCryptedUID string
	CloudTrustToken       string

	// SkipUserTokenCheck — bỏ qua check session.Token != "" ở đầu RunVerify.
	// Dùng cho iOS variants dùng app-level OAuth token trong header (không cần user token).
	SkipUserTokenCheck bool

	// ValidateToken — kiểm tra token hợp lệ cho platform này.
	// nil → default isValidUserToken (EAAAAU Android HOẶC EAAAAAY iOS).
	// iOS set "chỉ EAAAAAY" để loại token Android EAAAAU (Bloks CAA iOS chỉ nhận EAAAAAY).
	ValidateToken func(tok string) bool

	// FetchToken — lấy user token khi session.Token chưa hợp lệ (platform-specific).
	// nil → default REST /auth/login Android (EAAAAU). iOS set = CAA login iOS (EAAAAAY).
	// Trả token; có thể side-effect set session.Cookie bên trong closure.
	FetchToken func(ctx context.Context, session *instagram.Session) (string, error)

	// Phone — số điện thoại của account (từ session.Phone).
	// Dùng trong buildAddEmailBody: khi reg bằng phone, msg_previous_cp = phone (previous CP).
	Phone string

	// AAC (Account Access Context) — token session client-mint lúc reg (iOS Messenger).
	// Set từ session.AAC*; add-mail/confirm phải dùng lại đúng bộ aac của session create.
	AACJid string
	AACcs  string
	AACts  string

	// Flow-session IDs — UUID client-mint lúc reg (iOS Messenger). Set từ session.RegFlowID/
	// HeadersFlowID; add-mail/confirm phải dùng lại đúng bộ flow_id của session create.
	RegFlowID     string
	HeadersFlowID string

	// PassRaw + PassTS — mật khẩu thô và Unix timestamp dùng trong #PWD_ENC:0:ts:pass.
	// BuildAddMailBody cần để điền encrypted_password đúng vào reg_info (tránh template mismatch).
	PassRaw string
	PassTS  int64

	// RegistrationFlowID — UUID sinh ra 1 lần cho cả luồng verify, gửi trong reg_info.
	RegistrationFlowID string

	// BuildCloudTrustTokenBody — nếu không nil, được gọi sau AddEmail thành công để gửi
	// bk.cloud_trust_token.async (capture [384]). Nil = bỏ qua bước này.
	BuildCloudTrustTokenBody func(spec *Spec, deviceID, familyDevID, machineID string) string
}

// SessionCtx holds per-session values needed to build headers.
// Both legacy (s23/s555/s556/s557) and new (s558/s559) header styles
// access the same struct; fields unused by a style are simply ignored.
type SessionCtx struct {
	UA          string
	Token       string
	DeviceID    string
	FamilyDevID string
	MachineID   string
	AppnetSID   string
	AppnetNID   string
	BaseTid     int
	Sim         fakeinfo.SimProfile
	Locale      string

	// Pinned per-session values generated once via InitPinnedHeaders.
	DeviceGroup string
	ConnUUID    string
	TaLoggingID string

	// CloudTrustToken — X-Cloud-Trust-Token header value (iOS verify).
	// Set via SetupSessionCtx; used by BuildHeaders.
	CloudTrustToken string

	// ConnType — X-FB-Connection-Type (iOS verify): "wifi" hoặc
	// "mobile.CTRadioAccessTechnology*". Set via SetupSessionCtx.
	ConnType string
}

// InitPinnedHeaders generates pinned-per-session header values.
func (sc *SessionCtx) InitPinnedHeaders() {
	sc.DeviceGroup = GenPinnedDeviceGroup()
	sc.ConnUUID = GenPinnedConnUUID()
	sc.TaLoggingID = GenPinnedTaLoggingID()
}

// ─── Friendly name constants (shared across all variants) ────────────────────

const (
	AddEmailFriendlyName = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.async.contactpoint_email.async"
	ConfirmFriendlyName  = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.confirmation.async"
	ResendFriendlyName   = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.resend_confirmation.async"
	BgraphURL            = "https://b-graph.facebook.com/graphql"
	GraphURL             = "https://graph.facebook.com/graphql"
)

// ─── RunVerify ────────────────────────────────────────────────────────────────

// RunVerify is the shared orchestration for all verify variants.
// It handles: token check, client creation, cookie injection, email service,
// mail reuse, UA fixup, AddEmail, WaitOTP, Resend, ConfirmCode, CheckLiveDie,
// 2FA, PostConfirm. Variant-specific behaviour is injected via spec.
// isValidUserToken — token Facebook access hợp lệ: EAAAAU (Android) hoặc EAAAAAY (iOS).
// Khớp đúng isValidAndroidToken ở runner/scheduler.go để nhất quán toàn hệ thống.
// Verify CHỈ được chạy khi có user token thật — chặn token rỗng/rác/cookie lọt vào AddEmail.
func isValidUserToken(tok string) bool {
	return strings.HasPrefix(tok, "EAAAAU") || strings.HasPrefix(tok, "EAAAAAY")
}

func RunVerify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string), spec Spec) *instagram.VerifyResult {
	// Tag — ưu tiên UserApiLabel (tên API VER user chọn trong UI) hơn Spec.Tag hardcoded.
	// Vd: user chọn "api token" → log "[api token]" thay vì "[S23 Verify]".
	tag := spec.Tag
	if cfg != nil && cfg.UserApiLabel != "" {
		tag = "[" + cfg.UserApiLabel + "]"
	}
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(session.UID, msg)
		}
	}

	// validate — token hợp lệ cho platform này. iOS override = chỉ EAAAAAY.
	validate := spec.ValidateToken
	if validate == nil {
		validate = isValidUserToken
	}
	if !validate(session.Token) && !spec.SkipUserTokenCheck {
		// Token chưa hợp lệ → fetch theo platform.
		// iOS (spec.FetchToken != nil): CAA login iOS lấy EAAAAAY (KHÔNG dùng EAAAAU).
		// Android (default): REST /auth/login lấy EAAAAU từ uid+password.
		if spec.FetchToken != nil {
			notify(tag + " Token chưa đúng loại — login lấy token...")
			fetchCtx, fetchCancel := context.WithTimeout(ctx, 60*time.Second)
			tok, ferr := spec.FetchToken(fetchCtx, session)
			fetchCancel()
			if ferr != nil {
				notify(tag + " Login lấy token lỗi: " + ferr.Error())
			} else if validate(tok) {
				session.Token = tok
				notify(fmt.Sprintf("%s Login token OK — %s...", tag, tok[:Mmin(len(tok), 20)]))
			}
		} else if session.UID != "" && session.Password != "" {
			notify(tag + " Token rỗng — REST /auth/login (port S399)...")
			fetchCtx, fetchCancel := context.WithTimeout(ctx, 30*time.Second)
			// PORT S399 step 2: REST classic API stable, không phụ thuộc Bloks schema.
			fetched, newCookie := webreg.FetchAndroidTokenLegacyWithCookie(fetchCtx, session.UID, session.Password, session.Datr, "en_US", "", session.Proxy, "", func(m string) { notify(tag + " " + m) })
			fetchCancel()
			// Chỉ nhận token hợp lệ — login ra rỗng/rác thì coi như chưa có token.
			if validate(fetched) {
				session.Token = fetched
				if newCookie != "" {
					session.Cookie = newCookie // cookie mới từ login → UI cập nhật
				}
				notify(fmt.Sprintf("%s Fetched token OK — %s...", tag, fetched[:Mmin(len(fetched), 20)]))
			}
		}
		// Bắt buộc phải có token hợp lệ mới được vào verify. Token rỗng/rác/sai loại → bỏ account.
		// iOS: bắt buộc EAAAAAY (Bloks CAA iOS). Android-family: EAAAAU/EAAAAAY.
		if !validate(session.Token) {
			notify(tag + " ERROR: Missing/invalid access token!")
			return &instagram.VerifyResult{Status: "error", Message: "Missing/invalid access token (sai loại hoặc lấy token thất bại)"}
		}
	}

	tokenDisplay := session.Token
	if tokenDisplay == "" {
		tokenDisplay = "(app-token)"
	}
	notify(fmt.Sprintf("%s Starting... UID=%s Token=%s...",
		tag, session.UID, tokenDisplay[:Mmin(len(tokenDisplay), 20)]))

	// Create HTTP client
	clientFactory := CreateClient
	if spec.CreateClient != nil {
		clientFactory = spec.CreateClient
	}
	client, err := clientFactory(session.Proxy)
	if err != nil {
		notify(fmt.Sprintf("%s Client ERROR: %v", tag, err))
		return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Create client: %v", err)}
	}
	defer client.CloseIdleConnections()

	// Extract datr from cookie string if not already set
	datr := session.Datr
	if datr == "" {
		datr = ExtractDatrFromCookieStr(session.Cookie)
	}

	// Inject cookies (datr, sb, fr) for session consistency
	if session.Cookie != "" {
		InjectVerifyCookies(client, session.Cookie)
		if datr != "" {
			notify(fmt.Sprintf("%s Injected cookies (datr=%s...)", tag, datr[:Mmin(len(datr), 10)]))
		}
	}

	// Session IDs — reuse device_id from reg if available
	deviceID := session.DeviceID
	if deviceID == "" {
		deviceID = uuid.New().String()
	}
	familyDeviceID := session.FamilyDeviceID
	if familyDeviceID == "" {
		familyDeviceID = uuid.New().String()
	}
	waterfallID := uuid.New().String()
	machineID := "Kb_UaVc0y5UrH8GU29y9f_9c"
	if datr != "" {
		machineID = datr
	}
	if spec.MachineIDFunc != nil {
		machineID = spec.MachineIDFunc(datr)
	}

	// Proxy override for mail reading
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

	// Create email service
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

	// TempMail reuse: if reg already created a temp mail, reuse it
	var tempEmail string
	reuseMail := false
	if session.Email != "" && session.EmailMeta != "" && email.RestoreIfPossible(emailSvc, session.EmailMeta) {
		tempEmail = session.Email
		reuseMail = true
		origNotify := notify
		notify = func(msg string) { origNotify("[REUSE] " + msg) }
		notify(fmt.Sprintf("%s Reuse mail from register: %s (skip CreateEmail+AddEmail)", tag, tempEmail))
		if cfg.OnEmailCreated != nil {
			cfg.OnEmailCreated(tempEmail)
		}
	} else {
		var err error
		tempEmail, err = RetryCreateEmail(ctx, emailSvc, notify)
		if tempEmail != "" && cfg.OnEmailCreated != nil {
			cfg.OnEmailCreated(tempEmail)
		}
		if err != nil {
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Create email: %v", err)}
		}
		notify(fmt.Sprintf("%s Email: %s", tag, tempEmail))
	}

	firstName, lastName := SplitFullName(session.FullName)
	uid := session.UID
	gender := 2 // default male

	sim := fakeinfo.RandomSimProfile(CountryFromPhone(session.Phone))
	notify(fmt.Sprintf("%s SIM: %s (%s)", tag, sim.OperatorName, sim.HNI))

	locale := "en_US"
	if cfg.DeepFakeLocale {
		if l := ExtractLocaleFromCookie(session.Cookie); l != "" {
			locale = l
		}
	}

	// UA validation / regeneration
	verifyUA := session.UserAgent
	if spec.FixUA != nil {
		if corrected, logMsg := spec.FixUA(verifyUA, session.Phone); corrected != "" {
			verifyUA = corrected
			notify(fmt.Sprintf("%s %s", tag, logMsg))
		}
	}

	sc := buildSessionCtx(verifyUA, session.Token, deviceID, familyDeviceID, machineID, sim, locale)
	if spec.SetupSessionCtx != nil {
		spec.SetupSessionCtx(sc)
	}

	// === Step 1: Add email ===
	if !reuseMail {
		notify(fmt.Sprintf("%s Adding email %s [provider=%s]...", tag, tempEmail, cfg.MailProvider))
		addBody := spec.BuildAddEmailBody(&spec, tempEmail, uid, firstName, lastName, deviceID, familyDeviceID, waterfallID, machineID, locale, gender, sc.Sim)
		addHeaders := spec.BuildHeaders(sc, AddEmailFriendlyName, true)

		addTimeout := spec.AddEmailTimeout
		if addTimeout <= 0 {
			addTimeout = 30 * time.Second
		}
		addEndpoint := BgraphURL
		if spec.GraphEndpoint != "" {
			addEndpoint = spec.GraphEndpoint
		}
		addCtx, addCancel := context.WithTimeout(ctx, addTimeout)
		addResp, err := DoPost(addCtx, client, addEndpoint, addBody, addHeaders)
		addCancel()
		if err != nil {
			notify(fmt.Sprintf("%s Add email HTTP error: %v", tag, err))
			// Mail CHƯA tới FB (HTTP/network error) → trả về pool reuse, tránh phí.
			email.ReleaseIfPossible(ctx, emailSvc)
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Add email: %v", err), Email: tempEmail}
		}

		// CheckAddEmailSuccess override (Messenger appmessv3): bypass default detection khi
		// Bloks template chứa "error_message" field → false isExplicitError.
		addEmailDone := false
		if spec.CheckAddEmailSuccess != nil {
			addEmailDone = true
			if spec.CheckAddEmailSuccess(addResp) {
				notify(tag + " Email added OK (override) — waiting for OTP...")
			} else {
				short := SummarizeFBError(addResp)
				notify(tag + " Add email failed (override): " + short)
				email.ReleaseIfPossible(ctx, emailSvc)
				return &instagram.VerifyResult{Status: "error", Message: "Add email failed: " + short, Email: tempEmail}
			}
		}
		if !addEmailDone {
			// FB AddEmail response detection (port 2026-05-16):
			// 1. Success patterns: explicit success strings từ Bloks DSL hoặc CAA flow
			// 2. Error patterns: explicit error indicators (rate limit, checkpoint, invalid email, ...)
			// 3. Fallback: nếu response là Bloks action object (fb_bloks_action) mà KHÔNG có error
			//    indicator → optimistic SUCCESS (đợi OTP). Trước đây code reject mọi response không
			//    match success patterns → false positive khi FB đổi Bloks DSL structure.
			respLow := strings.ToLower(addResp)
			isSuccess := strings.Contains(addResp, "Check your email") ||
				strings.Contains(addResp, "code we sent to") ||
				strings.Contains(addResp, "CAA_REG_CONFIRMATION") ||
				strings.Contains(respLow, "check_email") ||
				strings.Contains(respLow, "caa_reg_confirmation") ||
				strings.Contains(respLow, "confirmation_code")
			isExplicitError := strings.Contains(respLow, "\"errors\":[{") ||
				strings.Contains(respLow, "\"error_message\"") ||
				strings.Contains(respLow, "email_already_used") ||
				strings.Contains(respLow, "email_is_invalid") ||
				strings.Contains(respLow, "rate_limit") ||
				strings.Contains(respLow, "too many requests") ||
				strings.Contains(respLow, "checkpoint") ||
				strings.Contains(respLow, "is_checkpointed") ||
				strings.Contains(respLow, "field_exception") ||
				strings.Contains(respLow, "session is invalid") ||
				strings.Contains(respLow, "session has expired") ||
				strings.Contains(respLow, "account is currently disabled") ||
				strings.Contains(respLow, "account has been disabled") ||
				strings.Contains(respLow, "something went wrong") ||
				strings.Contains(respLow, "we're sorry") ||
				strings.Contains(addResp, "confirmation_step_error")
			isBloksAction := strings.Contains(respLow, "fb_bloks_action") || strings.Contains(respLow, "action_bundle")
			// mailIsBad — mail thực sự hỏng (FB từ chối vì email): KHÔNG recycle (reuse sẽ
			// fail tiếp). Các lỗi khác (rate_limit/checkpoint/account problem) → mail còn tốt,
			// recycle được vì FB chưa link mail vào account.
			mailIsBad := strings.Contains(respLow, "email_already_used") ||
				strings.Contains(respLow, "email_is_invalid")
			if isSuccess {
				notify(tag + " Email added OK — waiting for OTP...")
				if spec.BuildCloudTrustTokenBody != nil {
					cttBody := spec.BuildCloudTrustTokenBody(&spec, deviceID, familyDeviceID, machineID)
					cttHeaders := spec.BuildHeaders(sc, "FBBloksActionRootQuery-com.bloks.www.bk.cloud_trust_token.async", false)
					cttEndpoint := BgraphURL
					if spec.GraphEndpoint != "" {
						cttEndpoint = spec.GraphEndpoint
					}
					cttCtx, cttCancel := context.WithTimeout(ctx, 15*time.Second)
					DoPost(cttCtx, client, cttEndpoint, cttBody, cttHeaders) //nolint:errcheck
					cttCancel()
				}
			} else if isExplicitError {
				short := SummarizeFBError(addResp)
				notify(tag + " Add email failed: " + short)
				if !mailIsBad {
					email.ReleaseIfPossible(ctx, emailSvc) // mail còn tốt → trả pool reuse
				}
				return &instagram.VerifyResult{Status: "error", Message: "Add email failed: " + short, Email: tempEmail}
			} else if isBloksAction {
				// Bloks action response mà không có error indicator → assume success, đợi OTP.
				notify(tag + " Email added (Bloks action, no explicit error) — waiting for OTP...")
			} else {
				// Response không match pattern nào → fall back reject. Mail chưa chắc đã add
				// (response lạ) → recycle để an toàn tránh phí.
				short := SummarizeFBError(addResp)
				notify(tag + " Add email failed: " + short)
				email.ReleaseIfPossible(ctx, emailSvc)
				return &instagram.VerifyResult{Status: "error", Message: "Add email failed: " + short, Email: tempEmail}
			}
		} // end if !addEmailDone
	} else {
		notify(tag + " Skip AddEmail (reuse mail from reg) — waiting for OTP...")
	}

	// === Step 2: Wait for OTP ===
	waitSec := cfg.TimeDelaySendCode
	if waitSec <= 0 {
		waitSec = 30
	}
	pollMs := cfg.WaitMailMs
	if pollMs <= 0 {
		pollMs = 2000
	}
	notify(fmt.Sprintf("%s Waiting for OTP... [mail: %s]", tag, emailSvc.GetEmail()))
	maxRetry := waitSec * 1000 / pollMs
	if maxRetry < 1 {
		maxRetry = 1
	}
	stopHB := StartOTPHeartbeat(ctx, notify, 5*time.Second, tag, emailSvc.GetEmail())
	code, err := emailSvc.WaitForCode(ctx, maxRetry, pollMs)
	stopHB()
	if err != nil {
		if !cfg.SendAgainCode {
			// OTP timeout = verify THẤT BẠI → trả mail về pool cho account khác (user request).
			email.ReleaseIfPossible(ctx, emailSvc)
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("OTP timeout: %v", err), Email: tempEmail}
		}
		notify(tag + " OTP timeout — resending code...")

		resendBody := spec.BuildResendBody(&spec, tempEmail, uid, firstName, lastName, deviceID, familyDeviceID, waterfallID, machineID, locale, gender, sc.Sim)
		resendHeaders := spec.BuildHeaders(sc, ResendFriendlyName, false)
		DoPost(ctx, client, GraphURL, resendBody, resendHeaders) //nolint:errcheck

		notify(tag + " Resent — waiting again...")
		stopHB2 := StartOTPHeartbeat(ctx, notify, 5*time.Second, tag, emailSvc.GetEmail())
		code, err = emailSvc.WaitForCode(ctx, maxRetry, pollMs)
		stopHB2()
		if err != nil {
			// OTP timeout sau resend = verify THẤT BẠI → trả mail về pool.
			email.ReleaseIfPossible(ctx, emailSvc)
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("OTP timeout after resend: %v", err), Email: tempEmail}
		}
	}
	notify(fmt.Sprintf("%s OTP: %s", tag, code))

	// Delay before confirm
	if cfg.DelayConfirmEmail > 0 {
		notify(fmt.Sprintf("%s Waiting %ds before confirm...", tag, cfg.DelayConfirmEmail))
		select {
		case <-ctx.Done():
			return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail}
		case <-time.After(time.Duration(cfg.DelayConfirmEmail) * time.Second):
		}
	}

	// === Step 3: Confirm code ===
	notify(fmt.Sprintf("%s Submitting OTP %s → confirm email...", tag, code))
	confirmBody := spec.BuildConfirmBody(&spec, tempEmail, code, uid, firstName, lastName, deviceID, familyDeviceID, waterfallID, machineID, locale, gender, sc.Sim)
	confirmHeaders := spec.BuildHeaders(sc, ConfirmFriendlyName, true)

	confirmEndpoint := BgraphURL
	if spec.GraphEndpoint != "" {
		confirmEndpoint = spec.GraphEndpoint
	}
	const confirmMaxRetries = 3
	var confirmResp string
	var confirmErr error
	confirmOK := false
	for attempt := 1; attempt <= confirmMaxRetries; attempt++ {
		confirmResp, confirmErr = DoPost(ctx, client, confirmEndpoint, confirmBody, confirmHeaders)
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

		isConfirmOK := false
		if spec.CheckConfirmSuccess != nil {
			isConfirmOK = spec.CheckConfirmSuccess(confirmResp)
		} else {
			isConfirmOK = strings.Contains(confirmResp, "confirmation_success")
		}
		if isConfirmOK {
			notify(tag + " Email confirmed!")
			confirmOK = true
			break
		}
		if strings.Contains(confirmResp, "checkpointed") || strings.Contains(confirmResp, "code\":459") {
			return &instagram.VerifyResult{Status: "Die", Message: "Checkpoint after confirm", Email: tempEmail}
		}
		notify(fmt.Sprintf("%s Confirm unexpected (attempt %d/%d): %s", tag, attempt, confirmMaxRetries, confirmResp[:Mmin(len(confirmResp), 200)]))
		if attempt < confirmMaxRetries {
			select {
			case <-ctx.Done():
				return &instagram.VerifyResult{Status: "error", Message: "Cancelled", Email: tempEmail}
			case <-time.After(2 * time.Second):
			}
		}
	}
	if !confirmOK {
		return &instagram.VerifyResult{Status: "error", Message: "Confirm failed after retries", Email: tempEmail}
	}

	// Wait before live check — periodic every 5s, bail early on die
	checkDelay := cfg.TimeDelayCheck
	if checkDelay <= 0 {
		checkDelay = 5
	}
	status := "Live"
	if cfg.CheckLiveDie {
		checkInterval := 5
		notify(fmt.Sprintf("%s Checking live/die every %ds for %ds total (UID=%s)...", tag, checkInterval, checkDelay, uid))
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
			notify(fmt.Sprintf("%s [%ds/%ds] Checking live/die UID=%s...", tag, elapsed, checkDelay, uid))
			checkFn := CheckLiveDieCombined
			if spec.CheckLiveDieFunc != nil {
				checkFn = spec.CheckLiveDieFunc
			}
			s := checkFn(ctx, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36", uid, session.Token)
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

	// 2FA
	twoFAKey := ""
	if spec.Enable2FA != nil && cfg.Enable2FA && status == "Live" {
		notify(tag + " Enabling 2FA TOTP...")
		emailOTPFn := func(maskedEmail string, _ int) string {
			notify(fmt.Sprintf("%s 2FA reauth — waiting OTP for %s...", tag, maskedEmail))
			c, e := emailSvc.WaitForCode(ctx, 3, 3000)
			if e != nil {
				notify(fmt.Sprintf("%s 2FA reauth OTP timeout: %v", tag, e))
				return ""
			}
			return c
		}
		secret, err2fa := spec.Enable2FA(ctx, session, uid, machineID, deviceID, emailOTPFn, notify)
		if err2fa != nil {
			notify(fmt.Sprintf("%s 2FA enable failed (non-fatal): %v", tag, err2fa))
		} else {
			twoFAKey = secret
			notify(fmt.Sprintf("%s 2FA enabled — secret: %s", tag, twoFAKey))
		}
	}

	// Post-confirm steps (e.g. addinfo)
	if spec.PostConfirm != nil && status == "Live" {
		spec.PostConfirm(ctx, session, cfg, notify)
	}

	notify(fmt.Sprintf("%s Done: %s — %s", tag, status, tempEmail))

	msg := fmt.Sprintf("%s — Email: %s", status, tempEmail)
	if twoFAKey != "" {
		msg = fmt.Sprintf("%s — Email: %s — 2FA: %s", status, tempEmail, twoFAKey)
	}

	// s23 returns UserAgent only if it's different from session.UserAgent (it doesn't override).
	// s555+ return verifyUA always. We always populate it here; s23 can ignore it.
	return &instagram.VerifyResult{
		Success:   status == "Live",
		Status:    status,
		Message:   msg,
		Email:     tempEmail,
		UserAgent: verifyUA,
		TwoFA:     twoFAKey,
	}
}

// buildSessionCtx constructs a SessionCtx and initialises pinned headers.
func buildSessionCtx(ua, token, deviceID, familyDevID, machineID string, sim fakeinfo.SimProfile, locale string) *SessionCtx {
	sc := &SessionCtx{
		UA:          ua,
		Token:       token,
		DeviceID:    deviceID,
		FamilyDevID: familyDevID,
		MachineID:   machineID,
		Sim:         sim,
		Locale:      locale,
	}
	sc.InitPinnedHeaders()
	return sc
}

// ─── appnetSID / appnetNID / baseTid helpers for legacy header style ──────────

// NewAppnetSID generates a 32-char hex-like appnet session ID.
func NewAppnetSID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:32]
}

// NewAppnetNID generates an appnet NID with trailing ",Wifi".
func NewAppnetNID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")[:32] + ",Wifi"
}

// SetAppnetFields populates AppnetSID, AppnetNID, BaseTid on sc for legacy header style.
func SetAppnetFields(sc *SessionCtx) {
	sc.AppnetSID = NewAppnetSID()
	sc.AppnetNID = NewAppnetNID()
	sc.BaseTid = 2000 + mrand.Intn(1000)
}

// ─── buildVariables (shared body helper) ─────────────────────────────────────

// BuildVariables builds the outer variables JSON with scale + nt_context.
// stylesID and isPushOn come from the Spec so each variant gets the right values.
func BuildVariables(paramsObj map[string]interface{}, bloksVer, stylesID string, isPushOn bool) string {
	return BuildVariablesWithTheme(paramsObj, bloksVer, stylesID, isPushOn, []interface{}{
		map[string]interface{}{"value": []string{"three_neutral_gray"}, "design_system_name": "XMDS"},
		map[string]interface{}{"value": []string{}, "design_system_name": "FDS"},
	})
}

// BuildVariablesWithTheme builds the outer variables JSON with a variant-specific theme_params payload.
func BuildVariablesWithTheme(paramsObj map[string]interface{}, bloksVer, stylesID string, isPushOn bool, themeParams []interface{}) string {
	ntCtx := map[string]interface{}{
		"using_white_navbar":           true,
		"styles_id":                    stylesID,
		"pixel_ratio":                  3,
		"is_push_on":                   isPushOn,
		"debug_tooling_metadata_token": nil,
		"is_flipper_enabled":           false,
		"theme_params":                 themeParams,
		"bloks_version":                bloksVer,
	}
	variables := map[string]interface{}{
		"params": paramsObj,
		"scale":  "3",
		"use_native_entrypoint_for_stars_on_reels": true,
		"nt_context": ntCtx,
	}
	return MustJSON(variables)
}

// BuildFormBody assembles an application/x-www-form-urlencoded body.
// docID comes from the Spec.
func BuildFormBody(friendlyName, docID, variablesJSON, traceID, locale string) string {
	if locale == "" {
		locale = "en_US"
	}
	form := url.Values{}
	form.Set("method", "post")
	form.Set("pretty", "false")
	form.Set("format", "json")
	form.Set("server_timestamps", "true")
	form.Set("locale", locale)
	form.Set("purpose", "fetch")
	form.Set("fb_api_req_friendly_name", friendlyName)
	form.Set("fb_api_caller_class", "graphservice")
	form.Set("client_doc_id", docID)
	form.Set("fb_api_client_context", `{"is_background":"0"}`)
	form.Set("variables", variablesJSON)
	form.Set("fb_api_analytics_tags", `["GraphServices"]`)
	form.Set("client_trace_id", traceID)
	return form.Encode()
}

// ─── enable2FA shared helper ─────────────────────────────────────────────────

// Enable2FAWithAndroidSec is shared across s555/s556/s557/s558/s559 — all use androidsec.
// Import androidsec from the caller package to avoid import cycle; pass it as a function.
type Enable2FAFn func(ctx context.Context, session *instagram.Session, uid, machineID, deviceID string, emailOTPFn func(string, int) string, notify func(string)) (string, error)

// ─── DoPost exposed via client (used by variants that need it directly) ───────

// GetClient is an alias type for the tls_client interface for external use.
type Client = tls_client.HttpClient
