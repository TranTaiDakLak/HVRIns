package secapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	mrand "math/rand"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"

	"HVRIns/internal/httpx"
)

// doPost gửi gzip-compressed POST với headers đã có. Mirror của doPost
// trong steps.go — sẽ được unify ở Phase 3.
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

	data, err := httpx.ReadBody(resp.Body, 512*1024)
	respStr := string(data)

	// Detect checkpoint qua response header — port C# FacebookCheckpointDetectorUtils.
	if integrity := resp.Header.Get("X-Fb-Integrity-Required"); integrity != "" {
		if strings.Contains(strings.ToLower(integrity), "checkpoint") {
			respStr = `{"error":{"code":459,"message":"checkpointed"}}` + respStr
		}
	}
	if resp.Header.Get("X-Fb-Integrity-Requires-Login") != "" {
		respStr = `{"error":{"message":"checkpointed"}}` + respStr
	}

	if resp.StatusCode >= 400 {
		end := len(respStr)
		if end > 300 {
			end = 300
		}
		return respStr, fmt.Errorf("HTTP %d: %s", resp.StatusCode, respStr[:end])
	}
	return respStr, err
}

// qplInstanceID — random int64 tương đương C#: _create_INTERNAL__latency_qpl_instance_id().
func qplInstanceID() int64 {
	return int64(mrand.Intn(900000000) + 100000000)
}

// jsonEscape escape chuỗi để nhúng vào JSON string value.
func jsonEscape(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
