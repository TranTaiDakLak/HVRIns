package iosmess

import (
	"math/rand"
	"time"

	tls_client "github.com/bogdanfinn/tls-client"
)

// Exported API — cho verify package (internal/facebook/verify/ios/iosmess) tái dùng
// body builders + transport, không phải duplicate code.

// Friendly names cho verify.
const (
	FnAddMail     = fnAddMail
	FnConfirm     = fnConfirm
	FnBottomsheet = fnBottomshee
	FnChangeMail  = fnChangeMail
)

// AppToken — Messenger app OAuth token (cho verify headers).
const AppToken = appToken

// Caps cho verify (name/birthday giữ giống reg để khớp reg_info).
const (
	CapFirstName = capFirstName
	CapLastName  = capLastName
	CapBirthday  = capBirthday
)

// NewIOSClient — tls-client Safari iOS + proxy.
func NewIOSClient(proxy string, timeoutSec int) (tls_client.HttpClient, error) {
	return newClient(proxy, timeoutSec)
}

// SendStep — POST 1 bước Bloks (app token + headers Pando). Trả (status, body, err).
func SendStep(client tls_client.HttpClient, body, friendly, deviceID, ua string) (int, string, error) {
	return sendBloks(client, body, friendly, deviceID, ua)
}

// SendStepWithToken — giống SendStep nhưng dùng token override (EAAG thay app-token).
func SendStepWithToken(client tls_client.HttpClient, body, friendly, deviceID, ua, token string) (int, string, error) {
	return sendBloksWithToken(client, body, friendly, deviceID, ua, token)
}

// IsConfirmOK — confirm response = thành công?
func IsConfirmOK(raw string) bool { return isConfirmSuccess(raw) }

// ExtractCryptedUID — bóc crypted_user_id (cho verify fallback nếu reg không trả).
func ExtractCryptedUID(raw string) string { return extractCryptedUID(raw) }

// BuildAddMailBody — contactpoint_email.async (add email, gate bằng cryptedUID) → trigger OTP.
// aacjid/aaccs/aacTs PHẢI là đúng bộ AAC đã dùng lúc create (truyền từ RegResult.AAC*).
// passRaw + passTS PHẢI là mật khẩu thô và ts Unix lúc create — server validate encrypted_password
// trong reg_info (nếu dùng template placeholder sẽ mismatch → is_email_valid=false).
func BuildAddMailBody(device, family, machine, waterfall, uid, cryptedUID, emailLocal, emailDomain, phone, aacjid, aaccs, aacTs, regFlowID, headersFlowID, passRaw string, passTS int64) (string, error) {
	return buildStep("contactpoint_email", stepData{
		fp:        fingerprint{device: device, family: family, machine: machine, waterfall: waterfall},
		uid:       uid,
		cryptedUID: cryptedUID,
		phone:     phone, // → msg_previous_cp = SĐT create (ngữ cảnh đổi phone→email → trigger OTP)
		firstName: capFirstName, lastName: capLastName, birthday: capBirthday,
		emailLoc: emailLocal, emailDom: emailDomain,
		aacjid: aacjid, aaccs: aaccs, aacTs: aacTs,
		regFlowID: regFlowID, headersFlowID: headersFlowID,
		pass: passRaw, ts: passTS,
	})
}

// BuildScreenLoadBody — render bottomsheet / change_email (which = "bottomsheet"|"change_email").
func BuildScreenLoadBody(which, device, family, machine, waterfall, uid, cryptedUID, emailLocal, emailDomain, phone, aacjid, aaccs, aacTs, regFlowID, headersFlowID string) (string, error) {
	return buildStep(which, stepData{
		fp:        fingerprint{device: device, family: family, machine: machine, waterfall: waterfall},
		uid:       uid,
		cryptedUID: cryptedUID,
		phone:     phone, // change_email render: msg_previous_cp = SĐT
		emailLoc: emailLocal, emailDom: emailDomain,
		aacjid: aacjid, aaccs: aaccs, aacTs: aacTs,
		regFlowID: regFlowID, headersFlowID: headersFlowID,
	})
}

// LoginFull — login + trả token, cookie, crypted_user_id (từ login response).
// Dùng cho ver flow mới: login → EAAG + cryptedUID → add-mail/confirm không cần reg session.
func LoginFull(proxy, uid, password, deviceID, familyID, datr, waterfall, ua string) (token, cookie, cryptedUID string, err error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	client, cerr := newClient(proxy, 60)
	if cerr != nil {
		return "", "", "", cerr
	}
	fp := fingerprint{device: deviceID, family: familyID, machine: datr, waterfall: waterfall}
	if ua == "" {
		ua = buildUA(r, "ver")
	}
	body, berr := buildLoginBody(fp, uid, password, time.Now().Unix())
	if berr != nil {
		return "", "", "", berr
	}
	_, raw, herr := sendBloks(client, body,
		"MSGBloksActionRootQuery-com.bloks.www.bloks.caa.login.async.send_login_request",
		deviceID, ua)
	if herr != nil {
		return "", "", "", herr
	}
	token, cookie, cryptedUID = extractLoginFull(raw)
	return token, cookie, cryptedUID, nil
}

// LoginRaw — login + trả raw response (debug/inspect).
func LoginRaw(proxy, uid, password, deviceID, familyID, datr, waterfall, ua string) (token, cookie, raw string, err error) {
	tok, ck, _, ferr := LoginFull(proxy, uid, password, deviceID, familyID, datr, waterfall, ua)
	return tok, ck, "", ferr
}

// GenAACParts — sinh bộ AAC mới (jid, cs, ts) dùng cho ver flow không có reg session.
func GenAACParts() (jid, cs, ts string) {
	return genAACParts()
}

// BuildConfirmBodyVerify — confirmation.async (uid + cryptedUID + email + OTP code).
func BuildConfirmBodyVerify(device, family, machine, waterfall, uid, cryptedUID, emailLocal, emailDomain, code, aacjid, aaccs, aacTs, regFlowID, headersFlowID string) (string, error) {
	return buildStep("confirm", stepData{
		fp:        fingerprint{device: device, family: family, machine: machine, waterfall: waterfall},
		uid:       uid,
		cryptedUID: cryptedUID,
		emailLoc: emailLocal, emailDom: emailDomain, code: code,
		firstName: capFirstName, lastName: capLastName, birthday: capBirthday,
		aacjid: aacjid, aaccs: aaccs, aacTs: aacTs,
		regFlowID: regFlowID, headersFlowID: headersFlowID,
	})
}
