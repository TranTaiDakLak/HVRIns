# Dev 1 — Sprint 04: Verify cuối & quyết định Pha 7

---

## S04-D1-T001 — Verify end-to-end
**Việc:** chạy đầy đủ cổng kiểm tra trên Windows sau khi cả D1 & D2 đã xong S03:
```powershell
go vet ./...
go test ./internal/...
npm --prefix frontend run build
wails build            # ra build/bin/HVRIns.exe
wails dev              # smoke: mở app, thao tác vài chức năng (accounts, settings, register dryrun nếu có)
```
Kiểm: `GetAppVersion()` đúng version; **số platform == baseline** (S00).
**Test:** tất cả lệnh trên PASS.
**DONE khi:** xanh hết + đã cập nhật `pm/task-board.md`, `completed-log.md`, `current-state.md`.

---

## S04-D1-T002 — Quyết định Pha 7 (internal/ deep mapping)
**Việc:** đánh giá có nên làm ánh xạ `internal/` → `domain/usecase/adapter` hay không.
- Khuyến nghị mặc định: **DEFER** (rủi ro cao, lợi ích thấp — xem D-004).
- Ghi quyết định cuối + lý do vào `pm/decision-log.md` (D-0xx).
- Nếu DEFER: tạo 1 issue/ghi chú "Pha 7 — backlog" trong `pm/risks.md` hoặc `current-state.md`
  để lần sau biết còn việc tuỳ chọn này.
- Nếu LÀM: tạo sprint mới (`sprint-05.md`) chi tiết theo `docs/rebuild/04` Pha 7 — di chuyển theo
  **khối** (register/, verify/), dùng công cụ refactor, verify platform count sau mỗi khối.

**Test:** — (quyết định). **DONE khi:** decision-log có dòng quyết định Pha 7.

---

### Hoàn tất dự án (phối hợp Dev 2)
Khi cả 2 dev xong S04: PM chạy `pm/review-checklist.md` mục "Tiêu chí hoàn thành toàn bộ".
