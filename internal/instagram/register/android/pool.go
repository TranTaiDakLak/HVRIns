// pool.go — Datr/machine_id cookie pool cho Android register.
//
// File này gộp 2 file cũ:
//   - cookiepool.go       → CookiePool (round-robin đơn giản) + extractDatr helper
//   - partitioned_pool.go → PartitionedDatrPool (per-slot partition, dùng production)
//
// Ngoài ra export `SharedPool` var — set từ app.go, cũng dùng bởi s23 (s23reg import
// `android.PartitionedDatrPool`). CookiePool giữ lại cho backward compat.
//
// C# mapping:
//   - FacebookMachineIdManager
//   - FacebookLogoutSessionUtils
//   - PartitionedDatrPool.cs (mỗi goroutine có partition riêng để tránh trùng datr)
package android

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// SharedPool là partitioned datr pool — mỗi slot goroutine có queue riêng.
// Set từ app.go trước khi batch bắt đầu.
var SharedPool *PartitionedDatrPool

// reDatr extract datr value từ cookie line (dùng chung cho cả CookiePool và PartitionedDatrPool).
var reDatr = regexp.MustCompile(`datr=([A-Za-z0-9_-]+)`)

// extractDatr tìm giá trị datr từ 1 dòng text.
func extractDatr(line string) string {
	if m := reDatr.FindStringSubmatch(line); len(m) > 1 {
		return m[1]
	}
	return ""
}

func validDatr(datr string) bool {
	datr = strings.TrimSpace(datr)
	return datr != "" && !strings.HasPrefix(datr, "_") && !strings.HasPrefix(datr, "-")
}

// ─── CookiePool (round-robin đơn giản — backward compat) ─────────────────────
//
// PORT từ C#: FacebookMachineIdManager + FacebookLogoutSessionUtils
// Flow: Load datr từ file cookie_initial.txt → track usage count → mỗi reg lấy 1
// datr chưa quá limit.
// Format file: uid|password|cookie_string|token|email|fullname|date|country
// datr được extract từ cookie_string: "...datr=xxx;..."

// CookiePool quản lý datr/machine_id pool cho register.
// Thread-safe — dùng cho nhiều goroutines đồng thời.
type CookiePool struct {
	mu       sync.Mutex
	datrs    []string       // danh sách datr đã load
	usage    map[string]int // đếm số lần dùng mỗi datr
	maxUsage int            // giới hạn tối đa mỗi datr (C#: LimitCookieInitialCount, default 9999)
	idx      int            // round-robin index
}

// NewCookiePool tạo pool mới với max usage limit.
func NewCookiePool(maxUsage int) *CookiePool {
	if maxUsage <= 0 {
		maxUsage = 9999
	}
	return &CookiePool{
		usage:    make(map[string]int),
		maxUsage: maxUsage,
	}
}

// LoadFromLines load datr từ danh sách dòng cookie_initial.
// Format mỗi dòng: uid|password|cookie_string|token|email|fullname|date|country.
// Extract datr từ cookie_string.
func (p *CookiePool) LoadFromLines(lines []string) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	count := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		datr := extractDatr(line)
		if !validDatr(datr) {
			continue
		}
		if _, exists := p.usage[datr]; exists {
			continue
		}
		p.datrs = append(p.datrs, datr)
		p.usage[datr] = 0
		count++
	}
	return count
}

// Acquire lấy 1 datr chưa quá limit, tăng usage counter.
// Trả về datr string, "" nếu pool rỗng hoặc tất cả đã quá limit.
func (p *CookiePool) Acquire() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.datrs) == 0 {
		return ""
	}

	// Round-robin tìm datr chưa quá limit
	for i := 0; i < len(p.datrs); i++ {
		idx := (p.idx + i) % len(p.datrs)
		datr := p.datrs[idx]
		if p.usage[datr] < p.maxUsage {
			p.usage[datr]++
			p.idx = (idx + 1) % len(p.datrs)
			return datr
		}
	}
	return "" // tất cả đã quá limit
}

