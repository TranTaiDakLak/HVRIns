package igandroid

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"HVRIns/internal/instagram"
)

// ── Profile generation ───────────────────────────────────────────────────────

func TestNewAndroidProfile_Fields(t *testing.T) {
	p, err := newAndroidProfile("en_GB")
	if err != nil {
		t.Fatalf("newAndroidProfile: %v", err)
	}
	if !strings.HasPrefix(p.AndroidID, "android-") {
		t.Errorf("AndroidID want prefix 'android-', got %q", p.AndroidID)
	}
	if len(p.AndroidID) != 24 { // "android-" (8) + 16 hex chars
		t.Errorf("AndroidID len want 24, got %d", len(p.AndroidID))
	}
	if len(p.DeviceID) != 36 {
		t.Errorf("DeviceID len want 36, got %d", len(p.DeviceID))
	}
	if len(p.FamilyDeviceID) != 36 {
		t.Errorf("FamilyDeviceID len want 36, got %d", len(p.FamilyDeviceID))
	}
	if len(p.RegMachineID) != 24 {
		t.Errorf("RegMachineID len want 24, got %d", len(p.RegMachineID))
	}
	if len(p.ConnUUID) != 32 {
		t.Errorf("ConnUUID len want 32, got %d", len(p.ConnUUID))
	}
	if !strings.HasPrefix(p.PigeonSID, "UFS-") || !strings.HasSuffix(p.PigeonSID, "-0") {
		t.Errorf("PigeonSID format wrong: %q", p.PigeonSID)
	}
	if p.keystoreKey == nil {
		t.Error("keystoreKey is nil")
	}
	if len(p.keystoreHash) != 64 { // hex SHA-256
		t.Errorf("keystoreHash len want 64, got %d", len(p.keystoreHash))
	}
	if !strings.Contains(p.UserAgent, "SM-G996B") {
		t.Errorf("UserAgent missing SM-G996B: %q", p.UserAgent)
	}
	if !strings.Contains(p.UserAgent, "en_GB") {
		t.Errorf("UserAgent missing locale: %q", p.UserAgent)
	}
	if p.AACInitTS == 0 {
		t.Error("AACInitTS is 0")
	}
	if len(p.AACCS) != 43 {
		t.Errorf("AACCS len want 43, got %d", len(p.AACCS))
	}
}

func TestNewAndroidProfile_Unique(t *testing.T) {
	p1, _ := newAndroidProfile("en_GB")
	p2, _ := newAndroidProfile("en_GB")
	if p1.DeviceID == p2.DeviceID {
		t.Error("DeviceID should be unique per profile")
	}
	if p1.RegMachineID == p2.RegMachineID {
		t.Error("RegMachineID should be unique per profile")
	}
}

// ── Bloks body ───────────────────────────────────────────────────────────────

func TestBuildBloksBody_Format(t *testing.T) {
	cip := map[string]any{"contactpoint": "test@example.com"}
	sp := map[string]any{"current_step": 0, "reg_context": ""}
	body := buildBloksBody(cip, sp)

	if !strings.HasPrefix(body, "params=") {
		t.Error("body must start with 'params='")
	}
	if !strings.Contains(body, "&bk_client_context=") {
		t.Error("body must contain '&bk_client_context='")
	}
	if !strings.Contains(body, "&bloks_versioning_id="+igAndroidBloksVer) {
		t.Errorf("body missing bloks_versioning_id")
	}

	// Extract and decode params
	idx := strings.Index(body, "&bk_client_context=")
	paramPart := strings.TrimPrefix(body[:idx], "params=")
	paramDecoded, err := url.QueryUnescape(paramPart)
	if err != nil {
		t.Fatalf("url decode params: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(paramDecoded), &m); err != nil {
		t.Errorf("params not valid JSON: %v", err)
	}
	if _, ok := m["client_input_params"]; !ok {
		t.Error("params missing client_input_params")
	}
	if _, ok := m["server_params"]; !ok {
		t.Error("params missing server_params")
	}
}

// ── reg_info JSON ────────────────────────────────────────────────────────────

func TestBuildRegInfoJSON_InitialState(t *testing.T) {
	p, _ := newAndroidProfile("en_GB")
	s := &regInfoState{Jurisdiction: "VN"}

	raw := buildRegInfoJSON(p, s)
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("buildRegInfoJSON invalid JSON: %v", err)
	}

	// Device fields always present
	for k, want := range map[string]string{
		"device_id":           p.AndroidID,
		"ig4a_qe_device_id":  p.DeviceID,
		"family_device_id":   p.FamilyDeviceID,
		"machine_id":         p.RegMachineID,
		"registration_flow_id": p.RegFlowID,
		"contactpoint_type":  "email",
	} {
		if m[k] != want {
			t.Errorf("key %q: want %q, got %v", k, want, m[k])
		}
	}

	// Nullable fields null in initial state
	for _, k := range []string{"contactpoint", "confirmation_code", "birthday", "full_name", "username"} {
		if v, ok := m[k]; !ok || v != nil {
			t.Errorf("key %q should be null, got %v (present=%v)", k, v, ok)
		}
	}
}

