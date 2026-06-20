# Forms & UI Reference — HVR (Hạ Vũ Clone)

> Mô tả đầy đủ toàn bộ form, nút bấm, logic tương tác của mọi trang trong dự án.  
> Căn cứ trực tiếp từ source code — không phỏng đoán.

---

## Mục lục

1. [Accounts — Toolbar & Grid](#1-accounts--toolbar--grid)
2. [Accounts — Import Dialog](#2-accounts--import-dialog)
3. [Accounts — Detail Panel](#3-accounts--detail-panel)
4. [Nguồn xác thực (AuthSource)](#4-nguồn-xác-thực-authsource)
5. [Thiết lập chạy (Interaction Setup)](#5-thiết-lập-chạy-interaction-setup)
6. [Cài đặt chung (General Settings)](#6-cài-đặt-chung-general-settings)
7. [Proxy Settings](#7-proxy-settings)
8. [Hiển thị (View Settings)](#8-hiển-thị-view-settings)
9. [Tương tác](#9-tương-tác)
10. [Đẩy tài khoản (Upload Site)](#10-đẩy-tài-khoản-upload-site)
11. [Thống kê (RegStats)](#11-thống-kê-regstats)
12. [Flow Settings](#12-flow-settings)
13. [Import cấu hình WeBM (LegacyImportWizard)](#13-import-cấu-hình-webm-legacyimportwizard)

---

## 1. Accounts — Toolbar & Grid

**File**: `modules/accounts/components/AccountsToolbar.vue` + `pages/AccountsPage.vue`

### Thiết kế Toolbar

```
[ ▶ Chạy  File ] ─────────── [ 📁 Result ] [ ⋮ Columns ] [ ⚙ Cài đặt ] | [ + Import ] [ 🗑 Xóa (n) ] | [ 🔍 Lọc... ] [↻]
```

### Nút bấm Toolbar

| Nút | Màu / Style | Enabled khi | Hành động |
|---|---|---|---|
| **▶ Chạy** | Xanh lá gradient (`#43a047→#2e7d32`) | `cloneHvMode=true` hoặc `fileMode=true` hoặc `selectedCount > 0` | Emit `run` → AccountsPage gọi backend RunRegister/RunVerify |
| Badge **API** / **File** trên nút Chạy | Trắng mờ, nhỏ | Luôn | Hiển thị mode hiện tại (CloneHV API hoặc File) |
| **■ Dừng** | Đỏ (`#dc2626`), shadow | Khi `isRunning=true` | Emit `stop` → AccountsPage gọi backend StopVerify |
| **⏳ Đang dừng...** | Cam (`#d97706`), spinner | Khi `isStopping=true` | Disabled — chỉ hiển thị trạng thái |
| **📁 Result** | Viền mặc định, chữ xanh lá | Luôn | Nếu có `resultFolderPath` → mở Explorer; nếu không → chọn thư mục |
| **⋮ Columns** | Viền mặc định, chữ tím (`#a78bfa`) | Luôn | Emit `toggle-columns` → mở/tắt column visibility panel |
| **⚙ Cài đặt** | Viền mặc định, chữ vàng (`#fbbf24`) | Luôn | Emit `settings` → mở settings modal/drawer |
| **+ Import** | Viền xanh brand, chữ xanh | Luôn | Emit `import` → mở ImportDialog |
| **🗑 Xóa (n)** | Viền đỏ nhạt, chữ đỏ | `selectedCount > 0` | Emit `delete` → confirm → xóa accounts đã chọn |
| **↻** (refresh) | Icon nhỏ, màu muted | Luôn | `location.reload()` — làm mới giao diện (fix WebView2 glitch sau nhiều giờ chạy). Nếu đang chạy: hiện `window.confirm` cảnh báo trước |
| **Input lọc** | `width:250px`, placeholder "Lọc theo UID, Email, Note..." | Luôn | Emit `update:filterKeyword` realtime khi gõ |

### Logic Run / Stop

```
Trạng thái hiển thị theo thứ tự ưu tiên:
  isStopping=true  → nút "⏳ Đang dừng..." (disabled)
  isRunning=false  → nút "▶ Chạy" (disabled nếu không có gì để chạy)
  isRunning=true   → nút "■ Dừng" (enabled)
```

### Grid — Account Table

**Columns mặc định hiển thị** (theo `ACCOUNT_COLUMNS` + `preferences.isColumnVisible`):

| Key | Label | Default visible | Ghi chú |
|---|---|---|---|
| id | STT | ✓ | Row index tự động |
| uid | UID | ✓ | Platform user ID, màu xanh brand |
| fullData | Dữ liệu gốc | — | Raw string import |
| password | Mật khẩu | ✓ | Có thể mask |
| twofa | 2FA | — | |
| email | Email | ✓ | |
| passMail | Pass Mail | — | |
| mailRecovery | Mail Khôi Phục | — | |
| cookie | Cookie | — | Có thể mask |
| token | Token | — | Có thể mask |
| status | Trạng thái | ✓ | Badge màu: live=xanh, die=đỏ, checkpoint=cam, new=xám, unknown=xám |
| checkpoint | Checkpoint | — | |
| statusAds | Quảng cáo | — | |
| bm | BM | — | Business Manager |
| tkqc | TKQC | — | Tài khoản quảng cáo |
| chatSupport | Chat Support | — | |
| fullName | Họ tên | — | |
| location | Vị trí | — | |
| avatar | Avatar | — | URL |
| cover | Cover | — | URL |
| phone | Phone | — | |
| proxy | Proxy | ✓ | |
| userAgent | UA | — | |
| note | Ghi chú | — | |
| noteRun | Ghi chú chạy | — | |
| importTime | Ngày nhập | — | |
| category | Thư mục | — | |
| lastRun | Chạy lần cuối | — | |
| runProxy | IP Chạy | ✓ | IP realtime trong lúc verify/reg |
| activity | Hoạt động | ✓ | Trạng thái action hiện tại |
| sourceCode | Source | — | |

### Grid Features

- **Checkbox chọn hàng**: click checkbox cột đầu hoặc click vào row để highlight
- **Dual selection**: "highlighted" (click row) vs "checked" (click checkbox) — 2 state khác nhau
- **Sort**: click header → asc/desc
- **Sticky header**: header không scroll theo body
- **Status bar dưới**: `Live: n | Die: n | Unknown: n | Bỏ đen: n | Đã chọn: n`
- **Context menu** (right-click):

```
┌────────────────────────┐
│  Copy UID              │
│  Copy Full Data        │
│  Copy Password         │
│  Copy Email            │
│  Copy Cookie           │
│  ─────────────────     │
│  Chạy account này      │
│  Verify account này    │
│  ─────────────────     │
│  Xem chi tiết          │
│  Sửa ghi chú           │
│  ─────────────────     │
│  Xóa                   │
└────────────────────────┘
```

---

## 2. Accounts — Import Dialog

**File**: `modules/accounts/components/AccountsImportDialog.vue`

### Thiết kế

Modal 3-in-1: 3 phương thức import độc lập trong 1 dialog.

```
┌──────────────────────────────────────────────────────┐
│  Import Accounts                                 [×] │
├──────────────────────────────────────────────────────┤
│                                                      │
│  § THÊM TỪ THƯ MỤC (persistent)                    │
│  Đường dẫn: [__________________] [Chọn thư mục]     │
│                                  [Làm mới từ thư mục]│
│  ✓ Đã thêm 45 account từ D:/acc/                    │
│                                                      │
│  ─────────────────────────────────────────────────  │
│                                                      │
│  § THÊM TỪ 1 FILE (persistent)                     │
│  [__________________] [Chọn file .txt]              │
│                                                      │
│  ─────────────────────────────────────────────────  │
│                                                      │
│  § NHẬP TAY / PASTE (one-time)                     │
│  ┌────────────────────────────────┐                  │
│  │ uid|pass|email|passmail|...   │                  │
│  │ uid|pass|email|...            │                  │
│  └────────────────────────────────┘                  │
│                                                      │
│                           [Hủy]  [✓ Import (n dòng)]│
└──────────────────────────────────────────────────────┘
```

### Các nút & logic

| Nút | Hành động | Logic |
|---|---|---|
| **Chọn thư mục** | `OpenFolderDialog()` → `SetAccountSourceFolder(path)` | Lưu path vào backend; backend quét toàn bộ `.txt` trong thư mục → trả về `{ imported: n }` |
| **Làm mới từ thư mục** | `RefreshAccountSource()` | Rescan thư mục đã chọn; chỉ thêm account chưa có trong DB |
| **Chọn file .txt** | `OpenFileDialogPath()` → `LoadAccountsFromFile(path)` | Pick 1 file; backend load toàn bộ vào grid; backend nhớ path để xóa dòng sau verify xong |
| **Import (n dòng)** | Emit `import(data)` | Chỉ active khi có text trong textarea. Backend parse dạng `uid\|pass\|email\|...` |
| **Hủy / ×** | Emit `close` | Đóng dialog |

**Trạng thái**:
- Khi `loading=true`: nút Import spinner + disabled
- Folder status: `type='success'` → chữ xanh; `type='error'` → chữ đỏ
- `onMounted`: auto-load `GetAccountSourceFolder()` để hiện path đã cấu hình sẵn; `ValidatePath()` kiểm tra folder còn tồn tại không

---

## 3. Accounts — Detail Panel

**File**: `modules/accounts/components/AccountsDetailPanel.vue`

### Thiết kế

Side panel slide-in từ phải, hiển thị toàn bộ trường của 1 account.

```
┌─────────────────────────────────────────┐
│  Chi tiết tài khoản             [Đóng] │
├─────────────────────────────────────────┤
│  UID:        [123456789]    [📋 Copy]   │
│  Mật khẩu:  [abc123]       [📋 Copy]   │
│  Email:      [a@b.com]      [📋 Copy]   │
│  Cookie:     [eyJ...]       [📋 Copy]   │
│  Token:      [...]          [📋 Copy]   │
│  Trạng thái: ● live                     │
│  Proxy:      [1.2.3.4:8080]            │
│  ...                                    │
│                                         │
│  [Ghi chú]                              │
│  [____________________________________] │
│                                [Lưu ghi chú]│
└─────────────────────────────────────────┘
```

---

## 4. Nguồn xác thực (AuthSource)

**File**: `pages/AuthSourcePage.vue` + `modules/auth-source/components/AuthSourcePanel.vue`

### Thiết kế Header trang

```
[← Thiết lập chạy]  Nguồn xác thực  Cấu hình mail/phone provider. Auto-save.  [● Tự động lưu]
```

- **[← Thiết lập chạy]**: Button quay lại route `/interaction-setup`
- **Auto-save status**: `idle` → "• Tự động lưu" | `saving` → "◑ Đang lưu..." | `saved` → "✔ Đã lưu" (xanh) | `error` → "⚠ Lỗi lưu" (đỏ)
- **Debounce**: 500ms — lưu ngay khi form thay đổi

---

### Tab chính: Mail / Phone

```
[ Mail ]  [ Phone ]
```

---

### Tab MAIL

#### Bước 1: Chọn danh mục

```
[ Temp Mail ]  [ Rent Mail ]     ☐ Kiểm tra Live / Die
```

- **Temp Mail**: chọn provider không mất tiền / tự xoay vòng
- **Rent Mail**: chọn provider mua OTP/Gmail có trả phí
- **Checkbox "Kiểm tra Live / Die"**: bind `form.checkLiveDieEnabled` — bật để verify sau khi reg

#### Bước 2: Chọn Provider (SearchableSelect — có tìm kiếm)

**Temp Mail Providers (26 provider):**

| Value | Label |
|---|---|
| `moakt` | Moakt |
| `@i2b.vn` | Mail1sec |
| `mohmal` | Mohmal |
| `tempmail-lol` | TempMail LOL |
| `mailtm` | Mail.tm |
| `tempmail-plus` | TempMail.plus |
| `dropmail` | Dropmail |
| `guerrillamail` | GuerrillaMail |
| `spam4me` | Spam4.me |
| `mail.cx` | Mail.cx |
| `inboxes` | Inboxes.com |
| `onesecmail` | 1secmail.com |
| `fviainboxes` | FviaInboxes.com |
| `mailermnx` | Mailer.mnx-family.com |
| `tempemail` | TempEmail.co |
| `tempmailto` | TempMailTo.com |
| `onesecemail` | 1secemail.com |
| `tempmail100` | TempMail100.com |
| `tempmailso` | TempMail.so |
| `tempmailorgpremium` | Temp-Mail.org Premium |
| `mailtempcom` | Mail-Temp.com |
| `wemakemail` | WeMakeMail *(cần API key)* |
| `mailhv` | MailHV *(cần API key)* |
| `i2b` | Mail i2b.vn |
| `vietxf` | VietXF |

**Rent Mail Providers (13 provider):**

| Value | Label |
|---|---|
| `zeus-x` | ZeusX |
| `dongvanfb` | DongVanFB |
| `store1s` | Store1s |
| `mail30s` | Mail30s |
| `muamail` | MuaMail |
| `unlimitmail` | UnlimitMail |
| `sptmail` | SPTMail |
| `emailapiinfo` | EmailAPI.info |
| `otpcheap` | OTP.cheap |
| `shopgmail9999` | ShopGmail9999 |
| `rentgmail` | RentGmail.online |
| `otpcodesms` | OtpCodesSms.site |
| `wmemail` | Wmemail.com |

**Logic khi chọn provider**:
1. `form.mailProvider = value`
2. Load `domain` từ `form.tempMailDomains[value]` (per-provider map)
3. Load `token` từ `form.tempMailTokens[value]` (per-provider map)
4. Gọi `saveNow()` ngay — force save không chờ debounce

#### Bước 3: Các field config tùy provider

**Providers có Domain field** (`tempMailHasDomain=true`):
- `moakt`, `@i2b.vn`, `tempmail-plus`, `wemakemail`, `mailhv`, `vietxf`

```
Domain cho [TenProvider] (cách nhau bằng dấu phẩy):
[______________________________________]
```

Placeholder theo provider:
- `moakt` → `tmpbox.net, other.net`
- `@i2b.vn` → `i2b.vn, other.net`
- `tempmail-plus` → `mailto.plus, fexpost.com, fexbox.org`
- `wemakemail` / `mailhv` → `Để trống = API tự chọn | hoặc: domain1.com, domain2.com`

**Token / API key field** (luôn hiện):

```
Token / API key cho [TenProvider] (bắt buộc / tuỳ chọn):
[______________________________________]
```

Bắt buộc (`tempMailTokenRequired=true`) với: `wemakemail`, `mailhv`, `vietxf`

---

#### Config đặc biệt theo từng provider

##### `wemakemail` — WeMakeMail
Sau khi nhập API key, hiện thêm:

```
[ 🔍 Tải domain từ API ]   ← nhập API key trước
```

Sau khi nhấn fetch thành công:

```
[ Free (n) ] [ Trả phí (n) ] [ Tất cả (n) ]  [PLAN_BADGE]  [Dùng free/trả phí/tất cả]  [Bỏ chọn tất cả]
· click domain để chọn / bỏ chọn

┌─ Chip domain ──────────────────────────────────────────┐
│  [✓ domain1.com] [domain2.net] [domain3.org] ...       │
└────────────────────────────────────────────────────────┘
```

| Nút | Logic |
|---|---|
| **🔍 Tải domain từ API** | Disabled nếu `!currentTempMailToken`. Gọi `FetchWeMakeMailDomains(apiKey)` → parse JSON `{free:[],paid:[],all:[],plan:string}` |
| **Free / Trả phí / Tất cả** | Switch tab — hiện domain set tương ứng; disabled nếu set rỗng |
| **Dùng [tab]** | `wmFillGroup(wmTabDomains)` → ghi toàn bộ domain tab hiện tại vào ô domain |
| **Bỏ chọn tất cả** | `currentTempMailDomain = ''` |
| **Chip domain** | Toggle: nếu đã có trong ô → xóa, chưa có → thêm. Style: active=xanh nhạt + ✓ |

##### `mailhv` / `vietxf` — MailHV / VietXF

```
[ 🔍 Tải domain từ API ]

[n domain]  [Dùng tất cả]  [Bỏ chọn tất cả]  · click domain để chọn
┌─ Chip domain ─────────────────────────────────┐
│  [✓ hvmail.net] [hvd.pro] ...                │
└───────────────────────────────────────────────┘
```

| Nút | Logic |
|---|---|
| **🔍 Tải domain từ API** | Gọi `FetchMailHVDomains(apiKey)` hoặc `FetchVietXFDomains(apiKey)` → trả về `{all:[]}` hoặc `{domains:[]}` |
| **Dùng tất cả** | `vxFillAll()` → ghi tất cả domain vào ô |
| Chip domain | Toggle tương tự WeMakeMail |

##### `tempmail-lol` — TempMail LOL

```
API Key (tuỳ chọn):
[Bearer token — để trống nếu dùng free tier]
```
Field: `form.tempMailLolApiKey`

##### `zeus-x` — ZeusX

```
API Key:      [__________________]  (bind: form.zeusXApiKey)
Loại mail:    [dropdown ZEUS_X_ACCOUNT_CODES]  (bind: form.zeusXAccountCode)

[ Kiểm tra tồn kho ]    [Tồn kho: 145] hoặc [Hết hàng]
```

| Nút | Logic |
|---|---|
| **Kiểm tra tồn kho** | Disabled khi `zeusXLoading`. Gọi `checkZeusXStock()` từ `useMailProviderStock` |
| Badge tồn kho | `vr-stock-badge--ok` (xanh) nếu `Instock > 0`; `vr-stock-badge--empty` (đỏ) nếu hết |

##### `dongvanfb` — DongVanFB

```
API Key:      [__________________]  (bind: form.dvfbApiKey)
Loại mail:    [dropdown DONGVANFB_ACCOUNT_TYPES]  (bind: form.dvfbAccountType)

[ Kiểm tra tồn kho ]    [Tồn kho: 67] hoặc [Hết hàng]
```

Disabled nếu `!form.dvfbApiKey`

##### `store1s` — Store1s

```
API Key:      [__________________]  (bind: form.store1sApiKey)
Product ID:   [dropdown STORE1S_PRODUCTS]  (bind: form.store1sProductId)

[ Kiểm tra tồn kho ]    [Tồn kho: 12]
```

##### `mail30s` — Mail30s

```
API Key:      [__________________]  (bind: form.mail30sApiKey)
Sản phẩm:    [-- Tải danh sách trước --]  (bind: form.mail30sProductSlug)
              options: từ API sau khi fetch

[ Tải sản phẩm ]    [Tồn: 30]
```

Dropdown populate sau khi nhấn "Tải sản phẩm": `{name, price_display, stock}`

##### `muamail` — MuaMail

```
API Key:      [__________________]  (bind: form.muaMailApiKey)
Product ID:   [__________________]  (bind: form.muaMailProductId)
```

##### `unlimitmail` — UnlimitMail

```
API Key (Token): [__________________]  (bind: form.unlimitMailApiKey)
Product ID:      [__________________]  (bind: form.unlimitMailProductId)
```

##### `sptmail` — SPTMail

```
API Key:       [__________________]  (bind: form.sptMailApiKey)
Service Code:  [__________________]  (bind: form.sptMailServiceCode)
Placeholder: "otpServiceCode từ sptmail.com..."
```

##### `emailapiinfo` — EmailAPI.info

```
API Key:       [__________________]  (bind: form.emailAPIInfoApiKey)
Product Code:  [gmail]               (bind: form.emailAPIInfoProductCode)
```

##### `otpcheap` — OTP.cheap

```
API Key:    [__________________]  (bind: form.otpCheapApiKey)
Service ID: [8]                   (bind: form.otpCheapServiceId)
Placeholder: "8 (Facebook)"
```

##### `shopgmail9999` — ShopGmail9999

```
API Key:  [__________________]  (bind: form.shopGmail9999ApiKey)
Service:  [facebook]            (bind: form.shopGmail9999Service)
```

##### `rentgmail` — RentGmail.online

```
Token:     [__________________]  (bind: form.rentGmailApiKey)
Platform:  [facebook]            (bind: form.rentGmailPlatform)
```

##### `otpcodesms` — OtpCodesSms.site

```
API Key:    [__________________]  (bind: form.otpCodesSmsApiKey)
Service ID: [__________________]  (bind: form.otpCodesSmsServiceId)
```

##### `wmemail` — Wmemail.com

```
Token (API Key):  [__________________]  (bind: form.wmemailApiKey)
Commodity ID:     [__________________]  (bind: form.wmemailCommodity)
Placeholder: "commodity_id (vd: gói Hotmail OAuth2)"
```

---

### Tab PHONE (Preview — chưa hoàn thiện)

```
[ SMS OTP ]  [ Rent Phone ]    🚧 Preview
```

**SMS OTP providers** (chip buttons):
`SMS-Activate | 5SIM.net | SMSHub | SMSCodes | OnlineSim | SMS-Man | SMSBower | TextVerified`

**Rent Phone providers** (chip buttons):
`RentSim.vn | SimRental | ViOTP | OtpSim`

**Fields** (disabled — chưa wire backend):
```
API Key:  [disabled]
Country:  [🇻🇳 Vietnam | 🇮🇩 Indonesia | 🇵🇭 Philippines | 🇮🇳 India | 🇺🇸 United States]
Service:  [Facebook | Instagram | Gmail]
```

> 🚧 Backend chưa wire. UI sẵn sàng, chờ implement.

---

## 5. Thiết lập chạy (Interaction Setup)

**File**: `pages/InteractionSetupPage.vue`

Đây là trang lớn nhất (4300+ dòng). Chia 3 accordion section dạng card: **① Reg account**, **② Verify / Xác thực**, **③ User Agent**.

### Header / Toolbar

```
Thiết lập chạy    [Default (1) ▼]  [◑ Đang lưu...]  [× Đóng]
```

**Profile selector** (`ProfileManager`): `[Default (1) ▼]` — chọn profile; các nút New / Clone / Rename / Delete / Export JSON / Import JSON / Reset nằm trong dropdown.

**Auto-save**: `useAutoSave` debounce 1.5s → `IInteractionService.save()`.

---

### Section ① Reg account (accordion, order: 1)

Click header để collapse/expand. Badge `[BẬT]` / `[TẮT]` theo `form.createEnabled`.

```
① Reg account                                            [BẬT] [▲]
───────────────────────────────────────────────────────────────────
API REG  — click chọn nhiều version · chuột phải để bỏ chọn tất cả   [Hiển thị]

  [S22] [S23] [S24] [S25] [S26]  (standard — Xanh lá khi active)

  Fb_399  Fb_415  Fb_416 ... Fb_425  (versioned — Vàng khi active)

  [Chưa chọn version nào → dùng mặc định: s23]

Pool:  [Android 0] [iPhone 0] [Request 0]        ☐ Tracking ID    [▶ Test]
       ☐ UA Gốc   ☐ Virtual spec   ☑ Build UA
       (UA Gốc bật → hiện thêm: ☐ Thay nhà mạng)
       (Test kết quả: hiện <code> chuỗi UA)

Reg luồng: [20]    Delay reg (s): [0]    Delay step (ms): [0]
☐ Auto-restart sau: [60] phút

───────────────────────────────────────────────────────────────────
REG SETTINGS

Lead Domain Mail:  [@gmail.com,@yahoo.com      ]
Password:          [mẫu password...             ]
Name:              [US ▼]    (US / VN / Random)
Mode:              [Mail (lead domain) ▼]
  Options: Mail | Phone | TempMail (reuse cho verify) | Mail-Temp.com | Random

☐ Xoay Mail ↔ Phone   Mail: [30] phút   Phone: [15] phút
  (Xoay chỉ áp dụng khi Mode ≠ TempMail/MailTemp)

Phone × Mail:  ○ Random Normal  ● Random File  ○ Fm Phone Code

───────────────────────────────────────────────────────────────────
COOKIE INITIAL  — Nguồn datr đầu vào cho pool register

Nguồn:   ● Từ file   ○ Tạo mới
         ☐ Add new pool (ghi datr mới vào file pool)
         [Mở file datr]  1.234 datr · pool: 456 · +12 saved

Giới hạn:
  ☐ Lượt dùng    [100]       ☐ Checkpoint    [5]
  ☐ Xóa khi đạt giới hạn
  ☐ Xóa sau      [120] phút
```

**Context menu chuột phải trên API REG chips:**
- Chọn › Tất cả / Fb_399–Fb_415 / Fb_416–Fb_425 / ...
- Dán từ JSON
- Bỏ chọn tất cả

**Nút [Hiển thị]**: mở panel filter ẩn/hiện từng nhóm platform button theo danh mục (Android / Versioned / ...).

| Field | v-model | Clamp |
|---|---|---|
| Reg luồng | `form.regThreads` | 1–600 |
| Delay reg | `form.delayReg` | ≥0 |
| Delay step | `form.delayStep` | ≥0 |
| Auto-restart | `form.autoRestartEnabled` + `form.autoRestartMinutes` | 1–999 |
| Lead Domain Mail | `form.leadDomainMail` | — |
| Password | `form.passwordReg` | — |
| Name locale | `form.nameRegLocale` | US/VN/random |
| Mode | `form.regMode` | 5 options |
| Rotate toggle | `form.regModeRotate` | disabled cho TempMail/MailTemp |
| Mail rotate phút | `form.regModeRotateMailMinutes` | ≥1 |
| Phone rotate phút | `form.regModeRotatePhoneMinutes` | ≥1 |
| Phone×Mail mode | `form.phoneMailMode` | radio |
| Cookie Initial method | `form.cookieInitialMethod` | file / new |
| Add new pool | `form.saveNewDatr` | checkbox |
| Lượt dùng | `form.limitCookieInitial` + `form.limitCookieInitialCount` | ≥1 |
| Checkpoint | `form.limitCheckpoint` + `form.limitCheckpointCount` | ≥1 |
| Xóa khi đạt | `form.deleteDatrCheckpoint` | checkbox |
| Xóa sau phút | `form.limitDatrAge` + `form.limitDatrAgeMinutes` | ≥1 |

---

### Section ② Verify / Xác thực (accordion, order: 2)

Click header collapse/expand. Badge `[BẬT]`/`[TẮT]` theo `form.verifyEnabled`. Toàn bộ fieldset disabled khi `verifyEnabled=false`.

```
② Verify / Xác thực                                      [BẬT] [▲]
───────────────────────────────────────────────────────────────────
API VERIFY  — click chọn nhiều version · chuột phải ...         [Hiển thị]

  [api android] [api token] [api mfb] [api web andr]  · [Fb_399] [Fb_415] ...

  [Đang chọn 3 version — mỗi account verify dùng 1 version (xoay vòng). Focus: api android]

Pool:  [Android 0] [iPhone 0] [Request 0]        ☐ Tracking ID    [▶ Test / 👁 Xem]
       ☐ UA Gốc   ☐ Virtual spec   ☑ Build UA   ☐ Thay nhà mạng (nếu UA Gốc)

Verify luồng: [0]    (= Reg luồng nếu Reg+Verify không Split mode)

───────────────────────────────────────────────────────────────────
TIMING & DELAY

Chờ OTP tối đa (s):     [30]   Check mail mỗi (s):      [5]
Resend OTP (lần):        [ 1]   Retry add mail (lần):    [0]
Trước submit OTP (s):   [ 1]   Sau submit OTP (s):       [5]
Reg → verify (s):       [ 1]   Trước check live (s):     [5]
Giữ kết quả UI (s):    [ 1]   Dùng lại mail (lần):      [3]  (hiện khi Re-use email)

☑ Tự gửi lại OTP khi timeout  ☐ Re-use email  ☐ Fm User TmpMail  ☐ Get datr on Live
```

**Context menu Verify** giống Reg: Chọn tất cả / chọn batch / Dán từ JSON / Bỏ chọn tất cả.

**Nút Test / Xem**: khi `useOriginalUA=false` → hiện `▶ Test` gọi backend sinh UA test; khi `useOriginalUA=true` → hiện `👁 Xem` để preview UA gốc.

| Field | v-model | Ghi chú |
|---|---|---|
| Verify luồng | `form.splitVerifyThreads` | Hidden khi Reg+Ver+noSplit |
| Chờ OTP tối đa | `form.timeDelaySendCode` | giây |
| Check mail mỗi | `form.waitMail` | giây |
| Resend OTP | `form.trySendCode` | 1–2 |
| Retry add mail | `form.addMailRetry` | 0–5 |
| Trước submit OTP | `form.delayConfirmEmail` | giây |
| Sau submit OTP | `form.timeDelayCheck` | giây |
| Reg → verify | `form.delayVeriReg` | giây |
| Trước check live | `form.delayCheckLive` | giây |
| Giữ kết quả UI | `form.delayDisplayResult` | 0–60s |
| Dùng lại mail | `form.useMailTimes` | ≥1; hiện khi reUseEmail |
| Tự gửi lại OTP | `form.sendAgainCode` | checkbox |
| Re-use email | `form.reUseEmail` | checkbox |
| Fm User TmpMail | `form.fmUserTmpMail` | checkbox |
| Get datr on Live | `form.getNewDatrOnLive` | checkbox |

---

### Section ③ User Agent (accordion, order: 3)

```
③ User Agent                                     [Android] [▲]
───────────────────────────────────────────────────────────────────
[Android  0]  [iPhone  0]  [Request  0]

Nguồn:  Config/UserAgent/ua_android.txt   [Mở file]
```

Badge hiện tên pool đang chọn. Nút **Mở file** → `ShellOpenUAFile()` mở file trong Notepad.

Pool mapping: `android` → `ua_android.txt`, `iphone` → `ua_iphone.txt`, `request` → `ua_request.txt`.

---

## 6. Cài đặt chung (General Settings)

**File**: `pages/GeneralSettingsPage.vue`

### Header

```
Cài đặt chung                      [Default (1) ▼]  [◑ Đang lưu...]  [× Đóng]
```

### Section: Đăng nhập & Môi trường

```
Dạng đăng nhập:  [dropdown các phương thức login]
Nghỉ:            [1000] ms
Delay luồng:     [0] ms

ℹ Số luồng đã chuyển sang "Thiết lập chạy" → Reg account / Verify để 2 bên tự cài luồng.
ℹ User Agent pool đã chuyển sang "Thiết lập chạy" — section "User Agent"
```

### Section: Nguồn tài khoản

```
○ Từ thư mục (đọc file .txt)   ○ Từ 1 file (chọn acc tick)   ○ Mua từ API (CloneHV)
Thư mục nguồn: [____________________________] [📁]
```

| Radio | Value | Hành động |
|---|---|---|
| Từ thư mục | `folder` | Quét toàn bộ `.txt` trong thư mục |
| Từ 1 file | `file` | Pick 1 file duy nhất, backend nhớ path |
| Mua từ API (CloneHV) | `clonehv` | Hiện form username/password/productId CloneHV |

### Section: Locale & Sim Network

```
LOCALE FAKE                      SIM NETWORK
○ Random  ● Match by IP          ○ Random  ● Match by IP
☑ Deep Fake in API               Loại mạng: [LTE ▼]
   Fake locale sâu hơn trong API call
```

### Section: Sau khi chạy

```
☐ Lưu cột lần chạy              ☐ Sao lưu database
   Ghi IP chạy, hoạt động vào      Tạo file backup sau mỗi lần chạy
   database

☐ Đóng app khi xong
```

### Profile Management (top-right)

```
[Default (1) ▼]  [New] [Clone] [Rename] [Delete]
[Export JSON] [Import JSON]
```

---

## 7. Proxy Settings

**File**: `pages/ProxySettingsPage.vue`

### Header

```
Proxy Settings              [● Tự động lưu]  [× Đóng]
```

### Layout: 2 cột (main-col | side-col)

---

#### Cột trái: §1 Nhà cung cấp IP + §2 Cấu hình proxy

##### §1 Nhà cung cấp IP

```
① Nhà cung cấp IP                    [PROVIDER_NAME]
──────────────────────────────────────────────────────
Nhà cung cấp: [dropdown IP_PROVIDERS ▼]  [?]

ℹ Dùng chung cho cả Đăng ký và Verify.
  Credentials và danh sách proxy cấu hình ở §2 bên dưới.
  [Thiết lập chạy §4] ← link
```

**IP Providers** (dropdown):

| Value | Label |
|---|---|
| `none` | Không dùng proxy |
| `hma` | HMA VPN |
| `proxy` | Proxy list (rotate) |
| `proxy_fixed` | Proxy cố định (từ cột Proxy của account) |
| `fpt` | FPT API Keys |
| `xproxy` | XProxy.vn |
| `tinsoft` | TinSoft |
| `shoplike` | Shoplike |
| `netproxy` | NetProxy |
| `minproxy` | MinProxy |
| `netproxy_gb` | NetProxy (dung lượng GB) |
| `proxy_popular` | Proxy Popular |
| `proxy_farm` | Proxy Farm |

##### §2 Cấu hình proxy

Layout: **REG (trên)** → divider dashed → **VERIFY (dưới)**

**Khi provider = `none`**:
```
✍️ REG — Không dùng proxy
   [icon] Không có proxy list — Provider hiện tại không dùng danh sách proxy.

──────────────────── (divider)

✅ Verify — Không dùng proxy
   [icon] Không đổi IP — Request chạy trực tiếp từ IP máy chủ.
```

**Khi provider = `hma`**:
```
✅ Verify — HMA VPN
   [🔒] HMA VPN — Đổi IP qua VPN hệ thống. Cài HMA riêng và bật trước khi chạy.
```

**Khi provider = `proxy` / `proxy_fixed`**:
```
Loại proxy:  ● HTTP  ○ SOCKS5

[textarea — Mỗi proxy một dòng]
host:port:user:pass
host:port:user_area-XX_session-ID_life-N:pass
...
Tự động nhận diện session proxy (có _session-, -zone-).  [n proxy]
```
*(Cả REG và VERIFY đều có block riêng, loại proxy chọn riêng)*

**Khi provider = `fpt`**:
```
FPT API Keys (n key):
[textarea — Mỗi key một dòng]
```

**Khi provider = `xproxy`**:
```
Link server:      [http://xproxy.vn/...]
Loại proxy:       ● HTTP  ○ SOCKS5
Luồng / IP: [1]  [?]
Chế độ: ● Dùng chung proxy  ○ Mỗi luồng 1 proxy
Danh sách proxy dự phòng (n proxy):
[textarea]
```

**Khi provider = `tinsoft` / `shoplike` / `netproxy` / `minproxy`**:
```
Keys (n key):
[textarea — Mỗi key một dòng]
Luồng / IP: [1]
```

**Khi provider = `netproxy_gb`**:
```
Key dung lượng: [____________________]
```

**Khi provider = `proxy_popular` / `proxy_farm`**:
```
Access Token: [____________________]
Keys (n key): [textarea]
Luồng / IP:   [1]
```

**Khi provider = shared** (`tinsoft`, `shoplike`, `netproxy`, `minproxy`, `netproxy_gb`, `proxy_popular`, `proxy_farm`):
```
ℹ Provider [NAME] — Reg dùng chung credentials với Verify (cấu hình ở §2 bên dưới).
```

---

#### Cột phải: §3 Kiểm tra kết nối + §4 Retry & Delay + §5 Proxy Mail

##### §3 Kiểm tra kết nối

```
API kiểm tra IP: [dropdown API_CHECK_IP_PROVIDERS ▼]
[ 🔍 Kiểm tra IP hiện tại ]    1.2.3.4
```

| Nút | Logic |
|---|---|
| **🔍 Kiểm tra IP hiện tại** | Disabled khi `checkingIp`. Gọi `CheckCurrentIPViaProxy()` (Wails) hoặc fetch từ ipify/ipinfo/nordvpn tùy provider |
| Badge IP | Hiện IP khi có kết quả |

##### §4 Retry & Delay IP

```
Số lần retry khi lỗi proxy:    [ 3 ] lần
   0 = không retry, dùng proxy tiếp theo ngay

Delay trước khi đổi proxy:     [ 0 ] ms
   0 = đổi ngay, 1000 = chờ 1 giây
```

##### §5 Proxy Mail

```
[☐ Proxy TempMail] [📧 TempMail (n)]  |  [☐ Proxy RentMail] [✉️ RentMail (n)]
──────────────────────────────────────────────────────
[textarea — active tab]
Mỗi dòng 1 proxy (host:port:user:pass hoặc http://user:pass@host:port)...

ℹ Tự động lưu khi nhập xong. Bật toggle "Proxy TempMail/Gmail" ở Thiết lập chạy để dùng.
```

| Nút/Control | Logic |
|---|---|
| **☐ Proxy TempMail** | `useProxyTempmail` — lưu vào `interaction.json` debounce 500ms |
| **☐ Proxy RentMail** | `useProxyRentmail` — lưu vào `interaction.json` |
| **📧 TempMail (n)** | Chuyển tab textarea sang TempMail list |
| **✉️ RentMail (n)** | Chuyển tab textarea sang RentMail list |
| Textarea | Auto-save 1s debounce sau khi thay đổi; gọi `SaveProxyList('tempmail'/'gmail', content)` |

---

## 8. Hiển thị (View Settings)

**File**: `pages/ViewSettingsPage.vue`

### Layout

```
Hiển thị
─────────────────────────────────────────────────────────

┌────────────────────────────────────┐  ┌───────────────────┐
│ MẬT ĐỘ BẢNG                       │  │ BẢO MẬT DỮ LIỆU   │
│                                    │  │                   │
│  ○ ▤ Compact    Row 28px, font 11px│  │ ○ Ẩn dữ liệu      │
│  ● ▦ Default    Row 36px, font 13px│  │   nhạy cảm        │
│  ○ ▧ Comfortable Row 44px, font 14px│  │   Password, Cookie│
│                                    │  │   Token hiện ••••• │
└────────────────────────────────────┘  └───────────────────┘

CỘT HIỂN THỊ                          7/27 cột     [Đặt lại mặc định]
─────────────────────────────────────────────────────────

TÀI KHOẢN                          ☐ (toggle all)
☑ UID  ☐ Dữ liệu gốc  ☑ Mật khẩu  ☐ 2FA  ☑ Email  ☐ Pass Email  ☐ Mail khôi phục  ☐ Cookie  ☐ Token

TRẠNG THÁI                         ☐ (toggle all)
☑ Trạng thái  ☐ Checkpoint  ☐ Quảng cáo  ☐ BM  ☐ TKQC  ☐ Chat Support

CHẠY & KHÁC                        ☐ (toggle all)
☐ Avatar  ☐ Cover  ☐ Phone  ☑ Proxy  ☐ UA  ☐ Ghi chú  ☐ Ghi chú chạy
☐ Ngày nhập  ☐ Thư mục  ☐ Chạy lần cuối  ☑ IP chạy  ☑ Hoạt động
```

### Controls

| Control | Binding | Logic |
|---|---|---|
| Radio Compact | `prefs.density = 'compact'` | Row 28px, font 11px |
| Radio Default | `prefs.density = 'default'` | Row 36px, font 13px |
| Radio Comfortable | `prefs.density = 'comfortable'` | Row 44px, font 14px |
| Toggle bảo mật | `prefs.maskSensitive` | Ẩn password/cookie/token → hiện `••••••` |
| Checkbox column | `prefs.toggleColumn(key)` | Toggle visibility, persist localStorage |
| Checkbox group header | `toggleGroup(keys, checked)` | Bật/tắt cả nhóm |
| **Đặt lại mặc định** | `prefs.resetColumns()` | Reset về columnVisibility mặc định |

**Persist**: tất cả lưu vào `localStorage` qua `usePreferencesStore` — survive reload.

---

## 9. Tương tác

**File**: `pages/TuongTacPage.vue`

### Header

```
⚡ Tương tác   Chạy tự động sau verify Live                    [Tự động lưu]
```

Auto-save: watch form deep → save ngay khi thay đổi (không debounce).

### Nhóm: Sau khi verify thành công

```
┌──────────────────────────────────────────────────────────┐
│ SAU KHI VERIFY THÀNH CÔNG                               │
├──────────────────────────────────────────────────────────┤
│ Upload Avatar                              [Toggle ○/●]  │
│ Pick ngẫu nhiên ảnh JPG/PNG từ thư mục đã chọn          │
│   [Config/Avatar    ] [📁]                              │
│   (hiện khi toggle bật)                                  │
├──────────────────────────────────────────────────────────┤
│ Bật 2FA (TOTP)                             [Toggle ○/●]  │
│ Kích hoạt xác thực 2 bước — lưu secret key vào output   │
├──────────────────────────────────────────────────────────┤
│ Cập nhật thông tin hồ sơ                   [Toggle ○/●]  │
│ Điền city, trường học, nơi làm việc từ Config/AddInfo/   │
│   (khi bật hiện thêm grid checkbox):                     │
│   ☐ Thành phố   ☐ Quê quán    ☐ Trường học              │
│   ☐ Đại học     ☐ Nơi làm việc  ☐ Độc thân              │
└──────────────────────────────────────────────────────────┘

[Các tính năng khác (like, comment, add friend…) sẽ được thêm vào đây]
```

### Controls chi tiết

| Control | Binding | Hành động |
|---|---|---|
| **Toggle Upload Avatar** | `form.uploadAvatar` | Bật → hiện ô path + nút browse |
| **📁 Browse avatar folder** | `form.avatarFolderPath` | `getFileDialogService().openFolder()` |
| **Toggle Bật 2FA** | `form.enable2fa` | Kích hoạt TOTP khi verify xong |
| **Toggle Cập nhật hồ sơ** | `form.addInfo` | Bật → hiện 6 checkbox thành phần |
| ☐ Thành phố | `form.addInfoCity` | |
| ☐ Quê quán | `form.addInfoHometown` | |
| ☐ Trường học | `form.addInfoSchool` | |
| ☐ Đại học | `form.addInfoCollege` | |
| ☐ Nơi làm việc | `form.addInfoWork` | |
| ☐ Độc thân | `form.addInfoRelationship` | |

---

## 10. Đẩy tài khoản (Upload Site)

**File**: `pages/UploadSitePage.vue`

### Thiết kế

2-panel: trái = cấu hình API, phải = danh sách sản phẩm.

```
⬆ Đẩy tài khoản                              [• Tự động lưu]
────────────────────────────────────────────────────────────

┌────────────────────────────┐  ┌──────────────────────────┐
│ CẤU HÌNH API               │  │ SẢN PHẨM                  │
│                            │  │                           │
│ API Key:                   │  │ [Tải danh sách]  [↻ Làm mới]│
│ [____________________]     │  │                           │
│                            │  │ ○ Product A — 5.000đ  45 │
│ ☐ Hiện mật khẩu            │  │ ● Product B — 8.000đ  12 │
│ Admin user: [___________]  │  │ ○ Product C — 3.000đ   0 │
│ Admin pass: [●●●●●●●●] 👁  │  │                           │
│ (giải thích)               │  └──────────────────────────┘
│                            │
│ Stock code:                │
│ [____________________]     │
│                            │
│ Số luồng đẩy: [10]         │
│ Delay giữa batch (ms): [500]│
└────────────────────────────┘
```

### Nút bấm

| Nút | Logic |
|---|---|
| **👁 Hiện/ẩn admin password** | `showAdminPassword` toggle — `type="password"` / `type="text"` |
| **Tải danh sách** | `loadProducts()` — gọi `GetBancloneProducts(apiKey, adminUser, adminPass)`. Disabled nếu `!form.apiKey` |
| **↻ Làm mới** | `loadProducts(false)` — force reload, không dùng cache |
| Radio sản phẩm | Chọn product → `form.code = product.code` (nếu có stock code thật) |

**Cache logic**:
- Sản phẩm được cache trong `localStorage` key `banclone_products_list`
- Stock code cache key `banclone_stock_map`
- Khi mount: load cache trước, nếu rỗng → auto-load từ API (silent)

**Auto-save**: watch form deep, debounce 600ms → `IUploadSiteService.save()`

---

## 11. Thống kê (RegStats)

**File**: `pages/RegStatsPage.vue`

### Header

```
Thống kê   [● REG đang chạy]  [● VERIFY đang chạy]  [■ Đã dừng]

                    [REG / VERIFY] [Mail Domain]    Cập nhật 10:23:45 · tự làm mới 10s    [↻ Làm mới]
```

### Nút bấm

| Nút | Logic |
|---|---|
| **REG / VERIFY** (tab) | `activeTab = 'reg-ver'` |
| **Mail Domain** (tab) | `activeTab = 'mail'` |
| **↻ Làm mới** | `fetchStats()` — disabled khi `loading`. Gọi `GetRegStats`, `GetVerifyStats`, `GetMailDomainStats` |

### Tab REG / VERIFY

```
┌──────────────────────────────────────┐  ┌───────────────────────────────────┐
│ Đăng ký (REG)                [Export]│  │ Xác thực (VERIFY)         [Export]│
├─────┬────────────┬────────┬──────────┤  ├─────┬────────────┬────────┬───────┤
│  #  │ Platform   │ OK     │ Fail     │  │  #  │ Platform   │ OK     │ Fail  │
├─────┼────────────┼────────┼──────────┤  ├─────┼────────────┼────────┼───────┤
│  1  │ S23        │ 380    │ 70       │  │  1  │ api android│ 210    │ 45    │
│  2  │ S399       │ 165    │ 55       │  │  2  │ api token  │  98    │ 22    │
└─────┴────────────┴────────┴──────────┘  └─────┴────────────┴────────┴───────┘
```

Component: `RegVerStatsTable` — có nút **Export** export CSV.

### Tab Mail Domain

```
┌───────┬──────────────┬──────┬──────┬──────┬───────┐
│  #    │ Domain       │ Veri │ Live │ Die  │ Rate  │
├───────┼──────────────┼──────┼──────┼──────┼───────┤
│  1    │ @gmail.com   │ 623  │ 541  │  82  │ 86.8% │
│  2    │ @yahoo.com   │ 211  │ 152  │  59  │ 72.0% │
└───────┴──────────────┴──────┴──────┴──────┴───────┘
```

Component: `MailDomainStatsTable`

**Auto-refresh**: `setInterval(fetchStats, 10_000)` — tự làm mới mỗi 10 giây.

**Offline state**: nếu `window.go.main.App` không có → hiện `"Không kết nối được backend — mở app qua Wails để xem thống kê."`

---

## 12. Flow Settings

**File**: `pages/FlowSettingsPage.vue`

### Layout

2 panel ngang: danh sách flow bên trái (280px) + chi tiết flow bên phải.

```
Flow Settings
──────────────────────────────────────────────────────────────────
┌──────────────────────┐  ┌──────────────────────────────────────┐
│ Flows (2)  [+ Thêm]  │  │ Flow Name        [engineType badge]  │
│──────────────────────│  │ Mô tả flow...                        │
│ > Flow A             │  │                                      │
│   Mô tả ngắn...      │  │ Steps (5)                            │
│──────────────────────│  │ ──────────────────────────────────── │
│   Flow B             │  │ # │ Action    │ Input  │ Timeout│Retry│On│
│   Mô tả ngắn...      │  │ 1 │ fill_name │ Random │  5s   │ 1  │ ✓│
│──────────────────────│  │ 2 │ submit    │        │ 10s   │ 2  │ ✓│
│ (Chưa có flow nào)   │  │ ...                                  │
└──────────────────────┘  └──────────────────────────────────────┘
```

### Nút bấm

| Nút | Trạng thái | Hành động |
|---|---|---|
| **+ Thêm** | Disabled (`title="Phase 2"`) | Chưa triển khai — Phase 2 |
| Flow item trong list | Click | `selectFlow(flow)` → hiện detail bên phải |

### Flow detail panel

| Field | Hiển thị |
|---|---|
| `flow.name` | h3 tên flow |
| `flow.engineType` | Badge info chip (vd: `go-script`) |
| `flow.description` | Mô tả |
| `flow.steps` | Table: #, Action, Input (mono, truncate), Timeout (s), Retry, On (✓/✕) |

**Trạng thái On/Off**: ✓ màu `success-text`, ✕ màu `danger-text`.

**Data source**: `getFlowService().list()` — mock hoặc backend trả về array `Flow[]`.

---

## 13. Import cấu hình WeBM (LegacyImportWizard)

**File**: `pages/LegacyImportWizard.vue`  
**Route**: `/legacy-import`

Wizard 4 bước để chuyển cấu hình từ tool WeBM cũ (`general.json` + `interaction.json`) sang cấu hình mới.

### Step indicator

```
[1] Nhập JSON → [2] Xem report → [3] Xác nhận → [4] Hoàn tất
```
- Step active: màu accent `#4fc3f7`
- Step done: tick ✓, màu xanh lá `#66bb6a`
- Step chưa đến: mờ 40%

---

### Bước 1 — Nhập JSON

```
Import cấu hình từ WeBM
Chuyển đổi general.json + interaction.json sang cấu hình mới một cách an toàn.

Mở thư mục cài đặt tool cũ, copy nội dung general.json và interaction.json...

┌──────────────────────────┐  ┌──────────────────────────┐
│ general.json             │  │ interaction.json          │
│ { "General": { ... } }   │  │ { "VerifyEnabled": ... }  │
│ (textarea 280px high)    │  │ (textarea 280px high)     │
└──────────────────────────┘  └──────────────────────────┘

[Lỗi: ... (nếu có)]

                                     [Hủy]  [Phân tích →]
```

| Nút | Enabled | Hành động |
|---|---|---|
| **Hủy** | Luôn | `router.back()` |
| **Phân tích →** | Luôn (validate trước khi gọi) | `getLegacyImportService().parse(generalJSON, interactionJSON)` → nếu thành công → bước 2 |

**Validation**: nếu cả 2 textarea trống → hiện lỗi "Vui lòng dán ít nhất một trong hai file JSON."

---

### Bước 2 — Xem report

```
[3 OK]  [1 Xác nhận]  [0 Nhạy cảm]  [2 Không hỗ trợ]  [6 tổng]

┌ Lỗi parse (nếu có) ──────────────────────────────────────────┐
│ Parse error message...                                        │
└───────────────────────────────────────────────────────────────┘

┌ Fields được map tự động (3) ─────────────────────────────────┐
│ Tên cũ           │ Path mới              │ Giá trị    │ OK   │
│ ThreadRequest    │ general.threadRequest │ 5          │ OK   │
└───────────────────────────────────────────────────────────────┘

┌ Cần xác nhận (1) ────────────────────────────────────────────┐
│ Tên cũ   │ Path mới │ Giá trị │ Ghi chú                      │
│ FolderAcc│ ...path  │ C:\...  │ Kiểm tra lại đường dẫn       │
└───────────────────────────────────────────────────────────────┘

┌ Fields nhạy cảm — sẽ được import (0) ────────────────────────┐
│ (giá trị thực không hiển thị — chỉ tên field)                │
└───────────────────────────────────────────────────────────────┘

┌ Không hỗ trợ — sẽ bỏ qua (2) ───────────────────────────────┐
│ Tên cũ      │ Giá trị │ Ghi chú                              │
│ OldFeature  │ true    │ Tính năng đã bỏ                      │
└───────────────────────────────────────────────────────────────┘

                                [← Quay lại]  [Tiếp theo →]
```

**Status badge màu sắc**:
- OK → xanh lá `#66bb6a`
- Xác nhận → cam `#ffa726`
- Nhạy cảm → tím `#ab47bc`
- Không hỗ trợ → đỏ `#ef5350`

---

### Bước 3 — Xác nhận

```
┌─────────────────────────────────────────────┐
│ Xác nhận import                             │
│                                             │
│ Hành động này sẽ ghi đè cấu hình hiện tại  │
│ bằng dữ liệu từ file cũ. Cấu hình cũ sẽ   │
│ bị mất. Nếu bạn muốn giữ bản backup, hãy  │
│ export trước.                               │
│                                             │
│ • 3 fields được import tự động             │
│ • 1 fields cần kiểm tra lại sau            │
│ • 0 fields nhạy cảm được import            │
│ • 2 fields không hỗ trợ sẽ bị bỏ qua      │
│                                             │
│ ☐ Tôi đã đọc report và xác nhận muốn import│
└─────────────────────────────────────────────┘

                      [← Xem lại]  [Xác nhận import]
```

| Nút | Enabled | Hành động |
|---|---|---|
| **← Xem lại** | Luôn | `step = 2` |
| **Xác nhận import** | Chỉ khi `confirmed=true` và `!loading` | `getLegacyImportService().apply(...)` → `step = 4` |

Nút **Xác nhận import** màu đỏ (`#d32f2f`).

---

### Bước 4 — Hoàn tất

**Thành công:**
```
┌──────────────────────────────────┐
│            ✓                     │
│    Import thành công             │
│                                  │
│  Cấu hình đã được cập nhật.     │
│  Các fields cần xác nhận hãy    │
│  kiểm tra lại trong Cài đặt chung│
│                                  │
│       [Đến Cài đặt chung]        │
└──────────────────────────────────┘
```

**Thất bại:**
```
┌──────────────────────────────────┐
│            ✗                     │
│    Import thất bại               │
│                                  │
│  <error message>                 │
│                                  │
│           [Thử lại]              │
└──────────────────────────────────┘
```

| Nút | Hành động |
|---|---|
| **Đến Cài đặt chung** | `router.push({ name: ROUTE_NAMES.GENERAL_SETTINGS })` |
| **Thử lại** | `step = 1` — reset về bước nhập JSON |

---

## Ghi chú chung về Auto-save

| Trang | Debounce | Service |
|---|---|---|
| Nguồn xác thực | 500ms | `IInteractionService.save()` |
| Thiết lập chạy | 1.5s (useAutoSave) | `IInteractionService.save()` |
| Flow Settings | N/A — read-only | `IFlowService.list()` chỉ đọc |
| Legacy Import Wizard | N/A — wizard | `ILegacyImportService.parse/apply()` |
| Cài đặt chung | 1.5s (useAutoSave) | `ISettingsService.save()` |
| Proxy Settings | 1s (custom timer) | `ISettingsService.save()` + `SaveProxyList()` |
| Tương tác | 0ms (watch immediate) | `IInteractionService.save()` |
| Đẩy tài khoản | 600ms (custom timer) | `IUploadSiteService.save()` |
| View Settings | Ngay lập tức | `usePreferencesStore` → localStorage |

## Ghi chú chung về Save Status indicator

Tất cả các trang đều có indicator góc phải header:

| State | Text | Màu |
|---|---|---|
| `idle` | `• Tự động lưu` | muted/xám |
| `saving` | `◑ Đang lưu...` | brand (xanh nhạt) |
| `saved` | `✔ Đã lưu` | xanh lá (#4caf50) |
| `error` | `⚠ Lỗi lưu` | đỏ (#f44336) |

---

*Tài liệu tổng hợp từ source code trực tiếp — cập nhật: 2026-05-19 (bổ sung §5 đầy đủ, §12 Flow Settings, §13 LegacyImportWizard)*
