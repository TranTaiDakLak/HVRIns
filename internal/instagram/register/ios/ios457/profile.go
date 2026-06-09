// profile.go — iOS555 IOSProfile builder + UA builder + random helpers.
// Data tables (devices, builds) → devices.go.
package ios457

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"HVRIns/internal/instagram/fakeinfo"
)

// ─── App constants (capture FBIOS v562) ──────────────────────────────────────

const (
	// fbAppVersion / fbBuildNum — dùng cho pwdkey.go app_version param (cần cố định).
	fbAppVersion = "457.0.0.38.108"
	fbBuildNum   = "581425567"

	// oauthToken — Authorization: OAuth <app_id>|<client_token>.
	// app_id 6628568379 = FB iOS app; client_token là token public ship kèm app.
	oauthToken = "6628568379|c1e620fa708a1d5696fb991c1bde5662"

	// Bloks constants capture từ v563 (EnterCode563).
	docIDAction       = "37580109603817209302818798466"
	bloksVersioningID = "fd79983e06b4c7af80f4d12f5cd88aa4f1ba7930cf38e6282404df0a613da04f"
	stylesID          = "f655b55dc108bc537d0b95221b9d61ab"

	// createAccountAppID — Bloks app_id cho bước tạo account (single-shot).
	createAccountAppID = "com.bloks.www.bloks.caa.reg.create.account.async"
	// createAccountFriendlyName — fb_api_req_friendly_name tương ứng.
	createAccountFriendlyName = "FBBloksActionRootQuery-" + createAccountAppID

	// pwdPrefix — encrypted_password prefix. Version 0 = plaintext (FB mã hóa
	// server-side). Android s563 dùng #PWD_FB4A:0: chạy OK trên cùng CAA backend.
	// Nếu create.account reject password → đổi sang RSA #PWD_WILDE:2: (xem docs).
	pwdPrefix = "#PWD_FB4A:0:"
)

// ─── iOS profile ─────────────────────────────────────────────────────────────

// IOSProfile chứa toàn bộ dữ liệu định danh cho 1 phiên reg iOS.
type IOSProfile struct {
	Device         iPhoneDevice
	UserAgent      string
	DeviceID       string              // IDFV — UUID hoa, vd "9F1071D0-006A-46E4-8673-313A57BCD7A3"
	FamilyDeviceID string              // UUID hoa
	WaterfallID    string              // UUID thường
	MachineID      string              // X-FB-Integrity-Machine-Id — 24 ký tự base64url
	CloudTrustID   string              // X-Cloud-Trust-Token
	PtmUUID        string              // x-fb-ptm-uuid — 32 hex hoa
	Locale         string              // vd "en_US", "vi_VN"
	ConnType       string              // X-FB-Connection-Type: wifi | mobile.CTRadioAccessTechnology*
	Sim            fakeinfo.SimProfile // SIM carrier — HNI cho X-FB-SIM-HNI
}

// BuildProfileFromDevice dựng IOSProfile tái dùng DeviceID/FamilyDeviceID/MachineID
// từ pool. Chỉ WaterfallID, CloudTrustID, PtmUUID là random mới mỗi lần.
func BuildProfileFromDevice(locale, countryCode string, dp DeviceProfile) IOSProfile {
	if locale == "" {
		locale = "en_US"
	}
	dev := getIPhoneDevices()[randInt(len(getIPhoneDevices()))]
	ua := buildIOSUA(dev, locale)
	return IOSProfile{
		Device:         dev,
		UserAgent:      ua,
		DeviceID:       dp.DeviceID,
		FamilyDeviceID: dp.FamilyDeviceID,
		WaterfallID:    uuid.New().String(),
		MachineID:      dp.MachineID,
		CloudTrustID:   strings.ToUpper(uuid.New().String()) + strings.ToUpper(uuid.New().String()),
		PtmUUID:        randHexUpper(32),
		Locale:         locale,
		ConnType:       fakeinfo.RandomIOSConnType(),
		Sim:            fakeinfo.RandomSimProfile(countryCode),
	}
}

// BuildProfile dựng 1 IOSProfile mới với device random từ pool + ID mới.
func BuildProfile(locale, countryCode string) IOSProfile {
	if locale == "" {
		locale = "en_US"
	}
	dev := getIPhoneDevices()[randInt(len(getIPhoneDevices()))]
	ua := buildIOSUA(dev, locale)
	return IOSProfile{
		Device:         dev,
		UserAgent:      ua,
		DeviceID:       upperUUID(),
		FamilyDeviceID: upperUUID(),
		WaterfallID:    uuid.New().String(),
		MachineID:      randBase64URL(24),
		CloudTrustID:   strings.ToUpper(uuid.New().String()) + strings.ToUpper(uuid.New().String()),
		PtmUUID:        randHexUpper(32),
		Locale:         locale,
		ConnType:       fakeinfo.RandomIOSConnType(),
		Sim:            fakeinfo.RandomSimProfile(countryCode),
	}
}

// buildIOSUA ghép native FBIOS User-Agent từ device + locale.
// FBAV/FBBV/FBRV random từ pool builds thật (nguồn: user-agents.net).
func buildIOSUA(d iPhoneDevice, locale string) string {
	b := randFBBuild()
	return fmt.Sprintf(
		"Mozilla/5.0 (iPhone; CPU iPhone OS %s like Mac OS X) "+
			"AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/%s "+
			"[FBAN/FBIOS;FBAV/%s;FBBV/%s;FBDV/%s;FBMD/iPhone;FBSN/iOS;"+
			"FBSV/%s;FBSS/%s;FBID/phone;FBLC/%s;FBOP/5;FBRV/%s]",
		d.IOSUnder, d.MobileBld, b.FBAV, b.FBBV, d.FBDV,
		d.IOSDot, d.FBSS, locale, b.FBRV,
	)
}

// ─── Random helpers ──────────────────────────────────────────────────────────

// upperUUID sinh UUID dạng IDFV của iOS — chữ hoa, có dấu gạch.
func upperUUID() string {
	return strings.ToUpper(uuid.New().String())
}

// randInt trả số ngẫu nhiên [0, n) bằng crypto/rand.
func randInt(n int) int {
	if n <= 1 {
		return 0
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	v := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	if v < 0 {
		v = -v
	}
	return v % n
}

// randBase64URL sinh chuỗi n ký tự thuộc bảng base64url [A-Za-z0-9_-].
// Dùng cho X-FB-Integrity-Machine-Id (capture: "yi0Qalz2ZUaIuL6q2AF_D0ti", 24 ký tự).
func randBase64URL(n int) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(b)
}

// randHexUpper sinh n ký tự hex hoa. Dùng cho x-fb-ptm-uuid.
func randHexUpper(n int) string {
	const hexU = "0123456789ABCDEF"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = hexU[int(b[i])%16]
	}
	return string(b)
}
