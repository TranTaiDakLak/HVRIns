// simnetwork.go — SIM/carrier info generator
// Mapping từ C#: SimNetworkUtils.GetRandomSimnetworkByCountryCode()
// Data đọc từ Config/SimNetwork/simnetworks.txt và Config/Locales/locales.txt
// qua ReloadOverrides() trong overrides.go.
package fakeinfo

import (
	"math/rand"
	"strings"
	"time"
)

var localeList []string

// SimProfile chứa thông tin SIM network cho device headers
type SimProfile struct {
	MCC          string // Mobile Country Code (452)
	MNC          string // Mobile Network Code (04)
	OperatorName string // Viettel, T-Mobile...
	CountryCode  string // VN, US, GB...
	HNI          string // MCC+MNC concatenated (45204)
}

var simList []SimProfile

// RandomLocale trả về 1 locale ngẫu nhiên từ locales.txt.
// Port C#: RandomUtils.RandomItemInFile(PathSingleton.LocalesFile) khi RandomLocaleOtions=1.
// Fallback "en_US" nếu file empty hoặc list rỗng.
func RandomLocale() string {
	if len(localeList) == 0 {
		return "en_US"
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	return localeList[r.Intn(len(localeList))]
}

// RandomSimProfile trả về SIM ngẫu nhiên. Nếu countryCode != "" → filter theo country.
func RandomSimProfile(countryCode string) SimProfile {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))

	if countryCode != "" {
		cc := strings.ToUpper(countryCode)
		var filtered []SimProfile
		for _, s := range simList {
			if s.CountryCode == cc {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) > 0 {
			return filtered[r.Intn(len(filtered))]
		}
	}

	if len(simList) > 0 {
		return simList[r.Intn(len(simList))]
	}

	// Fallback
	return SimProfile{
		MCC: "310", MNC: "260", OperatorName: "T-Mobile",
		CountryCode: "US", HNI: "310260",
	}
}

// iosConnTypes — pool giá trị X-FB-Connection-Type cho FBIOS (CoreTelephony).
// CELLULAR-ONLY (yêu cầu 2026-05-31): iOS reg/verify/login BẮT BUỘC chạy mạng di
// động để header X-FB-SIM-HNI luôn được gửi. Trên wifi FB iOS KHÔNG gửi sim-hni →
// fingerprint yếu, dễ bị soi. Đã bỏ entry "wifi". Định dạng "mobile.CTRadioAccessTechnology*".
var iosConnTypes = []string{
	"mobile.CTRadioAccessTechnologyLTE",
	"mobile.CTRadioAccessTechnologyLTE",
	"mobile.CTRadioAccessTechnologyLTE",
	"mobile.CTRadioAccessTechnologyNRNSA",
	"mobile.CTRadioAccessTechnologyNRNSA",
	"mobile.CTRadioAccessTechnologyNR",
	"mobile.CTRadioAccessTechnologyWCDMA",
	"mobile.CTRadioAccessTechnologyHSDPA",
}

// RandomIOSConnType trả X-FB-Connection-Type ngẫu nhiên theo định dạng FBIOS.
func RandomIOSConnType() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	return iosConnTypes[r.Intn(len(iosConnTypes))]
}

// LocaleFromCountry trả về locale string từ country code
func LocaleFromCountry(countryCode string) string {
	localeMap := map[string]string{
		"VN": "vi_VN", "US": "en_US", "GB": "en_GB", "DE": "de_DE",
		"FR": "fr_FR", "IT": "it_IT", "ES": "es_ES", "JP": "ja_JP",
		"KR": "ko_KR", "TH": "th_TH", "PH": "en_PH", "ID": "id_ID",
		"MY": "ms_MY", "SG": "en_SG", "TW": "zh_TW", "CN": "zh_CN",
		"CA": "en_CA", "BR": "pt_BR",
	}
	if l, ok := localeMap[strings.ToUpper(countryCode)]; ok {
		return l
	}
	return "en_US"
}
