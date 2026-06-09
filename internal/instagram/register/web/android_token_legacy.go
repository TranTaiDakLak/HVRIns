// android_token_legacy.go — REST classic `/auth/login` endpoint để lấy EAA token.
//
// Port pattern từ S399 (s399/body.go + register.go step 2):
//   POST https://b-graph.facebook.com/auth/login
//     Body: form-urlencoded với plaintext password #PWD_FB4A:0:ts:pwd
//     + api_key, sig (MD5), access_token (app token)
//   Response JSON: { access_token: "EAAAAU...", session_cookies: [...] }
//
// Khác hoàn toàn Bloks/RSA flow (graph.facebook.com/graphql):
//   - REST classic Facebook OAuth API (stable, không bị FB rotate schema)
//   - Không cần bloks_versioning_id (luôn outdated)
//   - Không cần RSA encrypt password
//   - Match S399 step 2 đang chạy ổn định
//
// Dùng cho WebAndroid reg + Web reg (cookie-only platforms) → lấy EAA sau reg.
package web

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	legacyAuthEndpoint   = "https://b-graph.facebook.com/auth/login"
	legacyAPIKey         = "882a8490361da98702bf97a021ddc14d"
	legacyOAuthToken     = "350685531728|62f8ce9f74b12f84c123cc23437a4a32"
	legacyAppSecret      = "62f8ce9f74b12f84c123cc23437a4a32"
	legacyFriendlyAuth   = "authenticate"
	legacyCallerAuth     = "Fb4aAuthHandler"
)

// SkipAuthLoginAtReg — khi true, reg cookie-only (WebAndroid / Web) KHÔNG gọi
// /auth/login lấy EAAAAU sau reg. Set bởi app.go khi verify platform là iOS:
// iOS verify tự login lấy EAAAAAY nên token EAAAAU lúc reg là vô dụng + sai loại.
// "Login nằm ở lúc verify, không phải reg." Mặc định false → giữ nguyên hành vi
// cho verify Android-family (vẫn lấy token sẵn lúc reg để verify khỏi login lại).
var SkipAuthLoginAtReg bool

