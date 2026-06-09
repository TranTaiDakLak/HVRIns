// Package s565 â€” Samsung S21+ + FB API 565 Facebook account verification.
// KhÃ¡c s558: doc_id/bloks_ver má»›i (559), styles_id má»›i, is_push_on=false.
// KHÃ”NG Ä‘á»¥ng Ä‘áº¿n verify/s558 hay verify/s557 â€” Ä‘Ã¢y lÃ  platform riÃªng biá»‡t.
package s565v2s23

import (
	"context"

	"HVRIns/internal/instagram"
)

// Verifier implements instagram.Verifier for S565 platform.
type Verifier struct{}

func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	return verifyAccount(ctx, session, cfg, outputPath, onStatus)
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformS565V2S23, func() instagram.Verifier {
		return &Verifier{}
	})
	instagram.RegisterPlatformVerifyUA(instagram.PlatformS565V2S23, RandomUA)
}
