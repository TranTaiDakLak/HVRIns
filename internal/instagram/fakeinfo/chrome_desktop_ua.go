// chrome_desktop_ua.go — Chrome Desktop browser UA builder (m.facebook.com / www.facebook.com).
//
// Format CHUẨN Chrome Desktop thật (Windows / macOS / Linux):
//   Mozilla/5.0 (<platform>) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/{ver} Safari/537.36
//
// Dùng cho api mfb verify (PlatformWeb) khi user tick "Build UA" — sinh UA random
// thay cho hardcoded Chrome 134 Win10 mặc định trong verify/web/steps.go.
//
// Combinations: 12 Chrome versions × 5 OS variants × 2 brand (Chrome/Edge) = ~120 UA unique
// → đủ đa dạng cho 10k+ verify không lặp pattern.
package fakeinfo

import (
	"fmt"
	"math/rand"
	"time"
)

// ChromeDesktopProfile chứa UA + thông tin liên quan cho sec-ch-ua headers.
type ChromeDesktopProfile struct {
	UserAgent             string // Full UA string
	ChromeVersion         string // "134" — major only (sec-ch-ua: v="134")
	ChromeVersionFull     string // "134.0.0.0" — 4-part (sec-ch-ua-full-version-list)
	Platform              string // "Windows" | "macOS" | "Linux"
	PlatformVersion       string // "10.0.0" | "15.7.1" | "6.5.0"
	IsEdge                bool   // true → brand = Microsoft Edge
}

// Chrome major versions phổ biến gần đây (recent ~2 years).
// Ưu tiên các version mới (130+) vì chiếm market share lớn hơn.
var chromeDesktopMajorVersions = []string{
	"110", "115", "118", "120", "122", "124", "126", "128", "130", "132", "134", "136",
}

// OS variants: (UA platform string, sec-ch-ua-platform, sec-ch-ua-platform-version).
type chromeDesktopOS struct {
	uaPlatform              string // dùng trong UA string
	chPlatform              string // dùng trong sec-ch-ua-platform header
	chPlatformVersion       string // dùng trong sec-ch-ua-platform-version header
}

var chromeDesktopOSes = []chromeDesktopOS{
	{"Windows NT 10.0; Win64; x64", "Windows", "10.0.0"},   // Win 10
	{"Windows NT 10.0; Win64; x64", "Windows", "15.0.0"},   // Win 11 (NT vẫn là 10.0 trong UA, distinguish qua ch-ua-platform-version)
	{"Windows NT 10.0; Win64; x64", "Windows", "13.0.0"},   // Win 11 (older minor)
	{"Macintosh; Intel Mac OS X 10_15_7", "macOS", "15.1.0"}, // macOS Sequoia
	{"Macintosh; Intel Mac OS X 10_15_7", "macOS", "14.6.0"}, // macOS Sonoma
}

// RandomChromeDesktopProfile sinh Chrome Desktop profile ngẫu nhiên.
// Mix Chrome 110-136, Windows 10/11, macOS 14/15, optional Edge brand.
func RandomChromeDesktopProfile() ChromeDesktopProfile {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))

	majorVer := chromeDesktopMajorVersions[r.Intn(len(chromeDesktopMajorVersions))]
	fullVer := majorVer + ".0.0.0"

	os := chromeDesktopOSes[r.Intn(len(chromeDesktopOSes))]

	// ~15% chance là Edge (cũng dùng Chromium engine, FB nhận diện như Chrome variant).
	isEdge := r.Intn(100) < 15
	var ua string
	if isEdge {
		ua = fmt.Sprintf(
			"Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36 Edg/%s.0.0.0",
			os.uaPlatform, majorVer, majorVer,
		)
	} else {
		ua = fmt.Sprintf(
			"Mozilla/5.0 (%s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s.0.0.0 Safari/537.36",
			os.uaPlatform, majorVer,
		)
	}

	return ChromeDesktopProfile{
		UserAgent:         ua,
		ChromeVersion:     majorVer,
		ChromeVersionFull: fullVer,
		Platform:          os.chPlatform,
		PlatformVersion:   os.chPlatformVersion,
		IsEdge:            isEdge,
	}
}

// RandomChromeDesktopUA shortcut — chỉ lấy UA string (dùng khi không cần sec-ch-ua).
func RandomChromeDesktopUA() string {
	return RandomChromeDesktopProfile().UserAgent
}
