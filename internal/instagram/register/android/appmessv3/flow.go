// flow.go — Luồng REG Messenger MULTI-STEP (bám capture DocWeMake/API/RegVerMess).
// Trình tự: aymh → name → birthday → email → password → create.account → (confirm OTP).
// Mỗi response trả reg_context (server-signed) feed bước kế. create.account mang
// reg_context đầy đủ (~10900 chars) — KHÁC single-shot reg_context:null trước đây.
package appmessv3

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/instagram"
	verlogin "HVRIns/internal/instagram/verify/android/appmessv3"
	"HVRIns/internal/proxy"

	"github.com/google/uuid"
)

const flowInfoJSON = `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`

const (
	appIDAymh        = "com.bloks.www.bloks.caa.reg.aymh_create_account_button.async"
	appIDName        = "com.bloks.www.bloks.caa.reg.name.async"
	appIDBirthday    = "com.bloks.www.bloks.caa.reg.birthday.async"
	appIDGender      = "com.bloks.www.bloks.caa.reg.gender.async"
	appIDEmail       = "com.bloks.www.bloks.caa.reg.async.contactpoint_email.async"
	appIDPassword    = "com.bloks.www.bloks.caa.reg.password.async"
	appIDCreate      = "com.bloks.www.bloks.caa.reg.create.account.async"
	appIDBottomsheet = "com.bloks.www.bloks.caa.reg.confirmation.fb.bottomsheet"
	appIDChangeEmail = "com.bloks.www.bloks.caa.reg.confirmation.change.email"
	appIDConfirm     = "com.bloks.www.bloks.caa.reg.confirmation.async"
)

// reRegContext bóc reg_context (blob base64url + đuôi |regX). Lấy match dài nhất.
var reRegContext = regexp.MustCompile(`[A-Za-z0-9_-]{200,}\|reg[a-z]`)

func extractRegContext(resp string) string {
	best := ""
	for _, m := range reRegContext.FindAllString(resp, -1) {
		if len(m) > len(best) {
			best = m
		}
	}
	return best
}

// regFlow giữ state xuyên suốt luồng multi-step.
type regFlow struct {
	sess        *session
	profile     AppMV3Profile
	notify      func(string)
	locale      string
	deviceID    string
	familyID    string
	waterfallID string
	machineID   string
	aac         string
	encPassword string
	email       string
	regContext  string
}

func qplInstance() int64 { return 57000000000000 + rand.Int63n(900000000000) }

func loisEmpty() map[string]interface{} { return map[string]interface{}{"lois_token": ""} }

func (f *regFlow) regInfo() string {
	return buildCreateRegInfo(f.profile, f.encPassword, f.email, "email", f.deviceID, f.familyID, f.machineID)
}

// commonServer dựng server_params chung cho các bước có reg_context + reg_info.
func (f *regFlow) commonServer(currentStep int, extra map[string]interface{}) map[string]interface{} {
	sp := map[string]interface{}{
		"is_from_logged_out":                0,
		"layered_homepage_experiment_group": nil,
		"device_id":                         f.deviceID,
		"login_surface":                     "login_home",
		"waterfall_id":                      f.waterfallID,
		"INTERNAL__latency_qpl_instance_id": qplInstance(),
		"flow_info":                         flowInfoJSON,
		"is_platform_login":                 0,
		"login_entry_point":                 "logged_out",
		"INTERNAL__latency_qpl_marker_id":   36707139,
		"reg_info":                          f.regInfo(),
		"family_device_id":                  f.familyID,
		"offline_experiment_group":          "caa_iteration_v3_perf_msg_6",
		"access_flow_version":               "pre_mt_behavior",
		"is_from_logged_in_switcher":        0,
		"current_step":                      currentStep,
	}
	if f.regContext != "" {
		sp["reg_context"] = f.regContext
	}
	for k, v := range extra {
		sp[k] = v
	}
	return sp
}

