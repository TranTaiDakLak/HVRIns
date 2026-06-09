// steps.go — S556 verify: thin wrapper delegating to verifybase.RunVerify.
// All shared orchestration logic lives in verifybase. This file contains only
// s556-specific constants, UA validation/builder, body builders, 2FA, and addinfo wiring.
package s556

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
	verifyDocID     = "11994080422955588194694478490"
	verifyBloksVer  = "385fe019aa6b5903bdad3a4799063e3fc70da9cd1fda8b54189bce078c701665"
	defaultStylesID = "6100e7e89411ccf67ace027cedecd84f"
)

func verifyAccount(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	spec := verifybase.Spec{
		Tag:             "[s556 verify]",
		DocID:           verifyDocID,
		BloksVer:        verifyBloksVer,
		StylesID:        defaultStylesID,
		IsPushOn:        true,
		AddEmailTimeout: 15 * time.Second,
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
			return newUA, fmt.Sprintf("UA regenerated (FBAV/556, country=%s)", label)
		},
		SetupSessionCtx: func(sc *verifybase.SessionCtx) {
			verifybase.SetAppnetFields(sc)
		},
		BuildHeaders: func(sc *verifybase.SessionCtx, friendlyName string, withZeroState bool) [][2]string {
			reqIdx := verifybase.FriendlyNameToReqIdx(friendlyName)
			return verifybase.BuildLegacyHeaders(sc, friendlyName, reqIdx, withZeroState)
		},
		BuildAddEmailBody: buildAddEmailBody,
		BuildConfirmBody:  buildConfirmBody,
		BuildResendBody:   buildResendBody,
		Enable2FA:         enable2FAForS556,
		PostConfirm: func(ctx context.Context, sess *instagram.Session, cfg *instagram.VerifyConfig, notify func(string)) {
			if cfg.AddInfo != nil && cfg.AddInfo.Enabled {
				notify("[s556 verify] Running AddInfo...")
				res := addinfo.RunAddInfo(ctx, sess, cfg.AddInfo, notify)
				if len(res.Notes) > 0 {
					notify(fmt.Sprintf("[s556 verify] AddInfo done: %s", strings.Join(res.Notes, ", ")))
				}
			}
		},
	}
	return verifybase.RunVerify(ctx, session, cfg, outputPath, onStatus, spec)
}

