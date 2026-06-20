# Risks

> Rủi ro chính + cách giảm thiểu. Bản đầy đủ 16 rủi ro: `docs/rebuild/07-checklist-rui-ro.md`.
> Cột Owner = dev cần canh chừng.

| # | Mức | Rủi ro | Giảm thiểu | Owner |
|---|-----|--------|------------|-------|
| R-1 | 🔴 | `AppVersion` cross-package: datadir.go vào internal/app không thấy `main.AppVersion` → nếu khai biến mới = "dev" → prod ghi data vào bin/dev (âm thầm) | Thread version qua `SetVersion()`; verify `GetAppVersion()` != "dev" ở bản build | D1 |
| R-2 | 🔴 | Quên `wails generate` / quên sửa import binding → FE gọi symbol không tồn tại | S02-D1-T004 làm liền sau move; smoke test wails dev | D1 |
| R-3 | 🔴 | Mất 1 blank-import → platform không đăng ký (vẫn compile) | Đếm số platform trước (S00) và sau (S02/S04) | D1 |
| R-4 | 🔴 | Secrets còn trong git history dù đã rm --cached | **Rotate creds** (bắt buộc) + cân nhắc filter-repo/BFG (S04-D2-T002) | D2 |
| R-5 | 🟠 | Cú chuyển không nguyên tử → trạng thái nửa vời không compile | Làm trọn trong 1 commit (S02-D1) | D1 |
| R-6 | 🟠 | main.go dùng `app.ctx`/`app.startup` (private) sau khi tách | Export + bọc ctx vào method `OnSecondInstance` | D1 |
| R-7 | 🟠 | `os.Chdir(appDataDir())` không còn là hành động đầu tiên → đọc/ghi sai thư mục | Giữ nguyên vị trí đầu `main()` | D1 |
| R-8 | 🟠 | App Windows-only → verify trên Linux/CI sẽ lỗi | Mọi verify chạy trên Windows; cổng = `wails build` | Cả 2 |
| R-9 | 🟡 | `go build` cây sạch lỗi go:embed (thiếu frontend/dist) | npm run build trước, hoặc dùng wails build | Cả 2 |
| R-10 | 🟡 | `go mod tidy` gỡ nhầm `golang.org/x/image` | Chạy tidy SAU khi giữ icongen; `go build ./tools/...` | D2 |
| R-11 | 🟡 | Đổi tên thư mục FE vỡ 192 import tương đối | Bật alias `@/` TRƯỚC; làm từng feature | D2 |
| R-12 | 🟡 | bridge/wails import resolve sai khi đổi tên (độ sâu) | Giữ NGUYÊN độ sâu khi rename bridge→services | D2 |
| R-13 | 🟡 | Import cycle khi App ↔ settings/adapter | Rà legacy.go sau move (S03-D1-T003) | D1 |
| R-14 | 🟢 | Nhầm 2 file cookie_initial.txt | Chỉ gỡ `Config/Cookie/...`, GIỮ `internal/cookie/embedded/...` | D2 |
| R-15 | 🟠 | Conflict 2 dev sửa trùng file | Tuân thủ bảng "Ranh giới file" trong sprint-plan; D2-S03 đợi D1-S02 | PM |

## Checklist rotate credentials (S00-D2-T003)
> Cập nhật 2026-06-20 — Dev 2. File đã gỡ khỏi tracking (git rm --cached). Rotate thật là việc THỦ CÔNG của chủ dự án.
- [ ] **TODO (chủ dự án)** Vô hiệu hoá/đổi phiên cookie FB (datr/xs/c_user/fr) trong các file lộ
- [ ] **TODO (chủ dự án)** Thu hồi token EAAG trong test_accounts_eaag*.txt
- [ ] **TODO (chủ dự án)** Đổi mật khẩu + thu hồi OAuth refresh token 10 tài khoản Hotmail (cmd/emailtest/main.go)
- [ ] **TODO (chủ dự án)** Lên lịch rewrite history (`git filter-repo`) — cần phối hợp toàn team

> ⚠️ Cảnh báo: 4 file lộ ĐÃ bị gỡ khỏi HEAD nhưng vẫn còn trong git history. Mọi credential trên phải coi là đã lộ và phải rotate. Rewrite history nếu repo không phải private-only.

## Kế hoạch rewrite git history (S04-D2-T002)

> ⚠️ KHÔNG tự chạy nếu repo có nhiều người clone. Cần thông báo team + mọi người re-clone sau.

### Công cụ cần cài
```powershell
pip install git-filter-repo    # khuyến nghị (hoặc tải BFG từ https://rtyley.github.io/bfg-repo-cleaner/)
```

### Lệnh xoá 5 file khỏi toàn bộ history
```powershell
git filter-repo --invert-paths `
  --path Config/Cookie/cookie_initial.txt `
  --path test_accounts_eaag.txt `
  --path test_accounts_eaag_new.txt `
  --path test_accounts_fresh.txt `
  --path cmd/emailtest/main.go
```

### Quy trình (đầy đủ)
1. **Backup repo**: `git clone --mirror . ../HVRIns-backup`
2. **Rotate credential TRƯỚC** (mục checklist trên — bắt buộc, dù có rewrite hay không)
3. Chạy `git filter-repo` (lệnh trên)
4. Force push tất cả branches: `git push --force --all`
5. Thông báo mọi người re-clone: `git clone <url>`
6. Xoá backup sau khi verify

### Trạng thái hiện tại
- [x] `git rm --cached` 4 file lộ khỏi HEAD (S00-D2-T002, commit 9bfe34a) ✅
- [ ] **TODO** Rotate credential (chủ dự án, xem checklist trên)
- [ ] **TODO** Cài git-filter-repo / BFG (chủ dự án xác nhận tool có trên máy)
- [ ] **TODO** Chạy filter-repo + force push (phối hợp team, lên lịch cẩn thận)

### Lưu ý thêm
- Sau rewrite, `git log` sẽ có hash mới hoàn toàn — mọi bản clone cũ đều outdated.
- Nếu repo đã được fork hoặc mirror ở đâu đó, những bản đó vẫn giữ credential cũ.
- Rewrite không có nghĩa credential đã "an toàn" — rotate là bắt buộc bất kể có rewrite hay không.
