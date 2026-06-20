# 05 — Config / Data Files / Pools / UA Builder

> Tài liệu này mô tả **CỰC KỲ CHI TIẾT** toàn bộ tầng dữ liệu (data layer) của HVR:
> các file trong `build/bin/Config/`, các store đọc chúng (`fbdata`, `cookie`, `fakeinfo`,
> `fakeinfo/uabuilder`), cơ chế **datr pool phân vùng** (`PartitionedDatrPool`), và
> **UA builder** (Android FB4A, Browser Chrome, ConfigFile pool, iOS FBIOS).
>
> Các tài liệu liên quan:
> - Runbook thêm version + luật token/login iOS: [add-facebook-reg-version.md](./add-facebook-reg-version.md) (§13.6/§13.7).
> - Khi tài liệu này nhắc tới "ai gọi pool/UA builder", xem chi tiết flow ở các doc reg/verify.

---

## 0. Sơ đồ tổng (data layer)

```
                          ┌──────────────────────────────────────────────────┐
                          │              build/bin/Config/                    │
                          │  (thư mục cạnh exe — user edit qua Notepad/UI)     │
                          └──────────────────────────────────────────────────┘
   DeviceInfo/            DeviceInfoIOS/        Cookie/            Settings/         Permanent/  Namereg/ Locales/
   ├ versions_and_builds  ├ ios_devices.txt     ├ cookie_initial   ├ interaction.json ├ mail.txt  ├ US/   ├ locales.txt
   │   [_reg|_ver].txt     ├ ios_app_builds.txt  ├ datr_pool.txt    ├ app_settings.json├ phone.txt └ VN/   SimNetwork/
   ├ devices.txt          └ ios_locales.txt      └ Pool{date}_N.txt └ general.json                       └ simnetworks.txt
   ├ carriers.txt                                                                  UserAgent/      phone_database/
   ├ densitis.txt                                                                  ├ Android_UG.txt  └ {Country}={CC}.{loc}.txt
   ├ device_cores.txt                                                              ├ iOS_UG.txt
   ├ devices_versions.txt                                                          ├ PC_UG.txt
   ├ screen_resolution.txt                                                         └ WebChrome_UA.txt
   ├ device_build_nums.txt
   ├ chrome_versions.txt
   └ googleapp_versions.txt
        │            │            │                 │                  │              │          │
        ▼            ▼            ▼                 ▼                  ▼              ▼          ▼
 ┌──────────────┐ ┌──────────┐ ┌──────────┐  ┌──────────────┐  ┌─────────────┐ ┌─────────┐ ┌─────────────┐
 │internal/fbdata│ │uabuilder │ │ios562    │  │internal/cookie│ │app.go        │ │fakeinfo │ │fakeinfo     │
 │(versions pool)│ │data_loader│ │devices.go│  │(datr files)   │ │InteractionCfg│ │overrides│ │phonecode/sim│
 └──────┬────────┘ └────┬─────┘ └────┬─────┘  └──────┬───────┘  └──────┬──────┘ └────┬────┘ └──────┬──────┘
        │               │            │               │                 │             │             │
        └───────────────┴────────────┴───────────────┴─────────────────┴─────────────┴─────────────┘
                                              │
                                              ▼
                  ┌──────────────────────────────────────────────────────┐
                  │  UA builder + Datr pool → register/verify workers       │
                  │  (Android FB4A / Browser Chrome / iOS FBIOS / pool UA) │
                  └──────────────────────────────────────────────────────┘
```

**Triết lý chung của tầng data (lặp lại ở mọi store):**

1. **File trên disk là source of truth chạy thật.** Code đọc `build/bin/Config/*` lúc runtime.
2. **Pattern "seed-dir / embed + override":**
   - `fakeinfo.SeedConfigDataIfMissing` chỉ tạo **cấu trúc thư mục** + vài placeholder rỗng (không seed content) — xem [seed_config.go](../../internal/facebook/fakeinfo/seed_config.go).
   - `cookie` + một vài store có **embedded seed** (ship sẵn data trong binary, extract khi file chưa tồn tại) — xem [cookie/store.go](../../internal/cookie/store.go) L30-34.
3. **Fallback hardcode** ở mọi loader: nếu file thiếu/rỗng/parse lỗi → dùng default trong code để app KHÔNG crash.
4. **Cache theo mtime** (uabuilder) hoặc `sync.Once` (device.go/ios562) — user sửa file → app pickup ở lần gọi kế (uabuilder) hoặc cần `Reload*` (sync.Once-based).

---

## 1. `internal/fbdata` — pool FBAV/FBBV (versions_and_builds)

**Mục tiêu:** quản lý pool cặp `Version|Build` (FBAV|FBBV) của Facebook Android app.
**File:** [internal/fbdata/store.go](../../internal/fbdata/store.go).

### 1.1 Ba pool tách riêng (refactor 2026-05-28)

| Pool | File | Hàm public | Fallback |
|------|------|-----------|----------|
| CHUNG | `Config/DeviceInfo/versions_and_builds.txt` | `Versions()` | hardcode `554.0.0.57.70\|918990560` |
| REG | `Config/DeviceInfo/versions_and_builds_reg.txt` | `VersionsReg()` | → pool CHUNG nếu rỗng |
| VER | `Config/DeviceInfo/versions_and_builds_ver.txt` | `VersionsVer()` | → pool CHUNG nếu rỗng |

**Format file** (mỗi dòng `FBAV|FBBV`, dòng `#`/rỗng bị skip):

```text
273.0.0.39.123|218047977
554.0.0.57.70|918990560
556.1.0.63.64|942217461
```

Parse: [`ParseVersionsAndBuilds`](../../internal/fbdata/store.go) L123-139 — `SplitN(line, "|", 2)`, bỏ dòng không có `|` hoặc field rỗng.

### 1.2 Đường dẫn & legacy fallback

