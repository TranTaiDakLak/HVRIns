// manager.go — Proxy Manager: chọn provider đúng theo IpProvider
// Mapping từ WeBM frmFacebook.Proxy.cs AcquireProxy() + InitializeProxiesAsync()
package proxy

import (
	"context"
	"fmt"

	"HVRIns/internal/proxy/providers"
)

// ManagerConfig cấu hình proxy manager — mapping từ frontend IpConfig + GeneralConfig
type ManagerConfig struct {
	Provider string // "none"|"proxy"|"proxy_fixed"|"tinsoft"|"shoplike"|"netproxy"|"minproxy"|"proxy_farm"

	// Manual proxy list (provider=proxy/proxy_fixed)
	ProxyList string

	// Tinsoft / WwProxy (Type 6)
	TinsoftKeys    string
	TinsoftThreads int

	// ShopLike (Type 7)
	ShoplikeKeys    string
	ShoplikeThreads int

	// NetProxy (Type 8 / 10)
	NetproxyKeys    string
	NetproxyThreads int

	// MinProxy (Type 9)
	MinproxyKeys    string
	MinproxyThreads int

	// ProxyFarm (Type 12)
	ProxyfarmKeys    string
	ProxyfarmThreads int
}

// Manager điều phối việc lấy proxy theo provider đã cấu hình
// Mapping từ WeBM AcquireProxy() switch(cboChangeIP)
type Manager struct {
	cfg      ManagerConfig
	pool     *Pool     // cho "proxy" / "proxy_fixed"
	provider Provider  // cho tất cả commercial provider
}

// NewManager khởi tạo Manager và các provider pool tương ứng theo cfg.Provider.
//
// cfg: ManagerConfig chứa toàn bộ cấu hình proxy — provider được chọn,
// proxy list thô (cho "proxy"/"proxy_fixed"), và keys + threads cho
// các provider thương mại (Tinsoft, ShopLike, NetProxy, MinProxy,
// ProxyFarm). Các field không dùng bởi provider được chọn bị bỏ qua.
//
// Hàm chỉ nên gọi 1 lần trước khi bắt đầu RunVerify — tương đương
// InitializeProxiesAsync() trong WeBM. Khởi tạo lại giữa chừng sẽ
// reset hết trạng thái round-robin và usage counter của pool.
//
// Chiến lược pool theo provider:
//   - "proxy"/"proxy_fixed": dùng Pool nội bộ, parse ProxyList thành slice.
//   - "tinsoft"/"shoplike"/"netproxy"/"minproxy"/"proxy_farm": khởi tạo
//     pool riêng của provider, mỗi pool quản lý semaphore theo ThreadPerIP.
//   - Các provider khác ("none", "hma", "fpt", "xproxy"): không khởi tạo
//     pool — Acquire() sẽ trả về chuỗi rỗng hoặc error tương ứng.
func NewManager(cfg ManagerConfig) *Manager {
	m := &Manager{cfg: cfg}

	switch cfg.Provider {
	case "proxy", "proxy_fixed":
		m.pool = NewPool(cfg.ProxyList)

	case "tinsoft":
		m.provider = providers.NewTinsoftPool(cfg.TinsoftKeys, cfg.TinsoftThreads)

	case "shoplike":
		m.provider = providers.NewShoplikePool(cfg.ShoplikeKeys, cfg.ShoplikeThreads)

	case "netproxy", "netproxy_gb":
		m.provider = providers.NewNetproxyPool(cfg.NetproxyKeys, cfg.NetproxyThreads)

	case "minproxy":
		m.provider = providers.NewMinproxyPool(cfg.MinproxyKeys, cfg.MinproxyThreads)

	case "proxy_farm":
		m.provider = providers.NewProxyfarmPool(cfg.ProxyfarmKeys, cfg.ProxyfarmThreads)
	}

	return m
}

// IsConfigured kiểm tra xem Manager có provider hợp lệ và sẵn sàng dùng
// không. Trả về false trong các trường hợp sau:
//
//   - Provider là "" hoặc "none": người dùng không cài proxy.
//   - Provider là "hma", "fpt", "xproxy": các provider này chưa được hỗ
//     trợ hoặc không cần quản lý pool (HMA dùng system-wide, FPT dùng
//     dial-up, xproxy cần service URL riêng) — xem như không configured.
//   - Provider hợp lệ nhưng pool nil hoặc rỗng (ví dụ ProxyList trống,
//     không có key nào được nhập).
//
// Caller dùng IsConfigured() để quyết định có bắt buộc proxy hay không
// trước khi RunVerify. Nếu false, Acquire() vẫn có thể gọi được nhưng
// sẽ trả về chuỗi proxy rỗng (chạy không proxy).
func (m *Manager) IsConfigured() bool {
	switch m.cfg.Provider {
	case "", "none", "hma", "fpt", "xproxy":
		return false
	case "proxy", "proxy_fixed":
		return m.pool != nil && m.pool.Len() > 0
	default:
		return m.provider != nil && m.provider.Len() > 0
	}
}

// Acquire lấy proxy cho một account/thread đang chạy.
// Trả về (proxyStr, releaseFunc, error).
//
// ctx: context để hủy chờ khi provider thương mại cần đợi slot khả dụng
// (ví dụ Tinsoft pool đã hết slot cho phép). Với "proxy"/"proxy_fixed"
// context không được dùng vì không có blocking.
//
// releaseFunc: PHẢI được gọi sau khi account xử lý xong — thường qua
// defer release(). Với "proxy" và "proxy_fixed" release là no-op. Với
// các provider thương mại (Tinsoft, ShopLike...) release giải phóng slot
// semaphore để thread khác có thể dùng proxy đó — tương đương
// DecrementActiveUsage() trong WeBM. Không gọi release dẫn đến deadlock
// (pool cạn slot mà không được trả lại).
//
// Chiến lược phân phối proxy:
//   - "proxy" (round-robin): mỗi account lấy proxy tiếp theo trong danh
//     sách theo thứ tự vòng tròn. Phân tải đều nhau.
//   - "proxy_fixed": tất cả account dùng proxy đầu tiên trong danh sách.
//     Dùng khi chỉ có 1 proxy hoặc muốn cố định IP.
//   - Provider thương mại: delegate sang pool riêng, pool tự quản lý
//     round-robin key + semaphore thread-per-IP.
func (m *Manager) Acquire(ctx context.Context) (string, func(), error) {
	switch m.cfg.Provider {
	case "", "none", "hma", "fpt":
		// Không proxy hoặc chưa hỗ trợ
		return "", func() {}, nil

	case "proxy":
		if m.pool == nil || m.pool.Len() == 0 {
			return "", func() {}, fmt.Errorf("proxy list trống")
		}
		return m.pool.Next(), func() {}, nil

	case "proxy_fixed":
		if m.pool == nil || m.pool.Len() == 0 {
			return "", func() {}, fmt.Errorf("proxy list trống")
		}
		// Cố định: tất cả account dùng proxy đầu tiên
		return m.pool.Fixed(), func() {}, nil

	case "xproxy":
		// Xproxy cần service URL riêng + device management — chưa implement
		return "", func() {}, fmt.Errorf("xproxy chưa được hỗ trợ trong phiên bản này")

	default:
		if m.provider == nil || m.provider.Len() == 0 {
			return "", func() {}, fmt.Errorf("provider '%s': không có key nào được cấu hình", m.cfg.Provider)
		}
		return m.provider.Acquire(ctx)
	}
}
