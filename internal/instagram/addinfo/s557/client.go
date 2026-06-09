package addinfo

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/google/uuid"

	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

func newClient(proxyStr string) (tls_client.HttpClient, error) {
	jar := tls_client.NewCookieJar()
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Okhttp4Android13),
		tls_client.WithCookieJar(jar),
		tls_client.WithInsecureSkipVerify(),
		tls_client.WithNotFollowRedirects(),
	}
	if proxyStr != "" {
		if f := proxy.FormatProxyURL(proxyStr); f != "" {
			opts = append(opts, tls_client.WithProxyUrl(f))
		}
	}
	return tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
}

func doPost(ctx context.Context, client tls_client.HttpClient, targetURL, body string, headers [][2]string) (string, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(body)); err != nil {
		return "", fmt.Errorf("gzip write: %v", err)
	}
	if err := gz.Close(); err != nil {
		return "", fmt.Errorf("gzip close: %v", err)
	}

	req, err := fhttp.NewRequestWithContext(ctx, "POST", targetURL, &buf)
	if err != nil {
		return "", err
	}

	order := make([]string, 0, len(headers))
	for _, kv := range headers {
		req.Header[kv[0]] = []string{kv[1]}
		order = append(order, kv[0])
	}
	req.Header[fhttp.HeaderOrderKey] = order

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP error: %v", err)
	}
	defer resp.Body.Close()

	data, _ := httpx.ReadBody(resp.Body, 512*1024)
	respStr := string(data)
	if resp.StatusCode >= 400 {
		n := len(respStr)
		if n > 300 {
			n = 300
		}
		return respStr, fmt.Errorf("HTTP %d: %s", resp.StatusCode, respStr[:n])
	}
	return respStr, nil
}

// relayHeaders builds the standard headers for graph.facebook.com RelayModern calls.
func relayHeaders(friendlyName, machineID, deviceID string, extra ...[2]string) [][2]string {
	usdid := genUSDID()
	h := [][2]string{
		{"x-fb-rmd", "state=URL_ELIGIBLE"},
		{"x-fb-friendly-name", "RelayFBNetwork_" + friendlyName},
		{"x-meta-zca", "empty_token"},
		{"user-agent", func() string {
			if ua := fakeinfo.RandomUAFromPool(fakeinfo.UAKindAndroid); ua != "" {
				return ua
			}
			return fakeinfo.BuildAndroidUAWithOpts(fakeinfo.RandomDeviceProfile(), "en_US", "", "", "", false, false)
		}()},
		{"x-fb-connection-type", "WIFI"},
		{"x-tigon-is-retry", "False"},
		{"x-fb-net-hni", "45204"},
		{"x-fb-integrity-machine-id", machineID},
		{"x-fb-request-analytics-tags", `{"network_tags":{"product":"350685531728","purpose":"none","retry_attempt":"0"},"application_tags":"unknown"}`},
		{"x-fb-sim-hni", "45204"},
		{"content-encoding", "gzip"},
		{"content-type", "application/x-www-form-urlencoded"},
		{"x-meta-usdid", usdid},
		{"app-scope-id-header", uuid.New().String()},
		{"x-zero-f-device-id", deviceID},
		{"priority", "u=3, i"},
		{"x-fb-device-group", "2610"},
		{"x-fb-network-properties", "Wifi;Validated;"},
		{"accept-encoding", "zstd, gzip, deflate"},
		{"x-fb-http-engine", "Tigon/Liger"},
		{"x-fb-client-ip", "True"},
		{"x-fb-server-cluster", "True"},
	}
	h = append(h, extra...)
	return h
}

func genUSDID() string {
	id := uuid.New().String()
	ts := fmt.Sprintf("%d", time.Now().Unix())
	payload := id + "." + ts
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return payload
	}
	hash := sha256.Sum256([]byte(payload))
	sig, _ := ecdsa.SignASN1(rand.Reader, key, hash[:])
	return payload + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func newUUID() string {
	return uuid.New().String()
}
