package validation_test

import (
	"testing"

	"HVRIns/internal/settings/model"
	"HVRIns/internal/settings/validation"
)

// TestValidate_DefaultSettings default settings phải pass
func TestValidate_DefaultSettings(t *testing.T) {
	app := model.DefaultAppSettings()
	if err := validation.Validate(app); err != nil {
		t.Errorf("DefaultAppSettings() should be valid, got: %v", err)
	}
}

// TestValidate_MissingVersion version < 1 phải fail
func TestValidate_MissingVersion(t *testing.T) {
	app := model.DefaultAppSettings()
	app.Version = 0
	if err := validation.Validate(app); err == nil {
		t.Error("version=0 should fail validation")
	}
}

// TestValidate_EmptyActiveProfileID activeProfileId rỗng phải fail
func TestValidate_EmptyActiveProfileID(t *testing.T) {
	app := model.DefaultAppSettings()
	app.ActiveProfileID = ""
	if err := validation.Validate(app); err == nil {
		t.Error("empty activeProfileId should fail validation")
	}
}

// TestValidate_NoProfiles phải có ít nhất 1 profile
func TestValidate_NoProfiles(t *testing.T) {
	app := model.DefaultAppSettings()
	app.Profiles = nil
	if err := validation.Validate(app); err == nil {
		t.Error("no profiles should fail validation")
	}
}

// TestValidate_ActiveProfileNotFound activeProfileId không tồn tại trong profiles phải fail
func TestValidate_ActiveProfileNotFound(t *testing.T) {
	app := model.DefaultAppSettings()
	app.ActiveProfileID = "nonexistent"
	if err := validation.Validate(app); err == nil {
		t.Error("activeProfileId not found in profiles should fail validation")
	}
}

// TestValidate_InvalidLoginPlatform loginPlatform không hợp lệ phải fail
func TestValidate_InvalidLoginPlatform(t *testing.T) {
	app := model.DefaultAppSettings()
	app.Global.LoginPlatform = "twitter"
	if err := validation.Validate(app); err == nil {
		t.Error("invalid loginPlatform should fail validation")
	}
}

// TestValidate_ThreadRequestTooHigh threadRequest > 600 phải fail
func TestValidate_ThreadRequestTooHigh(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Runtime.ThreadRequest = 700
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err == nil {
		t.Error("threadRequest=700 should fail validation")
	}
}

// TestValidate_ThreadRequestNegative threadRequest âm phải fail
func TestValidate_ThreadRequestNegative(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Runtime.ThreadRequest = -1
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err == nil {
		t.Error("threadRequest=-1 should fail validation")
	}
}

// TestValidate_InvalidAccountSource accountSource không hợp lệ phải fail
func TestValidate_InvalidAccountSource(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Account.Source = "database"
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err == nil {
		t.Error("invalid accountSource should fail validation")
	}
}

// TestValidate_ApiSourceWithoutUsername source=api mà không có username vẫn phải pass
// (credentials được cấu hình riêng, không bắt buộc lúc save settings)
func TestValidate_ApiSourceWithoutUsername(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Account.Source = "api"
	p.Account.CloneHV.Username = ""
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err != nil {
		t.Errorf("source=api without username should pass validation, got: %v", err)
	}
}

// TestValidate_ApiSourceWithUsername source=api có username phải pass
func TestValidate_ApiSourceWithUsername(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Account.Source = "api"
	p.Account.CloneHV.Username = "user123"
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err != nil {
		t.Errorf("source=api with username should pass, got: %v", err)
	}
}

// TestValidate_InvalidProxyType proxyType không hợp lệ phải fail
func TestValidate_InvalidProxyType(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Proxy.ProxyType = "ftp"
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err == nil {
		t.Error("invalid proxyType should fail validation")
	}
}

// TestValidate_InvalidCaptchaProvider captchaProvider không hợp lệ phải fail
func TestValidate_InvalidCaptchaProvider(t *testing.T) {
	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Captcha.Provider = "anticaptcha"
	app.UpsertProfile(*p)
	if err := validation.Validate(app); err == nil {
		t.Error("invalid captchaProvider should fail validation")
	}
}

// TestValidate_ValidFacebookPlatform "facebook" phải pass
func TestValidate_ValidFacebookPlatform(t *testing.T) {
	app := model.DefaultAppSettings()
	app.Global.LoginPlatform = "facebook"
	if err := validation.Validate(app); err != nil {
		t.Errorf("loginPlatform=facebook should pass, got: %v", err)
	}
}

// TestValidate_MultipleErrors nhiều lỗi cùng lúc
func TestValidate_MultipleErrors(t *testing.T) {
	app := model.AppSettings{
		Version:         0, // lỗi
		ActiveProfileID: "", // lỗi
		Profiles:        nil, // lỗi
	}
	err := validation.Validate(app)
	if err == nil {
		t.Fatal("multiple errors should fail validation")
	}
	ve, ok := err.(*validation.ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if len(ve.Errors) < 3 {
		t.Errorf("expected at least 3 errors, got %d: %v", len(ve.Errors), ve.Errors)
	}
}

// TestValidateProfile_Standalone kiểm tra ValidateProfile standalone
func TestValidateProfile_Standalone(t *testing.T) {
	p := model.DefaultProfile("test", "Test")
	if err := validation.ValidateProfile(p); err != nil {
		t.Errorf("default profile should be valid, got: %v", err)
	}
}
