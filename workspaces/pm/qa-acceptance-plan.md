# QA Acceptance Plan — test chức năng thật (Sprint 04)

> "Build xanh" CHƯA đủ. Phải xác nhận app **chạy đúng như trước** sau tái cấu trúc. Test thủ công
> trên Windows qua `wails dev` (hoặc bản `wails build`). Mục tiêu: không hồi quy chức năng.
>
> Dùng dữ liệu **giả/đã che**, KHÔNG dùng credential thật.

## 0. Tiền điều kiện
- [ ] Chạy `wails dev` → cửa sổ "Hạ Vũ" mở, không lỗi console.
- [ ] So sánh với hành vi baseline (ghi chú lúc Sprint 00 nếu có).

## 1. Smoke chức năng theo từng màn hình (FE ↔ App method)

| # | Màn hình (page) | Thao tác kiểm | Method backend liên quan | Kỳ vọng |
|---|------------------|---------------|--------------------------|---------|
| Q1 | App shell | Mở app, sidebar/header/status bar hiển thị | `GetResourceUsage` (status bar) | RAM/CPU cập nhật ở status bar |
| Q2 | Accounts | Grid render, chọn dòng, double-click | `ListAccounts` | Grid có cột, chọn được, không lỗi |
| Q3 | Accounts | Mở dialog Import, thử import file mẫu | `ImportAccounts` | Import chạy, đếm dòng đúng |
| Q4 | General Settings | Đổi 1 setting → lưu → reload app | `SaveSettings`/`LoadSettings` | Giá trị **giữ nguyên** sau reload (đọc/ghi đúng appDataDir) |
| Q5 | Proxy Settings | Nhập proxy mẫu, kiểm hiển thị | (proxy parse) | Format/hiển thị đúng |
| Q6 | Interaction Setup | Mở, chỉnh, lưu | `InteractionConfig` save | Lưu/đọc lại đúng |
| Q7 | Reg Stats | Mở trang thống kê | `GetRegStats`/`GetVerifyStats` | Bảng render, không crash |
| Q8 | Profiles | Tạo/đổi tên/đặt active profile | `CreateProfile`/`SetActiveProfile` | Profile tạo được, chuyển active OK |
| Q9 | Upload Site | Mở, start/stop (nếu an toàn) | `StartUploadSite`/`StopUploadSite` | Không lỗi binding |
| Q10 | Folder dialog | Nút chọn thư mục | `OpenFolderDialog` | Dialog mở, trả path |
| Q11 | Quit | Nhấn X → ConfirmDialog | `OnBeforeClose`/`EmitQuitConfirm`/`RequestQuit` | Hỏi xác nhận; confirm → thoát |
| Q12 | Second instance | Mở app lần 2 (cùng thư mục) | `OnSecondInstance` | App đang chạy hiện lên (không mở bản 2) |

> Q11/Q12 quan trọng vì liên quan `app.ctx` đã được bọc thành method export khi tách (R-6). Nếu
> Q12 không hiện cửa sổ → logic ctx bị hỏng khi move.

## 2. Kiểm hồi quy hệ thống (không nhìn thấy trên UI)
- [ ] **RG-1 AppVersion**: bản `scripts\build.bat` → mở app → version hiển thị KHÁC `"dev"` (R-1).
- [ ] **RG-2 Data dir**: ở dev, app đọc/ghi vào `bin/dev/Config` (không phải gốc repo); ở bản build, vào thư mục .exe.
- [ ] **RG-3 Platform count**: số platform đăng ký == baseline Sprint 00 (R-3). Cách đếm xem dev-1/sprint-01.md.
- [ ] **RG-4 go:embed**: app chạy được = frontend/dist + cookie/igcore/iosmess templates đã nhúng OK.
- [ ] **RG-5 Settings persistence**: thay đổi ở Q4 sống sót qua restart (xác nhận os.Chdir chạy đầu — R-7).

## 3. Kiểm cấu trúc & vệ sinh repo
- [ ] Gốc repo chỉ còn: `main.go`, `wails.json`, `go.mod`, `go.sum`, `README.md`, `CLAUDE.md`, `.gitignore`, `.gitattributes` + thư mục chuẩn.
- [ ] Không còn `app*.go` ở gốc; tất cả ở `internal/app/` là `package app`.
- [ ] `git ls-files` không còn `test_accounts*.txt`, `Config/Cookie/cookie_initial.txt`.
- [ ] `internal/cookie/embedded/cookie_initial.txt` **vẫn còn** (build input).
- [ ] `golang.org/x/image` còn trong go.mod (icongen ở tools/).
- [ ] `*_test.go` white-box vẫn cạnh code (không bị đẩy vào tests/).

## 4. Kết luận nghiệm thu
- PASS khi: toàn bộ Q1–Q12 không hồi quy + RG-1..5 đạt + mục 3 đạt.
- Bất kỳ FAIL → ghi vào `current-state.md` (Blocker) + tạo task sửa, KHÔNG đánh dự án "hoàn thành".

> Ghi kết quả chạy QA (ngày, ai chạy, PASS/FAIL từng mục) vào `completed-log.md` khi làm Sprint 04.
