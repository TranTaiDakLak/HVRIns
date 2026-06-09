// steps.go â€” S565 verify: thin wrapper delegating to verifybase.RunVerify.
// All shared orchestration logic lives in verifybase. This file contains only
// s565-specific constants, UA validation/builder, body builders, 2FA, and addinfo wiring.
// s565 differs from s558: new doc_id/bloks_ver, styles_id, is_push_on=false.
package s565v2s21

import (
	"context"
	"fmt"
	mrand "math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"HVRIns/internal/instagram"
	addinfo "HVRIns/internal/instagram/addinfo/s557"
	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/instagram/fakeinfo/uabuilder"
	androidsec "HVRIns/internal/instagram/security/android"
	"HVRIns/internal/instagram/verify/verifybase"
)

const (
	verifyDocID     = "11994080428380165281378204618"
	verifyBloksVer  = "8e49df647dadb22e676275e71f803c0045cccaf178e55a3033ee1e14eb0c816c"
	defaultStylesID = "0daae0d5b8cb6e6bd9943399b12bcd5b"
)

func verifyAccount(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	spec := verifybase.Spec{
		Tag:             "[s565 verify]",
		DocID:           verifyDocID,
		BloksVer:        verifyBloksVer,
		StylesID:        defaultStylesID,
		IsPushOn:        false,
		AddEmailTimeout: 30 * time.Second,
		FixUA: func(ua, phone string) (string, string) {
			if strings.Contains(ua, "FBAN/FB4A") {
				return "", ""
			}
			country := verifybase.CountryFromPhone(phone)
			newUA := RandomUA(country)
			label := country
			if label == "" {
				label = "random"
			}
			return newUA, fmt.Sprintf("UA regenerated (FBAV/565, country=%s)", label)
		},
		SetupSessionCtx: nil, // s565 does not use appnet/tasos fields
		BuildHeaders: func(sc *verifybase.SessionCtx, friendlyName string, withZeroState bool) [][2]string {
			return verifybase.BuildNewStyleHeaders(sc, friendlyName, withZeroState)
		},
		BuildAddEmailBody: buildAddEmailBody,
		BuildConfirmBody:  buildConfirmBody,
		BuildResendBody:   buildResendBody,
		Enable2FA:         enable2FAForS565S21,
		PostConfirm: func(ctx context.Context, sess *instagram.Session, cfg *instagram.VerifyConfig, notify func(string)) {
			if cfg.AddInfo != nil && cfg.AddInfo.Enabled {
				notify("[s565 verify] Running AddInfo...")
				res := addinfo.RunAddInfo(ctx, sess, cfg.AddInfo, notify)
				if len(res.Notes) > 0 {
					notify(fmt.Sprintf("[s565 verify] AddInfo done: %s", strings.Join(res.Notes, ", ")))
				}
			}
		},
	}
	return verifybase.RunVerify(ctx, session, cfg, outputPath, onStatus, spec)
}

func enable2FAForS565S21(ctx context.Context, session *instagram.Session, uid, machineID, deviceID string, emailOTPFn func(string, int) string, notify func(string)) (string, error) {
	sec := &androidsec.SecurityManager{EmailOTPFn: emailOTPFn}
	s := &instagram.Session{
		Token:     session.Token,
		UID:       uid,
		DeviceID:  deviceID,
		Datr:      machineID,
		UserAgent: session.UserAgent,
		Proxy:     session.Proxy,
		Password:  session.Password,
	}
	res, err := sec.Enable2FA(ctx, s)
	if err != nil {
		return "", err
	}
	notify(fmt.Sprintf("[s565 verify] 2FA secret generated: %s", res.Secret))
	return res.Secret, nil
}

func RandomUA(countryCode string) string {
	res, err := (&uabuilder.AndroidUABuilder{}).Build(uabuilder.UAOptions{
		PoolKind: "ver",
		CountryCode: countryCode,
		Locale:      fakeinfo.LocaleFromCountry(countryCode),
	})
	if err != nil {
		res, err = (&uabuilder.AndroidUABuilder{}).Build(uabuilder.UAOptions{
		PoolKind: "ver",})
		if err != nil {
			return ""
		}
	}
	return res.UserAgent
}

