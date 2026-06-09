// otpcodesms.go — OtpCodesSms.site service (phone number OTP rental)
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

const otpCodesSmsBaseURL = "https://otpcodesms.site/api"

// OtpCodesSms implements email.Service cho otpcodesms.site
type OtpCodesSms struct {
	apiKey    string
	serviceID string
	client    *http.Client
	onStatus  func(string)

	requestID string
	number    string
}

// NewOtpCodesSms tạo OtpCodesSms service.
// serviceID: ID dịch vụ tại otpcodesms.site.
func NewOtpCodesSms(apiKey, serviceID, proxyStr string) *OtpCodesSms {
	return &OtpCodesSms{
		apiKey:    apiKey,
		serviceID: serviceID,
		client:    proxy.CreateClient(proxyStr, 30*time.Second),
	}
}

// SetOnStatus gán callback nhận thông báo trạng thái.
func (o *OtpCodesSms) SetOnStatus(fn func(string)) { o.onStatus = fn }

// GetEmail trả về số điện thoại đã thuê (dùng như email trong interface).
func (o *OtpCodesSms) GetEmail() string { return o.number }

// Close no-op.
func (o *OtpCodesSms) Close() {}

func (o *OtpCodesSms) notify(msg string) {
	if o.onStatus != nil {
		o.onStatus(msg)
	}
}

// CreateEmail thuê 1 số điện thoại từ otpcodesms.site.
func (o *OtpCodesSms) CreateEmail(ctx context.Context) (string, error) {
	getURL := fmt.Sprintf("%s?key=%s&action=get_number&id=%s",
		otpCodesSmsBaseURL, url.QueryEscape(o.apiKey), url.QueryEscape(o.serviceID))

	req, err := http.NewRequestWithContext(ctx, "GET", getURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("otpcodesms get_number: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))

	var result struct {
		Number    string `json:"number"`
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("otpcodesms get_number parse: %w — body: %.200s", err, body)
	}
	if result.Number == "" || result.RequestID == "" {
		return "", fmt.Errorf("otpcodesms get_number: empty response — body: %.200s", body)
	}

	o.number = result.Number
	o.requestID = result.RequestID
	o.notify(fmt.Sprintf("[OtpCodesSms] Thuê số: %s", o.number))
	return o.number, nil
}

// WaitForCode poll OTP từ otpcodesms.site /get_code endpoint.
func (o *OtpCodesSms) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 20
	}
	if intervalMs == 0 {
		intervalMs = 3000
	}
	if o.requestID == "" {
		return "", fmt.Errorf("otpcodesms: chưa thuê số")
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

	return "", fmt.Errorf("otpcodesms: không nhận được OTP sau %d lần thử", maxRetry)
}

// lookupCode gọi GET /get_code một lần.
func (o *OtpCodesSms) lookupCode(ctx context.Context) (string, error) {
	checkURL := fmt.Sprintf("%s?key=%s&action=get_code&id=%s",
		otpCodesSmsBaseURL, url.QueryEscape(o.apiKey), url.QueryEscape(o.requestID))

	req, err := http.NewRequestWithContext(ctx, "GET", checkURL, nil)
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
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))

	var result struct {
		OTPCode string `json:"otp_code"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("otpcodesms get_code parse: %w — body: %.200s", err, body)
	}

	code := result.OTPCode
	if code == "" || code == "is_coming" || code == "timeout" {
		return "", nil
	}
	return code, nil
}
