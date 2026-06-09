// body.go — S399 register body builders + app constants.
//
// Khác hoàn toàn s555-s560 (Bloks/GraphQL):
//   - 2 endpoint: POST /app/users (register) + POST /auth/login (lấy token)
//   - friendly-name: registerAccount + authenticate
//   - Body form-urlencoded thuần, có sig MD5 + app_secret
//   - UA có Dalvik prefix: "Dalvik/2.1.0 (Linux; U; Android <os>; <model> Build/<buildID>) [FBAN/FB4A;FBAV/399.0.0.24.93;...]"
//
// Captured traffic confirmed (May 2026): /app/users trả new_user_id + machine_id;
// /auth/login dùng new_user_id làm email + machine_id từ step 1 → trả EAA token.
package s399

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// ─── S399 app constants ───────────────────────────────────────────────────────

const (
	s399FBAV = "399.0.0.24.93"
	s399FBBV = "440587081"

	s399APIKey       = "882a8490361da98702bf97a021ddc14d"
	s399OAuthToken   = "350685531728|62f8ce9f74b12f84c123cc23437a4a32"
	s399AppSecret    = "62f8ce9f74b12f84c123cc23437a4a32" // tail của OAuth token, dùng cho sig MD5
	s399FriendlyReg  = "registerAccount"
	s399FriendlyAuth = "authenticate"
	s399CallerReg    = "RegistrationCreateAccountFragment"
	s399CallerAuth   = "Fb4aAuthHandler"

	// OriginalUA — UA gốc cố định cho s399 (SM-S911B, Android 15, Viettel placeholder).
	// Chỉ FBCR thay theo IP — UseOriginalUA=true; các field khác giữ nguyên.
	OriginalUA = "Dalvik/2.1.0 (Linux; U; Android 15; SM-S911B Build/AP3A.240905.015.A2) [FBAN/FB4A;FBAV/399.0.0.24.93;FBPN/com.facebook.katana;FBLC/en_GB;FBBV/440587081;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBDV/SM-S911B;FBSV/15;FBLC/en_GB;FBOP/1;FBCA/arm64-v8a:armeabi-v7a;]"
)

// ─── Sig calculation ─────────────────────────────────────────────────────────
//
// FB classic sig: md5(<sorted param=value...> + app_secret).
// Sort theo key, concat dạng "key1=value1key2=value2..." (KHÔNG có dấu &),
// append app_secret, MD5, hex lowercase.

func computeSig(params map[string]string) string {
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
	sb.WriteString(s399AppSecret)
	sum := md5.Sum([]byte(sb.String()))
	return hex.EncodeToString(sum[:])
}

