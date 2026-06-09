# Luồng REGISTER end-to-end (mọi platform)

> Tài liệu này mô tả **chính xác luồng chạy thực tế** của chức năng REGISTER tài khoản Facebook trong app HVR — từ lúc frontend gọi `RunRegister` đến lúc account được ghi file + đẩy sang verify. Đọc kèm:
> - [add-facebook-reg-version.md](add-facebook-reg-version.md) — runbook **thêm version mới** (§1–§10). Tài liệu này KHÔNG lặp lại runbook; chỉ link tới khi liên quan.
> - [02-luong-verify.md](02-luong-verify.md) (nếu có) — luồng verify nối tiếp sau reg.
>
> Đường dẫn code dùng link tương đối từ `docs/facebook/` → lùi 2 cấp (`../../`).
>
> Mọi `file:line` trong tài liệu là vị trí **tại thời điểm viết** (2026-05-31). Code rất "nóng" (thay đổi liên tục) — nếu lệch vài dòng, tìm theo tên hàm.

---

## 0. Sơ đồ tổng quan

```
Frontend (AccountsPage.vue)
   │  handleRunRegister()  → bridge.runRegister(maxThreads)
   ▼
verify-runner.wails.ts  → RunRegister(maxThreads)   [Wails-bound]
   ▼
app.go  RunRegister(maxThreads)                      [app.go:7364]
   │
   ├─ 1. GATING        verify đang chạy? state machine idle? (registerMu)
   ├─ 2. CONFIG        LoadInteractionConfig → regPlatforms (multi-version) → maxThreads
   ├─ 3. PROXY warm    getRegProxyManager + pre-check format (async health)
   ├─ 4. OUTPUT        result_<tag>_<datetime>/  + resultpkg.Writer + CounterSet
   ├─ 5. DATR POOL     sharedCookiePool (PartitionedDatrPool) ← cookie_initial.txt / datr_pool / generate
   │                   gán cho TẤT CẢ platform pool (Android/S23/Sxxx/WebAndroid/iOS/...)
   ├─ 6. SESSION POOL  newRunResources (iOS HTTP / WebAndroid keep-session) + ios562 DevicePool
   ├─ 7. EMAIL POOL    nếu VerifyEnabled → emailrent.NewXxxPool (batch mua mail)
   ├─ 8. STATE→running ctx/cancel + regWorkerCtx (drain-safe) + auto-restart ticker
   ├─ 9. SLOTS         freeSlotsReg chan[1..maxThreads]  + (split) freeSlotsVer + VER pool
   │
   └─ SPAWNER goroutine (loop ∞ đến ctx.Done)
         │  nhận slotIdx ← freeSlotsReg
         │  chọn version slot = regPlatforms[(slotIdx-1)%N]   (round-robin cố định/slot)
         │  build profile + UA (3 mode) + phone/email theo RegMode
         ▼
       WORKER goroutine (1 account)
         ├─ acquire proxy (sticky) + RenderSession + CheckIP → countryCode
         ├─ gen phone theo country / build UA theo IP
         ├─ NewWorkerContext(platform) (keep-session)
         ├─ LOOP keep-session (maxKeepSessionRegs)
         │     ├─ datr ← SharedPool.GetNext(slotIdx)
         │     ├─ threadReg.Register(...) / WorkerContext.Register(...)
         │     │     ▼  ═══ PER-PLATFORM REG ═══
         │     │     • android (FB4A Bloks)  → POST b-graph/graphql, machine_id=datr
         │     │     • ios562/ios563 (FBIOS)  → POST graph/graphql, round-2 nosess
         │     │     • webandroid (Chrome)    → GET m.fb + POST wbloks + /auth/login
         │     │     • s399 (FB4A REST cũ)    → /app/users + /auth/login
         │     │     • s23 / sXXX (Bloks)     → POST b-graph/graphql
         │     ├─ EAA pre-fetch (Android-family verify) → FetchAndroidTokenLegacy
         │     ├─ emit register:token + saveRegOutcome (SuccessReg/NVR)
         │     ├─ persistNewDatr(datr mới) → Pool file + tất cả pool RAM
         │     └─ VERIFY?  inline (RunOneAccountAt)  HOẶC  split (→ splitVerifyCh → VER pool)
         └─ trả slotIdx về freeSlotsReg
```

---

## 1. Điểm vào — từ frontend tới `RunRegister`

**Mục tiêu**: hiểu lệnh Start Reg đi từ UI xuống Go ra sao.

1. **Frontend**: nút Start gọi `handleRunRegister()` ở [AccountsPage.vue:1861](../../frontend/src/pages/AccountsPage.vue). Hàm này resolve `maxThreads` rồi gọi bridge.
2. **Bridge layer**: `verifyRunnerWails.runRegister(maxThreads)` ở [verify-runner.wails.ts:35](../../frontend/src/bridge/wails/verify-runner.wails.ts) gọi thẳng generated binding `RunRegister(maxThreads)`.
3. **Wails-bound method**: `App.RunRegister(maxThreads int) string` ở [app.go:7364](../../app.go). Trả về string là message (rỗng/text) — frontend chỉ dùng để toast lỗi; mọi tiến triển thật được **đẩy qua event** (`runtime.EventsEmit`).

> `maxThreads` từ frontend **chỉ là fallback**. Nguồn truth thật sự là `interactionCfg.RegThreads` (xem [app.go:7404](../../app.go)) — reg/verify có thread count riêng. Tham số `maxThreads` chỉ dùng khi `RegThreads <= 0`.

**Sự kiện frontend lắng nghe** (mọi cập nhật reg đi qua đây):
| Event | Ý nghĩa | Emit tại |
|---|---|---|
| `register:output-path` | thư mục kết quả phiên này | [app.go:7640](../../app.go) |
| `register:status` | 1 dòng status/slot (path chậm, system msg) | rải rác |
| `register:batch-status` | gom nhiều slot, 1 event / 200ms | [app.go:8578](../../app.go) |
| `register:token` | UID + token + cookie ngay sau reg | [app.go:9553](../../app.go) |
| `register:account-done` | account hoàn tất (sau verify) | (cuối worker) |
| `register:complete` | spawner kết thúc (Stop/lỗi) | [app.go:8688](../../app.go) |
| `register:auto-restart-trigger` | tới giờ auto-restart | [app.go:8072](../../app.go) |

