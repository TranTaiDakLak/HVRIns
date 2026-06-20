# HVRIns — Hạ Vũ Instagram/Facebook Tool

> ⚠️ **PRIVATE REPO** — Chứa logic đăng ký/verify tài khoản mạng xã hội. Không chia sẻ công khai.

## Là gì

Desktop app Windows tự động đăng ký và xác minh tài khoản Instagram/Facebook hàng loạt.
Backend Go xử lý HTTP/automation; frontend Vue 3 hiển thị trạng thái và điều khiển.

**Stack:** Wails v2 · Go · Vue 3 + TypeScript · Windows-only

## Yêu cầu môi trường

| Tool | Phiên bản tối thiểu |
|------|---------------------|
| Go | 1.22+ |
| Node.js | 18+ |
| npm | 9+ |
| Wails CLI | v2.x (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`) |
| Windows | 10/11 (app dùng `*_windows.go`, không có bản thay thế) |

## Cách chạy

```powershell
# Dev mode (hot reload)
wails dev

# Build release
scripts\build.bat          # inject version tự động theo timestamp
# hoặc
wails build
```

> `scripts\build.bat` tự `cd` về gốc repo trước khi gọi `wails build`.

## Cây thư mục

```
HVRIns/
├── main.go                  # Entry point (ở gốc — xem ghi chú deviation bên dưới)
├── internal/
│   ├── app/                 # App struct + 87 method bind cho FE (đang tái cấu trúc)
│   ├── instagram/           # Engine đăng ký/verify (plugin-registry pattern)
│   └── ...                  # email, proxy, cookie, settings, runner, ...
├── frontend/
│   ├── src/
│   │   ├── bridge/          # Layer cách ly FE khỏi Wails binding trực tiếp
│   │   ├── modules/         # Các feature module (accounts, flow-settings, ...)
│   │   └── ...
│   └── wailsjs/             # Auto-generated Wails bindings (commit để không cần generate lại)
├── tools/icongen/           # Tool tạo icon (không phải app code)
├── scripts/                 # build.bat + legacy scripts
├── config/sample/           # Template file cấu hình (không chứa data thật)
├── docs/
│   ├── rebuild/             # Kế hoạch tái cấu trúc hiện tại (9 file)
│   ├── flows/               # Tài liệu luồng Facebook/Instagram
│   └── ...
└── workspaces/              # Sprint plan + task tracking cho 2 dev
```

Chi tiết đầy đủ: [docs/rebuild/02-cau-truc-dich.md](docs/rebuild/02-cau-truc-dich.md)

## Ghi chú deviation quan trọng

**`main.go` ở gốc repo** (không trong `cmd/app/`) vì `//go:embed all:frontend/dist` cấm dùng
đường dẫn `../`; `wails build` cũng giả định main package ở thư mục gốc. Xem chi tiết:
[docs/rebuild/00-tong-quan.md](docs/rebuild/00-tong-quan.md)

## Runtime data

App đọc/ghi data vào `appDataDir()` (thường `%APPDATA%\HVRIns\` hoặc thư mục được set bởi
`HVRINS_DATA_DIR`). **Không** đọc thư mục `Config/` ở gốc repo khi chạy.

Template cấu hình: `config/sample/` — copy sang `appDataDir/Config/` khi cần.

## Bảo mật

- Runtime data (cookie, proxy, mail) **không được commit** (đã thêm vào `.gitignore`).
- Credential thật phải được rotate — xem [workspaces/pm/risks.md](workspaces/pm/risks.md).
