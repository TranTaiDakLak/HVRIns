# Cơ chế Check Live/Die Instagram HIỆN TẠI trong dự án WeBM

> Tài liệu này mô tả **đúng cách dự án đang làm** (theo code hiện tại), không phải bản đề xuất.

---

## Tổng quan

Cơ chế: dùng **cookie của 1 (hoặc nhiều) tài khoản "checker"** (do người dùng dán tay) để mở **trang profile** của từng username cần check, rồi tìm chuỗi `"profile_id"` trong HTML trả về:
- Có `profile_id` → **LIVE**
- Không có → **DIE**

Chạy **đa luồng** (tối đa 20) cho danh sách nick đã chọn.

---

## Luồng chạy (step-by-step)

### B1. Trigger — nút check
`btnCheckLiveDieInstagram_Click` — [UI/Facebook/frmFacebook.Core.cs:3963](UI/Facebook/frmFacebook.Core.cs#L3963)
- Lấy các dòng đã tick chọn (`cChose == true`).
- Nếu chưa chọn nick → cảnh báo, dừng.

### B2. Nhập cookie checker
- Mở form `frmCheckInfowithCookie` (ShowDialog).
- Người dùng **dán danh sách cookie** (mỗi dòng 1 cookie) + tùy chọn `isCheckPost`.
- Cookie được tách theo dòng (`\n`, `\r`) → `lstCookie`.

### B3. Chạy đa luồng
- `maxConcurrentTasks = 20` (hoặc bằng số cookie nếu < 20).
- Dùng `ConcurrentQueue` các dòng nick; mỗi task dequeue 1 nick để xử lý.
- **Gán cookie theo chỉ số task:** `sessionCookie = lstCookie[taskIndex % cookieCount]`
  → tức **dùng cookie checker đã dán, KHÔNG dùng cookie riêng của từng nick**.

### B4. Check từng nick
Gọi `CheckLiveDieInstagramwithCookie(sessionCookie, accountRow, isCheckPost)` — [Platforms/WemakeSocial.cs:1992](Platforms/WemakeSocial.cs#L1992):
- `account = accountRow.Cells["cUID"]` (username).
- Nếu thiếu UID → `cRun = "Thiếu UID!"`, bỏ qua.
- Gọi tiếp `_ws.CheckLiveDieInstagramwithCookie(account, cookie)` (xem B5).
- Lấy `resp = httpMiniResponse`:
  - Nếu `!resp.IsSuccess` hoặc `Content` rỗng → `cRun = "Không thể lấy profile_id!"`, trả `false`.
- Parse: `profileId = Regex.Match(profileHtml, "\"profile_id\":\"(\\d+)\"").Groups[1].Value`
  - `profileId` khác rỗng → `cRun = "IG LIVE"`, trả **true**.
  - rỗng → `cRun = "IG DIE"`, trả **false**.

### B5. Gọi HTTP tới Instagram
`CheckLiveDieInstagramwithCookie(string accountId, string cookie)` — [WemakeMetaKit/Instagram/Login/Login.Cookie.BM.cs:57](../../WemakeMetaKit/WemakeMetaKit/Instagram/Login/Login.Cookie.BM.cs)

- Tạo `HttpClientRequest(f.Proxy, UA)` với **UA Chrome**:
  `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36`
- **Request:** `GET https://www.instagram.com/{accountId}/`
- **Headers:**
  | Header | Giá trị |
  |---|---|
  | accept | `text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,...` |
  | accept-language | `vi-VN,vi;q=0.9,en-US;q=0.8,en;q=0.7` |
  | cache-control | `max-age=0` |
  | **cookie** | (cookie checker được truyền vào) |
  | dpr | `1` |
  | priority | `u=0, i` |
  | sec-ch-prefers-color-scheme | `light` |
  | sec-ch-ua | `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"` |
  | sec-ch-ua-full-version-list | `...131.0.6778.205...` |
  | sec-ch-ua-mobile | `?0` |
  | sec-ch-ua-platform | `"Windows"` |
  | sec-ch-ua-platform-version | `"10.0.0"` |
  | sec-fetch-dest | `document` |
  | sec-fetch-mode | `navigate` |
  | sec-fetch-site | `none` |
  | sec-fetch-user | `?1` |
  | upgrade-insecure-requests | `1` |
  | viewport-width | `978` |
- Gọi `GetContentResponse(Method.GET, profileUrl, "", headers, init:false, autoaddReferInit:false)`.
- Trả về `Result` với `httpMiniResponse` = response thô (để bên gọi tự parse).

### B6. Ghi kết quả ra UI
Quay lại `btnCheckLiveDieInstagram_Click` — [frmFacebook.Core.cs:4037](UI/Facebook/frmFacebook.Core.cs#L4037):
- `isCheckSuccess == false` → `cRun = "Die!"`, đổi màu dòng, bỏ chọn.
- `isCheckSuccess == true` → `cRun = "Live !"`, đổi màu dòng, bỏ chọn.

---

## Sơ đồ rút gọn

```
[Chọn nick] 
   -> Form dán cookie checker (lstCookie)
   -> 20 luồng song song, mỗi nick gán cookie = lstCookie[taskIndex % cookieCount]
   -> GET https://www.instagram.com/{username}/   (kèm cookie checker)
   -> Regex tìm "profile_id":"<số>" trong HTML
        có  -> LIVE  (cRun = "Live !")
        không -> DIE (cRun = "Die!")
```

---

## Đặc điểm & giới hạn (của cơ chế hiện tại)

- **Dùng cookie checker dán tay**, gán theo `taskIndex % cookieCount` (cố định, không xoay khi cookie chết).
- **Check theo username** (mở profile target), không phải check session riêng của từng nick.
- **Phân loại bằng regex `"profile_id"`** trên HTML trang profile.
- Cần **IP/proxy sạch** + cookie checker còn sống; IP bẩn → trang không trả đúng → dễ báo nhầm.
- ⚠️ **Lưu ý kỹ thuật:** Instagram đã thay đổi cấu trúc HTML và **field `"profile_id"` không còn xuất hiện** trên trang profile như trước → regex này có thể **không match** dù nick còn sống → báo **DIE nhầm**. Đây là điểm cần kiểm tra/cập nhật nếu thấy kết quả sai.

---

## Vị trí code liên quan
| Thành phần | File |
|---|---|
| Nút trigger + đa luồng + ghi cRun | `UI/Facebook/frmFacebook.Core.cs` (`btnCheckLiveDieInstagram_Click`, ~3963) |
| Parse profile_id + LIVE/DIE | `Platforms/WemakeSocial.cs` (`CheckLiveDieInstagramwithCookie`, ~1992) |
| Gọi HTTP GET profile | `WemakeMetaKit/Instagram/Login/Login.Cookie.BM.cs` (`CheckLiveDieInstagramwithCookie`, ~57) |
| Form nhập cookie | `frmCheckInfowithCookie` |
