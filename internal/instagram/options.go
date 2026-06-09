// Package facebook — Per-feature option structs
// Mapping từ C#: FacebookApiCallOptions.* hierarchy (15 option types)
package instagram

// FeedOptions controls news feed reading behavior.
type FeedOptions struct {
	Limit       int    // max posts to fetch (default 20)
	After       string // pagination cursor
	Platform    string // "web" | "android"
}

// InteractionOptions controls like/comment/share behavior.
type InteractionOptions struct {
	Platform      string // "web" | "android"
	DelayMinMS    int    // min delay between actions in milliseconds
	DelayMaxMS    int    // max delay between actions in milliseconds
	RetryCount    int    // number of retries on failure
}

// SecurityOptions controls 2FA/checkpoint/password operations.
type SecurityOptions struct {
	Platform       string // "web" | "android" | "webandroid"
	TwoFAMethod    string // "app" | "sms"
	NewPassword    string // for ChangePassword
}

// RegisterOptions extends RegInput with platform routing.
type RegisterOptions struct {
	Platform string // "web" | "android" | "chrome" | "webandroid"
}

// VerifyOptions extends VerifyConfig with platform routing.
type VerifyOptions struct {
	Platform string // "web" | "android" | "webandroid" | "webrequest"
}
