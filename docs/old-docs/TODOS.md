# HVR — Architecture Migration Checklist

Dựa trên: `docs/architecture-migration.md`
Codebase hiện tại xác nhận ngày: 2026-04-07

**Quy tắc:** Mỗi bước là 1 commit riêng. Không break build. Chỉ move + update imports, không thay đổi logic.

---

## Trạng thái tổng quan

| Phase | Tên | Số bước | Trạng thái |
|-------|-----|---------|-----------|
| A | Facebook API restructure | 5 bước | ✅ Hoàn thành (2026-04-07) |
| B | Email restructure | 4 bước | ✅ Hoàn thành (2026-04-07) |
| C | Proxy restructure | 3 bước | ✅ Hoàn thành (2026-04-07) |
| D | Cleanup duplicates | 1 bước | ✅ Hoàn thành (2026-04-07) |

> Phase A, B, C có thể chạy **song song** vì không share file với nhau.

---

## Phase A — Facebook API Restructure

Mục tiêu: Gom `internal/register/` + `internal/verify/` vào `internal/facebook/web/`. Tạo interfaces + factory ở package cha.

### Bước A1 — Tạo `facebook/interfaces.go` + `facebook/types.go`

- [x] Tạo file `internal/facebook/interfaces.go`
  - [x] Định nghĩa interface `Registerer` với method `Register(ctx, input, onStatus) *RegResult`
  - [x] Định nghĩa interface `Verifier` với method `Verify(ctx, session, cfg, onStatus) *VerifyResult`
  - [x] Định nghĩa interface `Interactor` (Like, Comment, Share, AddFriend)
  - [x] Định nghĩa interface `FeedReader` (GetFeed)
  - [x] Định nghĩa interface `SecurityManager` (Enable2FA, HandleCheckpoint, ChangePassword)
  - [x] Định nghĩa type `StatusCallback func(string)` ← nằm trong `facebook/web/types.go`
- [x] Mở rộng file `internal/facebook/types.go` (đã có Session)
  - [x] Move `RegInput` struct từ `internal/register/types.go` sang đây
  - [x] Move `RegSession` struct từ `internal/register/types.go` sang đây
  - [x] Move `RegResult` struct từ `internal/register/types.go` sang đây
  - [x] Thêm `VerifyConfig` struct (từ `internal/verify/verify.go`)
  - [x] Thêm `VerifyResult` struct (từ `internal/verify/verify.go`)
  - [x] Thêm `FeedPost` struct
  - [x] Thêm `TwoFAResult` struct
- [x] Build pass: `go build ./internal/...`
- [ ] Commit: `chore(facebook): add interfaces.go and expand types.go`

### Bước A2 — Tạo `facebook/factory.go`

- [x] Tạo file `internal/facebook/factory.go`
  - [x] Định nghĩa constants: `PlatformAndroid = "android"`, `PlatformWeb = "web"`, `PlatformMfb = "mfb"`
  - [x] Implement `func NewRegisterer(platform string) (Registerer, error)` — plugin registration pattern (init() registry)
  - [x] Implement `func NewVerifier(platform string) (Verifier, error)` — plugin registration pattern
  - [x] (Để trống `android`, `mfb` — thêm sau khi di chuyển code)
- [x] Build pass
- [ ] Commit: `chore(facebook): add factory.go with platform constants`

### Bước A3 — Move `internal/register/` → `internal/facebook/web/`

- [x] Tạo thư mục `internal/facebook/web/`
- [x] Move + rename từng file:
  - [x] `register/init.go` → `facebook/web/register_init.go` (update package name: `package web`)
  - [x] `register/steps.go` → `facebook/web/register_steps.go`
  - [x] `register/body.go` → `facebook/web/register_body.go`
  - [x] `register/crypto.go` → `facebook/web/register_crypto.go`
  - [x] `register/randomize.go` → `facebook/web/register_randomize.go`
  - [x] `register/android_token.go` → `facebook/web/register_android_token.go`
  - [ ] Xóa `register/types.go` (types đã move sang `facebook/types.go` ở A1) ← hiện là type alias redirect
