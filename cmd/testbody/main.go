package main

import (
	"fmt"
	"regexp"

	"HVRIns/internal/proxy"
)

func main() {
	// Test với 2 pattern format khác nhau
	p1 := "proxy.proxyshare.com:5960:ps-k3b0n3zt2nnu_session-V9VVWKZLZL_life-10:GyvGaHqRnysbZudg"
	p2 := "proxy.proxyshare.com:5960:ps-k3b0n3zt2nnu_area-IN_session-V9VVWKZLZL_life-10:GyvGaHqRnysbZudg"
	p3 := "proxy.proxyshare.com:5960:ps-k3b0n3zt2nnu_area-IN-session-session_life-10:GyvGaHqRnysbZudg"

	re := regexp.MustCompile(`(?i)(_session-)([A-Za-z0-9]+)`)

	for i, p := range []string{p1, p2, p3} {
		fmt.Printf("\n=== Test %d ===\n", i+1)
		fmt.Printf("Input: %s\n", p)
		// Extract user part
		parts := []string{}
		for _, s := range []string{":"} {
			_ = s
		}
		_ = parts

		rendered := proxy.RenderSessionIfIsProxyServer(p)
		fmt.Printf("After RenderSession: %s\n", rendered)
		fmt.Printf("Same? %v\n", p == rendered)

		// Test regex match on user part
		// user = parts[2]
		splitParts := []string{}
		for i, c := range p {
			if c == ':' {
				splitParts = append(splitParts, p[:i])
				p = p[i+1:]
				break
			}
			_ = i
		}
		_ = splitParts

		// Direct regex test on user component
		userOnly := "ps-k3b0n3zt2nnu_session-V9VVWKZLZL_life-10"
		if i == 1 {
			userOnly = "ps-k3b0n3zt2nnu_area-IN_session-V9VVWKZLZL_life-10"
		} else if i == 2 {
			userOnly = "ps-k3b0n3zt2nnu_area-IN-session-session_life-10"
		}
		match := re.FindStringSubmatch(userOnly)
		fmt.Printf("Regex match on user: %v\n", match)
	}
}
