package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	verifybase "HVRIns/internal/instagram/verify/verifybase"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// UploadSiteSourceConfig — cấu hình cho 1 nguồn (reg hoặc ver)
type UploadSiteSourceConfig struct {
	Enabled bool `json:"enabled"`
}

// UploadSiteConfig — cấu hình đẩy tài khoản lên banclone.pro
type UploadSiteConfig struct {
	Reg                  UploadSiteSourceConfig `json:"reg"`
	Ver                  UploadSiteSourceConfig `json:"ver"`
	Code                 string                 `json:"code"`            // mã kho hàng (stock code)
	ApiKey               string                 `json:"apiKey"`          // API key admin
	AdminUsername        string                 `json:"adminUsername"`   // tài khoản đăng nhập banclone.pro
	AdminPassword        string                 `json:"adminPassword"`   // mật khẩu đăng nhập banclone.pro
	FilterDuplicate      bool                   `json:"filterDuplicate"` // true=lọc trùng UID
	DelayCheckSec        int                    `json:"delayCheckSec"`
	AccPerBatch          int                    `json:"accPerBatch"`
	DelayBetweenBatchSec int                    `json:"delayBetweenBatchSec"`
}

func defaultUploadSiteConfig() UploadSiteConfig {
	return UploadSiteConfig{
		Reg:                  UploadSiteSourceConfig{Enabled: false},
		Ver:                  UploadSiteSourceConfig{Enabled: false},
		Code:                 "69ea28f9e5e3e",
		ApiKey:               "6ddcacd6d2b59363401c516292a786aaq2Aa14OynFgKJi5lQY7tcEZhXjIvBPs0",
		FilterDuplicate:      false,
		DelayCheckSec:        25,
		AccPerBatch:          900,
		DelayBetweenBatchSec: 9,
	}
}

// SaveUploadSiteConfig lưu cấu hình đẩy tài khoản vào Config/Settings/uploadsite.json
func (a *App) SaveUploadSiteConfig(data UploadSiteConfig) string {
	const settingsDir = "Config/Settings"
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "Lỗi marshal: " + err.Error()
	}
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return "Lỗi tạo thư mục: " + err.Error()
	}
	if err := os.WriteFile(filepath.Join(settingsDir, "uploadsite.json"), b, 0644); err != nil {
		return "Lỗi ghi file: " + err.Error()
	}
	return "OK"
}

// LoadUploadSiteConfig đọc cấu hình đẩy tài khoản từ Config/Settings/uploadsite.json
func (a *App) LoadUploadSiteConfig() UploadSiteConfig {
	cfg := defaultUploadSiteConfig()
	b, err := os.ReadFile(filepath.Join("Config/Settings", "uploadsite.json"))
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(b, &cfg)
	return cfg
}

// ============================================================================
// Upload Site Runner — WAL (Write-Ahead Log) + Retry/Backoff + Soft-Stop
// ============================================================================
// Cốt lõi:
//   - In-memory queue + retry với exponential backoff (không persist cross-session).
//   - Dedup UID: cùng 1 UID không enqueue 2 lần trong cùng session reg.
//   - Stats + log per-session (lưu trong currentResultPath, không có folder Config/Upload riêng).
//   - Stop = soft-stop: chờ drain hết acc trong queue mới exit.
//   - "Acc cũ" từ session trước KHÔNG auto-upload — fresh start mỗi run.
// ============================================================================

const (
	uploadStatsFile = "upload_stats.json"
	uploadLogFile   = "upload_push_log.txt"

	uploadLogRotateInterval = 2 * time.Hour
	uploadLogMaxSize        = 2 * 1024 * 1024 // 2MB hard cap (phòng rotate chưa kịp)
	// uploadHardStopTimeout — Task 4: 10p → 60s. Soft-stop chờ in-flight push xong
	// (mỗi push max 300s từ pushTimeoutFor). Sau 60s bất kể, hard-cancel ctx →
	// goroutine push cũng exit qua ctx.Done. Pending acc còn trong pending.txt sẽ
	// được load lại khi mở lại upload. Trade-off: lose in-flight push >60s sau Stop
	// để Stop thật sự responsive.
	uploadHardStopTimeout = 60 * time.Second
)

