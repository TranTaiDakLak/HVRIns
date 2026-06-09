package android

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

// datrFormat: 24 ký tự base64url, ký tự 4-5 (1-indexed) là "Ga". Tag pos2
// nằm trong set datrTagChars (xem datr_gen.go).
var datrFormat = regexp.MustCompile(`^[A-Za-z0-9_-]{2}[cgkosw048]Ga[A-Za-z0-9_-]{19}$`)

func TestGenerateDatr_Format(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		d := GenerateDatr()
		if len(d) != 24 {
			t.Fatalf("expected 24 chars, got %d: %q", len(d), d)
		}
		if !datrFormat.MatchString(d) {
			t.Fatalf("format mismatch: %q", d)
		}
		if _, dup := seen[d]; dup {
			t.Fatalf("duplicate datr in 1000 iterations: %q", d)
		}
		seen[d] = struct{}{}
	}
}

func TestLoadGenerated_Pool(t *testing.T) {
	p := NewPartitionedPool(3)
	n := p.LoadGenerated(50)
	if n < 49 { // tolerate 1 collision in 50; >= 49 means generator phân tán đều
		t.Fatalf("expected ~50 datrs added, got %d", n)
	}
	if p.Size() != n {
		t.Fatalf("pool size mismatch: Size=%d, loaded=%d", p.Size(), n)
	}
}

func TestLoadFromFileTail_TakesLastN(t *testing.T) {
	// Tạo file giả SuccessVerify_No2FA.txt format: uid|pass|2fa|cookie|...
	// 5 dòng, cookie chứa datr=<unique>. Yêu cầu chỉ lấy 3 dòng cuối.
	dir := t.TempDir()
	path := filepath.Join(dir, "sv.txt")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	// 5 datr — chỉ pos1-2 + cuối thay đổi để pass extractDatr regex
	datrs := []string{
		"AAcGaaaaaaaaaaaaaaaaaaaa",
		"BBcGabbbbbbbbbbbbbbbbbbb",
		"CCcGacccccccccccccccccccc"[:24],
		"DDcGadddddddddddddddddddd"[:24],
		"EEcGaeeeeeeeeeeeeeeeeeeee"[:24],
	}
	for i, d := range datrs {
		fmt.Fprintf(f, "uid%d|pass|0|datr=%s;c_user=123|token|e@x|name|t|US\n", i, d)
	}
	f.Close()

	p := NewPartitionedPool(9)
	n, err := p.LoadFromFileTail(path, 3)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("expected 3 datrs (lastN), got %d", n)
	}
	// 3 datr cuối phải có mặt; 2 đầu phải KHÔNG có
	for _, d := range datrs[2:] {
		if _, ok := p.usageCount[d]; !ok {
			t.Fatalf("expected datr %q (last 3) to be loaded", d)
		}
	}
	for _, d := range datrs[:2] {
		if _, ok := p.usageCount[d]; ok {
			t.Fatalf("datr %q (first 2) must NOT be loaded — got it in pool", d)
		}
	}
}

func TestExpireOldDatrs(t *testing.T) {
	p := NewPartitionedPool(9)
	p.LoadGenerated(5)
	if p.Size() != 5 {
		t.Fatalf("setup: expected 5 in pool, got %d", p.Size())
	}
	// Force backdate timestamps: set maxAge tiny, sleep, expire.
	p.SetMaxAgeMinutes(1) // 1 phút
	// manually backdate all loadedAt sang quá khứ
	p.mu.Lock()
	for d := range p.loadedAt {
		p.loadedAt[d] = time.Now().Add(-2 * time.Minute)
	}
	p.mu.Unlock()
	removed := p.ExpireOldDatrs()
	if removed != 5 {
		t.Fatalf("expected 5 removed, got %d", removed)
	}
	if p.Size() != 0 {
		t.Fatalf("expected empty after expiry, got Size=%d", p.Size())
	}
}
