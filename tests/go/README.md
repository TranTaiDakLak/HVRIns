# tests/go — Black-box Go tests

Thư mục này chứa **black-box integration tests** (`package xxx_test`) — tức là test từ góc nhìn
bên ngoài package, không truy cập private symbol.

## Quy tắc

- **Test white-box** (cùng package, truy cập internal) → giữ **cạnh code** (`internal/xxx/xxx_test.go`)
- **Test black-box / integration** → đặt ở đây

Xem thêm: [docs/rebuild/02-cau-truc-dich.md](../../docs/rebuild/02-cau-truc-dich.md) mục 3.

## Cách chạy

```powershell
go test ./tests/go/...
```
