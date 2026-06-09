// http.go — S399 HTTP transport + request headers.
// Khác s5xx: dùng Liger engine, header subset (không có x-fb-rmd / x-zero-eh / friendly-name graphql).
package s399

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/url"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"

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
		return nil, fmt.Errorf("create s399 tls client: %w", err)
	}
	return &session{client: c}, nil
}

func (s *session) addCookie(name, value string) {
	u, _ := url.Parse("https://b-graph.facebook.com")
	s.client.SetCookies(u, []*fhttp.Cookie{
		{Name: name, Value: value, Path: "/", Domain: ".facebook.com"},
	})
}

// postGzip — gzip-compresses body và POST.
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

// buildRegisterHeaders — headers cho POST /app/users (step 1).
// Không có authorization (chưa login); friendly-name=registerAccount.
func buildRegisterHeaders(profile S399Profile, ua string) [][2]string {
	h := [][2]string{
		{"host", "b-graph.facebook.com"},
		{"x-fb-connection-quality", "EXCELLENT"},
		{"x-fb-sim-hni", profile.Sim.HNI},
		{"x-fb-net-hni", profile.Sim.HNI},
		{"user-agent", ua},
		{"content-encoding", "gzip"},
		{"zero-rated", "0"},
		{"x-fb-connection-bandwidth", "52371339"},
		{"content-type", "application/x-www-form-urlencoded"},
		{"x-fb-connection-type", profile.ConnType},
		{"x-fb-device-group", profile.DeviceGroup},
		{"x-tigon-is-retry", "False"},
		{"x-fb-friendly-name", s399FriendlyReg},
		{"x-fb-request-analytics-tags", `{"network_tags":{"retry_attempt":"0"},"application_tags":"unknown"}`},
		{"priority", "u=3, i"},
		{"accept-encoding", "gzip, deflate"},
		{"x-fb-http-engine", "Liger"},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
	}
	if profile.MachineID != "" {
		h = append(h, [2]string{"x-fb-integrity-machine-id", profile.MachineID})
	}
	return h
}

// buildLoginHeaders — headers cho POST /auth/login (step 2).
// Có authorization "OAuth null" (đặc thù v399 sau register flow); friendly-name=authenticate.
func buildLoginHeaders(profile S399Profile, ua string) [][2]string {
	h := [][2]string{
		{"host", "b-graph.facebook.com"},
		{"authorization", "OAuth null"},
		{"x-fb-connection-quality", "EXCELLENT"},
		{"x-fb-sim-hni", profile.Sim.HNI},
		{"x-fb-net-hni", profile.Sim.HNI},
		{"user-agent", ua},
		{"content-encoding", "gzip"},
		{"zero-rated", "0"},
		{"x-fb-connection-bandwidth", "52371339"},
		{"content-type", "application/x-www-form-urlencoded"},
		{"x-fb-connection-type", profile.ConnType},
		{"x-fb-device-group", profile.DeviceGroup},
		{"x-tigon-is-retry", "False"},
		{"x-fb-friendly-name", s399FriendlyAuth},
		{"x-fb-request-analytics-tags", `{"network_tags":{"retry_attempt":"0"},"application_tags":"unknown"}`},
		{"priority", "u=3, i"},
		{"accept-encoding", "gzip, deflate"},
		{"x-fb-http-engine", "Liger"},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
	}
	if profile.MachineID != "" {
		h = append(h, [2]string{"x-fb-integrity-machine-id", profile.MachineID})
	}
	return h
}
