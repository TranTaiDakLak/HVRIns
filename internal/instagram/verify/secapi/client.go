package secapi

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/google/uuid"

	"HVRIns/internal/proxy"
)

// Client gọi shared SecurityFeatureAPIAndroid endpoints với cấu hình từ Spec.
// Mapping C#: FacebookSecurityFeatureAPIAndroid (1 instance per account).
type Client struct {
	spec      Spec
	client    tls_client.HttpClient
	token     string // OAuth access token
	uid       string
	deviceID  string
	machineID string // = datr value
	locale    string
	ua        string // Android UA của variant
}

// NewClient khởi tạo Client với spec biến thể + session params.
// Caller PHẢI defer Close() để giải phóng idle TCP/TLS connection.
func NewClient(spec Spec, proxyStr, token, uid, deviceID, machineID, locale, ua string) (*Client, error) {
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
	}
	if proxyStr != "" {
		proxyURL := proxy.FormatProxyURL(proxyStr)
		if proxyURL != "" {
			opts = append(opts, tls_client.WithProxyUrl(proxyURL))
		}
	}
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return nil, err
	}
	return &Client{
		spec:      spec,
		client:    c,
		token:     token,
		uid:       uid,
		deviceID:  deviceID,
		machineID: machineID,
		locale:    locale,
		ua:        ua,
	}, nil
}

// Close giải phóng idle connection. Caller defer Close() sau NewClient.
func (c *Client) Close() {
	if c != nil && c.client != nil {
		c.client.CloseIdleConnections()
	}
}

// AddSubEmailResult kết quả từ AddSubEmail API call.
type AddSubEmailResult struct {
	Ok             bool
	TwoStepContext string // rỗng nếu không cần 2FA
}

