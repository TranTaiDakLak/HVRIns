// android.go — AndroidUABuilder port C# AndroidUserAgentBuilder.cs.
//
// Output FB4A native UA. Format:
//
//	[FBAN/FB4A;FBAV/<ver>;FBBV/<build>;FBDM={density=<d>,width=<w>,height=<h>};
//	 FBLC/<locale>;FBRV/0;FBCR/<carrier>;FBMF/<manufacturer>;FBBD/<brand>;
//	 FBPN/com.facebook.katana;FBDV/<model>;FBSV/<os>;FBOP/1;FBCA/<arch>]
//
// Khi AddVirtualSpecs=true, prepend:
//
//	Dalvik/2.1.0 (Linux; U; Android <os>; <model> Build/<buildID>) <FB_UG>
//
// buildID:
//   - "<Brand>-<Model>" (vd "samsung-SM-S911B")
//
// Lưu ý format chính xác (đối chiếu với C# AndroidUserAgentBuilder.cs L87-102):
//   - "FBDM={density=...}" KHÔNG phải "FBDM/{density=...}" (slash = bug version cũ)
//   - "FBMF/<manufacturer>" giữ nguyên capitalization từ devices file
//   - Đóng UA bằng "]" KHÔNG có ";]" ở cuối (C# old code có ";]" nhưng new code dùng "]")
package uabuilder

import (
	"fmt"
	"strings"
)

type AndroidUABuilder struct{}

func (b *AndroidUABuilder) Kind() UABuilderKind { return KindAndroidApp }

// Hardcode fallbacks — đảm bảo UA luôn build được dù Config/ files thiếu/lỗi.
// Match S23 baseline (bộ phổ biến nhất) — caller nào platform khác nên có Config riêng.
var fallbackAppVersions = []AppVersion{
	{Version: "554.0.0.57.70", Build: "918990560"},
	{Version: "555.0.0.49.59", Build: "920112839"},
	{Version: "556.1.0.63.64", Build: "942217461"},
}
var fallbackDevices = []DeviceSpec{
	{Manufacturer: "samsung", Brand: "samsung", Model: "SM-S911B", Width: 1080, Height: 2340, Density: "3.0", FBSS: "3"},
	{Manufacturer: "samsung", Brand: "samsung", Model: "SM-S918B", Width: 1440, Height: 3088, Density: "3.0", FBSS: "4"},
}

func (b *AndroidUABuilder) Build(opts UAOptions) (UABuildResult, error) {
	r := opts.NewRand()

	// 1. App version + build — pool source tuỳ PoolKind:
	//    "reg" → versions_and_builds_reg.txt (fallback chung)
	//    "ver" → versions_and_builds_ver.txt (fallback chung)
	//    ""    → versions_and_builds.txt (chung)
	var fbav, fbbv string
	if opts.PinAppVersion != "" && opts.PinBuild != "" {
		fbav = opts.PinAppVersion
		fbbv = opts.PinBuild
	} else {
		var appVersions []AppVersion
		var err error
		switch opts.PoolKind {
		case "reg":
			appVersions, err = LoadAppVersionsForReg()
		case "ver":
			appVersions, err = LoadAppVersionsForVer()
		default:
			appVersions, err = LoadAppVersionsForPlatform("")
		}
		if err != nil || len(appVersions) == 0 {
			appVersions = fallbackAppVersions
		}
		av := appVersions[r.Intn(len(appVersions))]
		fbav = av.Version
		fbbv = av.Build
	}

	// 2. Device pool (fallback hardcoded khi Config/ files thiếu)
	devices, _ := LoadDevicesForPlatform(opts.Platform)
	if samsungSPlatform(opts.Platform) {
		if filtered := filterSamsungSDevices(devices); len(filtered) > 0 {
			devices = filtered
		}
	} else if samsungG996Platform(opts.Platform) {
		if filtered := filterSamsungG996Devices(devices); len(filtered) > 0 {
			devices = filtered
		}
	}
	if len(devices) == 0 {
		devices = fallbackDevices
	}
	dev := devices[r.Intn(len(devices))]

	// 3. Density (từ device hoặc random từ densities.txt)
	density := dev.Density
	if density == "" {
		dList, err := loadDensities()
		if err == nil && len(dList) > 0 {
			density = pickRandom(r, dList)
		}
		if density == "" {
			density = "3.0"
		}
	}

	// 4. Resolution: device-specific nếu có, ngược lại random từ screen_resolution.txt
	width, height := dev.Width, dev.Height
	if width == 0 || height == 0 {
		resList, err := loadScreenResolutions()
		if err == nil && len(resList) > 0 {
			res := pickRandom(r, resList)
			parts := strings.Split(res, "x")
			if len(parts) == 2 {
				width = atoiSafe(parts[0])
				height = atoiSafe(parts[1])
			}
		}
		if width == 0 {
			width = 1080
		}
		if height == 0 {
			height = 2340
		}
	}

	// 5. Locale
	locale := opts.Locale
	if locale == "" {
		locale = "en_US"
	}

	// 6. Carrier: SimBrand override → picker theo quốc gia IP → carriers.txt random
	carrier := opts.SimBrand
	if carrier == "" {
		if picker := GetCarrierPicker(); picker != nil {
			carrier = picker(opts.CountryCode)
		}
	}
	if carrier == "" {
		cList, err := loadCarriers()
		if err == nil && len(cList) > 0 {
			carrier = pickRandom(r, cList)
		}
		if carrier == "" {
			carrier = "T-Mobile"
		}
	}

	// 7. CPU arch
	arch := ""
	coreList, err := loadCores()
	if err == nil {
		arch = pickRandom(r, coreList)
	}
	if arch == "" {
		arch = "arm64-v8a"
	}

	// 8. OS version (từ os_versions.txt random)
	osVer := ""
	osList, err := loadOSVersions()
	if err == nil {
		osVer = pickRandom(r, osList)
	}
	if osVer == "" {
		osVer = "13"
	}

	// 9. Build ID cho Dalvik prefix — Brand-Model (vd "samsung-SM-S911B").
	buildID := fmt.Sprintf("%s-%s", dev.Brand, dev.Model)

	// 10. Compose FB_UG (match C# L87-102 — KHÔNG slash sau FBDM, KHÔNG ;] cuối)
	fbUG := fmt.Sprintf(
		"[FBAN/FB4A;FBAV/%s;FBBV/%s;FBDM={density=%s,width=%d,height=%d};"+
			"FBLC/%s;FBRV/0;FBCR/%s;FBMF/%s;FBBD/%s;FBPN/com.facebook.katana;"+
			"FBDV/%s;FBSV/%s;FBOP/1;FBCA/%s]",
		fbav, fbbv, density, width, height,
		locale, carrier, dev.Manufacturer, dev.Brand,
		dev.Model, osVer, arch,
	)

	finalUA := fbUG
	if opts.AddVirtualSpecs {
		// Match C# L105: "Dalvik/2.1.0 (Linux; U; Android {osVer}; {Model} Build/{buildID}) {fbUG}"
		finalUA = fmt.Sprintf(
			"Dalvik/2.1.0 (Linux; U; Android %s; %s Build/%s) %s",
			osVer, dev.Model, buildID, fbUG,
		)
	}

	return UABuildResult{
		UserAgent:    finalUA,
		Kind:         KindAndroidApp,
		Locale:       locale,
		Manufacturer: dev.Manufacturer,
		Brand:        dev.Brand,
		Model:        dev.Model,
		OSVersion:    osVer,
		Density:      density,
		Width:        width,
		Height:       height,
		Carrier:      carrier,
		CPUArch:      arch,
		AppVersion:   fbav,
		AppBuild:     fbbv,
		BuildID:      buildID,
	}, nil
}

