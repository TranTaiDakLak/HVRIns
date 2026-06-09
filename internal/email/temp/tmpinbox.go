// tmpinbox.go — TmpInbox.com service (client-side email, HTML parse)
// Port từ C# TmpInboxAPI. Email sinh client-side, inbox lấy qua HTML scraping.
package temp

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const tmpInboxBaseURL = "https://tmpinbox.com"

var (
	tmpInboxMsgIDRegex = regexp.MustCompile(`hx-get="/inbox/[^/]+/message/(\d+)"`)
	tmpInboxBodyRegex  = regexp.MustCompile(`(?is)<pre[^>]*>(.*?)</pre>`)
	tmpInboxHTMLRegex  = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*message-html-content[^"]*"[^>]*>(.*?)</div>`)
)

// TmpInbox implements email.Service cho tmpinbox.com.
type TmpInbox struct {
	client *http.Client
	email  string
}

// NewTmpInbox tạo TmpInbox service.
func NewTmpInbox(proxyStr string) *TmpInbox {
	return &TmpInbox{client: proxy.CreateClient(proxyStr, 30*time.Second)}
}

// CreateEmail sinh email client-side (alias@tmpinbox.com).
func (t *TmpInbox) CreateEmail(_ context.Context) (string, error) {
	t.email = realisticEmail("tmpinbox.com")
	return t.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (t *TmpInbox) GetEmail() string { return t.email }

// Close no-op.
func (t *TmpInbox) Close() {}

// WaitForCode poll OTP từ tmpinbox.com inbox.
func (t *TmpInbox) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tmpinbox: chưa tạo email")
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
	return "", fmt.Errorf("tmpinbox: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TmpInbox) pollOnce(ctx context.Context) (string, error) {
	inboxURL := tmpInboxBaseURL + "/inbox/" + t.email + "/messages"
	req, _ := http.NewRequestWithContext(ctx, "GET", inboxURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	matches := tmpInboxMsgIDRegex.FindAllSubmatch(body, -1)
	for _, m := range matches {
		content, _ := t.getMessage(ctx, string(m[1]))
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

func (t *TmpInbox) getMessage(ctx context.Context, msgID string) (string, error) {
	msgURL := tmpInboxBaseURL + "/inbox/" + t.email + "/message/" + msgID
	req, _ := http.NewRequestWithContext(ctx, "GET", msgURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	if m := tmpInboxBodyRegex.FindSubmatch(body); len(m) > 1 {
		return string(m[1]), nil
	}
	if m := tmpInboxHTMLRegex.FindSubmatch(body); len(m) > 1 {
		return string(m[1]), nil
	}
	return string(body), nil
}