// AddSubEmail — POST fx.settings.contact_point.add.async
// Mapping C#: FacebookSecurityFeatureAPIAndroid.AddSubEmail()
// mainEmail: email chính của account (để gửi 2FA OTP nếu FB yêu cầu xác minh).
// Trả về TwoStepContext nếu FB yêu cầu 2FA, rỗng nếu thêm thẳng không cần xác minh.
func (c *Client) AddSubEmail(ctx context.Context, mainEmail, subEmail string) AddSubEmailResult {
	variables := buildAddSubEmailVariables(c.deviceID, c.uid, subEmail, c.spec)
	body := c.buildBody(FnAddSubEmail, c.spec.DocIDAddSubEmail, variables)
	headers := c.buildHeaders(FnAddSubEmail)

	resp, err := doPost(ctx, c.client, graphURL, body, headers)
	if err != nil || resp == "" {
		return AddSubEmailResult{}
	}

	// Parse two_step_verification_context — C# dùng 2 regex fallback
	// (SecurityFeatureAPIAndroid.cs:757, 762). FB route qua 1 trong 2 pattern
	// tuỳ experiment: INTERNAL_INFRA_screen_id (secured_action) hoặc
	// xfac_universal_entry_organic_sensitive_actions.
	cleaned := strings.ReplaceAll(resp, `\\\`, "")
	re1 := regexp.MustCompile(`INTERNAL_INFRA_screen_id".*?"(.*?)", "secured_action",`)
	re2 := regexp.MustCompile(`xfac_universal_entry_organic_sensitive_actions".*?"(.*?)"`)

	var ctx2 string
	if m := re1.FindStringSubmatch(cleaned); len(m) > 1 && m[1] != "" {
		ctx2 = m[1]
	} else if m := re2.FindStringSubmatch(cleaned); len(m) > 1 && m[1] != "" {
		ctx2 = m[1]
	}
	if ctx2 != "" {
		// C#: AddSubEmail gọi nội tuyến SendCodeVerifyEmail ngay sau khi có
		// twoStepCtx để FB gửi OTP về main email.
		c.SendCodeVerifyEmail(ctx, MaskEmail(mainEmail), ctx2)
		return AddSubEmailResult{Ok: true, TwoStepContext: ctx2}
	}

	// Không có 2FA context → check xem email có trong response không
	unescaped, _ := url.QueryUnescape(resp)
	if strings.Contains(unescaped, subEmail) || strings.Contains(resp, subEmail) {
		return AddSubEmailResult{Ok: true}
	}
	return AddSubEmailResult{}
}

// SendCodeVerifyEmail — kích OTP về main email để xác minh 2FA.
// Gọi sau khi AddSubEmail trả về TwoStepContext.
// Mapping C#: FacebookSecurityFeatureAPIAndroid.AddSubEmail (nội tuyến sau khi có context).
func (c *Client) SendCodeVerifyEmail(ctx context.Context, maskedMainEmail, twoStepCtx string) bool {
	variables := buildSendCodeVerifyEmailVariables(c.deviceID, maskedMainEmail, twoStepCtx, c.machineID, c.spec)
	body := c.buildBody(FnSendCodeVerify, c.spec.DocIDContactPoint, variables)
	headers := c.buildHeaders(FnSendCodeVerify)

	resp, err := doPost(ctx, c.client, graphURL, body, headers)
	if err != nil {
		return false
	}
	return strings.Contains(resp, "com.bloks.www.two_step_verification.send_code.async")
}

// ConfirmTwoStepVerification — xác nhận OTP 2FA từ main email với twoStepContext.
// Mapping C#: FacebookSecurityFeatureAPIAndroid.ConfirmTwoStepVerificationEmail()
// Success: response chứa "SESSION_STORE_BLOKS_SECURED_ACTION_REAUTH:success".
func (c *Client) ConfirmTwoStepVerification(ctx context.Context, maskedMainEmail, twoStepCtx, code string) bool {
	variables := buildConfirmTwoStepVariables(c.deviceID, maskedMainEmail, twoStepCtx, code, c.machineID, c.spec)
	body := c.buildBody(FnConfirmTwoStep, c.spec.DocIDContactPoint, variables)
	headers := c.buildHeaders(FnConfirmTwoStep)

	resp, err := doPost(ctx, c.client, graphURL, body, headers)
	if err != nil {
		return false
	}
	return strings.Contains(resp, "SESSION_STORE_BLOKS_SECURED_ACTION_REAUTH:success") ||
		strings.Contains(resp, "BLOKS_SECURED_ACTION_REAUTH_success")
}

// ConfirmSubEmailCode — confirm sub email bằng OTP từ sub email inbox.
// Mapping C#: FacebookSecurityFeatureAPIAndroid.ConfirmSubEmailCode()
// Success: response chứa "FX_SETTINGS_CONTACT_POINT:should_refresh_root_page".
func (c *Client) ConfirmSubEmailCode(ctx context.Context, code, subEmail string) bool {
	variables := buildConfirmSubEmailCodeVariables(c.deviceID, code, c.uid, subEmail, c.spec)
	body := c.buildBody(FnConfirmSubEmail, c.spec.DocIDConfirmSubEmail, variables)
	headers := c.buildHeaders(FnConfirmSubEmail)

	resp, err := doPost(ctx, c.client, graphURL, body, headers)
	if err != nil {
		return false
	}
	return strings.Contains(resp, "FX_SETTINGS_CONTACT_POINT:should_refresh_root_page")
}

// ─── HTTP helpers ────────────────────────────────────────────────────────────

// buildHeaders — mapping C#: BaseAndroidAPIHeadersWIFI + extra headers.
func (c *Client) buildHeaders(friendlyName string) [][2]string {
	analyticsTag := `{"network_tags":{"product":"350685531728","purpose":"fetch","request_category":"graphql","retry_attempt":"0"},"application_tags":"graphservice"}`
	return [][2]string{
		{"Authorization", "OAuth " + c.token},
		{"X-Fb-Friendly-Name", friendlyName},
		{"X-Fb-Connection-Type", "WIFI"},
		{"X-Fb-Sim-Hni", "45204"},
		{"X-Fb-Net-Hni", "45204"},
		{"X-Graphql-Client-Library", "graphservice"},
		{"X-Tigon-Is-Retry", "False"},
		{"X-Fb-Privacy-Context", "3643298472347298"},
		{"X-Graphql-Request-Purpose", "fetch"},
		{"X-Fb-Http-Engine", "Tigon/Liger"},
		{"X-Fb-Client-Ip", "True"},
		{"X-Fb-Server-Cluster", "True"},
		{"X-Fb-Request-Analytics-Tags", analyticsTag},
		{"App-Scope-Id-Header", c.deviceID},
		{"X-Zero-F-Device-Id", c.deviceID},
		{"X-Meta-Zca", c.spec.MetaZcaValue},
		{"User-Agent", c.ua},
		{"Content-Type", "application/x-www-form-urlencoded"},
	}
}

// buildBody — mapping C#: BaseAndroidGraphqlFormData().
func (c *Client) buildBody(friendlyName, docID, variables string) string {
	v := url.Values{}
	v.Set("method", "post")
	v.Set("pretty", "false")
	v.Set("format", "json")
	v.Set("server_timestamps", "true")
	v.Set("locale", c.locale)
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

// ─── Variables builders ──────────────────────────────────────────────────────
// Mapping C#: *MobileVariables() methods trong FacebookApiFormDataBuilder.
// Structure: triple-nested JSON (innermost → JSON string → JSON string → outer params).

func buildAddSubEmailVariables(deviceID, uid, email string, spec Spec) string {
	qplID := qplInstanceID()
	inner := fmt.Sprintf(`{"client_input_params":{"country":null,"family_device_id":"%s","device_id":"%s","selected_accounts":["%s"],"contact_point":"%s"},"server_params":{"contact_point_event_type":"add","serialized_states":{"input_error_message":"2;1uh64oomfy;0","input_error":"2;1uh64oomfx;0"},"INTERNAL__latency_qpl_marker_id":36707139,"contact_point_source":"fx_settings","requested_screen_component_type":null,"machine_id":null,"INTERNAL__latency_qpl_instance_id":%d,"should_send_via_whatsapp":0,"should_check_for_suspicious_email":1,"contact_point_type":"email"}}`,
		deviceID, deviceID, uid, url.QueryEscape(email), qplID)
	return buildContactPointParams(inner, "com.bloks.www.fx.settings.contact_point.add.async", spec)
}

func buildSendCodeVerifyEmailVariables(deviceID, maskedCP, twoStepCtx, machineID string, spec Spec) string {
	qplID := qplInstanceID()
	inner := fmt.Sprintf(`{"client_input_params":{},"server_params":{"INTERNAL__latency_qpl_marker_id":36707139,"block_store_machine_id":null,"device_id":"%s","cloud_trust_token":null,"masked_cp":"%s","challenge":"email","machine_id":"%s","INTERNAL__latency_qpl_instance_id":%d,"two_step_verification_context":"%s","flow_source":"secured_action"}}`,
		deviceID, maskedCP, machineID, qplID, twoStepCtx)
	return buildContactPointParams(inner, "com.bloks.www.two_step_verification.send_code.async", spec)
}

func buildConfirmTwoStepVariables(deviceID, maskedCP, twoStepCtx, code, machineID string, spec Spec) string {
	qplID := qplInstanceID()
	inner := fmt.Sprintf(`{"client_input_params":{"auth_secure_device_id":"","block_store_machine_id":null,"code":"%s","should_trust_device":0,"family_device_id":"%s","device_id":"%s","cloud_trust_token":null,"machine_id":"%s"},"server_params":{"INTERNAL__latency_qpl_marker_id":36707139,"block_store_machine_id":null,"device_id":"%s","cloud_trust_token":null,"masked_cp":"%s","challenge":"email","machine_id":"%s","INTERNAL__latency_qpl_instance_id":%d,"two_step_verification_context":"%s","flow_source":"secured_action"}}`,
		code, deviceID, deviceID, machineID, deviceID, maskedCP, machineID, qplID, twoStepCtx)
	return buildContactPointParams(inner, "com.bloks.www.two_step_verification.verify_code.async", spec)
}

func buildConfirmSubEmailCodeVariables(deviceID, code, uid, email string, spec Spec) string {
	qplID := qplInstanceID()
	inner := fmt.Sprintf(`{"client_input_params":{"pin_code":"%s","family_device_id":"%s"},"server_params":{"contact_point_event_type":"add","INTERNAL__latency_qpl_marker_id":36707139,"contact_point_source":"fx_settings","notif_medium":"email","requested_screen_component_type":null,"machine_id":null,"INTERNAL__latency_qpl_instance_id":%d,"normalized_contact_point":"%s","selected_accounts":"%s","contact_point_type":"email"}}`,
		code, deviceID, qplID, url.QueryEscape(email), uid)
	return buildContactPointParams(inner, "com.bloks.www.fx.settings.contact_point.verify.async", spec)
}

// buildContactPointParams bọc inner JSON vào triple-nested structure.
// Mapping C#: pre-encoded variables template với nt_context tuỳ biến theo spec.
func buildContactPointParams(innerJSON, appID string, spec Spec) string {
	// Level 3 (innermost): innerJSON đã là map
	// Level 2: {"params": "<json_escaped_innerJSON>"}
	l2 := fmt.Sprintf(`{"params":"%s"}`, jsonEscape(innerJSON))
	// Level 1: {"params": "<json_escaped_l2>", "bloks_versioning_id": "...", "app_id": "..."}
	l1 := fmt.Sprintf(`{"params":"%s","bloks_versioning_id":"%s","app_id":"%s"}`,
		jsonEscape(l2), spec.BloksVerContact, appID)
	// Outer: {"params": l1, "scale": "3", "nt_context": {...}} — nt_context dùng theme + is_push_on theo spec
	pushOn := "false"
	if spec.IsPushOn {
		pushOn = "true"
	}
	ntContext := fmt.Sprintf(
		`{"using_white_navbar":true,"styles_id":"ca49d9aab0f1291c1131e3f816210b58","pixel_ratio":3,"is_push_on":%s,"debug_tooling_metadata_token":null,"is_flipper_enabled":false,"theme_params":%s,"bloks_version":"%s"}`,
		pushOn, spec.ThemeParamsJSON, spec.BloksVerContact,
	)
	return fmt.Sprintf(`{"params":%s,"scale":"3","nt_context":%s}`, l1, ntContext)
}
