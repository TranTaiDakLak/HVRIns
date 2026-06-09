// otp_smail1s.go — Standalone Hotmail OAuth2 OTP reader qua smail1s.com.
// Tách từ store1s.fetchOTPSmail1s thành function chia sẻ — cho phép các provider
// Hotmail OAuth2 khác (zeus-x, dongvanfb, mail30s, muamail, unlimitmail, wmemail)
// dùng làm primary hoặc fallback reader thông qua otp_reader.go priority helper.
//
// Request: POST https://smail1s.com/get_messages
//
//	{"data": "email|pass|refresh_token|client_id", "mode": "oauth"}
//
// QUAN TRỌNG: phải dùng mode="oauth" (IMAP), KHÔNG dùng "graph".
// mode="graph" trả về inbox RỖNG (đã verify) dù hộp thư có mail → không bao giờ thấy OTP.
package rent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"HVRIns/internal/email/temp"
)

// readOTPViaSmail1s đọc OTP của mailbox OAuth2 Hotmail qua endpoint smail1s.com.
// Tốc độ trung bình ~4.5s/lần (so với ~2.6s của dongvanfb).
// Trả ("", nil) khi inbox rỗng (chưa có OTP); trả ("", err) khi lỗi mạng/parse.
//
// client: caller truyền HTTP client (có thể có proxy); fallback DirectClient (singleton)
// khi caller truyền nil — để tương thích với các provider gọi không qua proxy.
func readOTPViaSmail1s(ctx context.Context, email, pass, refreshToken, clientID string, client *http.Client) (string, error) {
	data := fmt.Sprintf("%s|%s|%s|%s", email, pass, refreshToken, clientID)
	payload := map[string]string{
		"data": data,
		"mode": "oauth",
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("smail1s marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://smail1s.com/get_messages",
		bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("smail1s create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://smail1s.com")
	req.Header.Set("Referer", "https://smail1s.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")

	httpClient := client
	if httpClient == nil {
		httpClient = DirectClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("smail1s get_code: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var result smail1sResp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("smail1s parse: %w — body: %.200s", err, respBody)
	}

	for _, item := range result.Data {
		// Ưu tiên Code field (server đã extract sẵn)
		for _, msg := range item.Messages {
			if code := strings.TrimSpace(msg.Code); code != "" {
				return code, nil
			}
		}
		// Fallback: parse Message body để extract OTP từ sender Facebook
		for _, msg := range item.Messages {
			if msg.Message == "" {
				continue
			}
			fromLower := strings.ToLower(msg.From)
			isFb := strings.Contains(fromLower, "facebookmail.com") ||
				strings.Contains(fromLower, "account.meta.com") ||
				strings.Contains(fromLower, "instagram")
			if !isFb {
				continue
			}
			if code := temp.ExtractCode(msg.Message); code != "" {
				return code, nil
			}
		}
	}

	return "", nil
}
