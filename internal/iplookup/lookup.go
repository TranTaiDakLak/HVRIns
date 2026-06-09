// Package iplookup — IP address information lookup
// Mapping từ C#: IIPLookupAPI + IpAddressInfo model
//
// Looks up geolocation and carrier info for an IP address (or current outbound IP).
// Used to determine account registration country and proxy validation.
//
// TODO: Port from C# IIPLookupAPI — integrate with ip-api.com or similar service,
// cache results, handle proxy-aware lookups.
package iplookup

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"HVRIns/internal/proxy"
)

// IpInfo holds geolocation and carrier info for an IP address.
// Mapping từ C#: IpAddressInfo
type IpInfo struct {
	IP          string `json:"query"`
	Country     string `json:"country"`
	CountryCode string `json:"countryCode"`
	Region      string `json:"regionName"`
	City        string `json:"city"`
	ISP         string `json:"isp"`
	Org         string `json:"org"`
	Timezone    string `json:"timezone"`
	Mobile      bool   `json:"mobile"`
	Proxy       bool   `json:"proxy"`
	Hosting     bool   `json:"hosting"`
	Status      string `json:"status"`
}

// Lookup implements the IP lookup contract.
// Mapping từ C#: IIPLookupAPI
type Lookup struct {
	proxyStr string
	timeout  time.Duration
}

// New creates a Lookup using the given proxy for outbound requests.
func New(proxyStr string) *Lookup {
	return &Lookup{proxyStr: proxyStr, timeout: 15 * time.Second}
}

// LookupIP fetches geolocation info for the given IP (or "" for the outbound IP).
// Uses ip-api.com JSON endpoint.
func (l *Lookup) LookupIP(ctx context.Context, ip string) (*IpInfo, error) {
	url := "http://ip-api.com/json"
	if ip != "" {
		url = fmt.Sprintf("http://ip-api.com/json/%s", ip)
	}
	url += "?fields=status,message,country,countryCode,regionName,city,isp,org,query,timezone,mobile,proxy,hosting"

	client := proxy.CreateClient(l.proxyStr, l.timeout)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("iplookup: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	var info IpInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("iplookup: parse: %w", err)
	}
	if info.Status != "success" {
		return nil, fmt.Errorf("iplookup: api returned status=%q", info.Status)
	}
	return &info, nil
}
