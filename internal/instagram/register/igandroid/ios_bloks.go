// ios_bloks.go — iOS Bloks registerer.
// Dùng /graphql_www + GraphQL variables format như iOS IG app thật,
// KHÔNG dùng /api/v1/bloks/async_action/ format của Android.
package igandroid

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"HVRIns/internal/igcore"
	"HVRIns/internal/instagram"
	"HVRIns/internal/proxy"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
)

// regJurisdictionRe trích xuất regulation_jurisdiction từ system_error response của IG.
// IG trả về bloks action chứa: \"COUNTRYCODE\" \"system_error\" → ta parse COUNTRYCODE.
var regJurisdictionRe = regexp.MustCompile(`\\"([A-Z]{2,3})\\" \\"system_error\\"`)

const (
	igIOSAppID       = "124024574287414"
	igIOSBloksVer    = "bbabb80f3de1e25c0c5c0bbfd6cae893124649276a36ead33328a4ea03a34b75"
	igIOSCaps        = "36r/F/8="
	igIOSClientDocID = "38523300859713187485104132294" // constant across tất cả steps
)

// igIOSVersions — Instagram iOS app versions (AppVer + Build cho User-Agent).
// Mở rộng từ 5 → 13 version để đa dạng UA (tránh pattern bot khi reg hàng loạt).
// Build ID lấy từ pool dự án sẵn có (igcore.igAppVersionPool + bộ ios_bloks gốc);
// IG reg KHÔNG validate version↔build nên mix an toàn — chỉ là chuỗi UA cho analytics.
var igIOSVersions = []struct{ AppVer, Build string }{
	{"435.0.0.25.108", "742613068"},
	{"428.0.0.26.88", "732266754"},
	{"420.0.0.45.97", "719879636"},
	{"416.0.0.37.86", "712543085"},
	{"410.1.0.36.70", "849447290"},
	{"407.0.0.24.99", "843437295"},
	{"405.0.0.24.95", "839437284"},
	{"402.0.0.21.80", "833337280"},
	{"400.0.0.32.126", "695501093"},
	{"399.0.0.18.73", "827437270"},
	{"390.0.0.31.136", "676625680"},
	{"380.0.0.24.127", "661064929"},
	{"374.0.0.21.121", "651580628"},
}

// igIOSDevices — iPhone models for UA.
var igIOSDevices = []struct{ Model, Screen, Scale string }{
	{"iPhone16,2", "1290x2796", "3.00"}, // iPhone 15 Pro Max
	{"iPhone15,5", "1290x2796", "3.00"}, // iPhone 14 Pro Max
	{"iPhone15,3", "1290x2796", "3.00"}, // iPhone 13 Pro Max
	{"iPhone14,8", "1284x2778", "3.00"}, // iPhone 12 Pro Max
	{"iPhone14,3", "1284x2778", "3.00"}, // iPhone 13 Pro Max
	{"iPhone13,4", "1284x2778", "3.00"}, // iPhone 12 Pro Max
	{"iPhone13,2", "1170x2532", "3.00"}, // iPhone 12
	{"iPhone12,1", "828x1792", "2.00"},  // iPhone 11
	{"iPhone11,8", "828x1792", "2.00"},  // iPhone XR
	{"iPhone9,1", "750x1334", "2.00"},   // iPhone 7
}

// igIOSSystems — iOS versions.
var igIOSSystems = []string{
	"18_3_2", "18_2_1", "17_7_4", "17_6_1", "17_5_1", "16_7_10", "15_8_4",
}

// iosLocale chứa thông tin locale/timezone theo quốc gia của proxy IP.
type iosLocale struct {
	Locale         string // vd "en_US"
	LangCode       string // vd "en"
	LangHeader     string // vd "en-US" (dùng trong accept-language / x-ig-device-locale)
	MappedLocale   string // vd "en_US" (x-ig-mapped-locale)
	Timezone       string // vd "America/New_York"
	TzOffset       string // vd "-18000" (seconds from UTC)
}

// localeFromCountry map country code (2 chữ hoa từ CheckProxyCountry) → iosLocale.
// Default en_US nếu không map được.
func localeFromCountry(cc string) iosLocale {
	switch cc {
	case "US":
		return iosLocale{"en_US", "en", "en-US", "en_US", "America/New_York", "-18000"}
	case "GB":
		return iosLocale{"en_GB", "en", "en-GB", "en_GB", "Europe/London", "0"}
	case "AU":
		return iosLocale{"en_AU", "en", "en-AU", "en_AU", "Australia/Sydney", "36000"}
	case "CA":
		return iosLocale{"en_CA", "en", "en-CA", "en_CA", "America/Toronto", "-18000"}
	case "DE":
		return iosLocale{"de_DE", "de", "de-DE", "de_DE", "Europe/Berlin", "3600"}
	case "FR":
		return iosLocale{"fr_FR", "fr", "fr-FR", "fr_FR", "Europe/Paris", "3600"}
	case "NL":
		return iosLocale{"nl_NL", "nl", "nl-NL", "nl_NL", "Europe/Amsterdam", "3600"}
	case "SG":
		return iosLocale{"en_SG", "en", "en-SG", "en_SG", "Asia/Singapore", "28800"}
	case "JP":
		return iosLocale{"ja_JP", "ja", "ja-JP", "ja_JP", "Asia/Tokyo", "32400"}
	case "TR":
		return iosLocale{"tr_TR", "tr", "tr-TR", "tr_TR", "Europe/Istanbul", "10800"}
	case "MY":
		return iosLocale{"ms_MY", "ms", "ms-MY", "ms_MY", "Asia/Kuala_Lumpur", "28800"}
	case "ID":
		return iosLocale{"id_ID", "id", "id-ID", "id_ID", "Asia/Jakarta", "25200"}
	case "TH":
		return iosLocale{"th_TH", "th", "th-TH", "th_TH", "Asia/Bangkok", "25200"}
	case "PH":
		return iosLocale{"en_PH", "en", "en-PH", "en_PH", "Asia/Manila", "28800"}
	case "IN":
		return iosLocale{"en_IN", "en", "en-IN", "en_IN", "Asia/Kolkata", "19800"}
	case "VN":
		return iosLocale{"vi_VN", "vi", "vi-VN", "vi_VN", "Asia/Ho_Chi_Minh", "25200"}
	default:
		return iosLocale{"en_US", "en", "en-US", "en_US", "America/New_York", "-18000"}
	}
}

