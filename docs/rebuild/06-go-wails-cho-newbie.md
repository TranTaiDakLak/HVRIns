# 06 — Khái niệm Go & Wails cho người mới

> Phần này giải thích các khái niệm Go/Wails **liên quan trực tiếp** tới việc tái cấu trúc, để bạn
> hiểu **vì sao** từng bước lại an toàn hay rủi ro. Đọc trước khi làm Pha 3.

## 1. Package — đơn vị biên dịch của Go

- Một **package** = một **thư mục** chứa các file `.go` cùng khai báo `package X`.
- Go **biên dịch theo package**, không theo từng file.
- 13 file `.go` ở gốc đều khai báo `package main` → Go coi chúng là **một chương trình duy nhất**.
  Vì vậy method trong `app_register.go` gọi được hàm/type trong `app.go` mà **không cần import,
  không cần tiền tố** — chúng cùng một package.

👉 **Hệ quả cho migration:** khi chuyển vào `internal/app`, **tất cả** file phải đổi sang
`package app` và di chuyển **cùng lúc**. Trạng thái nửa-`main`-nửa-`app` sẽ không compile.
Nhưng vì vẫn là **một** package mới, các lời gọi nội bộ giữa chúng **không cần sửa**.

## 2. `package main` đặc biệt

- Chỉ `package main` (có `func main()`) mới biên dịch ra **file chạy được** (`.exe`).
- Đó là lý do `main.go` mang `package main`. Khi các file khác thành `package app`, chúng trở thành
  **thư viện** — không tự chạy được, nhưng được import.
- **Không thể import một `package main`** từ package khác. Đó là vì sao mỗi `cmd/<x>/` là một
  chương trình độc lập, và **xoá chúng không làm hỏng phần library**.

## 3. Import path = module + đường dẫn thư mục

- `go.mod` khai báo `module HVRIns`. Nên thư mục `internal/cookie` được import là
  `HVRIns/internal/cookie`.
- **Đổi tên/di chuyển thư mục = đổi import path** ở **mọi nơi** import nó. Go **không** tự sửa —
  phải dùng IDE/`gopls`/`goimports`.

👉 **Hệ quả:** đây là lý do **không** di chuyển `internal/instagram` (2960 file, ~2900 import) ở
pass 1. Còn chuyển root vào `internal/app` thì an toàn hơn vì root là `package main` — **không ai
import nó**, nên không có import path nào cần sửa từ phía ngoài (chỉ `main.go` mới thêm import mới).

## 4. `internal/` — ranh giới private do Go cưỡng chế

- Package nằm dưới `internal/` **chỉ** được import bởi code trong cùng module.
- Đây chính là "private app logic" mà khung chuẩn muốn → đặt logic trong `internal/app` là đúng bài.

## 5. Exported vs unexported — quyết định bởi chữ cái đầu

- **Hoa** = exported (package khác thấy được; **Wails bind các method Hoa** cho frontend).
- **thường** = private (chỉ trong package).
- Ví dụ: `ListAccounts` (Hoa) → Vue gọi được; `ctx` (thường) → package khác không đụng tới.

👉 **Hệ quả:** sau khi tách App vào `internal/app`, mọi thứ `main.go` còn gọi xuyên package
**phải là Hoa**. Đó là lý do phải export `Startup`, `AppDataDir`, `ExpandEphemeralPortRange`...
Và `app.ctx` (thường) thì **không** truy cập trực tiếp được từ `main.go` → phải gói logic dùng
`ctx` vào một method Hoa.

## 6. `//go:embed` — nhúng file vào binary lúc biên dịch

- `//go:embed all:frontend/dist` "nướng" toàn bộ web đã build vào `.exe`.
- **Quy tắc chí mạng:** đường dẫn tính **tương đối so với thư mục chứa file `.go`**, và **không
  được dùng `../`** (không trỏ ra thư mục cha).

👉 **Đây là lý do #1** khiến `main.go` **phải ở gốc** (cạnh `frontend/`). Nếu đưa vào `cmd/app/`,
nó không thể `//go:embed ../../frontend/dist`.

- Repo có **5** chỗ dùng `go:embed` (đã xác nhận lại 2026-06-21):
  1. `main.go` → `frontend/dist`
  2. `internal/cookie/store.go` → `embedded/` (2 directives)
  3. `internal/igcore/template.go` → `templates/`
  4. `internal/instagram/register/ios/iosmess/embed.go` → `templates/`
- Nếu di chuyển 3 package internal đó, **phải mang theo thư mục asset** cạnh file `.go`, nếu không
  build lỗi.

> ⚠️ Trên bản clone sạch, `go build` package gốc **lỗi** vì `frontend/dist` (gitignored) chưa tồn
> tại → `go:embed` "no matching files found". Luôn build frontend trước, hoặc dùng `wails build`.

## 7. `-ldflags "-X path.Var=value"` — gán biến lúc link

- `wails build -ldflags "-X main.AppVersion=v26.06.20.30"` ghi giá trị vào biến `AppVersion` của
  **package main** lúc link.
- `path` = **import path đầy đủ của package**. `-X main.AppVersion` chỉ đúng khi `AppVersion` còn ở
  `package main`. Đưa nó sang package khác thì phải đổi chuỗi này (vd `-X HVRIns/internal/app.AppVersion`).