func TestBuildRegInfoJSON_AfterAllSteps(t *testing.T) {
	p, _ := newAndroidProfile("en_GB")
	savePwd := true
	skipYouth := true
	s := &regInfoState{
		ContactPoint:       "user@example.com",
		ConfirmationCode:   "IexKwq6m",
		EncryptedPassword:  "#PWD_INSTAGRAM:4:123:abc",
		Birthday:           "15-06-1992",
		AgeRange:           "o18",
		FullName:           "Test User",
		Username:           "test.user1234567",
		Jurisdiction:       "VN",
		ShouldSavePassword: &savePwd,
		ShouldSkipYouthTOS: &skipYouth,
		ScreenVisited:      []string{"CAA_REG_CONTACT_POINT_EMAIL", "CAA_REG_NAME"},
	}

	raw := buildRegInfoJSON(p, s)
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	checks := map[string]any{
		"contactpoint":         "user@example.com",
		"confirmation_code":    "IexKwq6m",
		"encrypted_password":   "#PWD_INSTAGRAM:4:123:abc",
		"birthday":             "15-06-1992",
		"age_range":            "o18",
		"full_name":            "Test User",
		"username":             "test.user1234567",
		"should_save_password": true,
		"should_skip_youth_tos": true,
	}
	for k, want := range checks {
		if m[k] != want {
			t.Errorf("key %q: want %v, got %v", k, want, m[k])
		}
	}

	// accounts_list_client = [] when username set
	v, ok := m["accounts_list_client"]
	if !ok || v == nil {
		t.Error("accounts_list_client should be [] when username is set")
	}

	// did_use_age = false (not nil) when age_range set
	if m["did_use_age"] != false {
		t.Errorf("did_use_age: want false, got %v", m["did_use_age"])
	}

	// youth_regulation_config
	cfg, ok := m["youth_regulation_config"].(map[string]any)
	if !ok {
		t.Fatal("youth_regulation_config missing or wrong type")
	}
	if cfg["consentJurisdiction"] != "VN" {
		t.Errorf("consentJurisdiction: want VN, got %v", cfg["consentJurisdiction"])
	}

	// screen_visited non-empty
	if sv, ok := m["screen_visited"].([]any); !ok || len(sv) == 0 {
		t.Error("screen_visited should be non-empty array")
	}
}

// ── AAC ──────────────────────────────────────────────────────────────────────

func TestBuildAACJSON(t *testing.T) {
	p, _ := newAndroidProfile("en_GB")
	raw := buildAACJSON(p)
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, k := range []string{"aac_init_timestamp", "aacjid", "aaccs"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing key %q", k)
		}
	}
}

// ── SafetyNet token ──────────────────────────────────────────────────────────

func TestBuildSafetynetToken(t *testing.T) {
	email := "test@example.com"
	tok := buildSafetynetToken(email)

	raw, err := base64.StdEncoding.DecodeString(tok)
	if err != nil {
		t.Fatalf("not valid base64: %v", err)
	}
	decoded := string(raw)
	if !strings.HasPrefix(decoded, email+"|") {
		t.Errorf("decoded token must start with %q|, got %q", email, decoded[:minInt(len(decoded), 40)])
	}
	// email|timestamp(10+)|16bytes → minimum length
	if len(raw) < len(email)+1+10+1+16 {
		t.Errorf("decoded too short: %d bytes", len(raw))
	}
}

// ── Attestation params ───────────────────────────────────────────────────────

