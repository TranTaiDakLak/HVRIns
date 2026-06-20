# Dev 1 — Sprint 03: Go test & dọn nợ kỹ thuật

Chạy **song song** với Dev 2 reorg frontend (bạn đụng `internal/**`, Dev 2 đụng `frontend/**` →
không trùng). Mục tiêu: thay các `cmd/*` scratch đã xoá bằng unit test thật + giảm rủi ro còn lại.

---

## S03-D1-T001 — Unit test thay cmd scratch đã xoá
**Bối cảnh:** Sprint 01 Dev 2 đã xoá `cmd/proxycheck`, `cmd/proxytest`, `cmd/testbody`,
`cmd/test_regex`... Chúng vốn là test in-ra-màn-hình. Giờ viết test thật.
**Việc:**
- `internal/proxy/proxy_test.go` (white-box, `package proxy`): table test cho `FormatProxyURL`,
  `RenderSessionIfIsProxyServer`, phân biệt static vs rotate — dùng input giả lập, **không creds thật**.
- Test regex: đặt cạnh hàm/parser sở hữu pattern (tìm bằng `git grep`), `package <đúng package>`.

**Test:**
```powershell
go test ./internal/proxy/...
go test ./internal/...
```
**DONE khi:** test mới PASS, không hardcode credential.

---

## S03-D1-T002 — (Tuỳ chọn) Stub cross-platform
> Chỉ làm nếu muốn `go build ./...` chạy được trên Linux/CI. Nếu app dứt khoát Windows-only, **skip**
> và ghi quyết định vào decision-log (D-009).

**Việc:** tạo `internal/app/cpu_other.go` + `internal/app/portrange_other.go` với build tag
`//go:build !windows`, cung cấp bản no-op/giá trị mặc định cho `getProcessCPUTime`, `getNumCPU`,
`ExpandEphemeralPortRange`.
**Test:**
```powershell
$env:GOOS="linux"; go build ./... ; Remove-Item Env:GOOS    # thử compile cross (không cần chạy)
go build .   # Windows vẫn xanh
```
**DONE khi:** (nếu làm) cross-compile xanh; (nếu skip) đã ghi D-009.

---

## S03-D1-T003 — Rà import cycle settings/adapter
**Bối cảnh (R-13):** `internal/settings/adapter/legacy.go` trước đây "nhân bản" struct từ
`package main` để **tránh** import main. Giờ App đã ở `internal/app` — cần chắc không tạo vòng
`internal/app ↔ internal/settings`.
**Việc:**
```powershell
go build ./...          # cycle sẽ báo lỗi compile ngay
```
Đọc `internal/settings/adapter/legacy.go`: nếu mirror struct giờ thừa (có thể import thẳng type từ
internal/app mà không tạo cycle) → ghi đề xuất vào decision-log; **chỉ refactor nếu an toàn**, không
thì để nguyên + ghi chú.
**Test:** `go build ./...` xanh; `go vet ./...`.
**DONE khi:** xác nhận không có cycle; ghi kết luận vào progress + decision-log.

---

### Sau Sprint 03
Cập nhật progress + task-board + completed-log.
