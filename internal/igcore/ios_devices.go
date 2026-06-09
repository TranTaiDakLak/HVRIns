// ios_devices.go — pool thiết bị iOS thực để rotate fingerprint mỗi luồng reg.
// Dữ liệu từ real device captures. Mỗi luồng nhận 1 device ngẫu nhiên.
package igcore

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// iOSDevice đại diện cho 1 model iPhone thực.
type iOSDevice struct {
	ModelID     string // iPhone14,3
	ModelName   string // iPhone 13 Pro Max
	ScreenW     int    // 1284
	ScreenH     int    // 2778
	Scale       string // 3.00
	DPI         int    // screen dpi (647 for 3x, 460 for 2x)
	MinIOS      int    // iOS major version tối thiểu
	MaxIOS      int    // iOS major version tối đa hỗ trợ
}

// iOSVersion = major.minor_patch dùng trong UA.
type iOSVersion struct {
	Display string // "17_4_1"
	Major   int
}

// igAppVersion chứa IG app version + build number.
type igAppVersion struct {
	Version string // "410.1.0.36.70"
	Build   string // "849447290"
}

// ── Device pool ──────────────────────────────────────────────────────────────

var iosDevicePool = []iOSDevice{
	// iPhone 8 series
	{"iPhone10,1", "iPhone 8", 750, 1334, "2.00", 326, 11, 16},
	{"iPhone10,4", "iPhone 8", 750, 1334, "2.00", 326, 11, 16},
	// iPhone X
	{"iPhone10,3", "iPhone X", 1125, 2436, "3.00", 458, 11, 16},
	{"iPhone10,6", "iPhone X", 1125, 2436, "3.00", 458, 11, 16},
	// iPhone XR / 11
	{"iPhone11,8", "iPhone XR", 828, 1792, "2.00", 326, 12, 17},
	{"iPhone12,1", "iPhone 11", 828, 1792, "2.00", 326, 13, 17},
	// iPhone XS / 11 Pro
	{"iPhone11,2", "iPhone XS", 1125, 2436, "3.00", 458, 12, 17},
	{"iPhone12,3", "iPhone 11 Pro", 1125, 2436, "3.00", 458, 13, 17},
	// iPhone XS Max / 11 Pro Max
	{"iPhone11,4", "iPhone XS Max", 1242, 2688, "3.00", 458, 12, 17},
	{"iPhone12,5", "iPhone 11 Pro Max", 1242, 2688, "3.00", 458, 13, 17},
	// iPhone 12 series
	{"iPhone13,2", "iPhone 12", 1170, 2532, "3.00", 460, 14, 17},
	{"iPhone13,3", "iPhone 12 Pro", 1170, 2532, "3.00", 460, 14, 17},
	{"iPhone13,4", "iPhone 12 Pro Max", 1284, 2778, "3.00", 458, 14, 17},
	// iPhone 13 series
	{"iPhone14,2", "iPhone 13 Pro", 1170, 2532, "3.00", 460, 15, 17},
	{"iPhone14,3", "iPhone 13 Pro Max", 1284, 2778, "3.00", 458, 15, 17},
	{"iPhone14,5", "iPhone 13", 1170, 2532, "3.00", 460, 15, 17},
	{"iPhone14,4", "iPhone 13 mini", 1080, 2340, "3.00", 476, 15, 17},
	// iPhone 14 series
	{"iPhone14,7", "iPhone 14", 1170, 2532, "3.00", 460, 16, 17},
	{"iPhone14,8", "iPhone 14 Plus", 1284, 2778, "3.00", 458, 16, 17},
	{"iPhone15,2", "iPhone 14 Pro", 1179, 2556, "3.00", 460, 16, 17},
	{"iPhone15,3", "iPhone 14 Pro Max", 1290, 2796, "3.00", 460, 16, 17},
	// iPhone 15 series
	{"iPhone15,4", "iPhone 15", 1179, 2556, "3.00", 460, 17, 17},
	{"iPhone15,5", "iPhone 15 Plus", 1290, 2796, "3.00", 460, 17, 17},
	{"iPhone16,1", "iPhone 15 Pro", 1179, 2556, "3.00", 460, 17, 17},
	{"iPhone16,2", "iPhone 15 Pro Max", 1290, 2796, "3.00", 460, 17, 17},
}

// ── iOS version pool ─────────────────────────────────────────────────────────

var iosVersionPool = []iOSVersion{
	{"15_6_1", 15}, {"15_7_1", 15}, {"15_7_2", 15},
	{"15_8_1", 15}, {"15_8_2", 15}, {"15_8_3", 15}, {"15_8_4", 15},
	{"16_0_3", 16}, {"16_1_1", 16}, {"16_2", 16},
	{"16_3_1", 16}, {"16_4_1", 16}, {"16_5_1", 16},
	{"16_6_1", 16}, {"16_7_2", 16}, {"16_7_5", 16},
	{"17_0_3", 17}, {"17_1_2", 17}, {"17_2_1", 17},
	{"17_3_1", 17}, {"17_4_1", 17}, {"17_5_1", 17},
	{"17_6", 17},
}

// ── IG App version pool ───────────────────────────────────────────────────────
// (version, build) từ real captures, stable release versions.

var igAppVersionPool = []igAppVersion{
	{"323.0.0.10.89", "565210949"},
	{"334.0.0.24.106", "587396094"},
	{"345.0.0.15.108", "603163266"},
	{"355.0.0.13.92", "618700572"},
	{"360.0.0.16.112", "627617621"},
	{"368.1.0.20.103", "638804958"},
	{"374.0.0.21.121", "651580628"},
	{"380.0.0.24.127", "661064929"},
	{"390.0.0.31.136", "676625680"},
	{"400.0.0.32.126", "695501093"},
	{"410.1.0.36.70", "849447290"},
	{"416.0.0.37.86", "712543085"},
	{"420.0.0.45.97", "719879636"},
	{"428.0.0.26.88", "732266754"},
	{"435.0.0.25.108", "742613068"},
}

