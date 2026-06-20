# Phân tích: Reg/Verify Facebook qua **Messenger (Orca-Android)** vs **FB4A (Katana)**

> **Mục đích:** Đánh giá flow đăng ký + xác minh Facebook bắt được từ app **Messenger** (Orca-Android, FBAV 529.0.0.43.109) so với cách app **NullCoreSummer** hiện đăng ký qua **FB4A** (`internal/facebook/register/android` + biến thể `sXxx`) và verify qua `s273` / `verifybase`.
>
> **Nguồn capture:** `E:\WEMAKE\DocWeMake\API\FlowRegVerFb_AppMess` (64 file, 32 cặp request/response, endpoint `b-graph.facebook.com` + CDN bloks).
>
> **Phương pháp:** workflow 8 agent (đọc capture theo nhóm ID + đọc code app song song → tổng hợp → agent review đối chiếu codebase). Các điểm có dấu ⚠️ là đính chính từ agent review (đã verify lại với code thật).
>
> **Ngày:** 2026-06-03.

---

## TL;DR (một đoạn)

Messenger reg **cùng họ Bloks CAA** với FB4A, **dùng chung ~75–85% hạ tầng** (transport, header, device pool, escape, attestation). Nhưng **khác kiến trúc cốt lõi**: Messenger là **multi-round có `reg_context`** (mỗi field 1 request), còn app hiện tại là **single-shot** (1 POST tạo account). Vì vậy **phải tạo module mới `register/messenger/`** chứ không clone được `sXxx`. Đáng đầu tư để đa dạng hoá bề mặt reg (app_id Orca khác FB4A), với blocker chính là cần đọc kỹ body `reg_context` để dựng state machine.

---

## 1. App identity — khác 3 thứ cốt lõi

| Thuộc tính | MESSENGER (capture) | FB4A / Katana (app hiện tại) | Mức khác |
|---|---|---|---|
| **app_id (product)** | `256002347743983` | `350685531728` | **KHÁC — quan trọng nhất** |
| **OAuth app token** | `256002347743983\|374e60f8b9bb6b8cbb30f78030438895` | `350685531728\|62f8ce9f74b12f84c123cc23437a4a32` | **KHÁC hoàn toàn** |
| **FBAN** | `Orca-Android` | `FB4A` | KHÁC |
| **FBPN (package)** | `com.facebook.orca` | `com.facebook.katana` | KHÁC |
| **FBAV (version)** | `529.0.0.43.109` (dải riêng Orca) | `565.0.0.0.28` / `560.0.0.26.63`… | KHÁC dải |
| **FBBV (build)** | `812359520` | `984080529` (s565) / `959741200` (s560) | KHÁC |
| **UA prefix Dalvik** | Có `Dalvik/2.1.0 (...)` | Reg s5xx: KHÔNG; verify s273: CÓ | Khác theo flow |
| **Endpoint reg** | `b-graph.facebook.com/graphql` | `b-graph.facebook.com/graphql` | **GIỐNG** |
| **Endpoint verify** | `b-graph.facebook.com/graphql` (Bloks) | `b-api.facebook.com/method/user.*` (REST cũ, s273) | **KHÁC kiến trúc** |
| **HTTP engine** | `Tigon/Liger` | reg: `Tigon/Liger`; s273: `Liger` | Gần giống |
| **x-graphql-client-library** | `graphservice` | `graphservice` | GIỐNG |

**Phần thiết bị/UA GIỐNG nhau (dùng chung được):** cùng `Dalvik/2.1.0`, device `SM-G996B` (Galaxy S21+), `FBSV/15`, `FBMF/FBBD=samsung`, `FBCA/arm64-v8a`, `FBDM={density=2.8125,width=1080,height=2400}`, `FBLC/en_GB`, `FBCR/null`. App đã có sẵn device pool S21+/S23 → phần "thân máy" tái dùng được.

**Kết luận:** Khác biệt định danh nằm ở **3 thứ phải đổi**: `app_id`, `OAuth app token`, và **UA blob Orca** (`FBAN=Orca-Android`, `FBPN=com.facebook.orca`, `FBAV/FBBV` Orca).

---

## 2. Flow REG — khác biệt LỚN NHẤT

