// security_api.go — TutVer 1 Security API calls cho S551V2 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s551v2

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S551V2: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080429528009468341525997",
	DocIDConfirmSubEmail: "11994080429528009468341525997",
	BloksVerContact:      "63694f6c778ab8e6d0330546791053080be9f1b08bf6985d8c613f005e34f478",
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
