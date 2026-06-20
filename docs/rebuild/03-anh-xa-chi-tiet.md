# 03 — Bảng ánh xạ chi tiết (hiện tại → đích)

> Bảng tra cứu khi thực thi. Cột **Hành động**: `MOVE` = di chuyển · `RENAME` = đổi tên ·
> `SPLIT` = tách nhỏ · `DELETE` = xoá · `KEEP` = giữ nguyên · `GITIGNORE` = gỡ track + ignore.
> Cột **Pha** trỏ tới bước trong [04-ke-hoach-thuc-thi.md](04-ke-hoach-thuc-thi.md).

---

## A. File Go ở gốc → `internal/app/`

> ⚠️ **Tất cả phải di chuyển CÙNG LÚC trong MỘT commit** và đổi `package main` → `package app`.
> Vì chúng là một package, trạng thái nửa vời sẽ không compile.

| Hiện tại | Đích | Hành động | Pha | Ghi chú |
|----------|------|-----------|-----|---------|
| `main.go` | `main.go` (gốc) | **KEEP** | 2 | Giữ `go:embed`, `AppVersion`; sửa để gọi `app.New()`, export-call |
| `app.go` | `internal/app/app.go` (+tách) | **MOVE + SPLIT** | 3 | `package main`→`app`; tách thành accounts/settings/profiles/upload/stats/resources/dialogs.go |
| `app_register.go` | `internal/app/register.go` | **MOVE** | 3 | Đổi package |
| `app_reg_sxxx.go` | `internal/app/register_sxxx.go` | **MOVE** | 3 | Chứa 1 blank-import; đi theo |
| `app_verify.go` | `internal/app/verify.go` | **MOVE** | 3 | Đổi package |
| `app_banclone.go` | `internal/app/banclone.go` | **MOVE** | 3 | Đổi package |
| `app_tempmail_reg.go` | `internal/app/tempmail.go` | **MOVE** | 3 | Đổi package |
| `app_getdatr.go` | `internal/app/getdatr.go` | **MOVE** | 3 | Đổi package |
| `debug.go` | `internal/app/debug.go` | **MOVE** | 3 | Đổi package |
| `datadir.go` | `internal/app/datadir.go` | **MOVE** | 3 | ⚠ đọc `AppVersion` — phải nhận version từ ngoài (xem rủi ro) |
| `cpu_windows.go` | `internal/app/cpu_windows.go` | **MOVE** | 3 | ⚠ **GIỮ hậu tố `_windows.go`** |
| `portrange_windows.go` | `internal/app/portrange_windows.go` | **MOVE** | 3 | ⚠ Giữ `_windows`; export `ExpandEphemeralPortRange` |
| `app_test.go` | `internal/app/app_test.go` | **MOVE** | 3 | White-box; **không** vào `tests/go` |

**Cần export (vì `main.go` gọi xuyên package sau khi tách):**

| Hiện tại (private) | Sau (public) | Vì sao |
|--------------------|--------------|--------|
| `startup` | `Startup` | `main.go` set `OnStartup: app.Startup` |
| `appDataDir()` | `app.AppDataDir()` | `main.go` gọi `os.Chdir(...)` |
| `expandEphemeralPortRange()` | `app.ExpandEphemeralPortRange()` | `main.go` gọi lúc khởi động |
| `app.ctx` (field) | gói logic dùng `ctx` vào method export (vd `app.OnSecondInstance()`) | `main.go` dùng `app.ctx` trong `OnSecondInstanceLaunch` — **không** nên expose raw ctx |
| `NewApp` | (đã public, giữ; có thể đổi tên `app.New`) | constructor |
| `IsConfirmedQuit`, `EmitQuitConfirm`, `RequestQuit` | (đã public) | dùng trong `OnBeforeClose` |

---

## B. `cmd/` → dọn sạch

| Hiện tại | Đích | Hành động | Pha | Ghi chú |
|----------|------|-----------|-----|---------|
| `cmd/icongen/` | `tools/icongen/` | **MOVE** | 2 | Tool thật; nơi DUY NHẤT dùng `golang.org/x/image` |
| `cmd/test_bloks_login/` | — | **DELETE** | 2 | main rỗng, DEPRECATED |
| `cmd/regtest/` (cả thư mục) | — | **DELETE** | 2 | scratch nhiều file; có 1 file `//go:build ignore` |
| `cmd/_testloginios/` | — | **DELETE** | 2 | scratch; hardcode proxy secret |
| `cmd/emailtest/` | — | **DELETE** | 2 | 🔴 chứa 10 Hotmail + token thật |
| `cmd/check_verified_email/` | — | **DELETE** | 2 | scratch (hoặc `tools/` nếu chủ muốn giữ) |
| `cmd/proxycheck/`, `cmd/proxytest/`, `cmd/testbody/` | → unit test `internal/proxy/*_test.go` | **DELETE** | 2 | Nên chuyển thành test bảng thật |
| `cmd/test_regex/` | → unit test cạnh parser | **DELETE** | 2 | |
| `cmd/test_ua/`, `cmd/testua/` | (tuỳ chọn) gộp `tools/ua-dump/` | **DELETE** | 2 | Trùng nhau; xoá hoặc giữ 1 |
| `cmd/test273/`, `cmd/test_eaag_flow/`, `cmd/test_messios/`, `cmd/testverios/`, `cmd/verifymess/`, `cmd/verifytest/` | — | **DELETE** | 2 | scratch network harness, không assertion |

