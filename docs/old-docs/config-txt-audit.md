# Audit: Tất cả file .txt được đọc từ Config/

> **Mục đích:** Liệt kê từng file `.txt` mà code thực sự đọc, dùng để làm gì,
> và so sánh với những gì phải có trong thư mục `Config/` để app chạy đúng.
>
> **Cập nhật:** 2026-05-21 — sau khi xóa toàn bộ thư mục `internal/facebook/fakeinfo/data/`
> và bỏ mọi embedded fallback. Tất cả dữ liệu hiện tại phải do **user tự cung cấp** trong `Config/`.

---

## Tóm tắt nhanh

| Thư mục | File code cần | Tự tạo khi khởi động | Ghi chú |
|---|:---:|:---:|---|
| `Config/Cookie/` | 2 | ✅ seed từ cookie/embedded/ | User paste cookie thủ công |
| `Config/Proxy/` | 2 (+1 legacy) | ✅ placeholder rỗng | User tự điền proxy |
| `Config/UserAgent/` | 4 | ❌ | **Phải copy thủ công** — pool rỗng → reg lỗi |
| `Config/DeviceInfo/` | 10 (+optional) | ❌ | **Phải copy thủ công** — xem chi tiết |
| `Config/Fbapp/` | 1 (+per-platform) | ❌ | **Phải copy thủ công** — có hardcode fallback |
| `Config/Namereg/` | 4 | ❌ | **Phải copy thủ công** — tên rỗng nếu thiếu |
| `Config/Locales/` | 1 | ❌ | Fallback `"en_US"` nếu thiếu |
| `Config/SimNetwork/` | 1 | ❌ | Fallback T-Mobile nếu thiếu |
| `Config/phone_database/` | N file (mỗi quốc gia 1 file) | ❌ | **Phải copy thủ công** — 237 file đã có sẵn trong `build/bin/Config` |
| `Config/Permanent/` | 2 | ✅ placeholder rỗng | Tích lũy tự động sau reg |
| `Config/TempMail/` | 0 | ✅ placeholder rỗng | File placeholder, code không đọc |
| `Config/` (gốc) | 1 | Tự tạo sau lần fetch đầu | Cache domain mail-temp.com |

---

## Chi tiết từng thư mục

### 1. `Config/Cookie/`

| File | Đọc bởi | Thời điểm | Định dạng | Ghi chú |
|---|---|---|---|---|
| `cookie_initial.txt` | `internal/cookie/store.go → LoadInitial()` | Khởi động (lazy) | Mỗi dòng: 1 cookie string chứa `datr=xxx` | User paste cookie vào đây |
| `datr_pool.txt` | `internal/cookie/store.go → LoadPool()` | Khởi động + mỗi lần append | Mỗi dòng: 1 giá trị datr | Tích lũy tự động sau mỗi lần đăng ký thành công |

> Các file `Pool*.txt`, `Pool20260518_*.txt`... là **output do app tạo ra**, không phải input.
> Code không đọc các file đó làm nguồn dữ liệu.
>
> Cookie vẫn được seed từ `internal/cookie/embedded/` (thư mục riêng, **không bị ảnh hưởng**
> bởi việc xóa `fakeinfo/data/`).

---

### 2. `Config/Proxy/`

| File | Đọc bởi | Thời điểm | Định dạng | Ghi chú |
|---|---|---|---|---|
| `proxy_tempmail.txt` | `internal/email/proxypool.go → LoadTempMailProxies()` | Khởi động / theo yêu cầu | `host:port:user:pass` hoặc `http://user:pass@host:port` | Proxy dùng khi xác minh qua temp mail |
| `proxy_rentmail.txt` | `internal/email/proxypool.go → LoadRentMailProxies()` | Khởi động / theo yêu cầu | Như trên | Proxy dùng cho các nhà cung cấp mail thuê |
| `proxy_gmail.txt` | `internal/email/proxypool.go` | Khởi động (tự đổi tên) | Như trên | **Legacy** — code tự đổi sang `proxy_rentmail.txt`. Có thể xóa file này sau |

> Placeholder rỗng cho `proxy_tempmail.txt` và `proxy_rentmail.txt` được tạo tự động
> nếu chưa tồn tại. Nếu rỗng, app trả về danh sách proxy rỗng (không lỗi).

---

### 3. `Config/UserAgent/`

**⚠️ Không còn embed fallback.** Nếu file thiếu → pool rỗng → UA rỗng → đăng ký thất bại.

