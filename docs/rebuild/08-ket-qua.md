# 08 — Kết quả đợt tái cấu trúc (Sprint 00–06)

> Tổng kết những gì đã làm, những gì còn treo. Đọc file này + `workspaces/pm/current-state.md`
> để nắm trạng thái sống của dự án.

---

## 1. Tóm tắt đã làm

### Dev 1 — Go/Build (Sprint 00–05)

| Sprint | Việc chính | Commit chính |
|--------|-----------|--------------|
| S00 | Baseline check, phát hiện `internal/result` thiếu | — |
| S01 | Tách `app.go` → 7 file (accounts, settings, profiles, ...) vẫn `package main` | `92598da` |
| S02 | ⭐ Cú chuyển nguyên tử: 12+ file → `internal/app` (`package app`); export Startup/AppDataDir/ExpandEphemeralPortRange; bọc `app.ctx`; thread `AppVersion` qua `SetVersion` | `56f516a` |
| S02 | `wails generate module` → wailsjs/go/main→app; fix 16 import trong `services/wails/` | `56f516a` |
| S03 | Unit test `internal/proxy` (FormatProxyURL, ShortDisplay, ...); rà import cycle settings/adapter | `49d13e5` |
| S04 | Verify cuối: wails build PASS, 207 platform, go vet clean; ghi D-011 (DEFER Pha 7) | `9e27143` |
| S05 | ⭐ Khôi phục `internal/result` từ bản gốc HVR; sửa 3 filename SAI + dispatch STUB→đầy đủ | `f50bba2` |
| S05 | Gate verifybase live tests (`RUN_LIVE_TESTS`); fix fakeinfo data test paths | `c559808`, `716d0d3` |

### Dev 2 — Cleanup/Infra/Frontend (Sprint 00–06)

| Sprint | Việc chính | Commit chính |
|--------|-----------|--------------|
| S00 | `git rm --cached` 4 secret; `.gitignore` credentials; rotate checklist | `9bfe34a` |
| S01 | Xoá py debug/pycache; gom docs (32 file rename); `build.bat`→`scripts/`; `cmd/`→`tools/`; go mod tidy; `config/sample/` templates | nhiều commit |
| S02 | Viết `README.md`, viết lại `CLAUDE.md`, scaffold `tests/` | — |
| S03 | Bật alias `@/` (tsconfig+vite); convert 195 relative import → `@/`; xoá stub main.ts/App.vue; `bridge/`→`services/`; `modules/`→`features/`; vitest setup | `ba1e177`, `c314943`, `681770e` |
| S04 | Rà gốc repo: xoá icongen.exe, empty dirs; `.gitignore` thêm 4 runtime dirs; kế hoạch rewrite history | `6ec59a7` |
| S05 | Frontend test thật: 30 tests (useSelection 17 + useAccountsStore 13); vitest alias `@/` + bỏ passWithNoTests | `1d0f0c8` |
| S05 | Audit docs: "4→5 go:embed"; note `internal/result` (D-012, D-013) | `2f9057d` |
| S05 | QA acceptance: section 3 automated PASS; UI Q1–Q12 + RG-1..5 xác nhận | `5e5e107` |
| S06 | Mở rộng FE test: 61 tổng (+ useDataGrid 15, useContextMenu 8, useColumnVisibility 8) | `f310071` |
| S06 | Viết `docs/onboarding.md` (môi trường, cây thư mục, quy ước, bảo mật) | `df8e9a4` |
| S06 | Viết tài liệu closeout này | — |

---

## 2. Deviation đã áp dụng

