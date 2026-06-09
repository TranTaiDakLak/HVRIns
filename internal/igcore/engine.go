// engine.go — orchestrate các step Bloks, carry-over reg_context.
package igcore

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// errThrottled — IG throttle tier contactpoint (lỗi tạm thời, nên retry với IP khác).
var errThrottled = errors.New("THROTTLED")

type engine struct {
	sess     *igSession
	p        *igProfile
	log      func(string, ...any)
	keyID    string
	pubKey   string
	proxyStr string // base proxy (để rotate IP khi retry create.account)

	regContext       string // server-issued, carry-over giữa các step
	confirmationCode string // token IG cấp sau confirmOTP — phải inject vào create.account
	Session          IGSession // cookies sau khi createAccount thành công
	lastResp         string
}

// aymh — bước vào flow: nhấn "Tạo tài khoản mới".
func (e *engine) aymh(ctx context.Context) error {
	body, err := loadTemplate("01_aymh.txt")
	if err != nil {
		return fmt.Errorf("load template: %w", err)
	}
	body = applyProfile(body, e.p)
	jid, ccs, _ := newAAC()
	body = setAAC(body, jid, ccs)

	resp, hdr, err := e.sess.post(ctx, graphqlPath, body,
		bloksHeaders(e.p, "com.bloks.www.bloks.caa.reg.aymh_create_account_button.async"))
	e.lastResp = resp
	if err != nil {
		e.log("  aymh HTTP err: %v (resp %d bytes)", err, len(resp))
	}
	// refresh X-MID nếu server set mới
	if m := hdr.Get("ig-set-x-mid"); m != "" {
		e.p.MachineID = m
	}
	if rc := parseRegContext(resp); rc != "" {
		e.regContext = rc
		e.log("  aymh → reg_context %d chars", len(rc))
	}
	if e.isBlocked(resp) {
		return fmt.Errorf("blocked: %s", e.shortErr(resp))
	}
	if !hasMarker(resp, "contactpoint") && !hasMarker(resp, "bloks_payload") {
		return fmt.Errorf("aymh response không có marker mong đợi (resp %d bytes): %s", len(resp), e.shortErr(resp))
	}
	return nil
}

// renderScreen — gọi 1 màn Bloks render (contactpoint_phone / contactpoint_email)
// để mô phỏng user đi qua form trước khi submit. Không bắt buộc thành công, chỉ
// để server thấy nav-chain.
func (e *engine) renderScreen(ctx context.Context, tmpl, appID string) {
	body, err := loadTemplate(tmpl)
	if err != nil {
		return
	}
	body = applyProfile(body, e.p)
	jid, ccs, _ := newAAC()
	body = setAAC(body, jid, ccs)
	resp, hdr, _ := e.sess.post(ctx, graphqlPath, body, bloksHeaders(e.p, appID))
	if m := hdr.Get("ig-set-x-mid"); m != "" {
		e.p.MachineID = m
	}
	if rc := parseRegContext(resp); rc != "" {
		e.regContext = rc
	}
	e.log("  render %s (resp %d bytes)", appID, len(resp))
}

