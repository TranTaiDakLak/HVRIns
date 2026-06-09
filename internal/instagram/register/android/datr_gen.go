// datr_gen.go — Sinh datr cookie mới khi user chọn CookieInitialMethod="new".
//
// Phân tích từ build/bin/Config/Cookie/cookie_initial.txt (~290 datr Facebook
// web thực tế) cho thấy cấu trúc:
//   - 24 ký tự base64url (A-Z, a-z, 0-9, -, _)
//   - Vị trí 3-4 luôn là "Ga" (signature byte: byte[2] low nibble = 0x6)
//   - Vị trí 2 là 1 trong vài tag quan sát được từ pool thực tế: g/o/w/k/s/4/8/0/c
//   - Các vị trí còn lại random
//
// Sinh datr nội bộ giúp user không phụ thuộc file cookie_initial.txt — phù hợp
// khi muốn reg với pool fresh, hoặc khi file bị cạn / chưa có.
package android

import (
	crand "crypto/rand"
	"encoding/base64"
	mrand "math/rand"
	"sync"
	"time"
)

// datrTagChars là tập các ký tự ở vị trí 2 trong datr Facebook, có trọng số gần
// với cookie_initial.txt mẫu. Các vị trí còn lại vẫn là base64url random.
var datrTagChars = []byte{
	'g', 'g', 'g', 'g', 'g', 'g', 'g', 'g',
	'o', 'o', 'o', 'o', 'o', 'o', 'o', 'o',
	'w', 'w', 'w', 'w', 'w', 'w', 'w', 'w',
	'k', 'k', 'k', 'k', 'k', 'k', 'k', 'k',
	's', 's', 's', 's', 's', 's',
	'4', '4', '4', '4',
	'8', '8', '8',
	'0',
	'c',
}

var datrRng = struct {
	mu sync.Mutex
	r  *mrand.Rand
}{r: mrand.New(mrand.NewSource(time.Now().UnixNano()))}

// GenerateDatr trả về 1 datr cookie 24 ký tự mô phỏng định dạng Facebook web.
// Nếu nguồn random hệ thống lỗi, fallback sang math/rand (vẫn duy nhất).
func GenerateDatr() string {
	var raw [18]byte
	if _, err := crand.Read(raw[:]); err != nil {
		datrRng.mu.Lock()
		datrRng.r.Read(raw[:])
		datrRng.mu.Unlock()
	}
	enc := base64.RawURLEncoding.EncodeToString(raw[:])
	// enc luôn dài 24 ký tự (18 byte = 144 bit = 24 base64 char).
	b := []byte(enc)
	// Pin signature 3 ký tự ở vị trí 2-4: tag + "Ga".
	datrRng.mu.Lock()
	tag := datrTagChars[datrRng.r.Intn(len(datrTagChars))]
	datrRng.mu.Unlock()
	b[2] = tag
	b[3] = 'G'
	b[4] = 'a'
	// validDatr loại datr bắt đầu bằng '_'. Tránh để generator bị reject:
	// thay '_' bằng 'A' (giữ entropy đủ vì chỉ 1 vị trí bị clamp).
	if b[0] == '_' {
		b[0] = 'A'
	}
	return string(b)
}

// LoadGenerated sinh n datr mới và thêm vào pool. Dùng khi
// CookieInitialMethod="new" — không có file để load.
// Trả về số datr đã thêm thành công (sau dedup nội bộ).
func (p *PartitionedDatrPool) LoadGenerated(n int) int {
	if n <= 0 {
		return 0
	}
	lines := make([]string, 0, n)
	for i := 0; i < n; i++ {
		lines = append(lines, GenerateDatr())
	}
	return p.LoadFromLines(lines)
}