👉 **Hệ quả + bẫy:** ta **giữ `AppVersion` ở `package main`** (gốc) để `build.bat` không phải đổi.
NHƯNG `datadir.go` (chuyển vào `internal/app`) đọc `AppVersion` để phân biệt dev/prod — nó **không
còn thấy** `main.AppVersion`. Phải **truyền** version vào `internal/app` (qua `SetVersion`/constructor).
Nếu quên, `AppVersion` trong package app mặc định `"dev"` → app **prod** sẽ ghi data vào `bin/dev`
(lỗi âm thầm, khó phát hiện).

## 8. Build constraint theo tên file (`_windows.go`)

- File kết thúc `_windows.go` **chỉ biên dịch trên Windows** — không cần comment `//go:build`.
- `cpu_windows.go`, `portrange_windows.go` dựa hoàn toàn vào hậu tố này.

👉 **Hệ quả:** khi di chuyển, **giữ nguyên hậu tố `_windows.go`**. Và vì không có bản `_linux.go`/
`_darwin.go` tương ứng, **app chỉ build được trên Windows** → mọi kiểm tra phải trên Windows;
`go build ./...` trên Linux/macOS sẽ lỗi (thiếu `getProcessCPUTime`, `expandEphemeralPortRange`).

## 9. Blank import & `init()` — "plugin registry"

- `import _ "some/pkg"` = import **chỉ để chạy `init()`** của package đó, không dùng tên nào.
- `init()` là hàm Go **tự chạy** khi package được nạp (trước `main`).
- Trong dự án này, mỗi package phiên bản (`s557`, `s23`...) gọi `RegisterPlatformRegisterer(...)`
  trong `init()`. App **blank-import** từng package để kích hoạt đăng ký. Có **207 dòng blank-import**.

👉 **Bẫy âm thầm:** nếu một blank-import bị mất/sai khi di chuyển, code **vẫn compile** nhưng
platform đó **không đăng ký** lúc chạy → một loại lỗi rất khó truy. Vì vậy ở Pha 0 ta **đếm số
platform đăng ký**, rồi so lại sau Pha 3/4. (Tin tốt: 206 blank-import nằm trong `app.go` + 1 trong
`app_reg_sxxx.go`, nên chúng **tự đi theo** khi chuyển 2 file này — không cần thao tác thêm.)

## 10. Import cycle — Go cấm vòng lặp phụ thuộc

- Go **không cho** package A import B trong khi B import A (trực tiếp hay gián tiếp).
- Codebase này cố tình tránh cycle: package `instagram` lõi **không** import các package phiên bản;
  ngược lại, các phiên bản import lõi, còn app blank-import các phiên bản.

👉 **Hệ quả:** khi (nếu) tách App thành `usecase/adapter` ở Pha 7, dễ vô tình tạo cycle. Luôn
`go build` sau mỗi thay đổi để bắt sớm. Đặc biệt: `internal/settings/adapter/legacy.go` hiện
"nhân bản" struct từ `package main` để **tránh** import main; khi App vào `internal/app`, cần rà
lại ranh giới này để không tạo cycle `internal/app ↔ internal/settings`.

## 11. Test: white-box vs black-box

- `*_test.go` với **cùng** tên package (`package app`) → thấy hàm **private** (white-box). **Phải
  ở cùng thư mục** với code.
- `*_test.go` với `package app_test` → chỉ thấy hàm **public** (black-box). Loại này mới vào `tests/go/`.

👉 **Hệ quả:** `app_test.go` test `isVerifiableAccountFile` (private) → **phải** ở `internal/app/`,
**không** chuyển vào `tests/go/`. 31 file `*_test.go` trong `internal/` cũng giữ nguyên tại chỗ.

## 12. Wails bindings — cầu nối Go ↔ JS

- `wails generate module` đọc struct `App` (cái được `Bind`) và sinh ra wrapper TypeScript ở
  `frontend/wailsjs/go/<tên-package>/`. **Thư mục đặt theo tên package Go.**
- Đó là vì sao đổi `package main` → `app` làm binding chuyển từ `wailsjs/go/main/` sang
  `wailsjs/go/app/` → 10 file `services/wails/*.ts` (hiện `bridge/wails/`) phải đổi import. Đây là
  **thay đổi FE duy nhất bắt buộc** do việc đổi package.

## 13. `wails build` vs `go build`

- `wails build` = **build frontend trước** (`npm run build` → `frontend/dist`) **rồi** mới compile Go.
  Nên `go:embed` luôn có file. → **Đây là cổng kiểm tra thật sự.**
- `go build ./...` **không** build frontend → có thể lỗi `go:embed` trên cây sạch, và lỗi trên
  non-Windows. Dùng nó để check compile nhanh nhưng đừng coi là cổng cuối.

## 14. Bộ ba kiểm tra cho mọi thay đổi Go

| Lệnh | Kiểm gì |
|------|---------|
| `go vet ./...` | Lỗi tĩnh, nghi vấn |
| `go build .` (sau khi `npm run build`) | Có compile không (nhớ bẫy go:embed) |
| `go test ./internal/...` | Test có pass không |
| `wails build` + `wails dev` | Cổng end-to-end (bắt buộc trên Windows) |

→ Quay lại: [04-ke-hoach-thuc-thi.md](04-ke-hoach-thuc-thi.md) · [07-checklist-rui-ro.md](07-checklist-rui-ro.md)
