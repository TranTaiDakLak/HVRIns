package iosmess

import (
	crand "crypto/rand"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	endpoint = "https://graph.facebook.com/graphql"
	appToken = "437626316973788|3e1a7033ae7883bfb31f35375bad9c7a"

	fnCreate     = "MSGBloksActionRootQuery-com.bloks.www.bloks.caa.reg.create.account.async"
	fnConfirm    = "MSGBloksActionRootQuery-com.bloks.www.bloks.caa.reg.confirmation.async"
	fnAddMail    = "MSGBloksActionRootQuery-com.bloks.www.bloks.caa.reg.async.contactpoint_email.async"
	fnAymh       = "MSGBloksActionRootQuery-com.bloks.www.bloks.caa.reg.aymh_create_account_button.async"
	fnNtm        = "MSGBloksActionRootQuery-com.bloks.www.bloks.caa.reg.async.expose_ntm_experiment.async"
	fnGenPhone     = "MSGBloksActionRootQuery-com.bloks.www.bloks.caa.reg.async.gen_unmasked_phone.async"
	fnContactPhone = "MSGBloksActionRootQuery-com.bloks.www.bloks.caa.reg.async.contactpoint_phone.async"
	fnName       = "MSGBloksActionRootQuery-com.bloks.www.bloks.caa.reg.name.async"
	fnBirthday   = "MSGBloksActionRootQuery-com.bloks.www.bloks.caa.reg.birthday.async"
	fnGender     = "MSGBloksActionRootQuery-com.bloks.www.bloks.caa.reg.gender.async"
	fnPassword   = "MSGBloksActionRootQuery-com.bloks.www.bloks.caa.reg.password.async"
	fnBottomshee = "BKAppRootQuery-com.bloks.www.bloks.caa.reg.confirmation.fb.bottomsheet"
	fnChangeMail = "BKAppRootQuery-com.bloks.www.bloks.caa.reg.confirmation.change.email"

	// ─── Giá trị CAPTURED cần thay (literal y hệt trong body encoded) ───
	capDeviceID   = "F06B6E7B-AB90-4296-81B0-F80A28126D15"
	capFamilyID   = "37312B3B-7404-491B-967D-72AE026CE3A7"
	capMachineID  = "AZoRau6llXrFeEQB5J1EOuJI"
	capWaterfall  = "53051f88-5332-4e35-82c6-653a0de51458"
	capEventReq   = "8856546d-27fb-4ac4-9b20-f52b3dd0f90f"
	capPwdLit     = "%23PWD_ENC%3A0%3A1780631473%3Along%5C%5C%5C%5C%5C%5C%5C%5Cu00400000"
	capPhone      = "%2B84985236417"
	capPhoneRaw   = "84985236417"
	capTypePhone  = "%22phone%5C%5C%5C%5C%5C%5C%5C%22"
	capEmailLoc   = "wyxaheh"
	capEmailDom   = "any.pink"
	capCode       = "42573"
	capUID        = "61590754915279" // user_id acc capture (confirm/addmail)
	capUID2       = "61590261616638" // user_id trong create template
	capCryptedUID = "AYgzuJOS-rwhBMZGNGPyvQfdkWQtlPFLUcdBbiNLW1EBlJd2YV2Ww4PpNU5BuJCPFIjzxoTfafAu0CwWT5Lo7e7MVvrKpnVr3O3QfnOfEzcYMQ"
	capFirstName  = "Bui"
	capLastName   = "Loi"
	capBirthday   = "28-02-1998"
	capGender     = "gender%5C%5C%5C%22%3A1" // gender:1 (nữ) trong gender.txt — substitute khi gender=2 (nam)
	emailAtEsc    = "%5C%5C%5C%5C%5C%5C%5C%5Cu0040"

	// AAC (Account Access Context) literals trong template capture — đã CHẾT.
	// Phải thay bằng aac TƯƠI client-mint, nhất quán xuyên 1 session (create→add-mail→confirm).
	// Server coi aac là token opaque client tự mint (giống sibling Android genAAC: aaccs random
	// 32 byte base64url). Gửi aac chết ghép crypted_user_id LIVE → cặp không khớp → server render
	// lại form (9KB) thay vì gửi OTP (204KB). Đây là blocker chính của add-mail.
	capAACjid = "dceb25df-a981-4dd3-b4ee-5b8b5bfd4042"
	capAACcs  = "TukJM7dr51_UbHi2D5Ibr_Ta_71ccfXhgJKITsNPIVA"
	capAACts  = "1780630812"

	// Flow-session IDs literal trong template capture — CHẾT (per-session, không phải per-device).
	// Server dùng registration_flow_id để bind userid vào flow đang chạy; flow_id chết (đã dùng ở
	// session capture cũ) → server không khớp được → state=UNKNOWN, render lại form 9KB thay vì OTP.
	// Phải mint UUID tươi 1 bộ/account, thread y hệt create→bottomsheet→change.email→add-mail→confirm
	// (giống GOOD: 0de0dad7/67a91b82 không đổi xuyên các bước). Client tự mint (xuất hiện ngay ở
	// request bottomsheet đầu, server chỉ echo lại).
	capRegFlowID     = "175bfeb3-68ba-4d04-9332-e60c29c23c77"
	capHeadersFlowID = "2fcd71da-3708-4c3c-b84b-debb5d798c8e"
)

