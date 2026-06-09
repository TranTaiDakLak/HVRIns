package adapter_test

import (
	"testing"

	"HVRIns/internal/settings/adapter"
	"HVRIns/internal/settings/model"
)

// sampleLegacySettings tạo dữ liệu legacy mẫu để test
func sampleLegacySettings() adapter.LegacySettingsData {
	return adapter.LegacySettingsData{
		General: adapter.LegacyGeneralConfig{
			ThreadRequest:     10,
			DelayRequest:      500,
			ThreadCheckInfo:   5,
			LoginPlatform:     "facebook",
			LoginMethod:       2,
			SaveRunColumn:     true,
			BackupDB:          false,
			CloseAfterDone:    false,
			AccountSourcePath: "/data/accounts",
			AccountSource:     "api",
			CloneHVUsername:   "user123",
			CloneHVPassword:   "pass456",
			CloneHVProductID:  "18",
			CloneHVAmount:     1,
			CaptchaProvider:   "2captcha",
			CaptchaKeys:       map[string]string{"2captcha": "KEY123", "capsolver": "", "ezcaptcha": "", "omocaptcha": ""},
			IpProvider:        "proxy",
			CheckIpBeforeRun:  true,
			DelayChangeIp:     2,
		},
		Ip: adapter.LegacyIpConfig{
			ProxyList:               "proxy1:8080:user:pass\nproxy2:8080:user:pass",
			ProxyType:               "http",
			FptKeys:                 "fpt_key_1",
			XproxyServiceUrl:        "http://xproxy.example.com",
			XproxyType:              "socks5",
			XproxyList:              "xp1\nxp2",
			XproxyThreadPerIp:       3,
			XproxyRunType:           "shared",
			TinsoftKeys:             "tinsoft_key",
			TinsoftThreadPerIp:      2,
			ProxyPopularKeys:        "pop_key",
			ProxyPopularThreadPerIp: 5,
			ProxyPopularAccessToken: "pop_token",
			ProxyFarmKeys:           "farm_key",
			ProxyFarmThreadPerIp:    4,
			ProxyFarmAccessToken:    "farm_token",
		},
	}
}

// sampleLegacyInteraction tạo dữ liệu interaction legacy mẫu để test
func sampleLegacyInteraction() adapter.LegacyInteractionConfig {
	return adapter.LegacyInteractionConfig{
		VerifyEnabled:       true,
		MailProvider:        "@gmail.com",
		MailList:            "mail1@gmail.com\nmail2@gmail.com",
		CheckLiveDieEnabled: true,
		TimeDelayCheck:      10,
		TimeDelaySendCode:   8,
		SendAgainCode:       true,
		OutputPath:          "/output/verify",
		UaIphoneList:        "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2)\nMozilla/5.0 (iPhone; CPU iPhone OS 16_0)",
		ZeusXApiKey:         "zeus_key",
		ZeusXAccountCode:    "zeus_code",
		DvfbApiKey:          "dvfb_key",
		DvfbAccountType:     "1",
		Store1sApiKey:       "store_key",
		Store1sProductID:    "40559",
		Mail30sApiKey:       "mail30s_key",
		Mail30sProductSlug:  "hotmail-oauth2",
		CreateEnabled:       true,
		CreateType:          "normal",
		CreateCookieList:    "cookie1\ncookie2",
		CreateOutputPath:    "/output/created",
	}
}

// TestFromLegacy_BasicConversion kiểm tra chuyển đổi cơ bản từ legacy → AppSettings
func TestFromLegacy_BasicConversion(t *testing.T) {
	s := sampleLegacySettings()
	ic := sampleLegacyInteraction()

	app := adapter.FromLegacy(s, ic)

	if app.Version != model.CurrentVersion {
		t.Errorf("version: got %d, want %d", app.Version, model.CurrentVersion)
	}
	if app.ActiveProfileID != "default" {
		t.Errorf("activeProfileId: got %s, want default", app.ActiveProfileID)
	}
}

