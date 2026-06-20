# REQUIREMENTS.md

# Project: Desktop Account Manager - Frontend Foundation (Vue + Wails + Go)

## 1. Mục tiêu của giai đoạn này

Giai đoạn hiện tại **chưa triển khai nghiệp vụ backend**.
Mục tiêu là dựng **khung frontend thật chuẩn**, chạy trên **Wails + Go + Vue 3 + TypeScript**, để sau này gắn nghiệp vụ backend vào mà không phải đập lại cấu trúc.

Trọng tâm của giai đoạn này:

1. Dựng shell desktop app bằng Wails.
2. Dựng kiến trúc frontend theo module.
3. Dựng app shell, layout, router, state management, UI foundation.
4. Dựng các màn hình khung:
   - Accounts
   - Flow Settings
   - Proxy Settings
   - View Settings
5. Chuẩn hoá cách frontend gọi Go/Wails bindings qua một lớp bridge thống nhất.
6. Dựng mock data + mock service để UI chạy được trước khi có backend nghiệp vụ thật.

---

## 2. Phạm vi công việc trong giai đoạn này

### In scope

- Khởi tạo project Wails dùng Go host + Vue frontend.
- Cấu trúc thư mục frontend chuẩn, dễ mở rộng.
- Cài TypeScript.
- Cài router.
- Cài Pinia.
- Tổ chức composables.
- Tổ chức UI components dùng lại.
- Tổ chức modules theo feature.
- Tạo layout chính cho desktop app.
- Tạo navigation/sidebar/header/status bar.
- Tạo các màn hình khung với mock data.
- Tạo lớp `bridge` để tách biệt UI với Wails-generated bindings.
- Tạo conventions rõ ràng cho component, store, composable, service, types.
- Tạo theme/light foundation phù hợp desktop tool.
- Viết tài liệu cấu trúc để sau này backend Go cắm vào.

### Out of scope

- Chưa làm nghiệp vụ backend thật.
- Chưa làm xử lý file thật.
- Chưa làm flow engine thật.
- Chưa làm proxy testing thật.
- Chưa làm concurrency runtime thật.
- Chưa làm persistence thật.
- Chưa làm auth/user system.
- Chưa làm import/export production-ready.

---

## 3. Stack công nghệ bắt buộc

- **Desktop shell**: Wails
- **Host runtime**: Go
- **Frontend**: Vue 3
- **Language**: TypeScript
- **Bundler/dev frontend**: Vite
- **State management**: Pinia
- **Routing**: Vue Router
- **Styling**: ưu tiên Tailwind CSS hoặc CSS variables + utility classes, nhưng phải thống nhất từ đầu
- **Testing frontend**: Vitest

Ghi chú:
- Không dùng Vue Options API cho code mới.
- Dùng **Composition API** + `<script setup>`.
- Frontend phải được thiết kế như một SPA desktop-first nằm bên trong Wails.

---

## 4. Nguyên tắc kiến trúc bắt buộc

### 4.1. Frontend là module-first

Không tổ chức code theo kiểu toàn bộ `components`, `services`, `views` lẫn lộn không biên giới.
Phải tổ chức theo **feature/module**.

Các module bắt buộc ở giai đoạn đầu:

- `accounts`
- `flow-settings`
- `proxy-settings`
- `view-settings`
- `dashboard` (nhẹ, có thể chỉ là landing page)
- `app-shell`

### 4.2. UI không được gọi trực tiếp Wails-generated bindings từ component

Phải có một lớp trung gian, ví dụ:

- `src/bridge/`
- `src/services/bridge/`

Mục tiêu:
- Component không import trực tiếp các file generated của Wails.
- Sau này thay mock bằng binding thật không ảnh hưởng component tree.
- Có thể test frontend dễ hơn.

### 4.3. Tách rõ 4 lớp frontend

Frontend phải có đủ 4 lớp sau:

1. **Presentation layer**
   - pages
   - layouts
   - ui components

2. **State layer**
   - pinia stores
   - view state
   - selected rows
   - filters
   - presets

3. **Use-case / composable layer**
   - các composables dùng lại
   - logic tái sử dụng cho grid, filters, dialogs, bridge calls

