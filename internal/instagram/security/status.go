// Package security — Status codes for security feature operations
// Mapping từ C#: AccountFeaturesAutoStatusCode, Enable2FAStatusCode
package security

// FeaturesAutoStatus is the overall outcome of a security features automation run.
// Mapping từ C#: AccountFeaturesAutoStatusCode
type FeaturesAutoStatus string

const (
	FeaturesAutoOK         FeaturesAutoStatus = "ok"
	FeaturesAutoFail       FeaturesAutoStatus = "fail"
	FeaturesAutoCheckpoint FeaturesAutoStatus = "checkpoint"
	FeaturesAutoSkip       FeaturesAutoStatus = "skip" // already configured
)

// Enable2FAStatus is the result of an Enable2FA operation.
// Mapping từ C#: Enable2FAStatusCode
type Enable2FAStatus string

const (
	Enable2FAOK         Enable2FAStatus = "ok"
	Enable2FAAlready    Enable2FAStatus = "already_enabled"
	Enable2FAFail       Enable2FAStatus = "fail"
	Enable2FACheckpoint Enable2FAStatus = "checkpoint"
)
