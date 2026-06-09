// tempmailplus.go — TempMail.plus service
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const tempMailPlusBaseURL = "https://tempmail.plus"

var tempMailPlusDefaultDomains = []string{
	"mailto.plus", "fexpost.com", "fexbox.org", "mailbox.in.ua",
	"rover.info", "chitthi.in", "fextemp.com", "any.pink", "merepost.com",
}

// TempMailPlus implements email.Service cho tempmail.plus
type TempMailPlus struct {
	client  *http.Client
	domains []string
	user    string
	domain  string
	email   string
}

// NewTempMailPlus tạo TempMailPlus service.
// domainList: danh sách domain cách nhau bởi newline (mỗi dòng 1 domain).
// Để trống → dùng toàn bộ 9 domain mặc định của tempmail.plus.
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewTempMailPlus(domainList, proxyStr string) *TempMailPlus {
	// Split theo cả newline VÀ dấu phẩy — user có thể nhập "a.com, b.com" hoặc mỗi dòng 1 domain.
	var domains []string
	parts := strings.FieldsFunc(domainList, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ','
	})
	for _, p := range parts {
		d := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(p), "@"))
		if d != "" {
			domains = append(domains, d)
		}
	}
	if len(domains) == 0 {
		domains = tempMailPlusDefaultDomains
	}
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &TempMailPlus{client: client, domains: domains}
}

// CreateEmail tạo địa chỉ email client-side, random pick domain từ danh sách.
func (t *TempMailPlus) CreateEmail(ctx context.Context) (string, error) {
	t.domain = t.domains[rand.Intn(len(t.domains))]
	user := randomString(8) + fmt.Sprintf("%04d", rand.Intn(10000))
	t.user = user
	t.email = user + "@" + t.domain
	return t.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (t *TempMailPlus) GetEmail() string { return t.email }

// Close cleanup resources.
func (t *TempMailPlus) Close() {}

// WaitForCode poll OTP từ tempmail.plus inbox.
func (t *TempMailPlus) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmail.plus: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := t.pollOnce(ctx)
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

	return "", fmt.Errorf("tempmail.plus: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy danh sách mail và extract code.
func (t *TempMailPlus) pollOnce(ctx context.Context) (string, error) {
	encodedEmail := url.QueryEscape(t.user + "@" + t.domain)
	inboxURL := fmt.Sprintf("%s/api/mails?email=%s&limit=20&epin=", tempMailPlusBaseURL, encodedEmail)

	req, err := http.NewRequestWithContext(ctx, "GET", inboxURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	var inbox struct {
		MailList []struct {
			MailID  int64  `json:"mail_id"`
			Subject string `json:"subject"`
		} `json:"mail_list"`
	}
	if err := json.Unmarshal(body, &inbox); err != nil {
		return "", nil // non-JSON hoặc lỗi parse → coi như inbox rỗng
	}

	for _, mail := range inbox.MailList {
		// Thử extract từ subject trước (nhanh hơn, không cần gọi thêm API)
		if code := ExtractCode(mail.Subject); code != "" {
			return code, nil
		}
		// Fallback: lấy nội dung email đầy đủ
		content, err := t.getContent(ctx, fmt.Sprintf("%d", mail.MailID))
		if err != nil {
			continue
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// getContent lấy nội dung email theo mail_id.
func (t *TempMailPlus) getContent(ctx context.Context, mailID string) (string, error) {
	encodedEmail := url.QueryEscape(t.user + "@" + t.domain)
	msgURL := fmt.Sprintf("%s/api/mails/%s?email=%s&epin=", tempMailPlusBaseURL, mailID, encodedEmail)

	req, err := http.NewRequestWithContext(ctx, "GET", msgURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	var msg struct {
		HTML string `json:"html"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", err
	}

	if msg.HTML != "" {
		return msg.HTML, nil
	}
	return msg.Text, nil
}
