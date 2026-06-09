// dinlaan.go — Dinlaan.com service (client-side email)
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

const dinlaanBaseURL = "https://dinlaan.com"

// Dinlaan implements email.Service cho dinlaan.com
type Dinlaan struct {
	client *http.Client
	email  string
}

// NewDinlaan tạo Dinlaan service.
func NewDinlaan(proxyStr string) *Dinlaan {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &Dinlaan{client: client}
}

// CreateEmail sinh địa chỉ email client-side.
func (m *Dinlaan) CreateEmail(_ context.Context) (string, error) {
	m.email = realisticEmail("dinlaan.com")
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *Dinlaan) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *Dinlaan) Close() {}

// WaitForCode poll OTP từ dinlaan.com inbox.
func (m *Dinlaan) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" {
		return "", fmt.Errorf("dinlaan: chưa tạo email")
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

	return "", fmt.Errorf("dinlaan: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy inbox và extract code.
func (m *Dinlaan) pollOnce(ctx context.Context) (string, error) {
	listURL := dinlaanBaseURL + "/checkmail.php?mail=" + url.QueryEscape(m.email) + "&latest_id=0"
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
			From    string `json:"from"`
			Subject string `json:"subject"`
		} `json:"emails"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", nil // API trả HTML/error page → coi như inbox rỗng
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
func (m *Dinlaan) getMessage(ctx context.Context, id string) (string, error) {
	msgURL := dinlaanBaseURL + "/viewmail.php?id=" + id
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

	var result struct {
		Email struct {
			Body string `json:"body"`
		} `json:"email"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.Email.Body, nil
}
