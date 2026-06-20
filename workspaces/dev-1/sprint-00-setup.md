# Dev 1 — Sprint 00: Setup & Baseline

Mục tiêu: có "green baseline" đã biết để so sánh về sau. **Chưa di chuyển code nào.**

---

## S00-D1-T001 — Đọc plan & kiểm tra môi trường
**Việc:**
1. Đọc `docs/rebuild/06-go-wails-cho-newbie.md`, `00-tong-quan.md`, `02-cau-truc-dich.md`, `04-ke-hoach-thuc-thi.md`, `07-checklist-rui-ro.md`.
2. Kiểm tra công cụ:
```powershell
go version          # cần khớp go.mod (go 1.25+)
node -v ; npm -v
wails version       # nếu thiếu: go install github.com/wailsapp/wails/v2/cmd/wails@latest
```
**DONE khi:** in được version cả 3; hiểu quyết định D-001 (main.go ở gốc) và D-007 (commit nguyên tử).
**Test:** — (đọc/kiểm môi trường)

---

## S00-D1-T002 — Chạy baseline + ghi số platform
**Việc:**
```powershell
npm --prefix frontend ci
npm --prefix frontend run build      # tạo frontend/dist cho go:embed
wails build                          # CỔNG THẬT — ra build/bin/HVRIns.exe
go test ./internal/...
```
Ghi lại **số platform đăng ký** (baseline). Cách lấy:
- Cách 1: chạy app/`wails dev` xem log khởi động có in số registerer không.
- Cách 2 (chắc chắn): tạm thêm log đếm trong code đăng ký (rồi revert), hoặc đếm số package
  blank-import: `git grep -c "_ \"HVRIns/internal/instagram" -- app.go app_reg_sxxx.go`.

Ghi con số vào `workspaces/pm/current-state.md` mục "Số liệu baseline".

**DONE khi:** `wails build` PASS, `go test ./internal/...` PASS, số platform đã ghi.
**Test:** wails build = PASS; go test = PASS.

> ⚠ Nếu `go build ./...` lỗi "no matching files found" → đúng dự kiến (chưa build FE). Dùng
> `wails build`/đã `npm run build` trước. Nếu lỗi khác → BLOCKED, ghi log.

---

## S00-D1-T003 — Dọn go.mod/go.sum dirty
**Bối cảnh:** `git status` lúc bắt đầu cho thấy `M go.mod`, `M go.sum`.
**Việc:**
```powershell
git diff go.mod go.sum     # xem đổi gì
# Nếu hợp lệ:
git add go.mod go.sum ; git commit -m "chore: lock go.mod/go.sum truoc khi tai cau truc"
# Hoặc nếu không cần thay đổi:
git checkout -- go.mod go.sum
```
**DONE khi:** `git status` không còn `M go.mod/go.sum`.
**Test:** `git status` sạch ở 2 file đó.

---

### Sau khi xong Sprint 00
- Cập nhật `progress.md`, đánh DONE trên `pm/task-board.md`, thêm dòng vào `completed-log.md`.
- Báo PM cập nhật `current-state.md` (baseline xanh → mở khoá Sprint 01).
