// register.go — Android V3 Registerer + WorkerContext + Register flow orchestrator.
//
// File này gộp 3 file cũ:
//   - register.go    → Registerer interface + init()
//   - worker_ctx.go  → WorkerContext (keep-session + device pinning)
//   - steps.go       → RegisterAccount wrapper + (w *WorkerContext).Register + fetchXZeroEH
//
// Flow port từ C# FacebookRegisterAPIAndroid.Register() + GetXZeroEh:
//
//	gen profile → pwd_key_fetch → encrypt password → POST /graphql create.account.async
//	→ parse response → fetch X-Zero-EH → build RegResult
package android

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	cookiestore "HVRIns/internal/cookie"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
)

// ─── Registerer interface + init ─────────────────────────────────────────────

// Registerer implements instagram.Registerer for Android native app API.
type Registerer struct{}

// Register thực hiện đăng ký tài khoản qua Android API.
func (r *Registerer) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	return RegisterAccount(ctx, input, onStatus)
}

func init() {
	instagram.RegisterPlatformRegisterer(instagram.PlatformAndroid, func() instagram.Registerer {
		return &Registerer{}
	})
}

// ─── WorkerContext (keep-session + device pinning) ───────────────────────────
//
// C# equivalent: `FacebookAccountModel` cached trong thread + `IHttpRequestClient`
// shared across multiple `RegisterWithKeepHttpSession` calls.
//
// Port tương tự S23 WorkerContext — áp dụng cho Android V3 API:
//   - Pin device/UA/Locale/DeviceID/FamilyDeviceID/Sim qua lifetime 1 goroutine worker.
//   - Reuse HTTP session (TCP pool + pwd_key result caching optional).
//   - N regs trong cùng worker share cùng fingerprint → FB trust hơn.

// WorkerContext là per-goroutine state cho Android V3 keep-session reg pattern.
type WorkerContext struct {
	sess    *session
	profile fakeinfo.FullRegProfile // device/UA/DeviceID/Sim fingerprint — pinned
}

// NewWorkerContext tạo context mới cho 1 worker goroutine.
// proxyStr: proxy đã render session.
// countryCode: "" = random SIM, "VN"/"US"/... = pin theo country.
func NewWorkerContext(proxyStr, countryCode string) (*WorkerContext, error) {
	sess, err := newSession(proxyStr)
	if err != nil {
		return nil, err
	}
	profile := fakeinfo.BuildFullRegProfile(countryCode)
	return &WorkerContext{sess: sess, profile: profile}, nil
}

// Close giải phóng idle TCP connections của HTTP client.
func (w *WorkerContext) Close() {
	if w != nil && w.sess != nil {
		w.sess.client.CloseIdleConnections()
	}
}

// Profile trả về pinned profile.
func (w *WorkerContext) Profile() fakeinfo.FullRegProfile { return w.profile }

// SetLocale override locale (C# LocaleFake=random → RandomLocale).
func (w *WorkerContext) SetLocale(locale string) {
	if w != nil && locale != "" {
		w.profile.Locale = locale
	}
}

// SetConnectionType override Xfb_connection_type (C# SimNetworkType).
func (w *WorkerContext) SetConnectionType(ct string) {
	if w != nil && ct != "" {
		w.profile.ConnectionType = ct
	}
}

// SetUAOptions rebuild UA với addVirtualSpecs — khi true prepend Dalvik prefix.
func (w *WorkerContext) SetUAOptions(addVirtualSpecs bool) {
	if w == nil {
		return
	}
	carrier := w.profile.Sim.OperatorName
	w.profile.UserAgent = fakeinfo.BuildAndroidUAWithOpts(
		w.profile.Device, w.profile.Locale, carrier,
		w.profile.FbVersion, w.profile.FbBuildNum,
		addVirtualSpecs, false,
	)
}

// SetUA override UserAgent bằng chuỗi raw (dùng khi UseRawUa = true).
func (w *WorkerContext) SetUA(ua string) {
	if w == nil || ua == "" {
		return
	}
	w.profile.UserAgent = ua
}

// ─── Register orchestrator ───────────────────────────────────────────────────
//
// Mapping từ C#: FacebookRegisterAPIAndroid.Register() + GetXZeroEh()
// Flow: gen profile → encrypt password → POST graphql → parse → fetch X-Zero-EH

// RegisterAccount thực hiện toàn bộ flow register Android API cho 1 account
// (single-shot wrapper). Workers muốn keep-session nên dùng WorkerContext.Register() trực tiếp.
func RegisterAccount(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	proxyStr := ""
	countryCode := ""
	if input != nil {
		proxyStr = input.Proxy
		if p, ok := fakeinfo.FindCountryByPhonePrefix(input.Phone); ok {
			countryCode = p.CountryCode
		}
	}
	wctx, err := NewWorkerContext(proxyStr, countryCode)
	if err != nil {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Create session: %v", err)}
	}
	defer wctx.Close()
	return wctx.Register(ctx, input, onStatus)
}

