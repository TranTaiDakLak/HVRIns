// tempforward.go — TempForward.com service (POST create, GET inbox)
package temp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const tempForwardBaseURL = "https://tempforward.com"

// TempForward implements email.Service cho tempforward.com
type TempForward struct {
	client *http.Client
	token  string
	email  string
}

// NewTempForward tạo TempForward service.
func NewTempForward(proxyStr string) *TempForward {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &TempForward{client: client}
}

// CreateEmail tạo mailbox mới qua POST /api/tempmail/create.
func (m *TempForward) CreateEmail(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST",
		tempForwardBaseURL+"/api/tempmail/create",
		bytes.NewBufferString(""))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://tempforward.com/")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tempforward create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var result struct {
		Success bool `json:"success"`
		Mailbox struct {
			Token   string `json:"token"`
			Address string `json:"address"`
		} `json:"mailbox"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Success || result.Mailbox.Token == "" {
		return "", fmt.Errorf("tempforward create: unexpected response: %.200s", body)
	}

	m.token = result.Mailbox.Token
	m.email = result.Mailbox.Address
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *TempForward) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *TempForward) Close() {}

// WaitForCode poll OTP từ tempforward.com inbox.
func (m *TempForward) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.token == "" {
		return "", fmt.Errorf("tempforward: chưa tạo email")
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

	return "", fmt.Errorf("tempforward: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy inbox và extract code.
func (m *TempForward) pollOnce(ctx context.Context) (string, error) {
	listURL := tempForwardBaseURL + "/api/tempmail/inbox?token=" + m.token
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

	var result struct {
		Emails []struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
		} `json:"emails"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("tempforward inbox parse: %w", err)
	}

	for _, msg := range result.Emails {
		content, err := m.getMessage(ctx, msg.ID)
		if err != nil {
			continue
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
		if code := ExtractCode(msg.Subject); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// getMessage lấy nội dung email theo ID.
func (m *TempForward) getMessage(ctx context.Context, id string) (string, error) {
	msgURL := tempForwardBaseURL + "/api/tempmail/email/" + id + "?token=" + m.token
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
		HTMLBody string `json:"html_body"`
		TextBody string `json:"text_body"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", err
	}

	if msg.HTMLBody != "" {
		return msg.HTMLBody, nil
	}
	return msg.TextBody, nil
}
