package main

import (
	"fmt"
	"HVRIns/internal/proxy"
)

func main() {
	cases := []string{
		"us.proxyshare.com:5959:ps-k3b0n3zt2nnu_area-US_session-2HNUFU64BE_life-5:GyvGaHqRnysbZudg",
		"us.proxyshare.com:5959:ps-k3b0n3zt2nnu:GyvGaHqRnysbZudg",
		"y3b758257-region-US-sid-oldsid-t-15:pass@host.com:5959",
	}
	for _, p := range cases {
		fmt.Println("in: ", p)
		for i := 0; i < 3; i++ {
			fmt.Printf("  call %d: %s\n", i+1, proxy.RenderSessionIfIsProxyServer(p))
		}
		fmt.Println()
	}
}
