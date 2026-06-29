// ios_graphql.go — iOS registration dùng graphql_www + Safari_IOS_15_6 TLS.
// Port từ IGDesktop — sử dụng template capture thật + placeholder replacement.
package igandroid

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/igcore"
	"HVRIns/internal/instagram"
	"HVRIns/internal/proxy"
	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
)

// ── Embed templates ────────────────────────────────────────────────────────

//go:embed templates/*.txt
var gqlTmplFS embed.FS

func gqlLoadTemplate(name string) (string, error) {
	b, err := gqlTmplFS.ReadFile("templates/" + name)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ── Constants ──────────────────────────────────────────────────────────────

const (
	gqlAppID       = "124024574287414"
	gqlBloksVer    = "bbabb80f3de1e25c0c5c0bbfd6cae893124649276a36ead33328a4ea03a34b75"
	gqlGraphqlPath = "https://i.instagram.com/graphql_www"
	)

// ── Profile ────────────────────────────────────────────────────────────────

type iosGQLProfile struct {
	DeviceID       string
	FamilyDeviceID string
	WaterfallID    string
	MachineID      string
	RegMachineID   string
	CloudTrustID   string
	PigeonSID      string
	ConnUUID       string
	RegFlowID      string
	UserAgent      string
	Locale         string
}

func newIOSGQLProfile() *iosGQLProfile {
	return &iosGQLProfile{
		DeviceID:       gqlUpperUUID(),
		FamilyDeviceID: gqlUpperUUID(),
		WaterfallID:    gqlHex32(),
		RegMachineID:   gqlRandBase64URL(24),
		CloudTrustID:   strings.ToUpper(uuid.New().String()) + strings.ToUpper(uuid.New().String()),
		PigeonSID:      "UFS-" + strings.ToUpper(uuid.New().String()) + "-1",
		ConnUUID:       gqlHex32(),
		RegFlowID:      uuid.New().String(),
		UserAgent:      randomIOSGQLUA(),
		Locale:         "en_US",
	}
}

// randomIOSGQLUA sinh UA ngẫu nhiên nhưng chỉ dùng iOS 15.x — khớp TLS Safari_IOS_15_6.
// iOS 16/17/18 với TLS fingerprint iOS 15.6 sẽ lộ inconsistency.
func randomIOSGQLUA() string {
	versions := []struct{ AppVer, Build string }{
		{"410.1.0.36.70", "849447290"},
		{"407.0.0.24.99", "843437295"},
		{"405.0.0.24.95", "839437284"},
	}
	devices := []struct{ Model, Screen, Scale string }{
		{"iPhone14,8", "1284x2778", "3.00"}, // iPhone 13 Pro Max
		{"iPhone13,4", "1284x2778", "3.00"}, // iPhone 12 Pro Max
		{"iPhone13,2", "1170x2532", "3.00"}, // iPhone 12
		{"iPhone12,1", "828x1792", "2.00"},  // iPhone 11
		{"iPhone11,8", "828x1792", "2.00"},  // iPhone XR
		{"iPhone9,1", "750x1334", "2.00"},   // iPhone 7
	}
	// Chỉ iOS 15.x — khớp với Safari_IOS_15_6 TLS fingerprint
	systems := []string{"15_8_4", "15_8_3", "15_8_2", "15_7_9", "15_7_8"}

	v := versions[mathRandN(len(versions))]
	d := devices[mathRandN(len(devices))]
	s := systems[mathRandN(len(systems))]
	return fmt.Sprintf(
		"Instagram %s (%s; iOS %s; en_US; en; scale=%s; %s; %s) AppleWebKit/420+",
		v.AppVer, d.Model, s, d.Scale, d.Screen, v.Build,
	)
}

func mathRandN(n int) int {
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	return int(b[0]) % n
}

func gqlUpperUUID() string { return strings.ToUpper(uuid.New().String()) }

func gqlHex32() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func gqlRandBase64URL(n int) string {
	const al = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = al[int(b[i])%len(al)]
	}
	return string(b)
}

func gqlNowUnix() int64 { return time.Now().Unix() }

func gqlNewAAC() (jid, ccs string) {
	return uuid.New().String(), gqlRandBase64URL(43)
}

func gqlGenUSDID(p *iosGQLProfile) string {
	id := p.DeviceID
	ts := fmt.Sprintf("%d", gqlNowUnix())
	payload := id + "." + ts
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return payload + ".err"
	}
	h := sha256.Sum256([]byte(payload))
	sig, err := ecdsa.SignASN1(rand.Reader, key, h[:])
	if err != nil {
		return payload + ".err"
	}
	return payload + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// ── Headers ────────────────────────────────────────────────────────────────

func gqlLocaleToAcceptLang(locale string) string {
	lang := strings.ReplaceAll(locale, "_", "-")
	code := locale[:2]
	if code == "en" {
		return lang + ";q=1.0"
	}
	return lang + ";q=1.0," + code + ";q=0.9,en-US;q=0.8"
}

func gqlLocaleToTimezone(locale string) string {
	switch locale[:2] {
	case "vi", "th", "id", "ms":
		return "25200"
	case "en":
		if strings.Contains(locale, "GB") {
			return "0"
		}
		return "-18000"
	default:
		return "25200"
	}
}

