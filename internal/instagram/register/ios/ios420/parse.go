// parse.go — iOS555 create.account response parser.
//
// Response là Bloks bytecode (S-expression) bọc nhiều lớp JSON-escape. Khi thành
// công, nhúng sẵn full login payload (capture [164]):
//
//	{"session_key":..,"uid":..,"access_token":"EAAAAAY..","session_cookies":[
//	   {"name":"c_user","value":..},{"name":"xs",..},{"name":"fr",..},{"name":"datr",..}]}
//
// → parser strip backslash rồi regex bắt UID + token + 4 cookie.
package ios420

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// reCreatedUID — UID bọc trong opcode (eud <số>). Capture: "(eud 61590446695709)".
	reCreatedUID = regexp.MustCompile(`\(eud (\d{10,})\)`)
	// reEAAToken — access token iOS app thật: prefix EAAAAA (6 chữ A) + ~244 ký tự.
	// SIẾT prefix để không match nhầm trace ID/encoded blob ngẫu nhiên bắt đầu EAA.
	// Capture [164]: "EAAAAAYsX7TsBR..." total 250 ký tự.
	reEAAToken = regexp.MustCompile(`EAAAAA[A-Za-z0-9]{100,}`)
	// reErrorMsg — thông báo lỗi bloks: (dq8 "<scope>:error_message" "<msg>").
	// Anchor `(dq8 ` quan trọng: tránh bắt nhầm key list `"client_error_message" "<next_key>"`
	// (false positive khiến reg nosess bị classify thành fail, chặn round 2).
	reErrorMsg = regexp.MustCompile(`\(dq8 "[A-Z_]+:error_message" "([^"]{3,200})"\)`)

	// Cookie extractors — chạy trên body đã strip backslash.
	reCkCUser = regexp.MustCompile(`"name":"c_user","value":"(\d{8,})"`)
	reCkXS    = regexp.MustCompile(`"name":"xs","value":"([^"]+)"`)
	reCkFR    = regexp.MustCompile(`"name":"fr","value":"([^"]+)"`)
	reCkDATR  = regexp.MustCompile(`"name":"datr","value":"([^"]+)"`)

	// reSessionlessCUID — trích sessionless_crypted_user_id từ Bloks DSL confirmation params.
	// Trong nosess response, token này (~110 char, prefix Q9T_) đứng liền trước
	// sessionless_flow_info JSON (`{"flow_name":`) trong cùng dkc values list.
	// Capture [162] response: "Q9T_BQEx..." "{"flow_name"... — pattern ổn định.
	reSessionlessCUID = regexp.MustCompile(`"(Q9T_[A-Za-z0-9_-]{50,150})" "\{"flow_name"`)
)

// regOutcome — kết quả parse 1 response create.account.
type regOutcome struct {
	UID         string
	AccessToken string
	Cookie      string
	DATR        string
	Blocked     bool
	// Partial — token cho round 2 khi response là nosess (UID có, session chưa cấp).
	// Nil nếu response đã thành công (full) hoặc fail.
	Partial *partialTokens
}

// partialTokens — token FB cấp sau create.account round 1, dùng cho round 2.
// Trích từ Bloks DSL keys/values list trong response nosess.
type partialTokens struct {
	PartiallyCreated string // fb_partially_created_reg_info
	EncryptedProps   string // fb_encrypted_partial_new_account_properties
	RegContext       string // reg_context
	Srnonce          string // srnonce
	SessionlessCUID  string // sessionless_crypted_user_id (~110 char Q9T_ blob)
}

