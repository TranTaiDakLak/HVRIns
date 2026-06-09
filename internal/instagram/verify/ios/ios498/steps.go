// steps.go — iOS555 verify: Spec + iOS Bloks CAA body builders + iOS headers.
//
// Delegate orchestration cho verifybase.RunVerify. Body theo single-shot CAA
// (reg_info + current_step) như verify Android s563, nhưng dùng envelope iOS
// (voiceover_enabled / generic_attachment_* / nt_context FDS+XMDS) + header FBIOS.
package ios498

import (
	"context"
	crand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	mrand "math/rand"
	"strings"
	"time"

	"github.com/google/uuid"

	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	ios498reg "HVRIns/internal/instagram/register/ios/ios498"
	"HVRIns/internal/instagram/verify/verifybase"
)

// ─── iOS app constants (capture APIRegVer_IOS) ───────────────────────────────

const (
	// Bloks constants capture từ v563 (EnterCode563).
	verifyDocID    = "37580109606372898175404581876"
	verifyBloksVer = "2929ef1141317336db1578750a853a1dbe022c9caeba6b822f5e1a01b7c2a37c"
	verifyStylesID = "41a4de4c20a80fadf5d7c245bbd505a1"

	// fbAppVersion / fbBuildNum — dùng cho header x-fb-request-analytics-tags (cần cố định).
	fbAppVersion = "498.1.0.49.107"
	fbBuildNum   = "689933446"
	// iOS FB app product id — dùng trong x-fb-request-analytics-tags.
	iosProductID = "6628568379"
)

// ─── verifyAccount: build Spec → RunVerify ───────────────────────────────────

func verifyAccount(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(session.UID, msg)
		}
	}

	if session.Token != "" {
		notify(fmt.Sprintf("[iOS498 Verify] User token flow — token=%.20s...", session.Token))
	} else if session.Srnonce != "" && session.SessionlessCryptedUID != "" {
		notify(fmt.Sprintf("[iOS498 Verify] Sessionless flow — srnonce=%.20s...", session.Srnonce))
	} else {
		notify("[iOS498 Verify] Không có token/srnonce — sẽ thử login lấy EAAAAU rồi verify")
	}

	cloudTrustToken := genCloudTrustToken()
	spec := verifybase.Spec{
		Tag:              "[iOS498 Verify]",
		DocID:            verifyDocID,
		BloksVer:         verifyBloksVer,
		StylesID:         verifyStylesID,
		IsPushOn:         false,
		AddEmailTimeout:  30 * time.Second,
		CreateClient:     verifybase.CreateIOSClient,
		GraphEndpoint:    verifybase.GraphURL,
		MachineIDFunc:    func(_ string) string { return iosMachineID() },
		CheckLiveDieFunc: verifybase.CheckLiveDieCombined,
		CheckConfirmSuccess: func(resp string) bool {
			// Primary: explicit confirmation_success marker (capture NhapOTP [7297]).
			if strings.Contains(resp, "confirmation_success") {
				return true
			}
			// Fallback: has fb_bloks_action, no confirmation_failure, no server error.
			hasBloks := strings.Contains(resp, "fb_bloks_action") || strings.Contains(resp, "fb_bloks_app")
			hasFailure := strings.Contains(resp, "confirmation_failure")
			hasServerError := strings.Contains(resp, "something went wrong") || strings.Contains(resp, "went wrong")
			return hasBloks && !hasFailure && !hasServerError
		},
		FixUA: func(ua, phone string) (string, string) {
			// UA iOS hợp lệ (FBAN/FBIOS) → giữ nguyên. Nếu là UA Android → thay.
			if containsFBIOS(ua) {
				return "", ""
			}
			newUA := RandomUA(verifybase.CountryFromPhone(phone))
			return newUA, "UA regenerated (FBIOS/562)"
		},
		SetupSessionCtx: func(sc *verifybase.SessionCtx) {
			sc.CloudTrustToken = cloudTrustToken
			sc.ConnType = fakeinfo.RandomIOSConnType()
			verifybase.SetAppnetFields(sc)
		},
		CloudTrustToken:          cloudTrustToken,
		Srnonce:                  session.Srnonce,
		SessionlessCryptedUID:    session.SessionlessCryptedUID,
		Phone:                    session.Phone,
		RegistrationFlowID:       uuid.New().String(),
		BuildHeaders:             buildIOSVerifyHeaders,
		BuildAddEmailBody:        buildAddEmailBody,
		BuildConfirmBody:         buildConfirmBody,
		BuildResendBody:          buildResendBody,
		BuildCloudTrustTokenBody: buildCloudTrustTokenBody,
		Enable2FA:                nil, // iOS555 chưa hỗ trợ 2FA
		PostConfirm:              nil,
		// iOS verify BẮT BUỘC user token iOS EAAAAAY — token Android EAAAAU KHÔNG hợp lệ
		// với endpoint Bloks CAA iOS.
		ValidateToken: func(tok string) bool {
			return strings.HasPrefix(tok, "EAAAAAY")
		},
		// Chưa có EAAAAAY (reg Android / reg iOS nosess / chỉ có EAAAAU / chỉ UID+pass)
		// → login iOS (CAA send_login_request) để lấy EAAAAAY. KHÔNG dùng /auth/login Android.
		FetchToken: func(fctx context.Context, sess *instagram.Session) (string, error) {
			cc := verifybase.CountryFromPhone(sess.Phone)
			tok, cookie, err := ios498reg.FetchIOSToken(fctx, sess.UID, sess.Password, sess.Datr, sess.Proxy, cc, notify)
			if err != nil {
				return "", err
			}
			if cookie != "" {
				sess.Cookie = cookie // ưu tiên cookie login MỚI (ghi đè cookie reg cũ/chết) để UI cập nhật
			}
			return tok, nil
		},
	}
	return verifybase.RunVerify(ctx, session, cfg, outputPath, onStatus, spec)
}