- `DefaultDir = "Config/DeviceInfo"` (mới, ưu tiên), `LegacyDir = "Config/Fbapp"` (cũ).
- `DefaultVersionsAndBuildsPath()` (L45-55): ưu tiên `DeviceInfo/`, fallback `Fbapp/`, không có thì trả path mới.
- Pool reg/ver **luôn** ở `Config/DeviceInfo/` (`splitDir()` L59-61).

### 1.3 Vòng đời (startup)

`app.go OnStartup` ([app.go](../../app.go) L795-802):

1. `fbdata.EnsureDir()` — tạo `Config/DeviceInfo/`.
2. `fbdata.EnsureSplitFiles()` (L84-112) — tạo 3 file: file CHUNG (copy từ `Fbapp/` legacy nếu có), 2 file split `_reg`/`_ver` luôn tạo **rỗng** (user tự điền).
3. `fbdata.Reload("")` (L173-184) — nạp cả 3 pool vào in-memory cache (`activeVersions`, `activeReg`, `activeVer`).
4. Log: `slog.Info("fb versions active", count, reg, ver, override)`.

> **Gotcha:** `SetDefaultVersions` (L163-168) được fakeinfo gọi trong init để set `defaultVersions` (embed). `applyMergeLocked` (L250-258): nếu file override CHUNG có data → dùng nó; ngược lại fallback default embed. File split KHÔNG merge với default — rỗng thì `VersionsReg/Ver` tự fallback pool CHUNG.

### 1.4 Ai đọc

- [`fakeinfo.RandomFbVersionReg()`](../../internal/facebook/fakeinfo/useragent.go) L24-26 → `fbdata.VersionsReg()`.
- `RandomFbVersionVer()` L30-32 → `fbdata.VersionsVer()` (vd [scheduler.go](../../internal/runner/scheduler.go) L546).
- `RandomFbVersion()` L18-20 → `fbdata.Versions()` (pool CHUNG).
- `pickRandomVersion` (L34-41): random 1 phần tử; pool rỗng → hardcode `554.0.0.57.70`.

> **Lưu ý 2 hệ song song:** `uabuilder` có loader RIÊNG cho cùng các file này
> (`LoadAppVersionsForReg/Ver/Platform` — xem §3.2). `fbdata` phục vụ
> `BuildAndroidUA` (useragent.go cũ), còn `uabuilder.AndroidUABuilder` đọc trực tiếp
> file qua `data_loader.go`. Cùng nguồn file, 2 code path.

---

## 2. `internal/cookie` — datr files (initial / pool / run-pool)

**Mục tiêu:** quản lý 3 dạng file datr trong `Config/Cookie/` + queue ghi datr ra disk.
**File:** [internal/cookie/store.go](../../internal/cookie/store.go).

### 2.1 Các file

| File | Hằng số | Vai trò |
|------|---------|---------|
| `cookie_initial.txt` | `InitialFilename` | **Input** — user paste datr/cookie line; nạp vào pool reg lúc start. Có embedded seed. |
| `datr_pool.txt` | `PoolFilename` | Pool tự tích lũy datr mới. Có embedded seed. |
| `Pool{YYYYMMDD}_{N}.txt` | (dynamic) | File pool **của mỗi lần chạy** — datr mới reg/verify thành công ghi append vào đây. |

**Format `cookie_initial.txt`:** mỗi dòng có thể là:
- raw datr value: `nB8GanxDi5e0NfQTEuDawVVk`
- full cookie line / account line chứa `datr=xxx;` → `ExtractDatr` (L76-82) regex `datr=([A-Za-z0-9_-]+)`.

`Pool*.txt` thường là raw datr 1 dòng/giá trị (vd `guYJajYG7Cz3AC39omw_4Mct`).

### 2.2 Embedded seed

```go
//go:embed embedded/cookie_initial.txt
var embeddedCookieInitial []byte
//go:embed embedded/datr_pool.txt
var embeddedDatrPool []byte
```

`SeedInitialIfMissing` (L98-108): nếu file chưa có → ghi embedded (hoặc comment placeholder). Gọi từ `app.go OnStartup` L787 và lúc `RunRegister` chuẩn bị pool (L7707).

### 2.3 `NewRunPoolPath` — sinh tên file pool/run

[`NewRunPoolPath(dir)`](../../internal/cookie/store.go) L546-569:
- Pattern: `{dir}/Pool{YYYYMMDD}_{N}.txt`, `N` tự tăng dựa trên file đã có cùng prefix ngày.
- Quét `os.ReadDir(dir)`, parse số sau prefix `Pool20260531_`, lấy max + 1.
- Gọi trong `RunRegister` ([app.go](../../app.go) L7733): `runPoolPath := cookie.NewRunPoolPath(defaultCookieDir())`.

### 2.4 Append datr

- **`AppendDatrToPool(path, datr)`** (L353-374): ghi trực tiếp `O_APPEND` vào file Pool, KHÔNG qua queue (Pool file append-only). Skip nếu datr rỗng / bắt đầu bằng `_` hoặc `-` (datr invalid). Đây là hàm `persistNewDatr` dùng ([app.go](../../app.go) L7753).
- **`AppendDatr(path, datr)`** (L149-199): append vào `datr_pool.txt` với **dedup in-memory** (`savedSet`) + lazy-load file vào set lần đầu. Idempotent.
- **`DatrFileQueue`** (L296-500): batch nhiều append/remove rồi flush mỗi `flushInterval` (default 1500ms). Dùng cho remove-everywhere. Tạo trong RunRegister L7735 với `queuePaths = [runPoolPath, cookieInitialFilePaths...]`.

### 2.5 Vòng đời cookie pool trong RunRegister

Xem chi tiết §4.2. Tóm tắt: app build `cookieInitialFilePaths` → tạo `runPoolPath` → tạo `DatrFileQueue` → load datr vào `PartitionedDatrPool` → set persist/remove hook → gán pool cho mọi platform.