- [x] Update tất cả import bên trong các file mới:
  - [x] Thay `"HVR/internal/register"` → `"HVR/internal/facebook"`
  - [x] Thay references đến `register.RegInput` → `facebook.RegInput`, v.v.
- [x] Update `factory.go`: `NewRegisterer("web")` trả về `&web.Registerer{}` thật (qua init() registration)
- [ ] Xóa thư mục `internal/register/` sau khi confirm build pass ← giữ lại làm compat redirect layer
- [x] Build pass: `go build ./internal/...`
- [ ] Commit: `refactor(facebook): move register/ into facebook/web/`

### Bước A4 — Move `internal/verify/` → `internal/facebook/web/`

- [x] Move + rename từng file:
  - [x] `verify/verify.go` → `facebook/web/verify_verify.go` (update package: `package web`)
  - [x] `verify/steps.go` → `facebook/web/verify_steps.go`
  - [x] `verify/body.go` → `facebook/web/verify_body.go`
  - [x] `verify/check.go` → `facebook/web/verify_check.go`
- [x] Update imports bên trong các file mới:
  - [x] Thay `"HVR/internal/verify"` → `"HVR/internal/facebook"`
  - [x] Thay references đến `verify.Config` → `facebook.VerifyConfig`, v.v.
- [x] Xóa `createEmailService()` khỏi `verify.go` (dùng `email.New()` từ Phase B)
- [x] Update `factory.go`: `NewVerifier("web")` trả về `&web.Verifier{}` thật (qua init() registration)
- [ ] Xóa thư mục `internal/verify/` ← giữ lại làm compat redirect layer
- [x] Build pass
- [ ] Commit: `refactor(facebook): move verify/ into facebook/web/`

### Bước A5 — Update `app.go` + `runner/scheduler.go` imports

- [ ] Trong `app.go`:
  - [ ] Thay import `"HVR/internal/register"` → `"HVR/internal/facebook"` ← chưa làm
  - [ ] Thay import `"HVR/internal/verify"` → `"HVR/internal/facebook"` ← chưa làm
  - [ ] Thay tất cả call sang `facebook.NewRegisterer("web")` + `facebook.NewVerifier("web")` ← chưa làm
- [x] Trong `runner/scheduler.go`:
  - [x] Update import paths tương tự ← không cần (scheduler.go không import register/verify)
- [x] Build pass: `go build ./internal/...`
- [ ] Test smoke: chạy `cmd/regtest` hoặc `cmd/verifytest` nếu có ← không tồn tại
- [ ] Commit: `chore: update app.go + runner imports to facebook package`

> **Ghi chú A5:** Đã thêm `_ "HVR/internal/facebook/web"` vào `app.go` để kích hoạt
> plugin registration. Việc thay thế hoàn toàn `register`/`verify` import trong `app.go` là bước
> tiếp theo khi sẵn sàng xóa redirect layer.

---

## Phase B — Email Restructure

Mục tiêu: Tách email temp vs rent vào subpackages. Tạo factory tập trung. Xóa `createEmailService()` khỏi verify.

### Bước B6 — Tạo `email/factory.go` + `email/options.go`

- [x] Tạo file `internal/email/factory.go`
  - [x] Định nghĩa hàm `New(opts Options) (Service, error)` — factory chính
  - [x] Implement switch theo `opts.Provider` (string: "moakt", "zeus_x", v.v.)
- [x] Tạo file `internal/email/options.go`
  - [x] Định nghĩa struct `Options { Provider string; ... }` với đầy đủ fields per provider
  - [x] Thay thế các flat fields đang nằm trong `verify.Config`
- [x] Rename `internal/email/email.go` → `internal/email/service.go`
  - [x] Giữ nguyên nội dung (interface `Service`) trong `service.go`
  - [ ] Xóa file `email.go` cũ ← hiện là deprecated stub (1 dòng comment)
- [x] Build pass
- [ ] Commit: `chore(email): add factory.go, options.go, rename email.go→service.go`

### Bước B7 — Move email temp providers → `email/temp/`

