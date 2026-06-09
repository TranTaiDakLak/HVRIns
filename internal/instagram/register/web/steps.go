// steps.go — Orchestrator cho 8 bước đăng ký Facebook
// Quy trình: FetchRegTokens → B1 → B2 → B3 → B4 → B5 → B6 → B7 → B8
package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	cookiestore "HVRIns/internal/cookie"
	"HVRIns/internal/instagram/register/android"
	"HVRIns/internal/proxy"

	"github.com/google/uuid"
)

// SharedPool là partitioned datr pool dùng chung — set từ app.go trước khi chạy reg.
var SharedPool *android.PartitionedDatrPool

// RegisterAccount thực hiện toàn bộ luồng đăng ký Facebook qua 8 bước Bloks.
//
// Tham số:
//   - ctx: context để hủy hoặc đặt timeout cho toàn bộ luồng. Mỗi bước kiểm tra
//     ctx.Err() trước khi tiếp tục; hủy ctx sẽ dừng luồng tại bước tiếp theo.
//   - input: con trỏ tới RegInput chứa toàn bộ thông tin đăng ký:
//   - FirstName, LastName: tên hiển thị trên Facebook.
//   - Birthday: ngày sinh định dạng "DD-MM-YYYY".
//   - Gender: 1 = nữ, 2 = nam.
//   - Phone: số điện thoại (VN "0xxx" hoặc quốc tế "+xxx").
//   - Password: mật khẩu plaintext (sẽ được mã hóa tại B6).
//   - Proxy: chuỗi "host:port:user:pass" hoặc rỗng để dùng IP thật.
//   - UserAgent: UA tùy chỉnh; rỗng → dùng defaultRegUA.
//   - DebugDir: thư mục lưu response từng bước; rỗng → không lưu.
//   - TutDatr: nếu khác rỗng, ghi đè datr cookie sau Init (TUT mode).
//   - onStatus: callback nhận thông báo trạng thái từng bước (có thể nil).
//
// Quy trình 8 bước:
//  1. Init (FetchRegTokens): GET r.php → lấy fb_dtsg, lsd, datr, public key.
//  2. B1: khởi tạo session đăng ký trên server Facebook.
//  3. B2: gửi first_name + last_name.
//  4. B3: gửi ngày sinh (birthday).
//  5. B4: gửi giới tính (gender).
//  6. B5: gửi số điện thoại (phone).
//  7. B6: mã hóa password bằng public key rồi gửi lên.
//  8. B7: xác nhận điều khoản sử dụng (TOS).
//  9. B8: submit form cuối → tạo tài khoản, nhận UID + cookie.
//
// Giữa các bước chính còn có các "Screen" request (Screen-Birthday,
// Screen-Gender, Screen-Phone, Screen-Password, Screen-SaveCreds) để
// mô phỏng hành vi ứng dụng native tải màn hình trước khi submit dữ liệu.
//
// TUT mode: nếu input.TutDatr != "", datr được ghi đè ngay sau Init và giữ
// nguyên xuyên suốt B1→B8. Dùng để tái sử dụng cookie từ pool TUT account.
//
// Hàm trả về *RegResult với Success=true và UID nếu tạo được tài khoản,
// Success=true và UID="" nếu cần xác nhận SĐT, hoặc Success=false kèm Message
// mô tả lỗi tại bước thất bại.
func RegisterAccount(ctx context.Context, input *RegInput, onStatus StatusCallback) *RegResult {
	// Wrapper: track outcome per-datr sau khi reg xong (wrap mọi return).
	var pickedMachineID string
	result := doRegisterAccount(ctx, input, onStatus, &pickedMachineID)
	if pickedMachineID != "" && SharedPool != nil {
		outcome := "unknown"
		if result != nil {
			if result.Success {
				outcome = "success"
			} else if strings.Contains(strings.ToLower(result.Message), "checkpoint") {
				outcome = "checkpoint"
			} else if strings.Contains(result.Message, "Blocked") {
				outcome = "fail"
			}
		}
		SharedPool.RecordResult(pickedMachineID, outcome)
		if (outcome == "success" || outcome == "fail") && onStatus != nil {
			s, f, u, used := SharedPool.GetStats(pickedMachineID)
			trunc := pickedMachineID
			if len(trunc) > 10 {
				trunc = trunc[:10]
			}
			onStatus(fmt.Sprintf("[Pool] Datr %s... → %s (used %d | S/F/U: %d/%d/%d)",
				trunc, outcome, used, s, f, u))
		}
	}
	return result
}