// AddDatr thêm datr mới vào pool (ví dụ từ account vừa reg thành công).
func (p *CookiePool) AddDatr(datr string) {
	if !validDatr(datr) {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.usage[datr]; !exists {
		p.datrs = append(p.datrs, datr)
		p.usage[datr] = 0
	}
}

// Size trả về số lượng datr trong pool.
func (p *CookiePool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.datrs)
}

// ─── PartitionedDatrPool (per-slot partition, dùng production) ───────────────
//
// Điểm khác biệt so với CookiePool:
//   - CookiePool: round-robin chung, nhiều goroutine có thể dùng cùng datr tuần tự
//   - PartitionedDatrPool: mỗi slot (goroutine) có queue riêng → không bao giờ trùng
//
// Lifecycle:
//
//	Register(slotIdx)   — gọi khi goroutine bắt đầu
//	GetNext(slotIdx)    — lấy datr chưa quá limit (rotate trong queue)
//	IncrementUsage(d)   — gọi sau mỗi lần reg (thành công hoặc thất bại)
//	AddDatr(d)          — datr mới từ reg thành công → phân phối vào pool
//	Unregister(slotIdx) — gọi khi goroutine kết thúc, datr còn lại chuyển cho slot khác

// PartitionedDatrPool là pool datr phân vùng theo slot (goroutine).
// Thread-safe qua sync.Mutex.
type PartitionedDatrPool struct {
	mu              sync.Mutex
	partitions      map[int][]string
	activeSlots     []int
	pending         []string
	usageCount      map[string]int
	successCount    map[string]int
	failCount       map[string]int
	unknownCount    map[string]int
	checkpointCount map[string]int
	loadedAt        map[string]time.Time // thời điểm datr được nạp vào pool (cho age expiry)
	exhausted       []string
	exhaustedSet    map[string]struct{}
	activeCount        int
	maxUsage           int
	maxCheckpoint      int
	maxAge             time.Duration // 0 = không expire theo tuổi
	fillBatch          int
	distributeIdx      int
	persistOnlyNew     bool
	deleteOnUsageLimit bool
	persistHook        func(string)
	removeHook         func(string)
}

