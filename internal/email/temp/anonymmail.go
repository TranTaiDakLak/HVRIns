// anonymmail.go — anonymmail.net service (own JSON API, form-encoded).
//
// Flow (reverse-engineer từ bundle, xác nhận 2026-06-18):
//   1. POST /api/getDomains            → [{"domain":"...","expires":...}]
//   2. POST /api/create  email=<local>@<domain>  (form) → {"success":true,"email":"..."}
//   3. POST /api/get     email=<addr>            (form) → {"<addr>":{"emails":[{token,subject,summary,body}]}}
//
// KHÔNG cần key, KHÔNG Cloudflare — http.Client thường qua proxy pool.
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const anonymMailBaseURL = "https://anonymmail.net"

// anonymMailKnownDomains — fallback nếu /api/getDomains fail (cập nhật 2026-06-18).
var anonymMailKnownDomains = []string{
	"california.edu.pl", "kayilo.com", "mailshun.com", "mailbali.com",
}

// AnonymMail implements email.Service cho anonymmail.net.
type AnonymMail struct {
	client        *http.Client
	email         string
	pinnedDomains []string // domain user chọn; rỗng = random toàn pool
}

// ParseAnonymMailDomains tách chuỗi "a.com, b.com" → []string.
func ParseAnonymMailDomains(raw string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' }) {
		if d := strings.TrimPrefix(strings.TrimSpace(p), "@"); d != "" {
			out = append(out, d)
		}
	}
	return out
}

// NewAnonymMail tạo AnonymMail service. pinnedDomains = domain user chọn; rỗng = random pool.
func NewAnonymMail(proxyStr string, pinnedDomains []string) *AnonymMail {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	c.Jar = jar
	return &AnonymMail{client: c, pinnedDomains: pinnedDomains}
}

func (a *AnonymMail) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", anonymMailBaseURL)
	req.Header.Set("Referer", anonymMailBaseURL+"/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
}

func (a *AnonymMail) postForm(ctx context.Context, path, body string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", anonymMailBaseURL+path, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	a.setHeaders(req)
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := httpx.ReadBody(resp.Body, 0)
	return b, nil
}

// CreateEmail tạo email: chọn domain (pinned hoặc fetch random) → POST /api/create.
func (a *AnonymMail) CreateEmail(ctx context.Context) (string, error) {
	domains := a.pinnedDomains
	if len(domains) == 0 {
		if dr, err := FetchAnonymMailDomains(ctx); err == nil && len(dr.Domains) > 0 {
			domains = dr.Domains
		} else {
			domains = anonymMailKnownDomains
		}
	}
	domain := domains[rand.Intn(len(domains))]
	local := realisticLocalPart()
	email := local + "@" + domain

	body, err := a.postForm(ctx, "/api/create", "email="+url.QueryEscape(email))
	if err != nil {
		return "", fmt.Errorf("anonymmail create: %w", err)
	}
	var result struct {
		Success bool   `json:"success"`
		Email   string `json:"email"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Success {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return "", fmt.Errorf("anonymmail create: thất bại snippet=%q", snippet)
	}
	a.email = email
	if result.Email != "" {
		a.email = result.Email
	}
	return a.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (a *AnonymMail) GetEmail() string { return a.email }

// Close no-op.
func (a *AnonymMail) Close() {}

// WaitForCode poll OTP từ inbox.
func (a *AnonymMail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if a.email == "" {
		return "", fmt.Errorf("anonymmail: chưa tạo email")
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if code, _ := a.pollOnce(ctx); code != "" {
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
	return "", fmt.Errorf("anonymmail: không nhận được OTP sau %d lần thử", maxRetry)
}

func (a *AnonymMail) pollOnce(ctx context.Context) (string, error) {
	body, err := a.postForm(ctx, "/api/get", "email="+url.QueryEscape(a.email))
	if err != nil {
		return "", err
	}
	// Response: {"<email>":{"created_at":"...","emails":[{token,subject,summary,body}]}}
	var outer map[string]struct {
		Emails []struct {
			Subject string `json:"subject"`
			Summary string `json:"summary"`
			Body    string `json:"body"`
		} `json:"emails"`
	}
	if err := json.Unmarshal(body, &outer); err != nil {
		return "", nil
	}
	for _, box := range outer {
		for _, m := range box.Emails {
			if code := ExtractCode(m.Subject); code != "" {
				return code, nil
			}
			if code := ExtractCode(m.Summary); code != "" {
				return code, nil
			}
			if code := ExtractCode(m.Body); code != "" {
				return code, nil
			}
		}
	}
	return "", nil
}

// AnonymMailDomainsResult là kết quả trả về cho FetchAnonymMailDomains.
type AnonymMailDomainsResult struct {
	Domains []string `json:"domains"`
}

// FetchAnonymMailDomains gọi POST /api/getDomains. KHÔNG cần key. Fallback hardcoded nếu fail.
func FetchAnonymMailDomains(ctx context.Context) (*AnonymMailDomainsResult, error) {
	jar, _ := cookiejar.New(nil)
	client := proxy.CreateClient("", 15*time.Second)
	client.Jar = jar
	req, err := http.NewRequestWithContext(ctx, "POST", anonymMailBaseURL+"/api/getDomains", nil)
	if err != nil {
		return &AnonymMailDomainsResult{Domains: anonymMailKnownDomains}, nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", anonymMailBaseURL+"/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	resp, err := client.Do(req)
	if err != nil {
		return &AnonymMailDomainsResult{Domains: anonymMailKnownDomains}, nil
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var list []struct {
		Domain string `json:"domain"`
	}
	if err := json.Unmarshal(body, &list); err != nil || len(list) == 0 {
		return &AnonymMailDomainsResult{Domains: anonymMailKnownDomains}, nil
	}
	var out []string
	for _, d := range list {
		if d.Domain != "" {
			out = append(out, d.Domain)
		}
	}
	if len(out) == 0 {
		return &AnonymMailDomainsResult{Domains: anonymMailKnownDomains}, nil
	}
	return &AnonymMailDomainsResult{Domains: out}, nil
}
