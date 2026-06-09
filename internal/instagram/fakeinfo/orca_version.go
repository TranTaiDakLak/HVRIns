// orca_version.go — Pool phiên bản Messenger/Orca (FBAV + FBBV) cho User-Agent.
//
// Giống FB4A (versions_and_builds_*.txt): random 1 CẶP (version, build) THẬT thay vì
// hardcode 1 giá trị. KHÔNG bịa build number — chỉ dùng cặp lấy từ capture thật.
//
// Mở rộng: thêm dòng "version|build" vào Config/DeviceInfo/versions_and_builds_orca.txt
// (cạnh exe). Nếu file thiếu/rỗng → dùng fallback dưới đây.
//
// LƯU Ý: doc_id/bloks_versioning_id trong BODY hiện khoá ở v530. UA FBAV khác (vd 529)
// là lệch nhẹ — FB chấp nhận (FB4A gửi UA 415 với doc_id 565 vẫn chạy). Khi có capture
// version mới + doc_id/bloks tương ứng thì mới nên thêm version xa hơn.
package fakeinfo

import (
	"math/rand"
	"strings"
	"sync"
	"time"
)

// OrcaAppVersion — cặp (FBAV, FBBV) của app Messenger/Orca.
type OrcaAppVersion struct {
	Version string // FBAV, vd "530.1.0.67.107"
	Build   string // FBBV, vd "814020040"
}

// orcaAppVersionsFallback — cặp THẬT trích từ capture Messenger (KHÔNG bịa).
//
//	V4/V5 → 530.1.0.67.107 / 814020040
//	V3    → 529.0.0.43.109 / 812359520
var orcaAppVersionsFallback = []OrcaAppVersion{
	{Version: "530.1.0.67.107", Build: "814020040"},
	{Version: "529.0.0.43.109", Build: "812359520"},
}

var (
	orcaAppVersionsOnce sync.Once
	orcaAppVersionsPool []OrcaAppVersion
)

// loadOrcaAppVersions đọc pool từ config (lazy, cache), fallback nếu thiếu.
func loadOrcaAppVersions() []OrcaAppVersion {
	orcaAppVersionsOnce.Do(func() {
		for _, line := range loadDeviceInfoLines("versions_and_builds_orca.txt") {
			parts := strings.SplitN(strings.TrimSpace(line), "|", 2)
			if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
				orcaAppVersionsPool = append(orcaAppVersionsPool, OrcaAppVersion{
					Version: strings.TrimSpace(parts[0]),
					Build:   strings.TrimSpace(parts[1]),
				})
			}
		}
		if len(orcaAppVersionsPool) == 0 {
			orcaAppVersionsPool = orcaAppVersionsFallback
		}
	})
	return orcaAppVersionsPool
}

// RandomOrcaAppVersion trả 1 cặp (FBAV, FBBV) Orca ngẫu nhiên từ pool (config hoặc fallback).
// Gọi 1 lần / account để UA nhất quán xuyên suốt mọi bước.
func RandomOrcaAppVersion() (fbav, fbbv string) {
	pool := loadOrcaAppVersions()
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	v := pool[r.Intn(len(pool))]
	return v.Version, v.Build
}

// ─── Messenger Android (appmessv3) — pool riêng theo reg/ver ──────────────────
// Đọc Config/DeviceInfo/mess_versions_and_builds_<kind>.txt (fallback base file →
// fallback cặp orca hardcode). Tách reg/ver để user chỉnh version riêng mỗi luồng.

var (
	messAndrMu    sync.Mutex
	messAndrCache = map[string][]OrcaAppVersion{}
)

func loadMessAndroidVersions(kind string) []OrcaAppVersion {
	messAndrMu.Lock()
	defer messAndrMu.Unlock()
	if p, ok := messAndrCache[kind]; ok {
		return p
	}
	lines := loadDeviceInfoLines("mess_versions_and_builds_" + kind + ".txt")
	if len(lines) == 0 {
		lines = loadDeviceInfoLines("mess_versions_and_builds.txt")
	}
	var out []OrcaAppVersion
	for _, line := range lines {
		parts := strings.SplitN(strings.TrimSpace(line), "|", 2)
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			out = append(out, OrcaAppVersion{
				Version: strings.TrimSpace(parts[0]),
				Build:   strings.TrimSpace(parts[1]),
			})
		}
	}
	if len(out) == 0 {
		out = orcaAppVersionsFallback
	}
	messAndrCache[kind] = out
	return out
}

// RandomMessOrcaAppVersion trả cặp (FBAV, FBBV) Messenger Android từ pool theo kind
// ("reg"/"ver"). Gọi 1 lần / account.
func RandomMessOrcaAppVersion(kind string) (fbav, fbbv string) {
	pool := loadMessAndroidVersions(kind)
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	v := pool[r.Intn(len(pool))]
	return v.Version, v.Build
}
