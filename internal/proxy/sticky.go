// sticky.go — Sticky proxy per worker slot.
// Port C# MainFormUISettings.KeepIPSuccess: sau khi 1 account thành công,
// worker slot giữ nguyên proxy + sessionID cho account kế. Fail → release.
package proxy

import (
	"context"
	"sync"
)

// AcquireFn raw acquire — trả về (proxyStr, releaseToPool, err)
type AcquireFn func(ctx context.Context) (string, func(), error)

// StickyManager bọc 1 AcquireFn với sticky-on-success behavior per worker slot.
//
// Vòng đời 1 entry:
//  1. worker gọi Acquire(ctx, workerID) lần đầu → AcquireFn raw → pin nếu enable
//  2. account chạy xong → worker gọi Release(workerID, success)
//     - success + enable → entry GIỮ NGUYÊN (no-op release)
//     - fail / !enable → call raw release + xóa entry
//  3. account kế dùng lại entry (nếu còn) hoặc acquire fresh
//  4. ReleaseAll() khi batch kết thúc → giải phóng toàn bộ entry còn pin
//
// Khi Enabled=false: hành vi giống acquire thường — không pin, release ngay.
type StickyManager struct {
	enabled bool
	acquire AcquireFn

	mu      sync.Mutex
	entries map[int]stickyEntry // workerID → entry
}

type stickyEntry struct {
	proxy   string
	release func() // raw release về pool
}

// NewStickyManager tạo wrapper. enabled=false → Acquire/Release hoạt động như pass-through.
func NewStickyManager(enabled bool, acquire AcquireFn) *StickyManager {
	return &StickyManager{
		enabled: enabled,
		acquire: acquire,
		entries: make(map[int]stickyEntry),
	}
}

// Acquire trả về (proxyStr, releaseFn) cho worker slot. Nếu đã pin → dùng lại
// entry cũ; releaseFn(success) quyết định giữ hay thả dựa theo KeepIPSuccess.
// Phù hợp signature runner.RunConfig.AcquireProxy.
func (m *StickyManager) Acquire(ctx context.Context, workerID int) (string, func(success bool)) {
	if m.enabled {
		m.mu.Lock()
		if e, ok := m.entries[workerID]; ok {
			m.mu.Unlock()
			return e.proxy, func(success bool) { m.release(workerID, success) }
		}
		m.mu.Unlock()
	}

	p, rel, err := m.acquire(ctx)
	if err != nil {
		return "", func(bool) {}
	}

	m.mu.Lock()
	m.entries[workerID] = stickyEntry{proxy: p, release: rel}
	m.mu.Unlock()

	return p, func(success bool) { m.release(workerID, success) }
}

// release — internal: success=true + enabled → giữ pin; ngược lại xóa + release raw.
func (m *StickyManager) release(workerID int, success bool) {
	m.mu.Lock()
	e, ok := m.entries[workerID]
	if !ok {
		m.mu.Unlock()
		return
	}
	keep := m.enabled && success
	if keep {
		m.mu.Unlock()
		return // giữ nguyên entry cho lần Acquire kế
	}
	delete(m.entries, workerID)
	m.mu.Unlock()

	if e.release != nil {
		e.release()
	}
}

// ReleaseAll giải phóng toàn bộ entry còn pin — gọi khi batch kết thúc.
func (m *StickyManager) ReleaseAll() {
	m.mu.Lock()
	entries := m.entries
	m.entries = make(map[int]stickyEntry)
	m.mu.Unlock()

	for _, e := range entries {
		if e.release != nil {
			e.release()
		}
	}
}
