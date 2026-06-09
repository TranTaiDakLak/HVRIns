// security_api.go — TutVer 1 Security API calls cho S563V4S21 verify.
// Toàn bộ logic chia sẻ ở package secapi. Constants build 563.0.0.23.73 (capture 2026-05).
package s563v5s21

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S563V4S21: docID build 563.0.0.23.73 mới, bloksVer giữ nguyên, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080426742906638567181549",
	DocIDConfirmSubEmail: "11994080426742906638567181549",
	BloksVerContact:      "f9d474f2985f03adc1635376403e3dd685665c1ad331ef9e08731e66a6f7e77a",
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
