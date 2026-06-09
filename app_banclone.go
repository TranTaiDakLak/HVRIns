// app_banclone.go — Banclone (e-commerce push) integration.
// Tách nguyên khối từ app.go — KHÔNG sửa logic.
package main

import (
	"HVRIns/internal/httpx"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// pushToBanclone đẩy batch accounts lên banclone.pro/api/importAccount.php (POST)
// API: code, api_key, filter, account (newline-separated) — POST body form-encoded
// pushToBanclone POST batch acc lên banclone.pro/importAccount.php.
// Timeout rất rộng vì site phản hồi chậm khi xử lý batch lớn — KHÔNG đặt timeout
// chặt vì sẽ trigger "Client.Timeout exceeded while awaiting headers" trong khi
// thực tế site vẫn đang xử lý → mất acc.
//
// HTTP transport dùng cho upload — share giữa các call để reuse keep-alive connection.
// Task 4 cap: ResponseHeaderTimeout giảm 10p → 60s. Đây là safety net khi server treo
// không trả header — ctx.WithTimeout(pushTimeoutFor) là primary cap (120-300s tổng).
// 60s đủ rộng cho banclone slow response nhưng đảm bảo hard-fail nhanh khi server die.
var bancloneTransport = &http.Transport{
	DialContext:           (&net.Dialer{Timeout: 30 * time.Second, KeepAlive: 60 * time.Second}).DialContext,
	TLSHandshakeTimeout:   30 * time.Second,
	ResponseHeaderTimeout: 60 * time.Second, // Task 4: 10p → 60s; ctx WithTimeout là primary cap
	IdleConnTimeout:       2 * time.Minute,
	ExpectContinueTimeout: 1 * time.Second,
	MaxIdleConns:          10,
	MaxIdleConnsPerHost:   uploadMaxConcurrent + 2, // đủ cho concurrent pushes
	DisableKeepAlives:     false,
}

// pushTimeoutFor — tính timeout động theo batch size (Task 4 spec).
// Banclone API: 1 acc ~0.5-1s xử lý server-side. Buffer 2× cho network jitter.
//
// Tiers (cap tối đa 300s, KHÔNG còn 600s/30min/60min):
//
//	batch ≤ 100  → 120s  (small batch — đủ rộng cho jitter)
//	batch ≤ 500  → 180s
//	batch > 500  → 300s  (hard cap — không khuyến khích batch >1000, sẽ vượt cap → fail)
func pushTimeoutFor(batchSize int) time.Duration {
	switch {
	case batchSize <= 100:
		return 120 * time.Second
	case batchSize <= 500:
		return 180 * time.Second
	default:
		return 300 * time.Second
	}
}

// pushToBanclone — push 1 batch account lên banclone.pro.
//
// ctx (Task 4): run-scoped ctx của runUploadSite. Stop upload cancel ctx → request
// abort ngay (Transport.CancelRequest qua ctx). KHÔNG dùng context.Background() ở đây.
// Nếu ctx == nil → defensive fallback Background (chỉ cho test/standalone, prod luôn pass run ctx).
//
// Timeout: pushTimeoutFor(batchSize) — 120s/180s/300s tier (xem hàm). Cap tối đa 300s.
// KHÔNG còn 60-min hay 30-min timeout.
func pushToBanclone(ctx context.Context, code, apiKey, filter string, lines []string) (int, error) {
	if len(lines) == 0 {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// Fast-fail: nếu ctx đã cancel TRƯỚC khi build request (vd Stop fired ngay sau enqueue)
	// → không cần build body string + alloc; trả luôn để retry path tự skip.
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("upload bị cancel (Stop) trước khi gửi: %w", err)
	}

	accountStr := strings.Join(lines, "\n")
	body := url.Values{
		"code":    {code},
		"api_key": {apiKey},
		"filter":  {filter},
		"account": {accountStr},
	}.Encode()

	// Dynamic timeout theo batch size (Task 4): batch nhỏ 120s, lớn cap 300s.
	timeout := pushTimeoutFor(len(lines))
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "POST",
		"https://banclone.pro/api/importAccount.php", strings.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("tạo request lỗi: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "HVRIns/1.0")
	req.Header.Set("Connection", "keep-alive")

	client := &http.Client{Transport: bancloneTransport}
	resp, err := client.Do(req)
	if err != nil {
		// Phân loại error rõ ràng để user biết nên retry hay sửa config.
		// Ưu tiên check parent ctx (Stop) trước → caller skip retry.
		errStr := err.Error()
		switch {
		case ctx.Err() != nil:
			return 0, fmt.Errorf("upload bị cancel (Stop): %w", ctx.Err())
		case reqCtx.Err() == context.DeadlineExceeded:
			return 0, fmt.Errorf("timeout %s — site quá chậm hoặc batch quá lớn", timeout)
		case strings.Contains(errStr, "Client.Timeout exceeded"), strings.Contains(errStr, "timeout awaiting response headers"):
			return 0, fmt.Errorf("timeout chờ phản hồi từ banclone.pro — sẽ retry")
		case strings.Contains(errStr, "no such host"), strings.Contains(errStr, "dns"):
			return 0, fmt.Errorf("DNS lỗi — kiểm tra mạng")
		case strings.Contains(errStr, "connection refused"), strings.Contains(errStr, "EOF"):
			return 0, fmt.Errorf("site từ chối/đứt kết nối — sẽ retry")
		case strings.Contains(errStr, "tls"), strings.Contains(errStr, "x509"):
			return 0, fmt.Errorf("TLS lỗi: %v", err)
		default:
			return 0, fmt.Errorf("kết nối thất bại: %w", err)
		}
	}
	defer func() {
		// Drain remaining body để Transport có thể reuse connection thay vì close.
		// io.ReadAll ở dưới đã đọc hết, nhưng nếu LimitReader cắt giữa chừng,
		// drain phần thừa rồi mới Close → đảm bảo keep-alive hoạt động.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	// Cap response read 64KB — banclone API trả JSON nhỏ (thường < 1KB), 64KB là dư.
	// Tránh case site trả response bất thường (HTML lỗi, redirect dài) nhồi RAM.
	const maxRespBytes = 64 * 1024
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxRespBytes))
	bodyStr := strings.TrimSpace(string(respBody))
	if len(bodyStr) > 500 {
		bodyStr = bodyStr[:500] + "..."
	}
	if resp.StatusCode == 429 || resp.StatusCode == 503 {
		return 0, fmt.Errorf("rate-limit/quá tải HTTP %d: %s", resp.StatusCode, bodyStr)
	}
	if resp.StatusCode >= 500 {
		return 0, fmt.Errorf("server lỗi HTTP %d: %s", resp.StatusCode, bodyStr)
	}
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("HTTP %d: %s", resp.StatusCode, bodyStr)
	}
	lower := strings.ToLower(bodyStr)
	if strings.Contains(lower, "error") || strings.Contains(lower, "\"success\":false") {
		return 0, fmt.Errorf("site trả lỗi: %s", bodyStr)
	}
	// Final check: nếu Stop fired SAU khi server đã accept request và trả 200 OK,
	// thì server-side đã commit batch → coi là success thật. Nhưng nếu user thấy
	// Stop đang chạy → caller (pushAsync) sẽ skip retry. Không return error ở đây
	// vì batch ĐÃ ghi vào banclone — báo success để khỏi double-push lần sau.
	return len(lines), nil
}

