# Thêm Facebook Reg / Verify Version — Runbook

Tài liệu này là **runbook + prompt template** dùng mỗi khi thêm phiên bản Reg hoặc Verify mới (Android FB4A / iOS / Web). Copy-paste prompt mẫu ở mục [§0](#0-prompt-mẫu) cho Claude/AI để tự động hoá toàn bộ flow.

**Cập nhật 2026-05-31**:
- **iOS Native (FBIOS) verify** — thêm vào §11 / §12 / §14 + mục mới §13.6. **RULE**: verify bằng iOS **BẮT BUỘC** user token iOS `EAAAAAY` — token Android `EAAAAU` KHÔNG dùng được. **Reg bằng Android nhưng verify iOS → phải login iOS lấy `EAAAAAY` trước**, không có thì iOS verify không chạy.
- **iOS cellular-only** ([simnetwork.go](../../internal/facebook/fakeinfo/simnetwork.go)): bỏ `"wifi"` khỏi `iosConnTypes` → iOS reg/verify/login LUÔN chạy mạng di động để header `x-fb-sim-hni` luôn được gửi (xem §13.6).
- **iOS CAA login flow** (`send_login_request`) — capture thật trả `EAAAAAY` + cookies; chi tiết header/body + bảng header chuẩn trong §13.6.
- **§13.7 mới — Login-at-verify + Token/Cookie realtime**: MỌI login dồn về VERIFY (reg không login); token ĐÚNG LOẠI theo platform verify (iOS↔`EAAAAAY`, Android↔`EAAAAU`, đối xứng); `preferUserAccessToken` nhận cả 2 prefix; luật hiển thị token/cookie lên UI + KNOWN issue (token chỉ emit lúc verify XONG, chưa realtime giữa verify); message login đồng nhất `[Login][iOS]`/`[Login][Android]`.
- **Doc move**: file này chuyển vào `docs/facebook/` (đường dẫn tham chiếu nội bộ lùi 1 cấp → `../../`).

**Cập nhật 2026-05-28**:
- **Pool FBAV split**: thêm 2 file `versions_and_builds_reg.txt` + `versions_and_builds_ver.txt` ở `Config/Fbapp/` — auto-tạo lúc app start, rỗng theo default → fallback `versions_and_builds.txt` chung ([§1.4](#14-pool-fbav-split--regtxt-và-vertxt-mới-2026-05-28)).
- **UAOptions.PoolKind**: builder honor `"reg"`/`"ver"` để pick pool tương ứng — wire vào toàn bộ 369 file register + verify (script auto-inject).
- **Template UA cho FB4A legacy era**: spec đầy đủ Dalvik + FB_FW/1 + FBDM/ + validator + result propagation pattern dùng lại cho platform mới như s273 ([§13.2.1](#1321-template-ua-cho-fb4a-legacy-era-s273-fbav--555--pattern-dùng-lại)).
- **Validator s273** trong `pickUAForVerifyPlatform`: bắt buộc Dalvik prefix — chặn UA reg s5xx modern pass nhầm gây fail "Invalid Email Domain".

**Cập nhật 2026-05-26**:
- Tách 3 phần: §0 Prompt mẫu, §I Thêm Reg version, §II Verify Platforms reference.
- Move `Config/Fbapp/versions_and_builds.txt` → `Config/DeviceInfo/versions_and_builds.txt` ([§1.1](#11-cấu-trúc-thư-mục-config-mới)). Fbapp/ vẫn được đọc làm fallback (không breaking).
- Tạo thư mục mới: `Config/DeviceInfoIOS/`, `Config/DeviceInfoPC/`.
- Thêm safety net cho scheduler: platform Android-family chưa handle → ERROR explicit thay vì silent cookie fallthrough ([§12 RULE](#12-rule-quan-trọng--token-required-platform-không-được-login-bằng-cookie)).
- **s273 verify rewrite**: bỏ hardcoded device pool, build UA HOÀN TOÀN từ `Config/DeviceInfo/*` ([§13.2](#132-android-fb4a-legacy-rest-s399-s273)). Thêm Step 3.5 post-confirm `/me?fields=email` luôn chạy.
- **Helper mới `DecodeUnicodeEscapes`** trong `verifybase` — decode `\uXXXX` log error_msg Thai/Arabic/CJK ([§16.4](#164-helper-bắt-buộc-cho-log-error)).
- **Checkbox UI mới "Verify lại Unknown ngay"**: pass 2 retry acc unknown với proxy mới (offset workerID slot) ([§15.1](#151-retry-unknown-ngay-pass-2-với-proxy-mới)).
- Đổi label `Kiểm tra Live / Die` → `Kiểm tra sau reg` (đổi vị trí về panel Verify, default ON).
- Ẩn `s399` khỏi UI (DISABLED_PLATFORM_KEYS) — code vẫn còn nguyên, kích hoạt lại bằng xoá 1 dòng ([§11 Platform ẩn](#platform-ẩn--khoá-tạm)).
- Tool mới: `cmd/check_verified_email` — kiểm tra acc trong `SuccessVerify_No2FA.txt` thật sự verified hay không ([§17](#17-công-cụ-kiểm-tra-acc-đã-ghi-debug-tool)).

---

## 0. Prompt mẫu

Copy 1 trong 2 đoạn dưới đây vào chat với Claude/AI khi cần thêm version mới.

### 0.1 Thêm Reg version Android

```
Tôi sẽ thêm Reg Android FB4A version <VERSION_FBAV>, BuildNumber <BUILD>, device <DEVICE>.
Hãy đọc docs/facebook/add-facebook-reg-version.md và thực hiện toàn bộ §I (mục 1-10):
- Cập nhật Config/DeviceInfo/versions_and_builds.txt (thêm dòng <FBAV>|<BUILD>)
- Tạo package internal/facebook/register/s<XXX>/ (copy từ s<base>)
- Khai báo PlatformS<XXX> trong factory.go
- Nối vào app.go (isRegPlatformXxx, originalUABaseForPlatform, worker context)
- Cập nhật InteractionSetupPage.vue (ORIGINAL_UA_STRINGS, REG_PLATFORM_LABELS, REG_PLATFORMS_VER)
- Cập nhật mock preview verify-runner.mock.ts
- Build backend (go build ./...) và frontend (npm run build) — KHÔNG được skip
- Check checklist §10 trước khi báo xong
```

### 0.2 Thêm Verify version Android

```
Tôi sẽ thêm Verify Android FB4A version <VERSION_FBAV>, endpoint <Bloks/REST>.
Hãy đọc docs/facebook/add-facebook-reg-version.md và thực hiện toàn bộ §II (mục 11-18):
- Tạo package internal/facebook/verify/s<XXX>/ (verify.go + helpers.go)
- Khai báo PlatformS<XXX> trong factory.go
- Thêm blank import vào app.go
- Thêm vào Android-family switch case trong scheduler.go (BẮT BUỘC — quên = scheduler báo FATAL)
- Thêm vào platformNeedsAndroidLoginToken trong app.go
- Thêm case expectedFBAVPrefix vào pickUAForVerifyPlatform
- Confirm response CHỈ accept positive marker ("true" / "result":true) — KHÔNG dùng negative check
- Post-confirm verify GET /me?fields=email với token (LUÔN chạy, không phụ thuộc CheckLiveDie)
- Build UA đọc từ Config/DeviceInfo/* (RandomFbVersion + RandomDeviceProfile + PickCountryCarrierLocale)
- Log error: dùng verifybase.DecodeUnicodeEscapes + truncate theo rune (§16.4)
- Cập nhật UI buttons + UA preview
- Build backend + frontend
- Test bằng cmd/check_verified_email với file SuccessVerify_No2FA.txt sau khi chạy thử
- Check checklist §16.5 trước khi báo xong
```

---

# Phần I — Thêm Reg Version mới

## Nguyên tắc

- Không để file sinh hàng loạt ở root nếu không thật sự cần. Nếu cần registry/helper lớn, tách thành file có tên rõ nghĩa theo chức năng, hoặc gom vào package/phạm vi đang có.
- Mỗi version Reg phải có cặp `FBAV/FBBV` đúng với device/platform đang khai báo.
- Nếu là S23 thì UA gốc phải giữ `FBDV/SM-S911B`, `FBSV/15`, `FBCA/arm64-v8a:;`.
- `UA Gốc` phải có kết quả cố định theo version. `Thay nhà mạng` chỉ được thay `FBCR`, không đổi `FBAV/FBBV/device`.
- Sau khi sửa logic phải test cả backend và frontend build.

## 1. Cập nhật file version/build

### 1.1 Cấu trúc thư mục Config mới

Từ 2026-05-26, các file device info được tổ chức theo platform:

```text
Config/
├── DeviceInfo/                       ← Android pool (FB4A native)
│   ├── versions_and_builds.txt       ← FBAV|FBBV pool (move từ Config/Fbapp/)
│   ├── devices.txt                   ← FBDV (Samsung SM-S911B, SM-G998B, ...)
│   ├── carriers.txt                  ← FBCR (Viettel, MobiFone, ...)
│   ├── locales.txt                   ← FBLC (en_US, vi_VN, ...)
│   ├── os_versions.txt               ← Android version (FBSV)
│   ├── densities.txt                 ← FBDM density
│   ├── screen_resolution.txt         ← FBDM width × height
│   ├── chrome_versions.txt           ← Chrome version (cho WebAndroid)
│   ├── buildnums.txt
│   ├── device_build_nums.txt
│   ├── device_cores.txt
│   ├── devices_versions.txt
│   └── googleapp_versions.txt
│
├── DeviceInfoIOS/                    ← (chuẩn bị) iOS pool
│   └── (TBD: ios_versions.txt, iphone_devices.txt, ...)
│
├── DeviceInfoPC/                     ← (chuẩn bị) PC/desktop pool
│   └── (TBD: pc_chrome_versions.txt, pc_os_versions.txt, ...)
│
└── Fbapp/                            ← Fallback path cho user CHƯA migrate
    └── versions_and_builds.txt       ← chỉ đọc khi DeviceInfo/ không có
```

**Trạng thái hiện tại** (2026-05-26 — đã migrate):
- Code đọc `versions_and_builds.txt` theo thứ tự ưu tiên (chỉ 2 path, đã bỏ per-platform variant — gom chung 1 file cho mọi Android FB4A) — xem [data_loader.go LoadAppVersionsForPlatform](../../internal/facebook/fakeinfo/uabuilder/data_loader.go) + [fbdata/store.go DefaultVersionsAndBuildsPath](../../internal/fbdata/store.go):
  1. `Config/DeviceInfo/versions_and_builds.txt` — **path khuyến nghị** (ưu tiên)
  2. `Config/Fbapp/versions_and_builds.txt` — fallback cho user cũ chưa migrate
- 3 folder seed trong [seed_config.go:21-34](../../internal/facebook/fakeinfo/seed_config.go#L21-L34) — tự tạo khi app start: `DeviceInfo/`, `DeviceInfoIOS/`, `DeviceInfoPC/`.
- **Migration step (user)**: copy/move file từ `Config/Fbapp/versions_and_builds.txt` sang `Config/DeviceInfo/versions_and_builds.txt`. Code tự ưu tiên path mới; nếu khách hàng chưa migrate, code vẫn đọc fallback từ `Fbapp/`. **Không breaking change**.

### 1.2 Định dạng `versions_and_builds.txt`

File generic chia sẻ cho TẤT CẢ Android FB4A platforms (không split per-version):

```txt
560.0.0.57.63|963497253
559.0.0.59.104|891221994
273.0.0.39.123|218047977
...
```

Mỗi dòng: `FBAV|FBBV`. Build UA random pick 1 dòng từ pool này.

### 1.3 UA modes — 3 cách build UA

| Mode | Nguồn | Behaviour |
|---|---|---|
| **UA Gốc** | Hardcoded per-platform trong `originalUABaseForPlatform(...)` | Trả nguyên UA gốc của capture. Toggle `Thay nhà mạng` chỉ thay `FBCR` (carrier), giữ nguyên `FBAV/FBBV/FBDV/FBSV`. Toggle `Add virtual spec` thêm field fingerprint. |
| **Build UA** | Random từ `Config/DeviceInfo/*` + split pool `versions_and_builds_reg/ver.txt` | Pick random `FBAV|FBBV` từ pool split (fallback chung) + random device từ `devices.txt` + random carrier/locale theo country. |
| **Original (account)** | `session.UserAgent` từ acc import | Dùng nguyên UA đã có sẵn trong account. Chỉ regen nếu UA invalid (không khớp FBAV expected). |

iOS / PC sẽ dùng tương tự nhưng đọc từ `Config/DeviceInfoIOS/` và `Config/DeviceInfoPC/` (chưa implement).

### 1.4 Pool FBAV split — `_reg.txt` và `_ver.txt` (mới 2026-05-28)

Để REG và VER có pool FBAV ĐỘC LẬP (vd: pool ver chỉ s273/547/551 cho legacy verify, pool reg chỉ 562/563/564 cho native register), app hỗ trợ 3 file song song:

```text
Config/DeviceInfo/
├── versions_and_builds_reg.txt    ← Pool REG (auto-tạo lúc app start, rỗng nếu chưa có)
└── versions_and_builds_ver.txt    ← Pool VER (auto-tạo lúc app start, rỗng nếu chưa có)

Config/Fbapp/  (legacy fallback)
└── versions_and_builds.txt        ← Pool CHUNG (fallback khi _reg/_ver rỗng)
```

> **Note**: file pool CHUNG `versions_and_builds.txt` được đọc theo thứ tự ưu tiên `Config/DeviceInfo/` → `Config/Fbapp/` (legacy). User có thể di chuyển sang DeviceInfo hoặc giữ ở Fbapp đều OK.

**Logic load** ([fbdata/store.go](../../internal/fbdata/store.go)):
- `RandomFbVersionReg()` → đọc `_reg.txt` nếu có data → fallback `versions_and_builds.txt` chung
- `RandomFbVersionVer()` → đọc `_ver.txt` nếu có data → fallback `versions_and_builds.txt` chung
- `RandomFbVersion()` → luôn pool chung (giữ backward-compat preview/internal)

**Wire vào builder** ([uabuilder/builder.go UAOptions](../../internal/facebook/fakeinfo/uabuilder/builder.go)):
```go
opts := uabuilder.UAOptions{
    PoolKind: "reg",  // "reg" / "ver" / "" → builder pick pool tương ứng
    CountryCode: "VN",
}
```

→ Tất cả 369 file `register/sXXX/profile.go` + `verify/sXXX/steps.go` đã được wire `PoolKind: "reg"|"ver"` (script `inject_poolkind.ps1` tự động patch). Khi thêm platform mới, **caller phải set `PoolKind`** trong UAOptions (REG = "reg", VER = "ver").

**Direct loader callers** trong register profile.go (build UA inline không qua AndroidUABuilder):
```go
// PHẢI dùng LoadAppVersionsForReg (KHÔNG phải LoadAppVersionsForPlatform).
if vers, err := uabuilder.LoadAppVersionsForReg(); err == nil && len(vers) > 0 {
    av := vers[r.Intn(len(vers))]
    fbav, fbbv = av.Version, av.Build
}
```

**Direct `fakeinfo.RandomFbVersion()` callers** (s273, app.go reg BuildUA path, scheduler.go ver BuildUA):
- REG path → `fakeinfo.RandomFbVersionReg()`
- VER path → `fakeinfo.RandomFbVersionVer()`

**Test button preview** ([app.go SimulatePlatformUA](../../app.go)):
- Frontend pass `kind: 'reg'` hoặc `kind: 'ver'` trong `PlatformUAConfig` (field `Kind string \`json:"kind"\``).
- Backend pick pool qua helper `pickFbVersionByKind(uaCfg.Kind)`.

**Khuyến nghị format file `_reg.txt` / `_ver.txt`** (mỗi dòng 1 cặp, dấu `#` = comment):
```text
# Pool reg cho native register (s562/s563/s564)
562.0.0.51.73|976057955
563.0.0.42.67|976056141
564.0.0.0.17|977893103
```

**Hot reload**: file CHUNG cache qua `sync.Once` (cần restart app sau edit). File split cache theo `mtime` (auto reload). Để force reload tất cả, gọi `fbdata.Reload("")`.

## 2. Tạo package Reg backend

Thư mục cần có:

```text
internal/facebook/register/sXXX/
```

Thường copy từ version gần nhất đang chạy ổn định, ví dụ:

```text
internal/facebook/register/s560/
```

Sau khi copy, sửa tối thiểu:

- `package sXXX` trong tất cả file Go của thư mục mới.
- `body.go`:
  - `OriginalUA`
  - comment platform nếu có.
  - các constant/body field có gán version/build nếu package cũ có.
- `register.go`:
  - tên field/profile UA nếu có dạng `s560UA`, `s558UA`.
  - `UserAgent()` phải trả đúng UA của package mới.
- `profile.go`:
  - struct/profile field liên quan UA nếu có.
- Bất kỳ file nào trong package có chuỗi `s560`, `S560`, `560`, `OriginalUA`.

Lệnh kiểm tra nhanh:

```powershell
rg -n "s560|S560|560\.|OriginalUA|FBAV|FBBV" internal/facebook/register/sXXX
```

## 3. Khai báo platform backend

File cần sửa: `internal/facebook/factory*.go`

Thêm constant:

```go
PlatformSXXX = "sXXX"
```

Nếu project đang chia nhóm factory, đặt vào file nhóm version tương ứng, ví dụ `factory_s500_s544.go`.

## 4. Nối package mới vào app logic

File cần sửa: `app.go` + helper/registry của nhóm version.

Cần đảm bảo các điểm sau nhận ra version mới:

- Hàm check platform Reg: `isRegPlatform500To544`, `isRegPlatform545To554`, hoặc hàm nhóm mới.
- Pool datr/cookie: hàm trả pool theo platform + register/unregister khi start/stop.
- Worker context: `newReg...WorkerContext(...)` + import package `internal/facebook/register/sXXX`.
- Route register: nhận đúng worker context khi `apiRegPlatform == sXXX` + gọi `Register(...)`.
- UA gốc: `originalUABaseForPlatform(...)` hoặc helper nhóm.
- Token/login/verify liên quan UA: Reg dùng Reg UA khi lấy token; Verify dùng Verify UA riêng (không ghi đè Reg).
- **warmSession**: **KHÔNG** gọi trong platform Android — warmSession login qua `m.facebook.com` (web cookie), sai luồng. Chỉ giữ nếu package cũ đã ổn; package mới KHÔNG copy vào.

Lệnh tìm:

```powershell
rg -n "isRegPlatform|newReg.*WorkerContext|originalUABaseForPlatform|PoolPointers|Register\(|UseOriginalUA|SetUA\(" app.go *.go
```

## 5. Cập nhật giao diện chọn version

File: `frontend/src/pages/InteractionSetupPage.vue`

Thêm version mới vào:

- `ORIGINAL_UA_STRINGS` — dùng chính xác UA gốc của backend.
- `REG_PLATFORM_LABELS` — tên hiển thị, ví dụ `S23 (Fb_560v3)`.
- Danh sách button Reg: `REG_PLATFORMS_VER`.
- Nếu version mới cũng dùng cho Verify thì thêm vào danh sách Verify tương ứng.

Lưu ý:

- Nút chọn platform phải gọi hàm select có `saveNow()`, không chỉ đợi state.
- Checkbox `UA Gốc`, `Build UA`, `Virtual spec`, `Thay nhà mạng` phải lưu ngay khi đổi.
- Preview UA phải lấy đúng config per-platform.

Lệnh tìm:

```powershell
rg -n "ORIGINAL_UA_STRINGS|REG_PLATFORM_LABELS|REG_PLATFORMS|selectRegPlatform|selectVerifyPlatform|simulatePlatformUA" frontend/src/pages/InteractionSetupPage.vue
```

## 6. Cập nhật mock preview frontend

File: `frontend/src/bridge/mock/verify-runner.mock.ts`

Không để fallback về version cũ như `559`. Nếu mock chưa có platform, báo thiếu rõ ràng:

```ts
return `Mock chưa có UA gốc cho platform ${platform}`
```

## 7. Kiểm tra version không bị lạc UA

Test 3 luồng:

- **Reg only**: chọn `apiRegPlatform = sXXX`, bật `UA Gốc`, bật/tắt `Thay nhà mạng`. Kết quả phải có `FBAV/FBBV` của `sXXX`.
- **Reg + Verify, không Split**: Reg UA dùng `sXXX`, Verify UA dùng platform Verify đang chọn.
- **Reg + Verify + Split**: Reg side không bị Verify UA ghi đè.

Lệnh tìm chuỗi version sai:

```powershell
rg -n "FBAV/559|959738728|s559|s559v2" app.go internal frontend/src Config
```

## 8. Test bắt buộc

```powershell
go test ./...
```

```powershell
npm run build
```

Build/test fail = chưa xong, không được claim xong version.

## 9. Checking UA (thêm XID vào cuối UA)

`Checking UA` thêm `XID/<random16>;` trước dấu `]` cuối UA.

```
[FBAN/FB4A;FBAV/560.0.0.57.63;...;FBCA/arm64-v8a;]
                    ↓ bật Checking UA
[FBAN/FB4A;FBAV/560.0.0.57.63;...;FBCA/arm64-v8a;XID/f458iwz5cu3hkl1t;]
```

Điểm triển khai:

- **Backend** — `PlatformUAConfig.CheckingUA bool`, `InteractionConfig.CheckingUA bool`, `applyRegPlatformUAConfig` / `applyVerifyPlatformUAConfig` copy field, `appendXIDToUA(ua) string` helper, áp dụng XID trong `SimulatePlatformUA` + `pickUAForVerifyPlatform` + reg worker (sau block `KeepUASuccess`).
- **Frontend** — interface `PlatformUAConfig` thêm `checkingUA: boolean`, checkbox `Checking UA` trong cả Reg và Ver section, `saveNow()` ngay khi đổi.

Lưu ý:
- XID sinh mới mỗi request (`rand.Intn`), không lưu lại.
- Checkbox độc lập với `UA Gốc`, `Build UA`, `Virtual spec`, `Thay nhà mạng`.

## 10. Checklist Reg (trước khi báo xong)

- [ ] Config version/build đã có dòng mới.
- [ ] Package `internal/facebook/register/sXXX` tồn tại.
- [ ] `OriginalUA` đúng `FBAV/FBBV/device/arm64-v8a`.
- [ ] Platform constant đã khai báo.
- [ ] App logic nhận platform mới.
- [ ] Pool/datr/cookie route đúng platform mới.
- [ ] Worker context mới được tạo và gọi `Register`.
- [ ] `originalUABaseForPlatform` trả được UA gốc.
- [ ] UI có nút version mới.
- [ ] UI preview có UA mới.
- [ ] Mock không fallback về version cũ.
- [ ] `go test ./...` pass.
- [ ] `npm run build` pass.

---

# Phần II — Verify Platforms Reference

Phần này mô tả các loại verify platform, mỗi loại cần auth gì, có lớp chống false-positive ra sao. Cập nhật mỗi lần thêm verify platform mới.

## 11. Tổng quan các loại Verify

4 họ chính, khác nhau ở **auth + endpoint + UA**:

| Họ | Platform key | Auth | Endpoint chính | UA bắt buộc | Session FB log |
|---|---|---|---|---|---|
| **Android FB4A (modern Bloks)** | `s23`, `s415`–`s563*`, `android` (= api token) | EAA token (`EAAAAU...`) | `b-graph.facebook.com/graphql` (Bloks `doc_id`) | Dalvik FB4A + FBAV khớp version | FB4A `<device>` |
| **Android FB4A (legacy REST)** | `s399` _(hidden UI 2026-05-26)_, `s273` | EAA token (`EAAAAU...`) | `b-graph.facebook.com/auth/login` (lấy token), `b-api.facebook.com/method/user.*` (s273) | Dalvik FB4A + FBAV khớp version | FB4A `<device>` |
| **WebAndroid (Chrome Mobile)** | `webandroid` (= api web andr) | Cookie session (`c_user; xs; datr; fr`) | `m.facebook.com/setemail`, `m.facebook.com/confirmation_cliff` | Chrome Mobile (pool ~116k combinations) | Chrome `<version>` browser |
| **Web MFB** | `web` (= api mfb) | Cookie session + `fb_dtsg` / `jazoest` | `m.facebook.com/changeemail` (form-encoded) | Chrome Mobile | Chrome browser |
| **iOS Native (FBIOS)** | `ios562`, `ios563` | **User token iOS `EAAAAAY...` BẮT BUỘC** (token Android `EAAAAU` KHÔNG dùng được) | `graph.facebook.com/graphql` (Bloks CAA iOS `doc_id`) | FBIOS (`FBAN/FBIOS`) khớp version | FBIOS `<iPhone>` |

Map UI dropdown → platform string: `verifyPlatformFromType(...)` trong [app.go](../../app.go).

### Platform ẩn / khoá tạm

Platform có trong code nhưng ẩn khỏi UI:

| Platform | Trạng thái | Lý do | Kích hoạt lại |
|---|---|---|---|
| `s399` | hidden (DISABLED_PLATFORM_KEYS) | Tạm khoá UI 2026-05-26 — code, package, scheduler, factory vẫn đầy đủ; chỉ ẩn button | Xoá `'s399'` khỏi `DISABLED_PLATFORM_KEYS` trong [InteractionSetupPage.vue:~816](../../frontend/src/pages/InteractionSetupPage.vue) |

## 12. RULE QUAN TRỌNG — Token-required platform KHÔNG được login bằng cookie

**Đây là bug recurring** trước đây làm "verify android nhưng FB log Chrome 118". Đã fix bằng safety net 2026-05-26.

### Rule

| Platform | Login mode | Lý do |
|---|---|---|
| `s23`, `s415`-`s563*`, `android`, `s399`, `s273` (Android-family) | **REST `/auth/login` lấy EAA token** | Endpoint b-graph/b-api dùng `Authorization: OAuth <EAA>` — không phải cookie |
| `webandroid` | **Skip login, dùng cookie trực tiếp** | Endpoint `m.facebook.com` dùng cookie + Chrome Mobile UA |
| `web` (api mfb) | **`LoginWithCookieMobile` (m.facebook.com)** | Cần parse `fb_dtsg` / `jazoest` từ HTML response |
| `ios562`, `ios563` (iOS native) | **iOS login lấy user token `EAAAAAY`** (KHÔNG dùng `EAAAAU` Android) | Endpoint Bloks CAA iOS chỉ chấp nhận user token iOS `EAAAAAY`; `EAAAAU` từ `/auth/login` Android KHÔNG verify được. Reg Android → ver iOS phải login iOS trước. |

### Triển khai (3 điểm code)

1. **scheduler.go switch case Android-family** ([scheduler.go:524-565](../../internal/runner/scheduler.go#L524-L565)) — liệt kê đầy đủ TẤT CẢ Android-family platform:

```go
switch verifyPlatform {
case facebook.PlatformS22, ..., facebook.PlatformS563S21,
    facebook.PlatformS399,
    facebook.PlatformS273:   // ← Quên add = bug cookie fallthrough!
    // → Fetch EAA token via REST /auth/login (FetchAndroidTokenLegacy)

case facebook.PlatformWebAndroid:
    // → Skip login (dùng cookie trực tiếp)

case facebook.PlatformWeb:
    // → LoginWithCookieMobile (parse fb_dtsg)

default:
    // → ERROR explicit: platform chưa được handle, BUG.
    //   Trước 2026-05-26 default = cookie login → silent bug.
}
```

2. **`platformNeedsAndroidLoginToken`** ([app.go:435-448](../../app.go#L435-L448)) — liệt kê đầy đủ Android-family.

3. **`pickUAForVerifyPlatform`** ([app.go:10773-10880](../../app.go#L10773-L10880)) — `expectedFBAVPrefix = "FBAV/XXX."` để validator buộc UA khớp version.

### Safety net (mới 2026-05-26)

`default` case trong scheduler đã đổi từ **silent cookie fallthrough** → **explicit ERROR**:

```go
default:
    notify("[FATAL] Platform 'sXXX' chưa được handle trong scheduler switch...")
    result.Status = "error"
    goto done
```

→ Nếu thêm Android platform mới mà quên add vào case Android-family, scheduler sẽ **báo lỗi rõ ràng** thay vì âm thầm verify bằng cookie web. Không còn "FB log Chrome 118 dù chọn Android" nữa.

## 13. Chi tiết auth + flow từng loại

### 13.1 Android FB4A modern Bloks (`s23`, `s415`–`s563*`, `android`)

**Auth**: EAA `access_token`. Truyền qua header `Authorization: OAuth <token>` HOẶC body `access_token=...`.

**Token lấy từ đâu**:
- Reg trực tiếp trả `EAAAAU...` (most cases).
- Reg không trả (Web / WebAndroid reg) → scheduler tự gọi `FetchAndroidTokenLegacy` (REST `/auth/login`) trước khi vào verifier. UA dùng default `androidUA` (FBAV/518 FB4A) — KHÔNG dùng session.UserAgent vì có thể Chrome.

**UA**: phải khớp FBAV của version (`FBAV/555.` cho `s555`, etc.). Validator: `expectedFBAVPrefix` trong `pickUAForVerifyPlatform`.

**Flow chung** (`verifybase.RunVerify`):
1. Inject cookies vào tls client (chỉ để match session fingerprint — KHÔNG bắt buộc cho token).
2. Build `nt_context` + session ctx với pinned ID (`device_id`, `family_device_id`).
3. POST Bloks AddEmail → wait OTP → POST Bloks Confirm.
4. (Optional) 2FA enable, PostConfirm hooks.
5. Live/Die loop nếu `cfg.CheckLiveDie = true`.

### 13.2 Android FB4A legacy REST (`s399`, `s273`)

Khác `s5xx`: dùng REST classic, không Bloks/GraphQL. Stable hơn vì không phụ thuộc Bloks schema rotation.

**Endpoint**:
- `s399`: POST `b-graph.facebook.com/auth/login`, POST `b-graph.facebook.com/me/changeemail` flow.
- `s273`: POST `b-api.facebook.com/method/user.editregistrationcontactpoint` (add), `b-api.facebook.com/method/user.confirmcontactpoint` (confirm), `b-api.facebook.com/method/user.sendconfirmationcode` (resend).

**UA**:
- `s399`: Dalvik FB4A `FBAV/399.` device S23. Locale/carrier theo country pool (`PickCountryCarrierLocale`).
- `s273` (rewrite 2026-05-26): build UA **HOÀN TOÀN** từ `Config/DeviceInfo/*` (không hardcode pool):
  - **FBAV/FBBV** → `fakeinfo.RandomFbVersion()` đọc từ `Config/DeviceInfo/versions_and_builds.txt`
  - **Device** (FBMF/FBBD/FBDV/FBSV/FBCA/Build) → `fakeinfo.RandomDeviceProfile()` đọc từ `devices.txt` + `buildnums.txt` + `os_versions.txt`
  - **FBDM** (density + screen) → từ device profile (`densities.txt` + `screen_resolution.txt`)
  - **FBLC** (locale) → `PickCountryCarrierLocale(countryCode)` theo IP, fallback `LocaleFromCountry()` hoặc `"en_US"`
  - **FBCR** (carrier) → country pool theo IP, fallback `RandomCarrier()`
  - **Format**: giữ structure capture (`Dalvik/...` prefix + `FB_FW/1` + `FBDM/{...}` dấu `/`)
  - → Thêm/update FBAV version mới: chỉ cần append 1 dòng vào `versions_and_builds.txt` — KHÔNG đụng code. Xem [s273/helpers.go RandomUA](../../internal/facebook/verify/s273/helpers.go).

**Post-confirm verify (s273 only — Step 3.5)** ([verify.go:328-345](../../internal/facebook/verify/s273/verify.go)):
- GET `graph.facebook.com/me?fields=id,email&access_token=<EAA>` với cookie jar rỗng.
- Email field non-empty → Live thực sự.
- Error 459 / 190 / `OAuthException` → DIE (catch checkpoint sau confirm).
- Có ID không có email → silent fail → error.
- **Chạy LUÔN** không phụ thuộc `cfg.CheckLiveDie`. Chỉ 1 GET nhẹ. 100% token-based, KHÔNG cookie.

### 13.2.1 Template UA cho FB4A legacy era (s273, FBAV ≤ 555) — pattern dùng lại

Đây là **spec UA chuẩn** cho mọi platform thuộc era legacy REST API (cũ hơn Bloks modern). Khi thêm platform tương tự s273 (vd: s4xx-s555 với endpoint `b-api.facebook.com/method/user.*`), follow template này.

#### Format UA chuẩn

```
Dalvik/2.1.0 (Linux; U; Android <OS>; <Model> Build/<BuildID>) [FBAN/FB4A;FBAV/<>;FBPN/com.facebook.katana;FBLC/<locale>;FBBV/<>;FBCR/<carrier>;FBMF/<mfg>;FBBD/<brand>;FBDV/<Model>;FBSV/<OS>;FBCA/<arch>;FBDM/{density=<>,width=<>,height=<>};FB_FW/1;FBRV/0;]
```

**ĐẶC ĐIỂM PHÂN BIỆT với format MODERN (s5xx Bloks)**:

| Field | LEGACY (s273-era) | MODERN (s5xx Bloks) |
|---|---|---|
| Dalvik prefix | ✅ **Bắt buộc** | ❌ Không có |
| `FB_FW/1` | ✅ Có | ❌ Không |
| `FBOP/1` | ❌ Không | ✅ Có |
| `FBDM` separator | `FBDM/{...}` (slash) | `FBDM={...}` (equal) |
| `;]` trailing | ✅ Có | ❌ Không |
| Field order | `FBPN` ngay sau `FBAV` | `FBPN` đặt cuối block FBDV |
| `FBCA` format | `arm64-v8a` thuần (anh user yêu cầu 2026-05-28 bỏ `:null`) | `arm64-v8a:` (trailing colon) |
| `FBLC` | Theo country phone (vi_VN/en_US/...) | Cố định `en_GB` |
| `FBSV` | Random từ os_versions.txt | Cố định `15` |

→ **Mismatch format = FB từ chối với `"Invalid Email Domain"` (legacy endpoint detect bot).** Phải dùng đúng format CỦA ERA endpoint.

#### Template code `RandomUA(countryCode)` cho legacy era

Reference: [s273/helpers.go RandomUA](../../internal/facebook/verify/s273/helpers.go).

```go
func RandomUA(countryCode string) string {
    // 1. FBAV/FBBV — pool VER (split file mới hoặc fallback chung)
    fbVer, fbBuild := fakeinfo.RandomFbVersionVer()
    
    // 2. Device random từ devices.txt (Model/Brand/Mfg + spec)
    device := fakeinfo.RandomDeviceProfile()

    // 3. Locale + carrier theo country phone (smart picker)
    locale, carrier := "", ""
    if countryCode != "" {
        locale, carrier = verifybase.PickCountryCarrierLocale(countryCode)
    }
    if locale == "" {
        locale = fakeinfo.LocaleFromCountry(countryCode)
    }
    if locale == "" {
        locale = "en_US"
    }
    if carrier == "" {
        if countryCode == "" {
            carrier = "Viettel"  // user không tick "Thay nhà mạng" → giữ default
        } else {
            carrier = fakeinfo.RandomCarrier()  // fallback global
            if carrier == "" {
                carrier = "Viettel"
            }
        }
    }

    // 4. Build ID từ buildnums.txt (Android build ID THẬT, không compose "<brand>-<model>")
    buildID := device.BuildID
    if buildID == "" {
        buildID = device.Brand + "-" + device.Model
    }

    // 5. FBCA: arch thuần (KHÔNG append :null)
    fbca := device.Architecture

    // 6. Compose UA — order chuẩn legacy
    return fmt.Sprintf(
        "Dalvik/2.1.0 (Linux; U; Android %s; %s Build/%s) "+
            "[FBAN/FB4A;FBAV/%s;FBPN/com.facebook.katana;FBLC/%s;FBBV/%s;FBCR/%s;"+
            "FBMF/%s;FBBD/%s;FBDV/%s;FBSV/%s;FBCA/%s;"+
            "FBDM/{density=%s,width=%d,height=%d};FB_FW/1;FBRV/0;]",
        device.OSVersion, device.Model, buildID,
        fbVer, locale, fbBuild, carrier,
        device.Manufacturer, device.Brand, device.Model, device.OSVersion,
        fbca,
        device.Density, device.ScreenWidth, device.ScreenHeight,
    )
}
```

#### Validator `IsPoolUA` — bảo vệ session UA

Trong verify.go, KHI account đã có `session.UserAgent` (từ reg) → check format trước khi quyết định regen:

```go
// IsPoolUA — UA hợp lệ cho s273 = FB4A + có Dalvik prefix.
// UA reg s5xx modern (không Dalvik) sẽ fail → s273 verify auto regen UA mới.
func IsPoolUA(ua string) bool {
    return strings.Contains(ua, "FBAN/FB4A") && strings.Contains(ua, "Dalvik/")
}

// Verify worker:
verifyUA := session.UserAgent
if !IsPoolUA(verifyUA) {
    country := verifybase.CountryFromPhone(session.Phone)
    verifyUA = RandomUA(country)
}
```

→ Account reg bằng s562 (UA modern không Dalvik) chuyển sang verify s273 → `IsPoolUA` return false → regen UA Dalvik mới. Tránh FB detect format mismatch endpoint.

#### Validator trong `pickUAForVerifyPlatform` — đặc biệt cho legacy era

[app.go pickUAForVerifyPlatform](../../app.go) có path "ưu tiên dùng UA cũ nếu valid" — phải **strict** với platform legacy:

```go
if verifyPlatform == facebook.PlatformS273 {
    isValidUA = func(ua string) bool {
        // BẮT BUỘC Dalvik prefix — tránh UA reg s5xx (modern không Dalvik) pass nhầm.
        if !strings.Contains(ua, "Dalvik/") {
            return false
        }
        // Accept multi-FBAV trong pool legacy era (273/547/551/555).
        return strings.Contains(ua, "FBAV/273.") ||
            strings.Contains(ua, "FBAV/547.") ||
            strings.Contains(ua, "FBAV/551.") ||
            strings.Contains(ua, "FBAV/555.")
    }
}
```

→ Quên check `Dalvik/` → UA reg s562 `[FBAN/FB4A;FBAV/547...]` (rác từ pool reg) pass validator → verify dùng UA modern → fail.

#### Result propagation — `UserAgent: verifyUA` vào MỌI early return

VerifyResult phải có `UserAgent` field ở mọi return (kể cả early fail) để `acc.UserAgent` được update đúng:

```go
// SAI — UA không propagate, acc giữ UA reg cũ (modern không Dalvik) → UI hiển thị sai:
return &facebook.VerifyResult{Status: "error", Message: "...", Email: tempEmail}

// ĐÚNG — UA Dalvik propagate, acc.UserAgent được update:
return &facebook.VerifyResult{Status: "error", Message: "...", Email: tempEmail, UserAgent: verifyUA}
```

S273 có **18 early return** đều cần `UserAgent: verifyUA`. PowerShell regex auto-patch:
```powershell
[regex]::Replace($content, '(&facebook\.VerifyResult\{[^}]*Email:\s*tempEmail)(\})', { ... })
```

#### Bug recurring — pool versions_and_builds rác

**Triệu chứng**: app fallback hardcoded `554.0.0.57.70` cho mọi account dù pool có 17 dòng.

**Nguyên nhân**: file `versions_and_builds.txt` chứa dòng RÁC `1.0.0.0.0.1|2313232` hoặc `2.2.0.0.0.1|2313232` (test data sót lại). Pool parse cả 4 dòng → 50% UA có FBAV/1.0 hoặc 2.2 (rác) → FB detect bot.

**Fix**: xóa rác trong pool, chỉ giữ FBAV thật. Pool tốt nhất chỉ chứa version đã test work với endpoint era (vd legacy: 273-555).

### 13.3 WebAndroid (Chrome Mobile, cookie-based)

**Auth**: cookie session (`c_user`, `xs`, `datr`, `fr`). UID khớp `c_user`.

**Token**: KHÔNG cần. `CheckLiveDieCombined` có thể truyền nhưng không bắt buộc.

**Endpoint**: `m.facebook.com` (mobile web). Form-encoded HTML/JSON.

**UA**: Chrome Mobile bắt buộc. Random từ pool ~116k combinations (`fakeinfo.RandomChromeAndroidProfile()`). UA Android FB4A bị FB từ chối.

**Flow** ([webandroid/verify.go](../../internal/facebook/verify/webandroid/verify.go)):
1. GET `m.facebook.com/changeemail` (lấy state token) → POST `setemail` với cookie.
2. Wait OTP.
3. POST `confirmation_cliff` với cookie + OTP + state.
4. (Optional) Enable2FA qua AccountsCenter.
5. CheckLiveDie loop nếu `cfg.CheckLiveDie = true`.
6. **Post-confirm pending check** (chỉ khi `cfg.CheckLiveDie = true`): GET `m.facebook.com/` với cookie → redirect `/confirmemail.php` (chưa confirm thực sự) hoặc `/checkpoint/` (acc lock) → demote Live → Die.

**Điểm yếu**: `detectPendingOrCheckpoint` nằm trong `if cfg.CheckLiveDie` → tắt checkbox = bỏ qua pending check = có thể false positive.

### 13.4 Web MFB (cookie + fb_dtsg)

**Auth**: cookie + `fb_dtsg` / `jazoest` (CSRF token lấy từ HTML `m.facebook.com` sau khi login bằng cookie).

**Bắt buộc login trước**: `facebook.LoginWithCookieMobile(ctx, session)` ([login.go](../../internal/facebook/login.go)) — GET `m.facebook.com/login` với cookie, parse `fb_dtsg`.

**Endpoint**: `m.facebook.com/changeemail` form POST.

**UA**: Chrome Mobile (giống WebAndroid).

### 13.5 Datr pool — nguồn machine_id chuẩn cho mọi platform

**Datr là gì**: chuỗi 24 ký tự FB gắn vào browser/app khi visit lần đầu (cookie `datr=xxx`). Dùng làm `machine_id` trong fingerprint → FB tracking và risk scoring. Nếu reg/verify dùng datr "lạ" hoặc rỗng → FB nghi ngờ.

**Quan trọng**: datr tốt → tỉ lệ checkpoint thấp; datr cũ/đã dùng nhiều → tỉ lệ checkpoint cao. Đây là asset valuable, cần manage cẩn thận.

**Pool location** ([internal/cookie/store.go](../../internal/cookie/store.go)):

```text
Config/Cookie/
├── cookie_initial.txt    ← user paste vào, INPUT cho reg (1 dòng = 1 datr/cookie đầy đủ)
└── datr_pool.txt          ← OUTPUT, tự tích lũy datr mới từ reg thành công (dedup)
```

Cả 2 file: 1 dòng = 1 entry. Format chấp nhận:
- Datr trần: `OHILajGNGFUlFciYOta5HkHS`
- Cookie chứa datr: `c_user=...; xs=...; datr=OHILajGNGFUlFciYOta5HkHS; ...` (regex extract `datr=([A-Za-z0-9_-]+)`)

Code seed cả 2 file từ embedded data lần đầu chạy (xem `embedded/cookie_initial.txt`, `embedded/datr_pool.txt`).

**Sử dụng datr theo platform** (luồng cụ thể):

| Platform / Step | Datr lấy từ đâu | Inject thế nào | Note |
|---|---|---|---|
| **Reg Android** (`s23`, `s5xx`, `s273` reg) | Random pop từ `cookie_initial.txt` + `datr_pool.txt` qua `androidreg.SharedPool` | Đặt vào `machine_id` field POST `/app/users` (Bloks body) hoặc REST `/auth/login` body | Mỗi datr dùng N lần (config `LimitCookieInitial` + `LimitCookieInitialCount`). Hết slot → unregister. |
| **Reg iOS** | Tương tự Android — pool chung | `machine_id` trong iOS body | Pool dùng chung với Android (chia limit slot). |
| **Reg Web / WebAndroid** | Auto-generate fresh (datr_gen) khi không có | Set cookie `datr=` trước GET `m.facebook.com/` | Sau reg success → ghi datr mới vào `datr_pool.txt` để tái sử dụng. |
| **Verify Android-family** (`s23`, `s5xx`, `s273`, `s399`) | `session.Datr` (từ reg) hoặc `ExtractDatrFromCookieStr(session.Cookie)` | Đặt `machine_id` trong session_ctx (Bloks payload) hoặc header | Nếu thiếu → `uuid.New().String()` fallback. |
| **Verify WebAndroid** | `session.Datr` hoặc cookie | Inject vào cookie jar trước POST | Match cookie session FB lưu. |
| **Verify Web MFB** | `session.Datr` hoặc cookie | Set cookie trong session login | Lấy datr sau login GET `m.facebook.com/login`. |
| **Verify s273 Step 3.5** (post-confirm) | KHÔNG cần datr | KHÔNG inject cookie | Pure token API (`/me?fields=email&access_token=`). |

**Save new datr** (option `SaveNewDatr` trong [app.go:4465](../../app.go) — config user):
- Bật: sau verify Live, dùng cookie + token + UA để gọi GraphQL lấy datr mới → `androidreg.SharedPool.AddDatrRawNoPersist` + ghi vào file → vòng đời pool tự lớn theo số reg Live thành công ("learning loop").

**Datr age limit** (`LimitDatrAgeMinutes`): datr quá cũ trong pool tự bị purge — tránh dùng đi dùng lại datr stale.

**Health-check pool**: nếu `androidreg.SharedPool.IsCompletelyEmpty()` → scheduler báo skip slot (không có datr → không reg được). Fallback 3 tầng:
1. Reload file `cookie_initial.txt` (user có thể đã paste thêm).
2. Đọc 100 dòng cuối `SuccessVerify_No2FA.txt` của run hiện tại (datr fresh từ acc vừa reg+verify Live).
3. Tạo batch datr mới qua `datr_gen` (Method=`new`).

**Cảnh báo khi thêm verify version mới**:
- Nếu platform mới cần datr (vd. Bloks-based) → trong helper builder phải gọi `ExtractDatrFromCookieStr(session.Cookie)` hoặc `session.Datr` và pass vào `machine_id`. KHÔNG hardcode UUID random — sẽ bị FB flag.
- Nếu platform mới chỉ dùng token (như s273 Step 3.5) → không inject cookie/datr, dùng `access_token` query param.

### 13.6 iOS Native (FBIOS) verify (`ios562`, `ios563`) — BẮT BUỘC token `EAAAAAY`

**Verifier hiện có**: `ios562` (đăng ký `RegisterPlatformVerifier(PlatformIOS562)` — [ios562/verify.go](../../internal/facebook/verify/ios562/verify.go)). Dropdown UI còn có `ios563` (cùng họ FBIOS).

**Auth**: user token iOS prefix **`EAAAAAY`** — KHÁC token Android `EAAAAU`. Truyền qua header `Authorization: OAuth <EAAAAAY>` ([ios562/steps.go](../../internal/facebook/verify/ios562/steps.go)). Nếu account có `srnonce` + `sessionlessCryptedUID` (từ reg iOS) → sessionless flow dùng app token; ngoài ra bắt buộc user token iOS.

**Endpoint**: Bloks CAA iOS (`graph.facebook.com/graphql`, doc_id iOS riêng). Thin wrapper `ios562.Verifier` → `verifybase.RunVerify` ([ios562/verify.go](../../internal/facebook/verify/ios562/verify.go)).

**UA**: FBIOS (`FBAN/FBIOS`). `FixUA` tự regen UA iOS nếu account đang giữ UA Android.

#### ⚠️ RULE — reg Android nhưng verify iOS → BẮT BUỘC login iOS lấy `EAAAAAY`

Endpoint Bloks CAA iOS **chỉ chấp nhận user token iOS `EAAAAAY`**. Token Android `EAAAAU` (lấy từ REST `/auth/login`) **KHÔNG verify được** bằng iOS.

| Luồng | Token sẵn có | Cần làm gì |
|---|---|---|
| Reg iOS (`ios562`/`ios563`) → ver iOS | reg trả `EAAAAAY` (hoặc `srnonce` + `sessionlessCryptedUID`) | Chạy được ngay |
| **Reg Android → ver iOS** | chỉ `EAAAAU` / UID+pass | **PHẢI login iOS** đổi lấy `EAAAAAY` trước khi verify. Không có `EAAAAAY` → iOS verify KHÔNG chạy |

#### Luồng iOS login lấy `EAAAAAY` (CAA `send_login_request`)

Tham chiếu capture thật: `D:\Git2026\facebook_repo\IOS\[4624] request/response_graph.facebook.com_message.txt`.

- **Endpoint**: `POST graph.facebook.com/graphql`, `fb_api_req_friendly_name = FBBloksActionRootQuery-com.bloks.www.bloks.caa.login.async.send_login_request` (CAA login Bloks).
- **Authorization**: **app token** `OAuth 6628568379|c1e620fa708a1d5696fb991c1bde5662` (= `iosAppToken` trong [ios562/steps.go](../../internal/facebook/verify/ios562/steps.go)). Login dùng **app token** để LẤY user token (không phải user token sẵn có).
- **Body** (`params.server_params` + `client_input_params`): `device_id`, `family_device_id`, `machine_id` (= **datr**, trùng header `X-FB-Integrity-Machine-Id`), `cloud_trust_token`, `password` (mã hoá `#PWD_*`), `contact_point` (uid/sđt), `login_credential_type`, `waterfall_id`, ...
- **Response** (login success) trả về:
  - `access_token`: **`EAAAAAY...`** ← user token iOS, gán vào `session.Token` rồi mới verify.
  - `session_cookies`: `c_user`, `xs`, `fr`, `datr`.

> **Reg Android → ver iOS**: account chỉ có UID+password (hoặc `EAAAAU`) → PHẢI chạy luồng CAA login iOS này (KHÔNG dùng `/auth/login` Android vốn trả `EAAAAU`) để lấy `EAAAAAY`. Các thông số device (`device_id`/`family_device_id`/`machine_id`=datr/`cloud_trust_token`/SIM/UA) phải **dùng chung** giữa login và verify để fingerprint nhất quán.

#### ⚠️ BẮT BUỘC — iOS chạy mạng di động (cellular), KHÔNG wifi → header `x-fb-sim-hni`

iOS reg/verify/login **phải set `X-FB-Connection-Type` = cellular** (`mobile.CTRadioAccessTechnology*`) để header **`x-fb-sim-hni` = `p.Sim.HNI`** (HNI = MCC+MNC, vd `45204`) luôn được gửi. Trên wifi FB iOS KHÔNG gửi sim-hni → fingerprint yếu.

- Verify đã set `{"x-fb-sim-hni", sc.Sim.HNI}` + `{"x-fb-connection-type", connType}` ([ios562/steps.go:177,180](../../internal/facebook/verify/ios562/steps.go#L177)); `sc.Sim` populate qua `fakeinfo.RandomSimProfile(CountryFromPhone(phone))` ([run.go:364](../../internal/facebook/verify/verifybase/run.go#L364)).
- Reg cũng có ([ios562/http.go:175](../../internal/facebook/register/ios562/http.go#L175), [pwdkey.go:95](../../internal/facebook/register/ios562/pwdkey.go#L95)); ConnType từ `fakeinfo.RandomIOSConnType()` ([ios562/profile.go:77,99](../../internal/facebook/register/ios562/profile.go#L77)).
- **✅ FIX 2026-05-31**: đã bỏ `"wifi"` khỏi `fakeinfo.iosConnTypes` ([simnetwork.go](../../internal/facebook/fakeinfo/simnetwork.go)) → `RandomIOSConnType()` luôn trả cellular → iOS luôn gửi `x-fb-sim-hni` nhất quán.

#### Header iOS chuẩn (theo capture `send_login_request` / AddMail)

| Header | Giá trị | Note |
|---|---|---|
| `User-Agent` | `...[FBAN/FBIOS;FBAV/563...;FBDV/iPhone9,1;...;FBLC/vi_VN;FBOP/5;...]` | FBIOS |
| `Authorization` | `OAuth <EAAAAAY>` (verify) / `OAuth <iosAppToken>` (login) | đúng loại |
| `X-FB-Connection-Type` | `mobile.CTRadioAccessTechnologyLTE` (cellular) | **KHÔNG wifi** |
| `x-fb-sim-hni` | `<HNI>` (vd `45204`) | chỉ có khi cellular |
| `X-FB-Integrity-Machine-Id` | `<datr>` | = `machine_id` trong body |
| `X-Cloud-Trust-Token` | 2×UUID upper nối liền (72 ký tự) | |
| `X-FB-Device-ID` / `X-FB-Family-Device-Id` | UUID upper | dùng lại từ reg |
| `X-FB-Friendly-Name` | `FBBloksActionRootQuery-com.bloks.www.bloks.caa.login.async.send_login_request` | login |

#### ✅ ĐÃ IMPLEMENT (2026-05-31)

- **iOS chỉ nhận `EAAAAAY`**: spec iOS set `ValidateToken = HasPrefix("EAAAAAY")` ([ios562/steps.go](../../internal/facebook/verify/ios562/steps.go)). `verifybase.RunVerify` dùng `spec.ValidateToken` (nil → default `isValidUserToken` cho Android, giữ nguyên hành vi cũ) ([run.go](../../internal/facebook/verify/verifybase/run.go)). Token Android `EAAAAU` đưa vào iOS verify → coi như chưa hợp lệ → buộc login iOS.
- **Login iOS lấy `EAAAAAY`**: `ios562reg.FetchIOSToken` ([register/ios562/login.go](../../internal/facebook/register/ios562/login.go)) chạy CAA `send_login_request` (app token → `EAAAAAY` + cookies; tái dùng `docIDAction`/`bloksVersioningID`/`encryptPasswordForReg`/`parseCreateAccountResponse`). Spec iOS set `FetchToken` gọi hàm này khi token chưa đúng loại — KHÔNG dùng `/auth/login` Android cho iOS nữa.
- **`ios563` verifier**: đã `RegisterPlatformVerifier(PlatformIOS563)` (dùng chung verifier ios562) ([verify.go](../../internal/facebook/verify/ios562/verify.go)) + thêm `PlatformIOS563` vào scheduler iOS case ([scheduler.go](../../internal/runner/scheduler.go)).
- **Áp dụng cả split lẫn non-split**: mọi đường verify (non-split inline `RunOneAccountAt`, verify-only `RunVerify`, split `RunOneAccountAt`) đều hội tụ về `runOneAccount → ver.Verify → verifybase.RunVerify → FetchToken` ([scheduler.go:753](../../internal/runner/scheduler.go#L753)) → login iOS chạy ở cả 2 chế độ.
- **Header/body login đối chiếu reg + ver**: dùng chung device fingerprint (`BuildProfile`) + `x-fb-sim-hni` cellular như reg/ver; thêm `x-fb-rmd: state=URL_ELIGIBLE` (có trong capture login + verify, reg không có). **Retry login tối đa 3 lần** (backoff 2s/4s) cho lỗi mạng/transient; checkpoint / sai mật khẩu → dừng ngay không retry ([login.go](../../internal/facebook/register/ios562/login.go)).
- **Lưu ý**: vì `EAAAAAY` là bắt buộc, luồng sessionless cũ (srnonce + app token, không user token) KHÔNG còn áp dụng cho iOS verify — account thiếu `EAAAAAY` sẽ login iOS (cần UID+password). `FetchIOSToken` build theo capture nhưng **chưa test với FB thật** (cần account + proxy live để chốt body login khớp 100%).

### 13.7 Login-at-verify + Token/Cookie hiển thị realtime (cập nhật 2026-05-31)

Luật xử lý token/cookie giữa REG và VERIFY (áp dụng CẢ split lẫn non-split). Bổ sung cho §13.6.

#### A. MỌI login/token-fetch nằm ở VERIFY, KHÔNG ở REG
- `web.SkipAuthLoginAtReg = interactionCfg.VerifyEnabled || verifyIsIOS(ApiVerifyPlatform)` ([app.go](../../app.go)). Khi sẽ chạy verify → reg cookie-only (WebAndroid/Web) KHÔNG gọi `/auth/login` lấy token; verify tự lấy.
- Block `[AutoVerify]` pre-fetch token lúc reg cũng bị gate `&& !SkipAuthLoginAtReg`.
- Reg Android FB4A (s5xx) / iOS native trả token NGAY trong response reg — đó KHÔNG phải login riêng, không gate. s399 `/auth/login` là step trong protocol reg của nó — KHÔNG đụng.

#### B. Token đúng loại theo platform verify (ĐỐI XỨNG)
| Verify platform | Token bắt buộc | Thiếu/sai loại → lấy bằng |
|---|---|---|
| iOS (`ios562`/`ios563`) | `EAAAAAY` | CAA `send_login_request` — `ios562reg.FetchIOSToken` |
| Android-family | `EAAAAU` | REST `/auth/login` — `FetchAndroidTokenLegacy` |
- iOS verify gặp `EAAAAU` → coi như chưa hợp lệ → login iOS lấy `EAAAAAY`.
- Android verify gặp `EAAAAAY` (reg iOS) → **bỏ token đó** (`if HasPrefix(session.Token,"EAAAAAY"){session.Token=""}` trước `needFetch`) → `/auth/login` lấy `EAAAAU` ([scheduler.go](../../internal/runner/scheduler.go)).
- `preferUserAccessToken` (Go [app.go](../../app.go) + frontend [AccountsPage.vue](../../frontend/src/pages/AccountsPage.vue)) ưu tiên token MỚI hợp lệ **`EAAAAU` HOẶC `EAAAAAY`** — KHÔNG chỉ `EAAAAU` (nếu không token iOS bị loại khi row đã có `EAAAAU` cũ).

#### C. iOS chạy cellular (x-fb-sim-hni)
- `fakeinfo.iosConnTypes` ([simnetwork.go](../../internal/facebook/fakeinfo/simnetwork.go)) bỏ `"wifi"` → `RandomIOSConnType()` luôn cellular → header `x-fb-sim-hni = p.Sim.HNI` luôn được gửi. iOS reg/verify/login đều cellular (wifi → FB không gửi sim-hni, fingerprint yếu).

#### D. Token/Cookie hiển thị realtime lên cột TOKEN/COOKIE
Đường đi: verify lấy token → `session.Token/Cookie` → back-fill `result.Token/Cookie` → `OnAccountDone` emit → frontend `verify:account-done` → `acc.token/acc.cookie`.
- `facebook.VerifyResult` KHÔNG có Cookie; `runner.AccountResult` đã thêm field `Cookie` ([scheduler.go](../../internal/runner/scheduler.go)).
- Sau `ver.Verify`: back-fill `result.Token = session.Token` + `result.Cookie = session.Cookie` (iOS: FetchToken set `session.Cookie` từ login; Android: set trong switch trước ver.Verify).
- Emit dùng **PARAM `token`/`cookie`** (= `result.*`), KHÔNG dùng `doneAcc.*` (doneAcc có thể rỗng nếu row không nằm trong `a.accounts`, vd split slot).
- **KHÔNG clear `a.accounts[i].Token`/`.Cookie`** sau `OnAccountDone` — chúng là cột hiển thị. Clear → `ListAccounts`/`fetchAccounts` (kích bởi `verify:complete` / `verify:accounts-updated`) refetch ra rỗng → mất token/cookie trên UI (status thì backend giữ nên status vẫn còn). Chỉ clear `FullData`/`SourceCode`/`NoteRun` (nặng, không hiển thị).
- Frontend handler `verify:account-done` ([AccountsPage.vue](../../frontend/src/pages/AccountsPage.vue)) đọc `data.token`/`data.cookie` → set `acc.token`/`acc.cookie`.
- ⚠️ **KNOWN / TODO**: token mới CHỈ emit ở `OnAccountDone` (lúc verify XONG), CHƯA emit ngay lúc login xong (giữa verify). Trong lúc verify dở, cột token chưa lên (non-split) hoặc hiện token reg cũ (split). Hướng fix: thêm callback realtime `OnTokenFetched` emit token+cookie NGAY sau login (giữa verify) — đường `VerifyConfig.OnTokenFetched` → `RunConfig` → emit `verify:token`.

#### E. Message login đồng nhất
- iOS: `[Login][iOS]`; Android: `[Login][Android]` (trước: `[iOS Login]` / `[Legacy Login]`).

## 14. Bảng cheat — account import cần field gì

Gom theo 3 NHÓM auth (không liệt platform riêng lẻ vì cùng nhóm = cùng requirement):

| Nhóm Verify | Platforms | UID | Cookie | Token EAA | Password | Datr | Note |
|---|---|:---:|:---:|:---:|:---:|:---:|---|
| **Android-family** (Bloks + Legacy REST) | `s23`, `s273`, `s399`, `s415`–`s563*`, `android` (api token) | ✓ | optional (chỉ extract locale; không inject) | ✓ **bắt buộc** | optional (fallback fetch token) | optional (machine_id) | Token thiếu + có UID+Password → scheduler fetch EAA qua REST `/auth/login` |
| **WebAndroid** (Chrome Mobile cookie) | `webandroid` (api web andr) | ✓ | ✓ **bắt buộc** (`c_user; xs; datr; fr`) | KHÔNG cần | KHÔNG cần | đã có trong cookie | Skip login, dùng cookie trực tiếp |
| **Web MFB** (cookie + fb_dtsg) | `web` (api mfb) | ✓ | ✓ **bắt buộc** | KHÔNG cần | KHÔNG cần | đã có trong cookie | Login `m.facebook.com/login` trước để parse `fb_dtsg`/`jazoest` |
| **iOS Native (FBIOS)** | `ios562`, `ios563` | ✓ | optional | ✓ **bắt buộc `EAAAAAY`** (token iOS) | optional | optional (machine_id) | ⚠️ Token Android `EAAAAU` KHÔNG đủ. **Reg Android → ver iOS PHẢI login iOS lấy `EAAAAAY`** mới chạy được. Reg iOS đã có sẵn `EAAAAAY`/srnonce. |

**Quy tắc nhận diện platform → nhóm**: xem `case` trong [scheduler.go:524-650](../../internal/runner/scheduler.go). Platform nào không có trong 1 trong 3 nhóm trên = scheduler báo FATAL error (safety net từ 2026-05-26).

## 15. Lớp chống false-positive (post-confirm verify)

**False-positive** = ghi Live vào `SuccessVerify_No2FA.txt` khi email không thực sự attached.

| Platform | Có post-confirm verify? | Chạy luôn? | Phương pháp |
|---|:---:|:---:|---|
| `s273` (sau fix 2026-05-26) | ✓ | ✓ **luôn** | GET `/me?fields=email` với token |
| `s399`, `s5xx` | qua `CheckLiveDieCombined` | chỉ khi `CheckLiveDie = true` | Picture endpoint (302 redirect) + token check |
| `webandroid` | `detectPendingOrCheckpoint` | chỉ khi `CheckLiveDie = true` | GET `m.facebook.com/` cookie → redirect check |
| `web` (mfb) | qua `CheckLiveDieCombined` | chỉ khi `CheckLiveDie = true` | Picture endpoint |

**Default: ON** (từ 2026-05-26) — [defaults.go:63](../../internal/settings/model/defaults.go#L63) đã đổi `CheckLiveDie: true` để mọi platform luôn có lớp post-confirm bảo vệ trừ khi user tắt thủ công.

**Tên hiển thị** vs **field nội bộ**:
- Field Go: `CheckLiveDie` / Field TS: `checkLiveDieEnabled` (giữ nguyên — không rename để không phá config user đã lưu).
- Label UI: **"Kiểm tra sau reg"** (đã đổi 2026-05-26 vì checkbox kiểm tra cả verify state, không chỉ Live/Die).
- Tooltip: "Bật lớp xác minh sau khi confirm OTP (post-verify): check pending/checkpoint để chống ghi nhầm acc chết..."

**Khuyến nghị**: giữ default ON cho mọi flow. Riêng `s273` có Step 3.5 (GET `/me?fields=email`) luôn chạy không phụ thuộc checkbox.

**Logic chờ delay**: dù `Kiểm tra sau reg` bật hay tắt, code vẫn **chờ đủ `Trước check live (s)` giây** (`cfg.TimeDelayCheck`) trước khi trả Live cuối cùng:
- Bật: vào loop check Live/Die mỗi 5s trong tổng `TimeDelayCheck` giây.
- Tắt: chỉ `time.Sleep(TimeDelayCheck)` rồi trả Live.

**Vị trí checkbox**: panel **Verify / Xác thực** trên `InteractionSetupPage`, đặt cạnh `Tự gửi lại OTP khi timeout` (xem [InteractionSetupPage.vue](../../frontend/src/pages/InteractionSetupPage.vue)).

### 15.1 Retry Unknown ngay (Pass 2 với proxy mới)

Acc kết quả `unknown` / `error` thường do lỗi network/proxy tạm thời — pool 100-500 acc chạy 1 lần dễ có vài chục acc unknown. Tính năng **"Verify lại Unknown ngay"** chạy pass 2 ngay sau pass 1 để recover phần này.

**Field**:
- Go: `RunConfig.RetryUnknownNow bool` ([scheduler.go RunConfig](../../internal/runner/scheduler.go))
- Go: `InteractionConfig.RetryUnknownNow bool` ([app.go](../../app.go))
- TS: `VerifyConfig.retryUnknownNow: boolean` ([interaction.types.ts](../../frontend/src/types/interaction.types.ts))
- UI: checkbox **"Verify lại Unknown ngay"** trong panel Verify, cạnh `Kiểm tra sau reg`.

**Logic** (sau `wg.Wait()` của pass 1 trong `RunVerify`):
1. Collect acc có `status == "" / "unknown" / "error"` → `retryItems`.
2. Tạo worker pool MỚI với `workerID + maxThreads` (offset slot) — sticky-proxy manager nhìn slot mới → buộc acquire proxy FRESH từ pool, KHÔNG reuse proxy đã fail pass 1.
3. Mỗi worker chạy `runOneAccount` với `IsRetry=true` (tránh runOneAccount tự retry nội bộ) + `RetryUnknownNow=false` (chống recursion pass 3).
4. Acc Live/Die → ghi đè kết quả pass 1. Acc vẫn unknown → giữ unknown.

**Tại sao offset workerID**:
- Sticky proxy: `Acquire(ctx, workerID)` → cache theo slot.
- Pass 1: slot 0..99 đã sticky → release(false) cho non-Live → proxy về pool.
- Pass 2: nếu cùng slot 0..99 → sticky cache có thể return proxy cũ.
- Pass 2 dùng slot 100..199 (chưa từng acquire) → forced fresh.

**Double insurance**: (a) Pass 1 release(false) đã trả proxy về pool + (b) Pass 2 offset slot tránh sticky cache.

**Default**: OFF (user bật manual qua checkbox).

## 16. Thêm Verify Version mới

### 16.1 Backend

1. **Tạo package** `internal/facebook/verify/sXXX/`:
   - `verify.go` — Verifier struct, `init()` register `RegisterPlatformVerifier` + `RegisterPlatformVerifyUA`.
   - `helpers.go` — UA pool/builder, body builders, header builders, post-confirm check helper.
2. **Khai báo constant** trong [internal/facebook/factory.go](../../internal/facebook/factory.go):
   ```go
   PlatformSXXX = "sXXX"
   ```
3. **Thêm vào Android-family case** trong [scheduler.go:524-565](../../internal/runner/scheduler.go) — **BẮT BUỘC**, quên = scheduler báo FATAL error.
4. **Thêm vào** [`platformNeedsAndroidLoginToken`](../../app.go) (~line 435-448).
5. **Thêm case** vào [`pickUAForVerifyPlatform`](../../app.go) (~line 10773-10880) với `expectedFBAVPrefix = "FBAV/XXX."`. Nếu pool nhiều FBAV (như s273) thì thêm custom validator sau switch.
6. **Blank import** `_ "HVR/internal/facebook/verify/sXXX"` trong [app.go imports](../../app.go) (~line 50-60).

### 16.2 Frontend

7. **Cập nhật** [InteractionSetupPage.vue](../../frontend/src/pages/InteractionSetupPage.vue):
   - `ORIGINAL_UA_STRINGS`
   - `VER_PLATFORM_LABELS`, `VER_PLATFORMS_VER`
   - `REG_PLATFORM_LABELS`, `REG_PLATFORMS_VER` (nếu version cũng dùng cho reg).

### 16.3 Hợp đồng verifier interface

Verifier phải implement:
```go
Verify(ctx context.Context, session *facebook.Session, cfg *facebook.VerifyConfig, outputPath string, onStatus func(uid, msg string)) *facebook.VerifyResult
```

`VerifyResult.Status` chỉ được là 1 trong:
- `"Live"` → ghi `SuccessVerify*.txt`
- `"Die"` → ghi `DieAfterVerify.txt`
- `"error"` / `"unknown"` → ghi `UnknownErrorCheckLiveDieApi.txt`

**KHÔNG được** trả `Status: "Live"` mà chưa thực sự xác minh. **Pattern bắt buộc**:

```go
// SAI — negative check, false positive rất cao:
if !isError(resp) {
    return &VerifyResult{Status: "Live"}
}

// ĐÚNG — positive check:
respTrim := strings.TrimSpace(resp)
isSuccess := respTrim == "true" ||
    strings.Contains(resp, `"result":true`)
if !isSuccess {
    return &VerifyResult{Status: "error", Message: "Confirm không có positive marker: " + resp}
}

// + Post-confirm verify (LUÔN chạy):
status, detail := postConfirmCheckEmail(ctx, client, session.Token)
if status == "DIE" {
    return &VerifyResult{Status: "Die", Message: "Token checkpoint after confirm: " + detail}
}
if status == "NO_EMAIL" {
    return &VerifyResult{Status: "error", Message: "Email không attached after confirm: " + detail}
}
```

### 16.4 Helper bắt buộc cho log error

Mọi verify package mới PHẢI dùng 2 helper sau khi log raw FB response (`error_msg` thường chứa Thai/Arabic/CJK escape thành `\uXXXX` → log không đọc được):

**`verifybase.DecodeUnicodeEscapes(s) string`** ([verifybase/helpers.go](../../internal/facebook/verify/verifybase/helpers.go)):
- Decode literal `\uXXXX` (6 chars) → ký tự thật.
- BMP only (skip surrogate pair — đủ cho FB error_msg).
- Tự skip nếu chuỗi không chứa `\u` (no-op nhanh).

**`verifybase.SummarizeFBError(resp) string`** đã tự decode + truncate theo **rune** (không phải byte) — dùng được ngay không cần wrap.

**Truncate riêng cho log raw body** (nếu cần): tạo `truncateForLog(s, n)` trong package mình giống [s273/helpers.go truncateForLog](../../internal/facebook/verify/s273/helpers.go) — pattern:

```go
func truncateForLog(s string, n int) string {
    decoded := verifybase.DecodeUnicodeEscapes(s)
    runes := []rune(decoded)
    if len(runes) <= n {
        return decoded
    }
    return string(runes[:n]) + "..."
}
```

**Quan trọng — truncate theo RUNE** không phải byte: Thai/CJK = 3 bytes UTF-8 mỗi ký tự, cắt theo byte sẽ phá string giữa multi-byte → log hỏng.

**Trước khi fix** (2026-05-26):
```
[S273 Verify] Confirm KHÔNG xác nhận được: {"error_code":3301,"error_msg":"รหัสยืน..."
```
**Sau khi fix**:
```
[S273 Verify] Confirm KHÔNG xác nhận được: {"error_code":3301,"error_msg":"รหัสยืนยันไม่ถูกต้อง"}
```

### 16.5 Checklist Verify

- [ ] Package verify tồn tại với `verify.go` + `helpers.go`.
- [ ] Constant platform khai báo trong `factory.go`.
- [ ] Blank import trong `app.go`.
- [ ] **Scheduler case Android-family có platform mới** (nếu là Android-family — sẽ FATAL nếu quên).
- [ ] `platformNeedsAndroidLoginToken` có platform mới.
- [ ] `pickUAForVerifyPlatform` có case `expectedFBAVPrefix`.
- [ ] UI buttons + labels + UA preview.
- [ ] **Confirm response check POSITIVE** (`"true"` / `"result":true`), KHÔNG dùng negative check.
- [ ] **Post-confirm GET /me?fields=email** chạy LUÔN (không phụ thuộc CheckLiveDie).
- [ ] **Log error có decode `\uXXXX` + truncate theo RUNE** (dùng `verifybase.DecodeUnicodeEscapes` + `truncateForLog` pattern §16.4).
- [ ] **Build UA đọc từ `Config/DeviceInfo/*`** nếu là pool approach (như s273) — KHÔNG hardcode device pool.
- [ ] **Test với country pool**: `RandomUA("VN")`, `RandomUA("US")`, `RandomUA("")` → locale + carrier khác nhau.
- [ ] Doc platform mới vào bảng §11-15 file này.
- [ ] `go build ./...` pass.
- [ ] `npm run build` pass.

## 17. Công cụ kiểm tra acc đã ghi (debug tool)

`cmd/check_verified_email/main.go` — kiểm tra acc trong `SuccessVerify_No2FA.txt` có email thực sự attached không (token-based, KHÔNG dùng cookie).

```powershell
go run ./cmd/check_verified_email "build/bin/result/result_<run>/SuccessVerify_No2FA.txt" -c=10
```

**Output**:
- In từng acc với status: `VERIFIED` / `NOT_VERIFIED` / `TOKEN_DEAD` / `ERROR`.
- Ghi 2 file cạnh input:
  - `<input>.verified.txt` — acc tin cậy có email thực tế.
  - `<input>.suspicious.txt` — acc nghi ngờ / token chết / lỗi → cần review.

**Chỉ dùng cho file có token EAA** (`s273`, `s399`, `s5xx`, `s23`, `android`). KHÔNG dùng được cho WebAndroid / Web result (không lưu token EAA).

## 18. Known issues (chưa fix)

_Hiện chưa có open issue. Khi phát hiện bug verify mới, thêm vào đây với format:_

```
### Bug N — <tóm tắt 1 dòng>
**Triệu chứng**: <user-visible behaviour>
**Nguyên nhân**: <root cause + file:line>
**Status**: open / investigating / waiting-on-user
```

Fix xong thì xoá khỏi mục này.
