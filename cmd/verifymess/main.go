// cmd/verifymess — test ver Mess Android (appmessv3) trên account reg sẵn (đọc SuccessReg.txt).
// Mục đích: chẩn đoán bug reg-mail KHÔNG ver được (reg-phone ver OK).
//
// Chạy: go run ./cmd/verifymess <file_SuccessReg.txt> <so_account> [proxy]
//   proxy dạng host:port:user:pass (tuỳ chọn — account NVR thường cần residential proxy để login).
package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/instagram"
	_ "HVRIns/internal/instagram/verify/android/appmessv3" // đăng ký verifier "appmessv3"
)

const (
	mailProvider = "mailhv"
	mailToken    = "wm_live_7934e1616a40efe21e623ca2871f3caf5748c2c294c187b8daad149fb2fc47fd"
	mailDomain   = "i2b.vn"
)

var reDatr = regexp.MustCompile(`datr=([^;]+)`)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run ./cmd/verifymess <SuccessReg.txt> <count> [proxy host:port:user:pass]")
		return
	}
	path, count := os.Args[1], os.Args[2]
	proxy := ""
	if len(os.Args) >= 4 {
		proxy = os.Args[3]
	}
	n := 1
	fmt.Sscanf(count, "%d", &n)

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("read file: %v\n", err)
		return
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")

	// START_INDEX: bỏ qua N account đầu (để mỗi biến thể experiment dùng account FRESH khác nhau).
	startIdx := 0
	fmt.Sscanf(os.Getenv("START_INDEX"), "%d", &startIdx)
	seen := 0

	done := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if seen < startIdx {
			seen++
			continue
		}
		f := strings.Split(line, "|")
		if len(f) < 4 {
			continue
		}
		uid, pass, cookie, token := f[0], f[1], f[2], f[3]
		datr := ""
		if m := reDatr.FindStringSubmatch(cookie); len(m) > 1 {
			datr = m[1]
		}

		// FRESH_TOKEN=1 → bỏ token EAAAAU cũ trong file (có thể hết hạn) → ver tự fetch token
		// tươi qua REST /auth/login (uid+pass+datr) như app. Tránh attestation do token stale.
		if os.Getenv("FRESH_TOKEN") == "1" {
			token = ""
		}
		fmt.Printf("\n========== ACCOUNT #%d  uid=%s ==========\n", done+1, uid)
		session := &instagram.Session{
			UID: uid, Password: pass, Cookie: cookie, Token: token,
			Datr: datr, Proxy: proxy, InputAccount: line,
		}
		session.UserAgent = instagram.DefaultUserAgent()

		cfg := &instagram.VerifyConfig{
			VerifyEnabled:  true,
			MailProvider:   mailProvider,
			TempMailToken:  mailToken,
			TempMailDomain: mailDomain,
			CheckLiveDie:   true,
			TimeDelayCheck: 8,
		}

		verPlat := os.Getenv("VER_PLATFORM")
		if verPlat == "" {
			verPlat = "appmessv3"
		}
		verifier, err := instagram.NewVerifier(verPlat)
		if err != nil {
			fmt.Printf("NewVerifier: %v\n", err)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
		result := verifier.Verify(ctx, session, cfg, "", func(u, msg string) {
			fmt.Printf("  [%s] %s\n", u, msg)
		})
		cancel()

		if result != nil {
			fmt.Printf(">>> KẾT QUẢ #%d: status=%s success=%t msg=%s email=%s\n",
				done+1, result.Status, result.Success, result.Message, result.Email)
		}
		done++
		if done >= n {
			break
		}
	}
	fmt.Printf("\n=== Xong %d account ===\n", done)
}
