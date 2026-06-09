// Package fbdata — kho dữ liệu database cho Facebook UA generation.
//
// Port từ C# config folders:
//   C# ./config/fbapp_inf/versions_and_builds.txt  → Config/Fbapp/versions_and_builds.txt
//
// 3 POOL TÁCH RIÊNG (refactor 2026-05-28):
//   1. versions_and_builds_reg.txt  → ưu tiên cho REG flow
//   2. versions_and_builds_ver.txt  → ưu tiên cho VER flow
//   3. versions_and_builds.txt      → fallback chung khi 1/2 file trên rỗng
//
// Logic chọn:
//   - VersionsReg(): nếu pool reg rỗng → fallback pool chung
//   - VersionsVer(): nếu pool ver rỗng → fallback pool chung
//   - Versions(): luôn trả pool chung (backward-compat cho code cũ)
//
// Pattern "embed + override":
//   1. Default dataset được embed vào binary (ship kèm app, chạy được ngay)
//   2. Nếu user tạo file ở Config/DeviceInfo/ → override dataset embed
//   3. Nếu file user rỗng/không parse được → fallback về embed (không fail như C#)
package fbdata

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// DefaultDir là thư mục override do user quản lý.
	// Path khuyến nghị (2026-05-26): Config/DeviceInfo/ — gom chung với devices.txt, carriers.txt.
	// Path legacy: Config/Fbapp/ — vẫn được đọc làm fallback cho khách hàng cũ.
	DefaultDir = "Config/DeviceInfo" // path mới (ưu tiên)
	LegacyDir  = "Config/Fbapp"      // path cũ (fallback)

	// File names — pool chung + 2 pool tách reg/ver.
	VersionsAndBuildsFilename    = "versions_and_builds.txt"
	VersionsAndBuildsRegFilename = "versions_and_builds_reg.txt"
	VersionsAndBuildsVerFilename = "versions_and_builds_ver.txt"
)

// DefaultVersionsAndBuildsPath trả về đường dẫn override file CHUNG đầu tiên tồn tại.
// Ưu tiên Config/DeviceInfo/, fallback Config/Fbapp/. Nếu cả 2 đều không có,
// trả path mới (caller có thể tạo file ở đó).
func DefaultVersionsAndBuildsPath() string {
	newPath := filepath.Join(DefaultDir, VersionsAndBuildsFilename)
	if _, err := os.Stat(newPath); err == nil {
		return newPath
	}
	legacyPath := filepath.Join(LegacyDir, VersionsAndBuildsFilename)
	if _, err := os.Stat(legacyPath); err == nil {
		return legacyPath
	}
	return newPath // không tồn tại — trả path mới làm default
}

// splitDir — folder chứa 2 file split (_reg.txt + _ver.txt).
// LUÔN dùng Config/DeviceInfo/ — user đổi yêu cầu 2026-05-28 sang gom với devices.txt/carriers.txt.
func splitDir() string {
	return DefaultDir
}

// regPath / verPath — đường dẫn file reg/ver riêng. Cùng folder với file chung.
func regPath() string { return filepath.Join(splitDir(), VersionsAndBuildsRegFilename) }
func verPath() string { return filepath.Join(splitDir(), VersionsAndBuildsVerFilename) }

// EnsureDir tạo thư mục Config/DeviceInfo/ nếu chưa tồn tại.
func EnsureDir() error {
	return os.MkdirAll(DefaultDir, 0755)
}

