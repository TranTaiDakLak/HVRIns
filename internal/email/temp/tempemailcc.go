// tempemailcc.go — tempemail.cc service (mail.tm-compatible JSON API)
//
// Flow (xác nhận qua research 2026-06-19):
//   1. GET  /api/domains                        → ["icmans.com", ...]
//   2. POST /api/accounts {email, password}     → {code:200, data:{token:JWT}}
//      Username PHẢI ≥7 chars; Chrome UA bắt buộc (plain UA → 403).
//   3. GET  /api/messages?limit=10 (Bearer JWT) → [{id,subject,intro,...}]
//   4. GET  /api/messages/{id}                  → {html:[...], text:"..."}
//
// QUAN TRỌNG: dùng www.tempemail.cc (non-www redirect về www).
//             Phải dùng Chrome User-Agent — UA khác bị 403.
package temp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

// jsonIDStr chuyển JSON-decoded ID (float64 hoặc string) sang string dùng trong URL.
// encoding/json decode số nguyên thành float64 khi target là interface{}.
// Dùng strconv.FormatInt để tránh scientific notation với số lớn (1e+06).
func jsonIDStr(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case float64:
		return strconv.FormatInt(int64(x), 10)
	case string:
		return x
	default:
		return fmt.Sprintf("%v", v)
	}
}

const (
	tempEmailCCBase = "https://www.tempemail.cc/api"
	// Chrome UA bắt buộc — UA thường bị 403.
	tempEmailCCUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36"
)

// tempEmailCCKnownDomains — fallback nếu GET /api/domains fail.
var tempEmailCCKnownDomains = []string{"icmans.com"}

// TempEmailCC implements email.Service cho tempemail.cc.
type TempEmailCC struct {
	client *http.Client
	email  string
	token  string // JWT từ POST /api/accounts
}

// NewTempEmailCC tạo TempEmailCC service.
func NewTempEmailCC(proxyStr string) *TempEmailCC {
	return &TempEmailCC{client: proxy.CreateClient(proxyStr, 30*time.Second)}
}

// tempEmailCCPassword sinh password n ký tự, ĐẢM BẢO có ≥1 chữ hoa, ≥1 chữ thường, ≥1 số.
// API tempemail.cc reject "password must contain at least 1 number" nếu thiếu chữ số.
func tempEmailCCPassword(n int) string {
	if n < 4 {
		n = 8
	}
	const upper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const lower = "abcdefghijklmnopqrstuvwxyz"
	const digit = "0123456789"
	const all = upper + lower + digit
	b := make([]byte, n)
	// 3 vị trí đầu cố định mỗi loại để bảo đảm policy, phần còn lại random.
	b[0] = upper[rand.Intn(len(upper))]
	b[1] = lower[rand.Intn(len(lower))]
	b[2] = digit[rand.Intn(len(digit))]
	for i := 3; i < n; i++ {
		b[i] = all[rand.Intn(len(all))]
	}
	// Xáo trộn để 3 ký tự bắt buộc không luôn ở đầu.
	rand.Shuffle(n, func(i, j int) { b[i], b[j] = b[j], b[i] })
	return string(b)
}

func (t *TempEmailCC) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", tempEmailCCUA)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://www.tempemail.cc")
	req.Header.Set("Referer", "https://www.tempemail.cc/")
}

func (t *TempEmailCC) pickDomain(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tempEmailCCBase+"/domains", nil)
	if err != nil {
		return tempEmailCCKnownDomains[0], nil
	}
	t.setHeaders(req)
	resp, err := t.client.Do(req)
	if err != nil {
		return tempEmailCCKnownDomains[0], nil
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 32*1024)

	var domains []string
	if err := json.Unmarshal(body, &domains); err != nil || len(domains) == 0 {
		return tempEmailCCKnownDomains[0], nil
	}
	return domains[rand.Intn(len(domains))], nil
}

// CreateEmail: pick domain → POST /api/accounts → JWT.
func (t *TempEmailCC) CreateEmail(ctx context.Context) (string, error) {
	domain, err := t.pickDomain(ctx)
	if err != nil {
		domain = tempEmailCCKnownDomains[0]
	}

	// realisticLocalPart() luôn ≥7 chars (yêu cầu của API)
	local := realisticLocalPart()
	addr := local + "@" + domain
	// Password phải có ≥1 chữ số (API reject "password must contain at least 1 number").
	pass := tempEmailCCPassword(12)

	payload, _ := json.Marshal(map[string]string{
		"email":    addr,
		"password": pass,
	})
	req, err := http.NewRequestWithContext(ctx, "POST",
		tempEmailCCBase+"/accounts", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	t.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tempemailcc create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("tempemailcc create parse: %w — body: %.200s", err, body)
	}
	// Kiểm tra token thay vì code cứng (code có thể là 0/200/201 tuỳ version API)
	if result.Data.Token == "" {
		return "", fmt.Errorf("tempemailcc create: no token — code=%d msg=%q (HTTP %d) — body: %.200s",
			result.Code, result.Message, resp.StatusCode, body)
	}
	t.email = addr
	t.token = result.Data.Token
	return t.email, nil
}

// GetEmail trả về địa chỉ đã tạo.
func (t *TempEmailCC) GetEmail() string { return t.email }

// Close no-op.
func (t *TempEmailCC) Close() {}

// WaitForCode poll OTP từ inbox.
func (t *TempEmailCC) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.token == "" {
		return "", fmt.Errorf("tempemailcc: chưa tạo email")
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if code, _ := t.pollOnce(ctx); code != "" {
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
	return "", fmt.Errorf("tempemailcc: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempEmailCC) pollOnce(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tempEmailCCBase+"/messages?limit=10", nil)
	if err != nil {
		return "", err
	}
	t.setHeaders(req)
	req.Header.Set("Authorization", "Bearer "+t.token)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	// Inbox THẬT (xác nhận live 2026-06-19) bọc envelope:
	//   {"code":200,"data":{"items":[{"id","subject","summary","from":{...}}],"pagination":{...}}}
	// OTP nằm sẵn trong subject + summary. (Code cũ parse mảng trần → fail âm thầm.)
	var result struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				ID      interface{} `json:"id"`
				Subject string      `json:"subject"`
				Summary string      `json:"summary"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", nil
	}
	for _, m := range result.Data.Items {
		if code := ExtractCode(m.Subject); code != "" {
			return code, nil
		}
		if code := ExtractCode(m.Summary); code != "" {
			return code, nil
		}
		if idStr := jsonIDStr(m.ID); idStr != "" {
			if content, _ := t.getMessage(ctx, idStr); content != "" {
				if code := ExtractCode(content); code != "" {
					return code, nil
				}
			}
		}
	}
	return "", nil
}

func (t *TempEmailCC) getMessage(ctx context.Context, id string) (string, error) {
	// Endpoint full-content documented = /api/messages/{id}/raw → {code,data:{raw:"<MIME>"}}.
	// (Docs: /api/messages/{id} KHÔNG có body field tài liệu; chỉ /raw có data.raw.)
	req, err := http.NewRequestWithContext(ctx, "GET", tempEmailCCBase+"/messages/"+id+"/raw", nil)
	if err != nil {
		return "", err
	}
	t.setHeaders(req)
	req.Header.Set("Authorization", "Bearer "+t.token)

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	var result struct {
		Data struct {
			Raw string `json:"raw"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err == nil && result.Data.Raw != "" {
		return result.Data.Raw, nil
	}
	// Fallback: quét nguyên raw body (subject/summary nằm trong đó).
	return string(body), nil
}
