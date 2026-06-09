// body.go — Build request body cho từng bước verify B1-B5 + Resend
// EXACT copy từ WeBM WemakeFacebook.Func.12.Verify.cs
// Dùng raw body templates, thay variables bằng strings.NewReplacer
package web

import (
	"fmt"
	"net/url"
	"strings"

	"HVRIns/internal/instagram"
)

// __bkv (Bloks Versioning Key) — captured 2026-05-18 từ browser thật.
// FB cập nhật mỗi vài tuần — nếu verify mfb bắt đầu fail nhiều, capture lại từ DevTools.
// Pattern: Network tab → request đến /async/wbloks/fetch/ → query string `__bkv=...`.
const bkv = "db8794f359300232099196cee680aa6e45d8b2f38cc437d220dd6ddc93feb992"

// Endpoints — exact từ WeBM
var (
	EndpointB1     = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.confirmation.fb.bottomsheet&type=app&__bkv=" + bkv
	EndpointB2     = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.confirmation.change.email&type=app&__bkv=" + bkv
	EndpointB3     = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.async.contactpoint_email.async&type=action&__bkv=" + bkv
	EndpointB4     = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.confirmation&type=app&__bkv=" + bkv
	EndpointB5     = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.confirmation.async&type=action&__bkv=" + bkv
	EndpointResend = "https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.reg.resend_confirmation.async&type=action&__bkv=" + bkv
	EndpointLang   = "https://m.facebook.com/intl/ajax/save_locale/"
)

// Referer headers — exact từ WeBM
var (
	RefererB1   = "https://m.facebook.com/confirmemail.php"
	RefererB2   = "https://m.facebook.com/confirmemail.php?next=https%3A%2F%2Fwww.facebook.com%2Flogin"
	RefererLang = "https://m.facebook.com/language.php"
)

// refererB5 tạo Referer header đặc biệt cho step B5 (submit OTP xác nhận email).
//
// B5 là bước submit OTP cuối cùng trong luồng đăng ký, Facebook yêu cầu Referer
// trỏ đúng đến trang confirmation với reg_info chứa contactpoint (email) và
// contactpoint_type=email. Nếu Referer sai hoặc thiếu, server sẽ từ chối request
// hoặc trả về lỗi CSRF. Các bước B1-B4 dùng Referer tĩnh (RefererB1/RefererB2),
// riêng B5 cần Referer động vì contactpoint chứa địa chỉ email cụ thể của user.
//
// emailUser: phần trước @ của địa chỉ email đã đăng ký (ví dụ "john.doe" trong "john.doe@gmail.com").
// emailDomain: phần sau @ của địa chỉ email (ví dụ "gmail.com").
// Trả về URL đầy đủ đến /caa/reg/confirmation/ với reg_info và flow_info được URL-encode kép.
func refererB5(emailUser, emailDomain string) string {
	encoded := url.QueryEscape(emailUser + "\u0040" + emailDomain)
	return fmt.Sprintf("https://m.facebook.com/caa/reg/confirmation/?reg_info=%%7B%%22contactpoint%%22%%3A%%22%s%%22%%2C%%22contactpoint_type%%22%%3A%%22email%%22%%2C%%22is_cp_auto_confirmed%%22%%3Afalse%%2C%%22fb_conf_source%%22%%3Anull%%2C%%22confirmation_medium%%22%%3Anull%%2C%%22registration_flow_id%%22%%3A%%22%%22%%7D&flow_info=%%7B%%22flow_name%%22%%3A%%22new_to_family_fb_default%%22%%2C%%22flow_type%%22%%3A%%22ntf%%22%%7D&current_step=10", encoded)
}

