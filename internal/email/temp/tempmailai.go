// tempmailai.go — temp-mail.ai service (Express REST JSON API)
//
// Flow (xác nhận qua research 2026-06-19):
//  1. POST /api/mailbox/random {}                          → {success, email, id}
//  2. GET  /api/mailbox/{encoded_email}/messages          → {success, messages:[{id,subject,text,html}]}
//  3. GET  /api/mailbox/{encoded_email}/message/{id}      → {success, message:{text,html}}
//
// QUAN TRỌNG: email trong URL phải dùng url.PathEscape (@ → %40).
// Rate limit: 50 req/600s per IP. KHÔNG cần key.
package temp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const (
	tempMailAIBaseURL = "https://temp-mail.ai/api"
	tempMailAIUA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

// TempMailAI implements email.Service cho temp-mail.ai.
type TempMailAI struct {
	client *http.Client
	email  string
}

// NewTempMailAI tạo TempMailAI service.
func NewTempMailAI(proxyStr string) *TempMailAI {
	return &TempMailAI{client: proxy.CreateClient(proxyStr, 30*time.Second)}
}

func (t *TempMailAI) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", tempMailAIUA)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://temp-mail.ai")
	req.Header.Set("Referer", "https://temp-mail.ai/")
}

// CreateEmail: POST /api/mailbox/random → email.
func (t *TempMailAI) CreateEmail(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST",
		tempMailAIBaseURL+"/mailbox/random", bytes.NewReader([]byte("{}")))
	if err != nil {
		return "", err
	}
	t.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tempmailai create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var result struct {
		Success bool   `json:"success"`
		Email   string `json:"email"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Success || result.Email == "" {
		return "", fmt.Errorf("tempmailai create: no email (HTTP %d) — body: %.200s", resp.StatusCode, body)
	}
	t.email = result.Email
	return t.email, nil
}

// GetEmail trả về địa chỉ đã tạo.
func (t *TempMailAI) GetEmail() string { return t.email }

// Close no-op.
func (t *TempMailAI) Close() {}

// WaitForCode poll OTP từ inbox.
func (t *TempMailAI) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmailai: chưa tạo email")
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if code, _ := t.pollOnce(ctx); code != "" {
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
	return "", fmt.Errorf("tempmailai: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempMailAI) pollOnce(ctx context.Context) (string, error) {
	// email phải PathEscape: "user@domain.com" → "user%40domain.com"
	encodedEmail := url.PathEscape(t.email)
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/mailbox/%s/messages", tempMailAIBaseURL, encodedEmail), nil)
	if err != nil {
		return "", err
	}
	t.setHeaders(req)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	var result struct {
		Success  bool `json:"success"`
		Messages []struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
			Text    string `json:"text"`
			HTML    string `json:"html"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Success {
		return "", nil
	}
	for _, m := range result.Messages {
		if code := ExtractCode(m.Subject); code != "" {
			return code, nil
		}
		if code := ExtractCode(m.Text); code != "" {
			return code, nil
		}
		if code := ExtractCode(m.HTML); code != "" {
			return code, nil
		}
		// Fallback: fetch full content nếu inline chưa đủ
		if content, _ := t.getMessage(ctx, m.ID); content != "" {
			if code := ExtractCode(content); code != "" {
				return code, nil
			}
		}
	}
	return "", nil
}

func (t *TempMailAI) getMessage(ctx context.Context, id string) (string, error) {
	encodedEmail := url.PathEscape(t.email)
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/mailbox/%s/message/%s", tempMailAIBaseURL, encodedEmail, id), nil)
	if err != nil {
		return "", err
	}
	t.setHeaders(req)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	var result struct {
		Success bool `json:"success"`
		Message struct {
			HTML string `json:"html"`
			Text string `json:"text"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.Message.HTML != "" {
		return result.Message.HTML, nil
	}
	return result.Message.Text, nil
}
