package iosmess

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	regmess "HVRIns/internal/instagram/register/ios/iosmess"
	fbweb "HVRIns/internal/instagram/register/web"
	"HVRIns/internal/instagram/verify/verifybase"
)

const tag = "[iOS Mess Verify]"

// verifyAccount — add-mail (trigger OTP) + screen-loads thủ công → RunVerify (reuse mail → OTP → confirm → live/die).
func verifyAccount(ctx context.Context, session *instagram.Session, cfg *instagram.VerifyConfig, outputPath string, onStatus func(uid string, msg string)) *instagram.VerifyResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(session.UID, msg)
		}
	}

	// ── Reconstruct fingerprint từ reg ──
	device := session.DeviceID
	family := session.FamilyDeviceID
	machine := session.Datr
	waterfall := session.Srnonce
	if waterfall == "" {
		waterfall = uuid.New().String()
	}
	cryptedUID := session.SessionlessCryptedUID
	// Account reg từ iOS version (ios5xx/FB iOS native) mang SessionlessCryptedUID dạng "Q9T_..."
	// — KHÁC crypted_user_id của Messenger ("AY..."). Blob Q9T_ KHÔNG dùng được cho body Mess
	// verify → phải login-first lấy crypted_user_id chuẩn. (Reg Mess iOS thì crypted đã là "AY..."
	// nên không login lại.) Điều kiện: rỗng HOẶC không đúng định dạng Messenger.
	needLoginFirst := cryptedUID == "" || !strings.HasPrefix(cryptedUID, "AY")
	// Tất cả bước ver dùng app-token (sendBloks mặc định). Không dùng EAAG user token.
	ua := session.UserAgent
	if ua == "" || !strings.Contains(ua, "MessengerLiteForiOS") {
		ua = regmess.RandomMessUA()
		session.UserAgent = ua
	}
	if session.UID == "" {
		return &instagram.VerifyResult{Status: "error", Message: "thiếu UID", UserAgent: ua}
	}

	// ── Login-first flow: khi thiếu crypted_user_id (ver từ file NVR không có reg session) ──
	// Login bằng app-token → server trả crypted_user_id trong response Bloks.
	// Capture TestVerIosMessByToken xác nhận: TẤT CẢ bước (login/bottomsheet/add-mail/confirm)
	// đều dùng app-token — KHÔNG dùng EAAG user token (user token trigger 459 checkpoint).
	if needLoginFirst && session.Password != "" {
		notify(tag + " crypted_uid thiếu/không chuẩn Mess → login iOS Mess lấy crypted_uid...")
		loginDevice := device
		if loginDevice == "" {
			loginDevice = strings.ToUpper(uuid.New().String())
		}
		loginFamily := family
		if loginFamily == "" {
			loginFamily = strings.ToUpper(uuid.New().String())
		}
		_, _, lCrypted, lErr := regmess.LoginFull(session.Proxy, session.UID, session.Password, loginDevice, loginFamily, machine, waterfall, ua)
		if lErr != nil {
			notify(fmt.Sprintf("%s login-first fail (%v) — tiếp tục không có cryptedUID", tag, lErr))
		} else {
			notify(fmt.Sprintf("%s login-first OK — cryptedUID=%t", tag, lCrypted != ""))
			if lCrypted != "" {
				cryptedUID = lCrypted
				if device == "" {
					device = loginDevice
					family = loginFamily
				}
			}
			// Sinh fresh AAC + flow IDs (không có từ reg session)
			if session.AACJid == "" {
				aacJid, aacCs, aacTs := regmess.GenAACParts()
				session.AACJid, session.AACcs, session.AACts = aacJid, aacCs, aacTs
				notify(fmt.Sprintf("%s gen fresh AAC: jid=%.8s...", tag, aacJid))
			}
			if session.PassRaw == "" {
				session.PassRaw = session.Password
				session.PassTS = time.Now().Unix()
			}
		}
	} else if needLoginFirst {
		notify(tag + " ⚠️ crypted_user_id thiếu/không chuẩn Mess và không có password — add-mail có thể fail")
	}

	// Email từ reg (reuse). Nếu rỗng, RunVerify sẽ tự tạo mail mới (nhưng add-mail thủ công cần email).
	email := session.Email
	if email == "" {
		notify(tag + " ⚠️ session.Email rỗng — không add-mail thủ công được, để RunVerify xử lý")
	} else {
		at := strings.SplitN(email, "@", 2)
		if len(at) == 2 {
			// ── THỨ TỰ GOOD: bottomsheet + change.email TRƯỚC (thiết lập ngữ cảnh đổi email),
			// rồi add-mail LAST (submit email → server gửi OTP). Add-mail trước screen-loads thì
			// server chưa transition userid sang change-email sub-flow → state=UNKNOWN, render 9KB. ──
			if client, err := regmess.NewIOSClient(session.Proxy, 60); err == nil {
				for _, which := range []struct{ name, fn string }{
					{"bottomsheet", regmess.FnBottomsheet}, {"change_email", regmess.FnChangeMail},
				} {
					slb, _ := regmess.BuildScreenLoadBody(which.name, device, family, machine, waterfall, session.UID, cryptedUID, at[0], at[1], session.Phone, session.AACJid, session.AACcs, session.AACts, session.RegFlowID, session.HeadersFlowID)
					_, _, _ = regmess.SendStep(client, slb, which.fn, device, ua)
					time.Sleep(400 * time.Millisecond)
				}
				amBody, _ := regmess.BuildAddMailBody(device, family, machine, waterfall, session.UID, cryptedUID, at[0], at[1], session.Phone, session.AACJid, session.AACcs, session.AACts, session.RegFlowID, session.HeadersFlowID, session.PassRaw, session.PassTS)
				_, amRaw, amErr := regmess.SendStep(client, amBody, regmess.FnAddMail, device, ua)
				if amErr != nil {
					notify(tag + " add-mail HTTP err: " + amErr.Error())
				} else {
					notify(fmt.Sprintf("%s add-mail (%d bytes) — OTP triggered", tag, len(amRaw)))
				}
			}
		}
	}

	// ── Spec cho RunVerify (reuse mail → WaitOTP → confirm → live/die) ──
	spec := verifybase.Spec{
		Tag:                   tag,
		Phone:                 session.Phone, // msg_previous_cp cho add-mail (đổi phone→email)
		AACJid:                session.AACJid, // AAC session của create — add-mail/confirm tái dùng
		AACcs:                 session.AACcs,
		AACts:                 session.AACts,
		RegFlowID:             session.RegFlowID, // flow_id session của create — add-mail/confirm tái dùng
		HeadersFlowID:         session.HeadersFlowID,
		PassRaw:               session.PassRaw, // mật khẩu thô + ts — add-mail cần encrypted_password đúng
		PassTS:                session.PassTS,
		IsPushOn:              false,
		SkipUserTokenCheck:    true,
		CreateClient:          verifybase.CreateIOSClient,
		GraphEndpoint:         verifybase.GraphURL,
		SessionlessCryptedUID: cryptedUID,
		Srnonce:               waterfall,
		AddEmailTimeout: 30 * time.Second,
		BuildHeaders: buildHeaders,
		BuildAddEmailBody:     buildAddEmailBody, // dùng nếu RunVerify KHÔNG reuse (tạo mail mới)
		BuildResendBody:       buildAddEmailBody, // resend = re-fire add-mail (trigger OTP lại)
		BuildConfirmBody:      buildConfirmBody,
		CheckAddEmailSuccess: func(resp string) bool {
			low := strings.ToLower(deepUnescape(resp))
			if strings.Contains(low, "email_already_used") || strings.Contains(low, "email_is_invalid") ||
				strings.Contains(low, "checkpoint") || strings.Contains(low, "\"errors\":[{") {
				return false
			}
			return strings.Contains(low, "caa_reg_confirmation") ||
				strings.Contains(low, "fb_bloks_action") || strings.Contains(low, "action_bundle") ||
				strings.Contains(low, "caa_reg_contactpoint")
		},
		CheckConfirmSuccess: func(resp string) bool { return regmess.IsConfirmOK(resp) },
		// Live/Die: login uid+password lấy token → CheckLiveDieByToken (catch checkpoint ngay).
		CheckLiveDieFunc: func(c context.Context, ua, uid, token string) string {
			if token != "" {
				if st := verifybase.CheckLiveDieByToken(c, ua, token); st != "Unknown" {
					return st
				}
			}
			lctx, cancel := context.WithTimeout(c, 30*time.Second)
			tok, _ := fbweb.FetchAndroidTokenLegacyWithCookie(lctx, session.UID, session.Password, session.Datr, "vi_VN", "VN", session.Proxy, "", nil)
			cancel()
			if tok != "" {
				return verifybase.CheckLiveDieByToken(c, ua, tok)
			}
			return verifybase.CheckLiveDieByPicture(c, ua, uid)
		},
	}
	result := verifybase.RunVerify(ctx, session, cfg, outputPath, onStatus, spec)

	// Account LIVE → login MessengerLite iOS (send_login_request) lấy token+cookie THẬT.
	// (Token/cookie từ reg nằm trong blob mã hoá caa_core_data_encrypted → không dùng được.)
	// Set session.Token/Cookie → scheduler propagate vào result/file output.
	if result != nil && result.Status == "Live" {
		notify(tag + " LIVE → login iOS lấy token+cookie...")
		tok, ck, lerr := regmess.LoginAndGetSession(session.Proxy, session.UID, session.Password, device, family, machine, waterfall, ua)
		if lerr != nil {
			notify(tag + " login lỗi: " + lerr.Error())
		} else {
			session.Token = tok
			session.Cookie = ck
			notify(fmt.Sprintf("%s login OK — token=%.16s... cookie=%t", tag, tok, ck != ""))
		}
	}
	return result
}

