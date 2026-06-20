// testdata_guard_test.go — shared guard cho các test phụ thuộc data runtime.
package fakeinfo

import (
	"os"
	"testing"
)

// skipIfNoConfigData skip các test khẳng định "full dataset" khi data runtime
// Config/* KHÔNG có mặt. Các dataset (phone code, locales, UA pool, mcc-mnc/SimNetwork,
// Namereg, fbdata source) được nạp lúc init() từ thư mục `Config/` (baseConfigDir,
// relative CWD) bằng os.ReadFile — đây là RUNTIME DATA bị .gitignore, KHÔNG nằm trong repo.
//
// Khi chạy `go test` (CWD = thư mục package, không có Config/), loader nạp dataset RỖNG
// → các assertion "≥ N entries" fail. Đây là vấn đề MÔI TRƯỜNG (thiếu data), KHÔNG phải
// regression. Để chạy thật: seed Config/ rồi `go test` từ app data dir, hoặc chạy app.
func skipIfNoConfigData(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(baseConfigDir); err != nil {
		t.Skipf("requires runtime Config/* data (gitignored, absent during go test): %v", err)
	}
}