// SetPersistHook đăng ký callback được gọi mỗi khi có datr MỚI (chưa tồn tại trong pool)
// được thêm qua AddDatr/AddDatrRaw. Dùng để persist datr ra file disk.
// Callback chạy ngoài lock để tránh giữ pool quá lâu nếu hook I/O chậm.
// hook nil → clear.
func (p *PartitionedDatrPool) SetPersistHook(hook func(datr string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.persistHook = hook
}

func (p *PartitionedDatrPool) SetRemoveHook(hook func(datr string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.removeHook = hook
}

func (p *PartitionedDatrPool) SetMaxCheckpoint(max int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxCheckpoint = max
}

func (p *PartitionedDatrPool) SetPersistOnlyNewDatr(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.persistOnlyNew = enabled
}

// SetDeleteOnUsageLimit bật chế độ xóa datr khi đạt giới hạn usage
// (LimitCookieInitialCount). Mặc định OFF: datr hết hạn usage chuyển sang
// exhausted để recycle. Khi ON: datr hết hạn usage sẽ bị remove khỏi pool +
// trigger removeHook (xóa khỏi file đĩa).
//
// Dùng kèm SetMaxCheckpoint để có 1 ý niệm thống nhất "Xóa khi đạt giới hạn":
// hết giới hạn usage HOẶC hết giới hạn checkpoint → datr bị loại bỏ hẳn.
func (p *PartitionedDatrPool) SetDeleteOnUsageLimit(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deleteOnUsageLimit = enabled
}

// SetMaxAgeMinutes đặt thời gian tối đa (phút) mà 1 datr được phép tồn tại trong pool
// kể từ lúc được nạp. 0 = không expire theo tuổi (mặc định).
// Datr quá tuổi sẽ được remove khi ExpireOldDatrs() chạy.
func (p *PartitionedDatrPool) SetMaxAgeMinutes(minutes int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if minutes <= 0 {
		p.maxAge = 0
		return
	}
	p.maxAge = time.Duration(minutes) * time.Minute
}

// ExpireOldDatrs quét pool, remove các datr có tuổi vượt maxAge (nếu được set).
// Fire removeHook cho mỗi datr bị xóa. Trả về số datr đã xóa.
// Gọi định kỳ từ background goroutine.
func (p *PartitionedDatrPool) ExpireOldDatrs() int {
	p.mu.Lock()
	if p.maxAge <= 0 || len(p.loadedAt) == 0 {
		p.mu.Unlock()
		return 0
	}
	cutoff := time.Now().Add(-p.maxAge)
	var expired []string
	for d, t := range p.loadedAt {
		if t.Before(cutoff) {
			expired = append(expired, d)
		}
	}
	for _, d := range expired {
		p.removeDatrLocked(d)
	}
	hook := p.removeHook
	p.mu.Unlock()
	if hook != nil {
		for _, d := range expired {
			hook(d)
		}
	}
	return len(expired)
}

// NewPartitionedPool tạo PartitionedDatrPool mới với giới hạn maxUsage.
func NewPartitionedPool(maxUsage int) *PartitionedDatrPool {
	if maxUsage <= 0 {
		maxUsage = 9999
	}
	return &PartitionedDatrPool{
		partitions:      make(map[int][]string),
		usageCount:      make(map[string]int),
		successCount:    make(map[string]int),
		failCount:       make(map[string]int),
		unknownCount:    make(map[string]int),
		checkpointCount: make(map[string]int),
		loadedAt:        make(map[string]time.Time),
		exhaustedSet:    make(map[string]struct{}),
		maxUsage:        maxUsage,
		fillBatch:       64,
	}
}

// RecordResult ghi nhận kết quả reg cho 1 datr.
// outcome: "success" | "fail" | "unknown" — đếm riêng từng loại.
// Gọi sau khi reg xong để track hiệu năng của từng datr.
func (p *PartitionedDatrPool) RecordResult(datr, outcome string) {
	if !validDatr(datr) {
		return
	}
	p.mu.Lock()
	remove := false
	hook := p.removeHook
	switch outcome {
	case "success":
		p.successCount[datr]++
		p.checkpointCount[datr] = 0
	case "fail":
		p.failCount[datr]++
	case "checkpoint":
		p.checkpointCount[datr]++
		if p.maxCheckpoint > 0 && p.checkpointCount[datr] >= p.maxCheckpoint {
			remove = p.removeDatrLocked(datr)
		}
	default:
		p.unknownCount[datr]++
	}
	p.mu.Unlock()
	if remove && hook != nil {
		hook(datr)
	}
}

// GetStats trả về (success, fail, unknown, usage) của 1 datr.
// usage = tổng lần đã dùng (bao gồm cả chưa có result).
func (p *PartitionedDatrPool) GetStats(datr string) (success, fail, unknown, usage int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.successCount[datr], p.failCount[datr], p.unknownCount[datr], p.usageCount[datr]
}

// Register đăng ký 1 slot vào pool.
// Steal 1/n datrs từ các slot đang active để cân bằng partition (C# behavior).
// Gọi ngay khi goroutine worker bắt đầu.
//
// RESTORED 2026-05-15 from 4/23 commit 8211874: implementation thực sự steal.
// Bug trước đây: chỉ gọi fillSlotLocked() (grab 64 datrs từ pending). Với 200 luồng
// + 9901 datrs, slots 1-154 lấy hết pending → slots 155-200 nhận 0 datr → 30% slot
// không reg được do "Thiếu datr". Sau khi restore steal, mỗi slot lấy 1/n của
// mỗi slot active → phân phối ĐỀU 49 datrs/slot cho 200 slots.
func (p *PartitionedDatrPool) Register(slotIdx int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.partitions[slotIdx]; exists {
		return
	}
	p.partitions[slotIdx] = nil
	p.activeSlots = append(p.activeSlots, slotIdx)

	// STEAL 1/n từ mỗi slot đang chạy để cân bằng (port C# behavior).
	n := len(p.activeSlots)
	if n > 1 {
		for _, existing := range p.activeSlots {
			if existing == slotIdx {
				continue
			}
			q := p.partitions[existing]
			stealCount := len(q) / n
			if stealCount == 0 {
				continue
			}
			stolen := make([]string, stealCount)
			copy(stolen, q[:stealCount])
			p.partitions[existing] = q[stealCount:]
			p.partitions[slotIdx] = append(p.partitions[slotIdx], stolen...)
		}
	}

	// Fill thêm từ pending nếu vẫn còn (cho slot đầu tiên hoặc khi pending còn data).
	p.fillSlotLocked(slotIdx)

	// Phân phối lại pending để các slot khác không thiếu.
	p.distributePending()
}

// Unregister hủy đăng ký slot.
// Datrs còn lại trong partition được trả về pending và phân phối cho các slot còn active.
// Gọi khi goroutine worker kết thúc (nên dùng defer).
func (p *PartitionedDatrPool) Unregister(slotIdx int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	q, exists := p.partitions[slotIdx]
	if !exists {
		return
	}
	p.pending = append(p.pending, q...)
	delete(p.partitions, slotIdx)

	for i, s := range p.activeSlots {
		if s == slotIdx {
			p.activeSlots = append(p.activeSlots[:i], p.activeSlots[i+1:]...)
			break
		}
	}
	if len(p.activeSlots) > 0 {
		for _, slot := range p.activeSlots {
			p.fillSlotLocked(slot)
		}
	}
}

// GetNext lấy 1 datr từ partition của slotIdx có usage < maxUsage.
// Datr được rotate về cuối queue (để dùng lại tối đa maxUsage lần).
// Trả về "" nếu partition rỗng hoặc tất cả đã hết limit.
//
// FIX 2026-05-15: nếu partition rỗng + pending rỗng → STEAL từ slot giàu nhất.
// Trước đây với 200 slots + 9901 datrs, sau khi steal-on-Register cân bằng (~49 datrs/slot),
// một số slot bị stolen xuống 0 trong khi slot khác vẫn có. GetNext không cứu được.
// → Add fallback steal-on-GetNext: lấy ½ từ richest slot.
func (p *PartitionedDatrPool) GetNext(slotIdx int) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Ensure partition exists (idempotent — register slot tự động nếu chưa)
	if _, ok := p.partitions[slotIdx]; !ok {
		p.partitions[slotIdx] = nil
	}

	q := p.partitions[slotIdx]
	if len(q) == 0 {
		p.fillSlotLocked(slotIdx)
		q = p.partitions[slotIdx]
	}
	// FALLBACK 1: steal ½ từ richest slot khác (FIX cho 200+ luồng + pool nhỏ).
	if len(q) == 0 {
		bestSlot := -1
		bestCount := 0
		for s, sq := range p.partitions {
			if s == slotIdx {
				continue
			}
			if len(sq) > bestCount {
				bestCount = len(sq)
				bestSlot = s
			}
		}
		if bestSlot >= 0 && bestCount > 0 {
			stealCount := bestCount / 2
			if stealCount < 1 {
				stealCount = 1
			}
			srcQ := p.partitions[bestSlot]
			stolen := make([]string, stealCount)
			copy(stolen, srcQ[:stealCount])
			p.partitions[bestSlot] = srcQ[stealCount:]
			q = append(q, stolen...)
			p.partitions[slotIdx] = q
		}
	}
	if len(q) == 0 && p.activeCount == 0 && len(p.exhausted) > 0 {
		p.recycleExhaustedLocked()
		p.fillSlotLocked(slotIdx)
		q = p.partitions[slotIdx]
	}
	attempts := len(q)
	for attempts > 0 {
		attempts--
		if len(q) == 0 {
			break
		}
		val := q[0]
		q = q[1:]
		if _, exists := p.usageCount[val]; !exists {
			continue
		}
		if _, exhausted := p.exhaustedSet[val]; exhausted {
			continue
		}
		usage := p.usageCount[val]
		if usage < p.maxUsage {
			q = append(q, val) // rotate lại cuối — dùng được tiếp
			p.partitions[slotIdx] = q
			return val
		}
		p.partitions[slotIdx] = q
		p.exhaustDatrLocked(val)
		q = p.partitions[slotIdx]
		// Usage limit reached: move to exhausted; it will be recycled when the active pool is empty.
	}
	p.partitions[slotIdx] = q
	if len(q) == 0 {
		p.fillSlotLocked(slotIdx)
		if len(p.partitions[slotIdx]) == 0 && p.activeCount == 0 && len(p.exhausted) > 0 {
			p.recycleExhaustedLocked()
			p.fillSlotLocked(slotIdx)
		}
	}
	return ""
}

// IncrementUsage tăng counter sau mỗi lần reg (thành công hay thất bại).
// Mapping C#: AddOrIncrementMachineIdUsage.
//
// Khi datr đạt maxUsage:
//   - deleteOnUsageLimit=false (mặc định): chuyển sang exhausted để recycle khi pool cạn
//   - deleteOnUsageLimit=true: remove hẳn + fire removeHook (xóa khỏi file đĩa)
func (p *PartitionedDatrPool) IncrementUsage(datr string) {
	if !validDatr(datr) {
		return
	}
	p.mu.Lock()
	p.usageCount[datr]++
	remove := false
	hook := p.removeHook
	if p.maxUsage > 0 && p.usageCount[datr] >= p.maxUsage {
		if p.deleteOnUsageLimit {
			remove = p.removeDatrLocked(datr)
		} else {
			p.exhaustDatrLocked(datr)
		}
	}
	p.mu.Unlock()
	if remove && hook != nil {
		hook(datr)
	}
}

// AddDatr thêm datr mới từ account vừa reg thành công.
// Phân phối ngay vào slot hiện có (round-robin) hoặc pending nếu chưa có slot nào.
// Trigger persistHook nếu datr chưa có trong pool.
// Mapping C#: TryAddNewDatrToPool.
func (p *PartitionedDatrPool) AddDatr(datr string) {
	if !validDatr(datr) {
		return
	}
	p.mu.Lock()
	_, existed := p.usageCount[datr]
	if !existed {
		p.usageCount[datr] = 0
		p.loadedAt[datr] = time.Now()
		p.activeCount++
		if !p.persistOnlyNew {
			p.addOneLocked(datr)
		}
	}
	hook := p.persistHook
	p.mu.Unlock()

	if !existed && hook != nil {
		hook(datr)
	}
}

// LoadFromLines load hàng loạt datr từ cookie lines (startup).
// Trả về số datr đã load thành công.
func (p *PartitionedDatrPool) LoadFromLines(lines []string) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	count := 0
	for _, line := range lines {
		datr := extractDatr(line)
		if datr == "" {
			datr = strings.TrimSpace(line)
		}
		if !validDatr(datr) {
			continue
		}
		if _, exists := p.usageCount[datr]; exists {
			continue // dedup
		}
		p.usageCount[datr] = 0
		p.loadedAt[datr] = time.Now()
		p.activeCount++
		p.addOneLocked(datr)
		count++
	}
	return count
}

