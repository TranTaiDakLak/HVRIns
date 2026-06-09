// inboxes.go — Inboxes.com service (client-side email, REST inbox API)
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const inboxesBaseURL = "https://inboxes.com"

// Inboxes implements email.Service cho inboxes.com
// Email được sinh client-side tại domain trbvm.com mà inboxes.com monitor.
type Inboxes struct {
	client *http.Client
	email  string
}

// NewInboxes tạo Inboxes service.
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewInboxes(proxyStr string) *Inboxes {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &Inboxes{client: client}
}

// CreateEmail sinh địa chỉ email client-side (domain trbvm.com).
func (i *Inboxes) CreateEmail(_ context.Context) (string, error) {
	i.email = realisticEmail("trbvm.com")
	return i.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (i *Inboxes) GetEmail() string { return i.email }

// Close cleanup resources.
func (i *Inboxes) Close() {}

// WaitForCode poll OTP từ inboxes.com inbox.
func (i *Inboxes) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if i.email == "" {
		return "", fmt.Errorf("inboxes: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := i.pollOnce(ctx)
		if err == nil && code != "" {
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

	return "", fmt.Errorf("inboxes: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy danh sách mail và extract code.
func (i *Inboxes) pollOnce(ctx context.Context) (string, error) {
	inboxURL := inboxesBaseURL + "/api/v2/inbox/" + url.PathEscape(i.email)
	req, err := http.NewRequestWithContext(ctx, "GET", inboxURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := i.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)

	var inbox struct {
		Msgs []struct {
			UID string `json:"uid"`
		} `json:"msgs"`
	}
	if err := json.Unmarshal(body, &inbox); err != nil {
		return "", fmt.Errorf("inboxes inbox parse: %w", err)
	}

	for _, msg := range inbox.Msgs {
		content, err := i.getMessage(ctx, msg.UID)
		if err != nil {
			continue
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// getMessage lấy nội dung email theo uid.
func (i *Inboxes) getMessage(ctx context.Context, uid string) (string, error) {
	msgURL := inboxesBaseURL + "/api/v2/message/" + url.PathEscape(uid)
	req, err := http.NewRequestWithContext(ctx, "GET", msgURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := i.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 128*1024)

	var msg struct {
		HTML string `json:"html"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &msg); err != nil {
		return "", err
	}

	if msg.HTML != "" {
		return msg.HTML, nil
	}
	return msg.Text, nil
}