---

## 2. Gating — chống chạy chồng

**Input**: gọi `RunRegister`. **Output**: hoặc reject (string lý do), hoặc transition `idle → running`.

1. **Chặn verify chạy song song** ([app.go:7366](../../app.go)): verify + register dùng chung `emailPool` → nếu `a.isRunning` (verify) true → trả `"Verify đang chạy..."`.
2. **State machine** ([app.go:7373](../../app.go)): đọc `runState(a.registerState.Load())`:
   - `runStateRunning` → `"Đang chạy đăng ký..."`.
   - `runStateStopping` → `"Đang dừng run cũ — chờ hoàn tất rồi start lại"` (spawner cũ chưa `wg.Wait()` + cleanup xong).
   - chỉ `runStateIdle` mới được tiếp.
3. **Generation counter** ([app.go:7387](../../app.go)): `a.registerGen++` → `myGen`. Chỉ dùng cho **identity check trong defer cleanup** (chống event/cleanup stale), KHÔNG dùng để gating.
4. **Transition idle → running** ([app.go:8028](../../app.go)): `a.registerState.Store(int32(runStateRunning))` được thực hiện **sau khi toàn bộ validation pass**, ngay trước khi spawn spawner. Validation early-return chỉ `Unlock + return` (state vẫn idle).

State machine định nghĩa tại [app.go:199–216](../../app.go). Spawner defer chịu trách nhiệm `stopping → idle` SAU `wg.Wait()` + cleanup native (xem §13).

---

## 3. Chọn platform reg + multi-version round-robin

**Mục tiêu**: 1 batch có thể chạy **nhiều version cùng lúc**, mỗi luồng (slot) cố định 1 version.

1. `interactionCfg := a.LoadInteractionConfig()` ([app.go:7398](../../app.go)).
2. `regPlatforms := regPlatformList(interactionCfg)` ([app.go:7411](../../app.go)) → list version user chọn.
   - Logic tại [app_reg_sxxx.go:201](../../app_reg_sxxx.go): ưu tiên `ApiRegPlatforms` (trim + dedup, giữ thứ tự); rỗng → `[ApiRegPlatform]`; vẫn rỗng → `[PlatformWeb]`.
3. `regPlatform = regPlatforms[0]` = **primary** — dùng cho tên folder, session pool, validation.
4. Round-robin gán per-slot trong spawner ([app.go:8723](../../app.go)):
   ```go
   slotPlatforms := regPlatformList(interactionCfg)         // reload realtime
   regPlatform = slotPlatforms[(slotIdx-1)%len(slotPlatforms)]
   ```
   → slot1→version[0], slot2→version[1], ... Slot **giữ nguyên version suốt đời** → keep-ip/keep-ua/keep-datr hoạt động y hệt single-version trong từng slot.
5. Cảnh báo nếu `len(regPlatforms) > maxThreads` ([app.go:8592](../../app.go)): chỉ `maxThreads` version đầu được chạy.

Tên folder kết quả (multi-version): ghép tag tất cả version `"s560-s561-s562"`; quá dài (>60 ký tự) → `"multiN"` ([app.go:7616](../../app.go)).

---

## 4. Build profile + UserAgent (3 mode UA)

**Mục tiêu**: mỗi account có profile (tên, ngày sinh, giới tính, contactpoint) + UA hợp lệ theo platform.

### 4.1 Profile gốc
Spawner gọi `webregister.RandomRegInput("", "", proxyStr)` ([app.go:8737](../../app.go)) → `facebook.RegInput` (định nghĩa [types.go:159](../../internal/facebook/types.go)): FirstName/LastName US, Birthday, Gender, Phone +84 VN default, Password rỗng.

Sau đó spawner override theo config:
- `PasswordReg` template ([app.go:8817](../../app.go)): `"Fb***2025*"` → mỗi `*` thay char random.
- `RegMode=Mail` ([app.go:8822](../../app.go)): sinh email từ `LeadDomainMail`, clear Phone.
- `RegMode=TempMail`/`MailTemp` ([app.go:8829](../../app.go)): clear cả 2 → worker tự acquire mail thật.
- `NameRegLocale=VN` ([app.go:8840](../../app.go)): override tên bằng DB tên VN.
- `PhoneMailMode=random-file` ([app.go:8847](../../app.go)): đọc `Config/Permanent/phone.txt` / `mail.txt`.
- `FmPhoneCode` ([app.go:8861](../../app.go)): convert E164 → local (`84912... → 0912...`).

### 4.2 Ba mode UA
UA được chọn **2 lần**: pre-gen cho UI (spawner, [app.go:8738](../../app.go)) + UA thật trong worker ([app.go:9154](../../app.go)). Logic 5 trạng thái (worker là nguồn truth):

| `UseOriginalUA` | `BuildUA` | `AddVirtualSpecAndroid` | Kết quả |
|---|---|---|---|
| `true` | – | – | **UA Gốc**: cố định theo platform (`originalUAForPlatformWithSim`), chỉ thay FBCR theo IP nếu `ReplaceCarrier` |
| `false` | `false` | `false` | **Pool UA**: lấy từ `Config/UserAgent/{kind}_UG.txt` (`fakeinfo.RandomUAFromPool`) |
| `false` | `false` | `true` | Pool UA + Dalvik prefix (`WrapWithDalvikPrefix`) |
| `false` | `true` | `false` | **Build UA**: build động từ `Config/DeviceInfo` |
| `false` | `true` | `true` | Build UA + Dalvik prefix |

