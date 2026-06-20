# HVR Architecture V2 — Full Design Doc + Migration Plan

Generated: 2026-04-07
Based on: VerifyCloneVIP C# project analysis
Status: DRAFT

---

## 1. Muc tieu

Port toan bo cau truc tu VerifyCloneVIP (C#) sang HVR (Go),
gom: Register multi-platform, Verify multi-platform, Feed, Interaction,
Security, Ads, FakeInfo, FormData, HttpClient, IpLookup, MachineId, Stats.

Tat ca phai co skeleton/stub san de khi port code C# co cho dat ngay.

---

## 2. VerifyCloneVIP — Mapping Reference

### 2.1 Interfaces (11 cai)

| C# Interface | Go Equivalent | Package |
|---|---|---|
| `IFacebookRegister` | `Registerer` interface | `facebook/` (giu nguyen) |
| `IFacebookVerifyAPI` | `Verifier` interface | `facebook/` (giu nguyen) |
| `IUserAgentBuilder` | `UserAgentBuilder` interface | `facebook/fakeinfo/` |
| `IHttpRequestClient` | `HttpClient` interface | `httpclient/` |
| `IRentMailServerAPI` | `Service` interface (da co) | `email/` |
| `ITempMailServerAPI` | `Service` interface (da co) | `email/` |
| `ITempProxyServer` | `Provider` interface (da co) | `proxy/` |
| `IIPLookupAPI` | `Lookup` interface | `iplookup/` |
| `IMailServiceAutomation` | `MailAutomation` interface | `email/automation/` |
| `IFacebookMachineIdManager` | `MachineIdManager` struct | `facebook/machineid/` |
| `IStaticsReport` | `Reporter` interface | `stats/` |

### 2.2 Register Implementations (4 platforms)

| C# Class | Platform Key | Go Package |
|---|---|---|
| `FacebookRegisterMfbRequest` | `"web"` | `facebook/register/web/` (da co) |
| `FacebookRegisterWebAndroidAPI` | `"webandroid"` | `facebook/register/webandroid/` |
| `FacebookRegisterMfbChrome` | `"chrome"` | `facebook/register/chrome/` |
| `FacebookRegisterAPIAndroid` | `"android"` | `facebook/register/android/` |

### 2.3 Verify Implementations (4 platforms)

| C# Class | Platform Key | Go Package |
|---|---|---|
| `FacebookVerifyAPIRequestCustomHttpWrapper` | `"web"` | `facebook/verify/web/` (da co) |
| `FacebookVerifyWebAndroidAPI` | `"webandroid"` | `facebook/verify/webandroid/` |
| `FacebookVerifyAPIToken` | `"android"` | `facebook/verify/android/` |
| `FacebookVerifyAPIRequest` | `"webrequest"` | `facebook/verify/webrequest/` |

### 2.4 Feature APIs

| C# Class | Go Package |
|---|---|
| `FacebookFeedAPI` (web) | `facebook/feed/web/` |
| `FacebookFeedAndroidAPI` | `facebook/feed/android/` |
| `FacebookInteractionAPI` (web) | `facebook/interaction/web/` |
| `FacebookInteractionAndroidAPI` | `facebook/interaction/android/` |
| `FacebookSecurityFeatureAPI` (web) | `facebook/security/web/` |
| `FacebookSecurityWebAndroidAPI` | `facebook/security/webandroid/` |
| `FacebookSecurityFeatureAPIAndroid` | `facebook/security/android/` |
| `FacebookAdsRequestAPI` | `facebook/ads/` |
| `FacebookInitializeAPIRequest` | `facebook/init/` |

### 2.5 Support Components

| C# Component | Go Package |
|---|---|
| `FakePersonalInfoBuilder` | `facebook/fakeinfo/builder.go` |
| `AndroidUserAgentBuilder` | `facebook/fakeinfo/useragent_android.go` |
| `BrowserAndroidUserAgentBuilder` | `facebook/fakeinfo/useragent_browser.go` |
| `ConfigFileUserAgentBuilder` | `facebook/fakeinfo/useragent_config.go` |
| `FacebookApiFormDataBuilder` | `facebook/formdata/builder.go` |
| `FacebookApiHeaderCollectionBuilder` | `facebook/formdata/headers.go` |
| `FacebookRequestFormDataPropModel` | `facebook/formdata/props.go` |
| `FacebookMachineIdManager` | `facebook/machineid/manager.go` |
| `FacebookCheckpointDetectorUtils` | `facebook/checkpoint/detector.go` |
| `SecurityExtension` (encrypt pwd) | `facebook/crypto/encrypt.go` |

### 2.6 Models (C# → Go types)

| C# Model | Go Struct | Location |
|---|---|---|
| `FacebookAccountModel` (50+ fields) | `Session` (da co) | `facebook/types.go` |
| `FacebookApiCallResult.*` (17 result types) | Per-feature result structs | `facebook/results.go` |
| `FacebookApiCallOptions.*` (15 option types) | Per-feature option structs | `facebook/options.go` |
| `MailSessionModel` | `MailSession` | `email/types.go` |
| `MailServiceResultModel.*` | Result structs | `email/types.go` |
| `ProxyInfoModel` | Already in `proxy/` | `proxy/types.go` |
| `IpAddressInfo` | `IpInfo` | `iplookup/types.go` |
| `DeviceInfoXmlModel` | `DeviceInfo` | `facebook/fakeinfo/device.go` |
| `SimNetworkInfoModel` | `SimNetwork` | `facebook/fakeinfo/simnetwork.go` |
| `PhoneNumberCountryCodeModel` | `PhoneCountry` | `facebook/fakeinfo/phone.go` |
| `DatagridViewFMainDTO` | N/A (Wails frontend handles) | — |

### 2.7 Enums (C# → Go constants)

| C# Enum | Go Location |
|---|---|
| `FbApiCallStatusCode` | `facebook/status.go` |
| `AddEmailStatusCode` | `facebook/verify/status.go` |
| `ConfirmEmailStatusCode` | `facebook/verify/status.go` |
| `VeriAccountAutoStatusCode` | `facebook/verify/status.go` |
| `AccountFeaturesAutoStatusCode` | `facebook/security/status.go` |
| `Enable2FAStatusCode` | `facebook/security/status.go` |
| `ProxyApiKeyStatus` | `proxy/status.go` (da co) |
| `RentProxyStatusCode` | `proxy/status.go` |
| `LoginGmailStatusCode` | `email/status.go` |
| `MailServerAutomationInstanceType` | `email/automation/types.go` |

### 2.8 Email/Proxy Providers Count

| Category | C# Count | Go Count (hien tai) | Target |
|---|---|---|---|
| Temp Mail | 51 | 4 | 51 (port dan) |
| Rent Mail | 14 | 4 | 14 (port dan) |
| Temp Proxy | 7 | 5 | 7 |
| Rotating Proxy | 1 | 0 | 1 |
| IP Lookup | 4 | 0 | 4 |

---

## 3. Target Go Structure

```
internal/
  facebook/
    types.go                    ← Session, LoginResult (KEEP)
    interfaces.go               ← ALL interfaces: Registerer, Verifier, Interactor,
    |                              FeedReader, SecurityManager (KEEP - avoid circular import)
    factory.go                  ← Plugin registration factory (KEEP)
    login.go                    ← Login logic (KEEP)
    options.go                  ← NEW: FacebookApiCallOptions equivalents
    results.go                  ← NEW: FacebookApiCallResult equivalents
    status.go                   ← NEW: FbApiCallStatusCode constants
    constants.go                ← NEW: FbAppVersionsConstants, UrlSingleton equivalents

    register/                   ← TACH TU facebook/web/register_*
      web/                      ← MOVE tu facebook/web/register_*.go
        init.go
        steps.go
        body.go
        crypto.go
        randomize.go            ← CHI GIU phan dung cho web reg
        android_token.go
        register.go             ← Registerer struct + init() registration
      android/                  ← NEW SKELETON
        register.go
      chrome/                   ← NEW SKELETON
        register.go
      webandroid/               ← NEW SKELETON
        register.go

    verify/                     ← TACH TU facebook/web/verify_*
      check.go                  ← MOVE tu verify/check.go (CheckLiveDie, SaveAccount)
      status.go                 ← NEW: AddEmail/ConfirmEmail status codes
      web/                      ← MOVE tu facebook/web/verify_*.go
        verify.go
        steps.go
        body.go
        verify_impl.go          ← Verifier struct + init() registration
      android/                  ← NEW SKELETON
        verify.go
      webandroid/               ← NEW SKELETON
        verify.go
      webrequest/               ← NEW SKELETON
        verify.go

    feed/                       ← NEW
      interfaces.go             ← FeedReader (move from facebook/interfaces.go)
                                   ... giu lai o facebook/interfaces.go de tranh circular
      web/
        feed.go                 ← SKELETON: GetFeedStories, LikeStory, CommentStory, ShareStory, CreateStory
      android/
        feed.go                 ← SKELETON: SharePost, FollowPage, GetVideoInfo

    interaction/                ← NEW
      web/
        interact.go             ← SKELETON: Follow
      android/
        interact.go             ← SKELETON: AddWork, AddHighSchool, AddCity, UploadCover, UploadAvatar

    security/                   ← NEW
      status.go                 ← AccountFeaturesAutoStatusCode, Enable2FAStatusCode
      web/
        security.go             ← SKELETON: ConfirmSubEmail, AddSubEmail, UploadAvatar, TurnOn2FA, ProfilePlus
      webandroid/
        security.go             ← SKELETON: TurnOn2FA, ConfirmTwoStepVerificationEmail
      android/
        security.go             ← SKELETON: DeactivateAccount, TurnOn2FA

    ads/                        ← NEW
      ads.go                    ← SKELETON: GetAdsAccount, AcceptAdsPolicy

    init/                       ← NEW
      init.go                   ← SKELETON: GetFormDataProperties

    fakeinfo/                   ← NEW (tu FakeInfoBuilder/)
      builder.go                ← RandomFirstName, RandomLastName, RandomPhoneNumber
      useragent.go              ← UserAgentBuilder interface
      useragent_android.go      ← AndroidUserAgentBuilder
      useragent_browser.go      ← BrowserAndroidUserAgentBuilder
      useragent_config.go       ← ConfigFileUserAgentBuilder
      device.go                 ← DeviceInfoXmlModel, LoadDeviceXmlFiles, GetRandomDevice
      simnetwork.go             ← SimNetworkInfoModel, LoadSimNetwork, GetRandom
      phone.go                  ← PhoneNumberCountryCode, LoadPhoneCodes, GenerateByCountry

    formdata/                   ← NEW (tu FacebookApiFormDataBuilder/)
      builder.go                ← Build form data cho cac API calls
      headers.go                ← Header collection builder (20+ header sets)
      props.go                  ← FormDataPropModel + builder methods

    machineid/                  ← NEW (tu FacebookMachineIdManager)
      manager.go                ← ConcurrentMap, AddOrIncrement, Remove, GetCount, Load

    checkpoint/                 ← NEW (tu FacebookCheckpointDetectorUtils)
      detector.go               ← IsCheckpointRequired, IsBlocked, IsLogoutSession

    crypto/                     ← MOVE tu register/web/crypto.go (shared giua platforms)
      encrypt.go                ← GetEncryptedPassword (AES-GCM + NaCl SealedBox)

  email/
    service.go                  ← Service interface (KEEP)
    factory.go                  ← Factory (KEEP)
    options.go                  ← Options (KEEP)
    pool.go                     ← CredPool re-export (KEEP for now)
    status.go                   ← NEW: LoginGmailStatusCode
    automation/                 ← NEW (tu IMailServiceAutomation)
      automation.go             ← MailAutomation interface + orchestration
      types.go                  ← MailSession, result types
    temp/
      moakt.go                  ← (da co)
      mail1sec.go               ← (da co)
      mohmal.go                 ← (da co)
      temporary_mail_net.go     ← (da co)
      ... (them 47 provider stubs dan)
    rent/
      zeus_x.go                 ← (da co)
      dongvanfb.go              ← (da co)
      store1s.go                ← (da co)
      mail30s.go                ← (da co)
      pool.go                   ← (da co)
      ... (them 10 provider stubs dan)

  proxy/
    provider.go                 ← Provider interface (KEEP)
    client.go                   ← CreateClient (KEEP)
    pool.go                     ← Pool (KEEP)
    manager.go                  ← Manager (KEEP)
    checkip.go                  ← (KEEP)
    health.go                   ← (KEEP)
    status.go                   ← NEW: ProxyApiKeyStatus, RentProxyStatusCode
    providers/
      tinsoft.go                ← (da co)
      shoplike.go               ← (da co)
      netproxy.go               ← (da co)
      minproxy.go               ← (da co)
      proxyfarm.go              ← (da co)
      shared.go                 ← (da co)
      rotating/                 ← NEW SKELETON
        proxyv6.go

  httpclient/                   ← NEW (tu IHttpRequestClient)
    client.go                   ← Configurable HTTP client: cookies, headers, redirect control
    types.go                    ← CustomHttpResponse, CustomHttpRequestConfig

  iplookup/                     ← NEW (tu IIPLookupAPI + 4 implementations)
    interfaces.go               ← Lookup interface
    types.go                    ← IpInfo struct
    ipinfo.go                   ← IpInfoIO implementation
    luna.go                     ← Luna implementation
    nordvpn.go                  ← NordVPN implementation
    adspower.go                 ← AdsPower implementation

  stats/                        ← NEW (tu IStaticsReport)
    reporter.go                 ← Reporter interface + BasicReporter struct
```

---

## 4. Circular Import Prevention

**Rule:** `facebook/` package KEEPS all shared interfaces and types.
Child packages (`register/web/`, `verify/android/`, etc.) import `facebook/` for types.
`facebook/factory.go` uses plugin registration (init() pattern), NEVER imports child packages.

```
facebook/                  ← interfaces + types + factory (registry)
  ↑                          NO imports of child packages
  |
  ├── register/web/        ← imports facebook/ for types
  ├── register/android/    ← imports facebook/ for types
  ├── verify/web/          ← imports facebook/ for types
  ├── verify/android/      ← imports facebook/ for types
  └── ...

app.go                     ← blank imports to trigger init()
  _ "HVR/internal/facebook/register/web"
  _ "HVR/internal/facebook/verify/web"
```

---

## 5. Migration Plan — Phases

### Phase E — Restructure facebook/register/ (tach tu facebook/web/)

**Muc tieu:** Move `facebook/web/register_*.go` → `facebook/register/web/`

**Steps:**
1. Tao thu muc `internal/facebook/register/web/`
2. Move files (doi package name `web` → `web`... giu nguyen vi cung la `package web`):
   - `facebook/web/register_init.go` → `facebook/register/web/init.go`
   - `facebook/web/register_steps.go` → `facebook/register/web/steps.go`
   - `facebook/web/register_body.go` → `facebook/register/web/body.go`
   - `facebook/web/register_crypto.go` → `facebook/register/web/crypto.go`
   - `facebook/web/register_randomize.go` → `facebook/register/web/randomize.go`
   - `facebook/web/register_android_token.go` → `facebook/register/web/android_token.go`
   - `facebook/web/register.go` → `facebook/register/web/register.go` (Registerer struct + init())
3. Update `facebook/web/types.go`: xoa register type aliases (giu verify type aliases)
4. Update `app.go` blank import: `_ "HVR/internal/facebook/register/web"`
5. Build pass
6. Commit: `refactor(facebook): move register into facebook/register/web/`

### Phase F — Restructure facebook/verify/ (tach tu facebook/web/)

**Muc tieu:** Move `facebook/web/verify_*.go` → `facebook/verify/web/`

**Steps:**
1. Tao thu muc `internal/facebook/verify/web/`
2. Move files:
   - `facebook/web/verify_verify.go` → `facebook/verify/web/verify.go`
   - `facebook/web/verify_steps.go` → `facebook/verify/web/steps.go`
   - `facebook/web/verify_body.go` → `facebook/verify/web/body.go`
   - `facebook/web/verify_check.go` → `facebook/verify/web/check.go`
   - `facebook/web/verify.go` → `facebook/verify/web/verify_impl.go` (Verifier struct + init())
3. Move `verify/check.go` utility functions (SaveAccountToFolder, CheckLiveDie) → `facebook/verify/check.go`
   - Update `runner/scheduler.go` import: `verify.SaveAccountToFolder` → `fbverify.SaveAccountToFolder`
4. Xoa `facebook/web/types.go` (khong con can)
5. Xoa thu muc `facebook/web/` (rong)
6. Update `app.go` blank import: `_ "HVR/internal/facebook/verify/web"`
7. Build pass
8. Commit: `refactor(facebook): move verify into facebook/verify/web/`

### Phase G — Add skeleton platform packages

**Muc tieu:** Tao stub cho tat ca platforms chua co, de khi port C# code co cho san.

**Steps:**
1. Tao `facebook/register/android/register.go`:
   ```go
   package android
   // TODO: Port from VerifyCloneVIP FacebookRegisterAPIAndroid.cs
   ```
2. Tao `facebook/register/chrome/register.go`
3. Tao `facebook/register/webandroid/register.go`
4. Tao `facebook/verify/android/verify.go`
5. Tao `facebook/verify/webandroid/verify.go`
6. Tao `facebook/verify/webrequest/verify.go`
7. Moi file co: `package <name>`, comment TODO, va stub struct
8. Build pass
9. Commit: `chore(facebook): add skeleton platform packages`

### Phase H — Add feature API skeletons

**Muc tieu:** Tao stub packages cho Feed, Interaction, Security, Ads, Init.

**Steps:**
1. `facebook/feed/web/feed.go` — stub methods: GetFeedStories, LikeStory, CommentStory, ShareStory, CreateStory
2. `facebook/feed/android/feed.go` — stub methods: SharePost, FollowPage, GetVideoInfo
3. `facebook/interaction/web/interact.go` — stub: Follow
4. `facebook/interaction/android/interact.go` — stub: AddWork, AddHighSchool, AddCity, UploadCover, UploadAvatar
5. `facebook/security/web/security.go` — stub: ConfirmSubEmail, AddSubEmail, UploadAvatar, TurnOn2FA, ProfilePlus
6. `facebook/security/webandroid/security.go` — stub: TurnOn2FA, ConfirmTwoStepVerification
7. `facebook/security/android/security.go` — stub: DeactivateAccount, TurnOn2FA
8. `facebook/security/status.go` — AccountFeaturesAutoStatusCode, Enable2FAStatusCode constants
9. `facebook/ads/ads.go` — stub: GetAdsAccount, AcceptAdsPolicy
10. `facebook/init/init.go` — stub: GetFormDataProperties
11. Build pass
12. Commit: `chore(facebook): add feed/interaction/security/ads skeletons`

### Phase I — Add FakeInfo + FormData + support packages

**Muc tieu:** Tao cac support packages tuong ung voi VerifyCloneVIP.

**Steps:**
1. `facebook/fakeinfo/builder.go` — RandomFirstName, RandomLastName, RandomPhoneNumber
   - Move logic tu `facebook/register/web/randomize.go` (hoac giu nguyen va re-export)
2. `facebook/fakeinfo/useragent.go` — UserAgentBuilder interface:
   ```go
   type UserAgentBuilder interface {
       GetUserAgent(locale string, addVirtualSpecs bool, useBuildNumFile bool, simBrand string) string
   }
   ```
3. `facebook/fakeinfo/useragent_android.go` — AndroidUserAgentBuilder stub
4. `facebook/fakeinfo/useragent_browser.go` — BrowserAndroidUserAgentBuilder stub
5. `facebook/fakeinfo/useragent_config.go` — ConfigFileUserAgentBuilder stub
6. `facebook/fakeinfo/device.go` — DeviceInfo struct + LoadDeviceFiles + GetRandom
7. `facebook/fakeinfo/simnetwork.go` — SimNetwork struct + LoadSimNetwork + GetRandom
8. `facebook/fakeinfo/phone.go` — PhoneCountry struct + LoadPhoneCodes + GenerateByCountry
9. `facebook/formdata/builder.go` — Form data builder stub
10. `facebook/formdata/headers.go` — Header collection builder stub
11. `facebook/formdata/props.go` — FormDataPropModel struct + builder methods
12. `facebook/machineid/manager.go` — MachineIdManager with ConcurrentMap
13. `facebook/checkpoint/detector.go` — IsCheckpointRequired, IsBlocked
14. `facebook/crypto/encrypt.go` — Move shared crypto tu register/web/crypto.go
15. Build pass
16. Commit: `chore(facebook): add fakeinfo/formdata/machineid/checkpoint packages`

### Phase J — Add infrastructure packages (httpclient, iplookup, stats)

**Muc tieu:** Tao infrastructure packages moi tuong ung VerifyCloneVIP.

**Steps:**
1. `httpclient/client.go`:
   ```go
   type Client interface {
       Get(url string) (*Response, error)
       PostForm(url string, data map[string]string) (*Response, error)
       PostJSON(url string, body []byte, contentType string) (*Response, error)
       AddCookie(cookie *http.Cookie)
       GetCookies(url string) []*http.Cookie
       SetHeader(key, value string)
       SetFollowRedirects(follow bool)
       SetTimeout(d time.Duration)
       Close() error
   }
   ```
2. `httpclient/types.go` — Response, RequestConfig structs
3. `iplookup/interfaces.go` — Lookup interface: `GetIpInfo(client *http.Client) (*IpInfo, error)`
4. `iplookup/types.go` — IpInfo struct { IpAddress, CountryCode string }
5. `iplookup/ipinfo.go` — IpInfoIO stub
6. `iplookup/luna.go` — Luna stub
7. `iplookup/nordvpn.go` — NordVPN stub
8. `iplookup/adspower.go` — AdsPower stub
9. `stats/reporter.go` — Reporter interface + BasicReporter:
   ```go
   type Reporter interface {
       IncrTotal()
       IncrSuccess()
       IncrFailure()
       IncrError()
       SuccessRate() float64
   }
   ```
10. Build pass
11. Commit: `chore: add httpclient/iplookup/stats infrastructure packages`

### Phase K — Add extended types (options, results, status codes)

**Muc tieu:** Them day du types/options/results tuong ung FacebookApiCallResult + Options.

**Steps:**
1. `facebook/status.go` — FbApiCallStatusCode constants
2. `facebook/options.go` — Option structs cho moi API call:
   - BaseOptions, AddEmailOptions, ConfirmEmailOptions, TurnOnTwofactorOptions,
   - UploadAvatarOptions, FollowOptions, CreateStoryOptions, etc.
3. `facebook/results.go` — Result structs cho moi API call:
   - BaseResult, RegisterResult (da co), AddEmailResult, ConfirmEmailResult,
   - TurnOnTwofactorResult, UploadAvatarResult, FeedStoriesResult, etc.
4. `facebook/constants.go` — FbLatestVer, FbOAuthToken, URLs
5. `facebook/verify/status.go` — AddEmailStatusCode, ConfirmEmailStatusCode, VeriAccountAutoStatusCode
6. `email/automation/automation.go` — MailAutomation interface:
   ```go
   type MailAutomation interface {
       CreateOrBuySession(ctx context.Context, username string, timeout time.Duration) (*MailSession, error)
       LoginIfRequired(ctx context.Context) error
       LookupMessages(ctx context.Context, timeout time.Duration) (string, error)
       LookupOTPs(rawMessages string, caseType OTPCase) ([]string, error)
       Close() error
   }
   ```
7. `email/automation/types.go` — MailSession, OTPCase, result types
8. Build pass
9. Commit: `chore(facebook): add options/results/status types`

### Phase L — Cleanup: xoa old register/, verify/, email stubs

**Muc tieu:** Xoa redirect layers va stale code.

**Steps:**
1. Update `app.go`:
   - Thay `register.RegisterAccount()` → goi qua factory hoac import truc tiep `facebook/register/web`
   - Thay `register.RandomRegInput()` → `fakeinfo.RandomRegInput()` hoac tuong tu
   - Thay `register.GeneratePhoneByCountry()` → `fakeinfo.GeneratePhoneByCountry()`
   - Thay `verify.Config` → `facebook.VerifyConfig`
   - Thay `verify.SaveAccountToFolder()` → `fbverify.SaveAccountToFolder()`
   - Xoa import `"HVR/internal/register"` va `"HVR/internal/verify"`
   - Thay `email.NewZeusXPool()` → `rent.NewZeusXPool()` (import `email/rent`)
   - Thay `email.NewStore1sPool()` → `rent.NewStore1sPool()`
   - Thay `email.NewMail30sPool()` → `rent.NewMail30sPool()`
2. Update `cmd/regtest/main.go` tuong tu
3. Update `cmd/verifytest/main.go` tuong tu
4. Update `cmd/emailtest/main.go` tuong tu
5. Update `runner/scheduler.go`:
   - `verify.VerifyAccount()` → factory hoac direct import
   - `verify.SaveAccountToFolder()` → `fbverify.SaveAccountToFolder()`
6. Xoa `internal/register/` (toan bo)
7. Xoa `internal/verify/` (toan bo, tru check_test.go → move)
8. Xoa `internal/email/email.go` (empty deprecated stub)
9. Xoa email root stubs: `moakt.go`, `mail1sec.go`, `mohmal.go`, `temporary_mail_net.go`,
   `dongvanfb_mail.go`, `store1s_mail.go`, `mail30s_mail.go`, `zeus_x.go`
10. Xoa `internal/email/pool.go` (re-export) → callers import `email/rent` truc tiep
11. Build pass + test pass
12. Commit: `refactor: remove legacy register/verify/email redirect layers`

---

## 6. Phase Priority & Dependencies

```
Phase E ─────→ Phase F ─────→ Phase L (cleanup)
  |               |
  └── Phase G ────┘  (skeleton platforms, phu thuoc E+F xong)
        |
        ├── Phase H  (feature API skeletons, doc lap)
        ├── Phase I  (fakeinfo/formdata/support, doc lap)
        ├── Phase J  (infrastructure, doc lap)
        └── Phase K  (types/options/results, doc lap)
```

**Phases co the chay song song:** H, I, J, K (sau khi G xong)
**Phases phai tuan tu:** E → F → G → L

---

## 7. Effort Estimate

| Phase | Mo ta | So files | Do kho |
|---|---|---|---|
| E | Move register → register/web/ | ~8 files | Trung binh (update imports) |
| F | Move verify → verify/web/ | ~6 files | Trung binh |
| G | Skeleton platforms | ~6 stub files | Thap |
| H | Feature API skeletons | ~10 stub files | Thap |
| I | FakeInfo + FormData + support | ~15 files | Trung binh (move + stub) |
| J | Infrastructure (httpclient/iplookup/stats) | ~10 files | Thap |
| K | Types/Options/Results/Status | ~8 files | Trung binh (nhieu types) |
| L | Cleanup old code | ~20 files xoa/update | Cao (update app.go + scheduler.go) |

**Tong:** ~80 files create/move/update

---

## 8. VerifyCloneVIP Method Signatures Reference

### IFacebookRegister Methods → Go

```go
// C#: Register(FacebookAccountModel, IHttpRequestClient, string deviceid, string waterfall_id) → RegisterResult
// Go:
type Registerer interface {
    Register(ctx context.Context, input *RegInput, onStatus func(string)) *RegResult
}
```

### IFacebookVerifyAPI Methods → Go

```go
// C#: AddEmail(AddEmailOptions) → AddEmailResult
// C#: ConfirmEmail(ConfirmEmailOptions) → ConfirmEmailResult
// Go:
type Verifier interface {
    Verify(ctx context.Context, session *Session, cfg *VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *VerifyResult
}
// Note: AddEmail + ConfirmEmail wrapped inside Verify() orchestration
```

### IUserAgentBuilder → Go

```go
type UserAgentBuilder interface {
    GetUserAgent(locale string, addVirtualSpecs bool, useBuildNumFile bool, simBrand string) string
}
```

### IMailServiceAutomation → Go

```go
type MailAutomation interface {
    CreateOrBuySession(ctx context.Context, username string, timeout time.Duration) (*MailSession, error)
    LoginIfRequired(ctx context.Context) error
    LookupMessages(ctx context.Context, timeout time.Duration) (string, error)
    LookupOTPs(rawMessages string, caseType OTPCase) ([]string, error)
    Close() error
}
```

### IHttpRequestClient → Go

```go
type HttpClient interface {
    Get(url string) (*Response, error)
    PostForm(url string, data map[string]string) (*Response, error)
    PostRaw(url string, body []byte, contentType string) (*Response, error)
    AddCookie(cookie *http.Cookie)
    GetCookies(url string) []*http.Cookie
    SetHeader(key, value string)
    ResponseHeaders() map[string]string
    SetFollowRedirects(follow bool)
    SetTimeout(d time.Duration)
    Close() error
}
```

---

## 9. Ghi chu

- **Khong thay doi logic** khi move file. Logic port tu C# lam rieng sau.
- **Moi phase = 1 commit** (hoac nhom nho commits).
- Skeleton files chi can: `package name`, struct stub, comment TODO voi ten file C# tuong ung.
- `facebook/interfaces.go` GIU NGUYEN o parent package de tranh circular imports.
- `factory.go` GIU NGUYEN plugin registration pattern.
- Email/Proxy providers port DAN DAN, khong can lam het 1 luc.
- `app.go` la file risk cao nhat (Phase L) — test ky truoc khi xoa redirect layers.
