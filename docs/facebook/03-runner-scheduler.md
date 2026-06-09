# 03 — Runner / Scheduler / Concurrency

> Tài liệu mô tả **chi tiết** lõi điều phối chạy song song của HVR: worker pool FIFO,
> luồng xử lý 1 account (`runOneAccount`), cơ chế retry 2 tầng, sticky proxy theo slot,
> context cancel (Stop), và **TRUE SPLIT mode** (REG pool ⟶ channel ⟶ VER pool độc lập).
>
> File code chính:
> - [internal/runner/scheduler.go](../../internal/runner/scheduler.go) — `RunVerify`, `runOneAccount`, `RunOneAccountAt`, helper retry.
> - [app.go](../../app.go) — `RunVerify` (Wails-bound), `RunRegister` (split + non-split), `StopVerify`, `StopRegister`, build `RunConfig` + callbacks.
> - [internal/proxy/sticky.go](../../internal/proxy/sticky.go) — `StickyManager` (KeepIPSuccess).
> - [internal/facebook/factory.go](../../internal/facebook/factory.go) — `NewVerifier` (plugin registry per platform).
>
> Runbook thêm version mới + luật token/login iOS: xem [add-facebook-reg-version.md](./add-facebook-reg-version.md) §13.6/§13.7. Tài liệu này **không lặp lại** runbook đó — tập trung mô tả **flow chạy thực tế**.

---

## 0. Sơ đồ tổng — 2 lớp điều phối

Có **2 lớp** điều phối, nằm ở 2 file khác nhau:

1. **Lớp scheduler (`internal/runner`)** — generic, không biết gì về Wails. Chỉ làm: nhận `[]AccountInput`, chạy song song N goroutine, mỗi goroutine gọi `runOneAccount` (login → token switch → verify → callback). Dùng cho **Verify standalone**.
2. **Lớp orchestration (`app.go`)** — biết Wails, emit event UI, đọc settings realtime, build `RunConfig` + callbacks, quản lý proxy/email pool, và **quyết định mode dispatch** (CloneHV pool / selected-file / folder streaming / register split / register non-split).

```text
                    ┌─────────────────────────────────────────────────────────┐
                    │                       app.go (Wails)                      │
                    │  RunVerify(cfgOverride)            RunRegister(maxThreads) │
                    │     │ build RunConfig + callbacks       │                 │
                    │     │ (OnAccountDone/OnEmailCreated/...) │                 │
                    └─────┼────────────────────────────────────┼────────────────┘
                          │                                     │
       ┌──────────────────┼──────────────────┐         ┌────────┴─────────┐
       ▼                  ▼                  ▼          ▼                  ▼
  CloneHV pool      selected-file      folder stream  NON-SPLIT        TRUE SPLIT
  (RunOneAccountAt   (RunVerify)       (RunOneAccountAt) reg worker      REG pool ──► splitVerifyCh ──► VER pool
   loop per slot)                                       = REG+VER inline  (RunOneAccountAt)   (RunOneAccountAt)
       │                  │                  │              │                  │
       └──────────────────┴──────────────────┴──────────────┴──────────────────┘
                          ▼  (mọi nhánh cuối cùng gọi vào)
                ┌───────────────────────────────────────────────┐
                │ runner.runOneAccount(ctx, acc, cfg, slot, cb)  │  ◄── scheduler.go:410
                │  1. GetVerifyConfig (realtime)                 │
                │  2. Acquire proxy (sticky per workerID)        │
                │  3. Build facebook.Session                     │
                │  4. Resolve verifyPlatform (round-robin)       │
                │  5. UA factory / BuildUA theo country          │
                │  6. OUTER loop ──► INNER loop:                 │
                │       switch verifyPlatform → token/login      │
                │       ver := NewVerifier(platform)             │
                │       verifyResult := ver.Verify(...)          │
                │  7. CheckLiveDie fallback (UID picture)        │
                │  8. done: save datr + OnAccountDone callback   │
                └───────────────────────────────────────────────┘
```

---

## 1. Cấu trúc dữ liệu: `RunConfig`, `AccountInput`, `AccountResult`

### 1.1 `AccountInput` — dữ liệu 1 account cần xử lý
[scheduler.go:22](../../internal/runner/scheduler.go) — struct gồm credential cơ bản (`UID/Cookie/Token/UserAgent/Proxy/Password`) + 3 nhóm field "carry-over từ reg sang verify":

- `DeviceID` / `FamilyDeviceID` (scheduler.go:33-34): tái dùng device khi verify cho account vừa reg.
- `Srnonce` / `SessionlessCryptedUID` (scheduler.go:37-38): token partial-reg của **iOS** → truyền sang verify confirm (ios562).
- `Email` / `EmailMeta` (scheduler.go:45-46): **TempMail reuse** — nếu account đã reg bằng mode TempMail thì email + creds provider đã có sẵn → verify *skip* `CreateEmail`+`AddEmail`, chỉ đọc OTP từ inbox cũ. Rỗng → verify tạo email mới.