**Pool FBAV split** (xem [add-facebook §1.4](add-facebook-reg-version.md#14-pool-fbav-split--regtxt-và-vertxt-mới-2026-05-28)): build UA cho reg dùng `versions_and_builds_reg.txt` qua `fakeinfo.RandomFbVersionReg()` ([app.go:8775](../../app.go)).

**Gotcha quan trọng** ([app.go:8758](../../app.go)): khi `BuildUA=true` + `RegMode∈{Phone,Random}` → **defer build UA** đến khi `CheckIP` biết countryCode (locale/carrier phải match IP). Lúc đó UI hiển thị UserAgent rỗng cho tới khi CheckIP xong; nếu CheckIP fail → **abort account** ([app.go:9011](../../app.go)).

Override UA chồng (đúng thứ tự): UA selection → `KeepUASuccess` (pin UA slot success, [app.go:9216](../../app.go)) → `TrackingIDReg` (thêm `XID/<random16>` vào UA, [app.go:9243](../../app.go); xem [add-facebook §9](add-facebook-reg-version.md#9-checking-ua-thêm-xid-vào-cuối-ua)).

Pool versions: `versions_and_builds_reg` chứa cặp `FBAV|FBBV` cho reg — chi tiết cách thêm version mới ở [add-facebook §1](add-facebook-reg-version.md#1-cập-nhật-file-versionbuild).

---

## 5. Datr / Cookie pool — trái tim của reg

> `datr` = "device authentication token", cookie định danh thiết bị. Facebook trust 1 datr đã "ấm" hơn datr mới sinh. Pool quản lý datr để: (a) tránh trùng giữa luồng, (b) giới hạn số lần tái dùng, (c) thu datr mới từ account thành công.

### 5.1 Nguồn datr ([app.go:7683–7926](../../app.go))
`CookieInitialMethod`:
- **`"new"`** ([app.go:7793](../../app.go)): sinh nội bộ `RegThreads × limit × 2` (tối thiểu 64) qua `sharedCookiePool.LoadGenerated(n)`.
- **`"file"`** / mặc định ([app.go:7807](../../app.go)): load từ:
  1. File user chọn (`CookieInitialFile`) hoặc `Config/Cookie/cookie_initial.txt` (seed nếu thiếu).
  2. Textarea paste (`CreateCookieList`).
  - Nếu method `"file"` mà **0 datr** → **DỪNG HOÀN TOÀN** ([app.go:7913](../../app.go)) (reg không datr → kết quả tệ).
- **`"tut"`** (`CreateType="tut"`): parse datr từ cookie list thành `tutDatrPool` ([app.go:7656](../../app.go)).

`cookieInitialLimit` default = **9** (match C# UI; KHÔNG để 9999 vô hạn vì 1 datr tái dùng nghìn lần → FB flag "over-reused device").

### 5.2 `PartitionedDatrPool` — pool dùng production
Định nghĩa [android/pool.go:161](../../internal/facebook/register/android/pool.go). Thiết kế **per-slot partition** (mỗi goroutine có queue riêng) thay vì round-robin chung → không bao giờ 2 luồng dùng cùng datr đồng thời.

Lifecycle API (gọi từ worker):
| Method | Khi nào | Hành vi | Vị trí |
|---|---|---|---|
| `Register(slotIdx)` | worker bắt đầu | tạo partition + **steal 1/n** datr từ các slot active để cân bằng | [pool.go:335](../../internal/facebook/register/android/pool.go) |
| `GetNext(slotIdx)` | mỗi lần reg | lấy datr `usage < maxUsage`, rotate về cuối queue; partition rỗng → fill từ pending → steal ½ richest slot → recycle exhausted | [pool.go:406](../../internal/facebook/register/android/pool.go) |
| `IncrementUsage(datr)` | sau mỗi reg (defer) | tăng counter; đạt limit → exhaust (recycle) hoặc remove (`deleteOnUsageLimit`) | [pool.go:493](../../internal/facebook/register/android/pool.go) |
| `RecordResult(datr,outcome)` | sau parse | đếm success/fail/checkpoint; đạt `maxCheckpoint` → remove | [pool.go:291](../../internal/facebook/register/android/pool.go) |
| `AddDatrRaw(datr)` | reg thành công | thêm datr MỚI → trả `true` nếu mới (trigger persistHook) | [pool.go:570](../../internal/facebook/register/android/pool.go) |
| `Unregister(slotIdx)` | worker kết thúc (defer) | trả datr còn lại về pending + phân phối lại | [pool.go:374](../../internal/facebook/register/android/pool.go) |

**Bug history quan trọng** ([pool.go:330](../../internal/facebook/register/android/pool.go)): với 200 luồng + 9901 datr, nếu chỉ `fillSlotLocked` thì slot 1-154 lấy hết pending → slot 155-200 nhận 0 datr → 30% slot "Thiếu datr". Fix: steal-on-Register + steal-on-GetNext (½ richest slot).

### 5.3 Pool dùng chung cho mọi platform
`sharedCookiePool := androidreg.NewPartitionedPool(cookieInitialLimit)` ([app.go:7789](../../app.go)) được **gán cho TẤT CẢ platform pool** ([app.go:7865](../../app.go)):
```go
allPlatformPools = {
  "Android": &androidreg.SharedPool, "S23": &s23reg.SharedPool,
  "S399": &s399reg.SharedPool, "WebAndroid": &webandroidreg.SharedPool,
  "iOS HTTP": &ioshttpreg.SharedPool, "Web": &webregister.SharedPool,
  "iOS562": &ios562reg.SharedDatrPool, "iOS563": &ios563reg.SharedDatrPool,
  + regSxxxPoolPointers()  // toàn bộ sXXX
}
for _, poolPtr := range allPlatformPools { *poolPtr = sharedCookiePool }
```
→ Mọi platform share 1 pool → user **hot-swap version giữa batch** mà không cần stop/start.

### 5.4 Persist & remove datr ([app.go:7744](../../app.go))
- `persistNewDatr(datr)`: chỉ khi `SaveNewDatr=true`. Ghi vào `runPoolPath` (`Pool{YYYYMMDD}_{N}.txt`) qua `cookie.AppendDatrToPool` + cộng vào tất cả pool RAM (`AddDatrRawNoPersist`) để pool count UI cập nhật realtime.
- `removeDatrEverywhere(datr)`: dedup qua `sync.Map removingDatr`, xóa khỏi file + tất cả pool.
- Hook gắn: `SetPersistHook` / `SetRemoveHook` / `SetPersistOnlyNewDatr(KeepDatrSuccess)` / `SetMaxCheckpoint` / `SetDeleteOnUsageLimit` / `SetMaxAgeMinutes` ([app.go:7822–7864](../../app.go)).
- `LimitDatrAge`: background ticker quét + `ExpireOldDatrs()` mỗi `maxAge/4` (≥30s).

### 5.5 Worker dùng datr
Trong worker, mỗi platform tự gọi `SharedPool.GetNext(slotIdx)` → set `profile.MachineID` → gửi làm `machine_id` (hoặc `datr` cookie / `x-fb-integrity-machine-id`). Spawner đăng ký/hủy slot vào pool tương ứng platform ([app.go:8908–8943](../../app.go)).

---

## 6. Chuẩn bị run resources khác

1. **Session pool keep-session** ([app.go:7935](../../app.go)): `newRunResources(runID, regPlatforms...)` ([app.go:138](../../app.go)) tạo `ioshttpreg.SessionPool` (iOS HTTP) + `webandroidreg.SessionPool` (WebAndroid) nếu list có. `publishGlobals()` gán global ref. Cleanup trong spawner defer ([app.go:174](../../app.go)) — đóng đúng pool của run + nil global nếu còn khớp (defense-in-depth chống stale pointer).
2. **iOS562 device profile pool** ([app.go:7893](../../app.go)): `ios562reg.NewDevicePool(5)` load từ `Config/Cookie/ios562_devices.txt`. Sau mỗi reg thành công, `register.go` tự `Add()` + persistHook ghi `DeviceID|FamilyDeviceID|MachineID`.
3. **Email pool** ([app.go:7955](../../app.go)): chỉ khi `VerifyEnabled`. Theo `MailProvider` tạo `emailrent.NewZeusXPool/NewDongVanFBPool/NewStore1sPool/NewMail30sPool` với `MailPoolBatch` (default 50). Wire `OnExhausted` (emit `email:pool-exhausted`) + `OnBought` (lưu `Config/RentMail/bought_<provider>.txt`).
4. **Writer + Counters** ([app.go:7646](../../app.go)): `resultpkg.NewWriter(outputPath)` + `CounterSet`, ticker auto-save 5s.
5. **Auto-restart ticker** ([app.go:8039](../../app.go)): poll config mỗi 5s; tới `AutoRestartMinutes` → emit trigger + transition `running → stopping` + cancel. Realtime apply (user bật/tắt giữa chừng có hiệu lực ngay).

---

## 7. Slot allocation + Split mode

### 7.1 Slots reg ([app.go:8097](../../app.go))
`freeSlotsReg := make(chan int, maxThreads)` chứa ID `1..maxThreads`. Dùng **channel có ID** thay vì counting semaphore → mỗi goroutine nhận 1 slot **UNIQUE, exclusive** (bug cũ: 2 goroutine cùng slotIdx emit về 1 row UI).

### 7.2 Split mode ([app.go:8126–8533](../../app.go))
`splitModeActive = SplitMode && VerifyEnabled`. Hai kiến trúc:
- **TRUE SPLIT (đang dùng khi split)**: reg worker reg xong → đẩy `splitVerifyJob` vào `splitVerifyCh` (buffer `maxThreads*5`, tối thiểu 500) rồi **return ngay** (giải phóng reg slot → reg full tốc độ). Pool VER riêng (`SplitVerifyThreads` goroutine, [app.go:8502](../../app.go)) đọc channel, verify với `verSlot` riêng (VER panel). REG dispatch xong → `close(splitVerifyCh)` → VER drain hết queue rồi thoát.
- **`regToVerCh`** ([app.go:8625](../../app.go)): **DEPRECATED** = `nil`. Từ 2026-05-15 Split Mode chỉ là **PURE UI option** (hiển thị 2 panel REG/VER); worker logic giống Normal Mode (1 worker = REG + VER inline). Các block `if regToVerCh != nil` đều skip.
- `splitWorkerCtx`: parent `a.ctx`, **KHÔNG bị cancel khi StopRegister** → account đã reg vẫn chạy hết verify; cancel ở spawner defer SAU `splitVerWg.Wait()`.

`verifySem` ([app.go:8106](../../app.go)): semaphore giới hạn số verify đồng thời trong split (`SplitVerifyThreads`).

---

## 8. Vòng đời 1 worker (reg 1 account)

**Input**: `slotIdx`, `prof` (profile + UA), `interactionCfg`, `regPlatform`. **Output**: account ghi file + (option) verify. Code: goroutine từ [app.go:8877](../../app.go).

1. **Recover panic** ([app.go:8892](../../app.go)): worker panic → log + emit `[PANIC]` status (không tắt im lặng).
2. **Register slot vào datr pool** theo platform ([app.go:8908](../../app.go)) + `defer Unregister`.
3. **Acquire proxy sticky** ([app.go:8948](../../app.go)): `regSticky.Acquire(ctx, slotIdx)`. `KeepIPSuccess` → success giữ proxy, fail thả về pool. `regSticky` tạo tại [app.go:8604](../../app.go).
4. **Render session** ([app.go:8969](../../app.go)): `proxy.RenderSessionIfIsProxyServer` → IP mới mỗi reg (session token). `ProxyKey = "slot_<idx>"` (tránh session poisoning khi share proxy).
5. **CheckIP** ([app.go:8993](../../app.go)): `proxy.CheckIP(ipCtx, prof.Proxy, ApiCheckIp)` (timeout 6s, parent = `ctx` để Stop hủy được). Trả `realIP` dạng `"89.200.217.100/cl"` → `displayProxy` + `countryCode="cl"`.
6. **Gen phone theo country** ([app.go:9026](../../app.go)): `fakeinfo.PhoneFromDatabase(countryCode)` → fallback `GeneratePhoneByCountry` → fallback VN. Bỏ qua khi Mail/TempMail mode.
7. **NewWorkerContext** theo platform ([app.go:9101–9146](../../app.go)): pin device/UA/locale/SIM/DeviceID qua lifetime worker (keep-session, FB trust hơn). Set locale (`LocaleFake=random`), connection type (`SimNetworkType`), UA options.
8. **UA selection** (xem §4.2) → `KeepUASuccess` → `TrackingIDReg`.
9. **Keep-session loop** ([app.go:9287](../../app.go)): `maxKeepSessionRegs`:
   - default = 1 (no loop).
   - = 10 cho iOS HTTP + cookie initial; = 5 cho S23/Sxxx/S399/Android/WebAndroid khi pool > 0.
   - Mỗi iteration: reload config realtime, regen fake info (attempt > 0), acquire TempMail (nếu mode), gọi `Register`.
   - **ctx vs regWorkerCtx**: `Register` dùng `regWorkerCtx` (KHÔNG cancel khi Stop → HTTP request đang chạy hoàn thành); `ctx` chỉ gate retry-delay + dispatch.
10. **Dispatch reg** ([app.go:9451](../../app.go)):
    ```go
    if s23WCtx != nil      { result = s23WCtx.Register(...) }
    else if sxxxWCtx != nil{ result = sxxxWCtx.Register(...) }
    else if s399WCtx != nil{ result = s399WCtx.Register(...) }
    else if androidWCtx    { result = androidWCtx.Register(...) }
    else if webandroidWCtx { result = webandroidWCtx.Register(...) }
    else if threadReg==nil { result = fail "không có registerer" }
    else                   { result = threadReg.Register(...) }   // ios562/ios563/web/...
    ```
    `threadReg, _ := facebook.NewRegisterer(regPlatform)` ([app.go:9081](../../app.go)) → factory ([factory.go:251](../../internal/facebook/factory.go)). Mỗi platform package tự `RegisterPlatformRegisterer` trong `init()`; app.go blank-import để trigger.

---

## 9. Per-platform REG flow (chi tiết)

> Tất cả endpoint dùng hằng số [constants.go](../../internal/facebook/constants.go): `BaseURLBGraph="https://b-graph.facebook.com"`, `BaseURLGraph="https://graph.facebook.com"`, `AndroidOAuthToken="350685531728|62f8ce9f74b12f84c123cc23437a4a32"`.

### 9.1 Android FB4A Bloks (`android`, và base cho `s23`, `sXXX`)
File: [android/register.go](../../internal/facebook/register/android/register.go), [android/body.go](../../internal/facebook/register/android/body.go), [android/extras.go](../../internal/facebook/register/android/extras.go).

Flow `WorkerContext.Register` ([android/register.go:144](../../internal/facebook/register/android/register.go)):
1. `sess.clearCookies()` (tránh cookie pollution giữa regs).
2. Override profile từ input; lấy `MachineID` ← `input.TutDatr` hoặc `SharedPool.GetNext(slotIdx)` ([register.go:185](../../internal/facebook/register/android/register.go)); `defer SharedPool.IncrementUsage`; set cookie `datr=<machineID>`.
3. Contactpoint (email > phone). Thiếu cả 2 → fail.
4. **GetPwdKey** ([extras.go:76](../../internal/facebook/register/android/extras.go)): `POST b-graph//pwd_key_fetch` (giữ double-slash của C#) → RSA public key + key_id. `EncryptPassword` ([extras.go:131](../../internal/facebook/register/android/extras.go)): RSA-PKCS1(rand 32B key) + AES-GCM(pw) → blob `#PWD_FB4A:2:{ts}:{base64}`. Fail → fallback plaintext `#PWD_FB4A:0:{ts}:{pw}`.
5. **POST `b-graph/graphql`** ([register.go:267](../../internal/facebook/register/android/register.go)) body từ `buildRegisterBody` ([body.go:55](../../internal/facebook/register/android/body.go)).
   - Body là form-urlencoded với field `variables=<url-encoded JSON 7 lớp>` + `client_doc_id=1199408042526631289603660492`.
   - **`createAccountVariablesV22`** ([body.go:89](../../internal/facebook/register/android/body.go)) build JSON từ trong ra ngoài: L4 `reg_info` (~180 field flat) → L3 `server_params`/`client_input_params` → L2 `{"params":esc(L3)}` → L1 → L0 `{params, scale:3, nt_context}` → `url.QueryEscape`.
   - **`machine_id`** xuất hiện cả trong `reg_info`, `server_params`, `client_input_params` ([body.go:174](../../internal/facebook/register/android/body.go), [body.go:361](../../internal/facebook/register/android/body.go), [body.go:387](../../internal/facebook/register/android/body.go)).
   - Field tạo random: `attestation_result` (keyHash/data/signature DER ECDSA giả), `safetynet_token`, `device_network_info` (theo SIM profile).
6. **Parse** `parseRegisterResponse` ([extras.go:464](../../internal/facebook/register/android/extras.go)):
   - **Strip toàn bộ backslash** trước (`strings.ReplaceAll(body,"\\","")` — port C#).
   - Detect block: `"couldn't create an account"`, `"integrity_block"`, `"create_failure"+created_userid,null`.
   - Regex: token `EAAAAU[...]{20,}` (ưu tiên) → `EAA[...]` fallback; UID `c_user","value":"(\d{10,})"`; xs/fr/datr theo `name":"<k>","value":"(...)"`.
   - Compose cookie: `c_user=...;xs=...;locale=...;fr=...;datr=...;`.
7. **fetchXZeroEH** ([register.go:356](../../internal/facebook/register/android/register.go)): sleep 1-2s rồi batch POST `mobile_zero_campaign` lấy `eligibility_hash` → append `;x_zero_eh=...` vào cookie.
8. `RecordResult(datr,"success")` + `AddDatrRaw(parsed.DATR)` → `RegResult{Success, UID, Cookie, AccessToken (EAAAAU), Password, UserAgent, DeviceID, FamilyDeviceID}`.

> S23 / sXXX kế thừa cấu trúc body này; chỉ khác `doc_id` / `bloks_versioning_id` / `styles_id` / device theo version. Cách thêm version: [add-facebook §2](add-facebook-reg-version.md#2-tạo-package-reg-backend).

### 9.2 iOS native FBIOS (`ios562` / `ios563`)
File: [ios562/register.go](../../internal/facebook/register/ios562/register.go), [ios562/parse.go](../../internal/facebook/register/ios562/parse.go), [ios562/http.go](../../internal/facebook/register/ios562/http.go).

> KHÁC `ios` (= MFB web m.facebook.com) và `ioshttp`. Đây là **native app FBIOS**: `graph.facebook.com/graphql` + OAuth app-token, UA `FBAN/FBIOS`.

Flow `doRegisterAccount` ([ios562/register.go:47](../../internal/facebook/register/ios562/register.go)):
1. Resolve contactpoint, locale theo country, fake profile.
2. **Build profile**: ưu tiên `SharedDevicePool.GetNext()` (DeviceID/FamilyDeviceID), fallback `BuildProfile` random ([register.go:110](../../internal/facebook/register/ios562/register.go)).
3. **Datr override** ([register.go:129](../../internal/facebook/register/ios562/register.go)): `SharedDatrPool.GetNext(slotIdx)` → `profile.MachineID` (gửi qua header `x-fb-integrity-machine-id`, [http.go:180](../../internal/facebook/register/ios562/http.go)). DatrPool cung cấp lịch sử session tin cậy, DevicePool cung cấp device ID.
4. **Session TLS** ([http.go:51](../../internal/facebook/register/ios562/http.go)): chọn `profiles.Safari_IOS_*` theo iOS major version.
5. **Encrypt password** (`#PWD_WILDE:2` RSA, fallback `#PWD_FB4A:0`).
6. **POST `graph.facebook.com/graphql`** gzip ([register.go:178](../../internal/facebook/register/ios562/register.go)) qua `sess.postGzip` (body gzip + `content-md5` header, [http.go:89](../../internal/facebook/register/ios562/http.go)). Headers map 1:1 capture FBIOS ([http.go:161](../../internal/facebook/register/ios562/http.go)).
7. **Parse round 1** `parseCreateAccountResponse` ([parse.go:222](../../internal/facebook/register/ios562/parse.go)):
   - Response là **Bloks bytecode (S-expression)** bọc JSON-escape. Strip backslash trước.
   - UID: `(eud <số>)` ([parse.go:20](../../internal/facebook/register/ios562/parse.go)) hoặc fallback cookie `c_user`.
   - Token: `EAAAAA[A-Za-z0-9]{100,}` ([parse.go:24](../../internal/facebook/register/ios562/parse.go)) — **EAAAAAY** (6 chữ A), siết prefix để không match nhầm trace ID.
   - Cookie: c_user/xs/fr/datr → `composeCookie`.
   - Full success = UID + (cookie hoặc token) → return.
   - **Nosess** (UID có, chưa có session): `extractPartialTokens` ([parse.go:100](../../internal/facebook/register/ios562/parse.go)) bóc `fb_partially_created_reg_info`, `srnonce`, `sessionless_crypted_user_id` từ Bloks DSL keys/values list (offset +4 do `flow_info` JSON tách thành 5 token).
8. **Round 2..N** ([register.go:202](../../internal/facebook/register/ios562/register.go)): khi nosess + có Partial → `buildCreateAccountRound2(profile, Partial)` gọi lại `create.account` đến khi full hoặc cạn (cap 3 vòng — capture cho thấy chain `[158]→[162]→[164]`).
9. Lưu device vào pool, thu datr mới (`SharedDatrPool.AddDatrRaw`), trả `RegResult` + `Srnonce` + `SessionlessCryptedUID` (truyền sang verify iOS).

> iOS562 thường trả **UID + cookie nhưng chưa có token** → message `"cần verify để lấy token"`. Token `EAAAAAY` được lấy lúc **VERIFY iOS** (login CAA), KHÔNG phải EAAAAU Android. Quy tắc: xem [add-facebook §13.6 (iOS Native verify)](add-facebook-reg-version.md#136-ios-native-fbios-verify-ios562-ios563--bắt-buộc-token-eaaaaay) + [§13.7](add-facebook-reg-version.md#137-login-at-verify--tokencookie-hiển-thị-realtime-cập-nhật-2026-05-31).
>
> `registerAccount` wrapper ([register.go:27](../../internal/facebook/register/ios562/register.go)) track outcome datr (`SharedDatrPool.RecordResult`).

### 9.3 WebAndroid (Chrome Mobile, cookie-based)
File: [webandroid/register.go](../../internal/facebook/register/webandroid/register.go).

Flow `WorkerContext.Register` ([webandroid/register.go:116](../../internal/facebook/register/webandroid/register.go)):
1. Profile = ChromeAndroid (UA + Chrome version + device + viewport + dpr) — **trình duyệt**, không phải Dalvik FB4A.
2. Seed datr ← `input.TutDatr` / `SharedPool.GetNext(slotIdx)` ([register.go:176](../../internal/facebook/register/webandroid/register.go)) → `seedCookies(sess, datr)`.
3. **Step 1 GET `m.facebook.com/`** ([register.go:218](../../internal/facebook/register/webandroid/register.go)) → `parsePageTokens` lấy `fb_dtsg`/`lsd`/`versioningID`. Thiếu `fb_dtsg` → fail.
4. **Step 2 POST `/async/wbloks/fetch/`** ([register.go:266](../../internal/facebook/register/webandroid/register.go)) (URL `postURL(versioningID)`) body `buildRegisterBody(params)`. Referer = `sess.finalURL` (sau redirect). → `parseUID(respBody)`. Thiếu UID → fail + `RecordResult fail`.
5. **Step 3 GET `m.facebook.com/`** lần 2 ([register.go:296](../../internal/facebook/register/webandroid/register.go)) collect cookies.
6. **Step 4 checkpoint** ([register.go:310](../../internal/facebook/register/webandroid/register.go)): nếu URL chứa "checkpoint" → GET logout URL bypass.
7. Extract cookie theo thứ tự FB `datr;sb;c_user;xs;fr;pas` (`getCookiesFBOrder`); fallback datr từ HTML. Đảm bảo có `locale=`.
8. **Lấy token qua `/auth/login`** ([register.go:396](../../internal/facebook/register/webandroid/register.go)): chỉ khi `!webreg.SkipAuthLoginAtReg`. WebAndroid reg **không tự có token** → phải login lại. Gọi `webreg.FetchAndroidTokenLegacy(...)` (xem §10) → `EAAAAU`.
9. Track datr success + `AddDatrRaw(newDatr)` + `AppendDatr` ra `datr_pool.txt`.

### 9.4 Web MFB + s399 legacy REST
- **Web** (`web`): cookie-based qua web flow, lấy token cũng qua `/auth/login` (gate `SkipAuthLoginAtReg`).
- **s399** ([s399/register.go:156](../../internal/facebook/register/s399/register.go)): FB4A native CŨ, **2 step REST** (KHÔNG Bloks):
  1. **Step 1 POST `b-graph/app/users`** (friendly-name `registerAccount`, [register.go:301](../../internal/facebook/register/s399/register.go)) → JSON `{new_user_id, machine_id}`. **Chỉ hỗ trợ email** (capture `email=...@gmail.com`); phone → fail.
  2. **Step 2 POST `b-graph/auth/login`** (friendly-name `authenticate`, [register.go:339](../../internal/facebook/register/s399/register.go)): dùng `new_user_id` làm email, password plaintext `#PWD_FB4A:0`, `machine_id` từ step 1, `api_key`+`sig` MD5+app token → JSON `{access_token: "EAAAAU...", session_cookies}`.
  - Compose cookie từ `session_cookies`; thu datr `AddDatrRaw`.

> Pattern 2-step REST của s399 chính là pattern được tái dùng cho `FetchAndroidTokenLegacy` (§10).

---

## 10. EAA token pre-fetch (Android-family verify)

**Mục tiêu**: account reg cookie-only (WebAndroid/Web) hoặc reg platform không tự có EAAAAU, nhưng verify cần token Android → login lấy token TRƯỚC khi ghi file/verify.

Block tại [app.go:9501](../../app.go), chạy SAU reg success, TRƯỚC ghi file (để cả file output lẫn split-ver channel mang token mới):
1. Điều kiện: reg success + `VerifyEnabled` + (`verifyNeedsEAA(ApiVerifyPlatform)` HOẶC `platformNeedsAndroidLoginToken(regPlatform)`) + token hiện chưa phải `EAAAAU`.
   - `verifyNeedsEAA` ([app.go:425](../../app.go)): true cho api android/token, s22-s26, s4xx, s5xx, s399, s273.
   - `platformNeedsAndroidLoginToken` ([app.go:463](../../app.go)): Android-family.
2. `webregister.SkipAuthLoginAtReg` ([app.go:7681](../../app.go)): set `true` khi `VerifyEnabled || verifyIsIOS(...)` → **reg cookie-only KHÔNG login lúc reg**. Block này chỉ chạy khi `!SkipAuthLoginAtReg`.
   - Triết lý: **"Login nằm ở lúc verify, không phải reg"**. iOS verify tự login lấy EAAAAAY → token EAAAAU lúc reg vô dụng + sai loại. Xem [add-facebook §13.7](add-facebook-reg-version.md#137-login-at-verify--tokencookie-hiển-thị-realtime-cập-nhật-2026-05-31).
3. `FetchAndroidTokenLegacy(...)` ([android_token_legacy.go:196](../../internal/facebook/register/web/android_token_legacy.go)): UA **bắt buộc Android FB4A** (`androidUAForToken`), không dùng UA reg (có thể Web/iOS/Chrome). `machine_id` = datr extract từ cookie.
   - Impl `fetchAndroidTokenLegacyImpl` ([android_token_legacy.go:75](../../internal/facebook/register/web/android_token_legacy.go)): `POST b-graph/auth/login` form (UID làm `email`, password `#PWD_FB4A:0`, `sig` MD5 sorted + app secret, `access_token`=`350685531728|...`). Parse JSON `access_token` + `session_cookies`; fallback regex `EAA`.
4. Chỉ chấp nhận token có prefix `EAAAAU` ([app.go:9529](../../app.go)) → `result.AccessToken = tok` + emit status. Token khác prefix → cảnh báo "không dùng cho verify".

---

## 11. Sau reg — ghi file, thu datr, đẩy verify

### 11.1 Emit token realtime ([app.go:9552](../../app.go))
Reg success → emit `register:token` (UID + token + cookie) ngay → bảng register cập nhật không chờ verify xong.

### 11.2 Ghi file ([app.go:9563](../../app.go))
Map `RegResult → Account`, status: `success→"live"`, message chứa `"checkpoint"→"checkpoint"`, `"block"→"blocked"`. Chạy `go saveRegOutcome(...)` ([app.go:2210](../../app.go)) async:
- `FormatReg` với `IsNVR=true` (register xong = **NVR** not-verified-yet).
- success → append `SuccessReg.txt` + `SuccessReg_NVR_UG_<instance>` (UA) + `SuccessNVREmail/Phone`.
- checkpoint/blocked/unknown → file tương ứng.
- `EmailMeta` (TempMail) được persist vào file (cần cho split-mode verify Restore).
- **Không lưu email** vào file account thành công (yêu cầu user).

### 11.3 Thu datr mới ([app.go:9628](../../app.go))
`extractDatrFromCookieLine(result.Cookie)` → `persistNewDatr(datr)`. TUT mode → thêm vào `dynamicTutPool`. Tích lũy phone/email thành công vào `Config/Permanent/` cho mode "Random File".

### 11.4 Đẩy sang verify
- **Split** ([app.go:9666](../../app.go)): tạo `splitVerifyJob{acc, prof, displayProxy, regResult}` đẩy vào `splitVerifyCh` (bounded → backpressure tự nhiên; ctx-aware để Stop không treo). VER pool xử lý qua `runSplitVerify` ([app.go:8170](../../app.go)) → `runner.RunOneAccountAt(splitWorkerCtx, acc, runCfg, dateFolder, verSlot, ...)`.
- **Inline (non-split)** ([app.go:9699](../../app.go)): cùng worker. Update `acc.UserAgent` sang verify-platform UA (`pickUAForVerifyPlatform`), delay `DelayVeriReg` (ctx run-scoped), build `VerifyConfig` + `RunConfig` (với `GetVerifyConfig`/`GetVerifyPlatform` reload realtime) → `runner.RunOneAccountAt`.
- Cả 2 dùng `AccountInput` ([scheduler.go:22](../../internal/runner/scheduler.go)) mang `DeviceID/FamilyDeviceID/Srnonce/SessionlessCryptedUID/Email/EmailMeta` để verify dùng lại context reg.
- Sau verify: `saveVerifyOutcome` ghi `SuccessVerify`, record stats (`recordVerifyOutcome`/`recordBuildUAVerVersion`/`recordMailDomainOutcome`), auto-upload nếu live (`enqueueForUpload`).

---

## 12. Stop, drain, complete

**StopRegister** (Wails-bound): transition `running → stopping` + `a.registerCancel()` (cancel `ctx` = dispatch) + `regWorkerCancel`/`splitWorkerCancel` tùy thời điểm.

Phân biệt context (rất quan trọng):
- **`ctx`** (run-scoped, hủy khi Stop): gate dispatch loop, retry delay, CheckIP, sleep `DelayVeriReg`.
- **`regWorkerCtx`** / **`splitWorkerCtx`** (parent `a.ctx`, KHÔNG hủy khi Stop): HTTP request reg + verify đang chạy chạy hết → account dở dang không bị abort.

**Spawner defer** ([app.go:8631](../../app.go)) — thứ tự cleanup:
1. `wg.Wait()` — chờ mọi reg worker xong.
2. Split: `close(splitVerifyCh)` → `splitVerWg.Wait()` → `splitWorkerCancel()` (VER drain xong). **Phải drain TRƯỚC `regCounters.Stop()`** vì verify ghi qua `regCounters`.
3. `persistUsedUnused(emailPool)` (lưu mail used/unused).
4. `regWorkerCancel` + `regBatchCancel` + clear caches + `regCounters.Stop()` + `regSticky.ReleaseAll()`.
5. `runRes.Cleanup()` (đóng session pool, nil global nếu khớp).
6. `registerCancel = nil` (nếu gen khớp) + transition `stopping → idle` + emit `register:complete`.

---

## 13. Edge case & gotcha tổng hợp

- **Validation early-return giữ state idle** ([app.go:7389](../../app.go)): mọi early-return chỉ `Unlock + return` — KHÔNG cần reset state vì transition `→running` chỉ xảy ra sau khi validation pass.
- **Pool cạn giữa chừng** ([app.go:9409](../../app.go)): method `"new"` → generate thêm; `"file"` → reload file → fallback đọc 100 dòng cuối `SuccessVerify_No2FA.txt` → `IsCompletelyEmpty()` → skip slot.
- **CheckIP fail + BuildUA Phone/Random** ([app.go:9011](../../app.go)): abort account (UA + phone phụ thuộc country code).
- **Multi-version > maxThreads**: chỉ N version đầu chạy ([app.go:8592](../../app.go)).
- **Batch emitter throttle** ([app.go:8546](../../app.go)): gom status 200ms; `sentCache` skip entry không đổi → tránh flood IPC. `frontendHidden` throttle thêm khi minimize.
- **Slot UNIQUE qua channel ID** ([app.go:8097](../../app.go)): không dùng counting semaphore (bug 2 goroutine cùng row UI).
- **iOS562 nosess loop cap 3** ([ios562/register.go:202](../../internal/facebook/register/ios562/register.go)): vượt → trả UID cookie-only, verify hoàn tất.
- **android parse strip backslash** ([android/extras.go:468](../../internal/facebook/register/android/extras.go)) + ios562 ([parse.go:226](../../internal/facebook/register/ios562/parse.go)): bắt buộc trước regex (response bọc nhiều lớp JSON-escape).
- **KeepSession iOS = 10 / Android = 5** ([app.go:9266](../../app.go)): warm session → success rate cao hơn nhưng device rotation ít hơn.
- **Token chỉ emit lúc verify XONG** (chưa realtime giữa verify) — KNOWN issue ghi ở [add-facebook §13.7](add-facebook-reg-version.md#137-login-at-verify--tokencookie-hiển-thị-realtime-cập-nhật-2026-05-31).

---

## 14. Vì sao thiết kế vậy (rationale)

- **1 pool dùng chung mọi platform** (§5.3): cho phép hot-swap version mid-batch — port UX mềm hơn C# (cost RAM ~1-3MB).
- **PartitionedDatrPool thay round-robin chung**: tránh 2 luồng cùng datr → giảm FB "over-reused device" flag; steal-on-Register/GetNext giải quyết phân phối không đều khi luồng ≫ datr.
- **EAA login dồn về verify, reg cookie-only** (§10): token đúng loại theo platform verify (iOS↔EAAAAAY, Android↔EAAAAU đối xứng) → tránh lấy token sai loại + lãng phí login.
- **regWorkerCtx tách khỏi ctx**: Stop nhanh (ngừng nhận acc mới) nhưng không vứt account dở (HTTP đang chạy hoàn thành) — UX "dừng êm".
- **TRUE SPLIT pool riêng**: reg không bị verify block → reg full tốc độ; VER drain độc lập.
- **State machine 3 trạng thái**: chống Start mới khi run cũ chưa thật sự release (workers + session pool + HTTP buffer) — single-semantic hơn pattern 2-flag cũ.

---

## 15. Liên kết runbook thêm version

Khi cần **thêm version reg mới** (không phải hiểu flow chạy), dùng:
- [add-facebook §1 — Cập nhật file version/build](add-facebook-reg-version.md#1-cập-nhật-file-versionbuild)
- [§2 — Tạo package Reg backend](add-facebook-reg-version.md#2-tạo-package-reg-backend)
- [§3 — Khai báo platform backend (factory.go)](add-facebook-reg-version.md#3-khai-báo-platform-backend)
- [§4 — Nối package vào app logic (isRegPlatformSxxx, worker context...)](add-facebook-reg-version.md#4-nối-package-mới-vào-app-logic)
- [§5 — Cập nhật UI chọn version](add-facebook-reg-version.md#5-cập-nhật-giao-diện-chọn-version)
- [§6 — Mock preview frontend](add-facebook-reg-version.md#6-cập-nhật-mock-preview-frontend)
- [§7 — Kiểm tra version không lạc UA](add-facebook-reg-version.md#7-kiểm-tra-version-không-bị-lạc-ua)
- [§8 — Test bắt buộc](add-facebook-reg-version.md#8-test-bắt-buộc)
- [§9 — Checking UA (XID)](add-facebook-reg-version.md#9-checking-ua-thêm-xid-vào-cuối-ua)
- [§10 — Checklist Reg](add-facebook-reg-version.md#10-checklist-reg-trước-khi-báo-xong)

Luật token/login iOS: [§13.6](add-facebook-reg-version.md#136-ios-native-fbios--reg--verify--login) + [§13.7](add-facebook-reg-version.md#137-login-at-verify--tokencookie-realtime).
