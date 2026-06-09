package verifybase

import (
	"fmt"
	mrand "math/rand"
)

// FriendlyNameToReqIdx maps a friendly name to the legacy reqIdx (0=add, 1=confirm, 2=resend).
func FriendlyNameToReqIdx(friendlyName string) int {
	switch friendlyName {
	case ConfirmFriendlyName:
		return 1
	case ResendFriendlyName:
		return 2
	default:
		return 0
	}
}

// BuildLegacyHeaders builds the headers used by s23/s555/s556/s557 verify variants.
// These variants use reqIdx to compute tid offset and include appnet/tasos/session headers.
// withZeroState=true for addEmail and confirm; false for resend.
func BuildLegacyHeaders(sc *SessionCtx, friendlyName string, reqIdx int, withZeroState bool) [][2]string {
	analyticsTag := `{"network_tags":{"product":"350685531728","request_category":"graphql","purpose":"fetch","retry_attempt":"0"},"application_tags":"graphservice"}`
	tid := sc.BaseTid + reqIdx*350
	sessionID := fmt.Sprintf("nid=BP2h//qhUbLv;tid=%d;nc=0;fc=0;bc=0", tid)
	_ = reqIdx // tid already computed above
	networkSig := fmt.Sprintf("7c:7b:68:e7:06:%02x", 50+mrand.Intn(10))

	if sc.DeviceGroup == "" {
		sc.InitPinnedHeaders()
	}

	h := [][2]string{
		{"x-fb-request-analytics-tags", analyticsTag},
		{"x-fb-rmd", "state=URL_ELIGIBLE"},
		{"user-agent", sc.UA},
		{"x-zero-f-device-id", sc.DeviceID},
		{"x-graphql-request-purpose", "fetch"},
		{"x-fb-friendly-name", friendlyName},
		{"x-fb-device-group", sc.DeviceGroup},
		{"x-zero-eh", "unknown"},
		{"content-type", "application/x-www-form-urlencoded"},
		{"x-fb-sim-hni", sc.Sim.HNI},
		{"authorization", "OAuth " + sc.Token},
		{"x-fb-ta-logging-ids", sc.TaLoggingID},
		{"x-fb-background-state", "1"},
		{"x-fb-privacy-context", "3643298472347298"},
	}
	if withZeroState {
		h = append(h, [2]string{"x-zero-state", "unknown"})
	}
	h = append(h,
		[2]string{"x-graphql-client-library", "graphservice"},
		[2]string{"x-meta-zca", "empty_token"},
		[2]string{"app-scope-id-header", sc.DeviceID},
		[2]string{"x-fb-connection-type", "WIFI"},
		[2]string{"x-fb-net-hni", sc.Sim.HNI},
		[2]string{"x-fb-session-id", sessionID},
		[2]string{"x-network-signal", networkSig},
		[2]string{"x-fb-network-properties", "Wifi;"},
		[2]string{"content-encoding", "gzip"},
		[2]string{"x-meta-usdid", GenUSDID()},
		[2]string{"priority", "u=0"},
		[2]string{"x-meta-enable-tasos-ss-bwe", "1"},
		[2]string{"x-fb-tasos-experimental", "1"},
		[2]string{"x-fb-appnetsession-sid", sc.AppnetSID},
		[2]string{"x-fb-appnetsession-nid", sc.AppnetNID},
		[2]string{"x-tigon-is-retry", "False"},
		[2]string{"accept-encoding", "zstd, gzip, deflate"},
		[2]string{"x-fb-http-engine", "Tigon/Liger"},
		[2]string{"x-fb-client-ip", "True"},
		[2]string{"x-fb-server-cluster", "True"},
		[2]string{"x-fb-conn-uuid-client", sc.ConnUUID},
	)
	return h
}

// BuildNewStyleHeaders builds the headers used by s558/s559 verify variants.
// These drop appnet/tasos/session/network-signal/privacy headers and add x-fb-integrity-machine-id.
// withZeroState=true for addEmail and confirm; false for resend.
func BuildNewStyleHeaders(sc *SessionCtx, friendlyName string, withZeroState bool) [][2]string {
	analyticsTag := `{"network_tags":{"product":"350685531728","request_category":"graphql","purpose":"fetch","retry_attempt":"0"},"application_tags":"graphservice"}`

	if sc.DeviceGroup == "" {
		sc.InitPinnedHeaders()
	}

	h := [][2]string{
		{"x-fb-request-analytics-tags", analyticsTag},
		{"x-fb-rmd", "state=URL_ELIGIBLE"},
		{"priority", "u=0"},
		{"content-encoding", "gzip"},
		{"x-fb-device-group", sc.DeviceGroup},
		{"x-fb-integrity-machine-id", sc.MachineID},
		{"x-zero-eh", "unknown"},
		{"user-agent", sc.UA},
		{"x-graphql-request-purpose", "fetch"},
		{"x-fb-friendly-name", friendlyName},
		{"x-zero-f-device-id", sc.DeviceID},
		{"x-tigon-is-retry", "False"},
	}
	if withZeroState {
		h = append(h, [2]string{"x-zero-state", "unknown"})
	}
	h = append(h,
		[2]string{"x-graphql-client-library", "graphservice"},
		[2]string{"x-fb-sim-hni", sc.Sim.HNI},
		[2]string{"content-type", "application/x-www-form-urlencoded"},
		[2]string{"x-fb-net-hni", sc.Sim.HNI},
		[2]string{"authorization", "OAuth " + sc.Token},
		[2]string{"x-meta-zca", "empty_token"},
		[2]string{"app-scope-id-header", sc.DeviceID},
		[2]string{"x-fb-connection-type", "WIFI"},
		[2]string{"x-meta-usdid", GenUSDID()},
		[2]string{"accept-encoding", "gzip, deflate"},
		[2]string{"x-fb-http-engine", "Tigon/Liger"},
		[2]string{"x-fb-client-ip", "True"},
		[2]string{"x-fb-server-cluster", "True"},
		[2]string{"x-fb-conn-uuid-client", sc.ConnUUID},
	)
	return h
}