// post gửi 1 bước → cập nhật regContext từ response → trả body.
func (f *regFlow) post(ctx context.Context, label, appID string, sp, cip map[string]interface{}) (string, error) {
	return f.postDoc(ctx, label, appID, "FbBloksActionRootQuery-", f.profile.DocID, sp, cip)
}

// postRender — AppRootQuery render (welcome/confirmation screens) dùng render doc_id.
func (f *regFlow) postRender(ctx context.Context, label, appID string, sp, cip map[string]interface{}) (string, error) {
	return f.postDoc(ctx, label, appID, "FbBloksAppRootQuery-", f.profile.RenderDocID, sp, cip)
}

func (f *regFlow) postDoc(ctx context.Context, label, appID, friendlyPrefix, docID string, sp, cip map[string]interface{}) (string, error) {
	body := buildStepBody(appID, friendlyPrefix, docID, f.profile.BloksVer, f.locale, sp, cip)
	headers := buildHeaders(f.profile)
	resp, err := f.sess.postGzip(ctx, instagram.BaseURLBGraph+"/graphql", body, headers)
	if err != nil {
		return resp, fmt.Errorf("%s: %w", label, err)
	}
	if rc := extractRegContext(resp); rc != "" {
		f.regContext = rc
	}
	f.notify(fmt.Sprintf("[appmv3] step %-9s ✓ resp=%d bytes rc=%d", label, len(resp), len(f.regContext)))
	return resp, nil
}

func (f *regFlow) baseClient() map[string]interface{} {
	return map[string]interface{}{
		"aac": f.aac, "block_store_machine_id": "", "lois_settings": loisEmpty(),
		"cloud_trust_token": nil, "zero_balance_state": "", "network_bssid": nil, "machine_id": "",
	}
}

// ─── Steps ───────────────────────────────────────────────────────────────────

func (f *regFlow) stepAymh(ctx context.Context) error {
	sp := map[string]interface{}{
		"is_from_logged_out": 0, "layered_homepage_experiment_group": nil,
		"should_expand_layered_bottom_sheet": 0, "is_from_lid_welcome_screen": 0,
		"device_id": f.deviceID, "should_show_wa_nta_bottom_sheet": 0,
		"login_surface": "login_home", "waterfall_id": f.waterfallID,
		"INTERNAL__latency_qpl_instance_id": qplInstance(),
		"reg_flow_source":                   "aymh_multi_profiles_native_integration_point",
		"is_caa_perf_enabled":               1, "is_platform_login": 0,
		"login_entry_point": "logged_out", "INTERNAL__latency_qpl_marker_id": 36707139,
		"family_device_id": f.familyID, "offline_experiment_group": "caa_iteration_v3_perf_msg_6",
		"entrypoint": "login_home_async", "event_step": "landing",
		"access_flow_version": "pre_mt_behavior", "is_eligible_for_igds_sac_reg_flow": 0,
		"is_from_logged_in_switcher": 0,
	}
	cip := map[string]interface{}{
		"should_show_nested_nta_bottom_sheet": 0, "accounts_list": []interface{}{},
		"username_input": "", "aac": f.aac, "block_store_machine_id": "",
		"lois_settings": loisEmpty(), "cloud_trust_token": nil, "zero_balance_state": "",
		"network_bssid": nil, "machine_id": "",
	}
	_, err := f.post(ctx, "aymh", appIDAymh, sp, cip)
	return err
}

func (f *regFlow) stepName(ctx context.Context) error {
	sp := f.commonServer(1, map[string]interface{}{
		"event_request_id": uuid.New().String(), "flow_modifier": flowInfoJSON,
	})
	cip := map[string]interface{}{
		"google_id_email": "", "device_network_info": nil, "firstname": f.profile.FirstName,
		"aac": f.aac, "block_store_machine_id": "", "lois_settings": loisEmpty(),
		"cloud_trust_token": nil, "zero_balance_state": "", "network_bssid": nil,
		"google_id_token": "", "machine_id": "", "lastname": f.profile.LastName,
	}
	_, err := f.post(ctx, "name", appIDName, sp, cip)
	return err
}

