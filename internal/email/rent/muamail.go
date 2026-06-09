// muamail.go — MuaMail.Store email service (mua Hotmail qua api.muamail.store)
// Đọc OTP qua tools.dongvanfb.net/api/get_messages_oauth2
package rent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"HVRIns/internal/proxy"
)

const muaMailBaseURL = "https://api.muamail.store"

// muaMailBuyResp — response từ muamail.store/products/buy
type muaMailBuyResp struct {
	Data []string `json:"data"` // format: email|password|client_id|refresh_token
}

// MuaMail implements email.Service
type MuaMail struct {
	apiKey      string
	productID   string
	client      *http.Client
	onStatus    func(string)
	otpPriority string // "dongvan" (default) | "unlimit" — chọn primary OTP reader

	// Populated after CreateEmail
	emailAddr    string
	password     string
	refreshToken string
	clientID     string
}

// NewMuaMail tạo MuaMail service.
func NewMuaMail(apiKey, productID, proxyStr string) *MuaMail {
	return &MuaMail{
		apiKey:    apiKey,
		productID: productID,
		client:    proxy.CreateClient(proxyStr, 30*time.Second),
	}
}

// SetOnStatus gán callback nhận thông báo trạng thái.
func (m *MuaMail) SetOnStatus(fn func(string)) { m.onStatus = fn }

// SetOTPPriority chọn nguồn đọc OTP ưu tiên ("dongvan" | "unlimit").
func (m *MuaMail) SetOTPPriority(p string) { m.otpPriority = p }

// GetEmail trả về địa chỉ email đã mua.
func (m *MuaMail) GetEmail() string { return m.emailAddr }

// Close no-op.
func (m *MuaMail) Close() {}

// notify gửi trạng thái lên callback UI.
func (m *MuaMail) notify(msg string) {
	if m.onStatus != nil {
		m.onStatus(msg)
	}
}

// CreateEmail mua 1 email từ muamail.store.
func (m *MuaMail) CreateEmail(ctx context.Context) (string, error) {
	const retryDelay = 5 * time.Second
	for attempt := 1; ; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		cred, err := m.buyOne(ctx)
		if err == nil {
			m.emailAddr = cred.Email
			m.password = cred.Password
			m.refreshToken = cred.RefreshToken
			m.clientID = cred.ClientId
			m.notify(fmt.Sprintf("[MuaMail] Mua thành công: %s", m.emailAddr))
			return m.emailAddr, nil
		}
		if isOutOfStock(err) {
			m.notify(fmt.Sprintf("[MuaMail] Hết hàng (product_id=%s), thử lại sau 5s... (lần %d)", m.productID, attempt))
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(retryDelay):
			}
			continue
		}
		return "", err
	}
}

// buyOne gọi GET /products/buy/ để mua 1 account.
// MuaMail format: email|password|client_id|refresh_token (client_id trước refresh_token, ngược store1s)
func (m *MuaMail) buyOne(ctx context.Context) (EmailCred, error) {
	buyURL := fmt.Sprintf("%s/products/buy/?api_key=%s&id=%s&quantity=1",
		muaMailBaseURL, m.apiKey, m.productID)

	req, err := http.NewRequestWithContext(ctx, "GET", buyURL, nil)
	if err != nil {
		return EmailCred{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := m.client.Do(req)
	if err != nil {
		return EmailCred{}, fmt.Errorf("muamail buy: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	var result muaMailBuyResp
	if err := json.Unmarshal(body, &result); err != nil {
		return EmailCred{}, fmt.Errorf("muamail buy parse: %w — body: %.200s", err, body)
	}

	if len(result.Data) == 0 {
		// Check for out-of-stock in raw body
		bodyLower := strings.ToLower(string(body))
		if strings.Contains(bodyLower, "hết hàng") || strings.Contains(bodyLower, "out of stock") ||
			strings.Contains(bodyLower, "insufficient") {
			return EmailCred{}, errOutOfStock
		}
		return EmailCred{}, fmt.Errorf("muamail buy: empty data — body: %.200s", body)
	}

	// Parse: email|password|client_id|refresh_token
	parts := strings.Split(result.Data[0], "|")
	if len(parts) < 4 {
		return EmailCred{}, fmt.Errorf("muamail buy: cannot parse line: %q", result.Data[0])
	}

	return EmailCred{
		Email:        strings.TrimSpace(parts[0]),
		Password:     strings.TrimSpace(parts[1]),
		ClientId:     strings.TrimSpace(parts[2]),
		RefreshToken: strings.TrimSpace(parts[3]),
	}, nil
}

// WaitForCode poll OTP qua tools.dongvanfb.net/api/get_code_oauth2.
func (m *MuaMail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 3
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := m.fetchOTP(ctx)
		if err == nil && code != "" {
			return code, nil
		}
		if err != nil {
			m.notify(fmt.Sprintf("[MuaMail][Poll %d/%d] Lỗi: %v", attempt+1, maxRetry, err))
		} else {
			m.notify(fmt.Sprintf("[MuaMail][Poll %d/%d] Chưa có OTP...", attempt+1, maxRetry))
		}

		if attempt < maxRetry-1 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			}
		}
	}

	return "", fmt.Errorf("muamail: no OTP code received after %d attempts", maxRetry)
}

// fetchOTP gọi tools.dongvanfb.net/api/get_messages_oauth2.
func (m *MuaMail) fetchOTP(ctx context.Context) (string, error) {
	return ReadOTPWithPriority(ctx, m.otpPriority, m.emailAddr, m.password, m.refreshToken, m.clientID, m.client, m.notify)
}
