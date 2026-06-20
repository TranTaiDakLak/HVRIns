# Dev 2 — Cleanup / Infra / Frontend

Chào bạn 👋 Bạn phụ trách dọn rác, hạ tầng, docs, và tái cấu trúc frontend.

## Bạn sở hữu vùng file nào (chỉ bạn sửa)
- `.gitignore`, `go.mod`/`go.sum` (chỉ qua `go mod tidy` ở S01), `cmd/**`, `scripts/**`, `tools/**`
- `docs/**`, `Config/**`, `config/**`, `.vscode/**`, `.kiro/**`
- `README.md`, `CLAUDE.md`, `wails.json`, `tests/**`
- `frontend/**` **TRỪ** `frontend/src/bridge/wails/*.ts` (Dev 1 sửa import binding ở Sprint 02)

## TUYỆT ĐỐI không đụng (của Dev 1)
- Root `*.go`, `main.go`, `internal/app/**`, các file Go khác trong `internal/**`.
- **Không** bắt đầu reorg `frontend/src/` cho tới khi **Dev 1 Sprint 02 = DONE** (binding đụng bridge/wails).

## Trước khi làm bất cứ gì
1. Đọc: `docs/rebuild/05-secrets-bao-mat.md` (BẮT BUỘC — bạn xử lý secrets), `01`, `03`, `02`.
2. Đọc `workspaces/pm/context-summary.md` + `current-state.md`.
3. Mọi lệnh chạy trên **Windows (PowerShell)**.

## Quy tắc DONE
- Tự test trước khi DONE; cập nhật `progress.md` + `pm/task-board.md` + `completed-log.md`.
- Kẹt → `BLOCKED`, ghi lý do.

## Các sprint của bạn
- `sprint-00-setup.md` — 🔴 secrets (làm sớm nhất, độc lập)
- `sprint-01.md` — dọn rác non-Go + config/sample (song song Dev 1)
- `sprint-02.md` — README/CLAUDE/wails.json + scaffold tests (song song Dev 1 cú chuyển)
- `sprint-03.md` — reorg frontend (CHỜ Dev 1 S02 xong)
- `sprint-04.md` — rà gốc gọn + kế hoạch history rewrite

## Cổng kiểm tra của bạn
```powershell
git status ; git check-ignore <path>     # xác minh ignore
go build ./tools/...                       # sau khi move icongen
npm --prefix frontend run build            # sau reorg FE
wails build ; wails dev                     # cổng thật (Windows)
```