### 1.2 `RunConfig` — cấu hình runtime + callback
[scheduler.go:50](../../internal/runner/scheduler.go). Các field quan trọng cho concurrency:

| Field | Vai trò | Dòng |
|---|---|---|
| `MaxThreads` | Số goroutine song song (clamp 1–500) | scheduler.go:51, 151-157 |
| `DelayMs` | Nghỉ giữa các lần dispatch trong 1 worker | scheduler.go:52, 169 |
| `DelayAfterResultMs` | Giữ status cuối trên UI N ms trước khi fetch account mới (cho user đọc kết quả) | scheduler.go:56, 210-215 |
| `WorkerCtx` | **Context riêng cho HTTP request của worker** — nếu set, worker dùng context này (KHÔNG bị cancel khi Stop) → chạy hết các bước hiện tại. Tách biệt với `ctx` dispatch | scheduler.go:65, 161-164 |
| `AcquireProxy(ctx, workerID)` | Trả `(proxyStr, release(success))` — sticky proxy theo **slot** | scheduler.go:75, 451-457 |
| `RequireProxy` | true → không có proxy thì **abort account** (không chạy bằng IP máy) | scheduler.go:77, 459-464 |
| `GetVerifyConfig() *VerifyConfig` | **Callback realtime** — mỗi goroutine gọi để lấy config mới nhất (đổi mail provider giữa batch có hiệu lực ngay) | scheduler.go:60, 412-413 |
| `GetVerifyPlatform() string` | **Round-robin platform** — chia version theo từng account (multi-version stats) | scheduler.go:70, 517-521 |
| `AddMailRetry` | Số outer attempt (mỗi attempt reload config → đổi email/provider) | scheduler.go:95, 432-435 |
| `RetryUnknownNow` | Sau pass 1 xong, tự verify lại mọi acc Unknown 1 pass nữa (slot mới) | scheduler.go:99, 244-301 |
| `IsRetry` | true → kết quả Unknown lưu thành Die (chống loop vô tận); cũng dùng để chặn recursion pass 3 | scheduler.go:91, 264-267 |
| `GetUseOriginalUA` / `GetBuildUA` / `GetAddVirtualSpec` | Realtime UA policy | scheduler.go:102-108, 509-511 |

**Callback realtime (biết `accountID`, do app.go inject):**

| Callback | Khi nào fire | Dòng |
|---|---|---|
| `OnRawProxy(id, proxy)` | Ngay khi proxy gán cho account (trước CheckIP) → cột PROXY | scheduler.go:80, 466-468 |
| `OnProxy(id, proxy)` | Sau CheckIP resolve (background goroutine) → cột IP CHẠY | scheduler.go:82, 471-480 |
| `OnEmailCreated(id, email)` | Ngay khi temp/rent mail tạo xong (trước verify done) → cột EMAIL | scheduler.go:85, 417-421 |
| `OnAccountDone(id, uid, status, msg, email, ua, twoFA, token, cookie, platform)` | Ngay khi account xong (trước `RunVerify` return) → ghi file + emit UI | scheduler.go:88, 871-873 |

> ⚠️ **Gotcha — inject `OnEmailCreated` vào `VerifyConfig`:** Callback runner-level biết `accountID`, còn verify-level (`VerifyConfig.OnEmailCreated func(email string)`) **không** biết. `runOneAccount` bắc cầu bằng closure capture `acc.ID` ([scheduler.go:417-421](../../internal/runner/scheduler.go)). **Mỗi lần** reload config (`GetVerifyConfig` trả pointer fresh) đều phải **re-inject** → có 3 chỗ inject: đầu hàm (417), đầu outer retry (565-569), mỗi attempt (753-757). Quên 1 chỗ → mất event email khi reload config.

### 1.3 `AccountResult` — kết quả
[scheduler.go:112](../../internal/runner/scheduler.go). `Status` ∈ `Live`/`Die`/`unknown`/`error`. `Token`/`Cookie` được **back-fill** sau verify (vd iOS login đổi `session.Token` → EAAAAAY, hoặc Android `/auth/login` trả cookie mới) → propagate lên UI/file.

---

## 2. `RunVerify` — FIFO worker pool dispatch

[scheduler.go:150](../../internal/runner/scheduler.go). Mục tiêu: chạy `[]AccountInput` song song theo kiểu **work-queue + N worker**, kết quả trả về cùng index input.

### Các bước

1. **Clamp `maxThreads`** về [1, 500] (scheduler.go:151-157).
2. **Tách 2 context** (scheduler.go:161-164):
   - `ctx` (truyền vào) = điều khiển **dispatch** (Stop cancel → ngừng đẩy account mới).
   - `workerCtx` = `config.WorkerCtx` nếu set, ngược lại = `ctx`. Đây là context dùng cho **HTTP request bên trong worker**.
