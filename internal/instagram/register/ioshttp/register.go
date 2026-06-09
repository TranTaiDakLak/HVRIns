// register.go — iOS HTTP (MFB) Registerer + register orchestrator.
//
// File này gộp 2 file cũ:
//   - register.go (cũ) → Registerer interface + init()
//   - steps.go           → registerAccount + UI routing + New/Old UI flows
//
// Package ioshttp — Facebook iOS/MFB (m.facebook.com) register.
// Mapping từ C#: FacebookRegisterMfbRequest (iOS HTTP variant).
// Flow: GET m.facebook.com/?locale=en_US → POST /async/wbloks/fetch/ → collect cookies.
// Dùng iPhone iOS Mobile Safari UA + Safari TLS fingerprint (không có Chrome sec-ch-ua*).
package ioshttp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	cookiestore "HVRIns/internal/cookie"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
)

// ─── Registerer interface + init ─────────────────────────────────────────────

// Registerer implements instagram.Registerer for iOS HTTP (MFB) platform.
type Registerer struct{}

// Register thực hiện đăng ký tài khoản qua iOS HTTP MFB flow.
func (r *Registerer) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	return registerAccount(ctx, input, onStatus)
}

func init() {
	instagram.RegisterPlatformRegisterer(instagram.PlatformIOS, func() instagram.Registerer {
		return &Registerer{}
	})
}

// ─── UI Route mode ───────────────────────────────────────────────────────────

// UIRouteMode controls Old/New UI routing.
type UIRouteMode string

const (
	UIRouteAuto    UIRouteMode = "auto"     // detect from page content
	UIRouteOldOnly UIRouteMode = "old_only" // force Old UI
	UIRouteNewOnly UIRouteMode = "new_only" // force New UI
)

// DefaultUIRoute is the default UI routing mode.
var DefaultUIRoute UIRouteMode = UIRouteAuto

// ─── register orchestrator ───────────────────────────────────────────────────
//
// PORT from C#: FacebookRegisterMfbRequest.RegisterWithRequestInitialAccount()
//             + RegisterWithNewUI() + RegisterWithOldUI()
//             + RegisterWithKeepHttpSession (session pool)
//
// Runtime flow matching C#:
//  1. Acquire or create session (SessionPool by proxy key)
//  2. Parse seed (Cookie Initial mode)
//  3. If seed=InitialAccount AND isFirst → login initial → logout → keep session
//  4. GET home page → detect UI
//  5. Wait 1000ms
//  6. GET /reg/ → extract tokens
//  7. Route: Old UI (regInstance present) or New UI
//  8. POST register → handle response
//  9. If success AND CookieInitial → logout new account → keep session
//  10. Store session back to pool

// registerAccount implements the full iOS HTTP (MFB) register flow.
// Maps to C#: RegisterWithKeepHttpSession → RegisterWithRequestInitialAccount.
func registerAccount(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	// Track outcome per-datr sau khi reg xong (wrapper pattern để gom mọi return).
	var pickedMachineID string
	result := doRegisterAccount(ctx, input, onStatus, &pickedMachineID)
	if pickedMachineID != "" && SharedPool != nil {
		outcome := "unknown"
		if result != nil {
			if result.Success {
				outcome = "success"
			} else if strings.Contains(strings.ToLower(result.Message), "checkpoint") {
				outcome = "checkpoint"
			} else if strings.Contains(result.Message, "Blocked") {
				outcome = "fail"
			}
		}
		SharedPool.RecordResult(pickedMachineID, outcome)
		if (outcome == "success" || outcome == "fail") && onStatus != nil {
			s, f, u, used := SharedPool.GetStats(pickedMachineID)
			trunc := pickedMachineID
			if len(trunc) > 10 {
				trunc = trunc[:10]
			}
			onStatus(fmt.Sprintf("[Pool] Datr %s... → %s (used %d | S/F/U: %d/%d/%d)",
				trunc, outcome, used, s, f, u))
		}
	}
	return result
}

