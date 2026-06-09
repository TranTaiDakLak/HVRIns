// Package android — Facebook Android platform Interactor (skeleton)
// Mapping từ C#: FacebookInteractionAndroidAPI
// TODO: Implement like/comment/share/addFriend via Android app API endpoints
package android

import (
	"context"

	"HVRIns/internal/instagram"
)

// Interactor implements instagram.Interactor for the Android platform.
type Interactor struct{}

func (i *Interactor) Like(_ context.Context, session *instagram.Session, postID string) error {
	return nil // TODO: implement
}

func (i *Interactor) Comment(_ context.Context, session *instagram.Session, postID string, text string) error {
	return nil // TODO: implement
}

func (i *Interactor) Share(_ context.Context, session *instagram.Session, postID string) error {
	return nil // TODO: implement
}

func (i *Interactor) AddFriend(_ context.Context, session *instagram.Session, targetUID string) error {
	return nil // TODO: implement
}

func init() {
	instagram.RegisterPlatformInteractor(instagram.PlatformAndroid, func() instagram.Interactor {
		return &Interactor{}
	})
}
