# 04 — Frontend UI & Luồng Event Realtime (backend ↔ frontend)

> Tài liệu mô tả CHI TIẾT lớp frontend của HVR và **luồng event realtime** giữa Go backend
> ([app.go](../../app.go)) và Vue frontend. Đây là phần "máu" của app: register/verify chạy ở Go,
> kết quả nhỏ giọt về UI qua `runtime.EventsEmit` → `EventsOn`. Hiểu sai luồng này = bảng REG/VER
> nhấp nháy sai, mất token/cookie, đếm sai Live/Die.
>
> Liên quan:
> - Runbook thêm version + luật token/login iOS: [add-facebook-reg-version.md](./add-facebook-reg-version.md)
>   (đặc biệt **§13.6** token iOS `EAAAAAY` và **§13.7** login-at-verify + token/cookie realtime — KHÔNG lặp lại ở đây).
> - Scheduler verify/register: [scheduler.go](../../internal/runner/scheduler.go).

---

## 0. Sơ đồ tổng — Event flow end-to-end

```
┌─────────────────────────── GO BACKEND (app.go) ───────────────────────────┐
│                                                                            │
│  RunRegister()  ── reg worker ──▶ register:status / register:batch-status  │
│       │                           register:token  (reg success, có UID)    │
│       │                           register:account-done (reg + verify xong)│
│       │  (inline verify / split)  verify:slot-assigned / verify:email      │
│       │                           verify:raw-proxy / verify:proxy          │
│       │                           verify:batch-status / verify:account-done│
│       └────────────────────────▶  register:complete / register:auto-restart│
│                                                                            │
│  RunVerify()    ── ver worker ──▶ verify:slot-assigned / verify:email      │
│       │                           verify:raw-proxy / verify:proxy          │
│       │                           verify:batch-status / verify:account-done│
│       │                           verify:accounts-updated (slot init)      │
│       └────────────────────────▶  verify:complete                          │
│                                                                            │
│  UploadSite   ─────────────────▶ upload-site:log / :stopped / :log-cleared │
│  Watchdog     ─────────────────▶ system:memory-warning / system:ui-reload  │
│  OnBeforeClose ────────────────▶ app:request-quit-confirm                  │
│  Folder watch ─────────────────▶ accounts:folder-updated                   │
│  Mail pool    ─────────────────▶ email:pool-exhausted                      │
│                                                                            │
│   runtime.EventsEmit(a.ctx, "<event>", payload)                            │
└────────────────────────────────────┬───────────────────────────────────────┘
                                      │  Wails IPC (WebView2)
                                      ▼
┌──────────────────────── VUE FRONTEND (frontend/src) ───────────────────────┐
│  bridge/wails/event-bus.wails.ts  →  EventsOn(name, cb) → trả unsub fn      │
│  bridge/client.ts  getEventBusService() → wails | mock (no-op)             │
│                                                                            │
│  AccountsPage.vue                                                          │
│    setupRegisterListeners() → registerThreads (REG pane, RAF batch)        │
│    setupVerifyListeners()   → accountsStore (VER pane)                     │
│    _alwaysOnUnsubs          → verify:email, email:pool-exhausted, folder   │
│                                                                            │
│  useAccountsStore.ts  applySlotAssigned / setRunProxy / ensureSlot ...     │
│  App.vue              system:memory-warning, system:ui-reload              │
│  AppTitleBar.vue      app:request-quit-confirm                             │
│  uploadLog.store.ts   upload-site:* (đăng ký qua window.runtime.EventsOn)  │
└────────────────────────────────────────────────────────────────────────────┘
```

**3 nhóm listener trong [AccountsPage.vue](../../frontend/src/pages/AccountsPage.vue)** (lưu unsub fn để cleanup chính xác, không `bus.off` global — xem §2.3):
- `_registerUnsubs` — đăng ký trong `setupRegisterListeners()` ([AccountsPage.vue:1561](../../frontend/src/pages/AccountsPage.vue)).
- `_verifyUnsubs` — đăng ký trong `setupVerifyListeners()` ([AccountsPage.vue:1245](../../frontend/src/pages/AccountsPage.vue)).
- `_alwaysOnUnsubs` — đăng ký trong `onMounted` ([AccountsPage.vue:572-624](../../frontend/src/pages/AccountsPage.vue)): các event cần fire ở MỌI chế độ (`verify:email`, `email:pool-exhausted`, `accounts:folder-updated`).

---

## 1. Cấu trúc `frontend/src`

