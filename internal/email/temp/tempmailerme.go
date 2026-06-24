// tempmailerme.go — temp-mailer.me service (token-based REST JSON API)
//
// Flow (xác nhận qua research 2026-06-19):
//   1. GET /api/mail                                    → {email, token}
//   2. GET /api/messages?email={email}&token={token}   → [{id,from,html,created_at}]
//   3. GET /api/message?id={id}                        → {html, text}  (fallback nếu inline chưa đủ)
//
// KHÔNG cần key, KHÔNG login. Email + token gắn liền nhau theo session.
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

const (
	tempMailerMeBaseURL = "https://temp-mailer.me/api"
	tempMailerMeUA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

// TempMailerMe implements email.Service cho temp-mailer.me.
type TempMailerMe struct {
	client *http.Client
	email  string
	token  string
}

// NewTempMailerMe tạo TempMailerMe service.
func NewTempMailerMe(proxyStr string) *TempMailerMe {
	return &TempMailerMe{client: proxy.CreateClient(proxyStr, 30*time.Second)}
}

func (t *TempMailerMe) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", tempMailerMeUA)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://temp-mailer.me/")
}

// CreateEmail: GET /api/mail → email + token.
func (t *TempMailerMe) CreateEmail(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tempMailerMeBaseURL+"/mail", nil)
	if err != nil {
		return "", err
	}
	t.setHeaders(req)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tempmailerme create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var result struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Email == "" || result.Token == "" {
		return "", fmt.Errorf("tempmailerme create: no email/token (HTTP %d) — body: %.200s", resp.StatusCode, body)
	}
	t.email = result.Email
	t.token = result.Token
	return t.email, nil
}

// GetEmail trả về địa chỉ đã tạo.
func (t *TempMailerMe) GetEmail() string { return t.email }

// Close no-op.
func (t *TempMailerMe) Close() {}

// WaitForCode poll OTP từ inbox.
func (t *TempMailerMe) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmailerme: chưa tạo email")
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
	return "", fmt.Errorf("tempmailerme: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempMailerMe) pollOnce(ctx context.Context) (string, error) {
	inboxURL := tempMailerMeBaseURL + "/messages?email=" +
		url.QueryEscape(t.email) + "&token=" + url.QueryEscape(t.token)
	req, err := http.NewRequestWithContext(ctx, "GET", inboxURL, nil)
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

	// Inbox = mảng trần (xác nhận live: shape=BARE-ARRAY). Parse field-agnostic:
	// quét ExtractCode trên TOÀN BỘ raw mỗi message (bắt code dù field tên subject/
	// html/text/body), + fallback getMessage(id). Không phụ thuộc 1 tên field cứng.
	var rawMsgs []json.RawMessage
	if err := json.Unmarshal(body, &rawMsgs); err != nil {
		return "", nil
	}
	for _, raw := range rawMsgs {
		// ExtractCode dùng pattern có-mỏ-neo (confirmation code/Facebook/letter-spacing)
		// nên quét nguyên blob JSON an toàn (id 18 số / epoch không khớp \b\d{4,8}\b).
		if code := ExtractCode(string(raw)); code != "" {
			return code, nil
		}
		var meta struct {
			ID interface{} `json:"id"`
		}
		_ = json.Unmarshal(raw, &meta)
		if idStr := jsonIDStr(meta.ID); idStr != "" {
			if content, _ := t.getMessage(ctx, idStr); content != "" {
				if code := ExtractCode(content); code != "" {
					return code, nil
				}
			}
		}
	}
	return "", nil
}

func (t *TempMailerMe) getMessage(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		tempMailerMeBaseURL+"/message?id="+url.QueryEscape(id), nil)
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
