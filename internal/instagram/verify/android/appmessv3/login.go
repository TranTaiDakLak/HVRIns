// login.go — Messenger (Orca) CAA login: send_login_request đăng nhập account bằng
// password (#PWD_MSGR) + contact_point, lấy session EAAD + session_cookies.
//
// Mô phỏng capture FlowRegVerFb_AppMess (Orca v529/v530) [send_login_request]:
//
//	POST https://b-graph.facebook.com/graphql
//	app_id = com.bloks.www.bloks.caa.login.async.send_login_request
//	authorization: OAuth <app-token Messenger 256002347743983|...>
//	credential chính: server_params.credential_type="password",
//	  client_input_params.contact_point=<uid>, password="#PWD_MSGR:0:ts:<plaintext>"
//	aymh_accounts/accounts_list/sso_token_map = context (facebook_local_auth token EAAAAU)
//	→ response Bloks nhúng: {access_token:"EAAD...", session_key, session_cookies:[c_user,xs,fr,datr]}
//
// QUAN TRỌNG (fix sau capture 2026-06-04):
//   - variables.params.params PHẢI là chuỗi format FB "{params:{...},}" (key params KHÔNG
//     quote + dấu phẩy cuối), KHÔNG phải JSON chuẩn → nếu sai FB trả #100 "Neither query_id".
//   - nt_context tối giản (chỉ is_flipper_enabled + theme_params FDS + debug token).
//   - form body KHÔNG có purpose; fb_api_client_context dùng false (không phải "0").
package appmessv3

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	mrand "math/rand"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"HVRIns/internal/instagram/verify/verifybase"
)

const (
	loginAppID        = "com.bloks.www.bloks.caa.login.async.send_login_request"
	loginFriendlyName = "FbBloksActionRootQuery-" + loginAppID
	loginDocID        = "11994080426288799937543572098"
	loginBloksVer     = "dadbbd68d34735f7a39b791542ad0ecd1b257eddf3e70ab790d47b3cedd8b093"

	// App-token Messenger (Orca) — public client token, dùng ở authorization khi login.
	messengerAppToken = "256002347743983|374e60f8b9bb6b8cbb30f78030438895"
	messengerProduct  = "256002347743983"
)

var (
	// reLoginToken — access_token EAAD trong response (đã strip backslash).
	reLoginToken         = regexp.MustCompile(`"access_token":"(EAA[A-Za-z0-9_-]{20,})"`)
	reLoginCryptedID     = regexp.MustCompile(`"crypted_user_id":"([A-Za-z0-9_-]{15,})"`)
	reLoginRegCtx        = regexp.MustCompile(`([A-Za-z0-9_-]{20,}\|regm)`)
	reLoginRegInfo       = regexp.MustCompile(`"reg_info"\s*:\s*"(\{[^"]{100,})"`)
	reLoginBirthday      = regexp.MustCompile(`"birthday"\s*:\s*"(\d{2}-\d{2}-\d{4})"`)
	reLoginMachineID     = regexp.MustCompile(`"machine_id"\s*:\s*"([A-Za-z0-9_-]{10,})"`)
	reLoginEncPwd        = regexp.MustCompile(`"encrypted_password"\s*:\s*"(#PWD_[^"]{20,})"`)
	reLoginFirstName     = regexp.MustCompile(`"first_name"\s*:\s*"([^"]{1,50})"`)
	reLoginLastName      = regexp.MustCompile(`"last_name"\s*:\s*"([^"]{1,50})"`)
	reLoginProfilePhoto  = regexp.MustCompile(`"profile_photo"\s*:\s*"(https://[^"]{20,})"`)
	reLoginWaterfallID   = regexp.MustCompile(`"waterfall_id"\s*:\s*"([a-fA-F0-9-]{30,})"`)
	reLoginContactpoint  = regexp.MustCompile(`"contactpoint"\s*:\s*"([^"]{3,})"`)
	reLoginContactpointT = regexp.MustCompile(`"contactpoint_type"\s*:\s*"([^"]+)"`)
	reLoginCUser         = regexp.MustCompile(`"name":"c_user","value":"(\d{6,})"`)
	reLoginXS            = regexp.MustCompile(`"name":"xs","value":"([^"]+)"`)
	reLoginFR            = regexp.MustCompile(`"name":"fr","value":"([^"]+)"`)
	reLoginDATR          = regexp.MustCompile(`"name":"datr","value":"([^"]+)"`)
)

