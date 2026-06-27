package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"HVRIns/internal/cookie"
	emailrent "HVRIns/internal/email/rent"
	emailtemp "HVRIns/internal/email/temp"
	"HVRIns/internal/fbdata"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	androidreg "HVRIns/internal/instagram/register/android"
	"HVRIns/internal/proxy"
	appsettings "HVRIns/internal/settings"
	"HVRIns/internal/settings/adapter"
)

// === SETTINGS ===

// GeneralConfig cấu hình chung — mapping từ frontend GeneralConfig
type GeneralConfig struct {
	ThreadRequest     int               `json:"threadRequest"`
	DelayRequest      int               `json:"delayRequest"`
	DelayThread       int               `json:"delayThread"`
	ApiCheckIp        int               `json:"apiCheckIp"`
	ThreadCheckInfo   int               `json:"threadCheckInfo"`
	LoginPlatform     string            `json:"loginPlatform"`
	LoginMethod       int               `json:"loginMethod"`
	SaveRunColumn     bool              `json:"saveRunColumn"`
	BackupDB          bool              `json:"backupDB"`
	CloseAfterDone    bool              `json:"closeAfterDone"`
	AccountSourcePath string            `json:"accountSourcePath"`
	AccountSource     string            `json:"accountSource"` // "folder" | "api"
	CloneHVUsername   string            `json:"cloneHvUsername"`
	CloneHVPassword   string            `json:"cloneHvPassword"`
	CloneHVProductID  string            `json:"cloneHvProductId"`
	CloneHVAmount     int               `json:"cloneHvAmount"`
	CaptchaProvider   string            `json:"captchaProvider"`
	CaptchaKeys       map[string]string `json:"captchaKeys"`
	IpProvider        string            `json:"ipProvider"`
	CheckIpBeforeRun  bool              `json:"checkIpBeforeRun"`
	DelayChangeIp     int               `json:"delayChangeIp"`
	// Locale & Device Fake
	LocaleFake    string `json:"localeFake"`
	DeepFakeInApi bool   `json:"deepFakeInApi"`
	// Cookie Initial
	CookieUse        bool   `json:"cookieUse"`
	CookieLimit      bool   `json:"cookieLimit"`
	CookieLimitCount int    `json:"cookieLimitCount"`
	CookieMode       string `json:"cookieMode"`
	// UA Custom
	UaAddSpecs   bool `json:"uaAddSpecs"`
	UaBuildFile  bool `json:"uaBuildFile"`
	UaCustomType int  `json:"uaCustomType"`
	// Sim Network
	SimNetworkMode string `json:"simNetworkMode"`
	SimNetworkType string `json:"simNetworkType"`
}

// IpConfig cấu hình IP — mapping từ frontend IpConfig
type IpConfig struct {
	ProxyList               string `json:"proxyList"`
	ProxyStickyList         string `json:"proxyStickyList"`
	ProxyActiveTab          string `json:"proxyActiveTab"` // "standard" | "sticky" — tab đang chọn trên UI
	ProxyType               string `json:"proxyType"`
	FptKeys                 string `json:"fptKeys"`
	XproxyServiceUrl        string `json:"xproxyServiceUrl"`
	XproxyType              string `json:"xproxyType"`
	XproxyList              string `json:"xproxyList"`
	XproxyThreadPerIp       int    `json:"xproxyThreadPerIp"`
	XproxyRunType           string `json:"xproxyRunType"`
	TinsoftKeys             string `json:"tinsoftKeys"`
	TinsoftThreadPerIp      int    `json:"tinsoftThreadPerIp"`
	ShoplikeKeys            string `json:"shoplikeKeys"`
	ShoplikeThreadPerIp     int    `json:"shoplikeThreadPerIp"`
	NetproxyKeys            string `json:"netproxyKeys"`
	NetproxyThreadPerIp     int    `json:"netproxyThreadPerIp"`
	MinproxyKeys            string `json:"minproxyKeys"`
	MinproxyThreadPerIp     int    `json:"minproxyThreadPerIp"`
	NetproxyGbKey           string `json:"netproxyGbKey"`
	ProxyPopularKeys        string `json:"proxyPopularKeys"`
	ProxyPopularThreadPerIp int    `json:"proxyPopularThreadPerIp"`
	ProxyPopularAccessToken string `json:"proxyPopularAccessToken"`
	ProxyFarmKeys           string `json:"proxyFarmKeys"`
	ProxyFarmThreadPerIp    int    `json:"proxyFarmThreadPerIp"`
	ProxyFarmAccessToken    string `json:"proxyFarmAccessToken"`
	// Proxy riêng cho Reg
	UseVerifyProxyForReg bool   `json:"useVerifyProxyForReg"`
	RegIpProvider        string `json:"regIpProvider"`
	RegProxyList         string `json:"regProxyList"`
	RegProxyStickyList   string `json:"regProxyStickyList"`
	RegProxyActiveTab    string `json:"regProxyActiveTab"`
	RegProxyType         string `json:"regProxyType"`
	// Retry & Delay
	ProxyRetry   int `json:"proxyRetry"`   // số lần retry khi proxy lỗi (0 = không retry)
	ProxyDelayMs int `json:"proxyDelayMs"` // delay ms trước khi đổi proxy
}

// activeProxyList trả về danh sách proxy verify.
// Session proxy (có _session-, -zone-, sid-) tự nhận diện per-line — không cần tab.
func activeProxyList(ip IpConfig) string {
	return ip.ProxyList
}

// activeRegProxyList trả về danh sách proxy reg.
func activeRegProxyList(ip IpConfig) string {
	return ip.RegProxyList
}

// SettingsData gói cả GeneralConfig + IpConfig
type SettingsData struct {
	General GeneralConfig `json:"general"`
	Ip      IpConfig      `json:"ip"`
}

// SaveSettings lưu cài đặt chung vào active profile và general.json
func (a *App) SaveSettings(data SettingsData) string {
	const settingsDir = "Config/Settings"

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "Lỗi marshal: " + err.Error()
	}
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return "Lỗi tạo thư mục: " + err.Error()
	}

	// 1. Sync vào active profile (profile-aware)
	var ls adapter.LegacySettingsData
	if err := json.Unmarshal(b, &ls); err != nil {
		slog.Warn("SaveSettings: sync appSettings thất bại", "err", err)
	}
	a.settingsMu.Lock()
	if p := a.appSettings.GetActiveProfile(); p != nil {
		adapter.ApplySettingsToProfile(p, &a.appSettings.Global, ls)
	}
	// Dừng watcher cũ nếu đang chạy
	if a.watcherCancel != nil {
		a.watcherCancel()
		a.watcherCancel = nil
	}
	app := a.appSettings
	a.settingsMu.Unlock()

	// 2. Persist AppSettings (chứa profile vừa cập nhật)
	if err := appsettings.Save(settingsDir, app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}

	// 3. Ghi general.json để backward-compat — fail ở đây nghĩa là settings không persist,
	// PHẢI return error rõ ràng để frontend không hiển thị "Đã lưu" giả.
	// 0600 vì file chứa proxy creds, captcha keys → chỉ owner đọc được.
	if err := os.WriteFile(filepath.Join(settingsDir, "general.json"), b, 0600); err != nil {
		slog.Error("SaveSettings: ghi general.json thất bại", "err", err)
		return "Lỗi ghi general.json: " + err.Error()
	}

	// 4. Sync AccountSourcePath → interaction.json VerifySourceFolderPath.
	// 2 field là 1 giá trị duy nhất — user edit ở General sẽ hiện ở Interaction.
	// Sync fail không fatal (general.json đã lưu) nhưng phải log để debug khi 2 UI hiện path khác nhau.
	interactionPath := filepath.Join(settingsDir, "interaction.json")
	if ib, err := os.ReadFile(interactionPath); err == nil {
		var ic InteractionConfig
		if uerr := json.Unmarshal(ib, &ic); uerr == nil {
			ic.VerifySourceFolderPath = data.General.AccountSourcePath
			if patched, merr := json.MarshalIndent(ic, "", "  "); merr == nil {
				if werr := os.WriteFile(interactionPath, patched, 0600); werr != nil {
					slog.Warn("SaveSettings: sync interaction.json (write) thất bại", "err", werr)
				}
			} else {
				slog.Warn("SaveSettings: sync interaction.json (marshal) thất bại", "err", merr)
			}
		} else {
			slog.Warn("SaveSettings: sync interaction.json (unmarshal) thất bại", "err", uerr)
		}
	} else if !os.IsNotExist(err) {
		slog.Warn("SaveSettings: đọc interaction.json thất bại", "err", err)
	}

	// 5. Invalidate proxy cache — settings có thể chứa IpProvider/keys → force recreate mgr
	a.InvalidateProxyCache()

	return "OK"
}

// LoadSettings đọc cài đặt chung từ general.json, fallback a.appSettings.
// First-run (file chưa tồn tại) → áp dụng full defaults (bool + string).
// Subsequent loads → chỉ fill string fields rỗng, giữ user's bool choice.
func (a *App) LoadSettings() SettingsData {
	generalPath := filepath.Join("Config/Settings", "general.json")

	// Đọc từ general.json nếu tồn tại — user đã save 1 lần
	if b, err := os.ReadFile(generalPath); err == nil {
		var data SettingsData
		if json.Unmarshal(b, &data) == nil {
			applyGeneralStringDefaults(&data.General) // chỉ fill string rỗng
			applyIpStringDefaults(&data.Ip)           // normalize proxyType rỗng → "http"
			return data
		}
	}

	// First-run: file chưa tồn tại → áp dụng FULL defaults.
	a.settingsMu.RLock()
	ls := adapter.ToLegacySettings(a.appSettings)
	a.settingsMu.RUnlock()
	var data SettingsData
	if b, err := json.Marshal(ls); err == nil {
		if err := json.Unmarshal(b, &data); err != nil {
			slog.Warn("LoadSettings fallback: unmarshal thất bại", "err", err)
		}
	}
	applyGeneralFullDefaults(&data.General) // first-run: set cả bool defaults
	return data
}

// applyIpStringDefaults fill các proxy type rỗng về mặc định "http".
// Phải gọi sau khi unmarshal để tránh radio button không được tích khi load lại.
func applyIpStringDefaults(ip *IpConfig) {
	if ip == nil {
		return
	}
	if ip.ProxyType == "" {
		ip.ProxyType = "http"
	}
	if ip.XproxyType == "" {
		ip.XproxyType = "http"
	}
	if ip.RegProxyType == "" {
		ip.RegProxyType = "http"
	}
}

