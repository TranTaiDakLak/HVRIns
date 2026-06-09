// extras.go — Android post-register side effects + response parsing + password crypto.
//
// File này gộp 4 file cũ:
//   - pwdkey.go    → GetPwdKey + EncryptPassword (RSA+AES-GCM V3 format)
//   - logout.go    → LogoutAccount (POST /auth/expire_session sau reg)
//   - xzero.go     → fetchXZeroEH body builder + urlEncodeFull + constants
//   - response.go  → parseRegisterResponse + parseXZeroEHResponse
//
// Port 1:1 từ C# VerifyCloneVIP — thứ tự header, body form, regex giữ nguyên.
package android

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/google/uuid"

	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/httpx"
)

// ─── Constants dùng chung ─────────────────────────────────────────────────────

const (
	pwdKeyFetchFriendlyName         = "pwdKeyFetch"
	logoutFriendlyName              = "logout"
	fetchLoginDataBatchFriendlyName = "fetchLoginData-batch"
)

// defaultMetaZcaBlob — base64-encoded AKA/GPIA disabled blob (C# const).
// Dùng bởi GetXZeroEH + LogoutAccount + buildBatchHeaders.
// Khác V3 Register (dùng "empty_token"), các API non-Register vẫn dùng blob.
const defaultMetaZcaBlob = "eyJhbmRyb2lkIjp7ImFrYSI6eyJkYXRhVG9TaWduIjoiIiwiZXJyb3JzIjpbIktFWVNUT1JFX0RJU0FCTEVEX0JZX0NPTkZJRyJdfSwiZ3BpYSI6eyJ0b2tlbiI6IiIsImVycm9ycyI6WyJQTEFZX0lOVEVHUklUWV9ESVNBQkxFRF9CWV9DT05GSUciXX19fQ"

// ─── GetPwdKey + EncryptPassword (RSA+AES-GCM V3) ────────────────────────────
//
// Port 1:1 từ C# FacebookRegisterAPIAndroidV2:
//   - GetPwdKey      (V2.cs L290-344) — POST /pwd_key_fetch → public_key + key_id
//   - EncryptPassword(V2.cs L179-258) — RSA-PKCS1(rand_key) + AES-GCM(pw, rand_key, iv)
//     output `#PWD_FB4A:2:{ts}:{base64_blob}`
//
// Blob format (little-endian, byte order):
//
//	[0x01][key_id][iv 12B][len(encrypted_rand_key) uint16 LE][encrypted_rand_key]
//	[auth_tag 16B][encrypted_password]

// PwdKeyResult struct trả về từ GetPwdKey.
type PwdKeyResult struct {
	PublicKey string
	KeyID     int
	OK        bool
}

var (
	rePwdPublicKey = regexp.MustCompile(`public_key":"(.*?)"`)
	rePwdKeyID     = regexp.MustCompile(`key_id":(\d+)`)
)

