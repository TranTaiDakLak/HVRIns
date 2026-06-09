// Package adapter — chuyển đổi hai chiều giữa AppSettings mới và legacy config structs.
// Đảm bảo backward-compat: app.go vẫn dùng SettingsData/InteractionConfig như cũ.
package adapter

import "HVRIns/internal/settings/model"

// ─── Legacy type mirrors ───────────────────────────────────────────────────
// Các struct này mirror app.go để tránh import từ package main.
// JSON tags phải khớp chính xác với app.go để roundtrip đúng.

// LegacyGeneralConfig mirrors app.go GeneralConfig
type LegacyGeneralConfig struct {
	ThreadRequest     int               `json:"threadRequest"`
	DelayRequest      int               `json:"delayRequest"`
	DelayThread       int               `json:"delayThread,omitempty"`
	ApiCheckIp        int               `json:"apiCheckIp,omitempty"`
	ThreadCheckInfo   int               `json:"threadCheckInfo"`
	LoginPlatform     string            `json:"loginPlatform"`
	LoginMethod       int               `json:"loginMethod"`
	SaveRunColumn     bool              `json:"saveRunColumn"`
	BackupDB          bool              `json:"backupDB"`
	CloseAfterDone    bool              `json:"closeAfterDone"`
	AccountSourcePath string            `json:"accountSourcePath"`
	AccountSource     string            `json:"accountSource"`
	CloneHVUsername   string            `json:"cloneHvUsername"`
	CloneHVPassword   string            `json:"cloneHvPassword"`
	CloneHVProductID  string            `json:"cloneHvProductId"`
	CloneHVAmount     int               `json:"cloneHvAmount"`
	CaptchaProvider   string            `json:"captchaProvider"`
	CaptchaKeys       map[string]string `json:"captchaKeys"`
	IpProvider        string            `json:"ipProvider"`
	CheckIpBeforeRun  bool              `json:"checkIpBeforeRun"`
	DelayChangeIp     int               `json:"delayChangeIp"`
	// Locale / Cookie / UA / SimNetwork — các field mới thêm vào GeneralConfig
	LocaleFake       string `json:"localeFake,omitempty"`
	DeepFakeInApi    bool   `json:"deepFakeInApi,omitempty"`
	CookieUse        bool   `json:"cookieUse,omitempty"`
	CookieLimit      bool   `json:"cookieLimit,omitempty"`
	CookieLimitCount int    `json:"cookieLimitCount,omitempty"`
	CookieMode       string `json:"cookieMode,omitempty"`
	UaAddSpecs       bool   `json:"uaAddSpecs,omitempty"`
	UaBuildFile      bool   `json:"uaBuildFile,omitempty"`
	UaCustomType     int    `json:"uaCustomType,omitempty"`
	SimNetworkMode   string `json:"simNetworkMode,omitempty"`
	SimNetworkType   string `json:"simNetworkType,omitempty"`
}

// LegacyIpConfig mirrors app.go IpConfig
type LegacyIpConfig struct {
	ProxyList               string `json:"proxyList"`
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
}

// LegacySettingsData mirrors app.go SettingsData
type LegacySettingsData struct {
	General LegacyGeneralConfig `json:"general"`
	Ip      LegacyIpConfig      `json:"ip"`
}

