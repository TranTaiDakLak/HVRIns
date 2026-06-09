// tempomintraccoon.go — Tempo.Mintraccoon.com service (POST create, GET inbox with token header)
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

const tempoMintraccoonBaseURL = "https://tempo.mintraccoon.com"

// TempoMintraccoon implements email.Service cho tempo.mintraccoon.com
type TempoMintraccoon struct {
	client *http.Client
	token  string
	email  string
}

// NewTempoMintraccoon tạo TempoMintraccoon service.
func NewTempoMintraccoon(proxyStr string) *TempoMintraccoon {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &TempoMintraccoon{client: client}
}

// CreateEmail tạo mailbox mới qua POST /api/email.
func (m *TempoMintraccoon) CreateEmail(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST",
		tempoMintraccoonBaseURL+"/api/email",
		bytes.NewBufferString(""))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://tempo.mintraccoon.com/")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tempomintraccoon create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var result struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Token == "" || result.Email == "" {
		return "", fmt.Errorf("tempomintraccoon create: unexpected response: %.200s", body)
	}

	m.token = result.Token
	m.email = result.Email
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *TempoMintraccoon) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *TempoMintraccoon) Close() {}

// WaitForCode poll OTP từ tempo.mintraccoon.com inbox.
func (m *TempoMintraccoon) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.token == "" {
		return "", fmt.Errorf("tempomintraccoon: chưa tạo email")
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

	return "", fmt.Errorf("tempomintraccoon: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy inbox và extract code.
func (m *TempoMintraccoon) pollOnce(ctx context.Context) (string, error) {
	listURL := tempoMintraccoonBaseURL + "/api/inbox/" + url.PathEscape(m.email)
	req, err := http.NewRequestWithContext(ctx, "GET", listURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Email-Token", m.token)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var result struct {
		Messages []struct {
			ID      string `json:"id"`
			From    string `json:"from"`
			Subject string `json:"subject"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("tempomintraccoon inbox parse: %w", err)
	}

	for _, msg := range result.Messages {
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
func (m *TempoMintraccoon) getMessage(ctx context.Context, id string) (string, error) {
	msgURL := tempoMintraccoonBaseURL + "/api/message/" + url.PathEscape(m.email) + "/" + id
	req, err := http.NewRequestWithContext(ctx, "GET", msgURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Email-Token", m.token)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)

	var msg struct {
		HTML     string `json:"html"`
		BodyText string `json:"body_text"`
		Text     string `json:"text"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", err
	}

	if msg.HTML != "" {
		return msg.HTML, nil
	}
	if msg.BodyText != "" {
		return msg.BodyText, nil
	}
	return msg.Text, nil
}
