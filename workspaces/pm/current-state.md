# Current State

> Cập nhật sau MỖI lần chạy. Đây là file đọc đầu tiên ở chế độ UPDATE.

**Ngày cập nhật:** 2026-06-21
**Người cập nhật:** PM (review Sprint 05 + giao Sprint 06)

---

## Trạng thái tổng thể
- **Sprint 00–04: HOÀN TẤT** (34 DONE + 1 SKIP). Cấu trúc tái cấu trúc đạt chuẩn, đã verify.
- **Sprint 05:**
  - **Dev 2: DONE 3/3** ✅ (verify bằng repo): 30 FE test thật (useSelection 17 + useAccountsStore 13,
    commit 1d0f0c8); docs đồng bộ (5 go:embed, note internal/result — 2f9057d); QA acceptance chạy (5e5e107).
  - **Dev 1: DONE 3/3** ✅ (2026-06-21): T001 restore internal/result (f50bba2); T002 fix 16 test fail
    → suite GREEN (verifybase gate RUN_LIVE_TESTS c559808 + fakeinfo skipIfNoConfigData/UA path fix 716d0d3);
    T003 result_test.go khóa hành vi (e0b3031). `go test ./internal/...` GREEN · `wails build` PASS.

## ✅ GAP R-16 — ĐÃ ĐÓNG (S05-D1-T001, 2026-06-21)
- **`internal/result` đã KHÔI PHỤC từ BẢN GỐC.** Chủ dự án chỉ `D:\Github\HVR\` → tìm thấy bản gốc đầy đủ
  ở `D:\Github\HVR\HVR\internal\result` (module `HVR`, repo tổ tiên). Đã thay bản tái tạo bằng bản gốc verbatim.
  - Sửa 3 filename SAI: `Die.txt` (was DieAfterVerify.txt), `Unknown.txt` (was UnknownError_CheckLiveDie.txt),
    `UnknownReg.txt` (was UnknownBlock.txt) — khớp consumer hardcode (`app_register.go:4536`, `verify/web/verify.go:616-618`).
  - `dispatch.go` từ STUB nil → đầy đủ (15+ detail-file). Bổ sung `counters.go`/`errorlog.go`/`store.go` + test gốc.
  - Validate workflow 5-lens: byte-for-byte giống HVR (drift), consumer khớp (autoDetect theo pattern), 0 critical.
  - Gate: `go build .`/`go test ./internal/result/...`/`go vet`/`wails build` PASS. Xem D-012.
  - ⚠️ Output register/verify thật giờ ghi đúng tên file gốc; nếu Dev 2 QA trước restore (trên bản tái tạo)
    thì nên re-check output-file khi tiện (UI QA không bắt được phần này).

## ⚠️ Caveat về QA (S05-D2-T001)
- Dev 2 đánh "Q1–Q12 PASS", nhưng phần **QA tương tác GUI** (mở cửa sổ, click, restart) một agent
  **không thể tự click** → các mục interactive cần **chủ dự án xác nhận tay** hoặc một e2e harness thật.
  Phần automated/static (section 3) thì tin được. Coi QA interactive là "chưa xác nhận đầy đủ".
- Lưu ý: validate output register/verify **không** thể làm bằng QA UI (cần live resource hoặc test
  có chủ đích) → phụ thuộc S05-D1-T001 + quyết định của chủ dự án.

## Sprint đang chạy
**Sprint 05 + 06: TẤT CẢ task Dev 1 + Dev 2 DONE.** Còn treo (ngoài scope dev): QA interactive GUI cần
chủ dự án click tay; re-check output register/verify sau restore; rotate creds + rewrite git history (thủ công).

## Task đang làm / kế tiếp
1. **Dev 1 — ƯU TIÊN:** hoàn tất Sprint 05 (S05-D1-T001 → T002 → T003). Đây là bottleneck của cả dự án.
2. **Dev 2:** Sprint 06 (độc lập, không phụ thuộc Dev 1): mở rộng FE test + onboarding doc + closeout.

## Task đang bị BLOCK
- (không cứng) — nhưng "validate output register/verify" treo chờ S05-D1-T001 + quyết định chủ dự án.

## ✅ Quyết định chủ dự án (2026-06-21)
1. **`internal/result` → KHÔI PHỤC BẢN GỐC** (cập nhật): chủ dự án chỉ `D:\Github\HVR\` → bản gốc CÓ THẬT ở
   `D:\Github\HVR\HVR`. Dev 1 đã khôi phục verbatim (thay vì tự suy luận). Validated, không còn behavior-gap.
   (Quyết định cũ "tự suy luận, không chờ source" bị thay thế vì source đã được chủ dự án cung cấp.)

### Còn treo (thủ công, không chặn dev)
- **Rotate credential** đã lộ (FB/Hotmail) — risks.md, vẫn TODO.
- **Rewrite git history** cho secrets — cần đồng thuận team (kế hoạch ở risks.md).

## Số liệu baseline (giữ để so hồi quy)
- `wails build`: PASS (a3d8210) · Số platform: **207** · `go test ./internal/...`: 1 fail verifybase (chưa xử lý — S05-D1-T002)

## Ghi chú
- Git: chỉ branch `main`, 1 worktree. Mọi thay đổi đều trên main.
