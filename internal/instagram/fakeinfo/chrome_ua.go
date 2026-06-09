// chrome_ua.go — Chrome Android browser UA builder (m.facebook.com).
//
// Format CHUẨN Chrome browser thật (không phải WebView):
//   Mozilla/5.0 (Linux; Android {os}; {model})
//   AppleWebKit/537.36 (KHTML, like Gecko)
//   Chrome/{chromeVer} Mobile Safari/537.36
//
// CHANGED 2026-05: bỏ `Build/{buildId}; wv)` (WebView marker) và `GoogleApp/{ver}`
// suffix. Chrome browser THẬT trên Android không có 2 marker này — FB dùng để
// detect bot vì traffic tới m.facebook.com qua Chrome browser KHÔNG bao giờ
// có wv/GoogleApp. Build ID + GoogleApp chỉ xuất hiện trong WebView nhúng app.
//
// Tất cả thông số đọc từ Config/DeviceInfo/ (cạnh exe). Không còn dùng embed data/.
package fakeinfo

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

var dprList = []string{"2", "3", "3.5"}
var viewportList = []string{"360", "375", "390", "412"}

// ChromeAndroidProfile chứa thông tin đầy đủ cho Chrome Android UA + sec-ch-ua headers
// Mapping từ C#: FacebookAccountModel (ChromeVersion, ChromeVersionFull, AndroidDeviceModel, etc.)
type ChromeAndroidProfile struct {
	UserAgent         string // Mozilla/5.0 (Linux; Android 12; SKQ1...) Chrome/121... Mobile Safari...
	ChromeVersion     string // "121" — major only (cho sec-ch-ua: v="121")
	ChromeVersionFull string // "121.0.6167" — 3-part (cho sec-ch-ua-full-version-list)
	AndroidModel      string // "Redmi Note 11" (từ devices.txt)
	AndroidOsVersion  string // "12" (từ os_versions.txt)
	BuildId           string // "SKQ1.210908.001" (từ buildnums.txt)
	Dpr               string // "2", "3", "3.5"
	ViewportWidth     string // "360", "375", "390", "412"
}

// RandomChromeAndroidProfile tạo Chrome Android browser profile ngẫu nhiên
// Mapping từ C#: BrowserAndroidUserAgentBuilder.GetUserAgent(use_buildnum_file=true)
func RandomChromeAndroidProfile() ChromeAndroidProfile {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))

	// Chrome version từ Config/DeviceInfo/chrome_versions.txt (format: "121.0.6167")
	chromeVer := "121.0.6167"
	if list := loadDeviceInfoLines("chrome_versions.txt"); len(list) > 0 {
		chromeVer = list[r.Intn(len(list))]
	}
	chromeMajor := strings.SplitN(chromeVer, ".", 2)[0]

	// Android OS version từ Config/DeviceInfo/os_versions.txt
	osVer := "12"
	if list := getOSVersionList(); len(list) > 0 {
		osVer = list[r.Intn(len(list))]
	}

	// Build ID — DÙNG TRONG DEVICE SLOT (port từ bản cũ ed6fd1a 4/23/2026).
	// Lý do: bản cũ đạt 22% Live rate vs bản đổi model đạt 0.8%. Build ID
	// (SP1A.210812.016 / QQ1D.200105.002 / 62.0.B.1.30...) "rare" → FB anti-bot
	// ít cluster signature → ít detect. Model thật (SM-S911B, Pixel 8...) phổ
	// biến trong FB cluster DB → dễ track.
	buildId := RandomBuildNum()

	// Device model — vẫn random và lưu trong struct (cho compatibility / debug headers).
	device := RandomDeviceProfile()
	model := device.Model
	if model == "" {
		model = "SM-S911B"
	}

	// Chrome version 3-part: bản cũ dùng đúng format trong file (146.0.7680),
	// KHÔNG thêm build suffix. Chrome thật cũng có UA 3-part vì format reduction
	// (https://developer.chrome.com/docs/web-platform/user-agent-reduction) —
	// các Chrome version mới gửi UA 3-part hoặc thậm chí 2-part.

	// DPR và Viewport
	dpr := dprList[r.Intn(len(dprList))]
	viewport := viewportList[r.Intn(len(viewportList))]

	// UA format: PORT từ bản cũ 4/23 — device slot = Build ID, Chrome = 3-part.
	// Mozilla/5.0 (Linux; Android <os>; <buildId>) ... Chrome/<3-part> Mobile Safari/...
	ua := fmt.Sprintf(
		"Mozilla/5.0 (Linux; Android %s; %s) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/%s Mobile Safari/537.36",
		osVer, buildId, chromeVer,
	)

	return ChromeAndroidProfile{
		UserAgent:         ua,
		ChromeVersion:     chromeMajor,
		ChromeVersionFull: chromeVer,
		AndroidModel:      model,
		AndroidOsVersion:  osVer,
		BuildId:           buildId,
		Dpr:               dpr,
		ViewportWidth:     viewport,
	}
}

// RandomBuildNum lấy build number ngẫu nhiên từ Config/DeviceInfo/buildnums.txt
func RandomBuildNum() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	if list := getBuildnumList(); len(list) > 0 {
		return list[r.Intn(len(list))]
	}
	return "SKQ1.210908.001"
}
