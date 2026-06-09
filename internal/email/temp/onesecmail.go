// onesecmail.go — 1secmail.com service (client-side email, REST API)
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const oneSecMailBaseURL = "https://www.1secmail.com/api/v1/"

// OneSecMail implements email.Service cho 1secmail.com
type OneSecMail struct {
	client *http.Client
	user   string
	domain string
	email  string
}

// NewOneSecMail tạo OneSecMail service.
func NewOneSecMail(proxyStr string) *OneSecMail {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &OneSecMail{client: client}
}

// CreateEmail sinh địa chỉ email client-side.
func (m *OneSecMail) CreateEmail(_ context.Context) (string, error) {
	m.user = realisticLocalPart()
	m.domain = "1secmail.com"
	m.email = m.user + "@" + m.domain
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *OneSecMail) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *OneSecMail) Close() {}

// WaitForCode poll OTP từ 1secmail.com inbox.
func (m *OneSecMail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" {
		return "", fmt.Errorf("onesecmail: chưa tạo email")
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

	return "", fmt.Errorf("onesecmail: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy inbox và extract code.
func (m *OneSecMail) pollOnce(ctx context.Context) (string, error) {
	listURL := oneSecMailBaseURL + "?action=getMessages&login=" +
		url.QueryEscape(m.user) + "&domain=" + url.QueryEscape(m.domain)

	req, err := http.NewRequestWithContext(ctx, "GET", listURL, nil)
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

	var msgs []struct {
		ID   int64  `json:"id"`
		From string `json:"from"`
	}
	if err := json.Unmarshal(body, &msgs); err != nil {
		return "", nil // API trả HTML/error page → coi như inbox rỗng
	}

	for _, msg := range msgs {
		if !strings.Contains(msg.From, "facebookmail.com") && !strings.Contains(strings.ToLower(msg.From), "instagram") {
			continue
		}
		content, err := m.getMessage(ctx, msg.ID)
		if err != nil {
			continue
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// getMessage lấy nội dung email theo ID.
func (m *OneSecMail) getMessage(ctx context.Context, id int64) (string, error) {
	msgURL := fmt.Sprintf("%s?action=readMessage&login=%s&domain=%s&id=%d",
		oneSecMailBaseURL, url.QueryEscape(m.user), url.QueryEscape(m.domain), id)

	req, err := http.NewRequestWithContext(ctx, "GET", msgURL, nil)
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

	var msg struct {
		HTMLBody string `json:"htmlBody"`
		Body     string `json:"body"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", err
	}

	if msg.HTMLBody != "" {
		return msg.HTMLBody, nil
	}
	return msg.Body, nil
}
