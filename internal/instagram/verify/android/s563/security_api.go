// security_api.go â€” TutVer 1 Security API calls cho S563 verify.
// ToÃ n bá»™ logic chia sáº» á»Ÿ package secapi.
package s563

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec â€” biáº¿n thá»ƒ S563: docID/bloksVer má»›i hoÃ n toÃ n, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "1199408042594970992837994886",
	DocIDConfirmSubEmail: "1199408042594970992837994886",
	BloksVerContact:      "752772a2bf689dac5dc5fab7c56e888fe08e0569bef53cee7890974e5457ec75",
	MetaZcaValue:         "empty_token",
	ThemeParamsJSON:      secapi.ThemeXMDS_FDS,
	IsPushOn:             false,
}

type securityAPI = secapi.Client
type addSubEmailResult = secapi.AddSubEmailResult

func newSecurityAPI(proxyStr, token, uid, deviceID, machineID, locale, ua string) (*securityAPI, error) {
	return secapi.NewClient(securitySpec, proxyStr, token, uid, deviceID, machineID, locale, ua)
}

func MaskEmail(email string) string { return secapi.MaskEmail(email) }
