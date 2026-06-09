// priyoemail.go — free.priyo.email service (REST + API key)
// Port từ C# PriyoEmailAPI. Yêu cầu API key từ https://v3.priyo.email (free tier 100k requests/month).
package temp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"HVRIns/internal/httpx"
	"HVRIns/internal/proxy"
)

const priyoEmailBaseURL = "https://free.priyo.email"

// PriyoEmail implements email.Service cho priyo.email.
// apiKey: bắt buộc — lấy từ https://v3.priyo.email
type PriyoEmail struct {
	client *http.Client
	apiKey string
	email  string
}

// NewPriyoEmail tạo PriyoEmail service. apiKey bắt buộc.
func NewPriyoEmail(apiKey, proxyStr string) *PriyoEmail {
	return &PriyoEmail{
		client: proxy.CreateClient(proxyStr, 30*time.Second),
		apiKey: apiKey,
	}
}

// CreateEmail gọi /api/random-email/{apiKey}.
func (p *PriyoEmail) CreateEmail(ctx context.Context) (string, error) {
	if p.apiKey == "" {
		return "", fmt.Errorf("priyoemail: missing API key")
	}
	req, _ := http.NewRequestWithContext(ctx, "GET",
		priyoEmailBaseURL+"/api/random-email/"+url.PathEscape(p.apiKey), nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("priyoemail: %w", err)
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 64*1024)
	var result struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("priyoemail parse: %w — body: %.200s", err, body)
	}
	if result.Email == "" {
		return "", fmt.Errorf("priyoemail: empty email — body: %.200s", body)
	}
	p.email = result.Email
	return p.email, nil
}

// GetEmail trả về địa chỉ email đã tạo.
func (p *PriyoEmail) GetEmail() string { return p.email }

// Close no-op.
func (p *PriyoEmail) Close() {}

// WaitForCode poll OTP.
func (p *PriyoEmail) WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error) {
	if maxRetry == 0 {
		maxRetry = 12
	}
	if intervalMs == 0 {
		intervalMs = 2000
	}
	if p.email == "" {
		return "", fmt.Errorf("priyoemail: chưa tạo email")
	}
	for attempt := 0; attempt < maxRetry; attempt++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		if code, _ := p.pollOnce(ctx); code != "" {
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
	return "", fmt.Errorf("priyoemail: không nhận được OTP sau %d lần thử", maxRetry)
}

func (p *PriyoEmail) pollOnce(ctx context.Context) (string, error) {
	msgsURL := priyoEmailBaseURL + "/api/messages/" +
		url.PathEscape(p.email) + "/" + url.PathEscape(p.apiKey)
	req, _ := http.NewRequestWithContext(ctx, "GET", msgsURL, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	// Body có thể là array trực tiếp hoặc {messages: [...]} hoặc {data: [...]}.
	var msgs []struct {
		ID        string `json:"id"`
		FromEmail string `json:"from_email"`
		From      string `json:"from"`
		Subject   string `json:"subject"`
	}
	if err := json.Unmarshal(body, &msgs); err != nil {
		var wrapper struct {
			Messages []struct {
				ID        string `json:"id"`
				FromEmail string `json:"from_email"`
				From      string `json:"from"`
				Subject   string `json:"subject"`
			} `json:"messages"`
			Data []struct {
				ID        string `json:"id"`
				FromEmail string `json:"from_email"`
				From      string `json:"from"`
				Subject   string `json:"subject"`
			} `json:"data"`
		}
		if err2 := json.Unmarshal(body, &wrapper); err2 != nil {
			return "", err
		}
		if len(wrapper.Messages) > 0 {
			for _, m := range wrapper.Messages {
				msgs = append(msgs, struct {
					ID        string `json:"id"`
					FromEmail string `json:"from_email"`
					From      string `json:"from"`
					Subject   string `json:"subject"`
				}(m))
			}
		} else {
			for _, m := range wrapper.Data {
				msgs = append(msgs, struct {
					ID        string `json:"id"`
					FromEmail string `json:"from_email"`
					From      string `json:"from"`
					Subject   string `json:"subject"`
				}(m))
			}
		}
	}

	for _, msg := range msgs {
		content, _ := p.getMessage(ctx, msg.ID)
		if content == "" {
			content = msg.Subject
		}
		if code := ExtractCode(content); code != "" {
			return code, nil
		}
	}
	return "", nil
}

func (p *PriyoEmail) getMessage(ctx context.Context, id string) (string, error) {
	msgURL := priyoEmailBaseURL + "/api/message/" + url.PathEscape(id) + "/" + url.PathEscape(p.apiKey)
	req, _ := http.NewRequestWithContext(ctx, "GET", msgURL, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := httpx.ReadBody(resp.Body, 256*1024)

	var result struct {
		Data struct {
			Content string `json:"content"`
			Body    string `json:"body"`
		} `json:"data"`
		Content string `json:"content"`
		Body    string `json:"body"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.Data.Content != "" {
		return result.Data.Content, nil
	}
	if result.Data.Body != "" {
		return result.Data.Body, nil
	}
	if result.Content != "" {
		return result.Content, nil
	}
	return result.Body, nil
}
