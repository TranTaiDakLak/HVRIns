// security_api.go — TutVer 1 Security API calls cho S455 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s455

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S455: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080422911873693149755833",
	DocIDConfirmSubEmail: "11994080422911873693149755833",
	BloksVerContact:      "646a342968698e749c600b41a0e530e4386d17fae2cf4aa21cf3ca7b47c3771d",
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
