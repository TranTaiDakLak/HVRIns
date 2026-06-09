package main

import (
	"fmt"
	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/instagram/fakeinfo/uabuilder"
)

func main() {
	// Point uabuilder tới Config thật trong build/bin
	uabuilder.SetConfigBaseDir(`build/bin/Config`)

	fmt.Println("=== RandomChromeAndroidProfile (embedded data) — 5 mẫu ===")
	for i := 0; i < 5; i++ {
		p := fakeinfo.RandomChromeAndroidProfile()
		fmt.Printf("[%d] %s\n", i+1, p.UserAgent)
	}

	fmt.Println("\n=== BrowserUABuilder (Config/DeviceInfo) — 5 mẫu ===")
	b := &uabuilder.BrowserUABuilder{}
	for i := 0; i < 5; i++ {
		res, err := b.Build(uabuilder.UAOptions{})
		if err != nil {
			fmt.Println("ERROR:", err)
			continue
		}
		// Strip metadata suffix |model|os trước khi in
		ua := res.UserAgent
		if idx := len(ua); idx > 0 {
			for j := len(ua) - 1; j >= 0; j-- {
				if ua[j] == '|' {
					ua = ua[:j]
					break
				}
			}
		}
		fmt.Printf("[%d] %s\n", i+1, ua)
	}
}
