// mailermnx.go — Mailer.mnx-family.com service (client-side email, inline content)
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const mailerMnxBaseURL = "https://mailer.mnx-family.com"

var mailerMnxDomainRe = regexp.MustCompile(`<option value="([^"]+)">`)
var mailerMnxFallbackDomains = []string{"mnx-family.com"}

// MailerMnx implements email.Service cho mailer.mnx-family.com
type MailerMnx struct {
	client  *http.Client
	domains []string
	email   string
}

// NewMailerMnx tạo MailerMnx service.
func NewMailerMnx(proxyStr string) *MailerMnx {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &MailerMnx{client: client, domains: mailerMnxFallbackDomains}
}

// CreateEmail lấy danh sách domain và sinh địa chỉ email client-side.
func (m *MailerMnx) CreateEmail(ctx context.Context) (string, error) {
	if err := m.fetchDomains(ctx); err != nil {
		// use fallback
	}
	domain := m.domains[0]
	m.email = realisticEmail(domain)
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *MailerMnx) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *MailerMnx) Close() {}

// fetchDomains lấy danh sách domain từ homepage.
func (m *MailerMnx) fetchDomains(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", mailerMnxBaseURL+"/", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	matches := mailerMnxDomainRe.FindAllSubmatch(body, -1)
	var domains []string
	for _, match := range matches {
		if len(match) > 1 {
			domains = append(domains, string(match[1]))
		}
	}
	if len(domains) > 0 {
		m.domains = domains
	}
	return nil
}

// WaitForCode poll OTP từ mailer.mnx-family.com inbox.
func (m *MailerMnx) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" {
		return "", fmt.Errorf("mailermnx: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := m.pollOnce(ctx)
		if err == nil && code != "" {
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

	return "", fmt.Errorf("mailermnx: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy danh sách mail và extract code từ nội dung inline.
func (m *MailerMnx) pollOnce(ctx context.Context) (string, error) {
	inboxURL := mailerMnxBaseURL + "/pesan?email=" + url.QueryEscape(m.email)
	req, err := http.NewRequestWithContext(ctx, "GET", inboxURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", mailerMnxBaseURL+"/")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)

	// API trả data: [] khi có mail, data: {} khi inbox rỗng → dùng RawMessage
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", nil // non-JSON response → coi như inbox rỗng
	}

	var msgs []struct {
		ID      string `json:"id"`
		From    string `json:"from"`
		Sender  string `json:"sender"`
		Subject string `json:"subject"`
		HTML    string `json:"html"`
		Text    string `json:"text"`
		Body    string `json:"body"`
	}
	// Bỏ qua lỗi nếu data là {} thay vì []
	_ = json.Unmarshal(envelope.Data, &msgs)

	for _, msg := range msgs {
		content := msg.HTML
		if content == "" {
			content = msg.Text
		}
		if content == "" {
			content = msg.Body
		}
		if content == "" {
			content = msg.Subject
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}