// LegacyInteractionConfig mirrors app.go InteractionConfig
type LegacyInteractionConfig struct {
	VerifyEnabled       bool   `json:"verifyEnabled"`
	MailProvider        string `json:"mailProvider"`
	MailList            string `json:"mailList"`
	CheckLiveDieEnabled bool   `json:"checkLiveDieEnabled"`
	TimeDelayCheck      int    `json:"timeDelayCheck"`
	TimeDelaySendCode   int    `json:"timeDelaySendCode"`
	SendAgainCode       bool   `json:"sendAgainCode"`
	OutputPath          string `json:"outputPath"`
	UaIphoneList        string            `json:"uaIphoneList"`
	UaPools             map[string]string `json:"uaPools,omitempty"`
	UaPoolKey           string            `json:"uaPoolKey,omitempty"`
	UaPoolFiles         map[string]string `json:"uaPoolFiles,omitempty"`
	ZeusXApiKey         string            `json:"zeusXApiKey"`
	ZeusXAccountCode    string `json:"zeusXAccountCode"`
	DvfbApiKey          string `json:"dvfbApiKey"`
	DvfbAccountType     string `json:"dvfbAccountType"`
	Store1sApiKey       string `json:"store1sApiKey"`
	Store1sProductID    string `json:"store1sProductId"`
	Mail30sApiKey       string `json:"mail30sApiKey"`
	Mail30sProductSlug  string `json:"mail30sProductSlug"`
	CreateEnabled       bool   `json:"createEnabled"`
	CreateType          string `json:"createType"`
	CreateCookieList    string `json:"createCookieList"`
	CreateOutputPath    string `json:"createOutputPath"`
	ResultFolderPath    string `json:"resultFolderPath,omitempty"`
}

// ─── FromLegacy: old config → AppSettings ─────────────────────────────────