// BancloneProduct thông tin 1 sản phẩm từ banclone.pro
// Code = mã kho hàng thật (từ admin HTML: ?action=product-stock&code=X), rỗng nếu không có admin cookie
type BancloneProduct struct {
	ID           string `json:"id"`
	Code         string `json:"code"` // stock code dùng cho importAccount.php
	Name         string `json:"name"`
	CategoryName string `json:"categoryName"`
	Price        string `json:"price"`
	Amount       int    `json:"amount"`
}

// bancloneAdminLogin đăng nhập banclone.pro bằng username+password.
// Flow: GET /client/login → lấy PHPSESSID + CSRF → POST /ajaxs/client/auth.php → lấy admin_login + user_login.
// Trả về cookie string dạng "admin_login=xxx; user_login=xxx" hoặc error.
//
// ctx: dùng cho cancel + timeout. App shutdown sẽ cancel a.ctx → request không treo.
func bancloneAdminLogin(ctx context.Context, username, password string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", err
	}
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36"
	client := &http.Client{Timeout: 30 * time.Second, Jar: jar}

	// Step 1: GET login page để lấy PHPSESSID + CSRF token
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://banclone.pro/client/login", nil)
	req.Header.Set("User-Agent", ua)
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("kết nối thất bại: %w", err)
	}
	body, _ := httpx.ReadBody(resp.Body, 512*1024)
	resp.Body.Close()

	// Parse CSRF token từ: <input type="hidden" id="csrf_token" value="...">
	csrfRe := regexp.MustCompile(`id="csrf_token"\s+value="([^"]+)"`)
	csrfMatch := csrfRe.FindSubmatch(body)
	if len(csrfMatch) < 2 {
		return "", fmt.Errorf("không tìm được CSRF token từ trang login")
	}
	csrfToken := string(csrfMatch[1])

	// Step 2: POST đăng nhập
	form := url.Values{
		"action":     {"Login"},
		"csrf_token": {csrfToken},
		"username":   {username},
		"password":   {password},
	}
	req2, _ := http.NewRequestWithContext(ctx, "POST", "https://banclone.pro/ajaxs/client/auth.php", strings.NewReader(form.Encode()))
	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req2.Header.Set("X-Requested-With", "XMLHttpRequest")
	req2.Header.Set("Referer", "https://banclone.pro/client/login")
	req2.Header.Set("User-Agent", ua)
	resp2, err := client.Do(req2)
	if err != nil {
		return "", fmt.Errorf("gửi đăng nhập thất bại: %w", err)
	}
	respBody, _ := httpx.ReadBody(resp2.Body, 64*1024)
	resp2.Body.Close()

	// Kiểm tra kết quả
	if !strings.Contains(string(respBody), `"success"`) {
		var errMsg struct {
			Msg string `json:"msg"`
		}
		json.Unmarshal(respBody, &errMsg)
		if errMsg.Msg != "" {
			return "", fmt.Errorf("đăng nhập thất bại: %s", errMsg.Msg)
		}
		return "", fmt.Errorf("đăng nhập thất bại")
	}

	// Lấy cookies từ jar
	u, _ := url.Parse("https://banclone.pro")
	var parts []string
	for _, c := range jar.Cookies(u) {
		if c.Name == "user_login" || c.Name == "admin_login" {
			parts = append(parts, c.Name+"="+c.Value)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("đăng nhập thành công nhưng không nhận được cookie xác thực")
	}
	return strings.Join(parts, "; "), nil
}

