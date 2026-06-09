# 06 — Các Subsystem phụ trợ (Email / Proxy / 2FA / Upload / Result / Counters / Stop-Cleanup)

> Tài liệu mô tả **chi tiết tới từng dòng code** các subsystem phụ trợ bao quanh luồng Register/Verify Facebook của HVR. Tập trung vào **flow chạy thực tế**: thứ tự gọi hàm, file:line, edge case, gotcha, và lý do thiết kế. Phần luồng reg/verify chính + luật token/login iOS xem [add-facebook-reg-version.md](./add-facebook-reg-version.md) (§13.6/§13.7).
>
> Link code dùng đường dẫn tương đối từ `docs/facebook/` lùi 2 cấp về repo root: `../../`.

---

## 0. Sơ đồ tổng — các subsystem cắm vào đâu trong 1 account verify

```
                  ┌──────────────────────── RunVerify (app.go:2280) ───────────────────────┐
                  │ 1. getSharedProxyManager() → proxy.Manager (manager.go)                │
                  │ 2. NewStickyManager(KeepIPSuccess, rawAcquire)  (sticky.go)            │
                  │ 3. emailPool = NewZeusXPool/... (rent/pool.go)  ← shared cả run        │
                  │ 4. verifyWriter = NewWriter(outputPath)  (writer.go)                   │
                  │ 5. verifyCounters = NewCounterSet(writer).Start(5s)  (counters.go)     │
                  │ 6. runner.RunConfig{ AcquireProxy, OnAccountDone, ... }                │
                  └────────────────────────────────┬──────────────────────────────────────┘
                                                    │  FIFO worker pool (scheduler.go:150)
                                ┌───────────────────▼──────────────────────┐
                                │ runOneAccount (scheduler.go:410)          │
                                │                                           │
   PROXY ──────────────────────► AcquireProxy(ctx,workerID) → proxyStr      │
   (sticky per slot)            │ + defer release(success = Status=="Live") │
                                │ + CheckIP() bg → cột IP CHẠY              │
                                │                                           │
                                │ ver.Verify(...) → verifybase.Run          │
   EMAIL ───────────────────────►   email.New(Options{Pool,ProxyOverride})  │  (factory.go)
   (temp/rent + reuse)         │     ├ Restore(EmailMeta) hoặc CreateEmail   │  (service.go)
                                │     └ WaitForCode(maxRetry,pollMs)         │
   2FA ─────────────────────────►   spec.Enable2FA() nếu cfg.Enable2FA+Live  │  (security/android)
                                │                                           │
                                └──────────────┬────────────────────────────┘
                                               │ OnAccountDone(...) callback (app.go:2956)
            ┌──────────────────────────────────┼─────────────────────────────────────┐
            ▼                                   ▼                                      ▼
  RESULT (writer.go)                 COUNTERS / STATS                        UPLOAD (live only)
  saveVerifyOutcome →                recordVerifyOutcome                     UploadAvatarS23 (async)
  SuccessVerify*/Die/Unknown         recordMailDomainOutcome                 enqueueForUpload → banclone
  + dispatch detail files            recordBuildUAVerVersion                 GetNewDatrOnLive (async)
                                     counters.FbAppVersion.Incr
```

Tất cả subsystem được **wire 1 lần ở RunVerify/RunRegister** rồi truyền xuống worker qua `runner.RunConfig`. Worker (scheduler) gọi callback ngược lên app.go để ghi file / cập nhật stats / kích upload. Khi Stop → cancel context dispatch, defer dọn sticky/pool/counters.

---

## 1. EMAIL subsystem — `internal/email`

Mục tiêu: cấp 1 địa chỉ email (temp hoặc rent) cho mỗi account khi verify, chờ OTP của Facebook, và (tùy chọn) tái sử dụng email đã tạo lúc register.

### 1.1 Interface chung — `Service`

File: [internal/email/service.go](../../internal/email/service.go)

```go
type Service interface {
    CreateEmail(ctx) (string, error)                       // tạo email tạm, trả địa chỉ
    WaitForCode(ctx, maxRetry int, intervalMs int) (string, error) // poll inbox lấy OTP
    GetEmail() string
    Close()
}
```

Ngoài ra có 3 **capability interface tùy chọn** (provider implement thêm, caller dùng type-assert):

| Interface | Method | Dùng cho | file:line |
|---|---|---|---|
| `Snapshotter` | `Snapshot() (string,error)` / `Restore(creds string) error` | Reuse mail register→verify | service.go:43-54 |
| `Releaser` | `Release(ctx) error` | Trả mail về pool khi verify fail sớm | service.go:62-65 |

Helper an toàn (type-assert + nil-check):
- `SnapshotIfPossible(svc)` → service.go:69
- `RestoreIfPossible(svc, creds)` → service.go:83
- `ReleaseIfPossible(ctx, svc)` → service.go:95 (best-effort, ignore lỗi)

> **Gotcha**: provider chưa implement `Snapshotter` → reuse mail tự động fall back về `CreateEmail` mới (không lỗi). Provider chưa implement `Releaser` → `ReleaseIfPossible` no-op.

### 1.2 Factory — chọn provider + chính sách proxy

File: [internal/email/factory.go](../../internal/email/factory.go) — hàm `New(opts Options)` (factory.go:32).

**Quyết định proxy (priority, factory.go:35-44):**
1. `effectiveProxy = opts.ProxyOverride` (proxy riêng từ `proxy_tempmail.txt` khi `UseProxyTempMail=true`).
2. Nếu rỗng → fallback `opts.ProxyStr` (= proxy FB đang verify account → mail đọc CÙNG IP với request FB, anti-fraud tốt hơn).
3. `rentMailProxy = opts.ProxyOverride` (rent mail **KHÔNG** fallback ProxyStr — tránh dùng datacenter IP mà provider có thể đã blacklist).

`New()` là một `switch opts.Provider` khổng lồ map ~50 key → constructor (factory.go:46-338). Hai nhóm:
- **Temp mail** (factory.go:48-180): `moakt`, `mail1sec`, `mailtempcom`, `dropmail`, `guerrillamail`, `mailtm`, `i2b`, `vietxf`, `mailhv`, ... → constructor nhận `effectiveProxy`.
- **Rent mail** (factory.go:182-334): `zeus-x`, `dongvanfb`, `store1s`, `mail30s`, `muamail`, `unlimitmail`, `wmemail`, ... → nhận `rentMailProxy`, set `Pool`, `OnStatus`, `OTPHotmailPriority`.

