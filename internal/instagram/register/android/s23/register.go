// Package s23 — Samsung Galaxy S23 Facebook registration API
// Uses S23-specific UA, HTTP/2, gzip body, ECDSA x-meta-usdid
// Same create.account.async endpoint as generic Android but with S23 fingerprint.
//
// File này gồm:
//   - Registerer interface + init() dispatcher (platform registry)
//   - WorkerContext (per-goroutine keep-session state) — port C# RegisterWithKeepHttpSession
//   - RegisterAccount wrapper (single-shot, tự tạo WorkerContext)
//   - (w *WorkerContext).Register method — dùng session + profile đã pin
//   - Helper min + countryFromPhone
package s23

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
)

// ─── Registerer interface + dispatcher ────────────────────────────────────────

// Registerer implements instagram.Registerer for Samsung Galaxy S22/S23/S24/S25/S26.
// Platform field — set bởi factory khi register — quyết định device pool dùng để build profile.
type Registerer struct {
	Platform string // "s22" | "s23" | "s24" | "s25" | "s26" — default "s23"
}

// Register performs Samsung Galaxy registration (interface entry point).
// Device profile chọn từ pool tương ứng platform (SM-S90x cho S22, SM-S91x cho S23, ...).
func (r *Registerer) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	platform := r.Platform
	if platform == "" {
		platform = "s23"
	}
	return registerAccountForPlatform(ctx, platform, input, onStatus)
}

func init() {
	// S23 chính + các biến thể S22/S24/S25/S26 — cùng flow, KHÁC device pool (model + specs).
	instagram.RegisterPlatformRegisterer(instagram.PlatformS23, func() instagram.Registerer { return &Registerer{Platform: "s23"} })
	//instagram.RegisterPlatformRegisterer(instagram.PlatformS22, func() instagram.Registerer { return &Registerer{Platform: "s22"} })
	//instagram.RegisterPlatformRegisterer(instagram.PlatformS24, func() instagram.Registerer { return &Registerer{Platform: "s24"} })
	//instagram.RegisterPlatformRegisterer(instagram.PlatformS25, func() instagram.Registerer { return &Registerer{Platform: "s25"} })
	//instagram.RegisterPlatformRegisterer(instagram.PlatformS26, func() instagram.Registerer { return &Registerer{Platform: "s26"} })
}

// ─── WorkerContext — keep-session + device pinning ────────────────────────────
//
// C# equivalent: `FacebookAccountModel` cached trong thread + `IHttpRequestClient`
// shared across multiple RegisterWithKeepHttpSession calls.
//
// Tại sao cần:
//   - Device/UA rotation mỗi reg → FB thấy quá nhiều "new devices" từ cùng IP
//   - New HTTP session mỗi reg → TCP handshake overhead + không match C# behavior
//
// Giải pháp: 1 WorkerContext per goroutine worker, reuse cho N reg attempts.
// Session + device fingerprint (Model, UA, Density, DeviceID, FamilyDeviceID, Sim)
// được pin; field per-reg (FirstName, LastName, Password, Birthday, MachineID) được
// override từ input mỗi lần Register().

// WorkerContext là per-goroutine state dùng cho keep-session reg pattern.
//
// Sử dụng:
//
//	wctx, err := NewWorkerContext(proxyStr, countryCode)
//	if err != nil { ... }
//	defer wctx.Close()
//
//	for i := 0; i < N; i++ {
//	    result := wctx.Register(ctx, input, onStatus)
//	    ...
//	}
type WorkerContext struct {
	sess        *session   // HTTP client (TCP pool) — shared across regs
	profile     S23Profile // device/UA/DeviceID/Sim fingerprint — pinned per worker
	platform    string     // cache để SetUAOptions rebuild (s22/s23/s24/.../s26)
	countryCode string     // cache để SetUAOptions rebuild
}

// NewWorkerContext tạo context mới cho 1 worker goroutine.
// proxyStr: proxy đã render session (rotating id mới mỗi worker khác nhau).
// countryCode: "" = random, "VN"/"US"/... = pin theo country.
func NewWorkerContext(proxyStr, countryCode string) (*WorkerContext, error) {
	return NewWorkerContextForPlatform("s23", proxyStr, countryCode)
}

