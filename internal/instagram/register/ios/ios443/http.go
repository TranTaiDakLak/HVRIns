// http.go — iOS555 TLS session + request headers.
//
// Headers map 1:1 từ capture APIRegVer_IOS [125]/[126] (Action request).
// Body gửi gzip (capture có Content-Encoding: gzip).
package ios443

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/google/uuid"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

// graphURL — endpoint native FBIOS gọi tới (capture: graph.facebook.com).
const graphURL = "https://graph.facebook.com/graphql"

// ─── HTTP session ────────────────────────────────────────────────────────────

type session struct {
	client tls_client.HttpClient
}

// NewSession exported alias — dùng bởi ios563.
func NewSession(proxyStr, iosVersion string) (*session, error) {
	return newSession(proxyStr, iosVersion)
}

// PostGzip exported alias — dùng bởi ios563.
func (s *session) PostGzip(ctx context.Context, url, body string, headers [][2]string) (string, error) {
	return s.postGzip(ctx, url, body, headers)
}

// newSession tạo TLS client với fingerprint khớp iOS version của device đang dùng.
func newSession(proxyStr string, iosVersion string) (*session, error) {
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(40),
		tls_client.WithClientProfile(tlsProfileForIOS(iosVersion)),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
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
		return nil, fmt.Errorf("create ios562 tls client: %w", err)
	}
	return &session{client: c}, nil
}

// tlsProfileForIOS chọn TLS client profile khớp với iOS major version.
// Mapping theo profiles có sẵn trong bogdanfinn/tls-client v1.14.0.
func tlsProfileForIOS(iosDot string) profiles.ClientProfile {
	switch {
	case len(iosDot) >= 2 && iosDot[:2] == "26":
		return profiles.Safari_IOS_26_0
	case len(iosDot) >= 2 && iosDot[:2] == "18":
		return profiles.Safari_IOS_18_5 // 18.5 gần nhất với 18.7.9
	case len(iosDot) >= 2 && iosDot[:2] == "17":
		return profiles.Safari_IOS_17_0
	case len(iosDot) >= 2 && iosDot[:2] == "16":
		return profiles.Safari_IOS_16_0
	default:
		return profiles.Safari_IOS_15_6 // iOS 15.x
	}
}

// postGzip gzip-nén body rồi POST. Trả về body response (string).
func (s *session) postGzip(ctx context.Context, targetURL, body string, headers [][2]string) (string, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(body)); err != nil {
		return "", fmt.Errorf("gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return "", fmt.Errorf("gzip close: %w", err)
	}
	md5Sum := md5.Sum(buf.Bytes())
	headers = append(headers, [2]string{"content-md5", base64.StdEncoding.EncodeToString(md5Sum[:])})

	req, err := fhttp.NewRequestWithContext(ctx, "POST", targetURL, &buf)
	if err != nil {
		return "", fmt.Errorf("create POST request: %w", err)
	}
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

// httpGet performs a GET request with preserved header order.
func (s *session) httpGet(ctx context.Context, targetURL string, headers [][2]string) (string, error) {
	req, err := fhttp.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("create GET request: %w", err)
	}
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

	data, err := httpx.ReadBody(resp.Body, 64*1024)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return string(data), fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return string(data), nil
}

// ─── Headers ─────────────────────────────────────────────────────────────────

// buildHeaders dựng header cho 1 Action request (create.account).
// Thứ tự + tên header giữ giống capture [434].
func buildHeaders(p IOSProfile) [][2]string {
	analyticsTag := `{"network_tags":{"product":"6628568379","request_category":"image","purpose":"fetch","retry_attempt":"0"},"application_tags":"BKImageComponent;recommendation_image;com.bloks.www.bloks.caa.reg.create.account.async"}`
	const hFriendlyName = "BKImageComponent,com.bloks.www.bloks.caa.reg.create.account.async,recommendation_image"
	return [][2]string{
		{"user-agent", p.UserAgent},
		{"accept-encoding", "gzip, deflate, br"},
		{"accept", "*/*"},
		{"connection", "keep-alive"},
		{"x-fb-appnetsession-sid", appnetSID()},
		{"x-meta-usdid", generateUSDID()},
		{"x-fb-http-engine", "Tigon/Liger"},
		{"x-meta-zca", `{"e": {"c":7}}`},
		{"x-fb-session-gk", "v1:gk:fb_ios_tasos_congestion_signal:@pass;"},
		{"authorization", "OAuth " + oauthToken},
		{"x-fb-sim-hni", p.Sim.HNI},
		{"content-encoding", "gzip"},
		{"x-fb-appnetsession-nid", appnetNID()},
		{"x-fb-connection-type", p.ConnType},
		{"x-cloud-trust-token", p.CloudTrustID},
		{"x-fb-integrity-machine-id", p.MachineID},
		{"x-fb-device-id", p.DeviceID},
		{"x-fb-friendly-name", hFriendlyName},
		{"x-fb-tasos-experimental", "1"},
		{"content-type", "application/x-www-form-urlencoded"},
		{"x-tigon-is-retry", "False"},
		{"x-fb-request-analytics-tags", analyticsTag},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
		{"x-fb-conn-uuid-client", connUUID()},
		{"x-graphql-client-library", "pando"},
		{"x-graphql-request-purpose", "fetch"},
	}
}

// connUUID — x-fb-conn-uuid-client: base64(16 raw UUID bytes).
func connUUID() string {
	id := uuid.New()
	return base64.StdEncoding.EncodeToString(id[:])
}

// appnetSID — x-fb-appnetsession-sid: 32-char lowercase hex (16 random bytes).
func appnetSID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// appnetNID — x-fb-appnetsession-nid: 32-char lowercase hex + ",Wifi".
func appnetNID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b) + ",Wifi"
}

// generateUSDID — x-meta-usdid: "{uuid}.{unix_ts}.{base64url ECDSA-P256 sig}".
// Khớp format capture; chữ ký tự sinh (server không verify ngược được key này).
func generateUSDID() string {
	id := strings.ToUpper(uuid.New().String())
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
