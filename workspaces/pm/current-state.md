# Current State

> Cập nhật sau MỖI lần chạy. Đây là file đọc đầu tiên ở chế độ UPDATE.

**Ngày cập nhật:** 2026-06-20
**Người cập nhật:** PM (khởi tạo workspaces)

---

## Trạng thái tổng thể
- **Giai đoạn:** Vừa khởi tạo `workspaces/`. Kế hoạch tái cấu trúc đã viết xong ở `docs/rebuild/` (9 file).
- **Code:** CHƯA di chuyển/xoá file source nào. Repo vẫn ở trạng thái trước tái cấu trúc.
- **Mốc cần đạt trước khi bắt đầu:** chạy baseline (S00-D1-T002) để có "green baseline" + số platform.

## Sprint đang chạy
**Sprint 00 — Setup & Safety** (chưa bắt đầu)

## Task đang làm
- (chưa có — chờ 2 dev nhận S00)

## Task đã xong gần nhất
- PM: khởi tạo toàn bộ `workspaces/` + chia 5 sprint cho 2 dev.

## Task đang bị BLOCK
- (không)

## Việc tiếp theo nên làm
1. **Dev 1:** S00-D1-T002 — chạy baseline (`wails build`, `go test`, ghi số platform vào file này).
2. **Dev 2:** S00-D2-T002 — xử lý secrets (`git rm --cached` + `.gitignore`) — **độc lập, làm ngay được**.

## Số liệu baseline (điền sau khi chạy S00-D1-T002)
- [ ] `wails build`: ___ (PASS/FAIL)
- [ ] `go test ./internal/...`: ___
- [ ] Số platform đăng ký (baseline): ___  ← dùng để so sánh sau Pha 3/4

## Ghi chú nhanh
- Phụ thuộc chéo quan trọng: **D2 Sprint 03 (FE rename bridge→services) phải đợi D1 Sprint 02 xong**
  (vì binding fix đụng `frontend/src/bridge/wails/*.ts`).