func (f *regFlow) stepBirthday(ctx context.Context) error {
	bdayStr, bdayTs := birthdayParts(f.profile.Birthday)
	sp := f.commonServer(2, nil)
	cip := map[string]interface{}{
		"client_timezone": "Asia/Ho_Chi_Minh", "aac": f.aac,
		"birthday_or_current_date_string": bdayStr, "zero_balance_state": "",
		"network_bssid": nil, "should_skip_youth_tos": 0, "machine_id": "",
		"accounts_list": []interface{}{}, "os_age_range": "", "block_store_machine_id": "",
		"birthday_timestamp": bdayTs, "lois_settings": loisEmpty(),
		"cloud_trust_token": nil, "is_youth_regulation_flow_complete": 0,
	}
	_, err := f.post(ctx, "birthday", appIDBirthday, sp, cip)
	return err
}

// stepGender (V5: bước step 3 MỚI giữa birthday→email). Submit gender + pronoun.
func (f *regFlow) stepGender(ctx context.Context) error {
	sp := f.commonServer(3, nil)
	gender := f.profile.Gender
	if gender == 0 {
		gender = 2
	}
	cip := map[string]interface{}{
		"aac": f.aac, "device_emails": []interface{}{}, "block_store_machine_id": "",
		"gender": gender, "pronoun": 0, "lois_settings": loisEmpty(),
		"cloud_trust_token": nil, "zero_balance_state": "", "network_bssid": nil,
		"device_phone_numbers": []interface{}{}, "machine_id": "", "custom_gender": "",
	}
	_, err := f.post(ctx, "gender", appIDGender, sp, cip)
	return err
}

func (f *regFlow) stepEmail(ctx context.Context) error {
	sp := f.commonServer(4, map[string]interface{}{
		"event_request_id": uuid.New().String(), "text_input_id": qplInstance(),
		"cp_funnel": 0, "cp_source": 0,
	})
	cip := map[string]interface{}{
		"aac": f.aac, "device_id": f.deviceID, "zero_balance_state": "",
		"network_bssid": nil, "msg_previous_cp": "", "machine_id": "",
		"switch_cp_first_time_loading": 1, "has_rejected_rel": 0, "seen_login_upsell": 0,
		"accounts_list": []interface{}{}, "email_prefilled": 0, "confirmed_cp_and_code": map[string]interface{}{},
		"family_device_id": f.familyID, "block_store_machine_id": "", "fb_ig_device_id": []interface{}{},
		"lois_settings": loisEmpty(), "cloud_trust_token": nil, "is_from_device_emails": 0,
		"email": f.email, "switch_cp_have_seen_suma": 0,
	}
	_, err := f.post(ctx, "email", appIDEmail, sp, cip)
	return err
}

func (f *regFlow) stepPassword(ctx context.Context) error {
	sp := f.commonServer(5, map[string]interface{}{
		"event_request_id": uuid.New().String(), "flow_modifier": flowInfoJSON,
	})
	cip := map[string]interface{}{
		"spi_action": 1, "safetynet_response": nil, "caa_play_integrity_attestation_result": "",
		"aac": f.aac, "safetynet_token": genSafetynetToken(), "whatsapp_installed_on_client": 0,
		"zero_balance_state": "", "network_bssid": nil,
		"attestation_result": map[string]interface{}{"errorMessage": "KeyAttestationException: No key found!"},
		"machine_id":         "", "headers_last_infra_flow_id_safetynet": "", "has_rejected_rel": 0,
		"email_oauth_token_map": map[string]interface{}{}, "block_store_machine_id": "",
		"fb_ig_device_id": []interface{}{}, "encrypted_msisdn_for_safetynet": "",
		"lois_settings": loisEmpty(), "cloud_trust_token": nil, "client_known_key_hash": "",
		"encrypted_password": f.encPassword,
	}
	_, err := f.post(ctx, "password", appIDPassword, sp, cip)
	return err
}

