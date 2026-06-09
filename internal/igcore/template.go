// template.go — nạp body template từ capture, thay field động.
//
// Chiến lược: body capture (url-encoded, nhiều lớp escape) được dùng nguyên si;
// các giá trị động (device_id, waterfall_id, email, ...) là UUID/token xuất hiện
// VERBATIM trong body bất kể lớp escape → ReplaceAll theo giá trị gốc (capture) →
// giá trị mới là an toàn và đủ.
package igcore

import (
	"embed"
	"fmt"
	"net/url"
	"strings"
)

//go:embed templates/*.txt
var tmplFS embed.FS

// Giá trị GỐC trong capture (để tìm & thay bằng giá trị phiên mới).
const (
	capDeviceID     = "780EF930-6A7E-4536-B0A7-491368931CE3"
	capFamilyDevID  = "37312B3B-7404-491B-967D-72AE026CE3A7"
	capWaterfallID  = "fa6c3900254c4faa9a6d32aa0accd5cd"
	capCloudTrust   = "4A3B0992-83A9-4EA5-BDF0-C2600A6E3828295E487A-B465-4F17-8F3E-2C6B4DF775FC"
	capRegFlowID    = "122925d4-0d65-47ea-8176-fda7e1b75b5f"
	capRegMachineID = "rKwaarWTOxLfrbI_ZPZ-31HW"
	capAacJID       = "1f2a44bd-3898-40c9-ae2f-5f9e5ed65fe4"
	capAacCS        = "iyKrsg4qm822sg0hbZGLN6PmxIWqSLa0Zxx8JbUM5-w"
	capEmail        = "quanvucong2k4%40gmail.com" // url-encode @ (submit)
	capEmailPlain   = "quanvucong2k4@gmail.com"
	capEmailUnicode = "quanvucong2k4%5C%5C%5C%5C%5C%5C%5C%5Cu0040gmail.com" // nested unicode @ (confirm)
	capEventReqID   = "076dd760-2f0e-4571-97ed-50ad62a6dc46"
)

