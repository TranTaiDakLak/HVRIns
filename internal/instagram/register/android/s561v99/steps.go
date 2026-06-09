// steps.go — Type Reg 2: step-specific constants, CIP builders, body/vars builders.
// Flow: POST step1(name) → step2(birthday) → step3(gender) → step5(email/phone)
//        → step6(password) → step7(create-check) → step8(create-final).
package s561v99

import (
	"fmt"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ─── Step friendly names & app IDs ───────────────────────────────────────────

const (
	s1FriendlyName = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.name.async"
	s1AppID        = "com.bloks.www.bloks.caa.reg.name.async"

	s2FriendlyName = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.birthday.async"
	s2AppID        = "com.bloks.www.bloks.caa.reg.birthday.async"

	s3FriendlyName = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.gender.async"
	s3AppID        = "com.bloks.www.bloks.caa.reg.gender.async"

	// Step 4: FETCH screen template email (Query, NOT Action) — port từ FB4A v561 capture.
	// Khác step Action: dùng FbBloksAppRootQuery + client_doc_id riêng + body minimal.
	// Mục đích: match đúng pattern app thật (step 1→2→3→4→5→...) → giảm rủi ro FB anti-bot.
	s4FriendlyName = "FbBloksAppRootQuery-com.bloks.www.bloks.caa.reg.contactpoint_email"
	s4AppID        = "com.bloks.www.bloks.caa.reg.contactpoint_email"
	s4DocID        = "10537346110209475073620198126" // doc_id khác với S560DocID

	s5FriendlyName = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.async.contactpoint_email.async"
	s5AppID        = "com.bloks.www.bloks.caa.reg.async.contactpoint_email.async"

	s6FriendlyName = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.password.async"
	s6AppID        = "com.bloks.www.bloks.caa.reg.password.async"

	// Steps 7 & 8 dùng cùng endpoint create.account như Type Reg 1
	s78AppID = "com.bloks.www.bloks.caa.reg.create.account.async"
)

// buildStep4Body — Query body cho FbBloksAppRootQuery contactpoint_email.
// Port từ capture FB4A v561: client_input_params minimal (lois_settings, zero_balance_state),
// server_params (is_from_logged_out, root_screen_id, aac, device_id, reg_context=null).
// Reg_context để null vì Go code không extract từ response trước (FB chấp nhận null).
//
// Pattern AAC: dùng S560AACJSON(ts) như các step Action khác — fresh aacjid mỗi call.
// (FB không kiểm tra aacjid consistency giữa các step trong session.)
func buildStep4Body(profile S560Profile, locale string) string {
	deviceID := profile.DeviceID
	ts := time.Now().Unix()

	// AAC JSON: same format as other steps, but escaped for nesting inside variables string.
	aacRaw := S560AACJSON(ts)                       // {"aac_init_timestamp":N,"aaccs":"...","aacjid":"..."}
	aacEscaped := strings.ReplaceAll(aacRaw, `"`, `\"`) // single-level escape for inner JSON

	innerJSON := `{\"client_input_params\":{` +
		`\"lois_settings\":{\"lois_token\":\"\"},` +
		`\"zero_balance_state\":\"init\"` +
		`},` +
		`\"server_params\":{` +
		`\"is_from_logged_out\":0,` +
		`\"root_screen_id\":\"bloks.caa.reg.contactpoint_phone\",` +
		`\"aac\":\"` + aacEscaped + `\",` +
		`\"device_id\":\"` + deviceID + `\",` +
		`\"reg_context\":null` +
		`}}`

	// Wrap: variables = {"params":{"params":"<innerJSON>","bloks_versioning_id":"...","app_id":"..."}}
	variablesJSON := `{"params":{"params":"` + innerJSON + `","bloks_versioning_id":"` + S560BloksVer + `","app_id":"` + s4AppID + `"}}`

	form := url.Values{}
	form.Set("method", "post")
	form.Set("pretty", "false")
	form.Set("format", "json")
	form.Set("server_timestamps", "true")
	form.Set("locale", locale)
	form.Set("purpose", "fetch")
	form.Set("fb_api_req_friendly_name", s4FriendlyName)
	form.Set("fb_api_caller_class", "graphservice")
	form.Set("client_doc_id", s4DocID)
	form.Set("fb_api_client_context", `{"is_background":false}`)
	form.Set("variables", variablesJSON)
	form.Set("fb_api_analytics_tags", `["GraphServices"]`)
	form.Set("client_trace_id", uuid.New().String())
	return form.Encode()
}

// ─── screen_visited arrays (pre-built, l4 / 7-backslash level) ───────────────

var (
	sv1    = buildSV([]string{"CAA_REG_WELCOME_SCREEN"})
	sv2    = buildSV([]string{"CAA_REG_WELCOME_SCREEN", "bloks.caa.reg.birthday"})
	sv3    = sv2 // gender screen không có ID riêng trong screen_visited
	sv5Eml = buildSV([]string{"CAA_REG_WELCOME_SCREEN", "bloks.caa.reg.birthday", "CAA_REG_CONTACT_POINT_PHONE", "CAA_REG_CONTACT_POINT_EMAIL"})
	sv5Ph  = buildSV([]string{"CAA_REG_WELCOME_SCREEN", "bloks.caa.reg.birthday", "CAA_REG_CONTACT_POINT_PHONE"})
	sv6Eml = buildSV([]string{"CAA_REG_WELCOME_SCREEN", "bloks.caa.reg.birthday", "CAA_REG_CONTACT_POINT_PHONE", "CAA_REG_CONTACT_POINT_EMAIL", "CAA_REG_PASSWORD"})
	sv6Ph  = buildSV([]string{"CAA_REG_WELCOME_SCREEN", "bloks.caa.reg.birthday", "CAA_REG_CONTACT_POINT_PHONE", "CAA_REG_PASSWORD"})
)

// buildSV builds URL-encoded screen_visited array at l4 level (7-backslash).
func buildSV(screens []string) string {
	parts := make([]string, len(screens))
	for i, s := range screens {
		parts[i] = lqOpen + s + lqClose
	}
	return "%5B" + strings.Join(parts, comma) + "%5D"
}

// ─── CIP helpers (3-backslash / client_input_params level) ───────────────────

const cq3 = "%5C%5C%5C%22" // URL-encoded triple-backslash quote

func cip3KvStr(k, v string) string {
	enc := url.QueryEscape(v) // space → +, special chars → %XX
	enc = strings.ReplaceAll(enc, "%22", cq3)
	return cq3 + k + cq3 + "%3A" + cq3 + enc + cq3
}

func cip3KvInt(k string, v int) string {
	return cq3 + k + cq3 + "%3A" + fmt.Sprintf("%d", v)
}

// cip3ObjRaw builds key:{innerRaw} at CIP level; innerRaw là content bên trong {}.
func cip3ObjRaw(k, innerRaw string) string {
	return cq3 + k + cq3 + "%3A%7B" + innerRaw + "%7D"
}

const cip3Sep = "%2C" // comma separator at CIP level

// buildBaseCIP returns common client_input_params fields shared by all steps.
func buildBaseCIP(aacEncoded, deviceID, waterfallID string) string {
	loisInner := cq3 + "lois_token" + cq3 + "%3A" + cq3 + cq3 // "lois_token":""
	return strings.Join([]string{
		cq3 + "aac" + cq3 + "%3A" + cq3 + aacEncoded + cq3,
		cip3KvStr("device_id", deviceID),
		cip3KvStr("waterfall_id", waterfallID),
		cip3KvStr("zero_balance_state", "init"),
		cip3KvStr("failed_birthday_year_count", ""),
		cip3KvStr("headers_last_infra_flow_id", ""),
		cip3KvInt("ig_partially_created_account_nonce_expiry", 0),
		cip3KvStr("machine_id", ""),
		cip3KvInt("reached_from_tos_screen", 1),
		cip3KvStr("ig_partially_created_account_nonce", ""),
		cip3KvStr("ck_nonce", ""),
		cip3ObjRaw("lois_settings", loisInner),
		cip3KvInt("ig_partially_created_account_user_id", 0),
		cip3KvStr("ck_id", ""),
		cip3KvStr("no_contact_perm_email_oauth_token", ""),
		cip3KvStr("encrypted_msisdn", ""),
		cip3KvStr("ck_error", ""),
	}, cip3Sep)
}

// ─── Birthday helpers ─────────────────────────────────────────────────────────

// bdayToUnix parses "DD-MM-YYYY" (profile format) → unix timestamp.
func bdayToUnix(bday string) int64 {
	var d, m, y int
	if _, err := fmt.Sscanf(bday, "%02d-%02d-%04d", &d, &m, &y); err != nil {
		return 0
	}
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC).Unix()
}