// applyGeneralStringDefaults chỉ fill string fields rỗng — giữ user's bool choice.
// Dùng khi general.json đã tồn tại (user đã save 1 lần).
func applyGeneralStringDefaults(c *GeneralConfig) {
	if c == nil {
		return
	}
	if strings.TrimSpace(c.LocaleFake) == "" {
		c.LocaleFake = "match-ip"
	}
	if strings.TrimSpace(c.SimNetworkMode) == "" {
		c.SimNetworkMode = "match-ip"
	}
	if strings.TrimSpace(c.SimNetworkType) == "" {
		c.SimNetworkType = "LTE"
	}
	if strings.TrimSpace(c.LoginPlatform) == "" {
		c.LoginPlatform = "facebook"
	}
	// LoginMethod: với Facebook, 0 không match option (chỉ có value=6 "Cookie mobile").
	// Nếu platform facebook và loginMethod chưa valid → set 6.
	if c.LoginPlatform == "facebook" && c.LoginMethod == 0 {
		c.LoginMethod = 6
	}
}

// applyGeneralFullDefaults áp dụng full defaults (bool + string) cho first-run.
// Chuẩn C# defaults — user khởi động app lần đầu tick sẵn các option quan trọng.
func applyGeneralFullDefaults(c *GeneralConfig) {
	if c == nil {
		return
	}
	applyGeneralStringDefaults(c)
	c.DeepFakeInApi = true       // C#: deep locale trong API payload
	c.UaAddSpecs = true          // C#: virtual spec Android
	c.LoginPlatform = "facebook" // default platform
	c.LoginMethod = 6            // "Cookie mobile" — facebook chỉ có 1 option (value=6)
	// KeepIPSuccess, UaBuildFile giữ zero (user tuỳ chọn)
}

// getSharedProxyManager trả về proxy.Manager dùng chung giữa REG và VER.
//
// Fast path (99% case khi đang chạy batch): RLock + version check, return cached.
// Tránh LoadSettings() disk I/O mỗi worker call — 50 workers × 5ms = 250ms bottleneck.
//
// Slow path (settings thay đổi): LoadSettings() + rebuild. Trigger bằng atomic
// proxyConfigVersion++ trong SaveSettings/SaveInteractionConfig/SaveIpConfig.
func (a *App) getSharedProxyManager() *proxy.Manager {
	currentVer := a.proxyConfigVersion.Load()

	// ── Fast path: cached + version khớp ───────────────────────────────────
	a.sharedProxyMgrMu.RLock()
	if a.sharedProxyMgr != nil && a.sharedProxyMgrVersion == currentVer {
		mgr := a.sharedProxyMgr
		a.sharedProxyMgrMu.RUnlock()
		return mgr
	}
	a.sharedProxyMgrMu.RUnlock()

	// ── Slow path: cần tạo mới ─────────────────────────────────────────────
	a.sharedProxyMgrMu.Lock()
	defer a.sharedProxyMgrMu.Unlock()

	// Re-check sau khi lấy write lock (double-check pattern) — có thể goroutine
	// khác đã tạo xong trong lúc ta chờ lock.
	currentVer = a.proxyConfigVersion.Load()
	if a.sharedProxyMgr != nil && a.sharedProxyMgrVersion == currentVer {
		return a.sharedProxyMgr
	}

	s := a.LoadSettings()
	key := s.General.IpProvider + "|" + s.Ip.ShoplikeKeys + "|" + s.Ip.TinsoftKeys + "|" +
		s.Ip.NetproxyKeys + "|" + s.Ip.MinproxyKeys + "|" + s.Ip.ProxyFarmKeys + "|" + activeProxyList(s.Ip)

	// Key cũng khớp → chỉ version bump hình thức, dùng lại instance.
	if a.sharedProxyMgr != nil && a.sharedProxyMgrKey == key {
		a.sharedProxyMgrVersion = currentVer
		return a.sharedProxyMgr
	}

	slog.Info("getSharedProxyManager: tạo mới", "provider", s.General.IpProvider, "shoplikeKeys_len", len(s.Ip.ShoplikeKeys))
	a.sharedProxyMgr = proxy.NewManager(proxy.ManagerConfig{
		Provider:         s.General.IpProvider,
		ProxyList:        activeProxyList(s.Ip),
		TinsoftKeys:      s.Ip.TinsoftKeys,
		TinsoftThreads:   s.Ip.TinsoftThreadPerIp,
		ShoplikeKeys:     s.Ip.ShoplikeKeys,
		ShoplikeThreads:  s.Ip.ShoplikeThreadPerIp,
		NetproxyKeys:     s.Ip.NetproxyKeys,
		NetproxyThreads:  s.Ip.NetproxyThreadPerIp,
		MinproxyKeys:     s.Ip.MinproxyKeys,
		MinproxyThreads:  s.Ip.MinproxyThreadPerIp,
		ProxyfarmKeys:    s.Ip.ProxyFarmKeys,
		ProxyfarmThreads: s.Ip.ProxyFarmThreadPerIp,
	})
	a.sharedProxyMgrKey = key
	a.sharedProxyMgrVersion = currentVer
	return a.sharedProxyMgr
}

// getRegProxyManager trả về proxy.Manager cho luồng Register.
// Nếu useVerifyProxyForReg=true → trả về sharedProxyMgr của verify.
// Ngược lại → tạo/cache manager riêng từ reg proxy config.
func (a *App) getRegProxyManager() *proxy.Manager {
	s := a.LoadSettings()
	if s.Ip.UseVerifyProxyForReg || s.Ip.RegIpProvider == "" || s.Ip.RegIpProvider == "none" {
		return a.getSharedProxyManager()
	}

	currentVer := a.proxyConfigVersion.Load()

	a.regProxyMgrMu.RLock()
	if a.regProxyMgr != nil && a.regProxyMgrVersion == currentVer {
		mgr := a.regProxyMgr
		a.regProxyMgrMu.RUnlock()
		return mgr
	}
	a.regProxyMgrMu.RUnlock()

	a.regProxyMgrMu.Lock()
	defer a.regProxyMgrMu.Unlock()

	currentVer = a.proxyConfigVersion.Load()
	if a.regProxyMgr != nil && a.regProxyMgrVersion == currentVer {
		return a.regProxyMgr
	}

	key := s.Ip.RegIpProvider + "|" + s.Ip.ShoplikeKeys + "|" + s.Ip.TinsoftKeys + "|" +
		s.Ip.NetproxyKeys + "|" + s.Ip.MinproxyKeys + "|" + s.Ip.ProxyFarmKeys + "|" + activeRegProxyList(s.Ip)

	if a.regProxyMgr != nil && a.regProxyMgrKey == key {
		a.regProxyMgrVersion = currentVer
		return a.regProxyMgr
	}

	slog.Info("getRegProxyManager: tạo mới", "provider", s.Ip.RegIpProvider)
	a.regProxyMgr = proxy.NewManager(proxy.ManagerConfig{
		Provider:         s.Ip.RegIpProvider,
		ProxyList:        activeRegProxyList(s.Ip),
		TinsoftKeys:      s.Ip.TinsoftKeys,
		TinsoftThreads:   s.Ip.TinsoftThreadPerIp,
		ShoplikeKeys:     s.Ip.ShoplikeKeys,
		ShoplikeThreads:  s.Ip.ShoplikeThreadPerIp,
		NetproxyKeys:     s.Ip.NetproxyKeys,
		NetproxyThreads:  s.Ip.NetproxyThreadPerIp,
		MinproxyKeys:     s.Ip.MinproxyKeys,
		MinproxyThreads:  s.Ip.MinproxyThreadPerIp,
		ProxyfarmKeys:    s.Ip.ProxyFarmKeys,
		ProxyfarmThreads: s.Ip.ProxyFarmThreadPerIp,
	})
	a.regProxyMgrKey = key
	a.regProxyMgrVersion = currentVer
	return a.regProxyMgr
}

// InvalidateProxyCache bump version để lần getSharedProxyManager kế force recreate
// nếu key thay đổi. Gọi từ SaveSettings/SaveInteractionConfig/SaveIpConfig.
func (a *App) InvalidateProxyCache() {
	a.proxyConfigVersion.Add(1)
}

// === INTERACTION CONFIG (Thiết lập chạy) ===

// PlatformUAConfig — cấu hình UA riêng cho từng API platform (reg/verify).
// Cho phép mỗi platform dùng bộ UA settings độc lập thay vì dùng chung global.
type PlatformUAConfig struct {
	UseOriginalUA         bool `json:"useOriginalUA"`
	AddVirtualSpecAndroid bool `json:"addVirtualSpecAndroid"`
	BuildUA               bool `json:"buildUA"`
	// ReplaceCarrier — chỉ có hiệu lực khi UseOriginalUA=true.
	ReplaceCarrier bool `json:"replaceCarrier"`
	// TrackingID — dùng cho SimulatePlatformUA preview; giá trị thực từ InteractionConfig.TrackingIDReg/Ver.
	TrackingID bool `json:"trackingID"`
	// UaPoolKey — override pool UA riêng cho platform này ("" = dùng global).
	UaPoolKey string `json:"uaPoolKey"`
	// Kind — "reg" hoặc "ver" — dùng để pick pool FBAV split đúng nguồn khi BuildUA=true.
	// Frontend SET khi gọi SimulatePlatformUA: regPlatformCfg → "reg", verPlatformCfg → "ver".
	Kind string `json:"kind"`
}

