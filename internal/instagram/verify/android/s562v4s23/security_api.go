// security_api.go — TutVer 1 Security API calls cho S560 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s562v4s23

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S560: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "119940804212653772530264493669",
	DocIDConfirmSubEmail: "119940804212653772530264493669",
	BloksVerContact:      "ea49f1cd6ce051bc2f45bf5e30780f144df6e9a7e2fe3bbf011a1b0c68a90344",
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
