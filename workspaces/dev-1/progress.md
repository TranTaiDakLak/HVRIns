# Dev 1 — Progress

> Tự cập nhật sau mỗi task: làm gì, test gì, còn vướng gì. Mục mới lên trên cùng mỗi sprint.

## Trạng thái hiện tại
- Sprint đang làm: **Sprint 08 — S08-D1-T001 DONE**
- Task hiện tại: HẾT VIỆC — không còn TODO/BLOCKED của Dev 1 (task-board: DONE 44, TODO 0)
- Blocker: —
- Trạng thái suite: `wails build` PASS · `go vet ./internal/app/...` PASS · bindings 90 method đồng bộ

## Baseline (S00-D1-T002 DONE)
- wails build: **PASS** (commit a3d8210, HVRIns.exe 48.8s)
- go test ./internal/...: mostly PASS · 1 fail verifybase (live account state - pre-existing)
- Số platform đăng ký (blank-import): **207**
- go:embed directives: **5** (docs nói 4, thực tế 5 — cookie/store.go có 2 embed)
- Commit baseline fix: a3d8210

---

## Nhật ký
### Sprint 08
- [S08-D1-T001] DONE 2026-06-21 — Regenerate & verify Wails bindings.
  wails generate module yêu cầu bin/dev/wails.json (chỉ có trong wails dev) → dùng wails build thay.
  wails build PASS (23s) → "Generating bindings: Done." — bindings regenerated, 0 file thay đổi (đã sync).
  So sánh: 91 Go exported method; Startup bị exclude đúng (lifecycle ctx.Context, không bind được).
  App.d.ts: 90 method = khớp 100%. go vet PASS. Không cần commit (bindings không đổi).
  Danh sách 90 method ghi trong completed-log.

### Sprint 07 — Idle
- Dev 1 idle 08:52 2026-06-21 — chờ PM giao việc (task-board: DONE 42, TODO 0)
- Dev 1 idle 08:47 2026-06-21 — chờ PM giao việc (task-board: DONE 42, TODO 0)
- Dev 1 idle 08:42 2026-06-21 — chờ PM giao việc (task-board: DONE 42, TODO 0)
- Dev 1 idle 08:37 2026-06-21 — chờ PM giao việc (task-board: DONE 42, TODO 0)
- Dev 1 idle 08:32 2026-06-21 — chờ PM giao việc (task-board: DONE 42, TODO 0)
- Dev 1 idle 08:27 2026-06-21 — chờ PM giao việc (task-board: DONE 42, TODO 0)
- Dev 1 idle 08:22 2026-06-21 — chờ PM giao việc (task-board: DONE 42, TODO 0)

### Sprint 07
- [S07-D1-T001] DONE 2026-06-21 — White-box test helper thuần internal/app.
  Thêm internal/app/helpers_test.go (60 test case mới, package app):
  isGUID (10), isAlphaNumeric (9), hasLetterAndDigit (7), isAllDigits (8),
  extractCUserFromCookie (6), extractFBAV (6), verifyPlatformDisplayName (6), autoDetectAccount (8).
  Bỏ qua hàm cần a.ctx/network/Wails runtime (ghi rõ trong header file).
  Test: go test ./internal/app/... → 96 tests PASS; go test ./internal/... GREEN; go vet PASS.

### Sprint 05
- [S05-D1-T003] DONE 2026-06-21 — Viết internal/result/result_test.go khóa hành vi.
  Test gốc đã có FormatReg/FormatVerify/UpsertUID → file này khóa GAP + hợp đồng chéo (tên riêng):
  TestParseEmailMetaFromLine (gap), TestFormatReg_EmailMetaRoundTrip (writer↔reader, meta chứa "|"),
  TestFormatReg/Verify_FieldOrderLock (vị trí field, timestamp cố định), TestUpsertUID_BehaviorLock (t.TempDir).
  Test: go test ./internal/result/... PASS; full suite GREEN; wails build PASS. Commit: e0b3031.

- [S05-D1-T002] DONE 2026-06-21 — Xử lý 16 test Go fail (suite về GREEN).
  verifybase (2 test): gọi FB Graph thật + token hết hạn → non-deterministic. Đổi guard testing.Short()
  → opt-in RUN_LIVE_TESTS=1 + lý do skip (commit c559808).
  fakeinfo (15 test): data Config/* runtime gitignored vắng lúc go test → dataset rỗng → fail. Thêm helper
  skipIfNoConfigData (testdata_guard_test.go) áp 14 test data-dependent. Bonus FIX: TestUAOverridePath là
  test cũ lệch code (Request_UG.txt→PC_UG.txt) → sửa expected (commit 716d0d3).
  Test: go test ./internal/... GREEN. PM REVIEW PASS.

- [S05-D1-T001] DONE 2026-06-21 — KHÔI PHỤC BẢN GỐC `internal/result` (thay bản tái tạo).
  Chủ dự án chỉ `D:\Github\HVR\` → tìm thấy bản gốc đầy đủ ở `D:\Github\HVR\HVR\internal\result`
  (module HVR — repo HVRIns tái cấu trúc ra; HVRIns3 là bản copy giống hệt). Bản gốc tự-chứa
  (chỉ stdlib, KHÔNG import HVR/internal/...) → khôi phục verbatim, KHÔNG cần đổi module path.
  Thay 5 file tái tạo (counter/dispatch/files/format/writer) bằng 7 file gốc
  (counter/counters/dispatch/errorlog/format/store/writer) + 2 test gốc (counter_test, store_test).
  Bản tái tạo SAI: (1) 3 filename constant — `DieAfterVerify.txt`→**`Die.txt`**,
  `UnknownError_CheckLiveDie.txt`→**`Unknown.txt`**, `UnknownBlock.txt`→**`UnknownReg.txt`**
  (consumer `app_register.go:4536` + `verify/web/verify.go:616-618` hardcode tên gốc → bản tái tạo
  từng tạo bug ẩn); (2) `dispatch.go` STUB trả nil → khôi phục đầy đủ (15+ detail-file phân loại lỗi);
  (3) thiếu `counters.go`/`errorlog.go`/`store.go`.
  Validate (workflow 5-lens): byte-for-byte giống HVR (drift), consumer khớp (autoDetectAccount parse
  theo pattern không theo vị trí), dispatch đủ constant, 0 critical.
  Test: `go build .` ✅ · `go test ./internal/result/...` ✅ (test gốc, gồm TestFormatReg/FormatVerify/
  UpsertUID) · `go test ./internal/app/...` ✅ · `go vet ./internal/result/... ./internal/app/...` ✅ ·
  `wails build` ✅ (HVRIns.exe). File: internal/result/* (9 file). Decision: D-012 (updated).
  Lưu ý cho Dev 2: docs/flows/*.md còn ghi tên file cũ (DieAfterVerify.txt/UnknownErrorCheckLiveDieApi.txt)
  → cần sync sang Die.txt/Unknown.txt (vùng docs của Dev 2, KHÔNG sửa từ đây).

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
