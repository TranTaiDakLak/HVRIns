package main

import (
	"context"
	"fmt"
	"log/slog"
	goruntime "runtime"
	"runtime/debug"
	"time"

	ioshttpreg "HVRIns/internal/instagram/register/ioshttp"
	webandroidreg "HVRIns/internal/instagram/register/webandroid"
	"HVRIns/internal/proxy"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) runMemoryMaintenance(ctx context.Context) {
	// Aggressive cadence cho 24/7: GC 1 phút, watchdog 20 giây.
	// Temp mail workload tạo ~2-3 MB/s allocation → GC 1min đủ catch up.
	gcTicker := time.NewTicker(1 * time.Minute)
	defer gcTicker.Stop()
	watchdogTicker := time.NewTicker(20 * time.Second)
	defer watchdogTicker.Stop()
	// Session pool cleanup: mỗi 10 phút close idle TCP/TLS conn của tất cả
	// session trong iOS/WebAndroid pool (không remove session). Giải phóng
	// native buffer NGOÀI Go heap — GC không động tới nên cần force close.
	// Quan trọng cho run dài (12h+): nếu không clean, idle conn tích lũy
	// → RSS tăng dần dù Go heap thấp.
	sessionPoolTicker := time.NewTicker(10 * time.Minute)
	defer sessionPoolTicker.Stop()

	// Cooldown auto cleanup — tránh trigger liên tục khi RSS dao động quanh threshold
	var lastAutoCleanup time.Time
	const autoCleanupCooldown = 3 * time.Minute

	// Adaptive GOGC theo run state:
	//   active (run đang chạy): GOGC=50 — GC aggressive, ngăn drift
	//   idle (không run): GOGC=200 — GC ít hơn, tiết kiệm CPU GC khi app idle
	// Swap chỉ khi state thực sự đổi để tránh syscall không cần thiết.
	const (
		gogcActive = 50
		gogcIdle   = 200
	)
	currentGOGC := gogcActive // startup giả định active để conservative

	for {
		select {
		case <-ctx.Done():
			return
		case <-gcTicker.C:
			var before goruntime.MemStats
			goruntime.ReadMemStats(&before)
			goruntime.GC()
			debug.FreeOSMemory()
			var after goruntime.MemStats
			goruntime.ReadMemStats(&after)
			freed := int64(before.HeapInuse) - int64(after.HeapInuse)
			slog.Info("memory maintenance gc",
				"heap_before_mb", before.HeapInuse/1024/1024,
				"heap_after_mb", after.HeapInuse/1024/1024,
				"freed_mb", freed/1024/1024,
				"rss_mb", getProcessMemoryMB(),
				"transport_pool", proxy.TransportPoolStats(),
				"goroutines", goruntime.NumGoroutine())
		case <-watchdogTicker.C:
			// Adaptive GOGC: khi run active → GC aggressive (50); idle → GC nhẹ (200)
			a.verifyMu.Lock()
			verRunning := a.isRunning
			a.verifyMu.Unlock()
			a.registerMu.Lock()
			regRunning := a.registerCancel != nil
			a.registerMu.Unlock()
			anyRunning := verRunning || regRunning
			desiredGOGC := gogcIdle
			if anyRunning {
				desiredGOGC = gogcActive
			}
			if desiredGOGC != currentGOGC {
				debug.SetGCPercent(desiredGOGC)
				currentGOGC = desiredGOGC
				slog.Debug("adaptive GOGC swap",
					"new_gogc", desiredGOGC,
					"reason", map[bool]string{true: "active", false: "idle"}[anyRunning])
			}

			var m goruntime.MemStats
			goruntime.ReadMemStats(&m)
			heapMB := m.HeapInuse / 1024 / 1024
			rssMB := getProcessMemoryMB() // process RSS thật (gồm WebView2 + native)
			numGoroutine := goruntime.NumGoroutine()

			// Goroutine leak detection: nếu > 2000 goroutine thì gần chắc là leak
			// (max case: 300 luồng × 3-4 goroutine mỗi thread = ~1200). Log warning.
			if numGoroutine > 2000 {
				slog.Warn("goroutine count high — possible leak",
					"goroutines", numGoroutine,
					"heap_mb", heapMB)
			}

			// Proactive GC: nếu heap > 600 MB → force GC silent (không notify).
			// App bình thường 200-500 MB cho 400 luồng; vượt 600 MB = drift, cần cleanup.
			if heapMB > 600 {
				goruntime.GC()
				debug.FreeOSMemory()
				slog.Debug("proactive gc triggered", "heap_mb", heapMB)
			}

			// AUTO CLEANUP — khi RSS process vượt threshold động → tự gọi cleanup
			// (close idle conn + GC + FreeOSMemory). Tránh phụ thuộc user bấm
			// nút "Dọn RAM" thủ công. Cooldown 3 phút để không thrash GC.
			//
			// Threshold dùng RSS thay vì heap vì heap có thể thấp nhưng native
			// TLS/WebView2 chiếm RSS — heap-based watchdog miss case này.
			//
			// 1500 MB: cleanup nhẹ (chỉ idle conn + GC)
			// 2000 MB: cleanup AGGRESSIVE (force GC 2 lần, log warn)
			if rssMB >= 1500 && time.Since(lastAutoCleanup) > autoCleanupCooldown {
				aggressive := rssMB >= 2000
				closedIOS := 0
				closedAndroid := 0
				if ioshttpreg.SharedSessionPool != nil {
					closedIOS = ioshttpreg.SharedSessionPool.CloseIdleConnsAll()
				}
				if webandroidreg.SharedSessionPool != nil {
					closedAndroid = webandroidreg.SharedSessionPool.CloseIdleConnsAll()
				}
				bancloneTransport.CloseIdleConnections()

				goruntime.GC()
				debug.FreeOSMemory()
				if aggressive {
					// 2nd GC pass — first GC moves objects to free list, second sweeps
					goruntime.GC()
					debug.FreeOSMemory()
				}

				rssAfter := getProcessMemoryMB()
				slog.Info("auto cleanup triggered",
					"reason", "rss_threshold",
					"rss_before_mb", rssMB,
					"rss_after_mb", rssAfter,
					"freed_mb", rssMB-rssAfter,
					"aggressive", aggressive,
					"ios_closed", closedIOS,
					"android_closed", closedAndroid,
					"goroutines", numGoroutine)
				lastAutoCleanup = time.Now()
			}

			// Warning threshold — cảnh báo user restart sau batch nếu vượt 1.5 GB heap.
			if heapMB > 1500 {
				slog.Warn("memory watchdog high",
					"heap_mb", heapMB,
					"sys_mb", m.Sys/1024/1024,
					"rss_mb", rssMB,
					"goroutines", numGoroutine)
				if a.ctx != nil {
					runtime.EventsEmit(a.ctx, "system:memory-warning", map[string]interface{}{
						"heapMB": heapMB,
						"rssMB":  rssMB,
						"msg":    fmt.Sprintf("⚠️ RAM app ở mức cao (heap %d MB / RSS %.0f MB) — cân nhắc restart sau batch hiện tại", heapMB, rssMB),
					})
				}
			}
		case <-sessionPoolTicker.C:
			// Periodic close idle conn trên session pool — không remove session,
			// chỉ giải phóng idle TCP/TLS native buffer. Chạy bất kể có register
			// đang chạy hay không (idempotent: nếu pool rỗng, return 0).
			//
			// Lifecycle ownership (Task 3): đọc global pointer KHÔNG có owning;
			// chỉ acts on pool nếu non-nil. Sau khi run kết thúc, runResources.Cleanup
			// đã CloseAll + nil global → ticker thấy nil → skip. Trong khi run đang
			// chạy, ticker thấy global = pool của run → CloseIdleConnsAll an toàn
			// (chỉ đóng IDLE conn, không abort active request, không drop session).
			closedIOS := 0
			closedAndroid := 0
			if ioshttpreg.SharedSessionPool != nil {
				closedIOS = ioshttpreg.SharedSessionPool.CloseIdleConnsAll()
			}
			if webandroidreg.SharedSessionPool != nil {
				closedAndroid = webandroidreg.SharedSessionPool.CloseIdleConnsAll()
			}
			// Close idle conn của banclone push transport — sau nhiều lần push
			// idle TCP/TLS có thể tích tụ trong transport pool.
			bancloneTransport.CloseIdleConnections()
			if closedIOS+closedAndroid > 0 {
				slog.Info("session pool idle cleanup",
					"ios_sessions", closedIOS,
					"android_sessions", closedAndroid)
			}
		}
	}
}