Provider chết được map sang error rõ ràng (vd `tempmailor` Cloudflare block factory.go:96, `mailcx` DNS chết factory.go:99). Provider không biết → `unknown provider` (factory.go:336).

### 1.3 Options — `email.Options`

File: [internal/email/options.go](../../internal/email/options.go)

Các field quan trọng:
- `Provider`, `ProxyStr`, `ProxyOverride` — xem §1.2.
- `Pool *CredPool` (options.go:27) — shared pool cho rent provider (zeus-x/dvfb/store1s/mail30s). Re-export type từ `rent.CredPool` (options.go:7).
- `CustomUsername` (options.go:105) — khi `FmUserTmpMail=true`, dùng username format từ login info thay vì UUID random.
- `OTPHotmailPriority` (options.go:117) — nguồn đọc OTP ưu tiên cho 7 provider Hotmail OAuth2 (`"dongvan"` default / `"unlimit"`).
- Per-provider API key/product (ZeusXApiKey, DvfbApiKey, ...).

### 1.4 Rent mail — `CredPool` (batch purchasing)

File: [internal/email/rent/pool.go](../../internal/email/rent/pool.go)

Vấn đề: rent mail tính phí mỗi con → mua từng cái khi nhiều thread chạy song song sẽ tốn nhiều API call + chi phí. `CredPool` gom mua batch.

**Mô hình mua (`Get` — pool.go:126):**
1. Pool còn mail → lấy ngay (pool.go:137-142).
2. Pool rỗng + đang có goroutine khác mua (`refilling=true`) → chờ 100ms rồi loop lại lấy từ batch đó (pool.go:145-153) — **serialize** để 1 lần mua phục vụ nhiều thread.
3. Pool rỗng + chưa ai mua → goroutine này set `refilling=true`, gọi `buyFn(ctx, firstBatchSize)` (pool.go:155-178).

`firstBatchSize` mặc định 50, cấu hình từ UI `MailPoolBatch` (app.go:2538). Sau batch đầu, **mỗi lần cạn lại mua đúng `firstBatchSize` con** (không phải 1).

**Tái sử dụng — `Return` (pool.go:205):** verify fail SỚM (add mail HTTP error / checkpoint) mà mail CHƯA bị FB consume → `Return` prepend mail vào **đầu** pool để account khác lấy trước (ưu tiên reuse, tránh hết hạn token OAuth2). **GOTCHA quan trọng (pool.go:202):** KHÔNG `Return` mail đã add thành công vào FB (OTP timeout) — sẽ gây lỗi "email already used".

**Bookkeeping cuối run:**
- `boughtIndex` (pool.go:42) ghi MỌI mail đã mua.
- `PartitionUsedUnused()` (pool.go:86) phân loại used (đã rời pool) vs unused (còn trong pool) để dump ra file lúc kết thúc run — `persistUsedUnused(a.emailPool)` (app.go:2532).
- `OnExhausted` (pool.go:40) fire 1 lần khi pool cạn + mua fail → app.go wire callback emit event + log (app.go:2568).

> Pool được tạo 1 lần cho cả run tại RunVerify (app.go:2542-2567), gán vào `a.emailPool` và truyền xuống mọi worker qua `VerifyConfig.EmailPool` (app.go:2884) → tất cả goroutine **dùng chung 1 instance**.

### 1.5 Ví dụ rent provider — ZeusX

File: [internal/email/rent/zeus_x.go](../../internal/email/rent/zeus_x.go)

- `CreateEmail` (zeus_x.go:92): nếu có `pool` → `pool.Get(ctx)` lấy creds; không → `buyLegacy` (mua đơn lẻ, retry khi hết hàng zeus_x.go:110).
- `buyBatch` (zeus_x.go:179): GET `api.zeus-x.ru/purchase?quantity=N`, parse `Accounts[]` → `[]EmailCred{Email,Password,RefreshToken,ClientId}`.
- `WaitForCode` (zeus_x.go:268): default maxRetry=3, interval=2000ms; mỗi vòng gọi `fetchOTPFromDVFB` → `ReadOTPWithPriority(...)`.
- `Snapshot/Restore` (zeus_x.go:330/348): serialize 4 field `email/password/refresh_token/client_id` ra JSON → verify Restore để đọc inbox mà không mua mới.
- `Release` (zeus_x.go:363): trả về **local pool** (zeus-x không có refund API server) + set `emailAddr=""` chống double-return.

**Out-of-stock vs hết tiền (zeus_x.go:233):** `isOutOfStock` PHẢI loại trừ `"insufficient balance"` — nếu nhầm balance là hết hàng thì pool retry vô hạn thay vì fail nhanh (gotcha đã sửa, comment zeus_x.go:230).

### 1.6 OTP reader priority (Hotmail OAuth2)

File: [internal/email/rent/otp_reader.go](../../internal/email/rent/otp_reader.go)

`ReadOTPWithPriority` (otp_reader.go:38): 2 reader đọc cùng mailbox Microsoft qua refresh_token+client_id:
- `OTPSourceDongVan` → `tools.dongvanfb.net/api/get_code_oauth2` (~2.6s, default).
- `OTPSourceUnlimit` → `smail1s.com/get_messages?mode=oauth` (~4.5s).

Flow: gọi primary theo priority; primary err → fallback reader còn lại; cả 2 fail → trả lỗi primary. `notify` log fallback lên UI.

### 1.7 Ví dụ temp mail có cache — mail-temp.com

File: [internal/email/temp/mailtempcom.go](../../internal/email/temp/mailtempcom.go)

- Email sinh **client-side** (không call API create): `user = realisticLocalPart()`, `domain = random từ list` (mailtempcom.go:84).
- **Domain cache 48h (`mailTempComCacheTTL`, mailtempcom.go:25):** `getOrFetchDomains` (mailtempcom.go:96) ưu tiên cache memory → cache file → fetch web (scrape regex từ homepage). Cache file path set lúc startup qua `SetMailTempComDomainCachePath` (mailtempcom.go:60); fallback `mailTempComDefaultDomains` (mailtempcom.go:28) khi fetch fail.
- `WaitForCode` (mailtempcom.go:184): default maxRetry=12, poll 2000ms. `pollOnce` (mailtempcom.go:214) GET `/temp-mail-box/` với `Cookie: surl={domain}/{user}` → parse HTML (regex `#email-table` → body blocks → `ExtractCode`).

