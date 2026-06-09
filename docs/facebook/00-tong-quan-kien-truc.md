# 00 — Tổng quan & Kiến trúc HVR

> **File cửa ngõ.** Đọc file này trước để có bản đồ tổng. Các file `01`–`06` (sẽ viết
> sau) đào sâu từng phần (register flow, verify flow, fakeinfo/UA, cookie/datr,
> email providers, proxy...). Khi cần biết "thêm version mới làm sao" → xem
> [add-facebook-reg-version.md](./add-facebook-reg-version.md) (runbook + §13.6/§13.7
> luật token & login iOS). File này tập trung mô tả **FLOW CHẠY THỰC TẾ** end-to-end.

HVR là một **desktop app** tự động **REGISTER (tạo)** + **VERIFY (xác minh email)**
tài khoản Facebook hàng loạt, qua nhiều "version API" của 4 họ client Facebook:

- **Android FB4A** (Samsung S23/S21+...): modern Bloks/GraphQL (`s23`, `s415`–`s564*`) và legacy REST (`s399`, `s273`).
- **iOS native FBIOS** (`ios562`, `ios563`): graph.facebook.com GraphQL + OAuth, token `EAAAAAY`.
- **WebAndroid** (Chrome Mobile, cookie-based): `webandroid`.
- **Web MFB** (m.facebook.com + fb_dtsg): `web` / `mfb`.

---

## 1. Tech stack & mô hình Wails

