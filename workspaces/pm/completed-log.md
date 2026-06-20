# Completed Log

> Ghi lại task đã DONE (ai làm, test gì, file đã sửa, ghi chú). Người sau đọc file này thay vì
> đọc lại chi tiết task. Thêm dòng mới lên ĐẦU mỗi mục sprint.

## Sprint 00
- [S00-D1-T001] DONE — Dev 1 — 2026-06-20
  - Việc: Đọc plan docs (00,02,04,06,07), check môi trường Go/Node/Wails
  - Test: Go 1.26.4 ✅, Node v24.16.0 ✅, npm v11.13.0 ✅, Wails v2.12.0 ✅
  - File: — (read-only)
  - Ghi chú: Hiểu D-001 (main.go ở gốc vì go:embed) và D-007 (commit nguyên tử). Phát hiện go.mod dirty = wails upgrade 2.11→2.12 (hợp lệ, sẽ commit ở T003).

- [S00-D1-T002] BLOCKED — Dev 1 — 2026-06-20
  - Lý do: `internal/result` package bị thiếu hoàn toàn (không có trong git history, không có trên disk). Được import bởi app.go:34, app_register.go:163, app_verify.go:14.
  - Hậu quả: wails build FAIL · wails generate module FAIL · npm run build FAIL (wailsjs/runtime missing vì generate không chạy được)
  - go test ./internal/... PASS phần lớn; 1 fail ở verifybase (live account kiểm tra thực tế)
  - Cần: user quyết định tạo `internal/result` hay cung cấp source

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
- [S01-D2-T001] DONE — Dev 2 — 2026-06-20
  - Việc: git rm 2 py debug + git rm --cached pycache
  - Test: git ls-files sạch ✅
  - File: _patch_datr_diag.py, decode_request.py (xoá); scripts/__pycache__/ (untrack)

- [S01-D2-T002] DONE — Dev 2 — 2026-06-20
  - Việc: gom docs (NVRINS_BUILD_GUIDE→docs/; README_TEST→docs/testing/eaag-verify-flow.md;
    old-docs(16)→docs/archive/; docs/facebook→docs/flows/; .kiro/specs→docs/rebuild/specs/ copy)
  - Test: git rename 32 file ✅; outdated note thêm vào eaag-verify-flow.md ✅
  - File: docs/ (nhiều rename); docs/rebuild/specs/ (3 file mới)

- [S01-D2-T003] DONE — Dev 2 — 2026-06-20
  - Việc: git mv build.bat→scripts/; thêm cd /d "%~dp0\.."; 3 script→scripts/legacy/
  - Test: scripts/build.bat có cd gốc ✅ (chưa chạy thật vì wails build pre-existing fail)
  - File: scripts/build.bat, scripts/legacy/{migrate,rename_identity}.ps1, scripts/legacy/recolor.py