func randomIOSUA(locale string) string {
	v := igIOSVersions[rand.Intn(len(igIOSVersions))]
	d := igIOSDevices[rand.Intn(len(igIOSDevices))]
	s := igIOSSystems[rand.Intn(len(igIOSSystems))]
	lang := "en"
	if len(locale) >= 2 {
		lang = locale[:2]
	}
	return fmt.Sprintf(
		"Instagram %s (%s; iOS %s; %s; %s; scale=%s; %s; %s) AppleWebKit/420+",
		v.AppVer, d.Model, s, locale, lang, d.Scale, d.Screen, v.Build,
	)
}

// iosEngine bọc igAndroidEngine với iOS-specific session state.
// Tách ra để không làm ô nhiễm igAndroidEngine bằng field iOS-specific.
type iosEngine struct {
	*igAndroidEngine
	cloudTrustToken string    // 2 UUID uppercase ghép liền — dùng nhất quán trong header + body
	usdID           string    // X-Meta-USDID — ổn định suốt session (như real iOS device)
	locale          iosLocale // locale/timezone theo country của proxy IP
}

// newIOSEngine khởi tạo iosEngine với state ngẫu nhiên per-session.
func newIOSEngine(base *igAndroidEngine) *iosEngine {
	// Cloud Trust Token: 2 UUID uppercase ghép trực tiếp (không separator giữa 2 UUID)
	// Ví dụ thật: "4A3B0992-83A9-4EA5-BDF0-C2600A6E3828295E487A-B465-4F17-8F3E-2C6B4DF775FC"
	ctt := strings.ToUpper(uuid.New().String()) + strings.ToUpper(uuid.New().String())

	// X-Meta-USDID: {deviceID}.{unix_ts}.{sig}
	// Real iOS dùng ECDSA P-256 từ Apple Secure Enclave — ta tạo chuỗi ngẫu nhiên cùng độ dài.
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	var rawSig [64]byte
	for i := range rawSig {
		rawSig[i] = byte(rand.Intn(256))
	}
	sig := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(rawSig[:])
	usdID := strings.ToUpper(base.p.DeviceID) + "." + ts + "." + sig

	return &iosEngine{
		igAndroidEngine: base,
		cloudTrustToken: ctt,
		usdID:           usdID,
	}
}

// iosCommonServerParams trả về server_params chuẩn iOS — override các giá trị Android sai.
// Real iOS dùng offline_experiment_group="caa_launch_ig", layered_homepage="default_control".
// login_surface/login_entry_point đổi sang "reg_existing_login" từ step 6 (birthday) trở đi.
func (eng *iosEngine) iosCommonServerParams(step int) map[string]any {
	sp := eng.commonServerParams(step) // base từ Android
	sp["offline_experiment_group"] = "caa_launch_ig"
	sp["layered_homepage_experiment_group"] = "default_control"
	sp["cloud_trust_token"] = eng.cloudTrustToken
	// device_id trong sp phải là UUID (iOS), không phải "android-HEX"
	sp["device_id"] = eng.p.DeviceID
	if step >= 6 {
		sp["login_surface"] = "reg_existing_login"
		sp["login_entry_point"] = "reg_existing_login"
	}
	return sp
}

// newIOSSession tạo TLS session với Safari_IOS_17_0 để khớp với iOS headers và caa_launch_ig.
// Dùng Android TLS (Okhttp4Android13) kết hợp caa_launch_ig sẽ bị IG block ngay tại submitEmail.
func newIOSSession(proxyStr string) (*androidSession, error) {
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(60),
		tls_client.WithClientProfile(profiles.Safari_IOS_17_0),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		tls_client.WithInsecureSkipVerify(),
		tls_client.WithNotFollowRedirects(),
	}
	if proxyStr != "" {
		if f := proxy.FormatProxyURL(proxyStr); f != "" {
			opts = append(opts, tls_client.WithProxyUrl(f))
		}
	}
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return nil, fmt.Errorf("create ios tls client: %w", err)
	}
	zr, _ := zstd.NewReader(nil)
	return &androidSession{client: c, zr: zr}, nil
}

// patchRegInfoForIOS sửa các trường reg_info không đúng cho iOS (so với Android defaults):
//   - device_id: UUID (p.DeviceID), không phải "android-HEX" (p.AndroidID)
//   - ig4a_qe_device_id: null (iOS không set field này)
//   - caa_reg_flow_source: "nta_native_integration_point" (iOS NTA flow)
//   - skip_slow_rel_check: false (iOS default)
func patchRegInfoForIOS(regInfoJSON string, p *androidProfile) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(regInfoJSON), &m); err != nil {
		return regInfoJSON
	}
	m["device_id"] = p.DeviceID
	m["ig4a_qe_device_id"] = nil
	m["caa_reg_flow_source"] = "nta_native_integration_point"
	m["skip_slow_rel_check"] = false
	b, err := json.Marshal(m)
	if err != nil {
		return regInfoJSON
	}
	return string(b)
}

// iosTimestamp trả về pigeon-rawclienttime format iOS: "1781142963.309754" (giây.microseconds).
func iosTimestamp() string {
	nowMicro := time.Now().UnixMicro()
	secs := nowMicro / 1_000_000
	micros := nowMicro % 1_000_000
	return fmt.Sprintf("%d.%06d", secs, micros)
}