| Thư mục/file | Vai trò |
|---|---|
| [app/router/routes.ts](../../frontend/src/app/router/routes.ts) | Khai báo route. `/` redirect → `/accounts`. Mọi page lazy-load (`() => import(...)`). |
| [app/App.vue](../../frontend/src/app/App.vue) | Root component. Lắng nghe `system:memory-warning`, `system:ui-reload`; phát `app:soft-cleanup` (window event nội bộ); gọi `NotifyVisibilityChange` cho backend throttle batch. |
| [layouts/AppLayout.vue](../../frontend/src/layouts/AppLayout.vue) | Shell desktop: title bar + sidebar + content + status bar. |
| [components/shell/AppTitleBar.vue](../../frontend/src/components/shell/AppTitleBar.vue) | Frameless title bar; lắng nghe `app:request-quit-confirm`. |
| **[pages/AccountsPage.vue](../../frontend/src/pages/AccountsPage.vue)** | **Trang chính**: grid REG/VER, toolbar, ~2560 dòng — chứa TOÀN BỘ listener register/verify. |
| [pages/InteractionSetupPage.vue](../../frontend/src/pages/InteractionSetupPage.vue) | Cấu hình "Thiết lập chạy" (verify platform, mail, UA pool...). |
| [pages/ProxySettingsPage.vue](../../frontend/src/pages/ProxySettingsPage.vue) | Cấu hình proxy. |
| [modules/accounts/store/useAccountsStore.ts](../../frontend/src/modules/accounts/store/useAccountsStore.ts) | Pinia store `accounts`: `accounts[]` + `accountsIndex` (Map O(1)), proxy cache, `applySlotAssigned`, `ensureSlot`. |
| [bridge/contracts.ts](../../frontend/src/bridge/contracts.ts) | Interface bridge (Account, IVerifyRunnerService, IEventBusService...). |
| [bridge/client.ts](../../frontend/src/bridge/client.ts) | Factory: detect `window.go.main` → trả `wails` impl, fallback `mock`. |
| [bridge/wails/event-bus.wails.ts](../../frontend/src/bridge/wails/event-bus.wails.ts) | Wrap `EventsOn`/`EventsOff`. |
| [bridge/mock/event-bus.mock.ts](../../frontend/src/bridge/mock/event-bus.mock.ts) | No-op: event KHÔNG fire ở mock mode (không có backend). |
| [components/grid/DataGrid.vue](../../frontend/src/components/grid/DataGrid.vue) | Grid foundation: virtual scroll, sticky header, selection, cell-select. |
| [stores/uploadLog.store.ts](../../frontend/src/stores/uploadLog.store.ts) | Store log upload site; lắng nghe `upload-site:*`. |

### 1.1 Bridge layer & cơ chế mock ↔ wails

`getEventBusService()` ([client.ts:135](../../frontend/src/bridge/client.ts)) → `waitForWails()` retry detect `window.go.main` tối đa 2s (10×200ms) vì Wails inject muộn hơn Vue mount. Khi có → `eventBusWails`, không có → `eventBusMock`.

```ts
// event-bus.wails.ts:8 — on() TRẢ VỀ unsub fn (selective per-listener)
on(event, callback): () => void {
  const unsub = EventsOn(event, callback)
  if (typeof unsub === 'function') return unsub
  return () => EventsOff(event)   // fallback Wails cũ
}
```

> **Gotcha mock mode**: [event-bus.mock.ts](../../frontend/src/bridge/mock/event-bus.mock.ts) `on()` trả no-op. Nên khi chạy `npm run dev` (không có Go) **không event realtime nào fire** — grid chỉ có mock account tĩnh. Đây là chủ đích (giai đoạn frontend foundation), không phải bug.

---

## 2. Cơ chế nền của lớp event

### 2.1 Backend batch + throttle (giảm flood IPC)

Backend KHÔNG emit từng message thô — nó gom lại để tránh flood WebView2 IPC:

- **`verify:batch-status`** — goroutine ticker gom `activityCache` (map `accountID→message`) thành 1 event mỗi **300ms** khi window visible, **2s** khi hidden ([app.go:3168-3221](../../app.go)). Chỉ gửi entry **thực sự đổi** (`sentCache` dirty-tracking) → giảm payload 80-95% khi idle.
  - `onStatus(accountID, uid, message)` chỉ ghi `a.activityCache.Store(id, msg)` lock-free ([app.go:3223-3226](../../app.go)); goroutine flush mới emit.
  - Visibility do `App.vue` báo qua `NotifyVisibilityChange(document.hidden)` ([App.vue:60-65](../../frontend/src/app/App.vue)) → backend set `a.frontendHidden`.
