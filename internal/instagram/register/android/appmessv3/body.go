// body.go — Messenger (Orca v530) reg.create.account.async body builder.
// Format Messenger: variables.params.params = "{params:{...},}" (FB custom), nt_context
// tối giản, #PWD_MSGR password, doc_id 1199408042628.., bloks dadbbd68.., Orca UA.
// Single-shot: reg_context=null + full reg_info → FB tạo account 1 call (như s565 FB4A).
package appmessv3

import (
	crand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"HVRIns/internal/instagram/fakeinfo"
)

// ─── Messenger (Orca) v530 app constants (capture V4) ─────────────────────────

var appmv3AppVersions = []struct {
	Version string
	Build   string
}{
	{"530.1.0.67.107", "814020040"}, // Orca v530 (capture V4)
}

const (
	appmv3DocID        = "11994080426288799937543572098"
	appmv3RenderDocID  = "10537346114757978319483558625" // AppRootQuery render (welcome/confirmation screens)
	appmv3BloksVer     = "dadbbd68d34735f7a39b791542ad0ecd1b257eddf3e70ab790d47b3cedd8b093"
	appmv3FriendlyName = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.create.account.async"
	appmv3CreateAppID  = "com.bloks.www.bloks.caa.reg.create.account.async"
	// App-token Messenger (Orca) — public client token.
	appmv3AppToken = "256002347743983|374e60f8b9bb6b8cbb30f78030438895"
	appmv3Product  = "256002347743983"

	// OriginalUA — UA Orca cố định cho platform appmv3reg.
	OriginalUA = "Dalvik/2.1.0 (Linux; U; Android 15; SM-G996B Build/AP3A.240905.015.A2) [FBAN/Orca-Android;FBAV/530.1.0.67.107;FBPN/com.facebook.orca;FBLC/en_GB;FBBV/814020040;FBCR/null;FBMF/samsung;FBBD/samsung;FBDV/SM-G996B;FBSV/15;FBCA/arm64-v8a:null;FBDM/{density=2.8125,width=1080,height=2400};FB_FW/1;]"
)

// S23Device — device variant (giữ cho profile.go tương thích, dùng device Orca).
type S23Device struct {
	Model   string
	Name    string
	Width   int
	Height  int
	Density string
	FBSS    string
}

var s23Devices = []S23Device{
	{Model: "SM-G996B", Name: "Galaxy S21+", Width: 1080, Height: 2400, Density: "2.8125", FBSS: "3"},
	{Model: "SM-S911B", Name: "Galaxy S23", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S918B", Name: "Galaxy S23 Ultra", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
}

var s24Devices = []S23Device{
	{Model: "SM-G996B", Name: "Galaxy S21+", Width: 1080, Height: 2400, Density: "2.8125", FBSS: "3"},
}

// DevicePoolForPlatform trả về pool device theo platform.
func DevicePoolForPlatform(platform string) []S23Device {
	switch platform {
	case "s24":
		return s24Devices
	default:
		return s23Devices
	}
}

// orcaUA dựng Messenger Orca v530 UA — DEVICE ĐA DẠNG mỗi account (pool FB4A:
// nhiều model/brand/OS/density/screen) để mỗi luồng test 1 UA KHÁC nhau, tránh
// FB gom. CHỈ giữ cố định phần app Messenger (FBAV/530, FBBV/814020040, com.facebook.orca)
// vì gắn với doc_id/bloks của API.
func orcaUA(locale string, profile AppMV3Profile) string {
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
	// Version cố định (535/545) → FBAV/FBBV khớp doc_id. Bản 530 (rỗng) → random từ pool.
	fbav, fbbv := profile.FBAV, profile.FBBV
	if fbav == "" || fbbv == "" {
		fbav, fbbv = fakeinfo.RandomMessOrcaAppVersion("reg")
	}
	return fmt.Sprintf(
		"Dalvik/2.1.0 (Linux; U; Android %s; %s Build/%s) "+
			"[FBAN/Orca-Android;FBAV/%s;FBPN/com.facebook.orca;FBLC/%s;"+
			"FBBV/%s;FBCR/null;FBMF/%s;FBBD/%s;FBDV/%s;FBSV/%s;"+
			"FBCA/%s:null;FBDM/{density=%s,width=%d,height=%d};FB_FW/1;]",
		osv, d.Model, buildID, fbav, locale, fbbv, manuf, brand, d.Model, osv, arch, dens, w, h,
	)
}

func mustJSON(v interface{}) string { b, _ := json.Marshal(v); return string(b) }

// genSafetynetToken — placeholder integrity token format base64("unknown|<unix>|<rand>")
// (bám capture: Play Integrity không có key → fallback "unknown|ts|bytes"). Tín hiệu anti-abuse.
func genSafetynetToken() string {
	raw := make([]byte, 20)
	if _, err := crand.Read(raw); err != nil {
		for i := range raw {
			raw[i] = byte(rand.Intn(256))
		}
	}
	inner := fmt.Sprintf("unknown|%d|%s", time.Now().Unix(), string(raw))
	return base64.StdEncoding.EncodeToString([]byte(inner))
}

func genMachineID() string {
	b := make([]byte, 18)
	if _, err := crand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(rand.Intn(256))
		}
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func genAAC() string {
	b := make([]byte, 32)
	if _, err := crand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(rand.Intn(256))
		}
	}
	return fmt.Sprintf(`{"aac_init_timestamp":%d,"aacjid":"%s","aaccs":"%s"}`,
		time.Now().Unix(), uuid.New().String(), base64.RawURLEncoding.EncodeToString(b))
}

