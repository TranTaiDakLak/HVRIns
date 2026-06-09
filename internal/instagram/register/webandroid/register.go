// register.go — Web Android Registerer + WorkerContext + Register orchestrator.
//
// File này gộp 3 file cũ:
//   - register.go (cũ) → Registerer interface + init()
//   - worker_ctx.go     → WorkerContext (keep-session + ChromeAndroid profile pinning)
//   - steps.go          → registerAccount + (w *WorkerContext).Register flow
//
// Package webandroid — Facebook Web Android hybrid register.
// Mapping từ C#: FacebookRegisterWebAndroidAPI.
// Flow: GET m.facebook.com/ → POST /async/wbloks/fetch/ → collect cookies.
// Dùng Chrome Android browser UA + TLS fingerprint (không phải FB4A Dalvik).
package webandroid

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	cookiestore "HVRIns/internal/cookie"
	"HVRIns/internal/instagram"
	webreg "HVRIns/internal/instagram/register/web"
	"HVRIns/internal/instagram/fakeinfo"
)

// ─── Registerer interface + init ─────────────────────────────────────────────

// Registerer implements instagram.Registerer for Web Android platform.
type Registerer struct{}

// Register thực hiện đăng ký tài khoản qua Web Android flow.
func (r *Registerer) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	return registerAccount(ctx, input, onStatus)
}

func init() {
	instagram.RegisterPlatformRegisterer(instagram.PlatformWebAndroid, func() instagram.Registerer {
		return &Registerer{}
	})
}

// ─── WorkerContext (keep-session + device pinning) ───────────────────────────
//
// C# equivalent: mỗi thread dùng chung `httpRequest` (IHttpRequestClient) +
// `FacebookAccountModel` cached với ChromeAndroid UA/device fingerprint cố định.
//
// Port tương tự S23/Android V3 — giảm device rotation → FB trust hơn:
//   - Pin ChromeAndroid profile (UA, Chrome version, device model, viewport, dpr).
//   - Reuse HTTP session (TCP pool + cookie jar cleared giữa các regs).

// WorkerContext per-goroutine state.
type WorkerContext struct {
	sess    *session
	profile fakeinfo.ChromeAndroidProfile // device/UA fingerprint — pinned
}

// NewWorkerContext tạo context mới cho 1 worker goroutine.
func NewWorkerContext(proxyStr string) (*WorkerContext, error) {
	sess, err := newSession(proxyStr)
	if err != nil {
		return nil, err
	}
	profile := fakeinfo.RandomChromeAndroidProfile()
	return &WorkerContext{sess: sess, profile: profile}, nil
}

// Close giải phóng idle TCP connections.
func (w *WorkerContext) Close() {
	if w != nil && w.sess != nil {
		w.sess.client.CloseIdleConnections()
	}
}

// Profile trả về pinned profile (caller đọc UA để display UI).
func (w *WorkerContext) Profile() fakeinfo.ChromeAndroidProfile { return w.profile }

// SetUA override UserAgent bằng chuỗi raw (dùng khi UseRawUa = true).
func (w *WorkerContext) SetUA(ua string) {
	if w == nil || ua == "" {
		return
	}
	w.profile.UserAgent = ua
}

// ─── Register orchestrator ───────────────────────────────────────────────────
//
// Mapping từ C#: FacebookRegisterWebAndroidAPI.Register().
//
// Flow:
//  1. GET m.facebook.com/ → extract versioningID + tokens + cookies
//  2. POST /async/wbloks/fetch/ → register → extract UID
//  3. GET m.facebook.com/ → collect final cookies
//  4. Logout nếu cần → extract cookie string

// registerAccount thực hiện toàn bộ flow Web Android register (single-shot wrapper).
// Mapping từ C#: FacebookRegisterWebAndroidAPI.Register().
// Workers muốn keep-session nên dùng WorkerContext.Register() trực tiếp.
func registerAccount(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	proxyStr := ""
	if input != nil {
		proxyStr = input.Proxy
	}
	wctx, err := NewWorkerContext(proxyStr)
	if err != nil {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Tạo HTTP client thất bại: %v", err)}
	}
	defer wctx.Close()
	return wctx.Register(ctx, input, onStatus)
}

