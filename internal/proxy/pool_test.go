// pool_test.go — Unit tests cho Pool (round-robin) và Manager.IsConfigured()
package proxy

import (
	"sync"
	"testing"
)

// ── Pool.NewPool ─────────────────────────────────────────────────────────────

func TestNewPool_ParsesLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantLen  int
	}{
		{"single proxy",       "1.2.3.4:8080",                        1},
		{"multiple proxies",   "1.2.3.4:8080\n5.6.7.8:3128",          2},
		{"empty lines skipped","1.2.3.4:8080\n\n\n5.6.7.8:3128\n",   2},
		{"whitespace trimmed", "  1.2.3.4:8080  \n  5.6.7.8:3128  ", 2},
		{"empty input",        "",                                      0},
		{"only newlines",      "\n\n\n",                               0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPool(tt.input)
			if p.Len() != tt.wantLen {
				t.Errorf("Len() = %d; want %d", p.Len(), tt.wantLen)
			}
		})
	}
}

// ── Pool.Next (round-robin) ──────────────────────────────────────────────────

func TestPool_Next_RoundRobin(t *testing.T) {
	p := NewPool("a\nb\nc")

	sequence := []string{"a", "b", "c", "a", "b", "c", "a"}
	for i, want := range sequence {
		got := p.Next()
		if got != want {
			t.Errorf("call %d: Next() = %q; want %q", i+1, got, want)
		}
	}
}

func TestPool_Next_SingleProxy(t *testing.T) {
	p := NewPool("only-proxy")
	for range 5 {
		if got := p.Next(); got != "only-proxy" {
			t.Errorf("Next() = %q; want %q", got, "only-proxy")
		}
	}
}

func TestPool_Next_EmptyPool(t *testing.T) {
	p := NewPool("")
	if got := p.Next(); got != "" {
		t.Errorf("Next() on empty pool = %q; want empty string", got)
	}
}

func TestPool_Next_Concurrent(t *testing.T) {
	const workers = 100
	p := NewPool("proxy1\nproxy2\nproxy3")

	var wg sync.WaitGroup
	results := make([]string, workers)

	for i := range workers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = p.Next()
		}(i)
	}
	wg.Wait()

	// Tất cả kết quả phải là proxy hợp lệ
	valid := map[string]bool{"proxy1": true, "proxy2": true, "proxy3": true}
	for i, r := range results {
		if !valid[r] {
			t.Errorf("worker %d got invalid proxy %q", i, r)
		}
	}
}

// ── Pool.Fixed ────────────────────────────────────────────────────────────────

func TestPool_Fixed_AlwaysFirst(t *testing.T) {
	p := NewPool("first\nsecond\nthird")
	for range 5 {
		if got := p.Fixed(); got != "first" {
			t.Errorf("Fixed() = %q; want %q", got, "first")
		}
	}
}

func TestPool_Fixed_EmptyPool(t *testing.T) {
	p := NewPool("")
	if got := p.Fixed(); got != "" {
		t.Errorf("Fixed() on empty pool = %q; want empty string", got)
	}
}

// ── Manager.IsConfigured ─────────────────────────────────────────────────────

func TestManager_IsConfigured(t *testing.T) {
	tests := []struct {
		name   string
		cfg    ManagerConfig
		want   bool
	}{
		{
			name: "provider none → not configured",
			cfg:  ManagerConfig{Provider: "none"},
			want: false,
		},
		{
			name: "provider empty → not configured",
			cfg:  ManagerConfig{Provider: ""},
			want: false,
		},
		{
			name: "hma → not configured (system-wide)",
			cfg:  ManagerConfig{Provider: "hma"},
			want: false,
		},
		{
			name: "fpt → not configured (dial-up)",
			cfg:  ManagerConfig{Provider: "fpt"},
			want: false,
		},
		{
			name: "xproxy → not configured",
			cfg:  ManagerConfig{Provider: "xproxy"},
			want: false,
		},
		{
			name: "proxy with empty list → not configured",
			cfg:  ManagerConfig{Provider: "proxy", ProxyList: ""},
			want: false,
		},
		{
			name: "proxy with list → configured",
			cfg:  ManagerConfig{Provider: "proxy", ProxyList: "1.2.3.4:8080"},
			want: true,
		},
		{
			name: "proxy_fixed with list → configured",
			cfg:  ManagerConfig{Provider: "proxy_fixed", ProxyList: "1.2.3.4:8080"},
			want: true,
		},
		{
			name: "proxy_fixed with empty list → not configured",
			cfg:  ManagerConfig{Provider: "proxy_fixed", ProxyList: ""},
			want: false,
		},
		{
			name: "tinsoft with keys → configured",
			cfg:  ManagerConfig{Provider: "tinsoft", TinsoftKeys: "key1\nkey2", TinsoftThreads: 2},
			want: true,
		},
		{
			name: "tinsoft without keys → not configured",
			cfg:  ManagerConfig{Provider: "tinsoft", TinsoftKeys: ""},
			want: false,
		},
		{
			name: "shoplike with keys → configured",
			cfg:  ManagerConfig{Provider: "shoplike", ShoplikeKeys: "key1", ShoplikeThreads: 1},
			want: true,
		},
		{
			name: "netproxy with keys → configured",
			cfg:  ManagerConfig{Provider: "netproxy", NetproxyKeys: "key1", NetproxyThreads: 1},
			want: true,
		},
		{
			name: "minproxy with keys → configured",
			cfg:  ManagerConfig{Provider: "minproxy", MinproxyKeys: "key1", MinproxyThreads: 1},
			want: true,
		},
		{
			name: "proxy_farm with keys → configured",
			cfg:  ManagerConfig{Provider: "proxy_farm", ProxyfarmKeys: "key1", ProxyfarmThreads: 1},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager(tt.cfg)
			if got := m.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v; want %v", got, tt.want)
			}
		})
	}
}
