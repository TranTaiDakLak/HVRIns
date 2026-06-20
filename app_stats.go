package main

import (
	"sort"
	"strings"

	"HVRIns/internal/instagram"
)

// regVersionStat — đếm thành công / thất bại của 1 version reg trong run hiện tại.
type regVersionStat struct {
	Success int
	Fail    int
}

// mailDomainStat — đếm verify / live theo domain mail trong run hiện tại.
type mailDomainStat struct {
	Veri int // tổng account đã verify (có email)
	Live int // số live
}

// MailDomainStatRow — 1 dòng trong tab "Thống kê Mail Domain".
type MailDomainStatRow struct {
	Index  int     `json:"index"`
	Domain string  `json:"domain"`
	Veri   int     `json:"veri"`
	Live   int     `json:"live"`
	Die    int     `json:"die"`
	Rate   float64 `json:"rate"` // live / veri (0–1)
}

// RegStatRow — 1 dòng trong tab "Thống kê REG" (STT, API, thành công, thất bại, tỉ lệ).
type RegStatRow struct {
	Index    int    `json:"index"`    // STT (1-based)
	Platform string `json:"platform"` // API / version (vd "s560", "android")
	Success  int    `json:"success"`
	Fail     int    `json:"fail"`
	Total    int    `json:"total"`
}

// NewApp khởi tạo App struct với slice accounts rỗng.
// Wails gọi hàm này 1 lần khi app khởi động — không nhận tham số.

// resetRegStats — khởi tạo lại bảng thống kê reg cho run mới (seed sẵn các version đã chọn
// để tab hiện đủ dòng kể cả khi version đó chưa có account nào).
func (a *App) resetRegStats(platforms []string) {
	a.regStatsMu.Lock()
	a.regStats = make(map[string]*regVersionStat, len(platforms)+1)
	for _, p := range platforms {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := a.regStats[p]; !ok {
			a.regStats[p] = &regVersionStat{}
		}
	}
	a.regStatsMu.Unlock()
}

// recordRegOutcome — ghi nhận 1 lần reg của version `platform` là thành công hay thất bại.
func (a *App) recordRegOutcome(platform string, success bool) {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return
	}
	a.regStatsMu.Lock()
	if a.regStats == nil {
		a.regStats = make(map[string]*regVersionStat)
	}
	st := a.regStats[platform]
	if st == nil {
		st = &regVersionStat{}
		a.regStats[platform] = st
	}
	if success {
		st.Success++
	} else {
		st.Fail++
	}
	a.regStatsMu.Unlock()
}

// GetRegStats — thống kê reg theo version cho run hiện tại (hoặc run gần nhất nếu đã dừng).
// Frontend poll mỗi ~10s. Sort theo platform để thứ tự dòng ổn định.
func (a *App) GetRegStats() []RegStatRow {
	a.regStatsMu.Lock()
	rows := make([]RegStatRow, 0, len(a.regStats))
	for k, st := range a.regStats {
		rows = append(rows, RegStatRow{
			Platform: k,
			Success:  st.Success,
			Fail:     st.Fail,
			Total:    st.Success + st.Fail,
		})
	}
	a.regStatsMu.Unlock()
	sort.Slice(rows, func(i, j int) bool { return rows[i].Platform < rows[j].Platform })
	for i := range rows {
		rows[i].Index = i + 1
	}
	return rows
}

// === Verify stats (song song với reg stats ở trên) ===

// verifyPlatformDisplayName chuyển internal platform ID → tên hiển thị cho thống kê.
// Đảm bảo stats luôn dùng cùng key với UI label người dùng đã chọn.
// sXXX platforms giữ nguyên vì tên internal đã khớp với label UI.
func verifyPlatformDisplayName(internal string) string {
	switch internal {
	case instagram.PlatformWeb:
		return "api mfb"
	case instagram.PlatformWebAndroid:
		return "api web andr"
	case instagram.PlatformS23:
		return "api android"
	case instagram.PlatformAndroid:
		return "api token"
	default:
		return internal
	}
}

func (a *App) resetVerifyStats(platforms []string) {
	a.verifyStatsMu.Lock()
	a.verifyStats = make(map[string]*regVersionStat, len(platforms)+1)
	for _, p := range platforms {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Resolve UI key → internal → display name để match với recordVerifyOutcome.
		// VD: "api web andr" → "webandroid" → "api web andr", "s561v99" → "s561v3".
		if resolved := verifyPlatformFromType(p); resolved != "" {
			p = verifyPlatformDisplayName(resolved)
		}
		if _, ok := a.verifyStats[p]; !ok {
			a.verifyStats[p] = &regVersionStat{}
		}
	}
	a.verifyStatsMu.Unlock()
	a.verifyPlatformRR.Store(0)
}

func (a *App) recordVerifyOutcome(platform string, success bool) {
	platform = verifyPlatformDisplayName(strings.TrimSpace(platform))
	if platform == "" {
		return
	}
	a.verifyStatsMu.Lock()
	if a.verifyStats == nil {
		a.verifyStats = make(map[string]*regVersionStat)
	}
	st := a.verifyStats[platform]
	if st == nil {
		st = &regVersionStat{}
		a.verifyStats[platform] = st
	}
	if success {
		st.Success++
	} else {
		st.Fail++
	}
	a.verifyStatsMu.Unlock()
}

