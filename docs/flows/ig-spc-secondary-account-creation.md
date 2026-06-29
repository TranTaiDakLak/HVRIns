# Instagram SPC — Secondary Profile Creation từ IG account đã login

> **TL;DR:** Một IG account đã login (parent) có thể tạo ra IG account mới (con) thông qua luồng SPC (Secondary Profile Creation) — đăng ký "Add account" từ Profile Switcher. **KHÔNG cần** Play Integrity, DroidGuard, hay Keystore attestation. Đã verify thực tế: 7/8 lần tạo thành công trong 2 lần test.
>
> **Reference capture:** `E:\WEMAKE\DocWeMake\RegIGinIG\V3\` (75 request/response, mitmproxy 2026-06-29)
>
> **Reference impl:** [`cmd/test_spc_ig/`](../../cmd/test_spc_ig/)

---

## 1. Bản chất luồng SPC

SPC = **Secondary Profile Creation**. Khi user bấm "Add account → Create new account" trong Profile Switcher của Instagram Android, app gọi 6 bước Bloks qua `i.instagram.com/graphql_www` để tạo account mới **dưới ngữ cảnh authenticated của parent**.

```
┌─ Parent IG (login, có Bearer IGT:2:...) ─┐
│                                            │
│  GET /fxcal/get_sso_accounts/ ─────────────┼─► Verify parent token live
│                                            │
│  POST /graphql_www                         │
│    bloks.caa.reg.username.async ───────────┼─► Submit email + bday + username
│                                            │
│  POST /graphql_www                         │
│    bloks.caa.reg.ac_optin.async ───────────┼─► Accept ToS / Account Center
│                                            │
│  POST /graphql_www                         │
│    bloks.caa.reg.create.account.async ─────┼─► CREATE ACCOUNT
│                                            │      → pk (uid mới)
│                                            │      → sessionid mới
│                                            │      → IG-Set-Password-Encryption-Pub-Key
│                                            │      → partially_created_account_nonce
│                                            │
└────────────────────────────────────────────┘
                                              │
                                              ▼
                                          IG con (uid mới)
                                          xuất hiện trong
                                          Account Switcher của parent
```

**Khác với engine `internal/igcore` hiện tại:**

| Tiêu chí | `igcore` (anonymous reg) | SPC (authenticated reg) |
|---|---|---|
| Auth context | Logged-out (chỉ x-mid) | **Bearer IGT:2: của parent** |
| Endpoint | `i.instagram.com` REST steps + `/graphql_www` | Tất cả `/graphql_www` Bloks |
| Password | `#PWD_INSTAGRAM:4` RSA+AES-GCM trong create | **KHÔNG truyền password** |
| OTP email | Trước khi create | **Sau** create (`/api/v1/challenge/`) |
| Output | uid + cookie + sessionid + password đã set | uid + sessionid + Pub-Key (chưa set password) |
| Phụ thuộc | Standalone (logged-out) | **Cần parent IG còn live** |

---

## 2. Constraint quan trọng

### 2.1 Parent IG bắt buộc còn live

Parent phải có Bearer token còn hiệu lực. Verify bằng Step A (`get_sso_accounts`) — nếu trả 200 + JSON list account → token live. Nếu trả 4xx → parent die, không thể chạy SPC.

### 2.2 Rate limit phía parent

1 parent tạo nhiều con liên tiếp sẽ bị server reject (response F trả Bloks UI thay vì `account_created=true`). Khuyến nghị:
- **Cooldown ≥ 30s** giữa 2 lần tạo từ cùng parent
- **Tối đa 5–10 con/parent/ngày** rồi xoay parent khác
- Spam → parent bị checkpoint → **mất luôn tài sản gốc**

### 2.3 KHÔNG cần Google attestation

Capture V3 ban đầu chứa 2 call Google:
- `www.googleapis.com/androidantiabuse/v1/x/create` — DroidGuard/SafetyNet
- `play-fe.googleapis.com/fdfe/integrity` — Play Integrity

Và header `x-ig-attest-params` chứa Play Integrity JWT + Keystore signed nonce.

