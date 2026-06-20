# Dev 2 — Sprint 01: Dọn rác non-Go + config/sample

Chạy **song song** Dev 1 (Dev 1 đụng `app.go`/`internal/app` — bạn KHÔNG đụng). Mọi việc ở đây
**không sửa file `.go` của app** nên không thể làm hỏng compile của app.

> Tham chiếu: `docs/rebuild/04` Pha 2 + Pha 5; bảng ánh xạ `docs/rebuild/03` mục B, E, F.

---

## S01-D2-T001 — Xoá python lạc + pycache
```powershell
git rm _patch_datr_diag.py decode_request.py
git rm -r --cached scripts/__pycache__
```
(`.gitignore` đã chặn `__pycache__/`,`*.pyc` từ S00.)
**Test:** `git status` sạch các path đó; `wails build` PASS.
**DONE khi:** xoá xong, build xanh.

---

## S01-D2-T002 — Gom docs
```powershell
git mv NVRINS_BUILD_GUIDE.md docs/NVRINS_BUILD_GUIDE.md
New-Item -ItemType Directory -Force docs/testing | Out-Null
git mv README_TEST_EAAG.md docs/testing/eaag-verify-flow.md
New-Item -ItemType Directory -Force docs/archive | Out-Null
git mv docs/old-docs/* docs/archive/ ; Remove-Item docs/old-docs -Recurse -Force -ErrorAction SilentlyContinue
git mv docs/facebook docs/flows
New-Item -ItemType Directory -Force docs/rebuild/specs | Out-Null
Copy-Item .kiro/specs/hvrins-instagram-clone/*.md docs/rebuild/specs/   # copy nếu còn dùng Kiro
```
Sửa link chết trong `docs/testing/eaag-verify-flow.md` (tham chiếu file/đường dẫn E:/WEMAKE không tồn tại) — ghi chú "outdated".
**Test:** `git status` thể hiện rename đúng; mở vài file kiểm link.
**DONE khi:** docs gom xong, không file doc nào còn lạc ở gốc (trừ README.md/CLAUDE.md).

---

## S01-D2-T003 — build.bat → scripts + legacy
```powershell
git mv build.bat scripts/build.bat
New-Item -ItemType Directory -Force scripts/legacy | Out-Null
git mv scripts/migrate.ps1 scripts/legacy/
git mv scripts/rename_identity.ps1 scripts/legacy/
git mv scripts/recolor.py scripts/legacy/
```
⚠ `scripts/build.bat` phải `cd` về gốc trước khi gọi `wails build` (vì wails cần gốc module). Thêm
ở đầu file: `cd /d "%~dp0\.."` (từ scripts/ lùi 1 cấp về gốc).
**Test:** chạy `scripts\build.bat` → ra `build/bin/HVRIns.exe` với version đúng định dạng.
**DONE khi:** build.bat chạy được từ vị trí mới.

---

## S01-D2-T004 — Dọn cmd/ + tools/ + go mod tidy
```powershell
New-Item -ItemType Directory -Force tools | Out-Null
git mv cmd/icongen tools/icongen
git rm -r cmd/test_bloks_login cmd/regtest cmd/_testloginios cmd/emailtest `
          cmd/check_verified_email cmd/proxycheck cmd/proxytest cmd/testbody `
          cmd/test_regex cmd/test_ua cmd/testua cmd/test273 cmd/test_eaag_flow `
          cmd/test_messios cmd/testverios cmd/verifymess cmd/verifytest
go mod tidy
```
**Test (R-10):**
```powershell
Select-String -Path go.mod -Pattern "golang.org/x/image"   # phải CÒN (icongen dùng)
go build ./tools/...                                         # icongen build xanh
wails build                                                  # app vẫn xanh
```
**DONE khi:** cmd/ chỉ còn (trống); icongen ở tools/; x/image còn; build xanh.

> ⚠ `cmd/emailtest` chứa secret → việc xoá nó cũng góp phần dọn secret (history vẫn còn — xem S04-D2-T002).

---

## S01-D2-T005 — config/sample + sửa launch.json
```powershell
New-Item -ItemType Directory -Force config/sample/Permanent, config/sample/Proxy, config/sample/TempMail, config/sample/DeviceInfo | Out-Null
git mv Config/Permanent/mail.txt   config/sample/Permanent/mail.example.txt
git mv Config/Permanent/phone.txt  config/sample/Permanent/phone.example.txt
git mv Config/Proxy/proxy_rentmail.txt config/sample/Proxy/proxy_rentmail.example.txt
git mv Config/Proxy/proxy_tempmail.txt config/sample/Proxy/proxy_tempmail.example.txt
git mv Config/TempMail/domains.txt config/sample/TempMail/domains.example.txt
git mv Config/DeviceInfo/versions_and_builds.txt     config/sample/DeviceInfo/versions_and_builds.example.txt
git mv Config/DeviceInfo/versions_and_builds_reg.txt config/sample/DeviceInfo/versions_and_builds_reg.example.txt
git mv Config/DeviceInfo/versions_and_builds_ver.txt config/sample/DeviceInfo/versions_and_builds_ver.example.txt
```
Sửa `.vscode/launch.json`: `"HVR_DATA_DIR"` → `"HVRINS_DATA_DIR"` (bug có sẵn; code đọc `HVRINS_DATA_DIR`).
**Test:** `wails dev` chạy, app đọc/ghi data đúng `bin/dev/Config` (dev). Không lỗi đọc file thiếu.
(Nhắc: app **không** đọc `Config/` gốc lúc chạy — xem D-002/`docs/rebuild/02` mục 4 — nên việc move template là an toàn.)
**DONE khi:** template ở config/sample; launch.json đúng env; wails dev xanh.

---

### Sau Sprint 01
Cập nhật progress + task-board + completed-log. Commit theo từng task để dễ bisect.
