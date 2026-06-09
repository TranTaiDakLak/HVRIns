// Package s558 — Samsung S23 + FB API 558 Facebook account verification.
// Khác s557: integrity-machine-id header, si_device_param_network_info trong body,
// doc_id/bloks_ver mới, bỏ appnet/tasos/session/privacy headers.
// KHÔNG đụng đến verify/s557 hay verify/s23 — đây là platform riêng biệt.
package s558

import (
	"context"

	"HVRIns/internal/instagram"
)

// Verifier implements instagram.Verifier for S558 platform.
type Verifier struct{}

func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	return verifyAccount(ctx, session, cfg, outputPath, onStatus)
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformS558, func() instagram.Verifier {
		return &Verifier{}
	})
	// Đăng ký UA factory để pickUAForVerifyPlatform sinh UA 558 đúng FBAV
	// (random device + carrier, không pick từ pool 554).
	instagram.RegisterPlatformVerifyUA(instagram.PlatformS558, RandomUA)
}
