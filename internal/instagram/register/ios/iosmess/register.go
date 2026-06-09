package iosmess

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"

	"github.com/google/uuid"
)

// ─── Registerer ───
type Registerer struct{}

func (r *Registerer) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	return registerAccount(ctx, input, onStatus)
}

func init() {
	instagram.RegisterPlatformRegisterer(instagram.PlatformIOSMessReg, func() instagram.Registerer {
		return &Registerer{}
	})
}

// ─── WorkerContext (implements regSxxxWorkerContext) ───
type WorkerContext struct {
	proxy, countryCode, locale, ua, connType string
}

func NewWorkerContext(proxyStr, countryCode string) (*WorkerContext, error) {
	return &WorkerContext{proxy: proxyStr, countryCode: countryCode, locale: "vi_VN"}, nil
}

func (w *WorkerContext) Close()                       {}
func (w *WorkerContext) SetLocale(l string)           { if l != "" { w.locale = l } }
func (w *WorkerContext) SetConnectionType(ct string)  { w.connType = ct }
func (w *WorkerContext) SetUAOptions(bool)            {}
func (w *WorkerContext) SetUA(ua string)              { w.ua = ua }
func (w *WorkerContext) UserAgent() string {
	if w.ua != "" {
		return w.ua
	}
	return buildUA(rand.New(rand.NewSource(time.Now().UnixNano())), "reg")
}

func registerAccount(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	proxyStr := ""
	if input != nil {
		proxyStr = input.Proxy
	}
	wctx, err := NewWorkerContext(proxyStr, "")
	if err != nil {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("create worker ctx: %v", err)}
	}
	defer wctx.Close()
	return wctx.Register(ctx, input, onStatus)
}

