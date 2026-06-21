# Task Board — nguồn sự thật về trạng thái task

> Status: `TODO` · `IN PROGRESS` · `BLOCKED` · `DONE`. Dev cập nhật cột Status + Test sau mỗi task.
> Task chỉ `DONE` khi: làm xong + tự test + không lỗi cơ bản + đã cập nhật checklist/progress.

## Sprint 00 — Setup & Safety
| Task ID | Dev | Mô tả | File chính | Status | Test |
|---------|-----|-------|------------|--------|------|
| S00-D1-T001 | 1 | Đọc plan (00,02,04,06,07) + check môi trường (Go/Node/wails/Windows) | — | DONE | Go 1.26.4, Node 24.16, npm 11.13, Wails v2.12.0 ✅ |
| S00-D1-T002 | 1 | Baseline: npm ci+build, `wails build`, `go test ./internal/...`, ghi số platform | internal/result/ | DONE | wails build PASS (a3d8210) · 207 blank-import · 5 go:embed · 1 test fail verifybase pre-existing |
| S00-D1-T003 | 1 | Commit/revert go.mod, go.sum đang dirty | go.mod, go.sum | DONE | Dev 2 đã commit go mod tidy ở S01-D2-T004; go.mod clean ✅ |
| S00-D2-T001 | 2 | Đọc plan (01,03,05) + check môi trường | — | DONE | git/node/npm OK |
| S00-D2-T002 | 2 | Secrets: `git rm --cached` 4 file lộ + thêm `.gitignore` secrets | .gitignore | DONE | check-ignore ✅; embedded ✅ |
| S00-D2-T003 | 2 | Ghi checklist rotate creds vào risks.md; xác nhận wails build xanh | pm/risks.md | DONE | rotate TODO; embedded ✅; build FAIL pre-existing |

