// ig_register.go — Instagram register adapter.
//
// Bọc engine reg Instagram thật (internal/igcore, port từ IGDesktop) vào interface
// Registerer của HVRIns. Toàn bộ platform register giờ chạy flow Instagram Bloks:
//   session → qe/sync → aymh → submitEmail → confirmOTP → setPassword →
//   setBirthday → setName → setUsername → acceptTOS → createAccount.
//
// Email + OTP do caller (runner) cấp qua RegInput.Email + RegInput.GetOTP
// (HVRIns đã có sẵn internal/email cho việc tạo mail tạm + đọc OTP). Proxy lấy
// từ RegInput.Proxy. Không dùng datr-pool / cookie-initial / UA-pool của flow FB cũ.
package instagram

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"HVRIns/internal/igcore"
)

// igRegisterer implement Registerer bằng engine Instagram (igcore).
type igRegisterer struct{}

// newIGRegisterer tạo adapter reg Instagram.
func newIGRegisterer() Registerer { return &igRegisterer{} }

// Register chạy đầy đủ flow đăng ký Instagram cho 1 account.
func (r *igRegisterer) Register(ctx context.Context, input *RegInput, onStatus func(string)) *RegResult {
	log := func(format string, args ...any) {
		if onStatus != nil {
			onStatus(fmt.Sprintf(format, args...))
		}
	}

	if input == nil {
		return &RegResult{Success: false, Message: "RegInput nil"}
	}
	// IG flow cần email + OTP reader do runner cấp (HVRIns internal/email).
	if strings.TrimSpace(input.Email) == "" || input.GetOTP == nil {
		return &RegResult{
			Success: false,
			Message: "IG register cần Email + GetOTP (mode TempMail/Mail) — chưa được cấp từ runner",
		}
	}
	addr := strings.TrimSpace(input.Email)
	password := input.Password
	if password == "" {
		password = fmt.Sprintf("Wemake@%04dXz", rand.Intn(10000))
	}

	// ── Session TLS + detect country ──────────────────────────────────────────
	log("init Khởi tạo session IG...")
	sess, err := igcore.NewIGSession(input.Proxy)
	if err != nil {
		return &RegResult{Success: false, Email: addr, Message: "session: " + err.Error()}
	}
	_, country := sess.CheckProxyIPCountry(ctx)
	if country == "" {
		country = "VN"
	}
	p := igcore.NewProfileForCountry(country)
	// Emit iOS UA sớm → app_register.go cập nhật prof.UserAgent ngay, hiển thị cột UA
	// trong khi reg vẫn đang chạy (không phải đợi Register() return).
	log("ua:%s", p.UserAgent)

	// ── Inject aged device (mid/datr/ig_did) từ pool nếu có ──────────────────
	// Tương tự cơ chế datr-pool bên FB: thiết bị "đã từng reg thành công"
	// có trust score cao hơn thiết bị hoàn toàn mới.
	if pool := igcore.SharedDevicePool; pool != nil {
		if dev := pool.Next(); dev != nil {
			sess.InjectAgedDevice(p, dev)
			midShort := dev.Mid
			if len(midShort) > 8 {
				midShort = midShort[:8] + "..."
			}
			log("device Inject aged device mid=%s ig_did=%s", midShort, dev.IgDID[:min(8, len(dev.IgDID))])
		}
	}

	// ── qe/sync → encryption key + X-MID (retry rotate IP) ────────────────────
	log("qesync Lấy encryption key...")
	var keyID, pubKey, xmid string
	useProxy := input.Proxy
	for i := 1; i <= 5; i++ {
		keyID, pubKey, xmid, err = sess.QeSync(ctx, p)
		if err == nil {
			break
		}
		useProxy = igcore.RotateSession(input.Proxy)
		if ns, e2 := igcore.NewIGSession(useProxy); e2 == nil {
			sess = ns
			p = igcore.NewProfileForCountry(country)
		}
		select {
		case <-ctx.Done():
			return &RegResult{Success: false, Email: addr, Message: "đã dừng"}
		case <-time.After(time.Duration(i) * time.Second):
		}
	}
	if err != nil {
		return &RegResult{Success: false, Email: addr, Message: "qe/sync fail: " + err.Error()}
	}
	p.MachineID = xmid

	eng := igcore.NewEngine(sess, p, keyID, pubKey, input.Proxy, func(string, ...any) {})

	// ── aymh ──────────────────────────────────────────────────────────────────
	log("aymh Bước khởi tạo...")
	_ = eng.Aymh(ctx)

	// ── submit email (retry on throttle, rotate IP) ───────────────────────────
	log("submit Gửi email...")
	submitOK := false
	for attempt := 1; attempt <= 15; attempt++ {
		if attempt > 1 {
			useProxy = igcore.RotateSession(input.Proxy)
			if ns, e2 := igcore.NewIGSession(useProxy); e2 == nil {
				np := igcore.NewProfileForCountry(country)
				if kid, pk, xm, e3 := ns.QeSync(ctx, np); e3 == nil {
					keyID, pubKey = kid, pk
					np.MachineID = xm
					p = np
					eng = igcore.NewEngine(ns, p, keyID, pubKey, useProxy, func(string, ...any) {})
				}
			}
			select {
			case <-ctx.Done():
				return &RegResult{Success: false, Email: addr, Message: "đã dừng"}
			case <-time.After(time.Duration(attempt) * time.Second):
			}
			log("submit Retry %d/15...", attempt)
		}
		if err = eng.SubmitEmail(ctx, addr); err == nil {
			submitOK = true
			break
		}
		if !igcore.IsThrottled(err) {
			return &RegResult{Success: false, Email: addr, Message: "submit: " + err.Error()}
		}
	}
	if !submitOK {
		return &RegResult{Success: false, Email: addr, Message: "submit throttled"}
	}

	// ── chờ OTP (runner cấp qua GetOTP) ───────────────────────────────────────
	log("otp Chờ OTP...")
	otp, err := input.GetOTP(ctx)
	if err != nil || strings.TrimSpace(otp) == "" {
		msg := "không nhận được OTP"
		if err != nil {
			msg = "OTP: " + err.Error()
		}
		return &RegResult{Success: false, Email: addr, Message: msg}
	}
	log("otp OTP: %s", otp)

	// ── confirm OTP ───────────────────────────────────────────────────────────
	log("confirm Xác nhận OTP...")
	for c := 1; c <= 5; c++ {
		err = eng.ConfirmOTP(ctx, addr, otp)
		if err == nil {
			break
		}
		if !igcore.IsThrottled(err) {
			return &RegResult{Success: false, Email: addr, Message: "confirmOTP: " + err.Error()}
		}
		time.Sleep(time.Duration(c) * time.Second)
	}
	if err != nil {
		return &RegResult{Success: false, Email: addr, Message: "confirmOTP fail: " + err.Error()}
	}

	// ── password / birthday / name / username / tos / create ──────────────────
	log("password Đặt mật khẩu...")
	if err := eng.SetPassword(ctx, addr, password); err != nil {
		return &RegResult{Success: false, Email: addr, Message: "password: " + err.Error()}
	}
	log("birthday Đặt ngày sinh...")
	_ = eng.SetBirthday(ctx, addr)

	name := buildIGName(input)
	log("name Đặt tên: %s", name)
	if err := eng.SetNameIG(ctx, addr, name); err != nil {
		return &RegResult{Success: false, Email: addr, Message: "name: " + err.Error()}
	}

	username := buildIGUsername()
	log("username Đặt username: %s", username)
	if err := eng.SetUsername(ctx, addr, username); err != nil {
		return &RegResult{Success: false, Email: addr, Message: "username: " + err.Error()}
	}

	_ = eng.AcceptTOS(ctx, addr)

	log("create Tạo tài khoản...")
	if err := eng.CreateAccount(ctx, addr, username, name); err != nil {
		return &RegResult{Success: false, Email: addr, Message: "create: " + err.Error()}
	}

	s := eng.Session()
	log("done 🎉 Thành công! UID: %s", s.UID)

	// Harvest mid/datr/ig_did vào pool → dùng cho reg tiếp theo
	if pool := igcore.SharedDevicePool; pool != nil && s.Mid != "" {
		if pool.Add(s.Mid, s.Datr, s.IgDID) {
			log("device Harvest vào pool: mid=%s... (pool size tăng)", s.Mid[:min(8, len(s.Mid))])
		}
	}

	log("checklive Kiểm tra live/die sau reg...")
	liveStatus := igcore.CheckLiveByCookie(ctx, s.FullCookie, p.UserAgent, input.Proxy)
	log("checklive → %s", liveStatus)

	base := &RegResult{
		UID:        s.UID,
		Cookie:     s.FullCookie,
		Password:   password,
		Email:      addr,
		UserAgent:  p.UserAgent,
		Username:   username,
		LiveStatus: liveStatus,
	}

	switch liveStatus {
	case "live":
		base.Success = true
		base.Message = "ok"
	case "die", "suspended":
		base.Success = false
		base.Message = "block: tài khoản bị khóa sau reg"
	default: // "checkpoint" hoặc "unknown" → coi như live, cần verify thêm
		base.Success = true
		base.Message = "ok"
	}
	return base
}

// buildIGName tạo tên hiển thị từ input hoặc random.
func buildIGName(input *RegInput) string {
	first := strings.TrimSpace(input.FirstName)
	last := strings.TrimSpace(input.LastName)
	full := strings.TrimSpace(last + " " + first)
	if full != "" {
		return full
	}
	pool := []string{"Alex", "Jordan", "Taylor", "Morgan", "Casey", "Riley", "Avery", "Quinn"}
	return fmt.Sprintf("%s%d", pool[rand.Intn(len(pool))], rand.Intn(9000)+1000)
}

// buildIGUsername sinh username dạng "word.NNNNNNN".
func buildIGUsername() string {
	words := []string{
		"hamster", "panda", "tiger", "eagle", "dolphin", "falcon", "otter", "koala",
		"rabbit", "wolf", "fox", "bear", "lion", "hawk", "puma", "lynx",
		"maple", "willow", "cedar", "aspen", "river", "ocean", "summer", "winter",
		"comet", "nova", "orbit", "pixel", "echo", "lunar", "solar", "ember",
	}
	return fmt.Sprintf("%s.%d", words[rand.Intn(len(words))], rand.Intn(9000000)+1000000)
}
