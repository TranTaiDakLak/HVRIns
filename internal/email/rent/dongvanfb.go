// dongvanfb_mail.go — DongVanFB email service (mua mail từ api.dongvanfb.net)
// Mua batch email qua Buy API → đọc OTP qua tools.dongvanfb.net/api/get_code_oauth2
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

// === DongVanFB Buy API response ===
type dvfbBuyResp struct {
	ErrorCode int    `json:"error_code"`
	Status    bool   `json:"status"`
	Message   string `json:"message"`
	Data      struct {
		OrderCode   string   `json:"order_code"`
		AccountType string   `json:"account_type"`
		Quality     int      `json:"quality"`
		Price       int      `json:"price"`
		TotalAmount int      `json:"total_amount"`
		Balance     int      `json:"balance"`
		ListData    []string `json:"list_data"` // format: email|password|refresh_token|client_id
	} `json:"data"`
}

// === DongVanFB Account Type API response ===
type dvfbAccountTypeResp struct {
	ErrorCode int    `json:"error_code"`
	Status    bool   `json:"status"`
	Message   string `json:"message"`
	Data      []struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Quality int    `json:"quality"`
		Price   int    `json:"price"`
	} `json:"data"`
}

// === DongVanFB get_code_oauth2 response ===
type dvfbCodeResp struct {
	Status  bool   `json:"status"`
	Email   string `json:"email"`
	Code    string `json:"code"`
	Content string `json:"content"`
	Date    string `json:"date"`
}

// DongVanFB implements email.Service
type DongVanFB struct {
	apiKey      string
	accountType string
	client      *http.Client
	onStatus    func(string)
	pool        *CredPool // shared pool
	otpPriority string    // "dongvan" (default) | "unlimit" — chọn primary OTP reader

	// Populated after CreateEmail
	emailAddr    string
	password     string
	refreshToken string
	clientId     string
}

// NewDongVanFB tạo DongVanFB service
func NewDongVanFB(apiKey, accountType, proxyStr string) *DongVanFB {
	return &DongVanFB{
		apiKey:      apiKey,
		accountType: accountType,
		client:      proxy.CreateClient(proxyStr, 30*time.Second),
	}
}

// SetPool gán shared pool
func (d *DongVanFB) SetPool(p *CredPool) { d.pool = p }

// SetOnStatus gán callback
func (d *DongVanFB) SetOnStatus(fn func(string)) { d.onStatus = fn }

// SetOTPPriority chọn nguồn đọc OTP ưu tiên ("dongvan" | "unlimit").
func (d *DongVanFB) SetOTPPriority(p string) { d.otpPriority = p }

// notify gửi thông báo trạng thái đến callback UI nếu đã được gán.
// msg: nội dung thông báo hiển thị trên UI (ví dụ: "Mua thành công: ...", "Hết hàng...").
func (d *DongVanFB) notify(msg string) {
	if d.onStatus != nil {
		d.onStatus(msg)
	}
}

// CreateEmail: nếu có pool → pull từ pool; không có → mua đơn lẻ
func (d *DongVanFB) CreateEmail(ctx context.Context) (string, error) {
	if d.pool != nil {
		cred, err := d.pool.Get(ctx)
		if err != nil {
			return "", err
		}
		d.emailAddr = cred.Email
		d.password = cred.Password
		d.refreshToken = cred.RefreshToken
		d.clientId = cred.ClientId
		d.notify(fmt.Sprintf("[DongVanFB] Lấy từ pool: %s", d.emailAddr))
		return d.emailAddr, nil
	}
	return d.buyLegacy(ctx)
}

