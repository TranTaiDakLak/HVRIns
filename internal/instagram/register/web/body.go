// body.go — Build request body cho từng bước đăng ký B1-B8
// Mapping EXACT từ API_RegFB.txt — Bloks CAA (Create Account API)
// BKV: c41a736f68c4f99148bbc5f20c1a79cb81e79585374dd83b20119a79174bd379
package web

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	bkvReg = "c41a736f68c4f99148bbc5f20c1a79cb81e79585374dd83b20119a79174bd379"
	hsReg  = "20543.BP:wbloks_caa_pkg.2.0...0"

	// Endpoints — appid + type + __bkv
	// Action steps
	endpointB1 = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.name_fb&type=app&__bkv=" + bkvReg
	endpointB2 = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.name.async&type=action&__bkv=" + bkvReg
	endpointB3 = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.birthday.async&type=action&__bkv=" + bkvReg
	endpointB4 = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.gender.async&type=action&__bkv=" + bkvReg
	endpointB5 = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.async.contactpoint_phone.async&type=action&__bkv=" + bkvReg
	endpointB6 = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.password.async&type=action&__bkv=" + bkvReg
	endpointB7 = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.tos&type=app&__bkv=" + bkvReg
	endpointB8 = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.create.account.async&type=action&__bkv=" + bkvReg
	// Screen-fetch (app type) — chèn giữa các action steps (từ API_RegFB.txt cập nhật)
	endpointScreenBirthday  = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.birthday&type=app&__bkv=" + bkvReg
	endpointScreenGender    = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.gender&type=app&__bkv=" + bkvReg
	endpointScreenPhone     = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.contactpoint_phone&type=app&__bkv=" + bkvReg
	endpointScreenPassword  = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.password&type=app&__bkv=" + bkvReg
	endpointScreenSaveCreds = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.save-credentials&type=app&__bkv=" + bkvReg

	refererReg = "https://m.facebook.com/r.php"

	flowInfoJSON = `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`

	// INTERNAL__latency_qpl_marker_id — constant từ API
	qplMarkerID = 36707139
)

// buildBaseBody — form body prefix cho mọi bước
// EXACT từ API_RegFB.txt:
// __aaid=0&__user=0&__a=1&__req=...&__hs=20543.BP:wbloks_caa_pkg.2.0...0&dpr=3&__ccg=GOOD&__rev=...&__s=...&__hsi=...&__dyn=...&fb_dtsg=...&jazoest=...&lsd=...&__jssesw=10
func buildBaseBody(s *RegSession, req string) string {
	return fmt.Sprintf(
		"__aaid=0&__user=0&__a=1&__req=%s&__hs=%s&dpr=3&__ccg=GOOD&__rev=%s&__s=%s&__hsi=%s&__dyn=%s&fb_dtsg=%s&jazoest=%s&lsd=%s&__jssesw=10",
		req,
		url.QueryEscape(hsReg),
		url.QueryEscape(s.Rev),
		url.QueryEscape(s.S),
		url.QueryEscape(s.Hsi),
		url.QueryEscape(s.Dyn),
		url.QueryEscape(s.FbDtsg),
		url.QueryEscape(s.Jazoest),
		url.QueryEscape(s.Lsd),
	)
}

// marshalParamsEncoded — marshal params JSON rồi URL-encode để dùng trong form body
// Format Bloks wbloks API: params=URL_ENCODE({"params":"JSON_STRING"})
// Trong đó JSON_STRING = {"server_params":{...},"client_input_params":{...}}
func marshalParamsEncoded(serverParams, clientParams map[string]interface{}) string {
	inner := map[string]interface{}{
		"server_params":       serverParams,
		"client_input_params": clientParams,
	}
	innerBytes, err := json.Marshal(inner)
	if err != nil {
		return ""
	}
	outer := map[string]interface{}{
		"params": string(innerBytes),
	}
	outerBytes, err := json.Marshal(outer)
	if err != nil {
		return ""
	}
	return url.QueryEscape(string(outerBytes))
}

// regInfoStr — trả về reg_info hiện tại hoặc initial nếu chưa có
func regInfoStr(s *RegSession) string {
	if s.RegInfo != "" {
		return s.RegInfo
	}
	return buildInitialRegInfo()
}

