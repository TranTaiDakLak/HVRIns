// body.go — iOS555 create.account request body builder (single-shot).
//
// Chiến lược: gửi 1 request duy nhất tới create.account.async với reg_context=null
// + full reg_info + current_step=8 (giống single-shot của Android s563, đã proven
// chạy trên cùng CAA backend). Body dựng bằng encoding/json — KHÔNG escape tay.
//
// Cấu trúc lồng:
//
//	form: ...&variables=<urlencoded JSON>
//	variables.params.params = JSON-string của {server_params, client_input_params}
//	server_params.reg_info  = JSON-string của toàn bộ field đăng ký
package ios462

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// regFields — dữ liệu đăng ký 1 account (đã resolve từ RegInput).
type regFields struct {
	firstName         string
	lastName          string
	birthday          string // "DD-MM-YYYY"
	gender            int    // 1=nữ, 2=nam
	contactpoint      string // email hoặc phone
	cpType            string // "email" | "phone"
	password          string // plaintext (for result storage)
	encryptedPassword string // #PWD_WILDE:2:... hoặc #PWD_FB4A:0:...
}

// buildCreateAccountBody dựng form-urlencoded body cho create.account.async.
func buildCreateAccountBody(p IOSProfile, f regFields) (string, error) {
	encPassword := f.encryptedPassword

	pixelRatio := 2
	if p.Device.FBSS == "3" {
		pixelRatio = 3
	}

	regInfo := buildRegInfo(p, f, encPassword)
	regInfoJSON, err := json.Marshal(regInfo)
	if err != nil {
		return "", fmt.Errorf("marshal reg_info: %w", err)
	}

	serverParams := map[string]any{
		"reg_info":                          string(regInfoJSON),
		"reg_context":                       nil,
		"current_step":                      8,
		"device_id":                         p.DeviceID,
		"family_device_id":                  p.FamilyDeviceID,
		"machine_id":                        p.MachineID,
		"waterfall_id":                      p.WaterfallID,
		"event_request_id":                  uuid.New().String(),
		"flow_info":                         `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`,
		"login_surface":                     "aymh_one_tap",
		"login_entry_point":                 "logged_out",
		"cloud_trust_token":                 p.CloudTrustID,
		"access_flow_version":               "pre_mt_behavior",
		"offline_experiment_group":          nil,
		"layered_homepage_experiment_group": "not_in_experiment",
		"bloks_controller_source":           "BloksCAARegUtils::getTriggerAccountSetupAction",
		"is_from_logged_out":                1,
		"is_from_logged_in_switcher":        0,
		"is_platform_login":                 0,
		"INTERNAL__latency_qpl_marker_id":   36707139,
		"INTERNAL__latency_qpl_instance_id": time.Now().UnixNano() % 1000000000000000,
	}

	clientInputParams := map[string]any{
		"device_id":               p.DeviceID,
		"family_device_id":        p.FamilyDeviceID,
		"machine_id":              p.MachineID,
		"block_store_machine_id":  "",
		"cloud_trust_token":       p.CloudTrustID,
		"network_bssid":           nil,
		"aac":                     "",
		"fb_ig_device_id":         []any{},
		"encrypted_msisdn":        "",
		"reached_from_tos_screen": 1,
		"lois_settings":           map[string]any{"lois_token": ""},
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
			"app_id":              createAccountAppID,
			"params":              string(innerJSON),
		},
		"generic_attachment_tall_cover_image_width": 670,
		"nt_context": map[string]any{
			"theme_params": []any{
				map[string]any{
					"design_system_name": "FDS",
					"value":              []string{"DARKER_PRIMARY_DEEMPHASIZED_BUTTON_BACKGROUND_TEST", "MEDIA_INNER_BORDER_WHITE_ALPHA_08_FOR_DARK_TEST", "DEFAULT"},
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
		"enable_voiceover_gating_for_accessibility_caption": "true",
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
		"fb_api_req_friendly_name=" + createAccountFriendlyName,
		"client_doc_id=" + docIDAction,
		"fb_api_client_context=" + url.QueryEscape(`{"is_background":"0"}`),
		"variables=" + url.QueryEscape(string(variablesJSON)),
	}
	return strings.Join(parts, "&"), nil
}

// buildRegInfo dựng object reg_info — toàn bộ field đăng ký.
// Field set port từ Android s563 (đã proven trên CAA backend), giá trị đổi sang iOS.
func buildRegInfo(p IOSProfile, f regFields, encPassword string) map[string]any {
	return map[string]any{
		"first_name":                              f.firstName,
		"last_name":                               f.lastName,
		"full_name":                               f.firstName + " " + f.lastName,
		"contactpoint":                            f.contactpoint,
		"ar_contactpoint":                         nil,
		"contactpoint_type":                       f.cpType,
		"is_using_unified_cp":                     nil,
		"unified_cp_screen_variant":               nil,
		"is_cp_auto_confirmed":                    false,
		"is_cp_auto_confirmable":                  false,
		"is_cp_claimed":                           false,
		"confirmation_code":                       nil,
		"birthday":                                f.birthday,
		"birthday_derived_from_age":               nil,
		"did_use_age":                             false,
		"gender":                                  f.gender,
		"use_custom_gender":                       false,
		"custom_gender":                           nil,
		"encrypted_password":                      encPassword,
		"username":                                nil,
		"username_prefill":                        nil,
		"fb_conf_source":                          nil,
		"device_id":                               p.DeviceID,
		"ig4a_qe_device_id":                       nil,
		"family_device_id":                        p.FamilyDeviceID,
		"user_id":                                 nil,
		"safetynet_token":                         nil,
		"skip_slow_rel_check":                     false,
		"safetynet_response":                      nil,
		"machine_id":                              p.MachineID,
		"profile_photo":                           nil,
		"profile_photo_id":                        nil,
		"profile_photo_upload_id":                 nil,
		"avatar":                                  nil,
		"email_oauth_token_no_contact_perm":       nil,
		"email_oauth_token":                       nil,
		"email_oauth_tokens":                      []any{},
		"should_skip_two_step_conf":               nil,
		"openid_tokens_for_testing":               nil,
		"encrypted_msisdn":                        nil,
		"encrypted_msisdn_for_safetynet":          nil,
		"cached_headers_safetynet_info":           nil,
		"should_skip_headers_safetynet":           nil,
		"headers_last_infra_flow_id":              nil,
		"headers_last_infra_flow_id_safetynet":    nil,
		"headers_flow_id":                         uuid.New().String(),
		"was_headers_prefill_available":           false,
		"sso_enabled":                             nil,
		"existing_accounts":                       nil,
		"used_ig_birthday":                        nil,
		"create_new_to_app_account":               nil,
		"skip_session_info":                       nil,
		"ck_error":                                nil,
		"ck_id":                                   nil,
		"ck_nonce":                                nil,
		"should_save_password":                    true,
		"fb_access_token":                         nil,
		"is_msplit_reg":                           nil,
		"is_spectra_reg":                          nil,
		"dema_account_consent_given":              nil,
		"user_id_of_msplit_creator":               nil,
		"msplit_creator_nonce":                    nil,
		"dma_data_combination_consent_given":      nil,
		"xapp_accounts":                           nil,
		"fb_device_id":                            nil,
		"fb_machine_id":                           nil,
		"ig_device_id":                            nil,
		"ig_machine_id":                           nil,
		"should_skip_nta_upsell":                  nil,
		"big_blue_token":                          nil,
		"caa_reg_flow_source":                     "aymh_multi_profiles_native_integration_point",
		"ig_authorization_token":                  nil,
		"full_sheet_flow":                         false,
		"crypted_user_id":                         nil,
		"is_caa_perf_enabled":                     true,
		"is_preform":                              false,
		"should_show_rel_error":                   false,
		"ignore_suma_check":                       false,
		"dismissed_login_upsell_with_cna":         false,
		"ignore_existing_login":                   false,
		"ignore_existing_login_from_suma":         false,
		"ignore_existing_login_after_errors":      false,
		"suggested_first_name":                    nil,
		"suggested_last_name":                     nil,
		"suggested_full_name":                     nil,
		"frl_authorization_token":                 nil,
		"post_form_errors":                        nil,
		"skip_step_without_errors":                false,
		"existing_account_exact_match_checked":    true,
		"existing_account_fuzzy_match_checked":    false,
		"email_oauth_exists":                      false,
		"confirmation_code_send_error":            nil,
		"is_too_young":                            false,
		"source_account_type":                     nil,
		"whatsapp_installed_on_client":            false,
		"confirmation_medium":                     nil,
		"source_credentials_type":                 nil,
		"source_cuid":                             nil,
		"source_account_reg_info":                 nil,
		"soap_creation_source":                    nil,
		"source_account_type_to_reg_info":         nil,
		"registration_flow_id":                    uuid.New().String(),
		"should_skip_youth_tos":                   false,
		"is_youth_regulation_flow_complete":       false,
		"is_on_cold_start":                        false,
		"email_prefilled":                         false,
		"cp_confirmed_by_auto_conf":               false,
		"in_sowa_experiment":                      false,
		"youth_regulation_config":                 nil,
		"eligible_to_flash_call_in_ig4a":          false,
		"attestation_result":                      nil,
		"request_data_and_challenge_nonce_string": nil,
		"confirmed_cp_and_code":                   nil,
		"notification_callback_id":                nil,
		"reg_suma_state":                          0,
		"is_msplit_neutral_choice":                false,
		"msg_previous_cp":                         nil,
		"ntp_import_source_info":                  nil,
		"youth_consent_decision_time":             nil,
		"should_show_spi_before_conf":             true,
		"google_oauth_account":                    nil,
		"is_reg_request_from_ig_suma":             false,
		"is_toa_reg":                              false,
		"is_threads_public":                       false,
		"spc_import_flow":                         false,
		"caa_play_integrity_attestation_result":   nil,
		"client_known_key_hash":                   nil,
		"flash_call_provider":                     nil,
		"spc_birthday_input":                      false,
		"failed_birthday_year_count":              nil,
		"user_presented_medium_source":            nil,
		"user_opted_out_of_ntp":                   nil,
		"is_from_registration_reminder":           false,
		"show_youth_reg_in_ig_spc":                false,
		"fb_suma_is_high_confidence":              nil,
		"screen_visited": []string{
			"CAA_REG_WELCOME_SCREEN",
			"bloks.caa.reg.birthday",
			"CAA_REG_CONTACT_POINT_EMAIL",
			"CAA_REG_CONTACT_POINT_PHONE",
			"CAA_REG_PASSWORD",
			"CAA_REG_SAVE_PASSWORD_CREDENTIALS",
			"CAA_REG_CONFIRMATION_SCREEN",
		},
		"fb_email_login_upsell_skip_suma_post_tos":  false,
		"fb_suma_is_from_email_login_upsell":        false,
		"fb_suma_is_from_phone_login_upsell":        false,
		"should_prefill_cp_in_ar":                   nil,
		"ig_partially_created_account_user_id":      nil,
		"ig_partially_created_account_nonce":        nil,
		"ig_partially_created_account_nonce_expiry": nil,
		"force_sessionless_nux_experience":          false,
		"has_seen_suma_landing_page_pre_conf":       false,
		"has_seen_suma_candidate_page_pre_conf":     false,
		"has_seen_confirmation_screen":              false,
		"suma_on_conf_threshold":                    -1,
		"should_show_error_msg":                     true,
		"th_profile_photo_token":                    nil,
		"attempted_silent_auth_in_fb":               false,
		"attempted_silent_auth_in_ig":               false,
		"cp_suma_results_map":                       nil,
		"source_username":                           nil,
		"next_uri":                                  nil,
		"should_use_next_uri":                       nil,
		"linking_entry_point":                       nil,
		"starter_pack_name":                         nil,
		"wa_data_bundle":                            nil,
		"tos_accepted_on_profile_info":              nil,
	}
}

// buildCreateAccountRound2 dựng body cho LẦN GỌI THỨ 2 create.account.async.
// Dùng partial tokens từ response round 1 (nosess). Map đúng cấu trúc capture [797]:
//   - server_params: partial tokens + device signals + step=8 + flags
//   - client_input_params: aac JSON string + lois + block_store + bssid
//   - KHÔNG có reg_info — account đã tạo, server tham chiếu qua fb_partially_created_reg_info.
func buildCreateAccountRound2(p IOSProfile, t *partialTokens) (string, error) {
	if t == nil || t.PartiallyCreated == "" || t.Srnonce == "" {
		return "", fmt.Errorf("round2: thiếu partial tokens")
	}
	pixelRatio := 2
	if p.Device.FBSS == "3" {
		pixelRatio = 3
	}
	lat := time.Now().UnixNano() % 1000000000000000
	flowInfo := `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`

	serverParams := map[string]any{
		"x_app_device_signals": map[string]any{
			"DEVICE_ID":  p.DeviceID,
			"MACHINE_ID": p.MachineID,
		},
		"login_surface":                     "unknown",
		"srnonce":                           t.Srnonce,
		"fb_partially_created_reg_info":     t.PartiallyCreated,
		"is_from_logged_in_switcher":        0,
		"is_platform_login":                 0,
		"access_flow_version":               "pre_mt_behavior",
		"cloud_trust_token":                 p.CloudTrustID,
		"flow_info":                         flowInfo,
		"INTERNAL__latency_qpl_instance_id": lat,
		"login_entry_point":                 "logged_out",
		"device_id":                         p.DeviceID,
		"should_ignore_existing_login":      1,
		"waterfall_id":                      p.WaterfallID,
		"INTERNAL__latency_qpl_marker_id":   36707139,
		"is_from_logged_out":                0,
		"should_ignore_suma_check":          1,
		"layered_homepage_experiment_group": nil,
		"is_from_registration_flow":         1,
		"reg_context":                       t.RegContext,
		"current_step":                      8,
		"machine_id":                        p.MachineID,
		"bloks_controller_source":           "BloksCAARegUtils::getTriggerAccountSetupAction",
		"offline_experiment_group":          nil,
		"sessionless_flow_info":             flowInfo,
		"family_device_id":                  p.FamilyDeviceID,
	}
	if t.EncryptedProps != "" {
		serverParams["fb_encrypted_partial_new_account_properties"] = t.EncryptedProps
	}
	if t.SessionlessCUID != "" {
		serverParams["sessionless_crypted_user_id"] = t.SessionlessCUID
	}

	aacObj := map[string]any{
		"aac_init_timestamp": time.Now().Unix(),
		"aacjid":             uuid.New().String(),
		"aaccs":              randBase64URL(44),
	}
	aacBytes, _ := json.Marshal(aacObj)

	clientInputParams := map[string]any{
		"block_store_machine_id": "",
		"lois_settings":          map[string]any{"lois_token": ""},
		"network_bssid":          nil,
		"aac":                    string(aacBytes),
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
			"app_id":              createAccountAppID,
			"params":              string(innerJSON),
		},
		"generic_attachment_tall_cover_image_width": 670,
		"nt_context": map[string]any{
			"theme_params": []any{
				map[string]any{
					"design_system_name": "FDS",
					"value":              []string{"DARKER_PRIMARY_DEEMPHASIZED_BUTTON_BACKGROUND_TEST", "MEDIA_INNER_BORDER_WHITE_ALPHA_08_FOR_DARK_TEST", "DEFAULT"},
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
		"enable_voiceover_gating_for_accessibility_caption": "true",
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
		"fb_api_req_friendly_name=" + createAccountFriendlyName,
		"client_doc_id=" + docIDAction,
		"fb_api_client_context=" + url.QueryEscape(`{"is_background":"0"}`),
		"variables=" + url.QueryEscape(string(variablesJSON)),
	}
	return strings.Join(parts, "&"), nil
}
