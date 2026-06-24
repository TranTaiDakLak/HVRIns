// tempmailnow.go — tempmail.now service (Flask backend, cookie session, no domain selection).
//
// Flow (xác nhận 2026-06-18):
//   1. GET /api/temp_email  → {"email":"...","created_at":"..."}  (set session cookie)
//   2. GET /fetch_emails    → {"emails":[...],"remaining_time":N}  (CÙNG cookie jar)
//
// KHÔNG cần key, KHÔNG Cloudflare — http.Client thường qua proxy pool.
// Session gắn với cookie nên Client phải có cookie jar dùng chung giữa 2 call.
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const tempMailNowBaseURL = "https://tempmail.now"

// TempMailNow implements email.Service cho tempmail.now.
type TempMailNow struct {
	client *http.Client
	email  string
}

// NewTempMailNow tạo TempMailNow service.
func NewTempMailNow(proxyStr string) *TempMailNow {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	c.Jar = jar
	return &TempMailNow{client: c}
}

func (t *TempMailNow) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tempMailNowBaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", tempMailNowBaseURL+"/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := httpx.ReadBody(resp.Body, 0)
	return b, nil
}

// CreateEmail tạo email qua GET /api/temp_email (session lưu trong cookie jar).
func (t *TempMailNow) CreateEmail(ctx context.Context) (string, error) {
	body, err := t.get(ctx, "/api/temp_email")
	if err != nil {
		return "", fmt.Errorf("tempmailnow create: %w", err)
	}
	var result struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Email == "" {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return "", fmt.Errorf("tempmailnow create: no email snippet=%q", snippet)
	}
	t.email = result.Email
	return t.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (t *TempMailNow) GetEmail() string { return t.email }

// Close no-op.
func (t *TempMailNow) Close() {}

// WaitForCode poll OTP từ /fetch_emails.
func (t *TempMailNow) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmailnow: chưa tạo email")
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
	return "", fmt.Errorf("tempmailnow: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempMailNow) pollOnce(ctx context.Context) (string, error) {
	body, err := t.get(ctx, "/fetch_emails")
	if err != nil {
		return "", err
	}
	var result struct {
		Emails []struct {
			Subject string `json:"subject"`
			From    string `json:"from"`
			Body    string `json:"body"`
			BodyTxt string `json:"body_text"`
			HTML    string `json:"html"`
			Content string `json:"content"`
		} `json:"emails"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", nil
	}
	for _, m := range result.Emails {
		for _, field := range []string{m.Subject, m.BodyTxt, m.Body, m.Content, m.HTML} {
			if code := ExtractCode(field); code != "" {
				return code, nil
			}
		}
	}
	return "", nil
}
