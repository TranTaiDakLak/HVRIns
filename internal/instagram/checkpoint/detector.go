// Package checkpoint — Detect and classify Facebook checkpoint challenges
// Mapping từ C#: FacebookCheckpointDetectorUtils
//
// Facebook returns checkpoint challenges when it suspects automated activity.
// This package provides detection (identify the checkpoint type) and basic
// classification (identity confirm, phone confirm, CAPTCHA, etc.).
//
// TODO: Port from C# FacebookCheckpointDetectorUtils + checkpoint handler flows
package checkpoint

// Type represents the category of a Facebook checkpoint challenge.
type Type int

const (
	TypeNone     Type = iota // No checkpoint detected
	TypeIdentity             // Identity confirmation (upload ID, confirm contacts)
	TypePhone                // Phone number confirmation OTP
	TypeEmail                // Email confirmation OTP
	TypeCaptcha              // CAPTCHA challenge
	TypeReview               // Account under review
	TypeDisabled             // Account disabled (permanent)
	TypeUnknown              // Detected but unclassified
)

// String returns a human-readable label for the checkpoint type.
func (t Type) String() string {
	switch t {
	case TypeNone:
		return "none"
	case TypeIdentity:
		return "identity"
	case TypePhone:
		return "phone"
	case TypeEmail:
		return "email"
	case TypeCaptcha:
		return "captcha"
	case TypeReview:
		return "review"
	case TypeDisabled:
		return "disabled"
	default:
		return "unknown"
	}
}

// Detect inspects a raw response body and returns the type of checkpoint
// challenge present, or TypeNone if no checkpoint is detected.
// TODO: Port full detection logic from C# FacebookCheckpointDetectorUtils
func Detect(responseBody string) Type {
	// Placeholder detection — TODO: implement full pattern matching
	if contains(responseBody, "checkpoint") {
		if contains(responseBody, "phone_confirm") || contains(responseBody, "confirm_phone") {
			return TypePhone
		}
		if contains(responseBody, "email_confirm") || contains(responseBody, "confirm_email") {
			return TypeEmail
		}
		if contains(responseBody, "captcha") || contains(responseBody, "recaptcha") {
			return TypeCaptcha
		}
		if contains(responseBody, "disabled") || contains(responseBody, "suspended") {
			return TypeDisabled
		}
		return TypeUnknown
	}
	return TypeNone
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
