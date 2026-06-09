// mailymg.go — Mailymg.com service (client-side email, GET inbox API)
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

const mailymgBaseURL = "https://mailymg.com"

// Mailymg implements email.Service cho mailymg.com
type Mailymg struct {
	client *http.Client
	email  string
}

// NewMailymg tạo Mailymg service.
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewMailymg(proxyStr string) *Mailymg {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &Mailymg{client: client}
}

// CreateEmail sinh địa chỉ email client-side (domain mailymg.com).
func (m *Mailymg) CreateEmail(_ context.Context) (string, error) {
	m.email = realisticEmail("mailymg.com")
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *Mailymg) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *Mailymg) Close() {}

// WaitForCode poll OTP từ mailymg.com inbox.
func (m *Mailymg) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" {
		return "", fmt.Errorf("mailymg: chưa tạo email")
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

	return "", fmt.Errorf("mailymg: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy danh sách mail và extract code.
func (m *Mailymg) pollOnce(ctx context.Context) (string, error) {
	inboxURL := mailymgBaseURL + "/api/mail/messages?to=" + url.QueryEscape(m.email)
	req, err := http.NewRequestWithContext(ctx, "GET", inboxURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	// Response: [{id, subject, message}]
	var msgs []struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &msgs); err != nil {
		return "", fmt.Errorf("mailymg inbox parse: %w", err)
	}

	for _, msg := range msgs {
		content := msg.Message
		if content == "" {
			content = msg.Subject
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}