// buildStepBody — generic Bloks action body cho 1 bước của luồng multi-step reg.
// Mọi bước dùng chung client_doc_id + bloks_versioning_id, chỉ khác app_id + params.
func buildStepBody(appID, friendlyPrefix, docID, bloksVer, locale string, serverParams, clientInput map[string]interface{}) string {
	if locale == "" {
		locale = "en_GB"
	}
	if friendlyPrefix == "" {
		friendlyPrefix = "FbBloksActionRootQuery-"
	}
	if docID == "" {
		docID = appmv3DocID
	}
	if bloksVer == "" {
		bloksVer = appmv3BloksVer
	}
	inner := map[string]interface{}{"client_input_params": clientInput, "server_params": serverParams}
	paramsStr := "{params:" + mustJSON(inner) + ",}"
	variables := map[string]interface{}{
		"params": map[string]interface{}{
			"params": paramsStr, "bloks_versioning_id": bloksVer, "app_id": appID,
		},
		"scale": "3",
		"nt_context": map[string]interface{}{
			"is_flipper_enabled":           false,
			"theme_params":                 []interface{}{map[string]interface{}{"value": []string{}, "design_system_name": "FDS"}},
			"debug_tooling_metadata_token": nil,
		},
	}
	form := url.Values{}
	form.Set("method", "post")
	form.Set("pretty", "false")
	form.Set("format", "json")
	form.Set("server_timestamps", "true")
	form.Set("locale", locale)
	form.Set("fb_api_req_friendly_name", friendlyPrefix+appID)
	form.Set("fb_api_caller_class", "graphservice")
	form.Set("client_doc_id", docID)
	form.Set("fb_api_client_context", `{"is_background":false}`)
	form.Set("variables", mustJSON(variables))
	form.Set("fb_api_analytics_tags", `["GraphServices"]`)
	form.Set("client_trace_id", uuid.New().String())
	return form.Encode()
}

// buildRegisterBody sinh form body create.account.async format Messenger.
func buildRegisterBody(profile AppMV3Profile, encPassword, contactpoint, contactpointType, locale string) string {
	if locale == "" {
		locale = "en_GB"
	}
	deviceID := profile.DeviceID
	familyDeviceID := profile.FamilyDeviceID
	// machine_id = datr pool đã warm (như s565 FB4A) → tín hiệu chống abuse.
	// Rỗng (không có datr) mới fallback random.
	machineID := profile.MachineID
	if machineID == "" {
		machineID = genMachineID()
	}
	waterfallID := uuid.New().String()
	lat := int64(66000000000000 + rand.Int63n(900000000000))

	regInfo := buildCreateRegInfo(profile, encPassword, contactpoint, contactpointType, deviceID, familyDeviceID, machineID)

	bloksVer := profile.BloksVer
	if bloksVer == "" {
		bloksVer = appmv3BloksVer
	}
	docID := profile.DocID
	if docID == "" {
		docID = appmv3DocID
	}

	clientInputParams := map[string]interface{}{
		"aac": genAAC(), "device_id": deviceID, "waterfall_id": waterfallID,
		"zero_balance_state": "", "network_bssid": nil, "failed_birthday_year_count": nil,
		"headers_last_infra_flow_id": "", "machine_id": machineID, "reached_from_tos_screen": 1,
		"block_store_machine_id": "", "lois_settings": map[string]interface{}{"lois_token": ""},
		"cloud_trust_token": nil, "ck_error": nil, "ck_nonce": nil, "ck_id": nil,
		"no_contact_perm_email_oauth_token": nil, "encrypted_msisdn": "",
	}
	serverParams := map[string]interface{}{
		"event_request_id": uuid.New().String(), "is_from_logged_out": 0,
		"layered_homepage_experiment_group": nil, "device_id": deviceID,
		"reg_context":   nil, // single-shot: không có context từ multi-step
		"login_surface": "unknown", "waterfall_id": waterfallID,
		"INTERNAL__latency_qpl_instance_id": lat,
		"flow_info":                         `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`,
		"is_platform_login":                 0, "INTERNAL__latency_qpl_marker_id": 36707139,
		"bloks_controller_source": "bk_caa_reg_icon_text_list_tos_screen",
		"reg_info":                regInfo, "family_device_id": familyDeviceID,
		"offline_experiment_group": nil, "access_flow_version": "pre_mt_behavior",
		"is_from_logged_in_switcher": 0, "current_step": 8,
	}

	inner := map[string]interface{}{"client_input_params": clientInputParams, "server_params": serverParams}
	paramsStr := "{params:" + mustJSON(inner) + ",}"
	variables := map[string]interface{}{
		"params": map[string]interface{}{
			"params": paramsStr, "bloks_versioning_id": bloksVer, "app_id": appmv3CreateAppID,
		},
		"scale": "3",
		"nt_context": map[string]interface{}{
			"is_flipper_enabled":           false,
			"theme_params":                 []interface{}{map[string]interface{}{"value": []string{}, "design_system_name": "FDS"}},
			"debug_tooling_metadata_token": nil,
		},
	}

	form := url.Values{}
	form.Set("method", "post")
	form.Set("pretty", "false")
	form.Set("format", "json")
	form.Set("server_timestamps", "true")
	form.Set("locale", locale)
	form.Set("fb_api_req_friendly_name", appmv3FriendlyName)
	form.Set("fb_api_caller_class", "graphservice")
	form.Set("client_doc_id", docID)
	form.Set("fb_api_client_context", `{"is_background":false}`)
	form.Set("variables", mustJSON(variables))
	form.Set("fb_api_analytics_tags", `["GraphServices"]`)
	form.Set("client_trace_id", uuid.New().String())
	return form.Encode()
}

