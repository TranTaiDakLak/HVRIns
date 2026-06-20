# Completed Log

> Ghi lại task đã DONE (ai làm, test gì, file đã sửa, ghi chú). Người sau đọc file này thay vì
> đọc lại chi tiết task. Thêm dòng mới lên ĐẦU mỗi mục sprint.

## Sprint 00
- [S00-D2-T001] DONE — Dev 2 — 2026-06-20
  - Việc: Đọc toàn bộ plan docs (05-secrets, 01, 03, 02); check git/node/npm
  - Test: git 2.54, node 24.16, npm 11.13; danh sách 4 file lộ + file CẤM xoá xác nhận
  - File: —
  - Ghi chú: Không có git-filter-repo/BFG; sẽ ghi plan vào S04-D2-T002

- [S00-D2-T002] DONE — Dev 2 — 2026-06-20
  - Việc: git rm --cached 4 file secrets + thêm .gitignore (Config/Cookie/, test_accounts*.txt, __pycache__/, *.pyc)
  - Test: `git check-ignore` 4 path → in ra đường dẫn ✅; `git ls-files` 4 file lộ → rỗng ✅; internal/cookie/embedded/cookie_initial.txt vẫn track ✅
  - File: .gitignore (thêm 8 dòng cuối); 4 file gỡ khỏi index
  - Commit: 9bfe34a "security: go credentials khoi tracking + chan .gitignore"

- [S00-D2-T003] DONE — Dev 2 — 2026-06-20
  - Việc: Điền checklist rotate creds vào risks.md; xác nhận file embedded nguyên vẹn
  - Test: `git ls-files internal/cookie/embedded/cookie_initial.txt` → còn track ✅
  - File: workspaces/pm/risks.md (cập nhật checklist)
  - Ghi chú: `wails build` PRE-EXISTING FAIL (go.mod dirty: wails v2.11→v2.12, thiếu wailsjs runtime) — KHÔNG phải do secrets changes. Đây là việc Dev 1 S00-D1-T003. Rotate credential thật: TODO cho chủ dự án.

## Sprint 01
*(chưa có task DONE)*

## Sprint 02
*(chưa có task DONE)*

## Sprint 03
*(chưa có task DONE)*

## Sprint 04
*(chưa có task DONE)*

---

### Mẫu một dòng log (copy khi DONE)
```
- [S01-D2-T004] DONE — Dev 2 — 2026-06-__
  - Việc: dọn cmd/, icongen→tools/, go mod tidy
  - Test: `go build ./tools/...` PASS; `wails build` PASS; x/image còn trong go.mod
  - File: cmd/* (xoá 17 dir), tools/icongen/, go.mod, go.sum
  - Ghi chú: <điều cần người sau biết>
```

---

## Khởi tạo
- [PM] 2026-06-20 — Tạo `workspaces/` (INIT), chia 5 sprint / 30 task cho 2 dev. Kế hoạch nguồn: `docs/rebuild/`.