**Thực tế test:** bỏ hoàn toàn cả 2 call Google + bỏ header → server vẫn cho `account_created=true`. Khả năng IG có server-side fallback cho client không có GMS, hoặc anti-abuse check khác (waterfall + parent reputation + body shape) đã đủ pass.

### 2.4 Body PHẢI clone verbatim từ capture

Đây là gotcha lớn nhất. Body Bloks create có ~400 field (`failed_birthday_year_count`, `caa_play_integrity_attestation_result`, `cached_headers_safetynet_info`, ...) hầu hết là `null`. Nếu build body minimal (chỉ ~30 field cần), server **vẫn trả 200** nhưng response chứa `create_failure : system_error` + Bloks UI báo lỗi generic.

→ Phải clone nguyên capture body, chỉ replace các token động.

---

## 3. 4 API endpoint cụ thể

### 3.1 Step A — `POST /api/v1/fxcal/get_sso_accounts/`

**Mục đích:** Verify parent token còn live + lấy thông tin account để hiển thị trong Profile Switcher.

**Request:**
```http
POST /api/v1/fxcal/get_sso_accounts/ HTTP/2
Host: i.instagram.com
Authorization: Bearer IGT:2:<parent_bearer_b64>
ig-intended-user-id: <parent_uid>
ig-u-ds-user-id: <parent_uid>
x-ig-app-id: 567067343352427
x-ig-device-id: <qe_device_id>
x-ig-family-device-id: <family_device_id>
x-ig-android-id: <device_id>
x-mid: <x_mid>
x-bloks-version-id: 2530c58174d063584f25e249151d5bc7c53db138cfc68b554daa78c6cd7356b0
user-agent: Instagram 421.0.0.51.66 Android (35/15; ...; samsung; SM-G996B; t2s; exynos2100; en_GB; ...)
content-type: application/x-www-form-urlencoded; charset=UTF-8
x-fb-friendly-name: IgApi: fxcal/get_sso_accounts/
accept-encoding: zstd

signed_body=SIGNATURE.<URL-encoded JSON>
```

Trong đó `<URL-encoded JSON>` là:
```json
{
  "surface": "account_switcher",
  "tokens": "[{\"account_type\":\"Instagram\",\"token_id\":0,\"token_str\":\"Bearer IGT:2:...\",\"user_fbid\":\"<parent_uid>\",\"token_type\":\"first_party\",\"token_app\":\"Instagram\",\"token_source\":\"active_account\"}]",
  "_uid": "<parent_uid>",
  "device_id": "<device_id>",
  "_uuid": "<qe_device_id>",
  "include_social_context": "false"
}
```

**Response 200 OK** — JSON chứa danh sách SSO account của parent. Quan trọng: chỉ cần status 200 là PASS.

**Failure mode:** 401/403 → parent token đã rotate/checkpoint → skip parent này.

---

### 3.2 Step D — `POST /graphql_www bloks.caa.reg.username.async`

**Mục đích:** Submit email + birthday + username dự kiến lên server, server kiểm tra availability và trả Bloks UI cho màn Account Center.

**Request:**
```http
POST /graphql_www HTTP/2
Host: i.instagram.com
content-type: application/x-www-form-urlencoded
x-fb-friendly-name: IGBloksAppRootQuery-com.bloks.www.bloks.caa.reg.username.async
x-client-doc-id: 356548512614739681018024088968
x-root-field-name: bloks_action
x-graphql-client-library: pando
x-ig-app-id: 567067343352427
x-ig-device-id: <qe_device_id>
x-ig-family-device-id: <family_device_id>
x-ig-android-id: <device_id>
x-mid: <x_mid>
x-bloks-version-id: 2530c58174d063584f25e249151d5bc7c53db138cfc68b554daa78c6cd7356b0
user-agent: Instagram 421.0.0.51.66 Android (...)
accept-encoding: zstd

method=post&pretty=false&format=json&server_timestamps=true&locale=user&purpose=fetch
&fb_api_req_friendly_name=IGBloksAppRootQuery-com.bloks.www.bloks.caa.reg.username.async
&client_doc_id=356548512614739681018024088968
&enable_canonical_naming=true
&enable_canonical_variable_overrides=true
&enable_canonical_naming_ambiguous_type_prefixing=true
&variables=<JSON ~23KB chứa params>
```

