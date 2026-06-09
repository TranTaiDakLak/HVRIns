// Package proxy — Transport pool để reuse http.Transport per-proxy.
// Giảm RAM leak khi chạy 24/7 với rotating proxy: thay vì tạo Transport mới mỗi CreateClient()
// (mỗi Transport giữ TLS cache + idle TCP buffers), pool reuse Transport theo key = proxy string.
//
// Eviction: LRU cap 200 entries (xem `maxTransportPoolSize`). Transport idle > 2 phút bị evict
// (xem `transportIdleTTL`). Cleanup goroutine quét pool mỗi 90s (xem `cleanupInterval`).
// Mỗi eviction gọi CloseIdleConnections() để free TCP connections.
//
// Rollback: set env TRANSPORT_POOL_DISABLED=1 → fallback behavior cũ (tạo Transport mỗi call).
package proxy

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"sync"
	"syscall"
	"time"
)

const (
	// maxTransportPoolSize cap số Transport đồng thời trong pool. Mỗi Transport ~200KB buffer
	// → 500 * 200KB = ~100 MB tối đa cho pool. Cân bằng giữa reuse hit rate và RAM.
	maxTransportPoolSize = 200

	// transportIdleTTL thời gian Transport không được Get sau đó sẽ bị evict + close idle conns.
	// 2 phút = đủ cho proxy session nhưng giải phóng nhanh hơn để giảm RAM 24/7.
	transportIdleTTL = 2 * time.Minute

	// cleanupInterval tần suất scan evict idle transports.
	cleanupInterval = 90 * time.Second
)

// pooledTransport bọc http.Transport kèm metadata cho LRU.
type pooledTransport struct {
	transport *http.Transport
	lastUsed  time.Time
}

// transportPool là singleton pool cho toàn app.
// Key = full proxy string (bao gồm session ID) → tránh auth conflict khi share Transport.
var transportPool = struct {
	mu       sync.Mutex
	entries  map[string]*pooledTransport
	disabled bool
	started  bool // cleanup goroutine đã start chưa (lazy init)
}{
	entries: make(map[string]*pooledTransport, maxTransportPoolSize),
}

// init: pool ENABLED mặc định.
// Disable qua env TRANSPORT_POOL_DISABLED=1 nếu cần rollback (mỗi CreateClient sẽ
// tạo Transport mới — không recommended, chỉ dùng để debug pool issues).
// Cấu hình transport (MaxConnsPerHost, IdleConnTimeout, ...) ở `newHTTPTransport`.
func init() {
	transportPool.disabled = os.Getenv("TRANSPORT_POOL_DISABLED") == "1"
}

// getOrCreateTransport trả về Transport cho proxyStr, reuse nếu đã có trong pool.
// Nếu pool disabled (env flag), luôn tạo Transport mới.
// Thread-safe: lock mutex khi access map.
func getOrCreateTransport(proxyStr string) *http.Transport {
	if transportPool.disabled {
		return newHTTPTransport(proxyStr)
	}

	// Lazy start cleanup goroutine ở lần gọi đầu (tránh start khi package init chưa có use case)
	transportPool.mu.Lock()
	if !transportPool.started {
		transportPool.started = true
		go transportPoolCleanupLoop()
	}

	// Reuse nếu đã có
	if pt, ok := transportPool.entries[proxyStr]; ok {
		pt.lastUsed = time.Now()
		transportPool.mu.Unlock()
		return pt.transport
	}

	// Cap check → evict oldest nếu pool full
	if len(transportPool.entries) >= maxTransportPoolSize {
		evictOldestLocked()
	}

	// Tạo mới + lưu vào pool
	t := newHTTPTransport(proxyStr)
	transportPool.entries[proxyStr] = &pooledTransport{
		transport: t,
		lastUsed:  time.Now(),
	}
	transportPool.mu.Unlock()
	return t
}

// newHTTPTransport tạo http.Transport mới với cấu hình chung (TLS, idle conns, timeouts).
// Được gọi từ getOrCreateTransport khi cache miss, hoặc trực tiếp nếu pool disabled.
//
// MaxConnsPerHost=100: cho phép 100 goroutine concurrent dùng cùng Transport mà không bị
// block chờ connection → tránh OTP timeout. TCP keepalive 30s giảm tích lũy TIME_WAIT
// trên Windows — nguyên nhân chính của lỗi WSAEADDRINUSE (port exhaustion).
func newHTTPTransport(proxyStr string) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		// SO_LINGER=0: close gửi RST thay vì FIN → skip TIME_WAIT hoàn toàn.
		// Fix WSAEADDRINUSE (port exhaustion) khi chạy nhiều luồng concurrent trên Windows.
		Control: func(_, _ string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				_ = syscall.SetsockoptLinger(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_LINGER,
					&syscall.Linger{Onoff: 1, Linger: 0})
			})
		},
	}
	t := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		MaxIdleConnsPerHost:   16,  // cache 16 idle conn/host — reuse nhiều hơn, giảm TCP churn → giảm WSAEADDRINUSE
		MaxConnsPerHost:       100, // giữ 100 để goroutines không bị block chờ slot
		MaxIdleConns:          128, // tổng idle cap toàn transport (tăng theo MaxIdleConnsPerHost)
		IdleConnTimeout:       20 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		DisableKeepAlives:     false,
	}
	if s := proxyStr; s != "" {
		if u := parseProxy(s); u != nil {
			t.Proxy = http.ProxyURL(u)
		}
	}
	return t
}

// evictOldestLocked (caller phải giữ mu) — tìm entry có lastUsed cũ nhất, close + xóa.
// O(N) scan — chỉ chạy khi pool đạt cap, không phải hot path thường xuyên.
func evictOldestLocked() {
	var oldestKey string
	var oldestTime time.Time
	for k, pt := range transportPool.entries {
		if oldestKey == "" || pt.lastUsed.Before(oldestTime) {
			oldestKey = k
			oldestTime = pt.lastUsed
		}
	}
	if oldestKey != "" {
		transportPool.entries[oldestKey].transport.CloseIdleConnections()
		delete(transportPool.entries, oldestKey)
	}
}

// transportPoolCleanupLoop chạy background mỗi cleanupInterval,
// evict các Transport idle quá transportIdleTTL → free idle TCP conns.
// Exit khi process shutdown (goroutine leak không vấn đề vì app-wide singleton).
func transportPoolCleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		transportPoolCleanupIdle()
	}
}

// transportPoolCleanupIdle scan toàn pool, evict entries không dùng quá TTL.
func transportPoolCleanupIdle() {
	cutoff := time.Now().Add(-transportIdleTTL)
	transportPool.mu.Lock()
	defer transportPool.mu.Unlock()
	for k, pt := range transportPool.entries {
		if pt.lastUsed.Before(cutoff) {
			pt.transport.CloseIdleConnections()
			delete(transportPool.entries, k)
		}
	}
}

// TransportPoolStats trả về kích thước pool hiện tại — dùng cho monitoring/debug.
// Chỉ đọc, thread-safe.
func TransportPoolStats() int {
	transportPool.mu.Lock()
	defer transportPool.mu.Unlock()
	return len(transportPool.entries)
}