// buildInitialRegInfo — reg_info ban đầu cho B1
// Keys và values lấy EXACT từ API_RegFB.txt B1 reg_info (toàn bộ 170+ fields)
func buildInitialRegInfo() string {
	ri := map[string]interface{}{
		// Identity
		"first_name": nil, "last_name": nil, "full_name": nil,
		"suggested_first_name": nil, "suggested_last_name": nil, "suggested_full_name": nil,
		// Contact point
		"contactpoint": nil, "ar_contactpoint": nil, "contactpoint_type": nil,
		"is_using_unified_cp": nil, "unified_cp_screen_variant": nil,
		"is_cp_auto_confirmed": false, "is_cp_auto_confirmable": false, "is_cp_claimed": false,
		"confirmation_code": nil, "confirmation_code_send_error": nil,
		"confirmation_medium": nil,
		"confirmed_cp_and_code": nil, "notification_callback_id": nil,
		"cp_confirmed_by_auto_conf": false,
		"login_contactpoint": nil, "login_contactpoint_type": nil,
		"msg_previous_cp": nil,
		// Birthday / age
		"birthday": nil, "birthday_derived_from_age": nil,
		"age_range": nil, "did_use_age": nil, "os_shared_age_range": nil,
		"is_too_young": false,
		"spc_birthday_input": false,
		"used_ig_birthday": nil,
		// Gender
		"gender": nil, "use_custom_gender": false, "custom_gender": nil,
		// Password / security
		"encrypted_password": nil,
		"safetynet_token": nil, "safetynet_response": nil,
		"skip_slow_rel_check": false,
		"encrypted_msisdn": nil, "encrypted_msisdn_for_safetynet": nil,
		"cached_headers_safetynet_info": nil, "should_skip_headers_safetynet": nil,
		"headers_last_infra_flow_id": nil, "headers_last_infra_flow_id_safetynet": nil,
		"headers_flow_id": nil, "was_headers_prefill_available": nil,
		"attestation_result": nil,
		"request_data_and_challenge_nonce_string": nil,
		"caa_play_integrity_attestation_result": nil,
		"client_known_key_hash": nil,
		// Device / machine
		"device_id": nil, "ig4a_qe_device_id": nil, "family_device_id": nil,
		"machine_id": nil,
		"fb_device_id": nil, "fb_machine_id": nil,
		"ig_device_id": nil, "ig_machine_id": nil,
		"device_network_info": nil,
		"device_zero_balance_state": nil,
		// User / account
		"user_id": nil,
		"username": nil, "username_prefill": nil,
		"accounts_list_client": nil,
		"fb_conf_source": nil,
		"crypted_user_id": nil,
		"fb_access_token": nil,
		"xapp_accounts": nil,
		"existing_accounts": nil,
		"source_username": nil,
		// Profile photo
		"profile_photo": nil, "profile_photo_id": nil, "profile_photo_upload_id": nil,
		"avatar": nil, "th_profile_photo_token": nil,
		// OAuth / email
		"email_oauth_token_no_contact_perm": nil, "email_oauth_token": nil,
		"email_oauth_tokens": nil, "sign_in_with_google_email": nil,
		"email_oauth_exists": false, "email_prefilled": false,
		"google_oauth_account": nil,
		"login_form_siwg_email": nil,
		// Flow flags
		"should_skip_two_step_conf": nil,
		"openid_tokens_for_testing": nil,
		"sso_enabled": nil,
		"create_new_to_app_account": nil, "skip_session_info": nil,
		"ck_error": nil, "ck_id": nil, "ck_nonce": nil,
		"should_save_password": nil,
		"is_msplit_reg": nil, "is_spectra_reg": nil,
		"dema_account_consent_given": nil, "spectra_reg_token": nil,
		"spectra_reg_guardian_id": nil, "spectra_reg_guardian_logged_in_context": nil,
		"user_id_of_msplit_creator": nil, "msplit_creator_nonce": nil,
		"dma_data_combination_consent_given": nil,
		"should_skip_nta_upsell": nil,
		"big_blue_token": nil, "caa_reg_flow_source": nil,
		"ig_authorization_token": nil,
		"full_sheet_flow": false, "is_caa_perf_enabled": false,
		"is_preform": true,
		"should_show_rel_error": false, "ignore_suma_check": false,
		"dismissed_login_upsell_with_cna": false,
		"ignore_existing_login": false, "ignore_existing_login_from_suma": false,
		"ignore_existing_login_after_errors": false,
		"frl_authorization_token": nil, "post_form_errors": nil,
		"registration_flow_id": "",
		"skip_step_without_errors": false,
		"existing_account_exact_match_checked": false,
		"existing_account_fuzzy_match_checked": false,
		// Youth / regulation
		"should_skip_youth_tos": false, "is_youth_regulation_flow_complete": false,
		"is_on_cold_start": false,
		"in_sowa_experiment": false,
		"youth_regulation_config": nil,
		"conf_allow_back_nav_after_change_cp": nil,
		"conf_bouncing_cliff_screen_type": nil, "conf_show_bouncing_cliff": nil,
		"youth_consent_decision_time": nil,
		"sk_pipa_consent_given": nil,
		"show_youth_reg_in_ig_spc": false,
		// Flash call / SMS
		"eligible_to_flash_call_in_ig4a": false, "eligible_to_mo_sms_in_ig4a": false,
		"mo_sms_ent_id": nil,
		"flash_call_permissions_status": nil,
		"gms_incoming_call_retriever_eligibility": nil,
		"flash_call_provider": nil, "is_in_gms_experience": nil,
		"flash_call_nonce_prefix_details": nil,
		// SUMA / login upsell
		"reg_suma_state": 0, "is_msplit_neutral_choice": false,
		"ntp_import_source_info": nil,
		"should_show_spi_before_conf": true,
		"is_reg_request_from_ig_suma": false,
		"fb_suma_is_high_confidence": nil,
		"fb_email_login_upsell_skip_suma_post_tos": false,
		"fb_suma_is_from_email_login_upsell": false,
		"fb_suma_is_from_phone_login_upsell": false,
		"has_seen_suma_landing_page_pre_conf": false,
		"has_seen_suma_candidate_page_pre_conf": false,
		"suma_on_conf_threshold": -1,
		"cp_suma_results_map": nil,
		"is_wanted_suma_user": nil,
		"user_opted_out_of_ntp": nil,
		// Threads / TOA / SPC
		"is_toa_reg": false, "is_threads_public": false,
		"spc_import_flow": false,
		// WhatsApp / MSISDN
		"whatsapp_installed_on_client": false,
		"should_delay_wa_disclosure": false,
		"wa_data_bundle": nil,
		// IG partial account
		"ig_partially_created_account_user_id": nil,
		"ig_partially_created_account_nonce": nil,
		"ig_partially_created_account_nonce_expiry": nil,
		// NUX / NTA
		"force_sessionless_nux_experience": false,
		"is_sessionless_nux": nil,
		"is_nta_shortened": false,
		"is_in_nta_single_form": false,
		"nta_single_form_variant": nil,
		// Screen tracking
		"screen_visited":        []interface{}{"CAA_REG_WELCOME_SCREEN"},
		"has_seen_confirmation_screen": false,
		"should_override_back_nav": false,
		"should_show_bday_after_name_suggestions": nil,
		"pp_to_nux_eligible": false,
		"should_show_error_msg": true,
		// Misc
		"source_account_type": nil, "source_credentials_type": nil,
		"source_cuid": nil, "source_account_reg_info": nil,
		"soap_creation_source": nil, "source_account_type_to_reg_info": nil,
		"source_account_image_asset_id": nil,
		"attempted_silent_auth_in_fb": false, "attempted_silent_auth_in_ig": false,
		"next_uri": nil, "should_use_next_uri": false,
		"linking_entry_point": nil,
		"fb_encrypted_partial_new_account_properties": nil,
		"starter_pack_name": nil, "starter_pack_creator_user_ids": nil,
		"bloks_controller_source": nil, "airwave_registration_code": nil,
		"failed_birthday_year_count": nil,
		"user_presented_medium_source": nil,
		"is_from_registration_reminder": false,
		"ig_footer_variant": "control",
		"is_from_web_lite_reg_controller": true,
		"account_setup_waterfall_id": nil,
		"passkey_eligible_device": nil,
		"fdid_available_on_start": nil, "fdid_rid_available_on_start": nil,
	}
	b, _ := json.Marshal(ri)
	return string(b)
}

