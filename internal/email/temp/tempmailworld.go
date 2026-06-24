// tempmailworld.go — tempmail.world service (Next.js, mailbox bind theo IP).
//
// Flow (xác nhận live 2026-06-19):
//   1. GET /api/domains   → {"id","email","sessionToken":"1234","expires"}
//      (vừa tạo guest mailbox vừa set cookie sessionToken=1234 — HẰNG SỐ chung)
//   2. GET /api/guestInbox            → [{id,from,name,subject,body,date}]  (KHÔNG truyền email)
//   3. GET /api/guestInbox/message/{id} → {..., htmlBody}
//
// ⚠️ QUAN TRỌNG — mailbox BIND THEO IP, KHÔNG theo cookie/param:
//   - sessionToken=1234 chỉ là cổng "có-cookie" (thiếu → 401), KHÔNG định danh mailbox.
//   - Server gắn "IP egress → email" lúc gọi /api/domains. /api/guestInbox trả mail
//     theo IP đang gọi (bỏ qua mọi param email).
//   - HỆ QUẢ: CreateEmail và WaitForCode PHẢI CÙNG 1 IP. Nếu proxy XOAY VÒNG mỗi
//     request (mỗi lần 1 IP khác) → tạo mail IP A, đọc inbox IP B → LUÔN RỖNG.
//     → tempmail.world cần proxy STICKY (giữ IP), KHÔNG dùng proxy rotating.
//   - Mỗi lần gọi /api/domains LẠI gán IP sang email MỚI → chỉ gọi 1 lần/mailbox.
//
// KHÔNG cần key, KHÔNG Cloudflare.
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strconv"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

// tempMailWorldStickyTTLMin — số phút giữ NGUYÊN IP cho 1 session, phủ khoảng cách
// thời gian reg→verify. iprocket/IPIDEA: -session-<id> KHÔNG kèm time chỉ giữ ~5 phút
// rồi đổi IP → reg tạo mail, verify (sau >5') đọc IP khác → rỗng. Thêm -sessTime-N
// (tối đa 120) để giữ lâu. 60' phủ hầu hết luồng inline + split mode.
const tempMailWorldStickyTTLMin = 60

var reTMWSession = regexp.MustCompile(`(?i)(-session-[a-z0-9]+)`)

// buildTMWStickyProxy: sticky proxy + kéo dài TTL giữ IP.
// EnsureStickySession pin IP qua -session-<id> nhưng (với zone proxy như iprocket)
// không set thời gian → default ~5' rồi rotate. Chèn -sessTime-N để giữ đủ lâu.
func buildTMWStickyProxy(proxyStr string) string {
	sticky := proxy.EnsureStickySession(proxyStr)
	if sticky == "" || sticky == proxyStr {
		return sticky // proxy tĩnh / không session-capable → giữ nguyên
	}
	low := strings.ToLower(sticky)
	// Đã có cơ chế thời gian (abcproxy -sessTime-, region -t-, proxyshare _life-) → thôi.
	if strings.Contains(low, "sesstime-") || strings.Contains(low, "-t-") || strings.Contains(low, "_life-") {
		return sticky
	}
	// Zone proxy (iprocket): có -session-<id> nhưng chưa có time → thêm -sessTime-N.
	if reTMWSession.MatchString(sticky) {
		return reTMWSession.ReplaceAllString(sticky, "${1}-sessTime-"+strconv.Itoa(tempMailWorldStickyTTLMin))
	}
	return sticky
}

const tempMailWorldBaseURL = "https://tempmail.world"

// tempMailWorldSessionCookie — sessionToken là HẰNG SỐ "1234" chung cho mọi guest
// (chỉ là cổng "có-cookie", không định danh mailbox; mailbox bind theo IP). Set thẳng
// để luồng REUSE (verify không gọi /api/domains) vẫn qua được gate, tránh 401.
const tempMailWorldSessionCookie = "1234"

// TempMailWorld implements email.Service cho tempmail.world.
type TempMailWorld struct {
	client      *http.Client
	email       string
	stickyProxy string // proxy đã sticky (pin IP) lúc tạo mailbox — persist để verify đọc CÙNG IP
}

