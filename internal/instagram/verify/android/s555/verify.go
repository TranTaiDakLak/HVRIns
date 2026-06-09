// Package s555 — Samsung S23 + FB API 555 Facebook account verification.
// Sử dụng verifyDocID và bloksVer mới (555), headers và nt_context khớp 555 traffic.
// KHÔNG đụng đến verify/s23 — đây là platform riêng biệt.
package s555

import (
	"context"

	"HVRIns/internal/instagram"
)

// Verifier implements instagram.Verifier for S555 platform.
type Verifier struct{}

func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	return verifyAccount(ctx, session, cfg, outputPath, onStatus)
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformS555, func() instagram.Verifier {
		return &Verifier{}
	})
	instagram.RegisterPlatformVerifyUA(instagram.PlatformS555, RandomUA)
}