// loisClient — client_input_params lois_settings dạng nested object (EXACT từ API)
func loisClient() map[string]interface{} {
	return map[string]interface{}{
		"lois_settings": map[string]interface{}{"lois_token": ""},
	}
}

// actionServerParams — server_params cho action steps có event_request_id (B2,B5,B6,B8)
func actionServerParams(s *RegSession, step int, eventID, instanceID string) map[string]interface{} {
	return map[string]interface{}{
		"event_request_id":                  eventID,
		"reg_info":                          regInfoStr(s),
		"flow_info":                         flowInfoJSON,
		"current_step":                      step,
		"INTERNAL__latency_qpl_marker_id":   qplMarkerID,
		"INTERNAL__latency_qpl_instance_id": instanceID,
		"device_id":                         nil,
		"family_device_id":                  nil,
		"waterfall_id":                      s.WaterfallID,
		"offline_experiment_group":          nil,
		"layered_homepage_experiment_group": nil,
		"is_platform_login":                 0,
		"is_from_logged_in_switcher":        0,
		"is_from_logged_out":                0,
		"access_flow_version":               "pre_mt_behavior",
		"login_surface":                     "unknown",
	}
}

// actionServerParamsNoEventID — server_params cho B3,B4 (không có event_request_id theo spec mới)
func actionServerParamsNoEvent(s *RegSession, step int, instanceID string) map[string]interface{} {
	return map[string]interface{}{
		"reg_info":                          regInfoStr(s),
		"flow_info":                         flowInfoJSON,
		"current_step":                      step,
		"INTERNAL__latency_qpl_marker_id":   qplMarkerID,
		"INTERNAL__latency_qpl_instance_id": instanceID,
		"device_id":                         nil,
		"family_device_id":                  nil,
		"waterfall_id":                      s.WaterfallID,
		"offline_experiment_group":          nil,
		"layered_homepage_experiment_group": nil,
		"is_platform_login":                 0,
		"is_from_logged_in_switcher":        0,
		"is_from_logged_out":                0,
		"access_flow_version":               "pre_mt_behavior",
		"login_surface":                     "unknown",
	}
}

