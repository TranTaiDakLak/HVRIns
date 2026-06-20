# Migration Design Note — Sprint 02 (S01-D1-T002)

> Kế hoạch kỹ thuật cho Sprint 02: `package main` → `package app`.
> Đọc trước `sprint-02.md` và `CLAUDE.md` (Deviation D-001, D-002).

---

## 1. Hàm/method cần export (lowercase → Uppercase)

| File hiện tại | Symbol hiện tại | Symbol sau S02 | Ghi chú |
|---------------|-----------------|----------------|---------|
| `datadir.go` | `func appDataDir() string` | `func AppDataDir() string` | free function, giữ _logic_ |
| `portrange_windows.go` | `func expandEphemeralPortRange()` | `func ExpandEphemeralPortRange()` | giữ `_windows.go` suffix |
| `app.go` | `func (a *App) startup(ctx)` | `func (a *App) Startup(ctx)` | OnStartup hook của Wails |

> **Không di chuyển `AppVersion`** — biến này ở lại `main.go` (package main).
> Sau Sprint 02, `package app` truy cập nó thông qua `buildVersion` (xem mục 3).

---

## 2. Method mới cần thêm: `OnSecondInstance()`

**Lý do**: `main.go` hiện truy cập `app.ctx` trực tiếp trong closure `OnSecondInstanceLaunch`.
Sau khi App chuyển sang `package app`, `app.ctx` là unexported field từ phía `main`. Phải bọc logic vào method.

**Hiện tại (main.go:72-78)**:
```go
OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
    wailsRuntime.WindowUnminimise(app.ctx)
    wailsRuntime.Show(app.ctx)
    wailsRuntime.WindowSetAlwaysOnTop(app.ctx, true)
    time.Sleep(80 * time.Millisecond)
    wailsRuntime.WindowSetAlwaysOnTop(app.ctx, false)
},
```

**Sau Sprint 02** — thêm method vào `internal/app/app.go`:
```go
func (a *App) OnSecondInstance() {
    wailsRuntime.WindowUnminimise(a.ctx)
    wailsRuntime.Show(a.ctx)
    wailsRuntime.WindowSetAlwaysOnTop(a.ctx, true)
    time.Sleep(80 * time.Millisecond)
    wailsRuntime.WindowSetAlwaysOnTop(a.ctx, false)
}
```

**`main.go` sửa thành**:
```go
OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
    appInstance.OnSecondInstance()
},
```

---

## 3. AppVersion — cách thread qua SetVersion

**Ràng buộc**: `AppVersion` (ldflags `-X main.AppVersion=...`) phải ở `package main`.
`datadir.go` hiện dùng `AppVersion` để phân biệt dev vs production.
`GetAppVersion()` trả về `AppVersion`.

**Giải pháp** — 2-level:

### 3a. Package-level `buildVersion` trong `package app`

Thêm vào `internal/app/datadir.go`:
```go
// buildVersion là bản copy của main.AppVersion, set bởi main() trước khi NewApp().
// Chỉ dùng trong AppDataDir() — cần biết dev vs prod ngay từ đầu, trước khi App struct tồn tại.
var buildVersion = "dev"

func AppDataDir() string {
    dataDirOnce.Do(func() {
        if d := os.Getenv("HVRINS_DATA_DIR"); d != "" { ... }
        if buildVersion == "dev" { ... }   // ← dùng buildVersion thay vì AppVersion
        ...
    })
    return dataDirVal
}
```

### 3b. Method `SetVersion` trên App struct

Thêm field `version string` vào `App` struct.
Thêm method:
```go
// SetVersion ghi AppVersion từ package main xuống package app.
// Gọi ngay sau NewApp(), trước wails.Run().
func (a *App) SetVersion(v string) {
    a.version = v
    buildVersion = v  // cũng update package-level var để AppDataDir dùng được
}
```