// reg_info template — 200+ fields giống WeBM byte-for-byte
// Phần này dùng chung cho B1, B2, B3, B4, B5, Resend
// Chỉ khác nhau ở contactpoint/contactpoint_type/device_id/user_id
const regInfoPhone = `%5C%5C%5C%22first_name%5C%5C%5C%22%3A%5C%5C%5C%22{FIRST}%5C%5C%5C%22%2C%5C%5C%5C%22last_name%5C%5C%5C%22%3A%5C%5C%5C%22{LAST}%5C%5C%5C%22%2C%5C%5C%5C%22full_name%5C%5C%5C%22%3A%5C%5C%5C%22{FIRST}+{LAST}%5C%5C%5C%22%2C%5C%5C%5C%22contactpoint%5C%5C%5C%22%3A%5C%5C%5C%22%2B{PHONE}%5C%5C%5C%22%2C%5C%5C%5C%22ar_contactpoint%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22contactpoint_type%5C%5C%5C%22%3A%5C%5C%5C%22phone%5C%5C%5C%22%2C%5C%5C%5C%22is_using_unified_cp%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22unified_cp_screen_variant%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_cp_auto_confirmed%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_cp_auto_confirmable%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_cp_claimed%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22confirmation_code%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22birthday%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22birthday_derived_from_age%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22age_range%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22did_use_age%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22os_shared_age_range%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22gender%5C%5C%5C%22%3A1%2C%5C%5C%5C%22use_custom_gender%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22custom_gender%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22encrypted_password%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22username%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22username_prefill%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22accounts_list_client%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22fb_conf_source%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22device_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ig4a_qe_device_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22family_device_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22user_id%5C%5C%5C%22%3A%5C%5C%5C%22{UID}%5C%5C%5C%22%2C%5C%5C%5C%22safetynet_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22skip_slow_rel_check%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22safetynet_response%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22machine_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22profile_photo%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22profile_photo_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22profile_photo_upload_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22avatar%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22email_oauth_token_no_contact_perm%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22email_oauth_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22email_oauth_tokens%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22sign_in_with_google_email%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_skip_two_step_conf%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22openid_tokens_for_testing%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22encrypted_msisdn%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22encrypted_msisdn_for_safetynet%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22cached_headers_safetynet_info%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_skip_headers_safetynet%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22headers_last_infra_flow_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22headers_last_infra_flow_id_safetynet%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22headers_flow_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22was_headers_prefill_available%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22sso_enabled%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22existing_accounts%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22used_ig_birthday%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22create_new_to_app_account%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22skip_session_info%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ck_error%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ck_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ck_nonce%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_save_password%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22fb_access_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_msplit_reg%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_spectra_reg%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22dema_account_consent_given%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22spectra_reg_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22spectra_reg_guardian_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22spectra_reg_guardian_logged_in_context%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22user_id_of_msplit_creator%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22msplit_creator_nonce%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22dma_data_combination_consent_given%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22xapp_accounts%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22fb_device_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22fb_machine_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ig_device_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ig_machine_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_skip_nta_upsell%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22big_blue_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22caa_reg_flow_source%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ig_authorization_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22full_sheet_flow%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22crypted_user_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_caa_perf_enabled%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_preform%5C%5C%5C%22%3Atrue%2C%5C%5C%5C%22should_show_rel_error%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22ignore_suma_check%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22dismissed_login_upsell_with_cna%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22ignore_existing_login%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22ignore_existing_login_from_suma%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22ignore_existing_login_after_errors%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22suggested_first_name%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22suggested_last_name%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22suggested_full_name%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22frl_authorization_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22post_form_errors%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22skip_step_without_errors%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22existing_account_exact_match_checked%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22existing_account_fuzzy_match_checked%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22email_oauth_exists%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22confirmation_code_send_error%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_too_young%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22source_account_type%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22whatsapp_installed_on_client%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22confirmation_medium%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22source_credentials_type%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22source_cuid%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22source_account_reg_info%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22soap_creation_source%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22source_account_type_to_reg_info%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22registration_flow_id%5C%5C%5C%22%3A%5C%5C%5C%22%5C%5C%5C%22%2C%5C%5C%5C%22should_skip_youth_tos%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_youth_regulation_flow_complete%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_on_cold_start%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22email_prefilled%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22cp_confirmed_by_auto_conf%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22in_sowa_experiment%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22youth_regulation_config%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22conf_allow_back_nav_after_change_cp%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22conf_bouncing_cliff_screen_type%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22conf_show_bouncing_cliff%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22eligible_to_flash_call_in_ig4a%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22eligible_to_mo_sms_in_ig4a%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22mo_sms_ent_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22flash_call_permissions_status%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22gms_incoming_call_retriever_eligibility%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22attestation_result%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22request_data_and_challenge_nonce_string%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22confirmed_cp_and_code%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22notification_callback_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22reg_suma_state%5C%5C%5C%22%3A0%2C%5C%5C%5C%22is_msplit_neutral_choice%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22msg_previous_cp%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ntp_import_source_info%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22youth_consent_decision_time%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22sk_pipa_consent_given%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_show_spi_before_conf%5C%5C%5C%22%3Atrue%2C%5C%5C%5C%22google_oauth_account%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_reg_request_from_ig_suma%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_toa_reg%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_threads_public%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22spc_import_flow%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22caa_play_integrity_attestation_result%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22client_known_key_hash%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22flash_call_provider%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_in_gms_experience%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22flash_call_nonce_prefix_details%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22spc_birthday_input%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22failed_birthday_year_count%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22user_presented_medium_source%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22user_opted_out_of_ntp%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_from_registration_reminder%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22show_youth_reg_in_ig_spc%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22fb_suma_is_high_confidence%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22screen_visited%5C%5C%5C%22%3A%5B%5C%5C%5C%22CAA_REG_CONFIRMATION_SCREEN%5C%5C%5C%22%5D%2C%5C%5C%5C%22fb_email_login_upsell_skip_suma_post_tos%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22fb_suma_is_from_email_login_upsell%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22fb_suma_is_from_phone_login_upsell%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22should_prefill_cp_in_ar%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ig_partially_created_account_user_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ig_partially_created_account_nonce%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ig_partially_created_account_nonce_expiry%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22force_sessionless_nux_experience%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22has_seen_suma_landing_page_pre_conf%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22has_seen_suma_candidate_page_pre_conf%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22has_seen_confirmation_screen%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22suma_on_conf_threshold%5C%5C%5C%22%3A-1%2C%5C%5C%5C%22pp_to_nux_eligible%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22should_show_error_msg%5C%5C%5C%22%3Atrue%2C%5C%5C%5C%22th_profile_photo_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22attempted_silent_auth_in_fb%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22attempted_silent_auth_in_ig%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22cp_suma_results_map%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22source_username%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22next_uri%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_use_next_uri%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22linking_entry_point%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22fb_encrypted_partial_new_account_properties%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22starter_pack_name%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22starter_pack_creator_user_ids%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22wa_data_bundle%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22bloks_controller_source%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22airwave_registration_code%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_sessionless_nux%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22login_contactpoint%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22login_contactpoint_type%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_nta_shortened%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22should_show_bday_after_name_suggestions%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_override_back_nav%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22ig_footer_variant%5C%5C%5C%22%3A%5C%5C%5C%22control%5C%5C%5C%22%2C%5C%5C%5C%22device_network_info%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_from_web_lite_reg_controller%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22login_form_siwg_email%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22account_setup_waterfall_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_wanted_suma_user%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22device_zero_balance_state%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_delay_wa_disclosure%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_in_nta_single_form%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22source_account_image_asset_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22passkey_eligible_device%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22nta_single_form_variant%5C%5C%5C%22%3Anull`