- **`register:batch-status`** — tương tự, ticker **200ms**, gom `regBatchCache` (key=threadIdx) → mảng updates, dirty-track theo `{phone,proxy,msg}` ([app.go:8546-8584](../../app.go)).

### 2.2 Frontend RAF batch (gộp re-render)

Frontend lại gom tiếp bằng `requestAnimationFrame` (~16ms) để 1 lần Vue re-render xử lý nhiều event — nếu không, 100 goroutine fail nhanh = 100 event/s làm JS engine kẹt, click không phản hồi:

- **Verify**: `verify:batch-status` push vào `_activityBuffer` (Map), `flushActivityBuffer()` qua RAF ([AccountsPage.vue:1263-1290](../../frontend/src/pages/AccountsPage.vue)).
- **Register**: `register:status` / `:token` / `:account-done` push vào `pendingStatus`/`pendingTokens`/`pendingDone`, flush thứ tự **status → token → done** trong `flushRegisterEvents()` ([AccountsPage.vue:1577-1690](../../frontend/src/pages/AccountsPage.vue)). Thứ tự QUAN TRỌNG: token flush sau status để entry đã tồn tại (nếu race, token tự tạo placeholder).

### 2.3 Cleanup listener chính xác (Task 6)

KHÔNG dùng `bus.off(eventName)` (global — xóa nhầm listener của store/component khác cùng nghe event đó). Thay vào đó lưu unsub fn:
```ts
let _verifyUnsubs: Array<() => void> = []
_verifyUnsubs.push(bus.on('verify:account-done', handler))   // lưu lại
// cleanup: clearUnsubs(_verifyUnsubs)  → gọi từng fn()
```
`clearUnsubs()` ([AccountsPage.vue:302-307](../../frontend/src/pages/AccountsPage.vue)) gọi mỗi fn rồi reset mảng. `setupXxxListeners()` luôn `clearUnsubs` trước khi đăng ký lại (re-register an toàn khi remount / restore sau UI reload). `onUnmounted` clear cả 3 nhóm ([AccountsPage.vue:800-804](../../frontend/src/pages/AccountsPage.vue)).

---

## 3. BẢNG TỔNG HỢP TẤT CẢ EVENT