func doRegisterAccount(ctx context.Context, input *RegInput, onStatus StatusCallback, pickedMachineIDOut *string) *RegResult {
	notify := func(msg string) {
		if onStatus != nil {
			onStatus(msg)
		}
	}

	s := &RegSession{
		Proxy:     input.Proxy,
		UserAgent: input.UserAgent,
		DebugDir:  input.DebugDir,
		Input:     input,
	}
	if s.UserAgent == "" {
		s.UserAgent = defaultRegUA
	}
	// Khởi tạo reg_info với initial values ngay từ đầu
	s.RegInfo = buildInitialRegInfo()

	// Init: lấy tokens từ trang đăng ký
	notify("[Init] Đang lấy tokens từ Facebook...")
	if err := FetchRegTokens(ctx, s); err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[Init] Lỗi kết nối: %v", err)}
	}
	if s.FbDtsg == "" {
		return &RegResult{Success: false, Message: "[Init] FAIL — không lấy được fb_dtsg"}
	}
	notify("[Init] Lấy tokens thành công")

	// TUT mode: ghi đè datr từ pool ngay sau init — dùng xuyên suốt B1→B8
	if input.TutDatr != "" {
		s.Datr = normalizeInitialDatr(input.TutDatr)
		if SharedPool != nil {
			sc, f, u, used := SharedPool.GetStats(s.Datr)
			trunc := s.Datr
			if len(trunc) > 10 {
				trunc = trunc[:10]
			}
			notify(fmt.Sprintf("[Init-TUT] Dùng datr %s... (used %d | S/F/U: %d/%d/%d)",
				trunc, used, sc, f, u))
		} else {
			notify("[Init-TUT] Dùng datr từ pool")
		}
	} else if SharedPool != nil {
		// Ưu tiên lấy datr từ pool nếu không có TutDatr input
		if poolDatr := SharedPool.GetNext(input.SlotIdx); poolDatr != "" {
			s.Datr = poolDatr
			sc, f, u, used := SharedPool.GetStats(poolDatr)
			notify(fmt.Sprintf("[Init] New initial %s (used %d | S/F/U: %d/%d/%d)",
				poolDatr, used, sc, f, u))
		}
	}
	// Track machine ID + increment usage sau khi reg xong
	if s.Datr != "" {
		if pickedMachineIDOut != nil {
			*pickedMachineIDOut = s.Datr
		}
		if SharedPool != nil {
			defer SharedPool.IncrementUsage(s.Datr)
		}
	}

	s.WaterfallID = uuid.New().String()

	// ── B1 ─────────────────────────────────────────────────────────────
	datrInfo := ""
	if s.Datr != "" && SharedPool != nil {
		sc, f, u, used := SharedPool.GetStats(s.Datr)
		trunc := s.Datr
		if len(trunc) > 10 {
			trunc = trunc[:10]
		}
		datrInfo = fmt.Sprintf(" | datr=%s... used=%d S/F/U=%d/%d/%d", trunc, used, sc, f, u)
	}
	notify(fmt.Sprintf("[B1] Khởi tạo session đăng ký%s...", datrInfo))
	b1Resp, b1Status, err := doRegPostRetry(ctx, s, endpointB1, buildB1Body(s))
	if err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B1] Lỗi kết nối: %v", err)}
	}
	if b1Status >= 400 {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B1] Lỗi HTTP %d (kiểm tra proxy hoặc thử lại)", b1Status)}
	}
	if errMsg := bloksError(b1Resp); errMsg != "" {
		return &RegResult{Success: false, Message: "[B1] Bloks error: " + errMsg}
	}
	updateSession(s, b1Resp)
	notify("[B1] Khởi tạo session OK")

	if ctx.Err() != nil {
		return &RegResult{Success: false, Message: "Đã dừng"}
	}

	// ── B2 ─────────────────────────────────────────────────────────────
	notify(fmt.Sprintf("[B2] Gửi tên: %s %s", input.FirstName, input.LastName))
	b2Resp, b2Status, err := doRegPostRetry(ctx, s, endpointB2, buildB2Body(s, input, uuid.New().String()))
	if err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B2] Lỗi kết nối: %v", err)}
	}
	debugSave(s, "B2", b2Resp)
	if b2Status >= 400 {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B2] Lỗi HTTP %d (kiểm tra proxy hoặc thử lại)", b2Status)}
	}
	if errMsg := bloksError(b2Resp); errMsg != "" {
		return &RegResult{Success: false, Message: "[B2] Bloks error: " + errMsg}
	}
	updateSession(s, b2Resp)
	updateRegInfoFields(s, map[string]interface{}{
		"first_name": input.FirstName,
		"last_name":  input.LastName,
		"full_name":  input.FirstName + " " + input.LastName,
	})
	notify(fmt.Sprintf("[B2] Đã gửi tên: %s %s", input.FirstName, input.LastName))

	if ctx.Err() != nil {
		return &RegResult{Success: false, Message: "Đã dừng"}
	}

	// ── Screen Birthday (app fetch) ─────────────────────────────────────
	notify("[Screen] Tải màn hình ngày sinh...")
	sbResp, sbStatus, err := doRegPostRetry(ctx, s, endpointScreenBirthday, buildScreenBirthdayBody(s))
	if err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[Screen-Birthday] Lỗi kết nối: %v", err)}
	}
	if sbStatus >= 400 {
		return &RegResult{Success: false, Message: fmt.Sprintf("[Screen-Birthday] Lỗi HTTP %d (kiểm tra proxy hoặc thử lại)", sbStatus)}
	}
	notify("[Screen-Birthday] Tải màn hình ngày sinh OK")
	updateSession(s, sbResp)
	addScreenVisited(s, "bloks.caa.reg.birthday")

	if ctx.Err() != nil {
		return &RegResult{Success: false, Message: "Đã dừng"}
	}

	// ── B3 ─────────────────────────────────────────────────────────────
	notify(fmt.Sprintf("[B3] Gửi ngày sinh: %s", input.Birthday))
	b3Resp, b3Status, err := doRegPostRetry(ctx, s, endpointB3, buildB3Body(s, input))
	if err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B3] Lỗi kết nối: %v", err)}
	}
	debugSave(s, "B3", b3Resp)
	if b3Status >= 400 {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B3] Lỗi HTTP %d (kiểm tra proxy hoặc thử lại)", b3Status)}
	}
	if errMsg := bloksError(b3Resp); errMsg != "" {
		return &RegResult{Success: false, Message: "[B3] Bloks error: " + errMsg}
	}
	updateSession(s, b3Resp)
	updateRegInfoFields(s, map[string]interface{}{
		"birthday":    input.Birthday,
		"age_range":   "o18",
		"did_use_age": false,
	})
	notify(fmt.Sprintf("[B3] Đã gửi ngày sinh: %s", input.Birthday))

	if ctx.Err() != nil {
		return &RegResult{Success: false, Message: "Đã dừng"}
	}

	// ── Screen Gender (app fetch) ───────────────────────────────────────
	notify("[Screen] Tải màn hình giới tính...")
	sgResp, sgStatus, err := doRegPostRetry(ctx, s, endpointScreenGender, buildScreenGenderBody(s))
	if err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[Screen-Gender] Lỗi kết nối: %v", err)}
	}
	if sgStatus >= 400 {
		return &RegResult{Success: false, Message: fmt.Sprintf("[Screen-Gender] Lỗi HTTP %d (kiểm tra proxy hoặc thử lại)", sgStatus)}
	}
	notify("[Screen-Gender] Tải màn hình giới tính OK")
	updateSession(s, sgResp)
	addScreenVisited(s, "bloks.caa.reg.gender")

	if ctx.Err() != nil {
		return &RegResult{Success: false, Message: "Đã dừng"}
	}

	// ── B4 ─────────────────────────────────────────────────────────────
	genderLabel := map[int]string{1: "Female", 2: "Male"}[input.Gender]
	notify(fmt.Sprintf("[B4] Gửi giới tính: %s", genderLabel))
	b4Resp, b4Status, err := doRegPostRetry(ctx, s, endpointB4, buildB4Body(s, input))
	if err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B4] Lỗi kết nối: %v", err)}
	}
	debugSave(s, "B4", b4Resp)
	if b4Status >= 400 {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B4] Lỗi HTTP %d (kiểm tra proxy hoặc thử lại)", b4Status)}
	}
	if errMsg := bloksError(b4Resp); errMsg != "" {
		return &RegResult{Success: false, Message: "[B4] Bloks error: " + errMsg}
	}
	updateSession(s, b4Resp)
	updateRegInfoFields(s, map[string]interface{}{
		"gender":            input.Gender,
		"use_custom_gender": false,
		"custom_gender":     nil,
	})
	notify(fmt.Sprintf("[B4] Đã gửi giới tính: %s", genderLabel))

	if ctx.Err() != nil {
		return &RegResult{Success: false, Message: "Đã dừng"}
	}

	// ── Screen Phone (app fetch) ────────────────────────────────────────
	notify("[Screen] Tải màn hình số điện thoại...")
	spResp, spStatus, err := doRegPostRetry(ctx, s, endpointScreenPhone, buildScreenPhoneBody(s))
	if err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[Screen-Phone] Lỗi kết nối: %v", err)}
	}
	if spStatus >= 400 {
		return &RegResult{Success: false, Message: fmt.Sprintf("[Screen-Phone] Lỗi HTTP %d (kiểm tra proxy hoặc thử lại)", spStatus)}
	}
	notify("[Screen-Phone] Tải màn hình số điện thoại OK")
	updateSession(s, spResp)
	addScreenVisited(s, "CAA_REG_CONTACT_POINT_PHONE")

	if ctx.Err() != nil {
		return &RegResult{Success: false, Message: "Đã dừng"}
	}

	// ── B5 ─────────────────────────────────────────────────────────────
	notify(fmt.Sprintf("[B5] Gửi số điện thoại: %s", input.Phone))
	b5Resp, b5Status, err := doRegPostRetry(ctx, s, endpointB5, buildB5Body(s, input, uuid.New().String()))
	if err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B5] Lỗi kết nối: %v", err)}
	}
	debugSave(s, "B5", b5Resp)
	if b5Status >= 400 {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B5] Lỗi HTTP %d (kiểm tra proxy hoặc thử lại)", b5Status)}
	}
	if errMsg := bloksError(b5Resp); errMsg != "" {
		return &RegResult{Success: false, Message: "[B5] Bloks error: " + errMsg}
	}
	updateSession(s, b5Resp)
	updateRegInfoFields(s, map[string]interface{}{
		"contactpoint":      toInternationalPhone(input.Phone),
		"contactpoint_type": "phone",
		"is_cp_claimed":     true,
	})
	notify(fmt.Sprintf("[B5] Đã gửi số điện thoại: %s", input.Phone))

	if ctx.Err() != nil {
		return &RegResult{Success: false, Message: "Đã dừng"}
	}

	// ── Screen Password (app fetch) ─────────────────────────────────────
	notify("[Screen] Tải màn hình mật khẩu...")
	swResp, swStatus, err := doRegPostRetry(ctx, s, endpointScreenPassword, buildScreenPasswordBody(s))
	if err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[Screen-Password] Lỗi kết nối: %v", err)}
	}
	if swStatus >= 400 {
		return &RegResult{Success: false, Message: fmt.Sprintf("[Screen-Password] Lỗi HTTP %d (kiểm tra proxy hoặc thử lại)", swStatus)}
	}
	notify("[Screen-Password] Tải màn hình mật khẩu OK")
	updateSession(s, swResp)
	addScreenVisited(s, "CAA_REG_PASSWORD")

	if ctx.Err() != nil {
		return &RegResult{Success: false, Message: "Đã dừng"}
	}

	// ── B6 ─────────────────────────────────────────────────────────────
	notify("[B6] Mã hóa và gửi mật khẩu...")
	if s.PubKeyHex == "" {
		return &RegResult{Success: false, Message: "[B6] FAIL — không có public key để encrypt password"}
	}
	encPwd, err := GenerateEncPassword(input.Password, s.PubKeyHex, s.PubKeyID, s.PubKeyVer)
	if err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B6] Lỗi encrypt: %v", err)}
	}
	b6Resp, b6Status, err := doRegPostRetry(ctx, s, endpointB6, buildB6Body(s, encPwd, uuid.New().String()))
	if err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B6] Lỗi kết nối: %v", err)}
	}
	debugSave(s, "B6", b6Resp)
	if b6Status >= 400 {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B6] Lỗi HTTP %d (kiểm tra proxy hoặc thử lại)", b6Status)}
	}
	if errMsg := bloksError(b6Resp); errMsg != "" {
		return &RegResult{Success: false, Message: "[B6] Bloks error: " + errMsg}
	}
	updateSession(s, b6Resp)
	updateRegInfoFields(s, map[string]interface{}{
		"encrypted_password": encPwd,
		"headers_flow_id":    uuid.New().String(),
	})
	notify("[B6] Đã gửi mật khẩu (đã mã hóa)")

	if ctx.Err() != nil {
		return &RegResult{Success: false, Message: "Đã dừng"}
	}

	// ── Screen Save-Credentials (app fetch) ─────────────────────────────
	notify("[Screen] Tải màn hình lưu thông tin đăng nhập...")
	scResp, scStatus, err := doRegPostRetry(ctx, s, endpointScreenSaveCreds, buildScreenSaveCredsBody(s))
	if err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[Screen-SaveCreds] Lỗi kết nối: %v", err)}
	}
	if scStatus >= 400 {
		return &RegResult{Success: false, Message: fmt.Sprintf("[Screen-SaveCreds] Lỗi HTTP %d (kiểm tra proxy hoặc thử lại)", scStatus)}
	}
	notify("[Screen-SaveCreds] OK")
	updateSession(s, scResp)
	addScreenVisited(s, "CAA_REG_SAVE_PASSWORD_CREDENTIALS")

	if ctx.Err() != nil {
		return &RegResult{Success: false, Message: "Đã dừng"}
	}

	// ── B7 ─────────────────────────────────────────────────────────────
	notify("[B7] Xác nhận điều khoản sử dụng...")
	b7Resp, b7Status, err := doRegPostRetry(ctx, s, endpointB7, buildB7Body(s))
	if err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B7] Lỗi kết nối: %v", err)}
	}
	debugSave(s, "B7", b7Resp)
	if b7Status >= 400 {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B7] Lỗi HTTP %d (kiểm tra proxy hoặc thử lại)", b7Status)}
	}
	if errMsg := bloksError(b7Resp); errMsg != "" {
		return &RegResult{Success: false, Message: "[B7] Bloks error: " + errMsg}
	}
	updateSession(s, b7Resp)
	updateRegInfoFields(s, map[string]interface{}{
		"should_save_password":                 true,
		"existing_account_exact_match_checked": true,
	})
	notify("[B7] Đã xác nhận điều khoản sử dụng")

	if ctx.Err() != nil {
		return &RegResult{Success: false, Message: "Đã dừng"}
	}

	// ── B8 ─────────────────────────────────────────────────────────────
	notify("[B8] Đang tạo tài khoản...")
	b8Resp, b8SetCookies, b8Status, err := doRegPostFullRetry(ctx, s, endpointB8, buildB8Body(s, uuid.New().String()))
	if err != nil {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B8] Lỗi kết nối: %v", err)}
	}
	debugSave(s, "B8", b8Resp)
	if b8Status >= 400 {
		return &RegResult{Success: false, Message: fmt.Sprintf("[B8] Lỗi HTTP %d (kiểm tra proxy hoặc thử lại)", b8Status)}
	}
	if errMsg := bloksError(b8Resp); errMsg != "" {
		return &RegResult{Success: false, Message: "[B8] Lỗi: " + errMsg}
	}

	uid := parseUIDFromResponse(b8Resp)
	if uid == "" {
		if strings.Contains(b8Resp, "confirm_phone") || strings.Contains(b8Resp, "send_code") ||
			strings.Contains(b8Resp, "verify_phone") || strings.Contains(b8Resp, "confirmation_required") {
			return &RegResult{
				Success: true,
				UID:     "",
				Message: fmt.Sprintf("Cần xác nhận SĐT — %s %s | phone=%s", input.FirstName, input.LastName, input.Phone),
			}
		}
		return &RegResult{Success: false, Message: fmt.Sprintf("[B8] Không thể tạo tài khoản | phone=%s", input.Phone)}
	}
	cookieStr := buildCookieStringFromHeaders(b8SetCookies, uid, s.Datr)
	accessToken := extractAccessTokenFromHeaders(b8SetCookies)
	if accessToken == "" {
		accessToken = extractAccessToken(b8Resp)
	}
	if accessToken == "" && input.Password != "" && !SkipAuthLoginAtReg {
		notify("[B8] Đang lấy access token qua REST /auth/login (port S399)...")
		androidCtx, androidCancel := context.WithTimeout(ctx, 30*time.Second)
		// PORT S399 step 2: REST classic API stable, không phụ thuộc Bloks schema.
		tok := FetchAndroidTokenLegacy(androidCtx, uid, input.Password, s.Datr, "en_US", "", s.Proxy, "", notify)
		androidCancel()
		if tok != "" {
			accessToken = tok
			notify("[B8] Lấy được access token (Legacy REST)")
		} else {
			notify("[B8] Không lấy được access token (bỏ qua)")
		}
	}
	notify(fmt.Sprintf("[B8] Tạo tài khoản thành công! UID=%s", uid))

	// Save datr vào datr_pool.txt + add vào SharedPool (C# TryAddNewDatrToPool + SaveDatrFromCookieIfNew)
	if datr := cookiestore.ExtractDatr(cookieStr); datr != "" {
		if SharedPool != nil {
			if SharedPool.AddDatrRaw(datr) {
				trunc := datr
				if len(trunc) > 10 {
					trunc = trunc[:10]
				}
				notify(fmt.Sprintf("[Pool] Datr mới: %s... (pool size: %d)", trunc, SharedPool.Size()))
			}
		}
	}

	return &RegResult{
		Success:     true,
		UID:         uid,
		Cookie:      cookieStr,
		AccessToken: accessToken,
		Password:    input.Password,
		Message:     fmt.Sprintf("OK — %s %s | phone=%s | uid=%s", input.FirstName, input.LastName, input.Phone, uid),
	}
}

