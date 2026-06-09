// Package clonehv — Client API cho clonehv.com
// API mua tài khoản Facebook để verify
package clonehv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"HVRIns/internal/httpx"
)

const baseURL = "https://clonehv.com/api"

// ProductInfo thông tin sản phẩm
type ProductInfo struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Price       int    `json:"price"`
	Amount      string `json:"amount"`
	Country     string `json:"country"`
	Description string `json:"description"`
}

type infoResponse struct {
	Status string      `json:"status"`
	Msg    string      `json:"msg"`
	Data   ProductInfo `json:"data"`
}

type buyItem struct {
	Account string `json:"account"`
}

// buyResponse hỗ trợ cả 2 format CloneHV trả về (C# parity):
//
//  1. {status, data:{trans_id, lists:[{account:...}]}}        — format mặc định
//  2. {status, transid, accounts:["uid|pass|cookie|..."]}      — top-level array (fallback)
//
// Extract trans_id: ưu tiên data.trans_id, fallback transid top-level.
type buyResponse struct {
	Status   string   `json:"status"`
	Msg      string   `json:"msg"`
	TransID  string   `json:"transid"`  // top-level fallback (format 2)
	Accounts []string `json:"accounts"` // top-level fallback (format 2)
	Data     struct {
		TransID  string    `json:"trans_id"` // chính (format 1)
		Category string    `json:"category"`
		Amount   int       `json:"amount"`
		Lists    []buyItem `json:"lists"` // format 1
	} `json:"data"`
}

var httpClient = &http.Client{Timeout: 30 * time.Second}

// GetProductInfo lấy thông tin chi tiết một sản phẩm từ CloneHV API
// (tồn kho, giá, tên, quốc gia...).
//
// ctx: context để timeout hoặc cancel request; nên đặt deadline hợp lý
// vì API call này chặn goroutine đang gọi.
//
// username, password: thông tin đăng nhập tài khoản CloneHV của người dùng.
// Cả hai đều được escape bằng url.QueryEscape trước khi ghép vào URL để
// tránh lỗi khi credentials chứa ký tự đặc biệt (@, +, v.v.).
//
// productID: mã sản phẩm cần tra cứu trên CloneHV (ví dụ "1234"). Cũng
// được escape để an toàn với các ID có ký tự phi ASCII.
//
// Nếu API trả về status != "success", hàm trả về error với nội dung
// result.Msg. Nếu body không parse được JSON, error kèm snippet tối đa
// 200 ký tự đầu của body để dễ debug mà không log toàn bộ response lớn.
func GetProductInfo(ctx context.Context, username, password, productID string) (*ProductInfo, error) {
	apiURL := fmt.Sprintf("%s/InfoResource.php?username=%s&password=%s&id=%s",
		baseURL,
		url.QueryEscape(username),
		url.QueryEscape(password),
		url.QueryEscape(productID),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := httpx.ReadBody(resp.Body, 64*1024)
	if err != nil {
		return nil, err
	}

	var result infoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse error: %v — body: %s", err, string(body)[:min(200, len(body))])
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("API lỗi: %s", result.Msg)
	}
	return &result.Data, nil
}

// BuyAccounts mua tài khoản Facebook từ CloneHV và trả về danh sách
// chuỗi account raw (định dạng do CloneHV quy định, thường là
// "cookie|uid|token|..." hoặc tương tự tùy sản phẩm).
//
// ctx: context để timeout hoặc cancel — API mua có thể chậm hơn info query
// vì CloneHV cần xử lý giao dịch phía server.
//
// username, password: thông tin đăng nhập CloneHV, được url.QueryEscape
// trước khi ghép URL.
//
// productID: mã sản phẩm cần mua, phải khớp với productID đã lấy từ
// GetProductInfo.
//
// amount: số lượng tài khoản cần mua trong một lần gọi. Nên đặt bằng
// batchSize của CredPool để tránh gọi API lẻ tẻ.
//
// Kết quả trả về: slice chuỗi account raw, bỏ qua các item có Account rỗng.
// Format chuỗi raw phụ thuộc loại sản phẩm CloneHV — caller tự parse.
// Nếu parse lỗi JSON, error kèm snippet <= 200 ký tự body để debug nhanh.
func BuyAccounts(ctx context.Context, username, password, productID string, amount int) ([]string, error) {
	apiURL := fmt.Sprintf("%s/BResource.php?username=%s&password=%s&id=%s&amount=%d",
		baseURL,
		url.QueryEscape(username),
		url.QueryEscape(password),
		url.QueryEscape(productID),
		amount,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := httpx.ReadBody(resp.Body, 256*1024)
	if err != nil {
		return nil, err
	}

	var result buyResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse error: %v — body: %s", err, string(body)[:min(200, len(body))])
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("API lỗi: %s", result.Msg)
	}

	// Port C# FetchBulkAccountsFromBResource: hỗ trợ 2 response formats.
	// Ưu tiên format 1 (data.lists), fallback format 2 (top-level accounts).
	accounts := make([]string, 0)
	for _, item := range result.Data.Lists {
		if item.Account != "" {
			accounts = append(accounts, item.Account)
		}
	}
	if len(accounts) == 0 {
		for _, acc := range result.Accounts {
			if acc != "" {
				accounts = append(accounts, acc)
			}
		}
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("API trả về 0 accounts (status=success nhưng lists/accounts rỗng)")
	}

	// Trans_id: ưu tiên data.trans_id, fallback transid top-level (match C#).
	transID := result.Data.TransID
	if transID == "" {
		transID = result.TransID
	}

	// Xoá đơn hàng async sau khi lấy accounts thành công (C# parity).
	// Không block return — DeleteOrder có lỗi cũng OK, best-effort cleanup.
	if transID != "" {
		go deleteOrder(username, password, productID, transID)
	}

	return accounts, nil
}

// deleteOrder xoá đơn hàng khỏi CloneHV sau khi đã lấy accounts.
// Fire-and-forget: không return error, không log. Timeout 15s.
// Port C# FMain.cs L2433-2446: DeleteOrder.php?username=X&password=Y&id=PID&trans_id=TID.
func deleteOrder(username, password, productID, transID string) {
	deleteURL := fmt.Sprintf("%s/DeleteOrder.php?username=%s&password=%s&id=%s&trans_id=%s",
		baseURL,
		url.QueryEscape(username),
		url.QueryEscape(password),
		url.QueryEscape(productID),
		url.QueryEscape(transID),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", deleteURL, nil)
	if err != nil {
		return
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body) // drain để TCP connection reusable
}

// min trả về giá trị nhỏ hơn giữa a và b.
//
// a, b: hai số nguyên cần so sánh; không giới hạn phạm vi.
//
// Dùng để tính độ dài snippet an toàn khi format error message sau lỗi
// JSON parse: string(body)[:min(200, len(body))] đảm bảo không bao giờ
// index out of range khi body ngắn hơn 200 byte.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
