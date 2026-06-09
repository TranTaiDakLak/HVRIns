// rentgmail.go — RentGmail.online service (rent Gmail account, poll OTP)
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

const rentGmailBaseURL = "https://rentgmail.online/prod"

// RentGmail implements email.Service cho rentgmail.online
type RentGmail struct {
	apiKey   string
	platform string
	client   *http.Client
	onStatus func(string)

	emailAddr string
	orderID   string
}

// NewRentGmail tạo RentGmail service.
// platform: tên nền tảng (vd: "facebook").
func NewRentGmail(apiKey, platform, proxyStr string) *RentGmail {
	if platform == "" {
		platform = "facebook"
	}
	return &RentGmail{
		apiKey:   apiKey,
		platform: platform,
		client:   proxy.CreateClient(proxyStr, 30*time.Second),
	}
}

// SetOnStatus gán callback nhận thông báo trạng thái.
func (r *RentGmail) SetOnStatus(fn func(string)) { r.onStatus = fn }

// GetEmail trả về địa chỉ email đã thuê.
func (r *RentGmail) GetEmail() string { return r.emailAddr }

// Close no-op.
func (r *RentGmail) Close() {}

func (r *RentGmail) notify(msg string) {
	if r.onStatus != nil {
		r.onStatus(msg)
	}
}

// CreateEmail thuê 1 Gmail account từ rentgmail.online.
func (r *RentGmail) CreateEmail(ctx context.Context) (string, error) {
	rentURL := fmt.Sprintf("%s/mail/order/rentMail?mailTypeCode=gmail&platform=%s&token=%s",
		rentGmailBaseURL, url.QueryEscape(r.platform), url.QueryEscape(r.apiKey))

	req, err := http.NewRequestWithContext(ctx, "GET", rentURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("rentgmail rentMail: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	var result struct {
		Code int `json:"code"`
		Data struct {
			Email   string `json:"email"`
			OrderID string `json:"orderId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("rentgmail rentMail parse: %w — body: %.200s", err, body)
	}
	if result.Code != 200 || result.Data.Email == "" {
		return "", fmt.Errorf("rentgmail rentMail: unexpected response — body: %.200s", body)
	}

	r.emailAddr = result.Data.Email
	r.orderID = result.Data.OrderID
	r.notify(fmt.Sprintf("[RentGmail] Mua thành công: %s", r.emailAddr))
	return r.emailAddr, nil
}

// WaitForCode poll OTP từ rentgmail.online /mailOtp endpoint.
func (r *RentGmail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 20
	}
	if intervalMs == 0 {
		intervalMs = 3000
	}
	if r.orderID == "" {
		return "", fmt.Errorf("rentgmail: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := r.lookupOTP(ctx)
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

	return "", fmt.Errorf("rentgmail: không nhận được OTP sau %d lần thử", maxRetry)
}

// lookupOTP gọi GET /mailOtp một lần.
func (r *RentGmail) lookupOTP(ctx context.Context) (string, error) {
	checkURL := fmt.Sprintf("%s/mail/order/mailOtp?orderId=%s&token=%s",
		rentGmailBaseURL, url.QueryEscape(r.orderID), url.QueryEscape(r.apiKey))

	req, err := http.NewRequestWithContext(ctx, "GET", checkURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	var result struct {
		Code int `json:"code"`
		Data struct {
			OTP string `json:"otp"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("rentgmail mailOtp parse: %w — body: %.200s", err, body)
	}

	if result.Code == 200 && result.Data.OTP != "" {
		return result.Data.OTP, nil
	}
	return "", nil
}
