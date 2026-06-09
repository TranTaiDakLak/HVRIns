// Package ios562 — Facebook iOS native (FBIOS) account verification, API v562.
//
// Thin wrapper: cung cấp verifybase.Spec cho iOS rồi delegate vào
// verifybase.RunVerify. Toàn bộ orchestration (add email → wait OTP → confirm →
// check live/die) nằm ở verifybase — check live/die dùng chung với Android.
//
// Phần iOS-specific: doc_id/bloks_ver iOS, header iOS (FBIOS), body builder
// theo Bloks CAA envelope của iOS.
package ios562

import (
	"context"

	"HVRIns/internal/instagram"
)

// Verifier implements instagram.Verifier cho platform iOS562.
type Verifier struct{}

// Verify thực hiện flow verify cho 1 account iOS562.
func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	return verifyAccount(ctx, session, cfg, outputPath, onStatus)
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformIOS562, func() instagram.Verifier {
		return &Verifier{}
	})
	instagram.RegisterPlatformVerifyUA(instagram.PlatformIOS562, RandomUA)

	// iOS563 verify dùng chung verifier iOS562 — cùng backend Bloks CAA iOS, cùng
	// doc_id/bloks_versioning_id v563. Token EAAAAAY bắt buộc (xử lý trong steps.go).
	instagram.RegisterPlatformVerifier(instagram.PlatformIOS563, func() instagram.Verifier {
		return &Verifier{}
	})
	instagram.RegisterPlatformVerifyUA(instagram.PlatformIOS563, RandomUA)
}