| File | Đọc bởi | Thời điểm | Định dạng | Dùng khi nào |
|---|---|---|---|---|
| `Android_UG.txt` | `fakeinfo/ua_pools.go → loadUAPool()` | Khởi động + `ReloadUAPools()` | Mỗi dòng: 1 chuỗi UA đầy đủ | `BuildUA=false`, platform android / s23 / s555... |
| `iOS_UG.txt` | Như trên | Như trên | Như trên | Platform iOS HTTP |
| `PC_UG.txt` | Như trên | Như trên | Như trên | Platform mfb / request (label "PC") |
| `WebChrome_UA.txt` | Như trên | Như trên | Như trên | Platform webandroid (Chrome Mobile) |

---

### 4. `Config/DeviceInfo/`

Có **2 module** đọc cùng thư mục này: `fakeinfo/device.go` (load một lần khi khởi động) và
`uabuilder/data_loader.go` (load theo yêu cầu, cache theo mtime).

#### File bắt buộc phải có

| File | Đọc bởi | Định dạng | Fallback nếu thiếu |
|---|---|---|---|
| `devices.txt` | `fakeinfo/device.go` + `uabuilder` | `Nhà sản xuất:Thương hiệu:Model` hoặc 7 trường có thêm kích thước màn hình | Cứng: samsung SM-S911B |
| `carriers.txt` | `fakeinfo/device.go` + `uabuilder` | Mỗi dòng: tên nhà mạng (T-Mobile, Viettel...) | Cứng: `"T-Mobile"` |
| `buildnums.txt` | `fakeinfo/device.go` | Mỗi dòng: mã build Android (SKQ1.210908.001) | Fallback: `Thương hiệu-Model` |
| `devices_versions.txt` | `uabuilder` **chỉ** | Mỗi dòng: chuỗi phiên bản ("9", "10"...) | **Lỗi nếu thiếu** (uabuilder) |
| `chrome_versions.txt` | `uabuilder` **chỉ** | Mỗi dòng: phiên bản Chrome (146.0.7680) | **Lỗi nếu thiếu** (uabuilder) |
| `googleapp_versions.txt` | `uabuilder` **chỉ** | Mỗi dòng: phiên bản Google Play Services | **Lỗi nếu thiếu** (uabuilder) |

#### File có fallback trong code — thiếu vẫn chạy được

| File | Đọc bởi | Định dạng | Fallback nếu thiếu |
|---|---|---|---|
| `os_versions.txt` | `fakeinfo/device.go` **chỉ** | Mỗi dòng: chuỗi phiên bản | Cứng: `["9","10","11","12","13"]` |
| `densitis.txt` | `fakeinfo/device.go` + `uabuilder` | Mỗi dòng: số thực (3.0, 3.5...) | Cứng: `["2.0","2.5","3.0","3.5","4.0"]` |
| `screen_resolution.txt` | `fakeinfo/device.go` + `uabuilder` | Mỗi dòng: `RỘNGxCAO` (1080x2340) | Cứng: 3 độ phân giải mặc định |
| `device_cores.txt` | `fakeinfo/device.go` + `uabuilder` | Mỗi dòng: kiến trúc CPU (arm64-v8a) | Cứng: `["armeabi-v7a","arm64-v8a"]` |

#### File tuỳ chọn

| File | Đọc bởi | Ghi chú |
|---|---|---|
| `{platform}_devices.txt` | `uabuilder → LoadDevicesForPlatform()` | Danh sách thiết bị riêng cho từng platform (s23, s561...). Không có → tự dùng `devices.txt` |

#### ⚠️ Lưu ý quan trọng: hai file trùng khái niệm

`fakeinfo/device.go` đọc `os_versions.txt` và `uabuilder` đọc `devices_versions.txt` — cả hai đều
chứa danh sách phiên bản Android nhưng dùng **tên file khác nhau**. Cần đồng bộ hoặc hợp nhất sau.

#### ❌ File thừa — code không đọc

| File | Ghi chú |
|---|---|
| `locales.txt` | Code đọc `Config/Locales/locales.txt`, không đọc file này. Có thể xóa |
| `device_build_nums.txt` | Không có code nào tham chiếu. Có thể xóa |

---

### 5. `Config/Fbapp/`

**⚠️ Không còn embed fallback.** Nếu thiếu → `RandomFbVersion()` trả về giá trị hardcode
`"554.0.0.57.70 / 918990560"` — đăng ký vẫn chạy nhưng UA có thể bị cũ.

