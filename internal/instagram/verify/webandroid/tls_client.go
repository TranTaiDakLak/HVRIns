// tls_client.go — TLS fingerprint-aware HTTP client cho VER WebAndroid.
//
// Mục đích: thay net/http (Go default TLS fingerprint) bằng bogdanfinn/tls-client
// (Chrome browser TLS fingerprint). FB anti-bot detect non-browser ngay từ TLS
// handshake (JA3) trước cả HTTP request — net/http JA3 không match Chrome thật
// → mọi request bị FB RST connection (EOF / forcibly closed) trước khi parse body.
package webandroid

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"

	"HVRIns/internal/proxy"
)

// chromeMajorRe extract Chrome major version từ UA string.
// "Mozilla/5.0 ... Chrome/141.0.7390.107 Mobile ..." → "141"
var chromeMajorRe = regexp.MustCompile(`Chrome/(\d+)`)

// extractChromeMajor parse Chrome major version từ UA, fallback "" nếu không tìm thấy.
func extractChromeMajor(ua string) string {
	m := chromeMajorRe.FindStringSubmatch(ua)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// pickChromeTLSProfileVer chọn TLS profile gần nhất với Chrome major version.
// tls-client v1.14 có 5 Chrome profiles: 120, 124, 133, 144, 146.
func pickChromeTLSProfileVer(chromeMajor string) profiles.ClientProfile {
	if chromeMajor == "" {
		return profiles.Chrome_146
	}
	major := 0
	for _, c := range chromeMajor {
		if c < '0' || c > '9' {
			break
		}
		major = major*10 + int(c-'0')
	}
	switch {
	case major >= 146:
		return profiles.Chrome_146
	case major >= 144:
		return profiles.Chrome_144
	case major >= 133:
		return profiles.Chrome_133
	case major >= 124:
		return profiles.Chrome_124
	case major >= 120:
		return profiles.Chrome_120
	case major > 0:
		return profiles.Chrome_120
	default:
		return profiles.Chrome_146
	}
}

// createTLSClient tạo tls-client với Chrome profile match Chrome version trong UA.
// Cookie jar để giữ session cookies giữa các request (cần thiết cho m.facebook.com flow).
// Timeout 30s: confirmation_cliff endpoint chậm 5-20s, qua proxy có thể 25s+.
func createTLSClient(proxyStr, chromeMajor string) (tls_client.HttpClient, error) {
	jar := tls_client.NewCookieJar()
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(pickChromeTLSProfileVer(chromeMajor)),
		tls_client.WithCookieJar(jar),
		tls_client.WithInsecureSkipVerify(),
	}
	if proxyStr != "" {
		if formatted := proxy.FormatProxyURL(proxyStr); formatted != "" {
			opts = append(opts, tls_client.WithProxyUrl(formatted))
		}
	}
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return nil, fmt.Errorf("create tls client: %w", err)
	}
	return c, nil
}

// applyHeadersTLS set headers vào fhttp.Request với header order preservation.
func applyHeadersTLS(req *fhttp.Request, headers http.Header, order []string) {
	for k, vals := range headers {
		req.Header[k] = append([]string(nil), vals...)
	}
	if len(order) > 0 {
		req.Header[fhttp.HeaderOrderKey] = order
	}
}

// doGetTLS thực hiện GET qua tls-client, trả về (body, finalURL, error).
func doGetTLS(ctx context.Context, client tls_client.HttpClient, targetURL string, headers http.Header, headerOrder []string) (string, string, error) {
	req, err := fhttp.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", "", err
	}
	applyHeadersTLS(req, headers, headerOrder)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(io.LimitReader(resp.Body, 3<<20))
	body := prependIntegritySentinelTLS(resp.Header, string(b))

	finalURL := targetURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}
	return body, finalURL, nil
}

// doPostTLS thực hiện POST form-urlencoded qua tls-client.
func doPostTLS(ctx context.Context, client tls_client.HttpClient, targetURL, formBody string, headers http.Header, headerOrder []string) (string, error) {
	req, err := fhttp.NewRequestWithContext(ctx, "POST", targetURL, strings.NewReader(formBody))
	if err != nil {
		return "", err
	}
	if headers.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	applyHeadersTLS(req, headers, headerOrder)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(io.LimitReader(resp.Body, 3<<20))
	return prependIntegritySentinelTLS(resp.Header, string(b)), nil
}

// prependIntegritySentinelTLS — variant của prependIntegritySentinel cho fhttp.Header.
func prependIntegritySentinelTLS(h fhttp.Header, body string) string {
	integrity := h.Get("X-Fb-Integrity-Required")
	if integrity != "" && strings.Contains(strings.ToLower(integrity), "checkpoint") {
		return `{"error":{"code":459,"message":"checkpointed"}}` + body
	}
	if h.Get("X-Fb-Integrity-Requires-Login") != "" {
		return `{"error":{"message":"checkpointed"}}` + body
	}
	if h.Get("X-Fb-Integrity-Enrollment") != "" {
		return `{"error":{"message":"checkpointed"}}` + body
	}
	return body
}

// seedCookieStringTLS parse cookie string "k1=v1; k2=v2" và seed vào jar.
func seedCookieStringTLS(client tls_client.HttpClient, cookieStr string) {
	if cookieStr == "" {
		return
	}
	jar := client.GetCookieJar()
	fbURL, _ := url.Parse("https://m.facebook.com")
	var cookies []*fhttp.Cookie
	for _, part := range strings.Split(cookieStr, ";") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		name := strings.TrimSpace(kv[0])
		if name == "" {
			continue
		}
		cookies = append(cookies, &fhttp.Cookie{Name: name, Value: strings.TrimSpace(kv[1])})
	}
	if len(cookies) > 0 {
		jar.SetCookies(fbURL, cookies)
	}
}

// addCookieTLS thêm 1 cookie key-value vào jar.
func addCookieTLS(client tls_client.HttpClient, name, value string) {
	jar := client.GetCookieJar()
	fbURL, _ := url.Parse("https://m.facebook.com")
	jar.SetCookies(fbURL, []*fhttp.Cookie{{Name: name, Value: value}})
}
