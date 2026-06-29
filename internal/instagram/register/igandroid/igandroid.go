// Package igandroid — Instagram Android native Bloks reg flow.
//
// Sử dụng POST /api/v1/bloks/async_action/{endpoint}/ với profile Okhttp4Android13.
// Khác iOS (graphql_www): mỗi bước là 1 bloks endpoint riêng, reg_info tích lũy
// client-side qua tất cả các step. reg_context được carry-over từ response trước.
package igandroid

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"sync"

	"HVRIns/internal/igcore"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/proxy"
)

// ── Constants ─────────────────────────────────────────────────────────────────

const (
	igAndroidAppID    = "567067343352427"
	igAndroidBloksVer = "16e9197b928710eafdf1e803935ed8c450a1a2e3eb696bff1184df088b900bcf" // khớp capture IGRegisterVIP 361.0.0.0.84
	igAndroidBase     = "https://i.instagram.com"

	safetynetErrResp = "API_ERROR: class com.google.android.gms.common.api.ApiException:17: The SafetyNet Attestation API is deprecated and no longer functional. Please migrate to the Play Integrity API."
)

// igAppVersion cặp appVersion + buildNumber để random vào UA.
// BloksVer hash dùng chung — chỉ UA thay đổi, không ảnh hưởng API call.
type igAppVersion struct{ AppVer, BuildNum string }

var igAndroidVersions = []igAppVersion{
	// ── 2026 (mới nhất) ───────────────────────────────────────────────────────
	{"434.0.0.0.58", "384358000"},  // Jun 10 2026
	{"433.0.0.47.68", "384207160"}, // Jun 9 2026
	{"431.0.0.47.82", "383907160"}, // May 30 2026
	{"430.0.0.53.80", "383757160"}, // May 29 2026
	// ── 2025 ──────────────────────────────────────────────────────────────────
	{"429.1.0.44.70", "383507666"}, // real
	{"428.0.0.34.67", "383405708"}, // real
	{"426.0.0.32.68", "383206897"}, // real
	{"416.0.0.47.66", "382206160"}, // real
	{"410.1.0.63.71", "381607160"}, // real
	// ── 2024 ──────────────────────────────────────────────────────────────────
	{"399.0.0.51.85", "795513222"}, // confirmed working
	{"394.0.0.46.89", "785703393"},
	{"387.0.0.43.105", "773337115"},
	{"380.0.0.38.91", "758637234"},
}

// androidDevice định nghĩa device spec dùng để build UA string.
type androidDevice struct {
	APILevel     int
	AndroidMajor int
	DPI          int
	Width        int
	Height       int
	Manufacturer string
	Model        string
	Codename     string
	Chipset      string
}

// igAndroidDevices là danh sách device thực tế phổ biến để random UA.
// Đủ lớn để giảm collision khi chạy nhiều luồng song song.
var igAndroidDevices = []androidDevice{
	// ── Android 13 (API 33) ───────────────────────────────────────────────────
	{33, 13, 421, 1080, 2400, "samsung", "SM-G991B", "o1q", "exynos2100"},           // Galaxy S21
	{33, 13, 405, 1080, 2400, "samsung", "SM-A536B", "a53x", "exynos1280"},          // Galaxy A53
	{33, 13, 480, 1080, 2400, "xiaomi", "2312DRAABL", "fuxi", "kalama"},             // Xiaomi 13
	{33, 13, 395, 1080, 2400, "xiaomi", "22101316UCP", "sunstone", "snapdragon695"}, // Redmi Note 12 Pro
	{33, 13, 450, 1080, 2412, "oneplus", "CPH2449", "salami", "taro"},               // OnePlus 11
	{33, 13, 402, 1080, 2400, "motorola", "motorola edge 40", "hayes", "mt6893"},    // Moto Edge 40
	// ── Android 14 (API 34) ───────────────────────────────────────────────────
	{34, 14, 393, 1080, 2340, "samsung", "SM-S911B", "dm1q", "snapdragon8gen2"},     // Galaxy S23
	{34, 14, 390, 1080, 2340, "samsung", "SM-S916B", "dm2q", "snapdragon8gen2"},     // Galaxy S23+
	{34, 14, 450, 1080, 2340, "samsung", "SM-S901B", "r0q", "exynos2200"},           // Galaxy S22
	{34, 14, 500, 1080, 2340, "samsung", "SM-S908B", "b0q", "exynos2200"},           // Galaxy S22 Ultra
	{34, 14, 400, 1080, 2340, "samsung", "SM-A546B", "a54x", "exynos1380"},          // Galaxy A54
	{34, 14, 420, 1080, 2400, "google", "Pixel 7", "panther", "gs201"},              // Pixel 7
	{34, 14, 428, 1080, 2400, "google", "Pixel 8", "shiba", "tensor_g3"},            // Pixel 8
	{34, 14, 429, 1080, 2400, "google", "Pixel 7a", "lynx", "gs201"},                // Pixel 7a
	{34, 14, 413, 1080, 2400, "xiaomi", "23049PCD8G", "thor", "snapdragon8gen2"},    // Xiaomi 13T
	{34, 14, 403, 1080, 2400, "motorola", "moto g84 5G", "bangkk", "snapdragon695"}, // Moto G84
	// ── Android 15 (API 35) ───────────────────────────────────────────────────
	{35, 15, 450, 1080, 2400, "samsung", "SM-G996B", "t2s", "exynos2100"}, // Galaxy S21+
	{35, 15, 416, 1080, 2340, "samsung", "SM-S921B", "e1q", "exynos2400"}, // Galaxy S24
	{35, 15, 422, 1080, 2424, "google", "Pixel 9", "tokay", "tensor_g4"},  // Pixel 9
	{35, 15, 429, 1080, 2088, "google", "Pixel 8a", "akita", "tensor_g3"}, // Pixel 8a
}

// randomAndroidUA chọn ngẫu nhiên 1 device và tạo UA string.
func randomAndroidUA(locale string) string {
	nd := new(big.Int).SetInt64(int64(len(igAndroidDevices)))
	di, _ := rand.Int(rand.Reader, nd)
	d := igAndroidDevices[di.Int64()]

	nv := new(big.Int).SetInt64(int64(len(igAndroidVersions)))
	vi, _ := rand.Int(rand.Reader, nv)
	v := igAndroidVersions[vi.Int64()]

	return fmt.Sprintf(
		"Instagram %s Android (%d/%d; %ddpi; %dx%d; %s; %s; %s; %s; %s; %s)",
		v.AppVer,
		d.APILevel, d.AndroidMajor,
		d.DPI, d.Width, d.Height,
		d.Manufacturer, d.Model, d.Codename, d.Chipset,
		locale, v.BuildNum,
	)
}

// ── Device profile ─────────────────────────────────────────────────────────────

type androidProfile struct {
	AndroidID      string // "android-{16hex}"
	DeviceID       string // lowercase UUID — x-ig-device-id / app_scoped_device_id
	FamilyDeviceID string // lowercase UUID — x-ig-family-device-id
	MachineID      string // X-MID (set after qe/sync)
	RegMachineID   string // 24-char alphanumeric — reg_info.machine_id
	WaterfallID    string // lowercase UUID
	PigeonSID      string // "UFS-{UUID}-0"
	ConnUUID       string // 32-hex
	RegFlowID      string // lowercase UUID
	UserAgent      string
	Locale         string
	AACInitTS      int64
	AACJID         string
	AACCS          string
	aacJSON        string // cache buildAACJSON (input cố định/account) — tránh marshal lại ~8 lần
}

func newAndroidProfile(locale string) (*androidProfile, error) {
	if locale == "" {
		locale = "en_GB"
	}
	// KHÔNG sinh ECDSA keystore key nữa: attestation gửi key_hash RỖNG (không ký),
	// key sinh ra trước đây không bao giờ được dùng → bỏ keygen P-256 mỗi account
	// (tốn CPU lớn khi chạy nhiều luồng).
	return &androidProfile{
		AndroidID:      "android-" + randHex(8),
		DeviceID:       strings.ToLower(uuid.New().String()),
		FamilyDeviceID: strings.ToLower(uuid.New().String()),
		RegMachineID:   randAlphanumeric(24),
		WaterfallID:    strings.ToLower(uuid.New().String()),
		PigeonSID:      "UFS-" + strings.ToLower(uuid.New().String()) + "-0",
		ConnUUID:       randHex(16),
		RegFlowID:      strings.ToLower(uuid.New().String()),
		UserAgent:      randomAndroidUA(locale),
		Locale:         locale,
		AACInitTS:      time.Now().Unix(),
		AACJID:         strings.ToLower(uuid.New().String()),
		AACCS:          randBase64URL(43),
	}, nil
}

// ── TLS session (Okhttp4Android13) ─────────────────────────────────────────────