// TestFromLegacy_GlobalSettings kiểm tra global settings được map đúng
func TestFromLegacy_GlobalSettings(t *testing.T) {
	s := sampleLegacySettings()
	ic := sampleLegacyInteraction()
	app := adapter.FromLegacy(s, ic)

	g := app.Global
	if g.LoginPlatform != "facebook" {
		t.Errorf("loginPlatform: got %s, want facebook", g.LoginPlatform)
	}
	if g.LoginMethod != 2 {
		t.Errorf("loginMethod: got %d, want 2", g.LoginMethod)
	}
	if !g.SaveRunColumn {
		t.Error("saveRunColumn: got false, want true")
	}
}

// TestFromLegacy_RuntimeSettings kiểm tra runtime settings được map đúng
func TestFromLegacy_RuntimeSettings(t *testing.T) {
	s := sampleLegacySettings()
	ic := sampleLegacyInteraction()
	app := adapter.FromLegacy(s, ic)

	p := app.GetActiveProfile()
	if p == nil {
		t.Fatal("active profile is nil")
	}
	r := p.Runtime
	if r.ThreadRequest != 10 {
		t.Errorf("threadRequest: got %d, want 10", r.ThreadRequest)
	}
	if r.DelayRequest != 500 {
		t.Errorf("delayRequest: got %d, want 500", r.DelayRequest)
	}
	if r.ThreadCheckInfo != 5 {
		t.Errorf("threadCheckInfo: got %d, want 5", r.ThreadCheckInfo)
	}
	if !r.CheckIpBeforeRun {
		t.Error("checkIpBeforeRun: got false, want true")
	}
}

// TestFromLegacy_AccountSource kiểm tra account source và CloneHV credentials
func TestFromLegacy_AccountSource(t *testing.T) {
	s := sampleLegacySettings()
	ic := sampleLegacyInteraction()
	app := adapter.FromLegacy(s, ic)

	p := app.GetActiveProfile()
	acc := p.Account

	if acc.Source != "api" {
		t.Errorf("source: got %s, want api", acc.Source)
	}
	if acc.FolderPath != "/data/accounts" {
		t.Errorf("folderPath: got %s, want /data/accounts", acc.FolderPath)
	}
	if !acc.CloneHV.Enabled {
		t.Error("cloneHv.enabled: got false, want true")
	}
	if acc.CloneHV.Username != "user123" {
		t.Errorf("cloneHv.username: got %s, want user123", acc.CloneHV.Username)
	}
	if acc.CloneHV.ProductID != "18" {
		t.Errorf("cloneHv.productId: got %s, want 18", acc.CloneHV.ProductID)
	}
}

// TestFromLegacy_ProxySettings kiểm tra proxy settings và providers
func TestFromLegacy_ProxySettings(t *testing.T) {
	s := sampleLegacySettings()
	ic := sampleLegacyInteraction()
	app := adapter.FromLegacy(s, ic)

	p := app.GetActiveProfile()
	proxy := p.Proxy

	if proxy.Provider != "proxy" {
		t.Errorf("proxy.provider: got %s, want proxy", proxy.Provider)
	}
	if proxy.ProxyType != "http" {
		t.Errorf("proxy.proxyType: got %s, want http", proxy.ProxyType)
	}
	fpt, ok := proxy.Providers["fpt"]
	if !ok {
		t.Error("providers.fpt: not found")
	} else if fpt.Keys != "fpt_key_1" {
		t.Errorf("providers.fpt.keys: got %s, want fpt_key_1", fpt.Keys)
	}

	xp := proxy.Providers["xproxy"]
	if xp.ServiceURL != "http://xproxy.example.com" {
		t.Errorf("providers.xproxy.serviceUrl: got %s", xp.ServiceURL)
	}
	if xp.ThreadPerIP != 3 {
		t.Errorf("providers.xproxy.threadPerIp: got %d, want 3", xp.ThreadPerIP)
	}
}

