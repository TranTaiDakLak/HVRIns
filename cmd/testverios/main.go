// cmd/testverios — test ver iOS Mess (iosmess) từ file NVR, login-first flow.
// Chạy: go run ./cmd/testverios <SuccessReg.txt> <so_account> [proxy]
package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/instagram"
	_ "HVRIns/internal/instagram/verify/ios/iosmess" // đăng ký verifier "iosmess"
)

const (
	mailProvider = "mailhv"
	mailToken    = "wm_live_7934e1616a40efe21e623ca2871f3caf5748c2c294c187b8daad149fb2fc47fd"
	mailDomain   = "i2b.vn"
)

var reDatr = regexp.MustCompile(`datr=([^;]+)`)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run ./cmd/testverios <SuccessReg.txt> <count> [proxy host:port:user:pass]")
		return
	}
	path, count := os.Args[1], os.Args[2]
	proxy := "unlimited.iprocket.io:12000:USERt1mbtV-zone-custom:Havu1988"
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

	startIdx := 0
	fmt.Sscanf(os.Getenv("START_INDEX"), "%d", &startIdx)
	seen, done := 0, 0

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
		if len(f) < 3 {
			continue
		}
		uid, pass, cookie := f[0], f[1], f[2]
		if len(uid) < 10 || uid == "2147483647" {
			continue
		}
		datr := ""
		if m := reDatr.FindStringSubmatch(cookie); len(m) > 1 {
			datr = m[1]
		}

		fmt.Printf("\n========== #%d  uid=%s ==========\n", done+1, uid)
		session := &instagram.Session{
			UID:      uid,
			Password: pass,
			Cookie:   cookie,
			Datr:     datr,
			Proxy:    proxy,
			// SessionlessCryptedUID rỗng → steps.go sẽ login-first → lấy EAAG + cryptedUID
		}

		cfg := &instagram.VerifyConfig{
			VerifyEnabled:  true,
			MailProvider:   mailProvider,
			TempMailToken:  mailToken,
			TempMailDomain: mailDomain,
			CheckLiveDie:   true,
			TimeDelayCheck: 8,
		}

		verifier, err := instagram.NewVerifier("iosmess")
		if err != nil {
			fmt.Printf("NewVerifier: %v\n", err)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
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