// FromLegacy chuyển đổi cặp config cũ (SettingsData + InteractionConfig)
// sang AppSettings v1 — đây là điểm vào chính cho migration.
//
// s: LegacySettingsData bao gồm LegacyGeneralConfig (thread, delay, login
// platform, account source, CloneHV credentials, captcha, IP provider...)
// và LegacyIpConfig (proxy list, keys của từng provider thương mại).
//
// ic: LegacyInteractionConfig chứa cấu hình verify (enabled, mail provider,
// mail list, delay times), UA pools, output path, và register/create config.
//
// Migration entry point: hàm bắt đầu bằng DefaultAppSettings() để lấy
// AppSettings với giá trị mặc định an toàn, sau đó ghi đè Global settings
// và build một Profile "default" từ toàn bộ s + ic. Profile được đưa vào
// AppSettings qua UpsertProfile() — nếu đã tồn tại profile "default" thì
// replace, nếu chưa thì thêm mới.
//
// Sau khi migrate, caller nên lưu AppSettings mới ra file v1 để các lần
// khởi động sau không cần migrate lại.
func FromLegacy(s LegacySettingsData, ic LegacyInteractionConfig) model.AppSettings {
	app := model.DefaultAppSettings()

	// Global settings từ GeneralConfig
	app.Global = model.GlobalSettings{
		LoginPlatform:  s.General.LoginPlatform,
		LoginMethod:    s.General.LoginMethod,
		SaveRunColumn:  s.General.SaveRunColumn,
		BackupDB:       s.General.BackupDB,
		CloseAfterDone: s.General.CloseAfterDone,
	}

	// Build profile "default" từ toàn bộ config
	p := model.DefaultProfile("default", "Default")

	// Runtime
	p.Runtime = model.RuntimeSettings{
		ThreadRequest:    s.General.ThreadRequest,
		DelayRequest:     s.General.DelayRequest,
		DelayThread:      s.General.DelayThread,
		ApiCheckIp:       s.General.ApiCheckIp,
		ThreadCheckInfo:  s.General.ThreadCheckInfo,
		DelayChangeIp:    s.General.DelayChangeIp,
		CheckIpBeforeRun: s.General.CheckIpBeforeRun,
	}

	// Account source
	p.Account = model.AccountSettings{
		Source:     s.General.AccountSource,
		FolderPath: s.General.AccountSourcePath,
		CloneHV: model.CloneHVCredentials{
			Enabled:   s.General.AccountSource == "api",
			Username:  s.General.CloneHVUsername,
			Password:  s.General.CloneHVPassword,
			ProductID: s.General.CloneHVProductID,
			Amount:    s.General.CloneHVAmount,
		},
	}

	// Proxy
	p.Proxy = model.ProxySettings{
		Provider:  s.General.IpProvider,
		ProxyList: s.Ip.ProxyList,
		ProxyType: s.Ip.ProxyType,
		Providers: map[string]model.ProxyProviderCfg{
			"fpt":          {Keys: s.Ip.FptKeys},
			"xproxy":       {ServiceURL: s.Ip.XproxyServiceUrl, Type: s.Ip.XproxyType, List: s.Ip.XproxyList, ThreadPerIP: s.Ip.XproxyThreadPerIp, RunType: s.Ip.XproxyRunType},
			"tinsoft":      {Keys: s.Ip.TinsoftKeys, ThreadPerIP: s.Ip.TinsoftThreadPerIp},
			"shoplike":     {Keys: s.Ip.ShoplikeKeys, ThreadPerIP: s.Ip.ShoplikeThreadPerIp},
			"netproxy":     {Keys: s.Ip.NetproxyKeys, ThreadPerIP: s.Ip.NetproxyThreadPerIp},
			"minproxy":     {Keys: s.Ip.MinproxyKeys, ThreadPerIP: s.Ip.MinproxyThreadPerIp},
			"netproxygb":   {Keys: s.Ip.NetproxyGbKey},
			"proxypopular": {Keys: s.Ip.ProxyPopularKeys, ThreadPerIP: s.Ip.ProxyPopularThreadPerIp, AccessToken: s.Ip.ProxyPopularAccessToken},
			"proxyfarm":    {Keys: s.Ip.ProxyFarmKeys, ThreadPerIP: s.Ip.ProxyFarmThreadPerIp, AccessToken: s.Ip.ProxyFarmAccessToken},
		},
	}

	// Verify
	p.Verify = model.VerifySettings{
		Enabled:           ic.VerifyEnabled,
		CheckLiveDie:      ic.CheckLiveDieEnabled,
		TimeDelayCheck:    ic.TimeDelayCheck,
		TimeDelaySendCode: ic.TimeDelaySendCode,
		SendAgainCode:     ic.SendAgainCode,
	}

	// Register/Create
	p.Register = model.RegisterSettings{
		Enabled:    ic.CreateEnabled,
		Type:       ic.CreateType,
		CookieList: ic.CreateCookieList,
		OutputPath: ic.CreateOutputPath,
	}

	// Mail
	p.Mail = model.MailSettings{
		Provider: ic.MailProvider,
		MailList: ic.MailList,
		Providers: map[string]model.MailProviderCfg{
			"zeusx":   {APIKey: ic.ZeusXApiKey, AccountCode: ic.ZeusXAccountCode},
			"dvfb":    {APIKey: ic.DvfbApiKey, AccountType: ic.DvfbAccountType},
			"store1s": {APIKey: ic.Store1sApiKey, ProductID: ic.Store1sProductID},
			"mail30s": {APIKey: ic.Mail30sApiKey, ProductSlug: ic.Mail30sProductSlug},
		},
	}

	// Captcha
	keys := s.General.CaptchaKeys
	if keys == nil {
		keys = map[string]string{"2captcha": "", "omocaptcha": "", "ezcaptcha": "", "capsolver": ""}
	}
	p.Captcha = model.CaptchaSettings{
		Provider: s.General.CaptchaProvider,
		Keys:     keys,
	}

	// Output
	p.Output = model.OutputSettings{
		VerifyPath:   ic.OutputPath,
		RegisterPath: ic.CreateOutputPath,
	}

	// Device — migration: nếu chưa có UaPools, khởi tạo từ UaIphoneList cũ
	pools := ic.UaPools
	if pools == nil {
		pools = map[string]string{"iphone": ic.UaIphoneList, "android": "", "chrome": ""}
	} else if _, ok := pools["iphone"]; !ok {
		pools["iphone"] = ic.UaIphoneList
	}
	poolKey := ic.UaPoolKey
	if poolKey == "" {
		poolKey = "iphone"
	}
	poolFiles := ic.UaPoolFiles
	if poolFiles == nil {
		poolFiles = map[string]string{}
	}
	p.Device = model.DeviceSettings{
		UAList:      ic.UaIphoneList,
		UaPools:     pools,
		UaPoolKey:   poolKey,
		UaPoolFiles: poolFiles,
	}

	app.UpsertProfile(p)
	return app
}

// ─── Partial update helpers ───────────────────────────────────────────────

