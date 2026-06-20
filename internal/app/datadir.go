package app

import (
	"os"
	"path/filepath"
	"sync"
)

// originalWD ghi lại CWD ngay khi package load — trước bất kỳ os.Chdir nào.
// Dùng để tính đường dẫn bin/dev/ khi chạy wails dev.
var originalWD = func() string {
	wd, _ := os.Getwd()
	return wd
}()

var (
	dataDirOnce sync.Once
	dataDirVal  string

	// buildVersion là bản copy của main.AppVersion, set bởi SetBuildVersion().
	// Cần trước khi App struct tồn tại (được gọi trong AppDataDir trước NewApp).
	buildVersion = "dev"
)

// SetBuildVersion phải được gọi từ main() trước AppDataDir() và NewApp().
func SetBuildVersion(v string) { buildVersion = v }

// AppDataDir trả về thư mục gốc chứa data/config (Config/, logs/, result/).
//
// Logic:
//   - Dev build  (buildVersion == "dev"): {project_root}/bin/dev/
//   - Production (buildVersion set bởi ldflags): thư mục chứa exe
//
// Override thủ công: set env HVRINS_DATA_DIR (dùng cho CI hoặc test script).
func AppDataDir() string {
	dataDirOnce.Do(func() {
		if d := os.Getenv("HVRINS_DATA_DIR"); d != "" {
			dataDirVal = filepath.Clean(d)
			return
		}
		if buildVersion == "dev" {
			dataDirVal = filepath.Join(originalWD, "bin", "dev")
			return
		}
		exe, err := os.Executable()
		if err == nil {
			dataDirVal = filepath.Dir(exe)
		} else {
			dataDirVal = "."
		}
	})
	return dataDirVal
}