3. **Đổ toàn bộ account vào buffered channel** rồi `close` (scheduler.go:176-180):
   ```go
   workCh := make(chan workItem, len(accounts))
   for i, acc := range accounts { workCh <- workItem{idx: i, account: acc} }
   close(workCh)
   ```
   Channel đã đóng + buffer đầy → N worker `range workCh` tự rút việc; khi cạn, `range` thoát. Đây là cốt lõi **FIFO work-stealing** (Go runtime tự phân phối, không cần semaphore riêng).
4. **Spawn `maxThreads` worker** (scheduler.go:186-236). Mỗi worker:
   - `workerID := w` — **slot ID ổn định** (0..maxThreads-1). Sticky proxy pin theo slot này, **không** theo goroutine ID.
   - `defer recover()` — 1 worker panic không giết cả pool (scheduler.go:191-195).
   - Vòng `for work := range workCh`:
     - Nếu `ctx.Err() != nil` (đã Stop) → ghi `Message: "Đã dừng"`, **continue** (drain channel cho hết, không xử lý) (scheduler.go:198-207).
     - Delay hiển thị kết quả (`DelayAfterResultMs`) + delay dispatch (`delayBetween`), cả 2 đều `select`-able trên `ctx.Done()` để Stop responsive (scheduler.go:210-221).
     - `first` flag: account đầu của mỗi worker **bỏ qua** mọi delay/notify "luồng tiếp theo" (scheduler.go:196, 210, 216, 223).
     - Gọi `runOneAccount(workerCtx, …, workerID, …)` — **dùng `workerCtx`** để worker chạy hết các bước dù dispatch đã cancel (scheduler.go:229).
     - Ghi `results[work.idx]` dưới `mu` (scheduler.go:231-233).
5. **`wg.Wait()`** chờ mọi worker xong (scheduler.go:238).
6. **Pass 2 — RetryUnknownNow** (scheduler.go:244-301): xem §6.
7. Return `results`.

```text
RunVerify dispatch (FIFO worker pool)
  workCh ── [acc0, acc1, acc2, ... accN]  (buffered, đã close)
              │      │      │
   ┌──────────┼──────┼──────┼──────────┐
   ▼          ▼      ▼      ▼          ▼
 worker0   worker1 worker2 ...     worker(maxThreads-1)
 slot=0    slot=1  slot=2          (workerID ổn định)
   │ range workCh tự rút việc khi rảnh (work-stealing)
   ▼
 runOneAccount(workerCtx, acc, cfg, slot, onStatus)
```

> **Vì sao tách `ctx` (dispatch) và `workerCtx` (HTTP)?** Khi user bấm Stop, ta muốn **ngừng nhận account mới** nhưng **để account đang chạy dở chạy hết** (tránh mất kết quả + tally lệch). Stop chỉ cancel dispatch `ctx`; `workerCtx` parent là `a.ctx` (app lifetime) nên vẫn sống. Xem §7.

---

## 3. `runOneAccount` — lõi xử lý 1 account

[scheduler.go:410](../../internal/runner/scheduler.go). Đây là hàm **duy nhất** mọi nhánh dispatch (verify standalone, CloneHV, split, non-split) đều hội tụ về. Mapping C#: `ExcuteOneThread()` + `ExecuteAction()`.

### Giai đoạn A — Setup (trước retry loop)

1. **Reload config realtime** (scheduler.go:412-413): `config.VerifyConfig = config.GetVerifyConfig()`.
2. **Inject `OnEmailCreated`** + emit diagnostic `MailProvider` (scheduler.go:417-426).
3. **Tính `maxAttempts=2` (inner) và `maxOuterAttempts`** (scheduler.go:429-435): outer = `max(AddMailRetry, 2)`.
4. **Acquire proxy** (scheduler.go:450-457):
   ```go
   proxyStr := acc.Proxy // ưu tiên proxy riêng của account
   if proxyStr == "" && config.AcquireProxy != nil {
       proxyStr, releaseProxy = config.AcquireProxy(ctx, workerID)
       defer func() { releaseProxy(result.Status == "Live") }() // Live → giữ IP cho acc kế
   }
   ```
   `release(success)` đọc `result.Status` **cuối cùng** (defer) → chỉ giữ proxy khi `Live` (KeepIPSuccess).
5. **RequireProxy guard** (scheduler.go:459-464): proxy bắt buộc nhưng rỗng → `Status="error"`, return ngay (không chạy bằng IP máy).
6. **CheckIP background** (scheduler.go:471-480): goroutine riêng resolve IP thật → `OnProxy`. **Không block** login; fail thì không emit gì (cột IP trống).
7. **Build `facebook.Session`** (pointer — login sẽ mutate token) (scheduler.go:483-501).
8. **Resolve `verifyPlatform` MỘT LẦN** cho cả account (scheduler.go:516-525):
   ```go
   verifyPlatform := config.VerifyPlatform
   if config.GetVerifyPlatform != nil { if p := config.GetVerifyPlatform(); p != "" { verifyPlatform = p } }
   if verifyPlatform == "" { verifyPlatform = facebook.PlatformWeb }
   result.VerifyPlatform = verifyPlatform
   ```
   > **Vì sao resolve 1 lần?** `GetVerifyPlatform` round-robin (mỗi gọi tăng counter) — nếu gọi nhiều lần trong 1 account sẽ lệch giữa quyết định "login-skip" và lần verify thật. Khoá 1 giá trị ổn định suốt account.
