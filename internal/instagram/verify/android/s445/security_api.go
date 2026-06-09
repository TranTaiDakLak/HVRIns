// security_api.go — TutVer 1 Security API calls cho S445 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s445

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S445: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "1199408042870689966045625207",
	DocIDConfirmSubEmail: "1199408042870689966045625207",
	BloksVerContact:      "2b781981a7f5aea22e309c8680b628b545bea6f0afa2b930a5ed772754ff06dc",
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