// ApplySettingsToProfile áp dụng LegacySettingsData lên một Profile đã tồn tại
// theo kiểu in-place (sửa trực tiếp qua con trỏ).
//
// p: Profile cần cập nhật — thường là profile đang active lấy từ
// AppSettings.GetActiveProfile(). Các field được ghi đè: Runtime,
// Account (Source, FolderPath, CloneHV), Proxy (toàn bộ providers),
// Captcha.
//
// global: con trỏ đến GlobalSettings của AppSettings — được ghi đè hoàn
// toàn bởi các field LoginPlatform, LoginMethod, SaveRunColumn, BackupDB,
// CloseAfterDone từ s.General.
//
// s: LegacySettingsData nguồn dữ liệu — thường là dữ liệu người dùng vừa
// lưu từ UI (frontend gửi lên qua Wails binding).
//
// Does NOT touch: Verify, Mail, Register, Output, Device — các nhóm này
// thuộc về InteractionConfig và được quản lý bởi ApplyInteractionToProfile.
// Tách biệt hai hàm đảm bảo mỗi hàm chỉ có một trách nhiệm và tránh ghi
// đè nhầm config của nhau khi người dùng save từng tab riêng lẻ trên UI.
func ApplySettingsToProfile(p *model.Profile, global *model.GlobalSettings, s LegacySettingsData) {
	*global = model.GlobalSettings{
		LoginPlatform:  s.General.LoginPlatform,
		LoginMethod:    s.General.LoginMethod,
		SaveRunColumn:  s.General.SaveRunColumn,
		BackupDB:       s.General.BackupDB,
		CloseAfterDone: s.General.CloseAfterDone,
	}
	p.Runtime = model.RuntimeSettings{
		ThreadRequest:    s.General.ThreadRequest,
		DelayRequest:     s.General.DelayRequest,
		DelayThread:      s.General.DelayThread,
		ApiCheckIp:       s.General.ApiCheckIp,
		ThreadCheckInfo:  s.General.ThreadCheckInfo,
		DelayChangeIp:    s.General.DelayChangeIp,
		CheckIpBeforeRun: s.General.CheckIpBeforeRun,
	}
	p.Account.Source = s.General.AccountSource
	p.Account.FolderPath = s.General.AccountSourcePath
	p.Account.CloneHV = model.CloneHVCredentials{
		Enabled:   s.General.AccountSource == "api",
		Username:  s.General.CloneHVUsername,
		Password:  s.General.CloneHVPassword,
		ProductID: s.General.CloneHVProductID,
		Amount:    s.General.CloneHVAmount,
	}

	keys := s.General.CaptchaKeys
	if keys == nil {
		keys = map[string]string{"2captcha": "", "omocaptcha": "", "ezcaptcha": "", "capsolver": ""}
	}
	p.Captcha = model.CaptchaSettings{Provider: s.General.CaptchaProvider, Keys: keys}

	if p.Proxy.Providers == nil {
		p.Proxy.Providers = map[string]model.ProxyProviderCfg{}
	}
	prov := p.Proxy.Providers
	prov["fpt"] = model.ProxyProviderCfg{Keys: s.Ip.FptKeys}
	prov["xproxy"] = model.ProxyProviderCfg{
		ServiceURL: s.Ip.XproxyServiceUrl, Type: s.Ip.XproxyType,
		List: s.Ip.XproxyList, ThreadPerIP: s.Ip.XproxyThreadPerIp, RunType: s.Ip.XproxyRunType,
	}
	prov["tinsoft"] = model.ProxyProviderCfg{Keys: s.Ip.TinsoftKeys, ThreadPerIP: s.Ip.TinsoftThreadPerIp}
	prov["shoplike"] = model.ProxyProviderCfg{Keys: s.Ip.ShoplikeKeys, ThreadPerIP: s.Ip.ShoplikeThreadPerIp}
	prov["netproxy"] = model.ProxyProviderCfg{Keys: s.Ip.NetproxyKeys, ThreadPerIP: s.Ip.NetproxyThreadPerIp}
	prov["minproxy"] = model.ProxyProviderCfg{Keys: s.Ip.MinproxyKeys, ThreadPerIP: s.Ip.MinproxyThreadPerIp}
	prov["netproxygb"] = model.ProxyProviderCfg{Keys: s.Ip.NetproxyGbKey}
	prov["proxypopular"] = model.ProxyProviderCfg{
		Keys: s.Ip.ProxyPopularKeys, ThreadPerIP: s.Ip.ProxyPopularThreadPerIp,
		AccessToken: s.Ip.ProxyPopularAccessToken,
	}
	prov["proxyfarm"] = model.ProxyProviderCfg{
		Keys: s.Ip.ProxyFarmKeys, ThreadPerIP: s.Ip.ProxyFarmThreadPerIp,
		AccessToken: s.Ip.ProxyFarmAccessToken,
	}
	p.Proxy = model.ProxySettings{
		Provider: s.General.IpProvider, ProxyList: s.Ip.ProxyList,
		ProxyType: s.Ip.ProxyType, Providers: prov,
	}
}

