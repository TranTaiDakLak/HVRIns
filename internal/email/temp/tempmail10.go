// tempmail10.go — TempMail10.com service (Laravel session + XSRF dual-token)
//
// Flow (xác nhận qua research 2026-06-19):
//  1. GET  /en (cookie jar)
//     → lưu XSRF-TOKEN cookie + csrf-token meta + tempmail10_session
//  2. POST /get_messages (Content-Type: application/x-www-form-urlencoded)
//     header: X-XSRF-TOKEN = url-decoded(XSRF-TOKEN cookie)
//     header: X-Requested-With: XMLHttpRequest
//     body:   _token={csrf}&captcha=
//     → {status:true, mailbox:"user@domain.com", messages:[{id,from,subject}]}
//  3. GET  /en/view/{id} → HTML page → ExtractCode
//
// Pattern giống MailWave nhưng body dùng form-encoded thay JSON.
// KHÔNG cần key. Domain rotate tự động.
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const (
	tempMail10BaseURL = "https://tempmail10.com"
	tempMail10UA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

var tempMail10CsrfRe = regexp.MustCompile(`<meta\s+name="csrf-token"\s+content="([^"]+)"`)

// TempMail10 implements email.Service cho tempmail10.com.
type TempMail10 struct {
	client    *http.Client
	email     string
	csrfToken string // từ <meta name="csrf-token"> trong GET /en
	xsrfToken string // url-decoded XSRF-TOKEN cookie → X-XSRF-TOKEN header
}

// NewTempMail10 tạo TempMail10 service.
func NewTempMail10(proxyStr string) *TempMail10 {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	c.Jar = jar
	return &TempMail10{client: c}
}

func (t *TempMail10) setFormHeaders(req *http.Request) {
	req.Header.Set("User-Agent", tempMail10UA)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", tempMail10BaseURL+"/en")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("X-XSRF-TOKEN", t.xsrfToken)
}

// CreateEmail: GET /en → CSRF tokens → POST /get_messages → mailbox.
func (t *TempMail10) CreateEmail(ctx context.Context) (string, error) {
	// Step 1: GET /en → cookies + csrf meta
	req, err := http.NewRequestWithContext(ctx, "GET", tempMail10BaseURL+"/en", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", tempMail10UA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tempmail10 init: %w", err)
	}
	defer resp.Body.Close()
	html, _ := httpx.ReadBody(resp.Body, 256*1024)

	// Extract <meta name="csrf-token" content="...">
	if match := tempMail10CsrfRe.FindSubmatch(html); len(match) >= 2 {
		t.csrfToken = string(match[1])
	}
	if t.csrfToken == "" {
		return "", fmt.Errorf("tempmail10 init: không tìm được csrf-token — body: %.200s", html)
	}

	// Extract XSRF-TOKEN cookie (Laravel URL-encodes giá trị này)
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "XSRF-TOKEN" {
			if decoded, err := url.QueryUnescape(cookie.Value); err == nil {
				t.xsrfToken = decoded
			} else {
				t.xsrfToken = cookie.Value
			}
			break
		}
	}
	if t.xsrfToken == "" {
		return "", fmt.Errorf("tempmail10 init: không tìm được XSRF-TOKEN cookie")
	}

	// Step 2: POST /get_messages (form) → mailbox
	formBody := "_token=" + url.QueryEscape(t.csrfToken) + "&captcha="
	req2, err := http.NewRequestWithContext(ctx, "POST",
		tempMail10BaseURL+"/get_messages", strings.NewReader(formBody))
	if err != nil {
		return "", err
	}
	t.setFormHeaders(req2)

	resp2, err := t.client.Do(req2)
	if err != nil {
		return "", fmt.Errorf("tempmail10 get_messages: %w", err)
	}
	defer resp2.Body.Close()
	body, _ := httpx.ReadBody(resp2.Body, 256*1024)

	var result struct {
		Status  bool   `json:"status"`
		Mailbox string `json:"mailbox"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Status || result.Mailbox == "" {
		return "", fmt.Errorf("tempmail10 get_messages: no mailbox (HTTP %d) — body: %.200s", resp2.StatusCode, body)
	}
	t.email = result.Mailbox
	return t.email, nil
}

// GetEmail trả về địa chỉ đã tạo.
func (t *TempMail10) GetEmail() string { return t.email }

// Close no-op.
func (t *TempMail10) Close() {}

// WaitForCode poll OTP từ inbox.
func (t *TempMail10) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmail10: chưa tạo email")
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
	return "", fmt.Errorf("tempmail10: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempMail10) pollOnce(ctx context.Context) (string, error) {
	formBody := "_token=" + url.QueryEscape(t.csrfToken) + "&captcha="
	req, err := http.NewRequestWithContext(ctx, "POST",
		tempMail10BaseURL+"/get_messages", strings.NewReader(formBody))
	if err != nil {
		return "", err
	}
	t.setFormHeaders(req)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	var result struct {
		Messages []struct {
			ID      interface{} `json:"id"` // có thể là int hoặc string
			From    string      `json:"from"`
			Subject string      `json:"subject"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", nil
	}
	for _, msg := range result.Messages {
		if code := ExtractCode(msg.Subject); code != "" {
			return code, nil
		}
		if idStr := jsonIDStr(msg.ID); idStr != "" && idStr != "0" {
			if content, _ := t.getViewContent(ctx, idStr); content != "" {
				if code := ExtractCode(content); code != "" {
					return code, nil
				}
			}
		}
	}
	return "", nil
}

func (t *TempMail10) getViewContent(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tempMail10BaseURL+"/en/view/"+id, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", tempMail10UA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Referer", tempMail10BaseURL+"/en")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)
	return strings.TrimSpace(string(body)), nil
}
