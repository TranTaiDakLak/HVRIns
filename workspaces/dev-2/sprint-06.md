# Dev 2 — Sprint 06: Hardening (độc lập, song song chờ Dev 1)

> Sprint 05 của bạn đã DONE. Sprint này là việc ĐỘC LẬP, KHÔNG đụng `internal/**`/`main.go`, KHÔNG
> phụ thuộc Dev 1. Mọi verify trên Windows.

---

## S06-D2-T001 — Mở rộng test coverage frontend
**Bối cảnh:** đã có test cho useSelection + useAccountsStore. Bổ sung các composable lõi của grid.
**Việc:** viết test vitest cho ≥3 composable nữa (chọn theo độ quan trọng):
- `useDataGrid` (sort, visible rows, state cơ bản)
- `useColumnVisibility` (toggle cột, persist)
- `useContextMenu` (mở/đóng, vị trí, item)
- (tuỳ chọn) `useAutoSave`, `useClipboard`
Mock `services/` nếu cần (đã có `services/mock/`).
**Test:**
```powershell
npm --prefix frontend run test
```
**DONE khi:** ≥3 composable mới có test PASS; tổng số test tăng rõ; không flaky.

---

## S06-D2-T002 — Viết `docs/onboarding.md`
**Việc:** tài liệu cho người mới (chủ dự án là newbie Go) phản ánh **cấu trúc SAU tái cấu trúc**:
- Yêu cầu môi trường (Go, Node, wails CLI) — Windows-only.
- Chạy dev: `wails dev`; build: `scripts\build.bat` / `wails build`; test: `go test ./internal/...`,
  `npm --prefix frontend run test`.
- Cây thư mục thật hiện tại (main.go gốc, `internal/app`, `internal/*`, `frontend/src/{features,services,components,composables}`, `tools/`, `scripts/`, `config/sample`, `docs/`, `tests/`).
- Quy ước: bridge→services, modules→features, alias `@/`, test white-box cạnh code.
- Trỏ tới `docs/rebuild/` cho lý do/deviation; cảnh báo secrets + repo PRIVATE.
**Test:** — (doc). **DONE khi:** doc khớp `git ls-files` thực tế, link nội bộ hợp lệ.

---

## S06-D2-T003 — Closeout doc `docs/rebuild/08-ket-qua.md`
**Việc:** tổng kết đợt tái cấu trúc:
- Đã làm: bảng Sprint 00–05 (D2) + 06, các commit chính.
- Deviation đã áp dụng (main.go ở gốc, installer giữ chỗ, Pha 7 defer...).
- **Việc còn TREO (quan trọng — ghi rõ để không quên):**
  - `internal/result` chưa validate (Dev 1 S05-D1-T001) — dispatch.go còn stub.
  - QA interactive cần chủ dự án xác nhận tay.
  - Rotate credential + rewrite history (chủ dự án).
- Trỏ `current-state.md` cho trạng thái sống.
**Test:** — (doc). **DONE khi:** doc phản ánh đúng trạng thái + danh sách treo rõ ràng.

---

### Sau Sprint 06
Cập nhật `pm/task-board.md` (dòng của bạn) → `dev-2/progress.md` → `pm/completed-log.md`. Commit theo
task (KHÔNG push). Nếu hết việc độc lập mà Dev 1 chưa xong → DỪNG, báo PM (không lấn việc Dev 1).