### 1.8 Proxy pool riêng cho mail

File: [internal/email/proxypool.go](../../internal/email/proxypool.go)

Hai pool tách biệt (tránh đụng proxy FB):
- `proxy_tempmail.txt` — `LoadTempMailProxies` (proxypool.go:63) / `PickTempMailProxy` (proxypool.go:102, random không consume).
- `proxy_rentmail.txt` — `LoadRentMailProxies` (proxypool.go:128, auto-migrate từ `proxy_gmail.txt` legacy proxypool.go:134) / `PickRentMailProxy` (proxypool.go:177, rỗng thì fallback dùng chung temp pool).

`IsRentMailProvider(provider)` (proxypool.go:56) — verify layer dùng để biết pick từ pool nào. Set `rentMailProviders` (proxypool.go:43): zeus-x, muamail, unlimitmail, dongvanfb, store1s, mail30s, wmemail.

### 1.9 Reuse email register → verify (TempMail reuse)

Đây là điểm nối email với reg/verify, xem [add-facebook-reg-version.md](./add-facebook-reg-version.md) phần TempMail. Tóm tắt flow runtime:

1. **Register** tạo temp mail, sau khi reg OK → `SnapshotIfPossible(svc)` lấy creds → ghi vào file SuccessReg dạng cột `|MM:<base64(json)>` (format `FormatReg`, [result/format.go](../../internal/result/format.go):61-64).
2. File reg được load lại làm input verify; `EmailMeta` parse qua `ParseEmailMetaFromLine` (format.go:74) → đặt vào `AccountInput.EmailMeta` (scheduler.go:46).
3. Scheduler forward `session.Email` + `session.EmailMeta` (scheduler.go:499-500).
4. **Verify** (verifybase): nếu `session.Email != "" && session.EmailMeta != "" && RestoreIfPossible(emailSvc, EmailMeta)` → `reuseMail=true`, skip CreateEmail + skip AddEmail (run.go:368-376), log prefix `[REUSE]`. Bước AddEmail skip ở run.go:507-509.

### 1.10 Chờ OTP trong verify — cấu hình maxRetry/poll

File: [internal/facebook/verify/verifybase/run.go](../../internal/facebook/verify/verifybase/run.go):511-549

```go
waitSec := cfg.TimeDelaySendCode; if waitSec<=0 { waitSec=30 }   // tổng giây chờ
pollMs := cfg.WaitMailMs;         if pollMs<=0  { pollMs=2000 }   // delay mỗi poll
maxRetry := waitSec*1000/pollMs;  if maxRetry<1 { maxRetry=1 }    // số lần poll
code, err := emailSvc.WaitForCode(ctx, maxRetry, pollMs)
```

- `StartOTPHeartbeat` (run.go:525) emit message "đang chờ OTP" mỗi 5s lên UI để user thấy luồng còn sống.
- Nếu timeout + `cfg.SendAgainCode=true` → resend OTP rồi `WaitForCode` lần 2 (run.go:534-548).
- **Trả mail về pool:** mọi nhánh OTP timeout gọi `email.ReleaseIfPossible(ctx, emailSvc)` (run.go:531/546) — mail chưa bị consume thì trả về pool reuse. `WaitMailMs` = UI giây × 1000 (app.go:2847).

---

## 2. PROXY subsystem — `internal/proxy`

Mục tiêu: cấp proxy cho mỗi account, hỗ trợ sticky-per-worker (giữ IP sau success), rotating session, và proxy riêng cho mail vs FB.

### 2.1 Manager — chọn provider

File: [internal/proxy/manager.go](../../internal/proxy/manager.go)

`ManagerConfig.Provider` (manager.go:14) quyết định nguồn proxy:
- `"proxy"` / `"proxy_fixed"` → dùng `Pool` nội bộ parse `ProxyList` (manager.go:69).
- `"tinsoft"`/`"shoplike"`/`"netproxy"`/`"minproxy"`/`"proxy_farm"` → khởi tạo commercial provider pool (manager.go:72-86), mỗi pool quản semaphore theo `ThreadPerIP`.
- `""`/`"none"`/`"hma"`/`"fpt"`/`"xproxy"` → KHÔNG khởi tạo pool.

`IsConfigured()` (manager.go:104): false cho `none/hma/fpt/xproxy`; với `proxy/proxy_fixed` cần `pool.Len()>0`; commercial cần `provider.Len()>0`. App dùng để set `RequireProxy` (app.go:2908).

`Acquire(ctx)` (manager.go:136) → `(proxyStr, releaseFunc, error)`:
- `none/hma/fpt` → `("", noop, nil)` chạy không proxy (manager.go:138).
- `proxy` → round-robin `pool.Next()` (manager.go:142).
- `proxy_fixed` → luôn proxy đầu `pool.Fixed()` (manager.go:148).
- `xproxy` → **chưa hỗ trợ** → error (manager.go:155).
- default → delegate `provider.Acquire(ctx)` (manager.go:163).

> **FPT** (dial-up) và **xproxy** (cần service URL + device management) trong code hiện tại chưa được quản lý pool → `IsConfigured()=false`, `Acquire` trả rỗng/error. Config field `FptKeys`, `XproxyServiceUrl/Type/List/ThreadPerIp/RunType` chỉ được lưu trong settings (app.go:4065-4070) — chờ implement.

### 2.2 Pool round-robin

File: [internal/proxy/pool.go](../../internal/proxy/pool.go)

`NewPool(lines)` split theo `\n` (pool.go:19). `Next()` (pool.go:33) round-robin index có mutex; `Fixed()` (pool.go:45) luôn trả phần tử [0]. Port C# `qProxyRotate` (Dequeue/Enqueue FIFO).

### 2.3 Sticky proxy per worker — KeepIPSuccess

File: [internal/proxy/sticky.go](../../internal/proxy/sticky.go)

Port C# `KeepIPSuccess`: 1 account thành công → worker slot **giữ nguyên** proxy cho account kế; fail → release.

`StickyManager` (sticky.go:25) bọc 1 `AcquireFn` raw, map `entries[workerID]→stickyEntry{proxy,release}` (sticky.go:31).

