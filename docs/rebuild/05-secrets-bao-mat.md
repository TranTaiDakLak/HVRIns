# 05 — ⚠️ Xử lý secret bị lộ (KHẨN CẤP)

> Việc này **độc lập** với tái cấu trúc và nên làm **sớm nhất có thể**. Đã xác nhận bằng
> `git check-ignore`: các file dưới đây **đang được track** (không bị ignore) và **chứa
> credential THẬT**.

## 1. Các file đang lộ credential

| File | Đang track? | Nội dung nhạy cảm |
|------|-------------|-------------------|
| `Config/Cookie/cookie_initial.txt` | ✅ track | 48KB / 1854 dòng token `datr` phiên Facebook thật |
| `test_accounts_eaag.txt` | ✅ track | ~7 tài khoản FB: `c_user`, `xs`, `datr`, `fr` + token EAAG |
| `test_accounts_eaag_new.txt` | ✅ track | Tương tự |
| `test_accounts_fresh.txt` | ✅ track | Tương tự |
| `cmd/emailtest/main.go` | ✅ track | 10 tài khoản Hotmail + mật khẩu + OAuth refresh token (hardcode trong source) |

## 2. ⚠️ File KHÔNG được đụng vào

| File | Vì sao giữ |
|------|------------|
| `internal/cookie/embedded/cookie_initial.txt` | Đây là **input bắt buộc của build** — `internal/cookie/store.go` dùng `//go:embed` để nhúng nó vào `.exe`. **Xoá file này = hỏng `wails build`.** Nó cùng tên với file lộ ở `Config/Cookie/` nhưng là file khác, đường dẫn khác. |

> 🧠 Hãy đọc đúng đường dẫn trước khi xoá. File cần gỡ là **`Config/Cookie/cookie_initial.txt`**
> (chữ C hoa, thư mục `Config` ở gốc). File cần **giữ** là `internal/cookie/embedded/cookie_initial.txt`.

## 3. Quy trình xử lý (theo thứ tự)

### Bước A — Rotate credential TRƯỚC

Coi như mọi credential trên **đã bị lộ** (chúng nằm trong git history, ai có quyền truy cập repo —
hoặc một bản clone bị rò — đều đọc được). Vì vậy:

1. **Vô hiệu hoá / đổi** các phiên cookie Facebook (`datr`, `xs`, `c_user`, `fr`) và token EAAG.
2. **Đổi mật khẩu + thu hồi OAuth token** của 10 tài khoản Hotmail trong `cmd/emailtest`.

> Đây là việc quan trọng nhất. Gỡ file khỏi git **không** làm credential an toàn trở lại — chỉ
> rotate mới thật sự khắc phục.

### Bước B — Gỡ track (giữ file trên đĩa)

```powershell
git rm --cached Config/Cookie/cookie_initial.txt
git rm --cached test_accounts_eaag.txt test_accounts_eaag_new.txt test_accounts_fresh.txt
# cmd/emailtest sẽ bị xoá hẳn trong Pha 2 (cả thư mục), nên không cần --cached riêng
```

`git rm --cached` **gỡ khỏi tracking nhưng giữ file trên ổ đĩa** — bạn vẫn dùng được tại máy,
chỉ là git không theo dõi nữa.

### Bước C — Thêm vào `.gitignore`

```gitignore
# ── Secrets / runtime data (KHÔNG commit) ──
Config/Cookie/
test_accounts*.txt

# ── Python build cache ──
__pycache__/
*.pyc
```

> Cân nhắc đảo chính sách nguy hiểm ở `.gitignore` dòng 6 (`!build/bin/Config/`) — dòng này đang
> **ép-track** dữ liệu runtime (cookie/proxy/mail) dưới `build/bin/Config`, và comment cũng tự thừa
> nhận "repo phải PRIVATE". Dữ liệu runtime chỉ nên ở `appDataDir()`, **không** nên commit.

### Bước D — Xử lý git history (bước riêng, có phối hợp)

`git rm --cached` **không** xoá file khỏi các commit cũ. Để gột sạch khỏi lịch sử:

```powershell
# Cần cài git-filter-repo (khuyến nghị) hoặc BFG Repo-Cleaner
git filter-repo --invert-paths `
  --path Config/Cookie/cookie_initial.txt `
  --path test_accounts_eaag.txt `
  --path test_accounts_eaag_new.txt `
  --path test_accounts_fresh.txt
```

⚠️ **Cảnh báo:** rewrite history **viết lại toàn bộ commit hash** → ảnh hưởng mọi người đang
clone repo. Đây là thao tác phối hợp (thông báo team, mọi người re-clone). Nếu repo chỉ có một
mình bạn và để private, có thể làm; nếu nhiều người, lên lịch cẩn thận.

> Dù có rewrite history hay không, **việc rotate credential ở Bước A vẫn là bắt buộc** vì có thể
> đã có bản sao bị rò.

### Bước E — Kiểm tra

```powershell
git status                      # không còn track các file secret
git check-ignore Config/Cookie/cookie_initial.txt   # phải in ra đường dẫn (= đã ignore)
wails build                     # vẫn xanh (vì internal/cookie/embedded/* không bị đụng)
```

## 4. Mẫu thay thế (tuỳ chọn)

Nếu cần file mẫu cho người mới biết định dạng, tạo bản **đã che** (không có cookie/token thật):

```
# config/sample/accounts.example.txt
# Định dạng: UID|Password|Cookies|EAAGToken|DateTime|Country|Status
100000000000000|REDACTED|datr=REDACTED; c_user=REDACTED; xs=REDACTED|EAAG-REDACTED|2026-01-01 00:00|VN|live
```

## 5. Tóm tắt

| Việc | Lệnh / hành động | Bắt buộc? |
|------|------------------|-----------|
| Rotate credential | (thủ công trên FB/Hotmail) | ✅ Bắt buộc |
| Gỡ track | `git rm --cached ...` | ✅ Bắt buộc |
| Thêm `.gitignore` | sửa file | ✅ Bắt buộc |
| Rewrite history | `git filter-repo` / BFG | ⬜ Khuyến nghị (phối hợp) |
| **KHÔNG** xoá `internal/cookie/embedded/cookie_initial.txt` | — | ✅ Bắt buộc nhớ |

→ Quay lại: [04-ke-hoach-thuc-thi.md](04-ke-hoach-thuc-thi.md) (Pha 1)