// buildCreateRegInfo — reg_info CAA cho create.account (account MỚI: birthday/password set, user_id null).
func buildCreateRegInfo(profile AppMV3Profile, encPassword, contactpoint, contactpointType, deviceID, familyDevID, machineID string) string {
	fullName := strings.TrimSpace(profile.FirstName + " " + profile.LastName)
	ri := map[string]interface{}{
		"first_name": profile.FirstName, "last_name": profile.LastName, "full_name": fullName,
		"contactpoint": contactpoint, "ar_contactpoint": nil, "contactpoint_type": contactpointType,
		"is_using_unified_cp": nil, "unified_cp_screen_variant": nil,
		"is_cp_auto_confirmed": false, "is_cp_auto_confirmable": false, "is_cp_claimed": false,
		"confirmation_code": nil, "birthday": profile.Birthday, "birthday_derived_from_age": nil,
		"age_range": "o18", "did_use_age": nil, "os_shared_age_range": nil,
		"gender": profile.Gender, "use_custom_gender": false, "custom_gender": nil,
		"encrypted_password": encPassword, "username": nil, "username_prefill": nil,
		"accounts_list_client": nil, "fb_conf_source": nil,
		"device_id": deviceID, "ig4a_qe_device_id": nil, "family_device_id": familyDevID,
		"fdid_available_on_start": nil, "fdid_rid_available_on_start": nil, "asdid_available_on_start": nil,
		"user_id": nil, "safetynet_token": genSafetynetToken(), "skip_slow_rel_check": false, "safetynet_response": nil,
		"machine_id": machineID, "profile_photo": nil, "profile_photo_id": nil, "profile_photo_upload_id": nil,
		"avatar": nil, "email_oauth_token_no_contact_perm": nil, "email_oauth_token": nil, "email_oauth_tokens": nil,
		"sign_in_with_google_email": nil, "should_skip_two_step_conf": nil, "openid_tokens_for_testing": nil,
		"encrypted_msisdn": nil, "encrypted_msisdn_for_safetynet": nil, "cached_headers_safetynet_info": nil,
		"should_skip_headers_safetynet": nil, "headers_last_infra_flow_id": nil, "headers_last_infra_flow_id_safetynet": nil,
		"headers_flow_id": nil, "was_headers_prefill_available": nil, "sso_enabled": nil, "existing_accounts": nil,
		"used_ig_birthday": nil, "create_new_to_app_account": nil, "skip_session_info": nil,
		"ck_error": nil, "ck_id": nil, "ck_nonce": nil, "should_save_password": true, "fb_access_token": nil,
		"is_msplit_reg": nil, "is_spectra_reg": nil, "dema_account_consent_given": nil, "spectra_entry_source": nil,
		"spectra_reg_token": nil, "spectra_reg_guardian_id": nil, "spectra_reg_guardian_logged_in_context": nil,
		"spectra_requester_user_id": nil, "user_id_of_msplit_creator": nil, "msplit_creator_nonce": nil,
		"dma_data_combination_consent_given": nil, "xapp_accounts": nil, "fb_device_id": nil, "fb_machine_id": nil,
		"ig_device_id": nil, "ig_machine_id": nil, "should_skip_nta_upsell": nil, "big_blue_token": nil,
		"caa_reg_flow_source": "aymh_multi_profiles_native_integration_point", "ig_authorization_token": nil,
		"full_sheet_flow": false, "crypted_user_id": nil,
		"is_ca_late_teen": nil, "is_early_teen": nil, "is_caa_perf_enabled": true, "is_preform": true,
		"should_show_rel_error": false, "ignore_suma_check": false, "dismissed_login_upsell_with_cna": false,
		"ignore_existing_login": false, "ignore_existing_login_from_suma": false, "ignore_existing_login_after_errors": false,
		"suggested_first_name": nil, "suggested_last_name": nil, "suggested_full_name": nil, "frl_authorization_token": nil,
		"post_form_errors": nil, "skip_step_without_errors": false, "existing_account_exact_match_checked": true,
		"existing_account_fuzzy_match_checked": false, "email_oauth_exists": false, "confirmation_code_send_error": nil,
		"is_too_young": false, "source_account_type": nil, "whatsapp_installed_on_client": false,
		"confirmation_medium": nil, "source_credentials_type": nil, "source_cuid": nil, "source_account_reg_info": nil,
		"soap_creation_source": nil, "source_account_type_to_reg_info": nil, "registration_flow_id": uuid.New().String(),
		"should_skip_youth_tos": true, "is_youth_regulation_flow_complete": false, "is_on_cold_start": false,
		"email_prefilled": false, "cp_confirmed_by_auto_conf": false, "in_sowa_experiment": false,
		"youth_regulation_config":             map[string]interface{}{"isEnabled": true, "consentJurisdiction": "VN", "shouldRaiseAgeGating": false, "ageOfConsent": nil, "ageOfParentalConsent": nil, "requiresAgeVerification": false, "requiresParentalConsent": false, "ageThresholdForRegBlocking": nil},
		"conf_allow_back_nav_after_change_cp": true, "conf_bouncing_cliff_screen_type": nil,
		"conf_show_bouncing_cliff": nil, "eligible_to_flash_call_in_ig4a": false, "eligible_to_mo_sms_in_ig4a": false,
		"mo_sms_ent_id": nil, "flash_call_permissions_status": nil, "gms_incoming_call_retriever_eligibility": nil,
		"attestation_result": map[string]interface{}{"errorMessage": "KeyAttestationException: No key found!"}, "request_data_and_challenge_nonce_string": nil, "confirmed_cp_and_code": nil,
		"notification_callback_id": nil, "reg_suma_state": 0, "is_msplit_neutral_choice": false, "msg_previous_cp": nil,
		"ntp_import_source_info": nil, "youth_consent_decision_time": nil, "sk_pipa_consent_given": nil,
		"should_show_spi_before_conf": true, "google_oauth_account": nil, "is_reg_request_from_ig_suma": false,
		"is_toa_reg": false, "is_threads_public": false, "spc_import_flow": false, "caa_play_integrity_attestation_result": nil,
		"client_known_key_hash": nil, "flash_call_provider": nil, "is_in_gms_experience": nil,
		"flash_call_nonce_prefix_details": nil, "spc_birthday_input": false, "failed_birthday_year_count": nil,
		"user_presented_medium_source": nil, "user_opted_out_of_ntp": nil, "is_from_registration_reminder": false,
		"show_youth_reg_in_ig_spc": false, "fb_suma_is_high_confidence": nil,
		"screen_visited": []interface{}{"CAA_REG_WELCOME_SCREEN", "bloks.caa.reg.name", "bloks.caa.reg.birthday", "CAA_REG_CONTACT_POINT", "CAA_REG_PASSWORD"},
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
		"bloks_controller_source": "bk_caa_reg_icon_text_list_tos_screen", "airwave_registration_code": nil,
		"is_sessionless_nux": nil, "login_contactpoint": nil, "login_contactpoint_type": nil,
		"should_show_bday_after_name_suggestions": nil, "should_override_back_nav": false, "ig_footer_variant": "control",
		"device_network_info": nil, "is_from_web_lite_reg_controller": nil, "login_form_siwg_email": nil,
		"account_setup_waterfall_id": nil, "is_wanted_suma_user": nil, "device_zero_balance_state": nil,
		"wa_to_ig_merged_tos_variant": nil, "is_in_nta_single_form": false, "source_account_image_asset_id": nil,
		"passkey_eligible_device": nil, "nta_control_reason": nil, "nta_risk_type": nil, "nta_single_form_variant": nil,
		"enable_survey": nil, "phone_prefetch_outcome": nil, "tos_accepted_on_profile_info": nil,
	}
	return mustJSON(ri)
}