**`Acquire(ctx, workerID)` (sticky.go:50)** → `(proxyStr, func(success bool))`:
1. Nếu `enabled` và đã pin slot → trả lại entry cũ (sticky.go:53-57).
2. Chưa pin → gọi `acquire(ctx)` raw, lưu entry (sticky.go:60-67).
3. release closure → `m.release(workerID, success)`.

**`release(workerID, success)` (sticky.go:73):** `keep := enabled && success`. keep → giữ entry; ngược lại → xóa entry + gọi raw release về pool.

**`ReleaseAll()` (sticky.go:94)** — giải phóng toàn bộ entry còn pin khi batch kết thúc. App `defer verifySticky.ReleaseAll()` (app.go:2492).

**Wiring (app.go:2480):**
```go
verifySticky := proxy.NewStickyManager(interaction.KeepIPSuccess, func(ctx) (string, func(), error) {
    mgr := a.getSharedProxyManager()        // reload realtime
    p, rel, err := mgr.Acquire(ctx)
    p = proxy.RenderSessionIfIsProxyServer(p)  // rotate session ID → IP mới
    return p, rel, nil
})
```

> **GOTCHA workerID:** sticky cache theo `workerID` (slot ổn định 0..maxThreads-1, scheduler.go:187), KHÔNG theo goroutine ID. `RunOneAccount(workerID=0)` deprecated vì làm mọi worker share 1 entry → cùng proxy/session (scheduler.go:884). Dùng `RunOneAccountAt(workerID)` cho pool mode (scheduler.go:894). Retry-Unknown pass 2 dùng `workerID + maxThreads` offset để force acquire proxy mới (scheduler.go:269-273).

### 2.4 Render proxy + session rotation

File: [internal/proxy/client.go](../../internal/proxy/client.go)

`RenderSessionIfIsProxyServer(proxyStr)` (client.go:50): rotate session token trong username → mỗi lần gọi = IP khác (rotating proxy). Hỗ trợ nhiều format/provider qua regex (client.go:15-41): `-sid-XXXX-t-NN`, `-region-XXX`, `-zone-XXX` (711proxy), `-ssid-` (NiceProxy), `_session-` (ProxyShare), `-sessid-` (lunaproxy), `-session-...-sessTime-` (abcproxy), `-sessionduration-` (smartproxy). Proxy tĩnh (không match pattern) → trả nguyên (client.go:110).

`FormatProxyURL(proxyStr)` (client.go:283) — chuẩn hóa thành URL `scheme://user:pass@host:port` cho `http.Transport`. `CreateClient(proxyStr, timeout)` (client.go:260) — tạo `*http.Client` có proxy.

Convention country suffix: proxy có thể có hậu tố `/<cc>` (vd `u:p@1.2.3.4:8080/vn`) → `extractCountryFromProxy` (scheduler.go:1040, chỉ nhận đúng 2 ký tự alpha) → dùng sinh UA country-aware (scheduler.go:531).

### 2.5 Acquire flow trong 1 account (scheduler)

File: [internal/runner/scheduler.go](../../internal/runner/scheduler.go):448-480

1. `proxyStr := acc.Proxy` (ưu tiên proxy riêng của account, scheduler.go:450).
2. Rỗng + có `AcquireProxy` → `proxyStr, releaseProxy = config.AcquireProxy(ctx, workerID)` (scheduler.go:453); `defer releaseProxy(result.Status=="Live")` (scheduler.go:454) — Live = giữ IP cho account kế.
3. `RequireProxy=true` + proxy rỗng → abort, status `"error"` "Không có proxy" (scheduler.go:459) — KHÔNG chạy bằng IP máy.
4. `OnRawProxy(accID, proxyStr)` (scheduler.go:466) → emit `verify:raw-proxy` cập nhật cột PROXY ngay.
5. `CheckIP` chạy **background goroutine** (scheduler.go:471-480): `proxy.CheckIP(ctx, proxyStr, APICheckIpAuto)` ([checkip.go](../../internal/proxy/checkip.go):74, chain fallback ip-api→adspower→luna→ipify) → `OnProxy(accID, actualIP)` cập nhật cột IP CHẠY. CheckIP fail → KHÔNG emit (cột để trống). Không block login.

### 2.6 Proxy riêng reg vs verify, temp vs rent

- Reg và Verify đều dùng **chung** `getSharedProxyManager()` (app.go:2475/8604) để chia sẻ cache commercial provider (cùng key → cùng proxy trong window).
- Sticky riêng cho mỗi run: `verifySticky` (app.go:2480), `regSticky` (app.go:8604), `splitVerifySticky` (app.go:10525).
- Proxy cho mail: factory ưu tiên `ProxyOverride` (pick từ `proxy_tempmail.txt`/`proxy_rentmail.txt`) → tách biệt proxy FB (§1.2, §1.8).

`ProxySettings` (UI) → `ManagerConfig` qua các field provider/keys/threads. Validation ở RunVerify (app.go:2440-2453) check key bắt buộc của provider đã chọn.

---

## 3. 2FA subsystem — `internal/facebook/security`

Mục tiêu: sau verify Live, nếu user bật, enable TOTP 2FA và lưu secret vào account (`TwoFA`).

### 3.1 Wiring trong verify

File: [internal/facebook/verify/verifybase/run.go](../../internal/facebook/verify/verifybase/run.go):659-678

```go
if spec.Enable2FA != nil && cfg.Enable2FA && status == "Live" {
    emailOTPFn := func(maskedEmail string, _ int) string {
        c, e := emailSvc.WaitForCode(ctx, 3, 3000)  // 2FA reauth có thể cần OTP email
        if e != nil { return "" }
        return c
    }
    secret, err2fa := spec.Enable2FA(ctx, session, uid, machineID, deviceID, emailOTPFn, notify)
    if err2fa != nil { notify("2FA enable failed (non-fatal)") }  // KHÔNG fail verify
    else { twoFAKey = secret }
}
```

`cfg.Enable2FA` từ `VerifyConfig` (types.go:391), set từ UI (app.go:2891). `spec.Enable2FA` là func per-platform (types.go:64). `twoFAKey` được nhét vào `VerifyResult.TwoFA` (run.go:701) → propagate qua scheduler `result.TwoFA` (scheduler.go:776) → `OnAccountDone` → ghi cột 2FA file SuccessVerify (format.go:114).

