# 02 — Luồng VERIFY (xác minh tài khoản Facebook) — End-to-End

> Tài liệu mô tả **CHI TIẾT TỪNG BƯỚC** của luồng VERIFY trong HVR: từ lúc user bấm "Chạy" trên UI → backend Go điều phối → từng platform verify gửi request lên Facebook → ghi kết quả ra file.
>
> Tài liệu này tập trung **FLOW CHẠY THỰC TẾ**. Phần *runbook thêm version* và *luật token/login iOS* đã có ở [add-facebook-reg-version.md](./add-facebook-reg-version.md) — đặc biệt:
> - [§12 — Token-required platform KHÔNG login bằng cookie](./add-facebook-reg-version.md#12-rule-quan-trọng--token-required-platform-không-được-login-bằng-cookie)
> - [§13.6 — iOS Native (FBIOS) verify BẮT BUỘC token EAAAAAY](./add-facebook-reg-version.md#136-ios-native-fbios-verify-ios562-ios563--bắt-buộc-token-eaaaaay)
> - [§13.7 — Login-at-verify + Token đúng loại theo platform (ĐỐI XỨNG)](./add-facebook-reg-version.md#137-login-at-verify--tokencookie-hiển-thị-realtime-cập-nhật-2026-05-31)
>
> Khi gặp các chủ đề đó, tài liệu này **link sang** thay vì lặp lại nguyên văn.

---

## Mục lục

1. [Sơ đồ tổng quan](#1-sơ-đồ-tổng-quan)
2. [Điểm vào: App.RunVerify (Wails-bound)](#2-điểm-vào-apprunverify-wails-bound)
3. [runner.RunVerify — FIFO worker pool](#3-runnerrunverify--fifo-worker-pool)
4. [runOneAccount — retry 2 tầng + token acquisition theo platform](#4-runoneaccount--retry-2-tầng--token-acquisition-theo-platform)
5. [Token acquisition theo platform (RẤT QUAN TRỌNG)](#5-token-acquisition-theo-platform-rất-quan-trọng)
6. [verifybase.RunVerify — pipeline chung 8 bước](#6-verifybaserunverify--pipeline-chung-8-bước)
7. [Per-platform verify (so sánh)](#7-per-platform-verify-so-sánh)
8. [Live/Die detection + lớp chống false-positive](#8-livedie-detection--lớp-chống-false-positive)
9. [Kết quả verify → file (SuccessVerify / Die / Unknown)](#9-kết-quả-verify--file-successverify--die--unknown)
10. [Phân loại lỗi & quy tắc retry](#10-phân-loại-lỗi--quy-tắc-retry)
11. [Edge case & gotcha](#11-edge-case--gotcha)

---

## 1. Sơ đồ tổng quan

```
┌──────────────────────── FRONTEND (Vue) ────────────────────────┐
│ User bấm "Chạy Verify" → gọi Wails binding App.RunVerify(cfg)   │
└──────────────────────────────┬──────────────────────────────────┘
                               │ (JSON cfgOverride)
                               ▼
┌──────────────────── app.go : App.RunVerify ────────────────────┐
│ • Lock isRunning, chặn trùng REG/VER (cùng email pool)          │
│ • Validate: nguồn account / mail provider / proxy / result dir  │
│ • Build facebook.VerifyConfig + runner.RunConfig                │
│ • Tạo email pool (rent) + output folder VerifyXxx_<timestamp>   │
│ • Wire callback: OnRawProxy/OnProxy/OnEmailCreated/OnAccountDone │
└──────────────────────────────┬──────────────────────────────────┘
                               │ runner.RunVerify(ctx, accounts, runCfg, onStatus)
                               ▼
┌──────────── internal/runner/scheduler.go : RunVerify ──────────┐
│ FIFO worker pool: N goroutine (semaphore) tự lấy account từ chan │
│ Mỗi worker → runOneAccount(...)                                 │
│ Sau pass 1: nếu RetryUnknownNow → pass 2 cho acc Unknown/Error   │
└──────────────────────────────┬──────────────────────────────────┘
                               │ runOneAccount(workerCtx, acc, cfg, ...)
                               ▼
┌──────────── runOneAccount : retry 2 tầng + token theo platform ─┐
│ 1. Acquire proxy (sticky theo workerID), set UA theo platform   │
│ 2. switch verifyPlatform:                                       │
│    • Android-family  → fetch EAAAAU qua /auth/login nếu thiếu    │
│    • iOS (562/563)   → KHÔNG pre-login, delegate xuống verify    │
│    • WebAndroid      → dùng cookie trực tiếp                     │
│    • Web (MFB)       → login cookie m.facebook.com (fb_dtsg)     │
│ 3. ver, _ := facebook.NewVerifier(platform)                     │
│ 4. ver.Verify(ctx, session, cfg, dateFolder, onStatus)          │
│ 5. Phân loại kết quả: Live/Die terminal; Unknown → retry/CheckUID│
└──────────────────────────────┬──────────────────────────────────┘
              ┌────────────────┼─────────────────────────────┐
              ▼                ▼                             ▼
   ┌───────────────────┐  ┌─────────────┐         ┌──────────────────┐
   │ verifybase        │  │ s273/s399   │         │ web / webandroid │
   │ .RunVerify (Bloks)│  │ (REST FB4A) │         │ (cookie m.fb.com)│
   │ s5xx/s6xx + iOS   │  │ legacy      │         │                  │
   └─────────┬─────────┘  └──────┬──────┘         └────────┬─────────┘
             │                   │                         │
             └───────────────────┴─────────────────────────┘
                               │ trả *facebook.VerifyResult{Status,Email,UserAgent,TwoFA}
                               ▼
┌──────────── app.go : OnAccountDone → saveVerifyOutcome ─────────┐
│ Live    → SuccessVerify.txt / SuccessVerify_No2FA.txt + UG file  │
│ Die     → DieAfterVerify.txt (UpsertUID)                         │
│ Unknown → UnknownErrorCheckLiveDieApi.txt                        │
│ + emit "verify:account-done" lên UI realtime                    │
└──────────────────────────────────────────────────────────────────┘
```

**Pipeline verify chung (verifybase.RunVerify) — 8 bước:**

```
Token check ──► CreateClient (TLS) ──► Inject cookie (datr/sb/fr)
     │                                          │
     ▼                                          ▼
Email service (temp/rent) ──► [reuse mail?] ──► FixUA (regen UA)
     │
     ▼
[STEP 1] AddEmail (Bloks) ──► [STEP 2] WaitOTP ─(timeout)─► Resend ──► WaitOTP
     │                                                                   │
     ▼                                                                   ▼
[STEP 3] ConfirmCode (CheckConfirmSuccess, retry x3) ──► [STEP 4] CheckLiveDie (loop 5s)
     │
     ▼
[STEP 5] 2FA (nếu bật + Live) ──► [STEP 6] PostConfirm (addinfo) ──► VerifyResult
```

---

## 2. Điểm vào: App.RunVerify (Wails-bound)

**File:** [app.go:2280](../../app.go) — `func (a *App) RunVerify(cfgOverride VerifyRunConfig) string`

**Mục tiêu:** nhận lệnh từ frontend, validate, chuẩn bị config + email pool + output folder, rồi giao cho `runner.RunVerify`.

**Input:** `VerifyRunConfig` (từ frontend qua Wails) — chứa `AccountIDs`, `MaxThreads`, `OutputPath`...

**Các bước (đánh số):**

1. **Chặn chạy đồng thời REG + VER** ([app.go:2283](../../app.go)) — cả hai dùng chung `emailPool`. Nếu `registerState` đang running/stopping → return chuỗi lỗi tiếng Việt.
2. **Lock `isRunning`** ([app.go:2290](../../app.go)) — verify chỉ chạy 1 batch tại 1 thời điểm.
3. **Đọc settings + interaction config** ([app.go:2303-2308](../../app.go)) — `LoadSettings()` + `LoadInteractionConfig()`, áp `applyVerifyPlatformUAConfig` (UA per-platform).
4. **Validation tuần tự** (mỗi lỗi → `failValidation` reset `isRunning` rồi return chuỗi):
   - Nguồn account: CloneHV credentials / file mode (tick chọn) / folder mode ([app.go:2322-2366](../../app.go)).
   - Mail provider: theo `interaction.MailProvider` — provider rent cần API key, custom cần MailList ([app.go:2369-2406](../../app.go)).
   - Result dir: validate nhẹ, fallback `defaultResultFolder()` nếu invalid ([app.go:2415-2429](../../app.go)).
   - Proxy: nếu chọn loại proxy thì phải có list/key ([app.go:2432-2453](../../app.go)).
5. **Merge maxThreads** ([app.go:2457-2464](../../app.go)) — ưu tiên `SplitVerifyThreads` > `RegThreads` > `ThreadRequest`.
6. **Sticky proxy manager** ([app.go:2480-2492](../../app.go)) — `proxy.NewStickyManager(KeepIPSuccess, ...)`: giữ proxy cho account kế nếu account trước Live.
7. **2 context riêng biệt** ([app.go:2501-2521](../../app.go)):
   - `poolCtx`/`verifyCancel` — điều khiển vòng dispatch (Stop dừng đẩy account mới).
   - `workerCtx`/`verifyWorkerCancel` — điều khiển HTTP requests trong worker (Stop cancel cả request đang chạy).
8. **Tạo email pool** ([app.go:2531-2582](../../app.go)) — chỉ cho rent provider (zeus-x/dongvanfb/store1s/mail30s). Wire `OnExhausted` (báo hết mail) + `OnBought` (lưu mail mua ra file).
9. **Build `facebook.VerifyConfig`** ([app.go:2585-2650](../../app.go)) — gồm `UserApiLabel` (tên API để log), mail provider keys, `CheckLiveDie`, `TimeDelaySendCode`, `WaitMailMs` (**UI giây × 1000 → ms**, gotcha cũ), `Enable2FA`, `AddInfo`...
10. **Tạo output folder** ([app.go:2662-2761](../../app.go)) — tên `VerifyXxx_<YYYYMMDD_HHMMSS>` theo platform (vd `VerifyS562_...`, `VerifyMfb_...`). `verifyWriter := resultpkg.NewWriter(outputPath)` + `verifyCounters`.
11. **Build `runner.RunConfig`** ([app.go:2794-2911](../../app.go)) — quan trọng:
    - `GetVerifyConfig` ([app.go:2826](../../app.go)): closure đọc lại config **realtime** mỗi attempt → user đổi mail provider giữa batch có hiệu lực ngay.
    - `GetVerifyPlatform` ([app.go:2822](../../app.go)): `a.nextVerifyPlatform()` — round-robin chia version theo từng account (multi-version stats).
    - `AcquireProxy: verifySticky.Acquire`, `RequireProxy: proxyManager.IsConfigured()`.
    - Callbacks: `OnRawProxy`, `OnProxy`, `OnEmailCreated`, `OnAccountDone`.
12. **Gọi `runner.RunVerify(ctx, accounts, runCfg, onStatus)`** → block đến hết batch.

**Output:** chuỗi rỗng `""` nếu start OK; chuỗi lỗi tiếng Việt nếu validation fail.

> **Gotcha — WaitMailMs:** field UI là **giây** (label "Wait mail (s)"), backend expect **mili-giây** → `WaitMailMs: interaction.WaitMail * 1000` ([app.go:2602](../../app.go)). Trước đây gán trực tiếp → poll mỗi 5ms thay vì 5000ms → poll 6000 lần.

---

## 3. runner.RunVerify — FIFO worker pool

**File:** [scheduler.go:150](../../internal/runner/scheduler.go) — `func RunVerify(ctx, accounts []AccountInput, config RunConfig, onStatus StatusCallback) []AccountResult`

**Mục tiêu:** chạy danh sách account song song theo FIFO worker pool, mapping từ C# WeBM `Run()` FIFO Scheduler.

**Input:** `accounts []AccountInput` (mỗi acc có UID/Cookie/Token/UserAgent/Proxy/Password/DeviceID/Srnonce/Email/EmailMeta — xem [scheduler.go:22](../../internal/runner/scheduler.go)).

**Các bước:**

1. **Clamp maxThreads** vào [1, 500] ([scheduler.go:151-157](../../internal/runner/scheduler.go)).
2. **workerCtx** ([scheduler.go:161-164](../../internal/runner/scheduler.go)): nếu `config.WorkerCtx != nil` thì worker dùng ctx này (không bị cancel khi Stop → chạy hết các bước HTTP hiện tại). `ctx` (poolCtx) chỉ dừng việc đẩy account mới.
3. **Work queue** ([scheduler.go:176-180](../../internal/runner/scheduler.go)): tất cả accounts đẩy vào `workCh` buffered channel rồi `close`.
4. **Start N workers** ([scheduler.go:186-236](../../internal/runner/scheduler.go)): mỗi goroutine có `workerID` ổn định (0..maxThreads-1) — dùng bởi sticky proxy. Mỗi worker:
   - `recover()` panic an toàn.
   - `for work := range workCh` — tự lấy account.
   - Nếu `ctx.Err() != nil` → ghi "Đã dừng" + bỏ qua.
   - Delay hiển thị: `DelayAfterResultMs` (giữ status cuối trên UI) + `delayBetween` ([scheduler.go:210-221](../../internal/runner/scheduler.go)).
   - Gọi `runOneAccount(workerCtx, work.account, config, dateFolder, workerID, onStatus)`.
   - Lưu kết quả vào `results[work.idx]` (mutex bảo vệ).
5. **wg.Wait()** — chờ tất cả worker xong.
6. **RetryUnknownNow — pass 2** ([scheduler.go:244-301](../../internal/runner/scheduler.go)): nếu user bật checkbox, re-queue các acc `status=="" / "unknown" / "error"` (Live/Die là terminal, không retry). Chạy với:
   - `retryCfg.IsRetry = true` → `runOneAccount` không retry vô hạn (Unknown giữ Unknown).
   - `workerIDOffset = maxThreads` → sticky-proxy dùng SLOT MỚI → force lấy proxy fresh khác pass 1.

**Output:** `[]AccountResult` cùng index với input.

---

## 4. runOneAccount — retry 2 tầng + token acquisition theo platform

**File:** [scheduler.go:410](../../internal/runner/scheduler.go) — `func runOneAccount(ctx, acc, config, dateFolder, workerID, onStatus) AccountResult`

**Mục tiêu:** chạy đầy đủ flow verify cho 1 account, có cơ chế retry 2 tầng. Mapping C# `ExcuteOneThread() + ExecuteAction()`.

**Các bước:**

1. **Lấy config mới nhất** ([scheduler.go:412-413](../../internal/runner/scheduler.go)) — `config.VerifyConfig = config.GetVerifyConfig()` (realtime).
2. **Inject `OnEmailCreated`** ([scheduler.go:417-421](../../internal/runner/scheduler.go)) — bridge runner-level callback (biết accountID) → verify-level (không biết accountID) qua closure capture `acc.ID`.
3. **Acquire proxy** ([scheduler.go:450-468](../../internal/runner/scheduler.go)) — ưu tiên `acc.Proxy`; nếu rỗng → `config.AcquireProxy(ctx, workerID)`. `defer releaseProxy(result.Status == "Live")` (giữ IP cho account kế nếu Live). Nếu `RequireProxy && proxy==""` → abort, KHÔNG dùng IP máy.
4. **CheckIP background** ([scheduler.go:471-480](../../internal/runner/scheduler.go)) — chỉ để hiển thị cột "IP CHẠY", không block login.
5. **Tạo `facebook.Session`** ([scheduler.go:483-501](../../internal/runner/scheduler.go)) — pointer (login sẽ mutate token). Forward `DeviceID/FamilyDeviceID/Srnonce/SessionlessCryptedUID/Email/EmailMeta` từ reg để verify reuse.
6. **Set UA theo platform** ([scheduler.go:527-551](../../internal/runner/scheduler.go)):
   - Nếu `!UseOriginalUA`: ưu tiên `facebook.PlatformVerifyUA(verifyPlatform, country)` (factory đăng ký bởi mỗi verify package qua `RegisterPlatformVerifyUA` — UA đúng FBAV của version).
   - Fallback `BuildUA` động nếu platform không có factory.
7. **Resolve `verifyPlatform` 1 LẦN** ([scheduler.go:516-525](../../internal/runner/scheduler.go)) — ổn định suốt account (tránh lệch giữa quyết định login-skip và verify thật). Default `facebook.PlatformWeb`.
8. **OUTER retry loop** ([scheduler.go:554-828](../../internal/runner/scheduler.go), `maxOuterAttempts ≥ 2`): mỗi outer attempt reload config (đổi mail provider) + reset `session.FbDtsg/Jazoest/Lsd`. Chạy lại khi inner loop xong mà vẫn Unknown (thường do OTP không về → cấp email mới).
9. **INNER retry loop** ([scheduler.go:582-822](../../internal/runner/scheduler.go), `maxAttempts = 2`): retry toàn bộ login+verify khi lỗi mạng tạm thời.
10. **`switch verifyPlatform`** ([scheduler.go:606-744](../../internal/runner/scheduler.go)) — **TOKEN ACQUISITION theo platform** (chi tiết §5).
11. **Resolve verifier + verify** ([scheduler.go:766-769](../../internal/runner/scheduler.go)):
    ```go
    ver, _ := facebook.NewVerifier(verifyPlatform)
    verifyResult := ver.Verify(ctx, session, config.VerifyConfig, dateFolder,
        func(uid, msg string) { notify(msg) })
    ```
12. **Phân loại kết quả** ([scheduler.go:771-821](../../internal/runner/scheduler.go)):
    - Propagate `result.Token = session.Token` (iOS login đổi token EAAAAU→EAAAAAY) + `result.Cookie = session.Cookie`.
    - `Live`/`Die` → `goto done` (terminal).
    - `isTokenDead` → `Die`; `isBloksCheckpoint` → `unknown`; `isNetworkError` → `unknown`; `isOTPError` → `break` (outer cấp email mới).
13. **Post-loop CheckUID** ([scheduler.go:835-849](../../internal/runner/scheduler.go)): nếu vẫn Unknown + có UID → `CheckLiveDieByPicture` để giảm thất thoát (UID chết → Die ngay).
14. **`done:`** ([scheduler.go:851-874](../../internal/runner/scheduler.go)):
    - Lưu datr vào `Config/Cookie/datr_pool.txt` cho mọi outcome có datr (async).
    - Gọi `config.OnAccountDone(...)` → app.go ghi file (xem §9).

**Output:** `AccountResult{Status, Message, Email, UserAgent, TwoFA, Token, Cookie, VerifyPlatform}` ([scheduler.go:112](../../internal/runner/scheduler.go)).

---

## 5. Token acquisition theo platform (RẤT QUAN TRỌNG)

> Đây là **đặc trưng cốt lõi** của verify. Mỗi họ platform cần một **loại auth khác nhau**, và hệ thống đảm bảo **đối xứng**: gặp sai loại token → login lại lấy đúng loại.

### 5.1 Bảng quy tắc

| Họ platform | Loại auth cần | Cách lấy | Prefix token |
|---|---|---|---|
| **Android-family** (s23, s273, s399, s415..s564, s5xx...) | User access token `EAAAAU` | REST `/auth/login` (`FetchAndroidTokenLegacyWithCookie`) | `EAAAAU` |
| **iOS** (ios562, ios563) | User access token `EAAAAAY` | CAA `send_login_request` (`ios562reg.FetchIOSToken`) | `EAAAAAY` |
| **WebAndroid** | Cookie m.facebook.com | dùng `session.Cookie` trực tiếp (không login) | — |
| **Web (MFB)** | fb_dtsg/jazoest từ cookie | `LoginWithCookieMobile` parse HTML | — |
| **Token API** | `EAA...` có sẵn | KHÔNG tự login | EAA* |

### 5.2 Android-family — fetch EAAAAU (scheduler tự lo TRƯỚC khi verify)

**File:** [scheduler.go:607-687](../../internal/runner/scheduler.go) (case Android-family).

Logic (đánh số):

1. **Loại token iOS sai loại** ([scheduler.go:653-656](../../internal/runner/scheduler.go)): token `EAAAAAY` KHÔNG hợp lệ cho endpoint b-graph Android → set `session.Token = ""` để login lại.
   ```go
   if strings.HasPrefix(session.Token, "EAAAAAY") {
       notify("Token iOS (EAAAAAY) không dùng cho Android verify → login lấy EAAAAU đúng loại...")
       session.Token = ""
   }
   ```
2. **Fetch nếu thiếu** ([scheduler.go:657-671](../../internal/runner/scheduler.go)): `needFetch = !isValidAndroidToken(token) && UID != "" && Password != ""`. Gọi `web.FetchAndroidTokenLegacyWithCookie(...)` (timeout 30s) → nhận `(tok, newCookie)`. Chỉ nhận token có prefix `EAAAAU`; cookie mới → propagate `result.Cookie` lên UI.
3. **Bỏ account nếu không có token** ([scheduler.go:680-687](../../internal/runner/scheduler.go)): nếu sau cùng vẫn không `isValidAndroidToken` → `Status="unknown"`, `goto done`.

`isValidAndroidToken` ([scheduler.go:901-903](../../internal/runner/scheduler.go)):
```go
func isValidAndroidToken(tok string) bool {
    return strings.HasPrefix(tok, "EAAAAU") || strings.HasPrefix(tok, "EAAAAAY")
}
```

**FetchAndroidTokenLegacy chi tiết** — [android_token_legacy.go:75](../../internal/facebook/register/web/android_token_legacy.go):
- Endpoint: `POST https://b-graph.facebook.com/auth/login` ([android_token_legacy.go:33](../../internal/facebook/register/web/android_token_legacy.go)).
- Body: form-urlencoded với password plaintext `#PWD_FB4A:0:<ts>:<pwd>` + `api_key` + `sig` (MD5 sorted params + app_secret) + `access_token` (app token) ([android_token_legacy.go:103-141](../../internal/facebook/register/web/android_token_legacy.go)).
- `generate_session_cookies=1` → response trả `session_cookies[]` → `composeLegacyCookies` ghép thành cookie string mới ([android_token_legacy.go:209-223](../../internal/facebook/register/web/android_token_legacy.go)).
- REST classic API **stable** — không phụ thuộc Bloks schema rotation. Đây là lý do "PORT S399".

### 5.3 iOS — fetch EAAAAAY (verifybase tự lo BÊN TRONG verify)

Khác Android: scheduler **KHÔNG pre-login** cho iOS ([scheduler.go:688-693](../../internal/runner/scheduler.go)):
```go
case facebook.PlatformIOS562, facebook.PlatformIOS563:
    notify("[iOS] Verify FBIOS (...) — token EAAAAAY bắt buộc (login trong verify nếu thiếu)")
```

Việc login được **delegate xuống** `verifybase.RunVerify` qua `Spec.ValidateToken` + `Spec.FetchToken` ([ios562/steps.go:108-123](../../internal/facebook/verify/ios562/steps.go)):
```go
ValidateToken: func(tok string) bool {
    return strings.HasPrefix(tok, "EAAAAAY")    // chỉ EAAAAAY — loại EAAAAU
},
FetchToken: func(fctx context.Context, sess *facebook.Session) (string, error) {
    cc := verifybase.CountryFromPhone(sess.Phone)
    tok, cookie, err := ios562reg.FetchIOSToken(fctx, sess.UID, sess.Password, sess.Datr, sess.Proxy, cc, notify)
    if err != nil { return "", err }
    if cookie != "" { sess.Cookie = cookie }   // ưu tiên cookie login MỚI
    return tok, nil
},
```

**FetchIOSToken chi tiết** — [login.go:35](../../internal/facebook/register/ios562/login.go):
- Endpoint: `graphURL` (`graph.facebook.com/graphql`) — CAA `send_login_request` ([login.go:87](../../internal/facebook/register/ios562/login.go)).
- Mã hóa password 1 lần qua RSA (`#PWD_WILDE:2`, fallback `#PWD_FB4A:0`) ([login.go:64](../../internal/facebook/register/ios562/login.go)).
- Retry login 3 lần (backoff) cho lỗi transient; lỗi dứt khoát (checkpoint / sai mật khẩu) → dừng ngay ([login.go:100-107](../../internal/facebook/register/ios562/login.go)).
- Chỉ nhận token `EAAAAAY` ([login.go:95-98](../../internal/facebook/register/ios562/login.go)):
  ```go
  if outcome, _ := parseCreateAccountResponse(resp); outcome != nil && strings.HasPrefix(outcome.AccessToken, "EAAAAAY") {
      return outcome.AccessToken, outcome.Cookie, nil
  }
  ```

### 5.4 Đối xứng (2 chiều)

```
Account có token EAAAAU (Android)  +  verify iOS    → ValidateToken EAAAAAY FAIL → FetchToken login iOS lấy EAAAAAY
Account có token EAAAAAY (iOS)     +  verify Android → strings.HasPrefix EAAAAAY → set Token="" → /auth/login lấy EAAAAU
```

Đây chính là luật trong [§13.7 — Token đúng loại theo platform (ĐỐI XỨNG)](./add-facebook-reg-version.md#137-login-at-verify--tokencookie-hiển-thị-realtime-cập-nhật-2026-05-31).

### 5.5 WebAndroid & Web (MFB)

- **WebAndroid** ([scheduler.go:694-702](../../internal/runner/scheduler.go)): cần `session.Cookie != ""` → skip login. Cookie rỗng → `Status="unknown"`, bỏ account.
- **Web/MFB** ([scheduler.go:703-732](../../internal/runner/scheduler.go)): **bắt buộc** `LoginWithCookieMobile(ctx, session)` để parse `fb_dtsg/jazoest/lsd/datr` từ HTML m.facebook.com. Lỗi `isCookieDead` → `goto done` (không retry); `isNetworkError` → `unknown`.
- **default (SAFETY NET)** ([scheduler.go:733-743](../../internal/runner/scheduler.go)): platform chưa add vào switch → `Status="error"` rõ ràng (tránh bug cũ: silent fall xuống cookie login).

---

## 6. verifybase.RunVerify — pipeline chung 8 bước

**File:** [verifybase/run.go:192](../../internal/facebook/verify/verifybase/run.go) — `func RunVerify(ctx, session, cfg, outputPath, onStatus, spec Spec) *facebook.VerifyResult`

**Mục tiêu:** orchestration **DÙNG CHUNG** cho tất cả variant Bloks (s23, s5xx, s6xx, iOS). Variant chỉ inject khác biệt qua `Spec` (function fields). Đây là điểm hội tụ của kiến trúc verify.

### 6.1 Spec — điểm tùy biến per-variant

**File:** [verifybase/run.go:23-131](../../internal/facebook/verify/verifybase/run.go). Các field quan trọng:

| Field | Mục đích |
|---|---|
| `Tag`, `DocID`, `BloksVer`, `StylesID`, `IsPushOn` | Constants per-version (client_doc_id, bloks_versioning_id...) |
| `FixUA` | Sửa/regen UA nếu sai platform (Android cần FBAN/FB4A, iOS cần FBAN/FBIOS) |
| `BuildHeaders` | Dựng header có thứ tự (legacy s23/s55x vs new-style s56x vs FBIOS) |
| `BuildAddEmailBody` / `BuildConfirmBody` / `BuildResendBody` | Body form-urlencoded per-version |
| `CreateClient` | nil → OkHttp4Android13 TLS; iOS → `CreateIOSClient` (Safari iOS) |
| `GraphEndpoint` | nil → `b-graph.facebook.com/graphql`; iOS → `graph.facebook.com/graphql` |
| `MachineIDFunc` | nil → datr; iOS → base64url 24-char |
| `CheckConfirmSuccess` | nil → contains "confirmation_success"; iOS override |
| `CheckLiveDieFunc` | nil → `CheckLiveDieCombined` (token-first); iOS → `CheckLiveDiePictureFirst` |
| `ValidateToken` / `FetchToken` | nil → EAAAAU/EAAAAAY + REST login; iOS → chỉ EAAAAAY + CAA login (§5.3) |
| `Enable2FA` / `PostConfirm` | hook 2FA TOTP + addinfo |
| `Srnonce` / `SessionlessCryptedUID` / `CloudTrustToken` | iOS session tokens từ reg partial |

### 6.2 Tiền xử lý (trước STEP 1)

1. **Tag** ([run.go:195-198](../../internal/facebook/verify/verifybase/run.go)) — ưu tiên `cfg.UserApiLabel` (tên API user chọn) hơn `Spec.Tag` hardcoded.
2. **Token check** ([run.go:206-246](../../internal/facebook/verify/verifybase/run.go)):
   - `validate := spec.ValidateToken` (nil → `isValidUserToken` = EAAAAU hoặc EAAAAAY, [run.go:188-190](../../internal/facebook/verify/verifybase/run.go)).
   - Nếu `!validate(token) && !SkipUserTokenCheck`:
     - `spec.FetchToken != nil` (iOS) → login CAA iOS lấy EAAAAAY.
     - else nếu có UID+Password → `webreg.FetchAndroidTokenLegacyWithCookie` (REST /auth/login).
   - Sau cùng vẫn không hợp lệ → return `VerifyResult{Status:"error", Message:"Missing/invalid access token..."}`. **Verify CHỈ chạy khi có user token thật** — chặn token rỗng/rác lọt vào AddEmail.
3. **CreateClient** ([run.go:256-265](../../internal/facebook/verify/verifybase/run.go)) — `CreateClient` (OkHttp Android) hoặc `spec.CreateClient` (iOS Safari). `defer client.CloseIdleConnections()`.
   - Client config ([helpers.go:79-94](../../internal/facebook/verify/verifybase/helpers.go)): timeout 30s, cookie jar, `WithInsecureSkipVerify`, `WithNotFollowRedirects`, proxy.
4. **Inject cookie** ([run.go:267-279](../../internal/facebook/verify/verifybase/run.go)) — extract `datr` từ cookie string, `InjectVerifyCookies` chỉ inject `datr/sb/fr` vào jar ([helpers.go:185-206](../../internal/facebook/verify/verifybase/helpers.go)).
5. **Session IDs** ([run.go:281-297](../../internal/facebook/verify/verifybase/run.go)) — reuse `deviceID/familyDeviceID` từ reg; `waterfallID` mới; `machineID` = datr (hoặc `MachineIDFunc`).
6. **Email service** ([run.go:299-363](../../internal/facebook/verify/verifybase/run.go)):
   - proxy override cho mail (rent → `PickRentMailProxy`, temp → `PickTempMailProxy`).
   - `customUsername` nếu `FmUserTmpMail` (từ phone/uid).
   - `email.New(email.Options{... tất cả provider keys ...})` + `defer emailSvc.Close()`.
7. **TempMail reuse** ([run.go:365-387](../../internal/facebook/verify/verifybase/run.go)) — nếu reg đã tạo mail tạm (`session.Email != "" && session.EmailMeta != ""`) và `email.RestoreIfPossible` OK → `reuseMail=true` → **skip CreateEmail + skip AddEmail**, đọc OTP từ inbox sẵn có. Notify prefix "[REUSE]".
   - else `RetryCreateEmail` (3 lần, [helpers.go:147-169](../../internal/facebook/verify/verifybase/helpers.go)).
8. **Chuẩn bị input body** ([run.go:389-415](../../internal/facebook/verify/verifybase/run.go)) — `SplitFullName`, `RandomSimProfile(country)`, locale (cookie nếu `DeepFakeLocale`), **FixUA** (regen UA), `buildSessionCtx` + `InitPinnedHeaders` + `SetupSessionCtx`.

### 6.3 STEP 1 — AddEmail (Bloks)

**File:** [run.go:417-509](../../internal/facebook/verify/verifybase/run.go).

1. Skip nếu `reuseMail`. Ngược lại:
2. Build body: `spec.BuildAddEmailBody(...)`, headers: `spec.BuildHeaders(sc, AddEmailFriendlyName, true)`.
3. Endpoint: `BgraphURL` (hoặc `spec.GraphEndpoint` cho iOS).
4. `addResp, err := DoPost(addCtx, client, addEndpoint, addBody, addHeaders)` (timeout 30s).
   - **DoPost** ([helpers.go:97-142](../../internal/facebook/verify/verifybase/helpers.go)): gzip body → `fhttp.NewRequestWithContext` POST, set header **có thứ tự** (`req.Header[HeaderOrderKey]`), detect checkpoint qua header `X-Fb-Integrity-Required` / `X-Fb-Integrity-Requires-Login` → inject `{"error":{"code":459,"message":"checkpointed"}}`.
5. **Phân loại response** ([run.go:447-506](../../internal/facebook/verify/verifybase/run.go)):
   - `isSuccess`: chứa "Check your email" / "CAA_REG_CONFIRMATION" / "confirmation_code"...
   - `isExplicitError`: "errors":[{ / rate_limit / checkpoint / email_already_used / account disabled...
   - `isBloksAction` fallback: chứa `fb_bloks_action` mà KHÔNG có error → **optimistic SUCCESS** (đợi OTP). Tránh false-negative khi FB đổi Bloks DSL.
   - `mailIsBad`: chỉ `email_already_used`/`email_is_invalid` → KHÔNG recycle mail. Lỗi khác → `email.ReleaseIfPossible` (trả mail về pool reuse, tránh phí).
6. **iOS cloud_trust_token async** ([run.go:478-488](../../internal/facebook/verify/verifybase/run.go)): nếu `spec.BuildCloudTrustTokenBody != nil` → POST `bk.cloud_trust_token.async` sau AddEmail thành công.

### 6.4 STEP 2 — WaitOTP + Resend

**File:** [run.go:511-549](../../internal/facebook/verify/verifybase/run.go).

1. `waitSec = cfg.TimeDelaySendCode` (def 30), `pollMs = cfg.WaitMailMs` (def 2000), `maxRetry = waitSec*1000/pollMs`.
2. `StartOTPHeartbeat` ([helpers.go:45-65](../../internal/facebook/verify/verifybase/helpers.go)) — phát status mỗi 5s trong lúc chờ (UI không treo).
3. `code, err := emailSvc.WaitForCode(ctx, maxRetry, pollMs)`.
4. **Timeout** → nếu `!cfg.SendAgainCode`: `email.ReleaseIfPossible` + return `Status:"error"` "OTP timeout". Nếu bật resend: `spec.BuildResendBody` → DoPost `ResendFriendlyName` → WaitForCode lần 2.

### 6.5 STEP 3 — ConfirmCode (retry x3)

**File:** [run.go:551-615](../../internal/facebook/verify/verifybase/run.go).

1. Delay `cfg.DelayConfirmEmail` trước confirm (nếu > 0).
2. Build `spec.BuildConfirmBody(... code ...)` + headers `ConfirmFriendlyName`.
3. Loop tối đa 3 lần:
   - `confirmResp, _ := DoPost(...)`.
   - `isConfirmOK`: `spec.CheckConfirmSuccess(resp)` (nil → contains "confirmation_success").
   - Nếu chứa "checkpointed" / `code":459` → return `Status:"Die"` "Checkpoint after confirm".
   - Confirm OK → `notify("Email confirmed!")`, break.
4. Không OK sau 3 lần → `Status:"error"` "Confirm failed after retries".

### 6.6 STEP 4 — CheckLiveDie (loop 5s, bail early)

**File:** [run.go:617-657](../../internal/facebook/verify/verifybase/run.go).

1. `status = "Live"` mặc định.
2. Nếu `cfg.CheckLiveDie`: loop mỗi `checkInterval=5s` đến hết `checkDelay` (`cfg.TimeDelayCheck`, def 5):
   - `checkFn := CheckLiveDieCombined` (hoặc `spec.CheckLiveDieFunc` cho iOS = `CheckLiveDiePictureFirst`).
   - `s := checkFn(ctx, session.UserAgent, uid, session.Token)`; nếu `"Die"` → bail ngay.
3. Nếu tắt CheckLiveDie → chỉ chờ `checkDelay` rồi báo Live.

### 6.7 STEP 5 — 2FA (nếu bật + Live)

**File:** [run.go:659-679](../../internal/facebook/verify/verifybase/run.go).

- Nếu `spec.Enable2FA != nil && cfg.Enable2FA && status=="Live"`:
  - `emailOTPFn` đọc OTP reauth từ cùng `emailSvc` (cap 3 lần poll).
  - `secret, err := spec.Enable2FA(...)`. Lỗi → non-fatal (log, không fail). Thành công → `twoFAKey = secret`.
- Ví dụ Android: `enable2FAForS560` ([s562/steps.go:74-91](../../internal/facebook/verify/s562/steps.go)) dùng `androidsec.SecurityManager`. iOS562 set `Enable2FA: nil` (chưa hỗ trợ).

### 6.8 STEP 6 — PostConfirm + VerifyResult

**File:** [run.go:681-703](../../internal/facebook/verify/verifybase/run.go).

- `spec.PostConfirm(...)` (nếu Live) — vd s562 chạy `addinfo.RunAddInfo` ([s562/steps.go:61-69](../../internal/facebook/verify/s562/steps.go)).
- Return:
  ```go
  return &facebook.VerifyResult{
      Success:   status == "Live",
      Status:    status,        // "Live"/"Die"
      Message:   msg,           // "Live — Email: x — 2FA: y"
      Email:     tempEmail,
      UserAgent: verifyUA,      // UA đã verify ok
      TwoFA:     twoFAKey,
  }
  ```

> **Ví dụ Spec Android new-style:** [s562/steps.go:33-72](../../internal/facebook/verify/s562/steps.go) — không set `ValidateToken/FetchToken` (dùng default verifybase), `BuildHeaders: verifybase.BuildNewStyleHeaders`, `IsPushOn: true`.

---

## 7. Per-platform verify (so sánh)

| Platform | Package | Auth | Endpoint | UA | Orchestration |
|---|---|---|---|---|---|
| **Android Bloks** (s23, s415..s564, s5xx/s6xx) | `verify/sXXX` | OAuth EAAAAU header | `b-graph.facebook.com/graphql` | FBAN/FB4A | `verifybase.RunVerify` (§6) |
| **iOS Native** (ios562, ios563) | `verify/ios562` | OAuth EAAAAAY header | `graph.facebook.com/graphql` | FBAN/FBIOS | `verifybase.RunVerify` (§6) + Spec iOS |
| **Android legacy** (s273) | `verify/s273` | OAuth EAAAAU header | `b-api.facebook.com/method/user.*` | Vivo V2242A FBAV/273 | tự viết (không qua verifybase) |
| **Android legacy** (s399) | `verify/s399` | OAuth EAAAAU header | `graph.facebook.com/me/*_contactpoint` | FBAV/399 | tự viết |
| **WebAndroid** | `verify/webandroid` | Cookie | `m.facebook.com/changeemail` → setemail | Chrome Mobile | tự viết |
| **Web (MFB)** | `verify/web` | Cookie + fb_dtsg | `m.facebook.com` B1-B5 | Chrome Desktop/Mobile | tự viết (B1→B5) |
| **Token API** | `verify/token` | EAA có sẵn | `api.facebook.com/method/user.*` | Android pool | tự viết |

### 7.1 Android Bloks (s5xx/s6xx) — thin wrapper

Mỗi version chỉ là 1 thin wrapper:
- `verify.go`: register vào factory ([s562/verify.go:19-24](../../internal/facebook/verify/s562/verify.go)).
- `steps.go`: build `Spec` rồi `return verifybase.RunVerify(...)` ([s562/steps.go:33-72](../../internal/facebook/verify/s562/steps.go)).
- Khác nhau chủ yếu: `verifyDocID`, `verifyBloksVer`, `defaultStylesID`, `IsPushOn`, style headers (legacy vs new).

### 7.2 iOS — Bloks CAA single-shot

[ios562/steps.go:43-126](../../internal/facebook/verify/ios562/steps.go): khác Android ở:
- `CreateClient: CreateIOSClient` (TLS Safari iOS).
- `GraphEndpoint: GraphURL` (`graph.facebook.com`, không phải `b-graph`).
- `MachineIDFunc: iosMachineID` (base64url 24-char).
- `CheckConfirmSuccess` custom (fb_bloks_action present + no confirmation_failure).
- `CheckLiveDieFunc: CheckLiveDiePictureFirst` (picture trước token).
- Body envelope iOS: `voiceover_enabled / generic_attachment_* / nt_context FDS+XMDS` ([ios562/steps.go:264-295](../../internal/facebook/verify/ios562/steps.go)), `reg_info` đầy đủ ~150 field ([ios562/steps.go:329-550](../../internal/facebook/verify/ios562/steps.go)).
- Headers FBIOS với `authorization: OAuth <EAAAAAY hoặc app token>` ([ios562/steps.go:168-215](../../internal/facebook/verify/ios562/steps.go)).

### 7.3 s273 / s399 — REST FB4A legacy (tự viết, KHÔNG dùng verifybase pipeline)

[s273/verify.go:42](../../internal/facebook/verify/s273/verify.go) + [s399/verify.go:44](../../internal/facebook/verify/s399/verify.go): cấu trúc giống nhau nhưng tự implement add/wait/confirm/check:
- Endpoint cũ REST: s273 `b-api.facebook.com/method/user.editregistrationcontactpoint` + `user.confirmcontactpoint`; s399 `graph.facebook.com/me/edit_registration_contactpoint` + `me/confirm_contactpoint`.
- Bắt buộc `session.Token != ""` + `session.UID != ""` ở đầu (token đã được scheduler fetch sẵn ở §5.2).
- Response success: bare `"true"` hoặc `{"result":true}` ([s273/verify.go:200-204](../../internal/facebook/verify/s273/verify.go)).
- **s273 có Step 3.5** ([s273/verify.go:333-352](../../internal/facebook/verify/s273/verify.go)): post-confirm check `Graph /me?fields=id,email` để bắt token checkpoint NGAY SAU confirm (DIE) hoặc email không attach (NO_EMAIL → silent fail). **LUÔN chạy** không phụ thuộc `cfg.CheckLiveDie`.
- Live/Die: dùng `CheckLiveDieByPicture` (không có token check vì REST).

### 7.4 WebAndroid — cookie m.facebook.com

[webandroid/verify.go:38](../../internal/facebook/verify/webandroid/verify.go):
- Bắt buộc `session.Cookie != ""`, normalize UA về Chrome Mobile ([webandroid/verify.go:55-59](../../internal/facebook/verify/webandroid/verify.go)).
- Flow: `addEmailWithNotify` (GET changeemail → POST setemail trả `state`) → WaitOTP → `confirmEmail(... state)` → optional Enable2FA (AccountsCenter) → CheckLiveDie.
- KHÔNG có resend cho WebAndroid ([webandroid/verify.go:181](../../internal/facebook/verify/webandroid/verify.go)).
- **Post-verify pending check** ([webandroid/verify.go:269-274](../../internal/facebook/verify/webandroid/verify.go)) — §8.4.

### 7.5 Web (MFB) — B1→B5 waterfall

[web/verify.go:93](../../internal/facebook/verify/web/verify.go) `VerifyAccount`:
1. `ChangeLanguageV2` → en_US.
2. Kiểm tra `session.FbDtsg && session.Jazoest` (từ login cookie ở scheduler) — thiếu → fail.
3. `runB1toB5` ([web/verify.go:360](../../internal/facebook/verify/web/verify.go)): **B1** SelectMail → **B2** ChangeEmail → **B3** SubmitEmail (FB gửi OTP) → **B4** LoadConfirmation → poll OTP (resend tối đa `MaxResend`, cap 2) → **B5** ConfirmOTP.
4. CheckLiveDie `CheckLiveDieCombined` + **post-verify pending check** ([web/verify.go:298-303](../../internal/facebook/verify/web/verify.go)).

---

## 8. Live/Die detection + lớp chống false-positive

**File:** [verifybase/helpers.go:219-415](../../internal/facebook/verify/verifybase/helpers.go).

### 8.1 CheckLiveDieByToken (gold standard)

[helpers.go:250-296](../../internal/facebook/verify/verifybase/helpers.go) — `GET graph.facebook.com/me?fields=id,name&access_token=<tok>`:
- **Die**: response chứa `oauthexception` / `checkpoint` / `session is invalid` / `account disabled`.
- **Live**: chứa `"id"` + `"name"`.
- **Unknown**: network error / token rỗng.
- **Lý do dùng token check**: FB invalidate token NGAY KHI checkpoint, trong khi picture endpoint trễ 30-60 phút.
- Dùng ĐÚNG UA của account (iOS→FBIOS, Android→FB4A) để FB không ghi session lệch fingerprint.

### 8.2 CheckLiveDieByPicture (fallback)

[helpers.go:351-415](../../internal/facebook/verify/verifybase/helpers.go) — `GET graph.facebook.com/{uid}/picture?type=normal` (client KHÔNG follow redirect):
- FB trả 302 → check `Location` header:
  - chứa `/C5yt7Cqf3zU.jpg` (avatar silhouette mặc định) → **Die**.
  - chứa `scontent.*.fbcdn.net` (CDN ảnh thật) → **Live**.
  - URL lạ → optimistic **Live**.
- Fallback Method 2: `&redirect=false` parse JSON body — không có `height` → Die.

### 8.3 Combined vs Picture-first

- **CheckLiveDieCombined** ([helpers.go:307-315](../../internal/facebook/verify/verifybase/helpers.go)) — **token-first**: có token → token check; "Die"/"Live" trả ngay; "Unknown" → fallback picture. **Default cho Android/Web/WebAndroid.**
- **CheckLiveDiePictureFirst** ([helpers.go:326-337](../../internal/facebook/verify/verifybase/helpers.go)) — **picture-first** rồi token fallback. **Dùng cho iOS** (qua `Spec.CheckLiveDieFunc`). Đánh đổi có chủ đích: picture trễ → có thể báo Live cho acc vừa checkpoint, token vẫn là lưới fallback.

### 8.4 Lớp chống false-positive (pending/checkpoint)

Picture endpoint vẫn trả ảnh kể cả account pending/checkpoint → cần lớp kiểm tra phụ:

- **Web** ([web/verify.go:298-303](../../internal/facebook/verify/web/verify.go)) + **WebAndroid** ([webandroid/verify.go:269-274](../../internal/facebook/verify/webandroid/verify.go)): nếu `status=="Live" && Cookie != ""` → `detectPendingOrCheckpoint(...)`: GET m.facebook.com/ với cookie → nếu redirect `/confirmemail.php` (pending) hoặc `/checkpoint/` (locked) → **demote Live → Die**. (`login.php` KHÔNG demote.)
- **s273** ([s273/verify.go:338-352](../../internal/facebook/verify/s273/verify.go)): Step 3.5 `Graph /me?fields=id,email` — token checkpointed ngay sau confirm → Die; email không attached → silent fail.
- **scheduler post-loop** ([scheduler.go:835-849](../../internal/runner/scheduler.go)): Unknown + có UID → `CheckLiveDieByPicture` → UID chết → Die.

---

## 9. Kết quả verify → file (SuccessVerify / Die / Unknown)

### 9.1 Đường đi kết quả

```
ver.Verify → VerifyResult (scheduler) → AccountResult → config.OnAccountDone(...) [app.go]
   → recordVerifyOutcome / recordBuildUAVerVersion / recordMailDomainOutcome (counters)
   → update a.accounts[i] (grid) + emit "verify:account-done"
   → go saveVerifyOutcome(verifyWriter, verifyCounters, status, message, doneAcc, verifyInstance)
```

`OnAccountDone` ([app.go:2956-3028](../../app.go)):
- `s := strings.ToLower(status)` ("" → "unknown").
- `a.recordVerifyOutcome(verifyPlatform, s=="live")` — multi-version stats.
- Learning loop: verify Live qua `PlatformWeb` → `AppendUAToPool(WebChrome, ua)` ([app.go:2970-2974](../../app.go)).
- Token: `preferUserAccessToken(old, new)` — giữ token đúng loại lên cột TOKEN.
- Cookie mới (login verify) → ghi `a.accounts[i].Cookie`.

### 9.2 saveVerifyOutcome — ghi file

**File:** [app.go:2060](../../app.go) — `func saveVerifyOutcome(writer, counters, status, message, acc, verifyInstance)`:

| Status | File | Format | Ghi chú |
|---|---|---|---|
| **live** (có 2FA) | `SuccessVerify.txt` | 9-field `FormatVerify` | + `SuccessVerifyUG_<instance>.txt` (UA) |
| **live** (không 2FA) | `SuccessVerify_No2FA.txt` | 6-field `FormatReg` (giống SuccessReg) | counter `FbAppVersion++` |
| **die** | `DieAfterVerify.txt` | `UpsertUID` (dedupe theo UID) | |
| **unknown** | `UnknownErrorCheckLiveDieApi.txt` | `Append` | |

- **Country** ([app.go:2075-2085](../../app.go)): ưu tiên `acc.Location` (từ proxy suffix) → fallback FBLC trong UA → fallback locale trong cookie.
- **Email KHÔNG được lưu** vào file account thành công (yêu cầu user, [app.go:2092](../../app.go)).
- **Dispatch detail files** ([app.go:2135-2141](../../app.go)) — `DispatchVerifyDetails(s, message, line)` ghi file con theo sub-status (ChinaMail_CantGetCode.txt, LoginFbFailed_<code>.txt...).

> **Gotcha:** `web.SaveAccountToFolder` ([web/verify.go:607-646](../../internal/facebook/verify/web/verify.go)) chỉ ghi **Die/Unknown** — **Live** do `saveVerifyOutcome` (app.go) ghi qua `OnAccountDone` để tránh ghi trùng 2 dòng/account. Tham khảo [§16 register doc](./add-facebook-reg-version.md#878-thêm-verify-version-mới) về layout file kết quả.

---

## 10. Phân loại lỗi & quy tắc retry

**File:** [scheduler.go:306-378](../../internal/runner/scheduler.go).

| Helper | Pattern match | Hành động |
|---|---|---|
| `isOTPError` | "không nhận được otp" / "otp timeout" / "no otp code received" | `break` inner → outer cấp email mới (KHÔNG retry inner: OTP đã hết hạn phía FB) |
| `isCookieDead` | "cookie không hợp lệ" / "đăng nhập bằng cookie thất bại" / "không có cookie" | `goto done` (gửi lại cookie chết = vô ích) |
| `isBloksCheckpoint` | "fb_bloks_action" / "bloks_bundle_action" / "bloks_payload" | `Status="unknown"` (cần thao tác thủ công) |
| `isTokenDead` | "malformed access token" / (http 401 + access token) | `Status="Die"` (token chết vĩnh viễn) |
| `isNetworkError` | "context deadline exceeded" / "connection refused" / "dial tcp" / "eof"... | `Status="unknown"` (retry cùng proxy = cùng kết quả) |

**Retry 2 tầng** (xem §4):
- **Inner** (`maxAttempts=2`): lỗi mạng tạm thời → login + verify lại.
- **Outer** (`maxOuterAttempts≥2`): inner xong vẫn Unknown (OTP không về) → reload config + email provider mới.
- **RetryUnknownNow pass 2** (scheduler-level): re-queue acc Unknown/Error với proxy slot mới.

---

## 11. Edge case & gotcha

1. **TempMail reuse** — reg tạo mail tạm + lưu `EmailMeta` → verify `RestoreIfPossible` → **skip CreateEmail + skip AddEmail** (verifybase/s273/s399) hoặc skip CreateEmail nhưng vẫn AddEmail (web/webandroid vì cần `state`). Tiết kiệm cost provider. Xem [run.go:365-387](../../internal/facebook/verify/verifybase/run.go).
2. **Mail recycle khi fail** — `email.ReleaseIfPossible` trả mail về pool reuse trên các lỗi: HTTP error AddEmail, OTP timeout, AddEmail error (TRỪ `email_already_used`/`email_is_invalid` = mail thực sự hỏng). Tránh phí tiền mail rent.
3. **Token rỗng/sai loại → BỎ account, KHÔNG verify** — bảo vệ chống AddEmail bằng cookie/token rác. Android-family Unknown ([scheduler.go:683-687](../../internal/runner/scheduler.go)), verifybase error ([run.go:242-245](../../internal/facebook/verify/verifybase/run.go)).
4. **Header order quan trọng** — `DoPost` set `req.Header[fhttp.HeaderOrderKey]` để TLS fingerprint khớp app thật ([helpers.go:112-117](../../internal/facebook/verify/verifybase/helpers.go)). Sai thứ tự → FB nghi bot.
5. **Checkpoint detect qua HTTP header** — `X-Fb-Integrity-Required` / `X-Fb-Integrity-Requires-Login` được DoPost inject thành JSON error 459 ([helpers.go:128-136](../../internal/facebook/verify/verifybase/helpers.go)) — port C# `FacebookCheckpointDetectorUtils`.
6. **iOS authorization fallback app token** — nếu không có user token EAAAAAY (sessionless flow) → header dùng `iosAppToken` `6628568379|...` ([ios562/steps.go:174-176](../../internal/facebook/verify/ios562/steps.go)).
7. **UserApiLabel = platform thực** — scheduler set `config.VerifyConfig.UserApiLabel = verifyPlatform` mỗi attempt ([scheduler.go:763-765](../../internal/runner/scheduler.go)) → log hiển thị đúng version round-robin (không phải focus version cố định).
8. **WorkerCtx vs poolCtx** — Stop cancel cả hai; nhưng worker đang chạy dở dùng WorkerCtx để chạy hết HTTP request hiện tại (không abort giữa chừng AddEmail/Confirm).
9. **OnEmailCreated realtime** — email hiện lên cột EMAIL/PHONE NGAY khi tạo (trước verify done) qua bridge closure capture accountID ([scheduler.go:417-421](../../internal/runner/scheduler.go) + [app.go:2940-2954](../../app.go)).
10. **DeepFakeLocale** — nếu bật, locale lấy từ cookie `locale=xx_YY` ([run.go:396-401](../../internal/facebook/verify/verifybase/run.go) + [helpers.go:209-217](../../internal/facebook/verify/verifybase/helpers.go)).

---

## Tham chiếu nhanh (file:line)

| Thành phần | File:line |
|---|---|
| App.RunVerify (entry) | [app.go:2280](../../app.go) |
| saveVerifyOutcome | [app.go:2060](../../app.go) |
| OnAccountDone | [app.go:2956](../../app.go) |
| runner.RunVerify (FIFO pool) | [scheduler.go:150](../../internal/runner/scheduler.go) |
| runOneAccount (retry + token switch) | [scheduler.go:410](../../internal/runner/scheduler.go) |
| isValidAndroidToken | [scheduler.go:901](../../internal/runner/scheduler.go) |
| verifybase.RunVerify (pipeline) | [run.go:192](../../internal/facebook/verify/verifybase/run.go) |
| Spec (điểm tùy biến) | [run.go:23](../../internal/facebook/verify/verifybase/run.go) |
| DoPost / CreateClient | [helpers.go:97](../../internal/facebook/verify/verifybase/helpers.go) / [helpers.go:70](../../internal/facebook/verify/verifybase/helpers.go) |
| CheckLiveDieCombined / PictureFirst | [helpers.go:307](../../internal/facebook/verify/verifybase/helpers.go) / [helpers.go:326](../../internal/facebook/verify/verifybase/helpers.go) |
| iOS Spec + body builders | [ios562/steps.go:43](../../internal/facebook/verify/ios562/steps.go) |
| FetchIOSToken (EAAAAAY) | [login.go:35](../../internal/facebook/register/ios562/login.go) |
| FetchAndroidTokenLegacy (EAAAAU) | [android_token_legacy.go:75](../../internal/facebook/register/web/android_token_legacy.go) |
| Factory NewVerifier | [factory.go:263](../../internal/facebook/factory.go) |
| s273 / s399 (REST legacy) | [s273/verify.go:42](../../internal/facebook/verify/s273/verify.go) / [s399/verify.go:44](../../internal/facebook/verify/s399/verify.go) |
| WebAndroid verify | [webandroid/verify.go:38](../../internal/facebook/verify/webandroid/verify.go) |
| Web MFB (B1-B5) | [web/verify.go:93](../../internal/facebook/verify/web/verify.go) |
| Verifier interface | [interfaces.go:18](../../internal/facebook/interfaces.go) |
| VerifyResult / VerifyConfig / Session | [types.go:412](../../internal/facebook/types.go) / [types.go:266](../../internal/facebook/types.go) / [types.go:24](../../internal/facebook/types.go) |
