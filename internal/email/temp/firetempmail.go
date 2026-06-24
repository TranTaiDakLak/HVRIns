// firetempmail.go — FireTempMail.com service (client-side email, inline content)
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const fireTempMailBaseURL = "https://mail.firetempmail.com"

var fireTempMailFallbackDomains = []string{
	"offredaily.sa.com",
	"ctm.edu.pl",
	"jobsdeforyou.sa.com",
}

// FireTempMail implements email.Service cho firetempmail.com
type FireTempMail struct {
	client *http.Client
	email  string
	domain string // preferred domain; empty = use index 0
}

// NewFireTempMail tạo FireTempMail service.
func NewFireTempMail(proxyStr string) *FireTempMail {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &FireTempMail{client: client}
}

// NewFireTempMailWithDomain tạo FireTempMail service với domain cụ thể.
func NewFireTempMailWithDomain(proxyStr, domain string) *FireTempMail {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &FireTempMail{client: client, domain: domain}
}

// CreateEmail sinh địa chỉ email client-side.
func (m *FireTempMail) CreateEmail(_ context.Context) (string, error) {
	domain := m.domain
	if domain == "" {
		domain = fireTempMailFallbackDomains[0]
	}
	m.email = realisticEmail(domain)
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *FireTempMail) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *FireTempMail) Close() {}

// WaitForCode poll OTP từ firetempmail.com inbox.
func (m *FireTempMail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" {
		return "", fmt.Errorf("firetempmail: chưa tạo email")
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

	return "", fmt.Errorf("firetempmail: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy danh sách mail và extract code từ nội dung inline.
func (m *FireTempMail) pollOnce(ctx context.Context) (string, error) {
	inboxURL := fireTempMailBaseURL + "/mail/get?address=" + url.QueryEscape(m.email)
	req, err := http.NewRequestWithContext(ctx, "GET", inboxURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://firetempmail.com/")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)

	var result struct {
		Mails []struct {
			Sender    string `json:"sender"`
			From      string `json:"from"`
			Subject   string `json:"subject"`
			Body      string `json:"body"`
			HTML      string `json:"html"`
			Preview   string `json:"preview"`
			Recipient string `json:"recipient"`
			Suffix    string `json:"suffix"`
		} `json:"mails"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("firetempmail inbox parse: %w", err)
	}

	for _, mail := range result.Mails {
		content := mail.Body
		if content == "" {
			content = mail.HTML
		}
		if content == "" {
			content = mail.Preview
		}
		if content == "" {
			content = mail.Subject
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}
