// proxypool.go — Load proxy list cho temp mail (port C# UseProxyTempmail).
//
// File: Config/Proxy/proxy_tempmail.txt (mỗi dòng 1 proxy: host:port hoặc host:port:user:pass)
// Khi verify với UseProxyTempMail=true → caller pick 1 proxy random từ pool này,
// truyền vào email.Options.ProxyOverride → email service dùng proxy riêng,
// không đụng proxy của account FB.
package email

import (
	"bufio"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	proxyTempMailDir   = "Config/Proxy"
	proxyTempMailFile  = "proxy_tempmail.txt"
	proxyRentMailFile  = "proxy_rentmail.txt"
	proxyGmailLegacy   = "proxy_gmail.txt" // legacy tên cũ — auto-migrate sang proxy_rentmail.txt nếu có
)

var (
	tempMailProxyMu    sync.RWMutex
	tempMailProxyList  []string
	tempMailProxyLoaded bool

	rentMailProxyMu     sync.RWMutex
	rentMailProxyList   []string
	rentMailProxyLoaded bool
)

// rentMailProviders — các provider rent mail CÓ hỗ trợ proxy per-request.
// Chỉ include những provider mà constructor đã accept proxyStr và an toàn
// để dùng proxy (không bị ban account API).
//
// KHÔNG include:
//   - dongvanfb, store1s, mail30s: constructor chưa có param proxy
//   - các rent provider có chính sách ban IP-proxy khác (sptmail, otpcheap, …)
//     (có thể bật thêm trong tương lai nếu user confirm)
var rentMailProviders = map[string]bool{
	"zeus-x":      true,
	"muamail":     true,
	"unlimitmail": true,
	"dongvanfb":   true,
	"store1s":     true,
	"mail30s":     true,
	"wmemail":     true,
}

// IsRentMailProvider trả true nếu provider dùng proxy pool rent mail
// (proxy_rentmail.txt) thay vì proxy pool temp mail.
// Dùng để verify layer biết nên pick từ pool nào.
func IsRentMailProvider(provider string) bool {
	return rentMailProviders[provider]
}

// LoadTempMailProxies đọc file Config/Proxy/proxy_tempmail.txt vào pool.
// Auto tạo file rỗng nếu chưa tồn tại (user biết path để paste proxy).
// Gọi khi app startup hoặc khi batch start nếu cần reload.
func LoadTempMailProxies() int {
	path := filepath.Join(proxyTempMailDir, proxyTempMailFile)

	// Ensure dir + file exists
	_ = os.MkdirAll(proxyTempMailDir, 0755)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Tạo file với hint
		if f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			_, _ = f.WriteString("# Proxy dành riêng cho Temp Mail (mỗi dòng 1 proxy)\n")
			_, _ = f.WriteString("# Format: host:port  HOẶC  host:port:user:pass\n")
			f.Close()
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}

	tempMailProxyMu.Lock()
	tempMailProxyList = lines
	tempMailProxyLoaded = true
	tempMailProxyMu.Unlock()
	return len(lines)
}

// PickTempMailProxy random 1 proxy từ pool. Return "" nếu pool rỗng.
// Thread-safe. Không consume (có thể pick trùng) — giống C# RandomItemInFile.
func PickTempMailProxy() string {
	tempMailProxyMu.RLock()
	defer tempMailProxyMu.RUnlock()
	// Lazy load nếu chưa init
	if !tempMailProxyLoaded {
		tempMailProxyMu.RUnlock()
		LoadTempMailProxies()
		tempMailProxyMu.RLock()
	}
	if len(tempMailProxyList) == 0 {
		return ""
	}
	return tempMailProxyList[rand.Intn(len(tempMailProxyList))]
}

// TempMailProxyCount trả về số proxy đang có trong pool.
func TempMailProxyCount() int {
	tempMailProxyMu.RLock()
	defer tempMailProxyMu.RUnlock()
	return len(tempMailProxyList)
}

// LoadRentMailProxies đọc file Config/Proxy/proxy_rentmail.txt vào pool.
// Auto-migrate: nếu proxy_rentmail.txt chưa tồn tại nhưng proxy_gmail.txt có
// → rename file legacy sang tên mới (giữ nguyên content user đã paste).
// Gọi khi app startup hoặc khi batch start.
func LoadRentMailProxies() int {
	_ = os.MkdirAll(proxyTempMailDir, 0755)

	newPath := filepath.Join(proxyTempMailDir, proxyRentMailFile)
	legacyPath := filepath.Join(proxyTempMailDir, proxyGmailLegacy)

	// Auto-migrate legacy file name nếu chỉ có file cũ.
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		if _, err2 := os.Stat(legacyPath); err2 == nil {
			_ = os.Rename(legacyPath, newPath)
		}
	}

	// Ensure file exists với hint cho user.
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		if f, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			_, _ = f.WriteString("# Proxy dành riêng cho Rent Mail (mỗi dòng 1 proxy)\n")
			_, _ = f.WriteString("# Chỉ áp dụng cho provider hỗ trợ: zeus-x, muamail, unlimitmail\n")
			_, _ = f.WriteString("# Format: host:port  HOẶC  host:port:user:pass\n")
			f.Close()
		}
	}

	f, err := os.Open(newPath)
	if err != nil {
		return 0
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}

	rentMailProxyMu.Lock()
	rentMailProxyList = lines
	rentMailProxyLoaded = true
	rentMailProxyMu.Unlock()
	return len(lines)
}

// PickRentMailProxy random 1 proxy từ rent mail pool.
// Nếu pool rỗng → fallback dùng chung với TempMail pool (proxy_tempmail.txt).
// Thread-safe. Không consume (có thể pick trùng) — giống PickTempMailProxy.
func PickRentMailProxy() string {
	rentMailProxyMu.RLock()
	if !rentMailProxyLoaded {
		rentMailProxyMu.RUnlock()
		LoadRentMailProxies()
		rentMailProxyMu.RLock()
	}
	if len(rentMailProxyList) > 0 {
		p := rentMailProxyList[rand.Intn(len(rentMailProxyList))]
		rentMailProxyMu.RUnlock()
		return p
	}
	rentMailProxyMu.RUnlock()
	// Pool RentMail rỗng → dùng chung TempMail pool.
	return PickTempMailProxy()
}

// RentMailProxyCount trả về số proxy đang có trong rent mail pool.
func RentMailProxyCount() int {
	rentMailProxyMu.RLock()
	defer rentMailProxyMu.RUnlock()
	return len(rentMailProxyList)
}
