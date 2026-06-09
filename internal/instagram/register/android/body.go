// body.go — Android Register V3 body builder.
//
// Port 1:1 từ C# FacebookRegisterAPIAndroidV2.CreateAccountVariables_v22 (L539-872)
// + RegisterFormDataV3 (L518-536).
// Source: E:\WEMAKE\FULL-REG-CLONE-HAVU\VerifyCloneVIP\API\FbAndroidApi\FacebookRegisterAPIAndroidV2.cs
//
// KHÁC với V1/V3 pre-existing Go: V22 schema dùng
//   - JSON string + EscapeForJsonStringValue + WebUtility.UrlEncode(l0) 1 lần
//   - Field set bổ sung: age_range, accounts_list_client, device_network_info, aac, ...
//   - attestation_result object thực (keyHash/data/signature), không phải {}
//   - safetynet_token có giá trị thực (random), không null
//   - should_save_password = false (V22), not true như V3 S23
//   - family_device_id = deviceid (không phải random FamilyDeviceID)
//   - x-zero-f-device-id = deviceid (header)
package android

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"HVRIns/internal/instagram/fakeinfo"
)

// docID V3 — C# CreateAccountVariables_v22 line 530.
const docID = "1199408042526631289603660492"

// friendlyName — C# L528.
const friendlyName = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.create.account.async"

// bloksVer V3 — C# L541.
const bloksVer = "d90663010f8c230bedf28906f2bac9c1d1f532a275373050778e36e76a7cb999"

// stylesId V3 — C# L542.
const stylesId = "6100e7e89411ccf67ace027cedecd84f"

// appId — C# L543.
const appId = "com.bloks.www.bloks.caa.reg.create.account.async"

// escJSON — replicate C# EscapeForJsonStringValue (V2 L403-407).
// Escape `\` thành `\\`, `"` thành `\"`. Dùng khi embed 1 JSON string vào level cha.
func escJSON(s string) string {
	return strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s)
}

// buildRegisterBody sinh form-urlencoded body cho /graphql (V3).
// Port C# RegisterFormDataV3 (V2.cs L518-536).
func buildRegisterBody(profile fakeinfo.FullRegProfile, encPassword, contactpoint, contactpointType, locale string) string {
	variables := createAccountVariablesV22(profile, encPassword, contactpoint, contactpointType)
	traceID := uuid.New().String()

	parts := []string{
		"method=post",
		"pretty=false",
		"format=json",
		"server_timestamps=true",
		"locale=" + locale,
		"purpose=fetch",
		"fb_api_req_friendly_name=" + friendlyName,
		"fb_api_caller_class=graphservice",
		"client_doc_id=" + docID,
		"fb_api_client_context=%7B%22is_background%22%3Afalse%7D",
		"variables=" + variables,
		"fb_api_analytics_tags=%5B%22GraphServices%22%5D",
		"client_trace_id=" + traceID,
	}
	return strings.Join(parts, "&")
}