| Event | Backend emit (app.go) | Payload chính | Frontend handler |
|---|---|---|---|
| `register:status` | [7023](../../app.go), [7450](../../app.go), [8586](../../app.go), [9534](../../app.go)... | `{index,phone,proxy,proxyServer,userAgent,msg}` hoặc `string` (RegisterFacebook đơn) | [1697](../../frontend/src/pages/AccountsPage.vue) → `pendingStatus` |
| `register:batch-status` | [8578](../../app.go) | `Array<{index,phone,proxy,proxyServer,userAgent,msg}>` | [1707](../../frontend/src/pages/AccountsPage.vue) → `registerThreads` |
| `register:token` | [9553](../../app.go) | `{index,uid,token,cookie}` | [1740](../../frontend/src/pages/AccountsPage.vue) → `pendingTokens` (+đếm success) |
| `register:account-done` | [10136](../../app.go) | `{index,phone,proxy,proxyServer,userAgent,success,uid,cookie,password,token,message,verifyStatus,verifyMessage,verifyEmail,checkpoint}` | [1732](../../frontend/src/pages/AccountsPage.vue) → `pendingDone` |
| `register:output-path` | [7640](../../app.go) | `{path}` | [1748](../../frontend/src/pages/AccountsPage.vue) → `activeRunOutputPath` |
| `register:complete` | [7458](../../app.go), [8688](../../app.go), [10906](../../app.go) | `{total, error?}` | [1752](../../frontend/src/pages/AccountsPage.vue) |
| `register:auto-restart-trigger` | [8072](../../app.go) | `{minutes}` | [1811](../../frontend/src/pages/AccountsPage.vue) |
| `verify:slot-assigned` | [3504](../../app.go), [8392](../../app.go), [10763](../../app.go) | `{slotId,uid,password,phone,status,userAgent,token,cookie}` | [1389](../../frontend/src/pages/AccountsPage.vue) → `applySlotAssigned` |
| `verify:email` | [2950](../../app.go), [8378/8413](../../app.go), [10653](../../app.go) | `{accountId,email}` | [617 (alwaysOn)](../../frontend/src/pages/AccountsPage.vue) |
| `verify:raw-proxy` | [2914](../../app.go), [8402](../../app.go), [10622](../../app.go) | `{accountId,proxy}` (full ip:port:user:pass) | [1291](../../frontend/src/pages/AccountsPage.vue) → `setDisplayProxy` |
| `verify:proxy` | [2933](../../app.go), [8407](../../app.go), [10639](../../app.go) | `{accountId,proxy}` (IP rút gọn + country) | [1294](../../frontend/src/pages/AccountsPage.vue) → `setRunProxy` |
| `verify:batch-status` | [3043/3092/3215](../../app.go), [8386](../../app.go) | `Array<{accountId,message}>` | [1276](../../frontend/src/pages/AccountsPage.vue) → `_activityBuffer` |
| `verify:account-done` | [3012](../../app.go), [8441/8520](../../app.go), [10043](../../app.go) | `{accountId,uid,status,message,token,cookie}` | [1299](../../frontend/src/pages/AccountsPage.vue) |
| `verify:output-path` | [2772](../../app.go) | `{path}` | [1255](../../frontend/src/pages/AccountsPage.vue) → `activeRunOutputPath` |
| `verify:accounts-updated` | [3268/3660](../../app.go), [10306](../../app.go) | `nil` | [1384](../../frontend/src/pages/AccountsPage.vue) → `fetchAccounts()` |
| `verify:complete` | [3362/3592/3613/3686](../../app.go) | `{total}` / `{error}` / `string` | [1358](../../frontend/src/pages/AccountsPage.vue) |
| `verify:status` | [2484/2525/2656](../../app.go), [3246/3597/3629](../../app.go) | `{accountId,uid,message}` (system msg, accountId=0) | **KHÔNG có handler** ⚠️ §6.1 |
| `split-verify:drained` | [10840](../../app.go) | `nil` | **KHÔNG có handler** ⚠️ §6.2 |
| `accounts:folder-updated` | [876/1357/1768](../../app.go) | `{imported, source}` | [572 (alwaysOn)](../../frontend/src/pages/AccountsPage.vue) |
| `email:pool-exhausted` | [2571](../../app.go), [8003](../../app.go) | `{provider, error}` | [604 (alwaysOn)](../../frontend/src/pages/AccountsPage.vue) |
| `upload-site:log` | [4972](../../app.go) | `{msg,uploaded,level}` | [uploadLog.store:127](../../frontend/src/stores/uploadLog.store.ts) |
| `upload-site:stopped` | [5197](../../app.go) | `nil` | [uploadLog.store:136](../../frontend/src/stores/uploadLog.store.ts) |
| `upload-site:log-cleared` | [5019/5031](../../app.go) | `nil` | [uploadLog.store:144](../../frontend/src/stores/uploadLog.store.ts) |
| `system:memory-warning` | [1033](../../app.go) | `{heapMB,msg}` | [App.vue:70](../../frontend/src/app/App.vue) |
| `system:ui-reload` | [850](../../app.go) | `nil` | [App.vue:71](../../frontend/src/app/App.vue) |
| `app:request-quit-confirm` | [3924](../../app.go) | `{registerRunning,verifyRunning}` | [AppTitleBar.vue:44](../../frontend/src/components/shell/AppTitleBar.vue) |

---

## 4. LUỒNG VERIFY (chạy verify riêng — `RunVerify`)

**Mục tiêu**: verify list account đã chọn / file / CloneHV pool, cập nhật grid realtime.
**Input**: tick account trong grid → `handleRun()` → `runner.run(config)` → backend `RunVerify`.

### 4.1 Cấu trúc grid VERIFY (normal mode)

Grid dùng `accountsStore.accounts` (mỗi slot = 1 row trong `a.accounts` backend). Cột chính ([columns.ts](../../frontend/src/constants/columns.ts)):

| Cột (key) | Nguồn realtime |
|---|---|
| UID (`uid`) | `verify:slot-assigned` |
| Email (`email`) | `verify:email` (gộp phone) |
| Cookie (`cookie`) | `verify:slot-assigned` + `verify:account-done` (cookie mới sau login) |
| Token (`token`) | `verify:slot-assigned` + `verify:account-done` (qua `preferUserAccessToken`) |
| Trạng thái (`status`) | `verify:slot-assigned` (`new`) → `verify:account-done` (`live/die/...`) |
| Proxy (`proxy`) | `verify:raw-proxy` → `setDisplayProxy` |
| IP chạy (`runProxy`) | `verify:proxy` → `setRunProxy` |
| Hoạt động (`activity`) | `verify:batch-status` (RAF buffer) |

### 4.2 Trình tự event 1 account verify