**Variables payload (cấu trúc 4 tầng):**
```
variables = {
  "params": {
    "params": "<STRING — JSON.stringify của reg payload>",
    "bloks_versioning_id": "2530c58174...",
    "infra_params": {"device_id": "<qe_device_id>"},
    "app_id": "com.bloks.www.bloks.caa.reg.username.async"
  },
  "bk_context": {
    "is_flipper_enabled": false,
    "theme_params": [{"value": ["three_neutral_gray"], "design_system_name": "XMDS"}],
    "debug_tooling_metadata_token": null
  }
}
```

**Reg payload (chuỗi JSON trong `params.params`):**
```
{
  "client_input_params": { ... ~30 field, hầu hết empty string hoặc null ... },
  "server_params": {
    "event_request_id": "<random UUID>",
    "device_id": "<device_id>",
    "waterfall_id": "<random UUID>",
    "flow_info": "{\"flow_type\":\"spc\",\"flow_name\":\"secondary_profile_creation_ig_default\"}",
    "bloks_controller_source": "bk_caa_reg_icon_text_list_aymh_screen",
    "reg_info": "<STRING — JSON.stringify của ~200 field>",
    "family_device_id": "<family_device_id>",
    "offline_experiment_group": "caa_iteration_v3_perf_ig_4",
    "access_flow_version": "pre_mt_behavior",
    "current_step": 0,
    "qe_device_id": "<qe_device_id>"
  }
}
```

**reg_info schema (~200 field, hầu hết `null`, các field cần điền):**

| Field | Giá trị | Nguồn |
|---|---|---|
| `device_id` | `android-0619f0ab0c5dba42` | bake từ capture (hoặc randomize) |
| `family_device_id` | `e8600531-590b-45f8-a617-...` | bake (hoặc random UUID) |
| `ig4a_qe_device_id` | `1f3b7429-d663-442a-9dba-...` | bake (hoặc random UUID) |
| `soap_creation_source` | `"profile_switcher"` | constant |
| `full_sheet_flow` | `true` | constant |
| `birthday` | `1992-02-28` | có thể fixed hoặc randomize range 18+ |
| `contactpoint` | `<email mới>` | từ email provider |
| `contactpoint_type` | `"email"` | constant |
| `username` | `<username dự kiến>` | random |
| `username_prefill` | giống `username` | constant |
| `ig_authorization_token` | `Bearer IGT:2:<parent_bearer_b64>` | từ parent |
| `machine_id` | `pDdCanKwmR1Nf95dtYOHRtpl` | bake (khác `x-mid` header) |
| `source_username` | `<parent.username>` | từ parent |
| `source_account_reg_info.source_credentials` | `Bearer IGT:2:<parent_bearer_b64>` | từ parent |
| `source_account_reg_info.source_account_type` | `1` | constant (IG account) |
| `source_account_reg_info.source_credentials_type` | `"access_token"` | constant |
| `screen_visited` | `["CAA_REG_IG_USERNAME"]` | constant |
| `should_show_error_msg` | `true` | constant |
| `existing_accounts` | `[{...thông tin parent...}]` | từ parent (optional, không bắt buộc) |

Còn lại tất cả là `null`.

**Response 200 OK:** Bloks UI bundle (~155KB) chứa layout màn hình Account Center Opt-in.

---

### 3.3 Step E — `POST /graphql_www bloks.caa.reg.ac_optin.async`

**Mục đích:** Hiển thị màn ToS / Account Center opt-in cho user. User bấm "Đồng ý" → app chuẩn bị create.

**Khác biệt body so với Step D:**
- `app_id` = `com.bloks.www.bloks.caa.reg.ac_optin.async`
- `current_step` = `2`
- `bloks_controller_source` = `"bk_caa_reg_icon_text_list_username_screen"`
- Còn lại giữ nguyên reg_info từ Step D

