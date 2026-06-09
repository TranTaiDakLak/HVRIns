// shopgmail9999.go — ShopGmail9999 email service (shopgmail9999.com)
// Mua Gmail qua createorder, đọc OTP qua CheckOtp2 endpoint.
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

const shopGmail9999BaseURL = "https://shopgmail9999.com"

// ShopGmail9999 implements email.Service cho shopgmail9999.com
type ShopGmail9999 struct {
	apiKey   string
	service  string
	client   *http.Client
	onStatus func(string)

	// Populated after CreateEmail
	emailAddr string
	orderID   string
}

// NewShopGmail9999 tạo ShopGmail9999 service.
// apiKey: API key cho shopgmail9999.com.
// service: tên dịch vụ (mặc định "facebook").
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewShopGmail9999(apiKey, service, proxyStr string) *ShopGmail9999 {
	if service == "" {
		service = "facebook"
	}
	return &ShopGmail9999{
		apiKey:  apiKey,
		service: service,
		client:  proxy.CreateClient(proxyStr, 30*time.Second),
	}
}

// SetOnStatus gán callback nhận thông báo trạng thái.
func (s *ShopGmail9999) SetOnStatus(fn func(string)) { s.onStatus = fn }

// GetEmail trả về địa chỉ email đã mua.
func (s *ShopGmail9999) GetEmail() string { return s.emailAddr }

// Close no-op.
func (s *ShopGmail9999) Close() {}

// notify gửi trạng thái lên callback UI.
func (s *ShopGmail9999) notify(msg string) {
	if s.onStatus != nil {
		s.onStatus(msg)
	}
}

// CreateEmail mua 1 email từ shopgmail9999.com.
func (s *ShopGmail9999) CreateEmail(ctx context.Context) (string, error) {
	buyURL := fmt.Sprintf("%s/createorder?apikey=%s&service=%s",
		shopGmail9999BaseURL,
		url.QueryEscape(s.apiKey),
		url.QueryEscape(s.service))

	req, err := http.NewRequestWithContext(ctx, "GET", buyURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("shopgmail9999 createorder: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	var result struct {
		Data struct {
			Email   string `json:"email"`
			OrderID string `json:"orderid"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("shopgmail9999 createorder parse: %w — body: %.200s", err, body)
	}
	if result.Data.Email == "" || result.Data.OrderID == "" {
		return "", fmt.Errorf("shopgmail9999 createorder: empty response — body: %.200s", body)
	}

	s.emailAddr = result.Data.Email
	s.orderID = result.Data.OrderID
	s.notify(fmt.Sprintf("[ShopGmail9999] Mua thành công: %s", s.emailAddr))
	return s.emailAddr, nil
}

// WaitForCode poll OTP từ shopgmail9999 /CheckOtp2 endpoint.
func (s *ShopGmail9999) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 20
	}
	if intervalMs == 0 {
		intervalMs = 3000
	}
	if s.orderID == "" {
		return "", fmt.Errorf("shopgmail9999: chưa tạo email")
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

	return "", fmt.Errorf("shopgmail9999: không nhận được OTP sau %d lần thử", maxRetry)
}

// lookupOTP gọi GET /CheckOtp2 một lần.
func (s *ShopGmail9999) lookupOTP(ctx context.Context) (string, error) {
	checkURL := fmt.Sprintf("%s/CheckOtp2?apikey=%s&orderid=%s&getbody=true",
		shopGmail9999BaseURL,
		url.QueryEscape(s.apiKey),
		url.QueryEscape(s.orderID))

	req, err := http.NewRequestWithContext(ctx, "GET", checkURL, nil)
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
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	var result struct {
		Data struct {
			OTP  string `json:"otp"`
			Body string `json:"body"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("shopgmail9999 checkotp parse: %w — body: %.200s", err, body)
	}

	if result.Data.OTP != "" {
		return result.Data.OTP, nil
	}
	// Fallback: extract từ body email
	if result.Data.Body != "" {
		return result.Data.Body, nil // caller sẽ dùng ExtractCode nếu cần
	}
	return "", nil
}
