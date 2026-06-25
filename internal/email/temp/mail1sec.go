// mail1sec.go — Mail1secMail service (@i2b.vn)
// Mapping từ WeBM AppCore/Email/Mail1secMail.cs
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const mail1secBaseURL = "https://mail1sec.com"

// codeRegex extract 5-8 digit code — từ WeBM Mail1secMail._codeRegex \b\d{5,8}\b
// Facebook gửi code 5-6 chữ số, đôi khi 8 chữ số (tuỳ account type)
var codeRegex = regexp.MustCompile(`\b(\d{5,8})\b`)

// Mail1sec implements email.Service cho mail1sec.com (@i2b.vn)
type Mail1sec struct {
	client         *http.Client
	user           string
	domain         string
	email          string
	onStatus       func(string) // progress callback → UI
	customUsername string       // FmUserTmpMail — dùng prefix từ login info
}

// SetCustomUsername set custom prefix cho email (port FmUserTmpMail).
func (m *Mail1sec) SetCustomUsername(username string) { m.customUsername = username }

// NewMail1sec tạo Mail1sec service với proxy.
// domain: một hoặc nhiều domain cách nhau bằng newline/dấu phẩy. Để trống = "i2b.vn".
func NewMail1sec(domain, proxyStr string) *Mail1sec {
	return &Mail1sec{
		client: proxy.CreateClient(proxyStr, 30*time.Second),
		domain: pickDomain(domain, []string{"i2b.vn"}),
	}
}

// SetOnStatus đăng ký callback nhận log progress khi poll OTP
func (m *Mail1sec) SetOnStatus(fn func(string)) { m.onStatus = fn }

func (m *Mail1sec) notify(msg string) {
	if m.onStatus != nil {
		m.onStatus(msg)
	}
}

// CreateEmail tạo email random@i2b.vn
// Mapping từ WeBM Mail1secMail.CreateEmail() + GetEmailRandom()
func (m *Mail1sec) CreateEmail(ctx context.Context) (string, error) {
	m.user = realisticLocalPart()
	// FmUserTmpMail override — dùng login-derived username nếu có, thêm suffix random
	// để tránh collision khi cùng login dùng nhiều lần.
	if m.customUsername != "" {
		m.user = m.customUsername + fmt.Sprintf("_%04d", rand.Intn(10000))
	}
	m.email = m.user + "@" + m.domain
	return m.email, nil
}

// GetEmail trả về địa chỉ email @i2b.vn đã tạo (trống nếu chưa gọi CreateEmail).
func (m *Mail1sec) GetEmail() string { return m.email }

// Close giải phóng tài nguyên — Mail1sec không giữ session nên là no-op.
func (m *Mail1sec) Close() {}

// WaitForCode poll OTP code từ mail1sec
// Mapping từ WeBM Mail1secMail.WaitForFacebookCodeAsync() — interval mặc định 5000ms giống WeBM
func (m *Mail1sec) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 5000 // WeBM dùng 5000ms — Facebook thường mất 30-50s để gửi mail
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		m.notify(fmt.Sprintf("[Mail1sec] Kiểm tra hộp thư %s lần %d/%d...", m.email, attempt+1, maxRetry))
		code, err := m.getCodeDirect(ctx)
		if err == nil && code != "" {
			return code, nil
		}
		if err != nil {
			m.notify(fmt.Sprintf("[Mail1sec] Lỗi kết nối: %v", err))
		}

		if attempt < maxRetry-1 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			}
		}
	}

	return "", fmt.Errorf("no OTP code received after %d attempts", maxRetry)
}

// getCodeDirect — mapping từ WeBM Mail1secMail.GetCodeDirectAsync()
// GET /check-mail/{user}%40{domain}?latest_id=0 → regex extract 5 digits
func (m *Mail1sec) getCodeDirect(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/check-mail/%s%%40%s?latest_id=0", mail1secBaseURL, m.user, m.domain)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json, text/html, */*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := httpx.ReadBody(resp.Body, 0)
	if err != nil {
		return "", err
	}

	if len(body) == 0 {
		return "", nil
	}

	// Parse JSON — extract code từ subject của từng email
	var result struct {
		Emails []struct {
			ID      json.RawMessage `json:"id"`
			From    string          `json:"from"`
			Subject string          `json:"subject"`
		} `json:"emails"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", nil // JSON lỗi → không có email
	}
	if len(result.Emails) == 0 {
		return "", nil
	}

	// Log tất cả email tìm được để debug
	for _, e := range result.Emails {
		m.notify(fmt.Sprintf("[Mail1sec] Inbox: from=%s subject=%s", e.From, e.Subject))
	}

	// Pass 1: ưu tiên email từ Facebook/Meta, tìm code trong subject
	for _, e := range result.Emails {
		from := strings.ToLower(e.From)
		if strings.Contains(from, "facebook") || strings.Contains(from, "meta") || strings.Contains(from, "facebookmail") || strings.Contains(from, "instagram") {
			if match := codeRegex.FindStringSubmatch(e.Subject); len(match) >= 2 {
				m.notify(fmt.Sprintf("[Mail1sec] OTP từ subject: %s", match[1]))
				return match[1], nil
			}
		}
	}

	// Pass 2: nếu không tìm được trong subject FB, tìm trong TẤT CẢ subject
	// (tránh false positive từ email ID bằng cách chỉ tìm trong subject, không tìm raw JSON)
	for _, e := range result.Emails {
		if match := codeRegex.FindStringSubmatch(e.Subject); len(match) >= 2 {
			m.notify(fmt.Sprintf("[Mail1sec] OTP từ subject (all): %s", match[1]))
			return match[1], nil
		}
	}

	return "", nil
}

// randomString tạo chuỗi random a-z
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