---

## 3. UA Builder — `internal/facebook/fakeinfo/uabuilder`

**Mục tiêu:** sinh User-Agent cho mọi platform reg/verify. Port 1:1 từ C# `FakeInfoBuilder/`.
**Entry:** [builder.go](../../internal/facebook/fakeinfo/uabuilder/builder.go).

### 3.1 Ba (bốn) loại builder + 3 mode

`ResolveBuilder(platform, builderType)` (builder.go L175-190) dispatch theo `builderType`:

| builderType | Hằng số | Builder | Output | C# tương đương |
|---|---|---|---|---|
| 0 | `BuilderTypeAndroidApp` | `AndroidUABuilder` | FB4A native UA (build động từ DeviceInfo) | AndroidUserAgentBuilder.cs |
| 1 | `BuilderTypeConfigFile` | `ConfigFileUABuilder` | random 1 UA từ pool file | ConfigFileUserAgentBuilder.cs |
| 2 | `BuilderTypeBrowserAndroid` | `BrowserUABuilder` | Mozilla Chrome Android (wv) | BrowserAndroidUserAgentBuilder.cs |

**Tương ứng 2 toggle trong InteractionConfig / PlatformUAConfig** ([app.go](../../app.go) L4608-4614):
- `BuildUA=true` → mode **build động** (AndroidUABuilder, builderType=0): ghép UA từ `Config/DeviceInfo/`.
- `BuildUA=false` → mode **pool file** (ConfigFileUABuilder, builderType=1): random từ `Config/UserAgent/<kind>_UG.txt`.
- `AddVirtualSpecAndroid` → prepend `Dalvik/2.1.0 (...)` (chỉ AndroidUABuilder honor).
- WebAndroid/Token → BrowserUABuilder (builderType=2).

### 3.2 `UAOptions` + `PoolKind` (reg/ver) — builder.go L42-74

```go
type UAOptions struct {
    Locale          string // FBLC; "" → "en_US"
    AddVirtualSpecs bool   // prepend Dalvik
    SimBrand        string // override FBCR
    CountryCode     string // pick carrier theo IP
    Platform        string // s22/s23/.../android — filter device pool
    PinAppVersion   string // ép FBAV cố định
    PinBuild        string // ép FBBV cố định
    PoolKind        string // "" | "reg" | "ver" — chọn file versions_and_builds[_reg|_ver]
    rand            *rand.Rand
}
```

**`PoolKind`** quyết định pool FBAV/FBBV khi random ([android.go](../../internal/facebook/fakeinfo/uabuilder/android.go) L57-64):
- `"reg"` → `LoadAppVersionsForReg()` → `versions_and_builds_reg.txt`.
- `"ver"` → `LoadAppVersionsForVer()` → `versions_and_builds_ver.txt`.
- `""` → `LoadAppVersionsForPlatform("")` → `versions_and_builds.txt` (fallback Fbapp legacy).

Tất cả fallback `fallbackAppVersions` (android.go L33-37) nếu file rỗng.

### 3.3 `data_loader.go` — đọc file + cache mtime

[data_loader.go](../../internal/facebook/fakeinfo/uabuilder/data_loader.go).

- `ConfigBaseDir` mặc định `"Config"`; override bằng `SetConfigBaseDir` (invalidate toàn bộ cache). `GetConfigBaseDir()` được ios562 dùng để tìm `DeviceInfoIOS/`.
- **3 cache độc lập** (`listCache`, `deviceCache`, `appCache`), tất cả key theo path + so `mtime.UnixNano()` → reload khi file đổi.

**Mapping file → loader:**

| File `Config/DeviceInfo/` | Loader | Kiểu |
|---|---|---|
| `densitis.txt` | `loadDensities` | `[]string` (vd `3.0`) |
| `device_cores.txt` | `loadCores` | `[]string` (FBCA arch, vd `arm64-v8a`, `x86:armeabi-v7a`) |
| `devices_versions.txt` | `loadOSVersions` | `[]string` (OS, vd `13`) |
| `screen_resolution.txt` | `loadScreenResolutions` | `[]string` `WxH` (vd `1080x2340`) |
| `carriers.txt` | `loadCarriers` | `[]string` (FBCR) |
| `chrome_versions.txt` | `loadChromeVersions` | `[]string` (Browser) |
| `googleapp_versions.txt` | `loadGoogleAppVersions` | `[]string` (Browser) |
| `device_build_nums.txt` | `loadStringList(...)` trực tiếp | build ID Mozilla (vd `SKQ1.210908.001`) |
| `<platform>_devices.txt` → `devices.txt` | `LoadDevicesForPlatform` | `[]DeviceSpec` |
| `versions_and_builds[_reg|_ver].txt` | `LoadAppVersionsFor*` | `[]AppVersion` |

**`DeviceSpec` (devices.txt) format** (L117-132): `Manufacturer:Brand:Model[:Width:Height[:Density[:FBSS]]]`
- 3 fields (phổ biến nhất, vd `Xiaomi:Xiaomi:ZT_216`): width/height = 0 → builder random từ `screen_resolution.txt` + `densitis.txt`.
- 5 fields: thêm `Width:Height`. 7 fields: thêm `Density:FBSS`.
- Loader `LoadDevicesForPlatform` (L149-177): thử `DeviceInfo/<platform>_devices.txt` trước, rồi `DeviceInfo/devices.txt`.

### 3.4 `AndroidUABuilder.Build` — flow chi tiết (android.go L43-199)

Input: `UAOptions`. Output: `UABuildResult{UserAgent, Model, OSVersion, Carrier, AppVersion,...}`.