func samsungSPlatform(platform string) bool {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "s22", "s23", "s24", "s25", "s26",
		"s415", "s425", "s435", "s445",
		"s416", "s417", "s418", "s419", "s420", "s421", "s422", "s423", "s424", "s426",
		"s427", "s428", "s429", "s430", "s431", "s432", "s433", "s434", "s436", "s437",
		"s438", "s439", "s440", "s441", "s442", "s443", "s444",
		"s446", "s447", "s448", "s449", "s450", "s451", "s452", "s453", "s454", "s455",
		"s456", "s457", "s458", "s459", "s460", "s461", "s462", "s463", "s464", "s465",
		"s466", "s467", "s468", "s469", "s470", "s471", "s472", "s473", "s474", "s475", "s476", "s477", "s478", "s479", "s480", "s481", "s482", "s483", "s484", "s485", "s486", "s487", "s488", "s489", "s490", "s491", "s492", "s493", "s494", "s496", "s497", "s498", "s499",
		"s495", "s555v2",
		"s558v2",
		"s556v2", "s557v2",
		"s553v2", "s554v2",
		"s551v2", "s552v2",
		"s550v2",
		"s545", "s546", "s547", "s548", "s549", "s550", "s551", "s552", "s553", "s554",
		"s555", "s556", "s557", "s558", "s559", "s559v2", "s560", "s560v2", "s560v3", "s561", "s561v2", "s562", "s562v3", "s563",
		"s563s21", "s563v3s21", "s563v4s21", "s563v5s21",
		"s563v6s23",
		"s564v1s23", "s564v2s23", "s564v3s23",
		"s565s23", "s565v2s23":
		return true
	default:
		return false
	}
}

// samsungG996Platform — platform dùng Samsung Galaxy S21+ (SM-G996*).
func samsungG996Platform(platform string) bool {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "s561v4s21", "s562v4s21", "s563v4s21", "s563v5s21",
		"s563v6s21",
		"s564v1s21", "s564v2s21", "s564v3s21",
		"s565s21", "s565v2s21":
		return true
	}
	return false
}

func filterSamsungG996Devices(devices []DeviceSpec) []DeviceSpec {
	out := make([]DeviceSpec, 0, len(devices))
	for _, d := range devices {
		if strings.EqualFold(d.Manufacturer, "samsung") &&
			strings.EqualFold(d.Brand, "samsung") &&
			strings.HasPrefix(strings.ToUpper(d.Model), "SM-G99") {
			out = append(out, d)
		}
	}
	return out
}

func filterSamsungSDevices(devices []DeviceSpec) []DeviceSpec {
	out := make([]DeviceSpec, 0, len(devices))
	for _, d := range devices {
		if strings.EqualFold(d.Manufacturer, "samsung") &&
			strings.EqualFold(d.Brand, "samsung") &&
			strings.HasPrefix(strings.ToUpper(d.Model), "SM-S") {
			out = append(out, d)
		}
	}
	return out
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
