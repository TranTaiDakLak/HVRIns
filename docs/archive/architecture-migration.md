# Architecture Migration: VerifyCloneVIP (C#) → HVR (Go + Wails + Vue 3)

## Tổng quan

Tài liệu này mô tả cấu trúc thư mục mới cho HVR (Go + Wails + Vue 3), được thiết kế dựa trên cấu trúc domain của VerifyCloneVIP (C#) nhưng sửa các lỗi kiến trúc của dự án cũ.

Nguyên tắc: **giữ cách phân chia nền tảng rõ ràng, bỏ các anti-pattern.**

---

## 1. Cấu trúc thư mục mới — HVR

### 1.1 Backend (Go) — `internal/`

```
internal/
│
│  ══════════════════════════════════════════════════════
│  FACEBOOK API — tách theo nền tảng, mỗi nền tảng
│  có đầy đủ: register, verify, interaction, feed, security
│  ══════════════════════════════════════════════════════
│
├── facebook/
│   ├── types.go                     # Session, RegInput, RegResult, VerifyResult
│   ├── interfaces.go                # Interface chung: Registerer, Verifier, Interactor...
│   ├── login.go                     # Cookie login (dùng chung mọi nền tảng)
│   ├── factory.go                   # facebook.NewRegisterer("android") → Registerer
│   ├── helpers.go                   # extractFirst(), parseTokens(), buildHeaders()...
│   │
│   ├── android/                     # ← C#: API/FbAndroidApi/
│   │   ├── register.go              #    FacebookRegisterAPIAndroid.cs
│   │   ├── verify.go                #    FacebookVerifyAndroidAPI.cs + FacebookVerifyAPIToken.cs
│   │   ├── interaction.go           #    FacebookInteractionAndroidAPI.cs
│   │   ├── feed.go                  #    FacebookFeedAndroidAPI.cs
│   │   └── security.go             #    FacebookSecurityFeatureAPIAndroid.cs
│   │
│   ├── web/                         # ← C#: API/FBWebApi/ (m.facebook.com endpoints)
│   │   ├── register.go              #    ← hiện tại: internal/register/ (cả thư mục)
│   │   ├── register_steps.go        #    B1-B8 orchestrator
│   │   ├── register_body.go         #    Request body builders
│   │   ├── register_crypto.go       #    Password encryption (Curve25519)
│   │   ├── register_init.go         #    FetchRegTokens từ r.php
│   │   ├── register_randomize.go    #    Fake info generation
│   │   ├── verify.go                #    ← hiện tại: internal/verify/ (cả thư mục)
│   │   ├── verify_steps.go          #    B1-B5 verify flow
│   │   ├── verify_body.go           #    Verify request body builders
│   │   ├── verify_check.go          #    CheckLiveDie + SaveToFolder
│   │   ├── interaction.go           #    FacebookInteractionAPI.cs (chưa implement)
│   │   ├── feed.go                  #    FacebookFeedAPI.cs (chưa implement)
│   │   ├── security.go              #    FacebookSecurityFeatureAPI.cs (chưa implement)
│   │   └── ads.go                   #    FacebookAdsRequestAPI.cs (chưa implement)
│   │
│   └── mfb/                         # ← C#: MfbChrome + MfbRequest (nếu cần sau)
│       ├── register.go              #    FacebookRegisterMfbChrome.cs
│       └── register_request.go      #    FacebookRegisterMfbRequest.cs
│
│  ══════════════════════════════════════════════════════
│  EMAIL — tách Temp (tạo mới) vs Rent (mua bằng API key)
│  ══════════════════════════════════════════════════════
│
├── email/
│   ├── service.go                   # Interface Service chung
│   ├── pool.go                      # CredPool — batch purchasing (giữ nguyên)
│   ├── factory.go                   # email.New(opts) → Service
│   ├── options.go                   # EmailOptions struct
│   │
│   ├── temp/                        # ← C#: API/TempMailServer/
│   │   ├── moakt.go                 #    MoaktMailAPI.cs
│   │   ├── mail1sec.go              #    Mail1secComAPI.cs
│   │   ├── mohmal.go                #    MohmalcomAPI.cs
│   │   └── temporary_mail_net.go    #    Temporary_mailNetAPI.cs
│   │
│   └── rent/                        # ← C#: API/RentMailServer/
│       ├── zeus_x.go                #    ZeusxAPI.cs
│       ├── dongvanfb.go             #    DongvanfbAPI.cs
│       ├── store1s.go               #    Store1sAPI.cs
│       └── mail30s.go               #    Mail30sAPI.cs
│
│  ══════════════════════════════════════════════════════
│  PROXY — tách Provider interface + implementations
│  ══════════════════════════════════════════════════════
│
├── proxy/
│   ├── provider.go                  # Interface Provider (Acquire, Release)
│   ├── client.go                    # CreateClient() — http.Client có proxy
│   ├── pool.go                      # Round-robin manual pool
│   ├── manager.go                   # Manager — dispatch theo Provider
│   ├── checkip.go                   # IP lookup services
│   │
│   └── providers/                   # ← C#: API/TempProxyServer/
│       ├── tinsoft.go               #    TinsoftProxy.cs
│       ├── shoplike.go              #    ShoplikeProxy.cs
│       ├── netproxy.go              #    Netproxyio.cs
│       ├── minproxy.go              #    Kiotproxy.cs
│       └── proxyfarm.go             #    DTProxy.cs
│
│  ══════════════════════════════════════════════════════
│  INFRASTRUCTURE — runner, config, external APIs
│  ══════════════════════════════════════════════════════
│
├── runner/                          # FIFO worker pool (giữ nguyên)
│   └── scheduler.go
│
├── clonehv/                         # CloneHV API — mua account (giữ nguyên)
│   └── client.go
│
└── config/                          # Config loader (giữ nguyên)
    └── settings.go
```

### 1.2 Frontend (Vue 3 + TypeScript) — `frontend/src/`

```
frontend/src/
├── app/                             # App bootstrap
├── layouts/                         # AppLayout (sidebar + header + statusbar)
├── pages/                           # Route pages
│   ├── AccountsPage.vue
│   ├── FlowSettingsPage.vue
│   ├── ProxySettingsPage.vue
│   ├── GeneralSettingsPage.vue
│   ├── InteractionSetupPage.vue
│   └── ViewSettingsPage.vue
├── bridge/                          # Wails binding abstraction
│   ├── contracts.ts                 # Interface definitions
│   └── client.ts                    # Wails call wrappers
├── composables/                     # Vue composables (useXxx)
├── stores/                          # Pinia stores
├── components/                      # Shared UI components
├── types/                           # TypeScript types
├── constants/                       # Route names, column defs
├── styles/                          # CSS tokens, reset, themes
└── assets/                          # Static assets
```

### 1.3 Root level

```
HVR/
├── app.go                           # Wails binding — THIN, không business logic
├── main.go                          # Wails app entry
├── go.mod / go.sum
├── wails.json
├── internal/                        # Tất cả business logic (xem 1.1)
├── frontend/                        # Vue 3 app (xem 1.2)
├── build/                           # Build assets (icon, manifest)
├── cmd/                             # CLI test tools
│   ├── regtest/
│   ├── verifytest/
│   └── emailtest/
├── Config/Settings/                 # Runtime config JSON
└── docs/                            # Documentation
```

---

## 2. Facebook API — Thiết kế chi tiết

### 2.1 Interfaces chung (facebook/interfaces.go)

Mỗi nền tảng (android, web, mfb) implement cùng bộ interface. Khi thêm nền tảng mới, chỉ cần tạo thư mục mới + implement.

```go
// facebook/interfaces.go

// Registerer — đăng ký tài khoản Facebook
type Registerer interface {
    Register(ctx context.Context, input *RegInput, onStatus StatusCallback) *RegResult
}

// Verifier — verify tài khoản (add email + confirm OTP)
type Verifier interface {
    Verify(ctx context.Context, session *Session, cfg *VerifyConfig, onStatus StatusCallback) *VerifyResult
}

// Interactor — tương tác với account (like, comment, share, add friend)
type Interactor interface {
    Like(ctx context.Context, session *Session, postID string) error
    Comment(ctx context.Context, session *Session, postID string, text string) error
    Share(ctx context.Context, session *Session, postID string) error
    AddFriend(ctx context.Context, session *Session, targetUID string) error
}

// FeedReader — đọc news feed
type FeedReader interface {
    GetFeed(ctx context.Context, session *Session, limit int) ([]FeedPost, error)
}

// SecurityManager — 2FA, checkpoint, đổi mật khẩu
type SecurityManager interface {
    Enable2FA(ctx context.Context, session *Session) (*TwoFAResult, error)
    HandleCheckpoint(ctx context.Context, session *Session) error
    ChangePassword(ctx context.Context, session *Session, newPassword string) error
}
```

### 2.2 Types chung (facebook/types.go)

```go
// facebook/types.go — dùng chung cho tất cả nền tảng

type Session struct { ... }       // Phiên đăng nhập (đã có)
type RegInput struct { ... }      // Input đăng ký (di chuyển từ register/types.go)
type RegResult struct { ... }     // Kết quả đăng ký (di chuyển từ register/types.go)
type VerifyConfig struct { ... }  // Config verify (di chuyển từ verify/verify.go)
type VerifyResult struct { ... }  // Kết quả verify (di chuyển từ verify/verify.go)
type FeedPost struct { ... }      // Bài viết trên feed
type TwoFAResult struct { ... }   // Kết quả bật 2FA
type StatusCallback func(string)  // Callback trạng thái
```

### 2.3 Factory (facebook/factory.go)

```go
// facebook/factory.go

const (
    PlatformAndroid = "android"
    PlatformWeb     = "web"
    PlatformMfb     = "mfb"
)

func NewRegisterer(platform string) (Registerer, error) {
    switch platform {
    case PlatformAndroid:
        return &android.Registerer{}, nil
    case PlatformWeb:
        return &web.Registerer{}, nil
    case PlatformMfb:
        return &mfb.Registerer{}, nil
    default:
        return nil, fmt.Errorf("unknown platform: %s", platform)
    }
}

func NewVerifier(platform string) (Verifier, error) {
    switch platform {
    case PlatformAndroid:
        return &android.Verifier{}, nil
    case PlatformWeb:
        return &web.Verifier{}, nil
    default:
        return nil, fmt.Errorf("unknown platform: %s", platform)
    }
}

// Tương tự cho NewInteractor(), NewFeedReader(), NewSecurityManager()
```

### 2.4 Luồng dữ liệu

```
app.go (Wails binding)
  │
  ├── "Chọn nền tảng nào?" ← từ frontend settings
  │
  ├── facebook.NewRegisterer("web")  ← factory trả interface
  │     └── web.Registerer.Register(ctx, input, onStatus)
  │           ├── web/register_init.go   → FetchRegTokens()
  │           ├── web/register_steps.go  → B1-B8
  │           └── web/register_crypto.go → EncryptPassword()
  │
  ├── facebook.NewVerifier("android")  ← có thể dùng nền tảng khác
  │     └── android.Verifier.Verify(ctx, session, cfg, onStatus)
  │           ├── android/verify.go      → AddEmail + ConfirmEmail
  │           └── email.New(...)         → tạo email service
  │
  └── runner.RunVerify(ctx, accounts, ...)
        └── mỗi goroutine gọi verifier.Verify()
```

---

## 3. Mapping chi tiết: C# → Go

### 3.1 Facebook API — theo nền tảng

| C# file | Go file | Nền tảng |
|---------|---------|----------|
| **FbAndroidApi/** | **facebook/android/** | |
| `FacebookRegisterAPIAndroid.cs` | `android/register.go` | Android |
| `FacebookVerifyAndroidAPI.cs` | `android/verify.go` | Android |
| `FacebookVerifyAPIToken.cs` | `android/verify.go` (cùng file) | Android |
| `FacebookInteractionAndroidAPI.cs` | `android/interaction.go` | Android |
| `FacebookFeedAndroidAPI.cs` | `android/feed.go` | Android |
| `FacebookSecurityFeatureAPIAndroid.cs` | `android/security.go` | Android |
| **FBWebApi/** | **facebook/web/** | |
| `FacebookRegisterWebAndroidAPI.cs` | `web/register.go` + `web/register_steps.go` | Web |
| `FacebookVerifyAPIRequest.cs` | `web/verify.go` + `web/verify_steps.go` | Web |
| `FacebookVerifyAPIRequestCustomHttpWrapper.cs` | `web/verify.go` (cùng file) | Web |
| `FacebookVerifyWebAndroidAPI.cs` | `web/verify.go` (cùng file) | Web |
| `FacebookInteractionAPI.cs` | `web/interaction.go` | Web |
| `FacebookFeedAPI.cs` | `web/feed.go` | Web |
| `FacebookSecurityFeatureAPI.cs` | `web/security.go` | Web |
| `FacebookAdsRequestAPI.cs` | `web/ads.go` | Web |
| `FacebookInitializeAPIRequest.cs` | `web/register_init.go` | Web |
| **FBWebApi/ (MFB)** | **facebook/mfb/** | |
| `FacebookRegisterMfbChrome.cs` | `mfb/register.go` | MFB |
| `FacebookRegisterMfbRequest.cs` | `mfb/register_request.go` | MFB |

### 3.2 Code hiện tại di chuyển đi đâu

| File hiện tại (NCS) | Di chuyển đến | Ghi chú |
|---------------------|---------------|---------|
| `internal/register/types.go` | `internal/facebook/types.go` | RegInput, RegSession, RegResult |
| `internal/register/init.go` | `internal/facebook/web/register_init.go` | FetchRegTokens, token extraction |
| `internal/register/steps.go` | `internal/facebook/web/register_steps.go` | B1-B8 orchestrator |
| `internal/register/body.go` | `internal/facebook/web/register_body.go` | Request body builders |
| `internal/register/crypto.go` | `internal/facebook/web/register_crypto.go` | Password encryption |
| `internal/register/randomize.go` | `internal/facebook/web/register_randomize.go` | Fake info |
| `internal/register/android_token.go` | `internal/facebook/web/register_android_token.go` | Android Bloks login sau web B8 (lấy access_token) |
| `internal/verify/verify.go` | `internal/facebook/web/verify.go` | VerifyAccount orchestrator |
| `internal/verify/steps.go` | `internal/facebook/web/verify_steps.go` | B1-B5 |
| `internal/verify/body.go` | `internal/facebook/web/verify_body.go` | Body builders |
| `internal/verify/check.go` | `internal/facebook/web/verify_check.go` | CheckLiveDie + SaveToFolder |
| `internal/facebook/types.go` | `internal/facebook/types.go` | Session struct (giữ nguyên vị trí, bổ sung thêm RegInput, RegResult) |
| `internal/facebook/login.go` | `internal/facebook/login.go` | Login flow (giữ nguyên vị trí) |
| `internal/email/email.go` | `internal/email/service.go` | Rename: file định nghĩa Service interface |

### 3.3 Email — Temp vs Rent

| C# file | Go file | Loại |
|---------|---------|------|
| **API/TempMailServer/** | **email/temp/** | |
| `MoaktMailAPI.cs` | `temp/moakt.go` | Temp |
| `Mail1secComAPI.cs` | `temp/mail1sec.go` | Temp |
| `MohmalcomAPI.cs` | `temp/mohmal.go` | Temp |
| `Temporary_mailNetAPI.cs` | `temp/temporary_mail_net.go` | Temp |
| **API/RentMailServer/** | **email/rent/** | |
| `ZeusxAPI.cs` | `rent/zeus_x.go` | Rent |
| `DongvanfbAPI.cs` | `rent/dongvanfb.go` | Rent |
| `Store1sAPI.cs` | `rent/store1s.go` | Rent |
| `Mail30sAPI.cs` | `rent/mail30s.go` | Rent |

### 3.4 Proxy — Providers

| C# file | Go file |
|---------|---------|
| **API/TempProxyServer/** | **proxy/providers/** |
| `TinsoftProxy.cs` | `providers/tinsoft.go` |
| `ShoplikeProxy.cs` | `providers/shoplike.go` |
| `Netproxyio.cs` | `providers/netproxy.go` |
| `Kiotproxy.cs` | `providers/minproxy.go` |
| `DTProxy.cs` | `providers/proxyfarm.go` |

### 3.5 Orchestration / Automation

| C# file | Go file | Ghi chú |
|---------|---------|---------|
| `Automation/FacebookRegisterAutomation.cs` | `facebook/web/register_steps.go` | Logic nằm trong platform |
| `Automation/VerifyAccountAPIAutomation.cs` | `facebook/web/verify.go` | Logic nằm trong platform |
| `Automation/TempMailServiceAutomation.cs` | `email/factory.go` | Factory thay thế |
| `Automation/RentMailServiceAutomation.cs` | `email/factory.go` | Factory thay thế |
| `View/FMain.cs` (God Object) | `app.go` (thin) + `runner/scheduler.go` | Tách rõ |

### 3.6 Config / Singleton / Utils

| C# file | Go file | Ghi chú |
|---------|---------|---------|
| `Singleton/GlobalVariables.cs` | Không có | Truyền qua params |
| `Config/MainFormUISettings.cs` | `config/settings.go` + JSON | Load 1 lần |
| `Config/IniFile.cs` | Không cần | Go dùng JSON |
| `Utils/InstanceCreateUtils.cs` | `facebook/factory.go` + `email/factory.go` | Mỗi domain 1 factory |
| `Utils/FacebookRequestUtils.cs` | `facebook/helpers.go` | Shared helpers |
| `Utils/GetConfirmCodeFromMessageUtils.cs` | `email/service.go` | Mỗi email tự extract code |
| `Utils/ProxyServicesUtils.cs` | `proxy/manager.go` | Nằm trong proxy package |
| `Interfaces/IFacebookRegister.cs` | `facebook/interfaces.go` | `Registerer` interface |
| `Interfaces/IFacebookVerifyAPI.cs` | `facebook/interfaces.go` | `Verifier` interface |
| `Interfaces/ITempMailServerAPI.cs` | `email/service.go` | `Service` interface |
| `Interfaces/IRentMailServerAPI.cs` | `email/service.go` | Cùng `Service` interface |
| `Interfaces/ITempProxyServer.cs` | `proxy/provider.go` | `Provider` interface |

---

## 4. So sánh cấu trúc: C# cũ vs Go mới

### Facebook API

```
C# (VerifyCloneVIP)                    Go (HVR)
─────────────────────                  ─────────────────────
API/                                   facebook/
  FbAndroidApi/                          android/
    FacebookRegisterAPIAndroid.cs           register.go
    FacebookVerifyAndroidAPI.cs             verify.go
    FacebookInteractionAndroidAPI.cs        interaction.go
    FacebookFeedAndroidAPI.cs               feed.go
    FacebookSecurityFeatureAPIAndroid.cs     security.go

  FBWebApi/                              web/
    FacebookRegisterWebAndroidAPI.cs        register.go
    FacebookVerifyAPIRequest.cs             verify.go
    FacebookInteractionAPI.cs               interaction.go
    FacebookFeedAPI.cs                      feed.go
    FacebookSecurityFeatureAPI.cs            security.go
    FacebookAdsRequestAPI.cs                ads.go
    FacebookRegisterMfbChrome.cs         mfb/
    FacebookRegisterMfbRequest.cs           register.go

Interfaces/                            facebook/
  IFacebookRegister.cs                   interfaces.go   ← tất cả interface 1 file
  IFacebookVerifyAPI.cs                  interfaces.go
```

### Email

```
C# (VerifyCloneVIP)                    Go (HVR)
─────────────────────                  ─────────────────────
API/TempMailServer/                    email/temp/
  MoaktMailAPI.cs                        moakt.go
  Mail1secComAPI.cs                      mail1sec.go
  MohmalcomAPI.cs                        mohmal.go
  (~40 files, nhiều dead)                (chỉ giữ cái đang dùng)

API/RentMailServer/                    email/rent/
  ZeusxAPI.cs                            zeus_x.go
  DongvanfbAPI.cs                        dongvanfb.go
  Store1sAPI.cs                          store1s.go
  Mail30sAPI.cs                          mail30s.go

Interfaces/                            email/
  ITempMailServerAPI.cs                  service.go      ← 1 interface chung
  IRentMailServerAPI.cs                  service.go

Utils/                                 email/
  InstanceCreateUtils (copy-paste x2)    factory.go      ← 1 factory, 0 copy-paste
```

### Proxy

```
C# (VerifyCloneVIP)                    Go (HVR)
─────────────────────                  ─────────────────────
API/TempProxyServer/                   proxy/providers/
  TinsoftProxy.cs                        tinsoft.go
  ShoplikeProxy.cs                       shoplike.go
  Netproxyio.cs                          netproxy.go

API/IpLookupServer/                    proxy/
  AdsPowerAPI.cs                         checkip.go
  IpInfoIOAPI.cs
```

---

## 5. Quy tắc phát triển

### Thêm nền tảng Facebook mới

1. Tạo thư mục: `internal/facebook/<tên_nền_tảng>/`
2. Implement interface cần thiết: `Registerer`, `Verifier`, etc.
3. Thêm 1 case vào `facebook/factory.go`
4. Thêm constant `PlatformXxx` vào `facebook/factory.go`

**Tổng: 2 chỗ sửa** (thư mục mới + 1 case factory)

### Thêm chức năng cho nền tảng có sẵn (ví dụ: thêm Ads cho Android)

1. Tạo file: `internal/facebook/android/ads.go`
2. Implement interface `AdsManager`

**Tổng: 1 file mới** (nếu interface đã có)

### Thêm email provider mới

1. Xác định loại: temp hay rent
2. Tạo file: `email/temp/xxx.go` hoặc `email/rent/xxx.go`
3. Implement interface `email.Service`
4. Thêm 1 case vào `email/factory.go`

**Tổng: 2 chỗ sửa**

### Thêm proxy provider mới

1. Tạo file: `proxy/providers/xxx.go`
2. Implement interface `proxy.Provider`
3. Thêm 1 case vào `proxy/manager.go`

**Tổng: 2 chỗ sửa**

---

## 6. Các vấn đề cần sửa khi migration

| # | Vấn đề hiện tại | File | Cách sửa |
|---|-----------------|------|----------|
| 1 | `parseRegProxyURL()` duplicate | `register/init.go:260` | Xóa, dùng `proxy.CreateClient()` |
| 2 | `buildRegHTTPClient()` duplicate | `register/init.go:208` | Xóa, dùng `proxy.CreateClient()` |
| 3 | `createEmailService()` nằm sai chỗ | `verify/verify.go:309` | Chuyển sang `email/factory.go` |
| 4 | `verify.Config` phình flat fields | `verify/verify.go` | Dùng `email.Options` struct riêng |
| 5 | Proxy providers không có interface | `proxy/manager.go` | Thêm `proxy.Provider` interface |
| 6 | `app.go` tích lũy concerns | `app.go` | Giữ thin, delegate sang `runner/` |

---

## 7. Thứ tự migration (từ hiện tại → target)

Mỗi bước commit riêng, không break build. **Chỉ di chuyển file + update imports, không thay đổi logic.**

| Phase | Bước | Việc cần làm | Risk |
|-------|------|-------------|------|
| **A** | 1 | Tạo `facebook/interfaces.go` + `facebook/types.go` (move types từ register/, verify/) | Thấp |
| **A** | 2 | Tạo `facebook/factory.go` với constants PlatformWeb, PlatformAndroid | Thấp |
| **A** | 3 | Move `internal/register/*.go` → `internal/facebook/web/register_*.go` | Trung bình |
| **A** | 4 | Move `internal/verify/*.go` → `internal/facebook/web/verify_*.go` | Trung bình |
| **A** | 5 | Update `app.go`, `runner/scheduler.go` imports | Thấp |
| | | | |
| **B** | 6 | Tạo `email/factory.go` + `email/options.go` | Thấp |
| **B** | 7 | Move `email/moakt.go`, `mail1sec.go`... → `email/temp/` | Trung bình |
| **B** | 8 | Move `email/zeus_x.go`, `dongvanfb.go`... → `email/rent/` | Trung bình |
| **B** | 9 | Xóa `createEmailService()` trong verify, gọi `email.New()` | Thấp |
| | | | |
| **C** | 10 | Tạo `proxy/provider.go` interface | Thấp |
| **C** | 11 | Move `proxy/tinsoft.go`... → `proxy/providers/` | Trung bình |
| **C** | 12 | Refactor `proxy/manager.go` dùng Provider interface | Trung bình |
| | | | |
| **D** | 13 | Xóa duplicate `parseRegProxyURL()`, dùng `proxy.CreateClient()` | Thấp |

**Phase A, B, C có thể chạy song song** vì không share file.

---

## 8. Nguyên tắc thiết kế

1. **Mỗi nền tảng 1 thư mục.** `facebook/android/`, `facebook/web/`, `facebook/mfb/`.
2. **Interface ở package cha.** `facebook/interfaces.go` chứa `Registerer`, `Verifier`... Không mỗi nền tảng 1 interface.
3. **Factory ở package cha.** `facebook/factory.go`, `email/factory.go`. Không nằm trong `utils/`.
4. **Types dùng chung ở package cha.** `facebook/types.go` chứa `Session`, `RegInput`... Không mỗi nền tảng define lại.
5. **Config truyền qua params.** Không static global. Không đọc file trong getter.
6. **Callback thay cho UI coupling.** `func(string)` thay cho `DatagridViewFMainDTO`.
7. **1 provider = 1 file.** Dễ tìm, dễ xóa.
8. **Thêm mới = tối đa 2 chỗ sửa.** File mới + 1 case factory. Nếu hơn thì thiết kế sai.
9. **`app.go` chỉ bind Wails.** Nhận request frontend, gọi `internal/`, trả kết quả.
10. **Không dead code.** Không file Copy, không file GÓC, không commented-out blocks.