// buyLegacy mua đơn lẻ (fallback khi không dùng pool)
func (d *DongVanFB) buyLegacy(ctx context.Context) (string, error) {
	const retryDelay = 5 * time.Second
	for attempt := 1; ; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		inStock, err := d.checkStock(ctx)
		if err != nil {
			return "", fmt.Errorf("dongvanfb check stock: %w", err)
		}
		if inStock <= 0 {
			d.notify(fmt.Sprintf("[DongVanFB] Hết hàng (account_type=%s), thử lại sau 5s... (lần %d)", d.accountType, attempt))
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(retryDelay):
			}
			continue
		}

		creds, err := d.buyBatch(ctx, 1)
		if err == nil && len(creds) > 0 {
			d.emailAddr = creds[0].Email
			d.password = creds[0].Password
			d.refreshToken = creds[0].RefreshToken
			d.clientId = creds[0].ClientId
			d.notify(fmt.Sprintf("[DongVanFB] Mua thành công: %s", d.emailAddr))
			return d.emailAddr, nil
		}
		errMsg := ""
		if err != nil {
			errMsg = strings.ToLower(err.Error())
		}
		if strings.Contains(errMsg, "out of stock") || strings.Contains(errMsg, "hết hàng") ||
			strings.Contains(errMsg, "not enough") || strings.Contains(errMsg, "insufficient") {
			d.notify(fmt.Sprintf("[DongVanFB] Hết hàng, thử lại sau 5s... (lần %d)", attempt))
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

// NewDongVanFBPool tạo CredPool dùng DongVanFB để mua batch
func NewDongVanFBPool(apiKey, accountType, proxyStr string, batchSize int, notify func(string)) *CredPool {
	d := NewDongVanFB(apiKey, accountType, proxyStr)
	d.onStatus = notify

	buyFn := func(ctx context.Context, n int) ([]EmailCred, error) {
		const retryDelay = 5 * time.Second
		for attempt := 1; ; attempt++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			// Check stock trước khi mua
			inStock, err := d.checkStock(ctx)
			if err != nil {
				return nil, fmt.Errorf("dongvanfb check stock: %w", err)
			}
			if inStock <= 0 {
				if notify != nil {
					notify(fmt.Sprintf("[DongVanFB Pool] Hết hàng, thử lại sau 5s... (lần %d)", attempt))
				}
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(retryDelay):
				}
				continue
			}

			// Mua tối đa số lượng còn trong kho
			buyCount := n
			if inStock < buyCount {
				buyCount = inStock
			}
			creds, err := d.buyBatch(ctx, buyCount)
			if err == nil {
				return creds, nil
			}
			errMsg := strings.ToLower(err.Error())
			if strings.Contains(errMsg, "out of stock") || strings.Contains(errMsg, "hết hàng") ||
				strings.Contains(errMsg, "not enough") || strings.Contains(errMsg, "insufficient") {
				if notify != nil {
					notify(fmt.Sprintf("[DongVanFB Pool] Hết hàng, thử lại sau 5s... (lần %d)", attempt))
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

// checkStock kiểm tra tồn kho theo account_type
func (d *DongVanFB) checkStock(ctx context.Context) (int, error) {
	endpoint := fmt.Sprintf("https://api.dongvanfb.net/user/account_type?apikey=%s",
		url.QueryEscape(d.apiKey))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return 0, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("dongvanfb account_type: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var result dvfbAccountTypeResp
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("dongvanfb account_type parse: %w — body: %.200s", err, body)
	}
	if !result.Status {
		return 0, fmt.Errorf("dongvanfb account_type: %s", result.Message)
	}

	for _, item := range result.Data {
		if fmt.Sprintf("%d", item.ID) == d.accountType {
			return item.Quality, nil
		}
	}
	return 0, fmt.Errorf("dongvanfb: không tìm thấy account_type=%s", d.accountType)
}

// buyBatch mua n emails cùng lúc
func (d *DongVanFB) buyBatch(ctx context.Context, n int) ([]EmailCred, error) {
	endpoint := fmt.Sprintf(
		"https://api.dongvanfb.net/user/buy?apikey=%s&account_type=%s&quality=%d&type=full",
		url.QueryEscape(d.apiKey),
		url.QueryEscape(d.accountType),
		n,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("dongvanfb buy: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var result dvfbBuyResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("dongvanfb buy parse: %w — body: %.200s", err, body)
	}
	if !result.Status || result.ErrorCode != 200 {
		return nil, fmt.Errorf("dongvanfb buy [%d]: %s", result.ErrorCode, result.Message)
	}
	if len(result.Data.ListData) == 0 {
		return nil, fmt.Errorf("dongvanfb buy: out of stock (no list_data)")
	}

	creds := make([]EmailCred, 0, len(result.Data.ListData))
	for _, line := range result.Data.ListData {
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}
		creds = append(creds, EmailCred{
			Email:        strings.TrimSpace(parts[0]),
			Password:     strings.TrimSpace(parts[1]),
			RefreshToken: strings.TrimSpace(parts[2]),
			ClientId:     strings.TrimSpace(parts[3]),
		})
	}
	if len(creds) == 0 {
		return nil, fmt.Errorf("dongvanfb buy: không parse được account nào")
	}
	d.notify(fmt.Sprintf("[DongVanFB] Mua batch %d email thành công (balance: %d)", len(creds), result.Data.Balance))
	return creds, nil
}

// WaitForCode poll dongvanfb get_code_oauth2
func (d *DongVanFB) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
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

		code, err := d.fetchOTP(ctx)
		if err == nil && code != "" {
			return code, nil
		}
		if err != nil {
			d.notify(fmt.Sprintf("[DongVanFB][Poll %d/%d] Lỗi: %v", attempt+1, maxRetry, err))
		} else {
			d.notify(fmt.Sprintf("[DongVanFB][Poll %d/%d] Chưa có OTP...", attempt+1, maxRetry))
		}

		if attempt < maxRetry-1 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			}
		}
	}

	return "", fmt.Errorf("dongvanfb: no OTP code received after %d attempts", maxRetry)
}