// GetVerifyStats — thống kê verify theo version cho run hiện tại (hoặc run gần nhất).
func (a *App) GetVerifyStats() []RegStatRow {
	a.verifyStatsMu.Lock()
	rows := make([]RegStatRow, 0, len(a.verifyStats))
	for k, st := range a.verifyStats {
		rows = append(rows, RegStatRow{
			Platform: k,
			Success:  st.Success,
			Fail:     st.Fail,
			Total:    st.Success + st.Fail,
		})
	}
	a.verifyStatsMu.Unlock()
	sort.Slice(rows, func(i, j int) bool { return rows[i].Platform < rows[j].Platform })
	for i := range rows {
		rows[i].Index = i + 1
	}
	return rows
}

// === Mail domain stats ===

func (a *App) resetMailDomainStats() {
	a.mailDomainStatsMu.Lock()
	a.mailDomainStats = make(map[string]*mailDomainStat)
	a.mailDomainStatsMu.Unlock()
}

func (a *App) recordMailDomainOutcome(email string, isLive bool) {
	at := strings.Index(email, "@")
	if at < 0 {
		return
	}
	domain := strings.ToLower(strings.TrimSpace(email[at:]))
	if domain == "" {
		return
	}
	a.mailDomainStatsMu.Lock()
	if a.mailDomainStats == nil {
		a.mailDomainStats = make(map[string]*mailDomainStat)
	}
	st := a.mailDomainStats[domain]
	if st == nil {
		st = &mailDomainStat{}
		a.mailDomainStats[domain] = st
	}
	st.Veri++
	if isLive {
		st.Live++
	}
	a.mailDomainStatsMu.Unlock()
}

// GetMailDomainStats — thống kê verify theo domain mail, sort theo Live desc.
func (a *App) GetMailDomainStats() []MailDomainStatRow {
	a.mailDomainStatsMu.Lock()
	rows := make([]MailDomainStatRow, 0, len(a.mailDomainStats))
	for domain, st := range a.mailDomainStats {
		die := st.Veri - st.Live
		if die < 0 {
			die = 0
		}
		rate := 0.0
		if st.Veri > 0 {
			rate = float64(st.Live) / float64(st.Veri)
		}
		rows = append(rows, MailDomainStatRow{
			Domain: domain,
			Veri:   st.Veri,
			Live:   st.Live,
			Die:    die,
			Rate:   rate,
		})
	}
	a.mailDomainStatsMu.Unlock()
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Live != rows[j].Live {
			return rows[i].Live > rows[j].Live
		}
		return rows[i].Domain < rows[j].Domain
	})
	for i := range rows {
		rows[i].Index = i + 1
	}
	return rows
}

// === Build UA stats (thống kê FBAV version theo Reg / Veri) ===

func extractFBAV(ua string) string {
	idx := strings.Index(ua, "FBAV/")
	if idx == -1 {
		return ""
	}
	rest := ua[idx+5:]
	if end := strings.IndexAny(rest, ";]"); end != -1 {
		return rest[:end]
	}
	return rest
}

func (a *App) resetBuildUAStats() {
	a.buildUARegStatsMu.Lock()
	a.buildUARegStats = make(map[string]*regVersionStat)
	a.buildUARegStatsMu.Unlock()
	a.buildUAVerStatsMu.Lock()
	a.buildUAVerStats = make(map[string]*regVersionStat)
	a.buildUAVerStatsMu.Unlock()
}

func (a *App) recordBuildUARegVersion(fbav string, success bool) {
	if fbav == "" {
		return
	}
	a.buildUARegStatsMu.Lock()
	if a.buildUARegStats == nil {
		a.buildUARegStats = make(map[string]*regVersionStat)
	}
	st := a.buildUARegStats[fbav]
	if st == nil {
		st = &regVersionStat{}
		a.buildUARegStats[fbav] = st
	}
	if success {
		st.Success++
	} else {
		st.Fail++
	}
	a.buildUARegStatsMu.Unlock()
}

func (a *App) recordBuildUAVerVersion(fbav string, success bool) {
	if fbav == "" {
		return
	}
	a.buildUAVerStatsMu.Lock()
	if a.buildUAVerStats == nil {
		a.buildUAVerStats = make(map[string]*regVersionStat)
	}
	st := a.buildUAVerStats[fbav]
	if st == nil {
		st = &regVersionStat{}
		a.buildUAVerStats[fbav] = st
	}
	if success {
		st.Success++
	} else {
		st.Fail++
	}
	a.buildUAVerStatsMu.Unlock()
}

func (a *App) GetBuildUARegStats() []RegStatRow {
	a.buildUARegStatsMu.Lock()
	rows := make([]RegStatRow, 0, len(a.buildUARegStats))
	for k, st := range a.buildUARegStats {
		rows = append(rows, RegStatRow{Platform: k, Success: st.Success, Fail: st.Fail, Total: st.Success + st.Fail})
	}
	a.buildUARegStatsMu.Unlock()
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Total != rows[j].Total {
			return rows[i].Total > rows[j].Total
		}
		return rows[i].Platform < rows[j].Platform
	})
	for i := range rows {
		rows[i].Index = i + 1
	}
	return rows
}

func (a *App) GetBuildUAVerStats() []RegStatRow {
	a.buildUAVerStatsMu.Lock()
	rows := make([]RegStatRow, 0, len(a.buildUAVerStats))
	for k, st := range a.buildUAVerStats {
		rows = append(rows, RegStatRow{Platform: k, Success: st.Success, Fail: st.Fail, Total: st.Success + st.Fail})
	}
	a.buildUAVerStatsMu.Unlock()
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Total != rows[j].Total {
			return rows[i].Total > rows[j].Total
		}
		return rows[i].Platform < rows[j].Platform
	})
	for i := range rows {
		rows[i].Index = i + 1
	}
	return rows
}