// UploadStats — counter cho UI. Lưu per-session trong currentResultPath.
type UploadStats struct {
	TotalUploaded       int    `json:"totalUploaded"`
	TotalFailed         int    `json:"totalFailed"`
	PendingCount        int    `json:"pendingCount"` // batch + pendingRetry + uploadCh (in-memory)
	ConsecutiveFailures int    `json:"consecutiveFailures"`
	DuplicateSkipped    int    `json:"duplicateSkipped"` // số UID bị dedup bỏ qua
	LastUploadAt        string `json:"lastUploadAt"`
	LastErrorAt         string `json:"lastErrorAt"`
	LastError           string `json:"lastError"`
	LastRotateAt        string `json:"lastRotateAt"`
	StartedAt           string `json:"startedAt"`
}

// uploadInMemoryPending — count "đang chờ" gồm pendingRetry + batch + channel.
// Cập nhật từ runUploadSite qua atomic.
var uploadPendingInMem atomic.Int32

// resultDir trả về thư mục result hiện tại; rỗng = chưa có session đang chạy.
func (a *App) resultDir() string {
	a.resultPathMu.Lock()
	defer a.resultPathMu.Unlock()
	return a.currentResultPath
}

func (a *App) uploadStatsPath() string {
	d := a.resultDir()
	if d == "" {
		return ""
	}
	return filepath.Join(d, uploadStatsFile)
}

func (a *App) uploadLogPath() string {
	d := a.resultDir()
	if d == "" {
		return ""
	}
	return filepath.Join(d, uploadLogFile)
}

// ──────────────────────────────────────────────────────────────────────────
// Dedup UID — tránh push trùng cùng 1 UID trong session
// ──────────────────────────────────────────────────────────────────────────

// extractUIDFromLine lấy UID từ dòng format "uid|pass|2fa|..." (FormatVerify).
func extractUIDFromLine(line string) string {
	if i := strings.IndexByte(line, '|'); i > 0 {
		return strings.TrimSpace(line[:i])
	}
	return strings.TrimSpace(line)
}

// extractTokenFromLine lấy Facebook access token (bắt đầu bằng EAA) từ dòng pipe-separated.
func extractTokenFromLine(line string) string {
	for _, f := range strings.Split(line, "|") {
		f = strings.TrimSpace(f)
		if strings.HasPrefix(f, "EAA") && len(f) > 20 {
			return f
		}
	}
	return ""
}

// ResetUploadSession xoá toàn bộ dedup UIDs + drain queue + clear retry — gọi khi bắt đầu run mới.
// Hook vào RunVerify/RunRegister sau khi set currentResultPath.
// Đảm bảo run mới không kế thừa acc cũ chưa push xong từ session trước.
func (a *App) ResetUploadSession() {
	a.uploadSeenUIDs.Range(func(k, _ any) bool {
		a.uploadSeenUIDs.Delete(k)
		return true
	})
	// Drain channel non-blocking — vứt acc cũ còn sót sau hard-stop của session trước.
	drained := 0
drainLoop:
	for {
		select {
		case <-a.uploadCh:
			drained++
		default:
			break drainLoop
		}
	}
	// Clear retry queue cũ
	a.uploadRetryMu.Lock()
	dropped := len(a.uploadRetryQueue)
	a.uploadRetryQueue = nil
	a.uploadRetryMu.Unlock()
	if drained > 0 || dropped > 0 {
		slog.Info("ResetUploadSession: drop state cũ", "queue", drained, "retry", dropped)
	}
	uploadPendingInMem.Store(0)
}

