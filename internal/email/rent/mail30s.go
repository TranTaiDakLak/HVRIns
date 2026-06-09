// mail30s_mail.go — Mail30s email service (mua Hotmail/Outlook từ mailotp.com / mail30s.com)
// Mua batch email qua /order/create API → đọc OTP qua tools.dongvanfb.net/api/get_code_oauth2
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

// === Order create API response ===
type mail30sOrderResp struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error"`
	Data    struct {
		Emails []struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			FullInfo string `json:"full_info"` // email|password|refresh_token|client_id[|...]
		} `json:"emails"`
	} `json:"data"`
}

// === Products API response (dùng check stock) ===
type mail30sProductsResp struct {
	Success bool `json:"success"`
	Data    []struct {
		ID           int    `json:"id"`
		Name         string `json:"name"`
		Slug         string `json:"slug"`
		Price        float64 `json:"price"`
		PriceDisplay string `json:"price_display"`
		Stock        int    `json:"stock"`
	} `json:"data"`
}

const mail30sBaseURL = "https://api.mailotp.com/api/automation"

// Mail30s implements email.Service
type Mail30s struct {
	apiKey      string
	productSlug string
	client      *http.Client
	onStatus    func(string)
	pool        *CredPool
	otpPriority string // "dongvan" (default) | "unlimit" — chọn primary OTP reader

	emailAddr    string
	password     string
	refreshToken string
	clientID     string
}

// NewMail30s tạo Mail30s service
func NewMail30s(apiKey, productSlug, proxyStr string) *Mail30s {
	return &Mail30s{
		apiKey:      apiKey,
		productSlug: productSlug,
		client:      proxy.CreateClient(proxyStr, 30*time.Second),
	}
}

// SetPool gán shared pool — CreateEmail sẽ pull từ pool thay vì mua riêng lẻ.
func (m *Mail30s) SetPool(p *CredPool) { m.pool = p }

// SetOnStatus gán callback nhận thông báo trạng thái hiển thị trên UI.
func (m *Mail30s) SetOnStatus(fn func(string)) { m.onStatus = fn }

// SetOTPPriority chọn nguồn đọc OTP ưu tiên ("dongvan" | "unlimit").
func (m *Mail30s) SetOTPPriority(p string) { m.otpPriority = p }

// GetEmail trả về địa chỉ email đã mua (trống nếu chưa gọi CreateEmail).
func (m *Mail30s) GetEmail() string { return m.emailAddr }

// Close giải phóng tài nguyên — Mail30s không giữ kết nối nên là no-op.
func (m *Mail30s) Close() {}

// notify gửi thông báo trạng thái đến callback UI nếu đã được gán.
// msg: nội dung thông báo (ví dụ: "Mua thành công: ...", "Hết hàng, thử lại...").
func (m *Mail30s) notify(msg string) {
	if m.onStatus != nil {
		m.onStatus(msg)
	}
}

// CreateEmail: nếu có pool → pull từ pool; không có → mua đơn lẻ
func (m *Mail30s) CreateEmail(ctx context.Context) (string, error) {
	if m.pool != nil {
		cred, err := m.pool.Get(ctx)
		if err != nil {
			return "", err
		}
		m.emailAddr = cred.Email
		m.password = cred.Password
		m.refreshToken = cred.RefreshToken
		m.clientID = cred.ClientId
		m.notify(fmt.Sprintf("[Mail30s] Lấy từ pool: %s", m.emailAddr))
		return m.emailAddr, nil
	}
	return m.buyLegacy(ctx)
}

