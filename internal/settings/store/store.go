// Package store — đọc/ghi AppSettings từ/sang disk.
// Phase 1: lưu file app_settings.json song song với general.json + interaction.json (cũ).
// Phase 4+: old files deprecated, app_settings.json là nguồn sự thật duy nhất.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"HVRIns/internal/settings/model"
	"HVRIns/internal/settings/validation"
)

const defaultFilename = "app_settings.json"

var mu sync.RWMutex

// LoadFrom đọc AppSettings từ thư mục dir.
// Trả về DefaultAppSettings() nếu file chưa tồn tại.
func LoadFrom(dir string) (model.AppSettings, error) {
	mu.RLock()
	defer mu.RUnlock()

	path := filepath.Join(dir, defaultFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.DefaultAppSettings(), nil
		}
		return model.DefaultAppSettings(), err
	}

	var app model.AppSettings
	if err := json.Unmarshal(data, &app); err != nil {
		return model.DefaultAppSettings(), err
	}

	// Điền default cho field bị thiếu (backward compat khi thêm field mới)
	applyDefaults(&app)
	return app, nil
}

// SaveTo lưu AppSettings xuống thư mục dir sau khi validate.
func SaveTo(dir string, app model.AppSettings) error {
	if err := validation.Validate(app); err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(app, "", "  ")
	if err != nil {
		return err
	}

	// 0600 — file chứa credentials/keys, chỉ owner đọc được
	return os.WriteFile(filepath.Join(dir, defaultFilename), data, 0600)
}

// Exists kiểm tra app_settings.json có tồn tại không
func Exists(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, defaultFilename))
	return err == nil
}

// applyDefaults điền giá trị mặc định cho các field bị thiếu hoặc zero-value.
// Dùng để backward-compat khi load file JSON cũ chưa có field mới (VD: thêm mail provider, captcha key).
// Gọi tự động sau khi json.Unmarshal trong LoadFrom.
//
// app: pointer tới AppSettings vừa đọc từ disk — sẽ bị modify in-place.
func applyDefaults(app *model.AppSettings) {
	if app.Version == 0 {
		app.Version = model.CurrentVersion
	}
	if app.ActiveProfileID == "" {
		app.ActiveProfileID = "default"
	}
	if app.SecretsRef.KeyRefs == nil {
		app.SecretsRef.KeyRefs = map[string]string{}
	}
	if len(app.Profiles) == 0 {
		app.Profiles = []model.Profile{model.DefaultProfile("default", "Default")}
	}
	for i := range app.Profiles {
		p := &app.Profiles[i]
		if p.Mail.Providers == nil {
			p.Mail.Providers = map[string]model.MailProviderCfg{}
		}
		// Điền các mail provider entry còn thiếu (backward compat: Phase 1 JSON không có entries)
		defaultMailProviders := map[string]model.MailProviderCfg{
			"zeusx":   {},
			"dvfb":    {AccountType: "1"},
			"store1s": {},
			"mail30s": {},
		}
		for prov, def := range defaultMailProviders {
			if _, ok := p.Mail.Providers[prov]; !ok {
				p.Mail.Providers[prov] = def
			}
		}
		if p.Proxy.Providers == nil {
			p.Proxy.Providers = map[string]model.ProxyProviderCfg{}
		}
		if p.Captcha.Keys == nil {
			p.Captcha.Keys = map[string]string{
				"2captcha": "", "omocaptcha": "", "ezcaptcha": "", "capsolver": "",
			}
		}
		// Điền các captcha key còn thiếu
		for _, k := range []string{"2captcha", "omocaptcha", "ezcaptcha", "capsolver"} {
			if _, ok := p.Captcha.Keys[k]; !ok {
				p.Captcha.Keys[k] = ""
			}
		}
	}
}
