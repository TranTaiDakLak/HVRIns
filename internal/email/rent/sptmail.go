// sptmail.go — SPTMail.com email service
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

const sptMailBaseURL = "https://api.sptmail.com"

// SPTMail implements email.Service cho api.sptmail.com
type SPTMail struct {
	apiKey      string
	serviceCode string
	client      *http.Client
	onStatus    func(string)

	// Populated after CreateEmail
	emailAddr string
}

// NewSPTMail tạo SPTMail service.
// apiKey: API key cho sptmail.com.
// serviceCode: mã dịch vụ OTP (otpServiceCode).
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewSPTMail(apiKey, serviceCode, proxyStr string) *SPTMail {
	return &SPTMail{
		apiKey:      apiKey,
		serviceCode: serviceCode,
		client:      proxy.CreateClient(proxyStr, 30*time.Second),
	}
}

// SetOnStatus gán callback nhận thông báo trạng thái.
func (s *SPTMail) SetOnStatus(fn func(string)) { s.onStatus = fn }

// GetEmail trả về địa chỉ email đã thuê.
func (s *SPTMail) GetEmail() string { return s.emailAddr }

// Close no-op.
func (s *SPTMail) Close() {}

// notify gửi trạng thái lên callback UI.
func (s *SPTMail) notify(msg string) {
	if s.onStatus != nil {
		s.onStatus(msg)
	}
}

// CreateEmail thuê email từ sptmail.com.
func (s *SPTMail) CreateEmail(ctx context.Context) (string, error) {
	rentURL := fmt.Sprintf("%s/api/otp-services/mail-otp-rental?apiKey=%s&otpServiceCode=%s",
		sptMailBaseURL,
		url.QueryEscape(s.apiKey),
		url.QueryEscape(s.serviceCode))

	req, err := http.NewRequestWithContext(ctx, "GET", rentURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sptmail rent: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	var result struct {
		Gmail string `json:"gmail"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Gmail == "" {
		return "", fmt.Errorf("sptmail rent: unexpected response: %s", string(body))
	}

	s.emailAddr = result.Gmail
	s.notify(fmt.Sprintf("[SPTMail] Thuê thành công: %s", s.emailAddr))
	return s.emailAddr, nil
}

// WaitForCode poll OTP từ sptmail lookup endpoint.
func (s *SPTMail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 20
	}
	if intervalMs == 0 {
		intervalMs = 3000
	}
	if s.emailAddr == "" {
		return "", fmt.Errorf("sptmail: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := s.lookupOTP(ctx)
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

	return "", fmt.Errorf("sptmail: không nhận được OTP sau %d lần thử", maxRetry)
}

// lookupOTP gọi /api/otp-services/mail-otp-lookup một lần.
func (s *SPTMail) lookupOTP(ctx context.Context) (string, error) {
	lookupURL := fmt.Sprintf("%s/api/otp-services/mail-otp-lookup?apiKey=%s&otpServiceCode=%s&gmail=%s",
		sptMailBaseURL,
		url.QueryEscape(s.apiKey),
		url.QueryEscape(s.serviceCode),
		url.QueryEscape(s.emailAddr))

	req, err := http.NewRequestWithContext(ctx, "GET", lookupURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))

	var result struct {
		OTP    string `json:"otp"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("sptmail lookup parse: %w — body: %.200s", err, body)
	}

	if result.Status == "SUCCESS" && result.OTP != "" {
		return result.OTP, nil
	}
	return "", nil
}
