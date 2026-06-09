// byomde.go — Byom.de service (client-side email, inline content)
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const byomDeBaseURL = "https://api.byom.de"

// ByomDe implements email.Service cho byom.de
type ByomDe struct {
	client    *http.Client
	localPart string
	email     string
}

// NewByomDe tạo ByomDe service.
func NewByomDe(proxyStr string) *ByomDe {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &ByomDe{client: client}
}

// CreateEmail sinh địa chỉ email client-side.
func (m *ByomDe) CreateEmail(_ context.Context) (string, error) {
	m.localPart = realisticLocalPart()
	m.email = m.localPart + "@byom.de"
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *ByomDe) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *ByomDe) Close() {}

// WaitForCode poll OTP từ byom.de inbox.
func (m *ByomDe) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" {
		return "", fmt.Errorf("byomde: chưa tạo email")
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

	return "", fmt.Errorf("byomde: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy inbox và extract code từ nội dung inline.
func (m *ByomDe) pollOnce(ctx context.Context) (string, error) {
	inboxURL := byomDeBaseURL + "/mails/" + m.localPart
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
	body, _ := httpx.ReadBody(resp.Body, 128*1024)

	var mails []struct {
		HTML string `json:"html"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &mails); err != nil {
		return "", fmt.Errorf("byomde inbox parse: %w", err)
	}

	for _, mail := range mails {
		content := mail.HTML
		if content == "" {
			content = mail.Text
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}
