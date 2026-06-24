// tempmail100free.go — tempmail100.com free (anonymous token) service
// Port từ C# Tempmail100FreeAPI. KHÔNG cần tài khoản/login.
// Flow: POST /init → anonymous token (server đôi khi trả sẵn address) →
//       POST /web/generate → GET /web/emails poll → GET /emails/content/{uuid}.
package temp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const tempMail100FreeBaseURL = "https://tempmail100.com"

// TempMail100Free implements email.Service cho tempmail100.com (free, anonymous).
// Khác bản Premium (TempMail100): không login tài khoản — dùng anonymous token
// từ POST /init, sau đó POST /web/generate để lấy địa chỉ.
type TempMail100Free struct {
	client *http.Client
	email  string
	token  string // anonymous token từ /init
}

// NewTempMail100Free tạo TempMail100Free service.
func NewTempMail100Free(proxyStr string) *TempMail100Free {
	return &TempMail100Free{client: proxy.CreateClient(proxyStr, 30*time.Second)}
}

// CreateEmail: POST /init → anonymous token → POST /web/generate → email.
func (t *TempMail100Free) CreateEmail(ctx context.Context) (string, error) {
	// Step 1: init — lấy anonymous token (server đôi khi trả sẵn address từ session cũ)
	initReq, _ := http.NewRequestWithContext(ctx, "POST",
		tempMail100FreeBaseURL+"/init", bytes.NewReader([]byte("{}")))
	initReq.Header.Set("Content-Type", "application/json;charset=UTF-8")
	initReq.Header.Set("Referer", tempMail100FreeBaseURL+"/")
	initReq.Header.Set("Origin", tempMail100FreeBaseURL)
	initReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	initReq.Header.Set("User-Agent", "Mozilla/5.0")

	initResp, err := t.client.Do(initReq)
	if err != nil {
		return "", fmt.Errorf("tempmail100free init: %w", err)
	}
	defer initResp.Body.Close()
	initBody, _ := httpx.ReadBody(initResp.Body, 64*1024)

	var initResult struct {
		Data struct {
			Token   string `json:"token"`
			Address string `json:"address"`
		} `json:"data"`
	}
	if err := json.Unmarshal(initBody, &initResult); err != nil {
		return "", fmt.Errorf("tempmail100free init parse: %w — body: %.200s", err, initBody)
	}
	if initResult.Data.Token == "" {
		return "", fmt.Errorf("tempmail100free init: empty token (HTTP %d) — body: %.200s",
			initResp.StatusCode, initBody)
	}
	t.token = initResult.Data.Token

	// Server đôi khi trả sẵn address (session cũ còn sống) → dùng luôn, bỏ /web/generate
	if initResult.Data.Address != "" {
		t.email = initResult.Data.Address
		return t.email, nil
	}

	// Step 2: generate random email
	genReq, _ := http.NewRequestWithContext(ctx, "POST",
		tempMail100FreeBaseURL+"/web/generate", bytes.NewReader([]byte("{}")))
	genReq.Header.Set("Content-Type", "application/json;charset=UTF-8")
	genReq.Header.Set("Authorization", t.token)
	genReq.Header.Set("Referer", tempMail100FreeBaseURL+"/")
	genReq.Header.Set("Origin", tempMail100FreeBaseURL)
	genReq.Header.Set("X-Requested-With", "XMLHttpRequest")
	genReq.Header.Set("User-Agent", "Mozilla/5.0")

	genResp, err := t.client.Do(genReq)
	if err != nil {
		return "", fmt.Errorf("tempmail100free generate: %w", err)
	}
	defer genResp.Body.Close()
	genBody, _ := httpx.ReadBody(genResp.Body, 64*1024)

	var genResult struct {
		Code int `json:"code"`
		Data struct {
			Email   string `json:"email"`
			Address string `json:"address"`
		} `json:"data"`
	}
	if err := json.Unmarshal(genBody, &genResult); err != nil {
		return "", fmt.Errorf("tempmail100free generate parse: %w — body: %.200s", err, genBody)
	}
	if genResult.Code != 0 {
		return "", fmt.Errorf("tempmail100free generate code=%d (HTTP %d) — body: %.200s",
			genResult.Code, genResp.StatusCode, genBody)
	}

	email := genResult.Data.Email
	if email == "" {
		email = genResult.Data.Address
	}
	if email == "" {
		return "", fmt.Errorf("tempmail100free generate: no email in response (HTTP %d) — body: %.200s",
			genResp.StatusCode, genBody)
	}
	t.email = email
	return t.email, nil
}

// GetEmail trả địa chỉ email đã tạo.
func (t *TempMail100Free) GetEmail() string { return t.email }

// Close — free provider không có API xóa mailbox.
func (t *TempMail100Free) Close() {}

// WaitForCode poll OTP từ inbox.
func (t *TempMail100Free) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmail100free: chưa tạo email")
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
	return "", fmt.Errorf("tempmail100free: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempMail100Free) pollOnce(ctx context.Context) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		tempMail100FreeBaseURL+"/web/emails", nil)
	req.Header.Set("Authorization", t.token)
	req.Header.Set("Referer", tempMail100FreeBaseURL+"/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)

	var result struct {
		Data struct {
			List []struct {
				UUID        string `json:"uuid"`
				FromAddress string `json:"fromAddress"`
				Subject     string `json:"subject"`
			} `json:"list"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("tempmail100free inbox parse: %w", err)
	}

	for _, msg := range result.Data.List {
		// Thử extract từ subject trước (không cần thêm HTTP round-trip)
		if code := ExtractCode(msg.Subject); code != "" {
			return code, nil
		}
		content, err := t.getMessage(ctx, msg.UUID)
		if err != nil {
			continue
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

func (t *TempMail100Free) getMessage(ctx context.Context, uuid string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET",
		tempMail100FreeBaseURL+"/emails/content/"+uuid, nil)
	req.Header.Set("Authorization", t.token)
	req.Header.Set("Referer", tempMail100FreeBaseURL+"/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	var result struct {
		Data struct {
			Content string `json:"content"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.Data.Content, nil
}