// InteractionConfig cấu hình chạy — mapping từ frontend VerifyConfig
type InteractionConfig struct {
	VerifyEnabled       bool   `json:"verifyEnabled"`
	MailProvider        string `json:"mailProvider"`
	MailList            string `json:"mailList"`
	CheckLiveDieEnabled bool   `json:"checkLiveDieEnabled"`
	TimeDelayCheck      int    `json:"timeDelayCheck"`
	TimeDelaySendCode   int    `json:"timeDelaySendCode"`
	SendAgainCode       bool   `json:"sendAgainCode"`
	OutputPath          string `json:"outputPath"`
	UaPoolKey           string `json:"uaPoolKey,omitempty"` // loại UA: "android"|"iphone"|"request"

	// ZeusX Hotmail — mua email để verify
	ZeusXApiKey      string `json:"zeusXApiKey"`
	ZeusXAccountCode string `json:"zeusXAccountCode"`

	// DongVanFB — mua email để verify
	DvfbApiKey      string `json:"dvfbApiKey"`
	DvfbAccountType string `json:"dvfbAccountType"` // account_type ID: "1"=HotMail NEW, "5"=Hotmail TRUSTED, ...

	// Store1s — mua email để verify
	Store1sApiKey    string `json:"store1sApiKey"`
	Store1sProductID string `json:"store1sProductId"` // product_id từ store1s.com (vd: "40559", "50510")

	// Mail30s (mailotp.com / mail30s.com) — mua email để verify
	Mail30sApiKey      string `json:"mail30sApiKey"`
	Mail30sProductSlug string `json:"mail30sProductSlug"` // product_slug từ mailotp.com

	// TempMailLol (api.tempmail.lol) — email tạm miễn phí / có API key
	TempMailLolApiKey string `json:"tempMailLolApiKey"` // optional Bearer token, free tier để trống

	// TempMailDomain domain tuỳ chỉnh cho provider đang chọn (backend đọc field này mỗi call).
	// Frontend giữ map riêng per-provider (TempMailDomains) — khi đổi provider, UI ghi slot tương ứng vào đây.
	TempMailDomain string `json:"tempMailDomain"`

	// TempMailDomains map per-provider domain. Frontend bind v-model theo provider đang chọn.
	// Backend chỉ dùng TempMailDomain (slot active) — map này tồn tại để persist giữa session.
	TempMailDomains map[string]string `json:"tempMailDomains,omitempty"`

	// TempMailToken — token/api key user nhập tay cho provider hiện hành (fallback khi
	// provider-specific field rỗng — vd tempMailLolApiKey/priyoEmailApiKey).
	TempMailToken string `json:"tempMailToken,omitempty"`

	// TempMailTokens map per-provider token — persist giữa session.
	TempMailTokens map[string]string `json:"tempMailTokens,omitempty"`

	// MuaMail config (api.muamail.store)
	MuaMailApiKey    string `json:"muaMailApiKey"`
	MuaMailProductID string `json:"muaMailProductId"`

	// UnlimitMail config (unlimitmail.com)
	UnlimitMailApiKey    string `json:"unlimitMailApiKey"`
	UnlimitMailProductID string `json:"unlimitMailProductId"`

	// SPTMail config (api.sptmail.com)
	SptMailApiKey      string `json:"sptMailApiKey"`
	SptMailServiceCode string `json:"sptMailServiceCode"`

	// EmailAPIInfo config (api.emailapi.info / gmail500.com)
	EmailAPIInfoApiKey      string `json:"emailAPIInfoApiKey"`
	EmailAPIInfoProductCode string `json:"emailAPIInfoProductCode"`

	// OtpCheap config (api.otp.cheap)
	OtpCheapApiKey    string `json:"otpCheapApiKey"`
	OtpCheapServiceID string `json:"otpCheapServiceId"`

	// ShopGmail9999 config (shopgmail9999.com)
	ShopGmail9999ApiKey  string `json:"shopGmail9999ApiKey"`
	ShopGmail9999Service string `json:"shopGmail9999Service"`

	// RentGmail config (rentgmail.online)
	RentGmailApiKey   string `json:"rentGmailApiKey"`
	RentGmailPlatform string `json:"rentGmailPlatform"`

	// OtpCodesSms config (otpcodesms.site)
	OtpCodesSmsApiKey    string `json:"otpCodesSmsApiKey"`
	OtpCodesSmsServiceID string `json:"otpCodesSmsServiceId"`

	// Wmemail config (www.wmemail.com)
	WmemailApiKey    string `json:"wmemailApiKey"`
	WmemailCommodity string `json:"wmemailCommodity"`

	// PriyoEmail config (free.priyo.email)
	PriyoEmailApiKey string `json:"priyoEmailApiKey"`

	// OTPHotmailPriority — nguồn đọc OTP ưu tiên cho 7 providers Hotmail OAuth2
	// (zeus-x, dongvanfb, store1s, mail30s, muamail, unlimitmail, wmemail).
	// Giá trị: "dongvan" (default) | "unlimit". Primary fail → fallback reader còn lại.
	OTPHotmailPriority string `json:"otpHotmailPriority"`

	// MailPoolBatch — số email mua batch đầu khi khởi động pool (mặc định 50).
	// Các lần sau khi pool cạn, mỗi luồng tự mua 1 con độc lập.
	MailPoolBatch int `json:"mailPoolBatch"`

	// Timing & Delay (Verify section)
	WaitCode          int `json:"waitCode"`
	WaitMail          int `json:"waitMail"`
	TrySendCode       int `json:"trySendCode"`
	UseMailTimes      int `json:"useMailTimes"`
	DelayConfirmEmail int `json:"delayConfirmEmail"`
	DelayCheckLive    int `json:"delayCheckLive"`
	DelayVeriReg      int `json:"delayVeriReg"`
	// AddMailRetry — số lần retry thêm khi add mail fail (0 = mặc định 2 outer attempts).
	// Mỗi retry gọi lại GetVerifyConfig() → đổi mail provider mid-run nếu user đổi provider.
	AddMailRetry int `json:"addMailRetry"`
	// RetryUnknownNow — sau khi pass 1 xong, tự động verify lại các acc Unknown/Error.
	// Chỉ chạy 1 pass thêm (không recursion). Bật từ UI: checkbox "Verify lại Unknown ngay".
	RetryUnknownNow bool `json:"retryUnknownNow"`

	// API & Logic (Verify section)
	ApiVerifyPlatform string `json:"apiVerifyPlatform"` // "api android"|"api mfb"|"api token"|"api web andr"
	// ApiVerifyPlatforms — multi-version verify. Nếu set (len>0) thì mỗi account verify
	// dùng 1 version round-robin từ list này (resolve 1 lần/account, ổn định suốt account).
	// Rỗng → fallback dùng ApiVerifyPlatform như cũ.
	ApiVerifyPlatforms []string `json:"apiVerifyPlatforms,omitempty"`
	ApiVerifyTokenType string   `json:"apiVerifyTokenType"` // "adspw"|"internal"|""

	// Reg account section
	ApiRegPlatform string `json:"apiRegPlatform"`
	// ApiRegPlatforms — multi-version reg. Nếu set (len>0), mỗi worker slot được gán
	// cố định 1 version theo round-robin (slot1→[0], slot2→[1], ...) suốt đời slot →
	// keep-ip / keep-ua / keep-datr hoạt động y hệt single-version trong từng slot.
	// Rỗng → fallback dùng ApiRegPlatform như cũ.
	ApiRegPlatforms           []string `json:"apiRegPlatforms,omitempty"`
	DelayReg                  int      `json:"delayReg"`
	DelayStep                 int      `json:"delayStep"` // delay giữa các step (ms), dùng cho s561v99
	LeadDomainMail            string   `json:"leadDomainMail"`
	PasswordReg               string   `json:"passwordReg"`
	NameRegLocale             string   `json:"nameRegLocale"`
	RegMode                   string   `json:"regMode"`
	RegModeRotate             bool     `json:"regModeRotate"`
	RegModeRotateMailMinutes  int      `json:"regModeRotateMailMinutes"`
	RegModeRotatePhoneMinutes int      `json:"regModeRotatePhoneMinutes"`
	VerifyAfterReg            bool     `json:"verifyAfterReg"`
	PhoneMailMode             string   `json:"phoneMailMode"`
	FmPhoneCode               bool     `json:"fmPhoneCode"` // C# FmPhoneCode — strip country code, prefix "0"
	UseUGForVerify            bool     `json:"useUGForVerify"`
	RegForVerify              bool     `json:"regForVerify"`

	// Cookie Initial — dùng cho MỌI platform reg (Android, S23, iOS, WebAndroid, MFB...)
	CookieInitialMethod     string `json:"cookieInitialMethod"`     // "file" | "new"
	LimitCookieInitial      bool   `json:"limitCookieInitial"`      // bật giới hạn số lần dùng mỗi cookie
	LimitCookieInitialCount int    `json:"limitCookieInitialCount"` // số lần tối đa mỗi cookie được dùng
	CookieInitialFile       string `json:"cookieInitialFile"`       // đường dẫn file cookie_initial.txt

	// Giới hạn checkpoint — tự dừng reg khi số checkpoint vượt ngưỡng
	LimitCheckpoint      bool `json:"limitCheckpoint"`
	LimitCheckpointCount int  `json:"limitCheckpointCount"`
	DeleteDatrCheckpoint bool `json:"deleteDatrCheckpoint"`

	// Giới hạn tuổi datr — xóa khỏi pool sau N phút kể từ lúc nạp
	LimitDatrAge        bool `json:"limitDatrAge"`
	LimitDatrAgeMinutes int  `json:"limitDatrAgeMinutes"`

	// SaveNewDatr — nếu true, ghi datr mới thu được từ cookie reg vào cookie_initial.txt
	SaveNewDatr bool `json:"saveNewDatr"`

	// Tạo tài khoản tự động
	CreateEnabled    bool   `json:"createEnabled"`
	CreateType       string `json:"createType"`       // "spam" | "tut"
	CreateCookieList string `json:"createCookieList"` // mỗi dòng một cookie
	CreateOutputPath string `json:"createOutputPath"` // thư mục lưu file tài khoản tạo thành công

	// Thư mục kết quả chung (SuccessReg, SuccessVerify, Die...)
	ResultFolderPath string `json:"resultFolderPath"`

	// Split mode: reg và verify chạy độc lập (reg ghi file → verify đọc file)
	SplitMode          bool `json:"splitMode"`
	SplitVerifyThreads int  `json:"splitVerifyThreads"` // số luồng verify riêng (0 = bằng regThreads)

	// RegThreads — số luồng register chạy song song. Trước đây nằm ở GeneralConfig.ThreadRequest,
	// đã chuyển vào InteractionConfig để reg và verify tự cài luồng riêng.
	RegThreads int `json:"regThreads"`

	// Device pool (mid/datr/ig_did) — tái dùng device aged giống datr-pool FB.
	// Harvest mid sau reg thành công → inject vào reg sau → IG thấy "thiết bị có lịch sử".
	// DevicePoolMaxUses: 1 mid dùng tối đa N lần rồi loại (chống IG link nhiều acc/device). 0 → default 2.
	// DevicePoolMinSize: pool phải đủ ≥ N device mới bật inject (tránh trùng mid lúc pool nhỏ). 0 → dùng RegThreads.
	DevicePoolMaxUses int `json:"devicePoolMaxUses"`
	DevicePoolMinSize int `json:"devicePoolMinSize"`

	// AutoRestart — sau N phút, tự động STOP toàn bộ tiến trình + RESET counters + RUN lại từ đầu.
	// Dùng để rotate proxy/datr pool (tránh burn dài).
	AutoRestartEnabled bool `json:"autoRestartEnabled"`
	AutoRestartMinutes int  `json:"autoRestartMinutes"` // mặc định 60 phút nếu enabled mà = 0

	// VerifySourceFolderPath — thư mục chứa file .txt tài khoản cần verify (verify-only mode).
	// Nếu set, RunVerify dùng folder này thay vì settings.General.AccountSourcePath.
	// Mỗi account được pop+xóa khỏi file ngay khi bắt đầu chạy.
	VerifySourceFolderPath string `json:"verifySourceFolderPath"`

	// KeepIPSuccess — Port C# MainFormUISettings.KeepIPSuccess.
	// Sau khi 1 account verify/reg thành công, giữ nguyên IP (proxy session) cho
	// account kế tiếp chạy trên CÙNG worker slot. Fail → release + acquire fresh.
	// Giảm bandwidth proxy + giữ "IP ngon" cho nhiều account liên tiếp.
	KeepIPSuccess bool `json:"keepIpSuccess"`

	// KeepUASuccess — giữ nguyên UA cho slot sau khi reg thành công.
	// Cùng pattern với KeepIPSuccess nhưng cho User-Agent: success → pin UA cho acc kế,
	// fail → UA mới. Giúp FB nhận fingerprint quen khi reg nhiều acc liên tiếp cùng slot.
	KeepUASuccess      bool `json:"keepUaSuccess"`
	KeepDatrSuccess    bool `json:"keepDatrSuccess"`
	KeepInitialSuccess bool `json:"keepInitialSuccess"` // Keep Contact: giữ email/phone của slot sau reg thành công.

	// AddVirtualSpecAndroid — prepend Dalvik/2.1.0 prefix trong UA (C# default true).
	// false → UA chỉ là FB4A blob, không Dalvik prefix.
	AddVirtualSpecAndroid bool `json:"addVirtualSpecAndroid"`

	// BuildUA — khi true dùng AndroidUABuilder để build UA động từ Config/DeviceInfo/.
	// false → dùng pool từ Config/UserAgent/<kind>_UG.txt.
	BuildUA bool `json:"buildUA"`

	// UseOriginalUA — khi true dùng UA gốc cố định theo platform (s555-s559).
	// Loại trừ lẫn nhau với BuildUA và AddVirtualSpecAndroid.
	UseOriginalUA bool `json:"useOriginalUA"`

	// ReplaceCarrier — chỉ có hiệu lực khi UseOriginalUA=true.
	// true (default) → thay FBCR/Viettel bằng nhà mạng khớp IP.
	// false → giữ nguyên carrier gốc trong UA.
	ReplaceCarrier bool `json:"replaceCarrier"`

	// TrackingIDReg/TrackingIDVer — thêm XID/<random16>; vào cuối UA (trước ]) cho Reg / Verify.
	TrackingIDReg bool `json:"trackingIDReg"`
	TrackingIDVer bool `json:"trackingIDVer"`

	// RegPlatformUA / VerifyPlatformUA — UA config riêng theo platform.
	// Key = apiRegPlatform / apiVerifyPlatform (vd "s559", "android").
	// Nếu key tồn tại → override BuildUA/AddVirtualSpecAndroid/UseOriginalUA toàn cục.
	RegPlatformUA    map[string]PlatformUAConfig `json:"regPlatformUA,omitempty"`
	VerifyPlatformUA map[string]PlatformUAConfig `json:"verifyPlatformUA,omitempty"`

	// ═══ Advanced verify options (port C# MainFormUISettings) ═══

	// ReUseEmail — reuse email đã verify success (ArchiveEmailCollection).
	// Sau verify OK → archive email → account kế có thể dùng lại (UsedCount < UseEmailTime).
	ReUseEmail   bool `json:"reUseEmail"`
	UseEmailTime int  `json:"useEmailTime"` // số lần tái dùng tối đa (default 1)

	// FmUserTmpMail — format username tempmail theo login info (phone/email) thay vì random.
	// Port StringUtils.CreateUsernameTmpMailFromLoginInf.
	FmUserTmpMail bool `json:"fmUserTmpMail"`

	// UseProxyTempMail — khi poll temp mail, dùng proxy riêng từ Config/Proxy/proxy_tempmail.txt.
	// Tránh temp mail rate limit IP.
	UseProxyTempMail bool `json:"useProxyTempmail"`

	// UseProxyGmail — khi dùng rent mail provider hỗ trợ proxy (zeus-x, muamail, unlimitmail),
	// pick proxy từ Config/Proxy/proxy_rentmail.txt.
	UseProxyGmail bool `json:"useProxyGmail"`

	// Enable2FA — sau khi verify email thành công, bật 2FA TOTP cho account.
	// Port C# FacebookSecurityFeatureAPIAndroid.TurnOnTwofactor. Trả secret 32-char
	// để user lưu cùng account (NVR|2FA format).
	Enable2FA bool `json:"enable2fa"`

	// GetNewDatrOnLive — sau khi verify Live, dùng token + cookie + UA của account đó
	// gọi GraphQL profile-switcher để lấy datr mới → thêm vào pool + ghi vào Pool file.
	// Hiệu quả hơn button GetNewDatrFromAccounts vì chạy inline, dùng đúng UA của verify.
	GetNewDatrOnLive bool `json:"getNewDatrOnLive"`

	// UploadAvatar — sau khi verify thành công (live), upload ảnh đại diện cho account.
	// Dùng S23 rupload flow: POST rupload.facebook.com → set via Bloks NUX mutation.
	UploadAvatar bool `json:"uploadAvatar"`

	// AvatarFolderPath — thư mục chứa ảnh JPEG/PNG để upload làm avatar.
	// Mỗi account live sẽ pick 1 ảnh ngẫu nhiên từ thư mục này.
	// Mặc định "Config/Avatar" nếu để trống.
	AvatarFolderPath string `json:"avatarFolderPath"`

	// DelayDisplayResult — giây giữ status cuối của account trên UI trước khi
	// fetch account mới vào slot. 0 = không delay (ghi đè ngay, khó đọc).
	// Khuyến nghị 3-5 giây để user đọc được email + status.
	DelayDisplayResult int `json:"delayDisplayResult"`

	// AddInfo — sau verify Live, cập nhật thông tin hồ sơ account.
	AddInfo             bool   `json:"addInfo"`
	AddInfoCity         bool   `json:"addInfoCity"`
	AddInfoHometown     bool   `json:"addInfoHometown"`
	AddInfoSchool       bool   `json:"addInfoSchool"`
	AddInfoCollege      bool   `json:"addInfoCollege"`
	AddInfoWork         bool   `json:"addInfoWork"`
	AddInfoRelationship bool   `json:"addInfoRelationship"`
	AddInfoDataDir      string `json:"addInfoDataDir"`
	AddInfoDelayMs      int    `json:"addInfoDelayMs"`

	// Auto-upload sau khi reg/ver xong — đọc config từ uploadsite.json
	AutoUploadAfterReg    bool `json:"autoUploadAfterReg"`
	AutoUploadAfterVerify bool `json:"autoUploadAfterVerify"`
}

