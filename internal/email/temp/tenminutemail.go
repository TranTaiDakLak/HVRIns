// tenminutemail.go — 10MinuteMail service
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const tenMinuteMailBaseURL = "https://10minutemail.com"

// TenMinuteMail implements email.Service cho 10minutemail.com
type TenMinuteMail struct {
	client *http.Client
	email  string
}

// NewTenMinuteMail tạo TenMinuteMail service.
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewTenMinuteMail(proxyStr string) *TenMinuteMail {
	jar, _ := cookiejar.New(nil)
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	client.Jar = jar
	return &TenMinuteMail{client: client}
}

// CreateEmail lấy địa chỉ email từ 10minutemail.com/session/address.
func (t *TenMinuteMail) CreateEmail(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tenMinuteMailBaseURL+"/session/address", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("10minutemail create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	var result struct {
		Address string `json:"address"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Address == "" {
		return "", fmt.Errorf("10minutemail create: unexpected response: %s", string(body))
	}

	t.email = result.Address
	return t.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (t *TenMinuteMail) GetEmail() string { return t.email }

// Close cleanup resources.
func (t *TenMinuteMail) Close() {}

// WaitForCode poll OTP từ 10minutemail inbox.
func (t *TenMinuteMail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("10minutemail: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := t.pollOnce(ctx)
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

	return "", fmt.Errorf("10minutemail: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy inbox một lần và extract code.
func (t *TenMinuteMail) pollOnce(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tenMinuteMailBaseURL+"/messages/messagesAfter/0", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	var messages []struct {
		ID              string `json:"id"`
		Sender          string `json:"sender"`
		Subject         string `json:"subject"`
		BodyHtmlContent string `json:"bodyHtmlContent"`
		BodyPlainText   string `json:"bodyPlainText"`
	}
	if err := json.Unmarshal(body, &messages); err != nil {
		return "", fmt.Errorf("10minutemail inbox parse: %w", err)
	}

	for _, msg := range messages {
		content := msg.BodyHtmlContent
		if content == "" {
			content = msg.BodyPlainText
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}
