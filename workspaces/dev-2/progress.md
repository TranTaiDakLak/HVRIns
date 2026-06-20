# Dev 2 — Progress

> Tự cập nhật sau mỗi task: làm gì, test gì, còn vướng gì. Mục mới lên trên cùng mỗi sprint.

## Trạng thái hiện tại
- Sprint đang làm: **Sprint 05** (S00→S04 D2 DONE)
- Task hiện tại: **S05-D2-T001 CHỜ Dev 1** (S05-D1-T001 phải DONE trước)
- Blocker: QA chức năng (Q3/Q7/RG-1..5) chờ Dev 1 validate internal/result

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
- [S03-D2-T001] DONE 2026-06-20 — bật alias @/ (tsconfig+vite); convert 195 relative import → @/.
  Test: npm run build PASS. 17 wailsjs imports giữ relative (độ sâu đúng). Commit ba1e177.
- [S03-D2-T002] DONE 2026-06-20 — xoá stub src/main.ts + src/App.vue (dead code).
  Quyết định D-009: giữ src/app/ (không làm phẳng). npm run build PASS. Commit c314943.
- [S03-D2-T003] DONE 2026-06-20 — bridge/ → services/ (giữ nguyên độ sâu thư mục).
  Update 39 file @/bridge → @/services; fix 2 dynamic import() còn sót. Commit 681770e.
- [S03-D2-T004] DONE 2026-06-20 — modules/ → features/; gom pages vào feature; vitest pass.
  AccountsPage, AuthSource, RegStats → features/pages/. settings/, schema/, reg-stats gom vào feature.
  routes.ts dynamic import → @/. package.json "test": "vitest run". passWithNoTests=true.
  npm build PASS; npm test exit 0. Swept vào commit a3d8210 (Dev 1 commit overlap).

### Sprint 04
- [S04-D2-T001] DONE 2026-06-20 — rà gốc repo gọn; tick review-checklist S03.
  Xoá icongen.exe + empty cmd/ + empty Config/. Sự cố: Config=config case-insensitive → restore.
  .gitignore thêm 4 runtime dirs. Commit 6ec59a7.
- [S04-D2-T002] DONE (S00) — kế hoạch rewrite git history đã ghi đầy đủ vào risks.md (commit ce1bf66).

### Sprint 05
- [S05-D2-T002] DONE 2026-06-21 — viết frontend test thật (30 tests PASS).
  vitest.config.ts: thêm alias @/ + bỏ passWithNoTests. useSelection.test.ts (17 tests).
  useAccountsStore.test.ts (13 tests) — vi.mock service/client. Commit 1d0f0c8.
- [S05-D2-T003] DONE 2026-06-21 — audit + đồng bộ docs.
  docs/06: 4→5 go:embed. project-scan.md: cập nhật. 01-hien-trang: note internal/result (D-012).
  decision-log: D-012 + D-013. Secret check: sạch. Commit 2f9057d.
- [S05-D2-T001] CHỜ Dev 1 S05-D1-T001 DONE.

---
### Mẫu dòng nhật ký
```
- [S01-D2-T004] DONE 2026-06-__ — dọn cmd/, icongen→tools, go mod tidy.
  Test: go build ./tools/... PASS; wails build PASS; x/image còn.
  File: cmd/* (xoá 17), tools/icongen/, go.mod/sum. Note: emailtest secret đã xoá khỏi HEAD.
```
