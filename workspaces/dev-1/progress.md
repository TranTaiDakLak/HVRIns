# Dev 1 — Progress

> Tự cập nhật sau mỗi task: làm gì, test gì, còn vướng gì. Mục mới lên trên cùng mỗi sprint.

## Trạng thái hiện tại
- Sprint đang làm: **Sprint 00**
- Task hiện tại: S00-D1-T002 **BLOCKED**
- Blocker: `internal/result` package không tồn tại trong repo nhưng được import bởi 3 file gốc

## Baseline (điền ở S00-D1-T002)
- wails build: **FAIL** (`app.go:34:2: package HVRIns/internal/result is not in std`)
- go test ./internal/...: FAIL (1 test verifybase về live account; plus root không compile)
- Số platform đăng ký: ___ (chưa lấy được vì build fail)
- 207 blank-import? ___ · 4 go:embed? ___

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
- [S02-D1-T003] DONE 2026-06-__ — main.go bootstrap mỏng.
  Test: go build . PASS, go test ./internal/app/... PASS.
  File: main.go, internal/app/datadir.go. Note: AppVersion thread qua SetVersion OK.
```