// GetPwdKey gọi /pwd_key_fetch để lấy RSA public key.
// Port C# GetPwdKey (V2.cs L290-344).
func GetPwdKey(ctx context.Context, sess *session, profile fakeinfo.FullRegProfile) PwdKeyResult {
	locale := profile.Locale
	if locale == "" {
		locale = "en_US"
	}
	cc := profile.Sim.CountryCode
	if cc == "" {
		cc = "US"
	}

	postURL := instagram.BaseURLBGraph + "//pwd_key_fetch" // C# giữ double slash
	body := fmt.Sprintf(
		"device_id=%s&version=2&flow=CONTROLLER_INITIALIZATION&locale=%s"+
			"&client_country_code=%s&method=GET&fb_api_req_friendly_name=pwdKeyFetch"+
			"&fb_api_caller_class=Fb4aAuthHandler&access_token=%s",
		profile.DeviceID, locale, cc, url.QueryEscape(instagram.AndroidOAuthToken),
	)

	req, err := fhttp.NewRequestWithContext(ctx, "POST", postURL, strings.NewReader(body))
	if err != nil {
		return PwdKeyResult{}
	}
	h := buildPwdKeyHeaders(profile)
	for _, kv := range h {
		req.Header[kv[0]] = []string{kv[1]}
	}
	req.Header["content-type"] = []string{"application/x-www-form-urlencoded"}
	req.Header["content-length"] = []string{fmt.Sprintf("%d", len(body))}

	resp, err := sess.client.Do(req)
	if err != nil {
		return PwdKeyResult{}
	}
	defer resp.Body.Close()
	raw, _ := httpx.ReadBody(resp.Body, 64*1024)
	s := string(raw)
	if !strings.Contains(s, `"public_key"`) {
		return PwdKeyResult{}
	}

	pkMatch := rePwdPublicKey.FindStringSubmatch(s)
	idMatch := rePwdKeyID.FindStringSubmatch(s)
	if len(pkMatch) < 2 || len(idMatch) < 2 {
		return PwdKeyResult{}
	}
	var keyID int
	fmt.Sscanf(idMatch[1], "%d", &keyID)
	if keyID == 0 || pkMatch[1] == "" {
		return PwdKeyResult{}
	}
	return PwdKeyResult{PublicKey: pkMatch[1], KeyID: keyID, OK: true}
}

// EncryptPassword port C# EncryptPassword (V2.cs L179-258).
// Trả về `#PWD_FB4A:2:{ts}:{base64_blob}` nếu thành công, chuỗi rỗng nếu lỗi.
func EncryptPassword(password, publicKey string, keyID int) string {
	// 32-byte random key cho AES + 12-byte IV cho GCM.
	randKey := make([]byte, 32)
	iv := make([]byte, 12)
	if _, err := rand.Read(randKey); err != nil {
		return ""
	}
	if _, err := rand.Read(iv); err != nil {
		return ""
	}

	// Parse RSA public key (strip PEM header/footer nếu có).
	pubKey, err := parseRSAPublicKey(publicKey)
	if err != nil {
		return ""
	}

	// RSA PKCS#1 v1.5 encrypt randKey.
	encryptedRandKey, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, randKey)
	if err != nil {
		return ""
	}

	ts := time.Now().Unix()
	timeBytes := []byte(fmt.Sprintf("%d", ts))

	// AES-GCM với additional data = timeBytes (tag 16B).
	block, err := aes.NewCipher(randKey)
	if err != nil {
		return ""
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	// Seal sinh ciphertext||tag (16B tag ở cuối).
	sealed := aesgcm.Seal(nil, iv, []byte(password), timeBytes)
	// Tách ciphertext và auth tag.
	ctLen := len(sealed) - 16
	if ctLen < 0 {
		return ""
	}
	encryptedPassword := sealed[:ctLen]
	authTag := sealed[ctLen:]

	// Build blob: [0x01][keyId][iv][len(enc_rand_key) uint16 LE][enc_rand_key][tag][enc_pw]
	var buf []byte
	buf = append(buf, 0x01)
	buf = append(buf, byte(keyID))
	buf = append(buf, iv...)
	lenBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(lenBytes, uint16(len(encryptedRandKey)))
	buf = append(buf, lenBytes...)
	buf = append(buf, encryptedRandKey...)
	buf = append(buf, authTag...)
	buf = append(buf, encryptedPassword...)

	encoded := base64.StdEncoding.EncodeToString(buf)
	return fmt.Sprintf("#PWD_FB4A:2:%d:%s", ts, encoded)
}