1. **App version+build:** `PinAppVersion/PinBuild` nếu set, ngược lại theo `PoolKind` (§3.2). Random 1 cặp.
2. **Device pool:** `LoadDevicesForPlatform(opts.Platform)`. Nếu platform là "Samsung S-series" (`samsungSPlatform` L202-224, danh sách rất dài s22..s563) → `filterSamsungSDevices` (chỉ giữ `samsung/samsung/SM-S*`). Rỗng → `fallbackDevices`.
3. **Density:** từ device, hoặc random `densitis.txt`, fallback `"3.0"`.
4. **Resolution:** từ device, hoặc random `screen_resolution.txt`, fallback `1080x2340`.
5. **Locale:** `opts.Locale` hoặc `"en_US"`.
6. **Carrier (FBCR):** `SimBrand` override → `GetCarrierPicker()(CountryCode)` → random `carriers.txt` → `"T-Mobile"`.
7. **CPU arch (FBCA):** random `device_cores.txt`, fallback `arm64-v8a`.
8. **OS version (FBSV):** random `devices_versions.txt`, fallback `13`.
9. **Build ID:** `Brand-Model` (vd `samsung-SM-S911B`).
10. **Compose FB_UG** (L164-171):

```
[FBAN/FB4A;FBAV/<ver>;FBBV/<build>;FBDM={density=<d>,width=<w>,height=<h>};
 FBLC/<locale>;FBRV/0;FBCR/<carrier>;FBMF/<manufacturer>;FBBD/<brand>;
 FBPN/com.facebook.katana;FBDV/<model>;FBSV/<os>;FBOP/1;FBCA/<arch>]
```

> **Gotcha format (android.go L16-19):** `FBDM={density=...}` (KHÔNG `FBDM/{...}` — slash là bug version cũ), và đóng bằng `]` (KHÔNG `;]`). Match C# AndroidUserAgentBuilder.cs L87-102.

11. **AddVirtualSpecs:** nếu true → prepend `Dalvik/2.1.0 (Linux; U; Android <os>; <Model> Build/<buildID>) ` (L176-179).

### 3.5 `BrowserUABuilder.Build` (browser.go) — Chrome Android WebView

Output: `Mozilla/5.0 (Linux; Android <os>; <model> Build/<buildId>; wv) AppleWebKit/537.36 ... Chrome/<ver> Mobile Safari/537.36 GoogleApp/<gaVer>`.
- Build ID lấy từ `device_build_nums.txt` (vd `SKQ1.210908.001`), fallback `Brand-Model`.
- Chrome ver từ `chrome_versions.txt`; nếu 3-part → tự thêm `.<50-200>` thành 4-part (L68-70). Fallback `146.0.7680.111`.
- GoogleApp từ `googleapp_versions.txt`, fallback `17.18.24.ve.arm64`.
- **Suffix metadata** `|<model>|<os>` append cuối UA để caller tách (L86). Dùng cho WebAndroid / Token.

### 3.6 `ConfigFileUABuilder.Build` (configfile.go) — pool prebuilt UA

- Đọc 1 UA random qua adapter `Source ConfigFileUASource` (function `func() string`).
- `AddVirtualSpecs` và `BuildUA` bị **IGNORE** — UA đã prebuilt.
- Pool rỗng → trả `errConfigFileEmptyPool` (caller fallback build động).

### 3.7 Wiring source pool — `uabuilder_wire.go`

[uabuilder_wire.go](../../internal/facebook/fakeinfo/uabuilder_wire.go) `init()`:
- `SetCarrierPicker(func(cc) { return RandomSimProfile(cc).OperatorName })` — carrier picker theo IP.
- `SetConfigFileSource("", ...)` default → `RandomUAFromPool(UAKindAndroid)`.
- Map per-platform → pool kind:
  - `android, s22..s557` → `UAKindAndroid` (Android_UG.txt).
  - `ios` → `UAKindIOS` (iOS_UG.txt).
  - `webandroid` → `UAKindAndroid` (fallback).
  - `mfb, request` → `UAKindRequest` (PC_UG.txt — Chrome Desktop).

### 3.8 Pool UA file-based — `ua_pools.go`

[ua_pools.go](../../internal/facebook/fakeinfo/ua_pools.go). Quản lý 4 pool prebuilt từ `Config/UserAgent/`:

| `UAPoolKind` | File override | Nội dung |
|---|---|---|
| `UAKindAndroid` | `Android_UG.txt` | FB4A native UA |
| `UAKindIOS` | `iOS_UG.txt` | FBIOS iPhone UA |
| `UAKindRequest` | `PC_UG.txt` | Chrome Desktop (label "PC"; đổi từ Request_UG.txt 2026-05) |
| `UAKindWebChrome` | `WebChrome_UA.txt` | Chrome Mobile Android (label "WebMobile") |

- **`loadUAPool`** (L66-81): đọc CHỈ từ `Config/UserAgent/<file>` (không embed làm default). Rỗng → pool nil → caller fallback build động.
- **`ReloadUAPools`** (L85-90): force reload — gọi từ `app.go OnStartup` L808 sau `EnsureUAOverrideDir`.
- **`AppendUAToPool(kind, ua)`** (L96-131): learning loop — verify OK với UA nào thì append lại pool (idempotent, dedup in-memory + ghi file `O_APPEND`).
- `RandomUAFromPool`, `UAPoolSize`, `UAPoolAll`, `UAPoolOverrideActive`, `UAOverridePath` — getter cho UI/wiring.

> **Gotcha:** `uaOverrideFiles[UAKindRequest] = "PC_UG.txt"` — UI/log gọi là "request"/"PC" nhưng file là `PC_UG.txt`. `Request_UG.txt` cũ vẫn còn trong thư mục nhưng KHÔNG được load.

---

## 4. Datr Pool — `PartitionedDatrPool` (register/android/pool.go)

**Mục tiêu:** mỗi worker (goroutine slot) có **partition datr riêng** → không bao giờ 2 worker dùng cùng datr trùng nhau trong cùng thời điểm.
**File:** [internal/facebook/register/android/pool.go](../../internal/facebook/register/android/pool.go).