// buildScreenFetchBody — body cho screen-fetch app calls (birthday/gender/phone/password)
// SP: waterfall_id, is_platform_login=0, is_from_logged_out=0, access_flow_version,
//     reg_info, flow_info (cần để server track session), login_surface, login_entry_point,
//     current_step, INTERNAL_INFRA_screen_id
// CP: lois_settings, machine_id, cloud_trust_token, block_store_machine_id, aac
func buildScreenFetchBody(s *RegSession, req string, step int, screenID string) string {
	sp := map[string]interface{}{
		"waterfall_id":             s.WaterfallID,
		"is_platform_login":        0,
		"is_from_logged_out":       0,
		"access_flow_version":      "pre_mt_behavior",
		"reg_info":                 regInfoStr(s),
		"flow_info":                flowInfoJSON,
		"login_surface":            "login_home",
		"login_entry_point":        "logged_out",
		"current_step":             step,
		"INTERNAL_INFRA_screen_id": screenID,
	}
	if s.RegContext != "" {
		sp["reg_context"] = s.RegContext
	}
	cp := map[string]interface{}{
		"lois_settings":          map[string]interface{}{"lois_token": ""},
		"machine_id":             "",
		"cloud_trust_token":      nil,
		"block_store_machine_id": "",
		"aac":                    "",
	}
	return buildBaseBody(s, req) + "&params=" + marshalParamsEncoded(sp, cp)
}

// --- B1: name_fb (app, __req=3) ---
// server_params: waterfall_id, is_platform_login, is_from_logged_out, access_flow_version,
//                reg_info, flow_info, current_step=1, INTERNAL_INFRA_screen_id
// (reg_context thêm vào nếu server đã trả về)
// client_input_params: lois_settings{lois_token}, machine_id, cloud_trust_token, block_store_machine_id, aac
func buildB1Body(s *RegSession) string {
	sp := map[string]interface{}{
		"waterfall_id":             s.WaterfallID,
		"is_platform_login":        0,
		"is_from_logged_out":       0,
		"access_flow_version":      "pre_mt_behavior",
		"reg_info":                 regInfoStr(s),
		"flow_info":                flowInfoJSON,
		"current_step":             1,
		"INTERNAL_INFRA_screen_id": "bloks.caa.reg.name_fb",
	}
	if s.RegContext != "" {
		sp["reg_context"] = s.RegContext
	}
	cp := map[string]interface{}{
		"lois_settings":          map[string]interface{}{"lois_token": ""},
		"machine_id":             "",
		"cloud_trust_token":      nil,
		"block_store_machine_id": "",
		"aac":                    "",
	}
	return buildBaseBody(s, "3") + "&params=" + marshalParamsEncoded(sp, cp)
}

// --- B2: name.async (action, __req=7) ---
// server_params: event_request_id, reg_info, flow_info, current_step=2, flow_modifier,
//                INTERNAL__latency_qpl_*, device_id, family_device_id, waterfall_id,
//                offline/layered, is_platform_login, is_from_logged_in_switcher, is_from_logged_out,
//                access_flow_version, login_surface
// client_input_params: zero_balance_state, firstname, lastname, google_id_token, google_id_email,
//                      device_network_info, cloud_trust_token, network_bssid, lois_settings, aac
func buildB2Body(s *RegSession, input *RegInput, eventID string) string {
	sp := actionServerParams(s, 2, eventID, "25876640600052")
	sp["flow_modifier"] = flowInfoJSON

	cp := map[string]interface{}{
		"zero_balance_state":   "",
		"firstname":            input.FirstName,
		"lastname":             input.LastName,
		"google_id_token":      "",
		"google_id_email":      "",
		"device_network_info":  nil,
		"cloud_trust_token":    nil,
		"network_bssid":        nil,
		"lois_settings":        map[string]interface{}{"lois_token": ""},
		"aac":                  "",
	}
	return buildBaseBody(s, "7") + "&params=" + marshalParamsEncoded(sp, cp)
}