// FetchMessengerSession đăng nhập account (uid + password plaintext) qua send_login_request,
// trả (accessToken EAAD, cookie "datr=..;c_user=..;xs=..;fr=..;", error).
// localAuthToken (EAAAAU) đi vào aymh context; credential chính là password.
func FetchMessengerSession(ctx context.Context, uid, password, localAuthToken, fullName, deviceID, familyDeviceID, machineID, ua, locale, proxyStr string, notify func(string)) (LoginContext, error) {
	log := func(m string) {
		if notify != nil {
			notify(m)
		}
	}
	uid = strings.TrimSpace(uid)
	password = strings.TrimSpace(password)
	var zero LoginContext
	if uid == "" || password == "" {
		return zero, fmt.Errorf("FetchMessengerSession: thiếu uid hoặc password")
	}
	if locale == "" {
		locale = "en_GB"
	}

	client, err := verifybase.CreateClient(proxyStr)
	if err != nil {
		return zero, fmt.Errorf("create client: %w", err)
	}
	defer client.CloseIdleConnections()

	body := buildSendLoginBody(uid, password, localAuthToken, fullName, deviceID, familyDeviceID, machineID, locale)
	headers := MessengerActionHeaders(ua, deviceID, familyDeviceID, loginFriendlyName)

	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return zero, ctx.Err()
			case <-time.After(time.Duration(attempt-1) * 2 * time.Second):
			}
			log(fmt.Sprintf("[AppMessV3 Login] Retry %d/%d...", attempt, maxAttempts))
		}
		log(fmt.Sprintf("[AppMessV3 Login] POST send_login_request (uid=%s, lần %d)...", uid, attempt))

		postCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		resp, perr := verifybase.DoPost(postCtx, client, verifybase.BgraphURL, body, headers)
		cancel()
		if perr != nil && resp == "" {
			lastErr = fmt.Errorf("HTTP login: %w", perr)
			continue
		}

		lctx := parseLoginResponse(resp)
		if lctx.Token != "" {
			log(fmt.Sprintf("[AppMessV3 Login] OK — token=%.16s... cookie=%t cuid=%t regctx=%t (lần %d)",
				lctx.Token, lctx.Cookie != "", lctx.CryptedUID != "", lctx.RegContext != "", attempt))
			return lctx, nil
		}

		clean := strings.ReplaceAll(resp, "\\", "")
		low := strings.ToLower(clean)
		switch {
		case strings.Contains(low, "checkpoint"):
			return zero, fmt.Errorf("login bị checkpoint")
		case strings.Contains(low, "incorrect_password") || strings.Contains(low, "wrong_password") || strings.Contains(low, "incorrect password"):
			return zero, fmt.Errorf("login sai mật khẩu")
		case strings.Contains(low, "login_blocked") || strings.Contains(low, "account is disabled"):
			return zero, fmt.Errorf("login bị chặn/account disabled")
		}
		snip := clean
		if len(snip) > 600 {
			snip = snip[:600]
		}
		log(fmt.Sprintf("[AppMessV3 Login] resp KHÔNG có access_token (%d bytes): %s", len(resp), snip))
		lastErr = fmt.Errorf("login không trả access_token (resp %d bytes)", len(resp))
	}
	return zero, fmt.Errorf("login thất bại sau %d lần: %w", maxAttempts, lastErr)
}

