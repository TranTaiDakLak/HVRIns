package proxy

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestSticky_DisabledAlwaysFresh(t *testing.T) {
	var acquires, releases atomic.Int32
	raw := func(ctx context.Context) (string, func(), error) {
		n := acquires.Add(1)
		return string(rune('A'+n-1)) + ":1", func() { releases.Add(1) }, nil
	}
	m := NewStickyManager(false, raw)

	// Worker 0: acquire → release(success=true) — disabled → always release
	p1, rel1 := m.Acquire(context.Background(), 0)
	rel1(true)
	// Worker 0 again: should acquire NEW proxy (not sticky)
	p2, rel2 := m.Acquire(context.Background(), 0)
	rel2(false)

	if p1 == p2 {
		t.Errorf("disabled mode should not pin: got same proxy %q twice", p1)
	}
	if got := releases.Load(); got != 2 {
		t.Errorf("releases=%d, want 2", got)
	}
}

func TestSticky_EnabledKeepsOnSuccess(t *testing.T) {
	var acquires, releases atomic.Int32
	raw := func(ctx context.Context) (string, func(), error) {
		n := acquires.Add(1)
		return string(rune('A'+n-1)) + ":1", func() { releases.Add(1) }, nil
	}
	m := NewStickyManager(true, raw)

	// Worker 0 first account → success → pin
	p1, rel1 := m.Acquire(context.Background(), 0)
	rel1(true)

	// Worker 0 next account → reuse pinned proxy
	p2, rel2 := m.Acquire(context.Background(), 0)
	if p1 != p2 {
		t.Errorf("sticky pin failed: got %q then %q", p1, p2)
	}
	if got := acquires.Load(); got != 1 {
		t.Errorf("should only acquire raw once, got %d", got)
	}
	if got := releases.Load(); got != 0 {
		t.Errorf("success + keep should not release, got %d", got)
	}
	// Fail this time → release + unpin
	rel2(false)
	if got := releases.Load(); got != 1 {
		t.Errorf("fail should release raw, got %d", got)
	}

	// Worker 0 again → acquire fresh
	p3, rel3 := m.Acquire(context.Background(), 0)
	rel3(true)
	if p3 == p1 {
		t.Errorf("after fail, should get new proxy, got same %q", p1)
	}
}

func TestSticky_PerWorkerIsolated(t *testing.T) {
	var acquires atomic.Int32
	raw := func(ctx context.Context) (string, func(), error) {
		n := acquires.Add(1)
		return string(rune('A'+n-1)) + ":1", func() {}, nil
	}
	m := NewStickyManager(true, raw)

	p0a, _ := m.Acquire(context.Background(), 0)
	p1a, _ := m.Acquire(context.Background(), 1)
	if p0a == p1a {
		t.Errorf("different workers should get different proxies")
	}
}

func TestSticky_ReleaseAllClearsPins(t *testing.T) {
	var releases atomic.Int32
	raw := func(ctx context.Context) (string, func(), error) {
		return "X:1", func() { releases.Add(1) }, nil
	}
	m := NewStickyManager(true, raw)

	_, rel := m.Acquire(context.Background(), 0)
	rel(true) // pin
	_, rel2 := m.Acquire(context.Background(), 1)
	rel2(true) // pin
	m.ReleaseAll()

	if got := releases.Load(); got != 2 {
		t.Errorf("ReleaseAll should release all pinned, got %d", got)
	}
}
