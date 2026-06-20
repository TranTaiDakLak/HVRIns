# Dev 2 — Progress

> Tự cập nhật sau mỗi task: làm gì, test gì, còn vướng gì. Mục mới lên trên cùng mỗi sprint.

## Trạng thái hiện tại
- Sprint đang làm: **Sprint 03** (S00+S01+S02 DONE — chờ Dev 1 S02 DONE trước khi làm FE reorg)
- Task hiện tại: **CHỜ Dev 1** — kiểm tra task-board S02-D1-* trước khi bắt Sprint 03
- Blocker: `wails build` BLOCKED (internal/result thiếu, Dev 1 scope); Dev 1 S02 chưa DONE

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
- [S01-D2-T001] DONE 2026-06-20 — xoá _patch_datr_diag.py, decode_request.py; rm --cached pycache.
- [S01-D2-T002] DONE 2026-06-20 — gom docs: NVRINS_BUILD_GUIDE→docs/, README_TEST→docs/testing/,
  old-docs(16)→archive, facebook→flows, .kiro/specs→docs/rebuild/specs.
- [S01-D2-T003] DONE 2026-06-20 — build.bat→scripts/ (cd /d head); legacy 3 script→scripts/legacy/.
- [S01-D2-T004] DONE 2026-06-20 — icongen→tools/; xoá 17 scratch cmd; go mod tidy;
  x/image còn ✅; go build ./tools/... ✅.
- [S01-D2-T005] DONE 2026-06-20 — 8 template→config/sample/*.example.txt; launch.json HVR→HVRINS.

### Sprint 02
- [S02-D2-T001] DONE 2026-06-20 — README.md gốc (stack/env/build/cây/bảo mật).
- [S02-D2-T002] DONE 2026-06-20 — CLAUDE.md rewrite (app thật); wails.json author điền.
- [S02-D2-T003] DONE 2026-06-20 — scaffold tests/go/ + tests/frontend/ với README.

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