// ApplyInteractionToProfile áp dụng LegacyInteractionConfig lên một Profile
// đã tồn tại theo kiểu in-place.
//
// p: Profile cần cập nhật. Các field được ghi đè: Verify (enabled, timing,
// live/die check), Mail (provider, mail list, tất cả provider configs),
// Register (create enabled/type/cookie/output), Output (verify path,
// register path), Device (UAList + UA pools).
//
// ic: LegacyInteractionConfig nguồn dữ liệu — thường từ frontend tab
// Interaction/Verify gửi lên.
//
// Does NOT touch: Runtime, Proxy, Captcha, Account — các nhóm này thuộc về
// SettingsData và được quản lý bởi ApplySettingsToProfile. Tách biệt này
// cho phép frontend save Settings tab và Interaction tab độc lập mà không
// cần gửi toàn bộ config mỗi lần.
//
// Device preservation: UaPools, UaPoolKey, UaPoolFiles được giữ nguyên từ
// profile hiện tại nếu ic không mang giá trị mới (nil hoặc rỗng). Điều này
// đảm bảo user-defined UA pools không bị xóa mất khi save các config khác
// không liên quan đến Device. Chỉ UAList (legacy iphone list) luôn được
// ghi đè vì ic.UaIphoneList luôn có giá trị xác định.
func ApplyInteractionToProfile(p *model.Profile, ic LegacyInteractionConfig) {
	p.Verify = model.VerifySettings{
		Enabled: ic.VerifyEnabled, CheckLiveDie: ic.CheckLiveDieEnabled,
		TimeDelayCheck: ic.TimeDelayCheck, TimeDelaySendCode: ic.TimeDelaySendCode,
		SendAgainCode: ic.SendAgainCode,
	}
	if p.Mail.Providers == nil {
		p.Mail.Providers = map[string]model.MailProviderCfg{}
	}
	p.Mail.Provider = ic.MailProvider
	p.Mail.MailList = ic.MailList
	p.Mail.Providers["zeusx"] = model.MailProviderCfg{APIKey: ic.ZeusXApiKey, AccountCode: ic.ZeusXAccountCode}
	p.Mail.Providers["dvfb"] = model.MailProviderCfg{APIKey: ic.DvfbApiKey, AccountType: ic.DvfbAccountType}
	p.Mail.Providers["store1s"] = model.MailProviderCfg{APIKey: ic.Store1sApiKey, ProductID: ic.Store1sProductID}
	p.Mail.Providers["mail30s"] = model.MailProviderCfg{APIKey: ic.Mail30sApiKey, ProductSlug: ic.Mail30sProductSlug}
	p.Register = model.RegisterSettings{
		Enabled: ic.CreateEnabled, Type: ic.CreateType,
		CookieList: ic.CreateCookieList, OutputPath: ic.CreateOutputPath,
	}
	p.Output = model.OutputSettings{VerifyPath: ic.OutputPath, RegisterPath: ic.CreateOutputPath}

	// Device: preserve UaPools/UaPoolKey/UaPoolFiles — chỉ ghi đè UAList
	pools := ic.UaPools
	if pools == nil {
		pools = p.Device.UaPools
	}
	poolFiles := ic.UaPoolFiles
	if poolFiles == nil {
		poolFiles = p.Device.UaPoolFiles
	}
	poolKey := ic.UaPoolKey
	if poolKey == "" {
		poolKey = p.Device.UaPoolKey
	}
	// Đảm bảo luôn có giá trị mặc định — tránh nil khi config cũ không có
	if pools == nil {
		pools = map[string]string{"iphone": "", "android": "", "chrome": ""}
	}
	if poolFiles == nil {
		poolFiles = map[string]string{}
	}
	if poolKey == "" {
		poolKey = "iphone"
	}
	p.Device = model.DeviceSettings{
		UAList:      ic.UaIphoneList,
		UaPools:     pools,
		UaPoolKey:   poolKey,
		UaPoolFiles: poolFiles,
	}
}

