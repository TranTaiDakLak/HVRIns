// tinsoft.go — Tinsoft / WwProxy provider
// Mapping từ WeBM WwProxy.cs + frmFacebook.Proxy.cs (Type 6)
// API: https://wwproxy.com/api/client/proxy/available?key=KEY&provinceId=-1
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

const tinsoftBase = "https://wwproxy.com/api/client"

type tinsoftInstance struct {
	apiKey      string
	activeCount int
	mu          sync.Mutex
}

// incr tăng bộ đếm số thread đang dùng instance này (gọi khi bắt đầu dùng proxy).
func (t *tinsoftInstance) incr() {
	t.mu.Lock()
	t.activeCount++
	t.mu.Unlock()
}

// decr giảm bộ đếm số thread đang dùng instance này (gọi khi release proxy).
func (t *tinsoftInstance) decr() {
	t.mu.Lock()
	if t.activeCount > 0 {
		t.activeCount--
	}
	t.mu.Unlock()
}

// active trả về số thread đang dùng instance này (thread-safe).
func (t *tinsoftInstance) active() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.activeCount
}

// getProxy gọi API wwproxy lấy proxy hiện tại
// Mapping từ WeBM WwProxy.ChangeProxyAsync() dòng 74-117
func (t *tinsoftInstance) getProxy(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/proxy/available?key=%s&provinceId=-1", tinsoftBase, t.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := apiClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("tinsoft request failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 32*1024)

	var result struct {
		Status string `json:"status"`
		Data   struct {
			Proxy     string `json:"proxy"`
			IpAddress string `json:"ipAddress"`
			Port      int    `json:"port"`
		} `json:"data"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("tinsoft parse: %w (body=%s)", err, string(body[:min(len(body), 200)]))
	}
	if result.Status != "OK" {
		return "", fmt.Errorf("tinsoft error: %s", result.Message)
	}

	if result.Data.Proxy != "" {
		return result.Data.Proxy, nil
	}
	return fmt.Sprintf("%s:%d", result.Data.IpAddress, result.Data.Port), nil
}

// TinsoftPool quản lý nhiều API key Tinsoft, phân phối theo least-active
// Mapping từ WeBM listWwProxy + AcquireProxy(Type=6)
type TinsoftPool struct {
	items       []*tinsoftInstance
	threadLimit int
	mu          sync.Mutex
}

// NewTinsoftPool tạo pool từ danh sách API keys (mỗi dòng một key)
func NewTinsoftPool(keysStr string, threadPerIP int) *TinsoftPool {
	if threadPerIP <= 0 {
		threadPerIP = 1
	}
	var items []*tinsoftInstance
	for _, key := range strings.Split(keysStr, "\n") {
		if key = strings.TrimSpace(key); key != "" {
			items = append(items, &tinsoftInstance{apiKey: key})
		}
	}
	return &TinsoftPool{items: items, threadLimit: threadPerIP}
}

// Len trả về số lượng Tinsoft API key trong pool.
func (p *TinsoftPool) Len() int { return len(p.items) }

// leastActive tìm instance ít dùng nhất có capacity
// Mapping từ WeBM: tìm proxy có TotalUsageCount nhỏ nhất
func (p *TinsoftPool) leastActive() *tinsoftInstance {
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

// Acquire lấy proxy từ instance ít dùng nhất, trả về (proxyStr, releaseFunc, error)
func (p *TinsoftPool) Acquire(ctx context.Context) (string, func(), error) {
	p.mu.Lock()
	// Tìm instance chưa đầy
	var inst *tinsoftInstance
	for _, item := range p.items {
		if item.active() < p.threadLimit {
			if inst == nil || item.active() < inst.active() {
				inst = item
			}
		}
	}
	// Fallback: dùng instance ít active nhất
	if inst == nil {
		inst = p.leastActive()
	}
	p.mu.Unlock()

	if inst == nil {
		return "", func() {}, fmt.Errorf("tinsoft: no instance available")
	}

	proxyStr, err := inst.getProxy(ctx)
	if err != nil {
		return "", func() {}, err
	}
	inst.incr()
	return proxyStr, func() { inst.decr() }, nil
}