// ─── Intermediate response check ─────────────────────────────────────────────

func checkStepResp(body, stepName string) error {
	clean := strings.ReplaceAll(body, "\\", "")
	switch {
	case strings.Contains(clean, "couldn't create an account for you"):
		return fmt.Errorf("%s: account creation denied", stepName)
	case strings.Contains(clean, "integrity_block"):
		return fmt.Errorf("%s: integrity_block", stepName)
	case strings.Contains(clean, "\"checkpoint\""):
		return fmt.Errorf("%s: checkpoint", stepName)
	}
	return nil
}

// ─── Step variables builder ───────────────────────────────────────────────────

// riState controls which fields are "committed" (non-null) in reg_info.
type riState struct {
	nameSet   bool // first_name, last_name
	bdaySet   bool // birthday
	genderSet bool // gender (int vs null)
	cpSet     bool // contactpoint, contactpoint_type
	pwdSet    bool // encrypted_password
}

type stepParams struct {
	profile          S560Profile
	friendlyName     string
	appID            string
	currentStep      int
	cipSpecific      string // step-specific CIP fields (no trailing separator)
	ri               riState
	screenVisited    string // pre-built %5B...%5D
	encPassword      string // needed when ri.pwdSet
	contactpoint     string
	contactpointType string
}

func buildStepVars(p stepParams) string {
	ts := time.Now().Unix()
	latencyQPL := fmt.Sprintf("%.13fE14", 1.0+rand.New(rand.NewSource(time.Now().UnixNano())).Float64())
	eventRequestID := uuid.New().String()
	regFlowID := uuid.New().String()

	prof := p.profile
	deviceID := prof.DeviceID
	waterfallID := prof.WaterfallID
	familyDeviceID := prof.FamilyDeviceID
	machineID := prof.MachineID
	machineID2 := prof.MachineID2
	androidID := prof.FullRegProfile.Device.AndroidID
	aacEncoded := S560EscL3StringValue(S560AACJSON(ts))
	safetynetToken := S560FakeSafetyNetToken(ts)
	playIntegrityToken := S560FakePlayIntegrityToken()
	cpForAttest := p.contactpoint
	if cpForAttest == "" {
		cpForAttest = "unknown"
	}

	// ── reg_info fields ──────────────────────────────────────────────────────
	var l4FN, l4LN string
	if p.ri.nameSet {
		l4FN = l4KvStr("first_name", prof.FirstName)
		l4LN = l4KvStr("last_name", prof.LastName)
	} else {
		l4FN = l4KvNull("first_name")
		l4LN = l4KvNull("last_name")
	}

	var l4CP, l4CPType string
	if p.ri.cpSet {
		l4CP = l4KvStr("contactpoint", encCP4RegInfo(p.contactpoint, p.contactpointType))
		l4CPType = l4KvStr("contactpoint_type", p.contactpointType)
	} else {
		l4CP = l4KvNull("contactpoint")
		l4CPType = l4KvNull("contactpoint_type")
	}

	var l4Bday string
	if p.ri.bdaySet {
		l4Bday = l4KvStr("birthday", prof.Birthday)
	} else {
		l4Bday = l4KvNull("birthday")
	}

	var l4Gender string
	if p.ri.genderSet {
		l4Gender = l4KvInt("gender", prof.Gender)
	} else {
		l4Gender = l4KvNull("gender")
	}

	var l4Pwd string
	if p.ri.pwdSet && p.encPassword != "" {
		encPwd := strings.ReplaceAll(p.encPassword, "/", `\\\\\\\\\\/`)
		l4Pwd = l4KvStr("encrypted_password", encPwd)
	} else {
		l4Pwd = l4KvNull("encrypted_password")
	}

	regInfoFields := []string{
		l4FN, l4LN,
		l4KvNull("full_name"),
		l4CP, l4KvNull("ar_contactpoint"), l4CPType,
		l4KvBool("is_using_unified_cp", false),
		l4KvNull("unified_cp_screen_variant"),
		l4KvBool("is_cp_auto_confirmed", false),
		l4KvBool("is_cp_auto_confirmable", false),
		l4KvBool("is_cp_claimed", false),
		l4KvNull("confirmation_code"),
		l4Bday,
		l4KvNull("birthday_derived_from_age"),
		l4KvStr("age_range", "o18"),
		l4KvBool("did_use_age", false),
		l4KvNull("os_shared_age_range"),
		l4Gender,
		l4KvBool("use_custom_gender", false),
		l4KvNull("custom_gender"),
		l4Pwd,
		l4KvNull("username"),
		l4KvNull("username_prefill"),
		l4KvEmptyArr("accounts_list_client"),
		l4KvNull("fb_conf_source"),
		l4KvStr("device_id", deviceID),
		l4KvNull("ig4a_qe_device_id"),
		l4KvStr("family_device_id", familyDeviceID),
		l4KvNull("user_id"),
		l4KvStr("safetynet_token", safetynetToken),
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
		l4KvNull("sign_in_with_google_email"),
		l4KvNull("should_skip_two_step_conf"),
		l4KvNull("openid_tokens_for_testing"),
		l4KvNull("encrypted_msisdn"),
		l4KvNull("encrypted_msisdn_for_safetynet"),
		l4KvNull("cached_headers_safetynet_info"),
		l4KvNull("should_skip_headers_safetynet"),
		l4KvNull("headers_last_infra_flow_id"),
		l4KvNull("headers_last_infra_flow_id_safetynet"),
		l4KvNull("headers_flow_id"),
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
		l4KvBool("eligible_to_mo_sms_in_ig4a", false),
		l4KvNull("mo_sms_ent_id"),
		l4KvRaw("flash_call_permissions_status",
			"%7B"+
				lqOpen+"READ_PHONE_STATE"+lqClose+colon+lqOpen+"DENIED"+lqClose+comma+
				lqOpen+"READ_CALL_LOG"+lqClose+colon+lqOpen+"DENIED"+lqClose+comma+
				lqOpen+"ANSWER_PHONE_CALLS"+lqClose+colon+lqOpen+"DENIED"+lqClose+
				"%7D"),
		l4KvStr("gms_incoming_call_retriever_eligibility", "eligible"),
		l4KvRaw("attestation_result", S560FakeAttestationResultEncoded(cpForAttest)),
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
		l4KvStr("caa_play_integrity_attestation_result", playIntegrityToken),
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
		l4KvRaw("screen_visited", p.screenVisited),
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

	// ── CIP section ──────────────────────────────────────────────────────────
	baseCIP := buildBaseCIP(aacEncoded, deviceID, waterfallID)
	var cipContent string
	if p.cipSpecific != "" {
		cipContent = p.cipSpecific + cip3Sep + baseCIP
	} else {
		cipContent = baseCIP
	}

	// ── Assemble full variables string ────────────────────────────────────────
	stepStr := fmt.Sprintf("%d", p.currentStep)
	return "%7B%22params%22%3A%7B%22params%22%3A%22%7B%5C%22params%5C%22%3A%5C%22%7B" +
		"%5C%5C%5C%22client_input_params%5C%5C%5C%22%3A%7B" +
		cipContent +
		"%7D%2C" +
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
		"%5C%5C%5C%22current_step%5C%5C%5C%22%3A" + stepStr + "%7D%7D" +
		"%5C%22%7D%22%2C%22bloks_versioning_id%22%3A%22" + S560BloksVer +
		"%22%2C%22app_id%22%3A%22" + p.appID +
		"%22%7D%2C%22scale%22%3A%223%22%2C%22nt_context%22%3A%7B%22using_white_navbar%22%3Atrue%2C%22styles_id%22%3A%22" + S560StylesID +
		"%22%2C%22pixel_ratio%22%3A3%2C%22is_push_on%22%3Atrue%2C%22debug_tooling_metadata_token%22%3Anull%2C%22is_flipper_enabled%22%3Afalse%2C%22theme_params%22%3A%5B%7B%22value%22%3A%5B%5D%2C%22design_system_name%22%3A%22FDS%22%7D%5D%2C%22bloks_version%22%3A%22" + S560BloksVer + "%22%7D%7D"
}

// encCP4RegInfo encodes contactpoint cho reg_info (l4 / 7-backslash level).
func encCP4RegInfo(cp, cpType string) string {
	if cpType == "phone" {
		return url.QueryEscape(cp)
	}
	if strings.Contains(cp, "@") {
		parts := strings.SplitN(cp, "@", 2)
		return parts[0] + "%5C%5C%5C%5C%5C%5C%5C%5Cu0040" + parts[1]
	}
	return cp
}

// buildStepBody builds the full POST body string for one step.
func buildStepBody(p stepParams, locale string) string {
	vars := buildStepVars(p)
	parts := []string{
		"method=post",
		"pretty=false",
		"format=json",
		"server_timestamps=true",
		"locale=" + locale,
		"purpose=fetch",
		"fb_api_req_friendly_name=" + p.friendlyName,
		"fb_api_caller_class=graphservice",
		"client_doc_id=" + S560DocID,
		"fb_api_client_context=%7B%22is_background%22%3Afalse%7D",
		"variables=" + vars,
		"fb_api_analytics_tags=%5B%22GraphServices%22%5D",
		"client_trace_id=" + uuid.New().String(),
	}
	return strings.Join(parts, "&")
}

// ─── Per-step CIP builders ────────────────────────────────────────────────────

func cipS1(firstName, lastName string) string {
	// Traffic: firstname, lastname trước common CIP.
	// url.QueryEscape: space → +, phù hợp với traffic "Tran+Nguyen".
	return cq3 + "firstname" + cq3 + "%3A" + cq3 + url.QueryEscape(firstName) + cq3 +
		cip3Sep +
		cq3 + "lastname" + cq3 + "%3A" + cq3 + url.QueryEscape(lastName) + cq3
}

func cipS2(bdayProfile string) string {
	// bdayProfile: "DD-MM-YYYY" — giữ nguyên format cho birthday_or_current_date_string
	// Traffic: "03-03-1990" (DD-MM-YYYY)
	return cip3KvStr("birthday_or_current_date_string", bdayProfile) +
		cip3Sep +
		cip3KvInt("birthday_timestamp", int(bdayToUnix(bdayProfile)))
}

func cipS3(gender int) string {
	return cip3KvInt("gender", gender) +
		cip3Sep +
		cip3KvInt("pronoun", 0)
}

func cipS5Email(email string) string {
	// Traffic: is_from_device_emails, email (key="email"), switch_cp_*
	emailEnc := url.QueryEscape(email) // @ → %40
	emailEnc = strings.ReplaceAll(emailEnc, "+", "%20")
	return strings.Join([]string{
		cip3KvInt("is_from_device_emails", 0),
		cq3 + "email" + cq3 + "%3A" + cq3 + emailEnc + cq3,
		cip3KvInt("switch_cp_first_time_loading", 1),
		cip3KvInt("switch_cp_have_seen_suma", 0),
		cip3KvInt("has_rejected_rel", 0),
		cip3KvInt("seen_login_upsell", 0),
	}, cip3Sep)
}

func cipS5Phone(phone string) string {
	return strings.Join([]string{
		cip3KvInt("is_from_device_emails", 0),
		cq3 + "phone" + cq3 + "%3A" + cq3 + url.QueryEscape(phone) + cq3,
		cip3KvInt("switch_cp_first_time_loading", 1),
		cip3KvInt("switch_cp_have_seen_suma", 0),
	}, cip3Sep)
}

func cipS6Pwd(encPwd string) string {
	// Traffic: encrypted_password ở CIP level (3-backslash)
	return cq3 + "encrypted_password" + cq3 + "%3A" + cq3 + url.QueryEscape(encPwd) + cq3
}