func gqlBloksHeaders(p *iosGQLProfile, friendlyAppID string) [][2]string {
	friendly := "IGBloksAppRootQuery-" + friendlyAppID
	analyticsTags := fmt.Sprintf(`{"network_tags":{"product":"%s","surface":"other","is_ad":"0","request_category":"api","purpose":"fetch","retry_attempt":"0"},"application_tags":{"is_nav_critical":"0"}}`, gqlAppID)
	locale := p.Locale
	if locale == "" {
		locale = "vi_VN"
	}
	lang := locale[:2]
	localeHyphen := strings.ReplaceAll(locale, "_", "-")

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
		{"x-meta-usdid", gqlGenUSDID(p)},
		{"x-ig-app-id", gqlAppID},
		{"x-mid", p.MachineID},
		{"x-ig-bandwidth-speed-kbps", "0.000"},
		{"x-cloud-trust-token", p.CloudTrustID},
		{"x-pigeon-session-id", p.PigeonSID},
		{"x-pigeon-rawclienttime", fmt.Sprintf("%d.000000", gqlNowUnix())},
		{"x-ig-bloks-serialize-payload", "true"},
		{"accept-language", gqlLocaleToAcceptLang(locale)},
		{"x-ig-timezone-offset", gqlLocaleToTimezone(locale)},
		{"x-fb-connection-type", "wifi"},
		{"x-ig-device-id", p.DeviceID},
		{"x-ig-family-device-id", p.FamilyDeviceID},
		{"ig-intended-user-id", "0"},
		{"x-ig-connection-type", "WiFi"},
		{"x-bloks-version-id", gqlBloksVer},
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

// ── Template placeholders ──────────────────────────────────────────────────

const (
	gqlCapDeviceID     = "780EF930-6A7E-4536-B0A7-491368931CE3"
	gqlCapFamilyDevID  = "37312B3B-7404-491B-967D-72AE026CE3A7"
	gqlCapWaterfallID  = "fa6c3900254c4faa9a6d32aa0accd5cd"
	gqlCapCloudTrust   = "4A3B0992-83A9-4EA5-BDF0-C2600A6E3828295E487A-B465-4F17-8F3E-2C6B4DF775FC"
	gqlCapRegFlowID    = "122925d4-0d65-47ea-8176-fda7e1b75b5f"
	gqlCapRegMachineID = "rKwaarWTOxLfrbI_ZPZ-31HW"
	gqlCapAacJID       = "1f2a44bd-3898-40c9-ae2f-5f9e5ed65fe4"
	gqlCapAacCS        = "iyKrsg4qm822sg0hbZGLN6PmxIWqSLa0Zxx8JbUM5-w"
	gqlCapEmail        = "quanvucong2k4%40gmail.com"
	gqlCapEmailPlain   = "quanvucong2k4@gmail.com"
	gqlCapEmailUnicode = "quanvucong2k4%5C%5C%5C%5C%5C%5C%5C%5Cu0040gmail.com"
	gqlCapEventReqID   = "076dd760-2f0e-4571-97ed-50ad62a6dc46"
	gqlCapCode         = "479123"
	gqlCapName         = "Koiu677"
	gqlCapUsername     = "quanvucong2k4"
	gqlCapBirthday     = "28-03-2000"
	gqlCapBirthdayTS   = "954235554"
	gqlCapConfirmToken = "zhveisAD"

	gqlCapEncPwdPrefix     = `encrypted_password%5C%5C%5C%22%3A%5C%5C%5C%22`
	gqlCapEncPwdClose      = `%5C%5C%5C%22%2C%5C%5C%5C%22username`
	gqlRegCtxOpen          = `reg_context%5C%5C%5C%22%3A%5C%5C%5C%22`
	gqlRegCtxClose         = `%5C%5C%5C%22`
	gqlAclOpen             = `accounts_list%5C%5C%5C%22%3A%5B`
	gqlAclClose            = `%5D%2C%5C%5C%5C%22fb_ig_device_id`
	gqlAclCloseConfirm     = `%5D%2C%5C%5C%5C%22network_bssid`
	gqlCapRegContextCreate = `reg_context%5C%5C%5C%22%3A%5C%5C%5C%22`
)

func gqlApplyProfile(body string, p *iosGQLProfile) string {
	locale := p.Locale
	if locale == "" {
		locale = "en_US"
	}
	return strings.NewReplacer(
		gqlCapDeviceID, p.DeviceID,
		gqlCapFamilyDevID, p.FamilyDeviceID,
		gqlCapWaterfallID, p.WaterfallID,
		gqlCapCloudTrust, p.CloudTrustID,
		gqlCapRegFlowID, p.RegFlowID,
		gqlCapRegMachineID, p.RegMachineID,
		"vi_VN", locale,
	).Replace(body)
}

func gqlSetEmail(body, email string) string {
	enc := url.QueryEscape(email)
	var unicodeNew string
	if at := strings.LastIndex(email, "@"); at >= 0 {
		unicodeNew = email[:at] + "%5C%5C%5C%5C%5C%5C%5C%5Cu0040" + email[at+1:]
	}
	return strings.NewReplacer(
		gqlCapEmail, enc,
		gqlCapEmailPlain, email,
		gqlCapEmailUnicode, unicodeNew,
	).Replace(body)
}

// gqlEventReqIDRe matches event_request_id":"<UUID> in triple-escaped URL-encoded template bodies.
// The constant gqlCapEventReqID only matches 04_submit_email.txt; other templates have different UUIDs.
var gqlEventReqIDRe = regexp.MustCompile(
	`(event_request_id%5C%5C%5C%22%3A%5C%5C%5C%22)` +
		`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`,
)

func gqlSetEventReqID(body, id string) string {
	if replaced := gqlEventReqIDRe.ReplaceAllString(body, "${1}"+id); replaced != body {
		return replaced
	}
	// Fallback: submit_email uses a different URL-encoding depth
	return strings.ReplaceAll(body, gqlCapEventReqID, id)
}

func gqlSetAAC(body, jid, ccs string) string {
	return strings.NewReplacer(gqlCapAacJID, jid, gqlCapAacCS, ccs).Replace(body)
}

func gqlSetCode(body, code string) string { return strings.ReplaceAll(body, gqlCapCode, code) }
func gqlSetName(body, name string) string { return strings.ReplaceAll(body, gqlCapName, name) }
func gqlSetUsername(body, u string) string {
	return strings.ReplaceAll(body, gqlCapUsername, u)
}

func gqlSetBirthday(body, ddmmyyyy string, ts int64) string {
	return strings.NewReplacer(gqlCapBirthday, ddmmyyyy, gqlCapBirthdayTS, fmt.Sprintf("%d", ts)).Replace(body)
}

func gqlSetConfirmationCode(body, token string) string {
	if token == "" {
		return body
	}
	return strings.ReplaceAll(body, gqlCapConfirmToken, token)
}

func gqlSetEncryptedPassword(body, encPwd string) string {
	i := strings.Index(body, gqlCapEncPwdPrefix)
	if i < 0 {
		return body
	}
	start := i + len(gqlCapEncPwdPrefix)
	j := strings.Index(body[start:], gqlCapEncPwdClose)
	if j < 0 {
		return body
	}
	return body[:start] + url.QueryEscape(encPwd) + body[start+j:]
}

func gqlStripAccountsList(body string) string {
	i := strings.Index(body, gqlAclOpen)
	if i < 0 {
		return body
	}
	start := i + len(gqlAclOpen)
	j := strings.Index(body[start:], gqlAclClose)
	if j < 0 {
		return body
	}
	return body[:start] + body[start+j:]
}

func gqlStripAccountsListConfirm(body string) string {
	i := strings.Index(body, gqlAclOpen)
	if i < 0 {
		return body
	}
	start := i + len(gqlAclOpen)
	j := strings.Index(body[start:], gqlAclCloseConfirm)
	if j < 0 {
		return body
	}
	return body[:start] + body[start+j:]
}

func gqlSetRegContextRaw(body, regCtxEncoded string) string {
	i := strings.Index(body, gqlRegCtxOpen)
	if i < 0 {
		return body
	}
	start := i + len(gqlRegCtxOpen)
	j := strings.Index(body[start:], gqlRegCtxClose)
	if j < 0 {
		return body
	}
	return body[:start] + regCtxEncoded + body[start+j:]
}

func gqlSetRegContextCreate(body, regCtx string) string {
	i := strings.Index(body, gqlCapRegContextCreate)
	if i < 0 {
		return body
	}
	start := i + len(gqlCapRegContextCreate)
	seg := body[start:]
	endAnchor := "%7Cregm"
	j := strings.Index(seg, endAnchor)
	if j < 0 {
		endAnchor = "%7cregm"
		j = strings.Index(seg, endAnchor)
	}
	if j < 0 {
		return body
	}
	return body[:start] + url.QueryEscape(regCtx) + body[start+j+len(endAnchor):]
}

// ── Response helpers ───────────────────────────────────────────────────────

var gqlReRegContext = regexp.MustCompile(`reg_context"[^"]*"([A-Za-z0-9_\-]{30,}\|?[a-z]*)"`)

func gqlParseRegContext(resp string) string {
	clean := strings.ReplaceAll(resp, `\`, "")
	if m := regexp.MustCompile(`([A-Za-z0-9_\-]{200,}\|regm)`).FindStringSubmatch(clean); len(m) > 1 {
		return m[1]
	}
	if m := gqlReRegContext.FindStringSubmatch(clean); len(m) > 1 {
		return m[1]
	}
	return ""
}

func gqlHasMarker(resp, marker string) bool {
	return strings.Contains(strings.ToLower(strings.ReplaceAll(resp, `\`, "")), strings.ToLower(marker))
}

func gqlExtractError(resp string) string {
	clean := strings.ReplaceAll(resp, `\\`, `\`)
	if m := regexp.MustCompile(`error_message[\\"]*\s*[\\"]*([^\\"]{4,200})`).FindStringSubmatch(clean); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func gqlShortErr(resp string) string {
	if m := gqlExtractError(resp); m != "" {
		return m
	}
	if len(resp) > 200 {
		return resp[:200]
	}
	return resp
}

func gqlIsBlocked(resp string) bool {
	return gqlHasMarker(resp, "integrity_block") ||
		(gqlHasMarker(resp, "rate") && gqlHasMarker(resp, "limit"))
}

func gqlExtractUID(resp string) string {
	clean := strings.ReplaceAll(resp, `\`, "")
	for _, re := range []*regexp.Regexp{
		regexp.MustCompile(`\(eud (\d{8,})\)`),
		regexp.MustCompile(`c_user","value":"(\d{8,})"`),
		regexp.MustCompile(`"pk":(\d{8,})`),
	} {
		if m := re.FindStringSubmatch(clean); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}

// ── TLS Session ────────────────────────────────────────────────────────────

type iosGQLSession struct {
	client tls_client.HttpClient
	zr     *zstd.Decoder
}

func newIOSGQLSession(proxyStr string) (*iosGQLSession, error) {
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(60),
		tls_client.WithClientProfile(profiles.Safari_IOS_15_6),
		tls_client.WithCookieJar(tls_client.NewCookieJar()),
		tls_client.WithInsecureSkipVerify(),
		tls_client.WithNotFollowRedirects(),
	}
	if proxyStr != "" {
		if f := proxy.FormatProxyURL(proxyStr); f != "" {
			opts = append(opts, tls_client.WithProxyUrl(f))
		}
	}
	c, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return nil, fmt.Errorf("create ios15 tls client: %w", err)
	}
	zr := sharedZstdDecoder // dùng chung — tránh leak goroutine decoder mỗi session
	return &iosGQLSession{client: c, zr: zr}, nil
}

func (s *iosGQLSession) decode(enc string, raw []byte) string {
	if strings.EqualFold(enc, "zstd") && s.zr != nil {
		if err := s.zr.Reset(strings.NewReader(string(raw))); err == nil {
			if dec, err := io.ReadAll(s.zr); err == nil && len(dec) > 0 {
				return string(dec)
			}
		}
	}
	return string(raw)
}

func (s *iosGQLSession) post(ctx context.Context, urlStr, body string, headers [][2]string) (resp string, hdr fhttp.Header, err error) {
	req, err2 := fhttp.NewRequestWithContext(ctx, "POST", urlStr, strings.NewReader(body))
	if err2 != nil {
		return "", nil, err2
	}
	order := make([]string, 0, len(headers))
	for _, h := range headers {
		req.Header.Set(h[0], h[1])
		order = append(order, h[0])
	}
	req.Header[fhttp.HeaderOrderKey] = order

	r, err2 := s.client.Do(req)
	if err2 != nil {
		return "", nil, err2
	}
	defer r.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	return s.decode(r.Header.Get("Content-Encoding"), raw), r.Header, nil
}

func (s *iosGQLSession) rotateProxy(proxyStr string) {
	if proxyStr == "" {
		return
	}
	rotated := igcore.RotateSession(proxyStr)
	if f := proxy.FormatProxyURL(rotated); f != "" {
		_ = s.client.SetProxy(f)
		s.client.CloseIdleConnections()
	}
}

// qeSync dùng chính iOS session (Safari_IOS_15_6) để lấy keyID, pubKey, X-MID.
// Fingerprint nhất quán với các bước reg sau đó — không dùng Android session.
func (s *iosGQLSession) qeSync(ctx context.Context, p *iosGQLProfile) (keyID, pubKey, xmid string, err error) {
	const qeSyncURL = "https://i.instagram.com/api/v1/qe/sync/"
	body := "id=" + p.DeviceID + "&experiments=ig_android_device_detection_info_upload"
	headers := [][2]string{
		{"user-agent", p.UserAgent},
		{"accept-encoding", "zstd"},
		{"accept", "*/*"},
		{"x-ig-app-id", gqlAppID},
		{"x-ig-capabilities", "36r/F/8="},
		{"x-ig-device-id", p.DeviceID},
		{"x-ig-family-device-id", p.FamilyDeviceID},
		{"x-ig-timezone-offset", gqlLocaleToTimezone(p.Locale)},
		{"content-type", "application/x-www-form-urlencoded; charset=UTF-8"},
	}
	req, err2 := fhttp.NewRequestWithContext(ctx, "POST", qeSyncURL, strings.NewReader(body))
	if err2 != nil {
		return "", "", "", err2
	}
	order := make([]string, 0, len(headers))
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
		order = append(order, kv[0])
	}
	req.Header[fhttp.HeaderOrderKey] = order
	resp, err2 := s.client.Do(req)
	if err2 != nil {
		return "", "", "", err2
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	s.decode(resp.Header.Get("Content-Encoding"), raw)

	keyID = resp.Header.Get("ig-set-password-encryption-key-id")
	pubKey = resp.Header.Get("ig-set-password-encryption-pub-key")
	xmid = resp.Header.Get("ig-set-x-mid")
	if keyID == "" || pubKey == "" {
		return "", "", xmid, fmt.Errorf("qe/sync iOS: missing key headers (HTTP %d)", resp.StatusCode)
	}
	return keyID, pubKey, xmid, nil
}

// checkProxyIPCountry detect country code của IP qua proxy, dùng ip-api.com.
// Trả ("", "") nếu không detect được.
func (s *iosGQLSession) checkProxyIPCountry(ctx context.Context) (ip, country string) {
	req, err := fhttp.NewRequestWithContext(ctx, "GET",
		"http://ip-api.com/json/?fields=query,countryCode", nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	r, err := s.client.Do(req)
	if err != nil {
		return "", ""
	}
	defer r.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
	body := string(raw)
	if m := regexp.MustCompile(`"query"\s*:\s*"([0-9.a-f:]+)"`).FindStringSubmatch(body); len(m) > 1 {
		ip = m[1]
	}
	if m := regexp.MustCompile(`"countryCode"\s*:\s*"([A-Z]{2})"`).FindStringSubmatch(body); len(m) > 1 {
		country = m[1]
	}
	return ip, country
}

// injectAgedDevice set cookie datr/mid/ig_did aged vào cookie jar của iOS session.
// Giả lập thiết bị có lịch sử — override mid từ qe/sync bằng mid aged.
func (s *iosGQLSession) injectAgedDevice(p *iosGQLProfile, dev *igcore.AgedDevice) {
	if dev == nil {
		return
	}
	if dev.Mid != "" {
		p.MachineID = dev.Mid
	}
	u, err := url.Parse("https://i.instagram.com")
	if err != nil {
		return
	}
	var cookies []*fhttp.Cookie
	if dev.Datr != "" {
		cookies = append(cookies, &fhttp.Cookie{Name: "datr", Value: dev.Datr, Domain: ".instagram.com", Path: "/"})
	}
	if dev.Mid != "" {
		cookies = append(cookies, &fhttp.Cookie{Name: "mid", Value: dev.Mid, Domain: ".instagram.com", Path: "/"})
	}
	if dev.IgDID != "" {
		cookies = append(cookies, &fhttp.Cookie{Name: "ig_did", Value: dev.IgDID, Domain: ".instagram.com", Path: "/"})
	}
	if len(cookies) > 0 {
		s.client.SetCookies(u, cookies)
	}
}

// ── Engine ─────────────────────────────────────────────────────────────────

var (
	errGQLThrottled      = errors.New("THROTTLED")
	errGQLDomainRejected = errors.New("EMAIL_DOMAIN_REJECTED")
	errGQLSessionFlagged = errors.New("SESSION_FLAGGED") // system_error — session poisoned, start fresh
)

type iosGQLEngine struct {
	sess     *iosGQLSession
	p        *iosGQLProfile
	log      func(string, ...any)
	keyID    string
	pubKey   string
	proxyStr string

	regContext       string
	confirmationCode string
	Session          igcore.IGSession
}

func (e *iosGQLEngine) aymh(ctx context.Context) error {
	body, err := gqlLoadTemplate("01_aymh.txt")
	if err != nil {
		return err
	}
	body = gqlApplyProfile(body, e.p)
	jid, ccs := gqlNewAAC()
	body = gqlSetAAC(body, jid, ccs)

	resp, hdr, _ := e.sess.post(ctx, gqlGraphqlPath, body,
		gqlBloksHeaders(e.p, "com.bloks.www.bloks.caa.reg.aymh_create_account_button.async"))
	if m := hdr.Get("ig-set-x-mid"); m != "" {
		e.p.MachineID = m
	}
	if rc := gqlParseRegContext(resp); rc != "" {
		e.regContext = rc
		e.log("  aymh → reg_context %d chars", len(rc))
	}
	if gqlIsBlocked(resp) {
		return fmt.Errorf("aymh blocked: %s", gqlShortErr(resp))
	}
	if !gqlHasMarker(resp, "contactpoint") && !gqlHasMarker(resp, "bloks_payload") {
		return fmt.Errorf("aymh bad response (%d bytes): %s", len(resp), gqlShortErr(resp))
	}
	return nil
}

func (e *iosGQLEngine) renderScreen(ctx context.Context, tmpl, appID string) {
	body, err := gqlLoadTemplate(tmpl)
	if err != nil {
		return
	}
	body = gqlApplyProfile(body, e.p)
	jid, ccs := gqlNewAAC()
	body = gqlSetAAC(body, jid, ccs)
	resp, hdr, _ := e.sess.post(ctx, gqlGraphqlPath, body, gqlBloksHeaders(e.p, appID))
	if m := hdr.Get("ig-set-x-mid"); m != "" {
		e.p.MachineID = m
	}
	if rc := gqlParseRegContext(resp); rc != "" {
		e.regContext = rc
	}
	e.log("  render %s (%d bytes)", appID, len(resp))
}

func (e *iosGQLEngine) submitEmail(ctx context.Context, addr string) error {
	body, err := gqlLoadTemplate("04_submit_email.txt")
	if err != nil {
		return err
	}
	body = gqlApplyProfile(body, e.p)
	body = gqlStripAccountsList(body)
	body = gqlSetEmail(body, addr)
	body = gqlSetEventReqID(body, uuid.New().String())
	jid, ccs := gqlNewAAC()
	body = gqlSetAAC(body, jid, ccs)

	resp, hdr, err := e.sess.post(ctx, gqlGraphqlPath, body,
		gqlBloksHeaders(e.p, "com.bloks.www.bloks.caa.reg.async.contactpoint_email.async"))
	if err != nil {
		e.log("  submitEmail HTTP err: %v", err)
	}
	if m := hdr.Get("ig-set-x-mid"); m != "" {
		e.p.MachineID = m
	}
	if rc := gqlParseRegContext(resp); rc != "" {
		e.regContext = rc
		e.log("  submitEmail → reg_context %d chars", len(rc))
	}
	if gqlIsBlocked(resp) {
		return fmt.Errorf("blocked: %s", gqlShortErr(resp))
	}
	if gqlHasMarker(resp, "confirmation") || gqlHasMarker(resp, "BloksCAARegConfirmation") {
		e.log("  submitEmail OK (%d bytes)", len(resp))
		return nil
	}
	if gqlHasMarker(resp, "login_upsell") || gqlHasMarker(resp, "existing_profile") {
		return fmt.Errorf("email đã tồn tại")
	}
	if gqlHasMarker(resp, "THROTTLING_REQUEST_GLOBAL") || gqlHasMarker(resp, "throttling request") {
		return errGQLThrottled
	}
	if gqlHasMarker(resp, "USER_REGISTER_INVALID_EMAIL") {
		return errGQLThrottled
	}
	return fmt.Errorf("submitEmail failed (%d bytes): %s", len(resp), gqlShortErr(resp))
}

func (e *iosGQLEngine) confirmOTP(ctx context.Context, addr, code string) error {
	body, err := gqlLoadTemplate("05_confirm.txt")
	if err != nil {
		return err
	}
	body = gqlApplyProfile(body, e.p)
	body = gqlStripAccountsListConfirm(body)
	body = gqlSetEmail(body, addr)
	body = gqlSetCode(body, code)
	body = gqlSetEventReqID(body, uuid.New().String())
	jid, ccs := gqlNewAAC()
	body = gqlSetAAC(body, jid, ccs)
	if e.regContext != "" {
		body = gqlSetRegContextRaw(body, strings.ReplaceAll(e.regContext, "|", "%7C"))
		e.log("  confirm dùng reg_context %d chars", len(e.regContext))
	}

	resp, _, err := e.sess.post(ctx, gqlGraphqlPath, body,
		gqlBloksHeaders(e.p, "com.bloks.www.bloks.caa.reg.confirmation.async"))
	if err != nil {
		e.log("  confirmOTP HTTP err: %v", err)
	}
	if gqlHasMarker(resp, "THROTTLING_REQUEST_GLOBAL") {
		return errGQLThrottled
	}
	if gqlHasMarker(resp, "integrity_block") {
		return fmt.Errorf("integrity_block: %s", gqlShortErr(resp))
	}
	if gqlHasMarker(resp, "BloksCAARegPassword") || gqlHasMarker(resp, "reg.password") || gqlHasMarker(resp, "gen_next_screen") {
		if cc := igcore.ParseConfirmationCode(resp); cc != "" {
			e.confirmationCode = cc
			e.log("  confirmOTP → confirmation_code %s", cc)
		}
		e.log("  confirmOTP OK (%d bytes)", len(resp))
		return nil
	}
	if gqlHasMarker(resp, "confirmation code you entered is invalid") || gqlHasMarker(resp, "invalid or has expired") {
		return fmt.Errorf("OTP sai/hết hạn")
	}
	return fmt.Errorf("confirmOTP failed (%d bytes): %s", len(resp), gqlShortErr(resp))
}

func (e *iosGQLEngine) step(ctx context.Context, tmpl, appID, addr string, extra func(string) string) (string, error) {
	body, err := gqlLoadTemplate(tmpl)
	if err != nil {
		return "", err
	}
	body = gqlApplyProfile(body, e.p)
	body = gqlSetEmail(body, addr)
	body = gqlStripAccountsListConfirm(body)
	body = gqlSetEventReqID(body, uuid.New().String())
	jid, ccs := gqlNewAAC()
	body = gqlSetAAC(body, jid, ccs)
	if e.regContext != "" {
		body = gqlSetRegContextRaw(body, strings.ReplaceAll(url.QueryEscape(e.regContext), "+", "%20"))
		body = gqlSetRegContextCreate(body, e.regContext)
	}
	if extra != nil {
		body = extra(body)
	}
	resp, _, err := e.sess.post(ctx, gqlGraphqlPath, body, gqlBloksHeaders(e.p, appID))
	if err != nil {
		e.log("  %s HTTP err: %v", appID, err)
	}
	if rc := gqlParseRegContext(resp); rc != "" {
		e.regContext = rc
	}
	if gqlHasMarker(resp, "THROTTLING_REQUEST_GLOBAL") {
		return resp, errGQLThrottled
	}
	if gqlIsBlocked(resp) {
		return resp, fmt.Errorf("blocked: %s", gqlShortErr(resp))
	}
	return resp, nil
}

func (e *iosGQLEngine) setPassword(ctx context.Context, addr, password string) error {
	encPwd, err := igcore.EncryptPassword(password, e.pubKey, e.keyID)
	if err != nil {
		return fmt.Errorf("encrypt password: %w", err)
	}
	resp, err := e.step(ctx, "06_password.txt", "com.bloks.www.bloks.caa.reg.password.async", addr, func(b string) string {
		return gqlSetEncryptedPassword(b, encPwd)
	})
	if err != nil {
		return err
	}
	if gqlHasMarker(resp, "birthday") || gqlHasMarker(resp, "gen_next_screen") || len(resp) > 50000 {
		e.log("  setPassword OK (%d bytes)", len(resp))
		return nil
	}
	return fmt.Errorf("setPassword failed (%d bytes): %s", len(resp), gqlShortErr(resp))
}

func (e *iosGQLEngine) setBirthday(ctx context.Context, addr string) error {
	year := int64(1990) + gqlNowUnix()%15
	ts := (year-1970)*365*24*3600 + gqlNowUnix()%1000000
	ddmmyyyy := fmt.Sprintf("15-06-%d", year)
	resp, err := e.step(ctx, "07_birthday.txt", "com.bloks.www.bloks.caa.reg.birthday.async", addr, func(b string) string {
		return gqlSetBirthday(b, ddmmyyyy, ts)
	})
	if err != nil {
		return err
	}
	if gqlHasMarker(resp, "name") || gqlHasMarker(resp, "gen_next_screen") || len(resp) > 50000 {
		e.log("  setBirthday OK (%d bytes)", len(resp))
		return nil
	}
	return fmt.Errorf("setBirthday failed (%d bytes): %s", len(resp), gqlShortErr(resp))
}

func (e *iosGQLEngine) setNameIG(ctx context.Context, addr, name string) error {
	resp, err := e.step(ctx, "08_name.txt", "com.bloks.www.bloks.caa.reg.name_ig_and_soap.async", addr, func(b string) string {
		return gqlSetName(b, name)
	})
	if err != nil {
		return err
	}
	if gqlHasMarker(resp, "username") || gqlHasMarker(resp, "gen_next_screen") || len(resp) > 50000 {
		e.log("  setName OK (%d bytes)", len(resp))
		return nil
	}
	return fmt.Errorf("setName failed (%d bytes): %s", len(resp), gqlShortErr(resp))
}

func (e *iosGQLEngine) setUsername(ctx context.Context, addr, username string) error {
	resp, err := e.step(ctx, "09_username.txt", "com.bloks.www.bloks.caa.reg.username.async", addr, func(b string) string {
		return gqlSetUsername(b, username)
	})
	if err != nil {
		return err
	}
	if gqlHasMarker(resp, "create") || gqlHasMarker(resp, "tos") || gqlHasMarker(resp, "gen_next_screen") || len(resp) > 50000 {
		e.log("  setUsername OK (%d bytes)", len(resp))
		return nil
	}
	return fmt.Errorf("setUsername failed (%d bytes): %s", len(resp), gqlShortErr(resp))
}

func (e *iosGQLEngine) acceptTOS(ctx context.Context, addr string) error {
	resp, err := e.step(ctx, "09_username.txt", "com.bloks.www.bloks.caa.reg.tos", addr, nil)
	if err != nil {
		e.log("  acceptTOS warn: %v — tiếp tục", err)
		return nil
	}
	e.log("  acceptTOS OK (%d bytes)", len(resp))
	return nil
}

func (e *iosGQLEngine) createAccount(ctx context.Context, addr, username, name string) error {
	const maxCreate = 15
	for i := 1; i <= maxCreate; i++ {
		resp, err := e.stepCreate(ctx, addr, username, name, i)
		if errors.Is(err, errGQLThrottled) {
			e.log("  createAccount %d: throttle → retry", i)
			gqlBackoff(i)
			continue
		}
		if errors.Is(err, errGQLDomainRejected) {
			return err
		}
		if err != nil {
			if strings.Contains(err.Error(), "blocked") {
				e.log("  createAccount %d: %v → retry", i, err)
				e.sess.rotateProxy(e.proxyStr)
				gqlBackoff(i)
				continue
			}
			return err
		}
		uid := gqlExtractUID(resp)
		if uid != "" {
			e.Session = igcore.ParseIGSession(resp)
			e.log("  createAccount SUCCESS — UID: %s (%d bytes)", uid, len(resp))
			return nil
		}
		if gqlHasMarker(resp, "create_success") || gqlHasMarker(resp, "sessionless_login_on_completion") {
			e.Session = igcore.ParseIGSession(resp)
			e.log("  createAccount SUCCESS (%d bytes)", len(resp))
			return nil
		}
		if gqlHasMarker(resp, "system_error") {
			idx := strings.Index(resp, "system_error")
			start := idx - 80
			if start < 0 {
				start = 0
			}
			end := idx + 120
			if end > len(resp) {
				end = len(resp)
			}
			e.log("  createAccount %d: system_error (session flagged — abort) ...%s...", i, resp[start:end])
			return errGQLSessionFlagged
		}
		if gqlHasMarker(resp, "integrity_block") {
			e.log("  createAccount %d: integrity_block → rotate IP + retry", i)
			e.sess.rotateProxy(e.proxyStr)
			gqlBackoff(i)
			continue
		}
		return fmt.Errorf("createAccount attempt %d unknown result (%d bytes): %s", i, len(resp), gqlShortErr(resp))
	}
	return fmt.Errorf("createAccount failed after %d attempts", maxCreate)
}

func (e *iosGQLEngine) stepCreate(ctx context.Context, addr, username, name string, attempt int) (string, error) {
	appID := "com.bloks.www.bloks.caa.reg.create.account.async"
	body, err := gqlLoadTemplate("10_create.txt")
	if err != nil {
		return "", err
	}
	body = gqlApplyProfile(body, e.p)
	body = gqlSetEmail(body, addr)
	body = gqlStripAccountsListConfirm(body)
	body = gqlSetEventReqID(body, uuid.New().String())
	jid, ccs := gqlNewAAC()
	body = gqlSetAAC(body, jid, ccs)
	if e.regContext != "" {
		body = gqlSetRegContextRaw(body, strings.ReplaceAll(url.QueryEscape(e.regContext), "+", "%20"))
		body = gqlSetRegContextCreate(body, e.regContext)
	}
	body = gqlSetUsername(body, username)
	body = gqlSetName(body, name)
	if e.confirmationCode != "" {
		body = gqlSetConfirmationCode(body, e.confirmationCode)
	}

	resp, _, err := e.sess.post(ctx, gqlGraphqlPath, body, gqlBloksHeaders(e.p, appID))
	if err != nil {
		e.log("  create attempt %d HTTP err: %v", attempt, err)
	}
	if rc := gqlParseRegContext(resp); rc != "" {
		e.regContext = rc
	}
	if gqlHasMarker(resp, "user_input_error") && gqlHasMarker(resp, "create_failure") {
		e.log("  create attempt %d: EMAIL DOMAIN REJECTED", attempt)
		return resp, errGQLDomainRejected
	}
	if gqlHasMarker(resp, "THROTTLING_REQUEST_GLOBAL") {
		return resp, errGQLThrottled
	}
	if gqlIsBlocked(resp) && !gqlHasMarker(resp, "create_failure") {
		return resp, fmt.Errorf("blocked: %s", gqlShortErr(resp))
	}
	return resp, nil
}

func gqlHumanDelay(ctx context.Context, minSec, maxSec int) {
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	d := time.Duration(minSec)*time.Second + time.Duration(int(b[0])%((maxSec-minSec)*1000))*time.Millisecond
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func gqlBackoff(i int) {
	// Exponential backoff: 5s, 10s, 15s, 20s, 25s, 30s capped — giúp proxy pool cycle IPs
	d := time.Duration(i*5) * time.Second
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	time.Sleep(d)
}

// ── Registerer ─────────────────────────────────────────────────────────────

type iosGQLRegisterer struct{}

func (r *iosGQLRegisterer) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	// agedDev: mid mượn từ pool — gắn nhãn [mid:...] vào mọi log của luồng này.
	var agedDev *igcore.AgedDevice
	midTag := func() string {
		if agedDev != nil {
			return "[mid:" + agedDev.Mid + "] "
		}
		return ""
	}

	status := func(msg string) {
		if onStatus != nil {
			onStatus(midTag() + msg)
		}
	}

	if strings.TrimSpace(input.Email) == "" || input.GetOTP == nil {
		return &instagram.RegResult{Success: false, Message: "igiosgql: cần Email + GetOTP"}
	}

	// addr và getOTP mutable — cập nhật khi retry với email mới sau SESSION_FLAGGED.
	addr := strings.TrimSpace(input.Email)
	getOTP := input.GetOTP

	fail := func(stage, msg string) *instagram.RegResult {
		return &instagram.RegResult{
			Success: false,
			Email:   addr,
			Message: fmt.Sprintf("%s[igiosgql/%s] %s", midTag(), stage, msg),
		}
	}

	password := input.Password
	if password == "" {
		password = buildPassword()
	}
	name := buildName(input)
	logf := func(f string, a ...any) { status(fmt.Sprintf(f, a...)) }

	// rebuildSession tạo iOS session mới + qe/sync lại — dùng khi rotate IP sau THROT.
	type sessionState struct {
		sess     *iosGQLSession
		p        *iosGQLProfile
		eng      *iosGQLEngine
		proxyStr string
	}
	buildSession := func(proxyStr string) (*sessionState, error) {
		rotated := igcore.RotateSession(proxyStr)
		sess, err := newIOSGQLSession(rotated)
		if err != nil {
			return nil, fmt.Errorf("session: %w", err)
		}

		ipCtx, ipCancel := context.WithTimeout(ctx, 8*time.Second)
		curIP, _ := sess.checkProxyIPCountry(ipCtx)
		ipCancel()
		if curIP != "" {
			logf("  IP: %s", curIP)
		}

		p := newIOSGQLProfile()

		var keyID, pubKey, xmid string
		// qe/sync retry 5 lần
		for qs := 1; qs <= 5; qs++ {
			keyID, pubKey, xmid, err = sess.qeSync(ctx, p)
			if err == nil {
				break
			}
			logf("  qeSync lần %d lỗi: %v", qs, err)
			time.Sleep(time.Duration(qs) * time.Second)
		}
		if err != nil {
			return nil, fmt.Errorf("qeSync: %w", err)
		}
		if xmid != "" {
			p.MachineID = xmid
		}

		// Inject aged device từ pool (giống IGDesktop UseDevicePool)
		deviceSrc := "fresh"
		if igcore.SharedDevicePool != nil {
			if dev := igcore.SharedDevicePool.Next(); dev != nil {
				sess.injectAgedDevice(p, dev)
				agedDev = dev // gắn nhãn [mid:...] vào log luồng + giữ aged mid
				deviceSrc = "pool"
				logf("  device aged injected")
			}
		}
		logf("  device_src=%s ua=%s", deviceSrc, p.UserAgent)

		eng := &iosGQLEngine{
			sess:     sess,
			p:        p,
			keyID:    keyID,
			pubKey:   pubKey,
			proxyStr: rotated,
			log:      logf,
		}
		return &sessionState{sess: sess, p: p, eng: eng, proxyStr: rotated}, nil
	}

	// ── Outer retry loop: mỗi lần SESSION_FLAGGED → email mới + session mới ──
	const maxOuter = 6
	for outerAttempt := 1; outerAttempt <= maxOuter; outerAttempt++ {
		if outerAttempt > 1 {
			if input.GetNewEmail == nil {
				return fail("createAccount", "SESSION_FLAGGED: không có GetNewEmail callback")
			}
			logf("🔄 SESSION_FLAGGED (lần %d/%d) — chờ 8s rồi tạo email mới...", outerAttempt, maxOuter)
			select {
			case <-ctx.Done():
				return fail("createAccount", "context cancelled khi chờ retry")
			case <-time.After(8 * time.Second):
			}
			newAddr, newGetOTP, freshErr := input.GetNewEmail(ctx)
			if freshErr != nil {
				return fail("createAccount", "GetNewEmail lỗi: "+freshErr.Error())
			}
			addr = newAddr
			getOTP = newGetOTP
			logf("📧 Email mới: %s", addr)
		}

	// ── Khởi tạo session ban đầu ───────────────────────────────────────────
	status("qe/sync")
	st, err := buildSession(input.Proxy)
	if err != nil {
		return fail("qeSync", err.Error())
	}
	// Close session cuối khi reg xong (tránh leak conn + readLoop goroutine HTTP/2).
	defer func() {
		if st != nil && st.sess != nil {
			st.sess.client.CloseIdleConnections()
		}
	}()
	status("ua:" + st.p.UserAgent)

	// ── aymh ───────────────────────────────────────────────────────────────
	status("aymh")
	if err := st.eng.aymh(ctx); err != nil {
		logf("  aymh warn: %v — tiếp tục", err)
	}
	st.eng.renderScreen(ctx, "03_contactpoint_email.txt", "com.bloks.www.bloks.caa.reg.nta.interstitial.async")

	// ── submitEmail: retry 15 lần, mỗi THROT → IP mới + session mới ──────
	status("submitEmail")
	const maxSubmit = 15
	var submitOK bool
	for attempt := 1; attempt <= maxSubmit; attempt++ {
		if attempt > 1 {
			logf("  submitEmail retry %d/%d — rotate IP...", attempt, maxSubmit)
			if ns, e2 := buildSession(input.Proxy); e2 == nil {
				st.sess.client.CloseIdleConnections() // close session CŨ trước khi thay
				st = ns
				// aymh lại với session mới
				_ = st.eng.aymh(ctx)
				st.eng.renderScreen(ctx, "03_contactpoint_email.txt", "com.bloks.www.bloks.caa.reg.nta.interstitial.async")
			}
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		if err = st.eng.submitEmail(ctx, addr); err == nil {
			submitOK = true
			break
		}
		if !errors.Is(err, errGQLThrottled) {
			return fail("submitEmail", err.Error())
		}
	}
	if !submitOK {
		return fail("submitEmail", fmt.Sprintf("throttled sau %d lần", maxSubmit))
	}

	// ── readOTP ────────────────────────────────────────────────────────────
	status("readOTP")
	otp, err := getOTP(ctx)
	if err != nil || otp == "" {
		msg := "GetOTP failed"
		if err != nil {
			msg = err.Error()
		}
		return fail("readOTP", msg)
	}

	// ── confirmOTP: retry 5 lần nếu THROT ─────────────────────────────────
	status("confirmOTP")
	for c := 1; c <= 5; c++ {
		err = st.eng.confirmOTP(ctx, addr, otp)
		if err == nil {
			break
		}
		if !errors.Is(err, errGQLThrottled) {
			return fail("confirmOTP", err.Error())
		}
		logf("  confirmOTP THROT lần %d — retry", c)
		time.Sleep(time.Duration(c) * time.Second)
	}
	if err != nil {
		return fail("confirmOTP", err.Error())
	}

	// ── setPassword ────────────────────────────────────────────────────────
	gqlHumanDelay(ctx, 3, 8)
	status("setPassword")
	if err := st.eng.setPassword(ctx, addr, password); err != nil {
		return fail("setPassword", err.Error())
	}

	// ── setBirthday ────────────────────────────────────────────────────────
	gqlHumanDelay(ctx, 4, 10)
	status("setBirthday")
	if err := st.eng.setBirthday(ctx, addr); err != nil {
		logf("  setBirthday warn: %v — tiếp tục", err)
	}

	// ── setName ────────────────────────────────────────────────────────────
	gqlHumanDelay(ctx, 3, 8)
	status("setName")
	if err := st.eng.setNameIG(ctx, addr, name); err != nil {
		return fail("setName", err.Error())
	}

	// ── setUsername (retry đổi username khi trùng/không hợp lệ) ──────────────
	gqlHumanDelay(ctx, 3, 8)
	var username string
	var unameErr error
	for ut := 1; ut <= 4; ut++ {
		username = buildUsername()
		status("setUsername:" + username)
		unameErr = st.eng.setUsername(ctx, addr, username)
		if unameErr == nil {
			break
		}
		status(fmt.Sprintf("setUsername trùng/lỗi (try %d/4) → đổi username", ut))
		gqlHumanDelay(ctx, 1, 3)
	}
	if unameErr != nil {
		return fail("setUsername", unameErr.Error())
	}

	// ── acceptTOS + createAccount ──────────────────────────────────────────
	gqlHumanDelay(ctx, 5, 15)
	_ = st.eng.acceptTOS(ctx, addr)
	status("createAccount")
	if err := st.eng.createAccount(ctx, addr, username, name); err != nil {
		if errors.Is(err, errGQLSessionFlagged) || errors.Is(err, errGQLDomainRejected) {
			continue // outer loop: retry với email mới
		}
		return fail("createAccount", err.Error())
	}

	igSession := st.eng.Session

	// ── Harvest device → pool cho lần reg sau (chung pool iOS với ig_ios_bloks).
	if igcore.SharedDevicePool != nil && igSession.SessionID != "" {
		if added := igcore.SharedDevicePool.Add(igSession.Mid, igSession.Datr, igSession.IgDID); added {
			logf("harvest device mid=%.8s… → pool", igSession.Mid)
		}
	}

	// KHÔNG checklive ở đây (block slot ~15s). createAccount OK = reg OK.
	// Live/Die check ASYNC ở app_register → slot nhả ngay.
	logf("reg OK (checklive async)")
	return &instagram.RegResult{
		UID:        igSession.UID,
		Username:   username,
		Password:   password,
		Cookie:     igSession.FullCookie,
		Email:      addr,
		UserAgent:  st.p.UserAgent,
		LiveStatus: "",
		Success:    true,
		Message:    "ok",
	}
	} // end outer retry loop
	return fail("createAccount", fmt.Sprintf("SESSION_FLAGGED — thất bại sau %d lần thử", maxOuter))
}

// ── Init ───────────────────────────────────────────────────────────────────

func init() {
	instagram.RegisterPlatformRegisterer("ig_ios_gql", func() instagram.Registerer {
		return &iosGQLRegisterer{}
	})
}
