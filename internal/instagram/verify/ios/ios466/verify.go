// Package ios562 — Facebook iOS native (FBIOS) account verification, API v562.
//
// Thin wrapper: cung cấp verifybase.Spec cho iOS rồi delegate vào
// verifybase.RunVerify. Toàn bộ orchestration (add email → wait OTP → confirm →
// check live/die) nằm ở verifybase — check live/die dùng chung với Android.
//
// Phần iOS-specific: doc_id/bloks_ver iOS, header iOS (FBIOS), body builder
// theo Bloks CAA envelope của iOS.
package ios466

import (
	"context"

	"HVRIns/internal/instagram"
)

// Verifier implements instagram.Verifier cho platform iOS555.
type Verifier struct{}

// Verify thực hiện flow verify cho 1 account iOS555.
func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	return verifyAccount(ctx, session, cfg, outputPath, onStatus)
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformIOS466, func() instagram.Verifier {
		return &Verifier{}
	})
	instagram.RegisterPlatformVerifyUA(instagram.PlatformIOS466, RandomUA)

}
