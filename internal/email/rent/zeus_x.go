// zeus_x.go — ZeusX email service (Hotmail/Outlook mua từ api.zeus-x.ru)
// Mua batch email qua Purchase API → đọc OTP qua tools.dongvanfb.net/api/get_code_oauth2
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

// === Purchase API response ===
type zeusXPurchaseResp struct {
	Code    int    `json:"Code"`
	Message string `json:"Message"`
	Data    struct {
		Accounts []struct {
			Email        string `json:"Email"`
			Password     string `json:"Password"`
			RefreshToken string `json:"RefreshToken"`
			ClientId     string `json:"ClientId"`
		} `json:"Accounts"`
	} `json:"Data"`
}

// === dongvanfb read messages response ===
type dvfbMessagesResp struct {
	Email    string `json:"email"`
	Status   bool   `json:"status"`
	Code     string `json:"code"`
	Content  string `json:"content"`
	Messages []struct {
		From    string `json:"from"`
		Subject string `json:"subject"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Date    string `json:"date"`
	} `json:"messages"`
}

// ZeusX implements email.Service — mua Hotmail/Outlook từ zeus-x.ru
type ZeusX struct {
	apiKey      string
	accountCode string
	proxyStr    string
	client      *http.Client
	onStatus    func(string)
	pool        *CredPool // shared pool — nếu set sẽ pull từ pool thay vì mua riêng lẻ
	otpPriority string    // "dongvan" (default) | "unlimit" — chọn primary OTP reader

	// Populated after CreateEmail
	emailAddr    string
	password     string
	refreshToken string
	clientId     string
}

// NewZeusX tạo ZeusX service
func NewZeusX(apiKey, accountCode, proxyStr string) *ZeusX {
	return &ZeusX{
		apiKey:      apiKey,
		accountCode: accountCode,
		proxyStr:    proxyStr,
		client:      proxy.CreateClient(proxyStr, 30*time.Second),
	}
}

// SetPool gán shared pool — CreateEmail sẽ pull từ pool thay vì mua riêng lẻ
func (z *ZeusX) SetPool(p *CredPool) { z.pool = p }

// SetOnStatus gán callback để notify UI
func (z *ZeusX) SetOnStatus(fn func(string)) { z.onStatus = fn }

// SetOTPPriority chọn nguồn đọc OTP ưu tiên ("dongvan" | "unlimit").
func (z *ZeusX) SetOTPPriority(p string) { z.otpPriority = p }

// notify gửi thông báo trạng thái đến callback UI nếu đã được gán.
// msg: nội dung thông báo (ví dụ: "Lấy từ pool: ...", "Hết hàng, thử lại...").
func (z *ZeusX) notify(msg string) {
	if z.onStatus != nil {
		z.onStatus(msg)
	}
}

// CreateEmail: nếu có pool → pull từ pool (batch); không có → mua đơn lẻ
func (z *ZeusX) CreateEmail(ctx context.Context) (string, error) {
	if z.pool != nil {
		cred, err := z.pool.Get(ctx)
		if err != nil {
			return "", err
		}
		z.emailAddr = cred.Email
		z.password = cred.Password
		z.refreshToken = cred.RefreshToken
		z.clientId = cred.ClientId
		z.notify(fmt.Sprintf("[ZeusX] Lấy từ pool: %s", z.emailAddr))
		return z.emailAddr, nil
	}
	// Fallback: mua đơn lẻ (legacy)
	return z.buyLegacy(ctx)
}

// buyLegacy mua 1 cái, retry khi hết hàng (fallback khi không dùng pool)
func (z *ZeusX) buyLegacy(ctx context.Context) (string, error) {
	const retryDelay = 5 * time.Second
	for attempt := 1; ; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		creds, err := z.buyBatch(ctx, 1)
		if err == nil && len(creds) > 0 {
			z.emailAddr = creds[0].Email
			z.password = creds[0].Password
			z.refreshToken = creds[0].RefreshToken
			z.clientId = creds[0].ClientId
			return z.emailAddr, nil
		}
		if err != nil && isOutOfStock(err) {
			z.notify(fmt.Sprintf("[ZeusX] Hết hàng, thử lại sau 5s... (lần %d)", attempt))
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(retryDelay):
			}
			continue
		}
		if err != nil {
			return "", err
		}
	}
}

// NewZeusXPool tạo CredPool dùng ZeusX để mua batch
// batchSize nên = maxThreads để 1 lần mua đủ cho tất cả luồng
func NewZeusXPool(apiKey, accountCode, proxyStr string, batchSize int, notify func(string)) *CredPool {
	z := NewZeusX(apiKey, accountCode, proxyStr)
	z.onStatus = notify

	buyFn := func(ctx context.Context, n int) ([]EmailCred, error) {
		const retryDelay = 5 * time.Second
		for attempt := 1; ; attempt++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			creds, err := z.buyBatch(ctx, n)
			if err == nil {
				return creds, nil
			}
			if isOutOfStock(err) {
				if notify != nil {
					notify(fmt.Sprintf("[ZeusX Pool] Hết hàng, thử lại sau 5s... (lần %d)", attempt))
				}
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(retryDelay):
				}
				continue
			}
			return nil, err
		}
	}

	threshold := poolMax(1, batchSize/3)
	return NewCredPool(batchSize, threshold, buyFn, notify)
}

// buyBatch gọi API mua n accounts cùng lúc
func (z *ZeusX) buyBatch(ctx context.Context, n int) ([]EmailCred, error) {
	endpoint := fmt.Sprintf(
		"https://api.zeus-x.ru/purchase?apikey=%s&accountcode=%s&quantity=%d",
		url.QueryEscape(z.apiKey),
		url.QueryEscape(z.accountCode),
		n,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := z.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zeus-x purchase: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var result zeusXPurchaseResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("zeus-x parse: %w — body: %.200s", err, body)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("zeus-x [%d]: %s", result.Code, result.Message)
	}
	if len(result.Data.Accounts) == 0 {
		return nil, errOutOfStock
	}

	creds := make([]EmailCred, 0, len(result.Data.Accounts))
	for _, acc := range result.Data.Accounts {
		creds = append(creds, EmailCred{
			Email:        acc.Email,
			Password:     acc.Password,
			RefreshToken: acc.RefreshToken,
			ClientId:     acc.ClientId,
		})
	}
	return creds, nil
}

// errOutOfStock sentinel
var errOutOfStock = fmt.Errorf("zeus-x: out of stock")

// isOutOfStock kiểm tra xem error có phải do HẾT HÀNG hay không.
// Nhận ra cả sentinel errOutOfStock và các chuỗi thông báo hết hàng phổ biến từ API.
// err: lỗi trả về từ buyBatch hoặc các API call.
//
// QUAN TRỌNG: phải LOẠI TRỪ "insufficient balance / số dư" — HẾT TIỀN KHÁC hết hàng.
// Trước đây match bare "insufficient" → "Insufficient balance" bị nhận nhầm là hết hàng
// → pool retry vô hạn thay vì fail → verify kẹt mãi.
func isOutOfStock(err error) bool {
	if err == errOutOfStock {
		return true
	}
	msg := strings.ToLower(err.Error())
	// HẾT TIỀN → KHÔNG phải hết hàng (fail nhanh, không retry vô hạn).
	if strings.Contains(msg, "balance") || strings.Contains(msg, "số dư") ||
		strings.Contains(msg, "insufficient balance") || strings.Contains(msg, "insufficient_balance") {
		return false
	}
	return strings.Contains(msg, "out of stock") ||
		strings.Contains(msg, "hết hàng") ||
		strings.Contains(msg, "insufficient quantity") ||
		strings.Contains(msg, "insufficient stock") ||
		strings.Contains(msg, "not enough") ||
		strings.Contains(msg, "no stock") ||
		strings.Contains(msg, "unavailable")
}

// isInsufficientBalance kiểm tra lỗi có phải HẾT TIỀN (số dư không đủ) không.
// Dùng để batch fallback: mua batch lớn fail vì hết tiền → thử mua 1 con (tận dụng
// số dư còn lại) thay vì fail toàn bộ batch.
func isInsufficientBalance(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "insufficient balance") ||
		strings.Contains(msg, "insufficient_balance") ||
		strings.Contains(msg, "số dư không đủ") ||
		strings.Contains(msg, "số dư") ||
		(strings.Contains(msg, "balance") && strings.Contains(msg, "insufficient"))
}

// WaitForCode poll OTP qua tools.dongvanfb.net/api/get_code_oauth2.
func (z *ZeusX) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
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

		code, err := z.fetchOTPFromDVFB(ctx)
		if err == nil && code != "" {
			return code, nil
		}
		if err != nil {
			z.notify(fmt.Sprintf("[ZeusX][Poll %d/%d] Lỗi: %v", attempt+1, maxRetry, err))
		} else {
			z.notify(fmt.Sprintf("[ZeusX][Poll %d/%d] Chưa có OTP...", attempt+1, maxRetry))
		}

		if attempt < maxRetry-1 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			}
		}
	}

	return "", fmt.Errorf("zeus-x: no OTP code received after %d attempts", maxRetry)
}

// fetchOTPFromDVFB gọi tools.dongvanfb.net/api/get_code_oauth2
func (z *ZeusX) fetchOTPFromDVFB(ctx context.Context) (string, error) {
	return ReadOTPWithPriority(ctx, z.otpPriority, z.emailAddr, z.password, z.refreshToken, z.clientId, z.client, z.notify)
}

// GetEmail trả về địa chỉ email đã mua (trống nếu chưa gọi CreateEmail).
func (z *ZeusX) GetEmail() string { return z.emailAddr }

// Close giải phóng tài nguyên — ZeusX không giữ kết nối nên là no-op.
func (z *ZeusX) Close() {}

// ─── Snapshotter (TempMail reuse) ──────────────────────────────────────────
//
// ZeusX dùng OAuth2 refresh_token + clientId để đọc inbox qua dongvanfb API.
// Snapshot encode 4 field: email/password/refreshToken/clientId. Restore re-init
// state để WaitForCode work mà không cần CreateEmail mới.

type zeusXSnapshot struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	RefreshToken string `json:"refresh_token"`
	ClientId     string `json:"client_id"`
}

// Snapshot serialize creds. Trả "" nếu chưa gọi CreateEmail.
func (z *ZeusX) Snapshot() (string, error) {
	if z.emailAddr == "" {
		return "", nil
	}
	b, err := json.Marshal(zeusXSnapshot{
		Email:        z.emailAddr,
		Password:     z.password,
		RefreshToken: z.refreshToken,
		ClientId:     z.clientId,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Restore re-init từ snapshot. Sau Restore, WaitForCode dùng creds này gọi
// dongvanfb API → đọc inbox.
func (z *ZeusX) Restore(creds string) error {
	var s zeusXSnapshot
	if err := json.Unmarshal([]byte(creds), &s); err != nil {
		return err
	}
	z.emailAddr = s.Email
	z.password = s.Password
	z.refreshToken = s.RefreshToken
	z.clientId = s.ClientId
	return nil
}

// Release trả mail CHƯA DÙNG về LOCAL pool (không refund server — zeus-x không có API).
// Mail đã trừ quota lúc Purchase, nhưng nếu verify fail sớm (chưa add vào FB account)
// thì trả về pool cho account khác dùng → tránh mua mail mới lãng phí.
func (z *ZeusX) Release(ctx context.Context) error {
	if z.pool != nil && z.emailAddr != "" {
		z.pool.Return(EmailCred{Email: z.emailAddr, Password: z.password, RefreshToken: z.refreshToken, ClientId: z.clientId})
		z.emailAddr = "" // mark consumed → tránh double-return
	}
	return nil
}
