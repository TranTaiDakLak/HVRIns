package iosmess

import (
	"HVRIns/internal/instagram/register/android"
)

// SharedPool — partitioned datr pool (mỗi slot goroutine có queue riêng).
// Mirror appmessv3.SharedPool. Wire trong app_reg_sxxx.go.
var SharedPool *android.PartitionedDatrPool

// LoadSharedPool nạp datr từ file vào SharedPool.
func LoadSharedPool(paths []string) int {
	if SharedPool == nil {
		SharedPool = android.NewPartitionedPool(50)
	}
	total := 0
	for _, p := range paths {
		if n, err := SharedPool.LoadFromFile(p); err == nil {
			total += n
		}
	}
	return total
}
