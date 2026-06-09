// Package s399 — Facebook FB4A v399 register (2-step flow).
//
// Khác hoàn toàn s555-s560:
//   - Step 1: POST /app/users (friendly-name=registerAccount) → trả new_user_id + machine_id
//   - Step 2: POST /auth/login (friendly-name=authenticate) → dùng new_user_id làm email,
//     trả access_token + session_cookies (c_user/xs/fr/datr)
//
// Endpoint native FB4A cũ (không qua Bloks/GraphQL). UA có Dalvik prefix.
package s399

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"

	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
)

// ─── Registerer ───────────────────────────────────────────────────────────────

type Registerer struct{}

func (r *Registerer) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	return registerAccount(ctx, input, onStatus)
}

func init() {
	instagram.RegisterPlatformRegisterer(instagram.PlatformS399, func() instagram.Registerer {
		return &Registerer{}
	})
}

// ─── WorkerContext ────────────────────────────────────────────────────────────

type WorkerContext struct {
	sess        *session
	profile     S399Profile
	countryCode string
}

func NewWorkerContext(proxyStr, countryCode string) (*WorkerContext, error) {
	sess, err := newSession(proxyStr)
	if err != nil {
		return nil, err
	}
	profile := BuildProfileForPlatform(countryCode)
	return &WorkerContext{sess: sess, profile: profile, countryCode: countryCode}, nil
}

func (w *WorkerContext) Close() {
	if w != nil && w.sess != nil {
		w.sess.client.CloseIdleConnections()
	}
}

func (w *WorkerContext) Profile() S399Profile { return w.profile }

func (w *WorkerContext) SetLocale(locale string) {
	if w != nil && locale != "" {
		w.profile.Locale = locale
	}
}

func (w *WorkerContext) SetConnectionType(ct string) {
	if w != nil && ct != "" {
		w.profile.ConnType = ct
		w.profile.ConnectionType = ct
	}
}

// SetUAOptions — placeholder để pattern khớp với s5xx (currently no-op vì s399 dùng
// Dalvik UA template động trong BuildProfileForPlatform).
func (w *WorkerContext) SetUAOptions(_ bool) {
	if w == nil {
		return
	}
	w.profile = BuildProfileForPlatform(w.countryCode)
}

func (w *WorkerContext) SetUA(ua string) {
	if w == nil || ua == "" {
		return
	}
	w.profile.S399UA = ua
}

// ─── Response models ──────────────────────────────────────────────────────────

type registerResponse struct {
	NewUserID                 string `json:"new_user_id"`
	MachineID                 string `json:"machine_id"`
	NormalizedCP              string `json:"normalized_cp"`
	AccountType               string `json:"account_type"`
	IsPhoneClaimConfirmed     bool   `json:"is_phone_claim_confirmed"`
	IsPhoneClaimPending       bool   `json:"is_phone_claim_pending"`
	ConfViaOAuthFailureReason string `json:"conf_via_oauth_failure_reason"`
	// Error fields
	Error      *registerError `json:"error,omitempty"`
	ErrorCode  int            `json:"error_code,omitempty"`
	ErrorMsg   string         `json:"error_msg,omitempty"`
	ErrorTitle string         `json:"error_title,omitempty"`
}

type registerError struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Type      string `json:"type"`
	ErrorData string `json:"error_user_msg,omitempty"`
}

type loginResponse struct {
	SessionKey     string          `json:"session_key"`
	UID            int64           `json:"uid"`
	AccessToken    string          `json:"access_token"`
	Secret         string          `json:"secret"`
	SessionCookies []sessionCookie `json:"session_cookies"`
	AnalyticsClaim string          `json:"analytics_claim"`
	Identifier     string          `json:"identifier"`
	Confirmed      bool            `json:"confirmed"`
	// Error fields
	Error     *registerError `json:"error,omitempty"`
	ErrorCode int            `json:"error_code,omitempty"`
	ErrorMsg  string         `json:"error_msg,omitempty"`
}

type sessionCookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}

// ─── Registration flow (2 step) ──────────────────────────────────────────────