| Lớp | Công nghệ | File / thư mục |
|-----|-----------|----------------|
| Backend | Go 1.25 ([go.mod](../../go.mod)) | [main.go](../../main.go), [app.go](../../app.go), `internal/**` |
| Desktop shell | [Wails v2](https://wails.io) (`github.com/wailsapp/wails/v2`) | [main.go](../../main.go), [wails.json](../../wails.json) |
| Frontend | Vue 3 + TypeScript + Vite | `frontend/src/**` |
| Router | vue-router (hash history) | [frontend/src/app/router](../../frontend/src/app/router) |
| State | Pinia | [frontend/src/stores](../../frontend/src/stores) |
| Bridge | layer trừu tượng Wails↔Vue | [frontend/src/bridge](../../frontend/src/bridge) |

### 1.1 Wails = WebView2 nhúng + cầu nối Go↔JS

Wails đóng gói một **WebView2 (Windows)** hiển thị frontend Vue (đã build vào
`frontend/dist`, embed thẳng vào exe qua `//go:embed all:frontend/dist` —
[main.go:20-21](../../main.go)). Cầu nối 2 chiều:

```
                       ┌──────────────────────────────────────────────┐
                       │  WebView2 (Vue 3 SPA)                         │
   USER  ───clicks───► │  AccountsPage.vue / store / composables       │
                       │            │ gọi qua bridge layer             │
                       │            ▼                                  │
                       │  frontend/src/bridge/wails/*.ts               │
                       │  → window.go.main.App.RunRegister(...)         │  ← generated bindings
                       └────────────┬─────────────────▲───────────────┘
                          (1) method call      (2) EventsEmit
                                    │                 │
                       ┌────────────▼─────────────────┴───────────────┐
                       │  Go backend — App struct (app.go)             │
                       │  RunRegister / RunVerify / StopXxx / ...       │
                       │  → internal/runner/scheduler.go                │
                       │  → internal/facebook/** (register/verify)      │
                       └───────────────────────────────────────────────┘
```

- **(1) Method call (JS → Go):** mọi `func (a *App) MethodXxx(...)` được `Bind`
  vào Wails ([main.go:109-111](../../main.go)) tự sinh ra binding JS
  `window.go.main.App.MethodXxx`. Wails serialize tham số JSON, gọi Go, trả promise.
- **(2) Event (Go → JS):** Go gọi `runtime.EventsEmit(a.ctx, "verify:status", {...})`,
  frontend lắng nghe qua `EventsOn("verify:status", cb)`. Đây là kênh **realtime** để
  push log/tiến trình từng account lên UI mà không cần polling.

### 1.2 Bridge layer frontend (không import binding trực tiếp)

UI **không** gọi `wailsjs/go/main/App` trực tiếp. Mọi truy cập đi qua
[frontend/src/bridge/client.ts](../../frontend/src/bridge/client.ts):

```ts
// client.ts — chọn implementation runtime: Wails thật vs mock (dev không có Go)
function isWails(): boolean {
  return typeof window !== 'undefined'
    && (window as any)['go'] !== undefined
    && (window as any)['go']['main'] !== undefined
}

export async function getVerifyRunnerService(): Promise<IVerifyRunnerService> {
  const wailsReady = await waitForWails()        // retry 10×200ms — window.go inject muộn hơn Vue mount
  if (wailsReady) {
    const { verifyRunnerWails } = await import('./wails/verify-runner.wails')
    return verifyRunnerWails
  }
  const { verifyRunnerMock } = await import('./mock/verify-runner.mock')
  return verifyRunnerMock
}
```

- **`contracts.ts`** — interface TS (vd `IVerifyRunnerService`, `IAccountService`).
- **`wails/*.ts`** — bản thật wrap generated binding. Vd
  [verify-runner.wails.ts](../../frontend/src/bridge/wails/verify-runner.wails.ts)
  gọi `RunVerify / RunRegister / StopVerify / StopRegister`.
- **`mock/*.ts`** — bản giả cho dev khi chạy `npm run dev` ngoài Wails.
- **Event bus** — [event-bus.wails.ts](../../frontend/src/bridge/wails/event-bus.wails.ts)
  wrap `EventsOn/EventsOff`, trả unsub function để cleanup chính xác per-listener.

> **Gotcha:** một số service (flows, proxies) hiện vẫn dùng mock ngay cả trên Wails
> (xem `client.ts:39-53`) — các binding tương ứng chưa được wire.

---

## 2. Cây thư mục tổng

```
HVR/
├── main.go                  # Entry point: wails.Run + App options (window, single-instance, bind)
├── app.go                   # ~12k dòng — App struct + TẤT CẢ Wails-bound methods + reg/verify orchestration
├── app_getdatr.go           # GetDatr (lấy datr riêng lẻ)
├── app_reg_sxxx.go          # Helper cho các platform reg "Sxxx" (regPlatformList, isRegPlatformSxxx...)
├── app_tempmail_reg.go      # Temp-mail acquire cho register
├── datadir.go               # appDataDir() — CWD = bin/dev (dev) hoặc thư mục exe (prod)
├── cpu_windows.go           # CPU affinity / RAM helpers (Windows)
├── portrange_windows.go     # Mở rộng ephemeral port range (giảm WSAEADDRINUSE)
├── debug.go                 # pprof/snapshot khi env DEBUG_PPROF=1
│
├── internal/
│   ├── runner/
│   │   └── scheduler.go     # FIFO worker pool: RunVerify, runOneAccount (retry 2 tầng)
│   ├── facebook/
│   │   ├── factory.go       # Platform registry (plugin pattern) + hằng số PlatformXxx
│   │   ├── interfaces.go    # Registerer / Verifier / Interactor / FeedReader / SecurityManager
│   │   ├── types.go         # Session, RegInput/RegResult, VerifyConfig/VerifyResult
│   │   ├── login.go         # LoginWithCookieMobile + ParseTokens (fb_dtsg/jazoest/lsd/datr)
│   │   ├── constants.go, options.go, results.go, status.go, machineid/
│   │   ├── register/        # s23, s399, s415..s564*, ios562/563, ioshttp, web, webandroid, android...
│   │   ├── verify/          # mirror register: s23, s415.., ios562, web, webandroid, token, verifybase/
│   │   ├── fakeinfo/        # Sinh device profile, UA (Android/iOS/Chrome), SIM, phone, locale
│   │   ├── addinfo/, checkpoint/, feed/, formdata/, interaction/, security/
│   ├── cookie/              # datr pool, cookie_initial seed, ExtractDatr/AppendDatr
│   ├── fbdata/              # FBAV version/build pool (_reg.txt / _ver.txt split)
│   ├── email/               # rent/ (ZeusX, DongVanFB...) + temp/ (~50 provider) + pool, OTP reader
│   ├── proxy/               # manager, providers (tinsoft/shoplike...), sticky, checkip, health
│   ├── result/              # Writer (UpsertUID dedupe), CounterSet, format file kết quả
│   ├── clonehv/             # CloneHV API client (mua account sẵn)
│   ├── settings/            # model + schema + adapter + store (app_settings.json)
│   ├── httpx/, httpclient/  # HTTP client tuning (proxy, TLS, transport pool)
│   ├── iplookup/, stats/, config/
│
├── frontend/src/
│   ├── main.ts, App.vue
│   ├── app/router/          # routes.ts + index.ts (hash history)
│   ├── pages/               # AccountsPage, ProxySettingsPage, InteractionSetupPage, RegStatsPage...
│   ├── modules/             # accounts, auth-source, flow-settings, proxy-settings, view-settings
│   ├── bridge/              # contracts.ts, client.ts, mock/, wails/
│   ├── stores/              # app.store, preferences.store, uploadLog.store (Pinia)
│   ├── components/, composables/, types/, utils/, constants/
│
├── build/bin/Config/        # Config seed runtime (Cookie, DeviceInfo, Fbapp, Proxy, Settings...)
├── cmd/                     # CLI test harness: regtest, verifytest, proxycheck, test_ua...
└── docs/facebook/           # Tài liệu (file này + add-facebook-reg-version.md)
```

> **Lưu ý:** `internal/facebook/register/` và `verify/` có **hơn 150 thư mục `sNNN`** —
> mỗi thư mục là một "version API" Facebook (FBAV cố định). Đa số là synthetic (kế thừa
> doc_id/bloks_ver từ một version "captured traffic" gần nhất). Xem hằng số + comment trong
> [factory.go:20-177](../../internal/facebook/factory.go).

---

## 3. Vòng đời ứng dụng (App lifecycle)

```
main() ([main.go:43])
  ├─ flag.Parse()  (-minimized)
  ├─ os.Chdir(appDataDir())          # CWD = bin/dev (dev) | thư mục exe (prod)  [datadir.go]
  ├─ expandEphemeralPortRange()      # netsh mở rộng port range (giảm WSAEADDRINUSE)
  ├─ app := NewApp()                 # [app.go:402] khởi tạo App{} (accounts rỗng, uploadCh)
  └─ wails.Run(&options.App{...})    # [main.go:66]
        ├─ SingleInstanceLock        # 1 instance PER THƯ MỤC (hash đường dẫn exe → UniqueId)
        ├─ Frameless window 1440x900, dark bg #0f1117
        ├─ OnStartup: app.startup    # [app.go:739]  ← seed config + warm pools
        ├─ OnBeforeClose             # chặn tắt nhầm → emit "app:request-quit-confirm"
        └─ Bind: [app]               # → window.go.main.App.* (generated bindings)
```

### 3.1 `App.startup(ctx)` — seed & warm ([app.go:739-881](../../app.go))

Đánh số các bước thực thi:

1. `a.ctx = ctx` — lưu context Wails (dùng cho mọi `EventsEmit` sau này).
2. **GC tuning 24/7:** `debug.SetGCPercent(50)` + `SetMemoryLimit` (auto 25% RAM, cap 4GB).
   Goroutine `FreeOSMemory` mỗi 5 phút (trả pages về OS).
3. `setupLogging()` — slog ghi vào `%APPDATA%/HVR/logs/`.
4. `appsettings.Load("Config/Settings")` → `a.appSettings` (auto-migrate từ
   general.json/interaction.json nếu chưa có app_settings.json).
5. `cookie.EnsureDir()` + `cookie.SeedInitialIfMissing()` — ship sẵn pool datr embedded.
6. `fbdata.EnsureSplitFiles()` + `fbdata.Reload("")` — load FBAV version/build pool.
7. `fakeinfo.ReloadUAPools()` + `SeedConfigDataIfMissing` + `LoadPhoneDatabase` +
   `ReloadOverrides` — warm các pool device/UA/phone/locale.
8. `go a.runMemoryMaintenance(ctx)` — watchdog RAM (GC 1ph, watchdog 20s,
   auto-cleanup khi RSS ≥ 1500MB, [app.go:892-1068](../../app.go)).
9. `go` periodic UI cleanup mỗi 12h (emit `system:ui-reload`).
10. `go` **auto-reload accounts** nếu session trước ở "file" mode (tránh grid trống sau restart).

### 3.2 App struct — trạng thái toàn cục ([app.go:285-366](../../app.go))

Các nhóm field quan trọng:

| Nhóm | Field | Vai trò |
|------|-------|---------|
| Context | `ctx context.Context` | context Wails — dùng cho mọi EventsEmit |
| Accounts | `accounts []Account` + `accountsMu sync.RWMutex` | grid accounts (RWMutex vì ListAccounts poll read-heavy) |
| Verify run | `isRunning bool`, `verifyMu`, `verifyCancel`, `verifyWorkerCancel`, `verifyStopping atomic.Bool` | trạng thái + 2 context (dispatch vs worker HTTP) |
| Register run | `registerState atomic.Int32` (idle/running/stopping), `registerMu`, `registerCancel`, `registerGen` | state machine gating Start |
| Email | `emailPool *email.CredPool` | shared pool mua batch email — 1 instance/run, dùng chung reg+verify |
| Proxy | `sharedProxyMgr` (verify), `regProxyMgr` (register), `proxyConfigVersion atomic.Int64` | cache manager, version để invalidate khi đổi settings |
| Stats | `regStats`, `verifyStats`, `mailDomainStats`, `verifyPlatformRR atomic.Int64` | thống kê theo version + counter round-robin multi-version |
| Upload | `uploadCh chan string`, `uploadSiteCancel`, `uploadRetryQueue` | đẩy account live lên site |
| Misc | `confirmedQuit atomic.Bool`, `frontendHidden atomic.Bool`, `sourceFilePath` | confirm quit, throttle IPC khi hidden, file mode |

> **Gotcha — verify ↔ register loại trừ nhau:** cả hai dùng chung `emailPool`, nên
> `RunVerify` reject nếu register đang chạy/dừng và ngược lại
> ([app.go:2283-2288](../../app.go), [app.go:7366-7371](../../app.go)).

---

## 4. Sơ đồ tổng — từ click "Chạy" đến kết quả

```
┌─ FRONTEND (Vue) ───────────────────────────────────────────────────────────┐
│ User bấm "Chạy Verify"                                                       │
│   AccountsPage → bridge getVerifyRunnerService().run(cfg)                    │
│   → verify-runner.wails.ts → RunVerify(main.VerifyRunConfig)                  │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                        │ Wails IPC (JSON)
┌─ BACKEND (Go) ────────────────────────▼──────────────────────────────────────┐
│ App.RunVerify(cfgOverride)  [app.go:2280]                                     │
│   1. Gate: register idle? + !isRunning  → set isRunning=true                  │
│   2. LoadSettings() + LoadInteractionConfig()                                 │
│   3. VALIDATION: nguồn account / mail provider / output / proxy               │
│   4. Tạo emailPool (rent) + proxy StickyManager + RunConfig                   │
│   5. Chọn MODE dispatch (file-select / file-stream / CloneHV)                 │
│        └─► runner.RunVerify(ctx, accounts, runCfg, onStatus)  [scheduler.go]  │
│                                                                               │
│ runner.RunVerify  [scheduler.go:150]                                          │
│   • workCh ← tất cả accounts (FIFO)                                           │
│   • spawn N=maxThreads goroutine (worker pool)                                │
│       each worker → runOneAccount(...)  [scheduler.go:410]                    │
│         a. AcquireProxy(workerID)  (sticky per slot)                          │
│         b. resolve verifyPlatform (round-robin nextVerifyPlatform)            │
│         c. set UA theo platform/country                                       │
│         d. OUTER loop (mail retry) × INNER loop (network retry):              │
│              - login (Web: cookie; Android: fetch EAA; iOS: token EAAAAAY)    │
│              - ver := facebook.NewVerifier(platform)                          │
│              - ver.Verify(ctx, session, cfg, ...) ──┐                         │
│         e. CheckLiveDieByPicture nếu vẫn Unknown    │                         │
│         f. AppendDatr(cookie) → datr_pool.txt       │                         │
│         g. config.OnAccountDone(...) ───────────────┼──► emit verify:account-done
│   • RetryUnknownNow pass-2 (proxy mới) nếu bật      │                         │
│                                                     ▼                         │
│ internal/facebook/verify/<platform>/  (steps.go) ── CreateEmail → AddEmail    │
│   → poll OTP (internal/email) → ConfirmCode → check Live/Die                  │
│                                                                               │
│ Callbacks (per account, realtime):                                            │
│   onStatus(id, uid, msg)  → EventsEmit "verify:status" / "verify:batch-status"│
│   OnRawProxy/OnProxy/OnEmailCreated → "verify:raw-proxy"/"verify:proxy"/"...:email"
│   OnAccountDone → cập nhật a.accounts + saveVerifyOutcome (ghi file) +         │
│                   EventsEmit "verify:account-done"                            │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                        │ EventsEmit (Go → JS)
┌─ FRONTEND ────────────────────────────▼──────────────────────────────────────┐
│ EventsOn("verify:status" / "verify:account-done" / ...) → cập nhật grid       │
│ EventsOn("verify:complete") → reset nút Run/Stop                              │
└───────────────────────────────────────────────────────────────────────────────┘
        +  ghi file kết quả: result/Writer → Live/Die/Unknown/SuccessVerify*.txt
```

Register có sơ đồ tương tự nhưng dispatch nằm **inline trong app.go** (không qua
`runner.RunVerify`) — xem §5.

---

## 5. Các "mode" chạy

| Mode | Mô tả | Entrypoint / cờ |
|------|-------|-----------------|
| **Register-only** | Chỉ tạo account, không verify. `VerifyEnabled=false`. | [`App.RunRegister`](../../app.go) — `interactionCfg.VerifyEnabled == false` |
| **Verify-only** | Verify account có sẵn (đọc từ file/folder/grid). | [`App.RunVerify`](../../app.go) |
| **Reg + Verify (non-split)** | Worker reg xong → **verify inline ngay trong cùng worker**. | `RunRegister` + `VerifyEnabled=true && SplitMode=false` ([app.go:9699](../../app.go)) |
| **Reg + Verify (SPLIT)** | Reg worker chỉ reg → đẩy account vào `splitVerifyCh` → VER pool riêng verify async. UI hiển thị 2 panel REG/VER. | `RunRegister` + `SplitMode && VerifyEnabled` ([app.go:8133](../../app.go), [9666](../../app.go)) |
| **CloneHV** | Mua account "clone" sẵn từ API CloneHV → verify liên tục theo pool. | `RunVerify` khi `AccountSource=="api"` + đủ creds ([app.go:2320](../../app.go), [3228](../../app.go)) |

### 5.1 Nguồn account của Verify (3 nhánh trong `RunVerify`)

1. **File-select mode** — user pick 1 file `.txt`, load vào grid, tick chọn rows →
   build `[]runner.AccountInput` từ `a.accounts` rồi gọi `runner.RunVerify` 1 lần
   ([app.go:3560-3618](../../app.go)).
2. **File-streaming mode** (mặc định folder) — tạo N slot rows, stream đọc từng account
   từ folder, replenish khi worker rảnh ([app.go:3621-3760](../../app.go),
   `isStreamingFileMode = true`).
3. **CloneHV pool mode** — vòng lặp mua bulk account từ CloneHV, gán vào slot, gọi
   `runner.RunOneAccountAt(slotID)` ([app.go:3228-3760](../../app.go)).

### 5.2 SPLIT vs non-split (Reg + Verify)

- **non-split** ([app.go:9699-9752+](../../app.go)): trong worker reg, sau khi
  `reg.Register` success → build `runner.AccountInput` → chờ `DelayVeriReg` →
  tạo `facebook.VerifyConfig` → `ver.Verify(...)` ngay tại đây. Reg slot bị giữ
  đến khi verify xong.
- **split** ([app.go:8133](../../app.go), [9666-9698](../../app.go)): reg worker đẩy
  `splitVerifyJob{acc,...}` vào `splitVerifyCh` (bounded → backpressure) rồi giải
  phóng reg slot ngay. Một VER pool riêng đọc channel và verify. Hai panel UI tách biệt
  (REG events vs `verify:batch-status`).

> **Pre-fetch token EAA:** trước cả 2 nhánh, nếu reg success mà verify platform là
> Android-family cần `EAAAAU` và account chưa có → worker gọi
> `webregister.FetchAndroidTokenLegacy` (REST `/auth/login`) để lấy token trước khi ghi
> file/verify ([app.go:9501-9548](../../app.go)). iOS (`EAAAAAY`) login trong verify, không pre-fetch ở reg.

---

## 6. Plugin registry — cách platform được nối vào

[factory.go](../../internal/facebook/factory.go) giữ các `map[string]factory`. Mỗi
platform package tự đăng ký trong `init()`, **app.go blank-import** để trigger:

```go
// app.go:57-108 — blank import kích hoạt init() registration
_ "HVR/internal/facebook/verify/s562"
_ "HVR/internal/facebook/verify/ios562"
// register/<platform>/<file>.go  → init():
//   facebook.RegisterPlatformRegisterer("s562", func() facebook.Registerer { ... })
//   facebook.RegisterPlatformVerifier("s562",   func() facebook.Verifier   { ... })
//   facebook.RegisterPlatformVerifyUA("s562",   func(cc string) string      { ... })
```

Runtime lookup:
- `facebook.NewRegisterer(platform)` → `Registerer` ([factory.go:251](../../internal/facebook/factory.go)).
- `facebook.NewVerifier(platform)` → `Verifier` ([factory.go:263](../../internal/facebook/factory.go)).
- `facebook.PlatformVerifyUA(platform, cc)` → UA string ([factory.go:236](../../internal/facebook/factory.go)).

Hợp đồng interface ([interfaces.go](../../internal/facebook/interfaces.go)):

```go
type Registerer interface {
    Register(ctx, input *RegInput, onStatus func(string)) *RegResult
}
type Verifier interface {
    Verify(ctx, session *Session, cfg *VerifyConfig, outputPath string,
           onStatus func(uid, msg string)) *VerifyResult
}
```

> **Tránh circular import:** package `facebook` KHÔNG import `facebook/<platform>`.
> Chiều phụ thuộc luôn là `platform → facebook` (qua `RegisterPlatformXxx`). Thêm version
> mới = tạo thư mục + implement + đăng ký init() + blank-import — KHÔNG sửa interface.
> Xem [add-facebook-reg-version.md §3-4](./add-facebook-reg-version.md).

---

## 7. Worker pool scheduler ([scheduler.go](../../internal/runner/scheduler.go))

`runner.RunVerify` ([scheduler.go:150](../../internal/runner/scheduler.go)) là FIFO
worker pool dùng cho verify (và CloneHV qua wrapper `RunOneAccountAt`):

1. `maxThreads` clamp `[1..500]`.
2. `workCh` buffered channel ← tất cả accounts; `close(workCh)`.
3. Spawn `maxThreads` goroutine; mỗi worker `for work := range workCh` → `runOneAccount`.
4. `workerID = 0..maxThreads-1` ổn định → sticky proxy pin theo slot (KeepIPSuccess).
5. 2 context: `ctx` (dispatch — cancel khi Stop) vs `WorkerCtx` (HTTP — chạy hết bước hiện tại).
6. **RetryUnknownNow** pass-2: re-queue account Unknown/Error với workerID offset
   (`+maxThreads`) để force proxy mới ([scheduler.go:244-301](../../internal/runner/scheduler.go)).

`runOneAccount` ([scheduler.go:410](../../internal/runner/scheduler.go)) — **retry 2 tầng**:

- **Outer loop** (`maxOuterAttempts`, mặc định 2 + `AddMailRetry`): reload
  `GetVerifyConfig()` (đổi mail provider mid-run), reset tokens → thử với email mới.
- **Inner loop** (`maxAttempts=2`): retry login+verify khi lỗi mạng tạm thời.
- Phân loại lỗi terminal vs retryable: `isCookieDead` / `isTokenDead` / `isNetworkError` /
  `isOTPError` / `isBloksCheckpoint` ([scheduler.go:306-378](../../internal/runner/scheduler.go)).
- **switch verifyPlatform** ([scheduler.go:606-744](../../internal/runner/scheduler.go))
  quyết định login: Android-family → fetch `EAAAAU` REST; iOS → delegate (token trong verify);
  WebAndroid → cookie trực tiếp; Web → `LoginWithCookieMobile` parse fb_dtsg;
  `default` → **FATAL** (bắt lỗi quên add platform vào switch).

---

## 8. Danh sách method App quan trọng (Wails-bound)

> Tất cả ở [app.go](../../app.go) (trừ ghi chú khác). Tên method = tên binding JS
> `window.go.main.App.<Method>`.

### 8.1 Chạy / dừng
| Method | Vai trò |
|--------|---------|
| `RunRegister(maxThreads int) string` ([7364](../../app.go)) | Bắt đầu register hàng loạt (đọc profile từ InteractionConfig). Trả message lỗi/OK. |
| `RunVerify(cfg VerifyRunConfig) string` ([2280](../../app.go)) | Bắt đầu verify (file/folder/CloneHV). |
| `StopRegister() string` ([10899](../../app.go)) | Cancel dispatch register; worker đang chạy hoàn tất nốt. |
| `StopVerify() string` ([3861](../../app.go)) | Cancel dispatch verify; set `verifyStopping`. |
| `IsRegisterRunning/IsRegisterStopping` ([3933](../../app.go),[3938](../../app.go)) | Đọc state machine register. |
| `IsVerifyRunning/IsVerifyStopping` ([3882](../../app.go),[3890](../../app.go)) | Đọc trạng thái verify. |
| `GetRunStatus() map[string]bool` ([3944](../../app.go)) | Bundle 4 trạng thái 1 lần (restore UI sau reload). |

### 8.2 Accounts / file
| Method | Vai trò |
|--------|---------|
| `ListAccounts(filter) AccountListResult` ([1674](../../app.go)) | Trả accounts cho grid (poll). |
| `GetAccount(id) (*Account, error)` ([1715](../../app.go)) | Chi tiết 1 account. |
| `LoadAccountsFromFile(path) ImportResult` ([1736](../../app.go)) | File mode. |
| `ImportAccounts(data) ImportResult` ([1776](../../app.go)) | Import paste. |
| `DeleteAccounts(ids) DeleteResult` ([1993](../../app.go)) | Xóa rows. |
| `SetAccountSourceFolder / RefreshAccountSource` ([1369](../../app.go),[1425](../../app.go)) | Folder mode. |

### 8.3 Settings / config / profile
| Method | Vai trò |
|--------|---------|
| `LoadSettings / SaveSettings` ([4188](../../app.go),[4116](../../app.go)) | Cài đặt chung (general). |
| `LoadInteractionConfig / SaveInteractionConfig` ([6019](../../app.go),[5963](../../app.go)) | Thiết lập chạy (verify/reg threads, mail provider, split...). |
| `LoadProxyList / SaveProxyList(kind)` ([6347](../../app.go),[6370](../../app.go)) | Quản lý danh sách proxy. |
| `ListProfiles / SetActiveProfile / CreateProfile / CloneProfile / ...` ([6792](../../app.go)+) | Đa profile cấu hình. |
| `ImportLegacyConfig / ParseLegacyConfig` ([6735](../../app.go),[6259](../../app.go)) | Migrate config WeBM (C#) cũ. |

### 8.4 Pool / version / UA (diagnostic)
| Method | Vai trò |
|--------|---------|
| `GetCookieInitialStatus(path)` ([6524](../../app.go)) | Đếm số datr trong file cookie_initial. |
| `GetDatrPoolSize() int` ([6553](../../app.go)) | Size in-memory datr pool (`androidreg.SharedPool`). |
| `GetPoolFileSaveCount() int` ([6563](../../app.go)) | Số datr đã ghi file trong run. |
| `GetFbAppStatus / ReloadFbAppVersions / SaveFbAppVersions` ([6595](../../app.go)+) | FBAV version pool. |
| `GetUAPoolsStatus / SaveUAPool / GetDefaultUAContent` ([6635](../../app.go)+) | UA pool quản lý. |
| `SimulatePlatformUA(platform, cfg) string` ([11532](../../app.go)) | Preview UA platform sẽ sinh (mock FE). |
| `GetFbVersionMap / ExportFbVersionPool` ([11495](../../app.go),[11425](../../app.go)) | Map version → FBAV. |

### 8.5 Stats / resource / dialog / misc
| Method | Vai trò |
|--------|---------|
| `GetRegStats / GetVerifyStats / GetMailDomainStats` ([7074](../../app.go)+) | Bảng thống kê theo version/domain. |
| `GetResourceUsage() AppResourceUsage` ([3975](../../app.go)) | RAM/CPU/goroutine cho status bar. |
| `DebugMemory / ForceMemoryCleanup` ([1076](../../app.go),[1165](../../app.go)) | Debug RAM. |
| `OpenTextFileDialog / OpenFolderDialog / OpenFolderInExplorer` ([1517](../../app.go)+) | Native dialog. |
| `NotifyVisibilityChange(hidden bool)` ([3897](../../app.go)) | Throttle IPC khi minimize. |
| `RequestQuit / IsConfirmedQuit / EmitQuitConfirm` ([3905](../../app.go)+) | Flow xác nhận tắt app. |
| `GetAppVersion() string` ([1411](../../app.go)) | Version inject từ ldflags. |
| Upload site: `StartUploadSite / StopUploadSite / GetUploadStats / ...` ([5039](../../app.go)+) | Đẩy account live lên site. |

---

## 9. Danh sách event (Go → JS)

Frontend lắng nghe qua bridge event bus. Các event chính:

| Event | Phát ở | Ý nghĩa |
|-------|--------|---------|
| `verify:status` | mọi notify system-level | log dòng đơn (accountId=0 = system). |
| `verify:batch-status` | dispatch loop | batch update nhiều row 1 lần (throttle khi hidden). |
| `verify:account-done` | OnAccountDone | 1 account verify xong (status/email/token/cookie). |
| `verify:raw-proxy` / `verify:proxy` / `verify:email` | callback runner | cập nhật cột PROXY / IP CHẠY / EMAIL realtime. |
| `verify:complete` | defer cuối RunVerify | reset nút Run/Stop. |
| `verify:accounts-updated` / `verify:output-path` | setup slot/output | refresh grid + hiển thị đường dẫn. |
| `register:status` / `register:token` / `register:account-done` / `register:complete` | RunRegister | tương tự cho register. |
| `system:ui-reload` / `system:memory-warning` | startup goroutines | cleanup UI / cảnh báo RAM. |
| `app:request-quit-confirm` | OnBeforeClose | show dialog xác nhận tắt. |
| `accounts:folder-updated` | auto-reload / refresh | grid refresh. |

---

## 10. Dữ liệu & thư mục runtime

- **CWD runtime** = `appDataDir()` ([datadir.go](../../datadir.go)): `bin/dev/` (dev, khi
  `AppVersion=="dev"`) hoặc thư mục chứa exe (prod). Override: env `HVR_DATA_DIR`.
- **Config seed** ở `build/bin/Config/` (template) → copy sang data dir. Gồm: `Cookie/`
  (datr pool + cookie_initial), `DeviceInfo*/`, `Fbapp/` (FBAV), `Proxy/`, `Settings/`
  (app_settings.json, interaction.json), `Namereg/`, `Locales/`, `SimNetwork/`,
  `PhoneDatabase/`, `Permanent/` (phone.txt/mail.txt tích lũy), `RentMail/`, `TempMail/`.
- **Kết quả** ghi qua [result.Writer](../../internal/result/writer.go) (UpsertUID dedupe):
  `Live.txt`, `Die.txt`, `Unknown.txt`, `SuccessVerify*.txt`, `SuccessReg_*.txt`. Một thư mục
  `VerifyAccount<timestamp>` / `RegAccount<timestamp>` per run.

---

## 11. Edge case & gotcha cần nhớ

1. **Verify ↔ Register loại trừ** — dùng chung `emailPool`; Start cái này khi cái kia
   running/stopping sẽ bị reject ([app.go:2283](../../app.go),[7377](../../app.go)).
2. **Stop = soft stop** — chỉ cancel dispatch context (poolCtx), worker dùng `WorkerCtx`
   nên chạy hết bước HTTP hiện tại; UI hiện "Đang dừng — chờ..." ([scheduler.go:198-207](../../internal/runner/scheduler.go)).
3. **Token đúng loại theo platform** (đối xứng) — Android verify cần `EAAAAU` (gặp
   `EAAAAAY` thì bỏ đi login lại); iOS verify cần `EAAAAAY` (gặp `EAAAAU` thì login iOS).
   Xem [scheduler.go:653-693](../../internal/runner/scheduler.go) +
   [add-facebook-reg-version.md §13.6/§13.7](./add-facebook-reg-version.md).
4. **`default` case scheduler = FATAL** — quên add platform Android-family vào switch sẽ
   bị bắt ngay thay vì âm thầm verify bằng cookie sai ([scheduler.go:733-743](../../internal/runner/scheduler.go)).
5. **Multi-version round-robin** — `nextVerifyPlatform()` chia version verify theo từng
   account qua `verifyPlatformRR atomic` ([app.go:7349](../../app.go)). UA phải khớp FBAV
   của version → dùng `PlatformVerifyUA` thay vì pool chung.
6. **Sticky proxy per slot** — `KeepIPSuccess` giữ proxy cho account kế nếu account này
   Live; `release(success)` dựa trên `result.Status == "Live"` ([scheduler.go:454-456](../../internal/runner/scheduler.go)).
7. **Datr lưu mọi outcome** — sau verify, nếu cookie chứa datr thì `AppendDatr` (chỉ ghi
   file, không add pool vì datr verify có thể đã bị flag) ([scheduler.go:851-864](../../internal/runner/scheduler.go)).
8. **Một số bridge service vẫn mock trên Wails** (flows/proxies) — chưa wire binding.

---

## 12. Bản đồ đọc tiếp

| Cần hiểu | Đọc file |
|----------|----------|
| Thêm version reg/verify mới (runbook) + luật token/login iOS | [add-facebook-reg-version.md](./add-facebook-reg-version.md) |
| Register flow chi tiết (B1-B8, Bloks, encrypt password) | `01-*` (sẽ viết) + [register/](../../internal/facebook/register) |
| Verify flow chi tiết (CreateEmail → AddEmail → OTP → ConfirmCode) | `02-*` + [verify/verifybase/](../../internal/facebook/verify/verifybase) |
| Sinh device/UA/SIM/phone | [fakeinfo/](../../internal/facebook/fakeinfo) |
| Cookie / datr pool | [cookie/](../../internal/cookie) |
| Email providers (rent + temp + OTP reader) | [email/](../../internal/email) |
| Proxy manager / sticky / providers | [proxy/](../../internal/proxy) |
| Ghi file kết quả + counters | [result/](../../internal/result) |

---

*Tài liệu mô tả flow chạy thực tế tại thời điểm khảo sát code (read-only). File:line có thể
trôi nhẹ sau commit — dùng tên hàm/symbol để định vị lại nếu lệch.*
