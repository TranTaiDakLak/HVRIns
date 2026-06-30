package igspc

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/klauspost/compress/zstd"

	igproxy "HVRIns/internal/proxy"
)

// ── Endpoint + constants từ capture V3 ──────────────────────────────────────────

const (
	igHost     = "https://i.instagram.com"
	graphqlURL = igHost + "/graphql_www"
	ssoURL     = igHost + "/api/v1/fxcal/get_sso_accounts/"

	appIDUsername = "com.bloks.www.bloks.caa.reg.username.async"
	appIDAcOptin  = "com.bloks.www.bloks.caa.reg.ac_optin.async"
	appIDCreate   = "com.bloks.www.bloks.caa.reg.create.account.async"

	bloksVersionID = "2530c58174d063584f25e249151d5bc7c53db138cfc68b554daa78c6cd7356b0"
	clientDocIDCre = "356548512614739681018024088968"
	ig4AAppID      = "567067343352427"
	uaCaptured     = "Instagram 421.0.0.51.66 Android (35/15; 450dpi; 1080x2400; samsung; SM-G996B; t2s; exynos2100; en_GB; 909555893)"
	deviceID       = "android-0619f0ab0c5dba42"
	familyDeviceID = "e8600531-590b-45f8-a617-0aa9146520bd"
	qeDeviceID     = "1f3b7429-d663-442a-9dba-463b15e23384"
	machineIDXMid  = "aj45lQABAAFnR8kPOXtsuhl75Xlb"
	metaUSDID      = "1576d6ca-ade9-4aeb-aeab-c3690bd90dba.1782727884.MEUCIQDtLK-A_7MAg-ys-Qdl2pWW__ZNLJ_Lc73jVVoB5_biKAIgEfrksaJyH_e6ivhamevGogJ_3dYFjvlFvM_YMf5juuo"
	metaZCA        = "eyJhbmRyb2lkIjp7ImFrYSI6eyJkYXRhVG9TaWduIjoie1widGltZVwiOlwiMTc4MjcyNDI4MDA5OVwiLFwiaGFzaFwiOlwiZHRMeXgtd0ZZVkhzU2R3Mkl2ay1RQkhlRnVyR2pydm5BeGpVaV9fdWpSRVwifSIsInNpZ25lZERhdGEiOiJNRVFDSUhsRXNTeFZscnpVc0pxRlB3YTM2UDA1OFdYUUc3d2kyck5JSU5jVzU1MTVBaUFKR3NvSTF6N29uaXpmenFrRTA0TTlMS2lTeGxoenQzSEZkY1VnWFdmZWNnIiwia2V5SGFzaCI6IjExNWJmNDQwNzE4MTI2NmYzYzE2ZGEzMTI1YWFiNTA2M2YxM2U2MWEyY2E3MmRkYzE5MWVhMTcxN2JhZmMyMWEiLCJsYXN0VXBsb2FkZWRLZXlUaW1lTXMiOjB9LCJncGlhIjp7InRva2VuIjoiIn0sInBheWxvYWQiOnsicGx1Z2lucyI6eyJiYXQiOnsic3RhIjoiRnVsbCIsImx2bCI6MTAwfSwic2N0Ijp7fSwiYWRiIjp7InVzYiI6MSwiYWRiIjoxLCJ1c2JfYWRiIjoxfX19fX0"
)

// ── HTTP session (TLS Safari iOS — giống igcore) ────────────────────────────────

type session struct {
	c tls_client.HttpClient
}

func newSession(proxyStr string) (*session, error) {
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(60),
		tls_client.WithClientProfile(profiles.Safari_IOS_15_6),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		tls_client.WithInsecureSkipVerify(),
		tls_client.WithNotFollowRedirects(),
	}
	if proxyStr != "" {
		if f := igproxy.FormatProxyURL(proxyStr); f != "" {
			opts = append(opts, tls_client.WithProxyUrl(f))
		}
	}
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return nil, err
	}
	return &session{c: c}, nil
}

func (s *session) close() {
	if s.c != nil {
		s.c.CloseIdleConnections() // tránh leak conn + HTTP/2 readLoop goroutine
	}
}

// post gửi POST với header order, decode zstd/gzip. Trả body + response header.
func (s *session) post(ctx context.Context, urlStr, body string, headers [][2]string) (string, fhttp.Header, error) {
	req, err := fhttp.NewRequestWithContext(ctx, "POST", urlStr, strings.NewReader(body))
	if err != nil {
		return "", nil, err
	}
	order := make([]string, 0, len(headers))
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
		order = append(order, kv[0])
	}
	req.Header[fhttp.HeaderOrderKey] = order

	resp, err := s.c.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return "", resp.Header, err
	}
	dec := decodeBody(resp.Header.Get("Content-Encoding"), raw)
	if resp.StatusCode >= 400 {
		return dec, resp.Header, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return dec, resp.Header, nil
}

func decodeBody(enc string, raw []byte) string {
	switch strings.ToLower(strings.TrimSpace(enc)) {
	case "gzip":
		if zr, err := gzip.NewReader(strings.NewReader(string(raw))); err == nil {
			defer zr.Close()
			if b, err := io.ReadAll(zr); err == nil {
				return string(b)
			}
		}
	case "zstd":
		if zd, err := zstd.NewReader(nil); err == nil {
			defer zd.Close()
			if b, err := zd.DecodeAll(raw, nil); err == nil {
				return string(b)
			}
		}
	}
	// IG hay trả zstd thiếu header → thử zstd anyway
	if zd, err := zstd.NewReader(nil); err == nil {
		defer zd.Close()
		if b, err := zd.DecodeAll(raw, nil); err == nil && len(b) > 0 {
			return string(b)
		}
	}
	return string(raw)
}

