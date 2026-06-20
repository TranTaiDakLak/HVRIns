# 04 — Kế hoạch thực thi (theo thứ tự an toàn)

> **Nguyên tắc vàng:** làm từng bước → chạy lệnh kiểm tra → **xanh mới đi tiếp** → commit lại.
> Như vậy nếu hỏng, bạn biết chính xác bước nào gây ra (và có thể `git bisect`).
>
> **Tất cả lệnh kiểm tra phải chạy trên WINDOWS** (app Windows-only). Shell ví dụ dưới đây
> dùng PowerShell.

## Sơ đồ các pha

```
Pha 0  Baseline & dọn git        ──► phải XANH trước khi bắt đầu
Pha 1  🔴 Secrets (khẩn cấp)      ──► độc lập, làm sớm nhất có thể
Pha 2  Dọn rác không-ảnh-hưởng-Go ──► docs, scripts, cmd scratch, python lạc
Pha 3  ⭐ Cú chuyển internal/app   ──► MỘT commit nguyên tử (rủi ro cao nhất)
Pha 4  Regenerate Wails bindings  ──► + sửa import FE
Pha 5  config/ + tooling          ──► template, launch.json
Pha 6  (sau) Đổi tên FE           ──► theo từng feature
Pha 7  (sau, tuỳ chọn) internal/  ──► domain/usecase/adapter
```

---

## PHA 0 — Baseline & dọn git (BẮT BUỘC làm đầu tiên)

Mục tiêu: có một mốc "xanh" đã biết để so sánh.

```powershell
# 1. Xử lý go.mod/go.sum đang "Modified" (git status cho thấy M go.mod, M go.sum)
git status
git diff go.mod go.sum          # xem đổi gì
git add go.mod go.sum; git commit -m "chore: lock go.mod/go.sum trước khi tái cấu trúc"
#   (hoặc git checkout -- go.mod go.sum nếu muốn bỏ thay đổi)

# 2. Build frontend để có frontend/dist (cần cho go:embed)
npm --prefix frontend ci
npm --prefix frontend run build

# 3. CỔNG KIỂM TRA THẬT: wails build (tự chạy npm build + compile Go + ldflags)
wails build

# 4. Test Go (chỉ phần không cần Windows-only nếu muốn, hoặc toàn bộ trên Windows)
go test ./internal/...

# 5. Ghi lại "baseline": đếm số platform được đăng ký (để so sánh sau Pha 3)
#    Tìm cách app log số registerer, hoặc tạm thêm log; ghi con số này lại.
```

✅ **Điều kiện qua Pha 0:** `wails build` ra file `build/bin/HVRIns.exe`, `go test ./internal/...`
pass, đã ghi lại số platform baseline. **Không bắt đầu di chuyển nếu chưa xanh.**

> 💡 Vì sao không dùng `go build ./...`? Trên bản sạch nó **lỗi** ở `go:embed` (chưa có
> `frontend/dist`) và **lỗi** trên non-Windows. `wails build` là cổng đáng tin.

---

## PHA 1 — 🔴 Secrets (khẩn cấp, độc lập)

> Chi tiết đầy đủ: [05-secrets-bao-mat.md](05-secrets-bao-mat.md). Tóm tắt:

```powershell
# Gỡ track (KHÔNG xoá file trên đĩa) các file chứa credential thật:
git rm --cached Config/Cookie/cookie_initial.txt
git rm --cached test_accounts_eaag.txt test_accounts_eaag_new.txt test_accounts_fresh.txt

# ⚠ TUYỆT ĐỐI KHÔNG đụng vào internal/cookie/embedded/cookie_initial.txt (file này go:embed, cần cho build!)
```

Thêm vào `.gitignore`:
```gitignore
# Secrets / runtime data
Config/Cookie/
test_accounts*.txt
__pycache__/
*.pyc
```

Sau đó:
1. **Rotate/vô hiệu hoá** mọi cookie/token đã lộ (coi như đã bị lộ vì còn trong git history).
2. Cân nhắc **rewrite history** (`git filter-repo` / BFG) như một bước riêng, có phối hợp.
3. Kiểm tra: `wails build` vẫn xanh (vì file embed của build không bị đụng).

✅ **Qua Pha 1:** `git status` không còn track secret; `wails build` vẫn xanh.

---

## PHA 2 — Dọn rác không ảnh hưởng Go build