type fingerprint struct{ device, family, machine, waterfall string }

// stepData — giá trị động cho 1 bước Bloks.
type stepData struct {
	fp         fingerprint
	phone      string // +84XXXXXXXXX
	firstName  string
	lastName   string
	birthday   string // dd-MM-yyyy
	gender     int    // 1=nữ, 2=nam (khớp tên random; default template = 1)
	pass       string
	ts         int64
	uid        string
	cryptedUID string // crypted_user_id từ create.account — gate của add-mail/confirm
	emailLoc   string
	emailDom   string
	code       string // OTP
	// AAC session token (1 bộ duy nhất/account, thread create→add-mail→confirm).
	aacjid string
	aaccs  string
	aacTs  string
	// Flow-session IDs (1 bộ/account, thread y hệt mọi bước — bind userid vào flow sống).
	regFlowID     string
	headersFlowID string
}

// buildStep thay toàn bộ dynamic fields trong template body (generic mọi bước).
func buildStep(tplName string, sd stepData) (string, error) {
	body, err := loadTemplate(tplName)
	if err != nil {
		return "", fmt.Errorf("load template %s: %w", tplName, err)
	}
	body = strings.ReplaceAll(body, capDeviceID, sd.fp.device)
	body = strings.ReplaceAll(body, capFamilyID, sd.fp.family)
	body = strings.ReplaceAll(body, capMachineID, sd.fp.machine)
	body = strings.ReplaceAll(body, capWaterfall, sd.fp.waterfall)
	body = strings.ReplaceAll(body, capEventReq, uuid.New().String())
	if sd.pass != "" && sd.ts > 0 {
		body = strings.ReplaceAll(body, capPwdLit,
			fmt.Sprintf("%%23PWD_ENC%%3A0%%3A%d%%3A%s", sd.ts, sd.pass))
	}
	if sd.phone != "" {
		newPhoneEnc := strings.ReplaceAll(url.QueryEscape(sd.phone), "+", "%2B")
		body = strings.ReplaceAll(body, capPhone, newPhoneEnc)
		body = strings.ReplaceAll(body, capPhoneRaw, strings.TrimPrefix(sd.phone, "+"))
	}
	if sd.firstName != "" {
		body = strings.ReplaceAll(body, capFirstName, sd.firstName)
	}
	if sd.lastName != "" {
		body = strings.ReplaceAll(body, capLastName, sd.lastName)
	}
	if sd.birthday != "" {
		body = strings.ReplaceAll(body, capBirthday, sd.birthday)
	}
	if sd.gender == 2 { // template mặc định gender:1 (nữ); chỉ thay khi nam → khớp tên random
		body = strings.ReplaceAll(body, capGender, "gender%5C%5C%5C%22%3A2")
	}
	if sd.uid != "" {
		body = strings.ReplaceAll(body, capUID, sd.uid)
		body = strings.ReplaceAll(body, capUID2, sd.uid)
	}
	if sd.cryptedUID != "" {
		body = strings.ReplaceAll(body, capCryptedUID, sd.cryptedUID)
	}
	// AAC: thay aac chết của template bằng aac tươi (cùng 1 bộ xuyên session).
	if sd.aacjid != "" {
		body = strings.ReplaceAll(body, capAACjid, sd.aacjid)
	}
	if sd.aaccs != "" {
		body = strings.ReplaceAll(body, capAACcs, sd.aaccs)
	}
	if sd.aacTs != "" {
		body = strings.ReplaceAll(body, capAACts, sd.aacTs)
	}
	// Flow-session IDs: thay literal chết bằng UUID tươi (cùng bộ xuyên session).
	if sd.regFlowID != "" {
		body = strings.ReplaceAll(body, capRegFlowID, sd.regFlowID)
	}
	if sd.headersFlowID != "" {
		body = strings.ReplaceAll(body, capHeadersFlowID, sd.headersFlowID)
	}
	if sd.emailLoc != "" {
		body = strings.ReplaceAll(body, capEmailLoc, sd.emailLoc)
	}
	if sd.emailDom != "" {
		body = strings.ReplaceAll(body, capEmailDom, sd.emailDom)
	}
	if sd.code != "" {
		body = strings.ReplaceAll(body, capCode, sd.code)
	}
	return body, nil
}

