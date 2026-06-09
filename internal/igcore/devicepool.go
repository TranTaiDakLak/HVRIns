// devicepool.go — pool device aged (datr/mid/ig_did) để inject vào reg.
// Dịch phương pháp datr-pool của NullCoreSummer (FB) sang IG iOS:
//   - Harvest mid/datr từ account LIVE → lưu pool
//   - Inject device aged trước reg → account mới "thừa hưởng" thiết bị có lịch sử
//   - Giới hạn usage/device để tránh IG link nhiều account chung 1 device → mass-ban.
package igcore

import (
	"bufio"
	"os"
	"strings"
	"sync"
)

// AgedDevice là 1 bộ định danh thiết bị aged harvest từ account live.
type AgedDevice struct {
	Mid   string // X-MID — machine id chính của IG iOS
	Datr  string // datr cookie — machine id Meta web-layer
	IgDID string // ig_did — Instagram device id
	uses  int    // số lần đã dùng (runtime)
}

// DevicePool quản lý pool aged device, thread-safe, file-backed.
type DevicePool struct {
	mu       sync.Mutex
	devices  []*AgedDevice
	seen     map[string]bool // dedupe theo mid
	idx      int             // round-robin cursor
	maxUses  int             // số lần tối đa 1 device được dùng
	filePath string
}

// NewDevicePool tạo pool với giới hạn usage và file lưu.
func NewDevicePool(filePath string, maxUses int) *DevicePool {
	if maxUses <= 0 {
		maxUses = 3
	}
	p := &DevicePool{
		seen:     make(map[string]bool),
		maxUses:  maxUses,
		filePath: filePath,
	}
	p.loadFromFile()
	return p
}

// Size trả số device hiện có trong pool.
func (p *DevicePool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.devices)
}

// Add thêm 1 device harvest được (dedupe theo mid). Trả true nếu là mới.
func (p *DevicePool) Add(mid, datr, igDID string) bool {
	mid = strings.TrimSpace(mid)
	if mid == "" {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.seen[mid] {
		return false
	}
	p.seen[mid] = true
	p.devices = append(p.devices, &AgedDevice{Mid: mid, Datr: datr, IgDID: igDID})
	p.appendToFile(mid, datr, igDID)
	return true
}

// Next lấy 1 device aged (round-robin), tăng usage. Trả nil nếu pool rỗng
// hoặc tất cả device đã vượt maxUses.
func (p *DevicePool) Next() *AgedDevice {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := len(p.devices)
	if n == 0 {
		return nil
	}
	for i := 0; i < n; i++ {
		d := p.devices[p.idx%n]
		p.idx++
		if d.uses < p.maxUses {
			d.uses++
			// trả copy để caller không sửa state pool
			return &AgedDevice{Mid: d.Mid, Datr: d.Datr, IgDID: d.IgDID}
		}
	}
	return nil // tất cả device đã hết lượt
}

// ── File persistence ──────────────────────────────────────────────────────────
// Format mỗi dòng: mid|datr|ig_did

func (p *DevicePool) loadFromFile() {
	f, err := os.Open(p.filePath)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "|")
		mid := parts[0]
		if mid == "" || p.seen[mid] {
			continue
		}
		datr, igDID := "", ""
		if len(parts) > 1 {
			datr = parts[1]
		}
		if len(parts) > 2 {
			igDID = parts[2]
		}
		p.seen[mid] = true
		p.devices = append(p.devices, &AgedDevice{Mid: mid, Datr: datr, IgDID: igDID})
	}
}

func (p *DevicePool) appendToFile(mid, datr, igDID string) {
	if p.filePath == "" {
		return
	}
	f, err := os.OpenFile(p.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(mid + "|" + datr + "|" + igDID + "\n")
}
