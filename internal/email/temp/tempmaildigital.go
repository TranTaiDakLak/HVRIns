// tempmaildigital.go — tempmail.digital service (Symfony + PHPSESSID, HTML scrape)
//
// Flow (xác nhận live qua agent 2026-06-19, OTP đọc đúng):
//  1. GET / (cookie jar) → Set-Cookie PHPSESSID + auto-assign địa chỉ trong HTML:
//     <input id="email-display-input" value="firstname.lastname@mofagrac.online">
//  2. GET /inbox/has-new?known_count=0 → {"has_new":bool,"count":N,"latest_at":...}
//  3. Khi có mail: GET / lại (homepage nhúng list) → scrape href="/en/email/{uuid}"
//  4. GET /en/email/{uuid} → HTML body → ExtractCode
//
// KHÔNG cần key/login/captcha. PHPSESSID gắn inbox với session. 1 domain (mofagrac.online).
package temp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const (
	tempMailDigitalBaseURL = "https://tempmail.digital"
	tempMailDigitalUA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

var (
	// id="email-display-input" ... value="addr"  (id trước value)
	tmDigitalEmailRe1 = regexp.MustCompile(`id="email-display-input"[^>]*value="([^"]+@[^"]+)"`)
	// value="addr" ... id="email-display-input"  (value trước id)
	tmDigitalEmailRe2 = regexp.MustCompile(`value="([^"]+@[^"]+)"[^>]*id="email-display-input"`)
	// fallback: bất kỳ địa chỉ email nào trong value=
	tmDigitalEmailRe3 = regexp.MustCompile(`value="([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})"`)
	// link tới từng message: /en/email/{uuid}
	tmDigitalMsgRe = regexp.MustCompile(`/en/email/([a-zA-Z0-9\-]+)`)
)

// TempMailDigital implements email.Service cho tempmail.digital.
type TempMailDigital struct {
	client *http.Client
	email  string
}

// NewTempMailDigital tạo TempMailDigital service.
func NewTempMailDigital(proxyStr string) *TempMailDigital {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	c.Jar = jar
	return &TempMailDigital{client: c}
}

func (t *TempMailDigital) getHTML(ctx context.Context, path string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tempMailDigitalBaseURL+path, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", tempMailDigitalUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Referer", tempMailDigitalBaseURL+"/")
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 512*1024)
	return body, resp.StatusCode, nil
}

// CreateEmail: GET / → PHPSESSID + địa chỉ auto-assign.
func (t *TempMailDigital) CreateEmail(ctx context.Context) (string, error) {
	html, status, err := t.getHTML(ctx, "/")
	if err != nil {
		return "", fmt.Errorf("tempmaildigital init: %w", err)
	}
	email := extractTmDigitalEmail(string(html))
	if email == "" {
		return "", fmt.Errorf("tempmaildigital init: không tìm được địa chỉ (HTTP %d)", status)
	}
	t.email = email
	return t.email, nil
}

func extractTmDigitalEmail(html string) string {
	for _, re := range []*regexp.Regexp{tmDigitalEmailRe1, tmDigitalEmailRe2, tmDigitalEmailRe3} {
		if m := re.FindStringSubmatch(html); len(m) >= 2 && strings.Contains(m[1], "@") {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

// GetEmail trả về địa chỉ đã tạo.
func (t *TempMailDigital) GetEmail() string { return t.email }

// Close no-op.
func (t *TempMailDigital) Close() {}

// WaitForCode poll OTP từ inbox.
func (t *TempMailDigital) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmaildigital: chưa tạo email")
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
	return "", fmt.Errorf("tempmaildigital: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempMailDigital) pollOnce(ctx context.Context) (string, error) {
	// GET / → homepage nhúng list message khi có mail.
	html, _, err := t.getHTML(ctx, "/")
	if err != nil {
		return "", err
	}
	// Thử ExtractCode trực tiếp trên homepage (subject thường hiển thị trong list).
	if code := ExtractCode(string(html)); code != "" {
		return code, nil
	}
	// Scrape link từng message rồi fetch nội dung đầy đủ.
	seen := map[string]bool{}
	for _, m := range tmDigitalMsgRe.FindAllStringSubmatch(string(html), -1) {
		uuid := m[1]
		if uuid == "" || seen[uuid] {
			continue
		}
		seen[uuid] = true
		content, _, err := t.getHTML(ctx, "/en/email/"+uuid)
		if err != nil {
			continue
		}
		if code := ExtractCode(string(content)); code != "" {
			return code, nil
		}
	}
	return "", nil
}
