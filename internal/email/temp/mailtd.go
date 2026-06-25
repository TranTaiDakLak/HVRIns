// mailtd.go — mail.td temporary email service (api.mail.td, PoW SHA-256)
//
// API (2026-04, port từ C# MailTdAPI — đã test hoạt động):
//
//	GET  https://api.mail.td/api/domains
//	     → {"domains":[{"domain":"sugtbt.com","default":true},...]}
//	POST https://api.mail.td/api/accounts
//	     Body: {"address":"u@dom","auth_key":"<64hex>","pow":{"t":<unix_s>,"n":"<nonce>","d":<diff>}}
//	     PoW: SHA-256(address_lower + timestamp_str + counter_str) có d leading-zero bits (diff=15)
//	     Server có thể trả {"status":"retry","required_difficulty":N,"token":"..."} → tăng diff
//	     → 201 {"id":"...","address":"...","token":"eyJ..."}
//	GET  https://api.mail.td/api/accounts/{id}/messages?page=1  (Bearer)
//	     → {"messages":[{"id":"...","from":"...","subject":"..."}],"page":1}
//	GET  https://api.mail.td/api/accounts/{id}/messages/{msgId}  (Bearer)
//	     → {"html_body"|"html":"...","text_body"|"text":"..."}
//
// mail.td KHÔNG bị Cloudflare → dùng net/http qua proxy như mailcx.go.
package temp

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	mathrand "math/rand"
	"net/http"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const (
	mailTdAPIBase           = "https://api.mail.td/api"
	mailTdInitialDifficulty = 15
	mailTdUA                = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
)

// MailTd implements email.Service cho mail.td
type MailTd struct {
	client      *http.Client
	token       string
	accountID   string
	email       string
	userDomains []string // domain user tự nhập (UI) — nếu có thì random từ đây thay vì API
	LastContent string   // raw email content của lần extract thành công gần nhất (debug)
}

// NewMailTd tạo MailTd service.
// domainList: domain user tự nhập (cách nhau dấu phẩy/newline). Rỗng = random từ API.
func NewMailTd(domainList, proxyStr string) *MailTd {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	var doms []string
	for _, d := range strings.FieldsFunc(domainList, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' || r == ' ' || r == ';' }) {
		if d = strings.TrimSpace(d); d != "" {
			doms = append(doms, d)
		}
	}
	return &MailTd{client: client, userDomains: doms}
}

// addCommonHeaders thêm header chung (Origin/Referer/UA giống browser mail.td).
func (m *MailTd) addCommonHeaders(req *http.Request) {
	req.Header.Set("User-Agent", mailTdUA)
	req.Header.Set("Origin", "https://mail.td")
	req.Header.Set("Referer", "https://mail.td/")
	req.Header.Set("Accept", "application/json")
}

// mailTdFallbackDomains — dùng khi GET /domains lỗi (random thay vì chỉ sugtbt.com).
var mailTdFallbackDomains = []string{"sugtbt.com", "qabq.com", "nqmo.com", "end.tw", "uuf.me", "6n9.net"}

// fetchDomain lấy 1 domain NGẪU NHIÊN từ GET /domains (KHÔNG chỉ default sugtbt.com)
// → phân tán email qua nhiều domain, tránh FB gắn cờ 1 domain duy nhất.
func (m *MailTd) fetchDomain(ctx context.Context) string {
	// User tự nhập domain (UI) → random từ list đó (ưu tiên cao nhất, bỏ qua API).
	if len(m.userDomains) > 0 {
		return m.userDomains[mathrand.Intn(len(m.userDomains))]
	}
	fallback := func() string {
		return mailTdFallbackDomains[mathrand.Intn(len(mailTdFallbackDomains))]
	}
	req, err := http.NewRequestWithContext(ctx, "GET", mailTdAPIBase+"/domains", nil)
	if err != nil {
		return fallback()
	}
	m.addCommonHeaders(req)
	resp, err := m.client.Do(req)
	if err != nil {
		return fallback()
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 16*1024)

	var result struct {
		Domains []struct {
			Domain  string `json:"domain"`
			Default bool   `json:"default"`
		} `json:"domains"`
	}
	if err := json.Unmarshal(body, &result); err != nil || len(result.Domains) == 0 {
		return fallback()
	}
	// Random từ TẤT CẢ domain API trả về (không chỉ default).
	return result.Domains[mathrand.Intn(len(result.Domains))].Domain
}

