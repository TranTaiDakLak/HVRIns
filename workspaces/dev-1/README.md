# Dev 1 — Go / Build Lead

Chào bạn 👋 Bạn phụ trách **critical path** phía Go & build.

## Bạn sở hữu vùng file nào (chỉ bạn sửa)
- Root `*.go`, `main.go`
- `internal/app/**` (mới tạo)
- `internal/proxy/**` và test mới trong `internal/**`
- `frontend/src/bridge/wails/*.ts` — **chỉ** sửa import binding ở Sprint 02

## TUYỆT ĐỐI không đụng (của Dev 2)
- `.gitignore`, `go.mod` (lệnh `go mod tidy` do Dev 2 chạy ở S01), `cmd/**`, `scripts/**`, `docs/**`,
  `Config/**`, `.vscode/**`, `README.md`, `CLAUDE.md`, `wails.json`, phần reorg `frontend/src/**` còn lại.

## Trước khi làm bất cứ gì
1. Đọc: `docs/rebuild/06-go-wails-cho-newbie.md` (BẮT BUỘC — bạn là newbie Go), rồi `00`, `02`, `04`, `07`.
2. Đọc `workspaces/pm/context-summary.md` + `current-state.md`.
3. Mọi lệnh chạy trên **Windows (PowerShell)**.

## Quy tắc DONE
- Tự test task trước khi đánh DONE (xem cột Test trong từng sprint).
- Cập nhật `progress.md` + `workspaces/pm/task-board.md` (status + test) + `completed-log.md`.
- Kẹt → đổi `BLOCKED`, ghi lý do vào progress.md + báo PM.

## Các sprint của bạn
- `sprint-00-setup.md` — baseline + dọn go.mod
- `sprint-01.md` — tách app.go + design note (song song Dev 2)
- `sprint-02.md` — ⭐ cú chuyển internal/app + bindings (critical path)
- `sprint-03.md` — viết Go test thay cmd scratch (song song Dev 2 reorg FE)
- `sprint-04.md` — verify cuối + quyết định Pha 7

## Cổng kiểm tra của bạn
```powershell
go vet ./...
npm --prefix frontend run build ; go build .     # nhớ build FE trước (go:embed)
go test ./internal/...
wails build                                       # cổng thật
wails dev                                         # smoke test
```
