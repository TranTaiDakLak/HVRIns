// http.go — S450 HTTP transport + request headers.
// Khác s557: bỏ appnetsession/tasos/qpl/network-props headers;
// thêm x-fb-rmd, x-zero-eh, x-zero-state; accept-encoding: gzip, deflate.
package s450

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
	mrand "math/rand"
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

// ─── HTTP session ─────────────────────────────────────────────────────────────

type session struct {
	client tls_client.HttpClient
}

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
		return nil, fmt.Errorf("create S560 tls client: %w", err)
	}
	return &session{client: c}, nil
}

// postGzip gzip-compresses body và gửi POST.
func (s *session) postGzip(ctx context.Context, targetURL, body string, headers [][2]string) (string, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(body)); err != nil {
		return "", fmt.Errorf("gzip write: %w", err)
	}
	if err := gz.Close(); err != nil {
		return "", fmt.Errorf("gzip close: %w", err)
	}
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

func (s *session) addCookie(name, value string) {
	u, _ := url.Parse("https://b-graph.facebook.com")
	s.client.SetCookies(u, []*fhttp.Cookie{
		{Name: name, Value: value, Path: "/", Domain: ".facebook.com"},
	})
}

func (s *session) clearCookies() {
	for _, rawURL := range []string{
		"https://b-graph.facebook.com",
		"https://graph.facebook.com",
		"https://m.facebook.com",
		"https://www.facebook.com",
	} {
		u, _ := url.Parse(rawURL)
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

func sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// buildHeaders — S560 header format (captured traffic).
// Khác s557: bỏ network-props/appnet/tasos/qpl; thêm rmd/zero-eh/zero-state.
func buildHeaders(profile S560Profile) [][2]string {
	analyticsTag := `{"network_tags":{"product":"350685531728","request_category":"graphql","purpose":"fetch","retry_attempt":"0"},"application_tags":"graphservice"}`

	h := [][2]string{
		{"x-fb-request-analytics-tags", analyticsTag},
		{"x-fb-rmd", "state=URL_ELIGIBLE"},
		{"priority", "u=0"},
		{"content-encoding", "gzip"},
		{"x-fb-device-group", profile.DeviceGroup},
	}

	// x-fb-integrity-machine-id conditional (khi có MachineID/datr)
	if profile.MachineID != "" {
		h = append(h, [2]string{"x-fb-integrity-machine-id", profile.MachineID})
	}

	h = append(h,
		[2]string{"x-zero-eh", S560ZeroEH},
		[2]string{"user-agent", profile.S560UA},
		[2]string{"x-graphql-request-purpose", "fetch"},
		[2]string{"x-fb-friendly-name", S560FriendlyName},
		[2]string{"x-zero-f-device-id", profile.FamilyDeviceID},
		[2]string{"x-tigon-is-retry", "False"},
		[2]string{"x-zero-state", "unknown"},
		[2]string{"x-graphql-client-library", "graphservice"},
		[2]string{"x-fb-sim-hni", profile.Sim.HNI},
		[2]string{"content-type", "application/x-www-form-urlencoded"},
		[2]string{"x-fb-net-hni", profile.Sim.HNI},
		[2]string{"authorization", "OAuth " + S560OAuthToken},
		[2]string{"x-meta-zca", S560MetaZCA},
		[2]string{"app-scope-id-header", profile.DeviceID},
		[2]string{"x-fb-connection-type", profile.ConnType},
		[2]string{"x-meta-usdid", generateUSDID()},
		[2]string{"accept-encoding", "gzip, deflate"},
		[2]string{"x-fb-http-engine", "Tigon/Liger"},
		[2]string{"x-fb-client-ip", "True"},
		[2]string{"x-fb-server-cluster", "True"},
		[2]string{"x-fb-conn-uuid-client", connUUID()},
	)
	return h
}

// randomHex32 sinh 32-char lowercase hex (placeholder x-zero-eh cho initial reg).
func randomHex32() string {
	r := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	const hex = "0123456789abcdef"
	b := make([]byte, 32)
	for i := range b {
		b[i] = hex[r.Intn(16)]
	}
	return string(b)
}

// generateUSDID — x-meta-usdid: ECDSA P-256 signed "{uuid}.{unix_ts}" → base64url.
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

// connUUID — x-fb-conn-uuid-client: base64(16 raw UUID bytes).
func connUUID() string {
	id := uuid.New()
	return base64.StdEncoding.EncodeToString(id[:])
}
