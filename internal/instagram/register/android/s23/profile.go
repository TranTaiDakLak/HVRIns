// profile.go — S23 device profile + UA builder + Cookie Initial seed + shared pool.
//
// File này gồm:
//   - S23Profile struct + BuildS23Profile (UA build từ device pool + SIM + locale)
//   - connUUID / randomAlphanumeric / randomHex helpers
//   - SeedMode + Seed + ParseSeed (3 modes: datr/full_cookie/initial_account)
//   - SharedPool (partitioned datr pool — reuse android.PartitionedDatrPool)
package s23

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

// ─── Shared cookie pool (partitioned datr pool) ──────────────────────────────

// SharedPool là partitioned datr pool — mỗi slot goroutine có queue riêng.
var SharedPool *android.PartitionedDatrPool

// S23Profile holds all S23-specific data for one registration session
type S23Profile struct {
	fakeinfo.FullRegProfile           // embed base profile
	Device                  S23Device // S23 device variant
	S23UA                   string    // S23-specific User-Agent
	ConnType                string    // WIFI / mobile.LTE
	USDID                   string    // x-meta-usdid (generated per session)
}

// BuildS23Profile creates a complete S23 registration profile.
// Deprecated shim — prefer BuildProfileForPlatform. Giữ cho compat test.
func BuildS23Profile(countryCode string) S23Profile {
	return BuildProfileForPlatform("s23", countryCode)
}

// BuildProfileForPlatform build profile theo platform (s22/s23/s24/s25/s26)
// với UA options mặc định (không Dalvik prefix).
// Wrapper backward compat — dùng BuildProfileForPlatformWithUA cho options chi tiết.
func BuildProfileForPlatform(platform, countryCode string) S23Profile {
	return BuildProfileForPlatformWithUA(platform, countryCode, false)
}

// BuildProfileForPlatformWithUA build profile có honor toggle UA:
//   - addVirtualSpecs: Dalvik/2.1.0 prefix (xem docs/UA_BUILDER_REFACTOR_PLAN.md §A.1).
//
// UA build qua uabuilder.AndroidUABuilder (single source of truth).
func BuildProfileForPlatformWithUA(platform, countryCode string, addVirtualSpecs bool) S23Profile {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))

	// Base profile (name, birthday, gender, sim, etc.)
	base := fakeinfo.BuildFullRegProfile(countryCode)

	locale := base.Locale
	if locale == "" {
		locale = "en_US"
	}

	// Build UA + pick device qua central uabuilder.
	// SimBrand từ SIM thật (nếu có) → carrier match SIM. Empty → uabuilder random từ carriers.txt.
	simBrand := base.Sim.OperatorName
	uaRes, uaErr := (&uabuilder.AndroidUABuilder{}).Build(uabuilder.UAOptions{
		PoolKind: "reg",
		Platform:        platform,
		Locale:          locale,
		SimBrand:        simBrand,
		AddVirtualSpecs: addVirtualSpecs,
	})
	if uaErr != nil {
		// Fallback: shouldn't happen nếu Config/ files OK — giữ best-effort.
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

	// S23Device adapter cho body builder cần Width/Height/Density/FBSS.
	// FBSS heuristic: Ultra (1440 width) = "4", còn lại "3" — match s23_devices.txt convention.
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

	// Override base profile fields
	base.UserAgent = uaRes.UserAgent
	base.DeviceGroup = fmt.Sprintf("%d", 1000+r.Intn(9000))
	base.DeviceID = uuid.New().String()
	base.FamilyDeviceID = uuid.New().String()
	base.WaterfallID = uuid.New().String()
	base.MachineID2 = randomAlphanumeric(r, 28)
	base.Device.Brand = uaRes.Brand
	base.Device.Model = uaRes.Model
	base.Device.OSVersion = uaRes.OSVersion
	base.Device.AndroidID = "android-" + randomHex(r, 16)

	// Connection type
	connType := "WIFI"
	if r.Intn(3) == 0 {
		connType = "mobile.LTE"
	}

	return S23Profile{
		FullRegProfile: base,
		Device:         device,
		S23UA:          uaRes.UserAgent,
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

// connUUID generates random base64 UUID for x-fb-conn-uuid-client (changes per request)
func connUUID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

// ─── Cookie Initial seed model (từ seed.go cũ) ───────────────────────────────
//
// Three modes matching C# source_type:
//
//	SeedModeDatrOnly       (source_type=1) — raw datr value → inject as datr cookie/machine_id
//	SeedModeFullCookie     (source_type=3) — "datr=x;sb=y;..." → seed all cookies
//	SeedModeInitialAccount (source_type=3) — "uid|password|cookie|..." → login+logout warm

// SeedMode identifies how a cookie initial entry should be used
type SeedMode int

const (
	SeedModeNone           SeedMode = iota // no seed provided
	SeedModeDatrOnly                       // raw datr value (e.g. "BYB0aXxxx")
	SeedModeFullCookie                     // cookie string with "datr=..." (e.g. "datr=xxx;sb=yyy")
	SeedModeInitialAccount                 // "uid|password|cookie|token" — login initial flow
)

// Seed is a parsed cookie initial entry
type Seed struct {
	Raw          string   // original input string
	Mode         SeedMode // parsed mode
	Datr         string   // extracted datr value (available in all non-None modes)
	CookieString string   // full cookie string (FullCookie and InitialAccount modes)
	UID          string   // account UID (InitialAccount mode only)
	Password     string   // account password (InitialAccount mode only)
	SourceLabel  string   // human-readable label for logging
}

// ParseSeed analyzes a raw cookie initial string and returns a structured Seed.
// Parse logic mirrors C# GetPerfectMachineId return values:
//
//	source_type=1 → datr value only (no pipes, no equals)
//	source_type=3 → full line "uid|pass|cookie|token|..."
func ParseSeed(raw string) Seed {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Seed{Mode: SeedModeNone, SourceLabel: "none"}
	}

	// Pipe-delimited: "uid|password|cookie_string|token|..."
	// C# source_type=3 with full initial account line
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

	// Full cookie string: "datr=xxx;sb=yyy;..."
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

	// Simple datr value (no pipes, no equals)
	return Seed{
		Raw:         raw,
		Mode:        SeedModeDatrOnly,
		Datr:        raw,
		SourceLabel: "datr_only(" + safeShort(raw, 8) + "...)",
	}
}

// seedExtractDatr pulls datr value from "datr=VALUE;..." cookie string
func seedExtractDatr(cookieStr string) string {
	for _, pair := range strings.Split(cookieStr, ";") {
		pair = strings.TrimSpace(pair)
		if strings.HasPrefix(pair, "datr=") {
			return strings.TrimPrefix(pair, "datr=")
		}
	}
	return ""
}

// safeShort returns up to n chars from s (safe for any length including 0)
func safeShort(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
