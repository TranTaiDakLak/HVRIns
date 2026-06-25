// expressmail.go — expressmail.app service (api2.expressmail.app backend, no domain selection).
//
// Flow (xác nhận 2026-06-18):
//  1. POST https://api2.expressmail.app/v2/anonymous/mailbox  body {} → {"id":"...","address":"..."}
//  2. GET  https://api2.expressmail.app/v2/anonymous/mailbox/{id}/messages?limit=20&skip=0 → [{...}]
//  3. GET  https://api2.expressmail.app/v2/anonymous/mailbox/{id}/messages/{msgId} → nội dung
//
// KHÔNG cần key, KHÔNG Cloudflare — http.Client thường qua proxy pool.
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const expressMailAPIBase = "https://api2.expressmail.app/v2/anonymous/mailbox"

// ExpressMail implements email.Service cho expressmail.app.
type ExpressMail struct {
	client    *http.Client
	email     string
	mailboxID string
}

// NewExpressMail tạo ExpressMail service.
func NewExpressMail(proxyStr string) *ExpressMail {
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	return &ExpressMail{client: c}
}

func (e *ExpressMail) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", "https://expressmail.app")
	req.Header.Set("Referer", "https://expressmail.app/")
}

// CreateEmail tạo mailbox qua POST /v2/anonymous/mailbox.
func (e *ExpressMail) CreateEmail(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", expressMailAPIBase, strings.NewReader("{}"))
	if err != nil {
		return "", err
	}
	e.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("expressmail create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	var result struct {
		ID      string `json:"id"`
		Address string `json:"address"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Address == "" || result.ID == "" {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return "", fmt.Errorf("expressmail create: no mailbox (HTTP %d) snippet=%q", resp.StatusCode, snippet)
	}
	e.email = result.Address
	e.mailboxID = result.ID
	return e.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (e *ExpressMail) GetEmail() string { return e.email }

// Close no-op.
func (e *ExpressMail) Close() {}

// WaitForCode poll OTP từ /messages.
func (e *ExpressMail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if e.mailboxID == "" {
		return "", fmt.Errorf("expressmail: chưa tạo mailbox")
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if code, _ := e.pollOnce(ctx); code != "" {
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
	return "", fmt.Errorf("expressmail: không nhận được OTP sau %d lần thử", maxRetry)
}

func (e *ExpressMail) get(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	e.setHeaders(req)
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := httpx.ReadBody(resp.Body, 0)
	return b, nil
}

func (e *ExpressMail) pollOnce(ctx context.Context) (string, error) {
	body, err := e.get(ctx, fmt.Sprintf("%s/%s/messages?limit=20&skip=0", expressMailAPIBase, e.mailboxID))
	if err != nil {
		return "", err
	}
	// Field names xác nhận từ bundle: subject, from, body_text, body_html, received_date.
	var list []struct {
		ID       string `json:"id"`
		Subject  string `json:"subject"`
		BodyText string `json:"body_text"`
		BodyHTML string `json:"body_html"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return "", nil
	}
	for _, m := range list {
		for _, field := range []string{m.Subject, m.BodyText, m.BodyHTML} {
			if code := ExtractCode(field); code != "" {
				return code, nil
			}
		}
		if m.ID != "" {
			if content := e.getMessage(ctx, m.ID); content != "" {
				if code := ExtractCode(content); code != "" {
					return code, nil
				}
			}
		}
	}
	return "", nil
}

func (e *ExpressMail) getMessage(ctx context.Context, msgID string) string {
	body, err := e.get(ctx, fmt.Sprintf("%s/%s/messages/%s", expressMailAPIBase, e.mailboxID, msgID))
	if err != nil {
		return ""
	}
	var msg struct {
		Subject  string `json:"subject"`
		BodyText string `json:"body_text"`
		BodyHTML string `json:"body_html"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return string(body)
	}
	return msg.Subject + "\n" + msg.BodyText + "\n" + msg.BodyHTML
}
