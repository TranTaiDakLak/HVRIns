# Conflict Matrix — ai sửa file nào (chống đụng)

> Mục đích: 2 dev làm song song mà KHÔNG sửa trùng file → ít merge conflict. Nếu một file không
> nằm trong vùng của bạn, ĐỪNG sửa — báo PM nếu cần.

## 1. Quy tắc sở hữu theo đường dẫn

| Đường dẫn / pattern | Chủ sở hữu | Sprint chạm vào | Ghi chú |
|---------------------|-----------|-----------------|---------|
| `main.go` | **Dev 1** | S02 | Bootstrap mỏng |
| root `app*.go`, `datadir.go`, `debug.go`, `*_windows.go`, `app_test.go` | **Dev 1** | S01 (tách), S02 (move) | Chuyển vào internal/app |
| `internal/app/**` | **Dev 1** | S02, S03 | Mới tạo |
| `internal/proxy/**`, test mới trong `internal/**` | **Dev 1** | S03 | Unit test |
| `internal/settings/adapter/legacy.go` | **Dev 1** | S03 | Rà import cycle |
| `frontend/src/services\|bridge/wails/*.ts` | **Dev 1 (S02)** → **Dev 2 (S03)** | sequential | D1 sửa import binding; D2 đổi tên thư mục SAU |
| `.gitignore` | **Dev 2** | S00, S01 | Mọi sửa .gitignore là Dev 2 |
| `go.mod` / `go.sum` | **Dev 1 (S00 dọn dirty)** + **Dev 2 (S01 tidy)** | xem mục 3 | Cần điều phối |
| `cmd/**`, `tools/**` | **Dev 2** | S01 | Dọn + move icongen |
| `scripts/**` | **Dev 2** | S01 | build.bat, legacy |
| `docs/**` | **Dev 2** | S01 | Gom docs |
| `Config/**`, `config/**` | **Dev 2** | S01 | → config/sample |
| `.vscode/**`, `.kiro/**` | **Dev 2** | S01 | launch.json, specs |
| `README.md`, `CLAUDE.md`, `wails.json` | **Dev 2** | S02 | Viết lại |
| `tests/**` | **Dev 2** | S02 | Scaffold |
| `frontend/**` (trừ bridge/wails) | **Dev 2** | S03 | Reorg FE |
| `workspaces/pm/*` (trừ shared bên dưới) | **PM** | — | Dev chỉ đọc |
| `workspaces/dev-1/*` | Dev 1 (+PM tạo) | — | progress.md do Dev 1 ghi |
| `workspaces/dev-2/*` | Dev 2 (+PM tạo) | — | progress.md do Dev 2 ghi |

## 2. File DÙNG CHUNG (cả 2 dev + PM ghi) — giao thức "append rows"

Các file này nhiều người ghi → **chỉ sửa DÒNG/Ô của mình**, đừng viết lại cả file:
- `pm/task-board.md` — mỗi dev chỉ đổi `Status`/`Test` ở **các dòng task của mình** (S__-D1-* hoặc S__-D2-*).
- `pm/completed-log.md` — **thêm dòng mới** dưới đúng mục Sprint, không sửa dòng người khác.
- `pm/current-state.md` — ưu tiên PM cập nhật; dev chỉ thêm số liệu của mình (vd baseline) ở ô trống.
- `pm/decision-log.md`, `pm/risks.md` — **thêm dòng mới** (D-009, R-16…), không sửa dòng cũ.

> Nếu phải sửa file dùng chung khi dev kia có thể đang sửa: commit nhỏ, sửa đúng ô, kéo (pull/rebase)
> trước khi push. Conflict ở các file `.md` này dễ giải quyết thủ công.

## 3. Điểm cần điều phối go.mod/go.sum
- **S00-D1-T003**: Dev 1 commit/revert trạng thái dirty ban đầu của go.mod/go.sum.
- **S01-D2-T004**: Dev 2 chạy `go mod tidy` (sau khi move icongen).
- ⚠ Nếu chạy song song: Dev 2 đợi Dev 1 commit S00 xong rồi mới `go mod tidy`, để tidy chạy trên
  go.mod đã sạch. Ghi trạng thái vào current-state để dev kia biết.

## 4. Sync points (mốc đồng bộ bắt buộc)
| # | Điều kiện | Mở khoá |
|---|-----------|---------|
| SYNC-1 | Dev 1 **S00 baseline DONE** (build xanh + số platform ghi) | Cả 2 dev bắt đầu S01 |
| SYNC-2 | Dev 1 **S02 DONE** (App→internal/app + bindings + import bridge/wails đã sửa) | Dev 2 được bắt **S03** (reorg FE) |
| SYNC-3 | Cả 2 **S03 DONE** | Cả 2 vào S04 finalize |

> Dev 2 có thể làm S00→S02 song song thoải mái (vùng file tách biệt). Chỉ **S03 của Dev 2 mới phụ
> thuộc S02 của Dev 1** (vì cùng đụng `frontend/src/.../wails`).

## 5. Khi vẫn lỡ đụng nhau
1. Dừng, báo PM (ghi vào `current-state.md` mục Blocker).
2. Xác định file tranh chấp thuộc vùng ai theo bảng mục 1 → người đó giữ, người kia rebase.
3. Nếu là file `.md` dùng chung → merge thủ công, giữ cả 2 phần.
