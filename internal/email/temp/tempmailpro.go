// tempmailpro.go — tempmailpro.io service (REST, client-side address gen)
//
// Flow (xác nhận live qua agent 2026-06-19):
//   1. Gen địa chỉ client-side: rand8 + "@tempmailpro.io" (1 domain duy nhất)
//   2. POST /api/emails/activate-session {"address":addr} → {"success":true}
//      (phải re-POST mỗi ~10 phút để giữ session sống; gate: chưa activate → inbox 404)
//   3. GET /api/emails/guest/{address}            → [{_id,from,subject,text,html}]
//   4. GET /api/emails/guest/{address}/{_id}      → message đầy đủ (fallback)
//
// KHÔNG cần key, KHÔNG login, KHÔNG Cloudflare. Rate limit: activate 30/60s, guest 500/900s.
package temp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const (
	tempMailProBaseURL = "https://tempmailpro.io"
	tempMailProDomain  = "tempmailpro.io"
	tempMailProUA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

// TempMailPro implements email.Service cho tempmailpro.io.
type TempMailPro struct {
	client      *http.Client
	email       string
	lastActived time.Time // thời điểm activate-session gần nhất (re-activate sau ~8 phút)
}

// NewTempMailPro tạo TempMailPro service.
func NewTempMailPro(proxyStr string) *TempMailPro {
	return &TempMailPro{client: proxy.CreateClient(proxyStr, 30*time.Second)}
}

func (t *TempMailPro) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", tempMailProUA)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", tempMailProBaseURL)
	req.Header.Set("Referer", tempMailProBaseURL+"/")
}

// randTempMailProLocal sinh 8 ký tự [a-z0-9].
func randTempMailProLocal() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// activateSession đăng ký địa chỉ với server (bắt buộc trước khi đọc inbox).
func (t *TempMailPro) activateSession(ctx context.Context) error {
	payload, _ := json.Marshal(map[string]string{"address": t.email})
	req, err := http.NewRequestWithContext(ctx, "POST",
		tempMailProBaseURL+"/api/emails/activate-session", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	t.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 16*1024)

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Success {
		return fmt.Errorf("activate-session fail (HTTP %d): %.150s", resp.StatusCode, body)
	}
	t.lastActived = time.Now()
	return nil
}

// CreateEmail: gen địa chỉ → activate-session.
func (t *TempMailPro) CreateEmail(ctx context.Context) (string, error) {
	t.email = randTempMailProLocal() + "@" + tempMailProDomain
	if err := t.activateSession(ctx); err != nil {
		return "", fmt.Errorf("tempmailpro create: %w", err)
	}
	return t.email, nil
}

// GetEmail trả về địa chỉ đã tạo.
func (t *TempMailPro) GetEmail() string { return t.email }

// Close no-op.
func (t *TempMailPro) Close() {}

// WaitForCode poll OTP từ inbox.
func (t *TempMailPro) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmailpro: chưa tạo email")
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		// Re-activate nếu session sắp hết hạn (>8 phút) để inbox không bị 404.
		if time.Since(t.lastActived) > 8*time.Minute {
			_ = t.activateSession(ctx)
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
	return "", fmt.Errorf("tempmailpro: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempMailPro) pollOnce(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		tempMailProBaseURL+"/api/emails/guest/"+t.email, nil)
	if err != nil {
		return "", err
	}
	t.setHeaders(req)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	var msgs []struct {
		ID      string `json:"_id"`
		Subject string `json:"subject"`
		Text    string `json:"text"`
		HTML    string `json:"html"`
	}
	if err := json.Unmarshal(body, &msgs); err != nil {
		return "", nil
	}
	for _, m := range msgs {
		if code := ExtractCode(m.Subject); code != "" {
			return code, nil
		}
		if code := ExtractCode(m.Text); code != "" {
			return code, nil
		}
		if code := ExtractCode(m.HTML); code != "" {
			return code, nil
		}
		// Fallback: lấy full message nếu list không chứa body
		if m.ID != "" {
			if content, _ := t.getMessage(ctx, m.ID); content != "" {
				if code := ExtractCode(content); code != "" {
					return code, nil
				}
			}
		}
	}
	return "", nil
}

func (t *TempMailPro) getMessage(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		tempMailProBaseURL+"/api/emails/guest/"+t.email+"/"+id, nil)
	if err != nil {
		return "", err
	}
	t.setHeaders(req)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	var msg struct {
		HTML string `json:"html"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", err
	}
	if msg.HTML != "" {
		return msg.HTML, nil
	}
	return msg.Text, nil
}
