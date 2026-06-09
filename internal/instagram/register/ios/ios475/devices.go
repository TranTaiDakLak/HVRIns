// devices.go — iOS555 device + FB app build data tables.
//
// Load từ file Config/DeviceInfoIOS/ nếu có, fallback về default hardcode.
// Format:
//
//	ios_devices.txt:    FBDV|IOSDot|IOSUnder|MobileBld|FBSS  (mỗi dòng 1 device)
//	ios_app_builds.txt: FBAV|FBBV|FBRV                       (mỗi dòng 1 build)
//
// Dòng bắt đầu bằng # là comment, bỏ qua.
// Nguồn mặc định: user-agents.net, whatismybrowser.com, ipsw.me, betawiki.net.
package ios475

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"HVRIns/internal/instagram/fakeinfo/uabuilder"
)

// ─── iPhone device table ─────────────────────────────────────────────────────

// iPhoneDevice mô tả 1 tổ hợp thiết bị iPhone + iOS thật.
// FBSS: 2 = @2x Retina (LCD 326ppi), 3 = @3x Super Retina (OLED 458-460ppi).
type iPhoneDevice struct {
	FBDV      string // Apple model identifier, vd "iPhone9,1"
	IOSDot    string // iOS version dạng dot — dùng trong FBSV, vd "15.8.8"
	IOSUnder  string // iOS version dạng underscore — dùng trong UA, vd "15_8_8"
	MobileBld string // iOS build number — dùng trong Mobile/xxx, vd "19H422"
	FBSS      string // Screen scale: "2" hoặc "3"
}

// defaultIPhoneDevices — fallback khi file ios_devices.txt không tồn tại/rỗng.
// Verified theo Apple release notes + ipsw.me (2026-05-28).
var defaultIPhoneDevices = []iPhoneDevice{
	// ── iPhone 7 / 7 Plus — iOS 15.8.8 (max) ─────────────────────────────
	{FBDV: "iPhone9,1", IOSDot: "15.8.8", IOSUnder: "15_8_8", MobileBld: "19H422", FBSS: "2"},
	{FBDV: "iPhone9,2", IOSDot: "15.8.8", IOSUnder: "15_8_8", MobileBld: "19H422", FBSS: "3"},
	// ── iPhone 8 / 8 Plus — iOS 16.7.16 (max) ────────────────────────────
	{FBDV: "iPhone10,1", IOSDot: "16.7.16", IOSUnder: "16_7_16", MobileBld: "20H392", FBSS: "2"},
	{FBDV: "iPhone10,4", IOSDot: "16.7.16", IOSUnder: "16_7_16", MobileBld: "20H392", FBSS: "2"},
	// ── iPhone XR / XS — iOS 18.7.9 (max) ───────────────────────────────
	{FBDV: "iPhone11,8", IOSDot: "18.7.9", IOSUnder: "18_7_9", MobileBld: "22H355", FBSS: "2"},
	{FBDV: "iPhone11,2", IOSDot: "18.7.9", IOSUnder: "18_7_9", MobileBld: "22H355", FBSS: "3"},
	// ── iPhone 11 / 11 Pro — iOS 18.7.9 (max) ────────────────────────────
	{FBDV: "iPhone12,1", IOSDot: "18.7.9", IOSUnder: "18_7_9", MobileBld: "22H355", FBSS: "2"},
	{FBDV: "iPhone12,3", IOSDot: "18.7.9", IOSUnder: "18_7_9", MobileBld: "22H355", FBSS: "3"},
	// ── iPhone 12 / 12 Pro Max — iOS 18.7.9 ──────────────────────────────
	{FBDV: "iPhone13,2", IOSDot: "18.7.9", IOSUnder: "18_7_9", MobileBld: "22H355", FBSS: "3"},
	{FBDV: "iPhone13,4", IOSDot: "18.7.9", IOSUnder: "18_7_9", MobileBld: "22H355", FBSS: "3"},
	// ── iPhone 13 / 13 Pro — iOS 18.7.9 ──────────────────────────────────
	{FBDV: "iPhone14,4", IOSDot: "18.7.9", IOSUnder: "18_7_9", MobileBld: "22H355", FBSS: "3"},
	{FBDV: "iPhone14,2", IOSDot: "18.7.9", IOSUnder: "18_7_9", MobileBld: "22H355", FBSS: "3"},
	// ── iPhone 14 / 14 Pro — iOS 18.7.9 ──────────────────────────────────
	{FBDV: "iPhone14,7", IOSDot: "18.7.9", IOSUnder: "18_7_9", MobileBld: "22H355", FBSS: "3"},
	{FBDV: "iPhone15,2", IOSDot: "18.7.9", IOSUnder: "18_7_9", MobileBld: "22H355", FBSS: "3"},
	// ── iPhone 15 / 15 Pro — iOS 26.5 ────────────────────────────────────
	{FBDV: "iPhone15,4", IOSDot: "26.5", IOSUnder: "26_5", MobileBld: "23F77", FBSS: "3"},
	{FBDV: "iPhone16,1", IOSDot: "26.5", IOSUnder: "26_5", MobileBld: "23F77", FBSS: "3"},
}

