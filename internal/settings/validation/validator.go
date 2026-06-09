// Package validation — kiểm tra tính hợp lệ của AppSettings trước khi save
package validation

import (
	"fmt"
	"strings"

	"HVRIns/internal/settings/model"
)

// ValidationError chứa danh sách lỗi validation
type ValidationError struct {
	Errors []string
}

// Error implement interface error — trả về tất cả lỗi nối bởi "; ".
func (e *ValidationError) Error() string {
	return "validation failed: " + strings.Join(e.Errors, "; ")
}

// IsValid trả về true nếu không có lỗi
func (e *ValidationError) IsValid() bool {
	return len(e.Errors) == 0
}

// Validate kiểm tra toàn bộ AppSettings, trả về nil nếu hợp lệ
func Validate(a model.AppSettings) error {
	var errs []string

	if a.Version < 1 {
		errs = append(errs, "version must be >= 1")
	}
	if a.ActiveProfileID == "" {
		errs = append(errs, "activeProfileId must not be empty")
	}
	if len(a.Profiles) == 0 {
		errs = append(errs, "must have at least one profile")
	}

	// Kiểm tra global
	errs = append(errs, validateGlobal(a.Global)...)

	// Kiểm tra từng profile
	activeFound := false
	for _, p := range a.Profiles {
		if p.ID == "" {
			errs = append(errs, "profile.id must not be empty")
			continue
		}
		if p.ID == a.ActiveProfileID {
			activeFound = true
		}
		errs = append(errs, validateProfile(p)...)
	}
	if len(a.Profiles) > 0 && !activeFound {
		errs = append(errs, fmt.Sprintf("activeProfileId '%s' not found in profiles", a.ActiveProfileID))
	}

	if len(errs) == 0 {
		return nil
	}
	return &ValidationError{Errors: errs}
}

// ValidateProfile kiểm tra một profile đơn lẻ
func ValidateProfile(p model.Profile) error {
	errs := validateProfile(p)
	if len(errs) == 0 {
		return nil
	}
	return &ValidationError{Errors: errs}
}

// ─── internal helpers ──────────────────────────────────────────────────────

// validateGlobal kiểm tra GlobalSettings — chỉ validate loginPlatform nếu có giá trị.
// g: GlobalSettings cần kiểm tra (áp dụng cho toàn app, không theo profile).
func validateGlobal(g model.GlobalSettings) []string {
	var errs []string
	validPlatforms := map[string]bool{"facebook": true, "instagram": true, "bm": true}
	if g.LoginPlatform != "" && !validPlatforms[g.LoginPlatform] {
		errs = append(errs, fmt.Sprintf("global.loginPlatform '%s' is not valid", g.LoginPlatform))
	}
	return errs
}

// validateProfile chạy tất cả sub-validators cho một Profile.
// p: profile cần kiểm tra (ID, Runtime, Account, Proxy, Verify, Captcha, Register).
// Trả về slice lỗi rỗng nếu hợp lệ, mỗi lỗi có prefix "profile[ID].field".
func validateProfile(p model.Profile) []string {
	var errs []string
	prefix := fmt.Sprintf("profile[%s]", p.ID)

	errs = append(errs, validateRuntime(prefix, p.Runtime)...)
	errs = append(errs, validateAccount(prefix, p.Account)...)
	errs = append(errs, validateProxy(prefix, p.Proxy)...)
	errs = append(errs, validateVerify(prefix, p.Verify)...)
	errs = append(errs, validateCaptcha(prefix, p.Captcha)...)
	errs = append(errs, validateRegister(prefix, p.Register)...)

	return errs
}