// createAccountVariablesV22 port C# CreateAccountVariables_v22 (L539-872).
//
// Cấu trúc (build theo thứ tự ngược — từ level-4 trong ra level-0 ngoài):
//   L4 regInfoJson   — JSON plain 180+ fields
//   L4 aacJsonContent— JSON plain 3 fields (embedded as string vào client_input_params.aac)
//   L3 server_params — JSON chứa reg_info = escJSON(regInfoJson) làm string value
//   L3 client_input  — JSON chứa aac = escJSON(aacJsonContent)
//   L3 body          — {client_input_params, server_params}
//   L2               — {"params": escJSON(L3)}
//   L1 params string — escJSON(L2)
//   L0 outer         — {"params": {"params": L1PS, "bloks_versioning_id", "app_id"}, "scale": "3", "nt_context": {...}}
//   Return url.QueryEscape(L0)
func createAccountVariablesV22(profile fakeinfo.FullRegProfile, encPassword, contactpoint, contactpointType string) string {
	ts := time.Now().Unix()
	qplID := createLatencyQplInstanceIDV2()
	flowID := uuid.New().String()
	eventReqID := uuid.New().String()
	regFlowID := uuid.New().String()

	isEmail := contactpointType == "email"

	machineID := profile.MachineID2
	if machineID == "" {
		machineID = randomAlphanumeric(24)
	}

	keyHash := generateKeyHashV2()
	attData := generateAttestationDataV2(contactpoint)
	attSig := generateAttestationSignatureV2()
	safetynetTok := generateSafetyNetTokenV2(ts)

	aacjid := uuid.New().String()
	aaccs := randomBase64URLRaw(32)
	// aac is a JSON string value containing {aac_init_timestamp, aacjid, aaccs}
	aacJSONContent := fmt.Sprintf(
		`{"aac_init_timestamp":%d,"aacjid":"%s","aaccs":"%s"}`,
		ts, aacjid, aaccs,
	)

	flowInfoContent := `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`

	// ── reg_info JSON plain (L4) ────────────────────────────────────────────
	// Port C# regInfoJson (V2.cs L580-787). Nội dung là JSON 1 cấp (flat), chứa ~180 fields.
	screenVisited := `["CAA_REG_WELCOME_SCREEN","bloks.caa.reg.birthday","CAA_REG_CONTACT_POINT_PHONE","CAA_REG_PASSWORD","CAA_REG_SAVE_PASSWORD_CREDENTIALS"]`
	if isEmail {
		screenVisited = `["CAA_REG_WELCOME_SCREEN","bloks.caa.reg.birthday","CAA_REG_CONTACT_POINT_PHONE","CAA_REG_CONTACT_POINT_EMAIL","CAA_REG_PASSWORD","CAA_REG_SAVE_PASSWORD_CREDENTIALS"]`
	}

	cpSumaResultsMap := "null"
	if isEmail {
		cpSumaResultsMap = fmt.Sprintf(
			`{"%s":{"suma_cuids":null,"has_high_conf_candidate":false,"highest_suma_score":0}}`,
			escJSON(contactpoint),
		)
	}

	deviceID := profile.DeviceID
	deviceNetworkInfo := buildDeviceNetworkInfo(profile)

	var sb strings.Builder
	sb.WriteByte('{')
	// Identity
	writeKV(&sb, "first_name", strQ(escJSON(profile.FirstName)))
	writeKV(&sb, "last_name", strQ(escJSON(profile.LastName)))
	writeKV(&sb, "full_name", "null")
	writeKV(&sb, "contactpoint", strQ(escJSON(contactpoint)))
	writeKV(&sb, "ar_contactpoint", "null")
	writeKV(&sb, "contactpoint_type", strQ(contactpointType))
	writeKV(&sb, "is_using_unified_cp", "false")
	writeKV(&sb, "unified_cp_screen_variant", "null")
	writeKV(&sb, "is_cp_auto_confirmed", "false")
	writeKV(&sb, "is_cp_auto_confirmable", "false")
	writeKV(&sb, "is_cp_claimed", "false")
	writeKV(&sb, "confirmation_code", "null")
	writeKV(&sb, "birthday", strQ(profile.Birthday))
	writeKV(&sb, "birthday_derived_from_age", "null")
	writeKV(&sb, "age_range", strQ("o18"))
	writeKV(&sb, "did_use_age", "false")
	writeKV(&sb, "os_shared_age_range", "null")
	writeKV(&sb, "gender", fmt.Sprintf("%d", profile.Gender))
	writeKV(&sb, "use_custom_gender", "false")
	writeKV(&sb, "custom_gender", "null")
	writeKV(&sb, "encrypted_password", strQ(escJSON(encPassword)))
	writeKV(&sb, "username", "null")
	writeKV(&sb, "username_prefill", "null")
	writeKV(&sb, "accounts_list_client", "[]")
	writeKV(&sb, "fb_conf_source", "null")
	// Device / family
	writeKV(&sb, "device_id", strQ(deviceID))
	writeKV(&sb, "ig4a_qe_device_id", "null")
	writeKV(&sb, "family_device_id", strQ(deviceID)) // V22: = deviceid
	writeKV(&sb, "fdid_available_on_start", "null")
	writeKV(&sb, "fdid_rid_available_on_start", "null")
	writeKV(&sb, "user_id", "null")
	writeKV(&sb, "safetynet_token", strQ(escJSON(safetynetTok)))
	writeKV(&sb, "skip_slow_rel_check", "false")
	writeKV(&sb, "safetynet_response", "null")
	writeKV(&sb, "machine_id", strQ(machineID))
	// Profile photo
	writeKV(&sb, "profile_photo", "null")
	writeKV(&sb, "profile_photo_id", "null")
	writeKV(&sb, "profile_photo_upload_id", "null")
	writeKV(&sb, "avatar", "null")
	// Email oauth
	writeKV(&sb, "email_oauth_token_no_contact_perm", "null")
	writeKV(&sb, "email_oauth_token", "null")
	writeKV(&sb, "email_oauth_tokens", "[]") // V22: array, not {}
	writeKV(&sb, "sign_in_with_google_email", "null")
	writeKV(&sb, "should_skip_two_step_conf", "null")
	writeKV(&sb, "openid_tokens_for_testing", "null")
	writeKV(&sb, "encrypted_msisdn", "null")
	writeKV(&sb, "encrypted_msisdn_for_safetynet", "null")
	// Safetynet headers
	writeKV(&sb, "cached_headers_safetynet_info", "null")
	writeKV(&sb, "should_skip_headers_safetynet", "null")
	writeKV(&sb, "headers_last_infra_flow_id", "null")
	writeKV(&sb, "headers_last_infra_flow_id_safetynet", strQ(flowID)) // V22: has guid
	writeKV(&sb, "headers_flow_id", "null")                            // V22: null
	writeKV(&sb, "was_headers_prefill_available", "null")
	// Sync info
	writeKV(&sb, "sso_enabled", "null")
	writeKV(&sb, "existing_accounts", "null")
	writeKV(&sb, "used_ig_birthday", "null")
	writeKV(&sb, "create_new_to_app_account", "null")
	writeKV(&sb, "skip_session_info", "null")
	writeKV(&sb, "ck_error", "null")
	writeKV(&sb, "ck_id", "null")
	writeKV(&sb, "ck_nonce", "null")
	writeKV(&sb, "should_save_password", "false") // V22: false!
	writeKV(&sb, "fb_access_token", "null")
	writeKV(&sb, "is_msplit_reg", "null")
	writeKV(&sb, "is_spectra_reg", "null")
	writeKV(&sb, "dema_account_consent_given", "null")
	writeKV(&sb, "spectra_reg_token", "null")
	writeKV(&sb, "spectra_reg_guardian_id", "null")
	writeKV(&sb, "spectra_reg_guardian_logged_in_context", "null")
	writeKV(&sb, "user_id_of_msplit_creator", "null")
	writeKV(&sb, "msplit_creator_nonce", "null")
	writeKV(&sb, "dma_data_combination_consent_given", "null")
	writeKV(&sb, "xapp_accounts", "null")
	writeKV(&sb, "fb_device_id", "null")
	writeKV(&sb, "fb_machine_id", "null")
	writeKV(&sb, "ig_device_id", "null")
	writeKV(&sb, "ig_machine_id", "null")
	writeKV(&sb, "should_skip_nta_upsell", "null")
	writeKV(&sb, "caa_reg_flow_source", strQ("login_home_native_integration_point"))
	writeKV(&sb, "ig_authorization_token", "null")
	writeKV(&sb, "full_sheet_flow", "false")
	writeKV(&sb, "crypted_user_id", "null")
	writeKV(&sb, "is_caa_perf_enabled", "true")
	writeKV(&sb, "is_preform", "true")
	writeKV(&sb, "should_show_rel_error", "false")
	writeKV(&sb, "ignore_suma_check", "false")
	writeKV(&sb, "dismissed_login_upsell_with_cna", "false")
	writeKV(&sb, "ignore_existing_login", "false")
	writeKV(&sb, "ignore_existing_login_from_suma", "false")
	writeKV(&sb, "ignore_existing_login_after_errors", "false")
	writeKV(&sb, "suggested_first_name", "null")
	writeKV(&sb, "suggested_last_name", "null")
	writeKV(&sb, "suggested_full_name", "null")
	writeKV(&sb, "frl_authorization_token", "null")
	writeKV(&sb, "post_form_errors", "null")
	writeKV(&sb, "skip_step_without_errors", "false")
	writeKV(&sb, "existing_account_exact_match_checked", "true")
	writeKV(&sb, "existing_account_fuzzy_match_checked", "false")
	writeKV(&sb, "email_oauth_exists", "false")
	writeKV(&sb, "confirmation_code_send_error", "null")
	writeKV(&sb, "is_too_young", "false")
	writeKV(&sb, "source_account_type", "null")
	writeKV(&sb, "whatsapp_installed_on_client", "false")
	writeKV(&sb, "confirmation_medium", "null")
	writeKV(&sb, "source_credentials_type", "null")
	writeKV(&sb, "source_cuid", "null")
	writeKV(&sb, "source_account_reg_info", "null")
	writeKV(&sb, "soap_creation_source", "null")
	writeKV(&sb, "source_account_type_to_reg_info", "null")
	writeKV(&sb, "registration_flow_id", strQ(regFlowID))
	writeKV(&sb, "should_skip_youth_tos", "false")
	writeKV(&sb, "is_youth_regulation_flow_complete", "false")
	writeKV(&sb, "is_on_cold_start", "false")
	writeKV(&sb, "email_prefilled", "false")
	writeKV(&sb, "cp_confirmed_by_auto_conf", "false")
	writeKV(&sb, "in_sowa_experiment", "false")
	writeKV(&sb, "youth_regulation_config", "null")
	writeKV(&sb, "conf_allow_back_nav_after_change_cp", "null")
	writeKV(&sb, "conf_bouncing_cliff_screen_type", "null")
	writeKV(&sb, "conf_show_bouncing_cliff", "null")
	writeKV(&sb, "eligible_to_flash_call_in_ig4a", "false")
	writeKV(&sb, "eligible_to_mo_sms_in_ig4a", "false")
	writeKV(&sb, "mo_sms_ent_id", "null")
	writeKV(&sb, "flash_call_permissions_status", `{"READ_PHONE_STATE":"DENIED","READ_CALL_LOG":"DENIED","ANSWER_PHONE_CALLS":"DENIED"}`)
	writeKV(&sb, "gms_incoming_call_retriever_eligibility", strQ("eligible"))
	writeKV(&sb, "attestation_result", fmt.Sprintf(
		`{"keyHash":"%s","data":"%s","signature":"%s"}`,
		keyHash, attData, escJSON(attSig),
	))
	writeKV(&sb, "request_data_and_challenge_nonce_string", "null")
	writeKV(&sb, "confirmed_cp_and_code", "null")
	writeKV(&sb, "notification_callback_id", "null")
	writeKV(&sb, "reg_suma_state", "0")
	writeKV(&sb, "is_msplit_neutral_choice", "false")
	writeKV(&sb, "msg_previous_cp", "null")
	writeKV(&sb, "ntp_import_source_info", "null")
	writeKV(&sb, "youth_consent_decision_time", "null")
	writeKV(&sb, "sk_pipa_consent_given", "null")
	writeKV(&sb, "should_show_spi_before_conf", "true")
	writeKV(&sb, "google_oauth_account", "null")
	writeKV(&sb, "is_reg_request_from_ig_suma", "false")
	writeKV(&sb, "is_toa_reg", "false")
	writeKV(&sb, "is_threads_public", "false")
	writeKV(&sb, "spc_import_flow", "false")
	writeKV(&sb, "caa_play_integrity_attestation_result", strQ("")) // V22: empty string
	writeKV(&sb, "client_known_key_hash", "null")
	writeKV(&sb, "flash_call_provider", "null")
	writeKV(&sb, "is_in_gms_experience", "null")
	writeKV(&sb, "flash_call_nonce_prefix_details", "null")
	writeKV(&sb, "spc_birthday_input", "false")
	writeKV(&sb, "failed_birthday_year_count", "null")
	writeKV(&sb, "user_presented_medium_source", "null")
	writeKV(&sb, "user_opted_out_of_ntp", "null")
	writeKV(&sb, "is_from_registration_reminder", "false")
	writeKV(&sb, "show_youth_reg_in_ig_spc", "false")
	writeKV(&sb, "fb_suma_is_high_confidence", "null")
	writeKV(&sb, "screen_visited", screenVisited)
	writeKV(&sb, "fb_email_login_upsell_skip_suma_post_tos", "false")
	writeKV(&sb, "fb_suma_is_from_email_login_upsell", "false")
	writeKV(&sb, "fb_suma_is_from_phone_login_upsell", "false")
	writeKV(&sb, "should_prefill_cp_in_ar", "null")
	writeKV(&sb, "ig_partially_created_account_user_id", "null")
	writeKV(&sb, "ig_partially_created_account_nonce", "null")
	writeKV(&sb, "ig_partially_created_account_nonce_expiry", "null")
	writeKV(&sb, "force_sessionless_nux_experience", "false")
	writeKV(&sb, "has_seen_suma_landing_page_pre_conf", "false")
	writeKV(&sb, "has_seen_suma_candidate_page_pre_conf", "false")
	writeKV(&sb, "has_seen_confirmation_screen", "false")
	writeKV(&sb, "suma_on_conf_threshold", "-1")
	writeKV(&sb, "should_show_error_msg", "true")
	writeKV(&sb, "th_profile_photo_token", "null")
	writeKV(&sb, "attempted_silent_auth_in_fb", "false")
	writeKV(&sb, "attempted_silent_auth_in_ig", "false")
	writeKV(&sb, "sa_prefetch_callback_id", "null")
	writeKV(&sb, "cp_suma_results_map", cpSumaResultsMap)
	writeKV(&sb, "source_username", "null")
	writeKV(&sb, "next_uri", "null")
	writeKV(&sb, "should_use_next_uri", "null")
	writeKV(&sb, "linking_entry_point", "null")
	writeKV(&sb, "fb_encrypted_partial_new_account_properties", "null")
	writeKV(&sb, "starter_pack_name", "null")
	writeKV(&sb, "starter_pack_creator_user_ids", "null")
	writeKV(&sb, "wa_data_bundle", "null")
	writeKV(&sb, "bloks_controller_source", "null")
	writeKV(&sb, "airwave_registration_code", "null")
	writeKV(&sb, "is_sessionless_nux", "null")
	writeKV(&sb, "login_contactpoint", "null")
	writeKV(&sb, "login_contactpoint_type", "null")
	writeKV(&sb, "is_nta_shortened", "false")
	writeKV(&sb, "should_show_bday_after_name_suggestions", "null")
	writeKV(&sb, "should_override_back_nav", "false")
	writeKV(&sb, "ig_footer_variant", strQ("control"))
	writeKV(&sb, "device_network_info", deviceNetworkInfo)
	writeKV(&sb, "is_from_web_lite_reg_controller", "null")
	writeKV(&sb, "login_form_siwg_email", "null")
	writeKV(&sb, "account_setup_waterfall_id", "null")
	writeKV(&sb, "is_wanted_suma_user", "false")
	writeKV(&sb, "device_zero_balance_state", strQ("init"))
	writeKV(&sb, "should_delay_wa_disclosure", "false")
	writeKV(&sb, "is_in_nta_single_form", "false")
	writeKV(&sb, "source_account_image_asset_id", "null")
	writeKV(&sb, "passkey_eligible_device", "null")
	writeKV(&sb, "nta_single_form_variant", "null")
	writeKV(&sb, "enable_survey", "null")
	writeLastKV(&sb, "phone_prefetch_outcome", "null")
	sb.WriteByte('}')
	regInfoJSON := sb.String()

	// ── server_params JSON (L3) ──────────────────────────────────────────────
	serverParams := `"server_params":{` +
		`"event_request_id":"` + eventReqID + `"` +
		`,"is_from_logged_out":0` +
		`,"layered_homepage_experiment_group":null` +
		`,"device_id":"` + deviceID + `"` +
		`,"reg_context":null` +
		`,"login_surface":"login_home"` +
		`,"waterfall_id":"` + profile.WaterfallID + `"` +
		`,"machine_id":"` + machineID + `"` +
		`,"INTERNAL__latency_qpl_instance_id":` + fmt.Sprintf("%d", qplID) +
		`,"flow_info":"` + escJSON(flowInfoContent) + `"` +
		`,"is_platform_login":0` +
		`,"login_entry_point":"logged_out"` +
		`,"INTERNAL__latency_qpl_marker_id":36707139` +
		`,"bloks_controller_source":"bk_caa_reg_icon_text_list_tos_screen"` +
		`,"reg_info":"` + escJSON(regInfoJSON) + `"` +
		`,"family_device_id":"` + deviceID + `"` +
		`,"offline_experiment_group":"caa_iteration_v6_perf_fb_2"` +
		`,"access_flow_version":"pre_mt_behavior"` +
		`,"is_from_logged_in_switcher":0` +
		`,"current_step":8` +
		`}`

	// ── client_input_params JSON (L3) ────────────────────────────────────────
	clientInputParams := `"client_input_params":{` +
		`"ck_error":""` +
		`,"aac":"` + escJSON(aacJSONContent) + `"` +
		`,"device_id":"` + deviceID + `"` +
		`,"waterfall_id":"` + profile.WaterfallID + `"` +
		`,"zero_balance_state":"init"` +
		`,"network_bssid":null` +
		`,"failed_birthday_year_count":""` +
		`,"headers_last_infra_flow_id":"` + flowID + `"` +
		`,"ig_partially_created_account_nonce_expiry":0` +
		`,"machine_id":"` + machineID + `"` +
		`,"reached_from_tos_screen":1` +
		`,"ig_partially_created_account_nonce":""` +
		`,"block_store_machine_id":null` +
		`,"ck_nonce":""` +
		`,"lois_settings":{"lois_token":""}` +
		`,"ig_partially_created_account_user_id":0` +
		`,"cloud_trust_token":null` +
		`,"ck_id":""` +
		`,"no_contact_perm_email_oauth_token":""` +
		`,"encrypted_msisdn":""` +
		`}`

	// ── L3 wrap ──────────────────────────────────────────────────────────────
	l3 := `{` + clientInputParams + `,` + serverParams + `}`
	// L2 = {"params": "<esc(L3)>"}
	l2 := `{"params":"` + escJSON(l3) + `"}`
	// L1 params string = esc(L2)
	l1ParamsStr := escJSON(l2)

	// ── nt_context ───────────────────────────────────────────────────────────
	ntContext := `{"using_white_navbar":true` +
		`,"styles_id":"` + stylesId + `"` +
		`,"pixel_ratio":3` +
		`,"is_push_on":true` +
		`,"debug_tooling_metadata_token":null` +
		`,"is_flipper_enabled":false` +
		`,"theme_params":[{"value":[],"design_system_name":"FDS"}]` +
		`,"bloks_version":"` + bloksVer + `"}`

	// ── L0 outer JSON — URL-encode toàn bộ một lần (C# WebUtility.UrlEncode) ─
	l0 := `{"params":{"params":"` + l1ParamsStr + `"` +
		`,"bloks_versioning_id":"` + bloksVer + `"` +
		`,"app_id":"` + appId + `"}` +
		`,"scale":"3"` +
		`,"nt_context":` + ntContext + `}`

	return url.QueryEscape(l0)
}

