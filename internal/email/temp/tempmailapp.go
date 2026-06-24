// tempmailapp.go — temp-mail.app service (Hono RPC JSON API, no auth)
//
// Flow (xác nhận qua research 2026-06-19):
//   1. Generate random 32-char hex "visitor-id" / part
//   2. GET /api/mail/address?part={id}  (header visitor-id: {id}) → {address}
//   3. GET /api/mail/list?part={localPart}                          → {message:[{subject,content}]}
//
// KHÔNG cần key, KHÔNG login, KHÔNG Cloudflare.
package temp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const (
	tempMailAppBaseURL = "https://temp-mail.app"
	tempMailAppUA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

// TempMailApp implements email.Service cho temp-mail.app.
type TempMailApp struct {
	client    *http.Client
	email     string
	visitorID string // UUID session key — phải truyền cả trong URL và header
	localPart string // phần trước @ — dùng cho inbox polling
}

// NewTempMailApp tạo TempMailApp service.
func NewTempMailApp(proxyStr string) *TempMailApp {
	return &TempMailApp{client: proxy.CreateClient(proxyStr, 30*time.Second)}
}

// randomHexID tạo 32-char hex string ngẫu nhiên làm visitor-id.
func randomHexID() string {
	b := make([]byte, 16)
	for i := range b {
		b[i] = byte(rand.Intn(256))
	}
	return hex.EncodeToString(b)
}

// CreateEmail: generate visitor-id → GET /api/mail/address?part={id}.
func (t *TempMailApp) CreateEmail(ctx context.Context) (string, error) {
	t.visitorID = randomHexID()

	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/mail/address?part=%s", tempMailAppBaseURL, t.visitorID), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", tempMailAppUA)
	req.Header.Set("visitor-id", t.visitorID)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", tempMailAppBaseURL+"/")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tempmailapp create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var result struct {
		Address string `json:"address"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Address == "" {
		return "", fmt.Errorf("tempmailapp create: no address (HTTP %d) — body: %.200s", resp.StatusCode, body)
	}
	t.email = result.Address
	if parts := strings.SplitN(t.email, "@", 2); len(parts) == 2 {
		t.localPart = parts[0]
	}
	return t.email, nil
}

// GetEmail trả về địa chỉ đã tạo.
func (t *TempMailApp) GetEmail() string { return t.email }

// Close no-op.
func (t *TempMailApp) Close() {}

// WaitForCode poll OTP từ inbox.
func (t *TempMailApp) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmailapp: chưa tạo email")
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
	return "", fmt.Errorf("tempmailapp: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempMailApp) pollOnce(ctx context.Context) (string, error) {
	// QUAN TRỌNG: inbox query bằng visitorID (session key) — KHÔNG phải localPart.
	// Xác nhận live: part=visitorID có mail, part=localPart luôn rỗng → bug cũ kẹt OTP.
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/api/mail/list?part=%s", tempMailAppBaseURL, t.visitorID), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", tempMailAppUA)
	req.Header.Set("visitor-id", t.visitorID)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", tempMailAppBaseURL+"/")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	var result struct {
		Message []struct {
			Subject string `json:"subject"`
			Content string `json:"content"` // HTML content inline
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", nil
	}
	for _, msg := range result.Message {
		if code := ExtractCode(msg.Subject); code != "" {
			return code, nil
		}
		if code := ExtractCode(msg.Content); code != "" {
			return code, nil
		}
	}
	return "", nil
}
