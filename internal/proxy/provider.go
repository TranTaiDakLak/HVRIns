// provider.go — Provider interface chung cho các commercial proxy pool
package proxy

import "context"

// Provider là interface chung cho các commercial proxy pool
// (Tinsoft, ShopLike, NetProxy, MinProxy, ProxyFarm).
//
// Manual proxy list (Pool) không implement interface này — Manager xử lý riêng.
type Provider interface {
	// Acquire lấy proxy từ pool. Trả về proxy string, release func (PHẢI gọi khi xong),
	// và error nếu không lấy được.
	Acquire(ctx context.Context) (string, func(), error)
	// Len trả về số lượng key/instance trong pool.
	Len() int
}