// AddDatrRaw thêm datr value trực tiếp (không cần parse từ cookie line).
// Trigger persistHook nếu datr chưa có trong pool.
// Trả về true nếu datr là MỚI (chưa tồn tại trong pool), false nếu đã có sẵn.
func (p *PartitionedDatrPool) AddDatrRaw(datr string) bool {
	return p.addDatrRaw(datr, true)
}

// AddDatrRawNoPersist thêm datr vào RAM pool nhưng không gọi persistHook.
// Dùng khi 1 datr mới đã được platform hiện tại persist rồi và cần sync sang
// các platform pool khác, tránh vòng lặp hook ghi file.
func (p *PartitionedDatrPool) AddDatrRawNoPersist(datr string) bool {
	return p.addDatrRaw(datr, false)
}

// LoadFromFileTail đọc lastN dòng cuối của file, extract datr từ mỗi dòng và
// nạp vào pool. Dùng cho fallback khi pool cạn — lấy datr từ SuccessVerify_No2FA.txt
// (lấy NEWEST entries, không phải oldest).
//
// Implementation: scan toàn bộ file 1 lần, giữ ring buffer lastN dòng cuối.
// Phù hợp file <50MB; lớn hơn nên dùng reverse-seek nhưng case này không cần.
func (p *PartitionedDatrPool) LoadFromFileTail(path string, lastN int) (int, error) {
	path = strings.TrimSpace(path)
	if path == "" || lastN <= 0 {
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

	tail := make([]string, 0, lastN)
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if len(tail) < lastN {
			tail = append(tail, line)
		} else {
			tail = append(tail[1:], line)
		}
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	count := 0
	for _, line := range tail {
		datr := extractDatr(line)
		if datr == "" {
			continue // không có datr trong dòng → skip (file SuccessVerify phải có)
		}
		if p.addDatrLoaded(datr) {
			count++
		}
	}
	return count, nil
}

// LoadFromFile streams cookie_initial.txt into the pool without keeping all raw
// cookie/account lines in memory.
func (p *PartitionedDatrPool) LoadFromFile(path string) (int, error) {
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
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		datr := extractDatr(line)
		if datr == "" {
			datr = line
		}
		if p.addDatrLoaded(datr) {
			count++
		}
	}
	if err := sc.Err(); err != nil {
		return count, err
	}
	return count, nil
}