// ─── ToLegacy: AppSettings → old config ───────────────────────────────────

// ToLegacySettings chuyển AppSettings v1 về LegacySettingsData để
// backward-compat với app.go (vẫn dùng SettingsData + IpConfig).
//
// a: AppSettings nguồn — thường là settings đang active trong memory sau
// khi đã apply các thay đổi từ UI.
//
// Roundtrip conversion: hàm lấy profile active qua GetActiveProfile(). Nếu
// không có profile nào (slice rỗng hoặc activeID không tìm thấy), dùng
// Profile rỗng với zero values để tránh panic — caller vẫn nhận được
// LegacySettingsData hợp lệ (với giá trị mặc định).
//
// GetActiveProfile() logic: tìm profile có ID khớp với a.ActiveProfileID.
// Nếu không tìm thấy (ví dụ profile bị xóa mà activeID chưa reset), trả
// về nil và hàm này fallback về Profile{}.
//
// Kết quả được dùng để: serialize ra file JSON legacy, hoặc pass vào
// app.go runtime đang chờ SettingsData struct.
func ToLegacySettings(a model.AppSettings) LegacySettingsData {
	p := a.GetActiveProfile()
	if p == nil {
		p = &model.Profile{}
	}

	prov := p.Proxy.Providers
	if prov == nil {
		prov = map[string]model.ProxyProviderCfg{}
	}
	get := func(key string) model.ProxyProviderCfg { return prov[key] }

	captchaKeys := p.Captcha.Keys
	if captchaKeys == nil {
		captchaKeys = map[string]string{}
	}

	return LegacySettingsData{
		General: LegacyGeneralConfig{
			ThreadRequest:     p.Runtime.ThreadRequest,
			DelayRequest:      p.Runtime.DelayRequest,
			DelayThread:       p.Runtime.DelayThread,
			ApiCheckIp:        p.Runtime.ApiCheckIp,
			ThreadCheckInfo:   p.Runtime.ThreadCheckInfo,
			LoginPlatform:     a.Global.LoginPlatform,
			LoginMethod:       a.Global.LoginMethod,
			SaveRunColumn:     a.Global.SaveRunColumn,
			BackupDB:          a.Global.BackupDB,
			CloseAfterDone:    a.Global.CloseAfterDone,
			AccountSourcePath: p.Account.FolderPath,
			AccountSource:     p.Account.Source,
			CloneHVUsername:   p.Account.CloneHV.Username,
			CloneHVPassword:   p.Account.CloneHV.Password,
			CloneHVProductID:  p.Account.CloneHV.ProductID,
			CloneHVAmount:     p.Account.CloneHV.Amount,
			CaptchaProvider:   p.Captcha.Provider,
			CaptchaKeys:       captchaKeys,
			IpProvider:        p.Proxy.Provider,
			CheckIpBeforeRun:  p.Runtime.CheckIpBeforeRun,
			DelayChangeIp:     p.Runtime.DelayChangeIp,
			// Locale / Cookie / UA / SimNetwork — preserved as-is (no model mapping yet)
		},
		Ip: LegacyIpConfig{
			ProxyList:               p.Proxy.ProxyList,
			ProxyType:               p.Proxy.ProxyType,
			FptKeys:                 get("fpt").Keys,
			XproxyServiceUrl:        get("xproxy").ServiceURL,
			XproxyType:              get("xproxy").Type,
			XproxyList:              get("xproxy").List,
			XproxyThreadPerIp:       get("xproxy").ThreadPerIP,
			XproxyRunType:           get("xproxy").RunType,
			TinsoftKeys:             get("tinsoft").Keys,
			TinsoftThreadPerIp:      get("tinsoft").ThreadPerIP,
			ShoplikeKeys:            get("shoplike").Keys,
			ShoplikeThreadPerIp:     get("shoplike").ThreadPerIP,
			NetproxyKeys:            get("netproxy").Keys,
			NetproxyThreadPerIp:     get("netproxy").ThreadPerIP,
			MinproxyKeys:            get("minproxy").Keys,
			MinproxyThreadPerIp:     get("minproxy").ThreadPerIP,
			NetproxyGbKey:           get("netproxygb").Keys,
			ProxyPopularKeys:        get("proxypopular").Keys,
			ProxyPopularThreadPerIp: get("proxypopular").ThreadPerIP,
			ProxyPopularAccessToken: get("proxypopular").AccessToken,
			ProxyFarmKeys:           get("proxyfarm").Keys,
			ProxyFarmThreadPerIp:    get("proxyfarm").ThreadPerIP,
			ProxyFarmAccessToken:    get("proxyfarm").AccessToken,
		},
	}
}