// BancloneLogin — export cho Vue: đăng nhập và trả cookie hoặc "ERR|..."
func (a *App) BancloneLogin(username, password string) string {
	cookie, err := bancloneAdminLogin(a.ctx, username, password)
	if err != nil {
		return "ERR|" + err.Error()
	}
	return cookie
}

// fetchStockCodeMap fetch admin HTML và parse map[productID]stockCode.
// Pattern trong HTML: product-stock&code=X ngay trước product-edit&id=Y (cùng <td>).
// stripPHPSessionID loại bỏ PHPSESSID khỏi cookie string.
// PHPSESSID ngắn hạn: khi hết hạn sẽ block admin_login (persistent token).
// Chỉ gửi admin_login + user_login là đủ để authenticate.
func stripPHPSessionID(cookie string) string {
	var parts []string
	for part := range strings.SplitSeq(cookie, ";") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(strings.ToLower(part), "phpsessid=") {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "; ")
}

func fetchStockCodeMap(ctx context.Context, adminCookie string) map[string]string {
	products := fetchAdminProductList(ctx, adminCookie)
	if len(products) == 0 {
		return nil
	}
	result := make(map[string]string, len(products))
	for _, p := range products {
		if p.Code != "" {
			result[p.ID] = p.Code
		}
	}
	return result
}

