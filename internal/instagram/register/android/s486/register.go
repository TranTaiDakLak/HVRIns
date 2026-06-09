// package s486 — Samsung Galaxy S23 + Facebook API 486 registration.
// Platform "s486": FBAV/486.0.0.0.86, new bloks_versioning_id/doc_id, gzip body,
// bỏ appnetsession/tasos/qpl headers, thêm x-fb-rmd/x-zero-eh/x-zero-state,
// theme_params=FDS-only, is_push_on=true.
//
// KHÔNG đụng đến register/s557 hay register/s558 — đây là platform riêng biệt.
package s486

import (
	"context"
	"fmt"
	"math/rand"
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
	instagram.RegisterPlatformRegisterer(instagram.PlatformS486, func() instagram.Registerer {
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
	profile := BuildProfileForPlatform("s486", countryCode)
	return &WorkerContext{sess: sess, profile: profile, platform: "s486", countryCode: countryCode}, nil
}

func (w *WorkerContext) Close() {
	if w != nil && w.sess != nil {
		w.sess.client.CloseIdleConnections()
	}
}

func (w *WorkerContext) Profile() S560Profile { return w.profile }
func (w *WorkerContext) UserAgent() string    { return w.profile.S560UA }

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

func (w *WorkerContext) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(msg)
		}
	}

	profile := w.profile
	sess := w.sess

	var seed Seed
	if input != nil && input.TutDatr != "" {
		seed = ParseSeed(input.TutDatr)
		if seed.Mode == SeedModeInitialAccount && input.CookieInitialMethod != "ck" {
			if seed.CookieString != "" {
				seed = Seed{
					Raw:          seed.Raw,
					Mode:         SeedModeFullCookie,
					CookieString: seed.CookieString,
					Datr:         seed.Datr,
					SourceLabel:  "file_mode(datr_only)",
				}
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
		// UseOriginalUA: force device/locale/SIM khớp với OriginalUA (SM-S911B, en_GB, samsung
		// + FBCR carrier do caller chọn). Tránh fingerprint mismatch giữa UA (FBCR/FBLC) và
		// body/headers (Sim.HNI/Locale). profile.Device shadowed bởi S23Device → dùng
		// FullRegProfile.Device để truy cập DeviceProfile embedded.
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
			notify(fmt.Sprintf("[s486] New initial %s (used %d | S/F/U: %d/%d/%d)",
				poolDatr, used, s, f, u))
		} else {
			notify(fmt.Sprintf("[s486] ⚠️ Pool EMPTY (slot=%d) — reg KHÔNG có datr!", slotIdx))
		}
	} else if profile.MachineID != "" && SharedPool != nil {
		s, f, u, used := SharedPool.GetStats(profile.MachineID)
		notify(fmt.Sprintf("[s486] Dùng datr %s | used=%d S/F/U=%d/%d/%d",
			profile.MachineID, used, s, f, u))
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
	notify("[s486] Bắt đầu reg")

	proxyStr := ""
	if input != nil {
		proxyStr = input.Proxy
	}

	notify(fmt.Sprintf("[s486] Start — %s %s | %s | %s | seed=%s",
		profile.FirstName, profile.LastName, contactpoint, profile.Device.Model, seed.SourceLabel))

	sess.clearCookies()

	if seed.Mode == SeedModeFullCookie && seed.CookieString != "" {
		for _, pair := range strings.Split(seed.CookieString, ";") {
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

	if seed.Mode == SeedModeInitialAccount {
		warmSession(ctx, sess, seed, proxyStr, notify)
	}

	ts := time.Now().Unix()
	encPassword := fmt.Sprintf("#PWD_FB4A:0:%d:%s", ts, password)
	notify("[s486] Password encrypted")

	locale := profile.Locale
	if locale == "" {
		locale = "en_US"
	}
	body := buildRegisterBody(profile, encPassword, contactpoint, contactpointType, locale)
	headers := buildHeaders(profile)

	datrInfo := " | datr=NONE"
	if profile.MachineID != "" && SharedPool != nil {
		s, f, u, used := SharedPool.GetStats(profile.MachineID)
		datrInfo = fmt.Sprintf(" | datr=%s | used=%d S/F/U=%d/%d/%d",
			profile.MachineID, used, s, f, u)
	}
	notify(fmt.Sprintf("[s486] POST graphql (%s%s)...", profile.Device.Model, datrInfo))

	respBody, err := sess.postGzip(ctx, instagram.BaseURLBGraph+"/graphql", body, headers)
	if err != nil {
		notify(fmt.Sprintf("[s486] HTTP error: %v", err))
		if profile.MachineID != "" && SharedPool != nil {
			SharedPool.RecordResult(profile.MachineID, "unknown")
		}
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("HTTP error: %v", err), Password: password}
	}

	notify(fmt.Sprintf("[s486] Response (%d bytes)", len(respBody)))

	parsed, err := parseRegisterResponse(respBody, locale)
	if err != nil {
		debugResp := respBody
		if len(debugResp) > 600 {
			debugResp = debugResp[:600]
		}
		if profile.MachineID != "" && SharedPool != nil {
			outcome := "unknown"
			if strings.Contains(strings.ToLower(err.Error()), "checkpoint") {
				outcome = "checkpoint"
			} else if parsed != nil && parsed.Blocked {
				outcome = "fail"
			}
			SharedPool.RecordResult(profile.MachineID, outcome)
			if outcome == "fail" {
				s, f, u, used := SharedPool.GetStats(profile.MachineID)
				notify(fmt.Sprintf("[Pool] Datr %s → fail (used=%d S/F/U=%d/%d/%d)",
					profile.MachineID, used, s, f, u))
			}
		}
		_ = debugResp
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Register failed: %v", err), Password: password}
	}

	notify(fmt.Sprintf("[s486] Success! UID=%s Token=%s...", parsed.UID, parsed.AccessToken[:smin(len(parsed.AccessToken), 20)]))

	time.Sleep(time.Duration(1000+rand.Intn(1000)) * time.Millisecond)
	xzeroEh := fetchXZeroEh(ctx, sess, profile, parsed.AccessToken, profile.DeviceID)
	if xzeroEh != "unknown" && xzeroEh != "" {
		notify(fmt.Sprintf("[s486] X-Zero-EH: %s", xzeroEh[:smin(len(xzeroEh), 16)]))
	}

	if parsed.DATR != "" && SharedPool != nil {
		if SharedPool.AddDatrRaw(parsed.DATR) {
			notify(fmt.Sprintf("[Pool] Datr mới: %s... (pool size: %d)", parsed.DATR[:smin(len(parsed.DATR), 10)], SharedPool.Size()))
		}
	}
	if profile.MachineID != "" && SharedPool != nil {
		SharedPool.RecordResult(profile.MachineID, "success")
		s, f, u, used := SharedPool.GetStats(profile.MachineID)
		notify(fmt.Sprintf("[Pool] Datr %s → success (used=%d S/F/U=%d/%d/%d)",
			profile.MachineID, used, s, f, u))
	}

	return &instagram.RegResult{
		Success:        true,
		UID:            parsed.UID,
		Cookie:         parsed.Cookie,
		AccessToken:    parsed.AccessToken,
		Password:       password,
		Message:        fmt.Sprintf("Register OK — UID: %s (S486 %s)", parsed.UID, profile.Device.Model),
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
