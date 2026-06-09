// security_api.go â€” TutVer 1 Security API calls cho S565 verify.
// ToÃ n bá»™ logic chia sáº» á»Ÿ package secapi.
package s565v2s21

import (
	"HVRIns/internal/instagram/verify/secapi"
)

// securitySpec â€” biáº¿n thá»ƒ S565: docID/bloksVer má»›i hoÃ n toÃ n, metaZca = "empty_token".
var securitySpec = secapi.Spec{
	DocIDAddSubEmail:     secapi.DocIDAddSubEmailDefault,
	DocIDContactPoint:    "11994080428380165281378204618",
	DocIDConfirmSubEmail: "11994080428380165281378204618",
	BloksVerContact:      "8e49df647dadb22e676275e71f803c0045cccaf178e55a3033ee1e14eb0c816c",
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
