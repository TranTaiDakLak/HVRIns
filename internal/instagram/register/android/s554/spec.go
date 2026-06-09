package s554

import (
	"fmt"
	"strings"

	"HVRIns/internal/instagram"
)

const (
	Platform         = instagram.PlatformS554
	s54xOAuthToken   = "350685531728|62f8ce9f74b12f84c123cc23437a4a32"
	s54xMetaZCA      = "empty_token"
	s54xFriendlyName = "FbBloksActionRootQuery-com.bloks.www.bloks.caa.reg.create.account.async"
	themeFDSOnly     = "%5B%7B%22value%22%3A%5B%5D%2C%22design_system_name%22%3A%22FDS%22%7D%5D"
	themeXMDSFDS     = "%5B%7B%22value%22%3A%5B%22three_neutral_gray%22%5D%2C%22design_system_name%22%3A%22XMDS%22%7D%2C%7B%22value%22%3A%5B%5D%2C%22design_system_name%22%3A%22FDS%22%7D%5D"
)

// Spec contains the captured constants for one FB4A registration version.
type Spec struct {
	Platform     string
	Label        string
	Version      string
	Build        string
	DocID        string
	BloksVer     string
	StylesID     string
	ZeroEH       string
	ThemeParams  string
	IsPushOn     bool
	DeviceGroup  string
	OAuthToken   string
	MetaZCA      string
	FriendlyName string
}

var spec = newSpec(Platform, "S554", "554.0.0.57.70", "926292396", "1199408042526631289603660492", "d90663010f8c230bedf28906f2bac9c1d1f532a275373050778e36e76a7cb999", "6100e7e89411ccf67ace027cedecd84f", "664c0faaac849cb891d0a261fbb72a12", themeFDSOnly, true)

func newSpec(platform, label, version, build, docID, bloksVer, stylesID, zeroEH, themeParams string, isPushOn bool) Spec {
	return Spec{
		Platform:     platform,
		Label:        label,
		Version:      version,
		Build:        build,
		DocID:        docID,
		BloksVer:     bloksVer,
		StylesID:     stylesID,
		ZeroEH:       zeroEH,
		ThemeParams:  themeParams,
		IsPushOn:     isPushOn,
		DeviceGroup:  "2610",
		OAuthToken:   s54xOAuthToken,
		MetaZCA:      s54xMetaZCA,
		FriendlyName: s54xFriendlyName,
	}
}

func SpecForPlatform(platform string) (Spec, bool) {
	if strings.EqualFold(strings.TrimSpace(platform), Platform) {
		return spec, true
	}
	return Spec{}, false
}

func IsPlatform(platform string) bool {
	_, ok := SpecForPlatform(platform)
	return ok
}

func OriginalUAForPlatform(platform string) string {
	spec, ok := SpecForPlatform(platform)
	if !ok {
		return ""
	}
	return fmt.Sprintf("[FBAN/FB4A;FBAV/%s;FBBV/%s;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]", spec.Version, spec.Build)
}

var OriginalUA = OriginalUAForPlatform(Platform)

func defaultSpec() Spec {
	return spec
}

func boolLiteral(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