// updateSession cập nhật trạng thái session từ response body của một bước Bloks.
//
// Tham số:
//   - s: con trỏ tới RegSession hiện tại; các trường được ghi trực tiếp vào s.
//   - response: chuỗi response body thô từ server (JSON/Bloks DSL).
//
// Luồng parse:
//  1. reg_info: parse JSON string "reg_info" từ response; nếu có thì ghi đè
//     s.RegInfo (token mang trạng thái đăng ký, bắt buộc cho mỗi bước tiếp theo).
//  2. reg_context: parse JSON string "reg_context" từ response; nếu có thì ghi
//     đè s.RegContext (token ngữ cảnh phiên đăng ký).
//  3. public key: nếu s.PubKeyHex còn rỗng, tìm public key và key_id trong
//     response (Facebook đôi khi trả public key trong response Bloks thay vì
//     chỉ trong r.php). Ghi đè nếu tìm thấy.
func updateSession(s *RegSession, response string) {
	if ri := parseRegInfoFromResponse(response); ri != "" {
		s.RegInfo = ri
	}
	if rc := parseRegContextFromResponse(response); rc != "" {
		s.RegContext = rc
	}
	if s.PubKeyHex == "" {
		if pk, kid := extractBloksEncryptKey(response); pk != "" {
			s.PubKeyHex = pk
			if kid != "" {
				s.PubKeyID = kid
			}
		}
	}
}

