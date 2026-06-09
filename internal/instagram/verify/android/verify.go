// Package android — Facebook Android V3 platform Verifier.
//
// Flow chia sẻ ~95% với S23 verify vì cả 2 dùng chung:
//   - doc_id = ChangeAndConfirmContactpointMobiledocid_v3 = 1199408042526631289603660492
//   - bloks_versioning_id d90663...b999
//   - reg_info schema 180+ fields
//   - server_params.current_step = 10
//   - AddEmail/ConfirmCode/Resend/CheckLive flow
//
// Khác biệt nhỏ (mà FB hiếm strict check cho verify):
//   - UA: FB4A Dalvik (không phải Samsung S23 UA) — nhưng verify dùng OAuth token
//     nên UA không quyết định success/fail như reg.
//   - x-meta-zca: base64 blob (không phải "empty_token").
//   - Resend URL: b-graph (giống S23 hiện tại).
//
// Quyết định: delegate hoàn toàn sang S23 verifier để tránh duplicate 1000+ dòng code.
// Ưu tiên ship flow working; nếu cần phân biệt header chính xác hơn sau này thì tách.
package android

import (
	"context"

	"HVRIns/internal/instagram"
	s23verify "HVRIns/internal/instagram/verify/android/s23"
)

// Verifier implements instagram.Verifier for the Android V3 native app platform.
// Delegates to S23 verifier (share same doc_id + schema).
type Verifier struct {
	inner s23verify.Verifier
}

// Verify performs account email verification via Android V3 flow.
// Internal delegate to S23 verifier — verify logic giống hệt vì chung schema.
func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	if onStatus != nil {
		onStatus(session.UID, "[AndroidV3] Verify (delegate S23 flow — cùng doc_id + schema)")
	}
	return v.inner.Verify(ctx, session, cfg, outputPath, onStatus)
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformAndroid, func() instagram.Verifier {
		return &Verifier{}
	})
}
