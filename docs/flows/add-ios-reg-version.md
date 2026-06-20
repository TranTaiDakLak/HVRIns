# Runbook — Thêm version iOS Native (FBIOS) Reg + Verify

> Tài liệu **chuyên cho iOS native (FBIOS)** — bổ sung cho [add-facebook-reg-version.md](./add-facebook-reg-version.md) (vốn tập trung Android FB4A). Mô tả **chính xác cách thêm 1 version iOS reg+verify mới** vào app, theo đúng quy trình đã dùng để thêm **`ios555`** (clone từ `ios562`) ngày 2026-05-31.
>
> Khác với Android (mỗi version là 1 thư mục `sNNN` synthetic), iOS native dùng **Bloks CAA pando GraphQL** (`graph.facebook.com/graphql`, `FBAN/FBIOS`, OAuth app-token) — pattern hoàn toàn khác FB4A. Reference chuẩn: **`internal/facebook/register/ios562/`** + **`internal/facebook/verify/ios562/`**.
>
> Đường dẫn code tham chiếu lùi 2 cấp từ `docs/facebook/` → `../../`.

---

## 0. Khi nào dùng tài liệu này

Khi có **capture thật** của 1 version FBIOS mới (vd FBAV `555.x`, `565.x`...) và muốn thêm nút version đó vào cả **Reg** lẫn **Verify** trong UI "Thiết lập chạy".

iOS native hiện có: `ios562` (multi-round, reg+ver), `ios563` (reg single-shot, ver dùng chung verifier 562), `ios555` (clone đầy đủ của 562). Tất cả đăng ký với prefix platform key `iosNNN`.

---

## 1. Nguyên tắc cốt lõi iOS (PHẢI nhớ)

