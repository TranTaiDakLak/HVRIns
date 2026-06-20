# Onboarding — HVRIns

> Hướng dẫn cho người mới vào dự án. Phản ánh cấu trúc **sau tái cấu trúc Sprint 00–06**.
> Xem `docs/rebuild/` để hiểu lý do từng quyết định.

---

## 1. Yêu cầu môi trường

| Công cụ | Phiên bản đề xuất | Ghi chú |
|---------|-------------------|---------|
| **Windows** | 10/11 (64-bit) | **Bắt buộc** — `cpu_windows.go`/`portrange_windows.go` không có bản Linux/macOS |
| **Go** | 1.22+ | `go version` |
| **Node.js** | 20+ | `node -v` |
| **npm** | 10+ | `npm -v` |
| **Wails CLI** | v2.x | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |

> ⚠️ **Windows-only**: `go build ./...` và `wails build` chỉ chạy trên Windows. Đừng cố build trên Linux/macOS — sẽ lỗi do thiếu `cpu_windows.go`.

---

## 2. Chạy và build

### Dev mode (hot-reload)
```powershell
wails dev
```
Mở cửa sổ "Hạ Vũ". Frontend reload nhanh; backend Go cần restart khi sửa file `.go`.

### Build production
```powershell
# Cách 1 (khuyến nghị — tự điền AppVersion)
scripts\build.bat

# Cách 2 (thủ công)
wails build
```
Output: `build/bin/HVRIns.exe` (hoặc `HVRIns-installer.exe` nếu dùng NSIS).

> `go build ./...` **KHÔNG DÙNG** làm cổng kiểm tra — lỗi `go:embed` khi `frontend/dist` chưa tồn tại.

---

## 3. Chạy test

```powershell
# Go tests
go test ./internal/...

# Frontend tests (vitest)
npm --prefix frontend run test

# Build tools (icongen)
go build ./tools/...
```

Baseline: `go test ./internal/...` có 2 pre-existing fail không chặn build:
- `verifybase.TestCheckLiveDieCombined_RealAccounts` — live account test (cần tài nguyên thật)
- `fakeinfo` — data tests từ repo gốc

---

## 4. Cây thư mục (sau tái cấu trúc)

```
HVRIns/
├── main.go                     # Entry Wails — GIỮ ở gốc (go:embed không dùng ../)
├── wails.json  go.mod  go.sum  # Config Wails + Go module
├── CLAUDE.md  README.md        # Hướng dẫn dự án
│
├── internal/
│   ├── app/                    # ⭐ Logic App (package app) — sau tái cấu trúc Sprint 02
│   │   ├── app.go  app_register.go  app_verify.go  ...
│   │   └── *_windows.go        # Build constraint — giữ hậu tố!
│   ├── instagram/              # Engine đăng ký/verify (2960 file, plugin-registry)
│   │   ├── register/android/sXXX/  # Mỗi sXXX = 1 phiên bản API (blank-import trong app.go)
│   │   ├── register/ios/
│   │   ├── verify/             # 624 file, verifybase import bởi 210 file khác
│   │   └── fakeinfo/           # Sinh thông tin giả (UA, phone code)
│   ├── result/                 # Output format/dispatch — khôi phục từ bản gốc HVR
│   ├── cookie/                 # Pool cookie (go:embed embedded/)
│   ├── igcore/                 # Engine IG core (go:embed templates/)
│   ├── email/                  # Client OTP/temp-mail
│   ├── proxy/                  # Client proxy + check IP
│   ├── settings/               # Đã phân tầng: model/schema/store/validation/adapter
│   ├── runner/  stats/  config/  httpclient/ ...  # Tiện ích
│   └── fbdata/                 # DB version UA
│
├── frontend/src/
│   ├── app/                    # Entry: main.ts, App.vue, router/
│   ├── features/               # Feature modules (sau tái cấu trúc Sprint 03)
│   │   ├── accounts/           # Grid tài khoản: pages/, store/, components/
│   │   ├── auth-source/        # Nguồn xác thực
│   │   ├── reg-stats/          # Thống kê đăng ký
│   │   └── settings/           # Cài đặt chung, proxy, interaction
│   ├── services/               # Layer cách ly FE ↔ Wails (thay bridge/ Sprint 03)
│   │   ├── contracts.ts        # Interface định nghĩa API
│   │   ├── client.ts           # getAccountService() — trả Wails hoặc mock
│   │   ├── wails/              # Wrapper Wails thật (import wailsjs/go/app/)
│   │   └── mock/               # Mock để test/dev offline
│   ├── composables/            # Logic UI tái sử dụng (useSelection, useDataGrid, ...)
│   ├── stores/                 # Pinia stores (useAccountsStore, usePreferencesStore)
│   ├── components/             # UI components (ui/, grid/, form/, feedback/, shell/)
│   ├── pages/                  # Các page không thuộc feature riêng
│   ├── constants/              # Định nghĩa columns, enum
│   └── types/                  # TypeScript types dùng chung
│
├── tools/icongen/              # Sinh icon .ico từ appicon.png (go build ./tools/...)
├── scripts/                    # build.bat, scripts/legacy/
├── config/sample/              # Template config (KHÔNG phải data thật)
│   ├── Proxy/proxy_*.example.txt
│   ├── TempMail/domains.example.txt
│   ├── Permanent/mail.example.txt  phone.example.txt
│   └── DeviceInfo/versions_*.example.txt
├── docs/                       # Tài liệu (rebuild/, flows/, archive/, testing/)
├── tests/                      # Black-box test (go/, frontend/) — white-box ở cạnh code
└── build/                      # Scaffold Wails (icon, manifest, installer NSIS)
```

