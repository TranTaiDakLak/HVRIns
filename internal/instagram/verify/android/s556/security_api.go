// security_api.go — TutVer 1 Security API calls cho S556 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s556

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S556: docID/bloksVer giống s557 (legacy), XMDS+FDS theme.
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080422955588194694478490",
	DocIDConfirmSubEmail: "11994080422955588194694478490",
	BloksVerContact:      "385fe019aa6b5903bdad3a4799063e3fc70da9cd1fda8b54189bce078c701665",
	MetaZcaValue:         secapi.MetaZcaBase64,
	ThemeParamsJSON:      secapi.ThemeXMDS_FDS,
	IsPushOn:             false,
}

type securityAPI = secapi.Client
type addSubEmailResult = secapi.AddSubEmailResult

func newSecurityAPI(proxyStr, token, uid, deviceID, machineID, locale, ua string) (*securityAPI, error) {
	return secapi.NewClient(securitySpec, proxyStr, token, uid, deviceID, machineID, locale, ua)
}

func MaskEmail(email string) string { return secapi.MaskEmail(email) }
