# Decision Log

> Ghi mọi quyết định kỹ thuật quan trọng + lý do. Đừng để quyết định chỉ nằm trong chat.

| # | Ngày | Quyết định | Lý do | Trạng thái |
|---|------|------------|-------|-----------|
| D-001 | 2026-06-20 | **`main.go` ở lại thư mục gốc** (KHÔNG chuyển vào `cmd/app/main.go` như chuẩn) | `//go:embed all:frontend/dist` cấm dùng `../`; `wails.json` không có trường vị trí main; `wails build` giả định main ở gốc. Đây là deviation có chủ ý, ghi rõ trong docs. | Chốt |
| D-002 | 2026-06-20 | **Chuyển toàn bộ logic App vào `internal/app` (`package app`)**, main.go chỉ còn bootstrap | Đạt tinh thần chuẩn "logic trong internal/, entry mỏng" mà không vi phạm ràng buộc go:embed. | Chốt |
| D-003 | 2026-06-20 | **`AppVersion` giữ ở `package main`** (gốc), truyền vào `internal/app` qua `SetVersion()`/constructor | Giữ `build.bat -X main.AppVersion` không phải đổi; nhưng `datadir.go` (vào internal/app) cần version → phải thread vào. Nếu khai biến mới trong app → mặc định "dev" → prod ghi data sai chỗ. | Chốt |
| D-004 | 2026-06-20 | **KHÔNG ánh xạ sâu `internal/` sang domain/usecase/adapter ở pass này** (Pha 7 tuỳ chọn) | Đổi đường dẫn ~2900 file + 207 blank-import = rủi ro cao, lợi ích thấp cho newbie. | Chốt (defer) |
| D-005 | 2026-06-20 | **Giữ `*_test.go` white-box cạnh code**, chỉ test black-box mới vào `tests/`; **giữ `internal/cookie/embedded/cookie_initial.txt`** | Quy tắc visibility của Go; file embedded là input build bắt buộc. | Chốt |
| D-006 | 2026-06-20 | **Installer giữ ở `build/windows/installer/`**, chỉ tài liệu hoá ở `infra/installer/` | Wails tự sinh `wails_tools.nsh` ở đó; di chuyển = hỏng `wails build -nsis`. | Chốt |
| D-007 | 2026-06-20 | **Cú chuyển internal/app là MỘT commit nguyên tử**; chỉ verify trên **Windows**; cổng kiểm tra là `wails build` | 12+ file cùng package (trạng thái nửa vời không compile); app Windows-only; `go build ./...` lỗi go:embed trên cây sạch. | Chốt |
| D-008 | 2026-06-20 | **Phân vai: D1=Go/Build, D2=Cleanup/Infra/FE**; D2-Sprint03 (FE) đợi D1-Sprint02 | Tránh 2 dev sửa trùng `frontend/src/bridge/wails` và root `*.go`. | Chốt |

| D-009 | 2026-06-20 | **Giữ `src/app/`** (KHÔNG làm phẳng `src/app/main.ts`→`src/main.ts`) | Flattening yêu cầu cập nhật `index.html` + mọi `@/app/router/...` import → rủi ro không cần thiết ở giai đoạn này. Stub cũ đã xoá; entry `src/app/main.ts` rõ ràng. | Chốt (S03-D2-T002) |
| D-010 | 2026-06-20 | **SKIP cross-platform stubs** (S03-D1-T002) — app dứt khoát Windows-only | `internal/proxy/transport_pool.go` dùng `syscall.Handle` (Windows-only); `wails.json` target Windows; không có CI Linux cần `go build ./...` xanh. Không cần `cpu_other.go`/`portrange_other.go`. | Chốt |

| D-011 | 2026-06-20 | **DEFER Pha 7** (internal/ deep mapping → domain/usecase/adapter) | 207 blank-import cross ~2900 file + circular risk settings/adapter → rủi ro cao, lợi ích thấp. D-004 xác nhận. Backlog item tạo trong current-state.md. | Chốt (DEFER) |
| D-012 | 2026-06-21 | **`internal/result` là package tái tạo** — ghi nhận để người sau biết | Package bị import nhưng chưa từng có trong repo gốc. Dev 1 dựng lại bằng suy luận (commit a3d8210). format.go/files.go/dispatch.go cần validate hành vi thật trước production. | Chốt (S05-D1-T001) |
| D-013 | 2026-06-21 | **`cmd/` xoá hoàn toàn** (khác target doc giữ empty `cmd/app/`) | Target `02-cau-truc-dich.md` nói `cmd/app/` để trống; nhưng cmd/ đã dọn sạch từ S01 (17 scratch xoá) → xoá thư mục trống thừa. Độ lệch nhỏ, chấp nhận được. | Chốt (S05-D2-T003) |
