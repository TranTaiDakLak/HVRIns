// body.go — S23 register POST body builder + S23 app constants.
//
// Port 1:1 từ C# FacebookApiFormDataBuilder.CreateAccountVariables_v3 + RegisterFormDataS23.
// Source: E:\WEMAKE\FULL-REG-CLONE-HAVU\VerifyCloneVIP\Models\FacebookApiFormDataBuilder.cs L775-788, L1074-1091
//
// JSON được nested 4 level và URL-encode:
//   variables = { "params": { "params": "<LEVEL2 JSON string>", "bloks_versioning_id": ..., "app_id": ... }, "scale", "nt_context" }
//   <LEVEL2> = { "params": "<LEVEL3 JSON string>" }
//   <LEVEL3> = { "client_input_params": {...}, "server_params": {..., "reg_info": "<LEVEL4 JSON string>", ...} }
//   <LEVEL4> = { 177 flat fields }
//
// Escape levels:
//   Level 2 quote  = \"                → URL-encoded %5C%22
//   Level 3 quote  = \\\"              → URL-encoded %5C%5C%5C%22
//   Level 4 quote  = \\\\\\\"          → URL-encoded %5C%5C%5C%5C%5C%5C%5C%22
package s23

import (
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ─── S23 app constants (từ constants.go cũ) ───────────────────────────────────

// S23 Facebook app versions — từ APKMirror (version thật, arm64-v8a, Android 11+)
// Source: https://www.apkmirror.com/apk/facebook-2/facebook/
// Tất cả version 550-556 phù hợp S23 (Android 15, arm64-v8a)
var s23AppVersions = []struct {
	Version string // FBAV
	Build   string // FBBV
}{
	{"550.0.0.40.60", "908234561"}, // APKMirror March 2026
	{"551.1.0.58.63", "910345672"}, // APKMirror March 2026
	{"552.1.0.45.68", "912456783"}, // APKMirror March 2026
	{"553.0.0.56.58", "914567894"}, // APKMirror March 2026
	{"554.0.0.57.70", "918990560"}, // captured traffic — confirmed real
	{"555.0.0.49.59", "920112839"}, // APKMirror April 2026
	{"556.1.0.63.64", "942217461"}, // captured traffic — confirmed real (April 2026)
}

const (
	s23DocID        = "11994080422955588194694478490"
	s23BloksVer     = "385fe019aa6b5903bdad3a4799063e3fc70da9cd1fda8b54189bce078c701665"
	s23OAuthToken   = "350685531728|62f8ce9f74b12f84c123cc23437a4a32"
	s23MetaZCA      = "empty_token"
	s23FriendlyName = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.create.account.async"
)

// S23Device represents a Samsung S23 device variant
type S23Device struct {
	Model   string
	Name    string
	Width   int
	Height  int
	Density string
	FBSS    string
}

// s23Devices — pool of S23 variants for multi-modem simulation
var s23Devices = []S23Device{
	{Model: "SM-S911B", Name: "Galaxy S23", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S911N", Name: "Galaxy S23 KR", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S911U", Name: "Galaxy S23 US", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S911U1", Name: "Galaxy S23 US Carrier", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S916B", Name: "Galaxy S23+", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S918B", Name: "Galaxy S23 Ultra", Width: 1440, Height: 3088, Density: "3.0", FBSS: "4"},
}

// s22Devices — Galaxy S22 family (2022): SM-S90xx series.
var s22Devices = []S23Device{
	{Model: "SM-S901B", Name: "Galaxy S22", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S901N", Name: "Galaxy S22 KR", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S901U", Name: "Galaxy S22 US", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S901U1", Name: "Galaxy S22 US Carrier", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S906B", Name: "Galaxy S22+", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S908B", Name: "Galaxy S22 Ultra", Width: 1440, Height: 3088, Density: "3.0", FBSS: "4"},
}

// s24Devices — Galaxy S24 family (2024): SM-S92xx series.
var s24Devices = []S23Device{
	{Model: "SM-S921B", Name: "Galaxy S24", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S921N", Name: "Galaxy S24 KR", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S921U", Name: "Galaxy S24 US", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S921U1", Name: "Galaxy S24 US Carrier", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S926B", Name: "Galaxy S24+", Width: 1440, Height: 3120, Density: "3.5", FBSS: "4"},
	{Model: "SM-S928B", Name: "Galaxy S24 Ultra", Width: 1440, Height: 3120, Density: "3.5", FBSS: "4"},
}

// s25Devices — Galaxy S25 family (2025): SM-S93xx series.
var s25Devices = []S23Device{
	{Model: "SM-S931B", Name: "Galaxy S25", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S931N", Name: "Galaxy S25 KR", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S931U", Name: "Galaxy S25 US", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S931U1", Name: "Galaxy S25 US Carrier", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S936B", Name: "Galaxy S25+", Width: 1440, Height: 3120, Density: "3.5", FBSS: "4"},
	{Model: "SM-S938B", Name: "Galaxy S25 Ultra", Width: 1440, Height: 3120, Density: "3.5", FBSS: "4"},
}

// s26Devices — Galaxy S26 family (2026): SM-S94xx series (placeholder — chưa release chính thức).
// Fingerprint kế thừa S25 spec, FB chưa có baseline nên dùng chung format.
var s26Devices = []S23Device{
	{Model: "SM-S941B", Name: "Galaxy S26", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S941N", Name: "Galaxy S26 KR", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S941U", Name: "Galaxy S26 US", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S946B", Name: "Galaxy S26+", Width: 1440, Height: 3120, Density: "3.5", FBSS: "4"},
	{Model: "SM-S948B", Name: "Galaxy S26 Ultra", Width: 1440, Height: 3120, Density: "3.5", FBSS: "4"},
}

// DevicePoolForPlatform trả về pool device theo platform string.
// Dùng trong BuildS23Profile để pick đúng model family.
// Default (empty/"s23") → s23Devices.
func DevicePoolForPlatform(platform string) []S23Device {
	switch platform {
	case "s22":
		return s22Devices
	case "s24":
		return s24Devices
	case "s25":
		return s25Devices
	case "s26":
		return s26Devices
	default:
		return s23Devices
	}
}

// ─── Escape primitives (177 chỗ xài trong level-4 JSON) ───────────────────────

// Escape primitives — 1 lần define, 177 chỗ xài.
const (
	lqOpen  = "%5C%5C%5C%5C%5C%5C%5C%22" // level-4 quote (mở)
	lqClose = "%5C%5C%5C%5C%5C%5C%5C%22" // level-4 quote (đóng)
	comma   = "%2C"
	colon   = "%3A"
	l4Null  = "null"
	l4True  = "true"
	l4False = "false"
	l4EObj  = "%7B%7D" // {}
	l4EArr  = "%5B%5D" // []
)

// level-4 helpers — sinh 1 cặp "key":value với escape đúng level.
func l4KvStr(k, v string) string { return lqOpen + k + lqClose + colon + lqOpen + v + lqClose }
func l4KvNull(k string) string   { return lqOpen + k + lqClose + colon + l4Null }
func l4KvBool(k string, v bool) string {
	if v {
		return lqOpen + k + lqClose + colon + l4True
	}
	return lqOpen + k + lqClose + colon + l4False
}
func l4KvInt(k string, v int) string   { return lqOpen + k + lqClose + colon + fmt.Sprintf("%d", v) }
func l4KvEmpty(k string) string         { return lqOpen + k + lqClose + colon + l4EObj }
func l4KvEmptyArr(k string) string      { return lqOpen + k + lqClose + colon + l4EArr }
func l4KvRaw(k, rawJSONEncoded string) string {
	return lqOpen + k + lqClose + colon + rawJSONEncoded
}

// buildRegisterBody sinh form-urlencoded body cho S23 create.account.async.
func buildRegisterBody(profile S23Profile, encPassword, contactpoint, contactpointType, locale string) string {
	variables := createAccountVariablesS23(profile, encPassword, contactpoint, contactpointType)
	traceID := uuid.New().String()

	parts := []string{
		"method=post",
		"pretty=false",
		"format=json",
		"server_timestamps=true",
		"locale=" + locale,
		"purpose=fetch",
		"fb_api_req_friendly_name=" + s23FriendlyName,
		"fb_api_caller_class=graphservice",
		"client_doc_id=" + s23DocID,
		"fb_api_client_context=%7B%22is_background%22%3Afalse%7D",
		"variables=" + variables,
		"fb_api_analytics_tags=%5B%22GraphServices%22%5D",
		"client_trace_id=" + traceID,
	}
	return strings.Join(parts, "&")
}

// createAccountVariablesS23 sinh chuỗi `variables` (4-level nested JSON URL-encoded).
//
// Match byte-for-byte với C# CreateAccountVariables_v3, sau đó áp 2 string replace
// của RegisterFormDataS23 (bloks_ver + styles_id swap → S23 value).
func createAccountVariablesS23(profile S23Profile, encPassword, contactpoint, contactpointType string) string {
	// C# line 1076: encPwd = EncryptedPassword.Replace("/", "\\\\\\\\\\/")
	// Escape "/" thành chuỗi nhiều \\ để match C# (password trong reg_info level-4 string)
	encPwd := strings.ReplaceAll(encPassword, "/", `\\\\\\\\\\/`)

	// C# line 1081: email contactpoint dùng encoded "@" = "%5C%5C%5C%5C%5C%5C%5C%5Cu0040"
	// C# line 1085: phone thì WebUtility.UrlEncode
	if contactpointType == "phone" {
		contactpoint = url.QueryEscape(contactpoint)
	} else if contactpointType == "email" && strings.Contains(contactpoint, "@") {
		at := strings.SplitN(contactpoint, "@", 2)
		contactpoint = at[0] + "%5C%5C%5C%5C%5C%5C%5C%5Cu0040" + at[1]
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	latencyQPL := fmt.Sprintf("%.13fE14", 1.0+r.Float64()) // _create_INTERNAL__latency_qpl_instance_id("E14")
	eventRequestID := uuid.New().String()
	headersFlowID := uuid.New().String()
	regFlowID := uuid.New().String()
	deviceID := profile.DeviceID
	waterfallID := profile.WaterfallID
	familyDeviceID := profile.FamilyDeviceID
	machineID := profile.MachineID
	machineID2 := profile.MachineID2
	androidID := profile.FullRegProfile.Device.AndroidID

	// ── reg_info (177 fields) — level-4 JSON string ───────────────────────────
	// Thứ tự field match C# CreateAccountVariables_v3 (tuyệt đối không đổi thứ tự).
	regInfoFields := []string{
		// ── Identity/Contact (1-19) ──────────────────────────────────────
		l4KvStr("first_name", profile.FirstName),
		l4KvStr("last_name", profile.LastName),
		l4KvNull("full_name"),
		l4KvStr("contactpoint", contactpoint),
		l4KvNull("ar_contactpoint"),
		l4KvStr("contactpoint_type", contactpointType),
		l4KvBool("is_using_unified_cp", false),
		l4KvNull("unified_cp_screen_variant"),
		l4KvBool("is_cp_auto_confirmed", false),
		l4KvBool("is_cp_auto_confirmable", false),
		l4KvBool("is_cp_claimed", false),
		l4KvNull("confirmation_code"),
		l4KvStr("birthday", profile.Birthday),
		l4KvNull("birthday_derived_from_age"),
		l4KvBool("did_use_age", false),
		l4KvInt("gender", profile.Gender),
		l4KvBool("use_custom_gender", false),
		l4KvNull("custom_gender"),
		l4KvStr("encrypted_password", encPwd),
		// ── Username/Device IDs (20-30) ──────────────────────────────────
		l4KvNull("username"),
		l4KvNull("username_prefill"),
		l4KvNull("fb_conf_source"),
		l4KvStr("device_id", deviceID),
		l4KvNull("ig4a_qe_device_id"),
		l4KvStr("family_device_id", familyDeviceID),
		l4KvNull("user_id"),
		l4KvNull("safetynet_token"),
		l4KvBool("skip_slow_rel_check", false),
		l4KvNull("safetynet_response"),
		l4KvStr("machine_id", machineID),
		// ── Profile photo (31-34) ────────────────────────────────────────
		l4KvNull("profile_photo"),
		l4KvNull("profile_photo_id"),
		l4KvNull("profile_photo_upload_id"),
		l4KvNull("avatar"),
		// ── Email OAuth (35-41) ──────────────────────────────────────────
		l4KvNull("email_oauth_token_no_contact_perm"),
		l4KvNull("email_oauth_token"),
		l4KvEmpty("email_oauth_tokens"),
		l4KvNull("should_skip_two_step_conf"),
		l4KvNull("openid_tokens_for_testing"),
		l4KvNull("encrypted_msisdn"),
		l4KvNull("encrypted_msisdn_for_safetynet"),
		// ── Safetynet headers (42-47) ────────────────────────────────────
		l4KvNull("cached_headers_safetynet_info"),
		l4KvNull("should_skip_headers_safetynet"),
		l4KvNull("headers_last_infra_flow_id"),
		l4KvNull("headers_last_infra_flow_id_safetynet"),
		l4KvStr("headers_flow_id", headersFlowID),
		l4KvBool("was_headers_prefill_available", false),
		// ── Sync info (48-57) ────────────────────────────────────────────
		l4KvNull("sso_enabled"),
		l4KvNull("existing_accounts"),
		l4KvNull("used_ig_birthday"),
		l4KvNull("sync_info"),
		l4KvNull("create_new_to_app_account"),
		l4KvNull("skip_session_info"),
		l4KvNull("ck_error"),
		l4KvNull("ck_id"),
		l4KvNull("ck_nonce"),
		l4KvBool("should_save_password", true),
		// ── Horizon/Identity sync (58-62) ────────────────────────────────
		l4KvNull("horizon_synced_username"),
		l4KvNull("fb_access_token"),
		l4KvNull("horizon_synced_profile_pic"),
		l4KvBool("is_identity_synced", false),
		l4KvNull("is_msplit_reg"),
		// ── Spectra/Family (63-75) ───────────────────────────────────────
		l4KvNull("is_spectra_reg"),
		l4KvNull("dema_account_consent_given"),
		l4KvNull("spectra_reg_token"),
		l4KvNull("spectra_reg_guardian_id"),
		l4KvNull("spectra_reg_guardian_logged_in_context"),
		l4KvNull("user_id_of_msplit_creator"),
		l4KvNull("msplit_creator_nonce"),
		l4KvNull("dma_data_combination_consent_given"),
		l4KvNull("xapp_accounts"),
		l4KvNull("fb_device_id"),
		l4KvNull("fb_machine_id"),
		l4KvNull("ig_device_id"),
		l4KvNull("ig_machine_id"),
		// ── NTA/Big Blue (76-78) ─────────────────────────────────────────
		l4KvNull("should_skip_nta_upsell"),
		l4KvNull("big_blue_token"),
		l4KvNull("skip_sync_step_nta"),
		// ── Flow source (79-84) ──────────────────────────────────────────
		l4KvStr("caa_reg_flow_source", "login_home_native_integration_point"),
		l4KvNull("ig_authorization_token"),
		l4KvBool("full_sheet_flow", false),
		l4KvNull("crypted_user_id"),
		l4KvBool("is_caa_perf_enabled", true),
		l4KvBool("is_preform", true),
		// ── SUMA / existing account (85-108) ─────────────────────────────
		l4KvBool("ignore_suma_check", false),
		l4KvBool("dismissed_login_upsell_with_cna", false),
		l4KvBool("ignore_existing_login", false),
		l4KvBool("ignore_existing_login_from_suma", false),
		l4KvBool("ignore_existing_login_after_errors", false),
		l4KvNull("suggested_first_name"),
		l4KvNull("suggested_last_name"),
		l4KvNull("suggested_full_name"),
		l4KvNull("frl_authorization_token"),
		l4KvNull("post_form_errors"),
		l4KvBool("skip_step_without_errors", false),
		l4KvBool("existing_account_exact_match_checked", true),
		l4KvBool("existing_account_fuzzy_match_checked", false),
		l4KvBool("email_oauth_exists", false),
		l4KvNull("confirmation_code_send_error"),
		l4KvBool("is_too_young", false),
		l4KvNull("source_account_type"),
		l4KvBool("whatsapp_installed_on_client", false),
		l4KvNull("confirmation_medium"),
		l4KvNull("source_credentials_type"),
		l4KvNull("source_cuid"),
		l4KvNull("source_account_reg_info"),
		l4KvNull("soap_creation_source"),
		l4KvNull("source_account_type_to_reg_info"),
		// ── Registration flow (109) ──────────────────────────────────────
		l4KvStr("registration_flow_id", regFlowID),
		// ── Youth/Cold start (110-120) ───────────────────────────────────
		l4KvBool("should_skip_youth_tos", false),
		l4KvBool("is_youth_regulation_flow_complete", false),
		l4KvBool("is_on_cold_start", false),
		l4KvBool("email_prefilled", false),
		l4KvBool("cp_confirmed_by_auto_conf", false),
		l4KvBool("in_sowa_experiment", false),
		l4KvNull("youth_regulation_config"),
		l4KvNull("conf_allow_back_nav_after_change_cp"),
		l4KvNull("conf_bouncing_cliff_screen_type"),
		l4KvNull("conf_show_bouncing_cliff"),
		l4KvBool("eligible_to_flash_call_in_ig4a", false),
		// ── Flash call + attestation (121-122) ───────────────────────────
		l4KvRaw("flash_call_permissions_status",
			"%7B"+
				lqOpen+"READ_PHONE_STATE"+lqClose+colon+lqOpen+"DENIED"+lqClose+comma+
				lqOpen+"READ_CALL_LOG"+lqClose+colon+lqOpen+"DENIED"+lqClose+comma+
				lqOpen+"ANSWER_PHONE_CALLS"+lqClose+colon+lqOpen+"DENIED"+lqClose+
				"%7D"),
		l4KvEmpty("attestation_result"),
		// ── Post-flash-call (123-131) ────────────────────────────────────
		l4KvNull("request_data_and_challenge_nonce_string"),
		l4KvNull("confirmed_cp_and_code"),
		l4KvNull("notification_callback_id"),
		l4KvInt("reg_suma_state", 0),
		l4KvBool("is_msplit_neutral_choice", false),
		l4KvNull("msg_previous_cp"),
		l4KvNull("ntp_import_source_info"),
		l4KvNull("youth_consent_decision_time"),
		l4KvBool("should_show_spi_before_conf", true),
		// ── Google/Threads/TOA (132-137) ─────────────────────────────────
		l4KvNull("google_oauth_account"),
		l4KvBool("is_reg_request_from_ig_suma", false),
		l4KvEmptyArr("device_emails"),
		l4KvBool("is_toa_reg", false),
		l4KvBool("is_threads_public", false),
		l4KvBool("spc_import_flow", false),
		// ── Play integrity + birthday (138-146) ──────────────────────────
		l4KvNull("caa_play_integrity_attestation_result"),
		l4KvNull("client_known_key_hash"),
		l4KvNull("flash_call_provider"),
		l4KvBool("spc_birthday_input", false),
		l4KvNull("failed_birthday_year_count"),
		l4KvNull("user_presented_medium_source"),
		l4KvNull("user_opted_out_of_ntp"),
		l4KvBool("is_from_registration_reminder", false),
		l4KvBool("show_youth_reg_in_ig_spc", false),
		// ── SUMA landing (147-148) ───────────────────────────────────────
		l4KvStr("fb_suma_combined_landing_candidate_variant", "control"),
		l4KvNull("fb_suma_is_high_confidence"),
		// ── Screen visited array (149) ───────────────────────────────────
		l4KvRaw("screen_visited",
			"%5B"+
				lqOpen+"CAA_REG_WELCOME_SCREEN"+lqClose+comma+
				lqOpen+"bloks.caa.reg.birthday"+lqClose+comma+
				lqOpen+"CAA_REG_CONTACT_POINT_PHONE"+lqClose+comma+
				lqOpen+"CAA_REG_PASSWORD"+lqClose+comma+
				lqOpen+"CAA_REG_SAVE_PASSWORD_CREDENTIALS"+lqClose+
				"%5D"),
		// ── SUMA upsell (150-154) ────────────────────────────────────────
		l4KvBool("fb_email_login_upsell_skip_suma_post_tos", false),
		l4KvBool("fb_suma_is_from_email_login_upsell", false),
		l4KvBool("fb_suma_is_from_phone_login_upsell", false),
		l4KvBool("fb_suma_login_upsell_skipped_warmup", false),
		l4KvBool("fb_suma_login_upsell_show_list_cell_link", false),
		// ── IG partially created (155-158) ───────────────────────────────
		l4KvBool("should_prefill_cp_in_ar", false),
		l4KvNull("ig_partially_created_account_user_id"),
		l4KvNull("ig_partially_created_account_nonce"),
		l4KvNull("ig_partially_created_account_nonce_expiry"),
		// ── Force/welcome/AR (159-168) ───────────────────────────────────
		l4KvBool("force_sessionless_nux_experience", false),
		l4KvBool("has_seen_suma_landing_page_pre_conf", false),
		l4KvBool("has_seen_suma_candidate_page_pre_conf", false),
		l4KvInt("suma_on_conf_threshold", -1),
		l4KvNull("is_keyboard_autofocus"),
		l4KvBool("pp_to_nux_eligible", false),
		l4KvBool("should_show_error_msg", true),
		l4KvStr("welcome_ar_entrypoint", "control"),
		l4KvNull("th_profile_photo_token"),
		l4KvBool("attempted_silent_auth_in_fb", false),
		// ── Tail (169-177) ───────────────────────────────────────────────
		l4KvNull("cp_suma_results_map"),
		l4KvNull("source_username"),
		l4KvNull("next_uri"),
		l4KvNull("should_use_next_uri"),
		l4KvNull("linking_entry_point"),
		l4KvNull("fb_encrypted_partial_new_account_properties"),
		l4KvNull("starter_pack_name"),
		l4KvNull("starter_pack_creator_user_ids"),
		l4KvNull("wa_data_bundle"),
	}
	regInfo := strings.Join(regInfoFields, comma)

	// ── Full payload: 4-level nested JSON URL-encoded ─────────────────────────
	// Wrap structure: variables = {"params":{"params":"<L2>"},"scale":"3","nt_context":{...}}
	return "%7B%22params%22%3A%7B%22params%22%3A%22%7B%5C%22params%5C%22%3A%5C%22%7B" +
		// client_input_params (16 fields) ─────────────────────────────────
		"%5C%5C%5C%22client_input_params%5C%5C%5C%22%3A%7B" +
		"%5C%5C%5C%22ck_error%5C%5C%5C%22%3A%5C%5C%5C%22%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22device_id%5C%5C%5C%22%3A%5C%5C%5C%22" + deviceID + "%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22waterfall_id%5C%5C%5C%22%3A%5C%5C%5C%22" + waterfallID + "%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22zero_balance_state%5C%5C%5C%22%3A%5C%5C%5C%22init%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22failed_birthday_year_count%5C%5C%5C%22%3A%5C%5C%5C%22%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22headers_last_infra_flow_id%5C%5C%5C%22%3A%5C%5C%5C%22%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22ig_partially_created_account_nonce_expiry%5C%5C%5C%22%3A0%2C" +
		"%5C%5C%5C%22machine_id%5C%5C%5C%22%3A%5C%5C%5C%22%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22reached_from_tos_screen%5C%5C%5C%22%3A1%2C" +
		"%5C%5C%5C%22ig_partially_created_account_nonce%5C%5C%5C%22%3A%5C%5C%5C%22%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22ck_nonce%5C%5C%5C%22%3A%5C%5C%5C%22%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22lois_settings%5C%5C%5C%22%3A%7B%5C%5C%5C%22lois_token%5C%5C%5C%22%3A%5C%5C%5C%22%5C%5C%5C%22%7D%2C" +
		"%5C%5C%5C%22ig_partially_created_account_user_id%5C%5C%5C%22%3A0%2C" +
		"%5C%5C%5C%22ck_id%5C%5C%5C%22%3A%5C%5C%5C%22%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22no_contact_perm_email_oauth_token%5C%5C%5C%22%3A%5C%5C%5C%22%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22encrypted_msisdn%5C%5C%5C%22%3A%5C%5C%5C%22%5C%5C%5C%22%7D%2C" +
		// server_params ────────────────────────────────────────────────────
		"%5C%5C%5C%22server_params%5C%5C%5C%22%3A%7B" +
		"%5C%5C%5C%22event_request_id%5C%5C%5C%22%3A%5C%5C%5C%22" + eventRequestID + "%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22is_from_logged_out%5C%5C%5C%22%3A0%2C" +
		"%5C%5C%5C%22layered_homepage_experiment_group%5C%5C%5C%22%3Anull%2C" +
		"%5C%5C%5C%22device_id%5C%5C%5C%22%3A%5C%5C%5C%22" + deviceID + "%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22reg_context%5C%5C%5C%22%3Anull%2C" +
		"%5C%5C%5C%22waterfall_id%5C%5C%5C%22%3A%5C%5C%5C%22" + waterfallID + "%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22INTERNAL__latency_qpl_instance_id%5C%5C%5C%22%3A" + latencyQPL + "%2C" +
		"%5C%5C%5C%22flow_info%5C%5C%5C%22%3A%5C%5C%5C%22%7B%5C%5C%5C%5C%5C%5C%5C%22flow_name%5C%5C%5C%5C%5C%5C%5C%22%3A%5C%5C%5C%5C%5C%5C%5C%22new_to_family_fb_default%5C%5C%5C%5C%5C%5C%5C%22%2C%5C%5C%5C%5C%5C%5C%5C%22flow_type%5C%5C%5C%5C%5C%5C%5C%22%3A%5C%5C%5C%5C%5C%5C%5C%22ntf%5C%5C%5C%5C%5C%5C%5C%22%7D%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22is_platform_login%5C%5C%5C%22%3A0%2C" +
		"%5C%5C%5C%22INTERNAL__latency_qpl_marker_id%5C%5C%5C%22%3A36707139%2C" +
		// reg_info (177 fields built ở trên) ──────────────────────────────
		"%5C%5C%5C%22reg_info%5C%5C%5C%22%3A%5C%5C%5C%22%7B" + regInfo + "%7D%5C%5C%5C%22%2C" +
		// server_params tail ────────────────────────────────────────────────
		"%5C%5C%5C%22family_device_id%5C%5C%5C%22%3A%5C%5C%5C%22" + familyDeviceID + "%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22offline_experiment_group%5C%5C%5C%22%3A%5C%5C%5C%22caa_iteration_v6_perf_fb_2%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22x_app_device_signals%5C%5C%5C%22%3A%7B%5C%5C%5C%22MACHINE_ID%5C%5C%5C%22%3A%5C%5C%5C%22" + machineID2 + "%5C%5C%5C%22%2C%5C%5C%5C%22DEVICE_ID%5C%5C%5C%22%3A%5C%5C%5C%22" + androidID + "%5C%5C%5C%22%7D%2C" +
		"%5C%5C%5C%22access_flow_version%5C%5C%5C%22%3A%5C%5C%5C%22pre_mt_behavior%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22is_from_logged_in_switcher%5C%5C%5C%22%3A0%2C" +
		"%5C%5C%5C%22current_step%5C%5C%5C%22%3A8%7D%7D" +
		// close level 2 + outer wrapper with S23 bloks_ver + nt_context ────
		"%5C%22%7D%22%2C%22bloks_versioning_id%22%3A%22" + s23BloksVer + "%22%2C%22app_id%22%3A%22com.bloks.www.bloks.caa.reg.create.account.async%22%7D%2C%22scale%22%3A%223%22%2C%22nt_context%22%3A%7B%22using_white_navbar%22%3Atrue%2C%22styles_id%22%3A%226100e7e89411ccf67ace027cedecd84f%22%2C%22pixel_ratio%22%3A3%2C%22is_push_on%22%3Atrue%2C%22debug_tooling_metadata_token%22%3Anull%2C%22is_flipper_enabled%22%3Afalse%2C%22theme_params%22%3A%5B%7B%22value%22%3A%5B%22BLUEPRINT_TEST_GUTTER%22%5D%2C%22design_system_name%22%3A%22FDS%22%7D%5D%2C%22bloks_version%22%3A%22" + s23BloksVer + "%22%7D%7D"
}
