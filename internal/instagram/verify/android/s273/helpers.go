// helpers.go — S273 verify body builders + headers + UA builder.
// Endpoint: b-api.facebook.com/method/user.editregistrationcontactpoint (add email)
//
//	b-api.facebook.com/method/user.confirmcontactpoint (confirm OTP)
//
// Khác s399: host b-api (không phải graph), path /method/user.xxx, body có field method=.
//
// UA build cho s273 (refactor 2026-05-28): UA GỐC CỐ ĐỊNH theo pattern s563v4/s564v1,
// pool 2 device Samsung Galaxy S21+ và S23, FBAV cố định 273.0.0.39.123.
// Chỉ FBCR thay theo country của phone (qua PickCountryCarrierLocale).
//
// Trước đây random từ devices.txt → model rác (YP-GS1, KIDS02...), FBAV stuck 554,
// carrier stuck T-Mobile → bot detection risk cao.
package s273

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"

	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/instagram/verify/verifybase"
)

// ─── Constants ───────────────────────────────────────────────────────────────

const (
	// Endpoint URLs — b-api.facebook.com (không phải graph)
	addEmailURL = "https://b-api.facebook.com/method/user.editregistrationcontactpoint"
	confirmURL  = "https://b-api.facebook.com/method/user.confirmcontactpoint"
	resendURL   = "https://b-api.facebook.com/method/user.sendconfirmationcode"

	addEmailFriendly = "editRegistrationContactpoint"
	confirmFriendly  = "confirmContactpoint"
	resendFriendly   = "sendConfirmationCode"

	// Full package path (khác s399 dùng short name)
	addEmailCaller = "com.facebook.confirmation.fragment.ConfContactpointFragment"
	confirmCaller  = "com.facebook.confirmation.fragment.ConfCodeInputFragment"
	resendCaller   = "com.facebook.confirmation.fragment.ConfCodeInputFragment"
)

// ─── UA builder (random device + buildID thật từ Config/DeviceInfo) ──────────

// RandomUA — wired vào factory.RegisterPlatformVerifyUA(PlatformS273, ...) trong verify.go.
//
// Build UA theo format CHUẨN capture API 273 (Dalvik prefix + FB_FW/1 + FBDM/ + FBCA :null):
//
//   Dalvik/2.1.0 (Linux; U; Android <OS>; <Model> Build/<BuildID>) [FBAN/FB4A;FBAV/<>;
//   FBPN/com.facebook.katana;FBLC/<locale>;FBBV/<>;FBCR/<>;FBMF/<>;FBBD/<>;FBDV/<>;
//   FBSV/<>;FBCA/<arch>:null;FBDM/{density=<>,width=<>,height=<>};FB_FW/1;FBRV/0;]
//
// Nguồn dữ liệu (random từ Config/DeviceInfo):
//   - FBAV/FBBV  → versions_and_builds.txt (qua RandomFbVersion)
//     User chọn random FBAV (2026-05-28) thay vì cố định 273 — capture cho thấy endpoint
//     /method/user.xxx chấp nhận nhiều version (547, 551, 555, 273...). Risk: FBAV cao
//     (558+) có thể bị FB reject. Đảm bảo pool versions_and_builds.txt KHÔNG chứa version
//     cao bất hợp lý nếu thấy verify fail nhiều.
//   - Device (FBMF/FBBD/FBDV/FBSV/BuildID/FBDM/FBCA) → devices.txt + buildnums.txt
//     + densities + screen_resolution + os_versions + device_cores
//     (qua RandomDeviceProfile — 1 lần gọi đủ cả combo)
//   - FBLC + FBCR → PickCountryCarrierLocale theo country của phone
//
// FBCA format: "<arch>:null" (theo capture chuẩn).
func RandomUA(countryCode string) string {
	fbVer, fbBuild := fakeinfo.RandomFbVersionVer()
	device := fakeinfo.RandomDeviceProfile()

	// Locale + carrier:
	//   - countryCode != ""  → theo country của phone (PickCountryCarrierLocale)
	//   - countryCode == ""  → giữ default cố định Viettel + vi_VN (user không tick
	//     "Thay nhà mạng" → không muốn random global, giữ FBCR default)
	locale, carrier := "", ""
	if countryCode != "" {
		locale, carrier = verifybase.PickCountryCarrierLocale(countryCode)
	}
	if locale == "" {
		locale = fakeinfo.LocaleFromCountry(countryCode)
	}
	if locale == "" {
		locale = "en_US"
	}
	if carrier == "" {
		if countryCode == "" {
			// User không tick "Thay nhà mạng" — giữ Viettel default thay vì random global.
			carrier = "Viettel"
		} else {
			// Country pool rỗng cho country này → fallback random global.
			carrier = fakeinfo.RandomCarrier()
			if carrier == "" {
				carrier = "Viettel"
			}
		}
	}

	buildID := device.BuildID
	if buildID == "" {
		buildID = device.Brand + "-" + device.Model
	}

	// FBCA: arch thuần từ device_cores.txt (user test 2026-05-28 — bỏ ":null").
	// Có thể là single (arm64-v8a) hoặc dual (x86:armeabi-v7a) tuỳ pool.
	fbca := device.Architecture

	return fmt.Sprintf(
		"Dalvik/2.1.0 (Linux; U; Android %s; %s Build/%s) "+
			"[FBAN/FB4A;FBAV/%s;FBPN/com.facebook.katana;FBLC/%s;FBBV/%s;FBCR/%s;"+
			"FBMF/%s;FBBD/%s;FBDV/%s;FBSV/%s;FBCA/%s;"+
			"FBDM/{density=%s,width=%d,height=%d};FB_FW/1;FBRV/0;]",
		device.OSVersion, device.Model, buildID,
		fbVer, locale, fbBuild, carrier,
		device.Manufacturer, device.Brand, device.Model, device.OSVersion,
		fbca,
		device.Density, device.ScreenWidth, device.ScreenHeight,
	)
}