// formEncode chuyển map thành body x-www-form-urlencoded (đã sort key + sig + access_token cuối).
func formEncode(params map[string]string) string {
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

// ─── Body builders ───────────────────────────────────────────────────────────

// RegisterParams — input cần build body cho POST /app/users.
type RegisterParams struct {
	Email          string // contactpoint email (vd "block.oi22vn@gmail.com")
	FirstName      string
	LastName       string
	Gender         string // "M" hoặc "F"
	BirthdayYMD    string // "YYYY-MM-DD"
	EncPassword    string // "#PWD_FB4A:2:<ts>:<encrypted>"
	SessionID      string // UUID generated client-side (ổn định trong 1 reg session)
	DeviceID       string // UUID
	RegInstance    string // UUID (thường giống DeviceID)
	FamilyDeviceID string // UUID
	AdvertisingID  string // UUID
	Locale         string // "en_GB"
	CountryCode    string // "VN"
	Jazoest        string // số nhỏ vd "22293"
	// Acquired timestamps (epoch ms) — dùng để fingerprint reg flow trên client
	StartTimestampMs            int64
	StartCompletedTimestampMs   int64
	NameAcquiredTimestampMs     int64
	BirthdayAcquiredTimestampMs int64
	GenderAcquiredTimestampMs   int64
	CPAcquiredTimestampMs       int64
	PWAcquiredTimestampMs       int64
	TOSAcquiredTimestampMs      int64
	// SnNonce / SnResult — SafetyNet attestation values (FB now ignores these but vẫn cần gửi)
	SnNonce  string
	SnResult string
}

// buildRegisterBody sinh form-urlencoded body cho POST /app/users.
// Trả body string + sig đã compute (sig include trong body).
func buildRegisterBody(p RegisterParams) string {
	params := map[string]string{
		"email":                        p.Email,
		"firstname":                    p.FirstName,
		"lastname":                     p.LastName,
		"gender":                       p.Gender,
		"password":                     p.EncPassword,
		"allow_local_pw":               "true",
		"birthday":                     p.BirthdayYMD,
		"session_id":                   p.SessionID,
		"consent_standards_test_group": "test",
		"play_service_version":         "-1",
		"cp_acquired_timestamp":        fmt.Sprintf("%d", p.CPAcquiredTimestampMs),
		"name_acquired_timestamp":      fmt.Sprintf("%d", p.NameAcquiredTimestampMs),
		"pw_acquired_timestamp":        fmt.Sprintf("%d", p.PWAcquiredTimestampMs),
		"start_timestamp":              fmt.Sprintf("%d", p.StartTimestampMs),
		"gender_acquired_timestamp":    fmt.Sprintf("%d", p.GenderAcquiredTimestampMs),
		"start_completed_timestamp":    fmt.Sprintf("%d", p.StartCompletedTimestampMs),
		"birthday_acquired_timestamp":  fmt.Sprintf("%d", p.BirthdayAcquiredTimestampMs),
		"tos_acquired_timestamp":       fmt.Sprintf("%d", p.TOSAcquiredTimestampMs),
		"device_has_previous_login":    "false",
		"return_multiple_errors":       "true",
		"attempt_login":                "true",
		"reg_instance":                 p.RegInstance,
		"device_id":                    p.DeviceID,
		"generate_machine_id":          "true",
		"format":                       "json",
		"skip_session_info":            "true",
		"sn_nonce":                     p.SnNonce,
		"sn_result":                    p.SnResult,
		"jazoest":                      p.Jazoest,
		"side_loading_id":              "NO_FILE",
		"advertising_id":               p.AdvertisingID,
		"family_device_id":             p.FamilyDeviceID,
		"locale":                       p.Locale,
		"client_country_code":          p.CountryCode,
		"fb_api_req_friendly_name":     s399FriendlyReg,
		"fb_api_caller_class":          s399CallerReg,
		"api_key":                      s399APIKey,
	}
	params["sig"] = computeSig(params)
	params["access_token"] = s399OAuthToken
	return formEncode(params)
}

// LoginParams — input cần build body cho POST /auth/login.
type LoginParams struct {
	UID            string // UID lấy từ step 1 (response.new_user_id)
	EncPassword    string // mã hóa lại với timestamp mới
	DeviceID       string // giống step 1
	FamilyDeviceID string // giống step 1
	AdvertisingID  string // giống step 1
	MachineID      string // lấy từ step 1 response (machine_id)
	Locale         string
	CountryCode    string
	Jazoest        string
}

// buildLoginBody sinh form-urlencoded body cho POST /auth/login.
func buildLoginBody(p LoginParams) string {
	params := map[string]string{
		"adid":                       p.AdvertisingID,
		"format":                     "json",
		"device_id":                  p.DeviceID,
		"email":                      p.UID, // KEY: dùng UID làm email khi login sau reg
		"password":                   p.EncPassword,
		"generate_analytics_claim":   "1",
		"community_id":               "",
		"linked_guest_account_userid": "",
		"cpl":                        "true",
		"family_device_id":           p.FamilyDeviceID,
		"secure_family_device_id":    "",
		"credentials_type":           "password",
		"enroll_misauth":             "false",
		"generate_session_cookies":   "1",
		"error_detail_type":          "button_with_disabled",
		"source":                     "register_api",
		"machine_id":                 p.MachineID,
		"jazoest":                    p.Jazoest,
		"meta_inf_fbmeta":            "NO_FILE",
		"advertiser_id":              p.AdvertisingID,
		"encrypted_msisdn":           "",
		"currently_logged_in_userid": "0",
		"locale":                     p.Locale,
		"client_country_code":        p.CountryCode,
		"fb_api_req_friendly_name":   s399FriendlyAuth,
		"fb_api_caller_class":        s399CallerAuth,
		"api_key":                    s399APIKey,
	}
	params["sig"] = computeSig(params)
	params["access_token"] = s399OAuthToken
	return formEncode(params)
}