func (p *PartitionedDatrPool) addDatrRaw(datr string, persist bool) bool {
	if !validDatr(datr) {
		return false
	}
	p.mu.Lock()
	_, existed := p.usageCount[datr]
	if !existed {
		p.usageCount[datr] = 0
		p.loadedAt[datr] = time.Now()
		p.activeCount++
		if !p.persistOnlyNew {
			p.addOneLocked(datr)
		}
	}
	hook := p.persistHook
	p.mu.Unlock()

	if persist && !existed && hook != nil {
		hook(datr)
	}
	return !existed
}

func (p *PartitionedDatrPool) addDatrLoaded(datr string) bool {
	if !validDatr(datr) {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.usageCount[datr]; exists {
		return false
	}
	p.usageCount[datr] = 0
	p.loadedAt[datr] = time.Now()
	p.activeCount++
	p.addOneLocked(datr)
	return true
}

// Size trả về tổng số datr còn trong pool (pending + tất cả partitions).
func (p *PartitionedDatrPool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.activeCount
}

// IsCompletelyEmpty báo cáo pool không còn datr nào cả (kể cả exhausted chờ recycle).
// Size()==0 vẫn có thể recycle exhausted; IsCompletelyEmpty==true thì thực sự trống.
func (p *PartitionedDatrPool) IsCompletelyEmpty() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.activeCount == 0 && len(p.exhausted) == 0
}