// Register dùng session + profile pinned trong WorkerContext.
// Reuse TCP pool + ChromeAndroid device fingerprint cho N regs trong cùng worker.
func (w *WorkerContext) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(msg)
		}
	}

	// Reuse pinned profile + session từ WorkerContext.
	prof := w.profile
	sess := w.sess

	// Clear cookies từ reg trước (tránh leak c_user/xs của account cũ).
	sess.clearCookies()

	// Password
	password := fakeinfo.RandomPassword()
	if input != nil && input.Password != "" {
		password = input.Password
	}

	// Contact point — C# ModeReg (0=Mail, 1=Phone)
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

	// Fake profile (name, birthday, gender) — regen per reg, không pin theo worker.
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

	// Tag platform gọn — phone/email đã có cột riêng.
	_ = cpType
	notify(fmt.Sprintf("[WebAndroid] Bắt đầu reg — %s %s | %s | UA: %s",
		fake.FirstName, fake.LastName, contactpoint, truncStr(prof.UserAgent, 60)))

	// Cookie Initial seeding — C#: GetPerfectMachineId → AddCookies
	// Ưu tiên TutDatr từ input, else lấy từ SharedPool.
	machineID := ""
	slotIdx := 0
	datrSource := "" // diagnostic: "tut" | "pool" | "pool-nil" | "pool-empty"
	if input != nil {
		slotIdx = input.SlotIdx
		if input.TutDatr != "" {
			machineID = input.TutDatr
			datrSource = "tut"
			seedCookies(sess, input.TutDatr)
		}
	}
	if machineID == "" {
		if SharedPool == nil {
			datrSource = "pool-NIL"
		} else if poolDatr := SharedPool.GetNext(slotIdx); poolDatr != "" {
			machineID = poolDatr
			datrSource = "pool"
			seedCookies(sess, poolDatr)
			s, f, u, used := SharedPool.GetStats(poolDatr)
			notify(fmt.Sprintf("[WebAndroid] New initial %s (used %d | S/F/U: %d/%d/%d)",
				poolDatr, used, s, f, u))
		} else {
			datrSource = fmt.Sprintf("pool-EMPTY(slot=%d)", slotIdx)
		}
	} else if SharedPool != nil {
		s, f, u, used := SharedPool.GetStats(machineID)
		notify(fmt.Sprintf("[WebAndroid] Dùng datr %s... (used %d | S/F/U: %d/%d/%d)",
			truncStr(machineID, 10), used, s, f, u))
	}
	if machineID != "" && SharedPool != nil {
		defer SharedPool.IncrementUsage(machineID)
	}

	// Datr display string — show source rõ khi NONE để debug pool issue.
	datrTag := fmt.Sprintf(" | datr=NONE(%s)", datrSource)
	if machineID != "" {
		datrTag = fmt.Sprintf(" | datr=%s(%s)", truncStr(machineID, 10), datrSource)
	}

	notify("[WebAndroid] GET m.facebook.com/...")
	navHeaders := buildNavHeaders(prof, "", "")
	pageHTML, err := sess.get(ctx, "https://m.facebook.com/", navHeaders)
	if err != nil {
		if machineID != "" && SharedPool != nil {
			SharedPool.RecordResult(machineID, "unknown")
		}
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("GET m.facebook.com thất bại: %v%s", err, datrTag)}
	}

	// Extract tokens từ page HTML
	tokens := parsePageTokens(pageHTML)
	if tokens.fbDtsg == "" {
		notify("[WebAndroid] Lỗi: không extract được fb_dtsg từ trang")
		if machineID != "" && SharedPool != nil {
			SharedPool.RecordResult(machineID, "unknown")
		}
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Không lấy được fb_dtsg từ m.facebook.com%s", datrTag)}
	}
	notify(fmt.Sprintf("[WebAndroid] Tokens OK — versioningID=%s lsd=%s", truncStr(tokens.versioningID, 16), tokens.lsd))

	// === Step 2: POST register ===
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
	body := buildRegisterBody(params)

	// Referer = URL sau redirect của GET (C#: _referrer_url = httpResponse.Target)
	referer := sess.finalURL
	if referer == "" {
		referer = "https://m.facebook.com/"
	}
	// C#: PerpectChromeAndroidPostHeadersFormat2(_referrer_url, MfbSingleUrl, accountInfo)
	postHeaders := buildPostHeaders(prof, referer, "https://m.facebook.com")

	datrInfo := ""
	if machineID != "" && SharedPool != nil {
		s, f, u, used := SharedPool.GetStats(machineID)
		datrInfo = fmt.Sprintf(" | datr=%s... used=%d S/F/U=%d/%d/%d",
			truncStr(machineID, 10), used, s, f, u)
	}
	notify(fmt.Sprintf("[WebAndroid] POST wbloks/fetch (__bkv=%s%s)...", truncStr(tokens.versioningID, 16), datrInfo))
	respBody, err := sess.post(ctx, postURL(tokens.versioningID), body, postHeaders)
	if err != nil {
		if machineID != "" && SharedPool != nil {
			SharedPool.RecordResult(machineID, "unknown")
		}
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("POST register thất bại: %v%s", err, datrTag)}
	}

	// Extract UID từ response
	uid := parseUID(respBody)
	if uid == "" {
		// Không kèm raw response vào notify — response có thể rất dài, làm UI lag.
		notify("[WebAndroid] Thất bại — không tìm được UID trong response")
		failDatrInfo := datrTag
		if machineID != "" && SharedPool != nil {
			SharedPool.RecordResult(machineID, "fail")
			s, f, u, used := SharedPool.GetStats(machineID)
			notify(fmt.Sprintf("[Pool] Datr %s... → fail (used %d | S/F/U: %d/%d/%d)",
				truncStr(machineID, 10), used, s, f, u))
			failDatrInfo = fmt.Sprintf(" | datr=%s used=%d", truncStr(machineID, 10), used)
		}
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Register failed: UID not found in response%s", failDatrInfo), Password: password}
	}
	notify(fmt.Sprintf("[WebAndroid] Thành công! UID=%s", uid))

	// === Step 3: GET m.facebook.com/ để collect cookies ===
	time.Sleep(500 * time.Millisecond)

	notify("[WebAndroid] GET m.facebook.com/ lần 2 — collect cookies...")
	navHeaders2 := buildNavHeaders(prof, "", "")
	pageHTML2, err := sess.get(ctx, "https://m.facebook.com/", navHeaders2)
	if err != nil {
		// Không critical — dùng cookies hiện có
		notify(fmt.Sprintf("[WebAndroid] GET lần 2 lỗi: %v (dùng cookie hiện có)", err))
		pageHTML2 = ""
	}

	// === Step 4: Logout nếu bị redirect tới checkpoint ===
	// C# L177-187: needLogout chỉ set true khi URL chứa "checkpoint" → bypass
	// bằng GET /checkpoint/.../logout/?next=m.facebook.com/...
	// C# L188-191: !VerifyAfterReg branch bị comment-out → Go không làm thêm logout logic.
	needLogout := false
	finalURL2 := sess.finalURL

	if isCheckpoint(finalURL2) {
		notify("[WebAndroid] Checkpoint detected — logout...")
		checkpointLogoutURL := "https://m.facebook.com/checkpoint/1501092823525282/logout/?next=https%3A%2F%2Fm.facebook.com%2F&__m_async_page__=&__big_pipe_on__="
		pageHTML2, _ = sess.get(ctx, checkpointLogoutURL, buildNavHeaders(prof, "", ""))
		needLogout = true
	}
	// needLogout flag hiện tại không trigger thêm /logout.php (C# cũng comment out).
	_ = needLogout

	// === Extract final cookie theo đúng thứ tự C# (L228) ===
	// C# format cố định: `datr={datr};sb={sb};c_user={c_user};xs={xs};fr={fr};pas={pas}`
	// xs được double URL-decode (C# L226: WebUtility.UrlDecode(WebUtility.UrlDecode(xs)))
	cookie := sess.getCookiesFBOrder()
	if cookie == "" {
		return &instagram.RegResult{Success: false, Message: "Không lấy được cookie sau register", Password: password}
	}

	// Fallback: lấy datr từ HTML nếu cookie jar thiếu (C# L230-237: create_success."(.*?)"(.*?).",)
	if !strings.Contains(cookie, "datr=") && pageHTML2 != "" {
		if datr := reFind(pageHTML2, `create_success\."(.*?)"(.*?)\.",`, 2); datr != "" {
			cookie = "datr=" + datr + ";" + cookie
		}
	}
	// Đảm bảo có locale trong cookie — C# L238-240: cookie += $";locale={Locale};" (có `;` cuối)
	locale := fakeinfo.RandomLocale()
	if locale == "" {
		locale = "en_US"
	}
	if !strings.Contains(cookie, "locale=") {
		cookie += ";locale=" + locale + ";"
	}

	notify(fmt.Sprintf("[WebAndroid] Cookie: %s", truncStr(cookie, 80)))

	// Track success cho datr đang dùng + add datr MỚI vào pool
	if machineID != "" && SharedPool != nil {
		SharedPool.RecordResult(machineID, "success")
		s, f, u, used := SharedPool.GetStats(machineID)
		notify(fmt.Sprintf("[Pool] Datr %s... → success (used %d | S/F/U: %d/%d/%d)",
			truncStr(machineID, 10), used, s, f, u))
	}
	if newDatr := cookiestore.ExtractDatr(cookie); newDatr != "" && SharedPool != nil {
		if SharedPool.AddDatrRaw(newDatr) {
			notify(fmt.Sprintf("[Pool] Datr mới: %s... (pool size: %d)", truncStr(newDatr, 10), SharedPool.Size()))
		}
	}

	// Save datr vào datr_pool.txt (C# SaveDatrFromCookieIfNew)
	if datr := cookiestore.ExtractDatr(cookie); datr != "" {
		go func(d string) {
			if err := cookiestore.AppendDatr("", d); err != nil {
				slog.Warn("save datr failed", "err", err, "datr", d)
			}
		}(datr)
	}

	// === Lấy Access Token qua Android API (C# FacebookVerifyAPIToken.Login) ===
	// WebAndroid reg không tự có token → phải login lại qua Android graphql endpoint.
	// Match C# FacebookVerifyAPIToken pattern: POST graph.facebook.com/graphql
	// với LoginMobileFormData → response có EAA... token.
	// Best-effort: nếu fail, trả reg success không token (user dùng cookie cũng OK).
	//
	// === Lấy Access Token qua REST `/auth/login` (PORT S399 step 2) ===
	// Flow:
	//   POST b-graph.facebook.com/auth/login
	//     - email: UID
	//     - password: #PWD_FB4A:0:ts:plaintext (KHÔNG encrypt)
	//     - machine_id: datr từ cookie
	//     - api_key + sig MD5 + app token
	//   Response JSON: { access_token: "EAAAAU...", session_cookies: [...] }
	//
	// REST classic API stable, KHÔNG dùng Bloks/GraphQL schema rotation.
	// Đã verify: 8/8 NVR WebAndroid accounts đều lấy được EAAAAU token.
	// (Test với proxy thực, kết quả 100% success rate).
	//
	// machineID = datr extract từ cookie reg (truyền qua param sau khi extract).
	regDatr := ""
	if i := strings.Index(cookie, "datr="); i >= 0 {
		s := i + 5
		e := s
		for e < len(cookie) && cookie[e] != ';' {
			e++
		}
		regDatr = cookie[s:e]
	}
	var accessToken string
	if uid != "" && password != "" && !webreg.SkipAuthLoginAtReg {
		notify("[WebAndroid] Lấy access token (REST /auth/login, port S399)...")
		tokenCtx, tokenCancel := context.WithTimeout(ctx, 30*time.Second)
		accessToken = webreg.FetchAndroidTokenLegacy(tokenCtx, uid, password, regDatr, locale, "", sess.proxyStr(), "", notify)
		tokenCancel()
		if accessToken != "" {
			notify(fmt.Sprintf("[WebAndroid] Access token OK — %s...", truncStr(accessToken, 20)))
		} else {
			notify("[WebAndroid] Không lấy được access token (bỏ qua, cookie vẫn OK)")
		}
	}
	_ = machineID

	return &instagram.RegResult{
		Success:     true,
		UID:         uid,
		Cookie:      cookie,
		AccessToken: accessToken,
		Password:    password,
		Message:     fmt.Sprintf("Register OK — UID: %s", uid),
		UserAgent:   prof.UserAgent,
	}
}

func truncStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