// â”€â”€â”€ Body builders (new style: DENIED perms, si_device_param_network_info) â”€â”€â”€

func buildRegInfo(emailAddr, uid, firstName, lastName, deviceID, familyDevID string, gender int, sim fakeinfo.SimProfile) string {
	fullName := firstName + " " + lastName
	if lastName == "" {
		fullName = firstName
	}
	ri := map[string]interface{}{
		"contactpoint":               emailAddr,
		"contactpoint_type":          "email",
		"encrypted_msisdn":           "",
		"headers_last_infra_flow_id": "",
		"flash_call_permissions_status": map[string]interface{}{
			"READ_PHONE_STATE":   "DENIED",
			"CALL_PHONE":         "DENIED",
			"READ_CALL_LOG":      "DENIED",
			"ANSWER_PHONE_CALLS": "DENIED",
		},
		"first_name":                           firstName,
		"last_name":                            lastName,
		"full_name":                            fullName,
		"is_using_unified_cp":                  false,
		"is_cp_claimed":                        false,
		"age_range":                            "o18",
		"gender":                               gender,
		"device_id":                            deviceID,
		"family_device_id":                     familyDevID,
		"user_id":                              uid,
		"profile_photo":                        nil,
		"whatsapp_installed_on_client":         false,
		"email_prefilled":                      false,
		"conf_allow_back_nav_after_change_nav": true,
		"gms_incoming_call_retriever_eligibility": "eligible",
		"attestation_result":                      map[string]interface{}{},
		"screen_visited":                          []interface{}{"CAA_REG_CONFIRMATION_SCREEN"},
		"suma_on_conf_threshold":                  nil,
	}
	return verifybase.MustJSON(ri)
}

func buildRegInfoForConfirm(emailAddr, uid, deviceID, familyDevID string, sim fakeinfo.SimProfile) string {
	ri := map[string]interface{}{
		"contactpoint":               emailAddr,
		"contactpoint_type":          "email",
		"encrypted_msisdn":           "",
		"headers_last_infra_flow_id": "",
		"flash_call_permissions_status": map[string]interface{}{
			"READ_PHONE_STATE":   "DENIED",
			"CALL_PHONE":         "DENIED",
			"READ_CALL_LOG":      "DENIED",
			"ANSWER_PHONE_CALLS": "DENIED",
		},
		"first_name":                           nil,
		"last_name":                            nil,
		"full_name":                            nil,
		"is_using_unified_cp":                  false,
		"is_cp_claimed":                        false,
		"age_range":                            nil,
		"gender":                               nil,
		"birthday":                             nil,
		"device_id":                            deviceID,
		"family_device_id":                     familyDevID,
		"user_id":                              uid,
		"profile_photo":                        nil,
		"whatsapp_installed_on_client":         false,
		"email_prefilled":                      false,
		"conf_allow_back_nav_after_change_nav": true,
		"gms_incoming_call_retriever_eligibility": "eligible",
		"attestation_result":                      map[string]interface{}{},
		"screen_visited":                          []interface{}{"CAA_REG_CONFIRMATION_SCREEN", "CAA_REG_CONFIRMATION_SCREEN"},
		"suma_on_conf_threshold":                  -1,
	}
	return verifybase.MustJSON(ri)
}

func siDeviceNetworkInfoStr(sim fakeinfo.SimProfile) map[string]interface{} {
	return map[string]interface{}{
		"default_subscription_info": map[string]interface{}{
			"network_type":           13,
			"is_data_roaming":        0,
			"is_esim":                nil,
			"is_gsm_roaming":         0,
			"is_sim_sms_capable":     nil,
			"is_mobile_data_enabled": 0,
			"sim_carrier_id":         1899,
			"sim_carrier_id_name":    sim.OperatorName,
			"sim_state":              5,
			"sim_operator":           sim.HNI,
			"sim_operator_name":      sim.OperatorName,
			"signal_strength":        2,
			"group_id_level_1":       nil,
			"network_operator":       sim.HNI,
		},
		"is_airplane_mode":           0,
		"is_active_network_cellular": 0,
		"is_device_sms_capable":      1,
		"sim_count":                  2,
		"is_wifi":                    1,
		"active_subscriptions_info":  nil,
	}
}

