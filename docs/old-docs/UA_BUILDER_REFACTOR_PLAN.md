# PLAN — Thống nhất build User-Agent + tách Auth Source ra tab riêng

> **Mục đích**: chỉ là plan + spec. Chưa code. User review xong mới làm.
> **Liên quan**: [REGISTER_LOGIC_COMPARISON.md](REGISTER_LOGIC_COMPARISON.md) §3.6 đã flag UA mess; file này detail hoá cách fix.
> **Phạm vi**: áp dụng cho TẤT CẢ register (s22/s23/s24/s25/s26/s555/s556/s557/android/web/webandroid/ioshttp) và toàn bộ verify sẽ thêm sau.

---

## A. Hiểu đúng C# về 2 toggle

### A.1 `addVirtualSpecAndroid` (= `add_virtual_specs`)
**Hành vi C#** ([`AndroidUserAgentBuilder.cs:103-110`](file:///D:/NCS/FULL%20REG%20CLONE%20HAVU/VerifyCloneVIP/FakeInfoBuilder/AndroidUserAgentBuilder.cs)):

```csharp
if (add_virtual_specs) {
    FullUserAgent = $"Dalvik/2.1.0 (Linux; U; Android {rand_osVersion}; {Model} Build/{rand_BuildId}) " + FB_UG;
} else {
    FullUserAgent = FB_UG;
}
```

→ Toggle này **prepend chuỗi `Dalvik/2.1.0 (Linux; U; Android {os}; {Model} Build/{build}) `** vào trước UA `[FBAN/FB4A;...]`.
→ `{os}` phải MATCH `FBSV/{os}` trong FB_UG.
→ `{Model}` phải MATCH `FBDV/{Model}` trong FB_UG.
→ `{build}` lấy từ A.2 (depend trên `useBuildNumFile`).

### A.2 `usingBuildNumFile` (= `use_buildnum_file`)
**Hành vi C#** ([`AndroidUserAgentBuilder.cs:55-64`](file:///D:/NCS/FULL%20REG%20CLONE%20HAVU/VerifyCloneVIP/FakeInfoBuilder/AndroidUserAgentBuilder.cs)):

```csharp
string rand_BuildId = $"{Brand}-{Model}";   // default e.g. "Samsung-SM-J110G"
if (use_buildnum_file) {
    rand_BuildId = FileOperationsUtils.ReadAsList(PathSingleton.Device_Build_Nums_Path).RandomItemInList();
    // = random line từ device_build_nums.txt e.g. "SQ1A.220105.002"
}
```