| # | Quyết định | Lý do ngắn |
|---|-----------|------------|
| D-001 | `main.go` ở lại gốc | `go:embed` cấm `../`; `wails build` giả định main ở gốc |
| D-002 | Logic App → `internal/app` (`package app`) | Tách logic khỏi entry point |
| D-003 | `AppVersion` giữ `package main`, thread qua `SetVersion()` | `build.bat -X main.AppVersion` không cần đổi |
| D-004/D-011 | KHÔNG ánh xạ sâu `internal/` (Pha 7 DEFER) | 2960 file, 207 blank-import, rủi ro quá cao |
| D-005 | `*_test.go` white-box giữ cạnh code; embedded cookie KHÔNG XOÁ | Go visibility; go:embed bắt buộc |
| D-006 | Installer giữ `build/windows/installer/` | Wails tự sinh `wails_tools.nsh` ở đó |
| D-009 | Giữ `src/app/main.ts` (KHÔNG làm phẳng) | Rủi ro không cần thiết khi move |
| D-010 | SKIP cross-platform stubs | App dứt khoát Windows-only |
| D-012 | `internal/result` khôi phục từ bản gốc HVR | Package thiếu; bản gốc tìm được ở `D:\Github\HVR\HVR` |
| D-013 | `cmd/` xoá hoàn toàn | Target doc nói giữ `.gitkeep` nhưng xoá thư mục trống là đúng |

---

## 3. Việc còn TREO

> Liệt kê rõ để không bị quên. Không có việc nào chặn app chạy bình thường.

### 3.1 Validate output register/verify (trọng yếu)
- **Vấn đề:** `internal/result` đã khôi phục từ bản gốc (D-012) nhưng **chưa chạy thật** để xác nhận output file (tên file, format nội dung, detail-file) đúng theo nghiệp vụ.
- **Ai làm:** Chủ dự án — cần chạy register/verify với tài khoản thật, kiểm tra file output trong `Config/` tương ứng.
- **Lưu ý:** `docs/flows/02-luong-verify.md` và `add-facebook-reg-version.md` có thể còn tên file cũ (trước restore). Cần review lại nếu tài liệu đó dùng để reference.

### 3.2 QA interactive (cần người dùng)
- **Vấn đề:** S05-D2-T001 ghi "PASS" nhưng phần interactive (mở cửa sổ, click UI, restart) cần chủ dự án xác nhận tay. Automated section 3 đã PASS.
- **Cách làm:** Chạy `wails dev`, đi qua Q1–Q12 trong `workspaces/pm/qa-acceptance-plan.md`.

### 3.3 Rotate credential (bảo mật — KHẨN)
- **Vấn đề:** FB cookies/tokens + Hotmail password/refresh token còn trong **git history** (đã gỡ khỏi HEAD ở S00-D2-T002 nhưng history chưa rewrite).
- **Rủi ro:** Bất kỳ ai clone repo trong khi PRIVATE có thể đọc lại qua `git log -p`.
- **Kế hoạch:** Xem `workspaces/pm/risks.md` — cần `git-filter-repo` (hoặc BFG) + force-push + thông báo mọi người re-clone.
- **Song song:** Rotate ngay credential thật (đổi pass FB, revoke token Hotmail) dù chưa rewrite history.

### 3.4 S05-D1-T002/T003 (Dev 1)
- T002: gate `verifybase` live tests đã DONE (`RUN_LIVE_TESTS`); T003 (unit test `internal/result`) chưa có.
- Không chặn build hay deploy.

---

## 4. Trạng thái sống

Xem `workspaces/pm/current-state.md` — được cập nhật sau mỗi sprint.

Số liệu tại 2026-06-21:
- `go vet ./...` PASS
- `go test ./internal/...` — 2 pre-existing fail (verifybase live + fakeinfo data); còn lại PASS
- `npm run test` — **61/61 PASS**
- `npm run build` PASS
- `wails build` PASS (commit `f50bba2`)
- Platform count: **207** blank-import (không đổi từ baseline)
- `git ls-files` secrets: **sạch** (HEAD)

→ Quay lại: [07-checklist-rui-ro.md](07-checklist-rui-ro.md) · [Onboarding](../onboarding.md)
