// tempmailid.go — temp-mail.id service (Laravel Livewire v3, Cloudflare).
//
// Flow (reverse-engineer + xác nhận live 2026-06-19):
//   1. GET / (tls-client Chrome JA3) → csrf-token meta + cookie XSRF/tmail_session
//      + wire:snapshot của 2 component: frontend.actions (form tạo) + frontend.app (inbox)
//   2. Tạo email theo domain CHỌN: POST /livewire/update component frontend.actions
//      updates {user, domain} + calls [{method:"create"}] → snapshot.data.email = user@domain
//   3. Đọc inbox: POST /livewire/update component frontend.app (refresh) → effects.html
//      + snapshot.data.messages → ExtractCode
//
// ⚠️ BẮT BUỘC proxy RESIDENTIAL — Cloudflare chặn IP datacenter (giống tempmail.so).
// 10 domain công khai cho chọn. Mailbox bind theo cookie tmail_session (giữ trong jar).
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"math/rand"
	"net/url"
	"regexp"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const (
	tempMailIdBaseURL = "https://temp-mail.id"
	tempMailIdUA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

// tempMailIdKnownDomains — fallback (cập nhật 2026-06-19 từ dropdown Select Domain).
var tempMailIdKnownDomains = []string{
	"curuth.com", "molix.tech", "nuvox.email", "hostme.my.id", "azemo.tech",
	"aquas.live", "raxio.app", "temzo.tech", "raxel.me", "xelvo.me",
}

var (
	reTmidCsrf     = regexp.MustCompile(`<meta name="csrf-token" content="([^"]+)"`)
	reTmidSnapshot = regexp.MustCompile(`wire:snapshot="([^"]+)"`)
	reTmidName     = regexp.MustCompile(`"name":"([^"]+)"`)
	reTmidDomains  = regexp.MustCompile(`(?i)"domains?"\s*:\s*\[\s*\[([^\]]+)\]`)
)

// TempMailId implements email.Service cho temp-mail.id.
type TempMailId struct {
	proxyStr      string
	client        tls_client.HttpClient
	email         string
	csrf          string
	appSnapshot   string // frontend.app snapshot — cập nhật mỗi lần poll (Livewire v3 stateful)
	pinnedDomains []string
}

// NewTempMailId tạo TempMailId service. pinnedDomains rỗng = random 10 domain.
func NewTempMailId(proxyStr string, pinnedDomains []string) *TempMailId {
	return &TempMailId{proxyStr: proxyStr, pinnedDomains: pinnedDomains}
}

// tempMailIdUsername sinh username SẠCH (chữ thường + số, bắt đầu bằng chữ, 8-11 ký tự).
// temp-mail.id KHÔNG chấp nhận gạch dưới/quá dài → mailbox không provision (RCPT 550).
func tempMailIdUsername() string {
	const lower = "abcdefghijklmnopqrstuvwxyz"
	const alnum = "abcdefghijklmnopqrstuvwxyz0123456789"
	n := 8 + rand.Intn(4)
	b := make([]byte, n)
	b[0] = lower[rand.Intn(len(lower))]
	for i := 1; i < n; i++ {
		b[i] = alnum[rand.Intn(len(alnum))]
	}
	return string(b)
}

// ParseTempMailIdDomains tách "a.com, b.tech" → []string.
func ParseTempMailIdDomains(raw string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' }) {
		if d := strings.TrimPrefix(strings.TrimSpace(p), "@"); d != "" {
			out = append(out, d)
		}
	}
	return out
}

func (t *TempMailId) newClient() (tls_client.HttpClient, error) {
	// Sticky residential — giữ 1 IP residential cho cả phiên (Cloudflare + cookie session).
	sticky := proxy.EnsureStickySession(t.proxyStr)
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithInsecureSkipVerify(),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
	}
	if f := proxy.FormatProxyURL(sticky); f != "" {
		opts = append(opts, tls_client.WithProxyUrl(f))
	}
	return tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
}

func (t *TempMailId) get(ctx context.Context, url string) (string, int, error) {
	req, err := fhttp.NewRequest("GET", url, nil)
	if err != nil {
		return "", 0, err
	}
	req = req.WithContext(ctx)
	req.Header = fhttp.Header{
		"user-agent":      {tempMailIdUA},
		"accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		"accept-language": {"en-US,en;q=0.9"},
		"referer":         {tempMailIdBaseURL + "/"},
		fhttp.HeaderOrderKey: {"user-agent", "accept", "accept-language", "referer"},
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 2*1024*1024)
	return string(body), resp.StatusCode, nil
}