// IsPoolUA — UA có phải FB4A Android native không. s273 dùng Dalvik prefix +
// FB4A blob → check cả 2 signature.
func IsPoolUA(ua string) bool {
	return strings.Contains(ua, "FBAN/FB4A") && strings.Contains(ua, "Dalvik/")
}

// truncateForLog cắt chuỗi cho log, thêm "..." nếu bị cắt. Dùng để log response body
// trong confirm step mà không spam log file.
//
// Tự decode \uXXXX escape (Thai/Arabic/CJK trong FB error_msg) → ký tự thật dễ đọc.
// Cắt theo RUNE để không phá multi-byte UTF-8 (Thai, CJK chiếm 3 bytes).
func truncateForLog(s string, n int) string {
	if n <= 0 {
		return ""
	}
	decoded := verifybase.DecodeUnicodeEscapes(s)
	runes := []rune(decoded)
	if len(runes) <= n {
		return decoded
	}
	return string(runes[:n]) + "..."
}

// postConfirmCheckEmail — GET graph.facebook.com/me?fields=id,email với access token
// để VERIFY thật sự email đã attach vào account chưa.
//
// Trả 1 trong 4 status:
//   - "OK":       FB trả về field "email" non-empty → confirm thật sự thành công
//   - "NO_EMAIL": HTTP 200 + có "id" nhưng không có "email" → confirm silent fail
//   - "DIE":      FB error code 459/190/OAuthException → token checkpoint/dead
//   - "UNKNOWN":  lỗi mạng hoặc parse fail → không quyết định được
//
// Dùng cho s273 verify Step 3.5 (catch silent failures và checkpoint sau confirm).
type postConfirmAPIResp struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Error *struct {
		Code    int    `json:"code"`
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func postConfirmCheckEmail(ctx context.Context, client tls_client.HttpClient, token string) (status, detail string) {
	if token == "" {
		return "UNKNOWN", "empty token"
	}
	endpoint := "https://graph.facebook.com/me?fields=id,email&access_token=" + url.QueryEscape(token)
	req, err := fhttp.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "UNKNOWN", "build req: " + err.Error()
	}
	req.Header.Set("user-agent", "Mozilla/5.0 (Linux; Android 9) FB4A")
	req.Header.Set("accept-encoding", "gzip, deflate")

	resp, err := client.Do(req)
	if err != nil {
		return "UNKNOWN", "HTTP: " + err.Error()
	}
	defer resp.Body.Close()

	// Reuse verifybase pattern: ReadBody is in another package, đọc trực tiếp ở đây cho gọn.
	buf := make([]byte, 32*1024)
	n, _ := resp.Body.Read(buf)
	body := buf[:n]

	var ar postConfirmAPIResp
	if jerr := json.Unmarshal(body, &ar); jerr != nil {
		return "UNKNOWN", "parse JSON: " + truncateForLog(string(body), 120)
	}

	if ar.Error != nil {
		msgLow := strings.ToLower(ar.Error.Message)
		if ar.Error.Code == 459 || strings.Contains(msgLow, "checkpoint") {
			return "DIE", fmt.Sprintf("FB error %d: %s", ar.Error.Code, truncateForLog(ar.Error.Message, 100))
		}
		if ar.Error.Code == 190 || strings.Contains(msgLow, "expired") || strings.Contains(msgLow, "invalid") {
			return "DIE", fmt.Sprintf("FB error %d (token dead): %s", ar.Error.Code, truncateForLog(ar.Error.Message, 100))
		}
		return "UNKNOWN", fmt.Sprintf("FB error %d: %s", ar.Error.Code, truncateForLog(ar.Error.Message, 100))
	}

	if strings.TrimSpace(ar.Email) != "" {
		return "OK", "email=" + ar.Email
	}
	if ar.ID != "" {
		return "NO_EMAIL", "id=" + ar.ID + " nhưng email rỗng"
	}
	return "UNKNOWN", "response không có id/email/error: " + truncateForLog(string(body), 120)
}

