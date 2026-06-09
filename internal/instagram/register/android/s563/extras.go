// extras.go â€” S563 post-register side effects + response parsing.
// Port 1:1 tá»« register/s557/extras.go â€” Ä‘á»•i profile type sang S563Profile,
// log tags â†’ [s563], connUUID â†’ base64 (gá»i connUUID() tá»« http.go).
package s563

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/fhttp/cookiejar"
	ftls "github.com/bogdanfinn/utls"
	"github.com/google/uuid"

	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/instagram/fakeinfo/uabuilder"
	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

// â”€â”€â”€ Constants â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func pickWarmUA() string {
	if ua := fakeinfo.RandomUAFromPool(fakeinfo.UAKindWebChrome); ua != "" {
		return ua
	}
	if res, err := (&uabuilder.BrowserUABuilder{}).Build(uabuilder.UAOptions{
		PoolKind: "reg",}); err == nil {
		return res.UserAgent
	}
	return "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Mobile Safari/537.36"
}

const defaultMetaZcaBlob = "eyJhbmRyb2lkIjp7ImFrYSI6eyJkYXRhVG9TaWduIjoiIiwiZXJyb3JzIjpbIktFWVNUT1JFX0RJU0FCTEVEX0JZX0NPTkZJRyJdfSwiZ3BpYSI6eyJ0b2tlbiI6IiIsImVycm9ycyI6WyJQTEFZX0lOVEVHUklUWV9ESVNBQkxFRF9CWV9DT05GSUciXX19fQ"

const (
	logoutFriendlyName              = "logout"
	fetchLoginDataBatchFriendlyName = "fetchLoginData-batch"
)

// â”€â”€â”€ Web session (m.facebook.com) â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

type webSession struct {
	client   *fhttp.Client
	jar      *cookiejar.Jar
	finalURL string
}

func newWebSession(proxyStr string) (*webSession, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create cookie jar: %w", err)
	}
	transport := &fhttp.Transport{
		TLSClientConfig:     &ftls.Config{InsecureSkipVerify: true},
		ForceAttemptHTTP2:   false,
		MaxIdleConnsPerHost: 2,
		IdleConnTimeout:     15 * time.Second,
	}
	if proxyStr != "" {
		if pURL := proxy.FormatProxyURL(proxyStr); pURL != "" {
			if u, err := url.Parse(pURL); err == nil {
				transport.Proxy = fhttp.ProxyURL(u)
			}
		}
	}
	c := &fhttp.Client{Jar: jar, Transport: transport, Timeout: 12 * time.Second}
	return &webSession{client: c, jar: jar}, nil
}

func (w *webSession) get(ctx context.Context, targetURL string, headers [][2]string) (string, error) {
	req, err := fhttp.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", err
	}
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.Request != nil && resp.Request.URL != nil {
		w.finalURL = resp.Request.URL.String()
	}
	data, _ := httpx.ReadBody(resp.Body, 1<<20)
	return string(data), nil
}

func (w *webSession) post(ctx context.Context, targetURL, body string, headers [][2]string) (string, error) {
	req, err := fhttp.NewRequestWithContext(ctx, "POST", targetURL, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
	}
	resp, err := w.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.Request != nil && resp.Request.URL != nil {
		w.finalURL = resp.Request.URL.String()
	}
	data, _ := httpx.ReadBody(resp.Body, 1<<20)
	return string(data), nil
}

func (w *webSession) addCookie(name, value string) {
	u, _ := url.Parse("https://m.facebook.com")
	w.jar.SetCookies(u, []*fhttp.Cookie{
		{Name: name, Value: value, Path: "/", Domain: ".facebook.com"},
	})
}

func (w *webSession) seedCookies(cookieStr string) {
	for _, pair := range strings.Split(cookieStr, ";") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) != 2 {
			continue
		}
		name := strings.TrimSpace(kv[0])
		if name == "c_user" || name == "xs" || name == "locale" {
			continue
		}
		w.addCookie(name, strings.TrimSpace(kv[1]))
	}
}

