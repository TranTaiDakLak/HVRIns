// cryptogmail.go — CryptoGmail.com service (client-side email, REST API)
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const cryptoGmailBaseURL = "https://cryptogmail.com"

// CryptoGmail implements email.Service cho cryptogmail.com
type CryptoGmail struct {
	client *http.Client
	email  string
}

// NewCryptoGmail tạo CryptoGmail service.
func NewCryptoGmail(proxyStr string) *CryptoGmail {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &CryptoGmail{client: client}
}

// CreateEmail sinh địa chỉ email client-side.
func (m *CryptoGmail) CreateEmail(_ context.Context) (string, error) {
	m.email = realisticEmail("gmail.com")
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *CryptoGmail) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *CryptoGmail) Close() {}

// WaitForCode poll OTP từ cryptogmail.com inbox.
func (m *CryptoGmail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" {
		return "", fmt.Errorf("cryptogmail: chưa tạo email")
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

	return "", fmt.Errorf("cryptogmail: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy inbox và extract code.
func (m *CryptoGmail) pollOnce(ctx context.Context) (string, error) {
	listURL := cryptoGmailBaseURL + "/api/emails?inbox=" + url.QueryEscape(m.email)
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
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", nil // API trả HTML/error page → coi như inbox rỗng
	}

	for _, msg := range result.Data {
		content, err := m.getMessage(ctx, msg.ID)
		if err != nil {
			continue
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// getMessage lấy nội dung email theo ID.
func (m *CryptoGmail) getMessage(ctx context.Context, id string) (string, error) {
	msgURL := cryptoGmailBaseURL + "/api/emails/" + id
	req, err := http.NewRequestWithContext(ctx, "GET", msgURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/html,text/plain")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)
	return string(body), nil
}
