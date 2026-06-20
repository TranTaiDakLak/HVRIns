// client_test.go — Unit tests cho FormatProxyURL, ShortDisplay, RenderSessionIfIsProxyServer.
// Thay thế cmd/proxycheck, cmd/proxytest (đã xoá Sprint 01 D2).
// Không dùng credential thật — chỉ input giả lập.
package proxy

import (
	"net/url"
	"strings"
	"testing"
)

// ── FormatProxyURL ───────────────────────────────────────────────────────────

func TestFormatProxyURL_Formats(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantHost string // kỳ vọng host:port trong URL kết quả
		wantUser string // kỳ vọng username (rỗng nếu không có auth)
		wantNil  bool   // kỳ vọng kết quả rỗng
	}{
		{
			name:     "empty",
			input:    "",
			wantNil:  true,
		},
		{
			name:     "ip:port — no auth",
			input:    "1.2.3.4:8080",
			wantHost: "1.2.3.4:8080",
		},
		{
			name:     "host:port:user:pass — default format",
			input:    "1.2.3.4:8080:myuser:mypass",
			wantHost: "1.2.3.4:8080",
			wantUser: "myuser",
		},
		{
			name:     "user:pass:host:port — NiceProxy format",
			input:    "myuser:mypass:1.2.3.4:8080",
			wantHost: "1.2.3.4:8080",
			wantUser: "myuser",
		},
		{
			name:     "user:pass@host:port — no scheme",
			input:    "myuser:mypass@1.2.3.4:8080",
			wantHost: "1.2.3.4:8080",
			wantUser: "myuser",
		},
		{
			name:     "http://user:pass@host:port — full URL",
			input:    "http://myuser:mypass@1.2.3.4:8080",
			wantHost: "1.2.3.4:8080",
			wantUser: "myuser",
		},
		{
			name:     "socks5://user:pass@host:port",
			input:    "socks5://myuser:mypass@1.2.3.4:1080",
			wantHost: "1.2.3.4:1080",
			wantUser: "myuser",
		},
		{
			name:     "host:port only — no auth fields",
			input:    "proxy.example.com:3128",
			wantHost: "proxy.example.com:3128",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatProxyURL(tt.input)
			if tt.wantNil {
				if got != "" {
					t.Errorf("FormatProxyURL(%q) = %q; want empty", tt.input, got)
				}
				return
			}
			if got == "" {
				t.Fatalf("FormatProxyURL(%q) = empty; want non-empty", tt.input)
			}
			u, err := url.Parse(got)
			if err != nil {
				t.Fatalf("result %q is not a valid URL: %v", got, err)
			}
			if u.Host != tt.wantHost {
				t.Errorf("host = %q; want %q", u.Host, tt.wantHost)
			}
			if tt.wantUser != "" {
				if u.User == nil || u.User.Username() != tt.wantUser {
					t.Errorf("user = %q; want %q", u.User, tt.wantUser)
				}
			}
		})
	}
}

func TestFormatProxyURL_SchemePreserved(t *testing.T) {
	cases := []struct {
		input      string
		wantScheme string
	}{
		{"http://u:p@h:8080", "http"},
		{"https://u:p@h:8080", "https"},
		{"socks5://u:p@h:1080", "socks5"},
		{"socks5h://u:p@h:1080", "socks5h"},
		{"socks4://h:1080", "socks4"},
	}
	for _, c := range cases {
		got := FormatProxyURL(c.input)
		u, err := url.Parse(got)
		if err != nil || u.Scheme != c.wantScheme {
			t.Errorf("FormatProxyURL(%q) scheme = %q; want %q (full: %q)", c.input, u.Scheme, c.wantScheme, got)
		}
	}
}

// ── ShortDisplay ─────────────────────────────────────────────────────────────

