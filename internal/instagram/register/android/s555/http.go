// http.go — S555 HTTP transport + request headers.
// Dùng cùng format API/headers/body với s557 — chỉ khác FB app version trong UA + bloks_versioning_id.
package s555

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
		return nil, fmt.Errorf("create s555 tls client: %w", err)
	}
	return &session{client: c}, nil
}

// postGzip gzip-compresses body và gửi POST — 555 traffic gửi gzip (content-encoding: gzip).
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

// buildHeaders — 555 header format (captured traffic).
func buildHeaders(profile S555Profile) [][2]string {
	analyticsTag := `{"network_tags":{"product":"350685531728","purpose":"fetch","request_category":"graphql","retry_attempt":"0"},"application_tags":"graphservice"}`
	qplJSON := `{"schema_version":"v3","inprogress_qpls":[],"snapshot_attributes":{}}`

	h := [][2]string{
		{"x-fb-request-analytics-tags", analyticsTag},
		{"x-fb-network-properties", "Wifi;Validated;"},
		{"content-encoding", "gzip"},
		{"user-agent", profile.S555UA},
		{"x-graphql-request-purpose", "fetch"},
		{"x-fb-friendly-name", s555FriendlyName},
		{"x-zero-f-device-id", profile.FamilyDeviceID},
		{"x-fb-device-group", profile.DeviceGroup},
		{"x-graphql-client-library", "graphservice"},
		{"x-fb-appnetsession-sid", profile.AppnetSID},
		{"x-fb-sim-hni", profile.Sim.HNI},
		{"content-type", "application/x-www-form-urlencoded"},
		{"x-fb-net-hni", profile.Sim.HNI},
		{"x-fb-appnetsession-nid", profile.AppnetNID},
		{"x-meta-tasos-tlbwe-config", "quic_transport_bwe:config_33"},
		{"x-meta-zca", s555MetaZCA},
		{"app-scope-id-header", profile.DeviceID},
		{"x-fb-connection-type", profile.ConnType},
		{"authorization", "OAuth " + s555OAuthToken},
		{"x-meta-usdid", generateUSDID()},
		{"priority", "u=0"},
		{"x-fb-qpl-active-flows-json", qplJSON},
		{"x-meta-enable-tasos-ss-bwe", "1"},
		{"x-tigon-is-retry", "False"},
		{"accept-encoding", "zstd, gzip, deflate"},
		{"x-fb-http-engine", "Tigon/Liger"},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
		{"x-fb-conn-uuid-client", connUUID()},
	}
	if profile.MachineID != "" {
		insert := [][2]string{{"x-fb-integrity-machine-id", profile.MachineID}}
		newH := make([][2]string, 0, len(h)+1)
		for _, kv := range h {
			newH = append(newH, kv)
			if kv[0] == "x-zero-f-device-id" {
				newH = append(newH, insert...)
			}
		}
		h = newH
	}
	return h
}

// generateUSDID — x-meta-usdid: ECDSA P-256 signed "{uuid}.{unix_ts}" -> base64url.
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

// connUUID — x-fb-conn-uuid-client: base64(16 raw UUID bytes) — 555 format.
func connUUID() string {
	id := uuid.New()
	return base64.StdEncoding.EncodeToString(id[:])
}
