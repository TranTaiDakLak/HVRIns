// Package fakeinfo — Generate fake personal info + device fingerprint for Facebook account creation
// Mapping từ C#: FakePersonalInfoBuilder + AndroidUserAgentBuilder + SimNetworkUtils + FacebookAccountModel.InstanceRandom()
package fakeinfo

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	firstNames   []string // US first names — nạp từ Config/Namereg/US/firstname.txt
	lastNames    []string // US last names — nạp từ Config/Namereg/US/lastname.txt
	vnFirstNames []string // VN first names — nạp từ Config/Namereg/VN/firstname.txt
	vnLastNames  []string // VN last names — nạp từ Config/Namereg/VN/lastname.txt
)

// FakeProfile contains generated fake personal info
type FakeProfile struct {
	FirstName string
	LastName  string
	Birthday  string // "DD-MM-YYYY"
	Gender    int    // 1=female, 2=male
}

// FullRegProfile chứa tất cả thông tin cần thiết cho 1 lần register
// Mapping từ C#: FacebookAccountModel sau InstanceRandom()
type FullRegProfile struct {
	FakeProfile
	Device         DeviceProfile
	Sim            SimProfile
	UserAgent      string
	FbVersion      string // 554.0.0.57.70
	FbBuildNum     string // 918990560
	DeviceID       string // UUID (= C# DeviceId)
	FamilyDeviceID string // UUID with dashes (= C# FamilyDeviceId)
	WaterfallID    string // UUID (= C# Waterfall_id)
	MachineID      string // datr cookie từ cookie pool (= C# MachineId) — bắt đầu rỗng
	MachineID2     string // random 28-char A-Za-z0-9 (= C# MachineId2) — luôn random
	Locale         string // en_US, vi_VN...
	CountryCode    string // US, VN...
	ConnectionType string // WIFI, mobile.LTE
	DeviceGroup    string // random 4-digit
	ConnUUID       string // UUID without dashes (= C# Xfb_conn_uuid_client)
}

// RandomFakeProfile tạo fake name + birthday + gender (dùng US name pool mặc định).
func RandomFakeProfile() FakeProfile {
	return RandomFakeProfileByLocale("")
}

// RandomFakeProfileByLocale tạo fake profile theo locale name database.
// Port C# MainFormUISettings.NameReg: "VN" → đọc VN names, khác → US names.
func RandomFakeProfileByLocale(locale string) FakeProfile {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))

	// Chọn name pool theo locale
	firsts := firstNames
	lasts := lastNames
	if strings.EqualFold(strings.TrimSpace(locale), "VN") && len(vnFirstNames) > 0 && len(vnLastNames) > 0 {
		firsts = vnFirstNames
		lasts = vnLastNames
	}

	firstName := "John"
	lastName := "Smith"
	if len(firsts) > 0 {
		firstName = firsts[r.Intn(len(firsts))]
	}
	if len(lasts) > 0 {
		lastName = lasts[r.Intn(len(lasts))]
	}

	// Birthday: 1970-2001, ngày 1-28, tháng 1-12
	day := r.Intn(28) + 1
	month := r.Intn(12) + 1
	year := 1970 + r.Intn(32) // 1970-2001

	gender := r.Intn(2) + 1 // 1=female, 2=male

	return FakeProfile{
		FirstName: firstName,
		LastName:  lastName,
		Birthday:  fmt.Sprintf("%02d-%02d-%d", day, month, year),
		Gender:    gender,
	}
}

// RandomPassword tạo password ngẫu nhiên (8-12 ký tự, A-Za-z0-9)
func RandomPassword() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length := 8 + r.Intn(5) // 8-12
	pwd := make([]byte, length)
	for i := range pwd {
		pwd[i] = chars[r.Intn(len(chars))]
	}
	return string(pwd)
}

