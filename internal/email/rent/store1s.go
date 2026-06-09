// store1s_mail.go — Store1s email service (mua Hotmail/Outlook từ store1s.com)
// Mua batch email qua buy_product API → đọc OTP qua tools.dongvanfb.net/api/get_messages_oauth2
package rent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"HVRIns/internal/proxy"
)

// === Buy API response ===
type store1sBuyResp struct {
	Status  string   `json:"status"`
	Msg     string   `json:"msg"`
	TransID string   `json:"trans_id"`
	Data    []string `json:"data"` // format: email|pass|refresh_token|client_id[|recovery]
}

// === Products API response (dùng check stock) ===
type store1sProductsResp struct {
	Status     string `json:"status"`
	Categories []struct {
		Products []struct {
			ID     string `json:"id"`
			Amount int    `json:"amount"`
		} `json:"products"`
	} `json:"categories"`
}

// Store1sProduct — 1 product trả về cho frontend (dropdown + stock).
type Store1sProduct struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price string `json:"price"`
	Stock int    `json:"stock"`
}

// store1sProductsFullResp parse đầy đủ id/name/price/amount từ products.php.
type store1sProductsFullResp struct {
	Status     string `json:"status"`
	Msg        string `json:"msg"`
	Categories []struct {
		Products []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Price  string `json:"price"`
			Amount int    `json:"amount"`
		} `json:"products"`
	} `json:"categories"`
}

// FetchStore1sProducts gọi products.php (qua backend Go — tránh CORS từ webview)
// và trả về danh sách product có stock > 0. Dùng cho UI dropdown + check tồn kho.
func FetchStore1sProducts(ctx context.Context, apiKey string) ([]Store1sProduct, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("thiếu API key")
	}
	url := "https://store1s.com/api/products.php?api_key=" + apiKey
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kết nối store1s: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var parsed store1sProductsFullResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse products: %w — body: %.200s", err, body)
	}
	if parsed.Status != "success" {
		return nil, fmt.Errorf("store1s: %s", parsed.Msg)
	}
	var out []Store1sProduct
	for _, cat := range parsed.Categories {
		for _, p := range cat.Products {
			out = append(out, Store1sProduct{ID: p.ID, Name: p.Name, Price: p.Price, Stock: p.Amount})
		}
	}
	return out, nil
}

// Store1s implements email.Service
type Store1s struct {
	apiKey      string
	productID   string
	client      *http.Client
	onStatus    func(string)
	pool        *CredPool
	otpPriority string // "dongvan" (default) | "unlimit" — chọn primary OTP reader

	// Populated after CreateEmail
	emailAddr    string
	password     string
	refreshToken string
	clientID     string
}

// NewStore1s tạo Store1s service
func NewStore1s(apiKey, productID, proxyStr string) *Store1s {
	return &Store1s{
		apiKey:    apiKey,
		productID: productID,
		client:    proxy.CreateClient(proxyStr, 30*time.Second),
	}
}

// SetPool gán shared pool — CreateEmail sẽ pull từ pool thay vì mua riêng lẻ.
func (s *Store1s) SetPool(p *CredPool) { s.pool = p }

// SetOnStatus gán callback nhận thông báo trạng thái hiển thị trên UI.
func (s *Store1s) SetOnStatus(fn func(string)) { s.onStatus = fn }

// SetOTPPriority chọn nguồn đọc OTP ưu tiên ("dongvan" | "unlimit").
func (s *Store1s) SetOTPPriority(p string) { s.otpPriority = p }

// GetEmail trả về địa chỉ email đã mua (trống nếu chưa gọi CreateEmail).
func (s *Store1s) GetEmail() string { return s.emailAddr }

// Close giải phóng tài nguyên — Store1s không giữ kết nối nên là no-op.
func (s *Store1s) Close() {}

// notify gửi thông báo trạng thái đến callback UI nếu đã được gán.
// msg: nội dung thông báo (ví dụ: "Mua thành công: ...", "Hết hàng, thử lại...").
func (s *Store1s) notify(msg string) {
	if s.onStatus != nil {
		s.onStatus(msg)
	}
}

// CreateEmail: nếu có pool → pull từ pool; không có → mua đơn lẻ
func (s *Store1s) CreateEmail(ctx context.Context) (string, error) {
	if s.pool != nil {
		cred, err := s.pool.Get(ctx)
		if err != nil {
			return "", err
		}
		s.emailAddr = cred.Email
		s.password = cred.Password
		s.refreshToken = cred.RefreshToken
		s.clientID = cred.ClientId
		s.notify(fmt.Sprintf("[Store1s] Lấy từ pool: %s", s.emailAddr))
		return s.emailAddr, nil
	}
	return s.buyLegacy(ctx)
}

