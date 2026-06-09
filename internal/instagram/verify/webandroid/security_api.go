// security_api.go — TutVer1 2FA via AccountsCenter Chrome Android API
// Mapping từ C#: FacebookSecurityWebAndroidAPI.TurnOnTwofactor
//
// Flow:
//  1. GET accountscenter.facebook.com/?__wblt=1 → parse tokens (DTSGInitData, LSD, hsi, spinr, spint)
//  2. POST generate TOTP key → may hit reauth (email OTP or password)
//  3. Extract TOTP secret from QR URL → compute code → POST confirm
package webandroid

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ─── Constants ────────────────────────────────────────────────────────────────

const (
	acctCenterGetURL = "https://accountscenter.facebook.com/?__wblt=1"
	acctCenterAPIURL = "https://accountscenter.facebook.com/api/graphql/"
	acctCenterOrigin = "https://accountscenter.facebook.com"
	acctCenter2faRef = "https://accountscenter.facebook.com/password_and_security/two_factor"

	// doc IDs — từ C# FacebookApiFormDataBuilder
	docIDGenKey     = "9837172312995248"  // Generate2FAKeyChromeAndroiddocid
	docIDSendCP     = "9889219791098314"  // SendCodeVerifyEmailChromeAndroiddocid
	docIDConfCP     = "9619343038102666"  // ConfirmCodeEmailTwostepVeriChromeAndroiddocid
	docIDConfTOTP   = "29164158613231327" // Confirm2FACodeChromeAndroiddocid
	docIDPassReauth = "9884242298280664"  // TwofactorPasswordReauthChromeAndroiddocid

	// friendly names — từ C# FacebookApiFormDataBuilder static fields
	fnGenKey     = "useFXSettingsTwoFactorGenerateTOTPKeyMutation"
	fnSendCP     = "useTwoStepVerificationSendCodeMutation"
	fnConfCP     = "useTwoFactorLoginValidateCodeMutation"
	fnConfTOTP   = "useFXSettingsTwoFactorEnableTOTPMutation"
	fnPassReauth = "FXPasswordReauthenticationMutation"
)

// ─── acctCenterState ─────────────────────────────────────────────────────────

// acctCenterState chứa form tokens extract từ accountscenter page.
// Mapping từ C#: FacebookRequestFormDataPropModel.BuildFromChromeAndroidAccountCenterPage
type acctCenterState struct {
	hsi     string
	dtsg    string
	jazoest string
	lsd     string
	spinr   string
	spint   string
}

// ─── Enable2FA ────────────────────────────────────────────────────────────────

