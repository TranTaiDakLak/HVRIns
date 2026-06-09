// Package webandroid — Facebook Web+Android hybrid platform SecurityManager (skeleton)
// Mapping từ C#: FacebookSecurityFeatureAPIAndroid (WebAndroid variant)
// TODO: Implement 2FA/checkpoint/password via Web+Android hybrid endpoints
package webandroid

import (
	"context"

	"HVRIns/internal/instagram"
)

// SecurityManager implements instagram.SecurityManager for the Web+Android platform.
type SecurityManager struct{}

func (s *SecurityManager) Enable2FA(_ context.Context, session *instagram.Session) (*instagram.TwoFAResult, error) {
	return nil, nil // TODO: implement
}

func (s *SecurityManager) HandleCheckpoint(_ context.Context, session *instagram.Session) error {
	return nil // TODO: implement
}

func (s *SecurityManager) ChangePassword(_ context.Context, session *instagram.Session, newPassword string) error {
	return nil // TODO: implement
}

func init() {
	instagram.RegisterPlatformSecurityManager(instagram.PlatformWebAndroid, func() instagram.SecurityManager {
		return &SecurityManager{}
	})
}
