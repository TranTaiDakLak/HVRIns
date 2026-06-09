package ioshttp

import "testing"

func TestParseSeed_None(t *testing.T) {
	s := ParseSeed("")
	if s.Mode != SeedModeNone {
		t.Errorf("expected SeedModeNone, got %d", s.Mode)
	}
}

func TestParseSeed_DatrOnly(t *testing.T) {
	s := ParseSeed("BYB0aMBxu79SmdedMp5Xb9EA")
	if s.Mode != SeedModeDatrOnly {
		t.Errorf("expected SeedModeDatrOnly, got %d", s.Mode)
	}
	if s.Datr != "BYB0aMBxu79SmdedMp5Xb9EA" {
		t.Errorf("wrong datr: %s", s.Datr)
	}
}

func TestParseSeed_FullCookie(t *testing.T) {
	s := ParseSeed("datr=ABC123;sb=XYZ;fr=789")
	if s.Mode != SeedModeFullCookie {
		t.Errorf("expected SeedModeFullCookie, got %d", s.Mode)
	}
	if s.Datr != "ABC123" {
		t.Errorf("wrong datr: %s", s.Datr)
	}
}

func TestParseSeed_InitialAccount(t *testing.T) {
	raw := "61578631587064|7l47dekfv3|c_user=61578631587064;xs=6:9CC4b52;datr=otp6aFL9;|EAAAAtoken|email|name|date|US"
	s := ParseSeed(raw)
	if s.Mode != SeedModeInitialAccount {
		t.Errorf("expected SeedModeInitialAccount, got %d", s.Mode)
	}
	if s.UID != "61578631587064" {
		t.Errorf("wrong UID: %s", s.UID)
	}
	if s.Password != "7l47dekfv3" {
		t.Errorf("wrong password: %s", s.Password)
	}
	if s.Datr != "otp6aFL9" {
		t.Errorf("wrong datr: %s", s.Datr)
	}
}

func TestParseSeed_InitialAccount_NoCookie(t *testing.T) {
	s := ParseSeed("12345678|mypassword")
	if s.Mode != SeedModeInitialAccount {
		t.Errorf("expected SeedModeInitialAccount, got %d", s.Mode)
	}
	if s.UID != "12345678" {
		t.Errorf("wrong UID: %s", s.UID)
	}
	if s.Password != "mypassword" {
		t.Errorf("wrong password: %s", s.Password)
	}
	if s.Datr != "" {
		t.Errorf("expected empty datr, got: %s", s.Datr)
	}
}

func TestSessionPool_AcquireMiss(t *testing.T) {
	pool := NewSessionPool(0)
	sess, isFirst := pool.Acquire("proxy1")
	if sess != nil {
		t.Error("expected nil session on miss")
	}
	if !isFirst {
		t.Error("expected isFirst=true on miss")
	}
}

func TestSessionPool_StoreAndHit(t *testing.T) {
	pool := NewSessionPool(0)
	s := &session{finalURL: "test"}
	pool.Store("proxy1", s)

	got, isFirst := pool.Acquire("proxy1")
	if got != s {
		t.Error("expected same session on hit")
	}
	if isFirst {
		t.Error("expected isFirst=false on hit")
	}
}

func TestSessionPool_Remove(t *testing.T) {
	pool := NewSessionPool(0)
	s := &session{finalURL: "test"}
	pool.Store("proxy1", s)
	pool.Remove("proxy1")

	got, isFirst := pool.Acquire("proxy1")
	if got != nil {
		t.Error("expected nil after remove")
	}
	if !isFirst {
		t.Error("expected isFirst=true after remove")
	}
}
