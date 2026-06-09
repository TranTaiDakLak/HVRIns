// moakt.go — MoaktMail service (13 domains)
// Mapping từ WeBM AppCore/Email/MoaktMail.cs
package temp

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const moaktBaseURL = "https://www.moakt.com"

// MoaktDomains — chỉ dùng tmpbox.net
var MoaktDomains = []string{
	"tmpbox.net",
}

// Facebook + Instagram senders — từ WeBM MoaktMail._fbSenders
var fbSenders = []string{
	"security@facebookmail.com",
	"security@account.meta.com",
	"noreply@facebookmail.com",
	"mail.instagram.com",
	"security@mail.instagram.com",
	"no-reply@mail.instagram.com",
	"instagram.com",
}

// Code extraction regexes — từ WeBM MoaktMail.ExtractCode()
//
// Coverage:
//   - Pattern 1-2: English ("is your confirmation code") — Facebook EN
//   - Pattern 3: HTML styled (letter-spacing:5px) — Facebook bất kỳ ngôn ngữ nào (struct chung)
//   - Pattern 4: Vietnamese ("là mã xác nhận") — Facebook VN
//   - Pattern 5: code đứng đầu subject — universal, mọi ngôn ngữ
//   - Pattern 6-9: các ngôn ngữ phổ biến khác (es/pt/id/fil) — fallback nếu pattern 5 miss
//   - Pattern 10: Facebook universal — "Facebook" + N-digit code anywhere
var (
	// Pattern 1: <li class="title">...(\d{4,8}) is your confirmation code</li>
	codePattern1 = regexp.MustCompile(`(?i)<li[^>]*class\s*=\s*["']title["'][^>]*>\s*(\d{4,8})\s+is\s+your\s+confirmation\s+code\s*</li>`)
	// Pattern 2: \b(\d{4,8})\b is your confirmation code
	codePattern2 = regexp.MustCompile(`(?i)\b(\d{4,8})\b\s+is\s+your\s+confirmation\s+code`)
	// Pattern 3: letter-spacing: 5px;...>(\d{4,8})<  — Facebook universal HTML structure
	codePattern3 = regexp.MustCompile(`(?is)letter-spacing\s*:\s*5px;?[^>]*>\s*(\d{4,8})\s*<`)
	// Pattern 4: tiếng Việt — "53227 là mã xác nhận của bạn" (Facebook VN)
	codePattern4 = regexp.MustCompile(`(?i)\b(\d{4,8})\b\s+l[aà]\s+m[aã]\s+x[aá]c\s+nh[aậ]n`)
	// Pattern 5: subject-line — standalone N-digit code at start, followed by space/end
	codePattern5 = regexp.MustCompile(`^(\d{4,8})\s`)
	// Pattern 6: Spanish — "57603 es tu código de confirmación"
	codePattern6 = regexp.MustCompile(`(?i)\b(\d{4,8})\b\s+es\s+(?:tu|su)\s+c[oó]digo\s+de\s+confirmaci[oó]n`)
	// Pattern 7: Portuguese — "57603 é o seu código de confirmação"
	codePattern7 = regexp.MustCompile(`(?i)\b(\d{4,8})\b\s+[ée]\s+(?:o\s+seu|seu)\s+c[oó]digo\s+de\s+confirma[çc][aã]o`)
	// Pattern 8: Indonesian — "57603 adalah kode konfirmasi"
	codePattern8 = regexp.MustCompile(`(?i)\b(\d{4,8})\b\s+adalah\s+kode\s+konfirmasi`)
	// Pattern 9: French — "57603 est votre code de confirmation"
	codePattern9 = regexp.MustCompile(`(?i)\b(\d{4,8})\b\s+est\s+(?:votre|ton)\s+code\s+de\s+confirmation`)
	// Pattern 10: standalone N-digit code preceded/followed by Facebook brand mention.
	// Bắt được "Mã Facebook của bạn: 57603" hoặc "Your Facebook code is 57603" trong mọi ngôn ngữ.
	codePattern10 = regexp.MustCompile(`(?is)Facebook[^<>]{0,50}?\b(\d{5,8})\b`)
	// Pattern 11: Instagram — "123456 is your Instagram code" (subject + body EN, code đứng trước)
	codePattern11 = regexp.MustCompile(`(?i)\b(\d{4,8})\b\s+is\s+your\s+Instagram\s+code`)
	// Pattern 12: Instagram VN — "123456 là mã Instagram của bạn"
	codePattern12 = regexp.MustCompile(`(?i)\b(\d{4,8})\b\s+l[aà]\s+m[aã]\s+Instagram`)
	// Pattern 13: Instagram — "your Instagram code is <b>998877</b>" (code đứng sau, cho phép HTML tag xen giữa).
	// Bám sát keyword "Instagram code is" để tránh dính số trong footer địa chỉ.
	codePattern13 = regexp.MustCompile(`(?is)Instagram\s+code\s+is\s*(?:<[^>]*>\s*)*(\d{4,8})`)
	// Mailbox ID regex
	mailIDRegex = regexp.MustCompile(`href="/[a-z]+/email/([a-zA-Z0-9\-]+)/delete"`)
)

// Moakt implements email.Service cho moakt.com
type Moakt struct {
	client         *http.Client
	email          string
	domain         string
	customUsername string // nếu set → dùng làm prefix email thay vì random (FmUserTmpMail)
}

// SetCustomUsername set custom prefix cho email (port FmUserTmpMail).
// username: prefix đã sanitize (từ login phone/email). "" = random như cũ.
func (m *Moakt) SetCustomUsername(username string) { m.customUsername = username }

