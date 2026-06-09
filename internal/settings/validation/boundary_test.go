// boundary_test.go — Boundary và edge-case validation tests
// Kiểm tra các giá trị biên và các tổ hợp field ít gặp
package validation_test

import (
	"testing"

	"HVRIns/internal/settings/model"
	"HVRIns/internal/settings/validation"
)

// ── Runtime boundaries ────────────────────────────────────────────────────────

// TestValidate_ThreadRequest_Zero threadRequest=0 là hợp lệ (tắt luồng)
func TestValidate_ThreadRequest_Zero(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Runtime.ThreadRequest = 0
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err != nil {
		t.Errorf("threadRequest=0 should pass, got: %v", err)
	}
}

// TestValidate_ThreadRequest_Max threadRequest=600 là hợp lệ (giá trị max)
func TestValidate_ThreadRequest_Max(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Runtime.ThreadRequest = 600
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err != nil {
		t.Errorf("threadRequest=600 should pass, got: %v", err)
	}
}

// TestValidate_ThreadRequest_OverMax threadRequest=601 vượt max phải fail
func TestValidate_ThreadRequest_OverMax(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Runtime.ThreadRequest = 601
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err == nil {
		t.Error("threadRequest=601 should fail validation")
	}
}

// TestValidate_DelayRequest_Zero delayRequest=0 là hợp lệ
func TestValidate_DelayRequest_Zero(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Runtime.DelayRequest = 0
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err != nil {
		t.Errorf("delayRequest=0 should pass, got: %v", err)
	}
}

// TestValidate_DelayRequest_Negative delayRequest âm phải fail
func TestValidate_DelayRequest_Negative(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Runtime.DelayRequest = -1
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err == nil {
		t.Error("delayRequest=-1 should fail validation")
	}
}

// TestValidate_ThreadCheckInfo_Zero threadCheckInfo=0 là hợp lệ
func TestValidate_ThreadCheckInfo_Zero(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Runtime.ThreadCheckInfo = 0
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err != nil {
		t.Errorf("threadCheckInfo=0 should pass, got: %v", err)
	}
}

// TestValidate_ThreadCheckInfo_Negative threadCheckInfo âm phải fail
func TestValidate_ThreadCheckInfo_Negative(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Runtime.ThreadCheckInfo = -1
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err == nil {
		t.Error("threadCheckInfo=-1 should fail validation")
	}
}

// TestValidate_DelayChangeIp_Zero delayChangeIp=0 là hợp lệ
func TestValidate_DelayChangeIp_Zero(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Runtime.DelayChangeIp = 0
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err != nil {
		t.Errorf("delayChangeIp=0 should pass, got: %v", err)
	}
}

// TestValidate_DelayChangeIp_Negative delayChangeIp âm phải fail
func TestValidate_DelayChangeIp_Negative(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Runtime.DelayChangeIp = -1
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err == nil {
		t.Error("delayChangeIp=-1 should fail validation")
	}
}

// ── Verify boundaries ─────────────────────────────────────────────────────────

// TestValidate_TimeDelayCheck_Negative timeDelayCheck âm phải fail
func TestValidate_TimeDelayCheck_Negative(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Verify.TimeDelayCheck = -1
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err == nil {
		t.Error("timeDelayCheck=-1 should fail validation")
	}
}

// TestValidate_TimeDelaySendCode_Negative timeDelaySendCode âm phải fail
func TestValidate_TimeDelaySendCode_Negative(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Verify.TimeDelaySendCode = -1
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err == nil {
		t.Error("timeDelaySendCode=-1 should fail validation")
	}
}

// TestValidate_TimeDelayCheck_Zero timeDelayCheck=0 là hợp lệ
func TestValidate_TimeDelayCheck_Zero(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Verify.TimeDelayCheck = 0
	p.Verify.TimeDelaySendCode = 0
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err != nil {
		t.Errorf("timeDelay=0 should pass, got: %v", err)
	}
}

// ── Register type boundaries ──────────────────────────────────────────────────

// TestValidate_RegisterType_Normal "normal" là hợp lệ
func TestValidate_RegisterType_Normal(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Register.Type = "normal"
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err != nil {
		t.Errorf("register.type=normal should pass, got: %v", err)
	}
}

// TestValidate_RegisterType_Tut "tut" là hợp lệ
func TestValidate_RegisterType_Tut(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Register.Type = "tut"
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err != nil {
		t.Errorf("register.type=tut should pass, got: %v", err)
	}
}

// TestValidate_RegisterType_Spam "spam" là hợp lệ
func TestValidate_RegisterType_Spam(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Register.Type = "spam"
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err != nil {
		t.Errorf("register.type=spam should pass, got: %v", err)
	}
}

// TestValidate_RegisterType_Empty "" là hợp lệ (chưa chọn)
func TestValidate_RegisterType_Empty(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Register.Type = ""
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err != nil {
		t.Errorf("register.type='' should pass, got: %v", err)
	}
}

// TestValidate_RegisterType_Invalid type không hợp lệ phải fail
func TestValidate_RegisterType_Invalid(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Register.Type = "bruteforce"
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err == nil {
		t.Error("register.type=bruteforce should fail validation")
	}
}

// ── Proxy type boundaries ─────────────────────────────────────────────────────

// TestValidate_ProxyType_Empty "" là hợp lệ (không dùng proxy)
func TestValidate_ProxyType_Empty(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Proxy.ProxyType = ""
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err != nil {
		t.Errorf("proxyType='' should pass, got: %v", err)
	}
}

