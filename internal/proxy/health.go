// Package proxy — proxy health check trước khi chạy
package proxy

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HealthResult kết quả check 1 proxy
type HealthResult struct {
	Proxy   string `json:"proxy"`
	Healthy bool   `json:"healthy"`
	Latency int    `json:"latency"` // ms
	Error   string `json:"error,omitempty"`
}

// CheckProxyHealth kiểm tra danh sách proxy song song và trả về kết quả
// cho từng proxy theo thứ tự gốc của slice đầu vào.
//
// ctx: context để hủy toàn bộ quá trình check — khi ctx bị cancel, các
// goroutine con cũng dừng lại nhờ context propagation qua checkOne.
//
// proxies: danh sách proxy cần kiểm tra, mỗi phần tử theo định dạng
// "ip:port" hoặc "ip:port:user:pass". Thứ tự kết quả trả về tương ứng
// 1-1 với thứ tự đầu vào.
//
// timeout: thời gian tối đa cho mỗi proxy check (cả connect + read). Nếu
// truyền 0 thì mặc định 5 giây. Nên đặt đủ ngắn (3-10s) vì proxy chậm
// thường là proxy chết.
//
// URL trung lập: hàm dùng https://www.gstatic.com/generate_204 thay vì
// Facebook hay Google Search để tránh tiêu tốn request quota hoặc trigger
// rate-limit của Facebook trong quá trình health check trước khi chạy.
// gstatic chấp nhận HTTP GET và trả về 204 No Content — nhẹ, nhanh, không
// có session hay cookie.
//
// Concurrency cap qua semaphore — tránh spawn 1000 goroutine + 1000 HTTP transport
// đồng thời (mỗi goroutine có HTTP client + TLS handshake = ~500KB-1MB native overhead).
// 50 song song đủ nhanh cho 1000 proxy: ~20 wave × 8s timeout/wave ≈ 160s tổng (cap mỗi
// proxy 8s timeout). Với 100 proxy: 2 wave × 8s = ~16s. Trade-off chấp nhận được.
const checkProxyConcurrencyCap = 50

// Tất cả goroutine check chạy song song nhưng giới hạn `checkProxyConcurrencyCap`,
// tổng thời gian check xấp xỉ bằng (proxy chậm nhất × ceil(N/cap)).
func CheckProxyHealth(ctx context.Context, proxies []string, timeout time.Duration) []HealthResult {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	results := make([]HealthResult, len(proxies))
	var wg sync.WaitGroup
	sem := make(chan struct{}, checkProxyConcurrencyCap)

	for i, p := range proxies {
		wg.Add(1)
		sem <- struct{}{} // acquire — block nếu đã có 50 goroutine đang chạy
		go func(idx int, proxyStr string) {
			defer wg.Done()
			defer func() { <-sem }() // release
			results[idx] = checkOne(ctx, proxyStr, timeout)
		}(i, p)
	}
	wg.Wait()
	return results
}

// FilterHealthy lọc kết quả từ CheckProxyHealth và trả về chỉ những proxy
// healthy (Healthy == true), giữ nguyên thứ tự.
//
// results: slice HealthResult từ CheckProxyHealth. Proxy không healthy
// (timeout, kết nối bị từ chối, status code bất thường) bị loại bỏ.
//
// Kết quả dùng để thay thế ProxyList ban đầu trước khi RunVerify, loại bỏ
// proxy chết khỏi pool ngay từ đầu thay vì phát hiện khi đang chạy.
func FilterHealthy(results []HealthResult) []string {
	var healthy []string
	for _, r := range results {
		if r.Healthy {
			healthy = append(healthy, r.Proxy)
		}
	}
	return healthy
}

// healthCheckEndpoints — fallback list cho health check.
// Nhiều proxy pool (proxyshare GB session, iprocket) chặn 1-2 endpoint cụ thể
// nhưng không chặn hết. Thử lần lượt, 1 cái trả 200 là coi như proxy healthy.
// Tất cả đều là HTTP plain (không HTTPS) — tránh TLS handshake fail trên proxy yếu.
var healthCheckEndpoints = []string{
	"http://checkip.amazonaws.com",       // AWS — stable, nhẹ, trả text IP
	"http://ip-api.com/json/?fields=query", // ip-api — nhanh, luôn ok trên proxyshare
	"http://api.ipify.org",               // ipify — fallback cuối, text IP
}

