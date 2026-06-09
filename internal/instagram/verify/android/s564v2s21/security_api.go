// security_api.go — TutVer 1 Security API calls cho S564V1S21 verify.
// Toàn bộ logic chia sẻ ở package secapi. Constants build 564.0.0.0.61 (capture 2026-05).
package s564v2s21

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S564V1S21: docID build 564.0.0.0.61 mới, bloksVer giữ nguyên, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080423734609707368500843",
	DocIDConfirmSubEmail: "11994080423734609707368500843",
	BloksVerContact:      "ebb84a871fc81d8889c76b8300f0825f5864655c74af6e62e512aedb615e5a80",
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
