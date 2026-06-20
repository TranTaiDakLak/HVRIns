# 01 — Phân tích hiện trạng

> Số liệu lấy từ `git ls-files` và đọc trực tiếp source tại thời điểm phân tích.
> Tổng: **~3327 file** được track.

## 1. Bản đồ thư mục gốc hiện tại

```
HVRIns/
├── .git/  .vscode/  .kiro/                 # meta / IDE
├── .gitattributes  .gitignore
├── CLAUDE.md                               # ⚠ lỗi thời (mô tả sai dự án)
├── README.md                               # rỗng (0 byte)
├── README_TEST_EAAG.md                     # doc test, nên vào docs/
├── NVRINS_BUILD_GUIDE.md                   # 21KB, nên vào docs/
│
├── main.go                                 # ✅ entry Wails (go:embed, AppVersion) — GIỮ Ở GỐC
├── app.go                                  # 🔴 317KB / 7315 dòng — struct App + 121 method
├── app_register.go                         # 🔴 219KB — RunRegister + helper
├── app_reg_sxxx.go                         # 64KB — helper register sxxx
├── app_verify.go                           # 64KB — RunVerify
├── app_banclone.go  app_getdatr.go  app_tempmail_reg.go
├── datadir.go  debug.go                    # helper (đọc AppVersion)
├── cpu_windows.go  portrange_windows.go    # chỉ build trên Windows
├── app_test.go                             # test white-box (gọi hàm private)
│
├── _patch_datr_diag.py  decode_request.py  # 🗑 script lạc (hardcode E:/WEMAKE/...)
├── test_accounts_eaag.txt                  # 🔴 SECRET: cookie+token FB thật
├── test_accounts_eaag_new.txt              # 🔴 SECRET
├── test_accounts_fresh.txt                 # 🔴 SECRET
├── build.bat                               # script build → nên vào scripts/
├── wails.json  go.mod  go.sum              # ✅ chuẩn, giữ ở gốc
│
├── cmd/         (20 file / ~18 chương trình scratch + 1 tool thật icongen)
├── internal/    (3103 file — toàn bộ business logic)
├── frontend/    (118 file — Vue 3 app)
├── docs/        (27 file — facebook/ + old-docs/)
├── Config/      (9 file — template + 1 secret)
├── build/       (9 file — scaffold Wails)
└── scripts/     (8 file — python + powershell + __pycache__ rác)
```

Chú thích: 🔴 = file lớn/secret cần xử lý · 🗑 = rác cần xoá · ✅ = đã đúng chỗ

---

## 2. Phân bố file theo thư mục

| Thư mục | Số file track | Ghi chú |
|---------|---------------|---------|
| `internal/` | 3103 | Chiếm 93% repo. `internal/instagram` = 2960 file |
| `frontend/` | 118 | Vue app, **đã khá chuẩn** |
| `docs/` | 27 | `facebook/` (11, đang dùng) + `old-docs/` (18, cũ) |
| `cmd/` | 20 | Gần như toàn bộ là scratch |
| `build/` | 9 | Scaffold build của Wails (icon, manifest, installer) |
| `Config/` | 9 | Template + 1 file secret |
| `scripts/` | 8 | Có cả `__pycache__` rác |
| (gốc, không thư mục) | ~28 | Phần lộn xộn chính |

### `internal/` chi tiết

| Package | Số file | Vai trò | Tầng (theo chuẩn) |
|---------|---------|---------|-------------------|
| `instagram` | 2960 | Engine register/verify IG/FB | adapter/external (về mặt khái niệm) |
| ├─ `register/android/sXXX` | 181 package | Mỗi sXXX = 1 phiên bản API FB đã capture | adapter/external |
| ├─ `register/ios/*` | 146 package | Tương tự cho iOS | adapter/external |
| ├─ `verify/*` + `verifybase` | 624 file | Luồng verify (verifybase được 210 file import) | adapter/external |
| ├─ `fakeinfo` | 23 | Sinh thông tin/UA giả (leaf dùng chung) | domain/fakeinfo |
| └─ web, security, addinfo, interaction, feed, checkpoint... | ~40 | Feature nhỏ | adapter/external |
| `email` | 74 | Client OTP/temp-mail (rent + temp) | adapter/external |
| `igcore` | 22 | Engine IG core (có go:embed templates) | adapter/external |
| `settings` | 18 | **Đã phân tầng đẹp** (model/schema/store/validation/adapter) | mẫu tham khảo |
| `proxy` | 16 | Client proxy + check IP | adapter/external |
| `cookie` | 4 | Pool cookie (có go:embed) | adapter/repository |
| `fbdata` | 2 | DB version UA | adapter/repository |
| `httpclient`, `httpx`, `config`, `clonehv`, `runner`, `iplookup`, `stats` | mỗi cái 1 | Tiện ích/orchestration | tuỳ |