// enqueueForUpload đẩy line vào queue, dedup theo UID.
// In-memory only — app crash = mất acc trong queue (theo yêu cầu user).
func (a *App) enqueueForUpload(line string) {
	if line == "" {
		return
	}
	uid := extractUIDFromLine(line)
	if uid != "" {
		if _, loaded := a.uploadSeenUIDs.LoadOrStore(uid, true); loaded {
			// UID đã enqueue trong session này → bỏ qua (tránh trùng).
			a.updateUploadStats(func(s *UploadStats) { s.DuplicateSkipped++ })
			slog.Debug("enqueueForUpload: skip duplicate UID", "uid", uid)
			return
		}
	}
	uploadPendingInMem.Add(1)
	// Block nếu channel đầy — đảm bảo không mất acc (channel size 5000 đủ rộng).
	a.uploadCh <- line
}

// ──────────────────────────────────────────────────────────────────────────
// Stats: lưu trong currentResultPath/upload_stats.json
// ──────────────────────────────────────────────────────────────────────────

func (a *App) loadUploadStats() UploadStats {
	a.uploadStatsMu.Lock()
	defer a.uploadStatsMu.Unlock()
	var s UploadStats
	path := a.uploadStatsPath()
	if path == "" {
		return s
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	return s
}

func (a *App) saveUploadStatsLocked(s UploadStats) {
	path := a.uploadStatsPath()
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

// updateUploadStats đọc-modify-ghi atomic dưới mutex. Skip nếu chưa có session.
func (a *App) updateUploadStats(fn func(*UploadStats)) {
	path := a.uploadStatsPath()
	if path == "" {
		return // chưa có session → không lưu (theo yêu cầu user)
	}
	a.uploadStatsMu.Lock()
	defer a.uploadStatsMu.Unlock()
	var s UploadStats
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	fn(&s)
	a.saveUploadStatsLocked(s)
}

// GetUploadStats — frontend API hiện counter trên UI.
func (a *App) GetUploadStats() UploadStats {
	s := a.loadUploadStats()
	s.PendingCount = int(uploadPendingInMem.Load())
	return s
}

// ──────────────────────────────────────────────────────────────────────────
// Logging: emit event + ghi file (có level + rotation)
// ──────────────────────────────────────────────────────────────────────────

const (
	logLevelInfo  = "info"
	logLevelOk    = "ok"
	logLevelWarn  = "warn"
	logLevelError = "error"
)

func (a *App) emitUploadLog(source, msg string, uploaded int) {
	a.emitUploadLogL(source, msg, uploaded, "")
}

// emitUploadLogL — biến thể có level. level rỗng → tự suy từ icon đầu msg.
func (a *App) emitUploadLogL(source, msg string, uploaded int, level string) {
	if level == "" {
		switch {
		case strings.HasPrefix(msg, "❌"):
			level = logLevelError
		case strings.HasPrefix(msg, "⚠"):
			level = logLevelWarn
		case strings.HasPrefix(msg, "✅"):
			level = logLevelOk
		default:
			level = logLevelInfo
		}
	}
	runtime.EventsEmit(a.ctx, "upload-site:log", map[string]interface{}{
		"source":   source,
		"msg":      msg,
		"uploaded": uploaded,
		"level":    level,
	})

	// Ghi vào upload_push_log.txt trong currentResultPath (per-session).
	// Nếu chưa có session → chỉ emit event, không ghi file.
	path := a.uploadLogPath()
	if path == "" {
		return
	}
	a.uploadLogMu.Lock()
	defer a.uploadLogMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	// Hard cap: nếu file > uploadLogMaxSize → truncate ngay (defense ngoài rotate timer).
	if fi, err := os.Stat(path); err == nil && fi.Size() > uploadLogMaxSize {
		_ = os.WriteFile(path, []byte{}, 0600)
	}
	line := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err == nil {
		_, _ = f.WriteString(line)
		_ = f.Close()
	}
}

// runUploadLogRotator chạy nền: mỗi 2h truncate log file + emit clear UI.
func (a *App) runUploadLogRotator(ctx context.Context) {
	t := time.NewTicker(uploadLogRotateInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if path := a.uploadLogPath(); path != "" {
				a.uploadLogMu.Lock()
				_ = os.WriteFile(path, []byte{}, 0600)
				a.uploadLogMu.Unlock()
			}
			a.updateUploadStats(func(s *UploadStats) {
				s.LastRotateAt = time.Now().Format(time.RFC3339)
			})
			runtime.EventsEmit(a.ctx, "upload-site:log-cleared", nil)
		}
	}
}

// ClearUploadLog — frontend gọi để clear ngay (nút Trash).
func (a *App) ClearUploadLog() {
	if path := a.uploadLogPath(); path != "" {
		a.uploadLogMu.Lock()
		_ = os.WriteFile(path, []byte{}, 0600)
		a.uploadLogMu.Unlock()
	}
	runtime.EventsEmit(a.ctx, "upload-site:log-cleared", nil)
}

// ──────────────────────────────────────────────────────────────────────────
// Lifecycle: Start / Stop / ensureRunning
// ──────────────────────────────────────────────────────────────────────────

// StartUploadSite bắt đầu goroutine upload (manual từ UI nếu cần).
func (a *App) StartUploadSite() string {
	a.uploadSiteMu.Lock()
	defer a.uploadSiteMu.Unlock()
	if a.uploadSiteCancel != nil {
		return "ALREADY_RUNNING"
	}
	cfg := a.LoadUploadSiteConfig()
	if !cfg.Reg.Enabled && !cfg.Ver.Enabled {
		return "NO_SOURCE"
	}
	a.startUploadSiteLocked(cfg)
	return "OK"
}

// startUploadSiteLocked — caller giữ uploadSiteMu.
func (a *App) startUploadSiteLocked(cfg UploadSiteConfig) {
	a.uploadStopping.Store(false)
	// Parent = a.ctx (Wails app lifecycle). Khi app shutdown, OnShutdown cancel a.ctx
	// → cascade cancel xuống upload runner + log rotator → goroutine exit gracefully.
	// Trước đây dùng context.Background() → upload tiếp tục chạy ngầm sau khi app close window.
	parent := a.ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	a.uploadSiteCancel = cancel
	// Bump gen — defer của run cũ (nếu chưa exit) sẽ thấy gen khác → KHÔNG clear cancel mới.
	a.uploadSiteGen++
	myGen := a.uploadSiteGen
	slog.Info("upload pool create", "runID", fmt.Sprintf("upload#%d", myGen))

	rotCtx, rotCancel := context.WithCancel(parent)
	a.uploadLogRotateCancel = rotCancel
	go a.runUploadLogRotator(rotCtx)

	go a.runUploadSite(ctx, cfg, myGen)
}

// StopUploadSite — soft-stop: chờ drain hết pending mới exit (bảo toàn acc).
// Block tối đa uploadHardStopTimeout; nếu site banclone die thì hard-cancel
// nhưng pending.txt vẫn còn → lần sau mở app load lại.
func (a *App) StopUploadSite() {
	a.uploadSiteMu.Lock()
	if a.uploadSiteCancel == nil {
		a.uploadSiteMu.Unlock()
		return
	}
	a.uploadStopping.Store(true)
	cancel := a.uploadSiteCancel
	a.uploadSiteMu.Unlock()

	a.emitUploadLogL("", "⏸ Đang dừng — chờ upload xong các acc còn lại...", 0, logLevelInfo)

	// Chờ runner exit tự nhiên (pending.txt rỗng) hoặc timeout.
	deadline := time.After(uploadHardStopTimeout)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			a.emitUploadLogL("", "⚠ Hard-stop sau timeout — pending.txt còn dữ liệu, mở app sau sẽ tự upload lại", 0, logLevelWarn)
			cancel()
			return
		case <-tick.C:
			a.uploadSiteMu.Lock()
			done := a.uploadSiteCancel == nil
			a.uploadSiteMu.Unlock()
			if done {
				return
			}
		}
	}
}