Các thao tác này **không đụng tới file `.go` của app** → không thể làm hỏng compile.

```powershell
# Python lạc (hardcode E:/WEMAKE/...)
git rm _patch_datr_diag.py decode_request.py

# Rác bytecode
git rm -r --cached scripts/__pycache__

# Docs về docs/
git mv NVRINS_BUILD_GUIDE.md docs/NVRINS_BUILD_GUIDE.md
New-Item -ItemType Directory -Force docs/testing | Out-Null
git mv README_TEST_EAAG.md docs/testing/eaag-verify-flow.md

# build.bat về scripts/ (nhớ: nó phải cd về gốc trước khi gọi wails build)
git mv build.bat scripts/build.bat

# Script migrate dùng-1-lần → scripts/legacy/
New-Item -ItemType Directory -Force scripts/legacy | Out-Null
git mv scripts/migrate.ps1 scripts/legacy/
git mv scripts/rename_identity.ps1 scripts/legacy/
git mv scripts/recolor.py scripts/legacy/

# docs cũ → archive (rà soát trước khi xoá hẳn)
New-Item -ItemType Directory -Force docs/archive | Out-Null
git mv docs/old-docs/* docs/archive/

# (tuỳ chọn) docs/facebook → docs/flows
git mv docs/facebook docs/flows

# .kiro specs → docs/rebuild/specs (giữ bản gốc nếu còn dùng Kiro IDE)
New-Item -ItemType Directory -Force docs/rebuild/specs | Out-Null
# copy thay vì move nếu vẫn dùng Kiro:
Copy-Item .kiro/specs/hvrins-instagram-clone/*.md docs/rebuild/specs/
```

**Dọn `cmd/`:**
```powershell
# Tool thật → tools/
New-Item -ItemType Directory -Force tools | Out-Null
git mv cmd/icongen tools/icongen

# Xoá scratch + stub chết (xem danh sách đầy đủ ở 03-anh-xa-chi-tiet.md mục B)
git rm -r cmd/test_bloks_login cmd/regtest cmd/_testloginios cmd/emailtest `
          cmd/check_verified_email cmd/proxycheck cmd/proxytest cmd/testbody `
          cmd/test_regex cmd/test_ua cmd/testua cmd/test273 cmd/test_eaag_flow `
          cmd/test_messios cmd/testverios cmd/verifymess cmd/verifytest

# Dọn dependency thừa (SAU khi đã quyết số phận icongen)
go mod tidy
```

**Kiểm tra Pha 2:**
```powershell
# golang.org/x/image phải còn (vì tools/icongen vẫn dùng):
Select-String -Path go.mod -Pattern "golang.org/x/image"
go build ./tools/...        # icongen vẫn build
wails build                 # app vẫn xanh
```

✅ **Qua Pha 2:** gốc repo đã gọn hẳn, `cmd/` chỉ còn (trống/được dọn), `wails build` xanh.
Commit: `chore: dọn rác gốc, gom docs/scripts, xoá cmd scratch`.

---

## PHA 3 — ⭐ Cú chuyển vào `internal/app/` (MỘT commit nguyên tử)

> Đây là bước **rủi ro cao nhất**. Làm trọn trong **một commit** vì trạng thái nửa vời không compile.
> Đọc kỹ [06-go-wails-cho-newbie.md](06-go-wails-cho-newbie.md) trước.

**Bước 3.1 — Di chuyển file (giữ history bằng `git mv`):**
```powershell
New-Item -ItemType Directory -Force internal/app | Out-Null
git mv app.go               internal/app/app.go
git mv app_register.go      internal/app/register.go
git mv app_reg_sxxx.go      internal/app/register_sxxx.go
git mv app_verify.go        internal/app/verify.go
git mv app_banclone.go      internal/app/banclone.go
git mv app_tempmail_reg.go  internal/app/tempmail.go
git mv app_getdatr.go       internal/app/getdatr.go
git mv debug.go             internal/app/debug.go
git mv datadir.go           internal/app/datadir.go
git mv cpu_windows.go       internal/app/cpu_windows.go        # ⚠ giữ hậu tố _windows
git mv portrange_windows.go internal/app/portrange_windows.go  # ⚠ giữ hậu tố _windows
git mv app_test.go          internal/app/app_test.go
```

**Bước 3.2 — Đổi package** trong **tất cả** file vừa chuyển: `package main` → `package app`.

**Bước 3.3 — Export những gì `main.go` cần** (xem bảng ở [03 mục A](03-anh-xa-chi-tiet.md#a-file-go-ở-gốc--internalapp)):
- `startup` → `Startup`
- `appDataDir()` → `AppDataDir()`
- `expandEphemeralPortRange()` → `ExpandEphemeralPortRange()`
- Gói logic dùng `app.ctx` (trong `OnSecondInstanceLaunch`) thành method export, ví dụ
  `func (a *App) OnSecondInstance()` — **đừng** expose raw `ctx`.

**Bước 3.4 — Xử lý `AppVersion`** (điểm tinh tế, dễ sai — xem [07 rủi ro #1](07-checklist-rui-ro.md)):
- **Giữ `var AppVersion` trong `package main` ở `main.go`** → `build.bat` (`-X main.AppVersion`)
  **không cần đổi**.
- Nhưng `datadir.go` (đã chuyển vào `internal/app`) **không còn thấy** `main.AppVersion`.
  → Truyền version vào: gọi `app.SetVersion(AppVersion)` ngay sau `app.New()`, hoặc
  `app.New(AppVersion)`. `datadir.go` đọc version từ field nội bộ thay vì biến package.

**Bước 3.5 — Sửa `main.go`** thành bootstrap mỏng:
```go
package main