func registerAccount(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	proxyStr := ""
	countryCode := ""
	if input != nil {
		proxyStr = input.Proxy
		countryCode = countryFromPhone(input.Phone)
	}
	wctx, err := NewWorkerContext(proxyStr, countryCode)
	if err != nil {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Create session failed: %v", err)}
	}
	defer wctx.Close()
	return wctx.Register(ctx, input, onStatus)
}

func (w *WorkerContext) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(msg)
		}
	}

	profile := w.profile
	sess := w.sess

	if input != nil {
		if input.FirstName != "" {
			profile.FirstName = input.FirstName
		}
		if input.LastName != "" {
			profile.LastName = input.LastName
		}
		if input.Birthday != "" {
			profile.Birthday = input.Birthday
		}
		if input.Gender > 0 {
			profile.Gender = input.Gender
		}
		if input.UserAgent != "" {
			profile.UserAgent = input.UserAgent
			profile.S399UA = input.UserAgent
		}
		// UseOriginalUA: force device/locale/SIM khớp OriginalUA (SM-S911B, en_GB, samsung
		// + FBCR carrier theo IP).
		if input.UseOriginalUA {
			if input.OriginalSim.OperatorName != "" {
				profile.Sim = input.OriginalSim
			}
			profile.Locale = "en_GB"
			profile.FullRegProfile.Device.Brand = "samsung"
			profile.FullRegProfile.Device.Model = "SM-S911B"
			profile.FullRegProfile.Device.OSVersion = "15"
		}
	}

	var seed Seed
	if input != nil && input.TutDatr != "" {
		seed = ParseSeed(input.TutDatr)
		if seed.Datr != "" {
			profile.MachineID = seed.Datr
		}
	}
	slotIdx := 0
	if input != nil {
		slotIdx = input.SlotIdx
	}
	if profile.MachineID == "" && SharedPool != nil {
		if poolDatr := SharedPool.GetNext(slotIdx); poolDatr != "" {
			profile.MachineID = poolDatr
			seed = Seed{Raw: poolDatr, Mode: SeedModeDatrOnly, Datr: poolDatr, SourceLabel: "pool"}
			s, f, u, used := SharedPool.GetStats(poolDatr)
			notify(fmt.Sprintf("[s399] New initial %s (used %d | S/F/U: %d/%d/%d)",
				poolDatr, used, s, f, u))
		}
	} else if profile.MachineID != "" && SharedPool != nil {
		s, f, u, used := SharedPool.GetStats(profile.MachineID)
		notify(fmt.Sprintf("[s399] Dùng datr %s... (used %d | S/F/U: %d/%d/%d)",
			safeShort(profile.MachineID, 10), used, s, f, u))
	}
	if profile.MachineID != "" {
		if SharedPool != nil {
			defer SharedPool.IncrementUsage(profile.MachineID)
		}
		sess.addCookie("datr", profile.MachineID)
	}
	if seed.CookieString != "" {
		seedCookieString(sess, seed.CookieString)
	}

	// Email contactpoint — s399 chỉ hỗ trợ email-based reg (capture: email=<...>@gmail.com).
	contactpointEmail := ""
	if input != nil && input.Email != "" {
		contactpointEmail = input.Email
	}
	if contactpointEmail == "" {
		return &instagram.RegResult{Success: false, Message: "S399 yêu cầu email (s399 không support phone-based reg)"}
	}

	password := fakeinfo.RandomPassword()
	if input != nil && input.Password != "" {
		password = input.Password
	}

	// Birthday format: input có dạng "DD-MM-YYYY", v399 cần "YYYY-MM-DD"
	birthdayYMD := convertBirthdayToYMD(profile.Birthday)
	gender := "F"
	if profile.Gender == 2 {
		gender = "M"
	}

	notify(fmt.Sprintf("[s399] Start — %s %s | %s | %s",
		profile.FirstName, profile.LastName, contactpointEmail, profile.Device.Model))

	// ─── Step 1: POST /app/users (registerAccount) ───────────────────────────
	now := time.Now()
	startTs := now.UnixMilli() - 50000 // giả lập user đã ở app ~50s trước reg
	encPwdReg := fmt.Sprintf("#PWD_FB4A:0:%d:%s", now.Unix(), password)
	regInstance := uuid.New().String()
	advertisingID := uuid.New().String()
	familyDeviceID := uuid.New().String()
	if profile.FamilyDeviceID != "" {
		familyDeviceID = profile.FamilyDeviceID
	}
	deviceID := uuid.New().String()
	if profile.DeviceID != "" {
		deviceID = profile.DeviceID
	}
	sessionID := uuid.New().String()
	jazoest := fmt.Sprintf("2%d", 2000+rand.Intn(8000))

	regParams := RegisterParams{
		Email:                       contactpointEmail,
		FirstName:                   profile.FirstName,
		LastName:                    profile.LastName,
		Gender:                      gender,
		BirthdayYMD:                 birthdayYMD,
		EncPassword:                 encPwdReg,
		SessionID:                   sessionID,
		DeviceID:                    deviceID,
		RegInstance:                 regInstance,
		FamilyDeviceID:              familyDeviceID,
		AdvertisingID:               advertisingID,
		Locale:                      profile.Locale,
		CountryCode:                 strings.ToUpper(profile.CountryCode),
		Jazoest:                     jazoest,
		StartTimestampMs:            startTs,
		StartCompletedTimestampMs:   startTs + 1000,
		NameAcquiredTimestampMs:     startTs + 15000,
		BirthdayAcquiredTimestampMs: startTs + 20000,
		GenderAcquiredTimestampMs:   startTs + 22000,
		CPAcquiredTimestampMs:       startTs + 32000,
		PWAcquiredTimestampMs:       startTs + 36000,
		TOSAcquiredTimestampMs:      now.UnixMilli(),
		SnNonce:                     "dW5rbm93bnwxNzc4MDkyNjA0fHnum0MydPIKED8v7iqdJ7+4sM2UWeiVrw==",
		SnResult:                    "API_ERROR: class X.Tae:17: The SafetyNet Attestation API is deprecated and no longer functional.",
	}
	regBody := buildRegisterBody(regParams)
	regHeaders := buildRegisterHeaders(profile, profile.S399UA)

	notify("[s399] Step 1: POST /app/users...")
	respBody, err := sess.postGzip(ctx, instagram.BaseURLBGraph+"/app/users", regBody, regHeaders)
	if err != nil {
		notify(fmt.Sprintf("[s399] Step 1 HTTP error: %v", err))
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Step 1 HTTP error: %v", err), Password: password}
	}

	var regResp registerResponse
	if err := json.Unmarshal([]byte(respBody), &regResp); err != nil {
		notify(fmt.Sprintf("[s399] Parse step 1 fail: %v", err))
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Parse step 1 fail: %v — body=%.300s", err, respBody), Password: password}
	}
	if regResp.Error != nil && regResp.Error.Message != "" {
		notify(fmt.Sprintf("[s399] Step 1 server error: %s", regResp.Error.Message))
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Step 1: %s", regResp.Error.Message), Password: password}
	}
	if regResp.NewUserID == "" {
		notify("[s399] Step 1: missing new_user_id")
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Step 1 no UID — body=%.300s", respBody), Password: password}
	}
	notify(fmt.Sprintf("[s399] Step 1 OK — UID=%s machine_id=%s", regResp.NewUserID, regResp.MachineID))

	// ─── Step 2: POST /auth/login (authenticate) ─────────────────────────────
	encPwdLogin := fmt.Sprintf("#PWD_FB4A:0:%d:%s", time.Now().Unix(), password)
	loginParams := LoginParams{
		UID:            regResp.NewUserID,
		EncPassword:    encPwdLogin,
		DeviceID:       deviceID,
		FamilyDeviceID: familyDeviceID,
		AdvertisingID:  advertisingID,
		MachineID:      regResp.MachineID,
		Locale:         profile.Locale,
		CountryCode:    strings.ToUpper(profile.CountryCode),
		Jazoest:        jazoest,
	}
	loginBody := buildLoginBody(loginParams)
	loginHeaders := buildLoginHeaders(profile, profile.S399UA)

	notify("[s399] Step 2: POST /auth/login...")
	loginRespBody, err := sess.postGzip(ctx, instagram.BaseURLBGraph+"/auth/login", loginBody, loginHeaders)
	if err != nil {
		notify(fmt.Sprintf("[s399] Step 2 HTTP error: %v", err))
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Step 2 HTTP error: %v", err), Password: password, UID: regResp.NewUserID}
	}

	var loginResp loginResponse
	if err := json.Unmarshal([]byte(loginRespBody), &loginResp); err != nil {
		notify(fmt.Sprintf("[s399] Parse step 2 fail: %v", err))
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Parse step 2 fail: %v — body=%.300s", err, loginRespBody), Password: password, UID: regResp.NewUserID}
	}
	if loginResp.Error != nil && loginResp.Error.Message != "" {
		notify(fmt.Sprintf("[s399] Step 2 server error: %s", loginResp.Error.Message))
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Step 2: %s", loginResp.Error.Message), Password: password, UID: regResp.NewUserID}
	}
	if loginResp.AccessToken == "" {
		notify("[s399] Step 2: missing access_token")
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Step 2 no token — body=%.300s", loginRespBody), Password: password, UID: regResp.NewUserID}
	}

	// Compose cookie string từ session_cookies array
	cookieStr := buildCookieString(loginResp.SessionCookies)

	notify(fmt.Sprintf("[s399] Step 2 OK — token len=%d, %d cookies", len(loginResp.AccessToken), len(loginResp.SessionCookies)))

	// Datr pool integration
	datr := extractDatrFromCookies(loginResp.SessionCookies)
	if datr != "" && SharedPool != nil {
		if SharedPool.AddDatrRaw(datr) {
			notify(fmt.Sprintf("[Pool] Datr mới: %s... (pool size: %d)", safeShort(datr, 10), SharedPool.Size()))
		}
	}

	return &instagram.RegResult{
		Success:        true,
		UID:            regResp.NewUserID,
		Cookie:         cookieStr,
		AccessToken:    loginResp.AccessToken,
		Password:       password,
		Message:        fmt.Sprintf("Register OK — UID: %s (S399 %s)", regResp.NewUserID, profile.Device.Model),
		UserAgent:      profile.S399UA,
		DeviceID:       deviceID,
		FamilyDeviceID: familyDeviceID,
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// convertBirthdayToYMD: "DD-MM-YYYY" → "YYYY-MM-DD". Empty → fallback "1990-01-01".
func convertBirthdayToYMD(in string) string {
	in = strings.TrimSpace(in)
	if in == "" {
		return "1990-01-01"
	}
	parts := strings.Split(in, "-")
	if len(parts) != 3 {
		return "1990-01-01"
	}
	if len(parts[2]) == 4 { // DD-MM-YYYY
		return parts[2] + "-" + parts[1] + "-" + parts[0]
	}
	return in // already YMD or unknown — pass through
}

func buildCookieString(cookies []sessionCookie) string {
	parts := make([]string, 0, len(cookies))
	for _, c := range cookies {
		if c.Name != "" && c.Value != "" {
			parts = append(parts, c.Name+"="+c.Value)
		}
	}
	return strings.Join(parts, "; ")
}

func extractDatrFromCookies(cookies []sessionCookie) string {
	for _, c := range cookies {
		if c.Name == "datr" {
			return c.Value
		}
	}
	return ""
}

func seedCookieString(sess *session, cookieStr string) {
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
		if name == "c_user" || name == "xs" || name == "datr" {
			continue
		}
		sess.addCookie(name, strings.TrimSpace(kv[1]))
	}
}

func countryFromPhone(phone string) string {
	if phone == "" {
		return ""
	}
	if p, ok := fakeinfo.FindCountryByPhonePrefix(phone); ok {
		return p.CountryCode
	}
	return ""
}
