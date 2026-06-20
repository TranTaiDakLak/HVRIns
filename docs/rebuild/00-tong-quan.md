# 00 — Tổng quan & quyết định kiến trúc

## 1. Dự án này thực sự là gì?

**HVRIns** (tên hiển thị: **"Hạ Vũ"**) là một **desktop app đã hoàn chỉnh**, viết bằng:

- **Wails v2** (framework gắn Go backend với webview frontend)
- **Go** cho toàn bộ business logic (module name: `HVRIns`)
- **Vue 3 + TypeScript** cho giao diện

Về chức năng, đây là **công cụ tự động đăng ký (register) và xác minh (verify) tài khoản
Instagram/Facebook** ở quy mô lớn (chạy nhiều luồng song song qua proxy, quản lý pool
cookie/mail/proxy, có runner/scheduler...).

> ⚠️ **Cảnh báo về `CLAUDE.md`**
> File `CLAUDE.md` ở gốc repo mô tả nhiệm vụ "dựng *frontend foundation*" cho một app
> "**chưa triển khai backend business logic**". Điều này **mâu thuẫn hoàn toàn** với thực tế:
> backend Go đã rất đồ sộ (`app.go` 317KB, `app_register.go` 219KB, cả cây `internal/instagram`
> 2960 file). `CLAUDE.md` là tàn dư cũ, **không đáng tin** và nên được viết lại để mô tả app thật,
> nếu không nó sẽ làm lạc hướng bất kỳ ai (hoặc AI) đọc nó sau này.

---

## 2. Vấn đề hiện tại — tại sao cần dọn?

Chạy `git ls-files | wc -l` cho ra **~3327 file** được track. Phần lớn là code thật,
nhưng **thư mục gốc đang rất bừa bộn**:

| Loại lộn xộn | Ví dụ | Vấn đề |
|--------------|-------|--------|
| File Go khổng lồ ở gốc | `app.go` (317KB/7315 dòng), `app_register.go` (219KB) | Tất cả là `package main` nằm phẳng ở gốc, khó tìm, khó đọc |
| Script Python lạc | `_patch_datr_diag.py`, `decode_request.py` | Hardcode đường dẫn máy khác (`E:/WEMAKE/...`), không liên quan build |
| File test chứa secret | `test_accounts_eaag.txt`, `test_accounts_fresh.txt` | **Chứa cookie + token Facebook THẬT**, đang bị commit |
| Markdown rải rác ở gốc | `NVRINS_BUILD_GUIDE.md`, `README_TEST_EAAG.md` | Nên nằm trong `docs/` |
| ~18 chương trình "scratch" trong `cmd/` | `cmd/test273`, `cmd/testbody`, `cmd/regtest`... | Test thủ công, không assertion, hardcode secret |
| Rác build | `scripts/__pycache__/*.pyc` | Bytecode Python không nên commit |
| `Config/` lẫn lộn | `Config/Cookie/cookie_initial.txt` (48KB token thật) lẫn với template trống | Vừa là mẫu, vừa là secret bị lộ |

→ Mục tiêu: **gốc repo gọn gàng, mọi thứ có chỗ rõ ràng, không lộ secret**, theo khung chuẩn.

---

## 3. Khung chuẩn đích (wails-go-vue/structured.md)

```
<app-name>/
├── docs/                   # tài liệu
├── infra/installer/        # script tạo installer (NSIS, Inno Setup)
├── scripts/                # script build/run/package
├── config/                 # file cấu hình mẫu
├── build/                  # output build của Wails (gitignored)
├── sidecar/<service>/      # binary phụ đi kèm (tuỳ chọn)
├── cmd/app/main.go         # entry point Wails (wails.Run + bind)
├── internal/               # code Go private
│   ├── app/                # bootstrap, DI, expose bridge cho FE
│   ├── domain/             # entity & business rule cốt lõi
│   ├── usecase/            # workflow ứng dụng
│   └── adapter/
│       ├── repository/     # lưu trữ (file/DB local)
│       └── external/       # client gọi API ngoài
├── frontend/src/{components,composables,features,services,main.ts}
├── tests/{go,frontend}     # test suite ngoài
└── wails.json, go.mod, go.sum, README.md, .gitignore
```

**Nguyên tắc của khung:** tách bạch Go vs FE; `cmd/` chỉ bootstrap, logic nằm trong
`internal/`; FE chia theo domain trong `features/`; `build/` bị gitignore. Và quan trọng:
*"nếu nhiều tầng nhưng app vẫn nhỏ thì giảm độ phức tạp"* — **đừng over-engineer**.

---