// doRegisterAccount — core flow, trả về RegResult. pickedMachineIDOut nhận datr
// đang dùng để caller track stats.
func doRegisterAccount(ctx context.Context, input *instagram.RegInput, onStatus func(string), pickedMachineIDOut *string) *instagram.RegResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(msg)
		}
	}

	// === Build profile ===
	prof := fakeinfo.RandomIPhoneProfile()
	password := fakeinfo.RandomPassword()
	if input != nil && input.Password != "" {
		password = input.Password
	}
	// C# ModeReg: 0=Mail, 1=Phone
	contactpoint := ""
	cpType := "phone"
	if input != nil {
		if input.Email != "" {
			contactpoint = input.Email
			cpType = "email"
		} else if input.Phone != "" {
			contactpoint = input.Phone
		}
	}
	if contactpoint == "" {
		return &instagram.RegResult{Success: false, Message: "Thiếu contactpoint (phone/email)"}
	}
	fake := fakeinfo.RandomFakeProfile()
	if input != nil {
		if input.FirstName != "" {
			fake.FirstName = input.FirstName
		}
		if input.LastName != "" {
			fake.LastName = input.LastName
		}
		if input.Birthday != "" {
			fake.Birthday = input.Birthday
		}
		if input.Gender > 0 {
			fake.Gender = input.Gender
		}
	}
	waterfallID := uuid.New().String()
	proxyStr := ""
	if input != nil {
		proxyStr = input.Proxy
	}

	// === Parse seed ===
	var seed Seed
	if input != nil && input.TutDatr != "" {
		seed = ParseSeed(input.TutDatr)
	}
	// Nếu chưa có seed → lấy datr từ SharedPool
	machineID := seed.Datr
	slotIdx := 0
	if input != nil {
		slotIdx = input.SlotIdx
	}
	if machineID == "" && SharedPool != nil {
		if poolDatr := SharedPool.GetNext(slotIdx); poolDatr != "" {
			machineID = poolDatr
			seed = Seed{Raw: poolDatr, Mode: SeedModeDatrOnly, Datr: poolDatr, SourceLabel: "pool"}
			s, f, u, used := SharedPool.GetStats(poolDatr)
			notify(fmt.Sprintf("[iOS HTTP] New initial %s (used %d | S/F/U: %d/%d/%d)",
				poolDatr, used, s, f, u))
		}
	} else if machineID != "" && SharedPool != nil {
		s, f, u, used := SharedPool.GetStats(machineID)
		trunc := machineID
		if len(trunc) > 10 {
			trunc = trunc[:10]
		}
		notify(fmt.Sprintf("[iOS HTTP] Dùng datr %s... (used %d | S/F/U: %d/%d/%d)",
			trunc, used, s, f, u))
	}
	if machineID != "" {
		if pickedMachineIDOut != nil {
			*pickedMachineIDOut = machineID
		}
		if SharedPool != nil {
			defer SharedPool.IncrementUsage(machineID)
		}
	}

	// Tag platform gọn — phone/email đã có cột riêng.
	_ = cpType
	notify(fmt.Sprintf("[iOS HTTP] Start — %s %s | %s | seed=%s",
		fake.FirstName, fake.LastName, contactpoint, seed.SourceLabel))

	// === Session pool (C#: RegisterWithKeepHttpSession) ===
	// isFirst=true  → session mới, cần GET home + login initial nếu có seed
	// isFirst=false → reuse session đã warm, skip GET home + login initial
	proxyKey := proxyStr
	if input != nil && input.ProxyKey != "" {
		proxyKey = input.ProxyKey
	}
	isFirst := true
	var sess *session
	if SharedSessionPool != nil {
		var existing *session
		existing, isFirst = SharedSessionPool.Acquire(proxyKey)
		if !isFirst && existing != nil {
			sess = existing
			notify("[iOS HTTP] Reuse warm session from pool")
		}
	}
	if sess == nil {
		var err error
		sess, err = newSession(proxyStr)
		if err != nil {
			return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Create session failed: %v", err)}
		}
		isFirst = true
	}
	defer func() {
		if sess != nil {
			sess.client.CloseIdleConnections()
		}
	}()
	if seed.Mode != SeedModeNone {
		sess.seedFromParsed(seed)
	}

	// === Step 0: GET home page (chỉ khi isFirst) ===
	var homeHTML string
	if isFirst {
		notify("[iOS HTTP] GET home page...")
		var err error
		homeHTML, err = sess.get(ctx, "https://m.facebook.com/?locale=en_US", buildNavHeaders(prof, "", ""))
		if err != nil || homeHTML == "" {
			if SharedSessionPool != nil {
				SharedSessionPool.Remove(proxyKey)
			}
			return &instagram.RegResult{Success: false, Message: fmt.Sprintf("GET home page failed: %v", err)}
		}

		// === Login initial (C#: MachineId.Contains("|") && isFirst → LoginFb → Logout) ===
		// SeedModeInitialAccount = uid|password → đăng nhập tài khoản mồi, warm session
		if seed.Mode == SeedModeInitialAccount {
			notify(fmt.Sprintf("[iOS HTTP] Login initial account (uid=%s)...", seed.UID[:min(len(seed.UID), 8)]))
			homeTokens := parsePageTokens(homeHTML)
			loginOK := false
			// Detect UI type từ homeHTML để chọn đúng login flow
			if homeTokens.versioningID != "" {
				loginOK = loginInitialNewUI(ctx, sess, prof, seed.UID, seed.Password, homeTokens.versioningID, waterfallID, homeTokens)
			} else {
				li := reFind(homeHTML, `name="li" value="(.*?)"`, 1)
				loginOK = loginInitialOldUI(ctx, sess, prof, seed.UID, seed.Password, li, homeTokens)
			}
			if loginOK {
				notify("[iOS HTTP] Login initial OK — logout...")
				logoutInitial(ctx, sess, prof)
				notify("[iOS HTTP] Session warmed")
			} else {
				notify("[iOS HTTP] Login initial FAILED — continue with cold session")
			}
		}

		// Wait 1000ms (C#: Task.Delay(1000).Wait())
		time.Sleep(1000 * time.Millisecond)
	}

	// === Step 1: GET /reg/ ===
	// C#: PerpectMfbNavHeadersFormat3("", "") — FollowRedirects=false
	datrInfo := " | datr=NONE"
	if machineID != "" && SharedPool != nil {
		sc, f, u, used := SharedPool.GetStats(machineID)
		trunc := machineID
		if len(trunc) > 10 {
			trunc = trunc[:10]
		}
		datrInfo = fmt.Sprintf(" | datr=%s... used=%d S/F/U=%d/%d/%d", trunc, used, sc, f, u)
	}
	notify(fmt.Sprintf("[iOS HTTP] init register%s...", datrInfo))
	regHTML, err := sess.get(ctx, "https://m.facebook.com/reg/?locale=en_US", buildNavHeaders(prof, "", ""))
	if err != nil || regHTML == "" {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("GET /reg/ failed: %v", err)}
	}

	tokens := parsePageTokens(regHTML)
	if tokens.fbDtsg == "" {
		return &instagram.RegResult{Success: false, Message: "No fb_dtsg from /reg/"}
	}

	params := regParams{
		tokens:       tokens,
		firstName:    fake.FirstName,
		lastName:     fake.LastName,
		contactpoint: contactpoint,
		cpType:       cpType,
		birthday:     fake.Birthday,
		gender:       fake.Gender,
		password:     password,
		waterfallID:  waterfallID,
	}

	// === UI Route decision ===
	var result *instagram.RegResult
	uiRoute := DefaultUIRoute

	switch uiRoute {
	case UIRouteOldOnly:
		if tokens.regInstance == "" {
			notify("[iOS HTTP] WARNING: old_only mode but no regInstance — trying anyway")
		}
		notify("[iOS HTTP] Old UI — submit register ( old UI )...")
		result = registerOldUI(ctx, sess, prof, params, tokens, notify, password)
	case UIRouteNewOnly:
		notify(fmt.Sprintf("[iOS HTTP] New UI — bkv=%s", truncStr(tokens.versioningID, 16)))
		result = registerNewUI(ctx, sess, prof, params, tokens, notify, password)
	default: // auto
		if tokens.regInstance != "" {
			notify("[iOS HTTP] Old UI detected (regInstance present) — submit register ( old UI )...")
			result = registerOldUI(ctx, sess, prof, params, tokens, notify, password)
		} else {
			notify(fmt.Sprintf("[iOS HTTP] New UI detected (no regInstance) — bkv=%s", truncStr(tokens.versioningID, 16)))
			result = registerNewUI(ctx, sess, prof, params, tokens, notify, password)
		}
	}

	// === Post-register: logout new account if CookieInitial enabled ===
	// C#: if (result == Success && CookieInitial) → LogoutFb()
	if result != nil && result.Success && seed.Mode != SeedModeNone {
		notify("[iOS HTTP] post-register logout (cookie initial mode)...")
		logoutInitial(ctx, sess, prof)
	}

	// === Store / Remove session in pool ===
	// C#: RegisterWithKeepHttpSession → pool.Store() on success, pool.Remove() on fatal failure
	if SharedSessionPool != nil {
		if result != nil && result.Success {
			SharedSessionPool.Store(proxyKey, sess)
			sess = nil // pool owns the session now — skip CloseIdleConnections in defer
		} else if result == nil || isFatalError(result.Message) {
			SharedSessionPool.Remove(proxyKey)
		}
		// Non-fatal failure: leave pool entry untouched — next reg will Acquire same session
	}

	return result
}

