// phonecode.go — Phone country code lookup
// Data source: Config/phone_database/ directory (per-country txt files).
// Filename format: {CountryName}={CountryCode}.{locale}.txt
//   e.g., "Albania=AL.sq_AL.txt", "ViệtNam US-GB=VN.vi_VN.txt"
// Content format: phone number patterns, one per line, e.g.:
//   +355672xxxxxx
//   +355673xxxxxx
// Trailing 'x' chars = variable digits (stripped when building match prefix).
package fakeinfo

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// PhoneCountry mô tả 1 quốc gia với phone prefix.
// Name:        Tên quốc gia (Vietnam, Albania, ...)
// CountryCode: ISO alpha-2 (VN, AL, ...)
// PhoneCode:   Common prefix của tất cả patterns trong file ("+84", "+3556", ...)
// AreaCode:    Luôn "" trong format mới — giữ để tương thích struct cũ
// Locale:      Locale từ tên file, e.g., "vi_VN", "sq_AL"
type PhoneCountry struct {
	Name        string
	CountryCode string
	PhoneCode   string
	AreaCode    string
	Locale      string
}

var (
	phoneMu         sync.RWMutex
	phoneList       []PhoneCountry          // deduplicated, 1 per country — cho PhoneCountries()
	phoneByCountry  = make(map[string]PhoneCountry) // key = CountryCode
	phonePrefixList []PhoneCountry          // per-line entries — cho FindCountryByPhonePrefix
)

// LoadPhoneDatabase đọc toàn bộ file trong dir vào memory.
// dir: "Config/phone_database" (relative) hoặc absolute path.
// Gọi từ app.go lúc startup sau khi data dir đã xác định.
func LoadPhoneDatabase(dir string) {
	if dir == "" {
		dir = "Config/phone_database"
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	var list []PhoneCountry
	byCC := make(map[string]PhoneCountry)
	var prefixList []PhoneCountry

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".txt") {
			continue
		}

		// Filename: "{CountryName}={CountryCode}.{locale}.txt"
		eqIdx := strings.Index(name, "=")
		if eqIdx < 0 {
			continue
		}
		countryName := strings.TrimSpace(name[:eqIdx])
		rest := strings.TrimSuffix(name[eqIdx+1:], ".txt")

		dotIdx := strings.Index(rest, ".")
		var countryCode, locale string
		if dotIdx >= 0 {
			countryCode = strings.ToUpper(rest[:dotIdx])
			locale = rest[dotIdx+1:]
		} else {
			countryCode = strings.ToUpper(rest)
		}

		if !isAlpha2(countryCode) {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		patterns := parsePhonePatterns(string(data))
		if len(patterns) == 0 {
			continue
		}

		phoneCode := commonPhonePrefix(patterns)
		if phoneCode == "+" || phoneCode == "" {
			phoneCode = patterns[0]
		}

		canonical := PhoneCountry{
			Name: countryName, CountryCode: countryCode,
			PhoneCode: phoneCode, Locale: locale,
		}

		// First file wins for duplicate country codes (data error in filenames).
		if _, exists := byCC[countryCode]; !exists {
			list = append(list, canonical)
			byCC[countryCode] = canonical
		}

		// Per-line entries cho longest-prefix matching.
		for _, pat := range patterns {
			prefixList = append(prefixList, PhoneCountry{
				Name: countryName, CountryCode: countryCode,
				PhoneCode: pat, Locale: locale,
			})
		}
	}

	phoneMu.Lock()
	phoneList = list
	phoneByCountry = byCC
	phonePrefixList = prefixList
	phoneMu.Unlock()
}

// LookupPhoneCode trả về phone country entry theo ISO alpha-2 country code.
// Trả về zero value + false nếu không tìm thấy.
func LookupPhoneCode(countryCode string) (PhoneCountry, bool) {
	cc := strings.ToUpper(strings.TrimSpace(countryCode))
	phoneMu.RLock()
	defer phoneMu.RUnlock()
	p, ok := phoneByCountry[cc]
	return p, ok
}

// PhoneCodeFor trả về phone prefix (kèm "+") cho country code; "" nếu không tìm thấy.
func PhoneCodeFor(countryCode string) string {
	if p, ok := LookupPhoneCode(countryCode); ok {
		return p.PhoneCode
	}
	return ""
}

// FindCountryByPhonePrefix tìm country từ số điện thoại bắt đầu bằng prefix.
// Thuật toán: longest-prefix matching trên phonePrefixList (per-line entries).
// Mỗi entry có PhoneCode = full pattern prefix (e.g., "+355672"), không có AreaCode.
func FindCountryByPhonePrefix(phoneNumber string) (PhoneCountry, bool) {
	num := strings.TrimSpace(phoneNumber)
	if num == "" {
		return PhoneCountry{}, false
	}
	if !strings.HasPrefix(num, "+") {
		num = "+" + num
	}
	phoneMu.RLock()
	defer phoneMu.RUnlock()

	var best PhoneCountry
	bestLen := 0
	for _, p := range phonePrefixList {
		if p.PhoneCode == "" {
			continue
		}
		if strings.HasPrefix(num, p.PhoneCode) && len(p.PhoneCode) > bestLen {
			best = p
			bestLen = len(p.PhoneCode)
		}
	}
	if bestLen > 0 {
		return best, true
	}
	return PhoneCountry{}, false
}

// PhoneCountries trả về snapshot copy của danh sách phone countries (1 per country code).
func PhoneCountries() []PhoneCountry {
	phoneMu.RLock()
	defer phoneMu.RUnlock()
	out := make([]PhoneCountry, len(phoneList))
	copy(out, phoneList)
	return out
}

// parsePhonePatterns đọc lines từ file content, strip trailing 'x', trả về valid prefixes.
func parsePhonePatterns(content string) []string {
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "+") {
			continue
		}
		stripped := strings.TrimRight(line, "x")
		if len(stripped) > 1 {
			out = append(out, stripped)
		}
	}
	return out
}

// commonPhonePrefix trả về common prefix của tất cả strings.
// Phone codes là ASCII nên byte-level slicing là an toàn.
func commonPhonePrefix(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	prefix := lines[0]
	for _, s := range lines[1:] {
		for len(prefix) > 0 && !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
		if prefix == "" {
			return ""
		}
	}
	return prefix
}

// isAlpha2 trả về true nếu s gồm đúng 2 ký tự A-Z.
func isAlpha2(s string) bool {
	return len(s) == 2 && s[0] >= 'A' && s[0] <= 'Z' && s[1] >= 'A' && s[1] <= 'Z'
}
