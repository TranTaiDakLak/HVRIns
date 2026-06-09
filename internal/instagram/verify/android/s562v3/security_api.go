// security_api.go — TutVer 1 Security API calls cho S560 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s562v3

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S560: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080424975422817712143844",
	DocIDConfirmSubEmail: "11994080424975422817712143844",
	BloksVerContact:      "e126032a6d9f10c51b99f542ebc351837d5f039af73ce199a40b86acf10a010b",
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
