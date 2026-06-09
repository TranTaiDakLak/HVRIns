// profile.go — S563V4S21 device profile + UA builder + cookie seed + shared pool.
// Device: Samsung Galaxy S21+ (SM-G996B), density 2.8125, 1080x2400.
// UA build cục bộ từ s21Devices (KHÔNG dùng global uabuilder filter SM-S*) để đảm bảo
// FBDV/SM-G996B luôn đúng.
package s563v6s21

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

// SharedPool là partitioned datr pool — mỗi slot goroutine có queue riêng.
var SharedPool *android.PartitionedDatrPool

// S560Profile holds all S563V4S21-specific data for one registration session.
type S560Profile struct {
	fakeinfo.FullRegProfile           // embed base profile
	Device                  S23Device // device variant (S23Device defined in body.go)
	S560UA                  string    // S563V4S21-specific User-Agent (FBAV 563, SM-G996B)
	ConnType                string    // WIFI / mobile.LTE
}

// BuildProfileForPlatform builds a complete S563V4S21 registration profile (default UA).
func BuildProfileForPlatform(platform, countryCode string) S560Profile {
	return BuildProfileForPlatformWithUA(platform, countryCode, false)
}

// BuildProfileForPlatformWithUA build profile có honor toggle UA addVirtualSpecs.
func BuildProfileForPlatformWithUA(platform, countryCode string, addVirtualSpecs bool) S560Profile {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))

	base := fakeinfo.BuildFullRegProfile(countryCode)

	locale := base.Locale
	if locale == "" {
		locale = "en_US"
	}

	carrier := base.Sim.OperatorName
	if carrier == "" {
		carrier = fakeinfo.RandomCarrier()
		if carrier == "" {
			carrier = "T-Mobile"
		}
	}

	device := s21Devices[r.Intn(len(s21Devices))]
	ua := buildS563V4S21UA(device, locale, carrier, addVirtualSpecs, r)

	base.UserAgent = ua
	base.DeviceGroup = "312" // captured RegS21 x-fb-device-group
	base.DeviceID = uuid.New().String()
	base.FamilyDeviceID = uuid.New().String()
	base.WaterfallID = uuid.New().String()
	base.MachineID2 = randomAlphanumeric(r, 28)
	base.Device.Brand = "samsung"
	base.Device.Model = device.Model
	base.Device.OSVersion = "15"
	base.Device.AndroidID = "android-" + randomHex(r, 16)

	connType := "WIFI"
	if r.Intn(3) == 0 {
		connType = "mobile.LTE"
	}

	return S560Profile{
		FullRegProfile: base,
		Device:         device,
		S560UA:         ua,
		ConnType:       connType,
	}
}

// buildS563V4S21UA — FB4A native UA cho SM-G996B (Galaxy S21+), version đọc từ versions_and_builds.txt.
func buildS563V4S21UA(device S23Device, locale, carrier string, addVirtualSpecs bool, r *rand.Rand) string {
	fbav, fbbv := "563.1.0.50.73", "986611012"
	if vers, err := uabuilder.LoadAppVersionsForReg(); err == nil && len(vers) > 0 {
		av := vers[r.Intn(len(vers))]
		fbav, fbbv = av.Version, av.Build
	}
	fbUG := fmt.Sprintf(
		"[FBAN/FB4A;FBAV/%s;FBBV/%s;FBDM={density=%s,width=%d,height=%d};"+
			"FBLC/%s;FBRV/0;FBCR/%s;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;"+
			"FBDV/%s;FBSV/15;FBOP/1;FBCA/arm64-v8a]",
		fbav, fbbv, device.Density, device.Width, device.Height, locale, carrier, device.Model,
	)
	if !addVirtualSpecs {
		return fbUG
	}
	return fmt.Sprintf(
		"Dalvik/2.1.0 (Linux; U; Android 15; %s Build/samsung-%s) %s",
		device.Model, device.Model, fbUG,
	)
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

// ─── Cookie seed model ────────────────────────────────────────────────────────

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
