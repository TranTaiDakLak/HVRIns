// phonedatabase.go — Đọc số điện thoại từ thư mục PhoneDatabase.
// Port C# RandomPhoneNumberWithCountryCode + ReplaceXSign.
//
// Format file: AnyName=CountryCode.locale_REGION.txt
// Mỗi dòng là pattern số, dùng 'x' hoặc 'X' làm wildcard digit.
// Ví dụ: "+6661xxxxxxx", "+84XXXXXXXXX"
package fakeinfo

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PhoneFromDatabase đọc số điện thoại ngẫu nhiên từ thư mục PhoneDatabase.
//
// Format file trong dir: AnyName=CC.locale.txt (vd: "Vietnam US-GB=VN.vi_VN.txt")
// Parse: phần sau dấu '=' cuối cùng, split theo '.', phần đầu là country code.
//
// Trả về ("", "") nếu:
//   - dir rỗng hoặc không đọc được
//   - không tìm thấy file nào match countryCode
//   - file rỗng hoặc không parse được
//
// countryCode: case-insensitive ("vn", "VN", "Vn" đều match).
func PhoneFromDatabase(countryCode, dir string) (phone, locale string) {
	if dir == "" || countryCode == "" {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	cc := strings.ToUpper(strings.TrimSpace(countryCode))
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Tìm dấu '=' cuối cùng để tách CountryCode.locale.txt
		eqIdx := strings.LastIndex(name, "=")
		if eqIdx < 0 {
			continue
		}
		// rest = "VN.vi_VN.txt" hoặc "VN.vi_VN.txt"
		rest := name[eqIdx+1:]
		// Loại bỏ extension .txt
		rest = strings.TrimSuffix(rest, ".txt")
		// parts[0] = CC, parts[1] = locale
		parts := strings.SplitN(rest, ".", 2)
		if len(parts) < 1 {
			continue
		}
		if strings.ToUpper(parts[0]) != cc {
			continue
		}
		if len(parts) >= 2 {
			locale = parts[1]
		}

		line := phoneDatabaseRandomLine(filepath.Join(dir, name), r)
		if line == "" {
			locale = ""
			return
		}
		phone = sanitizePhone(replaceXDigits(line, r))
		return
	}
	return
}

// sanitizePhone loại bỏ ký tự định dạng (-, space, BOM, dot) khỏi pattern phone,
// chỉ giữ "+" leading + digits. File pattern có thể chứa "+1204-548-xxxx" hoặc
// "+506 71x-xxxx" — FB API yêu cầu E.164 sạch.
func sanitizePhone(p string) string {
	p = strings.TrimSpace(p)
	// Strip UTF-8 BOM (EF BB BF) nếu file có BOM ở đầu
	p = strings.TrimPrefix(p, string([]byte{0xEF, 0xBB, 0xBF}))
	var b strings.Builder
	b.Grow(len(p))
	hasPlus := false
	for _, r := range p {
		if r == '+' && !hasPlus && b.Len() == 0 {
			b.WriteRune(r)
			hasPlus = true
			continue
		}
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
		// Bỏ qua '-', ' ', '.', '(', ')', và mọi ký tự non-digit khác
	}
	return b.String()
}

// replaceXDigits thay thế ký tự 'x' và 'X' trong pattern bằng chữ số ngẫu nhiên.
// Port C# RandomUtils.ReplaceXSign.
func replaceXDigits(pattern string, r *rand.Rand) string {
	out := make([]byte, len(pattern))
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == 'x' || pattern[i] == 'X' {
			out[i] = byte('0' + r.Intn(10))
		} else {
			out[i] = pattern[i]
		}
	}
	return string(out)
}

// phoneDatabaseRandomLine đọc 1 dòng ngẫu nhiên từ file, dùng rand đã seed sẵn.
func phoneDatabaseRandomLine(path string, r *rand.Rand) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := make([]string, 0, 64)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(strings.TrimSpace(line), "\r")
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return lines[r.Intn(len(lines))]
}