// Anchor: keys list của state block chứa partial tokens.
// FB Bloks DSL: (dkc "k0" "k1" ... "kN") (dkc v0 v1 ... vN) — keys + values song song.
// Vị trí trong values list (sau khi tính ALIGN):
//
//	idx 2  → fb_encrypted_partial_new_account_properties (Q9T_ blob)
//	idx 3  → fb_partially_created_reg_info  (Q8CKBQ blob ~8KB)
//	idx 24 → reg_context                    (AV... blob ~10KB)
//	idx 25 → srnonce                        (64-char alphanumeric)
//
// LƯU Ý OFFSET +4: flow_info ở idx-key 5 là chuỗi JSON `{"flow_name":...,"flow_type":...}`,
// tokenizer tách thành 5 tokens ({ : , : }) thay vì 1 → mọi idx-key sau 5 bị +4 trong
// values list.
// Keys list hiện tại (25 keys, verified 2026-05-28):
//
//	k0  should_ignore_suma_check                      → v[0]
//	k1  should_ignore_existing_login                  → v[1]
//	k2  fb_encrypted_partial_new_account_properties   → v[2]
//	k3  fb_partially_created_reg_info                 → v[3]
//	k4  bloks_controller_source                       → v[4]
//	k5  flow_info (JSON → 5 tokens)                   → v[5..9]
//	k6  current_step                                  → v[10]
//	...
//	k20 reg_context                                   → v[24]  (+4 từ flow_info)
//	k21 srnonce                                       → v[25]  (+4 từ flow_info)
//	k22 sessionless_crypted_user_id                   → v[26]
const (
	partialKeysAnchor = `"should_ignore_suma_check" "should_ignore_existing_login" "fb_encrypted_partial_new_account_properties" "fb_partially_created_reg_info"`
	idxEncryptedProps = 2
	idxPartialCreated = 3
	idxRegContext     = 24
	idxSrnonce        = 25
)

// extractPartialTokens bóc 3 token round-2 từ response nosess.
// Trả nil nếu không tìm thấy keys block hoặc không parse được đủ values.
func extractPartialTokens(body string) *partialTokens {
	clean := strings.ReplaceAll(body, "\\", "")
	keysIdx := strings.Index(clean, partialKeysAnchor)
	if keysIdx < 0 {
		return nil
	}
	// Tìm cuối keys list + đầu values list: `) (dkc `
	rel := strings.Index(clean[keysIdx:], `) (dkc `)
	if rel < 0 {
		return nil
	}
	valuesStart := keysIdx + rel + len(`) (dkc `)
	values := tokenizeBloksValues(clean[valuesStart:], 28)
	if len(values) <= idxSrnonce {
		return nil
	}
	t := &partialTokens{
		EncryptedProps:   values[idxEncryptedProps],
		PartiallyCreated: values[idxPartialCreated],
		RegContext:       values[idxRegContext],
		Srnonce:          values[idxSrnonce],
	}
	// Sanity check — tokens phải là blob thật, không phải "true"/"false"/null.
	// Khi tokenizer lệch idx (vd flow_info JSON tách khác), giá trị có thể là bool.
	if len(t.PartiallyCreated) < 100 || len(t.Srnonce) < 30 {
		return nil
	}
	if t.Srnonce == "true" || t.Srnonce == "false" {
		return nil
	}
	// sessionless_crypted_user_id: Q9T_ blob (~110 char) đứng trước sessionless_flow_info JSON.
	// Nằm trong confirmation-action params block, tách biệt với partialKeys block.
	if m := reSessionlessCUID.FindStringSubmatch(clean); len(m) > 1 {
		t.SessionlessCUID = m[1]
	}
	return t
}

// tokenizeBloksValues đọc danh sách value của 1 (dkc ...) trong Bloks DSL.
// Dừng khi gặp `)` ở depth 0 hoặc đủ maxTokens. Trả slice value đã unquote.
// Hỗ trợ: string "..." , true , false , null (→ ""), number, nested expr (skip).
func tokenizeBloksValues(s string, maxTokens int) []string {
	out := make([]string, 0, maxTokens)
	i := 0
	depth := 0
	for i < len(s) && len(out) < maxTokens {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n') {
			i++
		}
		if i >= len(s) {
			break
		}
		c := s[i]
		switch {
		case c == ')':
			if depth == 0 {
				return out
			}
			depth--
			i++
		case c == '(':
			// Nested expr → coi như placeholder "", skip đến cuối expr.
			startDepth := depth
			depth++
			i++
			for i < len(s) && depth > startDepth {
				if s[i] == '(' {
					depth++
				} else if s[i] == ')' {
					depth--
				} else if s[i] == '"' {
					// skip string inside
					j := strings.IndexByte(s[i+1:], '"')
					if j < 0 {
						return out
					}
					i = i + 1 + j
				}
				i++
			}
			if depth == startDepth {
				out = append(out, "")
			}
		case depth > 0:
			i++
		case c == '"':
			end := strings.IndexByte(s[i+1:], '"')
			if end < 0 {
				return out
			}
			out = append(out, s[i+1:i+1+end])
			i = i + 1 + end + 1
		case c == 't' && i+4 <= len(s) && s[i:i+4] == "true":
			out = append(out, "true")
			i += 4
		case c == 'f' && i+5 <= len(s) && s[i:i+5] == "false":
			out = append(out, "false")
			i += 5
		case c == 'n' && i+4 <= len(s) && s[i:i+4] == "null":
			out = append(out, "")
			i += 4
		case (c >= '0' && c <= '9') || c == '-':
			j := i + 1
			for j < len(s) && ((s[j] >= '0' && s[j] <= '9') || s[j] == '.') {
				j++
			}
			out = append(out, s[i:j])
			i = j
		default:
			i++
		}
	}
	return out
}