// ensureUploadRunning khởi động goroutine upload nếu chưa chạy.
// Gọi từ ver/reg callback để auto-start khi cần.
func (a *App) ensureUploadRunning(cfg UploadSiteConfig) {
	a.uploadSiteMu.Lock()
	defer a.uploadSiteMu.Unlock()
	if a.uploadSiteCancel != nil {
		return
	}
	a.startUploadSiteLocked(cfg)
}

// ──────────────────────────────────────────────────────────────────────────
// Main loop — Parallel pushes + Capped batch + Retry queue
// ──────────────────────────────────────────────────────────────────────────
//
// Cốt lõi sửa "death spiral" (batch tăng vô tận khi push lỗi):
//   - Mỗi push CỐ ĐỊNH ≤ accPerBatch (không gộp pendingRetry + batch thành mega-batch).
//   - Push chạy goroutine riêng → main loop không block khi site phản hồi chậm.
//   - Concurrency limit = uploadMaxConcurrent (mặc định 3) → không hammer site.
//   - Failed batch → quay vào retryQueue với retryAt = now + backoff (10s→30s→60s→2p cap).
//   - Delay giữa batch tính TỪ "send-start" của push trước (không phải sau khi xong).
// ──────────────────────────────────────────────────────────────────────────

const uploadMaxConcurrent = 1 // số push song song tối đa