9. **UA selection** (scheduler.go:527-551), chỉ khi `!useOrigUA`:
   - Ưu tiên `facebook.PlatformVerifyUA(platform, country)` (platform versioned/iOS có factory UA riêng).
   - Fallback `BuildUA` động (`fakeinfo.BuildAndroidUAWithOpts`) cho platform generic. Country lấy từ suffix proxy `/vn`, `/us`… qua `extractCountryFromProxy` (scheduler.go:1040).

### Giai đoạn B — OUTER retry loop
[scheduler.go:554-828](../../internal/runner/scheduler.go). `for outer := 0; outer < maxOuterAttempts; outer++`:
- `outer > 0`: notify "Thử lại với email mới", **reload `GetVerifyConfig` + re-inject OnEmailCreated**, reset `session.FbDtsg/Jazoest/Lsd` (scheduler.go:555-575). Reset `result.Status/Message/Email` (scheduler.go:578-580).

### Giai đoạn C — INNER retry loop (token/login switch + verify)
[scheduler.go:582-822](../../internal/runner/scheduler.go). `for attempt := 1; attempt <= maxAttempts; attempt++`. **Đây là phần token-type per platform** (§4).

---

## 4. Token-type per platform — `switch verifyPlatform` (lõi)

[scheduler.go:606-744](../../internal/runner/scheduler.go). Quyết định **cần login/fetch token gì** trước khi gọi verifier. 4 nhóm + safety net:

### 4.1 Android-family → access token EAAAAU (skip cookie login)
[scheduler.go:607-687](../../internal/runner/scheduler.go). Một danh sách **rất dài** các `PlatformS22…S564…`, `PlatformAndroid`, `PlatformS399`, `PlatformS273` (đồng bộ với `isAndroidVersionedPlatform` scheduler.go:905).

Logic (scheduler.go:653-687):
1. **Loại token iOS lạc loài**: nếu token bắt đầu `EAAAAAY` (iOS) → **clear** (`session.Token = ""`) vì token iOS không hợp lệ cho endpoint Android b-graph (scheduler.go:653-656).
2. **`needFetch`** = `!isValidAndroidToken(token) && UID != "" && Password != ""` (scheduler.go:657). `isValidAndroidToken` = prefix `EAAAAU` hoặc `EAAAAAY` (scheduler.go:901-903).
3. Nếu cần → **REST `/auth/login`** lấy EAA (scheduler.go:658-671):
   ```go
   tokCtx, tokCancel := context.WithTimeout(ctx, 30*time.Second)
   tok, newCookie := web.FetchAndroidTokenLegacyWithCookie(tokCtx, session.UID, session.Password, session.Datr, "en_US", "", session.Proxy, "", notify)
   tokCancel()
   if strings.HasPrefix(tok, "EAAAAU") {
       session.Token = tok; result.Token = tok            // propagate
       if newCookie != "" { session.Cookie = newCookie; result.Cookie = newCookie }
   }
   ```
   > **Port S399**: dùng REST classic API stable (không Bloks/GraphQL schema rotation). UA truyền `""` → auto FB4A UA (KHÔNG dùng `session.UserAgent` vì có thể là Chrome WebAndroid).
4. Nếu token đã hợp lệ sẵn → chỉ propagate `result.Token` (scheduler.go:672-676).
5. **Guard cuối**: nếu vẫn không có token hợp lệ → `Status="unknown"`, `goto done` (scheduler.go:680-687).

### 4.2 iOS (`PlatformIOS562`/`PlatformIOS563`) → **bỏ qua login pre-fetch**
[scheduler.go:688-693](../../internal/runner/scheduler.go). Scheduler **KHÔNG** pre-login ở đây — chỉ notify rồi để verifybase tự login iOS lấy `EAAAAAY` (qua `spec.FetchToken` trong `ios562/steps.go`) nếu account chưa có. Đây là điểm khác biệt cốt lõi so với Android.
> Chi tiết luật token/login iOS: [add-facebook-reg-version.md §13.7](./add-facebook-reg-version.md).

### 4.3 WebAndroid (`PlatformWebAndroid`) → cookie trực tiếp (skip login)
[scheduler.go:694-702](../../internal/runner/scheduler.go). Có cookie → notify skip; không cookie → `Status="unknown"`, `goto done`.

