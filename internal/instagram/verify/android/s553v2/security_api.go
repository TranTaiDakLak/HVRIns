// security_api.go — TutVer 1 Security API calls cho S553V2 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s553v2

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S553V2: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "119940804212296436181797579246",
	DocIDConfirmSubEmail: "119940804212296436181797579246",
	BloksVerContact:      "e6824e76c6ba5c44b62f86b6ae9c904cea8551b02256b18cba10b405e29d8f79",
	MetaZcaValue:         "empty_token",
	ThemeParamsJSON:      secapi.ThemeFDSOnly,
	IsPushOn:             false,
}

type securityAPI = secapi.Client
type addSubEmailResult = secapi.AddSubEmailResult

func newSecurityAPI(proxyStr, token, uid, deviceID, machineID, locale, ua string) (*securityAPI, error) {
	return secapi.NewClient(securitySpec, proxyStr, token, uid, deviceID, machineID, locale, ua)
}

func MaskEmail(email string) string { return secapi.MaskEmail(email) }
