// profile.go — device fingerprint + constants cho IG iOS reg.
package igcore

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const (
	igAppID        = "124024574287414"
	bloksVersionID = "bbabb80f3de1e25c0c5c0bbfd6cae893124649276a36ead33328a4ea03a34b75"
	// client_doc_id cho IGBloksAppRootQuery (dùng cho hầu hết step trừ contactpoint).
	docIDApp = "38523300859713187485104132294"
	// User-Agent IG iOS — iPhone9,1 / iOS 15.8.4 (khớp capture).
	igUserAgent = "Instagram 410.1.0.36.70 (iPhone9,1; iOS 15_8_4; vi_VN; vi; scale=2.00; 750x1334; 849447290) AppleWebKit/420+"

	// igAndroidAppID và UA dự phòng — Android fingerprint cho i.instagram.com nếu cần sau này.
	igAndroidAppID = "567067343352427"
	igAndroidUA    = "Instagram 309.0.0.40.113 Android (30/11; 420dpi; 1080x2201; Google; Pixel 5; redfin; google; en_US; 536920689)"
)

// igProfile — định danh thiết bị cho 1 phiên reg.
type igProfile struct {
	DeviceID       string // IDFV uppercase UUID  → X-IG-Device-ID
	FamilyDeviceID string // uppercase UUID        → X-IG-Family-Device-ID
	WaterfallID    string // 32-hex (no dash)       → waterfall_id
	MachineID      string // X-MID (server cấp qua qe/sync); ban đầu rỗng
	RegMachineID   string // machine_id trong reg_info — 24-char base64url
	CloudTrustID   string // X-Cloud-Trust-Token: 2 UUID upper nối liền
	PigeonSID      string // X-Pigeon-Session-Id
	ConnUUID       string // x-fb-conn-uuid-client: 32-hex
	RegFlowID      string // registration_flow_id (UUID)
	UserAgent      string
	Locale         string // vi_VN
}

func newProfile() *igProfile {
	return &igProfile{
		DeviceID:       upperUUID(),
		FamilyDeviceID: upperUUID(),
		WaterfallID:    hex32(),
		RegMachineID:   randBase64URL(24),
		CloudTrustID:   strings.ToUpper(uuid.New().String()) + strings.ToUpper(uuid.New().String()),
		PigeonSID:      "UFS-" + strings.ToUpper(uuid.New().String()) + "-1",
		ConnUUID:       hex32(),
		RegFlowID:      uuid.New().String(),
		UserAgent:      igUserAgent,
		Locale:         "vi_VN",
	}
}

func upperUUID() string { return strings.ToUpper(uuid.New().String()) }

func hex32() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func randBase64URL(n int) string {
	const al = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-"
	b := make([]byte, n)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = al[int(b[i])%len(al)]
	}
	return string(b)
}

func newAAC() (jid, ccs string, ts int64) {
	return uuid.New().String(), randBase64URL(43), nowUnix()
}
