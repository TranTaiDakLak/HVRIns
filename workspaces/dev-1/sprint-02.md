# Dev 1 — Sprint 02: ⭐ Cú chuyển vào internal/app (CRITICAL PATH)

> Đây là sprint rủi ro cao nhất. Làm theo `docs/rebuild/04-ke-hoach-thuc-thi.md` Pha 3+4 và
> `migration-note.md` (S01-D1-T002). **Toàn bộ cú chuyển nằm trong MỘT commit nguyên tử** (trạng
> thái nửa vời không compile). Dev 2 sprint này chỉ viết docs → không đụng file của bạn.

---

## S02-D1-T001 — Di chuyển file + đổi package
**Việc:** `git mv` tất cả file `.go` ở gốc (TRỪ `main.go`) vào `internal/app/`, giữ history:
```powershell
New-Item -ItemType Directory -Force internal/app | Out-Null
# Các file đã tách ở S01 + file gốc:
git mv app.go               internal/app/app.go
git mv app_accounts.go      internal/app/accounts.go
git mv app_settings.go      internal/app/settings.go
git mv app_profiles.go      internal/app/profiles.go
git mv app_upload.go        internal/app/upload.go
git mv app_stats.go         internal/app/stats.go
git mv app_resources.go     internal/app/resources.go
git mv app_dialogs.go       internal/app/dialogs.go
git mv app_register.go      internal/app/register.go
git mv app_reg_sxxx.go      internal/app/register_sxxx.go
git mv app_verify.go        internal/app/verify.go
git mv app_banclone.go      internal/app/banclone.go
git mv app_tempmail_reg.go  internal/app/tempmail.go
git mv app_getdatr.go       internal/app/getdatr.go
git mv debug.go             internal/app/debug.go
git mv datadir.go           internal/app/datadir.go
git mv cpu_windows.go       internal/app/cpu_windows.go        # ⚠ GIỮ hậu tố _windows
git mv portrange_windows.go internal/app/portrange_windows.go  # ⚠ GIỮ hậu tố _windows
git mv app_test.go          internal/app/app_test.go
```
Đổi dòng đầu **mọi file vừa move**: `package main` → `package app`.

**Test:** chưa compile được (main.go còn dở) — chấp nhận, làm tiếp T002/T003.
**DONE khi:** mọi file đã ở internal/app + đổi package.

> ⚠ ĐỪNG để sót file nào ở `package main`. ĐỪNG đổi tên file `_windows.go`.

---

## S02-D1-T002 — Export & wiring xuyên package
**Việc (trong package app):**
- Đổi tên (export): `startup`→`Startup`, `appDataDir`→`AppDataDir`, `expandEphemeralPortRange`→`ExpandEphemeralPortRange`. Sửa mọi nơi gọi nội bộ.
- Thêm `func (a *App) OnSecondInstance()` chứa logic show/unminimise (dùng `a.ctx` bên trong package).
- Thêm field `version string` + `func (a *App) SetVersion(v string)`; sửa `datadir.go` đọc `a.version` thay `AppVersion`. (Xem D-003.)
- Đảm bảo `GetAppVersion()` đọc `a.version`.

**Test:** `gofmt -w internal/app` ; (chưa build được nếu main.go chưa sửa).
**DONE khi:** không còn tham chiếu `AppVersion` (biến package) trong internal/app; ctx không bị expose ra ngoài.

---

## S02-D1-T003 — main.go thành bootstrap mỏng
**Việc:** sửa `main.go` (giữ `package main` @ gốc) theo mẫu `docs/rebuild/04` Pha 3.5:
- Giữ `//go:embed all:frontend/dist`, `var AppVersion = "dev"`.
- `import "HVRIns/internal/app"`; `a := app.New()` (hoặc `app.NewApp()`); `a.SetVersion(AppVersion)`.
- `os.Chdir(a.AppDataDir())` là **hành động đầu tiên** trong main() (R-7).
- `a.ExpandEphemeralPortRange()`.
- `OnStartup: a.Startup`, `OnSecondInstanceLaunch: func(...) { a.OnSecondInstance() }`,
  `OnBeforeClose` dùng `a.IsConfirmedQuit()/a.EmitQuitConfirm()`, `Bind: []interface{}{a}`.

**Test:**
```powershell
go vet ./internal/app/...
gofmt -l internal/app main.go
go test ./internal/app/...                       # app_test.go (white-box) phải PASS
npm --prefix frontend run build ; go build .     # compile package gốc xanh
```
**DONE khi:** go build . xanh, go test ./internal/app/... PASS.

> ⚠ Lúc này FE chưa chạy vì binding còn trỏ `go/main`. Sửa ở T004.

---

## S02-D1-T004 — Regenerate bindings + sửa import FE + verify
**Việc:**
```powershell
wails generate module        # vì package=app → sinh frontend/wailsjs/go/app/...
```
Sửa **~10 file** `frontend/src/bridge/wails/*.ts`: đổi `'../../../wailsjs/go/main/App'` →
`'../../../wailsjs/go/app/App'` (và `models` nếu đổi). (Đây là vùng file của bạn cho sprint này.)

```powershell
wails build
wails dev        # smoke test: cửa sổ mở, thử 1-2 thao tác FE gọi backend
```
**Verify bắt buộc (R-1, R-3):**
- `GetAppVersion()` trên bản build trả **version thật** (không "dev"). Test nhanh: build bằng
  `scripts/build.bat` (có ldflags) rồi mở app xem version.
- **Đếm lại số platform đăng ký** == baseline (S00). Nếu thiếu → có blank-import bị mất → BLOCKED.

**Test:** wails build PASS; wails dev OK; version đúng; platform count == baseline.
**DONE khi:** tất cả trên xanh → commit nguyên tử:
`refactor: move App into internal/app (package app); main.go bootstrap; regen bindings`.

---

### Sau Sprint 02
- Cập nhật progress + task-board + completed-log + decision-log (nếu có điều chỉnh).
- **Báo PM**: D1-S02 DONE → mở khoá **Dev 2 Sprint 03** (FE reorg).