// ── Locale pool (fallback khi không biết country) ─────────────────────────────

var localePool = []string{
	"vi_VN", "vi_VN", "vi_VN",
	"en_US", "en_GB", "en_AU",
	"th_TH", "id_ID", "ms_MY",
}

// countryLocaleMap map ISO country code → IG locale string.
var countryLocaleMap = map[string]string{
	// Đông Nam Á
	"VN": "vi_VN", "TH": "th_TH", "ID": "id_ID",
	"MY": "ms_MY", "PH": "en_PH", "SG": "en_SG",
	"KH": "km_KH", "MM": "my_MM", "LA": "lo_LA",
	// Châu Á
	"JP": "ja_JP", "KR": "ko_KR", "CN": "zh_CN",
	"TW": "zh_TW", "HK": "zh_HK", "IN": "hi_IN",
	"PK": "en_PK", "BD": "bn_BD",
	// Mỹ & Canada
	"US": "en_US", "CA": "en_CA", "MX": "es_MX",
	// Anh & Úc
	"GB": "en_GB", "AU": "en_AU", "NZ": "en_NZ",
	// Châu Âu
	"DE": "de_DE", "FR": "fr_FR", "IT": "it_IT",
	"ES": "es_ES", "PT": "pt_PT", "NL": "nl_NL",
	"SE": "sv_SE", "NO": "nb_NO", "FI": "fi_FI",
	"PL": "pl_PL", "RU": "ru_RU", "UA": "uk_UA",
	"TR": "tr_TR", "RO": "ro_RO", "CZ": "cs_CZ",
	"HU": "hu_HU", "GR": "el_GR", "BG": "bg_BG",
	// Trung Đông
	"SA": "ar_SA", "AE": "ar_AE", "EG": "ar_EG",
	"IL": "he_IL", "KW": "ar_KW", "QA": "ar_QA",
	// Nam Mỹ
	"BR": "pt_BR", "AR": "es_AR", "CO": "es_CO",
	"CL": "es_CL", "PE": "es_PE", "VE": "es_VE",
	// Châu Phi
	"ZA": "en_ZA", "NG": "en_NG", "KE": "sw_KE",
}

// CountryToLocale trả locale phù hợp với country code ISO.
// VD: "VN" → "vi_VN", "US" → "en_US".
// Trả "en_US" nếu không có trong map.
func CountryToLocale(countryCode string) string {
	if loc, ok := countryLocaleMap[countryCode]; ok {
		return loc
	}
	return "en_US"
}

// ── Random helpers ────────────────────────────────────────────────────────────

func cryptoIntn(n int) int {
	if n <= 0 {
		return 0
	}
	v, _ := rand.Int(rand.Reader, big.NewInt(int64(n)))
	return int(v.Int64())
}

func pickDevice() iOSDevice {
	return iosDevicePool[cryptoIntn(len(iosDevicePool))]
}

func pickIOSVersion(minMajor, maxMajor int) iOSVersion {
	var pool []iOSVersion
	for _, v := range iosVersionPool {
		if v.Major >= minMajor && v.Major <= maxMajor {
			pool = append(pool, v)
		}
	}
	if len(pool) == 0 {
		return iosVersionPool[cryptoIntn(len(iosVersionPool))]
	}
	return pool[cryptoIntn(len(pool))]
}

func pickAppVersion() igAppVersion {
	return igAppVersionPool[cryptoIntn(len(igAppVersionPool))]
}

func pickLocale() string {
	return localePool[cryptoIntn(len(localePool))]
}

// buildIOSUserAgent ghép UA string theo format chuẩn của IG iOS.
// Ví dụ: Instagram 410.1.0.36.70 (iPhone14,3; iOS 17_4_1; vi_VN; vi; scale=3.00; 1284x2778; 849447290)
func buildIOSUserAgent(dev iOSDevice, ios iOSVersion, app igAppVersion, locale string) string {
	lang := locale[:2] // "vi" from "vi_VN"
	return fmt.Sprintf(
		"Instagram %s (%s; iOS %s; %s; %s; scale=%s; %dx%d; %s) AppleWebKit/420+",
		app.Version, dev.ModelID, ios.Display, locale, lang,
		dev.Scale, dev.ScreenW, dev.ScreenH, app.Build,
	)
}

// newRandomProfile tạo profile với locale ngẫu nhiên (fallback khi không biết country).
func newRandomProfile() *igProfile {
	return newRandomProfileWithLocale(pickLocale())
}

// newRandomProfileWithLocale tạo profile iOS ngẫu nhiên nhưng locale được chỉ định.
// Dùng khi đã biết country code của proxy IP.
func newRandomProfileWithLocale(locale string) *igProfile {
	dev := pickDevice()
	ios := pickIOSVersion(dev.MinIOS, dev.MaxIOS)
	app := pickAppVersion()
	ua := buildIOSUserAgent(dev, ios, app, locale)

	return &igProfile{
		DeviceID:       upperUUID(),
		FamilyDeviceID: upperUUID(),
		WaterfallID:    hex32(),
		RegMachineID:   randBase64URL(24),
		CloudTrustID:   upperUUID()[:8] + upperUUID(),
		PigeonSID:      "UFS-" + upperUUID() + "-1",
		ConnUUID:       hex32(),
		RegFlowID:      newUUIDv4(),
		UserAgent:      ua,
		Locale:         locale,
	}
}

func newUUIDv4() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
