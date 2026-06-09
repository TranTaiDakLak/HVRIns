// proxyfarm.go — ProxyFarm provider
// Mapping từ WeBM ProxyFarm.cs + frmFacebook.Proxy.cs (Type 12)
// API: https://proxyxoay.shop/api/get.php?key=KEY&nhamang=random&tinhthanh=0
package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"HVRIns/internal/httpx"
)

const proxyfarmBase = "https://proxyxoay.shop/api/get.php"

type proxyfarmInstance struct {
	apiKey      string
	activeCount int
	mu          sync.Mutex
}

// incr tăng bộ đếm số thread đang dùng instance này.
func (f *proxyfarmInstance) incr() { f.mu.Lock(); f.activeCount++; f.mu.Unlock() }

// decr giảm bộ đếm số thread đang dùng instance này (không âm).
func (f *proxyfarmInstance) decr() {
	f.mu.Lock()
	if f.activeCount > 0 {
		f.activeCount--
	}
	f.mu.Unlock()
}

// active trả về số thread đang dùng instance này (thread-safe).
func (f *proxyfarmInstance) active() int { f.mu.Lock(); defer f.mu.Unlock(); return f.activeCount }

// getProxy gọi ProxyFarm API
// Mapping từ WeBM ProxyFarm.ChangeProxy() dòng 163-250
// Response status "100" = success
func (f *proxyfarmInstance) getProxy(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s?key=%s&nhamang=random&tinhthanh=0", proxyfarmBase, f.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := apiClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("proxyfarm request: %w", err)
	}
	body, _ := httpx.ReadBody(resp.Body, 32*1024)
	resp.Body.Close()

	var result struct {
		Status     string `json:"status"`    // "100" = success
		Proxyhttp  string `json:"proxyhttp"` // "ip:port:user:pass"
		Proxysocks string `json:"proxysocks"`
		Message    string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("proxyfarm parse: %w (body=%s)", err, string(body[:min(len(body), 200)]))
	}

	if result.Status != "100" {
		return "", fmt.Errorf("proxyfarm error status=%s: %s", result.Status, result.Message)
	}

	proxyStr := strings.TrimSpace(result.Proxyhttp)
	if proxyStr == "" {
		return "", fmt.Errorf("proxyfarm: empty proxy")
	}
	return proxyStr, nil
}

// ProxyfarmPool quản lý nhiều ProxyFarm API key
// Mapping từ WeBM listProxyFarms + AcquireProxy(Type=12)
type ProxyfarmPool struct {
	items       []*proxyfarmInstance
	threadLimit int
	mu          sync.Mutex
}

// NewProxyfarmPool tạo pool từ danh sách ProxyFarm API keys (mỗi dòng một key).
// keysStr: chuỗi nhiều key phân cách bằng newline.
// threadPerIP: số thread tối đa được dùng đồng thời trên mỗi key.
func NewProxyfarmPool(keysStr string, threadPerIP int) *ProxyfarmPool {
	if threadPerIP <= 0 {
		threadPerIP = 1
	}
	var items []*proxyfarmInstance
	for _, key := range strings.Split(keysStr, "\n") {
		if key = strings.TrimSpace(key); key != "" {
			items = append(items, &proxyfarmInstance{apiKey: key})
		}
	}
	return &ProxyfarmPool{items: items, threadLimit: threadPerIP}
}

// Len trả về số lượng ProxyFarm API key trong pool.
func (p *ProxyfarmPool) Len() int { return len(p.items) }

// leastActive tìm instance có activeCount thấp nhất (phân phối tải đều).
func (p *ProxyfarmPool) leastActive() *proxyfarmInstance {
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
func (p *ProxyfarmPool) Acquire(ctx context.Context) (string, func(), error) {
	p.mu.Lock()
	var inst *proxyfarmInstance
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
		return "", func() {}, fmt.Errorf("proxyfarm: no instance")
	}

	proxyStr, err := inst.getProxy(ctx)
	if err != nil {
		return "", func() {}, err
	}
	inst.incr()
	return proxyStr, func() { inst.decr() }, nil
}
