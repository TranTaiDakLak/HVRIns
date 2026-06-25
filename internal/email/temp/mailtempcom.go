// mailtempcom.go — mail-temp.com service (HTML scraping, client-side email).
// Port từ C# MailTempComAPI. Flow: user = random prefix, domain = random từ list.
// Inbox lấy qua GET /temp-mail-box/ với Cookie surl={domain}%2F{user}, parse HTML.
package temp

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const mailTempComBaseURL = "https://mail-temp.com"
const mailTempComCacheTTL = 48 * time.Hour

// Default domains fallback — được mở rộng thủ công theo danh sách thực tế của mail-temp.com.
var mailTempComDefaultDomains = []string{
	"afse-gh.top", "banancaocap.com", "bosakun.com", "gardu.dev",
	"googl.win", "gurur.store", "herilev.top", "k4money.com",
	"kakao-mail.com", "skytopway.com", "waroengpt.com",
	"fplq.xyz", "mailx.click", "rxwk.store", "svfp.xyz",
	"qvab.store", "txmm.online", "bpjr.store", "mzft.store",
	"nhvt.store", "pkhp.store", "rtwz.store", "txby.online",
	"vzaq.store", "xpmq.store", "ydlb.store", "zhjk.store",
	"mqke.store", "nqlp.store", "opkl.store", "pqmn.store",
	"rstv.store", "stuv.store", "tuvw.store", "uvwx.store",
}

var (
	// Regex extract domain từ HTML homepage (ví dụ: 'mailx.click', "fplq.xyz")
	mailTempComDomainRe = regexp.MustCompile(`['"]([a-z0-9\-]{3,}\.[a-z]{2,})['"]`)
	// Inbox table container
	mailTempComTableRe = regexp.MustCompile(`(?is)<div[^>]+id=["']email-table["'][^>]*>([\s\S]+)`)
	// Preview row: class chứa list-group-item-info hoặc list-group-item-success
	mailTempComRowRe = regexp.MustCompile(`(?is)<div[^>]+class=["'][^"']*list-group-item-(?:info|success)[^"']*["'][^>]*>([\s\S]+?)</div>\s*</div>`)
	// Message body block (inline content cùng trang)
	mailTempComBodyRe = regexp.MustCompile(`(?is)<div[^>]+class=["'][^"']*mess_bodiyy[^"']*["'][^>]*>([\s\S]+?)</div>\s*</div>\s*</div>\s*</div>`)

	// Domain cache — shared across all instances.
	domainCacheMu   sync.Mutex
	domainCacheList []string
	domainCacheAt   time.Time
	domainCacheFile string // set by SetMailTempComDomainCachePath
)

// SetMailTempComDomainCachePath đặt đường dẫn file cache domain.
// Gọi khi khởi động app. Sau đó mỗi lần fetch domain thành công sẽ ghi vào file này
// để lần sau dùng lại mà không cần fetch lại (TTL 48 giờ).
func SetMailTempComDomainCachePath(path string) {
	domainCacheMu.Lock()
	domainCacheFile = path
	domainCacheMu.Unlock()
}

// MailTempCom implements email.Service cho mail-temp.com.
type MailTempCom struct {
	client *http.Client
	user   string
	domain string
	email  string
}

// NewMailTempCom tạo MailTempCom service. Email sinh client-side (không call API create).
func NewMailTempCom(proxyStr string) *MailTempCom {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	c.Jar = jar
	return &MailTempCom{client: c}
}

// CreateEmail gen email local — user random + domain random từ pool.
// Domain list được cache 48 giờ (memory + file). Fallback về default list nếu fetch fail.
func (m *MailTempCom) CreateEmail(ctx context.Context) (string, error) {
	domains := getOrFetchDomains(ctx, m.client)
	if len(domains) == 0 {
		domains = mailTempComDefaultDomains
	}
	m.domain = domains[rand.Intn(len(domains))]
	m.user = realisticLocalPart()
	m.email = m.user + "@" + m.domain
	return m.email, nil
}

// getOrFetchDomains — đọc từ cache (memory hoặc file) nếu còn tươi; fetch từ web nếu cũ.
func getOrFetchDomains(ctx context.Context, client *http.Client) []string {
	domainCacheMu.Lock()
	cachePath := domainCacheFile
	// Cache memory còn tươi
	if len(domainCacheList) > 0 && time.Since(domainCacheAt) < mailTempComCacheTTL {
		out := domainCacheList
		domainCacheMu.Unlock()
		return out
	}
	domainCacheMu.Unlock()

	// Thử load từ file nếu có
	if cachePath != "" {
		if domains, modTime := loadDomainFile(cachePath); len(domains) > 0 && time.Since(modTime) < mailTempComCacheTTL {
			domainCacheMu.Lock()
			domainCacheList = domains
			domainCacheAt = modTime
			domainCacheMu.Unlock()
			return domains
		}
	}

	// Fetch từ web
	domains := fetchMailTempComDomains(ctx, client)
	if len(domains) > 0 {
		domainCacheMu.Lock()
		domainCacheList = domains
		domainCacheAt = time.Now()
		domainCacheMu.Unlock()
		if cachePath != "" {
			saveDomainFile(cachePath, domains)
		}
	}
	return domains
}

func fetchMailTempComDomains(ctx context.Context, client *http.Client) []string {
	req, _ := http.NewRequestWithContext(ctx, "GET", mailTempComBaseURL+"/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 512*1024)
	seen := map[string]bool{}
	var out []string
	for _, m := range mailTempComDomainRe.FindAllSubmatch(body, -1) {
		d := string(m[1])
		if !seen[d] && strings.Count(d, ".") >= 1 {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}

func loadDomainFile(path string) ([]string, time.Time) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, time.Time{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, time.Time{}
	}
	var domains []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && strings.ContainsRune(line, '.') && !strings.HasPrefix(line, "#") {
			domains = append(domains, line)
		}
	}
	return domains, fi.ModTime()
}

func saveDomainFile(path string, domains []string) {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	_ = os.WriteFile(path, []byte(strings.Join(domains, "\n")+"\n"), 0644)
}

// GetEmail trả về địa chỉ email đã tạo.
func (m *MailTempCom) GetEmail() string { return m.email }

// Close no-op.
func (m *MailTempCom) Close() {}

// WaitForCode poll OTP từ inbox.
func (m *MailTempCom) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if m.email == "" {
		return "", fmt.Errorf("mailtempcom: chưa tạo email")
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
	return "", fmt.Errorf("mailtempcom: không nhận được OTP sau %d lần thử", maxRetry)
}

func (m *MailTempCom) pollOnce(ctx context.Context) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", mailTempComBaseURL+"/temp-mail-box/", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", mailTempComBaseURL+"/")
	// Cookie surl={domain}%2F{user} — server dùng để identify mailbox
	req.Header.Set("Cookie", "surl="+url.QueryEscape(m.domain+"/"+m.user))
	resp, err := m.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 512*1024)

	// Extract #email-table block
	tableMatch := mailTempComTableRe.FindSubmatch(body)
	if len(tableMatch) < 2 {
		return "", nil
	}
	tableHTML := tableMatch[1]

	// Parse body blocks (inline content cùng trang)
	bodyMatches := mailTempComBodyRe.FindAllSubmatch(tableHTML, -1)
	for _, b := range bodyMatches {
		content := string(b[1])
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}

	// Fallback: parse rows + subject nếu body không có code
	rowMatches := mailTempComRowRe.FindAllSubmatch(tableHTML, -1)
	for _, r := range rowMatches {
		content := string(r[1])
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}
