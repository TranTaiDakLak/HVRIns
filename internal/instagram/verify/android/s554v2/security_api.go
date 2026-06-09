// security_api.go — TutVer 1 Security API calls cho S554V2 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s554v2

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S554V2: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "1199408042526631289603660492",
	DocIDConfirmSubEmail: "1199408042526631289603660492",
	BloksVerContact:      "d90663010f8c230bedf28906f2bac9c1d1f532a275373050778e36e76a7cb999",
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