// TestFromLegacy_MailSettings kiểm tra mail provider config
func TestFromLegacy_MailSettings(t *testing.T) {
	s := sampleLegacySettings()
	ic := sampleLegacyInteraction()
	app := adapter.FromLegacy(s, ic)

	p := app.GetActiveProfile()
	mail := p.Mail

	if mail.Provider != "@gmail.com" {
		t.Errorf("mail.provider: got %s, want @gmail.com", mail.Provider)
	}

	store1s, ok := mail.Providers["store1s"]
	if !ok {
		t.Error("mail.providers.store1s: not found")
	}
	if store1s.APIKey != "store_key" {
		t.Errorf("store1s.apiKey: got %s, want store_key", store1s.APIKey)
	}
	if store1s.ProductID != "40559" {
		t.Errorf("store1s.productId: got %s, want 40559", store1s.ProductID)
	}
}

// TestRoundtrip_SettingsData kiểm tra FromLegacy → ToLegacySettings → so sánh
func TestRoundtrip_SettingsData(t *testing.T) {
	orig := sampleLegacySettings()
	ic := sampleLegacyInteraction()

	app := adapter.FromLegacy(orig, ic)
	roundtrip := adapter.ToLegacySettings(app)

	g := roundtrip.General
	if g.ThreadRequest != orig.General.ThreadRequest {
		t.Errorf("roundtrip threadRequest: got %d, want %d", g.ThreadRequest, orig.General.ThreadRequest)
	}
	if g.AccountSource != orig.General.AccountSource {
		t.Errorf("roundtrip accountSource: got %s, want %s", g.AccountSource, orig.General.AccountSource)
	}
	if g.IpProvider != orig.General.IpProvider {
		t.Errorf("roundtrip ipProvider: got %s, want %s", g.IpProvider, orig.General.IpProvider)
	}
	if g.CaptchaProvider != orig.General.CaptchaProvider {
		t.Errorf("roundtrip captchaProvider: got %s, want %s", g.CaptchaProvider, orig.General.CaptchaProvider)
	}
	if roundtrip.Ip.ProxyList != orig.Ip.ProxyList {
		t.Errorf("roundtrip proxyList: got %s, want %s", roundtrip.Ip.ProxyList, orig.Ip.ProxyList)
	}
	if roundtrip.Ip.FptKeys != orig.Ip.FptKeys {
		t.Errorf("roundtrip fptKeys: got %s, want %s", roundtrip.Ip.FptKeys, orig.Ip.FptKeys)
	}
}

// TestRoundtrip_InteractionConfig kiểm tra FromLegacy → ToLegacyInteraction → so sánh
func TestRoundtrip_InteractionConfig(t *testing.T) {
	s := sampleLegacySettings()
	origIC := sampleLegacyInteraction()

	app := adapter.FromLegacy(s, origIC)
	roundtrip := adapter.ToLegacyInteraction(app)

	if roundtrip.VerifyEnabled != origIC.VerifyEnabled {
		t.Errorf("roundtrip verifyEnabled: got %v, want %v", roundtrip.VerifyEnabled, origIC.VerifyEnabled)
	}
	if roundtrip.MailProvider != origIC.MailProvider {
		t.Errorf("roundtrip mailProvider: got %s, want %s", roundtrip.MailProvider, origIC.MailProvider)
	}
	if roundtrip.Store1sProductID != origIC.Store1sProductID {
		t.Errorf("roundtrip store1sProductId: got %s, want %s", roundtrip.Store1sProductID, origIC.Store1sProductID)
	}
	if roundtrip.Mail30sProductSlug != origIC.Mail30sProductSlug {
		t.Errorf("roundtrip mail30sProductSlug: got %s, want %s", roundtrip.Mail30sProductSlug, origIC.Mail30sProductSlug)
	}
	if roundtrip.UaIphoneList != origIC.UaIphoneList {
		t.Errorf("roundtrip uaIphoneList: got %s, want %s", roundtrip.UaIphoneList, origIC.UaIphoneList)
	}
}

// TestFromLegacy_EmptyInput kiểm tra không panic khi input rỗng
func TestFromLegacy_EmptyInput(t *testing.T) {
	app := adapter.FromLegacy(adapter.LegacySettingsData{}, adapter.LegacyInteractionConfig{})
	if app.Version != model.CurrentVersion {
		t.Errorf("empty input: version got %d, want %d", app.Version, model.CurrentVersion)
	}
	p := app.GetActiveProfile()
	if p == nil {
		t.Error("empty input: GetActiveProfile() returned nil")
	}
}
