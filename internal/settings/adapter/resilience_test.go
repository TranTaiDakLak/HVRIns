// resilience_test.go — Guard chống crash khi nhận input lạ/rỗng/thiếu field
// Các test này đảm bảo system không panic với bất kỳ input edge case nào
package adapter_test

import (
	"testing"

	"HVRIns/internal/settings/adapter"
	"HVRIns/internal/settings/validation"
)

// TestFromLegacy_NilCaptchaKeysMap không panic khi CaptchaKeys là nil
func TestFromLegacy_NilCaptchaKeysMap(t *testing.T) {
	s := adapter.LegacySettingsData{
		General: adapter.LegacyGeneralConfig{
			ThreadRequest:   5,
			LoginPlatform:   "facebook",
			CaptchaProvider: "2captcha",
			CaptchaKeys:     nil, // explicit nil map
		},
	}
	// Không được panic
	app := adapter.FromLegacy(s, adapter.LegacyInteractionConfig{})
	p := app.GetActiveProfile()
	if p == nil {
		t.Fatal("profile nil with nil CaptchaKeys")
	}
	if p.Captcha.Keys == nil {
		t.Error("captcha.keys should be initialized even when input map is nil")
	}
}

// TestFromLegacy_NilProxyProviders không panic khi Providers map chưa được khởi tạo
func TestFromLegacy_NilProxyProviders(t *testing.T) {
	s := adapter.LegacySettingsData{
		General: adapter.LegacyGeneralConfig{
			IpProvider:    "proxy",
			ThreadRequest: 5,
		},
		Ip: adapter.LegacyIpConfig{
			ProxyList: "",
			ProxyType: "",
			// tất cả keys để rỗng
		},
	}
	app := adapter.FromLegacy(s, adapter.LegacyInteractionConfig{})
	p := app.GetActiveProfile()
	if p == nil {
		t.Fatal("profile nil")
	}
	if p.Proxy.Providers == nil {
		t.Error("proxy.providers should be non-nil map even with empty keys")
	}
}

// TestFromLegacy_EmptyMailProviders không panic khi mail provider config rỗng
func TestFromLegacy_EmptyMailProviders(t *testing.T) {
	ic := adapter.LegacyInteractionConfig{
		VerifyEnabled: true,
		MailProvider:  "@tmpbox.net",
		// tất cả API keys rỗng
	}
	app := adapter.FromLegacy(adapter.LegacySettingsData{}, ic)
	p := app.GetActiveProfile()
	if p == nil {
		t.Fatal("profile nil")
	}
	if p.Mail.Providers == nil {
		t.Error("mail.providers should not be nil")
	}
}

// TestFromLegacy_UnknownIpProvider provider không tồn tại — không panic, giữ nguyên string
func TestFromLegacy_UnknownIpProvider(t *testing.T) {
	s := adapter.LegacySettingsData{
		General: adapter.LegacyGeneralConfig{
			IpProvider: "unknown_future_provider_xyz",
		},
	}
	// Không panic
	app := adapter.FromLegacy(s, adapter.LegacyInteractionConfig{})
	p := app.GetActiveProfile()
	if p.Proxy.Provider != "unknown_future_provider_xyz" {
		t.Errorf("provider: got %s, want unknown_future_provider_xyz", p.Proxy.Provider)
	}
}

// TestFromLegacy_MaxThreadRequest threadRequest lớn không panic (validation bắt sau)
func TestFromLegacy_MaxThreadRequest(t *testing.T) {
	s := adapter.LegacySettingsData{
		General: adapter.LegacyGeneralConfig{
			ThreadRequest: 9999, // quá lớn, validation sẽ fail — nhưng FromLegacy không panic
		},
	}
	app := adapter.FromLegacy(s, adapter.LegacyInteractionConfig{})
	p := app.GetActiveProfile()
	if p.Runtime.ThreadRequest != 9999 {
		t.Errorf("threadRequest should be preserved as-is from legacy, got %d", p.Runtime.ThreadRequest)
	}
	// Validation đúng phải reject giá trị này
	if err := validation.Validate(app); err == nil {
		t.Error("threadRequest=9999 should fail validation")
	}
}

// TestFromLegacy_NegativeDelays giá trị âm không panic (validation bắt sau)
func TestFromLegacy_NegativeDelays(t *testing.T) {
	s := adapter.LegacySettingsData{
		General: adapter.LegacyGeneralConfig{
			ThreadRequest:  5,
			DelayRequest:   -100,
			DelayChangeIp:  -5,
			ThreadCheckInfo: -2,
		},
	}
	app := adapter.FromLegacy(s, adapter.LegacyInteractionConfig{})
	if err := validation.Validate(app); err == nil {
		t.Error("negative delays/threads should fail validation")
	}
}