> 🔁 **Sau khi xoá**: chạy `go mod tidy` **sau cùng**, và xác nhận `golang.org/x/image` vẫn còn
> (vì `tools/icongen` vẫn dùng). Nếu lỡ xoá cả icongen thì dependency này mới bị gỡ.

---

## C. `internal/` → giữ nguyên (pass 1)

| Hiện tại | Đích (khái niệm) | Hành động pass 1 | Ghi chú |
|----------|------------------|------------------|---------|
| `internal/instagram/**` (2960 file) | `adapter/external/instagram/` | **KEEP** | Di chuyển = đổi ~2900 import + 207 blank-import → để sau |
| `internal/email/**` | `adapter/external/email/` | **KEEP** | Có thể move khối (lợi ích thấp pass 1) |
| `internal/proxy`, `clonehv`, `iplookup`, `igcore` | `adapter/external/...` | **KEEP** | |
| `internal/cookie`, `fbdata`, `config`, `settings/store` | `adapter/repository/...` | **KEEP** | cookie & igcore **có go:embed** — nếu move phải mang theo thư mục asset |
| `internal/runner`, `stats` | `usecase/...` | **KEEP** | orchestration |
| `internal/settings` (model/schema/validation) | `domain/settings` | **KEEP** | Đã phân tầng đẹp — mẫu tham khảo |
| `internal/**/*_test.go` (31 file) | (giữ cạnh code) | **KEEP** | White-box; không gom vào `tests/` |

> Việc ánh xạ sâu sang `domain/usecase/adapter` được mô tả ở cột "khái niệm" để bạn hiểu hướng đi,
> nhưng **pha thực hiện là tuỳ chọn và để sau** (xem Pha 7 trong [04](04-ke-hoach-thuc-thi.md)).

---

## D. `frontend/` → đổi tên dần (pass 2)

| Hiện tại | Đích | Hành động | Pha | Ghi chú |
|----------|------|-----------|-----|---------|
| `frontend/` | `frontend/` | **KEEP** | — | **Phải ở gốc** (go:embed) |
| `frontend/src/main.ts` (stub `export {}`) | — | **DELETE** | 6 | Stub chết |
| `frontend/src/App.vue` (stub `<div/>`) | — | **DELETE** | 6 | Stub chết |
| `frontend/src/app/main.ts` | `frontend/src/main.ts` | **MOVE** | 6 | Cập nhật `index.html` trỏ lại |
| `frontend/src/app/App.vue` | `frontend/src/App.vue` | **MOVE** | 6 | |
| `frontend/src/app/router/` | `frontend/src/router/` | **MOVE** | 6 | |
| `frontend/src/bridge/` | `frontend/src/services/` | **RENAME** | 6 | ⚠ **giữ nguyên độ sâu** để `wails/*.ts` vẫn resolve `../../../wailsjs` |
| `frontend/src/modules/accounts/` | `frontend/src/features/accounts/` | **RENAME** | 6 | Gom `AccountsPage.vue` vào |
| `frontend/src/modules/auth-source/` | `frontend/src/features/auth-source/` | **RENAME** | 6 | |
| `frontend/src/pages/*` (11 file) | `frontend/src/features/<domain>/pages/` | **MOVE** | 6 | ⚠ rủi ro cao: `AccountsPage.vue` có 21 import tương đối |
| `frontend/src/components/settings/`, `src/schema/` | `frontend/src/features/settings/` | **MOVE** | 6 | |
| `frontend/src/components/{MailDomainStatsTable,RegVerStatsTable}.vue` | `features/reg-stats/components/` | **MOVE** | 6 | Đang nằm lẻ ở gốc components/ |
| `frontend/src/components/{ui,grid,form,feedback,shell}/` | (giữ nguyên) | **KEEP** | — | ✅ đã chuẩn |
| `frontend/src/composables/`, `stores/`, `types/`, `constants/`, `styles/`, `assets/` | (giữ) | **KEEP** | — | ✅ |
| `frontend/generate-icon.html` | `scripts/` hoặc xoá | **MOVE** | 2 | Tool 1 lần; trùng với `icongen` |
| `frontend/wailsjs/` | (giữ) | **KEEP** | — | Wails tự sinh; namespace `go/main`→`go/app` sau khi đổi package Go |

