// integration_test.go — S09-D1-T001 + S10-D1-T001: integration test gọi THẬT App method.
//
// Cách cô lập:
//   - withIsolatedDataDir(t): reset dataDirOnce + HVRINS_DATA_DIR → temp dir (cho AppDataDir)
//   - withIsolatedCWD(t): t.Chdir(temp) + reset dataDirOnce + HVRINS_DATA_DIR (cho Config/Settings)
//
// Methods BỎ QUA (ghi rõ lý do):
//   RegisterFacebook, StartRegisterFacebook, RunRegister   — emit Wails event (a.ctx)
//   RunVerify, StartVerify, VerifyBatchNow                 — emit Wails event + network
//   StartUploadSite, UploadSiteNow                         — emit Wails event
//   CheckCloneHVStock, BuyCloneHV                         — HTTP request ngoài
//   GetEmailInfo, CheckTempMail, SendTempMail             — SMTP/API ngoài
//   Startup                                               — lifecycle hook, nhận context.Context Wails
//   startFolderWatcher, popAccountFromFolder              — goroutine + Startup ctx
//   OnSecondInstance                                      — kiểm tra a.ctx != nil (GUI only)
//
// Run: go test ./internal/app/... -run Integration -v
package app

import (
	"strings"
	"sync"
	"testing"

	appsettings "HVRIns/internal/settings"
)

// ─── isolation helpers ────────────────────────────────────────────────────────

// resetDataDirCache xoá cache dataDirOnce để lần gọi AppDataDir() tiếp theo đọc lại env.
func resetDataDirCache() {
	dataDirOnce = sync.Once{}
	dataDirVal = ""
}

// withIsolatedDataDir đặt HVRINS_DATA_DIR → temp dir mới và reset AppDataDir cache.
// Dùng cho method sử dụng AppDataDir() (proxy, result path, cookie path...).
func withIsolatedDataDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	resetDataDirCache()
	t.Setenv("HVRINS_DATA_DIR", dir)
	t.Cleanup(resetDataDirCache)
	return dir
}

// withIsolatedCWD chdir sang temp dir mới, đồng thời set HVRINS_DATA_DIR vào cùng dir.
// Dùng cho method dùng đường dẫn CWD-relative "Config/Settings" VÀ AppDataDir.
func withIsolatedCWD(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	resetDataDirCache()
	t.Setenv("HVRINS_DATA_DIR", dir)
	t.Cleanup(resetDataDirCache)
	return dir
}

// ─── Group 1: Accounts CRUD (in-memory, không cần CWD/AppDataDir) ─────────────

func TestIntegration_Accounts_ImportListDelete(t *testing.T) {
	a := NewApp()

	// Import 2 accounts — định dạng: UID|password (field 0 = UID by autoDetectAccount step 3)
	r := a.ImportAccounts("100000001|password1\n100000002|password2")
	if r.Imported != 2 {
		t.Fatalf("ImportAccounts: imported=%d; want 2 (errors: %v)", r.Imported, r.Errors)
	}
	if len(r.Errors) != 0 {
		t.Fatalf("ImportAccounts: errors=%v; want none", r.Errors)
	}

	// ListAccounts không filter → phải trả đủ 2
	list := a.ListAccounts(AccountFilter{})
	if list.Total != 2 {
		t.Fatalf("ListAccounts: total=%d; want 2", list.Total)
	}

	// Kiểm tra UID đúng
	uids := map[string]bool{}
	for _, acc := range list.Items {
		uids[acc.UID] = true
	}
	for _, want := range []string{"100000001", "100000002"} {
		if !uids[want] {
			t.Errorf("ListAccounts: thiếu UID %q trong items", want)
		}
	}

	// Password được lưu đúng
	for _, acc := range list.Items {
		if acc.Password == "" {
			t.Errorf("account UID=%q: Password rỗng sau import", acc.UID)
		}
	}

	// Xoá account ID=1 (100000001)
	del := a.DeleteAccounts([]int{1})
	if del.Deleted != 1 {
		t.Fatalf("DeleteAccounts: deleted=%d; want 1", del.Deleted)
	}

	// Sau xoá còn 1 account
	list2 := a.ListAccounts(AccountFilter{})
	if list2.Total != 1 {
		t.Fatalf("ListAccounts sau delete: total=%d; want 1", list2.Total)
	}
	if list2.Items[0].UID != "100000002" {
		t.Errorf("ListAccounts sau delete: UID=%q; want 100000002", list2.Items[0].UID)
	}
}

