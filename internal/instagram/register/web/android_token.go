// android_token.go — Android HTTP helper + extract token regex.
//
// Trước đây chứa FetchAndroidToken/FetchAndroidTokenVerbose (RSA flow với
// b-graph.facebook.com/graphql + Bloks payload). Đã verify FAIL với FB schema hiện
// hành (field_exception 1675030). Removed 2026-05-15.
//
// Production flow lấy EAA token hiện tại dùng FetchAndroidTokenLegacy ở
// android_token_legacy.go — POST `/auth/login` REST classic, stable, không phụ thuộc
// Bloks/GraphQL schema rotation. Verified 8/8 NVR WebAndroid accounts thành công.
//
// File này giữ lại helper functions dùng chung:
//   - androidUA: default FB4A UA (fallback khi caller không pass UA)
//   - doAndroidHTTP: POST với Android headers + gzip auto-decompress
package web

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"HVRIns/internal/proxy"
)

// androidUA — default FB4A native UA (match WeBM hardcoded format).
//   - FBAV/518.0.0.63.86 (production), FBSV/9 (Android 9), SM-G998B (Samsung Note 21)
//   - KHÔNG có Mozilla/AppleWebKit/Chrome prefix (đó là WebView UA, sai cho native API)
//   - Caller bảo đảm UA này hoặc pass UA riêng (FetchAndroidTokenLegacy auto-fallback)
const androidUA = "[FBAN/FB4A;FBAV/518.0.0.63.86;FBBV/750617200;FBDM/{density=3.0,width=1080,height=1920};FBLC/en_US;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-G998B;FBSV/9;FBOP/1;FBCA/x86_64:arm64-v8a;]"

// doAndroidHTTP — POST request với Android headers + auto gzip decompress.
//
// Dùng bởi FetchAndroidTokenLegacy (android_token_legacy.go) để POST /auth/login.
// Returns response body string (đã decompress gzip nếu cần).
//
// ctx:      cancel/timeout context.
// endpoint: URL đích (vd b-graph.facebook.com/auth/login).
// body:     form-urlencoded body string (đã build sẵn).
// headers:  custom headers map (caller chuẩn bị Android-specific).
// proxyStr: proxy "ip:port" hoặc "ip:port:user:pass".
func doAndroidHTTP(ctx context.Context, endpoint, body string, headers map[string]string, proxyStr string) (string, error) {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		if gz, gerr := gzip.NewReader(resp.Body); gerr == nil {
			defer gz.Close()
			reader = gz
		}
	}
	b, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