// bloksError kiểm tra response Bloks có chứa lỗi đăng ký hay không.
//
// Facebook thường trả HTTP 200 ngay cả khi đăng ký thất bại; lỗi thật nằm
// trong body dưới dạng các pattern Bloks DSL như "registration_error",
// "phone_number_used", "create_failure", v.v.
//
// Tham số:
//   - response: chuỗi response body thô từ server.
//
// Trả về chuỗi mô tả lỗi (qua extractBloksErrorMsg) nếu phát hiện lỗi,
// hoặc chuỗi rỗng nếu response không có dấu hiệu lỗi.
func bloksError(response string) string {
	lower := strings.ToLower(response)

	for _, p := range []string{
		"registration_error",
		"phone_number_used",
		"invalid_phone",
		"bk_caa_reg_error",
		"account_creation_failed",
		"this phone number is already linked",
		"cannot create account",
		"create_failure",
		"exception_category\":\"system_error",
		"transition_failure_reg_tos",
	} {
		if strings.Contains(lower, p) {
			return extractBloksErrorMsg(response)
		}
	}

	if v := extractJSONStr(response, "errorSummary"); v != "" {
		return v
	}

	return ""
}

// extractBloksErrorMsg tìm và trả về chuỗi mô tả lỗi từ Bloks response.
//
// Tham số:
//   - response: chuỗi response body thô từ server.
//
// Thứ tự tìm kiếm (ưu tiên từ cụ thể đến tổng quát):
//  1. "errorSummary": tóm tắt lỗi chính xác nhất do Facebook cung cấp.
//  2. "title": tiêu đề dialog/modal lỗi.
//  3. "text": nội dung văn bản trong response Bloks.
//  4. "message": trường message chung.
//
// Nếu tìm được chuỗi có độ dài < 200 ký tự thì trả về luôn.
// Nếu không tìm được qua bốn key trên, trả về snippet 150 ký tự đầu của
// response kèm "..." để caller có context debug tối thiểu.
func extractBloksErrorMsg(response string) string {
	for _, key := range []string{"errorSummary", "title", "text", "message"} {
		if v := extractJSONStr(response, key); v != "" && len(v) < 200 {
			return v
		}
	}
	return snippet(response, 150)
}

