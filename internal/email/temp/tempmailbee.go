// tempmailbee.go — tempmailbee.com service (API nội bộ free, anonymous JWT).
//
// tempmailbee KHÔNG có API first-party công khai; đây là API mà chính frontend nó dùng
// (phát hiện qua JS trang chủ). Free, không cần mua key — chỉ cần auth ẩn danh.
//
// Flow (create đã verify live 2026-06-14):
//
//	auth:    POST /api/auth/anonymous/  {}              → {access_token, refresh_token, token_type:"Bearer", expires_in:600}
//	create:  POST /api/mailbox/create/?free_domain=1    → {"success":true,"email_address":"x@sendhelp.mom","expires_at":...}
//	list:    GET  /api/mailbox/emails/?email_address=x  → danh sách mail (cần CÙNG session cookie + Bearer đã tạo mailbox)
//	refresh: POST /api/auth/refresh/  {"refresh_token"} → access_token mới (giữ nguyên anon_id → giữ ownership)
//
// LƯU Ý quan trọng:
//   - Quyền đọc inbox bind theo anon_id trong JWT + session cookie. Phải dùng CÙNG cookie jar
//     xuyên suốt auth→create→read, nếu không server trả 403 "Unauthorized access to email".
//   - access_token hết hạn sau 600s → refresh bằng refresh_token (KHÔNG auth lại, vì auth mới
//     sinh anon_id khác → mất ownership mailbox cũ).
//   - Response read inbox chưa soi được shape chính xác (bị 403 lúc probe) → parser đi đệ quy mọi
//     string trong JSON rồi ExtractCode → shape-agnostic, không phụ thuộc tên field.
package temp

