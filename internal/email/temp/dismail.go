// dismail.go — Dismail.top service (REST API)
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

const dismailBaseURL = "https://dismail.top"

// Dismail implements email.Service cho dismail.top
type Dismail struct {
	client  *http.Client
	mailID  string
	email   string
}

// NewDismail tạo Dismail service.
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewDismail(proxyStr string) *Dismail {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &Dismail{client: client}
}

// CreateEmail tạo email mới qua POST /api/generate-email.
func (d *Dismail) CreateEmail(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST",
		dismailBaseURL+"/api/generate-email",
		bytes.NewBufferString("{}"))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("dismail create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 32*1024)

	var result struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.ID == "" || result.Email == "" {
		return "", fmt.Errorf("dismail create: unexpected response: %s", string(body))
	}

	d.mailID = result.ID
	d.email = result.Email
	return d.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (d *Dismail) GetEmail() string { return d.email }

// Close cleanup resources.
func (d *Dismail) Close() {}

// WaitForCode poll OTP từ dismail.top inbox.
func (d *Dismail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if d.mailID == "" {
		return "", fmt.Errorf("dismail: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := d.pollOnce(ctx)
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

	return "", fmt.Errorf("dismail: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy inbox và extract code.
func (d *Dismail) pollOnce(ctx context.Context) (string, error) {
	inboxURL := dismailBaseURL + "/api/check-inbox/" + d.mailID
	req, err := http.NewRequestWithContext(ctx, "GET", inboxURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var inbox struct {
		Messages []struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
			BodyHTML string `json:"body_html"`
			Body    string `json:"body"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &inbox); err != nil {
		return "", fmt.Errorf("dismail inbox parse: %w", err)
	}

	for _, msg := range inbox.Messages {
		// Thử subject trước
		if msg.Subject != "" {
			if code := ExtractCode(msg.Subject); code != "" {
				return code, nil
			}
		}
		// Đọc body_html trực tiếp từ inbox list (không cần getMessage)
		content := msg.BodyHTML
		if content == "" {
			content = msg.Body
		}
		// Fallback: gọi getMessage nếu inbox không có body
		if content == "" && msg.ID != "" {
			content, _ = d.getMessage(ctx, msg.ID)
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// getMessage lấy nội dung email theo id.
func (d *Dismail) getMessage(ctx context.Context, id string) (string, error) {
	msgURL := dismailBaseURL + "/api/get-message/" + id
	req, err := http.NewRequestWithContext(ctx, "GET", msgURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)

	var msg struct {
		Body    string `json:"body"`
		HTML    string `json:"html"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", err
	}

	if msg.HTML != "" {
		return msg.HTML, nil
	}
	if msg.Body != "" {
		return msg.Body, nil
	}
	return msg.Content, nil
}