4. **Bridge/service layer**
   - adapter gọi Wails bindings
   - mock services
   - DTO mapping

### 4.4. Dữ liệu mock trước, binding thật sau

Ở giai đoạn này, toàn bộ màn hình phải chạy được với mock data.
Không được chờ backend thật rồi mới dựng UI.

### 4.5. Desktop-first

Đây là desktop app, không phải website mobile-first.
Ưu tiên:
- bảng lớn
- sidebar cố định
- panel trái/phải
- toolbar thao tác nhanh
- keyboard-friendly
- data dense but readable

---

## 5. Cấu trúc thư mục frontend bắt buộc

Project phải tách rõ phần Go/Wails host và phần frontend.

Cấu trúc mục tiêu:

```text
project-root/
├─ app.go
├─ main.go
├─ wails.json
├─ frontend/
│  ├─ index.html
│  ├─ package.json
│  ├─ tsconfig.json
│  ├─ vite.config.ts
│  └─ src/
│     ├─ app/
│     │  ├─ App.vue
│     │  ├─ main.ts
│     │  ├─ router/
│     │  ├─ providers/
│     │  └─ guards/
│     ├─ layouts/
│     │  ├─ AppLayout.vue
│     │  └─ EmptyLayout.vue
│     ├─ pages/
│     │  ├─ DashboardPage.vue
│     │  ├─ AccountsPage.vue
│     │  ├─ FlowSettingsPage.vue
│     │  ├─ ProxySettingsPage.vue
│     │  └─ ViewSettingsPage.vue
│     ├─ modules/
│     │  ├─ accounts/
│     │  │  ├─ components/
│     │  │  ├─ store/
│     │  │  ├─ services/
│     │  │  ├─ composables/
│     │  │  ├─ types/
│     │  │  └─ mappers/
│     │  ├─ flow-settings/
│     │  ├─ proxy-settings/
│     │  └─ view-settings/
│     ├─ components/
│     │  ├─ ui/
│     │  ├─ grid/
│     │  ├─ form/
│     │  ├─ feedback/
│     │  └─ shell/
│     ├─ bridge/
│     │  ├─ client.ts
│     │  ├─ contracts.ts
│     │  ├─ mock/
│     │  └─ wails/
│     ├─ stores/
│     │  ├─ app.store.ts
│     │  └─ ui.store.ts
│     ├─ composables/
│     │  ├─ useDataGrid.ts
│     │  ├─ useDialog.ts
│     │  ├─ useSelection.ts
│     │  ├─ useAsyncState.ts
│     │  └─ useKeyboardShortcuts.ts
│     ├─ types/
│     ├─ utils/
│     ├─ constants/
│     ├─ assets/
│     └─ styles/
└─ docs/
   ├─ frontend-architecture.md
   └─ ui-wireframes.md
```

### Quy tắc thư mục

- `pages/` chỉ để page-level composition.
- `modules/` mới là nơi chứa phần lớn logic theo feature.
- `components/ui/` chỉ chứa component dùng chung thật sự.
- `bridge/` là nơi duy nhất được biết về Wails bindings/generated API.
- `stores/` toàn cục chỉ cho state dùng xuyên app.
- state theo module để trong module tương ứng.

---

## 6. Kiến trúc navigation và layout

App phải có một layout desktop chuẩn.

### Layout chính bắt buộc

- Sidebar trái
- Header trên cùng
- Content area ở giữa
- Optional right panel / inspector
- Status bar dưới cùng

### Menu chính bắt buộc

- Dashboard
- Accounts
- Flow Settings
- Proxy Settings
- View Settings

### Hành vi UI

- Sidebar collapse được.
- Header có quick actions.
- App nhớ route cuối cùng của user trong session hiện tại.
- Layout phải đủ chỗ cho data grid lớn.

---

## 7. Cấu trúc module Accounts

Đây là module quan trọng nhất.

### Mục tiêu của Accounts module

- Hiển thị bảng master grid.
- Hỗ trợ mock import vào bảng.
- Hỗ trợ search/filter/sort trên dữ liệu mock.
- Hỗ trợ row selection.
- Hỗ trợ toolbar actions.
- Hỗ trợ panel chi tiết hoặc dialog edit.