// LoginContext holds all state extracted from the login response Bloks bundle.
type LoginContext struct {
	Token            string
	Cookie           string
	CryptedUID       string
	RegContext       string // |regm blob — cần pass qua add-email/confirm
	Birthday         string
	MachineID        string
	EncPwd           string // #PWD_* từ reg gốc
	FirstName        string
	LastName         string
	ProfilePhoto     string
	WaterfallID      string
	Contactpoint     string // contactpoint GỐC của account (từ reg) → dùng làm reg_info.contactpoint
	ContactpointType string // "email" hoặc "phone"
}

func parseLoginResponse(body string) LoginContext {
	clean := strings.ReplaceAll(body, "\\", "")
	ctx := LoginContext{}
	if m := reLoginToken.FindStringSubmatch(clean); len(m) > 1 {
		ctx.Token = m[1]
	}
	ctx.CryptedUID = firstGroup(reLoginCryptedID, clean)
	ctx.RegContext = firstGroup(reLoginRegCtx, clean)
	ctx.Birthday = firstGroup(reLoginBirthday, clean)
	ctx.MachineID = firstGroup(reLoginMachineID, clean)
	ctx.EncPwd = firstGroup(reLoginEncPwd, clean)
	ctx.FirstName = firstGroup(reLoginFirstName, clean)
	ctx.LastName = firstGroup(reLoginLastName, clean)
	ctx.ProfilePhoto = firstGroup(reLoginProfilePhoto, clean)
	ctx.WaterfallID = firstGroup(reLoginWaterfallID, clean)
	ctx.Contactpoint = firstGroup(reLoginContactpoint, clean)
	ctx.ContactpointType = firstGroup(reLoginContactpointT, clean)
	cUser := firstGroup(reLoginCUser, clean)
	xs := firstGroup(reLoginXS, clean)
	fr := firstGroup(reLoginFR, clean)
	datr := firstGroup(reLoginDATR, clean)
	ctx.Cookie = composeCookie(cUser, xs, fr, datr)
	return ctx
}

