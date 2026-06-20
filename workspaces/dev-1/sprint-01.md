# Dev 1 — Sprint 01: Tách app.go & chuẩn bị cú chuyển

Chạy **song song** với Dev 2 (Dev 2 dọn cmd/docs/scripts — không đụng file của bạn).
Mục tiêu: làm `app.go` (317KB) nhỏ lại + lên thiết kế chi tiết cho Sprint 02, **chưa** đổi package.

> 💡 Vì sao tách app.go ngay bây giờ (khi còn ở gốc, `package main`)? Vì tách file trong **cùng
> một package** là thao tác thuần cosmetic — KHÔNG đổi import nào, compile y nguyên. Làm trước giúp
> Sprint 02 (di chuyển) gọn và dễ review hơn.

---

## S01-D1-T001 — Tách `app.go` theo trách nhiệm
**Việc:** trong cùng thư mục gốc, vẫn `package main`, cắt `app.go` thành các file con. Gợi ý nhóm
(điều chỉnh theo thực tế method):
| File mới | Nội dung |
|----------|----------|
| `app.go` | `type App`, `NewApp`, `startup`, field & wiring chung |
| `app_accounts.go` | Account/AccountFilter + ListAccounts/ImportAccounts/Delete/Load |
| `app_settings.go` | SettingsData/GeneralConfig/InteractionConfig + Load/Save |
| `app_profiles.go` | Create/Clone/Delete/Rename/SetActiveProfile |
| `app_upload.go` | StartUploadSite/StopUploadSite/UploadStats |
| `app_stats.go` | RegStatRow/MailDomainStatRow + GetRegStats/GetVerifyStats |
| `app_resources.go` | GetResourceUsage/DebugMemory/ForceMemoryCleanup |
| `app_dialogs.go` | OpenFolderDialog/ReadTextFile/ValidatePath |

Quy tắc:
- Chỉ **cắt-dán** khối hàm/type; KHÔNG sửa logic.
- Mỗi file mới mở đầu bằng `package main` + import cần thiết (chạy `goimports`/`gofmt`).
- **Giữ nguyên** các dòng `//go:embed` và blank-import ở `app.go` (đừng tách rời chúng nếu khó).

**Test:**
```powershell
gofmt -l .                 # không file nào cần format
npm --prefix frontend run build ; go build .   # compile xanh
go test ./internal/...
```
**DONE khi:** build + test xanh, app.go nhỏ đi rõ rệt, không đổi hành vi.

> ⚠ Đừng tách quá nhỏ/quá cầu kỳ. Mục tiêu là dễ đọc, không phải nghệ thuật.

---

## S01-D1-T002 — Migration Design Note (cho Sprint 02)
**Việc:** soạn ghi chú ngắn trong `workspaces/dev-1/migration-note.md` (tạo mới) gồm:
1. **Danh sách cần export** (vì `main.go` sẽ gọi xuyên package):
   - `startup` → `Startup`
   - `appDataDir()` → `AppDataDir()`
   - `expandEphemeralPortRange()` → `ExpandEphemeralPortRange()`
   - Xác nhận `NewApp`, `IsConfirmedQuit`, `EmitQuitConfirm`, `RequestQuit` đã public.
2. **Xử lý `app.ctx`**: `main.go` dùng `app.ctx` trong `OnSecondInstanceLaunch`. Thiết kế method
   export `func (a *App) OnSecondInstance()` gói toàn bộ logic show/unminimise vào trong (đừng
   expose raw `ctx`).
3. **Thread `AppVersion`** (xem D-003): thêm `func (a *App) SetVersion(v string)` + field `version`;
   `datadir.go` đọc `a.version` thay cho biến package `AppVersion`. `main.go` gọi `a.SetVersion(AppVersion)`.
4. Liệt kê **tất cả** chỗ `main.go` hiện gọi vào App (grep) để không sót khi export.

Sau khi soạn, copy phần quyết định vào `workspaces/pm/decision-log.md` (nếu có điều chỉnh so với D-003).

**Test:** — (tài liệu). **DONE khi:** note đủ 4 mục, đã đối chiếu code thật.

---

## S01-D1-T003 — Xác nhận blank-import & go:embed
**Việc:** ghi vào `progress.md` để Sprint 02 không làm hỏng:
```powershell
git grep -n "_ \"HVRIns/internal/instagram" -- app.go app_reg_sxxx.go | wc -l   # ~207
git grep -n "go:embed" -- "*.go"     # phải thấy đúng 4 chỗ
```
Xác nhận: 207 blank-import nằm trong `app.go` + `app_reg_sxxx.go` → **tự đi theo** khi move 2 file
này ở Sprint 02 (không cần thao tác thêm). go:embed của main.go ở lại gốc.

**Test:** số liệu khớp (207 ± theo thực tế; 4 go:embed). **DONE khi:** đã ghi số vào progress.md.

---

### Sau Sprint 01
Cập nhật progress + task-board + completed-log. Chờ Dev 2 xong Sprint 01 (không bắt buộc) rồi bắt Sprint 02.
