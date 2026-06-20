# Dev 2 — Sprint 02: Docs nền tảng + scaffold tests

Chạy **song song** Dev 1 (đang làm cú chuyển internal/app). **Bạn KHÔNG đụng** root `*.go`,
`internal/app`, hay `frontend/src/bridge/wails` trong sprint này → không conflict.

---

## S02-D2-T001 — Viết README.md gốc
**Bối cảnh:** `README.md` đang rỗng (0 byte).
**Việc:** viết README gốc gồm:
- HVRIns là gì (1 đoạn), stack (Wails+Go+Vue, Windows-only).
- Yêu cầu môi trường (Go ver, Node, wails CLI).
- Cách chạy dev: `wails dev`; cách build: `scripts\build.bat` (hoặc `wails build`).
- Cây thư mục theo chuẩn (tóm tắt, trỏ tới `docs/rebuild/02-cau-truc-dich.md`).
- Ghi chú deviation: main.go ở gốc (trỏ `docs/rebuild/00`).
- Cảnh báo: repo chứa logic nhạy cảm → giữ PRIVATE; runtime data ở `appDataDir()`.
**Test:** preview markdown render OK; link tới docs hợp lệ.
**DONE khi:** README có đủ mục trên.

---

## S02-D2-T002 — Viết lại CLAUDE.md + author wails.json
**Bối cảnh:** `CLAUDE.md` hiện mô tả "frontend foundation, chưa có backend" — SAI thực tế (D-001 context).
**Việc:**
- Viết lại `CLAUDE.md` mô tả **app thật**: backend Go ở `internal/` (sắp có `internal/app`), entry
  `main.go` ở gốc (giải thích deviation), FE Vue ở `frontend/` chia theo feature, build bằng wails,
  Windows-only. Nêu quy ước quan trọng (trỏ `workspaces/pm/context-summary.md`).
- `wails.json`: điền `author.name` + `author.email` (hỏi chủ dự án nếu cần; nếu chưa có, để placeholder
  hợp lý + ghi Notes).
**Test:** — (docs/config). **DONE khi:** CLAUDE.md không còn mâu thuẫn thực tế; wails.json hợp lệ JSON (`wails build` vẫn chạy).

> ⚠ Đừng xoá `CLAUDE.md` — tooling kỳ vọng nó ở gốc. Chỉ viết lại nội dung.

---

## S02-D2-T003 — Scaffold tests/{go,frontend}
**Việc:**
```powershell
New-Item -ItemType Directory -Force tests/go, tests/frontend | Out-Null
```
- `tests/go/README.md`: giải thích chỉ chứa test **black-box** (`package xxx_test`); test white-box
  giữ cạnh code (trỏ `docs/rebuild/02` mục 3).
- `tests/frontend/README.md`: nơi đặt test vitest tương lai; nhắc `vitest.config.ts` đã include `tests/**`.
- Thêm `.gitkeep` nếu thư mục trống.
**Test:** `git status` thấy thư mục mới. (Chưa thêm `"test"` script — để Sprint 03 cùng FE.)
**DONE khi:** scaffold + README giải thích rõ quy tắc test.

---

### Sau Sprint 02
Cập nhật progress + task-board + completed-log. **CHỜ Dev 1 báo S02 DONE** trước khi bắt Sprint 03
(reorg FE đụng `frontend/src/bridge` mà Dev 1 vừa sửa binding).