func (f *regFlow) stepCreate(ctx context.Context) (string, error) {
	sp := f.commonServer(8, map[string]interface{}{
		"event_request_id": uuid.New().String(), "bloks_controller_source": "bk_caa_reg_icon_text_list_tos_screen",
	})
	cip := map[string]interface{}{
		"ck_error": "", "aac": f.aac, "device_id": f.deviceID, "waterfall_id": f.waterfallID,
		"zero_balance_state": "", "network_bssid": nil, "failed_birthday_year_count": "",
		"headers_last_infra_flow_id": "", "ig_partially_created_account_nonce_expiry": 0,
		"machine_id": "", "reached_from_tos_screen": 1, "ig_partially_created_account_nonce": "",
		"block_store_machine_id": "", "ck_nonce": "", "lois_settings": loisEmpty(),
		"ig_partially_created_account_user_id": 0, "cloud_trust_token": nil, "ck_id": "",
		"no_contact_perm_email_oauth_token": "", "encrypted_msisdn": "",
	}
	return f.post(ctx, "create", appIDCreate, sp, cip)
}

// stepEmailConfirmStage — bước 1274: re-submit contactpoint ở confirmation stage
// (cp_funnel:1, cp_source:1, login_surface:registration, current_step:10, KHÔNG reg_context).
// Bắt buộc trước confirm.async để FB nhận code (thiếu → confirm "try again").

// stepBottomsheet — render confirmation bottomsheet (V5 [14627]) → đẩy server state.
func (f *regFlow) stepBottomsheet(ctx context.Context) (string, error) {
	sp := f.commonServer(10, map[string]interface{}{
		"aac": f.aac, "login_surface": "registration", "login_entry_point": "registration",
		"trigger": "default", "timer_id": "wa_retriever", "zero_tap_enabled": 0,
	})
	cip := map[string]interface{}{"lois_settings": loisEmpty()}
	return f.postRender(ctx, "bottomsheet", appIDBottomsheet, sp, cip)
}

// stepChangeEmail — render change.email screen (V5 [14631]). KHÔNG reg_context.
func (f *regFlow) stepChangeEmail(ctx context.Context) (string, error) {
	sp := f.commonServer(10, map[string]interface{}{
		"login_surface": "registration", "login_entry_point": "registration",
		"INTERNAL_INFRA_screen_id": "CAA_REG_CONFIRMATION_CHANGE_EMAIL",
	})
	delete(sp, "reg_context")
	cip := map[string]interface{}{"lois_settings": loisEmpty()}
	return f.postRender(ctx, "change-email", appIDChangeEmail, sp, cip)
}

func (f *regFlow) stepEmailConfirmStage(ctx context.Context) (string, error) {
	sp := f.commonServer(10, map[string]interface{}{
		"event_request_id": uuid.New().String(), "text_input_id": qplInstance(),
		"login_surface": "registration", "login_entry_point": "registration",
		"cp_funnel": 1, "cp_source": 1,
	})
	delete(sp, "reg_context")
	cip := map[string]interface{}{
		"aac": f.aac, "device_id": f.deviceID, "zero_balance_state": "",
		"network_bssid": nil, "msg_previous_cp": "", "machine_id": "",
		"switch_cp_first_time_loading": 0, "has_rejected_rel": 0, "seen_login_upsell": 0,
		"accounts_list": []interface{}{}, "email_prefilled": 0, "confirmed_cp_and_code": map[string]interface{}{},
		"family_device_id": f.familyID, "block_store_machine_id": "", "fb_ig_device_id": []interface{}{},
		"lois_settings": loisEmpty(), "cloud_trust_token": nil, "is_from_device_emails": 0,
		"email": f.email, "switch_cp_have_seen_suma": 0,
	}
	return f.post(ctx, "cp-confirm", appIDEmail, sp, cip)
}