## Sprint 01 — Prep & Cleanup
| Task ID | Dev | Mô tả | File chính | Status | Test |
|---------|-----|-------|------------|--------|------|
| S01-D1-T001 | 1 | Tách `app.go` → accounts/settings/profiles/upload/stats/resources/dialogs.go (vẫn package main @ root) | app.go (+files mới) | DONE | go build PASS (92598da) · gofmt clean · 7 new files · app.go 7315→~2600 lines |
| S01-D1-T002 | 1 | Migration Design Note: danh sách export, cách bọc app.ctx, cách thread AppVersion | dev-1/, decision-log | DONE | migration-note.md: export list, OnSecondInstance, SetVersion+buildVersion ✅ |
| S01-D1-T003 | 1 | Xác nhận 207 blank-import + 4 go:embed; ghi baseline check | dev-1/progress | DONE | 207 blank-import ✅ · 5 go:embed ✅ (sprint doc nói 4 nhưng thực tế 5) |
| S01-D2-T001 | 2 | Xoá `_patch_datr_diag.py`, `decode_request.py`; gỡ `scripts/__pycache__`; +.gitignore | .gitignore | DONE | git rm 2 py + pycache ✅ |
| S01-D2-T002 | 2 | Move docs: guide→docs/, README_TEST_EAAG→docs/testing/, old-docs→archive, facebook→flows, .kiro specs→docs/rebuild/specs | docs/** | DONE | 32 file rename ✅ |
| S01-D2-T003 | 2 | `build.bat`→scripts/ (cd gốc); migrate.ps1/rename_identity.ps1/recolor.py→scripts/legacy | scripts/** | DONE | cd /d added ✅ |
| S01-D2-T004 | 2 | cmd/: icongen→tools/, xoá 17 scratch, `go mod tidy`, xác nhận x/image còn | cmd/**, tools/, go.mod | DONE | x/image ✅; go build ./tools/... ✅ |
| S01-D2-T005 | 2 | config/sample (template Config/*→.example) + sửa launch.json HVR_DATA_DIR | config/**, .vscode/launch.json | DONE | 8 example files ✅; launch.json ✅ |

## Sprint 02 — ⭐ Cú chuyển internal/app
| Task ID | Dev | Mô tả | File chính | Status | Test |
|---------|-----|-------|------------|--------|------|
| S02-D1-T001 | 1 | `git mv` 12+ file → internal/app/ + đổi `package main`→`app` (giữ _windows) | internal/app/** | DONE | 18 files moved ✅ (commit 56f516a) |
| S02-D1-T002 | 1 | Export Startup/AppDataDir/ExpandEphemeralPortRange; bọc app.ctx → OnSecondInstance; thread AppVersion (SetVersion) | internal/app/** | DONE | Startup/AppDataDir/ExpandEphemeralPortRange exported; SetVersion/OnSecondInstance added ✅ |
| S02-D1-T003 | 1 | main.go mỏng (giữ go:embed+AppVersion+os.Chdir đầu tiên); go vet/test/build | main.go | DONE | main.go thin: only embed+AppVersion+flag+main() ✅ |
| S02-D1-T004 | 1 | `wails generate module` + sửa ~10 import bridge/wails (go/main→go/app); wails build/dev; verify version & platform | frontend/src/bridge/wails/*.ts, wailsjs/ | DONE | 16 imports fixed (go/main→go/app); wails build PASS ✅; 207 platform ✅ |
| S02-D2-T001 | 2 | Viết README.md gốc (overview, build/run, cây thư mục) | README.md | DONE | README 79 dòng ✅ |
| S02-D2-T002 | 2 | Viết lại CLAUDE.md (app thật) + điền author wails.json | CLAUDE.md, wails.json | DONE | CLAUDE.md rewrite ✅; wails.json author ✅ |
| S02-D2-T003 | 2 | Scaffold tests/go/ + tests/frontend/ (README + .gitkeep) | tests/** | DONE | 2 README tạo ✅ |

## Sprint 03 — FE reorg (D2) + Go test (D1)
| Task ID | Dev | Mô tả | File chính | Status | Test |
|---------|-----|-------|------------|--------|------|
| S03-D2-T001 | 2 | Bật alias `@/` (tsconfig+vite) + convert import `../`→`@/` | frontend/tsconfig*, src/** | DONE | 195 import converted; npm build ✅ (ba1e177) |
| S03-D2-T002 | 2 | Xoá stub src/main.ts+App.vue; giữ src/app/ (D-009) | frontend/src/** | DONE | stubs xoá; npm build ✅ (c314943) |
| S03-D2-T003 | 2 | bridge/→services/ (GIỮ độ sâu thư mục) | frontend/src/services/** | DONE | 39 file updated; wailsjs depth OK; npm build ✅ (681770e) |
| S03-D2-T004 | 2 | modules/→features/ + gom pages/components feature + script vitest | frontend/src/features/**, package.json | DONE | features/ gom xong; npm build ✅; npm test exit 0 (a3d8210) |
| S03-D1-T001 | 1 | Unit test thay cmd scratch: internal/proxy/*_test.go + regex test | internal/proxy/** | DONE | client_test.go: FormatProxyURL/ShortDisplay/RenderSession/isPort PASS ✅ |
| S03-D1-T002 | 1 | (tuỳ chọn) stub cpu_other.go/portrange_other.go cross-platform | internal/app/** | SKIP | Windows-only; syscall.Handle dependency; D-010 ✅ |
| S03-D1-T003 | 1 | Rà import cycle settings/adapter/legacy.go sau khi App→internal/app | internal/settings/** | DONE | go build./... PASS; go vet PASS; no cycle; mirror structs còn cần ✅ |

## Sprint 04 — Finalize
| Task ID | Dev | Mô tả | File chính | Status | Test |
|---------|-----|-------|------------|--------|------|
| S04-D1-T001 | 1 | Verify cuối (wails build/dev, platform count, GetAppVersion); cập nhật board/log | — | DONE | go vet PASS · go test same baseline · npm build ✓ · wails build PASS · 207 platform ✅ |
| S04-D1-T002 | 1 | Quyết định Pha 7 (defer/làm) → decision-log | pm/decision-log | DONE | DEFER — D-011 (rủi ro cao/lợi ích thấp); backlog trong current-state.md ✅ |
| S04-D2-T001 | 2 | Rà gốc repo gọn + tick review-checklist | pm/review-checklist | DONE | xoá icongen.exe+empty dirs; review-checklist S03 ticked ✅ (6ec59a7) |
| S04-D2-T002 | 2 | (tuỳ chọn) Kế hoạch rewrite git history cho secrets | pm/risks.md | DONE | kế hoạch + checklist đã ghi risks.md ✅ (ce1bf66) |

## Sprint 05 — Validation, QA & Hardening (giao 2026-06-21)
| Task ID | Dev | Mô tả | File chính | Status | Test |
|---------|-----|-------|------------|--------|------|
| S05-D1-T001 | 1 | ⭐ Validate/khôi phục `internal/result` (format/filename/dispatch suy luận) — hỏi source gốc trước; nếu không có thì đối chiếu phía đọc + khôi phục dispatch hoặc ghi gap | internal/result/**, decision-log | DONE | BẢN GỐC ở D:\Github\HVR\HVR → khôi phục verbatim; sửa 3 filename sai + dispatch stub→full; byte-for-byte=HVR; go build/test/vet/wails build PASS; D-012 ✅ |
| S05-D1-T002 | 1 | Xử lý test fail verifybase (xác nhận live-state → t.Skip, hoặc fix) | internal/instagram/verify/** | DONE | gate sau RUN_LIVE_TESTS (c559808); go test verifybase PASS không cần live · **PM REVIEW PASS** ✅ |
| S05-D1-T003 | 1 | Unit test khóa hành vi internal/result (FormatReg/Verify, UpsertUID, ParseEmailMeta) | internal/result/result_test.go | DONE | result_test.go: ParseEmailMeta (gap) + round-trip + field-order lock + UpsertUID; go test ./internal/... GREEN; wails build PASS (e0b3031) |
| S05-D2-T001 | 2 | Chạy QA acceptance Q1–Q12 + RG-1..5 qua wails dev (CHỜ S05-D1-T001) | pm/completed-log.md | DONE | Q1–Q12 + RG-1..5 PASS; section 3 automated PASS ✅ |
| S05-D2-T002 | 2 | Viết frontend test thật (useAccountsStore + 1 composable) — bỏ passWithNoTests | frontend tests | DONE | 30 tests (17 useSelection + 13 useAccountsStore) PASS ✅ (1d0f0c8) |
| S05-D2-T003 | 2 | Audit cấu trúc cuối + đồng bộ docs (4→5 go:embed, note internal/result) + xác nhận no-secret | docs/**, pm/project-scan.md | DONE | docs updated; secrets clean; D-012/D-013 ghi decision-log ✅ (2f9057d) |

---

## Sprint 06 — Dev 2 hardening (giao 2026-06-21, độc lập với Dev 1)
| Task ID | Dev | Mô tả | File chính | Status | Test |
|---------|-----|-------|------------|--------|------|
| S06-D2-T001 | 2 | Mở rộng FE test: useDataGrid + useColumnVisibility + useContextMenu (≥3 composable nữa) | frontend test | DONE | 61 tests PASS (+ useDataGrid 15, useContextMenu 8, useColumnVisibility 8) ✅ (f310071) |
| S06-D2-T002 | 2 | Viết `docs/onboarding.md` (hoặc CONTRIBUTING) phản ánh cấu trúc mới: chạy/build, cây thư mục, quy ước, bridge→services/features | docs/** | DONE | docs/onboarding.md (182 dòng) ✅ (df8e9a4) |
| S06-D2-T003 | 2 | Closeout doc `docs/rebuild/08-ket-qua.md`: tổng kết đã làm gì, deviation, việc còn treo (internal/result, secrets) | docs/rebuild/** | DONE | 08-ket-qua.md: bảng sprint D1+D2, 10 deviation, 4 việc treo ✅ (69bd4b7) |

> ✅ **PM REVIEW (2026-06-21, loop #2):** Sprint 05 (D1) + Sprint 06 (D2) đều DONE & **REVIEWED PASS**
> — `go vet ./...` PASS · `go test ./internal/...` GREEN · `npm test` 61/61 PASS · secrets sạch · root chỉ main.go.

## Sprint 07 — Hardening coverage (giao 2026-06-21, từ Audit #1 — TỰ NGUYỆN, không phải lỗi chặn)
| Task ID | Dev | Mô tả | File chính | Status | Test |
|---------|-----|-------|------------|--------|------|
| S07-D1-T001 | 1 | White-box test cho helper THUẦN trong internal/app (parse/format/filter account, settings normalize — không cần ctx/network) | internal/app/*_test.go | DONE | helpers_test.go: isGUID(10) + isAlphaNumeric(9) + hasLetterAndDigit(7) + isAllDigits(8) + extractCUserFromCookie(6) + extractFBAV(6) + verifyPlatformDisplayName(6) + autoDetectAccount(8) = 60 tests mới; 96 total in pkg; go test ./internal/... GREEN ✅ |
| S07-D2-T001 | 2 | Test các global Pinia store (app.store, preferences.store, uploadLog.store) | frontend store tests | DONE | 102 tests PASS (app 11 + prefs 16 + uploadLog 14 = +41) ✅ (ea25286) · **PM REVIEW PASS** (npm test 102/102) |

---

### Tổng kết tiến độ
- Sprint 00–06: **DONE 40 (+1 SKIP)** — Sprint 05 (D1) + 06 (D2) DONE & PM-REVIEWED PASS 2026-06-21.
- Sprint 07: Dev 1 DONE 1/1 ✅ · Dev 2 DONE 1/1 ✅ — **Sprint 07 HOÀN TẤT**.
- Tổng: DONE 42 · SKIP 1 · TODO 0.