// parseRSAPublicKey — strip PEM header/footer + base64 decode → x509 parse.
// Hỗ trợ cả dạng PEM và raw base64 (thường FB trả về raw).
func parseRSAPublicKey(publicKey string) (*rsa.PublicKey, error) {
	// C# normalize: replace \n / \r / spaces + strip PEM markers.
	k := publicKey
	k = strings.ReplaceAll(k, `\n`, "\n")
	k = strings.ReplaceAll(k, `\/`, "/")

	// Nếu có PEM markers → dùng pem.Decode; ngược lại coi như raw base64.
	var der []byte
	if strings.Contains(k, "-----BEGIN PUBLIC KEY-----") {
		block, _ := pem.Decode([]byte(k))
		if block == nil {
			return nil, fmt.Errorf("pem decode failed")
		}
		der = block.Bytes
	} else {
		k = strings.ReplaceAll(k, "-----BEGIN PUBLIC KEY-----", "")
		k = strings.ReplaceAll(k, "-----END PUBLIC KEY-----", "")
		k = strings.ReplaceAll(k, "\n", "")
		k = strings.ReplaceAll(k, "\r", "")
		k = strings.TrimSpace(k)
		var err error
		der, err = base64.StdEncoding.DecodeString(k)
		if err != nil {
			return nil, fmt.Errorf("base64 decode: %w", err)
		}
	}
	key, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return nil, fmt.Errorf("parse pkix: %w", err)
	}
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not RSA key")
	}
	return rsaKey, nil
}

// buildPwdKeyHeaders port C# GetPwdKey (V2.cs L298-308).
// Port BaseAndroidAPIHeadersWIFI + BaseAndroidDevicexConnectHeaders + extras.
func buildPwdKeyHeaders(profile fakeinfo.FullRegProfile) [][2]string {
	analyticsTag := `{"network_tags":{"product":"350685531728","purpose":"fetch","request_category":"graphql","retry_attempt":"0"},"application_tags":"graphservice"}`

	simHNI := profile.Sim.HNI
	if simHNI == "" {
		simHNI = "45204"
	}
	connType := profile.ConnectionType
	if connType == "" {
		connType = "WIFI"
	}

	return [][2]string{
		// access_token = "null" literal string (C# truyền "null")
		{"Authorization", "OAuth null"},
		{"X-Fb-Friendly-Name", pwdKeyFetchFriendlyName},
		{"X-Fb-Connection-Type", connType},
		{"X-Fb-Sim-Hni", simHNI},
		{"X-Fb-Net-Hni", simHNI},
		{"X-Zero-Eh", ""},
		{"X-Graphql-Client-Library", "graphservice"},
		{"X-Tigon-Is-Retry", "False"},
		{"X-Fb-Privacy-Context", "3643298472347298"},
		{"X-Graphql-Request-Purpose", "fetch"},
		{"X-Fb-Request-Analytics-Tags", analyticsTag},
		{"X-Fb-Http-Engine", "Tigon/Liger"},
		{"X-Fb-Client-Ip", "True"},
		{"X-Fb-Server-Cluster", "True"},
		// BaseAndroidDevicexConnectHeaders
		{"X-Fb-Device-Group", profile.DeviceGroup},
		{"X-Fb-Conn-Uuid-Client", generateConnUUIDClientV3()},
		// Extras cho pwd_key_fetch (V2.cs L306-308):
		{"X-Fb-Connection-Quality", "EXCELLENT"},
		{"X-Zero-F-Device-Id", profile.FamilyDeviceID},
		{"App-Scope-Id-Header", profile.DeviceID},
	}
}

// ─── LogoutAccount sau reg ───────────────────────────────────────────────────
//
// Port C# FacebookRegisterAPIAndroidV2.LogoutAccount L260-289.
//
// Flow:
//
//	POST b-graph.facebook.com/auth/expire_session
//	Body: reason=USER_INITIATED&device_id={dev}&retain_for_dbl=false&logout_source=REGISTRATION
//	      &locale={loc}&client_country_code={cc}&fb_api_req_friendly_name=logout
//	      &fb_api_caller_class=Fb4aLogoutOperationsHelper
//	Headers: BaseAndroidAPIHeadersWIFI(accessToken, "logout") + BaseAndroidDevicexConnectHeaders
//	         + X-Zero-F-Device-Id (FamilyDeviceId) + App-Scope-Id-Header (deviceid)
//	         + X-Meta-Zca=<base64 blob> + X-Fb-Connection-Quality=EXCELLENT

