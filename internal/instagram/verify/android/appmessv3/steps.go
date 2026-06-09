// steps.go — AppMess V3 verify: login kiểu V3 (Messenger send_login_request) → set
// token EAAD + session_cookies, rồi chạy luồng ver chuẩn (add email → OTP → confirm →
// live check) qua verifybase.RunVerify.
//
// Body add-email/confirm/resend BÁM SÁT capture V4 (Orca v530): format FB "{params:{...},}",
// nt_context tối giản, reg_info CAA đầy đủ, doc_id 1199408042628.., bloks dadbbd68..,
// login_surface=login_home. KHÁC s565 (FB4A): s565 dùng {"params":"..."} + nt_context rườm rà.
package appmessv3

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	mrand "math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"HVRIns/internal/instagram"
	addinfo "HVRIns/internal/instagram/addinfo/s557"
	"HVRIns/internal/instagram/fakeinfo"
	webreg "HVRIns/internal/instagram/register/web"
	androidsec "HVRIns/internal/instagram/security/android"
	"HVRIns/internal/instagram/verify/verifybase"
)

const (
	// Hằng số action queries Messenger v530 (capture V4: login/add-email/confirm DÙNG CHUNG).
	verifyDocID    = "11994080426288799937543572098"
	verifyBloksVer = "dadbbd68d34735f7a39b791542ad0ecd1b257eddf3e70ab790d47b3cedd8b093"
)

