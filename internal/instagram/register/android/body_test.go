package android

import (
	"net/url"
	"strings"
	"testing"

	"HVRIns/internal/instagram/fakeinfo"
)

// TestAndroidV3Body_SchemaMatch kiểm tra body có chứa các field V3 thiết yếu.
func TestAndroidV3Body_SchemaMatch(t *testing.T) {
	profile := fakeinfo.BuildFullRegProfile("VN")
	profile.FirstName = "Test"
	profile.LastName = "User"
	profile.Birthday = "1-1-1990"
	profile.Gender = 1
	profile.MachineID2 = "mid2abc"

	body := buildRegisterBody(profile, "#PWD_FB4A:0:123:secret", "0912345678", "phone", "en_US")

	if !strings.Contains(body, "variables=") {
		t.Fatal("missing variables= prefix")
	}
	if !strings.Contains(body, "client_doc_id="+docID) {
		t.Fatalf("missing V3 docid %s", docID)
	}

	decoded, err := url.QueryUnescape(body)
	if err != nil {
		t.Fatalf("unescape: %v", err)
	}

	// V3-specific fields (khác V1 Go cũ)
	v3Fields := []string{
		`age_range`, `os_shared_age_range`, `accounts_list_client`,
		`fdid_available_on_start`, `fdid_rid_available_on_start`,
		`sign_in_with_google_email`, `email_oauth_tokens`, `should_show_rel_error`,
		`eligible_to_mo_sms_in_ig4a`, `mo_sms_ent_id`,
		`gms_incoming_call_retriever_eligibility`,
		`sk_pipa_consent_given`, `caa_play_integrity_attestation_result`,
		`is_in_gms_experience`, `flash_call_nonce_prefix_details`,
		`has_seen_confirmation_screen`, `attempted_silent_auth_in_ig`,
		`sa_prefetch_callback_id`, `bloks_controller_source`,
		`airwave_registration_code`, `is_sessionless_nux`,
		`login_contactpoint`, `is_nta_shortened`, `should_override_back_nav`,
		`ig_footer_variant`, `device_network_info`,
		`is_from_web_lite_reg_controller`, `login_form_siwg_email`,
		`account_setup_waterfall_id`, `is_wanted_suma_user`,
		`device_zero_balance_state`, `should_delay_wa_disclosure`,
		`is_in_nta_single_form`, `source_account_image_asset_id`,
		`passkey_eligible_device`, `nta_single_form_variant`,
		`enable_survey`, `phone_prefetch_outcome`,
		// server_params V3
		`login_surface`, `login_entry_point`, `bloks_controller_source`,
		// client_input_params V3
		`aac`, `network_bssid`, `block_store_machine_id`, `cloud_trust_token`,
		// Common
		`attestation_result`, `safetynet_token`, `keyHash`,
	}
	for _, f := range v3Fields {
		if !strings.Contains(decoded, f) {
			t.Errorf("V3 body missing field: %s", f)
		}
	}

	// V3 constants phải xuất hiện
	if !strings.Contains(body, bloksVer) {
		t.Errorf("missing V3 bloks_ver")
	}
	if !strings.Contains(body, stylesId) {
		t.Errorf("missing V3 styles_id")
	}
	// Không được leak value V1
	if strings.Contains(body, "0b868a90533a800ff97deb1c85ace5bdbe52f18a1004907dff5d1bbda20b8b2e") {
		t.Errorf("V1 bloks_ver leaked")
	}
	if strings.Contains(body, "5e69aafc13b802e5d3b57b9257525433") {
		t.Errorf("V1 styles_id leaked")
	}

	// V22 specific: should_save_password = false (V3 S23 là true).
	// reg_info được embed như string qua nhiều level escape — check sub-pattern.
	if !strings.Contains(decoded, `should_save_password`) {
		t.Errorf("thiếu should_save_password key")
	}
	// Sub-pattern với escape level 4 (reg_info string trong server_params string trong l2 string)
	// Tìm sequence gần should_save_password + false ở sau key (cho phép backslash giữa chừng).
	idx := strings.Index(decoded, "should_save_password")
	if idx >= 0 {
		tail := decoded[idx : idx+100]
		if !strings.Contains(tail, "false") {
			t.Errorf("should_save_password không có false trong 100 ký tự tiếp, tail=%q", tail)
		}
	}
}

// TestAndroidV3_EncryptPassword_FallbackPlaintext kiểm tra nhánh fallback.
func TestAndroidV3_EncryptPassword_FallbackPlaintext(t *testing.T) {
	// Fake public key không hợp lệ → EncryptPassword trả ""
	got := EncryptPassword("mypass", "not-a-valid-key", 1)
	if got != "" {
		t.Errorf("invalid key should return empty, got %q", got)
	}
}

// TestAndroidV3_USDIDFormat kiểm tra format {uuid}.{ts}.{base64url}.
func TestAndroidV3_USDIDFormat(t *testing.T) {
	usdid := generateUSDIDV3()
	parts := strings.Split(usdid, ".")
	if len(parts) != 3 {
		t.Fatalf("USDID phải có 3 phần tách bởi '.', got %d: %q", len(parts), usdid)
	}
	// Part 0 = UUID format 8-4-4-4-12 (với dashes)
	if strings.Count(parts[0], "-") != 4 {
		t.Errorf("USDID part 0 không phải UUID: %q", parts[0])
	}
	// Part 2 = base64url → không có '+' '/' '='
	if strings.ContainsAny(parts[2], "+/=") {
		t.Errorf("USDID signature phải là base64url (no +/=): %q", parts[2])
	}
}

// TestAndroidV3_ConnUUIDClient kiểm tra format base64 16 bytes.
func TestAndroidV3_ConnUUIDClient(t *testing.T) {
	v := generateConnUUIDClientV3()
	// base64 16 bytes = 24 chars (với padding)
	if len(v) != 24 {
		t.Errorf("conn uuid client len = %d, expect 24", len(v))
	}
}