// SaveInteractionConfig lưu cấu hình thiết lập chạy vào active profile và interaction.json
func (a *App) SaveInteractionConfig(data InteractionConfig) string {
	const settingsDir = "Config/Settings"

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "Lỗi marshal: " + err.Error()
	}
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return "Lỗi tạo thư mục: " + err.Error()
	}

	// 1. Lưu vào active profile (profile-aware)
	a.settingsMu.Lock()
	if p := a.appSettings.GetActiveProfile(); p != nil {
		p.Interaction = json.RawMessage(b)
		// Sync legacy fields để runner dùng được ngay
		var lic adapter.LegacyInteractionConfig
		if jsonErr := json.Unmarshal(b, &lic); jsonErr == nil {
			adapter.ApplyInteractionToProfile(p, lic)
		}
	}
	app := a.appSettings
	a.settingsMu.Unlock()

	// 2. Persist AppSettings (chứa profile.Interaction vừa cập nhật)
	if err := appsettings.Save(settingsDir, app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}

	// 3. Ghi interaction.json để backward-compat (runner đọc trực tiếp)
	if err := os.WriteFile(filepath.Join(settingsDir, "interaction.json"), b, 0644); err != nil {
		slog.Warn("SaveInteractionConfig: ghi interaction.json thất bại", "err", err)
	}

	// 4. Sync VerifySourceFolderPath → general.json AccountSourcePath.
	// 2 field là 1 giá trị duy nhất — user edit ở Interaction sẽ hiện ở General.
	if data.VerifySourceFolderPath != "" {
		if gb, err := os.ReadFile(filepath.Join(settingsDir, "general.json")); err == nil {
			var gs SettingsData
			if json.Unmarshal(gb, &gs) == nil {
				if gs.General.AccountSourcePath != data.VerifySourceFolderPath {
					gs.General.AccountSourcePath = data.VerifySourceFolderPath
					if patched, err := json.MarshalIndent(gs, "", "  "); err == nil {
						_ = os.WriteFile(filepath.Join(settingsDir, "general.json"), patched, 0644)
					}
				}
			}
		}
	}

	return "OK"
}

// LoadInteractionConfig đọc cấu hình thiết lập chạy.
// Thứ tự ưu tiên: active profile.Interaction → interaction.json → a.appSettings (backward compat)
// First-run (chưa có config) → full defaults. Subsequent → chỉ string defaults.
func (a *App) LoadInteractionConfig() InteractionConfig {
	// 1. Đọc từ active profile (profile-aware, nguồn chính)
	a.settingsMu.RLock()
	p := a.appSettings.GetActiveProfile()
	var profileInteraction []byte
	if p != nil && len(p.Interaction) > 0 {
		profileInteraction = []byte(p.Interaction)
	}
	a.settingsMu.RUnlock()

	if len(profileInteraction) > 0 {
		var data InteractionConfig
		if json.Unmarshal(profileInteraction, &data) == nil {
			applyInteractionStringDefaults(&data)
			a.migrateThreadCount(&data)
			return data
		}
	}

	// 2. Fallback: interaction.json (migration từ format cũ)
	if b, err := os.ReadFile(filepath.Join("Config/Settings", "interaction.json")); err == nil {
		var data InteractionConfig
		if json.Unmarshal(b, &data) == nil {
			applyInteractionStringDefaults(&data)
			a.migrateThreadCount(&data)
			return data
		}
	}

	// 3. Fallback cuối (first-run): adapter + apply full defaults (bool + string).
	a.settingsMu.RLock()
	lic := adapter.ToLegacyInteraction(a.appSettings)
	a.settingsMu.RUnlock()
	var data InteractionConfig
	if b, err := json.Marshal(lic); err == nil {
		if err := json.Unmarshal(b, &data); err != nil {
			slog.Warn("LoadInteractionConfig fallback: unmarshal thất bại", "err", err)
		}
	}
	applyInteractionFullDefaults(&data)
	a.migrateThreadCount(&data)
	return data
}

// migrateThreadCount fill RegThreads từ general.ThreadRequest khi user chưa migrate
// sang field mới. Trước đây luồng nằm ở GeneralConfig; nay đã chuyển sang InteractionConfig
// để reg và verify tự cài luồng riêng.
func (a *App) migrateThreadCount(c *InteractionConfig) {
	if c == nil || c.RegThreads > 0 {
		return
	}
	t := a.LoadSettings().General.ThreadRequest
	if t > 0 {
		c.RegThreads = t
	} else {
		c.RegThreads = 20
	}
}