// fetchOTP gọi tools.dongvanfb.net/api/get_code_oauth2 qua d.client (có thể có proxy).
func (d *DongVanFB) fetchOTP(ctx context.Context) (string, error) {
	return ReadOTPWithPriority(ctx, d.otpPriority, d.emailAddr, d.password, d.refreshToken, d.clientId, d.client, d.notify)
}

// GetEmail trả về địa chỉ email đã mua/tạo (trống nếu chưa gọi CreateEmail).
func (d *DongVanFB) GetEmail() string { return d.emailAddr }

// Close giải phóng tài nguyên — DongVanFB không giữ kết nối nên là no-op.
func (d *DongVanFB) Close() {}

// ─── Snapshotter (TempMail reuse) ──────────────────────────────────────────
// DongVanFB cùng pattern OAuth2 với ZeusX (Hotmail/Outlook). Snapshot 4 field.

func (d *DongVanFB) Snapshot() (string, error) {
	if d.emailAddr == "" {
		return "", nil
	}
	b, err := json.Marshal(CommonOAuth2Snapshot{
		Email:        d.emailAddr,
		Password:     d.password,
		RefreshToken: d.refreshToken,
		ClientId:     d.clientId,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (d *DongVanFB) Restore(creds string) error {
	var s CommonOAuth2Snapshot
	if err := json.Unmarshal([]byte(creds), &s); err != nil {
		return err
	}
	d.emailAddr = s.Email
	d.password = s.Password
	d.refreshToken = s.RefreshToken
	d.clientId = s.ClientId
	return nil
}

// Release trả mail CHƯA DÙNG về local pool — tránh mua mới lãng phí khi verify fail sớm.
func (d *DongVanFB) Release(ctx context.Context) error {
	if d.pool != nil && d.emailAddr != "" {
		d.pool.Return(EmailCred{Email: d.emailAddr, Password: d.password, RefreshToken: d.refreshToken, ClientId: d.clientId})
		d.emailAddr = "" // mark consumed → tránh double-return
	}
	return nil
}