// mailTdHasLeadingZeroBits kiểm tra hash có `bits` leading-zero bits không.
func mailTdHasLeadingZeroBits(hash []byte, bits int) bool {
	fullBytes := bits / 8
	remainBits := bits % 8
	for i := 0; i < fullBytes; i++ {
		if hash[i] != 0 {
			return false
		}
	}
	if remainBits > 0 && fullBytes < len(hash) {
		mask := byte((0xFF << (8 - remainBits)) & 0xFF)
		return hash[fullBytes]&mask == 0
	}
	return true
}

// computePoW tìm counter sao cho SHA-256(address+timestamp+counter) có `difficulty` leading-zero bits.
func mailTdComputePoW(ctx context.Context, address string, timestamp int64, difficulty int) (string, error) {
	prefix := address + fmt.Sprintf("%d", timestamp)
	for counter := 0; ; counter++ {
		if counter&0xFFFFF == 0 { // check cancel mỗi ~1M iter
			if err := ctx.Err(); err != nil {
				return "", err
			}
		}
		input := prefix + fmt.Sprintf("%d", counter)
		sum := sha256.Sum256([]byte(input))
		if mailTdHasLeadingZeroBits(sum[:], difficulty) {
			return fmt.Sprintf("%d", counter), nil
		}
	}
}

// randHex64 sinh 64 ký tự hex (32 bytes) cho auth_key.
func randHex64() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(mathrand.Intn(256))
		}
	}
	return hex.EncodeToString(b)
}

// CreateEmail: domain → random user → PoW → POST /accounts.
func (m *MailTd) CreateEmail(ctx context.Context) (string, error) {
	domain := m.fetchDomain(ctx)
	user := realisticLocalPart() // dùng helper từ mailcx.go (cùng package)
	if len(user) > 20 {
		user = user[:20]
	}
	address := user + "@" + domain
	authKey := randHex64()

	difficulty := mailTdInitialDifficulty
	var powToken string

	for retry := 0; retry <= 3; retry++ {
		timestamp := time.Now().UTC().Unix()
		nonce, err := mailTdComputePoW(ctx, address, timestamp, difficulty)
		if err != nil {
			return "", fmt.Errorf("mail.td pow: %w", err)
		}

		powObj := map[string]any{"t": timestamp, "n": nonce, "d": difficulty}
		if powToken != "" {
			powObj["token"] = powToken
		}
		bodyObj := map[string]any{"address": address, "auth_key": authKey, "pow": powObj}
		bodyJSON, _ := json.Marshal(bodyObj)

		req, err := http.NewRequestWithContext(ctx, "POST", mailTdAPIBase+"/accounts", bytes.NewReader(bodyJSON))
		if err != nil {
			return "", err
		}
		m.addCommonHeaders(req)
		req.Header.Set("Content-Type", "application/json")

		resp, err := m.client.Do(req)
		if err != nil {
			return "", fmt.Errorf("mail.td accounts: %w", err)
		}
		respBody, _ := httpx.ReadBody(resp.Body, 32*1024)
		resp.Body.Close()

		var result struct {
			Status             string `json:"status"`
			RequiredDifficulty int    `json:"required_difficulty"`
			Token              string `json:"token"`
			ID                 string `json:"id"`
			Address            string `json:"address"`
		}
		if err := json.Unmarshal(respBody, &result); err != nil {
			return "", fmt.Errorf("mail.td parse: %s", string(respBody))
		}

		// Server yêu cầu tăng difficulty
		if result.Status == "retry" {
			if result.RequiredDifficulty > 0 {
				difficulty = result.RequiredDifficulty
			}
			powToken = result.Token
			continue
		}

		if result.ID != "" && result.Address != "" && result.Token != "" {
			m.token = result.Token
			m.accountID = result.ID
			m.email = result.Address
			return m.email, nil
		}
		return "", fmt.Errorf("mail.td: response không hợp lệ: %s", string(respBody))
	}
	return "", fmt.Errorf("mail.td: PoW retry quá 3 lần")
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *MailTd) GetEmail() string { return m.email }