// ── Headers ─────────────────────────────────────────────────────────────────────

func commonIGHeaders(pigeonSID, connUUID string) [][2]string {
	return [][2]string{
		{"host", "i.instagram.com"},
		{"accept-language", "en-GB, en-US"},
		{"x-bloks-is-layout-rtl", "false"},
		{"x-bloks-version-id", bloksVersionID},
		{"x-fb-client-ip", "True"},
		{"x-fb-connection-type", "WIFI"},
		{"x-fb-server-cluster", "True"},
		{"x-ig-android-id", deviceID},
		{"x-ig-app-id", ig4AAppID},
		{"x-ig-app-locale", "en_GB"},
		{"x-ig-capabilities", "3brTv10="},
		{"x-ig-connection-type", "WIFI"},
		{"x-ig-device-id", qeDeviceID},
		{"x-ig-device-locale", "en_GB"},
		{"x-ig-family-device-id", familyDeviceID},
		{"x-ig-is-foldable", "false"},
		{"x-ig-mapped-locale", "en_GB"},
		{"x-ig-timezone-offset", "25200"},
		{"x-mid", machineIDXMid},
		{"x-meta-usdid", metaUSDID},
		{"x-meta-zca", metaZCA},
		{"x-pigeon-rawclienttime", nowTimestamp()},
		{"x-pigeon-session-id", pigeonSID},
		{"x-tigon-is-retry", "False"},
		{"accept-encoding", "zstd"},
		{"user-agent", uaCaptured},
		{"x-fb-conn-uuid-client", connUUID},
		{"x-fb-http-engine", "Tigon/MNS/TCP"},
		{"x-fb-rmd", "state=URL_ELIGIBLE"},
	}
}

func bloksHeaders(appID string) [][2]string {
	h := commonIGHeaders(newPigeonSession(), newConnUUID())
	return append(h,
		[2]string{"content-type", "application/x-www-form-urlencoded"},
		[2]string{"x-fb-friendly-name", "IGBloksAppRootQuery-" + appID},
		[2]string{"x-client-doc-id", clientDocIDCre},
		[2]string{"x-root-field-name", "bloks_action"},
		[2]string{"x-graphql-client-library", "pando"},
		[2]string{"x-graphql-request-purpose", "fetch"},
		[2]string{"x-ig-validate-null-in-legacy-dict", "true"},
		[2]string{"priority", "u=3, i"},
	)
}

// ── Steps ───────────────────────────────────────────────────────────────────────

type stepOutcome struct {
	ok      bool
	preview string
}

func previewN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Step A — get_sso_accounts: verify parent token còn live.
func callGetSSO(ctx context.Context, s *session, parent Parent, pigeonSID, connUUID string) stepOutcome {
	bearerFull := ensureBearerPrefix(parent.Bearer)
	tokens := fmt.Sprintf(
		`[{"account_type":"Instagram","token_id":0,"token_str":"%s","user_fbid":"%s","token_type":"first_party","token_app":"Instagram","token_source":"active_account"}]`,
		bearerFull, parent.UID)
	body := fmt.Sprintf(
		`{"surface":"account_switcher","tokens":%q,"_uid":"%s","device_id":"%s","_uuid":"%s","include_social_context":"false"}`,
		tokens, parent.UID, deviceID, qeDeviceID)
	form := url.Values{"signed_body": {"SIGNATURE." + body}}.Encode()

	headers := commonIGHeaders(pigeonSID, connUUID)
	headers = append(headers,
		[2]string{"authorization", bearerFull},
		[2]string{"ig-intended-user-id", parent.UID},
		[2]string{"ig-u-ds-user-id", parent.UID},
		[2]string{"x-fb-friendly-name", "IgApi: fxcal/get_sso_accounts/"},
		[2]string{"content-type", "application/x-www-form-urlencoded; charset=UTF-8"},
	)
	respBody, _, err := s.post(ctx, ssoURL, form, headers)
	return stepOutcome{ok: err == nil, preview: previewN(respBody, 400)}
}

// callBloks — Step D (username) và Step E (ac_optin): clone body verbatim + replace token.
func callBloks(ctx context.Context, s *session, tpl, appID string, parent Parent, waterfallID, username, birthday, email string) stepOutcome {
	body := replaceBodyTokens(tpl, parent, waterfallID, username, birthday, email)
	respBody, _, err := s.post(ctx, graphqlURL, body, bloksHeaders(appID))
	return stepOutcome{ok: err == nil, preview: previewN(respBody, 800)}
}

// callCreate — Step F: tạo account thật. Trả body + header (để parse child creds).
func callCreate(ctx context.Context, s *session, parent Parent, waterfallID, username, birthday, email string) (string, fhttp.Header, error) {
	body := replaceBodyTokens(tplCreate, parent, waterfallID, username, birthday, email)
	return s.post(ctx, graphqlURL, body, bloksHeaders(appIDCreate))
}

// ensureBearerPrefix đảm bảo dạng "Bearer IGT:2:..." cho header authorization.
func ensureBearerPrefix(b string) string {
	b = strings.TrimSpace(b)
	if strings.HasPrefix(b, "Bearer ") {
		return b
	}
	return "Bearer " + b
}