## 4. ⭐ QUYẾT ĐỊNH KIẾN TRÚC QUAN TRỌNG NHẤT

> **`main.go` sẽ Ở LẠI thư mục gốc, KHÔNG chuyển vào `cmd/app/main.go`.**
> Đây là một **điểm lệch chuẩn có chủ ý và chính đáng.**

### Tại sao?

Khung chuẩn muốn entry point ở `cmd/app/main.go`. Nhưng `main.go` của dự án này có 2 ràng buộc
kỹ thuật khiến việc chuyển nó đi sẽ **làm hỏng build**:

1. **`//go:embed all:frontend/dist`** — Go nhúng toàn bộ web đã build vào file `.exe`.
   Đường dẫn của `go:embed` tính tương đối so với **thư mục chứa file `.go`**, và **Go cấm
   dùng `../` để trỏ ra thư mục cha**. Nếu `main.go` nằm ở `cmd/app/`, nó **không thể** với tới
   `frontend/dist` ở gốc → build lỗi.

2. **`wails build` giả định `package main` ở gốc module.** `wails.json` không có trường nào
   để chỉ định vị trí package main. Chuyển `main.go` vào `cmd/app/` thì `wails build` chạy từ
   gốc sẽ không tìm thấy main.

Ngoài ra, giữ `main.go` ở gốc còn giúp **không phải đổi**:
- ldflags `-X main.AppVersion=...` trong `build.bat`
- namespace binding FE (`wailsjs/go/main/...`)
- `.vscode/launch.json` (`program: ${workspaceFolder}`)

### Vậy ta vẫn theo chuẩn ở điểm nào?

Tinh thần cốt lõi của chuẩn là *"`cmd` chỉ bootstrap, logic nằm trong `internal/`"*. Ta đạt
được điều đó bằng cách:

- **Giữ `main.go` ở gốc nhưng làm nó MỎNG**: chỉ còn `go:embed`, đọc `AppVersion`, gọi
  `app.New()`, `wails.Run(...)`. (Hiện tại `main.go` đã khá mỏng — 117 dòng.)
- **Chuyển toàn bộ logic** (struct `App` + 129 method, 12 file `package main`) vào
  `internal/app/` thành `package app`.

→ Kết quả: gốc repo gọn (chỉ còn `main.go` + các file config chuẩn), logic nằm gọn trong
`internal/app/`, mà build vẫn chạy. Đây là sự đánh đổi đúng cho một app Wails thật.

> 📌 Nếu sau này thực sự muốn có `cmd/app/main.go` theo đúng chữ của chuẩn, sẽ phải: di dời
> `embed.FS` sang một package mà thư mục của nó là cha của `frontend/dist`, đổi ldflags, và cấu
> hình để `wails build` tìm được main không ở gốc — **phức tạp, rủi ro cao, không khuyến nghị
> cho giai đoạn này.**

---

## 5. Mục tiêu cụ thể của đợt tái cấu trúc

| Mục tiêu | Trạng thái sau khi xong |
|----------|--------------------------|
| Gốc repo gọn | Chỉ còn `main.go`, `wails.json`, `go.mod/sum`, `README.md`, `.gitignore`, `.gitattributes`, `CLAUDE.md` + các thư mục chuẩn |
| Logic Go gom lại | 12 file `package main` → `internal/app/` (`package app`), `app.go` được tách nhỏ |
| `cmd/` sạch | Chỉ giữ tool thật (icongen → `tools/`), xoá scratch |
| Secret không bị track | Cookie/token thật được gỡ khỏi git + rotate |
| Docs có tổ chức | `docs/rebuild/`, `docs/flows/`, `docs/archive/` |
| FE chuẩn hơn | `bridge/` → `services/`, `modules/` → `features/` (làm sau, theo từng feature) |
| Có khung `tests/` | `tests/go/`, `tests/frontend/` cho test black-box mới |

---

## 6. Phạm vi & cách tiếp cận

- **Đây là kế hoạch — chưa thực thi.** Việc thực thi chia thành các **pha có thứ tự an toàn**
  (xem [04-ke-hoach-thuc-thi.md](04-ke-hoach-thuc-thi.md)).
- **Pass 1** chỉ làm những việc rủi ro thấp + cú chuyển `internal/app` (một commit nguyên tử).
- **Pass 2+** (đổi tên FE, ánh xạ `internal/` sâu) là **tuỳ chọn, để sau**.
- Mỗi bước đều có **lệnh kiểm tra** và phải xanh trước khi đi tiếp.

→ Đọc tiếp: [01-hien-trang.md](01-hien-trang.md) để xem chi tiết hiện trạng.
