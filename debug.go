// debug.go — Instrumentation cho RAM/CPU/goroutine debugging.
//
// Hai feature opt-in qua environment variable:
//
//	DEBUG_PPROF=1
//	  Enable Go pprof HTTP endpoint tại 127.0.0.1:6060.
//	  Truy cập: http://127.0.0.1:6060/debug/pprof/{heap,goroutine,profile,...}
//	  CHỈ LISTEN LOCALHOST — không expose ra network.
//	  Sample command từ máy khác (tunnel/SSH): không thể truy cập trực tiếp.
//
//	DEBUG_SNAPSHOT=1
//	  Log snapshot RAM/goroutine/session pool mỗi 1 phút vào structured log.
//	  Dùng để theo dõi drift dài hạn không cần giữ frontend mở.
//	  Có thể combine với DEBUG_PPROF.
//
// Default (không env var): cả 2 feature TẮT, không tạo goroutine/listener thừa,
// production builds không bị ảnh hưởng performance.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	_ "net/http/pprof" // side-effect: register handlers vào DefaultServeMux
	"os"
	goruntime "runtime"
	"strconv"
	"strings"
	"time"
)

const (
	debugPProfAddr        = "127.0.0.1:6060"
	debugSnapshotInterval = 1 * time.Minute
)

// computeMemoryLimitMB trả về soft memory limit (MB) cho GOMEMLIMIT, theo độ ưu tiên:
//  1. Env GOMEMLIMIT_MB (vd "2048") — user override
//  2. Go runtime tự đọc env GOMEMLIMIT (vd "2GiB") — return 0, không gán thêm
//  3. Auto-detect: 25% system RAM, sàn 1024 MB, cap 4096 MB
//
// Return 0 → caller không gọi SetMemoryLimit (Go runtime tự xử lý qua GOMEMLIMIT env).
func computeMemoryLimitMB() uint64 {
	// 1. User override qua env GOMEMLIMIT_MB (số nguyên)
	if v := strings.TrimSpace(os.Getenv("GOMEMLIMIT_MB")); v != "" {
		n, err := strconv.ParseUint(v, 10, 64)
		if err == nil && n >= 256 { // sàn an toàn 256 MB
			return n
		}
	}
	// 2. Nếu user đã set GOMEMLIMIT (Go runtime tự đọc) → bỏ qua, để runtime xử lý
	if strings.TrimSpace(os.Getenv("GOMEMLIMIT")) != "" {
		return 0
	}
	// 3. Auto: 25% system RAM, sàn 1GB, cap 4GB
	systemMB := getSystemMemoryMB()
	if systemMB == 0 {
		return 1024 // fallback nếu syscall lỗi
	}
	limitMB := systemMB / 4
	if limitMB < 1024 {
		limitMB = 1024
	}
	if limitMB > 4096 {
		limitMB = 4096
	}
	return limitMB
}

// debugEnabled — pprof bật khi DEBUG_PPROF=1 (case-insensitive, accept "1"/"true"/"yes").
func debugPProfEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("DEBUG_PPROF")))
	return v == "1" || v == "true" || v == "yes"
}

func debugSnapshotEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("DEBUG_SNAPSHOT")))
	return v == "1" || v == "true" || v == "yes"
}

// startDebugInstrumentation khởi động pprof + snapshot logger nếu env var được set.
// Gọi 1 lần từ App.startup. Không làm gì nếu cả 2 env đều tắt.
func debugRegBreakpointEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("DEBUG_REG_BREAKPOINT")))
	return v == "1" || v == "true" || v == "yes"
}

func debugRegBreakpointShouldStop(platform string, slotIdx int) bool {
	if !debugRegBreakpointEnabled() {
		return false
	}
	if want := strings.TrimSpace(os.Getenv("DEBUG_REG_BREAKPOINT_PLATFORM")); want != "" &&
		!strings.EqualFold(want, strings.TrimSpace(platform)) {
		return false
	}
	if want := strings.TrimSpace(os.Getenv("DEBUG_REG_BREAKPOINT_SLOT")); want != "" {
		n, err := strconv.Atoi(want)
		if err != nil || n != slotIdx {
			return false
		}
	}
	return true
}

func debugRegBreakpoint(platform string, slotIdx, threadIdx, attempt int, contact, proxy, userAgent string) {
	if !debugRegBreakpointShouldStop(platform, slotIdx) {
		return
	}
	slog.Warn("REG debug breakpoint",
		"platform", platform,
		"slot", slotIdx,
		"thread", threadIdx,
		"attempt", attempt,
		"contact", contact,
		"proxy_set", proxy != "",
		"ua_len", len(userAgent),
	)
	goruntime.Breakpoint()
}

func (a *App) startDebugInstrumentation(ctx context.Context) {
	if debugPProfEnabled() {
		startPProfServer()
	}
	if debugSnapshotEnabled() {
		go a.runDebugSnapshotLogger(ctx)
	}
}

// startPProfServer mở HTTP listener pprof trên 127.0.0.1:6060.
// Goroutine sống cùng app process (không có cancel — pprof thường để mãi mãi
// trong session debug). Listener bị panic-recover để không crash app.
func startPProfServer() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("pprof server panic", "recovered", r)
			}
		}()
		slog.Info("pprof enabled", "addr", debugPProfAddr,
			"hint", "open http://"+debugPProfAddr+"/debug/pprof/")
		// http.DefaultServeMux đã được pprof register (qua import side-effect).
		// Lưu ý: listen 127.0.0.1 (localhost only) — KHÔNG expose ra LAN/internet.
		srv := &http.Server{
			Addr:              debugPProfAddr,
			Handler:           http.DefaultServeMux,
			ReadHeaderTimeout: 10 * time.Second,
		}
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Warn("pprof server exited", "err", err)
		}
	}()
}

// runDebugSnapshotLogger ghi snapshot RAM/goroutine/session mỗi `debugSnapshotInterval`.
// Exit khi ctx cancel (app shutdown).
func (a *App) runDebugSnapshotLogger(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("debug snapshot logger panic", "recovered", r)
		}
	}()
	slog.Info("debug snapshot logger enabled", "interval", debugSnapshotInterval)
	// Log lần đầu ngay
	a.logDebugSnapshot()
	t := time.NewTicker(debugSnapshotInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			a.logDebugSnapshot()
		}
	}
}

// logDebugSnapshot ghi 1 dòng structured log với toàn bộ metrics.
// Lưu vào structured log (slog file) — KHÔNG emit lên frontend tránh spam UI.
func (a *App) logDebugSnapshot() {
	var m goruntime.MemStats
	goruntime.ReadMemStats(&m)
	rssMB := getProcessMemoryMB()
	cpuPct := getProcessCPUPercent()
	a.verifyMu.Lock()
	verRunning := a.isRunning
	a.verifyMu.Unlock()
	regState := runState(a.registerState.Load())
	slog.Info("debug snapshot",
		"rssMB", fmt.Sprintf("%.1f", rssMB),
		"cpuPct", fmt.Sprintf("%.1f", cpuPct),
		"heapAllocMB", m.HeapAlloc/1024/1024,
		"heapInuseMB", m.HeapInuse/1024/1024,
		"sysMB", m.Sys/1024/1024,
		"numGC", m.NumGC,
		"goroutines", goruntime.NumGoroutine(),
		"uploadPending", uploadPendingInMem.Load(),
		"registerRunning", regState == runStateRunning,
		"registerStopping", regState == runStateStopping,
		"verifyRunning", verRunning,
		"verifyStopping", a.verifyStopping.Load(),
	)
}
