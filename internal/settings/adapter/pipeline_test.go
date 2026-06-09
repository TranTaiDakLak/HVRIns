// pipeline_test.go — End-to-end pipeline: JSON strings → FromLegacy → Validate → SaveTo → LoadFrom
// Chứng minh toàn bộ chuỗi không mất dữ liệu quan trọng
package adapter_test

import (
	"encoding/json"
	"testing"

	"HVRIns/internal/settings/adapter"
	"HVRIns/internal/settings/store"
	"HVRIns/internal/settings/validation"
)

// fullLegacyGeneralJSON là JSON giả lập general.json của tool cũ với đầy đủ field
const fullLegacyGeneralJSON = `{
  "general": {
    "threadRequest": 12,
    "delayRequest": 800,
    "threadCheckInfo": 6,
    "loginPlatform": "facebook",
    "loginMethod": 1,
    "saveRunColumn": true,
    "backupDB": false,
    "closeAfterDone": true,
    "accountSource": "api",
    "accountSourcePath": "C:\\accounts\\fb",
    "cloneHvUsername": "pipeuser",
    "cloneHvPassword": "pipepass",
    "cloneHvProductId": "18",
    "cloneHvAmount": 2,
    "captchaProvider": "capsolver",
    "captchaKeys": { "capsolver": "CAP123", "2captcha": "", "ezcaptcha": "", "omocaptcha": "" },
    "ipProvider": "tinsoft",
    "checkIpBeforeRun": true,
    "delayChangeIp": 4
  },
  "ip": {
    "proxyList": "1.2.3.4:8080:u:p\n5.6.7.8:8080:u:p\n9.10.11.12:8080:u:p",
    "proxyType": "http",
    "tinsoftKeys": "tinsoft_abc123",
    "tinsoftThreadPerIp": 2,
    "fptKeys": "fpt_xyz",
    "xproxyServiceUrl": "http://xproxy.test:8080",
    "xproxyType": "socks5",
    "xproxyThreadPerIp": 3,
    "xproxyRunType": "shared",
    "shoplikeKeys": "shoplike_k",
    "netproxyKeys": "netproxy_k",
    "minproxyKeys": "minproxy_k",
    "netproxyGbKey": "netproxygb_k",
    "proxyPopularKeys": "pp_key",
    "proxyPopularAccessToken": "pp_token",
    "proxyFarmKeys": "pf_key",
    "proxyFarmAccessToken": "pf_token"
  }
}`

const fullLegacyInteractionJSON = `{
  "verifyEnabled": true,
  "mailProvider": "store1s",
  "mailList": "",
  "checkLiveDieEnabled": true,
  "timeDelayCheck": 10,
  "timeDelaySendCode": 8,
  "sendAgainCode": true,
  "outputPath": "D:\\output\\live",
  "uaIphoneList": "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2)\nMozilla/5.0 (iPhone; CPU iPhone OS 16_0)",
  "zeusXApiKey": "zeus_pipe",
  "zeusXAccountCode": "ZC01",
  "dvfbApiKey": "dvfb_pipe",
  "dvfbAccountType": "2",
  "store1sApiKey": "store_pipe",
  "store1sProductId": "40559",
  "mail30sApiKey": "m30_pipe",
  "mail30sProductSlug": "hotmail-oauth2",
  "createEnabled": true,
  "createType": "normal",
  "createCookieList": "cookie1\ncookie2\ncookie3",
  "createOutputPath": "D:\\output\\created"
}`

// parseJSON là helper giả lập app.go ImportLegacyConfig: unmarshal JSON → struct
func parseJSON(t *testing.T, generalJSON, interactionJSON string) (adapter.LegacySettingsData, adapter.LegacyInteractionConfig) {
	t.Helper()
	var s adapter.LegacySettingsData
	var ic adapter.LegacyInteractionConfig
	if generalJSON != "" {
		if err := json.Unmarshal([]byte(generalJSON), &s); err != nil {
			t.Fatalf("unmarshal general: %v", err)
		}
	}
	if interactionJSON != "" {
		if err := json.Unmarshal([]byte(interactionJSON), &ic); err != nil {
			t.Fatalf("unmarshal interaction: %v", err)
		}
	}
	return s, ic
}