> 🔑 **Bắt buộc làm trước khi reorg FE**: bật alias `@/` (vite + tsconfig) và đổi import `../` →
> `@/` để việc di chuyển file không làm vỡ 192 import tương đối.

---

## E. Config / build / scripts

| Hiện tại | Đích | Hành động | Pha | Ghi chú |
|----------|------|-----------|-----|---------|
| `Config/Cookie/cookie_initial.txt` | — | **GITIGNORE** | 1 | 🔴 48KB token thật — gỡ track + rotate |
| `Config/Permanent/{mail,phone}.txt` | `config/sample/Permanent/*.example` | **MOVE** | 5 | Chỉ là comment header (an toàn) |
| `Config/Proxy/proxy_*.txt` | `config/sample/Proxy/*.example.txt` | **MOVE** | 5 | Template an toàn |
| `Config/TempMail/domains.txt` | `config/sample/TempMail/domains.example.txt` | **MOVE** | 5 | |
| `Config/DeviceInfo/versions_and_builds*.txt` | `config/sample/DeviceInfo/` | **MOVE** | 5 | 1 file rỗng 0 byte |
| `build/` (toàn bộ) | `build/` | **KEEP** | — | Wails cần ở gốc; `build/bin/` đã ignore |
| `build/windows/installer/{project.nsi,wails_tools.nsh}` | (giữ) + tài liệu ở `infra/installer/` | **KEEP** | — | ⚠ Wails tự sinh ở đây; **đừng move** (xem deviation) |
| `build.bat` | `scripts/build.bat` | **MOVE** | 2 | Phải `cd` về gốc trước khi `wails build` |
| `scripts/__pycache__/*.pyc` | — | **GITIGNORE** | 1 | Rác bytecode |
| `scripts/gen_icon.py`, `gen_ico.py`, `soak-monitor.ps1` | (giữ) | **KEEP** | — | Tiện ích còn dùng |
| `scripts/migrate.ps1`, `rename_identity.ps1`, `recolor.py` | `scripts/legacy/` | **MOVE** | 2 | Migrate 1 lần (đã xong) |
| `.gitignore` | (sửa) | **EDIT** | 1 | Thêm `Config/Cookie/`, `test_accounts*.txt`, `__pycache__/`, `*.pyc`; cân nhắc đảo chính sách dòng `!build/bin/Config/` |
| `.gitattributes` | (giữ) | **KEEP** | — | Đúng chuẩn |
| `wails.json` | (giữ) | **KEEP** | — | Có thể điền `author` |

---

## F. Docs & file gốc khác

| Hiện tại | Đích | Hành động | Pha | Ghi chú |
|----------|------|-----------|-----|---------|
| `_patch_datr_diag.py` | — | **DELETE** | 2 | Hardcode `E:/WEMAKE/...`; vẫn còn trong git history nếu cần |
| `decode_request.py` | — | **DELETE** | 2 | Tương tự |
| `test_accounts_eaag.txt` | — | **GITIGNORE** | 1 | 🔴 SECRET |
| `test_accounts_eaag_new.txt` | — | **GITIGNORE** | 1 | 🔴 SECRET |
| `test_accounts_fresh.txt` | — | **GITIGNORE** | 1 | 🔴 SECRET |
| `README.md` (rỗng) | `README.md` | **KEEP+VIẾT** | 7 | Viết overview thật |
| `README_TEST_EAAG.md` | `docs/testing/eaag-verify-flow.md` | **MOVE** | 2 | Sửa link chết bên trong |
| `NVRINS_BUILD_GUIDE.md` | `docs/NVRINS_BUILD_GUIDE.md` | **MOVE** | 2 | Tên cũ "NVRIns" — ghi chú lỗi thời |
| `CLAUDE.md` | `CLAUDE.md` (viết lại) | **KEEP+VIẾT** | 7 | Nội dung mâu thuẫn thực tế — viết lại mô tả app thật |
| `.kiro/specs/.../*.md` | `docs/rebuild/specs/` | **MOVE** | 2 | requirements/design/tasks.md là doc giá trị |
| `.kiro/.config.kiro` | (giữ làm IDE meta hoặc gitignore) | **KEEP** | — | Metadata Kiro IDE |
| `.vscode/launch.json` | (sửa nhỏ) | **EDIT** | 5 | Sửa `HVR_DATA_DIR` → `HVRINS_DATA_DIR` |
| `docs/facebook/` (11 file) | `docs/flows/` | **RENAME** | 2 | (tuỳ chọn) doc luồng đang dùng |
| `docs/old-docs/` (18 file) | `docs/archive/` | **MOVE** | 2 | Rà từng file trước khi xoá hẳn |

→ Đọc tiếp: [04-ke-hoach-thuc-thi.md](04-ke-hoach-thuc-thi.md) cho thứ tự thực thi an toàn.