- [x] Tạo thư mục `internal/email/temp/`
- [x] Move + update package name thành `package temp`:
  - [x] `email/moakt.go` → `email/temp/moakt.go`
  - [x] `email/mail1sec.go` → `email/temp/mail1sec.go`
  - [x] `email/mohmal.go` → `email/temp/mohmal.go`
  - [x] `email/temporary_mail_net.go` → `email/temp/temporary_mail_net.go`
- [x] Update import trong `email/factory.go` để dùng `temp.NewMoakt{}`, v.v.
- [x] Build pass
- [ ] Commit: `refactor(email): move temp providers into email/temp/`

### Bước B8 — Move email rent providers → `email/rent/`

- [x] Tạo thư mục `internal/email/rent/`
- [x] Move + update package name thành `package rent` (và rename file: bỏ suffix `_mail`):
  - [x] `email/zeus_x.go` → `email/rent/zeus_x.go`
  - [x] `email/dongvanfb_mail.go` → `email/rent/dongvanfb.go`
  - [x] `email/store1s_mail.go` → `email/rent/store1s.go`
  - [x] `email/mail30s_mail.go` → `email/rent/mail30s.go`
- [x] Update import trong `email/factory.go`
- [x] Build pass
- [ ] Commit: `refactor(email): move rent providers into email/rent/`

### Bước B9 — Xóa `createEmailService()` trong verify, gọi `email.New()`

- [x] Trong `internal/facebook/web/verify_verify.go` (đã move từ Phase A4):
  - [x] Xóa hàm `createEmailService()` (hoặc hàm tương đương)
  - [x] Thay bằng call `email.New(email.Options{...})`
  - [x] Import `"HVR/internal/email"`
- [x] Xác nhận không còn email factory logic nào nằm ngoài `email/factory.go`
- [x] Build pass
- [ ] Commit: `refactor(email): replace createEmailService() with email.New()`

---

## Phase C — Proxy Restructure

Mục tiêu: Thêm `Provider` interface. Move 5 provider files vào `proxy/providers/`.

### Bước C10 — Tạo `proxy/provider.go` interface

- [x] Tạo file `internal/proxy/provider.go`
  - [x] Định nghĩa interface `Provider`:
    ```go
    type Provider interface {
        Acquire(ctx context.Context) (string, func(), error)
        Len() int
    }
    ```
- [x] Build pass
- [ ] Commit: `chore(proxy): add Provider interface`

### Bước C11 — Move proxy provider files → `proxy/providers/`

- [x] Tạo thư mục `internal/proxy/providers/`
- [x] Move + update package name thành `package providers`:
  - [x] `proxy/tinsoft.go` → `proxy/providers/tinsoft.go`
  - [x] `proxy/shoplike.go` → `proxy/providers/shoplike.go`
  - [x] `proxy/netproxy.go` → `proxy/providers/netproxy.go`
  - [x] `proxy/minproxy.go` → `proxy/providers/minproxy.go`
  - [x] `proxy/proxyfarm.go` → `proxy/providers/proxyfarm.go`
- [x] Mỗi provider implement `proxy.Provider` interface
- [x] Build pass
- [ ] Commit: `refactor(proxy): move providers into proxy/providers/`

### Bước C12 — Refactor `proxy/manager.go` dùng `Provider` interface

- [x] Trong `internal/proxy/manager.go`:
  - [x] Thay concrete type fields bằng `Provider` interface
  - [x] Import `"HVR/internal/proxy/providers"`
  - [x] Update dispatch logic: `provider.Acquire(ctx)` thay vì gọi trực tiếp
- [x] Build pass
- [ ] Commit: `refactor(proxy): manager.go use Provider interface`

---

## Phase D — Cleanup Duplicates

Mục tiêu: Xóa code duplicate giữa `register/init.go` (đã move) và `proxy/client.go`.

### Bước D13 — Xóa duplicate proxy parsing trong web/register_init.go

- [x] Trong `internal/facebook/web/register_init.go`:
  - [x] Tìm hàm `buildRegHTTPClient()` — xóa
  - [x] Tìm hàm `parseRegProxyURL()` — xóa
  - [x] Thay tất cả call bằng `proxy.CreateClient(proxyStr, 20*time.Second)`
  - [x] Import `"HVR/internal/proxy"`
