// Package settings — public API cho settings platform.
//
// Sử dụng:
//
//	// Load (với auto-migrate từ config cũ nếu cần)
//	app, migrated, err := settings.Load("Config/Settings")
//
//	// Save
//	err := settings.Save("Config/Settings", app)
//
//	// Validate
//	err := settings.Validate(app)
//
// Phase 1: song song với general.json/interaction.json — không thay thế chúng.
// Phase 4+: app_settings.json trở thành nguồn sự thật duy nhất.
package settings

import (
	"HVRIns/internal/settings/model"
	"HVRIns/internal/settings/store"
	"HVRIns/internal/settings/validation"
)

// Load đọc AppSettings, tự động migrate từ file cũ nếu app_settings.json chưa tồn tại.
// Trả về (AppSettings, wasJustMigrated, error).
func Load(settingsDir string) (model.AppSettings, bool, error) {
	return store.MigrateIfNeeded(settingsDir)
}

// Save validate và lưu AppSettings xuống disk.
func Save(settingsDir string, app model.AppSettings) error {
	return store.SaveTo(settingsDir, app)
}

// Validate kiểm tra tính hợp lệ của AppSettings mà không save.
func Validate(app model.AppSettings) error {
	return validation.Validate(app)
}

// Default trả về AppSettings mặc định (cho new install).
func Default() model.AppSettings {
	return model.DefaultAppSettings()
}