// regInfoEmail — dùng cho B4, B5, Resend khi contactpoint đã chuyển sang email
// Khác regInfoPhone ở: contactpoint={emailUser}\u0040{emailDomain}, contactpoint_type=email, device_id={datr}
const regInfoEmailTemplate = `%5C%5C%5C%22first_name%5C%5C%5C%22%3A%5C%5C%5C%22{FIRST}%5C%5C%5C%22%2C%5C%5C%5C%22last_name%5C%5C%5C%22%3A%5C%5C%5C%22{LAST}%5C%5C%5C%22%2C%5C%5C%5C%22full_name%5C%5C%5C%22%3A%5C%5C%5C%22{FIRST}+{LAST}%5C%5C%5C%22%2C%5C%5C%5C%22contactpoint%5C%5C%5C%22%3A%5C%5C%5C%22{EMAILUSER}%5C%5C%5C%5Cu0040{EMAILDOMAIN}%5C%5C%5C%22%2C%5C%5C%5C%22ar_contactpoint%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22contactpoint_type%5C%5C%5C%22%3A%5C%5C%5C%22email%5C%5C%5C%22%2C%5C%5C%5C%22is_using_unified_cp%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22unified_cp_screen_variant%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_cp_auto_confirmed%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_cp_auto_confirmable%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_cp_claimed%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22confirmation_code%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22birthday%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22birthday_derived_from_age%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22age_range%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22did_use_age%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22os_shared_age_range%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22gender%5C%5C%5C%22%3A1%2C%5C%5C%5C%22use_custom_gender%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22custom_gender%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22encrypted_password%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22username%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22username_prefill%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22accounts_list_client%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22fb_conf_source%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22device_id%5C%5C%5C%22%3A%5C%5C%5C%22{DATR}%5C%5C%5C%22%2C%5C%5C%5C%22ig4a_qe_device_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22family_device_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22user_id%5C%5C%5C%22%3A%5C%5C%5C%22{UID}%5C%5C%5C%22%2C%5C%5C%5C%22safetynet_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22skip_slow_rel_check%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22safetynet_response%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22machine_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22profile_photo%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22profile_photo_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22profile_photo_upload_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22avatar%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22email_oauth_token_no_contact_perm%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22email_oauth_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22email_oauth_tokens%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22sign_in_with_google_email%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_skip_two_step_conf%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22openid_tokens_for_testing%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22encrypted_msisdn%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22encrypted_msisdn_for_safetynet%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22cached_headers_safetynet_info%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_skip_headers_safetynet%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22headers_last_infra_flow_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22headers_last_infra_flow_id_safetynet%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22headers_flow_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22was_headers_prefill_available%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22sso_enabled%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22existing_accounts%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22used_ig_birthday%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22create_new_to_app_account%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22skip_session_info%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ck_error%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ck_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ck_nonce%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_save_password%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22fb_access_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_msplit_reg%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_spectra_reg%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22dema_account_consent_given%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22spectra_reg_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22spectra_reg_guardian_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22spectra_reg_guardian_logged_in_context%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22user_id_of_msplit_creator%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22msplit_creator_nonce%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22dma_data_combination_consent_given%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22xapp_accounts%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22fb_device_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22fb_machine_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ig_device_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ig_machine_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_skip_nta_upsell%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22big_blue_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22caa_reg_flow_source%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ig_authorization_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22full_sheet_flow%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22crypted_user_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_caa_perf_enabled%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_preform%5C%5C%5C%22%3Atrue%2C%5C%5C%5C%22should_show_rel_error%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22ignore_suma_check%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22dismissed_login_upsell_with_cna%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22ignore_existing_login%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22ignore_existing_login_from_suma%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22ignore_existing_login_after_errors%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22suggested_first_name%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22suggested_last_name%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22suggested_full_name%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22frl_authorization_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22post_form_errors%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22skip_step_without_errors%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22existing_account_exact_match_checked%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22existing_account_fuzzy_match_checked%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22email_oauth_exists%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22confirmation_code_send_error%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_too_young%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22source_account_type%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22whatsapp_installed_on_client%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22confirmation_medium%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22source_credentials_type%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22source_cuid%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22source_account_reg_info%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22soap_creation_source%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22source_account_type_to_reg_info%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22registration_flow_id%5C%5C%5C%22%3A%5C%5C%5C%22%5C%5C%5C%22%2C%5C%5C%5C%22should_skip_youth_tos%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_youth_regulation_flow_complete%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_on_cold_start%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22email_prefilled%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22cp_confirmed_by_auto_conf%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22in_sowa_experiment%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22youth_regulation_config%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22conf_allow_back_nav_after_change_cp%5C%5C%5C%22%3Atrue%2C%5C%5C%5C%22conf_bouncing_cliff_screen_type%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22conf_show_bouncing_cliff%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22eligible_to_flash_call_in_ig4a%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22eligible_to_mo_sms_in_ig4a%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22mo_sms_ent_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22flash_call_permissions_status%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22gms_incoming_call_retriever_eligibility%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22attestation_result%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22request_data_and_challenge_nonce_string%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22confirmed_cp_and_code%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22notification_callback_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22reg_suma_state%5C%5C%5C%22%3A0%2C%5C%5C%5C%22is_msplit_neutral_choice%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22msg_previous_cp%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ntp_import_source_info%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22youth_consent_decision_time%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22sk_pipa_consent_given%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_show_spi_before_conf%5C%5C%5C%22%3Atrue%2C%5C%5C%5C%22google_oauth_account%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_reg_request_from_ig_suma%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_toa_reg%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_threads_public%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22spc_import_flow%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22caa_play_integrity_attestation_result%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22client_known_key_hash%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22flash_call_provider%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_in_gms_experience%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22flash_call_nonce_prefix_details%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22spc_birthday_input%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22failed_birthday_year_count%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22user_presented_medium_source%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22user_opted_out_of_ntp%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_from_registration_reminder%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22show_youth_reg_in_ig_spc%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22fb_suma_is_high_confidence%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22screen_visited%5C%5C%5C%22%3A%5B%5C%5C%5C%22CAA_REG_CONFIRMATION_SCREEN%5C%5C%5C%22%2C%5C%5C%5C%22CAA_REG_CONFIRMATION_SCREEN%5C%5C%5C%22%5D%2C%5C%5C%5C%22fb_email_login_upsell_skip_suma_post_tos%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22fb_suma_is_from_email_login_upsell%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22fb_suma_is_from_phone_login_upsell%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22should_prefill_cp_in_ar%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ig_partially_created_account_user_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ig_partially_created_account_nonce%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22ig_partially_created_account_nonce_expiry%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22force_sessionless_nux_experience%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22has_seen_suma_landing_page_pre_conf%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22has_seen_suma_candidate_page_pre_conf%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22has_seen_confirmation_screen%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22suma_on_conf_threshold%5C%5C%5C%22%3A-1%2C%5C%5C%5C%22pp_to_nux_eligible%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22should_show_error_msg%5C%5C%5C%22%3Atrue%2C%5C%5C%5C%22th_profile_photo_token%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22attempted_silent_auth_in_fb%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22attempted_silent_auth_in_ig%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22cp_suma_results_map%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22source_username%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22next_uri%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_use_next_uri%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22linking_entry_point%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22fb_encrypted_partial_new_account_properties%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22starter_pack_name%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22starter_pack_creator_user_ids%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22wa_data_bundle%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22bloks_controller_source%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22airwave_registration_code%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_sessionless_nux%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22login_contactpoint%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22login_contactpoint_type%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_nta_shortened%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22should_show_bday_after_name_suggestions%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_override_back_nav%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22ig_footer_variant%5C%5C%5C%22%3A%5C%5C%5C%22control%5C%5C%5C%22%2C%5C%5C%5C%22device_network_info%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_from_web_lite_reg_controller%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22login_form_siwg_email%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22account_setup_waterfall_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22is_wanted_suma_user%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22device_zero_balance_state%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22should_delay_wa_disclosure%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22is_in_nta_single_form%5C%5C%5C%22%3Afalse%2C%5C%5C%5C%22source_account_image_asset_id%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22passkey_eligible_device%5C%5C%5C%22%3Anull%2C%5C%5C%5C%22nta_single_form_variant%5C%5C%5C%22%3Anull`