> **GOTCHA**: 2FA fail là **non-fatal** — account vẫn Live, chỉ thiếu 2FA. Account Live không 2FA → ghi vào `SuccessVerify_No2FA.txt` (app.go:2104), Live có 2FA → `SuccessVerify.txt`.

### 3.2 Android SecurityManager — Enable2FA flow chi tiết

File: [internal/facebook/security/android/security.go](../../internal/facebook/security/android/security.go)

`Enable2FA(ctx, session)` (security.go:74) yêu cầu `session.Token` (OAuth). Tạo `androidSecAPI` (security.go:84) với tls_client (bogdanfinn), `defer client.CloseIdleConnections()` (security.go:99). Đăng ký qua `RegisterPlatformSecurityManager(PlatformAndroid, ...)` (security.go:113).

**`turnOnTwoFactor` (security.go:151)** — loop tối đa 3 lần reauth (Bloks GraphQL `graph.facebook.com/graphql`):
1. **SelectMethod** (security.go:156) — click "Enable 2FA" (`fnSelectMethod`).
2. **Generate TOTP Key** (security.go:162) — `fnGenKey`; kiểm tra response cần reauth password không (security.go:175).
3. Nếu cần → **Reauth Password** (security.go:184, `fnReauthPwd`, password encode `#PWD_FB4A:0:ts:pwd` security.go:348), thành công thì `continue` loop.
4. **Extract TOTP secret** từ QR URL (security.go:203, `extractTOTPSecret` regex `secret=` security.go:477).
5. **Submit TOTP** (security.go:214, `fnSubmitTOTP`); response chứa `FX_TWO_FACTOR_STATUS:is_enabled` → success ngay (security.go:225).
6. Nếu cần email OTP → `handleEmailOTPFor2FA` (security.go:248): send_code → `emailOTPFn` lấy OTP email → verify_code → re-submit TOTP.

