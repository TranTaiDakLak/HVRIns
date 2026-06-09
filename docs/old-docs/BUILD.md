# Build Guide — HVR

> Mục tiêu: build ra 1 file `.exe` duy nhất, nhỏ gọn, khó reverse engineering, chạy được trên máy sạch (không cần Go/Node/.NET runtime), đủ chất lượng gửi khách hàng.

---

## ⚡ QUICK START — Lệnh dùng hàng ngày

### 1. Đang code — chạy dev mode (hot reload)

```bash
wails dev
```

- Mở app trực tiếp, frontend tự reload khi sửa file
- Version hiện `dev` ở status bar
- **Dùng khi đang phát triển, KHÔNG dùng để gửi khách**

---

### 2. Build gửi khách — Git Bash / Terminal bash

```bash
wails build -platform windows/amd64 -ldflags "-s -w -H windowsgui -X main.AppVersion=$(date +%m.%d.%H.%M)" -trimpath
```

- `-s -w` → strip debug info, giảm size
- `-H windowsgui` → ẩn cửa sổ console đen
- `-X main.AppVersion=$(date +%m.%d.%H.%M)` → inject version theo giờ build, ví dụ `v04.19.14.30`
- `-trimpath` → xóa đường dẫn máy build khỏi binary
- **KHÔNG có `-clean`** → giữ nguyên các file/thư mục khác trong `build/bin/` (Config, result...)
- Output: `build/bin/HVR.exe`

---

### 3. Build gửi khách — PowerShell

```powershell
wails build -platform windows/amd64 -ldflags "-s -w -H windowsgui -X main.AppVersion=$((Get-Date).ToString('MM.dd.HH.mm'))" -trimpath
```

- Tương tự lệnh bash, nhưng dùng cú pháp PowerShell để lấy ngày giờ
- Output: `build/bin/HVR.exe`

---

> 💡 **Version format:** `vMM.DD.HH.mm` — tháng.ngày.giờ.phút lúc build
> Ví dụ build lúc 14:30 ngày 19/04 → status bar hiện `HVR v04.19.14.30`

---

## 1. Vì sao chọn Go + Wails thay vì C# AOT?

| Tiêu chí | Go + Wails | C# AOT Native |
|---|---|---|
| **Binary size** (release) | 15–25 MB | 50–120 MB |
| **Cold start** | < 100ms | 200–500ms |
| **RAM idle** | 30–60 MB | 80–200 MB |
| **Runtime dependency** | Zero — single exe | Cần .NET hoặc bundle to |
| **Reverse engineering** | Khó (no IL metadata, type info bị strip) | Dễ hơn (vẫn có IL footprints) |
| **Obfuscation tools** | `garble` — rename + string encrypt + strip | Dotfuscator / ConfuserEx (pay, không ổn) |
| **Cross-compile** | 1 lệnh `GOOS=windows` | Phức tạp, toolchain nặng |
| **Static linking** | Mặc định | Khó, kéo theo CoreCLR |
| **AV false-positive** | Thấp | Cao hơn do AOT compiler signature |

→ **Go là lựa chọn chiến lược đúng** cho tool desktop gửi khách, đặc biệt khi bảo mật + độ gọn gàng quan trọng.

---

## 2. Lệnh build hàng ngày

### Dev (hot reload — dùng khi đang code)
```bash
wails dev
```

### Build release với version tự động theo thời gian build

Format version: `vMM.DD.HH.mm` — ví dụ `v04.19.01.29` = tháng 4, ngày 19, 01 giờ 29 phút.

> ⚠️ **Không dùng `-clean`** — flag đó xóa toàn bộ `build/bin/` kể cả các file/thư mục khác (Config, result...) trước khi build.

**Git Bash / bash:**
```bash
wails build -platform windows/amd64 \
  -ldflags "-s -w -H windowsgui -X main.AppVersion=$(date +%m.%d.%H.%M)" \
  -trimpath
```

**PowerShell:**
```powershell
wails build -platform windows/amd64 `
  -ldflags "-s -w -H windowsgui -X main.AppVersion=$((Get-Date).ToString('MM.dd.HH.mm'))" `
  -trimpath
```

Version sẽ tự động hiện ở góc dưới phải status bar: `HVR v04.19.01.29`

---

## 3. Yêu cầu môi trường

```bash
go version        # >= 1.21 (project đang xài 1.25)
node --version    # >= 18
wails version     # v2.11+
```