// uploadMaxRetryAttempts — sau N attempts vẫn fail → drop batch (log + ghi pending.txt
// đã xử lý ở caller). Tránh retry vô hạn khi site die 24/7 → retryQueue grow vô tận.
// 5 lần × backoff 10s/30s/60s/2p/2p = ~5 phút trước khi drop. Site nào bị flag thật
// thì sau 5p coi như chết hẳn, đẩy thêm cũng vô ích.
const uploadMaxRetryAttempts = 5

// uploadMaxRetryQueueLen — cap hard size của retryQueue. Nếu vượt → drop oldest.
// Defense in-depth: kể cả khi site die rất lâu, RAM không grow vô tận.
const uploadMaxRetryQueueLen = 500

// retryItem — 1 batch push lỗi đang chờ retry.
type retryItem struct {
	accs     []string
	retryAt  time.Time
	attempts int
}

// uploadBackoff trả về thời gian chờ trước khi retry batch đó.
// 10s → 30s → 60s → 2p cap. Ngắn vì retry chạy parallel, không block.
func uploadBackoff(attempts int) time.Duration {
	switch {
	case attempts <= 0:
		return 0
	case attempts == 1:
		return 10 * time.Second
	case attempts == 2:
		return 30 * time.Second
	case attempts == 3:
		return 60 * time.Second
	default:
		return 2 * time.Minute
	}
}

