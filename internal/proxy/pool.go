// pool.go — Round-robin proxy pool cho manual proxy list
// Mapping từ WeBM: qProxyRotate Queue<string> — Dequeue/Enqueue (FIFO rotation)
package proxy

import (
	"strings"
	"sync"
)

// Pool quản lý manual proxy list theo round-robin
// WeBM Case 4: proxyInfo = qProxyRotate.Dequeue(); qProxyRotate.Enqueue(proxyInfo);
type Pool struct {
	items []string
	idx   int
	mu    sync.Mutex
}

// NewPool tạo pool từ danh sách proxy (mỗi dòng một proxy)
func NewPool(proxyLines string) *Pool {
	var items []string
	for _, line := range strings.Split(proxyLines, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			items = append(items, line)
		}
	}
	return &Pool{items: items}
}

// Len trả về số lượng proxy trong pool.
func (p *Pool) Len() int { return len(p.items) }

// Next trả về proxy tiếp theo theo round-robin (FIFO)
func (p *Pool) Next() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.items) == 0 {
		return ""
	}
	addr := p.items[p.idx]
	p.idx = (p.idx + 1) % len(p.items)
	return addr
}

// Fixed luôn trả về proxy đầu tiên (proxy_fixed mode)
func (p *Pool) Fixed() string {
	if len(p.items) == 0 {
		return ""
	}
	return p.items[0]
}