// TestPipeline_FullRoundtrip kiểm tra toàn bộ pipeline: JSON → FromLegacy → SaveTo → LoadFrom
func TestPipeline_FullRoundtrip(t *testing.T) {
	s, ic := parseJSON(t, fullLegacyGeneralJSON, fullLegacyInteractionJSON)

	app := adapter.FromLegacy(s, ic)

	// Validate phải pass
	if err := validation.Validate(app); err != nil {
		t.Fatalf("FromLegacy result failed validation: %v", err)
	}

	dir := t.TempDir()
	if err := store.SaveTo(dir, app); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	loaded, err := store.LoadFrom(dir)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	p := loaded.GetActiveProfile()
	if p == nil {
		t.Fatal("active profile nil after full roundtrip")
	}

	// Runtime
	if p.Runtime.ThreadRequest != 12 {
		t.Errorf("threadRequest: got %d, want 12", p.Runtime.ThreadRequest)
	}
	if p.Runtime.DelayRequest != 800 {
		t.Errorf("delayRequest: got %d, want 800", p.Runtime.DelayRequest)
	}
	if p.Runtime.ThreadCheckInfo != 6 {
		t.Errorf("threadCheckInfo: got %d, want 6", p.Runtime.ThreadCheckInfo)
	}
	if !p.Runtime.CheckIpBeforeRun {
		t.Error("checkIpBeforeRun: got false, want true")
	}
	if p.Runtime.DelayChangeIp != 4 {
		t.Errorf("delayChangeIp: got %d, want 4", p.Runtime.DelayChangeIp)
	}

	// Global
	if loaded.Global.LoginPlatform != "facebook" {
		t.Errorf("loginPlatform: got %s", loaded.Global.LoginPlatform)
	}
	if !loaded.Global.SaveRunColumn {
		t.Error("saveRunColumn: got false, want true")
	}
	if !loaded.Global.CloseAfterDone {
		t.Error("closeAfterDone: got false, want true")
	}

	// Account
	if p.Account.Source != "api" {
		t.Errorf("account.source: got %s, want api", p.Account.Source)
	}
	if p.Account.CloneHV.Username != "pipeuser" {
		t.Errorf("cloneHv.username: got %s", p.Account.CloneHV.Username)
	}
	if p.Account.CloneHV.Password != "pipepass" {
		t.Errorf("cloneHv.password: got %s", p.Account.CloneHV.Password)
	}
	if p.Account.CloneHV.Amount != 2 {
		t.Errorf("cloneHv.amount: got %d, want 2", p.Account.CloneHV.Amount)
	}

	// Proxy
	if p.Proxy.Provider != "tinsoft" {
		t.Errorf("proxy.provider: got %s, want tinsoft", p.Proxy.Provider)
	}
	if p.Proxy.Providers["tinsoft"].Keys != "tinsoft_abc123" {
		t.Errorf("tinsoft.keys: got %s", p.Proxy.Providers["tinsoft"].Keys)
	}
	if p.Proxy.Providers["fpt"].Keys != "fpt_xyz" {
		t.Errorf("fpt.keys: got %s", p.Proxy.Providers["fpt"].Keys)
	}
	if p.Proxy.Providers["xproxy"].ServiceURL != "http://xproxy.test:8080" {
		t.Errorf("xproxy.serviceUrl: got %s", p.Proxy.Providers["xproxy"].ServiceURL)
	}
	if p.Proxy.Providers["shoplike"].Keys != "shoplike_k" {
		t.Errorf("shoplike.keys: got %s", p.Proxy.Providers["shoplike"].Keys)
	}

	// Verify
	if !p.Verify.Enabled {
		t.Error("verify.enabled: got false, want true")
	}
	if p.Verify.TimeDelayCheck != 10 {
		t.Errorf("verify.timeDelayCheck: got %d, want 10", p.Verify.TimeDelayCheck)
	}
	if !p.Verify.SendAgainCode {
		t.Error("verify.sendAgainCode: got false, want true")
	}

	// Mail
	if p.Mail.Provider != "store1s" {
		t.Errorf("mail.provider: got %s, want store1s", p.Mail.Provider)
	}
	if p.Mail.Providers["store1s"].APIKey != "store_pipe" {
		t.Errorf("store1s.apiKey: got %s", p.Mail.Providers["store1s"].APIKey)
	}
	if p.Mail.Providers["mail30s"].ProductSlug != "hotmail-oauth2" {
		t.Errorf("mail30s.productSlug: got %s", p.Mail.Providers["mail30s"].ProductSlug)
	}

	// Captcha
	if p.Captcha.Provider != "capsolver" {
		t.Errorf("captcha.provider: got %s, want capsolver", p.Captcha.Provider)
	}
	if p.Captcha.Keys["capsolver"] != "CAP123" {
		t.Errorf("captcha.keys.capsolver: got %s, want CAP123", p.Captcha.Keys["capsolver"])
	}

	// Register
	if !p.Register.Enabled {
		t.Error("register.enabled: got false, want true")
	}
	if p.Register.Type != "normal" {
		t.Errorf("register.type: got %s, want normal", p.Register.Type)
	}

	// Output
	if p.Output.VerifyPath != "D:\\output\\live" {
		t.Errorf("output.verifyPath: got %s", p.Output.VerifyPath)
	}
	if p.Output.RegisterPath != "D:\\output\\created" {
		t.Errorf("output.registerPath: got %s", p.Output.RegisterPath)
	}

	// Device
	if p.Device.UAList == "" {
		t.Error("device.uaList: got empty, want non-empty")
	}
}