### Cột tối thiểu cần hiển thị

- STT
- UID
- Password
- Cookie
- Token
- Email
- Pass Mail
- Mail Recovery
- 2FA
- Import Time
- Source Code
- Status
- Note

### Thành phần module Accounts

- `AccountsToolbar`
- `AccountsGrid`
- `AccountsFilters`
- `AccountsDetailPanel`
- `AccountsImportDialog` (mock)
- `AccountsColumnsManager`

### Store của Accounts phải quản lý

- rows
- selectedRowIds
- filters
- sort state
- visible columns
- loading state
- current detail row

---

## 8. Cấu trúc module Flow Settings

### Mục tiêu

- Tạo khung UI cho cấu hình luồng chạy.
- Chưa cần runner thật.
- Phải có dữ liệu mock để nhìn được cấu trúc sản phẩm.

### Tính năng khung bắt buộc

- Danh sách flow bên trái hoặc tab list.
- Form chỉnh tên luồng, mô tả, engine type.
- Danh sách step dạng table/list.
- Thêm / sửa / xoá / đổi thứ tự step.
- Mỗi step có:
  - step no
  - action key
  - input text
  - param1..param5
  - timeout
  - retry
  - enable flag

---

## 9. Cấu trúc module Proxy Settings

### Mục tiêu

- Dựng form quản lý proxy ở mức UI foundation.

### Tính năng khung bắt buộc

- Danh sách proxy
- Form add/edit proxy
- Bulk import proxy (mock)
- Proxy test status giả lập
- Gán proxy vào selected accounts (mock action)

### Trường proxy tối thiểu

- Name
- Host
- Port
- Username
- Password
- Type
- Note
- Last Test Result

---

## 10. Cấu trúc module View Settings

### Mục tiêu

Cho phép tuỳ biến cách hiển thị bảng Accounts.

### Tính năng khung bắt buộc

- Chọn cột hiển thị
- Đổi thứ tự cột
- Pin cột trái
- Đổi density
- Bật/tắt wrap text
- Lưu preset hiển thị
- Áp dụng preset cho Accounts grid

---

## 11. State management bắt buộc

Dùng Pinia.

### State toàn cục

Dùng cho:
- app shell state
- sidebar collapsed
- theme
- current module
- notifications
- global loading

### State theo module

- Accounts: để trong `modules/accounts/store`
- Flow Settings: để trong module riêng
- Proxy Settings: để trong module riêng
- View Settings: để trong module riêng

### Quy tắc bắt buộc

- Không nhét toàn bộ state vào 1 store lớn.
- Store name phải rõ nghĩa, theo convention `useXxxStore`.
- Data fetching hoặc bridge calls không đặt bừa trong component nếu có thể đặt vào store/service/composable rõ ràng hơn.

---

## 12. Composables bắt buộc

Phải dùng Composition API + composables để tái sử dụng logic frontend.

Tối thiểu phải có:

- `useDataGrid()`
- `useSelection()`
- `useDialog()`
- `useAsyncState()`
- `useKeyboardShortcuts()`
- `useColumnVisibility()`

### Quy tắc composables

- Tên bắt đầu bằng `use`.
- Không nhét business flow của cả module vào một composable khổng lồ.
- Composable chỉ chứa logic dùng lại được.

---

## 13. Bridge layer bắt buộc

Đây là phần cực quan trọng với Wails.

Frontend **không được** import trực tiếp generated bindings trong page/component/module UI.

Phải có lớp như sau:

- `bridge/contracts.ts`
- `bridge/client.ts`
- `bridge/mock/*`
- `bridge/wails/*`

### Trách nhiệm

- `contracts.ts`: định nghĩa kiểu dữ liệu và các interface frontend cần.
- `mock/*`: implementation giả để UI chạy trước.
- `wails/*`: implementation gọi Wails-generated bindings thật.
- `client.ts`: nơi chọn mock hay real implementation.

### Quy tắc

- Toàn bộ page/module gọi qua bridge/service.
- Không để component biết chi tiết generated code của Wails.
- Phải thiết kế sao cho sau này chuyển từ mock sang real chỉ đổi wiring.