// iosHeaders trả về headers khớp với iOS IG app thật (graphql_www requests).
// friendlyName là giá trị X-FB-Friendly-Name, ví dụ "IGBloksAppRootQuery-com.bloks.www..."
func iosHeaders(p *androidProfile, friendlyName, cloudToken, usdID string, loc iosLocale) [][2]string {
	analyticsTag := `{"network_tags":{"product":"` + igIOSAppID +
		`","surface":"other","is_ad":"0","request_category":"api","purpose":"fetch","retry_attempt":"0"},` +
		`"application_tags":{"is_nav_critical":"0"}}`
	return [][2]string{
		{"user-agent", p.UserAgent},
		{"accept", "*/*"},
		{"accept-language", loc.LangHeader + ";q=1.0"},
		{"accept-encoding", "zstd"},
		{"content-type", "text/javascript; charset=utf-8"},
		{"ig-intended-user-id", "0"},
		{"x-fb-request-analytics-tags", analyticsTag},
		{"x-meta-usdid", usdID},
		{"x-fb-friendly-name", friendlyName},
		{"x-ig-bandwidth-speed-kbps", "0.000"},
		{"x-fb-connection-type", "wifi"},
		{"x-fb-rmd", "state=URL_ELIGIBLE"},
		{"x-ig-connection-type", "WiFi"},
		{"x-tigon-is-retry", "False"},
		{"x-fb-client-ip", "True"},
		{"x-fb-http-engine", "Tigon/MNS/mvfst-mobile"},
		{"x-fb-conn-uuid-client", p.ConnUUID},
		{"x-fb-server-cluster", "True"},
		{"x-ig-family-device-id", p.FamilyDeviceID},
		{"x-bloks-prism-extended-palette-gray", "false"},
		{"x-ig-connection-speed", "6kbps"},
		{"x-pigeon-rawclienttime", iosTimestamp()},
		{"x-ig-abr-connection-speed-kbps", "0"},
		{"x-ig-bloks-serialize-payload", "true"},
		{"x-bloks-prism-extended-palette-indigo", "false"},
		{"x-mid", p.MachineID},
		{"x-bloks-prism-extended-palette-polish-enabled", "false"},
		{"x-ig-app-id", igIOSAppID},
		{"x-bloks-prism-font-enabled", "false"},
		{"x-ig-capabilities", igIOSCaps},
		{"x-ig-mapped-locale", loc.MappedLocale},
		{"x-ig-app-locale", loc.LangCode},
		{"x-ig-device-locale", loc.LangHeader},
		{"x-bloks-prism-extended-palette-red", "false"},
		{"x-bloks-version-id", igIOSBloksVer},
		{"x-bloks-prism-ax-base-colors-enabled", "true"},
		{"x-cloud-trust-token", cloudToken},
		{"x-ig-timezone-offset", loc.TzOffset},
		{"x-bloks-prism-colors-enabled", "true"},
		{"x-bloks-prism-extended-palette-rest-of-colors", "false"},
		{"x-bloks-is-prism-enabled", "false"},
		{"x-pigeon-session-id", p.PigeonSID},
		{"x-ig-device-id", p.DeviceID},
		{"x-graphql-request-purpose", "fetch"},
		{"x-graphql-client-library", "pando"},
		{"x-stack", "distillery"},
		{"x-ig-www-claim", "0"},
		{"x-bloks-prism-link-colors-enabled", "0"},
	}
}

// buildIOSGQLBody xây POST body cho /graphql_www theo iOS format.
//
// Cấu trúc đúng từ capture real iOS (4 lớp JSON):
//
//	variables = {bk_context, params: {params: JSON(L3)}}   ← chỉ 2 field, không có app_id/bloks_versioning_id
//	L3 = {params: JSON(L4)}                                 ← không có client_input_params ở L3
//	L4 = {server_params: sp, client_input_params: cip}      ← client_input_params NẰM TRONG L4
func buildIOSGQLBody(endpoint string, cip, sp map[string]any, p *androidProfile, step int, locale string) (string, error) {
	sp["offline_experiment_group"] = "caa_launch_ig"
	sp["current_step"] = step
	sp["family_device_id"] = p.FamilyDeviceID

	l4 := map[string]any{
		"server_params":    sp,
		"client_input_params": cip,
	}
	l4JSON, err := json.Marshal(l4)
	if err != nil {
		return "", fmt.Errorf("marshal l4: %w", err)
	}

	l3 := map[string]any{"params": string(l4JSON)}
	l3JSON, err := json.Marshal(l3)
	if err != nil {
		return "", fmt.Errorf("marshal l3: %w", err)
	}

	variables := map[string]any{
		"bk_context": map[string]any{
			"pixel_ratio": 2,
			"styles_id":   "instagram",
			"theme_params": []any{
				map[string]any{
					"design_system_name": "XMDS",
					"value":             []string{"three_neutral_gray"},
				},
			},
		},
		"params": map[string]any{"params": string(l3JSON)},
	}
	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return "", fmt.Errorf("marshal variables: %w", err)
	}

	return "method=post&pretty=false&format=json&server_timestamps=true" +
		"&locale=" + locale + "&purpose=fetch" +
		"&fb_api_req_friendly_name=" + url.QueryEscape("IGBloksAppRootQuery-"+endpoint) +
		"&client_doc_id=" + igIOSClientDocID +
		"&enable_canonical_naming=true" +
		"&enable_canonical_variable_overrides=true" +
		"&enable_canonical_naming_ambiguous_type_prefixing=true" +
		"&variables=" + url.QueryEscape(string(variablesJSON)), nil
}

