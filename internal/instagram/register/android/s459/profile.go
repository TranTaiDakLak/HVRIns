// profile.go — S459 device profile + UA builder + cookie seed + shared pool.
package s459

import (
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

// S560Profile holds all S560-specific data for one registration session.
type S560Profile struct {
	fakeinfo.FullRegProfile           // embed base profile
	Device                  S23Device // device variant (S23Device defined in body.go)
	S560UA                  string    // S560-specific User-Agent (FBAV 560)
	ConnType                string    // WIFI / mobile.LTE
}

// BuildProfileForPlatform builds a complete S560 registration profile (default UA).
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

	simBrand := base.Sim.OperatorName
	uaRes, uaErr := (&uabuilder.AndroidUABuilder{}).Build(uabuilder.UAOptions{
		PoolKind: "reg",
		Platform:        platform,
		Locale:          locale,
		SimBrand:        simBrand,
		AddVirtualSpecs: addVirtualSpecs,
	})
	if uaErr != nil {
		uaRes = uabuilder.UABuildResult{
			UserAgent: "[FBAN/FB4A;]",
			Brand:     "samsung",
			Model:     "SM-S911B",
			OSVersion: "15",
			Density:   "3.0",
			Width:     1080,
			Height:    2340,
		}
	}

	fbss := "3"
	if uaRes.Width >= 1440 {
		fbss = "4"
	}
	device := S23Device{
		Model:   uaRes.Model,
		Name:    uaRes.Brand + " " + uaRes.Model,
		Width:   uaRes.Width,
		Height:  uaRes.Height,
		Density: uaRes.Density,
		FBSS:    fbss,
	}

	base.UserAgent = uaRes.UserAgent
	base.DeviceGroup = "2610"
	base.DeviceID = uuid.New().String()
	base.FamilyDeviceID = uuid.New().String()
	base.WaterfallID = uuid.New().String()
	base.MachineID2 = randomAlphanumeric(r, 28)
	base.Device.Brand = uaRes.Brand
	base.Device.Model = uaRes.Model
	base.Device.OSVersion = uaRes.OSVersion
	base.Device.AndroidID = "android-" + randomHex(r, 16)

	connType := "WIFI"
	if r.Intn(3) == 0 {
		connType = "mobile.LTE"
	}

	return S560Profile{
		FullRegProfile: base,
		Device:         device,
		S560UA:         uaRes.UserAgent,
		ConnType:       connType,
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