// DebugMemory trả về snapshot resource hiện tại — frontend có thể call để xem
// trạng thái RAM/goroutine/session pool. Dùng cho debug RAM leak khi chạy lâu.
//
// Để debug sâu hơn (heap profile / goroutine dump): set env DEBUG_PPROF=1 trước
// khi start app → pprof endpoint mở tại http://127.0.0.1:6060/debug/pprof/.
// Xem `debug.go` cho chi tiết enable/disable. KHÔNG bật mặc định production.
func (a *App) DebugMemory() map[string]interface{} {
	var m goruntime.MemStats
	goruntime.ReadMemStats(&m)

	iosSessions := 0
	if ioshttpreg.SharedSessionPool != nil {
		iosSessions = ioshttpreg.SharedSessionPool.Size()
	}
	androidSessions := 0
	if webandroidreg.SharedSessionPool != nil {
		androidSessions = webandroidreg.SharedSessionPool.Size()
	}

	a.accountsMu.RLock()
	accCount := len(a.accounts)
	a.accountsMu.RUnlock()

	// Run state — frontend dùng để hiển thị "Đang chạy/Đang dừng/Idle"
	a.verifyMu.Lock()
	verRunning := a.isRunning
	a.verifyMu.Unlock()
	regState := runState(a.registerState.Load())
	regRunning := regState == runStateRunning
	regStopping := regState == runStateStopping

	// LastGC — m.LastGC là nanoseconds since epoch; convert thành seconds-ago.
	// 0 nếu chưa có GC nào (app vừa start).
	lastGCAgoSec := uint64(0)
	if m.LastGC > 0 {
		nowNs := uint64(time.Now().UnixNano())
		if nowNs > m.LastGC {
			lastGCAgoSec = (nowNs - m.LastGC) / 1_000_000_000
		}
	}
	// LastGC pause: m.PauseNs là circular buffer 256 entries; index = (NumGC+255) % 256.
	lastPauseMs := uint64(0)
	if m.NumGC > 0 {
		lastPauseMs = m.PauseNs[(m.NumGC+255)%256] / 1_000_000
	}

	return map[string]interface{}{
		// Go runtime memory stats — granular cho Task 7 (heap idle/released = giá trị
		// có thể trả về OS qua FreeOSMemory; tăng = leak thật trong heap).
		"allocMB":        m.Alloc / 1024 / 1024,        // currently allocated heap (live objects)
		"heapAllocMB":    m.HeapAlloc / 1024 / 1024,    // bytes allocated from heap (== Alloc)
		"heapInuseMB":    m.HeapInuse / 1024 / 1024,    // bytes in in-use spans (incl free)
		"heapIdleMB":     m.HeapIdle / 1024 / 1024,     // bytes in idle spans (chưa trả về OS)
		"heapReleasedMB": m.HeapReleased / 1024 / 1024, // bytes đã trả về OS (subset của HeapIdle)
		"heapSysMB":      m.HeapSys / 1024 / 1024,      // total heap obtained from OS
		"sysMB":          m.Sys / 1024 / 1024,          // total bytes obtained from OS (heap+stack+other)
		"stackInuseMB":   m.StackInuse / 1024 / 1024,
		"numGC":          m.NumGC,                    // GC cycles run
		"numForcedGC":    m.NumForcedGC,              // forced GC count (manual runtime.GC calls)
		"lastGCAgoSec":   lastGCAgoSec,               // giây từ lần GC gần nhất (0 = vừa fire)
		"lastPauseMs":    lastPauseMs,                // pause của lần GC gần nhất
		"pauseTotalMs":   m.PauseTotalNs / 1_000_000, // tổng thời gian pause vì GC
		"gcCpuFraction":  m.GCCPUFraction,            // % CPU dành cho GC (0..1)
		"mallocs":        m.Mallocs,                  // tổng allocations từ start
		"frees":          m.Frees,                    // tổng frees từ start
		// Process-level (OS view) — khác Go heap, gồm WebView2 + native buffers
		"rssMB":  getProcessMemoryMB(), // RSS thực process
		"cpuPct": getProcessCPUPercent(),
		// Goroutine + concurrency
		"goroutines": goruntime.NumGoroutine(),
		"gomaxprocs": goruntime.GOMAXPROCS(0),
		// App-specific resources (ngoài Go heap — native HTTP/TLS buffers)
		"accountsInStore": accCount,
		"iosSessions":     iosSessions,
		"androidSessions": androidSessions,
		"transportPool":   proxy.TransportPoolStats(),
		"uploadPending":   uploadPendingInMem.Load(),
		// Run state — phục vụ cả frontend status bar lẫn debug
		"registerRunning":  regRunning,
		"registerStopping": regStopping,
		"verifyRunning":    verRunning,
		"verifyStopping":   a.verifyStopping.Load(),
		"uploadStopping":   a.uploadStopping.Load(),
		// Debug instrumentation flags — frontend có thểbạn đã hiển thị badge "PPROF ON"
		"pprofEnabled": debugPProfEnabled(),
	}
}

