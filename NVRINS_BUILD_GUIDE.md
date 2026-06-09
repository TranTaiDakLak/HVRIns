# NVRIns Build Guide — Clone HVRFb → NVRIns (Instagram)

Tài liệu này tổng hợp toàn bộ các thay đổi đã làm khi clone `HVR` → `HVRIns` và rebrand sang Instagram. Dùng để áp dụng lại quy trình tương tự khi build `NVRIns` từ `HVRFb`.

Mục lục:
1. [Phase 1 — Migration code (HVRFb → NVRIns)](#phase-1--migration-code-hvrfb--nvrins)
2. [Phase 2 — Rebrand visual sang Instagram](#phase-2--rebrand-visual-sang-instagram)
3. [Phase 3 — Build và verify](#phase-3--build-và-verify)
4. [Checklist hoàn tất](#checklist-hoàn-tất)

---

## Phase 1 — Migration code (HVRFb → NVRIns)

### 1.1 Copy repo

```powershell
Copy-Item -Recurse D:\NCS\HVRFb D:\NCS\NVRIns
Set-Location D:\NCS\NVRIns
```

### 1.2 Đổi module/app name

**`go.mod`:**
```go
module NVRIns
```

**`wails.json`:**
```json
{
  "name": "NVRIns",
  "outputfilename": "NVRIns"
}
```

### 1.3 Rename folder `internal/facebook` → `internal/instagram`

```powershell
Move-Item internal\facebook internal\instagram
```

### 1.4 Bulk replace imports + symbols

Tạo script PowerShell `migrate.ps1`:

```powershell
$root = "D:\NCS\NVRIns"
$files = Get-ChildItem -Path $root -Recurse -Include *.go -File
foreach ($f in $files) {
    $content = [IO.File]::ReadAllText($f.FullName)
    $orig = $content
    # Đổi import path
    $content = $content -replace '"HVRFb/internal/facebook', '"NVRIns/internal/instagram'
    $content = $content -replace '"HVRFb/', '"NVRIns/'
    # Đổi package declaration
    $content = $content -replace '\bpackage facebook\b', 'package instagram'
    # Đổi symbol reference (chỉ những UpperCase để tránh dính facebook.com)
    $content = $content -replace '\bfacebook\.([A-Z])', 'instagram.$1'
    if ($content -ne $orig) {
        [IO.File]::WriteAllText($f.FullName, $content)
        Write-Output ("MOD " + $f.FullName.Substring($root.Length+1))
    }
}
```

Chạy:
```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\migrate.ps1
```

**Lưu ý quan trọng**: regex `\bfacebook\.([A-Z])` chỉ match symbol reference (`facebook.Session`, `facebook.RegInput`), KHÔNG match `facebook.com` trong URL strings — sẽ stub xử lý sau.

### 1.5 Stub Facebook-specific platforms

Theo migration guide gốc:
- Giữ `register/web`, `verify/web`, `register/android`, `verify/android` làm khung
- Các platform `s545`–`s560v3` (Facebook API version) **không có nghĩa với Instagram** — stub trả `unsupported platform` thay vì copy logic Facebook
- Các token Facebook (`fb_dtsg`, `jazoest`, `lsd`, `datr`, `c_user`) để TODO chờ Instagram equivalent
- UA pool `FBAN/FB4A`, `FBPN/com.facebook.katana` để TODO

### 1.6 Update runtime paths trong `app.go`

```go
// Cũ
filepath.Join(appData, "HVRFb", "logs")
filepath.Join(docs, "HVRFb", "result")
filepath.Join(docs, "HVRFb", "Config")

// Mới
filepath.Join(appData, "NVRIns", "logs")
filepath.Join(docs, "NVRIns", "result")
filepath.Join(docs, "NVRIns", "Config")
```

### 1.7 Regenerate Wails bindings

```powershell
wails generate module
```

Hoặc chỉ chạy `wails dev` lần đầu để bindings tự generate.

---

## Phase 2 — Rebrand visual sang Instagram

### 2.1 App icon — Instagram gradient

**File:** `build/appicon.png` + `build/windows/icon.ico`

**Yêu cầu:**
- 1024x1024 PNG, rounded corner (radius 160)
- Background: gradient diagonal top-left → bottom-right qua các điểm:
  - `#405DE6` (blue-purple) → `#833AB4` (purple) → `#E1306C` (pink) → `#FD1D1D` (red) → `#FCAF45` (yellow-orange)
- Chữ "NVRIns" (hoặc tên app) ở giữa, font Arial Bold ~280pt, màu trắng
- Subtitle "Ha Vu VIP PRO" ở dưới, font Arial ~66pt, màu trắng 73% opacity

**Script Python generate** (cần Pillow):

```python
from PIL import Image, ImageDraw, ImageFont

SIZE = 1024
RADIUS = 160

STOPS = [
    (0.0,  (64,  93,  230)),   # #405DE6
    (0.25, (131, 58,  180)),   # #833AB4
    (0.5,  (225, 48,  108)),   # #E1306C
    (0.75, (253, 29,  29)),    # #FD1D1D
    (1.0,  (252, 175, 69)),    # #FCAF45
]

def interp(t, stops):
    for i in range(len(stops)-1):
        t0,c0 = stops[i]; t1,c1 = stops[i+1]
        if t0 <= t <= t1:
            f = (t-t0)/(t1-t0)
            return tuple(int(c0[j] + f*(c1[j]-c0[j])) for j in range(3))
    return stops[-1][1]

img = Image.new('RGBA', (SIZE, SIZE))
px = img.load()
for y in range(SIZE):
    for x in range(SIZE):
        r,g,b = interp((x+y)/(2*(SIZE-1)), STOPS)
        px[x,y] = (r,g,b,255)

mask = Image.new('L', (SIZE, SIZE), 0)
ImageDraw.Draw(mask).rounded_rectangle([0,0,SIZE-1,SIZE-1], radius=RADIUS, fill=255)
img.putalpha(mask)

draw = ImageDraw.Draw(img)
fb = ImageFont.truetype('C:/Windows/Fonts/arialbd.ttf', 280)
fs = ImageFont.truetype('C:/Windows/Fonts/arial.ttf', 66)

main = "NVRIns"  # đổi text nếu cần
b = draw.textbbox((0,0), main, font=fb)
tx = (SIZE - (b[2]-b[0]))//2 - b[0]
draw.text((tx, 310), main, font=fb, fill=(255,255,255,255))

sub = "Ha Vu VIP PRO"
b = draw.textbbox((0,0), sub, font=fs)
sx = (SIZE - (b[2]-b[0]))//2 - b[0]
draw.text((sx, 730), sub, font=fs, fill=(255,255,255,185))

img.save(r'D:\NCS\NVRIns\build\appicon.png', 'PNG')

# Generate .ico cho Windows
ico = Image.open(r'D:\NCS\NVRIns\build\appicon.png').convert('RGBA')
sizes = [16, 32, 48, 64, 128, 256]
frames = [ico.resize((s,s), Image.LANCZOS) for s in sizes]
frames[0].save(r'D:\NCS\NVRIns\build\windows\icon.ico', format='ICO',
               sizes=[(s,s) for s in sizes], append_images=frames[1:])
```

### 2.2 Design tokens — Instagram palette

**File:** `frontend/src/styles/tokens.css`

```css
:root {
  /* === Brand — Instagram palette === */
  --brand-primary: #E1306C;
  --brand-primary-hover: #C41E5A;
  --brand-primary-active: #a8185c;
  --brand-primary-bg: rgba(225, 48, 108, 0.14);
  --brand-primary-border: rgba(225, 48, 108, 0.35);
  --brand-gradient: linear-gradient(135deg, #833AB4, #E1306C, #FD1D1D);

  /* === Accent — dùng cho checkbox, tab active, section highlight === */
  --accent: #E1306C;
  --accent-hover: #C41E5A;
  --accent-bg: rgba(225, 48, 108, 0.12);
  --accent-border: rgba(225, 48, 108, 0.30);

  /* === Surfaces === */
  --surface-base: #0f1117;
  --surface-elevated: #161b22;
  --surface-sunken: #0d1117;
  --sidebar-bg: #0e0c18;       /* sidebar tối hơn header, tông tím */

  /* === Borders === */
  --border-focus: #E1306C;     /* focus ring Instagram pink */
  /* ... các token khác giữ nguyên */
}
```

**File:** `frontend/src/styles/light.css`

```css
[data-theme="light"] {
  --border-focus: var(--accent);
  --brand-primary: #E1306C;
  --brand-primary-hover: #C41E5A;
  --brand-primary-active: #a8185c;
  --brand-primary-bg: rgba(225, 48, 108, 0.09);
  --brand-primary-border: rgba(225, 48, 108, 0.30);
  --brand-gradient: linear-gradient(135deg, #833AB4, #E1306C, #FD1D1D);
  --accent: #E1306C;
  --accent-hover: #C41E5A;
  --accent-bg: rgba(225, 48, 108, 0.09);
  --accent-border: rgba(225, 48, 108, 0.25);

  /* Primary alias (proxy-mail-tab) */
  --primary-text: var(--brand-primary-active);
  --primary-subtle: rgba(225,48,108, 0.10);
  --primary-solid: rgba(225,48,108, 0.40);
}

/* Sidebar light mode */
[data-theme="light"] .sidebar__item--active {
  background: #ffffff;
  color: var(--accent);
  font-weight: 600;
  box-shadow: inset 3px 0 0 var(--accent), 0 1px 3px rgba(0,0,0,0.08);
}
[data-theme="light"] .sidebar__badge {
  background: var(--accent-bg);
  color: var(--accent);
}

/* Grid row selected */
[data-theme="light"] .data-grid__row--selected { background: rgba(225,48,108,0.12); }

/* Input focus */
[data-theme="light"] input:focus,
[data-theme="light"] textarea:focus,
[data-theme="light"] select:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px var(--accent-border);
}
```

### 2.3 Sidebar tách màu khỏi header

**File:** `frontend/src/components/shell/AppSidebar.vue`

```css
.sidebar {
  width: var(--sidebar-width);
  background: var(--sidebar-bg, var(--surface-elevated));
  border-right: 2px solid transparent;
  border-image: linear-gradient(180deg, #405DE6 0%, #833AB4 30%, #E1306C 60%, #FD1D1D 80%, #FCAF45 100%) 1;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  transition: width var(--transition-normal);
  overflow: hidden;
}
```

Giải thích:
- `--sidebar-bg: #0e0c18` (tím tối) — phân biệt với header `#161b22`
- `border-right` dùng `border-image` gradient Instagram làm accent line

### 2.4 TitleBar — Instagram logo

**File:** `frontend/src/components/shell/AppTitleBar.vue`

Thay SVG cũ bằng icon Instagram (rounded square + circle + dot):

```html
<svg class="titlebar__icon" width="16" height="16" viewBox="0 0 24 24" fill="none">
  <defs>
    <linearGradient id="ig-grad-title" x1="0%" y1="100%" x2="100%" y2="0%">
      <stop offset="0%" stop-color="#FCAF45"/>
      <stop offset="35%" stop-color="#E1306C"/>
      <stop offset="70%" stop-color="#833AB4"/>
      <stop offset="100%" stop-color="#405DE6"/>
    </linearGradient>
  </defs>
  <rect x="2" y="2" width="20" height="20" rx="6" ry="6" stroke="url(#ig-grad-title)" stroke-width="2" fill="none"/>
  <circle cx="12" cy="12" r="4.5" stroke="url(#ig-grad-title)" stroke-width="2" fill="none"/>
  <circle cx="17.5" cy="6.5" r="1.2" fill="url(#ig-grad-title)"/>
</svg>
```

### 2.5 Run/Stop button — Instagram gradient

**File:** `frontend/src/modules/accounts/components/AccountsToolbar.vue`

```css
/* Nút Chạy: tím → hồng → đỏ */
.toolbar__btn--run {
  background: linear-gradient(135deg, #833AB4, #E1306C, #FD1D1D);
  border-color: #E1306C;
  color: white;
  font-weight: 600;
  min-width: 80px;
}
.toolbar__btn--run:hover:not(:disabled) {
  background: linear-gradient(135deg, #6d2f9a, #c4235c, #e01010);
  opacity: 0.92;
}

/* Nút Dừng: hồng → đỏ */
.toolbar__btn--stop {
  background: linear-gradient(135deg, #E1306C, #FD1D1D);
  border-color: #FD1D1D;
  color: white;
  font-weight: 700;
  min-width: 80px;
  box-shadow: 0 0 0 1px rgba(225, 48, 108, 0.4);
}
.toolbar__btn--stop:hover:not(:disabled) {
  background: linear-gradient(135deg, #c4235c, #e01010);
}

/* Nút "Đang dừng": xanh-tím → tím */
.toolbar__btn--stopping {
  background: linear-gradient(135deg, #405DE6, #833AB4);
  border-color: #833AB4;
  color: white;
  font-weight: 700;
  cursor: not-allowed;
  opacity: 0.85;
}
```

### 2.6 Primary button (Import từ file, Thoát) — Instagram gradient

**File:** `frontend/src/components/ui/BaseButton.vue`

```css
.base-btn--primary {
  background: var(--brand-gradient, linear-gradient(135deg, #833AB4, #E1306C, #FD1D1D));
  color: #fff;
  border-color: var(--brand-primary);
}
.base-btn--primary:hover:not(:disabled) {
  background: linear-gradient(135deg, #6d2f9a, #c4235c, #e01010);
  border-color: var(--brand-primary-hover);
}
.base-btn--primary:active:not(:disabled) {
  background: var(--brand-primary-active);
}
```

**File:** `frontend/src/pages/AccountsPage.vue` (`.accounts-page__cta`)

```css
.accounts-page__cta {
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-md);
  background: var(--brand-gradient, linear-gradient(135deg, #833AB4, #E1306C, #FD1D1D));
  color: white;
  font-weight: 600;
  border: none;
}
```

### 2.7 Gộp footer thành 1 dòng (dùng Vue Teleport)

**File:** `frontend/src/components/shell/AppStatusBar.vue`

Bỏ prop `stats`, thêm slot teleport target + icon-only buttons:

```html
<template>
  <footer class="status-bar">
    <!-- Page-specific slot — filled by active page via Teleport -->
    <div id="status-bar-page-slot" class="status-bar__page-slot"></div>

    <!-- Resource usage -->
    <span class="status-bar__item status-bar__resource">CPU: {{ cpuPct.toFixed(1) }}%</span>
    <span class="status-bar__divider">|</span>
    <span class="status-bar__item status-bar__resource">RAM: {{ ramMb.toFixed(1) }} MB</span>
    <span class="status-bar__divider">|</span>

    <!-- Icon-only buttons với CSS tooltip -->
    <div class="status-bar__icon-wrap" data-tip="Dọn RAM (buffer + idle TCP)">
      <button class="status-bar__icon-btn" :class="{ 'status-bar__icon-btn--spinning': cleaning }"
              :disabled="cleaning" @click="softCleanup">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6"/>
          <path d="M10 11v6M14 11v6"/>
        </svg>
      </button>
    </div>
    <div class="status-bar__icon-wrap" data-tip="Reload UI (mất state hiện tại)">
      <button class="status-bar__icon-btn status-bar__icon-btn--warn" @click="hardReload">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
          <path d="M3 3v5h5"/>
        </svg>
      </button>
    </div>
    <span class="status-bar__divider">|</span>

    <span class="status-bar__item">Hạ Vũ {{ appVersion }}</span>
  </footer>
</template>

<style scoped>
.status-bar {
  height: var(--statusbar-height);
  background: var(--surface-elevated);
  border-top: 1px solid var(--border-subtle);
  display: flex;
  align-items: center;
  padding: 0 var(--space-3);
  gap: var(--space-3);
  font-size: var(--font-size-xs);
  color: var(--text-muted);
  flex-shrink: 0;
}

.status-bar__page-slot {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  overflow: hidden;
}

.status-bar__item { display: flex; align-items: center; white-space: nowrap; }
.status-bar__resource { color: var(--text-secondary); }
.status-bar__divider { color: var(--border-default); }

/* Icon-only button + CSS tooltip */
.status-bar__icon-wrap { position: relative; display: flex; align-items: center; }
.status-bar__icon-wrap::after {
  content: attr(data-tip);
  position: absolute;
  bottom: calc(100% + 6px);
  left: 50%;
  transform: translateX(-50%);
  white-space: nowrap;
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  color: var(--text-primary);
  font-size: 11px;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  pointer-events: none;
  opacity: 0;
  transition: opacity 120ms;
  box-shadow: var(--shadow-md);
  z-index: var(--z-tooltip);
}
.status-bar__icon-wrap:hover::after { opacity: 1; }

.status-bar__icon-btn {
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-muted);
  width: 22px;
  height: 22px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 120ms, color 120ms, border-color 120ms;
  padding: 0;
}
.status-bar__icon-btn:hover:not(:disabled) {
  background: var(--surface-hover-strong);
  border-color: var(--border-default);
  color: var(--text-primary);
}
.status-bar__icon-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.status-bar__icon-btn--warn:hover { border-color: var(--warning-solid); color: var(--warning-text); }
.status-bar__icon-btn--spinning { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
```

**File:** `frontend/src/pages/AccountsPage.vue`

Bọc stats bar trong `<Teleport>`:

```html
<!-- Stats bar — teleport vào slot của AppStatusBar để gộp thành 1 dòng -->
<Teleport to="#status-bar-page-slot">
<div class="accounts-stats-bar">
  <template v-if="isRegisterRunning || registerThreads.size > 0">
    <!-- ... existing REG/VERIFY/IDLE templates ... -->
  </template>
</div>
</Teleport>
```

Đổi CSS class:

```css
.accounts-stats-bar {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  user-select: none;
}
/* các .stats-item* giữ nguyên */
```

**Lưu ý**: Vue scoped CSS data-attribute vẫn áp dụng cho teleported content, nên `.stats-item*` không cần đổi gì.

### 2.8 Bulk replace hard-coded blue/cyan trong các page

Trong InteractionSetupPage, GeneralSettingsPage, AuthSourcePanel có nhiều chỗ hard-code `#4fc3f7` (cyan) và `#3b82f6` (Tailwind blue). Chạy script:

```python
import re
files = [
    'frontend/src/pages/InteractionSetupPage.vue',
    'frontend/src/pages/GeneralSettingsPage.vue',
    'frontend/src/modules/auth-source/components/AuthSourcePanel.vue',
]
replacements = [
    (r'(?<!var\(--accent, )#4fc3f7\b(?!\s*\))', 'var(--accent)'),
    (r'(?<!var\(--accent-hover, )#3b82f6\b(?!\s*\))', 'var(--accent)'),
    (r'rgba\(79\s*,\s*195\s*,\s*247\s*,', 'rgba(225,48,108,'),
    (r'var\(--accent,\s*#4fc3f7\)', 'var(--accent)'),
]
for path in files:
    content = open(path, encoding='utf-8').read()
    for pat, repl in replacements:
        content = re.sub(pat, repl, content, flags=re.IGNORECASE)
    open(path, 'w', encoding='utf-8').write(content)
```

Sau replacement, tất cả checkbox `accent-color`, tab active, section highlight, badge sẽ tự pick up Instagram pink qua `var(--accent)`.

---

## Phase 3 — Build và verify

```powershell
# Frontend
cd frontend
npm install
npm run build       # phải pass

# Backend
cd ..
go build ./...       # phải compile, có thể có warning về stubbed platforms

# Full Wails build
wails build         # sinh ra NVRIns.exe trong build/bin/
```

Verify chạy thực tế:
```powershell
wails dev           # mở app, kiểm tra:
                    # - icon Instagram gradient hiện đúng
                    # - sidebar tối hơn header, có gradient line phải
                    # - logo Instagram ở titlebar
                    # - footer 1 dòng, stats Accounts hiện khi vào tab
                    # - chuyển tab khác, footer chỉ còn CPU/RAM
                    # - hover 2 nút icon thấy tooltip
                    # - nút Chạy/Dừng/Import/Thoát đều gradient pink
                    # - checkbox khi tick là pink
                    # - sidebar item active có accent pink
```

---

## Checklist hoàn tất

### Code migration
- [ ] `go.mod` là `module NVRIns`
- [ ] `wails.json` name/outputfilename là `NVRIns`
- [ ] Không còn import `HVRFb/internal/...`
- [ ] Không còn import `internal/facebook` trong backend
- [ ] Có `internal/instagram` build được
- [ ] `app.go` import platform Instagram và gọi `instagram.*`
- [ ] `internal/runner` gọi `instagram.*`
- [ ] Config rieng Facebook (`Fbapp`, `datr_pool.txt`, FBAN UA) đã stub/TODO
- [ ] Runtime paths (`%APPDATA%/NVRIns`, `Documents/NVRIns`) đã đổi
- [ ] Wails bindings đã regenerate

### Visual rebrand
- [ ] `build/appicon.png` — Instagram gradient, chữ trắng
- [ ] `build/windows/icon.ico` — multi-size từ PNG mới
- [ ] `tokens.css` — `--brand-primary: #E1306C`, `--accent: #E1306C`, `--brand-gradient`, `--sidebar-bg: #0e0c18`
- [ ] `light.css` — Windows blue → Instagram pink, sidebar/grid/input focus
- [ ] `AppSidebar.vue` — `--sidebar-bg` + `border-image` gradient
- [ ] `AppTitleBar.vue` — Instagram camera SVG logo
- [ ] `AccountsToolbar.vue` — nút Chạy/Dừng/Stopping gradient Instagram
- [ ] `BaseButton.vue` primary variant — `--brand-gradient`
- [ ] `AccountsPage.vue` `.accounts-page__cta` — `--brand-gradient`
- [ ] `AppStatusBar.vue` — teleport slot + 2 icon buttons với CSS tooltip
- [ ] `AccountsPage.vue` stats bar wrap `<Teleport to="#status-bar-page-slot">`
- [ ] InteractionSetupPage / GeneralSettingsPage / AuthSourcePanel — `#4fc3f7`, `#3b82f6` đã replace bằng `var(--accent)`

### Build
- [ ] `npm run build` pass
- [ ] `go build ./...` pass (hoặc danh sách package được phép skip)
- [ ] `wails build` tạo `NVRIns.exe`
- [ ] Chạy thực tế: tất cả UI element rebrand đúng

---

## Ghi chú quan trọng

1. **Instagram protocol chưa port**: Phase 1 chỉ tạo skeleton build-được. Các flow Instagram thực tế (endpoint, header, cookie name, token parser) phải port riêng từng package. Stub trả `unsupported platform` là đủ cho build pass.

2. **Token Facebook không port blind**: `fb_dtsg`, `jazoest`, `lsd`, `datr`, `c_user` là Facebook-specific. Instagram có các session token khác (`csrftoken`, `ds_user_id`, `sessionid`, `mid`, `ig_did`, `rur`). Khi port từng flow phải thay đúng nghĩa.

3. **UA pool**: `FBAN/FB4A`, `FBPN/com.facebook.katana` là Facebook app. Instagram UA pool dùng `Instagram XXX.X.X.XXX Android` hoặc `Instagram XXX iPhone OS`. File `Config/UserAgent/*` cần thay nội dung.

4. **Wails bindings**: Sau khi đổi Go type/package, **bắt buộc** regenerate bindings (`wails generate module` hoặc chạy `wails dev` lần đầu) — nếu không frontend sẽ vẫn dùng symbol cũ.

5. **Vue Teleport**: Stats bar trong AccountsPage giữ nguyên reactive data (computed `pageStats`, `isRegisterRunning`, etc.), chỉ render ra slot khác. Khi `keep-alive` cache AccountsPage và user chuyển tab, teleport tự động unmount → footer trống. Khi quay lại tab, mount lại → stats hiện. Không cần store trung gian.

6. **CSS tooltip vs `title` attribute**: Dùng CSS `::after` + `attr(data-tip)` thay vì native `title` để tránh delay 1-2s của trình duyệt và styling consistent với theme.
