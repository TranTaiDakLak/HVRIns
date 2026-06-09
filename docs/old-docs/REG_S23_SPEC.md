# SPEC — S23 Register API (Go vs C# parity)

**Mục đích**: spec 100% chính xác để port Go `internal/facebook/register/s23` khớp byte-level với C# `FacebookRegisterAPIAndroidS23.cs`. User review spec này trước khi tôi sửa code.

**Nguồn C#**: `E:\WEMAKE\FULL-REG-CLONE-HAVU\VerifyCloneVIP\`

---

## 1. Flow tổng quát

```
RegAndVerify(thread) → (Automation)
 │
 ├── [Optional] FacebookRegisterAPIAndroidS23.GetPwdKey(...)
 │   → POST b-graph.facebook.com//pwd_key_fetch
 │   (S23.Register KHÔNG tự gọi hàm này — Automation/FMain quyết định)
 │
 ├── FacebookRegisterAPIAndroidS23.Register(accountInfo, httpRequest, deviceid, waterfall_id)
 │   │
 │   ├── Set accountInfo.EncryptedPassword = "#PWD_FB4A:0:{ts}:{password}" (PLAIN, no RSA)
 │   │
 │   ├── Build headers (see section 3)
 │   │
 │   ├── payload = FacebookApiFormDataBuilder.RegisterFormDataS23(accountInfo, deviceid, waterfall_id, locale, S23BloksVer)
 │   │   = CreateAccountVariables_v3 + 2 string replace (bloks_ver, styles_id)
 │   │
 │   ├── POST UrlSingleton.BGraphqlUrl (= https://b-graph.facebook.com/graphql)
 │   │   Content-Type: application/x-www-form-urlencoded
 │   │   Content-Length: payload.Length
 │   │   → response body
 │   │
 │   ├── Parse response:
 │   │     - "couldn't create an account for you" → Blocked
 │   │     - No c_user found → UnknownBlockType
 │   │     - Success:
 │   │         * Strip "\\" from response
 │   │         * AccessToken = "EAA" + regex `EAA(.*?)"`
 │   │         * Uid = regex `c_user","value":"(.*?)"`
 │   │         * xs = regex `name":"xs","value":"(.*?)"`
 │   │         * fr = regex `name":"fr","value":"(.*?)"`
 │   │         * datr = regex `name":"datr","value":"(.*?)"`
 │   │         * Cookie = "c_user={uid};xs={xs};locale={locale};fr={fr};datr={datr};"
 │   │         * MachineId = datr (nếu datr != "")
 │   │         * FacebookLogoutSessionUtils.TryAddNewDatrToPool(Cookie)
 │   │
 │   └── SleepRandom(1,2) → GetXZeroEh()
 │       → POST b-graph.facebook.com/?include_headers=false&decode_body_json=false&streamable_json_response=true&locale=...&client_country_code=...
 │
 └── [Optional] LogoutAccount (C# Automation gọi nếu !VerifyAfterReg)
     → POST b-graph.facebook.com/auth/expire_session
```

---

## 2. Constants

### 2.1 S23-specific

```csharp
// FacebookRegisterAPIAndroidS23.cs L54-57
private const string S23BloksVer    = "d90663010f8c230bedf28906f2bac9c1d1f532a275373050778e36e76a7cb999";
private const string S23MetaZCA     = "empty_token";
private const string S23OAuthToken  = "350685531728|62f8ce9f74b12f84c123cc23437a4a32";
```

**Go hiện tại** ([constants.go](internal/facebook/register/s23/constants.go)):
- bloksVer = `d906630...b999` ✅
- MetaZCA = `empty_token` ✅
- OAuthToken = `350685531728|62f8ce9f74b12f84c123cc23437a4a32` ✅

### 2.2 V3 schema constants (inherited, then swapped for S23)

```csharp
// CreateAccountVariables_v3 dùng 2 value V1, RegisterFormDataS23 replace thành S23:
// V1 bloks_ver: "0b868a90533a800ff97deb1c85ace5bdbe52f18a1004907dff5d1bbda20b8b2e"
// V1 styles_id: "5e69aafc13b802e5d3b57b9257525433"
// S23 swap:
//   bloks_ver → S23BloksVer (d90663...)
//   styles_id → "6100e7e89411ccf67ace027cedecd84f"

// FacebookApiFormDataBuilder.cs L36, L85
public static string RegisterAndroid_fb_api_req_friendly_name = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.create.account.async";
private static string ChangeAndConfirmContactpointMobiledocid_v3 => "1199408042526631289603660492";
```

**Go** ([constants.go](internal/facebook/register/s23/constants.go)):
- s23FriendlyName = match ✅
- s23DocID = `"1199408042526631289603660492"` → **cần verify** (audit trước báo sai)

### 2.3 Other constants in body

| Field | C# Value | Go status |
|---|---|---|
| `offline_experiment_group` | `caa_iteration_v6_perf_fb_2` | ✅ match |
| `access_flow_version` | `pre_mt_behavior` | ✅ match |
| `current_step` | `8` | ✅ match |
| `app_id` | `com.bloks.www.bloks.caa.reg.create.account.async` | ✅ match |
| `INTERNAL__latency_qpl_marker_id` | `36707139` | ✅ match |
| `caa_reg_flow_source` | `login_home_native_integration_point` | ✅ match |
| `flow_name` (nested) | `new_to_family_fb_default` | ✅ match |
| `flow_type` (nested) | `ntf` | ✅ match |
| `styles_id` (nt_context) | S23 value `6100e7e89411ccf67ace027cedecd84f` | ✅ match |
| `scale` | HARDCODE `"3"` trong body | ⚠️ Go dùng `profile.Device.FBSS` (có thể "3" hoặc "4") — **C# fixed "3"** |
| `theme_params.value` | `["BLUEPRINT_TEST_GUTTER"]` (1 item) | ✅ match |

---

## 3. Headers (order matters!)

Thứ tự C# build (từ `Register` method):

```csharp
// (A) Base: RegisterWIFIHeaderCollection(S23OAuthToken, true, accountInfo)
//     → trả về dict headers chứa Authorization/OAuth + X-Fb-Connection-Type + X-Fb-Sim-Hni/Net-Hni + ...
//     (cần đọc FacebookApiHeaderCollectionBuilder.cs để enumerate full)

// (B) Additional headers:
"X-Fb-Device-Group": accountInfo.DeviceGroup
"App-Scope-Id-Header": deviceid
"X-Fb-Integrity-Machine-Id": accountInfo.MachineId  (chỉ nếu MachineId != "")
"X-Zero-F-Device-Id": accountInfo.FamilyDeviceId

// (C) S23-specific overrides:
"X-Meta-Zca": "empty_token"
"x-meta-usdid": GenerateUSDID()     // ECDSA P-256 sign
"x-fb-conn-uuid-client": Guid.NewGuid().ToString("N")  // UUID không dấu "-"

// (D) Auto-added:
"Content-Length": payload.Length
"Content-Type": application/x-www-form-urlencoded (from CustomHttpSingleton.FormUrlEncoded)
```

**Go cần verify** ([headers.go](internal/facebook/register/s23/headers.go)):
- Full list + order khớp (A) → (B) → (C) → (D)
- `USDID` ECDSA sign đã implement

**ACTION**: cần đọc `FacebookApiHeaderCollectionBuilder.cs` để liệt kê (A) đầy đủ. Audit trước chưa có list này.

---

## 4. Body — CreateAccountVariables_v3

### Cấu trúc JSON nested 4 levels (URL-encoded):

```
variables={
  "params": {
    "params": "{
      \"params\": \"{
        \\\"client_input_params\\\": { 16 fields },
        \\\"server_params\\\": {
          common fields (device_id, flow_info, reg_info...) + 
          reg_info (string containing 170+ fields JSON-escaped),
          x_app_device_signals: {MACHINE_ID, DEVICE_ID},
          current_step: 8
        }
      }\"
    }",
    "bloks_versioning_id": "<S23 bloks>",
    "app_id": "com.bloks.www.bloks.caa.reg.create.account.async"
  },
  "scale": "3",
  "nt_context": {
    "using_white_navbar": true,
    "styles_id": "6100e7e89411ccf67ace027cedecd84f",  (S23 value)
    "pixel_ratio": 3,
    "is_push_on": true,
    "debug_tooling_metadata_token": null,
    "is_flipper_enabled": false,
    "theme_params": [{"value": ["BLUEPRINT_TEST_GUTTER"], "design_system_name": "FDS"}],
    "bloks_version": "<S23 bloks>"
  }
}
```

### 4.1 client_input_params (16 fields)

| # | Field | Value | Go |
|---|---|---|---|
| 1 | ck_error | `""` | ✅ |
| 2 | device_id | `{deviceid}` | ✅ |
| 3 | waterfall_id | `{waterfall_id}` | ✅ |
| 4 | zero_balance_state | `"init"` | ✅ |
| 5 | failed_birthday_year_count | `""` | ✅ |
| 6 | headers_last_infra_flow_id | `""` | ✅ |
| 7 | ig_partially_created_account_nonce_expiry | `0` | ✅ |
| 8 | machine_id | `""` (empty! khác reg_info.machine_id) | ✅ |
| 9 | reached_from_tos_screen | `1` | ✅ |
| 10 | ig_partially_created_account_nonce | `""` | ✅ |
| 11 | ck_nonce | `""` | ✅ |
| 12 | lois_settings | `{"lois_token": ""}` | ✅ |
| 13 | ig_partially_created_account_user_id | `0` | ✅ |
| 14 | ck_id | `""` | ✅ |
| 15 | no_contact_perm_email_oauth_token | `""` | ✅ |
| 16 | encrypted_msisdn | `""` | ✅ |

**Go client_input_params**: **KHỚP 100%** ✅

### 4.2 server_params top-level fields

| # | Field | Value | Go |
|---|---|---|---|
| 1 | event_request_id | `{guid}` | ✅ |
| 2 | is_from_logged_out | `0` | ✅ |
| 3 | layered_homepage_experiment_group | `null` | ✅ |
| 4 | device_id | `{deviceid}` | ✅ |
| 5 | reg_context | `null` | ✅ |
| 6 | waterfall_id | `{waterfall_id}` | ✅ |
| 7 | INTERNAL__latency_qpl_instance_id | `{calc}` | ✅ |
| 8 | flow_info | `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}` | ✅ |
| 9 | is_platform_login | `0` | ✅ |
| 10 | INTERNAL__latency_qpl_marker_id | `36707139` | ✅ |
| 11 | reg_info | (string JSON) — **170+ fields, see 4.3** | ⚠️ Go thiếu nhiều |
| 12 | family_device_id | `{accountInfo.FamilyDeviceId}` | ✅ |
| 13 | offline_experiment_group | `caa_iteration_v6_perf_fb_2` | ✅ |
| 14 | x_app_device_signals | `{MACHINE_ID: accountInfo.MachineId2, DEVICE_ID: accountInfo.AndroidId}` | ✅ |
| 15 | access_flow_version | `pre_mt_behavior` | ✅ |
| 16 | is_from_logged_in_switcher | `0` | ✅ |
| 17 | current_step | `8` | ✅ |

**Go server_params top-level**: **KHỚP 100%** ✅

### 4.3 reg_info — **điểm yếu lớn nhất**

C# liệt kê **~177 fields** trong `reg_info`. Go hiện chỉ có **~30 fields**.

**Missing fields (theo thứ tự C#)**:

<details>
<summary><b>Nhóm 1: Identity/Contact (cần check) — 12 fields</b></summary>

- `unified_cp_screen_variant: null`
- `confirmation_code: null`
- `birthday_derived_from_age: null`
- `custom_gender: null`
- `username: null`
- `username_prefill: null`
- `fb_conf_source: null`
- `ig4a_qe_device_id: null`
- `user_id: null`
- `safetynet_token: null`
- `skip_slow_rel_check: false`
- `safetynet_response: null`
</details>

<details>
<summary><b>Nhóm 2: Profile photo — 4 fields</b></summary>

- `profile_photo: null`
- `profile_photo_id: null`
- `profile_photo_upload_id: null`
- `avatar: null`
</details>

<details>
<summary><b>Nhóm 3: Email OAuth — 7 fields</b></summary>

- `email_oauth_token_no_contact_perm: null`
- `email_oauth_token: null`
- `email_oauth_tokens: {}`
- `should_skip_two_step_conf: null`
- `openid_tokens_for_testing: null`
- `encrypted_msisdn: null`
- `encrypted_msisdn_for_safetynet: null`
</details>

<details>
<summary><b>Nhóm 4: Safetynet headers — 6 fields</b></summary>

- `cached_headers_safetynet_info: null`
- `should_skip_headers_safetynet: null`
- `headers_last_infra_flow_id: null`
- `headers_last_infra_flow_id_safetynet: null`
- `headers_flow_id: {guid}` (Go có nhưng position khác?)
- `was_headers_prefill_available: false`
</details>

<details>
<summary><b>Nhóm 5: Sync/Account Creation — 11 fields</b></summary>

- `sso_enabled: null`
- `existing_accounts: null`
- `used_ig_birthday: null`
- `sync_info: null`
- `create_new_to_app_account: null`
- `skip_session_info: null`
- `ck_error: null`
- `ck_id: null`
- `ck_nonce: null`
- `horizon_synced_username: null`
- `fb_access_token: null`
</details>

<details>
<summary><b>Nhóm 6: Horizon/Identity sync — 3 fields</b></summary>

- `horizon_synced_profile_pic: null`
- `is_identity_synced: false`
- `is_msplit_reg: null`
</details>

<details>
<summary><b>Nhóm 7: Spectra/Family sync — 12 fields</b></summary>

- `is_spectra_reg: null`
- `dema_account_consent_given: null`
- `spectra_reg_token: null`
- `spectra_reg_guardian_id: null`
- `spectra_reg_guardian_logged_in_context: null`
- `user_id_of_msplit_creator: null`
- `msplit_creator_nonce: null`
- `dma_data_combination_consent_given: null`
- `xapp_accounts: null`
- `fb_device_id: null`
- `fb_machine_id: null`
- `ig_device_id: null`
</details>

<details>
<summary><b>Nhóm 8: NTA/Big Blue — 6 fields</b></summary>

- `ig_machine_id: null`
- `should_skip_nta_upsell: null`
- `big_blue_token: null`
- `skip_sync_step_nta: null`
- `ig_authorization_token: null`
- `full_sheet_flow: false`
</details>

<details>
<summary><b>Nhóm 9: SUMA / existing account check — 24 fields</b></summary>

- `crypted_user_id: null`
- `ignore_suma_check: false`
- `dismissed_login_upsell_with_cna: false`
- `ignore_existing_login: false`
- `ignore_existing_login_from_suma: false`
- `ignore_existing_login_after_errors: false`
- `suggested_first_name: null`
- `suggested_last_name: null`
- `suggested_full_name: null`
- `frl_authorization_token: null`
- `post_form_errors: null`
- `skip_step_without_errors: false`
- `existing_account_exact_match_checked: true`
- `existing_account_fuzzy_match_checked: false`
- `email_oauth_exists: false`
- `confirmation_code_send_error: null`
- `is_too_young: false`
- `source_account_type: null`
- `whatsapp_installed_on_client: false`
- `confirmation_medium: null`
- `source_credentials_type: null`
- `source_cuid: null`
- `source_account_reg_info: null`
- `soap_creation_source: null`
</details>

<details>
<summary><b>Nhóm 10: Youth/Cold start — 13 fields</b></summary>

- `source_account_type_to_reg_info: null`
- `should_skip_youth_tos: false`
- `is_youth_regulation_flow_complete: false`
- `is_on_cold_start: false`
- `email_prefilled: false`
- `cp_confirmed_by_auto_conf: false`
- `in_sowa_experiment: false`
- `youth_regulation_config: null`
- `conf_allow_back_nav_after_change_cp: null`
- `conf_bouncing_cliff_screen_type: null`
- `conf_show_bouncing_cliff: null`
- `eligible_to_flash_call_in_ig4a: false`
- `request_data_and_challenge_nonce_string: null`
</details>

<details>
<summary><b>Nhóm 11: Confirmation/Notification — 14 fields</b></summary>

- `confirmed_cp_and_code: null`
- `notification_callback_id: null`
- `reg_suma_state: 0`
- `is_msplit_neutral_choice: false`
- `msg_previous_cp: null`
- `ntp_import_source_info: null`
- `youth_consent_decision_time: null`
- `should_show_spi_before_conf: true`
- `google_oauth_account: null`
- `is_reg_request_from_ig_suma: false`
- `device_emails: []`
- `is_toa_reg: false`
- `is_threads_public: false`
- `spc_import_flow: false`
</details>

<details>
<summary><b>Nhóm 12: Play Integrity + other — 11 fields</b></summary>

- `caa_play_integrity_attestation_result: null`
- `client_known_key_hash: null`
- `flash_call_provider: null`
- `spc_birthday_input: false`
- `failed_birthday_year_count: null` (note: client_input_params có 1 cái cùng tên = "")
- `user_presented_medium_source: null`
- `user_opted_out_of_ntp: null`
- `is_from_registration_reminder: false`
- `show_youth_reg_in_ig_spc: false`
- `fb_suma_combined_landing_candidate_variant: "control"`
- `fb_suma_is_high_confidence: null`
</details>

<details>
<summary><b>Nhóm 13: screen_visited + SUMA — 6 fields</b></summary>

- `screen_visited: [...]` ✅ Go có, nhưng hardcode `CONTACT_POINT_PHONE`. C# dynamic theo contactpoint_type:
  - phone: `["CAA_REG_WELCOME_SCREEN","bloks.caa.reg.birthday","CAA_REG_CONTACT_POINT_PHONE","CAA_REG_PASSWORD","CAA_REG_SAVE_PASSWORD_CREDENTIALS"]`
  - **C# CODE CHO S23 hardcode PHONE cả 2 case** (Tôi kiểm tra lại — không thấy C# branch theo email trong S23 body. Go OK)
- `fb_email_login_upsell_skip_suma_post_tos: false`
- `fb_suma_is_from_email_login_upsell: false`
- `fb_suma_is_from_phone_login_upsell: false`
- `fb_suma_login_upsell_skipped_warmup: false`
- `fb_suma_login_upsell_show_list_cell_link: false`
</details>

<details>
<summary><b>Nhóm 14: cuối — 23 fields</b></summary>

- `should_prefill_cp_in_ar: false` (Go có nhưng ở đâu?)
- `ig_partially_created_account_user_id: null`
- `ig_partially_created_account_nonce: null`
- `ig_partially_created_account_nonce_expiry: null`
- `force_sessionless_nux_experience: false`
- `has_seen_suma_landing_page_pre_conf: false`
- `has_seen_suma_candidate_page_pre_conf: false`
- `suma_on_conf_threshold: -1`
- `is_keyboard_autofocus: null`
- `pp_to_nux_eligible: false`
- `should_show_error_msg: true`
- `welcome_ar_entrypoint: "control"`
- `th_profile_photo_token: null`
- `attempted_silent_auth_in_fb: false`
- `cp_suma_results_map: null`
- `source_username: null`
- `next_uri: null`
- `should_use_next_uri: null`
- `linking_entry_point: null`
- `fb_encrypted_partial_new_account_properties: null`
- `starter_pack_name: null`
- `starter_pack_creator_user_ids: null`
- `wa_data_bundle: null`
</details>

**Go currently has in reg_info** (~30 fields):
- first_name, last_name, full_name, contactpoint, ar_contactpoint, contactpoint_type, is_using_unified_cp, is_cp_auto_confirmed, is_cp_auto_confirmable, is_cp_claimed, birthday, did_use_age, gender, use_custom_gender, encrypted_password, device_id, family_device_id, machine_id, should_save_password, caa_reg_flow_source, is_caa_perf_enabled, is_preform, registration_flow_id, headers_flow_id, screen_visited, should_prefill_cp_in_ar, flash_call_permissions_status, attestation_result

---

## 5. Fields Go đã có nhưng C# KHÔNG có (over-send)

Cần kiểm tra và có thể XÓA khỏi Go:
- Không phát hiện — Go subset của C#.

---

## 6. URL-escape levels trong payload

C# nested 4 levels → cần escape `"` thành:
- Level 1 (outer): `"` → `"` (no escape)
- Level 2 (params.params string): `"` → `\"` → URL-encoded `%5C%22`
- Level 3 (params.params.params): `\"` → `\\\"` → URL-encoded `%5C%5C%5C%22`
- Level 4 (reg_info nested): `\\\"` → `\\\\\\\"` → URL-encoded `%5C%5C%5C%5C%5C%5C%5C%22`