→ Toggle này **chỉ thay `{build}` slot trong Dalvik prefix** từ `"{Brand}-{Model}"` thành 1 dòng random của `config/device_info/device_build_nums.txt`.
→ **Trong `AndroidUserAgentBuilder`**: `useBuildNumFile` **CHỈ có hiệu lực khi `addVirtualSpecs=true`**. Nếu `addVirtualSpecs=false` thì `useBuildNumFile` BẬT/TẮT đều như nhau (vì không có Dalvik prefix để chèn build number).
→ **Trong `BrowserAndroidUserAgentBuilder`** ([`BrowserAndroidUserAgentBuilder.cs:43-53`](file:///D:/NCS/FULL%20REG%20CLONE%20HAVU/VerifyCloneVIP/FakeInfoBuilder/BrowserAndroidUserAgentBuilder.cs)): `useBuildNumFile` luôn có hiệu lực vì nó replace slot `{rand_BuildId}` ngay trong template Mozilla `Mozilla/5.0 (Linux; Android {os}; {build}) AppleWebKit/...`.

### A.3 Diễn giải lại câu user nói
> "khi tích vào usingBuildNumFile thì thay vì sử dụng UA trong list thì nó sẽ build theo các thông số của `config/device_info`"

**Ngữ nghĩa kỹ thuật chính xác** (theo C# code):
- "UA trong list" thực ra ám chỉ `ConfigFileUserAgentBuilder` đọc full UA string từ `config/useragent/Android_UG.txt`. Việc chọn class builder này là **constructor param `useragent_type`** của mỗi class register/verify (`0=Android builder, 1=ConfigFile builder, 2=Browser builder`), KHÔNG phải toggle `usingBuildNumFile`.
- `usingBuildNumFile` chỉ thay slot `{build}` trong Dalvik prefix (Android) hoặc Mozilla `(Linux; Android ...; {build})` (Browser).
- Cụm 6 file `config/device_info/*.txt` được dùng **mặc định** mỗi lần build UA (densitis, device_cores, devices_versions, screen_resolution, carriers + mobile_devices + fbapp_inf), không liên quan toggle `usingBuildNumFile`.

→ **Hai toggle là 2 dimension khác nhau**, có thể combine 4 trạng thái:

| addVirtualSpecs | useBuildNumFile | Output (Android builder) |
|---|---|---|
| false | false | `[FBAN/FB4A;…]` (chỉ FB_UG) |
| false | true  | `[FBAN/FB4A;…]` (chỉ FB_UG) — **build_num bị ignore** |
| true  | false | `Dalvik/2.1.0 (Linux; U; Android 9; SM-J110G Build/Samsung-SM-J110G) [FBAN/FB4A;…]` |
| true  | true  | `Dalvik/2.1.0 (Linux; U; Android 9; SM-J110G Build/SQ1A.220105.002) [FBAN/FB4A;…]` |

→ **UI cần disable / dim "Dùng file build number" khi "Thêm virtual spec Android" tắt** (vì không có hiệu lực) — sẽ giúp user không nhầm.
→ **TRỪ KHI** ta đang chọn API REG = WebAndroid/ChromeAndroid (Browser builder) thì `useBuildNumFile` luôn có hiệu lực bất kể `addVirtualSpecs`.

---

## B. Inventory mess trong Go HVR hiện tại

### B.1 Duplicate UA template ở 4 platform
Cùng 1 chuỗi format string xuất hiện 4 lần (s23/s555/s556/s557):

```go
"[FBAN/FB4A;FBAV/%s;FBBV/%s;FBDM/{density=%s,width=%d,height=%d};FBLC/%s;FBRV/0;FBCR/%s;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/%s;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]"
```

**Sai sót**:
- ❌ `FBDM/{density=...}` — dấu `/` thừa. C# code mới ([`AndroidUserAgentBuilder.cs:90`](file:///D:/NCS/FULL%20REG%20CLONE%20HAVU/VerifyCloneVIP/FakeInfoBuilder/AndroidUserAgentBuilder.cs)) gen `FBDM={density=...}` (KHÔNG slash). User example xác nhận `FBDM={density=3.5,...}`.
- ❌ `FBMF/samsung;FBBD/samsung` — hardcode lowercase. C# đọc từ `mobile_devices/android_devices.txt` line như `Samsung:Samsung:CT1001` → `Samsung` capital.
- ❌ `FBSV/15` — pin Android 15. C# random từ `devices_versions.txt` (`9,10,11,12,13`).
- ❌ `FBCA/arm64-v8a:` — pin 1 arch. C# random từ `device_cores.txt`. User example dùng `armeabi-v7a:armeabi`.

### B.2 Sai semantics `SetUAOptions`
`s23/register.go:127-137`, `s555/register.go:74`, `s556/register.go:74`, `s557/register.go:75`:

```go
// SetUAOptions rebuild S23 UA với addVirtualSpecs + useBuildNumFile.
func (w *WorkerContext) SetUAOptions(addVirtualSpecs, useBuildNumFile bool) {
    if w == nil || addVirtualSpecs {
        return // default behavior — giữ nguyên S23UA đã build
    }
    if idx := indexOf(w.profile.S23UA, "[FBAN/FB4A"); idx > 0 {
        w.profile.S23UA = w.profile.S23UA[idx:]
    }
}
```

**Sai chiều logic**:
- C# **mặc định KHÔNG có** Dalvik prefix; toggle `addVirtualSpecs=true` mới THÊM vào.
- Go **mặc định cũng KHÔNG có** prefix (template ở B.1 không sinh Dalvik) → strip cũng vô nghĩa.
- Kết quả: `addVirtualSpecs=true` → no-op (đáng lẽ phải prepend Dalvik). `addVirtualSpecs=false` → strip không có gì để strip.
- `useBuildNumFile` **bị ignore hoàn toàn** ở mọi platform.

### B.3 Không có file-driven config
C# đọc 8 file:

| Path | Mục đích | Go hiện tại |
|---|---|---|
| `config/device_info/densitis.txt` | density (3.5/3.0/4.0) | ❌ hardcode trong `s23Devices.Density` |
| `config/device_info/device_cores.txt` | CPU arch | ❌ pin `arm64-v8a:` trong template |
| `config/device_info/devices_versions.txt` | OS version | ❌ pin `15` trong template |
| `config/device_info/screen_resolution.txt` | width × height | ❌ hardcode trong `s23Devices.Width/Height` |
| `config/device_info/carriers.txt` | FBCR | ⚠️ có nhưng nguồn khác (sim) |
| `config/device_info/device_build_nums.txt` | Dalvik build ID | ❌ chưa support |
| `config/mobile_devices/android_devices.txt` | model:brand:manufacturer | ❌ hardcode `s23Devices` 6 entries |
| `config/fbapp_inf/versions_and_builds.txt` | FBAV|FBBV | ❌ hardcode `s23AppVersions` 7 entries |

→ User update 1 device mới phải SỬA CODE Go. C# chỉ cần edit txt → reload.

### B.4 Web/iOS UA hardcode
- [`web/android_token.go:33`](../internal/facebook/register/web/android_token.go) — 1 chuỗi UA cứng.
- [`webandroid/register.go`](../internal/facebook/register/webandroid/register.go) — UA pin trong `ChromeAndroidProfile`, không support 2 toggle.
- [`ioshttp/register.go`](../internal/facebook/register/ioshttp/register.go) — UA Safari iOS pin.

→ 3 platform này bỏ qua hoàn toàn 2 toggle. C# `BrowserAndroidUserAgentBuilder` ít nhất honor `useBuildNumFile`. Go không.

### B.5 UI mess
[`InteractionSetupPage.vue:796-818`](../frontend/src/pages/InteractionSetupPage.vue):
- 2 checkbox `usingBuildNumFile` + `addVirtualSpecAndroid` đang nằm cùng row với `Phone × Mail` mode (random-normal/random-file/fm-phone) — **sai context**.
- Không có nhóm "User Agent" rõ ràng.
- Không có hint disable khi `addVirtualSpecAndroid=false`.

[`InteractionSetupPage.vue:867-960+`](../frontend/src/pages/InteractionSetupPage.vue):
- "Nguồn xác thực" (Mail/Phone) đang là **shared footer** cố định ở dưới cùng page → muốn truy cập phải scroll dài.
- User muốn tách thành **tab riêng** ngang hàng với Accounts/Flow Settings/Proxy Settings/View Settings.

---

## C. Refactor đề xuất (Backend Go)

### C.1 Tạo package mới `internal/facebook/fakeinfo/uabuilder`

```
internal/facebook/fakeinfo/uabuilder/
  builder.go        // interface UABuilder + UAOptions struct
  android.go        // AndroidUABuilder (port AndroidUserAgentBuilder.cs)
  browser.go        // BrowserAndroidUABuilder (port BrowserAndroidUserAgentBuilder.cs)
  configfile.go     // ConfigFileUABuilder (đọc 1 UA từ file list)
  ios.go            // iOSUABuilder (cho ioshttp; chuẩn lại fingerprint Safari)
  registry.go       // map platform string → factory(useragent_type) → UABuilder
  data_loader.go    // loader các file txt (lazy + reload khi mtime đổi)
  data_loader_test.go
```

**Interface**:
```go
type UAOptions struct {
    Locale             string  // "en_US"
    AddVirtualSpecs    bool    // → Dalvik/2.1.0 prefix
    UseBuildNumFile    bool    // → Dalvik Build/{random từ file}
    SimBrand           string  // FBCR override (từ SIM); "" = random từ carriers.txt
    Platform           string  // "s22"/"s23"/.../"android" — chỉ device_pool filter
    AppVersion, Build  string  // "" = random; pin nếu user muốn force version cụ thể
}

type UABuildResult struct {
    UserAgent      string  // chuỗi cuối (có/không Dalvik prefix)
    Model          string  // device model (FBDV) — caller cần cho `AndroidDeviceModel`
    OSVersion      string  // (FBSV) — caller cần cho `AndroidOsVersion`
    Manufacturer   string
    Brand          string
}

type UABuilder interface {
    Build(opts UAOptions) (UABuildResult, error)
    Kind() string  // "android-app" | "browser-android" | "config-file" | "ios-safari"
}
```

**Factory** (per-class kiểu C#):
```go
// uaBuilderType: 0=Android, 1=ConfigFile (Android_UG.txt), 2=Browser (cho Token/WebAndroid)
func ResolveBuilder(platform string, uaBuilderType int) UABuilder
```

### C.2 Layout config files (mirror C#)
**Tạo `Config/device_info/`** (đặt cạnh thư mục `Config/` hiện có) chứa:
```
Config/
  device_info/
    densitis.txt           # 3.5\n3.0\n4.0
    device_cores.txt       # arm64-v8a\narmeabi-v7a\narmeabi-v7a:armeabi
    devices_versions.txt   # 9\n10\n11\n12\n13\n14\n15
    screen_resolution.txt  # 1080x2340\n1440x3120\n...
    carriers.txt           # AT&T\nVerizon\n...
    device_build_nums.txt  # SKQ1.210908.001\nSQ1A.220105.002\n...
  mobile_devices/
    android_devices.txt    # Samsung:Samsung:SM-S911B\nSamsung:Samsung:SM-J110G\n...
    s23_devices.txt        # filter của android_devices.txt khi platform=s23
    s24_devices.txt
    ...
  fbapp_inf/
    versions_and_builds.txt           # 558.0.0.70.72|953309395\n...
    versions_and_builds_s22.txt       # phiên bản phù hợp S22
    versions_and_builds_s23.txt       # ...
    ...
  useragent/
    Android_UG.txt   # full UA pre-baked (cho ConfigFile builder)
    Token_UG.txt
    Request_UG.txt
    iOS_UG.txt
    Chrome_Versions.txt
```

→ Copy file gốc từ `D:/NCS/FULL REG CLONE HAVU/VerifyCloneVIP/bin/Debug/config/` qua `d:/NCS/HVR/Config/`.
→ Dữ liệu hiện trong `internal/facebook/fakeinfo/data/ua_android_pool.txt` migrate sang `Config/useragent/Android_UG.txt`.

### C.3 Per-platform refactor
Mỗi `register/<platform>/profile.go` xoá UA-build inline, gọi central:

```go
// s23/profile.go — sau refactor
func BuildProfileForPlatform(platform, countryCode string, uaOpts uabuilder.UAOptions) S23Profile {
    base := fakeinfo.BuildFullRegProfile(countryCode)
    builder := uabuilder.ResolveBuilder(platform, uaBuilderTypeFromInput)
    res, _ := builder.Build(uaOpts)
    base.UserAgent = res.UserAgent
    base.Device.Model     = res.Model
    base.Device.OSVersion = res.OSVersion
    // …
    return S23Profile{FullRegProfile: base, S23UA: res.UserAgent, …}
}
```

Xoá hoàn toàn `SetUAOptions` cũ (sai chiều) ở mỗi platform.
Giữ `SetUA(ua string)` để override raw (UseRawUa).

### C.4 Truyền 2 toggle qua RegInput
Thêm vào [`facebook.RegInput`](../internal/facebook/types.go):
```go
type RegInput struct {
    // …existing
    AddVirtualSpecs bool   // forward từ settings.UA.AddVirtualSpecAndroid
    UseBuildNumFile bool   // forward từ settings.UA.UsingBuildNumFile
    UseRawUa        bool   // bypass builder, dùng UserAgent input thẳng
    UABuilderType   int    // 0=AndroidApp / 1=ConfigFile / 2=Browser (tương ứng useragent_type C#)
}
```
→ Runner đọc settings.json, nhồi vào RegInput cho mọi reg call.

---

## D. Refactor đề xuất (Frontend Vue)

### D.1 Group "User Agent" trong section "Reg account" (InteractionSetupPage)
- Tạo subsection mới `<div class="rp-subsection rp-subsection--ua">` đặt **NGAY DƯỚI** `API REG + Delay reg` (tức trước "Reg Settings").
- Field bên trong:
  ```
  ┌─ User Agent ──────────────────────────────────────────────────┐
  │ Nguồn UA:    ( ) Build từ device_info  (mặc định)             │
  │              ( ) Đọc từ file Android_UG.txt                   │
  │              ( ) Browser Chrome (cho Token/WebAndroid)        │
  │              ( ) Raw UA (paste tay)                           │
  │                                                                │
  │ [✓] Thêm virtual spec Android (Dalvik prefix)                 │
  │ [ ] Dùng file build number   ← disabled khi (^) tắt           │
  │ Raw UA: [_________________________________] (chỉ enable khi   │
  │                                              chọn "Raw UA")   │
  │                                                                │
  │ Preview UA hiện tại:                                           │
  │ [FBAN/FB4A;FBAV/558.0.0.70.72;FBBV/953309395;FBDM={density=… │
  └────────────────────────────────────────────────────────────────┘
  ```
- Bỏ 2 checkbox `usingBuildNumFile` + `addVirtualSpecAndroid` ra khỏi `rp-checks-row` của Phone × Mail.
- Thêm `<FieldHelp field="addVirtualSpecAndroid" />` + `<FieldHelp field="usingBuildNumFile" />` giải thích semantics A.1/A.2.

### D.2 Tab "Nguồn xác thực" mới (top-level)
- **Sidebar route mới**: `/auth-source` (label "Nguồn xác thực", icon mail).
- Tạo `pages/AuthSourcePage.vue` + `modules/auth-source/` (mirror cấu trúc accounts module).
- Move toàn bộ block `<div class="rp-shared-footer">` (lines ~867-960+ của InteractionSetupPage) sang page mới:
  - 2 tab Mail / Phone (giữ logic `authSourceTab`).
  - Mail panel: `mailCategory` (Temp/Rent), provider dropdown, domain, check Live/Die.
  - Phone panel: provider chọn SMS service.
- InteractionSetupPage **xoá hết** auth source footer.
- Settings store split: `useMailProviderStock` move từ InteractionSetupPage sang AuthSourcePage; reactive form share qua Pinia (`useAuthSourceStore`).

### D.3 Cập nhật router + sidebar
[`frontend/src/app/`](../frontend/src/app/) (assume routes ở đây): thêm route mới + nav item.

### D.4 Bridge contract bổ sung
[`frontend/src/bridge/contracts.ts`](../frontend/src/bridge/contracts.ts):
```ts
export interface UASettings {
  builderType: 'android-app' | 'config-file' | 'browser-android' | 'raw'
  addVirtualSpecs: boolean
  useBuildNumFile: boolean
  rawUserAgent?: string
  // optional pin
  pinAppVersion?: string
  pinBuild?: string
}

export interface AuthSourceSettings {
  mode: 'mail' | 'phone'
  mail: { category: 'temp'|'rent', provider: MailProviderType, domain?: string, checkLiveDie: boolean }
  phone: { provider: PhoneProviderType, /* … */ }
}
```
→ Mock service trả default; bridge wails sau gắn vào `app.go`.

---

## E. Spec dài hạn — yêu cầu mọi reg/verify mới phải đáp ứng

> Section này lưu lại để mỗi khi thêm platform mới (vd: `s27`, `iosV2`, `verifyAndroidS24`, ...), dev đối chiếu checklist.

### E.1 Bắt buộc
1. **KHÔNG** hardcode UA template trong package register/verify.
2. **PHẢI** gọi `uabuilder.ResolveBuilder(platform, uaBuilderType).Build(opts)`.
3. **PHẢI** honor `opts.AddVirtualSpecs` và `opts.UseBuildNumFile` đúng semantics A.1/A.2.
4. **PHẢI** đọc data từ `Config/device_info/`, `Config/mobile_devices/`, `Config/fbapp_inf/` — không hardcode device pool/version pool.
5. **PHẢI** expose `SetUA(raw string)` cho đường UseRawUa, không bypass builder bằng strip/append.
6. **PHẢI** trả về `UABuildResult.Model + OSVersion + Manufacturer + Brand` để caller lưu vào account model (cho payload sau).

### E.2 Khuyến nghị
1. Mỗi platform có `Config/mobile_devices/<platform>_devices.txt` filter để pool device match thế hệ máy (S23 không lẫn SM-J110G).
2. Mỗi platform có `Config/fbapp_inf/versions_and_builds_<platform>.txt` để pin pool app version match capability.
3. Khi platform dùng builder type khác mặc định (vd Token API mặc định Browser), expose sang UI dropdown `Nguồn UA`.
4. Verify class dùng chung builder (không reinvent UA build).
5. `ForceUseUGReg` (giữ UA reg dùng tiếp cho verify) implement bằng cách **cache `UABuildResult` của reg call**, verify gọi `SetUA(cached.UserAgent)` thay vì build lại.

### E.3 Quy tắc test
- Mỗi builder phải có `*_test.go` chứng minh:
  - Build với `addVirtualSpecs=false useBuildNumFile=false` → không có `Dalvik/`.
  - Build với `addVirtualSpecs=true useBuildNumFile=false` → có `Dalvik/2.1.0 (...; {Brand}-{Model})`.
  - Build với `addVirtualSpecs=true useBuildNumFile=true` → có `Dalvik/2.1.0 (...; <line từ device_build_nums.txt>)`.
  - Build với `addVirtualSpecs=false useBuildNumFile=true` → KHÔNG khác trường hợp 1 (vì build_num bị ignore).
  - `Model`/`OSVersion` trong UA phải match field `FBDV/`/`FBSV/`.

### E.4 Quy tắc UI
- Toggle `Dùng file build number` PHẢI disabled khi `Thêm virtual spec Android` tắt **VÀ** builder type là `android-app` (vì lúc đó build_num có effect chỉ với Browser builder).
- `Raw UA` text input chỉ enable khi `Nguồn UA = "Raw UA"`.
- Preview UA realtime cập nhật khi đổi bất kỳ toggle nào.

---

## F. Thứ tự thực thi (sau khi user duyệt plan)

| # | Việc | Phụ thuộc | Estimate |
|---|---|---|---|
| 1 | Copy `config/device_info/`, `config/mobile_devices/`, `config/fbapp_inf/`, `config/useragent/` từ HAVU → `d:/NCS/HVR/Config/` | – | 5 min |
| 2 | Tạo package `internal/facebook/fakeinfo/uabuilder` (interface + 4 impl + loader + tests) | #1 | ~3h |
| 3 | Refactor `s23/profile.go` xài uabuilder, xoá `SetUAOptions` sai | #2 | 30 min |
| 4 | Lặp lại #3 cho s22/s24/s25/s26 (đang là alias s23 — chỉ đổi `BuildProfileForPlatform("s24", …)`) | #3 | 30 min |
| 5 | Refactor s555/s556/s557 tương tự | #2 | 1h |
| 6 | Refactor android (V22) — cẩn thận giữ V22 UA quirks | #2 | 1h |
| 7 | Refactor webandroid + web + ioshttp | #2 | 1.5h |
| 8 | Thêm 2 field vào `RegInput`, runner forward từ settings | #3-#7 | 30 min |
| 9 | Frontend D.1 — group User Agent trong InteractionSetupPage | – | 1h |
| 10 | Frontend D.2 — tách AuthSourcePage thành tab mới | – | 2h |
| 11 | Bridge contracts D.4 + mock | #9-#10 | 30 min |
| 12 | Doc update [REGISTER_LOGIC_COMPARISON.md](REGISTER_LOGIC_COMPARISON.md) §3.6 + §6.1 reflect refactor | sau cùng | 15 min |

→ Tổng ~12-13h work. Có thể split PR.

---

## G. Quyết định đã chốt (APPROVED 2026-05-05)

| # | Câu hỏi | Quyết định |
|---|---|---|
| 1 | Nguồn UA selector | **Dùng lại `UA_POOLS` hiện tại** (3 button: android/iphone/chrome). Move xuống User Agent group, làm rõ active state (bold + highlight đậm). |
| 2 | Layout config | **`d:/NCS/HVR/Config/`** (top-level). User edit txt không cần rebuild. |
| 3 | `useBuildNumFile` UI dim rule | **Bỏ dim hoàn toàn**. Tích là cứ thêm, không kiểm tra builder type. |
| 4 | `Nguồn xác thực` tab | **Top-level sidebar tab**. Là tab lớn ngang hàng Accounts/Flow Settings/... |
| 5 | App version pool | **Per-platform** (s22/s23/s24/s25/s26/s555/s556/s557 mỗi platform 1 file `versions_and_builds_<platform>.txt`). Sau này update thêm. |

### Tác động ngược lại các section đã viết

- **Section A.3 / E.4**: bỏ rule dim `useBuildNumFile` (Q3).
- **Section C.2 / D.4**: file layout dùng `Config/{DeviceInfo,MobileDevices,Fbapp,UserAgent}/` (đặt tên CamelCase cho khớp folder hiện có như `Config/UserAgent/`).
- **Section D.1**: thay vì dropdown `Nguồn UA` 4 option mới, dùng lại 3 button `UA_POOLS` (android/iphone/chrome) ngay trong group User Agent — bỏ controlbar selector ở top.
- **Section E.2.2**: per-platform pool là MUST (không còn optional).

### Files đã copy / tạo (Step 1 done)

```
d:/NCS/HVR/Config/
  DeviceInfo/
    densities.txt          ← copy từ HAVU densitis.txt (đổi tên cho đúng chính tả)
    device_cores.txt       ← extend: arm64-v8a + arm64-v8a:armeabi-v7a:armeabi + armeabi-v7a:armeabi
    screen_resolution.txt  ← copy HAVU
    carriers.txt           ← copy HAVU
    buildnums.txt          ← (đã có sẵn)
    chrome_versions.txt    ← (đã có sẵn)
    devices.txt            ← (đã có sẵn — generic)
    os_versions.txt        ← (đã có sẵn)
  MobileDevices/
    android_devices.txt    ← copy HAVU (2190 lines, generic Android pool)
    s22_devices.txt        ← Galaxy S22 SM-S90xx (6 entries)
    s23_devices.txt        ← Galaxy S23 SM-S91xx (6)
    s24_devices.txt        ← Galaxy S24 SM-S92xx (6)
    s25_devices.txt        ← Galaxy S25 SM-S93xx (6)
    s26_devices.txt        ← Galaxy S26 SM-S94xx (5)
    s555_devices.txt       ← clone s23
    s556_devices.txt       ← clone s23
    s557_devices.txt       ← clone s23
  Fbapp/
    versions_and_builds.txt        ← (đã có sẵn — generic, dùng cho android V22)
    versions_and_builds_s23.txt    ← 7 phiên bản 550-556
    versions_and_builds_s555.txt   ← 1 phiên bản 555.0.0.49.59
    versions_and_builds_s556.txt   ← 1 phiên bản 556.1.0.63.64
    versions_and_builds_s557.txt   ← 1 phiên bản 557.0.0.59.72
  UserAgent/
    Android_UG.txt         ← (đã có)
    iOS_UG.txt             ← (đã có)
    Request_UG.txt         ← (đã có)
    Token_UG.txt           ← (đã có)
```

> Format MobileDevices/sXX_devices.txt: `Manufacturer:Brand:Model:Width:Height:Density:FBSS` (7 fields).
> Format MobileDevices/android_devices.txt: `Manufacturer:Brand:Model[:extra]` (3-5 fields, tương thích HAVU).
