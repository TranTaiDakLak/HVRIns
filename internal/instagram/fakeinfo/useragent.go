// useragent.go — Facebook Android (FB4A) User-Agent builder
// Mapping từ C#: AndroidUserAgentBuilder.GetUserAgent()
// Format: [FBAN/FB4A;FBAV/{ver};FBBV/{build};FBDM/{density=d,width=w,height=h};FBLC/{locale};...]
package fakeinfo

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"HVRIns/internal/fbdata"
)

// RandomFbVersion trả về version + build number ngẫu nhiên từ pool CHUNG.
// Nguồn: Config/DeviceInfo/versions_and_builds.txt (hoặc legacy Config/Fbapp/).
// Fallback hardcode khi pool rỗng — giữ app chạy được khi data lỗi.
func RandomFbVersion() (version, buildNum string) {
	return pickRandomVersion(fbdata.Versions())
}

// RandomFbVersionReg — pool REG riêng (versions_and_builds_reg.txt).
// Nếu pool reg rỗng → tự fallback pool CHUNG (logic trong fbdata.VersionsReg).
func RandomFbVersionReg() (version, buildNum string) {
	return pickRandomVersion(fbdata.VersionsReg())
}

// RandomFbVersionVer — pool VER riêng (versions_and_builds_ver.txt).
// Nếu pool ver rỗng → tự fallback pool CHUNG (logic trong fbdata.VersionsVer).
func RandomFbVersionVer() (version, buildNum string) {
	return pickRandomVersion(fbdata.VersionsVer())
}

func pickRandomVersion(pool []fbdata.FbVersion) (version, buildNum string) {
	if len(pool) > 0 {
		r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
		v := pool[r.Intn(len(pool))]
		return v.Version, v.Build
	}
	return "554.0.0.57.70", "918990560"
}

// BuildAndroidUA tạo User-Agent chuẩn FB4A cho Android (default: full specs + buildfile).
// Để tuỳ chỉnh addVirtualSpecs / useBuildNumFile, dùng BuildAndroidUAWithOpts.
func BuildAndroidUA(device DeviceProfile, locale, carrier, fbVer, fbBuild string) string {
	return BuildAndroidUAWithOpts(device, locale, carrier, fbVer, fbBuild, true, true)
}

// BuildAndroidUAWithOpts — C# AndroidUserAgentBuilder.GetUserAgent(locale, addSpecs, buildFile, simBrand).
//
// addVirtualSpecs=true → prepend "Dalvik/2.1.0 (Linux; U; Android ...)" (default, FB trust cao).
// addVirtualSpecs=false → chỉ FB4A blob, không Dalvik prefix.
//
// useBuildNumFile=true → dùng device.BuildID (từ buildnums.txt, vd "SKQ1.210908.001").
// useBuildNumFile=false → tự generate build ID đơn giản "Brand-Model".
func BuildAndroidUAWithOpts(device DeviceProfile, locale, carrier, fbVer, fbBuild string, addVirtualSpecs, useBuildNumFile bool) string {
	if locale == "" {
		locale = "en_US"
	}
	if carrier == "" {
		// Random từ pool 647 carrier (data/carriers.txt) — KHÔNG hardcode Viettel.
		// Pool rỗng (rất hiếm) → fallback "T-Mobile" (carrier US generic).
		carrier = RandomCarrier()
		if carrier == "" {
			carrier = "T-Mobile"
		}
	}
	if fbVer == "" || fbBuild == "" {
		fbVer, fbBuild = RandomFbVersion()
	}

	fb4a := fmt.Sprintf(
		"[FBAN/FB4A;FBAV/%s;FBBV/%s;FBDM={density=%s,width=%d,height=%d};FBLC/%s;FBRV/0;FBCR/%s;FBMF/%s;FBBD/%s;FBPN/com.facebook.katana;FBDV/%s;FBSV/%s;FBOP/1;FBCA/%s]",
		fbVer, fbBuild,
		device.Density, device.ScreenWidth, device.ScreenHeight,
		locale,
		carrier,
		device.Manufacturer, device.Brand,
		device.Model,
		device.OSVersion,
		device.Architecture,
	)

	if !addVirtualSpecs {
		return fb4a // không Dalvik prefix
	}

	// Dalvik prefix — build ID từ file (buildnums.txt) hoặc tự tạo.
	buildID := ""
	if useBuildNumFile {
		buildID = device.BuildID
	}
	if buildID == "" {
		buildID = device.Brand + "-" + device.Model
	}
	return fmt.Sprintf("Dalvik/2.1.0 (Linux; U; Android %s; %s Build/%s) %s",
		device.OSVersion, device.Model, buildID, fb4a)
}

// RandomAndroidUA tạo UA ngẫu nhiên hoàn chỉnh
// Dùng carrier từ C# carriers.txt (647 carriers) thay vì SIM operator name
func RandomAndroidUA(countryCode string) string {
	device := RandomDeviceProfile()
	locale := LocaleFromCountry(countryCode)
	carrier := RandomCarrier()
	fbVer, fbBuild := RandomFbVersion()
	return BuildAndroidUA(device, locale, carrier, fbVer, fbBuild)
}

// WrapWithDalvikPrefix prepend "Dalvik/2.1.0 (...) " vào trước UA FB4A.
// Extract model/os từ FBDV/FBSV trong UA; fallback sang random device khi parse fail.
// Nếu UA đã có Dalvik prefix sẵn → trả về nguyên UA (không double-wrap).
func WrapWithDalvikPrefix(ua string) string {
	if ua == "" {
		return ua
	}
	if strings.HasPrefix(ua, "Dalvik/") {
		return ua
	}
	model := extractUAField(ua, "FBDV/")
	osVer := extractUAField(ua, "FBSV/")
	brand := extractUAField(ua, "FBBD/")
	if model == "" || osVer == "" || brand == "" {
		dp := RandomDeviceProfile()
		if model == "" {
			model = dp.Model
		}
		if osVer == "" {
			osVer = dp.OSVersion
		}
		if brand == "" {
			brand = dp.Brand
		}
	}
	buildID := brand + "-" + model
	return fmt.Sprintf("Dalvik/2.1.0 (Linux; U; Android %s; %s Build/%s) %s", osVer, model, buildID, ua)
}

// extractUAField đọc giá trị giữa "<key>" và ký tự delimit kế tiếp (";" hoặc "]").
func extractUAField(ua, key string) string {
	i := strings.Index(ua, key)
	if i < 0 {
		return ""
	}
	rest := ua[i+len(key):]
	end := strings.IndexAny(rest, ";]")
	if end < 0 {
		return rest
	}
	return rest[:end]
}
