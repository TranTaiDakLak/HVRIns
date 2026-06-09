package adapter_test

import (
	"strings"
	"testing"

	"HVRIns/internal/settings/adapter"
)

// TestBuildMappingReport_RuntimeFieldsMapped kiểm tra runtime fields vào MappedOk
func TestBuildMappingReport_RuntimeFieldsMapped(t *testing.T) {
	s := sampleLegacySettings()
	ic := sampleLegacyInteraction()

	r := adapter.BuildMappingReport(s, ic)

	if len(r.MappedOk) == 0 {
		t.Fatal("mappedOk should not be empty")
	}

	findOK := func(key string) bool {
		for _, f := range r.MappedOk {
			if f.LegacyKey == key {
				return true
			}
		}
		return false
	}

	for _, key := range []string{"threadRequest", "delayRequest", "threadCheckInfo", "loginPlatform"} {
		if !findOK(key) {
			t.Errorf("expected %q in mappedOk", key)
		}
	}
}

// TestBuildMappingReport_PathsRequireConfirmation kiểm tra path fields vào NeedsConfirm
func TestBuildMappingReport_PathsRequireConfirmation(t *testing.T) {
	s := sampleLegacySettings()
	ic := sampleLegacyInteraction()

	r := adapter.BuildMappingReport(s, ic)

	findConfirm := func(key string) bool {
		for _, f := range r.NeedsConfirm {
			if f.LegacyKey == key {
				return true
			}
		}
		return false
	}

	if !findConfirm("accountSourcePath") {
		t.Error("accountSourcePath should be in needsConfirm")
	}
	if !findConfirm("outputPath") {
		t.Error("outputPath should be in needsConfirm")
	}
}

// TestBuildMappingReport_SensitiveFieldsMasked kiểm tra sensitive fields được mask
func TestBuildMappingReport_SensitiveFieldsMasked(t *testing.T) {
	s := sampleLegacySettings()
	ic := sampleLegacyInteraction()

	r := adapter.BuildMappingReport(s, ic)

	if len(r.Sensitive) == 0 {
		t.Fatal("sensitive should not be empty (cloneHvPassword, apikeys present)")
	}
	for _, f := range r.Sensitive {
		if f.DisplayValue != "***" {
			t.Errorf("sensitive field %q displayValue should be '***', got %q", f.LegacyKey, f.DisplayValue)
		}
	}
}

// TestBuildMappingReport_SensitiveFieldsPresent kiểm tra các sensitive field cụ thể có mặt
func TestBuildMappingReport_SensitiveFieldsPresent(t *testing.T) {
	s := sampleLegacySettings()
	ic := sampleLegacyInteraction()

	r := adapter.BuildMappingReport(s, ic)

	sensitiveKeys := map[string]bool{}
	for _, f := range r.Sensitive {
		sensitiveKeys[f.LegacyKey] = true
	}

	for _, expected := range []string{"cloneHvPassword", "store1sApiKey", "mail30sApiKey"} {
		if !sensitiveKeys[expected] {
			t.Errorf("%q should be in sensitive", expected)
		}
	}
}

// TestBuildMappingReport_UnsupportedHMA kiểm tra HMA vào Unsupported
func TestBuildMappingReport_UnsupportedHMA(t *testing.T) {
	s := sampleLegacySettings()
	s.General.IpProvider = "hma"
	ic := sampleLegacyInteraction()

	r := adapter.BuildMappingReport(s, ic)

	found := false
	for _, f := range r.Unsupported {
		if f.LegacyKey == "ipProvider" && strings.Contains(f.Note, "HMA") {
			found = true
		}
	}
	if !found {
		t.Error("hma ipProvider should appear in unsupported")
	}
}

// TestBuildMappingReport_UnsupportedPlatform kiểm tra instagram vào Unsupported
func TestBuildMappingReport_UnsupportedPlatform(t *testing.T) {
	s := sampleLegacySettings()
	s.General.LoginPlatform = "instagram"

	r := adapter.BuildMappingReport(s, sampleLegacyInteraction())

	found := false
	for _, f := range r.Unsupported {
		if f.LegacyKey == "loginPlatform" {
			found = true
		}
	}
	if !found {
		t.Error("instagram loginPlatform should appear in unsupported")
	}
}

// TestBuildMappingReport_EmptyInput không panic khi input rỗng
func TestBuildMappingReport_EmptyInput(t *testing.T) {
	r := adapter.BuildMappingReport(adapter.LegacySettingsData{}, adapter.LegacyInteractionConfig{})

	if len(r.ParseErrors) != 0 {
		t.Errorf("empty input: unexpected parse errors: %v", r.ParseErrors)
	}
	if len(r.MappedOk) == 0 {
		t.Error("even empty input should produce some mappedOk entries (boolean/default fields)")
	}
}

// TestBuildMappingReport_ProxyListCountedCorrectly kiểm tra proxy list đếm dòng
func TestBuildMappingReport_ProxyListCountedCorrectly(t *testing.T) {
	s := sampleLegacySettings()
	ic := sampleLegacyInteraction()

	r := adapter.BuildMappingReport(s, ic)

	found := false
	for _, f := range r.MappedOk {
		if f.LegacyKey == "proxyList" {
			if !strings.Contains(f.DisplayValue, "proxy entries") {
				t.Errorf("proxyList displayValue should contain 'proxy entries', got %q", f.DisplayValue)
			}
			found = true
		}
	}
	if !found {
		t.Error("proxyList should be in mappedOk")
	}
}

// TestBuildMappingReport_UnknownMailProviderConfirm kiểm tra unknown mail provider
func TestBuildMappingReport_UnknownMailProviderConfirm(t *testing.T) {
	s := sampleLegacySettings()
	ic := sampleLegacyInteraction()
	ic.MailProvider = "@unknown-custom.vn"

	r := adapter.BuildMappingReport(s, ic)

	found := false
	for _, f := range r.NeedsConfirm {
		if f.LegacyKey == "mailProvider" {
			found = true
		}
	}
	if !found {
		t.Error("unknown mailProvider should be in needsConfirm")
	}
}
