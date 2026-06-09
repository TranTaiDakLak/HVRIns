// package s555v2 — Samsung S23 + FB API 555 Facebook account verification.
// Khác s558: doc_id/bloks_ver mới (559), styles_id mới, is_push_on=false.
// KHÔNG đụng đến verify/s558 hay verify/s557 — đây là platform riêng biệt.
package s555v2

import (
	"context"

	"HVRIns/internal/instagram"
)

// Verifier implements instagram.Verifier for S555V2 platform.
type Verifier struct{}

func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	return verifyAccount(ctx, session, cfg, outputPath, onStatus)
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformS555V2, func() instagram.Verifier {
		return &Verifier{}
	})
	instagram.RegisterPlatformVerifyUA(instagram.PlatformS555V2, RandomUA)
}
