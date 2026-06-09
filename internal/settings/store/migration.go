// migration.go — detect và migrate từ config cũ sang app_settings.json
package store

import (
	"encoding/json"
	"os"
	"path/filepath"

	"HVRIns/internal/settings/adapter"
	"HVRIns/internal/settings/model"
)

// MigrateIfNeeded kiểm tra xem app_settings.json đã tồn tại chưa.
// Nếu chưa, đọc general.json + interaction.json (old format) và migrate.
// Trả về (AppSettings, migrated, error).
// - migrated = true: đã migrate từ file cũ
// - migrated = false: đã có app_settings.json sẵn hoặc không có gì để migrate
func MigrateIfNeeded(settingsDir string) (model.AppSettings, bool, error) {
	// Nếu đã có file mới, load nó
	if Exists(settingsDir) {
		app, err := LoadFrom(settingsDir)
		return app, false, err
	}

	// Thử đọc file cũ
	generalPath := filepath.Join(settingsDir, "general.json")
	interactionPath := filepath.Join(settingsDir, "interaction.json")

	generalData, errG := os.ReadFile(generalPath)
	interactionData, errI := os.ReadFile(interactionPath)

	// Nếu không có file nào → dùng default, không cần migrate
	if os.IsNotExist(errG) && os.IsNotExist(errI) {
		return model.DefaultAppSettings(), false, nil
	}

	// Parse legacy files
	var legacySettings adapter.LegacySettingsData
	var legacyInteraction adapter.LegacyInteractionConfig

	if errG == nil {
		_ = json.Unmarshal(generalData, &legacySettings)
	}
	if errI == nil {
		_ = json.Unmarshal(interactionData, &legacyInteraction)
	}

	// Convert sang AppSettings
	app := adapter.FromLegacy(legacySettings, legacyInteraction)

	// Save file mới (best-effort — không block startup nếu thất bại)
	_ = SaveTo(settingsDir, app)

	return app, true, nil
}