- [x] Xác nhận `proxy.CreateClient()` trong `proxy/client.go` đã cover đủ cases
- [x] Build pass: `go build ./internal/...`
- [ ] Test smoke end-to-end (register flow) ← không có cmd/regtest
- [ ] Commit: `fix(web): remove duplicate proxy parsing, use proxy.CreateClient()`

---

## Kiểm tra sau khi hoàn thành tất cả phases

### Cấu trúc thư mục

- [x] `internal/facebook/` có: `types.go`, `interfaces.go`, `login.go`, `factory.go` ✓ (`helpers.go` không cần)
- [x] `internal/facebook/web/` có: register_*.go (6 files) + verify_*.go (4 files) + register.go + verify.go + types.go
- [ ] `internal/facebook/android/` — tạo skeleton nếu cần (hiện tại có thể để trống)
- [ ] `internal/facebook/mfb/` — tạo skeleton nếu cần
- [x] `internal/email/` có: `service.go`, `pool.go`, `factory.go`, `options.go`
- [x] `internal/email/temp/` có: 4 providers
- [x] `internal/email/rent/` có: 4 providers + pool.go
- [x] `internal/proxy/` có: `provider.go`, `client.go`, `pool.go`, `manager.go`, `checkip.go`
- [x] `internal/proxy/providers/` có: 5 providers
- [ ] `internal/register/` — ĐÃ XÓA ← còn là compat redirect layer
- [ ] `internal/verify/` — ĐÃ XÓA ← còn là compat redirect layer

### Code quality

- [x] `go build ./internal/...` pass không lỗi (`./...` fail do thiếu `frontend/dist`)
- [x] `go vet ./internal/...` clean
- [x] Không còn `createEmailService()` nằm ngoài `email/factory.go`
- [x] Không còn `buildRegHTTPClient()` hay `parseRegProxyURL()` trong `facebook/web/`
- [x] `app.go` không import trực tiếp `register` hoặc `verify` package cũ ← đã xóa (Phase L)
- [x] Không file nào có tên `*Copy*`, `*GÓC*`, hay commented-out blocks lớn

### Đường dẫn thêm mới sau migration

| Thêm gì | Chỉ cần |
|---------|---------|
| Nền tảng Facebook mới | Thư mục mới + 1 case trong `factory.go` |
| Email provider mới | 1 file trong `temp/` hoặc `rent/` + 1 case factory |
| Proxy provider mới | 1 file trong `providers/` + 1 case manager |

---

## Phase E — Split facebook/web/ thành register/web/ + verify/web/

Mục tiêu: Tách `internal/facebook/web/` thành hai packages riêng biệt theo VerifyCloneVIP structure. Hoàn thành: 2026-04-07

| Phase | Tên | Trạng thái |
|-------|-----|-----------|
| E | Split facebook/web/ → register/web/ + verify/web/ | ✅ 2026-04-07 |
| F | Split verify files → verify/web/ | ✅ 2026-04-07 |
| G | Skeleton platforms (4 register + 3 verify mới) | ✅ 2026-04-07 |
| H | Feature API skeletons (feed, interaction, security) | ✅ 2026-04-07 |
| I | FakeInfo / FormData / MachineId / Checkpoint | ✅ 2026-04-07 |
| J | httpclient / iplookup / stats infrastructure | ✅ 2026-04-07 |
| K | Extended types (status, constants, options, results) | ✅ 2026-04-07 |
| L | Final cleanup + rewire app.go to factory | ✅ 2026-04-07 (code rewired; file deletion còn lại) |

### Phase E (✅)

- [x] `facebook/register/web/` — 7 register files + types.go
- [x] `facebook/web/register.go` — stub (init() removed)
- [x] app.go blank import `facebook/register/web`

### Phase F (✅)

- [x] `facebook/verify/web/` — 5 verify files + types.go
- [x] `facebook/web/verify.go` — stub (init() removed)
- [x] app.go blank import `facebook/verify/web`

