// otp_reader.go — Priority-based OTP reader cho Hotmail OAuth2 providers.
// Cho phép user chọn nguồn đọc OTP ưu tiên (DongVanFB hoặc Smail1s alias "UnlimitMail")
// — nếu primary fail thì fallback sang reader còn lại.
//
// Hiện app có 2 reader thực sự đọc Hotmail OAuth2 (cùng mailbox Microsoft qua
// refresh_token+client_id, kết quả tương đương):
//
//	OTPSourceDongVan ("dongvan")  → tools.dongvanfb.net/api/get_code_oauth2   ~2.6s
//	OTPSourceUnlimit ("unlimit")  → smail1s.com/get_messages?mode=oauth      ~4.5s
//
// Default: primary=dongvan, fallback=unlimit (giữ behavior cũ của store1s).
package rent

import (
	"context"
	"fmt"
	"net/http"
)

// OTP source identifiers — dùng trong email.Options.OTPHotmailPriority.
const (
	OTPSourceDongVan = "dongvan" // tools.dongvanfb.net/api/get_code_oauth2
	OTPSourceUnlimit = "unlimit" // smail1s.com/get_messages?mode=oauth (UnlimitMail backend)
)

// otpReaderFn — chuẩn hoá signature cho mọi OTP reader.
type otpReaderFn func(ctx context.Context, email, pass, refreshToken, clientID string, client *http.Client) (string, error)

// ReadOTPWithPriority đọc OTP của mailbox Hotmail OAuth2 với ưu tiên user-defined.
//
// Flow:
//  1. Gọi primary reader theo priority (default dongvan nếu priority rỗng/không hợp lệ)
//  2. Primary err==nil → return luôn (code có thể "" nếu inbox rỗng — caller poll tiếp)
//  3. Primary lỗi mạng/parse → fallback sang reader còn lại
//  4. Cả 2 fail → trả lỗi của primary (giữ thông tin lỗi đầu tiên)
//
// notify: optional callback log fallback events lên UI (truyền nil nếu không cần).
func ReadOTPWithPriority(
	ctx context.Context,
	priority string,
	email, pass, refreshToken, clientID string,
	client *http.Client,
	notify func(string),
) (string, error) {
	primary, primaryName, fallback, fallbackName := selectReaders(priority)

	code, err := primary(ctx, email, pass, refreshToken, clientID, client)
	if err == nil {
		return code, nil
	}

	// Primary lỗi → fallback. Notify để user biết primary fail.
	if notify != nil {
		notify(fmt.Sprintf("[OTP] %s fail (%v) → fallback %s", primaryName, err, fallbackName))
	}
	code2, err2 := fallback(ctx, email, pass, refreshToken, clientID, client)
	if err2 == nil {
		return code2, nil
	}

	// Cả 2 fail → trả lỗi primary kèm context fallback.
	return "", fmt.Errorf("OTP read both fail: %s=%w; %s=%v", primaryName, err, fallbackName, err2)
}

// selectReaders quyết định primary + fallback theo priority key.
// Default: dongvan primary, unlimit fallback.
func selectReaders(priority string) (otpReaderFn, string, otpReaderFn, string) {
	switch priority {
	case OTPSourceUnlimit:
		return readOTPViaSmail1s, "unlimit", readOTPViaDongvanfb, "dongvan"
	default: // OTPSourceDongVan hoặc rỗng/không hợp lệ
		return readOTPViaDongvanfb, "dongvan", readOTPViaSmail1s, "unlimit"
	}
}
