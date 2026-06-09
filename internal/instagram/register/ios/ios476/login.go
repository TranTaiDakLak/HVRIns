// login.go — iOS CAA login (send_login_request) để lấy user token EAAAAAY.
//
// Dùng khi VERIFY iOS mà account CHƯA có token EAAAAAY (vd reg bằng Android,
// hoặc reg iOS nosess chỉ có srnonce). Login bằng app token (OAuth) → FB trả về
// access_token "EAAAAAY..." + session_cookies (c_user/xs/fr/datr).
//
// Tham chiếu capture thật:
//
//	D:\Git2026\facebook_repo\IOS\[4624] request/response_graph.facebook.com_message.txt
//	app_id = com.bloks.www.bloks.caa.login.async.send_login_request
//
// QUAN TRỌNG: chỉ trả token prefix EAAAAAY (iOS) — token Android EAAAAU KHÔNG hợp lệ
// cho endpoint Bloks CAA iOS. Login dùng CÙNG doc_id/bloks_versioning_id/styles_id
// như reg create.account (chỉ khác app_id + friendly_name).
package ios476

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"HVRIns/internal/instagram/fakeinfo"
)

const (
	loginAppID        = "com.bloks.www.bloks.caa.login.async.send_login_request"
	loginFriendlyName = "FBBloksActionRootQuery-" + loginAppID
)

// FetchIOSToken thực hiện CAA login iOS để lấy user token EAAAAAY.
// Trả về (token EAAAAAY, cookie "c_user=..;xs=..;fr=..;datr=..;", error).
// Header chạy CELLULAR (x-fb-sim-hni) + app token oauthToken.
func FetchIOSToken(ctx context.Context, uid, password, datr, proxyStr, countryCode string, notify func(string)) (string, string, error) {
	log := func(m string) {
		if notify != nil {
			notify(m)
		}
	}
	uid = strings.TrimSpace(uid)
	password = strings.TrimSpace(password)
	if uid == "" || password == "" {
		return "", "", fmt.Errorf("FetchIOSToken: thiếu uid hoặc password")
	}

	locale := fakeinfo.LocaleFromCountry(countryCode)
	if locale == "" {
		locale = "en_US"
	}
	profile := BuildProfile(locale, countryCode)
	if d := strings.TrimSpace(datr); d != "" {
		// Dùng lại datr của account làm X-FB-Integrity-Machine-Id (trust continuity).
		profile.MachineID = d
	}

	sess, err := newSession(proxyStr, profile.Device.IOSDot)
	if err != nil {
		return "", "", fmt.Errorf("create ios session: %w", err)
	}
	defer sess.client.CloseIdleConnections()

	// Mã hóa password 1 lần (#PWD_WILDE:2 via RSA pwd_key_fetch, fallback #PWD_FB4A:0).
	encPwd := encryptPasswordForReg(ctx, sess, profile, password)
	body, err := buildLoginRequestBody(profile, uid, encPwd)
	if err != nil {
		return "", "", fmt.Errorf("build login body: %w", err)
	}

	// Retry login: lỗi mạng / response transient (không có token) → thử lại tối đa
	// loginMaxAttempts lần (backoff tăng dần). Lỗi DỨT KHOÁT (checkpoint / sai mật
	// khẩu) → dừng ngay, không retry.
	const loginMaxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= loginMaxAttempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return "", "", ctx.Err()
			case <-time.After(time.Duration(attempt-1) * 2 * time.Second):
			}
			log(fmt.Sprintf("[Login][iOS] Retry %d/%d...", attempt, loginMaxAttempts))
		}
		log(fmt.Sprintf("[Login][iOS] POST send_login_request (uid=%s, dev=%s, lần %d)...", uid, profile.Device.FBDV, attempt))

		// Build header mới mỗi lần (appnet sid / usdid / conn-uuid fresh).
		resp, perr := sess.postGzip(ctx, graphURL, body, buildLoginHeaders(profile))
		if perr != nil && resp == "" {
			lastErr = fmt.Errorf("HTTP login: %w", perr)
			continue // transient → retry
		}

		// Tái dùng parser create.account: reEAAToken match EAAAAA... (loại EAAAAU) +
		// extract c_user/xs/fr/datr từ session_cookies trong response login.
		if outcome, _ := parseCreateAccountResponse(resp); outcome != nil && strings.HasPrefix(outcome.AccessToken, "EAAAAAY") {
			log(fmt.Sprintf("[Login][iOS] OK — token EAAAAAY (len=%d, lần %d)", len(outcome.AccessToken), attempt))
			return outcome.AccessToken, outcome.Cookie, nil
		}

		// Lỗi DỨT KHOÁT → KHÔNG retry.
		low := strings.ToLower(strings.ReplaceAll(resp, "\\", ""))
		switch {
		case strings.Contains(low, "checkpoint"):
			return "", "", fmt.Errorf("login bị checkpoint")
		case strings.Contains(low, "incorrect_password") || strings.Contains(low, "wrong_password") || strings.Contains(low, "incorrect password"):
			return "", "", fmt.Errorf("login sai mật khẩu")
		}
		lastErr = fmt.Errorf("login không trả token EAAAAAY (resp %d bytes)", len(resp))
		// transient/unknown → retry vòng sau
	}
	return "", "", fmt.Errorf("login thất bại sau %d lần: %w", loginMaxAttempts, lastErr)
}