// NewWorkerContextForPlatform tạo WorkerContext với device pool matching platform.
// platform: "s22"/"s23"/"s24"/"s25"/"s26" — default "s23" nếu empty/unknown.
func NewWorkerContextForPlatform(platform, proxyStr, countryCode string) (*WorkerContext, error) {
	sess, err := newSession(proxyStr)
	if err != nil {
		return nil, err
	}
	profile := BuildProfileForPlatform(platform, countryCode)
	return &WorkerContext{sess: sess, profile: profile, platform: platform, countryCode: countryCode}, nil
}

// Close giải phóng idle TCP connections của HTTP client.
func (w *WorkerContext) Close() {
	if w != nil && w.sess != nil {
		w.sess.client.CloseIdleConnections()
	}
}

// Profile trả về pinned profile để caller đọc fingerprint (display UA, v.v.).
func (w *WorkerContext) Profile() S23Profile { return w.profile }

// SetLocale override locale của profile (C# LocaleFake=random → RandomLocale).
// Phải gọi trước Register() — locale được dùng trong build body + headers.
func (w *WorkerContext) SetLocale(locale string) {
	if w != nil && locale != "" {
		w.profile.Locale = locale
	}
}

// SetConnectionType override Xfb_connection_type (C# SimNetworkType: WIFI/LTE/HSDPA/unknown).
func (w *WorkerContext) SetConnectionType(ct string) {
	if w != nil && ct != "" {
		w.profile.ConnType = ct
		w.profile.ConnectionType = ct
	}
}

// SetUAOptions rebuild profile UA với addVirtualSpecs — khi true prepend Dalvik prefix.
func (w *WorkerContext) SetUAOptions(addVirtualSpecs bool) {
	if w == nil {
		return
	}
	w.profile = BuildProfileForPlatformWithUA(w.platform, w.countryCode, addVirtualSpecs)
}

// SetUA override toàn bộ S23 UA bằng chuỗi raw (dùng khi UseRawUa = true).
func (w *WorkerContext) SetUA(ua string) {
	if w == nil || ua == "" {
		return
	}
	w.profile.S23UA = ua
}

func indexOf(s, sub string) int {
	return strings.Index(s, sub)
}

// ─── Registration flow ────────────────────────────────────────────────────────

// RegisterAccount performs S23 registration (single-shot, creates own session+profile).
// Backward-compat wrapper — internally tạo WorkerContext 1 lần.
// Workers muốn keep-session nên dùng WorkerContext.Register() trực tiếp.
func RegisterAccount(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	return registerAccountForPlatform(ctx, "s23", input, onStatus)
}

// registerAccountForPlatform là entry point per-platform (s22/s23/s24/s25/s26).
// Tạo WorkerContext với device pool matching platform + gọi Register.
func registerAccountForPlatform(ctx context.Context, platform string, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	proxyStr := ""
	countryCode := ""
	if input != nil {
		proxyStr = input.Proxy
		countryCode = countryFromPhone(input.Phone)
	}
	wctx, err := NewWorkerContextForPlatform(platform, proxyStr, countryCode)
	if err != nil {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Create session failed: %v", err)}
	}
	defer wctx.Close()
	return wctx.Register(ctx, input, onStatus)
}

