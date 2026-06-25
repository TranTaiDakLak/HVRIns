// tempmailxxyz.go — tempmailx.xyz service (Laravel Livewire v2, stateful)
//
// Flow (xác nhận live qua agent 2026-06-19, OTP đọc đúng):
//  1. GET / (cookie jar) → server auto-assign địa chỉ + bind qua cookies. Scrape:
//     - email:    const email = 'rojtod@imail.sbs';
//     - csrf:     window.livewire_token = '...';
//     - wire:initial-data của component "frontend.app" (fingerprint + serverMemo)
//  2. POST /livewire/message/frontend.app  (JSON, X-CSRF-TOKEN + X-Livewire:true)
//     body: {fingerprint, serverMemo, updates:[syncEmail, fetchMessages]}
//     → response.serverMemo.data.messages = [{subject, content, ...}]
//     Phải lưu serverMemo trả về (đã update checksum) cho lần poll kế.
//
// KHÔNG cần key. Cloudflare KHÔNG challenge GET/fetchMessages (chỉ custom-username
// cần Turnstile — ta dùng địa chỉ auto-assign nên bỏ qua).
package temp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/cookiejar"
	"regexp"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const (
	tempMailXBaseURL = "https://tempmailx.xyz"
	tempMailXUA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
)

var (
	tmxEmailRe = regexp.MustCompile(`email\s*=\s*'([^']+@[^']+)'`)
	tmxTokenRe = regexp.MustCompile(`livewire_token\s*=\s*'([^']+)'`)
	tmxInitRe  = regexp.MustCompile(`wire:initial-data="([^"]*)"`)
)

// TempMailX implements email.Service cho tempmailx.xyz.
type TempMailX struct {
	client      *http.Client
	email       string
	csrfToken   string
	fingerprint json.RawMessage // opaque — echo nguyên trạng
	serverMemo  json.RawMessage // cập nhật sau mỗi response (checksum thay đổi)
	synced      bool            // đã gửi syncEmail chưa
}

// NewTempMailX tạo TempMailX service.
func NewTempMailX(proxyStr string) *TempMailX {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	c.Jar = jar
	return &TempMailX{client: c}
}

// CreateEmail: GET / → scrape email + token + initial-data của frontend.app.
func (t *TempMailX) CreateEmail(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", tempMailXBaseURL+"/", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", tempMailXUA)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tempmailx init: %w", err)
	}
	defer resp.Body.Close()
	htmlBody, _ := httpx.ReadBody(resp.Body, 1024*1024)
	hs := string(htmlBody)

	if m := tmxEmailRe.FindStringSubmatch(hs); len(m) >= 2 {
		t.email = strings.TrimSpace(m[1])
	}
	if m := tmxTokenRe.FindStringSubmatch(hs); len(m) >= 2 {
		t.csrfToken = m[1]
	}
	if t.email == "" || t.csrfToken == "" {
		return "", fmt.Errorf("tempmailx init: thiếu email/token (HTTP %d)", resp.StatusCode)
	}

	// Tìm wire:initial-data của component frontend.app.
	if !t.parseInitialData(hs) {
		return "", fmt.Errorf("tempmailx init: không tìm được initial-data frontend.app")
	}
	return t.email, nil
}

// parseInitialData tách fingerprint + serverMemo từ block initial-data của frontend.app.
func (t *TempMailX) parseInitialData(hs string) bool {
	for _, m := range tmxInitRe.FindAllStringSubmatch(hs, -1) {
		jsonStr := html.UnescapeString(m[1])
		var probe struct {
			Fingerprint struct {
				Name string `json:"name"`
			} `json:"fingerprint"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &probe); err != nil {
			continue
		}
		if probe.Fingerprint.Name != "frontend.app" {
			continue
		}
		var data struct {
			Fingerprint json.RawMessage `json:"fingerprint"`
			ServerMemo  json.RawMessage `json:"serverMemo"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
			continue
		}
		t.fingerprint = data.Fingerprint
		t.serverMemo = data.ServerMemo
		return true
	}
	return false
}

// GetEmail trả về địa chỉ đã tạo.
func (t *TempMailX) GetEmail() string { return t.email }

// Close no-op.
func (t *TempMailX) Close() {}

// WaitForCode poll OTP từ inbox.
func (t *TempMailX) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempmailx: chưa tạo email")
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if code, _ := t.pollOnce(ctx); code != "" {
			return code, nil
		}
		if attempt < maxRetry-1 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(intervalMs) * time.Millisecond):
			}
		}
	}
	return "", fmt.Errorf("tempmailx: không nhận được OTP sau %d lần thử", maxRetry)
}

type tmxUpdate struct {
	Type    string                 `json:"type"`
	Payload map[string]interface{} `json:"payload"`
}

func (t *TempMailX) pollOnce(ctx context.Context) (string, error) {
	var updates []tmxUpdate
	if !t.synced {
		updates = append(updates, tmxUpdate{
			Type:    "fireEvent",
			Payload: map[string]interface{}{"id": "sync1", "event": "syncEmail", "params": []interface{}{t.email}},
		})
	}
	updates = append(updates, tmxUpdate{
		Type:    "fireEvent",
		Payload: map[string]interface{}{"id": "fetch1", "event": "fetchMessages", "params": []interface{}{}},
	})

	reqBody, _ := json.Marshal(map[string]interface{}{
		"fingerprint": t.fingerprint,
		"serverMemo":  t.serverMemo,
		"updates":     updates,
	})

	req, err := http.NewRequestWithContext(ctx, "POST",
		tempMailXBaseURL+"/livewire/message/frontend.app", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", tempMailXUA)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("X-CSRF-TOKEN", t.csrfToken)
	req.Header.Set("X-Livewire", "true")
	req.Header.Set("Referer", tempMailXBaseURL+"/")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 512*1024)

	var lwResp struct {
		ServerMemo json.RawMessage `json:"serverMemo"`
	}
	if err := json.Unmarshal(body, &lwResp); err != nil {
		return "", nil
	}
	if len(lwResp.ServerMemo) > 0 {
		t.serverMemo = lwResp.ServerMemo // lưu memo mới (checksum update) cho lần sau
		t.synced = true
	}

	// Lấy messages từ serverMemo.data.messages
	var memo struct {
		Data struct {
			Messages []struct {
				Subject string `json:"subject"`
				Content string `json:"content"`
			} `json:"messages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(lwResp.ServerMemo, &memo); err != nil {
		return "", nil
	}
	for _, m := range memo.Data.Messages {
		if code := ExtractCode(m.Subject); code != "" {
			return code, nil
		}
		if code := ExtractCode(m.Content); code != "" {
			return code, nil
		}
	}
	return "", nil
}