// LogoutAccount best-effort logout sau reg. Lỗi được nuốt.
func LogoutAccount(ctx context.Context, sess *session, profile fakeinfo.FullRegProfile, accessToken string) {
	locale := profile.Locale
	if locale == "" {
		locale = "en_US"
	}
	cc := profile.Sim.CountryCode
	if cc == "" {
		cc = "US"
	}

	body := "reason=USER_INITIATED" +
		"&device_id=" + profile.DeviceID +
		"&retain_for_dbl=false" +
		"&logout_source=REGISTRATION" +
		"&locale=" + locale +
		"&client_country_code=" + cc +
		"&fb_api_req_friendly_name=" + logoutFriendlyName +
		"&fb_api_caller_class=Fb4aLogoutOperationsHelper"

	postURL := "https://b-graph.facebook.com/auth/expire_session"
	req, err := fhttp.NewRequestWithContext(ctx, "POST", postURL, strings.NewReader(body))
	if err != nil {
		return
	}
	for _, kv := range buildLogoutHeaders(profile, accessToken) {
		req.Header[kv[0]] = []string{kv[1]}
	}
	req.Header["content-type"] = []string{"application/x-www-form-urlencoded"}

	resp, err := sess.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
}

func buildLogoutHeaders(profile fakeinfo.FullRegProfile, accessToken string) [][2]string {
	analyticsTag := `{"network_tags":{"product":"350685531728","purpose":"fetch","request_category":"graphql","retry_attempt":"0"},"application_tags":"graphservice"}`

	simHNI := profile.Sim.HNI
	if simHNI == "" {
		simHNI = "45204"
	}
	connType := profile.ConnectionType
	if connType == "" {
		connType = "WIFI"
	}

	return [][2]string{
		{"Authorization", "OAuth " + accessToken},
		{"X-Fb-Friendly-Name", logoutFriendlyName},
		{"X-Fb-Connection-Type", connType},
		{"X-Fb-Sim-Hni", simHNI},
		{"X-Fb-Net-Hni", simHNI},
		{"X-Zero-Eh", ""},
		{"X-Graphql-Client-Library", "graphservice"},
		{"X-Tigon-Is-Retry", "False"},
		{"X-Fb-Privacy-Context", "3643298472347298"},
		{"X-Graphql-Request-Purpose", "fetch"},
		{"X-Fb-Request-Analytics-Tags", analyticsTag},
		{"X-Fb-Http-Engine", "Tigon/Liger"},
		{"X-Fb-Client-Ip", "True"},
		{"X-Fb-Server-Cluster", "True"},
		{"X-Fb-Device-Group", profile.DeviceGroup},
		{"X-Fb-Conn-Uuid-Client", strings.ReplaceAll(uuid.New().String(), "-", "")},
		{"X-Zero-F-Device-Id", profile.FamilyDeviceID},
		{"App-Scope-Id-Header", profile.DeviceID},
		{"X-Meta-Zca", defaultMetaZcaBlob},
		{"X-Fb-Connection-Quality", "EXCELLENT"},
	}
}

// ─── GetXZeroEH (sau register thành công) ────────────────────────────────────
//
// Port từ C# FacebookRegisterAPIAndroidV2 (base Register) + GetXzeroEhMobileFormData.
// Flow identical với S23: batch POST `mobile_zero_campaign` với carrier_mcc/sim_mcc/
// interface/eligibility_hash/token_hash.
//
// Headers: BaseAndroidAPIHeadersWIFI + BaseAndroidDevicexConnectHeaders
//
//	+ X-Zero-F-Device-Id (FamilyDeviceId) + App-Scope-Id-Header (deviceid)
//	+ X-Zero-State=unknown + X-Meta-Zca=<base64 blob>
//	  (khác V3 Register dùng empty_token)