// Register sử dụng session + profile đã pin trong WorkerContext.
// Reuse TCP pool + device fingerprint cho N regs trong cùng worker goroutine.
func (w *WorkerContext) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(msg)
		}
	}

	// Deep copy profile — tránh mutate pinned profile khi override từ input.
	profile := w.profile
	sess := w.sess

	// === Parse seed (mồi) từ cookie_initial line ===
	// input.TutDatr có thể là: datr value, full cookie string, hoặc "uid|pass|cookie|token"
	// CookieInitialMethod="ck" → cho phép SeedModeInitialAccount (login warm).
	// method "file"/"new" → strip uid|password, chỉ lấy datr (respect user choice).
	var seed Seed
	if input != nil && input.TutDatr != "" {
		seed = ParseSeed(input.TutDatr)
		// Gate warm mode: chỉ "ck" mới được login mồi. Các mode khác: downgrade về
		// SeedModeFullCookie (nếu có cookie string) hoặc SeedModeDatrOnly.
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

	// Override from input
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
			profile.S23UA = input.UserAgent
		}
		// MachineID: dùng seed.Datr thay vì raw TutDatr (tránh full line làm datr cookie)
		if seed.Datr != "" {
			profile.MachineID = seed.Datr
		}
	}

	// Cookie pool: datr từ partition của slot này (C#: GetPerfectMachineId)
	slotIdx := 0
	if input != nil {
		slotIdx = input.SlotIdx
	}
	if profile.MachineID == "" && SharedPool != nil {
		if poolDatr := SharedPool.GetNext(slotIdx); poolDatr != "" {
			profile.MachineID = poolDatr
			s, f, u, used := SharedPool.GetStats(poolDatr)
			notify(fmt.Sprintf("[s23] New initial %s (used %d | S/F/U: %d/%d/%d)",
				poolDatr, used, s, f, u))
		} else {
			notify(fmt.Sprintf("[s23] ⚠️ Pool EMPTY (slot=%d) — reg KHÔNG có datr!", slotIdx))
		}
	} else if profile.MachineID != "" && SharedPool != nil {
		// Đã có datr từ seed/TutDatr → vẫn show stats để user theo dõi
		s, f, u, used := SharedPool.GetStats(profile.MachineID)
		notify(fmt.Sprintf("[s23] Dùng datr %s | used=%d S/F/U=%d/%d/%d",
			profile.MachineID, used, s, f, u))
	}
	// IncrementUsage sau mỗi lần reg (C#: AddOrIncrementMachineIdUsage)
	// Record outcome sau reg: success nếu parsed OK, fail nếu blocked, unknown còn lại
	if profile.MachineID != "" && SharedPool != nil {
		defer SharedPool.IncrementUsage(profile.MachineID)
	}

	password := fakeinfo.RandomPassword()
	if input != nil && input.Password != "" {
		password = input.Password
	}

	// C# ModeReg: 0=Mail (email contactpoint), 1=Phone
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
	// Tag platform gọn — phone/email đã hiện ở cột riêng, không cần lặp lại trong mỗi msg.
	notify("[s23] Bắt đầu reg")
	_ = contactpointType // reserved for future per-type logic; tag stays simple

	// proxyStr đã được pin khi NewWorkerContext; giữ biến local để reference warm session
	proxyStr := ""
	if input != nil {
		proxyStr = input.Proxy
	}

	notify(fmt.Sprintf("[s23] Start — %s %s | %s | %s | seed=%s",
		profile.FirstName, profile.LastName, contactpoint, profile.Device.Model, seed.SourceLabel))

	// === Session đã sẵn trong WorkerContext — reuse ===
	// Clear cookie jar từ reg trước (tránh cookie pollution giữa các regs).
	sess.clearCookies()

	// KHÔNG inject datr thành cookie HTTP — match C# (S23.Register L76-101 block
	// `httpRequest.AddCookie("datr", ...)` toàn bộ đã bị comment out). C# chỉ signal
	// datr qua HEADER `X-Fb-Integrity-Machine-Id` + BODY `reg_info.machine_id`.
	// Inject double (cookie + header) gây FB nhìn thấy signal lạ → flag xuống success rate.
	//
	// Nếu có SeedModeFullCookie (method=ck với line chứa cookie string), seed khác
	// (fr, sb, pas...) vẫn inject ở block dưới — chỉ skip `datr` cụ thể.

	// === Seed cookies từ full cookie string (SeedModeFullCookie) ===
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
				continue // datr đã được set ở trên, c_user/xs skip
			}
			sess.addCookie(name, strings.TrimSpace(kv[1]))
		}
	}

	// === Warm session: login initial + logout (SeedModeInitialAccount) ===
	// C# equivalent: RegisterWithRequestInitialAccount khi MachineId.Contains("|") && isFirst
	if seed.Mode == SeedModeInitialAccount {
		warmSession(ctx, sess, seed, proxyStr, notify)
	}

	// === Encrypt password ===
	ts := time.Now().Unix()
	encPassword := fmt.Sprintf("#PWD_FB4A:0:%d:%s", ts, password)
	notify("[s23] Password encrypted")

	// === Build request ===
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
	notify(fmt.Sprintf("[s23] POST graphql (%s%s)...", profile.Device.Model, datrInfo))

	// === Send register request (gzip) ===
	respBody, err := sess.postGzip(ctx, instagram.BaseURLBGraph+"/graphql", body, headers)
	if err != nil {
		notify(fmt.Sprintf("[s23] HTTP error: %v", err))
		if profile.MachineID != "" && SharedPool != nil {
			SharedPool.RecordResult(profile.MachineID, "unknown")
		}
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("HTTP error: %v", err), Password: password}
	}

	// === Parse response ===
	notify(fmt.Sprintf("[s23] Response (%d bytes)", len(respBody)))

	parsed, err := parseRegisterResponse(respBody, locale)
	if err != nil {
		// Debug: hiện response đầu để biết FB trả gì
		debugResp := respBody
		if len(debugResp) > 600 {
			debugResp = debugResp[:600]
		}
		if profile.MachineID != "" && SharedPool != nil {
			outcome := "fail"
			if strings.Contains(strings.ToLower(err.Error()), "checkpoint") {
				outcome = "checkpoint"
			} else if parsed != nil && parsed.Blocked {
				outcome = "fail"
			} else {
				outcome = "unknown"
			}
			SharedPool.RecordResult(profile.MachineID, outcome)
			if outcome == "fail" {
				s, f, u, used := SharedPool.GetStats(profile.MachineID)
				notify(fmt.Sprintf("[Pool] Datr %s → fail (used=%d S/F/U=%d/%d/%d)",
					profile.MachineID, used, s, f, u))
			}
		}
		// KHÔNG kèm raw response — response FB chứa bloks_payload rất dài (~30KB) làm UI lag + bloat memory.
		// Raw response đã được log vào file debug nếu user bật DebugDir.
		_ = debugResp
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Register failed: %v", err), Password: password}
	}

	notify(fmt.Sprintf("[s23] Success! UID=%s Token=%s...", parsed.UID, parsed.AccessToken[:min(len(parsed.AccessToken), 20)]))

	// === Fetch X-Zero-EH (eligibility hash) — port C# S23.Register L124-125 ===
	time.Sleep(time.Duration(1000+rand.Intn(1000)) * time.Millisecond)
	xzeroEh := fetchXZeroEh(ctx, sess, profile, parsed.AccessToken, profile.DeviceID)
	if xzeroEh != "unknown" && xzeroEh != "" {
		notify(fmt.Sprintf("[s23] X-Zero-EH: %s", xzeroEh[:min(len(xzeroEh), 16)]))
	}

	// Add datr to pool from successful reg
	if parsed.DATR != "" && SharedPool != nil {
		if SharedPool.AddDatrRaw(parsed.DATR) {
			notify(fmt.Sprintf("[Pool] Datr mới: %s... (pool size: %d)", parsed.DATR[:min(len(parsed.DATR), 10)], SharedPool.Size()))
		}
	}
	// Track success cho datr đang dùng (performance tracking)
	if profile.MachineID != "" && SharedPool != nil {
		SharedPool.RecordResult(profile.MachineID, "success")
		s, f, u, used := SharedPool.GetStats(profile.MachineID)
		notify(fmt.Sprintf("[Pool] Datr %s → success (used=%d S/F/U=%d/%d/%d)",
			profile.MachineID, used, s, f, u))
	}

	// Logout endpoint có sẵn qua LogoutAccount(sess, profile, accessToken, deviceID).
	// Caller (Automation) gọi khi cần — port C# pattern tách logout khỏi Register().

	cookie := parsed.Cookie

	return &instagram.RegResult{
		Success:        true,
		UID:            parsed.UID,
		Cookie:         cookie,
		AccessToken:    parsed.AccessToken,
		Password:       password,
		Message:        fmt.Sprintf("Register OK — UID: %s (S23 %s)", parsed.UID, profile.Device.Model),
		UserAgent:      profile.S23UA,
		DeviceID:       profile.DeviceID,
		FamilyDeviceID: profile.FamilyDeviceID,
	}
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// countryFromPhone maps phone prefix → ISO country code for SIM selection.
// SIM country matches phone country — normal for someone using their home SIM
// regardless of proxy IP (WiFi/VPN scenario is expected and not flagged by Facebook).
func countryFromPhone(phone string) string {
	if phone == "" {
		return ""
	}
	if p, ok := fakeinfo.FindCountryByPhonePrefix(phone); ok {
		return p.CountryCode
	}
	return ""
}