---

## 5. Quy ước quan trọng

### 5.1 Import frontend — luôn dùng alias `@/`
```ts
// ✅ Đúng
import { useAccountsStore } from '@/features/accounts/store/useAccountsStore'

// ❌ Sai — relative import dễ hỏng khi move file
import { useAccountsStore } from '../../features/accounts/store/useAccountsStore'
```

Ngoại lệ duy nhất: `wailsjs/` import giữ relative vì Wails generate code tại `frontend/wailsjs/`.

### 5.2 Gọi backend — PHẢI qua `services/`
```ts
// ✅ Đúng — qua layer services/
import { getAccountService } from '@/services/client'
const svc = await getAccountService()
await svc.list(filter)

// ❌ Sai — gọi thẳng Wails binding từ component
import { ListAccounts } from '../../../wailsjs/go/app/App'
```
Lý do: `services/client.ts` xử lý timeout Wails, fallback mock, và dễ test.

### 5.3 Test — white-box cạnh code, black-box vào `tests/`
```
internal/app/app_test.go        ← white-box (package app) — ở cạnh code
tests/go/                       ← black-box (package app_test) — riêng thư mục
frontend/src/**/*.test.ts       ← vitest (bất kỳ thư mục src/ nào)
```

### 5.4 Files `_windows.go` — KHÔNG xoá hậu tố
`cpu_windows.go`, `portrange_windows.go` dùng hậu tố làm build constraint. Khi di chuyển file, giữ nguyên tên.

### 5.5 `internal/result/` — cẩn thận khi sửa
Package này được khôi phục từ bản gốc `HVR` (repo tổ tiên). Format tên file output của register/verify phụ thuộc vào constant ở đây.

---

## 6. Cảnh báo bảo mật

> **Repo phải PRIVATE** — credentials đã lộ trong git history (chưa rewrite).

- `internal/cookie/embedded/cookie_initial.txt` — **CẤM XOÁ** (go:embed bắt buộc)
- Runtime data (`Config/Cookie/`, `Config/Proxy/`, `Config/TempMail/`, ...) — đã gitignore, không commit
- Credential thật (FB, Hotmail) còn trong git history → cần rotate và rewrite history (xem `workspaces/pm/risks.md`)

---

## 7. Trỏ đọc thêm

| Tài liệu | Nội dung |
|----------|---------|
| `docs/rebuild/01-hien-trang.md` | Phân tích hiện trạng ban đầu |
| `docs/rebuild/02-cau-truc-dich.md` | Cấu trúc đích sau tái cấu trúc |
| `docs/rebuild/06-go-wails-cho-newbie.md` | Giải thích Go/Wails cho người mới |
| `docs/rebuild/08-ket-qua.md` | Tổng kết đợt tái cấu trúc |
| `workspaces/pm/decision-log.md` | Mọi quyết định kỹ thuật + lý do |
| `workspaces/pm/risks.md` | Rủi ro còn treo (secrets, history) |
| `CLAUDE.md` | Hướng dẫn cho Claude Code |
