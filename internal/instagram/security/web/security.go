// Package web — Facebook Web platform SecurityManager (skeleton)
// Mapping từ C#: FacebookSecurityFeatureAPI (web variant)
// TODO: Implement 2FA enable, checkpoint handling, password change via web endpoints
package web

import (
	"context"

	"HVRIns/internal/instagram"
)

// SecurityManager implements instagram.SecurityManager for the web platform.
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
	instagram.RegisterPlatformSecurityManager(instagram.PlatformWeb, func() instagram.SecurityManager {
		return &SecurityManager{}
	})
}