// regInfoEmailB4Template — giống regInfoEmailTemplate nhưng screen_visited chỉ 1 entry (WeBM B4 line 662)
// Khác regInfoEmailTemplate ở: screen_visited=["CAA_REG_CONFIRMATION_SCREEN"] thay vì 2 entries
var regInfoEmailB4Template = strings.Replace(
	regInfoEmailTemplate,
	`%5C%5C%5C%22CAA_REG_CONFIRMATION_SCREEN%5C%5C%5C%22%2C%5C%5C%5C%22CAA_REG_CONFIRMATION_SCREEN%5C%5C%5C%22`,
	`%5C%5C%5C%22CAA_REG_CONFIRMATION_SCREEN%5C%5C%5C%22`,
	1,
)

// replaceVars thay thế tất cả biến placeholder trong chuỗi template bằng giá trị từ map vars.
//
// Hàm này là cốt lõi để build body string cho mỗi bước B1-B5 và Resend:
// thay vì concatenate string thủ công, các template lớn (regInfoPhone, regInfoEmailTemplate)
// chứa placeholder dạng {KEY} và hàm này thực hiện thay thế hàng loạt một lần duy nhất
// thông qua strings.NewReplacer để tránh lỗi thiếu/sót biến và tối ưu hiệu năng.
//
// template: chuỗi raw body chứa placeholder dạng {KEY} (ví dụ {FIRST}, {UID}, {DATR}).
// vars: map[string]string ánh xạ tên placeholder (không có dấu {}) sang giá trị thực tế.
// Trả về chuỗi đã được thay thế toàn bộ placeholder.
func replaceVars(template string, vars map[string]string) string {
	pairs := make([]string, 0, len(vars)*2)
	for k, v := range vars {
		pairs = append(pairs, "{"+k+"}", v)
	}
	return strings.NewReplacer(pairs...).Replace(template)
}

// vrfy_buildRegInfoPhone tạo chuỗi reg_info dùng cho các bước B1, B2, B3 (contactpoint là số điện thoại).
//
// Trong giai đoạn đầu của luồng xác nhận (B1-B3), Facebook vẫn nhận diện tài khoản qua
// số điện thoại gốc (contactpoint_type=phone). Hàm này dùng template regInfoPhone —
// chứa toàn bộ 200+ fields theo WeBM — và thay thế 4 biến cần thiết:
//
// s: Facebook session chứa s.Phone (số điện thoại không có dấu +) và s.UID (user ID).
// firstName: tên đăng ký của tài khoản, điền vào first_name và full_name trong reg_info.
// lastName: họ đăng ký của tài khoản, điền vào last_name và full_name trong reg_info.
// Trả về chuỗi reg_info đã URL-encode kép, sẵn sàng nhúng vào body params của B1/B2/B3.
func vrfy_buildRegInfoPhone(s *instagram.Session, firstName, lastName string) string {
	return replaceVars(regInfoPhone, map[string]string{
		"FIRST": firstName, "LAST": lastName,
		"PHONE": s.Phone, "UID": s.UID,
	})
}

// vrfy_buildRegInfoEmail tạo chuỗi reg_info cho các bước B5 và Resend (contactpoint đã chuyển sang email).
//
// Khác vrfy_buildRegInfoPhone ở 3 điểm chính:
//   1. contactpoint="{emailUser}\u0040{emailDomain}" thay vì số điện thoại.
//   2. contactpoint_type="email" thay vì "phone".
//   3. device_id="{DATR}" (cookie datr từ session) thay vì null — Facebook dùng datr
//      để liên kết thiết bị với session email confirmation.
//
// Dùng template regInfoEmailTemplate với screen_visited chứa 2 entries (đã qua B4 lần nữa).
//
// s: Facebook session chứa s.Datr (cookie datr) và s.UID.
// firstName, lastName: tên đăng ký.
// emailUser: phần trước @ của email (ví dụ "john.doe").
// emailDomain: phần sau @ của email (ví dụ "gmail.com").
// Trả về chuỗi reg_info URL-encode kép dùng cho B5 và Resend.
func vrfy_buildRegInfoEmail(s *instagram.Session, firstName, lastName, emailUser, emailDomain string) string {
	return replaceVars(regInfoEmailTemplate, map[string]string{
		"FIRST": firstName, "LAST": lastName,
		"EMAILUSER": emailUser, "EMAILDOMAIN": emailDomain,
		"DATR": s.Datr, "UID": s.UID,
	})
}