// buyLegacy mua đơn lẻ, retry khi hết hàng
func (s *Store1s) buyLegacy(ctx context.Context) (string, error) {
	const retryDelay = 5 * time.Second
	for attempt := 1; ; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		creds, err := s.buyBatch(ctx, 1)
		if err == nil && len(creds) > 0 {
			s.emailAddr = creds[0].Email
			s.password = creds[0].Password
			s.refreshToken = creds[0].RefreshToken
			s.clientID = creds[0].ClientId
			s.notify(fmt.Sprintf("[Store1s] Mua thành công: %s", s.emailAddr))
			return s.emailAddr, nil
		}
		if err != nil && isOutOfStock(err) {
			s.notify(fmt.Sprintf("[Store1s] Hết hàng (product_id=%s), thử lại sau 5s... (lần %d)", s.productID, attempt))
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

// NewStore1sPool tạo CredPool dùng Store1s để mua batch
func NewStore1sPool(apiKey, productID, proxyStr string, batchSize int, notify func(string)) *CredPool {
	s := NewStore1s(apiKey, productID, proxyStr)
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
					notify(fmt.Sprintf("[Store1s Pool] Hết hàng, thử lại sau 5s... (lần %d)", attempt))
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

// buyBatch gọi POST /api/buy_product, mua n accounts
func (s *Store1s) buyBatch(ctx context.Context, n int) ([]EmailCred, error) {
	// Build multipart form-data
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("action", "buyProduct")
	_ = w.WriteField("id", s.productID)
	_ = w.WriteField("amount", fmt.Sprintf("%d", n))
	_ = w.WriteField("coupon", "")
	_ = w.WriteField("api_key", s.apiKey)
	_ = w.Close()

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://store1s.com/api/buy_product",
		bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("store1s buy: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var result store1sBuyResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("store1s buy parse: %w — body: %.200s", err, body)
	}
	if result.Status != "success" {
		msgLower := strings.ToLower(result.Msg)
		if strings.Contains(msgLower, "hết hàng") || strings.Contains(msgLower, "out of stock") ||
			strings.Contains(msgLower, "không đủ") || strings.Contains(msgLower, "insufficient") ||
			strings.Contains(msgLower, "không có hàng") {
			return nil, errOutOfStock
		}
		return nil, fmt.Errorf("store1s buy: %s", result.Msg)
	}
	if len(result.Data) == 0 {
		return nil, errOutOfStock
	}

	creds := make([]EmailCred, 0, len(result.Data))
	for _, line := range result.Data {
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
		return nil, fmt.Errorf("store1s buy: không parse được account nào từ %d dòng", len(result.Data))
	}
	s.notify(fmt.Sprintf("[Store1s] Mua batch %d email thành công (trans_id: %s)", len(creds), result.TransID))
	return creds, nil
}

// WaitForCode poll OTP qua tools.dongvanfb.net/api/get_code_oauth2.
func (s *Store1s) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
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

		code, err := s.fetchOTP(ctx)
		if err == nil && code != "" {
			return code, nil
		}
		if err != nil {
			s.notify(fmt.Sprintf("[Store1s][Poll %d/%d] Lỗi: %v", attempt+1, maxRetry, err))
		} else {
			s.notify(fmt.Sprintf("[Store1s][Poll %d/%d] Chưa có OTP...", attempt+1, maxRetry))
		}

		if attempt < maxRetry-1 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			}
		}
	}

	return "", fmt.Errorf("store1s: no OTP code received after %d attempts", maxRetry)
}

// smail1sResp — response từ smail1s.com/get_messages
type smail1sResp struct {
	Data []struct {
		Email    string `json:"email"`
		Error    interface{} `json:"error"`
		Messages []struct {
			From    string `json:"from"`
			Subject string `json:"subject"`
			Code    string `json:"code"`
			Message string `json:"message"`
			Date    string `json:"date"`
		} `json:"messages"`
	} `json:"data"`
}

// fetchOTP đọc OTP của mailbox OAuth2 (Hotmail/Outlook) qua priority helper.
// User chọn primary qua SetOTPPriority ("dongvan" hoặc "unlimit"); helper tự fallback
// sang reader còn lại nếu primary fail (lỗi mạng/parse). Default: dongvan primary.
//
// Cả 2 reader đọc CÙNG 1 mailbox Microsoft qua refresh_token+client_id nên kết quả
// tương đương; dual-source tăng khả năng bắt OTP đúng lúc + chịu lỗi 1 endpoint.
func (s *Store1s) fetchOTP(ctx context.Context) (string, error) {
	return ReadOTPWithPriority(ctx, s.otpPriority, s.emailAddr, s.password, s.refreshToken, s.clientID, s.client, s.notify)
}