```
1. RunVerify khởi tạo N slot rows (Status="waiting") → verify:accounts-updated (nil)
                                                          → FE fetchAccounts() (full refresh, 1 lần)
2. Worker lấy slotID rảnh → emit:
   verify:slot-assigned {slotId,uid,password,phone,status:"new",userAgent,token,cookie}
         → applySlotAssigned: set uid/pwd/phone/status/UA/token/cookie, RESET email/runProxy/activity
   verify:raw-proxy {accountId,proxy}   → setDisplayProxy (cột PROXY)
   verify:proxy     {accountId,proxy}   → setRunProxy     (cột IP CHẠY)
   verify:email     {accountId,email}   → acc.email       (cột EMAIL)
3. Trong lúc verify chạy (login, add mail, OTP...):
   verify:batch-status [{accountId,message}]  → _activityBuffer → flush RAF → acc.activity
   (BỎ QUA nếu acc.status đã FINAL — tránh ghi đè kết quả)
4. Verify xong:
   verify:account-done {accountId,uid,status,message,token,cookie}
         → acc.status, acc.activity=message, acc.token=preferUserAccessToken(...), acc.cookie
         → đếm verifyTotalLive/Die/Unknown (dedupe theo UID, §5.2)
         → uncheck account khỏi selection
         → (chỉ file-mode standalone) setTimeout 1.5s deleteAccounts([id]) auto-clear
5. Toàn bộ xong:
   verify:complete {total} → isVerifyRunning=false, auto-clear verifiedRunIds, fetchAccounts()
```

### 4.3 `applySlotAssigned` — cập nhật in-place (hot path)

[useAccountsStore.ts:220-258](../../frontend/src/modules/accounts/store/useAccountsStore.ts). Thay full `fetchAccounts()` bằng O(1) `accountsIndex.get(slotId)`:
- Set `uid/password/phone/status/userAgent/token/cookie`.
- **RESET** `email/runProxy/activity/noteRun` về `''` (slot tái dùng cho account mới — không để dữ liệu account cũ leak).
- **KHÔNG reset** `acc.proxy` về `''` — tránh "sort jump" khi user đang sort cột PROXY; `verify:raw-proxy` sẽ ghi đè ngay sau.
- Xóa `runProxyCache`/`displayProxyCache` cho slot → `fetchAccounts` không restore giá trị cũ.
- **Race-safe**: nếu `slotId` chưa có trong index (slot-assigned đến trước khi `fetchAccounts` slot init xong) → tạo placeholder qua `createEmptyAccount` thay vì drop event.

### 4.4 Proxy cache — vì sao tách `runProxyCache`/`displayProxyCache`

[useAccountsStore.ts:189-211](../../frontend/src/modules/accounts/store/useAccountsStore.ts). `fetchAccounts` map lại `accounts[]` từ backend, **restore** proxy từ cache để không mất IP đang chạy:
```ts
runProxy: runProxyCache.get(acc.id) ?? acc.runProxy ?? '',
proxy:    displayProxyCache.get(acc.id) ?? acc.proxy ?? '',
```
Cache cap **2000 entries** (LRU, `capMap`) để chạy 24/7 không leak RAM. `setRunProxy`/`setDisplayProxy` ghi cache + in-memory đồng thời.

### 4.5 Backend `OnAccountDone` — KHÔNG clear token/cookie

[app.go:2956-3160](../../app.go). Khi account verify xong, backend update `a.accounts[i]` rồi clear field nặng (`FullData/SourceCode/NoteRun`) để giải phóng RAM (file mode 1M+ account) — **NHƯNG cố ý GIỮ `Token`/`Cookie`** ([app.go:3000-3001](../../app.go)):
```go
a.accounts[i].FullData = ""
a.accounts[i].Cookie = doneAcc.Cookie  // GIỮ cookie — clear sẽ bị fetchAccounts refetch ra rỗng
a.accounts[i].Token  = doneAcc.Token   // GIỮ token  — clear sẽ bị fetchAccounts refetch ra rỗng
```
> Đây chính là **caveat §13.7 D** của [add-facebook-reg-version.md](./add-facebook-reg-version.md): `verify:complete`/`verify:accounts-updated` kích `fetchAccounts()` (refetch từ `ListAccounts`). Nếu backend clear `Token`/`Cookie` thì refetch ra rỗng → cột TOKEN/COOKIE trên UI **biến mất** (status vẫn còn vì backend giữ). Status backend giữ nên không sao, nhưng token/cookie là cột hiển thị → phải giữ.

---

## 5. LUỒNG REGISTER (`RunRegister`) + inline/split verify

**Mục tiêu**: tạo account liên tục N luồng; nếu bật Verify → verify ngay sau reg.
**Input**: `handleRunRegister()` → `runner.runRegister(threads)` → backend `RunRegister`.