| File | Đọc bởi | Thời điểm | Định dạng | Ghi chú |
|---|---|---|---|---|
| `versions_and_builds.txt` | `fbdata/store.go → Reload()` + `uabuilder` | Khởi động + `Reload()` | Mỗi dòng: `phiên_bản\|build` (554.0.0.57.70\|918990560) | File chính, phải có |
| `versions_and_builds_{platform}.txt` | `uabuilder → LoadAppVersionsForPlatform()` | Theo yêu cầu | Như trên | Tuỳ chọn. Ví dụ: `_s23.txt`, `_s561.txt`. Không có → dùng file chính |

---

### 6. `Config/Namereg/`

**⚠️ Không còn embed fallback.** Nếu thiếu → danh sách tên rỗng → tên đăng ký rỗng → lỗi.

| File | Đọc bởi | Thời điểm | Định dạng |
|---|---|---|---|
| `US/firstname.txt` | `fakeinfo/overrides.go → ReloadOverrides()` | Khởi động | Mỗi dòng: 1 tên |
| `US/lastname.txt` | Như trên | Như trên | Như trên |
| `VN/firstname.txt` | Như trên | Như trên | Như trên |
| `VN/lastname.txt` | Như trên | Như trên | Như trên |

---

### 7. `Config/Locales/`

| File | Đọc bởi | Thời điểm | Định dạng | Fallback nếu thiếu |
|---|---|---|---|---|
| `locales.txt` | `fakeinfo/overrides.go → ReloadOverrides()` | Khởi động | Mỗi dòng: mã locale (en_US, vi_VN) | `"en_US"` |

---

### 8. `Config/SimNetwork/`

| File | Đọc bởi | Thời điểm | Định dạng | Fallback nếu thiếu |
|---|---|---|---|---|
| `simnetworks.txt` | `fakeinfo/overrides.go → loadSimNetworkOverride()` | Khởi động | `MCC\|MNC\|TênMạng\|MãQuốcGia` | T-Mobile US cứng |

> **`carriers.txt`** trong thư mục này: code **không đọc**. Có thể xóa hoặc giữ làm tài liệu tham khảo.

---

### 9. `Config/phone_database/`

#### Thư mục tra cứu mã điện thoại quốc gia

| Nguồn | Đọc bởi | Thời điểm | Định dạng | Fallback nếu thiếu |
|---|---|---|---|---|
| `Config/phone_database/*.txt` | `fakeinfo/phonecode.go → LoadPhoneDatabase()` | Khởi động (gọi từ `app.go`) | Tên file: `{QuốcGia}={CC}.{locale}.txt`; mỗi dòng trong file: `+<mã><xxxxxx>` | Danh sách quốc gia rỗng → `PhoneCodeFor()` trả `""` |

> `phone_codes.txt` (file CSV cũ) đã bị xóa.
> Dữ liệu nay nằm trong thư mục `Config/phone_database/`, mỗi file là 1 quốc gia.
> Mã điện thoại (`PhoneCode`) = common prefix của tất cả patterns trong file.

#### File pattern số điện thoại theo quốc gia

| File | Đọc bởi | Thời điểm | Định dạng | Fallback nếu thiếu |
|---|---|---|---|---|
| `{TênBất Kỳ}={MãISO}.{locale}.txt` | `fakeinfo/phonedatabase.go → PhoneFromDatabase()` | Theo yêu cầu khi đăng ký | Mỗi dòng: pattern số với `x`/`X` là chữ số bất kỳ. Ví dụ: `+84XXXXXXXXX` | Trả về `("","")` → không thể đăng ký số điện thoại |

> **Cách đặt tên file:** phần sau dấu `=` cuối quyết định quốc gia match.
> Ví dụ: `Vietnam US-GB=VN.vi_VN.txt` → match quốc gia `VN`, locale `vi_VN`.
>
> App tự ghi `_missing_country_codes.txt` mỗi khi gặp mã quốc gia không có file pattern
> → dùng file đó để biết cần bổ sung quốc gia nào.

---

### 10. `Config/Permanent/`

| File | Đọc bởi | Thời điểm | Định dạng | Ghi chú |
|---|---|---|---|---|
| `phone.txt` | `fakeinfo/builder.go → RandomLineFromFile()` + `app.go → GetPermanentFileCounts()` | Theo yêu cầu + hiển thị UI | Mỗi dòng: 1 số điện thoại | Tích lũy tự động sau mỗi lần đăng ký thành công |
| `mail.txt` | Như trên | Như trên | Mỗi dòng: 1 địa chỉ email | Như trên |