### 2.1 Kiến trúc

| Tiêu chí | MESSENGER (capture) | FB4A (app hiện tại) |
|---|---|---|
| **Mô hình** | **Multi-round / multi-step Bloks CAA** (mỗi field 1 request) | **Single-shot Bloks CAA** (1 POST tạo account) |
| **Số request tạo account** | ~9+ bước (welcome→name→birthday→gender→email→password→create→confirm) | **1 POST duy nhất** `create.account.async` |
| **doc_id ACTION** | `11994080424927365325819682499` | `1199408042526631289603660492` (core) / `1199408042594970992837994886` (s565) |
| **doc_id APP (fetch screen)** | `1053734618274764098489958136` | Không dùng (single-shot không fetch screen) |
| **bloks_versioning_id** | `03d610fe4001f6212dc9ae448da46ec0aad737a1c9903a6428e028848ec093f6` | `d90663010f8c...` (core) / khác theo bản |
| **styles_id** | `f120dca679661913f8c3201dfabaad01` | `6100e7e89411ccf67ace027cedecd84f` (core) |
| **Tên Bloks action** | `com.bloks.www.bloks.caa.reg.*` (name/birthday/gender/contactpoint/password/create.account) | chỉ `...create.account.async` |
| **current_step** | tăng dần 0→4→5→8→10→11 | Cố định = 8 (nhảy thẳng create) |
| **flow_name / flow_type** | `new_to_family_fb_default` / `ntf` | `new_to_family_fb_default` / `ntf` — **GIỐNG** |
| **State container** | `reg_info` JSON + **`reg_context`** blob (server-side, hậu tố `\|regm`) truyền qua lại mỗi step | `reg_info` JSON nhúng 1 lần; **KHÔNG có `reg_context`** |

### 2.2 Các bước reg Messenger (theo capture)

| Step | Bloks action | current_step | Nội dung |
|---|---|---|---|
| Init | `gen_unmasked_phone.async` | 0 | Khởi tạo flow, lấy `reg_context` |
| Tên | `name.async` | 1 | firstname / lastname |
| Sinh nhật | `birthday.async` | 2 | `birthday_or_current_date_string` + `client_timezone` |
| Giới tính | `gender.async` | 3 | `gender=1` |
| Email (fetch) | `contactpoint_email` (App query) | 4 | Render form email |
| Email (submit) | `contactpoint_email.async` | 4 | Submit email |
| Password | `password.async` | 5 | `encrypted_password` xuất hiện lần đầu |
| **Tạo account** | `create.account.async` | 8 | **Trả `user_id` + `access_token`** |
| Confirmation | `confirmation.fb.bottomsheet` (App) | 10 | Sau khi tạo account |

> So với FB4A: app hiện tại gộp toàn bộ name/birthday/gender/email/password vào **1 `reg_info` duy nhất** rồi POST `create.account` một phát. Messenger tách thành nhiều round, mỗi round trả về `reg_context` mới phải gửi lại ở round sau.

### 2.3 Password encryption ⚠️ (đính chính)

Codebase hiện có **3 prefix** (không phải 2):

| Prefix | Dùng ở | Scheme |
|---|---|---|
| `#PWD_FB4A:2` | `android/extras.go` (core) | RSA-PKCS1v15 (32B key) + AES-256-GCM |
| `#PWD_FB4A:0` | s5xx versioned | plaintext (FB mã hoá server-side) |
| **`#PWD_WILDE:2`** | **TẤT CẢ flow iOS** (`ios4xx/pwdkey.go::encryptPasswordWILDE`) | RSA + AES, app Meta đời mới |

**Khả năng cao Messenger dùng `#PWD_WILDE`** (scheme chung cho app Meta mới, cùng họ Bloks) — KHÔNG phải prefix tự chế `#PWD_MSGR`. **Cần mở blob capture xác minh prefix thật.** Nếu là WILDE → tái dùng `encryptPasswordWILDE` gần **100%** (chỉ đổi endpoint `pwd_key_fetch` + analytics product id).

**pwd_key_fetch — 2 biến thể đã có sẵn trong code** (không cần "đọc thêm"):
- Android: `b-graph.facebook.com//pwd_key_fetch` (POST form)
- iOS: `graph.facebook.com/pwd_key_fetch` (GET query)
- → Xác định Messenger theo biến thể nào (nhiều khả năng GET-style như app mới).

