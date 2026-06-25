// mailtm.go — Mail.tm service (JWT-based temp mail)
package temp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const mailTmBaseURL = "https://api.mail.tm"

// MailTm implements email.Service cho api.mail.tm
type MailTm struct {
	client  *http.Client
	email   string
	pass    string
	token   string
	msgBase string
}

// NewMailTm tạo MailTm service.
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewMailTm(proxyStr string) *MailTm {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &MailTm{client: client}
}

// CreateEmail tạo tài khoản mail.tm và lấy JWT token.
func (m *MailTm) CreateEmail(ctx context.Context) (string, error) {
	domain, err := m.pickDomain(ctx)
	if err != nil {
		return "", fmt.Errorf("mailtm pick domain: %w", err)
	}

	user := realisticLocalPart()
	pass := randomMailTmPass(12)
	addr := user + "@" + domain

	// POST /accounts — retry on 429 (rate limit from concurrent requests).
	accPayload, _ := json.Marshal(map[string]string{"address": addr, "password": pass})
	var createStatus int
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			delay := time.Duration(attempt*3) * time.Second
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(delay):
			}
		}
		req, err := http.NewRequestWithContext(ctx, "POST", mailTmBaseURL+"/accounts", bytes.NewReader(accPayload))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")

		resp, err := m.client.Do(req)
		if err != nil {
			return "", fmt.Errorf("mailtm create account: %w", err)
		}
		createStatus = resp.StatusCode
		if resp.StatusCode == 201 {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			break
		}
		body, _ := httpx.ReadBody(resp.Body, 0)
		resp.Body.Close()
		if resp.StatusCode != 429 {
			return "", fmt.Errorf("mailtm create account: status %d — %s", resp.StatusCode, string(body))
		}
	}
	if createStatus != 201 {
		return "", fmt.Errorf("mailtm create account: rate limited (429) after retries")
	}

	// Wait for account to propagate before requesting token.
	// mail.tm eventual consistency can take 5-30s — use aggressive retry.
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(6 * time.Second):
	}

	// POST /token — retry on 401/422 with increasing backoff (propagation delay).
	tokPayload, _ := json.Marshal(map[string]string{"address": addr, "password": pass})
	var tokResult struct {
		Token string `json:"token"`
	}
	tokBackoffs := []time.Duration{5 * time.Second, 8 * time.Second, 10 * time.Second, 12 * time.Second}
	var lastTokBody []byte
	for attempt := 0; attempt <= len(tokBackoffs); attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(tokBackoffs[attempt-1]):
			}
		}
		req2, err := http.NewRequestWithContext(ctx, "POST", mailTmBaseURL+"/token", bytes.NewReader(tokPayload))
		if err != nil {
			return "", err
		}
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Accept", "application/json")

		resp2, err := m.client.Do(req2)
		if err != nil {
			return "", fmt.Errorf("mailtm get token: %w", err)
		}
		lastTokBody, _ = httpx.ReadBody(resp2.Body, 0)
		resp2.Body.Close()

		if err := json.Unmarshal(lastTokBody, &tokResult); err == nil && tokResult.Token != "" {
			break
		}
	}
	if tokResult.Token == "" {
		return "", fmt.Errorf("mailtm get token: unexpected response: %s", string(lastTokBody))
	}

	m.email = addr
	m.pass = pass
	m.token = tokResult.Token
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *MailTm) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *MailTm) Close() {}

// WaitForCode poll OTP từ mail.tm inbox.
func (m *MailTm) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.token == "" {
		return "", fmt.Errorf("mailtm: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := m.pollOnce(ctx)
		if err == errMailTmAccountGone {
			return "", err // account bị xóa → không retry
		}
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

	return "", fmt.Errorf("mailtm: không nhận được OTP sau %d lần thử", maxRetry)
}

// pickDomain lấy random domain từ GET /domains để phân tán tải giữa các domain.
// Retry on network errors (DNS fail, timeout) up to 3 times with 5s backoff.
func (m *MailTm) pickDomain(ctx context.Context) (string, error) {
	var body []byte
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(5 * time.Second):
			}
		}
		req, err := http.NewRequestWithContext(ctx, "GET", mailTmBaseURL+"/domains?page=1", nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Accept", "application/json")
		resp, err := m.client.Do(req)
		if err != nil {
			if attempt == 2 {
				return "", fmt.Errorf("mailtm get domains: %w", err)
			}
			continue
		}
		body, _ = httpx.ReadBody(resp.Body, 0)
		resp.Body.Close()
		break
	}
	if len(body) == 0 {
		return "", fmt.Errorf("mailtm get domains: empty response")
	}

	var domains []string

	// Thử parse dạng Hydra (cũ): {"hydra:member":[{"domain":"..."},...]}
	var hydra struct {
		HydraMember []struct {
			Domain string `json:"domain"`
		} `json:"hydra:member"`
	}
	if err := json.Unmarshal(body, &hydra); err == nil && len(hydra.HydraMember) > 0 {
		for _, d := range hydra.HydraMember {
			domains = append(domains, d.Domain)
		}
	}

	// Thử parse dạng array thẳng (mới): [{"domain":"..."}]
	if len(domains) == 0 {
		var arr []struct {
			Domain string `json:"domain"`
		}
		if err := json.Unmarshal(body, &arr); err == nil {
			for _, d := range arr {
				domains = append(domains, d.Domain)
			}
		}
	}

	if len(domains) == 0 {
		return "", fmt.Errorf("mailtm domains: unexpected response: %s", string(body))
	}
	return domains[rand.Intn(len(domains))], nil
}

// mailTmMessage là struct dùng chung cho cả Hydra và array format.
type mailTmMessage struct {
	ID      string `json:"id"`
	Subject string `json:"subject"`
	Intro   string `json:"intro"`
}

// errMailTmAccountGone là sentinel báo account bị xóa → dừng poll ngay.
var errMailTmAccountGone = fmt.Errorf("mailtm: account no longer exists")

// pollOnce lấy inbox một lần và extract code.
// Hỗ trợ cả API cũ (hydra:member) và API mới (array thẳng).
func (m *MailTm) pollOnce(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mailTmBaseURL+"/messages?page=1", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.token)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	// 401 = account bị xóa → không retry nữa
	if resp.StatusCode == 401 {
		return "", errMailTmAccountGone
	}

	var messages []mailTmMessage

	// Thử parse Hydra (cũ): {"hydra:member":[...]}
	var hydra struct {
		HydraMember []mailTmMessage `json:"hydra:member"`
	}
	if err := json.Unmarshal(body, &hydra); err == nil && len(hydra.HydraMember) > 0 {
		messages = hydra.HydraMember
	} else {
		// Thử parse array thẳng (mới): [...]
		if err := json.Unmarshal(body, &messages); err != nil {
			return "", nil // non-JSON → coi như inbox rỗng
		}
	}

	for _, msg := range messages {
		// Thử extract từ Subject trước — nhanh hơn, không cần gọi getMessage
		if code := ExtractCode(msg.Subject); code != "" {
			return code, nil
		}
		if code := ExtractCode(msg.Intro); code != "" {
			return code, nil
		}
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
func (m *MailTm) getMessage(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mailTmBaseURL+"/messages/"+id, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.token)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	var msg struct {
		HTML []string `json:"html"`
		Text string   `json:"text"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", err
	}

	if len(msg.HTML) > 0 && msg.HTML[0] != "" {
		return msg.HTML[0], nil
	}
	return msg.Text, nil
}

// randomMailTmPass tạo mật khẩu gồm chữ hoa + số.
func randomMailTmPass(n int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
