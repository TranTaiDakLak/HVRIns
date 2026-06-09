// shoplike.go — ShopLike proxy provider
// Mapping từ WeBM ShopLike.cs + frmFacebook.Proxy.cs (Type 7)
// API: http://proxy.shoplike.vn/Api/getNewProxy?access_token=TOKEN
package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"HVRIns/internal/httpx"
)

const shoplikeBase = "http://proxy.shoplike.vn/Api"

type shoplikeInstance struct {
	apiKey      string
	activeCount int
	mu          sync.Mutex

	// cache proxy — tất cả goroutine dùng chung key này đều dùng cùng 1 proxy
	// ShopLike cho phép dùng 1 IP trong N giây (nextChange), chỉ cần gọi API 1 lần
	cachedProxy string
	cacheExpiry time.Time
	fetching    bool // đang gọi API, các goroutine khác chờ
	fetchCond   *sync.Cond
}

// incr tăng bộ đếm số thread đang dùng instance này.
func (s *shoplikeInstance) incr() { s.mu.Lock(); s.activeCount++; s.mu.Unlock() }

// decr giảm bộ đếm số thread đang dùng instance này (không âm).
func (s *shoplikeInstance) decr() {
	s.mu.Lock()
	if s.activeCount > 0 {
		s.activeCount--
	}
	s.mu.Unlock()
}

// active trả về số thread đang dùng instance này (thread-safe).
func (s *shoplikeInstance) active() int { s.mu.Lock(); defer s.mu.Unlock(); return s.activeCount }

// getProxy trả về proxy hiện tại của key này.
// Nếu cache còn hạn → trả về ngay (không gọi API).
// Nếu cache hết hạn → gọi API 1 lần, các goroutine khác chờ kết quả.
func (s *shoplikeInstance) getProxy(ctx context.Context) (string, error) {
	s.mu.Lock()
	if s.fetchCond == nil {
		s.fetchCond = sync.NewCond(&s.mu)
	}

	// Cache còn hạn → trả về ngay
	if s.cachedProxy != "" && time.Now().Before(s.cacheExpiry) {
		p := s.cachedProxy
		s.mu.Unlock()
		return p, nil
	}

	// Goroutine khác đang fetch → chờ
	for s.fetching {
		s.fetchCond.Wait()
		if s.cachedProxy != "" && time.Now().Before(s.cacheExpiry) {
			p := s.cachedProxy
			s.mu.Unlock()
			return p, nil
		}
	}

	// Chúng ta fetch
	s.fetching = true
	s.mu.Unlock()

	proxy, nextChange, err := s.callAPI(ctx)

	s.mu.Lock()
	s.fetching = false
	if err == nil && proxy != "" {
		s.cachedProxy = proxy
		// cache hết hạn sớm hơn 2s để tránh dùng proxy sắp hết hạn
		ttl := time.Duration(nextChange)*time.Second - 2*time.Second
		if ttl < 5*time.Second {
			ttl = 5 * time.Second
		}
		s.cacheExpiry = time.Now().Add(ttl)
	}
	s.fetchCond.Broadcast()
	s.mu.Unlock()

	return proxy, err
}

// callAPI thực sự gọi HTTP đến ShopLike API.
func (s *shoplikeInstance) callAPI(ctx context.Context) (proxyStr string, nextChange int, err error) {
	for attempt := 0; attempt < 3; attempt++ {
		select {
		case <-ctx.Done():
			return "", 0, ctx.Err()
		default:
		}

		apiURL := fmt.Sprintf("%s/getNewProxy?access_token=%s", shoplikeBase, s.apiKey)
		req, e := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if e != nil {
			return "", 0, e
		}

		resp, e := apiClient.Do(req)
		if e != nil {
			err = fmt.Errorf("shoplike request: %w", e)
			continue
		}
		body, _ := httpx.ReadBody(resp.Body, 32*1024)
		resp.Body.Close()

		var result struct {
			Status  string `json:"status"`
			Message string `json:"message"`
			Data    struct {
				Proxy      string `json:"proxy"`
				NextChange int    `json:"nextChange"`
			} `json:"data"`
		}

		if e := json.Unmarshal(body, &result); e != nil {
			return "", 0, fmt.Errorf("shoplike parse: %w (body=%s)", e, string(body[:min(len(body), 200)]))
		}

		if result.Status == "success" && result.Data.Proxy != "" {
			// Không log key prefix — 8 ký tự đầu giảm entropy đáng kể nếu log file leak
			slog.Info("shoplike callAPI ok", "proxy", result.Data.Proxy, "nextChange", result.Data.NextChange)
			return result.Data.Proxy, result.Data.NextChange, nil
		}
		err = fmt.Errorf("shoplike: %s", result.Message)
		slog.Warn("shoplike callAPI fail", "status", result.Status, "msg", result.Message, "attempt", attempt)
	}
	return "", 0, err
}

// ShoplikePool quản lý nhiều ShopLike API key
// Mapping từ WeBM listShopLike + AcquireProxy(Type=7)
type ShoplikePool struct {
	items       []*shoplikeInstance
	threadLimit int
	mu          sync.Mutex
}

// NewShoplikePool tạo pool từ danh sách ShopLike API keys (mỗi dòng một key).
// keysStr: chuỗi nhiều key phân cách bằng newline.
// threadPerIP: số thread tối đa được dùng đồng thời trên mỗi key.
func NewShoplikePool(keysStr string, threadPerIP int) *ShoplikePool {
	if threadPerIP <= 0 {
		threadPerIP = 1
	}
	var items []*shoplikeInstance
	for _, key := range strings.Split(keysStr, "\n") {
		if key = strings.TrimSpace(key); key != "" {
			inst := &shoplikeInstance{apiKey: key}
			inst.fetchCond = sync.NewCond(&inst.mu)
			items = append(items, inst)
		}
	}
	return &ShoplikePool{items: items, threadLimit: threadPerIP}
}

// Len trả về số lượng ShopLike API key trong pool.
func (p *ShoplikePool) Len() int { return len(p.items) }

// leastActive tìm instance có activeCount thấp nhất (phân phối tải đều).
func (p *ShoplikePool) leastActive() *shoplikeInstance {
	if len(p.items) == 0 {
		return nil
	}
	best := p.items[0]
	for _, item := range p.items[1:] {
		if item.active() < best.active() {
			best = item
		}
	}
	return best
}

// Acquire lấy proxy từ instance ít dùng nhất còn capacity, trả về (proxyStr, releaseFunc, error).
// ctx: context để cancel nếu bị dừng.
// releaseFunc PHẢI được gọi khi account xong để giảm activeCount.
func (p *ShoplikePool) Acquire(ctx context.Context) (string, func(), error) {
	p.mu.Lock()
	var inst *shoplikeInstance
	for _, item := range p.items {
		if item.active() < p.threadLimit {
			if inst == nil || item.active() < inst.active() {
				inst = item
			}
		}
	}
	if inst == nil {
		inst = p.leastActive()
	}
	p.mu.Unlock()

	if inst == nil {
		return "", func() {}, fmt.Errorf("shoplike: no instance")
	}

	proxyStr, err := inst.getProxy(ctx)
	if err != nil {
		return "", func() {}, err
	}
	inst.incr()
	return proxyStr, func() { inst.decr() }, nil
}