type warmTokens struct {
	versioningID string
	fbDtsg       string
	lsd          string
	hsi          string
	jazoest      string
	spinR        string
}

func parseWarmTokens(html string) warmTokens {
	t := warmTokens{}
	t.versioningID = wReFind(html, `versioningID:"(.*?)"`, 1)
	if t.versioningID == "" {
		t.versioningID = "3d5c3de42fc5b6024ad0c5b11df14b7e394d65431ca0c4e41086cd5297527e18"
	}
	t.fbDtsg = wReFind(html, `dtsg":{"token":"(.*?)",`, 1)
	if t.fbDtsg == "" {
		t.fbDtsg = wReFind(html, `"token":"([^"]+)"`, 1)
	}
	t.lsd = wReFind(html, `LSD".*?token":"(.*?)"`, 1)
	t.hsi = wReFind(html, `hsi":"(.*?)",`, 1)
	t.jazoest = wReFind(html, `name="jazoest" value="(\d+)"`, 1)
	if t.jazoest == "" {
		t.jazoest = wReFind(html, `jazoest", "(\d+)",`, 1)
	}
	t.spinR = wReFind(html, `__spin_r":(.*?),`, 1)
	return t
}

func warmNavHeaders(referer string) [][2]string {
	h := [][2]string{
		{"Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
		{"upgrade-insecure-requests", "1"},
		{"sec-fetch-site", "none"},
		{"sec-fetch-mode", "navigate"},
		{"sec-fetch-dest", "document"},
	}
	if referer != "" {
		h = append(h, [2]string{"referer", referer})
	}
	h = append(h,
		[2]string{"accept-language", "en-US,en;q=0.9"},
		[2]string{"User-Agent", pickWarmUA()},
	)
	return h
}

func warmPostHeaders(referer string) [][2]string {
	h := [][2]string{
		{"accept", "*/*"},
		{"sec-fetch-site", "same-origin"},
		{"sec-fetch-mode", "cors"},
		{"sec-fetch-dest", "empty"},
	}
	if referer != "" {
		h = append(h, [2]string{"referer", referer})
	}
	h = append(h,
		[2]string{"accept-language", "en-US,en;q=0.9"},
		[2]string{"User-Agent", pickWarmUA()},
		[2]string{"Content-Type", "application/x-www-form-urlencoded"},
	)
	return h
}

func loginWarmNewUI(ctx context.Context, w *webSession, uid, password string, tokens warmTokens) bool {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	waterfallID := uuid.New().String()

	body := fmt.Sprintf(
		"__aaid=0&__user=0&__a=1&__req=4"+
			"&__hs=20350.BP%%3Awbloks_caa_pkg.2.0...0"+
			"&dpr=3&__ccg=GOOD"+
			"&__rev=%s&__s=&__hsi=%s&__dyn=&locale=en_US"+
			"&fb_dtsg=%s&jazoest=%s&lsd=%s"+
			"&params=%%7B%%22params%%22%%3A%%22%%7B%%5C%%22server_params%%5C%%22%%3A%%7B%%5C%%22credential_type%%5C%%22%%3A%%5C%%22password%%5C%%22%%2C%%5C%%22waterfall_id%%5C%%22%%3A%%5C%%22%s%%5C%%22%%2C%%5C%%22access_flow_version%%5C%%22%%3A%%5C%%22pre_mt_behavior%%5C%%22%%2C%%5C%%22is_from_logged_in_switcher%%5C%%22%%3A0%%2C%%5C%%22is_from_logged_out%%5C%%22%%3A0%%7D%%2C%%5C%%22client_input_params%%5C%%22%%3A%%7B%%5C%%22contact_point%%5C%%22%%3A%%5C%%22%s%%5C%%22%%2C%%5C%%22password%%5C%%22%%3A%%5C%%22%%23PWD_BROWSER%%3A0%%3A%s%%3A%s%%5C%%22%%2C%%5C%%22machine_id%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22accounts_list%%5C%%22%%3A%%5B%%5D%%7D%%7D%%22%%7D",
		tokens.spinR, tokens.hsi,
		tokens.fbDtsg, tokens.jazoest, tokens.lsd,
		waterfallID,
		uid, ts, password,
	)

	loginURL := fmt.Sprintf(
		"https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.login.async.send_login_request&type=action&__bkv=%s",
		tokens.versioningID,
	)

	respBody, err := w.post(ctx, loginURL, body, warmPostHeaders("https://m.facebook.com/?locale=en_US"))
	if err != nil || respBody == "" {
		return false
	}
	if !strings.Contains(respBody, "com.bloks.www.caa.login.save-credentials") ||
		!strings.Contains(respBody, "currentUser") {
		return false
	}

	time.Sleep(1000 * time.Millisecond)
	dtsg2 := wReFind(respBody, `dtsgToken":"(.*?)"`, 1)
	jazoest2 := wReFind(respBody, `sprinkleValue":"(\d+)"`, 1)
	enc2 := wReFind(respBody, `encrypted":"(.*?)"`, 1)
	loginUID := wReFind(respBody, `currentUser":(\d+),`, 1)

	nonceBody := fmt.Sprintf(
		"client_event_flow=&fb_dtsg=%s&jazoest=%s&lsd=%s"+
			"&__dyn=&__csr=&__req=a&__fmt=1&__a=%s&__user=%s&__wma=1",
		dtsg2, jazoest2, tokens.lsd, enc2, loginUID,
	)
	nonceH := warmPostHeaders("https://m.facebook.com/login/save-device/")
	nonceH = append(nonceH,
		[2]string{"x-response-format", "JSONStream"},
		[2]string{"x-requested-with", "XMLHttpRequest"},
		[2]string{"x-fb-lsd", tokens.lsd},
		[2]string{"x-asbd-id", "359341"},
	)
	nonceResp, _ := w.post(ctx, "https://m.facebook.com/login/device-based/update-nonce/async/", nonceBody, nonceH)
	if strings.Contains(nonceResp, "success\":true") {
		time.Sleep(1000 * time.Millisecond)
		w.get(ctx, "https://m.facebook.com?deoia=1", warmNavHeaders("https://m.facebook.com/login/save-device/"))
	}
	return true
}

func loginWarmOldUI(ctx context.Context, w *webSession, uid, password string, tokens warmTokens) bool {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	encpass := fmt.Sprintf("%%23PWD_BROWSER%%3A0%%3A%s%%3A%s", ts, password)

	body := fmt.Sprintf(
		"m_ts=%s&li=&try_number=0&unrecognized_tries=0"+
			"&email=%s&prefill_contact_point=%s"+
			"&prefill_source=&prefill_type=password"+
			"&first_prefill_source=&first_prefill_type=contact_point"+
			"&had_cp_prefilled=true&had_password_prefilled=true"+
			"&is_smart_lock=false&bi_xrwh=0"+
			"&encpass=%s"+
			"&fb_dtsg=%s&jazoest=%s&lsd=%s"+
			"&__dyn=&__csr=&__req=4&__fmt=1&__a=%s&__user=0",
		tokens.spinR,
		uid, uid,
		encpass,
		tokens.fbDtsg, tokens.jazoest, tokens.lsd,
		tokens.spinR,
	)

	postH := warmPostHeaders("https://m.facebook.com/?locale=en_US")
	postH = append(postH,
		[2]string{"x-response-format", "JSONStream"},
		[2]string{"x-requested-with", "XMLHttpRequest"},
		[2]string{"x-fb-lsd", tokens.lsd},
		[2]string{"x-asbd-id", "359341"},
	)

	respBody, err := w.post(ctx,
		"https://m.facebook.com/login/device-based/login/async/?refsrc=deprecated&lwv=100",
		body, postH)
	if err != nil || respBody == "" {
		return false
	}

	unescaped := wUnescape(wUnescape(respBody))
	if !strings.Contains(unescaped, "save-device") {
		return false
	}

	time.Sleep(1000 * time.Millisecond)
	saveHTML, err := w.get(ctx,
		"https://m.facebook.com/login/save-device/?login_source=login",
		warmNavHeaders("https://m.facebook.com/?locale=en_US"))
	if err != nil || saveHTML == "" || strings.Contains(w.finalURL, "checkpoint") {
		return false
	}

	dtsg2 := wReFind(saveHTML, `dtsg":{"token":"(.*?)",`, 1)
	jazoest2 := wReFind(saveHTML, `name="jazoest" value="(\d+)"`, 1)
	nonceBody := fmt.Sprintf("fb_dtsg=%s&jazoest=%s&flow=interstitial_nux&next=&nux_source=regular_login", dtsg2, jazoest2)
	time.Sleep(1000 * time.Millisecond)
	w.post(ctx,
		"https://m.facebook.com/login/device-based/update-nonce/",
		nonceBody,
		warmNavHeaders("https://m.facebook.com/login/save-device/?login_source=login&soft=hjk"))
	return true
}

func logoutWarm(ctx context.Context, w *webSession) {
	urlCf := "https://m.facebook.com/confirmemail.php?next=https%3A%2F%2Fm.facebook.com%2F%3Fdeoia%3D1&soft=hjk"
	html, err := w.get(ctx, urlCf, warmNavHeaders(""))
	if err != nil || html == "" {
		return
	}
	hash := wReFind(html, `logout\.php[^"]*h=([^"&]+)`, 1)
	if hash == "" {
		hash = wReFind(html, `loggedOutHash:"(.*?)"`, 1)
	}
	if hash == "" {
		return
	}
	w.get(ctx, "https://m.facebook.com/logout.php?h="+hash, warmNavHeaders(urlCf))
}

func copyWarmCookies(w *webSession, s *session) int {
	skip := map[string]bool{"c_user": true, "xs": true, "locale": true}
	n := 0
	seen := map[string]bool{}
	for _, rawURL := range []string{"https://m.facebook.com", "https://facebook.com"} {
		u, _ := url.Parse(rawURL)
		for _, c := range w.jar.Cookies(u) {
			if !skip[c.Name] && c.Value != "" && !seen[c.Name] {
				seen[c.Name] = true
				s.addCookie(c.Name, c.Value)
				n++
			}
		}
	}
	return n
}

func warmSession(ctx context.Context, s *session, seed Seed, proxyStr string, notify func(string)) bool {
	if seed.Mode != SeedModeInitialAccount || seed.UID == "" || seed.Password == "" {
		return false
	}

	notify(fmt.Sprintf("[s563] Warm session â€” login %s...", safeShort(seed.UID, 8)))

	ws, err := newWebSession(proxyStr)
	if err != nil {
		notify(fmt.Sprintf("[s563] Warm: create session failed: %v", err))
		return false
	}
	defer ws.client.CloseIdleConnections()

	if seed.CookieString != "" {
		ws.seedCookies(seed.CookieString)
	} else if seed.Datr != "" {
		ws.addCookie("datr", seed.Datr)
	}

	homeHTML, err := ws.get(ctx, "https://m.facebook.com/?locale=en_US", warmNavHeaders(""))
	if err != nil || homeHTML == "" {
		notify("[s563] Warm: GET home failed")
		return false
	}
	tokens := parseWarmTokens(homeHTML)
	if tokens.fbDtsg == "" {
		regHTML, _ := ws.get(ctx, "https://m.facebook.com/reg/?locale=en_US",
			warmNavHeaders("https://m.facebook.com/?locale=en_US"))
		if regHTML != "" {
			tokens = parseWarmTokens(regHTML)
		}
	}

	loginOK := loginWarmNewUI(ctx, ws, seed.UID, seed.Password, tokens)
	if !loginOK {
		loginOK = loginWarmOldUI(ctx, ws, seed.UID, seed.Password, tokens)
	}
	if !loginOK {
		notify("[s563] Warm: login failed â€” reg without warm")
		return false
	}

	notify("[s563] Warm: login OK â†’ logout...")
	logoutWarm(ctx, ws)

	n := copyWarmCookies(ws, s)
	notify(fmt.Sprintf("[s563] Warm: done â€” %d cookies transferred", n))
	return true
}

func wReFind(s, pattern string, group int) string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}
	m := re.FindStringSubmatch(s)
	if len(m) > group {
		return m[group]
	}
	return ""
}

