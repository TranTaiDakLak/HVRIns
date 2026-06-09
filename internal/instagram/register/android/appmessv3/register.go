// Package appmv3 â€” Samsung Galaxy S23 + Facebook API 565 registration.
// Platform "appmv3": FBAV/565.0.0.0.28, new bloks_versioning_id/doc_id, gzip body,
// bá» appnetsession/tasos/qpl headers, thÃªm x-fb-rmd/x-zero-eh/x-zero-state,
// theme_params=[XMDS three_neutral_gray + FDS empty], is_push_on=false.
//
// KHÃ”NG Ä‘á»¥ng Ä‘áº¿n register/s557 hay register/s558 â€” Ä‘Ã¢y lÃ  platform riÃªng biá»‡t.
package appmessv3

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
)

// â”€â”€â”€ Registerer â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

type Registerer struct{ platform string }

func (r *Registerer) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	plat := r.platform
	if plat == "" {
		plat = "appmv3"
	}
	return registerAccount(ctx, plat, input, onStatus)
}

func init() {
	// key factory → internal platform (versions.go). Thêm version = thêm 1 dòng.
	for key, plat := range map[string]string{
		instagram.PlatformAppMV3:     "appmv3",
		instagram.PlatformAppMV3_535: "appmv3535",
		instagram.PlatformAppMV3_545: "appmv3545",
		instagram.PlatformAppMV3_555: "appmv3555",
		instagram.PlatformAppMV3_563: "appmv3563",
		instagram.PlatformAppMV3_564: "appmv3564",
		instagram.PlatformAppMV3_565: "appmv3565",
		instagram.PlatformAppMV3_525: "appmv3525",
		instagram.PlatformAppMV3_515: "appmv3515",
		instagram.PlatformAppMV3_505: "appmv3505",
		instagram.PlatformAppMV3_490: "appmv3490",
	} {
		p := plat
		instagram.RegisterPlatformRegisterer(key, func() instagram.Registerer { return &Registerer{platform: p} })
	}
}

// â”€â”€â”€ WorkerContext â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

type WorkerContext struct {
	sess        *session
	profile     AppMV3Profile
	platform    string
	countryCode string
}

// NewWorkerContext — giữ signature cũ cho app (default platform appmv3 = 530).
func NewWorkerContext(proxyStr, countryCode string) (*WorkerContext, error) {
	return NewWorkerContextPlatform(proxyStr, countryCode, "appmv3")
}

// NewWorkerContextPlatform — worker context theo platform/version (appmv3 / appmv3535 / appmv3545).
func NewWorkerContextPlatform(proxyStr, countryCode, platform string) (*WorkerContext, error) {
	if platform == "" {
		platform = "appmv3"
	}
	sess, err := newSession(proxyStr)
	if err != nil {
		return nil, err
	}
	profile := BuildProfileForPlatform(platform, countryCode)
	return &WorkerContext{sess: sess, profile: profile, platform: platform, countryCode: countryCode}, nil
}

func (w *WorkerContext) Close() {
	if w != nil && w.sess != nil {
		w.sess.client.CloseIdleConnections()
	}
}

func (w *WorkerContext) Profile() AppMV3Profile { return w.profile }
func (w *WorkerContext) UserAgent() string      { return w.profile.AppMV3UA }

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

// SetUAOptions rebuild profile UA â€” addVirtualSpecs=true prepend Dalvik prefix.
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
	w.profile.AppMV3UA = ua
}