**Header đặc biệt:** Capture V3 gắn `x-ig-attest-params` ở đây. **Thực tế bỏ header này, request vẫn PASS.**

**Response 200 OK:** Bloks UI bundle (~94KB) chứa layout màn ToS.

---

### 3.4 Step F — `POST /graphql_www bloks.caa.reg.create.account.async` ⭐

**Mục đích:** **TẠO ACCOUNT THỰC SỰ.** Server tạo user mới và trả về uid + sessionid + RSA pub key.

**Endpoint nội bộ phía server:** `/api/v1/multiple_accounts/create_secondary_account/` (Bloks GraphQL alias).

**Khác biệt body so với Step E:**
- `app_id` = `com.bloks.www.bloks.caa.reg.create.account.async`
- `current_step` = `4`
- `bloks_controller_source` = `"bk_caa_reg_icon_text_list_tos_screen"`
- `reached_from_tos_screen` = `1` (flag chứng minh user đã qua ToS)
- `encrypted_password` = `null` (**KHÔNG truyền password**)
- Còn lại giữ nguyên

**Response 200 OK** — chứa Bloks bundle, trong đó parse được:

```json
{
  "account_created": true,
  "pk": 48528135904,
  "pk_id": "48528135904",
  "fbid_v2": 17841448717126016,
  "username": "Instagram user",
  "status": "ok",
  "multiple_users_on_device": true,
  "sessionid": "48528135904%3AzrmwyAddUFdCac%3A1%3AAYimuIjyP5ais8XRLgegcTrNvQXW5OXxL9JlPpVQg",
  "partially_created_account_nonce": "9VO82uCuDR65bWLMr0HyKinmXKb39KhoXkw5qgE2FyZMyS88SfFCVoYzyUi6BfZC",
  "partially_created_account_nonce_expiry": 1782725139
}
```

**Header response (set vào client):**
- `IG-Set-Authorization: Bearer IGT:2:<new_bearer_b64>`
- `IG-Set-Password-Encryption-Key-Id: 60`
- `IG-Set-Password-Encryption-Pub-Key: <RSA pub key base64>` — để set password sau
- `IG-Set-X-MID: <new x_mid>`
- `ig-set-ig-u-ds-user-id: <new_uid>`

**Failure mode:** Response 200 nhưng KHÔNG có `account_created=true` → Bloks UI báo lỗi generic ("We're sorry, but something went wrong."). Nguyên nhân thường gặp:
- Body thiếu field (build minimal thay vì verbatim)
- Rate limit từ parent (vừa tạo cách đó < 30s)
- Parent reputation thấp (account cũ, bị flag)
- IP từ proxy bị flag

---

## 4. Token cần thay khi clone capture

Khi clone body verbatim từ capture, chỉ cần thay **7 token động** (tất cả là string replace, không cần re-build JSON):

| Token capture | Replace bằng | Vì sao |
|---|---|---|
| `frost.ninja7158@17a.imgui.de` | Email mới từ provider | Email unique per account |
| `platypus.1114188` | Username random | Tránh collision |
| `1992-02-28` | Giữ nguyên, hoặc randomize | OK để fixed |
| `kenneth_roberts2001` | `<parent.username>` | Parent thay đổi |
| `Bearer IGT:2:<old_b64>` | `<parent.bearer>` | Parent token thay đổi (URL-encode khi cần) |
| `f82bedd4-d5fd-45c8-be11-5e3e4e679f42` | Random UUID mới | Mỗi session 1 waterfall |
| `0bc947a8-9075-445f-b829-a4394a3815a3` | Random UUID mới | event_request_id |

**KHÔNG cần thay** (capture value vẫn work):
- `device_id` (`android-0619f0ab0c5dba42`)
- `family_device_id` (`e8600531-590b-45f8-a617-0aa9146520bd`)
- `qe_device_id` (`1f3b7429-d663-442a-9dba-463b15e23384`)
- `machine_id` (`pDdCanKwmR1Nf95dtYOHRtpl` và `aj45lQABAAFnR8kPOXtsuhl75Xlb`)
- `bloks_versioning_id`
- `client_doc_id`

