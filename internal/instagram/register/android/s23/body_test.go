package s23

import (
	"net/url"
	"strings"
	"testing"
)

// TestRegisterBody_Contains177Fields kiểm tra body đã có đủ ~177 fields reg_info
// khớp C# CreateAccountVariables_v3.
func TestRegisterBody_Contains177Fields(t *testing.T) {
	profile := BuildS23Profile("VN")
	profile.FirstName = "Test"
	profile.LastName = "User"
	profile.Birthday = "1990-1-1"
	profile.Gender = 1
	profile.MachineID = "datr_abc"
	profile.MachineID2 = "mid2_xyz"

	body := buildRegisterBody(profile, "#PWD_FB4A:0:123:secret", "0912345678", "phone", "en_US")

	// Body phải có variables= prefix
	if !strings.Contains(body, "variables=") {
		t.Fatal("missing variables= in body")
	}

	// URL-decode 1 level để dễ kiểm
	decoded, err := url.QueryUnescape(body)
	if err != nil {
		t.Fatalf("url decode: %v", err)
	}

	// Các field bắt buộc phải xuất hiện trong reg_info (sample ~30 từ 177)
	mustHave := []string{
		// identity
		`first_name`, `last_name`, `full_name`, `contactpoint`, `contactpoint_type`, `birthday`, `gender`, `encrypted_password`,
		// device
		`device_id`, `family_device_id`, `machine_id`, `waterfall_id`,
		// flags được thêm theo C# V3 (đã miss ở Go cũ):
		`unified_cp_screen_variant`, `birthday_derived_from_age`, `custom_gender`,
		`username`, `username_prefill`, `fb_conf_source`,
		`safetynet_token`, `safetynet_response`, `should_skip_two_step_conf`,
		`openid_tokens_for_testing`, `encrypted_msisdn_for_safetynet`,
		`cached_headers_safetynet_info`, `headers_last_infra_flow_id`,
		`was_headers_prefill_available`, `sso_enabled`, `existing_accounts`,
		`horizon_synced_username`, `fb_access_token`, `horizon_synced_profile_pic`,
		`is_identity_synced`, `is_spectra_reg`, `spectra_reg_token`,
		`xapp_accounts`, `ig_device_id`, `ig_machine_id`, `big_blue_token`,
		`ig_authorization_token`, `full_sheet_flow`, `crypted_user_id`,
		`ignore_suma_check`, `suggested_first_name`, `frl_authorization_token`,
		`existing_account_exact_match_checked`, `existing_account_fuzzy_match_checked`,
		`whatsapp_installed_on_client`, `source_account_type_to_reg_info`,
		`registration_flow_id`, `should_skip_youth_tos`, `email_prefilled`,
		`in_sowa_experiment`, `youth_regulation_config`, `eligible_to_flash_call_in_ig4a`,
		`flash_call_permissions_status`, `attestation_result`,
		`reg_suma_state`, `should_show_spi_before_conf`, `google_oauth_account`,
		`device_emails`, `is_toa_reg`, `is_threads_public`, `spc_import_flow`,
		`caa_play_integrity_attestation_result`, `spc_birthday_input`,
		`fb_suma_combined_landing_candidate_variant`, `screen_visited`,
		`fb_email_login_upsell_skip_suma_post_tos`, `should_prefill_cp_in_ar`,
		`ig_partially_created_account_user_id`, `force_sessionless_nux_experience`,
		`suma_on_conf_threshold`, `welcome_ar_entrypoint`, `th_profile_photo_token`,
		`cp_suma_results_map`, `next_uri`, `linking_entry_point`,
		`fb_encrypted_partial_new_account_properties`, `starter_pack_name`,
		`wa_data_bundle`,
		// server_params top-level
		`offline_experiment_group`, `x_app_device_signals`, `access_flow_version`,
		`current_step`,
		// outer
		`bloks_versioning_id`, `app_id`, `nt_context`, `styles_id`, `theme_params`,
	}
	for _, f := range mustHave {
		if !strings.Contains(decoded, f) {
			t.Errorf("body missing field: %s", f)
		}
	}

	// S23-specific constants phải xuất hiện trực tiếp (không URL-encoded) ở outer wrapper
	if !strings.Contains(body, s23BloksVer) {
		t.Errorf("missing S23 bloks_ver in body")
	}
	// Styles_id S23 swap = "6100e7e89411ccf67ace027cedecd84f"
	if !strings.Contains(body, "6100e7e89411ccf67ace027cedecd84f") {
		t.Errorf("missing S23 styles_id in body")
	}
	// Bloks_ver V1 KHÔNG được xuất hiện (đã bị swap)
	if strings.Contains(body, "0b868a90533a800ff97deb1c85ace5bdbe52f18a1004907dff5d1bbda20b8b2e") {
		t.Errorf("V1 bloks_ver leaked — RegisterFormDataS23 swap không đúng")
	}
	if strings.Contains(body, "5e69aafc13b802e5d3b57b9257525433") {
		t.Errorf("V1 styles_id leaked — S23 swap không đúng")
	}

	// scale phải là "3" (C# hardcode)
	if !strings.Contains(body, `%22scale%22%3A%223%22`) {
		t.Errorf("scale không phải \"3\" — C# hardcode scale=3")
	}

	// client_doc_id phải là V3 docid
	if !strings.Contains(body, "client_doc_id="+s23DocID) {
		t.Errorf("missing s23 docid")
	}
}

// TestXZeroEhBody kiểm tra payload GetXZeroEh khớp C# GetXzeroEhMobileFormData.
func TestXZeroEhBody(t *testing.T) {
	profile := BuildS23Profile("VN")
	profile.Sim.MCC = "452"
	profile.Sim.MNC = "04"
	profile.ConnType = "WIFI"
	profile.Locale = "en_US"
	profile.Sim.CountryCode = "VN"

	body := buildXZeroEhBody(profile)

	// Có prefix + suffix của C#
	if !strings.HasPrefix(body, "batch=") {
		t.Errorf("body không bắt đầu bằng batch=")
	}
	if !strings.Contains(body, "fb_api_caller_class=Fb4aAuthHandler") {
		t.Errorf("missing fb_api_caller_class=Fb4aAuthHandler")
	}
	if !strings.Contains(body, "fb_api_req_friendly_name=fetchLoginData-batch") {
		t.Errorf("missing fetchLoginData-batch friendly name")
	}

	// Inner batch phải có carrier_mcc/sim_mcc/interface theo SIM
	decoded, err := url.QueryUnescape(body)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	wants := []string{
		`"method":"POST"`,
		`"relative_url":"mobile_zero_campaign"`,
		`"name":"fetchZeroToken"`,
		`carrier_mcc=452`, `carrier_mnc=04`,
		`sim_mcc=452`, `sim_mnc=04`,
		`interface=wifi`, // ConnType.ToLower()
		`dialtone_enabled=false`,
		`needs_backup_rules=true`,
		`request_reason=login`,
		`fb_api_req_friendly_name=fetchZeroToken`,
		`locale=en_US`,
		`client_country_code=VN`,
	}
	for _, w := range wants {
		if !strings.Contains(decoded, w) {
			t.Errorf("xzero body missing: %q", w)
		}
	}
}
