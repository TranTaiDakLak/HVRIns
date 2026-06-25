// mailcx.go — Mail.cx service (api.mail.cx v1 REST API — updated 2026-05)
//
// API mới: https://api.mail.cx/v1/  (không phải /api/v1/ cũ đã chết)
// Auth: header x-api-token: tm_live_...  (lấy từ mail.cx/dashboard)
// Domain: GET /v1/config → system_domains[0].domain  (hiện tại: ddker.com)
// Flow: config → random email → GET /v1/inbox/{email} → GET /v1/email/{id}
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const mailCxBaseURL = "https://api.mail.cx/v1"

// MailCx implements email.Service cho mail.cx
type MailCx struct {
	client   *http.Client
	apiToken string
	email    string
	domain   string
}

// NewMailCx tạo MailCx service.
// apiToken: token tm_live_... từ mail.cx/dashboard
// proxyStr: proxy URL, để trống nếu không dùng proxy
func NewMailCx(apiToken, proxyStr string) *MailCx {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &MailCx{client: client, apiToken: apiToken}
}

// addAuth thêm x-api-token header
func (m *MailCx) addAuth(req *http.Request) {
	if m.apiToken != "" {
		req.Header.Set("x-api-token", m.apiToken)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
}

// fetchDomain lấy domain từ GET /v1/config (không cần auth)
func (m *MailCx) fetchDomain(ctx context.Context) string {
	req, err := http.NewRequestWithContext(ctx, "GET", mailCxBaseURL+"/config", nil)
	if err != nil {
		return "ddker.com"
	}
	req.Header.Set("Accept", "application/json")
	resp, err := m.client.Do(req)
	if err != nil {
		return "ddker.com"
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 8*1024)

	var cfg struct {
		SystemDomains []struct {
			Domain  string `json:"domain"`
			Default bool   `json:"default"`
		} `json:"system_domains"`
	}
	if err := json.Unmarshal(body, &cfg); err != nil || len(cfg.SystemDomains) == 0 {
		return "ddker.com"
	}
	// Random từ TẤT CẢ system_domains (không chỉ default) → phân tán email qua nhiều domain.
	return cfg.SystemDomains[rand.Intn(len(cfg.SystemDomains))].Domain
}

// randomLocalPart tạo local part ngẫu nhiên theo pattern mail.cx: a-z0-9._-
func randomLocalPart() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 10)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// CreateEmail lấy domain từ config, sinh email random (mailbox là implicit).
func (m *MailCx) CreateEmail(ctx context.Context) (string, error) {
	if m.apiToken == "" {
		return "", fmt.Errorf("mail.cx: chưa có API token — vào mail.cx/dashboard lấy token tm_live_...")
	}
	m.domain = m.fetchDomain(ctx)
	m.email = realisticLocalPart() + "@" + m.domain
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *MailCx) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *MailCx) Close() {}

// WaitForCode dùng long-poll của mail.cx: ?count=1 → server đợi đến khi có mail.
// Mỗi round chỉ tốn 1 request thay vì poll liên tục → tiết kiệm quota 500/day.
func (m *MailCx) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 5 // mỗi retry = 1 long-poll request (~60s server wait)
	}
	if m.email == "" {
		return "", fmt.Errorf("mail.cx: chưa tạo email")
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		code, err := m.longPollOnce(ctx)
		if err != nil {
			// rate limit → đợi 60s rồi retry
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(60 * time.Second):
			}
			continue
		}
		if code != "" {
			return code, nil
		}
	}
	return "", fmt.Errorf("mail.cx: không nhận được OTP sau %d lần thử", maxRetry)
}

// longPollOnce dùng ?count=1 — server đợi đến khi có ít nhất 1 mail (tối đa ~60s).
// Chỉ tốn 1 quota request per call.
func (m *MailCx) longPollOnce(ctx context.Context) (string, error) {
	// count=1: server block đến khi có mail hoặc timeout
	url := fmt.Sprintf("%s/inbox/%s?count=1&limit=5", mailCxBaseURL, m.email)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	m.addAuth(req)

	// Timeout dài hơn cho long-poll (server có thể giữ kết nối ~60s)
	longClient := *m.client
	longClient.Timeout = 90 * time.Second
	resp, err := longClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return "", fmt.Errorf("mail.cx: rate_limit/quota_exceeded")
	}

	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		var msgs []struct {
			ID string `json:"id"`
		}
		if err2 := json.Unmarshal(body, &msgs); err2 != nil {
			return "", nil
		}
		for _, msg := range msgs {
			if content, _ := m.getMessage(ctx, msg.ID); content != "" {
				if code := ExtractCode(content); code != "" {
					return code, nil
				}
			}
		}
		return "", nil
	}

	for _, msg := range result.Messages {
		if content, _ := m.getMessage(ctx, msg.ID); content != "" {
			if code := ExtractCode(content); code != "" {
				return code, nil
			}
		}
	}
	return "", nil
}

// getMessage lấy nội dung email theo id — GET /v1/email/{id}
func (m *MailCx) getMessage(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/email/%s", mailCxBaseURL, id), nil)
	if err != nil {
		return "", err
	}
	m.addAuth(req)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)

	var msg struct {
		Body struct {
			HTML string `json:"html"`
			Text string `json:"text"`
		} `json:"body"`
		HTML string `json:"html"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", err
	}
	if msg.Body.HTML != "" {
		return msg.Body.HTML, nil
	}
	if msg.Body.Text != "" {
		return msg.Body.Text, nil
	}
	if msg.HTML != "" {
		return msg.HTML, nil
	}
	return msg.Text, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