// TestPipeline_OnlyGeneralJSON khi chỉ có general.json, interaction defaults được dùng
// Dùng accountSource="folder" để tránh CloneHV username validation
func TestPipeline_OnlyGeneralJSON(t *testing.T) {
	// general JSON với folder source (không cần CloneHV username)
	folderGeneralJSON := `{
  "general": {
    "threadRequest": 12,
    "delayRequest": 800,
    "threadCheckInfo": 6,
    "loginPlatform": "facebook",
    "accountSource": "folder",
    "captchaProvider": "capsolver",
    "captchaKeys": { "capsolver": "CAP123", "2captcha": "", "ezcaptcha": "", "omocaptcha": "" },
    "ipProvider": "none"
  },
  "ip": {}
}`
	s, ic := parseJSON(t, folderGeneralJSON, "")

	app := adapter.FromLegacy(s, ic)
	if err := validation.Validate(app); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	p := app.GetActiveProfile()
	if p.Runtime.ThreadRequest != 12 {
		t.Errorf("threadRequest: got %d, want 12", p.Runtime.ThreadRequest)
	}
	if p.Mail.Providers == nil {
		t.Error("mail.providers nil when only general JSON provided")
	}
}

// TestPipeline_OnlyInteractionJSON khi chỉ có interaction.json, general defaults được dùng
func TestPipeline_OnlyInteractionJSON(t *testing.T) {
	s, ic := parseJSON(t, "", fullLegacyInteractionJSON)

	app := adapter.FromLegacy(s, ic)
	if err := validation.Validate(app); err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	p := app.GetActiveProfile()
	if p == nil {
		t.Fatal("profile nil")
	}
	// interaction data được map đúng
	if p.Mail.Provider != "store1s" {
		t.Errorf("mail.provider: got %s, want store1s", p.Mail.Provider)
	}
	// runtime defaults (0 values từ empty struct)
	if p.Runtime.ThreadRequest < 0 {
		t.Errorf("threadRequest âm: %d", p.Runtime.ThreadRequest)
	}
}

// TestPipeline_SaveLoad_PreservesProxyProvidersMap sau roundtrip map Providers không nil
func TestPipeline_SaveLoad_PreservesProxyProvidersMap(t *testing.T) {
	s, ic := parseJSON(t, fullLegacyGeneralJSON, fullLegacyInteractionJSON)
	app := adapter.FromLegacy(s, ic)

	dir := t.TempDir()
	if err := store.SaveTo(dir, app); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}
	loaded, err := store.LoadFrom(dir)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	p := loaded.GetActiveProfile()
	if p.Proxy.Providers == nil {
		t.Error("proxy.providers nil after roundtrip")
	}
	if p.Mail.Providers == nil {
		t.Error("mail.providers nil after roundtrip")
	}
	if p.Captcha.Keys == nil {
		t.Error("captcha.keys nil after roundtrip")
	}
}

// TestPipeline_BuildReport_MatchesMigratedFields report field count phải nhất quán với migration
func TestPipeline_BuildReport_MatchesMigratedFields(t *testing.T) {
	s, ic := parseJSON(t, fullLegacyGeneralJSON, fullLegacyInteractionJSON)

	report := adapter.BuildMappingReport(s, ic)

	// Sau khi parse đầy đủ, phải có ít nhất 15 MappedOk fields
	if len(report.MappedOk) < 15 {
		t.Errorf("mappedOk: got %d, want >= 15", len(report.MappedOk))
	}
	// Không được có ParseErrors
	if len(report.ParseErrors) > 0 {
		t.Errorf("unexpected parseErrors: %v", report.ParseErrors)
	}
	// Các path fields phải vào NeedsConfirm (không thể xác nhận path trên máy khác)
	confirmKeys := map[string]bool{}
	for _, f := range report.NeedsConfirm {
		confirmKeys[f.LegacyKey] = true
	}
	if !confirmKeys["accountSourcePath"] {
		t.Error("accountSourcePath should be in needsConfirm")
	}
	if !confirmKeys["outputPath"] {
		t.Error("outputPath should be in needsConfirm")
	}
	// capsolver là sensitive
	sensitiveKeys := map[string]bool{}
	for _, f := range report.Sensitive {
		sensitiveKeys[f.LegacyKey] = true
	}
	if !sensitiveKeys["captchaKeys.capsolver"] {
		t.Error("captchaKeys.capsolver should be in sensitive")
	}
}