// Register sử dụng session + profile đã pin trong WorkerContext.
// Reuse TCP pool + device fingerprint cho N regs trong cùng worker goroutine.
// Port C# RegisterWithKeepHttpSession pattern.
func (w *WorkerContext) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(msg)
		}
	}

	// Deep copy pinned profile — tránh mutate khi override từ input.
	profile := w.profile
	sess := w.sess

	// Clear cookie jar từ reg trước (tránh cookie pollution giữa regs).
	sess.clearCookies()

	// Override từ input nếu có
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
		}
		if input.TutDatr != "" {
			profile.MachineID = normalizeInitialDatr(input.TutDatr)
		}
	}

	// Cookie initial pool: lấy datr từ partition của slot này (C#: GetPerfectMachineId)
	slotIdx := 0
	if input != nil {
		slotIdx = input.SlotIdx
	}
	if profile.MachineID == "" && SharedPool != nil {
		if poolDatr := SharedPool.GetNext(slotIdx); poolDatr != "" {
			profile.MachineID = poolDatr
			s, f, u, used := SharedPool.GetStats(poolDatr)
			notify(fmt.Sprintf("[Android] New initial %s | used=%d S/F/U=%d/%d/%d",
				poolDatr, used, s, f, u))
		}
	} else if profile.MachineID != "" && SharedPool != nil {
		s, f, u, used := SharedPool.GetStats(profile.MachineID)
		notify(fmt.Sprintf("[Android] Dùng datr %s | used=%d S/F/U=%d/%d/%d",
			profile.MachineID, used, s, f, u))
	}
	// IncrementUsage sau mỗi lần reg (C#: AddOrIncrementMachineIdUsage)
	if profile.MachineID != "" && SharedPool != nil {
		defer SharedPool.IncrementUsage(profile.MachineID)
	}
	if profile.MachineID != "" {
		sess.addCookie("datr", profile.MachineID)
	}

	// Password: dùng từ input hoặc generate
	password := fakeinfo.RandomPassword()
	if input != nil && input.Password != "" {
		password = input.Password
	}

	// Phone/Email contact point — C# ModeReg (0=Mail, 1=Phone)
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
		return &instagram.RegResult{Success: false, Message: "Thiếu contactpoint (phone/email)"}
	}

	// Tag platform gọn — phone/email đã có cột riêng.
	_ = contactpointType
	_ = contactpoint
	notify(fmt.Sprintf("[Android] Bắt đầu reg — %s %s | %s | %s",
		profile.FirstName, profile.LastName, contactpoint, profile.Locale))
	notify(fmt.Sprintf("[Android] Device: %s %s | OS %s | UA: %s", profile.Device.Brand, profile.Device.Model, profile.Device.OSVersion, truncStr(profile.UserAgent, 60)))

	// === Session đã sẵn trong WorkerContext, reuse cho pwdKey → register → xzero → logout ===
	// (sess = w.sess ở đầu; cookies đã clear)

	// === Step 3: GetPwdKey → encrypt password (V3). Fallback plaintext nếu thất bại. ===
	locale := profile.Locale
	if locale == "" {
		locale = "en_US"
	}
	var encPassword string
	pk := GetPwdKey(ctx, sess, profile)
	if pk.OK {
		encPassword = EncryptPassword(password, pk.PublicKey, pk.KeyID)
		if encPassword != "" {
			notify("[Android] Password encrypted (RSA+AES-GCM, V3)")
		}
	}
	if encPassword == "" {
		ts := time.Now().Unix()
		encPassword = fmt.Sprintf("#PWD_FB4A:0:%d:%s", ts, password)
		notify("[Android] Password encrypted (plaintext fallback)")
	}

	// === Step 4: Build request + POST register ===
	body := buildRegisterBody(profile, encPassword, contactpoint, contactpointType, locale)
	headers := buildRegisterHeaders(profile)

	datrInfo := ""
	if profile.MachineID != "" && SharedPool != nil {
		s, f, u, used := SharedPool.GetStats(profile.MachineID)
		datrInfo = fmt.Sprintf(" | datr=%s | used=%d S/F/U=%d/%d/%d",
			profile.MachineID, used, s, f, u)
	}
	notify(fmt.Sprintf("[Android] POST graphql V3%s...", datrInfo))
	respBody, err := sess.post(ctx, instagram.BaseURLBGraph+"/graphql", body, headers)
	if err != nil {
		notify(fmt.Sprintf("[Android] HTTP error: %v", err))
		if profile.MachineID != "" && SharedPool != nil {
			SharedPool.RecordResult(profile.MachineID, "unknown")
		}
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("HTTP error: %v", err), Password: password}
	}

	// Chỉ save debug khi có DebugDir (không tự động nữa)
	if input != nil && input.DebugDir != "" {
		debugTS := time.Now().Format("150405")
		saveDebug(input.DebugDir, fmt.Sprintf("response_%s.txt", debugTS), respBody)
	}

	// === Step 5: Parse response ===
	notify(fmt.Sprintf("[Android] Response (%d bytes)", len(respBody)))

	parsed, err := parseRegisterResponse(respBody, profile.Locale)
	if err != nil {
		errMsg := fmt.Sprintf("Register failed: %v", err)
		notify(fmt.Sprintf("[Android] %s", errMsg))
		if profile.MachineID != "" && SharedPool != nil {
			outcome := "unknown"
			if strings.Contains(strings.ToLower(errMsg), "checkpoint") {
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
		return &instagram.RegResult{Success: false, Message: errMsg, Password: password}
	}

	notify(fmt.Sprintf("[Android] Thành công! UID=%s Token=%s...", parsed.UID, truncStr(parsed.AccessToken, 20)))

	// === Step 6: Fetch eligibility_hash (X-Zero-EH) ===
	// C#: RandomUtils.SleepRandom(1, 2) — chờ FB xử lý account trước khi gọi XZeroEH
	time.Sleep(time.Duration(1000+rand.Intn(1000)) * time.Millisecond)

	xZeroEH := ""
	if parsed.AccessToken != "" {
		notify("[Android] Lấy eligibility_hash (X-Zero-EH)...")
		xZeroEH = fetchXZeroEH(ctx, sess, profile, parsed.AccessToken, locale)
		if xZeroEH != "" {
			notify(fmt.Sprintf("[Android] X-Zero-EH: %s", truncStr(xZeroEH, 20)))
		}
	}

	// Thu datr từ reg thành công → thêm vào pool (C#: TryAddNewDatrToPool)
	if parsed.DATR != "" && SharedPool != nil {
		if SharedPool.AddDatrRaw(parsed.DATR) {
			notify(fmt.Sprintf("[Pool] Datr mới: %s... (pool size: %d)", truncStr(parsed.DATR, 10), SharedPool.Size()))
		}
	}
	// Track success cho datr đang dùng
	if profile.MachineID != "" && SharedPool != nil {
		SharedPool.RecordResult(profile.MachineID, "success")
		s, f, u, used := SharedPool.GetStats(profile.MachineID)
		notify(fmt.Sprintf("[Pool] Datr %s → success (used=%d S/F/U=%d/%d/%d)",
			profile.MachineID, used, s, f, u))
	}

	// === Build result ===
	cookie := parsed.Cookie
	if xZeroEH != "" {
		cookie += ";x_zero_eh=" + xZeroEH
	}

	return &instagram.RegResult{
		Success:        true,
		UID:            parsed.UID,
		Cookie:         cookie,
		AccessToken:    parsed.AccessToken,
		Password:       password,
		Message:        fmt.Sprintf("Register OK — UID: %s", parsed.UID),
		UserAgent:      profile.UserAgent,
		DeviceID:       profile.DeviceID,
		FamilyDeviceID: profile.FamilyDeviceID,
	}
}

// fetchXZeroEH lấy eligibility_hash sau khi register thành công.
// Dùng cùng session (cookie jar + proxy) với register để match C# pattern.
func fetchXZeroEH(ctx context.Context, sess *session, profile fakeinfo.FullRegProfile, accessToken, locale string) string {
	body := buildXZeroEHBody(profile, accessToken, locale)
	headers := buildBatchHeaders(profile, accessToken)

	url := fmt.Sprintf("%s/?include_headers=false&decode_body_json=false&streamable_json_response=true&locale=%s",
		instagram.BaseURLBGraph, profile.Locale)

	respBody, err := sess.post(ctx, url, body, headers)
	if err != nil {
		return ""
	}
	return parseXZeroEHResponse(respBody)
}

// saveDebug lưu debug content vào file
func saveDebug(dir, filename, content string) {
	os.MkdirAll(dir, 0755)
	os.WriteFile(filepath.Join(dir, filename), []byte(content), 0644)
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func normalizeInitialDatr(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if datr := cookiestore.ExtractDatr(raw); datr != "" {
		return datr
	}
	if strings.ContainsAny(raw, "|=;") {
		return ""
	}
	return raw
}
