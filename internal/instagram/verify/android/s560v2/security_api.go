// security_api.go — TutVer 1 Security API calls cho S560 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s560v2

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S560: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "119940804210382635615484396344",
	DocIDConfirmSubEmail: "119940804210382635615484396344",
	BloksVerContact:      "182a4e03087cc46a88a95c6e5747a622a0ab08e2134522bd0e6da65b9ceea9fd",
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
