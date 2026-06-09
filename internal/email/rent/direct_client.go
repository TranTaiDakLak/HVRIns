// Package rent — shared direct HTTP client dùng cho các rent mail provider
// (dongvanfb, mail30s, muamail, store1s, zeus_x, unlimitmail, wmemail).
//
// Các provider này gọi trực tiếp API không qua proxy → cần 1 client singleton
// thay vì tạo &http.Client{} mỗi request. Tránh leak Transport + idle TCP conns
// khi chạy 24/7 với throughput cao (~50-100 requests/s).
package rent

import (
	"crypto/tls"
	"net/http"
	"time"
)

var directTransport = &http.Transport{
	TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	MaxIdleConns:          200,
	MaxIdleConnsPerHost:   50,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   8 * time.Second,
	ResponseHeaderTimeout: 10 * time.Second,
	DisableKeepAlives:     false,
}

// DirectClient singleton dùng cho tất cả rent mail API call trực tiếp (không qua proxy).
// Timeout per-request được set qua context.WithTimeout, không set trên Client.Timeout
// để 1 client dùng chung cho mọi endpoint khác timeout needs.
var DirectClient = &http.Client{
	Transport: directTransport,
	Timeout:   10 * time.Second,
}
