//go:build ignore

// debug.go — dump r.php và B1 response để tìm public key
// Chạy: go run ./cmd/regtest/debug.go
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"
)

func runDebug() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		Timeout:   20 * time.Second,
	}

	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/134.0.1911.158 Mobile/15E148 Safari/604.1"

	req, _ := http.NewRequestWithContext(ctx, "GET", "https://m.facebook.com/r.php", nil)
	req.Header.Set("User-Agent", ua)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("ERROR:", err)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	resp.Body.Close()
	html := string(body)

	_ = os.WriteFile("debug_rphp.html", body, 0644)
	fmt.Printf("r.php: HTTP %d, %d bytes → saved to debug_rphp.html\n", resp.StatusCode, len(body))

	// Tìm public_key với nhiều patterns khác nhau
	patterns := []string{
		`"public_key":"([^"]+)"`,
		`"public_key"\s*:\s*"([^"]+)"`,
		`public_key=([0-9a-fA-F]+)`,
		`encrypt\[public_key\]=([^\s&]+)`,
		`"key":"([0-9a-fA-F]{64})"`,
		`"pw_enc_pub_key"\s*:\s*"([^"]+)"`,
		`"pubKey"\s*:\s*"([^"]+)"`,
		`"encryptPassword"[^}]*"publicKey"\s*:\s*"([^"]+)"`,
	}
	fmt.Println("\n=== Public Key Search ===")
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		m := re.FindStringSubmatch(html)
		if len(m) >= 2 {
			fmt.Printf("FOUND with pattern %q:\n  value=%q (len=%d)\n", p, m[1][:min(m[1], 60)], len(m[1]))
		} else {
			fmt.Printf("NOT FOUND: %q\n", p)
		}
	}

	// Tìm key_id và version
	fmt.Println("\n=== Key ID / Version ===")
	for _, p := range []string{
		`"key_id"\s*:\s*"?(\d+)"?`,
		`"keyId"\s*:\s*"?(\d+)"?`,
		`"version"\s*:\s*"?(\d+)"?`,
	} {
		re := regexp.MustCompile(p)
		m := re.FindStringSubmatch(html)
		if len(m) >= 2 {
			fmt.Printf("  %q → %q\n", p, m[1])
		}
	}
}

func min(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