func (p *PartitionedDatrPool) RemoveDatr(datr string) bool {
	if !validDatr(datr) {
		return false
	}
	p.mu.Lock()
	removed := p.removeDatrLocked(datr)
	hook := p.removeHook
	p.mu.Unlock()
	if removed && hook != nil {
		hook(datr)
	}
	return removed
}

// ── internal helpers ──────────────────────────────────────────────────────────

// addOneLocked phân phối 1 datr vào pool. Phải gọi trong lock.
func (p *PartitionedDatrPool) addOneLocked(datr string) {
	if len(p.activeSlots) == 0 {
		p.pending = append(p.pending, datr)
		return
	}
	idx := p.distributeIdx % len(p.activeSlots)
	p.distributeIdx++
	slot := p.activeSlots[idx]
	p.partitions[slot] = append(p.partitions[slot], datr)
}

// distributePending phân phối tất cả pending round-robin vào các active slots.
// Phải gọi trong lock.
func (p *PartitionedDatrPool) removeDatrLocked(datr string) bool {
	removed := false
	_, wasExhausted := p.exhaustedSet[datr]
	filter := func(in []string) []string {
		out := in[:0]
		for _, v := range in {
			if v == datr {
				removed = true
				continue
			}
			out = append(out, v)
		}
		return out
	}
	for slot, q := range p.partitions {
		p.partitions[slot] = filter(q)
	}
	p.pending = filter(p.pending)
	if _, ok := p.usageCount[datr]; ok {
		removed = true
		if !wasExhausted && p.activeCount > 0 {
			p.activeCount--
		}
	}
	delete(p.usageCount, datr)
	delete(p.successCount, datr)
	delete(p.failCount, datr)
	delete(p.unknownCount, datr)
	delete(p.checkpointCount, datr)
	delete(p.loadedAt, datr)
	delete(p.exhaustedSet, datr)
	if wasExhausted {
		p.exhausted = filter(p.exhausted)
	}
	return removed
}