// Register — TOÀN BỘ flow Messenger Lite iOS (email-primary):
// pre-steps → create.account → add-mail + screen-loads → GetOTP → confirm.
func (w *WorkerContext) Register(ctx context.Context, input *instagram.RegInput, onStatus func(string)) *instagram.RegResult {
	notify := func(m string) {
		if onStatus != nil {
			onStatus(m)
		}
	}
	if input == nil {
		return &instagram.RegResult{Success: false, Message: "nil input"}
	}
	// Reg phone-only: KHÔNG yêu cầu email ở create. Email (nếu có) chỉ dùng cho contactpoint;
	// flow chuẩn = reg phone → ver mới add tempmail. Email rỗng → local/domain rỗng, create dùng phone.
	local, domain := "", ""
	if input.Email != "" {
		at := strings.SplitN(input.Email, "@", 2)
		if len(at) != 2 {
			return &instagram.RegResult{Success: false, Message: "email không hợp lệ: " + input.Email}
		}
		local, domain = at[0], at[1]
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(input.SlotIdx)*7919))

	// Fingerprint — device/family iOS IDFV (UUID hoa), datr từ pool/input, waterfall mới.
	datr := input.TutDatr
	if datr == "" && SharedPool != nil {
		if d := SharedPool.GetNext(input.SlotIdx); d != "" {
			datr = d
		}
	}
	if datr == "" {
		datr = randB64(r, 24)
		notify("[iOSMess] ⚠️ Không có datr pool — dùng random (dễ integrity_block)")
	}
	if SharedPool != nil && datr != "" {
		defer SharedPool.IncrementUsage(datr)
	}
	fp := fingerprint{
		device:    strings.ToUpper(uuid.New().String()),
		family:    strings.ToUpper(uuid.New().String()),
		machine:   datr,
		waterfall: uuid.New().String(),
	}
	ua := input.UserAgent
	if ua == "" || !strings.Contains(ua, "MessengerLiteForiOS") {
		ua = buildUA(r, "reg")
	}
	pass := input.Password
	if pass == "" {
		pass = fmt.Sprintf("Mli%d%s", 1000+r.Intn(9000), randAlpha(r, 4))
	}
	fake := fakeinfo.RandomFakeProfile()
	firstName := orStr(input.FirstName, fake.FirstName)
	lastName := orStr(input.LastName, fake.LastName)
	birthday := orStr(input.Birthday, fake.Birthday)
	gender := input.Gender
	if gender == 0 {
		gender = fake.Gender
	}
	ts := time.Now().Unix()

	client, err := newClient(w.proxy, 90)
	if err != nil {
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("create client: %v", err), Password: pass}
	}

	recordPool := func(outcome string) {
		if SharedPool != nil && datr != "" {
			SharedPool.RecordResult(datr, outcome)
		}
	}

	notify(fmt.Sprintf("[iOSMess] Start — %s %s | %s | datr=%s", firstName, lastName, input.Email, shortStr(datr, 8)))

	// AAC (Account Access Context) — sinh 1 bộ TƯƠI/account, thread y hệt qua TẤT CẢ bước
	// (pre-steps → create → add-mail → confirm). Thay aac chết trong template; nếu để aac chết,
	// server không bind được session → add-mail render lại form 9KB thay vì gửi OTP.
	aacJid, aacCs, aacTs := genAACParts()
	notify(fmt.Sprintf("[iOSMess] AAC minted: jid=%s ts=%s", shortStr(aacJid, 8), aacTs))

	// Flow-session IDs — mint 1 bộ UUID/account, thread y hệt mọi bước (giống AAC).
	// Bind userid vào flow sống; literal chết trong template → server render lại form 9KB.
	regFlowID, headersFlowID := uuid.New().String(), uuid.New().String()
	notify(fmt.Sprintf("[iOSMess] flow_id minted: reg=%s hdr=%s", shortStr(regFlowID, 8), shortStr(headersFlowID, 8)))

	// TEST attestation: pin device+family+aac = capture THẬT (aaccs là attestation hợp lệ cho
	// device F06B). Nếu add-mail ra 205KB khi pin → xác nhận aaccs attestation là cổng chặn OTP.
	if os.Getenv("HVR_PIN_CAPTURE") == "1" {
		fp.device = capDeviceID
		fp.family = capFamilyID
		aacJid, aacCs, aacTs = capAACjid, capAACcs, capAACts
		notify("[iOSMess] PIN_CAPTURE: device+aac = capture thật (test attestation gate)")
	}

	// ── Pre-steps (email-primary) ──
	sd0 := stepData{fp: fp, firstName: firstName, lastName: lastName, birthday: birthday,
		gender: gender,
		pass:   pass, ts: ts, emailLoc: local, emailDom: domain,
		aacjid: aacJid, aaccs: aacCs, aacTs: aacTs,
		regFlowID: regFlowID, headersFlowID: headersFlowID}
	// THỨ TỰ GOOD (phone-first): ...→gender→contactpoint_PHONE→password→create.
	// Trước đây dùng contactpoint_EMAIL ở đây (sót từ email-first) → account tạo với contactpoint
	// = email pending, rồi post-create lại "đổi sang email" → server loạn state → add-mail bounce.
	preSteps := []struct{ tpl, fn string }{
		{"aymh", fnAymh}, {"ntm", fnNtm}, {"genphone", fnGenPhone},
		{"name", fnName}, {"birthday", fnBirthday}, {"gender", fnGender},
		{"contactpoint_phone", fnContactPhone}, {"password", fnPassword},
	}
	var chainRegCtx string // reg_context tươi nhất server cấp qua các pre-steps (E-RC chain)
	for _, s := range preSteps {
		select {
		case <-ctx.Done():
			return &instagram.RegResult{Success: false, Message: "ctx cancelled", Password: pass}
		default:
		}
		b, berr := buildStep(s.tpl, sd0)
		if berr != nil {
			return &instagram.RegResult{Success: false, Message: berr.Error(), Password: pass}
		}
		_, presp, _ := sendBloks(client, b, s.fn, fp.device, ua) // pre-steps best-effort
		if v := extractRegContextIOS(presp); v != "" {
			chainRegCtx = v // chain: giữ reg_context mới nhất cho create
		}
		time.Sleep(300 * time.Millisecond)
	}
	notify(fmt.Sprintf("[iOSMess] reg_context chained: %d bytes", len(chainRegCtx)))

	// ── create.account ──
	// RegMode=Phone: dùng SĐT từ list của user (E.164 +cc...); RegMode=TempMail: random VN.
	// Phone là contactpoint lúc create + msg_previous_cp ở add-mail (ver đổi phone→email).
	phone := strings.TrimSpace(input.Phone)
	if phone == "" {
		phone = randVNPhone(r)
	} else if !strings.HasPrefix(phone, "+") {
		phone = "+" + phone
	}
	createBody, err := buildCreateBody(fp, local, domain, pass, ts, chainRegCtx, phone, aacJid, aacCs, aacTs, regFlowID, headersFlowID)
	if err != nil {
		return &instagram.RegResult{Success: false, Message: err.Error(), Password: pass}
	}
	if os.Getenv("HVR_SAVE_CREATE") == "1" {
		_ = os.WriteFile(fmt.Sprintf("createreq_%d.txt", input.SlotIdx), []byte(createBody), 0o644)
	}
	st, raw, err := sendBloks(client, createBody, fnCreate, fp.device, ua)
	if err != nil {
		recordPool("unknown")
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("create HTTP: %v", err), Password: pass}
	}
	if os.Getenv("HVR_SAVE_CREATE") == "1" {
		_ = os.WriteFile(fmt.Sprintf("createresp_%d.json", input.SlotIdx), []byte(raw), 0o644)
	}
	if strings.Contains(strings.ToLower(raw), "integrity_block") {
		recordPool("checkpoint")
		return &instagram.RegResult{Success: false, Message: "integrity_block (create)", Password: pass}
	}
	uid := extractUID(raw)
	if uid == "" {
		recordPool("fail")
		return &instagram.RegResult{Success: false, Message: fmt.Sprintf("create no-uid (HTTP %d, %d bytes)", st, len(raw)), Password: pass}
	}
	cryptedUID := extractCryptedUID(raw)
	if cryptedUID == "" {
		// Không có crypted_user_id → verify add-mail sẽ fail (server gate). Vẫn trả uid để log.
		notify("[iOSMess] ⚠️ create OK nhưng KHÔNG bóc được crypted_user_id")
	}
	notify(fmt.Sprintf("[iOSMess] create OK — UID=%s crypted=%s", uid, shortStr(cryptedUID, 12)))

	// REG create-ONLY: trả uid + crypted_uid + Email/EmailMeta cho VERIFY add-mail+confirm.
	// (verify reuse mail qua EmailMeta → add-mail(crypted_uid) → OTP → confirm → live/die)
	recordPool("success")
	return buildResult(true, uid, pass, ts, ua, fp, cryptedUID, input.Email, input.EmailMeta, phone,
		aacJid, aacCs, aacTs, regFlowID, headersFlowID,
		fmt.Sprintf("iOS Mess reg OK (create) — UID %s, chờ verify confirm", uid))
}