---

## 14. Wails integration foundation

Chưa làm backend nghiệp vụ, nhưng phải chuẩn hoá foundation tích hợp với Wails ngay từ đầu.

### Bắt buộc

- Có lớp frontend gọi bindings Go qua bridge.
- Có event integration point để sau này nhận status update từ Go.
- Có chỗ tách riêng cho runtime events.
- Có adapter cho:
  - file dialog
  - app window actions
  - notifications
  - runtime events

### Quy tắc

- Không trộn runtime event handling vào page component lớn.
- Event listener phải được quản lý tập trung.
- Khi unmount phải cleanup listener đúng cách.

---

## 15. Thiết kế UI/UX bắt buộc

### Phong cách

- Desktop-first
- Sạch, gọn, rõ trạng thái
- Tập trung vào bảng dữ liệu lớn
- Dễ thao tác chuột phải / toolbar / keyboard

### Bắt buộc có

- Toolbar nhất quán
- Panel/dialog nhất quán
- Empty state tử tế
- Loading state tử tế
- Error state tử tế
- Confirm dialog chuẩn
- Toast/notification chuẩn

### Data grid

Grid là thành phần lõi, phải thiết kế làm reusable component foundation.

Grid tối thiểu phải hỗ trợ:
- sort
- filter cơ bản
- multi-select
- visible columns
- sticky header
- row highlight
- double click row
- context menu hook

---

## 16. Testing foundation

Bắt buộc chuẩn bị test foundation ngay từ đầu.

### Tối thiểu phải có

- Unit test cho composables quan trọng
- Unit test cho stores
- Component test cho các UI component phức tạp nếu có

### Ưu tiên test

1. `useDataGrid`
2. `useSelection`
3. Accounts store
4. Column visibility logic
5. Bridge mock wiring

---

## 17. Quy tắc code frontend bắt buộc

- Dùng TypeScript strict hợp lý.
- Dùng `<script setup>`.
- Không viết component quá 300-400 dòng nếu tránh được.
- Tách component con khi UI bắt đầu có nhiều responsibility.
- Không viết store god-object.
- Không viết page god-component.
- Không dùng magic strings lung tung cho route, event, action.
- Dùng constants/enums/types rõ ràng.

---

## 18. Mốc hoàn thành của giai đoạn này

Giai đoạn này chỉ được coi là xong khi đạt đủ:

1. App Wails khởi chạy được.
2. Frontend Vue chạy ổn bên trong app.
3. Có layout desktop hoàn chỉnh.
4. Có đủ 4 màn hình khung:
   - Accounts
   - Flow Settings
   - Proxy Settings
   - View Settings
5. Accounts có grid chạy bằng mock data.
6. Có Pinia, Router, composables foundation.
7. Có bridge layer mock + wiring sẵn cho Wails.
8. Có tài liệu mô tả cấu trúc frontend.
9. Có test foundation tối thiểu.

---

## 19. Thứ tự triển khai bắt buộc

1. Khởi tạo Wails + Vue + TS.
2. Dựng app shell + layout.
3. Dựng router + navigation.
4. Dựng stores toàn cục.
5. Dựng bridge layer.
6. Dựng Accounts module.
7. Dựng Flow Settings module.
8. Dựng Proxy Settings module.
9. Dựng View Settings module.
10. Dựng test foundation.
11. Cập nhật docs.

---

## 20. Không được làm trong giai đoạn này

- Không mở rộng sang backend business logic.
- Không tự thêm database nếu chưa được yêu cầu.
- Không tự thêm flow engine thật.
- Không tự thêm task queue.
- Không tự thêm auth phức tạp.
- Không tự đổi kiến trúc sang web app/server app riêng biệt.
- Không bỏ qua bridge layer rồi gọi Wails binding trực tiếp từ component.

---

## 21. Kết quả mong muốn

Kết quả cuối cùng phải là một **frontend foundation chuẩn, sạch, desktop-first**, chạy trên **Vue + Wails + Go**, đủ vững để sau này cắm backend nghiệp vụ, flow engine, file processing, proxy runtime, mail types, concurrency engine mà không phải đập lại cấu trúc frontend.
