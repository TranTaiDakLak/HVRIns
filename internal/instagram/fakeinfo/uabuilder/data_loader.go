// data_loader.go — đọc các file dữ liệu từ Config/ với cache + reload theo mtime.
//
// Mapping file ↔ data type (tên file khớp với thư mục Config/DeviceInfo/ thực tế):
//
//	Config/DeviceInfo/densitis.txt          → []string ["3.0", "3.5", "4.0"]
//	Config/DeviceInfo/device_cores.txt      → []string ["arm64-v8a", "armeabi-v7a:armeabi"]
//	Config/DeviceInfo/devices_versions.txt  → []string ["9", "10", "11", ...]
//	Config/DeviceInfo/screen_resolution.txt → []ScreenRes [{1080,2340}, ...]
//	Config/DeviceInfo/carriers.txt          → []string ["AT&T", "T-Mobile", ...]
//	Config/DeviceInfo/chrome_versions.txt   → []string ["146.0.7680", ...]
//	Config/DeviceInfo/<platform>_devices.txt → []DeviceSpec (per-platform nếu có)
//	Config/DeviceInfo/devices.txt            → []DeviceSpec (chung, fallback)
//	Config/Fbapp/versions_and_builds_<platform>.txt → []AppVersion
//	Config/Fbapp/versions_and_builds.txt    → []AppVersion (generic, fallback)
//
// Cache strategy: load lần đầu khi cần. Reload khi file mtime thay đổi (poll on-demand).
// User edit txt qua Notepad → app pickup ngay ở call kế tiếp.
package uabuilder

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// ConfigBaseDir là root của Config/ folder. Override bằng SetConfigBaseDir() nếu test
// hoặc app chạy với CWD khác.
var (
	configBaseMu  sync.RWMutex
	configBaseDir = "Config"
)

// SetConfigBaseDir override base dir (chủ yếu cho test).
func SetConfigBaseDir(dir string) {
	configBaseMu.Lock()
	defer configBaseMu.Unlock()
	configBaseDir = dir
	// Invalidate cache toàn bộ vì path đã đổi.
	listCacheMu.Lock()
	listCache = map[string]*cachedList{}
	listCacheMu.Unlock()
	deviceCacheMu.Lock()
	deviceCache = map[string]*cachedDevices{}
	deviceCacheMu.Unlock()
	appCacheMu.Lock()
	appCache = map[string]*cachedApps{}
	appCacheMu.Unlock()
}

// GetConfigBaseDir trả về base dir hiện tại.
func GetConfigBaseDir() string {
	configBaseMu.RLock()
	defer configBaseMu.RUnlock()
	return configBaseDir
}

// configPath ghép tên file thành path tuyệt đối.
func configPath(parts ...string) string {
	all := append([]string{GetConfigBaseDir()}, parts...)
	return filepath.Join(all...)
}

// ─── Cached []string lists (densities, cores, os, carriers, buildnums, chrome) ─

type cachedList struct {
	mtime  int64
	values []string
}

var (
	listCacheMu sync.Mutex
	listCache   = map[string]*cachedList{}
)

func loadStringList(relPath string) ([]string, error) {
	full := configPath(relPath)
	st, err := os.Stat(full)
	if err != nil {
		return nil, fmt.Errorf("uabuilder: stat %s: %w", full, err)
	}
	mtime := st.ModTime().UnixNano()

	listCacheMu.Lock()
	defer listCacheMu.Unlock()
	if c, ok := listCache[full]; ok && c.mtime == mtime {
		return c.values, nil
	}

	f, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("uabuilder: open %s: %w", full, err)
	}
	defer f.Close()

	out := []string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("uabuilder: read %s: %w", full, err)
	}
	listCache[full] = &cachedList{mtime: mtime, values: out}
	return out, nil
}

// ─── Devices ────────────────────────────────────────────────────────────────

// DeviceSpec parse từ DeviceInfo/*.txt. Format có 2 dạng:
//
//	Generic (3-5 fields):  Manufacturer:Brand:Model[:Width:Height]
//	Per-platform (7 fields): Manufacturer:Brand:Model:Width:Height:Density:FBSS
//
// Khi field thiếu, các field dimension trả 0 → builder tự pick từ
// screen_resolution.txt + densities.txt random.
type DeviceSpec struct {
	Manufacturer string
	Brand        string
	Model        string
	Width        int // 0 = unknown, builder fallback random
	Height       int // 0 = unknown
	Density      string
	FBSS         string
}

type cachedDevices struct {
	mtime int64
	specs []DeviceSpec
}

var (
	deviceCacheMu sync.Mutex
	deviceCache   = map[string]*cachedDevices{}
)