### 4.4 Web MFB (`PlatformWeb`) → **bắt buộc login cookie** m.facebook.com
[scheduler.go:703-732](../../internal/runner/scheduler.go). Chỉ platform này gọi `facebook.LoginWithCookieMobile(ctx, session)` để parse `fb_dtsg/jazoest/lsd/datr`:
```go
loginResult, err := facebook.LoginWithCookieMobile(ctx, session)
if err != nil || loginResult == nil || !loginResult.Success {
    if isCookieDead(msg) { goto done }            // cookie chết → không retry
    if isNetworkError(msg) { Status="unknown"; goto done } // retry cùng proxy vô ích
    if attempt == maxAttempts { notify(...) }
    continue                                        // inner retry login lại
}
```

### 4.5 SAFETY NET — `default`
[scheduler.go:733-743](../../internal/runner/scheduler.go). **Cực quan trọng.** Platform chưa khai báo trong case nào → emit `[FATAL]` + `Status="error"` + `goto done`.
> **Bug recurring đã phòng**: trước đây thêm platform Android mới mà **quên** add vào list Android-family → silent fall xuống cookie login → "đăng nhập bằng cookie" sai logic (Android verify bằng cookie luôn fail). Giờ explicit error để dev nhận ra ngay. **Khi thêm version mới, BẮT BUỘC add vào cả case switch (scheduler.go:607) lẫn `isAndroidVersionedPlatform` (scheduler.go:905).**

### 4.6 Sau switch — gọi verifier
[scheduler.go:749-785](../../internal/runner/scheduler.go):
1. Reload config mỗi attempt + re-inject OnEmailCreated (scheduler.go:749-758).
2. `config.VerifyConfig.UserApiLabel = verifyPlatform` — log đúng version round-robin (scheduler.go:763-765).
3. `ver, _ := facebook.NewVerifier(verifyPlatform)` ([factory.go:263](../../internal/facebook/factory.go)) — plugin registry lookup.
4. `verifyResult := ver.Verify(ctx, session, config.VerifyConfig, dateFolder, onStatus-bridge)` (scheduler.go:767).
5. **Back-fill** `result.Token`/`result.Cookie` từ `session` (verifier có thể đã mutate, vd iOS FetchToken set token EAAAAAY + cookie) (scheduler.go:779-785).

---

## 5. Phân loại kết quả + dừng/retry (sau verify trong inner loop)

[scheduler.go:787-821](../../internal/runner/scheduler.go). Thứ tự kiểm tra (ưu tiên trên xuống):

| Điều kiện | Hành động | Helper | Dòng |
|---|---|---|---|
| `Status == Live`/`Die` | `goto done` (terminal, không retry) | — | 788-790 |
| `isTokenDead(msg)` | `Status="Die"`, `goto done` | scheduler.go:356 (401 + malformed token) | 792-796 |
| `isBloksCheckpoint(msg)` | `Status="unknown"`, `goto done` (cần thao tác tay) | scheduler.go:347 | 798-802 |
| `isNetworkError(msg)` | `Status="unknown"`, `goto done` (retry cùng proxy vô ích) | scheduler.go:366 | 805-809 |
| `isOTPError(msg)` | `break` inner → outer loop cấp email mới | scheduler.go:318 | 812-814 |
| `ctx.Err()` | `"Đã dừng"`, `goto done` | — | 815-818 |
| `attempt == maxAttempts` | notify thất bại, hết inner loop | — | 819-821 |

> **Vì sao OTP error `break` chứ không `goto done`?** Email OTP đã gửi + hết hạn phía FB; retry login chỉ tạo OTP rác. Thoát inner → **outer loop** cấp **email mới** (AddMailRetry). Còn cookie chết / token chết / network error thì retry hoàn toàn vô ích → `goto done` luôn.

### 5.1 CheckLiveDie fallback (post-loop)
[scheduler.go:835-849](../../internal/runner/scheduler.go). Nếu sau **tất cả** outer attempt vẫn `!= Live/Die` và có `UID`:
```go
liveStatus := verifybase.CheckLiveDieByPicture(checkCtx, session.UserAgent, result.UID) // 12s timeout
// "Die" → Status=Die (UID chết, không ghi Unknown.txt vô ích)
// "Live" → giữ unknown (hết lượt retry); network lỗi → giữ unknown
```
> Chỉ chạy khi loop **thoát bình thường** (không phải `goto done` từ early-exit như cookie chết).

### 5.2 `done:` — finalize
[scheduler.go:851-874](../../internal/runner/scheduler.go):
1. **Save datr** vào `Config/Cookie/datr_pool.txt` (async goroutine) cho mọi outcome có cookie chứa datr (scheduler.go:856-864). Match C# `SaveDatrFromCookieIfNew`.
2. File kết quả Live/Die/Unknown **KHÔNG** ghi ở đây — handle qua `OnAccountDone` → `saveVerifyOutcome` (app.go) với `UpsertUID` dedupe (scheduler.go:866-868).
3. **`OnAccountDone(...)`** emit realtime — trước khi return (scheduler.go:871-873).

---

## 6. Pass 2 — RetryUnknownNow (trong `RunVerify`)

