// http.go — Android HTTP transport + register V3 headers.
//
// File này gộp 2 file cũ:
//   - httpclient.go → doPost, session struct + newSession/post/clearCookies
//   - headers.go    → buildRegisterHeaders + buildBatchHeaders + V3 generators
//
// Dùng bogdanfinn/tls-client với Okhttp4Android13 profile để TLS fingerprint
// giống Facebook Android app (OkHttp4).
package android

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/google/uuid"

	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

// ─── HTTP client (single-shot POST with Android OkHttp4 TLS fingerprint) ─────

// doPost gửi POST với Android OkHttp4 TLS fingerprint.
// headers là [][2]string (ordered pairs) để đảm bảo thứ tự header đúng như C#.
// Header key case được giữ nguyên (lowercase x-zero-eh, x-fb-request-analytics-tags).
func doPost(ctx context.Context, targetURL, body string, headers [][2]string, proxyStr string) (string, error) {
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Okhttp4Android13),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithInsecureSkipVerify(),
	}
	if proxyStr != "" {
		if formatted := proxy.FormatProxyURL(proxyStr); formatted != "" {
			opts = append(opts, tls_client.WithProxyUrl(formatted))
		}
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return "", fmt.Errorf("create tls client: %w", err)
	}
	defer client.CloseIdleConnections() // Giải phóng connection + memory sau mỗi request

	req, err := fhttp.NewRequestWithContext(ctx, "POST", targetURL, strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	// Set headers giữ nguyên case (dùng map trực tiếp, không qua .Set() — .Set() sẽ canonicalize)
	// HeaderOrderKey báo cho tls-client gửi headers theo đúng thứ tự C#
	bodyLen := fmt.Sprintf("%d", len(body))

	headerOrder := make([]string, 0, len(headers)+1)
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
		if kv[0] == "User-Agent" {
			// C# thêm Content-Length trước User-Agent (sau X-Meta-Zca)
			req.Header["Content-Length"] = []string{bodyLen}
			headerOrder = append(headerOrder, "Content-Length")
		}
		headerOrder = append(headerOrder, kv[0])
	}
	// Fallback: nếu không tìm thấy User-Agent, append Content-Length cuối
	if req.Header["Content-Length"] == nil {
		req.Header["Content-Length"] = []string{bodyLen}
		headerOrder = append(headerOrder, "Content-Length")
	}

	// Enforce header order for HTTP/2 HPACK (critical for TLS fingerprint matching C#)
	req.Header[fhttp.HeaderOrderKey] = headerOrder

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := httpx.ReadBody(resp.Body, 512*1024)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return string(data), fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return string(data), nil
}

// formatProxyURL — giữ lại để không break code khác.
func formatProxyURL(proxyStr string) string {
	return proxyStr
}

// createHTTPClient — giữ lại cho backward compat.
func createHTTPClient(proxyStr string, timeout time.Duration) interface{} {
	return nil
}

// ─── Session (1 client = 1 account = 1 cookie jar) ───────────────────────────
//
// Port C# IHttpRequestClient pattern. Shared giữa pwdKeyFetch → register
// → xzero_eh → logout để match C# V3 flow (cookie jar persist).

// session — 1 client = 1 account = 1 cookie jar.
type session struct {
	client tls_client.HttpClient
}

// newSession tạo HTTP client với cookie jar persist giữa các request trong 1 account.
func newSession(proxyStr string) (*session, error) {
	jar := tls_client.NewCookieJar()
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Okhttp4Android13),
		tls_client.WithCookieJar(jar),
		tls_client.WithInsecureSkipVerify(),
		tls_client.WithNotFollowRedirects(),
	}
	if proxyStr != "" {
		if formatted := proxy.FormatProxyURL(proxyStr); formatted != "" {
			opts = append(opts, tls_client.WithProxyUrl(formatted))
		}
	}
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return nil, fmt.Errorf("create android tls client: %w", err)
	}
	return &session{client: c}, nil
}

