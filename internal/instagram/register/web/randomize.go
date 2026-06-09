// randomize.go — Sinh dữ liệu ngẫu nhiên cho tài khoản Facebook
// Tên tiếng Anh/quốc tế, ngày sinh (18-35 tuổi), giới tính
package web

import (
	"fmt"
	"math/rand"
	"time"
)

// Đầu số điện thoại Việt Nam hợp lệ (10 số)
var vnPrefixes = []string{
	// Viettel
	"032", "033", "034", "035", "036", "037", "038", "039",
	"086", "096", "097", "098",
	// Mobifone
	"070", "076", "077", "078", "079", "089", "090", "093",
	// Vinaphone
	"081", "082", "083", "084", "085", "088", "091", "094",
	// Vietnamobile
	"052", "056", "058", "092",
	// Gmobile
	"059", "099",
}

var (
	maleFirstNames = []string{
		"James", "John", "Robert", "Michael", "William", "David", "Richard", "Joseph",
		"Thomas", "Charles", "Christopher", "Daniel", "Matthew", "Anthony", "Mark",
		"Donald", "Steven", "Paul", "Andrew", "Joshua", "Kenneth", "Kevin", "Brian",
		"George", "Timothy", "Ronald", "Edward", "Jason", "Jeffrey", "Ryan",
		"Jacob", "Gary", "Nicholas", "Eric", "Jonathan", "Stephen", "Larry",
		"Justin", "Scott", "Brandon", "Frank", "Benjamin", "Gregory", "Samuel",
		"Raymond", "Patrick", "Alexander", "Jack", "Dennis", "Jerry",
		"Tyler", "Aaron", "Henry", "Jose", "Adam", "Douglas", "Nathan",
		"Peter", "Zachary", "Kyle", "Noah", "Alan", "Sean", "Christian",
	}

	femaleFirstNames = []string{
		"Mary", "Patricia", "Jennifer", "Linda", "Barbara", "Elizabeth", "Susan",
		"Jessica", "Sarah", "Karen", "Lisa", "Nancy", "Betty", "Margaret", "Sandra",
		"Ashley", "Dorothy", "Kimberly", "Emily", "Donna", "Michelle", "Carol",
		"Amanda", "Melissa", "Deborah", "Stephanie", "Rebecca", "Sharon", "Laura",
		"Cynthia", "Kathleen", "Amy", "Angela", "Shirley", "Anna", "Brenda",
		"Pamela", "Emma", "Nicole", "Helen", "Samantha", "Katherine", "Christine",
		"Debra", "Rachel", "Carolyn", "Janet", "Catherine", "Maria", "Heather",
		"Diana", "Julie", "Joyce", "Victoria", "Ruth", "Virginia", "Lauren",
		"Kelly", "Christina", "Joan", "Evelyn", "Judith", "Andrea", "Hannah",
	}

	lastNames = []string{
		"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis",
		"Rodriguez", "Martinez", "Hernandez", "Lopez", "Gonzalez", "Wilson", "Anderson",
		"Thomas", "Taylor", "Moore", "Jackson", "Martin", "Lee", "Perez", "Thompson",
		"White", "Harris", "Sanchez", "Clark", "Ramirez", "Lewis", "Robinson",
		"Walker", "Young", "Allen", "King", "Wright", "Scott", "Torres", "Nguyen",
		"Hill", "Flores", "Green", "Adams", "Nelson", "Baker", "Hall", "Rivera",
		"Campbell", "Mitchell", "Carter", "Roberts", "Gomez", "Phillips", "Evans",
		"Turner", "Diaz", "Parker", "Cruz", "Edwards", "Collins", "Reyes",
		"Stewart", "Morris", "Morales", "Murphy", "Cook", "Rogers", "Gutierrez",
		"Ortiz", "Morgan", "Cooper", "Peterson", "Bailey", "Reed", "Kelly",
		"Howard", "Ramos", "Kim", "Cox", "Ward", "Richardson", "Watson",
		"Brooks", "Chavez", "Wood", "James", "Bennett", "Gray", "Mendoza",
		"Ruiz", "Hughes", "Price", "Alvarez", "Castillo", "Sanders", "Patel",
	}

	pwChars = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
)