// sharedZstdDecoder — 1 decoder DÙNG CHUNG cho toàn package. DecodeAll an toàn
// gọi đồng thời nhiều goroutine. Trước đây mỗi session tạo decoder riêng (spawn
// ~GOMAXPROCS goroutine/decoder) và KHÔNG Close → leak goroutine, CPU tăng dần
// theo số account (đặc biệt country-detect tạo session phụ mỗi acc). Dùng chung
// → tổng goroutine bị giới hạn, không leak theo số luồng.
var sharedZstdDecoder, _ = zstd.NewReader(nil)

type androidSession struct {
	client tls_client.HttpClient
	zr     *zstd.Decoder
}

func newAndroidSession(proxyStr string) (*androidSession, error) {
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(60),
		tls_client.WithClientProfile(profiles.Okhttp4Android13),
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
		return nil, fmt.Errorf("create android tls client: %w", err)
	}
	return &androidSession{client: c, zr: sharedZstdDecoder}, nil
}

func (s *androidSession) post(ctx context.Context, rawURL, body string, headers [][2]string) (string, fhttp.Header, error) {
	req, err := fhttp.NewRequestWithContext(ctx, "POST", rawURL, strings.NewReader(body))
	if err != nil {
		return "", nil, err
	}
	order := make([]string, 0, len(headers))
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
		order = append(order, kv[0])
	}
	req.Header[fhttp.HeaderOrderKey] = order
	resp, err := s.client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	dec := s.decode(resp.Header.Get("Content-Encoding"), raw)
	if resp.StatusCode >= 400 {
		return dec, resp.Header, fmt.Errorf("HTTP %d: %s", resp.StatusCode, firstN(dec, 200))
	}
	return dec, resp.Header, nil
}

func (s *androidSession) decode(_ string, raw []byte) string {
	if out, err := s.zr.DecodeAll(raw, nil); err == nil && len(out) > 0 {
		return string(out)
	}
	return string(raw)
}

var ccCountryRe = regexp.MustCompile(`"countryCode"\s*:\s*"([A-Z]{2})"`)

