// body.go — s530 register POST body builder + app constants.
// FBAV/530.0.0.48.74, FBBV/465017152.
// Khác s559: doc_id mới, bloks_ver mới, FBAV/FBBV mới.
// Logic identical với s559 (is_push_on=false, theme_params=[XMDS+FDS], aac="" trong client_input_params).
package s530

import (
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ─── s530 app constants ───────────────────────────────────────────────────────

var s530AppVersions = []struct {
	Version string // FBAV
	Build   string // FBBV
}{
	{"530.0.0.48.74", "465017152"}, // captured traffic — confirmed
}

const (
	s530DocID        = "119940804210382635615484396344"
	s530BloksVer     = "182a4e03087cc46a88a95c6e5747a622a0ab08e2134522bd0e6da65b9ceea9fd"
	s530OAuthToken   = "350685531728|62f8ce9f74b12f84c123cc23437a4a32"
	s530MetaZCA      = "empty_token"
	s530FriendlyName = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.create.account.async"
	s530StylesID     = "fefbd1d6b452bcac026b965ec70e37ee"

	// OriginalUA — UA gốc cố định cho platform s530 (SM-S911B, en_GB, Viettel).
	OriginalUA = "[FBAN/FB4A;FBAV/530.0.0.48.74;FBBV/465017152;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]"
)

// S23Device represents a Samsung S23 device variant (reused for s530)
type S23Device struct {
	Model   string
	Name    string
	Width   int
	Height  int
	Density string
	FBSS    string
}

var s23Devices = []S23Device{
	{Model: "SM-S911B", Name: "Galaxy S23", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S911N", Name: "Galaxy S23 KR", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S911U", Name: "Galaxy S23 US", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S911U1", Name: "Galaxy S23 US Carrier", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S916B", Name: "Galaxy S23+", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S918B", Name: "Galaxy S23 Ultra", Width: 1440, Height: 3088, Density: "3.0", FBSS: "4"},
}

var s24Devices = []S23Device{
	{Model: "SM-S921B", Name: "Galaxy S24", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S921U", Name: "Galaxy S24 US", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Model: "SM-S926B", Name: "Galaxy S24+", Width: 1440, Height: 3120, Density: "3.5", FBSS: "4"},
	{Model: "SM-S928B", Name: "Galaxy S24 Ultra", Width: 1440, Height: 3120, Density: "3.5", FBSS: "4"},
}

// DevicePoolForPlatform trả về pool device theo platform.
func DevicePoolForPlatform(platform string) []S23Device {
	switch platform {
	case "s24":
		return s24Devices
	default: // "s530", "s23"
		return s23Devices
	}
}

// ─── Escape primitives ────────────────────────────────────────────────────────

const (
	lqOpen  = "%5C%5C%5C%5C%5C%5C%5C%22"
	lqClose = "%5C%5C%5C%5C%5C%5C%5C%22"
	comma   = "%2C"
	colon   = "%3A"
	l4Null  = "null"
	l4True  = "true"
	l4False = "false"
	l4EObj  = "%7B%7D"
	l4EArr  = "%5B%5D"
)

func l4KvStr(k, v string) string { return lqOpen + k + lqClose + colon + lqOpen + v + lqClose }
func l4KvNull(k string) string   { return lqOpen + k + lqClose + colon + l4Null }
func l4KvBool(k string, v bool) string {
	if v {
		return lqOpen + k + lqClose + colon + l4True
	}
	return lqOpen + k + lqClose + colon + l4False
}
func l4KvInt(k string, v int) string { return lqOpen + k + lqClose + colon + fmt.Sprintf("%d", v) }
func l4KvEmpty(k string) string      { return lqOpen + k + lqClose + colon + l4EObj }
func l4KvEmptyArr(k string) string   { return lqOpen + k + lqClose + colon + l4EArr }
func l4KvRaw(k, rawJSONEncoded string) string {
	return lqOpen + k + lqClose + colon + rawJSONEncoded
}

// buildRegisterBody sinh form-urlencoded body cho s530 reg.create.account.async.
func buildRegisterBody(profile s530Profile, encPassword, contactpoint, contactpointType, locale string) string {
	variables := createAccountVariabless530(profile, encPassword, contactpoint, contactpointType)
	traceID := uuid.New().String()

	parts := []string{
		"method=post",
		"pretty=false",
		"format=json",
		"server_timestamps=true",
		"locale=" + locale,
		"purpose=fetch",
		"fb_api_req_friendly_name=" + s530FriendlyName,
		"fb_api_caller_class=graphservice",
		"client_doc_id=" + s530DocID,
		"fb_api_client_context=%7B%22is_background%22%3Afalse%7D",
		"variables=" + variables,
		"fb_api_analytics_tags=%5B%22GraphServices%22%5D",
		"client_trace_id=" + traceID,
	}
	return strings.Join(parts, "&")
}

func createAccountVariabless530(profile s530Profile, encPassword, contactpoint, contactpointType string) string {
	encPwd := strings.ReplaceAll(encPassword, "/", `\\\\\\\\\\/`)

	if contactpointType == "phone" {
		contactpoint = url.QueryEscape(contactpoint)
	} else if contactpointType == "email" && strings.Contains(contactpoint, "@") {
		at := strings.SplitN(contactpoint, "@", 2)
		contactpoint = at[0] + "%5C%5C%5C%5C%5C%5C%5C%5Cu0040" + at[1]
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	latencyQPL := fmt.Sprintf("%.13fE14", 1.0+r.Float64())
	eventRequestID := uuid.New().String()
	headersFlowID := uuid.New().String()
	regFlowID := uuid.New().String()
	deviceID := profile.DeviceID
	waterfallID := profile.WaterfallID
	familyDeviceID := profile.FamilyDeviceID
	machineID := profile.MachineID
	machineID2 := profile.MachineID2
	androidID := profile.FullRegProfile.Device.AndroidID

	regInfoFields := []string{
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
		l4KvNull("profile_photo"),
		l4KvNull("profile_photo_id"),
		l4KvNull("profile_photo_upload_id"),
		l4KvNull("avatar"),
		l4KvNull("email_oauth_token_no_contact_perm"),
		l4KvNull("email_oauth_token"),
		l4KvEmpty("email_oauth_tokens"),
		l4KvNull("should_skip_two_step_conf"),
		l4KvNull("openid_tokens_for_testing"),
		l4KvNull("encrypted_msisdn"),
		l4KvNull("encrypted_msisdn_for_safetynet"),
		l4KvNull("cached_headers_safetynet_info"),
		l4KvNull("should_skip_headers_safetynet"),
		l4KvNull("headers_last_infra_flow_id"),
		l4KvNull("headers_last_infra_flow_id_safetynet"),
		l4KvStr("headers_flow_id", headersFlowID),
		l4KvBool("was_headers_prefill_available", false),
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
		l4KvNull("horizon_synced_username"),
		l4KvNull("fb_access_token"),
		l4KvNull("horizon_synced_profile_pic"),
		l4KvBool("is_identity_synced", false),
		l4KvNull("is_msplit_reg"),
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
		l4KvNull("should_skip_nta_upsell"),
		l4KvNull("big_blue_token"),
		l4KvNull("skip_sync_step_nta"),
		l4KvStr("caa_reg_flow_source", "login_home_native_integration_point"),
		l4KvNull("ig_authorization_token"),
		l4KvBool("full_sheet_flow", false),
		l4KvNull("crypted_user_id"),
		l4KvBool("is_caa_perf_enabled", true),
		l4KvBool("is_preform", true),
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
		l4KvStr("registration_flow_id", regFlowID),
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
		l4KvRaw("flash_call_permissions_status",
			"%7B"+
				lqOpen+"READ_PHONE_STATE"+lqClose+colon+lqOpen+"DENIED"+lqClose+comma+
				lqOpen+"READ_CALL_LOG"+lqClose+colon+lqOpen+"DENIED"+lqClose+comma+
				lqOpen+"ANSWER_PHONE_CALLS"+lqClose+colon+lqOpen+"DENIED"+lqClose+
				"%7D"),
		l4KvEmpty("attestation_result"),
		l4KvNull("request_data_and_challenge_nonce_string"),
		l4KvNull("confirmed_cp_and_code"),
		l4KvNull("notification_callback_id"),
		l4KvInt("reg_suma_state", 0),
		l4KvBool("is_msplit_neutral_choice", false),
		l4KvNull("msg_previous_cp"),
		l4KvNull("ntp_import_source_info"),
		l4KvNull("youth_consent_decision_time"),
		l4KvBool("should_show_spi_before_conf", true),
		l4KvNull("google_oauth_account"),
		l4KvBool("is_reg_request_from_ig_suma", false),
		l4KvEmptyArr("device_emails"),
		l4KvBool("is_toa_reg", false),
		l4KvBool("is_threads_public", false),
		l4KvBool("spc_import_flow", false),
		l4KvNull("caa_play_integrity_attestation_result"),
		l4KvNull("client_known_key_hash"),
		l4KvNull("flash_call_provider"),
		l4KvBool("spc_birthday_input", false),
		l4KvNull("failed_birthday_year_count"),
		l4KvNull("user_presented_medium_source"),
		l4KvNull("user_opted_out_of_ntp"),
		l4KvBool("is_from_registration_reminder", false),
		l4KvBool("show_youth_reg_in_ig_spc", false),
		l4KvStr("fb_suma_combined_landing_candidate_variant", "control"),
		l4KvNull("fb_suma_is_high_confidence"),
		l4KvRaw("screen_visited",
			"%5B"+
				lqOpen+"CAA_REG_WELCOME_SCREEN"+lqClose+comma+
				lqOpen+"bloks.caa.reg.birthday"+lqClose+comma+
				lqOpen+"CAA_REG_CONTACT_POINT_PHONE"+lqClose+comma+
				lqOpen+"CAA_REG_PASSWORD"+lqClose+comma+
				lqOpen+"CAA_REG_SAVE_PASSWORD_CREDENTIALS"+lqClose+
				"%5D"),
		l4KvBool("fb_email_login_upsell_skip_suma_post_tos", false),
		l4KvBool("fb_suma_is_from_email_login_upsell", false),
		l4KvBool("fb_suma_is_from_phone_login_upsell", false),
		l4KvBool("fb_suma_login_upsell_skipped_warmup", false),
		l4KvBool("fb_suma_login_upsell_show_list_cell_link", false),
		l4KvBool("should_prefill_cp_in_ar", false),
		l4KvNull("ig_partially_created_account_user_id"),
		l4KvNull("ig_partially_created_account_nonce"),
		l4KvNull("ig_partially_created_account_nonce_expiry"),
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

	// ── 560 client_input_params giữ nguyên cấu trúc 559 (aac="") ────────────────
	return "%7B%22params%22%3A%7B%22params%22%3A%22%7B%5C%22params%5C%22%3A%5C%22%7B" +
		"%5C%5C%5C%22client_input_params%5C%5C%5C%22%3A%7B" +
		"%5C%5C%5C%22ck_error%5C%5C%5C%22%3A%5C%5C%5C%22%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22aac%5C%5C%5C%22%3A%5C%5C%5C%22%5C%5C%5C%22%2C" +
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
		"%5C%5C%5C%22reg_info%5C%5C%5C%22%3A%5C%5C%5C%22%7B" + regInfo + "%7D%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22family_device_id%5C%5C%5C%22%3A%5C%5C%5C%22" + familyDeviceID + "%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22offline_experiment_group%5C%5C%5C%22%3A%5C%5C%5C%22caa_iteration_v6_perf_fb_2%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22x_app_device_signals%5C%5C%5C%22%3A%7B%5C%5C%5C%22MACHINE_ID%5C%5C%5C%22%3A%5C%5C%5C%22" + machineID2 + "%5C%5C%5C%22%2C%5C%5C%5C%22DEVICE_ID%5C%5C%5C%22%3A%5C%5C%5C%22" + androidID + "%5C%5C%5C%22%7D%2C" +
		"%5C%5C%5C%22access_flow_version%5C%5C%5C%22%3A%5C%5C%5C%22pre_mt_behavior%5C%5C%5C%22%2C" +
		"%5C%5C%5C%22is_from_logged_in_switcher%5C%5C%5C%22%3A0%2C" +
		"%5C%5C%5C%22current_step%5C%5C%5C%22%3A8%7D%7D" +
		"%5C%22%7D%22%2C%22bloks_versioning_id%22%3A%22" + s530BloksVer + "%22%2C%22app_id%22%3A%22com.bloks.www.bloks.caa.reg.create.account.async%22%7D%2C%22scale%22%3A%223%22%2C%22nt_context%22%3A%7B%22using_white_navbar%22%3Atrue%2C%22styles_id%22%3A%22" + s530StylesID + "%22%2C%22pixel_ratio%22%3A3%2C%22is_push_on%22%3Afalse%2C%22debug_tooling_metadata_token%22%3Anull%2C%22is_flipper_enabled%22%3Afalse%2C%22theme_params%22%3A%5B%7B%22value%22%3A%5B%22three_neutral_gray%22%5D%2C%22design_system_name%22%3A%22XMDS%22%7D%2C%7B%22value%22%3A%5B%5D%2C%22design_system_name%22%3A%22FDS%22%7D%5D%2C%22bloks_version%22%3A%22" + s530BloksVer + "%22%7D%7D"
}
