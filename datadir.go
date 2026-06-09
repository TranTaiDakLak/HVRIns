package main

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
)

// appDataDir trả về thư mục gốc chứa data/config (Config/, logs/, result/).
//
// Logic:
//   - Dev build  (AppVersion == "dev"): {project_root}/bin/dev/
//   - Production (AppVersion set bởi ldflags): thư mục chứa exe
//
// Override thủ công: set env HVRINS_DATA_DIR (dùng cho CI hoặc test script).
func appDataDir() string {
	dataDirOnce.Do(func() {
		if d := os.Getenv("HVRINS_DATA_DIR"); d != "" {
			dataDirVal = filepath.Clean(d)
			return
		}
		if AppVersion == "dev" {
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