### 4.1 Cấu trúc & các "kho"

`PartitionedDatrPool` (L161-184) gồm:

| Trường | Ý nghĩa |
|---|---|
| `partitions map[int][]string` | queue datr riêng theo slotIdx |
| `pending []string` | datr chưa phân phối (chưa có slot active để nhận) |
| `exhausted []string` + `exhaustedSet` | datr đã đạt `maxUsage` — chờ **recycle** khi pool cạn |
| `usageCount` / `successCount` / `failCount` / `unknownCount` / `checkpointCount` | thống kê per-datr |
| `loadedAt map[string]time.Time` | thời điểm nạp — cho age expiry |
| `activeCount int` | tổng datr active (pending + partitions, KHÔNG tính exhausted) |
| `maxUsage` | giới hạn dùng/datr (= `LimitCookieInitialCount`, default 9999) |
| `maxCheckpoint` | giới hạn checkpoint/datr trước khi xóa |
| `maxAge time.Duration` | tuổi tối đa (0 = không expire) |
| `fillBatch int` (=64) | số datr fill vào 1 slot mỗi lần |
| `deleteOnUsageLimit bool` | true → xóa hẳn khi hết usage; false → chuyển exhausted recycle |
| `persistOnlyNew bool` | KeepDatrSuccess — chỉ persist datr mới, không add vào partitions ngay |
| `persistHook` / `removeHook` | callback ghi/xóa datr ra disk |

### 4.2 Lifecycle (gọi từ RunRegister)

```
Startup pool (app.go L7789-7867):
  NewPartitionedPool(cookieInitialLimit)
    → LoadFromFile(cookieInitialFilePaths)   // hoặc LoadGenerated nếu method="new"
    → SetPersistHook(persistNewDatr)          // ghi datr mới vào Pool{date}_N.txt
    → SetRemoveHook(removeDatrEverywhere)
    → SetPersistOnlyNewDatr(KeepDatrSuccess)
    → if DeleteDatrCheckpoint: SetMaxCheckpoint + (LimitCookieInitial→SetDeleteOnUsageLimit)
    → if LimitDatrAge: SetMaxAgeMinutes + background ticker gọi ExpireOldDatrs()
    → gán cho mọi platform pool (allPlatformPools, L7865-7867)

Mỗi worker (app.go L8908-8942):
  SharedPool.Register(slotIdx)            // defer Unregister
  loop:
    datr := SharedPool.GetNext(slotIdx)   // lấy datr chưa quá limit
    ... reg ...
    SharedPool.IncrementUsage(datr)       // hoặc RecordResult(datr, outcome)
    nếu thu được datr mới: SharedPool.AddDatrRaw(datr)
```

**`Register(slotIdx)`** (L335-369): đăng ký slot + **STEAL 1/n** datr từ mỗi slot đang active (cân bằng partition).
> **Gotcha (L329-334):** restore 2026-05-15 — không steal thì với 200 luồng + 9901 datr, slot 155-200 nhận 0 datr → "Thiếu datr". Steal đảm bảo phân phối đều ~49 datr/slot.

**`GetNext(slotIdx)`** (L406-485) thứ tự fallback khi partition rỗng:
1. `fillSlotLocked` từ `pending`.
2. **STEAL ½** từ slot giàu nhất (FALLBACK 1, L420-445).
3. Nếu `activeCount==0` và có `exhausted` → `recycleExhaustedLocked` rồi fill lại.
4. Rotate datr về cuối queue nếu `usage < maxUsage` (dùng lại tối đa maxUsage lần). Đạt limit → `exhaustDatrLocked`.

**`IncrementUsage(datr)`** (L493-512): tăng counter; khi đạt `maxUsage`:
- `deleteOnUsageLimit=false` (mặc định) → `exhaustDatrLocked` (chờ recycle).
- `deleteOnUsageLimit=true` → `removeDatrLocked` + fire `removeHook` (xóa khỏi file).

