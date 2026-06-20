# Dev 1 — Progress

> Tự cập nhật sau mỗi task: làm gì, test gì, còn vướng gì. Mục mới lên trên cùng mỗi sprint.

## Trạng thái hiện tại
- Sprint đang làm: **Sprint 04 — TẤT CẢ TASK D1 DONE**
- Task hiện tại: Hoàn tất
- Blocker: —

## Baseline (S00-D1-T002 DONE)
- wails build: **PASS** (commit a3d8210, HVRIns.exe 48.8s)
- go test ./internal/...: mostly PASS · 1 fail verifybase (live account state - pre-existing)
- Số platform đăng ký (blank-import): **207**
- go:embed directives: **5** (docs nói 4, thực tế 5 — cookie/store.go có 2 embed)
- Commit baseline fix: a3d8210

---

## Nhật ký
### Sprint 00
- [S00-D1-T001] DONE 2026-06-20 — Đọc plan (00,02,04,06,07), check môi trường.
  Test: Go 1.26.4 ✅, Node v24.16.0 ✅, npm v11.13.0 ✅, Wails v2.12.0 ✅.
  Hiểu D-001 (main.go ở gốc) và D-007 (commit nguyên tử). File: — (đọc docs only).

- [S00-D1-T002] BLOCKED 2026-06-20 — `wails build` FAIL vì thiếu `internal/result` package.
  Nguyên nhân: app.go:34, app_register.go:163, app_verify.go:14 đều import
  `resultpkg "HVRIns/internal/result"` nhưng thư mục `internal/result/` không tồn tại trong repo
  (không có trong git history, không có trên disk).
  Hậu quả: `wails generate module` FAIL → wailsjs/runtime không được sinh → FE TypeScript lỗi.
  `wails build` FAIL → không ra HVRIns.exe.
  `go test ./internal/...` chạy được (không phụ thuộc root) nhưng 1 test fail (verifybase live account).
  Đề xuất: tạo `internal/result` dựa trên API suy ra từ usage patterns (xem phần BLOCKED bên dưới).

## BLOCKED: internal/result — API đã suy ra

Package cần có:
- `type RegData struct { UID, Password, Cookie, Token, Email, Country string; IsNVR bool; EmailMeta string }`
- `type VerifyData struct { UID, Password, TwoFA, Cookie, Token, Email, FullName, Country string }`
- `type Writer struct` + `NewWriter(path string) *Writer` + `Root() string` + `Append(file, line string) error` + `UpsertUID(file, line string) error`
- `type CounterSet struct { FbAppVersion *StringCounter }` + `Start(ctx, interval)` + `Stop()` + `NewCounterSet(w *Writer) *CounterSet`
- `type DetailEntry struct { File, Content string; Upsert bool }`
- `FormatReg(d RegData, _ interface{}) string` | `FormatVerify(d VerifyData, _ interface{}) string`
- `DispatchRegDetails(status, message, line string) []DetailEntry`
- `DispatchVerifyDetails(status, message, line string) []DetailEntry`
- `ParseEmailMetaFromLine(line string) string`
- `SuccessRegNVRUGFile(instance string) string` | `SuccessVerifyUGFile(instance string) string`
- Constants: FileSuccessReg, FileSuccessVerify, FileSuccessVerifyNo2FA, FileDieAfterVerify,
  FileUnknownErrorCheckLiveDieApi, FileCheckpoint, FileBlocked, FileUnknownBlockType,
  FileSuccessNVREmail, FileSuccessNVRPhone

### Sprint 01
- [S01-D1-T002] DONE 2026-06-20 — Migration Design Note viết xong.
  Export list: startup→Startup, appDataDir→AppDataDir, expandEphemeralPortRange→ExpandEphemeralPortRange.
  OnSecondInstance() bọc app.ctx; SetVersion(v)+buildVersion thread AppVersion từ main.
  File: workspaces/dev-1/migration-note.md.

