package fakeinfo

import (
	"regexp"
	"testing"

	"HVRIns/internal/fbdata"
)

// TestFbDataEmbedLoaded — init() của fakeinfo phải nạp file embed vào fbdata.
// Nếu thay đổi file data/versions_and_builds.txt, số expected sẽ thay đổi — đó là tín hiệu
// test cần update chứ không phải bug.
func TestFbDataEmbedLoaded(t *testing.T) {
	skipIfNoConfigData(t)
	size := fbdata.Size()
	if size < 100 {
		t.Errorf("expected embed dataset > 100 entries (C# merged data ~2320), got %d", size)
	}
}

func TestRandomFbVersion_ValidFormat(t *testing.T) {
	verRE := regexp.MustCompile(`^\d+\.\d+\.\d+\.\d+\.\d+$`)
	buildRE := regexp.MustCompile(`^\d+$`)
	// Gọi nhiều lần để chắc chắn mọi entry đều valid format
	for i := 0; i < 50; i++ {
		v, b := RandomFbVersion()
		if !verRE.MatchString(v) {
			t.Errorf("iter %d: version %q không đúng format x.x.x.x.x", i, v)
		}
		if !buildRE.MatchString(b) {
			t.Errorf("iter %d: build %q không phải số", i, b)
		}
	}
}

func TestSimList_HasGlobalCoverage(t *testing.T) {
	skipIfNoConfigData(t)
	// Sau khi import full mcc-mnc.csv (~2695 entries), list phải cover đủ các country lớn.
	if len(simList) < 1000 {
		t.Errorf("simList quá ít: %d — kỳ vọng > 1000 entries (C# mcc-mnc.csv full)", len(simList))
	}
	// Spot check: các country lớn phải có SIM
	mustHave := []string{"VN", "US", "TH", "IN", "GB", "DE", "JP", "BR"}
	for _, cc := range mustHave {
		sim := RandomSimProfile(cc)
		if sim.CountryCode != cc {
			t.Errorf("country %s không có SIM — got %+v", cc, sim)
		}
	}
}

func TestNames_FullDataset(t *testing.T) {
	skipIfNoConfigData(t)
	if len(firstNames) < 500 {
		t.Errorf("firstNames %d < 500 — kỳ vọng 999 từ C# US/firstname.txt", len(firstNames))
	}
	if len(lastNames) < 500 {
		t.Errorf("lastNames %d < 500 — kỳ vọng 999 từ C# US/lastname.txt", len(lastNames))
	}
}

func TestRandomLocale(t *testing.T) {
	skipIfNoConfigData(t)
	if len(localeList) < 30 {
		t.Errorf("localeList %d < 30 — kỳ vọng 42 từ C# locales.txt", len(localeList))
	}
	// 20 lần random đều phải ra locale format hợp lệ
	localeRE := regexp.MustCompile(`^[a-z]{2,3}_[A-Z]{2}$`)
	for i := 0; i < 20; i++ {
		l := RandomLocale()
		if !localeRE.MatchString(l) {
			t.Errorf("iter %d: locale %q không đúng format xx_YY", i, l)
		}
	}
}

func TestChromeVersions_Recent(t *testing.T) {
	// Sau khi copy Chrome_Versions.txt mới (146), list phải có version cao hơn 140
	versions := loadDeviceInfoLines("chrome_versions.txt")
	if len(versions) == 0 {
		t.Skip("chrome_versions.txt chưa có (runtime file) — bỏ qua test này")
		return
	}
	found := false
	for _, v := range versions {
		if len(v) >= 3 && v[0] == '1' && (v[1] == '4' || v[1] == '5') {
			// Chrome 14x, 15x
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Chrome versions chưa update — không tìm thấy 14x.x.x: %v", versions)
	}
}