// writeKV — append "key":value, với dấu `,` prefix từ field thứ 2.
// Dùng internal flag trong builder (tự track bằng len).
func writeKV(sb *strings.Builder, key, rawValue string) {
	if sb.Len() > 1 { // 1 = "{" đã ghi
		sb.WriteByte(',')
	}
	sb.WriteByte('"')
	sb.WriteString(key)
	sb.WriteString(`":`)
	sb.WriteString(rawValue)
}

// writeLastKV giống writeKV — semantic alias cho field cuối (để đọc code rõ).
func writeLastKV(sb *strings.Builder, key, rawValue string) {
	writeKV(sb, key, rawValue)
}

// strQ wrap `"s"` — dùng cho value type string trong JSON.
func strQ(s string) string { return `"` + s + `"` }

// buildDeviceNetworkInfo sinh object device_network_info dựa trên SIM profile.
// Port C# V2.cs L774.
func buildDeviceNetworkInfo(profile fakeinfo.FullRegProfile) string {
	simOp := profile.Sim.HNI
	if simOp == "" {
		simOp = "45204"
	}
	// Sim rỗng → random toàn pool carriers (KHÔNG dính Viettel).
	carrierName := profile.Sim.OperatorName
	if carrierName == "" {
		carrierName = fakeinfo.RandomCarrier()
		if carrierName == "" {
			carrierName = "T-Mobile"
		}
	}
	// Carrier ID/name cho phần default_subscription_info — C# hardcode 1899/Viettel,
	// nhưng ở đây map theo carrier thực để nhất quán với UA.
	carrierID := 1899

	return `{"default_subscription_info":{` +
		`"network_type":13` +
		`,"is_data_roaming":false` +
		`,"is_esim":null` +
		`,"is_gsm_roaming":false` +
		`,"is_sim_sms_capable":null` +
		`,"is_mobile_data_enabled":false` +
		`,"sim_carrier_id":` + fmt.Sprintf("%d", carrierID) +
		`,"sim_carrier_id_name":"` + escJSON(strings.ReplaceAll(carrierName, " ", "+")) + `"` +
		`,"sim_state":5` +
		`,"sim_operator":` + simOp +
		`,"sim_operator_name":"` + escJSON(carrierName) + `"` +
		`,"signal_strength":2` +
		`,"group_id_level_1":null` +
		`,"network_operator":` + simOp +
		`}` +
		`,"sim_count":2` +
		`,"is_wifi":true` +
		`,"is_airplane_mode":false` +
		`,"is_active_network_cellular":false` +
		`,"is_device_sms_capable":true` +
		`,"active_subscriptions_info":null}`
}

