# Project Scan — kết quả phân tích repo

> Snapshot hiện trạng tại 2026-06-20 (từ đợt phân tích sâu 8-agent). Chi tiết: `docs/rebuild/01-hien-trang.md`.

## Số liệu
- Tổng file track: **~3327**
- `internal/` 3103 (instagram 2960: register 2274, verify 624) · `frontend/` 118 · `docs/` 27 ·
  `cmd/` 20 · `build/` 9 · `Config/` 9 · `scripts/` 8

## File Go ở gốc (13, đều `package main`)
| File | KB | Vai trò |
|------|----|---------|
| main.go | 3.9 | Entry: go:embed, AppVersion, wails.Run, Bind |
| app.go | 317 | struct App + NewApp + 121 method + ~27 type |
| app_register.go | 219 | RunRegister + helper |
| app_reg_sxxx.go | 64 | helper register sxxx (1 blank-import) |
| app_verify.go | 64 | RunVerify |
| app_banclone.go | 17 | BancloneLogin/GetBancloneProducts |
| app_tempmail_reg.go | 11 | temp-mail |
| debug.go | 6.5 | pprof/memory (3 method) |
| app_getdatr.go | 5.7 | datr refresh |
| cpu_windows.go | 4.6 | CPU (Windows-only) |
| app_test.go | 3.0 | test white-box (hàm private) |
| datadir.go | 1.0 | appDataDir() — đọc AppVersion |
| portrange_windows.go | 0.8 | expandEphemeralPortRange (Windows-only) |

## Điểm kỹ thuật then chốt
- **207 blank-import** kích hoạt đăng ký platform (206 trong app.go, 1 trong app_reg_sxxx.go).
- **5 go:embed** (xác nhận 2026-06-21): main.go(frontend/dist) · internal/cookie/store.go(embedded/ — 2 directives) · internal/igcore/template.go(templates/) · internal/instagram/register/ios/iosmess/embed.go(templates/).
- **Windows-only**: cpu/portrange không có bản `_other.go`.
- **go:embed cấm `../`** → main.go phải ở gốc.
- `os.Chdir(appDataDir())` ở main() là hành động đầu tiên → mọi path tương đối tính từ đó.

## 🔴 Secrets bị track (xác nhận bằng git check-ignore)
- `Config/Cookie/cookie_initial.txt` (48KB token datr thật)
- `test_accounts_eaag.txt`, `test_accounts_eaag_new.txt`, `test_accounts_fresh.txt` (cookie+token FB)
- `cmd/emailtest/main.go` (10 Hotmail + OAuth token)
- ⚠ KHÔNG nhầm với `internal/cookie/embedded/cookie_initial.txt` (cần cho build).

## Rác / cần dọn
- Root python lạc: `_patch_datr_diag.py`, `decode_request.py` (hardcode E:/WEMAKE/...)
- `scripts/__pycache__/*.pyc`
- `cmd/`: 17 chương trình scratch + 2 stub chết; 1 tool thật `icongen` (giữ → tools/).
- docs/old-docs (18 file cũ); markdown rải ở gốc (NVRINS_BUILD_GUIDE.md, README_TEST_EAAG.md).
- `CLAUDE.md` nội dung mâu thuẫn thực tế; `README.md` rỗng.

## Frontend (đã khá chuẩn)
- bridge/ (= services theo chuẩn) · modules/ (= features) · pages/ · components/{ui,grid,form,feedback,shell} · composables · stores · 
- 2 stub chết: src/main.ts, src/App.vue (entry thật: src/app/main.ts).
- **192 import tương đối, 0 alias @/** → phải bật alias trước khi reorg.
- vitest đã cài nhưng 0 test + thiếu script `test`.