// setRegRequestHeaders — đặt headers chuẩn cho mọi request đăng ký
func setRegRequestHeaders(req *http.Request, s *RegSession) {
	req.Header.Set("content-type", "application/x-www-form-urlencoded;charset=UTF-8")
	req.Header.Set("accept", "*/*")
	req.Header.Set("accept-language", "vi-VN,vi;q=0.9,fr-FR;q=0.8,fr;q=0.7,en-US;q=0.6,en;q=0.5")
	req.Header.Set("origin", "https://m.facebook.com")
	req.Header.Set("priority", "u=1, i")
	req.Header.Set("referer", refererReg)
	req.Header.Set("sec-ch-prefers-color-scheme", "light")
	req.Header.Set("sec-ch-ua", `"Chromium";v="134", "Not-A.Brand";v="24", "Google Chrome";v="134"`)
	req.Header.Set("sec-ch-ua-full-version-list", `"Chromium";v="134.0.6998.165", "Not-A.Brand";v="24.0.0.0", "Google Chrome";v="134.0.6998.165"`)
	req.Header.Set("sec-ch-ua-mobile", "?1")
	req.Header.Set("sec-ch-ua-model", `"iPhone"`)
	req.Header.Set("sec-ch-ua-platform", `"iOS"`)
	req.Header.Set("sec-ch-ua-platform-version", `"18.5"`)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("user-agent", s.UserAgent)
	if s.Datr != "" {
		req.Header.Set("Cookie", "datr="+s.Datr+"; ps_l=1; ps_n=1")
	}
}