// buildCreateAccountGQLBody xây body graphql_www cho bước createAccount theo cấu trúc iOS thật.
// createAccount khác các bước khác: reg_info, current_step, offline_experiment_group, family_device_id
// nằm ở top-level L4 (không trong server_params). server_params chỉ chứa các field iOS-specific.
func buildCreateAccountGQLBody(eng *iosEngine) (string, error) {
	const endpoint = "com.bloks.www.bloks.caa.reg.create.account.async"

	sp := map[string]any{
		"INTERNAL__latency_qpl_marker_id":   36707139,
		"INTERNAL__latency_qpl_instance_id": rand.Int63n(9e14) + 1e14,
		"event_request_id":                  uuid.New().String(),
		"login_surface":                     "reg_existing_login",
		"is_from_logged_in_switcher":        0,
		"is_platform_login":                 0,
		"access_flow_version":               "pre_mt_behavior",
		"cloud_trust_token":                 eng.cloudTrustToken,
		"flow_info":                         `{"flow_name":"new_to_family_ig_default","flow_type":"ntf"}`,
		"login_entry_point":                 "reg_existing_login",
		"device_id":                         eng.p.DeviceID,
		"waterfall_id":                      eng.p.WaterfallID,
		"is_from_logged_out":                0,
		"layered_homepage_experiment_group": "default_control",
		"reg_context":                       eng.regContext,
	}

	cip := map[string]any{
		"aac":                                    buildAACJSON(eng.p),
		"ig_partially_created_account_user_id":   0,
		"machine_id":                             eng.p.MachineID,
		"ck_id":                                  nil,
		"zero_balance_state":                     "",
		"ck_nonce":                               nil,
		"ig_partially_created_account_nonce":     "",
		"encrypted_msisdn":                       "",
		"network_bssid":                          nil,
		"waterfall_id":                           eng.p.WaterfallID,
		"failed_birthday_year_count":             "",
		"ck_error":                               "CKErrorDomain: 9",
		"no_contact_perm_email_oauth_token":      "",
		"lois_settings":                          map[string]any{"lois_token": ""},
		"reached_from_tos_screen":                1,
		"device_id":                              eng.p.DeviceID,
		"ig_partially_created_account_nonce_expiry": 0,
		"headers_last_infra_flow_id":             "",
	}

	l4 := map[string]any{
		"server_params":           sp,
		"reg_info":                buildRegInfoJSON(eng.p, eng.state),
		"current_step":            9,
		"bloks_controller_source": "bk_caa_reg_icon_text_list_tos_screen",
		"offline_experiment_group": "caa_launch_ig",
		"family_device_id":        eng.p.FamilyDeviceID,
		"client_input_params":     cip,
	}
	l4JSON, err := json.Marshal(l4)
	if err != nil {
		return "", fmt.Errorf("marshal l4: %w", err)
	}

	l3 := map[string]any{"params": string(l4JSON)}
	l3JSON, err := json.Marshal(l3)
	if err != nil {
		return "", fmt.Errorf("marshal l3: %w", err)
	}

	vars := map[string]any{
		"bk_context": map[string]any{
			"pixel_ratio": 2,
			"styles_id":   "instagram",
			"theme_params": []any{
				map[string]any{
					"design_system_name": "XMDS",
					"value":             []string{"three_neutral_gray"},
				},
			},
		},
		"params": map[string]any{"params": string(l3JSON)},
	}
	varsJSON, err := json.Marshal(vars)
	if err != nil {
		return "", fmt.Errorf("marshal vars: %w", err)
	}

	return "method=post&pretty=false&format=json&server_timestamps=true" +
		"&locale=" + eng.locale.Locale + "&purpose=fetch" +
		"&fb_api_req_friendly_name=" + url.QueryEscape("IGBloksAppRootQuery-"+endpoint) +
		"&client_doc_id=" + igIOSClientDocID +
		"&enable_canonical_naming=true" +
		"&enable_canonical_variable_overrides=true" +
		"&enable_canonical_naming_ambiguous_type_prefixing=true" +
		"&variables=" + url.QueryEscape(string(varsJSON)), nil
}

// buildIOSAsyncBody — POST body cho /api/v1/bloks/async_action/ với iOS bloks version.
// Format giống Android nhưng dùng igIOSBloksVer.
func buildIOSAsyncBody(cip, sp map[string]any) string {
	params := map[string]any{
		"client_input_params": cip,
		"server_params":       sp,
	}
	bkCtx := map[string]any{
		"bloks_version": igIOSBloksVer,
		"styles_id":     "instagram",
	}
	paramsJSON, _ := json.Marshal(params)
	bkCtxJSON, _ := json.Marshal(bkCtx)
	return "params=" + url.QueryEscape(string(paramsJSON)) +
		"&bk_client_context=" + url.QueryEscape(string(bkCtxJSON)) +
		"&bloks_versioning_id=" + igIOSBloksVer
}

// iosAsyncPost gửi request đến /api/v1/bloks/async_action/{endpoint}/ với iOS headers + iOS params.
// Dùng async_action (Android format) nhưng với iOS bloks version và iOS-specific headers.
// Giữ offline_experiment_group Android để server route đúng handler — tránh mismatch khi
// dùng async_action nhưng server nghĩ là iOS graphql_www flow.
func iosAsyncPost(ctx context.Context, eng *iosEngine, endpoint string, cip, sp map[string]any, extraHdr [][2]string) (string, error) {

	body := buildIOSAsyncBody(cip, sp)

	hdr := iosHeaders(eng.p, endpoint, eng.cloudTrustToken, eng.usdID, eng.locale)
	// async_action dùng application/x-www-form-urlencoded (không phải text/javascript)
	for i, h := range hdr {
		if h[0] == "content-type" {
			hdr[i][1] = "application/x-www-form-urlencoded"
			break
		}
	}
	if len(extraHdr) > 0 {
		hdr = append(hdr, extraHdr...)
	}

	endpointURL := igAndroidBase + "/api/v1/bloks/async_action/" + endpoint + "/"
	resp, _, err := eng.sess.post(ctx, endpointURL, body, hdr)

	if dir := debugDir(); dir != "" {
		parts := strings.Split(endpoint, ".")
		var stepName string
		if len(parts) >= 2 {
			stepName = parts[len(parts)-2] + "_" + parts[len(parts)-1]
		} else {
			stepName = endpoint
		}
		writeDebug(dir, "ios_req_"+stepName+".txt", body)
		writeDebug(dir, "ios_resp_"+stepName+".txt", resp)
	}
	if err != nil {
		return resp, err
	}
	if rc := igcore.ParseRegContext(resp); rc != "" {
		eng.regContext = rc
	}
	return resp, nil
}

