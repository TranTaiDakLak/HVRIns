// security_api.go — TutVer 1 Security API calls cho S23 verify.
// Toàn bộ logic chia sẻ ở package secapi; file này chỉ khai báo Spec và
// alias để steps.go gọi qua tên cũ (securityAPI / addSubEmailResult / MaskEmail).
package s23

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S23: doc_id chuẩn cũ, base64 metaZca, FDS-only theme, is_push_on=true.
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080422955588194694478490",
	DocIDConfirmSubEmail: "11994080422955588194694478490",
	BloksVerContact:      "385fe019aa6b5903bdad3a4799063e3fc70da9cd1fda8b54189bce078c701665",
	MetaZcaValue:         secapi.MetaZcaBase64,
	ThemeParamsJSON:      secapi.ThemeS23,
	IsPushOn:             true,
}

// Aliases — giữ tên cũ để steps.go không phải sửa.
type securityAPI = secapi.Client
type addSubEmailResult = secapi.AddSubEmailResult

// newSecurityAPI tạo Client với spec S23.
func newSecurityAPI(proxyStr, token, uid, deviceID, machineID, locale, ua string) (*securityAPI, error) {
	return secapi.NewClient(securitySpec, proxyStr, token, uid, deviceID, machineID, locale, ua)
}

// MaskEmail — re-export từ secapi để steps.go gọi qua tên cũ.
func MaskEmail(email string) string { return secapi.MaskEmail(email) }