> Placeholder rỗng được tạo tự động nếu chưa tồn tại.

---

### 11. `Config/TempMail/`

| File | Đọc bởi | Ghi chú |
|---|---|---|
| `domains.txt` | **Không có code nào đọc** | Chỉ là placeholder. Code dùng nhà cung cấp mặc định hoặc fetch web trực tiếp |

---

### 12. `Config/` (gốc)

| File | Đọc bởi | Thời điểm | Định dạng | Ghi chú |
|---|---|---|---|---|
| `mailtempcom_domains.txt` | `internal/email/temp → loadDomainFile()` | Theo yêu cầu (cache 48h theo mtime) | Mỗi dòng: 1 tên miền (tmpbox.net) | Tự tạo sau lần fetch web đầu tiên. Không cần tạo thủ công |

---

## Bảng tổng hợp

> Cột **"Tự tạo"**: app tự tạo file / thư mục khi chưa tồn tại.
> Cột **"Fallback"**: có giá trị dự phòng trong code nếu file thiếu.

| Đường dẫn (tương đối từ Config/) | Code đọc | Tự tạo | Fallback trong code | Trạng thái |
|---|:---:|:---:|:---:|---|
| `Cookie/cookie_initial.txt` | ✅ | ✅ | — | User paste cookie vào |
| `Cookie/datr_pool.txt` | ✅ | ✅ | — | Tích lũy tự động |
| `Proxy/proxy_tempmail.txt` | ✅ | ✅ placeholder | ✅ list rỗng | User điền proxy |
| `Proxy/proxy_rentmail.txt` | ✅ | ✅ placeholder | ✅ list rỗng | User điền proxy |
| `Proxy/proxy_gmail.txt` | ⚠️ legacy | ❌ | — | Tự chuyển sang `proxy_rentmail.txt` — có thể xóa |
| `UserAgent/Android_UG.txt` | ✅ | ❌ | ❌ | **Phải copy thủ công** — pool rỗng → lỗi |
| `UserAgent/iOS_UG.txt` | ✅ | ❌ | ❌ | Như trên |
| `UserAgent/PC_UG.txt` | ✅ | ❌ | ❌ | Như trên |
| `UserAgent/WebChrome_UA.txt` | ✅ | ❌ | ❌ | Như trên |
| `DeviceInfo/devices.txt` | ✅ | ❌ | ✅ cứng | **Nên có** — fallback quá đơn giản |
| `DeviceInfo/carriers.txt` | ✅ | ❌ | ✅ cứng | **Nên có** — fallback chỉ "T-Mobile" |
| `DeviceInfo/buildnums.txt` | ✅ | ❌ | ✅ cứng | **Nên có** |
| `DeviceInfo/os_versions.txt` | ✅ (device.go) | ❌ | ✅ `["9"…"13"]` | Ổn nếu thiếu |
| `DeviceInfo/densitis.txt` | ✅ | ❌ | ✅ cứng | Ổn nếu thiếu |
| `DeviceInfo/screen_resolution.txt` | ✅ | ❌ | ✅ cứng | Ổn nếu thiếu |
| `DeviceInfo/device_cores.txt` | ✅ | ❌ | ✅ cứng | Ổn nếu thiếu |
| `DeviceInfo/devices_versions.txt` | ✅ (uabuilder) | ❌ | ❌ | **Phải có** — uabuilder lỗi nếu thiếu |
| `DeviceInfo/chrome_versions.txt` | ✅ (uabuilder) | ❌ | ❌ | **Phải có** — uabuilder lỗi nếu thiếu |
| `DeviceInfo/googleapp_versions.txt` | ✅ (uabuilder) | ❌ | ❌ | **Phải có** — uabuilder lỗi nếu thiếu |
| `DeviceInfo/{platform}_devices.txt` | ✅ tuỳ chọn | ❌ | ✅ về `devices.txt` | Tuỳ chọn — chỉ cần nếu muốn thiết bị riêng |
| `DeviceInfo/locales.txt` | ❌ | ❌ | — | **Thừa** — code đọc `Locales/locales.txt` |
| `DeviceInfo/device_build_nums.txt` | ❌ | ❌ | — | **Thừa** — không có code nào đọc |
| `Fbapp/versions_and_builds.txt` | ✅ | ❌ | ✅ hardcode cũ | **Phải có** — fallback quá cũ |
| `Fbapp/versions_and_builds_{platform}.txt` | ✅ tuỳ chọn | ❌ | ✅ về file chính | Tuỳ chọn |
| `Namereg/US/firstname.txt` | ✅ | ❌ | ❌ | **Phải có** — tên rỗng → lỗi |
| `Namereg/US/lastname.txt` | ✅ | ❌ | ❌ | Như trên |
| `Namereg/VN/firstname.txt` | ✅ | ❌ | ❌ | Như trên |
| `Namereg/VN/lastname.txt` | ✅ | ❌ | ❌ | Như trên |
| `Locales/locales.txt` | ✅ | ❌ | ✅ `"en_US"` | Ổn nếu thiếu |
| `SimNetwork/simnetworks.txt` | ✅ | ❌ | ✅ T-Mobile US | Ổn nếu thiếu |
| `SimNetwork/carriers.txt` | ❌ | ❌ | — | **Thừa** — không có code nào đọc |
| `phone_database/{quốc gia}=CC.locale.txt` | ✅ | ❌ | ❌ | **Phải copy thủ công** — 237 file, đã có sẵn trong `build/bin/Config` |
| `Permanent/phone.txt` | ✅ | ✅ placeholder | — | Tích lũy tự động |
| `Permanent/mail.txt` | ✅ | ✅ placeholder | — | Tích lũy tự động |
| `TempMail/domains.txt` | ❌ | ✅ placeholder | — | **Thừa** — placeholder không được đọc |
| `mailtempcom_domains.txt` (gốc) | ✅ | Tự tạo sau fetch | ✅ fetch web | Không cần tạo thủ công |

