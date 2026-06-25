// tempmail100.go — tempmail100.com service (JWT login + premium mailboxes)
// Port từ C# Tempmail100API. Flow: POST /login → JWT → GET /api/user/domains → random pick →
// POST /api/mailboxes để tạo → GET /api/inbox poll → GET /api/emails/content/{uuid} đọc body.
package temp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const tempMail100BaseURL = "https://tempmail100.com"

// Login credentials — port từ C# hardcoded. Override bằng env TEMPMAIL100_EMAIL / TEMPMAIL100_PASSWORD.
// Defaults đã bị leak qua git history → nên rotate + override qua env trên máy chạy production.
const (
	defaultTempMail100Email    = "tranxuanbacnd@gmail.com"
	defaultTempMail100Password = "Khanhlinh@3007!@@"
)

func tempMail100Email() string {
	if v := os.Getenv("TEMPMAIL100_EMAIL"); v != "" {
		return v
	}
	return defaultTempMail100Email
}

func tempMail100Password() string {
	if v := os.Getenv("TEMPMAIL100_PASSWORD"); v != "" {
		return v
	}
	return defaultTempMail100Password
}

// TempMail100 implements email.Service cho tempmail100.com.
type TempMail100 struct {
	client *http.Client
	email  string
	token  string
}

// NewTempMail100 tạo TempMail100 service.
func NewTempMail100(proxyStr string) *TempMail100 {
	return &TempMail100{client: proxy.CreateClient(proxyStr, 30*time.Second)}
}

// CreateEmail: login → lấy domain → tạo mailbox.
func (t *TempMail100) CreateEmail(ctx context.Context) (string, error) {
	if err := t.login(ctx); err != nil {
		return "", fmt.Errorf("tempmail100 login: %w", err)
	}
	domain, err := t.pickDomain(ctx)
	if err != nil {
		return "", fmt.Errorf("tempmail100 domains: %w", err)
	}
	username := realisticLocalPart()
	if err := t.createMailbox(ctx, username, domain); err != nil {
		return "", fmt.Errorf("tempmail100 mailbox: %w", err)
	}
	t.email = username + "@" + domain
	return t.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (t *TempMail100) GetEmail() string { return t.email }

// Close xoá mailbox (port C# DeleteMailbox).
func (t *TempMail100) Close() {
	if t.token == "" || t.email == "" {
		return
	}
	payload, _ := json.Marshal(map[string][]string{"address": {t.email}})
	req, _ := http.NewRequest("DELETE", tempMail100BaseURL+"/api/mailboxes", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", t.token)
	resp, err := t.client.Do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
}

func (t *TempMail100) login(ctx context.Context) error {
	payload, _ := json.Marshal(map[string]string{
		"email":    tempMail100Email(),
		"password": tempMail100Password(),
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", tempMail100BaseURL+"/login", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Referer", tempMail100BaseURL+"/premium/login")
	req.Header.Set("Origin", tempMail100BaseURL)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)
	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse: %w — body: %.200s", err, body)
	}
	if result.Data.Token == "" {
		return fmt.Errorf("empty token — body: %.200s", body)
	}
	t.token = result.Data.Token
	return nil
}

func (t *TempMail100) pickDomain(ctx context.Context) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", tempMail100BaseURL+"/api/user/domains", nil)
	req.Header.Set("Authorization", t.token)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)
	var result struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if len(result.Data) == 0 {
		return "", fmt.Errorf("no domains")
	}
	return result.Data[rand.Intn(len(result.Data))], nil
}

func (t *TempMail100) createMailbox(ctx context.Context, name, domain string) error {
	payload, _ := json.Marshal(map[string]string{"name": name, "domain": domain})
	req, _ := http.NewRequestWithContext(ctx, "POST", tempMail100BaseURL+"/api/mailboxes", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("Authorization", t.token)
	req.Header.Set("Referer", tempMail100BaseURL+"/premium/mailboxes")
	req.Header.Set("Origin", tempMail100BaseURL)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)
	var result struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		return fmt.Errorf("create failed code=%d — body: %.200s", result.Code, body)
	}
	return nil
}

// WaitForCode poll OTP.
func (t *TempMail100) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmail100: chưa tạo email")
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
	return "", fmt.Errorf("tempmail100: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempMail100) pollOnce(ctx context.Context) (string, error) {
	inboxURL := tempMail100BaseURL + "/api/inbox?addr=" + url.QueryEscape(t.email)
	req, _ := http.NewRequestWithContext(ctx, "GET", inboxURL, nil)
	req.Header.Set("Authorization", t.token)
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
		return "", err
	}
	for _, msg := range result.Data.List {
		content, _ := t.getMessage(ctx, msg.UUID)
		if content == "" {
			content = msg.Subject
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

func (t *TempMail100) getMessage(ctx context.Context, uuid string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", tempMail100BaseURL+"/api/emails/content/"+uuid, nil)
	req.Header.Set("Authorization", t.token)
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
