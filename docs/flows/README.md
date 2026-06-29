# HVR — Tài liệu FLOW dự án (Facebook Register & Verify)

> Bộ tài liệu **mô tả luồng chạy thực tế** của HVR — desktop app **Wails (Go backend) + Vue 3 + TypeScript** dùng để tự động **REGISTER** (đăng ký) và **VERIFY** (xác minh email) tài khoản Facebook trên 4 họ platform: **Android FB4A**, **iOS native FBIOS**, **WebAndroid**, **Web MFB**.

Đây là **file index** (mục lục) cho toàn bộ thư mục `docs/facebook/`. Bắt đầu từ đây nếu bạn lần đầu đọc codebase, hoặc dùng [bảng "muốn tìm X → đọc file nào"](#-muốn-tìm-x--đọc-file-nào) để nhảy thẳng tới phần cần.

---

## 🎯 Mục đích thư mục `docs/facebook/`

Thư mục này chứa **tài liệu FLOW** — nghĩa là mô tả **luồng chạy end-to-end thực tế** của ứng dụng, đọc trực tiếp từ code (`app.go`, `internal/**`, `frontend/src/**`), với `file:line` cụ thể, trích đoạn code load-bearing, sơ đồ luồng và edge case. Mục tiêu:

- Người mới đọc hiểu **dữ liệu đi từ đâu đến đâu**: từ lúc bấm nút "Chạy" trên UI → backend Go → kết quả realtime trả lại UI → ghi file.
- Người maintain biết **sửa ở đâu khi thêm version / platform mới** (xem [runbook](./add-facebook-reg-version.md)).
- Phân biệt rõ **phần nào là logic thật, phần nào là gotcha/edge case** dễ vấp.

Khác với `docs/old-docs/` (xem [ghi chú bên dưới](#-ghi-chú-về-docsold-docs)) là tài liệu **lịch sử / thiết kế cũ** — không phản ánh đúng code hiện tại 100%.

---

## 📚 Mục lục tài liệu

Đọc theo thứ tự số (00 → 06) nếu muốn hiểu toàn bộ. Mỗi file độc lập, có sơ đồ riêng và bảng tham chiếu `file:line`.

| # | File | Nội dung chính |
|---|------|----------------|
| **00** | [00-tong-quan-kien-truc.md](./00-tong-quan-kien-truc.md) | **Tổng quan & kiến trúc.** Tech stack Wails (WebView2, JS↔Go, EventsEmit), cây thư mục, vòng đời app (`main` → `wails.Run` → `App.startup`), [sơ đồ tổng từ click "Chạy" đến kết quả](./00-tong-quan-kien-truc.md#4-sơ-đồ-tổng--từ-click-chạy-đến-kết-quả), các "mode" chạy, plugin registry (factory), worker pool, danh sách method App + danh sách event, edge case. **Đọc file này trước.** |
| **01** | [01-luong-register.md](./01-luong-register.md) | **Luồng REGISTER end-to-end.** Từ frontend → [`RunRegister`](../../app.go) → spawner → worker → dispatch per-platform (Android Bloks / iOS FBIOS / WebAndroid / s399 legacy). Gating, multi-version round-robin, **Datr/Cookie pool** (`PartitionedDatrPool`), EAA token pre-fetch, ghi file + đẩy sang verify, stop/drain/complete. |
| **02** | [02-luong-verify.md](./02-luong-verify.md) | **Luồng VERIFY end-to-end.** [`App.RunVerify`](../../app.go) → `runner.RunVerify` (FIFO pool) → `runOneAccount` (retry 2 tầng) → token acquisition per-platform (EAAAAU/EAAAAAY/cookie) → [`verifybase` pipeline 8 bước](./02-luong-verify.md#6-verifybaserunverify--pipeline-chung-8-bước) → Live/Die detection + chống false-positive → ghi file. |
| **03** | [03-runner-scheduler.md](./03-runner-scheduler.md) | **Runner / Scheduler / Concurrency.** Cấu trúc `RunConfig`/`AccountInput`/`AccountResult`, FIFO worker pool, `runOneAccount`, `switch verifyPlatform` (token-type), pass 2 `RetryUnknownNow`, hai context (`poolCtx` vs `workerCtx`) + soft-stop, **5 nhánh dispatch** trong `app.go`, sticky proxy. |
| **04** | [04-frontend-ui-events.md](./04-frontend-ui-events.md) | **Frontend UI & event realtime.** Bridge layer (mock ↔ wails), batch/throttle backend + RAF frontend, [bảng tổng hợp TẤT CẢ event](./04-frontend-ui-events.md#3-bảng-tổng-hợp-tất-cả-event) (payload + `file:line` cả Go lẫn Vue), luồng VERIFY/REGISTER trên grid, split mode 2 pane, dedupe UID. |
| **05** | [05-config-data-pools.md](./05-config-data-pools.md) | **Config / Data files / Pools / UA Builder.** `internal/fbdata` (FBAV/FBBV), `internal/cookie` (datr files), `uabuilder` (3 mode UA), `PartitionedDatrPool` lifecycle, SimProfile/HNI/Locale, iOS native data, `phone_database`, mapping `interaction.json` → hành vi pool/UA. |
| **06** | [06-subsystems.md](./06-subsystems.md) | **Subsystem phụ trợ.** Email (`Service`/factory/CredPool/OTP reader), Proxy (Manager/Pool/StickyManager/CheckIP), 2FA (Android Enable2FA + TOTP), Upload (avatar + banclone.pro), Result files (writer 2 tầng + format), Counters/Stats, Stop/Cleanup. |

### Runbook (tài liệu thao tác — khác với tài liệu flow)

| File | Nội dung |
|------|----------|
| [add-facebook-reg-version.md](./add-facebook-reg-version.md) | **Runbook thêm Reg / Verify version mới (Android FB4A là chính).** Hướng dẫn từng bước (§1–§10 thêm reg version, §11–§18 verify platforms reference). Đặc biệt: [§12 — RULE token-required KHÔNG login bằng cookie](./add-facebook-reg-version.md#12-rule-quan-trọng--token-required-platform-không-được-login-bằng-cookie), [§13.6 — iOS native FBIOS bắt buộc token EAAAAAY](./add-facebook-reg-version.md#136-ios-native-fbios-verify-ios562-ios563--bắt-buộc-token-eaaaaay), [§13.7 — Login-at-verify + Token/Cookie realtime](./add-facebook-reg-version.md#137-login-at-verify--tokencookie-hiển-thị-realtime-cập-nhật-2026-05-31). Các file 00–06 **link tới runbook** thay vì lặp lại nội dung. |
| [add-ios-reg-version.md](./add-ios-reg-version.md) | **Runbook CHUYÊN iOS Native (FBIOS) — thêm version reg+verify mới.** Quy trình clone `ios562` → `iosNNN` (trích doc_id/bloks/styles/FBAV từ capture, sed swap constants, wiring 8 điểm app.go + factory + scheduler + frontend, build verify). Ví dụ tham chiếu: `ios555` (2026-05-31). |
| [ig-spc-secondary-account-creation.md](./ig-spc-secondary-account-creation.md) | **🆕 Instagram SPC — tạo IG account mới từ IG parent đã login.** Tài liệu kỹ thuật + API contract đầy đủ cho luồng "Add account → Create new" trong Profile Switcher: 4 endpoint Bloks (`get_sso_accounts`, `username.async`, `ac_optin.async`, `create.account.async`), 7 token động cần replace, gotcha "body verbatim", rate-limit awareness, plan port vào engine `internal/instagram/register/igspc/`. Verified working 2026-06-29 (7/8 success rate). Test ref: [`cmd/test_spc_ig/`](../../cmd/test_spc_ig/). |

### 📦 Ghi chú về `docs/old-docs/`

Thư mục [`docs/old-docs/`](../old-docs/) chứa **tài liệu cũ / lịch sử** (REQUIREMENTS, architecture-migration, REGISTER_LOGIC_COMPARISON, RAM_SPLIT_500_500_ANALYSIS, UA_BUILDER_REFACTOR_PLAN, platform-tool-blueprint, forms-ui-reference, BUILD, TODOS…). Chúng hữu ích để hiểu **bối cảnh lịch sử** và quyết định thiết kế, nhưng **KHÔNG đảm bảo khớp 100% với code hiện tại**. Khi cần luồng chạy chính xác, dùng các file 00–06 ở trên (đọc trực tiếp từ code snapshot mới nhất).

---

## 🔄 FLOW TỔNG — kể chuyện 1 lần chạy (end-to-end)

Phần này nối tất cả các file lại bằng một mạch chuyện ngắn. Mỗi đoạn link tới file chi tiết tương ứng.

### Sơ đồ tổng

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ FRONTEND (Vue 3 + TS, chạy trong WebView2)                          [doc 04]   │
│  AccountsPage.vue  ──click "Chạy"──▶  bridge/client.ts (mock | wails)          │
│        ▲                                      │ JS→Go method call               │
│        │ EventsOn (realtime)                  ▼                                 │
└────────┼──────────────────────────────────────────────────────────────────────┘
         │ EventsEmit (Go→JS)            ┌──────────────────────────────────┐
         │                               │ app.go — App (orchestration)     │
         │                               │  RunRegister  |  RunVerify        │
         │                               └───────┬───────────────┬──────────┘
         │                          REGISTER     │               │   VERIFY
         │                            [doc 01]    ▼               ▼  [doc 02/03]
         │                   ┌───────────────────────┐  ┌─────────────────────────┐
         │                   │ spawner→worker pool    │  │ runner.RunVerify (FIFO) │
         │                   │ (inline trong app.go)  │  │ scheduler.go worker pool │
         │                   │  per-platform Register │  │  runOneAccount (retry x2)│
         │                   └───────────┬───────────┘  │  switch verifyPlatform   │
         │                               │              │   → token EAAAAU/EAAAAAY  │
         │      reg OK → đẩy verify       │              │   / cookie login         │
         │   (inline | TRUE SPLIT channel)└─────────────▶  verifybase pipeline 8 b. │
         │                                              └───────────┬─────────────┘
         │                                                          │
         │    ┌─────────────────────────────────────────────────────┘
         │    ▼ callbacks (OnAccountDone, OnSlotAssigned…) → EventsEmit ngược lên UI
         │  ┌──────────────────────────────────────────────────────────────────┐
         │  │ SUBSYSTEMS [doc 06]: Email (OTP) · Proxy (sticky) · 2FA · Upload   │
         │  │ DATA/POOLS [doc 05]: Datr pool · UA builder · fbdata · SimProfile  │
         │  │ RESULT [doc 06]: SuccessReg / SuccessVerify / Die / Unknown (.txt) │
         │  └──────────────────────────────────────────────────────────────────┘
         └──────────────────────────────── grid cập nhật realtime + ghi file kết quả
```

### Diễn giải từng bước

1. **Khởi động** — [`main.go`](../../main.go) gọi `wails.Run`, dựng WebView2 và bind các method của `App`. [`App.startup`](../../app.go) seed Config (datr, fbdata…) và warm pool. Frontend [`bridge/client.ts`](../../frontend/src/bridge/client.ts) tự chọn **mock** (dev browser) hoặc **wails** (binding thật). → Chi tiết: [doc 00](./00-tong-quan-kien-truc.md#3-vòng-đời-ứng-dụng-app-lifecycle).

2. **Người dùng bấm "Chạy"** trên [`AccountsPage.vue`](../../frontend/src/pages/AccountsPage.vue) → qua bridge gọi Go-bound method `RunRegister` hoặc `RunVerify`. Frontend đồng thời `EventsOn` để nghe kết quả realtime. → Chi tiết: [doc 04 §1–§2](./04-frontend-ui-events.md#1-cấu-trúc-frontendsrc).

3. **Nhánh REGISTER** — [`RunRegister`](../../app.go) kiểm tra gating (chống chạy chồng), chọn platform + version (round-robin per-slot), chuẩn bị **run resources** (session pool, email pool, **datr pool** `PartitionedDatrPool`, writer), rồi spawn worker pool. Mỗi worker: lấy proxy sticky → `CheckIP` → build profile + UA → **dispatch theo platform** (Android FB4A Bloks / iOS FBIOS / WebAndroid / s399 legacy REST). → Chi tiết: [doc 01](./01-luong-register.md#0-sơ-đồ-tổng-quan).

4. **Reg thành công** → ghi file `SuccessReg`, thu datr mới, và **đẩy account sang verify**: hoặc **inline** (verify ngay trong cùng worker, non-split), hoặc **TRUE SPLIT** (đẩy qua channel sang một VER pool riêng). → Chi tiết: [doc 01 §11](./01-luong-register.md#11-sau-reg--ghi-file-thu-datr-đẩy-verify) + [doc 03 §8 nhánh 4–5](./03-runner-scheduler.md#8-5-nhánh-dispatch-trong-appgo).

5. **Nhánh VERIFY** — dù đến từ reg hay chạy verify riêng, đều hội tụ về [`runner.RunVerify`](../../internal/runner/scheduler.go) (FIFO worker pool). Mỗi account vào `runOneAccount` (retry 2 tầng). Tại đây `switch verifyPlatform` quyết định **cách lấy auth**: Android-family fetch **token EAAAAU** (`/auth/login`), iOS fetch **token EAAAAAY** (bên trong verifybase), WebAndroid/Web dùng **cookie login**. → Chi tiết: [doc 03 §4](./03-runner-scheduler.md#4-token-type-per-platform--switch-verifyplatform-lõi) + [doc 02 §5](./02-luong-verify.md#5-token-acquisition-theo-platform-rất-quan-trọng).

6. **Pipeline verify** — [`verifybase.RunVerify`](../../internal/facebook/verify/verifybase/run.go) chạy 8 bước chung: Token → Client → Cookie → Email (mua/lấy từ subsystem email, [doc 06 §1](./06-subsystems.md#1-email-subsystem--internalemail)) → FixUA → AddEmail → WaitOTP/Resend → ConfirmCode → **CheckLiveDie** (+ 3 lớp chống false-positive) → 2FA → PostConfirm. Proxy lấy từ subsystem proxy (sticky theo workerID, [doc 06 §2](./06-subsystems.md#2-proxy-subsystem--internalproxy)). → Chi tiết: [doc 02 §6](./02-luong-verify.md#6-verifybaserunverify--pipeline-chung-8-bước).

7. **Kết quả** — mỗi account được phân loại Live/Die/Unknown, ghi file (`SuccessVerify` / `Die` / `Unknown`, qua result writer 2 tầng, [doc 06 §5](./06-subsystems.md#5-result-files--internalresult)) và bắn callback `OnAccountDone` → `EventsEmit` ngược lên UI. → Chi tiết: [doc 02 §9](./02-luong-verify.md#9-kết-quả-verify--file-successverify--die--unknown).

8. **UI cập nhật realtime** — backend gom/throttle event (ticker batch-status), frontend gộp re-render bằng RAF, cập nhật grid in-place (`applySlotAssigned`), đếm success (split mode hiển thị 2 pane REG/VERIFY, dedupe theo UID). → Chi tiết: [doc 04 §3–§5](./04-frontend-ui-events.md#3-bảng-tổng-hợp-tất-cả-event).

9. **Dừng** — bấm Stop là **soft-stop**: cancel `poolCtx` (ngừng nhận account mới) nhưng worker chạy hết account hiện tại qua `workerCtx`; defer cleanup giải phóng proxy/email/writer. → Chi tiết: [doc 03 §7](./03-runner-scheduler.md#7-concurrency--context-cancel-stop) + [doc 06 §7](./06-subsystems.md#7-stop--cleanup).

---

## 🔍 Muốn tìm X → đọc file nào

| Bạn muốn tìm / hiểu… | Đọc file | Mục cụ thể |
|----------------------|----------|------------|
| Bức tranh tổng, tech stack Wails, vòng đời app | [00](./00-tong-quan-kien-truc.md) | §1, §3, §4 |
| Danh sách method `App` Wails-bound | [00](./00-tong-quan-kien-truc.md) | [§8](./00-tong-quan-kien-truc.md#8-danh-sách-method-app-quan-trọng-wails-bound) |
| Danh sách event Go→JS (tổng quát) | [00](./00-tong-quan-kien-truc.md) | [§9](./00-tong-quan-kien-truc.md#9-danh-sách-event-go--js) |
| Bảng event **đầy đủ** + payload + `file:line` Go/Vue | [04](./04-frontend-ui-events.md) | [§3](./04-frontend-ui-events.md#3-bảng-tổng-hợp-tất-cả-event) |
| Plugin registry / factory / cách nối platform | [00](./00-tong-quan-kien-truc.md) | [§6](./00-tong-quan-kien-truc.md#6-plugin-registry--cách-platform-được-nối-vào) |
| Luồng đăng ký 1 account (REGISTER) | [01](./01-luong-register.md) | toàn bộ |
| Vòng đời 1 worker register | [01](./01-luong-register.md) | [§8](./01-luong-register.md#8-vòng-đời-1-worker-reg-1-account) |
| Per-platform reg (Android/iOS/WebAndroid/s399) | [01](./01-luong-register.md) | [§9](./01-luong-register.md#9-per-platform-reg-flow-chi-tiết) |
| Datr / cookie pool (`PartitionedDatrPool`) | [01](./01-luong-register.md) · [05](./05-config-data-pools.md) | [01 §5](./01-luong-register.md#5-datr--cookie-pool--trái-tim-của-reg) · [05 §4](./05-config-data-pools.md#4-datr-pool--partitioneddatrpool-registerandroidpoolgo) |
| EAA token pre-fetch lúc register | [01](./01-luong-register.md) | [§10](./01-luong-register.md#10-eaa-token-pre-fetch-android-family-verify) |
| Luồng verify 1 account (VERIFY) | [02](./02-luong-verify.md) | toàn bộ |
| Token acquisition per-platform (EAAAAU/EAAAAAY/cookie) | [02](./02-luong-verify.md) · [03](./03-runner-scheduler.md) | [02 §5](./02-luong-verify.md#5-token-acquisition-theo-platform-rất-quan-trọng) · [03 §4](./03-runner-scheduler.md#4-token-type-per-platform--switch-verifyplatform-lõi) |
| Pipeline verifybase 8 bước | [02](./02-luong-verify.md) | [§6](./02-luong-verify.md#6-verifybaserunverify--pipeline-chung-8-bước) |
| Live/Die detection + chống false-positive | [02](./02-luong-verify.md) | [§8](./02-luong-verify.md#8-livedie-detection--lớp-chống-false-positive) |
| Worker pool / concurrency / context cancel | [03](./03-runner-scheduler.md) | [§2, §7](./03-runner-scheduler.md#7-concurrency--context-cancel-stop) |
| 5 nhánh dispatch (CloneHV / file / folder / inline / split) | [03](./03-runner-scheduler.md) | [§8](./03-runner-scheduler.md#8-5-nhánh-dispatch-trong-appgo) |
| Split mode (TRUE SPLIT vs non-split) | [01](./01-luong-register.md) · [03](./03-runner-scheduler.md) · [04](./04-frontend-ui-events.md) | [01 §7](./01-luong-register.md#7-slot-allocation--split-mode) · [03 §8](./03-runner-scheduler.md#8-5-nhánh-dispatch-trong-appgo) · [04 §5.3](./04-frontend-ui-events.md#53-split-mode-ui-2-pane-reg--verify) |
| Sticky proxy / KeepIPSuccess | [03](./03-runner-scheduler.md) · [06](./06-subsystems.md) | [03 §7.3](./03-runner-scheduler.md#73-sticky-proxy-theo-workerid-keepipsuccess) · [06 §2.3](./06-subsystems.md#23-sticky-proxy-per-worker--keepipsuccess) |
| Frontend bridge layer (mock ↔ wails) | [04](./04-frontend-ui-events.md) · [00](./00-tong-quan-kien-truc.md) | [04 §1.1](./04-frontend-ui-events.md#11-bridge-layer--cơ-chế-mock--wails) · [00 §1.2](./00-tong-quan-kien-truc.md#12-bridge-layer-frontend-không-import-binding-trực-tiếp) |
| Grid UI cập nhật realtime / dedupe UID | [04](./04-frontend-ui-events.md) | [§4, §5.4](./04-frontend-ui-events.md#54-dedupe-đếm-verify-theo-uid-chống-double-count) |
| UA Builder (3 mode) / pool UA | [05](./05-config-data-pools.md) | [§3](./05-config-data-pools.md#3-ua-builder--internalfacebookfakeinfouabuilder) |
| File config `interaction.json` điều khiển gì | [05](./05-config-data-pools.md) | [§8](./05-config-data-pools.md#8-settings--interactionjson-các-field-điều-khiển-data-layer) |
| Pool versions/builds (FBAV/FBBV) | [05](./05-config-data-pools.md) | [§1](./05-config-data-pools.md#1-internalfbdata--pool-fbavfbbv-versions_and_builds) |
| SimProfile / HNI / Locale / iOS device | [05](./05-config-data-pools.md) | [§5, §6](./05-config-data-pools.md#6-ios-native-data--registerios562devicesgo--ios-ua) |
| Email subsystem (OTP, rent/temp mail) | [06](./06-subsystems.md) | [§1](./06-subsystems.md#1-email-subsystem--internalemail) |
| Proxy subsystem (Manager/Pool/CheckIP) | [06](./06-subsystems.md) | [§2](./06-subsystems.md#2-proxy-subsystem--internalproxy) |
| 2FA (Enable2FA + TOTP) | [06](./06-subsystems.md) | [§3](./06-subsystems.md#3-2fa-subsystem--internalfacebooksecurity) |
| Upload avatar / upload site banclone.pro | [06](./06-subsystems.md) | [§4](./06-subsystems.md#4-upload-subsystem--avatar--site-bancloneparo) |
| File kết quả (.txt) + format + đường dẫn output | [06](./06-subsystems.md) | [§5](./06-subsystems.md#5-result-files--internalresult) |
| Counters / Stats / bảng thống kê UI | [06](./06-subsystems.md) | [§6](./06-subsystems.md#6-counters--stats) |
| Stop / cleanup / soft-stop | [03](./03-runner-scheduler.md) · [06](./06-subsystems.md) | [03 §7](./03-runner-scheduler.md#7-concurrency--context-cancel-stop) · [06 §7](./06-subsystems.md#7-stop--cleanup) |
| **Thêm reg version mới** (thao tác) | [add-facebook-reg-version.md](./add-facebook-reg-version.md) | §1–§10 |
| **Thêm verify version mới** (thao tác) | [add-facebook-reg-version.md](./add-facebook-reg-version.md) | [§16](./add-facebook-reg-version.md#16-thêm-verify-version-mới) |
| Luật token-required KHÔNG login cookie | [add-facebook-reg-version.md](./add-facebook-reg-version.md) | [§12](./add-facebook-reg-version.md#12-rule-quan-trọng--token-required-platform-không-được-login-bằng-cookie) |
| iOS native FBIOS bắt buộc token EAAAAAY | [add-facebook-reg-version.md](./add-facebook-reg-version.md) | [§13.6](./add-facebook-reg-version.md#136-ios-native-fbios-verify-ios562-ios563--bắt-buộc-token-eaaaaay) |
| Login-at-verify + Token/Cookie realtime | [add-facebook-reg-version.md](./add-facebook-reg-version.md) | [§13.7](./add-facebook-reg-version.md#137-login-at-verify--tokencookie-hiển-thị-realtime-cập-nhật-2026-05-31) |
| **🆕 IG SPC — tạo IG mới từ IG live (parent)** | [ig-spc-secondary-account-creation.md](./ig-spc-secondary-account-creation.md) | toàn bộ |
| IG SPC: 4 API endpoint cụ thể | [ig-spc-secondary-account-creation.md](./ig-spc-secondary-account-creation.md) | [§3](./ig-spc-secondary-account-creation.md#3-4-api-endpoint-cụ-thể) |
| IG SPC: 7 token cần replace khi clone body | [ig-spc-secondary-account-creation.md](./ig-spc-secondary-account-creation.md) | [§4](./ig-spc-secondary-account-creation.md#4-token-cần-thay-khi-clone-capture) |
| IG SPC: plan port vào engine HVRIns | [ig-spc-secondary-account-creation.md](./ig-spc-secondary-account-creation.md) | [§6](./ig-spc-secondary-account-creation.md#6-plan-port-vào-engine-hvrins) |

---

## 🗂️ Điểm vào code chính (để tự khảo sát)

| Vai trò | File code |
|---------|-----------|
| Entrypoint + Wails bind | [main.go](../../main.go) |
| Orchestration (App struct, RunRegister, RunVerify, callbacks) | [app.go](../../app.go) |
| Worker pool verify / scheduler | [internal/runner/scheduler.go](../../internal/runner/scheduler.go) |
| Factory đăng ký platform | [internal/facebook/factory.go](../../internal/facebook/factory.go) |
| Types (Session, RegInput/Result, VerifyConfig/Result) | [internal/facebook/types.go](../../internal/facebook/types.go) |
| Pipeline verify chung | [internal/facebook/verify/verifybase/run.go](../../internal/facebook/verify/verifybase/run.go) |
| Datr pool | [internal/facebook/register/android/pool.go](../../internal/facebook/register/android/pool.go) |
| Bridge layer frontend | [frontend/src/bridge/client.ts](../../frontend/src/bridge/client.ts) |
| Trang chính UI | [frontend/src/pages/AccountsPage.vue](../../frontend/src/pages/AccountsPage.vue) |

---

> **Lưu ý:** các số `file:line` trong toàn bộ tài liệu dựa trên snapshot code tại thời điểm viết (2026-05-31). `app.go` rất lớn (~12k dòng) và đang có thay đổi chưa commit, nên dòng có thể lệch nhẹ — luôn `Grep` tên hàm để xác nhận vị trí mới nhất.