// clearCookies xoá toàn bộ cookie facebook.com domains. Dùng khi WorkerContext
// reuse session cho reg kế — cookies từ pwd_key/reg trước không nên leak sang.
func (s *session) clearCookies() {
	urls := []string{
		"https://b-graph.facebook.com",
		"https://graph.facebook.com",
		"https://m.facebook.com",
		"https://www.facebook.com",
	}
	for _, rawURL := range urls {
		u, _ := url.Parse(rawURL)
		existing := s.client.GetCookies(u)
		if len(existing) == 0 {
			continue
		}
		expired := make([]*fhttp.Cookie, 0, len(existing))
		for _, c := range existing {
			expired = append(expired, &fhttp.Cookie{Name: c.Name, Value: "", Path: "/", Domain: c.Domain, MaxAge: -1})
		}
		s.client.SetCookies(u, expired)
	}
}

func (s *session) addCookie(name, value string) {
	u, _ := url.Parse("https://b-graph.facebook.com")
	s.client.SetCookies(u, []*fhttp.Cookie{
		{Name: name, Value: value, Path: "/", Domain: ".facebook.com"},
	})
}

// post — POST qua session, áp header order giống doPost.
func (s *session) post(ctx context.Context, targetURL, body string, headers [][2]string) (string, error) {
	req, err := fhttp.NewRequestWithContext(ctx, "POST", targetURL, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	headerOrder := make([]string, 0, len(headers)+1)
	bodyLen := fmt.Sprintf("%d", len(body))
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
		if kv[0] == "user-agent" || kv[0] == "User-Agent" {
			req.Header["content-length"] = []string{bodyLen}
			headerOrder = append(headerOrder, "content-length")
		}
		headerOrder = append(headerOrder, kv[0])
	}
	if req.Header["content-length"] == nil && req.Header["Content-Length"] == nil {
		req.Header["content-length"] = []string{bodyLen}
		headerOrder = append(headerOrder, "content-length")
	}
	req.Header[fhttp.HeaderOrderKey] = headerOrder

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := httpx.ReadBody(resp.Body, 512*1024)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return string(data), fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return string(data), nil
}

// ─── Register V3 headers ─────────────────────────────────────────────────────
//
// Port 1:1 từ C# FacebookRegisterAPIAndroidV2:
//   - RegisterWIFIHeaderCollectionV3  (V2.cs L464-516)
//   - Register() additional headers    (V2.cs L900-909)
//
// V3 thay đổi so với V1/V2 bản cũ:
//   - x-meta-usdid (format {uuid}.{ts}.{base64url_ecdsa_sig})
//   - x-fb-conn-uuid-client (16 random bytes base64 — KHÔNG phải uuid-no-dashes)
//   - x-meta-zca = "empty_token" (không còn dùng base64 blob cho Register)
//   - x-fb-request-analytics-tags có trường `request_category` TRƯỚC `purpose`
//   - x-zero-f-device-id = deviceid (không còn family_device_id)
//   - x-fb-integrity-machine-id dùng MachineId2 (không phải datr/MachineId)
//   - Thứ tự header mới (theo traffic capture)

// buildRegisterHeaders sinh headers POST /graphql create.account.async (V3).
// Match C# RegisterWIFIHeaderCollectionV3 + Register additional L900-909.
func buildRegisterHeaders(profile fakeinfo.FullRegProfile) [][2]string {
	// analyticsTag thứ tự V3: network_tags = product / request_category / purpose / retry_attempt
	analyticsTag := `{"network_tags":{"product":"350685531728","request_category":"graphql","purpose":"fetch","retry_attempt":"0"},"application_tags":"graphservice"}`

	xZeroEh := strings.ReplaceAll(uuid.New().String(), "-", "")

	simHNI := profile.Sim.HNI
	if simHNI == "" {
		simHNI = "45204"
	}
	connType := profile.ConnectionType
	if connType == "" {
		connType = "WIFI"
	}

	// ── RegisterWIFIHeaderCollectionV3 (V2.cs L466-515) ─────────────────────
	h := [][2]string{
		{"x-fb-request-analytics-tags", analyticsTag},
		{"x-fb-rmd", "state=URL_ELIGIBLE"},
		{"x-zero-eh", xZeroEh},
		{"x-fb-friendly-name", instagram.AndroidRegFriendlyName},
		{"x-graphql-request-purpose", "fetch"},
		{"x-tigon-is-retry", "False"},
		{"x-graphql-client-library", "graphservice"},
		{"x-fb-net-hni", simHNI},
		{"x-fb-sim-hni", simHNI},
		{"authorization", "OAuth " + instagram.AndroidOAuthToken},
		{"x-zero-state", "unknown"},
		{"x-meta-zca", "empty_token"}, // V3: empty_token
		{"x-fb-connection-type", connType},
		{"x-meta-usdid", generateUSDIDV3()},
		{"x-fb-http-engine", "Tigon/Liger"},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
		{"x-fb-conn-uuid-client", generateConnUUIDClientV3()},
	}

	// ── Register() additional (V2.cs L905-909) ──────────────────────────────
	h = append(h, [2]string{"x-fb-device-group", profile.DeviceGroup})
	h = append(h, [2]string{"app-scope-id-header", profile.DeviceID})
	if profile.MachineID != "" {
		h = append(h, [2]string{"x-fb-integrity-machine-id", profile.MachineID})
	} else if profile.MachineID2 != "" {
		h = append(h, [2]string{"x-fb-integrity-machine-id", profile.MachineID2})
	}
	// V3: x-zero-f-device-id = deviceid (NOT familyDeviceID)
	h = append(h, [2]string{"x-zero-f-device-id", profile.DeviceID})

	// ── UA + content-type (auto) ────────────────────────────────────────────
	h = append(h, [2]string{"user-agent", profile.UserAgent})
	h = append(h, [2]string{"content-type", "application/x-www-form-urlencoded"})

	return h
}