// LoadDevicesForPlatform đọc thiết bị từ Config/DeviceInfo/:
//  1. DeviceInfo/<platform>_devices.txt  (per-platform nếu có)
//  2. DeviceInfo/devices.txt             (chung — fallback)
//
// Không còn dùng MobileDevices/ — file thực tế nằm trong DeviceInfo/.
func LoadDevicesForPlatform(platform string) ([]DeviceSpec, error) {
	candidates := []string{}
	if platform != "" {
		candidates = append(candidates, filepath.Join("DeviceInfo", platform+"_devices.txt"))
	}
	candidates = append(candidates, filepath.Join("DeviceInfo", "devices.txt"))

	var lastErr error
	for _, rel := range candidates {
		full := configPath(rel)
		st, err := os.Stat(full)
		if err != nil {
			lastErr = err
			continue
		}
		specs, err := loadDevicesFile(full, st.ModTime().UnixNano())
		if err != nil {
			lastErr = err
			continue
		}
		if len(specs) > 0 {
			return specs, nil
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("uabuilder: no device pool for platform=%q: %w", platform, lastErr)
	}
	return nil, fmt.Errorf("uabuilder: no device pool for platform=%q", platform)
}

func loadDevicesFile(full string, mtime int64) ([]DeviceSpec, error) {
	deviceCacheMu.Lock()
	defer deviceCacheMu.Unlock()
	if c, ok := deviceCache[full]; ok && c.mtime == mtime {
		return c.specs, nil
	}

	f, err := os.Open(full)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	specs := []DeviceSpec{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue // line không hợp lệ, skip
		}
		ds := DeviceSpec{
			Manufacturer: parts[0],
			Brand:        parts[1],
			Model:        parts[2],
		}
		if len(parts) >= 5 {
			ds.Width, _ = strconv.Atoi(parts[3])
			ds.Height, _ = strconv.Atoi(parts[4])
		}
		if len(parts) >= 6 {
			ds.Density = parts[5]
		}
		if len(parts) >= 7 {
			ds.FBSS = parts[6]
		}
		specs = append(specs, ds)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	deviceCache[full] = &cachedDevices{mtime: mtime, specs: specs}
	return specs, nil
}

// ─── App versions (Fbapp/versions_and_builds_*) ────────────────────────────

// AppVersion = 1 dòng "FBAV|FBBV" của file versions_and_builds.
type AppVersion struct {
	Version string // FBAV
	Build   string // FBBV
}

type cachedApps struct {
	mtime    int64
	versions []AppVersion
}

var (
	appCacheMu sync.Mutex
	appCache   = map[string]*cachedApps{}
)

// LoadAppVersionsForPlatform đọc pool FBAV|FBBV CHUNG cho mọi Android FB4A platform.
// Param `platform` giữ cho backward-compat — không dùng để chọn file.
//
// Thứ tự ưu tiên:
//  1. Config/DeviceInfo/versions_and_builds.txt — path khuyến nghị (2026-05-26)
//  2. Config/Fbapp/versions_and_builds.txt      — legacy fallback
func LoadAppVersionsForPlatform(platform string) ([]AppVersion, error) {
	_ = platform
	return loadAppVersionsCandidates([]string{
		filepath.Join("DeviceInfo", "versions_and_builds.txt"),
		filepath.Join("Fbapp", "versions_and_builds.txt"),
	})
}

// LoadAppVersionsForReg — pool REG riêng (versions_and_builds_reg.txt).
// CHỈ tìm ở Config/DeviceInfo/ → fallback pool CHUNG nếu rỗng/không tồn tại.
func LoadAppVersionsForReg() ([]AppVersion, error) {
	vers, err := loadAppVersionsCandidates([]string{
		filepath.Join("DeviceInfo", "versions_and_builds_reg.txt"),
	})
	if err == nil && len(vers) > 0 {
		return vers, nil
	}
	return LoadAppVersionsForPlatform("")
}

// LoadAppVersionsForVer — pool VER riêng (versions_and_builds_ver.txt).
// CHỈ tìm ở Config/DeviceInfo/ → fallback pool CHUNG nếu rỗng/không tồn tại.
func LoadAppVersionsForVer() ([]AppVersion, error) {
	vers, err := loadAppVersionsCandidates([]string{
		filepath.Join("DeviceInfo", "versions_and_builds_ver.txt"),
	})
	if err == nil && len(vers) > 0 {
		return vers, nil
	}
	return LoadAppVersionsForPlatform("")
}

func loadAppVersionsCandidates(candidates []string) ([]AppVersion, error) {
	var lastErr error
	for _, rel := range candidates {
		full := configPath(rel)
		st, err := os.Stat(full)
		if err != nil {
			lastErr = err
			continue
		}
		vers, err := loadAppVersionsFile(full, st.ModTime().UnixNano())
		if err != nil {
			lastErr = err
			continue
		}
		if len(vers) > 0 {
			return vers, nil
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("uabuilder: no app version pool: %w", lastErr)
	}
	return nil, fmt.Errorf("uabuilder: no app version pool")
}

func loadAppVersionsFile(full string, mtime int64) ([]AppVersion, error) {
	appCacheMu.Lock()
	defer appCacheMu.Unlock()
	if c, ok := appCache[full]; ok && c.mtime == mtime {
		return c.versions, nil
	}

	f, err := os.Open(full)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := []AppVersion{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		out = append(out, AppVersion{Version: strings.TrimSpace(parts[0]), Build: strings.TrimSpace(parts[1])})
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	appCache[full] = &cachedApps{mtime: mtime, versions: out}
	return out, nil
}

// ─── Convenience loaders ────────────────────────────────────────────────────

func loadDensities() ([]string, error) {
	return loadStringList(filepath.Join("DeviceInfo", "densitis.txt"))
}
func loadCores() ([]string, error) {
	return loadStringList(filepath.Join("DeviceInfo", "device_cores.txt"))
}
func loadOSVersions() ([]string, error) {
	return loadStringList(filepath.Join("DeviceInfo", "devices_versions.txt"))
}
func loadScreenResolutions() ([]string, error) {
	return loadStringList(filepath.Join("DeviceInfo", "screen_resolution.txt"))
}
func loadCarriers() ([]string, error) {
	return loadStringList(filepath.Join("DeviceInfo", "carriers.txt"))
}
func loadChromeVersions() ([]string, error) {
	return loadStringList(filepath.Join("DeviceInfo", "chrome_versions.txt"))
}
func loadGoogleAppVersions() ([]string, error) {
	return loadStringList(filepath.Join("DeviceInfo", "googleapp_versions.txt"))
}

// pickRandom trả về 1 phần tử random từ list. "" nếu list rỗng.
func pickRandom(r interface{ Intn(int) int }, list []string) string {
	if len(list) == 0 {
		return ""
	}
	return list[r.Intn(len(list))]
}
