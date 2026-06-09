// netproxy.go — NetProxy provider
// Mapping từ WeBM NetProxy.cs + frmFacebook.Proxy.cs (Type 8)
// API: https://api.netproxy.io/api/rotateProxy/getNewProxy?apiKey=KEY
package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"HVRIns/internal/httpx"
)

const netproxyBase = "https://api.netproxy.io/api/rotateProxy"

type netproxyInstance struct {
	apiKey      string
	activeCount int
	mu          sync.Mutex
}

// incr tăng bộ đếm số thread đang dùng instance này.
func (n *netproxyInstance) incr() { n.mu.Lock(); n.activeCount++; n.mu.Unlock() }

// decr giảm bộ đếm số thread đang dùng instance này (không âm).
func (n *netproxyInstance) decr() {
	n.mu.Lock()
	if n.activeCount > 0 {
		n.activeCount--
	}
	n.mu.Unlock()
}

// active trả về số thread đang dùng instance này (thread-safe).
func (n *netproxyInstance) active() int { n.mu.Lock(); defer n.mu.Unlock(); return n.activeCount }

// getProxy gọi API NetProxy, poll status nếu cần chờ
// Mapping từ WeBM NetProxy.GetProxyAsync() dòng 165-170 + GetNewProxyAsync() dòng 99-127
func (n *netproxyInstance) getProxy(ctx context.Context) (string, error) {
	// Thử lấy proxy mới trước
	url := fmt.Sprintf("%s/getNewProxy?apiKey=%s", netproxyBase, n.apiKey)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := apiClient.Do(req)
	if err != nil {
		// Thử lấy proxy hiện tại
		return n.getCurrentProxy(ctx, apiClient)
	}
	body, _ := httpx.ReadBody(resp.Body, 32*1024)
	resp.Body.Close()

	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Proxy      string `json:"proxy"`
			Username   string `json:"username"`
			Password   string `json:"password"`
			NextChange int    `json:"nextChange"`
		} `json:"data"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("netproxy parse: %w (body=%s)", err, string(body[:min(len(body), 200)]))
	}

	if result.Success && result.Data.Proxy != "" {
		proxyStr := result.Data.Proxy
		// Nếu có credentials, assemble format ip:port:user:pass
		if result.Data.Username != "" {
			parts := strings.SplitN(proxyStr, ":", 2)
			if len(parts) == 2 {
				proxyStr = fmt.Sprintf("%s:%s:%s:%s", parts[0], parts[1], result.Data.Username, result.Data.Password)
			}
		}
		return proxyStr, nil
	}

	// Fallback: lấy proxy hiện tại
	return n.getCurrentProxy(ctx, apiClient)
}

// getCurrentProxy poll getCurrentProxy endpoint tối đa 10 lần (fallback khi getNewProxy thất bại).
// ctx: context để cancel nếu bị dừng.
// client: HTTP client dùng chung với getProxy.
func (n *netproxyInstance) getCurrentProxy(ctx context.Context, client *http.Client) (string, error) {
	// Poll proxy hiện tại với retry
	for i := 0; i < 10; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		url := fmt.Sprintf("%s/getCurrentProxy?apiKey=%s", netproxyBase, n.apiKey)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return "", err
		}

		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		body, _ := httpx.ReadBody(resp.Body, 32*1024)
		resp.Body.Close()

		var result struct {
			Success bool `json:"success"`
			Data    struct {
				Proxy    string `json:"proxy"`
				Username string `json:"username"`
				Password string `json:"password"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &result); err == nil && result.Success && result.Data.Proxy != "" {
			proxyStr := result.Data.Proxy
			if result.Data.Username != "" {
				parts := strings.SplitN(proxyStr, ":", 2)
				if len(parts) == 2 {
					proxyStr = fmt.Sprintf("%s:%s:%s:%s", parts[0], parts[1], result.Data.Username, result.Data.Password)
				}
			}
			return proxyStr, nil
		}

		time.Sleep(1 * time.Second)
	}
	return "", fmt.Errorf("netproxy: timeout waiting for proxy")
}

// NetproxyPool quản lý nhiều NetProxy API key
// Mapping từ WeBM listNetProxy + AcquireProxy(Type=8)
type NetproxyPool struct {
	items       []*netproxyInstance
	threadLimit int
	mu          sync.Mutex
}

// NewNetproxyPool tạo pool từ danh sách NetProxy API keys (mỗi dòng một key).
// keysStr: chuỗi nhiều key phân cách bằng newline.
// threadPerIP: số thread tối đa được dùng đồng thời trên mỗi key.
func NewNetproxyPool(keysStr string, threadPerIP int) *NetproxyPool {
	if threadPerIP <= 0 {
		threadPerIP = 1
	}
	var items []*netproxyInstance
	for _, key := range strings.Split(keysStr, "\n") {
		if key = strings.TrimSpace(key); key != "" {
			items = append(items, &netproxyInstance{apiKey: key})
		}
	}
	return &NetproxyPool{items: items, threadLimit: threadPerIP}
}

// Len trả về số lượng NetProxy API key trong pool.
func (p *NetproxyPool) Len() int { return len(p.items) }

// leastActive tìm instance có activeCount thấp nhất (phân phối tải đều).
func (p *NetproxyPool) leastActive() *netproxyInstance {
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
func (p *NetproxyPool) Acquire(ctx context.Context) (string, func(), error) {
	p.mu.Lock()
	var inst *netproxyInstance
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
		return "", func() {}, fmt.Errorf("netproxy: no instance")
	}

	proxyStr, err := inst.getProxy(ctx)
	if err != nil {
		return "", func() {}, err
	}
	inst.incr()
	return proxyStr, func() { inst.decr() }, nil
}
