// vietxf.go — VietXF service (vietxf.com)
// Dùng query param key= (hoặc X-API-KEY header). Flow:
//  1. CreateEmail: dùng domain list đã cấu hình hoặc fetch từ API
//     → ghép username@domain ngẫu nhiên
//  2. WaitForCode: GET /getmail/{email}?key={api_key}[&latest_id={id}]
//     → extract OTP từ subject, chỉ xử lý mail từ facebookmail.com
//     → dùng latest_id để chỉ lấy mail mới hơn lần poll trước
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const vietxfBase = "https://vietxf.com"

// VietXF implements email.Service cho vietxf.com.
type VietXF struct {
	client        *http.Client
	apiKey        string
	email         string
	latestID      string   // ID cao nhất đã thấy (string theo API) — dùng cho &latest_id poll tiếp theo
	configDomains []string // domains do user cấu hình (ưu tiên); rỗng = fetch từ API
	cachedDomains []string // domains lấy từ API (cache per instance)
}

// NewVietXF tạo VietXF service.
func NewVietXF(apiKey, proxyStr string, configDomains []string) *VietXF {
	return &VietXF{
		client:        proxy.CreateClient(proxyStr, 30*time.Second),
		apiKey:        apiKey,
		configDomains: configDomains,
	}
}

func (w *VietXF) GetEmail() string { return w.email }
func (w *VietXF) Close()           {}

// CreateEmail chọn domain ngẫu nhiên và ghép username@domain.
func (w *VietXF) CreateEmail(ctx context.Context) (string, error) {
	if w.apiKey == "" {
		return "", fmt.Errorf("vietxf: thiếu API key")
	}
	domains := w.configDomains
	if len(domains) == 0 {
		if len(w.cachedDomains) == 0 {
			fetched, err := w.fetchDomains(ctx)
			if err != nil {
				return "", err
			}
			w.cachedDomains = fetched
		}
		domains = w.cachedDomains
	}
	if len(domains) == 0 {
		return "", fmt.Errorf("vietxf: không có domain nào khả dụng")
	}
	w.email = realisticEmail(domains[rand.Intn(len(domains))])
	w.latestID = ""
	return w.email, nil
}

// WaitForCode poll inbox cho đến khi tìm thấy OTP.
func (w *VietXF) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if w.email == "" {
		return "", fmt.Errorf("vietxf: email chưa được tạo")
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
	return "", fmt.Errorf("vietxf: không nhận được OTP sau %d lần thử", maxRetry)
}

type vietxfEmail struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Date    string `json:"date"`
}

// pollOnce gọi GET /getmail/{email}?key={api_key}[&latest_id={id}].
// Dùng latest_id để chỉ nhận mail mới hơn lần poll trước.
func (w *VietXF) pollOnce(ctx context.Context) (string, error) {
	apiURL := vietxfBase + "/getmail/" + url.PathEscape(w.email) +
		"?key=" + url.QueryEscape(w.apiKey)
	if w.latestID != "" {
		apiURL += "&latest_id=" + w.latestID
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
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
		Success bool          `json:"success"`
		Emails  []vietxfEmail `json:"emails"`
	}
	if err := json.Unmarshal(body, &inbox); err != nil || !inbox.Success {
		return "", nil
	}
	for _, m := range inbox.Emails {
		if m.ID > w.latestID {
			w.latestID = m.ID // string compare — đủ dùng vì ID là số nguyên tăng dần
		}
		if !isFacebookOTPSender(m.From) {
			continue
		}
		if code := ExtractCode(m.Subject); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// fetchDomains gọi GET /api/account/domains?key={api_key}.
func (w *VietXF) fetchDomains(ctx context.Context) ([]string, error) {
	_, domains, err := fetchVietXFDomains(ctx, w.client, w.apiKey)
	return domains, err
}

// VietXFDomainsResult là kết quả trả về cho frontend.
type VietXFDomainsResult struct {
	Domains []string `json:"domains"`
}

// FetchVietXFDomains là hàm standalone — dùng từ app.go để hiển thị domain cho user.
func FetchVietXFDomains(ctx context.Context, apiKey string) (*VietXFDomainsResult, error) {
	client := proxy.CreateClient("", 15*time.Second)
	_, domains, err := fetchVietXFDomains(ctx, client, apiKey)
	if err != nil {
		return nil, err
	}
	if domains == nil {
		domains = []string{}
	}
	return &VietXFDomainsResult{Domains: domains}, nil
}

func fetchVietXFDomains(ctx context.Context, client *http.Client, apiKey string) (string, []string, error) {
	apiURL := vietxfBase + "/api/account/domains?key=" + url.QueryEscape(apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("vietxf domains: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var result struct {
		Success bool     `json:"success"`
		Domains []string `json:"domains"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", nil, fmt.Errorf("vietxf: không đọc được domain — body: %.200s", body)
	}
	if !result.Success {
		return "", nil, fmt.Errorf("vietxf: API trả success=false — body: %.200s", body)
	}
	return "", result.Domains, nil
}

// ParseVietXFDomains tách chuỗi domain cách nhau bằng dấu phẩy/newline.
func ParseVietXFDomains(raw string) []string {
	return ParseMailHVDomains(raw) // cùng logic split
}
