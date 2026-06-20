# Dev 2 — Sprint 00: 🔴 Secrets & Setup

> Việc này **độc lập** với baseline của Dev 1 — làm sớm nhất có thể. Chi tiết: `docs/rebuild/05-secrets-bao-mat.md`.

---

## S00-D2-T001 — Đọc plan & môi trường
**Việc:** đọc `docs/rebuild/05-secrets-bao-mat.md`, `01-hien-trang.md`, `03-anh-xa-chi-tiet.md`, `02-cau-truc-dich.md`.
Kiểm: `git --version`, `node -v`, `npm -v`. (Có `git-filter-repo`/BFG không? — cần cho S04-D2-T002.)
**Test:** — **DONE khi:** nắm rõ danh sách file lộ + file CẤM xoá (`internal/cookie/embedded/cookie_initial.txt`).

---

## S00-D2-T002 — Gỡ secrets khỏi tracking
**⚠ ĐỌC KỸ:** chỉ gỡ file ở `Config/Cookie/`, **KHÔNG** đụng `internal/cookie/embedded/cookie_initial.txt` (R-14).
**Việc:**
```powershell
git rm --cached Config/Cookie/cookie_initial.txt
git rm --cached test_accounts_eaag.txt test_accounts_eaag_new.txt test_accounts_fresh.txt
```
Thêm vào `.gitignore` (cuối file):
```gitignore
# ── Secrets / runtime data ──
Config/Cookie/
test_accounts*.txt
# ── Python cache ──
__pycache__/
*.pyc
```
**Test:**
```powershell
git check-ignore Config/Cookie/cookie_initial.txt   # phải in ra path = đã ignore
git status                                            # không còn track 4 file đó
git ls-files internal/cookie/embedded/               # cookie_initial.txt VẪN còn track
```
**DONE khi:** 4 file lộ hết track; file embedded còn nguyên.

> Commit: `security: gỡ credential khỏi tracking + chặn .gitignore`.

---

## S00-D2-T003 — Checklist rotate + xác nhận build
**Việc:**
- Điền/đánh dấu checklist rotate trong `workspaces/pm/risks.md` mục "Checklist rotate credentials".
  (Việc rotate thật trên FB/Hotmail là của chủ dự án — nếu chưa làm được, ghi rõ TODO + cảnh báo.)
- Xác nhận build không bị ảnh hưởng (vì file embedded của build không bị đụng):
```powershell
npm --prefix frontend run build ; wails build
```
**Test:** wails build PASS.
**DONE khi:** checklist rotate đã ghi trạng thái; wails build xanh.

---

### Sau Sprint 00
Cập nhật progress + task-board + completed-log. Sang Sprint 01 (song song Dev 1).
