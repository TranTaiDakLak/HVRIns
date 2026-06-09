// http.go â€” S565 HTTP transport + request headers.
// KhÃ¡c s557: bá» appnetsession/tasos/qpl/network-props headers;
// thÃªm x-fb-rmd, x-zero-eh, x-zero-state; accept-encoding: gzip, deflate.
package appmessv3

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

// â”€â”€â”€ HTTP session â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

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
		return nil, fmt.Errorf("create appmv3 tls client: %w", err)
	}
	return &session{client: c}, nil
}

// postGzip gzip-compresses body vÃ  gá»­i POST.
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

// buildHeaders â€” S565 header format (captured traffic).
// KhÃ¡c s557: bá» network-props/appnet/tasos/qpl; thÃªm rmd/zero-eh/zero-state.
func buildHeaders(profile AppMV3Profile) [][2]string {
	// Messenger (Orca) create.account headers: APP token + content-encoding gzip
	// (postGzip nén body, BẮT BUỘC header này) + device-scope, khớp capture V4.
	analyticsTag := `{"network_tags":{"product":"` + appmv3Product + `","request_category":"graphql","purpose":"none","retry_attempt":"0"},"application_tags":"graphservice"}`
	h := [][2]string{
		{"x-fb-request-analytics-tags", analyticsTag},
		{"x-fb-rmd", "state=URL_ELIGIBLE"},
		{"priority", "u=3, i"},
		{"user-agent", profile.AppMV3UA},
		{"x-graphql-client-library", "graphservice"},
		{"content-type", "application/x-www-form-urlencoded"},
		{"content-encoding", "gzip"},
		{"x-zero-eh", randomHex32()},
		{"authorization", "OAuth " + appmv3AppToken},
		{"x-zero-state", "unknown"},
		{"x-zero-f-device-id", profile.FamilyDeviceID},
		{"app-scope-id-header", profile.DeviceID},
		{"x-fb-friendly-name", appmv3FriendlyName},
		{"x-fb-connection-type", "WIFI"},
		{"x-tigon-is-retry", "False"},
		{"accept-encoding", "gzip, deflate"},
		{"x-fb-http-engine", "Tigon/Liger"},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
	}
	// x-fb-integrity-machine-id khi có datr/machine_id warm (như s565).
	if profile.MachineID != "" {
		h = append(h, [2]string{"x-fb-integrity-machine-id", profile.MachineID})
	}
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

// generateUSDID â€” x-meta-usdid: ECDSA P-256 signed "{uuid}.{unix_ts}" â†’ base64url.
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

// connUUID â€” x-fb-conn-uuid-client: base64(16 raw UUID bytes).
func connUUID() string {
	id := uuid.New()
	return base64.StdEncoding.EncodeToString(id[:])
}