// RandomRegInput tạo một RegInput với các trường được sinh ngẫu nhiên.
//
// Tham số:
//   - phone: số điện thoại do caller cung cấp; nếu rỗng thì tự sinh số VN ngẫu nhiên.
//   - password: mật khẩu do caller cung cấp; nếu rỗng thì tự sinh mật khẩu ngẫu nhiên.
//   - proxy: chuỗi proxy "host:port:user:pass"; rỗng → không dùng proxy.
//
// Các trường được sinh ngẫu nhiên trong mọi lần gọi:
//   - Gender: 1 (nữ) hoặc 2 (nam).
//   - FirstName: tên tiếng Anh phù hợp với gender.
//   - LastName: họ tiếng Anh.
//   - Birthday: ngày sinh ngẫu nhiên trong khoảng tuổi 18-35.
//
// Seed được kết hợp từ time.Now().UnixNano() và rand.Int63() để tránh các
// luồng chạy đồng thời sinh ra cùng seed khi gọi liên tiếp trong cùng millisecond.
func RandomRegInput(phone, password, proxy string) RegInput {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))

	gender := r.Intn(2) + 1 // 1=female, 2=male

	var firstName string
	if gender == 2 {
		firstName = maleFirstNames[r.Intn(len(maleFirstNames))]
	} else {
		firstName = femaleFirstNames[r.Intn(len(femaleFirstNames))]
	}
	lastName := lastNames[r.Intn(len(lastNames))]

	birthday := randomBirthday(r)

	if phone == "" {
		phone = randomVNPhone(r)
	}
	if password == "" {
		password = randomPassword(r)
	}

	return RegInput{
		FirstName: firstName,
		LastName:  lastName,
		Birthday:  birthday,
		Gender:    gender,
		Phone:     phone,
		Password:  password,
		Proxy:     proxy,
	}
}