> 💡 Cấu trúc `internal/instagram` dùng **pattern plugin-registry**: package gốc định nghĩa
> interface + registry; mỗi package `sXXX` tự đăng ký vào registry trong hàm `init()`; app
> dùng *blank import* (`_ "HVRIns/internal/instagram/..."`) để kích hoạt. Có **207 dòng
> blank-import** trong các file `app*.go` ở gốc. Pattern này tránh được import cycle và rất tinh tế
> → **không được phá**.

---

## 3. Các file Go ở gốc (chi tiết)

Cả 13 file đều là **`package main`** → Go coi chúng là **một chương trình duy nhất**.
Vì vậy một method ở `app_register.go` gọi được hàm/type trong `app.go` mà không cần import.

| File | KB | Nội dung chính | Số method `*App` |
|------|----|----|----|
| `main.go` | 3.9 | Entry: `go:embed`, `AppVersion`, `wails.Run`, `Bind` | 0 |
| `app.go` | 317 | `type App struct`, `NewApp`, `startup` + 121 method + ~27 type | 121 |
| `app_register.go` | 219 | `RunRegister` + helper register | nhiều |
| `app_reg_sxxx.go` | 64 | Helper register sxxx (hàm tự do) | 0 |
| `app_verify.go` | 64 | `RunVerify` | 1 |
| `app_banclone.go` | 17 | `BancloneLogin`, `GetBancloneProducts` | 2 |
| `app_tempmail_reg.go` | 11 | Lấy temp-mail cho register/verify | 1 |
| `debug.go` | 6.5 | pprof/snapshot/memory | 3 |
| `app_getdatr.go` | 5.7 | datr-refresh qua GraphQL | 0 |
| `cpu_windows.go` | 4.6 | Đo CPU (chỉ Windows) | 0 |
| `app_test.go` | 3.0 | Test `isVerifiableAccountFile` (private) | 0 (test) |
| `datadir.go` | 1.0 | `appDataDir()` — đọc `AppVersion` | 0 |
| `portrange_windows.go` | 0.8 | `expandEphemeralPortRange()` (chỉ Windows) | 0 |

**Tổng: 129 method** gắn vào `*App`, trong đó **87 method exported** = chính là API mà
frontend Vue gọi được qua Wails.

---

## 4. `cmd/` — gần như toàn bộ là rác

`cmd/app/` **chưa tồn tại**. Entry thật nằm ở gốc (`main.go`). 18 chương trình còn lại là
**test thủ công** (in ra stdout, không assertion, hardcode đường dẫn/secret):

- **1 tool thật cần giữ**: `cmd/icongen` — sinh `build/windows/icon.ico` từ `appicon.png`;
  là nơi **duy nhất** dùng dependency `golang.org/x/image`.
- **2 stub chết**: `cmd/test_bloks_login` (main rỗng, DEPRECATED), `cmd/regtest/s23body_check.go`
  (`//go:build ignore`, chỉ là bia mộ).
- **~15 scratch còn lại**: `_testloginios`, `check_verified_email`, `emailtest` (chứa 10 tài
  khoản Hotmail + refresh token thật!), `proxycheck`, `proxytest`, `regtest`, `test273`,
  `test_eaag_flow`, `test_messios`, `test_regex`, `test_ua`, `testbody`, `testua`, `testverios`,
  `verifymess`, `verifytest`.

> Newbie note: mỗi thư mục `cmd/<x>/` là **một chương trình `package main` độc lập**, chạy bằng
> `go run ./cmd/<x>`. **Không** chương trình nào import chương trình khác (Go cấm import `package main`),
> nên **xoá chúng không thể làm hỏng phần code library**. Rủi ro duy nhất theo chiều ngược lại:
> chúng import `internal/*`, nên nếu đổi đường dẫn `internal/*` thì chúng sẽ lỗi.

