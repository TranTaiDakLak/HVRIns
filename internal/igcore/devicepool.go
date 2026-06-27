// devicepool.go — pool device aged (datr/mid/ig_did) để inject vào reg.
// Dịch phương pháp datr-pool của NullCoreSummer (FB) sang IG iOS:
//   - Harvest mid/datr từ account LIVE → lưu pool
//   - Inject device aged trước reg → account mới "thừa hưởng" thiết bị có lịch sử
//   - Giới hạn usage/device để tránh IG link nhiều account chung 1 device → mass-ban.
package igcore

import (
	"bufio"
	"log/slog"
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
	minSize  int             // pool phải đủ ≥ minSize device thì Next() mới trả (0 = trả ngay khi có ≥1)
	filePath string
}

// NewDevicePool tạo pool với giới hạn usage và file lưu.
// maxUses <= 0 nghĩa là KHÔNG giới hạn — 1 mid tái dùng vô hạn (round-robin).
func NewDevicePool(filePath string, maxUses int) *DevicePool {
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

// SetMinSize đặt ngưỡng tối thiểu: pool phải có ≥ n device thì Next() mới trả device.
// Tránh nhiều account dùng chung vài mid khi pool còn nhỏ → giảm rủi ro IG link-ban.
// n ≤ 0 nghĩa là không giới hạn (trả ngay khi có ≥1 device).
func (p *DevicePool) SetMinSize(n int) {
	p.mu.Lock()
	p.minSize = n
	p.mu.Unlock()
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
// hoặc chưa đủ ngưỡng minSize.
//
// Cơ chế RECYCLE (giống PartitionedDatrPool bên FB): khi maxUses > 0 và TẤT CẢ
// device đã dùng đủ maxUses lần → reset uses=0 cho cả pool rồi dùng lại từ đầu.
// → "luôn có mid dùng, hết thì xoay lại" nhưng có nhịp nghỉ (mỗi mid chỉ phục vụ
// maxUses acc trong 1 vòng trước khi cả pool cùng được tái chế). maxUses <= 0 = vô hạn.
func (p *DevicePool) Next() *AgedDevice {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := len(p.devices)
	if n == 0 || n < p.minSize {
		return nil // pool rỗng hoặc chưa đủ ngưỡng minSize → reg device tươi
	}
	// Vòng 1: tìm device chưa hết lượt.
	for i := 0; i < n; i++ {
		d := p.devices[p.idx%n]
		p.idx++
		if p.maxUses <= 0 || d.uses < p.maxUses {
			d.uses++
			return &AgedDevice{Mid: d.Mid, Datr: d.Datr, IgDID: d.IgDID} // copy, caller không sửa state
		}
	}
	// Tất cả đã hết lượt (maxUses > 0) → RECYCLE: reset toàn bộ rồi cấp device kế tiếp.
	for _, d := range p.devices {
		d.uses = 0
	}
	d := p.devices[p.idx%n]
	p.idx++
	d.uses++
	return &AgedDevice{Mid: d.Mid, Datr: d.Datr, IgDID: d.IgDID}
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
		// KHÔNG nuốt im lặng: mid vẫn nằm trong memory pool phiên này, nhưng nếu
		// ghi file fail → mất khi restart. Log để biết thay vì âm thầm mất mid.
		slog.Warn("devicepool: mở file pool để ghi thất bại", "file", p.filePath, "mid", mid, "err", err)
		return
	}
	defer f.Close()
	if _, werr := f.WriteString(mid + "|" + datr + "|" + igDID + "\n"); werr != nil {
		slog.Warn("devicepool: ghi mid vào file pool thất bại", "file", p.filePath, "mid", mid, "err", werr)
	}
}
