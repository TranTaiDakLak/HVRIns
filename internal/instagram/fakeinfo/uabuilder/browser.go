// browser.go — BrowserUABuilder port C# BrowserAndroidUserAgentBuilder.cs.
//
// Output Chrome Android WebView UA. Format:
//
//	Mozilla/5.0 (Linux; Android {os}; {model} Build/{buildId}; wv)
//	  AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0
//	  Chrome/{chromeVer} Mobile Safari/537.36 GoogleApp/{googleAppVer}
//
// Tất cả thông số đọc từ Config/DeviceInfo. Suffix "|<model>|<os>" append cho caller metadata.
package uabuilder

import (
	"fmt"
	"strings"
)

type BrowserUABuilder struct{}

func (b *BrowserUABuilder) Kind() UABuilderKind { return KindBrowserAndroid }

func (b *BrowserUABuilder) Build(opts UAOptions) (UABuildResult, error) {
	r := opts.NewRand()

	// 1. App version + build (metadata; không xuất hiện trong Chrome UA)
	var fbav, fbbv string
	if opts.PinAppVersion != "" && opts.PinBuild != "" {
		fbav = opts.PinAppVersion
		fbbv = opts.PinBuild
	} else {
		appVersions, err := LoadAppVersionsForPlatform(opts.Platform)
		if err == nil && len(appVersions) > 0 {
			av := appVersions[r.Intn(len(appVersions))]
			fbav = av.Version
			fbbv = av.Build
		}
	}

	// 2. Device model từ Config/DeviceInfo/devices.txt
	devices, _ := LoadDevicesForPlatform(opts.Platform)
	if len(devices) == 0 {
		devices = fallbackDevices
	}
	dev := devices[r.Intn(len(devices))]

	// Build ID từ Config/DeviceInfo/device_build_nums.txt (SKQ1.210908.001, AP3A.240905.015.A2, ...)
	buildID := ""
	if buildNums, err := loadStringList("DeviceInfo/device_build_nums.txt"); err == nil && len(buildNums) > 0 {
		buildID = buildNums[r.Intn(len(buildNums))]
	}
	if buildID == "" {
		buildID = fmt.Sprintf("%s-%s", dev.Brand, dev.Model)
	}

	// 3. OS version
	osVer := ""
	osList, _ := loadOSVersions()
	osVer = pickRandom(r, osList)
	if osVer == "" {
		osVer = "13"
	}

	// 4. Chrome version (4-part: file có 3-part → thêm build suffix ngẫu nhiên)
	chromeVer := ""
	chromeList, _ := loadChromeVersions()
	chromeVer = pickRandom(r, chromeList)
	if chromeVer == "" {
		chromeVer = "146.0.7680.111"
	} else if strings.Count(chromeVer, ".") == 2 {
		chromeVer = fmt.Sprintf("%s.%d", chromeVer, 50+r.Intn(150))
	}

	// 5. GoogleApp version (17.18.24.ve.arm64, ...)
	googleAppVer := "17.18.24.ve.arm64"
	googleList, _ := loadGoogleAppVersions()
	if g := pickRandom(r, googleList); g != "" {
		googleAppVer = g
	}

	// 6. Compose UA
	ua := fmt.Sprintf(
		"Mozilla/5.0 (Linux; Android %s; %s Build/%s; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/%s Mobile Safari/537.36 GoogleApp/%s",
		osVer, dev.Model, buildID, chromeVer, googleAppVer,
	)

	// 7. Append metadata suffix để caller tách model/os
	ua += "|" + dev.Model + "|" + osVer

	locale := opts.Locale
	if locale == "" {
		locale = "en_US"
	}
	carrier := opts.SimBrand

	return UABuildResult{
		UserAgent:     ua,
		Kind:          KindBrowserAndroid,
		Locale:        locale,
		Manufacturer:  dev.Manufacturer,
		Brand:         dev.Brand,
		Model:         dev.Model,
		OSVersion:     osVer,
		Carrier:       carrier,
		AppVersion:    fbav,
		AppBuild:      fbbv,
		BuildID:       buildID,
		ChromeVersion: chromeVer,
	}, nil
}
