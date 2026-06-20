# Dev 2 — Sprint 07: Hardening coverage (FE global stores)

> Tự nguyện, giá trị thật. Mục tiêu: test các Pinia store toàn cục (mới chỉ test composables +
> accounts store). Mọi verify trên Windows.

---

## S07-D2-T001 — Test các global Pinia store
**Bối cảnh:** đã test composables (useDataGrid/useSelection/useColumnVisibility/useContextMenu) +
useAccountsStore. Còn `app.store`, `preferences.store`, `uploadLog.store` chưa có test.
**Việc:** viết test vitest (setActivePinia(createPinia())) cho các store toàn cục:
- `app.store`: state khởi tạo, action chính, getter.
- `preferences.store`: set/persist preference, default.
- `uploadLog.store`: thêm/clear log, giới hạn (nếu có).
Mock `services/` nếu store gọi (đã có services/mock/).
**Test:**
```powershell
npm --prefix frontend run test
```
**DONE khi:** ≥2 store toàn cục có test PASS; tổng test tăng; không flaky.

> Nếu một store quá mỏng/không có logic đáng test → ghi chú và bỏ qua store đó (không bịa test rỗng).

---

### Sau Sprint 07
Cập nhật task-board (dòng của mình) + dev-2/progress.md + completed-log.md. Commit (KHÔNG push).
Hết việc → vòng lặp 5 phút: ScheduleWakeup(300s) kiểm tra task-board; không còn task Dev 2 →
in "DEV 2: HẾT VIỆC" và dừng. KHÔNG lấn việc Dev 1.
