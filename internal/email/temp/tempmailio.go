// tempmailio.go — temp-mail.io / temp-mail.org service (v3 public API).
//
// Flow (reverse-engineer từ api.internal.temp-mail.io, xác nhận 2026-06-18):
//   1. POST /api/v3/email/new  body {"name","domain"} (hoặc {} = server random)
//      → {"email":"...","token":"..."}
//   2. GET  /api/v3/email/{addr}/messages → [{id,subject,body_text,body_html}]
//
// KHÔNG cần API key, KHÔNG Cloudflare — dùng http.Client thường qua proxy pool.
// Domain list lấy qua GET /api/v3/domains (public).
package temp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const tempMailIoAPIBase = "https://api.internal.temp-mail.io/api/v3"
const tempMailIoUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"

// tempMailIoKnownDomains — fallback nếu API /domains fail (cập nhật 2026-06-18).
var tempMailIoKnownDomains = []string{
	"bltiwd.com", "wnbaldwy.com", "bwmyga.com", "ozsaip.com",
}

// TempMailIo implements email.Service cho temp-mail.io.
type TempMailIo struct {
	client        *http.Client
	email         string
	pinnedDomains []string // domain user chọn; rỗng = server random
}

// ParseTempMailIoDomains tách chuỗi "a.com, b.com" → []string{"a.com","b.com"}.
func ParseTempMailIoDomains(raw string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' }) {
		if d := strings.TrimPrefix(strings.TrimSpace(p), "@"); d != "" {
			out = append(out, d)
		}
	}
	return out
}

// NewTempMailIo tạo TempMailIo service. pinnedDomains = domain user chọn; rỗng = server random.
func NewTempMailIo(proxyStr string, pinnedDomains []string) *TempMailIo {
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	return &TempMailIo{client: c, pinnedDomains: pinnedDomains}
}

func (t *TempMailIo) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", tempMailIoUA)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://temp-mail.io")
	req.Header.Set("Referer", "https://temp-mail.io/")
}

// CreateEmail tạo email qua POST /email/new. Nếu có pinnedDomains thì random-pick 1.
func (t *TempMailIo) CreateEmail(ctx context.Context) (string, error) {
	payload := map[string]string{}
	if len(t.pinnedDomains) > 0 {
		payload["name"] = realisticLocalPart()
		payload["domain"] = t.pinnedDomains[rand.Intn(len(t.pinnedDomains))]
	}
	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", tempMailIoAPIBase+"/email/new", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	t.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tempmailio create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	var result struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Email == "" {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return "", fmt.Errorf("tempmailio create: no email (HTTP %d) snippet=%q", resp.StatusCode, snippet)
	}
	t.email = result.Email
	return t.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (t *TempMailIo) GetEmail() string { return t.email }

// Close no-op.
func (t *TempMailIo) Close() {}

// WaitForCode poll OTP từ inbox.
func (t *TempMailIo) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmailio: chưa tạo email")
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if code, _ := t.pollOnce(ctx); code != "" {
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
	return "", fmt.Errorf("tempmailio: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempMailIo) pollOnce(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/email/%s/messages", tempMailIoAPIBase, t.email)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	t.setHeaders(req)
	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	var msgs []struct {
		Subject  string `json:"subject"`
		BodyText string `json:"body_text"`
		BodyHTML string `json:"body_html"`
	}
	if err := json.Unmarshal(body, &msgs); err != nil {
		return "", nil
	}
	for _, m := range msgs {
		if code := ExtractCode(m.Subject); code != "" {
			return code, nil
		}
		if code := ExtractCode(m.BodyText); code != "" {
			return code, nil
		}
		if code := ExtractCode(m.BodyHTML); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// TempMailIoDomainsResult là kết quả trả về cho FetchTempMailIoDomains.
type TempMailIoDomainsResult struct {
	Domains []string `json:"domains"`
}

// FetchTempMailIoDomains gọi GET /domains. KHÔNG cần key. Fallback hardcoded nếu API fail.
func FetchTempMailIoDomains(ctx context.Context) (*TempMailIoDomainsResult, error) {
	client := proxy.CreateClient("", 15*time.Second)
	req, err := http.NewRequestWithContext(ctx, "GET", tempMailIoAPIBase+"/domains", nil)
	if err != nil {
		return &TempMailIoDomainsResult{Domains: tempMailIoKnownDomains}, nil
	}
	req.Header.Set("User-Agent", tempMailIoUA)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://temp-mail.io/")

	resp, err := client.Do(req)
	if err != nil {
		return &TempMailIoDomainsResult{Domains: tempMailIoKnownDomains}, nil
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var result struct {
		Domains []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"domains"`
	}
	if err := json.Unmarshal(body, &result); err != nil || len(result.Domains) == 0 {
		return &TempMailIoDomainsResult{Domains: tempMailIoKnownDomains}, nil
	}
	var out []string
	for _, d := range result.Domains {
		if d.Name != "" {
			out = append(out, d.Name)
		}
	}
	if len(out) == 0 {
		return &TempMailIoDomainsResult{Domains: tempMailIoKnownDomains}, nil
	}
	return &TempMailIoDomainsResult{Domains: out}, nil
}
