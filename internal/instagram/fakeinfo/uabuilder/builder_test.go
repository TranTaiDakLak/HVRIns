// builder_test.go — verify UA builder outputs for AndroidUABuilder + BrowserUABuilder.
package uabuilder

import (
	"strings"
	"testing"
)

func init() {
	// Test chạy từ /internal/facebook/fakeinfo/uabuilder/ → Config/ ở root project (../../../../Config).
	SetConfigBaseDir("../../../../Config")
}

func TestAndroidUA_NoVirtualSpecs(t *testing.T) {
	b := &AndroidUABuilder{}
	res, err := b.Build(UAOptions{Platform: "s23", Locale: "en_US"})
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if strings.HasPrefix(res.UserAgent, "Dalvik/2.1.0") {
		t.Errorf("Expected NO Dalvik prefix when AddVirtualSpecs=false, got: %s", res.UserAgent)
	}
	if !strings.Contains(res.UserAgent, "[FBAN/FB4A;FBAV/") {
		t.Errorf("UA missing FBAN/FB4A: %s", res.UserAgent)
	}
	if !strings.Contains(res.UserAgent, "FBDM={density=") {
		t.Errorf("UA must use 'FBDM={density=...}' (no slash), got: %s", res.UserAgent)
	}
	if !strings.Contains(res.UserAgent, "FBDV/"+res.Model) {
		t.Errorf("UA FBDV/<Model> mismatch: model=%s ua=%s", res.Model, res.UserAgent)
	}
}

func TestAndroidUA_VirtualSpecs(t *testing.T) {
	b := &AndroidUABuilder{}
	res, err := b.Build(UAOptions{
		Platform:        "s23",
		Locale:          "en_US",
		AddVirtualSpecs: true,
	})
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if !strings.HasPrefix(res.UserAgent, "Dalvik/2.1.0") {
		t.Errorf("Expected Dalvik prefix when AddVirtualSpecs=true, got: %s", res.UserAgent)
	}
	expectedBuild := res.Brand + "-" + res.Model
	if !strings.Contains(res.UserAgent, "Build/"+expectedBuild) {
		t.Errorf("Expected Build/%s (Brand-Model), got UA=%s", expectedBuild, res.UserAgent)
	}
}

func TestAndroidUA_NoVirtualSpecs_NoBuildInUA(t *testing.T) {
	b := &AndroidUABuilder{}
	res, err := b.Build(UAOptions{
		Platform:        "s23",
		Locale:          "en_US",
		AddVirtualSpecs: false,
	})
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if strings.HasPrefix(res.UserAgent, "Dalvik/2.1.0") {
		t.Errorf("Expected NO Dalvik prefix, got: %s", res.UserAgent)
	}
	if strings.Contains(res.UserAgent, "Build/") {
		t.Errorf("Expected no Build/ in UA when no virtual specs, got: %s", res.UserAgent)
	}
}

func TestAndroidUA_PerPlatformDevicePool(t *testing.T) {
	b := &AndroidUABuilder{}
	for _, plat := range []string{"s22", "s23", "s24", "s25", "s26"} {
		res, err := b.Build(UAOptions{Platform: plat, Locale: "en_US"})
		if err != nil {
			t.Fatalf("Build platform=%s error: %v", plat, err)
		}
		if !strings.Contains(res.Model, "SM-S") {
			t.Errorf("Platform %s should pick SM-S* model, got %s", plat, res.Model)
		}
	}
}

func TestBrowserUA_Build(t *testing.T) {
	b := &BrowserUABuilder{}
	res, err := b.Build(UAOptions{
		Platform: "s23",
	})
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}
	if !strings.HasPrefix(res.UserAgent, "Mozilla/5.0 ") {
		t.Errorf("Expected Mozilla/5.0 prefix, got: %s", res.UserAgent)
	}
	if !strings.Contains(res.UserAgent, "Chrome/") {
		t.Errorf("Expected Chrome/ token, got: %s", res.UserAgent)
	}
	if res.ChromeVersion == "" {
		t.Errorf("ChromeVersion empty")
	}
}

func TestResolveBuilder(t *testing.T) {
	if _, err := ResolveBuilder("s23", BuilderTypeAndroidApp); err != nil {
		t.Errorf("AndroidApp resolve error: %v", err)
	}
	if _, err := ResolveBuilder("s23", BuilderTypeBrowserAndroid); err != nil {
		t.Errorf("BrowserAndroid resolve error: %v", err)
	}
	if _, err := ResolveBuilder("s23", 99); err == nil {
		t.Errorf("Expected error for unknown builderType=99")
	}
}