// GeneratePhoneByCountry sinh số điện thoại ngẫu nhiên hợp lệ theo quốc gia.
//
// Tham số:
//   - countryCode: mã quốc gia 2 chữ cái viết thường theo chuẩn ISO 3166-1 alpha-2
//     (ví dụ: "vn", "us", "in", "gh", "br"). Thường lấy từ geolocation của IP proxy.
//
// Định dạng trả về:
//   - "vn": dạng nội địa "0xxxxxxxxx" (10 số, prefix VN hợp lệ).
//   - Các quốc gia khác: dạng quốc tế "+[country_code][number]" theo quy tắc
//     từng mạng di động (prefix, độ dài, v.v.).
//
// Fallback: country code không được hỗ trợ → sinh số VN ngẫu nhiên.
//
// Seed được sinh mới mỗi lần gọi (giống RandomRegInput) để đảm bảo tính
// ngẫu nhiên khi gọi đồng thời từ nhiều goroutine.
func GeneratePhoneByCountry(countryCode string) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + rand.Int63()))
	switch countryCode {
	case "vn": // Vietnam: +84 [3-9]x xxxxxxx (E.164 — match format các country khác)
		return randomVNPhone(r)
	case "cl": // Chile: +56 9xxxxxxxx (8 digits after 9)
		return "+569" + randomDigits(r, 8)
	case "uy": // Uruguay: +598 9xxxxxxx (7 digits after 9)
		return "+5989" + randomDigits(r, 7)
	case "gh": // Ghana: +233 2x/5x xxxxxxx
		pfx := []string{"20", "23", "24", "25", "26", "27", "28", "50", "54", "55", "57", "59"}
		return "+233" + pfx[r.Intn(len(pfx))] + randomDigits(r, 6)
	case "in": // India: +91 [6-9]xxxxxxxxx
		first := []string{"6", "7", "8", "9"}
		return "+91" + first[r.Intn(len(first))] + randomDigits(r, 9)
	case "mx": // Mexico mobile: +52 1 [2-9]xxxxxxxxx
		return "+521" + fmt.Sprintf("%d", 2+r.Intn(8)) + randomDigits(r, 9)
	case "jo": // Jordan: +962 7x xxxxxxx
		return "+9627" + fmt.Sprintf("%d", r.Intn(10)) + randomDigits(r, 7)
	case "py": // Paraguay: +595 9xx xxxxxx
		return "+5959" + fmt.Sprintf("%d%d", r.Intn(10), r.Intn(10)) + randomDigits(r, 6)
	case "bo": // Bolivia: +591 [67]xxxxxxx
		return "+591" + []string{"6", "7"}[r.Intn(2)] + randomDigits(r, 7)
	case "pe": // Peru: +51 9xxxxxxxx
		return "+519" + randomDigits(r, 8)
	case "ec": // Ecuador: +593 9xxxxxxxx
		return "+5939" + randomDigits(r, 8)
	case "co": // Colombia: +57 3xxxxxxxxx
		return "+573" + randomDigits(r, 9)
	case "ar": // Argentina: +54 9 11 xxxxxxxx
		return "+5491" + randomDigits(r, 10)
	case "br": // Brazil: +55 [ddd] 9xxxxxxxx
		ddd := []string{"11", "21", "31", "41", "51", "61", "71", "81", "85", "91"}
		return "+55" + ddd[r.Intn(len(ddd))] + "9" + randomDigits(r, 8)
	case "ng": // Nigeria: +234 [7-9]0xxxxxxxx
		return "+234" + []string{"70", "80", "81", "90", "91"}[r.Intn(5)] + randomDigits(r, 8)
	case "ke": // Kenya: +254 7xx xxxxxx
		return "+2547" + fmt.Sprintf("%d%d", r.Intn(10), r.Intn(10)) + randomDigits(r, 6)
	case "ph": // Philippines: +63 9xxxxxxxxx
		return "+639" + randomDigits(r, 9)
	case "th": // Thailand: +66 [6-9]xxxxxxxx
		return "+66" + []string{"6", "8", "9"}[r.Intn(3)] + randomDigits(r, 8)
	case "id": // Indonesia: +62 8xx xxxxxxxx
		return "+628" + fmt.Sprintf("%d%d", r.Intn(10), r.Intn(10)) + randomDigits(r, 8)
	case "my": // Malaysia: +60 1x xxxxxxxx
		return "+601" + fmt.Sprintf("%d", r.Intn(10)) + randomDigits(r, 8)
	case "us", "ca": // US/Canada: +1 [2-9]xx [2-9]xxxxxx
		area := fmt.Sprintf("%d%d%d", 2+r.Intn(8), r.Intn(10), r.Intn(10))
		return "+1" + area + fmt.Sprintf("%d", 2+r.Intn(8)) + randomDigits(r, 6)
	case "gb": // UK: +44 7xxx xxxxxx
		return "+447" + randomDigits(r, 9)
	case "fr": // France: +33 6/7 xxxxxxxx
		return "+33" + []string{"6", "7"}[r.Intn(2)] + randomDigits(r, 8)
	case "de": // Germany: +49 15x xxxxxxx
		return "+4915" + fmt.Sprintf("%d", r.Intn(10)) + randomDigits(r, 7)
	case "tr": // Turkey: +90 5xx xxxxxxx
		return "+905" + fmt.Sprintf("%d%d", r.Intn(10), r.Intn(10)) + randomDigits(r, 7)
	case "sa": // Saudi Arabia: +966 5xxxxxxxx
		return "+9665" + randomDigits(r, 8)
	case "ae": // UAE: +971 5x xxxxxxx
		return "+9715" + fmt.Sprintf("%d", r.Intn(10)) + randomDigits(r, 7)
	case "eg": // Egypt: +20 1x xxxxxxxxx
		return "+201" + fmt.Sprintf("%d", r.Intn(10)) + randomDigits(r, 9)
	case "pk": // Pakistan: +92 3xx xxxxxxx
		return "+923" + fmt.Sprintf("%d%d", r.Intn(10), r.Intn(10)) + randomDigits(r, 7)
	case "bd": // Bangladesh: +880 1x xxxxxxxx
		return "+8801" + fmt.Sprintf("%d", r.Intn(10)) + randomDigits(r, 8)
	default:
		return ""
	}
}

// randomDigits sinh một chuỗi gồm n chữ số thập phân ngẫu nhiên ('0'-'9').
//
// Tham số:
//   - r: nguồn ngẫu nhiên đã được khởi tạo seed; caller chịu trách nhiệm
//     tạo và tái sử dụng r trong cùng một lần gọi để đảm bảo hiệu suất.
//   - n: số lượng chữ số cần sinh.
//
// Ví dụ: randomDigits(r, 7) có thể trả về "4829301".
// Không có khoảng cách, không có ký tự phân cách; kết quả là string thuần số.
func randomDigits(r *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('0' + r.Intn(10))
	}
	return string(b)
}