func buildResult(ok bool, uid, pass string, passTS int64, ua string, fp fingerprint, cryptedUID, email, emailMeta, phone, aacjid, aaccs, aacTs, regFlowID, headersFlowID, msg string) *instagram.RegResult {
	return &instagram.RegResult{
		Success:               ok,
		UID:                   uid,
		Password:              pass,
		Cookie:                "datr=" + fp.machine, // verify đọc session.Datr
		UserAgent:             ua,
		DeviceID:              fp.device,
		FamilyDeviceID:        fp.family,
		SessionlessCryptedUID: cryptedUID,
		Srnonce:               fp.waterfall,
		Email:                 email,
		EmailMeta:             emailMeta,
		Phone:                 phone,
		AACJid:                aacjid,
		AACcs:                 aaccs,
		AACts:                 aacTs,
		RegFlowID:             regFlowID,
		HeadersFlowID:         headersFlowID,
		PassRaw:               pass,
		PassTS:                passTS,
		Message:               msg,
	}
}

// ─── helpers ───
func orStr(v, def string) string {
	if v != "" {
		return v
	}
	return def
}

func shortStr(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

func randB64(r *rand.Rand, n int) string {
	const c = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	b := make([]byte, n)
	for i := range b {
		b[i] = c[r.Intn(len(c))]
	}
	return string(b)
}

func randAlpha(r *rand.Rand, n int) string {
	const c = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = c[r.Intn(len(c))]
	}
	return string(b)
}
