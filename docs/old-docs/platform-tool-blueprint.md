# Platform Automation Tool — Blueprint

> Tài liệu phân tích toàn diện dự án HVR (Hạ Vũ Clone) và thiết kế lại cho nền tảng mới.
> Dùng làm blueprint cho dự án C# WPF .NET 10.
> **Không gắn với Facebook — có thể áp dụng cho bất kỳ nền tảng nào.**

---

## Mục lục

1. [Tổng quan kiến trúc hiện tại](#1-tổng-quan-kiến-trúc-hiện-tại)
2. [Shell & Layout](#2-shell--layout)
3. [Sidebar — Toàn bộ menu (không bỏ sót)](#3-sidebar--toàn-bộ-menu-không-bỏ-sót)
4. [Chi tiết từng trang](#4-chi-tiết-từng-trang)
5. [Cài đặt chung (General Settings)](#5-cài-đặt-chung-general-settings)
6. [Hiển thị (View Settings)](#6-hiển-thị-view-settings)
7. [Data Model — Account](#7-data-model--account)
8. [Bridge Layer & Services](#8-bridge-layer--services)
9. [State Management](#9-state-management)
10. [Thiết kế lại cho C# WPF .NET 10](#10-thiết-kế-lại-cho-c-wpf-net-10)
11. [Architecture Map: Vue → WPF](#11-architecture-map-vue--wpf)
12. [Cấu trúc thư mục WPF đề xuất](#12-cấu-trúc-thư-mục-wpf-đề-xuất)
13. [Checklist triển khai WPF](#13-checklist-triển-khai-wpf)

---

## 1. Tổng quan kiến trúc hiện tại

### Stack kỹ thuật gốc

| Layer | Công nghệ |
|---|---|
| Desktop runtime | Wails v2 (Go + embedded browser) |
| Frontend | Vue 3 + TypeScript + Pinia + Vue Router |
| Backend | Go 1.21+ |
| Build | Vite (frontend) + `wails build` |
| Packaging | Single `.exe` (~26MB) |

### Nguyên tắc kiến trúc

- **Module-first**: code tổ chức theo feature, không theo layer kỹ thuật
- **Bridge layer**: UI không import trực tiếp Go bindings — đi qua contract interface
- **Mock-first**: mọi service đều có mock để chạy UI mà không cần backend
- **Desktop shell chuẩn**: title bar + sidebar + header + content area + status bar

---

## 2. Shell & Layout

### Cấu trúc layout tổng thể

```
┌──────────────────────────────────────────────────────────┐
│  TitleBar (frameless — min/max/close + drag handle)       │
├────────────┬─────────────────────────────────────────────┤
│            │  Header (toggle sidebar | theme | profile)  │
│  Sidebar   ├─────────────────────────────────────────────┤
│            │                                             │
│ [Nav items]│         Main Content Area                   │
│            │       (Router View / Page)                  │
│            │                                             │
│ [Divider]  │                                             │
│            │                                             │
│ [Settings] ├─────────────────────────────────────────────┤
│            │  StatusBar (CPU | RAM | connection status)  │
└────────────┴─────────────────────────────────────────────┘
```

### Các thành phần shell

| Component | Mô tả |
|---|---|
| `AppTitleBar` | Frameless window — drag handle + min/max/close button; không dùng Windows native title bar |
| `AppHeader` | Toggle collapse sidebar, icon theme dark/light, avatar + tên user, notification bell |
| `AppSidebar` | Collapsible left nav — expanded (~200px) / collapsed (~50px icon-only); nhãn ẩn khi collapse |
| `AppStatusBar` | Dòng dưới cùng — CPU%, RAM MB, trạng thái kết nối (Connected / Mock / Disconnected) |

### Sidebar behavior

- **Collapse**: click vào logo/toggle → ẩn text label, chỉ còn icon
- **Active state**: item hiện tại highlight màu brand + font bold
- **Badge**: tab Accounts có badge số lượng account hiện có
- **Divider**: đường mỏng phân tách nhóm chức năng vs nhóm cài đặt

---

## 3. Sidebar — Toàn bộ menu (không bỏ sót)

```
┌─────────────────────┐
│  🔷 Hạ Vũ  Clone   │  ← Brand logo + tên app
├─────────────────────┤
│                     │
│  👤 Accounts    [n] │  ← Badge: tổng số account
│  🌐 Proxy Settings  │
│  🔧 Thiết lập chạy  │  ← Run configuration
│  ✉  Nguồn xác thực │  ← Auth source (mail/OTP source)
│  ⚡ Tương tác       │  ← Interaction / activity
│  ⬆  Đẩy tài khoản  │  ← Upload accounts to platform
│  📊 Thống kê        │  ← Statistics & analytics
│                     │
│  ─────────────────  │  ← Divider
│                     │
│  ⚙  Cài đặt chung  │  ← General settings
│  👁  Hiển thị       │  ← View / display settings
│                     │
└─────────────────────┘
```

### Mapping route đầy đủ

| Menu label | Route path | Icon |
|---|---|---|
| Accounts | `/accounts` | Users |
| Proxy Settings | `/proxy-settings` | Globe |
| Thiết lập chạy | `/interaction-setup` | Wrench |
| Nguồn xác thực | `/auth-source` | Mail |
| Tương tác | `/tuong-tac` | Zap |
| Đẩy tài khoản | `/upload-site` | ArrowUpToLine |
| Thống kê | `/reg-stats` | BarChart3 |
| *(divider)* | — | — |
| Cài đặt chung | `/general-settings` | Settings |
| Hiển thị | `/view-settings` | Eye |

> **Không có route ẩn**: `LegacyImportWizard` (`/legacy-import`) không hiện trong sidebar — chỉ truy cập qua wizard flow từ trang khác.

---

## 4. Chi tiết từng trang

---

### 4.1 Accounts (Trang chủ)

**Route**: `/accounts`  
**Mục đích**: Quản lý toàn bộ tài khoản — workspace chính của user.

#### Layout trang

```
┌─────────────────────────────────────────────────────────────┐
│  Toolbar                                                    │
│  [Import] [Delete] [Run] [Verify] [Stop] [Filter…] [Search] │
├─────────────────────────────────────────────────────────────┤
│  Stats bar:  Live: 45  |  Die: 12  |  Unknown: 3           │
├─────────────────────────────────────────────────────────────┤
│  Data Grid (sticky header, sortable columns)                │
│  ┌──┬────────┬──────────┬───────────┬────────┬──────────┐  │
│  │☐ │ UID    │ Password │ Status    │ Proxy  │ Note     │  │
│  ├──┼────────┼──────────┼───────────┼────────┼──────────┤  │
│  │☐ │ 123…  │ abc…     │ ● live    │ 1.2.3… │ …        │  │
│  │☑ │ 456…  │ xyz…     │ ● die     │ 5.6.7… │ …        │  │
│  └──┴────────┴──────────┴───────────┴────────┴──────────┘  │
│  [Right-click → Context Menu]                               │
├─────────────────────────────────────────────────────────────┤
│  Detail Panel (slide-in hoặc bottom panel khi chọn row)    │
└─────────────────────────────────────────────────────────────┘
```

#### Toolbar actions

| Button | Hành động |
|---|---|
| Import | Mở dialog paste/dán dữ liệu hàng loạt |
| Delete | Xoá accounts đã chọn (có confirm dialog) |
| Run | Chạy với accounts đang chọn |
| Verify | Xác thực tài khoản |
| Stop | Dừng tiến trình đang chạy |
| Filter | Dropdown lọc theo status, category |
| Search | Tìm kiếm realtime theo UID/email/note |

#### Context menu (right-click)

```
┌─────────────────────┐
│  Copy UID           │
│  Copy Full Data     │
│  Copy Password      │
│  Copy Email         │
│  Copy Cookie        │
│  ─────────────────  │
│  Run This Account   │
│  Verify This Account│
│  ─────────────────  │
│  View Detail        │
│  Edit Note          │
│  ─────────────────  │
│  Delete             │
└─────────────────────┘
```

#### Grid columns (40+ cột)

**Nhóm Tài khoản**

| Column | Key | Mô tả |
|---|---|---|
| # | id | Row index |
| Full Data | fullData | Dữ liệu thô gốc |
| UID | uid | Platform user ID |
| Password | password | Mật khẩu |
| 2FA | twofa | Backup code 2FA |
| Email | email | Email liên kết |
| Pass Mail | passMail | Mật khẩu email |
| Recovery Mail | mailRecovery | Email khôi phục |
| Cookie | cookie | Browser cookies |
| Token | token | Auth token |

**Nhóm Trạng thái**

| Column | Key | Mô tả |
|---|---|---|
| Status | status | live / die / checkpoint / new / unknown |
| Checkpoint | checkpoint | Loại checkpoint |
| Status Ads | statusAds | Trạng thái tài khoản quảng cáo |
| BM | bm | Business Manager |
| TKQC | tkqc | Tài khoản quảng cáo |
| Chat Support | chatSupport | Trạng thái hỗ trợ chat |

**Nhóm Chạy & Khác**

| Column | Key | Mô tả |
|---|---|---|
| Full Name | fullName | Tên hiển thị |
| Location | location | Quốc gia |
| Avatar | avatar | URL ảnh đại diện |
| Cover | cover | URL ảnh bìa |
| Phone | phone | Số điện thoại |
| Proxy | proxy | Proxy cấu hình |
| User Agent | userAgent | Browser UA |
| Note | note | Ghi chú thủ công |
| Note Run | noteRun | Ghi chú khi chạy |
| Import Time | importTime | Thời gian nhập |
| Category | category | Thư mục/nhóm |
| Last Run | lastRun | Lần chạy cuối |
| Run Proxy | runProxy | IP thực đang dùng (realtime) |
| Activity | activity | Hoạt động hiện tại |
| Source Code | sourceCode | Nguồn dữ liệu |

#### Grid features

- **Row selection**: checkbox check + highlight/selected (2 states khác nhau)
- **Drag select**: kéo chuột để chọn nhiều rows
- **Sticky header**: header cố định khi scroll
- **Sorting**: click header → asc/desc
- **Column visibility**: tắt/bật từng cột trong ViewSettings
- **Right-click context menu**: position theo chuột, auto-flip nếu gần cạnh màn hình
- **Double click**: mở detail panel
- **Real-time update**: `runProxy` + `status` update khi backend emit event
- **Display cap**: hiển thị tối đa 2000 rows để tránh lag

#### Import dialog

```
┌─────────────────────────────────────────┐
│  Import Accounts                    [×] │
├─────────────────────────────────────────┤
│  Paste dữ liệu vào đây (1 account/dòng) │
│  ┌─────────────────────────────────┐    │
│  │ uid|pass|email|...             │    │
│  │ uid|pass|email|...             │    │
│  └─────────────────────────────────┘    │
│                                         │
│  Định dạng: uid|pass|email|passmail|… │
│                                         │
│  [Cancel]                    [Import]   │
└─────────────────────────────────────────┘
```

#### Detail panel

Slide-in từ phải hoặc panel dưới, hiển thị **tất cả trường** của 1 account với layout 2 cột.  
Bao gồm: full data, status badges, timestamps, proxy info, notes.

---

### 4.2 Proxy Settings

**Route**: `/proxy-settings`  
**Mục đích**: Quản lý danh sách proxy server.

#### Layout trang

```
┌─────────────────────────────────────────────────────┐
│  Toolbar: [Add Proxy] [Delete] [Test Selected]      │
├──────┬──────────┬──────┬──────┬─────────┬──────────┤
│  #   │ Name     │ Host │ Port │ Type    │ Status   │
├──────┼──────────┼──────┼──────┼─────────┼──────────┤
│  1   │ Main     │ …    │ 1080 │ SOCKS5  │ ✓ OK    │
│  2   │ Backup   │ …    │ 8080 │ HTTP    │ ✗ Fail  │
└──────┴──────────┴──────┴──────┴─────────┴──────────┘

[Edit Form - bottom or side panel]
  Name: ____  Host: ____  Port: ____
  Type: [HTTP|SOCKS4|SOCKS5|HTTPS]
  Username: ____  Password: ____
  Note: ____
  [Save] [Test]
```

#### Proxy data model

```
id, name, host, port, username, password,
type (HTTP/SOCKS4/SOCKS5/HTTPS), note, lastTestResult
```

#### Test result

```
success: bool, latency: ms, ip: string, error?: string
```

---

### 4.3 Thiết lập chạy (Interaction Setup / Run Configuration)

**Route**: `/interaction-setup`  
**Mục đích**: Cấu hình profile chạy — verify, register, mail, UA, output.

#### Sections (accordion / collapsible)

##### Section 1: Nguồn tài khoản

```
Nguồn tài khoản
  ○ Folder (chọn thư mục chứa file .txt account)
    Path: [____________] [Browse]
  ○ CloneHV (lấy account từ dịch vụ CloneHV)
    Username: ____  Password: ____
    Product ID: [dropdown]  Amount: [number]
    [Check Stock]  → "Còn 150 tài khoản | 5,000đ/cái"
```

##### Section 2: Verify Configuration

```
Verify (Xác thực tài khoản)
  ☑ Bật verify
  
  Mail Provider: [dropdown]
    → Gmail Rent (ZeusX / DongVanFB / Rentgmail / ShopGmail9999 / MuaMail / EmailAPIInfo)
    → Temp Mail (TempMail.com / Guerrilla / ...)
    → OTP SMS (OTPCheap / OTPCodeSMS)
  
  [Khi chọn ZeusX]
    Account code: [dropdown các loại TK ZeusX]
    API Token: [____]
  
  [Khi chọn DongVanFB]
    Account type: [dropdown]
    API Token: [____]
  
  [Khi chọn Temp Mail]
    Domain: [____]   (ví dụ: @tempmail.net)
    Token/API key: [____]
  
  Thư mục Live:  [____________] [Browse]
  Thư mục Die:   [____________] [Browse]
  
  Delay check (s):    [  2  ] [?]
  Delay send code (s):[  5  ] [?]
  ☐ Gửi lại mã nếu không nhận được
```

##### Section 3: User Agent — Register Platform

```
User Agent — Register
  Chọn pool UA: [dropdown]
    → android_pool_1
    → iphone_pool_2
    → custom
  
  File UA: [____________] [Browse]  
  Số UA trong pool: 234
  
  ☐ Dùng raw UA (không format lại)
  
  [Simulate UA] → Preview 1 UA ngẫu nhiên từ pool
```

##### Section 4: User Agent — Verify Platform

Tương tự Section 3 nhưng riêng cho verify process.

##### Section 5: Cookie Initial

```
Cookie Initial
  File cookie khởi đầu: [____________] (auto-fill từ default path)
  [Mở file]  [Mở thư mục]
  
  Trạng thái: ● File tồn tại (245 bytes)
              ○ File rỗng — paste datr vào để dùng
```

##### Section 6: Lead Domain Mail

```
Lead Domain Mail
  Domains ưu tiên (phân tách bằng dấu phẩy):
  [@gmail.com, @yahoo.com, @outlook.com]
  [?] Dùng khi cần tạo email lead tự động
```

##### Section 7: Proxy — TempMail

```
Proxy cho TempMail
  [ip:port|user:pass]
  [ip:port]
  ...
  (load từ Config/Proxy/tempmail.txt)
  Số proxy: 12
```

##### Section 8: Proxy — Gmail

Tương tự Section 7, dùng riêng cho Gmail.

##### Section 9: Luồng đăng ký (Register Flow)

```
Luồng đăng ký
  Số thread đăng ký: [20] (1–600)
  
  Login method per platform: [dropdown per platform]
    Platform A: [Native / WebView / API]
    Platform B: [Native / WebView / API]
  
  Captcha Provider: [2Captcha | DeathByCaptcha | Anti-Captcha | ...]
    API Key: [____]
```

##### Section 10: Output & Kết quả

```
Thư mục kết quả:  [./result/] (auto-fill, không cho đổi)
[Mở thư mục kết quả]
```

#### Profile Management (xuất hiện ở đầu trang)

```
Profile: [dropdown chọn profile]  [New] [Clone] [Rename] [Delete]
[Export JSON] [Import JSON]
Auto-save: bật — lưu sau 1.5s khi không có thay đổi
```

---

### 4.4 Nguồn xác thực (Auth Source)

**Route**: `/auth-source`  
**Mục đích**: Cấu hình các nguồn cung cấp email/OTP để xác thực.

#### Nội dung

```
Auth Source Configuration
  
  ┌──────────────────────────────────────┐
  │  Provider list:                      │
  │  ● Gmail Rent                       │
  │    ├─ ZeusX          [configure]    │
  │    ├─ DongVanFB      [configure]    │
  │    ├─ Rentgmail      [configure]    │
  │    ├─ ShopGmail9999  [configure]    │
  │    ├─ MuaMail        [configure]    │
  │    └─ EmailAPIInfo   [configure]    │
  │                                      │
  │  ● Temp Mail                        │
  │    ├─ TempMail.com   [configure]    │
  │    ├─ Guerrilla      [configure]    │
  │    ├─ Mail30s        [configure]    │
  │    └─ ...            [configure]    │
  │                                      │
  │  ● OTP SMS                          │
  │    ├─ OTPCheap       [configure]    │
  │    └─ OTPCodeSMS     [configure]    │
  └──────────────────────────────────────┘
  
  [Configure panel bên phải khi click provider]
    Token / API Key: [____]
    Account Code: [dropdown nếu có]
    Pool size hiện tại: 45
    [Test Connection]
```

---

### 4.5 Tương tác (Interaction / Activity)

**Route**: `/tuong-tac`  
**Mục đích**: Cấu hình hành vi tương tác của tài khoản trên nền tảng.

#### Nội dung

```
Tương tác
  
  Feed
    ☑ Scroll feed sau đăng nhập
    Thời gian scroll (s): [10–60]
    
  Like & Comment
    ☐ Auto-like bài viết
    ☐ Auto-comment
    Comment pool: [____textarea____]
    
  Post timing
    Delay giữa các action (s): [min] → [max]
    
  Follow
    ☐ Auto-follow
    Target list: [____]
```

---

### 4.6 Đẩy tài khoản (Upload Site)

**Route**: `/upload-site`  
**Mục đích**: Upload/đồng bộ danh sách tài khoản lên hệ thống/web ngoài.

#### Layout

```
Upload Accounts to Site
  
  Endpoint URL: [https://____]
  API Key: [____]
  
  Chọn accounts để upload:
  ○ Tất cả accounts
  ○ Accounts đã chọn ([n] đang chọn)
  ○ Lọc theo status: [live | die | all]
  
  Batch size: [50]
  
  [Preview] [Upload Now]
  
  ─────────────────────────────────
  Lịch sử upload:
  ┌────────────┬──────┬────────┬────────┐
  │ Time       │ Count│ Status │ Detail │
  ├────────────┼──────┼────────┼────────┤
  │ 10:23:45   │  145 │ OK     │ [view] │
  │ 09:15:20   │   23 │ Fail   │ [view] │
  └────────────┴──────┴────────┴────────┘
```

#### Upload config model

```
endpoint: string
apiKey: string
batchSize: number
includeStatus: 'all' | 'live' | 'die'
fieldMapping: Record<string, string>
```

---

### 4.7 Thống kê (Statistics)

**Route**: `/reg-stats`  
**Mục đích**: Dashboard thống kê register/verify.

#### Layout

```
Thống kê
  
  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
  │  Total      │ │  Live       │ │  Die        │ │  Error      │
  │  1,234      │ │  856 (69%)  │ │  312 (25%)  │ │  66 (5%)    │
  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘
  
  [Chart: Line graph — accounts over time]
  
  Per Platform Breakdown:
  ┌──────────────┬───────┬─────┬─────┬──────────┐
  │ Platform     │ Total │ OK  │ Fail│ Rate     │
  ├──────────────┼───────┼─────┼─────┼──────────┤
  │ Platform A   │  450  │ 380 │  70 │ 84.4%    │
  │ Platform B   │  220  │ 165 │  55 │ 75.0%    │
  └──────────────┴───────┴─────┴─────┴──────────┘
  
  Mail Domain Stats:
  ┌──────────────┬────────┬──────┐
  │ Domain       │ Count  │ Rate │
  ├──────────────┼────────┼──────┤
  │ @gmail.com   │  623   │ 87%  │
  │ @yahoo.com   │  211   │ 72%  │
  └──────────────┴────────┴──────┘
  
  [Export CSV] [Reset Stats]
```

---

## 5. Cài đặt chung (General Settings)

**Route**: `/general-settings`  
**Mục đích**: Cấu hình toàn cục — threading, login method, captcha, IP provider.

### Layout trang

```
Cài đặt chung
Profile: [dropdown] [New] [Clone] [Rename] [Delete]

────────────────────────────────────────────────────

Section: Threading
  Số thread tối đa: [50] (1–600) [?]

Section: Login Methods
  Mỗi platform → dropdown chọn method:
  Platform A:  [Native | WebView | API v1 | API v2]
  Platform B:  [Native | WebView | API v1]

Section: Captcha
  Provider: [2Captcha | DeathByCaptcha | Anti-Captcha | CapMonster | NoCaptcha]
  API Key 2Captcha:         [____]
  API Key DeathByCaptcha:   [user:pass]
  API Key Anti-Captcha:     [____]
  API Key CapMonster:       [____]
  [Test key]

Section: IP Provider (dùng để đổi IP)
  Provider: [None | Proxy | VPN A | VPN B | Custom API]
  Token/API key: [____]
  Endpoint đổi IP: [____]

────────────────────────────────────────────────────
Auto-save: bật
```

### GeneralConfig model

```
maxThreads: number
loginMethods: Record<platform, method>
captchaProvider: string
captchaKeys: Record<provider, key>
ipProvider: string
ipProviderToken: string
accountSource: 'folder' | 'platform-api'
accountSourcePath: string
```

---

## 6. Hiển thị (View Settings)

**Route**: `/view-settings`  
**Mục đích**: Tùy chỉnh giao diện — cột hiển thị, mật độ bảng, bảo mật.

### Layout trang

```
Hiển thị
  
  ┌─────────────────────────────────────────────────┐
  │  Mật độ bảng                                    │
  │  ○ ▤ Compact    (row 28px, font 11px)           │
  │  ● ▦ Default    (row 36px, font 13px)           │
  │  ○ ▧ Comfortable (row 44px, font 14px)          │
  └─────────────────────────────────────────────────┘
  
  ┌─────────────────────────────────────────────────┐
  │  Bảo mật dữ liệu                               │
  │  ☐ Ẩn Password (hiển thị ••••••)               │
  │  ☐ Ẩn Cookie (hiển thị [truncated])             │
  │  ☐ Ẩn Token                                    │
  └─────────────────────────────────────────────────┘
  
  ┌─────────────────────────────────────────────────┐
  │  Cột hiển thị                                   │
  │                                                 │
  │  Nhóm: Tài khoản      [Bật tất cả] [Tắt tất cả]│
  │  ☑ UID    ☑ Password  ☑ 2FA   ☑ Email  …      │
  │                                                 │
  │  Nhóm: Trạng thái     [Bật tất cả] [Tắt tất cả]│
  │  ☑ Status  ☑ Checkpoint  ☑ StatusAds  …       │
  │                                                 │
  │  Nhóm: Chạy & Khác    [Bật tất cả] [Tắt tất cả]│
  │  ☑ Proxy  ☑ UA  ☑ Note  ☑ RunProxy  …        │
  │                                                 │
  │  Đang hiển thị: 18/34 cột                      │
  └─────────────────────────────────────────────────┘
```

### Preferences model

```
theme: 'dark' | 'light'
density: 'compact' | 'default' | 'comfortable'
columnVisibility: Record<columnKey, boolean>
maskPassword: boolean
maskCookie: boolean
maskToken: boolean
```

---

## 7. Data Model — Account

> Model đầy đủ, platform-agnostic. Mapping với HVR, đổi tên trường cho phù hợp nền tảng mới.

```csharp
public class Account
{
    public int    Id          { get; set; }
    public string Uid         { get; set; } = "";  // Platform user ID
    public string FullData    { get; set; } = "";  // Raw import string
    public string Password    { get; set; } = "";
    public string TwoFa       { get; set; } = "";  // 2FA backup codes
    public string Email       { get; set; } = "";
    public string PassMail    { get; set; } = "";  // Email password
    public string MailRecovery{ get; set; } = "";
    public string Cookie      { get; set; } = "";
    public string Token       { get; set; } = "";

    // Status
    public AccountStatus Status     { get; set; } = AccountStatus.Unknown;
    public string        Checkpoint { get; set; } = "";
    public string        StatusAds  { get; set; } = "";
    public string        Bm         { get; set; } = "";  // Business Manager
    public string        Tkqc       { get; set; } = "";  // Ad account
    public string        ChatSupport{ get; set; } = "";

    // Profile info
    public string FullName   { get; set; } = "";
    public string Location   { get; set; } = "";
    public string Avatar     { get; set; } = "";
    public string Cover      { get; set; } = "";
    public string Phone      { get; set; } = "";

    // Network
    public string Proxy     { get; set; } = "";
    public string UserAgent { get; set; } = "";
    public string RunProxy  { get; set; } = "";  // Realtime IP during run

    // Meta
    public string Note       { get; set; } = "";
    public string NoteRun    { get; set; } = "";
    public string ImportTime { get; set; } = "";
    public string Category   { get; set; } = "";
    public int?   CategoryId { get; set; }
    public string LastRun    { get; set; } = "";
    public string Activity   { get; set; } = "";
    public string SourceCode { get; set; } = "";
}

public enum AccountStatus { Live, Die, Checkpoint, New, Unknown }
```

---

## 8. Bridge Layer & Services

### Service contracts (platform-agnostic)

```
IAccountService
  list(filter)     → AccountListResult
  get(id)          → Account
  import(rawData)  → ImportResult
  delete(ids)      → DeleteResult

ISettingsService
  save(data)  → void
  load()      → SettingsData

IRunnerService
  run(config)       → void
  stop()            → void
  isRunning()       → bool
  runRegister(threads) → void
  stopRegister()    → void

IProxyService
  list()            → Proxy[]
  save(proxy)       → void
  delete(id)        → void
  test(id)          → ProxyTestResult

IProfileService
  list()            → ProfileInfo[]
  getActiveId()     → string
  setActive(id)     → void
  create(name)      → string
  clone(name)       → string
  rename(id, name)  → void
  delete(id)        → void

IFileDialogService
  openFolder()      → string
  openFile()        → string
  validatePath(p)   → bool

IEventBusService
  on(event, callback) → unsubscribe fn
  off(events)         → void

IResourceUsageService
  get()             → { ramMb, cpuPct }

IUploadSiteService
  save(config)      → void
  load()            → UploadSiteConfig

IStatsService
  getSummary()      → StatsSummary
  getTimeSeries()   → TimeSeriesData[]
  reset()           → void
```

### Real-time events

| Event | Payload | Khi nào |
|---|---|---|
| `runner:status` | `{ accountId, uid, message }` | Mỗi lần update 1 account |
| `runner:complete` | `{ total, success, failed, path }` | Xong toàn bộ |
| `runner:accounts-updated` | `{ ids: int[] }` | Batch update nhiều account |
| `runner:proxy-assigned` | `{ accountId, ip }` | Gán IP mới cho account |
| `runner:slot-assigned` | `{ slotId, uid, status, ua }` | Phân slot cho account |
| `register:complete` | `{ total, success }` | Register xong |

---

## 9. State Management

### Global state

```
AppState
  sidebarCollapsed: bool (persist)
  notifications: Notification[]
  connectionStatus: 'connected' | 'disconnected' | 'mock'

PreferencesState
  theme: 'dark' | 'light' (persist)
  density: 'compact' | 'default' | 'comfortable' (persist)
  columnVisibility: map<key, bool> (persist)
  maskPassword: bool (persist)
  maskCookie: bool (persist)

UploadLogState
  logs: UploadLog[]
```

### Module state (Accounts)

```
AccountsState
  accounts: Account[]
  accountsIndex: map<id, Account>  ← O(1) lookup
  loading: bool
  error: string?
  filter: AccountFilter
  isRunnerRunning: bool (persist — survive page nav)
  runProxyCache: LRU<id, ip>  (max 2000 entries)
  detailAccount: Account?
  showDetailPanel: bool
  total: int
```

### LRU Cache strategy

Khi backend emit `runner:proxy-assigned` → update `runProxyCache[id] = ip`.  
Khi `fetchAccounts()` → merge cache vào kết quả mới (tránh mất IP realtime).  
Khi cache > 2000 entries → drop 20% oldest.

---

## 10. Thiết kế lại cho C# WPF .NET 10

### Stack đề xuất

| Layer | Công nghệ |
|---|---|
| UI Framework | WPF (.NET 10) |
| MVVM | CommunityToolkit.Mvvm |
| Data Grid | WPF DataGrid hoặc Syncfusion DataGrid (advanced) |
| Navigation | Frame + NavigationService hoặc Prism |
| Dependency Injection | Microsoft.Extensions.DependencyInjection |
| Notification/Events | WeakReferenceMessenger (CommunityToolkit) |
| Settings persistence | System.Text.Json + isolated storage hoặc AppData |
| HTTP Client | HttpClient + System.Net.Http.Json |
| Async | Task/CancellationToken pattern |
| Theming | ResourceDictionary + DynamicResource |
| Packaging | Self-contained single-file exe (.NET 10 publish) |

### Không cần

- Wails (không dùng browser)
- Vue/Vite (không dùng web frontend)
- Go backend (logic viết thẳng vào C# services)

---

## 11. Architecture Map: Vue → WPF

| Vue/Wails concept | WPF equivalent |
|---|---|
| `App.vue` + AppLayout | `MainWindow.xaml` + `ShellView` |
| Vue Router | `Frame` + `NavigationService` (hoặc Prism `RegionManager`) |
| Pinia store | ViewModel + `ObservableObject` (CommunityToolkit.Mvvm) |
| `<script setup>` + composables | ViewModel + Service classes |
| Bridge contracts (interface) | C# interface (`IAccountService`, etc.) |
| Mock implementations | `XxxMockService` implementing same interface |
| Wails bindings | Actual C# service implementations (business logic trong cùng process) |
| EventBus (Wails runtime events) | `WeakReferenceMessenger` (CommunityToolkit) |
| `useAutoSave` composable | Timer + `PropertyChangedCallback` trong ViewModel |
| `useSelection` composable | `SelectionHelper` class (shared behavior) |
| `DataGrid.vue` component | WPF `DataGrid` + `DataGridColumn` definitions |
| Context menu | WPF `ContextMenu` + `MenuItem` bindings |
| Toast notifications | Custom `ToastControl` overlay hoặc `MaterialDesignInXaml` snackbar |
| Modal dialogs | `Window` dialog hoặc overlay `UserControl` |
| Sidebar collapse | `GridSplitter` + `ColumnDefinition.Width` animation |
| Theme toggle | `ResourceDictionary.MergedDictionaries` swap |
| StatusBar | WPF `StatusBar` control |
| TitleBar (frameless) | `WindowStyle=None` + custom `Border` drag-handle |
| Profile management | `IProfileService` + JSON file per profile trong AppData |
| Column visibility | `DataGridColumn.Visibility` bound to preferences |
| Auto-save | `DispatcherTimer` + debounce 1.5s |

---

## 12. Cấu trúc thư mục WPF đề xuất

```
PlatformTool/
├── App.xaml / App.xaml.cs             ← Entry point, DI setup
│
├── Shell/
│   ├── MainWindow.xaml                ← Chứa: TitleBar + Sidebar + Frame + StatusBar
│   ├── MainWindowViewModel.cs
│   ├── AppTitleBar.xaml               ← Frameless drag handle + min/max/close
│   ├── AppSidebar.xaml                ← Navigation menu (collapsible)
│   ├── AppSidebarViewModel.cs
│   └── AppStatusBar.xaml              ← CPU / RAM / connection
│
├── Views/                             ← Pages (UserControl), 1:1 với route
│   ├── AccountsView.xaml
│   ├── AccountsView.xaml.cs
│   ├── ProxySettingsView.xaml
│   ├── RunConfigView.xaml             ← "Thiết lập chạy"
│   ├── AuthSourceView.xaml
│   ├── InteractionView.xaml           ← "Tương tác"
│   ├── UploadSiteView.xaml
│   ├── StatisticsView.xaml
│   ├── GeneralSettingsView.xaml
│   └── ViewSettingsView.xaml
│
├── ViewModels/                        ← 1 ViewModel per View
│   ├── AccountsViewModel.cs
│   ├── ProxySettingsViewModel.cs
│   ├── RunConfigViewModel.cs
│   ├── AuthSourceViewModel.cs
│   ├── InteractionViewModel.cs
│   ├── UploadSiteViewModel.cs
│   ├── StatisticsViewModel.cs
│   ├── GeneralSettingsViewModel.cs
│   └── ViewSettingsViewModel.cs
│
├── Controls/                          ← Reusable controls (WPF UserControl)
│   ├── DataGrid/
│   │   ├── AccountDataGrid.xaml       ← Wrapped DataGrid với cấu hình column
│   │   └── AccountContextMenu.xaml
│   ├── Dialog/
│   │   ├── ImportDialog.xaml
│   │   ├── ConfirmDialog.xaml
│   │   └── AccountDetailPanel.xaml
│   ├── Settings/
│   │   ├── ProfileManager.xaml
│   │   ├── ProfileManagerViewModel.cs
│   │   ├── FieldHelp.xaml             ← Tooltip giải thích field
│   │   └── PresetBar.xaml
│   └── Feedback/
│       ├── ToastHost.xaml             ← Overlay container cho toast
│       └── ToastItem.xaml
│
├── Services/                          ← Business logic (no UI)
│   ├── Contracts/
│   │   ├── IAccountService.cs
│   │   ├── ISettingsService.cs
│   │   ├── IRunnerService.cs
│   │   ├── IProxyService.cs
│   │   ├── IProfileService.cs
│   │   ├── IFileDialogService.cs
│   │   ├── IEventBusService.cs
│   │   ├── IResourceUsageService.cs
│   │   ├── IUploadSiteService.cs
│   │   └── IStatsService.cs
│   │
│   ├── Implementation/
│   │   ├── AccountService.cs
│   │   ├── SettingsService.cs
│   │   ├── RunnerService.cs
│   │   ├── ProxyService.cs
│   │   ├── ProfileService.cs
│   │   ├── FileDialogService.cs
│   │   ├── ResourceUsageService.cs
│   │   ├── UploadSiteService.cs
│   │   └── StatsService.cs
│   │
│   └── Mock/                          ← Dev/test mode (implement same interfaces)
│       ├── MockAccountService.cs
│       ├── MockRunnerService.cs
│       └── MockEventBusService.cs
│
├── Models/                            ← Data models (POCO)
│   ├── Account.cs
│   ├── AccountFilter.cs
│   ├── Proxy.cs
│   ├── Flow.cs
│   ├── Profile.cs
│   ├── GeneralConfig.cs
│   ├── RunConfig.cs
│   ├── UploadSiteConfig.cs
│   └── AppPreferences.cs
│
├── Stores/                            ← Singleton state (shared across ViewModels)
│   ├── AppStore.cs                    ← sidebar state, notifications
│   ├── PreferencesStore.cs            ← theme, density, column visibility
│   ├── AccountsStore.cs               ← account list + LRU proxy cache
│   └── UploadLogStore.cs
│
├── Infrastructure/
│   ├── Navigation/
│   │   ├── INavigationService.cs
│   │   └── NavigationService.cs
│   ├── EventBus/
│   │   └── WeakEventBus.cs            ← Wrapper around WeakReferenceMessenger
│   ├── Settings/
│   │   └── JsonSettingsStorage.cs     ← Persist/load JSON settings
│   └── Helpers/
│       ├── ClipboardHelper.cs
│       ├── DebounceTimer.cs           ← Auto-save debounce
│       ├── LruCache.cs               ← LRU cache for runProxy
│       └── ResourceMonitor.cs         ← CPU/RAM polling
│
├── Themes/
│   ├── Dark.xaml                      ← Dark theme ResourceDictionary
│   ├── Light.xaml                     ← Light theme
│   ├── Colors.xaml                    ← Brand colors, semantic colors
│   ├── Typography.xaml               ← Font sizes, weights
│   └── Controls.xaml                  ← Shared control styles
│
├── Constants/
│   ├── ColumnDefinitions.cs          ← Account grid column metadata
│   ├── AccountContextMenuItems.cs    ← Context menu items
│   └── RouteNames.cs                 ← View name constants
│
└── Assets/
    ├── Icons/                         ← .ico, .png icons
    └── Fonts/
```

---

## 13. Checklist triển khai WPF

### Phase 1 — Shell & Foundation

- [ ] Setup project: .NET 10 WPF, CommunityToolkit.Mvvm, DI
- [ ] Frameless `MainWindow` (WindowStyle=None) với drag-handle TitleBar
- [ ] `AppSidebar` collapsible (9 menu items theo đúng danh sách trên)
- [ ] `AppHeader` với sidebar toggle, theme toggle
- [ ] `AppStatusBar` với CPU/RAM polling
- [ ] Navigation service (Frame-based)
- [ ] Theme system: Dark/Light với ResourceDictionary swap
- [ ] DI container setup (register tất cả services)
- [ ] Mock service implementations

### Phase 2 — Accounts Module (ưu tiên số 1)

- [ ] `AccountsView` + `AccountsViewModel`
- [ ] WPF DataGrid với 40+ cột, column visibility binding
- [ ] Toolbar với: Import, Delete, Run, Verify, Stop, Filter, Search
- [ ] Row selection (checkbox + highlight dual-state)
- [ ] Context menu (right-click)
- [ ] Import dialog
- [ ] Account detail panel
- [ ] Stats bar (Live/Die/Unknown count)
- [ ] Real-time update via EventBus
- [ ] `AccountsStore` với LRU proxy cache
- [ ] `IAccountService` + `MockAccountService`

### Phase 3 — Settings Pages

- [ ] `GeneralSettingsView` (threading, login methods, captcha, IP provider)
- [ ] `RunConfigView` (mail provider, verify config, UA pools, output path)
- [ ] `ProxySettingsView` (proxy CRUD + test)
- [ ] `ViewSettingsView` (density, column visibility, masking)
- [ ] `ProfileManager` control (create/clone/rename/delete/export/import)
- [ ] `PreferencesStore` với persist to AppData
- [ ] Auto-save debounce (1.5s)

### Phase 4 — Other Pages

- [ ] `AuthSourceView` (provider list + configure panel)
- [ ] `InteractionView` (feed, like, comment, timing config)
- [ ] `UploadSiteView` (upload config + history log)
- [ ] `StatisticsView` (summary cards + charts + per-platform table)

### Phase 5 — Runner Integration

- [ ] `IRunnerService` + implementation
- [ ] `WeakEventBus` với các events: `runner:status`, `runner:complete`, etc.
- [ ] Runner state persist (sidebar badge, toolbar Run/Stop state)
- [ ] `ResourceMonitor` polling CPU/RAM

### Phase 6 — Polish & Packaging

- [ ] Column visibility complete (ViewSettings → Accounts grid live-update)
- [ ] Theme toggle live-update không cần restart
- [ ] Toast notification overlay
- [ ] Confirm dialog cho destructive actions
- [ ] Error handling + user-friendly messages
- [ ] Single-file publish (.NET 10 self-contained)
- [ ] Application icon + manifest

---

## Ghi chú quan trọng cho nền tảng mới

### Thay thế "Facebook-specific" bằng generic

| HVR (Facebook) | Platform Tool (generic) |
|---|---|
| UID | Platform User ID |
| Cookie/Token | Auth credentials phù hợp platform |
| BM (Business Manager) | Tính năng tương đương platform |
| TKQC (Ad account) | Subscription / account tier |
| Checkpoint | Account restriction |
| Captcha provider | Verification provider |
| Lead domain mail | Target email domains |
| CloneHV | Account provider API |
| ZeusX / DongVanFB | Auth source providers của platform mới |

### Giữ nguyên

- Bridge/Service pattern (interface + implementation + mock)
- Module-first structure
- LRU proxy cache cho realtime data
- Auto-save debounce
- Profile management (save/load/clone settings)
- Column visibility system
- Real-time event bus
- Status bar CPU/RAM
- Dual-state row selection (checkbox vs highlight)
- Context menu
- Import dialog (paste bulk data)

### Thêm mới (so với HVR)

- Vì C# WPF: toàn bộ business logic nằm trong process → không cần bridge async
- Có thể dùng `ObservableCollection<Account>` trực tiếp với virtual scroll
- `BackgroundService` hoặc `Task.Run` cho runner thay vì Go goroutine
- `System.Diagnostics.Process.GetCurrentProcess()` cho resource monitoring
- Windows API P/Invoke nếu cần native features

---

*Blueprint này được tổng hợp từ phân tích toàn bộ dự án HVR (Hạ Vũ Clone) — Wails + Vue 3 + Go.*  
*Ngày tạo: 2026-05-18*
