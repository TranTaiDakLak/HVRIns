// mailwave.go — MailWave.dev service (Laravel session + XSRF dual-token)
//
// Flow (xác nhận qua research 2026-06-19):
//  1. GET  / (cookie jar)
//     → lưu XSRF-TOKEN cookie (url-decoded) + csrf-token từ <meta name="csrf-token">
//     + mail_wave_session cookie
//  2. POST /get_messages  (Content-Type: application/json)
//     header: X-XSRF-TOKEN = url-decoded(XSRF-TOKEN cookie)
//     header: X-Requested-With: XMLHttpRequest
//     body:   {"_token": "{csrf_meta_value}"}
//     → {status:true, mailbox:"user@domain.com", messages:[{id,subject}], histories}
//  3. GET  /view/{id} → HTML page → ExtractCode
//
// KHÔNG cần key. Domain tự động gán (random server-side). Đổi domain cần reCAPTCHA.
package temp

import (
	"bytes"
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
	mailWaveBaseURL = "https://mailwave.dev"
	mailWaveUA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

var mailWaveCsrfRe = regexp.MustCompile(`<meta\s+name="csrf-token"\s+content="([^"]+)"`)

// MailWave implements email.Service cho mailwave.dev.
type MailWave struct {
	client    *http.Client
	email     string
	csrfToken string // từ <meta name="csrf-token"> trong GET /
	xsrfToken string // url-decoded XSRF-TOKEN cookie → X-XSRF-TOKEN header
}

// NewMailWave tạo MailWave service.
func NewMailWave(proxyStr string) *MailWave {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	c.Jar = jar
	return &MailWave{client: c}
}

func (m *MailWave) setAjaxHeaders(req *http.Request) {
	req.Header.Set("User-Agent", mailWaveUA)
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", mailWaveBaseURL+"/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("X-XSRF-TOKEN", m.xsrfToken)
}

// CreateEmail: GET / → CSRF tokens → POST /get_messages → email.
func (m *MailWave) CreateEmail(ctx context.Context) (string, error) {
	// Step 1: GET / để lấy session cookies + csrf meta tag
	req, err := http.NewRequestWithContext(ctx, "GET", mailWaveBaseURL+"/", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", mailWaveUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("mailwave init: %w", err)
	}
	defer resp.Body.Close()
	html, _ := httpx.ReadBody(resp.Body, 256*1024)

	// Extract <meta name="csrf-token" content="...">
	if match := mailWaveCsrfRe.FindSubmatch(html); len(match) >= 2 {
		m.csrfToken = string(match[1])
	}
	if m.csrfToken == "" {
		return "", fmt.Errorf("mailwave init: không tìm được csrf-token — body: %.200s", html)
	}

	// Extract XSRF-TOKEN cookie (Laravel URL-encodes giá trị này)
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "XSRF-TOKEN" {
			if decoded, err := url.QueryUnescape(cookie.Value); err == nil {
				m.xsrfToken = decoded
			} else {
				m.xsrfToken = cookie.Value
			}
			break
		}
	}
	if m.xsrfToken == "" {
		return "", fmt.Errorf("mailwave init: không tìm được XSRF-TOKEN cookie")
	}

	// Step 2: POST /get_messages → mailbox address
	email, err := m.fetchMessages(ctx)
	if err != nil {
		return "", err
	}
	m.email = email
	return m.email, nil
}

// fetchMessages gọi POST /get_messages và trả về mailbox hoặc messages.
func (m *MailWave) fetchMessages(ctx context.Context) (string, error) {
	payload, _ := json.Marshal(map[string]string{"_token": m.csrfToken})
	req, err := http.NewRequestWithContext(ctx, "POST",
		mailWaveBaseURL+"/get_messages", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	m.setAjaxHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("mailwave get_messages: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	var result struct {
		Status  bool   `json:"status"`
		Mailbox string `json:"mailbox"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Status || result.Mailbox == "" {
		return "", fmt.Errorf("mailwave get_messages: no mailbox (HTTP %d) — body: %.200s", resp.StatusCode, body)
	}
	return result.Mailbox, nil
}

// GetEmail trả về địa chỉ đã tạo.
func (m *MailWave) GetEmail() string { return m.email }

// Close no-op.
func (m *MailWave) Close() {}

// WaitForCode poll OTP từ inbox.
func (m *MailWave) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" {
		return "", fmt.Errorf("mailwave: chưa tạo email")
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if code, _ := m.pollOnce(ctx); code != "" {
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
	return "", fmt.Errorf("mailwave: không nhận được OTP sau %d lần thử", maxRetry)
}

func (m *MailWave) pollOnce(ctx context.Context) (string, error) {
	payload, _ := json.Marshal(map[string]string{"_token": m.csrfToken})
	req, err := http.NewRequestWithContext(ctx, "POST",
		mailWaveBaseURL+"/get_messages", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	m.setAjaxHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	var result struct {
		Messages []struct {
			ID      interface{} `json:"id"` // có thể là int hoặc string
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
			if content, _ := m.getViewContent(ctx, idStr); content != "" {
				if code := ExtractCode(content); code != "" {
					return code, nil
				}
			}
		}
	}
	return "", nil
}

func (m *MailWave) getViewContent(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mailWaveBaseURL+"/view/"+id, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", mailWaveUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Referer", mailWaveBaseURL+"/")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)
	return strings.TrimSpace(string(body)), nil
}