// vrfy_buildRegInfoEmailB4 tạo chuỗi reg_info cho bước B4 (load màn hình xác nhận email lần đầu).
//
// Giống vrfy_buildRegInfoEmail (contactpoint=email, contactpoint_type=email, device_id=datr)
// nhưng khác ở trường screen_visited:
//   - vrfy_buildRegInfoEmail (dùng cho B5/Resend): screen_visited=["CAA_REG_CONFIRMATION_SCREEN","CAA_REG_CONFIRMATION_SCREEN"]
//     (2 entries, user đã thấy màn hình confirmation ít nhất 2 lần).
//   - vrfy_buildRegInfoEmailB4 (dùng cho B4): screen_visited=["CAA_REG_CONFIRMATION_SCREEN"]
//     (1 entry, user mới vào màn hình confirmation lần đầu theo WeBM line 662).
//
// Facebook dùng screen_visited để theo dõi lịch sử navigation và điều chỉnh UX
// (ví dụ nút back). Gửi sai số entries có thể khiến server trả về UI state không đúng.
//
// s: Facebook session chứa s.Datr và s.UID.
// firstName, lastName: tên đăng ký.
// emailUser: phần trước @ của email.
// emailDomain: phần sau @ của email.
// Trả về chuỗi reg_info URL-encode kép dùng riêng cho B4.
func vrfy_buildRegInfoEmailB4(s *instagram.Session, firstName, lastName, emailUser, emailDomain string) string {
	return replaceVars(regInfoEmailB4Template, map[string]string{
		"FIRST": firstName, "LAST": lastName,
		"EMAILUSER": emailUser, "EMAILDOMAIN": emailDomain,
		"DATR": s.Datr, "UID": s.UID,
	})
}

// vrfy_buildBaseParams tạo phần prefix chung của request body cho các bước B1-B5.
//
// Prefix này chứa các meta-param Facebook dùng để xác thực request và định danh client:
// __user, __a, __req, __hs, dpr, __ccg, __rev, __s, __hsi, __dyn, fb_dtsg, jazoest, lsd.
// Mỗi bước B1-B5 gọi hàm này rồi nối thêm "&params=..." vào cuối.
//
// s: Facebook session chứa toàn bộ token/cookie cần thiết (UID, Rev, S, Hsi, FbDtsg, Jazoest, Lsd).
// reqNum: số thứ tự request trong chuỗi (sequence number), truyền vào __req.
//
//	Các bước dùng: B1="2", B2="4", B3="5", B4/B5="6". Phải tăng dần theo thứ tự
//	WeBM để server không reject do phát hiện request out-of-order.
//
// hsVer: phiên bản header signature, truyền vào __hs (ví dụ "20538" cho B1, "20539" cho B2-B5).
//
//	Khác nhau giữa các bước theo đúng WeBM; sai giá trị có thể bị từ chối.
//
// Trả về chuỗi URL-encoded sẵn sàng nối với "&params=...".
func vrfy_buildBaseParams(s *instagram.Session, reqNum, hsVer string) string {
	return fmt.Sprintf("__user=%s&__a=1&__req=%s&__hs=%s.BP%%3Amtouch_pkg.2.0...0&dpr=3&__ccg=EXCELLENT&__rev=%s&__s=%s&__hsi=%s&__dyn=&fb_dtsg=%s&jazoest=%s&lsd=%s",
		s.UID, reqNum, hsVer, s.Rev, s.S, s.Hsi, s.FbDtsg, s.Jazoest, s.Lsd)
}

// ============================================================
// Build body functions cho từng step
// ============================================================

// vrfy_buildB1Body tạo request body cho step B1 — xác nhận email đăng ký (phone → email redirect).
// s: Facebook session chứa token, UID, datr và các param request.
// waterfallId: UUID định danh luồng đăng ký hiện tại.
// firstName, lastName: tên đăng ký của tài khoản mới.
// Trả về chuỗi body URL-encoded, gửi đến EndpointB1.
func vrfy_buildB1Body(s *instagram.Session, waterfallId, firstName, lastName string) string {
	regInfo := vrfy_buildRegInfoPhone(s, firstName, lastName)
	// EXACT từ WeBM line 316: sau reg_info có flow_info, current_step, INTERNAL_INFRA_screen_id
	params := `%7B%22params%22%3A%22%7B%5C%22server_params%5C%22%3A%7B%5C%22waterfall_id%5C%22%3A%5C%22` + waterfallId + `%5C%22%2C%5C%22is_platform_login%5C%22%3A0%2C%5C%22is_from_logged_out%5C%22%3A0%2C%5C%22access_flow_version%5C%22%3A%5C%22pre_mt_behavior%5C%22%2C%5C%22trigger%5C%22%3A%5C%22default%5C%22%2C%5C%22timer_id%5C%22%3A%5C%22wa_retriever%5C%22%2C%5C%22zero_tap_enabled%5C%22%3A0%2C%5C%22reg_info%5C%22%3A%5C%22%7B` + regInfo + `%7D%5C%22%2C%5C%22flow_info%5C%22%3A%5C%22%7B%5C%5C%5C%22flow_name%5C%5C%5C%22%3A%5C%5C%5C%22new_to_family_fb_default%5C%5C%5C%22%2C%5C%5C%5C%22flow_type%5C%5C%5C%22%3A%5C%5C%5C%22ntf%5C%5C%5C%22%7D%5C%22%2C%5C%22current_step%5C%22%3A10%2C%5C%22INTERNAL_INFRA_screen_id%5C%22%3A%5C%224qihkx%3A264%5C%22%7D%2C%5C%22client_input_params%5C%22%3A%7B%5C%22lois_settings%5C%22%3A%7B%5C%22lois_token%5C%22%3A%5C%22%5C%22%7D%2C%5C%22aac%5C%22%3A%5C%22%5C%22%7D%7D%22%7D`
	return vrfy_buildBaseParams(s, "2", "20538") + "&params=" + params
}