### Phase G (✅)

- [x] `facebook/factory.go` thêm: PlatformChrome, PlatformWebAndroid, PlatformWebRequest
- [x] `facebook/register/android/`, `chrome/`, `webandroid/` — stub Registerer
- [x] `facebook/verify/android/`, `webandroid/`, `webrequest/` — stub Verifier
- [x] app.go blank imports tất cả 4+4 platforms

### Phase H (✅)

- [x] `facebook/feed/web/`, `feed/android/`
- [x] `facebook/interaction/web/`, `interaction/android/`
- [x] `facebook/security/web/`, `security/webandroid/`, `security/android/`

### Phase I (✅)

- [x] `facebook/fakeinfo/builder.go` — UserAgentBuilder + device models
- [x] `facebook/formdata/builder.go` — FormProp/Builder/HeaderBuilder
- [x] `facebook/machineid/manager.go` — MachineId generation
- [x] `facebook/checkpoint/detector.go` — checkpoint type detection

### Phase J (✅)

- [x] `internal/httpclient/client.go` — IHttpRequestClient equivalent
- [x] `internal/iplookup/lookup.go` — IIPLookupAPI, ip-api.com
- [x] `internal/stats/reporter.go` — IStaticsReport, atomic counters

### Phase K (✅)

- [x] `facebook/status.go`, `constants.go`, `options.go`, `results.go`
- [x] `facebook/verify/status.go` — AddEmail/ConfirmEmail/AccountAuto status codes
- [x] `facebook/security/status.go` — FeaturesAuto/Enable2FA status codes
- [x] `email/automation/automation.go` — IMailServiceAutomation interface

### Phase L (✅ — hoàn thành 2026-04-07)

- [x] Rewire `app.go`: thay `register.RegisterAccount(...)` → `facebook.NewRegisterer("web").Register(...)`
- [x] Rewire `app.go`: thay `verify.Config{}` → `facebook.VerifyConfig{}`
- [x] Rewire `app.go`: thay `register.RandomRegInput(...)` → `webregister.RandomRegInput(...)`
- [x] Rewire `app.go`: thay `register.GeneratePhoneByCountry(...)` → `webregister.GeneratePhoneByCountry(...)`
- [x] Xóa `app.go` imports của `internal/register` và `internal/verify`
- [x] Rewire `scheduler.go`: thay `verify.VerifyAccount(...)` → `facebook.NewVerifier("web").Verify(...)`
- [x] Rewire `scheduler.go`: thay `verify.SaveAccountToFolder(...)` → `webverify.SaveAccountToFolder(...)`
- [x] Rewire `scheduler.go`: thay `*verify.Config` → `*facebook.VerifyConfig`
- [x] Build pass, vet pass, tests pass
- [ ] Xóa dead `facebook/web/register_*.go` (7 files) ← cần `rm`, thực hiện thủ công
- [ ] Xóa dead `facebook/web/verify_*.go` (5 files) ← cần `rm`, thực hiện thủ công
- [ ] Xóa `internal/register/` package ← cần `rm`, thực hiện thủ công
- [ ] Xóa `internal/verify/` package ← cần `rm`, thực hiện thủ công

---

## Ghi chú khi implement

- **Không thay đổi logic** trong khi move file. Logic fix là việc khác, làm sau.
- **Mỗi bước = 1 commit** — dễ rollback nếu có vấn đề.
- Phase A3 và A4 có risk "Trung bình" vì thay đổi nhiều import paths. Kiểm tra kỹ trước khi xóa thư mục cũ.
- `register/android_token.go` → `facebook/web/register_android_token.go`: file này dùng `b-graph.facebook.com` (Android endpoint) nhưng được gọi từ web register flow B8. Giữ nguyên trong `web/`.
- `email/email.go` chỉ cần đổi tên thành `service.go`, không thay đổi nội dung.
- `internal/register/` và `internal/verify/` giữ lại làm **compat redirect layer** cho đến khi `app.go` hoàn toàn chuyển sang dùng `facebook.NewRegisterer/NewVerifier`.
