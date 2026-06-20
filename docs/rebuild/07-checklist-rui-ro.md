# 07 — Checklist tổng & danh sách rủi ro

## A. Checklist thực thi (in ra, tick từng ô)

### Pha 0 — Baseline
- [ ] `git diff go.mod go.sum` đã xem; commit/revert để cây git sạch
- [ ] `npm --prefix frontend ci && npm --prefix frontend run build` chạy được
- [ ] `wails build` ra `build/bin/HVRIns.exe` (XANH)
- [ ] `go test ./internal/...` pass
- [ ] Đã ghi lại **số platform đăng ký** (baseline để so sau)

### Pha 1 — Secrets 🔴
- [ ] Đã **rotate** cookie/token FB + Hotmail (việc quan trọng nhất)
- [ ] `git rm --cached` 4 file secret (Config/Cookie/cookie_initial.txt + 3 file test_accounts)
- [ ] **KHÔNG** đụng `internal/cookie/embedded/cookie_initial.txt`
- [ ] `.gitignore` đã thêm `Config/Cookie/`, `test_accounts*.txt`, `__pycache__/`, `*.pyc`
- [ ] (tuỳ chọn) lên kế hoạch rewrite history (filter-repo/BFG)
- [ ] `wails build` vẫn xanh

### Pha 2 — Dọn rác
- [ ] Xoá `_patch_datr_diag.py`, `decode_request.py`
- [ ] `git rm -r --cached scripts/__pycache__`
- [ ] Move docs: `NVRINS_BUILD_GUIDE.md`, `README_TEST_EAAG.md` → `docs/`
- [ ] Move `build.bat` → `scripts/` (giữ logic `cd` về gốc)
- [ ] Move script migrate 1-lần → `scripts/legacy/`
- [ ] Move `docs/old-docs/` → `docs/archive/`
- [ ] Move `cmd/icongen` → `tools/icongen`
- [ ] Xoá toàn bộ `cmd/*` scratch (17 thư mục)
- [ ] `go mod tidy` và xác nhận `golang.org/x/image` còn
- [ ] `wails build` xanh → commit

### Pha 3 — Chuyển internal/app ⭐ (một commit)
- [ ] `git mv` 12 file vào `internal/app/` (giữ hậu tố `_windows.go`)
- [ ] Đổi `package main` → `package app` trong tất cả
- [ ] Export: `Startup`, `AppDataDir`, `ExpandEphemeralPortRange`
- [ ] Gói logic dùng `app.ctx` thành method export (không expose raw ctx)
- [ ] Xử lý `AppVersion`: giữ ở `main`, truyền vào app qua `SetVersion`/constructor
- [ ] `main.go` mỏng: giữ `go:embed`, `os.Chdir(a.AppDataDir())` là **hành động đầu tiên**
- [ ] `go vet`, `go test ./internal/app/...` pass, `go build .` xanh → commit

### Pha 4 — Bindings + FE import
- [ ] `wails generate module` (sinh `wailsjs/go/app/`)
- [ ] Sửa ~10 import trong `frontend/src/bridge/wails/*.ts` (`go/main` → `go/app`)
- [ ] `wails build` + `wails dev` smoke test
- [ ] `GetAppVersion()` trả version đúng (KHÔNG phải `"dev"`)
- [ ] Số platform đăng ký == baseline (Pha 0)
- [ ] Commit

### Pha 5 — config + tooling
- [ ] Move template `Config/*` → `config/sample/*.example`
- [ ] Sửa `.vscode/launch.json`: `HVR_DATA_DIR` → `HVRINS_DATA_DIR`
- [ ] `wails dev` đọc/ghi data đúng chỗ → commit

### Pha 6 — FE (sau, từng feature)
- [ ] Bật alias `@/` (tsconfig + vite), đổi import `../` → `@/`
- [ ] Xoá stub `src/main.ts`, `src/App.vue`
- [ ] `bridge/` → `services/` (giữ độ sâu thư mục)
- [ ] `modules/` → `features/`, gom `pages/`
- [ ] Thêm `"test": "vitest run"` vào `package.json`
- [ ] Build + smoke test sau **mỗi** feature

### Pha 7 — internal/ sâu (tuỳ chọn, để sau)
- [ ] Chỉ làm nếu thực sự cần; di chuyển theo **khối**, dùng công cụ, không sửa tay
- [ ] Đếm lại platform sau mỗi khối

### Hoàn thiện
- [ ] Viết `README.md` gốc (đang rỗng)
- [ ] Viết lại `CLAUDE.md` cho đúng app thật
- [ ] Điền `author` trong `wails.json`

---

## B. Danh sách rủi ro (xếp theo mức độ nguy hiểm)

### 🔴 RR-1 — `AppVersion` cross-package (bẫy âm thầm)
`datadir.go` đọc `AppVersion` để chọn dev (`bin/dev`) vs prod (thư mục exe). Khi `datadir.go` vào
`internal/app` mà `AppVersion` ở lại `package main`, nó **không thấy** biến đó nữa.
- **Nếu compile lỗi:** dễ phát hiện, sửa bằng cách truyền version vào.
- **Nếu "sửa ẩu"** bằng cách khai một `var AppVersion` mới trong package app → nó mặc định `"dev"`
  → **app prod ghi data sai chỗ** (vào `bin/dev`), không báo lỗi.
