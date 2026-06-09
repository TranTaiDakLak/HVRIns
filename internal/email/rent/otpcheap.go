// otpcheap.go — OTP.cheap email service (api.otp.cheap)
// Thuê Gmail qua neworder, poll email và OTP qua getorder.
package rent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"HVRIns/internal/proxy"
)

const otpCheapBaseURL = "https://api.otp.cheap"

// OtpCheap implements email.Service cho api.otp.cheap
type OtpCheap struct {
	apiKey    string
	serviceID string
	client    *http.Client
	onStatus  func(string)

	// Populated after CreateEmail
	emailAddr string
	quid      string
}

// NewOtpCheap tạo OtpCheap service.
// apiKey: API key cho otp.cheap.
// serviceID: service_id (ví dụ: "8" cho Facebook).
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewOtpCheap(apiKey, serviceID, proxyStr string) *OtpCheap {
	if serviceID == "" {
		serviceID = "8" // Facebook
	}
	return &OtpCheap{
		apiKey:    apiKey,
		serviceID: serviceID,
		client:    proxy.CreateClient(proxyStr, 30*time.Second),
	}
}

// SetOnStatus gán callback nhận thông báo trạng thái.
func (o *OtpCheap) SetOnStatus(fn func(string)) { o.onStatus = fn }

// GetEmail trả về địa chỉ email đã thuê.
func (o *OtpCheap) GetEmail() string { return o.emailAddr }

// Close no-op.
func (o *OtpCheap) Close() {}

// notify gửi trạng thái lên callback UI.
func (o *OtpCheap) notify(msg string) {
	if o.onStatus != nil {
		o.onStatus(msg)
	}
}

// CreateEmail tạo order và lấy địa chỉ email.
func (o *OtpCheap) CreateEmail(ctx context.Context) (string, error) {
	// Bước 1: Tạo order
	newURL := fmt.Sprintf("%s/neworder?api_key=%s&service_id=%s&priority=cheap",
		otpCheapBaseURL,
		url.QueryEscape(o.apiKey),
		url.QueryEscape(o.serviceID))

	req, err := http.NewRequestWithContext(ctx, "GET", newURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("otpcheap neworder: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))

	var orderResult struct {
		Quid string `json:"quid"`
	}
	if err := json.Unmarshal(body, &orderResult); err != nil || orderResult.Quid == "" {
		return "", fmt.Errorf("otpcheap neworder: unexpected response: %s", string(body))
	}
	o.quid = orderResult.Quid

	// Bước 2: Poll getorder để lấy email (timeout ~30s)
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		email, err := o.getOrderEmail(ctx)
		if err == nil && email != "" {
			o.emailAddr = email
			o.notify(fmt.Sprintf("[OtpCheap] Thuê thành công: %s", email))
			return email, nil
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}

	return "", fmt.Errorf("otpcheap: không nhận được email sau 30s")
}

// getOrderEmail gọi GET /getorder và trả về email nếu đã có.
func (o *OtpCheap) getOrderEmail(ctx context.Context) (string, error) {
	getURL := fmt.Sprintf("%s/getorder?api_key=%s&quid=%s",
		otpCheapBaseURL,
		url.QueryEscape(o.apiKey),
		url.QueryEscape(o.quid))

	req, err := http.NewRequestWithContext(ctx, "GET", getURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))

	var result struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("otpcheap getorder parse: %w", err)
	}
	return result.Email, nil
}

// WaitForCode poll OTP từ otp.cheap /getorder endpoint.
func (o *OtpCheap) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 20
	}
	if intervalMs == 0 {
		intervalMs = 3000
	}
	if o.quid == "" {
		return "", fmt.Errorf("otpcheap: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := o.lookupCode(ctx)
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

	return "", fmt.Errorf("otpcheap: không nhận được OTP sau %d lần thử", maxRetry)
}

// lookupCode gọi GET /getorder một lần để lấy code.
func (o *OtpCheap) lookupCode(ctx context.Context) (string, error) {
	getURL := fmt.Sprintf("%s/getorder?api_key=%s&quid=%s",
		otpCheapBaseURL,
		url.QueryEscape(o.apiKey),
		url.QueryEscape(o.quid))

	req, err := http.NewRequestWithContext(ctx, "GET", getURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))

	var result struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("otpcheap getorder parse: %w", err)
	}
	return result.Code, nil
}