---

## 5. `frontend/` — đã khá chuẩn

Tin tốt: FE **gần như đã theo chuẩn**. Khác biệt chủ yếu là **tên gọi**:

| Hiện tại | Chuẩn muốn | Việc cần làm |
|----------|------------|--------------|
| `src/bridge/` (contracts/client/mock/wails) | `src/services/` | Đổi tên (giữ nguyên độ sâu thư mục!) |
| `src/modules/` (accounts, auth-source) | `src/features/` | Đổi tên + gom page tương ứng vào |
| `src/pages/` (11 page) | `src/features/<domain>/pages/` | Gom theo feature (rủi ro cao: import tương đối) |
| `src/app/main.ts` + `src/app/App.vue` | `src/main.ts` + `src/App.vue` | Có thể giữ nguyên (đang chạy) hoặc làm phẳng |

**Điểm cần lưu ý:**
- **`src/main.ts` và `src/App.vue` hiện là STUB CHẾT** (`export {}` và `<div/>`). Entry thật là
  `src/app/main.ts`. → xoá stub.
- **192 import tương đối (`../`) trên 69 file, 0 dùng alias `@/`** dù `vite.config.ts` đã định
  nghĩa `@ → src`. → Đổi tên/di chuyển file sẽ làm hỏng hàng loạt import. **Phải bật alias `@/`
  trước khi reorg.**
- `frontend/wailsjs/` là **code Wails tự sinh** (`wails generate`), ở vị trí cố định. Các file
  `src/bridge/wails/*.ts` import nó qua `../../../wailsjs/...` → giữ nguyên độ sâu khi đổi tên.
- Đã cài `vitest` nhưng **0 file test** và `package.json` **không có script `test`**.

---

## 6. 🔴 Vấn đề bảo mật (KHẨN CẤP)

Các file sau **đang được track trong git** và **chứa credential THẬT** (đã xác nhận bằng
`git check-ignore` — chúng KHÔNG bị ignore):

| File | Nội dung nhạy cảm |
|------|-------------------|
| `Config/Cookie/cookie_initial.txt` | 48KB / 1854 dòng token `datr` phiên thật |
| `test_accounts_eaag.txt` | ~7 tài khoản FB: `c_user`, `xs`, `datr`, `fr` + token EAAG |
| `test_accounts_eaag_new.txt` | Tương tự |
| `test_accounts_fresh.txt` | Tương tự |
| `cmd/emailtest/main.go` | 10 tài khoản Hotmail + mật khẩu + OAuth refresh token hardcode |

> ⚠️ **Đừng nhầm**: `internal/cookie/embedded/cookie_initial.txt` **cũng** được track nhưng là
> **input bắt buộc của build** (`internal/cookie/store.go` dùng `//go:embed` nó). **KHÔNG được xoá file này.**
> Chỉ xoá bản ở `Config/Cookie/cookie_initial.txt`.

Hiện repo chỉ an toàn nhờ **được để PRIVATE** (comment trong `.gitignore` cũng tự thừa nhận điều này) —
rất mong manh. Cách xử lý chi tiết: [05-secrets-bao-mat.md](05-secrets-bao-mat.md).

---

## 7. Hai sự thật kỹ thuật quan trọng (ảnh hưởng cách kiểm tra)

1. **App chỉ build được trên Windows.** `cpu_windows.go` và `portrange_windows.go` cung cấp các hàm
   `getProcessCPUTime`, `expandEphemeralPortRange`, `getNumCPU` — và **không có bản thay thế** cho
   Linux/macOS. → `go build ./...` trên Linux/macOS/CI **sẽ lỗi** vì lý do không liên quan migration.
   **Mọi kiểm tra phải chạy trên Windows.**

2. **`go build ./...` trên bản clone sạch sẽ LỖI** ở bước `go:embed`. Vì `frontend/dist` bị gitignore
   (không có file nào được track), `//go:embed all:frontend/dist` không tìm thấy file → lỗ
   *"no matching files found"*. → Phải **build frontend trước** (`npm run build`) hoặc dùng thẳng
   **`wails build`** (nó tự chạy `npm run build` trước). **`wails build` là cổng kiểm tra thật sự.**

→ Đọc tiếp: [02-cau-truc-dich.md](02-cau-truc-dich.md) để xem cấu trúc đích.