### 5.1 Bảng REGISTER (REG pane) — `registerThreads`

KHÁC verify: REG dùng `registerThreads: Map<index, RegThread>` (LOCAL state, không phải store). Mỗi `index` = goroutine slot. Render qua `regGridRows` computed ([AccountsPage.vue:432-463](../../frontend/src/pages/AccountsPage.vue)) — sort theo `index` để vị trí row ổn định, cap 250 rows.

`RegThread` ([AccountsPage.vue:122-140](../../frontend/src/pages/AccountsPage.vue)) map sang cột grid:
- UID=`uid`, Password=`password`, Cookie=`cookie`, Token=`token`, UA=`userAgent`, Hoạt động=`activity`.
- **Email** = `verifyEmail || phone` — mail đang verify ưu tiên hơn login reg, KHÔNG bị status ghi đè ([:447-450](../../frontend/src/pages/AccountsPage.vue)).
- Proxy=`proxyServer` (ip:port:user:pass), IP chạy=`proxy` (real IP).
- Status hiển thị = `regRowStatus(t)` ([:485-503](../../frontend/src/pages/AccountsPage.vue)): `new`(xanh)→`nvr`(vàng, reg xong chưa verify)→`live`(xanh lá)/`verfail`(xám)/`addmail`(cam, đang retry add-mail)/`unknown`/`die`(đỏ).

### 5.2 Trình tự event 1 account register (có inline verify)

```
1. register:status {index,phone,proxy,...,msg}
      → pendingStatus → flush: tạo/cập nhật RegThread, status='running', activity=msg
      (phone==="system" → pushLog, KHÔNG vào bảng)
2. register:batch-status [{index,...,msg}]   → cập nhật activity (skip nếu thread đã done)
3. (reg success, có UID) register:token {index,uid,token,cookie}
      → pendingTokens → flush: set uid/token/cookie + ĐẾM regTotalSuccess++ / regTotalProcessed++ NGAY
      (không đợi verify — verify mất vài phút; counter live realtime)
4. (nếu bật verify) inline verify chạy → các verify:* event (slot-assigned/email/proxy/batch-status)
      ⚠ Ở NORMAL (non-split): verify event đi vào registerThreads qua verify:email (verifyEmail),
        các verify:batch-status dùng accountId=threadIdx nhưng accountsStore không có row đó nếu non-split.
5. register:account-done {index,success,uid,cookie,token,...,verifyStatus,verifyMessage,verifyEmail,checkpoint}
      → pendingDone → flush:
         - set uid/password/cookie/token/UA/proxy, status='success'|'failed', finishedAt=now
         - verifyStatus 'live'/'die' → màu row; rawVerifyStatus giữ status thô để phân loại unknown
         - success=true: regTotalLive/Die/Unknown++ (success ĐÃ đếm ở register:token — KHÔNG double)
         - success=false: regTotalProcessed++ + regTotalFail++ (+regTotalCheckpoint nếu checkpoint)
6. register:complete {total} → isRegisterRunning=false, stopElapsedTimer, (đk) fetchAccounts
```

> **Vì sao đếm success ở `register:token` chứ không ở `register:account-done`?** ([AccountsPage.vue:1628-1631](../../frontend/src/pages/AccountsPage.vue)) — reg thành công xong là có UID/token ngay, nhưng `register:account-done` chỉ đến SAU khi verify xong (vài phút). Đếm ở token → counter "Reg thành công" nhảy realtime. `account-done` chỉ đếm FAIL (token không fire cho fail) + verify outcome.

### 5.3 Split mode (UI 2 pane REG + VERIFY)

Khi `cfg.splitMode && cfg.verifyEnabled` ([AccountsPage.vue:1422](../../frontend/src/pages/AccountsPage.vue)):
- `splitModeActive=true` → template render 2 `DataGrid` xếp dọc ([AccountsPage.vue:1992-2090](../../frontend/src/pages/AccountsPage.vue)):
  - **REG pane** (trên): `:rows="regGridRows"` (= `registerThreads`).
  - **VERIFY pane** (dưới): `:rows="grid.visibleItems.value"` (= `accountsStore.accounts` qua `useDataGrid`).
- `handleRun()` gọi `setupVerifyListeners()` TRƯỚC `handleRunRegister()` → cả 2 nhóm listener active.
- Backend split: reg worker emit `register:*` cho REG pane; verify worker emit `verify:slot-assigned`/`verify:account-done` cho VER pane (slot riêng `verSlot`, [app.go:8392/8441](../../app.go)).
- **2026-05-15**: split = PURE UI option — worker logic GIỐNG normal (1 worker = REG + VER inline). Backend chỉ tách event để 2 pane hiển thị riêng. `regToVerCh` pool độc lập đã DEPRECATED (`if false`).
- `splitModeActive` persist `localStorage['havu:splitModeActive']` để sống qua UI reload; authoritative restore từ backend `IsRegisterRunning()` + config ở `onMounted` ([AccountsPage.vue:676-688](../../frontend/src/pages/AccountsPage.vue)).