- [S01-D2-T004] DONE — Dev 2 — 2026-06-20
  - Việc: icongen→tools/icongen; git rm 17 scratch cmd/ (kể cả emailtest chứa secret); go mod tidy
  - Test: x/image còn ✅; go build ./tools/... PASS ✅
  - File: cmd/* (xoá 17 dir); tools/icongen/; go.mod/sum; ghi chú: emailtest secret còn trong history

- [S01-D2-T005] DONE — Dev 2 — 2026-06-20
  - Việc: 8 template Config/*→config/sample/*.example.txt; launch.json HVR_DATA_DIR→HVRINS_DATA_DIR
  - Test: Config/ rỗng trong index ✅; config/sample/ 8 file ✅
  - File: config/sample/**, .vscode/launch.json

## Sprint 02
- [S02-D2-T001] DONE — Dev 2 — 2026-06-20
  - Việc: Viết README.md gốc (stack, env req, cách chạy, cây thư mục, deviation, security)
  - Test: markdown preview OK; link tới docs/rebuild/ hợp lệ
  - File: README.md (79 dòng mới)

- [S02-D2-T002] DONE — Dev 2 — 2026-06-20
  - Việc: Viết lại CLAUDE.md (app thật, bỏ "greenfield frontend"); điền wails.json author từ git config
  - Test: CLAUDE.md không còn mâu thuẫn thực tế; wails.json valid JSON ✅
  - File: CLAUDE.md (rewrite 47 dòng); wails.json (author điền)

- [S02-D2-T003] DONE — Dev 2 — 2026-06-20
  - Việc: scaffold tests/go/README.md + tests/frontend/README.md
  - Test: git status thấy 2 file mới ✅
  - File: tests/go/README.md, tests/frontend/README.md

## Sprint 03
- [S03-D2-T001] DONE — Dev 2 — 2026-06-20
  - Việc: bật alias @/ (tsconfig + vite.config); viết script convert-imports.cjs; convert 195 relative import
  - Test: npm run build PASS; 17 wailsjs imports giữ relative ✅
  - File: frontend/tsconfig.json, frontend/vite.config.ts, 69 file src/**
  - Commit: ba1e177

- [S03-D2-T002] DONE — Dev 2 — 2026-06-20
  - Việc: xoá stub src/main.ts (export {}), src/App.vue (empty); giữ src/app/ (D-009)
  - Test: npm run build PASS ✅; index.html vẫn → src/app/main.ts
  - File: frontend/src/main.ts (xoá), frontend/src/App.vue (xoá)
  - Ghi chú: không flatten src/app/ — rủi ro không cần thiết, ghi decision-log D-009
  - Commit: c314943

- [S03-D2-T003] DONE — Dev 2 — 2026-06-20
  - Việc: git mv bridge → services; update 39 file @/bridge → @/services; fix 2 dynamic import() sót
  - Test: npm run build PASS ✅; wailsjs relative path (../../../) vẫn đúng ✅
  - File: frontend/src/services/** (rename từ bridge/), AccountsImportDialog.vue, AccountsPage.vue
  - Commit: 681770e

- [S03-D2-T004] DONE — Dev 2 — 2026-06-20
  - Việc: modules→features; AccountsPage/AuthSource/RegStats → features/*/pages/; settings+schema+reg-stats gom feature; routes.ts dynamic import → @/; "test":"vitest run"; passWithNoTests=true
  - Test: npm run build PASS ✅; npm test exit 0 ✅
  - File: frontend/src/features/**, frontend/package.json, frontend/vitest.config.ts, frontend/src/app/router/routes.ts
  - Ghi chú: changes swept vào commit a3d8210 (Dev 1 commit overlap — staging area shared)

## Sprint 05
- [S05-D2-T002] DONE — Dev 2 — 2026-06-21
  - Việc: vitest.config.ts thêm alias @/ + bỏ passWithNoTests; viết useSelection.test.ts (17 tests) + useAccountsStore.test.ts (13 tests)
  - Test: 30/30 PASS ✅; npm run build PASS ✅
  - File: frontend/vitest.config.ts, frontend/src/composables/useSelection.test.ts, frontend/src/features/accounts/store/useAccountsStore.test.ts
  - Ghi chú: vi.mock('@/services/client') để bypass waitForWails() 2s; setActivePinia(createPinia()) mỗi test
  - Commit: 1d0f0c8

- [S05-D2-T003] DONE — Dev 2 — 2026-06-21
  - Việc: audit cấu trúc vs 02-cau-truc-dich.md; sửa "4→5 go:embed" trong docs + project-scan; note internal/result (D-012); D-013 (cmd/ deleted); secret check; embedded còn
  - Test: git ls-files sạch ✅; embedded cookie còn ✅; x/image còn ✅
  - File: docs/rebuild/06-go-wails-cho-newbie.md, docs/rebuild/01-hien-trang.md, workspaces/pm/project-scan.md, workspaces/pm/decision-log.md
  - Commit: 2f9057d

## Sprint 04
- [S04-D2-T001] DONE — Dev 2 — 2026-06-20
  - Việc: Rà gốc repo; xoá icongen.exe + empty cmd/ + empty Config/ ở gốc
  - Sự cố: Windows case-insensitive `Config/`=`config/` → xoá nhầm config/sample/ → khôi phục bằng `git restore`
  - Thêm .gitignore: Config/Proxy/, Config/TempMail/, Config/Permanent/, Config/DeviceInfo/ (runtime dirs)
  - Tick review-checklist Sprint 03 (4 hạng mục D2 ✅), cập nhật sprint 04 notes
  - Test: `git ls-files config/sample/` → 8 file ✅; `git status` sạch
  - Commit: 6ec59a7

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