func loadTemplate(name string) (string, error) {
	b, err := tmplFS.ReadFile("templates/" + name)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// applyProfile thay toàn bộ định danh thiết bị động trong body.
func applyProfile(body string, p *igProfile) string {
	r := strings.NewReplacer(
		capDeviceID, p.DeviceID,
		capFamilyDevID, p.FamilyDeviceID,
		capWaterfallID, p.WaterfallID,
		capCloudTrust, p.CloudTrustID,
		capRegFlowID, p.RegFlowID,
		capRegMachineID, p.RegMachineID,
	)
	return r.Replace(body)
}

// setEmail thay email trong tất cả dạng encode (3 lớp escape khác nhau tùy bước).
func setEmail(body, email string) string {
	enc := url.QueryEscape(email) // a@b.com → a%40b.com

	// Dạng unicode encode cho nested JSON: @ → @ rồi url-encode nhiều lớp.
	// capEmailUnicode hardcode domain gmail.com — cần thay cả chuỗi đó bằng email mới.
	// Tạo dạng unicode của email mới: user%5C%5C%5C%5C%5C%5C%5C%5Cu0040domain.
	// Dạng unicode: user + %5C%5C%5C%5C%5C%5C%5C%5Cu0040 + domain.
	// user/domain là alphanumeric+dot+hyphen → KHÔNG cần url-encode thêm (để tránh double-encode %).
	var unicodeNew string
	if at := strings.LastIndex(email, "@"); at >= 0 {
		user := email[:at]
		domain := email[at+1:]
		unicodeNew = user + "%5C%5C%5C%5C%5C%5C%5C%5Cu0040" + domain
	}

	r := strings.NewReplacer(
		capEmail, enc,
		capEmailPlain, email,
		capEmailUnicode, unicodeNew, // thay capture email unicode → email mới unicode
	)
	return r.Replace(body)
}

// setEventReqID thay event_request_id (UUID mới mỗi submit).
func setEventReqID(body, id string) string {
	return strings.ReplaceAll(body, capEventReqID, id)
}

// setAAC thay aacjid + aaccs.
func setAAC(body, jid, ccs string) string {
	return strings.NewReplacer(capAacJID, jid, capAacCS, ccs).Replace(body)
}

// capCode — OTP gốc trong confirm template (capture).
const capCode = "479123"

// ─── Anchor cho password / birthday / name / username / create ──────────────

// capEncPwdPrefix — prefix encrypted_password trong template (field header trước giá trị).
const capEncPwdPrefix = `encrypted_password%5C%5C%5C%22%3A%5C%5C%5C%22`

// capEncPwdClose — chuỗi đóng field encrypted_password (field tiếp theo).
const capEncPwdClose = `%5C%5C%5C%22%2C%5C%5C%5C%22username`

// setEncryptedPassword thay encrypted_password bằng giá trị mã hóa mới.
func setEncryptedPassword(body, encPwd string) string {
	i := strings.Index(body, capEncPwdPrefix)
	if i < 0 {
		return body
	}
	start := i + len(capEncPwdPrefix)
	j := strings.Index(body[start:], capEncPwdClose)
	if j < 0 {
		return body
	}
	// url-encode encPwd (chứa #:/ cần encode).
	enc := url.QueryEscape(encPwd)
	// Dạng trong body: %5C%5C%5C%22 bao quanh → giữ prefix/suffix, chỉ thay value.
	return body[:start] + enc + body[start+j:]
}

// capBirthday / capBirthdayTS — birthday gốc trong capture.
const (
	capBirthday   = "28-03-2000"
	capBirthdayTS = "954235554"
)

// setBirthday thay ngày sinh (DD-MM-YYYY) + timestamp unix.
func setBirthday(body, ddmmyyyy string, ts int64) string {
	tsStr := fmt.Sprintf("%d", ts)
	return strings.NewReplacer(capBirthday, ddmmyyyy, capBirthdayTS, tsStr).Replace(body)
}

// capName — tên gốc trong capture.
const capName = "Koiu677"

// setName thay tên người dùng (full name, 1 field).
func setName(body, name string) string {
	return strings.ReplaceAll(body, capName, name)
}

// capUsername — username gốc trong capture (validation_text).
const capUsername = "quanvucong2k4"

// setUsername thay username (validation_text).
func setUsername(body, username string) string {
	return strings.ReplaceAll(body, capUsername, username)
}

// capRegContextCreate — prefix reg_context trong create.account template.
const capRegContextCreate = `reg_context%5C%5C%5C%22%3A%5C%5C%5C%22`

// setRegContextCreate thay reg_context trong create.account body.
// reg_context value kết thúc bằng |regm (url-encode → %7Cregm), anchor này duy nhất trong body.
func setRegContextCreate(body, regCtx string) string {
	i := strings.Index(body, capRegContextCreate)
	if i < 0 {
		return body
	}
	start := i + len(capRegContextCreate)
	seg := body[start:]
	// Tìm %7Cregm (= |regm url-encoded) — anchor cuối duy nhất của reg_context blob.
	endAnchor := "%7Cregm"
	j := strings.Index(seg, endAnchor)
	if j < 0 {
		// Fallback: thử dạng không encode
		endAnchor = "%7cregm"
		j = strings.Index(seg, endAnchor)
	}
	if j < 0 {
		return body
	}
	end := j + len(endAnchor) // bao gồm cả "|regm"
	// url-encode regCtx (chứa ký tự đặc biệt như | / +)
	enc := url.QueryEscape(regCtx) // | → %7C, regm giữ nguyên
	return body[:start] + enc + body[start+end:]
}

// capConfirmToken — confirmation_code token gốc trong capture (base64 8 chars).
const capConfirmToken = "zhveisAD"

// setConfirmationCode thay confirmation_code token trong create body.
func setConfirmationCode(body, token string) string {
	if token == "" {
		return body
	}
	return strings.ReplaceAll(body, capConfirmToken, token)
}

// setCode thay confirmation_code.
func setCode(body, code string) string {
	return strings.ReplaceAll(body, capCode, code)
}

// capRegContextConfirm — reg_context gốc trong confirm template (capture, prefix AVj0T0Bq...).
// Full value nằm giữa aclRegOpen và aclRegClose.
const (
	regCtxOpen  = `reg_context%5C%5C%5C%22%3A%5C%5C%5C%22`
	regCtxClose = `%5C%5C%5C%22`
)

// setRegContextRaw thay reg_context (đã encode sẵn dạng raw-url) trong body.
// regCtxEncoded là reg_context server cấp, đã được url-encode ĐÚNG lớp như trong body.
func setRegContextRaw(body, regCtxEncoded string) string {
	i := strings.Index(body, regCtxOpen)
	if i < 0 {
		return body
	}
	start := i + len(regCtxOpen)
	j := strings.Index(body[start:], regCtxClose)
	if j < 0 {
		return body
	}
	return body[:start] + regCtxEncoded + body[start+j:]
}

// aclCloseConfirm — anchor đóng accounts_list trong confirm template (field network_bssid).
const aclCloseConfirm = `%5D%2C%5C%5C%5C%22network_bssid`

// stripAccountsListConfirm strip accounts_list trong confirm body.
func stripAccountsListConfirm(body string) string {
	i := strings.Index(body, aclOpen)
	if i < 0 {
		return body
	}
	start := i + len(aclOpen)
	j := strings.Index(body[start:], aclCloseConfirm)
	if j < 0 {
		return body
	}
	return body[:start] + body[start+j:]
}

// Anchor strip accounts_list (string-index, KHÔNG regex để tránh phá nested escape).
// Mảng nằm giữa: accounts_list%5C%5C%5C%22%3A%5B  ...  %5D%2C%5C%5C%5C%22fb_ig_device_id
const (
	aclOpen  = `accounts_list%5C%5C%5C%22%3A%5B`
	aclClose = `%5D%2C%5C%5C%5C%22fb_ig_device_id`
)

// stripAccountsList thay nội dung mảng accounts_list bằng rỗng (giữ "[]"), an toàn
// theo anchor string. Account MỚI chưa đăng nhập gì → list rỗng là hợp lý; gửi token
// đăng nhập thật của máy capture từ device_id mới → server nghi ngờ.
func stripAccountsList(body string) string {
	i := strings.Index(body, aclOpen)
	if i < 0 {
		return body
	}
	start := i + len(aclOpen) // ngay sau '['
	j := strings.Index(body[start:], aclClose)
	if j < 0 {
		return body
	}
	// body[start : start+j] = nội dung mảng (các object) → xóa, để lại [] .
	return body[:start] + body[start+j:]
}

// injectRegContext — thay reg_context cũ trong body (nếu có) bằng giá trị server mới.
// submit_email template không chứa reg_context nên đây là no-op cho bước đó; dùng cho
// các step sau (confirmation/create.account) khi cần carry-over.
func injectRegContext(body, regCtx string) string {
	// reg_context trong body url-encoded nằm trong nhiều lớp; capture submit_email
	// không có. Với step có reg_context, thay theo prefix |regm blob đã encode.
	// Hiện chưa cần cho Milestone 1 → trả nguyên body.
	return body
}
