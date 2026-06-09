// pool.go — Device profile pool cho iOS540 native reg.
//
// Lưu lại DeviceID + FamilyDeviceID + MachineID từ reg thành công để tái dùng.
// Thay vì sinh random mỗi lần, pool tái dùng định danh thiết bị đã "qua mặt" FB —
// giúp tăng tỉ lệ thành công (thiết bị đã từng reg OK có danh tiếng tốt hơn).
//
// File format (1 dòng/profile): DeviceID|FamilyDeviceID|MachineID
// Ví dụ: 43E5F05E-2392-4988-829E-2B6DAF96F411|E2853315-7852-4512-80FB-50230C0F1C69|6AJEtA5JgQgYUOoG9wvfTwNB
package ios548

import (
	"bufio"
	"os"
	"strings"
	"sync"

	androidreg "HVRIns/internal/instagram/register/android"
)

// DeviceProfile là bộ định danh thiết bị iOS cố định.
type DeviceProfile struct {
	DeviceID       string // IDFV — uppercase UUID
	FamilyDeviceID string // uppercase UUID
	MachineID      string // 24-char base64url
}

// SharedDevicePool là global pool — set từ app.go trước khi batch bắt đầu.
var SharedDevicePool *DevicePool

// SharedDatrPool là partitioned datr pool dùng chung — set từ app.go.
// Cùng cơ chế với Android/S23: datr từ file cookie_initial được dùng làm
// X-FB-Integrity-Machine-Id thay vì sinh random, giúp tăng trust score.
var SharedDatrPool *androidreg.PartitionedDatrPool

// DevicePool quản lý device profile pool cho ios562 reg.
// Thread-safe. Round-robin với usage counter.
type DevicePool struct {
	mu          sync.Mutex
	profiles    []DeviceProfile
	usageCount  map[string]int // key = DeviceID
	maxUsage    int
	idx         int
	persistHook func(DeviceProfile)
}

// NewDevicePool tạo pool mới. maxUsage=0 → dùng default 5.
func NewDevicePool(maxUsage int) *DevicePool {
	if maxUsage <= 0 {
		maxUsage = 5
	}
	return &DevicePool{
		usageCount: make(map[string]int),
		maxUsage:   maxUsage,
	}
}

// SetPersistHook đăng ký callback được gọi mỗi khi profile MỚI được thêm vào pool.
func (p *DevicePool) SetPersistHook(hook func(DeviceProfile)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.persistHook = hook
}

// Add thêm profile mới từ reg thành công. Trigger persistHook nếu DeviceID chưa có.
func (p *DevicePool) Add(dp DeviceProfile) {
	if dp.DeviceID == "" {
		return
	}
	p.mu.Lock()
	_, existed := p.usageCount[dp.DeviceID]
	if !existed {
		p.profiles = append(p.profiles, dp)
		p.usageCount[dp.DeviceID] = 0
	}
	hook := p.persistHook
	p.mu.Unlock()

	if !existed && hook != nil {
		hook(dp)
	}
}

// GetNext lấy 1 profile chưa quá maxUsage theo round-robin.
// Trả về nil nếu pool rỗng hoặc tất cả đã đủ lần dùng.
func (p *DevicePool) GetNext() *DeviceProfile {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := len(p.profiles)
	if n == 0 {
		return nil
	}
	for i := 0; i < n; i++ {
		idx := (p.idx + i) % n
		dp := &p.profiles[idx]
		if p.usageCount[dp.DeviceID] < p.maxUsage {
			p.usageCount[dp.DeviceID]++
			p.idx = (idx + 1) % n
			copy := *dp
			return &copy
		}
	}
	return nil
}

// Size trả về số profile trong pool.
func (p *DevicePool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.profiles)
}

// LoadFromFile đọc file pipe-delimited, nạp vào pool.
// Bỏ qua dòng trống và dòng format sai.
func (p *DevicePool) LoadFromFile(path string) (int, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return 0, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer f.Close()

	count := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		dp := DeviceProfile{
			DeviceID:       strings.TrimSpace(parts[0]),
			FamilyDeviceID: strings.TrimSpace(parts[1]),
			MachineID:      strings.TrimSpace(parts[2]),
		}
		if dp.DeviceID == "" || dp.FamilyDeviceID == "" || dp.MachineID == "" {
			continue
		}
		p.mu.Lock()
		if _, exists := p.usageCount[dp.DeviceID]; !exists {
			p.profiles = append(p.profiles, dp)
			p.usageCount[dp.DeviceID] = 0
			count++
		}
		p.mu.Unlock()
	}
	return count, sc.Err()
}
