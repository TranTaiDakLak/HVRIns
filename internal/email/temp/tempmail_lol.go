// tempmail_lol.go — TempMailLOL service (api.tempmail.lol/v2)
// Mapping từ WeBM TempmailLOL.cs + TempmailLolAPI.cs
package temp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const tempmailLolBaseURL = "https://api.tempmail.lol/v2"

// TempMailLolDomains — danh sách domain tùy chọn (để trống = API tự chọn ngẫu nhiên).
// Mapping từ WeBM: khi domain == "RANDOM" hoặc rỗng → truyền null để server pick.
var TempMailLolDomains []string

// TempMailLol implements email.Service cho api.tempmail.lol
type TempMailLol struct {
	client *http.Client
	email  string
	token  string
	apiKey string // optional — free tier không cần
}

// NewTempMailLol tạo TempMailLol service.
// apiKey: optional Bearer token (để trống nếu dùng free tier).
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewTempMailLol(apiKey, proxyStr string) *TempMailLol {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &TempMailLol{
		client: client,
		apiKey: apiKey,
	}
}

// CreateEmail tạo email tạm qua POST /v2/inbox/create.
// Mapping từ WeBM TempmailLOL.CreateRandomEmail() — dùng domain=null để API tự chọn.
func (t *TempMailLol) CreateEmail(ctx context.Context) (string, error) {
	// domain=null → server pick ngẫu nhiên (giống WeBM RANDOM mode)
	payload := map[string]interface{}{
		"domain":  nil,
		"captcha": nil,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", tempmailLolBaseURL+"/inbox/create", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if t.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tempmail.lol create: %w", err)
	}
	defer resp.Body.Close()

	data, _ := httpx.ReadBody(resp.Body, 0)

	var result struct {
		Address string `json:"address"`
		Token   string `json:"token"`
	}
	if err := json.Unmarshal(data, &result); err != nil || result.Address == "" {
		return "", fmt.Errorf("tempmail.lol create: unexpected response: %s", string(data))
	}

	t.email = result.Address
	t.token = result.Token
	return t.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (t *TempMailLol) GetEmail() string { return t.email }

// Close cleanup resources.
func (t *TempMailLol) Close() {}

// WaitForCode poll OTP code từ tempmail.lol inbox.
// Mapping từ WeBM TempmailLolAPI.GetMailboxes()
func (t *TempMailLol) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.token == "" {
		return "", fmt.Errorf("tempmail.lol: inbox chưa được tạo")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := t.pollOnce(ctx)
		if err != nil {
			// token expired → báo lỗi rõ ràng
			if strings.Contains(err.Error(), "expired") {
				return "", fmt.Errorf("tempmail.lol: inbox token đã hết hạn")
			}
			// network error → tiếp tục retry
		} else if code != "" {
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

	return "", fmt.Errorf("tempmail.lol: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce gọi GET /v2/inbox?token=... một lần, trả code nếu có.
func (t *TempMailLol) pollOnce(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		tempmailLolBaseURL+"/inbox?token="+t.token, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	if t.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+t.apiKey)
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, _ := httpx.ReadBody(resp.Body, 0)

	var inbox struct {
		Expired bool `json:"expired"`
		Emails  []struct {
			ID      string `json:"_id"`
			From    string `json:"from"`
			Subject string `json:"subject"`
			HTML    string `json:"html"`
			Body    string `json:"body"`
		} `json:"emails"`
	}
	if err := json.Unmarshal(data, &inbox); err != nil {
		return "", fmt.Errorf("parse inbox: %w", err)
	}
	if inbox.Expired {
		return "", fmt.Errorf("inbox expired")
	}

	for _, email := range inbox.Emails {
		content := email.HTML
		if content == "" {
			content = email.Body
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}