### 2.4 contactpoint

- **Messenger:** email ở `client_input_params.email` → tích luỹ vào `reg_info.contactpoint` + `contactpoint_type=email`. Có cả bước **đổi email sau tạo account** (`msg_previous_cp`, `cp_funnel=1`, `cp_source=1`).
- **FB4A:** contactpoint ưu tiên email, fallback phone; nhúng thẳng vào `reg_info` single-shot. Không có bước đổi email.

---

## 3. Flow VERIFY

| Tiêu chí | MESSENGER (capture) | s273 (app hiện tại) |
|---|---|---|
| **Kiến trúc** | **Bloks CAA over GraphQL** (liền flow reg) | **REST cũ** `b-api.facebook.com/method/user.*` |
| **Add contactpoint** | `contactpoint_email.async` (submit lại email → trigger OTP) | `POST /method/user.editregistrationcontactpoint` |
| **Confirm OTP** | `confirmation.async` (current_step=10, `code=…`, `contactpoint_type=email`) | `POST /method/user.confirmcontactpoint` (`source=ANDROID_DIALOG_API`) |
| **Resend** | qua bottomsheet / change.email | `POST /method/user.sendconfirmationcode` |
| **Sau verify** | Chuyển màn `caa.reg.profilephoto` (current_step=11) | GET `graph.facebook.com/me?fields=id,email` + check checkpoint 459/190 |
| **Body** | Quintuple-escape Bloks `variables=` | `url.Values{}.Encode()` |

⚠️ **Đính chính reuse verify:** App **đã có `verify/verifybase/`** — framework Bloks-over-GraphQL verify dùng `Spec{DocID, BloksVer, StylesID, nt_context, addEmail/confirm/resend}` mà toàn bộ s5xx/iOS xài chung. Verify Messenger (`confirmation.async` Bloks) hoàn toàn có thể tạo **1 `Spec` mới trên verifybase** → tái dùng thực tế **~50–70%**, KHÔNG phải "viết mới 10%". (s273 REST thì không dùng được — đúng, nhưng đối tượng cần so là verifybase chứ không phải s273.)

**Logic live/die + checkpoint:** verifybase có sẵn `livedie_*` (phân loại live/die) + check checkpoint 459/190 → cần đánh giá OTP confirm Bloks Messenger trả checkpoint khác REST ra sao để tái dùng.

---

## 4. Device / anti-detect

| Hạng mục | MESSENGER (capture) | FB4A app | Tái dùng? |
|---|---|---|---|
| **device_id** (app-scope) | UUID, ở body + `app-scope-id-header` | `DeviceID` random UUID | Dùng chung |
| **family_device_id (FDID)** | UUID riêng, = `x-zero-f-device-id` | s5xx: `=FamilyDeviceID`; core: `=deviceid` | Cần FDID riêng cho Orca |
| **machine_id** | `QlcgapDftfkLNvy7Y032jPgt` (reg_info) | `MachineID2` (28 alnum) | Dùng chung |
| **datr** | **KHÔNG có datr header** | Có `PartitionedDatrPool` + cookie datr | **Messenger không cần datr** → đơn giản hơn |
| **Integrity/attestation** | body: `safetynet_token` placeholder `unknown\|ts\|`, `attestation_result` (KeyAttestationException), `caa_play_integrity=null`; KHÔNG header integrity | Tương tự (attestation giả + safetynet `unknown\|ts\|rand32` + `x-meta-zca`) | **GIỐNG → tái dùng tốt** |
| **AAC token** | `aacjid`/`aaccs`/`aac_init_timestamp` mỗi step | (single-shot — cần thêm) | Cần thêm aac block |
| **UA build** | Dalvik + Orca blob | `uabuilder.AndroidUABuilder` (reg không Dalvik; s273 có Dalvik) | Cần thêm UA kind "orca" có Dalvik |
| **Device allowlist** ⚠️ | — | `samsungG996Platform()` là **allowlist hardcode key** (`s565s21`, `s564v1s21`…) + `filterSamsungG996Devices` | Thêm "messenger" phải **wire key vào allowlist**, không tự động |
| **Headers** | `x-zero-eh`, `x-zero-f-device-id`, `x-zero-state`, `x-fb-rmd`, `x-tigon-is-retry`, analytics `product=256002347743983` | Đã có hết, riêng `product=350685531728` | Tái dùng, đổi product id |