Cài Wails CLI nếu chưa có:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Cài thêm tool tối ưu (tuỳ chọn):

```bash
go install mvdan.cc/garble@latest    # obfuscator
# UPX: https://upx.github.io/  → thêm vào PATH
```

---

## 3. Build Release — Lệnh tối ưu nhất (KHUYẾN NGHỊ)

Đây là lệnh tối ưu **thực sự chạy được**, đã verify pattern với Wails v2:

### Windows (bash / Git Bash)

```bash
wails build \
  -platform windows/amd64 \
  -ldflags "-s -w -H windowsgui -X main.version=1.0.0 -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -trimpath \
  -clean \
  -tags production
```

### Windows (CMD / PowerShell)

```powershell
wails build -platform windows/amd64 -ldflags "-s -w -H windowsgui" -trimpath -clean -tags production
```

Output: `build/bin/HVR.exe` (~18–25 MB)

---

### Giải thích từng flag

| Flag | Tác dụng |
|---|---|
| `-s` | Strip symbol table → giảm size + khó debug reverse |
| `-w` | Strip DWARF debug info → giảm size thêm |
| `-H windowsgui` | **Ẩn console window đen** khi chạy (bắt buộc cho GUI app) |
| `-X main.version=...` | Inject version vào binary runtime |
| `-trimpath` | **Xoá path máy build** khỏi binary (bảo mật, tránh lộ tên user) |
| `-clean` | Clean cache `build/bin` trước khi build — tránh file cũ lẫn vào |
| `-tags production` | Activate build tag `production` (nếu code dùng) |

---

## 4. Build Maximum Security — Có Obfuscation (garble)

`garble` xáo tên function, method, struct field và **mã hoá string literals**. Khó RE hơn hẳn.

### ⚠️ Lưu ý quan trọng về garble + Wails

Garble **KHÔNG** chạy được qua `GOFLAGS="-toolexec=garble"` (đây là misconception phổ biến — sai syntax). Garble là wrapper thay thế lệnh `go build`, không phải toolexec tool.

**Cách đúng để dùng garble với Wails:**

#### Phương án A — Build 2 bước (clean & chắc chắn)

```bash
# Bước 1: Build frontend
cd frontend
npm ci
npm run build
cd ..

# Bước 2: Generate Wails bindings (nếu có)
wails generate module

# Bước 3: Build Go binary bằng garble thay go build
garble -literals -tiny build \
  -tags production,desktop \
  -ldflags "-s -w -H windowsgui -X main.version=1.0.0" \
  -trimpath \
  -o build/bin/HVR.exe \
  .
```

| Garble flag | Tác dụng |
|---|---|
| `-literals` | Mã hoá toàn bộ string literal trong binary |
| `-tiny` | Bỏ line numbers + thêm metadata → binary nhỏ hơn, khó debug hơn |

**Nhược điểm:** Phương án này **không có** icon + manifest + resource mà Wails thường đóng gói. Dùng khi không cần icon fancy.

#### Phương án B — Wrapper script (giữ nguyên Wails pipeline)

Tạo file `go-wrapper.bat` (Windows) để ép Wails dùng garble:

```batch
@echo off
REM go-wrapper.bat
if "%1"=="build" (
    garble -literals -tiny %*
) else (
    go.exe %*
)
```

Đặt wrapper vào đầu PATH, rename/symlink thành `go.exe` khi build:

```bash
# Trước build
set PATH=C:\path\to\wrapper;%PATH%

# Build như bình thường
wails build -platform windows/amd64 -ldflags "-s -w -H windowsgui" -trimpath -clean

# Sau build
# Khôi phục PATH
```

→ Giữ được icon, manifest, installer của Wails + có obfuscation.

---

## 5. Build kèm Installer (gửi khách cần wizard cài đặt)

```bash
wails build -platform windows/amd64 -nsis -ldflags "-s -w -H windowsgui" -trimpath -clean
```

