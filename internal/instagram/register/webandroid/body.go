// body.go — Form data builder cho Web Android register POST
// Mapping từ C#: FacebookApiFormDataBuilder.CreateAccountChromeAndroidFormData()
// POST tới: m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.create.account.async&type=action&__bkv={versioningID}
package webandroid

import (
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
)

// regParams chứa tất cả thông số cần thiết để build body
type regParams struct {
	tokens       pageTokens
	firstName    string
	lastName     string
	contactpoint string // số điện thoại hoặc email
	cpType       string // "phone" hoặc "email"
	birthday     string // "DD-MM-YYYY"
	gender       int    // 1=female, 2=male
	password     string // plain text (sẽ dùng #PWD_BROWSER:0:ts:pass)
	waterfallID  string // UUID
}

// buildRegisterBody tạo form-urlencoded body cho register POST
// Mapping chính xác từ C#: CreateAccountChromeAndroidFormData()
func buildRegisterBody(p regParams) string {
	ts := fmt.Sprintf("%d", time.Now().Unix())

	// fullName raw — url.QueryEscape(outerParams) ở cuối buildParamsValue đã encode 1 lần
	// C# pre-encode trong template, Go encode 1 lần cuối → cùng kết quả
	fullName := p.firstName + " " + p.lastName

	// 3 UUIDs khác nhau (C#: Guid.NewGuid().ToString())
	eventRequestID := uuid.New().String()
	headersFlowID := uuid.New().String()
	regFlowID := uuid.New().String()

	encPassword := fmt.Sprintf("#PWD_BROWSER:0:%s:%s", ts, p.password)

	// Build body theo đúng format C# CreateAccountChromeAndroidFormData
	// Các trường __hs, dpr, __ccg là fixed values từ C# source
	body := fmt.Sprintf(
		"__aaid=0&__user=0&__a=1&__req=1f"+
			"&__hs=20210.BP%%3Awbloks_caa_pkg.2.0...0"+
			"&dpr=3"+
			"&__ccg=GOOD"+
			"&__rev=%s"+
			"&__s="+
			"&__hsi=%s"+
			"&__dyn="+
			"&fb_dtsg=%s"+
			"&jazoest=%s"+
			"&lsd=%s"+
			"&params=%s",
		p.tokens.spinR,
		p.tokens.hsi,
		p.tokens.fbDtsg, // C# gửi raw, không URL-encode
		p.tokens.jazoest,
		p.tokens.lsd,
		buildParamsValue(eventRequestID, headersFlowID, regFlowID, p, p.contactpoint, fullName, encPassword, ts),
	)

	return body
}