func wUnescape(s string) string {
	re := regexp.MustCompile(`\\(.)`)
	return re.ReplaceAllStringFunc(s, func(m string) string {
		if len(m) < 2 {
			return m
		}
		switch m[1] {
		case 'n':
			return "\n"
		case 'r':
			return "\r"
		case 't':
			return "\t"
		case '\\':
			return "\\"
		default:
			return string(m[1])
		}
	})
}

// â”€â”€â”€ LogoutAccount â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func LogoutAccount(ctx context.Context, sess *session, profile S563Profile, accessToken, deviceID string) {
	locale := profile.Locale
	if locale == "" {
		locale = "en_US"
	}
	cc := profile.Sim.CountryCode
	if cc == "" {
		cc = "US"
	}

	body := "reason=USER_INITIATED" +
		"&device_id=" + deviceID +
		"&retain_for_dbl=false" +
		"&logout_source=REGISTRATION" +
		"&locale=" + locale +
		"&client_country_code=" + cc +
		"&fb_api_req_friendly_name=" + logoutFriendlyName +
		"&fb_api_caller_class=Fb4aLogoutOperationsHelper"

	postURL := "https://b-graph.facebook.com/auth/expire_session"
	req, err := fhttp.NewRequestWithContext(ctx, "POST", postURL, bytes.NewReader([]byte(body)))
	if err != nil {
		return
	}

	h := buildLogoutHeaders(profile, accessToken, deviceID)
	for _, kv := range h {
		req.Header[kv[0]] = []string{kv[1]}
	}
	req.Header["content-type"] = []string{"application/x-www-form-urlencoded"}
	req.Header["content-length"] = []string{fmt.Sprintf("%d", len(body))}

	resp, err := sess.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
}

