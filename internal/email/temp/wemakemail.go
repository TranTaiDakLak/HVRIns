// wemakemail.go — WeMakeMail service (wemakemail.com REST API)
// Dùng api_token query param. Flow:
//   1. CreateEmail: dùng domain list đã cấu hình (hoặc fetch từ API nếu chưa set)
//      → ghép username@domain ngẫu nhiên
//   2. WaitForCode: GET /api/inbox/{email}?limit=50&offset=0 → list messages
//      → subject thường đã chứa OTP (vd "57603 is your confirmation code")
//      → fallback: GET /api/message/{id} đọc textBody/htmlBody nếu subject không khớp
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

const wemakemailBase = "https://wemakemail.com"

// WeMakeMail implements email.Service cho wemakemail.com.
type WeMakeMail struct {
	client         *http.Client
	apiKey         string
	email          string
	configDomains  []string // domains do user cấu hình (ưu tiên); nếu rỗng → fetch từ API
	fetchedDomains []string // domains từ API (cache per instance, chỉ tên)
}

// NewWeMakeMail tạo WeMakeMail service.
// apiKey: bắt buộc (wm_live_... token).
// configDomains: danh sách domain user muốn dùng; rỗng = tự lấy từ API.
func NewWeMakeMail(apiKey, proxyStr string, configDomains []string) *WeMakeMail {
	return &WeMakeMail{
		client:        proxy.CreateClient(proxyStr, 30*time.Second),
		apiKey:        apiKey,
		configDomains: configDomains,
	}
}

// GetEmail trả địa chỉ email đã tạo.
func (w *WeMakeMail) GetEmail() string { return w.email }

// Close no-op.
func (w *WeMakeMail) Close() {}

// CreateEmail chọn domain ngẫu nhiên và ghép username@domain.
// Dùng configDomains nếu có, ngược lại fetch từ API.
func (w *WeMakeMail) CreateEmail(ctx context.Context) (string, error) {
	if w.apiKey == "" {
		return "", fmt.Errorf("wemakemail: thiếu API key")
	}
	domains := w.configDomains
	if len(domains) == 0 {
		if len(w.fetchedDomains) == 0 {
			fetched, err := w.fetchDomains(ctx)
			if err != nil {
				return "", err
			}
			w.fetchedDomains = fetched
		}
		domains = w.fetchedDomains
	}
	if len(domains) == 0 {
		return "", fmt.Errorf("wemakemail: không có domain nào khả dụng")
	}
	domain := domains[rand.Intn(len(domains))]
	w.email = realisticEmail(domain)
	return w.email, nil
}

// fetchDomains gọi API và trả slice tên domain (chỉ verified).
func (w *WeMakeMail) fetchDomains(ctx context.Context) ([]string, error) {
	_, items, err := fetchWeMakeMailDomains(ctx, w.client, w.apiKey)
	if err != nil {
		return nil, err
	}
	return itemsToNames(items), nil
}

