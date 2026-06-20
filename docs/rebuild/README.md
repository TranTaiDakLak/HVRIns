# Kế hoạch tái cấu trúc dự án HVRIns (Hạ Vũ)

> Thư mục này chứa **kế hoạch chi tiết** để dọn dẹp và sắp xếp lại toàn bộ dự án
> theo khung chuẩn [wails-go-vue/structured.md](https://github.com/TranTaiDakLak/structure_standard_pack/blob/main/10-templates/desktop/wails-go-vue/structured.md).
>
> **Đây mới chỉ là KẾ HOẠCH.** Chưa có file nào trong source code bị di chuyển/xoá.
> Đọc hết, hiểu rủi ro, rồi mới thực thi từng bước (xem [04-ke-hoach-thuc-thi.md](04-ke-hoach-thuc-thi.md)).

---

## Đọc theo thứ tự nào?

| # | File | Nội dung | Dành cho ai |
|---|------|----------|-------------|
| 0 | [00-tong-quan.md](00-tong-quan.md) | Dự án này **thực sự** là gì, vấn đề hiện tại, mục tiêu, và **quyết định kiến trúc quan trọng nhất** | Đọc đầu tiên |
| 1 | [01-hien-trang.md](01-hien-trang.md) | Phân tích hiện trạng: thống kê file, cái gì là rác, cái gì là secret bị lộ | Hiểu "tại sao cần dọn" |
| 2 | [02-cau-truc-dich.md](02-cau-truc-dich.md) | Cây thư mục đích (đã chỉnh cho dự án này) + giải thích từng folder | Hiểu "dọn về đâu" |
| 3 | [03-anh-xa-chi-tiet.md](03-anh-xa-chi-tiet.md) | Bảng ánh xạ chi tiết: **mỗi** đường dẫn hiện tại → đích → hành động | Tra cứu khi làm |
| 4 | [04-ke-hoach-thuc-thi.md](04-ke-hoach-thuc-thi.md) | **Các bước thực thi theo thứ tự an toàn** + lệnh kiểm tra sau mỗi bước | Làm theo từng bước |
| 5 | [05-secrets-bao-mat.md](05-secrets-bao-mat.md) | ⚠️ **KHẨN CẤP**: credential thật đang bị commit vào git — cách xử lý | Làm TRƯỚC mọi thứ |
| 6 | [06-go-wails-cho-newbie.md](06-go-wails-cho-newbie.md) | Giải thích khái niệm Go/Wails (package, go:embed, ldflags, bindings...) | Newbie Go đọc để hiểu rủi ro |
| 7 | [07-checklist-rui-ro.md](07-checklist-rui-ro.md) | Checklist tổng + danh sách rủi ro cần canh chừng | In ra, tick từng ô |

---

## TL;DR — 6 điều quan trọng nhất

1. **Đây KHÔNG phải dự án "frontend foundation" như `CLAUDE.md` mô tả.** Nó là một app
   Wails + Go + Vue **đã hoàn chỉnh** (công cụ tự động đăng ký/verify tài khoản
   Instagram/Facebook). `CLAUDE.md` đang lỗi thời và sẽ làm lạc hướng. Xem [01](01-hien-trang.md).

2. **Có credential THẬT đang bị commit** (cookie phiên Facebook + token EAAG trong
   `Config/Cookie/cookie_initial.txt` và `test_accounts_*.txt`). Đây là việc cần xử lý
   **đầu tiên**, độc lập với việc dọn cấu trúc. Xem [05](05-secrets-bao-mat.md).

3. **Sự lộn xộn chính nằm ở thư mục gốc**: 13 file Go khổng lồ (`app.go` 317KB!),
   script Python lạc, file `.txt` test, nhiều file markdown. Giải pháp: gom logic Go
   vào `internal/app/`, giữ `main.go` mỏng ở gốc.

4. **Một điểm "lệch chuẩn" có chủ ý**: khung chuẩn muốn entry ở `cmd/app/main.go`,
   nhưng `main.go` của dự án này **phải ở lại thư mục gốc** vì `//go:embed all:frontend/dist`
   không cho phép trỏ ra thư mục cha (`../`), và `wails build` mặc định tìm `package main`
   ở gốc module. Đây là quyết định đúng đắn, được giải thích kỹ ở [00](00-tong-quan.md) và [02](02-cau-truc-dich.md).

5. **App hiện chỉ build được trên Windows** (các file `*_windows.go` không có bản
   thay thế cho Linux/macOS). Mọi lệnh kiểm tra phải chạy **trên Windows**, và
   `wails build` mới là "cổng kiểm tra" thật sự (không phải `go build ./...`).

6. **Không "đập đi xây lại" phần `internal/`.** Cây `internal/instagram` có ~2960 file
   với pattern plugin-registry rất tinh tế. Việc ánh xạ sâu sang `domain/usecase/adapter`
   là **tuỳ chọn, để sau**, rủi ro cao mà lợi ích thấp cho người mới. Pass đầu chỉ
   gom root `package main` vào `internal/app`.

---

## Nguyên tắc xuyên suốt

- **An toàn trên hết**: mỗi bước phải build/test xanh **trước khi** sang bước sau, và
  commit lại để có thể `git bisect` nếu hỏng.
- **Tường minh cho người mới**: ưu tiên thao tác dễ hiểu, ít rủi ro hơn là refactor "thông minh".
- **Giữ tinh thần chuẩn, không cứng nhắc**: theo *tinh thần* (logic trong `internal/`,
  entry mỏng, FE chia theo feature, gốc gọn) — và ghi rõ những chỗ buộc phải lệch chuẩn.

---

*Tài liệu được tạo từ một đợt phân tích sâu toàn bộ repo (8 agent đọc song song
từng tiểu hệ thống). Mọi con số/đường dẫn đều lấy từ code thật tại thời điểm phân tích.*
