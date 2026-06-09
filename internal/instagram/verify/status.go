// Package verify — Status codes for verify operations
// Mapping từ C#: AddEmailStatusCode, ConfirmEmailStatusCode, VeriAccountAutoStatusCode
package verify

// AddEmailStatus is the result of an AddEmail operation.
// Mapping từ C#: AddEmailStatusCode
type AddEmailStatus string

const (
	AddEmailOK            AddEmailStatus = "ok"
	AddEmailAlreadyLinked AddEmailStatus = "already_linked"
	AddEmailRateLimit     AddEmailStatus = "rate_limit"
	AddEmailCheckpoint    AddEmailStatus = "checkpoint"
	AddEmailFail          AddEmailStatus = "fail"
)

// ConfirmEmailStatus is the result of a ConfirmEmail (OTP) operation.
// Mapping từ C#: ConfirmEmailStatusCode
type ConfirmEmailStatus string

const (
	ConfirmEmailOK          ConfirmEmailStatus = "ok"
	ConfirmEmailWrongCode   ConfirmEmailStatus = "wrong_code"
	ConfirmEmailExpired     ConfirmEmailStatus = "expired"
	ConfirmEmailMaxAttempts ConfirmEmailStatus = "max_attempts"
	ConfirmEmailFail        ConfirmEmailStatus = "fail"
)

// AccountAutoStatus is the overall outcome of a full verify-account automation run.
// Mapping từ C#: VeriAccountAutoStatusCode
type AccountAutoStatus string

const (
	AccountAutoLive       AccountAutoStatus = "live"
	AccountAutoDie        AccountAutoStatus = "die"
	AccountAutoCheckpoint AccountAutoStatus = "checkpoint"
	AccountAutoError      AccountAutoStatus = "error"
	AccountAutoSkip       AccountAutoStatus = "skip" // already verified
)