func enable2FAForS556(ctx context.Context, session *instagram.Session, uid, machineID, deviceID string, emailOTPFn func(string, int) string, notify func(string)) (string, error) {
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
	notify(fmt.Sprintf("[s556 verify] 2FA secret generated: %s", res.Secret))
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

// ─── Body builders (legacy style: GRANTED perms, device_network_info) ────────

func buildRegInfo(emailAddr, uid, firstName, lastName, deviceID, familyDevID string, gender int, sim fakeinfo.SimProfile) string {
	fullName := firstName + " " + lastName
	if lastName == "" {
		fullName = firstName
	}
	hni, _ := strconv.Atoi(sim.HNI)
	if hni == 0 {
		hni = 45204
	}
	simInfo := map[string]interface{}{
		"is_mobile_data_enabled": false,
		"sim_operator":           hni,
		"is_gsm_roaming":         false,
		"group_id_level_1":       "00000000",
		"sim_operator_name":      sim.OperatorName,
		"sim_state":              5,
		"network_operator":       hni,
		"is_data_roaming":        false,
		"is_esim":                false,
		"is_sim_sms_capable":     true,
		"sim_carrier_id":         1899,
		"signal_strength":        2,
		"network_type":           13,
		"sim_carrier_id_name":    sim.OperatorName,
	}
	deviceNetworkInfo := map[string]interface{}{
		"default_subscription_info":  simInfo,
		"sim_count":                  2,
		"is_wifi":                    true,
		"is_airplane_mode":           false,
		"is_active_network_cellular": false,
		"is_device_sms_capable":      true,
		"active_subscriptions_info":  []interface{}{simInfo},
	}
	ri := map[string]interface{}{
		"contactpoint":             emailAddr,
		"contactpoint_type":        "email",
		"encrypted_msisdn":         "",
		"headers_last_infra_flow_id": "",
		"flash_call_permissions_status": map[string]interface{}{
			"READ_PHONE_STATE":   "GRANTED",
			"CALL_PHONE":         "GRANTED",
			"READ_CALL_LOG":      "GRANTED",
			"ANSWER_PHONE_CALLS": "GRANTED",
		},
		"first_name":                              firstName,
		"last_name":                               lastName,
		"full_name":                               fullName,
		"is_using_unified_cp":                     false,
		"is_cp_claimed":                           false,
		"age_range":                               "o18",
		"gender":                                  gender,
		"device_id":                               deviceID,
		"family_device_id":                        familyDevID,
		"user_id":                                 uid,
		"profile_photo":                           nil,
		"whatsapp_installed_on_client":            false,
		"email_prefilled":                         false,
		"conf_allow_back_nav_after_change_nav":    true,
		"gms_incoming_call_retriever_eligibility": "eligible",
		"attestation_result":                      map[string]interface{}{},
		"screen_visited":                          []interface{}{"CAA_REG_CONFIRMATION_SCREEN"},
		"suma_on_conf_threshold":                  0.8,
		"device_network_info":                     deviceNetworkInfo,
	}
	return verifybase.MustJSON(ri)
}

func buildRegInfoForConfirm(emailAddr, uid, deviceID, familyDevID string, sim fakeinfo.SimProfile) string {
	hni, _ := strconv.Atoi(sim.HNI)
	if hni == 0 {
		hni = 45204
	}
	simInfo := map[string]interface{}{
		"is_mobile_data_enabled": false,
		"sim_operator":           hni,
		"is_gsm_roaming":         false,
		"group_id_level_1":       "00000000",
		"sim_operator_name":      sim.OperatorName,
		"sim_state":              5,
		"network_operator":       hni,
		"is_data_roaming":        false,
		"is_esim":                false,
		"is_sim_sms_capable":     true,
		"sim_carrier_id":         1899,
		"signal_strength":        2,
		"network_type":           13,
		"sim_carrier_id_name":    sim.OperatorName,
	}
	deviceNetworkInfo := map[string]interface{}{
		"default_subscription_info":  simInfo,
		"sim_count":                  2,
		"is_wifi":                    true,
		"is_airplane_mode":           false,
		"is_active_network_cellular": false,
		"is_device_sms_capable":      true,
		"active_subscriptions_info":  []interface{}{simInfo},
	}
	ri := map[string]interface{}{
		"contactpoint":             emailAddr,
		"contactpoint_type":        "email",
		"encrypted_msisdn":         "",
		"headers_last_infra_flow_id": "",
		"flash_call_permissions_status": map[string]interface{}{
			"READ_PHONE_STATE":   "DENIED",
			"CALL_PHONE":         "DENIED",
			"READ_CALL_LOG":      "DENIED",
			"ANSWER_PHONE_CALLS": "DENIED",
		},
		"first_name":                              nil,
		"last_name":                               nil,
		"full_name":                               nil,
		"is_using_unified_cp":                     false,
		"is_cp_claimed":                           false,
		"age_range":                               nil,
		"gender":                                  nil,
		"birthday":                                nil,
		"device_id":                               deviceID,
		"family_device_id":                        familyDevID,
		"user_id":                                 uid,
		"profile_photo":                           nil,
		"whatsapp_installed_on_client":            false,
		"email_prefilled":                         false,
		"conf_allow_back_nav_after_change_nav":    true,
		"gms_incoming_call_retriever_eligibility": "eligible",
		"attestation_result":                      map[string]interface{}{},
		"screen_visited":                          []interface{}{"CAA_REG_CONFIRMATION_SCREEN", "CAA_REG_CONFIRMATION_SCREEN"},
		"suma_on_conf_threshold":                  0.8,
		"device_network_info":                     deviceNetworkInfo,
	}
	return verifybase.MustJSON(ri)
}

func buildAddEmailBody(spec *verifybase.Spec, emailAddr, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string {
	traceID := uuid.New().String()
	eventReqID := uuid.New().String()
	latency := int64(80000000000000 + mrand.Int63n(9000000000000))
	bssid := verifybase.NetworkBSSID()
	regInfo := buildRegInfo(emailAddr, uid, firstName, lastName, deviceID, familyDevID, gender, sim)

	level3 := map[string]interface{}{
		"client_input_params": map[string]interface{}{
			"aac":                          "",
			"device_id":                    deviceID,
			"zero_balance_state":           "init",
			"msg_previous_cp":              "",
			"switch_cp_first_time_loading": 1,
			"has_rejected_rel":             0,
			"accounts_list":                []interface{}{},
			"email_prefilled":              0,
			"confirmed_cp_and_code":        map[string]interface{}{},
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
			"network_bssid":                     bssid,
			"machine_id":                        machineID,
			"INTERNAL__latency_qpl_instance_id": latency + mrand.Int63n(500),
			"INTERNAL__latency_qpl_marker_id":   36707139,
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
	bssid := verifybase.NetworkBSSID()
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
		},
		"server_params": map[string]interface{}{
			"event_request_id":                  eventReqID,
			"is_from_logged_out":                0,
			"text_input_id":                     latency,
			"layered_homepage_experiment_group": nil,
			"device_id":                         deviceID,
			"login_surface":                     "unknown",
			"waterfall_id":                      waterfallID,
			"network_bssid":                     bssid,
			"wa_timer_id":                       "wa_retriever",
			"machine_id":                        machineID,
			"INTERNAL__latency_qpl_instance_id": latency + mrand.Int63n(500),
			"INTERNAL__latency_qpl_marker_id":   36707139,
			"flow_info":                         `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`,
			"is_platform_login":                 0,
			"sms_retriever_started_prior_step":  0,
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
	bssid := verifybase.NetworkBSSID()
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
			"network_bssid":                     bssid,
			"machine_id":                        machineID,
			"INTERNAL__latency_qpl_instance_id": latency,
			"INTERNAL__latency_qpl_marker_id":   36707139,
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
