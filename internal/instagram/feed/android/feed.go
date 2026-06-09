// Package android — Facebook Android platform FeedReader (skeleton)
// Mapping từ C#: FacebookFeedAndroidAPI
// TODO: Implement news feed reading via Android app API endpoints
package android

import (
	"context"

	"HVRIns/internal/instagram"
)

// FeedReader implements instagram.FeedReader for the Android platform.
type FeedReader struct{}

// GetFeed retrieves the news feed for the given session.
// Currently returns not-implemented; full implementation pending.
func (f *FeedReader) GetFeed(_ context.Context, session *instagram.Session, limit int) ([]instagram.FeedPost, error) {
	return nil, nil // TODO: implement
}

func init() {
	instagram.RegisterPlatformFeedReader(instagram.PlatformAndroid, func() instagram.FeedReader {
		return &FeedReader{}
	})
}
