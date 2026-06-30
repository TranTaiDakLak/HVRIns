// Package igspc — Instagram Secondary Profile Creation (SPC).
//
// Tạo IG account CON từ 1 IG account CHA (parent) đã login (Bearer IGT:2:).
// Đây là luồng "Add account → Create new" trong Profile Switcher của IG Android.
//
// KHÁC engine reg thường (internal/igcore):
//   - Authenticated bằng Bearer IGT:2: của parent (không phải logged-out + x-mid).
//   - Tất cả qua /graphql_www Bloks (4 bước: get_sso → username → ac_optin → create).
//   - KHÔNG cần Play Integrity / Keystore attestation (server fallback cho client non-GMS).
//   - KHÔNG truyền password — con tạo ra passwordless (set sau nếu cần).
//
// Body Bloks ~400 field → CLONE VERBATIM từ capture V3 (go:embed templates/),
// chỉ replace 7 token động. Build minimal = server trả system_error.
//
// Verified 2026-06-29: 1 parent live đẻ 2-3 con (account_created=true) với -no-attest.
// Doc: docs/flows/ig-spc-secondary-account-creation.md
package igspc

import (
	"context"
	_ "embed"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

//go:embed templates/username.txt
var tplUsername string

//go:embed templates/ac_optin.txt
var tplAcOptin string

//go:embed templates/create.txt
var tplCreate string

// Parent — IG account cha đã login, dùng làm ngữ cảnh tạo con.
type Parent struct {
	Username string // @handle của parent (điền "source_username")
	UID      string // ds_user_id của parent
	Bearer   string // "IGT:2:<base64>" — từ result.BuildIGBearerToken(cookie)
}

// Options — cấu hình cho 1 lần tạo con.
type Options struct {
	Proxy     string // proxy string (vd "host:port:user:pass")
	Email     string // email tạm ĐÃ tạo sẵn (contactpoint của con) — caller cấp
	UserAgent string // optional; rỗng → dùng UA capture
	Birthday  string // optional; rỗng → "1992-02-28"
	Username  string // optional; rỗng → random
}

// Child — kết quả 1 con tạo ra.
type Child struct {
	Success   bool
	UID       string // pk (uid con)
	Sessionid string
	Mid       string
	Csrftoken string
	Cookie    string // "csrftoken=...;mid=...;ds_user_id=...;sessionid=..."
	Bearer    string // IGT:2: của con (dựng từ cookie)
	Username  string // username submit (con thực tế hiện "Instagram user")
	Email     string
	Nonce     string // partially_created_account_nonce (cho Phase 2 verify)
	PubKey    string // password encryption pub key (cho Phase 3 set password)
	ParentUID string
	Message   string
}

// CreateChild chạy SPC flow tạo 1 con từ parent. Luôn no-attest (production default).
// Trả Child{Success:true} nếu account_created=true; ngược lại Success=false + Message.
// error chỉ trả khi lỗi hạ tầng (session/network) — lỗi logic nằm trong Child.Message.
func CreateChild(ctx context.Context, parent Parent, opts Options) (Child, error) {
	child := Child{ParentUID: parent.UID, Email: opts.Email}

	if strings.TrimSpace(parent.Bearer) == "" || strings.TrimSpace(parent.UID) == "" {
		return child, fmt.Errorf("igspc: parent thiếu bearer/uid")
	}
	if strings.TrimSpace(opts.Email) == "" {
		return child, fmt.Errorf("igspc: thiếu email contactpoint")
	}

	birthday := opts.Birthday
	if birthday == "" {
		birthday = "1992-02-28"
	}
	username := opts.Username
	if username == "" {
		username = randomUsername()
	}
	child.Username = username

	sess, err := newSession(opts.Proxy)
	if err != nil {
		return child, fmt.Errorf("igspc: tạo session: %w", err)
	}
	defer sess.close()

	waterfallID := uuid.NewString()
	pigeonSID := newPigeonSession()
	connUUID := newConnUUID()

	// Step A — verify parent token còn live (200 = ok).
	if st := callGetSSO(ctx, sess, parent, pigeonSID, connUUID); !st.ok {
		child.Message = "Step A get_sso_accounts FAIL: " + st.preview
		return child, nil
	}

	// Step D — username.async (submit email + bday + username).
	if st := callBloks(ctx, sess, tplUsername, appIDUsername, parent, waterfallID, username, birthday, opts.Email); !st.ok {
		child.Message = "Step D username FAIL: " + st.preview
		return child, nil
	}

	// Step E — ac_optin.async (ToS / Account Center).
	if st := callBloks(ctx, sess, tplAcOptin, appIDAcOptin, parent, waterfallID, username, birthday, opts.Email); !st.ok {
		child.Message = "Step E ac_optin FAIL: " + st.preview
		return child, nil
	}

	// Step F — create.account.async ⭐ (tạo account thật).
	body, header, ferr := callCreate(ctx, sess, parent, waterfallID, username, birthday, opts.Email)
	if ferr != nil {
		child.Message = "Step F create network: " + ferr.Error()
		return child, ferr // lỗi mạng → trả error để caller retry
	}
	parseCreateResponse(body, header, &child)
	if !child.Success {
		if child.Message == "" {
			child.Message = "create không có account_created=true"
		}
		return child, nil
	}
	if child.Message == "" {
		child.Message = "account_created=true"
	}
	return child, nil
}

// ── random helpers ────────────────────────────────────────────────────────────

const usernameChars = "abcdefghijklmnopqrstuvwxyz0123456789"

func randomUsername() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = usernameChars[int(b[i])%len(usernameChars)]
	}
	// dạng "<word>.<6 ký tự>" — giống app thật, tránh collision
	return "user_" + string(b)
}

func newPigeonSession() string { return "UFS-" + uuid.NewString() + "-0" }

func newConnUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func nowTimestamp() string {
	return fmt.Sprintf("%.3f", float64(time.Now().UnixMilli())/1000.0)
}
