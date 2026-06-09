// extras.go — iOS HTTP (MFB) seed parsing + initial login/logout warm + response parsers.
//
// File này gộp 3 file cũ:
//   - seed.go           → Seed + ParseSeed (3 modes: datr/full_cookie/initial_account)
//   - initial_login.go  → loginInitialNewUI + loginInitialOldUI + logoutInitial
//   - parser.go         → pageTokens + parsePageTokens + UID/cookie/datr extractors
//
// PORT từ C#: FacebookRegisterMfbRequest LoginFbOldUI/NewUI/LogoutFb + BuildFromChromeAndroidPage.
package ioshttp

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/instagram/fakeinfo"
)

// ─── Seed model (Cookie Initial) ─────────────────────────────────────────────
//
// Maps to C#: GetPerfectMachineId → MachineId → RegisterWithRequestInitialAccount.
//
// Three seed modes matching C# source_type behavior:
//
//	SeedModeDatrOnly       (source_type=1) — just a datr value, no login initial
//	SeedModeFullCookie     (source_type=3) — full cookie string, seed cookies
//	SeedModeInitialAccount (source_type=3) — uid|password, login initial + logout
//
// C# behavior by mode:
//
//	DatrOnly:        MachineId = "BYB0a..." → no "|" → login initial SKIPPED
//	FullCookie:      MachineId = "datr=xxx;sb=yyy" → no "|" → login initial SKIPPED, cookies seeded
//	InitialAccount:  MachineId = "uid|password" → has "|" → login initial RUNS

// SeedMode identifies how a cookie initial value should be used.
type SeedMode int

const (
	SeedModeNone           SeedMode = iota // no seed provided
	SeedModeDatrOnly                       // raw datr value (no "=" or "|")
	SeedModeFullCookie                     // cookie string containing "datr=" (may have "|" separating parts)
	SeedModeInitialAccount                 // "uid|password" format for login initial
)

// Seed is a parsed cookie initial value.
type Seed struct {
	Raw          string   // original input string
	Mode         SeedMode // parsed mode
	Datr         string   // extracted datr value (all modes except None)
	CookieString string   // full cookie string (FullCookie mode)
	UID          string   // account UID (InitialAccount mode)
	Password     string   // account password (InitialAccount mode)
	SourceLabel  string   // human-readable label for logging
}

// ParseSeed analyzes a raw cookie initial string and returns a structured Seed.
// Parse logic matches C# GetPerfectMachineId return values:
//
//	source_type=1 → datr value only (no pipes, no equals)
//	source_type=3 → full line "uid|pass|cookie|token|..."
func ParseSeed(raw string) Seed {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Seed{Mode: SeedModeNone, SourceLabel: "none"}
	}

	// Pipe-delimited format: "uid|password|cookie_string|token|..."
	if strings.Contains(raw, "|") {
		parts := strings.Split(raw, "|")

		// If first part looks like a UID (digits) and second is non-empty → InitialAccount
		if len(parts) >= 2 && len(parts[0]) > 0 && len(parts[1]) > 0 {
			s := Seed{
				Raw:         raw,
				Mode:        SeedModeInitialAccount,
				UID:         parts[0],
				Password:    parts[1],
				SourceLabel: "initial_account(uid=" + parts[0][:min(len(parts[0]), 8)] + "...)",
			}
			// Also extract datr from cookie part if available (parts[2])
			if len(parts) >= 3 {
				s.CookieString = parts[2]
				s.Datr = extractDatrValue(parts[2])
			}
			return s
		}
	}

	// Full cookie string: "datr=xxx;sb=yyy;..."
	if strings.Contains(raw, "datr=") {
		datr := extractDatrValue(raw)
		return Seed{
			Raw:          raw,
			Mode:         SeedModeFullCookie,
			CookieString: raw,
			Datr:         datr,
			SourceLabel:  "full_cookie(datr=" + datr[:min(len(datr), 8)] + "...)",
		}
	}

	// Simple datr value (no pipes, no equals) — C# source_type=1.
	return Seed{
		Raw:         raw,
		Mode:        SeedModeDatrOnly,
		Datr:        raw,
		SourceLabel: "datr_only(" + raw[:min(len(raw), 8)] + "...)",
	}
}

