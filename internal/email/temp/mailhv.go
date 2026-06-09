// mailhv.go — MailHV service (dulich360.com REST API)
// Dùng api_token query param. Flow:
//   1. CreateEmail: dùng domain list đã cấu hình (hoặc fetch từ API nếu chưa set)
//      → ghép username@domain ngẫu nhiên
//   2. WaitForCode: GET /api/inbox/{email}?limit=50&offset=0 → list messages
//      → extract OTP TRỰC TIẾP từ subject (vd "57603 is your confirmation code")
//      → KHÔNG fetch body chi tiết → tiết kiệm 1 round-trip per message
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

const mailhvBase = "https://dulich360.com"

// MailHV implements email.Service cho dulich360.com.
type MailHV struct {
	client         *http.Client
	apiKey         string
	email          string
	configDomains  []string // domains do user cấu hình (ưu tiên); nếu rỗng → fetch từ API
	fetchedDomains []string // domains từ API (cache per instance, chỉ tên)
}

// NewMailHV tạo MailHV service.
// apiKey: bắt buộc (api_token).
// configDomains: danh sách domain user muốn dùng; rỗng = tự lấy từ API.
func NewMailHV(apiKey, proxyStr string, configDomains []string) *MailHV {
	return &MailHV{
		client:        proxy.CreateClient(proxyStr, 30*time.Second),
		apiKey:        apiKey,
		configDomains: configDomains,
	}
}

// GetEmail trả địa chỉ email đã tạo.
func (w *MailHV) GetEmail() string { return w.email }

// Close no-op.
func (w *MailHV) Close() {}

// CreateEmail chọn domain ngẫu nhiên và ghép username@domain.
// Dùng configDomains nếu có, ngược lại fetch từ API.
func (w *MailHV) CreateEmail(ctx context.Context) (string, error) {
	if w.apiKey == "" {
		return "", fmt.Errorf("mailhv: thiếu API key")
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
		return "", fmt.Errorf("mailhv: không có domain nào khả dụng")
	}
	domain := domains[rand.Intn(len(domains))]
	w.email = realisticEmail(domain)
	return w.email, nil
}

// fetchDomains gọi API và trả slice tên domain (chỉ verified).
func (w *MailHV) fetchDomains(ctx context.Context) ([]string, error) {
	_, items, err := fetchMailHVDomains(ctx, w.client, w.apiKey)
	if err != nil {
		return nil, err
	}
	return mailhvItemsToNames(items), nil
}

// mailhvItemsToNames lấy tên domain từ slice mailhvDomainItem (lọc bỏ domain chưa verified).
func mailhvItemsToNames(items []mailhvDomainItem) []string {
	out := make([]string, 0, len(items))
	for _, d := range items {
		name := d.domainName()
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

// MailHVDomainsResult là kết quả phân loại domain trả về cho frontend.
type MailHVDomainsResult struct {
	Plan string   `json:"plan"` // tên gói tài khoản, vd "business", "free"
	Free []string `json:"free"` // domain tier "free"
	Paid []string `json:"paid"` // domain tier khác free (premium, business, pro, …)
	All  []string `json:"all"`  // tất cả domain
}

// FetchMailHVDomains là hàm standalone — dùng từ app.go để hiển thị domain cho user.
// Trả MailHVDomainsResult phân loại free / paid (premium+business+pro gộp chung) + tên gói.
func FetchMailHVDomains(ctx context.Context, apiKey string) (*MailHVDomainsResult, error) {
	client := proxy.CreateClient("", 15*time.Second)
	plan, items, err := fetchMailHVDomains(ctx, client, apiKey)
	if err != nil {
		return nil, err
	}
	res := &MailHVDomainsResult{Plan: plan}
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

// mailhvDomainItem là 1 domain entry trong response của dulich360.com.
type mailhvDomainItem struct {
	Domain           string `json:"domain"`
	Name             string `json:"name"`   // fallback nếu API dùng "name"
	Tier             string `json:"tier"`   // "free" | "premium" | "business" | "pro"
	Source           string `json:"source"` // "system" | "custom"
	Verified         bool   `json:"verified"`
	SubdomainAllowed bool   `json:"subdomainAllowed"`
}

func (d mailhvDomainItem) domainName() string {
	if d.Domain != "" {
		return d.Domain
	}
	return d.Name
}

// fetchMailHVDomains gọi API và parse response dạng:
//
//	{"count":N,"plan":"business","domains":[{"domain":"...","tier":"free"|"business",...}]}
//
// Trả (planName, []mailhvDomainItem, error).
func fetchMailHVDomains(ctx context.Context, client *http.Client, apiKey string) (string, []mailhvDomainItem, error) {
	apiURL := mailhvBase + "/api/account/domains?api_token=" + url.QueryEscape(apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("mailhv domains: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	// Format chính: {"count":N,"plan":"...","domains":[{...}]}
	var wrapped struct {
		Plan    string             `json:"plan"`
		Count   int                `json:"count"`
		Domains []mailhvDomainItem `json:"domains"`
		Data    []mailhvDomainItem `json:"data"`
		Results []mailhvDomainItem `json:"results"`
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
	var objList []mailhvDomainItem
	if json.Unmarshal(body, &objList) == nil && len(objList) > 0 {
		return "", objList, nil
	}
	return "", nil, fmt.Errorf("mailhv: không đọc được domain — body: %.200s", body)
}

// WaitForCode poll inbox để phát hiện mail mới, đọc nội dung rồi extract OTP.
func (w *MailHV) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if w.email == "" {
		return "", fmt.Errorf("mailhv: email chưa được tạo")
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
	return "", fmt.Errorf("mailhv: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce gọi GET /api/inbox/{email}?limit=50&offset=0 — endpoint trả full message list.
// Strategy: CHỈ extract OTP từ subject, CHỈ duyệt mail từ facebookmail.com.
//
// Dulich360 luôn nhét OTP vào subject (vd "40428 là mã xác nhận của bạn") nên
// không cần fetch body. Filter from=facebookmail.com để bỏ qua mail rác như
// noreply@business.facebook.com (thông báo hạn chế, không chứa code) — tránh
// ExtractCode bắt nhầm số trong subject của các mail không phải OTP.
func (w *MailHV) pollOnce(ctx context.Context) (string, error) {
	inboxURL := mailhvBase + "/api/inbox/" + url.PathEscape(w.email) +
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
		if !isFacebookOTPSender(m.From) {
			continue
		}
		if code := ExtractCode(m.Subject); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// isFacebookOTPSender xác định mail có phải từ Facebook OTP system không.
// OTP đến từ *@facebookmail.com (registration@, security@, ...);
// noreply@business.facebook.com là mail thông báo, không có code.
func isFacebookOTPSender(from string) bool {
	return strings.Contains(strings.ToLower(from), "facebookmail.com") || strings.Contains(strings.ToLower(from), "instagram")
}

// ParseMailHVDomains tách chuỗi domain cách nhau bằng dấu phẩy/newline.
func ParseMailHVDomains(raw string) []string {
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
