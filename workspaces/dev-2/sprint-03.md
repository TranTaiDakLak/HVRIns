# Dev 2 — Sprint 03: Tái cấu trúc Frontend

> ⛔ **ĐIỀU KIỆN BẮT ĐẦU:** Dev 1 Sprint 02 phải = DONE (binding đã đổi `go/main`→`go/app`,
> import `bridge/wails/*.ts` đã sửa). Nếu chưa, bạn sẽ đụng cùng file → conflict.
>
> Tham chiếu: `docs/rebuild/04` Pha 6 + `docs/rebuild/03` mục D. **Làm từng feature một**, build sau mỗi bước.

---

## S03-D2-T001 — Bật alias `@/` + convert import
**Vì sao trước tiên (R-11):** có 192 import `../` và 0 alias. Bật `@/` rồi đổi import → di chuyển
file sau này không vỡ đường dẫn.
**Việc:**
- `frontend/tsconfig.json`: thêm `compilerOptions.baseUrl: "."` + `paths: { "@/*": ["src/*"] }`
  (khớp `vite.config.ts` đã map `@ → src`).
- Đổi dần import `'../../../...'` → `'@/...'` (ưu tiên các file sâu: AccountsPage 21 import, InteractionSetupPage 13, GeneralSettingsPage 10).
**Test:**
```powershell
npm --prefix frontend run build
```
**DONE khi:** build xanh, không còn (hoặc còn rất ít) import `../../`.

---

## S03-D2-T002 — Xoá stub + (tuỳ chọn) làm phẳng entry
**Việc:**
- Xoá 2 stub chết: `frontend/src/main.ts` (`export {}`), `frontend/src/App.vue` (`<div/>`). Xác nhận
  `index.html` đang trỏ `src/app/main.ts` (không trỏ stub).
- (Tuỳ chọn) Làm phẳng: `src/app/main.ts`→`src/main.ts`, `src/app/App.vue`→`src/App.vue`,
  `src/app/router/`→`src/router/`; cập nhật `index.html` `<script src>` và import liên quan.
  Nếu thấy rủi ro, GIỮ `src/app/` và chỉ xoá stub — ghi quyết định vào decision-log (D-0xx).
**Test:** `npm run build` + `wails dev` (cửa sổ mở, router chạy).
**DONE khi:** không còn stub; app chạy đúng entry.

---

## S03-D2-T003 — bridge/ → services/
**⚠ Quy tắc sống còn (R-12):** GIỮ NGUYÊN độ sâu thư mục để `wails/*.ts` còn resolve `../../../wailsjs`.
`services/wails/` cùng độ sâu với `bridge/wails/` nên `../../../wailsjs` vẫn đúng — đừng đổi độ sâu.
**Việc:**
```powershell
git mv frontend/src/bridge frontend/src/services
```
- Cập nhật mọi import `'@/bridge/...'`/`'.../bridge/...'` → `services`.
- Kiểm `services/client.ts`, `contracts.ts`, `mock/`, `wails/` còn nguyên cấu trúc.
**Test:** `npm run build` + `wails dev` (FE gọi backend qua services OK — mock & wails đều chạy).
**DONE khi:** đổi tên xong, FE chạy thật.

---

## S03-D2-T004 — modules/ → features/ + gom pages + vitest
**Việc (làm từng feature):**
```powershell
git mv frontend/src/modules frontend/src/features
```
- Gom page tương ứng vào feature: `pages/AccountsPage.vue` → `features/accounts/pages/`; tương tự
  auth-source. Component feature-specific: `components/settings/` + `src/schema/` → `features/settings/`;
  `MailDomainStatsTable.vue`,`RegVerStatsTable.vue` → `features/reg-stats/components/`.
- Cập nhật import + `router/routes.ts`.
- Thêm script test: trong `frontend/package.json` thêm `"test": "vitest run"`.
**Test (sau MỖI feature):**
```powershell
npm --prefix frontend run build
wails dev
```
**DONE khi:** tất cả page/feature build + chạy; `npm --prefix frontend run test` chạy được (kể cả 0 test).

> Giữ `components/{ui,grid,form,feedback,shell}/`, `composables/`, `stores/`, `types/`, `constants/`,
> `styles/`, `assets/` nguyên (đã chuẩn).

---

### Sau Sprint 03
Cập nhật progress + task-board + completed-log + decision-log (lựa chọn làm phẳng entry).
