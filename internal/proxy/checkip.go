// Package proxy — CheckIP lấy IP thực khi đi qua proxy
package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// reIP validate chuỗi có phải IPv4 không (đơn giản)
var reIP = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)

// Endpoint constants
const (
	ipApiURL     = "http://ip-api.com/json/?fields=status,country,countryCode,city,isp,query"
	adspowerURL  = "https://ip-scan.adspower.net/sys/config/ip/get-visitor-ip"
	lumtestURL   = "http://lumtest.com/myip.json"
	ipinfoURL    = "https://ipinfo.io/json"
)

// API constants — khớp với frontend API_CHECK_IP_PROVIDERS
const (
	APICheckIpAuto     = 0 // ip-api.com (Auto)
	APICheckIpAdsPower = 1
	APICheckIpLuna     = 2
	APICheckIpInfoIO   = 3
	APICheckIpNordVPN  = 4
)

type ipApiResp struct {
	Status      string `json:"status"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	City        string `json:"city"`
	ISP         string `json:"isp"`
	Query       string `json:"query"` // IP
}

type adspowerResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		IP     string `json:"ip"`
		IpData struct {
			Country string `json:"country"`
		} `json:"ip_data"`
	} `json:"data"`
}

type lumtestResp struct {
	IP      string `json:"ip"`
	Country string `json:"country"` // ISO alpha-2 uppercase
}

type ipinfoResp struct {
	IP      string `json:"ip"`
	Country string `json:"country"` // ISO alpha-2 uppercase
}

// CheckIP request qua proxy để lấy IP thực của kết nối.
// Trả về dạng "89.200.217.100/vn" (IP/country), fallback sang plain IP nếu không lấy được country.
//
// preferredAPI: API ưu tiên (khớp frontend apiCheckIp):
//
//	0 = Auto/ip-api, 1 = AdsPower, 2 = Luna, 3 = IpInfo.io, 4 = NordVPN
//
// Fallback chain luôn là: preferred → ip-api → adspower → luna → ipify (plain IP)
// Các bước trùng nhau được bỏ qua.
func CheckIP(ctx context.Context, proxyStr string, preferredAPI int) (string, error) {
	client := CreateClient(proxyStr, 6*time.Second)

	type fetcherFn func(context.Context, *http.Client) (string, error)

	byAPI := map[int]fetcherFn{
		APICheckIpAuto:     fetchIpApi,
		APICheckIpAdsPower: fetchAdsPower,
		APICheckIpLuna:     fetchLunaProxy,
		APICheckIpInfoIO:   fetchIpInfoIO,
		APICheckIpNordVPN:  fetchIpInfoIO, // NordVPN không có public JSON endpoint riêng → dùng ipinfo
	}

	// Chain: preferred → ip-api → adspower → luna, dedup thứ tự
	order := []int{preferredAPI, APICheckIpAuto, APICheckIpAdsPower, APICheckIpLuna}
	seen := make(map[int]bool, len(order))
	for _, api := range order {
		if seen[api] {
			continue
		}
		seen[api] = true
		fn, ok := byAPI[api]
		if !ok {
			continue
		}
		if result, err := fn(ctx, client); err == nil && result != "" {
			return result, nil
		}
	}

	// Absolute fallback: plain IP không có country
	if ip, err := fetchPlainIP(ctx, client, "https://api.ipify.org"); err == nil && ip != "" {
		return ip, nil
	}

	return "", fmt.Errorf("không lấy được IP qua proxy")
}

// fetchIpApi request ip-api.com, trả về "IP/countryCode" (vd: "213.196.103.42/rs").
func fetchIpApi(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", ipApiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/134.0.0.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}

	var data ipApiResp
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	if data.Status != "success" || data.Query == "" {
		return "", fmt.Errorf("ip-api fail: status=%s", data.Status)
	}

	cc := strings.ToLower(strings.TrimSpace(data.CountryCode))
	if cc != "" {
		return data.Query + "/" + cc, nil
	}
	return data.Query, nil
}

// fetchAdsPower request adspower API, trả về "IP/country" (vd: "116.103.130.43/vn")
func fetchAdsPower(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", adspowerURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}

	var data adspowerResp
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	if data.Code != 0 || data.Data.IP == "" {
		return "", fmt.Errorf("adspower error: %s", data.Msg)
	}

	ip := data.Data.IP
	country := strings.ToLower(strings.TrimSpace(data.Data.IpData.Country))
	if country != "" {
		return ip + "/" + country, nil
	}
	return ip, nil
}

// fetchLunaProxy request lumtest.com (Bright Data/Luna Proxy), trả về "IP/country".
func fetchLunaProxy(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", lumtestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/134.0.0.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}

	var data lumtestResp
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	if data.IP == "" {
		return "", fmt.Errorf("lumtest: empty IP")
	}

	cc := strings.ToLower(strings.TrimSpace(data.Country))
	if cc != "" {
		return data.IP + "/" + cc, nil
	}
	return data.IP, nil
}

// fetchIpInfoIO request ipinfo.io, trả về "IP/country".
func fetchIpInfoIO(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", ipinfoURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/134.0.0.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}

	var data ipinfoResp
	if err := json.Unmarshal(body, &data); err != nil {
		return "", err
	}
	if data.IP == "" {
		return "", fmt.Errorf("ipinfo: empty IP")
	}

	cc := strings.ToLower(strings.TrimSpace(data.Country))
	if cc != "" {
		return data.IP + "/" + cc, nil
	}
	return data.IP, nil
}

// fetchPlainIP lấy IP text thuần từ endpoint đơn giản.
// Validate response phải là IPv4 thật — tránh lưu JSON error của proxy.
func fetchPlainIP(ctx context.Context, client *http.Client, endpoint string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(body))
	if !reIP.MatchString(ip) {
		return "", fmt.Errorf("response không phải IPv4: %s", ip[:min(len(ip), 40)])
	}
	return ip, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