// ─── Body builders ───────────────────────────────────────────────────────────

// buildAddEmailBody — POST /method/user.editregistrationcontactpoint
func buildAddEmailBody(emailAddr, locale, countryCode string) string {
	params := url.Values{}
	params.Set("add_contactpoint", emailAddr)
	params.Set("add_contactpoint_type", "EMAIL")
	params.Set("format", "json")
	params.Set("locale", locale)
	params.Set("client_country_code", countryCode)
	params.Set("method", "user.editregistrationcontactpoint")
	params.Set("fb_api_req_friendly_name", addEmailFriendly)
	params.Set("fb_api_caller_class", addEmailCaller)
	return params.Encode()
}

// buildConfirmCodeBody — POST /method/user.confirmcontactpoint
func buildConfirmCodeBody(emailAddr, code, locale, countryCode string) string {
	params := url.Values{}
	params.Set("normalized_contactpoint", emailAddr)
	params.Set("contactpoint_type", "EMAIL")
	params.Set("code", code)
	params.Set("source", "ANDROID_DIALOG_API")
	params.Set("surface", "hard_cliff")
	params.Set("format", "json")
	params.Set("locale", locale)
	params.Set("client_country_code", countryCode)
	params.Set("method", "user.confirmcontactpoint")
	params.Set("fb_api_req_friendly_name", confirmFriendly)
	params.Set("fb_api_caller_class", confirmCaller)
	return params.Encode()
}

// buildResendBody — POST /method/user.sendconfirmationcode
func buildResendBody(emailAddr, locale, countryCode string) string {
	params := url.Values{}
	params.Set("normalized_contactpoint", emailAddr)
	params.Set("contactpoint_type", "EMAIL")
	params.Set("format", "json")
	params.Set("locale", locale)
	params.Set("client_country_code", countryCode)
	params.Set("method", "user.sendconfirmationcode")
	params.Set("fb_api_req_friendly_name", resendFriendly)
	params.Set("fb_api_caller_class", resendCaller)
	return params.Encode()
}

// ─── Headers ─────────────────────────────────────────────────────────────────

// buildVerifyHeaders — headers cho cả 2 endpoint. friendlyName đổi theo step.
func buildVerifyHeaders(ua, token, simHNI, friendlyName string) [][2]string {
	if simHNI == "" {
		simHNI = "45201" // MobiFone VN default
	}
	return [][2]string{
		{"host", "b-api.facebook.com"},
		{"x-fb-connection-quality", "EXCELLENT"},
		{"x-fb-sim-hni", simHNI},
		{"x-fb-connection-type", "unknown"},
		{"user-agent", ua},
		{"x-fb-connection-bandwidth", "7903429"},
		{"authorization", "OAuth " + token},
		{"x-fb-friendly-name", friendlyName},
		{"x-fb-net-hni", simHNI},
		{"x-zero-state", "unknown"},
		{"content-encoding", "gzip"},
		{"content-type", "application/x-www-form-urlencoded"},
		{"x-tigon-is-retry", "False"},
		{"accept-encoding", "gzip, deflate"},
		{"x-fb-http-engine", "Liger"},
	}
}