// postLivewire gửi POST /livewire/update với 1 component (snapshot + updates + calls).
// Trả về (componentSnapshotMới, effectsHTML, error).
func (t *TempMailId) postLivewire(ctx context.Context, snapshot string, updates map[string]interface{}, calls []map[string]interface{}) (string, string, error) {
	if updates == nil {
		updates = map[string]interface{}{}
	}
	if calls == nil {
		calls = []map[string]interface{}{}
	}
	reqBody, _ := json.Marshal(map[string]interface{}{
		"_token": t.csrf,
		"components": []map[string]interface{}{
			{"snapshot": snapshot, "updates": updates, "calls": calls},
		},
	})
	req, err := fhttp.NewRequest("POST", tempMailIdBaseURL+"/livewire/update", strings.NewReader(string(reqBody)))
	if err != nil {
		return "", "", err
	}
	req = req.WithContext(ctx)
	req.Header = fhttp.Header{
		"user-agent":   {tempMailIdUA},
		"content-type": {"application/json"},
		"accept":       {"text/html, application/xhtml+xml"},
		"x-livewire":   {""},
		"x-csrf-token": {t.csrf},
		"referer":      {tempMailIdBaseURL + "/"},
		"origin":       {tempMailIdBaseURL},
		fhttp.HeaderOrderKey: {"user-agent", "content-type", "accept", "x-livewire", "x-csrf-token", "referer", "origin"},
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 512*1024)

	var result struct {
		Components []struct {
			Snapshot string `json:"snapshot"`
			Effects  struct {
				HTML string `json:"html"`
			} `json:"effects"`
		} `json:"components"`
	}
	if err := json.Unmarshal(body, &result); err != nil || len(result.Components) == 0 {
		return "", "", fmt.Errorf("livewire parse (HTTP %d): %.200s", resp.StatusCode, body)
	}
	return result.Components[0].Snapshot, result.Components[0].Effects.HTML, nil
}

// snapshotOf tách wire:snapshot của component theo tên (HTML-unescaped).
func tempMailIdSnapshotOf(htmlBody, compName string) string {
	for _, m := range reTmidSnapshot.FindAllStringSubmatch(htmlBody, -1) {
		s := html.UnescapeString(m[1])
		if strings.Contains(s, `"name":"`+compName+`"`) {
			return s
		}
	}
	return ""
}

