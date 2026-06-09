// shared.go — Shared HTTP client dùng cho tất cả proxy provider API calls
package providers

import (
	"net/http"
	"time"
)

// apiClient — shared client dùng cho tất cả proxy provider API calls.
// Tránh tạo Transport mới mỗi lần gọi — tiết kiệm memory + TLS handshake overhead khi 100+ luồng.
var apiClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
	},
}