// vrfy_buildB2Body tạo request body cho step B2 — chuyển contactpoint sang email.
// s: Facebook session.
// waterfallId: UUID định danh luồng đăng ký.
// firstName, lastName: tên đăng ký.
// Trả về body URL-encoded gửi đến EndpointB2, INTERNAL_INFRA_screen_id=CAA_REG_CONFIRMATION_CHANGE_EMAIL.
func vrfy_buildB2Body(s *instagram.Session, waterfallId, firstName, lastName string) string {
	regInfo := vrfy_buildRegInfoPhone(s, firstName, lastName)
	params := `%7B%22params%22%3A%22%7B%5C%22server_params%5C%22%3A%7B%5C%22waterfall_id%5C%22%3A%5C%22` + waterfallId + `%5C%22%2C%5C%22is_platform_login%5C%22%3A0%2C%5C%22is_from_logged_out%5C%22%3A0%2C%5C%22access_flow_version%5C%22%3A%5C%22pre_mt_behavior%5C%22%2C%5C%22reg_info%5C%22%3A%5C%22%7B` + regInfo + `%7D%5C%22%2C%5C%22flow_info%5C%22%3A%5C%22%7B%5C%5C%5C%22flow_name%5C%5C%5C%22%3A%5C%5C%5C%22new_to_family_fb_default%5C%5C%5C%22%2C%5C%5C%5C%22flow_type%5C%5C%5C%22%3A%5C%5C%5C%22ntf%5C%5C%5C%22%7D%5C%22%2C%5C%22current_step%5C%22%3A10%2C%5C%22INTERNAL_INFRA_screen_id%5C%22%3A%5C%22CAA_REG_CONFIRMATION_CHANGE_EMAIL%5C%22%7D%2C%5C%22client_input_params%5C%22%3A%7B%5C%22lois_settings%5C%22%3A%7B%5C%22lois_token%5C%22%3A%5C%22%5C%22%7D%2C%5C%22aac%5C%22%3A%5C%22%5C%22%7D%7D%22%7D`
	return vrfy_buildBaseParams(s, "4", "20539") + "&params=" + params
}

// vrfy_buildB3Body tạo request body cho step B3 — submit địa chỉ email vào form đăng ký.
// s: Facebook session.
// waterfallId: UUID định danh luồng đăng ký.
// firstName, lastName: tên đăng ký.
// emailAddr: địa chỉ email đã tạo (dạng user@domain), được URL-encode trong params.
// eventRequestId: UUID riêng cho event này, truyền vào server_params.
// Trả về body URL-encoded gửi đến EndpointB3.
func vrfy_buildB3Body(s *instagram.Session, waterfallId, firstName, lastName, emailAddr, eventRequestId string) string {
	encodedEmail := url.QueryEscape(emailAddr)
	regInfo := vrfy_buildRegInfoPhone(s, firstName, lastName)
	// WeBM B3: conf_allow_back_nav_after_change_cp = true (không phải null như B1/B2)
	regInfo = strings.Replace(regInfo,
		`%5C%5C%5C%22conf_allow_back_nav_after_change_cp%5C%5C%5C%22%3Anull`,
		`%5C%5C%5C%22conf_allow_back_nav_after_change_cp%5C%5C%5C%22%3Atrue`,
		1)
	// WeBM B3: server_params co day du flow_info, current_step, INTERNAL__latency_qpl_*, device_id, waterfall_id, is_platform_login, access_flow_version, login_surface
	// client_input_params co device_id, family_device_id, cloud_trust_token, accounts_list, fb_ig_device_id, confirmed_cp_and_code, switch_cp_first_time_loading, v.v.
	params := `%7B%22params%22%3A%22%7B%5C%22server_params%5C%22%3A%7B%5C%22event_request_id%5C%22%3A%5C%22` + eventRequestId + `%5C%22%2C%5C%22cp_funnel%5C%22%3A1%2C%5C%22cp_source%5C%22%3A1%2C%5C%22text_input_id%5C%22%3A%5C%2251111625100062%5C%22%2C%5C%22reg_info%5C%22%3A%5C%22%7B` + regInfo + `%7D%5C%22%2C%5C%22flow_info%5C%22%3A%5C%22%7B%5C%5C%5C%22flow_name%5C%5C%5C%22%3A%5C%5C%5C%22new_to_family_fb_default%5C%5C%5C%22%2C%5C%5C%5C%22flow_type%5C%5C%5C%22%3A%5C%5C%5C%22ntf%5C%5C%5C%22%7D%5C%22%2C%5C%22current_step%5C%22%3A10%2C%5C%22INTERNAL__latency_qpl_marker_id%5C%22%3A36707139%2C%5C%22INTERNAL__latency_qpl_instance_id%5C%22%3A%5C%2251111625100108%5C%22%2C%5C%22device_id%5C%22%3Anull%2C%5C%22family_device_id%5C%22%3Anull%2C%5C%22waterfall_id%5C%22%3A%5C%22` + waterfallId + `%5C%22%2C%5C%22offline_experiment_group%5C%22%3Anull%2C%5C%22layered_homepage_experiment_group%5C%22%3Anull%2C%5C%22is_platform_login%5C%22%3A0%2C%5C%22is_from_logged_in_switcher%5C%22%3A0%2C%5C%22is_from_logged_out%5C%22%3A0%2C%5C%22access_flow_version%5C%22%3A%5C%22pre_mt_behavior%5C%22%2C%5C%22login_surface%5C%22%3A%5C%22unknown%5C%22%7D%2C%5C%22client_input_params%5C%22%3A%7B%5C%22device_id%5C%22%3A%5C%22%5C%22%2C%5C%22family_device_id%5C%22%3A%5C%22%5C%22%2C%5C%22cloud_trust_token%5C%22%3Anull%2C%5C%22block_store_machine_id%5C%22%3A%5C%22%5C%22%2C%5C%22zero_balance_state%5C%22%3A%5C%22%5C%22%2C%5C%22email%5C%22%3A%5C%22` + encodedEmail + `%5C%22%2C%5C%22email_prefilled%5C%22%3A0%2C%5C%22accounts_list%5C%22%3A%5B%5D%2C%5C%22fb_ig_device_id%5C%22%3A%5B%5D%2C%5C%22confirmed_cp_and_code%5C%22%3A%7B%7D%2C%5C%22is_from_device_emails%5C%22%3A0%2C%5C%22msg_previous_cp%5C%22%3A%5C%22%5C%22%2C%5C%22switch_cp_first_time_loading%5C%22%3A1%2C%5C%22switch_cp_have_seen_suma%5C%22%3A0%2C%5C%22network_bssid%5C%22%3Anull%2C%5C%22lois_settings%5C%22%3A%7B%5C%22lois_token%5C%22%3A%5C%22%5C%22%7D%2C%5C%22aac%5C%22%3A%5C%22%5C%22%7D%7D%22%7D`
	return vrfy_buildBaseParams(s, "5", "20539") + "&params=" + params
}

