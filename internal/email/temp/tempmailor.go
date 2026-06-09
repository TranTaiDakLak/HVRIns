// tempmailor.go — Temp-Mail.org service (web2.temp-mail.org REST API)
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

const tempMailOrgBaseURL = "https://web2.temp-mail.org"

// TempMailOrg implements email.Service cho web2.temp-mail.org
type TempMailOrg struct {
	client *http.Client
	token  string
	email  string
}

// NewTempMailOrg tạo TempMailOrg service.
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewTempMailOrg(proxyStr string) *TempMailOrg {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &TempMailOrg{client: client}
}

// CreateEmail tạo mailbox mới qua POST /mailbox.
func (t *TempMailOrg) CreateEmail(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST",
		tempMailOrgBaseURL+"/mailbox",
		bytes.NewBufferString("{}"))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tempmail.org create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var result struct {
		Token   string `json:"token"`
		Mailbox string `json:"mailbox"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Token == "" || result.Mailbox == "" {
		return "", fmt.Errorf("tempmail.org create: unexpected response: %s", string(body))
	}

	t.token = result.Token
	t.email = result.Mailbox
	return t.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (t *TempMailOrg) GetEmail() string { return t.email }

// Close cleanup resources.
func (t *TempMailOrg) Close() {}

// WaitForCode poll OTP từ temp-mail.org inbox.
func (t *TempMailOrg) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.token == "" {
		return "", fmt.Errorf("tempmail.org: chưa tạo email")
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

	return "", fmt.Errorf("tempmail.org: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy danh sách email và extract code.
func (t *TempMailOrg) pollOnce(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		tempMailOrgBaseURL+"/messages", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var msgs []struct {
		ID string `json:"_id"`
	}
	if err := json.Unmarshal(body, &msgs); err != nil {
		return "", fmt.Errorf("tempmail.org inbox parse: %w", err)
	}

	for _, msg := range msgs {
		content, err := t.getMessage(ctx, msg.ID)
		if err != nil {
			continue
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// getMessage lấy nội dung email theo _id.
func (t *TempMailOrg) getMessage(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		tempMailOrgBaseURL+"/messages/"+id, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)

	var msg struct {
		BodyHTML string `json:"bodyHtml"`
		BodyText string `json:"bodyText"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", err
	}

	if msg.BodyHTML != "" {
		return msg.BodyHTML, nil
	}
	return msg.BodyText, nil
}