// submitEmail — gọi async.contactpoint_email.async với email → IG gửi OTP.
func (e *engine) submitEmail(ctx context.Context, addr string) error {
	body, err := loadTemplate("04_submit_email.txt")
	if err != nil {
		return fmt.Errorf("load template: %w", err)
	}
	body = applyProfile(body, e.p)
	body = stripAccountsList(body) // account mới → list rỗng (bỏ token thật của máy capture)
	body = setEmail(body, addr)
	body = setEventReqID(body, uuid.New().String())
	jid, ccs, _ := newAAC()
	body = setAAC(body, jid, ccs)
	if e.regContext != "" {
		body = injectRegContext(body, e.regContext)
	}
	if dbg := dumpDir(); dbg != "" {
		writeDebug(dbg, "submit_email_req.txt", body)
	}

	resp, hdr, err := e.sess.post(ctx, graphqlPath, body,
		bloksHeaders(e.p, "com.bloks.www.bloks.caa.reg.async.contactpoint_email.async"))
	e.lastResp = resp
	if err != nil {
		e.log("  submitEmail HTTP err: %v (resp %d bytes)", err, len(resp))
	}
	if m := hdr.Get("ig-set-x-mid"); m != "" {
		e.p.MachineID = m
	}
	if rc := parseRegContext(resp); rc != "" {
		e.regContext = rc
		e.log("  submitEmail → reg_context %d chars", len(rc))
	}
	if dbg := dumpDir(); dbg != "" {
		writeDebug(dbg, "submit_email_resp.txt", resp)
	}
	if e.isBlocked(resp) {
		return fmt.Errorf("blocked: %s", e.shortErr(resp))
	}
	// Marker thành công: push màn confirmation HOẶC text "đã gửi".
	if hasMarker(resp, "confirmation") || hasMarker(resp, "đã gửi") || hasMarker(resp, "BloksCAARegConfirmation") {
		e.log("  submitEmail SUCCESS — resp %d bytes (confirmation screen)", len(resp))
		return nil
	}
	if hasMarker(resp, "login_upsell") || hasMarker(resp, "existing_profile") {
		return fmt.Errorf("email đã tồn tại (login_upsell) — dùng email khác")
	}
	if dbg := dumpDir(); dbg != "" {
		writeDebug(dbg, "submit_email_resp.txt", resp)
	}
	// Throttling phía IG (tier ig_feta_contactpoint_api) — lỗi TẠM THỜI, retry IP khác.
	if hasMarker(resp, "THROTTLING_REQUEST_GLOBAL") || hasMarker(resp, "throttling request") {
		return errThrottled
	}
	if hasMarker(resp, "USER_REGISTER_INVALID_EMAIL") {
		if dbg := dumpDir(); dbg != "" {
			writeDebug(dbg, "invalid_email_resp.txt", resp)
		}
		// Bóc lý do thật trong action (throttle? email taken? domain blocked?).
		reason := extractInvalidReason(resp)
		e.log("  INVALID_EMAIL reason: %s", reason)
		// Nếu là throttle trá hình → retry như throttle.
		if hasMarker(resp, "throttl") || reason == "" {
			return errThrottled
		}
		return fmt.Errorf("INVALID_EMAIL: %s", reason)
	}
	return fmt.Errorf("submitEmail không thấy marker confirmation (resp %d bytes): %s", len(resp), e.shortErr(resp))
}