// doRegPost gửi một POST request tới wbloks endpoint của Facebook.
//
// Tham số:
//   - ctx: context để hủy hoặc đặt timeout request.
//   - s: session hiện tại; dùng s.Proxy để tạo HTTP client và s.UserAgent,
//     s.Datr để đặt headers (bao gồm Cookie: datr=...).
//   - endpoint: URL đầy đủ của wbloks endpoint (ví dụ endpointB1).
//   - body: URL-encoded form body của request.
//
// Response body được giới hạn tối đa 2 MB (2<<20 bytes) để tránh OOM khi
// server trả về payload bất thường.
//
// Trả về (responseBody, httpStatusCode, error).
func doRegPost(ctx context.Context, s *RegSession, endpoint, body string) (string, int, error) {
	client := proxy.CreateClient(s.Proxy, 30*time.Second)
	defer client.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(body))
	if err != nil {
		return "", 0, err
	}
	setRegRequestHeaders(req, s)
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return string(respBody), resp.StatusCode, nil
}

// doRegPostFull giống doRegPost nhưng trả thêm toàn bộ Set-Cookie headers.
//
// Tham số:
//   - ctx: context để hủy hoặc đặt timeout request.
//   - s: session hiện tại; dùng s.Proxy, s.UserAgent, s.Datr như doRegPost.
//   - endpoint: URL đầy đủ của wbloks endpoint.
//   - body: URL-encoded form body của request.
//
// Set-Cookie headers cần thiết cho bước B8 vì server Facebook trả về
// c_user (UID), xs (session token), và access_token trong header thay vì
// body sau khi tạo tài khoản thành công. Các cookie này được dùng để
// build cookie string và extract access token kết quả.
//
// Response body cũng được giới hạn 2 MB.
//
// Trả về (responseBody, setCookieHeaders, httpStatusCode, error).
func doRegPostFull(ctx context.Context, s *RegSession, endpoint, body string) (respBody string, setCookies []string, statusCode int, err error) {
	client := proxy.CreateClient(s.Proxy, 30*time.Second)
	defer client.CloseIdleConnections()
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(body))
	if err != nil {
		return "", nil, 0, err
	}
	setRegRequestHeaders(req, s)
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, 0, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return string(b), resp.Header["Set-Cookie"], resp.StatusCode, nil
}

