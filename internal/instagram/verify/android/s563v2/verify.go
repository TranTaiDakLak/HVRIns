// package s563v2 — Samsung S23 + FB API 563v2 Facebook account verification.
// Clone from s562v3 with new doc_id/bloks_ver, FBAV/563.0.0.0.26 + FBBV/972941018.
// KHÔNG đụng đến verify/s563 hay verify/s562v3 — đây là platform riêng biệt.
package s563v2

import (
	"context"

	"HVRIns/internal/instagram"
)

// Verifier implements instagram.Verifier for S563V2 platform.
type Verifier struct{}

func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	return verifyAccount(ctx, session, cfg, outputPath, onStatus)
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformS563V2, func() instagram.Verifier {
		return &Verifier{}
	})
	instagram.RegisterPlatformVerifyUA(instagram.PlatformS563V2, RandomUA)
}