// buildXZeroEHBody port C# GetXzeroEhMobileFormData (FormDataBuilder.cs L215-219).
// Inner batch request body (URL-encoded form) gói vào `batch=[{...}]` JSON.
func buildXZeroEHBody(profile fakeinfo.FullRegProfile, accessToken, locale string) string {
	_ = accessToken // access_token ở header, không ở body
	mcc := profile.Sim.MCC
	mnc := profile.Sim.MNC
	conn := "wifi" // Android default WIFI; mobile.lte tùy profile extend sau
	cc := profile.Sim.CountryCode
	if cc == "" {
		cc = "US"
	}
	if locale == "" {
		locale = profile.Locale
	}
	if locale == "" {
		locale = "en_US"
	}

	inner := fmt.Sprintf(
		"carrier_mcc=%s&carrier_mnc=%s&sim_mcc=%s&sim_mnc=%s&format=json"+
			"&interface=%s&dialtone_enabled=false&needs_backup_rules=true"+
			"&token_hash=&request_reason=login&eligibility_hash="+
			"&locale=%s&client_country_code=%s&fb_api_req_friendly_name=fetchZeroToken",
		mcc, mnc, mcc, mnc, conn, locale, cc,
	)

	batchJSON := fmt.Sprintf(
		`[{"method":"POST","body":"%s","name":"fetchZeroToken","omit_response_on_success":false,"relative_url":"mobile_zero_campaign"}]`,
		strings.ReplaceAll(inner, `"`, `\"`),
	)

	return "batch=" + urlEncodeFull(batchJSON) +
		"&fb_api_caller_class=Fb4aAuthHandler" +
		"&fb_api_req_friendly_name=fetchLoginData-batch"
}

// urlEncodeFull — match C# WebUtility.UrlEncode (encode tất cả trừ ALPHA/DIGIT/-._~).
func urlEncodeFull(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", c)
	}
	return b.String()
}

// ─── Register response parser ────────────────────────────────────────────────
//
// Port CHÍNH XÁC từ C# FacebookRegisterAPIAndroid.Register() response parsing:
//
//	resstr = resstr.Replace("\\", "")   ← strip TẤT CẢ backslash trước
//	AccessToken: EAA(.*?)"
//	UID:         c_user","value":"(.*?)"
//	xs:          name":"xs","value":"(.*?)"
//	fr:          name":"fr","value":"(.*?)"
//	datr:        name":"datr","value":"(.*?)"

var (
	// C# exact patterns (sau khi strip backslash):
	reUserAccessToken = regexp.MustCompile(`EAAAAU[A-Za-z0-9+/=_-]{20,}`)
	reAccessToken     = regexp.MustCompile(`EAA([A-Za-z0-9+/=_-]{10,})`)
	reCUser           = regexp.MustCompile(`c_user","value":"(\d{10,})"`)
	reXS              = regexp.MustCompile(`name":"xs","value":"([^"]+)"`)
	reFR              = regexp.MustCompile(`name":"fr","value":"([^"]+)"`)
	reDATR            = regexp.MustCompile(`name":"datr","value":"([^"]+)"`)

	// Fallback patterns (nếu C# pattern không match)
	reAccessTokenFB = regexp.MustCompile(`EAA[A-Za-z0-9+/=_-]{20,}`)
	reUIDGeneric    = regexp.MustCompile(`"uid["\s:,\\]+(\d{10,})`)
	reCreatedUID    = regexp.MustCompile(`created_user(?:id)?["\s:,\\]+(\d{10,})`)
)

// RegResponse kết quả parsed từ register response.
type RegResponse struct {
	UID         string
	AccessToken string
	Cookie      string
	XS          string
	FR          string
	DATR        string
	Blocked     bool
	RawBody     string
}

