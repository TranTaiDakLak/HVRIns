# Test iOS Messenger Verification Flow với EAAG Token

## 📋 Mô tả

Test script để verify flow iOS Messenger sử dụng **EAAG (User Token)** thay vì APP Token.

### Flow Sequence:
1. **Login Request** (`send_login_request`) - Dùng EAAG token
2. **Bottomsheet** (`bloks.caa.reg.confirmation.fb.bottomsheet`)
3. **Change Email** (`bloks.caa.reg.confirmation.change.email`)
4. **Email Validation** (`bloks.caa.reg.async.contactpoint_email.async`)

## 🔑 Điểm khác biệt so với flow hiện tại

| Aspect | Flow hiện tại (APP Token) | Flow EAAG (User Token) |
|--------|---------------------------|------------------------|
| **Authorization** | `OAuth <APP_TOKEN>` | `OAuth <EAAG_TOKEN>` |
| **Token format** | Fixed app token | User-specific EAAG token (bắt đầu `EAAG...`) |
| **Body params** | Không có `spectra_guardian_token` | Có `spectra_guardian_token` (cũng là EAAG token) |
| **Use case** | Verify account từ reg session | Verify account có sẵn EAAG token |

## 📦 Chuẩn bị

### 1. Chuẩn bị test accounts

Tạo/sửa file `test_accounts_eaag.txt` với format:

```
UID|Password|Cookies|EAAGToken|DateTime|Country|Status
```

**Ví dụ:**
```
61590833603441|mypass123||EAAGOBNEq5twBRhtvwICBE4ZCQwfQhedNSZBtnvU...|2026-06-08|VN|Active
61590833603442|pass456||EAAGOBNEq5twBRxxxxxxxxxxxxxxxxxxxxxxxxxxxxx...|2026-06-08|US|Active
```

**Lưu ý:**
- Token **PHẢI** bắt đầu bằng `EAAG` (EAAG user token)
- Cookies có thể để trống: `||`
- DateTime format: `YYYY-MM-DD`
- Country: 2-letter code (`VN`, `US`, etc.)

### 2. Lấy EAAG Token

**Cách 1: Từ capture iOS Messenger**
- Capture traffic từ iOS Messenger app
- Tìm request có header `Authorization: OAuth EAAG...`
- Copy token EAAG

**Cách 2: Từ login response**
- Login vào iOS Messenger
- Token EAAG sẽ có trong response

## 🚀 Chạy test

```bash
# Build
go build -o test_eaag.exe test_verify_iosmess_eaag.go

# Run
./test_eaag.exe
```

Hoặc chạy trực tiếp:
```bash
go run test_verify_iosmess_eaag.go
```

## 📊 Output mẫu

```
=== TEST iOS Messenger Verification Flow với EAAG Token ===

📋 Đã load 2 test accounts

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🧪 Test Account 1/2 - UID: 61590833603441
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📤 Step 1: Login Request...
   ✅ Success - 234ms - Status:200 - Size:8192 bytes
📤 Step 2: Bottomsheet...
   ✅ Success - 156ms - Status:200 - Size:32984 bytes
📤 Step 3: Change Email...
   ✅ Success - 189ms - Status:200 - Size:21900 bytes
📤 Step 4: Email Validation...
   ❌ Failed - 201ms - Error: Email is invalid

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📊 SUMMARY REPORT
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Login               : 2/2 (100.0% success)
Bottomsheet         : 2/2 (100.0% success)
Change Email        : 2/2 (100.0% success)
Email Validation    : 0/2 (0.0% success)

OVERALL             : 6/8 (75.0% success)

🔍 Error Summary:
   • Email is invalid: 2 times
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## 🔍 Phân tích kết quả

### Success Indicators

**Login Request:**
- ✅ HTTP 200
- ✅ Response không chứa `"errors":[{`
- ✅ Response không chứa `checkpoint`

**Bottomsheet:**
- ✅ HTTP 200
- ✅ Response size > 30KB (typical)

**Change Email:**
- ✅ HTTP 200
- ✅ Response size > 20KB (typical)

**Email Validation:**
- ✅ HTTP 200
- ✅ Response chứa `caa_reg_confirmation` hoặc `caa_reg_contactpoint`
- ❌ Response chứa `email_already_used` → Email đã được dùng
- ❌ Response chứa `email_is_invalid` → Email không hợp lệ
- ❌ Response chứa `checkpoint` → Account bị checkpoint

## 📝 So sánh với capture thực tế

### Capture từ doc (VerMessByTokenEaag)

**Request [10538] - Login:**
- Token: `EAAGOBNEq5twBRhtvwICBE4...` (EAAG)
- Body có `spectra_guardian_token` (cũng là EAAG token)
- User ID: `61590833603441`

**Request [10604] - Bottomsheet:**
- Token: `EAAGOBNEq5twBRhtvwICBE4...` (EAAG)
- Response size: 32984 bytes

**Request [10606] - Change Email:**
- Token: `EAAGOBNEq5twBRhtvwICBE4...` (EAAG)
- Response size: 21900 bytes

**Request [10668] - Email Validation:**
- Token: `EAAGOBNEq5twBRhtvwICBE4...` (EAAG)
- Email: `yqssu@rover.info`
- Result: **FAILED** - Email invalid

### Kết luận từ capture
- Flow EAAG hoạt động cho 3 bước đầu (Login, Bottomsheet, Change Email)
- Bước Email Validation **thất bại** do email không hợp lệ
- Có thể do:
  - Email `yqssu@rover.info` không hợp lệ
  - Account đã có email khác
  - Facebook không chấp nhận email từ domain `rover.info`

## 🎯 Next Steps

1. **Test với email hợp lệ hơn:**
   - Sử dụng email thật (Gmail, Outlook, etc.)
   - Hoặc email disposable được Facebook chấp nhận

2. **Capture OTP flow:**
   - Nếu email validation thành công → capture OTP confirmation request
   - Implement OTP confirmation step

3. **So sánh với APP Token flow:**
   - Chạy song song cả 2 flows
   - So sánh success rate
   - Xác định flow nào ổn định hơn

## 🐛 Troubleshooting

### "Token không phải EAAG token"
- ✅ Check token có bắt đầu bằng `EAAG` không
- ✅ Copy đầy đủ token (thường dài ~200-300 ký tự)

### "Checkpoint triggered"
- Account bị checkpoint
- Cần verify qua browser trước
- Hoặc dùng account khác

### "Email is invalid"
- Thử email từ domain khác (@gmail.com, @outlook.com)
- Check email format (phải có @ và domain hợp lệ)

### HTTP 429 (Too Many Requests)
- Rate limit
- Tăng delay giữa các requests
- Giảm số lượng test accounts

## 📚 Tham khảo

- Flow capture: `e:\WEMAKE\DocWeMake\FlowRegFB_IOS\VerMessByTokenEaag\`
- Code hiện tại: `e:\WEMAKE\NullCoreSummer\internal\facebook\verify\ios\iosmess\`
