// login.go — Messenger Lite iOS LOGIN (send_login_request) → access_token + session_cookies.
//
// Capture: E:\WEMAKE\DocWeMake\FlowRegFB_IOS\Login_IOSMes (1 req/resp).
// friendly_name MSGBloksActionRootQuery-com.bloks.www.bloks.caa.login.async.send_login_request,
// doc_id/bloks/app-token GIỐNG reg. contact_point = UID, password #PWD_ENC:0:ts:plain,
// device/family/machine = đúng cap reg (cùng device session). Response lồng sâu (escaped nhiều
// lớp) chứa "access_token":"EAA..." + "session_cookies":[{name,value}...].
//
// Dùng: VER iOS Mess sau khi confirm + check LIVE → login lấy token+cookie thật cho account live
// (token/cookie reg nằm trong blob mã hoá caa_core_data_encrypted nên không dùng được).
package iosmess

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Hằng số literal CỦA capture login (substitute sang giá trị account).
const (
	capLoginUID = "61590833603441"                              // contact_point trong login.txt
	capLoginPwd = "%23PWD_ENC%3A0%3A1780804139%3AQ6FJEBqQi9"    // password literal (URL-encoded)
)

var (
	reLoginAccessToken  = regexp.MustCompile(`access_token":"(EAA[A-Za-z0-9]+)`)
	reLoginCryptedUID   = regexp.MustCompile(`"crypted_user_id":"([A-Za-z0-9_\-]{20,})`)
	// session_cookies pairs (sau khi unescape backslash): "name":"X","value":"Y"
	reLoginCookiePair = regexp.MustCompile(`"name":"([^"]+)","value":"([^"]*)"`)
)

// buildLoginBody dựng body send_login_request từ template login.txt + substitute account.
func buildLoginBody(fp fingerprint, uid, password string, ts int64) (string, error) {
	body, err := loadTemplate("login")
	if err != nil {
		return "", err
	}
	body = strings.ReplaceAll(body, capDeviceID, fp.device)
	body = strings.ReplaceAll(body, capFamilyID, fp.family)
	body = strings.ReplaceAll(body, capMachineID, fp.machine)
	if fp.waterfall != "" {
		body = strings.ReplaceAll(body, capWaterfall, fp.waterfall)
	}
	body = strings.ReplaceAll(body, capEventReq, uuid.New().String())
	body = strings.ReplaceAll(body, capLoginUID, uid)
	if password != "" && ts > 0 {
		body = strings.ReplaceAll(body, capLoginPwd,
			fmt.Sprintf("%%23PWD_ENC%%3A0%%3A%d%%3A%s", ts, password))
	}
	return body, nil
}

// extractLoginSession bóc access_token + cookie + crypted_user_id từ response login (unescape trước).
func extractLoginSession(resp string) (token, cookie string) {
	token, cookie, _ = extractLoginFull(resp)
	return token, cookie
}

// extractLoginFull bóc token + cookie + crypted_user_id từ login response.
func extractLoginFull(resp string) (token, cookie, cryptedUID string) {
	clean := strings.ReplaceAll(resp, "\\", "")
	if m := reLoginAccessToken.FindStringSubmatch(clean); len(m) > 1 {
		token = m[1]
	}
	if m := reLoginCryptedUID.FindStringSubmatch(clean); len(m) > 1 {
		cryptedUID = m[1]
	}
	// session_cookies → ghép "name=value;" (giới hạn vùng sau "session_cookies" để tránh nhiễu).
	if idx := strings.Index(clean, "session_cookies"); idx >= 0 {
		region := clean[idx:]
		if end := strings.Index(region, "]"); end > 0 {
			region = region[:end]
		}
		var sb strings.Builder
		for _, pr := range reLoginCookiePair.FindAllStringSubmatch(region, -1) {
			if pr[1] == "" || pr[2] == "" {
				continue
			}
			sb.WriteString(pr[1])
			sb.WriteString("=")
			sb.WriteString(pr[2])
			sb.WriteString(";")
		}
		cookie = sb.String()
	}
	return token, cookie, cryptedUID
}

// LoginAndGetSession login MessengerLite iOS (contact_point=uid + password) → token + cookie.
// Gọi từ VER iOS Mess cho account đã check LIVE. ua rỗng → tự build.
func LoginAndGetSession(proxy, uid, password, deviceID, familyID, datr, waterfall, ua string) (token, cookie string, err error) {
	if strings.TrimSpace(uid) == "" || strings.TrimSpace(password) == "" {
		return "", "", fmt.Errorf("login: thiếu uid/password")
	}
	client, err := newClient(proxy, 60)
	if err != nil {
		return "", "", err
	}
	fp := fingerprint{device: deviceID, family: familyID, machine: datr, waterfall: waterfall}
	if ua == "" {
		ua = buildUA(rand.New(rand.NewSource(time.Now().UnixNano())), "ver")
	}
	body, err := buildLoginBody(fp, uid, password, time.Now().Unix())
	if err != nil {
		return "", "", err
	}
	_, resp, err := sendBloks(client, body,
		"MSGBloksActionRootQuery-com.bloks.www.bloks.caa.login.async.send_login_request",
		deviceID, ua)
	if err != nil {
		return "", "", fmt.Errorf("login HTTP: %w", err)
	}
	token, cookie = extractLoginSession(resp)
	if token == "" {
		return "", cookie, fmt.Errorf("login: không bóc được access_token (resp %dB)", len(resp))
	}
	return token, cookie, nil
}