// buildSendLoginBody dựng form body send_login_request BÁM SÁT capture thật.
func buildSendLoginBody(uid, password, localAuthToken, fullName, deviceID, familyDeviceID, machineID, locale string) string {
	ts := time.Now().Unix()
	waterfallID := uuid.New().String()
	secureFamilyDevID := uuid.New().String()
	lat := int64(60000000000000 + mrand.Int63n(9000000000000))
	// Password: #PWD_MSGR:0:<ts>:<plaintext> (key 0 = plaintext qua HTTPS, như reg #PWD_FB4A:0).
	encPwd := fmt.Sprintf("#PWD_MSGR:0:%d:%s", ts, password)

	// aymh context: chính account này (facebook_local_auth token EAAAAU sẵn có).
	var aymhAccounts []interface{}
	ssoTokenMap := "{}"
	var accountsList []interface{}
	if localAuthToken != "" {
		aymhAccounts = []interface{}{map[string]interface{}{
			"profiles": map[string]interface{}{
				uid: map[string]interface{}{
					"previously_authenticated_nonce": "",
					"is_derived":                     0,
					"credentials":                    []interface{}{map[string]interface{}{"credential_type": "facebook_local_auth", "blob": "", "token": localAuthToken}},
					"account_center_id":              uid,
					"profile_picture_url":            "",
					"small_profile_picture_url":      nil,
					"notification_count":             0,
					"last_access_time":               0,
					"token":                          localAuthToken,
					"has_smartlock":                  0,
					"credential_type":                "facebook_local_auth",
					"sim_phone_number":               nil,
					"from_accurate_privacy_result":   0,
					"user_id":                        uid,
					"ob_user_id":                     nil,
					"name":                           fullName,
					"source_credential_type":         "facebook_local_auth",
					"account_source":                 "",
				},
			},
			"id": uid,
		}}
		ssoTokenMap = fmt.Sprintf(`{"%s":[{"credential_type":"facebook_local_auth","token":"%s"}]}`, uid, localAuthToken)
		accountsList = []interface{}{map[string]interface{}{"uid": uid, "credential_type": "facebook_local_auth", "token": localAuthToken}}
	} else {
		aymhAccounts = []interface{}{}
		accountsList = []interface{}{}
	}

	clientInputParams := map[string]interface{}{
		"blocked_uids":                            []interface{}{},
		"aac":                                     genAAC(),
		"sim_phones":                              []interface{}{""},
		"aymh_accounts":                           aymhAccounts,
		"network_bssid":                           nil,
		"secure_family_device_id":                 secureFamilyDevID,
		"has_granted_read_contacts_permissions":   0,
		"auth_secure_device_id":                   "",
		"has_whatsapp_installed":                  0,
		"password":                                encPwd,
		"sso_token_map_json_string":               ssoTokenMap,
		"block_store_machine_id":                  "",
		"cloud_trust_token":                       nil,
		"event_flow":                              "login_manual",
		"password_contains_non_ascii":             "false",
		"client_known_key_hash":                   "",
		"sso_accounts_auth_data":                  []interface{}{},
		"encrypted_msisdn":                        "",
		"has_granted_read_phone_permissions":      0,
		"app_manager_id":                          "",
		"should_show_nested_nta_from_aymh":        1,
		"device_id":                               deviceID,
		"zero_balance_state":                      "",
		"login_attempt_count":                     1,
		"machine_id":                              "",
		"accounts_list":                           accountsList,
		"gms_incoming_call_retriever_eligibility": "client_not_supported",
		"family_device_id":                        familyDeviceID,
		"fb_ig_device_id":                         []interface{}{},
		"device_emails":                           []interface{}{},
		"try_num":                                 1,
		"lois_settings":                           map[string]interface{}{"lois_token": ""},
		"event_step":                              "home_page",
		"headers_infra_flow_id":                   "",
		"openid_tokens":                           map[string]interface{}{},
		"contact_point":                           uid,
	}
	serverParams := map[string]interface{}{
		"should_trigger_override_login_2fa_action":     0,
		"is_from_logged_out":                           0,
		"should_trigger_override_login_success_action": 0,
		"login_credential_type":                        "none",
		"server_login_source":                          "login",
		"waterfall_id":                                 waterfallID,
		"two_step_login_type":                          "one_step_login",
		"login_source":                                 "Login",
		"is_platform_login":                            0,
		"pw_encryption_try_count":                      1,
		"login_entry_point":                            "logged_out",
		"INTERNAL__latency_qpl_marker_id":              36707139,
		"is_from_aymh":                                 1,
		"offline_experiment_group":                     "caa_iteration_v3_perf_msg_6",
		"is_from_landing_page":                         0,
		"left_nav_button_action":                       "BACK",
		"password_text_input_id":                       "ax6j0t:102",
		"is_from_empty_password":                       0,
		"is_from_msplit_fallback":                      0,
		"ar_event_source":                              "login_home_page",
		"username_text_input_id":                       "ax6j0t:101",
		"layered_homepage_experiment_group":            nil,
		"device_id":                                    deviceID,
		"login_surface":                                "login_home",
		"INTERNAL__latency_qpl_instance_id":            lat,
		"reg_flow_source":                              "aymh_multi_profiles_native_integration_point",
		"is_caa_perf_enabled":                          1,
		"credential_type":                              "password",
		"is_from_password_entry_page":                  0,
		"caller":                                       "gslr",
		"family_device_id":                             familyDeviceID,
		"is_from_assistive_id":                         0,
		"access_flow_version":                          "pre_mt_behavior",
		"is_from_logged_in_switcher":                   0,
	}

	inner := map[string]interface{}{"client_input_params": clientInputParams, "server_params": serverParams}
	// FB custom format: {params:{...},} — key params KHÔNG quote + dấu phẩy cuối.
	paramsStr := "{params:" + verifybase.MustJSON(inner) + ",}"

	variables := map[string]interface{}{
		"params": map[string]interface{}{
			"params":              paramsStr,
			"bloks_versioning_id": loginBloksVer,
			"app_id":              loginAppID,
		},
		"scale": "3",
		"nt_context": map[string]interface{}{
			"is_flipper_enabled":           false,
			"theme_params":                 []interface{}{map[string]interface{}{"value": []string{}, "design_system_name": "FDS"}},
			"debug_tooling_metadata_token": nil,
		},
	}
	variablesJSON := verifybase.MustJSON(variables)

	form := url.Values{}
	form.Set("method", "post")
	form.Set("pretty", "false")
	form.Set("format", "json")
	form.Set("server_timestamps", "true")
	form.Set("locale", locale)
	form.Set("fb_api_req_friendly_name", loginFriendlyName)
	form.Set("fb_api_caller_class", "graphservice")
	form.Set("client_doc_id", loginDocID)
	form.Set("fb_api_client_context", `{"is_background":false}`)
	form.Set("variables", variablesJSON)
	form.Set("fb_api_analytics_tags", `["GraphServices"]`)
	form.Set("client_trace_id", uuid.New().String())
	return form.Encode()
}