// confirmSucceeded — confirm OTP THẬT SỰ qua khi chuyển sang profilephoto, KHÔNG có
// "try again"/"transition_failure" (so capture: success→profilephoto, fail→try again).
func confirmSucceeded(resp string) bool {
	if strings.Contains(resp, "try again") || strings.Contains(resp, "transition_failure") {
		return false
	}
	return strings.Contains(resp, "profilephoto") ||
		strings.Contains(resp, "registration.profile") ||
		strings.Contains(resp, "transition_success")
}

// stepConfirm gửi OTP code (bước 10). Capture: confirm KHÔNG mang reg_context.
func (f *regFlow) stepConfirm(ctx context.Context, code string) (string, error) {
	sp := f.commonServer(10, map[string]interface{}{
		"event_request_id": uuid.New().String(), "text_input_id": qplInstance(),
		"login_surface": "registration", "login_entry_point": "registration",
		"wa_timer_id": "wa_retriever", "sms_retriever_started_prior_step": 0,
	})
	delete(sp, "reg_context") // confirm dùng reg_info + code, không có reg_context
	cip := map[string]interface{}{
		"confirmed_cp_and_code": map[string]interface{}{}, "aac": f.aac,
		"family_device_id": f.familyID, "block_store_machine_id": "", "code": code,
		"fb_ig_device_id": []interface{}{}, "device_id": f.deviceID,
		"lois_settings": loisEmpty(), "cloud_trust_token": nil, "network_bssid": nil, "machine_id": "",
	}
	return f.post(ctx, "confirm", appIDConfirm, sp, cip)
}

// runMultiStepReg chạy aymh→...→create.account (retry nếu bị "couldn't create").
// Trả về response của create.account để parse uid/token.
func runMultiStepReg(ctx context.Context, f *regFlow, createRetries int) (string, error) {
	steps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"aymh", f.stepAymh},
		{"name", f.stepName},
		{"birthday", f.stepBirthday},
		{"gender", f.stepGender},
		{"email", f.stepEmail},
		{"password", f.stepPassword},
	}
	for _, s := range steps {
		if err := s.fn(ctx); err != nil {
			return "", err
		}
		if f.regContext == "" {
			return "", fmt.Errorf("step %s: không lấy được reg_context (flow đứt)", s.name)
		}
		sleep(700 + rand.Intn(900))
	}

	var resp string
	var err error
	for i := 0; i < createRetries; i++ {
		resp, err = f.stepCreate(ctx)
		if err != nil {
			return resp, err
		}
		if !strings.Contains(resp, "create an account for you") {
			return resp, nil // không bị block → xong
		}
		f.notify(fmt.Sprintf("[appmv3] create.account bị block (lần %d/%d), retry...", i+1, createRetries))
		sleep(1500 + rand.Intn(1500))
	}
	return resp, nil
}

// RegisterEmailFlowWithOTP — entry cho test/CLI: build profile rồi gọi core.
func RegisterEmailFlowWithOTP(ctx context.Context, proxyStr, email, password, datr, countryCode string, getOTP func(context.Context) (string, error), notify func(string)) *instagram.RegResult {
	profile := BuildProfileForPlatform("appmv3", countryCode)
	if datr != "" {
		profile.MachineID = datr
	}
	return RunEmailRegFlow(ctx, proxyStr, profile, email, password, getOTP, 25, notify)
}

