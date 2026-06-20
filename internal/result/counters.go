// counters.go — CounterSet gộp 3 counter của 1 session + auto-save ticker.
//
// Port từ C# FMain.tmupdate_countrysuccess_Tick (mỗi 5-10s):
//   File.WriteAllText(SavePath + "\\CountrySuccess.txt", ...)
//   File.WriteAllText(SavePath + "\\FbAppVersisonSuccess.txt", ...)
//   File.WriteAllText(SavePath + "\\FbLocalesSuccess.txt", ...)
//
// Go dùng time.Ticker thay cho WinForms Timer, goroutine riêng cho save loop.
//
// Usage:
//   cs := NewCounterSet(writer)
//   cs.Start(ctx, 5*time.Second)
//   // ... worker goroutines gọi cs.Country.Incr("VN"), cs.FbAppVersion.Incr("554|918"), ...
//   cs.Stop()  // flush final + dừng ticker
package result

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// CounterSet chứa counter của 1 session và quản lý auto-save.
// Tạo bằng NewCounterSet(writer), khởi động bằng Start(ctx, interval),
// dừng bằng Stop() — Stop cũng flush lần cuối đảm bảo không mất data.
type CounterSet struct {
	writer *Writer

	// FbAppVersion track "{version}|{build}" → count (FbAppVersisonSuccess.txt).
	FbAppVersion *Counter

	mu       sync.Mutex
	cancel   context.CancelFunc
	doneCh   chan struct{}
	running  bool
}

// NewCounterSet tạo CounterSet bind với 1 Writer.
// writer có thể nil — khi đó các Flush/auto-save sẽ no-op (counter vẫn đếm bình thường).
func NewCounterSet(writer *Writer) *CounterSet {
	return &CounterSet{
		writer:       writer,
		FbAppVersion: NewCounter(),
	}
}

// Start khởi động goroutine auto-save counters mỗi interval.
// ctx: khi cancel → ticker dừng, KHÔNG flush lần cuối (dùng Stop() để flush).
// interval: khoảng cách giữa các lần save, nên >= 1s; < 1s sẽ clamp về 1s để tránh I/O dày đặc.
//
// Gọi Start 2 lần là no-op (không tạo ticker mới).
func (cs *CounterSet) Start(ctx context.Context, interval time.Duration) {
	cs.mu.Lock()
	if cs.running {
		cs.mu.Unlock()
		return
	}
	if interval < time.Second {
		interval = time.Second
	}
	ictx, cancel := context.WithCancel(ctx)
	cs.cancel = cancel
	cs.doneCh = make(chan struct{})
	cs.running = true
	cs.mu.Unlock()

	go cs.runSaveLoop(ictx, interval)
}

// Stop dừng ticker, flush counters lần cuối để không mất data mới nhất.
// Idempotent — gọi nhiều lần không sao.
func (cs *CounterSet) Stop() {
	cs.mu.Lock()
	if !cs.running {
		cs.mu.Unlock()
		return
	}
	cancel := cs.cancel
	done := cs.doneCh
	cs.running = false
	cs.cancel = nil
	cs.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done // đợi save loop exit
	}
	// Flush lần cuối
	_ = cs.Flush()
}

// Flush ghi 3 counter ra 3 file ngay lập tức (overwrite, port C# File.WriteAllText).
// Counter rỗng → bỏ qua file đó (không tạo file rỗng).
// writer nil → no-op, trả nil.
func (cs *CounterSet) Flush() error {
	if cs == nil || cs.writer == nil {
		return nil
	}
	if s := cs.FbAppVersion.DumpSorted(); s != "" {
		if err := cs.writer.Overwrite(FileFbAppVersionSuccess, s); err != nil {
			return err
		}
	}
	return nil
}

// runSaveLoop goroutine chạy ticker. Nhận ctx để cancel.
// Mỗi tick gọi Flush; lỗi được log qua slog nhưng không abort loop.
func (cs *CounterSet) runSaveLoop(ctx context.Context, interval time.Duration) {
	defer close(cs.doneCh)

	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := cs.Flush(); err != nil {
				slog.Warn("CounterSet flush", "err", err)
			}
		}
	}
}