func (a *App) runUploadSite(ctx context.Context, cfg UploadSiteConfig, myGen int64) {
	defer func() {
		a.uploadSiteMu.Lock()
		// CHỈ clear cancel + rotate khi gen còn khớp — nếu Stop→Start nhanh, run mới
		// đã bump gen và assign cancel mới; defer của run cũ KHÔNG được phép clear nhầm.
		if a.uploadSiteGen == myGen {
			a.uploadSiteCancel = nil
			if a.uploadLogRotateCancel != nil {
				a.uploadLogRotateCancel()
				a.uploadLogRotateCancel = nil
			}
		}
		a.uploadSiteMu.Unlock()
		// uploadStopping flag chỉ reset nếu là run hiện tại — nếu run mới đã start,
		// nó tự reset stopping=false rồi (line 4002), không cần đụng.
		if a.uploadSiteGen == myGen {
			a.uploadStopping.Store(false)
		}
		// Task 4: close idle TCP/TLS conn của bancloneTransport sau run.
		// Transport là singleton (shared giữa các run) nhưng giải phóng idle conn
		// sau mỗi run giảm RSS rõ rệt khi user chạy upload ngắt quãng. Active push
		// đã exit qua wg-equivalent (inFlight + signalWake → main loop), không bị abort.
		bancloneTransport.CloseIdleConnections()
		slog.Info("upload pool cleanup", "runID", fmt.Sprintf("upload#%d", myGen))
		a.emitUploadLogL("", "🛑 Đã dừng goroutine upload.", 0, logLevelInfo)
		runtime.EventsEmit(a.ctx, "upload-site:stopped", nil)
	}()

	a.updateUploadStats(func(s *UploadStats) {
		s.StartedAt = time.Now().Format(time.RFC3339)
	})

	checkInterval := time.Duration(cfg.DelayCheckSec) * time.Second
	if checkInterval <= 0 {
		checkInterval = 30 * time.Second
	}

	a.emitUploadLogL("", fmt.Sprintf("🚀 Bắt đầu — tối đa %d acc/lần, %d push song song, delay check %ds",
		cfg.AccPerBatch, uploadMaxConcurrent, cfg.DelayCheckSec), 0, logLevelInfo)

	var (
		batch          []string     // accs đang tích lũy chờ gửi
		inFlight       atomic.Int32 // số push đang chạy
		lastFlushStart time.Time    // mốc tính delayBetweenBatch (từ start)
		lastIdleLog    time.Time
		wakeup         = make(chan struct{}, 1) // signal main loop chạy lại sớm
		hardStop       bool
	)

	latestCfg := func() UploadSiteConfig { return a.LoadUploadSiteConfig() }
	signalWake := func() {
		select {
		case wakeup <- struct{}{}:
		default:
		}
	}

	// pushAsync chạy 1 push trong goroutine. Tăng inFlight, push xong giải phóng slot.
	// Trên error: enqueue lại vào retryQueue với backoff.
	pushAsync := func(accs []string, attempts int) {
		inFlight.Add(1)
		go func() {
			defer func() {
				inFlight.Add(-1)
				signalWake()
			}()
			c := latestCfg()
			filter := "1"
			if !c.FilterDuplicate {
				filter = "0"
			}
			// ctx = runUploadSite ctx (run-scoped). Stop sẽ cancel push đang chạy
			// thay vì để TCP keep-alive treo đến hết 180s timeout.
			n, err := pushToBanclone(ctx, c.Code, c.ApiKey, filter, accs)
			if err != nil {
				newAttempts := attempts + 1
				a.updateUploadStats(func(s *UploadStats) {
					s.TotalFailed += len(accs)
					s.ConsecutiveFailures = newAttempts
					s.LastError = err.Error()
					s.LastErrorAt = time.Now().Format(time.RFC3339)
					s.PendingCount = int(uploadPendingInMem.Load())
				})

				// Skip retry nếu run đã Stop — tránh enqueue vào queue mà main loop
				// không pop nữa → retryQueue grow trong memory cho đến app close.
				// Pending acc vẫn còn trong pending.txt → lần next start sẽ load lại.
				if ctx.Err() != nil {
					slog.Info("uploadSite: skip retry — ctx cancelled", "accs", len(accs))
					a.emitUploadLogL("", fmt.Sprintf("⏹ Stop giữa chừng — bỏ qua retry %d acc (sẽ load lại lần sau)",
						len(accs)), 0, logLevelInfo)
					return
				}

				// Cap MAX retry — sau N lần vẫn fail thì drop batch, không retry tiếp.
				// Site die hẳn thì retry mãi vô ích + grow RAM. Pending.txt giữ acc đó để
				// lần start sau load lại từ disk.
				if newAttempts >= uploadMaxRetryAttempts {
					slog.Warn("uploadSite: drop batch sau max retry",
						"attempts", newAttempts, "accs", len(accs), "err", err)
					a.emitUploadLogL("", fmt.Sprintf("⛔ Bỏ qua %d acc sau %d lần retry (site die hoặc lỗi cứng): %v",
						len(accs), newAttempts, err), 0, logLevelError)
					return
				}

				delay := uploadBackoff(newAttempts)
				slog.Warn("uploadSite: push lỗi", "err", err, "attempts", newAttempts, "accs", len(accs))
				a.emitUploadLogL("", fmt.Sprintf("❌ Push lỗi (%d acc, lần %d/%d): %v — retry sau %s",
					len(accs), newAttempts, uploadMaxRetryAttempts, err, delay), 0, logLevelError)
				// Push lại vào retryQueue. Cần lock vì chạy trong goroutine khác main loop.
				a.uploadRetryMu.Lock()
				// Cap hard size — nếu queue vượt, drop oldest entry trước khi append.
				// Defense in-depth: kể cả khi site treo rất lâu, retryQueue không grow vô tận.
				if len(a.uploadRetryQueue) >= uploadMaxRetryQueueLen {
					dropped := a.uploadRetryQueue[0]
					a.uploadRetryQueue = a.uploadRetryQueue[1:]
					slog.Warn("uploadSite: retry queue full, drop oldest",
						"droppedAccs", len(dropped.accs), "queueCap", uploadMaxRetryQueueLen)
				}
				a.uploadRetryQueue = append(a.uploadRetryQueue, &retryItem{
					accs:     accs,
					retryAt:  time.Now().Add(delay),
					attempts: newAttempts,
				})
				a.uploadRetryMu.Unlock()
				return
			}
			// Success
			uploadPendingInMem.Add(-int32(len(accs)))
			if uploadPendingInMem.Load() < 0 {
				uploadPendingInMem.Store(0)
			}
			a.updateUploadStats(func(s *UploadStats) {
				s.TotalUploaded += n
				s.ConsecutiveFailures = 0
				s.LastUploadAt = time.Now().Format(time.RFC3339)
				s.PendingCount = int(uploadPendingInMem.Load())
			})
			a.emitUploadLogL("", fmt.Sprintf("✅ Tải lên thành công: %d accounts", n), n, logLevelOk)
		}()
	}

	const uploadCheckUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36"

	// uploadCheckLine check live/die bằng UID (picture endpoint) trước khi add vào batch.
	// Dùng CheckLiveDieByPicture thay vì token — tránh false positive khi token bị checkpoint
	// nhưng account vẫn live (picture endpoint không đòi token).
	uploadCheckLine := func(line string) {
		uid := extractUIDFromLine(line)
		if uid == "" {
			batch = append(batch, line)
			return
		}
		checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		result := verifybase.CheckLiveDieByPicture(checkCtx, uploadCheckUA, uid)
		if result == "Die" {
			a.emitUploadLogL(uid, "⛔ Bỏ qua upload — UID die", 0, logLevelWarn)
			a.updateUploadStats(func(s *UploadStats) { s.TotalFailed++ })
			return
		}
		batch = append(batch, line)
	}

	// drainBatch hút thêm acc từ uploadCh vào batch local (non-blocking).
	drainBatch := func() {
		for {
			select {
			case more, ok := <-a.uploadCh:
				if !ok {
					return
				}
				uploadCheckLine(more)
			default:
				return
			}
		}
	}

	// pickRetryReady lấy 1 retry item đã đến hạn (retryAt ≤ now). Nil nếu không có.
	// Lock retryMu khi truy cập queue.
	pickRetryReady := func() *retryItem {
		now := time.Now()
		a.uploadRetryMu.Lock()
		defer a.uploadRetryMu.Unlock()
		for i, item := range a.uploadRetryQueue {
			if !item.retryAt.After(now) {
				// Pop item này — giữ thứ tự còn lại.
				a.uploadRetryQueue = append(a.uploadRetryQueue[:i], a.uploadRetryQueue[i+1:]...)
				return item
			}
		}
		return nil
	}

	retryQueueLen := func() int {
		a.uploadRetryMu.Lock()
		defer a.uploadRetryMu.Unlock()
		return len(a.uploadRetryQueue)
	}

	// nextRetryAt — thời điểm sớm nhất 1 retry item sẽ đến hạn (cho timer).
	nextRetryAt := func() time.Time {
		a.uploadRetryMu.Lock()
		defer a.uploadRetryMu.Unlock()
		var earliest time.Time
		for _, item := range a.uploadRetryQueue {
			if earliest.IsZero() || item.retryAt.Before(earliest) {
				earliest = item.retryAt
			}
		}
		return earliest
	}

	// kickPushes thử khởi động các push mới (retry trước, batch sau).
	// Tôn trọng: maxBatch, maxConcurrent, delayBetween (từ send-start).
	kickPushes := func() {
		c := latestCfg()
		maxBatch := c.AccPerBatch
		if maxBatch <= 0 {
			maxBatch = 100
		}
		delayBetween := time.Duration(c.DelayBetweenBatchSec) * time.Second

		for inFlight.Load() < int32(uploadMaxConcurrent) {
			// Gate delayBetween — TỪ LÚC LASTFLUSHSTART, không phải sau khi push xong.
			if !lastFlushStart.IsZero() && delayBetween > 0 {
				if elapsed := time.Since(lastFlushStart); elapsed < delayBetween {
					return // chưa đủ delay → để tick sau wake lại
				}
			}

			// Ưu tiên retry trước (FIFO theo retryAt).
			if item := pickRetryReady(); item != nil {
				lastFlushStart = time.Now()
				a.emitUploadLogL("", fmt.Sprintf("🔁 Retry %d acc (lần %d)", len(item.accs), item.attempts), 0, logLevelInfo)
				pushAsync(item.accs, item.attempts)
				continue
			}

			// Batch mới — cap ≤ maxBatch.
			drainBatch()
			if len(batch) == 0 {
				return
			}
			n := len(batch)
			if n > maxBatch {
				n = maxBatch
			}
			toSend := batch[:n]
			batch = batch[n:]

			lastFlushStart = time.Now()
			a.emitUploadLogL("", fmt.Sprintf("⏫ Đang tải lên %d accounts...", n), 0, logLevelInfo)
			pushAsync(toSend, 0)
		}
	}

	// Soft-stop exit: stopping && batch=∅ && retryQueue=∅ && inFlight=0 && channel=∅.
	allDrained := func() bool {
		if !a.uploadStopping.Load() {
			return false
		}
		if len(batch) > 0 || retryQueueLen() > 0 || inFlight.Load() > 0 {
			return false
		}
		select {
		case line, ok := <-a.uploadCh:
			if ok {
				uploadCheckLine(line)
			}
			return false
		default:
			return true
		}
	}

	for {
		if hardStop {
			return
		}
		drainBatch()
		kickPushes()
		if allDrained() {
			return
		}

		// Tính thời gian chờ: min(checkInterval, time-to-next-retry, time-to-delay-end).
		waitDur := checkInterval
		c := latestCfg()
		if delay := time.Duration(c.DelayBetweenBatchSec) * time.Second; delay > 0 && !lastFlushStart.IsZero() {
			if rem := delay - time.Since(lastFlushStart); rem > 0 && rem < waitDur {
				waitDur = rem
			}
		}
		if t := nextRetryAt(); !t.IsZero() {
			if rem := time.Until(t); rem > 0 && rem < waitDur {
				waitDur = rem
			}
		}
		if waitDur < 100*time.Millisecond {
			waitDur = 100 * time.Millisecond
		}

		select {
		case <-ctx.Done():
			// Hard-stop từ StopUploadSite timeout. Để các push đang chạy hoàn tất background.
			a.uploadStopping.Store(true)
			hardStop = true
			return

		case <-wakeup:
			// goroutine push xong → vòng lại để kick push mới.

		case line, ok := <-a.uploadCh:
			if !ok {
				a.uploadStopping.Store(true)
				return
			}
			uploadCheckLine(line)

		case <-time.After(waitDur):
			// Idle log — không có gì làm.
			if len(batch) == 0 && retryQueueLen() == 0 && inFlight.Load() == 0 {
				if time.Since(lastIdleLog) >= 60*time.Second {
					a.emitUploadLogL("", fmt.Sprintf("⏳ Chờ acc... (check mỗi %ds)", c.DelayCheckSec), 0, logLevelInfo)
					lastIdleLog = time.Now()
				}
			}
		}
	}
}

// readNewLinesFrom đọc các dòng mới trong file kể từ byte offset.
// Trả về (lines, newOffset, error). newOffset chỉ tăng khi có dòng được đọc thành công.
func readNewLinesFrom(filePath string, fromByte int64) ([]string, int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fromByte, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fromByte, err
	}
	if info.Size() <= fromByte {
		return nil, fromByte, nil
	}

	if _, err := f.Seek(fromByte, io.SeekStart); err != nil {
		return nil, fromByte, err
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fromByte, err
	}

	raw := strings.Split(string(b), "\n")
	var lines []string
	for _, l := range raw {
		l = strings.TrimSpace(l)
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines, fromByte + int64(len(b)), nil
}