// buildLoginHeaders dựng header cho send_login_request (giống buildHeaders nhưng
// friendly-name login + thêm x-fb-family-device-id như capture login).
func buildLoginHeaders(p IOSProfile) [][2]string {
	analyticsTag := `{"network_tags":{"product":"6628568379","request_category":"graphql","purpose":"fetch","retry_attempt":"0"}}`
	return [][2]string{
		{"user-agent", p.UserAgent},
		{"accept-encoding", "gzip, deflate, br"},
		{"accept", "*/*"},
		{"connection", "keep-alive"},
		{"x-fb-rmd", "state=URL_ELIGIBLE"}, // capture login + verify có header này (reg không)
		{"x-fb-appnetsession-sid", appnetSID()},
		{"x-meta-usdid", generateUSDID()},
		{"x-fb-http-engine", "Tigon/Liger"},
		{"x-meta-zca", `{"e": {"c":7}}`},
		{"x-fb-session-gk", "v1:gk:fb_ios_tasos_congestion_signal:@pass;"},
		{"authorization", "OAuth " + oauthToken},
		{"x-fb-sim-hni", p.Sim.HNI},
		{"content-encoding", "gzip"},
		{"x-fb-appnetsession-nid", appnetNID()},
		{"x-fb-connection-type", p.ConnType},
		{"x-cloud-trust-token", p.CloudTrustID},
		{"x-fb-integrity-machine-id", p.MachineID},
		{"x-fb-device-id", p.DeviceID},
		{"x-fb-family-device-id", p.FamilyDeviceID},
		{"x-fb-friendly-name", loginFriendlyName},
		{"x-fb-tasos-experimental", "1"},
		{"content-type", "application/x-www-form-urlencoded"},
		{"x-tigon-is-retry", "False"},
		{"x-fb-request-analytics-tags", analyticsTag},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
		{"x-fb-conn-uuid-client", connUUID()},
		{"x-graphql-client-library", "pando"},
		{"x-graphql-request-purpose", "fetch"},
	}
}

