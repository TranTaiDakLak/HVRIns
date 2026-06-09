# So sánh logic Register: NCS/HVR (Go) vs FULL REG CLONE HAVU (C#)

> **Phạm vi**: chỉ phân tích nhánh **Register** (chưa đụng Verify).
> **Nguồn**:
> - Dự án hiện tại: [d:/NCS/HVR](../) — code Go, chủ yếu folder [internal/facebook/register/](../internal/facebook/register/)
>   (đặc biệt [s23/](../internal/facebook/register/s23) và [android/](../internal/facebook/register/android)).
> - Dự án gốc: `D:/NCS/FULL REG CLONE HAVU/` — code C# WinForms, tham chiếu qua docs ở
>   [`docs/`](file:///D:/NCS/FULL%20REG%20CLONE%20HAVU/docs) (file 04, 05, 12, 18 là chính).
> **Mục tiêu**: so sánh kiến trúc, flow, **đặc biệt phần build header + body params** để rebuild Go đúng "byte-level" với C#.

---

## 0. TL;DR — Bảng đối chiếu nhanh

| Khía cạnh | C# (HAVU) | Go (HVR) |
|---|---|---|
| Stack | WinForms + LeafxNetLib (LeafxNetLibWrapper) | Wails + Vue + Go + bogdanfinn/tls-client |
| TLS fingerprint | Native .NET HttpWebRequest (Liger string trong header) | tls-client profile `Okhttp4Android13` (HTTP/2, JA3 thật giống OkHttp4) |
| Tổ chức | 1 class per API: `FacebookRegisterAPIAndroidS23`, `FacebookRegisterAPIAndroidV2`, `FacebookRegisterMfbRequest`, ... | 1 package per platform: [s23/](../internal/facebook/register/s23), [android/](../internal/facebook/register/android), [s557/](../internal/facebook/register/s557), [web/](../internal/facebook/register/web), ... |
| Pattern shared session | `IHttpRequestClient` reuse trong `RegisterWithKeepHttpSession` | `WorkerContext{sess, profile}` per goroutine — reuse cookie jar + device fingerprint |
| Body builder | `FacebookApiFormDataBuilder` (1 file ~1500 dòng) | `body.go` mỗi platform, port 1:1 từng schema |
| Header builder | `FacebookApiHeaderCollectionBuilder` (helper static) | `http.go` mỗi platform |
| Datr/Cookie pool | `FacebookLogoutSessionUtils.GetPerfectMachineId` + file `cookie_initial.txt` | [`SharedPool`](../internal/facebook/register/android/pool.go) = `PartitionedDatrPool` (per-slot queue) |
| Warm session | `RegisterWithRequestInitialAccount` (nội bộ Mfb) | [`warmSession`](../internal/facebook/register/s23/extras.go) (m.facebook.com web flow tách hẳn) |
| Encrypted password | `#PWD_FB4A:0:{ts}:{plain}` | Giống hệt — [register.go:335](../internal/facebook/register/s23/register.go#L335) |
| Endpoint | `https://b-graph.facebook.com/graphql` (S23) | `facebook.BaseURLBGraph + "/graphql"` |
| Result parsing | Regex thuần (`EAA*`, `c_user","value":"..."`, ...) | Y hệt — [extras.go:706-715](../internal/facebook/register/s23/extras.go#L706-L715) |

→ **Kết luận sơ bộ**: Go bám rất sát C# về schema. **Khác biệt thật sự nằm ở**: (a) cấu trúc multi-platform/factory; (b) cách xử lý warm session (Go tách hẳn web layer ra); (c) một số khác biệt nhỏ về thứ tự header và case-sensitivity (xem mục 4).

---

## 1. Flow tổng thể của Register

### 1.1 C# (HAVU) — `FacebookRegisterAutomation.Register`
Pseudo (từ [`docs/04-automation-orchestrators.md`](file:///D:/NCS/FULL%20REG%20CLONE%20HAVU/docs/04-automation-orchestrators.md) §4.1):

```
Register(o):
  1. IPLookup → set FMainDTO.IP, .Country
  2. (CookieInitial?)  MachineId = GetPerfectMachineId(method, limit)
  3. UA = UserAgentBuilder.GetUserAgent(locale, addVirtualSpecs, useBuildnumFile, sim_brand)
     - Nếu Browser builder → split "ua|model|os" thành (ua, AndroidDeviceModel, AndroidOsVersion)
  4. FacebookAccountModel.InstanceRandom(country) → name/birthday/gender/sim/locale/email/phone
  5. DeviceId  = Guid.NewGuid()
     Waterfall = Guid.NewGuid()
     httpClient.UserAgent = ua
  6. regResult = facebookRegister.Register(account, http, deviceid, waterfall_id)
  7. IncrementMachineIdUsage(MachineId)
  8. Sync FacebookAccount → FMainDTO (Uid/Pass/Cookie/Token/Login/FullName)
  9. CheckLiveUid(Uid) → Checkpoint? Error? Success?
     - Liberal: network lỗi check live vẫn trả Success (set IsErrorWhileCheckLive=true)
```

Mỗi `IFacebookRegister` impl (S23, V2 Android, Token, Mfb, Chrome) tự xử lý phần 6 — build header + body + POST + parse + sync vào `FacebookAccount`.

### 1.2 Go (HVR) — `s23.WorkerContext.Register`
Pseudo (từ [register.go:179-433](../internal/facebook/register/s23/register.go#L179-L433)):

```
Register(ctx, input, onStatus):
  1. ParseSeed(input.TutDatr) → Seed{Mode, Datr, CookieString, UID, Password, ...}
     - Gate: chỉ method "ck" mới được SeedModeInitialAccount (warm login)
     - Khác mode: downgrade về SeedModeFullCookie / SeedModeDatrOnly
  2. profile := w.profile (deep copy — tránh mutate pinned)
     - Override FirstName/LastName/Birthday/Gender/UA/MachineID từ input
  3. SharedPool: nếu profile.MachineID == "" → GetNext(slotIdx)
     - Defer SharedPool.IncrementUsage(machineID)
  4. password = input.Password ?? RandomPassword()
  5. contactpoint, type ← input.Email | input.Phone
  6. sess.clearCookies()  ← KHÔNG inject datr thành cookie HTTP (match C# block đã comment-out)
  7. SeedModeFullCookie → seed cookies (skip c_user/xs/datr)
  8. SeedModeInitialAccount → warmSession(ctx, sess, seed, proxy, notify)
  9. encPassword = "#PWD_FB4A:0:{ts}:{plain}"
 10. body = buildRegisterBody(profile, encPassword, contactpoint, type, locale)
     headers = buildHeaders(profile)
 11. respBody = sess.postGzip(BaseURLBGraph + "/graphql", body, headers)   ← postGzip thực chất KHÔNG gzip
 12. parsed = parseRegisterResponse(respBody, locale)
     - "couldn't create an account for you" / "integrity_block" → Blocked
     - regex: AccessToken, UID (4 fallback patterns), xs, fr, datr
 13. Sleep random(1000-2000ms) → fetchXZeroEh(sess, profile, accessToken, deviceID)
 14. Pool: AddDatrRaw(parsed.DATR), RecordResult(machineID, "success")
 15. Trả RegResult{Success, UID, Cookie, AccessToken, Password, UserAgent, DeviceID, FamilyDeviceID}
```

### 1.3 Khác biệt cốt lõi

| # | C# | Go | Ghi chú |
|---|---|---|---|
| 1 | IPLookup TRƯỚC reg | Không có trong Register, dồn lên runner/automation | Go đã tách concern: register module thuần API |
| 2 | UA random từ `IUserAgentBuilder` mỗi reg | UA pin trong `WorkerContext.profile` (rebuild khi `NewWorkerContext`) | Go pre-build profile cho 1 worker → reuse N reg, giảm overhead |
| 3 | DeviceId / Waterfall_id sinh trong automation | Sinh trong `BuildProfileForPlatform` (profile.go:83-85) | Go gắn chặt vào device profile |
| 4 | `RegisterWithKeepHttpSession` chỉ Mfb dùng | Mọi platform Go đều dùng `WorkerContext` (= keep-session) | Go thống nhất pattern |
| 5 | CheckLiveUid là phần của Register | Tách ra runner — register chỉ trả parsed UID/Cookie/Token | Lỗi sạch hơn |
| 6 | Warm = `RegisterWithRequestInitialAccount` (Mfb only) | `warmSession` = web HTTP/1.1 client riêng (fhttp + cookiejar), **best-effort 12s timeout** | Go tách network stack: TLS fingerprint cho reg ≠ web cho warm |

---

## 2. Build BODY params — phân tích chi tiết

### 2.1 C# `CreateAccountVariables_v3` (S23)
- File [`FacebookApiFormDataBuilder.cs`](file:///D:/NCS/FULL%20REG%20CLONE%20HAVU/VerifyCloneVIP/Models/FacebookApiFormDataBuilder.cs) L1074-1091.
- Thực chất là `CreateAccountVariables_v3` (V1) + 2 string-replace để swap:
  - `bloks_versioning_id` → `S23BloksVer` (`d90663...b999`)
  - `styles_id` → `6100e7e89411ccf67ace027cedecd84f`
- Cấu trúc 4 level nested JSON, **đã URL-encode toàn bộ** trước khi đặt vào form param `variables=`:
  ```
  variables = {
    "params": {
      "params": "<L2 JSON string>",
      "bloks_versioning_id": "<S23BloksVer>",
      "app_id": "com.bloks.www.bloks.caa.reg.create.account.async"
    },
    "scale": "3",
    "nt_context": { using_white_navbar, styles_id, pixel_ratio, is_push_on, ... bloks_version }
  }
  ```
  L2 = `{"params":"<L3>"}`
  L3 = `{"client_input_params":{...16 fields...},"server_params":{..., "reg_info":"<L4>", ...}}`
  L4 = `{...177 flat fields trong reg_info...}`
- **Escape level**: mỗi cấp lồng nhau, dấu `"` được nhân lên `\` (`\"`, `\\\"`, `\\\\\\\"`). Sau khi URL-encode → `%22`, `%5C%22`, `%5C%5C%5C%22`, `%5C%5C%5C%5C%5C%5C%5C%22`.

### 2.2 Go `createAccountVariablesS23`
- File [s23/body.go:190-478](../internal/facebook/register/s23/body.go#L190-L478).
- Port 1:1, có comment chỉ chính xác dòng C# tương ứng.
- Dùng helper level-4: `l4KvStr/l4KvNull/l4KvBool/l4KvInt/l4KvEmpty/l4KvEmptyArr/l4KvRaw` để tránh viết tay 177 lần escape sequence `%5C%5C%5C%5C%5C%5C%5C%22`.
- Constants escape:
  ```go
  lqOpen  = "%5C%5C%5C%5C%5C%5C%5C%22"  // level-4 quote (đã URL-encode)
  lqClose = "%5C%5C%5C%5C%5C%5C%5C%22"
  comma   = "%2C"
  colon   = "%3A"
  ```
- Special escape:
  - `encPassword` → replace `/` thành `\\\\\\\\\\/` (match C# L1076).
  - `contactpoint` (email) → encode `@` thành `%5C%5C%5C%5C%5C%5C%5C%5Cu0040` (match C# L1081).
  - `contactpoint` (phone) → `url.QueryEscape` chuẩn (C# `WebUtility.UrlEncode`, L1085).

### 2.3 reg_info — 177 fields (L4)
Thứ tự field PHẢI giữ nguyên (Facebook server validate strict order trong vài endpoint).
Bảng đối chiếu tóm tắt:

| Block | Field range | C# | Go [s23/body.go](../internal/facebook/register/s23/body.go) |
|---|---|---|---|
| Identity/Contact | 1-19 | first_name…encrypted_password | L220-238 ✅ |
| Username/Device IDs | 20-30 | username…machine_id | L240-250 ✅ |
| Profile photo | 31-34 | profile_photo… | L252-255 ✅ |
| Email OAuth | 35-41 | email_oauth_token… | L257-263 ✅ |
| Safetynet headers | 42-47 | cached_headers_safetynet… | L265-270 ✅ |
| Sync info | 48-57 | sso_enabled…should_save_password | L272-281 ✅ |
| Horizon/Identity | 58-62 | horizon_synced_username… | L283-287 ✅ |
| Spectra/Family | 63-75 | is_spectra_reg…ig_machine_id | L289-301 ✅ |
| NTA/Big Blue | 76-78 | should_skip_nta_upsell… | L303-305 ✅ |
| Flow source | 79-84 | caa_reg_flow_source… | L307-312 ✅ |
| SUMA / existing | 85-108 | ignore_suma_check…source_account_type_to_reg_info | L314-337 ✅ |
| Registration flow | 109 | registration_flow_id | L339 ✅ |
| Youth/Cold start | 110-120 | should_skip_youth_tos…eligible_to_flash_call_in_ig4a | L341-351 ✅ |
| Flash call + attestation | 121-122 | flash_call_permissions_status (object), attestation_result ({}) | L353-359 ✅ |
| Post-flash-call | 123-131 | request_data_and_challenge_nonce_string…should_show_spi_before_conf | L361-369 ✅ |
| Google/Threads/TOA | 132-137 | google_oauth_account…spc_import_flow | L371-376 ✅ |
| Play integrity + birthday | 138-146 | caa_play_integrity_attestation_result…show_youth_reg_in_ig_spc | L378-386 ✅ |
| SUMA landing | 147-148 | fb_suma_combined_landing_candidate_variant, fb_suma_is_high_confidence | L388-389 ✅ |
| Screen visited | 149 | array of 5 screen names | L391-398 ✅ |
| SUMA upsell | 150-154 | fb_email_login_upsell_skip_suma_post_tos… | L400-404 ✅ |
| IG partially created | 155-158 | should_prefill_cp_in_ar… | L406-409 ✅ |
| Force/welcome/AR | 159-168 | force_sessionless_nux_experience…attempted_silent_auth_in_fb | L411-420 ✅ |
| Tail | 169-177 | cp_suma_results_map…wa_data_bundle | L422-430 ✅ |

→ Go đã đầy đủ 177 fields, đúng thứ tự. **Đây là phần khó nhất và Go đã port chuẩn.**

### 2.4 client_input_params + server_params (L3)
Go inline trong return statement [s23/body.go:436-477](../internal/facebook/register/s23/body.go#L436-L477):

```
%7B%22params%22%3A%7B%22params%22%3A%22%7B
  client_input_params { ck_error, device_id, waterfall_id, zero_balance_state, failed_birthday_year_count,
                        headers_last_infra_flow_id, ig_partially_created_account_nonce_expiry, machine_id,
                        reached_from_tos_screen, ig_partially_created_account_nonce, ck_nonce,
                        lois_settings{lois_token}, ig_partially_created_account_user_id, ck_id,
                        no_contact_perm_email_oauth_token, encrypted_msisdn }   // 16 fields
  server_params      { event_request_id, is_from_logged_out, layered_homepage_experiment_group, device_id,
                        reg_context, waterfall_id, INTERNAL__latency_qpl_instance_id, flow_info(string!),
                        is_platform_login, INTERNAL__latency_qpl_marker_id, reg_info(string of L4!),
                        family_device_id, offline_experiment_group, x_app_device_signals{MACHINE_ID, DEVICE_ID},
                        access_flow_version, is_from_logged_in_switcher, current_step }
}%7D
%7D%2C%22bloks_versioning_id%22%3A%22... S23BloksVer ...
%2C%22app_id%22%3A%22com.bloks.www.bloks.caa.reg.create.account.async
%2C%22scale%22%3A%223
%2C%22nt_context%22%3A%7B using_white_navbar:true, styles_id, pixel_ratio:3, is_push_on:true,
                        debug_tooling_metadata_token:null, is_flipper_enabled:false, theme_params:[…FDS…],
                        bloks_version }
```

`flow_info` = `"{\"flow_name\":\"new_to_family_fb_default\",\"flow_type\":\"ntf\"}"` (string của JSON nội).
`x_app_device_signals.MACHINE_ID` = `profile.MachineID2` (random alphanum 28), **không phải `MachineID`** (= datr).
`x_app_device_signals.DEVICE_ID` = `profile.FullRegProfile.Device.AndroidID` (`"android-" + 16 hex`).

### 2.5 Form-urlencoded outer — `buildRegisterBody`
Cả C# và Go đều ráp:
```
method=post&pretty=false&format=json&server_timestamps=true&locale={locale}&purpose=fetch
&fb_api_req_friendly_name=FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.create.account.async
&fb_api_caller_class=graphservice
&client_doc_id={s23DocID}
&fb_api_client_context=%7B%22is_background%22%3Afalse%7D
&variables={createAccountVariablesS23 result}
&fb_api_analytics_tags=%5B%22GraphServices%22%5D
&client_trace_id={uuid}
```
→ [s23/body.go:163-184](../internal/facebook/register/s23/body.go#L163-L184) match C# `RegisterFormDataS23`.

### 2.6 Khác biệt body giữa S23 và Android V22
- Schema gốc rất khác. Android V22 (Go: [android/body.go](../internal/facebook/register/android/body.go), C# `FacebookRegisterAPIAndroidV2.CreateAccountVariables_v22`) **không nested 4 level URL-encoded**, mà:
  - Build JSON object plain → `escJSON` → embed làm string của level cha.
  - Khâu cuối cùng URL-encode 1 lần (`url.QueryEscape(L0)`).
  - `safetynet_token` thực sự sinh giá trị (không null).
  - `attestation_result` là object thật (keyHash/data/signature), không `{}`.
  - `should_save_password = false` (S23 = true).
  - `family_device_id` = `deviceid` (S23: random `FamilyDeviceID` riêng).
- `friendlyName` và `app_id` **giống nhau** (cùng dùng `com.bloks.www.bloks.caa.reg.create.account.async`).
- Đây là chi tiết quan trọng: **muốn build nền frontend chuyển platform, contract phải nắm các trường overrides này** (sẽ ảnh hưởng UI Flow Settings).

---

## 3. Build HEADER params — phân tích chi tiết

### 3.1 C# header order (S23)
Source: [`docs/05-fb-api-and-endpoints.md §5.3`](file:///D:/NCS/FULL%20REG%20CLONE%20HAVU/docs/05-fb-api-and-endpoints.md) + [`FacebookApiHeaderCollectionBuilder.cs`](file:///D:/NCS/FULL%20REG%20CLONE%20HAVU/VerifyCloneVIP/Models/FacebookApiHeaderCollectionBuilder.cs) L490-546.

```
=== RegisterWIFIHeaderCollection (S23OAuthToken, addsimhni=true) ===
Authorization: OAuth {S23OAuthToken}
X-Fb-Connection-Type: {WIFI | mobile.LTE}
X-Fb-Sim-Hni: {HNI}
X-Fb-Net-Hni: {HNI}

=== FullRegisterHeader() ===
X-Graphql-Client-Library: graphservice
X-Tigon-Is-Retry: False
X-Graphql-Request-Purpose: fetch
X-Fb-Privacy-Context: 3643298472347298
x-fb-request-analytics-tags: {"network_tags":{"product":"350685531728","purpose":"fetch","request_category":"graphql","retry_attempt":"0"},"application_tags":"graphservice"}
x-zero-eh: {random 32 hex}
x-zero-state: unknown
X-Fb-Http-Engine: Tigon/Liger
X-Fb-Client-Ip: True
X-Fb-Server-Cluster: True
X-Fb-Rmd: state=URL_ELIGIBLE
X-Fb-Friendly-Name: FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.create.account.async

=== S23.Register additional (L74-85) ===
X-Fb-Device-Group: {deviceGroup}
App-Scope-Id-Header: {deviceid}
X-Fb-Integrity-Machine-Id: {machineId}     # CHỈ KHI machineId != ""
X-Zero-F-Device-Id: {familyDeviceId}

=== S23-specific overrides (L83-85) ===
X-Meta-Zca: empty_token                    # OVERRIDE _defaultMetaZcaHeaderValue
x-meta-usdid: {ECDSA P-256 sign({uuid}.{ts}) → base64url}
x-fb-conn-uuid-client: {uuid no-dashes}

=== Auto ===
User-Agent: {ua}
Content-Type: application/x-www-form-urlencoded
Content-Length: {len(body)}                # C# Register L99 thêm trước Post
```

### 3.2 Go header order (S23)
File [s23/http.go:194-241](../internal/facebook/register/s23/http.go#L194-L241):

```go
// === RegisterWIFIHeaderCollection ===
Authorization                 OAuth s23OAuthToken
X-Fb-Connection-Type          profile.ConnType
X-Fb-Sim-Hni                  profile.Sim.HNI
X-Fb-Net-Hni                  profile.Sim.HNI

// === FullRegisterHeader ===
X-Graphql-Client-Library      graphservice
X-Tigon-Is-Retry              False
X-Graphql-Request-Purpose     fetch
X-Fb-Privacy-Context          3643298472347298
x-fb-request-analytics-tags   {…network_tags…}
x-zero-eh                     uuid no-dashes
x-zero-state                  unknown
X-Fb-Http-Engine              Tigon/Liger
X-Fb-Client-Ip                True
X-Fb-Server-Cluster           True
X-Fb-Rmd                      state=URL_ELIGIBLE
X-Fb-Friendly-Name            s23FriendlyName

// === Post-collection (S23.Register L74-85) ===
X-Fb-Device-Group             profile.DeviceGroup
App-Scope-Id-Header           profile.DeviceID
[X-Fb-Integrity-Machine-Id    profile.MachineID]          // chỉ khi != ""
X-Zero-F-Device-Id            profile.FamilyDeviceID

// === S23 overrides ===
X-Meta-Zca                    s23MetaZCA  ("empty_token")
x-meta-usdid                  generateUSDID()             // ECDSA P-256, đủ chuẩn
x-fb-conn-uuid-client         connUUID()                  // uuid no-dashes

// === Auto ===
user-agent                    profile.S23UA
content-type                  application/x-www-form-urlencoded
```

→ `Content-Length` Go **không append explicit** trong `buildHeaders` (S23). Phía `session.post` cũng không tự thêm như `android/http.go` đang làm — đây là một **điểm cần audit** (mục 6).

### 3.3 Bảng so sánh từng header (S23)

| Header | C# | Go [s23/http.go](../internal/facebook/register/s23/http.go) | Trạng thái |
|---|---|---|---|
| Authorization | `OAuth {S23OAuthToken}` | `OAuth ` + `s23OAuthToken` | ✅ Khớp |
| X-Fb-Connection-Type | `WIFI` / `mobile.LTE` | profile.ConnType (tương tự) | ✅ Khớp |
| X-Fb-Sim-Hni | `{mcc}{mnc}` | `profile.Sim.HNI` | ✅ Khớp |
| X-Fb-Net-Hni | `{mcc}{mnc}` | `profile.Sim.HNI` | ✅ Khớp |
| X-Fb-Connection-Bandwidth | có (random) | **KHÔNG** | ⚠️ Go thiếu (xem mục 6.1) |
| X-Fb-Connection-Quality | EXCELLENT (logout/xzero) | có ở `buildLogoutHeaders` nhưng KHÔNG ở `buildHeaders` register | ⚠️ Có thể cần |
| X-Fb-Connection-Token | có | **KHÔNG** | ⚠️ Go thiếu |
| X-Graphql-Client-Library | graphservice | graphservice | ✅ |
| X-Tigon-Is-Retry | False | False | ✅ |
| X-Graphql-Request-Purpose | fetch | fetch | ✅ |
| X-Fb-Privacy-Context | 3643298472347298 | 3643298472347298 | ✅ |
| x-fb-request-analytics-tags | JSON | JSON cùng schema | ✅ — nhưng **thứ tự field trong JSON khác Android V3** (xem 3.4) |
| x-zero-eh | random 32 hex | uuid no-dashes (32 hex) | ✅ |
| x-zero-state | unknown | unknown | ✅ |
| X-Fb-Http-Engine | Tigon/Liger | Tigon/Liger | ✅ |
| X-Fb-Client-Ip | True | True | ✅ |
| X-Fb-Server-Cluster | True | True | ✅ |
| X-Fb-Rmd | state=URL_ELIGIBLE | state=URL_ELIGIBLE | ✅ |
| X-Fb-Friendly-Name | FbBloksActionRootQuery-… | `s23FriendlyName` const | ✅ |
| X-Fb-Device-Group | random 4-digit | `profile.DeviceGroup` (= `1000+r.Intn(9000)`) | ✅ |
| App-Scope-Id-Header | DeviceId | profile.DeviceID | ✅ |
| X-Fb-Integrity-Machine-Id | MachineId nếu != "" | có guard `if profile.MachineID != ""` | ✅ |
| X-Zero-F-Device-Id | FamilyDeviceId (S23) **/** DeviceId (V3 Android) | `profile.FamilyDeviceID` ở S23 / `profile.DeviceID` ở Android | ✅ Đúng phân biệt |
| X-Meta-Zca | `empty_token` (S23 reg)<br>base64 blob (logout/xzero) | `s23MetaZCA = "empty_token"` register, `defaultMetaZcaBlob` ở logout/xzero | ✅ Đúng phân biệt |
| x-meta-usdid | `{uuid}.{ts}.{base64url ECDSA sig}` | `generateUSDID()` (ECDSA P-256 thật) | ✅ Khớp format, **Go ECDSA sign thực**; Android V3 (V22) Go giả lập DER 69 bytes random — xem 3.5 |
| x-fb-conn-uuid-client | uuid no-dashes (S23) **/** 16 random byte base64 (V3 Android) | S23: `uuid no-dashes`. Android V3: `generateConnUUIDClientV3` (16 bytes base64) | ✅ Đúng phân biệt |
| User-Agent | `[FBAN/FB4A;…]` (có thể thêm `Dalvik/...` prefix) | `profile.S23UA` (xem 3.6) | ✅ |
| Content-Length | C# auto-add trước Post | Go S23: **không add explicit**. Go Android V22: thêm sau User-Agent, có HeaderOrderKey | ⚠️ Cần review (xem 6.1) |
| Content-Type | application/x-www-form-urlencoded | application/x-www-form-urlencoded | ✅ |

### 3.4 Khác biệt JSON `x-fb-request-analytics-tags`
- **S23 (C# + Go):** `{"network_tags":{"product":"350685531728","purpose":"fetch","request_category":"graphql","retry_attempt":"0"},"application_tags":"graphservice"}`
- **Android V3/V22 (C# RegisterWIFIHeaderCollectionV3 + Go [android/http.go:222](../internal/facebook/register/android/http.go#L222)):** `{"network_tags":{"product":"350685531728","request_category":"graphql","purpose":"fetch","retry_attempt":"0"},"application_tags":"graphservice"}`
  → V3 đảo thứ tự `request_category` và `purpose`. Đây là **fingerprint differentiator** giữa 2 schema. Go đang đúng trong cả 2 file.

### 3.5 `x-meta-usdid` — ECDSA P-256
| | C# | Go S23 | Go Android V3 |
|---|---|---|---|
| Method | ECDSA sign byte payload, key generate per session | `ecdsa.SignASN1` thật (P-256, SHA-256) | random bytes 69-byte DER format (0x30 0x45 0x02 0x21 0x00 ... 0x02 0x20 ...) |
| Output format | `{uuid}.{ts}.{base64url(sig)}` | `{uuid}.{ts}.{base64url RawURL(sig)}` | `{uuid}.{ts}.{base64 std no-pad → +→- /→_}` |
| Tradeoff | Đúng nhất | Đúng nhất | "Fake-it" — không dùng key thật, vẫn pass server vì server không verify |

→ **S23 chuẩn hơn Android V3** trong code Go hiện tại. Nếu FB siết verify usdid sau này → Android V3 sẽ chết trước S23.

### 3.6 User-Agent S23
Format từ [s23/profile.go:73-78](../internal/facebook/register/s23/profile.go#L73-L78):
```
[FBAN/FB4A;FBAV/{appVer};FBBV/{buildNum};FBDM/{density={d},width={w},height={h}};
 FBLC/{locale};FBRV/0;FBCR/{carrier};FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;
 FBDV/{model};FBSV/15;FBOP/1;FBCA/arm64-v8a:;]
```
`appVer` chọn random từ `s23AppVersions` (7 phiên bản từ 550 → 556, có cả `554.0.0.57.70|918990560` và `556.1.0.63.64|942217461` được capture từ traffic thật — xem [s23/body.go:33-44](../internal/facebook/register/s23/body.go#L33-L44)).

So với C# `AndroidUserAgentBuilder` (docs §12.2): **format y hệt** nhưng:
- Go pin `FBSV/15` (Android 15), C# random từ `device_versions.txt`.
- Go pin `FBCA/arm64-v8a:` (cho arm64), C# random từ `device_cores.txt` (`armeabi-v7a`).
- Go có dấu `:` cuối `FBCA/arm64-v8a:` — match traffic thật S23.
- Go **không** support `Dalvik/2.1.0 (...)` prefix mặc định; có method `SetUAOptions(addVirtualSpecs, useBuildNumFile)` **nhưng** logic strip prefix khi `addVirtualSpecs=false` — ngược chiều: trong Go, UA mặc định **không** có Dalvik, addVirtualSpecs=true (default) là no-op, false thì strip — do `BuildProfileForPlatform` không build Dalvik prefix nên tham số `addVirtualSpecs` của S23 **gần như chết** (không có gì để strip).

→ **Go cần xem lại UA option API ở S23** nếu muốn đồng bộ với hành vi C# `AndroidUserAgentBuilder`.

### 3.7 Header case-sensitivity (HTTP/2 HPACK)
- HTTP/2 đặc tả tất cả header tên là lowercase. Nếu tls-client gửi `Authorization` (Pascal) thì underlying bogdanfinn/fhttp tự lowercase trước khi HPACK encode → server vẫn nhận đúng.
- **Tuy nhiên** thứ tự header (`HeaderOrderKey`) **CRITICAL** vì FB fingerprint cả thứ tự HPACK.
- Go S23 [s23/http.go:73-76](../internal/facebook/register/s23/http.go#L73-L76) **có** `req.Header[fhttp.HeaderOrderKey] = order` → tốt.
- Go Android [android/http.go:64-81](../internal/facebook/register/android/http.go#L64-L81) cũng có `headerOrder` + `HeaderOrderKey` + thêm `Content-Length` ngay trước `User-Agent` (match C# behavior).
- C# .NET HttpWebRequest có behavior riêng: tự re-order vài header (e.g. `Host`, `Content-Length` luôn cuối) — Go đang cố mô phỏng. **Đây là điểm dễ drift nhất** giữa 2 stack.

---

## 4. Warm session (datr mồi) & cookie pool

### 4.1 C# `RegisterWithRequestInitialAccount`
- File: `FacebookRegisterMfbRequest.cs`.
- Trigger: `MachineId.Contains("|") && isFirst` (= datr là full line `uid|pass|cookie|token`).
- Flow: GET m.facebook.com → parse CSRF → login (new UI bloks → fallback old UI legacy form) → wait → logout. Tất cả qua **cùng `IHttpRequestClient`** (TLS fingerprint giữ nguyên).
- Dùng để "đốt" cookie datr cho FB nhận diện → tăng success rate khi reg.

### 4.2 Go `warmSession`
- File: [s23/extras.go:389-441](../internal/facebook/register/s23/extras.go#L389-L441).
- **Tách riêng web client** (`fhttp.Client` plain HTTP/1.1, không TLS fingerprint Android) — vì warm là m.facebook.com (web Samsung Browser UA), khác hẳn API S23.
- Best-effort: timeout 12s, fail im → reg vẫn proceed.
- Sau warm: copy cookies (`datr`, `sb`, `fr`, ...) từ web jar sang S23 tls-client jar (skip `c_user`, `xs`, `locale`).
- Logic login: `loginWarmNewUI` (wbloks CAA) → fallback `loginWarmOldUI` (legacy form).
- Khác C#: Go **không** giữ TLS fingerprint chung warm vs reg — không vấn đề vì warm là web, reg là native API → 2 fingerprint khác nhau là chính xác hơn C#.

### 4.3 Datr/Cookie pool
| | C# | Go |
|---|---|---|
| Pool struct | `FacebookLogoutSessionUtils.GetPerfectMachineId(method, limit)` | `PartitionedDatrPool` (per-slot queue) |
| Source file | `cookie_initial.txt` | Pool nội bộ + load từ `Config/datr_pool.json` |
| Tracking | `IncrementMachineIdUsage(machineId)` (count + limit per machine) | `IncrementUsage` + `RecordResult` (success/fail/unknown) |
| Modes | source_type 1 (datr only) / 3 (full line) | `SeedModeDatrOnly / FullCookie / InitialAccount` ([profile.go:138-146](../internal/facebook/register/s23/profile.go#L138-L146)) |
| Inject | C#: chỉ qua header `X-Fb-Integrity-Machine-Id` + body `reg_info.machine_id`. Block cookie `datr` đã comment-out. | Y hệt — Go [register.go:301-306](../internal/facebook/register/s23/register.go#L301-L306) viết comment chỉ rõ "KHÔNG inject datr thành cookie HTTP — match C# block đã comment out". |
| Warm gate | Mfb mới có `RegisterWithRequestInitialAccount` | Go gate `if input.CookieInitialMethod == "ck"` — modes khác (file/new) downgrade về `FullCookie` hoặc `DatrOnly`. |

→ Go **đã hiểu** điểm tinh tế "không double-signal datr" của FB và bám đúng C# rev mới nhất.

---

## 5. Post-register side effects

### 5.1 X-Zero-Eh fetch (sau reg success)
| | C# | Go |
|---|---|---|
| Endpoint | `POST b-graph.facebook.com/?include_headers=false&decode_body_json=false&streamable_json_response=true&locale=...&client_country_code=...` | Y hệt ([extras.go:638-641](../internal/facebook/register/s23/extras.go#L638-L641)) |
| Body | `batch=[{"method":"POST","body":"carrier_mcc=...","name":"fetchZeroToken","relative_url":"mobile_zero_campaign"}]` + `fb_api_caller_class=Fb4aAuthHandler` + `fb_api_req_friendly_name=fetchLoginData-batch` | Y hệt ([extras.go:583-609](../internal/facebook/register/s23/extras.go#L583-L609)). Encode bằng `urlEncodeFull` mô phỏng `WebUtility.UrlEncode`. |
| Headers extras | `X-Zero-F-Device-Id` + `App-Scope-Id-Header` + `X-Zero-State=unknown` + `X-Meta-Zca=base64Blob` | Y hệt ([extras.go:672-700](../internal/facebook/register/s23/extras.go#L672-L700)) |
| Khoảng delay sau reg | Sleep random 1-2s | `1000+rand.Intn(1000)` ms ([register.go:397](../internal/facebook/register/s23/register.go#L397)) ✅ |
| Parse | regex `eligibility_hash":"(.*?)"` | Y hệt ([extras.go:581](../internal/facebook/register/s23/extras.go#L581)) |

### 5.2 Logout (optional)
- C# Automation gọi `LogoutAccount` khi `MainFormUISettings.VerifyAfterReg == false`.
- Go: hàm `LogoutAccount` có sẵn ([extras.go:498-536](../internal/facebook/register/s23/extras.go#L498-L536)), nhưng **không tự gọi trong `Register`** — caller (Automation/runner) tự quyết định, match C# pattern.

### 5.3 Response parsing
- Cả 2 dùng regex thuần (không parse JSON đầy đủ vì FB hay đổi schema).
- Go có thêm fallback patterns cho UID:
  - `reCUser`: `c_user","value":"(\d{10,})"`  (chuẩn)
  - `reBloksUID`: `currentUser["\s:,]+(\d{10,})`  (S23 bloks)
  - `reCreatedUID`: `created_user(?:id)?["\s:,\\]+(\d{10,})`
  - `reSaveCredUID`: `SaveCredential[^}]*?(\d{10,18})`
- **Tốt hơn C# (chỉ dùng 1 pattern)** → bắt được nhiều biến thể response của FB.

---

## 6. Điểm cần audit / khác biệt cần cân nhắc

### 6.1 Headers thiếu so với C# header collection
- Go S23 `buildHeaders` thiếu:
  - `X-Fb-Connection-Bandwidth: {random}` — C# có (V1 docs L108).
  - `X-Fb-Connection-Quality: EXCELLENT` — C# có (docs L109).
  - `X-Fb-Connection-Token: {fb_conn_token}` — C# có (docs L116).
- Go S23 `buildHeaders` không explicit thêm `Content-Length` — phụ thuộc tls-client tự thêm. Nếu không có → POST vẫn OK với HTTP/2 (vì `:length` pseudo header không tồn tại; HTTP/2 dùng DATA frames length thật) — nhưng C# header order trong fingerprint có thể khác.

→ **Đề xuất**: thêm 3 header trên vào `buildHeaders` để fingerprint sát hơn nếu success rate đang giảm.

### 6.2 JSON `flow_info` value
- C# (S23): `"{\"flow_name\":\"new_to_family_fb_default\",\"flow_type\":\"ntf\"}"` (escape 1 lần vào level-3).
- Go S23 [body.go:464](../internal/facebook/register/s23/body.go#L464): hardcoded chuỗi escape sẵn `%5C%5C%5C%5C%5C%5C%5C%22flow_name%5C%5C%5C...` — match nếu tính đúng escape level. **Đã verify trong test [body_test.go](../internal/facebook/register/s23/body_test.go)**.

### 6.3 `gzip` flag
- C# S23 KHÔNG gzip body (form-urlencoded plain).
- Go có hàm `postGzip` nhưng [s23/http.go:98-102](../internal/facebook/register/s23/http.go#L98-L102) **đã neutralize** → gọi `s.post` thường. Comment đúng. Caller cứ gọi `postGzip` cho thuận, không sai.

### 6.4 ConnType "WIFI" vs "mobile.LTE"
- C# random: WIFI nhiều hơn LTE.
- Go [s23/profile.go:93-96](../internal/facebook/register/s23/profile.go#L93-L96): 1/3 xác suất `mobile.LTE`, còn lại WIFI → tỉ lệ 33% LTE.
- **Cân nhắc** điều chỉnh nếu user thấy LTE thường bị FB nghi hơn WIFI.

### 6.5 `DeviceGroup` random range
- C# `device_group` lấy từ file config (có thể là alpha số).
- Go: `1000 + rand(9000)` → 4 chữ số. Có khả năng hẹp pool. **Cần verify** giá trị thực FB chấp nhận.

### 6.6 `BloksVer` đã update?
- C# const `S23BloksVer = "d90663010f8c230bedf28906f2bac9c1d1f532a275373050778e36e76a7cb999"` (capture cũ).
- Go [s23/body.go:48](../internal/facebook/register/s23/body.go#L48) đã update sang `385fe019aa6b5903bdad3a4799063e3fc70da9cd1fda8b54189bce078c701665` (mới).
- → Go đi trước C# — có thể match traffic capture mới hơn.

### 6.7 `s23DocID` thay đổi
- C# const `1199408042526631289603660492`.
- Go [s23/body.go:47](../internal/facebook/register/s23/body.go#L47): `11994080422955588194694478490` — **khác C#** (29 chữ số, có thể là misprint khi update).
- ⚠️ **Audit**: confirm doc_id mới. Nếu sai → server trả `client_doc_id mismatch` → reg fail.

### 6.8 `purpose` duplicate
- Form-urlencoded outer có `purpose=fetch`.
- Header `X-Graphql-Request-Purpose: fetch`.
- JSON analytics tag `network_tags.purpose: fetch`.
- C# pre-existing docs có 1 chỗ (V1) gọi `prefetch` thay vì `fetch` ở header (docs §5.3 L122). → Cần kiểm chứng version nào FB đang dùng.

---

## 7. Ánh xạ class C# ↔ package Go

| C# Class | Mục tiêu | Go package |
|---|---|---|
| `FacebookRegisterAPIAndroidS23` | S23/S22/S24/S25/S26 register | [internal/facebook/register/s23](../internal/facebook/register/s23) |
| `FacebookRegisterAPIAndroidV2` | Generic Android (V22 schema) | [internal/facebook/register/android](../internal/facebook/register/android) |
| `FacebookRegisterAPIToken` | Token-only API path | (chưa có Go equivalent rõ ràng — kiểm tra [register/web](../internal/facebook/register/web)) |
| `FacebookRegisterMfbRequest` | m.facebook.com keep session | một phần warmSession của S23 + có thể package web |
| `FacebookRegisterMfbChrome` | Selenium browser | Không có Go equiv (out of scope) |
| `FacebookApiHeaderCollectionBuilder` | Helper static build header | Mỗi platform có `buildHeaders()` riêng trong `http.go` |
| `FacebookApiFormDataBuilder` | Helper static build body | Mỗi platform có `buildRegisterBody()` trong `body.go` |
| `FacebookLogoutSessionUtils` | Datr pool + warm | [register/android/pool.go](../internal/facebook/register/android/pool.go) (`PartitionedDatrPool`) |
| `IUserAgentBuilder` (3 impl) | Build UA | Inline trong `BuildProfileForPlatform` (chưa abstract — cân nhắc tạo `internal/facebook/fakeinfo/uabuilder` nếu sau này muốn config-driven) |
| `FacebookAccountModel` | Profile struct + InstanceRandom | `fakeinfo.FullRegProfile` + `fakeinfo.BuildFullRegProfile(country)` |

---

## 8. Đề xuất hành động (cho phase rebuild frontend)

> Hiện tại nhiệm vụ là dựng frontend foundation — backend đang stub. Phần này note để nhóm backend tham chiếu khi viết Go bridge contracts.

1. **Bridge contract phải expose đủ 4 trường settings ảnh hưởng Register**: `Platform`, `CookieInitialMethod`, `AddVirtualSpecAndroid`, `UseRawUa`. Đây là 4 lever thay đổi build header/body.
2. **Stats panel** nên có cột `S/F/U` per datr (success/fail/unknown) — Go pool đã track, frontend chỉ cần hiển thị qua bridge.
3. **Flow Settings page** nên cho user chọn:
   - Platform (s22/s23/s24/s25/s26 / android-V22 / web).
   - ConnType (WIFI / mobile.LTE / random 67/33).
   - addVirtualSpecs (toggle Dalvik prefix — hiện S23 chưa hỗ trợ, cần BE bổ sung).
   - useBuildNumFile (toggle).
4. **View Settings page** show realtime: device model picked, UA, datr pool stats, x-zero-eh hash mới nhất. Hỗ trợ debug field-by-field.
5. **Account detail dialog** show full request/response của reg call — tốt cho audit `client_doc_id` hay `bloks_versioning_id` mismatch.

---

## 9. Tham chiếu file

- C# (FULL REG CLONE HAVU):
  - [`docs/04-automation-orchestrators.md`](file:///D:/NCS/FULL%20REG%20CLONE%20HAVU/docs/04-automation-orchestrators.md) §4.1 — flow Register.
  - [`docs/05-fb-api-and-endpoints.md`](file:///D:/NCS/FULL%20REG%20CLONE%20HAVU/docs/05-fb-api-and-endpoints.md) — endpoints + headers.
  - [`docs/12-fake-info-builders.md`](file:///D:/NCS/FULL%20REG%20CLONE%20HAVU/docs/12-fake-info-builders.md) — UA builder.
  - [`docs/18-regandverify-line-by-line.md`](file:///D:/NCS/FULL%20REG%20CLONE%20HAVU/docs/18-regandverify-line-by-line.md) §18.6 — register inner loop.
- Go (HVR):
  - [internal/facebook/register/s23/register.go](../internal/facebook/register/s23/register.go) — entry + WorkerContext.
  - [internal/facebook/register/s23/body.go](../internal/facebook/register/s23/body.go) — body builder (177 fields + 4-level nest).
  - [internal/facebook/register/s23/http.go](../internal/facebook/register/s23/http.go) — headers + USDID + tls session.
  - [internal/facebook/register/s23/profile.go](../internal/facebook/register/s23/profile.go) — UA + Seed parser + pool wiring.
  - [internal/facebook/register/s23/extras.go](../internal/facebook/register/s23/extras.go) — warm session + logout + xzero + response parser.
  - [internal/facebook/register/android/body.go](../internal/facebook/register/android/body.go) — V22 schema (khác S23).
  - [internal/facebook/register/android/http.go](../internal/facebook/register/android/http.go) — V3 headers (analytics tag thứ tự khác).
  - [docs/REG_S23_SPEC.md](REG_S23_SPEC.md) — spec port S23 hiện tại.
