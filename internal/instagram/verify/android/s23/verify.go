// Package s23 — Samsung S23 Facebook account verification (add email + confirm OTP)
// Captured from APi_Verify_Android_S23.txt:
//   Step 1: POST graph.facebook.com/graphql — contactpoint_email.async (add email)
//   Step 2: POST graph.facebook.com/graphql — confirmation.async (confirm OTP code)
//
// Key differences from generic Android verify:
//   - Host: graph.facebook.com (not b-graph)
//   - Auth: User OAuth token (not app token)
//   - S23 headers: x-meta-usdid, x-fb-conn-uuid-client, content-encoding: gzip
//   - Same UA as S23 registration
package s23

import (
	"context"

	"HVRIns/internal/instagram"
)

// Verifier implements instagram.Verifier for S23 platform
type Verifier struct{}

// Verify performs S23 email verification: add email → wait OTP → confirm
func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	return verifyAccount(ctx, session, cfg, outputPath, onStatus)
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformS23, func() instagram.Verifier {
		return &Verifier{}
	})
}
