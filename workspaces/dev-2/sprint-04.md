# Dev 2 — Sprint 04: Rà gốc gọn & secrets history

---

## S04-D2-T001 — Rà gốc repo gọn + review-checklist
**Việc:** xác nhận thư mục gốc giờ chỉ còn file/thư mục chuẩn:
```powershell
git ls-files | ForEach-Object { ($_ -split '/')[0] } | Sort-Object -Unique
# Mong đợi ở gốc (không thư mục): main.go, wails.json, go.mod, go.sum, README.md, CLAUDE.md, .gitignore, .gitattributes
# Thư mục: cmd/(trống/dọn), internal/, frontend/, tools/, scripts/, config/, build/, docs/, tests/, infra/(nếu tạo), workspaces/
```
- Không còn: `app*.go` ở gốc, `*.py` lạc, `test_accounts*.txt`, `build.bat` ở gốc, `Config/` rác.
- Tick `workspaces/pm/review-checklist.md` các mục đã đạt; mục nào chưa → ghi vào current-state.
**Test:** lệnh trên cho danh sách gọn đúng mong đợi.
**DONE khi:** gốc gọn + review-checklist cập nhật.

---

## S04-D2-T002 — (Tuỳ chọn) Kế hoạch rewrite git history cho secrets
> Bắt buộc nhớ (R-4/R-15): `git rm --cached` KHÔNG xoá secret khỏi history. Rotate creds là bắt buộc;
> rewrite history là bước phối hợp.
**Việc:** soạn kế hoạch (KHÔNG tự chạy nếu repo có nhiều người — cần đồng thuận) vào `pm/risks.md`:
- Công cụ: `git filter-repo` (khuyến nghị) hoặc BFG.
- Lệnh mẫu:
  ```
  git filter-repo --invert-paths `
    --path Config/Cookie/cookie_initial.txt `
    --path test_accounts_eaag.txt --path test_accounts_eaag_new.txt --path test_accounts_fresh.txt `
    --path cmd/emailtest/main.go
  ```
- Cảnh báo: viết lại toàn bộ commit hash → mọi người phải re-clone.
- Xác nhận lại: đã rotate creds (nếu chưa → đánh dấu TODO đỏ trong risks.md).
**Test:** — (kế hoạch). **DONE khi:** risks.md có quy trình + trạng thái rotate.

---

### Hoàn tất dự án (phối hợp Dev 1)
Khi cả 2 dev xong S04: chốt `pm/review-checklist.md` mục "Tiêu chí hoàn thành toàn bộ" + cập nhật
`current-state.md` sang trạng thái "Tái cấu trúc hoàn tất".