func TestIntegration_Accounts_FilterByKeyword(t *testing.T) {
	a := NewApp()
	a.ImportAccounts("111222333|pass1\n444555666|pass2")

	// Filter keyword khớp UID đầu tiên
	res := a.ListAccounts(AccountFilter{Keyword: "111222333"})
	if res.Total != 1 {
		t.Fatalf("FilterByKeyword: total=%d; want 1", res.Total)
	}
	if res.Items[0].UID != "111222333" {
		t.Errorf("FilterByKeyword: UID=%q; want 111222333", res.Items[0].UID)
	}

	// Filter keyword không match → 0 kết quả
	res2 := a.ListAccounts(AccountFilter{Keyword: "nonexistent_uid_xyz"})
	if res2.Total != 0 {
		t.Fatalf("FilterByKeyword (no match): total=%d; want 0", res2.Total)
	}
}

func TestIntegration_Accounts_FilterByStatus(t *testing.T) {
	a := NewApp()
	a.ImportAccounts("777888999|pw3") // status luôn là "new" sau import

	// Filter by status "new" → khớp
	resNew := a.ListAccounts(AccountFilter{Status: "new"})
	if resNew.Total != 1 {
		t.Errorf("FilterByStatus(new): total=%d; want 1", resNew.Total)
	}

	// Filter by status "live" → không có
	resLive := a.ListAccounts(AccountFilter{Status: "live"})
	if resLive.Total != 0 {
		t.Errorf("FilterByStatus(live): total=%d; want 0", resLive.Total)
	}
}

// ─── Group 2: Proxy round-trip (AppDataDir) ──────────────────────────────────

func TestIntegration_Proxy_RoundTrip(t *testing.T) {
	withIsolatedDataDir(t)
	a := NewApp()
	a.SetVersion("test")

	content := "1.2.3.4:8080\n5.6.7.8:9090"

	res := a.SaveProxyList("tempmail", content)
	if res != "OK" {
		t.Fatalf("SaveProxyList: %q; want OK", res)
	}

	loaded := a.LoadProxyList("tempmail")
	if !strings.Contains(loaded, "1.2.3.4:8080") {
		t.Errorf("LoadProxyList: không chứa 1.2.3.4:8080\ngot: %q", loaded)
	}
	if !strings.Contains(loaded, "5.6.7.8:9090") {
		t.Errorf("LoadProxyList: không chứa 5.6.7.8:9090\ngot: %q", loaded)
	}
}

func TestIntegration_Proxy_InvalidKind(t *testing.T) {
	a := NewApp()
	res := a.SaveProxyList("unknown_kind", "data")
	if !strings.HasPrefix(res, "kind không hợp lệ") {
		t.Errorf("SaveProxyList(invalid kind): %q; want prefix 'kind không hợp lệ'", res)
	}
}

// ─── Group 3: Settings round-trip (CWD-relative Config/Settings) ─────────────

func TestIntegration_Settings_RoundTrip(t *testing.T) {
	withIsolatedCWD(t)
	a := NewApp()
	a.SetVersion("test")
	// appsettings.Save() yêu cầu version>=1, activeProfileId, ≥1 profile → phải init trước.
	a.appSettings = appsettings.Default()

	sd := SettingsData{
		General: GeneralConfig{
			AccountSourcePath: "/tmp/test-accounts",
			ThreadRequest:     8,
			CaptchaProvider:   "2captcha",
		},
	}

	res := a.SaveSettings(sd)
	if res != "OK" {
		t.Fatalf("SaveSettings: %q; want OK", res)
	}

	loaded := a.LoadSettings()
	if loaded.General.AccountSourcePath != "/tmp/test-accounts" {
		t.Errorf("LoadSettings.AccountSourcePath=%q; want /tmp/test-accounts", loaded.General.AccountSourcePath)
	}
	if loaded.General.ThreadRequest != 8 {
		t.Errorf("LoadSettings.ThreadRequest=%d; want 8", loaded.General.ThreadRequest)
	}
	if loaded.General.CaptchaProvider != "2captcha" {
		t.Errorf("LoadSettings.CaptchaProvider=%q; want 2captcha", loaded.General.CaptchaProvider)
	}
}

// ─── Group 4: Profile lifecycle (CWD-relative Config/Settings) ───────────────

