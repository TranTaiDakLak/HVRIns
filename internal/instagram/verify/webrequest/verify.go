// Package webrequest — Facebook WebRequest platform Verifier (skeleton)
// Mapping từ C#: VerifyAccountAPIAutomation (WebRequest variant using raw HTTP)
// TODO: Implement using raw HTTP request flow for email verification
package webrequest

import (
	"context"

	"HVRIns/internal/instagram"
)

// Verifier implements instagram.Verifier for the WebRequest platform.
type Verifier struct{}

// Verify performs account email verification via raw WebRequest flow.
// Currently returns not-implemented result; full implementation pending.
func (v *Verifier) Verify(_ context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	if onStatus != nil {
		onStatus(session.UID, "[webrequest] Verify: not yet implemented")
	}
	return &instagram.VerifyResult{Status: "error", Message: "webrequest verify: not implemented"}
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformWebRequest, func() instagram.Verifier {
		return &Verifier{}
	})
}
