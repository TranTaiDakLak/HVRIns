# Integration Plan — chiến lược branch & thứ tự merge

> Mục tiêu: 2 dev làm song song nhưng gộp lại an toàn, không vỡ build. Mọi verify trên **Windows**;
> cổng = `wails build`.

## 1. Chọn mô hình làm việc

### Phương án A — Tuần tự trên 1 branch (đơn giản, khuyến nghị cho newbie)
- Tạo branch tái cấu trúc: `git switch -c refactor/structure`
- Chạy theo thứ tự sync (xem conflict-matrix SYNC-1→3). Mỗi task 1 commit nhỏ.
- Ưu: không cần merge phức tạp. Nhược: ít song song hơn.

### Phương án B — Mỗi dev một worktree/branch (song song thật)
```powershell
git switch -c refactor/structure          # branch tích hợp
git worktree add ../HVRIns-dev1 -b dev1/go-restructure refactor/structure
git worktree add ../HVRIns-dev2 -b dev2/cleanup-fe     refactor/structure
```
- Dev 1 làm ở `../HVRIns-dev1`, Dev 2 ở `../HVRIns-dev2`.
- Ưu: chạy đồng thời, không đụng working-tree. Nhược: phải merge theo thứ tự dưới.

> Nếu chọn B: vì app Windows-only và FE cần regenerate bindings, **merge theo đúng thứ tự mục 3**.

## 2. Quy ước commit
- 1 task = 1 commit (trừ S02 cú chuyển = 1 commit nguyên tử gồm nhiều file).
- Mẫu message: `refactor(app): move App into internal/app [S02-D1-T001..T004]`,
  `chore(cleanup): remove cmd scratch, icongen→tools [S01-D2-T004]`,
  `security: untrack leaked creds [S00-D2-T002]`.
- **KHÔNG push** tự động; chờ chủ dự án (push là outward-facing).

## 3. Thứ tự MERGE (nếu dùng Phương án B)

```
1. dev2/cleanup-fe  (Sprint 00–01: secrets, dọn rác, config/sample, go mod tidy)
       └─ merge vào refactor/structure  → verify: wails build xanh
2. dev1/go-restructure  (Sprint 01–02: tách app.go, cú chuyển internal/app, bindings)
       └─ merge vào refactor/structure  → verify: wails build + wails dev, version, platform count
3. dev2/cleanup-fe  (Sprint 02–03: README/CLAUDE/tests, RỒI reorg FE sau khi #2 đã có go/app)
       └─ merge  → verify: npm build, wails dev, FE gọi services OK
4. dev1/go-restructure (Sprint 03: Go tests) + dev2 (Sprint 03 còn lại)
5. Cả 2 Sprint 04 finalize
```

**Lý do thứ tự này:**
- Bước cleanup (Dev 2 S00–01) không đụng Go logic → merge trước, tạo nền sạch.
- Cú chuyển Go (Dev 1 S02) đổi package → sinh `wailsjs/go/app` + sửa import `bridge/wails`. Phải vào
  TRƯỚC khi Dev 2 đổi tên `bridge→services` (S03), nếu không 2 thay đổi chồng lên cùng file.
- FE reorg (Dev 2 S03) vào sau cùng vì đụng nhiều file frontend.

## 4. Điểm dễ conflict & cách xử lý
| File | Vì sao | Cách tránh |
|------|--------|-----------|
| `frontend/src/.../wails/*.ts` | D1 sửa import (S02) rồi D2 đổi tên thư mục (S03) | Merge D1 trước; D2 rename sau, giữ độ sâu |
| `go.mod`/`go.sum` | D1 (dọn dirty) + D2 (tidy) | D1 commit S00 trước; D2 tidy sau |
| `.gitignore` | D2 sửa nhiều lần (S00, S01) | Chỉ D2 sửa; gộp các thay đổi |
| `pm/task-board.md`, `completed-log.md` | cả 2 ghi | append đúng dòng của mình; conflict .md merge tay |

## 5. Checklist trước khi gộp về `main`
- [ ] `refactor/structure` qua hết review-checklist Sprint 04
- [ ] `wails build` + `wails dev` xanh trên Windows
- [ ] `GetAppVersion()` đúng version; số platform == baseline
- [ ] Không còn secret bị track; `internal/cookie/embedded/cookie_initial.txt` còn nguyên
- [ ] Squash/clean history nếu muốn; **chủ dự án** quyết định push & PR