// TestFromLegacy_UnicodeInPaths unicode paths không panic
func TestFromLegacy_UnicodeInPaths(t *testing.T) {
	s := adapter.LegacySettingsData{
		General: adapter.LegacyGeneralConfig{
			AccountSourcePath: "C:\\Users\\Nguyễn Văn A\\tài khoản fb",
			ThreadRequest:     5,
		},
	}
	ic := adapter.LegacyInteractionConfig{
		OutputPath: "/home/đỗ thị b/output",
	}
	app := adapter.FromLegacy(s, ic)
	p := app.GetActiveProfile()
	if p.Account.FolderPath != "C:\\Users\\Nguyễn Văn A\\tài khoản fb" {
		t.Errorf("folderPath unicode: got %s", p.Account.FolderPath)
	}
	if p.Output.VerifyPath != "/home/đỗ thị b/output" {
		t.Errorf("output.verifyPath unicode: got %s", p.Output.VerifyPath)
	}
}

// TestFromLegacy_LargeProxyList proxy list với nhiều dòng không panic
func TestFromLegacy_LargeProxyList(t *testing.T) {
	proxyList := ""
	for i := 0; i < 1000; i++ {
		proxyList += "1.2.3.4:8080:user:pass\n"
	}
	s := adapter.LegacySettingsData{
		General: adapter.LegacyGeneralConfig{ThreadRequest: 5},
		Ip: adapter.LegacyIpConfig{
			ProxyList: proxyList,
			ProxyType: "http",
		},
	}
	app := adapter.FromLegacy(s, adapter.LegacyInteractionConfig{})
	p := app.GetActiveProfile()
	if p.Proxy.ProxyList == "" {
		t.Error("proxyList should not be empty for 1000-line input")
	}
}

// TestFromLegacy_CloneHVAmountZero amount=0 map đúng (valid, berarti default)
func TestFromLegacy_CloneHVAmountZero(t *testing.T) {
	s := adapter.LegacySettingsData{
		General: adapter.LegacyGeneralConfig{
			AccountSource:   "api",
			CloneHVUsername: "user",
			CloneHVAmount:   0,
		},
	}
	app := adapter.FromLegacy(s, adapter.LegacyInteractionConfig{})
	p := app.GetActiveProfile()
	if p.Account.CloneHV.Amount != 0 {
		t.Errorf("cloneHv.amount: got %d, want 0", p.Account.CloneHV.Amount)
	}
}

// TestBuildMappingReport_NoSensitiveFieldsWhenEmpty không có sensitive fields khi input rỗng
func TestBuildMappingReport_NoSensitiveFieldsWhenEmpty(t *testing.T) {
	r := adapter.BuildMappingReport(adapter.LegacySettingsData{}, adapter.LegacyInteractionConfig{})
	if len(r.Sensitive) != 0 {
		t.Errorf("no sensitive fields expected for empty input, got %d", len(r.Sensitive))
	}
}

// TestBuildMappingReport_MultipleSensitiveKeys nhiều captcha keys đều vào Sensitive
func TestBuildMappingReport_MultipleSensitiveKeys(t *testing.T) {
	s := adapter.LegacySettingsData{
		General: adapter.LegacyGeneralConfig{
			CaptchaProvider: "2captcha",
			CaptchaKeys: map[string]string{
				"2captcha":   "key_2captcha",
				"capsolver":  "key_capsolver",
				"ezcaptcha":  "key_ezcaptcha",
				"omocaptcha": "key_omocaptcha",
			},
		},
	}
	r := adapter.BuildMappingReport(s, adapter.LegacyInteractionConfig{})

	sensitiveKeys := map[string]bool{}
	for _, f := range r.Sensitive {
		sensitiveKeys[f.LegacyKey] = true
	}
	for _, prov := range []string{"2captcha", "capsolver", "ezcaptcha", "omocaptcha"} {
		key := "captchaKeys." + prov
		if !sensitiveKeys[key] {
			t.Errorf("%s should be in sensitive", key)
		}
	}
}

// TestBuildMappingReport_BMPlatformUnsupported platform=bm vào Unsupported
func TestBuildMappingReport_BMPlatformUnsupported(t *testing.T) {
	s := adapter.LegacySettingsData{
		General: adapter.LegacyGeneralConfig{
			LoginPlatform: "bm",
		},
	}
	r := adapter.BuildMappingReport(s, adapter.LegacyInteractionConfig{})
	found := false
	for _, f := range r.Unsupported {
		if f.LegacyKey == "loginPlatform" {
			found = true
		}
	}
	if !found {
		t.Error("loginPlatform=bm should be in unsupported")
	}
}

// TestBuildMappingReport_AllSensitiveDisplayMasked tất cả sensitive hiển thị "***"
func TestBuildMappingReport_AllSensitiveDisplayMasked(t *testing.T) {
	s := adapter.LegacySettingsData{
		General: adapter.LegacyGeneralConfig{
			CloneHVPassword: "secret",
		},
	}
	ic := adapter.LegacyInteractionConfig{
		ZeusXApiKey:   "key",
		DvfbApiKey:    "key",
		Store1sApiKey: "key",
		Mail30sApiKey: "key",
	}
	r := adapter.BuildMappingReport(s, ic)
	for _, f := range r.Sensitive {
		if f.DisplayValue != "***" {
			t.Errorf("sensitive field %q: displayValue=%q, want '***'", f.LegacyKey, f.DisplayValue)
		}
	}
}