// Enable2FA bật 2FA TOTP cho account qua Chrome Android AccountsCenter API.
//
// accountEmail: email thực của account (dùng để mask + nhận OTP khi reauth).
// emailOTPFn:   func(maskedEmail string, waitSec int) string — callback lấy OTP từ inbox;
//
//	trả về "" nếu timeout.
//
// Trả về (totpBase32Secret, error). Secret này cần lưu lại để tạo code đăng nhập sau.
func Enable2FA(
	ctx context.Context,
	proxyStr, cookie, uid, ua, password, accountEmail string,
	emailOTPFn func(maskedEmail string, waitSec int) string,
	notify func(string),
) (string, error) {
	client := createClient(proxyStr)
	defer client.CloseIdleConnections()
	ua = chromeUA(ua)

	notify("[2FA] Fetching account center page...")
	state, err := getAcctCenterState(ctx, client, cookie, ua)
	if err != nil {
		return "", fmt.Errorf("accountscenter GET: %w", err)
	}

	for attempt := 1; attempt <= 5; attempt++ {
		notify(fmt.Sprintf("[2FA] Generate key attempt %d...", attempt))

		resp, err := acctCenterPost(ctx, client, cookie, ua, uid, state,
			fnGenKey, docIDGenKey, buildGenKeyVars(uid))
		if err != nil {
			return "", fmt.Errorf("generate2FAKey: %w", err)
		}
		norm := wa2faUnescape(resp)

		switch {
		case strings.Contains(norm, `challenge_type":"reauth`):
			// ── Reauth via email OTP ──────────────────────────────────────
			encCtx := reExtract(norm, `"encrypted_context":"(.*?)"`)
			if encCtx == "" {
				return "", fmt.Errorf("reauth: encrypted_context not found")
			}
			masked := maskEmail(accountEmail)
			notify(fmt.Sprintf("[2FA] Email reauth — sending code to %s...", masked))

			// C#: up to 3 send-code attempts
			sentOK := false
			for try := 0; try < 3 && !sentOK; try++ {
				// Cancellable: Stop verify exit ngay thay vì chờ 1s × 3 retry.
				select {
				case <-time.After(time.Second):
				case <-ctx.Done():
					return "", ctx.Err()
				}
				r, e := acctCenterPost(ctx, client, cookie, ua, uid, state,
					fnSendCP, docIDSendCP, buildSendCPVars(masked, encCtx))
				if e != nil {
					notify(fmt.Sprintf("[2FA] Send code error (try %d): %v", try+1, e))
					continue
				}
				if strings.Contains(wa2faUnescape(r), `"is_success":true`) {
					sentOK = true
				}
			}
			if !sentOK {
				return "", fmt.Errorf("reauth: failed to send email code")
			}

			if emailOTPFn == nil {
				return "", fmt.Errorf("reauth: no emailOTPFn provided")
			}
			notify("[2FA] Waiting for reauth OTP code...")
			code := emailOTPFn(masked, 30)
			if code == "" {
				return "", fmt.Errorf("reauth: OTP timeout")
			}

			r, err := acctCenterPost(ctx, client, cookie, ua, uid, state,
				fnConfCP, docIDConfCP, buildConfirmCPVars(masked, encCtx, code))
			if err != nil {
				return "", fmt.Errorf("reauth confirm: %w", err)
			}
			if !strings.Contains(wa2faUnescape(r), `"is_code_valid":true`) {
				return "", fmt.Errorf("reauth: OTP code invalid")
			}
			notify("[2FA] Reauth email confirmed — retrying key generation...")
			continue

		case strings.Contains(norm, `challenge_type":"password`):
			// ── Reauth via password ───────────────────────────────────────
			if password == "" {
				return "", fmt.Errorf("password reauth required but password not provided")
			}
			notify("[2FA] Password reauth required...")
			ts := time.Now().Unix()
			// C#: WebUtility.UrlEncode(TwoFactorPasswordReauthVariables(uid, password))
			encodedVars := url.QueryEscape(buildPasswordReauthVars(uid, password, ts))
			r, err := acctCenterPost(ctx, client, cookie, ua, uid, state,
				fnPassReauth, docIDPassReauth, encodedVars)
			if err != nil {
				return "", fmt.Errorf("password reauth POST: %w", err)
			}
			if !strings.Contains(wa2faUnescape(r), `"is_reauth_successful":true`) {
				return "", fmt.Errorf("password reauth failed")
			}
			notify("[2FA] Password reauth OK — retrying key generation...")
			continue

		default:
			// ── Success path: extract TOTP secret from QR URL ─────────────
			// C#: Regex.Match(result_unescape, "https://www.facebook.com/qr/show/code/(.*?)\"")
			qrPath := reExtract(norm, `https://www\.facebook\.com/qr/show/code/(.*?)"`)
			if qrPath == "" {
				return "", fmt.Errorf("generate2FAKey: no QR URL in response (attempt %d)", attempt)
			}
			decoded, _ := url.QueryUnescape(qrPath)
			// C#: Regex.Match(_link_2fa, "secret=(.*?)&")
			secret := reExtract(decoded, `secret=(.*?)&`)
			if secret == "" {
				secret = reExtract(decoded, `secret=([^&"]+)`)
			}
			if secret == "" {
				return "", fmt.Errorf("cannot extract TOTP secret from: %s", decoded)
			}

			notify("[2FA] Got secret — computing TOTP code...")
			totpCode, err := wa2faTOTP(secret)
			if err != nil {
				return "", fmt.Errorf("generateTOTP: %w", err)
			}
			notify(fmt.Sprintf("[2FA] Confirming TOTP code: %s", totpCode))

			r, err := acctCenterPost(ctx, client, cookie, ua, uid, state,
				fnConfTOTP, docIDConfTOTP, buildConfirmTOTPVars(uid, totpCode))
			if err != nil {
				return "", fmt.Errorf("confirm TOTP POST: %w", err)
			}
			// C#: if (resultapi.Contains("success\":true"))
			if !strings.Contains(wa2faUnescape(r), `"success":true`) {
				return "", fmt.Errorf("confirm TOTP: unexpected response")
			}
			notify("[2FA] 2FA enabled successfully!")
			return secret, nil
		}
	}
	return "", fmt.Errorf("enable2FA: max retries exceeded")
}

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