// genCloudTrustToken tạo X-Cloud-Trust-Token: 2 UUID uppercase nối liền (72 chars).
func genCloudTrustToken() string {
	return strings.ToUpper(uuid.New().String()) + strings.ToUpper(uuid.New().String())
}

func containsFBIOS(ua string) bool {
	for i := 0; i+8 <= len(ua); i++ {
		if ua[i:i+8] == "FBAN/FBI" {
			return true
		}
	}
	return false
}

// ─── iOS UA ──────────────────────────────────────────────────────────────────

// RandomUA trả native FBIOS User-Agent random từ pool thiết bị + pool FB builds.
// Data tables → devices.go.
func RandomUA(countryCode string) string {
	locale := fakeinfo.LocaleFromCountry(countryCode)
	if locale == "" {
		locale = randVerifyIOSLocale()
	}
	devs := getVerifyIOSDevices()
	d := devs[mrand.Intn(len(devs))]
	b := randVerifyFBBuild()
	return fmt.Sprintf(
		"Mozilla/5.0 (iPhone; CPU iPhone OS %s like Mac OS X) "+
			"AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/%s "+
			"[FBAN/FBIOS;FBAV/%s;FBBV/%s;FBDV/%s;FBMD/iPhone;FBSN/iOS;"+
			"FBSV/%s;FBSS/%s;FBID/phone;FBLC/%s;FBOP/5;FBRV/%s]",
		d.IOSUnder, d.MobileBld, b.FBAV, b.FBBV, d.FBDV,
		d.IOSDot, d.FBSS, locale, b.FBRV,
	)
}

// ─── Headers (FBIOS) ─────────────────────────────────────────────────────────

const iosAppToken = "6628568379|c1e620fa708a1d5696fb991c1bde5662"

func buildIOSVerifyHeaders(sc *verifybase.SessionCtx, friendlyName string, withZeroState bool) [][2]string {
	_ = withZeroState
	analyticsTag := `{"network_tags":{"product":"` + iosProductID + `","request_category":"graphql","purpose":"fetch","retry_attempt":"0"}}`
	iosFriendlyName := strings.Replace(friendlyName, "FbBloksActionRootQuery-", "FBBloksActionRootQuery-", 1)
	// Authorization: dùng user token (EAAAAAY) nếu có, fallback app token cho sessionless flow.
	authToken := sc.Token
	if authToken == "" {
		authToken = iosAppToken
	}
	// connType — guard: nếu SetupSessionCtx chưa set thì fallback LTE (khớp capture).
	connType := sc.ConnType
	if connType == "" {
		connType = "mobile.CTRadioAccessTechnologyLTE"
	}
	// Thứ tự và headers theo đúng capture [498] AddMailV3.
	headers := [][2]string{
		{"user-agent", sc.UA},
		{"accept-encoding", "gzip, deflate, br"},
		{"accept", "*/*"},
		{"x-fb-rmd", "state=URL_ELIGIBLE"},
		{"x-fb-appnetsession-sid", sc.AppnetSID},
		{"x-meta-usdid", verifybase.GenUSDID()},
		{"x-fb-http-engine", "Tigon/Liger"},
		{"x-meta-zca", `{"e": {"c":7}}`},
		{"x-fb-session-gk", "v1:gk:fb_ios_tasos_congestion_signal:@pass;"},
		{"authorization", "OAuth " + authToken},
		{"x-fb-sim-hni", sc.Sim.HNI},
		{"content-encoding", "gzip"},
		{"x-fb-appnetsession-nid", sc.AppnetNID},
		{"x-fb-connection-type", connType},
		{"x-cloud-trust-token", sc.CloudTrustToken},
		{"x-fb-integrity-machine-id", sc.MachineID},
		{"x-fb-device-id", sc.DeviceID},
		{"x-fb-friendly-name", iosFriendlyName},
		{"x-fb-tasos-experimental", "1"},
		{"content-type", "application/x-www-form-urlencoded"},
		{"x-tigon-is-retry", "False"},
		{"x-fb-request-analytics-tags", analyticsTag},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
		{"x-fb-conn-uuid-client", connUUID()},
		{"x-graphql-client-library", "pando"},
		{"x-graphql-request-purpose", "prefetch"},
		{"x-fb-aed", "684"},
		{"x-fb-optimizer", "0"},
	}
	return headers
}