// iosGQLPost gửi request đến /graphql_www — thay thế hoàn toàn /api/v1/bloks/async_action/.
func iosGQLPost(ctx context.Context, eng *iosEngine, endpoint string, cip, sp map[string]any, step int, extraHdr [][2]string) (string, error) {
	sp["cloud_trust_token"] = eng.cloudTrustToken

	body, err := buildIOSGQLBody(endpoint, cip, sp, eng.p, step, eng.locale.Locale)
	if err != nil {
		return "", fmt.Errorf("iosGQLPost: %w", err)
	}

	friendlyName := "IGBloksAppRootQuery-" + endpoint
	hdr := iosHeaders(eng.p, friendlyName, eng.cloudTrustToken, eng.usdID, eng.locale)
	if len(extraHdr) > 0 {
		hdr = append(hdr, extraHdr...)
	}

	resp, _, err := eng.sess.post(ctx, igAndroidBase+"/graphql_www", body, hdr)

	if dir := debugDir(); dir != "" {
		parts := strings.Split(endpoint, ".")
		var stepName string
		if len(parts) >= 2 {
			stepName = parts[len(parts)-2] + "_" + parts[len(parts)-1]
		} else {
			stepName = endpoint
		}
		writeDebug(dir, "ios_gql_req_"+stepName+".txt", body)
		writeDebug(dir, "ios_gql_resp_"+stepName+".txt", resp)
	}
	if err != nil {
		return resp, err
	}
	// Phát hiện lỗi GraphQL trả về từ server (thay vì chạy tiếp khi thật ra đã fail).
	if strings.Contains(resp, `"errors":[`) {
		return resp, fmt.Errorf("graphql error: %.200s", resp)
	}
	if rc := igcore.ParseRegContext(resp); rc != "" {
		eng.regContext = rc
	}
	return resp, nil
}

// iosExposeNTM là bước warm-up bắt buộc của iOS — gọi trước bất kỳ /graphql_www nào.
// POST /api/v1/bloks/async_action/...expose_ntm_experiment.async/ với body rỗng.
// Friendly-name trong [665] capture: nta_landing_logging.async (khác endpoint URL).
func iosExposeNTM(ctx context.Context, eng *iosEngine) {
	const (
		ntmURL      = igAndroidBase + "/api/v1/bloks/async_action/com.bloks.www.bloks.caa.reg.async.expose_ntm_experiment.async/"
		ntmFriendly = "IGBloksAppRootQuery-com.bloks.www.bloks.caa.reg.nta_landing_logging.async"
	)
	hdr := iosHeaders(eng.p, ntmFriendly, eng.cloudTrustToken, eng.usdID, eng.locale)
	// async_action dùng application/x-www-form-urlencoded, không phải text/javascript
	for i, h := range hdr {
		if h[0] == "content-type" {
			hdr[i][1] = "application/x-www-form-urlencoded"
			break
		}
	}
	resp, _, _ := eng.sess.post(ctx, ntmURL, "", hdr)
	if dir := debugDir(); dir != "" {
		writeDebug(dir, "ios_warmup_resp.txt", resp)
	}
}

// ── Registration steps ────────────────────────────────────────────────────────

func iosSubmitEmail(ctx context.Context, eng *iosEngine, addr string) error {
	eng.state.ScreenVisited = append(eng.state.ScreenVisited,
		"CAA_REG_CONTACT_POINT_PHONE", "CAA_REG_CONTACT_POINT_EMAIL")
	eng.state.ContactPoint = addr
	cip := map[string]any{
		"aac":                          buildAACJSON(eng.p),
		"email":                        addr,
		"device_id":                    eng.p.DeviceID,
		"family_device_id":             eng.p.FamilyDeviceID,
		"confirmed_cp_and_code":        map[string]any{},
		"accounts_list":                []any{},
		"fb_ig_device_id":              []any{},
		"lois_settings":                map[string]any{"lois_token": ""},
		"cloud_trust_token":            eng.cloudTrustToken,
		"network_bssid":                nil,
		"zero_balance_state":           "",
		"msg_previous_cp":              "",
		"block_store_machine_id":       "",
		"switch_cp_first_time_loading": 1,
		"switch_cp_have_seen_suma":     0,
		"has_rejected_rel":             0,
		"seen_login_upsell":            0,
		"email_prefilled":              0,
		"is_from_device_emails":        0,
	}
	sp := eng.iosCommonServerParams(0)
	sp["reg_info"] = patchRegInfoForIOS(buildRegInfoJSON(eng.p, eng.state), eng.p)
	sp["cp_funnel"] = 0
	sp["cp_source"] = 0
	sp["text_input_id"] = randQplID()
	resp, err := iosAsyncPost(ctx, eng, "com.bloks.www.bloks.caa.reg.async.contactpoint_email.async", cip, sp, nil)
	if err != nil {
		return err
	}
	if strings.Contains(resp, "USER_REGISTER_INVALID_EMAIL") ||
		strings.Contains(resp, "restrict certain activity") ||
		strings.Contains(resp, "is_email_valid\",false") {
		return fmt.Errorf("submitEmail blocked by IG: email/IP restricted")
	}
	return nil
}