// retryDelays — exponential backoff: 1s, 2s, 4s (tổng tối đa ~15s cho 3 retries)
var retryDelays = []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

// shouldRetry — retry khi lỗi kết nối, HTTP 429 (rate limit), hoặc HTTP 503 (service unavailable)
func shouldRetry(err error, status int) bool {
	if err != nil {
		return true
	}
	return status == 429 || status == 503
}

// doRegPostRetry gửi POST request và tự động retry tối đa 3 lần khi gặp lỗi
// kết nối hoặc server trả về HTTP 429 (rate limit) / 503 (service unavailable).
//
// Tham số:
//   - ctx: context để hủy; nếu ctx bị hủy trong thời gian chờ backoff thì
//     hàm thoát ngay lập tức và trả về ctx.Err().
//   - s: session hiện tại, truyền thẳng vào doRegPost.
//   - endpoint: URL endpoint đích.
//   - body: URL-encoded form body.
//
// Chiến lược retry:
//   - Lần 1 thất bại → chờ 1s rồi thử lần 2.
//   - Lần 2 thất bại → chờ 2s rồi thử lần 3.
//   - Lần 3 thất bại → chờ 4s rồi thử lần 4 (lần cuối).
//   - Tổng thời gian chờ tối đa: 7s (không tính thời gian request).
//
// Điều kiện retry (shouldRetry): err != nil (lỗi mạng) HOẶC status 429/503.
func doRegPostRetry(ctx context.Context, s *RegSession, endpoint, body string) (string, int, error) {
	resp, status, err := doRegPost(ctx, s, endpoint, body)
	for i := 0; i < len(retryDelays) && shouldRetry(err, status); i++ {
		if ctx.Err() != nil {
			return resp, status, ctx.Err()
		}
		select {
		case <-time.After(retryDelays[i]):
		case <-ctx.Done():
			return resp, status, ctx.Err()
		}
		resp, status, err = doRegPost(ctx, s, endpoint, body)
	}
	return resp, status, err
}