// buildHeaders — iOS Messenger app-token Pando headers cho RunVerify confirm/resend POST.
func buildHeaders(sc *verifybase.SessionCtx, friendlyName string, withZeroState bool) [][2]string {
	_ = withZeroState
	return [][2]string{
		{"user-agent", sc.UA},
		{"x-graphql-client-library", "pando"},
		{"x-graphql-request-purpose", "fetch"},
		{"content-type", "application/x-www-form-urlencoded"},
		// BẮT BUỘC: verifybase.DoPost gzip-nén body trước khi gửi. Thiếu header này → FB không giải
		// nén được → đọc body gzip như form-urlencoded → "(#100) Neither query_id nor query string".
		{"content-encoding", "gzip"},
		{"accept-encoding", "gzip, deflate, br"},
		{"x-fb-friendly-name", friendlyName},
		{"authorization", "OAuth " + regmess.AppToken},
		{"x-meta-usdid-uuid", strings.ToUpper(uuid.New().String())},
		{"x-tigon-is-retry", "False"},
		{"x-fb-device-id", sc.DeviceID},
		{"x-fb-conn-uuid-client", strings.ReplaceAll(uuid.New().String(), "-", "")[:32]},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
		{"x-fb-http-engine", "Tigon/MNS/mvfst-mobile"},
	}
}