func iosConfirmOTP(ctx context.Context, eng *iosEngine, addr, otp string) error {
	eng.state.ScreenVisited = append(eng.state.ScreenVisited, "CAA_REG_CONFIRMATION_SCREEN")
	cip := map[string]any{
		"code":                  otp,
		"confirmed_cp_and_code": map[string]any{},
		"aac":                   buildAACJSON(eng.p),
		"cloud_trust_token":     eng.cloudTrustToken,
		"device_id":             eng.p.DeviceID,
		"family_device_id":      eng.p.FamilyDeviceID,
		"fb_ig_device_id":       []any{},
		"block_store_machine_id": "",
		"lois_settings":         map[string]any{"lois_token": ""},
		"network_bssid":         nil,
	}
	sp := eng.iosCommonServerParams(3)
	sp["reg_info"] = patchRegInfoForIOS(buildRegInfoJSON(eng.p, eng.state), eng.p)
	resp, err := iosAsyncPost(ctx, eng, "com.bloks.www.bloks.caa.reg.confirmation.async", cip, sp, nil)
	if err != nil {
		return err
	}
	if cc := igcore.ParseConfirmationCode(resp); cc != "" {
		eng.state.ConfirmationCode = cc
	}
	return nil
}

func iosSetPassword(ctx context.Context, eng *iosEngine, addr, password string) error {
	eng.state.ScreenVisited = append(eng.state.ScreenVisited, "CAA_REG_PASSWORD")
	encPwd, err := igcore.EncryptPassword(password, eng.pubKey, eng.keyID)
	if err != nil {
		return fmt.Errorf("encrypt password: %w", err)
	}
	eng.state.EncryptedPassword = encPwd
	savePwd := true
	eng.state.ShouldSavePassword = &savePwd
	cip := map[string]any{
		"encrypted_password": encPwd,
		"machine_id":         eng.p.MachineID,
		"spi_action":         1,
		"aac":                buildAACJSON(eng.p),
	}
	sp := eng.iosCommonServerParams(4)
	sp["reg_info"] = patchRegInfoForIOS(buildRegInfoJSON(eng.p, eng.state), eng.p)
	_, err = iosAsyncPost(ctx, eng, "com.bloks.www.bloks.caa.reg.password.async", cip, sp, nil)
	return err
}

func iosBirthday(ctx context.Context, eng *iosEngine) error {
	year := 1990 + randIntn(11)
	month := 1 + randIntn(12)
	day := 1 + randIntn(28)
	bday := fmt.Sprintf("%02d-%02d-%04d", day, month, year)
	bdayTS := birthdayUnix(year, month, day)
	eng.state.Birthday = bday
	eng.state.AgeRange = "o18"
	skipYouth := true
	eng.state.ShouldSkipYouthTOS = &skipYouth
	eng.state.ScreenVisited = append(eng.state.ScreenVisited, "bloks.caa.reg.birthday")
	cip := map[string]any{
		"accounts_list":                   []any{},
		"client_timezone":                 eng.locale.Timezone,
		"aac":                             buildAACJSON(eng.p),
		"birthday_or_current_date_string": bday,
		"os_age_range":                    "",
		"birthday_timestamp":              bdayTS,
		"lois_settings":                   map[string]any{"lois_token": ""},
		"zero_balance_state":              "",
		"network_bssid":                   nil,
		"should_skip_youth_tos":           0,
		"is_youth_regulation_flow_complete": 0,
	}
	sp := eng.iosCommonServerParams(6)
	sp["reg_info"] = patchRegInfoForIOS(buildRegInfoJSON(eng.p, eng.state), eng.p)
	_, err := iosAsyncPost(ctx, eng, "com.bloks.www.bloks.caa.reg.birthday.async", cip, sp, nil)
	return err
}

func iosSetName(ctx context.Context, eng *iosEngine, name string) error {
	eng.state.FullName = name
	eng.state.ScreenVisited = append(eng.state.ScreenVisited, "CAA_REG_IG_NAME_SCREEN")
	cip := map[string]any{
		"name":               name,
		"aac":                buildAACJSON(eng.p),
		"accounts_list":      []any{},
		"lois_settings":      map[string]any{"lois_token": ""},
		"network_bssid":      nil,
		"zero_balance_state": "",
	}
	sp := eng.iosCommonServerParams(7)
	sp["reg_info"] = patchRegInfoForIOS(buildRegInfoJSON(eng.p, eng.state), eng.p)
	_, err := iosAsyncPost(ctx, eng, "com.bloks.www.bloks.caa.reg.name_ig_and_soap.async", cip, sp, nil)
	return err
}

// iosPrecheckCloudID gọi bước precheck giữa name và username — như real iOS.
// POST /api/v1/accounts/precheck_cloud_id/ với body rỗng.
func iosPrecheckCloudID(ctx context.Context, eng *iosEngine) {
	hdr := iosHeaders(eng.p, "/accounts/precheck_cloud_id/", eng.cloudTrustToken, eng.usdID, eng.locale)
	_, _, _ = eng.sess.post(ctx, igAndroidBase+"/api/v1/accounts/precheck_cloud_id/", "", hdr)
}

func iosSetUsername(ctx context.Context, eng *iosEngine, username string) error {
	eng.state.UsernamePrefill = username
	eng.state.ScreenVisited = append(eng.state.ScreenVisited, "CAA_REG_IG_USERNAME")
	cip := map[string]any{
		"validation_text":    username,
		"aac":                buildAACJSON(eng.p),
		"family_device_id":   eng.p.FamilyDeviceID,
		"device_id":          eng.p.DeviceID,
		"lois_settings":      map[string]any{"lois_token": ""},
		"zero_balance_state": "",
		"network_bssid":      nil,
		"qe_device_id":       eng.p.DeviceID,
	}
	sp := eng.iosCommonServerParams(8)
	sp["reg_info"] = patchRegInfoForIOS(buildRegInfoJSON(eng.p, eng.state), eng.p)
	sp["text_input_id"] = randQplID()
	sp["suggestions_container_id"] = randQplID()
	sp["action"] = 1
	sp["screen_id"] = randQplID()
	sp["post_tos"] = 0
	sp["input_id"] = randQplID()
	_, err := iosAsyncPost(ctx, eng, "com.bloks.www.bloks.caa.reg.username.async", cip, sp, nil)
	if err == nil {
		eng.state.Username = username
	}
	return err
}

