package iosmess

import (
	"io"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/google/uuid"
)

// newClient — tls-client Safari iOS + proxy (format host:port:user:pass / http://...).
func newClient(proxy string, timeoutSec int) (tls_client.HttpClient, error) {
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(timeoutSec),
		tls_client.WithClientProfile(profiles.Safari_IOS_17_0),
		tls_client.WithInsecureSkipVerify(),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		tls_client.WithNotFollowRedirects(),
	}
	if p := formatProxy(proxy); p != "" {
		opts = append(opts, tls_client.WithProxyUrl(p))
	}
	return tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
}

// sendBloks — POST 1 bước Bloks (app token + headers Pando), trả (status, body).
func sendBloks(client tls_client.HttpClient, body, friendly, deviceID, ua string) (int, string, error) {
	return sendBloksWithToken(client, body, friendly, deviceID, ua, "")
}

// sendBloksWithToken — giống sendBloks nhưng nhận token override.
// token="" → dùng appToken (app-token mặc định).
// token="EAAG..." → dùng user token (cho ver flow login-first).
func sendBloksWithToken(client tls_client.HttpClient, body, friendly, deviceID, ua, token string) (int, string, error) {
	authToken := appToken
	if token != "" {
		authToken = token
	}
	req, _ := fhttp.NewRequest("POST", endpoint, strings.NewReader(body))
	hdr := [][2]string{
		{"user-agent", ua},
		{"x-graphql-client-library", "pando"},
		{"x-graphql-request-purpose", "fetch"},
		{"content-type", "application/x-www-form-urlencoded"},
		{"x-fb-friendly-name", friendly},
		{"authorization", "OAuth " + authToken},
		{"x-fb-request-analytics-tags", `{"network_tags":{"product":"437626316973788","request_category":"graphql","purpose":"fetch","retry_attempt":"0"}}`},
		{"x-meta-usdid-uuid", strings.ToUpper(uuid.New().String())},
		{"x-fb-rmd", "fail=Server:INVALID_MAP,Default:INVALID_MAP;v=;ip=;tkn=;reqTime=0;recvTime=0"},
		{"x-tigon-is-retry", "False"},
		{"x-fb-device-id", deviceID},
		{"x-fb-conn-uuid-client", strings.ReplaceAll(uuid.New().String(), "-", "")[:32]},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
		{"x-fb-http-engine", "Tigon/MNS/mvfst-mobile"},
	}
	order := make([]string, 0, len(hdr))
	for _, kv := range hdr {
		req.Header[kv[0]] = []string{kv[1]}
		order = append(order, kv[0])
	}
	req.Header[fhttp.HeaderOrderKey] = order
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw), nil
}

// formatProxy — host:port:user:pass / user:pass@host:port / http://... → http URL.
func formatProxy(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") || strings.HasPrefix(p, "socks5://") {
		return p
	}
	if strings.Contains(p, "@") {
		return "http://" + p
	}
	parts := strings.Split(p, ":")
	switch len(parts) {
	case 2:
		return "http://" + parts[0] + ":" + parts[1]
	case 4:
		return "http://" + parts[2] + ":" + parts[3] + "@" + parts[0] + ":" + parts[1]
	}
	return "http://" + p
}
