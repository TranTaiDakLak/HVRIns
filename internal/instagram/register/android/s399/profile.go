// profile.go — S399 device profile + UA builder + cookie seed + shared pool.
package s399

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"

	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/instagram/fakeinfo/uabuilder"
	"HVRIns/internal/instagram/register/android"
)

// SharedPool — partitioned datr pool dùng chung cho tất cả s399 worker.
var SharedPool *android.PartitionedDatrPool

// S399Device — Samsung S23 device variant (s399 chỉ hỗ trợ pool S23 vì OriginalUA
// hardcode SM-S911B; user có thể mở rộng sau).
type S399Device struct {
	Model     string
	Brand     string
	OSVersion string
	BuildID   string // vd "AP3A.240905.015.A2"
	Width     int
	Height    int
	Density   string
}

var s399Devices = []S399Device{
	{Model: "SM-S911B", Brand: "samsung", OSVersion: "15", BuildID: "AP3A.240905.015.A2", Width: 1080, Height: 2340, Density: "3.0"},
	{Model: "SM-S911U", Brand: "samsung", OSVersion: "15", BuildID: "AP3A.240905.015.A2", Width: 1080, Height: 2340, Density: "3.0"},
	{Model: "SM-S916B", Brand: "samsung", OSVersion: "15", BuildID: "AP3A.240905.015.A2", Width: 1080, Height: 2340, Density: "3.0"},
	{Model: "SM-S918B", Brand: "samsung", OSVersion: "15", BuildID: "AP3A.240905.015.A2", Width: 1440, Height: 3088, Density: "3.0"},
}

// S399Profile — toàn bộ data cần cho 1 session reg s399.
type S399Profile struct {
	fakeinfo.FullRegProfile
	Device      S399Device
	S399UA      string
	ConnType    string
	DeviceGroup string
}

// BuildProfileForPlatform build profile mặc định (random device + UA generated theo locale/sim).
func BuildProfileForPlatform(countryCode string) S399Profile {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))

	base := fakeinfo.BuildFullRegProfile(countryCode)
	if base.Locale == "" {
		base.Locale = "en_US"
	}

	dev := s399Devices[r.Intn(len(s399Devices))]
	carrier := base.Sim.OperatorName
	if carrier == "" {
		carrier = "Viettel"
	}

	fbav, fbbv := s399FBAV, s399FBBV
	if vers, err := uabuilder.LoadAppVersionsForReg(); err == nil && len(vers) > 0 {
		av := vers[r.Intn(len(vers))]
		fbav, fbbv = av.Version, av.Build
	}
	ua := fmt.Sprintf(
		"Dalvik/2.1.0 (Linux; U; Android %s; %s Build/%s) [FBAN/FB4A;FBAV/%s;FBPN/com.facebook.katana;FBLC/%s;FBBV/%s;FBCR/%s;FBMF/%s;FBBD/%s;FBDV/%s;FBSV/%s;FBLC/%s;FBOP/1;FBCA/arm64-v8a:armeabi-v7a;]",
		dev.OSVersion, dev.Model, dev.BuildID,
		fbav, base.Locale, fbbv, carrier,
		dev.Brand, dev.Brand, dev.Model, dev.OSVersion, base.Locale,
	)

	base.UserAgent = ua
	base.DeviceID = uuid.New().String()
	base.FamilyDeviceID = uuid.New().String()
	base.WaterfallID = uuid.New().String()
	base.MachineID2 = randomAlphanumeric(r, 28)
	base.Device.Brand = dev.Brand
	base.Device.Model = dev.Model
	base.Device.OSVersion = dev.OSVersion
	base.Device.AndroidID = "android-" + randomHex(r, 16)

	connType := "WIFI"
	if r.Intn(3) == 0 {
		connType = "MOBILE.LTE"
	}

	return S399Profile{
		FullRegProfile: base,
		Device:         dev,
		S399UA:         ua,
		ConnType:       connType,
		DeviceGroup:    fmt.Sprintf("%d", 1000+r.Intn(9000)),
	}
}

func randomAlphanumeric(r *rand.Rand, n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}
	return string(b)
}

func randomHex(r *rand.Rand, n int) string {
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hex[r.Intn(16)]
	}
	return string(b)
}

// ─── Cookie seed model (port từ s559) ────────────────────────────────────────

type SeedMode int

const (
	SeedModeNone SeedMode = iota
	SeedModeDatrOnly
	SeedModeFullCookie
	SeedModeInitialAccount
)

type Seed struct {
	Raw          string
	Mode         SeedMode
	Datr         string
	CookieString string
	UID          string
	Password     string
	SourceLabel  string
}

func ParseSeed(raw string) Seed {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Seed{Mode: SeedModeNone, SourceLabel: "none"}
	}
	if strings.Contains(raw, "|") {
		parts := strings.Split(raw, "|")
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			s := Seed{
				Raw:         raw,
				Mode:        SeedModeInitialAccount,
				UID:         parts[0],
				Password:    parts[1],
				SourceLabel: "initial_account(uid=" + safeShort(parts[0], 8) + "...)",
			}
			if len(parts) >= 3 {
				s.CookieString = parts[2]
				s.Datr = seedExtractDatr(parts[2])
			}
			return s
		}
	}
	if strings.Contains(raw, "datr=") {
		datr := seedExtractDatr(raw)
		return Seed{
			Raw:          raw,
			Mode:         SeedModeFullCookie,
			CookieString: raw,
			Datr:         datr,
			SourceLabel:  "full_cookie(datr=" + safeShort(datr, 8) + "...)",
		}
	}
	return Seed{
		Raw:         raw,
		Mode:        SeedModeDatrOnly,
		Datr:        raw,
		SourceLabel: "datr_only(" + safeShort(raw, 8) + "...)",
	}
}

func seedExtractDatr(cookieStr string) string {
	for _, pair := range strings.Split(cookieStr, ";") {
		pair = strings.TrimSpace(pair)
		if strings.HasPrefix(pair, "datr=") {
			return strings.TrimPrefix(pair, "datr=")
		}
	}
	return ""
}

func safeShort(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