// doRegPostFullRetry là phiên bản retry của doRegPostFull, dùng riêng cho B8.
//
// Cơ chế retry và backoff giống hệt doRegPostRetry (1s → 2s → 4s, tối đa
// 3 lần retry). Điểm khác biệt duy nhất là trả thêm []string Set-Cookie.
//
// Tham số:
//   - ctx: context để hủy trong khi chờ backoff.
//   - s: session hiện tại.
//   - endpoint: URL endpoint đích.
//   - body: URL-encoded form body.
//
// Trả về (responseBody, setCookieHeaders, httpStatusCode, error).
func doRegPostFullRetry(ctx context.Context, s *RegSession, endpoint, body string) (string, []string, int, error) {
	resp, cookies, status, err := doRegPostFull(ctx, s, endpoint, body)
	for i := 0; i < len(retryDelays) && shouldRetry(err, status); i++ {
		if ctx.Err() != nil {
			return resp, cookies, status, ctx.Err()
		}
		select {
		case <-time.After(retryDelays[i]):
		case <-ctx.Done():
			return resp, cookies, status, ctx.Err()
		}
		resp, cookies, status, err = doRegPostFull(ctx, s, endpoint, body)
	}
	return resp, cookies, status, err
}

// debugSave lưu response body ra file để kiểm tra trong quá trình phát triển.
//
// Tham số:
//   - s: session hiện tại; s.DebugDir là thư mục đầu ra (rỗng → hàm no-op).
//   - name: tên file (không có extension), thường là tên bước như "B1", "B8".
//   - body: nội dung response body cần lưu.
//
// File được ghi tại s.DebugDir/<name>.txt. Thư mục được tạo tự động nếu chưa
// tồn tại. Lỗi ghi file bị bỏ qua (dùng _ =) vì đây là chức năng debug
// tùy chọn, không ảnh hưởng đến luồng đăng ký chính.
func debugSave(s *RegSession, name, body string) {
	if s.DebugDir == "" {
		return
	}
	_ = os.MkdirAll(s.DebugDir, 0755)
	_ = os.WriteFile(s.DebugDir+"/"+name+".txt", []byte(body), 0644)
}

// snippet cắt ngắn chuỗi s về tối đa n ký tự và nối thêm "..." nếu bị cắt.
//
// Tham số:
//   - s: chuỗi gốc, thường là response body thô từ Facebook.
//   - n: số ký tự tối đa muốn giữ lại.
//
// Dùng trong extractBloksErrorMsg khi không tìm được key lỗi cụ thể, để
// trả về phần đầu response đủ ngắn làm fallback error message. Dấu "..."
// cuối giúp caller biết chuỗi đã bị cắt.
func snippet(s string, n int) string {
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

// truncReg cắt ngắn chuỗi s về tối đa n ký tự mà không thêm "...".
//
// Tham số:
//   - s: chuỗi gốc, thường là token hoặc cookie value.
//   - n: số ký tự tối đa muốn giữ lại.
//
// Khác snippet: không thêm dấu "..." sau khi cắt. Dùng trong log notify
// để hiển thị phần đầu token ngắn gọn (ví dụ: 20 ký tự đầu của fb_dtsg)
// mà không gây nhầm lẫn về việc có dữ liệu phía sau hay không.
func truncReg(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func normalizeInitialDatr(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if datr := cookiestore.ExtractDatr(raw); datr != "" {
		return datr
	}
	if strings.ContainsAny(raw, "|=;") {
		return ""
	}
	return raw
}
