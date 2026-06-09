// tempmailto.go — TempMailTo.com service (CSRF + /get_messages inline content)
// Port từ C# TempMailToAPI. Flow: GET homepage → extract CSRF → POST /get_messages
// khởi tạo session → mailbox random. Message content trả kèm list, không cần fetch lẻ.
package temp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const tempMailToBaseURL = "https://tempmailto.com"

var csrfMetaRegex = regexp.MustCompile(`<meta name="csrf-token" content="([^"]+)"`)

// TempMailTo implements email.Service cho tempmailto.com.
type TempMailTo struct {
	client    *http.Client
	email     string
	csrfToken string
}

// NewTempMailTo tạo TempMailTo service.
func NewTempMailTo(proxyStr string) *TempMailTo {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	c.Jar = jar
	return &TempMailTo{client: c}
}

// CreateEmail lấy CSRF + khởi tạo session + mailbox random.
func (t *TempMailTo) CreateEmail(ctx context.Context) (string, error) {
	email, err := csrfInitSession(ctx, t.client, tempMailToBaseURL, &t.csrfToken)
	if err != nil {
		return "", fmt.Errorf("tempmailto: %w", err)
	}
	t.email = email
	return t.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (t *TempMailTo) GetEmail() string { return t.email }

// Close no-op.
func (t *TempMailTo) Close() {}

// WaitForCode poll OTP.
func (t *TempMailTo) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmailto: chưa tạo email")
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if code, _ := csrfPollInbox(ctx, t.client, tempMailToBaseURL, t.csrfToken); code != "" {
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
	return "", fmt.Errorf("tempmailto: không nhận được OTP sau %d lần thử", maxRetry)
}

// csrfInitSession là helper chung cho tempmailto + 1secemail (cùng pattern Laravel CSRF).
// Trả về email vừa init session.
func csrfInitSession(ctx context.Context, client *http.Client, baseURL string, csrfOut *string) (string, error) {
	// Step 1: GET homepage → extract CSRF token (set cookie session)
	req, _ := http.NewRequestWithContext(ctx, "GET", baseURL+"/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", baseURL+"/")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("home: %w", err)
	}
	defer resp.Body.Close()
	html, _ := httpx.ReadBody(resp.Body, 256*1024)
	m := csrfMetaRegex.FindSubmatch(html)
	if len(m) < 2 || len(m[1]) == 0 {
		return "", fmt.Errorf("csrf token not found")
	}
	*csrfOut = string(m[1])

	// Step 2: POST /get_messages → init session, trả về mailbox
	payload, _ := json.Marshal(map[string]string{"_token": *csrfOut, "captcha": ""})
	req2, _ := http.NewRequestWithContext(ctx, "POST", baseURL+"/get_messages", bytes.NewReader(payload))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-CSRF-TOKEN", *csrfOut)
	req2.Header.Set("X-Requested-With", "XMLHttpRequest")
	req2.Header.Set("User-Agent", "Mozilla/5.0")
	req2.Header.Set("Referer", baseURL+"/")
	resp2, err := client.Do(req2)
	if err != nil {
		return "", fmt.Errorf("get_messages: %w", err)
	}
	defer resp2.Body.Close()
	body, _ := httpx.ReadBody(resp2.Body, 256*1024)
	var result struct {
		Status  bool   `json:"status"`
		Mailbox string `json:"mailbox"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse init: %w — body: %.200s", err, body)
	}
	if result.Mailbox == "" {
		return "", fmt.Errorf("empty mailbox in init response")
	}
	return result.Mailbox, nil
}

// csrfPollInbox gọi POST /get_messages với CSRF token → iterate messages → extract code.
// Content được trả kèm trong list (field "content"), không cần fetch lẻ.
func csrfPollInbox(ctx context.Context, client *http.Client, baseURL, csrf string) (string, error) {
	payload, _ := json.Marshal(map[string]string{"_token": csrf, "captcha": ""})
	req, _ := http.NewRequestWithContext(ctx, "POST", baseURL+"/get_messages", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-TOKEN", csrf)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", baseURL+"/")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 512*1024)

	var result struct {
		Messages []struct {
			ID      string `json:"id"`
			From    string `json:"from_email"`
			Subject string `json:"subject"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	for _, msg := range result.Messages {
		content := msg.Content
		if content == "" {
			content = msg.Subject
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}
