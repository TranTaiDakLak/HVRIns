// temporary_mail_net.go — GuerrillaMail service (guerrillamail.com)
// Thay thế temporary-mail.net vì site đó bị Cloudflare block toàn bộ request từ Go HTTP.
// GuerrillaMail cung cấp REST API công khai, không yêu cầu JS, không có protection.
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const guerrillaBaseURL = "https://api.guerrillamail.com/ajax.php"

type guerrillaGetEmailResp struct {
	EmailAddr string `json:"email_addr"`
	SidToken  string `json:"sid_token"`
}

type guerrillaCheckResp struct {
	List []struct {
		MailID   string `json:"mail_id"`
		MailBody string `json:"mail_body"`
	} `json:"list"`
}

// GuerrillaMail implements email.Service dùng api.guerrillamail.com
type GuerrillaMail struct {
	client   *http.Client
	email    string
	sidToken string
}

// NewTemporaryMailNet tạo GuerrillaMail service (giữ tên constructor để không break caller)
func NewTemporaryMailNet(proxyStr string) *GuerrillaMail {
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	return &GuerrillaMail{client: c}
}

// CreateEmail tạo địa chỉ email random qua GuerrillaMail API
func (g *GuerrillaMail) CreateEmail(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", guerrillaBaseURL+"?f=get_email_address", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("guerrillamail: lỗi kết nối: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var parsed guerrillaGetEmailResp
	if err := json.Unmarshal(body, &parsed); err != nil || parsed.EmailAddr == "" {
		return "", fmt.Errorf("guerrillamail: không parse được email từ response")
	}

	g.email = parsed.EmailAddr
	g.sidToken = parsed.SidToken
	return g.email, nil
}

// GetEmail trả về địa chỉ email guerrillamail đã tạo (trống nếu chưa gọi CreateEmail).
func (g *GuerrillaMail) GetEmail() string { return g.email }

// Close giải phóng tài nguyên — GuerrillaMail không giữ kết nối nên là no-op.
func (g *GuerrillaMail) Close() {}

// WaitForCode poll OTP từ GuerrillaMail inbox
func (g *GuerrillaMail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := g.checkInbox(ctx)
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
	return "", fmt.Errorf("guerrillamail: không nhận được OTP sau %d lần thử", maxRetry)
}

// checkInbox — GET /ajax.php?f=check_email&seq=0&sid_token=TOKEN
func (g *GuerrillaMail) checkInbox(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s?f=check_email&seq=0&sid_token=%s", guerrillaBaseURL, g.sidToken)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 512*1024)

	var parsed guerrillaCheckResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}

	for _, mail := range parsed.List {
		if code := ExtractCode(mail.MailBody); code != "" {
			return code, nil
		}
	}
	return "", nil
}