> **VERIFIED totals gộp 2 nguồn** ([AccountsPage.vue:189-191](../../frontend/src/pages/AccountsPage.vue)): normal mode verify inline → `regTotalLive/Die`; split mode VER pool → `verifyTotalLive/Die` (qua `verify:account-done`). `verifiedLiveTotal = regTotalLive + verifyTotalLive` (1 trong 2 luôn =0 tùy mode).

### 5.4 Dedupe đếm verify theo UID (chống double-count)

[AccountsPage.vue:1309-1328](../../frontend/src/pages/AccountsPage.vue). Split mode: account `unknown` được pop lại từ `Unknown.txt` → chạy slot KHÁC (slotId khác) nhưng CÙNG UID. Nếu dedupe theo slotId → Tổng cộng dồn gấp đôi. Fix:
- `_verifyLastStatusByUid: Map<uid, status>` — dedupe theo UID (ưu tiên), fallback `_verifyLastStatus` theo slotId.
- Có `prev` → trừ counter status cũ, cộng status mới (KHÔNG tăng Processed). Không có `prev` → `verifyTotalProcessed++`.
- `verify:slot-assigned` xóa `_verifyLastStatus[slotId]` ([:1395](../../frontend/src/pages/AccountsPage.vue)) để slot tái dùng được tính Processed mới.

---

## 6. Edge case & gotcha (PHẢI BIẾT)

### 6.1 `verify:status` KHÔNG có frontend handler ⚠️
Backend emit `verify:status` cho system message (`accountId:0, uid:"system"`) tại nhiều chỗ ([app.go:2484/2525/2656/3246/3597/3629](../../app.go)) — proxy error, "[Result] path", "[File] source folder", "hết mail"... NHƯNG `setupVerifyListeners` **không đăng ký** `verify:status`. → Các message system này **bị drop khỏi UI** (chỉ vào slog backend). Đối lập: `register:status` với `phone:"system"` → `pushLog` vào log panel ([AccountsPage.vue:1698](../../frontend/src/pages/AccountsPage.vue)). Nếu muốn hiện system msg verify, cần thêm listener.

### 6.2 `split-verify:drained` KHÔNG có frontend handler ⚠️
Backend emit khi VER pool (split) drain hết queue ([app.go:10840](../../app.go)) để "báo UI unlock Start" — nhưng frontend **chưa đăng ký**. Hiện UI unlock dựa vào `register:complete`. Đây là dead event ở phía FE (an toàn, chỉ là TODO chưa wire).

### 6.3 Payload `msg` vs `message` không đồng nhất ⚠️
- `register:status`/`register:batch-status` dùng key **`msg`**.
- `verify:batch-status`/`verify:status`/`verify:account-done` dùng key **`message`**.
- Một vài `verify:status` ([app.go:2656](../../app.go)) lại dùng `msg` (không khớp `message` mà handler-nếu-có sẽ đọc). Khi thêm listener verify:status phải đọc cả 2 key.

### 6.4 `register:status` 2 dạng payload
`RegisterFacebook()` (đăng ký ĐƠN, [app.go:7023](../../app.go)) emit `register:status` là **string thuần**, trong khi `RunRegister` (batch) emit là **object** `{index,phone,...,msg}`. Handler `setupRegisterListeners` chỉ xử lý dạng object (`data.phone`, `data.index`). `RegisterFacebook` hiện không được UI gọi (chỉ batch) nên không xung đột, nhưng cần lưu ý nếu wire lại.

### 6.5 Race `slot-assigned` đến trước `fetchAccounts` slot-init
`verify:accounts-updated` (nil) trigger `fetchAccounts()` (async, có IPC round-trip). Backend có thể emit `verify:slot-assigned`/`verify:batch-status` TRƯỚC khi fetch xong → slot chưa có trong `accountsIndex`. Giải pháp: `applySlotAssigned` + `ensureSlot` tự tạo placeholder row ([useAccountsStore.ts:231-237, 264-272](../../frontend/src/modules/accounts/store/useAccountsStore.ts)) → không drop event.

