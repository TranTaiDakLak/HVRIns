# tests/frontend — Frontend integration tests (Vitest)

Thư mục này dành cho **Vitest integration tests** — test phối hợp nhiều component/module,
hoặc test cần mount toàn bộ app context.

## Quy tắc

- **Unit test** component/composable/store đơn lẻ → đặt cạnh source trong `frontend/src/`
- **Integration / E2E test** → đặt ở đây

Cấu hình Vitest: `frontend/vitest.config.ts` (hoặc `vite.config.ts` nếu dùng chung).
Đảm bảo `include` pattern bao gồm `tests/**` nếu cần.

## Cách chạy

```powershell
npm --prefix frontend run test
# hoặc
npm --prefix frontend run test:watch
```

> Sprint 03 sẽ thêm `"test"` script vào `frontend/package.json` khi setup Vitest.
