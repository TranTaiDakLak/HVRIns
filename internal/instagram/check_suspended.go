// check_suspended.go — kiểm tra account IG live/suspended ngay sau reg.
package instagram

import (
	"context"

	"HVRIns/internal/igcore"
)

// SuspendedStatus là kết quả check.
type SuspendedStatus int

const (
	SuspendedStatusLive      SuspendedStatus = iota // account live, không bị suspend
	SuspendedStatusSuspended                        // account bị suspend
	SuspendedStatusUnknown                          // không check được (timeout, network error)
)

// CheckSuspended kiểm tra account IG live/suspended bằng cách kiểm tra username
// có còn tồn tại trên Instagram không. Dùng Chrome TLS fingerprint qua web_profile_info.
// cookieStr: cookie của account cần check (dùng làm checker cookie).
// username: tên người dùng Instagram (vd "falcon.3900382").
// proxyStr: proxy (có thể rỗng).
func CheckSuspended(ctx context.Context, cookieStr, username, proxyStr string) SuspendedStatus {
	if cookieStr == "" || username == "" {
		return SuspendedStatusUnknown
	}
	switch igcore.CheckLiveByCheckerCookie(ctx, cookieStr, username, proxyStr) {
	case "live":
		return SuspendedStatusLive
	case "die":
		return SuspendedStatusSuspended
	default:
		return SuspendedStatusUnknown
	}
}