// --- Screen birthday (app, __req=b) — chèn giữa B2 và B3 ---
func buildScreenBirthdayBody(s *RegSession) string {
	return buildScreenFetchBody(s, "b", 2, "bloks.caa.reg.birthday")
}

// --- B3: birthday.async (action, __req=f) ---
// SP: không có event_request_id (theo spec mới), có reg_context nếu server đã trả về
func buildB3Body(s *RegSession, input *RegInput) string {
	sp := actionServerParamsNoEvent(s, 2, "26195474300184")
	if s.RegContext != "" {
		sp["reg_context"] = s.RegContext
	}
	cp := map[string]interface{}{
		"zero_balance_state":                "",
		"birthday_timestamp":               parseBirthday(input.Birthday),
		"should_skip_youth_tos":            0,
		"is_youth_regulation_flow_complete": 0,
		"os_age_range":                     "",
		"accounts_list":                    []interface{}{},
		"cloud_trust_token":                nil,
		"network_bssid":                    nil,
		"lois_settings":                    map[string]interface{}{"lois_token": ""},
		"aac":                              "",
	}
	return buildBaseBody(s, "f") + "&params=" + marshalParamsEncoded(sp, cp)
}

// --- Screen gender (app, __req=h) — chèn giữa B3 và B4 ---
func buildScreenGenderBody(s *RegSession) string {
	return buildScreenFetchBody(s, "h", 3, "bw206f:26")
}

// --- B4: gender.async (action, __req=j) ---
// SP: không có event_request_id (theo spec mới)
func buildB4Body(s *RegSession, input *RegInput) string {
	sp := actionServerParamsNoEvent(s, 3, "26671765400158")
	cp := map[string]interface{}{
		"zero_balance_state":   "",
		"gender":               input.Gender,
		"pronoun":              0,
		"custom_gender":        "",
		"device_phone_numbers": []interface{}{},
		"device_emails":        []interface{}{},
		"cloud_trust_token":    nil,
		"network_bssid":        nil,
		"lois_settings":        map[string]interface{}{"lois_token": ""},
		"aac":                  "",
	}
	return buildBaseBody(s, "j") + "&params=" + marshalParamsEncoded(sp, cp)
}

// --- Screen phone (app, __req=m) — chèn giữa B4 và B5 ---
func buildScreenPhoneBody(s *RegSession) string {
	return buildScreenFetchBody(s, "m", 4, "CAA_REG_CONTACT_POINT_PHONE")
}

// --- Screen password (app, __req=t) — chèn giữa B5 và B6 ---
func buildScreenPasswordBody(s *RegSession) string {
	return buildScreenFetchBody(s, "t", 5, "CAA_REG_PASSWORD")
}

// --- Screen save-credentials (app, __req=12) — chèn giữa B6 và B7 ---
// SP có thêm device_id="" và post_tos=0 so với các screen thông thường
func buildScreenSaveCredsBody(s *RegSession) string {
	sp := map[string]interface{}{
		"device_id":                "",
		"waterfall_id":             s.WaterfallID,
		"is_platform_login":        0,
		"is_from_logged_out":       0,
		"access_flow_version":      "pre_mt_behavior",
		"reg_info":                 regInfoStr(s),
		"flow_info":                flowInfoJSON,
		"login_surface":            "login_home",
		"login_entry_point":        "logged_out",
		"post_tos":                 0,
		"current_step":             6,
		"INTERNAL_INFRA_screen_id": "CAA_REG_SAVE_PASSWORD_CREDENTIALS",
	}
	if s.RegContext != "" {
		sp["reg_context"] = s.RegContext
	}
	cp := map[string]interface{}{
		"lois_settings":          map[string]interface{}{"lois_token": ""},
		"machine_id":             "",
		"cloud_trust_token":      nil,
		"block_store_machine_id": "",
		"aac":                    "",
	}
	return buildBaseBody(s, "12") + "&params=" + marshalParamsEncoded(sp, cp)
}

