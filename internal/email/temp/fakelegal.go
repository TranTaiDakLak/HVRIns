// fakelegal.go — fake.legal service (free JSON API, không cần key).
//
// Flow (đã verify live 2026-06-14):
//
//	CreateEmail: GET /api/inbox/new           → {"success":true,"address":"x@trashmails.lol","expiresIn":"3 minutes"}
//	Poll list:   GET /api/inbox/{address}      → {"success":true,"emails":[{id,from,subject,date}],"exists":true}
//	Get body:    GET /api/email/{id}           → {"success":true,"email":{id,from,subject,text,html}}
//
// LƯU Ý: server CHẶN User-Agent rỗng/curl mặc định (trả connection-reset). Bắt buộc set UA
// trình duyệt. TTL mailbox chỉ ~3 phút nên reuse register→verify gần như không khả thi.
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const fakeLegalBaseURL = "https://fake.legal"

// UA trình duyệt — BẮT BUỘC, UA mặc định bị fake.legal chặn.
const fakeLegalUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// FakeLegal implements email.Service cho fake.legal.
type FakeLegal struct {
	client        *http.Client
	email         string
	configDomains []string // tên domain user chọn để ghim; rỗng = server random
}

// NewFakeLegal tạo FakeLegal service. proxyStr để trống nếu không dùng proxy.
// configDomains: danh sách domain muốn ghim (vd "imgui.de"); rỗng = để server random.
func NewFakeLegal(proxyStr string, configDomains []string) *FakeLegal {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	c.Jar = jar
	return &FakeLegal{client: c, configDomains: configDomains}
}

// CreateEmail gọi GET /api/inbox/new lấy địa chỉ server cấp.
func (m *FakeLegal) CreateEmail(ctx context.Context) (string, error) {
	endpoint := fakeLegalBaseURL + "/api/inbox/new"
	// Ghim domain nếu user chọn (fake.legal nhận ?domain=<tên>); rỗng = server random.
	if len(m.configDomains) > 0 {
		endpoint += "?domain=" + m.configDomains[rand.Intn(len(m.configDomains))]
	}
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	m.setHeaders(req)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fakelegal create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 16*1024)

	var result struct {
		Success bool   `json:"success"`
		Address string `json:"address"`
	}
	if err := json.Unmarshal(body, &result); err != nil || !result.Success || result.Address == "" {
		return "", fmt.Errorf("fakelegal create: unexpected response: %s", string(body))
	}

	m.email = result.Address
	return m.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *FakeLegal) GetEmail() string { return m.email }

// Close cleanup resources.
func (m *FakeLegal) Close() {}

// WaitForCode poll OTP từ inbox fake.legal.
func (m *FakeLegal) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" {
		return "", fmt.Errorf("fakelegal: chưa tạo email")
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

	return "", fmt.Errorf("fakelegal: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy danh sách mail rồi extract code (subject trước, body sau).
func (m *FakeLegal) pollOnce(ctx context.Context) (string, error) {
	listURL := fakeLegalBaseURL + "/api/inbox/" + m.email
	req, err := http.NewRequestWithContext(ctx, "GET", listURL, nil)
	if err != nil {
		return "", err
	}
	m.setHeaders(req)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var list struct {
		Success bool `json:"success"`
		Emails  []struct {
			ID      string `json:"id"`
			Subject string `json:"subject"`
		} `json:"emails"`
	}
	if err := json.Unmarshal(body, &list); err != nil || !list.Success {
		return "", nil // non-JSON / inbox chưa sẵn sàng → coi như rỗng
	}

	for _, e := range list.Emails {
		// Subject trước — OTP FB thường nằm ngay subject, đỡ 1 request.
		if code := ExtractCode(e.Subject); code != "" {
			return code, nil
		}
		if e.ID == "" {
			continue
		}
		content, err := m.getMessage(ctx, e.ID)
		if err != nil {
			continue
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// getMessage lấy nội dung email theo id (ưu tiên html, fallback text).
func (m *FakeLegal) getMessage(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fakeLegalBaseURL+"/api/email/"+id, nil)
	if err != nil {
		return "", err
	}
	m.setHeaders(req)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	var msg struct {
		Success bool `json:"success"`
		Email   struct {
			Subject string `json:"subject"`
			Text    string `json:"text"`
			HTML    string `json:"html"`
		} `json:"email"`
	}
	if err := json.Unmarshal(body, &msg); err != nil || !msg.Success {
		return "", fmt.Errorf("fakelegal getMessage: unexpected response")
	}

	if msg.Email.HTML != "" {
		return msg.Email.HTML, nil
	}
	if msg.Email.Text != "" {
		return msg.Email.Text, nil
	}
	return msg.Email.Subject, nil
}

func (m *FakeLegal) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", fakeLegalUA)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", fakeLegalBaseURL+"/")
}

// FakeLegalDomainsResult — danh sách tên domain trả cho frontend.
type FakeLegalDomainsResult struct {
	Domains []string `json:"domains"`
}

// FetchFakeLegalDomains lấy list domain từ GET /api/stats (KHÔNG cần key).
func FetchFakeLegalDomains(ctx context.Context) (*FakeLegalDomainsResult, error) {
	client := proxy.CreateClient("", 15*time.Second)
	req, err := http.NewRequestWithContext(ctx, "GET", fakeLegalBaseURL+"/api/stats", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fakeLegalUA)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 32*1024)

	var stats struct {
		Domains []string `json:"domains"`
	}
	if err := json.Unmarshal(body, &stats); err != nil || len(stats.Domains) == 0 {
		return nil, fmt.Errorf("fakelegal: không đọc được domain — %.150s", body)
	}
	return &FakeLegalDomainsResult{Domains: stats.Domains}, nil
}

// ParseFakeLegalDomains tách chuỗi domain cách nhau bằng dấu phẩy/newline.
func ParseFakeLegalDomains(raw string) []string {
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