func connUUID() string {
	id := uuid.New()
	return base64.StdEncoding.EncodeToString(id[:])
}

// iosMachineID generates a 24-char base64url (no-padding) iOS machine ID.
// 18 random bytes → RawURLEncoding = exactly 24 chars, matching captures.
func iosMachineID() string {
	b := make([]byte, 18)
	if _, err := crand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(mrand.Intn(256))
		}
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// genAAC generates the aac (Anti-Automation Code) JSON string sent in iOS body.
// Format matches captures: {"aac_init_timestamp":<unix>,"aacjid":<uuid>,"aaccs":<32-byte-base64url>}
func genAAC() string {
	b := make([]byte, 32)
	if _, err := crand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(mrand.Intn(256))
		}
	}
	return fmt.Sprintf(`{"aac_init_timestamp":%d,"aacjid":"%s","aaccs":"%s"}`,
		time.Now().Unix(), uuid.New().String(), base64.RawURLEncoding.EncodeToString(b))
}

// ─── iOS variables envelope ──────────────────────────────────────────────────

// fdsThemeParamsAddEmail — 3 giá trị theo capture [498] AddMailV3.
var fdsThemeParamsAddEmail = []string{
	"DARKER_PRIMARY_DEEMPHASIZED_BUTTON_BACKGROUND_TEST",
	"MEDIA_INNER_BORDER_WHITE_ALPHA_08_FOR_DARK_TEST",
	"DEFAULT",
}

// fdsThemeParamsResendConfirm — 2 giá trị theo capture [516][519] Resend/Confirm.
var fdsThemeParamsResendConfirm = []string{
	"DARKER_PRIMARY_DEEMPHASIZED_BUTTON_BACKGROUND_TEST",
	"DEFAULT",
}

// buildIOSVariables bọc paramsObj trong envelope variables của iOS.
// fdsValues: danh sách FDS theme values — khác nhau theo bước (AddEmail vs Resend/Confirm).
func buildIOSVariables(paramsObj map[string]any, fdsValues []string) map[string]any {
	return map[string]any{
		"voiceover_enabled":            "false",
		"should_include_delegate_page": true,
		"scale":                        2,
		"generic_attachment_tall_cover_image_height":         352,
		"include_workplace_fields":                           "false",
		"formatType":                                         "concise",
		"generic_attachment_tall_cover_image_width_no_scale": 335,
		"params": paramsObj,
		"generic_attachment_tall_cover_image_width": 670,
		"nt_context": map[string]any{
			"theme_params": []any{
				map[string]any{
					"design_system_name": "FDS",
					"value":              fdsValues,
				},
				map[string]any{
					"design_system_name": "XMDS",
					"value":              []string{"three_neutral_gray"},
				},
			},
			"pixel_ratio":   2,
			"styles_id":     verifyStylesID,
			"bloks_version": verifyBloksVer,
		},
		"generic_attachment_small_cover_image_height":       80,
		"enable_voiceover_gating_for_accessibility_caption": "true",
		"device": "iphone",
		"generic_attachment_small_cover_image_width": 80,
	}
}

// buildIOSVerifyBody ghép form body hoàn chỉnh từ server_params + client_input_params.
// docID cho phép mỗi bước dùng client_doc_id riêng (addEmail dùng addMailDocID, các bước khác dùng verifyDocID).
// fdsValues: FDS theme_params khác nhau theo bước — truyền fdsThemeParamsAddEmail hoặc fdsThemeParamsResendConfirm.
func buildIOSVerifyBody(appID, friendlyName, docID, locale string, fdsValues []string, serverParams, clientInputParams map[string]any) (string, error) {
	inner := map[string]any{
		"server_params":       serverParams,
		"client_input_params": clientInputParams,
	}
	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return "", err
	}
	paramsObj := map[string]any{
		"bloks_versioning_id": verifyBloksVer,
		"app_id":              appID,
		"params":              string(innerJSON),
	}
	variablesJSON, err := json.Marshal(buildIOSVariables(paramsObj, fdsValues))
	if err != nil {
		return "", err
	}
	traceID := uuid.New().String()
	return verifybase.BuildFormBody(friendlyName, docID, string(variablesJSON), traceID, locale), nil
}

// ─── reg_info builders ───────────────────────────────────────────────────────

func latencyID() int64 {
	return int64(80000000000000 + mrand.Int63n(9000000000000))
}

