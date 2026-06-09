// security_api.go — TutVer 1 Security API calls cho S560 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s561

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S560: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "119940804210934765769791410287",
	DocIDConfirmSubEmail: "119940804210934765769791410287",
	BloksVerContact:      "9d448b7c3b47250635f0acbdd801409700933b0eba77ec236358984692f4d562",
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