func buildLogoutHeaders(profile S563Profile, accessToken, deviceID string) [][2]string {
	analyticsTag := `{"network_tags":{"product":"350685531728","purpose":"fetch","request_category":"graphql","retry_attempt":"0"},"application_tags":"graphservice"}`

	return [][2]string{
		{"Authorization", "OAuth " + accessToken},
		{"X-Fb-Friendly-Name", logoutFriendlyName},
		{"X-Fb-Connection-Type", profile.ConnType},
		{"X-Fb-Sim-Hni", profile.Sim.HNI},
		{"X-Fb-Net-Hni", profile.Sim.HNI},
		{"X-Graphql-Client-Library", "graphservice"},
		{"X-Tigon-Is-Retry", "False"},
		{"X-Graphql-Request-Purpose", "fetch"},
		{"X-Fb-Request-Analytics-Tags", analyticsTag},
		{"X-Fb-Http-Engine", "Tigon/Liger"},
		{"X-Fb-Client-Ip", "True"},
		{"X-Fb-Server-Cluster", "True"},
		{"X-Fb-Device-Group", profile.DeviceGroup},
		{"X-Fb-Conn-Uuid-Client", connUUID()},
		{"X-Zero-F-Device-Id", profile.FamilyDeviceID},
		{"App-Scope-Id-Header", deviceID},
		{"X-Meta-Zca", defaultMetaZcaBlob},
		{"X-Fb-Connection-Quality", "EXCELLENT"},
	}
}

