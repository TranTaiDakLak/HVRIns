// buslink24.go — Buslink24.com service (client-side email, inline content)
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

const buslink24BaseURL = "https://buslink24.com"

// Buslink24 implements email.Service cho buslink24.com
type Buslink24 struct {
	client *http.Client
	email  string
}

// NewBuslink24 tạo Buslink24 service.
func NewBuslink24(proxyStr string) *Buslink24 {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &Buslink24{client: client}
}

// CreateEmail sinh địa chỉ email client-side (domain cố định buslink24.com).
func (m *Buslink24) CreateEmail(_ context.Context) (string, error) {
	m.email = realisticEmail("buslink24.com")
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *Buslink24) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *Buslink24) Close() {}

// WaitForCode poll OTP từ buslink24.com inbox.
func (m *Buslink24) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" {
		return "", fmt.Errorf("buslink24: chưa tạo email")
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

	return "", fmt.Errorf("buslink24: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy danh sách mail và extract code từ nội dung inline.
func (m *Buslink24) pollOnce(ctx context.Context) (string, error) {
	inboxURL := buslink24BaseURL + "/api/mail/messages?to=" + url.QueryEscape(m.email)
	req, err := http.NewRequestWithContext(ctx, "GET", inboxURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://buslink24.com/")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)

	var msgs []struct {
		ID      string `json:"id"`
		Subject string `json:"subject"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &msgs); err != nil {
		return "", nil // API trả HTML/error page → coi như inbox rỗng
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