**`RecordResult(datr, outcome)`** (L291-316): đếm success/fail/unknown/**checkpoint**. Khi `checkpointCount >= maxCheckpoint` → remove datr + hook.

**`AddDatr` / `AddDatrRaw` / `AddDatrRawNoPersist`** (L518-690): thêm datr mới từ reg/verify thành công. `persistOnlyNew=true` → KHÔNG add vào partitions, chỉ trigger persistHook. `AddDatrRawNoPersist` dùng để sync datr mới sang pool platform khác mà tránh vòng lặp ghi file (app.go L7757-7762).

**`ExpireOldDatrs()`** (L243-267): quét `loadedAt`, remove datr quá `maxAge`, fire removeHook. Gọi định kỳ bởi background ticker (app.go L7847-7863, interval = `maxAge/4`, min 30s).

**Recycle vs Delete:** đây là 2 triết lý "datr hết hạn":
- Default → **recycle** (reset usage=0 khi pool cạn) để dùng lại đến vô tận.
- `DeleteDatrCheckpoint` + `LimitCookieInitial` → **delete** hẳn (1 datr chỉ sống đúng N lần rồi biến mất khỏi file).

### 4.3 `SharedPool` toàn cục + đa platform

- `SharedPool *PartitionedDatrPool` (L27) — biến package, set từ app.go.
- Mọi platform reg chia sẻ **cùng 1 instance** `sharedCookiePool` (app.go L7865-7867 gán cho `androidreg/s23reg/s399reg/webandroidreg/ioshttpreg/webregister.SharedPool` + `ios562reg/ios563reg.SharedDatrPool`).
- iOS562 dùng `SharedDatrPool` (cùng type) — [ios562/pool.go](../../internal/facebook/register/ios562/pool.go) L33, đọc qua `GetNext`/`AddDatrRaw` trong register.go (L130-131, L251-252).

### 4.4 `CookiePool` (round-robin — backward compat)

Cùng file (L55-143). Đơn giản hơn: round-robin chung, không partition. Giữ lại cho code cũ, KHÔNG dùng production.

---

## 5. SimProfile / HNI / Locale — `fakeinfo`

### 5.1 `simnetwork.go` — SIM/carrier

[simnetwork.go](../../internal/facebook/fakeinfo/simnetwork.go).
- `SimProfile{MCC, MNC, OperatorName, CountryCode, HNI}` — `HNI = MCC+MNC` (vd `45204`), dùng cho header `X-FB-SIM-HNI`.
- **Data:** `Config/SimNetwork/simnetworks.txt`, format `MCC|MNC|OperatorName|CountryCode` (vd `202|05|Vodafone|GR`). Load qua `loadSimNetworkOverride` (overrides.go L97-123).
- `RandomSimProfile(countryCode)` (L38-63): filter theo country, fallback random toàn list, fallback hardcode T-Mobile US.
- `RandomLocale()` (L29-35): random từ `localeList` (Config/Locales/locales.txt), fallback `en_US`.
- `LocaleFromCountry(cc)` (L87-99): map hardcode 18 country → locale.

### 5.2 `iosConnTypes` — cellular-only (yêu cầu 2026-05-31)

[simnetwork.go](../../internal/facebook/fakeinfo/simnetwork.go) L69-84.
- Pool `X-FB-Connection-Type` cho FBIOS: **CHỈ mobile** (đã bỏ "wifi"). Lý do: trên wifi FB iOS không gửi `X-FB-SIM-HNI` → fingerprint yếu, dễ bị soi.
- Giá trị: `mobile.CTRadioAccessTechnologyLTE` (weight cao), `...NRNSA`, `...NR`, `...WCDMA`, `...HSDPA`.
- `RandomIOSConnType()` được iOS562 profile dùng (profile.go L77, L99).

---

## 6. iOS native data — `register/ios562/devices.go` + iOS UA

### 6.1 File `Config/DeviceInfoIOS/`

[ios562/devices.go](../../internal/facebook/register/ios562/devices.go). Path qua `uabuilder.GetConfigBaseDir() + "DeviceInfoIOS/"`.

| File | Format | Loader | Fallback |
|---|---|---|---|
| `ios_devices.txt` | `FBDV\|IOSDot\|IOSUnder\|MobileBld\|FBSS` (5 fields) | `loadIOSDevicesFromFile` (L88-120) | `defaultIPhoneDevices` (L37-62) |
| `ios_app_builds.txt` | `FBAV\|FBBV\|FBRV` (3 fields) | `loadFBBuildsFromFile` (L159-189) | `defaultFBBuilds` (L135-140) |
| `ios_locales.txt` | 1 locale/dòng | `loadIOSLocalesFromFile` (L225-243) | `defaultIOSLocales` (L198-206) |

**Ví dụ:**
- `ios_devices.txt`: `iPhone9,1|15.8.8|15_8_8|19H422|2` (FBSS: 2=@2x Retina, 3=@3x OLED).
- `ios_app_builds.txt`: `562.0.0.61.70|974804325|979621922`.

Load lazy bằng `sync.Once` (`iPhoneDevicesOnce`, `fbBuildsOnce`, `iosLocalesOnce`). `ReloadIOSDevices()` (L83-86) reset Once để reload khi user sửa file.

### 6.2 `buildIOSUA` — ghép UA FBIOS (profile.go L106-116)

```
Mozilla/5.0 (iPhone; CPU iPhone OS <IOSUnder> like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko)
 Mobile/<MobileBld> [FBAN/FBIOS;FBAV/<FBAV>;FBBV/<FBBV>;FBDV/<FBDV>;FBMD/iPhone;FBSN/iOS;
 FBSV/<IOSDot>;FBSS/<FBSS>;FBID/phone;FBLC/<locale>;FBOP/5;FBRV/<FBRV>]
```

- Device random từ `getIPhoneDevices()`, build random từ `randFBBuild()`.
- Gọi trong `BuildProfile` / `BuildProfileFromDevice` (profile.go L61-102) — sinh kèm `DeviceID/FamilyDeviceID/MachineID/...` + `ConnType` (cellular) + `Sim`.

### 6.3 `ios_ua.go` — pool UA iOS (fallback đọc file)

[ios_ua.go](../../internal/facebook/fakeinfo/ios_ua.go). `RandomIPhoneProfile()` đọc random từ file UA iOS (legacy path `Config/Settings/useragent/iOS_UG.txt`, `config/useragent/iOS_UG.txt`, `iOS_UG.txt`) → parse FBDV/iOS version. Fallback generate. Đây là path CŨ, song song với `ua_pools.go` (UAKindIOS).
> **Gotcha:** `loadIOSUAPool` (L32-54) tìm path khác với `ua_pools.go` (`Config/UserAgent/iOS_UG.txt`). Đây là 2 hệ riêng — `RandomIPhoneProfile` cho luồng cũ, `RandomUAFromPool(UAKindIOS)` cho ConfigFileUABuilder.

---

## 7. `phone_database` + name/permanent — `fakeinfo`

### 7.1 `phonecode.go` — phone country code

[phonecode.go](../../internal/facebook/fakeinfo/phonecode.go).
- **Data:** thư mục `Config/phone_database/` — mỗi file 1 quốc gia, tên `{CountryName}={CountryCode}.{locale}.txt` (vd `Australia=AU.en_GB.txt`).
- **Nội dung:** pattern số điện thoại 1 dòng (vd `+6144xxxxxxx`); trailing `x` = chữ số biến đổi (strip khi build prefix).
- `LoadPhoneDatabase(dir)` (L42-124): gọi từ `app.go OnStartup` L820. Parse từng file:
  - `PhoneCode = commonPhonePrefix(patterns)` (prefix chung, vd `+614`).
  - Build `phoneList` (1/country, dedup), `phoneByCountry`, `phonePrefixList` (per-line cho longest-prefix match).
- `LookupPhoneCode(cc)`, `PhoneCodeFor(cc)`, `FindCountryByPhonePrefix(num)` (longest-prefix matching).

### 7.2 Name + Locale + Permanent — `overrides.go`

[overrides.go](../../internal/facebook/fakeinfo/overrides.go) `ReloadOverrides()` (gọi app.go L824):

| Override | File | Pointer |
|---|---|---|
| Tên US | `Config/Namereg/US/firstname.txt`, `lastname.txt` | `firstNames`, `lastNames` |
| Tên VN | `Config/Namereg/VN/firstname.txt`, `lastname.txt` | `vnFirstNames`, `vnLastNames` |
| Locales | `Config/Locales/locales.txt` | `localeList` |
| SimNetwork | `Config/SimNetwork/simnetworks.txt` | `simList` |

- File rỗng/không tồn tại → giữ embedded (KHÔNG ghi đè) — `loadLinesFileOverride` L77-93.
- `NameReg=US|VN` chọn pool tên (field `nameRegLocale` trong interaction.json).
- **Không** override: DeviceInfo, PhoneDatabase, Carriers (đọc trực tiếp).

**Permanent** (`Config/Permanent/mail.txt`, `phone.txt`): list mail/phone có sẵn — dùng khi `phoneMailMode = "random-file"`. Format 1 entry/dòng, `#` comment. Seed placeholder bởi `SeedConfigDataIfMissing` (seed_config.go L51-58).

---

## 8. Settings — `interaction.json` (các field điều khiển data layer)

**File runtime:** `Config/Settings/interaction.json`.
**Struct:** [`InteractionConfig`](../../app.go) L4411-4680. Đọc qua `LoadInteractionConfig` (app.go L6019), ghi qua `SaveInteractionConfig` (L5963).

> Hệ settings có 2 lớp: `internal/settings` (model/adapter/store — chuẩn mới, profile-based,
> xem [defaults.go](../../internal/settings/model/defaults.go)) và `interaction.json` (legacy
> runner đọc trực tiếp). `SaveInteractionConfig` ghi cả vào active profile (qua
> `adapter.LegacyInteractionConfig`) lẫn `interaction.json` để backward-compat (L5979-5994).

### 8.1 Các field chính ảnh hưởng pool / UA / data

| Field JSON | Go | Tác dụng tới data layer |
|---|---|---|
| `createEnabled` | `CreateEnabled` | bật luồng register |
| `verifyEnabled` | `VerifyEnabled` | bật luồng verify; ảnh hưởng `SkipAuthLoginAtReg` (app.go L7681) |
| `splitMode` | `SplitMode` | reg/verify chạy độc lập (reg ghi file → verify đọc) |
| `apiRegPlatform` / `apiRegPlatforms` | `ApiRegPlatform[s]` | chọn platform reg → quyết định UA builder + pool dùng |
| `apiVerifyPlatform` / `apiVerifyPlatforms` | `ApiVerifyPlatform[s]` | platform verify (multi-version round-robin) |
| `cookieInitialMethod` | `CookieInitialMethod` | `"file"` (đọc file) \| `"new"` (sinh datr nội bộ qua `LoadGenerated`) |
| `cookieInitialFile` | `CookieInitialFile` | đường dẫn file cookie_initial.txt (rỗng → default) |
| `limitCookieInitial` / `limitCookieInitialCount` | `LimitCookieInitial[Count]` | `maxUsage` của pool (= cookieInitialLimit) |
| `limitCheckpoint` / `limitCheckpointCount` | `LimitCheckpoint[Count]` | `maxCheckpoint` của pool |
| `deleteDatrCheckpoint` | `DeleteDatrCheckpoint` | bật `SetMaxCheckpoint` + (`LimitCookieInitial` → `SetDeleteOnUsageLimit`) |
| `limitDatrAge` / `limitDatrAgeMinutes` | `LimitDatrAge[Minutes]` | `SetMaxAgeMinutes` + ticker `ExpireOldDatrs` |
| `saveNewDatr` | `SaveNewDatr` | bật `persistNewDatr` → ghi datr mới vào `Pool{date}_N.txt` |
| `keepDatrSuccess` | `KeepDatrSuccess` | `SetPersistOnlyNewDatr` |
| `getNewDatrOnLive` | `GetNewDatrOnLive` | sau verify Live → gọi GraphQL lấy datr mới (inline) thêm vào pool + Pool file (app.go L3089) |
| `buildUA` | `BuildUA` | true → build động (AndroidUABuilder) \| false → pool file (ConfigFileUABuilder) |
| `addVirtualSpecAndroid` | `AddVirtualSpecAndroid` | prepend Dalvik prefix |
| `useOriginalUA` / `replaceCarrier` | `UseOriginalUA`/`ReplaceCarrier` | dùng UA gốc cố định (s5xx); replace FBCR theo IP |
| `uaPoolKey` | `UaPoolKey` | loại UA pool ("android"/"iphone"/"request"/"webchrome") |
| `trackingIDReg` / `trackingIDVer` | `TrackingIDReg`/`Ver` | append `XID/<rand16>;` cuối UA |
| `regPlatformUA` / `verifyPlatformUA` | `RegPlatformUA`/`VerifyPlatformUA` | **override UA config theo từng platform** (key = platform name) |
| `nameRegLocale` | `NameRegLocale` | "US" \| "VN" → chọn pool tên |
| `phoneMailMode` | `PhoneMailMode` | `"random-normal"` \| `"random-file"` (dùng Permanent/) |
| `resultFolderPath` | `ResultFolderPath` | thư mục SuccessReg/SuccessVerify/Die |

### 8.2 `regPlatformUA` / `verifyPlatformUA` — UA config per-platform

Map `platform → PlatformUAConfig{useOriginalUA, addVirtualSpecAndroid, buildUA, replaceCarrier, trackingID, uaPoolKey, kind}` (interaction.json L116-1782).
- `applyRegPlatformUAConfig(cfg, platform)` (app.go L493) / `applyVerifyPlatformUAConfig` (L479): nếu key tồn tại → override các field UA toàn cục bằng config của platform đó.
- Ví dụ: `webandroid` → `buildUA=false, useOriginalUA=true, uaPoolKey="webchrome"` (dùng pool WebChrome). `s23` → `buildUA=true, uaPoolKey="android"`. `s556` → `useOriginalUA=true, buildUA=false`.

### 8.3 `app_settings.json` / `general.json`

- `app_settings.json`: format mới profile-based ([model/types.go] + defaults.go), auto-migrate từ general.json/interaction.json khi chưa có (`appsettings.Load`, app.go L771-775).
- `general.json`: format cũ (LoginPlatform, AccountSourcePath...). Còn được sync 2 chiều với app_settings/interaction để runner cũ chạy được.

---

## 9. Edge case & Gotcha tổng hợp

1. **Hai hệ đọc versions_and_builds:** `fbdata` (cho `BuildAndroidUA` cũ) vs `uabuilder.LoadAppVersionsFor*` (cho `AndroidUABuilder`). Cùng file, khác cache. Sửa file → cả 2 pickup (fbdata cần `Reload`, uabuilder tự reload theo mtime).
2. **Hai hệ iOS UA:** `ios_ua.go RandomIPhoneProfile` (path `Config/Settings/useragent/...`) vs `ua_pools.go UAKindIOS` (`Config/UserAgent/iOS_UG.txt`) vs `ios562 buildIOSUA` (build động từ `DeviceInfoIOS/`). Ba code path khác nhau cho 3 luồng khác nhau.
3. **`PC_UG.txt` ≠ `Request_UG.txt`:** `UAKindRequest` map sang `PC_UG.txt` (2026-05). File `Request_UG.txt` còn trong thư mục nhưng không được load.
4. **datr invalid:** mọi append/add skip datr rỗng hoặc bắt đầu `_`/`-` (`validDatr`, pool.go L40-43; cookie.go L151).
5. **`cookieInitialMethod="file"` mà 0 datr → DỪNG hẳn** (app.go L7911-7925) — tránh reg fake không datr.
6. **Steal partition là tối quan trọng** khi nhiều luồng + pool nhỏ (xem §4.2 — bug 30% slot thiếu datr nếu không steal).
7. **Recycle vs Delete datr:** mặc định recycle (dùng vô hạn); chỉ delete khi `DeleteDatrCheckpoint`+`LimitCookieInitial` bật.
8. **`SeedConfigDataIfMissing` KHÔNG seed content** — chỉ tạo thư mục + placeholder. Data device/version/UA do user tự cung cấp (hoặc embedded seed của `cookie`/`fbdata`).
9. **Samsung S-platform filter:** `samsungSPlatform` có danh sách rất dài (s22..s563). Platform nằm trong list → device pool chỉ lấy `samsung/SM-S*`. Nếu `devices.txt` không có SM-S* → fallback `fallbackDevices`.
10. **Pool file mỗi run riêng:** `Pool{YYYYMMDD}_{N}.txt` tăng N mỗi lần Start — không ghi đè run cũ, dễ truy vết datr thu được theo phiên.

---

## 10. Tham chiếu file:line nhanh

| Chủ đề | File:line |
|---|---|
| FBAV/FBBV pool (3 split) | [fbdata/store.go](../../internal/fbdata/store.go) L84-258 |
| RandomFbVersion[Reg/Ver] | [fakeinfo/useragent.go](../../internal/facebook/fakeinfo/useragent.go) L18-41 |
| datr files (initial/pool/run) | [cookie/store.go](../../internal/cookie/store.go) L36-45, L353-374, L546-569 |
| UAOptions / PoolKind | [uabuilder/builder.go](../../internal/facebook/fakeinfo/uabuilder/builder.go) L42-74 |
| data_loader file→type | [uabuilder/data_loader.go](../../internal/facebook/fakeinfo/uabuilder/data_loader.go) L1-371 |
| AndroidUABuilder.Build | [uabuilder/android.go](../../internal/facebook/fakeinfo/uabuilder/android.go) L43-199 |
| BrowserUABuilder.Build | [uabuilder/browser.go](../../internal/facebook/fakeinfo/uabuilder/browser.go) L21-108 |
| ConfigFileUABuilder | [uabuilder/configfile.go](../../internal/facebook/fakeinfo/uabuilder/configfile.go) L22-39 |
| UA pool wiring | [uabuilder_wire.go](../../internal/facebook/fakeinfo/uabuilder_wire.go) L14-38 |
| UA pools (file-based) | [ua_pools.go](../../internal/facebook/fakeinfo/ua_pools.go) L40-202 |
| PartitionedDatrPool | [register/android/pool.go](../../internal/facebook/register/android/pool.go) L161-880 |
| SimProfile/HNI/connType | [fakeinfo/simnetwork.go](../../internal/facebook/fakeinfo/simnetwork.go) L16-99 |
| iOS device/build/locale | [register/ios562/devices.go](../../internal/facebook/register/ios562/devices.go) L37-248 |
| buildIOSUA | [register/ios562/profile.go](../../internal/facebook/register/ios562/profile.go) L106-116 |
| phone_database | [fakeinfo/phonecode.go](../../internal/facebook/fakeinfo/phonecode.go) L42-219 |
| overrides (name/locale/sim) | [fakeinfo/overrides.go](../../internal/facebook/fakeinfo/overrides.go) L42-123 |
| seed dirs | [fakeinfo/seed_config.go](../../internal/facebook/fakeinfo/seed_config.go) L15-69 |
| InteractionConfig struct | [app.go](../../app.go) L4411-4680 |
| startup data wiring | [app.go](../../app.go) L780-824 |
| cookie/datr pool setup | [app.go](../../app.go) L7667-7925 |
