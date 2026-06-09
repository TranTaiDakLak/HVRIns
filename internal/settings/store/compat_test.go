// compat_test.go — Backward compatibility: load app_settings.json từ phiên bản cũ hơn
// Đảm bảo Phase 3+ fields được fill bởi applyDefaults khi load file Phase 1/2 cũ
package store_test

import (
	"os"
	"testing"

	"HVRIns/internal/settings/model"
	"HVRIns/internal/settings/store"
)

// TestLoadFrom_Phase1JSONMissingPhase3Fields Phase 1 JSON thiếu các field Phase 3
// applyDefaults phải fill captcha.keys, mail.providers, proxy.providers
func TestLoadFrom_Phase1JSONMissingPhase3Fields(t *testing.T) {
	dir := t.TempDir()

	// app_settings.json format Phase 1: thiếu captcha.keys map, thiếu mail.providers
	phase1JSON := `{
  "version": 1,
  "activeProfileId": "default",
  "global": { "loginPlatform": "facebook", "loginMethod": 0 },
  "profiles": [
    {
      "id": "default",
      "name": "Default",
      "runtime": { "threadRequest": 20, "delayRequest": 500, "threadCheckInfo": 10 },
      "account": { "source": "folder" },
      "proxy": { "provider": "none" },
      "verify": { "enabled": false },
      "register": { "enabled": false },
      "mail": { "provider": "@i2b.vn" },
      "captcha": { "provider": "2captcha" },
      "output": {},
      "device": {}
    }
  ]
}`
	if err := os.WriteFile(dir+"/app_settings.json", []byte(phase1JSON), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	app, err := store.LoadFrom(dir)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	p := app.GetActiveProfile()
	if p == nil {
		t.Fatal("profile nil")
	}

	// captcha.keys phải được fill bởi applyDefaults
	if p.Captcha.Keys == nil {
		t.Error("captcha.keys should be filled by applyDefaults for Phase 1 JSON")
	}
	for _, k := range []string{"2captcha", "capsolver", "ezcaptcha", "omocaptcha"} {
		if _, ok := p.Captcha.Keys[k]; !ok {
			t.Errorf("captcha.keys[%s] missing after applyDefaults", k)
		}
	}

	// mail.providers phải được fill
	if p.Mail.Providers == nil {
		t.Error("mail.providers should be filled by applyDefaults")
	}
	for _, prov := range []string{"zeusx", "dvfb", "store1s", "mail30s"} {
		if _, ok := p.Mail.Providers[prov]; !ok {
			t.Errorf("mail.providers[%s] missing after applyDefaults", prov)
		}
	}

	// proxy.providers không nil
	if p.Proxy.Providers == nil {
		t.Error("proxy.providers should not be nil after load")
	}
}

// TestLoadFrom_ExtraUnknownFields extra fields trong JSON không gây error
func TestLoadFrom_ExtraUnknownFields(t *testing.T) {
	dir := t.TempDir()

	withExtraFields := `{
  "version": 1,
  "activeProfileId": "default",
  "unknownFutureField": "ignored",
  "global": { "loginPlatform": "facebook", "newGlobalFieldPhase9": true },
  "profiles": [
    {
      "id": "default",
      "name": "Default",
      "runtime": { "threadRequest": 15, "futureRuntimeField": 999 },
      "account": { "source": "folder" },
      "proxy": { "provider": "none" },
      "verify": { "enabled": false },
      "register": { "enabled": false },
      "mail": { "provider": "@i2b.vn" },
      "captcha": { "provider": "2captcha" },
      "output": {},
      "device": {}
    }
  ]
}`
	if err := os.WriteFile(dir+"/app_settings.json", []byte(withExtraFields), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	app, err := store.LoadFrom(dir)
	if err != nil {
		t.Fatalf("LoadFrom should not error on extra JSON fields, got: %v", err)
	}

	p := app.GetActiveProfile()
	if p == nil {
		t.Fatal("profile nil")
	}
	if p.Runtime.ThreadRequest != 15 {
		t.Errorf("threadRequest: got %d, want 15", p.Runtime.ThreadRequest)
	}
}

// TestLoadFrom_PartialProfileFields JSON thiếu nhiều profile fields → defaults fill in
func TestLoadFrom_PartialProfileFields(t *testing.T) {
	dir := t.TempDir()

	bareProfile := `{
  "version": 1,
  "activeProfileId": "minimal",
  "profiles": [ { "id": "minimal", "name": "Minimal" } ]
}`
	if err := os.WriteFile(dir+"/app_settings.json", []byte(bareProfile), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	app, err := store.LoadFrom(dir)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	p := app.GetActiveProfile()
	if p == nil {
		t.Fatal("profile nil for bare profile")
	}
	if p.Captcha.Keys == nil {
		t.Error("captcha.keys nil for minimal profile — applyDefaults should fill it")
	}
}

// TestLoadFrom_WrongTypeCoercion JSON với wrong type không crash (json tolerate gracefully)
func TestLoadFrom_WrongTypeCoercion(t *testing.T) {
	dir := t.TempDir()

	// threadRequest là string thay vì number — Go's json.Unmarshal trả lỗi, store tolerate
	wrongType := `{
  "version": 1,
  "activeProfileId": "default",
  "profiles": [
    {
      "id": "default",
      "name": "Default",
      "runtime": { "threadRequest": "not-a-number" }
    }
  ]
}`
	if err := os.WriteFile(dir+"/app_settings.json", []byte(wrongType), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Không được panic — lỗi parse có thể xảy ra nhưng không crash
	_, _ = store.LoadFrom(dir)
}

// TestLoadFrom_EmptyProfilesArray profiles=[] → applyDefaults tạo profile mặc định
func TestLoadFrom_EmptyProfilesArray(t *testing.T) {
	dir := t.TempDir()

	noProfiles := `{ "version": 1, "activeProfileId": "default", "profiles": [] }`
	if err := os.WriteFile(dir+"/app_settings.json", []byte(noProfiles), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	app, err := store.LoadFrom(dir)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if len(app.Profiles) == 0 {
		t.Error("applyDefaults should create default profile when profiles array is empty")
	}
}

// TestMigrateIfNeeded_InvalidGeneralJSON general.json không hợp lệ → không panic
func TestMigrateIfNeeded_InvalidGeneralJSON(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(dir+"/general.json", []byte(`{ "general": { "threadReq`), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Không panic — quan trọng nhất
	app, _, err := store.MigrateIfNeeded(dir)
	if err != nil {
		// Error is acceptable
		return
	}
	if app.Version < 1 {
		t.Error("version should be >= 1 even with invalid general.json")
	}
}

// TestMigrateIfNeeded_EmptyJSONObjects general.json và interaction.json đều là {} → không crash
func TestMigrateIfNeeded_EmptyJSONObjects(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(dir+"/general.json", []byte(`{}`), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := os.WriteFile(dir+"/interaction.json", []byte(`{}`), 0644); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	app, migrated, err := store.MigrateIfNeeded(dir)
	if err != nil {
		t.Fatalf("unexpected error with empty JSON files: %v", err)
	}
	if !migrated {
		t.Error("expected migrated=true when general.json exists (even empty)")
	}
	if app.Version < 1 {
		t.Error("version should be >= 1")
	}
}

// TestSaveLoad_CaptchaKeysPreserved captcha keys giữ nguyên sau save→load roundtrip
func TestSaveLoad_CaptchaKeysPreserved(t *testing.T) {
	dir := t.TempDir()

	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Captcha.Keys["2captcha"] = "my_real_key"
	p.Captcha.Keys["capsolver"] = "cap_key"
	app.UpsertProfile(*p)

	if err := store.SaveTo(dir, app); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}
	loaded, err := store.LoadFrom(dir)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	lp := loaded.GetActiveProfile()
	if lp == nil {
		t.Fatal("profile nil after load")
	}
	if lp.Captcha.Keys["2captcha"] != "my_real_key" {
		t.Errorf("2captcha key lost: got %q", lp.Captcha.Keys["2captcha"])
	}
	if lp.Captcha.Keys["capsolver"] != "cap_key" {
		t.Errorf("capsolver key lost: got %q", lp.Captcha.Keys["capsolver"])
	}
}

// TestSaveLoad_MailProviderConfigPreserved mail provider API keys giữ nguyên sau roundtrip
func TestSaveLoad_MailProviderConfigPreserved(t *testing.T) {
	dir := t.TempDir()

	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Mail.Provider = "store1s"
	p.Mail.Providers["store1s"] = model.MailProviderCfg{
		APIKey:    "store_key_123",
		ProductID: "40559",
	}
	p.Mail.Providers["mail30s"] = model.MailProviderCfg{
		APIKey:      "m30_key_456",
		ProductSlug: "hotmail-oauth2",
	}
	app.UpsertProfile(*p)

	if err := store.SaveTo(dir, app); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}
	loaded, err := store.LoadFrom(dir)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	lp := loaded.GetActiveProfile()
	if lp.Mail.Provider != "store1s" {
		t.Errorf("mail.provider: got %s", lp.Mail.Provider)
	}
	if lp.Mail.Providers["store1s"].APIKey != "store_key_123" {
		t.Errorf("store1s.apiKey: got %s", lp.Mail.Providers["store1s"].APIKey)
	}
	if lp.Mail.Providers["mail30s"].ProductSlug != "hotmail-oauth2" {
		t.Errorf("mail30s.productSlug: got %s", lp.Mail.Providers["mail30s"].ProductSlug)
	}
}

// TestSaveLoad_RegisterSettingsPreserved register config giữ nguyên sau roundtrip
func TestSaveLoad_RegisterSettingsPreserved(t *testing.T) {
	dir := t.TempDir()

	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Register.Enabled = true
	p.Register.Type = "tut"
	p.Register.CookieList = "cookie1\ncookie2"
	p.Register.OutputPath = "/output/register"
	app.UpsertProfile(*p)

	if err := store.SaveTo(dir, app); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}
	loaded, err := store.LoadFrom(dir)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	lp := loaded.GetActiveProfile()
	if !lp.Register.Enabled {
		t.Error("register.enabled: got false, want true")
	}
	if lp.Register.Type != "tut" {
		t.Errorf("register.type: got %s, want tut", lp.Register.Type)
	}
	if lp.Register.OutputPath != "/output/register" {
		t.Errorf("register.outputPath: got %s", lp.Register.OutputPath)
	}
}

// TestSaveLoad_ProxyProviderMapPreserved proxy providers map giữ nguyên sau roundtrip
func TestSaveLoad_ProxyProviderMapPreserved(t *testing.T) {
	dir := t.TempDir()

	app := model.DefaultAppSettings()
	p := app.GetActiveProfile()
	p.Proxy.Provider = "tinsoft"
	p.Proxy.ProxyType = "http"
	p.Proxy.Providers["tinsoft"] = model.ProxyProviderCfg{Keys: "ts_key_abc", ThreadPerIP: 3}
	p.Proxy.Providers["fpt"] = model.ProxyProviderCfg{Keys: "fpt_key_xyz"}
	app.UpsertProfile(*p)

	if err := store.SaveTo(dir, app); err != nil {
		t.Fatalf("SaveTo: %v", err)
	}
	loaded, err := store.LoadFrom(dir)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	lp := loaded.GetActiveProfile()
	if lp.Proxy.Provider != "tinsoft" {
		t.Errorf("proxy.provider: got %s", lp.Proxy.Provider)
	}
	if lp.Proxy.Providers["tinsoft"].Keys != "ts_key_abc" {
		t.Errorf("tinsoft.keys: got %s", lp.Proxy.Providers["tinsoft"].Keys)
	}
	if lp.Proxy.Providers["fpt"].Keys != "fpt_key_xyz" {
		t.Errorf("fpt.keys: got %s", lp.Proxy.Providers["fpt"].Keys)
	}
}