// isFatalError checks if error message indicates a corrupted session.
func isFatalError(msg string) bool {
	return strings.Contains(msg, "Create session failed") ||
		strings.Contains(msg, "GET home page failed") ||
		strings.Contains(msg, "HTTP error")
}

// registerNewUI — New UI (wbloks) flow.
// C#: FacebookRegisterMfbRequest.RegisterWithNewUI().
func registerNewUI(ctx context.Context, sess *session, prof fakeinfo.IPhoneProfile,
	params regParams, tokens pageTokens, notify func(string), password string,
) *instagram.RegResult {
	time.Sleep(1000 * time.Millisecond)

	body := buildRegisterBody(params)
	referer := sess.finalURL
	if referer == "" {
		referer = "https://m.facebook.com/reg/"
	}

	respBody, err := sess.post(ctx, postURL(tokens.versioningID), body, buildPostHeaders(prof, referer, "https://m.facebook.com"))
	if err != nil {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("POST wbloks failed: %v", err)}
	}

	// C#: IsBlockedRegister() check BEFORE UID extraction
	if isActuallyBlocked(respBody) {
		return &instagram.RegResult{Success: false, Message: "Register failed: Blocked", Password: password}
	}

	uid := parseUID(respBody)
	if uid == "" {
		return &instagram.RegResult{Success: false, Message: "Register failed: UID not found in response", Password: password}
	}
	notify(fmt.Sprintf("[iOS HTTP] UID=%s — finalizing...", uid))

	time.Sleep(2000 * time.Millisecond)

	finalHTML, err := sess.get(ctx, "https://m.facebook.com/", buildNavHeaders(prof, "", ""))
	if err != nil {
		finalHTML = ""
	}

	if isCheckpoint(sess.finalURL) {
		cookie := sess.getCookiesStr()
		return &instagram.RegResult{Success: false, Message: "Checkpoint", Password: password, Cookie: cookie}
	}

	return extractFinalResult(sess, finalHTML, uid, password, prof.UserAgent, notify)
}

