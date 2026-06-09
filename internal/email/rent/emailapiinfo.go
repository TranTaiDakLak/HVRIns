// emailapiinfo.go — EmailApi.Info (Gmail500) email service
// Mua Gmail qua api.emailapi.info, đọc OTP qua /code endpoint.
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

const emailAPIInfoBaseURL = "https://api.emailapi.info"

// EmailAPIInfo implements email.Service cho api.emailapi.info (gmail500.com)
type EmailAPIInfo struct {
	apiKey      string
	productCode string
	client      *http.Client
	onStatus    func(string)

	// Populated after CreateEmail
	emailAddr string
	orderNo   string
}

// NewEmailAPIInfo tạo EmailAPIInfo service.
// apiKey: API key cho emailapi.info.
// productCode: mã sản phẩm email (ví dụ: "gmail").
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewEmailAPIInfo(apiKey, productCode, proxyStr string) *EmailAPIInfo {
	if productCode == "" {
		productCode = "gmail"
	}
	return &EmailAPIInfo{
		apiKey:      apiKey,
		productCode: productCode,
		client:      proxy.CreateClient(proxyStr, 30*time.Second),
	}
}

// SetOnStatus gán callback nhận thông báo trạng thái.
func (e *EmailAPIInfo) SetOnStatus(fn func(string)) { e.onStatus = fn }

// GetEmail trả về địa chỉ email đã mua.
func (e *EmailAPIInfo) GetEmail() string { return e.emailAddr }

// Close no-op.
func (e *EmailAPIInfo) Close() {}

// notify gửi trạng thái lên callback UI.
func (e *EmailAPIInfo) notify(msg string) {
	if e.onStatus != nil {
		e.onStatus(msg)
	}
}

// CreateEmail mua 1 email từ emailapi.info.
func (e *EmailAPIInfo) CreateEmail(ctx context.Context) (string, error) {
	buyURL := fmt.Sprintf("%s/openapi/v2/mail/code/buy?apiKey=%s&productCode=%s",
		emailAPIInfoBaseURL,
		url.QueryEscape(e.apiKey),
		url.QueryEscape(e.productCode))

	req, err := http.NewRequestWithContext(ctx, "GET", buyURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("emailapiinfo buy: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	var result struct {
		Data struct {
			OrderDetail struct {
				Address string `json:"address"`
			} `json:"orderDetail"`
			OrderNo string `json:"orderNo"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("emailapiinfo buy parse: %w — body: %.200s", err, body)
	}
	if result.Data.OrderDetail.Address == "" || result.Data.OrderNo == "" {
		return "", fmt.Errorf("emailapiinfo buy: empty response — body: %.200s", body)
	}

	e.emailAddr = result.Data.OrderDetail.Address
	e.orderNo = result.Data.OrderNo
	e.notify(fmt.Sprintf("[EmailAPIInfo] Mua thành công: %s", e.emailAddr))
	return e.emailAddr, nil
}

// WaitForCode poll OTP từ emailapi.info /code endpoint.
func (e *EmailAPIInfo) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 20
	}
	if intervalMs == 0 {
		intervalMs = 3000
	}
	if e.orderNo == "" {
		return "", fmt.Errorf("emailapiinfo: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := e.lookupCode(ctx)
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

	return "", fmt.Errorf("emailapiinfo: không nhận được OTP sau %d lần thử", maxRetry)
}

// lookupCode gọi GET /code một lần.
func (e *EmailAPIInfo) lookupCode(ctx context.Context) (string, error) {
	codeURL := fmt.Sprintf("%s/code?apiKey=%s&orderNo=%s",
		emailAPIInfoBaseURL,
		url.QueryEscape(e.apiKey),
		url.QueryEscape(e.orderNo))

	req, err := http.NewRequestWithContext(ctx, "GET", codeURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := e.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))

	var result struct {
		Data struct {
			Code string `json:"code"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("emailapiinfo code parse: %w — body: %.200s", err, body)
	}
	return result.Data.Code, nil
}