// itemsToNames lấy tên domain từ slice wmDomainItem (lọc bỏ domain chưa verified).
func itemsToNames(items []wmDomainItem) []string {
	out := make([]string, 0, len(items))
	for _, d := range items {
		name := d.domainName()
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

// WmDomainsResult là kết quả phân loại domain trả về cho frontend.
type WmDomainsResult struct {
	Plan string   `json:"plan"` // tên gói tài khoản, vd "business", "free"
	Free []string `json:"free"` // domain tier "free"
	Paid []string `json:"paid"` // domain tier khác free (premium, business, pro, …)
	All  []string `json:"all"`  // tất cả domain
}

// FetchWeMakeMailDomains là hàm standalone — dùng từ app.go để hiển thị domain cho user.
// Trả WmDomainsResult phân loại free / paid (premium+business+pro gộp chung) + tên gói.
func FetchWeMakeMailDomains(ctx context.Context, apiKey string) (*WmDomainsResult, error) {
	client := proxy.CreateClient("", 15*time.Second)
	plan, items, err := fetchWeMakeMailDomains(ctx, client, apiKey)
	if err != nil {
		return nil, err
	}
	res := &WmDomainsResult{Plan: plan}
	for _, d := range items {
		name := d.domainName()
		if name == "" {
			continue
		}
		res.All = append(res.All, name)
		if d.Tier == "free" || d.Tier == "" {
			res.Free = append(res.Free, name)
		} else {
			res.Paid = append(res.Paid, name)
		}
	}
	if res.Free == nil {
		res.Free = []string{}
	}
	if res.Paid == nil {
		res.Paid = []string{}
	}
	if res.All == nil {
		res.All = []string{}
	}
	return res, nil
}

// wmDomainItem là 1 domain entry trong response của wemakemail.com.
type wmDomainItem struct {
	Domain           string `json:"domain"`
	Name             string `json:"name"`   // fallback nếu API dùng "name"
	Tier             string `json:"tier"`   // "free" | "premium" | "business" | "pro"
	Source           string `json:"source"` // "system" | "custom"
	Verified         bool   `json:"verified"`
	SubdomainAllowed bool   `json:"subdomainAllowed"`
}

func (d wmDomainItem) domainName() string {
	if d.Domain != "" {
		return d.Domain
	}
	return d.Name
}

// fetchWeMakeMailDomains gọi API và parse response dạng:
//
//	{"count":N,"plan":"business","domains":[{"domain":"...","tier":"free"|"business",...}]}
//
// Trả (planName, []wmDomainItem, error).
func fetchWeMakeMailDomains(ctx context.Context, client *http.Client, apiKey string) (string, []wmDomainItem, error) {
	apiURL := wemakemailBase + "/api/account/domains?api_token=" + url.QueryEscape(apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("wemakemail domains: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	// Format chính: {"count":N,"plan":"...","domains":[{...}]}
	var wrapped struct {
		Plan    string         `json:"plan"`
		Count   int            `json:"count"`
		Domains []wmDomainItem `json:"domains"`
		Data    []wmDomainItem `json:"data"`
		Results []wmDomainItem `json:"results"`
	}
	if json.Unmarshal(body, &wrapped) == nil {
		items := wrapped.Domains
		if len(items) == 0 {
			items = wrapped.Data
		}
		if len(items) == 0 {
			items = wrapped.Results
		}
		if len(items) > 0 {
			return wrapped.Plan, items, nil
		}
	}
	// Fallback: array trực tiếp [{"domain":"..."}]
	var objList []wmDomainItem
	if json.Unmarshal(body, &objList) == nil && len(objList) > 0 {
		return "", objList, nil
	}
	return "", nil, fmt.Errorf("wemakemail: không đọc được domain — body: %.200s", body)
}

// WaitForCode poll /head để phát hiện mail mới, đọc nội dung rồi extract OTP.
func (w *WeMakeMail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if w.email == "" {
		return "", fmt.Errorf("wemakemail: email chưa được tạo")
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if code, _ := w.pollOnce(ctx); code != "" {
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
	return "", fmt.Errorf("wemakemail: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce gọi GET /api/inbox/{email}?limit=50&offset=0 — endpoint trả full message list.
// (KHÔNG dùng /head: endpoint đó chỉ trả {topId, countApprox, lastSeen} — không phải list mail.)
//
// Strategy: thử subject trước (nhanh, không tốn API call), nếu KHÔNG match thì LUÔN fetch
// full body — không bao giờ skip body. Vì:
//   - Subject có thể trống/không chứa code (Facebook đôi khi gửi subject chung chung)
//   - Subject có thể là ngôn ngữ pattern chưa cover → fallback body (HTML structural pattern
//     như letter-spacing:5px hoạt động bất kể ngôn ngữ)
//
// Inbox trả về newest-first → OTP mới nhất được check đầu tiên.
func (w *WeMakeMail) pollOnce(ctx context.Context) (string, error) {
	inboxURL := wemakemailBase + "/api/inbox/" + url.PathEscape(w.email) +
		"?limit=50&offset=0&api_token=" + url.QueryEscape(w.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", inboxURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)

	var inbox struct {
		Messages []struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
			From    string `json:"from"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &inbox); err != nil {
		return "", nil
	}
	for _, m := range inbox.Messages {
		// Try subject first (cheap — không cần API call thêm)
		if code := ExtractCode(m.Subject); code != "" {
			return code, nil
		}
		// LUÔN fallback fetch body — subject không có/không match cũng phải check body.
		if m.ID == "" {
			continue
		}
		if code, _ := w.getMessage(ctx, m.ID); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// getMessage gọi GET /api/message/{id} và extract OTP từ nội dung.
// API trả về fields camelCase: textBody, htmlBody (KHÔNG phải text/html_body).
func (w *WeMakeMail) getMessage(ctx context.Context, id string) (string, error) {
	msgURL := wemakemailBase + "/api/message/" + url.PathEscape(id) +
		"?api_token=" + url.QueryEscape(w.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", msgURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 512*1024)

	var msg struct {
		Subject  string `json:"subject"`
		TextBody string `json:"textBody"`
		HtmlBody string `json:"htmlBody"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", err
	}
	for _, s := range []string{msg.TextBody, msg.HtmlBody, msg.Subject} {
		if code := ExtractCode(s); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// ParseWeMakeMailDomains tách chuỗi domain cách nhau bằng dấu phẩy/newline.
func ParseWeMakeMailDomains(raw string) []string {
	var out []string
	for _, s := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	}) {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