// parseCreateAccountResponse phân tích body response create.account.
// Trả lỗi nếu reg không thành công (blocked / không có UID / lỗi rõ ràng).
// ParseCreateAccountResponse exported alias — dùng bởi ios563.
func ParseCreateAccountResponse(body string) (*regOutcome, error) {
	return parseCreateAccountResponse(body)
}

func parseCreateAccountResponse(body string) (*regOutcome, error) {
	out := &regOutcome{}
	// clean: bỏ mọi backslash để regex xuyên qua các lớp JSON-escape.
	// An toàn vì giá trị UID/token/cookie của FB không chứa '\'.
	clean := strings.ReplaceAll(body, "\\", "")
	low := strings.ToLower(clean)

	// ── Blocked / từ chối tạo ──────────────────────────────────────────────
	if strings.Contains(low, "integrity_block") {
		out.Blocked = true
		return out, fmt.Errorf("Facebook blocked: integrity_block")
	}
	if strings.Contains(low, "couldn't create an account") ||
		strings.Contains(low, "account creation denied") {
		out.Blocked = true
		return out, fmt.Errorf("Facebook blocked: account creation denied")
	}

	// ── UID + token + cookie ───────────────────────────────────────────────
	if m := reCreatedUID.FindStringSubmatch(clean); len(m) > 1 {
		out.UID = m[1]
	}
	if m := reEAAToken.FindString(clean); m != "" {
		out.AccessToken = m
	}

	cUser := firstGroup(reCkCUser, clean)
	xs := firstGroup(reCkXS, clean)
	fr := firstGroup(reCkFR, clean)
	datr := firstGroup(reCkDATR, clean)
	out.DATR = datr
	// UID fallback từ cookie c_user nếu (eud ...) không bắt được.
	if out.UID == "" {
		out.UID = cUser
	}
	out.Cookie = composeCookie(cUser, xs, fr, datr)

	// Full success: có UID + session thật (cookie hoặc token). Exit ngay.
	// KHÔNG dùng string "create_success" làm điều kiện vì chuỗi này có thể
	// xuất hiện trong Bloks bytecode nosess → khiến exit sớm, bỏ qua round 2.
	if out.UID != "" && (out.Cookie != "" || out.AccessToken != "") {
		return out, nil
	}

	// Nosess: UID có nhưng FB chưa cấp session → extract partial tokens cho round 2.
	if out.UID != "" {
		out.Partial = extractPartialTokens(body)
	}

	// ── Lỗi rõ ràng từ bloks error_message ─────────────────────────────────
	if m := reErrorMsg.FindStringSubmatch(clean); len(m) > 1 {
		return out, fmt.Errorf("Register failed: %s", strings.TrimSpace(m[1]))
	}
	if strings.Contains(low, "checkpoint") {
		return out, fmt.Errorf("Register failed: Checkpoint")
	}
	if out.UID != "" {
		return out, nil
	}
	return out, fmt.Errorf("Register failed: no UID in response")
}

// composeCookie ghép cookie string chuẩn (c_user;xs;fr;datr). Bỏ field rỗng.
func composeCookie(cUser, xs, fr, datr string) string {
	var parts []string
	if datr != "" {
		parts = append(parts, "datr="+datr)
	}
	if cUser != "" {
		parts = append(parts, "c_user="+cUser)
	}
	if xs != "" {
		parts = append(parts, "xs="+xs)
	}
	if fr != "" {
		parts = append(parts, "fr="+fr)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ";") + ";"
}

// firstGroup trả group 1 của match đầu tiên, "" nếu không khớp.
func firstGroup(re *regexp.Regexp, s string) string {
	if m := re.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}
	return ""
}