// registerOldUI — Old UI (/reg/submit/) flow.
// C#: FacebookRegisterMfbRequest.RegisterWithOldUI().
func registerOldUI(ctx context.Context, sess *session, prof fakeinfo.IPhoneProfile,
	params regParams, tokens pageTokens, notify func(string), password string,
) *instagram.RegResult {
	time.Sleep(1000 * time.Millisecond)

	body := buildOldUIRegisterBody(params, tokens)
	referer := "https://m.facebook.com/reg?soft=hjk"

	postHeaders := buildPostHeaders(prof, referer, "https://m.facebook.com")
	postHeaders = append(postHeaders,
		[2]string{"x-response-format", "JSONStream"},
		[2]string{"x-requested-with", "XMLHttpRequest"},
		[2]string{"x-fb-lsd", tokens.lsd},
		[2]string{"x-asbd-id", "359341"},
	)

	submitURL := oldUIPostURL(tokens.privacyToken)
	respBody, err := sess.post(ctx, submitURL, body, postHeaders)
	if err != nil {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("POST /reg/submit/ failed: %v", err)}
	}

	// Step 1: Blocked check (before unescape) — C#: responseStr.IsBlockedRegister()
	if isActuallyBlocked(respBody) {
		return &instagram.RegResult{Success: false, Message: "Register failed: Blocked", Password: password}
	}

	// Step 2: Double unescape — C#: Regex.Unescape × 2
	unescaped := unescapeResponse(unescapeResponse(respBody))

	// Step 3: save-device / confirmemail
	if isSaveDevice(unescaped) || isConfirmMailURL(unescaped) {
		notify("[iOS HTTP] save device...")
		return handleSaveDevice(ctx, sess, prof, tokens, respBody, password, notify)
	}

	// Step 4: MCheckpointRedirect
	if isBlockedRegister(respBody) {
		cookie := sess.getCookiesStr()
		uid := parseCUserFromCookie(cookie)
		if uid != "" && cookie != "" {
			return &instagram.RegResult{Success: false, Message: "Checkpoint", Password: password, Cookie: cookie}
		}
		return &instagram.RegResult{Success: false, Message: "Register failed: Checkpoint", Password: password}
	}

	// Step 5: Unknown — C#: SetCookieAndUid → UnknownBlockType
	cookie := sess.getCookiesStr()
	uid := parseCUserFromCookie(cookie)
	if uid != "" && cookie != "" {
		return buildSuccess(cookie, uid, password, prof.UserAgent, notify)
	}
	// Không embed raw response — response có thể chứa bloks payload rất dài.
	_ = respBody
	return &instagram.RegResult{Success: false, Message: "Register failed: unknown response", Password: password}
}