**`/` trong encrypted_password**: C# escape thành `\\\\\\\\\\/` trước khi đưa vào level 4 string → Go có line 44 `strings.ReplaceAll(encPassword, "/", ...)` ✅

---

## 7. GetXZeroEh (bước sau reg success)

**C# flow** (S23.cs L253-295):

```
POST https://b-graph.facebook.com/?include_headers=false&decode_body_json=false&streamable_json_response=true&locale={locale}&client_country_code={cc}

Headers:
  (A) BaseAndroidAPIHeadersWIFI(AccessToken, "fetchLoginData-batch", accountInfo)
  (B) BaseAndroidDevicexConnectHeaders(accountInfo)
  (C) "X-Zero-F-Device-Id": FamilyDeviceId
      "App-Scope-Id-Header": deviceid
      "X-Zero-State": "unknown"
      "X-Meta-Zca": _defaultMetaZcaHeaderValue (NOTE: khác Register() — dùng base64 blob, không empty_token)

Body: FacebookApiFormDataBuilder.GetXzeroEhMobileFormData(accountInfo)
  → cần đọc — đây là batch fetchZeroToken với carrier_mcc/sim_mcc/interface/eligibility_hash/token_hash

Parse: regex `eligibility_hash":"(.*?)"`
```

**Go hiện tại**: ❌ body XZeroEH dùng `/me?fields=eligibility_hash` hoàn toàn sai. Cần port `GetXzeroEhMobileFormData`.

---

## 8. LogoutAccount (tùy chọn sau reg)

**C# flow** (S23.cs L168-196):

```
POST https://b-graph.facebook.com/auth/expire_session