var (
	iPhoneDevicesOnce sync.Once
	iPhoneDevices     []iPhoneDevice
)

// getIPhoneDevices trả pool device — load từ file lần đầu, sau đó cache.
func getIPhoneDevices() []iPhoneDevice {
	iPhoneDevicesOnce.Do(func() {
		loaded := loadIOSDevicesFromFile()
		if len(loaded) > 0 {
			iPhoneDevices = loaded
		} else {
			iPhoneDevices = defaultIPhoneDevices
		}
	})
	return iPhoneDevices
}

// ReloadIOSDevices buộc reload từ file lần tiếp theo (dùng khi user cập nhật file).
func ReloadIOSDevices() {
	iPhoneDevicesOnce = sync.Once{}
	fbBuildsOnce = sync.Once{}
}

func loadIOSDevicesFromFile() []iPhoneDevice {
	path := filepath.Join(uabuilder.GetConfigBaseDir(), "DeviceInfoIOS", "ios_devices.txt")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []iPhoneDevice
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 5 {
			continue
		}
		d := iPhoneDevice{
			FBDV:      strings.TrimSpace(parts[0]),
			IOSDot:    strings.TrimSpace(parts[1]),
			IOSUnder:  strings.TrimSpace(parts[2]),
			MobileBld: strings.TrimSpace(parts[3]),
			FBSS:      strings.TrimSpace(parts[4]),
		}
		if d.FBDV == "" || d.IOSDot == "" {
			continue
		}
		out = append(out, d)
	}
	return out
}

// ─── FB app build pool ────────────────────────────────────────────────────────

// fbBuild là 1 bộ version thật của FB iOS app.
// FBAV+FBBV là cặp cố định theo App Store release.
// FBRV vary theo device/install (có thể là "0").
type fbBuild struct {
	FBAV string
	FBBV string
	FBRV string
}

// defaultFBBuilds — fallback khi file builds.txt không tồn tại/rỗng.
// Chỉ giữ versions gần v562 (bloksVersioningID/docID capture từ v562).
var defaultFBBuilds = []fbBuild{
	{"555.0.0.36.63", "923840166", "0"},         // v563 (capture EnterCode563)
	{"562.0.0.61.70", "974804325", "979621922"}, // v562 May 2026
	{"559.0.0.56.80", "955347213", "960697664"}, // v559 Apr 2026 (confirmed)
	{"558.0.0.58.74", "948561885", "957194181"}, // v558 Apr 2026 (confirmed)
}

var (
	fbBuildsOnce sync.Once
	fbBuilds     []fbBuild
)

func getFBBuilds() []fbBuild {
	fbBuildsOnce.Do(func() {
		loaded := loadFBBuildsFromFile()
		if len(loaded) > 0 {
			fbBuilds = loaded
		} else {
			fbBuilds = defaultFBBuilds
		}
	})
	return fbBuilds
}

func loadFBBuildsFromFile() []fbBuild {
	path := filepath.Join(uabuilder.GetConfigBaseDir(), "DeviceInfoIOS", "ios_app_builds_reg.txt")
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		path = filepath.Join(uabuilder.GetConfigBaseDir(), "DeviceInfoIOS", "ios_app_builds.txt")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []fbBuild
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) != 3 {
			continue
		}
		b := fbBuild{
			FBAV: strings.TrimSpace(parts[0]),
			FBBV: strings.TrimSpace(parts[1]),
			FBRV: strings.TrimSpace(parts[2]),
		}
		if b.FBAV == "" || b.FBBV == "" {
			continue
		}
		out = append(out, b)
	}
	return out
}

// ─── iOS locale pool ─────────────────────────────────────────────────────────

var (
	iosLocales     []string
	iosLocalesOnce sync.Once
)

var defaultIOSLocales = []string{
	"en_US", "en_GB", "en_AU", "en_CA", "en_IN",
	"es_ES", "es_MX", "es_AR", "fr_FR", "fr_CA",
	"de_DE", "it_IT", "pt_BR", "pt_PT", "ru_RU",
	"zh_CN", "zh_TW", "ja_JP", "ko_KR", "ar_SA",
	"tr_TR", "vi_VN", "th_TH", "id_ID", "ms_MY",
	"pl_PL", "nl_NL", "sv_SE", "da_DK", "fi_FI",
	"he_IL", "hi_IN", "tl_PH",
}

func getIOSLocales() []string {
	iosLocalesOnce.Do(func() {
		if loaded := loadIOSLocalesFromFile(); len(loaded) > 0 {
			iosLocales = loaded
		} else {
			iosLocales = defaultIOSLocales
		}
	})
	return iosLocales
}

// RandIOSLocale trả locale random từ pool ios_locales.txt.
func RandIOSLocale() string {
	pool := getIOSLocales()
	return pool[randInt(len(pool))]
}

func loadIOSLocalesFromFile() []string {
	path := filepath.Join(uabuilder.GetConfigBaseDir(), "DeviceInfoIOS", "ios_locales.txt")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}

func randFBBuild() fbBuild {
	builds := getFBBuilds()
	return builds[randInt(len(builds))]
}
