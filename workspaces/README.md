# Workspaces — Hệ điều phối team (PM + 2 Dev)

> Đây là "phòng điều hành" của dự án tái cấu trúc HVRIns. Mọi quản lý/tiến độ nằm ở đây, **không**
> để thất lạc trong chat. Kế hoạch kỹ thuật nguồn: `docs/rebuild/`.

## Bản đồ thư mục
```
workspaces/
├── README.md                 # file này
├── pm/                       # nguồn quản lý chính (Tech Lead/PM)
│   ├── current-state.md      # ⭐ ĐỌC ĐẦU TIÊN mỗi phiên (UPDATE mode)
│   ├── context-summary.md    # ngữ cảnh ngắn (nạp nhanh, tiết kiệm token)
│   ├── project-scan.md       # snapshot số liệu repo lúc khởi tạo
│   ├── sprint-plan.md        # 5 sprint + sơ đồ phụ thuộc + ranh giới file
│   ├── task-board.md         # 30 task, ID duy nhất, status (DEV cập nhật)
│   ├── completed-log.md      # log task DONE (DEV cập nhật)
│   ├── decision-log.md       # quyết định kỹ thuật (D-001…)
│   ├── risks.md              # rủi ro + checklist rotate creds
│   ├── review-checklist.md   # PM nghiệm thu từng sprint
│   ├── conflict-matrix.md    # 🆕 ai sửa file nào (chống đụng)
│   ├── integration-plan.md   # 🆕 chiến lược branch + thứ tự merge
│   └── qa-acceptance-plan.md # 🆕 test chức năng thật khi nghiệm thu
├── dev-1/                    # Go/Build lead
│   ├── README.md  KICKOFF.md  progress.md  sprint-00..04.md
└── dev-2/                    # Cleanup/Infra/Frontend
    └── README.md  KICKOFF.md  progress.md  sprint-00..04.md
```

## 2 chế độ làm việc
- **INIT** (đã xong): chưa có `workspaces/` → phân tích repo, chia việc, tạo cấu trúc này.
- **UPDATE** (từ giờ): đã có `workspaces/` → **không đọc lại toàn bộ chat/source**. Đọc theo thứ tự:
  `pm/current-state.md` → `context-summary.md` → `task-board.md` → `sprint-plan.md` →
  `completed-log.md` → `decision-log.md` → `dev-*/progress.md`. Chỉ mở source khi task hiện tại cần.

## Giao thức cập nhật ngược (sau MỖI lần chạy — BẮT BUỘC)
Để người/agent sau tốn ít token:
1. Dev cập nhật `pm/task-board.md` (status + test) cho task mình làm.
2. Dev ghi `dev-*/progress.md` (làm gì, test gì, vướng gì).
3. Dev thêm dòng `pm/completed-log.md` khi DONE.
4. Quyết định mới → `pm/decision-log.md`. Rủi ro mới → `pm/risks.md`.
5. PM cập nhật `pm/current-state.md` (sprint đang chạy, task DONE gần nhất, blocker, việc kế tiếp).
6. Thông tin quan trọng trong chat → chuyển vào `context-summary.md`/`decision-log.md`, đừng để trong chat.

## Bắt đầu nhanh
- Dev 1: mở `dev-1/KICKOFF.md`, dán prompt vào 1 phiên Claude Code.
- Dev 2: mở `dev-2/KICKOFF.md`, dán prompt vào 1 phiên Claude Code khác.
- Trước khi chạy song song: đọc `pm/integration-plan.md` (chọn branch/worktree hay tuần tự).

## Quy ước vàng (toàn dự án)
1. `main.go` ở **gốc** (không `cmd/app`) — go:embed cấm `../`. [D-001]
2. Cú chuyển `internal/app` = **1 commit nguyên tử**. [D-007]
3. Cổng kiểm tra = **`wails build`** trên **Windows** (không phải `go build ./...`).
4. Test white-box ở cạnh code, không gom vào `tests/`. [D-005]
5. **KHÔNG** xoá `internal/cookie/embedded/cookie_initial.txt`. [R-14]
6. Tôn trọng `pm/conflict-matrix.md` — không sửa file của dev kia.
