# Current State

> Cập nhật sau MỖI lần chạy. Đây là file đọc đầu tiên ở chế độ UPDATE.

**Ngày cập nhật:** 2026-06-21
**Người cập nhật:** PM (review cuối Sprint 00–04 + giao Sprint 05)

---

## Trạng thái tổng thể
- **Sprint 00–04: HOÀN TẤT** — 34/34 task DONE (+1 SKIP có chủ đích: S03-D1-T002 cross-platform stub, D-010).
- **Cấu trúc tái cấu trúc đã xong và verify (PM review 2026-06-21):**
  - Gốc repo chỉ còn `main.go` + file/thư mục chuẩn ✅
  - `internal/app/` 19 file `package app` (App + 121 method tách nhỏ) ✅
  - Frontend: `bridge/→services/`, `modules/→features/`, alias `@/`, stub đã xoá ✅
  - Secrets gỡ khỏi track; `internal/cookie/embedded/cookie_initial.txt` còn nguyên ✅
  - `wails build` PASS (commit a3d8210); 207 platform đăng ký == baseline ✅
- **git status: sạch.** Pha 7 (internal/ deep mapping): DEFER (D-011).

## ⚠️ PHÁT HIỆN QUAN TRỌNG khi review (cần Sprint 05 xử lý)
- **`internal/result` là package TÁI TẠO bằng suy luận** — nó vốn bị import nhưng CHƯA TỪNG có trong
  repo (repo gốc không build được). Dev 1 đã dựng lại để unblock (commit a3d8210). Hệ quả:
  - `writer.go`, `counter.go`: implement thật, ổn.
  - `format.go` (`FormatReg`/`FormatVerify`): **thứ tự field + format SUY LUẬN** — có thể lệch bản gốc
    → file output (`SuccessReg.txt`...) có thể sai định dạng so với consumer.
  - `files.go`: **tên file constant SUY LUẬN** — phải khớp tên mà chỗ ĐỌC mong đợi.
  - `dispatch.go`: **STUB trả `nil`** — logic ghi detail-file theo sub-status đang MẤT.
  - → `wails build` xanh KHÔNG đảm bảo hành vi đúng. **Phải validate ở Sprint 05.**
- Còn **1 test fail ở verifybase** (Dev 1 ghi "pre-existing, live account state") — cần xác nhận/skip.

## Sprint đang chạy
**Sprint 05 — Validation, QA & Hardening** (vừa giao — xem task-board + sprint-plan)

## Task đang làm
- (chưa bắt đầu — chờ 2 dev nhận Sprint 05)

## Task đã xong gần nhất
- Toàn bộ Sprint 00–04 (D1 + D2). PM: review + giao Sprint 05.

## Task đang bị BLOCK
- (không) — blocker `internal/result` cũ đã được Dev 1 unblock; giờ chuyển thành task VALIDATE.

## Việc tiếp theo nên làm
1. **Dev 1:** S05-D1-T001 — validate/khôi phục `internal/result` (ưu tiên cao nhất).
2. **Dev 2:** S05-D2-T002 — viết frontend test thật (độc lập, làm song song ngay được).
3. **Dev 2:** S05-D2-T001 — chạy QA acceptance (Q1–Q12) **sau khi** Dev 1 xong S05-D1-T001.

## ❓ Cần CHỦ DỰ ÁN quyết
1. **Source gốc `internal/result`?** Có ở máy khác (E:/WEMAKE/NullCoreSummer...)/backup không? Nếu có →
   khôi phục bản gốc thay vì tin bản suy luận. (Ảnh hưởng trực tiếp S05-D1-T001.)
2. **Rotate credential** đã lộ (FB/Hotmail) — việc thủ công (xem risks.md, vẫn TODO).
3. **Rewrite git history** cho secrets (kế hoạch đã có ở risks.md) — cần đồng thuận team.

## Số liệu baseline
- `wails build`: PASS (a3d8210)
- `go test ./internal/...`: hầu hết PASS, 1 fail verifybase (live account state — cần xác nhận S05-D1-T002)
- Số platform đăng ký: **207** (blank-import) — dùng so sánh hồi quy

## Ghi chú
- 5 `go:embed` (không phải 4 như doc ban đầu): main.go, cookie/store.go, igcore/template.go,
  iosmess/embed.go, + 1 nữa (Dev 1 xác nhận). Cần đồng bộ docs ở S05-D2-T003.