Sửa `GetAppVersion()`:
```go
func (a *App) GetAppVersion() string {
    return a.version  // không còn phụ thuộc global AppVersion
}
```

### 3c. Thứ tự gọi trong `main.go` sau Sprint 02

```go
import igapp "HVRIns/internal/app"

func main() {
    flag.Parse()

    igapp.SetBuildVersion(AppVersion)           // ← set trước AppDataDir
    if err := os.Chdir(igapp.AppDataDir()); err != nil { ... }
    igapp.ExpandEphemeralPortRange()

    appInstance := igapp.NewApp()
    appInstance.SetVersion(AppVersion)          // ← set cho GetAppVersion()
    ...
    wails.Run(&options.App{
        OnStartup: appInstance.Startup,
        OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
            appInstance.OnSecondInstance()
        },
        OnBeforeClose: func(_ context.Context) bool {
            if appInstance.IsConfirmedQuit() { return false }
            appInstance.EmitQuitConfirm()
            return true
        },
        Bind: []interface{}{ appInstance },
        ...
    })
}
```

> **Lưu ý**: `SetBuildVersion` là free function (không phải method) vì cần gọi TRƯỚC `NewApp()`.

---

## 4. Danh sách toàn bộ calls từ main.go vào App (hiện tại)

| main.go call | Loại | Sau Sprint 02 |
|--------------|------|---------------|
| `appDataDir()` | free func | `igapp.AppDataDir()` |
| `expandEphemeralPortRange()` | free func | `igapp.ExpandEphemeralPortRange()` |
| `NewApp()` | constructor | `igapp.NewApp()` |
| `app.startup` | method ref | `appInstance.Startup` |
| `app.ctx` (trong closure) | field access | bọc thành `appInstance.OnSecondInstance()` |
| `app.IsConfirmedQuit()` | method | `appInstance.IsConfirmedQuit()` (đã public) |
| `app.EmitQuitConfirm()` | method | `appInstance.EmitQuitConfirm()` (đã public) |

---

## 5. Files bị ảnh hưởng Sprint 02

### Chuyển vào `internal/app/` (git mv)

```
app.go               → internal/app/app.go
app_accounts.go      → internal/app/app_accounts.go
app_dialogs.go       → internal/app/app_dialogs.go
app_resources.go     → internal/app/app_resources.go
app_settings.go      → internal/app/app_settings.go
app_stats.go         → internal/app/app_stats.go
app_upload.go        → internal/app/app_upload.go
app_profiles.go      → internal/app/app_profiles.go
app_getdatr.go       → internal/app/app_getdatr.go
app_reg_sxxx.go      → internal/app/app_reg_sxxx.go   ← giữ 207 blank-import
app_register.go      → internal/app/app_register.go
app_verify.go        → internal/app/app_verify.go
app_banclone.go      → internal/app/app_banclone.go
datadir.go           → internal/app/datadir.go
portrange_windows.go → internal/app/portrange_windows.go  ← giữ _windows suffix
cpu_windows.go       → internal/app/cpu_windows.go         ← giữ _windows suffix
```

> Tổng: **16 files** (thực tế nhiều hơn 12 như sprint doc nói).
> Một commit nguyên tử (D-007).

### Sửa package header

Tất cả file trên: `package main` → `package app`

### Sửa `main.go` (stays at root)

- Thêm import `igapp "HVRIns/internal/app"`
- Xoá import các package nội bộ không còn dùng trực tiếp
- Áp dụng thứ tự gọi ở mục 3c

### Verify sau Sprint 02

```powershell
wails generate module      # sinh lại wailsjs/go/main/ → kiểm tra path có đổi không
# nếu path đổi (go/main → go/app), sửa ~10 import trong frontend/src/bridge/wails/*.ts
go build .                 # xanh
wails build                # xanh
.\HVRIns.exe               # GetAppVersion() != "dev" khi build production
```

---

*Tạo: 2026-06-20 — S01-D1-T002*