// checkProxyCountry detect country code của proxy bằng CHÍNH client reg (tái dùng
// kết nối + proxy) — thay vì tạo 1 igcore session MỚI mỗi account (mỗi cái = 1 TLS
// client + handshake riêng → tốn CPU lớn khi nhiều luồng). Trả "" nếu lỗi.
func (s *androidSession) checkProxyCountry(ctx context.Context) string {
	req, err := fhttp.NewRequestWithContext(ctx, "GET", "http://ip-api.com/json/?fields=countryCode", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := s.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if m := ccCountryRe.FindStringSubmatch(string(body)); len(m) > 1 {
		return m[1]
	}
	return ""
}

// ── Android device pool (datr/mid/ig_did) ─────────────────────────────────────
// Tương tự igcore.SharedDevicePool nhưng dành riêng cho ig_android session.
// Set từ app_register.go trước batch, harvest/inject tự động trong Register().

// SharedAndroidDevicePool là pool device aged cho ig_android.
// nil nghĩa là tính năng inject không hoạt động.
var SharedAndroidDevicePool *igcore.DevicePool

// harvestDevice trích xuất mid/datr/ig_did từ cookie jar của session vừa reg xong.
func harvestDevice(sess *androidSession) *igcore.AgedDevice {
	u, _ := url.Parse("https://i.instagram.com")
	cookies := sess.client.GetCookies(u)
	var mid, datr, igDID string
	for _, c := range cookies {
		switch c.Name {
		case "mid":
			mid = c.Value
		case "datr":
			datr = c.Value
		case "ig_did":
			igDID = c.Value
		}
	}
	if mid == "" && datr == "" {
		return nil
	}
	return &igcore.AgedDevice{Mid: mid, Datr: datr, IgDID: igDID}
}

// sessionFromJar builds an IGSession from cookies stored in the TLS client jar.
// Fallback for when createAccount response body doesn't embed session cookies
// (IG sends them via Set-Cookie headers instead).
func sessionFromJar(sess *androidSession) igcore.IGSession {
	u, _ := url.Parse("https://i.instagram.com")
	cookies := sess.client.GetCookies(u)
	var s igcore.IGSession
	for _, c := range cookies {
		switch c.Name {
		case "sessionid":
			s.SessionID = c.Value
		case "ds_user_id":
			s.DSUserID = c.Value
		case "csrftoken":
			s.CSRFToken = c.Value
		case "datr":
			s.Datr = c.Value
		case "ig_did":
			s.IgDID = c.Value
		case "mid":
			s.Mid = c.Value
		case "rur":
			s.Rur = c.Value
		}
	}
	if s.DSUserID != "" {
		s.UID = s.DSUserID
	}
	var parts []string
	if s.CSRFToken != "" {
		parts = append(parts, "csrftoken="+s.CSRFToken)
	}
	if s.Datr != "" {
		parts = append(parts, "datr="+s.Datr)
	}
	if s.IgDID != "" {
		parts = append(parts, "ig_did="+s.IgDID)
	}
	if s.Mid != "" {
		parts = append(parts, "mid="+s.Mid)
	}
	if s.Rur != "" {
		parts = append(parts, "rur="+s.Rur)
	}
	if s.DSUserID != "" {
		parts = append(parts, "ds_user_id="+s.DSUserID)
	}
	if s.SessionID != "" {
		parts = append(parts, "sessionid="+s.SessionID)
	}
	s.FullCookie = strings.Join(parts, ";")
	return s
}

// injectAgedDevice set cookie datr/mid/ig_did vào session trước reg và override MachineID.
func injectAgedDevice(sess *androidSession, p *androidProfile, dev *igcore.AgedDevice) {
	if dev == nil {
		return
	}
	u, _ := url.Parse("https://i.instagram.com")
	var cookies []*fhttp.Cookie
	if dev.Datr != "" {
		cookies = append(cookies, &fhttp.Cookie{Name: "datr", Value: dev.Datr, Domain: ".instagram.com", Path: "/"})
	}
	if dev.Mid != "" {
		cookies = append(cookies, &fhttp.Cookie{Name: "mid", Value: dev.Mid, Domain: ".instagram.com", Path: "/"})
		p.MachineID = dev.Mid // dùng mid aged làm X-MID header
	}
	if dev.IgDID != "" {
		cookies = append(cookies, &fhttp.Cookie{Name: "ig_did", Value: dev.IgDID, Domain: ".instagram.com", Path: "/"})
	}
	if len(cookies) > 0 {
		sess.client.SetCookies(u, cookies)
	}
}

// ── Registration state ──────────────────────────────────────────────────────────

type regInfoState struct {
	ContactPoint       string
	ConfirmationCode   string
	EncryptedPassword  string
	SafetynetToken     string
	SafetynetResponse  string
	Birthday           string // "DD-MM-YYYY"
	AgeRange           string // "o18"
	FullName           string
	Username           string // confirmed username (set after setUsername succeeds)
	UsernamePrefill    string // suggested/chosen username sent as validation_text in setUsername
	Jurisdiction       string // ISO-2 country for youth_regulation_config
	ScreenVisited      []string
	ShouldSavePassword *bool
	ShouldSkipYouthTOS *bool
}

// ── Bloks body builder ──────────────────────────────────────────────────────────

func buildBloksBody(clientInputParams, serverParams map[string]any) string {
	params := map[string]any{
		"client_input_params": clientInputParams,
		"server_params":       serverParams,
	}
	bkCtx := map[string]any{
		"bloks_version": igAndroidBloksVer,
		"styles_id":     "instagram",
	}
	paramsJSON, _ := json.Marshal(params)
	bkCtxJSON, _ := json.Marshal(bkCtx)
	return "params=" + url.QueryEscape(string(paramsJSON)) +
		"&bk_client_context=" + url.QueryEscape(string(bkCtxJSON)) +
		"&bloks_versioning_id=" + igAndroidBloksVer
}

// ── reg_info JSON builder ───────────────────────────────────────────────────────

// regMachineIDForRegInfo trả X-MID làm reg_info.machine_id. Capture IGRegisterVIP cho thấy
// reg_info.machine_id = X-MID (ig-set-x-mid, "ajwC...") — KHÔNG phải random 24-char. Trước
// đây dùng p.RegMachineID (random) → lệch với x-mid header + cip.machine_id → tín hiệu bất
// thường. Fallback RegMachineID nếu X-MID chưa có (không nên xảy ra: attestation/qe lấy X-MID
// trước mọi bước reg_info).
func regMachineIDForRegInfo(p *androidProfile) string {
	if p.MachineID != "" {
		return p.MachineID
	}
	return p.RegMachineID
}

func buildRegInfoMap(p *androidProfile, s *regInfoState) map[string]any {
	nilStr := func(v string) any {
		if v == "" {
			return nil
		}
		return v
	}
	nilBool := func(v *bool) any {
		if v == nil {
			return nil
		}
		return *v
	}
	nilStrSlice := func(v []string) any {
		if v == nil {
			return nil
		}
		return v
	}
	var accountsList any = nil
	var emailOAuth any = nil
	if s.Username != "" {
		accountsList = []any{}
		emailOAuth = []any{}
	}
	var didUseAge any = nil
	if s.AgeRange != "" {
		b := false
		didUseAge = b
	}

	jurisdiction := s.Jurisdiction
	if jurisdiction == "" {
		jurisdiction = "VN"
	}
	youthCfg := map[string]any{
		"isEnabled":                  true,
		"consentJurisdiction":        jurisdiction,
		"shouldRaiseAgeGating":       false,
		"ageOfConsent":               nil,
		"ageOfParentalConsent":       nil,
		"requiresAgeVerification":    false,
		"requiresParentalConsent":    false,
		"ageThresholdForRegBlocking": nil,
	}

	m := map[string]any{
		// Contact
		"contactpoint":              nilStr(s.ContactPoint),
		"ar_contactpoint":           nil,
		"contactpoint_type":         "email",
		"is_using_unified_cp":       false,
		"unified_cp_screen_variant": "control",
		"is_cp_auto_confirmed":      false,
		"is_cp_auto_confirmable":    false,
		"is_cp_claimed":             false,
		// Confirmation
		"confirmation_code": nilStr(s.ConfirmationCode),
		// Name
		"first_name":           nil,
		"last_name":            nil,
		"full_name":            nilStr(s.FullName),
		"suggested_first_name": nil,
		"suggested_last_name":  nil,
		"suggested_full_name":  nil,
		// Birthday
		"birthday":                   nilStr(s.Birthday),
		"birthday_derived_from_age":  nil,
		"age_range":                  nilStr(s.AgeRange),
		"did_use_age":                didUseAge,
		"os_shared_age_range":        nil,
		"spc_birthday_input":         false,
		"failed_birthday_year_count": nil,
		// Gender
		"gender":            nil,
		"use_custom_gender": false,
		"custom_gender":     nil,
		// Password / SafetyNet
		"encrypted_password":  nilStr(s.EncryptedPassword),
		"safetynet_token":     nilStr(s.SafetynetToken),
		"safetynet_response":  nilStr(s.SafetynetResponse),
		"skip_slow_rel_check": true,
		// Username
		"username":             nilStr(s.Username),
		"username_prefill":     nilStr(s.UsernamePrefill),
		"accounts_list_client": accountsList,
		// Device
		"device_id":                   p.AndroidID,
		"ig4a_qe_device_id":           p.DeviceID,
		"family_device_id":            p.FamilyDeviceID,
		"machine_id":                  regMachineIDForRegInfo(p), // X-MID (khớp capture), KHÔNG random
		"registration_flow_id":        p.RegFlowID,
		"fdid_available_on_start":     true,
		"fdid_rid_available_on_start": true,
		"asdid_available_on_start":    true,
		// Screen state
		"screen_visited":      nilStrSlice(s.ScreenVisited),
		"caa_reg_flow_source": "lid_landing_screen",
		// TOS / save password
		"should_save_password":  nilBool(s.ShouldSavePassword),
		"should_skip_youth_tos": nilBool(s.ShouldSkipYouthTOS),
		// Profile photo
		"profile_photo":           nil,
		"profile_photo_id":        nil,
		"profile_photo_upload_id": nil,
		"avatar":                  nil,
		// OAuth
		"email_oauth_token":                 nil,
		"email_oauth_tokens":                emailOAuth,
		"email_oauth_token_no_contact_perm": nil,
		"sign_in_with_google_email":         nil,
		"should_skip_two_step_conf":         nil,
		// Facebook / social
		"fb_access_token":                          nil,
		"fb_email_login_upsell_skip_suma_post_tos": false,
		"fb_suma_is_from_email_login_upsell":       false,
		"fb_suma_is_from_phone_login_upsell":       false,
		"fb_suma_is_high_confidence":               nil,
		"fb_device_id":                             nil,
		"fb_machine_id":                            nil,
		// Flags
		"is_preform":                            true,
		"is_caa_perf_enabled":                   true,
		"full_sheet_flow":                       false,
		"should_show_rel_error":                 false,
		"ignore_suma_check":                     false,
		"dismissed_login_upsell_with_cna":       false,
		"ignore_existing_login":                 false,
		"ignore_existing_login_from_suma":       false,
		"ignore_existing_login_after_errors":    false,
		"skip_step_without_errors":              false,
		"existing_account_exact_match_checked":  false,
		"existing_account_fuzzy_match_checked":  false,
		"email_oauth_exists":                    false,
		"is_too_young":                          false,
		"whatsapp_installed_on_client":          false,
		"email_prefilled":                       false,
		"cp_confirmed_by_auto_conf":             false,
		"in_sowa_experiment":                    false,
		"is_msplit_neutral_choice":              false,
		"is_youth_regulation_flow_complete":     false,
		"is_on_cold_start":                      false,
		"is_reg_request_from_ig_suma":           false,
		"is_toa_reg":                            false,
		"is_threads_public":                     false,
		"spc_import_flow":                       false,
		"is_from_registration_reminder":         false,
		"show_youth_reg_in_ig_spc":              false,
		"force_sessionless_nux_experience":      false,
		"has_seen_suma_landing_page_pre_conf":   false,
		"has_seen_suma_candidate_page_pre_conf": false,
		"has_seen_confirmation_screen":          false,
		"should_show_error_msg":                 true,
		"should_show_spi_before_conf":           true,
		"should_override_back_nav":              false,
		"eligible_to_flash_call_in_ig4a":        false,
		"eligible_to_mo_sms_in_ig4a":            false,
		"attempted_silent_auth_in_fb":           false,
		"attempted_silent_auth_in_ig":           false,
		// Misc
		"ig_footer_variant":               "control",
		"reg_suma_state":                  0,
		"suma_on_conf_threshold":          -1,
		"is_in_nta_single_form":           false,
		"is_from_web_lite_reg_controller": nil,
		// Nil fields
		"confirmation_code_send_error":          nil,
		"consent_jurisdiction_at_gate":          nil,
		"consent_jurisdiction_at_inflow":        nil,
		"user_id":                               nil,
		"bloks_controller_source":               nil,
		"caa_play_integrity_attestation_result": nil,
		"client_known_key_hash":                 nil,
		"youth_regulation_config":               youthCfg,
	}
	return m
}

// buildRegInfoJSON marshal map reg_info → JSON string (dùng cho android callers).
func buildRegInfoJSON(p *androidProfile, s *regInfoState) string {
	b, _ := json.Marshal(buildRegInfoMap(p, s))
	return string(b)
}

// buildRegInfoIOSJSON build reg_info + set thẳng 4 field iOS rồi marshal MỘT lần.
// Thay patchRegInfoForIOS(buildRegInfoJSON(...)) vốn: build map → marshal → unmarshal
// lại 100 field → sửa 4 field → marshal lại (tốn CPU, chạy 7 lần/account × 400 luồng).
func buildRegInfoIOSJSON(p *androidProfile, s *regInfoState) string {
	m := buildRegInfoMap(p, s)
	m["device_id"] = p.DeviceID
	m["ig4a_qe_device_id"] = nil
	m["caa_reg_flow_source"] = "nta_native_integration_point"
	m["skip_slow_rel_check"] = false
	b, _ := json.Marshal(m)
	return string(b)
}

// ── Common headers ──────────────────────────────────────────────────────────────

func androidHeaders(p *androidProfile, endpoint string) [][2]string {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	return [][2]string{
		{"accept-language", p.Locale + ", en-US"},
		{"content-type", "application/x-www-form-urlencoded; charset=UTF-8"},
		{"ig-intended-user-id", "0"},
		{"priority", "u=3"},
		{"x-bloks-is-layout-rtl", "false"},
		{"x-bloks-prism-button-version", "CONTROL"},
		{"x-bloks-prism-colors-enabled", "false"},
		{"x-bloks-prism-elevated-background-fix", "true"},
		{"x-bloks-prism-extended-palette-gray-red", "false"},
		{"x-bloks-prism-extended-palette-indigo", "false"},
		{"x-bloks-prism-extended-palette-rest-of-colors", "false"},
		{"x-bloks-prism-font-enabled", "false"},
		{"x-bloks-prism-indigo-link-version", "0"},
		{"x-bloks-version-id", igAndroidBloksVer},
		{"x-fb-client-ip", "True"},
		{"x-fb-connection-type", "WIFI"},
		{"x-fb-friendly-name", "IgApi: " + endpoint},
		{"x-fb-server-cluster", "True"},
		{"x-ig-android-id", p.AndroidID},
		{"x-ig-app-id", igAndroidAppID},
		{"x-ig-app-locale", p.Locale},
		{"x-ig-bandwidth-speed-kbps", "1599.000"},
		{"x-ig-bandwidth-totalbytes-b", "471936"},
		{"x-ig-bandwidth-totaltime-ms", "295"},
		{"x-ig-capabilities", "3brTv10="},
		{"x-ig-connection-type", "WIFI"},
		{"x-ig-device-id", p.DeviceID},
		{"x-ig-device-locale", p.Locale},
		{"x-ig-family-device-id", p.FamilyDeviceID},
		{"x-ig-is-foldable", "false"},
		{"x-ig-mapped-locale", p.Locale},
		{"x-ig-timezone-offset", "25200"},
		{"x-ig-www-claim", "0"},
		{"x-mid", p.MachineID},
		{"x-pigeon-rawclienttime", ts + ".000"},
		{"x-pigeon-session-id", p.PigeonSID},
		{"x-tigon-is-retry", "False"},
		{"accept-encoding", "zstd"},
		{"user-agent", p.UserAgent},
		{"x-fb-conn-uuid-client", p.ConnUUID},
		{"x-fb-http-engine", "Tigon/MNS/TCP"},
	}
}

// ── qe/sync ─────────────────────────────────────────────────────────────────────

func (s *androidSession) qeSync(ctx context.Context, p *androidProfile) (keyID, pubKey, xmid string, err error) {
	body := "id=" + p.DeviceID + "&experiments=ig_android_device_detection_info_upload"
	headers := [][2]string{
		{"user-agent", p.UserAgent},
		{"accept-encoding", "gzip"},
		{"accept", "*/*"},
		{"x-ig-app-id", igAndroidAppID},
		{"x-ig-capabilities", "3brTv10="},
		{"x-ig-android-id", p.AndroidID},
		{"x-ig-device-id", p.DeviceID},
		{"x-ig-family-device-id", p.FamilyDeviceID},
		{"x-mid", p.MachineID},
		{"content-type", "application/x-www-form-urlencoded; charset=UTF-8"},
	}
	req, err := fhttp.NewRequestWithContext(ctx, "POST",
		igAndroidBase+"/api/v1/qe/sync/", strings.NewReader(body))
	if err != nil {
		return
	}
	order := make([]string, 0, len(headers))
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
		order = append(order, kv[0])
	}
	req.Header[fhttp.HeaderOrderKey] = order
	resp, err := s.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))

	keyID = resp.Header.Get("ig-set-password-encryption-key-id")
	pubKey = resp.Header.Get("ig-set-password-encryption-pub-key")
	xmid = resp.Header.Get("ig-set-x-mid")
	if keyID == "" || pubKey == "" {
		err = fmt.Errorf("qe/sync: missing key headers (HTTP %d)", resp.StatusCode)
	}
	return
}

