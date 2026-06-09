// spam4me.go — Spam4.me service (GuerrillaMail platform variant, site=spam4.me)
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const spam4MeBaseURL = "https://www.guerrillamail.com"
const spam4MeSite = "spam4.me"

// Spam4Me implements email.Service cho spam4.me (dùng GuerrillaMail platform)
type Spam4Me struct {
	client   *http.Client
	email    string
	sidToken string
	alias    string
}

// NewSpam4Me tạo Spam4Me service.
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewSpam4Me(proxyStr string) *Spam4Me {
	jar, _ := cookiejar.New(nil)
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	client.Jar = jar
	return &Spam4Me{client: client}
}

// CreateEmail tạo email qua POST /ajax.php?f=set_email_user&site=spam4.me.
func (s *Spam4Me) CreateEmail(ctx context.Context) (string, error) {
	user := realisticLocalPart()

	form := url.Values{
		"f":          {"set_email_user"},
		"email_user": {user},
		"lang":       {"en"},
		"site":       {spam4MeSite},
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		spam4MeBaseURL+"/ajax.php",
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("spam4me create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	var result struct {
		EmailAddr string `json:"email_addr"`
		SidToken  string `json:"sid_token"`
		Alias     string `json:"alias"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.EmailAddr == "" {
		return "", fmt.Errorf("spam4me create: unexpected response: %s", string(body))
	}

	s.email = result.EmailAddr
	s.sidToken = result.SidToken
	s.alias = result.Alias
	if s.alias == "" {
		s.alias = user
	}
	return s.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (s *Spam4Me) GetEmail() string { return s.email }

// Close cleanup resources.
func (s *Spam4Me) Close() {}

// WaitForCode poll OTP từ spam4.me inbox.
func (s *Spam4Me) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if s.email == "" {
		return "", fmt.Errorf("spam4me: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := s.pollOnce(ctx)
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

	return "", fmt.Errorf("spam4me: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy inbox và extract code.
func (s *Spam4Me) pollOnce(ctx context.Context) (string, error) {
	inboxURL := fmt.Sprintf("%s/ajax.php?f=get_email_list&site=%s&in=%s&offset=0",
		spam4MeBaseURL, spam4MeSite, url.QueryEscape(s.alias))

	req, err := http.NewRequestWithContext(ctx, "GET", inboxURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", "GuerrillaMailSession="+s.sidToken)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	// mail_id có thể là string "247066949" hoặc int 1 (welcome mail) — dùng RawMessage
	var inbox struct {
		List []struct {
			MailIDRaw json.RawMessage `json:"mail_id"`
			Subject   string          `json:"mail_subject"`
		} `json:"list"`
	}
	if err := json.Unmarshal(body, &inbox); err != nil {
		return "", fmt.Errorf("spam4me inbox parse: %w", err)
	}

	for _, mail := range inbox.List {
		var mailID string
		if len(mail.MailIDRaw) > 0 {
			if mail.MailIDRaw[0] == '"' {
				_ = json.Unmarshal(mail.MailIDRaw, &mailID)
			} else {
				mailID = string(mail.MailIDRaw)
			}
		}
		if mailID == "" || mailID == "0" || mailID == "1" {
			continue
		}
		// Thử extract từ subject trước
		if subject := html.UnescapeString(mail.Subject); subject != "" {
			if code := ExtractCode(subject); code != "" {
				return code, nil
			}
		}
		content, err := s.getMailContent(ctx, mailID)
		if err != nil {
			continue
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// getMailContent lấy nội dung email theo mail_id.
func (s *Spam4Me) getMailContent(ctx context.Context, mailID string) (string, error) {
	fetchURL := fmt.Sprintf("%s/ajax.php?f=fetch_email&site=%s&email_id=%s",
		spam4MeBaseURL, spam4MeSite, url.QueryEscape(mailID))

	req, err := http.NewRequestWithContext(ctx, "GET", fetchURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", "GuerrillaMailSession="+s.sidToken)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	var msg struct {
		MailBody string `json:"mail_body"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", err
	}
	return msg.MailBody, nil
}