// buyLegacy mua đơn lẻ, retry khi hết hàng
func (m *Mail30s) buyLegacy(ctx context.Context) (string, error) {
	const retryDelay = 5 * time.Second
	for attempt := 1; ; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		creds, err := m.buyBatch(ctx, 1)
		if err == nil && len(creds) > 0 {
			m.emailAddr = creds[0].Email
			m.password = creds[0].Password
			m.refreshToken = creds[0].RefreshToken
			m.clientID = creds[0].ClientId
			m.notify(fmt.Sprintf("[Mail30s] Mua thành công: %s", m.emailAddr))
			return m.emailAddr, nil
		}
		if err != nil && isOutOfStock(err) {
			m.notify(fmt.Sprintf("[Mail30s] Hết hàng (slug=%s), thử lại sau 5s... (lần %d)", m.productSlug, attempt))
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

// NewMail30sPool tạo CredPool dùng Mail30s để mua batch
func NewMail30sPool(apiKey, productSlug, proxyStr string, batchSize int, notify func(string)) *CredPool {
	s := NewMail30s(apiKey, productSlug, proxyStr)
	s.onStatus = notify

	buyFn := func(ctx context.Context, n int) ([]EmailCred, error) {
		const retryDelay = 5 * time.Second
		for attempt := 1; ; attempt++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			creds, err := s.buyBatch(ctx, n)
			if err == nil {
				return creds, nil
			}
			if isOutOfStock(err) {
				if notify != nil {
					notify(fmt.Sprintf("[Mail30s Pool] Hết hàng, thử lại sau 5s... (lần %d)", attempt))
				}
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(retryDelay):
				}
				continue
			}
			// HẾT TIỀN khi mua batch lớn → thử mua 1 con (tận dụng số dư còn lại).
			// Vd số dư đủ 2 mail nhưng batch 10 fail → mua 1 con/lần cho đến khi hết tiền hẳn.
			if isInsufficientBalance(err) && n > 1 {
				if notify != nil {
					notify("[Mail30s Pool] Số dư không đủ mua batch — thử mua 1 con...")
				}
				if creds1, err1 := s.buyBatch(ctx, 1); err1 == nil {
					return creds1, nil
				}
				// Mua 1 vẫn fail = hết tiền hẳn → trả lỗi rõ ràng.
				return nil, fmt.Errorf("hết tiền (số dư không đủ mua mail)")
			}
			return nil, err
		}
	}

	threshold := poolMax(1, batchSize/3)
	return NewCredPool(batchSize, threshold, buyFn, notify)
}

// buyBatch gọi GET /order/create, mua n accounts
func (m *Mail30s) buyBatch(ctx context.Context, n int) ([]EmailCred, error) {
	params := url.Values{}
	params.Set("api_key", m.apiKey)
	params.Set("product_slug", m.productSlug)
	params.Set("quantity", fmt.Sprintf("%d", n))
	reqURL := mail30sBaseURL + "/order/create?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mail30s buy: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var result mail30sOrderResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("mail30s buy parse: %w — body: %.200s", err, body)
	}

	if !result.Success {
		errCode := strings.ToUpper(result.Error)
		if errCode == "OUT_OF_STOCK" {
			return nil, errOutOfStock
		}
		if errCode == "INSUFFICIENT_BALANCE" {
			return nil, fmt.Errorf("mail30s: số dư không đủ")
		}
		return nil, fmt.Errorf("mail30s buy: %s", result.Message)
	}

	if len(result.Data.Emails) == 0 {
		return nil, errOutOfStock
	}

	creds := make([]EmailCred, 0, len(result.Data.Emails))
	for _, item := range result.Data.Emails {
		// full_info = email|password|refresh_token|client_id[|...]
		parts := strings.Split(item.FullInfo, "|")
		if len(parts) >= 4 {
			creds = append(creds, EmailCred{
				Email:        strings.TrimSpace(parts[0]),
				Password:     strings.TrimSpace(parts[1]),
				RefreshToken: strings.TrimSpace(parts[2]),
				ClientId:     strings.TrimSpace(parts[3]),
			})
		} else if len(parts) >= 2 {
			// fallback: email|password (không có OAuth2)
			creds = append(creds, EmailCred{
				Email:    strings.TrimSpace(parts[0]),
				Password: strings.TrimSpace(parts[1]),
			})
		} else if item.Email != "" {
			creds = append(creds, EmailCred{
				Email:    item.Email,
				Password: item.Password,
			})
		}
	}

	if len(creds) == 0 {
		return nil, fmt.Errorf("mail30s buy: không parse được account nào từ %d dòng", len(result.Data.Emails))
	}
	m.notify(fmt.Sprintf("[Mail30s] Mua batch %d email thành công", len(creds)))
	return creds, nil
}

// WaitForCode poll OTP qua tools.dongvanfb.net/api/get_code_oauth2.
func (m *Mail30s) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
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
			m.notify(fmt.Sprintf("[Mail30s][Poll %d/%d] Lỗi: %v", attempt+1, maxRetry, err))
		} else {
			m.notify(fmt.Sprintf("[Mail30s][Poll %d/%d] Chưa có OTP...", attempt+1, maxRetry))
		}

		if attempt < maxRetry-1 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			}
		}
	}

	return "", fmt.Errorf("mail30s: no OTP code received after %d attempts", maxRetry)
}

// fetchOTP gọi tools.dongvanfb.net/api/get_code_oauth2
func (m *Mail30s) fetchOTP(ctx context.Context) (string, error) {
	return ReadOTPWithPriority(ctx, m.otpPriority, m.emailAddr, m.password, m.refreshToken, m.clientID, m.client, m.notify)
}