// randomVNPhone sinh số điện thoại Việt Nam hợp lệ ở định dạng E.164.
//
// Tham số:
//   - r: nguồn ngẫu nhiên đã được khởi tạo seed.
//
// Cấu trúc: "+84" + [prefix 2 số sau khi strip "0"] + [7 chữ số suffix].
// Prefix được chọn ngẫu nhiên từ vnPrefixes (5 nhà mạng: Viettel, Mobifone,
// Vinaphone, Vietnamobile, Gmobile). vnPrefixes lưu dạng local "0xx" (3 ký
// tự), strip "0" đầu khi build E.164.
//
// Kết quả: "+84xxxxxxxxx" (12 ký tự, gồm dấu +).
//
// Lý do dùng E.164 thay vì local "0xxxxxxxxx":
//   - Nhất quán với các country khác trong GeneratePhoneByCountry (US, Chile,
//     PH, BD... đều dùng E.164).
//   - Facebook reg API expect E.164 — tỉ lệ register thành công cao hơn.
//   - User import phone từ file vẫn dùng path riêng (ConvertPhoneToLocal),
//     không bị ảnh hưởng bởi function này.
func randomVNPhone(r *rand.Rand) string {
	prefix := vnPrefixes[r.Intn(len(vnPrefixes))]
	suffix := fmt.Sprintf("%07d", r.Intn(10000000))
	// Strip "0" đầu của prefix (vd "032" → "32") rồi prepend "+84".
	return "+84" + prefix[1:] + suffix
}

// randomBirthday sinh ngày sinh ngẫu nhiên trong khoảng tuổi 18 đến 35.
//
// Tham số:
//   - r: nguồn ngẫu nhiên đã được khởi tạo seed.
//
// Khoảng tuổi 18-35 được chọn để tài khoản vượt qua kiểm tra tuổi tối thiểu
// của Facebook (13+) với biên độ an toàn, và không quá cao để tránh pattern
// bất thường.
//
// Xử lý ngày trong tháng:
//   - Tháng 1, 3, 5, 7, 8, 10, 12: tối đa 31 ngày.
//   - Tháng 4, 6, 9, 11: tối đa 30 ngày.
//   - Tháng 2: tối đa 28 ngày (không xét năm nhuận để đơn giản hóa).
//
// Định dạng trả về: "DD-MM-YYYY" (ví dụ: "15-07-1998").
func randomBirthday(r *rand.Rand) string {
	now := time.Now()
	ageYears := 18 + r.Intn(18)
	year := now.Year() - ageYears
	month := 1 + r.Intn(12)
	maxDay := 28
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		maxDay = 31
	case 4, 6, 9, 11:
		maxDay = 30
	}
	day := 1 + r.Intn(maxDay)
	return fmt.Sprintf("%02d-%02d-%04d", day, month, year)
}

// randomPassword sinh mật khẩu ngẫu nhiên đủ độ mạnh gồm đúng 10 ký tự.
//
// Tham số:
//   - r: nguồn ngẫu nhiên đã được khởi tạo seed.
//
// Bảng ký tự (pwChars): a-z + A-Z + 0-9 (62 ký tự).
//
// Đảm bảo tính hợp lệ:
//   - Vị trí 0 (ký tự đầu) luôn là chữ hoa A-Z.
//   - Vị trí 9 (ký tự cuối) luôn là chữ số 0-9.
//   - 8 ký tự giữa ngẫu nhiên từ pwChars.
//
// Thiết kế này thỏa mãn yêu cầu mật khẩu Facebook: ít nhất 1 chữ hoa và
// ít nhất 1 chữ số, tổng độ dài 10 ký tự.
func randomPassword(r *rand.Rand) string {
	b := make([]byte, 10)
	for i := range b {
		b[i] = pwChars[r.Intn(len(pwChars))]
	}
	// Đảm bảo có ít nhất 1 chữ hoa, 1 số
	b[0] = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"[r.Intn(26)]
	b[9] = "0123456789"[r.Intn(10)]
	return string(b)
}