// ── SafetyNet token ──────────────────────────────────────────────────────────────

func buildSafetynetToken(email string) string {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	payload := email + "|" + ts + "|"
	extra := make([]byte, 16)
	_, _ = rand.Read(extra)
	raw := []byte(payload)
	raw = append(raw, extra...)
	return base64.StdEncoding.EncodeToString(raw)
}

// ── AAC helper ───────────────────────────────────────────────────────────────────

func buildAACJSON(p *androidProfile) string {
	if p.aacJSON != "" {
		return p.aacJSON // cache: input cố định/account, profile riêng mỗi goroutine (no race)
	}
	m := map[string]any{
		"aac_init_timestamp": p.AACInitTS,
		"aacjid":             p.AACJID,
		"aaccs":              p.AACCS,
	}
	b, _ := json.Marshal(m)
	p.aacJSON = string(b)
	return p.aacJSON
}

// ── Attestation ──────────────────────────────────────────────────────────────────

type attestResult struct {
	keystoreNonce      string
	keystoreSigned     string // base64url(ECDSA DER sig)
	keystoreHash       string // hex SHA-256 of DER pubkey
	playIntegrityNonce string // from server (not used in signed nonce)
}

var reNonce = regexp.MustCompile(`"challenge_nonce"\s*:\s*"([^"]+)"`)

// attestKeystore — khớp capture IGRegisterVIP. Đây là bước ĐẦU TIÊN của flow:
//
//	POST b.i.instagram.com/api/v1/attestation/create_android_keystore/
//	body: app_scoped_device_id=<uuid>&key_hash=   (key_hash RỖNG — KHÔNG ký ECDSA)
//	→ response: {"challenge_nonce","key_nonce","status":"ok"} + header ig-set-x-mid
//
// Mục đích chính: lấy X-MID (machine_id) + đăng ký device. challenge_nonce KHÔNG dùng lại
// (capture: attestation_result=null trong create.account).
func (s *androidSession) attestKeystore(ctx context.Context, p *androidProfile) (nonce, xmid string, err error) {
	body := "app_scoped_device_id=" + p.DeviceID + "&key_hash="
	headers := androidHeaders(p, "create_android_keystore")
	resp, hdr, err := s.post(ctx, "https://b.i.instagram.com/api/v1/attestation/create_android_keystore/", body, headers)
	if err != nil {
		return
	}
	if m := reNonce.FindStringSubmatch(resp); len(m) > 1 {
		nonce = m[1]
	}
	xmid = hdr.Get("ig-set-x-mid")
	return
}

func (s *androidSession) attestPlayIntegrity(ctx context.Context, p *androidProfile) (nonce string, err error) {
	body := "app_scoped_device_id=" + p.DeviceID
	headers := androidHeaders(p, "create_android_playintegrity")
	resp, _, err := s.post(ctx, igAndroidBase+"/api/v1/create_android_playintegrity/", body, headers)
	if err != nil {
		return
	}
	m := reNonce.FindStringSubmatch(resp)
	if len(m) < 2 {
		err = fmt.Errorf("play_integrity: no challenge_nonce")
		return
	}
	nonce = m[1]
	return
}

func buildAttestParams(p *androidProfile, at *attestResult) string {
	// x-ig-attest-params JSON format from captured traffic.
	// errors:[-3] = API not available (e.g. no Play Services / old device).
	// Using -3 for both attestations when nonce is empty signals "not available" rather than "fake success".
	keystoreErr := []int{0}
	if at.keystoreNonce == "" {
		keystoreErr = []int{-3}
	}
	keystoreAttests := []map[string]any{
		{
			"errors":       keystoreErr,
			"key_hash":     at.keystoreHash,
			"nonce":        at.keystoreNonce,
			"signed_nonce": at.keystoreSigned,
		},
	}
	playIntegrityAttests := []map[string]any{
		{
			"errors":          []int{-3},
			"integrity_token": "",
			"nonce":           at.playIntegrityNonce,
		},
	}
	m := map[string]any{
		"keystore_attests":       keystoreAttests,
		"play_integrity_attests": playIntegrityAttests,
	}
	b, _ := json.Marshal(m)
	return string(b)
}

// ── Bloks step implementations ───────────────────────────────────────────────────

type igAndroidEngine struct {
	sess       *androidSession
	p          *androidProfile
	state      *regInfoState
	regContext string
	keyID      string
	pubKey     string
	proxyStr   string
	log        func(string, ...any)
}

func (e *igAndroidEngine) logf(format string, args ...any) {
	if e.log != nil {
		e.log(format, args...)
	}
}

func (e *igAndroidEngine) updateRegContext(resp string) {
	if rc := igcore.ParseRegContext(resp); rc != "" {
		e.regContext = rc
	}
}

func (e *igAndroidEngine) commonServerParams(step int) map[string]any {
	flowInfoJSON := `{"flow_name":"new_to_family_ig_default","flow_type":"ntf"}`
	return map[string]any{
		"reg_context":                       e.regContext,
		"current_step":                      step,
		"event_request_id":                  strings.ToLower(uuid.New().String()),
		"INTERNAL__latency_qpl_marker_id":   36707139,
		"INTERNAL__latency_qpl_instance_id": randQplID(),
		"is_from_logged_out":                0,
		"is_from_logged_in_switcher":        0,
		"is_platform_login":                 0,
		"login_surface":                     "unknown",
		"access_flow_version":               "pre_mt_behavior",
		"offline_experiment_group":          "caa_iteration_v3_perf_ig_4",
		"layered_homepage_experiment_group": "Deploy: Not in Experiment",
		"flow_info":                         flowInfoJSON,
		"device_id":                         e.p.AndroidID,
		"family_device_id":                  e.p.FamilyDeviceID,
		"qe_device_id":                      e.p.DeviceID,
		"waterfall_id":                      e.p.WaterfallID,
	}
}

func (e *igAndroidEngine) bloksPost(ctx context.Context, endpoint string, clientInputParams, serverParams map[string]any, extraHeaders [][2]string) (string, fhttp.Header, error) {
	body := buildBloksBody(clientInputParams, serverParams)
	hdr := androidHeaders(e.p, endpoint)
	if len(extraHeaders) > 0 {
		hdr = append(hdr, extraHeaders...)
	}
	endpointURL := igAndroidBase + "/api/v1/bloks/apps/" + endpoint + "/" // capture dùng /apps/ (không phải /async_action/)
	resp, header, err := e.sess.post(ctx, endpointURL, body, hdr)

	// Debug dump — chỉ khi IGDEBUG_DIR được set
	if dir := debugDir(); dir != "" {
		stepName := endpoint[strings.LastIndex(endpoint, ".")+1:] // lấy phần cuối endpoint
		writeDebug(dir, "android_req_"+stepName+".txt", body)
		writeDebug(dir, "android_resp_"+stepName+".txt", resp)
	}
	return resp, header, err
}

