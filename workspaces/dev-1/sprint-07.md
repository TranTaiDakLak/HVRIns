# Dev 1 — Sprint 07: Hardening coverage (Go bridge layer)

> Tự nguyện, giá trị thật. KHÔNG phải lỗi chặn. Mục tiêu: khóa hành vi các helper THUẦN trong
> internal/app (bridge layer FE-facing hiện gần như chưa có test). Mọi verify trên Windows.

---

## S07-D1-T001 — White-box test cho helper thuần trong internal/app
**Bối cảnh:** internal/app có ~87 method exported nhưng mới chỉ `app_test.go` (test isVerifiableAccountFile…).
Nhiều helper THUẦN (không cần Wails runtime/ctx/network) có thể test trực tiếp.
**Việc:** thêm test (package app, white-box) cho các helper thuần — chọn cái có giá trị + dễ cô lập:
- Parse/format account (vd định dạng dòng account, tách field, lọc theo AccountFilter).
- Settings normalize/merge (nếu có hàm thuần xử lý SettingsData/GeneralConfig/InteractionConfig).
- Bất kỳ helper map/format nào dùng cho stats/upload mà không cần ctx.
Quy tắc: CHỈ test hàm không cần `a.ctx`/network/Wails runtime. Hàm cần ctx → bỏ qua (ghi chú lý do).
**Test:**
```powershell
go test ./internal/app/...
go vet ./internal/app/...
```
**DONE khi:** thêm ≥1 nhóm test mới PASS cho helper thuần; suite internal vẫn GREEN; ghi rõ helper nào
cố ý bỏ (cần ctx/network).

> Nếu rà thấy hầu hết helper đều dính ctx/network và không có gì test được thêm một cách có giá trị →
> ghi nhận điều đó vào progress + decision-log và đánh DONE (kết luận "đã đạt trần test thực tế").
> KHÔNG bịa test vô nghĩa.

---

### Sau Sprint 07
Cập nhật task-board (dòng của mình) + dev-1/progress.md + completed-log.md. Commit (KHÔNG push).
Hết việc → theo vòng lặp 5 phút: ScheduleWakeup(300s) kiểm tra task-board; nếu không còn task Dev 1 →
in "DEV 1: HẾT VIỆC" và dừng.