// --- B5: contactpoint_phone.async (action, __req=p) ---
// server_params: + event_request_id, cp_funnel, cp_source, text_input_id
// client_input_params: device_id, family_device_id, cloud_trust_token, block_store_machine_id,
//                      zero_balance_state, phone, accounts_list, build_type, encrypted_msisdn,
//                      headers_infra_flow_id, was_headers_prefill_available, was_headers_prefill_used,
//                      fb_ig_device_id, whatsapp_installed_on_client, confirmed_cp_and_code,
//                      msg_previous_cp, switch_cp_first_time_loading, switch_cp_have_seen_suma,
//                      login_upsell_phone_list, country_code, prefill_attempted_silent_auth,
//                      device_network_info, network_bssid, lois_settings, aac
func buildB5Body(s *RegSession, input *RegInput, eventID string) string {
	sp := actionServerParams(s, 4, eventID, "26765818300075")
	sp["cp_funnel"] = 0
	sp["cp_source"] = 0
	sp["text_input_id"] = "26765818300042"

	cp := map[string]interface{}{
		"device_id":                     "",
		"family_device_id":              "",
		"cloud_trust_token":             nil,
		"block_store_machine_id":        "",
		"zero_balance_state":            "",
		"phone":                         input.Phone,
		"accounts_list":                 []interface{}{},
		"build_type":                    "",
		"encrypted_msisdn":              "",
		"headers_infra_flow_id":         "",
		"was_headers_prefill_available": 0,
		"was_headers_prefill_used":      0,
		"fb_ig_device_id":               []interface{}{},
		"whatsapp_installed_on_client":  0,
		"confirmed_cp_and_code":         map[string]interface{}{},
		"msg_previous_cp":               "",
		"switch_cp_first_time_loading":  1,
		"switch_cp_have_seen_suma":      0,
		"login_upsell_phone_list":       []interface{}{},
		"country_code":                  "",
		"prefill_attempted_silent_auth": 0,
		"device_network_info":           nil,
		"network_bssid":                 nil,
		"lois_settings":                 map[string]interface{}{"lois_token": ""},
		"aac":                           "",
	}
	return buildBaseBody(s, "p") + "&params=" + marshalParamsEncoded(sp, cp)
}

// --- B6: password.async (action, __req=z) ---
// server_params: + spi_action, flow_modifier
// client_input_params: machine_id, cloud_trust_token, block_store_machine_id, zero_balance_state,
//                      encrypted_password, safetynet_token, safetynet_response, email_oauth_token_map,
//                      whatsapp_installed_on_client, encrypted_msisdn_for_safetynet,
//                      headers_last_infra_flow_id_safetynet, fb_ig_device_id,
//                      caa_play_integrity_attestation_result, client_known_key_hash,
//                      network_bssid, lois_settings, aac
func buildB6Body(s *RegSession, encryptedPassword string, eventID string) string {
	sp := actionServerParams(s, 5, eventID, "27211602300262")
	sp["spi_action"] = 0
	sp["flow_modifier"] = flowInfoJSON

	cp := map[string]interface{}{
		"machine_id":                            "",
		"cloud_trust_token":                     nil,
		"block_store_machine_id":                "",
		"zero_balance_state":                    "",
		"encrypted_password":                    encryptedPassword,
		"safetynet_token":                       "",
		"safetynet_response":                    "",
		"email_oauth_token_map":                 map[string]interface{}{},
		"whatsapp_installed_on_client":          0,
		"encrypted_msisdn_for_safetynet":        "",
		"headers_last_infra_flow_id_safetynet":  "",
		"fb_ig_device_id":                       []interface{}{},
		"caa_play_integrity_attestation_result": "",
		"client_known_key_hash":                 "",
		"network_bssid":                         nil,
		"lois_settings":                         map[string]interface{}{"lois_token": ""},
		"aac":                                   "",
	}
	return buildBaseBody(s, "z") + "&params=" + marshalParamsEncoded(sp, cp)
}

// --- B7: tos (app, __req=14) ---
// server_params: waterfall_id, is_platform_login, is_from_logged_out, access_flow_version,
//                tos_type, reg_info, flow_info, current_step=8, INTERNAL_INFRA_screen_id
// (reg_context thêm vào nếu có)
// client_input_params: lois_settings, machine_id, cloud_trust_token, block_store_machine_id, aac
func buildB7Body(s *RegSession) string {
	sp := map[string]interface{}{
		"waterfall_id":             s.WaterfallID,
		"is_platform_login":        0,
		"is_from_logged_out":       0,
		"access_flow_version":      "pre_mt_behavior",
		"tos_type":                 "vietnam",
		"reg_info":                 regInfoStr(s),
		"flow_info":                flowInfoJSON,
		"current_step":             8,
		"INTERNAL_INFRA_screen_id": "CAA_REG_TERMS_OF_SERVICE",
	}
	if s.RegContext != "" {
		sp["reg_context"] = s.RegContext
	}
	cp := map[string]interface{}{
		"lois_settings":          map[string]interface{}{"lois_token": ""},
		"machine_id":             "",
		"cloud_trust_token":      nil,
		"block_store_machine_id": "",
		"aac":                    "",
	}
	return buildBaseBody(s, "14") + "&params=" + marshalParamsEncoded(sp, cp)
}

