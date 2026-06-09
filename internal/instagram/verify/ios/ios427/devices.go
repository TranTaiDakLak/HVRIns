// devices.go — iOS555 verify device + FB app build data tables.
//
// Load từ file Config/DeviceInfoIOS/ nếu có, fallback về default hardcode.
// Format:
//
//	ios_devices.txt:    FBDV|IOSDot|IOSUnder|MobileBld|FBSS
//	ios_app_builds.txt: FBAV|FBBV|FBRV
//
// Dòng bắt đầu bằng # là comment, bỏ qua.
// Đồng bộ với reg/ios562/devices.go (cùng file config, cùng format).
package ios427

import (
	"bufio"
	mrand "math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"HVRIns/internal/instagram/fakeinfo/uabuilder"
)

// ─── iPhone device pool ───────────────────────────────────────────────────────

type verifyIOSDevice struct {
	FBDV      string
	IOSDot    string
	IOSUnder  string
	MobileBld string
	FBSS      string
}

var defaultVerifyIOSDevices = []verifyIOSDevice{
	{"iPhone8,1", "15.8.3", "15_8_3", "19H307", "2"},
	{"iPhone8,2", "15.8.3", "15_8_3", "19H307", "3"},
	{"iPhone9,1", "15.8.8", "15_8_8", "19H422", "2"},
	{"iPhone9,2", "15.8.8", "15_8_8", "19H422", "3"},
	{"iPhone10,1", "16.7.16", "16_7_16", "20H392", "2"},
	{"iPhone10,2", "16.7.16", "16_7_16", "20H392", "3"},
	{"iPhone10,3", "16.7.16", "16_7_16", "20H392", "3"},
	{"iPhone11,2", "18.7.9", "18_7_9", "22H355", "3"},
	{"iPhone11,8", "18.7.9", "18_7_9", "22H355", "2"},
	{"iPhone12,1", "18.7.9", "18_7_9", "22H355", "2"},
	{"iPhone12,3", "18.7.9", "18_7_9", "22H355", "3"},
	{"iPhone12,8", "18.7.9", "18_7_9", "22H355", "2"},
	{"iPhone13,2", "18.7.9", "18_7_9", "22H355", "3"},
	{"iPhone13,4", "18.7.9", "18_7_9", "22H355", "3"},
	{"iPhone14,2", "18.7.9", "18_7_9", "22H355", "3"},
	{"iPhone14,5", "18.7.9", "18_7_9", "22H355", "3"},
	{"iPhone14,6", "18.7.9", "18_7_9", "22H355", "2"},
	{"iPhone14,7", "18.7.9", "18_7_9", "22H355", "3"},
	{"iPhone15,2", "18.7.9", "18_7_9", "22H355", "3"},
	{"iPhone15,4", "26.5", "26_5", "23F77", "3"},
	{"iPhone15,5", "26.5", "26_5", "23F77", "3"},
	{"iPhone16,1", "26.5", "26_5", "23F77", "3"},
	{"iPhone16,2", "26.5", "26_5", "23F77", "3"},
	{"iPhone17,1", "26.5", "26_5", "23F77", "3"},
	{"iPhone17,3", "26.5", "26_5", "23F77", "3"},
}

var (
	verifyDevicesOnce sync.Once
	verifyIOSDevices  []verifyIOSDevice
)

func getVerifyIOSDevices() []verifyIOSDevice {
	verifyDevicesOnce.Do(func() {
		loaded := loadVerifyDevicesFromFile()
		if len(loaded) > 0 {
			verifyIOSDevices = loaded
		} else {
			verifyIOSDevices = defaultVerifyIOSDevices
		}
	})
	return verifyIOSDevices
}

func loadVerifyDevicesFromFile() []verifyIOSDevice {
	path := filepath.Join(uabuilder.GetConfigBaseDir(), "DeviceInfoIOS", "ios_devices.txt")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []verifyIOSDevice
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
		d := verifyIOSDevice{
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

type verifyFBBuild struct {
	FBAV string
	FBBV string
	FBRV string
}

var defaultVerifyFBBuilds = []verifyFBBuild{
	{"555.0.0.36.63", "923840166", "0"},         // v563 (capture EnterCode563)
	{"562.0.0.61.70", "974804325", "979621922"}, // v562 May 2026
	{"559.0.0.56.80", "955347213", "960697664"}, // v559 Apr 2026 (confirmed)
	{"558.0.0.58.74", "948561885", "957194181"}, // v558 Apr 2026 (confirmed)
}

var (
	verifyBuildsOnce sync.Once
	verifyFBBuilds   []verifyFBBuild
)

func getVerifyFBBuilds() []verifyFBBuild {
	verifyBuildsOnce.Do(func() {
		loaded := loadVerifyBuildsFromFile()
		if len(loaded) > 0 {
			verifyFBBuilds = loaded
		} else {
			verifyFBBuilds = defaultVerifyFBBuilds
		}
	})
	return verifyFBBuilds
}

func loadVerifyBuildsFromFile() []verifyFBBuild {
	path := filepath.Join(uabuilder.GetConfigBaseDir(), "DeviceInfoIOS", "ios_app_builds_ver.txt")
	if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
		path = filepath.Join(uabuilder.GetConfigBaseDir(), "DeviceInfoIOS", "ios_app_builds.txt")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []verifyFBBuild
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
		b := verifyFBBuild{
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

func randVerifyFBBuild() verifyFBBuild {
	builds := getVerifyFBBuilds()
	return builds[mrand.Intn(len(builds))]
}

// ─── iOS locale pool ─────────────────────────────────────────────────────────

var (
	verifyLocalesOnce sync.Once
	verifyIOSLocales  []string
)

var defaultVerifyIOSLocales = []string{
	"en_US", "en_GB", "en_AU", "en_CA", "en_IN",
	"es_ES", "es_MX", "es_AR", "fr_FR", "fr_CA",
	"de_DE", "it_IT", "pt_BR", "pt_PT", "ru_RU",
	"zh_CN", "zh_TW", "ja_JP", "ko_KR", "ar_SA",
	"tr_TR", "vi_VN", "th_TH", "id_ID", "ms_MY",
	"pl_PL", "nl_NL", "sv_SE", "he_IL", "hi_IN",
}

func randVerifyIOSLocale() string {
	verifyLocalesOnce.Do(func() {
		path := filepath.Join(uabuilder.GetConfigBaseDir(), "DeviceInfoIOS", "ios_locales.txt")
		f, err := os.Open(path)
		if err != nil {
			verifyIOSLocales = defaultVerifyIOSLocales
			return
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			verifyIOSLocales = append(verifyIOSLocales, line)
		}
		if len(verifyIOSLocales) == 0 {
			verifyIOSLocales = defaultVerifyIOSLocales
		}
	})
	return verifyIOSLocales[mrand.Intn(len(verifyIOSLocales))]
}
