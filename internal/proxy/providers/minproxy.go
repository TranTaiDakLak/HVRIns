// minproxy.go — MinProxy provider
// Mapping từ WeBM MinProxy.cs + frmFacebook.Proxy.cs (Type 9)
// API: http://dash.minproxy.vn/api/rotating/v1/proxy/get-new-proxy?api_key=KEY
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

const minproxyBase = "http://dash.minproxy.vn/api/rotating/v1/proxy"

type minproxyInstance struct {
	apiKey      string
	activeCount int
	mu          sync.Mutex
}

// incr tăng bộ đếm số thread đang dùng instance này.
func (m *minproxyInstance) incr() { m.mu.Lock(); m.activeCount++; m.mu.Unlock() }

// decr giảm bộ đếm số thread đang dùng instance này (không âm).
func (m *minproxyInstance) decr() {
	m.mu.Lock()
	if m.activeCount > 0 {
		m.activeCount--
	}
	m.mu.Unlock()
}

// active trả về số thread đang dùng instance này (thread-safe).
func (m *minproxyInstance) active() int { m.mu.Lock(); defer m.mu.Unlock(); return m.activeCount }

// getProxy gọi MinProxy API lấy proxy mới
// Mapping từ WeBM MinProxy.ChangeProxy() dòng 136-185
func (m *minproxyInstance) getProxy(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/get-new-proxy?api_key=%s", minproxyBase, m.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := apiClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("minproxy request: %w", err)
	}
	body, _ := httpx.ReadBody(resp.Body, 32*1024)
	resp.Body.Close()

	var result struct {
		Code string `json:"code"`
		Data struct {
			HttpProxy   string `json:"http_proxy"`
			Socks5      string `json:"socks5"`
			NextRequest int    `json:"next_request"` // giây chờ trước khi đổi tiếp
			Timeout     int    `json:"timeout"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("minproxy parse: %w (body=%s)", err, string(body[:min(len(body), 200)]))
	}

	// Code "2" = success theo WeBM
	if result.Code != "2" && result.Code != "200" {
		return "", fmt.Errorf("minproxy error code %s: %s", result.Code, result.Message)
	}

	proxyStr := result.Data.HttpProxy
	if proxyStr == "" {
		return "", fmt.Errorf("minproxy: empty proxy returned")
	}

	return proxyStr, nil
}

// MinproxyPool quản lý nhiều MinProxy API key
// Mapping từ WeBM listMinProxy + AcquireProxy(Type=9)
type MinproxyPool struct {
	items       []*minproxyInstance
	threadLimit int
	mu          sync.Mutex
}

// NewMinproxyPool tạo pool từ danh sách MinProxy API keys (mỗi dòng một key).
// keysStr: chuỗi nhiều key phân cách bằng newline.
// threadPerIP: số thread tối đa được dùng đồng thời trên mỗi key.
func NewMinproxyPool(keysStr string, threadPerIP int) *MinproxyPool {
	if threadPerIP <= 0 {
		threadPerIP = 1
	}
	var items []*minproxyInstance
	for _, key := range strings.Split(keysStr, "\n") {
		if key = strings.TrimSpace(key); key != "" {
			items = append(items, &minproxyInstance{apiKey: key})
		}
	}
	return &MinproxyPool{items: items, threadLimit: threadPerIP}
}

// Len trả về số lượng MinProxy API key trong pool.
func (p *MinproxyPool) Len() int { return len(p.items) }

// leastActive tìm instance có activeCount thấp nhất (phân phối tải đều).
func (p *MinproxyPool) leastActive() *minproxyInstance {
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
func (p *MinproxyPool) Acquire(ctx context.Context) (string, func(), error) {
	p.mu.Lock()
	var inst *minproxyInstance
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
		return "", func() {}, fmt.Errorf("minproxy: no instance")
	}

	proxyStr, err := inst.getProxy(ctx)
	if err != nil {
		return "", func() {}, err
	}
	inst.incr()
	return proxyStr, func() { inst.decr() }, nil
}

// parseMinproxyStr tách "ip:port" hoặc "ip:port:user:pass" từ MinProxy response
func parseMinproxyStr(s string) string {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	if len(parts) >= 4 {
		return fmt.Sprintf("%s:%s:%s:%s", parts[0], parts[1], parts[2], parts[3])
	}
	if len(parts) == 2 {
		return s
	}
	return s
}