**TOTP generator (security.go:519):** tự implement RFC 6238 HMAC-SHA1 (KHÔNG gọi external `2fa.live` như C#) — base32 decode secret, counter = `UnixTime/30`, dynamic truncation → 6 chữ số. `TwoFAResult{Success, Secret}` (security.go:226). Secret này = key user nhập vào app authenticator sau này.

Có các SecurityManager khác: [security/web](../../internal/facebook/security/web/security.go), [security/webandroid](../../internal/facebook/security/webandroid/security.go). AccountsCenter flow (fx.settings.security) là phần Bloks endpoint `com.bloks.www.fx.settings.security.two_factor.*` (security.go:52-57).

---

## 4. UPLOAD subsystem — avatar + site (banclone.pro)

Cả hai chạy **async sau verify Live**, không block flow, được kích trong `OnAccountDone` (app.go:3030-3127).

### 4.1 Upload avatar — `UploadAvatarS23`

File: [internal/facebook/interaction/android/upload_avatar.go](../../internal/facebook/interaction/android/upload_avatar.go)

Kích hoạt (app.go:3030-3067): chỉ khi `interaction.UploadAvatar && doneAcc.Token != ""`. `avatarDir = interaction.AvatarFolderPath` (default `Config/Avatar`, app.go:3035). Chạy goroutine riêng có recover + `context.WithTimeout(workerCtx, 60s)` (app.go:3057). Emit `verify:batch-status` cho UI.

**Flow `UploadAvatarS23(ctx, proxyStr, token, ua, avatarDir)` (upload_avatar.go:39):**
1. `pickRandomImage(avatarDir)` (upload_avatar.go:264) — random .jpg/.jpeg/.png từ folder.
2. Tính `md5` + `uploadID = {md5}-0-{len}-{ts}-{ts+rand}` (upload_avatar.go:49).
3. **rupload GET** check offset (upload_avatar.go:82, `Resumable-Upload-Get`).
4. **rupload POST** binary ảnh (upload_avatar.go:107, headers `x-entity-length/name/type`, `offset:0`) → parse `{"h": handle}` (upload_avatar.go:135).
5. **setAvatarNUX** (upload_avatar.go:147) — POST graphql Bloks `com.bloks.www.bloks.nux.profilepicture.async.upload` với handle, body gzip + nested params (upload_avatar.go:190).

`defer client.CloseIdleConnections()` (upload_avatar.go:54) per-call free TCP/TLS. Lỗi upload chỉ log warning (app.go:3060), KHÔNG ảnh hưởng kết quả verify.

### 4.2 Upload site — đẩy account lên banclone.pro

File config: `UploadSiteConfig` (app.go:4699-4711), lưu `Config/Settings/uploadsite.json` (`SaveUploadSiteConfig` app.go:4727 / `LoadUploadSiteConfig` app.go:4742). Default code/apikey app.go:4713.

Kích hoạt sau verify Live (app.go:3098-3126):
```go
if uploadCfg := a.LoadUploadSiteConfig(); uploadCfg.Ver.Enabled && uploadCfg.Code != "" && uploadCfg.ApiKey != "" {
    country := extract từ UA (FBLC) (app.go:3101)
    // format line: nếu KHÔNG có 2FA → FormatReg; có 2FA → FormatVerify (email luôn để rỗng)
    a.ensureUploadRunning(uploadCfg)   // bảo đảm runner goroutine đang chạy
    a.enqueueForUpload(uploadLine)     // đẩy vào in-memory queue
}
```

**Upload Site Runner** (app.go:4753+): in-memory queue + retry exponential backoff + dedup UID per-session + soft-stop (drain hết queue mới exit). `pushToBanclone` (app.go:5561) POST batch lên `banclone.pro/api/importAccount.php` (transport tái dùng `bancloneTransport` app.go:5531, `CloseIdleConnections` sau run app.go:5194). `bancloneAdminLogin` (app.go:5673) cho admin operations. Stats/log per-session: `upload_stats.json`, `upload_push_log.txt` (app.go:4765). `uploadSiteCancel/uploadSiteGen` (app.go:332) — generation counter chống cleanup stale.

> **GOTCHA**: "acc cũ" từ session trước KHÔNG auto-upload — mỗi run fresh start (app.go:4761). Email KHÔNG được upload (user request, app.go:3113/3119).

### 4.3 GetNewDatrOnLive (post-live, liên quan cookie)

app.go:3071-3096: account Live + có token → goroutine gọi `fetchNewDatrFromAccountUA` (GraphQL profile-switcher) lấy datr mới → ghi pool file `datrNewVer{YYYYMMDD}.txt` + add vào `androidreg.SharedPool`. Path tạo 1 lần ở RunVerify (`verifyDatrPoolPath`, app.go:2511-2514).

---

## 5. RESULT files — `internal/result`

Mục tiêu: ghi kết quả reg/verify ra thư mục output theo định dạng C# (pipe-delimited), thread-safe, dedup UID.

### 5.1 Kiến trúc 2 tầng

File: [internal/result/store.go](../../internal/result/store.go) (low-level) + [internal/result/writer.go](../../internal/result/writer.go) (high-level).

- **store.go** — `AppendLine` (store.go:65), `UpsertByUID` (store.go:105), `Overwrite` (store.go:159). Per-file mutex `lockFor(path)` (store.go:33) → 2 file khác nhau ghi parallel không block. Lazy `ensureDir` (store.go:46). `UpsertByUID` ghi atomic tempfile→rename (`writeAllLinesAtomic` store.go:187) chống corrupt.
- **writer.go** — `Writer{root}` (writer.go:46) gắn 1 folder run; `Append/UpsertUID/Overwrite` + `Path` (writer.go:64). `NewWriter(root)` (writer.go:53) không mkdir ngay (lazy).

### 5.2 Tên file chuẩn (constants)

File: [internal/result/writer.go](../../internal/result/writer.go):11-39

| Loại | Constant | Tên file | Cấu trúc |
|---|---|---|---|
| Reg | `FileSuccessReg` | `SuccessReg.txt` | `uid\|pass\|cookie\|token\|time\|country\|NVR` |
| Reg | `FileSuccessNVRPhone/Email` | `SuccessNVR_Phone/Email.txt` | theo login type |
| Reg | `FileCheckpoint` | `Checkpoint.txt` | reg bị checkpoint |
| Reg | `FileBlocked` | `Blocked.txt` | reg bị FB block |
| Reg | `FileUnknownBlockType` | `UnknownReg.txt` | reg unknown |
| Reg | `FileSuccessButErrorCheckLive` | `Success_but_error_checklive.txt` | |
| Verify | `FileSuccessVerify` | `SuccessVerify.txt` | `uid\|pass\|2fa\|cookie\|token\|email\|fullname\|time\|country` |
| Verify | `FileSuccessVerifyNo2FA` | `SuccessVerify_No2FA.txt` | Live nhưng chưa 2FA (format 6 field) |
| Verify | `FileDieAfterVerify` | `Die.txt` | **UPSERT theo UID** (dedup) |
| Verify | `FileUnknownErrorCheckLiveDieApi` | `Unknown.txt` | verify unknown |
| Error | `FileChinaMailCantGetCode`/`FileBuyMailCantGetCode` | mail không trả code | |
| Counter | `FileFbAppVersionSuccess` | `FbAppVersisonSuccess.txt` (typo giữ giống C#) | auto-save overwrite |

Tên file động theo status code (writer.go:87-119): `VerifyFailedFile(status)` → `VerifyFailed_<status>.txt`, `LoginFbFailedFile`, `LoginGmailFile`, `SuccessVerifyUGFile` (UA file, đã bỏ suffix instance writer.go:112).

### 5.3 Format line

File: [internal/result/format.go](../../internal/result/format.go)

- `FormatReg(d RegData, now)` (format.go:42): `uid|pass|cookie|token[|email]|time|country[|NVR][|MM:base64]`. Email chèn chỉ khi có; `IsNVR` → `|NVR`; `EmailMeta` → `|MM:` + base64 (chống ký tự `|` trong JSON phá format, format.go:62). Time format C# `02-01-2006 15:04:05`.
- `FormatVerify(d VerifyData, now)` (format.go:109): `uid|pass|2fa|cookie|token|email|fullname|time|country` + `|Tạo TKQC OK...|currency` nếu `HasCalledOpenAds`.
- `ParseEmailMetaFromLine(line)` (format.go:74): tìm cột `MM:` → base64 decode → trả EmailMeta cho reuse (§1.9). Backward-compat: không có cột → trả `""`.

### 5.4 saveVerifyOutcome — ghi kết quả 1 account

File: [app.go](../../app.go):2060 (`saveVerifyOutcome`)

Gọi async trong `OnAccountDone` (app.go:3027). writer nil/root rỗng → no-op (app.go:2068). Resolve country: `acc.Location` → FBLC từ UA → locale từ cookie (app.go:2075-2085). Build line `FormatVerify` (app.go:2086). Switch status (app.go:2098):
- **live** (app.go:2099): có 2FA → `SuccessVerify.txt` (FormatVerify); không 2FA → `SuccessVerify_No2FA.txt` (FormatReg 6 field, app.go:2104). Ghi UA vào `SuccessVerifyUG.txt` (app.go:2116). `counters.FbAppVersion.Incr(fbVer)` (app.go:2121).
- **die** (app.go:2124): `UpsertUID(Die.txt, line)` — dedup theo UID.
- **default/unknown** (app.go:2128): `Append(Unknown.txt, line)`.

Sau đó dispatch detail files: `DispatchVerifyDetails(s, message, line)` (app.go:2135).

> **GOTCHA**: file write chỉ qua `OnAccountDone → saveVerifyOutcome`. `SaveAccountToFolder` cũ đã bỏ để tránh ghi 2 lần / tạo trùng `Die.txt` + `DieAfterVerify.txt` (scheduler.go:866-868).

### 5.5 Dispatch detail (sub-status → file)

File: [internal/result/dispatch.go](../../internal/result/dispatch.go)

`DispatchVerifyDetails(status, message, line)` (dispatch.go:35) → `[]DetailDispatch{File, Content, Upsert}`. Detect sub-status từ `message`:
- die + checkpoint → `VerifyFailed_Checkpoint.txt` (dispatch.go:49); die + detect code → `VerifyFailed_<code>.txt` (dispatch.go:53, `detectVerifyFailCode` dispatch.go:135).
- unknown: "cant get code" → `ChinaMail_/BuyMail_CantGetCode.txt` (dispatch.go:63); "login fb failed" → `LoginFbFailed_<code>.txt`; "login gmail" → `LoginGmail_<code>.txt` (dispatch.go:90).

`DispatchRegDetails` (dispatch.go:108) tương tự cho reg (Writer đã ghi SuccessReg/Checkpoint/Blocked ở saveRegOutcome, chỉ thêm detail unknown).

### 5.6 Error log per-session

File: [internal/result/errorlog.go](../../internal/result/errorlog.go)

`Writer.LogError(context, err)` (errorlog.go:29) → append `errordata.txt` với timestamp. `Writer.RecordPanic(recovered, context)` (errorlog.go:62) — ghi panic stack (caller phải tự `recover()` rồi pass vào). Khác `%APPDATA%/logs/` (toàn app): `errordata.txt` scope theo từng run folder.

### 5.7 Đường dẫn output

`Writer` được tạo từ `outputPath` (= `cfgOverride.OutputPath` hoặc `interaction.OutputPath`, app.go:2828). Tên folder dạng `VerifyMfb_20260418_103015` / `RegAndroid_...`. `Path()` join root + filename (writer.go:64).

---

## 6. COUNTERS / STATS

Hai hệ thống tách biệt: (a) **CounterSet** ghi file auto-save (theo run), (b) **stats trong App struct** cho bảng thống kê UI.

### 6.1 Counter map thread-safe

File: [internal/result/counter.go](../../internal/result/counter.go)

`Counter` = map key→count có RWMutex (counter.go:22). `Incr(key)` (counter.go:34, skip key rỗng), `Add`, `Get`, `Snapshot` (copy), `Reset`. `DumpSorted()` (counter.go:101) → text `key|count\n` sort giảm dần theo count (port C#: country/version nhiều success nhất ở đầu). Rỗng → trả `""` (không tạo file rỗng).

### 6.2 CounterSet — auto-save ticker

File: [internal/result/counters.go](../../internal/result/counters.go)

`CounterSet` (counters.go:27) bind 1 `Writer`, hiện chỉ có 1 counter active `FbAppVersion` (track `{version}|{build}`). `Start(ctx, interval)` (counters.go:53) chạy goroutine ticker (interval clamp ≥1s, counters.go:59) → mỗi tick `Flush()` (counters.go:98) ghi overwrite `FbAppVersisonSuccess.txt` (port C# WinForms Timer 5-10s). `Stop()` (counters.go:73) cancel + đợi loop exit + flush lần cuối (idempotent).

**Wiring (app.go:2789-2792):**
```go
verifyCounters := resultpkg.NewCounterSet(verifyWriter)
verifyCounters.Start(a.ctx, 5*time.Second)
defer verifyCounters.Stop()      // flush cuối khi RunVerify return
```
`saveVerifyOutcome` gọi `counters.FbAppVersion.Incr(fbVer)` khi Live (app.go:2121). Reg cũng có `regCounters` (app.go:7647-7650), split verify `splitVerifyCounters` (app.go:10205).

### 6.3 Stats trong App struct (bảng UI)

File: [app.go](../../app.go) — 3 nhóm stats (map + mutex), record trong `OnAccountDone` (app.go:2964-2968):

| record fn | map | Get fn (UI) | file:line |
|---|---|---|---|
| `recordVerifyOutcome(platform, success)` | `verifyStats` (per version) | `GetVerifyStats()` → `[]RegStatRow` | app.go:7134 / 7157 |
| `recordMailDomainOutcome(email, isLive)` | `mailDomainStats` (per domain) | `GetMailDomainStats()` → sort Live desc | app.go:7184 / 7210 |
| `recordBuildUAVerVersion(fbav, success)` | `buildUAVerStats` (per FBAV) | `GetBuildUAVerStats()` | app.go:7287 / 7327 |

- `recordVerifyOutcome` (app.go:7134): resolve platform display name, tăng `Success`/`Fail`. Hỗ trợ multi-version: platform = platform thực tế round-robin (`verifyPlatform` từ scheduler, app.go:2961).
- `recordMailDomainOutcome` (app.go:7184): lấy domain sau `@`, tăng `Veri`/`Live`.
- `recordBuildUAVerVersion` (app.go:7287): `extractFBAV(ua)` (app.go:7245, regex `FBAV/...`).
- Reg song song: `recordBuildUARegVersion` (app.go:7266) + `buildUARegStats`.

`stats.Reporter` ([internal/stats/reporter.go](../../internal/stats/reporter.go)) là counters atomic đơn giản (RegSuccess/Fail, VerifyLive/Die/Error, ProxyFail) cho status bar — phần lớn TODO (reporter.go:7), record stats chính nằm trong App struct.

---

## 7. STOP / CLEANUP

### 7.1 Hai context tách biệt (Stop "mềm")

File: [app.go](../../app.go):2501-2521

- `poolCtx` (app.go:2501) — kiểm soát **dispatch** accounts mới. Cancel = dừng đẩy account vào pool.
- `workerCtx` (app.go:2507) — kiểm soát **HTTP requests** của worker. Cancel = abort request đang chạy.

Gán cancel vào struct trong lock (app.go:2518-2521) tránh data race với StopVerify.

`RunConfig.WorkerCtx = workerCtx` (app.go:2907) → worker dùng workerCtx cho `runOneAccount` (scheduler.go:229) nên KHÔNG bị abort khi chỉ cancel poolCtx — chạy hết các bước HTTP hiện tại.

### 7.2 StopVerify — soft-stop

File: [app.go](../../app.go):3861 (`StopVerify`)

```go
if !a.isRunning { ... return "Không có verify nào đang chạy" }
if a.verifyStopping.Load() { return "Đang dừng..." }   // chống double-stop
a.verifyStopping.Store(true)
if a.verifyCancel != nil { a.verifyCancel() }  // CHỈ cancel dispatch — workers chạy tiếp
```

> **Thiết kế "soft-stop"**: Stop chỉ cancel **dispatch** (poolCtx). Workers đang chạy dở **KHÔNG bị abort** → chạy hết account hiện tại (đọc OTP xong, ghi kết quả) rồi exit tự nhiên. `IsVerifyStopping()` (app.go:3890) cho FE disable nút Start tránh overlap.

Scheduler phía dưới: worker loop check `ctx.Err()` (= poolCtx) đầu mỗi vòng (scheduler.go:198), nếu cancelled → ghi result "Đã dừng" + tiếp tục drain queue nhanh (scheduler.go:199-207). `RunVerify` chờ `wg.Wait()` (scheduler.go:238) — block đến khi mọi worker exit.

### 7.3 Cleanup khi RunVerify kết thúc (defer)

Tại RunVerify, các defer dọn theo LIFO:
- `defer verifySticky.ReleaseAll()` (app.go:2492) — trả mọi proxy còn pin về pool.
- `defer verifyCounters.Stop()` (app.go:2792) — flush counter lần cuối + dừng ticker.
- Đầu run đã `a.emailPool.Close()` + `persistUsedUnused` cho pool cũ (app.go:2531-2535) — giải phóng credential slice + dump used/unused.

`CredPool.Close()` (rent/pool.go:217) set `creds=nil`. `StickyManager.ReleaseAll()` (sticky.go:94) gọi raw release từng entry. Worker pool tự dọn qua `wg.Wait` + recover panic (scheduler.go:191-195).

### 7.4 StopRegister

File: [app.go](../../app.go):10899 — Register dùng **state machine** (`registerState`: idle/running/stopping, app.go:7377) + `registerGen` generation counter chống event/cleanup stale từ run cũ (app.go:7387). Gating: chỉ Start được khi state=idle (app.go:7378-7384). Reg cũng dùng `regSticky` + `regCounters` cleanup tương tự.

### 7.5 Quit app (cleanup toàn cục)

`RequestQuit` (app.go:3905) set `confirmedQuit` → `runtime.Quit`. `main.go OnBeforeClose` check `IsConfirmedQuit()` (app.go:3913): false → block + `EmitQuitConfirm()` (app.go:3919) show dialog FE; true → allow close + cleanup. Transport `CloseIdleConnections` (banclone app.go:5194, avatar/2FA per-call defer) để free TCP/TLS khi run kết thúc.

---

## 8. Bảng tra nhanh điểm nối (file:line)

| Subsystem | Wiring point (app.go) | Core impl |
|---|---|---|
| Email factory | `email.New(Options{...})` verifybase run.go:319 | factory.go:32 |
| Email pool (rent) | app.go:2542-2567 (`NewZeusXPool`...) | rent/pool.go:104 |
| Email reuse | run.go:368 (`RestoreIfPossible`) | service.go:83, format.go:74 |
| Email proxy pool | factory.go:35 (ProxyOverride) | proxypool.go:102/177 |
| Proxy manager | `getSharedProxyManager()` app.go:2475 | manager.go:65 |
| Proxy sticky | `NewStickyManager` app.go:2480 | sticky.go:39 |
| Proxy acquire (per acc) | `AcquireProxy` scheduler.go:453 | manager.go:136 |
| Proxy render session | app.go:2489 | client.go:50 |
| CheckIP (cột IP) | scheduler.go:474 | checkip.go:74 |
| 2FA enable | run.go:661 (`spec.Enable2FA`) | security/android/security.go:74 |
| Upload avatar | app.go:3047 (`UploadAvatarS23`) | upload_avatar.go:39 |
| Upload site | app.go:3099 (`enqueueForUpload`) | app.go:4753+ / pushToBanclone app.go:5561 |
| Result writer | `NewWriter(outputPath)` app.go:2784 | writer.go:53, store.go |
| saveVerifyOutcome | app.go:3027 (async) | app.go:2060 |
| Counter auto-save | app.go:2789 (`NewCounterSet.Start`) | counters.go:53, counter.go |
| Stats (UI) | app.go:2964-2968 (record*) | app.go:7134/7184/7287 |
| Stop verify | `StopVerify` app.go:3861 | poolCtx cancel app.go:2519 |
| Cleanup | defer ReleaseAll/Stop/Close app.go:2492/2792 | sticky.go:94, pool.go:217 |

---

## 9. Edge cases & gotchas (tổng hợp)

1. **Reuse mail chỉ skip CreateEmail+AddEmail khi đủ cả 3:** `session.Email != "" && session.EmailMeta != "" && RestoreIfPossible` (run.go:368). Thiếu 1 → tạo mail mới.
2. **Return mail về pool chỉ với mail PRISTINE** (chưa add thành công vào FB). OTP timeout (mail đã add) → KHÔNG return, tránh "email already used" (pool.go:202, run.go:531/546).
3. **out-of-stock ≠ hết tiền** — phân biệt để không retry vô hạn khi hết tiền (zeus_x.go:230-249).
4. **Sticky cache theo workerID slot**, không theo goroutine. Retry-Unknown dùng offset slot để ép proxy mới (scheduler.go:269).
5. **RequireProxy=true** → proxy rỗng = abort account "error" (không chạy IP máy, scheduler.go:459). `IsConfigured()` quyết định flag này (app.go:2908).
6. **FPT / xproxy chưa implement** — config lưu nhưng `Acquire` trả rỗng/error (manager.go:155).
7. **2FA fail = non-fatal** — account vẫn Live; Live không 2FA → `SuccessVerify_No2FA.txt` (app.go:2104).
8. **Upload avatar/site = async post-live**, lỗi chỉ log, không ảnh hưởng verify. Upload site bị abort khi Stop run (workerCtx parent), avatar timeout 60s (app.go:3057).
9. **Die ghi UPSERT theo UID** — tránh duplicate trong `Die.txt` (store.go:105, app.go:2125).
10. **File result chỉ ghi 1 lần** qua OnAccountDone→saveVerifyOutcome (bỏ SaveAccountToFolder cũ, scheduler.go:866).
11. **EmailMeta base64-encode** trong file reg (cột `MM:`) — chống ký tự `|` phá format; parser legacy skip cột này (format.go:62).
12. **Counter file rỗng không được tạo** — `DumpSorted` trả `""` (counter.go:104), `Flush` skip (counters.go:102).
13. **Stop là soft** — workers chạy hết account hiện tại; muốn abort ngay phải cancel workerCtx (chỉ khi quit, không expose qua StopVerify).

---

## 10. Liên kết tài liệu

- Luồng reg/verify chính, luật token/login iOS (§13.6/§13.7), runbook thêm version: [add-facebook-reg-version.md](./add-facebook-reg-version.md)
- Scheduler / worker pool: [internal/runner/scheduler.go](../../internal/runner/scheduler.go)
- App orchestration: [app.go](../../app.go)
