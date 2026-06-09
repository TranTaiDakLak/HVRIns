// Package appmessv3 — Facebook Messenger (Orca) "App Mess V3" verifier.
//
// Luồng Ver mô phỏng capture FlowRegVerFb_AppMessV3:
//   - Login bằng Messenger Bloks send_login_request (facebook_local_auth + EAAAAU sẵn có)
//     → lấy session EAAD + session_cookies.
//   - Sau đó chạy luồng ver chuẩn: add email → chờ OTP → confirm → check live → (2FA/AddInfo).
//
// Mục đích: test luồng login V3 (Messenger) khác với REST /auth/login mà các ver Android
// hiện tại đang dùng.
package appmessv3

import (
	"context"

	"HVRIns/internal/instagram"
)

// Verifier implements instagram.Verifier for the AppMess V3 (Messenger) platform.
type Verifier struct{ platform string }

func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	plat := v.platform
	if plat == "" {
		plat = instagram.PlatformAppMessV3
	}
	return verifyAccount(ctx, plat, session, cfg, outputPath, onStatus)
}

// verifyUAFor trả UA func theo platform — version cố định (535/545) hoặc pool (530).
func verifyUAFor(platform string) func(string) string {
	vv := vverForPlatform(platform)
	if vv.fbav == "" {
		return RandomUA
	}
	return func(country string) string { return randomOrcaUAVer(country, vv.fbav, vv.fbbv) }
}

func init() {
	for _, plat := range []string{
		instagram.PlatformAppMessV3, instagram.PlatformAppMessV3_535, instagram.PlatformAppMessV3_545,
		instagram.PlatformAppMessV3_555, instagram.PlatformAppMessV3_563, instagram.PlatformAppMessV3_564,
		instagram.PlatformAppMessV3_565, instagram.PlatformAppMessV3_525, instagram.PlatformAppMessV3_515,
		instagram.PlatformAppMessV3_505, instagram.PlatformAppMessV3_490,
	} {
		p := plat
		instagram.RegisterPlatformVerifier(p, func() instagram.Verifier { return &Verifier{platform: p} })
		instagram.RegisterPlatformVerifyUA(p, verifyUAFor(p))
	}
}