func iosCreateAccount(ctx context.Context, eng *iosEngine) (igcore.IGSession, error) {
	eng.state.ScreenVisited = append(eng.state.ScreenVisited, "bk_caa_reg_icon_text_list_tos_screen")

	const maxAttempts = 3 // retry createAccount (giảm từ 12 — tránh chờ lâu khi IP fail)
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		cip := map[string]any{
			"aac":                                    buildAACJSON(eng.p),
			"reached_from_tos_screen":                1,
			"machine_id":                             eng.p.MachineID,
			"device_id":                              eng.p.DeviceID,
			"waterfall_id":                           eng.p.WaterfallID,
			"ck_error":                               "CKErrorDomain: 9",
			"ck_id":                                  nil,
			"ck_nonce":                               nil,
			"lois_settings":                          map[string]any{"lois_token": ""},
			"network_bssid":                          nil,
			"zero_balance_state":                     "",
			"encrypted_msisdn":                       "",
			"failed_birthday_year_count":             "",
			"headers_last_infra_flow_id":             "",
			"ig_partially_created_account_nonce":     "",
			"ig_partially_created_account_nonce_expiry": 0,
			"ig_partially_created_account_user_id":   0,
			"no_contact_perm_email_oauth_token":      "",
		}
		sp := eng.iosCommonServerParams(9)
		sp["reg_info"] = patchRegInfoForIOS(buildRegInfoJSON(eng.p, eng.state), eng.p)
		sp["bloks_controller_source"] = "bk_caa_reg_icon_text_list_tos_screen"

		resp, err := iosAsyncPost(ctx, eng, "com.bloks.www.bloks.caa.reg.create.account.async", cip, sp, nil)
		if err != nil {
			return igcore.IGSession{}, fmt.Errorf("createAccount: %w", err)
		}

		igSess := igcore.ParseIGSession(resp)
		if igSess.SessionID == "" {
			igSess = sessionFromJar(eng.sess)
		} else {
			jar := sessionFromJar(eng.sess)
			if igSess.Mid == "" {
				igSess.Mid = jar.Mid
			}
			if igSess.Datr == "" {
				igSess.Datr = jar.Datr
			}
			if igSess.IgDID == "" {
				igSess.IgDID = jar.IgDID
			}
			if igSess.Rur == "" {
				igSess.Rur = jar.Rur
			}
			igSess.FullCookie = igcore.BuildFullCookieStr(igSess)
		}
		if igSess.SessionID != "" {
			return igSess, nil
		}

		isIntegrity := strings.Contains(resp, "integrity_block")
		isSystemErr := strings.Contains(resp, "system_error")
		if isIntegrity || isSystemErr {
			if attempt < maxAttempts {
				// Quyết định đổi IP hay giữ IP:
				//   integrity_block → LUÔN đổi IP (IP bị flag/rate-limit, cần IP sạch).
				//   system_error    → IG báo jurisdiction lệch GeoIP của IP hiện tại.
				//                      Học jurisdiction IG trả về rồi GIỮ IP retry → jurisdiction
				//                      khớp đúng GeoIP của chính IP đó → qua được. Đổi IP lúc này
				//                      làm jurisdiction vừa học thành stale → lệch nhịp mãi mãi.
				//                      CHỈ đổi IP khi jurisdiction đã khớp mà vẫn lỗi (học bế tắc).
				rotate := true
				reason := "integrity_block"
				if isSystemErr {
					reason = "system_error"
					if m := regJurisdictionRe.FindStringSubmatch(resp); len(m) > 1 {
						igCC := m[1]
						if eng.state.Jurisdiction != igCC {
							eng.logf("ios createAccount %d/%d: system_error jurisdiction %q→%q", attempt, maxAttempts, eng.state.Jurisdiction, igCC)
							eng.state.Jurisdiction = igCC
							rotate = false // vừa học jurisdiction mới → giữ IP cho khớp
						}
					}
				}
				if ctx.Err() != nil {
					return igcore.IGSession{}, ctx.Err()
				}
				if rotate {
					eng.logf("ios createAccount %d/%d: %s → rotate IP + retry", attempt, maxAttempts, reason)
					// SetProxy thay đổi IP nhưng giữ nguyên cookie jar (session cookies từ các bước trước).
					// CloseIdleConnections bắt buộc để tránh reuse TLS session cũ với proxy mới (bad record MAC).
					newProxy := igcore.RotateSession(eng.proxyStr)
					if proxyURL := proxy.FormatProxyURL(newProxy); proxyURL != "" {
						if setErr := eng.sess.client.SetProxy(proxyURL); setErr == nil {
							eng.sess.client.CloseIdleConnections()
							eng.proxyStr = newProxy
						}
					}
				} else {
					eng.logf("ios createAccount %d/%d: %s → giữ IP + retry (jurisdiction khớp GeoIP)", attempt, maxAttempts, reason)
				}
				continue
			}
			reason := "integrity_block"
			if isSystemErr {
				reason = "system_error"
			}
			return igcore.IGSession{}, fmt.Errorf("createAccount: %s sau %d lần", reason, maxAttempts)
		}
		return igcore.IGSession{}, fmt.Errorf("createAccount: no session (%.200s)", resp)
	}
	return igcore.IGSession{}, fmt.Errorf("createAccount: thất bại sau %d lần", maxAttempts)
}

// ── Registerer ───────────────────────────────────────────────────────────────

type igIOSBloksRegisterer struct{}

