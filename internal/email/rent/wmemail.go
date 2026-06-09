// wmemail.go — Wmemail.com email service (mua Hotmail qua www.wmemail.com)
// Port từ C# WmemailAPI. Flow:
//   1. Rent: POST www.wmemail.com/user/api/api/trade (form-urlencoded) → parse JSON
//      data.secret dạng "email----password----clientid----refreshtoken".
//   2. Lookup OTP: POST tools.dongvanfb.net/api/get_code_oauth2 (JSON, có field pass + type).
package rent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"HVRIns/internal/proxy"
)

const wmemailBaseURL = "https://www.wmemail.com"

// wmemailBuyResp — response từ wmemail.com/user/api/api/trade
type wmemailBuyResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TradeNo string `json:"tradeNo"`
		Secret  string `json:"secret"`
	} `json:"data"`
}

// Wmemail implements email.Service — mua Hotmail/Outlook từ wmemail.com
type Wmemail struct {
	apiKey      string
	commodity   string // commodity_id (service id) của gói mail
	client      *http.Client
	onStatus    func(string)
	otpPriority string // "dongvan" (default) | "unlimit" — chọn primary OTP reader

	// Populated after CreateEmail
	emailAddr    string
	password     string
	refreshToken string
	clientID     string
}

// NewWmemail tạo Wmemail service.
// apiKey = token trong C#, commodity = commodity_id (service id của gói hotmail).
func NewWmemail(apiKey, commodity, proxyStr string) *Wmemail {
	return &Wmemail{
		apiKey:    apiKey,
		commodity: commodity,
		client:    proxy.CreateClient(proxyStr, 30*time.Second),
	}
}

// SetOnStatus gán callback nhận thông báo trạng thái.
func (w *Wmemail) SetOnStatus(fn func(string)) { w.onStatus = fn }

// SetOTPPriority chọn nguồn đọc OTP ưu tiên ("dongvan" | "unlimit").
func (w *Wmemail) SetOTPPriority(p string) { w.otpPriority = p }

// GetEmail trả về địa chỉ email đã mua.
func (w *Wmemail) GetEmail() string { return w.emailAddr }

// Close no-op.
func (w *Wmemail) Close() {}

func (w *Wmemail) notify(msg string) {
	if w.onStatus != nil {
		w.onStatus(msg)
	}
}

// CreateEmail mua 1 email từ wmemail.com, retry khi hết hàng.
func (w *Wmemail) CreateEmail(ctx context.Context) (string, error) {
	const retryDelay = 5 * time.Second
	for attempt := 1; ; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		cred, err := w.buyOne(ctx)
		if err == nil {
			w.emailAddr = cred.Email
			w.password = cred.Password
			w.refreshToken = cred.RefreshToken
			w.clientID = cred.ClientId
			w.notify(fmt.Sprintf("[Wmemail] Mua thành công: %s", w.emailAddr))
			return w.emailAddr, nil
		}
		if isOutOfStock(err) {
			w.notify(fmt.Sprintf("[Wmemail] Hết hàng (commodity=%s), thử lại sau 5s... (lần %d)", w.commodity, attempt))
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

// buyOne gọi POST /user/api/api/trade form-urlencoded để mua 1 account.
// Response JSON: {"code":200,"data":{"tradeNo":"...","secret":"email----pass----client_id----refresh_token"}}
func (w *Wmemail) buyOne(ctx context.Context) (EmailCred, error) {
	form := url.Values{}
	form.Set("commodity_id", w.commodity)
	form.Set("num", "1")
	form.Set("token", w.apiKey)

	req, err := http.NewRequestWithContext(ctx, "POST",
		wmemailBaseURL+"/user/api/api/trade",
		strings.NewReader(form.Encode()))
	if err != nil {
		return EmailCred{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := w.client.Do(req)
	if err != nil {
		return EmailCred{}, fmt.Errorf("wmemail buy: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	var result wmemailBuyResp
	if err := json.Unmarshal(body, &result); err != nil {
		return EmailCred{}, fmt.Errorf("wmemail buy parse: %w — body: %.200s", err, body)
	}
	if result.Code != 200 || result.Data.Secret == "" {
		bodyLower := strings.ToLower(string(body))
		if strings.Contains(bodyLower, "库存") || strings.Contains(bodyLower, "out of stock") ||
			strings.Contains(bodyLower, "hết hàng") || strings.Contains(bodyLower, "insufficient") ||
			strings.Contains(bodyLower, "sold out") {
			return EmailCred{}, errOutOfStock
		}
		return EmailCred{}, fmt.Errorf("wmemail buy failed [code=%d]: %s — body: %.200s",
			result.Code, result.Message, body)
	}

	// Secret format (theo C#): email----password----clientId----refreshToken
	parts := strings.Split(result.Data.Secret, "----")
	if len(parts) < 1 || parts[0] == "" {
		return EmailCred{}, fmt.Errorf("wmemail buy: invalid secret format: %s", result.Data.Secret)
	}
	cred := EmailCred{Email: parts[0]}
	if len(parts) >= 2 {
		cred.Password = parts[1]
	}
	if len(parts) >= 4 {
		cred.ClientId = parts[2]
		cred.RefreshToken = parts[3]
	}
	return cred, nil
}

// WaitForCode poll OTP qua tools.dongvanfb.net/api/get_code_oauth2.
func (w *Wmemail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
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

		code, err := w.fetchOTP(ctx)
		if err == nil && code != "" {
			return code, nil
		}
		if err != nil {
			w.notify(fmt.Sprintf("[WmEmail][Poll %d/%d] Lỗi: %v", attempt+1, maxRetry, err))
		} else {
			w.notify(fmt.Sprintf("[WmEmail][Poll %d/%d] Chưa có OTP...", attempt+1, maxRetry))
		}

		if attempt < maxRetry-1 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			}
		}
	}
	return "", fmt.Errorf("wmemail: no OTP code received after %d attempts", maxRetry)
}

// fetchOTP gọi tools.dongvanfb.net/api/get_code_oauth2.
func (w *Wmemail) fetchOTP(ctx context.Context) (string, error) {
	return ReadOTPWithPriority(ctx, w.otpPriority, w.emailAddr, w.password, w.refreshToken, w.clientID, w.client, w.notify)
}