// buildCreateBody — create.account.
// E1 PHONE-FIRST (bám capture GOOD): create bằng SĐT VN fresh + GIỮ contactpoint_type=phone.
// Email được add ở bước verify sau (change.email → contactpoint_email → confirm OTP).
// (Trước đây email-first: nhét email vào chỗ phone + đổi type→email — bị integrity_block 100%.)
func buildCreateBody(fp fingerprint, local, domain, pass string, ts int64, regCtx, phone, aacjid, aaccs, aacTs, regFlowID, headersFlowID string) (string, error) {
	body, err := loadTemplate("create")
	if err != nil {
		return "", err
	}
	body = strings.ReplaceAll(body, capDeviceID, fp.device)
	body = strings.ReplaceAll(body, capFamilyID, fp.family)
	body = strings.ReplaceAll(body, capMachineID, fp.machine)
	body = strings.ReplaceAll(body, capWaterfall, fp.waterfall)
	body = strings.ReplaceAll(body, capEventReq, uuid.New().String())
	body = strings.ReplaceAll(body, capPwdLit,
		fmt.Sprintf("%%23PWD_ENC%%3A0%%3A%d%%3A%s", ts, pass))
	// SĐT phone-first (do register.go cấp, tái dùng cho msg_previous_cp ở add-mail), GIỮ type=phone.
	newPhoneEnc := strings.ReplaceAll(url.QueryEscape(phone), "+", "%2B")
	body = strings.ReplaceAll(body, capPhone, newPhoneEnc)
	body = strings.ReplaceAll(body, capPhoneRaw, strings.TrimPrefix(phone, "+"))
	// AAC tươi (cùng bộ với add-mail/confirm) — bind account vào aac sống ngay từ create.
	if aacjid != "" {
		body = strings.ReplaceAll(body, capAACjid, aacjid)
	}
	if aaccs != "" {
		body = strings.ReplaceAll(body, capAACcs, aaccs)
	}
	if aacTs != "" {
		body = strings.ReplaceAll(body, capAACts, aacTs)
	}
	// Flow-session IDs tươi (cùng bộ với add-mail/confirm) — bind userid vào flow sống từ create.
	if regFlowID != "" {
		body = strings.ReplaceAll(body, capRegFlowID, regFlowID)
	}
	if headersFlowID != "" {
		body = strings.ReplaceAll(body, capHeadersFlowID, headersFlowID)
	}
	// E-RC: thay reg_context TĨNH (blob cũ của device capture) bằng reg_context TƯƠI
	// server cấp ở pre-steps (chain) → khớp device hiện tại, tránh replay.
	if regCtx != "" {
		body = reRegCtxIOS.ReplaceAllString(body, regCtx)
	}
	return body, nil
}