// handleSaveDevice — C#: RegisterWithOldUI save-device branch.
func handleSaveDevice(ctx context.Context, sess *session, prof fakeinfo.IPhoneProfile,
	tokens pageTokens, prevResponse, password string, notify func(string),
) *instagram.RegResult {
	loggerID := tokens.loggerID

	time.Sleep(1000 * time.Millisecond)

	refSaveDevice := fmt.Sprintf(
		"https://m.facebook.com/login/save-device/?next=%%2Fconfirmemail.php%%3Fresend&login_source=account_creation&logger_id=%s",
		loggerID,
	)
	saveDeviceHTML, err := sess.get(ctx, refSaveDevice, buildNavHeaders(prof, "https://m.facebook.com/reg/?soft=hjk", ""))
	if err != nil || saveDeviceHTML == "" {
		cookie := sess.getCookiesStr()
		uid := parseCUserFromCookie(cookie)
		if uid != "" && cookie != "" {
			return buildSuccess(cookie, uid, password, prof.UserAgent, notify)
		}
		return &instagram.RegResult{Success: false, Message: "save-device GET error", Password: password}
	}

	if isCheckpoint(sess.finalURL) {
		cookie := sess.getCookiesStr()
		return &instagram.RegResult{Success: false, Message: "Checkpoint after save-device", Password: password, Cookie: cookie}
	}

	dtsg := reFind(saveDeviceHTML, `dtsg":{"token":"(.*?)",`, 1)
	jazoest := reFind(saveDeviceHTML, `name="jazoest" value="(\d+)"`, 1)
	if dtsg == "" {
		cookie := sess.getCookiesStr()
		uid := parseCUserFromCookie(cookie)
		if uid != "" && cookie != "" {
			return buildSuccess(cookie, uid, password, prof.UserAgent, notify)
		}
		return &instagram.RegResult{Success: false, Message: "no dtsg in save-device", Password: password}
	}

	noncePOSTBody := fmt.Sprintf(
		"fb_dtsg=%s&jazoest=%s&flow=interstitial_nux&next=%%2Fconfirmemail.php%%3Fresend&nux_source=dbl_nux_after_reg",
		dtsg, jazoest,
	)
	nonceHeaders := buildNavHeaders(prof, refSaveDevice, "https://m.facebook.com")
	nonceHeaders = append(nonceHeaders, [2]string{"Content-Type", "application/x-www-form-urlencoded"})

	time.Sleep(1000 * time.Millisecond)
	sess.post(ctx, "https://m.facebook.com/login/device-based/update-nonce/", noncePOSTBody, nonceHeaders)

	if isCheckpoint(sess.finalURL) {
		cookie := sess.getCookiesStr()
		return &instagram.RegResult{Success: false, Message: "Checkpoint after update-nonce", Password: password, Cookie: cookie}
	}
	if isConfirmMailURL(sess.finalURL) {
		cookie := sess.getCookiesStr()
		uid := parseCUserFromCookie(cookie)
		if uid != "" && cookie != "" {
			return buildSuccess(cookie, uid, password, prof.UserAgent, notify)
		}
		return &instagram.RegResult{Success: true, Message: "Register OK — confirm email required", Password: password, Cookie: sess.getCookiesStr()}
	}
	return &instagram.RegResult{Success: false, Message: "unknown result after save-device", Password: password}
}