// buildLoginRequestBody dựng form-urlencoded body cho send_login_request.
// Cấu trúc envelope giống create.account (variables.params.params = JSON-string
// của {server_params, client_input_params}), chỉ khác app_id + friendly_name.
func buildLoginRequestBody(p IOSProfile, uid, encPassword string) (string, error) {
	pixelRatio := 2
	if p.Device.FBSS == "3" {
		pixelRatio = 3
	}
	lat := time.Now().UnixNano() % 1000000000000000

	serverParams := map[string]any{
		"device_id":                         p.DeviceID,
		"family_device_id":                  p.FamilyDeviceID,
		"INTERNAL__latency_qpl_marker_id":   36707139,
		"INTERNAL__latency_qpl_instance_id": lat,
		"login_source":                      "Login",
		"login_credential_type":             "password",
		"server_login_source":               "login",
		"login_entry_point":                 "logged_out",
		"is_from_logged_out":                1,
		"is_from_logged_in_switcher":        0,
		"is_from_landing_page":              0,
		"is_from_aymh":                      0,
		"is_platform_login":                 0,
		"is_from_password_entry_page":       0,
		"is_from_empty_password":            0,
		"is_from_assistive_id":              0,
		"access_flow_version":               "pre_mt_behavior",
		"login_surface":                     "login_home",
		"waterfall_id":                      p.WaterfallID,
		"credential_type":                   "password",
		"caller":                            "gslr",
		"two_step_login_type":               "one_step_login",
		"reg_flow_source":                   "login_home_native_integration_point",
		"pw_encryption_try_count":           1,
		"is_caa_perf_enabled":               1,
		"offline_experiment_group":          nil,
		"layered_homepage_experiment_group": "not_in_experiment",
		"x_app_device_signals": map[string]any{
			"DEVICE_ID":  upperUUID(),
			"MACHINE_ID": p.MachineID,
		},
	}

	clientInputParams := map[string]any{
		"device_id":                             p.DeviceID,
		"family_device_id":                      p.FamilyDeviceID,
		"secure_family_device_id":               "",
		"machine_id":                            p.MachineID,
		"cloud_trust_token":                     p.CloudTrustID,
		"password":                              encPassword,
		"contact_point":                         uid,
		"event_flow":                            "login_manual",
		"event_step":                            "home_page",
		"login_attempt_count":                   1,
		"try_num":                               1,
		"zero_balance_state":                    "",
		"has_whatsapp_installed":                0,
		"network_bssid":                         nil,
		"openid_tokens":                         map[string]any{},
		"accounts_list":                         []any{},
		"aymh_accounts":                         []any{},
		"sso_accounts_auth_data":                []any{},
		"sso_token_map_json_string":             "{}",
		"block_store_machine_id":                "",
		"encrypted_msisdn":                      "",
		"auth_secure_device_id":                 "",
		"headers_infra_flow_id":                 "",
		"client_known_key_hash":                 "",
		"blocked_uids":                          []any{},
		"fb_ig_device_id":                       []any{},
		"password_contains_non_ascii":           "false",
		"has_granted_read_phone_permissions":    0,
		"has_granted_read_contacts_permissions": 0,
		"lois_settings":                         map[string]any{"lois_token": ""},
	}

	inner := map[string]any{
		"server_params":       serverParams,
		"client_input_params": clientInputParams,
	}
	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return "", fmt.Errorf("marshal inner params: %w", err)
	}

	variables := map[string]any{
		"voiceover_enabled":            "false",
		"should_include_delegate_page": true,
		"scale":                        pixelRatio,
		"generic_attachment_tall_cover_image_height":         352,
		"include_workplace_fields":                           "false",
		"formatType":                                         "concise",
		"generic_attachment_tall_cover_image_width_no_scale": 335,
		"params": map[string]any{
			"bloks_versioning_id": bloksVersioningID,
			"app_id":              loginAppID,
			"params":              string(innerJSON),
		},
		"generic_attachment_tall_cover_image_width": 670,
		"nt_context": map[string]any{
			"theme_params": []any{
				map[string]any{
					"design_system_name": "FDS",
					"value":              []string{"DARKER_PRIMARY_DEEMPHASIZED_BUTTON_BACKGROUND_TEST", "DEFAULT"},
				},
				map[string]any{
					"design_system_name": "XMDS",
					"value":              []string{"three_neutral_gray"},
				},
			},
			"pixel_ratio":   pixelRatio,
			"styles_id":     stylesID,
			"bloks_version": bloksVersioningID,
		},
		"generic_attachment_small_cover_image_height":       80,
		"enable_voiceover_gating_for_accessibility_caption": "false",
		"device": "iphone",
		"generic_attachment_small_cover_image_width": 80,
	}
	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return "", fmt.Errorf("marshal variables: %w", err)
	}

	parts := []string{
		"method=post",
		"pretty=false",
		"format=json",
		"server_timestamps=true",
		"locale=" + p.Locale,
		"purpose=fetch",
		"fb_api_req_friendly_name=" + loginFriendlyName,
		"client_doc_id=" + docIDAction,
		"fb_api_client_context=" + url.QueryEscape(`{"is_background":"0"}`),
		"variables=" + url.QueryEscape(string(variablesJSON)),
	}
	return strings.Join(parts, "&"), nil
}