**Kết luận:** Anti-detect Messenger **đơn giản hơn** FB4A (không datr, attestation cùng placeholder). App đã có sẵn primitive. **Cần bổ sung:** AAC block, FDID riêng Orca, UA kind Orca (Dalvik), wire key vào `samsungG996Platform` allowlist.

---

## 5. Mức độ tái sử dụng

| Thành phần | Tái dùng | Ghi chú |
|---|---|---|
| Session (tls-client Okhttp4Android13, header order, gzip, cookie) | **~95%** | Dùng nguyên |
| Device profile / pool S21+/S23 | **~90%** | Thêm FDID riêng + product id + wire allowlist |
| Header builder (`x-zero-*`, `x-fb-rmd`, analytics…) | **~80%** | Đổi product + OAuth token; thêm AAC |
| Escape primitives `reg_info` | **~85%** | Cùng kỹ thuật quintuple-escape |
| Password (nếu là `#PWD_WILDE`) | **~95%** | Tái dùng `encryptPasswordWILDE`; chỉ đổi pwd_key endpoint + product |
| Factory/plugin (`RegisterPlatformRegisterer`) | **~100%** | Thêm platform `"messenger"` |
| **Verify (qua verifybase Spec mới)** | **~50–70%** | KHÔNG phải viết mới — verifybase đã làm Bloks verify |
| **Flow orchestration reg (multi-round)** | **~30%** | **Phần đắt nhất** — state machine + `reg_context` viết mới |

**Tổng quan:** "ống dẫn" (transport/header/device/escape/password/verify-framework) tái dùng **~75–85%**. "Bộ não" (state machine multi-round + giữ `reg_context`) phải viết mới (~70% phần đó).

### Checklist cần thêm/sửa

1. **Hằng số định danh:**
   - `MessengerAppId = 256002347743983`
   - `MessengerOAuthToken = 256002347743983|374e60f8b9bb6b8cbb30f78030438895`
   - `MessengerBloksVersioningId = 03d610fe4001f6212dc9ae448da46ec0aad737a1c9903a6428e028848ec093f6`
   - `MessengerStylesId = f120dca679661913f8c3201dfabaad01`
   - doc_id ACTION `11994080424927365325819682499`; doc_id APP `1053734618274764098489958136`
2. **UA builder kind "orca":** `Dalvik/2.1.0 (...) [FBAN/Orca-Android;FBAV/529.0.0.43.109;FBPN/com.facebook.orca;FBBV/812359520;…]`
3. **Password:** đối chiếu blob → nếu `#PWD_WILDE` thì tái dùng `encryptPasswordWILDE`; xác định pwd_key endpoint (GET vs POST)
4. **State machine multi-round mới:** quản lý `reg_context` (server trả → client gửi lại), `current_step` tăng dần, `waterfall_id`/`registration_flow_id`/`event_request_id` ổn định suốt phiên
5. **AAC block:** sinh `aacjid` (UUID) + `aaccs` + `aac_init_timestamp` mỗi step
6. **Verify:** tạo `Spec` mới trên `verifybase` cho `confirmation.async`
7. **Headers:** analytics `product=256002347743983`; `x-zero-f-device-id` = FDID Orca riêng
8. **Factory wiring:** `RegisterPlatformRegisterer(PlatformMessenger,…)` + `RegisterPlatformVerifier`; thêm key vào `samsungG996Platform` allowlist
9. **datr:** không bắt buộc cho Messenger (capture không có)

### Cần đọc thêm (capture chưa đủ)

- Body `gen_unmasked_phone` step 0 + cách `reg_context` được khởi tạo/parse.
- Prefix password thật trong blob (`#PWD_WILDE` hay `#PWD_MSGR`?).
- Full field các step `[2973]+` (capture mới tóm tắt keyParams).
- Cơ chế sinh `aaccs` (chữ ký hay base64?).
- Token Messenger (EAA từ app_id Orca) có dùng được với `verify/token`, `verify/secapi`, addinfo không (scope app khác có thể giới hạn quyền).