// applyInteractionStringDefaults fill string/int defaults cho field rỗng.
// Dùng khi config đã tồn tại (user đã save 1 lần) — giữ bool choice của user.
func applyInteractionStringDefaults(c *InteractionConfig) {
	if c == nil {
		return
	}
	// REG string defaults
	if strings.TrimSpace(c.ApiRegPlatform) == "" {
		c.ApiRegPlatform = "s23" // khớp value option ở <select> (lowercase)
	}
	if strings.TrimSpace(c.LeadDomainMail) == "" {
		c.LeadDomainMail = "@gmail.com,@yahoo.com"
	}
	if strings.TrimSpace(c.NameRegLocale) == "" {
		c.NameRegLocale = "US"
	}
	if strings.TrimSpace(c.RegMode) == "" {
		c.RegMode = "Phone"
	}
	if c.RegModeRotateMailMinutes <= 0 {
		c.RegModeRotateMailMinutes = 360
	}
	if c.RegModeRotatePhoneMinutes <= 0 {
		c.RegModeRotatePhoneMinutes = 360
	}
	if strings.TrimSpace(c.PhoneMailMode) == "" {
		c.PhoneMailMode = "random-normal"
	}
	// Device pool: maxUses default 9 (giống FB) — mỗi mid dùng 9 lần rồi cả pool
	// recycle (xoay lại). minSize giữ 0 = inject ngay khi pool có ≥1 device.
	if c.DevicePoolMaxUses == 0 {
		c.DevicePoolMaxUses = 9
	}
	// VERIFY string defaults
	if strings.TrimSpace(c.ApiVerifyPlatform) == "" {
		c.ApiVerifyPlatform = "api android"
	}
	if strings.TrimSpace(c.ApiVerifyTokenType) == "" {
		c.ApiVerifyTokenType = "adspw"
	}
	if strings.TrimSpace(c.MailProvider) == "" {
		c.MailProvider = "mail1sec"
	}
	// Cookie Initial — chỉ còn 2 method: "file" (mặc định) hoặc "new" (sinh datr nội bộ).
	// Config cũ có method="ck" (đã bỏ khỏi UI) được migrate về "file".
	method := strings.ToLower(strings.TrimSpace(c.CookieInitialMethod))
	if method == "" || method == "ck" {
		c.CookieInitialMethod = "file"
	}
	// Create type
	if strings.TrimSpace(c.CreateType) == "" {
		c.CreateType = "spam"
	}
	// Timing defaults — user muốn tất cả = 1 (nhanh, đơn giản, user tự chỉnh nếu cần)
	if c.WaitCode <= 0 {
		c.WaitCode = 1
	}
	if c.WaitMail <= 0 {
		c.WaitMail = 1
	}
	if c.TrySendCode <= 0 {
		c.TrySendCode = 1
	}
	if c.UseMailTimes <= 0 {
		c.UseMailTimes = 1
	}
	if c.DelayConfirmEmail <= 0 {
		c.DelayConfirmEmail = 1
	}
	if c.DelayCheckLive <= 0 {
		c.DelayCheckLive = 1
	}
	if c.DelayVeriReg <= 0 {
		c.DelayVeriReg = 1
	}
	if c.DelayDisplayResult <= 0 {
		c.DelayDisplayResult = 1
	}
	if c.TimeDelayCheck <= 0 {
		c.TimeDelayCheck = 1
	}
	if c.TimeDelaySendCode <= 0 {
		c.TimeDelaySendCode = 1
	}
	if c.LimitCookieInitialCount <= 0 {
		c.LimitCookieInitialCount = 3
	}
	if c.LimitDatrAgeMinutes <= 0 {
		c.LimitDatrAgeMinutes = 60
	}
}

func applyRegModeRotation(c InteractionConfig, startedAt, now time.Time) InteractionConfig {
	c.RegMode = effectiveRegMode(c, startedAt, now)
	return c
}