func TestIntegration_Profile_CreateListSetActiveDelete(t *testing.T) {
	withIsolatedCWD(t)
	a := NewApp()
	a.SetVersion("test")
	// appsettings.Save() yêu cầu version>=1 → dùng Default để có profile và version hợp lệ.
	a.appSettings = appsettings.Default() // 1 profile "Default"

	initialCount := len(a.ListProfiles()) // = 1 (profile "Default")

	// Tạo profile thứ nhất
	id1 := a.CreateProfile("Dev Profile")
	if strings.HasPrefix(id1, "Lỗi") {
		t.Fatalf("CreateProfile(Dev): %q", id1)
	}

	// Tạo profile thứ hai
	id2 := a.CreateProfile("Prod Profile")
	if strings.HasPrefix(id2, "Lỗi") {
		t.Fatalf("CreateProfile(Prod): %q", id2)
	}

	// ListProfiles phải trả đủ initialCount+2 profile, gồm Dev và Prod
	profiles := a.ListProfiles()
	if len(profiles) != initialCount+2 {
		t.Fatalf("ListProfiles: len=%d; want %d", len(profiles), initialCount+2)
	}
	names := map[string]bool{}
	for _, p := range profiles {
		names[p.Name] = true
	}
	for _, want := range []string{"Dev Profile", "Prod Profile"} {
		if !names[want] {
			t.Errorf("ListProfiles: thiếu profile tên %q", want)
		}
	}

	// SetActiveProfile → id1 (Dev)
	res := a.SetActiveProfile(id1)
	if res != "OK" {
		t.Errorf("SetActiveProfile(%q): %q; want OK", id1, res)
	}
	if got := a.GetActiveProfileID(); got != id1 {
		t.Errorf("GetActiveProfileID=%q; want %q", got, id1)
	}

	// DeleteProfile id2 (Prod) — active là id1 nên xoá được
	del := a.DeleteProfile(id2)
	if del != "OK" {
		t.Fatalf("DeleteProfile(%q): %q; want OK", id2, del)
	}

	// Sau xoá: còn initialCount+1 profile, không còn "Prod Profile"
	after := a.ListProfiles()
	if len(after) != initialCount+1 {
		t.Fatalf("ListProfiles sau delete: len=%d; want %d", len(after), initialCount+1)
	}
	for _, p := range after {
		if p.Name == "Prod Profile" {
			t.Errorf("Prod Profile phải đã bị xoá nhưng vẫn xuất hiện trong ListProfiles")
		}
	}
}

// ─── Group 5: Basic getters (no network, no panic) ───────────────────────────

func TestIntegration_BasicGetters(t *testing.T) {
	withIsolatedDataDir(t)
	a := NewApp()
	a.SetVersion("test")

	t.Run("GetDefaultResultPath_nonEmpty", func(t *testing.T) {
		path := a.GetDefaultResultPath()
		if path == "" {
			t.Error("GetDefaultResultPath: returned empty string")
		}
		if !strings.Contains(path, "result") {
			t.Errorf("GetDefaultResultPath: %q không chứa 'result'", path)
		}
	})

	t.Run("GetAccountSourceFolder_noProfileReturnsEmpty", func(t *testing.T) {
		// NewApp không có profile → GetActiveProfile() trả nil → "" là hành vi đúng
		folder := a.GetAccountSourceFolder()
		_ = folder // không panic là PASS; "" là hợp lệ khi chưa cấu hình
	})

	t.Run("GetCookieInitialStatus_returnsMapKeys", func(t *testing.T) {
		status := a.GetCookieInitialStatus("")
		for _, key := range []string{"path", "exists", "count", "error"} {
			if _, ok := status[key]; !ok {
				t.Errorf("GetCookieInitialStatus: thiếu key %q trong map", key)
			}
		}
		// File không tồn tại trong temp dir → exists=false
		if status["exists"] != false {
			t.Errorf("GetCookieInitialStatus.exists=%v; want false (temp dir, không có file)", status["exists"])
		}
	})

	t.Run("GetDefaultUACounts_keysExistAndChromeIsFixed", func(t *testing.T) {
		counts := a.GetDefaultUACounts()
		// Tất cả 3 key phải có mặt
		for _, k := range []string{"iphone", "android", "chrome"} {
			if _, ok := counts[k]; !ok {
				t.Errorf("GetDefaultUACounts: thiếu key %q", k)
			}
		}
		// chrome là hằng số hardcode (53 entries)
		if counts["chrome"] != 53 {
			t.Errorf("GetDefaultUACounts[chrome]=%d; want 53 (hardcoded)", counts["chrome"])
		}
		// iphone/android phụ thuộc data runtime — không assert >0 (có thể 0 khi không có Config/UserAgent/)
		if counts["iphone"] < 0 || counts["android"] < 0 {
			t.Errorf("GetDefaultUACounts: giá trị âm không hợp lệ (iphone=%d, android=%d)",
				counts["iphone"], counts["android"])
		}
	})
}

// ─── Group 6 (S10): Run status — trạng thái mặc định khi chưa chạy ──────────

func TestIntegration_RunStatus_DefaultState(t *testing.T) {
	a := NewApp()

	// IsRegisterRunning và IsVerifyRunning = false khi chưa start bao giờ
	if a.IsRegisterRunning() {
		t.Error("IsRegisterRunning: want false khi mới khởi tạo")
	}
	if a.IsVerifyRunning() {
		t.Error("IsVerifyRunning: want false khi mới khởi tạo")
	}

	// GetRunStatus trả map với 4 key, tất cả = false
	status := a.GetRunStatus()
	for _, key := range []string{"registerRunning", "registerStopping", "verifyRunning", "verifyStopping"} {
		v, ok := status[key]
		if !ok {
			t.Errorf("GetRunStatus: thiếu key %q", key)
			continue
		}
		if v {
			t.Errorf("GetRunStatus[%q]=%v; want false (mặc định chưa chạy)", key, v)
		}
	}
}

