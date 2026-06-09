# CLAUDE.md

## Vai trò của bạn

Bạn là kỹ sư phần mềm senior chịu trách nhiệm dựng **frontend foundation** cho một desktop app chạy bằng **Wails + Go + Vue 3 + TypeScript**.

Giai đoạn hiện tại **chưa triển khai backend business logic**.
Nhiệm vụ của bạn là dựng **khung frontend chuẩn, sạch, dễ mở rộng** để nhóm có thể phát triển lâu dài.

---

## Mệnh lệnh thực thi bắt buộc

1. **Đọc toàn bộ `REQUIREMENTS.md` trước khi làm bất cứ thứ gì.**
2. **Tạo Task List dạng checklist markdown trước.**
3. **Sau khi tạo checklist, phải thực hiện ngay trong cùng run. Không chỉ plan rồi dừng.**
4. Nếu thiếu thông tin nhỏ, **tự chọn mặc định hợp lý**, ghi rõ vào mục assumptions, và tiếp tục làm.
5. Không hỏi lại những câu không thật sự cần thiết.
6. Sau mỗi mốc lớn, cập nhật tài liệu liên quan.
7. Chỉ được báo hoàn thành khi code chạy được và đã verify bằng command thực tế.

---

## Mục tiêu của giai đoạn này

Dựng một bộ khung frontend cho desktop app với các đặc điểm sau:

- chạy trong Wails
- frontend dùng Vue 3 + TypeScript
- có router, state management, composables, bridge layer
- có app shell desktop chuẩn
- có đủ 4 màn hình khung:
  - Accounts
  - Flow Settings
  - Proxy Settings
  - View Settings
- dùng mock data/mock bridge để UI chạy được trước
- chưa làm backend business logic thật

---

## Phạm vi công việc bạn được phép làm

### Được phép làm

- Khởi tạo Wails app nếu cần.
- Khởi tạo frontend Vue 3 + TS.
- Thiết lập cấu trúc thư mục frontend.
- Thêm Vue Router.
- Thêm Pinia.
- Tạo layouts, pages, modules, composables, stores, services, bridge layer.
- Tạo mock services và wiring.
- Tạo component nền cho data grid, toolbar, dialogs, shell.
- Tạo docs mô tả kiến trúc frontend.
- Viết test foundation tối thiểu.

### Không được làm

- Không tự triển khai backend nghiệp vụ.
- Không thêm database.
- Không thêm flow engine thật.
- Không thêm runner/concurrency thật.
- Không thêm auth phức tạp.
- Không gọi trực tiếp generated Wails bindings từ component/page.
- Không phá vỡ cấu trúc module-first.

---

## Yêu cầu kỹ thuật bắt buộc

### 1. Frontend phải module-first

Không tổ chức code kiểu phẳng, lẫn lộn.
Phải có các module rõ ràng:

- accounts
- flow-settings
- proxy-settings
- view-settings

### 2. Phải có bridge layer

Tuyệt đối không để component import trực tiếp Wails-generated bindings.

Phải có cấu trúc tương đương:

- `frontend/src/bridge/contracts.ts`
- `frontend/src/bridge/client.ts`
- `frontend/src/bridge/mock/*`
- `frontend/src/bridge/wails/*`

Mục tiêu:
- UI không phụ thuộc trực tiếp vào generated code.
- Sau này đổi mock sang binding thật không phải sửa hàng loạt.

### 3. Phải có app shell desktop chuẩn

Bắt buộc có:
- sidebar
- header
- main content
- status bar

### 4. Phải có routing chuẩn

Các route tối thiểu:
- `/`
- `/accounts`
- `/flow-settings`
- `/proxy-settings`
- `/view-settings`

### 5. Accounts module là ưu tiên số 1

Accounts page phải có:
- grid
- toolbar
- filter area
- selection logic
- mock import hook
- column visibility hook
- detail panel hoặc dialog khung

### 6. State management

Dùng Pinia.
Không dùng một global store khổng lồ.

### 7. Composition API

Bắt buộc dùng:
- Vue 3 Composition API
- `<script setup>`
- composables cho logic dùng lại

