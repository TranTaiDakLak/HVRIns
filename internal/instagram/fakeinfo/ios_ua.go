// ios_ua.go — iPhone iOS Facebook (FBiOS) User-Agent
// C#: ConfigFileUserAgentBuilder(PathSingleton.IosApiUgFile)
// Đọc random 1 dòng từ file iOS_UG.txt — KHÔNG generate bằng code
package fakeinfo

import (
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// IPhoneProfile chứa thông tin từ FBiOS UA
type IPhoneProfile struct {
	UserAgent         string
	IOSVersionUA      string // "17_0"
	IOSVersionDisplay string // "17.0"
	Model             string // "iPhone15,2"
}

// iosUAPool — UA pool loaded from file (lazy init, thread-safe)
var (
	iosUAPool     []string
	iosUAPoolOnce sync.Once
)

// loadIOSUAPool đọc file iOS_UG.txt
// C#: ConfigFileUserAgentBuilder reads from IosApiUgFile
func loadIOSUAPool() {
	paths := []string{
		"Config/Settings/useragent/iOS_UG.txt",
		"config/useragent/iOS_UG.txt",
		"iOS_UG.txt",
	}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && strings.Contains(line, "FBAN/") {
				iosUAPool = append(iosUAPool, line)
			}
		}
		if len(iosUAPool) > 0 {
			return
		}
	}
}

// RandomIPhoneProfile — random UA từ file iOS_UG.txt (giống C#)
// Fallback: generate nếu file không tìm thấy
func RandomIPhoneProfile() IPhoneProfile {
	iosUAPoolOnce.Do(loadIOSUAPool)

	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))

	if len(iosUAPool) > 0 {
		ua := iosUAPool[r.Intn(len(iosUAPool))]
		return parseIPhoneProfile(ua)
	}

	// Fallback: generate (khi file không có)
	return generateFallbackProfile(r)
}

// parseIPhoneProfile extract thông tin từ UA string
func parseIPhoneProfile(ua string) IPhoneProfile {
	p := IPhoneProfile{UserAgent: ua}

	// Extract iOS version: "CPU iPhone OS 16_1 like"
	if m := regexp.MustCompile(`iPhone OS (\d+_\d+)`).FindStringSubmatch(ua); len(m) > 1 {
		p.IOSVersionUA = m[1]
		p.IOSVersionDisplay = strings.ReplaceAll(m[1], "_", ".")
	}

	// Extract device model: "FBDV/iPhone15,3"
	if m := regexp.MustCompile(`FBDV/([^;]+)`).FindStringSubmatch(ua); len(m) > 1 {
		p.Model = m[1]
	}

	return p
}

// generateFallbackProfile — fallback khi không có file UA
func generateFallbackProfile(r *rand.Rand) IPhoneProfile {
	type iosEntry struct {
		underscored, dotted, build string
	}
	entries := []iosEntry{
		{"15_7", "15.7", "19H349"}, {"16_0", "16.0", "20A362"},
		{"16_1", "16.1", "20B82"}, {"16_2", "16.2", "20C65"},
		{"17_0", "17.0", "21A329"}, {"17_1", "17.1", "21B74"},
	}
	models := []string{"iPhone13,2", "iPhone14,2", "iPhone14,5", "iPhone15,2", "iPhone15,3", "iPhone16,1"}

	ios := entries[r.Intn(len(entries))]
	model := models[r.Intn(len(models))]
	fbav := "446.0.0.58.33"
	fbbv := 500000000 + r.Intn(500000000)
	fbrv := 500000000 + r.Intn(500000000)
	fbss := "3"

	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS " + ios.underscored + " like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/" + ios.build +
		" [FBAN/FBIOS;FBAV/" + fbav + ";FBBV/" + itoa(fbbv) + ";FBDV/" + model + ";FBMD/iPhone;FBSN/iOS;FBSV/" + ios.dotted + ";FBSS/" + fbss + ";FBID/phone;FBLC/en_US;FBOP/5;FBRV/" + itoa(fbrv) + ";IABMV/1]"

	return IPhoneProfile{
		UserAgent:         ua,
		IOSVersionUA:      ios.underscored,
		IOSVersionDisplay: ios.dotted,
		Model:             model,
	}
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
