// Package httpclient — HTTP client with full cookie/header management for Facebook API
// Mapping từ C#: IHttpRequestClient + HttpRequestClient implementation
//
// Provides a stateful HTTP client that maintains cookies, supports proxy routing,
// and exposes Get/PostForm/PostRaw methods matching the C# interface pattern.
//
// The key difference from Go's standard http.Client: this client maintains a
// persistent cookie jar across requests (like a browser session) and exposes
// typed header management methods.
//
// TODO: Port full implementation from C# HttpRequestClient —
//   cookie jar persistence, Referer tracking, CSRF token management,
//   connection reuse with idle connection timeout.
package httpclient

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"HVRIns/internal/proxy"
)

// Client is a stateful HTTP client with session cookies and proxy support.
// Mapping từ C#: IHttpRequestClient
type Client struct {
	httpClient *http.Client
	jar        *cookiejar.Jar
	userAgent  string
	proxyStr   string
}

// New creates a Client with the given proxy and timeout.
// proxyStr format: "host:port:user:pass" or "" for direct.
func New(proxyStr string, timeout time.Duration) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	base := proxy.CreateClient(proxyStr, timeout)
	base.Jar = jar
	return &Client{
		httpClient: base,
		jar:        jar,
		proxyStr:   proxyStr,
	}, nil
}

// SetUserAgent sets the User-Agent header for all subsequent requests.
func (c *Client) SetUserAgent(ua string) {
	c.userAgent = ua
}

// Get performs a GET request and returns the response body as a string.
// TODO: implement full cookie/redirect/header handling
func (c *Client) Get(ctx context.Context, targetURL string, headers map[string]string) (string, int, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return "", 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	// TODO: read body with size limit
	return "", resp.StatusCode, nil
}

// PostForm performs a POST with URL-encoded form body.
// TODO: implement full cookie/header/redirect handling
func (c *Client) PostForm(ctx context.Context, targetURL string, formBody string, headers map[string]string) (string, int, error) {
	// Placeholder — TODO: implement
	return "", 0, nil
}

// PostRaw performs a POST with a raw body and specified content type.
// TODO: implement full cookie/header/redirect handling
func (c *Client) PostRaw(ctx context.Context, targetURL string, body []byte, contentType string, headers map[string]string) (string, int, error) {
	// Placeholder — TODO: implement
	return "", 0, nil
}

// GetCookieValue returns the value of a named cookie for the given URL.
func (c *Client) GetCookieValue(targetURL, name string) string {
	u, err := url.Parse(targetURL)
	if err != nil {
		return ""
	}
	for _, cookie := range c.jar.Cookies(u) {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}

// SetCookie injects a cookie into the jar for the given URL.
func (c *Client) SetCookie(targetURL, name, value string) {
	u, err := url.Parse(targetURL)
	if err != nil {
		return
	}
	c.jar.SetCookies(u, []*http.Cookie{{Name: name, Value: value}})
}
