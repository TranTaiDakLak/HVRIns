// security_api.go — TutVer 1 Security API calls cho S564V1S23 verify.
// Toàn bộ logic chia sẻ ở package secapi. Constants build 564.0.0.0.17 (capture 2026-05).
package s564v1s23

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S564V1S23: docID build 564.0.0.0.17 mới, bloksVer giữ nguyên, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080428908932823877210452",
	DocIDConfirmSubEmail: "11994080428908932823877210452",
	BloksVerContact:      "c36891378e117e5eaef5b658a382d93c08b183fa9495d1facec29fbb62a81f8a",
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
