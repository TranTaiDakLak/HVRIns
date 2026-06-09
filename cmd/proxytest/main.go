package main

import (
	"fmt"

	"HVRIns/internal/proxy"
)

func main() {
	proxies := []string{
		"45.43.57.149:3000:wkiPYKNH2YRh-region-Random:pxzXAfvWgSmP",
		"global.unl.711proxy.com:12000:USER255485-zone-custom:Havu1988",
		"unlimited.iprocket.io:12000:USER796383:6ba611",
		"proxy.proxyshare.com:5959:ps-tnqxbfvf06nd_area-GB_session-sess001_life-15:7MxYRkK1IQkSHqbj",
	}

	fmt.Println("=== parseProxy() — URL format cho HTTP client ===")
	for _, p := range proxies {
		url := proxy.FormatProxyURL(p)
		fmt.Printf("IN:  %s\nOUT: %s\n\n", p, url)
	}

	fmt.Println("=== RenderSessionIfIsProxyServer() — rotate session ===")
	for _, p := range proxies {
		r1 := proxy.RenderSessionIfIsProxyServer(p)
		r2 := proxy.RenderSessionIfIsProxyServer(p)
		sameAsInput := r1 == p
		bothSame := r1 == r2
		fmt.Printf("IN:   %s\nOUT1: %s\nOUT2: %s\n(static: %v | rotate: %v)\n\n", p, r1, r2, sameAsInput, !bothSame)
	}
}
