package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	appsettings "HVRIns/internal/settings"
	"HVRIns/internal/settings/adapter"
	"HVRIns/internal/settings/model"
)

// === Profile Management ===

// ProfileInfo — thông tin rút gọn của một profile trả về frontend
type ProfileInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// generateProfileID tạo unique ID cho profile mới
func generateProfileID() string {
	return fmt.Sprintf("p_%d", time.Now().UnixNano())
}

// ListProfiles trả về danh sách tất cả profiles
func (a *App) ListProfiles() []ProfileInfo {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	out := make([]ProfileInfo, len(a.appSettings.Profiles))
	for i, p := range a.appSettings.Profiles {
		out[i] = ProfileInfo{ID: p.ID, Name: p.Name}
	}
	return out
}

// GetActiveProfileID trả về ID của profile đang active
func (a *App) GetActiveProfileID() string {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	return a.appSettings.ActiveProfileID
}

// SetActiveProfile chuyển sang profile theo ID và lưu
func (a *App) SetActiveProfile(id string) string {
	a.settingsMu.Lock()
	found := false
	for _, p := range a.appSettings.Profiles {
		if p.ID == id {
			found = true
			break
		}
	}
	if !found {
		a.settingsMu.Unlock()
		return "Lỗi: profile không tồn tại"
	}
	a.appSettings.ActiveProfileID = id
	app := a.appSettings
	a.settingsMu.Unlock()
	if err := appsettings.Save("Config/Settings", app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}
	a.syncActiveProfileToFiles()
	return "OK"
}

// CreateProfile tạo profile mới từ default settings và kích hoạt nó
func (a *App) CreateProfile(name string) string {
	if name == "" {
		return "Lỗi: tên profile không được rỗng"
	}
	a.settingsMu.Lock()
	id := generateProfileID()
	p := model.DefaultProfile(id, name)
	a.appSettings.UpsertProfile(p)
	a.appSettings.ActiveProfileID = id
	app := a.appSettings
	a.settingsMu.Unlock()
	if err := appsettings.Save("Config/Settings", app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}
	a.syncActiveProfileToFiles()
	return id
}

// CloneProfile nhân bản profile đang active với tên mới và kích hoạt bản sao
func (a *App) CloneProfile(name string) string {
	if name == "" {
		return "Lỗi: tên profile không được rỗng"
	}
	a.settingsMu.Lock()
	src := a.appSettings.GetActiveProfile()
	if src == nil {
		a.settingsMu.Unlock()
		return "Lỗi: không có profile active"
	}
	cloned := *src
	cloned.ID = generateProfileID()
	cloned.Name = name
	a.appSettings.UpsertProfile(cloned)
	a.appSettings.ActiveProfileID = cloned.ID
	app := a.appSettings
	a.settingsMu.Unlock()
	if err := appsettings.Save("Config/Settings", app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}
	a.syncActiveProfileToFiles()
	return cloned.ID
}

// RenameProfile đổi tên profile theo ID
func (a *App) RenameProfile(id, name string) string {
	if name == "" {
		return "Lỗi: tên profile không được rỗng"
	}
	a.settingsMu.Lock()
	found := false
	for i := range a.appSettings.Profiles {
		if a.appSettings.Profiles[i].ID == id {
			a.appSettings.Profiles[i].Name = name
			found = true
			break
		}
	}
	if !found {
		a.settingsMu.Unlock()
		return "Lỗi: profile không tồn tại"
	}
	app := a.appSettings
	a.settingsMu.Unlock()
	if err := appsettings.Save("Config/Settings", app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}
	return "OK"
}

// DeleteProfile xóa profile theo ID — không thể xóa nếu chỉ còn 1
func (a *App) DeleteProfile(id string) string {
	a.settingsMu.Lock()
	if len(a.appSettings.Profiles) <= 1 {
		a.settingsMu.Unlock()
		return "Lỗi: phải giữ ít nhất 1 profile"
	}
	newProfiles := make([]model.Profile, 0, len(a.appSettings.Profiles)-1)
	for _, p := range a.appSettings.Profiles {
		if p.ID != id {
			newProfiles = append(newProfiles, p)
		}
	}
	if len(newProfiles) == len(a.appSettings.Profiles) {
		a.settingsMu.Unlock()
		return "Lỗi: profile không tồn tại"
	}
	a.appSettings.Profiles = newProfiles
	if a.appSettings.ActiveProfileID == id {
		a.appSettings.ActiveProfileID = newProfiles[0].ID
	}
	app := a.appSettings
	a.settingsMu.Unlock()
	if err := appsettings.Save("Config/Settings", app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}
	return "OK"
}

// syncActiveProfileToFiles ghi settings + interaction config của profile đang active
// xuống general.json và interaction.json, để LoadSettings/LoadInteractionConfig
// đọc đúng dữ liệu sau khi switch profile.
// Caller phải KHÔNG giữ settingsMu khi gọi hàm này.
func (a *App) syncActiveProfileToFiles() {
	const settingsDir = "Config/Settings"
	_ = os.MkdirAll(settingsDir, 0755)

	a.settingsMu.RLock()
	ls := adapter.ToLegacySettings(a.appSettings)
	p := a.appSettings.GetActiveProfile()
	var profileInteraction []byte
	if p != nil && len(p.Interaction) > 0 {
		profileInteraction = []byte(p.Interaction)
	}
	lic := adapter.ToLegacyInteraction(a.appSettings)
	a.settingsMu.RUnlock()

	// Ghi general.json
	var settings SettingsData
	if b, err := json.Marshal(ls); err == nil {
		if err := json.Unmarshal(b, &settings); err != nil {
			slog.Warn("syncSettingsFiles: unmarshal general thất bại", "err", err)
		}
	}
	if b, err := json.MarshalIndent(settings, "", "  "); err == nil {
		if err := os.WriteFile(filepath.Join(settingsDir, "general.json"), b, 0644); err != nil {
			slog.Warn("syncSettingsFiles: ghi general.json thất bại", "err", err)
		}
	}

	// Ghi interaction.json — ưu tiên profile.Interaction, fallback adapter
	if len(profileInteraction) > 0 {
		// Profile đã có interaction riêng → ghi thẳng
		if err := os.WriteFile(filepath.Join(settingsDir, "interaction.json"), profileInteraction, 0644); err != nil {
			slog.Warn("syncSettingsFiles: ghi interaction.json thất bại", "err", err)
		}
	} else {
		// Profile chưa lưu interaction → dùng adapter (backward compat)
		var interaction InteractionConfig
		if b, err := json.Marshal(lic); err == nil {
			if err := json.Unmarshal(b, &interaction); err != nil {
				slog.Warn("syncSettingsFiles: unmarshal interaction thất bại", "err", err)
			}
		}
		if b, err := json.MarshalIndent(interaction, "", "  "); err == nil {
			if err := os.WriteFile(filepath.Join(settingsDir, "interaction.json"), b, 0644); err != nil {
				slog.Warn("syncSettingsFiles: ghi interaction.json thất bại", "err", err)
			}
		}
	}
}
