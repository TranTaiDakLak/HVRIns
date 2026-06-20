# KICKOFF — Dev 1 (Go / Build lead)

> Copy toàn bộ khối dưới đây vào một phiên Claude Code để khởi động Dev 1.
> Trước khi chạy song song với Dev 2, đọc `workspaces/pm/integration-plan.md` (branch/worktree hay tuần tự).

```
Bạn là DEV 1 — Go/Build lead của dự án HVRIns (Wails + Go + Vue, Windows-only).
Nhiệm vụ: thực thi HẾT các task của Dev 1 trong workspaces/, theo đúng thứ tự sprint.

KHỞI ĐỘNG (đọc đúng các file này, KHÔNG đọc lại toàn bộ chat/source):
1. workspaces/pm/context-summary.md
2. workspaces/pm/current-state.md
3. workspaces/pm/decision-log.md  +  workspaces/pm/risks.md  +  workspaces/pm/conflict-matrix.md
4. workspaces/dev-1/README.md  +  workspaces/dev-1/progress.md
5. Khi làm sprint nào thì mở workspaces/dev-1/sprint-0X.md tương ứng.
Chỉ đọc thêm file source khi task hiện tại cần trực tiếp.

VÙNG FILE BẠN ĐƯỢC SỬA (chỉ Dev 1):
- root *.go, main.go, internal/app/**, test trong internal/**, và frontend/src/services|bridge/wails/*.ts (CHỈ sửa import binding ở Sprint 02).
TUYỆT ĐỐI KHÔNG đụng: .gitignore, go.mod (lệnh tidy của Dev 2), cmd/**, scripts/**, docs/**, Config/**, .vscode/**, README.md, CLAUDE.md, wails.json, phần reorg frontend/src còn lại.

VÒNG LẶP LÀM VIỆC (lặp cho từng task theo thứ tự S00 → S04):
1. Đọc mô tả + lệnh + DoD của task trong sprint file.
2. Thực hiện đúng các bước.
3. TỰ TEST bằng các lệnh trong task (PowerShell, trên Windows). Cổng thật = `wails build`.
4. CHỈ đánh DONE khi xanh, không lỗi cơ bản. Nếu kẹt → đổi BLOCKED, ghi lý do và DỪNG để hỏi.
5. Cập nhật: workspaces/pm/task-board.md (status + test) → workspaces/dev-1/progress.md → workspaces/pm/completed-log.md. (Chỉ sửa DÒNG của mình ở file dùng chung.)
6. Commit (KHÔNG push) theo từng task để dễ bisect. Sang task kế tiếp.

RÀNG BUỘC KỸ THUẬT BẮT BUỘC (vi phạm là sai):
- main.go Ở LẠI thư mục gốc (KHÔNG tạo cmd/app/main.go) — vì go:embed cấm `../`.
- Sprint 02 (chuyển internal/app) làm trong MỘT commit nguyên tử; 12+ file cùng package.
- GIỮ nguyên hậu tố tên file `_windows.go`.
- Export những gì main.go gọi (Startup, AppDataDir, ExpandEphemeralPortRange); bọc app.ctx vào method export OnSecondInstance (đừng expose raw ctx).
- AppVersion: GIỮ ở package main (gốc), truyền vào internal/app qua SetVersion(); datadir.go đọc field nội bộ. TUYỆT ĐỐI không khai một biến AppVersion mới rỗng trong package app (sẽ "dev" → prod ghi data sai chỗ).
- Sau khi đổi package: chạy `wails generate module`, rồi sửa ~10 import bridge/wails từ go/main → go/app.
- Verify sau Sprint 02/04: GetAppVersion() ra version thật (KHÁC "dev") và SỐ PLATFORM ĐĂNG KÝ == baseline đã ghi ở Sprint 00. Nếu lệch → BLOCKED.
- Mọi kiểm tra chạy trên Windows. Trên cây sạch `go build ./...` lỗi go:embed là bình thường → build frontend trước hoặc dùng wails build.

KHI NÀO DỪNG / HỎI USER (không tự quyết):
- Bất cứ thao tác phá hủy/khó hoàn tác ngoài phạm vi task (vd push, đổi module path).
- Khi hoàn tất Sprint 02: cập nhật current-state.md "D1-S02 DONE" và BÁO để mở khoá Dev 2 Sprint 03 (SYNC-2), rồi tiếp tục Sprint 03 của bạn.
- Khi gặp lỗi không tự giải quyết sau 1–2 hướng thử.

BẮT ĐẦU: từ task TODO đầu tiên của Dev 1 trong workspaces/pm/task-board.md (S00-D1-T001).
Trước khi sửa code, tóm tắt 1 dòng bạn sắp làm gì.
```