// buildParamsValue tạo giá trị URL-encoded cho field params=
// Đây là triple-encoded JSON theo đúng format C# (params > server_params > reg_info)
// cp và fullName ĐÃ được URL-encode trước (khớp C#: WebUtility.UrlEncode)
// rồi url.QueryEscape(outerParams) encode lần 2 ở cuối — đúng C# double-encode pattern.
func buildParamsValue(
	eventRequestID, headersFlowID, regFlowID string,
	p regParams,
	cp, fullName, encPassword, ts string,
) string {
	// reg_info JSON (innermost) — triple-escaped trong C#
	// gender: 1=female, 2=male (FB dùng số nguyên)
	regInfo := fmt.Sprintf(
		`{"first_name":"%s","last_name":"%s","full_name":"%s","contactpoint":"%s","ar_contactpoint":null,"contactpoint_type":"%s","is_using_unified_cp":false,"unified_cp_screen_variant":null,"is_cp_auto_confirmed":false,"is_cp_auto_confirmable":false,"confirmation_code":null,"birthday":"%s","did_use_age":false,"gender":%d,"use_custom_gender":false,"custom_gender":null,"encrypted_password":"%s","username":null,"username_prefill":null,"fb_conf_source":null,"device_id":null,"ig4a_qe_device_id":null,"family_device_id":null,"nta_eligibility_reason":null,"ig_nta_test_group":null,"user_id":null,"safetynet_token":null,"safetynet_response":null,"machine_id":null,"profile_photo":null,"profile_photo_id":null,"profile_photo_upload_id":null,"avatar":null,"email_oauth_token_no_contact_perm":null,"email_oauth_token":null,"email_oauth_tokens":[],"should_skip_two_step_conf":null,"openid_tokens_for_testing":null,"encrypted_msisdn":null,"encrypted_msisdn_for_safetynet":null,"cached_headers_safetynet_info":null,"should_skip_headers_safetynet":null,"headers_last_infra_flow_id":null,"headers_last_infra_flow_id_safetynet":null,"headers_flow_id":"%s","was_headers_prefill_available":false,"sso_enabled":null,"existing_accounts":null,"used_ig_birthday":null,"sync_info":null,"create_new_to_app_account":null,"skip_session_info":null,"ck_error":null,"ck_id":null,"ck_nonce":null,"should_save_password":true,"horizon_synced_username":null,"fb_access_token":null,"horizon_synced_profile_pic":null,"is_identity_synced":false,"is_msplit_reg":null,"is_spectra_reg":null,"user_id_of_msplit_creator":null,"msplit_creator_nonce":null,"dma_data_combination_consent_given":null,"xapp_accounts":null,"fb_device_id":null,"fb_machine_id":null,"ig_device_id":null,"ig_machine_id":null,"should_skip_nta_upsell":null,"big_blue_token":null,"skip_sync_step_nta":null,"caa_reg_flow_source":"login_home_native_integration_point","ig_authorization_token":null,"full_sheet_flow":false,"crypted_user_id":null,"is_caa_perf_enabled":false,"is_preform":true,"ignore_suma_check":false,"ignore_existing_login":false,"ignore_existing_login_from_suma":false,"ignore_existing_login_after_errors":false,"suggested_first_name":null,"suggested_last_name":null,"suggested_full_name":null,"replace_id_sync_variant":null,"is_redirect_from_nta_replace_id_sync_variant":false,"frl_authorization_token":null,"post_form_errors":null,"skip_step_without_errors":false,"existing_account_exact_match_checked":true,"existing_account_fuzzy_match_checked":false,"email_oauth_exists":false,"confirmation_code_send_error":null,"is_too_young":false,"source_account_type":null,"whatsapp_installed_on_client":false,"confirmation_medium":null,"source_credentials_type":null,"source_cuid":null,"source_account_reg_info":null,"soap_creation_source":null,"source_account_type_to_reg_info":null,"registration_flow_id":"%s","should_skip_youth_tos":false,"is_youth_regulation_flow_complete":false,"is_on_cold_start":false,"email_prefilled":false,"cp_confirmed_by_auto_conf":false,"auto_conf_info":null,"in_sowa_experiment":false,"youth_regulation_config":null,"conf_allow_back_nav_after_change_cp":null,"conf_bouncing_cliff_screen_type":null,"conf_show_bouncing_cliff":null,"eligible_to_flash_call_in_ig4a":false,"flash_call_permissions_status":null,"attestation_result":null,"request_data_and_challenge_nonce_string":null,"confirmed_cp_and_code":null,"notification_callback_id":null,"reg_suma_state":0,"is_msplit_neutral_choice":false,"msg_previous_cp":null,"ntp_import_source_info":null,"youth_consent_decision_time":null,"should_show_spi_before_conf":true,"google_oauth_account":null,"is_reg_request_from_ig_suma":false,"device_emails":[],"is_toa_reg":false,"is_threads_public":false,"spc_import_flow":false,"caa_play_integrity_attestation_result":null,"flash_call_provider":null,"spc_birthday_input":false,"failed_birthday_year_count":null,"user_presented_medium_source":null,"user_opted_out_of_ntp":null,"is_from_registration_reminder":false,"show_youth_reg_in_ig_spc":false,"fb_suma_combined_landing_candidate_variant":"control","fb_suma_is_high_confidence":null,"screen_visited":["CAA_REG_WELCOME_SCREEN","bloks.caa.reg.birthday","CAA_REG_CONTACT_POINT_PHONE","CAA_REG_PASSWORD","CAA_REG_SAVE_PASSWORD_CREDENTIALS"],"fb_email_login_upsell_skip_suma_post_tos":false,"fb_suma_is_from_email_login_upsell":false,"fb_suma_is_from_phone_login_upsell":false,"fb_suma_login_upsell_skipped_warmup":false,"fb_suma_login_upsell_show_list_cell_link":false,"should_prefill_cp_in_ar":true,"ig_partially_created_account_user_id":null,"ig_partially_created_account_nonce":null,"ig_partially_created_account_nonce_expiry":null,"has_seen_suma_landing_page_pre_conf":false,"has_seen_suma_candidate_page_pre_conf":false,"nta_login_footer_variant":"control","is_keyboard_autofocus":null}`,
		p.firstName, p.lastName, fullName,
		cp, p.cpType,
		p.birthday, p.gender,
		encPassword,
		headersFlowID,
		regFlowID,
	)

	// server_params JSON (middle level)
	serverParams := fmt.Sprintf(
		`{"event_request_id":"%s","reg_info":"%s","flow_info":"{\"flow_name\":\"new_to_family_fb_default\",\"flow_type\":\"ntf\"}","current_step":8,"INTERNAL__latency_qpl_marker_id":36707139,"INTERNAL__latency_qpl_instance_id":"89920502200020","device_id":null,"family_device_id":null,"waterfall_id":"%s","offline_experiment_group":null,"layered_homepage_experiment_group":null,"is_platform_login":0,"is_from_logged_in_switcher":0,"is_from_logged_out":0,"access_flow_version":"pre_mt_behavior"}`,
		eventRequestID,
		escapeJSONString(regInfo),
		p.waterfallID,
	)

	// client_input_params JSON
	clientInputParams := fmt.Sprintf(
		`{"device_id":"","waterfall_id":"%s","machine_id":"","ck_error":"","ck_id":"","ck_nonce":"","encrypted_msisdn":"","headers_last_infra_flow_id":"","reached_from_tos_screen":1,"no_contact_perm_email_oauth_token":"","failed_birthday_year_count":"{}","ig_partially_created_account_user_id":0,"ig_partially_created_account_nonce":"","ig_partially_created_account_nonce_expiry":0,"lois_settings":{"lois_token":""}}`,
		p.waterfallID,
	)

	// outer params JSON
	outerParams := fmt.Sprintf(
		`{"params":"{\"server_params\":%s,\"client_input_params\":%s}"}`,
		escapeJSONString(serverParams),
		escapeJSONString(clientInputParams),
	)

	return url.QueryEscape(outerParams)
}

// escapeJSONString escape chuỗi JSON để nhúng vào chuỗi JSON khác
// C# tương đương: dùng \ trước mỗi " trong chuỗi con
func escapeJSONString(s string) string {
	// Escape backslash trước, sau đó escape dấu nháy đôi
	s = replaceAll(s, `\`, `\\`)
	s = replaceAll(s, `"`, `\"`)
	return s
}

func replaceAll(s, old, new string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); {
		if len(s)-i >= len(old) && s[i:i+len(old)] == old {
			result = append(result, new...)
			i += len(old)
		} else {
			result = append(result, s[i])
			i++
		}
	}
	return string(result)
}

// postURL tạo URL POST endpoint với versioningID
func postURL(versioningID string) string {
	return fmt.Sprintf(
		"https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.create.account.async&type=action&__bkv=%s",
		versioningID,
	)
}