**KHUYẾN NGHỊ:** Khi port vào engine HVRIns, **randomize cả `device_id`/`family_device_id`/`qe_device_id`** để tránh dùng chung 1 device cho hàng ngàn account → IG flag bulk-creation. Mỗi run tạo 1 device fingerprint mới (UUID).

---

## 5. Implementation reference

**File test working:** [`cmd/test_spc_ig/main.go`](../../cmd/test_spc_ig/main.go)

**3 file capture embed:**
- `cmd/test_spc_ig/capture_3586_username.txt` (33.8 KB) — body Step D
- `cmd/test_spc_ig/capture_3592_ac_optin.txt` (33.1 KB) — body Step E
- `cmd/test_spc_ig/capture_3597_create.txt` (34.4 KB) — body Step F

**Test result đã verify (2026-06-29):**
- Run 1: 2/4 success (1 thread)
- Run 2: 3/4 success (2 thread, 1 fail vì rate-limit)
- Account mới tạo được có `pk` valid + `sessionid` valid + `Bearer IGT:2:` mới

**Lệnh chạy:**
```powershell
cd E:\WEMAKE\HVRIns
go run .\cmd\test_spc_ig -no-attest `
  -proxy="unlimited.iprocket.io:12000:USERt1mbtV-zone-custom:Phatloc888" `
  -threads=2 `
  -dump=.\debug_spc
```

Flag `-no-attest` bỏ Step B + Step C + header `x-ig-attest-params`. Đây là default cho engine production.

---

## 6. Plan port vào engine HVRIns

### 6.1 Package mới

Tạo `internal/instagram/register/igspc/` với cấu trúc:
```
internal/instagram/register/igspc/
├── igspc.go            # Registerer implementation + factory init()
├── steps.go            # 4 step (A, D, E, F)
├── body_builder.go     # Token replacement logic
├── session.go          # TLS session wrapper (reuse igcore profile)
├── parent_pool.go      # Quản lý parent IG pool (rotate + cooldown)
└── templates/
    ├── username.txt    # Body Step D (go:embed)
    ├── ac_optin.txt    # Body Step E (go:embed)
    └── create.txt      # Body Step F (go:embed)