---

## Những việc cần làm

### Bước 1 — File phải copy ngay (app không chạy đúng nếu thiếu)

```
Config/UserAgent/
    Android_UG.txt          ← copy từ fakeinfo/data/ua_android_pool.txt (cũ)
    iOS_UG.txt              ← copy từ fakeinfo/data/ua_ios_pool.txt (cũ)
    PC_UG.txt               ← copy từ fakeinfo/data/ua_request_pool.txt (cũ)
    WebChrome_UA.txt        ← copy từ fakeinfo/data/ua_webchrome_pool.txt (cũ)

Config/Namereg/
    US/firstname.txt        ← copy từ fakeinfo/data/firstnames.txt (cũ)
    US/lastname.txt         ← copy từ fakeinfo/data/lastnames.txt (cũ)
    VN/firstname.txt        ← copy từ fakeinfo/data/vn_firstnames.txt (cũ)
    VN/lastname.txt         ← copy từ fakeinfo/data/vn_lastnames.txt (cũ)

Config/Fbapp/
    versions_and_builds.txt ← copy từ fakeinfo/data/versions_and_builds.txt (cũ)

Config/phone_database/
    {QuốcGia}={CC}.{locale}.txt (237 file) ← đã có sẵn trong build/bin/Config/phone_database/
                                              Mỗi file chứa patterns số điện thoại dạng +84XXXXXXX

Config/DeviceInfo/
    devices.txt             ← phải có từ bộ dữ liệu thiết bị
    carriers.txt            ← phải có từ bộ dữ liệu nhà mạng
    buildnums.txt           ← phải có từ bộ dữ liệu build
    devices_versions.txt    ← phải có (uabuilder lỗi nếu thiếu)
    chrome_versions.txt     ← phải có (uabuilder lỗi nếu thiếu)
    googleapp_versions.txt  ← phải có (uabuilder lỗi nếu thiếu)
```

### Bước 2 — Cân nhắc hợp nhất 2 file trùng khái niệm

`DeviceInfo/os_versions.txt` (`fakeinfo/device.go`) và `DeviceInfo/devices_versions.txt` (`uabuilder`) đều chứa danh sách phiên bản Android nhưng dùng tên file khác nhau. Nên đồng nhất về 1 tên để tránh nhầm lẫn khi cập nhật dữ liệu.

### Bước 3 — Xóa file thừa trong Config/ (tuỳ chọn)

```
Config/DeviceInfo/locales.txt       → xóa (nhầm với Config/Locales/locales.txt)
Config/DeviceInfo/device_build_nums.txt → xóa (không có code đọc)
Config/SimNetwork/carriers.txt      → xóa (không có code đọc)
Config/TempMail/domains.txt         → xóa hoặc giữ placeholder
```
