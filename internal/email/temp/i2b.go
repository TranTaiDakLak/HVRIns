// i2b.go — I2bMail service (mail.i2b.vn)
// API không cần auth — chỉ cần địa chỉ email.
// Flow:
//   1. CreateEmail: ghép username ngẫu nhiên + @i2b.vn
//   2. WaitForCode: GET /api/mail/messages?to={email} → list messages → extract OTP từ subject/body
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

const i2bBase   = "https://mail.i2b.vn"
const i2bDomain = "i2b.vn"

// I2bMail implements email.Service cho mail.i2b.vn.
type I2bMail struct {
	client *http.Client
	email  string
}

// NewI2bMail tạo I2bMail service.
func NewI2bMail(proxyStr string) *I2bMail {
	return &I2bMail{client: proxy.CreateClient(proxyStr, 30*time.Second)}
}

func (w *I2bMail) GetEmail() string { return w.email }
func (w *I2bMail) Close()           {}

// CreateEmail ghép username ngẫu nhiên với domain i2b.vn.
func (w *I2bMail) CreateEmail(_ context.Context) (string, error) {
	w.email = realisticEmail(i2bDomain)
	return w.email, nil
}

func (w *I2bMail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if w.email == "" {
		return "", fmt.Errorf("i2b: email chưa được tạo")
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if code, _ := w.pollOnce(ctx); code != "" {
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
	return "", fmt.Errorf("i2b: không nhận được OTP sau %d lần thử", maxRetry)
}

type i2bMessage struct {
	Key     string `json:"key"`
	Subject string `json:"subject"`
	From    string `json:"from"`
	Body    string `json:"body"`
}

func (w *I2bMail) pollOnce(ctx context.Context) (string, error) {
	apiURL := i2bBase + "/api/mail/messages?to=" + url.QueryEscape(w.email)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)

	var wrapped struct {
		Messages []i2bMessage `json:"messages"`
	}
	if json.Unmarshal(body, &wrapped) == nil && len(wrapped.Messages) > 0 {
		return extractI2bCode(wrapped.Messages), nil
	}
	var list []i2bMessage
	if json.Unmarshal(body, &list) == nil {
		return extractI2bCode(list), nil
	}
	return "", nil
}

func extractI2bCode(msgs []i2bMessage) string {
	for _, m := range msgs {
		if !isFacebookOTPSender(m.From) {
			continue
		}
		if code := ExtractCode(m.Subject); code != "" {
			return code
		}
		if code := ExtractCode(m.Body); code != "" {
			return code
		}
	}
	return ""
}
