// extras.go — Web Android response parsers + helpers.
//
// File này đổi tên từ parser.go cũ (giữ content nguyên, chỉ tên khác) để đồng bộ
// naming pattern với các package register khác (s23/android/ioshttp/webandroid).
//
// Mapping từ C#: FacebookRequestFormDataPropModel.BuildFromChromeAndroidPage()
// + Regex extractions trong FacebookRegisterWebAndroidAPI.Register().
package webandroid

import (
	"regexp"
	"strings"
)

// pageTokens chứa tokens extract từ GET m.facebook.com/.
type pageTokens struct {
	versioningID string // bloks versioning ID (extract từ HTML)
	fbDtsg       string // fb_dtsg CSRF token
	lsd          string // LSD anti-CSRF token
	hsi          string // HTTP Signature Identifier
	jazoest      string // security token phụ
	spinR        string // __spin_r / __rev
	spinT        string // __spin_t
}

// parsePageTokens extract tất cả tokens từ HTML của m.facebook.com/.
// Mapping từ C#: BuildFromChromeAndroidPage() + versioningID regex.
func parsePageTokens(html string) pageTokens {
	t := pageTokens{}

	// versioningID — C#: Regex.Match(resstr, "versioningID:\"(.*?)\"")
	t.versioningID = reFind(html, `versioningID:"(.*?)"`, 1)
	if t.versioningID == "" {
		t.versioningID = "702c2f684e5cb91415ff73ea04c6b82d5580487fbd0a90975765b0adee500940"
	}

	// fb_dtsg — C#: dtsg":{"token":"(.*?)",  (KHÔNG có fallback — chỉ dùng pattern chính xác)
	t.fbDtsg = reFind(html, `dtsg":{"token":"(.*?)",`, 1)

	// lsd — C#: LSD"(.*?)token":"(.*?)"
	t.lsd = reFind(html, `LSD".*?token":"(.*?)"`, 1)

	// hsi — C#: hsi":"(.*?)",
	t.hsi = reFind(html, `hsi":"(.*?)",`, 1)

	// jazoest — C#: jazoest", "(\d+)",
	t.jazoest = reFind(html, `jazoest", "(\d+)",`, 1)

	// spin_r — C#: __spin_r":(.*?),
	t.spinR = reFind(html, `__spin_r":(.*?),`, 1)

	// spin_t — C#: __spin_t":(.*?),
	t.spinT = reFind(html, `__spin_t":(.*?),`, 1)

	return t
}

// parseUID extract UID từ register response.
// C#: Regex.Match(resstr, "currentUser\":(\d+),").
// Bloks mới: "success_response":"{...\"uid\":61573350035998}".
func parseUID(body string) string {
	// Pattern 1: currentUser":{digits} — format cũ
	if uid := reFind(body, `currentUser":(\d+),`, 1); uid != "" {
		return uid
	}
	// Pattern 2: "current_user_id":"digits"
	if uid := reFind(body, `current_user_id":"(\d+)"`, 1); uid != "" {
		return uid
	}
	// Pattern 3: \"uid\":digits — trong Bloks success_response JSON string
	if uid := reFind(body, `\\"uid\\":(\d+)`, 1); uid != "" {
		return uid
	}
	// Pattern 4: SaveCredential bloks action chứa UID dạng string literal
	if uid := reFind(body, `SaveCredential.*?\\"(\d{10,18})\\"`, 1); uid != "" {
		return uid
	}
	return ""
}

// parseLogoutHash extract logout hash từ HTML.
// C#: Regex.Match(resstr, "logout.php.?.h=(.*?)\"").
func parseLogoutHash(html string) string {
	h := reFind(html, `logout\.php[^"]*h=([^"&]+)`, 1)
	return strings.ReplaceAll(h, "amp;", "")
}

// isCheckpoint kiểm tra URL hoặc HTML có phải checkpoint.
func isCheckpoint(urlOrHTML string) bool {
	return strings.Contains(urlOrHTML, "checkpoint")
}

// reFind tìm pattern trong s và trả về group index.
func reFind(s, pattern string, group int) string {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}
	m := re.FindStringSubmatch(s)
	if len(m) > group {
		return m[group]
	}
	return ""
}
