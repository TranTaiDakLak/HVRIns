// tempmailorgpremium.go — temp-mail.org Premium service (JSON-RPC, Cloudflare protected)
// Port từ C# TempMailOrgPremiumAPI. Flow: GET homepage warmup CF → RPC user.login → sid →
// RPC getdomains → pick random → RPC mailbox.new → email. Poll inbox qua mailbox.messagesv2.
// LƯU Ý: Cloudflare có thể chặn, login dùng tài khoản shared từ C# gốc.
package temp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const (
	tempMailOrgPremiumBaseURL = "https://temp-mail.org"
	tempMailOrgPremiumRPCURL  = "https://papi2.temp-mail.org/rpc/"

	// Shared credentials port từ C# — đã leak qua git history. Override bằng env
	// TEMPMAILORG_USER / TEMPMAILORG_PASSWORD trên production.
	defaultTempMailOrgPremiumUser     = "havu88.i2b@gmail.com"
	defaultTempMailOrgPremiumPassword = "Khanhlinh@3007@@"

	tempMailOrgPremiumUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36"
)

func tempMailOrgPremiumUser_() string {
	if v := os.Getenv("TEMPMAILORG_USER"); v != "" {
		return v
	}
	return defaultTempMailOrgPremiumUser
}

func tempMailOrgPremiumPassword_() string {
	if v := os.Getenv("TEMPMAILORG_PASSWORD"); v != "" {
		return v
	}
	return defaultTempMailOrgPremiumPassword
}

// TempMailOrgPremium implements email.Service cho temp-mail.org (Premium).
type TempMailOrgPremium struct {
	client *http.Client
	email  string
	sid    string
}

// NewTempMailOrgPremium tạo provider.
func NewTempMailOrgPremium(proxyStr string) *TempMailOrgPremium {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	c.Jar = jar
	return &TempMailOrgPremium{client: c}
}

// rpcCall gọi JSON-RPC 2.0 tới papi2 endpoint với params đã format.
func (t *TempMailOrgPremium) rpcCall(ctx context.Context, method string, params map[string]interface{}) (map[string]interface{}, error) {
	payload, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      "jsonrpc",
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", tempMailOrgPremiumRPCURL, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", tempMailOrgPremiumUA)
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("Accept-Language", "vi,en;q=0.9")
	req.Header.Set("Origin", tempMailOrgPremiumBaseURL)
	req.Header.Set("Referer", tempMailOrgPremiumBaseURL+"/")
	req.Header.Set("sec-ch-ua", `"Chromium";v="146", "Not-A.Brand";v="24", "Google Chrome";v="146"`)
	req.Header.Set("sec-ch-ua-mobile", "?0")
	req.Header.Set("sec-ch-ua-platform", `"Windows"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-site")
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rpc %s: %w", method, err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 512*1024)
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" || strings.HasPrefix(trimmed, "<") {
		return nil, fmt.Errorf("cloudflare blocked rpc %s: not JSON — body: %.200s", method, body)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("rpc %s parse: %w — body: %.200s", method, err, body)
	}
	return result, nil
}

// CreateEmail: warmup CF → login → domains → mailbox.new.
func (t *TempMailOrgPremium) CreateEmail(ctx context.Context) (string, error) {
	// Step 1: GET homepage để lấy Cloudflare cookies
	req, _ := http.NewRequestWithContext(ctx, "GET", tempMailOrgPremiumBaseURL+"/vi/premium", nil)
	req.Header.Set("User-Agent", tempMailOrgPremiumUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tempmailorgpremium warmup: %w", err)
	}
	_ = resp.Body.Close()

	// Step 2: Login
	loginResp, err := t.rpcCall(ctx, "user.login", map[string]interface{}{
		"username": tempMailOrgPremiumUser_(),
		"password": tempMailOrgPremiumPassword_(),
		"provider": "paddle",
	})
	if err != nil {
		return "", err
	}
	res, _ := loginResp["result"].(map[string]interface{})
	if sid, ok := res["sid"].(string); ok && sid != "" {
		t.sid = sid
	} else {
		return "", fmt.Errorf("tempmailorgpremium login: no sid — resp: %v", loginResp)
	}

	// Step 3: Get random domain
	domainsResp, err := t.rpcCall(ctx, "getdomains", map[string]interface{}{"sid": t.sid})
	if err != nil {
		return "", err
	}
	dRes, _ := domainsResp["result"].(map[string]interface{})
	domainsArr, _ := dRes["domains"].([]interface{})
	if len(domainsArr) == 0 {
		return "", fmt.Errorf("tempmailorgpremium: no domains")
	}
	domain, _ := domainsArr[rand.Intn(len(domainsArr))].(string)
	if domain == "" {
		return "", fmt.Errorf("tempmailorgpremium: empty domain")
	}

	// Step 4: Create mailbox
	user := realisticLocalPart()
	email := user + "@" + domain
	createResp, err := t.rpcCall(ctx, "mailbox.new", map[string]interface{}{
		"sid":   t.sid,
		"email": email,
	})
	if err != nil {
		return "", err
	}
	cRes, _ := createResp["result"].(map[string]interface{})
	if resultEmail, ok := cRes["email"].(string); ok && resultEmail != "" {
		t.email = resultEmail
	} else {
		return "", fmt.Errorf("tempmailorgpremium: mailbox.new failed — resp: %v", createResp)
	}
	return t.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (t *TempMailOrgPremium) GetEmail() string { return t.email }

// Close xoá mailbox qua mailbox.delete (best-effort).
func (t *TempMailOrgPremium) Close() {
	if t.sid == "" || t.email == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = t.rpcCall(ctx, "mailbox.delete", map[string]interface{}{
		"sid":   t.sid,
		"email": t.email,
	})
}

// WaitForCode poll OTP.
func (t *TempMailOrgPremium) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmailorgpremium: chưa tạo email")
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
	return "", fmt.Errorf("tempmailorgpremium: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempMailOrgPremium) pollOnce(ctx context.Context) (string, error) {
	resp, err := t.rpcCall(ctx, "mailbox.messagesv2", map[string]interface{}{
		"sid":   t.sid,
		"email": t.email,
		"page":  1,
		"limit": 10,
	})
	if err != nil {
		return "", err
	}
	res, _ := resp["result"].(map[string]interface{})
	mails, _ := res["mails"].([]interface{})
	for _, m := range mails {
		msg, _ := m.(map[string]interface{})
		mailID, _ := msg["mail_id"].(string)
		subject, _ := msg["mail_subject"].(string)
		content, _ := t.getMessage(ctx, mailID)
		if content == "" {
			content = subject
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

func (t *TempMailOrgPremium) getMessage(ctx context.Context, mailID string) (string, error) {
	resp, err := t.rpcCall(ctx, "mail.get", map[string]interface{}{
		"sid":  t.sid,
		"mail": mailID,
	})
	if err != nil {
		return "", err
	}
	res, _ := resp["result"].(map[string]interface{})
	message, _ := res["message"].(map[string]interface{})
	if html, ok := message["mail_html"].(string); ok && html != "" {
		return html, nil
	}
	if text, ok := message["mail_text"].(string); ok {
		return text, nil
	}
	return "", nil
}