[scheduler.go:244-301](../../internal/runner/scheduler.go). Chỉ chạy khi `RetryUnknownNow && !IsRetry && ctx.Err()==nil`:

1. **Gom acc Unknown**: `results[i].Status` ∈ {`""`, `unknown`, `error`} + `UID != ""` (Live/Die là terminal, không retry) (scheduler.go:246-253).
2. Đẩy vào `retryCh` mới, close.
3. **`retryCfg`** = copy config với `IsRetry=true` (chống loop), `RetryUnknownNow=false` (chống recursion pass 3) (scheduler.go:265-267).
4. **`workerIDOffset = maxThreads`** (scheduler.go:273): retry worker dùng slot ID **mới** (`w + maxThreads`) → sticky manager **force acquire proxy fresh** (không reuse proxy đã fail pass 1). Double-insurance: pass 1 đã `release(false)` mọi acc non-Live, nhưng pool nhỏ vẫn có thể bắt trúng → offset slot đảm bảo fresh.
5. Spawn `maxThreads` retry worker mới, `rwg.Wait()`.

---

## 7. Concurrency & Context cancel (Stop)

### 7.1 Hai context, vai trò khác nhau
Tại [app.go:2501-2521](../../app.go) (verify):
```go
poolCtx, poolCancel := context.WithCancel(a.ctx)     // dispatch ctx
ctx := poolCtx
workerCtx, workerCancel := context.WithCancel(a.ctx) // worker HTTP ctx
a.verifyCancel = poolCancel; a.verifyWorkerCancel = workerCancel // gán trong verifyMu
```
- `poolCtx` → `RunVerify(ctx, …)` (điều khiển dispatch loop).
- `workerCtx` → `RunConfig.WorkerCtx` (điều khiển HTTP request worker).

### 7.2 `StopVerify` — graceful
[app.go:3861-3878](../../app.go):
```go
a.verifyStopping.Store(true)
if a.verifyCancel != nil { a.verifyCancel() } // CHỈ cancel dispatch — workers KHÔNG bị abort
```
> Stop verify chỉ cancel **dispatch** (`poolCancel`). Worker đang chạy verify tiếp tục đến khi xong (vì `workerCtx` chưa cancel) → tránh mất kết quả. `workerCancel()` chỉ được gọi ở **defer của goroutine dispatch** sau khi `wg.Wait()` (vd folder mode app.go:3678-3679). State machine: `isRunning` + `verifyStopping` (atomic) chặn Start mới khi đang stopping (app.go:2295-2298).

### 7.3 Sticky proxy theo workerID (KeepIPSuccess)
[internal/proxy/sticky.go](../../internal/proxy/sticky.go). `StickyManager.Acquire(ctx, workerID)`:
- Nếu `enabled` + đã có entry cho `workerID` → **dùng lại proxy cũ** (sticky.go:52-58).
- Ngược lại acquire fresh, lưu `entries[workerID]` (sticky.go:60-67).
- `release(success)`: `enabled && success` → **giữ pin** (no-op); ngược lại xóa entry + raw release (sticky.go:73-91).
- `ReleaseAll()` cuối batch giải phóng toàn bộ (sticky.go:94-105).

App.go build wrapper qua `getSharedProxyManager()` (chung pool REG+VER, tránh double API call) tại [app.go:2480-2492](../../app.go); `RunConfig.AcquireProxy = verifySticky.Acquire`, defer `verifySticky.ReleaseAll()`.

> **Vì sao pin theo `workerID` (slot) chứ không goroutine?** Slot ID ổn định suốt batch (0..maxThreads-1); goroutine có thể đổi. Pin theo slot → mỗi luồng UI có 1 proxy session ổn định, account kế trên cùng slot kế thừa IP nếu account trước Live. Đây cũng là lý do `RunOneAccount(workerID=0)` bị đánh dấu **DEPRECATED** (scheduler.go:884-886): mọi worker share 1 entry → defeat session rotation. **Luôn dùng `RunOneAccountAt(workerID)`** với slot ID stable.

### 7.4 Realtime config/platform reload
- `GetVerifyConfig` (app.go:2826-2904): mỗi goroutine `LoadInteractionConfig()` fresh → đổi mail provider/delay giữa batch có hiệu lực ngay, không cần restart.
- `GetVerifyPlatform` → `a.nextVerifyPlatform()` (app.go:7349-7359): atomic counter round-robin trên `verifyPlatformKeyList` → chia version đều cho từng account (multi-version stats qua `recordVerifyOutcome` app.go:7134).

---

## 8. 5 nhánh dispatch trong `app.go`

`a.RunVerify(cfgOverride)` ([app.go:2280](../../app.go)) sau khi validate (account source/mail/proxy/output) + build `runCfg` ([app.go:2794](../../app.go)) sẽ rẽ thành 3 nhánh verify; `a.RunRegister` ([app.go:7364](../../app.go)) có 2 nhánh.

