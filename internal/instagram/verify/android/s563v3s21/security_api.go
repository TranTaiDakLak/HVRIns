// security_api.go — TutVer 1 Security API calls cho S563V3S21 verify.
// Toàn bộ logic chia sẻ ở package secapi. Constants build 563.0.0.0.48 (capture 2026-05).
package s563v3s21

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S563V3S21: docID build 563.0.0.0.48 mới, bloksVer giữ nguyên, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080422283368733301219073",
	DocIDConfirmSubEmail: "11994080422283368733301219073",
	BloksVerContact:      "aa4e5e759b2f90492ba1b311dc98988f125b76705083a7e8b3fa11a3262f7459",
	MetaZcaValue:         "empty_token",
	ThemeParamsJSON:      secapi.ThemeFDSOnly,
	IsPushOn:             true,
}

type securityAPI = secapi.Client
type addSubEmailResult = secapi.AddSubEmailResult

func newSecurityAPI(proxyStr, token, uid, deviceID, machineID, locale, ua string) (*securityAPI, error) {
	return secapi.NewClient(securitySpec, proxyStr, token, uid, deviceID, machineID, locale, ua)
}

func MaskEmail(email string) string { return secapi.MaskEmail(email) }