// debugDir reads IGDEBUG_DIR lazily so it works even if set after package init.
func debugDir() string { return os.Getenv("IGDEBUG_DIR") }

func writeDebug(dir, name, content string) {
	_ = os.MkdirAll(dir, 0755)
	_ = os.WriteFile(dir+"/"+name, []byte(content), 0644)
}

// submitEmail — step 0.
func (e *igAndroidEngine) submitEmail(ctx context.Context, addr string) error {
	e.state.ScreenVisited = append(e.state.ScreenVisited,
		"CAA_REG_CONTACT_POINT_PHONE", "CAA_REG_CONTACT_POINT_EMAIL")
	e.state.ContactPoint = addr

	regInfo := buildRegInfoJSON(e.p, e.state)
	cip := map[string]any{
		"aac":                          buildAACJSON(e.p),
		"email":                        addr,
		"device_id":                    e.p.AndroidID,
		"family_device_id":             e.p.FamilyDeviceID,
		"confirmed_cp_and_code":        map[string]any{},
		"accounts_list":                []any{},
		"fb_ig_device_id":              []any{},
		"lois_settings":                map[string]any{"lois_token": ""},
		"cloud_trust_token":            nil,
		"network_bssid":                nil,
		"zero_balance_state":           "",
		"msg_previous_cp":              "",
		"email_token":                  "",
		"block_store_machine_id":       "",
		"switch_cp_first_time_loading": 1,
		"switch_cp_have_seen_suma":     0,
		"has_rejected_rel":             0,
		"seen_login_upsell":            0,
		"email_prefilled":              0,
		"is_from_device_emails":        0,
	}
	sp := e.commonServerParams(0)
	sp["reg_info"] = regInfo
	sp["cp_funnel"] = 0
	sp["cp_source"] = 0
	sp["text_input_id"] = randQplID()

	resp, _, err := e.bloksPost(ctx,
		"com.bloks.www.bloks.caa.reg.async.contactpoint_email.async", cip, sp, nil)
	if err != nil {
		return fmt.Errorf("submitEmail: %w", err)
	}
	e.updateRegContext(resp)
	e.logf("submitEmail OK (reg_context len=%d)", len(e.regContext))
	return nil
}

// confirmOTP — step 3.
func (e *igAndroidEngine) confirmOTP(ctx context.Context, addr, otp string) error {
	e.state.ScreenVisited = append(e.state.ScreenVisited, "CAA_REG_CONFIRMATION_SCREEN")
	regInfo := buildRegInfoJSON(e.p, e.state)
	cip := map[string]any{
		"code":                  otp,
		"confirmed_cp_and_code": map[string]any{},
		"aac":                   buildAACJSON(e.p),
	}
	sp := e.commonServerParams(3)
	sp["reg_info"] = regInfo

	resp, _, err := e.bloksPost(ctx,
		"com.bloks.www.bloks.caa.reg.confirmation.async", cip, sp, nil)
	if err != nil {
		return fmt.Errorf("confirmOTP: %w", err)
	}
	e.updateRegContext(resp)
	// Extract confirmation_code (8-char server token)
	if cc := igcore.ParseConfirmationCode(resp); cc != "" {
		e.state.ConfirmationCode = cc
		e.logf("confirmOTP OK, confirmation_code=%s", cc)
	} else {
		e.logf("confirmOTP: warning — no confirmation_code in response")
	}
	return nil
}

// setPassword — step 4.
func (e *igAndroidEngine) setPassword(ctx context.Context, addr, password string) error {
	e.state.ScreenVisited = append(e.state.ScreenVisited, "CAA_REG_PASSWORD")
	// Plaintext password (version :0:) — capture IGRegisterVIP dùng format này, KHÔNG mã hóa RSA.
	// "#PWD_INSTAGRAM:0:{unix_ts}:{password_thật}" — IG nhận trực tiếp qua HTTPS.
	encPwd := fmt.Sprintf("#PWD_INSTAGRAM:0:%d:%s", time.Now().Unix(), password)
	snt := buildSafetynetToken(addr)
	e.state.SafetynetToken = snt
	e.state.SafetynetResponse = safetynetErrResp
	e.state.EncryptedPassword = encPwd
	savePwd := true
	e.state.ShouldSavePassword = &savePwd

	regInfo := buildRegInfoJSON(e.p, e.state)
	cip := map[string]any{
		"encrypted_password":                    encPwd,
		"safetynet_token":                       snt,
		"safetynet_response":                    safetynetErrResp,
		"machine_id":                            e.p.MachineID, // X-MID (NOT RegMachineID)
		"spi_action":                            1,
		"caa_play_integrity_attestation_result": "",
		"aac":                                   buildAACJSON(e.p),
	}
	sp := e.commonServerParams(4)
	sp["reg_info"] = regInfo

	resp, _, err := e.bloksPost(ctx,
		"com.bloks.www.bloks.caa.reg.password.async", cip, sp, nil)
	if err != nil {
		return fmt.Errorf("setPassword: %w", err)
	}
	e.updateRegContext(resp)
	e.logf("setPassword OK")
	return nil
}

// setBirthday — step 6.
func (e *igAndroidEngine) setBirthday(ctx context.Context) error {
	// Random birthday 1990–2000 (adult), format DD-MM-YYYY
	year := 1990 + randIntn(11)
	month := 1 + randIntn(12)
	day := 1 + randIntn(28)
	bday := fmt.Sprintf("%02d-%02d-%04d", day, month, year)
	// birthday_timestamp: unix seconds of midnight UTC for that date
	bdayTS := birthdayUnix(year, month, day)
	e.state.Birthday = bday
	e.state.AgeRange = "o18"
	skipYouth := true
	e.state.ShouldSkipYouthTOS = &skipYouth
	e.state.ScreenVisited = append(e.state.ScreenVisited, "bloks.caa.reg.birthday")

	regInfo := buildRegInfoJSON(e.p, e.state)
	cip := map[string]any{
		"accounts_list":                     []any{},
		"client_timezone":                   "Asia/Ho_Chi_Minh",
		"aac":                               buildAACJSON(e.p),
		"birthday_or_current_date_string":   bday,
		"os_age_range":                      "",
		"birthday_timestamp":                bdayTS,
		"lois_settings":                     map[string]any{"lois_token": ""},
		"cloud_trust_token":                 nil,
		"zero_balance_state":                "",
		"network_bssid":                     nil,
		"should_skip_youth_tos":             0,
		"is_youth_regulation_flow_complete": 0,
	}
	sp := e.commonServerParams(6)
	sp["reg_info"] = regInfo

	resp, _, err := e.bloksPost(ctx,
		"com.bloks.www.bloks.caa.reg.birthday.async", cip, sp, nil)
	if err != nil {
		return fmt.Errorf("setBirthday: %w", err)
	}
	e.updateRegContext(resp)
	e.logf("setBirthday OK (%s)", bday)
	return nil
}

func birthdayUnix(year, month, day int) int64 {
	// Approximate unix timestamp: rough seconds since epoch for Jan 1 of year + month + day offset
	// Not using time package to avoid import issues; calculation is close enough for IG
	daysFromEpoch := int64((year-1970)*365 + (year-1970)/4 - (year-1970)/100 + (year-1970)/400)
	monthDays := [13]int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
		monthDays[2] = 29
	}
	for m := 1; m < month; m++ {
		daysFromEpoch += int64(monthDays[m])
	}
	daysFromEpoch += int64(day - 1)
	return daysFromEpoch * 86400
}

// setName — step 7.
func (e *igAndroidEngine) setName(ctx context.Context, name string) error {
	e.state.FullName = name
	e.state.ScreenVisited = append(e.state.ScreenVisited, "CAA_REG_IG_NAME_SCREEN")

	regInfo := buildRegInfoJSON(e.p, e.state)
	cip := map[string]any{
		"full_name": name,
		"aac":       buildAACJSON(e.p),
	}
	sp := e.commonServerParams(7)
	sp["reg_info"] = regInfo

	resp, _, err := e.bloksPost(ctx,
		"com.bloks.www.bloks.caa.reg.name_ig_and_soap.async", cip, sp, nil)
	if err != nil {
		return fmt.Errorf("setName: %w", err)
	}
	e.updateRegContext(resp)
	e.logf("setName OK (%s)", name)
	return nil
}

