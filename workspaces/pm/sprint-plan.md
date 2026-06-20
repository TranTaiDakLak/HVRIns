# Sprint Plan — Tái cấu trúc HVRIns

> Bao phủ TOÀN BỘ công việc còn lại của `docs/rebuild/`. 5 sprint (00→04), 2 dev song song.
> Nguyên tắc: hạn chế 2 dev sửa trùng file. Mỗi task có ID `S{SPRINT}-D{DEV}-T{NUMBER}`.

## Phân vai
- **Dev 1 — Go / Build lead.** Sở hữu *critical path*: baseline, tách `app.go`, cú chuyển
  `internal/app`, regenerate bindings, AppVersion, viết Go test.
- **Dev 2 — Cleanup / Infra / Frontend.** Secrets, dọn rác non-Go, `config/sample`, docs,
  README/CLAUDE, rồi tái cấu trúc frontend.

## Ranh giới file (tránh conflict)
| Vùng file | Chủ sở hữu |
|-----------|-----------|
| Root `*.go`, `main.go`, `internal/app/**` | **Dev 1 only** |
| `.gitignore`, `go.mod` (tidy), `cmd/**`, `scripts/**`, `docs/**`, `Config/**`, `.vscode/**` | **Dev 2 only** |
| `frontend/src/bridge/wails/*.ts` (sửa import binding) | **Dev 1** (Sprint 02) |
| `frontend/src/**` (rename services/features, alias) | **Dev 2** (Sprint 03, SAU D1-S02) |
| `internal/proxy`, `internal/**` (test mới) | **Dev 1** (Sprint 03) |
| `README.md`, `CLAUDE.md`, `wails.json`, `tests/**` | **Dev 2** |

## Sơ đồ phụ thuộc
```
S00 D1 baseline ─┐
S00 D2 secrets  ─┤ (song song)
                 ▼
S01 D1 tách app.go + design note  ║  S01 D2 dọn rác non-Go + config/sample   (song song)
                 ▼
S02 D1 ⭐ cú chuyển internal/app + bindings   ║  S02 D2 README/CLAUDE/wails.json/tests scaffold
                 ▼  (D1-S02 xong là điều kiện mở khoá D2-S03)
S03 D2 reorg frontend (alias→services→features)  ║  S03 D1 viết Go test thay cmd scratch
                 ▼
S04 D1 verify cuối + quyết định Pha 7  ║  S04 D2 rà gốc gọn + kế hoạch history rewrite
```

---

## Sprint 00 — Setup & Safety
**Mục tiêu:** có green baseline + gỡ secret. Không di chuyển code logic.
- Dev 1: `S00-D1-T001` đọc plan & môi trường · `S00-D1-T002` baseline (wails build, go test, ghi số platform) · `S00-D1-T003` dọn go.mod/go.sum dirty.
- Dev 2: `S00-D2-T001` đọc plan & môi trường · `S00-D2-T002` secrets `git rm --cached` + `.gitignore` · `S00-D2-T003` ghi checklist rotate creds vào risks.md.
**DoD:** wails build PASS; số platform đã ghi; secret không còn track; wails build vẫn xanh.

## Sprint 01 — Prep & Cleanup (song song, không đụng nhau)
**Mục tiêu:** chuẩn bị cho cú chuyển + dọn sạch phần non-Go.
- Dev 1: `S01-D1-T001` tách `app.go` → 7 file con (vẫn package main @ root) · `S01-D1-T002` viết Migration Design Note (export list, app.ctx, AppVersion) → decision-log · `S01-D1-T003` xác nhận vị trí 207 blank-import + 4 go:embed.
- Dev 2: `S01-D2-T001` xoá python lạc + __pycache__ + .gitignore · `S01-D2-T002` move docs (guide/testing/archive/flows/specs) · `S01-D2-T003` build.bat→scripts + scripts/legacy · `S01-D2-T004` dọn cmd/ (icongen→tools, xoá scratch, go mod tidy) · `S01-D2-T005` config/sample + sửa launch.json.
**DoD:** `go build .` xanh sau tách app.go; gốc repo bớt rác; wails build xanh.