// getAcctCenterState GETs accountscenter page và parse form tokens.
// Mapping từ C#: FacebookRequestFormDataPropModel.BuildFromChromeAndroidAccountCenterPage
func getAcctCenterState(ctx context.Context, client *http.Client, cookie, ua string) (*acctCenterState, error) {
	h := make(http.Header)
	h.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	h.Set("Upgrade-Insecure-Requests", "1")
	h.Set("Sec-Fetch-Site", "none")
	h.Set("Sec-Fetch-Mode", "navigate")
	h.Set("Sec-Fetch-User", "?1")
	h.Set("Sec-Fetch-Dest", "document")
	h.Set("Priority", "u=0, i")
	h.Set("Accept-Language", "en-US,en;q=0.9")
	// C#: customreferer=MfbSingleUrlPrefix="https://m.facebook.com/"
	h.Set("Referer", "https://m.facebook.com/")
	h.Set("User-Agent", ua)
	h.Set("Cookie", cookie)

	body, finalURL, err := doGet(ctx, client, acctCenterGetURL, h)
	if err != nil {
		return nil, err
	}
	if isLogoutURL(finalURL) {
		return nil, fmt.Errorf("accountscenter: checkpoint/logout at %s", finalURL)
	}
	if body == "" {
		return nil, fmt.Errorf("accountscenter: empty response")
	}

	// C#: Task.Delay(TimeSpan.FromSeconds(1)).Wait()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(time.Second):
	}

	// C# regex patterns:
	//   hsi     = hsi\":\"(\d+)\","
	//   dtsg    = DTSGInitData\"(.*?)\"token\":\"(.*?)\"   → group 2
	//   jazoest = jazoest=(\d+)\"
	//   lsd     = LSD\"(.*?)token\":\"(.*?)\"             → group 2
	//   spinr   = __spin_r\":(.*?),
	//   spint   = __spin_t\":(.*?),
	state := &acctCenterState{
		hsi:     reExtract(body, `hsi":"(\d+)"`),
		dtsg:    reExtractG2(body, `DTSGInitData"[\s\S]*?"token":"([\s\S]*?)"`),
		jazoest: reExtract(body, `jazoest=(\d+)"`),
		lsd:     reExtractG2(body, `LSD"[\s\S]*?token":"([\s\S]*?)"`),
		spinr:   reExtract(body, `__spin_r":([\s\S]*?),`),
		spint:   reExtract(body, `__spin_t":([\s\S]*?),`),
	}
	if state.dtsg == "" {
		return nil, fmt.Errorf("accountscenter: cannot extract fb_dtsg (checkpoint?)")
	}
	return state, nil
}

