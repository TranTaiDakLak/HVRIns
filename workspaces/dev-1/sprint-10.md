# Dev 1 — Sprint 10: Integration test phủ nốt App method (Go)

> Tiếp tục "test thật cẩn thận": gọi THẬT các App method sau các nút CHƯA được test runtime ở S09.
> Dùng temp data dir như S09 (`t.Setenv("HVRINS_DATA_DIR", t.TempDir())` + `app.New()`/`SetVersion`).

---

## S10-D1-T001 — Integration test bổ sung
**Việc:** thêm test vào `internal/app/integration_test.go` (hoặc file mới cùng package) cho các method
local (bỏ method cần network/ctx — ghi lý do):
- `GetCookieInitialStatus()` — trả map có key hợp lý (chi tiết hơn S09).
- `GetDatrPoolSize()` / `GetPoolFileSaveCount()` — số ≥ 0, không panic khi pool trống.
- `GetUAPoolsStatus()` / `GetDefaultUACounts()` — trả cấu trúc hợp lệ (key tồn tại; Chrome fixed nếu áp dụng).
- `SetAccountSourceFolder(p)` → `GetAccountSourceFolder()` trả lại đúng p (round-trip; cần profile active nếu logic yêu cầu — dựng profile trước).
- `GetRunStatus()` / `IsRegisterRunning()` / `IsVerifyRunning()` — trạng thái MẶC ĐỊNH (chưa chạy) trả false/idle, không panic.
- `GetDefaultResultPath()` / `GetDefaultCookiePaths()` — non-empty / cấu trúc hợp lý.

**Test:**
```powershell
go test ./internal/app/... -run Integration -v
go test ./internal/...
```
**DONE khi:** ≥5 nhóm test mới gọi thật + assert, PASS; ghi method nào bỏ (network/ctx) + lý do.
Nếu method nào panic khi thiếu ctx/profile → ghi lại (phát hiện bug runtime nút đó).

---

### Sau Sprint 10
Cập nhật task-board (dòng mình) + dev-1/progress.md + completed-log.md. **COMMIT** (đừng để untracked như S09!).
Vòng lặp 5': hết task → ScheduleWakeup(300s) chờ việc mới (KHÔNG tự dừng).
