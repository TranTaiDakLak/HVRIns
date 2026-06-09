// helpers.go — S399 verify body builders + headers + UA template.
// Khác s5xx Bloks: gửi form params phẳng tới graph.facebook.com/me/* (không qua /graphql).
package s399

import (
	"fmt"
	mrand "math/rand"
	"net/url"
	"strings"
	"time"

	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/instagram/verify/verifybase"
)

// ─── S399 verify constants ───────────────────────────────────────────────────

const (
	s399FBAV = "399.0.0.24.93"
	s399FBBV = "440587081"

	// Endpoint URLs (graph.facebook.com, không phải b-graph)
	addEmailURL = "https://graph.facebook.com/me/edit_registration_contactpoint"
	confirmURL  = "https://graph.facebook.com/me/confirm_contactpoint"

	addEmailFriendly = "editRegistrationContactpoint"
	confirmFriendly  = "confirmContactpoint"
	addEmailCaller   = "ConfContactpointFragment"
	confirmCaller    = "ConfCodeInputFragment"
)

// ─── Device pool (giống register/s399) ───────────────────────────────────────

type s399VerifyDevice struct {
	Model     string
	BuildID   string
	OSVersion string
	Width     int
	Height    int
	Density   string
}

var s399VerifyDevices = []s399VerifyDevice{
	{Model: "SM-S911B", BuildID: "AP3A.240905.015.A2", OSVersion: "15", Width: 1080, Height: 2115, Density: "3.0"},
	{Model: "SM-S911U", BuildID: "AP3A.240905.015.A2", OSVersion: "15", Width: 1080, Height: 2115, Density: "3.0"},
	{Model: "SM-S916B", BuildID: "AP3A.240905.015.A2", OSVersion: "15", Width: 1080, Height: 2115, Density: "3.0"},
	{Model: "SM-S918B", BuildID: "AP3A.240905.015.A2", OSVersion: "15", Width: 1440, Height: 3088, Density: "3.0"},
}

// build399VerifyUA — UA template cho verify v399 (KHÔNG có Dalvik prefix).
// Chỉ FBAV/FBBV/locale/carrier/device thay đổi; structure giữ nguyên.
func build399VerifyUA(locale, carrier string) string {
	if locale == "" {
		locale = "en_US"
	}
	if carrier == "" {
		carrier = fakeinfo.RandomCarrier()
		if carrier == "" {
			carrier = "T-Mobile"
		}
	}
	d := s399VerifyDevices[mrand.Intn(len(s399VerifyDevices))]
	return fmt.Sprintf(
		"[FBAN/FB4A;FBAV/%s;FBBV/%s;FBDM/{density=%s,width=%d,height=%d};FBLC/%s;FBRV/0;FBCR/%s;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/%s;FBSV/%s;FBOP/1;FBCA/arm64-v8a:;]",
		s399FBAV, s399FBBV, d.Density, d.Width, d.Height, locale, carrier, d.Model, d.OSVersion,
	)
}

// RandomUA — wired vào factory.RegisterPlatformVerifyUA(PlatformS399, ...) trong verify.go.
func RandomUA(countryCode string) string {
	locale, carrier := verifybase.PickCountryCarrierLocale(countryCode)
	return build399VerifyUA(locale, carrier)
}

// ─── Body builders ───────────────────────────────────────────────────────────

// buildAddEmailBody — POST /me/edit_registration_contactpoint
func buildAddEmailBody(emailAddr, locale, countryCode string) string {
	params := url.Values{}
	params.Set("add_contactpoint", emailAddr)
	params.Set("add_contactpoint_type", "EMAIL")
	params.Set("format", "json")
	params.Set("locale", locale)
	params.Set("client_country_code", countryCode)
	params.Set("fb_api_req_friendly_name", addEmailFriendly)
	params.Set("fb_api_caller_class", addEmailCaller)
	return params.Encode()
}

// buildConfirmCodeBody — POST /me/confirm_contactpoint
func buildConfirmCodeBody(emailAddr, code, locale, countryCode string) string {
	params := url.Values{}
	params.Set("normalized_contactpoint", emailAddr)
	params.Set("contactpoint_type", "EMAIL")
	params.Set("code", code)
	params.Set("source", "ANDROID_DIALOG_API")
	params.Set("surface", "hard_cliff")
	params.Set("format", "json")
	params.Set("locale", locale)
	params.Set("client_country_code", countryCode)
	params.Set("fb_api_req_friendly_name", confirmFriendly)
	params.Set("fb_api_caller_class", confirmCaller)
	return params.Encode()
}

// ─── Headers ─────────────────────────────────────────────────────────────────

// buildVerifyHeaders — common headers cho cả 2 endpoint. friendlyName đổi theo step.
func buildVerifyHeaders(ua, token, simHNI, friendlyName, deviceGroup, connType string) [][2]string {
	if simHNI == "" {
		simHNI = "45204"
	}
	if deviceGroup == "" {
		deviceGroup = "2610"
	}
	if connType == "" {
		connType = "WIFI"
	}
	return [][2]string{
		{"host", "graph.facebook.com"},
		{"x-zero-eh", "2,,AV9ML7-dZ3dEwmYTK9Cdbx-cvsdK6_aYDrPvCU5vgsFC-4T1dh_O8hcdZQOvDJw-G7k"},
		{"x-fb-connection-quality", "EXCELLENT"},
		{"x-fb-sim-hni", simHNI},
		{"user-agent", ua},
		{"x-fb-connection-bandwidth", "85398833"},
		{"x-fb-client-id", randomClientID()},
		{"authorization", "OAuth " + token},
		{"x-fb-net-hni", simHNI},
		{"content-encoding", "gzip"},
		{"zero-rated", "0"},
		{"content-type", "application/x-www-form-urlencoded"},
		{"x-fb-connection-type", connType},
		{"x-fb-device-group", deviceGroup},
		{"x-tigon-is-retry", "False"},
		{"x-fb-friendly-name", friendlyName},
		{"x-fb-request-analytics-tags", `{"network_tags":{"retry_attempt":"0"},"application_tags":"unknown"}`},
		{"priority", "u=3, i"},
		{"accept-encoding", "gzip, deflate"},
		{"x-fb-http-engine", "Liger"},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
	}
}

func randomClientID() string {
	r := mrand.New(mrand.NewSource(time.Now().UnixNano() + mrand.Int63()))
	const hex = "0123456789abcdef"
	pat := []int{8, 4, 4, 4, 12}
	parts := make([]string, len(pat))
	for i, n := range pat {
		b := make([]byte, n)
		for j := range b {
			b[j] = hex[r.Intn(16)]
		}
		parts[i] = string(b)
	}
	return strings.Join(parts, "-")
}
