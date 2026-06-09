// security_api.go — TutVer 1 Security API calls cho S557V2 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s557v2

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S557V2: docID/bloksVer mới hoàn toàn, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080426813324353671116601",
	DocIDConfirmSubEmail: "11994080426813324353671116601",
	BloksVerContact:      "4592ee75c51ab0ea74b16750062e87fa8b695e24c94b9ae0100ef933b22f0499",
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