// buildAddEmailRegInfo — reg_info cho bước add email (capture [498] AddMailV3).
func buildAddEmailRegInfo(emailAddr, firstName, lastName, deviceID, familyDevID, machineID, registrationFlowID string, gender int) string {
	ri := map[string]any{
		"first_name":                              firstName,
		"last_name":                               lastName,
		"full_name":                               nil,
		"contactpoint":                            emailAddr,
		"ar_contactpoint":                         nil,
		"contactpoint_type":                       "email",
		"is_using_unified_cp":                     false,
		"unified_cp_screen_variant":               nil,
		"is_cp_auto_confirmed":                    false,
		"is_cp_auto_confirmable":                  false,
		"is_cp_claimed":                           false,
		"confirmation_code":                       nil,
		"birthday":                                nil,
		"birthday_derived_from_age":               nil,
		"age_range":                               "o18",
		"did_use_age":                             false,
		"os_shared_age_range":                     nil,
		"gender":                                  gender,
		"use_custom_gender":                       false,
		"custom_gender":                           nil,
		"encrypted_password":                      nil,
		"username":                                nil,
		"username_prefill":                        nil,
		"accounts_list_client":                    []any{},
		"fb_conf_source":                          nil,
		"device_id":                               deviceID,
		"ig4a_qe_device_id":                       nil,
		"family_device_id":                        familyDevID,
		"fdid_available_on_start":                 nil,
		"fdid_rid_available_on_start":             nil,
		"asdid_available_on_start":                nil,
		"user_id":                                 nil,
		"safetynet_token":                         nil,
		"skip_slow_rel_check":                     false,
		"safetynet_response":                      nil,
		"machine_id":                              machineID,
		"profile_photo":                           nil,
		"profile_photo_id":                        nil,
		"profile_photo_upload_id":                 nil,
		"avatar":                                  nil,
		"email_oauth_token_no_contact_perm":       nil,
		"email_oauth_token":                       nil,
		"email_oauth_tokens":                      []any{},
		"sign_in_with_google_email":               nil,
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
		"spectra_entry_source":                    nil,
		"spectra_reg_token":                       nil,
		"spectra_reg_guardian_id":                 nil,
		"spectra_reg_guardian_logged_in_context":  nil,
		"spectra_requester_user_id":               nil,
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
		"is_ca_late_teen":                         nil,
		"is_early_teen":                           nil,
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
		"registration_flow_id":                    registrationFlowID,
		"should_skip_youth_tos":                   false,
		"is_youth_regulation_flow_complete":       false,
		"is_on_cold_start":                        false,
		"email_prefilled":                         false,
		"cp_confirmed_by_auto_conf":               false,
		"in_sowa_experiment":                      false,
		"youth_regulation_config":                 nil,
		"conf_allow_back_nav_after_change_cp":     true,
		"conf_bouncing_cliff_screen_type":         nil,
		"conf_show_bouncing_cliff":                nil,
		"eligible_to_flash_call_in_ig4a":          false,
		"eligible_to_mo_sms_in_ig4a":              false,
		"mo_sms_ent_id":                           nil,
		"flash_call_permissions_status":           nil,
		"gms_incoming_call_retriever_eligibility": nil,
		"attestation_result":                      nil,
		"request_data_and_challenge_nonce_string": nil,
		"confirmed_cp_and_code":                   nil,
		"notification_callback_id":                nil,
		"reg_suma_state":                          0,
		"is_msplit_neutral_choice":                false,
		"msg_previous_cp":                         nil,
		"ntp_import_source_info":                  nil,
		"youth_consent_decision_time":             nil,
		"sk_pipa_consent_given":                   nil,
		"should_show_spi_before_conf":             true,
		"google_oauth_account":                    nil,
		"is_reg_request_from_ig_suma":             false,
		"is_toa_reg":                              false,
		"is_threads_public":                       false,
		"spc_import_flow":                         false,
		"caa_play_integrity_attestation_result":   nil,
		"client_known_key_hash":                   "QWSor+rArTyWfhlbjsFqp834EyrOBCr2jangGdDSzUE=",
		"flash_call_provider":                     nil,
		"is_in_gms_experience":                    nil,
		"flash_call_nonce_prefix_details":         nil,
		"spc_birthday_input":                      false,
		"failed_birthday_year_count":              nil,
		"user_presented_medium_source":            "cora_strong_recommendation",
		"user_opted_out_of_ntp":                   nil,
		"is_from_registration_reminder":           false,
		"show_youth_reg_in_ig_spc":                false,
		"fb_suma_is_high_confidence":              nil,
		"screen_visited": []any{
			"CAA_REG_WELCOME_SCREEN",
			"bloks.caa.reg.birthday",
			"CAA_REG_CONTACT_POINT_PHONE",
			"CAA_REG_PASSWORD",
			"CAA_REG_CONFIRMATION_SCREEN",
			"CAA_REG_CONFIRMATION_SCREEN",
		},
		"fb_email_login_upsell_skip_suma_post_tos":    false,
		"fb_suma_is_from_email_login_upsell":          false,
		"fb_suma_is_from_phone_login_upsell":          false,
		"should_prefill_cp_in_ar":                     true,
		"ig_partially_created_account_user_id":        0,
		"ig_partially_created_account_nonce":          nil,
		"ig_partially_created_account_nonce_expiry":   0,
		"force_sessionless_nux_experience":            false,
		"has_seen_suma_landing_page_pre_conf":         false,
		"has_seen_suma_candidate_page_pre_conf":       false,
		"has_seen_confirmation_screen":                false,
		"suma_on_conf_threshold":                      -1,
		"should_show_error_msg":                       true,
		"th_profile_photo_token":                      nil,
		"attempted_silent_auth_in_fb":                 false,
		"attempted_silent_auth_in_ig":                 false,
		"sa_prefetch_callback_id":                     nil,
		"cp_suma_results_map":                         nil,
		"source_username":                             nil,
		"next_uri":                                    nil,
		"should_use_next_uri":                         nil,
		"linking_entry_point":                         nil,
		"fb_encrypted_partial_new_account_properties": nil,
		"starter_pack_name":                           nil,
		"starter_pack_creator_user_ids":               nil,
		"wa_data_bundle":                              nil,
		"bloks_controller_source":                     "bk_caa_reg_icon_text_list_tos_screen",
		"airwave_registration_code":                   nil,
		"is_sessionless_nux":                          nil,
		"login_contactpoint":                          nil,
		"login_contactpoint_type":                     nil,
		"should_show_bday_after_name_suggestions":     nil,
		"should_override_back_nav":                    false,
		"ig_footer_variant":                           "control",
		"device_network_info":                         nil,
		"is_from_web_lite_reg_controller":             nil,
		"login_form_siwg_email":                       nil,
		"account_setup_waterfall_id":                  nil,
		"is_wanted_suma_user":                         false,
		"device_zero_balance_state":                   nil,
		"wa_to_ig_merged_tos_variant":                 nil,
		"is_in_nta_single_form":                       false,
		"source_account_image_asset_id":               nil,
		"passkey_eligible_device":                     false,
		"nta_control_reason":                          nil,
		"nta_risk_type":                               nil,
		"nta_single_form_variant":                     nil,
		"enable_survey":                               nil,
		"phone_prefetch_outcome":                      nil,
		"tos_accepted_on_profile_info":                nil,
	}
	return verifybase.MustJSON(ri)
}