// â”€â”€â”€ fetchXZeroEh â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

var xzeroEhRegex = regexp.MustCompile(`eligibility_hash":"(.*?)"`)

func buildXZeroEhBody(profile S563Profile) string {
	mcc := profile.Sim.MCC
	mnc := profile.Sim.MNC
	conn := strings.ToLower(profile.ConnType)
	locale := profile.Locale
	cc := profile.Sim.CountryCode

	inner := fmt.Sprintf(
		"carrier_mcc=%s&carrier_mnc=%s&sim_mcc=%s&sim_mnc=%s&format=json"+
			"&interface=%s&dialtone_enabled=false&needs_backup_rules=true"+
			"&token_hash=&request_reason=login&eligibility_hash="+
			"&locale=%s&client_country_code=%s&fb_api_req_friendly_name=fetchZeroToken",
		mcc, mnc, mcc, mnc, conn, locale, cc,
	)

	batchJSON := fmt.Sprintf(
		`[{"method":"POST","body":"%s","name":"fetchZeroToken","omit_response_on_success":false,"relative_url":"mobile_zero_campaign"}]`,
		strings.ReplaceAll(inner, `"`, `\"`),
	)

	encoded := urlEncodeFull(batchJSON)

	return "batch=" + encoded +
		"&fb_api_caller_class=Fb4aAuthHandler" +
		"&fb_api_req_friendly_name=fetchLoginData-batch"
}

