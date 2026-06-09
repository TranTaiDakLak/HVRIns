// security_api.go — TutVer 1 Security API calls cho S425 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s425

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S425: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "119940804215467192679525549522",
	DocIDConfirmSubEmail: "119940804215467192679525549522",
	BloksVerContact:      "4bec40073343147852635a8f492649cda5f794bc733bb5aa2012fd5509201c66",
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
