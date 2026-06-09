// security_api.go — TutVer 1 Security API calls cho S435 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s435

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S435: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080428003704954252325796",
	DocIDConfirmSubEmail: "11994080428003704954252325796",
	BloksVerContact:      "81beec0074667d3bccd1808eb9072789f784db11770b14ef5aa9a756e810ebe5",
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
