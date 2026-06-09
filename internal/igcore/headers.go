// headers.go — header builder cho graphql_www (x-ig-* scheme), khớp capture.
package igcore

import (
	"fmt"
	"strings"
)

// localeToAcceptLang chuyển "vi_VN" → "vi-VN;q=1.0".
func localeToAcceptLang(locale string) string {
	lang := strings.ReplaceAll(locale, "_", "-")
	code := locale[:2] // "vi"
	if code == "en" {
		return lang + ";q=1.0"
	}
	return lang + ";q=1.0," + code + ";q=0.9,en-US;q=0.8"
}

// localeToTimezone trả timezone offset phù hợp với locale.
func localeToTimezone(locale string) string {
	switch locale[:2] {
	case "vi", "th", "id", "ms":
		return "25200" // UTC+7
	case "en":
		if strings.Contains(locale, "AU") {
			return "36000" // UTC+10
		}
		if strings.Contains(locale, "GB") {
			return "0"
		}
		return "-18000" // US Eastern
	default:
		return "25200"
	}
}

// bloksHeaders dựng header cho 1 POST graphql_www Bloks step.
// friendlyAppID = app_id của step (vd com.bloks.www.bloks.caa.reg.create.account.async).
func bloksHeaders(p *igProfile, friendlyAppID string) [][2]string {
	friendly := "IGBloksAppRootQuery-" + friendlyAppID
	analyticsTags := fmt.Sprintf(`{"network_tags":{"product":"%s","surface":"other","is_ad":"0","request_category":"api","purpose":"fetch","retry_attempt":"0"},"application_tags":{"is_nav_critical":"0"}}`, igAppID)

	locale := p.Locale
	if locale == "" {
		locale = "vi_VN"
	}
	lang := locale[:2]
	localeHyphen := strings.ReplaceAll(locale, "_", "-")
	acceptLang := localeToAcceptLang(locale)
	tzOffset := localeToTimezone(locale)

	return [][2]string{
		{"user-agent", p.UserAgent},
		{"accept-encoding", "zstd"},
		{"accept", "*/*"},
		{"x-fb-friendly-name", friendly},
		{"content-type", "application/x-www-form-urlencoded; charset=UTF-8"},
		{"x-graphql-request-purpose", "fetch"},
		{"x-graphql-client-library", "pando"},
		{"x-fb-request-analytics-tags", analyticsTags},
		{"x-fb-rmd", "state=URL_ELIGIBLE"},
		{"x-meta-usdid", genUSDID(p)},
		{"x-ig-app-id", igAppID},
		{"x-mid", p.MachineID},
		{"x-ig-bandwidth-speed-kbps", "0.000"},
		{"x-cloud-trust-token", p.CloudTrustID},
		{"x-pigeon-session-id", p.PigeonSID},
		{"x-pigeon-rawclienttime", fmt.Sprintf("%d.000000", nowUnix())},
		{"x-ig-bloks-serialize-payload", "true"},
		{"accept-language", acceptLang},
		{"x-ig-timezone-offset", tzOffset},
		{"x-fb-connection-type", "wifi"},
		{"x-ig-device-id", p.DeviceID},
		{"x-ig-family-device-id", p.FamilyDeviceID},
		{"ig-intended-user-id", "0"},
		{"x-ig-connection-type", "WiFi"},
		{"x-bloks-version-id", bloksVersionID},
		{"x-tigon-is-retry", "False"},
		{"x-fb-server-cluster", "True"},
		{"x-fb-client-ip", "True"},
		{"x-fb-conn-uuid-client", p.ConnUUID},
		{"x-bloks-prism-extended-palette-gray", "false"},
		{"x-ig-connection-speed", "59kbps"},
		{"x-ig-abr-connection-speed-kbps", "185"},
		{"x-bloks-prism-extended-palette-indigo", "false"},
		{"x-bloks-prism-extended-palette-polish-enabled", "false"},
		{"x-bloks-prism-link-colors-enabled", "0"},
		{"x-bloks-prism-font-enabled", "false"},
		{"x-ig-capabilities", "36r/F/8="},
		{"x-ig-mapped-locale", locale},
		{"x-ig-app-locale", lang},
		{"x-ig-device-locale", localeHyphen},
		{"x-bloks-prism-extended-palette-red", "false"},
		{"x-bloks-prism-ax-base-colors-enabled", "true"},
		{"x-bloks-prism-colors-enabled", "true"},
		{"x-bloks-prism-extended-palette-rest-of-colors", "false"},
		{"x-bloks-is-prism-enabled", "false"},
	}
}
