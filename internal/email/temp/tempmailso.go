// tempmailso.go — tempmail.so service (Chrome TLS fingerprint, no-PoW API 2026-05)
//
// Flow chuẩn (reverse-engineer từ utilities.js network()):
//   1. GET /us/ (Chrome TLS) → nhận tm_session cookie
//   2. GET /us/api/inbox?requestTime={ms}&lang=us  (KHÔNG có PoW/x param)
//      headers: Accept: application/json, Content-type: application/json, X-Inbox-Lifespan: 600
//   3. GET /us/api/inbox/messagehtmlbody/{id}?... đọc nội dung OTP
//
// LƯU Ý: tempmail.so có Cloudflare JS Detection. Từ IP datacenter sẽ bị
// "Action Not Allowed" (cần cf_clearance). Qua proxy RESIDENTIAL thường pass.
// Dùng bogdanfinn/tls-client với Chrome fingerprint để qua tầng JA3.
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const (
	tempMailSoBaseURL  = "https://tempmail.so"
	tempMailSoUA       = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	tempMailSoLifespan = 600
)

var tempMailSoEmailRegex = regexp.MustCompile(`"name":"([^"]+)"`)

// TempMailSo implements email.Service cho tempmail.so.
type TempMailSo struct {
	proxyStr string
	email    string
	client   tls_client.HttpClient // giữ session cookie giữa CreateEmail và WaitForCode
}

// NewTempMailSo tạo TempMailSo service.
func NewTempMailSo(proxyStr string) *TempMailSo {
	return &TempMailSo{proxyStr: proxyStr}
}

// newClient tạo tls-client Chrome fingerprint (bypass Cloudflare JA3).
func (t *TempMailSo) newClient() (tls_client.HttpClient, error) {
	chromeProfiles := []profiles.ClientProfile{
		profiles.Chrome_120, profiles.Chrome_124, profiles.Chrome_133,
	}
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(chromeProfiles[rand.Intn(len(chromeProfiles))]),
		tls_client.WithInsecureSkipVerify(),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
	}
	if t.proxyStr != "" {
		if formatted := proxy.FormatProxyURL(t.proxyStr); formatted != "" {
			opts = append(opts, tls_client.WithProxyUrl(formatted))
		}
	}
	return tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
}

// warmup: GET /us/ để Cloudflare set tm_session cookie.
func (t *TempMailSo) warmup(ctx context.Context, c tls_client.HttpClient) error {
	req, err := fhttp.NewRequest("GET", tempMailSoBaseURL+"/us/", nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	req.Header = fhttp.Header{
		"user-agent":      {tempMailSoUA},
		"accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8"},
		"accept-language": {"en-US,en;q=0.9"},
		"sec-fetch-dest":  {"document"},
		"sec-fetch-mode":  {"navigate"},
		"sec-fetch-site":  {"none"},
		fhttp.HeaderOrderKey: {"user-agent", "accept", "accept-language",
			"sec-fetch-dest", "sec-fetch-mode", "sec-fetch-site"},
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("tempmailso warmup: %w", err)
	}
	resp.Body.Close()
	return nil
}

// inboxCall GET /us/api/inbox — trả raw body + status.
func (t *TempMailSo) inboxCall(ctx context.Context, c tls_client.HttpClient) ([]byte, int, error) {
	now := time.Now().UnixMilli()
	url := fmt.Sprintf("%s/us/api/inbox?requestTime=%d&lang=us", tempMailSoBaseURL, now)
	req, err := fhttp.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	req = req.WithContext(ctx)
	// Headers CHÍNH XÁC theo utilities.js network()
	req.Header = fhttp.Header{
		"accept":           {"application/json"},
		"content-type":     {"application/json"},
		"x-inbox-lifespan": {fmt.Sprintf("%d", tempMailSoLifespan)},
		"user-agent":       {tempMailSoUA},
		"referer":          {tempMailSoBaseURL + "/us/"},
		fhttp.HeaderOrderKey: {"accept", "content-type", "x-inbox-lifespan", "user-agent", "referer"},
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("tempmailso inbox: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)
	return body, resp.StatusCode, nil
}

// CreateEmail: warmup → inbox call → parse email.
func (t *TempMailSo) CreateEmail(ctx context.Context) (string, error) {
	c, err := t.newClient()
	if err != nil {
		return "", fmt.Errorf("tempmailso: tạo client lỗi: %w", err)
	}
	t.client = c

	if err := t.warmup(ctx, c); err != nil {
		return "", err
	}
	// Đợi nhẹ như browser load page
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(1500 * time.Millisecond):
	}

	body, status, err := t.inboxCall(ctx, c)
	if err != nil {
		return "", err
	}
	if status == 429 {
		return "", fmt.Errorf("tempmailso: rate limited (429) — đổi proxy")
	}
	if status == 403 || strings.Contains(string(body), "Action Not Allowed") {
		return "", fmt.Errorf("tempmailso: Cloudflare chặn (cần proxy residential)")
	}

	// Parse email — regex hoặc JSON
	if m := tempMailSoEmailRegex.FindSubmatch(body); len(m) >= 2 {
		t.email = string(m[1])
		return t.email, nil
	}
	var result struct {
		Data struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &result) == nil && result.Data.Name != "" {
		t.email = result.Data.Name
		return t.email, nil
	}
	return "", fmt.Errorf("tempmailso: không parse được email — status=%d body=%.100s",
		status, strings.TrimSpace(string(body)))
}

// GetEmail trả về địa chỉ email đã tạo.
func (t *TempMailSo) GetEmail() string { return t.email }

// Close no-op.
func (t *TempMailSo) Close() {}

// WaitForCode poll OTP.
func (t *TempMailSo) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 3000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmailso: chưa tạo email")
	}
	c := t.client
	if c == nil {
		var err error
		c, err = t.newClient()
		if err != nil {
			return "", err
		}
		if err := t.warmup(ctx, c); err != nil {
			return "", err
		}
		t.client = c
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if code, _ := t.pollOnce(ctx, c); code != "" {
			return code, nil
		}
		if attempt < maxRetry-1 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			}
		}
	}
	return "", fmt.Errorf("tempmailso: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempMailSo) pollOnce(ctx context.Context, c tls_client.HttpClient) (string, error) {
	body, _, err := t.inboxCall(ctx, c)
	if err != nil {
		return "", err
	}
	var inbox struct {
		Data struct {
			Inbox []struct {
				ID      string `json:"id"`
				Subject string `json:"subject"`
			} `json:"inbox"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &inbox); err != nil {
		return "", nil
	}
	for _, msg := range inbox.Data.Inbox {
		content, _ := t.getMessage(ctx, c, msg.ID)
		if content == "" {
			content = msg.Subject
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

func (t *TempMailSo) getMessage(ctx context.Context, c tls_client.HttpClient, id string) (string, error) {
	now := time.Now().UnixMilli()
	url := fmt.Sprintf("%s/us/api/inbox/messagehtmlbody/%s?requestTime=%d&lang=us",
		tempMailSoBaseURL, id, now)
	req, err := fhttp.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req = req.WithContext(ctx)
	req.Header = fhttp.Header{
		"accept":           {"application/json"},
		"content-type":     {"application/json"},
		"x-inbox-lifespan": {fmt.Sprintf("%d", tempMailSoLifespan)},
		"user-agent":       {tempMailSoUA},
		"referer":          {tempMailSoBaseURL + "/us/"},
		fhttp.HeaderOrderKey: {"accept", "content-type", "x-inbox-lifespan", "user-agent", "referer"},
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	var result struct {
		Data struct {
			HTML string `json:"html"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.Data.HTML, nil
}
