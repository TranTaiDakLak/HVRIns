package store_test

import (
	"os"
	"testing"

	"HVRIns/internal/settings/model"
	"HVRIns/internal/settings/store"
)

// TestLoadFrom_NonExistentDir trả về DefaultAppSettings nếu file chưa có
func TestLoadFrom_NonExistentDir(t *testing.T) {
	app, err := store.LoadFrom("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Errorf("expected no error for missing file, got: %v", err)
	}
	if app.Version != model.CurrentVersion {
		t.Errorf("version: got %d, want %d", app.Version, model.CurrentVersion)
	}
}

// TestSaveAndLoad_Roundtrip kiểm tra save→load roundtrip
func TestSaveAndLoad_Roundtrip(t *testing.T) {
	dir := t.TempDir()

	orig := model.DefaultAppSettings()
	orig.ActiveProfileID = "default"

	p := orig.GetActiveProfile()
	p.Runtime.ThreadRequest = 42
	p.Runtime.DelayRequest = 750
	p.Account.Source = "api"
	p.Account.CloneHV.Username = "testuser"
	p.Mail.Provider = "@gmail.com"
	orig.UpsertProfile(*p)

	if err := store.SaveTo(dir, orig); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	loaded, err := store.LoadFrom(dir)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if loaded.Version != orig.Version {
		t.Errorf("version: got %d, want %d", loaded.Version, orig.Version)
	}
	if loaded.ActiveProfileID != orig.ActiveProfileID {
		t.Errorf("activeProfileId: got %s, want %s", loaded.ActiveProfileID, orig.ActiveProfileID)
	}

	lp := loaded.GetActiveProfile()
	if lp == nil {
		t.Fatal("loaded active profile is nil")
	}
	if lp.Runtime.ThreadRequest != 42 {
		t.Errorf("threadRequest: got %d, want 42", lp.Runtime.ThreadRequest)
	}
	if lp.Runtime.DelayRequest != 750 {
		t.Errorf("delayRequest: got %d, want 750", lp.Runtime.DelayRequest)
	}
	if lp.Account.CloneHV.Username != "testuser" {
		t.Errorf("cloneHv.username: got %s, want testuser", lp.Account.CloneHV.Username)
	}
	if lp.Mail.Provider != "@gmail.com" {
		t.Errorf("mail.provider: got %s, want @gmail.com", lp.Mail.Provider)
	}
}

// TestSaveTo_InvalidSettings validation error phải block save
func TestSaveTo_InvalidSettings(t *testing.T) {
	dir := t.TempDir()
	invalid := model.AppSettings{
		Version:         0, // invalid
		ActiveProfileID: "",
	}
	if err := store.SaveTo(dir, invalid); err == nil {
		t.Error("SaveTo with invalid settings should return error")
	}
}

// TestExists trả về false trước khi save, true sau khi save
func TestExists(t *testing.T) {
	dir := t.TempDir()

	if store.Exists(dir) {
		t.Error("Exists should be false before any save")
	}

	app := model.DefaultAppSettings()
	if err := store.SaveTo(dir, app); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	if !store.Exists(dir) {
		t.Error("Exists should be true after save")
	}
}

// TestLoadFrom_AppliesDefaults kiểm tra applyDefaults fill missing fields
func TestLoadFrom_AppliesDefaults(t *testing.T) {
	dir := t.TempDir()

	// Viết JSON tối giản (thiếu nhiều field)
	minimal := []byte(`{"version":1,"activeProfileId":"default","profiles":[{"id":"default","name":"Default"}]}`)
	if err := os.WriteFile(dir+"/app_settings.json", minimal, 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	app, err := store.LoadFrom(dir)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	p := app.GetActiveProfile()
	if p == nil {
		t.Fatal("profile is nil after defaults applied")
	}
	if p.Captcha.Keys == nil {
		t.Error("captcha.keys should be filled by defaults")
	}
	if p.Mail.Providers == nil {
		t.Error("mail.providers should be filled by defaults")
	}
}

// TestSaveAndLoad_MultipleProfiles kiểm tra multiple profiles
func TestSaveAndLoad_MultipleProfiles(t *testing.T) {
	dir := t.TempDir()

	app := model.DefaultAppSettings()
	p2 := model.DefaultProfile("work", "Work Profile")
	p2.Runtime.ThreadRequest = 50
	app.UpsertProfile(p2)

	if len(app.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(app.Profiles))
	}

	if err := store.SaveTo(dir, app); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	loaded, err := store.LoadFrom(dir)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if len(loaded.Profiles) != 2 {
		t.Errorf("expected 2 profiles after load, got %d", len(loaded.Profiles))
	}
}