## Sprint 02 — ⭐ Cú chuyển internal/app (critical path)
**Mục tiêu:** App logic vào `internal/app`, main.go mỏng, bindings & FE import cập nhật.
- Dev 1: `S02-D1-T001` git mv + đổi package · `S02-D1-T002` export + bọc ctx + thread AppVersion · `S02-D1-T003` main.go mỏng + go vet/test/build · `S02-D1-T004` wails generate + sửa 10 import bridge/wails + wails build/dev + verify version & platform count.
- Dev 2: `S02-D2-T001` viết README.md gốc · `S02-D2-T002` viết lại CLAUDE.md + author wails.json · `S02-D2-T003` scaffold tests/{go,frontend}.
**DoD:** wails dev mở app, FE gọi backend qua binding mới; GetAppVersion != "dev"; platform count == baseline.

## Sprint 03 — Frontend reorg (D2) + Go test (D1)
**Mục tiêu:** FE theo chuẩn features/services; thay cmd scratch bằng unit test thật.
- Dev 2 (sau khi D1-S02 DONE): `S03-D2-T001` bật alias @/ + convert import · `S03-D2-T002` xoá stub + làm phẳng src/app · `S03-D2-T003` bridge→services (giữ độ sâu) · `S03-D2-T004` modules→features + gom pages + script vitest.
- Dev 1: `S03-D1-T001` unit test internal/proxy + regex (thay cmd scratch) · `S03-D1-T002` (tuỳ chọn) stub `_other.go` cross-platform · `S03-D1-T003` rà import cycle settings/adapter.
**DoD:** npm run build + wails dev xanh sau mỗi feature; go test pass.

## Sprint 04 — Finalize & Verify
**Mục tiêu:** chốt, verify end-to-end, cập nhật toàn bộ file quản lý.
- Dev 1: `S04-D1-T001` verify cuối (wails build/dev, platform, version) + cập nhật board/log · `S04-D1-T002` quyết định Pha 7 (defer/làm) → decision-log.
- Dev 2: `S04-D2-T001` rà gốc gọn + review-checklist · `S04-D2-T002` (tuỳ chọn) kế hoạch rewrite history cho secrets.
**DoD:** review-checklist tick hết mục bắt buộc; gốc repo chỉ còn file/thư mục chuẩn.
**Trạng thái:** ✅ HOÀN TẤT 2026-06-20 (34 task DONE + 1 SKIP). Cấu trúc đã verify.

## Sprint 05 — Validation, QA & Hardening (giao 2026-06-21)
**Bối cảnh:** cấu trúc xong nhưng còn 2 việc "build-xanh-không-thấy": (1) `internal/result` là package
TÁI TẠO bằng suy luận (format/filename/dispatch chưa chắc khớp bản gốc); (2) chưa QA chức năng thật.
**Mục tiêu:** đảm bảo HÀNH VI đúng + thêm test thật + đồng bộ docs.
- Dev 1: `S05-D1-T001` ⭐ validate/khôi phục internal/result · `S05-D1-T002` xử lý test fail verifybase · `S05-D1-T003` unit test khóa hành vi result.
- Dev 2: `S05-D2-T002` viết FE test thật (làm trước, độc lập) · `S05-D2-T001` chạy QA acceptance Q1–Q12 (CHỜ S05-D1-T001) · `S05-D2-T003` audit cấu trúc + đồng bộ docs.
**Phụ thuộc:** S05-D2-T001 (QA register/verify) đợi S05-D1-T001 (result đúng). Còn lại song song.
**DoD:** internal/result đúng/được ghi gap rõ; QA Q1–Q12 + RG-1..5 đạt; có FE test thật PASS; docs khớp thực tế.

### Cần CHỦ DỰ ÁN quyết (không phải dev)
1. Có source gốc `internal/result` ở đâu không? (quyết định khôi phục vs validate-suy-luận ở S05-D1-T001)
2. Rotate credential đã lộ (FB/Hotmail) — thủ công.
3. Rewrite git history cho secrets — cần đồng thuận team.
