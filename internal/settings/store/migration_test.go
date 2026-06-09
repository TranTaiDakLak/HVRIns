package store_test

import (
	"encoding/json"
	"os"
	"testing"

	"HVRIns/internal/settings/model"
	"HVRIns/internal/settings/store"
)

// writeLegacyFiles ghi general.json + interaction.json giống format cũ
func writeLegacyFiles(t *testing.T, dir string) {
	t.Helper()

	general := map[string]interface{}{
		"general": map[string]interface{}{
			"threadRequest":    15,
			"delayRequest":     600,
			"threadCheckInfo":  8,
			"loginPlatform":    "facebook",
			"loginMethod":      3,
			"accountSource":    "folder",
			"accountSourcePath": "/legacy/accounts",
			"captchaProvider":  "capsolver",
			"captchaKeys":      map[string]string{"capsolver": "cs_key", "2captcha": ""},
			"ipProvider":       "proxy",
			"checkIpBeforeRun": true,
			"delayChangeIp":    5,
		},
		"ip": map[string]interface{}{
			"proxyList": "proxy1:8080:u:p",
			"proxyType": "http",
		},
	}
	interaction := map[string]interface{}{
		"verifyEnabled":       true,
		"mailProvider":        "@i2b.vn",
		"timeDelayCheck":      7,
		"timeDelaySendCode":   6,
		"outputPath":          "/legacy/output",
		"uaIphoneList":        "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2)",
		"cloneHvEnabled":      false,
		"cloneHvUsername":     "legacy_user",
		"cloneHvPassword":     "legacy_pass",
		"cloneHvProductId":    "99",
		"store1sApiKey":       "s1_key",
		"store1sProductId":    "40559",
		"mail30sApiKey":       "m30_key",
		"mail30sProductSlug":  "hotmail-oauth2",
		"createEnabled":       false,
		"createType":          "normal",
	}

	writeJSON := func(name string, v interface{}) {
		data, _ := json.MarshalIndent(v, "", "  ")
		if err := os.WriteFile(dir+"/"+name+".json", data, 0644); err != nil {
			t.Fatalf("write %s failed: %v", name, err)
		}
	}
	writeJSON("general", general)
	writeJSON("interaction", interaction)
}

// TestMigrateIfNeeded_NoFiles empty dir → default settings, not migrated
func TestMigrateIfNeeded_NoFiles(t *testing.T) {
	dir := t.TempDir()
	app, migrated, err := store.MigrateIfNeeded(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if migrated {
		t.Error("expected migrated=false for empty dir")
	}
	if app.Version != model.CurrentVersion {
		t.Errorf("version: got %d, want %d", app.Version, model.CurrentVersion)
	}
}

// TestMigrateIfNeeded_LegacyFiles migrate từ file cũ
func TestMigrateIfNeeded_LegacyFiles(t *testing.T) {
	dir := t.TempDir()
	writeLegacyFiles(t, dir)

	app, migrated, err := store.MigrateIfNeeded(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !migrated {
		t.Error("expected migrated=true when legacy files exist")
	}

	// Kiểm tra app_settings.json đã được tạo
	if !store.Exists(dir) {
		t.Error("app_settings.json should be created after migration")
	}

	// Kiểm tra data đã migrate đúng
	p := app.GetActiveProfile()
	if p == nil {
		t.Fatal("active profile is nil after migration")
	}
	if p.Runtime.ThreadRequest != 15 {
		t.Errorf("threadRequest: got %d, want 15", p.Runtime.ThreadRequest)
	}
	if p.Runtime.DelayRequest != 600 {
		t.Errorf("delayRequest: got %d, want 600", p.Runtime.DelayRequest)
	}
	if p.Account.Source != "folder" {
		t.Errorf("account.source: got %s, want folder", p.Account.Source)
	}
	if p.Account.FolderPath != "/legacy/accounts" {
		t.Errorf("account.folderPath: got %s, want /legacy/accounts", p.Account.FolderPath)
	}
	if p.Verify.TimeDelayCheck != 7 {
		t.Errorf("verify.timeDelayCheck: got %d, want 7", p.Verify.TimeDelayCheck)
	}
	if p.Device.UAList != "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2)" {
		t.Errorf("device.uaList: got %s", p.Device.UAList)
	}
	if p.Mail.Providers["store1s"].APIKey != "s1_key" {
		t.Errorf("mail.store1s.apiKey: got %s, want s1_key", p.Mail.Providers["store1s"].APIKey)
	}
}

// TestMigrateIfNeeded_AlreadyMigrated nếu đã có app_settings.json → load nó, migrated=false
func TestMigrateIfNeeded_AlreadyMigrated(t *testing.T) {
	dir := t.TempDir()
	writeLegacyFiles(t, dir)

	// First migration
	_, migrated1, err := store.MigrateIfNeeded(dir)
	if err != nil || !migrated1 {
		t.Fatalf("first migration failed: migrated=%v, err=%v", migrated1, err)
	}

	// Second call — app_settings.json đã tồn tại
	app2, migrated2, err := store.MigrateIfNeeded(dir)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if migrated2 {
		t.Error("expected migrated=false on second call (file already exists)")
	}

	// Data vẫn đúng
	p := app2.GetActiveProfile()
	if p == nil {
		t.Fatal("profile nil on second load")
	}
	if p.Runtime.ThreadRequest != 15 {
		t.Errorf("threadRequest after second load: got %d, want 15", p.Runtime.ThreadRequest)
	}
}

// TestMigrateIfNeeded_OnlyGeneralFile chỉ có general.json, không có interaction.json
func TestMigrateIfNeeded_OnlyGeneralFile(t *testing.T) {
	dir := t.TempDir()

	general := map[string]interface{}{
		"general": map[string]interface{}{
			"threadRequest": 5,
			"loginPlatform": "facebook",
			"accountSource": "folder",
		},
		"ip": map[string]interface{}{},
	}
	data, _ := json.MarshalIndent(general, "", "  ")
	_ = os.WriteFile(dir+"/general.json", data, 0644)

	app, migrated, err := store.MigrateIfNeeded(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !migrated {
		t.Error("expected migrated=true when general.json exists")
	}

	p := app.GetActiveProfile()
	if p == nil {
		t.Fatal("profile nil")
	}
	if p.Runtime.ThreadRequest != 5 {
		t.Errorf("threadRequest: got %d, want 5", p.Runtime.ThreadRequest)
	}
}
