// uabuilder_wire.go — đăng ký ConfigFile UA pool source cho package uabuilder.
//
// Tách file riêng để init() chạy SAU init() của ua_pools.go (cùng package).
// Map per-platform → kind:
//   - iOS HTTP              → iOS_UG.txt    (UAKindIOS)
//   - Default (s23/android) → Android_UG.txt (UAKindAndroid)
//   - Request-based         → Request_UG.txt (UAKindRequest)
package fakeinfo

import (
	"HVRIns/internal/instagram/fakeinfo/uabuilder"
)

func init() {
	// Carrier picker — chọn operator theo quốc gia IP (VN→Viettel, US→T-Mobile, v.v.)
	uabuilder.SetCarrierPicker(func(countryCode string) string {
		return RandomSimProfile(countryCode).OperatorName
	})

	// Default fallback (key = "")
	uabuilder.SetConfigFileSource("", func() string { return RandomUAFromPool(UAKindAndroid) })

	// Android-family platforms — dùng pool Android_UG
	for _, p := range []string{"android", "s22", "s23", "s24", "s25", "s26", "s555", "s556", "s557"} {
		p := p
		uabuilder.SetConfigFileSource(p, func() string { return RandomUAFromPool(UAKindAndroid) })
	}

	// iOS HTTP — dùng pool iOS_UG
	uabuilder.SetConfigFileSource("ios", func() string { return RandomUAFromPool(UAKindIOS) })

	// WebAndroid — fallback về Android pool
	uabuilder.SetConfigFileSource("webandroid", func() string { return RandomUAFromPool(UAKindAndroid) })

	// Request-based (mfb)
	uabuilder.SetConfigFileSource("mfb", func() string { return RandomUAFromPool(UAKindRequest) })
	uabuilder.SetConfigFileSource("request", func() string { return RandomUAFromPool(UAKindRequest) })
}
