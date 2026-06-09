// Package uabuilder — central User-Agent builder for all reg/verify platforms.
//
// Port semantics 1:1 từ C# `VerifyCloneVIP/FakeInfoBuilder/`:
//   - AndroidUserAgentBuilder.cs        → AndroidUABuilder (FB4A native UA + optional Dalvik prefix)
//   - BrowserAndroidUserAgentBuilder.cs → BrowserUABuilder  (Mozilla Chrome Android UA)
//   - ConfigFileUserAgentBuilder.cs     → ConfigFileUABuilder (random từ pre-built UA list)
//
// Yêu cầu: xem docs/UA_BUILDER_REFACTOR_PLAN.md §A và §E.
//
// Hai toggle quan trọng:
//
//   - UAOptions.AddVirtualSpecs (= MainFormUISettings.AddVirtualSpecAndroid):
//     prepend "Dalvik/2.1.0 (Linux; U; Android <os>; <Model> Build/<build>) "
//     vào trước UA FB4A. Chỉ AndroidUABuilder honor.
//
// BuildUA (MainFormUISettings.BuildUA trước đây là UsingBuildNumFile):
// khi true → dùng AndroidUABuilder để build UA từ Config/DeviceInfo/.
// khi false → dùng pool từ Config/UserAgent/<kind>_UG.txt.
package uabuilder

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// UABuilderKind — loại builder đầu ra (cho debug + logging).
type UABuilderKind string

const (
	KindAndroidApp     UABuilderKind = "android-app"     // AndroidUserAgentBuilder
	KindBrowserAndroid UABuilderKind = "browser-android" // BrowserAndroidUserAgentBuilder
	KindConfigFile     UABuilderKind = "config-file"     // ConfigFileUserAgentBuilder
	KindIOSSafari      UABuilderKind = "ios-safari"      // iOS Safari (không có C# tương đương)
)

// UAOptions là tham số khi gọi UABuilder.Build().
//
// Dùng UAOptions.NewRand() để có rand riêng cho call (tránh race khi gọi từ nhiều goroutine).
type UAOptions struct {
	// Locale của UA (FBLC/<locale>). Empty → "en_US".
	Locale string

	// AddVirtualSpecs prepend Dalvik prefix khi true. Chỉ AndroidUABuilder honor.
	AddVirtualSpecs bool

	// SimBrand override FBCR (carrier). Empty → dùng CarrierPicker(CountryCode) hoặc random.
	SimBrand string

	// CountryCode (2-char ISO, vd "VN", "US") — dùng để pick carrier theo quốc gia IP.
	// CarrierPicker được wire từ fakeinfo.RandomSimProfile.
	CountryCode string

	// Platform tên (s22/s23/s24/s25/s26/s555/s556/s557/android). Builder dùng để
	// chọn pool device match thế hệ máy. Empty → dùng pool generic (android_devices.txt).
	Platform string

	// PinAppVersion + PinBuild (tuỳ chọn) — nếu cả 2 không empty thì dùng cố định
	// thay vì random. Hữu ích khi user muốn force 1 phiên bản FB cụ thể.
	PinAppVersion string
	PinBuild      string

	// PoolKind chỉ định pool FBAV/FBBV source khi builder random version:
	//   - ""    → pool CHUNG (versions_and_builds.txt)
	//   - "reg" → pool REG (versions_and_builds_reg.txt, fallback chung)
	//   - "ver" → pool VER (versions_and_builds_ver.txt, fallback chung)
	// Caller (register/verify s5xx) phải set field này để dùng split pool.
	PoolKind string

	// rand — nếu nil, builder tự seed. Truyền vào nếu caller muốn giữ deterministic.
	rand *rand.Rand
}

// NewRand tạo rand instance cho 1 call (tránh share mutex của math/rand global).
func (o *UAOptions) NewRand() *rand.Rand {
	if o.rand != nil {
		return o.rand
	}
	return rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
}

// UABuildResult chứa UA + metadata về device đã pick.
//
// Caller (register/verify) cần lưu các field Model/OSVersion/... vào account model
// để dùng làm payload sau này (vd FacebookAccountModel.AndroidDeviceModel).
type UABuildResult struct {
	UserAgent     string // chuỗi UA cuối (có thể có Dalvik prefix)
	Kind          UABuilderKind
	Locale        string
	Manufacturer  string
	Brand         string
	Model         string // FBDV/<Model>
	OSVersion     string // FBSV/<OSVersion>
	Density       string
	Width         int
	Height        int
	Carrier       string // FBCR
	CPUArch       string // FBCA
	AppVersion    string // FBAV
	AppBuild      string // FBBV
	BuildID       string // dùng cho Dalvik Build/<BuildID> hoặc Mozilla (Linux; Android <os>; <buildID>)
	ChromeVersion string // chỉ Browser builder
}