// extractDatrValue pulls the datr value from a cookie string like "c_user=xxx;datr=VALUE;locale=en".
func extractDatrValue(cookieStr string) string {
	for _, pair := range strings.Split(cookieStr, ";") {
		pair = strings.TrimSpace(pair)
		if strings.HasPrefix(pair, "datr=") {
			return strings.TrimPrefix(pair, "datr=")
		}
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── Initial login/logout warm ───────────────────────────────────────────────
//
// PORT CHÍNH XÁC từ C#: FacebookRegisterMfbRequest.
//
//	LoginFbOldUI()   — lines 367-424
//	LoginFbNewUI()   — lines 425-477
//	LogoutFb()       — lines 478-506
//
// Flow: login account cũ → session có cookies tin cậy (datr, sb, fr...)
//
//	→ logout → cookies vẫn giữ → register account mới dễ pass hơn

// loginInitialOldUI — C#: LoginFbOldUI().
// POST login/device-based/login/async/ → save-device → update-nonce.
func loginInitialOldUI(ctx context.Context, sess *session, prof fakeinfo.IPhoneProfile,
	uid, password, li string, tokens pageTokens, notify ...func(string),
) bool {
	log := func(msg string) {
		if len(notify) > 0 && notify[0] != nil {
			notify[0](msg)
		}
	}
	_ = log
	ts := fmt.Sprintf("%d", time.Now().Unix())
	encpass := fmt.Sprintf("%%23PWD_BROWSER%%3A0%%3A%s%%3A%s", ts, password)

	// C#: LoginMfbOldUIFormData — body rất lớn với browser fingerprint
	body := fmt.Sprintf(
		"m_ts=%s&li=%s&try_number=0&unrecognized_tries=0"+
			"&email=%s&prefill_contact_point=%s"+
			"&prefill_source=browser_dropdown&prefill_type=password"+
			"&first_prefill_source=browser_dropdown&first_prefill_type=contact_point"+
			"&had_cp_prefilled=true&had_password_prefilled=true"+
			"&is_smart_lock=false&bi_xrwh=0"+
			"&bi_wvdp=%%7B%%22hwc%%22%%3Atrue%%2C%%22hwcr%%22%%3Atrue%%2C%%22has_dnt%%22%%3Atrue%%2C%%22has_standalone%%22%%3Afalse%%2C%%22wnd_toStr_toStr%%22%%3A%%22function%%20toString()%%20%%7B%%20%%5Bnative%%20code%%5D%%20%%7D%%22%%2C%%22hasPerm%%22%%3Atrue%%2C%%22permission_query_toString%%22%%3A%%22function%%20query()%%20%%7B%%20%%5Bnative%%20code%%5D%%20%%7D%%22%%2C%%22permission_query_toString_toString%%22%%3A%%22function%%20toString()%%20%%7B%%20%%5Bnative%%20code%%5D%%20%%7D%%22%%2C%%22has_seWo%%22%%3Atrue%%2C%%22has_meDe%%22%%3Atrue%%2C%%22has_creds%%22%%3Atrue%%2C%%22has_hwi_bt%%22%%3Afalse%%2C%%22has_agjsi%%22%%3Afalse%%2C%%22iframeProto%%22%%3A%%22function%%20get%%20contentWindow()%%20%%7B%%20%%5Bnative%%20code%%5D%%20%%7D%%22%%2C%%22remap%%22%%3Afalse%%2C%%22iframeData%%22%%3A%%7B%%22hwc%%22%%3Atrue%%2C%%22hwcr%%22%%3Afalse%%2C%%22has_dnt%%22%%3Atrue%%2C%%22has_standalone%%22%%3Afalse%%2C%%22wnd_toStr_toStr%%22%%3A%%22function%%20toString()%%20%%7B%%20%%5Bnative%%20code%%5D%%20%%7D%%22%%2C%%22hasPerm%%22%%3Atrue%%2C%%22permission_query_toString%%22%%3A%%22function%%20query()%%20%%7B%%20%%5Bnative%%20code%%5D%%20%%7D%%22%%2C%%22permission_query_toString_toString%%22%%3A%%22function%%20toString()%%20%%7B%%20%%5Bnative%%20code%%5D%%20%%7D%%22%%2C%%22has_seWo%%22%%3Atrue%%2C%%22has_meDe%%22%%3Atrue%%2C%%22has_creds%%22%%3Atrue%%2C%%22has_hwi_bt%%22%%3Afalse%%2C%%22has_agjsi%%22%%3Afalse%%7D%%7D"+
			"&encpass=%s"+
			"&fb_dtsg=%s&jazoest=%s&lsd=%s"+
			"&__dyn=&__csr=&__hsdp=&__hblp=&__sjsp=&__req=4&__fmt=1&__a=%s&__user=0",
		tokens.spinR, li,
		uid, uid,
		encpass,
		tokens.fbDtsg, tokens.jazoest, tokens.lsd,
		tokens.encryptedStr,
	)

	// C#: PerpectMfbPostHeadersFormat3(MfbSingleUrlWithUS, MfbSingleUrl)
	postHeaders := buildPostHeaders(prof, "https://m.facebook.com/?locale=en_US", "https://m.facebook.com")
	postHeaders = append(postHeaders,
		[2]string{"x-response-format", "JSONStream"},
		[2]string{"x-requested-with", "XMLHttpRequest"},
		[2]string{"x-fb-lsd", tokens.lsd},
		[2]string{"x-asbd-id", "359341"},
	)

	respBody, err := sess.post(ctx, "https://m.facebook.com/login/device-based/login/async/?refsrc=deprecated&lwv=100", body, postHeaders)
	if err != nil {
		return false
	}
	if respBody == "" {
		return false
	}

	// C#: Regex.Unescape twice, check "login/save-device/"
	unescaped := unescapeResponse(unescapeResponse(respBody))
	if !strings.Contains(unescaped, "login/save-device/") {
		_ = fmt.Sprintf("login response: %s", respBody[:min(len(respBody), 200)])
		return false
	}

	// GET save-device
	time.Sleep(1000 * time.Millisecond)
	saveDeviceHTML, err := sess.get(ctx, "https://m.facebook.com/login/save-device/?login_source=login",
		buildNavHeaders(prof, "https://m.facebook.com/?locale=en_US", ""))
	if err != nil || saveDeviceHTML == "" {
		return false
	}
	if isCheckpoint(sess.finalURL) {
		return false
	}

	// POST update-nonce
	time.Sleep(1000 * time.Millisecond)
	dtsg2 := reFind(saveDeviceHTML, `dtsg":{"token":"(.*?)",`, 1)
	jazoest2 := reFind(saveDeviceHTML, `name="jazoest" value="(\d+)"`, 1)
	nonceBody := fmt.Sprintf("fb_dtsg=%s&jazoest=%s&flow=interstitial_nux&next=&nux_source=regular_login", dtsg2, jazoest2)
	nonceHeaders := buildNavHeaders(prof, "https://m.facebook.com/login/save-device/?login_source=login&soft=hjk", "https://m.facebook.com")
	nonceHeaders = append(nonceHeaders, [2]string{"Content-Type", "application/x-www-form-urlencoded"})
	sess.post(ctx, "https://m.facebook.com/login/device-based/update-nonce/", nonceBody, nonceHeaders)
	return true
}

// loginInitialNewUI — C#: LoginFbNewUI().
// POST wbloks login → save-credentials → update-nonce/async.
func loginInitialNewUI(ctx context.Context, sess *session, prof fakeinfo.IPhoneProfile,
	uid, password, bkv, waterfallID string, tokens pageTokens,
) bool {
	ts := fmt.Sprintf("%d", time.Now().Unix())

	// C#: LoginMfbNewUIFormData — pre-encoded wbloks login body
	body := fmt.Sprintf(
		"__aaid=0&__user=0&__a=1&__req=4"+
			"&__hs=20350.BP%%3Awbloks_caa_pkg.2.0...0"+
			"&dpr=3&__ccg=GOOD"+
			"&__rev=%s&__s=&__hsi=%s&__dyn=&locale=en_US"+
			"&fb_dtsg=%s&jazoest=%s&lsd=%s"+
			"&params=%%7B%%22params%%22%%3A%%22%%7B%%5C%%22server_params%%5C%%22%%3A%%7B%%5C%%22credential_type%%5C%%22%%3A%%5C%%22password%%5C%%22%%2C%%5C%%22username_text_input_id%%5C%%22%%3A%%5C%%2273jc4t%%3A61%%5C%%22%%2C%%5C%%22password_text_input_id%%5C%%22%%3A%%5C%%2273jc4t%%3A62%%5C%%22%%2C%%5C%%22login_source%%5C%%22%%3A%%5C%%22Login%%5C%%22%%2C%%5C%%22login_credential_type%%5C%%22%%3A%%5C%%22none%%5C%%22%%2C%%5C%%22server_login_source%%5C%%22%%3A%%5C%%22login%%5C%%22%%2C%%5C%%22ar_event_source%%5C%%22%%3A%%5C%%22login_home_page%%5C%%22%%2C%%5C%%22should_trigger_override_login_success_action%%5C%%22%%3A0%%2C%%5C%%22should_trigger_override_login_2fa_action%%5C%%22%%3A0%%2C%%5C%%22is_caa_perf_enabled%%5C%%22%%3A0%%2C%%5C%%22reg_flow_source%%5C%%22%%3A%%5C%%22login_home_native_integration_point%%5C%%22%%2C%%5C%%22caller%%5C%%22%%3A%%5C%%22gslr%%5C%%22%%2C%%5C%%22is_from_landing_page%%5C%%22%%3A0%%2C%%5C%%22is_from_empty_password%%5C%%22%%3A0%%2C%%5C%%22is_from_aymh%%5C%%22%%3A0%%2C%%5C%%22is_from_password_entry_page%%5C%%22%%3A0%%2C%%5C%%22is_from_assistive_id%%5C%%22%%3A0%%2C%%5C%%22is_from_msplit_fallback%%5C%%22%%3A0%%2C%%5C%%22two_step_login_type%%5C%%22%%3A%%5C%%22one_step_login%%5C%%22%%2C%%5C%%22is_vanilla_password_page_empty_password%%5C%%22%%3A0%%2C%%5C%%22INTERNAL__latency_qpl_marker_id%%5C%%22%%3A36707139%%2C%%5C%%22INTERNAL__latency_qpl_instance_id%%5C%%22%%3A%%5C%%2242920426900440%%5C%%22%%2C%%5C%%22device_id%%5C%%22%%3Anull%%2C%%5C%%22family_device_id%%5C%%22%%3Anull%%2C%%5C%%22waterfall_id%%5C%%22%%3A%%5C%%22%s%%5C%%22%%2C%%5C%%22offline_experiment_group%%5C%%22%%3Anull%%2C%%5C%%22layered_homepage_experiment_group%%5C%%22%%3Anull%%2C%%5C%%22is_platform_login%%5C%%22%%3A0%%2C%%5C%%22is_from_logged_in_switcher%%5C%%22%%3A0%%2C%%5C%%22is_from_logged_out%%5C%%22%%3A0%%2C%%5C%%22access_flow_version%%5C%%22%%3A%%5C%%22pre_mt_behavior%%5C%%22%%7D%%2C%%5C%%22client_input_params%%5C%%22%%3A%%7B%%5C%%22machine_id%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22cloud_trust_token%%5C%%22%%3Anull%%2C%%5C%%22block_store_machine_id%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22zero_balance_state%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22contact_point%%5C%%22%%3A%%5C%%22%s%%5C%%22%%2C%%5C%%22password%%5C%%22%%3A%%5C%%22%%23PWD_BROWSER%%3A0%%3A%s%%3A%s%%5C%%22%%2C%%5C%%22accounts_list%%5C%%22%%3A%%5B%%5D%%2C%%5C%%22fb_ig_device_id%%5C%%22%%3A%%5B%%5D%%2C%%5C%%22secure_family_device_id%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22encrypted_msisdn%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22headers_infra_flow_id%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22try_num%%5C%%22%%3A1%%2C%%5C%%22login_attempt_count%%5C%%22%%3A1%%2C%%5C%%22event_flow%%5C%%22%%3A%%5C%%22login_manual%%5C%%22%%2C%%5C%%22event_step%%5C%%22%%3A%%5C%%22home_page%%5C%%22%%2C%%5C%%22openid_tokens%%5C%%22%%3A%%7B%%7D%%2C%%5C%%22auth_secure_device_id%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22client_known_key_hash%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22has_whatsapp_installed%%5C%%22%%3A0%%2C%%5C%%22sso_token_map_json_string%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22should_show_nested_nta_from_aymh%%5C%%22%%3A0%%2C%%5C%%22password_contains_non_ascii%%5C%%22%%3A%%5C%%22false%%5C%%22%%2C%%5C%%22has_granted_read_contacts_permissions%%5C%%22%%3A0%%2C%%5C%%22has_granted_read_phone_permissions%%5C%%22%%3A0%%2C%%5C%%22app_manager_id%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22aymh_accounts%%5C%%22%%3A%%5B%%7B%%5C%%22id%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22profiles%%5C%%22%%3A%%7B%%5C%%22id%%5C%%22%%3A%%7B%%5C%%22user_id%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22name%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22profile_picture_url%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22small_profile_picture_url%%5C%%22%%3Anull%%2C%%5C%%22notification_count%%5C%%22%%3A0%%2C%%5C%%22credential_type%%5C%%22%%3A%%5C%%22none%%5C%%22%%2C%%5C%%22token%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22last_access_time%%5C%%22%%3A0%%2C%%5C%%22is_derived%%5C%%22%%3A0%%2C%%5C%%22username%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22password%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22has_smartlock%%5C%%22%%3A0%%2C%%5C%%22account_center_id%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22account_source%%5C%%22%%3A%%5C%%22%%5C%%22%%2C%%5C%%22credentials%%5C%%22%%3A%%5B%%5D%%2C%%5C%%22nta_eligibility_reason%%5C%%22%%3Anull%%2C%%5C%%22from_accurate_privacy_result%%5C%%22%%3A0%%2C%%5C%%22dbln_validated%%5C%%22%%3A0%%7D%%7D%%7D%%5D%%2C%%5C%%22lois_settings%%5C%%22%%3A%%7B%%5C%%22lois_token%%5C%%22%%3A%%5C%%22%%5C%%22%%7D%%7D%%7D%%22%%7D",
		tokens.spinR, tokens.hsi,
		tokens.fbDtsg, tokens.jazoest, tokens.lsd,
		waterfallID,
		uid, ts, password,
	)

	// C#: PerpectMfbPostHeadersFormat3(MfbSingleUrlWithUS, MfbSingleUrl)
	postHeaders := buildPostHeaders(prof, "https://m.facebook.com/?locale=en_US", "https://m.facebook.com")
	loginURL := fmt.Sprintf("https://m.facebook.com/async/wbloks/fetch/?appid=com.bloks.www.bloks.caa.login.async.send_login_request&type=action&__bkv=%s", bkv)
	respBody, err := sess.post(ctx, loginURL, body, postHeaders)
	if err != nil || respBody == "" {
		return false
	}

	// C#: check "com.bloks.www.caa.login.save-credentials" && "currentUser"
	if !strings.Contains(respBody, "com.bloks.www.caa.login.save-credentials") || !strings.Contains(respBody, "currentUser") {
		return false
	}

	// POST update-nonce/async
	time.Sleep(1000 * time.Millisecond)
	dtsg2 := reFind(respBody, `dtsgToken":"(.*?)"`, 1)
	enc2 := reFind(respBody, `encrypted":"(.*?)"`, 1)
	jazoest2 := reFind(respBody, `sprinkleValue":"(\d+)"`, 1)
	loginUID := reFind(respBody, `currentUser":(\d+),`, 1)

	nonceBody := fmt.Sprintf(
		"client_event_flow=&fb_dtsg=%s&jazoest=%s&lsd=%s"+
			"&__dyn=&__csr=&__hsdp=&__hblp=&__sjsp=&__req=a&__fmt=1&__a=%s&__user=%s&__wma=1",
		dtsg2, jazoest2, tokens.lsd, enc2, loginUID,
	)
	nonceHeaders := buildPostHeaders(prof, "https://m.facebook.com/login/save-device/", "https://m.facebook.com")
	nonceHeaders = append(nonceHeaders,
		[2]string{"x-response-format", "JSONStream"},
		[2]string{"x-requested-with", "XMLHttpRequest"},
		[2]string{"x-fb-lsd", tokens.lsd},
		[2]string{"x-asbd-id", "359341"},
	)
	nonceResp, _ := sess.post(ctx, "https://m.facebook.com/login/device-based/update-nonce/async/", nonceBody, nonceHeaders)
	if nonceResp != "" && strings.Contains(nonceResp, "success\":true") {
		time.Sleep(1000 * time.Millisecond)
		sess.get(ctx, "https://m.facebook.com?deoia=1", buildNavHeaders(prof, "https://m.facebook.com/login/save-device/", ""))
	}
	return true
}

// logoutInitial — C#: LogoutFb().
// GET confirmemail → extract logout hash → GET logout.php.
func logoutInitial(ctx context.Context, sess *session, prof fakeinfo.IPhoneProfile) {
	urlCf := "https://m.facebook.com/confirmemail.php?next=https%3A%2F%2Fm.facebook.com%2F%3Fdeoia%3D1&soft=hjk"
	html, err := sess.get(ctx, urlCf, buildNavHeaders(prof, "", ""))
	if err != nil || html == "" {
		return
	}
	logoutHash := parseLogoutHash(html)
	if logoutHash == "" {
		return
	}
	sess.get(ctx, "https://m.facebook.com/logout.php?h="+logoutHash, buildNavHeaders(prof, urlCf, ""))
}

// ─── Response parsers (page tokens + UID/cookie/datr extractors) ─────────────
//
// Mapping từ C#: FacebookRequestFormDataPropModel.BuildFromChromeAndroidPage()
// + Regex extractions trong FacebookRegisterMfbRequest.

// pageTokens chứa tokens extract từ GET m.facebook.com/ (dùng cho cả New UI và Old UI).
type pageTokens struct {
	versioningID string // bloks versioning ID (New UI)
	fbDtsg       string // fb_dtsg CSRF token
	lsd          string // LSD anti-CSRF token
	hsi          string // HTTP Signature Identifier
	jazoest      string // security token phụ
	spinR        string // __spin_r / __rev
	spinT        string // __spin_t

	// Old UI specific fields
	encryptedStr    string // encrypted="..." — dùng làm __a trong POST
	regInstance     string // name="reg_instance" value="..."
	regImpressionID string // name="reg_impression_id" value="..."
	loggerID        string // name="logger_id" value="..."
	privacyToken    string // privacy_mutation_token= trong URL submit
}

// defaultBKV — C#: _default_bkv (fallback khi không tìm được versioningID).
const defaultBKV = "3d5c3de42fc5b6024ad0c5b11df14b7e394d65431ca0c4e41086cd5297527e18"

// parsePageTokens extract tất cả tokens từ HTML của m.facebook.com/reg/.
// Mapping từ C#: BuildFromChromeAndroidPage() + versioningID regex + Old UI fields.
func parsePageTokens(html string) pageTokens {
	t := pageTokens{}

	// versioningID — New UI: C#: Regex.Match(resstr, "versioningID:\"(.*?)\"")
	t.versioningID = reFind(html, `versioningID:"(.*?)"`, 1)
	if t.versioningID == "" {
		t.versioningID = defaultBKV
	}

	// fb_dtsg
	t.fbDtsg = reFind(html, `dtsg":{"token":"(.*?)",`, 1)
	if t.fbDtsg == "" {
		t.fbDtsg = reFind(html, `"token":"([^"]+)"`, 1)
	}

	// lsd
	t.lsd = reFind(html, `LSD".*?token":"(.*?)"`, 1)

	// hsi
	t.hsi = reFind(html, `hsi":"(.*?)",`, 1)

	// jazoest — C#: name="jazoest" value="(\d+)"
	t.jazoest = reFind(html, `name="jazoest" value="(\d+)"`, 1)
	if t.jazoest == "" {
		t.jazoest = reFind(html, `jazoest", "(\d+)",`, 1)
	}

	// spin_r
	t.spinR = reFind(html, `__spin_r":(.*?),`, 1)

	// spin_t
	t.spinT = reFind(html, `__spin_t":(.*?),`, 1)

	// Old UI fields
	// C#: Regex.Match(httpResponse.ResponseBody, "encrypted\":\"(.*?)\"")
	t.encryptedStr = reFind(html, `encrypted":"(.*?)"`, 1)
	// C#: Regex.Match(httpResponse.ResponseBody, "name=\"reg_instance\" value=\"(.*?)\"")
	t.regInstance = reFind(html, `name="reg_instance" value="(.*?)"`, 1)
	// C#: Regex.Match(httpResponse.ResponseBody, "name=\"reg_impression_id\" value=\"(.*?)\"")
	t.regImpressionID = reFind(html, `name="reg_impression_id" value="(.*?)"`, 1)
	// C#: Regex.Match(httpResponse.ResponseBody, "name=\"logger_id\" value=\"(.*?)\"")
	t.loggerID = reFind(html, `name="logger_id" value="(.*?)"`, 1)

	// C#: Regex.Match(responseStr, "reg/submit/.?.privacy_mutation_token=(.*?)\"")
	privacyRaw := reFind(html, `reg/submit/[^"]*privacy_mutation_token=([^"&]+)`, 1)
	t.privacyToken = strings.ReplaceAll(privacyRaw, "amp;", "")

	return t
}

// parseUID extract UID từ register response.
// C#: Regex.Match(responseStr, "currentUser\":(\d+),").
func parseUID(body string) string {
	if uid := reFind(body, `currentUser":(\d+),`, 1); uid != "" {
		return uid
	}
	if uid := reFind(body, `current_user_id":"(\d+)"`, 1); uid != "" {
		return uid
	}
	if uid := reFind(body, `\\"uid\\":(\d+)`, 1); uid != "" {
		return uid
	}
	if uid := reFind(body, `SaveCredential.*?\\"(\d{10,18})\\"`, 1); uid != "" {
		return uid
	}
	// Old UI: c_user từ cookie string (SetCookieAndUid trong C#)
	return ""
}

// parseCUserFromCookie extract UID từ cookie string.
// C#: SetCookieAndUid → Regex.Match(cookie, "c_user=(\d+);").
func parseCUserFromCookie(cookie string) string {
	return reFind(cookie, `c_user=(\d+)`, 1)
}

// parseDatrFromHTML extract datr từ HTML thành công.
// C#: Regex.Match(resstr, "create_success.\"(.*?)\"(.*?).\",").
func parseDatrFromHTML(html string) string {
	return reFind(html, `create_success\."(.*?)"(.*?)\.",`, 2)
}

// parseLogoutHash extract logout hash từ HTML.
func parseLogoutHash(html string) string {
	// Pattern 1: logout.php?h=
	h := reFind(html, `logout\.php[^"]*h=([^"&]+)`, 1)
	if h != "" {
		return strings.ReplaceAll(h, "amp;", "")
	}
	// Pattern 2: loggedOutHash:"..." — C#: Regex.Match(responseStr, "loggedOutHash:\"(.*?)\"")
	h2 := reFind(html, `loggedOutHash:"(.*?)"`, 1)
	return h2
}

// isNewUI kiểm tra HTML có phải New UI (wbloks/Bloks CAA).
// C#: pageSource.Contains("bk.components").
func isNewUI(html string) bool {
	return strings.Contains(html, "bk.components")
}

// isCheckpoint kiểm tra URL/HTML có phải checkpoint.
func isCheckpoint(urlOrHTML string) bool {
	return strings.Contains(urlOrHTML, "checkpoint")
}

// isSaveDevice kiểm tra response có yêu cầu save-device.
// C#: responseStr.Contains("save-device") || responseStr.Contains("confirmemail.").
// Gọi SAU khi đã unescapeResponse×2.
func isSaveDevice(body string) bool {
	return strings.Contains(body, "save-device") || strings.Contains(body, "confirmemail.")
}

// isBlockedRegister kiểm tra response chứa MCheckpointRedirect / ID checkpoint.
// C#: ResponseBody.Contains("MCheckpointRedirect") || ResponseBody.Contains("1501092823525282").
// Dùng trên body CHƯA unescape (original ResponseBody).
func isBlockedRegister(body string) bool {
	return strings.Contains(body, "MCheckpointRedirect") || strings.Contains(body, "1501092823525282")
}

// isActuallyBlocked kiểm tra lỗi đăng ký thật sự (không phải checkpoint).
// C#: FacebookChromeUtils.IsBlockedRegister(pageSource) — kiểm tra ĐẦU TIÊN trước save-device.
func isActuallyBlocked(body string) bool {
	blockedKw := []string{
		"We couldn't create",
		"was an error with your registration",
		"Registration Error",
		"Something Went Wrong",
		"try refreshing the page or closing and",
	}
	for _, kw := range blockedKw {
		if strings.Contains(body, kw) {
			return true
		}
	}
	return false
}

// isConfirmMailURL kiểm tra URL/body chứa đường dẫn xác nhận email.
// C#: FacebookChromeUtils.IsConfirmMailUrl(url).
func isConfirmMailURL(s string) bool {
	keywords := []string{"notifmedium", "conf/notifmedium", "confirmation", "confirmemail.php"}
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

// unescapeResponse thực hiện 1 lần unescape tương đương C#'s Regex.Unescape.
// Chuyển \/ → /, \" → ", \\ → \, \n → newline, \r → CR, \t → tab.
// Gọi 2 lần để khớp C#: Regex.Unescape(Regex.Unescape(s)).
func unescapeResponse(s string) string {
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

// reFind tìm pattern trong s và trả về group index.
func reFind(s, pattern string, group int) string {
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
