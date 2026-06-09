// http.go — S23 HTTP transport + request headers.
//
// File này gồm:
//   - session struct: tls-client với HTTP/2 + Okhttp4Android13 profile
//   - HTTP methods: post / postGzip / getCookiesStr / addCookie / clearCookies
//   - buildHeaders: V3 header order match C# RegisterWIFIHeaderCollection +
//                   FullRegisterHeader + S23-specific overrides
//   - generateUSDID: ECDSA P-256 signed token cho x-meta-usdid
package s23

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/google/uuid"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

// ─── HTTP session (tls-client) ────────────────────────────────────────────────

type session struct {
	client tls_client.HttpClient
}

// newSession creates HTTP/2 TLS client with Okhttp4Android13 profile (closest to Samsung S23)
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
		return nil, fmt.Errorf("create s23 tls client: %w", err)
	}
	return &session{client: c}, nil
}

// post sends POST form-urlencoded body plaintext (C# S23 KHÔNG gzip).
// Match IHttpRequestClient.Post() + CustomHttpSingleton.FormUrlEncoded trong C#.
func (s *session) post(ctx context.Context, targetURL, body string, headers [][2]string) (string, error) {
	req, err := fhttp.NewRequestWithContext(ctx, "POST", targetURL, strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create POST request: %w", err)
	}

	// Apply headers with order
	order := make([]string, 0, len(headers))
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
		order = append(order, kv[0])
	}
	req.Header[fhttp.HeaderOrderKey] = order

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

// postGzip giữ lại cho backward compat — gọi s.post (không gzip) để match C#.
// Nếu cần thực sự gzip sau này (giả định tls-client đang fake stack khác),
// bật lại logic gzip ở đây.
func (s *session) postGzip(ctx context.Context, targetURL, body string, headers [][2]string) (string, error) {
	_ = bytes.Buffer{} // keep import
	_ = gzip.Writer{}  // keep import
	return s.post(ctx, targetURL, body, headers)
}

// getCookiesStr returns all cookies for facebook.com
func (s *session) getCookiesStr() string {
	seen := map[string]bool{}
	parts := make([]string, 0)
	for _, rawURL := range []string{"https://b-graph.facebook.com", "https://graph.facebook.com"} {
		u, _ := url.Parse(rawURL)
		for _, c := range s.client.GetCookies(u) {
			if !seen[c.Name] {
				seen[c.Name] = true
				parts = append(parts, c.Name+"="+c.Value)
			}
		}
	}
	return strings.Join(parts, ";")
}

// addCookie sets a cookie on the client
func (s *session) addCookie(name, value string) {
	u, _ := url.Parse("https://b-graph.facebook.com")
	s.client.SetCookies(u, []*fhttp.Cookie{
		{Name: name, Value: value, Path: "/", Domain: ".facebook.com"},
	})
}

// clearCookies xoá toàn bộ cookie cho facebook.com domains. Dùng khi worker
// context reuse session cho reg kế — cookies từ reg trước không nên leak sang.
func (s *session) clearCookies() {
	for _, rawURL := range []string{
		"https://b-graph.facebook.com",
		"https://graph.facebook.com",
		"https://m.facebook.com",
		"https://www.facebook.com",
	} {
		u, _ := url.Parse(rawURL)
		// Expire tất cả cookie hiện tại bằng cách set MaxAge=-1.
		existing := s.client.GetCookies(u)
		expired := make([]*fhttp.Cookie, 0, len(existing))
		for _, c := range existing {
			expired = append(expired, &fhttp.Cookie{Name: c.Name, Value: "", Path: "/", Domain: c.Domain, MaxAge: -1})
		}
		if len(expired) > 0 {
			s.client.SetCookies(u, expired)
		}
	}
}

// sleep helper
func sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// ─── Register headers ─────────────────────────────────────────────────────────
//
// Port C# FacebookApiHeaderCollectionBuilder:
//   - RegisterWIFIHeaderCollection (L490-527)
//   - FullRegisterHeader (L528-546)
//
// Order matters — Dictionary.Add C# preserves insertion order. HTTP/2 Facebook
// không enforce case, nhưng để match wire format chuẩn thì giữ nguyên tên C#.