// ToLegacyInteraction chuyển AppSettings v1 về LegacyInteractionConfig để
// backward-compat với app.go (runner vẫn đọc InteractionConfig trực tiếp).
//
// a: AppSettings nguồn. Cũng dùng GetActiveProfile() giống ToLegacySettings
// — fallback về Profile{} nếu không tìm thấy profile active.
//
// Conversion cho runner: kết quả LegacyInteractionConfig được truyền vào
// các bước verify/create trong runner (steps.go) thông qua app.go, do đó
// phải ánh xạ đầy đủ: Verify settings, Mail provider + tất cả API keys,
// Register/Create config, Output paths, và Device UA pools.
//
// UaPools, UaPoolKey, UaPoolFiles được copy nguyên từ Profile.Device để
// đảm bảo runner luôn có đủ UA data dù config đến từ migration hay UI save.
func ToLegacyInteraction(a model.AppSettings) LegacyInteractionConfig {
	p := a.GetActiveProfile()
	if p == nil {
		p = &model.Profile{}
	}

	mail := p.Mail.Providers
	if mail == nil {
		mail = map[string]model.MailProviderCfg{}
	}
	getM := func(key string) model.MailProviderCfg { return mail[key] }

	return LegacyInteractionConfig{
		VerifyEnabled:       p.Verify.Enabled,
		MailProvider:        p.Mail.Provider,
		MailList:            p.Mail.MailList,
		CheckLiveDieEnabled: p.Verify.CheckLiveDie,
		TimeDelayCheck:      p.Verify.TimeDelayCheck,
		TimeDelaySendCode:   p.Verify.TimeDelaySendCode,
		SendAgainCode:       p.Verify.SendAgainCode,
		OutputPath:          p.Output.VerifyPath,
		UaIphoneList:        p.Device.UAList,
		UaPools:             p.Device.UaPools,
		UaPoolKey:           p.Device.UaPoolKey,
		UaPoolFiles:         p.Device.UaPoolFiles,
		ZeusXApiKey:         getM("zeusx").APIKey,
		ZeusXAccountCode:    getM("zeusx").AccountCode,
		DvfbApiKey:          getM("dvfb").APIKey,
		DvfbAccountType:     getM("dvfb").AccountType,
		Store1sApiKey:       getM("store1s").APIKey,
		Store1sProductID:    getM("store1s").ProductID,
		Mail30sApiKey:       getM("mail30s").APIKey,
		Mail30sProductSlug:  getM("mail30s").ProductSlug,
		CreateEnabled:       p.Register.Enabled,
		CreateType:          p.Register.Type,
		CreateCookieList:    p.Register.CookieList,
		CreateOutputPath:    p.Register.OutputPath,
	}
}
