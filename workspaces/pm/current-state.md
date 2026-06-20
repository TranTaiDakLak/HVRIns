# Current State

> Cập nhật sau MỖI lần chạy. Đây là file đọc đầu tiên ở chế độ UPDATE.

**Ngày cập nhật:** 2026-06-20
**Người cập nhật:** Dev 1

---

## Trạng thái tổng thể
- **Giai đoạn:** Sprint 00 đang chạy. Dev 2 đã xong S00 (T001/T002/T003). Dev 1 đang BLOCKED ở S00-D1-T002.
- **Code:** CHƯA di chuyển/xoá file source nào. Repo ở trạng thái trước tái cấu trúc.
- **⚠️ BLOCKER PHÁT HIỆN:** Package `internal/result` không tồn tại trong repo nhưng được import bởi 3 file gốc (app.go, app_register.go, app_verify.go). Toàn bộ build fail vì lý do này.

## Sprint đang chạy
**Sprint 00 — Setup & Safety**

## Task đang làm
- S00-D1-T002: BLOCKED (internal/result missing)
- S00-D1-T003: TODO (đang chờ)

## Task đã xong gần nhất
- S00-D1-T001: DONE (môi trường OK)
- S00-D2-T001/T002/T003: DONE

## Task đang bị BLOCK
- **S00-D1-T002 BLOCKED**: Package `HVRIns/internal/result` không tồn tại.
  - app.go:34, app_register.go:163, app_verify.go:14 đều import `resultpkg "HVRIns/internal/result"`
  - Hậu quả: `wails build` FAIL · `wails generate module` FAIL · `npm run build` FAIL (wailsjs/runtime không được sinh)
  - Cần quyết định: (A) Dev 1 tạo package từ API đã suy ra hoàn chỉnh, hoặc (B) user cung cấp source

## Số liệu baseline (điền sau khi chạy S00-D1-T002)
- [ ] `wails build`: **FAIL** (internal/result missing)
- [ ] `go test ./internal/...`: mostly PASS; 1 fail verifybase (live account state)
- [ ] Số platform đăng ký (baseline): ___ (chưa lấy được)

## Ghi chú nhanh
- Phụ thuộc chéo quan trọng: **D2 Sprint 03 (FE rename bridge→services) phải đợi D1 Sprint 02 xong**
  (vì binding fix đụng `frontend/src/bridge/wails/*.ts`).
- go.mod dirty = wails upgrade 2.11→2.12 (hợp lệ, D1 sẽ commit ở S00-D1-T003 sau khi unblock).
