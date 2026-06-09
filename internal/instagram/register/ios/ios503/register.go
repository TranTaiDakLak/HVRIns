// Package ios562 — Facebook iOS native (FBIOS) registration, API v562.
//
// Native FBIOS app reg qua Bloks CAA: POST graph.facebook.com/graphql với
// OAuth app-token, 1 request single-shot create.account.async.
// KHÁC register/ioshttp (đó là MFB web — m.facebook.com, Safari UA, cookie-based).
//
// Derive từ capture E:\WEMAKE\DocWeMake\FlowRegFB_IOS\APIRegVer_IOS.
package ios503

import (
	"context"
	"fmt"

	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
)

// Registerer implements instagram.Registerer cho platform iOS555.
type Registerer struct{}

// Register thực hiện đăng ký 1 account qua native FBIOS flow.
func (r *Registerer) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	return registerAccount(ctx, input, onStatus)
}

// registerAccount là wrapper: track datr outcome sau khi reg xong.
func registerAccount(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	var pickedDatr string
	result := doRegisterAccount(ctx, input, onStatus, &pickedDatr)
	if pickedDatr != "" && SharedDatrPool != nil {
		outcome := "fail"
		if result != nil && result.Success {
			outcome = "success"
		}
		SharedDatrPool.RecordResult(pickedDatr, outcome)
	}
	return result
}

func init() {
	instagram.RegisterPlatformRegisterer(instagram.PlatformIOS503, func() instagram.Registerer {
		return &Registerer{}
	})
}

