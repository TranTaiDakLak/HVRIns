// tempamail.go — tempamail.com service (form-encoded REST API)
//
// Flow (xác nhận qua research 2026-06-19):
//   1. POST https://api.tempamail.com/webapp/client/create
//      body: app_uuid=a5x-cj6a-ka1q (form-encoded)
//      → {client:{uuid}, email:{id,address}}
//   2. POST https://api.tempamail.com/webapp/messages
//      body: uuid={client_uuid}&selected_email_id={email_id}&known_message_id=
//      → {messages:[{subject,body,html}]}
//
// KHÔNG cần key. Content-Type: application/x-www-form-urlencoded.
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const (
	tempAmailAPIBase = "https://api.tempamail.com/webapp"
	tempAmailAppUUID = "a5x-cj6a-ka1q" // hardcoded trong page JS
	tempAmailOrigin  = "https://tempamail.com"
)

// TempAmail implements email.Service cho tempamail.com.
type TempAmail struct {
	client     *http.Client
	email      string
	clientUUID string
	emailID    int
}

// NewTempAmail tạo TempAmail service.
func NewTempAmail(proxyStr string) *TempAmail {
	jar, _ := cookiejar.New(nil)
	c := proxy.CreateClient(proxyStr, 30*time.Second)
	c.Jar = jar
	return &TempAmail{client: c}
}

func (t *TempAmail) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", tempAmailOrigin)
	req.Header.Set("Referer", tempAmailOrigin+"/")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
}

func (t *TempAmail) postForm(ctx context.Context, path string, values url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST",
		tempAmailAPIBase+path, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	t.setHeaders(req)
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return httpx.ReadBody(resp.Body, 256*1024)
}

// CreateEmail: POST /client/create → client.uuid + email.id + email.address.
func (t *TempAmail) CreateEmail(ctx context.Context) (string, error) {
	body, err := t.postForm(ctx, "/client/create", url.Values{
		"app_uuid": {tempAmailAppUUID},
	})
	if err != nil {
		return "", fmt.Errorf("tempamail create: %w", err)
	}

	var result struct {
		Client struct {
			UUID string `json:"uuid"`
		} `json:"client"`
		Email struct {
			ID      int    `json:"id"`
			Address string `json:"address"`
		} `json:"email"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Client.UUID == "" {
		return "", fmt.Errorf("tempamail create: parse error — body: %.200s", body)
	}
	t.clientUUID = result.Client.UUID
	t.emailID = result.Email.ID
	t.email = result.Email.Address
	return t.email, nil
}

// GetEmail trả về địa chỉ đã tạo.
func (t *TempAmail) GetEmail() string { return t.email }

// Close no-op.
func (t *TempAmail) Close() {}

// WaitForCode poll OTP từ inbox.
func (t *TempAmail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if t.email == "" {
		return "", fmt.Errorf("tempamail: chưa tạo email")
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
	return "", fmt.Errorf("tempamail: không nhận được OTP sau %d lần thử", maxRetry)
}

func (t *TempAmail) pollOnce(ctx context.Context) (string, error) {
	body, err := t.postForm(ctx, "/messages", url.Values{
		"uuid":              {t.clientUUID},
		"selected_email_id": {strconv.Itoa(t.emailID)},
		"known_message_id":  {""},
	})
	if err != nil {
		return "", err
	}

	var result struct {
		Messages []struct {
			Subject string `json:"subject"`
			Body    string `json:"body"`
			HTML    string `json:"html"`
			Text    string `json:"text"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", nil
	}
	for _, m := range result.Messages {
		if code := ExtractCode(m.Subject); code != "" {
			return code, nil
		}
		if code := ExtractCode(m.Body); code != "" {
			return code, nil
		}
		if code := ExtractCode(m.HTML); code != "" {
			return code, nil
		}
		if code := ExtractCode(m.Text); code != "" {
			return code, nil
		}
	}
	return "", nil
}