// fetchAdminProductList scrape FULL danh sách product từ admin HTML page.
// Dùng khi API /api/products.php bị giới hạn quyền (API key chỉ trả 1 phần).
// Trả về list rỗng nếu không có cookie hoặc fetch fail.
func fetchAdminProductList(ctx context.Context, adminCookie string) []BancloneProduct {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(adminCookie) == "" {
		return nil
	}
	cleanCookie := stripPHPSessionID(adminCookie)
	if strings.TrimSpace(cleanCookie) == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://banclone.pro/?module=admin&action=products", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Cookie", cleanCookie)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "vi,en;q=0.9")
	req.Header.Set("Cache-Control", "max-age=0")
	req.Header.Set("Referer", "https://banclone.pro/?module=admin&action=home")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/147.0.0.0 Safari/537.36")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-User", "?1")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 2<<20)

	// Two-step parse: tìm các "block" bắt đầu từ stock_code, sau đó parse từng field trong block.
	// Single mega-regex hay fail vì whitespace giữa field (~2700 chars tổng) > limit {0,N}.
	bodyStr := string(body)
	stockRe := regexp.MustCompile(`product-stock&(?:amp;)?code=([a-zA-Z0-9]+)`)
	idRe := regexp.MustCompile(`product-edit&(?:amp;)?id=(\d+)`)
	catRe := regexp.MustCompile(`bg-primary">\s*([^<]+?)\s*</span>`)
	priceRe := regexp.MustCompile(`Giá bán:[\s\S]{0,200}?<b[^>]*>\s*([^<]+?)\s*</b>`)
	amountRe := regexp.MustCompile(`bg-success-transparent">\s*(\d+)`)

	anchors := stockRe.FindAllStringSubmatchIndex(bodyStr, -1)
	var result []BancloneProduct
	seen := make(map[string]bool)
	for i, idx := range anchors {
		start := idx[0]
		end := len(bodyStr)
		if i+1 < len(anchors) {
			end = anchors[i+1][0]
		}
		block := bodyStr[start:end]
		code := bodyStr[idx[2]:idx[3]]

		idM := idRe.FindStringSubmatch(block)
		if len(idM) < 2 {
			continue
		}
		id := idM[1]
		if seen[id] {
			continue
		}
		seen[id] = true

		var name string
		if nm := regexp.MustCompile(`product-edit&(?:amp;)?id=` + id + `">\s*([^<]+?)\s*</a>`).
			FindStringSubmatch(block); len(nm) >= 2 {
			name = nm[1]
		}
		var category string
		if cm := catRe.FindStringSubmatch(block); len(cm) >= 2 {
			category = cm[1]
		}
		var price string
		if pm := priceRe.FindStringSubmatch(block); len(pm) >= 2 {
			price = strings.TrimSuffix(strings.TrimSpace(pm[1]), "đ")
		}
		amount := 0
		if am := amountRe.FindStringSubmatch(block); len(am) >= 2 {
			fmt.Sscanf(am[1], "%d", &amount)
		}

		result = append(result, BancloneProduct{
			ID:           id,
			Code:         code,
			Name:         name,
			CategoryName: category,
			Price:        price,
			Amount:       amount,
		})
	}
	return result
}

// GetBancloneProducts lấy danh sách sản phẩm từ banclone.pro/api/products.php.
// adminUsername+adminPassword (tuỳ chọn): nếu có, tự login lấy cookie → fetch stock code thật.
// Trả về JSON array BancloneProduct hoặc chuỗi lỗi "ERR|..."
func (a *App) GetBancloneProducts(apiKey, adminUsername, adminPassword string) string {
	if strings.TrimSpace(apiKey) == "" {
		return `ERR|API key trống`
	}
	apiURL := "https://banclone.pro/api/products.php?api_key=" + url.QueryEscape(apiKey)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return "ERR|Kết nối thất bại: " + err.Error()
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 512*1024)

	var raw struct {
		Status     string `json:"status"`
		Msg        string `json:"msg"`
		Categories []struct {
			Name     string `json:"name"`
			Products []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Price  string `json:"price"`
				Amount int    `json:"amount"`
			} `json:"products"`
		} `json:"categories"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "ERR|Parse JSON thất bại: " + err.Error()
	}
	if raw.Status != "success" {
		return "ERR|" + raw.Msg
	}

	var adminCookie string
	if strings.TrimSpace(adminUsername) != "" && strings.TrimSpace(adminPassword) != "" {
		if c, err := bancloneAdminLogin(a.ctx, adminUsername, adminPassword); err == nil {
			adminCookie = c
		}
	}

	// PRIORITY: nếu có admin cookie → scrape full từ HTML admin (không bị giới hạn API key).
	// API products.php có thể bị giới hạn quyền → chỉ trả 1 phần. Admin HTML trả full.
	adminProducts := fetchAdminProductList(a.ctx, adminCookie)
	if len(adminProducts) > 0 {
		out, _ := json.Marshal(adminProducts)
		return string(out)
	}

	// FALLBACK: không có admin cookie hoặc scrape fail → dùng API products.php
	// (sẽ giới hạn theo quyền API key)
	stockMap := fetchStockCodeMap(a.ctx, adminCookie)
	var products []BancloneProduct
	for _, cat := range raw.Categories {
		for _, p := range cat.Products {
			products = append(products, BancloneProduct{
				ID:           p.ID,
				Code:         stockMap[p.ID], // rỗng nếu không có admin cookie
				Name:         p.Name,
				CategoryName: cat.Name,
				Price:        p.Price,
				Amount:       p.Amount,
			})
		}
	}

	out, _ := json.Marshal(products)
	return string(out)
}
