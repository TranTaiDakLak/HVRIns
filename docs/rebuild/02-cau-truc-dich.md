# 02 — Cấu trúc đích (đã chỉnh cho HVRIns)

Đây là cây thư mục **đích** — đã điều chỉnh khung chuẩn cho đúng thực tế của một app Wails thật.
Phần "lệch chuẩn" được đánh dấu rõ và giải thích.

## 1. Cây thư mục đích

```
HVRIns/
├── main.go                      # ⚠ LỆCH CHUẨN có chủ ý: entry Wails Ở LẠI GỐC
│                                #    (giữ go:embed + AppVersion + wails.Run; gọi internal/app)
├── wails.json                   # ✅ config Wails (phải ở gốc)
├── go.mod  go.sum               # ✅
├── README.md                    # viết lại (hiện đang rỗng)
├── CLAUDE.md                    # viết lại cho đúng dự án (hoặc đưa vào docs/agent/)
├── .gitignore  .gitattributes
│
├── cmd/
│   └── app/                     # (TRỐNG — xem ghi chú deviation; KHÔNG tạo main.go ở đây)
│
├── internal/
│   ├── app/                     # 🆕 struct App + toàn bộ method bind cho FE (chuyển từ gốc)
│   │   ├── app.go               #    (tách nhỏ từ file 317KB)
│   │   ├── accounts.go          #    ListAccounts/Import/Delete...
│   │   ├── settings.go          #    SettingsData + Load/Save
│   │   ├── profiles.go          #    Create/Clone/Delete profile
│   │   ├── register.go          #    (app_register.go)
│   │   ├── register_sxxx.go     #    (app_reg_sxxx.go)
│   │   ├── verify.go            #    (app_verify.go)
│   │   ├── banclone.go          #    (app_banclone.go)
│   │   ├── tempmail.go          #    (app_tempmail_reg.go)
│   │   ├── getdatr.go           #    (app_getdatr.go)
│   │   ├── upload.go  stats.go  resources.go  dialogs.go   # các nhóm tách từ app.go
│   │   ├── debug.go             #    (debug.go)
│   │   ├── datadir.go           #    (datadir.go) — nhận version từ ngoài
│   │   ├── cpu_windows.go       #    ⚠ GIỮ NGUYÊN hậu tố _windows.go
│   │   ├── portrange_windows.go #    ⚠ GIỮ NGUYÊN hậu tố _windows.go
│   │   └── app_test.go          #    test white-box (cùng package)
│   │
│   ├── instagram/               # GIỮ NGUYÊN ĐƯỜNG DẪN (2960 file, plugin-registry)
│   ├── email/  igcore/  proxy/  cookie/  fbdata/  settings/
│   ├── runner/  stats/  iplookup/  httpclient/  httpx/  config/  clonehv/
│   │
│   └── (TƯƠNG LAI — tuỳ chọn, pha sau)
│       ├── domain/              # type lõi (Session, RegInput, fakeinfo...)
│       ├── usecase/             # runner, factory/registry dispatch
│       └── adapter/
│           ├── external/        # instagram versions, email, proxy, clonehv, iplookup...
│           └── repository/      # cookie, fbdata, settings/store, config
│
├── frontend/                    # GIỮ Ở GỐC (vì go:embed) — đã khá chuẩn
│   ├── index.html
│   ├── package.json  vite.config.ts  vitest.config.ts  tsconfig*.json
│   ├── wailsjs/                 # Wails tự sinh (vị trí cố định)
│   └── src/
│       ├── main.ts  App.vue     # entry thật (gộp từ src/app/, xoá stub cũ)
│       ├── router/
│       ├── components/{ui,grid,form,feedback,shell}/   # ✅ đã chuẩn
│       ├── composables/         # ✅
│       ├── features/            # 🔄 đổi tên từ modules/ (accounts, auth-source, settings, reg-stats...)
│       │   └── <feature>/{pages,components,store}/
│       ├── services/            # 🔄 đổi tên từ bridge/ (contracts/client/mock/wails)
│       ├── stores/  types/  constants/  styles/  assets/
│
├── tools/
│   └── icongen/                 # 🔄 chuyển từ cmd/icongen (tool build thật)
│
├── scripts/                     # build.bat→đây; gen_icon/gen_ico/soak-monitor; legacy/ cho script cũ
│   ├── build.bat (hoặc build.ps1)
│   ├── gen_icon.py  gen_ico.py  soak-monitor.ps1
│   └── legacy/                  # migrate.ps1, rename_identity.ps1, recolor.py (đã dùng 1 lần)
│
├── config/
│   └── sample/                  # 🔄 template "sạch" từ Config/ (đặt hậu tố .example)
│
├── build/                       # ✅ GIỮ NGUYÊN (scaffold Wails) — build/bin/ vẫn gitignored
│   ├── appicon.png  darwin/  windows/
│   └── windows/installer/       # ⚠ GIỮ NGUYÊN (Wails tự sinh ở đây; xem deviation installer)
│
├── infra/
│   └── installer/               # (tuỳ chọn) bản sao/tài liệu trỏ về build/windows/installer
│
├── docs/
│   ├── rebuild/                 # 📄 KẾ HOẠCH NÀY
│   ├── flows/                   # 🔄 đổi tên từ docs/facebook/ (doc luồng đang dùng)
│   ├── archive/                 # 🔄 đổi tên từ docs/old-docs/
│   ├── testing/                 # README_TEST_EAAG.md → đây
│   └── NVRINS_BUILD_GUIDE.md    # chuyển từ gốc
│
└── tests/
    ├── go/                      # (tương lai) test black-box Go
    └── frontend/                # (tương lai) test vitest
```

