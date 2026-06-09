// test_ua — diagnostic CLI để in UA generated cho mỗi platform.
// Chạy từ project root: `go run ./cmd/test_ua/`
//
// Hữu ích khi cần verify uabuilder đọc đúng Config/MobileDevices/<plat>_devices.txt
// + Config/Fbapp/versions_and_builds_<plat>.txt sau khi user edit.
package main

import (
	"fmt"

	"HVRIns/internal/instagram/fakeinfo/uabuilder"
)

func main() {
	for _, plat := range []string{"s22", "s23", "s24", "s25", "s26", "s555", "s556", "s557", "android", ""} {
		for _, virtual := range []bool{false, true} {
			res, err := (&uabuilder.AndroidUABuilder{}).Build(uabuilder.UAOptions{
				Platform:        plat,
				Locale:          "en_US",
				AddVirtualSpecs: virtual,
			})
			fmt.Printf("[%s | virtual=%v] err=%v\n  ua=%s\n\n",
				plat, virtual, err, res.UserAgent)
		}
	}
}