### Nhánh 1 — CloneHV pool (verify)
[app.go:~3270-3546](../../app.go). Port C# `ConcurrentQueue + SemaphoreSlim`: N worker loop, mỗi worker `fetchOneAccount` (dequeue buffer / non-blocking bulk-buy 50) → `addAndBuildInput` (ghi session file, update slot row) → `RunOneAccountAt(workerCtx, inp, runCfg, dateFolder, slotID, onStatus)` (app.go:3519). Slot row cố định N dòng, update in-place. Vòng lặp đến khi `ctx.Done()`.

### Nhánh 2 — selected-file (verify)
[app.go:3553-3619](../../app.go). User tick account trong grid → build `targets []AccountInput` từ `a.accounts` filter theo `AccountIDs` → `runner.RunVerify(ctx, targets, runCfg, onStatus)` (app.go:3615). Đây là nhánh **duy nhất** gọi thẳng `RunVerify` (worker pool §2).

### Nhánh 3 — folder streaming (verify)
[app.go:3621-3789+](../../app.go). Tạo N slot row "waiting". `freeSlots chan int` (slot rảnh), `replenishCh` (signal đọc bổ sung). `readAndStart(count)`: pop account từ folder (`popAccountFromFolder`), lấy `slotID := <-freeSlots`, `startWorker(inp, slotID)`. Mỗi worker `RunOneAccountAt(workerCtx, …, slotID, …)` (app.go:3703); defer trả `freeSlots <- slotID` + `replenishCh <- struct{}{}` (đọc bổ sung 1 account). Có guard `maxConsecutiveInvalid=100` chống spin CPU khi file sai format (app.go:3717-3744).

### Nhánh 4 — RunRegister NON-SPLIT (reg + verify inline)
[app.go:9699-10070+](../../app.go). Reg worker reg xong → **verify inline ngay trong cùng worker**: build `acc` từ `result` (UID/Token/Cookie/DeviceID/Email…) → đổi UA verify-platform → delay `DelayVeriReg` (ctx run-scoped) → `RunOneAccountAt(regWorkerCtx, acc, runCfg, …, threadIdx, verifyOnStatus)` (app.go:10018). Reg slot **bị block** đến khi verify xong → reg không full tốc độ.
- `verifySem` (app.go:8106-8109, 10008-10014): nếu `SplitMode && SplitVerifyThreads>0` thì semaphore giới hạn verify đồng thời ≤ `SplitVerifyThreads` (REG chạy maxThreads nhưng verify inline phải acquire permit).

### Nhánh 5 — RunRegister TRUE SPLIT (REG pool ⟶ channel ⟶ VER pool)
[app.go:8111-8163, 8499-8533, 8645-8648, 9666-9698](../../app.go). **Khác biệt cốt lõi**: reg worker reg xong **KHÔNG verify inline** mà đẩy job vào `splitVerifyCh` rồi **return ngay** (giải phóng reg slot → reg full tốc độ). Một **pool VER riêng** verify async.

```text
TRUE SPLIT MODE  (splitModeActive = SplitMode && VerifyEnabled, app.go:8133)

 REG pool (maxThreads)                       VER pool (splitVerThreads)
 ┌───────────────────┐                       ┌────────────────────────────┐
 │ reg worker #i      │   splitVerifyJob      │ VER worker (×splitVerThreads)│
 │  reg success ──────┼──► splitVerifyCh ─────┼─► verSlot := <-freeSlotsVer  │
 │  push job          │   (buffered           │   runSplitVerify(verSlot,job)│
 │  RETURN (free slot)│    maxThreads*5,≥500) │     RunOneAccountAt(          │
 │  → reg tiếp full   │                       │       splitWorkerCtx, acc,   │
 └───────────────────┘                       │       runCfg, verSlot, …)    │
        │                                     │   freeSlotsVer <- verSlot    │
        │ spawner defer:                      └────────────────────────────┘
        │  wg.Wait()  (reg workers xong)
        │  close(splitVerifyCh) ──────────────► VER range thoát sau khi DRAIN hết queue
        │  splitVerWg.Wait()  (VER xong)
        │  splitWorkerCancel()  (giải phóng splitWorkerCtx)
```

