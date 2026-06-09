// security_api.go — TutVer 1 Security API calls cho S550V2 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s550v2

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S550V2: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080421180714473335159102",
	DocIDConfirmSubEmail: "11994080421180714473335159102",
	BloksVerContact:      "747cad6608b439e9762d76e9c7e91a0c487d51a32958302fb5fe7424ca1a1ef1",
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
