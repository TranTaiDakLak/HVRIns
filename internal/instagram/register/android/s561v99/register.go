// register.go — s561v99: Type Reg 2 (step-by-step) registration.
// Platform "s561v99": same UA as s561v2 (FBAV/561.0.0.42.67), uses 8-step flow.
//
// KHÔNG import từ register/s561v3 — đây là package riêng biệt.
package s561v99

import (
	"context"
	"fmt"
	"strings"
	"time"

	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
)

// ─── Registerer ───────────────────────────────────────────────────────────────

type Registerer struct{}

func (r *Registerer) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	return registerAccount(ctx, input, onStatus)
}

func init() {
	instagram.RegisterPlatformRegisterer(instagram.PlatformS561V99, func() instagram.Registerer {
		return &Registerer{}
	})
}

// ─── WorkerContext ────────────────────────────────────────────────────────────

type WorkerContext struct {
	sess        *session
	profile     S560Profile
	platform    string
	countryCode string
}

func NewWorkerContext(proxyStr, countryCode string) (*WorkerContext, error) {
	sess, err := newSession(proxyStr)
	if err != nil {
		return nil, err
	}
	profile := BuildProfileForPlatform("s561v2", countryCode)
	return &WorkerContext{sess: sess, profile: profile, platform: "s561v2", countryCode: countryCode}, nil
}

func (w *WorkerContext) Close() {
	if w != nil && w.sess != nil {
		w.sess.client.CloseIdleConnections()
	}
}

func (w *WorkerContext) Profile() S560Profile { return w.profile }
func (w *WorkerContext) UserAgent() string     { return w.profile.S560UA }

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

// SetUAOptions rebuild profile UA — addVirtualSpecs=true prepend Dalvik prefix.
func (w *WorkerContext) SetUAOptions(addVirtualSpecs bool) {
	if w == nil {
		return
	}
	w.profile = BuildProfileForPlatformWithUA(w.platform, w.countryCode, addVirtualSpecs)
}

func (w *WorkerContext) SetUA(ua string) {
	if w == nil || ua == "" {
		return
	}
	w.profile.S560UA = ua
}

// ─── Registration flow ────────────────────────────────────────────────────────

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

// Register calls RegisterV2 (step-by-step flow) — this is the primary registration path.
func (w *WorkerContext) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	return w.RegisterV2(ctx, input, onStatus)
}

// ─── RegisterV2 ───────────────────────────────────────────────────────────────