func TestBuildAttestParams(t *testing.T) {
	p, _ := newAndroidProfile("en_GB")
	at := &attestResult{
		keystoreNonce:      "IVhe56cAMxK1YGidwNvWfoTy4a8UnRBE",
		keystoreSigned:     "MEYCIQDXStEP",
		keystoreHash:       p.keystoreHash,
		playIntegrityNonce: "yM96tbUon0RIaATQlqLYm2VS1NEGZ8Pe",
	}
	raw := buildAttestParams(p, at)

	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	ka, ok := m["keystore_attests"].([]any)
	if !ok || len(ka) == 0 {
		t.Fatal("keystore_attests missing/empty")
	}
	k0 := ka[0].(map[string]any)
	errs0 := k0["errors"].([]any)
	if len(errs0) == 0 || errs0[0].(float64) != 0 {
		t.Errorf("keystore errors should be [0], got %v", errs0)
	}
	if k0["signed_nonce"] != "MEYCIQDXStEP" {
		t.Errorf("signed_nonce wrong: %v", k0["signed_nonce"])
	}

	pi, ok := m["play_integrity_attests"].([]any)
	if !ok || len(pi) == 0 {
		t.Fatal("play_integrity_attests missing/empty")
	}
	pi0 := pi[0].(map[string]any)
	errs1 := pi0["errors"].([]any)
	if len(errs1) == 0 || errs1[0].(float64) != -3 {
		t.Errorf("play_integrity errors should be [-3], got %v", errs1)
	}
	if pi0["integrity_token"] != "" {
		t.Errorf("integrity_token should be empty, got %v", pi0["integrity_token"])
	}
}

// ── Headers ──────────────────────────────────────────────────────────────────

func TestAndroidHeaders(t *testing.T) {
	p, _ := newAndroidProfile("en_GB")
	p.MachineID = "testmid123"
	headers := androidHeaders(p, "test_endpoint")

	hMap := make(map[string]string)
	for _, kv := range headers {
		hMap[kv[0]] = kv[1]
	}

	for _, k := range []string{
		"x-ig-app-id", "x-ig-android-id", "x-ig-device-id", "x-ig-family-device-id",
		"x-mid", "x-bloks-version-id", "x-ig-capabilities", "user-agent",
		"content-type", "accept-encoding",
	} {
		if hMap[k] == "" {
			t.Errorf("missing required header: %q", k)
		}
	}
	if hMap["x-ig-app-id"] != igAndroidAppID {
		t.Errorf("x-ig-app-id: want %s, got %s", igAndroidAppID, hMap["x-ig-app-id"])
	}
	if hMap["x-bloks-version-id"] != igAndroidBloksVer {
		t.Error("x-bloks-version-id mismatch")
	}
	if hMap["x-mid"] != "testmid123" {
		t.Errorf("x-mid wrong: %q", hMap["x-mid"])
	}
	if hMap["accept-encoding"] != "zstd" {
		t.Errorf("accept-encoding should be zstd, got %q", hMap["accept-encoding"])
	}
	if !strings.Contains(hMap["x-fb-friendly-name"], "test_endpoint") {
		t.Errorf("x-fb-friendly-name missing endpoint: %q", hMap["x-fb-friendly-name"])
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func TestRandHelpers(t *testing.T) {
	h := randHex(8)
	if len(h) != 16 {
		t.Errorf("randHex(8): want 16 chars, got %d", len(h))
	}

	a := randAlphanumeric(24)
	if len(a) != 24 {
		t.Errorf("randAlphanumeric(24): want 24 chars, got %d", len(a))
	}

	b := randBase64URL(43)
	if len(b) != 43 {
		t.Errorf("randBase64URL(43): want 43 chars, got %d", len(b))
	}
}

func TestBuildName_Fallback(t *testing.T) {
	n := buildName(&instagram.RegInput{})
	if n == "" {
		t.Error("buildName with empty input should return non-empty fallback")
	}
}

func TestBuildName_FromInput(t *testing.T) {
	n := buildName(&instagram.RegInput{FirstName: "Long", LastName: "Le"})
	if !strings.Contains(n, "Long") || !strings.Contains(n, "Le") {
		t.Errorf("buildName should use FirstName+LastName, got %q", n)
	}
}

func TestBuildUsername_Format(t *testing.T) {
	u := buildUsername()
	if !strings.Contains(u, ".") {
		t.Errorf("buildUsername should contain '.', got %q", u)
	}
	parts := strings.SplitN(u, ".", 2)
	if len(parts[1]) < 6 {
		t.Errorf("username number part too short: %q", parts[1])
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
