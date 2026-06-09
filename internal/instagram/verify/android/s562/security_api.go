// security_api.go — TutVer 1 Security API calls cho S560 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s562

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S560: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080428164317824000724862",
	DocIDConfirmSubEmail: "11994080428164317824000724862",
	BloksVerContact:      "43e92dfca371c08a4cfacb4b4340e2f8c2ee67957c7965e326ea6631c0629e84",
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