// doRegisterAccount — orchestrator: resolve input → build profile/body → POST → parse.
func doRegisterAccount(ctx context.Context, input *instagram.RegInput, onStatus func(string), pickedDatrOut *string) *instagram.RegResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(msg)
		}
	}

	// ── Resolve contactpoint ───────────────────────────────────────────────
	contactpoint := ""
	cpType := "phone"
	proxyStr := ""
	if input != nil {
		proxyStr = input.Proxy
		if input.Email != "" {
			contactpoint = input.Email
			cpType = "email"
		} else if input.Phone != "" {
			contactpoint = input.Phone
		}
	}
	if contactpoint == "" {
		return &instagram.RegResult{Success: false, Message: "Thiếu contactpoint (email/phone)"}
	}

	// ── Locale theo country ────────────────────────────────────────────────
	countryCode := ""
	if input != nil && input.Phone != "" {
		if p, ok := fakeinfo.FindCountryByPhonePrefix(input.Phone); ok {
			countryCode = p.CountryCode
		}
	}
	locale := fakeinfo.LocaleFromCountry(countryCode)
	if locale == "" {
		locale = RandIOSLocale()
	}

	// ── Fake profile (name/birthday/gender) ────────────────────────────────
	fake := fakeinfo.RandomFakeProfileByLocale(countryCode)
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

	password := fakeinfo.RandomPassword()
	if input != nil && input.Password != "" {
		password = input.Password
	}

	slotIdx := 0
	if input != nil {
		slotIdx = input.SlotIdx
	}

	// ── Build profile (pool → random fallback) ────────────────────────────
	var profile IOSProfile
	if SharedDevicePool != nil {
		if dp := SharedDevicePool.GetNext(); dp != nil {
			profile = BuildProfileFromDevice(locale, countryCode, *dp)
			notify(fmt.Sprintf("[iOS503] Start — %s %s | %s | %s [pool dev]",
				fake.FirstName, fake.LastName, contactpoint, profile.Device.FBDV))
		}
	}
	if profile.DeviceID == "" {
		profile = BuildProfile(locale, countryCode)
		notify(fmt.Sprintf("[iOS503] Start — %s %s | %s | %s [rand dev]",
			fake.FirstName, fake.LastName, contactpoint, profile.Device.FBDV))
	}

	// ── Datr pool → override MachineID (X-FB-Integrity-Machine-Id) ───────
	// Luôn lấy datr mới từ SharedDatrPool nếu có — override bất kể MachineID
	// hiện tại là từ DevicePool hay random. DatrPool cung cấp lịch sử session
	// tin cậy (Facebook trust score), DevicePool cung cấp DeviceID/FamilyDeviceID.
	var pickedDatr string
	if SharedDatrPool != nil {
		if datr := SharedDatrPool.GetNext(slotIdx); datr != "" {
			pickedDatr = datr
			profile.MachineID = datr
			s, f, u, used := SharedDatrPool.GetStats(datr)
			pfx := datr
			if len(pfx) > 8 {
				pfx = pfx[:8]
			}
			notify(fmt.Sprintf("[iOS503] Dùng datr %s... | used=%d S/F/U=%d/%d/%d",
				pfx, used, s, f, u))
		} else {
			notify("[iOS503] ⚠️ DatrPool EMPTY — dùng MachineID random")
		}
	}
	if pickedDatrOut != nil {
		*pickedDatrOut = pickedDatr
	}

	// ── HTTP session (trước body để dùng cho pwd_key_fetch) ───────────────
	sess, err := newSession(proxyStr, profile.Device.IOSDot)
	if err != nil {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Create session lỗi: %v", err), Password: password, UserAgent: profile.UserAgent}
	}
	defer sess.client.CloseIdleConnections()

	// ── Mã hóa password (#PWD_WILDE:2 via RSA, fallback #PWD_FB4A:0) ─────
	encPwd := encryptPasswordForReg(ctx, sess, profile, password)

	// ── Build body ────────────────────────────────────────────────────────
	fields := regFields{
		firstName:         fake.FirstName,
		lastName:          fake.LastName,
		birthday:          fake.Birthday,
		gender:            fake.Gender,
		contactpoint:      contactpoint,
		cpType:            cpType,
		password:          password,
		encryptedPassword: encPwd,
	}
	body, err := buildCreateAccountBody(profile, fields)
	if err != nil {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Build body lỗi: %v", err), Password: password, UserAgent: profile.UserAgent}
	}

	headers := buildHeaders(profile)
	notify(fmt.Sprintf("[iOS503] POST create.account (%s)...", profile.Device.FBDV))

	respBody, err := sess.postGzip(ctx, graphURL, body, headers)
	if err != nil {
		// HTTP error vẫn có thể kèm body — thử parse trước khi báo lỗi.
		if respBody == "" {
			return &instagram.RegResult{Success: false, Message: fmt.Sprintf("HTTP lỗi: %v", err), Password: password, UserAgent: profile.UserAgent}
		}
	}
	notify(fmt.Sprintf("[iOS503] Response (%d bytes)", len(respBody)))

	// ── Parse ──────────────────────────────────────────────────────────────
	outcome, perr := parseCreateAccountResponse(respBody)

	if perr != nil {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("Register failed: %v", perr), Password: password, UserAgent: profile.UserAgent}
	}

	// ── Round 2: nosess → gọi lại create.account với token vừa cấp ─────────
	// Khi outcome là "nosess" (có UID nhưng chưa có session), FB đã cấp
	// fb_partially_created_reg_info + srnonce trong response — phải gọi lại
	// create.account.async với 2 token đó (giống [164] trong capture) để hoàn tất.
	// ── Loop round 2..N: khi nosess + có Partial tokens → gọi lại đến khi full hoặc cạn ─
	// Capture cho thấy chain [158]→[162]→[164] = 3 lần create.account. Tôi cap ở 3 vòng
	// (round 1 + 2 retries). Mỗi vòng dùng partial mới từ response trước.
	const maxRounds = 3
	for round := 2; round <= maxRounds; round++ {
		if !(outcome.UID != "" && outcome.AccessToken == "" && outcome.Partial != nil) {
			if outcome.UID != "" && outcome.AccessToken == "" {
				notify(fmt.Sprintf("[iOS503] Round %d skip — Partial nil", round))
			}
			break
		}
		notify(fmt.Sprintf("[iOS503] Round %d (partial=%d, srn=%d, regctx=%d)",
			round, len(outcome.Partial.PartiallyCreated), len(outcome.Partial.Srnonce), len(outcome.Partial.RegContext)))
		bodyN, berr := buildCreateAccountRound2(profile, outcome.Partial)
		if berr != nil {
			notify(fmt.Sprintf("[iOS503] Round %d build body lỗi: %v", round, berr))
			break
		}
		headersN := buildHeaders(profile)
		respN, herr := sess.postGzip(ctx, graphURL, bodyN, headersN)
		outN, perrN := parseCreateAccountResponse(respN)
		notify(fmt.Sprintf("[iOS503] Round %d resp %d bytes, err=%v", round, len(respN), herr))
		if perrN != nil || outN == nil || outN.UID == "" {
			break
		}
		// Round N trả full → done.
		if outN.AccessToken != "" || outN.Cookie != "" {
			notify(fmt.Sprintf("[iOS503] Round %d OK — token=%v cookie=%v",
				round, outN.AccessToken != "", outN.Cookie != ""))
			outcome = outN
			break
		}
		// Round N trả nosess với Partial mới → tiếp tục vòng kế.
		outcome = outN
	}

	// Cookie từ session_cookies trong response — gắn thêm locale nếu có.
	cookie := outcome.Cookie
	if cookie != "" {
		cookie += "locale=" + locale + ";"
	}

	// Lưu device profile vào pool để tái dùng ở reg tiếp theo.
	if SharedDevicePool != nil {
		SharedDevicePool.Add(DeviceProfile{
			DeviceID:       profile.DeviceID,
			FamilyDeviceID: profile.FamilyDeviceID,
			MachineID:      profile.MachineID,
		})
	}

	// Thu datr từ response (session_cookies) → thêm vào SharedDatrPool.
	if outcome.DATR != "" && SharedDatrPool != nil {
		if SharedDatrPool.AddDatrRaw(outcome.DATR) {
			pfx := outcome.DATR
			if len(pfx) > 10 {
				pfx = pfx[:10]
			}
			notify(fmt.Sprintf("[iOS555Pool] Datr mới: %s... (pool size: %d)", pfx, SharedDatrPool.Size()))
		}
	}

	msg := fmt.Sprintf("Register OK — UID: %s (iOS555)", outcome.UID)
	if outcome.AccessToken == "" {
		msg += " — cần verify để lấy token"
	}
	notify("[iOS503] " + msg)

	var srnonce, sessionlessCUID string
	if outcome.Partial != nil {
		srnonce = outcome.Partial.Srnonce
		// SessionlessCUID = Q9T_ blob (~120 chars) dùng cho sessionless verify.
		// PartiallyCreated = Q8CKBQ blob (~9KB) dùng cho Round 2 reg — không dùng ở đây.
		sessionlessCUID = outcome.Partial.SessionlessCUID
	}

	return &instagram.RegResult{
		Success:               true,
		UID:                   outcome.UID,
		Cookie:                cookie,
		AccessToken:           outcome.AccessToken,
		Password:              password,
		Message:               msg,
		UserAgent:             profile.UserAgent,
		DeviceID:              profile.DeviceID,
		FamilyDeviceID:        profile.FamilyDeviceID,
		Srnonce:               srnonce,
		SessionlessCryptedUID: sessionlessCUID,
	}
}