// buildResendConfirmRegInfo — reg_info minimal cho bước resend/confirm (capture [516][519]).
// Chỉ set các field có giá trị thực, còn lại để null theo đúng capture.
func buildResendConfirmRegInfo(emailAddr, uid, firstName, lastName, deviceID string, gender int) string {
	ri := map[string]any{
		"contactpoint_type": "email",
		"contactpoint":      emailAddr,
		"device_id":         deviceID,
		"first_name":        firstName,
		"last_name":         lastName,
		"full_name":         strings.TrimSpace(firstName + " " + lastName),
		"gender":            gender,
		"screen_visited":    []any{"CAA_REG_CONFIRMATION_SCREEN"},
		"user_id":           uid,
		// All remaining fields null per capture
		"account_setup_waterfall_id":                  nil,
		"accounts_list_client":                        nil,
		"age_range":                                   nil,
		"airwave_registration_code":                   nil,
		"ar_contactpoint":                             nil,
		"asdid_available_on_start":                    nil,
		"attempted_silent_auth_in_fb":                 nil,
		"attempted_silent_auth_in_ig":                 nil,
		"attestation_result":                          nil,
		"avatar":                                      nil,
		"big_blue_token":                              nil,
		"birthday":                                    nil,
		"birthday_derived_from_age":                   nil,
		"bloks_controller_source":                     nil,
		"caa_play_integrity_attestation_result":       nil,
		"caa_reg_flow_source":                         nil,
		"cached_headers_safetynet_info":               nil,
		"ck_error":                                    nil,
		"ck_id":                                       nil,
		"ck_nonce":                                    nil,
		"client_known_key_hash":                       nil,
		"conf_allow_back_nav_after_change_cp":         nil,
		"conf_bouncing_cliff_screen_type":             nil,
		"conf_show_bouncing_cliff":                    nil,
		"confirmation_code":                           nil,
		"confirmation_code_send_error":                nil,
		"confirmation_medium":                         nil,
		"confirmed_cp_and_code":                       nil,
		"cp_confirmed_by_auto_conf":                   nil,
		"cp_suma_results_map":                         nil,
		"create_new_to_app_account":                   nil,
		"crypted_user_id":                             nil,
		"custom_gender":                               nil,
		"dema_account_consent_given":                  nil,
		"device_network_info":                         nil,
		"device_zero_balance_state":                   nil,
		"did_use_age":                                 nil,
		"dismissed_login_upsell_with_cna":             nil,
		"dma_data_combination_consent_given":          nil,
		"eligible_to_flash_call_in_ig4a":              nil,
		"eligible_to_mo_sms_in_ig4a":                  nil,
		"email_oauth_exists":                          nil,
		"email_oauth_token":                           nil,
		"email_oauth_token_no_contact_perm":           nil,
		"email_oauth_tokens":                          nil,
		"email_prefilled":                             nil,
		"enable_survey":                               nil,
		"encrypted_msisdn":                            nil,
		"encrypted_msisdn_for_safetynet":              nil,
		"encrypted_password":                          nil,
		"existing_account_exact_match_checked":        nil,
		"existing_account_fuzzy_match_checked":        nil,
		"existing_accounts":                           nil,
		"failed_birthday_year_count":                  nil,
		"family_device_id":                            nil,
		"fb_access_token":                             nil,
		"fb_conf_source":                              nil,
		"fb_device_id":                                nil,
		"fb_email_login_upsell_skip_suma_post_tos":    nil,
		"fb_encrypted_partial_new_account_properties": nil,
		"fb_machine_id":                               nil,
		"fb_suma_is_from_email_login_upsell":          nil,
		"fb_suma_is_from_phone_login_upsell":          nil,
		"fb_suma_is_high_confidence":                  nil,
		"fdid_available_on_start":                     nil,
		"fdid_rid_available_on_start":                 nil,
		"flash_call_nonce_prefix_details":             nil,
		"flash_call_permissions_status":               nil,
		"flash_call_provider":                         nil,
		"force_sessionless_nux_experience":            nil,
		"frl_authorization_token":                     nil,
		"full_sheet_flow":                             nil,
		"gms_incoming_call_retriever_eligibility":     nil,
		"google_oauth_account":                        nil,
		"has_seen_confirmation_screen":                nil,
		"has_seen_suma_candidate_page_pre_conf":       nil,
		"has_seen_suma_landing_page_pre_conf":         nil,
		"headers_flow_id":                             nil,
		"headers_last_infra_flow_id":                  nil,
		"headers_last_infra_flow_id_safetynet":        nil,
		"ig4a_qe_device_id":                           nil,
		"ig_authorization_token":                      nil,
		"ig_device_id":                                nil,
		"ig_footer_variant":                           nil,
		"ig_machine_id":                               nil,
		"ig_partially_created_account_nonce":          nil,
		"ig_partially_created_account_nonce_expiry":   nil,
		"ig_partially_created_account_user_id":        nil,
		"ignore_existing_login":                       nil,
		"ignore_existing_login_after_errors":          nil,
		"ignore_existing_login_from_suma":             nil,
		"ignore_suma_check":                           nil,
		"in_sowa_experiment":                          nil,
		"is_ca_late_teen":                             nil,
		"is_caa_perf_enabled":                         nil,
		"is_cp_auto_confirmable":                      nil,
		"is_cp_auto_confirmed":                        nil,
		"is_cp_claimed":                               nil,
		"is_early_teen":                               nil,
		"is_from_registration_reminder":               nil,
		"is_from_web_lite_reg_controller":             nil,
		"is_in_gms_experience":                        nil,
		"is_in_nta_single_form":                       nil,
		"is_msplit_neutral_choice":                    nil,
		"is_msplit_reg":                               nil,
		"is_on_cold_start":                            nil,
		"is_preform":                                  nil,
		"is_reg_request_from_ig_suma":                 nil,
		"is_sessionless_nux":                          nil,
		"is_spectra_reg":                              nil,
		"is_threads_public":                           nil,
		"is_toa_reg":                                  nil,
		"is_too_young":                                nil,
		"is_using_unified_cp":                         nil,
		"is_wanted_suma_user":                         nil,
		"is_youth_regulation_flow_complete":           nil,
		"linking_entry_point":                         nil,
		"login_contactpoint":                          nil,
		"login_contactpoint_type":                     nil,
		"login_form_siwg_email":                       nil,
		"machine_id":                                  nil,
		"mo_sms_ent_id":                               nil,
		"msg_previous_cp":                             nil,
		"msplit_creator_nonce":                        nil,
		"next_uri":                                    nil,
		"notification_callback_id":                    nil,
		"nta_control_reason":                          nil,
		"nta_risk_type":                               nil,
		"nta_single_form_variant":                     nil,
		"ntp_import_source_info":                      nil,
		"openid_tokens_for_testing":                   nil,
		"os_shared_age_range":                         nil,
		"passkey_eligible_device":                     nil,
		"phone_prefetch_outcome":                      nil,
		"post_form_errors":                            nil,
		"profile_photo":                               nil,
		"profile_photo_id":                            nil,
		"profile_photo_upload_id":                     nil,
		"reg_suma_state":                              nil,
		"registration_flow_id":                        nil,
		"request_data_and_challenge_nonce_string":     nil,
		"safetynet_response":                          nil,
		"safetynet_token":                             nil,
		"should_override_back_nav":                    nil,
		"should_prefill_cp_in_ar":                     nil,
		"should_save_password":                        nil,
		"should_show_bday_after_name_suggestions":     nil,
		"should_show_error_msg":                       nil,
		"should_show_rel_error":                       nil,
		"should_show_spi_before_conf":                 nil,
		"should_skip_headers_safetynet":               nil,
		"should_skip_nta_upsell":                      nil,
		"should_skip_two_step_conf":                   nil,
		"should_skip_youth_tos":                       nil,
		"should_use_next_uri":                         nil,
		"show_youth_reg_in_ig_spc":                    nil,
		"sign_in_with_google_email":                   nil,
		"sk_pipa_consent_given":                       nil,
		"skip_session_info":                           nil,
		"skip_slow_rel_check":                         nil,
		"skip_step_without_errors":                    nil,
		"soap_creation_source":                        nil,
		"source_account_image_asset_id":               nil,
		"source_account_reg_info":                     nil,
		"source_account_type":                         nil,
		"source_account_type_to_reg_info":             nil,
		"source_credentials_type":                     nil,
		"source_cuid":                                 nil,
		"source_username":                             nil,
		"spc_birthday_input":                          nil,
		"spc_import_flow":                             nil,
		"spectra_entry_source":                        nil,
		"spectra_reg_guardian_id":                     nil,
		"spectra_reg_guardian_logged_in_context":      nil,
		"spectra_reg_token":                           nil,
		"spectra_requester_user_id":                   nil,
		"sso_enabled":                                 nil,
		"starter_pack_creator_user_ids":               nil,
		"starter_pack_name":                           nil,
		"suggested_first_name":                        nil,
		"suggested_full_name":                         nil,
		"suggested_last_name":                         nil,
		"suma_on_conf_threshold":                      nil,
		"th_profile_photo_token":                      nil,
		"tos_accepted_on_profile_info":                nil,
		"unified_cp_screen_variant":                   nil,
		"use_custom_gender":                           nil,
		"used_ig_birthday":                            nil,
		"user_id_of_msplit_creator":                   nil,
		"user_opted_out_of_ntp":                       nil,
		"user_presented_medium_source":                nil,
		"username":                                    nil,
		"username_prefill":                            nil,
		"wa_data_bundle":                              nil,
		"wa_to_ig_merged_tos_variant":                 nil,
		"was_headers_prefill_available":               nil,
		"whatsapp_installed_on_client":                nil,
		"xapp_accounts":                               nil,
		"youth_consent_decision_time":                 nil,
		"youth_regulation_config":                     nil,
	}
	return verifybase.MustJSON(ri)
}