// --- B8: create.account.async (action, __req=17) ---
// server_params: + bloks_controller_source (không có reg_context)
// client_input_params: device_id, waterfall_id, machine_id, zero_balance_state,
//                      ck_error, ck_id, ck_nonce, encrypted_msisdn,
//                      headers_last_infra_flow_id, reached_from_tos_screen,
//                      no_contact_perm_email_oauth_token, failed_birthday_year_count,
//                      ig_partially_created_account_*, cloud_trust_token,
//                      network_bssid, lois_settings, aac
func buildB8Body(s *RegSession, eventID string) string {
	sp := actionServerParams(s, 8, eventID, "27453009000059")
	sp["bloks_controller_source"] = "bk_caa_reg_icon_text_list_tos_screen"

	cp := map[string]interface{}{
		"device_id":                                "",
		"waterfall_id":                             s.WaterfallID,
		"machine_id":                               "",
		"zero_balance_state":                       "",
		"ck_error":                                 "",
		"ck_id":                                    "",
		"ck_nonce":                                 "",
		"encrypted_msisdn":                         "",
		"headers_last_infra_flow_id":               "",
		"reached_from_tos_screen":                  1,
		"no_contact_perm_email_oauth_token":        "",
		"failed_birthday_year_count":               "{}",
		"ig_partially_created_account_user_id":     0,
		"ig_partially_created_account_nonce":       "",
		"ig_partially_created_account_nonce_expiry": 0,
		"cloud_trust_token":                        nil,
		"network_bssid":                            nil,
		"lois_settings":                            map[string]interface{}{"lois_token": ""},
		"aac":                                      "",
	}
	return buildBaseBody(s, "17") + "&params=" + marshalParamsEncoded(sp, cp)
}

