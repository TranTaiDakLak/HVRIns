# CLAUDE.md

> Đây là hướng dẫn cho Claude Code khi làm việc với repo này.
> Đọc thêm: `workspaces/pm/context-summary.md` để nắm ngữ cảnh tái cấu trúc.

## App này là gì

**HVRIns** — desktop app **đã hoàn chỉnh** (KHÔNG phải greenfield). Công cụ tự động đăng ký và
xác minh tài khoản Instagram/Facebook hàng loạt. Tên hiển thị "Hạ Vũ".

**Stack:** Wails v2 · Go (backend) · Vue 3 + TypeScript (frontend) · **Windows-only**

## Cấu trúc quan trọng

```
main.go                  # Entry point — PHẢI ở gốc (xem Deviation D-001 bên dưới)
internal/
  app/                   # App struct + bind methods cho Wails FE (đang tái cấu trúc)
  instagram/             # Engine đăng ký/verify — 2960 file, plugin-registry pattern
  {email,proxy,cookie,settings,runner,...}  # Business logic
frontend/src/
  bridge/                # Layer cách ly FE khỏi Wails binding (mock/ và wails/)
  modules/               # Feature modules: accounts, flow-settings, proxy-settings, view-settings
tools/icongen/           # Build tool, không phải app code
config/sample/           # Template config (không chứa credential thật)
workspaces/              # Sprint plan + task board (2 dev đang tái cấu trúc)
```

## Deviation quan trọng (ghi nhớ)

| # | Quyết định | Lý do |
|---|-----------|-------|
| D-001 | `main.go` ở lại gốc (không vào `cmd/app/`) | `//go:embed all:frontend/dist` cấm `../`; wails build giả định main ở gốc |
| D-002 | Logic App vào `internal/app` (package app) | Tách logic khỏi entry point |
| D-005 | `*_test.go` white-box giữ cạnh code; embedded cookie KHÔNG XOÁ | Go visibility rules; file go:embed bắt buộc |

## Cổng kiểm tra (Windows-only)

```powershell
wails build          # build thật — cổng chính
wails dev            # dev mode
go build ./tools/... # kiểm tra tools/icongen riêng
```

> `go build ./...` lỗi trên cây sạch (thiếu `frontend/dist` cho go:embed). Dùng `wails build`.

## Quy tắc khi sửa code

- **KHÔNG xoá** `internal/cookie/embedded/cookie_initial.txt` — go:embed, cần build
- **KHÔNG gọi trực tiếp** Wails-generated binding từ component/page — phải qua `frontend/src/bridge/`
- **KHÔNG commit** runtime data: cookie, proxy, mail (đã `.gitignore`)
- Hậu tố `_windows.go` phải giữ khi di chuyển file
- Tái cấu trúc `internal/app` là **1 commit nguyên tử** (12+ file cùng package)

## Phân công tái cấu trúc (đang chạy)

- **Dev 1** — Go/Build: tách app.go, cú chuyển internal/app, bindings, AppVersion, Go tests
- **Dev 2** — Cleanup/Infra/FE: secrets, dọn rác, docs, README, FE restructure

Chi tiết task: `workspaces/pm/task-board.md`

## Bảo mật

Repo phải **PRIVATE**. Credential thật đã được gỡ khỏi HEAD (S00-D2-T002) nhưng còn trong
git history — cần rotate. Xem `workspaces/pm/risks.md`.