// parseRegisterResponse parse response body từ graphql register.
// Chính xác theo C# flow:
//  1. resstr.Replace("\\", "") → strip toàn bộ backslash
//  2. regex trên string đã clean
func parseRegisterResponse(body, locale string) (*RegResponse, error) {
	resp := &RegResponse{RawBody: body}

	// C#: resstr.Replace("\\", "") — strip TẤT CẢ ký tự backslash
	clean := strings.ReplaceAll(body, "\\", "")

	// Detect các dạng block từ Facebook
	switch {
	case strings.Contains(body, "couldn't create an account for you"),
		strings.Contains(clean, "couldn't create an account for you"):
		resp.Blocked = true
		return resp, fmt.Errorf("Facebook blocked: account creation denied")
	case strings.Contains(clean, "integrity_block"):
		resp.Blocked = true
		return resp, fmt.Errorf("Facebook blocked: integrity_block (proxy/fingerprint bị nhận diện)")
	case strings.Contains(clean, `"create_failure"`) && strings.Contains(clean, `"created_userid",null`):
		resp.Blocked = true
		return resp, fmt.Errorf("Facebook blocked: create_failure")
	}

	if m := reUserAccessToken.FindString(clean); m != "" {
		resp.AccessToken = m
	}
	if resp.AccessToken == "" {
		if m := reAccessTokenFB.FindString(clean); m != "" {
			resp.AccessToken = m
		}
	}
	if resp.AccessToken == "" {
		// C#: accountInfo.AccessToken = "EAA" + Regex.Match(resstr, "EAA(.*?)\"").Groups[1].Value
		if m := reAccessToken.FindStringSubmatch(clean); len(m) > 1 {
			resp.AccessToken = "EAA" + m[1]
		}
	}

	// C#: accountInfo.Uid = Regex.Match(resstr, "c_user\",\"value\":\"(.*?)\"").Groups[1].Value
	if m := reCUser.FindStringSubmatch(clean); len(m) > 1 {
		resp.UID = m[1]
	}
	// Fallback nếu c_user pattern không match
	if resp.UID == "" {
		for _, re := range []*regexp.Regexp{reCreatedUID, reUIDGeneric} {
			if m := re.FindStringSubmatch(clean); len(m) > 1 {
				resp.UID = m[1]
				break
			}
		}
	}

	if resp.UID == "" {
		return resp, fmt.Errorf("no UID found in response")
	}

	// C#: xs = Regex.Match(resstr, "name\":\"xs\",\"value\":\"(.*?)\"")
	if m := reXS.FindStringSubmatch(clean); len(m) > 1 {
		resp.XS = m[1]
	}
	// C#: fr = Regex.Match(resstr, "name\":\"fr\",\"value\":\"(.*?)\"")
	if m := reFR.FindStringSubmatch(clean); len(m) > 1 {
		resp.FR = m[1]
	}
	// C#: datr = Regex.Match(resstr, "name\":\"datr\",\"value\":\"(.*?)\"")
	if m := reDATR.FindStringSubmatch(clean); len(m) > 1 {
		resp.DATR = m[1]
	}

	// C#: cookies.Add("c_user", uid); xs; locale; fr; datr
	// Cookie = string.Join(";", cookies.Where(x => !string.IsNullOrEmpty(x.Value)))
	var parts []string
	parts = append(parts, "c_user="+resp.UID)
	if resp.XS != "" {
		parts = append(parts, "xs="+resp.XS)
	}
	if locale != "" {
		parts = append(parts, "locale="+locale)
	}
	if resp.FR != "" {
		parts = append(parts, "fr="+resp.FR)
	}
	if resp.DATR != "" {
		parts = append(parts, "datr="+resp.DATR)
	}
	// C#: if (!accountInfo.Cookie.EndsWith(";")) accountInfo.Cookie += ";";
	resp.Cookie = strings.Join(parts, ";") + ";"

	return resp, nil
}

// parseXZeroEHResponse extract eligibility_hash từ batch response.
func parseXZeroEHResponse(body string) string {
	re := regexp.MustCompile(`eligibility_hash["\s:=\\]+([A-Za-z0-9_-]+)`)
	clean := strings.ReplaceAll(body, "\\", "")
	if m := re.FindStringSubmatch(clean); len(m) > 1 {
		return m[1]
	}
	return ""
}