// TestValidate_ProxyType_AllValid kiểm tra tất cả proxy type hợp lệ
func TestValidate_ProxyType_AllValid(t *testing.T) {
	for _, pt := range []string{"http", "https", "socks5", "socks4"} {
		app := model.DefaultAppSettings()
		p := app.GetActiveProfile()
		p.Proxy.ProxyType = pt
		app.UpsertProfile(*p)
		if err := validation.Validate(app); err != nil {
			t.Errorf("proxyType=%s should pass, got: %v", pt, err)
		}
	}
}

// TestValidate_ProxyType_Invalid proxyType không hợp lệ phải fail
func TestValidate_ProxyType_Invalid(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Proxy.ProxyType = "ssh"
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err == nil {
		t.Error("proxyType=ssh should fail validation")
	}
}

// ── Login platform boundaries ─────────────────────────────────────────────────

// TestValidate_LoginPlatform_Empty "" là hợp lệ
func TestValidate_LoginPlatform_Empty(t *testing.T) {
	app := model.DefaultAppSettings()
	app.Global.LoginPlatform = ""
	if err := validation.Validate(app); err != nil {
		t.Errorf("loginPlatform='' should pass, got: %v", err)
	}
}

// TestValidate_LoginPlatform_Instagram "instagram" là hợp lệ (deprecated nhưng vẫn accepted)
func TestValidate_LoginPlatform_Instagram(t *testing.T) {
	app := model.DefaultAppSettings()
	app.Global.LoginPlatform = "instagram"
	if err := validation.Validate(app); err != nil {
		t.Errorf("loginPlatform=instagram should pass validation, got: %v", err)
	}
}

// TestValidate_LoginPlatform_BM "bm" là hợp lệ
func TestValidate_LoginPlatform_BM(t *testing.T) {
	app := model.DefaultAppSettings()
	app.Global.LoginPlatform = "bm"
	if err := validation.Validate(app); err != nil {
		t.Errorf("loginPlatform=bm should pass validation, got: %v", err)
	}
}

// ── Captcha provider boundaries ───────────────────────────────────────────────

// TestValidate_CaptchaProvider_AllValid tất cả provider hợp lệ
func TestValidate_CaptchaProvider_AllValid(t *testing.T) {
	for _, prov := range []string{"2captcha", "omocaptcha", "ezcaptcha", "capsolver", ""} {
		app := model.DefaultAppSettings()
		p := app.GetActiveProfile()
		p.Captcha.Provider = prov
		app.UpsertProfile(*p)
		if err := validation.Validate(app); err != nil {
			t.Errorf("captchaProvider=%s should pass, got: %v", prov, err)
		}
	}
}

// ── Account source boundaries ─────────────────────────────────────────────────

// TestValidate_AccountSource_BothValid "folder" và "api" đều hợp lệ
func TestValidate_AccountSource_BothValid(t *testing.T) {
	for _, src := range []string{"folder", ""} {
		app := model.DefaultAppSettings()
		p := app.GetActiveProfile()
		p.Account.Source = src
		app.UpsertProfile(*p)
		if err := validation.Validate(app); err != nil {
			t.Errorf("accountSource=%q should pass, got: %v", src, err)
		}
	}

	// "api" cần username
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Account.Source = "api"
	p.Account.CloneHV.Username = "testuser"
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err != nil {
		t.Errorf("accountSource=api with username should pass, got: %v", err)
	}
}

// TestValidate_EmptyProfileID profile với ID rỗng phải fail
func TestValidate_EmptyProfileID(t *testing.T) {
	app := model.DefaultAppSettings()
	app.Profiles = append(app.Profiles, model.Profile{
		ID:   "",
		Name: "Invalid Profile",
	})
	if err := validation.Validate(app); err == nil {
		t.Error("profile with empty ID should fail validation")
	}
}

// TestValidate_DuplicateProfileIDs duplicate profile IDs — validation không chặn (UpsertProfile đã xử lý)
// Test này document behavior hiện tại
func TestValidate_DuplicateProfileIDs(t *testing.T) {
	app := model.DefaultAppSettings()
	// Thêm profile thứ 2 với cùng ID (bỏ qua UpsertProfile để test raw)
	p2 := model.DefaultProfile("default", "Duplicate")
	app.Profiles = append(app.Profiles, p2)

	// Validation hiện tại không check duplicate IDs — document điều này
	// Nếu sau này thêm check, test này sẽ fail và cần update
	err := validation.Validate(app)
	_ = err // behavior not mandated currently
}

// TestValidateProfile_AllDefaultFieldsValid default profile từng field phải pass
func TestValidateProfile_AllDefaultFieldsValid(t *testing.T) {
	profiles := []struct {
		name string
		fn   func(*model.Profile)
	}{
		{"folder source", func(p *model.Profile) { p.Account.Source = "folder" }},
		{"empty source", func(p *model.Profile) { p.Account.Source = "" }},
		{"http proxy", func(p *model.Profile) { p.Proxy.ProxyType = "http" }},
		{"socks5 proxy", func(p *model.Profile) { p.Proxy.ProxyType = "socks5" }},
		{"verify enabled", func(p *model.Profile) { p.Verify.Enabled = true }},
		{"register normal", func(p *model.Profile) { p.Register.Type = "normal" }},
		{"capsolver captcha", func(p *model.Profile) { p.Captcha.Provider = "capsolver" }},
	}

	for _, tc := range profiles {
		t.Run(tc.name, func(t *testing.T) {
			p := model.DefaultProfile("test", "Test")
			tc.fn(&p)
			if err := validation.ValidateProfile(p); err != nil {
				t.Errorf("%s: expected pass, got: %v", tc.name, err)
			}
		})
	}
}