func siDeviceNetworkInfoInt(sim fakeinfo.SimProfile) map[string]interface{} {
	hni, _ := strconv.Atoi(sim.HNI)
	if hni == 0 {
		hni = 45204
	}
	return map[string]interface{}{
		"default_subscription_info": map[string]interface{}{
			"is_mobile_data_enabled": 0,
			"sim_operator":           hni,
			"is_gsm_roaming":         0,
			"group_id_level_1":       nil,
			"sim_operator_name":      sim.OperatorName,
			"sim_state":              5,
			"network_operator":       hni,
			"is_data_roaming":        0,
			"is_esim":                nil,
			"is_sim_sms_capable":     nil,
			"sim_carrier_id":         1899,
			"signal_strength":        2,
			"network_type":           13,
			"sim_carrier_id_name":    sim.OperatorName,
		},
		"sim_count":                  2,
		"is_wifi":                    1,
		"is_airplane_mode":           0,
		"is_active_network_cellular": 0,
		"is_device_sms_capable":      1,
		"active_subscriptions_info":  nil,
	}
}

func buildAddEmailBody(spec *verifybase.Spec, emailAddr, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string {
	traceID := uuid.New().String()
	eventReqID := uuid.New().String()
	latency := int64(80000000000000 + mrand.Int63n(9000000000000))
	regInfo := buildRegInfo(emailAddr, uid, firstName, lastName, deviceID, familyDevID, gender, sim)

	level3 := map[string]interface{}{
		"client_input_params": map[string]interface{}{
			"aac":                          "",
			"device_id":                    deviceID,
			"zero_balance_state":           "init",
			"network_bssid":                nil,
			"msg_previous_cp":              "",
			"machine_id":                   machineID,
			"switch_cp_first_time_loading": 1,
			"has_rejected_rel":             0,
			"seen_login_upsell":            0,
			"accounts_list":                []interface{}{},
			"email_prefilled":              0,
			"confirmed_cp_and_code":        map[string]interface{}{},
			"si_device_param_network_info": siDeviceNetworkInfoStr(sim),
			"family_device_id":             familyDevID,
			"block_store_machine_id":       nil,
			"fb_ig_device_id":              []interface{}{},
			"lois_settings":                map[string]interface{}{"lois_token": ""},
			"cloud_trust_token":            nil,
			"is_from_device_emails":        0,
			"email":                        emailAddr,
			"switch_cp_have_seen_suma":     0,
		},
		"server_params": map[string]interface{}{
			"event_request_id":                  eventReqID,
			"is_from_logged_out":                0,
			"text_input_id":                     latency,
			"layered_homepage_experiment_group": nil,
			"device_id":                         deviceID,
			"login_surface":                     "unknown",
			"waterfall_id":                      waterfallID,
			"INTERNAL__latency_qpl_instance_id": latency + mrand.Int63n(500),
			"flow_info":                         `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`,
			"is_platform_login":                 0,
			"reg_info":                          regInfo,
			"family_device_id":                  familyDevID,
			"offline_experiment_group":          "caa_iteration_v6_perf_fb_2",
			"cp_funnel":                         1,
			"cp_source":                         1,
			"access_flow_version":               "pre_mt_behavior",
			"is_from_logged_in_switcher":        0,
			"current_step":                      10,
		},
	}
	level2 := map[string]interface{}{"params": verifybase.MustJSON(level3)}
	paramsObj := map[string]interface{}{
		"params":              verifybase.MustJSON(level2),
		"bloks_versioning_id": spec.BloksVer,
		"app_id":              "com.bloks.www.bloks.caa.reg.async.contactpoint_email.async",
	}
	variablesJSON := verifybase.BuildVariables(paramsObj, spec.BloksVer, spec.StylesID, spec.IsPushOn)
	return verifybase.BuildFormBody(verifybase.AddEmailFriendlyName, spec.DocID, variablesJSON, traceID, locale)
}

func buildConfirmBody(spec *verifybase.Spec, emailAddr, code, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string {
	_, _, _ = firstName, lastName, gender
	traceID := uuid.New().String()
	eventReqID := uuid.New().String()
	latency := int64(80000000000000 + mrand.Int63n(9000000000000))
	regInfo := buildRegInfoForConfirm(emailAddr, uid, deviceID, familyDevID, sim)

	level3 := map[string]interface{}{
		"client_input_params": map[string]interface{}{
			"confirmed_cp_and_code":  map[string]interface{}{},
			"aac":                    "",
			"family_device_id":       familyDevID,
			"block_store_machine_id": nil,
			"code":                   code,
			"fb_ig_device_id":        []interface{}{},
			"device_id":              deviceID,
			"lois_settings":          map[string]interface{}{"lois_token": ""},
			"cloud_trust_token":      nil,
			"network_bssid":          nil,
		},
		"server_params": map[string]interface{}{
			"event_request_id":                  eventReqID,
			"is_from_logged_out":                0,
			"text_input_id":                     latency,
			"layered_homepage_experiment_group": nil,
			"device_id":                         deviceID,
			"login_surface":                     "unknown",
			"waterfall_id":                      waterfallID,
			"wa_timer_id":                       "wa_retriever",
			"machine_id":                        machineID,
			"INTERNAL__latency_qpl_instance_id": latency + mrand.Int63n(500),
			"flow_info":                         `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`,
			"is_platform_login":                 0,
			"sms_retriever_started_prior_step":  0,
			"si_device_param_network_info":      siDeviceNetworkInfoInt(sim),
			"reg_info":                          regInfo,
			"family_device_id":                  familyDevID,
			"offline_experiment_group":          "caa_iteration_v6_perf_fb_2",
			"access_flow_version":               "pre_mt_behavior",
			"is_from_logged_in_switcher":        0,
			"current_step":                      10,
		},
	}
	level2 := map[string]interface{}{"params": verifybase.MustJSON(level3)}
	paramsObj := map[string]interface{}{
		"params":              verifybase.MustJSON(level2),
		"bloks_versioning_id": spec.BloksVer,
		"app_id":              "com.bloks.www.bloks.caa.reg.confirmation.async",
	}
	variablesJSON := verifybase.BuildVariables(paramsObj, spec.BloksVer, spec.StylesID, spec.IsPushOn)
	return verifybase.BuildFormBody(verifybase.ConfirmFriendlyName, spec.DocID, variablesJSON, traceID, locale)
}

func buildResendBody(spec *verifybase.Spec, emailAddr, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string {
	traceID := uuid.New().String()
	latency := int64(80000000000000 + mrand.Int63n(9000000000000))
	regInfo := buildRegInfo(emailAddr, uid, firstName, lastName, deviceID, familyDevID, gender, sim)

	level3 := map[string]interface{}{
		"client_input_params": map[string]interface{}{
			"aac":                    "",
			"block_store_machine_id": nil,
			"device_id":              deviceID,
			"lois_settings":          map[string]interface{}{"lois_token": ""},
			"cloud_trust_token":      nil,
		},
		"server_params": map[string]interface{}{
			"is_from_logged_out":                0,
			"layered_homepage_experiment_group": nil,
			"device_id":                         deviceID,
			"login_surface":                     "unknown",
			"waterfall_id":                      waterfallID,
			"machine_id":                        machineID,
			"INTERNAL__latency_qpl_instance_id": latency,
			"flow_info":                         `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`,
			"is_platform_login":                 0,
			"reg_info":                          regInfo,
			"family_device_id":                  familyDevID,
			"offline_experiment_group":          "caa_iteration_v6_perf_fb_2",
			"access_flow_version":               "pre_mt_behavior",
			"is_from_logged_in_switcher":        0,
			"current_step":                      10,
		},
	}
	level2 := map[string]interface{}{"params": verifybase.MustJSON(level3)}
	paramsObj := map[string]interface{}{
		"params":              verifybase.MustJSON(level2),
		"bloks_versioning_id": spec.BloksVer,
		"app_id":              "com.bloks.www.bloks.caa.reg.resend_confirmation.async",
	}
	variablesJSON := verifybase.BuildVariables(paramsObj, spec.BloksVer, spec.StylesID, spec.IsPushOn)
	return verifybase.BuildFormBody(verifybase.ResendFriendlyName, spec.DocID, variablesJSON, traceID, locale)
}
