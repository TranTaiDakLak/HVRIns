# Dev 2 — Progress

> Tự cập nhật sau mỗi task: làm gì, test gì, còn vướng gì. Mục mới lên trên cùng mỗi sprint.

## Trạng thái hiện tại
- Sprint đang làm: **Sprint 01** (S00 DONE)
- Task hiện tại: S01-D2-T001
- Blocker: `wails build` pre-existing fail (dirty go.mod, wails 2.12 binding issue) — Dev 1 scope

## Secrets status (S00-D2-T002/T003)
- 4 file lộ đã rm --cached? ✅ DONE (commit 9bfe34a)
- .gitignore đã chặn? ✅ DONE (Config/Cookie/, test_accounts*.txt, __pycache__/, *.pyc)
- File CẤM xoá `internal/cookie/embedded/cookie_initial.txt` còn nguyên? ✅ VERIFIED
- Đã rotate creds (FB/Hotmail)? ❌ TODO — việc thủ công của chủ dự án (xem risks.md)

---

## Nhật ký
### Sprint 00
- [S00-D2-T001] DONE 2026-06-20 — Đọc plan, check env (git 2.54/node 24/npm 11).
- [S00-D2-T002] DONE 2026-06-20 — secrets: git rm --cached 4 file + .gitignore. Commit 9bfe34a.
  Test: check-ignore ✅; ls-files rỗng ✅; embedded còn ✅.
- [S00-D2-T003] DONE 2026-06-20 — Ghi risks.md checklist rotate. Embedded file intact.
  Note: wails build PRE-EXISTING FAIL (dirty go.mod wails 2.12) — không phải do S00-D2-T002.

### Sprint 01
- (chưa có)

### Sprint 02
- (chưa có)

### Sprint 03
- (chưa có)

### Sprint 04
- (chưa có)

---
### Mẫu dòng nhật ký
```
- [S01-D2-T004] DONE 2026-06-__ — dọn cmd/, icongen→tools, go mod tidy.
  Test: go build ./tools/... PASS; wails build PASS; x/image còn.
  File: cmd/* (xoá 17), tools/icongen/, go.mod/sum. Note: emailtest secret đã xoá khỏi HEAD.
```