Chú thích: 🆕 mới · 🔄 đổi tên/di chuyển · ⚠ cần cẩn thận · ✅ đã đúng

---

## 2. Giải thích từng thư mục (cho newbie Go)

### `main.go` (ở gốc) — entry point mỏng
File `package main` duy nhất. Chỉ chứa: directive `//go:embed all:frontend/dist`, biến
`AppVersion`, và hàm `main()` gọi `app.New()` rồi `wails.Run(...)`. **Không chứa business logic.**
→ Vì sao ở gốc mà không vào `cmd/app/`: xem mục deviation bên dưới.

### `cmd/app/` — để TRỐNG (deviation)
Theo chuẩn, đây là nơi đặt entry. Nhưng do ràng buộc `go:embed` + `wails build`, ta để trống và
giữ entry ở gốc. (Có thể tạo file `cmd/app/.gitkeep` + README giải thích, hoặc bỏ luôn `cmd/`.)

### `internal/app/` — "bridge layer" (trái tim mới)
Đây chính là cái chuẩn gọi *"bootstrap, DI, expose bridge cho FE"*. Toàn bộ struct `App` và các
method `(a *App) XxxYyy()` (mà Vue gọi qua Wails) nằm ở đây, dưới **một package tên `app`**.
- `internal/` là **ranh giới private do Go cưỡng chế**: chỉ code trong cùng module mới import được.
- File khổng lồ `app.go` được **tách thành nhiều file nhỏ** theo trách nhiệm — vì cùng package nên
  tách file **không làm đổi import nào**, rủi ro gần như 0.

### `internal/instagram/`, `email/`, ... — business logic hiện có
**Giữ nguyên đường dẫn ở pass 1.** Về *khái niệm* chúng map sang `adapter/external` (client gọi
API ngoài) và `adapter/repository` (lưu trữ), nhưng **di chuyển thật = đổi đường dẫn import của
~2900 file** → rủi ro rất cao. Ánh xạ sâu này là **pha tuỳ chọn, để sau**.

### `frontend/` (ở gốc) — Vue app
**Phải ở gốc** vì `main.go` nhúng `frontend/dist`. Bên trong đã khá chuẩn; chỉ cần đổi tên
`bridge/ → services/`, `modules/ → features/`, xoá stub chết — và **làm từ từ theo từng feature**
sau khi đã bật alias `@/`.

### `tools/` — tool dev/build thật
Khác với `cmd/` (scratch), đây là nơi đặt tool **đáng giữ**: `icongen`. Vẫn là `package main`,
chạy bằng `go run ./tools/icongen`.

### `scripts/` — script build/run/ops
`build.bat`, script sinh icon (Python/Pillow), `soak-monitor.ps1`. Các script migrate dùng-một-lần
đưa vào `scripts/legacy/` hoặc xoá.

### `config/sample/` — cấu hình MẪU
Chỉ chứa **template sạch** (không secret), đặt hậu tố `.example`. Dữ liệu thật mà app dùng lúc chạy
**không** nằm đây — nó nằm ở `appDataDir()` (xem mục 4).