func urlEncodeFull(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.' || c == '~' {
			b.WriteByte(c)
			continue
		}
		fmt.Fprintf(&b, "%%%02X", c)
	}
	return b.String()
}

func fetchXZeroEh(ctx context.Context, sess *session, profile S563Profile, accessToken, deviceID string) string {
	locale := profile.Locale
	if locale == "" {
		locale = "en_US"
	}
	cc := profile.Sim.CountryCode
	if cc == "" {
		cc = "US"
	}

	postURL := fmt.Sprintf(
		"https://b-graph.facebook.com/?include_headers=false&decode_body_json=false&streamable_json_response=true&locale=%s&client_country_code=%s",
		locale, cc,
	)

	body := buildXZeroEhBody(profile)

	req, err := fhttp.NewRequestWithContext(ctx, "POST", postURL, bytes.NewReader([]byte(body)))
	if err != nil {
		return "unknown"
	}

	h := buildXZeroHeaders(profile, accessToken, deviceID)
	for _, kv := range h {
		req.Header[kv[0]] = []string{kv[1]}
	}
	req.Header["content-type"] = []string{"application/x-www-form-urlencoded"}
	req.Header["content-length"] = []string{fmt.Sprintf("%d", len(body))}

	resp, err := sess.client.Do(req)
	if err != nil {
		return "unknown"
	}
	defer resp.Body.Close()

	raw, _ := httpx.ReadBody(resp.Body, 256*1024)
	m := xzeroEhRegex.FindStringSubmatch(string(raw))
	if len(m) >= 2 && m[1] != "" {
		return m[1]
	}
	return "unknown"
}

func buildXZeroHeaders(profile S563Profile, accessToken, deviceID string) [][2]string {
	analyticsTag := `{"network_tags":{"product":"350685531728","purpose":"fetch","request_category":"graphql","retry_attempt":"0"},"application_tags":"graphservice"}`

	return [][2]string{
		{"Authorization", "OAuth " + accessToken},
		{"X-Fb-Friendly-Name", fetchLoginDataBatchFriendlyName},
		{"X-Fb-Connection-Type", profile.ConnType},
		{"X-Fb-Sim-Hni", profile.Sim.HNI},
		{"X-Fb-Net-Hni", profile.Sim.HNI},
		{"X-Graphql-Client-Library", "graphservice"},
		{"X-Tigon-Is-Retry", "False"},
		{"X-Graphql-Request-Purpose", "fetch"},
		{"X-Fb-Request-Analytics-Tags", analyticsTag},
		{"X-Fb-Http-Engine", "Tigon/Liger"},
		{"X-Fb-Client-Ip", "True"},
		{"X-Fb-Server-Cluster", "True"},
		{"X-Fb-Device-Group", profile.DeviceGroup},
		{"X-Fb-Conn-Uuid-Client", connUUID()},
		{"X-Zero-F-Device-Id", profile.FamilyDeviceID},
		{"App-Scope-Id-Header", deviceID},
		{"X-Meta-Zca", defaultMetaZcaBlob},
	}
}