// PasswordFromTemplate sinh password từ template — thay mỗi dấu `*` bằng ký tự
// ngẫu nhiên [A-Za-z0-9]. Port C# RandomUtils.ReplaceAsterisks.
// Vd: "Fb***2025*" → "Fba7X2025y"
// Nếu template rỗng → fallback RandomPassword.
func PasswordFromTemplate(tpl string) string {
	tpl = strings.TrimSpace(tpl)
	if tpl == "" {
		return RandomPassword()
	}
	// Mẫu có nội dung NHƯNG không có '*' nào (vd "Long2299@kk") → nối thêm 4 char random
	// cuối để mỗi account ra pass KHÁC nhau (chống bẫy toàn bộ nick trùng 1 pass cố định).
	// Mẫu đã có '*' → tôn trọng đúng vị trí random user chỉ định, không thêm gì.
	if !strings.Contains(tpl, "*") {
		tpl += "****"
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	out := make([]byte, 0, len(tpl))
	for i := 0; i < len(tpl); i++ {
		if tpl[i] == '*' {
			out = append(out, chars[r.Intn(len(chars))])
		} else {
			out = append(out, tpl[i])
		}
	}
	return string(out)
}

// RandomEmailFromDomains sinh email random từ CSV domain list (C# RandomEmail).
// Format C#: `{a6}.{b6}_{c6}{3num}{domain}` — vd "abcdef.ghijkl_mnopqr123@gmail.com"
// domains rỗng → default "@gmail.com".
func RandomEmailFromDomains(domainsCSV string) string {
	domainsCSV = strings.TrimSpace(domainsCSV)
	if domainsCSV == "" {
		domainsCSV = "@gmail.com"
	}
	parts := strings.Split(domainsCSV, ",")
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, "@") {
			p = "@" + p
		}
		cleaned = append(cleaned, p)
	}
	if len(cleaned) == 0 {
		cleaned = []string{"@gmail.com"}
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	domain := cleaned[r.Intn(len(cleaned))]

	lower := "abcdefghijklmnopqrstuvwxyz"
	digits := "0123456789"
	randStr := func(n int, pool string) string {
		b := make([]byte, n)
		for i := range b {
			b[i] = pool[r.Intn(len(pool))]
		}
		return string(b)
	}
	return randStr(3, lower) + "." + randStr(3, lower) + randStr(3, lower) + randStr(5, digits) + domain
}

// normalizeNameASCII strips diacritics and keeps only a-z letters (lowercase).
// Handles Vietnamese: "Nguyễn Văn Hải" → "nguyenvanhai".
func normalizeNameASCII(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)
	out := make([]byte, 0, len(result))
	for _, c := range result {
		if c >= 'a' && c <= 'z' {
			out = append(out, byte(c))
		}
	}
	return string(out)
}

// parseBirthdayParts splits "DD-MM-YYYY" → ("22", "05", "1990").
func parseBirthdayParts(birthday string) (day, month, year string) {
	parts := strings.SplitN(birthday, "-", 3)
	if len(parts) == 3 {
		return parts[0], parts[1], parts[2]
	}
	return "01", "01", "2000"
}

// pickEmailDomain chọn ngẫu nhiên 1 domain từ CSV list.
func pickEmailDomain(domainsCSV string, r *rand.Rand) string {
	domainsCSV = strings.TrimSpace(domainsCSV)
	if domainsCSV == "" {
		return "@gmail.com"
	}
	parts := strings.Split(domainsCSV, ",")
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, "@") {
			p = "@" + p
		}
		cleaned = append(cleaned, p)
	}
	if len(cleaned) == 0 {
		return "@gmail.com"
	}
	return cleaned[r.Intn(len(cleaned))]
}