// vrfy_buildB4Body tạo request body cho step B4 — load màn hình xác nhận email (confirmation screen).
// s: Facebook session, s.Datr được dùng làm device_id.
// waterfallId: UUID định danh luồng.
// firstName, lastName: tên đăng ký.
// emailUser, emailDomain: phần user và domain của email đã submit ở B3.
// Trả về body URL-encoded gửi đến EndpointB4, INTERNAL_INFRA_screen_id=CAA_REG_CONFIRMATION_SCREEN.
func vrfy_buildB4Body(s *instagram.Session, waterfallId, firstName, lastName, emailUser, emailDomain string) string {
	regInfo := vrfy_buildRegInfoEmailB4(s, firstName, lastName, emailUser, emailDomain)
	// EXACT từ WeBM B4 (line 662): EMAIL reg_info, __req=6, __hs=20539
	// server_params: device_id(datr) → waterfall_id → is_platform_login → is_from_logged_out → access_flow_version → reg_info → flow_info → current_step → INTERNAL_INFRA_screen_id(CAA_REG_CONFIRMATION_SCREEN)
	params := `%7B%22params%22%3A%22%7B%5C%22server_params%5C%22%3A%7B%5C%22device_id%5C%22%3A%5C%22` + s.Datr + `%5C%22%2C%5C%22waterfall_id%5C%22%3A%5C%22` + waterfallId + `%5C%22%2C%5C%22is_platform_login%5C%22%3A0%2C%5C%22is_from_logged_out%5C%22%3A0%2C%5C%22access_flow_version%5C%22%3A%5C%22pre_mt_behavior%5C%22%2C%5C%22reg_info%5C%22%3A%5C%22%7B` + regInfo + `%7D%5C%22%2C%5C%22flow_info%5C%22%3A%5C%22%7B%5C%5C%5C%22flow_name%5C%5C%5C%22%3A%5C%5C%5C%22new_to_family_fb_default%5C%5C%5C%22%2C%5C%5C%5C%22flow_type%5C%5C%5C%22%3A%5C%5C%5C%22ntf%5C%5C%5C%22%7D%5C%22%2C%5C%22current_step%5C%22%3A10%2C%5C%22INTERNAL_INFRA_screen_id%5C%22%3A%5C%22CAA_REG_CONFIRMATION_SCREEN%5C%22%7D%2C%5C%22client_input_params%5C%22%3A%7B%5C%22lois_settings%5C%22%3A%7B%5C%22lois_token%5C%22%3A%5C%22%5C%22%7D%2C%5C%22aac%5C%22%3A%5C%22%5C%22%7D%7D%22%7D`
	return vrfy_buildBaseParams(s, "6", "20539") + "&params=" + params
}

// vrfy_buildB5Body tạo request body cho step B5 — submit OTP code để hoàn tất đăng ký.
// s: Facebook session, s.Datr dùng làm device_id.
// waterfallId: UUID định danh luồng.
// firstName, lastName: tên đăng ký.
// emailUser, emailDomain: email đã xác nhận ở B3/B4.
// code: OTP code lấy từ hộp thư email.
// Trả về body URL-encoded gửi đến EndpointB5.
func vrfy_buildB5Body(s *instagram.Session, waterfallId, firstName, lastName, emailUser, emailDomain, code string) string {
	regInfo := vrfy_buildRegInfoEmail(s, firstName, lastName, emailUser, emailDomain)
	eventRequestId := generateUUID()
	// EXACT từ WeBM ConfirmOTP() line 736 — đầy đủ server_params + client_input_params
	// server_params: event_request_id → text_input_id → sms_retriever → wa_timer_id → reg_info → flow_info → current_step → INTERNAL__latency_qpl_* → device_id(datr) → family_device_id → waterfall_id → offline/layered → is_platform_login → is_from_logged_in_switcher → is_from_logged_out → access_flow_version → login_surface
	// client_input_params: cloud_trust_token → block_store_machine_id → code → fb_ig_device_id → confirmed_cp_and_code → network_bssid → lois_settings → aac
	params := `%7B%22params%22%3A%22%7B%5C%22server_params%5C%22%3A%7B%5C%22event_request_id%5C%22%3A%5C%22` + eventRequestId + `%5C%22%2C%5C%22text_input_id%5C%22%3A%5C%2256640198000004%5C%22%2C%5C%22sms_retriever_started_prior_step%5C%22%3A0%2C%5C%22wa_timer_id%5C%22%3A%5C%22wa_retriever%5C%22%2C%5C%22reg_info%5C%22%3A%5C%22%7B` + regInfo + `%7D%5C%22%2C%5C%22flow_info%5C%22%3A%5C%22%7B%5C%5C%5C%22flow_name%5C%5C%5C%22%3A%5C%5C%5C%22new_to_family_fb_default%5C%5C%5C%22%2C%5C%5C%5C%22flow_type%5C%5C%5C%22%3A%5C%5C%5C%22ntf%5C%5C%5C%22%7D%5C%22%2C%5C%22current_step%5C%22%3A10%2C%5C%22INTERNAL__latency_qpl_marker_id%5C%22%3A36707139%2C%5C%22INTERNAL__latency_qpl_instance_id%5C%22%3A%5C%2256640198000208%5C%22%2C%5C%22device_id%5C%22%3A%5C%22` + s.Datr + `%5C%22%2C%5C%22family_device_id%5C%22%3Anull%2C%5C%22waterfall_id%5C%22%3A%5C%22` + waterfallId + `%5C%22%2C%5C%22offline_experiment_group%5C%22%3Anull%2C%5C%22layered_homepage_experiment_group%5C%22%3Anull%2C%5C%22is_platform_login%5C%22%3A0%2C%5C%22is_from_logged_in_switcher%5C%22%3A0%2C%5C%22is_from_logged_out%5C%22%3A0%2C%5C%22access_flow_version%5C%22%3A%5C%22pre_mt_behavior%5C%22%2C%5C%22login_surface%5C%22%3A%5C%22unknown%5C%22%7D%2C%5C%22client_input_params%5C%22%3A%7B%5C%22cloud_trust_token%5C%22%3Anull%2C%5C%22block_store_machine_id%5C%22%3A%5C%22%5C%22%2C%5C%22code%5C%22%3A%5C%22` + code + `%5C%22%2C%5C%22fb_ig_device_id%5C%22%3A%5B%5D%2C%5C%22confirmed_cp_and_code%5C%22%3A%7B%7D%2C%5C%22network_bssid%5C%22%3Anull%2C%5C%22lois_settings%5C%22%3A%7B%5C%22lois_token%5C%22%3A%5C%22%5C%22%7D%2C%5C%22aac%5C%22%3A%5C%22%5C%22%7D%7D%22%7D`
	return vrfy_buildBaseParams(s, "6", "20539") + "&params=" + params
}