func effectiveRegMode(c InteractionConfig, startedAt, now time.Time) string {
	base := strings.TrimSpace(c.RegMode)
	if base == "" {
		base = "Phone"
	}
	if !c.RegModeRotate {
		return base
	}

	phoneMode := strings.EqualFold(base, "Phone")
	mailMode := strings.EqualFold(base, "Mail")
	if !phoneMode && !mailMode {
		return base
	}

	phoneDur := time.Duration(c.RegModeRotatePhoneMinutes) * time.Minute
	mailDur := time.Duration(c.RegModeRotateMailMinutes) * time.Minute
	if phoneDur <= 0 {
		phoneDur = 360 * time.Minute
	}
	if mailDur <= 0 {
		mailDur = 360 * time.Minute
	}

	elapsed := now.Sub(startedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	pos := elapsed % (phoneDur + mailDur)
	if mailMode {
		if pos < mailDur {
			return "Mail"
		}
		return "Phone"
	}
	if pos < phoneDur {
		return "Phone"
	}
	return "Mail"
}

// applyInteractionFullDefaults áp dụng full defaults (bool + string) cho first-run.
// Chuẩn C# defaults — user khởi động app lần đầu tick sẵn các option quan trọng.
func applyInteractionFullDefaults(c *InteractionConfig) {
	if c == nil {
		return
	}
	applyInteractionStringDefaults(c)
	c.VerifyEnabled = true       // tick Verify panel
	c.CheckLiveDieEnabled = true // tick Kiểm tra Live/Die
	c.SendAgainCode = true       // tick Gửi lại code nếu không nhận
	c.VerifyAfterReg = true      // tự động verify sau reg thành công
	c.KeepIPSuccess = true       // C# KeepIPSuccess default — giữ proxy cho acc kế
	c.CreateEnabled = true       // Register panel ON sẵn — user muốn reg là chính
	// UA gốc là default — chỉ khi user tick thủ công mới build động với Dalvik/buildFile
	c.AddVirtualSpecAndroid = false
	c.BuildUA = false
	// ReUseEmail, FmUserTmpMail, UseProxyTempMail, FmPhoneCode, SplitMode...
	// giữ zero (false) — user tuỳ chọn, không tick sẵn.
}

// === Legacy Import ===

// LegacyFieldEntry — một field trong báo cáo mapping từ legacy config
type LegacyFieldEntry struct {
	LegacyKey    string `json:"legacyKey"`
	NewPath      string `json:"newPath"`
	DisplayValue string `json:"displayValue"`
	Status       string `json:"status"` // "ok" | "confirm" | "sensitive" | "unsupported"
	Note         string `json:"note"`
}

// LegacyMappingReport — kết quả phân tích mapping legacy config
type LegacyMappingReport struct {
	MappedOk     []LegacyFieldEntry `json:"mappedOk"`
	NeedsConfirm []LegacyFieldEntry `json:"needsConfirm"`
	Sensitive    []LegacyFieldEntry `json:"sensitive"`
	Unsupported  []LegacyFieldEntry `json:"unsupported"`
	ParseErrors  []string           `json:"parseErrors"`
}

// LegacyParseResult — kết quả trả về từ ParseLegacyConfig
type LegacyParseResult struct {
	Report LegacyMappingReport `json:"report"`
	Error  string              `json:"error"`
}

// ParseLegacyConfig phân tích cặp JSON general + interaction từ tool cũ,
// trả về MappingReport mô tả từng field — KHÔNG lưu, chỉ preview.
func (a *App) ParseLegacyConfig(generalJSON, interactionJSON string) LegacyParseResult {
	var s adapter.LegacySettingsData
	var ic adapter.LegacyInteractionConfig
	var parseErrors []string

	if generalJSON != "" {
		if err := json.Unmarshal([]byte(generalJSON), &s); err != nil {
			parseErrors = append(parseErrors, "general.json: "+err.Error())
		}
	}
	if interactionJSON != "" {
		if err := json.Unmarshal([]byte(interactionJSON), &ic); err != nil {
			parseErrors = append(parseErrors, "interaction.json: "+err.Error())
		}
	}

	if len(parseErrors) > 0 && generalJSON != "" && interactionJSON != "" {
		return LegacyParseResult{Error: strings.Join(parseErrors, "; ")}
	}

	report := adapter.BuildMappingReport(s, ic)
	report.ParseErrors = append(report.ParseErrors, parseErrors...)

	// Convert adapter.MappedField → LegacyFieldEntry
	conv := func(fields []adapter.MappedField) []LegacyFieldEntry {
		out := make([]LegacyFieldEntry, len(fields))
		for i, f := range fields {
			out[i] = LegacyFieldEntry{f.LegacyKey, f.NewPath, f.DisplayValue, string(f.Status), f.Note}
		}
		return out
	}

	return LegacyParseResult{
		Report: LegacyMappingReport{
			MappedOk:     conv(report.MappedOk),
			NeedsConfirm: conv(report.NeedsConfirm),
			Sensitive:    conv(report.Sensitive),
			Unsupported:  conv(report.Unsupported),
			ParseErrors:  report.ParseErrors,
		},
	}
}

// CheckCurrentIPViaProxy kiểm tra IP hiện tại qua proxy đầu tiên trong danh sách Proxy Settings.
// Nếu không có proxy nào cấu hình → check IP trực tiếp (IP thật của máy).
// Kết quả: "IP/country" (vd "1.2.3.4/vn") hoặc chỉ IP nếu không lấy được country.
func (a *App) CheckCurrentIPViaProxy() string {
	// Lấy proxy đầu tiên từ Proxy Settings (IpConfig.ProxyList).
	settings := a.LoadSettings()
	proxyStr := ""
	if list := strings.TrimSpace(activeProxyList(settings.Ip)); list != "" {
		for _, line := range strings.Split(list, "\n") {
			if p := strings.TrimSpace(line); p != "" {
				proxyStr = p
				break
			}
		}
	}

	// Render session ID mới (nếu proxy hỗ trợ rotating) để test IP hiện tại đúng.
	if proxyStr != "" {
		proxyStr = proxy.RenderSessionIfIsProxyServer(proxyStr)
	}

	// 20s tổng: đủ budget cho amazonaws (1-3s) + adspower (6s timeout) + ipify fallback.
	// Proxy rotating pool (iprocket KZ/EU) adspower hay block → cần fall-through.
	// Parent = a.ctx → app shutdown cancel được lookup; trước đây dùng context.Background()
	// khiến CheckIP treo đến hết 20s ngay cả khi user đóng app giữa chừng.
	parent := a.ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()

	ip, err := proxy.CheckIP(ctx, proxyStr, settings.General.ApiCheckIp)
	if err != nil || ip == "" {
		if err == nil {
			return "Lỗi: không lấy được IP qua proxy"
		}
		return "Lỗi: " + err.Error()
	}
	return ip
}

// LoadProxyList đọc nội dung file proxy (proxy_tempmail.txt hoặc proxy_rentmail.txt).
// kind: "tempmail" hoặc "gmail" (kind "gmail" map sang proxy_rentmail.txt để giữ back-compat API).
// Trả "" nếu file chưa tồn tại.
func (a *App) LoadProxyList(kind string) string {
	path := proxyListPath(kind)
	if path == "" {
		return ""
	}
	// Auto-migrate: nếu file mới chưa có nhưng file legacy có → rename.
	if strings.ToLower(strings.TrimSpace(kind)) == "gmail" {
		legacyPath := filepath.Join(filepath.Dir(path), "proxy_gmail.txt")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if _, err2 := os.Stat(legacyPath); err2 == nil {
				_ = os.Rename(legacyPath, path)
			}
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// SaveProxyList lưu text vào proxy list file cạnh exe (Config/Proxy/).
// kind: "tempmail" hoặc "gmail".
func (a *App) SaveProxyList(kind, content string) string {
	path := proxyListPath(kind)
	if path == "" {
		return "kind không hợp lệ (chỉ nhận tempmail/gmail)"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "Lỗi tạo thư mục: " + err.Error()
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "Lỗi lưu: " + err.Error()
	}
	return "OK"
}

// proxyListPath trả về absolute path của file proxy theo kind.
// Tự tạo folder Config/Proxy/ cạnh exe để tương thích với email.LoadTempMailProxies.
func proxyListPath(kind string) string {
	filename := ""
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "tempmail":
		filename = "proxy_tempmail.txt"
	case "gmail":
		filename = "proxy_rentmail.txt"
	default:
		return ""
	}
	return filepath.Join(AppDataDir(), "Config", "Proxy", filename)
}

// GetDefaultUACounts trả về số UA embedded sẵn cho mỗi pool (iphone/android/chrome).
// Frontend hiển thị "X UA" khi textarea pool rỗng — cho user biết app có data mặc định.
func (a *App) GetDefaultUACounts() map[string]int {
	return map[string]int{
		"iphone":  fakeinfo.UAPoolSize(fakeinfo.UAKindIOS),     // FBIOS iPhone UAs
		"android": fakeinfo.UAPoolSize(fakeinfo.UAKindAndroid), // FB4A Android UAs
		"chrome":  53,                                          // Chrome versions (53 entries)
	}
}

// GetDefaultUAContent trả về toàn bộ nội dung UA pool mặc định cho 1 kind,
// join bằng "\n" — frontend fill vào textarea khi user chưa nhập/override.
// kind: "iphone" | "android" | "chrome".
// "chrome" hiện chưa có file embed riêng → trả về rỗng (fallback placeholder UI).
func (a *App) GetDefaultUAContent(kind string) string {
	var k fakeinfo.UAPoolKind
	switch kind {
	case "iphone":
		k = fakeinfo.UAKindIOS
	case "android":
		k = fakeinfo.UAKindAndroid
	default:
		return ""
	}
	list := fakeinfo.UAPoolAll(k)
	return strings.Join(list, "\n")
}

// GetDefaultCookiePaths trả về đường dẫn mặc định của cookie folder cho frontend hiển thị.
// Frontend dùng để show placeholder nếu user chưa chọn file cookie initial.
// GetDefaultCookiePaths trả về absolute paths (cạnh exe) cho cookie files.
// Tự seed data mẫu từ embedded (cookie_initial.txt + datr_pool.txt) nếu
// chưa tồn tại — user chạy máy mới là có pool datr sẵn sàng, không cần
// paste thủ công.
func (a *App) GetDefaultCookiePaths() map[string]string {
	initialPath := defaultCookieInitialPath()
	dir := filepath.Dir(initialPath)

	// Seed cả 2 file nếu chưa có — ship datr mẫu sẵn trong exe (embedded).
	cookie.SeedInitialIfMissing(initialPath)

	return map[string]string{
		"dir":     dir,
		"initial": initialPath,
	}
}

func defaultCookieInitialPath() string {
	absDir := filepath.Join(AppDataDir(), cookie.DefaultDir)
	if err := os.MkdirAll(absDir, 0755); err == nil {
		return filepath.Join(absDir, cookie.InitialFilename)
	}
	return cookie.DefaultInitialPath()
}

func defaultCookieDir() string {
	absDir := filepath.Join(AppDataDir(), cookie.DefaultDir)
	if err := os.MkdirAll(absDir, 0755); err == nil {
		return absDir
	}
	return cookie.DefaultDir
}

func resolveCookieInitialPath(path string) string {
	path = strings.Trim(strings.TrimSpace(path), `"'`)
	if path == "" {
		return defaultCookieInitialPath()
	}
	clean := filepath.Clean(path)
	isRootedNoVolume := filepath.VolumeName(clean) == "" &&
		(strings.HasPrefix(clean, `\`) || strings.HasPrefix(clean, `/`)) &&
		!strings.HasPrefix(clean, `\\`)
	if isRootedNoVolume {
		rel := strings.TrimLeft(clean, `\/`)
		if wd, err := os.Getwd(); err == nil {
			candidate := filepath.Join(wd, rel)
			if _, statErr := os.Stat(candidate); statErr == nil {
				return candidate
			}
		}
		return filepath.Join(AppDataDir(), rel)
	}
	if filepath.IsAbs(clean) {
		return clean
	}
	if _, err := os.Stat(clean); err == nil {
		return clean
	}
	candidate := filepath.Join(AppDataDir(), clean)
	if _, statErr := os.Stat(candidate); statErr == nil {
		return candidate
	}
	return clean
}

func countCookieInitialDatrLines(path string) (int, error) {
	path = resolveCookieInitialPath(path)
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		datr := cookie.ExtractDatr(line)
		if datr == "" {
			datr = line
		}
		if datr != "" && !strings.HasPrefix(datr, "_") && !strings.HasPrefix(datr, "-") {
			count++
		}
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

func (a *App) GetCookieInitialStatus(path string) map[string]interface{} {
	resolved := defaultCookieInitialPath()
	status := map[string]interface{}{
		"path":   resolved,
		"exists": false,
		"count":  0,
		"error":  "",
	}
	info, err := os.Stat(resolved)
	if err != nil {
		status["error"] = "File không tồn tại: " + resolved
		return status
	}
	if info.IsDir() {
		status["error"] = "Đường dẫn là thư mục, cần chọn file: " + resolved
		return status
	}
	status["exists"] = true
	count, err := countCookieInitialDatrLines(resolved)
	if err != nil {
		status["error"] = err.Error()
		return status
	}
	status["count"] = count
	return status
}

// GetDatrPoolSize trả về số datr đang có trong in-memory pool (androidreg.SharedPool).
// Trả về 0 khi chưa có run nào khởi động pool.
func (a *App) GetDatrPoolSize() int {
	p := androidreg.SharedPool
	if p == nil {
		return 0
	}
	return p.Size()
}

// GetPoolFileSaveCount trả về số datr đã ghi vào Pool file trong run hiện tại.
// Reset về 0 mỗi khi bắt đầu RunRegister mới.
func (a *App) GetPoolFileSaveCount() int {
	return int(a.poolFileSaved.Load())
}

func (a *App) OpenCookieInitialFile(path string) string {
	resolved := defaultCookieInitialPath()
	if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
		return "Không tạo được thư mục: " + err.Error()
	}
	if _, err := os.Stat(resolved); os.IsNotExist(err) {
		if err := os.WriteFile(resolved, []byte(""), 0600); err != nil {
			return "Không tạo được file: " + err.Error()
		}
	}
	if err := exec.Command("notepad", resolved).Start(); err != nil {
		return "Không mở được file: " + err.Error()
	}
	return "OK"
}

// FbAppStatus — trạng thái dataset FB app version cho frontend.
// path: đường dẫn file override user đang dùng (hoặc default).
// count: số version đang active (default + override sau merge).
// overrideActive: true nếu file user có dữ liệu hợp lệ và đang override embed.
type FbAppStatus struct {
	Path           string `json:"path"`
	Count          int    `json:"count"`
	OverrideActive bool   `json:"overrideActive"`
}

// GetFbAppStatus trả về trạng thái hiện tại của FB versions dataset.
// Frontend hiển thị số version active + đường dẫn file để user biết có override chưa.
func (a *App) GetFbAppStatus() FbAppStatus {
	return FbAppStatus{
		Path:           fbdata.DefaultVersionsAndBuildsPath(),
		Count:          fbdata.Size(),
		OverrideActive: fbdata.OverrideActive(),
	}
}

// ReloadFbAppVersions ép nạp lại file override từ Config/Fbapp/versions_and_builds.txt.
// Gọi sau khi user save file qua UI hoặc edit file thủ công.
// Trả về count active sau reload — 0 nếu parse fail (fallback về default embed).
func (a *App) ReloadFbAppVersions() int {
	fbdata.Reload("")
	return fbdata.Size()
}

// SaveFbAppVersions ghi text user nhập từ UI vào Config/Fbapp/versions_and_builds.txt rồi reload.
// text: nội dung file, mỗi dòng "version|build".
// Trả về message: "OK (N versions)" nếu thành công, "Lỗi: ..." nếu fail.
func (a *App) SaveFbAppVersions(text string) string {
	if err := fbdata.EnsureDir(); err != nil {
		return "Lỗi tạo thư mục: " + err.Error()
	}
	path := fbdata.DefaultVersionsAndBuildsPath()
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		return "Lỗi ghi file: " + err.Error()
	}
	fbdata.Reload("")
	return fmt.Sprintf("OK (%d versions)", fbdata.Size())
}

// UAPoolStatus — trạng thái 1 UA pool cho frontend hiển thị.
type UAPoolStatus struct {
	Kind           string `json:"kind"`           // "android" | "ios" | "request"
	Path           string `json:"path"`           // Config/UserAgent/<kind>_UG.txt
	Count          int    `json:"count"`          // số UA active
	OverrideActive bool   `json:"overrideActive"` // đang dùng file user?
}

// GetUAPoolsStatus trả về trạng thái 3 pool cho frontend.
func (a *App) GetUAPoolsStatus() []UAPoolStatus {
	kinds := []fakeinfo.UAPoolKind{
		fakeinfo.UAKindAndroid,
		fakeinfo.UAKindIOS,
		fakeinfo.UAKindRequest,
		fakeinfo.UAKindWebChrome,
		fakeinfo.UAKindAndroidMess,
		fakeinfo.UAKindIOSMess,
	}
	out := make([]UAPoolStatus, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, UAPoolStatus{
			Kind:           string(k),
			Path:           fakeinfo.UAOverridePath(k),
			Count:          fakeinfo.UAPoolSize(k),
			OverrideActive: fakeinfo.UAPoolOverrideActive(k),
		})
	}
	return out
}

// SaveUAPool ghi user UA list vào Config/UserAgent/<kind>_UG.txt rồi reload.
// kind: "android" | "ios" | "request" | "webchrome".
// text: nội dung, mỗi dòng 1 UA.
// Trả về message "OK (N UA)" hoặc "Lỗi: ...".
func (a *App) SaveUAPool(kind string, text string) string {
	k := fakeinfo.UAPoolKind(kind)
	path := fakeinfo.UAOverridePath(k)
	if path == "" || filepath.Base(path) == "" {
		return fmt.Sprintf("Lỗi: kind %q không hợp lệ (phải là android|ios|request|webchrome|android_mess|ios_mess)", kind)
	}
	if err := fakeinfo.EnsureUAOverrideDir(); err != nil {
		return "Lỗi tạo thư mục: " + err.Error()
	}
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		return "Lỗi ghi file: " + err.Error()
	}
	fakeinfo.ReloadUAPools()
	return fmt.Sprintf("OK (%d UA)", fakeinfo.UAPoolSize(k))
}

// OpenUAFileInEditor mở file UA pool trong Notepad.
// poolKey: "android" | "iphone" | "request" — UI key khớp với form.uaPoolKey ở frontend.
func (a *App) OpenUAFileInEditor(poolKey string) string {
	k := uaKindFromPoolKey(poolKey)
	relPath := fakeinfo.UAOverridePath(k)
	if relPath == "" {
		return "pool không hợp lệ"
	}
	// Tạo file rỗng nếu chưa tồn tại để Notepad mở được ngay
	if err := fakeinfo.EnsureUAOverrideDir(); err == nil {
		if _, statErr := os.Stat(relPath); os.IsNotExist(statErr) {
			_ = os.WriteFile(relPath, []byte(""), 0644)
		}
	}
	absPath, err := filepath.Abs(relPath)
	if err != nil {
		return "không resolve được path: " + err.Error()
	}
	if err := exec.Command("notepad", absPath).Start(); err != nil {
		return "không mở được: " + err.Error()
	}
	return "OK"
}

// PhoneCountryInfo — 1 entry phone country cho frontend.
type PhoneCountryInfo struct {
	Name        string `json:"name"`
	CountryCode string `json:"countryCode"`
	PhoneCode   string `json:"phoneCode"`
	AreaCode    string `json:"areaCode"`
}

// GetPhoneCountries trả về toàn bộ danh sách phone countries.
// Dùng cho UI dropdown chọn country khi nhập số điện thoại.
func (a *App) GetPhoneCountries() []PhoneCountryInfo {
	list := fakeinfo.PhoneCountries()
	out := make([]PhoneCountryInfo, 0, len(list))
	for _, p := range list {
		out = append(out, PhoneCountryInfo{
			Name: p.Name, CountryCode: p.CountryCode,
			PhoneCode: p.PhoneCode, AreaCode: p.AreaCode,
		})
	}
	return out
}

// LookupPhoneCountry trả về info country từ ISO alpha-2 code.
// Trả về zero struct nếu không tìm thấy (Name = "" flag cho frontend).
func (a *App) LookupPhoneCountry(countryCode string) PhoneCountryInfo {
	p, ok := fakeinfo.LookupPhoneCode(countryCode)
	if !ok {
		return PhoneCountryInfo{}
	}
	return PhoneCountryInfo{
		Name: p.Name, CountryCode: p.CountryCode,
		PhoneCode: p.PhoneCode, AreaCode: p.AreaCode,
	}
}

// ImportLegacyConfig áp dụng cặp JSON general + interaction từ tool cũ vào AppSettings.
// Chỉ gọi sau khi user đã xác nhận ở wizard.
func (a *App) ImportLegacyConfig(generalJSON, interactionJSON string) string {
	var s adapter.LegacySettingsData
	var ic adapter.LegacyInteractionConfig

	if generalJSON != "" {
		if err := json.Unmarshal([]byte(generalJSON), &s); err != nil {
			return "Lỗi parse general.json: " + err.Error()
		}
	} else {
		// Giữ nguyên settings hiện tại
		a.settingsMu.RLock()
		s = adapter.ToLegacySettings(a.appSettings)
		a.settingsMu.RUnlock()
	}

	if interactionJSON != "" {
		if err := json.Unmarshal([]byte(interactionJSON), &ic); err != nil {
			return "Lỗi parse interaction.json: " + err.Error()
		}
	} else {
		// Giữ nguyên interaction hiện tại
		a.settingsMu.RLock()
		ic = adapter.ToLegacyInteraction(a.appSettings)
		a.settingsMu.RUnlock()
	}

	a.settingsMu.Lock()
	p := a.appSettings.GetActiveProfile()
	if p == nil {
		a.settingsMu.Unlock()
		return "Lỗi: không có profile active"
	}
	adapter.ApplySettingsToProfile(p, &a.appSettings.Global, s)
	adapter.ApplyInteractionToProfile(p, ic)
	app := a.appSettings
	a.settingsMu.Unlock()

	if err := appsettings.Save("Config/Settings", app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}
	return "OK"
}

// FetchWeMakeMailDomains gọi API wemakemail.com và trả danh sách domain của tài khoản.
// Trả chuỗi JSON {"plan":"business","free":[...],"paid":[...],"all":[...]} hoặc "ERROR: ..." nếu thất bại.
func (a *App) FetchWeMakeMailDomains(apiKey string) string {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	result, err := emailtemp.FetchWeMakeMailDomains(ctx, apiKey)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	b, _ := json.Marshal(result)
	return string(b)
}

// FetchVietXFDomains gọi API vietxf.com và trả danh sách domain của tài khoản.
// Trả chuỗi JSON {"domains":[...]} hoặc "ERROR: ..." nếu thất bại.
func (a *App) FetchVietXFDomains(apiKey string) string {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	result, err := emailtemp.FetchVietXFDomains(ctx, apiKey)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	b, _ := json.Marshal(result)
	return string(b)
}

// FetchStore1sProducts gọi store1s.com products.php QUA BACKEND (tránh CORS từ webview)
// và trả JSON [{id,name,price,stock}] hoặc "ERROR: ...". Dùng cho dropdown + check tồn kho.
func (a *App) FetchStore1sProducts(apiKey string) string {
	ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
	defer cancel()
	products, err := emailrent.FetchStore1sProducts(ctx, apiKey)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	b, _ := json.Marshal(products)
	return string(b)
}

// FetchMailHVDomains gọi API dulich360.com và trả danh sách domain của tài khoản.
// Trả chuỗi JSON {"plan":"...","free":[...],"paid":[...],"all":[...]} hoặc "ERROR: ..." nếu thất bại.
func (a *App) FetchMailHVDomains(apiKey string) string {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	result, err := emailtemp.FetchMailHVDomains(ctx, apiKey)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	b, _ := json.Marshal(result)
	return string(b)
}

// OpenVersionsAndBuildsFile mở file versions_and_builds_<kind>.txt trong text editor mặc định.
// kind: "reg" → versions_and_builds_reg.txt, "ver" → versions_and_builds_ver.txt, "" → versions_and_builds.txt.
// Auto-tạo file rỗng nếu chưa tồn tại. Trả "OK" hoặc "ERR|..." để frontend hiện thông báo.
func (a *App) OpenVersionsAndBuildsFile(kind string) string {
	var path string
	switch kind {
	case "reg-ios":
		path = filepath.Join("Config", "DeviceInfoIOS", "ios_app_builds_reg.txt")
	case "ver-ios":
		path = filepath.Join("Config", "DeviceInfoIOS", "ios_app_builds_ver.txt")
	case "reg":
		path = filepath.Join("Config", "DeviceInfo", "versions_and_builds_reg.txt")
	case "ver":
		path = filepath.Join("Config", "DeviceInfo", "versions_and_builds_ver.txt")
	default:
		path = filepath.Join("Config", "DeviceInfo", "versions_and_builds.txt")
	}

	// Ensure parent dir + file tồn tại (auto-tạo rỗng nếu chưa có).
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "ERR|Không tạo được thư mục: " + err.Error()
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, nil, 0644); err != nil {
			return "ERR|Không tạo được file: " + err.Error()
		}
	}

	// Absolute path để OS open chính xác (start "" relative path đôi khi resolve sai).
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}

	// Open Explorer ở folder chứa file + HIGHLIGHT file đó:
	// - Windows: explorer.exe /select,<file>
	// - macOS:   open -R <file>  (Reveal in Finder)
	// - Linux:   xdg-open <folder>  (chỉ mở folder, không highlight được)
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", "/select,"+abs)
	case "darwin":
		cmd = exec.Command("open", "-R", abs)
	default:
		cmd = exec.Command("xdg-open", filepath.Dir(abs))
	}
	if err := cmd.Start(); err != nil {
		return "ERR|Không mở được file: " + err.Error()
	}
	go func() { _ = cmd.Wait() }()
	return "OK"
}

// ExportFbVersionPool ghi danh sách FBAV được chọn (với FBBV lookup từ pool) ra file
// trong thư mục result hiện tại + mở Explorer highlight file đó.
//
// kind: "reg" / "ver" — phục vụ naming file output.
// fbavList: list FBAV string user đã chọn trong UI table (vd ["564.0.0.0.17", "563.0.0.42.67"]).
//
// File output: <resultDir>/versions_and_builds_<kind>_<timestamp>.txt với format FBAV|FBBV mỗi dòng.
// Nếu chưa có session đang chạy (resultDir rỗng) → fallback Config/DeviceInfo/.
//
// Trả "OK|<num_written>" hoặc "ERR|<msg>".
func (a *App) ExportFbVersionPool(kind string, fbavList []string) string {
	if len(fbavList) == 0 {
		return "ERR|Chưa chọn FBAV nào"
	}
	if kind != "reg" && kind != "ver" {
		return "ERR|kind phải là 'reg' hoặc 'ver'"
	}

	// Lookup FBBV cho từng FBAV.
	versionMap := a.GetFbVersionMap()
	lines := make([]string, 0, len(fbavList))
	missing := 0
	for _, fbav := range fbavList {
		fbbv, ok := versionMap[fbav]
		if !ok || fbbv == "" {
			missing++
			continue
		}
		lines = append(lines, fbav+"|"+fbbv)
	}
	if len(lines) == 0 {
		return "ERR|Không tìm thấy FBBV cho bất kỳ FBAV nào trong pool"
	}

	// Chọn target dir: ưu tiên result session hiện tại, fallback Config/DeviceInfo/.
	targetDir := a.resultDir()
	if targetDir == "" {
		targetDir = filepath.Join("Config", "DeviceInfo")
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "ERR|Không tạo được thư mục: " + err.Error()
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("versions_and_builds_%s_%s.txt", kind, timestamp)
	outPath := filepath.Join(targetDir, filename)

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return "ERR|Không ghi được file: " + err.Error()
	}

	// Open Explorer + highlight file (giống OpenVersionsAndBuildsFile).
	abs, err := filepath.Abs(outPath)
	if err != nil {
		abs = outPath
	}
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", "/select,"+abs)
	case "darwin":
		cmd = exec.Command("open", "-R", abs)
	default:
		cmd = exec.Command("xdg-open", filepath.Dir(abs))
	}
	if err := cmd.Start(); err == nil {
		go func() { _ = cmd.Wait() }()
	}

	msg := fmt.Sprintf("OK|%d", len(lines))
	if missing > 0 {
		msg += fmt.Sprintf("|missing=%d", missing)
	}
	return msg
}

// GetFbVersionMap trả map FBAV → FBBV từ pool CHUNG (+ split _reg/_ver nếu có).
// Dùng cho UI RegStatsTable export → lookup FBBV theo FBAV đã thống kê.
// Format trả JSON-marshallable: {"564.0.0.0.17": "977893103", ...}.
func (a *App) GetFbVersionMap() map[string]string {
	out := make(map[string]string)
	// Pool chung + 2 pool split — gom hết, FBBV mới override cũ nếu trùng FBAV.
	for _, pool := range [][]fbdata.FbVersion{
		fbdata.Versions(),
		fbdata.VersionsReg(),
		fbdata.VersionsVer(),
	} {
		for _, v := range pool {
			if v.Version != "" && v.Build != "" {
				out[v.Version] = v.Build
			}
		}
	}
	return out
}

// pickFbVersionByKind pick cặp FBAV/FBBV theo kind ("reg"/"ver"/""):
//   - "reg" → versions_and_builds_reg.txt (fallback chung)
//   - "ver" → versions_and_builds_ver.txt (fallback chung)
//   - ""    → versions_and_builds.txt (chung)
func pickFbVersionByKind(kind string) (string, string) {
	switch kind {
	case "reg":
		return fakeinfo.RandomFbVersionReg()
	case "ver":
		return fakeinfo.RandomFbVersionVer()
	default:
		return fakeinfo.RandomFbVersion()
	}
}

// SimulatePlatformUA giả lập 1 UA như khi chạy thực, dùng để xem trước trong popup cài đặt.
// Trả "{note}\n{ua}" — frontend tách dòng đầu làm label, dòng sau làm code display.
// platform có thể là: UI verify string ("api mfb", "api web andr", "api android", "api token")
// HOẶC platform ID nội bộ ("web", "webandroid", "s23", "s561v99", ...).
// Resolve UI string về platform nội bộ trước khi dispatch để switch case match đúng.
func (a *App) SimulatePlatformUA(platform string, uaCfg PlatformUAConfig) string {
	// Resolve UI verify type → internal platform ID
	// CHỈ resolve khi input là UI dropdown string (có space, vd "api android" / "api token" / "api mfb" / "api web andr").
	// Platform ID nội bộ (s518, s564v1s23, webandroid, ...) giữ nguyên — verifyPlatformFromType default
	// trả PlatformWeb cho mọi case không match → sẽ dispatch nhầm sang WebChrome pool gây ra UA iPhone.
	if strings.Contains(platform, " ") {
		if resolved := verifyPlatformFromType(platform); resolved != "" {
			platform = resolved
		}
	}
	countries := []string{"VN", "US", "PH", "ID", "TH", "MY"}
	countryCode := countries[rand.Intn(len(countries))]
	maybeXID := func(ua string) string {
		if uaCfg.TrackingID && ua != "" {
			return appendXIDToUA(ua)
		}
		return ua
	}

	// iOS native (FBIOS) — UseOriginalUA ưu tiên trước: trả UA gốc cố định theo version.
	// Nếu không tick UA Gốc → random từ device pool iOS (FBAN/FBIOS).
	// Early-return ở đây để không rơi xuống nhánh BuildAndroidUA (bug hiện UA FB4A).
	if verifyIsIOS(platform) {
		var ua string
		note := "iOS FBIOS"
		if uaCfg.UseOriginalUA {
			ua = originalUAForPlatform(platform, "")
			note += " · UA Gốc"
		}
		if ua == "" {
			ua = instagram.PlatformVerifyUA(platform, countryCode)
			note += " · " + countryCode
		}
		if ua == "" {
			return "[iOS UA không khả dụng]\nPlatform \"" + platform + "\" chưa đăng ký VerifyUA (FBIOS)."
		}
		if uaCfg.Kind != "" {
			note += " · " + uaCfg.Kind
		}
		if uaCfg.TrackingID {
			note += " · +XID"
		}
		return note + "\n" + maybeXID(ua)
	}

	if uaCfg.UseOriginalUA {
		origCC := countryCode
		note := "UA Gốc · random device · FBCR theo " + countryCode
		if !uaCfg.ReplaceCarrier {
			origCC = ""
			note = "UA Gốc · random device · giữ FBCR/Viettel default"
		}
		ua := originalUAForPlatform(platform, origCC)
		if ua == "" {
			// Fallback: platform random UA per-account (vd s273) — dùng factory.
			// Truyền origCC (= "" khi user không tick "Thay nhà mạng") để factory
			// hiểu được intent: giữ default carrier thay vì random theo country.
			if factoryUA := instagram.PlatformVerifyUA(platform, origCC); factoryUA != "" {
				ua = factoryUA
			}
		}
		if ua == "" {
			return "[UA Gốc không khả dụng]\nPlatform \"" + platform + "\" không có UA gốc cố định."
		}
		if uaCfg.TrackingID {
			note += " · +XID"
		}
		return note + "\n" + maybeXID(ua)
	}

	if uaCfg.BuildUA {
		// Web-based platforms cần Chrome browser UA, không phải FBAN/FB4A.
		switch platform {
		case instagram.PlatformWeb:
			// Đọc từ WebChrome pool (Config/UserAgent/WebChrome_UA.txt) — user-managed.
			ua := fakeinfo.RandomUAFromPool(fakeinfo.UAKindWebChrome)
			size := fakeinfo.UAPoolSize(fakeinfo.UAKindWebChrome)
			if ua == "" {
				return fmt.Sprintf("[WebChrome pool rỗng]\nPaste UA vào %s rồi reload pool.", fakeinfo.UAOverridePath(fakeinfo.UAKindWebChrome))
			}
			note := fmt.Sprintf("Build UA · WebChrome pool (%d UA)", size)
			if uaCfg.TrackingID {
				note += " · +XID"
			}
			return note + "\n" + maybeXID(ua)
		case instagram.PlatformWebAndroid:
			prof := fakeinfo.RandomChromeAndroidProfile()
			note := "Build UA · Chrome Mobile · Android " + prof.AndroidOsVersion
			if uaCfg.TrackingID {
				note += " · +XID"
			}
			return note + "\n" + maybeXID(prof.UserAgent)
		}

		// Default: FB4A native Android UA cho s5xx/s4xx/android/s23/...
		dev := fakeinfo.RandomDeviceProfile()
		locale := fakeinfo.LocaleFromCountry(countryCode)
		sim := fakeinfo.RandomSimProfile(countryCode)
		carrier := sim.OperatorName
		if carrier == "" {
			carrier = fakeinfo.RandomCarrier()
		}
		fbVer, fbBuild := pickFbVersionByKind(uaCfg.Kind)
		ua := fakeinfo.BuildAndroidUAWithOpts(dev, locale, carrier, fbVer, fbBuild, uaCfg.AddVirtualSpecAndroid, false)
		note := "Build UA · " + countryCode + " · " + carrier
		if uaCfg.Kind != "" {
			note += " · pool=" + uaCfg.Kind
		}
		if uaCfg.AddVirtualSpecAndroid {
			note += " · +Dalvik"
		}
		if uaCfg.TrackingID {
			note += " · +XID"
		}
		return note + "\n" + maybeXID(ua)
	}

	// Pool UA
	// FIX: ưu tiên per-platform uaCfg.UaPoolKey (set qua applyBuildUADefault frontend
	// theo từng platform — vd ios562 → "iphone"). Chỉ fallback global cfg.UaPoolKey
	// khi per-platform rỗng (legacy hoặc chưa init).
	poolKey := uaCfg.UaPoolKey
	if poolKey == "" {
		cfg := a.LoadInteractionConfig()
		poolKey = cfg.UaPoolKey
	}
	kind := uaKindFromPoolKey(poolKey)
	ua := fakeinfo.RandomUAFromPool(kind)
	if ua == "" {
		dev := fakeinfo.RandomDeviceProfile()
		locale := fakeinfo.LocaleFromCountry(countryCode)
		carrier := fakeinfo.RandomCarrier()
		fbVer, fbBuild := pickFbVersionByKind(uaCfg.Kind)
		ua = fakeinfo.BuildAndroidUAWithOpts(dev, locale, carrier, fbVer, fbBuild, uaCfg.AddVirtualSpecAndroid, false)
		note := "Pool rỗng → build random · " + countryCode
		if uaCfg.TrackingID {
			note += " · +XID"
		}
		return note + "\n" + maybeXID(ua)
	}
	if uaCfg.AddVirtualSpecAndroid {
		ua = fakeinfo.WrapWithDalvikPrefix(ua)
	}
	note := "Pool " + string(kind)
	if uaCfg.AddVirtualSpecAndroid {
		note += " · +Dalvik"
	}
	if uaCfg.TrackingID {
		note += " · +XID"
	}
	return note + "\n" + maybeXID(ua)
}

// originalUAForPlatformWithSim trả UA gốc + SIM được dùng cho FBCR thay thế.
// Caller dùng SIM này để override profile.Sim trong register/verify → đảm bảo
// HNI/MCC/MNC trong body/headers khớp với FBCR carrier trong UA. Nếu không
// nhất quán: UA nói "Viettel" nhưng body có HNI Vinaphone → fingerprint mismatch.
func originalUAForPlatformWithSim(platform, countryCode string) (string, fakeinfo.SimProfile) {
	base := originalUABaseForPlatform(platform)
	if base == "" {
		return "", fakeinfo.SimProfile{}
	}
	var sim fakeinfo.SimProfile
	if countryCode != "" {
		sim = fakeinfo.RandomSimProfile(countryCode)
		if sim.OperatorName != "" {
			base = reFBCR.ReplaceAllString(base, "FBCR/"+sim.OperatorName)
		}
	}
	return base, sim
}

// mapSimNetworkType convert GUI value → Xfb_connection_type (port C# MainFormUISettings.SimNetworkType).
// 0=WIFI, 1=mobile.LTE, 2=cell.CTRadioAccessTechnologyHSDPA, 3=unknown.
// Input có thể là "WIFI", "LTE", "HSDPA", "unknown", "3G", "4G"... Trả "" để giữ default.
func mapSimNetworkType(t string) string {
	t = strings.TrimSpace(t)
	switch strings.ToUpper(t) {
	case "", "0":
		return "" // không override
	case "WIFI":
		return "WIFI"
	case "LTE", "4G", "MOBILE.LTE", "1":
		return "mobile.LTE"
	case "HSDPA", "3G", "CELL.CTRADIOACCESSTECHNOLOGYHSDPA", "2":
		return "cell.CTRadioAccessTechnologyHSDPA"
	case "UNKNOWN", "3":
		return "unknown"
	}
	return ""
}

// defaultPermanentDir trả về folder chứa permanent phone.txt + mail.txt.
// Port C# PathSingleton.PermanentPhonexMailFolder. Auto-create nếu chưa có.
func defaultPermanentDir() string {
	candidate := filepath.Join(AppDataDir(), "Config", "Permanent")
	if mkErr := os.MkdirAll(candidate, 0755); mkErr == nil {
		return candidate
	}
	homeDir, _ := os.UserHomeDir()
	fallback := filepath.Join(homeDir, "Documents", "HVR", "Config", "Permanent")
	_ = os.MkdirAll(fallback, 0755)
	return fallback
}

// missingPhoneCCMu serialize append vào missing-country-codes log từ nhiều reg goroutines.
