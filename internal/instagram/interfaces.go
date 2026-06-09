// Package facebook — Interfaces chung cho tất cả nền tảng Facebook
// Mỗi nền tảng (android/, web/, mfb/) implement cùng bộ interface này.
// Thêm nền tảng mới = tạo thư mục + implement — không sửa interface.
package instagram

import "context"

// Registerer — đăng ký tài khoản Facebook mới
// Mapping từ C#: IFacebookRegister (Interfaces/IFacebookRegister.cs)
type Registerer interface {
	// Register thực hiện toàn bộ flow đăng ký (B1-B8).
	// onStatus: callback nhận log tiến trình realtime, có thể nil.
	Register(ctx context.Context, input *RegInput, onStatus func(string)) *RegResult
}

// Verifier — verify tài khoản (add email + confirm OTP)
// Mapping từ C#: IFacebookVerifyAPI (Interfaces/IFacebookVerifyAPI.cs)
type Verifier interface {
	// Verify thực hiện flow verify: đổi ngôn ngữ → tạo email → B1-B5 → check live/die.
	// outputPath: thư mục ghi Live.txt/Die.txt, rỗng thì dùng cfg.OutputPath.
	// onStatus: callback nhận (uid, msg) realtime, có thể nil.
	Verify(ctx context.Context, session *Session, cfg *VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *VerifyResult
}

// Interactor — tương tác với account (like, comment, share, add friend)
// Mapping từ C#: FacebookInteractionAndroidAPI / FacebookInteractionAPI
type Interactor interface {
	Like(ctx context.Context, session *Session, postID string) error
	Comment(ctx context.Context, session *Session, postID string, text string) error
	Share(ctx context.Context, session *Session, postID string) error
	AddFriend(ctx context.Context, session *Session, targetUID string) error
}

// FeedReader — đọc news feed
// Mapping từ C#: FacebookFeedAndroidAPI / FacebookFeedAPI
type FeedReader interface {
	GetFeed(ctx context.Context, session *Session, limit int) ([]FeedPost, error)
}

// SecurityManager — 2FA, checkpoint, đổi mật khẩu
// Mapping từ C#: FacebookSecurityFeatureAPIAndroid / FacebookSecurityFeatureAPI
type SecurityManager interface {
	Enable2FA(ctx context.Context, session *Session) (*TwoFAResult, error)
	HandleCheckpoint(ctx context.Context, session *Session) error
	ChangePassword(ctx context.Context, session *Session, newPassword string) error
}