Yêu cầu: cài [NSIS](https://nsis.sourceforge.io/Download) và thêm vào PATH.

Output:
- `build/bin/HVR.exe` — portable
- `build/bin/HVR-amd64-installer.exe` — installer có giao diện wizard

---

## 6. Post-Build — Nén & Ký binary

### 6.1. Nén bằng UPX (giảm ~60% size)

```bash
upx --best --lzma --ultra-brute build/bin/HVR.exe
```

| Flag | Tác dụng |
|---|---|
| `--best` | Mức nén cao nhất (level 9) |
| `--lzma` | Thuật toán LZMA — nén chặt hơn zlib mặc định |
| `--ultra-brute` | Thử mọi phương án → chậm nhất nhưng nhỏ nhất |

⚠️ **Cảnh báo UPX:**
- Một số AV (Windows Defender, Kaspersky) flag file UPX-packed thành false-positive. Nếu gửi khách corp/bank → **không nên UPX**.
- Làm binary chậm start thêm ~50–100ms (do decompress runtime).

### 6.2. Code Signing (quan trọng với Windows SmartScreen)

Nếu không ký, Windows 10/11 sẽ hiện cảnh báo **"Windows protected your PC"** → khách sợ, không dám chạy.

```bash
# Với signtool (Windows SDK)
signtool sign /fd SHA256 /tr http://timestamp.digicert.com /td SHA256 \
  /f your-cert.pfx /p YOUR_PASSWORD \
  build/bin/HVR.exe
```

→ Cần mua code signing cert (~$200–400/năm từ Sectigo / DigiCert).

**Giải pháp rẻ:** self-signed cert chỉ dùng nội bộ — khách vẫn phải "Run anyway".

---

## 7. Combo tối ưu nhất — 1 lệnh

**Lệnh tối ưu nhất cho đa số trường hợp** (cân bằng size / bảo mật / tương thích AV):

```bash
# Build
wails build -platform windows/amd64 \
  -ldflags "-s -w -H windowsgui -X main.version=1.0.0" \
  -trimpath -clean

# Nén (chỉ khi không sợ AV false-positive)
upx --best --lzma build/bin/HVR.exe
```

Kết quả: **~10–15 MB single exe**, không có debug info, không lộ path, ẩn console, khó RE ở mức cơ bản.

Nếu muốn **paranoid tối đa**: thêm garble (phương án B) + code signing.

---

## 8. Kích thước binary dự kiến

| Cấp độ | Lệnh | Size | Thời gian build |
|---|---|---|---|
| Debug | `wails build -debug` | 80–120 MB | ~30s |
| Default release | `wails build` | 40–60 MB | ~40s |
| Optimized | `-ldflags "-s -w" -trimpath` | 18–25 MB | ~45s |
| Optimized + UPX `--best` | + UPX | 8–12 MB | ~60s |
| Optimized + UPX `--ultra-brute` | + UPX brute | 7–10 MB | ~3–5 phút |
| Garble + Optimized + UPX | full stack | 8–13 MB | ~5–7 phút |

---

## 9. Checklist trước khi gửi khách

- [ ] Đã chạy `npm ci` trong `frontend/` để clean install
- [ ] Đã `wails build ... -clean` để dọn cache
- [ ] Binary < 30 MB
- [ ] Test trên máy **sạch** không có Go / Node / .NET
- [ ] Test trên Windows 10 + Windows 11
- [ ] Icon hiện đúng (`build/windows/icon.ico`)
- [ ] Version info trong `build/windows/info.json` đã update
- [ ] Scan VirusTotal trước để kiểm false-positive rate
- [ ] Nếu có → code signing
- [ ] Test double-click `.exe` → mở app bình thường, không console đen
- [ ] Test drag `.exe` qua máy khác → chạy được

---

## 10. Troubleshooting

| Lỗi | Nguyên nhân | Fix |
|---|---|---|
| Console đen chớp khi mở app | Thiếu `-H windowsgui` trong ldflags | Thêm `-ldflags "-H windowsgui"` |
| Size binary quá lớn (>40MB) | Thiếu `-s -w` | Thêm `-ldflags "-s -w"` |
| Windows Defender quarantine | UPX-packed + unsigned | Code signing, hoặc bỏ UPX |
| Garble báo lỗi `cannot find package` | Dep chưa fetch | Chạy `go mod download` trước |
| `wails build` báo Node error | Node version < 18 | Upgrade Node |
| Icon không hiện | `build/windows/icon.ico` missing | Regenerate bằng `wails generate icons` |

---

## 11. Reference

- Wails build docs: https://wails.io/docs/reference/cli/#build
- Garble: https://github.com/burrowers/garble
- UPX: https://upx.github.io/
- Go linker flags: https://pkg.go.dev/cmd/link