// buildHeaders sinh headers cho POST /graphql register.
//
// Thứ tự (port từ C# Register + RegisterWIFIHeaderCollection + FullRegisterHeader):
//  1. RegisterWIFIHeaderCollection(S23OAuthToken, true, accountInfo):
//     - Authorization: OAuth {token}
//     - X-Fb-Connection-Type: {WIFI | mobile.LTE}
//     - X-Fb-Sim-Hni: {HNI}     (addsimhni=true)
//     - X-Fb-Net-Hni: {HNI}
//     ⬇ FullRegisterHeader():
//     - X-Graphql-Client-Library: graphservice
//     - X-Tigon-Is-Retry: False
//     - X-Graphql-Request-Purpose: fetch
//     - X-Fb-Privacy-Context: 3643298472347298
//     - x-fb-request-analytics-tags: {...}
//     - x-zero-eh: {random}
//     - x-zero-state: unknown
//     - X-Fb-Http-Engine: Tigon/Liger
//     - X-Fb-Client-Ip: True
//     - X-Fb-Server-Cluster: True
//     - X-Fb-Rmd: state=URL_ELIGIBLE
//     - X-Fb-Friendly-Name: FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.create.account.async
//  2. Thêm sau (S23.Register L74-85):
//     - X-Fb-Device-Group: {deviceGroup}
//     - App-Scope-Id-Header: {deviceid}
//     - X-Fb-Integrity-Machine-Id: {machineId}  (chỉ khi MachineId != "")
//     - X-Zero-F-Device-Id: {familyDeviceId}
//  3. S23-specific overrides:
//     - X-Meta-Zca: "empty_token"   (override _defaultMetaZcaHeaderValue)
//     - x-meta-usdid: {ECDSA sign}
//     - x-fb-conn-uuid-client: {uuid-no-dashes}
func buildHeaders(profile S23Profile) [][2]string {
	analyticsTag := `{"network_tags":{"product":"350685531728","purpose":"fetch","request_category":"graphql","retry_attempt":"0"},"application_tags":"graphservice"}`

	xZeroEh := strings.ReplaceAll(uuid.New().String(), "-", "")

	h := [][2]string{
		// === RegisterWIFIHeaderCollection ===
		{"Authorization", "OAuth " + s23OAuthToken},
		{"X-Fb-Connection-Type", profile.ConnType},
		{"X-Fb-Sim-Hni", profile.Sim.HNI},
		{"X-Fb-Net-Hni", profile.Sim.HNI},
		// === FullRegisterHeader ===
		{"X-Graphql-Client-Library", "graphservice"},
		{"X-Tigon-Is-Retry", "False"},
		{"X-Graphql-Request-Purpose", "fetch"},
		{"X-Fb-Privacy-Context", "3643298472347298"},
		{"x-fb-request-analytics-tags", analyticsTag},
		{"x-zero-eh", xZeroEh},
		{"x-zero-state", "unknown"},
		{"X-Fb-Http-Engine", "Tigon/Liger"},
		{"X-Fb-Client-Ip", "True"},
		{"X-Fb-Server-Cluster", "True"},
		{"X-Fb-Rmd", "state=URL_ELIGIBLE"},
		{"X-Fb-Friendly-Name", s23FriendlyName},
		// === S23.Register post-collection (L74-85) ===
		{"X-Fb-Device-Group", profile.DeviceGroup},
		{"App-Scope-Id-Header", profile.DeviceID},
	}
	// X-Fb-Integrity-Machine-Id chỉ khi có datr mồi (C# guard `if (!string.IsNullOrEmpty)`)
	if profile.MachineID != "" {
		h = append(h, [2]string{"X-Fb-Integrity-Machine-Id", profile.MachineID})
	}
	h = append(h, [2]string{"X-Zero-F-Device-Id", profile.FamilyDeviceID})

	// === S23 overrides (L83-85) ===
	h = append(h,
		[2]string{"X-Meta-Zca", s23MetaZCA},
		[2]string{"x-meta-usdid", generateUSDID()},
		[2]string{"x-fb-conn-uuid-client", connUUID()},
	)

	// === User-Agent + Content-Type (auto) ===
	h = append(h,
		[2]string{"user-agent", profile.S23UA},
		[2]string{"content-type", "application/x-www-form-urlencoded"},
	)
	return h
}

// generateUSDID — x-meta-usdid: ECDSA P-256 signed "{uuid}.{unix_ts}" → base64url.
// Port C# FacebookRegisterAPIAndroidS23.GenerateUSDID (L310-328).
func generateUSDID() string {
	id := uuid.New().String()
	ts := fmt.Sprintf("%d", time.Now().Unix())
	payload := id + "." + ts

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return payload + ".error"
	}
	hash := sha256.Sum256([]byte(payload))
	sig, err := ecdsa.SignASN1(rand.Reader, key, hash[:])
	if err != nil {
		return payload + ".error"
	}

	return payload + "." + base64.RawURLEncoding.EncodeToString(sig)
}
