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
- [ ] 4 file secret không còn track; `.gitignore` đã chặn; `internal/cookie/embedded/...` còn nguyên
- [ ] Đã rotate credential (hoặc ghi rõ kế hoạch trong risks.md)

## Nghiệm thu Sprint 01
- [ ] `app.go` đã tách thành các file con; `go build .` vẫn xanh (cùng package, không đổi import)
- [ ] Root sạch python lạc + __pycache__; docs đã gom; build.bat ở scripts/
- [ ] cmd/ chỉ còn (dọn xong); icongen ở tools/; `golang.org/x/image` còn trong go.mod
- [ ] config/sample có template .example; launch.json đã sửa HVRINS_DATA_DIR

## Nghiệm thu Sprint 02 (quan trọng nhất)
- [ ] 12+ file ở `internal/app` là `package app`; main.go là bootstrap mỏng ở gốc
- [ ] go:embed vẫn ở main.go; `os.Chdir(...)` là hành động đầu tiên
- [ ] `wails generate module` chạy; import bridge/wails đã đổi `go/main`→`go/app`
- [ ] `wails dev` mở app, FE gọi backend OK
- [ ] `GetAppVersion()` trả version thật (KHÔNG "dev")
- [ ] Số platform đăng ký == baseline
- [ ] `app_test.go` pass ở vị trí mới (white-box)
- [ ] README.md gốc đã viết; CLAUDE.md đã viết lại; tests/ scaffold tồn tại

## Nghiệm thu Sprint 03
- [ ] Alias `@/` bật; import đã convert; npm build xanh
- [ ] Stub src/main.ts + App.vue đã xoá; entry chạy đúng
- [ ] bridge→services (độ sâu giữ nguyên); modules→features; pages gom đúng feature
- [ ] script `test` chạy `vitest run`; (có/không) test mẫu
- [ ] Go: unit test thay cmd scratch pass; rà import cycle xong

## Nghiệm thu Sprint 04
- [ ] Verify end-to-end xanh; platform count + version OK
- [ ] Gốc repo chỉ còn: main.go, wails.json, go.mod/sum, README.md, CLAUDE.md, .gitignore/.gitattributes + thư mục chuẩn
- [ ] Quyết định Pha 7 đã ghi decision-log
- [ ] Tất cả file pm/ đã cập nhật (current-state, task-board, completed-log)

## Tiêu chí "hoàn thành toàn bộ"
- [ ] Tất cả task bắt buộc = DONE (task tuỳ chọn có thể defer, ghi rõ)
- [ ] Không còn secret bị track
- [ ] `wails build` + `wails dev` xanh trên Windows
- [ ] Cấu trúc khớp `docs/rebuild/02-cau-truc-dich.md` (kể cả các deviation đã ghi)
