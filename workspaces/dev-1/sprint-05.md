# Dev 1 — Sprint 05: Validation & Hardening (Go)

> Cấu trúc đã xong. Sprint này lo **đúng hành vi** (không chỉ "build xanh"), tập trung vào package
> `internal/result` mà bạn đã phải tái tạo. Mọi verify trên Windows; cổng = `wails build` + `go test`.

---

## S05-D1-T001 — ⭐ Validate / khôi phục `internal/result` (ƯU TIÊN CAO NHẤT)
**Bối cảnh:** `internal/result` vốn bị import nhưng KHÔNG có trong repo gốc; bạn đã dựng lại để
unblock (a3d8210). `wails build` xanh nhưng hành vi chưa được kiểm chứng. 3 vùng rủi ro:
- `format.go` (`FormatReg`/`FormatVerify`): thứ tự field/format **suy luận**.
- `files.go`: tên file constant **suy luận**.
- `dispatch.go`: **STUB trả nil** — mất logic ghi detail-file theo sub-status.

**Việc:**
1. **Hỏi PM/chủ dự án trước:** có source gốc `internal/result` ở đâu không (máy khác/backup/repo
   nguồn). NẾU CÓ → khôi phục bản gốc, bỏ bản suy luận, build lại. (Xem current-state.md mục "Cần chủ dự án quyết".)
2. NẾU KHÔNG có bản gốc → **validate bằng đối chiếu phía ĐỌC** (consumer của các file kết quả):
   - Tìm nơi đọc lại: `popAccountFromFolder` (app), FE legacy-import wizard parser, `ParseEmailMetaFromLine`,
     `docs/flows/05-config-data-pools.md`, và mọi chỗ `git grep` `SuccessReg`/`SuccessVerify`/`|MM:`.
   - Đối chiếu **thứ tự field** `FormatReg`/`FormatVerify` với cách parser tách field. Sửa nếu lệch.
   - Đối chiếu **tên file** trong `files.go` với tên mà code/đọc/operator mong đợi.
3. **dispatch.go:** dò `docs/flows/*` + cách app gọi `DispatchRegDetails/DispatchVerifyDetails` để biết
   detail-file gốc cần ghi gì (checkpoint/blocked/die...). Khôi phục logic; nếu không đủ thông tin để
   khôi phục an toàn → GIỮ stub nhưng thêm comment `// TODO(result): dispatch chưa khôi phục — xem ...`
   và ghi rõ "behavior gap" để chủ dự án biết.
**Test:**
```powershell
go vet ./internal/result/... ./internal/app/...
npm --prefix frontend run build ; go build .
wails build
```
**DONE khi:** format/filename khớp consumer (hoặc khôi phục bản gốc); dispatch khôi phục HOẶC ghi rõ
gap; ghi quyết định vào `pm/decision-log.md` (D-012) + cập nhật `current-state.md`.

> ⚠ Đây là rủi ro hành vi lớn nhất còn lại. Đừng đánh DONE chỉ vì build xanh.

---

## S05-D1-T002 — Xử lý test fail ở verifybase
**Việc:**
```powershell
go test ./internal/instagram/verify/... 2>&1 | Select-String -Pattern "FAIL|PASS|ok " | Select-Object -Last 20
```
- Xác định test nào fail, vì sao. Nếu phụ thuộc **live account/network state** (không deterministic)
  → đánh `t.Skip("requires live account/network")` + ghi lý do; KHÔNG để fail âm thầm.
- Nếu là regression do migration → fix.
**Test:** `go test ./internal/...` xanh, hoặc mọi fail còn lại đều có `t.Skip` + lý do.
**DONE khi:** không còn fail "bí ẩn"; trạng thái test giải thích được.

---

## S05-D1-T003 — Unit test khóa hành vi `internal/result`
**Việc:** viết `internal/result/result_test.go` (package result, white-box):
- `FormatReg`/`FormatVerify`: kiểm thứ tự field + suffix `|MM:` (dùng input cố định; nếu có phần
  timestamp, tách/giả lập để test ổn định).
- `Writer.UpsertUID`: append khi UID mới, replace khi UID trùng (dùng `t.TempDir()`).
- `ParseEmailMetaFromLine`: case `MM:` thuần + case suffix `|MM:` + case không phải meta.
**Test:** `go test ./internal/result/...` PASS.
**DONE khi:** test mới PASS, khóa được format/writer/parse để lần sau đổi không vỡ âm thầm.

---

### Sau Sprint 05
Cập nhật `pm/task-board.md` (chỉ dòng của bạn) → `dev-1/progress.md` → `pm/completed-log.md` →
`pm/decision-log.md` (D-012). Commit theo task (KHÔNG push). Báo PM khi S05-D1-T001 DONE để Dev 2
chạy QA register/verify (Q3/Q7).