func TestShortDisplay(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"1.2.3.4:8080", "1.2.3.4:8080"},
		{"1.2.3.4:8080:user:pass", "1.2.3.4:8080"},
		{"user:pass@1.2.3.4:8080", "1.2.3.4:8080"},
		{"http://user:pass@1.2.3.4:8080", "1.2.3.4:8080"},
		{"socks5://user:pass@proxy.host:1080", "proxy.host:1080"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ShortDisplay(tt.input)
			if got != tt.want {
				t.Errorf("ShortDisplay(%q) = %q; want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ── RenderSessionIfIsProxyServer — static proxy (không rotate) ───────────────

func TestRenderSession_StaticProxy_Unchanged(t *testing.T) {
	// Static proxy: không có pattern rotate → trả về y nguyên
	cases := []string{
		"1.2.3.4:8080",
		"1.2.3.4:8080:plainuser:plainpass",
		"http://plainuser:plainpass@1.2.3.4:8080",
		"plainuser:plainpass@1.2.3.4:8080",
	}
	for _, c := range cases {
		got := RenderSessionIfIsProxyServer(c)
		if got != c {
			t.Errorf("static proxy %q should be unchanged; got %q", c, got)
		}
	}
}

func TestRenderSession_Empty(t *testing.T) {
	if got := RenderSessionIfIsProxyServer(""); got != "" {
		t.Errorf("RenderSession(\"\") = %q; want empty", got)
	}
}

// ── RenderSessionIfIsProxyServer — rotate proxy (session thay đổi) ───────────

func TestRenderSession_SSIDProxy_SessionRotated(t *testing.T) {
	// NiceProxy -ssid- format: mỗi lần gọi → session ID mới
	proxy := "soialvin_Y4uV-ssid-n1nlmUkcxa:pass@proxy.niceproxy.io:1234"
	got1 := RenderSessionIfIsProxyServer(proxy)
	got2 := RenderSessionIfIsProxyServer(proxy)

	if got1 == proxy {
		t.Error("ssid proxy: result should differ from input (session should rotate)")
	}
	if got1 == got2 {
		t.Error("ssid proxy: two consecutive calls should produce different session IDs")
	}
	if !strings.Contains(got1, "-ssid-") {
		t.Errorf("ssid proxy: result should contain -ssid-; got %q", got1)
	}
}

func TestRenderSession_ZoneProxy_SessionAdded(t *testing.T) {
	// Zone proxy (711proxy etc.) -zone-custom: mỗi lần gọi thêm -session-XXXX
	proxy := "USER255485-zone-custom:pass@gate.711proxy.io:8080"
	got := RenderSessionIfIsProxyServer(proxy)

	if got == proxy {
		t.Error("zone proxy: result should differ from input")
	}
	if !strings.Contains(strings.ToLower(got), "-session-") {
		t.Errorf("zone proxy: result should contain -session-; got %q", got)
	}
}

func TestRenderSession_ProxyShare_SessionRotated(t *testing.T) {
	// ProxyShare _session- format
	proxy := "ps-k3b0n3zt2nnu_area-US_session-2HNUFU64BE_life-5:pass@gate.proxyshare.io:3128"
	got1 := RenderSessionIfIsProxyServer(proxy)
	got2 := RenderSessionIfIsProxyServer(proxy)

	if got1 == proxy {
		t.Error("proxyshare: result should differ from input")
	}
	if got1 == got2 {
		t.Error("proxyshare: two calls should produce different sessions")
	}
}

// ── isPort ───────────────────────────────────────────────────────────────────

func TestIsPort(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"80", true},
		{"8080", true},
		{"65535", true},
		{"1", true},
		{"0", false},     // 0 không hợp lệ
		{"65536", false}, // vượt max
		{"abc", false},
		{"", false},
		{"123456", false}, // quá dài (>5 chữ số)
		{"8 80", false},   // có khoảng trắng
	}

	for _, tt := range tests {
		if got := isPort(tt.s); got != tt.want {
			t.Errorf("isPort(%q) = %v; want %v", tt.s, got, tt.want)
		}
	}
}
