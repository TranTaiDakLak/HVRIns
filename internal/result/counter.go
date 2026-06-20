// counter.go — Counter map thread-safe để track key→count.
//
// Port từ C# FMain._countrysuccessCounter / _fbappversionsuccessCounter / _fblocalesuccessCounter
// dùng chung pattern Dictionary<string,int> + lock.
//
// Dùng cho các file auto-save:
//   CountrySuccess.txt        — VN|150, US|80...
//   FbAppVersisonSuccess.txt  — 554.0.0.57.70|918990560|50
//   FbLocalesSuccess.txt      — vi_VN|100, en_US|50...
//
// Counter tách khỏi CounterSet (counters.go) để dễ test unit riêng biệt.
package result

import (
	"sort"
	"strings"
	"sync"
)

// Counter là map thread-safe của key→count.
// Zero value không valid — dùng NewCounter().
type Counter struct {
	mu   sync.RWMutex
	data map[string]int
}

// NewCounter tạo Counter rỗng.
func NewCounter() *Counter {
	return &Counter{data: make(map[string]int)}
}

// Incr tăng counter của key lên 1. Tạo entry nếu chưa có.
// key rỗng bị bỏ qua (tránh ghi file có dòng "|1" vô nghĩa).
func (c *Counter) Incr(key string) {
	if c == nil {
		return
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	c.mu.Lock()
	c.data[key]++
	c.mu.Unlock()
}

// Add tăng counter của key thêm n (có thể âm). Key rỗng bỏ qua.
func (c *Counter) Add(key string, n int) {
	if c == nil {
		return
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}
	c.mu.Lock()
	c.data[key] += n
	c.mu.Unlock()
}

// Get trả về count hiện tại của key (0 nếu chưa có).
func (c *Counter) Get(key string) int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.data[key]
}

// Snapshot trả về copy map — caller có thể mutate không affect internal state.
func (c *Counter) Snapshot() map[string]int {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make(map[string]int, len(c.data))
	for k, v := range c.data {
		out[k] = v
	}
	return out
}

// Size trả về số entries trong counter.
func (c *Counter) Size() int {
	if c == nil {
		return 0
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// DumpSorted trả về text format "key|count\n" sắp xếp giảm dần theo count
// (key cùng count thì sort alphabetical).
//
// Port C# pattern: user mở file thấy country/version có nhiều success nhất ở đầu.
//
// Trả về chuỗi rỗng nếu counter rỗng (không tạo file).
func (c *Counter) DumpSorted() string {
	snap := c.Snapshot()
	if len(snap) == 0 {
		return ""
	}
	type entry struct {
		key   string
		count int
	}
	list := make([]entry, 0, len(snap))
	for k, v := range snap {
		list = append(list, entry{k, v})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].count != list[j].count {
			return list[i].count > list[j].count // desc by count
		}
		return list[i].key < list[j].key // asc by key
	})
	var sb strings.Builder
	for _, e := range list {
		sb.WriteString(e.key)
		sb.WriteByte('|')
		sb.WriteString(itoa(e.count))
		sb.WriteByte('\n')
	}
	return sb.String()
}

// Reset xóa sạch counter (dùng khi bắt đầu session mới).
func (c *Counter) Reset() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.data = make(map[string]int)
	c.mu.Unlock()
}

// itoa avoids strconv import to keep this file dep-free.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