// confirmOTP — gọi confirmation.async với mã OTP, dùng reg_context của phiên submit.
func (e *engine) confirmOTP(ctx context.Context, addr, code string) error {
	body, err := loadTemplate("05_confirm.txt")
	if err != nil {
		return fmt.Errorf("load template: %w", err)
	}
	body = applyProfile(body, e.p)
	body = stripAccountsListConfirm(body)
	body = setEmail(body, addr)
	body = setCode(body, code)
	body = setEventReqID(body, uuid.New().String())
	jid, ccs, _ := newAAC()
	body = setAAC(body, jid, ccs)
	// reg_context: server cấp ở response submit (e.regContext) → inject (encode | thành %7C).
	if e.regContext != "" {
		enc := strings.ReplaceAll(e.regContext, "|", "%7C")
		body = setRegContextRaw(body, enc)
		e.log("  confirm dùng reg_context %d chars", len(e.regContext))
	}

	if dbg := dumpDir(); dbg != "" {
		writeDebug(dbg, "confirm_req.txt", body)
		// Verify code present in body
		if !strings.Contains(body, code) {
			e.log("  ⚠️ CODE %q KHÔNG nằm trong confirm body — setCode lỗi!", code)
		} else {
			e.log("  ✅ code %q xác nhận có trong body", code)
		}
	}
	resp, _, err := e.sess.post(ctx, graphqlPath, body,
		bloksHeaders(e.p, "com.bloks.www.bloks.caa.reg.confirmation.async"))
	if dbg := dumpDir(); dbg != "" {
		writeDebug(dbg, "confirm_resp.txt", resp)
	}
	if err != nil {
		e.log("  confirm HTTP err: %v (resp %d bytes)", err, len(resp))
	}
	if hasMarker(resp, "THROTTLING_REQUEST_GLOBAL") || hasMarker(resp, "throttling request") {
		return errThrottled
	}
	if hasMarker(resp, "integrity_block") {
		return fmt.Errorf("integrity_block: %s", e.shortErr(resp))
	}
	// Thành công: is_confirmed=true / push màn password / gen_next_screen.
	low := strings.ToLower(strings.ReplaceAll(resp, `\`, ""))
	if strings.Contains(low, "is_confirmed\"") && strings.Contains(low, "true") &&
		!strings.Contains(low, "the confirmation code you entered is invalid") {
		// Heuristic — cần xem response để chắc; nhưng nếu có password screen thì chắc.
	}
	if hasMarker(resp, "BloksCAARegPassword") || hasMarker(resp, "reg.password") || hasMarker(resp, "gen_next_screen") {
		// Parse confirmation_code token — IG cấp sau xác nhận OTP, dùng trong create.account.
		if cc := parseConfirmationCode(resp); cc != "" {
			e.confirmationCode = cc
			e.log("  confirmOTP → confirmation_code %s", cc)
		}
		e.log("  confirmOTP SUCCESS — push màn password (resp %d bytes)", len(resp))
		return nil
	}
	if hasMarker(resp, "confirmation code you entered is invalid") || hasMarker(resp, "invalid or has expired") {
		return fmt.Errorf("OTP SAI/HẾT HẠN")
	}
	return fmt.Errorf("confirmOTP không rõ kết quả (resp %d bytes): %s", len(resp), e.shortErr(resp))
}

// step thực hiện 1 bước Bloks chung: apply profile/email + strip accounts + inject reg_context,
// POST rồi parse reg_context từ response. Trả (respBody, error).
func (e *engine) step(ctx context.Context, tmpl, appID, addr string, extra func(string) string) (string, error) {
	body, err := loadTemplate(tmpl)
	if err != nil {
		return "", fmt.Errorf("load %s: %w", tmpl, err)
	}
	body = applyProfile(body, e.p)
	body = setEmail(body, addr)
	// strip accounts_list (thử cả 2 anchor)
	body = stripAccountsListConfirm(body)
	if e.regContext != "" {
		body = setRegContextRaw(body, strings.ReplaceAll(url.QueryEscape(e.regContext), "+", "%20"))
		body = setRegContextCreate(body, e.regContext)
	}
	if extra != nil {
		body = extra(body)
	}
	if dbg := dumpDir(); dbg != "" {
		writeDebug(dbg, appID+"_req.txt", body)
	}
	resp, _, err := e.sess.post(ctx, graphqlPath, body, bloksHeaders(e.p, appID))
	if dbg := dumpDir(); dbg != "" {
		writeDebug(dbg, appID+"_resp.txt", resp)
	}
	if err != nil {
		e.log("  %s HTTP err: %v (resp %d bytes)", appID, err, len(resp))
	}
	if rc := parseRegContext(resp); rc != "" {
		e.regContext = rc
		e.log("  %s → reg_context %d chars", appID, len(rc))
	}
	if hasMarker(resp, "THROTTLING_REQUEST_GLOBAL") || hasMarker(resp, "throttling request") {
		return resp, errThrottled
	}
	if e.isBlocked(resp) {
		return resp, fmt.Errorf("blocked: %s", e.shortErr(resp))
	}
	return resp, nil
}

// setPassword — bước đặt mật khẩu.
func (e *engine) setPassword(ctx context.Context, addr, password string) error {
	encPwd, err := encryptPasswordInstagram(password, e.pubKey, e.keyID)
	if err != nil {
		return fmt.Errorf("encrypt password: %w", err)
	}
	e.log("  password encrypted (#PWD_INSTAGRAM:4 OK)")
	appID := "com.bloks.www.bloks.caa.reg.password.async"
	resp, err := e.step(ctx, "06_password.txt", appID, addr, func(b string) string {
		return setEncryptedPassword(b, encPwd)
	})
	if err != nil {
		return err
	}
	if hasMarker(resp, "birthday") || hasMarker(resp, "gen_next_screen") || len(resp) > 50000 {
		e.log("  setPassword OK (resp %d bytes)", len(resp))
		return nil
	}
	return fmt.Errorf("setPassword không rõ kết quả (resp %d bytes): %s", len(resp), e.shortErr(resp))
}

// setBirthday — bước nhập ngày sinh.
func (e *engine) setBirthday(ctx context.Context, addr string) error {
	// Sinh ngày sinh random 20-35 tuổi (age_range=o18).
	year := 1990 + nowUnix()%15 // 1990-2004
	ts := int64(year-1970)*365*24*3600 + int64(nowUnix()%1000000)
	ddmmyyyy := fmt.Sprintf("15-06-%d", year)
	appID := "com.bloks.www.bloks.caa.reg.birthday.async"
	resp, err := e.step(ctx, "07_birthday.txt", appID, addr, func(b string) string {
		return setBirthday(b, ddmmyyyy, ts)
	})
	if err != nil {
		return err
	}
	if hasMarker(resp, "name") || hasMarker(resp, "gen_next_screen") || len(resp) > 50000 {
		e.log("  setBirthday OK (resp %d bytes)", len(resp))
		return nil
	}
	return fmt.Errorf("setBirthday không rõ kết quả (resp %d bytes): %s", len(resp), e.shortErr(resp))
}

// setName — bước đặt tên.
func (e *engine) setNameIG(ctx context.Context, addr, name string) error {
	appID := "com.bloks.www.bloks.caa.reg.name_ig_and_soap.async"
	resp, err := e.step(ctx, "08_name.txt", appID, addr, func(b string) string {
		return setName(b, name)
	})
	if err != nil {
		return err
	}
	if hasMarker(resp, "username") || hasMarker(resp, "gen_next_screen") || len(resp) > 50000 {
		e.log("  setName OK (resp %d bytes)", len(resp))
		return nil
	}
	return fmt.Errorf("setName không rõ kết quả (resp %d bytes): %s", len(resp), e.shortErr(resp))
}

// setUsername — bước đặt username.
func (e *engine) setUsername(ctx context.Context, addr, username string) error {
	appID := "com.bloks.www.bloks.caa.reg.username.async"
	resp, err := e.step(ctx, "09_username.txt", appID, addr, func(b string) string {
		return setUsername(b, username)
	})
	if err != nil {
		return err
	}
	if hasMarker(resp, "create") || hasMarker(resp, "tos") || hasMarker(resp, "gen_next_screen") || len(resp) > 50000 {
		e.log("  setUsername OK (resp %d bytes)", len(resp))
		return nil
	}
	return fmt.Errorf("setUsername không rõ kết quả (resp %d bytes): %s", len(resp), e.shortErr(resp))
}

// acceptTOS — bước chấp nhận Terms of Service (giữa setUsername và createAccount).
// Dùng username template với TOS app_id — state variables đều null nên body tương thích.
func (e *engine) acceptTOS(ctx context.Context, addr string) error {
	appID := "com.bloks.www.bloks.caa.reg.tos"
	resp, err := e.step(ctx, "09_username.txt", appID, addr, nil)
	if err != nil {
		// TOS có thể fail nhẹ mà không cần retry — log và tiếp tục
		e.log("  acceptTOS warn: %v (resp %d bytes) — tiếp tục", err, len(resp))
		return nil
	}
	e.log("  acceptTOS OK (resp %d bytes)", len(resp))
	return nil
}

// errEmailDomainRejected — IG từ chối email domain tại create.account (domain bị blacklist).
// Không retry được vì lỗi permanent cho domain này.
var errEmailDomainRejected = errors.New("EMAIL_DOMAIN_REJECTED")

// createAccount — bước cuối tạo account, retry nếu throttle hoặc integrity_block tạm thời.
// username + name dùng để thay capture values trong create body.
func (e *engine) createAccount(ctx context.Context, addr, username, name string) error {
	const maxCreate = 8
	for i := 1; i <= maxCreate; i++ {

		resp, err := e.stepCreate(ctx, addr, username, name, i)

		if err == errThrottled {
			e.log("  createAccount attempt %d: throttle → retry", i)
			backoffCreate(i)
			continue
		}
		if err == errEmailDomainRejected {
			return err
		}
		if err != nil {
			// integrity_block mà KHÔNG có create_failure → real block screen, retry same session.
			msg := err.Error()
			if strings.Contains(msg, "blocked") {
				e.log("  createAccount attempt %d: %v → backoff retry (same session)", i, err)
				backoffCreate(i)
				continue
			}
			return err
		}
		// Parse UID từ response.
		uid := extractUID(resp)
		if uid != "" {
			e.Session = parseIGSession(resp)
			e.log("  🎉 createAccount SUCCESS — UID: %s (resp %d bytes)", uid, len(resp))
			return nil
		}
		if hasMarker(resp, "create_success") || hasMarker(resp, "sessionless_login_on_completion") {
			e.log("  🎉 createAccount SUCCESS (no uid parsed, resp %d bytes)", len(resp))
			return nil
		}
		if hasMarker(resp, "integrity_block") {
			e.log("  createAccount attempt %d: integrity_block (no create_failure) → retry", i)
			backoffCreate(i)
			continue
		}
		return fmt.Errorf("createAccount không rõ kết quả attempt %d (resp %d bytes): %s", i, len(resp), e.shortErr(resp))
	}
	return fmt.Errorf("createAccount thất bại sau %d lần", maxCreate)
}

// stepCreate — step riêng cho create.account, detect email domain rejection trước khi isBlocked.
func (e *engine) stepCreate(ctx context.Context, addr, username, name string, attempt int) (string, error) {
	appID := "com.bloks.www.bloks.caa.reg.create.account.async"
	body, err := loadTemplate("10_create.txt")
	if err != nil {
		return "", fmt.Errorf("load 10_create.txt: %w", err)
	}
	body = applyProfile(body, e.p)
	body = setEmail(body, addr)
	body = stripAccountsListConfirm(body)
	if e.regContext != "" {
		body = setRegContextRaw(body, strings.ReplaceAll(url.QueryEscape(e.regContext), "+", "%20"))
		body = setRegContextCreate(body, e.regContext)
	}
	body = setEventReqID(body, uuid.New().String())
	body = setUsername(body, username)
	body = setName(body, name)
	if e.confirmationCode != "" {
		body = setConfirmationCode(body, e.confirmationCode)
	}

	if dbg := dumpDir(); dbg != "" {
		writeDebug(dbg, fmt.Sprintf("create_attempt_%d_req.txt", attempt), body)
	}

	resp, _, err2 := e.sess.post(ctx, graphqlPath, body, bloksHeaders(e.p, appID))
	if dbg := dumpDir(); dbg != "" {
		writeDebug(dbg, fmt.Sprintf("create_attempt_%d_resp.txt", attempt), resp)
	}
	if err2 != nil {
		e.log("  create attempt %d HTTP err: %v (resp %d bytes)", attempt, err2, len(resp))
	}
	if rc := parseRegContext(resp); rc != "" {
		e.regContext = rc
	}

	// Email domain blacklisted: IG trả user_input_error + create_failure + notif_medium=email.
	// Đây là lỗi PERMANENT cho domain này → không retry.
	if hasMarker(resp, "user_input_error") && hasMarker(resp, "create_failure") {
		e.log("  create attempt %d: EMAIL DOMAIN REJECTED (user_input_error+create_failure)", attempt)
		return resp, errEmailDomainRejected
	}

	if hasMarker(resp, "THROTTLING_REQUEST_GLOBAL") || hasMarker(resp, "throttling request") {
		return resp, errThrottled
	}
	// isBlocked chỉ trigger khi KHÔNG có create_failure (tức là response thật sự là block screen).
	if e.isBlocked(resp) && !hasMarker(resp, "create_failure") {
		return resp, fmt.Errorf("blocked: %s", e.shortErr(resp))
	}
	return resp, nil
}

func backoffCreate(i int) {
	d := 3 * time.Second
	if i > 5 {
		d = 8 * time.Second
	}
	time.Sleep(d)
}

func extractUID(resp string) string {
	clean := strings.ReplaceAll(resp, `\`, "")
	if m := regexp.MustCompile(`\(eud (\d{8,})\)`).FindStringSubmatch(clean); len(m) > 1 {
		return m[1]
	}
	if m := regexp.MustCompile(`c_user","value":"(\d{8,})"`).FindStringSubmatch(clean); len(m) > 1 {
		return m[1]
	}
	if m := regexp.MustCompile(`"pk":(\d{8,})`).FindStringSubmatch(clean); len(m) > 1 {
		return m[1]
	}
	return ""
}

func (e *engine) isBlocked(resp string) bool {
	return hasMarker(resp, "integrity_block") || hasMarker(resp, "thử lại sau") ||
		hasMarker(resp, "rate") && hasMarker(resp, "limit")
}

func (e *engine) shortErr(resp string) string {
	if m := extractError(resp); m != "" {
		return m
	}
	if len(resp) > 200 {
		return resp[:200]
	}
	return resp
}
