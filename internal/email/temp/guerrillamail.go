// guerrillamail.go — GuerrillaMailWWW service (www.guerrillamail.com, set_email_user + alias inbox)
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

const guerrillaMailWWWBaseURL = "https://www.guerrillamail.com"

// GuerrillaMailWWW implements email.Service cho www.guerrillamail.com
// Dùng set_email_user để chọn username, sau đó poll inbox qua get_email_list + fetch_email.
type GuerrillaMailWWW struct {
	client   *http.Client
	email    string
	sidToken string
	alias    string
}

// NewGuerrillaMail tạo GuerrillaMailWWW service.
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewGuerrillaMail(proxyStr string) *GuerrillaMailWWW {
	jar, _ := cookiejar.New(nil)
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	client.Jar = jar
	return &GuerrillaMailWWW{client: client}
}

// CreateEmail tạo email qua POST /ajax.php?f=set_email_user.
func (g *GuerrillaMailWWW) CreateEmail(ctx context.Context) (string, error) {
	user := realisticLocalPart()

	form := url.Values{
		"f":          {"set_email_user"},
		"email_user": {user},
		"lang":       {"en"},
		"site":       {"guerrillamail.com"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		guerrillaMailWWWBaseURL+"/ajax.php",
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("guerrillamail create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	var result struct {
		EmailAddr string `json:"email_addr"`
		SidToken  string `json:"sid_token"`
		Alias     string `json:"alias"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.EmailAddr == "" {
		return "", fmt.Errorf("guerrillamail create: unexpected response: %s", string(body))
	}

	g.email = result.EmailAddr
	g.sidToken = result.SidToken
	g.alias = result.Alias
	if g.alias == "" {
		g.alias = user
	}
	return g.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (g *GuerrillaMailWWW) GetEmail() string { return g.email }

// Close cleanup resources.
func (g *GuerrillaMailWWW) Close() {}

// WaitForCode poll OTP từ guerrillamail inbox.
func (g *GuerrillaMailWWW) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if g.email == "" {
		return "", fmt.Errorf("guerrillamail: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := g.pollOnce(ctx)
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

// pollOnce lấy inbox và extract code.
func (g *GuerrillaMailWWW) pollOnce(ctx context.Context) (string, error) {
	inboxURL := fmt.Sprintf("%s/ajax.php?f=get_email_list&site=guerrillamail.com&in=%s&offset=0",
		guerrillaMailWWWBaseURL, url.QueryEscape(g.alias))

	req, err := http.NewRequestWithContext(ctx, "GET", inboxURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", "GuerrillaMailSession="+g.sidToken)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	resp, err := g.client.Do(req)
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
		return "", fmt.Errorf("guerrillamail inbox parse: %w", err)
	}

	for _, mail := range inbox.List {
		// Normalise mail_id: "247066949" → "247066949", 1 → "1"
		var mailID string
		if len(mail.MailIDRaw) > 0 {
			if mail.MailIDRaw[0] == '"' {
				_ = json.Unmarshal(mail.MailIDRaw, &mailID)
			} else {
				mailID = string(mail.MailIDRaw)
			}
		}
		if mailID == "" || mailID == "0" || mailID == "1" {
			continue // bỏ qua welcome mail (id=1) và empty
		}
		// Thử extract từ subject trước (nhanh, không cần fetch body)
		if subject := html.UnescapeString(mail.Subject); subject != "" {
			if code := ExtractCode(subject); code != "" {
				return code, nil
			}
		}
		// Fallback: fetch full body
		content, err := g.getMailContent(ctx, mailID)
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
func (g *GuerrillaMailWWW) getMailContent(ctx context.Context, mailID string) (string, error) {
	fetchURL := fmt.Sprintf("%s/ajax.php?f=fetch_email&site=guerrillamail.com&email_id=%s",
		guerrillaMailWWWBaseURL, url.QueryEscape(mailID))

	req, err := http.NewRequestWithContext(ctx, "GET", fetchURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Cookie", "GuerrillaMailSession="+g.sidToken)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	resp, err := g.client.Do(req)
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
