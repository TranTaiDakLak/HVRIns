// Package web — Facebook Web platform FeedReader (skeleton)
// Mapping từ C#: FacebookFeedAPI (web variant)
// TODO: Implement news feed reading via m.facebook.com GraphQL/Bloks endpoints
package web

import (
	"context"

	"HVRIns/internal/instagram"
)

// FeedReader implements instagram.FeedReader for the web platform.
type FeedReader struct{}

// GetFeed retrieves the news feed for the given session.
// Currently returns not-implemented; full implementation pending.
func (f *FeedReader) GetFeed(_ context.Context, session *instagram.Session, limit int) ([]instagram.FeedPost, error) {
	return nil, nil // TODO: implement
}

func init() {
	instagram.RegisterPlatformFeedReader(instagram.PlatformWeb, func() instagram.FeedReader {
		return &FeedReader{}
	})
}