// Close no-op.
func (m *MailTd) Close() {}

// WaitForCode poll OTP.
func (m *MailTd) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" || m.accountID == "" {
		return "", fmt.Errorf("mail.td: chưa tạo email")
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
	return "", fmt.Errorf("mail.td: không nhận được OTP sau %d lần thử", maxRetry)
}

// MessageIDs trả về danh sách ID tin nhắn hiện có (để track tin nhắn MỚI sau add-mail).
func (m *MailTd) MessageIDs(ctx context.Context) []string {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/accounts/%s/messages?page=1", mailTdAPIBase, m.accountID), nil)
	if err != nil {
		return nil
	}
	m.addCommonHeaders(req)
	req.Header.Set("Authorization", "Bearer "+m.token)
	resp, err := m.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)
	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil
	}
	ids := make([]string, 0, len(result.Messages))
	for _, msg := range result.Messages {
		ids = append(ids, msg.ID)
	}
	return ids
}

// WaitForNewCode poll chờ tin nhắn MỚI (không nằm trong knownIDs) để lấy OTP.
// Dùng sau khi gọi add-mail để tránh lấy nhầm OTP cũ từ create.account.
func (m *MailTd) WaitForNewCode(ctx context.Context, knownIDs []string, maxRetry, intervalMs int) (string, error) {
	known := map[string]bool{}
	for _, id := range knownIDs {
		known[id] = true
	}
	if maxRetry <= 0 {
		maxRetry = 40
	}
	if intervalMs <= 0 {
		intervalMs = 3000
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		ids := m.MessageIDs(ctx)
		for _, id := range ids {
			if known[id] {
				continue // tin nhắn cũ — bỏ qua
			}
			if content, _ := m.getMessage(ctx, id); content != "" {
				if code := ExtractCode(content); code != "" {
					m.LastContent = content
					return code, nil
				}
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
	return "", fmt.Errorf("mail.td: không nhận được OTP mới sau %d lần thử", maxRetry)
}

func (m *MailTd) pollOnce(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/accounts/%s/messages?page=1", mailTdAPIBase, m.accountID), nil)
	if err != nil {
		return "", err
	}
	m.addCommonHeaders(req)
	req.Header.Set("Authorization", "Bearer "+m.token)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var result struct {
		Messages []struct {
			ID string `json:"id"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", nil
	}
	for _, msg := range result.Messages {
		if content, _ := m.getMessage(ctx, msg.ID); content != "" {
			if code := ExtractCode(content); code != "" {
				m.LastContent = content
				return code, nil
			}
		}
	}
	return "", nil
}

func (m *MailTd) getMessage(ctx context.Context, id string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/accounts/%s/messages/%s", mailTdAPIBase, m.accountID, id), nil)
	if err != nil {
		return "", err
	}
	m.addCommonHeaders(req)
	req.Header.Set("Authorization", "Bearer "+m.token)

	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	// Defensive: API có thể trả html_body/text_body HOẶC html/text — đọc hết.
	var msg struct {
		HTMLBody string `json:"html_body"`
		TextBody string `json:"text_body"`
		HTML     string `json:"html"`
		Text     string `json:"text"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", err
	}
	if msg.HTMLBody != "" {
		return msg.HTMLBody, nil
	}
	if msg.HTML != "" {
		return msg.HTML, nil
	}
	if msg.TextBody != "" {
		return msg.TextBody, nil
	}
	return msg.Text, nil
}
