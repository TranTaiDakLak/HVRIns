// session.go — TLS session cho IG iOS (Safari/iOS profile) + POST graphql_www.
//
// IG reg iOS dùng HTTP qua i.instagram.com với header x-ig-*. Response nén zstd.
// Tái dùng proxy layer của dự án (internal/proxy.FormatProxyURL).
package igcore

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/klauspost/compress/zstd"

	"HVRIns/internal/proxy"
)

const (
	igHost      = "https://i.instagram.com"
	graphqlPath = igHost + "/graphql_www"
	qeSyncURL   = igHost + "/api/v1/qe/sync/"
)

// sharedZstdDecoder — 1 decoder DÙNG CHUNG cho cả package. DecodeAll an toàn gọi
// đồng thời. Trước đây mỗi session tạo decoder riêng (spawn ~GOMAXPROCS goroutine)
// và KHÔNG Close → leak goroutine theo số session (country-detect tạo session phụ
// mỗi account). Dùng chung → tổng goroutine bị giới hạn.
var sharedZstdDecoder, _ = zstd.NewReader(nil)

type igSession struct {
	client tls_client.HttpClient
	zr     *zstd.Decoder
}

func newIGSession(proxyStr string) (*igSession, error) {
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(60),
		tls_client.WithClientProfile(profiles.Safari_IOS_15_6),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		tls_client.WithInsecureSkipVerify(),
		tls_client.WithNotFollowRedirects(),
	}
	if proxyStr != "" {
		if f := proxy.FormatProxyURL(proxyStr); f != "" {
			opts = append(opts, tls_client.WithProxyUrl(f))
		}
	}
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return nil, fmt.Errorf("create tls client: %w", err)
	}
	return &igSession{client: c, zr: sharedZstdDecoder}, nil
}

// post gửi form body với header order chuẩn, decode response (zstd/gzip/plain).
// Trả về (bodyString, responseHeaders, error).
func (s *igSession) post(ctx context.Context, url, body string, headers [][2]string) (string, fhttp.Header, error) {
	req, err := fhttp.NewRequestWithContext(ctx, "POST", url, strings.NewReader(body))
	if err != nil {
		return "", nil, err
	}
	order := make([]string, 0, len(headers))
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
		order = append(order, kv[0])
	}
	req.Header[fhttp.HeaderOrderKey] = order

	resp, err := s.client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return "", resp.Header, err
	}
	dec := s.decode(resp.Header.Get("Content-Encoding"), raw)
	if resp.StatusCode >= 400 {
		return dec, resp.Header, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return dec, resp.Header, nil
}

func (s *igSession) decode(enc string, raw []byte) string {
	switch strings.ToLower(strings.TrimSpace(enc)) {
	case "zstd":
		if out, err := s.zr.DecodeAll(raw, nil); err == nil {
			return string(out)
		}
	case "gzip":
		// hiếm; thử zstd reader fail thì trả raw
	}
	// thử zstd anyway (IG hay trả zstd kể cả khi header thiếu)
	if out, err := s.zr.DecodeAll(raw, nil); err == nil && len(out) > 0 {
		return string(out)
	}
	return string(raw)
}

// qeSync gọi /api/v1/qe/sync/ để lấy password encryption key + X-MID.
// Trả về (keyID, pubKeyPEM, xMID, error).
func (s *igSession) qeSync(ctx context.Context, p *igProfile) (string, string, string, error) {
	form := "id=" + p.DeviceID + "&experiments=ig_android_device_detection_info_upload"
	headers := [][2]string{
		{"user-agent", p.UserAgent},
		{"accept-encoding", "gzip"},
		{"accept", "*/*"},
		{"x-ig-app-id", igAppID},
		{"x-ig-capabilities", "36r/F/8="},
		{"x-ig-device-id", p.DeviceID},
		{"x-ig-family-device-id", p.FamilyDeviceID},
		{"content-type", "application/x-www-form-urlencoded; charset=UTF-8"},
	}
	req, err := fhttp.NewRequestWithContext(ctx, "POST", qeSyncURL, strings.NewReader(form))
	if err != nil {
		return "", "", "", err
	}
	order := make([]string, 0, len(headers))
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
		order = append(order, kv[0])
	}
	req.Header[fhttp.HeaderOrderKey] = order
	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))

	keyID := resp.Header.Get("ig-set-password-encryption-key-id")
	pubKey := resp.Header.Get("ig-set-password-encryption-pub-key")
	xmid := resp.Header.Get("ig-set-x-mid")
	if keyID == "" || pubKey == "" {
		return "", "", xmid, fmt.Errorf("qe/sync không trả key (HTTP %d)", resp.StatusCode)
	}
	return keyID, pubKey, xmid, nil
}

// newChromeCheckSession tạo TLS session với Chrome 133 fingerprint.
// Dùng cho CheckLiveByCheckerCookie vì web_profile_info yêu cầu Chrome TLS.
func newChromeCheckSession(proxyStr string) (*igSession, error) {
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		tls_client.WithInsecureSkipVerify(),
		tls_client.WithNotFollowRedirects(),
	}
	if proxyStr != "" {
		if f := proxy.FormatProxyURL(proxyStr); f != "" {
			opts = append(opts, tls_client.WithProxyUrl(f))
		}
	}
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return nil, fmt.Errorf("create chrome tls client: %w", err)
	}
	return &igSession{client: c, zr: sharedZstdDecoder}, nil
}

var _ = bytes.MinRead
