// security_api.go — TutVer 1 Security API calls cho S555 verify.
// Toàn bộ logic chia sẻ ở package secapi.
package s555

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec — biến thể S555: docID/bloksVer riêng, base64 metaZca, XMDS+FDS theme, is_push_on=false.
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "119940804217734265480409226803",
	DocIDConfirmSubEmail: "119940804217734265480409226803",
	BloksVerContact:      "d1583f026cccd22345fea8de656bb1d8162dabcca3249d6a0610be47545ec31a",
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