// checkOne xác nhận proxy sống bằng 2 tier, **TCP FIRST** để nhanh:
//   - Tier 1: TCP dial tới proxy host:port (< 1s nếu sống, ~3s nếu chết)
//   - Tier 2: Nếu TCP fail, thử 1 HTTP endpoint qua proxy (fallback hiếm gặp)
//
// Lý do TCP first: proxy sống = listen port TCP. Không cần gửi HTTP GET tới bên ngoài
// vì nhiều proxy pool (proxyshare session, iprocket) chặn một số endpoint HTTP cụ thể
// dù proxy hoạt động bình thường. Test HTTP tốn 5-20s vô ích trong trường hợp đó.
//
// ctx: context cha để hủy.
// proxyStr: "ip:port" hoặc "ip:port:user:pass".
// timeout: total budget — TCP dùng min(3s, timeout/3); HTTP fallback dùng phần còn lại.
func checkOne(ctx context.Context, proxyStr string, timeout time.Duration) HealthResult {
	result := HealthResult{Proxy: proxyStr}
	start := time.Now()

	// Tier 1: TCP dial — nhanh (< 1s) và đủ để xác nhận proxy listen
	tcpTimeout := 3 * time.Second
	if timeout < 3*time.Second {
		tcpTimeout = timeout
	}
	host := extractProxyHost(proxyStr)
	if host != "" {
		dialCtx, dialCancel := context.WithTimeout(ctx, tcpTimeout)
		conn, derr := (&net.Dialer{}).DialContext(dialCtx, "tcp", host)
		dialCancel()
		if derr == nil {
			conn.Close()
			result.Latency = int(time.Since(start).Milliseconds())
			result.Healthy = true
			return result
		}
		// TCP fail — có thể do DNS / firewall chặn port 5960 ở client, thử HTTP CONNECT qua proxy
		result.Error = "tcp dial: " + derr.Error()
	} else {
		result.Error = "proxy format không parse được host:port"
	}

	// Tier 2: HTTP CONNECT fallback — chỉ 1 endpoint, budget còn lại sau TCP
	httpBudget := timeout - time.Since(start)
	if httpBudget < 2*time.Second {
		httpBudget = 2 * time.Second
	}
	client := CreateClient(proxyStr, httpBudget)
	epCtx, cancel := context.WithTimeout(ctx, httpBudget)
	defer cancel()
	req, err := http.NewRequestWithContext(epCtx, "GET", healthCheckEndpoints[0], nil)
	if err != nil {
		result.Latency = int(time.Since(start).Milliseconds())
		return result
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		result.Latency = int(time.Since(start).Milliseconds())
		result.Error = "http fallback: " + err.Error()
		return result
	}
	resp.Body.Close()
	result.Latency = int(time.Since(start).Milliseconds())
	if resp.StatusCode == 200 {
		result.Healthy = true
		result.Error = "" // clear
	} else {
		result.Error = "http fallback: unexpected status: " + resp.Status
	}
	return result
}

// extractProxyHost lấy "host:port" từ proxy string để dùng cho TCP dial fallback.
// Hỗ trợ các format: "host:port", "host:port:user:pass", "user:pass@host:port",
// "http://user:pass@host:port", "socks5://host:port". Trả về "" nếu parse fail.
func extractProxyHost(proxyStr string) string {
	s := strings.TrimSpace(proxyStr)
	if s == "" {
		return ""
	}
	// parseProxy đã handle mọi format và trả về url.URL với Host=host:port
	if u := parseProxy(s); u != nil && u.Host != "" {
		return u.Host
	}
	// Fallback: legacy "host:port:..." → lấy 2 parts đầu
	parts := strings.Split(s, ":")
	if len(parts) >= 2 && isPort(parts[1]) {
		return parts[0] + ":" + parts[1]
	}
	return ""
}