// acctCenterPost POSTs tới accountscenter/api/graphql với Chrome Android headers.
// Mapping từ C#: PerpectChromeAndroidPostHeaders + BaseGraphqlChromeAndroidAccountCenterFormData
func acctCenterPost(
	ctx context.Context,
	client *http.Client,
	cookie, ua, uid string,
	state *acctCenterState,
	friendlyName, docID, variables string,
) (string, error) {
	body := buildAcctCenterBody(uid, state, friendlyName, docID, variables)
	h := make(http.Header)
	// C#: PerpectChromeAndroidPostHeaders(locale, WWWACT_MainUrl, WWWACT_Enable2faUrl, friendlyName, lsd)
	h.Set("Accept", "*/*")
	h.Set("Accept-Language", "en-US,en;q=0.9")
	h.Set("Origin", acctCenterOrigin)
	h.Set("Referer", acctCenter2faRef)
	h.Set("X-FB-Friendly-Name", friendlyName)
	h.Set("X-Fb-Lsd", state.lsd)
	h.Set("Sec-Fetch-Site", "same-origin")
	h.Set("Sec-Fetch-Mode", "cors")
	h.Set("Sec-Fetch-Dest", "empty")
	h.Set("Priority", "u=0, i")
	h.Set("User-Agent", ua)
	h.Set("Cookie", cookie)
	return doPost(ctx, client, acctCenterAPIURL, body, h)
}

// buildAcctCenterBody — BaseGraphqlChromeAndroidAccountCenterFormData equivalent.
// variables phải là string đã URL-encoded (giống C# variable builder methods).
// C# dùng string.Join("&", map{key}={value}) — KHÔNG encode thêm giá trị.
func buildAcctCenterBody(uid string, s *acctCenterState, friendlyName, docID, variables string) string {
	fields := []string{
		"av=" + uid,
		"__user=" + uid,
		"__a=1",
		"__req=r",
		"__hs=20219.HYP%3Aaccounts_center_pkg.2.1...0",
		"dpr=1",
		"__ccg=EXCELLENT",
		"__rev=" + s.spinr,
		"__s=",
		"__hsi=" + s.hsi,
		"__dyn=",
		"__csr=",
		"__hsdp=",
		"__hblpi=",
		"__hblpn=",
		"__comet_req=5",
		"fb_dtsg=" + s.dtsg,
		"jazoest=" + s.jazoest,
		"lsd=" + s.lsd,
		"__spin_r=" + s.spinr,
		"__spin_b=trunk",
		"__spin_t=" + s.spint,
		"fb_api_caller_class=RelayModern",
		"fb_api_req_friendly_name=" + url.QueryEscape(friendlyName),
		"variables=" + variables, // already URL-encoded by callers
		"server_timestamps=true",
		"doc_id=" + docID,
	}
	return strings.Join(fields, "&")
}

// ─── Variable builders ────────────────────────────────────────────────────────
// Tất cả hàm bên dưới trả về URL-encoded JSON, giống C# variable builder methods.

// buildGenKeyVars — Generate2FAKeyChromeAndroidVariables
func buildGenKeyVars(uid string) string {
	raw := fmt.Sprintf(`{"input":{"client_mutation_id":"%s","actor_id":"%s","account_id":"%s","account_type":"FACEBOOK","device_id":"device_id_fetch_datr","fdid":"device_id_fetch_datr"}}`,
		uuid.New().String(), uid, uid)
	return url.QueryEscape(raw)
}

// buildSendCPVars — SendCodeVerifyEmailChromeAndroidVariables
func buildSendCPVars(maskedCP, encCtx string) string {
	raw := fmt.Sprintf(`{"encryptedContext":"%s","challenge":"EMAIL","maskedContactPoint":"%s"}`,
		encCtx, maskedCP)
	return url.QueryEscape(raw)
}

// buildConfirmCPVars — SubmitCodeEmailTwostepVerificationChromeAndroidVariables
func buildConfirmCPVars(maskedCP, encCtx, code string) string {
	raw := fmt.Sprintf(`{"code":{"sensitive_string_value":"%s"},"method":"EMAIL","flow":"SECURED_ACTION","encryptedContext":"%s","maskedContactPoint":"%s","next_uri":null}`,
		code, encCtx, maskedCP)
	return url.QueryEscape(raw)
}