// genAAC sinh aac (anti-automation code) JSON string khớp format capture.
func genAAC() string {
	b := make([]byte, 32)
	if _, err := crand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(mrand.Intn(256))
		}
	}
	return fmt.Sprintf(`{"aac_init_timestamp":%d,"aacjid":"%s","aaccs":"%s"}`,
		time.Now().Unix(), uuid.New().String(), base64.RawURLEncoding.EncodeToString(b))
}

// MessengerActionHeaders — header chung cho mọi FbBloksActionRootQuery Messenger
// (login + add-email + confirm + resend): APP-token Messenger (KHÔNG dùng user token),
// content-encoding gzip (BẮT BUỘC vì DoPost gzip body, thiếu = #100), device-scope headers.
func MessengerActionHeaders(ua, deviceID, familyDeviceID, friendlyName string) [][2]string {
	if ua == "" {
		ua = RandomUA("")
	}
	analyticsTag := `{"network_tags":{"product":"` + messengerProduct + `","request_category":"graphql","purpose":"none","retry_attempt":"0"},"application_tags":"graphservice"}`
	return [][2]string{
		{"x-fb-request-analytics-tags", analyticsTag},
		{"x-fb-rmd", "state=URL_ELIGIBLE"},
		{"priority", "u=3, i"},
		{"user-agent", ua},
		{"x-graphql-client-library", "graphservice"},
		{"content-type", "application/x-www-form-urlencoded"},
		{"content-encoding", "gzip"}, // BẮT BUỘC: DoPost gzip body → FB cần header này để giải nén (thiếu = #100)
		{"x-zero-eh", "664c0faaac849cb891d0a261fbb72a12"},
		{"authorization", "OAuth " + messengerAppToken},
		{"x-zero-state", "unknown"},
		{"x-zero-f-device-id", familyDeviceID},
		{"app-scope-id-header", deviceID},
		{"x-fb-friendly-name", friendlyName},
		{"x-fb-connection-type", "WIFI"},
		{"x-tigon-is-retry", "False"},
		{"accept-encoding", "gzip, deflate"},
		{"x-fb-http-engine", "Tigon/Liger"},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
	}
}

// composeCookie ghép cookie chuẩn (datr;c_user;xs;fr), bỏ field rỗng.
func composeCookie(cUser, xs, fr, datr string) string {
	var parts []string
	if datr != "" {
		parts = append(parts, "datr="+datr)
	}
	if cUser != "" {
		parts = append(parts, "c_user="+cUser)
	}
	if xs != "" {
		parts = append(parts, "xs="+xs)
	}
	if fr != "" {
		parts = append(parts, "fr="+fr)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ";") + ";"
}

func firstGroup(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}
	return ""
}

