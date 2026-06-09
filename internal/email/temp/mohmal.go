// mohmal.go — Mohmal service (mohmal.com)
// Mapping từ WeBM VerifyCloneVIP/API/TempMailServer/MohmalcomAPI.cs
package temp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const mohmalBaseURL = "https://www.mohmal.com"

var mohmalMsgIDRegex = regexp.MustCompile(`data-msg-id="(\d+)"`)

// Mohmal implements email.Service cho mohmal.com
type Mohmal struct {
	client *http.Client
	email  string
}

// NewMohmal tạo Mohmal service với cookie jar
func NewMohmal(proxyStr string) *Mohmal {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	c.Jar = jar
	return &Mohmal{client: c}
}

// CreateEmail tạo email random qua GET mohmal.com/en/create/random
// Mapping từ WeBM MohmalcomAPI.CreateRandomEmail(user="", domain="RANDOM")
func (m *Mohmal) CreateEmail(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mohmalBaseURL+"/en/create/random", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 1<<20)
	bodyStr := string(body)

	re := regexp.MustCompile(`data-email="(.*?)"`)
	match := re.FindStringSubmatch(bodyStr)
	if len(match) >= 2 && match[1] != "" {
		m.email = strings.TrimSpace(match[1])
		return m.email, nil
	}
	return "", fmt.Errorf("mohmal: không tạo được email")
}

// GetEmail trả về địa chỉ email đã tạo trên mohmal.com (trống nếu chưa gọi CreateEmail).
func (m *Mohmal) GetEmail() string { return m.email }

// Close giải phóng tài nguyên — Mohmal dùng cookie jar tự quản lý, không cần đóng thủ công.
func (m *Mohmal) Close() {}

// WaitForCode poll OTP từ mohmal inbox
// Mapping từ WeBM MohmalcomAPI.GetMailboxes() + GetMailContent()
func (m *Mohmal) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
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

		ids, _ := m.getMailboxIDs(ctx)
		for _, id := range ids {
			html, _ := m.getMailContent(ctx, id)
			if html == "" {
				continue
			}
			if code := ExtractCode(html); code != "" {
				return code, nil
			}
		}

		if attempt < maxRetry-1 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			}
		}
	}
	return "", fmt.Errorf("mohmal: không nhận được OTP sau %d lần thử", maxRetry)
}

// getMailboxIDs — GET mohmal.com/en/refresh, extract data-msg-id
// Mapping từ WeBM MohmalcomAPI.GetMailboxes()
func (m *Mohmal) getMailboxIDs(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mohmalBaseURL+"/en/refresh", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 1<<20)
	bodyStr := string(body)

	matches := mohmalMsgIDRegex.FindAllStringSubmatch(bodyStr, -1)
	seen := make(map[string]bool)
	var ids []string
	for _, m := range matches {
		if len(m) >= 2 && !seen[m[1]] {
			seen[m[1]] = true
			ids = append(ids, m[1])
		}
	}
	return ids, nil
}

// getMailContent — GET mohmal.com/en/message/{id}
// Mapping từ WeBM MohmalcomAPI.GetMailContent()
func (m *Mohmal) getMailContent(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", mohmalBaseURL+"/en/message/"+id, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 1<<20)
	return string(body), nil
}