### 8. Testing

Tối thiểu phải có:
- unit test cho composables quan trọng
- unit test cho accounts store

---

## Cấu trúc thư mục mục tiêu

Bạn phải bám sát cấu trúc này, có thể điều chỉnh nhỏ nếu hợp lý hơn nhưng không được phá vỡ tinh thần kiến trúc:

```text
frontend/
  src/
    app/
    layouts/
    pages/
    modules/
      accounts/
      flow-settings/
      proxy-settings/
      view-settings/
    components/
      ui/
      grid/
      form/
      feedback/
      shell/
    bridge/
      contracts.ts
      client.ts
      mock/
      wails/
    stores/
    composables/
    types/
    utils/
    constants/
    assets/
    styles/
```

---

## Quy tắc implementation bắt buộc

### 1. Không over-engineer

- Không dựng framework quá mức.
- Không dựng abstraction thừa.
- Chỉ dựng đủ cho nền tảng sạch và mở rộng.

### 2. Không under-engineer

- Không làm page khổng lồ.
- Không để toàn bộ logic trong 1 component.
- Không bỏ bridge layer.
- Không bỏ module boundaries.

### 3. Data grid là thành phần nền tảng

Phải thiết kế grid foundation sao cho sau này Accounts page dùng lâu dài được.
Grid tối thiểu phải có:
- row selection
- sorting cơ bản
- visible columns
- sticky header
- row click/double click hook
- context menu hook

### 4. Mock-first

Mọi màn hình phải có mock data để render được.
Không được để page trống vì “chưa có backend”.

### 5. Naming conventions

- component: PascalCase
- composable: `useXxx`
- store: `useXxxStore`
- types: rõ nghĩa, tránh `IData`, `ItemModel`, `BaseResponse` mơ hồ
- route names: constant hoặc enum

---

## Yêu cầu về docs

Bạn phải tạo hoặc cập nhật thêm tài liệu sau nếu chưa có:

- `docs/frontend-architecture.md`
- `docs/ui-wireframes.md`
- `docs/frontend-decisions.md`

Trong đó phải mô tả:
- cấu trúc thư mục
- lý do chia module như hiện tại
- luồng dữ liệu frontend
- bridge layer hoạt động ra sao
- cách sau này backend Go/Wails sẽ gắn vào

---

## Yêu cầu về output cuối cùng

Khi hoàn thành, output cuối phải có đúng các phần sau:

### 1. ✅ Task checklist
- checklist đã hoàn thành

### 2. 🔧 Summary
- đã làm gì
- app shell đã có gì
- module nào đã dựng xong

### 3. 🧩 Files changed
- liệt kê cây thư mục và các file chính đã tạo/sửa

### 4. 🧪 Commands run + results
- các lệnh thực tế đã chạy
- kết quả tóm tắt của từng lệnh
- chỉ claim thành công nếu command thực sự pass

### 5. 🧠 Assumptions
- các giả định đã tự chốt

### 6. ⚠️ Risks & follow-ups
- phần nào hiện đang là mock
- phần nào chờ backend/business gắn vào sau

---

## Tiêu chí hoàn thành bắt buộc

Chỉ được coi là hoàn thành khi đạt đủ:

1. Project chạy được bằng lệnh dev phù hợp.
2. Wails shell + Vue frontend khởi động được.
3. Có app shell desktop hoàn chỉnh.
4. Có đủ 4 page khung.
5. Accounts page render grid với mock data.
6. Có Pinia, Router, composables foundation.
7. Có bridge layer mock + wiring rõ ràng.
8. Có docs frontend.
9. Có test foundation tối thiểu và test pass.

Nếu chưa đạt đủ các mục trên, không được nói là hoàn thành.

---

## Điều đặc biệt quan trọng

- Giai đoạn này **chưa được lạc sang backend nghiệp vụ**.
- Tuy nhiên phải **chừa sẵn điểm nối** để backend Go/Wails bind vào sau.
- Hãy ưu tiên dựng nền frontend đúng ngay từ đầu hơn là làm nhiều tính năng giả nhưng cấu trúc xấu.
