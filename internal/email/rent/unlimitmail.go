// unlimitmail.go — UnlimitMail.com email service
// Đọc OTP qua tools.dongvanfb.net/api/get_messages_oauth2
package rent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"HVRIns/internal/proxy"
)

const unlimitMailBaseURL = "https://unlimitmail.com"

// unlimitMailBuyResp — response từ unlimitmail.com/api/buyHotMailUd
type unlimitMailBuyResp struct {
	Data []struct {
		Email        string `json:"email"`
		Password     string `json:"password"`
		RefreshToken string `json:"refresh_token"`
		ClientID     string `json:"client_id"`
	} `json:"data"`
}

// UnlimitMail implements email.Service
type UnlimitMail struct {
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

// NewUnlimitMail tạo UnlimitMail service.
func NewUnlimitMail(apiKey, productID, proxyStr string) *UnlimitMail {
	return &UnlimitMail{
		apiKey:    apiKey,
		productID: productID,
		client:    proxy.CreateClient(proxyStr, 30*time.Second),
	}
}

// SetOnStatus gán callback nhận thông báo trạng thái.
func (u *UnlimitMail) SetOnStatus(fn func(string)) { u.onStatus = fn }

// SetOTPPriority chọn nguồn đọc OTP ưu tiên ("dongvan" | "unlimit").
func (u *UnlimitMail) SetOTPPriority(p string) { u.otpPriority = p }

// GetEmail trả về địa chỉ email đã mua.
func (u *UnlimitMail) GetEmail() string { return u.emailAddr }

// Close no-op.
func (u *UnlimitMail) Close() {}

// notify gửi trạng thái lên callback UI.
func (u *UnlimitMail) notify(msg string) {
	if u.onStatus != nil {
		u.onStatus(msg)
	}
}

// CreateEmail mua 1 email từ unlimitmail.com.
func (u *UnlimitMail) CreateEmail(ctx context.Context) (string, error) {
	const retryDelay = 5 * time.Second
	for attempt := 1; ; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		cred, err := u.buyOne(ctx)
		if err == nil {
			u.emailAddr = cred.Email
			u.password = cred.Password
			u.refreshToken = cred.RefreshToken
			u.clientID = cred.ClientId
			u.notify(fmt.Sprintf("[UnlimitMail] Mua thành công: %s", u.emailAddr))
			return u.emailAddr, nil
		}
		if isOutOfStock(err) {
			u.notify(fmt.Sprintf("[UnlimitMail] Hết hàng (product_id=%s), thử lại sau 5s... (lần %d)", u.productID, attempt))
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

// buyOne gọi POST /api/buyHotMailUd để mua 1 account.
func (u *UnlimitMail) buyOne(ctx context.Context) (EmailCred, error) {
	payload := map[string]interface{}{
		"quantity":   1,
		"token":      u.apiKey,
		"product_id": u.productID,
		"type":       "email_pass_refresh_client",
	}
	bodyBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST",
		unlimitMailBaseURL+"/api/buyHotMailUd",
		bytes.NewReader(bodyBytes))
	if err != nil {
		return EmailCred{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := u.client.Do(req)
	if err != nil {
		return EmailCred{}, fmt.Errorf("unlimitmail buy: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	var result unlimitMailBuyResp
	if err := json.Unmarshal(body, &result); err != nil {
		return EmailCred{}, fmt.Errorf("unlimitmail buy parse: %w — body: %.200s", err, body)
	}

	if len(result.Data) == 0 {
		bodyLower := strings.ToLower(string(body))
		if strings.Contains(bodyLower, "hết hàng") || strings.Contains(bodyLower, "out of stock") ||
			strings.Contains(bodyLower, "insufficient") {
			return EmailCred{}, errOutOfStock
		}
		return EmailCred{}, fmt.Errorf("unlimitmail buy: empty data — body: %.200s", body)
	}

	item := result.Data[0]
	if item.Email == "" {
		return EmailCred{}, fmt.Errorf("unlimitmail buy: missing email in response")
	}

	return EmailCred{
		Email:        item.Email,
		Password:     item.Password,
		RefreshToken: item.RefreshToken,
		ClientId:     item.ClientID,
	}, nil
}

// WaitForCode poll OTP qua tools.dongvanfb.net/api/get_code_oauth2.
func (u *UnlimitMail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
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

		code, err := u.fetchOTP(ctx)
		if err == nil && code != "" {
			return code, nil
		}
		if err != nil {
			u.notify(fmt.Sprintf("[UnlimitMail][Poll %d/%d] Lỗi: %v", attempt+1, maxRetry, err))
		} else {
			u.notify(fmt.Sprintf("[UnlimitMail][Poll %d/%d] Chưa có OTP...", attempt+1, maxRetry))
		}

		if attempt < maxRetry-1 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			}
		}
	}

	return "", fmt.Errorf("unlimitmail: no OTP code received after %d attempts", maxRetry)
}

// fetchOTP đọc OTP qua priority helper (dongvan primary, unlimit fallback — hoặc đảo lại).
func (u *UnlimitMail) fetchOTP(ctx context.Context) (string, error) {
	return ReadOTPWithPriority(ctx, u.otpPriority, u.emailAddr, u.password, u.refreshToken, u.clientID, u.client, u.notify)
}
