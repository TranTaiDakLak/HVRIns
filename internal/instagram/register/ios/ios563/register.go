// Package ios563 — Facebook iOS native (FBIOS) registration, API v563.
//
// Single-shot: 1 POST create.account.async, không Round 2.
// Headers và body đúng capture RegIos563 (2026-05-30).
// Profile, device pool, datr pool dùng chung với ios562.
package ios563

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256" //nolint
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"

	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	ios562 "HVRIns/internal/instagram/register/ios/ios562"
	androidreg "HVRIns/internal/instagram/register/android"
)

// ─── Constants (capture RegIos563 / EnterCode563) ────────────────────────────

const (
	graphURL          = "https://graph.facebook.com/graphql"
	bloksVersioningID = "3be43264256070fd9e818b280c6ceee37df0f87b8ae0ed538cf3ec6b1d600d6a"
	docIDAction       = "375801096013313544589153066091"
	stylesID          = "2d85de7a219912218014b808f1a34dda"
	oauthToken        = "6628568379|c1e620fa708a1d5696fb991c1bde5662"
	createAccountAppID = "com.bloks.www.bloks.caa.reg.create.account.async"
	friendlyName       = "BKImageComponent,com.bloks.www.bloks.caa.reg.create.account.async,recommendation_image"
	analyticsTag       = `{"network_tags":{"product":"6628568379","request_category":"image","purpose":"fetch","retry_attempt":"0"},"application_tags":"BKImageComponent;recommendation_image;com.bloks.www.bloks.caa.reg.create.account.async"}`
)

// SharedDatrPool — dùng chung pool datr với ios562.
var SharedDatrPool *androidreg.PartitionedDatrPool

// Registerer implements instagram.Registerer cho platform iOS563.
type Registerer struct{}

func (r *Registerer) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	return registerAccount(ctx, input, onStatus)
}

func init() {
	instagram.RegisterPlatformRegisterer(instagram.PlatformIOS563, func() instagram.Registerer {
		return &Registerer{}
	})
}

func registerAccount(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	var pickedDatr string
	result := doRegister(ctx, input, onStatus, &pickedDatr)
	if pickedDatr != "" && SharedDatrPool != nil {
		outcome := "fail"
		if result != nil && result.Success {
			outcome = "success"
		}
		SharedDatrPool.RecordResult(pickedDatr, outcome)
	}
	return result
}

