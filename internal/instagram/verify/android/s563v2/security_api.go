// security_api.go — TutVer 1 Security API calls cho S563V2 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s563v2

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S563V2: docID/bloksVer mới hoàn toàn (build 563.0.0.0.26), metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "119940804212271057732132442903",
	DocIDConfirmSubEmail: "119940804212271057732132442903",
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
