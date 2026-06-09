// onesecemail.go — 1secemail.com service (CSRF, flow giống TempMailTo)
// Port từ C# OneSecEmailAPI. Khác với OneSecMailAPI (1secmail.com) — đây là 1secemail.com.
package temp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"time"

	"HVRIns/internal/proxy"
)

const oneSecEmailBaseURL = "https://1secemail.com"

// OneSecEmail implements email.Service cho 1secemail.com.
type OneSecEmail struct {
	client    *http.Client
	email     string
	csrfToken string
}

// NewOneSecEmail tạo OneSecEmail service.
func NewOneSecEmail(proxyStr string) *OneSecEmail {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	c.Jar = jar
	return &OneSecEmail{client: c}
}

// CreateEmail khởi tạo session qua CSRF.
func (o *OneSecEmail) CreateEmail(ctx context.Context) (string, error) {
	email, err := csrfInitSession(ctx, o.client, oneSecEmailBaseURL, &o.csrfToken)
	if err != nil {
		return "", fmt.Errorf("1secemail: %w", err)
	}
	o.email = email
	return o.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (o *OneSecEmail) GetEmail() string { return o.email }

// Close no-op.
func (o *OneSecEmail) Close() {}

// WaitForCode poll OTP qua shared CSRF helper.
func (o *OneSecEmail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if o.email == "" {
		return "", fmt.Errorf("1secemail: chưa tạo email")
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if code, _ := csrfPollInbox(ctx, o.client, oneSecEmailBaseURL, o.csrfToken); code != "" {
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
	return "", fmt.Errorf("1secemail: không nhận được OTP sau %d lần thử", maxRetry)
}