// validateRuntime kiểm tra RuntimeSettings: threadRequest [0-600], các delay không âm.
// prefix: tiền tố lỗi dạng "profile[ID]" để dễ xác định profile nào bị lỗi.
// r: runtime config cần kiểm tra (threads, delays).
func validateRuntime(prefix string, r model.RuntimeSettings) []string {
	var errs []string
	if r.ThreadRequest < 0 {
		errs = append(errs, fmt.Sprintf("%s.runtime.threadRequest must be >= 0", prefix))
	}
	if r.ThreadRequest > 600 {
		errs = append(errs, fmt.Sprintf("%s.runtime.threadRequest must be <= 600", prefix))
	}
	if r.DelayRequest < 0 {
		errs = append(errs, fmt.Sprintf("%s.runtime.delayRequest must be >= 0", prefix))
	}
	if r.ThreadCheckInfo < 0 {
		errs = append(errs, fmt.Sprintf("%s.runtime.threadCheckInfo must be >= 0", prefix))
	}
	if r.DelayChangeIp < 0 {
		errs = append(errs, fmt.Sprintf("%s.runtime.delayChangeIp must be >= 0", prefix))
	}
	return errs
}

// validateAccount kiểm tra AccountSettings: source phải là "folder", "api" hoặc rỗng.
// prefix: tiền tố lỗi dạng "profile[ID]".
// a: account config cần kiểm tra (source, folderPath, cloneHv credentials).
func validateAccount(prefix string, a model.AccountSettings) []string {
	var errs []string
	// "file" mode mới: user pick 1 file .txt → load accounts vào grid → tick chọn để verify.
	validSources := map[string]bool{"folder": true, "file": true, "api": true, "": true}
	if !validSources[a.Source] {
		errs = append(errs, fmt.Sprintf("%s.account.source '%s' is not valid (must be 'folder', 'file', or 'api')", prefix, a.Source))
	}
	return errs
}

// validateProxy kiểm tra ProxySettings: proxyType phải là http/https/socks5/socks4 hoặc rỗng.
// prefix: tiền tố lỗi dạng "profile[ID]".
// p: proxy config cần kiểm tra.
func validateProxy(prefix string, p model.ProxySettings) []string {
	var errs []string
	validTypes := map[string]bool{"http": true, "https": true, "socks5": true, "socks4": true, "": true}
	if !validTypes[p.ProxyType] {
		errs = append(errs, fmt.Sprintf("%s.proxy.proxyType '%s' is not valid", prefix, p.ProxyType))
	}
	return errs
}

// validateVerify kiểm tra VerifySettings: timeDelayCheck và timeDelaySendCode không được âm.
// prefix: tiền tố lỗi dạng "profile[ID]".
// v: verify config cần kiểm tra (delays, live/die check).
func validateVerify(prefix string, v model.VerifySettings) []string {
	var errs []string
	if v.TimeDelayCheck < 0 {
		errs = append(errs, fmt.Sprintf("%s.verify.timeDelayCheck must be >= 0", prefix))
	}
	if v.TimeDelaySendCode < 0 {
		errs = append(errs, fmt.Sprintf("%s.verify.timeDelaySendCode must be >= 0", prefix))
	}
	return errs
}

// validateCaptcha kiểm tra CaptchaSettings: provider phải là 2captcha/omocaptcha/ezcaptcha/capsolver hoặc rỗng.
// prefix: tiền tố lỗi dạng "profile[ID]".
// c: captcha config cần kiểm tra.
func validateCaptcha(prefix string, c model.CaptchaSettings) []string {
	var errs []string
	validProviders := map[string]bool{
		"2captcha": true, "omocaptcha": true, "ezcaptcha": true, "capsolver": true, "": true,
	}
	if !validProviders[c.Provider] {
		errs = append(errs, fmt.Sprintf("%s.captcha.provider '%s' is not valid", prefix, c.Provider))
	}
	return errs
}

// validateRegister kiểm tra RegisterSettings: type phải là "spam", "tut", "normal" hoặc rỗng.
// prefix: tiền tố lỗi dạng "profile[ID]".
// r: register config cần kiểm tra.
func validateRegister(prefix string, r model.RegisterSettings) []string {
	var errs []string
	validTypes := map[string]bool{"spam": true, "tut": true, "normal": true, "": true}
	if !validTypes[r.Type] {
		errs = append(errs, fmt.Sprintf("%s.register.type '%s' is not valid (must be 'spam' or 'tut')", prefix, r.Type))
	}
	return errs
}