// NewMoakt tạo Moakt service.
// domain: một hoặc nhiều domain cách nhau bằng newline/dấu phẩy (vd: "tmpbox.net\nother.net").
// Mỗi lần CreateEmail sẽ random pick 1 domain từ danh sách. Để trống = dùng mặc định.
func NewMoakt(domain, proxyStr string) *Moakt {
	jar, _ := cookiejar.New(nil)
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	client.Jar = jar
	return &Moakt{client: client, domain: pickDomain(domain, MoaktDomains)}
}

// pickDomain chọn ngẫu nhiên 1 domain từ chuỗi nhiều domain (newline hoặc dấu phẩy).
// Nếu rỗng thì dùng fallback[0].
func pickDomain(raw string, fallback []string) string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ','
	})
	var domains []string
	for _, p := range parts {
		d := strings.TrimPrefix(strings.TrimSpace(p), "@")
		if d != "" {
			domains = append(domains, d)
		}
	}
	if len(domains) == 0 {
		return fallback[0]
	}
	return domains[rand.Intn(len(domains))]
}

// CreateEmail tạo email random qua POST moakt.com/en/inbox
// Mapping từ WeBM MoaktMail.CreateRandomEmailAsync()
func (m *Moakt) CreateEmail(ctx context.Context) (string, error) {
	username := randomString(9) + "_" + fmt.Sprintf("%06d", rand.Intn(1000000))
	// FmUserTmpMail override — dùng login-derived username nếu có.
	// Thêm suffix ngẫu nhiên để tránh collision nếu cùng login được dùng nhiều lần.
	if m.customUsername != "" {
		username = m.customUsername + fmt.Sprintf("_%04d", rand.Intn(10000))
	}

	form := url.Values{
		"domain":           {m.domain},
		"username":         {username},
		"setemail":         {"Create"},
		"preferred_domain": {m.domain},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", moaktBaseURL+"/en/inbox",
		strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := httpx.ReadBody(resp.Body, 0)
	bodyStr := string(body)

	// Extract email từ <span id="email-address">...</span>
	re := regexp.MustCompile(`id="email-address">(.*?)<`)
	match := re.FindStringSubmatch(bodyStr)
	if len(match) >= 2 && match[1] != "" {
		m.email = strings.TrimSpace(match[1])
		return m.email, nil
	}

	return "", fmt.Errorf("failed to create email on moakt.com")
}

// GetEmail trả về địa chỉ email đã tạo trên moakt.com (trống nếu chưa gọi CreateEmail).
func (m *Moakt) GetEmail() string { return m.email }

// Close giải phóng tài nguyên — Moakt dùng cookie jar tự quản lý, không cần đóng thủ công.
func (m *Moakt) Close() {}

// WaitForCode poll OTP code từ moakt inbox
// Mapping từ WeBM MoaktMail.WaitForFacebookCodeAsync()
func (m *Moakt) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		// Lấy danh sách mail IDs
		ids, _ := m.getMailboxIDs(ctx)
		for _, id := range ids {
			html, _ := m.getMailContent(ctx, id)
			if html == "" {
				continue
			}

			// Check sender Facebook
			htmlLower := strings.ToLower(html)
			isFb := false
			for _, sender := range fbSenders {
				if strings.Contains(htmlLower, strings.ToLower(sender)) {
					isFb = true
					break
				}
			}

			code := ExtractCode(html)
			if code != "" && (isFb || true) { // fallback: trả code nếu có
				return code, nil
			}
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

// getMailboxIDs — mapping từ WeBM MoaktMail.GetMailboxIdsAsync()
func (m *Moakt) getMailboxIDs(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", moaktBaseURL+"/en/inbox", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := httpx.ReadBody(resp.Body, 0)
	matches := mailIDRegex.FindAllStringSubmatch(string(body), -1)

	var ids []string
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) >= 2 && !seen[match[1]] {
			ids = append(ids, match[1])
			seen[match[1]] = true
		}
	}
	return ids, nil
}

// getMailContent — mapping từ WeBM MoaktMail.GetMailContentAsync()
func (m *Moakt) getMailContent(ctx context.Context, mailID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", moaktBaseURL+"/en/email/"+mailID, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := httpx.ReadBody(resp.Body, 0)
	return string(body), nil
}

// ExtractCode extract OTP code từ email content (HTML body, plain text, hoặc subject line).
//
// Thứ tự ưu tiên: pattern cụ thể (text-based, ngôn ngữ rõ ràng) → pattern HTML structural →
// pattern universal (code-at-start, Facebook brand). Càng cụ thể càng ít false positive.
//
// Subject pattern 5 (`^digit\s`) dùng cho subject; với body Facebook EU/Asia thường
// rơi vào pattern 3 (HTML letter-spacing) bất kể ngôn ngữ.
func ExtractCode(content string) string {
	if content == "" {
		return ""
	}
	patterns := []*regexp.Regexp{
		codePattern1, // EN HTML <li class="title">
		codePattern2, // EN "is your confirmation code"
		codePattern3, // Facebook HTML letter-spacing:5px (universal)
		codePattern4, // VN "là mã xác nhận"
		codePattern6, // ES "es tu código de confirmación"
		codePattern7, // PT "é seu código de confirmação"
		codePattern8, // ID "adalah kode konfirmasi"
		codePattern9, // FR "est votre code de confirmation"
		codePattern11, // IG "is your Instagram code"
		codePattern12, // IG VN "là mã Instagram"
		codePattern5, // subject "code <space>..."
		codePattern10, // Facebook + code anywhere (last resort)
		codePattern13, // Instagram + code anywhere (last resort)
	}
	for _, p := range patterns {
		if m := p.FindStringSubmatch(content); len(m) >= 2 {
			return m[1]
		}
	}
	return ""
}