// extractFinalResult — cookie extraction after final GET (New UI).
func extractFinalResult(sess *session, finalHTML, uid, password, userAgent string, notify func(string)) *instagram.RegResult {
	cookie := sess.getCookiesStr()
	if cookie == "" {
		return &instagram.RegResult{Success: false, Message: "No cookies after register", Password: password}
	}
	if !strings.Contains(cookie, "datr") && finalHTML != "" {
		if datr := parseDatrFromHTML(finalHTML); datr != "" {
			cookie = "datr=" + datr + ";" + cookie
		}
	}
	if !strings.Contains(cookie, "locale") {
		cookie += ";locale=en_US"
	}
	return buildSuccess(cookie, uid, password, userAgent, notify)
}

func buildSuccess(cookie, uid, password, userAgent string, notify func(string)) *instagram.RegResult {
	if !strings.Contains(cookie, "locale") {
		cookie += ";locale=en_US"
	}
	// Save datr vào datr_pool.txt + add vào SharedPool (C# TryAddNewDatrToPool + SaveDatrFromCookieIfNew)
	if datr := cookiestore.ExtractDatr(cookie); datr != "" {
		if SharedPool != nil {
			if SharedPool.AddDatrRaw(datr) && notify != nil {
				trunc := datr
				if len(trunc) > 10 {
					trunc = trunc[:10]
				}
				notify(fmt.Sprintf("[Pool] Datr mới: %s... (pool size: %d)", trunc, SharedPool.Size()))
			}
		}
	}
	return &instagram.RegResult{
		Success:   true,
		UID:       uid,
		Cookie:    cookie,
		Password:  password,
		Message:   fmt.Sprintf("Register OK — UID: %s", uid),
		UserAgent: userAgent,
	}
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