// NewTempMailWorld tạo TempMailWorld service.
//
// QUAN TRỌNG: mailbox bind theo IP egress (xem doc đầu file). Proxy residential
// thường XOAY VÒNG IP mỗi request → tạo mail 1 IP, đọc inbox IP khác → luôn rỗng.
// EnsureStickySession thêm -session-<id> CỐ ĐỊNH (tạo 1 lần, dùng suốt vòng đời
// instance) → mọi request cùng 1 IP. Xác nhận live với iprocket: -session-X pin IP.
func NewTempMailWorld(proxyStr string) *TempMailWorld {
	sticky := buildTMWStickyProxy(proxyStr)
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(sticky, 30*time.Second)
	c.Jar = jar
	return &TempMailWorld{client: c, stickyProxy: sticky}
}

// rebuildClient tạo lại http client với sticky proxy ĐÃ LƯU (gọi từ Restore lúc verify)
// → verify đọc inbox CÙNG IP egress mà reg đã bind mailbox. Bắt buộc cho mailbox bind-IP.
func (t *TempMailWorld) rebuildClient(stickyProxy string) {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(stickyProxy, 30*time.Second)
	c.Jar = jar
	t.client = c
	t.stickyProxy = stickyProxy
}

func (t *TempMailWorld) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tempMailWorldBaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", tempMailWorldBaseURL+"/")
	// Luôn gửi sessionToken=1234 (hằng số) — luồng reuse không gọi /api/domains nên
	// jar không tự có cookie → thiếu là 401. Set thẳng để pollOnce luôn qua gate.
	req.Header.Set("Cookie", "sessionToken="+tempMailWorldSessionCookie)
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := httpx.ReadBody(resp.Body, 0)
	return b, nil
}

// CreateEmail tạo guest mailbox qua GET /api/domains (set session cookie).
func (t *TempMailWorld) CreateEmail(ctx context.Context) (string, error) {
	body, err := t.get(ctx, "/api/domains")
	if err != nil {
		return "", fmt.Errorf("tempmailworld create: %w", err)
	}
	var result struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Email == "" {
		snippet := strings.TrimSpace(string(body))
		if len(snippet) > 200 {
			snippet = snippet[:200]
		}
		return "", fmt.Errorf("tempmailworld create: no email snippet=%q", snippet)
	}
	t.email = result.Email
	return t.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (t *TempMailWorld) GetEmail() string { return t.email }

// Close no-op.
func (t *TempMailWorld) Close() {}

// WaitForCode poll OTP từ /api/guestInbox.
func (t *TempMailWorld) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmailworld: chưa tạo email")
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
	return "", fmt.Errorf("tempmailworld: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempMailWorld) pollOnce(ctx context.Context) (string, error) {
	body, err := t.get(ctx, "/api/guestInbox")
	if err != nil {
		return "", err
	}
	// Inbox = mảng trần [{id,from,name,subject,body,date}]. Parse field-agnostic:
	// ExtractCode trên TOÀN BỘ raw mỗi message (bắt subject/body dù tên field khác),
	// + fallback getMessage(id) lấy htmlBody đầy đủ.
	var rawMsgs []json.RawMessage
	if err := json.Unmarshal(body, &rawMsgs); err != nil {
		return "", nil
	}
	for _, raw := range rawMsgs {
		if code := ExtractCode(string(raw)); code != "" {
			return code, nil
		}
		var meta struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(raw, &meta)
		if meta.ID != "" {
			if content := t.getMessage(ctx, meta.ID); content != "" {
				if code := ExtractCode(content); code != "" {
					return code, nil
				}
			}
		}
	}
	return "", nil
}

func (t *TempMailWorld) getMessage(ctx context.Context, id string) string {
	body, err := t.get(ctx, "/api/guestInbox/message/"+id)
	if err != nil {
		return ""
	}
	var msg struct {
		Subject  string `json:"subject"`
		From     string `json:"from"`
		HTMLBody string `json:"htmlBody"`
		Body     string `json:"body"`
		HTML     string `json:"html"`
		Text     string `json:"text"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return string(body)
	}
	combined := msg.Subject + "\n" + msg.HTMLBody + "\n" + msg.Body + "\n" + msg.HTML + "\n" + msg.Text
	if strings.TrimSpace(combined) == "" {
		return string(body) // fallback: quét raw
	}
	return combined
}
