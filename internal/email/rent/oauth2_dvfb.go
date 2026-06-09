// oauth2_dvfb.go — shared OAuth2 OTP reader qua tools.dongvanfb.net/api/get_code_oauth2.
// Dùng cho các provider Hotmail/Outlook OAuth2 đọc Facebook OTP
// từ cùng 1 mailbox Microsoft bằng refresh_token + client_id.
package rent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// readOTPViaDongvanfb gọi tools.dongvanfb.net/api/get_code_oauth2 và trả về Facebook OTP.
// c: HTTP client có proxy; nil = dùng DirectClient (không proxy).
// Trả về (code, nil) nếu tìm thấy; ("", nil) nếu chưa có code; ("", err) nếu lỗi mạng/parse.
func readOTPViaDongvanfb(ctx context.Context, email, pass, refreshToken, clientID string, c *http.Client) (string, error) {
	if c == nil {
		c = DirectClient
	}
	payload := map[string]string{
		"email":         email,
		"pass":          pass,
		"refresh_token": refreshToken,
		"client_id":     clientID,
		"type":          "facebook",
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://tools.dongvanfb.net/api/get_code_oauth2",
		bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://dongvanfb.net")
	req.Header.Set("Referer", "https://dongvanfb.net/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36")

	resp, err := c.Do(req)
	if err != nil {
		return "", fmt.Errorf("dvfb request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var result dvfbCodeResp
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("dvfb parse: %w", err)
	}

	if !result.Status {
		return "", nil
	}
	return strings.TrimSpace(result.Code), nil
}
