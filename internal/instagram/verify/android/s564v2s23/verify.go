// package s564v2s23 — Samsung Galaxy S23+ (SM-S911B) + FB API 563 account verification.
// Clone từ verify/s563v2 — cùng doc_id/bloks_ver build 563.0.0.0.26, FBBV/972941018.
// Khác s563v2: device trong UA builder = SM-S911B (density 3.0, 1080x2400).
// KHÔNG đụng đến verify/s563, verify/s563v2 hay verify/s562v3 — đây là platform riêng biệt.
package s564v2s23

import (
	"context"

	"HVRIns/internal/instagram"
)

// Verifier implements instagram.Verifier for S564V1S21 platform.
type Verifier struct{}

func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	return verifyAccount(ctx, session, cfg, outputPath, onStatus)
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformS564V2S23, func() instagram.Verifier {
		return &Verifier{}
	})
	instagram.RegisterPlatformVerifyUA(instagram.PlatformS564V2S23, RandomUA)
}