func (w *WorkerContext) RegisterV2(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(msg)
		}
	}

	profile := w.profile
	sess := w.sess

	// ── Seed / profile override ────────────────────────────────────────────────
	var seed Seed
	if input != nil && input.TutDatr != "" {
		seed = ParseSeed(input.TutDatr)
		if seed.Mode == SeedModeInitialAccount && input.CookieInitialMethod != "ck" {
			if seed.CookieString != "" {
				seed = Seed{Raw: seed.Raw, Mode: SeedModeFullCookie, CookieString: seed.CookieString, Datr: seed.Datr, SourceLabel: "file_mode(datr_only)"}
			} else if seed.Datr != "" {
				seed = Seed{Raw: seed.Raw, Mode: SeedModeDatrOnly, Datr: seed.Datr, SourceLabel: "file_mode(datr_only)"}
			} else {
				seed = Seed{Mode: SeedModeNone, SourceLabel: "file_mode(skipped_uid_pass)"}
			}
		}
	}
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
			profile.S560UA = input.UserAgent
		}
		if input.UseOriginalUA {
			if input.OriginalSim.OperatorName != "" {
				profile.Sim = input.OriginalSim
			}
			profile.Locale = "en_GB"
			profile.FullRegProfile.Device.Brand = "samsung"
			profile.FullRegProfile.Device.Model = "SM-S911B"
			profile.FullRegProfile.Device.OSVersion = "15"
		}
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
			s, f, u, used := SharedPool.GetStats(poolDatr)
			notify(fmt.Sprintf("[s561v99] New initial %s (used %d | S/F/U: %d/%d/%d)", poolDatr, used, s, f, u))
		} else {
			notify(fmt.Sprintf("[s561v99] ⚠️ Pool EMPTY (slot=%d) — reg KHÔNG có datr!", slotIdx))
		}
	} else if profile.MachineID != "" && SharedPool != nil {
		s, f, u, used := SharedPool.GetStats(profile.MachineID)
		notify(fmt.Sprintf("[s561v99] Dùng datr %s | used=%d S/F/U=%d/%d/%d", profile.MachineID, used, s, f, u))
	}
	if profile.MachineID != "" && SharedPool != nil {
		defer SharedPool.IncrementUsage(profile.MachineID)
	}

	password := fakeinfo.RandomPassword()
	if input != nil && input.Password != "" {
		password = input.Password
	}

	contactpoint := ""
	contactpointType := "phone"
	if input != nil {
		if input.Email != "" {
			contactpoint = input.Email
			contactpointType = "email"
		} else if input.Phone != "" {
			contactpoint = input.Phone
		}
	}
	if contactpoint == "" {
		return &instagram.RegResult{Success: false, Message: "Missing contactpoint (phone/email)"}
	}

	locale := profile.Locale
	if locale == "" {
		locale = "en_US"
	}

	sv5, sv67 := sv5Eml, sv6Eml
	if contactpointType == "phone" {
		sv5, sv67 = sv5Ph, sv6Ph
	}

	delayStep := 0
	if input != nil {
		delayStep = input.DelayStep
	}

	datrShown := "NONE"
	if profile.MachineID != "" {
		datrShown = safeShort(profile.MachineID, 12)
	}
	notify(fmt.Sprintf("[s561v99] Start — %s %s | %s | %s | datr=%s | seed=%s | delay=%dms",
		profile.FirstName, profile.LastName, contactpoint, profile.Device.Model, datrShown, seed.SourceLabel, delayStep))
	sess.clearCookies()

	graphURL := instagram.BaseURLBGraph + "/graphql"

	postStep := func(stepName string, p stepParams) (string, error) {
		if delayStep > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(delayStep) * time.Millisecond):
			}
		}
		body := buildStepBody(p, locale)
		hdrs := buildStepHeaders(profile, p.friendlyName)
		resp, err := sess.postGzip(ctx, graphURL, body, hdrs)
		if err != nil {
			return "", fmt.Errorf("%s HTTP: %w", stepName, err)
		}
		if err := checkStepResp(resp, stepName); err != nil {
			datrTag := "NONE"
			if profile.MachineID != "" {
				datrTag = safeShort(profile.MachineID, 12)
			}
			return resp, fmt.Errorf("%s [datr=%s]: %w", stepName, datrTag, err)
		}
		notify(fmt.Sprintf("[s561v99] %s OK (%d bytes)", stepName, len(resp)))
		return resp, nil
	}

	commonCP := func(ri riState) stepParams {
		return stepParams{
			profile:          profile,
			contactpoint:     contactpoint,
			contactpointType: contactpointType,
			ri:               ri,
		}
	}

	// Step 1: Name
	p1 := commonCP(riState{})
	p1.friendlyName = s1FriendlyName
	p1.appID = s1AppID
	p1.currentStep = 1
	p1.cipSpecific = cipS1(profile.FirstName, profile.LastName)
	p1.screenVisited = sv1
	if _, err := postStep("Step1(name)", p1); err != nil {
		return regErr(err.Error(), password)
	}

	// Step 2: Birthday
	p2 := commonCP(riState{nameSet: true})
	p2.friendlyName = s2FriendlyName
	p2.appID = s2AppID
	p2.currentStep = 2
	p2.cipSpecific = cipS2(profile.Birthday)
	p2.screenVisited = sv2
	if _, err := postStep("Step2(birthday)", p2); err != nil {
		return regErr(err.Error(), password)
	}

	// Step 3: Gender
	p3 := commonCP(riState{nameSet: true, bdaySet: true})
	p3.friendlyName = s3FriendlyName
	p3.appID = s3AppID
	p3.currentStep = 3
	p3.cipSpecific = cipS3(profile.Gender)
	p3.screenVisited = sv3
	if _, err := postStep("Step3(gender)", p3); err != nil {
		return regErr(err.Error(), password)
	}

	// Step 4: DISABLED — body format chưa verify đúng, gây 100% fail tại Step 7 (create-check denied).
	// Để lại buildStep4Body() trong steps.go để re-enable khi có packet capture chuẩn.
	// _ = s4FriendlyName // silence unused warning

	// Step 5: Contactpoint submit (email / phone)
	var cip5 string
	if contactpointType == "email" {
		cip5 = cipS5Email(contactpoint)
	} else {
		cip5 = cipS5Phone(contactpoint)
	}
	p5 := commonCP(riState{nameSet: true, bdaySet: true, genderSet: true})
	p5.friendlyName = s5FriendlyName
	p5.appID = s5AppID
	p5.currentStep = 4 // traffic xác nhận: step5 gửi current_step=4
	p5.cipSpecific = cip5
	p5.screenVisited = sv5
	if _, err := postStep("Step5(contactpoint)", p5); err != nil {
		return regErr(err.Error(), password)
	}

	// Encrypt password (#PWD_FB4A:2 RSA+AES-GCM, fallback to :0 plaintext)
	notify("[s561v99] Fetching pwd key for encryption...")
	encPassword := ""
	if pk := fetchPwdKey(ctx, sess, profile); pk.OK {
		encPassword = EncryptPassword(password, pk.PublicKey, pk.KeyID)
		if encPassword != "" {
			notify("[s561v99] Password encrypted (#PWD_FB4A:2)")
		}
	}
	if encPassword == "" {
		ts := time.Now().Unix()
		encPassword = fmt.Sprintf("#PWD_FB4A:0:%d:%s", ts, password)
		notify("[s561v99] Password plaintext fallback (#PWD_FB4A:0)")
	}

	// Step 6: Password
	p6 := commonCP(riState{nameSet: true, bdaySet: true, genderSet: true, cpSet: true})
	p6.friendlyName = s6FriendlyName
	p6.appID = s6AppID
	p6.currentStep = 5 // traffic xác nhận: step6 gửi current_step=5
	p6.cipSpecific = cipS6Pwd(encPassword)
	p6.screenVisited = sv67
	p6.encPassword = encPassword
	if _, err := postStep("Step6(password)", p6); err != nil {
		return regErr(err.Error(), password)
	}

	// Step 7: Create account check
	riAll := riState{nameSet: true, bdaySet: true, genderSet: true, cpSet: true, pwdSet: true}
	p7 := commonCP(riAll)
	p7.friendlyName = S560FriendlyName
	p7.appID = s78AppID
	p7.currentStep = 8 // traffic: step7 gửi current_step=8
	p7.screenVisited = sv67
	p7.encPassword = encPassword
	if _, err := postStep("Step7(create-check)", p7); err != nil {
		return regErr(err.Error(), password)
	}

	// Step 8: Create account final
	p8 := commonCP(riAll)
	p8.friendlyName = S560FriendlyName
	p8.appID = s78AppID
	p8.currentStep = 8
	p8.screenVisited = sv67
	p8.encPassword = encPassword
	respBody, err := func() (string, error) {
		body := buildStepBody(p8, locale)
		hdrs := buildStepHeaders(profile, S560FriendlyName)
		return sess.postGzip(ctx, graphURL, body, hdrs)
	}()
	if err != nil {
		notify(fmt.Sprintf("[s561v99] Step8 HTTP error: %v", err))
		if profile.MachineID != "" && SharedPool != nil {
			SharedPool.RecordResult(profile.MachineID, "unknown")
		}
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Step8 HTTP: %v", err), Password: password}
	}
	notify(fmt.Sprintf("[s561v99] Step8 response (%d bytes)", len(respBody)))

	parsed, err := parseRegisterResponse(respBody, locale)
	if err != nil {
		if profile.MachineID != "" && SharedPool != nil {
			outcome := "unknown"
			if strings.Contains(strings.ToLower(err.Error()), "checkpoint") {
				outcome = "checkpoint"
			} else if parsed != nil && parsed.Blocked {
				outcome = "fail"
			}
			SharedPool.RecordResult(profile.MachineID, outcome)
		}
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("V2 failed: %v", err), Password: password}
	}

	notify(fmt.Sprintf("[s561v99] Success! UID=%s Token=%s...",
		parsed.UID, parsed.AccessToken[:smin(len(parsed.AccessToken), 20)]))

	if parsed.DATR != "" && SharedPool != nil {
		if SharedPool.AddDatrRaw(parsed.DATR) {
			notify(fmt.Sprintf("[Pool] Datr mới (V99): %s (pool=%d)", parsed.DATR[:smin(len(parsed.DATR), 10)], SharedPool.Size()))
		}
	}
	if profile.MachineID != "" && SharedPool != nil {
		SharedPool.RecordResult(profile.MachineID, "success")
		s, f, u, used := SharedPool.GetStats(profile.MachineID)
		notify(fmt.Sprintf("[Pool] Datr %s → success V99 (used=%d S/F/U=%d/%d/%d)", profile.MachineID, used, s, f, u))
	}

	return &instagram.RegResult{
		Success:        true,
		UID:            parsed.UID,
		Cookie:         parsed.Cookie,
		AccessToken:    parsed.AccessToken,
		Password:       password,
		Message:        fmt.Sprintf("Register OK (V99) — UID: %s (%s)", parsed.UID, profile.Device.Model),
		UserAgent:      profile.S560UA,
		DeviceID:       profile.DeviceID,
		FamilyDeviceID: profile.FamilyDeviceID,
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func smin(a, b int) int {
	if a < b {
		return a
	}
	return b
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

func regErr(msg, password string) *instagram.RegResult {
	return &instagram.RegResult{Success: false, Message: msg, Password: password}
}