// AppRoot doc_id + nt_context (confirmation.fb.bottomsheet + confirmation.change.email).
// Khác Action doc_id (login/add-email/confirm), AppRoot dùng doc khác + nt_context đầy đủ.
const (
	appRootDocID        = "10537346114757978319483558625"
	appRootStylesID     = "f120dca679661913f8c3201dfabaad01"
	bottomsheetAppID    = "com.bloks.www.bloks.caa.reg.confirmation.fb.bottomsheet"
	bottomsheetFriendly = "FbBloksAppRootQuery-" + bottomsheetAppID
	changeEmailAppID    = "com.bloks.www.bloks.caa.reg.confirmation.change.email"
	changeEmailFriendly = "FbBloksAppRootQuery-" + changeEmailAppID
)

// FetchConfirmationContext gọi 2 bước render (bottomsheet → change.email) để
// lấy reg_context đúng trạng thái cho contactpoint_email.async.
// Capture V4: [4145]→[4146]→[4162].
// Trả reg_context mới (hoặc giữ nguyên nếu fail).
func FetchConfirmationContext(ctx context.Context, client verifybase.Client, lctx LoginContext, deviceID, familyDeviceID, uid, ua, locale string, notify func(string)) string {
	log := func(m string) {
		if notify != nil {
			notify(m)
		}
	}
	regCtx := lctx.RegContext
	if regCtx == "" {
		log("[AppMessV3] FetchConfirmationContext: không có reg_context → skip")
		return ""
	}

	// reg_info đầy đủ cho render steps (dùng data từ login response)
	regInfo := buildBottomsheetRegInfo(lctx, deviceID, familyDeviceID, uid)

	// Bước 1: confirmation.fb.bottomsheet
	log("[AppMessV3] Render confirmation.fb.bottomsheet...")
	body1 := buildAppRootBody(bottomsheetAppID, bottomsheetFriendly, lctx.WaterfallID, regCtx, regInfo, deviceID, familyDeviceID, locale, "")
	hdr1 := MessengerActionHeaders(ua, deviceID, familyDeviceID, bottomsheetFriendly)
	postCtx1, cancel1 := context.WithTimeout(ctx, 20*time.Second)
	resp1, _ := verifybase.DoPost(postCtx1, client, verifybase.BgraphURL, body1, hdr1)
	cancel1()
	if newCtx := firstGroup(reLoginRegCtx, strings.ReplaceAll(resp1, "\\", "")); newCtx != "" {
		regCtx = newCtx
		log("[AppMessV3] Bottomsheet OK — reg_context updated")
	}

	// Bước 2: confirmation.change.email (capture V5 14631: INTERNAL_INFRA_screen_id riêng)
	log("[AppMessV3] Render confirmation.change.email...")
	body2 := buildAppRootBody(changeEmailAppID, changeEmailFriendly, lctx.WaterfallID, regCtx, regInfo, deviceID, familyDeviceID, locale, "CAA_REG_CONFIRMATION_CHANGE_EMAIL")
	hdr2 := MessengerActionHeaders(ua, deviceID, familyDeviceID, changeEmailFriendly)
	postCtx2, cancel2 := context.WithTimeout(ctx, 20*time.Second)
	resp2, _ := verifybase.DoPost(postCtx2, client, verifybase.BgraphURL, body2, hdr2)
	cancel2()
	if newCtx := firstGroup(reLoginRegCtx, strings.ReplaceAll(resp2, "\\", "")); newCtx != "" {
		regCtx = newCtx
		log("[AppMessV3] Change.email OK — reg_context updated")
	}

	return regCtx
}

