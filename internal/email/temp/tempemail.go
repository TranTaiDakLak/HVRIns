// tempemail.go — TempEmail.co service (server-assigned email, JSON + HTML inbox)
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const tempEmailCoBaseURL = "https://tempemail.co"

var tempEmailCoDataIDRe = regexp.MustCompile(`data-id=['"'](\d+)['"']`)

// TempEmailCo implements email.Service cho tempemail.co
type TempEmailCo struct {
	client *http.Client
	email  string
}

// NewTempEmailCo tạo TempEmailCo service.
func NewTempEmailCo(proxyStr string) *TempEmailCo {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &TempEmailCo{client: client}
}

// CreateEmail lấy địa chỉ email ngẫu nhiên từ server (retry cho đến khi nhận @tempemail.co).
func (m *TempEmailCo) CreateEmail(ctx context.Context) (string, error) {
	for i := 0; i < 10; i++ {
		addr, err := m.requestRandom(ctx)
		if err != nil {
			return "", err
		}
		if addr != "" {
			m.email = addr
			return m.email, nil
		}
	}
	return "", fmt.Errorf("tempemail.co: không lấy được địa chỉ @tempemail.co sau 10 lần thử")
}

// requestRandom gọi GET /mail/random, trả về address nếu là @tempemail.co.
func (m *TempEmailCo) requestRandom(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tempEmailCoBaseURL+"/mail/random", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://tempemail.co/")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 16*1024)

	var result struct {
		Result  bool   `json:"result"`
		Address string `json:"address"`
		ID      string `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Result {
		return "", nil
	}

	addr := result.Address
	if addr == "" {
		addr = result.ID
	}
	if len(addr) > 0 {
		// Only accept @tempemail.co addresses (server may assign others)
		if len(addr) >= 13 && addr[len(addr)-13:] == "@tempemail.co" {
			return addr, nil
		}
	}
	return "", nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *TempEmailCo) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *TempEmailCo) Close() {}

// WaitForCode poll OTP từ tempemail.co inbox.
func (m *TempEmailCo) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" {
		return "", fmt.Errorf("tempemail.co: chưa tạo email")
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

	return "", fmt.Errorf("tempemail.co: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy danh sách ID và extract code.
func (m *TempEmailCo) pollOnce(ctx context.Context) (string, error) {
	listURL := tempEmailCoBaseURL + "/get-mails?mail_id=" + url.QueryEscape(m.email) + "&unseen=0&is_new=1"
	req, err := http.NewRequestWithContext(ctx, "GET", listURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://tempemail.co/")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var result struct {
		Result bool   `json:"result"`
		Mails  string `json:"mails"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Result {
		return "", nil
	}

	// Extract data-id from HTML content
	matches := tempEmailCoDataIDRe.FindAllStringSubmatch(result.Mails, -1)
	for _, match := range matches {
		if len(match) > 1 {
			content, err := m.getMessage(ctx, match[1])
			if err != nil {
				continue
			}
			if code := ExtractCode(content); code != "" {
				return code, nil
			}
		}
	}
	return "", nil
}

// getMessage lấy nội dung email theo ID.
func (m *TempEmailCo) getMessage(ctx context.Context, id string) (string, error) {
	msgURL := tempEmailCoBaseURL + "/mail/info?id=" + id
	req, err := http.NewRequestWithContext(ctx, "GET", msgURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://tempemail.co/")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)

	var msg struct {
		Result bool `json:"result"`
		Mail   struct {
			TextHTML  string `json:"textHtml"`
			TextPlain string `json:"textPlain"`
		} `json:"mail"`
	}
	if err := json.Unmarshal(body, &msg); err != nil || !msg.Result {
		return "", fmt.Errorf("tempemail.co getMessage: unexpected response")
	}

	if msg.Mail.TextHTML != "" {
		return msg.Mail.TextHTML, nil
	}
	return msg.Mail.TextPlain, nil
}