// EmailFromProfile sinh email dựa trên tên + ngày sinh để trông giống email thật.
// 30 pattern đa dạng: dot/underscore/concat, reversed, truncated, initial, embedded date.
// Mọi pattern đều có ít nhất rand(2-4) ký tự để giảm trùng lặp với account đã có.
// domains rỗng → default "@gmail.com".
func EmailFromProfile(firstName, lastName, birthday, domainsCSV string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	domain := pickEmailDomain(domainsCSV, r)

	fn := normalizeNameASCII(firstName)
	ln := normalizeNameASCII(lastName)
	if fn == "" {
		fn = "user"
	}
	if ln == "" {
		ln = "acc"
	}

	day, month, year := parseBirthdayParts(birthday)
	yy := year
	if len(year) == 4 {
		yy = year[2:]
	}

	const alphanum = "abcdefghijklmnopqrstuvwxyz0123456789"
	const alpha = "abcdefghijklmnopqrstuvwxyz"
	const digits = "0123456789"
	randStr := func(n int, pool string) string {
		b := make([]byte, n)
		for i := range b {
			b[i] = pool[r.Intn(len(pool))]
		}
		return string(b)
	}
	// rn returns a random length in [lo, hi]
	rn := func(lo, hi int) int { return lo + r.Intn(hi-lo+1) }

	const maxPart = 10
	if len(fn) > maxPart {
		fn = fn[:maxPart]
	}
	if len(ln) > maxPart {
		ln = ln[:maxPart]
	}
	fi := string(fn[0])
	li := string(ln[0])

	// truncated prefixes: first 4 chars (or full if shorter)
	fn4 := fn
	if len(fn4) > 4 {
		fn4 = fn4[:4]
	}
	ln4 := ln
	if len(ln4) > 4 {
		ln4 = ln4[:4]
	}

	patterns := []string{
		// --- dot separator, date + year ---
		fn + "." + ln + day + month + year + randStr(rn(3, 5), alphanum),        // john.smith22051990ab3c
		fn + "." + ln + day + month + year + randStr(rn(4, 6), digits),          // john.smith22051990129384
		fn + "." + ln + year + day + month + randStr(rn(3, 5), alphanum),        // john.smith199022054ab3
		fn + "." + ln + year + month + day + randStr(rn(3, 4), alphanum),        // john.smith19900522abc
		fn + "." + ln + year + randStr(rn(4, 6), alphanum),                      // john.smith1990xyz4ab
		fn + "." + ln + day + month + randStr(rn(4, 6), alphanum),               // john.smith2205ab3c4d
		fn + "." + ln + yy + day + month + randStr(rn(3, 5), alphanum),          // john.smith902205abc4d
		fn + "." + ln + year + li + randStr(rn(3, 5), digits),                   // john.smith1990s4298
		fn + "." + ln + randStr(rn(5, 7), alphanum),                             // john.smithxyz7ab3
		fn + "." + ln + randStr(rn(4, 6), digits) + randStr(rn(2, 3), alpha),    // john.smith198422ab

		// --- dot separator, year in middle / prefix ---
		fn + year + "." + ln + randStr(rn(3, 5), alphanum),                      // john1990.smithab3c
		fn + "." + year + ln + randStr(rn(3, 4), alphanum),                      // john.1990smithabc
		fn + "." + ln + month + year + randStr(rn(3, 5), alphanum),              // john.smith051990ab3c

		// --- underscore separator ---
		fn + "_" + ln + day + month + year + randStr(rn(3, 5), alphanum),        // john_smith22051990ab3
		fn + "_" + ln + year + randStr(rn(3, 5), alphanum),                      // john_smith1990abc4d
		fn + "_" + ln + yy + randStr(rn(4, 6), alphanum),                        // john_smith90abc5d3
		fn + "_" + year + "_" + ln + randStr(rn(3, 5), alphanum),                // john_1990_smithabc4
		ln + "_" + fn + day + month + year + randStr(rn(2, 4), alphanum),        // smith_john22051990ab
		fn + "_" + ln + randStr(rn(4, 6), digits) + randStr(rn(2, 3), alpha),    // john_smith198422ab

		// --- no separator ---
		fn + ln + day + month + year + randStr(rn(3, 5), alphanum),              // johnsmith22051990ab3
		fn + ln + year + day + month + randStr(rn(3, 4), alphanum),              // johnsmith199022054abc
		fn + ln + yy + randStr(rn(4, 6), alphanum),                              // johnsmith90abcde4
		fn + ln + year + randStr(rn(3, 5), alphanum),                            // johnsmith1990ab3c
		fn + ln + month + day + year + randStr(rn(2, 4), alphanum),              // johnsmith0522199042a

		// --- initial.lastname ---
		fi + "." + ln + day + month + year + randStr(rn(3, 5), alphanum),        // j.smith22051990ab3c
		fi + "." + ln + year + randStr(rn(4, 6), alphanum),                      // j.smith1990xyz4ab
		fi + "." + ln + day + month + randStr(rn(4, 6), alphanum),               // j.smith2205abc4d3
		fi + "." + ln + year + month + randStr(rn(3, 4), alphanum),              // j.smith19900522abc

		// --- initial+lastname (no dot) ---
		fi + ln + day + month + year + randStr(rn(3, 5), alphanum),              // jsmith22051990ab3c
		fi + ln + year + randStr(rn(4, 6), alphanum),                            // jsmith1990ab3cd4
		fi + ln + yy + randStr(rn(4, 6), alphanum),                              // jsmith90ab3c4d

		// --- reversed: ln.fn / ln_fn ---
		ln + "." + fn + day + month + year + randStr(rn(3, 4), alphanum),        // smith.john22051990ab
		ln + "." + fn + year + randStr(rn(4, 6), alphanum),                      // smith.john1990xyz4
		ln + "_" + fn + day + month + year + randStr(rn(2, 4), alphanum),        // smith_john22051990ab
		ln + fn + year + randStr(rn(3, 5), alphanum),                            // smithjohn1990xyz4

		// --- year in middle ---
		fn + year + ln + randStr(rn(3, 5), alphanum),                            // john1990smithab3c
		fn + yy + ln + day + month + randStr(rn(3, 4), alphanum),                // john90smith2205abc
		fn + day + year + ln + randStr(rn(3, 4), alphanum),                      // john221990smithabc

		// --- truncated name + longer suffix ---
		fn4 + ln + year + randStr(rn(4, 6), alphanum),                           // johsmith1990abcde
		fn4 + "." + ln + day + month + year + randStr(rn(3, 4), alphanum),       // joh.smith22051990abc
		fn + ln4 + day + month + year + randStr(rn(3, 4), alphanum),             // johnsmit22051990abc

		// --- initials + full date ---
		fi + li + day + month + year + randStr(rn(4, 6), alphanum),              // js22051990abcde
	}

	return patterns[r.Intn(len(patterns))] + domain
}