// buildAddEmailBody — AddEmail (khi RunVerify tạo mail mới, không reuse). Friendly name = contactpoint_email.
func buildAddEmailBody(spec *verifybase.Spec, emailAddr, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string {
	at := strings.SplitN(emailAddr, "@", 2)
	if len(at) != 2 {
		return ""
	}
	body, _ := regmess.BuildAddMailBody(deviceID, familyDevID, machineID, waterfallID, uid, spec.SessionlessCryptedUID, at[0], at[1], spec.Phone, spec.AACJid, spec.AACcs, spec.AACts, spec.RegFlowID, spec.HeadersFlowID, spec.PassRaw, spec.PassTS)
	return body
}

// buildConfirmBody — confirmation.async (uid + cryptedUID + email + OTP code).
func buildConfirmBody(spec *verifybase.Spec, emailAddr, code, uid, firstName, lastName, deviceID, familyDevID, waterfallID, machineID, locale string, gender int, sim fakeinfo.SimProfile) string {
	at := strings.SplitN(emailAddr, "@", 2)
	if len(at) != 2 {
		return ""
	}
	body, _ := regmess.BuildConfirmBodyVerify(deviceID, familyDevID, machineID, waterfallID, uid, spec.SessionlessCryptedUID, at[0], at[1], code, spec.AACJid, spec.AACcs, spec.AACts, spec.RegFlowID, spec.HeadersFlowID)
	return body
}

// deepUnescape — local copy (tránh export thêm).
func deepUnescape(s string) string {
	p := s
	r := strings.NewReplacer(`\\`, `\`, `\"`, `"`, `\/`, `/`)
	for i := 0; i < 8; i++ {
		n := r.Replace(p)
		if n == p {
			break
		}
		p = n
	}
	return p
}
