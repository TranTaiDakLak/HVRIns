// Package facebook — Per-feature API call result structs
// Mapping từ C#: FacebookApiCallResult.* hierarchy (17 result types)
package instagram

// FeedResult holds the outcome of a news feed read operation.
type FeedResult struct {
	Status  Status
	Posts   []FeedPost
	Cursor  string // pagination cursor for next page
	Message string
}

// InteractionResult holds the outcome of a like/comment/share/addFriend operation.
type InteractionResult struct {
	Status  Status
	PostID  string
	Message string
}

// SecurityResult holds the outcome of a security operation (2FA, checkpoint, password).
type SecurityResult struct {
	Status    Status
	TwoFA     *TwoFAResult
	Message   string
}

// CheckpointResult holds information about a detected checkpoint challenge.
type CheckpointResult struct {
	Status  Status
	Type    string // from checkpoint.Type.String()
	Message string
}

// LoginResultFull extends LoginResult with session cookie data.
// Mapping từ C#: FacebookLoginResult (extended)
type LoginResultFull struct {
	Status      Status
	Session     *Session
	AccessToken string
	Message     string
}