import (
	"bytes"
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

const tempMailBeeBaseURL = "https://tempmailbee.com/api"
const tempMailBeeUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// TempMailBee implements email.Service cho tempmailbee.com.
type TempMailBee struct {
	client        *http.Client
	email         string
	accessToken   string
	refreshToken  string
	configDomains []string // tên domain user chọn để ghim; rỗng = server random
}

// NewTempMailBee tạo TempMailBee service. proxyStr để trống nếu không dùng proxy.
// configDomains: danh sách domain muốn ghim (vd "sendhelp.mom"); rỗng = để server random.
func NewTempMailBee(proxyStr string, configDomains []string) *TempMailBee {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	c.Jar = jar // BẮT BUỘC giữ session cookie xuyên suốt auth→create→read
	return &TempMailBee{client: c, configDomains: configDomains}
}

// CreateEmail: auth ẩn danh → tạo mailbox random trên free domain.
func (m *TempMailBee) CreateEmail(ctx context.Context) (string, error) {
	if err := m.auth(ctx); err != nil {
		return "", err
	}

	createURL := tempMailBeeBaseURL + "/mailbox/create/?free_domain=1"
	// Ghim domain nếu user chọn: tạo địa chỉ cụ thể qua ?email_address=<local>@<domain>.
	if len(m.configDomains) > 0 {
		domain := m.configDomains[rand.Intn(len(m.configDomains))]
		localPart := realisticLocalPart()
		createURL = tempMailBeeBaseURL + "/mailbox/create/?email_address=" + url.QueryEscape(localPart+"@"+domain)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", createURL, nil)
	if err != nil {
		return "", err
	}
	m.setAuthHeaders(req)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tempmailbee create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 16*1024)

	var result struct {
		Success bool   `json:"success"`
		Email   string `json:"email_address"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Email == "" {
		return "", fmt.Errorf("tempmailbee create: unexpected response: %s", string(body))
	}

	m.email = result.Email
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *TempMailBee) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *TempMailBee) Close() {}

// WaitForCode poll OTP từ inbox tempmailbee.
func (m *TempMailBee) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" {
		return "", fmt.Errorf("tempmailbee: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		if code, _ := m.pollOnce(ctx); code != "" {
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

	return "", fmt.Errorf("tempmailbee: không nhận được OTP sau %d lần thử", maxRetry)
}

// auth lấy access_token + refresh_token ẩn danh.
func (m *TempMailBee) auth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "POST", tempMailBeeBaseURL+"/auth/anonymous/", bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", tempMailBeeUA)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://tempmailbee.com/")

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("tempmailbee auth: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 16*1024)

	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &tok); err != nil || tok.AccessToken == "" {
		return fmt.Errorf("tempmailbee auth: unexpected response: %s", string(body))
	}
	m.accessToken = tok.AccessToken
	m.refreshToken = tok.RefreshToken
	return nil
}

// refresh đổi access_token mới từ refresh_token (giữ nguyên anon_id → giữ ownership).
func (m *TempMailBee) refresh(ctx context.Context) error {
	if m.refreshToken == "" {
		return fmt.Errorf("tempmailbee refresh: no refresh_token")
	}
	payload, _ := json.Marshal(map[string]string{"refresh_token": m.refreshToken})
	req, err := http.NewRequestWithContext(ctx, "POST", tempMailBeeBaseURL+"/auth/refresh/", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", tempMailBeeUA)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://tempmailbee.com/")

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 16*1024)

	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tok); err != nil || tok.AccessToken == "" {
		return fmt.Errorf("tempmailbee refresh: unexpected response: %s", string(body))
	}
	m.accessToken = tok.AccessToken
	return nil
}

// pollOnce lấy inbox 1 lần; tự refresh token nếu 401, rồi extract code.
func (m *TempMailBee) pollOnce(ctx context.Context) (string, error) {
	body, status, err := m.fetchInbox(ctx)
	if err != nil {
		return "", err
	}
	// 401 = token hết hạn → refresh (giữ ownership) rồi thử lại 1 lần.
	if status == http.StatusUnauthorized {
		if rerr := m.refresh(ctx); rerr != nil {
			return "", rerr
		}
		body, status, err = m.fetchInbox(ctx)
		if err != nil {
			return "", err
		}
	}
	if status != http.StatusOK {
		return "", nil
	}
	return extractCodeFromAnyJSON(body), nil
}

// fetchInbox gọi GET /mailbox/emails/?email_address=... trả body + status.
func (m *TempMailBee) fetchInbox(ctx context.Context) ([]byte, int, error) {
	inboxURL := tempMailBeeBaseURL + "/mailbox/emails/?email_address=" + url.QueryEscape(m.email)
	req, err := http.NewRequestWithContext(ctx, "GET", inboxURL, nil)
	if err != nil {
		return nil, 0, err
	}
	m.setAuthHeaders(req)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)
	return body, resp.StatusCode, nil
}

func (m *TempMailBee) setAuthHeaders(req *http.Request) {
	req.Header.Set("User-Agent", tempMailBeeUA)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://tempmailbee.com/")
	if m.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+m.accessToken)
	}
}

// TempMailBeeDomainsResult — danh sách tên domain trả cho frontend.
type TempMailBeeDomainsResult struct {
	Domains []string `json:"domains"`
}

// FetchTempMailBeeDomains auth ẩn danh rồi GET /domains/ → available_domains. KHÔNG cần key.
func FetchTempMailBeeDomains(ctx context.Context) (*TempMailBeeDomainsResult, error) {
	m := NewTempMailBee("", nil)
	if err := m.auth(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", tempMailBeeBaseURL+"/domains/", nil)
	if err != nil {
		return nil, err
	}
	m.setAuthHeaders(req)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 32*1024)

	var d struct {
		AvailableDomains []string `json:"available_domains"`
	}
	if err := json.Unmarshal(body, &d); err != nil || len(d.AvailableDomains) == 0 {
		return nil, fmt.Errorf("tempmailbee: không đọc được domain — %.150s", body)
	}
	return &TempMailBeeDomainsResult{Domains: d.AvailableDomains}, nil
}

// ParseTempMailBeeDomains tách chuỗi domain cách nhau bằng dấu phẩy/newline.
func ParseTempMailBeeDomains(raw string) []string {
	var out []string
	for _, s := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	}) {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// extractCodeFromAnyJSON đệ quy mọi string value trong JSON rồi ExtractCode.
// Shape-agnostic: không phụ thuộc tên field (subject/body/html/text/...).
// An toàn false-positive vì ExtractCode chủ yếu match cụm FB-specific, không match id/timestamp.
func extractCodeFromAnyJSON(body []byte) string {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return ""
	}
	return walkJSONForCode(v)
}

func walkJSONForCode(v any) string {
	switch t := v.(type) {
	case string:
		return ExtractCode(t)
	case []any:
		for _, e := range t {
			if c := walkJSONForCode(e); c != "" {
				return c
			}
		}
	case map[string]any:
		for _, e := range t {
			if c := walkJSONForCode(e); c != "" {
				return c
			}
		}
	}
	return ""
}
