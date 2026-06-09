// dropmail.go — Dropmail.me service (GraphQL)
package temp

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const dropmailBaseURL = "https://dropmail.me"

// Dropmail implements email.Service cho dropmail.me
type Dropmail struct {
	client *http.Client
	token  string // random hex16 dùng cho endpoint
	sessID string // session id từ introduceSession
	email  string
}

// NewDropmail tạo Dropmail service.
// proxyStr: proxy URL, để trống nếu không dùng proxy.
func NewDropmail(proxyStr string) *Dropmail {
	client := proxy.CreateClient(proxyStr, 30*time.Second)
	return &Dropmail{client: client}
}

// CreateEmail tạo session qua GraphQL mutation.
func (d *Dropmail) CreateEmail(ctx context.Context) (string, error) {
	d.token = randomHex(16)

	query := `{"query":"mutation {introduceSession {id, expiresAt, addresses {address}}}"}`
	req, err := http.NewRequestWithContext(ctx, "POST",
		dropmailBaseURL+"/api/graphql/"+d.token,
		bytes.NewBufferString(query))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("dropmail create: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	var result struct {
		Data struct {
			IntroduceSession struct {
				ID        string `json:"id"`
				Addresses []struct {
					Address string `json:"address"`
				} `json:"addresses"`
			} `json:"introduceSession"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("dropmail create parse: %w — body: %.200s", err, body)
	}

	sess := result.Data.IntroduceSession
	if sess.ID == "" || len(sess.Addresses) == 0 {
		return "", fmt.Errorf("dropmail create: unexpected response: %s", string(body))
	}

	d.sessID = sess.ID
	d.email = sess.Addresses[0].Address
	return d.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (d *Dropmail) GetEmail() string { return d.email }

// Close cleanup resources.
func (d *Dropmail) Close() {}

// WaitForCode poll OTP từ dropmail inbox.
func (d *Dropmail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if d.sessID == "" {
		return "", fmt.Errorf("dropmail: chưa tạo email")
	}

	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		code, err := d.pollOnce(ctx)
		if err == nil && code != "" {
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

	return "", fmt.Errorf("dropmail: không nhận được OTP sau %d lần thử", maxRetry)
}

// pollOnce lấy inbox qua GraphQL query.
func (d *Dropmail) pollOnce(ctx context.Context) (string, error) {
	query := fmt.Sprintf(`{"query":"query {session(id:\"%s\"){mails{id,fromAddr,headerSubject,html}}}"}`, d.sessID)
	req, err := http.NewRequestWithContext(ctx, "POST",
		dropmailBaseURL+"/api/graphql/"+d.token,
		bytes.NewBufferString(query))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 0)

	var result struct {
		Data struct {
			Session struct {
				Mails []struct {
					ID            string `json:"id"`
					FromAddr      string `json:"fromAddr"`
					HeaderSubject string `json:"headerSubject"`
					HTML          string `json:"html"`
				} `json:"mails"`
			} `json:"session"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("dropmail inbox parse: %w", err)
	}

	for _, mail := range result.Data.Session.Mails {
		if code := ExtractCode(mail.HTML); code != "" {
			return code, nil
		}
	}
	return "", nil
}

// randomHex tạo chuỗi hex ngẫu nhiên n bytes (kết quả dài 2*n ký tự).
func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b) //nolint:gosec
	return hex.EncodeToString(b)
}