// RunEmailRegFlow — CORE: multi-step (retry CẢ FLOW, session/IP mới mỗi lần vì
// create.account 300043 là xác suất theo IP) + confirm OTP nếu getOTP != nil.
//   - getOTP == nil → dừng sau create-pass (account TẠO, pending OTP) → Success, ver confirm sau.
//   - getOTP != nil → đọc OTP + confirm → Success kèm uid (best-effort).
func RunEmailRegFlow(ctx context.Context, proxyStr string, baseProfile AppMV3Profile, email, password string, getOTP func(context.Context) (string, error), flowAttempts int, notify func(string)) *instagram.RegResult {
	profile := baseProfile
	locale := profile.Locale
	if locale == "" {
		locale = "en_GB"
	}
	if !strings.Contains(profile.AppMV3UA, "FBAN/Orca-Android") {
		profile.AppMV3UA = orcaUA(locale, profile)
	}
	machineID := profile.MachineID
	if machineID == "" {
		machineID = genMachineID()
	}
	ts := time.Now().Unix()
	encPassword := fmt.Sprintf("#PWD_MSGR:0:%d:%s", ts, password)
	if flowAttempts < 1 {
		flowAttempts = 1
	}

	for attempt := 1; attempt <= flowAttempts; attempt++ {
		// Render session MỚI mỗi attempt → IP MỚI (proxy -zone-/-session- xoay).
		// Bắt buộc: create.account 300043 theo IP; cùng IP retry đều fail.
		attemptProxy := proxy.RenderSessionIfIsProxyServer(proxyStr)
		sess, serr := newSession(attemptProxy)
		if serr != nil {
			notify(fmt.Sprintf("[appmv3] session attempt %d lỗi: %v", attempt, serr))
			continue
		}
		// Xoay datr MỚI mỗi attempt (usage-aware) → tránh 1 datr bị dùng quá nhiều
		// lần → flag → block. App làm vậy; thiếu = CLI 1 datr cố định bị flag.
		attemptMachineID := machineID
		if SharedPool != nil && SharedPool.Size() > 0 {
			if d := SharedPool.GetNext(0); d != "" {
				attemptMachineID = d
			}
		}
		attemptProfile := profile
		attemptProfile.MachineID = attemptMachineID
		flow := &regFlow{
			sess: sess, profile: attemptProfile, notify: notify, locale: locale,
			deviceID: attemptProfile.DeviceID, familyID: attemptProfile.FamilyDeviceID,
			waterfallID: uuid.New().String(), machineID: attemptMachineID,
			aac: genAAC(), encPassword: encPassword, email: email,
		}
		notify(fmt.Sprintf("[appmv3] ═══ Flow %d/%d (%s %s | %s) ═══",
			attempt, flowAttempts, profile.FirstName, profile.LastName, email))

		createResp, ferr := runMultiStepReg(ctx, flow, 1)
		if ferr != nil {
			notify(fmt.Sprintf("[appmv3] flow err: %v", ferr))
			sess.client.CloseIdleConnections()
			continue
		}
		if strings.Contains(createResp, "create an account for you") {
			notify(fmt.Sprintf("[appmv3] flow %d: create blocked (300043) → IP khác...", attempt))
			sess.client.CloseIdleConnections()
			sleep(2000 + rand.Intn(2000))
			continue
		}

		notify("[appmv3] ✓✓ create.account QUA — account đã tạo")

		res := &instagram.RegResult{
			Success: true, Email: email, Password: password, UserAgent: profile.AppMV3UA,
			DeviceID: profile.DeviceID, FamilyDeviceID: profile.FamilyDeviceID,
		}
		if getOTP == nil {
			sess.client.CloseIdleConnections()
			res.Message = "Account TẠO ĐƯỢC (pending OTP) — ver sẽ confirm + login"
			return res
		}

		// Bước 1274 (capture): re-submit contactpoint ở confirmation stage — set up
		// state cho confirm.async nhận code (thiếu bước này → confirm "try again").
		notify("[appmv3] confirmation stage (render bottomsheet + change.email + cp re-submit)...")
		_, _ = flow.stepBottomsheet(ctx)
		sleep(500 + rand.Intn(500))
		_, _ = flow.stepChangeEmail(ctx)
		sleep(500 + rand.Intn(500))
		_, _ = flow.stepEmailConfirmStage(ctx)
		sleep(1000 + rand.Intn(800))

		notify("[appmv3] chờ OTP từ mail...")
		code, oerr := getOTP(ctx)
		if oerr != nil || code == "" {
			sess.client.CloseIdleConnections()
			return &instagram.RegResult{Success: false, Email: email, Password: password,
				Message: fmt.Sprintf("Account ĐÃ TẠO nhưng OTP không về (%v)", oerr)}
		}

		// Confirm OTP — retry tối đa 3 lần, CHỈ tính qua khi confirmSucceeded (→ profilephoto).
		var confResp string
		confirmed := false
		for ci := 0; ci < 3; ci++ {
			notify(fmt.Sprintf("[appmv3] OTP=%s → confirm (lần %d)...", code, ci+1))
			r, cerr := flow.stepConfirm(ctx, code)
			if cerr != nil {
				notify(fmt.Sprintf("[appmv3] confirm HTTP err: %v", cerr))
				break
			}
			confResp = r
			if confirmSucceeded(confResp) {
				confirmed = true
				break
			}
			notify("[appmv3] confirm chưa qua (try again) — đọc lại mã + thử lại...")
			sleep(4000)
			if nc, _ := getOTP(ctx); nc != "" {
				code = nc
			}
		}
		sess.client.CloseIdleConnections()

		res.UID = extractAccountUID(confResp)
		if confirmed {
			notify("[appmv3] ✓ confirm OTP QUA (email verified → profilephoto)")
		} else {
			notify("[appmv3] ⚠ confirm OTP CHƯA qua — account tạo nhưng email CHƯA verify (NVR)")
		}

		// LOGIN PHASE — login email+password → uid + EAAD token + cookie.
		notify("[appmv3] login lấy token+cookie...")
		sleep(1500 + rand.Intn(1500))
		fullName := strings.TrimSpace(profile.FirstName + " " + profile.LastName)
		loginCtx, lerr := verlogin.FetchMessengerSession(ctx, email, password, "", fullName,
			profile.DeviceID, profile.FamilyDeviceID, machineID, profile.AppMV3UA, locale, attemptProxy, notify)
		cfTag := "NVR"
		if confirmed {
			cfTag = "confirmed"
		}
		if lerr == nil && loginCtx.Token != "" {
			res.AccessToken = loginCtx.Token
			res.Cookie = loginCtx.Cookie
			if u := uidFromCookie(loginCtx.Cookie); u != "" {
				res.UID = u
			}
			res.Message = fmt.Sprintf("Register+Login OK (%s) — UID: %s", cfTag, res.UID)
			notify(fmt.Sprintf("[appmv3] ✓✓✓ LOGIN OK uid=%s token=%s (%s)", res.UID, safeShort(loginCtx.Token, 16), cfTag))
		} else {
			res.Message = fmt.Sprintf("Account tạo (%s), login chưa ra token (%v)", cfTag, lerr)
			notify(fmt.Sprintf("[appmv3] login chưa ra token (%v)", lerr))
		}
		return res
	}
	return &instagram.RegResult{Success: false, Email: email, Password: password,
		Message: fmt.Sprintf("create.account blocked (300043) sau %d flow — proxy bị gate", flowAttempts)}
}

// uidFromCookie bóc c_user (= uid) từ cookie string.
func uidFromCookie(cookie string) string {
	if m := regexp.MustCompile(`c_user=(\d+)`).FindStringSubmatch(cookie); len(m) > 1 {
		return m[1]
	}
	return ""
}

// extractAccountUID best-effort bóc uid account mới từ confirm response (Bloks).
func extractAccountUID(resp string) string {
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`"user_id\\*"\s*:\s*\\*"?(\d{8,17})`),
		regexp.MustCompile(`"uid\\*"\s*:\s*\\*"?(\d{8,17})`),
		regexp.MustCompile(`account_id\\*"?\s*:\s*\\*"?(\d{8,17})`),
		regexp.MustCompile(`c_user\D{0,6}(\d{8,17})`),
	} {
		if m := re.FindStringSubmatch(resp); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

// birthdayParts đổi "DD-MM-YYYY" → (string, unix_ts).
func birthdayParts(b string) (string, int64) {
	t, err := time.Parse("02-01-2006", b)
	if err != nil {
		return b, 983411136
	}
	return b, t.Unix()
}
