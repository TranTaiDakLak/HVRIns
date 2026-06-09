// altmails.go — AltMails.com service (tempmail.altmails.com REST + CSRF)
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const altMailsBaseURL = "https://tempmail.altmails.com"

// AltMails implements email.Service cho tempmail.altmails.com
type AltMails struct {
	client    *http.Client
	email     string
	csrfToken string
}

// NewAltMails tạo AltMails service.
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewAltMails(proxyStr string) *AltMails {
	jar, _ := cookiejar.New(nil)
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	client.Jar = jar
	return &AltMails{client: client}
}

// CreateEmail lấy email ngẫu nhiên từ GET /random-email-address.
func (a *AltMails) CreateEmail(ctx context.Context) (string, error) {
	// Bước 1: Lấy CSRF token từ homepage
	if err := a.fetchCSRF(ctx); err != nil {
		return "", err
	}

	// Bước 2: Lấy email ngẫu nhiên
	req, err := http.NewRequestWithContext(ctx, "GET",
		altMailsBaseURL+"/random-email-address", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json, text/plain")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("X-CSRF-TOKEN", a.csrfToken)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("altmails get email: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 32*1024)

	// Response có thể là JSON {"email":"..."} hoặc plain text
	var result struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &result); err == nil && result.Email != "" {
		a.email = result.Email
	} else {
		// Plain text
		raw := strings.TrimSpace(string(body))
		if raw == "" || !strings.Contains(raw, "@") {
			return "", fmt.Errorf("altmails get email: unexpected response: %s", string(body))
		}
		a.email = raw
	}
	return a.email, nil
}

// fetchCSRF lấy CSRF token từ homepage (meta tag hoặc cookie XSRF-TOKEN).
func (a *AltMails) fetchCSRF(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", altMailsBaseURL+"/", nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "text/html")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("altmails csrf: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	// Tìm <meta name="csrf-token" content="...">
	re := regexp.MustCompile(`(?i)<meta\s+name=["']csrf-token["']\s+content=["']([^"']+)["']`)
	matches := re.FindSubmatch(body)
	if len(matches) >= 2 {
		a.csrfToken = string(matches[1])
		return nil
	}

	// Thử tìm XSRF-TOKEN từ cookie jar
	for _, cookie := range a.client.Jar.Cookies(resp.Request.URL) {
		if cookie.Name == "XSRF-TOKEN" {
			a.csrfToken = cookie.Value
			return nil
		}
	}

	// Không tìm thấy CSRF — tiếp tục không có token (một số endpoint không yêu cầu)
	return nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (a *AltMails) GetEmail() string { return a.email }

// Close cleanup resources.
func (a *AltMails) Close() {}

// WaitForCode poll OTP từ altmails inbox.
func (a *AltMails) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if a.email == "" {
		return "", fmt.Errorf("altmails: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := a.pollOnce(ctx)
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

	return "", fmt.Errorf("altmails: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy danh sách mail và extract code.
func (a *AltMails) pollOnce(ctx context.Context) (string, error) {
	fetchURL := altMailsBaseURL + "/fetch-emails/" + a.email
	req, err := http.NewRequestWithContext(ctx, "POST", fetchURL,
		strings.NewReader(""))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("X-CSRF-TOKEN", a.csrfToken)

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var msgs []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &msgs); err != nil {
		return "", nil // API trả {} khi rỗng thay vì [] → coi như inbox rỗng
	}

	for _, msg := range msgs {
		content, err := a.getMailContent(ctx, msg.ID)
		if err != nil {
			continue
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// getMailContent lấy nội dung email theo id.
func (a *AltMails) getMailContent(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		altMailsBaseURL+"/view/"+id, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/html,application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)
	return string(body), nil
}