// UABuilder là interface chung cho mọi loại UA builder.
type UABuilder interface {
	Build(opts UAOptions) (UABuildResult, error)
	Kind() UABuilderKind
}

// ─── Errors ─────────────────────────────────────────────────────────────────

var (
	// ErrNoDevicesAvailable trả về khi pool device cho platform rỗng.
	ErrNoDevicesAvailable = errors.New("uabuilder: device pool empty")
	// ErrNoVersionsAvailable trả về khi pool app version rỗng.
	ErrNoVersionsAvailable = errors.New("uabuilder: app version pool empty")
)

// ─── Registry: factory per (platform, builder_type) ─────────────────────────

// uaBuilderType matches C# `useragent_type` constructor param:
//
//	0 = AndroidUserAgentBuilder       (FB4A app UA, build từ device_info)
//	1 = ConfigFileUserAgentBuilder   (random từ Android_UG.txt / iOS_UG.txt / Request_UG.txt)
//	2 = BrowserAndroidUserAgentBuilder (Mozilla Chrome Android UA)
const (
	BuilderTypeAndroidApp     = 0
	BuilderTypeConfigFile     = 1
	BuilderTypeBrowserAndroid = 2
)

// registryMu guards configFileSourceFor.
var registryMu sync.RWMutex

// configFileSourceFor maps (platform → ConfigFile UA pool source). Mặc định
// fallback về Android_UG khi platform không có entry riêng.
//
// Dùng kind từ ua_pools.go (UAKindAndroid/UAKindIOS/UAKindRequest)
// — file này không import trực tiếp để tránh circular; ConfigFileUABuilder
// nhận pool source qua function adapter ConfigFileUASource.
var configFileSourceFor = map[string]ConfigFileUASource{}

// ConfigFileUASource là adapter function lấy 1 UA random từ pool (vd RandomUAFromPool).
// Tách function adapter để tránh ConfigFileUABuilder phụ thuộc package fakeinfo.
type ConfigFileUASource func() string

// SetConfigFileSource đăng ký source cho 1 platform. Goi 1 lần lúc init từ
// package fakeinfo (init.go của fakeinfo) — sau đó ResolveBuilder dùng đến.
func SetConfigFileSource(platform string, src ConfigFileUASource) {
	registryMu.Lock()
	defer registryMu.Unlock()
	configFileSourceFor[platform] = src
}

func getConfigFileSource(platform string) ConfigFileUASource {
	registryMu.RLock()
	defer registryMu.RUnlock()
	if src, ok := configFileSourceFor[platform]; ok {
		return src
	}
	return configFileSourceFor[""] // fallback "default"
}

// ResolveBuilder trả về builder phù hợp cho (platform, builderType).
//
// Convention:
//   - builderType=0 (AndroidApp): dùng AndroidUABuilder cho mọi platform Android.
//   - builderType=1 (ConfigFile): dùng pool UA pre-built tương ứng kind.
//   - builderType=2 (BrowserAndroid): dùng BrowserUABuilder (cho WebAndroid, Token).
//
// Platform được forward vào AndroidUABuilder để filter device pool theo thế hệ máy.
func ResolveBuilder(platform string, builderType int) (UABuilder, error) {
	switch builderType {
	case BuilderTypeAndroidApp:
		return &AndroidUABuilder{}, nil
	case BuilderTypeConfigFile:
		src := getConfigFileSource(platform)
		if src == nil {
			return nil, fmt.Errorf("uabuilder: no ConfigFile UA source registered for platform=%q", platform)
		}
		return &ConfigFileUABuilder{Source: src}, nil
	case BuilderTypeBrowserAndroid:
		return &BrowserUABuilder{}, nil
	default:
		return nil, fmt.Errorf("uabuilder: unknown builderType=%d (expected 0|1|2)", builderType)
	}
}

// MustResolve là helper panic-on-error cho test/init code.
func MustResolve(platform string, builderType int) UABuilder {
	b, err := ResolveBuilder(platform, builderType)
	if err != nil {
		panic(err)
	}
	return b
}

// ─── CarrierPicker registry ──────────────────────────────────────────────────

// CarrierPickerFunc chọn carrier phù hợp theo countryCode (ISO-2, vd "VN", "US").
// Trả "" nếu không tìm được → caller fallback carriers.txt random.
type CarrierPickerFunc func(countryCode string) string

var (
	carrierPickerMu sync.RWMutex
	carrierPicker   CarrierPickerFunc
)

// SetCarrierPicker đăng ký picker (gọi 1 lần từ fakeinfo.init()).
func SetCarrierPicker(fn CarrierPickerFunc) {
	carrierPickerMu.Lock()
	defer carrierPickerMu.Unlock()
	carrierPicker = fn
}

// GetCarrierPicker trả về picker đã đăng ký, hoặc nil.
func GetCarrierPicker() CarrierPickerFunc {
	carrierPickerMu.RLock()
	defer carrierPickerMu.RUnlock()
	return carrierPicker
}
