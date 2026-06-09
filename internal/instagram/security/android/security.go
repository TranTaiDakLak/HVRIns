// security.go — Android SecurityManager: Enable2FA, HandleCheckpoint, ChangePassword
// Mapping từ C#: FacebookSecurityFeatureAPIAndroid
//
// Enable2FA flow (TurnOnTwofactor):
//  1. SelectMethod  → FbBloksAppRootQuery-com.bloks.www.fx.settings.security.two_factor.select_method
//  2. GenerateKey   → FbBloksActionRootQuery-com.bloks.www.fx.settings.security.two_factor.totp.generate_key
//     (nếu reauth cần) → FbBloksActionRootQuery-com.bloks.www.fx.reauth.password.async
//  3. SubmitTOTP    → FbBloksActionRootQuery-com.bloks.www.fx.settings.security.two_factor.totp.enable
//     (nếu email 2FA cần) → send_code.async + verify_code.async + re-submit TOTP
//
// Endpoint: https://graph.facebook.com/graphql
package android

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	mrand "math/rand"
	"net/url"
	"regexp"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/google/uuid"

	"HVRIns/internal/instagram"
	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

// ─── Constants ────────────────────────────────────────────────────────────────

const (
	mobileGraphqlURL = "https://graph.facebook.com/graphql"

	// bloks versioning IDs
	bloksVer2FA     = "a6e1cbd957de452adecbf0ed72ed377d98023bff8ead23a8117997f39ad9ad86"
	bloksVerContact = "58ad14305693256c5ef8d22c3ba6f00eb38e8e2526063676056aea65395dcc01"

	// doc IDs — C# constants
	docIDSelectMethod = "1053734614299266934128029406"     // TwofactorSelectMethoddocid
	docIDOpsV3        = "1199408042526631289603660492"     // ChangeAndConfirmContactpointMobiledocid_v3 + ConfirmCodeSubEmailMobiledocid

	// friendly names
	fnSelectMethod  = "FbBloksAppRootQuery-com.bloks.www.fx.settings.security.two_factor.select_method"
	fnGenKey        = "FbBloksActionRootQuery-com.bloks.www.fx.settings.security.two_factor.totp.generate_key"
	fnSubmitTOTP    = "FbBloksActionRootQuery-com.bloks.www.fx.settings.security.two_factor.totp.enable"
	fnReauthPwd     = "FbBloksActionRootQuery-com.bloks.www.fx.reauth.password.async"
	fnSendCodeVrf   = "FbBloksActionRootQuery-com.bloks.www.two_step_verification.send_code.async"
	fnConfirmTwoStp = "FbBloksActionRootQuery-com.bloks.www.two_step_verification.verify_code.async"

	// X-Meta-Zca fixed value — C#: _defaultMetaZcaHeaderValue
	metaZcaValue = "eyJhbmRyb2lkIjp7ImFrYSI6eyJkYXRhVG9TaWduIjoiIiwiZXJyb3JzIjpbIktFWVNUT1JFX0RJU0FCTEVEX0JZX0NPTkZJRyJdfSwiZ3BpYSI6eyJ0b2tlbiI6IiIsImVycm9ycyI6WyJQTEFZX0lOVEVHUklUWV9ESVNBQkxFRF9CWV9DT05GSUciXX19fQ"
)

// ─── SecurityManager ──────────────────────────────────────────────────────────

// SecurityManager implements instagram.SecurityManager for the Android platform.
type SecurityManager struct {
	// EmailOTPFn: callback để lấy OTP từ email khi FB yêu cầu xác minh email.
	// maskedEmail: email đã mask (v**@gmail.com), waitSec: số giây tối đa chờ.
	// Trả về OTP code string, hoặc "" nếu không lấy được.
	// Nếu nil → bỏ qua bước xác minh email (Enable2FA sẽ fail nếu FB bắt buộc).
	EmailOTPFn func(maskedEmail string, waitSec int) string
}

