// security_api.go — TutVer 1 Security API calls cho S558V2 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s558v2

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S558V2: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "119940804212218769903962986547",
	DocIDConfirmSubEmail: "119940804212218769903962986547",
	BloksVerContact:      "675d0e80077ec27b1248cdcfb41273ddb3e589b7aa4b2a45f6c013a5b682486b",
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