func (r *igIOSBloksRegisterer) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	status := func(msg string) {
		if onStatus != nil {
			onStatus(msg)
		}
	}

	fail := func(stage, msg string) *instagram.RegResult {
		return &instagram.RegResult{
			Success: false,
			Email:   input.Email,
			Message: fmt.Sprintf("[igiosbloks/%s] %s", stage, msg),
		}
	}

	if strings.TrimSpace(input.Email) == "" || input.GetOTP == nil {
		return &instagram.RegResult{Success: false, Message: "igiosbloks: cần Email + GetOTP"}
	}

	proxyStr := igcore.RotateSession(input.Proxy)

	sess, err := newIOSSession(proxyStr)
	if err != nil {
		return fail("session", err.Error())
	}

	// Detect country qua proxy IP để locale/timezone khớp với IP thật.
	// Dùng goroutine + select để enforce timeout — tls-client không đảm bảo cancel theo context.
	cc := ""
	{
		ccCh := make(chan string, 1)
		go func() {
			if geoSess, err := igcore.NewIGSession(proxyStr); err == nil {
				ccCh <- geoSess.CheckProxyCountry(context.Background())
			} else {
				ccCh <- ""
			}
		}()
		select {
		case cc = <-ccCh:
		case <-time.After(8 * time.Second):
		case <-ctx.Done():
		}
	}
	loc := localeFromCountry(cc)
	status("country:" + cc + " locale:" + loc.Locale)

	p, err := newAndroidProfile(loc.Locale)
	if err != nil {
		return fail("profile", err.Error())
	}
	// iOS: dùng UUID làm device_id (không phải "android-HEX")
	p.UserAgent = randomIOSUA(loc.Locale)
	p.AndroidID = p.DeviceID
	status("ua:" + p.UserAgent)

	// qe/sync để lấy encryption key
	status("qe/sync")
	qeSyncBody := "id=" + p.DeviceID + "&experiments=ig_android_device_detection_info_upload"
	qeHeaders := [][2]string{
		{"user-agent", p.UserAgent},
		{"accept-encoding", "gzip"},
		{"accept", "*/*"},
		{"x-ig-app-id", igIOSAppID},
		{"x-ig-capabilities", igIOSCaps},
		{"x-ig-device-id", p.DeviceID},
		{"x-ig-family-device-id", p.FamilyDeviceID},
		{"x-mid", p.MachineID},
		{"content-type", "application/x-www-form-urlencoded; charset=UTF-8"},
	}
	qeResp, qeHeader, qeErr := sess.post(ctx, igAndroidBase+"/api/v1/qe/sync/", qeSyncBody, qeHeaders)
	_ = qeResp
	var keyID, pubKey, xmid string
	if qeErr == nil {
		keyID = qeHeader.Get("ig-set-password-encryption-key-id")
		pubKey = qeHeader.Get("ig-set-password-encryption-pub-key")
		xmid = qeHeader.Get("ig-set-x-mid")
	}
	if xmid != "" {
		p.MachineID = xmid
	}
	if keyID == "" || pubKey == "" {
		keyID, pubKey, xmid, err = sess.qeSync(ctx, p)
		if err != nil {
			return fail("qeSync", err.Error())
		}
		if xmid != "" {
			p.MachineID = xmid
		}
	}

	base := &igAndroidEngine{
		sess:     sess,
		p:        p,
		state:    &regInfoState{Jurisdiction: cc},
		keyID:    keyID,
		pubKey:   pubKey,
		proxyStr: proxyStr,
		log:      func(f string, a ...any) { status(fmt.Sprintf(f, a...)) },
	}
	eng := newIOSEngine(base)
	eng.locale = loc

	addr := strings.TrimSpace(input.Email)
	password := input.Password
	if password == "" {
		password = buildPassword()
	}
	name := buildName(input)

	// exposeNTM — bước warm-up iOS bắt buộc trước /graphql_www
	status("exposeNTM")
	iosExposeNTM(ctx, eng)

	status("submitEmail")
	if err := iosSubmitEmail(ctx, eng, addr); err != nil {
		return fail("submitEmail", err.Error())
	}

	status("readOTP")
	otp, err := input.GetOTP(ctx)
	if err != nil || otp == "" {
		msg := "GetOTP failed"
		if err != nil {
			msg = err.Error()
		}
		return fail("readOTP", msg)
	}

	status("confirmOTP")
	if err := iosConfirmOTP(ctx, eng, addr, otp); err != nil {
		return fail("confirmOTP", err.Error())
	}

	status("setPassword")
	if err := iosSetPassword(ctx, eng, addr, password); err != nil {
		return fail("setPassword", err.Error())
	}

	status("setBirthday")
	if err := iosBirthday(ctx, eng); err != nil {
		return fail("setBirthday", err.Error())
	}

	status("setName")
	if err := iosSetName(ctx, eng, name); err != nil {
		return fail("setName", err.Error())
	}

	// precheck_cloud_id — bước iOS native giữa name và username
	status("precheckCloudID")
	iosPrecheckCloudID(ctx, eng)

	username := buildUsername()
	status("setUsername")
	if err := iosSetUsername(ctx, eng, username); err != nil {
		return fail("setUsername", err.Error())
	}

	status("createAccount")
	igSess, err := iosCreateAccount(ctx, eng)
	if err != nil {
		return fail("createAccount", err.Error())
	}

	liveCtx, liveCancel := context.WithTimeout(ctx, 15*time.Second)
	defer liveCancel()
	liveStatus := igcore.CheckLiveByCookie(liveCtx, igSess.FullCookie, p.UserAgent, proxyStr)
	status(fmt.Sprintf("checklive → %s", liveStatus))

	base2 := &instagram.RegResult{
		UID:            igSess.UID,
		Username:       username,
		Password:       password,
		Cookie:         igSess.FullCookie,
		Email:          addr,
		DeviceID:       p.DeviceID,
		FamilyDeviceID: p.FamilyDeviceID,
		UserAgent:      p.UserAgent,
		LiveStatus:     liveStatus,
	}
	switch liveStatus {
	case "checkpoint":
		base2.Success = false
		base2.Message = "checkpoint: tài khoản cần xác minh"
	case "suspended":
		base2.Success = false
		base2.Message = "blocked: tài khoản bị treo"
	case "die":
		base2.Success = false
		base2.Message = "die: session chết ngay sau reg"
	default:
		base2.Success = true
		base2.Message = "ok"
	}
	return base2
}

func init() {
	instagram.RegisterPlatformRegisterer("ig_ios_bloks", func() instagram.Registerer {
		return &igIOSBloksRegisterer{}
	})
}