### `build/` — vừa là source vừa là output (cẩn thận!)
Wails dùng `build/` cho **asset nguồn** (icon, manifest, plist, script installer — được track) và
`build/bin/` cho **output** (gitignored). **Đừng xoá/di chuyển cả `build/`** — `wails build` cần nó ở gốc.

### `docs/`, `tests/` — như chuẩn
`docs/rebuild/` là kế hoạch này. `tests/go` và `tests/frontend` chỉ dành cho **test black-box mới**
(xem mục 3 về vì sao test cũ phải ở cạnh code).

---

## 3. Quy tắc về test (quan trọng, dễ sai)

> **Không gom hết `*_test.go` vào `tests/go/`.**

Trong Go, test có 2 loại:
- **White-box** (`package app`): nhìn thấy hàm/type **private** (chữ thường). **Phải nằm cùng thư
  mục** với code nó test. Ví dụ: `app_test.go` test `isVerifiableAccountFile` (private) → phải ở
  `internal/app/`. 31 file `*_test.go` trong `internal/` cũng vậy → **giữ nguyên tại chỗ**.
- **Black-box** (`package app_test`): chỉ thấy hàm **public**. Loại này mới đặt ở `tests/go/`.

→ `tests/go/` và `tests/frontend/` ở pass đầu sẽ **trống** (hoặc chỉ có test mới viết sau).

---

## 4. Hai cây "Config" — đừng nhầm lẫn

Đây là điểm hay gây nhầm nhất:

| | `Config/` ở gốc repo | `appDataDir()/Config/` (runtime) |
|---|---|---|
| Vị trí | Thư mục gốc, **được track** | Dev: `bin/dev/Config`; Prod: thư mục chứa `.exe` |
| App có đọc lúc chạy? | **KHÔNG** | **CÓ** (đây mới là dữ liệu thật) |
| Nội dung | Template + 1 file secret bị lộ | Cookie/proxy/mail/settings thật |
| Git | Cần dọn (template → `config/sample/`, secret → gỡ) | Đã gitignore |

**Vì sao app không đọc `Config/` ở gốc?** Vì `main()` gọi `os.Chdir(appDataDir())` **ngay đầu tiên**
(main.go dòng 48). Sau lệnh này, mọi đường dẫn tương đối như `"Config/Settings"` trong code được
tính từ `appDataDir()`, **không phải** từ gốc repo.

> 🟢 **Hệ quả quan trọng:** đổi tên/xoá `Config/` ở gốc **KHÔNG làm hỏng app lúc chạy** (vì app
> không bao giờ có CWD ở gốc repo). Đây thuần tuý là việc dọn dẹp + xử lý secret. (Một agent phân tích
> ban đầu lo "đổi `Config/` sẽ làm hỏng literal đường dẫn trong Go" — nhưng điều đó chỉ đúng nếu ai
> đó bỏ `os.Chdir`. Đã được xác minh và làm rõ.)

---

## 5. Bảng "lệch chuẩn có chủ ý" (deviations)

| Khung chuẩn nói | Ta làm | Lý do |
|------------------|--------|-------|
| Entry ở `cmd/app/main.go` | `main.go` ở **gốc** | `go:embed` cấm `../`; `wails build` giả định main ở gốc |
| `config/` (chữ thường) | `config/sample/` cho template; runtime data ở `appDataDir()` | Tách rõ mẫu vs dữ liệu thật + secret |
| `infra/installer/` | **Giữ** ở `build/windows/installer/`, chỉ tài liệu hoá ở `infra/` | Wails tự sinh `wails_tools.nsh` ở đó; di chuyển = hỏng `wails build -nsis` |
| `tests/{go,frontend}` cho mọi test | Test white-box **ở cạnh code**; chỉ black-box vào `tests/` | Quy tắc visibility của Go |
| `internal/{domain,usecase,adapter}` | Giữ `internal/instagram` v.v.; ánh xạ sâu **để sau** | Đổi đường dẫn ~2900 file = rủi ro cao, lợi ích thấp cho newbie |

→ Đọc tiếp: [03-anh-xa-chi-tiet.md](03-anh-xa-chi-tiet.md) cho bảng ánh xạ từng-file.