1. **Token verify BẮT BUỘC là `EAAAAAY`** (user token iOS). Token Android `EAAAAU` KHÔNG verify được bằng endpoint Bloks CAA iOS. Reg Android → ver iOS phải login iOS (`send_login_request`) lấy `EAAAAAY` trước. Xem [add-facebook-reg-version.md §13.6/§13.7](./add-facebook-reg-version.md#136-ios-native-fbios-verify-ios562-ios563--bắt-buộc-token-eaaaaay).
2. **Endpoint = `graph.facebook.com/graphql`** (KHÔNG phải `b-graph`). 1 `client_doc_id` dùng chung cho MỌI bloks screen; screen chọn bằng `app_id` + `fb_api_req_friendly_name`, KHÔNG bằng doc_id.
3. **Password = `#PWD_WILDE:2`** (RSA+AES qua `pwd_key_fetch`), fallback `#PWD_FB4A:0` plaintext. (Android dùng `#PWD_FB4A`.)
4. **Cellular bắt buộc**: `X-FB-Connection-Type` = `mobile.CTRadioAccessTechnology*` để header `x-fb-sim-hni` luôn được gửi (xem [`fakeinfo.RandomIOSConnType`](../../internal/facebook/fakeinfo/simnetwork.go)).
5. **Reg flow = multi-round nosess**: round 1 `create.account.async` có thể trả `nosess` (UID + `srnonce` + `fb_partially_created_reg_info`) → round 2..3 gọi lại với partial tokens đến khi full (có `access_token EAAAAAY` + `session_cookies`). `ios562` đã xử lý sẵn loop này.
6. **UA = `FBAN/FBIOS`** với FBAV/FBBV của version. Verify KHÔNG validate FBAV prefix cho iOS (chỉ check chứa `FBAN/FBIOS`) — xem `pickUAForVerifyPlatform`.

---

## 2. Bước 1 — Đọc capture, trích constants version-specific

Capture iOS 1 version thường gồm (vd `DocWeMake/FlowRegFB_IOS/FlowRegVer_IOSVersion/<ver>/`):

| Thư mục capture | Bloks `app_id` (screen) | Ý nghĩa |
|---|---|---|
| `Reg/` | `com.bloks.www.bloks.caa.reg.create.account.async` | Tạo account (có thể 2 file = round 1 + round 2) |
| `ChangeMail/` | `com.bloks.www.bloks.caa.reg.async.contactpoint_email.async` | Add email (verify step 1) |
| `EnterCode/` | `com.bloks.www.bloks.caa.reg.confirmation.async` | Confirm OTP (verify step 3) |

**Trích các constant SAU (giống nhau ở MỌI request của 1 version)** từ body form + `variables`:

| Constant | Lấy từ | Ví dụ (ios555) |
|---|---|---|
| `client_doc_id` | form field `client_doc_id=` | `375801096015005826790906511828` |
| `bloks_versioning_id` | `variables.params.bloks_versioning_id` (= `nt_context.bloks_version`) | `f908dc6a7bcf9f7e18df639bee16e92cdeffc15409d6ac8131c8660d788859eb` |
| `styles_id` | `variables.nt_context.styles_id` | `37f6c230b4d4185e3ef16caa1ad26449` |
| **FBAV** | User-Agent `FBAV/` | `555.0.0.36.63` |
| **FBBV** | User-Agent `FBBV/` | `923840166` |
| OAuth app-token | header `Authorization: OAuth ...` (reg) | `6628568379|c1e620fa708a1d5696fb991c1bde5662` (thường KHÔNG đổi giữa version) |

> So sánh: `ios562`/`ios563` dùng `client_doc_id=375801096013313544589153066091`, bloks `3be43264...`, styles `2d85de7a...`. `ios555` là bộ KHÁC (`375801096015005826790906511828` / `f908dc...` / `37f6c230...`). **Đây chính là lý do 555 cần package verify riêng — không thể dùng chung verifier 562 (vốn nhúng constant 563).**

---

## 3. Bước 2 — Clone package backend (reg + verify)

Reference = `ios562` (đầy đủ multi-round + verify steps). Dùng `cp` + `sed`:

```bash
cd internal/facebook
cp -r register/ios562 register/iosNNN
cp -r verify/ios562   verify/iosNNN

# 1) đổi tên package
sed -i 's/^package ios562/package iosNNN/' register/iosNNN/*.go verify/iosNNN/*.go

# 2) constant Bloks version-specific (562/563 -> NNN)
sed -i 's/<DOCID_562>/<DOCID_NNN>/g; s/<BLOKS_562>/<BLOKS_NNN>/g; s/<STYLES_562>/<STYLES_NNN>/g' register/iosNNN/*.go verify/iosNNN/*.go

# 3) FBAV/FBBV (const fbAppVersion/fbBuildNum + dòng default build trong devices.go)
sed -i 's/563\.0\.0\.67\.72/<FBAV_NNN>/g; s/980285082/<FBBV_NNN>/g' register/iosNNN/*.go verify/iosNNN/*.go

# 4) platform key + nhãn notify
sed -i 's/PlatformIOS562/PlatformIOSNNN/g; s/iOS562/iOSNNN/g' register/iosNNN/*.go verify/iosNNN/*.go

# 5) verify import FetchIOSToken trỏ về register/iosNNN (login dùng constant NNN)
sed -i 's#register/ios562#register/iosNNN#g; s/ios562reg/iosNNNreg/g' verify/iosNNN/*.go
```

**SỬA TAY bắt buộc sau sed:**

- **`verify/iosNNN/verify.go`** — file gốc 562 đăng ký CẢ `PlatformIOS562` LẪN `PlatformIOS563` (563 dùng chung verifier 562). Khi clone phải **XOÁ block đăng ký 563** để chỉ còn `PlatformIOSNNN`:
  ```bash
  sed -i '/iOS563 verify dùng chung/,/RegisterPlatformVerifyUA(facebook.PlatformIOS563, RandomUA)/d' verify/iosNNN/verify.go
  ```
  → `init()` chỉ còn `RegisterPlatformVerifier(PlatformIOSNNN)` + `RegisterPlatformVerifyUA(PlatformIOSNNN, RandomUA)`.

- **`verify/iosNNN/steps.go`** — đảm bảo `Spec` dùng `CheckLiveDieFunc: verifybase.CheckLiveDieCombined` (token-first). **KHÔNG** dùng `CheckLiveDiePictureFirst` vì picture endpoint delay 30-60 phút → báo Live sai cho acc vừa checkpoint:
  ```go
  CheckLiveDieFunc: verifybase.CheckLiveDieCombined,
  ```

- **`register/iosNNN/devices.go` + `verify/iosNNN/devices.go`** — sau khi clone từ 562, `loadFBBuildsFromFile` đã tự đọc đúng file riêng:
  - Reg: `Config/DeviceInfoIOS/ios_app_builds_reg.txt` (fallback → `ios_app_builds.txt`)
  - Ver: `Config/DeviceInfoIOS/ios_app_builds_ver.txt` (fallback → `ios_app_builds.txt`)
  
  Không cần sửa gì thêm — logic fallback đã có sẵn trong template clone.

**File trong mỗi package (clone từ 562):**

| `register/iosNNN/` | Vai trò |
|---|---|
| `profile.go` | **Constants version** (`docIDAction`, `bloksVersioningID`, `stylesID`, `fbAppVersion`, `fbBuildNum`, `oauthToken`) + IOSProfile builder + `buildIOSUA` |
| `register.go` | `init()` `RegisterPlatformRegisterer(PlatformIOSNNN)` + orchestrator `doRegisterAccount` (multi-round nosess loop) |
| `body.go` | `buildCreateAccountBody` (round 1) + `buildCreateAccountRound2` (nosess) — reg_info ~150 field |
| `parse.go` | parse Bloks bytecode → UID/`EAAAAAY`/cookie/partial tokens |
| `http.go` | TLS session (Safari iOS profile) + `postGzip` + headers FBIOS |
| `pwdkey.go` | `pwd_key_fetch` + `#PWD_WILDE:2` encrypt |
| `login.go` | `FetchIOSToken` (CAA `send_login_request` → `EAAAAAY`) — dùng bởi verify |
| `pool.go` | `DevicePool` + `SharedDevicePool` + `SharedDatrPool` (vars) |
| `devices.go` | iPhone device table + FB build pool (load `Config/DeviceInfoIOS/`) |

| `verify/iosNNN/` | Vai trò |
|---|---|
| `verify.go` | `init()` `RegisterPlatformVerifier/VerifyUA(PlatformIOSNNN)` |
| `steps.go` | **Constants verify** (`verifyDocID`, `verifyBloksVer`, `verifyStylesID`, `fbAppVersion`, `fbBuildNum`) + Spec + body builders (AddEmail/Confirm/Resend) + `RandomUA` + `FetchToken`→`iosNNNreg.FetchIOSToken` |
| `devices.go` | device/build/locale pool cho verify UA |

---

## 4. Bước 3 — Khai báo platform constant

[internal/facebook/factory.go](../../internal/facebook/factory.go) — thêm vào block iOS Native:

```go
PlatformIOSNNN = "iosNNN" // iPhone + FB iOS app (FBIOS) API NNN — multi-round nosess handshake, reg+ver (clone 562)
```

---

## 5. Bước 4 — Wiring vào `app.go` (8 điểm)

Tất cả ở [app.go](../../app.go). Dùng `iosNNN`/`PlatformIOSNNN`/`iosNNNreg` song song với `ios562`:

| # | Vị trí (anchor) | Sửa gì |
|---|---|---|
| 1 | import block (~L50) | `iosNNNreg "HVR/internal/facebook/register/iosNNN"` (named — trigger `init()` + dùng pool) |
| 2 | import block (~L108) | `_ "HVR/internal/facebook/verify/iosNNN"` (blank — trigger verifier `init()`) |
| 3 | `verifyIsIOS()` (~L440) | thêm `"iosNNN"` vào `case "ios562", "ios563":` |
| 4 | `verifyPlatformFromType()` (~L697) | thêm `case "iosNNN": return facebook.PlatformIOSNNN` |
| 5 | `allPlatformPools` map (~L7726) | `"iOSNNN": &iosNNNreg.SharedDatrPool,` (datr pool dùng chung) |
| 6 | guard `loadedCookieInitialCount` (~L7882) | thêm `regPlatform != facebook.PlatformIOSNNN &&` (loại iOS khỏi msg CookieInitial Android) |
| 7 | reg pre-gen UA switch (~L8743) | thêm `facebook.PlatformIOSNNN` vào `case facebook.PlatformIOS562, facebook.PlatformIOS563:` (UA build nội bộ, để trống) |
| 8a | datr Register block (~L8924) | thêm block: `if regPlatform == facebook.PlatformIOSNNN && iosNNNreg.SharedDatrPool != nil { Register(prof.SlotIdx); defer Unregister(prof.SlotIdx) }` |
| 8b | `pickUAForVerifyPlatform` (~L11135) | thêm `PlatformIOSNNN` vào `case facebook.PlatformIOS562:` (validator `FBAN/FBIOS`) |
| 8c | `pickUAForVerifyPlatform` (~L11145) | thêm `&& verifyPlatform != facebook.PlatformIOSNNN` vào điều kiện exempt FBAV-prefix |
| 8d | `originalUABaseForPlatform` (~L11439) | thêm case để UA Gốc hoạt động trong UI: `case facebook.PlatformIOSNNN: return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/<FBAV_NNN>;FBBV/<FBBV_NNN>;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"` |

**Device pool (tuỳ chọn, parity với 562)** — sau block iOS562 device pool (~L7909), thêm block tương tự cho iosNNN (`iosNNN_devices.txt` + `iosNNNreg.NewDevicePool(5)` + `SetPersistHook` + `iosNNNreg.SharedDevicePool = ...`). Nếu BỎ QUA, `iosNNNreg.SharedDevicePool` = nil → reg dùng device random mỗi lần (vẫn chạy, chỉ mất tối ưu reuse device tin cậy).

> **KHÔNG cần** đụng `app_reg_sxxx.go` (iOS không qua `isRegPlatformSxxx`; reg dispatch dùng `facebook.NewRegisterer(regPlatform)` qua factory — chỉ cần `init()` chạy nhờ named import #1).

---

## 6. Bước 5 — Scheduler (token acquisition)

[internal/runner/scheduler.go](../../internal/runner/scheduler.go) — thêm `PlatformIOSNNN` vào case iOS (~L688):

```go
case facebook.PlatformIOS562, facebook.PlatformIOS563, facebook.PlatformIOSNNN:
    // scheduler KHÔNG pre-login iOS — verifybase tự login lấy EAAAAAY qua spec.FetchToken
```

> ⚠️ Quên bước này → `default` case scheduler báo `[FATAL]` (safety net). iOS KHÔNG nằm trong Android-family switch.

---

## 7. Bước 6 — Frontend (UI nút version)

[frontend/src/pages/InteractionSetupPage.vue](../../frontend/src/pages/InteractionSetupPage.vue) — 6 điểm:

| Vị trí | Sửa |
|---|---|
| `REG_PLATFORM_LABELS` (~L715) | `iosNNN: 'iOS App (FBIOS NNN)',` |
| `VER_PLATFORM_LABELS` (~L726) | `iosNNN: 'iOS App (FBIOS NNN)',` |
| `REG_PLATFORMS_IOS` (~L812) | `{ key: 'iosNNN', label: 'iOS_NNN' },` |
| `VER_PLATFORMS_IOS` (~L838) | `{ key: 'iosNNN', label: 'iOS_NNN' },` |
| `ORIGINAL_UA_STRINGS` (~L445) | Thêm entry để preview UA Gốc trong UI: `iosNNN: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/<FBAV_NNN>;FBBV/<FBBV_NNN>;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',` |

> **`IOS_PLATFORM_KEY_SET` tự động** — khi đã thêm vào `REG_PLATFORMS_IOS` + `VER_PLATFORMS_IOS`, platform mới tự được nhận diện là iOS trong toàn bộ UI: pool button chỉ hiện iOS, ẩn Virtual spec, màu UA preview xanh, nút File version mở `ios_app_builds_reg/ver.txt`. Không cần sửa thêm chỗ nào khác trong frontend.

> **Mock** ([verify-runner.mock.ts](../../frontend/src/bridge/mock/verify-runner.mock.ts)) KHÔNG có iOS → trả "Mock chưa có UA gốc" cho `iosNNN` (giống ios562, chấp nhận được — UA preview thật chạy qua backend `SimulatePlatformUA` trên Wails).

---

## 8. Bước 7 — Build & verify (BẮT BUỘC)

```bash
gofmt -w internal/facebook/register/iosNNN/ internal/facebook/verify/iosNNN/ app.go internal/facebook/factory.go internal/runner/scheduler.go
go build ./...                                              # PHẢI exit 0
go vet ./internal/facebook/register/iosNNN/... ./internal/facebook/verify/iosNNN/...   # PHẢI exit 0
cd frontend && npm run build                                # PHẢI exit 0
```

Build/vet fail = CHƯA xong. KHÔNG claim hoàn thành.

---

## 9. Checklist

- [ ] Trích đúng `client_doc_id` / `bloks_versioning_id` / `styles_id` / FBAV / FBBV từ capture.
- [ ] `register/iosNNN/` tồn tại, `profile.go` có 5 constant đúng version, `init()` đăng ký `PlatformIOSNNN`.
- [ ] `verify/iosNNN/` tồn tại, `steps.go` có 3 verify constant đúng + `FetchToken`→`iosNNNreg.FetchIOSToken` + `CheckLiveDieFunc: verifybase.CheckLiveDieCombined`; `verify.go` chỉ đăng ký `PlatformIOSNNN` (đã xoá block 563).
- [ ] `factory.go` có `PlatformIOSNNN`.
- [ ] `app.go`: named import + blank import + `verifyIsIOS` + `verifyPlatformFromType` + `allPlatformPools` + guard + reg switch + datr block + `pickUAForVerifyPlatform` (case + exempt) + `originalUABaseForPlatform` (case UA Gốc).
- [ ] `scheduler.go` iOS case có `PlatformIOSNNN`.
- [ ] Frontend: 6 entry (2 label map + REG list + VER list + `ORIGINAL_UA_STRINGS`). `IOS_PLATFORM_KEY_SET` tự xử lý phần còn lại.
- [ ] (Tuỳ chọn) device pool wiring `iosNNN_devices.txt`.
- [ ] `go build ./...` + `go vet` + `npm run build` đều pass.
- [ ] (Sau khi có account + proxy live) test thật reg + verify 1 account → ra `EAAAAAY` + email confirmed.

---

## 10. Ví dụ tham chiếu — ios555 (2026-05-31)

`ios555` được thêm theo đúng runbook này (clone `ios562`):
- Constants: doc_id `375801096015005826790906511828`, bloks `f908dc6a7bcf9f7e18df639bee16e92cdeffc15409d6ac8131c8660d788859eb`, styles `37f6c230b4d4185e3ef16caa1ad26449`, FBAV `555.0.0.36.63`, FBBV `923840166`.
- Reg flow: multi-round nosess (capture `Reg/[138]` thành công trả `EAAAAAY` + cookies sau round-2 với `srnonce` + `fb_partially_created_reg_info`).
- Verify: AddEmail (`contactpoint_email.async`) → Confirm (`confirmation.async`), token `EAAAAAY` bắt buộc.
- Đã build green (`go build ./...` + `go vet` + `npm run build` exit 0). **Chưa test với FB thật** — cần account + proxy live để chốt body khớp 100% (giống ghi chú iOS563).

---

> ⚠️ **Caveat**: như iOS562/563, code build từ capture nhưng **chưa verify với FB thật**. Khi test live, nếu `create.account` reject password → đổi `pwdPrefix` / kiểm tra `pwd_key_fetch`; nếu Bloks báo lỗi schema → đối chiếu lại `client_doc_id`/`bloks_versioning_id` với capture mới nhất.