func (s *SecurityManager) Enable2FA(ctx context.Context, session *instagram.Session) (*instagram.TwoFAResult, error) {
	if session.Token == "" {
		return nil, fmt.Errorf("Enable2FA requires access token (session.Token empty)")
	}

	machineID := session.Datr
	if len(machineID) < 3 {
		machineID = ""
	}

	api := &androidSecAPI{
		client:    nil, // khởi tạo bên dưới
		token:     session.Token,
		uid:       session.UID,
		deviceID:  session.DeviceID,
		machineID: machineID,
		password:  session.Password,
		locale:    "en_US",
		ua:        session.UserAgent,
	}
	if err := api.init(session.Proxy); err != nil {
		return nil, fmt.Errorf("Enable2FA: create HTTP client: %w", err)
	}
	// Task 5: tls_client per Enable2FA call → close idle conn khi func return
	// để fhttp.Transport free TCP/TLS buffer. Trước đây để GC dọn (non-deterministic).
	defer api.client.CloseIdleConnections()

	return api.turnOnTwoFactor(ctx, s.EmailOTPFn)
}

func (s *SecurityManager) HandleCheckpoint(_ context.Context, session *instagram.Session) error {
	return nil // TODO: implement
}

func (s *SecurityManager) ChangePassword(_ context.Context, session *instagram.Session, newPassword string) error {
	return nil // TODO: implement
}

func init() {
	instagram.RegisterPlatformSecurityManager(instagram.PlatformAndroid, func() instagram.SecurityManager {
		return &SecurityManager{}
	})
}

// ─── androidSecAPI ────────────────────────────────────────────────────────────

type androidSecAPI struct {
	client    tls_client.HttpClient
	token     string
	uid       string
	deviceID  string
	machineID string // datr value
	password  string
	locale    string
	ua        string
}

func (a *androidSecAPI) init(proxyStr string) error {
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
	}
	if proxyStr != "" {
		if pURL := proxy.FormatProxyURL(proxyStr); pURL != "" {
			opts = append(opts, tls_client.WithProxyUrl(pURL))
		}
	}
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return err
	}
	a.client = c
	return nil
}

// ─── TurnOnTwofactor ──────────────────────────────────────────────────────────