func (p *PartitionedDatrPool) exhaustDatrLocked(datr string) {
	if !validDatr(datr) {
		return
	}
	if _, exists := p.usageCount[datr]; !exists {
		return
	}
	if _, exists := p.exhaustedSet[datr]; exists {
		return
	}
	filter := func(in []string) []string {
		out := in[:0]
		for _, v := range in {
			if v == datr {
				continue
			}
			out = append(out, v)
		}
		return out
	}
	for slot, q := range p.partitions {
		p.partitions[slot] = filter(q)
	}
	p.pending = filter(p.pending)
	p.exhausted = append(p.exhausted, datr)
	p.exhaustedSet[datr] = struct{}{}
	if p.activeCount > 0 {
		p.activeCount--
	}
}

func (p *PartitionedDatrPool) recycleExhaustedLocked() {
	if len(p.exhausted) == 0 {
		return
	}
	recycled := p.exhausted
	p.exhausted = nil
	p.exhaustedSet = make(map[string]struct{})
	for _, datr := range recycled {
		if !validDatr(datr) {
			continue
		}
		if _, exists := p.usageCount[datr]; !exists {
			continue
		}
		p.usageCount[datr] = 0
		p.activeCount++
		p.addOneLocked(datr)
	}
}

func (p *PartitionedDatrPool) fillSlotLocked(slotIdx int) {
	if _, ok := p.partitions[slotIdx]; !ok {
		return
	}
	if p.fillBatch <= 0 {
		p.fillBatch = 64
	}
	q := p.partitions[slotIdx]
	for len(q) < p.fillBatch && len(p.pending) > 0 {
		datr := p.pending[0]
		p.pending = p.pending[1:]
		if _, exists := p.usageCount[datr]; !exists {
			continue
		}
		if _, exhausted := p.exhaustedSet[datr]; exhausted {
			continue
		}
		if p.maxUsage > 0 && p.usageCount[datr] >= p.maxUsage {
			p.exhaustDatrLocked(datr)
			continue
		}
		q = append(q, datr)
	}
	p.partitions[slotIdx] = q
}

func (p *PartitionedDatrPool) distributePending() {
	if len(p.pending) == 0 || len(p.activeSlots) == 0 {
		return
	}
	for _, datr := range p.pending {
		idx := p.distributeIdx % len(p.activeSlots)
		p.distributeIdx++
		slot := p.activeSlots[idx]
		p.partitions[slot] = append(p.partitions[slot], datr)
	}
	p.pending = nil
}
