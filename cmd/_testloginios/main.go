// Tạm: test flow login-first iOS Mess ver (LoginFull → EAAG + cryptedUID).
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	iosmess "HVRIns/internal/instagram/register/ios/iosmess"

	"github.com/google/uuid"
)

const proxy = "unlimited.iprocket.io:12000:USERt1mbtV-zone-custom:Havu1988"

var reDatr = regexp.MustCompile(`datr=([^;]+)`)

func main() {
	path := `e:\WEMAKE\NullCoreSummer\build\bin\result\result_multi9_20260608_095902\SuccessReg.txt`
	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")

	tested := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" { continue }
		f := strings.Split(line, "|")
		if len(f) < 3 { continue }
		uid, pass, cookie := f[0], f[1], f[2]
		if len(uid) < 10 || uid == "2147483647" { continue }
		m := reDatr.FindStringSubmatch(cookie)
		if len(m) < 2 { continue }
		datr := m[1]

		device := strings.ToUpper(uuid.New().String())
		family := strings.ToUpper(uuid.New().String())
		waterfall := uuid.New().String()

		tested++
		fmt.Printf("\n=== #%d uid=%s ===\n", tested, uid)
		t0 := time.Now()
		token, _, cryptedUID, err := iosmess.LoginFull(proxy, uid, pass, device, family, datr, waterfall, "")
		fmt.Printf("  elapsed: %v\n", time.Since(t0).Round(time.Millisecond))

		if err != nil || token == "" {
			fmt.Printf("  ❌ LOGIN FAIL: %v\n", err)
			if tested >= 5 { break }
			continue
		}

		fmt.Printf("  ✅ EAAG token    = %.35s...\n", token)
		if cryptedUID != "" {
			fmt.Printf("  ✅ crypted_uid   = %.50s...\n", cryptedUID)
		} else {
			fmt.Println("  ❌ crypted_uid   = KHÔNG có")
		}

		// Sinh fresh AAC + flow IDs
		aacJid, _, _ := iosmess.GenAACParts()
		fmt.Printf("  ✅ AAC jid       = %.16s...\n", aacJid)
		fmt.Printf("  Sẵn sàng ver: token=%t cryptedUID=%t aac=%t\n",
			token != "", cryptedUID != "", aacJid != "")

		if tested >= 3 { break }
	}
	fmt.Printf("\n=== Tested %d ===\n", tested)
}