// â”€â”€â”€ Register response parser â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

var (
	reUserAccessToken = regexp.MustCompile(`EAAAAU[A-Za-z0-9+/=_-]{20,}`)
	reAccessToken     = regexp.MustCompile(`EAA([A-Za-z0-9+/=_-]{10,})`)
	reCUser           = regexp.MustCompile(`c_user","value":"(\d{10,})"`)
	reXS              = regexp.MustCompile(`name":"xs","value":"([^"]+)"`)
	reFR              = regexp.MustCompile(`name":"fr","value":"([^"]+)"`)
	reDATR            = regexp.MustCompile(`name":"datr","value":"([^"]+)"`)
	reCreatedUID      = regexp.MustCompile(`created_user(?:id)?["\s:,\\]+(\d{10,})`)
	reBloksUID        = regexp.MustCompile(`currentUser["\s:,]+(\d{10,})`)
	reBloksToken      = regexp.MustCompile(`access_token["\s:,]+"(EAA[A-Za-z0-9+/=_-]{20,})"`)
	reSaveCredUID     = regexp.MustCompile(`SaveCredential[^}]*?(\d{10,18})`)
)

type regResponse struct {
	UID         string
	AccessToken string
	Cookie      string
	DATR        string
	Blocked     bool
}

func parseRegisterResponse(body, locale string) (*regResponse, error) {
	resp := &regResponse{}
	clean := strings.ReplaceAll(body, "\\", "")

	if strings.Contains(body, "couldn't create an account for you") ||
		strings.Contains(clean, "couldn't create an account for you") {
		resp.Blocked = true
		return resp, fmt.Errorf("Facebook blocked: account creation denied")
	}
	if strings.Contains(clean, "integrity_block") {
		resp.Blocked = true
		return resp, fmt.Errorf("Facebook blocked: integrity_block")
	}

	if m := reUserAccessToken.FindString(clean); m != "" {
		resp.AccessToken = m
	}
	if resp.AccessToken == "" {
		if m := reBloksToken.FindStringSubmatch(clean); len(m) > 1 {
			resp.AccessToken = m[1]
		}
	}
	if resp.AccessToken == "" {
		if m := reAccessToken.FindStringSubmatch(clean); len(m) > 1 {
			resp.AccessToken = "EAA" + m[1]
		}
	}

	if m := reCUser.FindStringSubmatch(clean); len(m) > 1 {
		resp.UID = m[1]
	}
	if resp.UID == "" {
		if m := reBloksUID.FindStringSubmatch(clean); len(m) > 1 {
			resp.UID = m[1]
		}
	}
	if resp.UID == "" {
		if m := reCreatedUID.FindStringSubmatch(clean); len(m) > 1 {
			resp.UID = m[1]
		}
	}
	if resp.UID == "" {
		if m := reSaveCredUID.FindStringSubmatch(clean); len(m) > 1 {
			resp.UID = m[1]
		}
	}
	if resp.UID == "" {
		return resp, fmt.Errorf("no UID found in response")
	}

	var parts []string
	parts = append(parts, "c_user="+resp.UID)
	if m := reXS.FindStringSubmatch(clean); len(m) > 1 {
		parts = append(parts, "xs="+m[1])
	}
	if locale != "" {
		parts = append(parts, "locale="+locale)
	}
	if m := reFR.FindStringSubmatch(clean); len(m) > 1 {
		parts = append(parts, "fr="+m[1])
	}
	if m := reDATR.FindStringSubmatch(clean); len(m) > 1 {
		resp.DATR = m[1]
		parts = append(parts, "datr="+m[1])
	}
	resp.Cookie = strings.Join(parts, ";") + ";"
	return resp, nil
}