// generateUSDIDV3 port C# V2.cs L492-504.
// Format: `{uuid}.{unix_ts}.{base64url_ecdsa_sig_69bytes}`
// sig dùng random bytes với byte 0/1 = 0x30 0x45 (DER SEQUENCE), byte 2/3 = 0x02 0x21,
// byte 4 = 0x00, byte 37/38 = 0x02 0x20 — format DER ECDSA signature ~69 bytes.
func generateUSDIDV3() string {
	ts := time.Now().Unix()
	id := uuid.New().String()

	sig := make([]byte, 69)
	rand.Read(sig)
	sig[0] = 0x30
	sig[1] = 0x45
	sig[2] = 0x02
	sig[3] = 0x21
	sig[4] = 0x00
	sig[37] = 0x02
	sig[38] = 0x20

	sigB64 := base64.StdEncoding.EncodeToString(sig)
	sigB64 = strings.TrimRight(sigB64, "=")
	sigB64 = strings.ReplaceAll(sigB64, "+", "-")
	sigB64 = strings.ReplaceAll(sigB64, "/", "_")

	return fmt.Sprintf("%s.%d.%s", id, ts, sigB64)
}

// generateConnUUIDClientV3 port C# V2.cs L510-513.
// 16 random bytes → base64 (có padding).
func generateConnUUIDClientV3() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

// buildBatchHeaders giữ cho backward-compat — XZero fetch dùng bộ headers khác
// (xem extras.go). Hàm này chỉ dùng nếu caller cũ vẫn gọi.
func buildBatchHeaders(profile fakeinfo.FullRegProfile, accessToken string) [][2]string {
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
		{"X-Fb-Friendly-Name", fetchLoginDataBatchFriendlyName},
		{"X-Fb-Connection-Type", connType},
		{"X-Fb-Sim-Hni", simHNI},
		{"X-Fb-Net-Hni", simHNI},
		{"X-Zero-Eh", ""},
		{"X-Graphql-Client-Library", "graphservice"},
		{"X-Tigon-Is-Retry", "False"},
		{"X-Fb-Privacy-Context", "3643298472347298"},
		{"X-Graphql-Request-Purpose", "fetch"},
		{"x-fb-request-analytics-tags", analyticsTag},
		{"X-Fb-Http-Engine", "Tigon/Liger"},
		{"X-Fb-Client-Ip", "True"},
		{"X-Fb-Server-Cluster", "True"},
		{"X-Fb-Device-Group", profile.DeviceGroup},
		{"X-Fb-Conn-Uuid-Client", strings.ReplaceAll(uuid.New().String(), "-", "")},
		{"App-Scope-Id-Header", profile.DeviceID},
		{"X-Zero-F-Device-Id", profile.FamilyDeviceID},
		{"X-Zero-State", "unknown"},
		{"X-Meta-Zca", defaultMetaZcaBlob},
		{"User-Agent", profile.UserAgent},
		{"Content-Type", "application/x-www-form-urlencoded"},
	}
}
