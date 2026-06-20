# Audit Log (PM)

> PM ghi mỗi lần chạy AUDIT TOÀN BỘ (khi hết TODO/BLOCKED). Idempotent: KHÔNG tạo lại finding đã có ở đây.

## Audit #1 — 2026-06-21 (sau khi Sprint 05 + 06 DONE)

**Gate health:** `go vet ./...` PASS · `go test ./internal/...` GREEN · `npm test` 61/61 PASS · `go build ./internal/...` PASS. (wails build: PASS gần nhất tại commit của Dev 1; thay đổi sau đó chỉ là test/docs → không ảnh hưởng binary.)

**Trục đã rà:**
| Trục | Kết quả |
|------|---------|
| Build/test health | ✅ xanh (vet/test/npm) |
| Hành vi đúng (internal/result) | ✅ đã khôi phục bản gốc verbatim; không TODO/gap thật |
| Vệ sinh (secrets/dead code) | ✅ secrets không track; embedded cookie còn; root chỉ main.go |
| Cấu trúc vs docs/rebuild/02 | ✅ khớp (internal/app, features/services, tools/config/tests) |
| Docs khớp thực tế | ✅ onboarding + 08-ket-qua + project-scan đồng bộ |
| Pha 7 (internal/ layering) | ⏸️ DEFER (D-011) — rủi ro cao/lợi ích thấp, KHÔNG raise |

**Finding ĐÁNG LÀM (→ Sprint 07, hardening coverage):**
- F1: **Bridge layer Go (internal/app) gần như chưa có test** — chỉ `app_test.go` (1 nhóm). Helper thuần (parse/format/filter account, settings normalize) nên có white-box test để khóa hành vi API FE-facing. → S07-D1-T001.
- F2: **Global Pinia stores chưa test** — mới test composables + accounts store. `app.store`, `preferences.store`, `uploadLog.store` chưa có test. → S07-D2-T001.

**KHÔNG raise (tránh việc vặt / đã có chỗ khác):**
- Coverage toàn bộ ~3000 file internal/instagram: không khả thi/không đáng cho giai đoạn này.
- Rotate credential / rewrite git history / QA interactive: việc THỦ CÔNG của chủ dự án (đã ở risks.md), không phải task dev.
- ESLint/Prettier: nice-to-have, chưa xác nhận có/không config sẵn → bỏ qua lần này để khỏi tạo finding trùng.

**Kết luận:** Dự án tái cấu trúc về cơ bản HOÀN TẤT & validated. Sprint 07 là hardening tự nguyện (tăng coverage), không phải lỗi chặn.