// ForceMemoryCleanup — frontend gọi để force cleanup ngay (idle conn + GC + FreeOS).
// Dùng khi user thấy RAM cao và muốn cleanup không cần restart app.
//
// Lifecycle ownership (Task 3): đọc global pointer chỉ để CloseIdleConnsAll (KHÔNG
// CloseAll/drop session). Run đang chạy → đóng idle conn của run đó (an toàn). Run
// đã end → runResources.Cleanup đã nil global → đọc nil → skip. KHÔNG bao giờ thay
// thế lifecycle Cleanup của run.
func (a *App) ForceMemoryCleanup() map[string]interface{} {
	closedIOS := 0
	closedAndroid := 0
	if ioshttpreg.SharedSessionPool != nil {
		closedIOS = ioshttpreg.SharedSessionPool.CloseIdleConnsAll()
	}
	if webandroidreg.SharedSessionPool != nil {
		closedAndroid = webandroidreg.SharedSessionPool.CloseIdleConnsAll()
	}
	// Close idle connections của banclone push transport — sau hàng nghìn lần push,
	// idle TCP/TLS state có thể tích tụ.
	bancloneTransport.CloseIdleConnections()

	var before goruntime.MemStats
	goruntime.ReadMemStats(&before)
	goruntime.GC()
	debug.FreeOSMemory()
	var after goruntime.MemStats
	goruntime.ReadMemStats(&after)

	freedMB := int64(before.HeapInuse-after.HeapInuse) / 1024 / 1024
	slog.Info("manual memory cleanup",
		"ios_sessions", closedIOS,
		"android_sessions", closedAndroid,
		"heap_freed_mb", freedMB,
		"heap_after_mb", after.HeapInuse/1024/1024)

	return map[string]interface{}{
		"iosSessionsClosed":     closedIOS,
		"androidSessionsClosed": closedAndroid,
		"heapBeforeMB":          before.HeapInuse / 1024 / 1024,
		"heapAfterMB":           after.HeapInuse / 1024 / 1024,
		"freedMB":               freedMB,
	}
}