```

### 6.2 Đăng ký platform

Trong `internal/instagram/factory.go`:
```go
const PlatformIGSPC = "ig_spc"
```

Trong `igspc/igspc.go`:
```go
func init() {
    instagram.RegisterPlatformRegisterer(instagram.PlatformIGSPC, func() instagram.Registerer {
        return &igSPCRegisterer{}
    })
}
```

Trong `internal/app/app.go` (hoặc nơi blank-import):
```go
_ "HVRIns/internal/instagram/register/igspc"
```

### 6.3 Input contract

Cần extend `RegInput` (hoặc tạo struct mới) để pass parent:
```go
type SPCRegInput struct {
    *instagram.RegInput          // chứa Email, Proxy, GetOTP, ...
    ParentBearer    string       // "Bearer IGT:2:..."
    ParentUsername  string
    ParentUID       string
}
```

Hoặc thêm field vào `RegInput.Extra map[string]any` để không phá interface hiện tại.

### 6.4 Parent pool

Tạo `internal/instagram/register/igspc/parent_pool.go` với:
- Load parent list từ file (format giống Live.txt: `username|password|cookie|bearer|date|`)
- Mỗi parent có `lastUsedAt time.Time` + `usageCountToday int`
- `Acquire()` chọn parent có cooldown đủ (≥ 30s) và usage_count < 10/day
- `Release(success bool)`:
  - Success → tăng count, lưu lại
  - Fail vì rate-limit → đặt cooldown 5 phút
  - Fail vì checkpoint → mark parent dead, remove khỏi pool

### 6.5 Output

Engine trả về `RegResult` chuẩn:
```go
result.UID = "48528135904"
result.Username = "Instagram user"  // chưa set username thật
result.Cookie = "<reconstructed từ sessionid + x_mid + ig_did + ds_user_id>"
result.AccessToken = "<new bearer IGT:2:...>"  // từ header IG-Set-Authorization
result.Extra = map[string]any{
    "spc_parent_uid":           parent.UID,
    "password_encryption_pub_key": pubKey,
    "password_encryption_key_id":  "60",
    "partially_created_nonce":     nonce,
    "needs_email_verify":          true,
}
```

### 6.6 Email verify (chưa implement)

Sau Step F, response chứa challenge_context. Cần gọi tiếp:
- `GET /api/v1/challenge/?challenge_context=crpd_...`
- `POST /api/v1/bloks/apps/com.bloks.www.checkpoint.ufac.controller/`
- Submit OTP từ email → verify done

Có thể tách thành verifier riêng `verify/igspc/` hoặc giữ inline trong register tùy quyết định.

### 6.7 Frontend

Thêm vào `frontend/src/constants/`:
```ts
REG_PLATFORM_LABELS["ig_spc"] = "IG SPC (tạo từ parent)"
REG_PLATFORMS_VER["ig_spc"] = []  // verify dùng cookie thường
```

UI cần thêm field cho user upload parent pool file (giống upload mail/proxy hiện tại).

### 6.8 Rate limit awareness ở scheduler

Trong `internal/runner/scheduler.go`, khi `PlatformIGSPC`:
- KHÔNG dùng sticky proxy (mỗi run nên đổi IP để giảm flag bulk)
- Spread out request: tối thiểu 30s giữa các slot nếu dùng cùng parent
- `KeepIPSuccess = false` (forced)

---

## 7. Failure mode + diagnostic

### 7.1 Step A fail (401/403)
→ Parent token rotated hoặc parent bị checkpoint. **Hành động:** mark parent dead, không retry.

### 7.2 Step D/E fail (response 200 nhưng có lỗi)
→ Body shape sai hoặc waterfall_id reuse. **Hành động:** regenerate UUIDs và retry.

### 7.3 Step F response không có `account_created=true`

Parse Bloks bundle để tìm marker:
- `bouncing_cliff` → server bounce flow vì anti-abuse
- `create_failure : system_error` → body shape sai
- `We're sorry, but something went wrong` → generic, có thể rate-limit

**Hành động:**
- Đặt cooldown parent 5 phút
- Nếu fail 3 lần liên tiếp với cùng parent → mark parent quarantine 24h

### 7.4 Account tạo nhưng bị disable sau vài phút

IG chạy delayed anti-abuse check. Account "Instagram user" chưa set username/avatar dễ bị disable.

**Hành động:** Verify email + set username ngay trong session đầu (chưa implement).

---

## 8. Risk model

| Risk | Impact | Mitigation |
|---|---|
| Parent bị checkpoint do spam | **Mất tài sản gốc** | Cooldown 30s + max 5-10 con/parent/day + rotate pool |
| Account con bị disable < 24h | Phí công | Verify email + set username trong session đầu |
| IG update server check (require attestation thật) | **Flow chết hoàn toàn** | Monitor success rate; backup plan = port lên Android farm + Frida hook |
| Capture body bị stale (Bloks version cũ) | Server reject | Định kỳ capture lại (mỗi 2-3 tháng) khi IG update |
| Bearer token rotation | Step A fail | Refresh bearer từ parent (login lại) hoặc bỏ parent |

---

## 9. Tham chiếu

- **Capture gốc:** `E:\WEMAKE\DocWeMake\RegIGinIG\V3\` (75 request/response)
- **Test impl:** `cmd/test_spc_ig/main.go`
- **Live.txt format:** `username|password|cookie|bearer|date|`
- **Doc related:**
  - [`00-tong-quan-kien-truc.md`](./00-tong-quan-kien-truc.md) — kiến trúc tổng
  - [`01-luong-register.md`](./01-luong-register.md) — luồng anonymous reg hiện tại
  - [`02-luong-verify.md`](./02-luong-verify.md) — verify flow

---

*Doc tạo 2026-06-29. Cập nhật khi engine `igspc` được implement chính thức.*
