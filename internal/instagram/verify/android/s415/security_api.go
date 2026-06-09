// security_api.go — TutVer 1 Security API calls cho S415 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s415

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S415: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080429355460875507864846",
	DocIDConfirmSubEmail: "11994080429355460875507864846",
	BloksVerContact:      "bd8701d8d9ba91295f01e2b6aef0ee8ab6184dc25dd3b46af83702f27c3079b7",
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