// buildCookieStringFromHeaders — build cookie string từ Set-Cookie response headers (B8)
// Output: "c_user=UID;xs=2:xxx:2:...:locale=vi_VN;fr=...;datr=..."
func buildCookieStringFromHeaders(setCookies []string, uidFallback, datrFallback string) string {
	cookieMap := make(map[string]string)
	for _, h := range setCookies {
		// "name=value; Path=...; Domain=...; ..."
		parts := strings.SplitN(h, ";", 2)
		kv := strings.SplitN(strings.TrimSpace(parts[0]), "=", 2)
		if len(kv) == 2 && kv[0] != "" {
			cookieMap[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	// Fallback nếu Set-Cookie không có
	if _, ok := cookieMap["c_user"]; !ok && uidFallback != "" {
		cookieMap["c_user"] = uidFallback
	}
	if _, ok := cookieMap["datr"]; !ok && datrFallback != "" {
		cookieMap["datr"] = datrFallback
	}
	var parts []string
	for _, name := range []string{"c_user", "xs", "locale", "fr", "datr"} {
		if v, ok := cookieMap[name]; ok && v != "" {
			parts = append(parts, name+"="+v)
		}
	}
	return strings.Join(parts, ";")
}

// extractAccessTokenFromHeaders — tìm EAAAAU... access token trong Set-Cookie headers.
// Ưu tiên format EAAAAU (Android user token), fallback bất kỳ EAA nếu không tìm thấy.
func extractAccessTokenFromHeaders(setCookies []string) string {
	reUser := regexp.MustCompile(`EAAAAU[A-Za-z0-9_\-+/]{20,}`)
	reAny := regexp.MustCompile(`EAA[A-Za-z0-9_\-+/]{20,}`)
	var fallback string
	for _, h := range setCookies {
		if m := reUser.FindString(h); m != "" {
			return m
		}
		if fallback == "" {
			if m := reAny.FindString(h); m != "" {
				fallback = m
			}
		}
	}
	return fallback
}

// parseCookiesFromB8Body — extract cookie values từ B8 response body JSON
// B8 response chứa "c_user" / "xs" / "locale" / "fr" / "datr" embedded trong JSON
func parseCookiesFromB8Body(body, uid, datr string) string {
	cookieMap := make(map[string]string)
	if uid != "" {
		cookieMap["c_user"] = uid
	}
	// xs cookie
	if xs := extractJSONStr(body, "xs"); xs != "" {
		cookieMap["xs"] = xs
	}
	// locale
	if loc := extractJSONStr(body, "locale"); loc != "" {
		cookieMap["locale"] = loc
	}
	// fr cookie
	if fr := extractJSONStr(body, "fr"); fr != "" {
		cookieMap["fr"] = fr
	}
	// datr
	if d := extractJSONStr(body, "datr"); d != "" {
		cookieMap["datr"] = d
	} else if datr != "" {
		cookieMap["datr"] = datr
	}

	var result []string
	for _, name := range []string{"c_user", "xs", "locale", "fr", "datr"} {
		if v, ok := cookieMap[name]; ok && v != "" {
			result = append(result, name+"="+v)
		}
	}
	return strings.Join(result, ";")
}

// extractAccessToken — find EAAAAU... access token từ Android login response body.
// Ưu tiên format EAAAAU (Android user token), fallback bất kỳ EAA nếu không tìm thấy.
func extractAccessToken(body string) string {
	if m := regexp.MustCompile(`EAAAAU[A-Za-z0-9_\-+/]{20,}`).FindString(body); m != "" {
		return m
	}
	return regexp.MustCompile(`EAA[A-Za-z0-9_\-+/]{20,}`).FindString(body)
}

// parseBirthday — "DD-MM-YYYY" → Unix timestamp (UTC midnight)
func parseBirthday(birthday string) int64 {
	parts := strings.Split(birthday, "-")
	if len(parts) != 3 {
		return 0
	}
	day, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])
	year, _ := strconv.Atoi(parts[2])
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return t.Unix()
}

// parseRegInfoFromResponse — extract reg_info JSON string từ Bloks response
func parseRegInfoFromResponse(response string) string {
	return extractJSONStr(response, "reg_info")
}

// updateRegInfoFields — parse reg_info hiện tại, merge fields mới, serialize lại
// Dùng để update manually vì Bloks DSL responses không trả về reg_info ở outer level
func updateRegInfoFields(s *RegSession, fields map[string]interface{}) {
	src := s.RegInfo
	if src == "" {
		src = buildInitialRegInfo()
	}
	var ri map[string]interface{}
	if err := json.Unmarshal([]byte(src), &ri); err != nil {
		ri = map[string]interface{}{}
	}
	for k, v := range fields {
		ri[k] = v
	}
	b, _ := json.Marshal(ri)
	s.RegInfo = string(b)
}

// addScreenVisited — thêm screen vào reg_info.screen_visited (không trùng)
func addScreenVisited(s *RegSession, screen string) {
	src := s.RegInfo
	if src == "" {
		src = buildInitialRegInfo()
	}
	var ri map[string]interface{}
	if err := json.Unmarshal([]byte(src), &ri); err != nil {
		ri = map[string]interface{}{}
	}
	var screens []interface{}
	if sv, ok := ri["screen_visited"]; ok {
		if slice, ok2 := sv.([]interface{}); ok2 {
			screens = slice
		}
	}
	for _, existing := range screens {
		if existing == screen {
			return // đã có, bỏ qua
		}
	}
	ri["screen_visited"] = append(screens, screen)
	b, _ := json.Marshal(ri)
	s.RegInfo = string(b)
}

// toInternationalPhone — chuyển số về dạng quốc tế "+countrycode..."
// Nếu đã có "+" → giữ nguyên (non-VN countries)
// Nếu dạng "0xxx" → VN: "+84xxx"
func toInternationalPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if strings.HasPrefix(phone, "+") {
		return phone
	}
	if strings.HasPrefix(phone, "0") {
		return "+84" + phone[1:]
	}
	return phone
}

// parseRegContextFromResponse — extract reg_context từ Bloks response
func parseRegContextFromResponse(response string) string {
	return extractJSONStr(response, "reg_context")
}

// parseUIDFromResponse — extract UID từ response B8
// API_RegFB.txt: B8 success response chứa "currentUser":61575441153102 trong ajaxUpdateAfterLogin
func parseUIDFromResponse(response string) string {
	for _, pattern := range []string{
		`"currentUser"\s*:\s*(\d{10,})`,  // B8: ajaxUpdateAfterLogin.currentUser (PRIMARY)
		`"uid"\s*:\s*"(\d+)"`,
		`"user_id"\s*:\s*"(\d+)"`,
		`"uid"\s*:\s*(\d+)`,
		`"user_id"\s*:\s*(\d+)`,
		`c_user=(\d+)`,
	} {
		if v := extractFirstReg(response, pattern); v != "" {
			return v
		}
	}
	return ""
}