// createLatencyQplInstanceIDV2 port C# V2.cs L456-462.
// Random 64-bit in range [100000000000000, 899999999999999].
func createLatencyQplInstanceIDV2() int64 {
	buf := make([]byte, 8)
	rand.Read(buf)
	val := int64(buf[0])<<56 | int64(buf[1])<<48 | int64(buf[2])<<40 | int64(buf[3])<<32 |
		int64(buf[4])<<24 | int64(buf[5])<<16 | int64(buf[6])<<8 | int64(buf[7])
	if val < 0 {
		val = -val
	}
	return 100000000000000 + (val % 800000000000000)
}

// generateKeyHashV2 port C# V2.cs L409-417: SHA256(random 32 bytes) → hex lowercase.
func generateKeyHashV2() string {
	b := make([]byte, 32)
	rand.Read(b)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// generateAttestationDataV2 port C# V2.cs L419-426.
// base64({"challenge_nonce":"<base64(24 rand bytes)>","username":"<contactpoint>"})
func generateAttestationDataV2(contactpoint string) string {
	nonce := make([]byte, 24)
	rand.Read(nonce)
	json := fmt.Sprintf(`{"challenge_nonce":"%s","username":"%s"}`,
		base64.StdEncoding.EncodeToString(nonce),
		escJSON(contactpoint),
	)
	return base64.StdEncoding.EncodeToString([]byte(json))
}

// generateAttestationSignatureV2 port C# V2.cs L428-442.
// DER ECDSA signature 70 bytes: 0x30 0x44, 0x02 0x20, r(32), 0x02 0x20, s(32) — all base64.
func generateAttestationSignatureV2() string {
	r := make([]byte, 32)
	s := make([]byte, 32)
	rand.Read(r)
	rand.Read(s)
	r[0] &= 0x7F
	s[0] &= 0x7F
	der := make([]byte, 70)
	der[0] = 0x30
	der[1] = 0x44
	der[2] = 0x02
	der[3] = 0x20
	copy(der[4:36], r)
	der[36] = 0x02
	der[37] = 0x20
	copy(der[38:70], s)
	return base64.StdEncoding.EncodeToString(der)
}

// generateSafetyNetTokenV2 port C# V2.cs L444-454.
// base64("unknown|{ts}|" + random 32 bytes)
func generateSafetyNetTokenV2(ts int64) string {
	rnd := make([]byte, 32)
	rand.Read(rnd)
	prefix := []byte(fmt.Sprintf("unknown|%d|", ts))
	combined := append(prefix, rnd...)
	return base64.StdEncoding.EncodeToString(combined)
}

// randomAlphanumeric sinh chuỗi a-zA-Z0-9 độ dài n.
func randomAlphanumeric(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}

// randomBase64URLRaw sinh base64-url-raw (không padding) của n random bytes.
func randomBase64URLRaw(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
