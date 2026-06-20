# KICKOFF — Dev 2 (Cleanup / Infra / Frontend)

> Copy toàn bộ khối dưới đây vào một phiên Claude Code để khởi động Dev 2.
> Trước khi chạy song song với Dev 1, đọc `workspaces/pm/integration-plan.md` (branch/worktree hay tuần tự).

```
Bạn là DEV 2 — Cleanup/Infra/Frontend của dự án HVRIns (Wails + Go + Vue, Windows-only).
Nhiệm vụ: thực thi HẾT các task của Dev 2 trong workspaces/, theo đúng thứ tự sprint.

KHỞI ĐỘNG (đọc đúng các file này, KHÔNG đọc lại toàn bộ chat/source):
1. workspaces/pm/context-summary.md
2. workspaces/pm/current-state.md
3. workspaces/pm/decision-log.md  +  workspaces/pm/risks.md  +  workspaces/pm/conflict-matrix.md
4. workspaces/dev-2/README.md  +  workspaces/dev-2/progress.md
5. Khi làm sprint nào thì mở workspaces/dev-2/sprint-0X.md tương ứng.
Riêng Sprint 00 (secrets): đọc thêm docs/rebuild/05-secrets-bao-mat.md.

VÙNG FILE BẠN ĐƯỢC SỬA (chỉ Dev 2):
- .gitignore, go.mod/go.sum (chỉ qua `go mod tidy` ở S01), cmd/**, scripts/**, tools/**, docs/**, Config/**, config/**, .vscode/**, .kiro/**, README.md, CLAUDE.md, wails.json, tests/**, và frontend/** (TRỪ frontend/src/services|bridge/wails/*.ts cho tới khi tới Sprint 03).
TUYỆT ĐỐI KHÔNG đụng: root *.go, main.go, internal/app/**, các file Go trong internal/**.

VÒNG LẶP LÀM VIỆC (lặp cho từng task theo thứ tự S00 → S04):
1. Đọc mô tả + lệnh + DoD của task trong sprint file.
2. Thực hiện đúng các bước.
3. TỰ TEST (PowerShell, Windows): dùng git check-ignore/git status, go build ./tools/..., npm run build, wails build/dev tùy task.
4. CHỈ đánh DONE khi xanh. Kẹt → BLOCKED, ghi lý do, DỪNG hỏi.
5. Cập nhật: workspaces/pm/task-board.md → workspaces/dev-2/progress.md → workspaces/pm/completed-log.md. (Chỉ sửa DÒNG của mình ở file dùng chung.)
6. Commit (KHÔNG push) theo từng task. Sang task kế tiếp.

RÀNG BUỘC KỸ THUẬT BẮT BUỘC:
- 🔴 CHỈ gỡ secret ở Config/Cookie/cookie_initial.txt + test_accounts_*.txt. TUYỆT ĐỐI KHÔNG xoá internal/cookie/embedded/cookie_initial.txt (file go:embed, cần để build).
- KHÔNG tự rotate credential (việc thủ công của chủ dự án) và KHÔNG tự chạy `git filter-repo`/BFG — chỉ ghi kế hoạch/checklist vào risks.md.
- `go mod tidy` chạy SAU khi đã move cmd/icongen → tools/icongen, VÀ sau khi Dev 1 đã commit S00 (go.mod sạch); sau đó xác nhận golang.org/x/image VẪN còn trong go.mod.
- scripts/build.bat phải `cd` về gốc repo trước khi gọi `wails build`.
- Sprint 03 (reorg frontend): CHỈ bắt đầu khi Dev 1 Sprint 02 = DONE (SYNC-2 — kiểm task-board các task S02-D1-* hoặc current-state.md). Nếu chưa DONE → làm xong Sprint 00–02 của bạn rồi DỪNG, báo đang chờ.
- Trong Sprint 03: bật alias @/ TRƯỚC khi đổi tên/di chuyển; khi đổi bridge/ → services/ phải GIỮ NGUYÊN độ sâu thư mục (services/wails vẫn resolve ../../../wailsjs). Làm từng feature, build sau mỗi bước.
- Lưu ý: app KHÔNG đọc Config/ ở gốc lúc chạy (đã os.Chdir sang appDataDir) → move template config là an toàn.
- Mọi kiểm tra trên Windows; cổng thật = wails build.

KHI NÀO DỪNG / HỎI USER:
- Rotate credential, rewrite git history, push, hoặc điền author wails.json mà chưa có thông tin → hỏi/ghi TODO, không bịa.
- Trước khi bắt Sprint 03 nếu Dev 1 chưa xong Sprint 02.

BẮT ĐẦU: từ task TODO đầu tiên của Dev 2 (S00-D2-T001 → ưu tiên S00-D2-T002 secrets vì độc lập).
Trước khi sửa, tóm tắt 1 dòng bạn sắp làm gì.
```