// legacyLoginResponse — response từ /auth/login.
// FB trả uid là NUMBER (không có quote) → dùng json.Number để accept cả 2.
type legacyLoginResponse struct {
	AccessToken    string                   `json:"access_token"`
	UID            json.Number              `json:"uid"`
	SessionKey     string                   `json:"session_key"`
	SessionSecret  string                   `json:"secret"`
	SessionCookies []map[string]interface{} `json:"session_cookies"`
	Error          *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// FetchAndroidTokenLegacy POST `/auth/login` để lấy EAA access token.
// Port S399 step 2. REST classic — stable, không phụ thuộc Bloks schema.
//
// Args:
//   - uid: Facebook UID (làm email field)
//   - password: plaintext password
//   - machineID: datr value (lấy từ cookie reg)
//   - locale: ngôn ngữ ("en_US" default)
//   - countryCode: country code ("US" default)
//   - proxyStr: proxy route
//   - userAgent: FB4A UA (caller bảo đảm đúng format)
//   - notify: callback log từng bước
func fetchAndroidTokenLegacyImpl(ctx context.Context, uid, password, machineID, locale, countryCode, proxyStr, userAgent string, notify func(string)) (string, string) {
	emit := func(msg string) {
		if notify != nil {
			notify(msg)
		}
	}
	if userAgent == "" {
		userAgent = androidUA
	}
	// Strip WebView prefix nếu lỡ truyền nhầm
	if strings.HasPrefix(userAgent, "Mozilla/") {
		if idx := strings.LastIndex(userAgent, "[FBAN/FB4A"); idx >= 0 {
			userAgent = userAgent[idx:]
		} else {
			userAgent = androidUA
		}
	}
	if locale == "" {
		locale = "en_US"
	}
	if countryCode == "" {
		countryCode = "US"
	}

	deviceID := uuid.New().String()
	familyDeviceID := deviceID
	advertisingID := uuid.New().String()
	ts := time.Now().Unix()
	encPwd := fmt.Sprintf("#PWD_FB4A:0:%d:%s", ts, password)
	jazoest := fmt.Sprintf("%d", 21000+int(ts%1000))

	emit("[Login][Android] (1/2) Build form body...")

	// Build params (giống S399 LoginMobileFormData)
	params := map[string]string{
		"adid":                        advertisingID,
		"format":                      "json",
		"device_id":                   deviceID,
		"email":                       uid, // UID làm email khi login sau reg
		"password":                    encPwd,
		"generate_analytics_claim":    "1",
		"community_id":                "",
		"linked_guest_account_userid": "",
		"cpl":                         "true",
		"family_device_id":            familyDeviceID,
		"secure_family_device_id":     "",
		"credentials_type":            "password",
		"enroll_misauth":              "false",
		"generate_session_cookies":    "1",
		"error_detail_type":           "button_with_disabled",
		"source":                      "register_api",
		"machine_id":                  machineID,
		"jazoest":                     jazoest,
		"meta_inf_fbmeta":             "NO_FILE",
		"advertiser_id":               advertisingID,
		"encrypted_msisdn":            "",
		"currently_logged_in_userid":  "0",
		"locale":                      locale,
		"client_country_code":         countryCode,
		"fb_api_req_friendly_name":    legacyFriendlyAuth,
		"fb_api_caller_class":         legacyCallerAuth,
		"api_key":                     legacyAPIKey,
	}
	params["sig"] = legacyComputeSig(params)
	params["access_token"] = legacyOAuthToken

	body := legacyFormEncode(params)

	emit("[Login][Android] (2/2) POST b-graph.facebook.com/auth/login...")

	headers := map[string]string{
		"User-Agent":         userAgent,
		"Accept-Encoding":    "gzip",
		"Content-Type":       "application/x-www-form-urlencoded",
		"X-Fb-Friendly-Name": legacyFriendlyAuth,
		"X-Fb-Connection-Type": "WIFI",
		"X-Fb-Http-Engine":   "Tigon/Liger",
		"X-Fb-Client-Ip":     "True",
	}
	resp, err := doAndroidHTTP(ctx, legacyAuthEndpoint, body, headers, proxyStr)
	if err != nil {
		emit(fmt.Sprintf("[Login][Android] ❌ POST fail: %v", err))
		return "", ""
	}

	// Parse JSON
	var parsed legacyLoginResponse
	if jerr := json.Unmarshal([]byte(resp), &parsed); jerr != nil {
		// fallback: regex
		preview := resp
		if len(preview) > 400 {
			preview = preview[:400]
		}
		emit(fmt.Sprintf("[Login][Android] ⚠ Parse JSON fail (%v), thử regex... response: %s", jerr, preview))
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		emit(fmt.Sprintf("[Login][Android] ❌ FB error: %s (code=%d)", parsed.Error.Message, parsed.Error.Code))
		return "", ""
	}
	if parsed.AccessToken != "" {
		emit(fmt.Sprintf("[Login][Android] ✅ Got token (len=%d)", len(parsed.AccessToken)))
		return parsed.AccessToken, composeLegacyCookies(parsed.SessionCookies)
	}

	// Fallback: regex extract EAA from response body
	tok := extractAccessToken(resp)
	if tok != "" {
		emit(fmt.Sprintf("[Login][Android] ✅ Got EAA via regex (len=%d)", len(tok)))
		return tok, ""
	}

	preview := resp
	if len(preview) > 500 {
		preview = preview[:500]
	}
	emit(fmt.Sprintf("[Login][Android] ❌ Không tìm thấy token — response: %s", preview))
	return "", ""
}

// FetchAndroidTokenLegacy — wrapper giữ signature cũ (chỉ trả token), cho các caller
// lúc REG không cần cookie. Verify dùng FetchAndroidTokenLegacyWithCookie để lấy cookie mới.
func FetchAndroidTokenLegacy(ctx context.Context, uid, password, machineID, locale, countryCode, proxyStr, userAgent string, notify func(string)) string {
	tok, _ := fetchAndroidTokenLegacyImpl(ctx, uid, password, machineID, locale, countryCode, proxyStr, userAgent, notify)
	return tok
}

// FetchAndroidTokenLegacyWithCookie trả (token, cookie). Cookie = "name=value;..." ghép
// từ session_cookies (c_user/xs/fr/datr) của response /auth/login — dùng ở VERIFY để
// cập nhật cookie MỚI lên UI sau login (nguyên tắc: login xong phải update cookie+token).
func FetchAndroidTokenLegacyWithCookie(ctx context.Context, uid, password, machineID, locale, countryCode, proxyStr, userAgent string, notify func(string)) (string, string) {
	return fetchAndroidTokenLegacyImpl(ctx, uid, password, machineID, locale, countryCode, proxyStr, userAgent, notify)
}

// composeLegacyCookies ghép session_cookies array → cookie string "name=value;...".
func composeLegacyCookies(cks []map[string]interface{}) string {
	var b strings.Builder
	for _, c := range cks {
		name, _ := c["name"].(string)
		val, _ := c["value"].(string)
		if name == "" || val == "" {
			continue
		}
		b.WriteString(name)
		b.WriteString("=")
		b.WriteString(val)
		b.WriteString(";")
	}
	return b.String()
}

// legacyComputeSig — port S399 computeSig (MD5 of sorted "key=value" + app_secret).
func legacyComputeSig(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "sig" || k == "access_token" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(params[k])
	}
	sb.WriteString(legacyAppSecret)
	sum := md5.Sum([]byte(sb.String()))
	return hex.EncodeToString(sum[:])
}

// legacyFormEncode — port S399 formEncode (URL-encoded form body với sorted keys).
func legacyFormEncode(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	first := true
	for _, k := range keys {
		if !first {
			sb.WriteString("&")
		}
		first = false
		sb.WriteString(url.QueryEscape(k))
		sb.WriteString("=")
		sb.WriteString(url.QueryEscape(params[k]))
	}
	return sb.String()
}
