# Dev 1 — Sprint 09: Integration test GỌI THẬT App method (Go runtime)

> "Test thật cẩn thận" phía Go: gọi THẬT các method backend mà nút FE gọi, verify hành vi (đọc lại
> đúng dữ liệu), không chỉ kiểm tồn tại. Mọi verify trên Windows.

---

## S09-D1-T001 — Integration test App method (không cần network)
**Bối cảnh:** các nút FE gọi App method qua binding. Đã verify method TỒN TẠI (S08). Giờ verify CHẠY ĐÚNG.
**Cách cô lập runtime:** tạo data dir tạm để không đụng data thật:
```go
t.Setenv("HVRINS_DATA_DIR", t.TempDir())   // appDataDir() honor env này
a := app.New(); a.SetVersion("test")        // hoặc NewApp(); KHÔNG cần Wails ctx cho method thuần data
```
> Method nào cần `a.ctx` (emit event) / network (register/verify/clonehv/mail) → BỎ QUA, ghi chú lý do.
> Chỉ test method thao tác file/data cục bộ.

**Cover (chọn cái cô lập được, ưu tiên cái nút FE hay gọi):**
- Settings: `SaveSettings(x)` → `LoadSettings()` trả lại đúng x (round-trip).
- Accounts: `ImportAccounts`/`LoadAccountsFromFile` (file tạm) → `ListAccounts` đúng số/đúng field → `DeleteAccounts` → list giảm.
- Profile: `CreateProfile("p1")` → `ListProfiles` chứa p1 → `SetActiveProfile`/`DeleteProfile`.
- Proxy: `SaveProxyList(list)` → `LoadProxyList()` trả đúng list.
- Result: ghi 1 record (qua API có sẵn) → đọc file trong data dir thấy đúng định dạng.
- `GetCookieInitialStatus` / `GetDefaultResultPath` / `GetAccountSourceFolder` trả giá trị hợp lý.

**Test:**
```powershell
$env:RUN_LIVE_TESTS=$null
go test ./internal/app/... -run Integration -v
go test ./internal/...   # toàn bộ vẫn GREEN
```
**DONE khi:** có ≥4 nhóm integration test GỌI THẬT method, assert hành vi (không chỉ existence), PASS;
ghi rõ method nào bỏ qua (cần ctx/network) + lý do. KHÔNG hardcode credential.

> Lưu ý: nếu method có chữ ký cần kiểu phức tạp, dựng input tối thiểu hợp lệ. Nếu một method panic khi
> thiếu ctx → đó là phát hiện đáng giá: ghi vào progress + báo (có thể là bug runtime nút đó).

---

### Sau Sprint 09
Cập nhật task-board (dòng của mình) + dev-1/progress.md + completed-log.md. Commit (KHÔNG push).
Vòng lặp 5': hết task → ScheduleWakeup(300s) chờ việc mới (KHÔNG tự dừng).