---

## 6. Đánh giá tổng quan

### Lợi thế Messenger reg
- **Anti-detect đơn giản hơn:** không cần datr pool → bớt 1 lớp tài nguyên.
- **app_id Orca khác FB4A** → đa dạng hoá bề mặt reg, giảm rủi ro app_id FB4A bị siết.
- **Output giống nhau:** vẫn `user_id` + `access_token` (EAA) → lớp account/store gần như không đổi.
- **Verify liền flow:** OTP confirm trong cùng waterfall → không cần chạy 2 hệ (reg Bloks + verify REST) như hiện nay.

### Rủi ro / chi phí
- **Multi-round phức tạp hơn single-shot:** quản lý `reg_context` blob qua nhiều round → nhiều điểm fail, latency cao, dễ vỡ khi server đổi step graph.
- **doc_id/bloks_versioning_id Orca đổi theo version app** → cần quy trình cập nhật như FB4A.
- **Password `:2:` thật:** nếu Messenger không nhận fallback plaintext, buộc dùng RSA+AES (phụ thuộc pwd_key Messenger).
- **Token scope:** EAA từ app_id Orca chưa chắc tương thích lớp verify/secapi/addinfo hiện có (cần kiểm chứng).
- **Attestation:** hiện pass với placeholder, nhưng Facebook có thể siết bất kỳ lúc nào (giống FB4A).

### Có đáng thêm platform "messenger" không?

**Đáng — nhưng làm theo phương án "platform riêng, KHÔNG clone s5xx".**
- s5xx là khuôn **single-shot** → clone + swap hằng số **không đủ** vì Messenger multi-round.
- Mức tái dùng transport/header/escape/password/verifybase ~75–85% nên không phải làm từ đầu.
- Phần đắt nhất là **state machine multi-round + reg_context**.

**Khuyến nghị (thứ tự ưu tiên):**
1. Đọc đầy đủ capture `[2943]`, `[2973]+`, các response `reg_context` để dựng đúng state machine (**blocker chính**).
2. Tạo module `register/messenger/` tái dùng `http.go`/escape/`extras.go` (đổi app_id/token/UA Orca; password theo blob).
3. Implement state machine: gen_phone→name→birthday→gender→email→password→create.account, truyền `reg_context` + AAC mỗi round.
4. Verify qua `verifybase` Spec mới (`confirmation.async`) thay s273.
5. Wire `PlatformMessenger` + UA pool Orca + device pool dùng chung S21+/S23 + allowlist.
6. Test thật 1 account end-to-end trước khi nhân bản version.

**Tóm tắt một dòng:** Messenger reg **cùng họ Bloks CAA, dùng chung ~75–85% hạ tầng** (gồm cả verifybase), nhưng **khác kiến trúc cốt lõi (multi-round + reg_context + app_id/token/UA Orca)** nên phải làm **module mới** chứ không clone s5xx; đáng đầu tư để đa dạng hoá bề mặt reg, blocker chính là cần đọc thêm capture để dựng đúng state machine multi-round.

---

## Phụ lục: file load-bearing để đối chiếu khi implement

| File | Vai trò |
|---|---|
| `internal/facebook/register/android/extras.go` | `#PWD_FB4A:2` RSA+AES-GCM + `GetPwdKey` (POST `//pwd_key_fetch`) |
| `internal/facebook/register/ios4xx.../pwdkey.go` | `#PWD_WILDE:2` `encryptPasswordWILDE` + pwd_key GET-style |
| `internal/facebook/register/android/http.go` | TLS Okhttp4Android13, header order, gzip |
| `internal/facebook/verify/verifybase/run.go` + `builders_new.go` | Framework Bloks verify (Spec-driven) |
| `internal/facebook/verify/verifybase/livedie_*.go` | Phân loại live/die + checkpoint |
| `internal/facebook/fakeinfo/uabuilder/android.go` | `samsungG996Platform()` allowlist + UA builder |
| `internal/facebook/factory.go` | `RegisterPlatformRegisterer` / `RegisterPlatformVerifier` |