import (
    "context"
    "embed"
    // ...
    "HVRIns/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

var AppVersion = "dev"   // ldflags vẫn là -X main.AppVersion

func main() {
    flag.Parse()
    a := app.New()
    a.SetVersion(AppVersion)               // truyền version vào internal/app
    if err := os.Chdir(a.AppDataDir()); err != nil { /* ... */ }  // PHẢI là hành động đầu tiên
    a.ExpandEphemeralPortRange()
    err := wails.Run(&options.App{
        // ... Title/Width/...
        AssetServer: &assetserver.Options{Assets: assets},
        OnStartup:   a.Startup,
        OnBeforeClose: func(_ context.Context) bool { /* dùng a.IsConfirmedQuit()... */ },
        SingleInstanceLock: &options.SingleInstanceLock{
            UniqueId: instanceUniqueID(),
            OnSecondInstanceLaunch: func(_ options.SecondInstanceData) { a.OnSecondInstance() },
        },
        Bind: []interface{}{a},
    })
    // ...
}
```

> ⚠️ Giữ `os.Chdir(a.AppDataDir())` là **hành động đầu tiên** trong `main()` (trước `wails.Run`),
> nếu không mọi đọc đường dẫn tương đối (`Config/...`, `logs/`) sẽ trỏ sai chỗ.

**Bước 3.6 — Kiểm tra (chưa regenerate bindings nên FE chưa chạy, nhưng Go phải compile):**
```powershell
gofmt -w internal/app
go vet ./internal/app/...
go test ./internal/app/...     # app_test.go phải pass (chứng tỏ hàm private resolve đúng package)
npm --prefix frontend run build
go build .                     # compile package main ở gốc (cần frontend/dist đã build)
```

✅ **Qua Pha 3:** Go compile xanh, `app_test.go` pass. Commit nguyên tử:
`refactor: chuyển App logic vào internal/app (package app), main.go còn bootstrap`.

---

## PHA 4 — Regenerate Wails bindings + sửa import FE

Vì package của `App` đổi `main` → `app`, binding tự sinh sẽ chuyển từ `wailsjs/go/main/` sang
`wailsjs/go/app/`.

```powershell
wails generate module          # sinh lại frontend/wailsjs/go/app/...
```

Sửa **~10 file** `frontend/src/bridge/wails/*.ts`:
```diff
- import { Xxx } from '../../../wailsjs/go/main/App'
+ import { Xxx } from '../../../wailsjs/go/app/App'
```
(và đường dẫn `models` nếu đổi).

**Kiểm tra:**
```powershell
npm --prefix frontend run build
wails build
wails dev          # smoke test: cửa sổ mở, FE gọi được method App
```

✅ **Qua Pha 4:** app chạy thật, FE gọi backend qua binding mới. **Kiểm tra 2 điều dễ sai:**
- `GetAppVersion()` trả về version đã inject (**không** phải `"dev"`) → nếu là `"dev"`, app prod sẽ
  ghi data vào `bin/dev` (xem rủi ro AppVersion).
- Số platform đăng ký == baseline ở Pha 0 (blank-import không bị mất).

Commit: `chore: regenerate bindings (go/app), cập nhật import FE`.

---

## PHA 5 — config/ mẫu + tooling

```powershell
# Template "sạch" → config/sample (đặt .example cho rõ là mẫu)
New-Item -ItemType Directory -Force config/sample/Permanent, config/sample/Proxy, `
         config/sample/TempMail, config/sample/DeviceInfo | Out-Null
git mv Config/Permanent/mail.txt   config/sample/Permanent/mail.example.txt
git mv Config/Permanent/phone.txt  config/sample/Permanent/phone.example.txt
git mv Config/Proxy/proxy_rentmail.txt config/sample/Proxy/proxy_rentmail.example.txt
git mv Config/Proxy/proxy_tempmail.txt config/sample/Proxy/proxy_tempmail.example.txt
git mv Config/TempMail/domains.txt config/sample/TempMail/domains.example.txt
git mv Config/DeviceInfo/versions_and_builds*.txt config/sample/DeviceInfo/
```

Sửa `.vscode/launch.json`: `HVR_DATA_DIR` → `HVRINS_DATA_DIR` (bug có sẵn; code đọc
`HVRINS_DATA_DIR`). `program` giữ `${workspaceFolder}` (main vẫn ở gốc).

**Kiểm tra:** `wails dev` chạy, app đọc/ghi data đúng chỗ (`bin/dev/Config` ở dev).

✅ Commit: `chore: tách config mẫu sang config/sample, sửa launch.json`.

---

## PHA 6 — (Sau) Đổi tên frontend theo từng feature

> **Không làm cùng pha cấu trúc Go.** Làm **từng feature một**, kiểm tra sau mỗi feature.

Thứ tự an toàn:
1. **Bật alias `@/` trước tiên**: thêm `paths` vào `tsconfig.json` cho khớp `vite.config.ts`
   (đã map `@ → src`). Đổi dần import `../../../` → `@/...`. Việc này khiến di chuyển file
   **không** làm vỡ import.
2. Xoá 2 stub chết: `src/main.ts`, `src/App.vue`.
3. (tuỳ chọn) Làm phẳng `src/app/` → `src/main.ts` + `src/App.vue` + `src/router/`, cập nhật
   `index.html`.
4. `bridge/` → `services/` (**giữ nguyên độ sâu thư mục** để `wails/*.ts` còn resolve `../../../wailsjs`).
5. `modules/` → `features/`, gom `pages/*` vào từng feature.
6. Thêm script test: `"test": "vitest run"` trong `frontend/package.json`.

**Kiểm tra sau MỖI feature:** `npm --prefix frontend run build` + `wails dev`.

---

## PHA 7 — (Sau, TUỲ CHỌN) Ánh xạ `internal/` sâu

> Rủi ro cao, lợi ích thấp cho người mới. **Chỉ làm nếu thực sự cần** và sau khi đã quen Go.

Nếu làm:
- Di chuyển **theo khối nguyên** (cả `register/`, cả `verify/`), **không** từng file.
- Dùng công cụ IDE "move package" / `gopls` rename, **không** sửa tay.
- `verifybase` (210 importer) và `fakeinfo` (nhiều importer) đổi **sau cùng**.
- Sau mỗi khối: `wails build` + **đếm lại số platform đăng ký** (blank-import dễ mất).
- Coi chừng **import cycle** khi tách App thành usecase/adapter.

---

## Bảng lệnh kiểm tra dùng lại

| Mục đích | Lệnh (Windows) |
|----------|----------------|
| Tĩnh | `go vet ./...` |
| Compile package gốc | `npm --prefix frontend run build; go build .` |
| Test Go | `go test ./internal/...` |
| **Cổng thật** | `wails build` (ra `build/bin/HVRIns.exe`) |
| Smoke test | `wails dev` |
| Dọn dep | `go mod tidy` (chạy **sau cùng**) |

> ❗ Sau **mỗi** pha: chạy `wails build`, nếu xanh thì **commit ngay** rồi mới sang pha sau.

→ Đọc tiếp: [05-secrets-bao-mat.md](05-secrets-bao-mat.md) · [07-checklist-rui-ro.md](07-checklist-rui-ro.md)
