package igspc

import (
	"encoding/base64"
	"net/url"
	"regexp"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/google/uuid"
)

// ── Token capture (giá trị trong body V3, session kenneth_roberts2001) ───────────
//
// Body Bloks ~400 field clone verbatim từ templates/, chỉ replace các token động này.

const (
	captureEmail          = "frost.ninja7158@17a.imgui.de"
	captureUsername2      = "platypus.1114188"
	captureBirthday       = "1992-02-28"
	captureParentUsername = "kenneth_roberts2001"
	captureWaterfallID    = "f82bedd4-d5fd-45c8-be11-5e3e4e679f42"
	captureEventReqID     = "0bc947a8-9075-445f-b829-a4394a3815a3" // có trong create.txt
	captureEventReqID2    = "eb1e8ad3-94fc-4d96-bedc-1f990b4c6d89" // có trong username/ac_optin.txt
	// Bearer cũ trong capture (đã rotated) — URL-encoded trong body.
	captureBearerB64 = "IGT%3A2%3AeyJkc191c2VyX2lkIjoiMTc2OTc3NDgwOTkiLCJzZXNzaW9uaWQiOiIxNzY5Nzc0ODA5OSUzQXBZTlBKbUFqbk1zelVtJTNBMjUlM0FBWWlaVkhtU3RKVUJXelBQUGVhY3NKQ2tRbjlKNVNzM1gwYkRIQm1KQkEifQ%3D%3D"
)

// replaceBodyTokens clone body capture verbatim, thay token động (KHÔNG re-escape JSON).
func replaceBodyTokens(tpl string, parent Parent, waterfallID, username, birthday, email string) string {
	// Bearer parent: "IGT:2:<b64>" (đã bỏ "Bearer " nếu có) → URL-encode để khớp body.
	parentBearerB64 := strings.TrimPrefix(strings.TrimSpace(parent.Bearer), "Bearer ")
	parentBearerB64Enc := url.QueryEscape(parentBearerB64)

	emailEnc := url.QueryEscape(email)
	captureEmailEnc := url.QueryEscape(captureEmail)
	newEventReqID := uuid.NewString()

	s := tpl
	// Thứ tự: token dài/đặc thù trước để tránh collision.
	s = strings.ReplaceAll(s, captureBearerB64, parentBearerB64Enc)
	s = strings.ReplaceAll(s, captureEmailEnc, emailEnc)
	s = strings.ReplaceAll(s, captureEmail, email) // phòng chỗ chưa URL-encode
	s = strings.ReplaceAll(s, captureUsername2, username)
	s = strings.ReplaceAll(s, captureParentUsername, parent.Username)
	s = strings.ReplaceAll(s, captureWaterfallID, waterfallID)
	s = strings.ReplaceAll(s, captureEventReqID, newEventReqID)
	s = strings.ReplaceAll(s, captureEventReqID2, newEventReqID)
	// Birthday cuối (chuỗi ngắn dễ trùng) — chỉ thay nếu khác mặc định.
	if birthday != "" && birthday != captureBirthday {
		s = strings.ReplaceAll(s, captureBirthday, birthday)
	}
	return s
}

// ── Parse create response → child creds ─────────────────────────────────────────

var (
	// Create response = Bloks bundle escape nhiều tầng, NHÚNG các dòng "Set-Cookie:"
	// trong body (KHÔNG phải HTTP header). Sau khi bỏ backslash + chuẩn hoá u0025→%,
	// cookie con xuất hiện dạng "name=value; Domain=...".
	reAccountCreated = regexp.MustCompile(`account_created"\s*:\s*true`)
	reCkSession      = regexp.MustCompile(`sessionid=([^;]+)`)
	reCkCsrf         = regexp.MustCompile(`csrftoken=([^;]+)`)
	reCkDsUser       = regexp.MustCompile(`ds_user_id=([^;]+)`)
	reNonce          = regexp.MustCompile(`partially_created_account_nonce"\s*:\s*"([^"]+)`)
)

// parseCreateResponse trích uid/sessionid/csrf/cookie/bearer của CON từ response
// create.account.async. Nguồn dữ liệu = BODY (Set-Cookie nhúng), KHÔNG phải HTTP header
// (response chỉ có header generic — đã verify thực tế).
func parseCreateResponse(body string, _ fhttp.Header, child *Child) {
	// Bỏ backslash (escape nhiều tầng) + u0025→% (% mất backslash sau flatten).
	flat := strings.ReplaceAll(body, `\`, "")
	flat = strings.ReplaceAll(flat, "u0025", "%")

	if !reAccountCreated.MatchString(flat) {
		child.Success = false
		switch {
		case strings.Contains(flat, "bouncing_cliff"):
			child.Message = "server bounce (anti-abuse)"
		case strings.Contains(flat, "system_error"):
			child.Message = "create_failure: system_error (body shape?)"
		case strings.Contains(flat, "something went wrong"):
			child.Message = "generic error (rate-limit/reputation?)"
		}
		return
	}
	child.Success = true

	if m := reCkSession.FindStringSubmatch(flat); len(m) == 2 {
		child.Sessionid = m[1]
	}
	if m := reCkCsrf.FindStringSubmatch(flat); len(m) == 2 {
		child.Csrftoken = m[1]
	}
	if m := reNonce.FindStringSubmatch(flat); len(m) == 2 {
		child.Nonce = m[1]
	}
	// uid: ưu tiên prefix của sessionid (chắc chắn), fallback ds_user_id cookie.
	child.UID = uidFromSessionid(child.Sessionid)
	if child.UID == "" {
		if m := reCkDsUser.FindStringSubmatch(flat); len(m) == 2 {
			child.UID = m[1]
		}
	}

	child.Cookie = buildChildCookie(child)
	child.Bearer = buildBearerFromCookie(child.UID, child.Sessionid)
}

// uidFromSessionid lấy uid = phần trước "%3A" (hoặc ':') đầu tiên của sessionid.
func uidFromSessionid(sid string) string {
	for _, sep := range []string{"%3A", "%3a", ":"} {
		if i := strings.Index(sid, sep); i > 0 {
			return sid[:i]
		}
	}
	return ""
}

// buildChildCookie ghép cookie "csrftoken=...;mid=...;ds_user_id=...;sessionid=..."
// (chỉ phần có giá trị). Đủ cho CheckLiveByCookie (cần sessionid+ds_user_id, csrf optional).
func buildChildCookie(c *Child) string {
	var parts []string
	if c.Csrftoken != "" {
		parts = append(parts, "csrftoken="+c.Csrftoken)
	}
	if c.Mid != "" {
		parts = append(parts, "mid="+c.Mid)
	}
	if c.UID != "" {
		parts = append(parts, "ds_user_id="+c.UID)
	}
	if c.Sessionid != "" {
		parts = append(parts, "sessionid="+c.Sessionid)
	}
	return strings.Join(parts, ";")
}

// buildBearerFromCookie dựng "IGT:2:" + base64 — KHỚP result.BuildIGBearerToken.
func buildBearerFromCookie(uid, sessionid string) string {
	if uid == "" || sessionid == "" {
		return ""
	}
	payload := `{"ds_user_id":"` + uid + `","sessionid":"` + sessionid + `","should_use_header_over_cookies":true}`
	return "IGT:2:" + base64.StdEncoding.EncodeToString([]byte(payload))
}

