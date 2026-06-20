# Context Summary — HVRIns (Hạ Vũ)

> File ngắn gọn để người sau nạp ngữ cảnh nhanh. Đừng viết dài. Chi tiết nằm ở `docs/rebuild/`.

## Dự án là gì
Desktop app **đã hoàn chỉnh** (KHÔNG phải greenfield như `CLAUDE.md` cũ mô tả). Công cụ tự động
đăng ký/verify tài khoản Instagram/Facebook hàng loạt. Tên hiển thị "Hạ Vũ", module Go `HVRIns`.

## Stack
- **Wails v2** (gắn Go ↔ webview) · **Go** (backend) · **Vue 3 + TS** (frontend)
- Build: `wails build` (tự chạy `npm run build` trước). Versioning qua `build.bat` (`-ldflags -X main.AppVersion`).
- **Chỉ build trên Windows** (file `*_windows.go` không có bản thay thế).

## Module chính
- `internal/instagram` (2960 file) — engine register/verify, pattern **plugin-registry** (init()+blank-import). KHÔNG phá.
- `internal/{email,proxy,cookie,fbdata,igcore,settings,runner,...}` — business logic.
- Root `*.go` (13 file, `package main`) — struct `App` + 129 method (87 bind cho FE). → sẽ vào `internal/app`.
- `frontend/src` — đã khá chuẩn; có bridge layer (mock/wails), Pinia, router.

## Mục tiêu hiện tại
Tái cấu trúc theo khung **wails-go-vue/structured.md**: gốc gọn, logic vào `internal/app`, FE chia
theo feature, xử lý secret bị lộ. Kế hoạch đầy đủ: `docs/rebuild/`.

## Quy ước quan trọng (BẮT BUỘC nhớ)
1. **`main.go` Ở LẠI gốc** (không vào `cmd/app`) — vì `go:embed` cấm `../` + `wails build` giả định main ở gốc.
2. **Cổng kiểm tra = `wails build`** (không phải `go build ./...`); chạy trên **Windows**.
3. **Cú chuyển `internal/app` là MỘT commit nguyên tử** (12+ file cùng package).
4. **Không gom `*_test.go` white-box vào `tests/`** — giữ cạnh code.
5. **KHÔNG xoá** `internal/cookie/embedded/cookie_initial.txt` (go:embed, cần build). File lộ là `Config/Cookie/cookie_initial.txt`.
6. Giữ hậu tố `_windows.go` khi di chuyển.

## Phân công (xem task-board.md)
- **Dev 1 = Go/Build lead**: baseline, tách app.go, cú chuyển internal/app, bindings, AppVersion, Go tests.
- **Dev 2 = Cleanup/Infra/FE**: secrets, dọn rác non-Go, config/sample, docs, README/CLAUDE, FE restructure.