- **Cách đúng:** giữ `AppVersion` ở `main`, gọi `a.SetVersion(AppVersion)` sau `app.New()`;
  `datadir.go` đọc từ field nội bộ. **Kiểm tra:** `GetAppVersion()` ở bản build phải ra version thật.

### 🔴 RR-2 — `go:embed` không cho `../` (lý do main.go ở gốc)
Chuyển `main.go` vào `cmd/app/` làm `//go:embed all:frontend/dist` **vỡ** (không trỏ ra cha được)
và `wails build` không tìm thấy main. → **Giữ `main.go` ở gốc** (deviation đã ghi).

### 🔴 RR-3 — Đổi package làm vỡ binding FE
`package main` → `app` ⇒ binding chuyển `wailsjs/go/main` → `wailsjs/go/app` ⇒ 10 file
`bridge/wails/*.ts` lỗi import tới khi: (a) chạy `wails generate module`, (b) sửa import. Quên
regenerate = FE gọi symbol không tồn tại.

### 🔴 RR-4 — Blank-import mất → platform không đăng ký (âm thầm)
207 blank-import kích hoạt `init()` đăng ký platform. Mất một dòng = vẫn compile nhưng platform
biến mất lúc chạy. → **Đếm số platform trước/sau.** (Giảm rủi ro: chúng nằm trong `app.go`/
`app_reg_sxxx.go` nên tự đi theo.)

### 🟠 RR-5 — Tính nguyên tử của cú chuyển
12 file là **một** package; trạng thái nửa vời không compile. → Làm trọn trong **một commit**.

### 🟠 RR-6 — `main.go` dùng symbol private của App
`main.go` hiện gọi `app.ctx`, `app.startup` (chữ thường). Sau khi tách phải export. `ctx` nên gói
vào method export (vd `OnSecondInstance`), **không** expose raw ctx. (`RequestQuit`,
`IsConfirmedQuit`, `EmitQuitConfirm` đã public.)

### 🟠 RR-7 — `os.Chdir(appDataDir())` phải chạy đầu tiên
Mọi đọc đường dẫn tương đối (`Config/...`, `logs/`) phụ thuộc lệnh này đã chạy. Nếu cú chuyển vô
tình đẩy nó vào `app.Startup` (chạy muộn hơn) hay làm mất → đọc/ghi sai thư mục. → Giữ là hành
động **đầu tiên** trong `main()`.

### 🟠 RR-8 — App chỉ build trên Windows
Không có bản `_other.go` cho cpu/portrange. → Mọi verify trên **Windows**; `go build ./...` trên
non-Windows/CI sẽ lỗi vì lý do không liên quan, che mất regression thật.

### 🟡 RR-9 — `go build` trên cây sạch lỗi go:embed
`frontend/dist` gitignored → bản clone mới không có dir → `go:embed` lỗi. → Build frontend trước
hoặc dùng `wails build`.

### 🟡 RR-10 — Đổi đường dẫn `internal/*` lan rộng
`app.go` import ~30 package `internal/*`; các tool `cmd/*` cũng import. Đổi tên một thư mục
`internal/` = sửa mọi nơi import + nguy cơ import cycle. → **Hoãn** ánh xạ sâu (Pha 7).

### 🟡 RR-11 — go:embed trong internal/ (cookie, igcore, iosmess)
3 package internal có `go:embed`. Nếu (Pha 7) di chuyển, **mang theo thư mục asset** cạnh file `.go`.

### 🟡 RR-12 — Installer move làm vỡ `wails build -nsis`
Wails tự sinh `wails_tools.nsh` ở `build/windows/installer/`; `project.nsi` dùng đường dẫn tương
đối `..\..\bin`. → **Giữ** ở chỗ cũ, chỉ tài liệu hoá ở `infra/installer/`.

### 🟡 RR-13 — FE: 192 import tương đối, 0 alias
Di chuyển file FE làm vỡ hàng loạt `../`. → **Bật alias `@/` trước**, làm từng feature.

### 🟡 RR-14 — `go mod tidy` gỡ nhầm dependency
`golang.org/x/image` chỉ `tools/icongen` dùng. Chạy `go mod tidy` **sau** khi đã quyết giữ icongen,
rồi `go build ./tools/...` xác nhận.

### 🟢 RR-15 — Secret còn trong history
`git rm --cached` không gột history. → Rotate credential là bắt buộc; rewrite history là bước riêng.

### 🟢 RR-16 — Nhầm 2 file `cookie_initial.txt`
Gỡ `Config/Cookie/cookie_initial.txt` (lộ) nhưng **giữ** `internal/cookie/embedded/cookie_initial.txt`
(go:embed, cần cho build).

---

## C. "Định nghĩa hoàn thành" cho mỗi pass

**Pass cấu trúc (Pha 0–5) coi là xong khi:**
1. `wails build` ra `HVRIns.exe` (Windows) — XANH
2. `wails dev` mở app, FE gọi được method backend
3. `go test ./internal/...` pass (gồm `app_test.go` ở vị trí mới)
4. `GetAppVersion()` trả version thật (không phải `"dev"`)
5. Số platform đăng ký == baseline
6. Gốc repo gọn: chỉ còn `main.go` + file config chuẩn + thư mục chuẩn
7. Không còn secret bị track

→ Quay lại [README.md](README.md) · [04-ke-hoach-thuc-thi.md](04-ke-hoach-thuc-thi.md)