// buildConfirmTOTPVars — Submit2FACodeChromeAndroidVariables
func buildConfirmTOTPVars(uid, code string) string {
	raw := fmt.Sprintf(`{"input":{"client_mutation_id":"%s","actor_id":"%s","account_id":"%s","account_type":"FACEBOOK","verification_code":"%s","device_id":"device_id_fetch_datr","fdid":"device_id_fetch_datr"}}`,
		uuid.New().String(), uid, uid, code)
	return url.QueryEscape(raw)
}

// buildPasswordReauthVars — TwoFactorPasswordReauthVariables (raw JSON, chưa URL-encode).
// Caller phải url.QueryEscape kết quả trước khi truyền vào acctCenterPost.
func buildPasswordReauthVars(uid, password string, timestamp int64) string {
	return fmt.Sprintf(`{"input":{"account_id":%s,"account_type":"FACEBOOK","category_name":null,"password":{"sensitive_string_value":"#PWD_BROWSER:0:%d:%s"},"actor_id":"%s","client_mutation_id":"1"}}`,
		uid, timestamp, password, uid)
}

// ─── Regex helpers ────────────────────────────────────────────────────────────

// reExtractG2 tìm group[1] của pattern có dạng "prefix.*?capture".
// Dùng cho DTSGInitData và LSD patterns có 1 non-capturing prefix.
// Compile mỗi lần vì pattern khác nhau — dùng trong init path, không hot.
func reExtractG2(src, pattern string) string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}
	m := re.FindStringSubmatch(src)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// ─── Utility ─────────────────────────────────────────────────────────────────

// wa2faUnescape normalise JSON string escaping trong FB API response.
// Tương đương C#: Regex.Unescape(resultapi) — xử lý trường hợp phổ biến nhất.
func wa2faUnescape(s string) string {
	s = strings.ReplaceAll(s, `\/`, `/`)
	s = strings.ReplaceAll(s, `\"`, `"`)
	return s
}

// maskEmail mask email address, giống C# StringUtils.MaskEmail(email).ToLower().
// "abc@gmail.com" → "a*c@gmail.com"   (community domain → domain unchanged)
// "ab@example.org" → "ab@e******.org" (other domain → mask after first char)
func maskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return strings.ToLower(email)
	}
	local, domain := parts[0], parts[1]
	maskedLocal := maskLocalPart(local)

	// C#: CommunityMailDomains = { "hotmail.", "gmail.", "outlook." }
	for _, d := range []string{"hotmail.", "gmail.", "outlook."} {
		if strings.HasPrefix(domain, d) {
			return strings.ToLower(maskedLocal + "@" + domain)
		}
	}

	// Non-community domain: keep first char, mask middle, keep .suffix
	dotIdx := strings.Index(domain, ".")
	if dotIdx <= 1 {
		return strings.ToLower(maskedLocal + "@" + domain)
	}
	maskedDomain := domain[:1] + strings.Repeat("*", dotIdx-1) + domain[dotIdx:]
	return strings.ToLower(maskedLocal + "@" + maskedDomain)
}

func maskLocalPart(local string) string {
	if len(local) <= 2 {
		return local
	}
	out := []byte(local)
	for i := 1; i < len(out)-1; i++ {
		out[i] = '*'
	}
	return string(out)
}

// wa2faTOTP generates 6-digit TOTP code (RFC 6238, HMAC-SHA1, 30s window).
// secret: base32-encoded TOTP key (case-insensitive, padding optional).
func wa2faTOTP(secret string) (string, error) {
	secret = strings.ToUpper(strings.TrimRight(strings.TrimSpace(secret), "="))
	if pad := len(secret) % 8; pad != 0 {
		secret += strings.Repeat("=", 8-pad)
	}
	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("base32 decode: %w", err)
	}

	counter := uint64(time.Now().Unix() / 30)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf[:])
	h := mac.Sum(nil)

	offset := h[len(h)-1] & 0x0f
	val := int64(binary.BigEndian.Uint32(h[offset:offset+4]) & 0x7fffffff)
	mod := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil).Int64()
	return fmt.Sprintf("%06d", val%mod), nil
}