// ─── Body builders ───────────────────────────────────────────────────────────

func buildAddEmailBody(spec *verifybase.Spec, emailAddr, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string {
	_, _ = uid, sim
	// Capture [498] AddMailV3: flow phone-reg → add email.
	regInfo := buildAddEmailRegInfo(emailAddr, firstName, lastName, deviceID, familyDevID, machineID, spec.RegistrationFlowID, gender)
	ctt := spec.CloudTrustToken
	lat := latencyID()

	serverParams := map[string]any{
		"INTERNAL__latency_qpl_marker_id":   36707139,
		"x_app_device_signals":              map[string]any{"DEVICE_ID": deviceID, "MACHINE_ID": machineID},
		"event_request_id":                  uuid.New().String(),
		"login_surface":                     "aymh_one_tap",
		"cp_source":                         1,
		"srnonce":                           spec.Srnonce,
		"is_from_logged_in_switcher":        0,
		"access_flow_version":               "pre_mt_behavior",
		"cloud_trust_token":                 ctt,
		"flow_info":                         `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`,
		"INTERNAL__latency_qpl_instance_id": lat,
		"login_entry_point":                 "logged_out",
		"is_platform_login":                 0,
		"device_id":                         deviceID,
		"waterfall_id":                      waterfallID,
		"sessionless_crypted_user_id":       spec.SessionlessCryptedUID,
		"cp_funnel":                         1,
		"is_from_logged_out":                1,
		"text_input_id":                     lat - mrand.Int63n(500),
		"layered_homepage_experiment_group": "not_in_experiment",
		"is_from_registration_flow":         1,
		"reg_info":                          regInfo,
		"current_step":                      10,
		"machine_id":                        machineID,
		"offline_experiment_group":          nil,
		"sessionless_flow_info":             `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`,
		"family_device_id":                  familyDevID,
	}
	clientInputParams := map[string]any{
		"aac":                          genAAC(),
		"switch_cp_have_seen_suma":     0,
		"family_device_id":             familyDevID,
		"seen_login_upsell":            0,
		"block_store_machine_id":       "",
		"confirmed_cp_and_code":        map[string]any{},
		"zero_balance_state":           "",
		"cloud_trust_token":            ctt,
		"accounts_list":                []any{},
		"fb_ig_device_id":              []any{},
		"network_bssid":                nil,
		"email_prefilled":              0,
		"lois_settings":                map[string]any{"lois_token": ""},
		"has_rejected_rel":             0,
		"msg_previous_cp":              spec.Phone,
		"device_id":                    deviceID,
		"is_from_device_emails":        0,
		"email":                        emailAddr,
		"switch_cp_first_time_loading": 1,
	}
	body, err := buildIOSVerifyBody(
		"com.bloks.www.bloks.caa.reg.async.contactpoint_email.async",
		verifybase.AddEmailFriendlyName, verifyDocID, locale, fdsThemeParamsAddEmail, serverParams, clientInputParams)
	if err != nil {
		return ""
	}
	return body
}

func buildConfirmBody(spec *verifybase.Spec, emailAddr, code, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string {
	_ = sim
	lat := latencyID()
	regInfo := buildResendConfirmRegInfo(emailAddr, uid, firstName, lastName, deviceID, gender)
	ctt := spec.CloudTrustToken

	// Capture [519] ConfirmCode.
	serverParams := map[string]any{
		"INTERNAL__latency_qpl_marker_id":   36707139,
		"INTERNAL__latency_qpl_instance_id": lat,
		"event_request_id":                  uuid.New().String(),
		"login_surface":                     "unknown",
		"srnonce":                           spec.Srnonce,
		"sessionless_crypted_user_id":       spec.SessionlessCryptedUID,
		"is_from_logged_in_switcher":        0,
		"is_platform_login":                 0,
		"is_from_logged_out":                0,
		"is_from_registration_flow":         0,
		"access_flow_version":               "pre_mt_behavior",
		"sms_retriever_started_prior_step":  0,
		"flow_info":                         `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`,
		"sessionless_flow_info":             `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`,
		"text_input_id":                     lat - mrand.Int63n(500),
		"device_id":                         deviceID,
		"waterfall_id":                      waterfallID,
		"wa_timer_id":                       "wa_retriever",
		"layered_homepage_experiment_group": nil,
		"offline_experiment_group":          nil,
		"family_device_id":                  nil,
		"current_step":                      10,
		"reg_info":                          regInfo,
	}
	clientInputParams := map[string]any{
		"confirmed_cp_and_code":  map[string]any{},
		"lois_settings":          map[string]any{"lois_token": ""},
		"network_bssid":          nil,
		"cloud_trust_token":      ctt,
		"code":                   code,
		"family_device_id":       familyDevID,
		"device_id":              deviceID,
		"block_store_machine_id": "",
		"aac":                    "",
		"fb_ig_device_id":        []any{},
		"machine_id":             machineID,
	}
	body, err := buildIOSVerifyBody(
		"com.bloks.www.bloks.caa.reg.confirmation.async",
		verifybase.ConfirmFriendlyName, verifyDocID, locale, fdsThemeParamsResendConfirm, serverParams, clientInputParams)
	if err != nil {
		return ""
	}
	return body
}

func buildResendBody(spec *verifybase.Spec, emailAddr, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string {
	_ = sim
	lat := latencyID()
	regInfo := buildResendConfirmRegInfo(emailAddr, uid, firstName, lastName, deviceID, gender)
	ctt := spec.CloudTrustToken

	// Capture [516] ResendCode.
	serverParams := map[string]any{
		"INTERNAL__latency_qpl_marker_id":   36707139,
		"INTERNAL__latency_qpl_instance_id": lat,
		"srnonce":                           spec.Srnonce,
		"login_surface":                     "unknown",
		"flow_info":                         `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`,
		"sessionless_flow_info":             `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`,
		"sessionless_crypted_user_id":       spec.SessionlessCryptedUID,
		"access_flow_version":               "pre_mt_behavior",
		"is_from_logged_in_switcher":        0,
		"is_platform_login":                 0,
		"is_from_logged_out":                0,
		"is_from_registration_flow":         0,
		"current_step":                      10,
		"device_id":                         deviceID,
		"waterfall_id":                      waterfallID,
		"layered_homepage_experiment_group": nil,
		"offline_experiment_group":          nil,
		"family_device_id":                  nil,
		"reg_info":                          regInfo,
	}
	clientInputParams := map[string]any{
		"machine_id":             machineID,
		"network_bssid":          nil,
		"device_id":              deviceID,
		"cloud_trust_token":      ctt,
		"aac":                    "",
		"block_store_machine_id": "",
		"lois_settings":          map[string]any{"lois_token": ""},
	}
	body, err := buildIOSVerifyBody(
		"com.bloks.www.bloks.caa.reg.resend_confirmation.async",
		verifybase.ResendFriendlyName, verifyDocID, locale, fdsThemeParamsResendConfirm, serverParams, clientInputParams)
	if err != nil {
		return ""
	}
	return body
}

// buildCloudTrustTokenBody — body cho bk.cloud_trust_token.async (capture [384]).
// Gọi sau AddEmail thành công để refresh/validate cloud trust token trên server.
func buildCloudTrustTokenBody(spec *verifybase.Spec, deviceID, familyDevID, machineID string) string {
	serverParams := map[string]any{
		"machine_id":  machineID,
		"device_id":   deviceID,
		"cloud_token": spec.CloudTrustToken,
	}
	variablesObj := buildIOSVariables(map[string]any{
		"bloks_versioning_id": verifyBloksVer,
		"app_id":              "com.bloks.www.bk.cloud_trust_token.async",
		"params":              verifybase.MustJSON(map[string]any{"server_params": serverParams}),
	}, fdsThemeParamsResendConfirm)
	variablesJSON, err := json.Marshal(variablesObj)
	if err != nil {
		return ""
	}
	traceID := uuid.New().String()
	return verifybase.BuildFormBody(
		"FBBloksActionRootQuery-com.bloks.www.bk.cloud_trust_token.async",
		verifyDocID, string(variablesJSON), traceID, "")
}