// setUsername — step ~8.
// CIP key is "validation_text" (not "username"). username=null in reg_info at this step;
// only username_prefill is set. After success, we promote it to Username for createAccount.
func (e *igAndroidEngine) setUsername(ctx context.Context, username string) error {
	e.state.UsernamePrefill = username
	e.state.ScreenVisited = appendScreenOnce(e.state.ScreenVisited, "CAA_REG_IG_USERNAME")

	regInfo := buildRegInfoJSON(e.p, e.state)
	cip := map[string]any{
		"validation_text":    username,
		"aac":                buildAACJSON(e.p),
		"family_device_id":   e.p.FamilyDeviceID,
		"device_id":          e.p.AndroidID,
		"lois_settings":      map[string]any{"lois_token": ""},
		"cloud_trust_token":  nil,
		"zero_balance_state": "",
		"network_bssid":      nil,
		"qe_device_id":       e.p.DeviceID,
	}
	sp := e.commonServerParams(8)
	sp["reg_info"] = regInfo
	sp["text_input_id"] = randQplID()
	sp["suggestions_container_id"] = randQplID()
	sp["action"] = 1
	sp["screen_id"] = randQplID()
	sp["post_tos"] = 0
	sp["input_id"] = randQplID()

	resp, _, err := e.bloksPost(ctx,
		"com.bloks.www.bloks.caa.reg.username.async", cip, sp, nil)
	if err != nil {
		return fmt.Errorf("setUsername: %w", err)
	}
	e.updateRegContext(resp)
	// Promote prefill to confirmed username for subsequent steps
	e.state.Username = username
	e.logf("setUsername OK (%s)", username)
	return nil
}

// createAccount — step 9. Retry khi integrity_block (rate limit tạm thời).
func (e *igAndroidEngine) createAccount(ctx context.Context, addr, username, name string) (igcore.IGSession, error) {
	e.state.ScreenVisited = append(e.state.ScreenVisited,
		"bk_caa_reg_icon_text_list_tos_screen")

	// Capture IGRegisterVIP: create.account KHÔNG gửi header x-ig-attest-params,
	// reg_info.attestation_result = null. Bỏ injection attest params.

	const maxAttempts = 3 // retry: system_error → học jurisdiction + giữ IP; integrity_block → đổi IP
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		regInfo := buildRegInfoJSON(e.p, e.state)
		cip := map[string]any{
			"reached_from_tos_screen": 1,
			"machine_id":              e.p.MachineID,
			"aac":                     buildAACJSON(e.p),
		}
		sp := e.commonServerParams(9)
		sp["reg_info"] = regInfo
		sp["bloks_controller_source"] = "bk_caa_reg_icon_text_list_tos_screen"

		resp, _, err := e.bloksPost(ctx,
			"com.bloks.www.bloks.caa.reg.create.account.async", cip, sp, nil)
		if err != nil {
			return igcore.IGSession{}, fmt.Errorf("createAccount: %w", err)
		}

		igSess := igcore.ParseIGSession(resp)
		if igSess.SessionID == "" {
			// Fallback: IG may send session via Set-Cookie headers (stored in jar) not in body
			igSess = sessionFromJar(e.sess)
			if igSess.SessionID != "" {
				e.logf("createAccount OK (from jar) uid=%s", igSess.UID)
			}
		} else {
			// Enrich with jar cookies (mid, datr, ig_did, rur come from Set-Cookie headers)
			jar := sessionFromJar(e.sess)
			if igSess.Mid == "" {
				igSess.Mid = jar.Mid
			}
			if igSess.Mid == "" && e.p.MachineID != "" {
				igSess.Mid = e.p.MachineID // fallback: ig-set-x-mid không tự vào jar
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
			incCreateOK()
			e.logf("createAccount OK uid=%s mid=%.8s…", igSess.UID, igSess.Mid)
			return igSess, nil
		}

		isIntegrity := strings.Contains(resp, "integrity_block")
		isSystemErr := strings.Contains(resp, "system_error")
		isLOIS := strings.Contains(resp, "LOIS:lois_token") || strings.Contains(resp, "lois_token")
		if isIntegrity || isSystemErr || isLOIS {
			if attempt < maxAttempts {
				if ctx.Err() != nil {
					return igcore.IGSession{}, ctx.Err()
				}
				// Quyết định đổi IP hay giữ IP (port từ iOS Bloks):
				//   integrity_block / lois → ĐỔI IP (IP flag/gate, cần IP sạch).
				//   system_error → GIỮ IP + học jurisdiction IG trả về → retry cùng IP khớp
				//                  GeoIP → qua. Đổi IP lúc này làm jurisdiction vừa học stale → vô dụng.
				rotate := isIntegrity
				reason := "integrity_block"
				if isSystemErr {
					reason = "system_error"
					if m := regJurisdictionRe.FindStringSubmatch(resp); len(m) > 1 {
						if igCC := m[1]; e.state.Jurisdiction != igCC {
							e.logf("createAccount %d/%d: system_error jurisdiction %q→%q", attempt, maxAttempts, e.state.Jurisdiction, igCC)
							e.state.Jurisdiction = igCC
						}
					}
				} else if isLOIS {
					rotate = true
					reason = "lois_gate"
					if attempt == 1 {
						if dir := debugDir(); dir != "" {
							writeDebug(dir, "createaccount_lois.txt", resp)
						}
					}
				}
				if rotate {
					e.logf("createAccount attempt %d/%d: %s → rotate IP + retry", attempt, maxAttempts, reason)
					// Đổi IP: tạo session mới (giữ proxy mới). reg_info rebuild ở vòng sau.
					newProxy := igcore.RotateSession(e.proxyStr)
					if newSess, err := newAndroidSession(newProxy); err == nil {
						e.sess.client.CloseIdleConnections() // close session CŨ trước khi thay
						e.sess = newSess
						e.proxyStr = newProxy
					}
				} else {
					// Giữ IP + session (cookie jar) — chỉ retry với jurisdiction mới đã học.
					e.logf("createAccount attempt %d/%d: %s → giữ IP + retry (jurisdiction khớp GeoIP)", attempt, maxAttempts, reason)
				}
				continue
			}
			reason := "integrity_block"
			if isSystemErr {
				reason = "system_error"
				incSystemErr()
			} else if isLOIS {
				reason = "lois_gate"
				incIntegrity()
			} else {
				incIntegrity()
			}
			return igcore.IGSession{}, fmt.Errorf("createAccount: %s sau %d lần thử", reason, maxAttempts)
		}

		// Save unexpected GetPayload response for investigation
		if strings.Contains(resp, "GetPayload") {
			if dir := debugDir(); dir != "" {
				writeDebug(dir, fmt.Sprintf("createaccount_getpayload_%d.txt", attempt), resp)
			}
		}
		incNoSession()
		captureNoSessionSample(resp) // dump 3 mẫu đầu → xem parser có bỏ sót session không
		return igcore.IGSession{}, fmt.Errorf("createAccount: no session in response (%.200s)", resp)
	}
	return igcore.IGSession{}, fmt.Errorf("createAccount: thất bại sau %d lần", maxAttempts)
}

// checkLive gọi GET i.instagram.com/api/v1/accounts/current_user/ dùng TLS client
// Android hiện có (cùng fingerprint + proxy). Trả "live" | "checkpoint" |
// "suspended" | "die" | "unknown".
func (e *igAndroidEngine) checkLive(ctx context.Context, fullCookie string) string {
	if fullCookie == "" {
		return "unknown"
	}
	req, err := fhttp.NewRequestWithContext(ctx, "GET",
		igAndroidBase+"/api/v1/accounts/current_user/?edit=true", nil)
	if err != nil {
		return "unknown"
	}
	req.Header[fhttp.HeaderOrderKey] = []string{
		"cookie", "user-agent", "x-ig-app-id", "x-csrftoken", "accept", "accept-encoding",
	}
	req.Header.Set("Cookie", fullCookie)
	req.Header.Set("User-Agent", e.p.UserAgent)
	req.Header.Set("X-IG-App-ID", igAndroidAppID)
	req.Header.Set("X-CSRFToken", extractCSRF(fullCookie))
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "zstd")

	resp, err := e.sess.client.Do(req)
	if err != nil {
		return "unknown"
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	body := e.sess.decode(resp.Header.Get("Content-Encoding"), raw)
	return igcore.ClassifyLiveResponse(resp.StatusCode, body)
}

// extractCSRF lấy csrftoken từ cookie string.
func extractCSRF(cookie string) string {
	for _, part := range strings.Split(cookie, ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 && strings.TrimSpace(kv[0]) == "csrftoken" {
			return strings.TrimSpace(kv[1])
		}
	}
	return ""
}

// ── Name / username helpers ──────────────────────────────────────────────────────

func buildName(input *instagram.RegInput) string {
	first := strings.TrimSpace(input.FirstName)
	last := strings.TrimSpace(input.LastName)
	full := strings.TrimSpace(last + " " + first)
	if full != "" {
		return full
	}
	pool := []string{"Alex", "Jordan", "Taylor", "Morgan", "Casey", "Riley", "Avery", "Quinn"}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	n := int(b[0]) % len(pool)
	suffix := int(b[1])%9000 + 1000
	return fmt.Sprintf("%s%d", pool[n], suffix)
}

