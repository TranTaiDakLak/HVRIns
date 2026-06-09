// cmd/check_verified_email — kiểm tra acc trong SuccessVerify_No2FA.txt có thực sự
// verified email không (gọi Graph API /me?fields=id,email với token EAA của từng acc).
//
// Dùng: go run ./cmd/check_verified_email <path-to-SuccessVerify_No2FA.txt> [--proxy=ip:port:user:pass]
//
// Output:
//   - In ra từng acc với trạng thái: VERIFIED / NOT_VERIFIED / TOKEN_DEAD / ERROR
//   - Ghi 2 file cạnh input:
//       <input>.verified.txt    — acc có email thật trên FB (đáng tin)
//       <input>.suspicious.txt  — acc không có email / token chết / lỗi (cần xem lại)
//
// Format input (giống SuccessVerify_No2FA.txt):
//   UID|password|cookie|token|datetime|country
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type apiResp struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Error *struct {
		Code    int    `json:"code"`
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

type result struct {
	line       string
	uid        string
	status     string // VERIFIED | NOT_VERIFIED | TOKEN_DEAD | ERROR
	detail     string
}

func main() {
	proxyStr := flag.String("proxy", "", "Optional proxy ip:port hoặc ip:port:user:pass")
	concurrency := flag.Int("c", 5, "Số request song song")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Println("Usage: go run ./cmd/check_verified_email <SuccessVerify_No2FA.txt> [--proxy=...] [-c=5]")
		os.Exit(1)
	}
	inputPath := flag.Arg(0)

	f, err := os.Open(inputPath)
	if err != nil {
		fmt.Printf("Open input fail: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	verifiedPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".verified.txt"
	suspiciousPath := strings.TrimSuffix(inputPath, filepath.Ext(inputPath)) + ".suspicious.txt"

	verifiedFile, _ := os.Create(verifiedPath)
	defer verifiedFile.Close()
	suspiciousFile, _ := os.Create(suspiciousPath)
	defer suspiciousFile.Close()

	// HTTP client (optional proxy)
	tr := &http.Transport{}
	if *proxyStr != "" {
		if u := buildProxyURL(*proxyStr); u != nil {
			tr.Proxy = http.ProxyURL(u)
			fmt.Printf("[*] Using proxy: %s\n", u.String())
		}
	}
	client := &http.Client{Transport: tr, Timeout: 15 * time.Second}

	// Worker pool
	lines := make(chan string, 64)
	results := make(chan result, 64)
	var wg sync.WaitGroup
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for line := range lines {
				results <- checkAccount(client, line)
			}
		}()
	}

	// Reader goroutine
	go func() {
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 1024*1024), 8*1024*1024)
		for scanner.Scan() {
			s := strings.TrimSpace(scanner.Text())
			if s == "" {
				continue
			}
			lines <- s
		}
		close(lines)
		wg.Wait()
		close(results)
	}()

	// Collect results + counters
	var nVerified, nNotVerified, nTokenDead, nError, nTotal atomic.Int32
	for r := range results {
		nTotal.Add(1)
		switch r.status {
		case "VERIFIED":
			nVerified.Add(1)
			verifiedFile.WriteString(r.line + "\n")
		case "NOT_VERIFIED":
			nNotVerified.Add(1)
			suspiciousFile.WriteString(r.line + "\n")
		case "TOKEN_DEAD":
			nTokenDead.Add(1)
			suspiciousFile.WriteString(r.line + "\n")
		default:
			nError.Add(1)
			suspiciousFile.WriteString(r.line + "\n")
		}
		fmt.Printf("[%s] UID=%s %s\n", r.status, r.uid, r.detail)
	}

	fmt.Println("\n=== TỔNG KẾT ===")
	fmt.Printf("Total:        %d\n", nTotal.Load())
	fmt.Printf("VERIFIED:     %d  (thực sự có email) → %s\n", nVerified.Load(), verifiedPath)
	fmt.Printf("NOT_VERIFIED: %d  (KHÔNG có email — ghi nhầm!)\n", nNotVerified.Load())
	fmt.Printf("TOKEN_DEAD:   %d  (token chết, không kiểm được)\n", nTokenDead.Load())
	fmt.Printf("ERROR:        %d  (lỗi mạng/parse)\n", nError.Load())
	fmt.Printf("Suspicious:   %s\n", suspiciousPath)
}

// checkAccount parse 1 line, call Graph API /me?fields=id,email, trả status.
func checkAccount(client *http.Client, line string) result {
	parts := strings.Split(line, "|")
	if len(parts) < 4 {
		return result{line: line, status: "ERROR", detail: "line không đủ field (cần UID|pass|cookie|token|...)"}
	}
	uid := strings.TrimSpace(parts[0])
	token := strings.TrimSpace(parts[3])
	if !strings.HasPrefix(token, "EAA") {
		return result{line: line, uid: uid, status: "ERROR", detail: "token không có prefix EAA"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	// Graph API: /me?fields=id,email với access_token. FB-trusted endpoint, không cần Android UA.
	req, err := http.NewRequestWithContext(ctx, "GET",
		"https://graph.facebook.com/me?fields=id,email&access_token="+url.QueryEscape(token), nil)
	if err != nil {
		return result{line: line, uid: uid, status: "ERROR", detail: err.Error()}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 9) FB4A")

	resp, err := client.Do(req)
	if err != nil {
		return result{line: line, uid: uid, status: "ERROR", detail: "HTTP: " + err.Error()}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	var ar apiResp
	if jerr := json.Unmarshal(body, &ar); jerr != nil {
		return result{line: line, uid: uid, status: "ERROR", detail: "parse JSON: " + jerr.Error()}
	}

	if ar.Error != nil {
		// Token chết phổ biến: code 190 / type OAuthException
		if ar.Error.Code == 190 || strings.Contains(strings.ToLower(ar.Error.Message), "expired") ||
			strings.Contains(strings.ToLower(ar.Error.Message), "invalid") {
			return result{line: line, uid: uid, status: "TOKEN_DEAD",
				detail: fmt.Sprintf("FB err %d: %s", ar.Error.Code, truncate(ar.Error.Message, 80))}
		}
		return result{line: line, uid: uid, status: "ERROR",
			detail: fmt.Sprintf("FB err %d: %s", ar.Error.Code, truncate(ar.Error.Message, 80))}
	}

	if ar.Email != "" {
		return result{line: line, uid: uid, status: "VERIFIED", detail: "email=" + ar.Email}
	}
	return result{line: line, uid: uid, status: "NOT_VERIFIED", detail: "không có field email — confirm fail!"}
}

func buildProxyURL(proxyStr string) *url.URL {
	parts := strings.Split(proxyStr, ":")
	switch len(parts) {
	case 2:
		u, _ := url.Parse("http://" + proxyStr)
		return u
	case 4:
		u, _ := url.Parse(fmt.Sprintf("http://%s:%s@%s:%s", parts[2], parts[3], parts[0], parts[1]))
		return u
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
