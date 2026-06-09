// security_api.go — TutVer 1 Security API calls cho S552V2 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s552v2

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S552V2: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080424316867831018805237",
	DocIDConfirmSubEmail: "11994080424316867831018805237",
	BloksVerContact:      "51ee0924c1a66b5dd670a44370f6278f66485250d78f72d036fb4ff6480c1d16",
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
