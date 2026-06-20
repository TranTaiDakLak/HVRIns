# Review Checklist (PM dùng để nghiệm thu)

> Tick khi nghiệm thu từng pha. Chỉ coi pass khi chạy lệnh thật, không claim suông.

## Cổng kiểm tra dùng lại (Windows)
- [ ] `go vet ./...`
- [ ] `npm --prefix frontend run build` rồi `go build .`
- [ ] `go test ./internal/...`
- [ ] `wails build` → có `build/bin/HVRIns.exe`  ← cổng thật
- [ ] `wails dev` → cửa sổ mở, FE gọi được backend

## Nghiệm thu Sprint 00
- [ ] Baseline xanh, số platform đã ghi vào current-state.md
- [ ] go.mod/go.sum sạch (đã commit/revert)
- [x] 4 file secret không còn track; `.gitignore` đã chặn; `internal/cookie/embedded/...` còn nguyên
- [ ] Đã rotate credential (ghi rõ TODO + kế hoạch rewrite trong risks.md — chủ dự án cần làm)

## Nghiệm thu Sprint 01
- [ ] `app.go` đã tách thành các file con; `go build .` vẫn xanh (cùng package, không đổi import)
- [x] Root sạch python lạc + __pycache__; docs đã gom; build.bat ở scripts/
- [x] cmd/ dọn xong (icongen ở tools/, 17 scratch đã xoá); `golang.org/x/image` còn trong go.mod ✅
- [x] config/sample có template .example; launch.json đã sửa HVRINS_DATA_DIR

## Nghiệm thu Sprint 02 (quan trọng nhất)
- [ ] 12+ file ở `internal/app` là `package app`; main.go là bootstrap mỏng ở gốc
- [ ] go:embed vẫn ở main.go; `os.Chdir(...)` là hành động đầu tiên
- [ ] `wails generate module` chạy; import bridge/wails đã đổi `go/main`→`go/app`
- [ ] `wails dev` mở app, FE gọi backend OK
- [ ] `GetAppVersion()` trả version thật (KHÔNG "dev")
- [ ] Số platform đăng ký == baseline
- [ ] `app_test.go` pass ở vị trí mới (white-box)
- [x] README.md gốc đã viết; CLAUDE.md đã viết lại; tests/ scaffold tồn tại ✅ (D2 done)

## Nghiệm thu Sprint 03
- [x] Alias `@/` bật; 195 import đã convert; npm build xanh ✅ (ba1e177)
- [x] Stub src/main.ts + App.vue đã xoá; entry src/app/main.ts chạy đúng ✅ (c314943)
- [x] bridge→services (độ sâu giữ nguyên); modules→features; pages gom đúng feature ✅ (681770e, a3d8210)
- [x] script `test` chạy `vitest run`; passWithNoTests=true; npm test exit 0 ✅ (a3d8210)
- [ ] Go: unit test thay cmd scratch pass; rà import cycle xong (Dev 1 scope)

## Nghiệm thu Sprint 04
- [ ] Verify end-to-end xanh; platform count + version OK (Dev 1 scope)
- [ ] Gốc repo gọn: icongen.exe + empty cmd/ + empty Config/ đã xoá ✅ D2; root Go files còn chờ Dev 1 S02
- [ ] Quyết định Pha 7 đã ghi decision-log (Dev 1 scope)
- [x] Tất cả file pm/ D2 đã cập nhật: task-board, completed-log, progress, risks, review-checklist ✅

## Tiêu chí "hoàn thành toàn bộ"
- [ ] Tất cả task bắt buộc = DONE (task tuỳ chọn có thể defer, ghi rõ)
- [ ] Không còn secret bị track
- [ ] `wails build` + `wails dev` xanh trên Windows
- [ ] Cấu trúc khớp `docs/rebuild/02-cau-truc-dich.md` (kể cả các deviation đã ghi)