// â”€â”€â”€ Registration flow â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func registerAccount(ctx context.Context, platform string, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	proxyStr := ""
	countryCode := ""
	if input != nil {
		proxyStr = input.Proxy
		countryCode = countryFromPhone(input.Phone)
	}
	wctx, err := NewWorkerContextPlatform(proxyStr, countryCode, platform)
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
			profile.AppMV3UA = input.UserAgent
		}
		// UseOriginalUA: force device/locale/SIM khá»›p vá»›i OriginalUA (SM-S911B, en_GB, samsung
		// + FBCR carrier do caller chá»n). TrÃ¡nh fingerprint mismatch giá»¯a UA (FBCR/FBLC) vÃ
		// body/headers (Sim.HNI/Locale). profile.Device shadowed bá»Ÿi S23Device â†’ dÃ¹ng
		// FullRegProfile.Device Ä‘á»ƒ truy cáº­p DeviceProfile embedded.
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
			notify(fmt.Sprintf("[appmv3] New initial %s (used %d | S/F/U: %d/%d/%d)",
				poolDatr, used, s, f, u))
		} else {
			notify(fmt.Sprintf("[appmv3] âš ï¸ Pool EMPTY (slot=%d) â€” reg KHÃ”NG cÃ³ datr!", slotIdx))
		}
	} else if profile.MachineID != "" && SharedPool != nil {
		s, f, u, used := SharedPool.GetStats(profile.MachineID)
		notify(fmt.Sprintf("[appmv3] DÃ¹ng datr %s | used=%d S/F/U=%d/%d/%d",
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
	notify("[appmv3] Báº¯t Ä‘áº§u reg")

	proxyStr := ""
	if input != nil {
		proxyStr = input.Proxy
	}

	notify(fmt.Sprintf("[appmv3] Start â€” %s %s | %s | %s | seed=%s",
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
	encPassword := fmt.Sprintf("#PWD_MSGR:0:%d:%s", ts, password)
	notify("[appmv3] Password encrypted (#PWD_MSGR)")

	locale := profile.Locale
	if locale == "" {
		locale = "en_GB"
	}
	// UA Messenger Orca (ghi đè UA FB4A từ profile builder).
	if !strings.Contains(profile.AppMV3UA, "FBAN/Orca-Android") {
		profile.AppMV3UA = orcaUA(locale, profile)
	}
	if contactpointType == "email" {
		// Luồng MULTI-STEP bám capture + confirm OTP (nếu app cấp input.GetOTP).
		// RunEmailRegFlow tự retry cả flow (IP mới mỗi lần) khi create dính 300043.
		px := ""
		var getOTP func(context.Context) (string, error)
		if input != nil {
			px = input.Proxy
			getOTP = input.GetOTP
		}
		res := RunEmailRegFlow(ctx, px, profile, contactpoint, password, getOTP, 4, notify)
		if profile.MachineID != "" && SharedPool != nil {
			if res != nil && res.Success {
				SharedPool.RecordResult(profile.MachineID, "success")
			} else {
				SharedPool.RecordResult(profile.MachineID, "fail")
			}
		}
		return res
	}

	// Phone: single-shot fallback (chưa capture luồng phone multi-step).
	body := buildRegisterBody(profile, encPassword, contactpoint, contactpointType, locale)
	headers := buildHeaders(profile)
	notify(fmt.Sprintf("[appmv3] POST graphql single-shot (%s | phone)...", profile.Device.Model))
	respBody, perr := sess.postGzip(ctx, instagram.BaseURLBGraph+"/graphql", body, headers)
	if input != nil && input.DebugDir != "" {
		_ = os.MkdirAll(input.DebugDir, 0o755)
		_ = os.WriteFile(filepath.Join(input.DebugDir, "appmv3_response.txt"), []byte(respBody), 0o644)
	}
	if perr != nil {
		notify(fmt.Sprintf("[appmv3] HTTP error: %v", perr))
		if profile.MachineID != "" && SharedPool != nil {
			SharedPool.RecordResult(profile.MachineID, "unknown")
		}
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("HTTP error: %v", perr), Password: password}
	}

	notify(fmt.Sprintf("[appmv3] create.account response (%d bytes)", len(respBody)))

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
				notify(fmt.Sprintf("[Pool] Datr %s â†’ fail (used=%d S/F/U=%d/%d/%d)",
					profile.MachineID, used, s, f, u))
			}
		}
		_ = debugResp
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Register failed: %v", err), Password: password}
	}

	notify(fmt.Sprintf("[appmv3] Success! UID=%s Token=%s...", parsed.UID, parsed.AccessToken[:smin(len(parsed.AccessToken), 20)]))

	time.Sleep(time.Duration(1000+rand.Intn(1000)) * time.Millisecond)
	xzeroEh := fetchXZeroEh(ctx, sess, profile, parsed.AccessToken, profile.DeviceID)
	if xzeroEh != "unknown" && xzeroEh != "" {
		notify(fmt.Sprintf("[appmv3] X-Zero-EH: %s", xzeroEh[:smin(len(xzeroEh), 16)]))
	}

	if parsed.DATR != "" && SharedPool != nil {
		if SharedPool.AddDatrRaw(parsed.DATR) {
			notify(fmt.Sprintf("[Pool] Datr má»›i: %s... (pool size: %d)", parsed.DATR[:smin(len(parsed.DATR), 10)], SharedPool.Size()))
		}
	}
	if profile.MachineID != "" && SharedPool != nil {
		SharedPool.RecordResult(profile.MachineID, "success")
		s, f, u, used := SharedPool.GetStats(profile.MachineID)
		notify(fmt.Sprintf("[Pool] Datr %s â†’ success (used=%d S/F/U=%d/%d/%d)",
			profile.MachineID, used, s, f, u))
	}

	return &instagram.RegResult{
		Success:        true,
		UID:            parsed.UID,
		Cookie:         parsed.Cookie,
		AccessToken:    parsed.AccessToken,
		Password:       password,
		Message:        fmt.Sprintf("Register OK â€” UID: %s (S565 %s)", parsed.UID, profile.Device.Model),
		UserAgent:      profile.AppMV3UA,
		DeviceID:       profile.DeviceID,
		FamilyDeviceID: profile.FamilyDeviceID,
	}
}

// â”€â”€â”€ Helpers â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

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
