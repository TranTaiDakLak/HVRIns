# Dev 2 — Sprint 05: QA chức năng + Test frontend + Audit cuối

> Cấu trúc FE đã xong. Sprint này chứng minh app **chạy đúng** (QA chức năng) + thêm test thật +
> đồng bộ docs. Mọi verify trên Windows.

---

## S05-D2-T002 — Viết frontend test thật (làm TRƯỚC, độc lập)
**Bối cảnh:** hiện `npm test` chạy với `passWithNoTests=true` (0 test thật). CLAUDE.md cũ yêu cầu
tối thiểu test cho composable + accounts store.
**Việc:** viết test vitest (đặt cạnh code hoặc trong `tests/frontend/` theo `vitest.config.ts`):
- `useAccountsStore` (Pinia): khởi tạo, set danh sách, selection, filter cơ bản — dùng
  `@pinia/testing` hoặc `setActivePinia(createPinia())`.
- 1 composable: `useSelection` hoặc `useColumnVisibility` — test logic chọn/toggle cột.
- Mock service layer nếu store gọi `services/` (đã có `services/mock/`).
**Test:**
```powershell
npm --prefix frontend run test
```
**DONE khi:** có ≥2 test thật PASS (không còn dựa vào passWithNoTests).

---

## S05-D2-T001 — Chạy QA acceptance (Q1–Q12)  [CHỜ Dev 1 S05-D1-T001]
**Bối cảnh:** Q3 (import)/Q7 (reg stats)/register-verify output phụ thuộc `internal/result` đúng →
bắt đầu **sau khi** Dev 1 báo S05-D1-T001 DONE. Các mục UI thuần (Q1/Q2/Q4/Q10/Q11/Q12) có thể làm trước.
**Việc:** mở `wails dev`, chạy lần lượt 12 mục Q1–Q12 trong `workspaces/pm/qa-acceptance-plan.md`
+ 5 kiểm hồi quy RG-1..RG-5. Với mỗi mục ghi PASS/FAIL + ghi chú.
**Đặc biệt kiểm:**
- Q4 (settings persistence qua restart), Q11 (ConfirmDialog khi đóng), Q12 (second instance hiện
  cửa sổ — liên quan `app.ctx` đã bọc thành `OnSecondInstance`).
- RG-1: bản `scripts\build.bat` → `GetAppVersion()` KHÁC `"dev"`.
- RG-3: số platform == 207.
**Test:** ghi bảng kết quả vào `pm/completed-log.md` (ngày, ai chạy, PASS/FAIL từng mục).
**DONE khi:** toàn bộ Q + RG đạt; mục FAIL nào → tạo task sửa + ghi current-state (Blocker), không bỏ qua.

---

## S05-D2-T003 — Audit cấu trúc + đồng bộ docs
**Việc:**
- Đối chiếu cây thực tế với `docs/rebuild/02-cau-truc-dich.md`; ghi lệch (nếu có).
- Sửa drift đã biết: "**4 go:embed**" → **5** trong `docs/rebuild/*` + `pm/project-scan.md`.
- Thêm 1 đoạn ngắn vào `docs/rebuild/01-hien-trang.md` (hoặc README docs) ghi nhận `internal/result`
  là package tái tạo (trỏ tới quyết định D-012) để người sau biết.
- Xác nhận `.gitignore` chặn đủ runtime dirs (bin/dev, build/bin/Config runtime, result/, logs).
- Xác nhận `git ls-files` không còn secret; `internal/cookie/embedded/cookie_initial.txt` còn.
**Test:**
```powershell
git ls-files | Select-String -Pattern "test_accounts|Config/Cookie/cookie_initial" # rỗng
git ls-files internal/cookie/embedded/cookie_initial.txt                            # còn
```
**DONE khi:** docs khớp thực tế; không secret bị track; tree khớp target.

---

### Sau Sprint 05
Cập nhật `pm/task-board.md` (dòng của bạn) → `dev-2/progress.md` → `pm/completed-log.md`. Commit theo
task (KHÔNG push). Sau khi cả 2 dev xong S05: PM chốt `pm/review-checklist.md` mục "Tiêu chí hoàn thành toàn bộ".