### 6.6 FINAL_STATUSES chặn batch-status ghi đè kết quả
`FINAL_STATUSES = {live,die,unknown,checkpoint}` ([AccountsPage.vue:265](../../frontend/src/pages/AccountsPage.vue)). `verify:batch-status` và `flushActivityBuffer` BỎ QUA account đã FINAL ([:1282, :1270](../../frontend/src/pages/AccountsPage.vue)) — tránh message hoạt động đến muộn ghi đè kết quả "live/die" cuối cùng từ `verify:account-done`.

### 6.7 Auto-clear row chỉ ở file-mode standalone
`verify:account-done` setTimeout 1.5s xóa row CHỈ khi `accountSource==='file'` và KHÔNG phải slot-recycling mode (split/api/folder) ([AccountsPage.vue:1335-1354](../../frontend/src/pages/AccountsPage.vue)). Slot-recycling mode giữ row để backend update in-place khi push account mới vào slot. `currentRunId` guard chống setTimeout cũ xóa nhầm row của run mới (Stop→Run nhanh).

### 6.8 Restore state sau UI reload (auto 6h / F5)
`onMounted` đọc `GetRunStatus()` từ backend (single source of truth) ([AccountsPage.vue:633-695](../../frontend/src/pages/AccountsPage.vue)): nếu reg/verify còn chạy → re-`setupXxxListeners()`, `restoreRunStats()` (localStorage `havu:runStats`), `resumeElapsedTimer()` (localStorage `havu:runStartTime`), restore `splitModeActive` theo config. Counter + elapsed survive reload nhờ localStorage.

### 6.9 KeepAlive — listener sống qua đổi tab
AccountsPage dùng KeepAlive: `onDeactivated` lưu `_deactivatedWhileRunning`; `onActivated` SKIP `fetchAccounts` nếu đang chạy (in-memory event data mới hơn backend, tránh race ghi đè) ([AccountsPage.vue:699-722](../../frontend/src/pages/AccountsPage.vue)). Listener KHÔNG hủy khi đổi tab — chỉ hủy ở `onUnmounted`.

---

## 7. Các event hệ thống ngoài AccountsPage

- **`system:memory-warning`** ([App.vue:18-23](../../frontend/src/app/App.vue)) — Go heap >500MB → toast warning, cooldown 10 phút.
- **`system:ui-reload`** ([App.vue:30-50](../../frontend/src/app/App.vue)) — soft cleanup mỗi 12h: clear log buffer, clear proxy cache, dispatch `app:soft-cleanup` (window event → AccountsPage `handleSoftCleanup` trim `registerLogs`). KHÔNG `location.reload()` (giữ UX).
- **`app:request-quit-confirm`** ([AppTitleBar.vue:44-54](../../frontend/src/components/shell/AppTitleBar.vue)) — user nhấn X/Alt+F4 → backend `OnBeforeClose` block + emit `{registerRunning,verifyRunning}` → FE mở ConfirmDialog (cảnh báo nếu đang chạy) → confirm → `App.RequestQuit()`.
- **`upload-site:log` / `:stopped` / `:log-cleared`** ([uploadLog.store.ts:127-148](../../frontend/src/stores/uploadLog.store.ts)) — đăng ký qua `window.runtime.EventsOn` (không qua bridge), cap log 200 dòng, adaptive polling `GetUploadStats`.
- **`accounts:folder-updated` / `email:pool-exhausted` / `verify:email`** — đăng ký ở `_alwaysOnUnsubs` ([AccountsPage.vue:572-624](../../frontend/src/pages/AccountsPage.vue)) để fire ở MỌI chế độ. `email:pool-exhausted` throttle 30s/provider.

---

## 8. Checklist khi thêm/sửa event mới

1. Backend: `runtime.EventsEmit(a.ctx, "namespace:name", payload)` — payload là `map[string]interface{}` hoặc slice. Thống nhất key (`message` cho verify, `msg` cho register — xem §6.3).
2. Frontend: chọn đúng nhóm listener (`_registerUnsubs` / `_verifyUnsubs` / `_alwaysOnUnsubs`) tùy chế độ event cần fire.
3. `bus.on(...)` LUÔN `_xxxUnsubs.push(...)` để cleanup chính xác (đừng `bus.off` global).
4. Hot-path (tần suất cao, mỗi account) → gom qua RAF/`_activityBuffer`, không update reactive trực tiếp.
5. Update in-place qua `accountsIndex` (O(1)), tránh `fetchAccounts()` trừ khi cần full refresh (slot init / complete).
6. KHÔNG clear `Token`/`Cookie` ở backend `OnAccountDone` (caveat §4.5 / §13.7 D).
7. Cập nhật bảng §3 của tài liệu này.