// emailFromSnapshot tách data.email từ 1 snapshot JSON.
func emailFromSnapshot(snapshot string) string {
	var snap struct {
		Data struct {
			Email string `json:"email"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(snapshot), &snap)
	return snap.Data.Email
}

// cookieString lấy cookie session hiện tại (để persist qua snapshot).
func (t *TempMailId) cookieString() string {
	if t.client == nil {
		return ""
	}
	u, _ := url.Parse(tempMailIdBaseURL)
	var parts []string
	for _, ck := range t.client.GetCookies(u) {
		parts = append(parts, ck.Name+"="+ck.Value)
	}
	return strings.Join(parts, "; ")
}

// RestoreSession rebuild tls-client + nạp lại cookie + state (gọi từ Restore lúc verify).
// temp-mail.id bind mailbox theo cookie tmail_session (KHÔNG theo IP) → verify dùng lại
// cookie reg là đọc được mailbox cũ, miễn proxy residential (qua Cloudflare).
func (t *TempMailId) RestoreSession(email, csrf, appSnapshot, cookies string) error {
	t.email, t.csrf, t.appSnapshot = email, csrf, appSnapshot
	c, err := t.newClient()
	if err != nil {
		return err
	}
	t.client = c
	if cookies != "" {
		u, _ := url.Parse(tempMailIdBaseURL)
		var cks []*fhttp.Cookie
		for _, kv := range strings.Split(cookies, ";") {
			kv = strings.TrimSpace(kv)
			if i := strings.IndexByte(kv, '='); i > 0 {
				cks = append(cks, &fhttp.Cookie{Name: kv[:i], Value: kv[i+1:]})
			}
		}
		t.client.SetCookies(u, cks)
	}
	return nil
}

// SnapshotFields trả các field cần persist (cho Snapshotter).
func (t *TempMailId) SnapshotFields() (email, csrf, appSnapshot, cookies string) {
	return t.email, t.csrf, t.appSnapshot, t.cookieString()
}

// CreateEmail: GET / → tạo email theo domain chọn qua frontend.actions.
func (t *TempMailId) CreateEmail(ctx context.Context) (string, error) {
	c, err := t.newClient()
	if err != nil {
		return "", fmt.Errorf("tempmailid: client: %w", err)
	}
	t.client = c

	htmlBody, status, err := t.get(ctx, tempMailIdBaseURL+"/")
	if err != nil {
		return "", fmt.Errorf("tempmailid GET /: %w", err)
	}
	if status == 403 || strings.Contains(htmlBody, "Just a moment") {
		return "", fmt.Errorf("tempmailid: Cloudflare chặn (cần proxy residential)")
	}
	if m := reTmidCsrf.FindStringSubmatch(htmlBody); len(m) >= 2 {
		t.csrf = m[1]
	}
	actionsSnap := tempMailIdSnapshotOf(htmlBody, "frontend.actions")
	if t.csrf == "" || actionsSnap == "" {
		return "", fmt.Errorf("tempmailid: thiếu csrf/snapshot (HTTP %d)", status)
	}

	// Chọn domain: pinned hoặc 10 domain công khai.
	domains := t.pinnedDomains
	if len(domains) == 0 {
		domains = tempMailIdKnownDomains
	}
	domain := domains[rand.Intn(len(domains))]
	user := tempMailIdUsername()

	// POST create(user, domain) → email = user@domain.
	newSnap, _, err := t.postLivewire(ctx, actionsSnap,
		map[string]interface{}{"user": user, "domain": domain},
		[]map[string]interface{}{{"path": "", "method": "create", "params": []interface{}{}}})
	if err != nil {
		return "", fmt.Errorf("tempmailid create: %w", err)
	}
	t.email = emailFromSnapshot(newSnap)
	if t.email == "" {
		t.email = user + "@" + domain // fallback theo input
	}

	// GET /mailbox → frontend.app snapshot (bind email mới) cho poll.
	// Email AUTHORITATIVE = session mailbox (cái THẬT nhận mail), ưu tiên hơn email
	// từ create-response (phòng khi server sanitize username).
	mb, _, err := t.get(ctx, tempMailIdBaseURL+"/mailbox")
	if err == nil {
		if s := tempMailIdSnapshotOf(mb, "frontend.app"); s != "" {
			t.appSnapshot = s
			if sessEmail := emailFromSnapshot(s); sessEmail != "" {
				t.email = sessEmail
			}
		}
		// csrf trang /mailbox có thể khác — cập nhật.
		if m := reTmidCsrf.FindStringSubmatch(mb); len(m) >= 2 {
			t.csrf = m[1]
		}
	}
	return t.email, nil
}

// GetEmail trả về địa chỉ đã tạo.
func (t *TempMailId) GetEmail() string { return t.email }

// Close no-op.
func (t *TempMailId) Close() {}

// WaitForCode poll OTP từ frontend.app.
func (t *TempMailId) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 3000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmailid: chưa tạo email")
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
	return "", fmt.Errorf("tempmailid: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempMailId) pollOnce(ctx context.Context) (string, error) {
	// 1. GET /mailbox → quét full page (phòng khi message render server-side)
	//    + làm tươi snapshot/csrf cho Livewire refresh.
	if mb, _, err := t.get(ctx, tempMailIdBaseURL+"/mailbox"); err == nil {
		if code := ExtractCode(mb); code != "" {
			return code, nil
		}
		if s := tempMailIdSnapshotOf(mb, "frontend.app"); s != "" {
			t.appSnapshot = s
		}
		if m := reTmidCsrf.FindStringSubmatch(mb); len(m) >= 2 {
			t.csrf = m[1]
		}
	}
	if t.appSnapshot == "" {
		return "", nil
	}

	// 2. Gọi fetchMessages (frontend.app listener) — đây MỚI là cách fetch mail.
	//    Refresh RỖNG không trigger fetchMessages → messages mãi rỗng dù mail có đó.
	//    syncEmail trước để component biết email (phòng khi cần).
	newSnap, effectsHTML, err := t.postLivewire(ctx, t.appSnapshot, nil,
		[]map[string]interface{}{
			{"path": "", "method": "syncEmail", "params": []interface{}{t.email}},
			{"path": "", "method": "fetchMessages", "params": []interface{}{}},
		})
	if err != nil {
		return "", nil
	}
	if newSnap != "" {
		t.appSnapshot = newSnap // giữ snapshot mới (Livewire v3 stateful)
	}
	if code := ExtractCode(effectsHTML); code != "" {
		return code, nil
	}
	if code := ExtractCode(newSnap); code != "" {
		return code, nil
	}
	return "", nil
}

// ─── Domain list ────────────────────────────────────────────────────────────

// TempMailIdDomainsResult — kết quả FetchTempMailIdDomains.
type TempMailIdDomainsResult struct {
	Domains []string `json:"domains"`
}

// FetchTempMailIdDomains lấy 10 domain công khai từ snapshot frontend.actions.
// BẮT BUỘC proxy residential (Cloudflare). proxyStr rỗng = fallback hardcoded.
func FetchTempMailIdDomains(ctx context.Context, proxyStr string) (*TempMailIdDomainsResult, error) {
	t := &TempMailId{proxyStr: proxyStr}
	c, err := t.newClient()
	if err != nil {
		return &TempMailIdDomainsResult{Domains: tempMailIdKnownDomains}, nil
	}
	t.client = c
	htmlBody, status, err := t.get(ctx, tempMailIdBaseURL+"/")
	if err != nil || status == 403 {
		return &TempMailIdDomainsResult{Domains: tempMailIdKnownDomains}, nil
	}
	snap := tempMailIdSnapshotOf(htmlBody, "frontend.actions")
	if m := reTmidDomains.FindStringSubmatch(snap); len(m) >= 2 {
		var out []string
		for _, d := range strings.Split(m[1], ",") {
			d = strings.Trim(strings.TrimSpace(d), `"`)
			if d != "" && strings.Contains(d, ".") {
				out = append(out, d)
			}
		}
		if len(out) > 0 {
			return &TempMailIdDomainsResult{Domains: out}, nil
		}
	}
	return &TempMailIdDomainsResult{Domains: tempMailIdKnownDomains}, nil
}
