// Package facebook — Status codes for API call results
// Mapping từ C#: FbApiCallStatusCode enum
package instagram

// Status represents the outcome of a Facebook API call.
// Mapping từ C#: FbApiCallStatusCode
type Status string

const (
	StatusSuccess          Status = "success"
	StatusFail             Status = "fail"
	StatusCheckpoint       Status = "checkpoint"
	StatusDisabled         Status = "disabled"
	StatusProxyError       Status = "proxy_error"
	StatusTimeout          Status = "timeout"
	StatusRateLimit        Status = "rate_limit"
	StatusInvalidSession   Status = "invalid_session"
	StatusNotImplemented   Status = "not_implemented"
	StatusNeedVerify       Status = "need_verify"
)

// IsSuccess returns true if the status indicates a successful outcome.
func (s Status) IsSuccess() bool { return s == StatusSuccess }

// IsError returns true if the status indicates any error condition.
func (s Status) IsError() bool { return s != StatusSuccess && s != StatusNeedVerify }