// turnOnTwoFactor — full flow mapping từ C#: TurnOnTwofactor()
func (a *androidSecAPI) turnOnTwoFactor(ctx context.Context, emailOTPFn func(string, int) string) (*instagram.TwoFAResult, error) {
	const maxReauthRetry = 3

	for attempt := 0; attempt < maxReauthRetry; attempt++ {
		// === Step 1: SelectMethod (click "Enable 2FA") ===
		selBody := a.buildBody(fnSelectMethod, docIDSelectMethod, a.buildSelectMethodVars())
		selHeaders := a.buildHeaders(fnSelectMethod, "{\"schema_version\":\"v3\",\"inprogress_qpls\":[{\"marker_id\":25952257,\"annotations\":{\"current_endpoint\":\"com.bloks.www.fxcal.settings.identity_selection:com.bloks.www.fxcal.settings.identity_selection\"}}],\"snapshot_attributes\":{}}")
		a.doPostIgnore(ctx, selBody, selHeaders)
		sleepRandom(3, 6)

		// === Step 2: Generate TOTP Key ===
		genBody := a.buildBody(fnGenKey, docIDOpsV3, a.buildGenKeyVars())
		genHeaders := a.buildHeaders(fnGenKey, "{\"schema_version\":\"v3\",\"inprogress_qpls\":[{\"marker_id\":25952257,\"annotations\":{\"current_endpoint\":\"com.bloks.www.fx.settings.security.two_factor.select_method:com.bloks.www.fx.settings.security.two_factor.select_method\"}}],\"snapshot_attributes\":{}}")
		genResp, err := a.doPost(ctx, genBody, genHeaders)
		if err != nil {
			return nil, fmt.Errorf("generate key request: %w", err)
		}
		if isCheckpoint(genResp) {
			return nil, fmt.Errorf("checkpoint detected at generate key step")
		}

		sleepRandom(3, 6)

		// Kiểm tra cần reauth password không
		needReauth := strings.Contains(genResp, "BLOKS_SECURED_ACTION_REAUTH_PASSWORD_ENTRY") ||
			strings.Contains(genResp, "password_reauth_viewed") ||
			strings.Contains(genResp, "settings.security.secured_action.reauth_async")
		unescaped := strings.ReplaceAll(genResp, `\\\`, "")
		if !needReauth && (!strings.Contains(unescaped, "qr_code_uri") && !strings.Contains(genResp, "data=otpauth")) {
			needReauth = true
		}

		if needReauth {
			// === Reauth Password ===
			sleepRandom(3, 6)
			reauthBody := a.buildBody(fnReauthPwd, docIDOpsV3, a.buildReauthPasswordVars())
			reauthHeaders := a.buildHeaders(fnReauthPwd, "")
			reauthResp, err := a.doPost(ctx, reauthBody, reauthHeaders)
			if err != nil {
				return nil, fmt.Errorf("reauth password request: %w", err)
			}
			if isCheckpoint(reauthResp) {
				return nil, fmt.Errorf("checkpoint at reauth password")
			}
			if !strings.Contains(reauthResp, "BLOKS_FX_REAUTH_PASSWORD_ENTRY:success") {
				return nil, fmt.Errorf("reauth password failed (attempt %d)", attempt+1)
			}
			sleepRandom(3, 6)
			continue // goto GenKeyPoint
		}

		// === Extract TOTP secret from QR URL ===
		secret, err := extractTOTPSecret(unescaped)
		if err != nil {
			return nil, fmt.Errorf("extract TOTP secret: %w", err)
		}

		// === Step 3: Submit TOTP Code ===
		totpCode, err := generateTOTP(secret)
		if err != nil {
			return nil, fmt.Errorf("generate TOTP: %w", err)
		}

		submitBody := a.buildBody(fnSubmitTOTP, docIDOpsV3, a.buildSubmitTOTPVars(totpCode))
		submitHeaders := a.buildHeaders(fnSubmitTOTP, "{\"schema_version\":\"v3\",\"inprogress_qpls\":[{\"marker_id\":25952257,\"annotations\":{\"current_endpoint\":\"com.bloks.www.fx.settings.security.two_factor.totp.code:com.bloks.www.fx.settings.security.two_factor.totp.code\"}}],\"snapshot_attributes\":{}}")
		submitResp, err := a.doPost(ctx, submitBody, submitHeaders)
		if err != nil {
			return nil, fmt.Errorf("submit TOTP request: %w", err)
		}
		if isCheckpoint(submitResp) {
			return nil, fmt.Errorf("checkpoint at submit TOTP")
		}

		// Nếu thành công ngay (không cần email OTP)
		if strings.Contains(submitResp, "FX_TWO_FACTOR_STATUS:is_enabled") {
			return &instagram.TwoFAResult{Success: true, Secret: secret}, nil
		}

		// Kiểm tra xem có cần xác minh email không
		twoStepCtx := extractTwoStepContext(submitResp)
		if twoStepCtx == "" {
			return nil, fmt.Errorf("Enable2FA: unexpected response (no success, no two_step_context): %s", submitResp[:min2FA(len(submitResp), 400)])
		}

		// === Email OTP verification ===
		result, err := a.handleEmailOTPFor2FA(ctx, twoStepCtx, secret, emailOTPFn)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	return nil, fmt.Errorf("Enable2FA: max reauth retries exceeded")
}

// handleEmailOTPFor2FA xử lý bước xác minh email khi Enable2FA bị yêu cầu
// Mapping từ C# TurnOnTwofactor: phần sau khi có two_step_verification_context
func (a *androidSecAPI) handleEmailOTPFor2FA(ctx context.Context, twoStepCtx, secret string, emailOTPFn func(string, int) string) (*instagram.TwoFAResult, error) {
	if emailOTPFn == nil {
		return nil, fmt.Errorf("Enable2FA requires email OTP (two_step_context found) but EmailOTPFn is nil")
	}

	maskedEmail := "" // FB sẽ gửi OTP về email đã đăng ký, ta dùng masked placeholder

	for sendAttempt := 0; sendAttempt <= 3; sendAttempt++ {
		sleepRandom(3, 6)

		// Gửi OTP về email
		sendBody := a.buildBody(fnSendCodeVrf, docIDOpsV3, a.buildSendCodeVars(maskedEmail, twoStepCtx))
		sendHeaders := a.buildHeaders(fnSendCodeVrf, "")
		sendResp, err := a.doPost(ctx, sendBody, sendHeaders)
		if err != nil {
			sendAttempt++
			continue
		}
		if isCheckpoint(sendResp) {
			return nil, fmt.Errorf("checkpoint at send 2FA code email")
		}
		if !strings.Contains(sendResp, "com.bloks.www.two_step_verification.send_code.async") {
			sendAttempt++
			continue
		}

		// Đợi OTP từ email (tối đa 20 giây)
		code := emailOTPFn(maskedEmail, 20)
		if code == "" {
			sendAttempt++
			continue
		}

		// Xác nhận OTP
		cfmBody := a.buildBody(fnConfirmTwoStp, docIDOpsV3, a.buildConfirmTwoStepVars(maskedEmail, twoStepCtx, code))
		cfmHeaders := a.buildHeaders(fnConfirmTwoStp, "")
		cfmResp, err := a.doPost(ctx, cfmBody, cfmHeaders)
		if err != nil || isCheckpoint(cfmResp) {
			return nil, fmt.Errorf("confirm two-step verification failed")
		}
		if !strings.Contains(cfmResp, "SESSION_STORE_BLOKS_SECURED_ACTION_REAUTH:success") &&
			!strings.Contains(cfmResp, "BLOKS_SECURED_ACTION_REAUTH_success") {
			return nil, fmt.Errorf("two-step confirm: unexpected response")
		}

		// Re-submit TOTP sau khi xác minh email
		sleepRandom(3, 6)
		totpCode, err := generateTOTP(secret)
		if err != nil {
			return nil, fmt.Errorf("re-generate TOTP: %w", err)
		}
		resubBody := a.buildBody(fnSubmitTOTP, docIDOpsV3, a.buildSubmitTOTPVars(totpCode))
		resubHeaders := a.buildHeaders(fnSubmitTOTP, "{\"schema_version\":\"v3\",\"inprogress_qpls\":[{\"marker_id\":25952257,\"annotations\":{\"current_endpoint\":\"com.bloks.www.fx.settings.security.two_factor.totp.code:com.bloks.www.fx.settings.security.two_factor.totp.code\"}}],\"snapshot_attributes\":{}}")
		resubResp, err := a.doPost(ctx, resubBody, resubHeaders)
		if err != nil || isCheckpoint(resubResp) {
			return nil, fmt.Errorf("re-submit TOTP failed")
		}
		if strings.Contains(resubResp, "FX_TWO_FACTOR_STATUS:is_enabled") {
			return &instagram.TwoFAResult{Success: true, Secret: secret}, nil
		}
		return nil, fmt.Errorf("Enable2FA: re-submit TOTP did not confirm: %s", resubResp[:min2FA(len(resubResp), 400)])
	}

	return nil, fmt.Errorf("Enable2FA: email OTP exhausted (3 attempts)")
}

// ─── Variables builders ───────────────────────────────────────────────────────

// buildSelectMethodVars — TwofactorSelectMethodMobileVariables(uid, machineID)
func (a *androidSecAPI) buildSelectMethodVars() string {
	// SelectMethod dùng FbBloksAppRootQuery — format Bloks Root Query (khác Action query)
	inner := fmt.Sprintf(`{params:{"client_input_params":{"machine_id":"%s"},"server_params":{"account_type":0,"account_id":%s,"INTERNAL_INFRA_screen_id":"0","should_show_done_button":0}},}`,
		a.machineID, a.uid)
	l1 := fmt.Sprintf(`{"params":"%s","bloks_versioning_id":"%s","app_id":"com.bloks.www.fx.settings.security.two_factor.select_method"}`,
		jsonEscape2FA(inner), bloksVer2FA)
	ntCtx := fmt.Sprintf(`{"using_white_navbar":true,"styles_id":"ca49d9aab0f1291c1131e3f816210b58","pixel_ratio":3,"is_push_on":true,"debug_tooling_metadata_token":null,"is_flipper_enabled":false,"theme_params":[{"value":["BLUEPRINT_TEST_GUTTER","BLUEPRINT_TEST_ROUNDED_CORNERS_NO_GUTTERS"],"design_system_name":"FDS"}],"bloks_version":"%s"}`, bloksVer2FA)
	return fmt.Sprintf(`{"params":%s,"scale":"3","use_native_entrypoint_for_stars_on_reels":true,"nt_context":%s}`, l1, ntCtx)
}

// buildGenKeyVars — Generate2FAKeyMobileVariables(deviceID, uid, machineID)
// Note: C# dùng device_id cho machine_id trong client_input_params, không phải datr
func (a *androidSecAPI) buildGenKeyVars() string {
	qpl := qplID2FA()
	inner := fmt.Sprintf(`{"client_input_params":{"family_device_id":"%s","device_id":"%s","machine_id":"%s"},"server_params":{"requested_screen_component_type":null,"account_type":0,"machine_id":"%s","INTERNAL__latency_qpl_marker_id":36707139,"INTERNAL__latency_qpl_instance_id":%d,"account_id":%s}}`,
		a.deviceID, a.deviceID, a.deviceID,
		a.deviceID, qpl, a.uid)
	return build2FAActionParams(inner, "com.bloks.www.fx.settings.security.two_factor.totp.generate_key", bloksVer2FA)
}

// buildSubmitTOTPVars — Submit2FACodeMobileVariables(deviceID, uid, code, machineID)
func (a *androidSecAPI) buildSubmitTOTPVars(code string) string {
	qpl := qplID2FA()
	inner := fmt.Sprintf(`{"client_input_params":{"family_device_id":"%s","device_id":"%s","machine_id":"%s","verification_code":"%s"},"server_params":{"account_type":0,"INTERNAL__latency_qpl_marker_id":36707139,"INTERNAL__latency_qpl_instance_id":%d,"account_id":%s}}`,
		a.deviceID, a.deviceID, a.machineID, code, qpl, a.uid)
	return build2FAActionParams(inner, "com.bloks.www.fx.settings.security.two_factor.totp.enable", bloksVer2FA)
}

// buildReauthPasswordVars — ReauthPasswordDeactiveAccountMobileVariables(uid, password, deviceID, machineID)
func (a *androidSecAPI) buildReauthPasswordVars() string {
	ts := time.Now().Unix()
	encPwd := fmt.Sprintf("#PWD_FB4A:0:%d:%s", ts, a.password)
	qpl := qplID2FA()
	inner := fmt.Sprintf(`{"client_input_params":{"password":"%s","machine_id":"%s"},"server_params":{"account_type":0,"INTERNAL__latency_qpl_marker_id":36707139,"account_id":"%s","category_name":"SecuredActionAccountDnDCategory","requested_screen_component_type":null,"machine_id":null,"INTERNAL__latency_qpl_instance_id":%d,"node_identifier":"deletion_and_deactivation","request_data":"{}","on_success_override":1}}`,
		encPwd, a.machineID, a.uid, qpl)
	return build2FAActionParams(inner, "com.bloks.www.fx.settings.security.secured_action.reauth_async", bloksVer2FA)
}

// buildSendCodeVars — SendCodeVerifyEmailMobileVariables(deviceID, maskedCP, twoStepCtx, machineID)
func (a *androidSecAPI) buildSendCodeVars(maskedCP, twoStepCtx string) string {
	qpl := qplID2FA()
	inner := fmt.Sprintf(`{"client_input_params":{},"server_params":{"INTERNAL__latency_qpl_marker_id":36707139,"block_store_machine_id":null,"device_id":"%s","cloud_trust_token":null,"masked_cp":"%s","challenge":"email","machine_id":"%s","INTERNAL__latency_qpl_instance_id":%d,"two_step_verification_context":"%s","flow_source":"secured_action"}}`,
		a.deviceID, maskedCP, a.machineID, qpl, twoStepCtx)
	return build2FAActionParams(inner, "com.bloks.www.two_step_verification.send_code.async", bloksVerContact)
}

// buildConfirmTwoStepVars — SubmitCodeEmailTwostepVerificationMobileVariables(deviceID, maskedCP, twoStepCtx, code, machineID)
func (a *androidSecAPI) buildConfirmTwoStepVars(maskedCP, twoStepCtx, code string) string {
	qpl := qplID2FA()
	inner := fmt.Sprintf(`{"client_input_params":{"auth_secure_device_id":"","block_store_machine_id":null,"code":"%s","should_trust_device":0,"family_device_id":"%s","device_id":"%s","cloud_trust_token":null,"machine_id":"%s"},"server_params":{"INTERNAL__latency_qpl_marker_id":36707139,"block_store_machine_id":null,"device_id":"%s","cloud_trust_token":null,"masked_cp":"%s","challenge":"email","machine_id":"%s","INTERNAL__latency_qpl_instance_id":%d,"two_step_verification_context":"%s","flow_source":"secured_action"}}`,
		code, a.deviceID, a.deviceID, a.machineID,
		a.deviceID, maskedCP, a.machineID, qpl, twoStepCtx)
	return build2FAActionParams(inner, "com.bloks.www.two_step_verification.verify_code.async", bloksVerContact)
}

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

func (a *androidSecAPI) buildHeaders(friendlyName, qplFlowsJSON string) [][2]string {
	analyticsTag := `{"network_tags":{"product":"350685531728","purpose":"fetch","request_category":"graphql","retry_attempt":"0"},"application_tags":"graphservice"}`
	h := [][2]string{
		{"Authorization", "OAuth " + a.token},
		{"X-Fb-Friendly-Name", friendlyName},
		{"X-Fb-Connection-Type", "WIFI"},
		{"X-Fb-Sim-Hni", "45204"},
		{"X-Fb-Net-Hni", "45204"},
		{"X-Zero-Eh", ""},
		{"X-Graphql-Client-Library", "graphservice"},
		{"X-Tigon-Is-Retry", "False"},
		{"X-Fb-Privacy-Context", "3643298472347298"},
		{"X-Graphql-Request-Purpose", "fetch"},
		{"X-Fb-Http-Engine", "Tigon/Liger"},
		{"X-Fb-Client-Ip", "True"},
		{"X-Fb-Server-Cluster", "True"},
		{"X-Fb-Request-Analytics-Tags", analyticsTag},
		{"App-Scope-Id-Header", a.deviceID},
		{"X-Zero-F-Device-Id", a.deviceID},
		{"X-Meta-Zca", metaZcaValue},
		{"User-Agent", a.ua},
		{"Content-Type", "application/x-www-form-urlencoded"},
		{"Content-Encoding", "gzip"},
	}
	if qplFlowsJSON != "" {
		h = append(h, [2]string{"X-Fb-Qpl-Active-Flows-Json", qplFlowsJSON})
	}
	return h
}

// buildBody — BaseAndroidGraphqlFormData()
func (a *androidSecAPI) buildBody(friendlyName, docID, variables string) string {
	v := url.Values{}
	v.Set("method", "post")
	v.Set("pretty", "false")
	v.Set("format", "json")
	v.Set("server_timestamps", "true")
	v.Set("locale", a.locale)
	v.Set("purpose", "fetch")
	v.Set("fb_api_req_friendly_name", friendlyName)
	v.Set("fb_api_caller_class", "graphservice")
	v.Set("client_doc_id", docID)
	v.Set("fb_api_client_context", `{"is_background":false}`)
	v.Set("variables", variables)
	v.Set("fb_api_analytics_tags", `["GraphServices"]`)
	v.Set("client_trace_id", uuid.New().String())
	return v.Encode()
}

func (a *androidSecAPI) doPost(ctx context.Context, body string, headers [][2]string) (string, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(body)); err != nil {
		return "", fmt.Errorf("gzip: %w", err)
	}
	if err := gz.Close(); err != nil {
		return "", fmt.Errorf("gzip close: %w", err)
	}

	req, err := fhttp.NewRequestWithContext(ctx, "POST", mobileGraphqlURL, &buf)
	if err != nil {
		return "", err
	}
	order := make([]string, 0, len(headers))
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
		order = append(order, kv[0])
	}
	req.Header[fhttp.HeaderOrderKey] = order

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP: %w", err)
	}
	defer resp.Body.Close()
	data, err := httpx.ReadBody(resp.Body, 512*1024)
	return string(data), err
}

func (a *androidSecAPI) doPostIgnore(ctx context.Context, body string, headers [][2]string) {
	_, _ = a.doPost(ctx, body, headers)
}

// ─── Bloks variables structure helpers ───────────────────────────────────────

// build2FAActionParams bọc inner JSON vào triple-nested structure (FbBloksActionRootQuery format)
// Outer: {"params":{...},"scale":"3","use_native_entrypoint_for_stars_on_reels":true,"nt_context":{...}}
func build2FAActionParams(innerJSON, appID, bloksVer string) string {
	// Level 3 (innermost) = innerJSON
	// Level 2: {"params": "<json_escaped_L3>"}
	l2 := fmt.Sprintf(`{"params":"%s"}`, jsonEscape2FA(innerJSON))
	// Level 1: {"params": "<json_escaped_L2>", "bloks_versioning_id": "...", "app_id": "..."}
	l1 := fmt.Sprintf(`{"params":"%s","bloks_versioning_id":"%s","app_id":"%s"}`,
		jsonEscape2FA(l2), bloksVer, appID)
	ntCtx := fmt.Sprintf(`{"using_white_navbar":true,"styles_id":"ca49d9aab0f1291c1131e3f816210b58","pixel_ratio":3,"is_push_on":true,"debug_tooling_metadata_token":null,"is_flipper_enabled":false,"theme_params":[{"value":["BLUEPRINT_TEST_GUTTER","BLUEPRINT_TEST_ROUNDED_CORNERS_NO_GUTTERS"],"design_system_name":"FDS"}],"bloks_version":"%s"}`, bloksVer)
	return fmt.Sprintf(`{"params":%s,"scale":"3","use_native_entrypoint_for_stars_on_reels":true,"nt_context":%s}`, l1, ntCtx)
}

// ─── Parsing helpers ──────────────────────────────────────────────────────────

// extractTOTPSecret lấy base32 secret từ QR URL trong response
// C#: Regex.Match("https://www.facebook.com/qr/show/code/(.*?)\"")
//     → URL decode → Regex.Match("secret=(.*?)&")
func extractTOTPSecret(resp string) (string, error) {
	re1 := regexp.MustCompile(`https://www\.facebook\.com/qr/show/code/(.*?)"`)
	m := re1.FindStringSubmatch(resp)
	if len(m) < 2 || m[1] == "" {
		return "", fmt.Errorf("QR URL not found in response")
	}
	qrPath, _ := url.QueryUnescape(m[1])
	re2 := regexp.MustCompile(`secret=([^&"]+)`)
	m2 := re2.FindStringSubmatch(qrPath)
	if len(m2) < 2 || m2[1] == "" {
		return "", fmt.Errorf("secret not found in QR path: %s", qrPath)
	}
	return m2[1], nil
}

// extractTwoStepContext lấy two_step_verification_context từ response
// C#: Regex "INTERNAL_INFRA_screen_id\"(.*?)\"(.*?)\", \"secured_action\","
func extractTwoStepContext(resp string) string {
	cleaned := strings.ReplaceAll(resp, `\\\`, "")
	re := regexp.MustCompile(`INTERNAL_INFRA_screen_id".*?"(.*?)", "secured_action",`)
	m := re.FindStringSubmatch(cleaned)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// isCheckpoint kiểm tra response có phải checkpoint không
func isCheckpoint(resp string) bool {
	checkWords := []string{"checkpoint", "login.php", "nt/screen/?", "checkpoint_url", "checkpoint_flow_id"}
	for _, w := range checkWords {
		if strings.Contains(resp, w) {
			return true
		}
	}
	return false
}

// ─── TOTP generator ───────────────────────────────────────────────────────────

// generateTOTP sinh 6-chữ-số TOTP theo RFC 6238 (không cần external API)
// C# dùng https://2fa.live/tok/{secret} — Go dùng HMAC-SHA1 tự implement
func generateTOTP(secret string) (string, error) {
	// Base32 decode (loại bỏ khoảng trắng, uppercase)
	secret = strings.ToUpper(strings.ReplaceAll(secret, " ", ""))
	var key []byte
	var err error
	// Thử không có padding trước, nếu fail thì thêm padding
	key, err = base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		key, err = base32.StdEncoding.DecodeString(secret)
		if err != nil {
			return "", fmt.Errorf("base32 decode: %w", err)
		}
	}

	// counter = floor(UnixTime / 30) — 8 bytes big endian
	counter := uint64(time.Now().Unix() / 30)
	msg := make([]byte, 8)
	binary.BigEndian.PutUint64(msg, counter)

	// HMAC-SHA1
	mac := hmac.New(sha1.New, key)
	mac.Write(msg)
	h := mac.Sum(nil)

	// Dynamic truncation
	offset := h[len(h)-1] & 0x0F
	code := (uint32(h[offset])&0x7F)<<24 |
		uint32(h[offset+1])<<16 |
		uint32(h[offset+2])<<8 |
		uint32(h[offset+3])
	code %= 1_000_000

	return fmt.Sprintf("%06d", code), nil
}

// ─── Utilities ────────────────────────────────────────────────────────────────

func jsonEscape2FA(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func qplID2FA() int64 {
	return int64(mrand.Intn(900000000) + 100000000)
}

func sleepRandom(minSec, maxSec int) {
	d := minSec + mrand.Intn(maxSec-minSec+1)
	time.Sleep(time.Duration(d) * time.Second)
}

func min2FA(a, b int) int {
	if a < b {
		return a
	}
	return b
}