Headers:
  (A) BaseAndroidAPIHeadersWIFI(AccessToken, "logout", accountInfo)
  (B) BaseAndroidDevicexConnectHeaders(accountInfo)
  (C) "X-Zero-F-Device-Id": FamilyDeviceId
      "App-Scope-Id-Header": deviceid
      "X-Meta-Zca": _defaultMetaZcaHeaderValue  (base64 blob, không empty_token)
      "X-Fb-Connection-Quality": "EXCELLENT"

Body (form-urlencoded):
  reason=USER_INITIATED
  device_id={deviceid}
  retain_for_dbl=false
  logout_source=REGISTRATION
  locale={locale}
  client_country_code={cc}
  fb_api_req_friendly_name=logout
  fb_api_caller_class=Fb4aLogoutOperationsHelper
```

**Go hiện tại**: ❌ Thiếu hoàn toàn. Cần implement.

---

## 9. Response parsing

**Regex C#** (identical in Go):

```
EAA(.*?)"                              → AccessToken (prepend "EAA")
c_user","value":"(.*?)"                → Uid
name":"xs","value":"(.*?)"             → xs
name":"fr","value":"(.*?)"             → fr
name":"datr","value":"(.*?)"           → datr
```

Trước khi regex, strip `\\` khỏi response body.

**Go response.go** cần verify khớp.

---

## 10. Summary — Điểm CẦN FIX

### ⭐ CRITICAL (body sai → FB có thể reject)

1. **reg_info thiếu ~150 fields** → port đầy đủ theo section 4.3
2. **GetXZeroEh body sai** → port `GetXzeroEhMobileFormData` (batch fetchZeroToken)

### 🔸 HIGH (thiếu endpoint / logic)

3. **LogoutAccount** → thêm method gọi `/auth/expire_session`
4. **Headers `RegisterWIFIHeaderCollection` + `BaseAndroidAPIHeadersWIFI` + `BaseAndroidDevicexConnectHeaders`** → cần đọc `FacebookApiHeaderCollectionBuilder.cs` để enumerate đầy đủ header list Go đang build
5. **`X-Meta-Zca` khác nhau theo endpoint**: Register dùng `empty_token`, GetXZeroEh + Logout dùng `_defaultMetaZcaHeaderValue` (base64 blob)

### 🔹 MEDIUM

6. **`scale` hardcoded `"3"`** trong C# body, Go đang dùng `profile.Device.FBSS` dynamic — có thể đúng hoặc sai tùy quan điểm
7. **`headers_flow_id` position** trong reg_info — verify khớp C# order
8. **`contactpoint email` escape** — C# encode `@` khác: `accountInfo.Email.Split('@')[0] + "%5C%5C%5C%5C%5C%5C%5C%5Cu0040" + accountInfo.Email.Split('@')[1]` — Go cần port

### 🔸 LOW

9. **`GetPwdKey` endpoint** — C# có nhưng Register() KHÔNG gọi. Nếu muốn encrypt password `#PWD_FB4A:2:` thì Automation phải gọi GetPwdKey rồi RSA+AES-GCM encrypt password trước khi Register. Gap này **không critical cho S23** vì C# S23 dùng plaintext ok.

---

## 11. Đề xuất thứ tự port

**Priority 1 (phải làm)**:
1. Đọc `FacebookApiHeaderCollectionBuilder.cs` để spec chính xác header (section 3)
2. Đọc `GetXzeroEhMobileFormData` trong FormDataBuilder để spec body XZeroEH (section 7)
3. Port reg_info thêm ~150 fields (section 4.3)

**Priority 2 (nên làm)**:
4. Port GetXZeroEh body đúng
5. Port LogoutAccount

**Priority 3 (tùy)**:
6. Review scale + email contactpoint encoding

---

## 12. Câu hỏi cho user

1. **Có chắc muốn port full 177 fields** reg_info? FB có thể chấp nhận subset nếu fields required là top-level. Nhưng C# rõ ràng gửi tất cả → ít rủi ro hơn.
2. **Có cần port GetPwdKey + RSA encrypt**? Nếu C# đang dùng plaintext và reg pass → có thể skip.
3. **`scale` giữ dynamic `profile.Device.FBSS` hay hardcode "3"** theo C#?

Đọc xong spec này, confirm tôi fix từng mục hay làm tất cả 1 lần.
