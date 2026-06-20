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
| S01-D1-T001 | 1 | Tách `app.go` → accounts/settings/profiles/upload/stats/resources/dialogs.go (vẫn package main @ root) | app.go (+files mới) | TODO | — |
| S01-D1-T002 | 1 | Migration Design Note: danh sách export, cách bọc app.ctx, cách thread AppVersion | dev-1/, decision-log | TODO | — |
| S01-D1-T003 | 1 | Xác nhận 207 blank-import + 4 go:embed; ghi baseline check | dev-1/progress | TODO | — |
| S01-D2-T001 | 2 | Xoá `_patch_datr_diag.py`, `decode_request.py`; gỡ `scripts/__pycache__`; +.gitignore | .gitignore | DONE | git rm 2 py + pycache ✅ |
| S01-D2-T002 | 2 | Move docs: guide→docs/, README_TEST_EAAG→docs/testing/, old-docs→archive, facebook→flows, .kiro specs→docs/rebuild/specs | docs/** | DONE | 32 file rename ✅ |
| S01-D2-T003 | 2 | `build.bat`→scripts/ (cd gốc); migrate.ps1/rename_identity.ps1/recolor.py→scripts/legacy | scripts/** | DONE | cd /d added ✅ |
| S01-D2-T004 | 2 | cmd/: icongen→tools/, xoá 17 scratch, `go mod tidy`, xác nhận x/image còn | cmd/**, tools/, go.mod | DONE | x/image ✅; go build ./tools/... ✅ |
| S01-D2-T005 | 2 | config/sample (template Config/*→.example) + sửa launch.json HVR_DATA_DIR | config/**, .vscode/launch.json | DONE | 8 example files ✅; launch.json ✅ |

## Sprint 02 — ⭐ Cú chuyển internal/app
| Task ID | Dev | Mô tả | File chính | Status | Test |
|---------|-----|-------|------------|--------|------|
| S02-D1-T001 | 1 | `git mv` 12+ file → internal/app/ + đổi `package main`→`app` (giữ _windows) | internal/app/** | TODO | — |
| S02-D1-T002 | 1 | Export Startup/AppDataDir/ExpandEphemeralPortRange; bọc app.ctx → OnSecondInstance; thread AppVersion (SetVersion) | internal/app/** | TODO | — |
| S02-D1-T003 | 1 | main.go mỏng (giữ go:embed+AppVersion+os.Chdir đầu tiên); go vet/test/build | main.go | TODO | — |
| S02-D1-T004 | 1 | `wails generate module` + sửa ~10 import bridge/wails (go/main→go/app); wails build/dev; verify version & platform | frontend/src/bridge/wails/*.ts, wailsjs/ | TODO | — |
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
| S03-D1-T001 | 1 | Unit test thay cmd scratch: internal/proxy/*_test.go + regex test | internal/proxy/** | TODO | — |
| S03-D1-T002 | 1 | (tuỳ chọn) stub cpu_other.go/portrange_other.go cross-platform | internal/app/** | TODO | — |
| S03-D1-T003 | 1 | Rà import cycle settings/adapter/legacy.go sau khi App→internal/app | internal/settings/** | TODO | — |

## Sprint 04 — Finalize
| Task ID | Dev | Mô tả | File chính | Status | Test |
|---------|-----|-------|------------|--------|------|
| S04-D1-T001 | 1 | Verify cuối (wails build/dev, platform count, GetAppVersion); cập nhật board/log | — | TODO | — |
| S04-D1-T002 | 1 | Quyết định Pha 7 (defer/làm) → decision-log | pm/decision-log | TODO | — |
| S04-D2-T001 | 2 | Rà gốc repo gọn + tick review-checklist | pm/review-checklist | TODO | — |
| S04-D2-T002 | 2 | (tuỳ chọn) Kế hoạch rewrite git history cho secrets | pm/risks.md | TODO | — |

---

### Tổng kết tiến độ
- TODO: 8 · IN PROGRESS: 0 · BLOCKED: 0 · DONE: 22  (cập nhật 2026-06-20 D2 sau S03)
