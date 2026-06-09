// cmd/regtest/main.go — Test thực tế từng bước đăng ký Facebook
// Chạy: go run ./cmd/regtest/
// Debug r.php: go run ./cmd/regtest/ --debug-rphp
// Với proxy:   go run ./cmd/regtest/ host:port:user:pass
package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/instagram"
	webregister "HVRIns/internal/instagram/register/web"
)

func main() {
	proxy := ""
	args := os.Args[1:]

	if len(args) > 0 && args[0] == "--debug-rphp" {
		debugRphp("")
		return
	}
	if len(args) > 0 {
		proxy = args[0]
	}

	debugDir := "reg_debug"
	_ = os.MkdirAll(debugDir, 0755)

	fmt.Println("=== Facebook Registration Test — Verbose Mode ===")
	fmt.Printf("Proxy   : %q\n", proxy)
	fmt.Printf("Debug   : responses saved to ./%s/\n\n", debugDir)

	input := webregister.RandomRegInput("", "", proxy)
	input.DebugDir = debugDir
	fmt.Printf("Name    : %s %s\n", input.FirstName, input.LastName)
	fmt.Printf("Birthday: %s\n", input.Birthday)
	fmt.Printf("Gender  : %d\n", input.Gender)
	fmt.Printf("Phone   : %s\n", input.Phone)
	fmt.Printf("Password: %s\n\n", input.Password)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	reg, err := instagram.NewRegisterer("web")
	if err != nil {
		fmt.Printf("❌ NewRegisterer: %v\n", err)
		return
	}
	result := reg.Register(ctx, &input, func(msg string) {
		fmt.Println("[LOG]", msg)
	})

	fmt.Println()
	if result.Success {
		fmt.Printf("✅ THÀNH CÔNG — UID=%s\n", result.UID)
		fmt.Println("   ", result.Message)
	} else {
		fmt.Printf("❌ THẤT BẠI — %s\n", result.Message)
	}

	fmt.Println()
	analyzeDebugFiles(debugDir)
}

func analyzeDebugFiles(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	if len(entries) == 0 {
		return
	}
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("PHÂN TÍCH RESPONSES ĐÃ LƯU")
	fmt.Println("═══════════════════════════════════════════")

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		data, err := os.ReadFile(dir + "/" + name)
		if err != nil {
			continue
		}
		resp := string(data)
		step := strings.TrimSuffix(name, ".txt")
		fmt.Printf("\n[%s] %d bytes\n", step, len(resp))
		analyzeBloksResponse(step, resp)

		if step == "B8" {
			fmt.Println("  ── B8 snippet (1500 chars) ──")
			s := resp
			if len(s) > 1500 {
				s = s[:1500] + "..."
			}
			fmt.Println(" ", s)
		}
	}
}

func analyzeBloksResponse(step, resp string) {
	checks := []struct {
		label, pattern string
	}{
		{"reg_info", `"reg_info"`},
		{"reg_context", `"reg_context"`},
		{"currentUser", `"currentUser"\s*:\s*\d{10,}`},
		{"UID", `"uid"\s*:\s*"?\d+`},
		{"user_id", `"user_id"\s*:\s*"?\d+`},
		{"c_user cookie", `c_user=\d+`},
		{"errorSummary", `"errorSummary"\s*:\s*"[^"]+"`},
		{"registration_error", `registration_error`},
		{"phone_number_used", `phone_number_used`},
		{"confirmation_required", `confirmation_required`},
		{"confirm_phone", `confirm_phone|phone_confirm|verify_phone`},
		{"send_code", `send_code|sendCode`},
		{"account_created", `account_created|signup_success`},
		{"access_token", `access_token`},
		{"login_token", `login_token`},
		{"session_cookies", `session_key|session_secret`},
	}

	for _, c := range checks {
		re, err := regexp.Compile(`(?i)` + c.pattern)
		if err != nil {
			continue
		}
		m := re.FindString(resp)
		if m != "" {
			ctx := extractContext(resp, m, 100)
			fmt.Printf("  ✅ %-28s → ...%s...\n", c.label, ctx)
		}
	}

	reKey := regexp.MustCompile(`CAA_[A-Z_]+`)
	keys := uniqueStrings(reKey.FindAllString(resp, -1))
	if len(keys) > 0 {
		fmt.Printf("  🔑 CAA_keys: %s\n", strings.Join(keys, ", "))
	}

	reUID := regexp.MustCompile(`"(?:currentUser|uid|user_id)"\s*:\s*"?(\d{8,})`)
	uids := reUID.FindAllStringSubmatch(resp, 5)
	for _, u := range uids {
		if len(u) >= 2 {
			fmt.Printf("  🎯 UID: %s\n", u[1])
		}
	}

	reToken := regexp.MustCompile(`EAAB[A-Za-z0-9]{20,}`)
	tokens := reToken.FindAllString(resp, 3)
	for _, t := range tokens {
		fmt.Printf("  🔐 Token: %s...\n", t[:min(len(t), 40)])
	}
}

func extractContext(resp, match string, n int) string {
	idx := strings.Index(resp, match)
	if idx < 0 {
		return match
	}
	start := idx - 20
	if start < 0 {
		start = 0
	}
	end := idx + len(match) + n
	if end > len(resp) {
		end = len(resp)
	}
	s := strings.ReplaceAll(resp[start:end], "\n", " ")
	if len(s) > 140 {
		s = s[:140]
	}
	return s
}

func uniqueStrings(ss []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func debugRphp(proxy string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, urlInfo := range []struct{ label, url string }{
		{"r.php", "https://m.facebook.com/r.php"},
		{"login", "https://m.facebook.com/login/"},
	} {
		fmt.Printf("\n=== Debug %s ===\n", urlInfo.label)
		html, err := webregister.FetchRegHTMLFull(ctx, proxy, "", urlInfo.url)
		if err != nil {
			fmt.Println("ERROR:", err)
			continue
		}
		fname := "debug_" + urlInfo.label + ".html"
		_ = os.WriteFile(fname, []byte(html), 0644)
		fmt.Printf("%d bytes → %s\n\n", len(html), fname)
		searchKey(html)
	}
}

func searchKey(html string) {
	for _, p := range []struct{ name, pat string }{
		{"publicKey", `"publicKey"\s*:\s*"([^"]+)"`},
		{"public_key", `"public_key"\s*:\s*"([^"]+)"`},
		{"keyId", `"keyId"\s*:\s*(\d+)`},
	} {
		re, _ := regexp.Compile(p.pat)
		m := re.FindStringSubmatch(html)
		if len(m) >= 2 {
			v := m[1]
			if len(v) > 80 {
				v = v[:80] + "..."
			}
			fmt.Printf("  ✅ [%s] = %q\n", p.name, v)
		} else {
			fmt.Printf("  ✗  [%s] not found\n", p.name)
		}
	}
}