// RandomLineFromFile đọc file + trả về 1 dòng random.
// Trả "" nếu file không tồn tại/rỗng/lỗi — caller fallback sang generator khác.
// Port C# RandomUtils.RandomItemInFile.
func RandomLineFromFile(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := make([]string, 0, 256)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimRight(line, "\r")
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	if len(lines) == 0 {
		return ""
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	return lines[r.Intn(len(lines))]
}

// ConvertPhoneToLocal chuyển số E164 → local format bằng cách strip country code,
// prefix "0". Port C# FacebookRegisterAutomation.FmPhoneCode logic:
//
//	"84912345678" (VN) → "0912345678"
//	"+84912345678"      → "0912345678"
//	Phone bắt đầu bằng "0" → giữ nguyên.
//
// countryCode phải là ISO2 ("VN", "US"...); lookup qua PhoneCodeFor.
func ConvertPhoneToLocal(phone, countryCode string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.TrimPrefix(phone, "+")
	if phone == "" {
		return phone
	}
	if strings.HasPrefix(phone, "0") {
		return phone // đã local
	}
	code := PhoneCodeFor(countryCode) // "+84"
	code = strings.TrimPrefix(code, "+")
	if code == "" {
		return phone
	}
	if strings.HasPrefix(phone, code) {
		return "0" + phone[len(code):]
	}
	return phone
}

// BuildFullRegProfile tạo toàn bộ profile cho 1 lần register
// countryCode: "" = random, "VN" = Vietnamese SIM/locale
func BuildFullRegProfile(countryCode string) FullRegProfile {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))

	fake := RandomFakeProfile()
	device := RandomDeviceProfile()
	sim := RandomSimProfile(countryCode)
	locale := LocaleFromCountry(sim.CountryCode)
	fbVer, fbBuild := RandomFbVersion()
	ua := BuildAndroidUA(device, locale, sim.OperatorName, fbVer, fbBuild)

	// MachineID2: random 28-char A-Za-z0-9 (C#: RandomUtils.RandomString(28))
	// KHÔNG có ký tự _ hay - (chỉ alphanumeric như C#)
	const machineChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	machineID2Bytes := make([]byte, 28)
	for i := range machineID2Bytes {
		machineID2Bytes[i] = machineChars[r.Intn(len(machineChars))]
	}

	// DeviceGroup: random 4-digit (C#: Next(1000, 9999) = 1000-9998)
	deviceGroup := fmt.Sprintf("%d", 1000+r.Intn(8999))

	// ConnectionType: chủ yếu WIFI
	connTypes := []string{"WIFI", "WIFI", "WIFI", "mobile.LTE"}
	connType := connTypes[r.Intn(len(connTypes))]

	return FullRegProfile{
		FakeProfile:    fake,
		Device:         device,
		Sim:            sim,
		UserAgent:      ua,
		FbVersion:      fbVer,
		FbBuildNum:     fbBuild,
		DeviceID:       uuid.New().String(),
		FamilyDeviceID: uuid.New().String(),
		WaterfallID:    uuid.New().String(),
		MachineID:      "",                      // sẽ được set từ cookie pool (C#: MachineId bắt đầu rỗng)
		MachineID2:     string(machineID2Bytes), // luôn random (C#: MachineId2 = RandomString(28))
		Locale:         locale,
		CountryCode:    sim.CountryCode,
		ConnectionType: connType,
		DeviceGroup:    deviceGroup,
		ConnUUID:       strings.ReplaceAll(uuid.New().String(), "-", ""),
	}
}

// UserAgentBuilder interface (cho backward compat)
type UserAgentBuilder interface {
	GetUserAgent(locale string, addVirtualSpecs bool, useBuildNumFile bool, simBrand string) string
}
