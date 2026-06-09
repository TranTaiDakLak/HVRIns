// Package iosmess (verify) — Messenger Lite iOS VERIFY: add-mail + OTP confirm + live/die.
//
// Dùng state từ reg (PlatformIOSMessReg): session.SessionlessCryptedUID (crypted_user_id),
// session.DeviceID/FamilyDeviceID/Datr (fingerprint), session.Email/EmailMeta (reuse mail).
// Verify fire add-mail (trigger OTP) + screen-loads thủ công, rồi verifybase.RunVerify
// (reuse mail → WaitOTP → confirm → live/die). KHÔNG login (dùng app token).
//
// Tái dùng body builders + transport từ register/ios/iosmess (export.go).
package iosmess

import (
	"context"

	"HVRIns/internal/instagram"
	regmess "HVRIns/internal/instagram/register/ios/iosmess"
)

type Verifier struct{}

func (v *Verifier) Verify(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	return verifyAccount(ctx, session, cfg, outputPath, onStatus)
}

func init() {
	instagram.RegisterPlatformVerifier(instagram.PlatformIOSMess, func() instagram.Verifier {
		return &Verifier{}
	})
	// RandomMessUA: random MessengerLite UA per-account (đa dạng device/version/locale),
	// KHÔNG dùng RandomUA(country) deterministic (mọi country 2 ký tự → cùng seed → cùng UA).
	instagram.RegisterPlatformVerifyUA(instagram.PlatformIOSMess, func(countryCode string) string {
		_ = countryCode
		return regmess.RandomMessUA()
	})
}