// vrfy_buildResendBaseParams tạo phần prefix base params dành riêng cho request Resend OTP.
//
// Khác vrfy_buildBaseParams ở các điểm sau (theo WeBM):
//   - __req="a" (fixed string, không phải số thứ tự vì Resend là action độc lập).
//   - __hs="20544.BP:mtouch_pkg.2.0...0" (phiên bản hs khác với B1-B5, cố định cho Resend).
//   - __ccg="GOOD" thay vì "EXCELLENT" — Facebook phân biệt connection quality giữa
//     các loại action; Resend dùng "GOOD" theo WeBM để match đúng client context.
//   - __dyn có giá trị đầy đủ (không rỗng như B1-B5) vì WeBM truyền __dyn khác nhau
//     tùy theo app state tại thời điểm gửi Resend.
//
// s: Facebook session chứa UID, Rev, S, Hsi, FbDtsg, Jazoest, Lsd.
// Trả về chuỗi URL-encoded sẵn sàng nối với "&params=...".
func vrfy_buildResendBaseParams(s *instagram.Session) string {
	return fmt.Sprintf("__user=%s&__a=1&__req=a&__hs=20544.BP%%3Amtouch_pkg.2.0...0&dpr=3&__ccg=GOOD&__rev=%s&__s=%s&__hsi=%s&__dyn=1KQdAG1mws8-t0BBBwno4a2i5U4e1FwKwSwMxW0Horx67o1g8hw23E52q1ew2io0D24o1sE522G0pS0H83bw4FwmE2ewnE2Lx-220n6azo7u0zE2ZwrU6qEbU1kU1bo8Xw8S0QU3yw&fb_dtsg=%s&jazoest=%s&lsd=%s",
		s.UID, s.Rev, s.S, s.Hsi, s.FbDtsg, s.Jazoest, s.Lsd)
}

// vrfy_buildResendBody tạo request body để gửi lại OTP code về email.
// s: Facebook session.
// waterfallId: UUID định danh luồng.
// firstName, lastName: tên đăng ký.
// emailUser, emailDomain: email cần gửi lại OTP.
// Trả về body URL-encoded gửi đến EndpointResend (dùng vrfy_buildResendBaseParams thay vì vrfy_buildBaseParams).
func vrfy_buildResendBody(s *instagram.Session, waterfallId, firstName, lastName, emailUser, emailDomain string) string {
	regInfo := vrfy_buildRegInfoEmail(s, firstName, lastName, emailUser, emailDomain)
	// EXACT từ WeBM Resend — server_params đầy đủ với flow_info, current_step, latency qpl, device_id, waterfall_id, access_flow_version, login_surface
	// client_input_params thêm device_id, cloud_trust_token, network_bssid theo WeBM
	params := `%7B%22params%22%3A%22%7B%5C%22server_params%5C%22%3A%7B%5C%22reg_info%5C%22%3A%5C%22%7B` + regInfo + `%7D%5C%22%2C%5C%22flow_info%5C%22%3A%5C%22%7B%5C%5C%5C%22flow_name%5C%5C%5C%22%3A%5C%5C%5C%22new_to_family_fb_default%5C%5C%5C%22%2C%5C%5C%5C%22flow_type%5C%5C%5C%22%3A%5C%5C%5C%22ntf%5C%5C%5C%22%7D%5C%22%2C%5C%22current_step%5C%22%3A10%2C%5C%22INTERNAL__latency_qpl_marker_id%5C%22%3A36707139%2C%5C%22INTERNAL__latency_qpl_instance_id%5C%22%3A%5C%22146973146400000%5C%22%2C%5C%22device_id%5C%22%3Anull%2C%5C%22family_device_id%5C%22%3Anull%2C%5C%22waterfall_id%5C%22%3A%5C%22` + waterfallId + `%5C%22%2C%5C%22offline_experiment_group%5C%22%3Anull%2C%5C%22layered_homepage_experiment_group%5C%22%3Anull%2C%5C%22is_platform_login%5C%22%3A0%2C%5C%22is_from_logged_in_switcher%5C%22%3A0%2C%5C%22is_from_logged_out%5C%22%3A0%2C%5C%22access_flow_version%5C%22%3A%5C%22pre_mt_behavior%5C%22%2C%5C%22login_surface%5C%22%3A%5C%22unknown%5C%22%7D%2C%5C%22client_input_params%5C%22%3A%7B%5C%22device_id%5C%22%3A%5C%22%5C%22%2C%5C%22cloud_trust_token%5C%22%3Anull%2C%5C%22network_bssid%5C%22%3Anull%2C%5C%22lois_settings%5C%22%3A%7B%5C%22lois_token%5C%22%3A%5C%22%5C%22%7D%2C%5C%22aac%5C%22%3A%5C%22%5C%22%7D%7D%22%7D`
	return vrfy_buildResendBaseParams(s) + "&params=" + params
}