// buildBottomsheetRegInfo dựng reg_info đầy đủ cho render steps (dùng data từ login response).
func buildBottomsheetRegInfo(lctx LoginContext, deviceID, familyDevID, uid string) string {
	ri := map[string]interface{}{
		"first_name": lctx.FirstName, "last_name": lctx.LastName,
		"full_name": strings.TrimSpace(lctx.FirstName + " " + lctx.LastName),
		"birthday":  lctx.Birthday, "age_range": "o18", "gender": 2,
		"contactpoint": lctx.Contactpoint, "contactpoint_type": lctx.ContactpointType,
		"device_id": deviceID, "family_device_id": familyDevID,
		"user_id": uid, "machine_id": lctx.MachineID,
		"crypted_user_id":    lctx.CryptedUID,
		"encrypted_password": lctx.EncPwd,
		"profile_photo":      lctx.ProfilePhoto, "profile_photo_id": nil,
		"is_cp_claimed": false, "is_cp_auto_confirmed": false, "is_cp_auto_confirmable": false,
		"is_caa_perf_enabled": false, "is_preform": true,
		"screen_visited":       []interface{}{"CAA_REG_CONFIRMATION_SCREEN"},
		"registration_flow_id": "", "suma_on_conf_threshold": -1,
		"should_show_error_msg": true, "should_show_rel_error": false,
		"ig_footer_variant": "control", "full_sheet_flow": false,
		"is_in_nta_single_form": false,
	}
	return verifybase.MustJSON(ri)
}

// buildAppRootBody dựng form body AppRoot query (purpose=fetch) cho render steps.
// Capture V5: render SAU create.account (bottomsheet/change.email) dùng login_surface=registration,
// login_entry_point=registration, current_step=10. screenID = INTERNAL_INFRA_screen_id (rỗng = bỏ qua).
func buildAppRootBody(appID, friendlyName, waterfallID, regCtx, regInfo, deviceID, familyDeviceID, locale, screenID string) string {
	lat := int64(66000000000000 + mrand.Int63n(900000000000))
	serverParams := map[string]interface{}{
		"is_from_logged_out": 0, "device_id": deviceID, "family_device_id": familyDeviceID,
		"login_surface": "registration", "waterfall_id": waterfallID,
		"trigger": "default", "timer_id": "wa_retriever",
		"is_platform_login": 0, "login_entry_point": "registration",
		"INTERNAL__latency_qpl_marker_id":   36707139,
		"INTERNAL__latency_qpl_instance_id": lat,
		"flow_info":                         `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`,
		"reg_context":                       regCtx, "reg_info": regInfo,
		"offline_experiment_group": "caa_iteration_v3_perf_msg_6",
		"zero_tap_enabled":         0, "access_flow_version": "pre_mt_behavior", "current_step": 10,
	}
	if screenID != "" {
		serverParams["INTERNAL_INFRA_screen_id"] = screenID
	}
	inner := map[string]interface{}{
		"client_input_params": map[string]interface{}{
			"lois_settings": map[string]interface{}{"lois_token": ""},
			"aac":           genAAC(),
		},
		"server_params": serverParams,
	}
	paramsStr := "{params:" + verifybase.MustJSON(inner) + ",}"
	variables := map[string]interface{}{
		"params": map[string]interface{}{
			"params": paramsStr, "bloks_versioning_id": loginBloksVer, "app_id": appID,
		},
		"scale": "3",
		"nt_context": map[string]interface{}{
			"using_white_navbar": true, "styles_id": appRootStylesID, "pixel_ratio": 3,
			"is_push_on": true, "debug_tooling_metadata_token": nil,
			"is_flipper_enabled": false,
			"theme_params":       []interface{}{map[string]interface{}{"value": []string{}, "design_system_name": "FDS"}},
			"bloks_version":      loginBloksVer,
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
	form.Set("purpose", "fetch")
	form.Set("fb_api_req_friendly_name", friendlyName)
	form.Set("fb_api_caller_class", "graphservice")
	form.Set("client_doc_id", appRootDocID)
	form.Set("fb_api_client_context", `{"is_background":false}`)
	form.Set("variables", verifybase.MustJSON(variables))
	form.Set("fb_api_analytics_tags", `["GraphServices"]`)
	form.Set("client_trace_id", uuid.New().String())
	return form.Encode()
}