**Chi tiết:**
1. **Setup** (app.go:8149-8163): `splitVerifyCh = make(chan, max(maxThreads*5, 500))` (buffer lớn → REG chạy vượt VER không bị block = true split). `freeSlotsVer = make(chan int, splitVerThreads)` nạp `1..splitVerThreads`. `splitWorkerCtx` parent `a.ctx` (KHÔNG bị Stop register cancel).
2. **Reg worker push job** (app.go:9666-9698): build `splitVerifyJob{acc, prof, displayProxy, regResult}`, đẩy `select { case splitVerifyCh <- job: case <-ctx.Done(): }` (ctx-aware để Stop không treo; channel đầy → backpressure tự nhiên).
3. **VER worker pool** (app.go:8499-8533): `splitVerThreads` goroutine `for job := range splitVerifyCh`. Mỗi job: `verSlot := <-freeSlotsVer` → `runSplitVerify(verSlot, job)` (recover per-job để 1 panic không giết worker) → `freeSlotsVer <- verSlot` (recycle slot).
4. **`runSplitVerify`** (app.go:8170-8497): reload config, đổi UA verify-platform, `acc.ID = verSlot`, emit `verify:slot-assigned`/`raw-proxy`/`email` cho **VER panel** (slot ID riêng, không đụng REG panel), `RunOneAccountAt(splitWorkerCtx, acc, runCfg, …, verSlot, verifyOnStatus)` (app.go:8421), emit `verify:account-done`, ghi file qua `saveVerifyOutcome(regWriter, …)`, auto-upload nếu live.
5. **Drain + shutdown** (app.go:8639-8649): spawner defer — sau `wg.Wait()` (reg xong, không còn job mới) → `close(splitVerifyCh)` → VER `range` drain hết queue → `splitVerWg.Wait()` → `splitWorkerCancel()`. **Phải drain TRƯỚC `regCounters.Stop()`** vì `runSplitVerify` ghi qua `regCounters`.

**So sánh NON-SPLIT vs TRUE SPLIT:**

| | NON-SPLIT (inline) | TRUE SPLIT |
|---|---|---|
| Verify chạy ở đâu | Trong reg worker (cùng goroutine) | VER pool riêng (goroutine khác) |
| Reg slot có bị block bởi verify | **Có** — đợi verify xong mới free | **Không** — push channel rồi free ngay |
| Tốc độ reg | Giới hạn bởi verify | Full tốc độ |
| Slot ID verify | `threadIdx` (chung reg) | `verSlot` riêng (VER panel) |
| Context worker | `regWorkerCtx` | `splitWorkerCtx` (drain-scoped) |
| Giới hạn verify đồng thời | `verifySem` (≤ SplitVerifyThreads) | Số VER worker = `splitVerThreads` |

> **Lưu ý nhầm lẫn `regToVerCh`** (app.go:8621-8625): biến **DEPRECATED**, luôn `nil`. Comment cũ "Split Mode = PURE UI option" (app.go:8622, 9659) **mâu thuẫn** với code TRUE SPLIT đang active (app.go:8133, 9667) — comment lỗi thời; luồng thực tế là **true split qua `splitVerifyCh`**. Tin code, không tin comment "PURE UI".

---

## 9. Edge case & Gotcha (tổng hợp)

1. **Quên add platform vào switch** → SAFETY NET default → `[FATAL]` + `error` (§4.5). Phải sync 2 nơi: `switch` (scheduler.go:607) + `isAndroidVersionedPlatform` (scheduler.go:905).
2. **Token iOS lạc trong Android verify**: `EAAAAAY` bị clear để login lấy `EAAAAU` đúng loại (scheduler.go:653-656). Đối xứng: iOS verify chỉ nhận `EAAAAAY`, gặp `EAAAAU` thì login iOS lại.
3. **`RunOneAccount(workerID=0)` DEPRECATED** — share 1 sticky entry → mọi worker cùng proxy. Dùng `RunOneAccountAt(workerID)` (scheduler.go:884-896).
4. **Re-inject `OnEmailCreated`** ở cả 3 chỗ reload config, nếu không sẽ mất event email (scheduler.go:417/565/753).
5. **Verify dùng `workerCtx` (HTTP) ≠ dispatch `ctx`** — Stop verify chỉ ngừng dispatch, account đang chạy chạy hết (app.go:3875).
6. **TRUE SPLIT dùng `splitWorkerCtx` parent `a.ctx`** — Stop register KHÔNG cancel verify đang chạy; chỉ ngừng nhận reg mới, VER drain hết queue rồi cancel ở defer (app.go:8645-8648).
7. **Buffer `splitVerifyCh` lớn** (≥500) để REG không bị VER block; đầy thì backpressure (hiếm) (app.go:8150-8157).
8. **`maxConsecutiveInvalid=100`** chống spin CPU 100% khi folder toàn dòng sai format (app.go:3717).
9. **RetryUnknownNow offset slot = +maxThreads** → fresh proxy pass 2 (scheduler.go:273).
10. **`OnAccountDone` chịu trách nhiệm ghi file + emit UI** — `runOneAccount` KHÔNG tự ghi (bỏ `SaveAccountToFolder` cũ để tránh ghi 2 lần) (scheduler.go:866-868).

---

## 10. Liên kết tài liệu

- Thêm version reg/verify mới + luật token/login iOS: [add-facebook-reg-version.md](./add-facebook-reg-version.md) (§13.6 token, §13.7 login iOS).
- Verifier per platform (`ver.Verify`): [internal/facebook/factory.go](../../internal/facebook/factory.go) `NewVerifier`.
- Sticky proxy: [internal/proxy/sticky.go](../../internal/proxy/sticky.go).
- Mapping verify API key → platform const: [app.go:589](../../app.go) `verifyPlatformFromType`.