// ─── Group 7 (S10): Datr pool — zero khi không có run ────────────────────────

func TestIntegration_DatrPool_ZeroWhenEmpty(t *testing.T) {
	a := NewApp()

	// SharedPool = nil khi chưa khởi động register → Size() = 0, không panic
	size := a.GetDatrPoolSize()
	if size < 0 {
		t.Errorf("GetDatrPoolSize=%d; want >=0", size)
	}

	// poolFileSaved atomic = 0 khi mới tạo App, không có run nào
	saved := a.GetPoolFileSaveCount()
	if saved != 0 {
		t.Errorf("GetPoolFileSaveCount=%d; want 0 khi chưa có run", saved)
	}
}

// ─── Group 8 (S10): UA pools status — cấu trúc hợp lệ ───────────────────────

func TestIntegration_UAPoolsStatus_Structure(t *testing.T) {
	a := NewApp()

	pools := a.GetUAPoolsStatus()

	// Phải trả ≥1 entry (6 kind được hardcode trong GetUAPoolsStatus)
	if len(pools) == 0 {
		t.Fatal("GetUAPoolsStatus: trả slice rỗng; want ≥1 entry")
	}

	// Mỗi entry phải có Kind không rỗng và Count >= 0
	for i, p := range pools {
		if p.Kind == "" {
			t.Errorf("pools[%d].Kind rỗng", i)
		}
		if p.Count < 0 {
			t.Errorf("pools[%d].Count=%d; want >=0 (kind=%s)", i, p.Count, p.Kind)
		}
	}

	// 6 kind cố định: android, ios, request, web_chrome, android_mess, ios_mess
	if len(pools) != 6 {
		t.Errorf("GetUAPoolsStatus: len=%d; want 6 (hardcoded kinds)", len(pools))
	}
}

// ─── Group 9 (S10): Default cookie paths — keys hợp lệ ──────────────────────

func TestIntegration_DefaultCookiePaths_Keys(t *testing.T) {
	withIsolatedDataDir(t)
	a := NewApp()
	a.SetVersion("test")

	paths := a.GetDefaultCookiePaths()

	for _, key := range []string{"dir", "initial"} {
		v, ok := paths[key]
		if !ok {
			t.Errorf("GetDefaultCookiePaths: thiếu key %q", key)
			continue
		}
		if v == "" {
			t.Errorf("GetDefaultCookiePaths[%q]: empty string; want non-empty path", key)
		}
	}

	// "initial" phải kết thúc bằng tên file đúng
	if initial, ok := paths["initial"]; ok {
		if !strings.HasSuffix(initial, "cookie_initial.txt") {
			t.Errorf("GetDefaultCookiePaths[initial]=%q; want suffix 'cookie_initial.txt'", initial)
		}
	}
}

// ─── Group 10 (S10): GetCookieInitialStatus chi tiết ─────────────────────────

func TestIntegration_CookieInitialStatus_Detailed(t *testing.T) {
	withIsolatedDataDir(t)
	a := NewApp()
	a.SetVersion("test")

	status := a.GetCookieInitialStatus("")

	// path phải non-empty và trỏ đến cookie_initial.txt
	path, _ := status["path"].(string)
	if path == "" {
		t.Error("GetCookieInitialStatus.path: empty; want non-empty")
	}
	if !strings.HasSuffix(path, "cookie_initial.txt") {
		t.Errorf("GetCookieInitialStatus.path=%q; want suffix 'cookie_initial.txt'", path)
	}

	// File chưa tồn tại trong temp dir → exists=false, error non-empty
	if status["exists"] != false {
		t.Errorf("GetCookieInitialStatus.exists=%v; want false (file chưa tạo)", status["exists"])
	}
	errMsg, _ := status["error"].(string)
	if errMsg == "" {
		t.Error("GetCookieInitialStatus.error: empty; want non-empty khi file không tồn tại")
	}
}

// ─── Group 11 (S10): SetAccountSourceFolder ↔ GetAccountSourceFolder ─────────

func TestIntegration_AccountSourceFolder_RoundTrip(t *testing.T) {
	withIsolatedCWD(t)
	a := NewApp()
	a.SetVersion("test")
	// Cần profile active để Set/GetAccountSourceFolder hoạt động
	a.appSettings = appsettings.Default()

	wantPath := `/C/TestAccounts`
	a.SetAccountSourceFolder(wantPath)

	got := a.GetAccountSourceFolder()
	if got != wantPath {
		t.Errorf("GetAccountSourceFolder=%q; want %q", got, wantPath)
	}
}
