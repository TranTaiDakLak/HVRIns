// overrides.go — Load user overrides từ Config/ folder, thay thế embedded data.
// Port C# config/: user edit file → app pick up mà không cần rebuild.
//
// Mỗi function đọc file text (mỗi dòng 1 entry), fallback về embedded khi file
// không tồn tại hoặc rỗng. Gọi từ app.go startup sau khi seed.
package fakeinfo

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	overrideMu sync.Mutex
	baseConfigDir = "Config"
)

// SetConfigBaseDir thiết lập thư mục Config gốc (để test hoặc custom).
func SetConfigBaseDir(dir string) {
	overrideMu.Lock()
	defer overrideMu.Unlock()
	if dir != "" {
		baseConfigDir = dir
	}
}

// ReloadOverrides đọc tất cả file override từ Config/ và áp dụng vào package state.
// Được app.go gọi 1 lần lúc startup sau khi SeedConfigDataIfMissing.
//
// Override áp dụng cho:
//   - Namereg/US: firstnames + lastnames (khi NameReg=US)
//   - Namereg/VN: vn_firstnames + vn_lastnames (khi NameReg=VN)
//   - SimNetwork: simnetworks.txt (60+ sim operator info)
//   - Locales: locales.txt
//
// Không override:
//   - DeviceInfo (devices phức tạp, không phải plain text)
//   - PhoneDatabase (phone codes — hiếm khi cần edit)
//   - Carriers (không trực tiếp expose)
func ReloadOverrides() {
	overrideMu.Lock()
	defer overrideMu.Unlock()

	// Namereg US
	loadLinesFileOverride(
		filepath.Join(baseConfigDir, "Namereg", "US", "firstname.txt"),
		&firstNames,
	)
	loadLinesFileOverride(
		filepath.Join(baseConfigDir, "Namereg", "US", "lastname.txt"),
		&lastNames,
	)
	// Namereg VN
	loadLinesFileOverride(
		filepath.Join(baseConfigDir, "Namereg", "VN", "firstname.txt"),
		&vnFirstNames,
	)
	loadLinesFileOverride(
		filepath.Join(baseConfigDir, "Namereg", "VN", "lastname.txt"),
		&vnLastNames,
	)

	// Locales
	loadLinesFileOverride(
		filepath.Join(baseConfigDir, "Locales", "locales.txt"),
		&localeList,
	)

	// Sim Networks — format "MCC|MNC|OperatorName|CountryCode"
	loadSimNetworkOverride(filepath.Join(baseConfigDir, "SimNetwork", "simnetworks.txt"))
}

// loadLinesFileOverride đọc file text + update pointer nếu file có nội dung hợp lệ.
// Không ghi đè nếu file rỗng/không tồn tại — giữ nguyên embedded.
func loadLinesFileOverride(path string, dest *[]string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := make([]string, 0, 256)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue // skip rỗng + comment
		}
		lines = append(lines, line)
	}
	if len(lines) > 0 {
		*dest = lines
	}
}

// loadSimNetworkOverride đọc simnetworks.txt format "MCC|MNC|OperatorName|CountryCode".
// Giống init() trong simnetwork.go nhưng đọc từ disk thay vì embed.
func loadSimNetworkOverride(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	parsed := make([]SimProfile, 0, 64)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) != 4 {
			continue
		}
		parsed = append(parsed, SimProfile{
			MCC:          parts[0],
			MNC:          parts[1],
			OperatorName: parts[2],
			CountryCode:  parts[3],
			HNI:          parts[0] + parts[1],
		})
	}
	if len(parsed) > 0 {
		simList = parsed
	}
}