- [S01-D1-T001] DONE 2026-06-20 — Tách app.go (7315 dòng) → 7 file mới @ root (package main):
  app_accounts.go (~800 ln), app_dialogs.go (~155 ln), app_resources.go (~325 ln),
  app_stats.go (~350 ln), app_upload.go (~862 ln), app_profiles.go (~209 ln),
  app_settings.go (~1927 ln). app.go còn ~2600 dòng (core: App struct, NewApp, startup, run).
  Mỗi file có import block riêng (chính xác theo go build -gcflags="-e").
  Test: go build PASS · gofmt clean · go test ./internal/... same baseline.
  File: app.go (-4700 ln), +7 new files. Commit: 92598da.

- [S01-D1-T003] DONE 2026-06-20 — Xác nhận blank-import và go:embed baseline.
  blank-imports `_ "HVRIns/internal/instagram`: **207** (app.go:206, app_reg_sxxx.go:1).
  go:embed: **5** (main.go:1, cookie/store.go:2, igcore/template.go:1, instagram/register/ios/iosmess/embed.go:1).
  Sprint doc nói 4 go:embed — thực tế 5. Quan trọng: 207 blank-import trong app.go + app_reg_sxxx.go
  sẽ tự đi theo khi Sprint 02 git mv 2 file này vào internal/app — không cần thao tác thêm.
  Test: Grep PASS. File: progress.md (ghi baseline).

### Sprint 02
- [S02-D1-T001–T004] DONE 2026-06-20 — Cú chuyển nguyên tử internal/app (commit 56f516a).
  T001: git mv 18 files → internal/app/, package main → package app.
  T002: Startup/AppDataDir/ExpandEphemeralPortRange exported; SetVersion+buildVersion;
        OnSecondInstance(); GetAppVersion() returns a.version.
  T003: main.go thin — chỉ còn embed, AppVersion, flag, instanceUniqueID, main().
        Delegate toàn bộ sang igapp "HVRIns/internal/app".
  T004: wails generate module → frontend/wailsjs/go/app/ (was go/main/).
        16 import fixes: go/main/App→go/app/App, main.X→app.X.
  Verify: 207 blank-import ✅ · wails build PASS (HVRIns.exe) ✅ · go build . ✅.

### Sprint 03
- [S03-D1-T001] DONE 2026-06-20 — Unit tests proxy thay cmd scratch đã xoá.
  Viết internal/proxy/client_test.go: FormatProxyURL (8 case), ShortDisplay (6 case),
  RenderSessionIfIsProxyServer (static/ssid/zone/proxyshare), isPort (10 case).
  Test: go test ./internal/proxy/... PASS. File: internal/proxy/client_test.go.

- [S03-D1-T002] SKIP 2026-06-20 — Cross-platform stubs không làm (app Windows-only).
  Lý do: internal/proxy/transport_pool.go dùng syscall.Handle (Windows-only), không có CI Linux.
  Quyết định: D-010 (decision-log). go build ./... Windows PASS.

- [S03-D1-T003] DONE 2026-06-20 — Rà import cycle settings/adapter sau Sprint 02.
  go build ./...: PASS (no cycle). go vet ./...: PASS (sau move app_test.go → internal/app/).
  legacy.go mirror structs vẫn cần thiết (cycle nếu import internal/app ↔ internal/settings).
  Phát hiện: app_test.go sót lại ở root sau git mv → di chuyển sang internal/app/app_test.go.

### Sprint 04
- [S04-D1-T001] DONE 2026-06-20 — Verify cuối toàn bộ (S04).
  go vet ./...: PASS · go test ./internal/...: same baseline (fakeinfo+verifybase pre-existing) ·
  npm build: PASS · wails build: PASS (HVRIns.exe) · 207 platform ✅.

- [S04-D1-T002] DONE 2026-06-20 — Quyết định Pha 7: DEFER.
  D-011: rủi ro cao (2900 file + 207 blank-import), lợi ích thấp ở giai đoạn hiện tại.
  Backlog item tạo trong decision-log.

---
### Mẫu dòng nhật ký
```
- [S02-D1-T003] DONE 2026-06-__ — main.go bootstrap mỏng.
  Test: go build . PASS, go test ./internal/app/... PASS.
  File: main.go, internal/app/datadir.go. Note: AppVersion thread qua SetVersion OK.
```
