package fbdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseVersionsAndBuilds(t *testing.T) {
	in := `
# comment nhưng không có |
554.0.0.57.70|918990560
540.0.0.41.68|832114220

|invalid-left
rightonly|
541.0.0.43.69|836002441
`
	got := ParseVersionsAndBuilds(in)
	if len(got) != 3 {
		t.Fatalf("expected 3 valid rows, got %d: %+v", len(got), got)
	}
	if got[0].Version != "554.0.0.57.70" || got[0].Build != "918990560" {
		t.Errorf("first row wrong: %+v", got[0])
	}
	if got[2].Version != "541.0.0.43.69" || got[2].Build != "836002441" {
		t.Errorf("third row wrong: %+v", got[2])
	}
}

func TestSetDefaultVersions_NoOverride(t *testing.T) {
	// Reset state
	mu.Lock()
	defaultVersions = nil
	activeVersions = nil
	overridePath = ""
	mu.Unlock()

	defaults := []FbVersion{
		{Version: "1.0", Build: "100"},
		{Version: "2.0", Build: "200"},
	}
	SetDefaultVersions(defaults)

	if Size() != 2 {
		t.Errorf("Size() = %d, want 2", Size())
	}
	v := Versions()
	if len(v) != 2 || v[0].Version != "1.0" {
		t.Errorf("Versions() wrong: %+v", v)
	}
	if OverrideActive() {
		t.Error("OverrideActive() should be false when no override path set")
	}
}

func TestReload_OverrideReplacesDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "versions_and_builds.txt")
	_ = os.WriteFile(path, []byte("9.9.9.9.9|999999999\n"), 0644)

	mu.Lock()
	defaultVersions = []FbVersion{{Version: "1.0", Build: "100"}}
	activeVersions = nil
	overridePath = ""
	mu.Unlock()

	SetDefaultVersions(defaultVersions)
	if Size() != 1 || Versions()[0].Version != "1.0" {
		t.Fatalf("default not set correctly: %+v", Versions())
	}

	Reload(path)
	if Size() != 1 {
		t.Fatalf("after reload expected 1 entry, got %d", Size())
	}
	if Versions()[0].Version != "9.9.9.9.9" {
		t.Errorf("override not applied: %+v", Versions())
	}
	if !OverrideActive() {
		t.Error("OverrideActive() should be true")
	}
}

func TestReload_EmptyFileFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "versions_and_builds.txt")
	_ = os.WriteFile(path, []byte(""), 0644) // file rỗng

	mu.Lock()
	defaultVersions = []FbVersion{{Version: "default", Build: "111"}}
	overridePath = ""
	mu.Unlock()

	SetDefaultVersions(defaultVersions)
	Reload(path)

	if Size() != 1 || Versions()[0].Version != "default" {
		t.Errorf("empty override should fallback to default, got %+v", Versions())
	}
}

func TestReload_MissingFileFallsBackToDefault(t *testing.T) {
	mu.Lock()
	defaultVersions = []FbVersion{{Version: "keep", Build: "1"}}
	overridePath = ""
	mu.Unlock()

	SetDefaultVersions(defaultVersions)
	Reload(filepath.Join(t.TempDir(), "does-not-exist.txt"))

	if Size() != 1 || Versions()[0].Version != "keep" {
		t.Errorf("missing override should fallback to default, got %+v", Versions())
	}
}

func TestVersions_ReturnsCopy(t *testing.T) {
	mu.Lock()
	defaultVersions = []FbVersion{{Version: "orig", Build: "1"}}
	mu.Unlock()
	SetDefaultVersions(defaultVersions)

	got := Versions()
	got[0].Version = "MUTATED"

	if Versions()[0].Version == "MUTATED" {
		t.Error("Versions() must return copy — caller mutation leaked into active cache")
	}
}