func doRegister(ctx context.Context, input *instagram.RegInput, onStatus func(string), pickedDatrOut *string) *instagram.RegResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(msg)
		}
	}

	// ── Resolve contactpoint ──────────────────────────────────────────────────
	contactpoint, cpType, proxyStr := "", "phone", ""
	if input != nil {
		proxyStr = input.Proxy
		if input.Email != "" {
			contactpoint, cpType = input.Email, "email"
		} else if input.Phone != "" {
			contactpoint = input.Phone
		}
	}
	if contactpoint == "" {
		return &instagram.RegResult{Success: false, Message: "Thiếu contactpoint"}
	}

	// ── Locale + fake profile ─────────────────────────────────────────────────
	countryCode := ""
	if input != nil && input.Phone != "" {
		if p, ok := fakeinfo.FindCountryByPhonePrefix(input.Phone); ok {
			countryCode = p.CountryCode
		}
	}
	locale := fakeinfo.LocaleFromCountry(countryCode)
	if locale == "" {
		locale = "en_US"
	}
	fake := fakeinfo.RandomFakeProfileByLocale(countryCode)
	if input != nil {
		if input.FirstName != "" { fake.FirstName = input.FirstName }
		if input.LastName != "" { fake.LastName = input.LastName }
		if input.Birthday != "" { fake.Birthday = input.Birthday }
		if input.Gender > 0 { fake.Gender = input.Gender }
	}
	password := fakeinfo.RandomPassword()
	if input != nil && input.Password != "" {
		password = input.Password
	}

	slotIdx := 0
	if input != nil { slotIdx = input.SlotIdx }

	// ── Build profile ─────────────────────────────────────────────────────────
	var profile ios562.IOSProfile
	if ios562.SharedDevicePool != nil {
		if dp := ios562.SharedDevicePool.GetNext(); dp != nil {
			profile = ios562.BuildProfileFromDevice(locale, countryCode, *dp)
			notify(fmt.Sprintf("[iOS563] Start — %s %s | %s | %s [pool dev]",
				fake.FirstName, fake.LastName, contactpoint, profile.Device.FBDV))
		}
	}
	if profile.DeviceID == "" {
		profile = ios562.BuildProfile(locale, countryCode)
		notify(fmt.Sprintf("[iOS563] Start — %s %s | %s | %s [rand dev]",
			fake.FirstName, fake.LastName, contactpoint, profile.Device.FBDV))
	}

	// ── Datr pool ─────────────────────────────────────────────────────────────
	var pickedDatr string
	if SharedDatrPool != nil {
		if datr := SharedDatrPool.GetNext(slotIdx); datr != "" {
			pickedDatr = datr
			profile.MachineID = datr
			pfx := datr
			if len(pfx) > 8 { pfx = pfx[:8] }
			_, _, _, used := SharedDatrPool.GetStats(datr)
			notify(fmt.Sprintf("[iOS563] Datr %s... used=%d", pfx, used))
		}
	}
	if pickedDatrOut != nil { *pickedDatrOut = pickedDatr }

	// ── HTTP session ──────────────────────────────────────────────────────────
	sess, err := ios562.NewSession(proxyStr, profile.Device.IOSDot)
	if err != nil {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Session lỗi: %v", err), Password: password}
	}

	// ── Build body ────────────────────────────────────────────────────────────
	body, err := buildBody(profile, contactpoint, cpType, fake, password, locale)
	if err != nil {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Build body lỗi: %v", err), Password: password}
	}

	headers := buildHeaders(profile)
	notify(fmt.Sprintf("[iOS563] POST create.account (%s)...", profile.Device.FBDV))

	respBody, herr := sess.PostGzip(ctx, graphURL, body, headers)
	if herr != nil && respBody == "" {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("HTTP lỗi: %v", herr), Password: password}
	}
	notify(fmt.Sprintf("[iOS563] Response %d bytes", len(respBody)))

	// ── Parse ─────────────────────────────────────────────────────────────────
	outcome, perr := ios562.ParseCreateAccountResponse(respBody)
	if perr != nil {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Parse lỗi: %v", perr), Password: password}
	}

	// ── Collect datr từ response ──────────────────────────────────────────────
	if outcome.DATR != "" && SharedDatrPool != nil {
		if SharedDatrPool.AddDatrRaw(outcome.DATR) {
			notify(fmt.Sprintf("[iOS563Pool] Datr mới: %s...", outcome.DATR[:min(10, len(outcome.DATR))]))
		}
	}

	// ── Save device profile ───────────────────────────────────────────────────
	if ios562.SharedDevicePool != nil {
		ios562.SharedDevicePool.Add(ios562.DeviceProfile{
			DeviceID:       profile.DeviceID,
			FamilyDeviceID: profile.FamilyDeviceID,
			MachineID:      profile.MachineID,
		})
	}

	cookie := outcome.Cookie
	if cookie != "" { cookie += "locale=" + locale + ";" }

	msg := fmt.Sprintf("Register OK — UID: %s (iOS563)", outcome.UID)
	if outcome.AccessToken == "" { msg += " — cần verify" }
	notify("[iOS563] " + msg)

	return &instagram.RegResult{
		Success:     true,
		UID:         outcome.UID,
		Cookie:      cookie,
		AccessToken: outcome.AccessToken,
		Password:    password,
		Message:     msg,
		UserAgent:   profile.UserAgent,
		DeviceID:    profile.DeviceID,
		FamilyDeviceID: profile.FamilyDeviceID,
	}
}

func min(a, b int) int {
	if a < b { return a }
	return b
}

// ─── Headers (capture RegIos563) ─────────────────────────────────────────────

func buildHeaders(p ios562.IOSProfile) [][2]string {
	return [][2]string{
		{"user-agent", p.UserAgent},
		{"accept-encoding", "gzip, deflate, br"},
		{"accept", "*/*"},
		{"connection", "keep-alive"},
		{"x-meta-usdid", generateUSDID()},
		{"x-fb-http-engine", "Tigon/Liger"},
		{"x-meta-zca", `{"e": {"c":7}}`},
		{"x-tigon-is-retry", "False"},
		{"authorization", "OAuth " + oauthToken},
		{"content-encoding", "gzip"},
		{"x-fb-connection-type", p.ConnType},
		{"x-cloud-trust-token", p.CloudTrustID},
		{"x-fb-integrity-machine-id", p.MachineID},
		{"x-fb-device-id", p.DeviceID},
		{"x-fb-friendly-name", friendlyName},
		{"content-type", "application/x-www-form-urlencoded"},
		{"x-fb-request-analytics-tags", analyticsTag},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
		{"x-fb-conn-uuid-client", connUUID()},
		{"x-graphql-client-library", "pando"},
		{"x-graphql-request-purpose", "fetch"},
		{"x-fb-aed", "683"},
		{"x-fb-optimizer", "0"},
	}
}

// ─── Body ─────────────────────────────────────────────────────────────────────