func verifyAccount(ctx context.Context, platform string, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	vv := vverForPlatform(platform) // doc_id/bloks/FBAV theo version (530/535/545)
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(session.UID, msg)
		}
	}

	country := verifybase.CountryFromPhone(session.Phone)
	ua := session.UserAgent
	if !strings.Contains(ua, "FBAN/Orca-Android") {
		ua = randomOrcaUAVer(country, vv.fbav, vv.fbbv)
	}

	deviceID := session.DeviceID
	if deviceID == "" {
		deviceID = uuid.New().String()
	}
	familyDeviceID := session.FamilyDeviceID
	if familyDeviceID == "" {
		familyDeviceID = deviceID
	}
	machineID := session.Datr
	if machineID == "" {
		machineID = genMachineID()
	}

	// ── Token: lấy EAAAAU (context) rồi login Messenger bằng password → EAAD ──
	baseTok := strings.TrimSpace(session.Token)
	if !strings.HasPrefix(baseTok, "EAAAAU") {
		if session.UID != "" && session.Password != "" {
			notify("[AppMessV3 Verify] Lấy EAAAAU context — REST /auth/login...")
			lctx, lcancel := context.WithTimeout(ctx, 30*time.Second)
			fetched, newCookie := webreg.FetchAndroidTokenLegacyWithCookie(lctx, session.UID, session.Password, session.Datr, "en_US", country, session.Proxy, "", func(m string) { notify(m) })
			lcancel()
			if strings.HasPrefix(fetched, "EAAAAU") {
				baseTok = fetched
				if newCookie != "" {
					session.Cookie = newCookie
				}
			}
		}
	}
	// Login Messenger dùng PASSWORD làm credential chính; EAAAAU chỉ là aymh context (optional).
	if session.UID == "" || session.Password == "" {
		notify("[AppMessV3 Verify] ERROR: thiếu uid hoặc password để login V3")
		return &instagram.VerifyResult{Status: "error", Message: "Thiếu uid/password cho login V3"}
	}
	if !strings.HasPrefix(baseTok, "EAAAAU") {
		baseTok = "" // không có context token → aymh rỗng, login bằng password
	}

	notify("[AppMessV3 Verify] Login V3 (send_login_request, password)...")
	loginCtx, loginCancel := context.WithTimeout(ctx, 60*time.Second)
	lctx, lerr := FetchMessengerSession(loginCtx, session.UID, session.Password, baseTok, session.FullName, deviceID, familyDeviceID, machineID, ua, "en_GB", session.Proxy, notify)
	loginCancel()
	if lerr != nil || lctx.Token == "" {
		notify(fmt.Sprintf("[AppMessV3 Verify] Login V3 lỗi (%v) — fallback dùng EAAAAU", lerr))
		if baseTok == "" {
			return &instagram.VerifyResult{Status: "error", Message: fmt.Sprintf("Login V3 fail + không có EAAAAU fallback: %v", lerr)}
		}
		session.Token = baseTok
	} else {
		notify(fmt.Sprintf("[AppMessV3 Verify] Login V3 OK — token EAAD=%.16s... cookie=%t regctx=%t",
			lctx.Token, lctx.Cookie != "", lctx.RegContext != ""))
		session.Token = lctx.Token
		if lctx.Cookie != "" {
			session.Cookie = lctx.Cookie
		}
	}
	// Tên thật của account từ login response → reg_info.first_name/last_name khớp (capture V5).
	if name := strings.TrimSpace(lctx.FirstName + " " + lctx.LastName); name != "" {
		session.FullName = name
	}
	session.UserAgent = ua

	// DIAGNOSTIC: contactpoint GỐC của account (từ login). previous_cp = phone → flow "đổi
	// phone→email" (capture V5, OK). previous_cp = email (account reg-mail) → FB rẽ sub-flow
	// "đổi email→email" KHÁC → confirm dễ fail. Log để chẩn đoán reg-mail vs reg-phone.
	notify(fmt.Sprintf("[AppMessV3 Verify] previous_cp=%q type=%q (reg-mail nếu type=email)",
		lctx.Contactpoint, lctx.ContactpointType))

	// Gọi 2 bước render (bottomsheet → change.email) để lấy reg_context đúng trạng thái.
	// Nếu WaterfallID rỗng (regex miss) → dùng fresh UUID (FB vẫn accept).
	finalRegCtx := lctx.RegContext
	renderWaterfallID := lctx.WaterfallID
	if renderWaterfallID == "" {
		renderWaterfallID = uuid.New().String() // fallback: fresh waterfall
	}
	if lctx.RegContext != "" {
		if renderClient, err := verifybase.CreateClient(session.Proxy); err == nil {
			// Patch lctx với waterfall fallback cho render steps
			lctxRender := lctx
			lctxRender.WaterfallID = renderWaterfallID
			renderCtx, renderCancel := context.WithTimeout(ctx, 60*time.Second)
			updated := FetchConfirmationContext(renderCtx, renderClient, lctxRender, deviceID, familyDeviceID, session.UID, ua, "en_GB", notify)
			renderCancel()
			renderClient.CloseIdleConnections()
			if updated != "" {
				finalRegCtx = updated
			}
		}
	}
	// currentRegCtx theo dõi reg_context mới nhất — được update từ add-email response.
	// buildConfirmBody đọc từ con trỏ này để dùng reg_context tươi nhất.
	currentRegCtx := finalRegCtx

	// isEmailCP: account reg-MAIL (contactpoint gốc = email). Capture VerMailByregMail cho thấy
	// add-email/confirm của reg-mail phải dùng luồng LOGIN_HOME (login_surface=login_home,
	// login_entry_point=logged_out, msg_previous_cp="", KHÔNG gửi reg_context) — KHÁC luồng
	// registration của reg-phone. Branch theo type để KHÔNG đụng reg-phone đang chạy tốt.
	isEmailCP := strings.EqualFold(strings.TrimSpace(lctx.ContactpointType), "email")

	spec := verifybase.Spec{
		Tag:                   "[AppMessV3 Verify]",
		DocID:                 vv.docID,
		BloksVer:              vv.bloksVer,
		IsPushOn:              false,
		AddEmailTimeout:       30 * time.Second,
		SkipUserTokenCheck:    true,
		SessionlessCryptedUID: lctx.CryptedUID,
		Phone:                 lctx.Contactpoint, // original contactpoint (email/phone từ reg) → reg_info.contactpoint
		Srnonce:               finalRegCtx,
		// Check live/die kiểu PICTURE-FIRST (giống iOS): /picture endpoint trước → token sau.
		// Tránh hit /me?access_token bằng UA Chrome Windows ngay sau session Messenger-Android
		// (lệch vân tay → dễ checkpoint). Token vẫn là lưới fallback khi picture "Unknown".
		CheckLiveDieFunc: verifybase.CheckLiveDiePictureFirst,
		// CheckAddEmailSuccess: Messenger Bloks LUÔN có "error_message" field → false isExplicitError.
		// Đồng thời trích reg_context từ add-email response → update currentRegCtx cho buildConfirmBody.
		CheckAddEmailSuccess: func(resp string) bool {
			clean := strings.ReplaceAll(resp, "\\", "")
			// Trích reg_context mới từ add-email response → confirm dùng
			if rc := firstGroup(reLoginRegCtx, clean); rc != "" {
				currentRegCtx = rc
			}
			low := strings.ToLower(clean)
			hasBloks := strings.Contains(low, "fb_bloks_action") || strings.Contains(low, "action_bundle")
			hasSuccess := strings.Contains(clean, "Check your email") ||
				strings.Contains(clean, "CAA_REG_CONFIRMATION") ||
				strings.Contains(low, "confirmation_code")
			realError := strings.Contains(low, "email_already_used") ||
				strings.Contains(low, "email_is_invalid") ||
				strings.Contains(low, "rate_limit") ||
				strings.Contains(low, "too many requests") ||
				strings.Contains(low, "checkpoint") ||
				strings.Contains(low, "session is invalid") ||
				strings.Contains(low, "session has expired") ||
				strings.Contains(low, "account is currently disabled") ||
				strings.Contains(low, "\"errors\":[{")
			if realError {
				return false
			}
			return hasSuccess || hasBloks
		},
		CheckConfirmSuccess: func(resp string) bool {
			clean := strings.ReplaceAll(resp, "\\", "")
			snip := clean
			if len(snip) > 700 {
				snip = snip[:700]
			}
			notify("[AppMessV3 Verify] Confirm resp: " + snip)
			if strings.Contains(clean, "confirmation_success") {
				return true
			}
			low := strings.ToLower(clean)
			hasBloks := strings.Contains(low, "fb_bloks_action") || strings.Contains(low, "fb_bloks_app")
			bad := strings.Contains(low, "confirmation_failure") ||
				strings.Contains(low, "checkpoint") ||
				strings.Contains(low, "something went wrong") ||
				strings.Contains(low, "we're sorry")
			return hasBloks && !bad
		},
		FixUA: func(curUA, phone string) (string, string) {
			if strings.Contains(curUA, "FBAN/Orca-Android") {
				return "", ""
			}
			cc := verifybase.CountryFromPhone(phone)
			return randomOrcaUAVer(cc, vv.fbav, vv.fbbv), "UA regenerated (Orca)"
		},
		BuildHeaders: func(sc *verifybase.SessionCtx, friendlyName string, withZeroState bool) [][2]string {
			_ = withZeroState
			// V4: add-email/confirm dùng APP token Messenger (sessionless reg flow), KHÔNG user token.
			return MessengerActionHeaders(sc.UA, sc.DeviceID, sc.FamilyDevID, friendlyName)
		},
		// previousCP/birthday/encPwd lấy từ login response (capture V5: add-mail là CHANGE EMAIL,
		// cần state đầy đủ của account NVR đang pending confirmation).
		// QUAN TRỌNG: ép waterfall_id = renderWaterfallID (giống render bottomsheet/change.email)
		// thay vì waterfallID RunVerify sinh mới → CẢ flow 1 waterfall_id (khớp capture V5, server
		// track state nhất quán → confirm transition sang ProfilePhoto không fail).
		BuildAddEmailBody: func(spec *verifybase.Spec, emailAddr, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string {
			return buildAddEmailBody(spec, emailAddr, uid, firstName, lastName, deviceID, familyDevID, renderWaterfallID, machineID, locale, gender, sim, lctx.Contactpoint, lctx.Birthday, lctx.EncPwd, isEmailCP)
		},
		// BuildConfirmBody inline: đọc currentRegCtx (được update từ add-email response)
		// thay vì spec.Srnonce cố định → confirm dùng reg_context tươi nhất.
		BuildConfirmBody: func(spec *verifybase.Spec, emailAddr, code, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string {
			specCopy := *spec
			specCopy.Srnonce = currentRegCtx
			return buildConfirmBody(&specCopy, emailAddr, code, uid, firstName, lastName, deviceID, familyDevID, renderWaterfallID, machineID, locale, gender, sim, lctx.Birthday, lctx.EncPwd, isEmailCP)
		},
		BuildResendBody: func(spec *verifybase.Spec, emailAddr, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string {
			return buildResendBody(spec, emailAddr, uid, firstName, lastName, deviceID, familyDevID, renderWaterfallID, machineID, locale, gender, sim, lctx.Birthday, lctx.EncPwd)
		},
		Enable2FA:       enable2FAForAppMessV3,
		PostConfirm: func(ctx context.Context, sess *instagram.Session, cfg *instagram.VerifyConfig, notify func(string)) {
			if cfg.AddInfo != nil && cfg.AddInfo.Enabled {
				notify("[AppMessV3 Verify] Running AddInfo...")
				res := addinfo.RunAddInfo(ctx, sess, cfg.AddInfo, notify)
				if len(res.Notes) > 0 {
					notify(fmt.Sprintf("[AppMessV3 Verify] AddInfo done: %s", strings.Join(res.Notes, ", ")))
				}
			}
		},
	}
	return verifybase.RunVerify(ctx, session, cfg, outputPath, onStatus, spec)
}

func enable2FAForAppMessV3(ctx context.Context, session *instagram.Session, uid, machineID, deviceID string, emailOTPFn func(string, int) string, notify func(string)) (string, error) {
	sec := &androidsec.SecurityManager{EmailOTPFn: emailOTPFn}
	s := &instagram.Session{
		Token:     session.Token,
		UID:       uid,
		DeviceID:  deviceID,
		Datr:      machineID,
		UserAgent: session.UserAgent,
		Proxy:     session.Proxy,
		Password:  session.Password,
	}
	res, err := sec.Enable2FA(ctx, s)
	if err != nil {
		return "", err
	}
	notify(fmt.Sprintf("[AppMessV3 Verify] 2FA secret generated: %s", res.Secret))
	return res.Secret, nil
}

// genMachineID — 24-char base64url (18 random bytes), khớp format machine_id capture.
func genMachineID() string {
	b := make([]byte, 18)
	if _, err := crand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(mrand.Intn(256))
		}
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// ─── Orca (Messenger) User-Agent v530 ─────────────────────────────────────────

// RandomUA trả Messenger (Orca) Android UA v530.1.0.67.107 với DEVICE ĐA DẠNG từ
// fakeinfo.RandomDeviceProfile() (pool device thật Config/DeviceInfo/devices.txt — model/
// brand/OS/density/screen/arch/build random). CHỈ cố định app Messenger: FBAV/530.1.0.67.107,
// FBBV/814020040, FBPN/com.facebook.orca (bắt buộc vì gắn doc_id/bloks API). Giống hệt orcaUA
// bên reg → mỗi account 1 UA khác hẳn, nhất quán xuyên suốt login→add-email→confirm.
func RandomUA(countryCode string) string {
	return randomOrcaUAVer(countryCode, "", "")
}

// randomOrcaUAVer — như RandomUA nhưng FBAV/FBBV cố định nếu truyền vào (535/545).
// Rỗng → lấy ngẫu nhiên từ pool (mess_versions_and_builds_ver.txt) như bản 530.
func randomOrcaUAVer(countryCode, fbav, fbbv string) string {
	locale := fakeinfo.LocaleFromCountry(countryCode)
	if locale == "" {
		locale = "en_GB"
	}
	d := fakeinfo.RandomDeviceProfile()
	osv := d.OSVersion
	if osv == "" {
		osv = "14"
	}
	buildID := d.BuildID
	if buildID == "" {
		buildID = "AP3A.240905.015.A2"
	}
	arch := d.Architecture
	if arch == "" {
		arch = "arm64-v8a"
	}
	manuf := d.Manufacturer
	if manuf == "" {
		manuf = "samsung"
	}
	brand := d.Brand
	if brand == "" {
		brand = "samsung"
	}
	dens := d.Density
	if dens == "" {
		dens = "3.0"
	}
	w, h := d.ScreenWidth, d.ScreenHeight
	if w == 0 || h == 0 {
		w, h = 1080, 2340
	}
	if fbav == "" || fbbv == "" {
		fbav, fbbv = fakeinfo.RandomMessOrcaAppVersion("ver") // 530: ngẫu nhiên từ mess_versions_and_builds_ver.txt
	}
	return fmt.Sprintf(
		"Dalvik/2.1.0 (Linux; U; Android %s; %s Build/%s) "+
			"[FBAN/Orca-Android;FBAV/%s;FBPN/com.facebook.orca;FBLC/%s;"+
			"FBBV/%s;FBCR/null;FBMF/%s;FBBD/%s;FBDV/%s;FBSV/%s;"+
			"FBCA/%s:null;FBDM/{density=%s,width=%d,height=%d};FB_FW/1;]",
		osv, d.Model, buildID, fbav, locale, fbbv, manuf, brand, d.Model, osv, arch, dens, w, h,
	)
}

// ─── Messenger body builders (format {params:{...},} + nt_context tối giản) ────

// buildMessengerActionBody ghép form body action query Messenger theo capture V4.
func buildMessengerActionBody(appID, friendlyName, docID, bloksVer, locale string, clientInputParams, serverParams map[string]interface{}) string {
	if docID == "" {
		docID = verifyDocID
	}
	if bloksVer == "" {
		bloksVer = verifyBloksVer
	}
	inner := map[string]interface{}{"client_input_params": clientInputParams, "server_params": serverParams}
	// FB custom format: {params:{...},} — key params KHÔNG quote + dấu phẩy cuối.
	paramsStr := "{params:" + verifybase.MustJSON(inner) + ",}"
	variables := map[string]interface{}{
		"params": map[string]interface{}{
			"params":              paramsStr,
			"bloks_versioning_id": bloksVer,
			"app_id":              appID,
		},
		"scale": "3",
		"nt_context": map[string]interface{}{
			"is_flipper_enabled":           false,
			"theme_params":                 []interface{}{map[string]interface{}{"value": []string{}, "design_system_name": "FDS"}},
			"debug_tooling_metadata_token": nil,
		},
	}
	if locale == "" {
		locale = "en_GB"
	}
	form := url.Values{}
	form.Set("method", "post")
	form.Set("pretty", "false")
	form.Set("format", "json")
	form.Set("server_timestamps", "true")
	form.Set("locale", locale)
	form.Set("fb_api_req_friendly_name", friendlyName)
	form.Set("fb_api_caller_class", "graphservice")
	// client_doc_id PHẢI khớp version của bloks_versioning_id (capture: cả 2 đều per-version).
	// Trước đây hardcode verifyDocID (530) → mismatch với bloks_versioning_id per-version cho
	// mọi version ≠530 → server resolve lệch schema. Dùng docID đã truyền vào (= vv.docID).
	form.Set("client_doc_id", docID)
	form.Set("fb_api_client_context", `{"is_background":false}`)
	form.Set("variables", verifybase.MustJSON(variables))
	form.Set("fb_api_analytics_tags", `["GraphServices"]`)
	form.Set("client_trace_id", uuid.New().String())
	return form.Encode()
}

// buildMessengerRegInfo dựng reg_info CAA đầy đủ (field set capture V4) cho verify.
// Giá trị verify-only: name/email/gender/uid/machine_id có thật; encrypted_password,
// crypted_user_id, profile_photo, birthday = null (không có khi verify standalone).
// buildMessengerRegInfo dựng reg_info CAA đầy đủ.
// QUAN TRỌNG: contactpoint = contactpoint GỐC của account (từ reg/login response),
// KHÔNG phải email mới đang add. Client_input_params.email mới là email mới.
// origCP: contactpoint gốc (phone E.164 hoặc email từ reg). Nếu rỗng → dùng email mới.
func buildMessengerRegInfo(firstName, lastName, origCP, email, uid, deviceID, familyDevID, machineID, cryptedUID string, gender int, screenVisited []interface{}, birthday, encPwd string) string {
	fullName := strings.TrimSpace(firstName + " " + lastName)
	var cuid interface{}
	if cryptedUID != "" {
		cuid = cryptedUID
	}
	// reg_info.contactpoint = ORIGINAL contactpoint của account (không phải email mới)
	cp := origCP
	if cp == "" {
		cp = email // fallback nếu không biết original
	}
	// birthday + encrypted_password lấy từ login response (capture V5 14632 mang đủ state này).
	var bday interface{}
	if birthday != "" {
		bday = birthday
	}
	var enc interface{}
	if encPwd != "" {
		enc = encPwd
	}
	ri := map[string]interface{}{
		"first_name": firstName, "last_name": lastName, "full_name": fullName,
		"contactpoint": cp, "ar_contactpoint": nil, "contactpoint_type": "email",
		"is_using_unified_cp": nil, "unified_cp_screen_variant": nil,
		"is_cp_auto_confirmed": false, "is_cp_auto_confirmable": false, "is_cp_claimed": false,
		"confirmation_code": nil, "birthday": bday, "birthday_derived_from_age": nil,
		"age_range": "o18", "did_use_age": nil, "os_shared_age_range": nil,
		"gender": gender, "use_custom_gender": false, "custom_gender": nil,
		"encrypted_password": enc, "username": nil, "username_prefill": nil,
		"accounts_list_client": nil, "fb_conf_source": nil,
		"device_id": deviceID, "ig4a_qe_device_id": nil, "family_device_id": familyDevID,
		"fdid_available_on_start": nil, "fdid_rid_available_on_start": nil, "asdid_available_on_start": nil,
		"user_id": uid, "safetynet_token": nil, "skip_slow_rel_check": false, "safetynet_response": nil,
		"machine_id": machineID, "profile_photo": nil, "profile_photo_id": nil, "profile_photo_upload_id": nil,
		"avatar": nil, "email_oauth_token_no_contact_perm": nil, "email_oauth_token": nil, "email_oauth_tokens": nil,
		"sign_in_with_google_email": nil, "should_skip_two_step_conf": nil, "openid_tokens_for_testing": nil,
		"encrypted_msisdn": nil, "encrypted_msisdn_for_safetynet": nil, "cached_headers_safetynet_info": nil,
		"should_skip_headers_safetynet": nil, "headers_last_infra_flow_id": nil, "headers_last_infra_flow_id_safetynet": nil,
		"headers_flow_id": nil, "was_headers_prefill_available": nil, "sso_enabled": nil, "existing_accounts": nil,
		"used_ig_birthday": nil, "create_new_to_app_account": nil, "skip_session_info": nil,
		"ck_error": nil, "ck_id": nil, "ck_nonce": nil, "should_save_password": nil, "fb_access_token": nil,
		"is_msplit_reg": nil, "is_spectra_reg": nil, "dema_account_consent_given": nil, "spectra_entry_source": nil,
		"spectra_reg_token": nil, "spectra_reg_guardian_id": nil, "spectra_reg_guardian_logged_in_context": nil,
		"spectra_requester_user_id": nil, "user_id_of_msplit_creator": nil, "msplit_creator_nonce": nil,
		"dma_data_combination_consent_given": nil, "xapp_accounts": nil, "fb_device_id": nil, "fb_machine_id": nil,
		"ig_device_id": nil, "ig_machine_id": nil, "should_skip_nta_upsell": nil, "big_blue_token": nil,
		"caa_reg_flow_source": nil, "ig_authorization_token": nil, "full_sheet_flow": false, "crypted_user_id": cuid,
		"is_ca_late_teen": nil, "is_early_teen": nil, "is_caa_perf_enabled": false, "is_preform": true,
		"should_show_rel_error": false, "ignore_suma_check": false, "dismissed_login_upsell_with_cna": false,
		"ignore_existing_login": false, "ignore_existing_login_from_suma": false, "ignore_existing_login_after_errors": false,
		"suggested_first_name": nil, "suggested_last_name": nil, "suggested_full_name": nil, "frl_authorization_token": nil,
		"post_form_errors": nil, "skip_step_without_errors": false, "existing_account_exact_match_checked": false,
		"existing_account_fuzzy_match_checked": false, "email_oauth_exists": false, "confirmation_code_send_error": nil,
		"is_too_young": false, "source_account_type": nil, "whatsapp_installed_on_client": false,
		"confirmation_medium": nil, "source_credentials_type": nil, "source_cuid": nil, "source_account_reg_info": nil,
		"soap_creation_source": nil, "source_account_type_to_reg_info": nil, "registration_flow_id": "",
		"should_skip_youth_tos": false, "is_youth_regulation_flow_complete": false, "is_on_cold_start": false,
		"email_prefilled": false, "cp_confirmed_by_auto_conf": false, "in_sowa_experiment": false,
		"youth_regulation_config": nil, "conf_allow_back_nav_after_change_cp": true, "conf_bouncing_cliff_screen_type": nil,
		"conf_show_bouncing_cliff": nil, "eligible_to_flash_call_in_ig4a": false, "eligible_to_mo_sms_in_ig4a": false,
		"mo_sms_ent_id": nil, "flash_call_permissions_status": nil, "gms_incoming_call_retriever_eligibility": nil,
		"attestation_result": nil, "request_data_and_challenge_nonce_string": nil, "confirmed_cp_and_code": nil,
		"notification_callback_id": nil, "reg_suma_state": 0, "is_msplit_neutral_choice": false, "msg_previous_cp": nil,
		"ntp_import_source_info": nil, "youth_consent_decision_time": nil, "sk_pipa_consent_given": nil,
		"should_show_spi_before_conf": true, "google_oauth_account": nil, "is_reg_request_from_ig_suma": false,
		"is_toa_reg": false, "is_threads_public": false, "spc_import_flow": false, "caa_play_integrity_attestation_result": nil,
		"client_known_key_hash": nil, "flash_call_provider": nil, "is_in_gms_experience": nil,
		"flash_call_nonce_prefix_details": nil, "spc_birthday_input": false, "failed_birthday_year_count": nil,
		"user_presented_medium_source": nil, "user_opted_out_of_ntp": nil, "is_from_registration_reminder": false,
		"show_youth_reg_in_ig_spc": false, "fb_suma_is_high_confidence": nil, "screen_visited": screenVisited,
		"fb_email_login_upsell_skip_suma_post_tos": false, "fb_suma_is_from_email_login_upsell": false,
		"fb_suma_is_from_phone_login_upsell": false, "should_prefill_cp_in_ar": nil,
		"ig_partially_created_account_user_id": nil, "ig_partially_created_account_nonce": nil,
		"ig_partially_created_account_nonce_expiry": nil, "force_sessionless_nux_experience": false,
		"has_seen_suma_landing_page_pre_conf": false, "has_seen_suma_candidate_page_pre_conf": false,
		"has_seen_confirmation_screen": false, "suma_on_conf_threshold": -1, "should_show_error_msg": true,
		"th_profile_photo_token": nil, "attempted_silent_auth_in_fb": false, "attempted_silent_auth_in_ig": false,
		"sa_prefetch_callback_id": nil, "cp_suma_results_map": nil, "source_username": nil, "next_uri": nil,
		"should_use_next_uri": nil, "linking_entry_point": nil, "fb_encrypted_partial_new_account_properties": nil,
		"starter_pack_name": nil, "starter_pack_creator_user_ids": nil, "wa_data_bundle": nil, "wa_encrypted_auth_data": nil,
		"bloks_controller_source": nil, "airwave_registration_code": nil, "is_sessionless_nux": nil,
		"login_contactpoint": nil, "login_contactpoint_type": nil, "should_show_bday_after_name_suggestions": nil,
		"should_override_back_nav": false, "ig_footer_variant": "control", "device_network_info": nil,
		"is_from_web_lite_reg_controller": nil, "login_form_siwg_email": nil, "account_setup_waterfall_id": nil,
		"is_wanted_suma_user": nil, "device_zero_balance_state": nil, "wa_to_ig_merged_tos_variant": nil,
		"is_in_nta_single_form": false, "source_account_image_asset_id": nil, "passkey_eligible_device": nil,
		"nta_control_reason": nil, "nta_risk_type": nil, "nta_single_form_variant": nil, "enable_survey": nil,
		"phone_prefetch_outcome": nil, "tos_accepted_on_profile_info": nil,
	}
	return verifybase.MustJSON(ri)
}

func buildAddEmailBody(spec *verifybase.Spec, emailAddr, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile, previousCP, birthday, encPwd string, emailCP bool) string {
	_ = sim
	lat := int64(66000000000000 + mrand.Int63n(900000000000))
	regInfo := buildMessengerRegInfo(firstName, lastName, spec.Phone, emailAddr, uid, deviceID, familyDevID, machineID, spec.SessionlessCryptedUID, gender, []interface{}{"CAA_REG_CONFIRMATION_SCREEN"}, birthday, encPwd)

	// Reg-MAIL (email-CP): theo capture VerMailByregMail — msg_previous_cp RỖNG, login_surface
	// =login_home, login_entry_point=logged_out, KHÔNG gửi reg_context. Reg-phone giữ nguyên cũ.
	msgPrevCP := previousCP
	loginSurface, loginEntry := "registration", "registration"
	if emailCP {
		msgPrevCP = ""
		loginSurface, loginEntry = "login_home", "logged_out"
	}

	clientInputParams := map[string]interface{}{
		"aac": genAAC(), "device_id": deviceID, "zero_balance_state": "", "network_bssid": nil,
		"msg_previous_cp": msgPrevCP, "machine_id": "", "switch_cp_first_time_loading": 1, "has_rejected_rel": 0,
		"seen_login_upsell": 0, "accounts_list": []interface{}{}, "email_prefilled": 0,
		"confirmed_cp_and_code": map[string]interface{}{}, "family_device_id": familyDevID,
		"block_store_machine_id": "", "fb_ig_device_id": []interface{}{}, "lois_settings": map[string]interface{}{"lois_token": ""},
		"cloud_trust_token": nil, "is_from_device_emails": 0, "email": emailAddr, "switch_cp_have_seen_suma": 0,
	}
	serverParams := map[string]interface{}{
		"event_request_id": uuid.New().String(), "is_from_logged_out": 0, "text_input_id": lat,
		"layered_homepage_experiment_group": nil, "device_id": deviceID, "login_surface": loginSurface,
		"waterfall_id": waterfallID, "INTERNAL__latency_qpl_instance_id": lat + mrand.Int63n(500),
		"flow_info": `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`, "is_platform_login": 0,
		"login_entry_point": loginEntry, "INTERNAL__latency_qpl_marker_id": 36707139, "reg_info": regInfo,
		"family_device_id": familyDevID, "offline_experiment_group": "caa_iteration_v3_perf_msg_6",
		"cp_funnel": 1, "cp_source": 1, "access_flow_version": "pre_mt_behavior",
		"is_from_logged_in_switcher": 0, "current_step": 10,
	}
	if !emailCP {
		// reg-phone: giữ reg_context như cũ. reg-mail: capture KHÔNG gửi reg_context ở add-email.
		serverParams["reg_context"] = spec.Srnonce
	}
	body := buildMessengerActionBody("com.bloks.www.bloks.caa.reg.async.contactpoint_email.async", verifybase.AddEmailFriendlyName, spec.DocID, spec.BloksVer, locale, clientInputParams, serverParams)
	return body
}

func buildConfirmBody(spec *verifybase.Spec, emailAddr, code, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile, birthday, encPwd string, emailCP bool) string {
	_ = sim
	lat := int64(66000000000000 + mrand.Int63n(900000000000))
	// Reg-MAIL: capture confirm [1429] dùng reg_info.contactpoint = EMAIL MỚI (đang verify),
	// KHÔNG phải email gốc. Email gốc (gmail giả ở reg-mail) KHÔNG tồn tại → confirm với nó →
	// FB "Invalid email". Reg-phone: giữ origCP (phone gốc hợp lệ) như cũ.
	cpForConfirm := spec.Phone
	if emailCP {
		cpForConfirm = emailAddr
	}
	regInfo := buildMessengerRegInfo(firstName, lastName, cpForConfirm, emailAddr, uid, deviceID, familyDevID, machineID, spec.SessionlessCryptedUID, gender, []interface{}{"CAA_REG_CONFIRMATION_SCREEN", "CAA_REG_CONFIRMATION_SCREEN"}, birthday, encPwd)

	// Reg-MAIL: capture confirm [1429] dùng login_surface=login_home/logged_out + KHÔNG reg_context.
	loginSurface, loginEntry := "registration", "registration"
	if emailCP {
		loginSurface, loginEntry = "login_home", "logged_out"
	}

	clientInputParams := map[string]interface{}{
		"confirmed_cp_and_code": map[string]interface{}{}, "aac": genAAC(), "family_device_id": familyDevID,
		"block_store_machine_id": "", "code": code, "fb_ig_device_id": []interface{}{}, "device_id": deviceID,
		"lois_settings": map[string]interface{}{"lois_token": ""}, "cloud_trust_token": nil,
		"network_bssid": nil, "machine_id": "",
	}
	serverParams := map[string]interface{}{
		"event_request_id": uuid.New().String(), "is_from_logged_out": 0, "text_input_id": lat,
		"layered_homepage_experiment_group": nil, "device_id": deviceID, "login_surface": loginSurface,
		"waterfall_id": waterfallID, "wa_timer_id": "wa_retriever", "INTERNAL__latency_qpl_instance_id": lat + mrand.Int63n(500),
		"flow_info": `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`, "is_platform_login": 0,
		"sms_retriever_started_prior_step": 0, "login_entry_point": loginEntry, "INTERNAL__latency_qpl_marker_id": 36707139,
		"reg_info": regInfo, "family_device_id": familyDevID, "offline_experiment_group": "caa_iteration_v3_perf_msg_6",
		"access_flow_version": "pre_mt_behavior", "is_from_logged_in_switcher": 0, "current_step": 10,
	}
	if !emailCP {
		serverParams["reg_context"] = spec.Srnonce
	}
	body := buildMessengerActionBody("com.bloks.www.bloks.caa.reg.confirmation.async", verifybase.ConfirmFriendlyName, spec.DocID, spec.BloksVer, locale, clientInputParams, serverParams)
	return body
}

func buildResendBody(spec *verifybase.Spec, emailAddr, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile, birthday, encPwd string) string {
	_ = sim
	lat := int64(66000000000000 + mrand.Int63n(900000000000))
	regInfo := buildMessengerRegInfo(firstName, lastName, spec.Phone, emailAddr, uid, deviceID, familyDevID, machineID, spec.SessionlessCryptedUID, gender, []interface{}{"CAA_REG_CONFIRMATION_SCREEN"}, birthday, encPwd)

	clientInputParams := map[string]interface{}{
		"aac": genAAC(), "block_store_machine_id": "", "device_id": deviceID,
		"lois_settings": map[string]interface{}{"lois_token": ""}, "cloud_trust_token": nil,
		"network_bssid": nil, "machine_id": "", "family_device_id": familyDevID,
	}
	serverParams := map[string]interface{}{
		"is_from_logged_out": 0, "layered_homepage_experiment_group": nil, "device_id": deviceID,
		"login_surface": "registration", "waterfall_id": waterfallID, "INTERNAL__latency_qpl_instance_id": lat,
		"flow_info": `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`, "is_platform_login": 0,
		"login_entry_point": "registration", "INTERNAL__latency_qpl_marker_id": 36707139, "reg_info": regInfo,
		"family_device_id": familyDevID, "offline_experiment_group": "caa_iteration_v3_perf_msg_6",
		"access_flow_version": "pre_mt_behavior", "is_from_logged_in_switcher": 0, "current_step": 10,
		"reg_context": spec.Srnonce,
	}
	body := buildMessengerActionBody("com.bloks.www.bloks.caa.reg.resend_confirmation.async", verifybase.ResendFriendlyName, spec.DocID, spec.BloksVer, locale, clientInputParams, serverParams)
	return body
}