// EnsureSplitFiles tạo 3 file pool ở Config/DeviceInfo/ nếu chưa tồn tại:
//   - versions_and_builds.txt        (CHUNG)
//   - versions_and_builds_reg.txt    (REG)
//   - versions_and_builds_ver.txt    (VER)
//
// File CHUNG: nếu Config/Fbapp/versions_and_builds.txt có data → COPY content
// (để user cũ không mất pool). Nếu legacy cũng không có → tạo rỗng.
//
// File split (_reg.txt + _ver.txt): luôn tạo RỖNG (0 bytes) — user tự điền version.
// Khi rỗng → caller dùng pool chung làm fallback (xem VersionsReg/VersionsVer).
//
// Gọi 1 lần lúc app start sau khi đã EnsureDir().
func EnsureSplitFiles() error {
	dir := splitDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 1. File CHUNG — tạo ở DeviceInfo nếu chưa có, copy từ Fbapp legacy nếu có.
	commonPath := filepath.Join(dir, VersionsAndBuildsFilename)
	if _, err := os.Stat(commonPath); os.IsNotExist(err) {
		var content []byte
		legacyPath := filepath.Join(LegacyDir, VersionsAndBuildsFilename)
		if data, err := os.ReadFile(legacyPath); err == nil && len(data) > 0 {
			content = data // copy từ Fbapp legacy giữ pool cũ
		}
		if err := os.WriteFile(commonPath, content, 0644); err != nil {
			return err
		}
	}

	// 2. File split — luôn tạo rỗng nếu chưa có.
	for _, p := range []string{regPath(), verPath()} {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			if err := os.WriteFile(p, nil, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// FbVersion mô tả 1 cặp Facebook Android app version + build number.
type FbVersion struct {
	Version string
	Build   string
}

// ParseVersionsAndBuilds parse nội dung file versions_and_builds.txt.
// Bỏ qua các dòng rỗng, không chứa "|", hoặc có field rỗng.
// Comment "#" đầu dòng cũng bị skip.
func ParseVersionsAndBuilds(content string) []FbVersion {
	var out []FbVersion
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "|") {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		v := strings.TrimSpace(parts[0])
		b := strings.TrimSpace(parts[1])
		if v == "" || b == "" {
			continue
		}
		out = append(out, FbVersion{Version: v, Build: b})
	}
	return out
}

// loadOverride đọc file override nếu tồn tại, return nil nếu không có/parse fail.
func loadOverride(path string) []FbVersion {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return ParseVersionsAndBuilds(string(data))
}

// ── In-memory cache ──────────────────────────────────────────────────────────

var (
	mu              sync.RWMutex
	defaultVersions []FbVersion // từ embed (set bởi fakeinfo.init)
	activeVersions  []FbVersion // pool CHUNG (versions_and_builds.txt)
	activeReg       []FbVersion // pool REG (versions_and_builds_reg.txt), nil/empty = chưa có
	activeVer       []FbVersion // pool VER (versions_and_builds_ver.txt), nil/empty = chưa có
	overridePath    string      // path của file CHUNG đang dùng
)

// SetDefaultVersions ghi đè default dataset và trigger merge với override.
// Được fakeinfo gọi trong init() sau khi parse file embed.
func SetDefaultVersions(v []FbVersion) {
	mu.Lock()
	defer mu.Unlock()
	defaultVersions = append([]FbVersion(nil), v...)
	applyMergeLocked()
}

// Reload tái nạp ALL pool từ disk (chung + reg + ver).
// path rỗng → dùng DefaultVersionsAndBuildsPath() cho pool chung.
// Pool reg/ver luôn từ DefaultDir (Config/DeviceInfo/).
func Reload(path string) {
	if path == "" {
		path = DefaultVersionsAndBuildsPath()
	}
	mu.Lock()
	defer mu.Unlock()
	overridePath = path
	applyMergeLocked()
	// Load split files (reg/ver). Rỗng → caller fallback sang pool chung.
	activeReg = loadOverride(regPath())
	activeVer = loadOverride(verPath())
}

// Versions trả về snapshot của active versions CHUNG.
// Thread-safe: trả về copy, caller có thể sửa thoải mái.
func Versions() []FbVersion {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]FbVersion, len(activeVersions))
	copy(out, activeVersions)
	return out
}

// VersionsReg trả về pool REG nếu có data, ngược lại fallback pool chung.
// Dùng cho RandomFbVersionReg() để build UA reg.
func VersionsReg() []FbVersion {
	mu.RLock()
	defer mu.RUnlock()
	if len(activeReg) > 0 {
		out := make([]FbVersion, len(activeReg))
		copy(out, activeReg)
		return out
	}
	out := make([]FbVersion, len(activeVersions))
	copy(out, activeVersions)
	return out
}

// VersionsVer trả về pool VER nếu có data, ngược lại fallback pool chung.
// Dùng cho RandomFbVersionVer() để build UA verify.
func VersionsVer() []FbVersion {
	mu.RLock()
	defer mu.RUnlock()
	if len(activeVer) > 0 {
		out := make([]FbVersion, len(activeVer))
		copy(out, activeVer)
		return out
	}
	out := make([]FbVersion, len(activeVersions))
	copy(out, activeVersions)
	return out
}

// Size trả về số FbVersion đang active trong pool chung.
func Size() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(activeVersions)
}

// SizeReg/SizeVer — số version trong pool split (0 nếu file rỗng/chưa tồn tại).
func SizeReg() int { mu.RLock(); defer mu.RUnlock(); return len(activeReg) }
func SizeVer() int { mu.RLock(); defer mu.RUnlock(); return len(activeVer) }

// OverrideActive trả về true nếu đang dùng file user override (file tồn tại + có dữ liệu hợp lệ).
func OverrideActive() bool {
	mu.RLock()
	defer mu.RUnlock()
	if overridePath == "" {
		return false
	}
	return loadOverride(overridePath) != nil
}

// ── Internal ─────────────────────────────────────────────────────────────────

// applyMergeLocked tính activeVersions = override CHUNG (nếu có) HOẶC default.
func applyMergeLocked() {
	if overridePath != "" {
		if userList := loadOverride(overridePath); len(userList) > 0 {
			activeVersions = userList
			return
		}
	}
	activeVersions = append([]FbVersion(nil), defaultVersions...)
}