func buildBody(p ios562.IOSProfile, contactpoint, cpType string, fake fakeinfo.FakeProfile, password, locale string) (string, error) {
	regInfo := map[string]any{
		"first_name":        fake.FirstName,
		"last_name":         fake.LastName,
		"contactpoint":      contactpoint,
		"contactpoint_type": cpType,
		"birthday":          fake.Birthday,
		"gender":            fake.Gender,
		"encrypted_password": fmt.Sprintf("#PWD_FB4A:0:%d:%s", time.Now().Unix(), password),
		"device_id":         p.DeviceID,
		"family_device_id":  p.FamilyDeviceID,
		"machine_id":        p.MachineID,
		"reg_flow_source":   "cacheable_aymh_screen",
		"is_caa_perf_enabled": true,
	}
	regInfoJSON, err := json.Marshal(regInfo)
	if err != nil {
		return "", fmt.Errorf("marshal reg_info: %w", err)
	}

	pixelRatio := 2
	if p.Device.FBSS == "3" { pixelRatio = 3 }

	serverParams := map[string]any{
		"reg_info":                          string(regInfoJSON),
		"reg_context":                       nil,
		"current_step":                      8,
		"device_id":                         p.DeviceID,
		"family_device_id":                  p.FamilyDeviceID,
		"machine_id":                        p.MachineID,
		"waterfall_id":                      p.WaterfallID,
		"event_request_id":                  generateUUID(),
		"flow_info":                         `{"flow_name":"new_to_family_fb_default","flow_type":"ntf"}`,
		"login_surface":                     "unknown",
		"login_entry_point":                 "logged_out",
		"cloud_trust_token":                 p.CloudTrustID,
		"access_flow_version":               "pre_mt_behavior",
		"offline_experiment_group":          nil,
		"layered_homepage_experiment_group": nil,
		"bloks_controller_source":           "BloksCAARegUtils::getTriggerAccountSetupAction",
		"is_from_logged_out":                0,
		"is_from_logged_in_switcher":        0,
		"is_platform_login":                 0,
		"is_from_registration_flow":         1,
		"INTERNAL__latency_qpl_marker_id":   36707139,
		"INTERNAL__latency_qpl_instance_id": time.Now().UnixNano() % 1000000000000000,
	}

	clientInputParams := map[string]any{
		"device_id":              p.DeviceID,
		"family_device_id":       p.FamilyDeviceID,
		"machine_id":             p.MachineID,
		"block_store_machine_id": "",
		"cloud_trust_token":      p.CloudTrustID,
		"network_bssid":          nil,
		"aac":                    "",
		"fb_ig_device_id":        []any{},
		"lois_settings":          map[string]any{"lois_token": ""},
	}

	inner := map[string]any{
		"server_params":       serverParams,
		"client_input_params": clientInputParams,
	}
	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return "", fmt.Errorf("marshal inner: %w", err)
	}

	variables := map[string]any{
		"voiceover_enabled":            "false",
		"should_include_delegate_page": true,
		"scale":                        pixelRatio,
		"generic_attachment_tall_cover_image_height":         352,
		"include_workplace_fields":                           "false",
		"formatType":                                         "concise",
		"generic_attachment_tall_cover_image_width_no_scale": 335,
		"params": map[string]any{
			"bloks_versioning_id": bloksVersioningID,
			"app_id":              createAccountAppID,
			"params":              string(innerJSON),
		},
		"generic_attachment_tall_cover_image_width": 670,
		"nt_context": map[string]any{
			"theme_params": []any{
				map[string]any{"design_system_name": "FDS", "value": []string{"DEFAULT"}},
				map[string]any{"design_system_name": "XMDS", "value": []string{"three_neutral_gray"}},
			},
			"pixel_ratio":   pixelRatio,
			"styles_id":     stylesID,
			"bloks_version": bloksVersioningID,
		},
		"generic_attachment_small_cover_image_height":       80,
		"enable_voiceover_gating_for_accessibility_caption": "true",
		"device": "iphone",
		"generic_attachment_small_cover_image_width": 80,
	}
	variablesJSON, err := json.Marshal(variables)
	if err != nil {
		return "", fmt.Errorf("marshal variables: %w", err)
	}

	form := url.Values{}
	form.Set("method", "post")
	form.Set("pretty", "false")
	form.Set("format", "json")
	form.Set("server_timestamps", "true")
	form.Set("locale", locale)
	form.Set("purpose", "fetch")
	form.Set("fb_api_req_friendly_name", "FBBloksActionRootQuery-"+createAccountAppID)
	form.Set("client_doc_id", docIDAction)
	form.Set("fb_api_client_context", `{"is_background":"0"}`)
	form.Set("variables", string(variablesJSON))
	return form.Encode(), nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// generateUUID trả UUID v4 string.
func generateUUID() string {
	return uuid.New().String()
}

// connUUID — x-fb-conn-uuid-client: base64(UUID bytes).
func connUUID() string {
	id := uuid.New()
	return base64.StdEncoding.EncodeToString(id[:])
}

// generateUSDID — x-meta-usdid: "{UUID}.{unix_ts}.{ECDSA-P256 sig}".
func generateUSDID() string {
	id := uuid.New().String()
	ts := fmt.Sprintf("%d", time.Now().Unix())
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

