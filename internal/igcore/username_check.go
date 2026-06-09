// username_check.go — check live/die IG bằng username qua API dvpro.vn.
// Không cần cookie/session, chỉ cần username. Có rate limit nên cần delay giữa các call.
package igcore

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const dvproCheckURL = "https://tools.dvpro.vn/check_usname_instagram/index.php?action=checkOne"

var statusRe = regexp.MustCompile(`"status"\s*:\s*"([^"]+)"`)

// CheckUsernameStatus gọi API dvpro check 1 username.
// Trả: "live" | "die" | "unknown".
// Dùng http.Client thường (không qua proxy IG) vì đây là API bên thứ 3.
func CheckUsernameStatus(ctx context.Context, username string) string {
	username = strings.TrimSpace(username)
	if username == "" {
		return "unknown"
	}

	payload := fmt.Sprintf(`{"account":%q}`, username)
	req, err := http.NewRequestWithContext(ctx, "POST", dvproCheckURL,
		strings.NewReader(payload))
	if err != nil {
		return "unknown"
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://tools.dvpro.vn")
	req.Header.Set("Referer", "https://tools.dvpro.vn/check_usname_instagram/index.php")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "unknown"
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	return parseDvproStatus(string(body))
}

// parseDvproStatus bóc status từ JSON response của dvpro.
// {"username":"x","status":"live","time":"..."} → "live"
func parseDvproStatus(body string) string {
	m := statusRe.FindStringSubmatch(body)
	if len(m) < 2 {
		return "unknown"
	}
	switch strings.ToLower(m[1]) {
	case "live":
		return "live"
	case "die":
		return "die"
	default:
		return "unknown"
	}
}