// reRegCtxIOS bắt blob reg_context (base64url AV... dài) trong response/template.
var reRegCtxIOS = regexp.MustCompile(`AV[A-Za-z0-9_-]{300,}`)

// extractRegContextIOS lấy reg_context dài nhất từ response 1 bước Bloks.
func extractRegContextIOS(resp string) string {
	best := ""
	for _, m := range reRegCtxIOS.FindAllString(resp, -1) {
		if len(m) > len(best) {
			best = m
		}
	}
	return best
}

// ─── response parsing ───
var (
	reUID     = regexp.MustCompile(`\b(615\d{10,12}|1000\d{8,12})\b`)
	reCrypted = regexp.MustCompile(`crypted_user_id[^A-Za-z0-9]{1,30}(AY[A-Za-z0-9_+/=\-]{20,})`)
)

func extractUID(raw string) string {
	m := reUID.FindStringSubmatch(deepUnescape(raw))
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func extractCryptedUID(raw string) string {
	m := reCrypted.FindStringSubmatch(raw)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

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

// isConfirmSuccess — confirm THẬT (port từ cmd/test_messios, đã validate bằng test thực).
func isConfirmSuccess(raw string) bool {
	u := deepUnescape(raw)
	low := strings.ToLower(u)
	if strings.Contains(low, "invalid_input") {
		return false
	}
	if strings.Contains(u, `"should_show_error":true`) {
		return false
	}
	if reEAA.MatchString(raw) {
		return true
	}
	if strings.Contains(u, `"is_confirmed":true`) {
		return true
	}
	if strings.Contains(u, "CAA_ACCOUNT_ACCESS_CONTEXT:aac") {
		return true
	}
	// 11KB checkpoint-phase response = confirm accepted (đã validate login được)
	if strings.Contains(low, "caa_core_data_encrypted") &&
		strings.Contains(u, "CAA_LOGIN_FALLBACK:fallback_triggered") {
		return true
	}
	if strings.Contains(low, "save-credentials") || strings.Contains(low, "save_credentials") ||
		strings.Contains(low, "nux_type") || strings.Contains(low, "caa_nux") {
		return true
	}
	return false
}

var reEAA = regexp.MustCompile(`EAA[A-Za-z0-9]{24,}`)

// genAACParts sinh 1 bộ AAC (Account Access Context) TƯƠI client-mint cho 1 session.
// Port từ sibling Android appmessv3 genAAC(): aaccs = 32 byte random base64url (token opaque,
// server KHÔNG validate chữ ký — chỉ cần nhất quán xuyên session). aacjid = uuid, ts = now (giây).
// Phải sinh 1 LẦN/account rồi tái dùng y hệt qua create→add-mail→confirm (giống capture GOOD).
func genAACParts() (aacjid, aaccs, aacTs string) {
	b := make([]byte, 32)
	if _, err := crand.Read(b); err != nil {
		for i := range b {
			b[i] = byte(rand.Intn(256))
		}
	}
	return uuid.New().String(), base64.RawURLEncoding.EncodeToString(b), strconv.FormatInt(time.Now().Unix(), 10)
}

// randPhoneVN sinh số điện thoại VN (cho phone-primary, hiện default email-primary).
func randVNPhone(r *rand.Rand) string {
	prefixes := []string{"91", "94", "96", "97", "98", "32", "33", "34", "35", "36", "37", "38", "39",
		"56", "58", "70", "76", "77", "78", "79", "81", "82", "83", "84", "85", "86", "89"}
	return "+84" + prefixes[r.Intn(len(prefixes))] + fmt.Sprintf("%07d", r.Intn(10000000))
}