// buildUsername sinh username giống thật từ pool tên (first+last, ~120x120 US names),
// nhiều pattern + separator + số kiểu người thật (năm sinh / 2–4 số). Thay kiểu cũ
// {từ-thiên-nhiên}.{7số} (tiger.1234567) dễ lộ pattern bot khi tạo hàng loạt.
// Dùng CHUNG cho cả iOS (ig_ios_bloks) lẫn Android (ig_android).
func buildUsername() string {
	p := fakeinfo.RandomFakeProfile()
	first := sanitizeUsername(p.FirstName)
	last := sanitizeUsername(p.LastName)
	if first == "" {
		first = "alex"
	}
	if last == "" {
		last = "smith"
	}

	b := make([]byte, 5)
	_, _ = rand.Read(b)
	// separator nghiêng về không-dấu (giống đa số username thật)
	sep := []string{"", ".", "_", ""}[int(b[0])%4]

	// Số đuôi entropy CAO để giảm trùng (cả nội bộ lẫn với username IG đã tồn tại).
	// Bỏ kiểu 2 chữ số (00–99) quá dễ trùng. Ưu tiên 5–7 chữ số.
	var num string
	switch int(b[1]) % 10 {
	case 0, 1: // 20%: năm sinh 1985–2010 (trông tự nhiên)
		num = strconv.Itoa(1985 + randN(26))
	case 2, 3: // 20%: 4 chữ số 1000–9999
		num = strconv.Itoa(1000 + randN(9000))
	default: // 60%: 5–7 chữ số 100000–9999999 (entropy cao nhất → ít trùng)
		num = strconv.Itoa(100000 + randN(9900000))
	}

	// thân username (nghiêng về first+last)
	var stem string
	switch int(b[3]) % 4 {
	case 1:
		stem = first[:1] + last // bhamilton
	case 2:
		stem = first // brian
	default:
		stem = first + sep + last // brian.hamilton / brianhamilton
	}

	// đôi khi chèn separator trước số: brian.hamilton_1995
	join := ""
	if sep != "" && int(b[4])%3 == 0 {
		join = sep
	}
	return clampUsername(stem + join + num)
}

// sanitizeUsername giữ a-z0-9 (lowercase), bỏ dấu/khoảng trắng — IG username chỉ
// cho phép chữ thường, số, '.', '_'.
func sanitizeUsername(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var sb strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// clampUsername đảm bảo IG-valid: <=30 ký tự, không bắt đầu/kết thúc bằng '.'/'_'.
func clampUsername(s string) string {
	if len(s) > 30 {
		s = s[:30]
	}
	s = strings.Trim(s, "._")
	if s == "" {
		b := make([]byte, 3)
		_, _ = rand.Read(b)
		s = fmt.Sprintf("user%d", 1000+int(b[0])<<8+int(b[1]))
	}
	return s
}

// randN trả số nguyên crypto-random trong [0, max). max<=0 → 0.
// Dùng cho số đuôi username entropy cao (giảm trùng).
func randN(max int) int {
	if max <= 0 {
		return 0
	}
	bi, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(bi.Int64())
}

// appendScreenOnce thêm screen vào ScreenVisited nếu CHƯA có (tránh trùng khi
// retry username gọi setUsername nhiều lần).
func appendScreenOnce(screens []string, s string) []string {
	for _, x := range screens {
		if x == s {
			return screens
		}
	}
	return append(screens, s)
}

func buildPassword() string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	out := make([]byte, 10)
	for i := range b {
		out[i] = chars[int(b[i])%len(chars)]
	}
	return "Wm" + string(out) + "!"
}

// ── Registerer implementation ────────────────────────────────────────────────────

type igAndroidRegisterer struct{}

func (r *igAndroidRegisterer) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	// agedDev: mid mượn từ pool — gắn nhãn [mid:...] vào mọi log của luồng này.
	var agedDev *igcore.AgedDevice
	midTag := func() string {
		if agedDev != nil {
			return "[mid:" + agedDev.Mid + "] "
		}
		return ""
	}

	status := func(msg string) {
		if onStatus != nil {
			onStatus(midTag() + msg)
		}
	}

	fail := func(stage, msg string) *instagram.RegResult {
		return &instagram.RegResult{
			Success: false,
			Email:   input.Email,
			Message: fmt.Sprintf("%s[igandroid/%s] %s", midTag(), stage, msg),
		}
	}

	if strings.TrimSpace(input.Email) == "" || input.GetOTP == nil {
		return &instagram.RegResult{
			Success: false,
			Message: "igandroid: cần Email + GetOTP callback",
		}
	}

	proxyStr := igcore.RotateSession(input.Proxy)

	// ── Build session + profile ───
	sess, err := newAndroidSession(proxyStr)
	if err != nil {
		return fail("session", err.Error())
	}
	// Giải phóng conn + readLoop goroutine khi reg xong (tránh leak → CPU tăng dần).
	defer sess.client.CloseIdleConnections()
	p, err := newAndroidProfile("en_GB")
	if err != nil {
		return fail("profile", err.Error())
	}

	// Báo UA sớm để UI hiển thị ngay từ đầu flow (pattern "ua:<value>")
	status("ua:" + p.UserAgent)

	// ── Inject aged device nếu pool có — làm trước mọi request ───
	if SharedAndroidDevicePool != nil {
		if dev := SharedAndroidDevicePool.Next(); dev != nil {
			injectAgedDevice(sess, p, dev)
			agedDev = dev
			status("inject aged device") // nhãn [mid:...] tự thêm bởi status()
		}
	}

	// ── Country detection for youth_regulation_config ───
	jurisdiction := "VN"
	{
		ccCh := make(chan string, 1)
		ccCtx, ccCancel := context.WithTimeout(ctx, 4*time.Second)
		go func() {
			// Tái dùng client reg + ctx CÓ timeout → goroutine không lingering khi proxy chậm.
			ccCh <- sess.checkProxyCountry(ccCtx)
		}()
		cc := ""
		select {
		case cc = <-ccCh:
		case <-ccCtx.Done():
		}
		ccCancel()
		if cc != "" {
			jurisdiction = cc
		}
	}

	// ── Attestation ĐẦU TIÊN — lấy X-MID (machine_id). Capture IGRegisterVIP: đây là
	//    request đầu tiên, KHÔNG gọi qe/sync (password plaintext nên không cần key mã hóa). ───
	status("attestation")
	_, xmid, atErr := sess.attestKeystore(ctx, p)
	if atErr != nil {
		status(fmt.Sprintf("attestation warning: %v", atErr))
	}
	if xmid != "" {
		p.MachineID = xmid // X-MID từ ig-set-x-mid response, dùng cho mọi request sau
	}

	state := &regInfoState{Jurisdiction: jurisdiction}
	eng := &igAndroidEngine{
		sess:     sess,
		p:        p,
		state:    state,
		proxyStr: proxyStr,
		log: func(f string, a ...any) {
			status(fmt.Sprintf(f, a...))
		},
	}

	addr := strings.TrimSpace(input.Email)
	password := input.Password
	if password == "" {
		password = buildPassword()
	}
	name := buildName(input)
	if name == "" {
		name = "Alex1234"
	}

	// ── Step 0: submit email ───
	status("submitEmail")
	if err := eng.submitEmail(ctx, addr); err != nil {
		return fail("submitEmail", err.Error())
	}

	// ── Read OTP via callback ───
	status("readOTP")
	otp, err := input.GetOTP(ctx)
	if err != nil || otp == "" {
		msg := "GetOTP failed"
		if err != nil {
			msg = err.Error()
		}
		return fail("readOTP", msg)
	}

	// ── Step 3: confirm OTP ───
	status("confirmOTP")
	if err := eng.confirmOTP(ctx, addr, otp); err != nil {
		return fail("confirmOTP", err.Error())
	}

	// ── Step 4: set password ───
	status("setPassword")
	if err := eng.setPassword(ctx, addr, password); err != nil {
		return fail("setPassword", err.Error())
	}

	// ── Step 6: set birthday ───
	status("setBirthday")
	if err := eng.setBirthday(ctx); err != nil {
		return fail("setBirthday", err.Error())
	}

	// ── Step 7: set name ───
	status("setName")
	if err := eng.setName(ctx, name); err != nil {
		return fail("setName", err.Error())
	}

	// (attestation đã chạy ở đầu flow — capture không có bước attestation giữa name↔username)

	// ── Set username (retry đổi username khi trùng/không hợp lệ) ───
	var username string
	var unameErr error
	for ut := 1; ut <= 4; ut++ {
		username = buildUsername()
		status("setUsername")
		unameErr = eng.setUsername(ctx, username)
		if unameErr == nil {
			break
		}
		status(fmt.Sprintf("setUsername trùng/lỗi (try %d/4) → đổi username", ut))
	}
	if unameErr != nil {
		return fail("setUsername", unameErr.Error())
	}

	// ── Step 9: create account ───
	status("createAccount")
	igSess, err := eng.createAccount(ctx, addr, username, name)
	if err != nil {
		return fail("createAccount", err.Error())
	}

	// ── Harvest device → pool cho lần reg sau. Lấy từ igSess (merge mid từ X-MID
	// + ig_did/datr từ response body) thay vì jar trần (jar bloks thường thiếu
	// ig_did/datr). dedupe theo mid trong Add().
	if SharedAndroidDevicePool != nil && igSess.SessionID != "" {
		if added := SharedAndroidDevicePool.Add(igSess.Mid, igSess.Datr, igSess.IgDID); added {
			status(fmt.Sprintf("harvest device mid=%.8s… → pool", igSess.Mid))
		}
	}

	// KHÔNG checklive ở đây (block slot ~15s). createAccount trả sessionid = reg OK.
	// Live/Die check ASYNC ở app_register → slot nhả ngay cho luồng kế.
	status("reg OK (checklive async)")
	return &instagram.RegResult{
		UID:            igSess.UID,
		Username:       username,
		Password:       password,
		Cookie:         igSess.FullCookie,
		Email:          addr,
		DeviceID:       p.DeviceID,
		FamilyDeviceID: p.FamilyDeviceID,
		UserAgent:      p.UserAgent,
		LiveStatus:     "",
		Success:        true,
		Message:        "ok",
	}
}

