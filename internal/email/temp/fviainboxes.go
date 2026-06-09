// fviainboxes.go — FviaInboxes.com service (client-side email, Bearer auth)
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const fviaInboxesBaseURL = "https://fviainboxes.com"

// Shared token port từ C# — đã leak qua git history. Override bằng env FVIAINBOXES_TOKEN trên production.
const defaultFviaInboxesToken = "af2b556e5e719052ca9193bace296b4fe9015bdc6c2c6ec28447d57c56187941"

func fviaInboxesToken() string {
	if v := os.Getenv("FVIAINBOXES_TOKEN"); v != "" {
		return v
	}
	return defaultFviaInboxesToken
}

// FviaInboxes implements email.Service cho fviainboxes.com
type FviaInboxes struct {
	client *http.Client
	user   string
	domain string
	email  string
}

// NewFviaInboxes tạo FviaInboxes service.
func NewFviaInboxes(proxyStr string) *FviaInboxes {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &FviaInboxes{client: client}
}

// CreateEmail sinh địa chỉ email client-side (domain hotmail.com — IMAP relay service).
// fviainboxes.com là IMAP proxy cho Hotmail/Outlook mua từ Store1s, không phải disposable email.
// User thường kết hợp với store1s để lấy email rồi dùng fviainboxes để đọc OTP.
func (m *FviaInboxes) CreateEmail(_ context.Context) (string, error) {
	m.user = realisticLocalPart()
	m.domain = "hotmail.com"
	m.email = m.user + "@" + m.domain
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *FviaInboxes) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *FviaInboxes) Close() {}

// WaitForCode poll OTP từ fviainboxes.com inbox.
func (m *FviaInboxes) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" {
		return "", fmt.Errorf("fviainboxes: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := m.pollOnce(ctx)
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

	return "", fmt.Errorf("fviainboxes: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy inbox và extract code.
func (m *FviaInboxes) pollOnce(ctx context.Context) (string, error) {
	listURL := fviaInboxesBaseURL + "/messages?username=" + url.QueryEscape(m.user) + "&domain=" + url.QueryEscape(m.domain)
	req, err := http.NewRequestWithContext(ctx, "GET", listURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+fviaInboxesToken())
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var result struct {
		Result []struct {
			ID      string `json:"id"`
			From    string `json:"from"`
			Subject string `json:"subject"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("fviainboxes inbox parse: %w", err)
	}

	for _, msg := range result.Result {
		content, err := m.getMessage(ctx, msg.ID)
		if err != nil {
			continue
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
		if code := ExtractCode(msg.Subject); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// getMessage lấy nội dung email theo ID.
func (m *FviaInboxes) getMessage(ctx context.Context, id string) (string, error) {
	msgURL := fviaInboxesBaseURL + "/message?username=" + url.QueryEscape(m.user) + "&domain=" + url.QueryEscape(m.domain) + "&id=" + url.QueryEscape(id)
	req, err := http.NewRequestWithContext(ctx, "GET", msgURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+fviaInboxesToken())
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)

	content := string(body)
	// Unescape nếu là JSON string
	content = strings.ReplaceAll(content, `\n`, "\n")
	content = strings.ReplaceAll(content, `\t`, "\t")
	return content, nil
}