// SubmitEmailOTP triggers an OTP email send for the given address using the Android Bloks
// contactpoint_email.async endpoint. Call this from the iOS flow when graphql_www returns
// an OTP confirmation screen — the iOS client lacks a Bloks reg_context so it cannot call
// the Bloks endpoint directly; this function creates a minimal Android Bloks session instead.
//
// waterfallID: pass the iOS session's waterfallID so the Android Bloks session uses the same
// waterfall, making the OTP IG sends linkable to the iOS graphql_www session on retry.
// Pass "" to generate a fresh waterfall.
func SubmitEmailOTP(ctx context.Context, email, proxyStr, waterfallID string) error {
	sess, err := newAndroidSession(proxyStr)
	if err != nil {
		return fmt.Errorf("SubmitEmailOTP session: %w", err)
	}
	defer sess.client.CloseIdleConnections()
	p, err := newAndroidProfile("en_GB")
	if err != nil {
		return fmt.Errorf("SubmitEmailOTP profile: %w", err)
	}
	if waterfallID != "" {
		p.WaterfallID = waterfallID
	}
	_, _, xmid, qErr := sess.qeSync(ctx, p)
	if qErr != nil {
		return fmt.Errorf("SubmitEmailOTP qeSync: %w", qErr)
	}
	if xmid != "" {
		p.MachineID = xmid
	}
	state := &regInfoState{ContactPoint: email}
	eng := &igAndroidEngine{
		sess:  sess,
		p:     p,
		state: state,
	}
	if err := eng.submitEmail(ctx, email); err != nil {
		return fmt.Errorf("SubmitEmailOTP submitEmail: %w", err)
	}
	return nil
}

// QeSync chạy qe/sync và trả về mid + ig_did thực tế từ IG server.
// Dùng để xem format thực tế hoặc pre-harvest pool.
func QeSync(ctx context.Context, proxyStr string) (mid, igDID, ua string, err error) {
	sess, sErr := newAndroidSession(proxyStr)
	if sErr != nil {
		return "", "", "", fmt.Errorf("session: %w", sErr)
	}
	defer sess.client.CloseIdleConnections() // pre-harvest gọi 400×/run → bắt buộc close
	p, pErr := newAndroidProfile("en_GB")
	if pErr != nil {
		return "", "", "", fmt.Errorf("profile: %w", pErr)
	}
	_, _, xmid, qErr := sess.qeSync(ctx, p)
	if qErr != nil {
		return "", "", "", fmt.Errorf("qeSync: %w", qErr)
	}
	u, _ := url.Parse("https://i.instagram.com")
	for _, c := range sess.client.GetCookies(u) {
		switch c.Name {
		case "mid":
			mid = c.Value
		case "ig_did":
			igDID = c.Value
		}
	}
	if mid == "" {
		mid = xmid // fallback về header nếu cookie chưa set
	}
	return mid, igDID, p.UserAgent, nil
}

// GenerateMid tạo fake mid theo pattern IG: "aik0" + 24-char base64url.
// Dùng để test xem IG có validate mid server-side không.
func GenerateMid() string {
	return "aik0" + randBase64URL(24)
}

// MidFromDatr tạo fake mid bằng cách ghép prefix cố định với datr 24-char từ FB pool.
func MidFromDatr(datr string) string {
	return "aik0" + datr
}

// QeSyncWithMid inject midIn vào session trước qe/sync, trả về:
//   - midUsed: mid thực sự đang dùng sau qe/sync
//   - midServer: mid server trả về (ig-set-x-mid)
//   - serverOverride: true nếu server trả mid khác với midIn
func QeSyncWithMid(ctx context.Context, proxyStr, midIn string) (midUsed, midServer string, serverOverride bool, err error) {
	sess, sErr := newAndroidSession(proxyStr)
	if sErr != nil {
		return "", "", false, fmt.Errorf("session: %w", sErr)
	}
	defer sess.client.CloseIdleConnections()
	p, pErr := newAndroidProfile("en_GB")
	if pErr != nil {
		return "", "", false, fmt.Errorf("profile: %w", pErr)
	}
	// Inject mid trước qe/sync
	if midIn != "" {
		p.MachineID = midIn
		u, _ := url.Parse("https://i.instagram.com")
		sess.client.SetCookies(u, []*fhttp.Cookie{
			{Name: "mid", Value: midIn, Domain: ".instagram.com", Path: "/"},
		})
	}
	_, _, xmid, qErr := sess.qeSync(ctx, p)
	if qErr != nil {
		return "", "", false, fmt.Errorf("qeSync: %w", qErr)
	}
	midServer = xmid
	// mid sau qe/sync: nếu server trả mid mới và khác → server override
	serverOverride = xmid != "" && xmid != midIn
	if midIn != "" {
		midUsed = midIn // giữ mid inject nếu có
	} else {
		midUsed = xmid
	}
	return midUsed, midServer, serverOverride, nil
}

// PreHarvestPool chạy n qe/sync song song qua proxyStr, thu mid thật từ IG server
// và add vào pool. Giống cơ chế datr pool bên FB — chạy trước batch reg để pool
// có mid aged sẵn, inject vào session mới → thiết bị "có lịch sử".
//
// Dùng SharedAndroidDevicePool nếu pool != nil, hoặc pool truyền vào.
// logFn nhận thông báo tiến độ (có thể nil).
func PreHarvestPool(ctx context.Context, proxyStr string, n int, pool *igcore.DevicePool, logFn func(string)) int {
	if pool == nil {
		pool = SharedAndroidDevicePool
	}
	if pool == nil || n <= 0 {
		return 0
	}
	const maxWorkers = 10
	sem := make(chan struct{}, maxWorkers)
	var mu sync.Mutex
	var wg sync.WaitGroup
	added := 0

	for i := 0; i < n; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			tCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			defer cancel()
			mid, igDID, _, err := QeSync(tCtx, proxyStr)
			if err != nil || mid == "" {
				return
			}
			if pool.Add(mid, "", igDID) {
				mu.Lock()
				added++
				cur := added
				mu.Unlock()
				if logFn != nil {
					logFn(fmt.Sprintf("[MidPool] +%d mid=%.12s…", cur, mid))
				}
			}
		}()
	}
	wg.Wait()
	return added
}

// ── Random helpers (crypto/rand based) ──────────────────────────────────────────

func randHex(bytes int) string {
	b := make([]byte, bytes)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func randAlphanumeric(n int) string {
	const al = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = al[int(b[i])%len(al)]
	}
	return string(b)
}

func randBase64URL(n int) string {
	const al = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = al[int(b[i])%len(al)]
	}
	return string(b)
}

// randIntn returns a cryptographically random int in [0, n).
func randIntn(n int) int {
	if n <= 0 {
		return 0
	}
	max := new(big.Int).SetInt64(int64(n))
	v, _ := rand.Int(rand.Reader, max)
	return int(v.Int64())
}

// randQplID returns a random float-like int64 for INTERNAL__latency_qpl_instance_id.
func randQplID() int64 {
	b := make([]byte, 7)
	_, _ = rand.Read(b)
	var v int64
	for _, x := range b {
		v = v<<8 | int64(x)
	}
	if v < 0 {
		v = -v
	}
	return v
}

func firstN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// NewRegisterer returns an Instagram Android Bloks registerer for direct use.
// Useful when another flow (e.g. iOS) needs to fall back to the Android multi-step flow.
func NewRegisterer() instagram.Registerer {
	return &igAndroidRegisterer{}
}

// ── Plugin registration ──────────────────────────────────────────────────────────

func init() {
	instagram.RegisterPlatformRegisterer(instagram.PlatformIGAndroid, func() instagram.Registerer {
		return &igAndroidRegisterer{}
	})
}
