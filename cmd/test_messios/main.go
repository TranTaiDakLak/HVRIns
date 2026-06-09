// cmd/test_messios — harness test flow iOS Messenger reg+verify (HIỆN TẠI, chưa fix).
// Đo từng giai đoạn: create / integrity_block / confirm / live / die.
// Chạy: go run ./cmd/test_messios [N]   (N = số luồng, default 10)
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"HVRIns/internal/email"
	"HVRIns/internal/instagram"
	iosmess "HVRIns/internal/instagram/register/ios/iosmess"
	web "HVRIns/internal/instagram/register/web"
	"HVRIns/internal/instagram/verify/verifybase"
)

const (
	proxy    = "unlimited.iprocket.io:12000:USERt1mbtV-zone-custom:Havu1988"
	datrFile = "build/bin/Config/Cookie/Pool20260608_7.txt"
	provider = "tempmail-plus"
)

// runNonce phân biệt session-id giữa các lần chạy (tránh tái dùng IP sticky cũ).
var runNonce = time.Now().UnixNano() % 1000000

// stickyProxy tạo proxy STICKY 1 IP/account: thêm -session-<id>-sessTime-10 vào username.
// BẮT BUỘC: cả flow 1 account (create→bottomsheet→change.email→add-mail→confirm) PHẢI đi 1 IP,
// nếu để proxy xoay IP mỗi request (mặc định zone-custom) → Facebook thấy phiên nhảy nước →
// integrity_block + add-mail không bind funnel (9KB thay vì OTP).
func stickyProxy(slot int) string {
	p := strings.Split(proxy, ":")
	if len(p) != 4 {
		return proxy
	}
	// -region-VN: IP Việt Nam (khớp account VN locale/+84 → giảm integrity_block).
	// -session-<id>-sessTime-10: sticky 1 IP/account suốt flow (funnel add-mail bind được).
	user := fmt.Sprintf("%s-region-VN-session-hvr%d_%d-sessTime-10", p[2], runNonce, slot)
	return strings.Join([]string{p[0], p[1], user, p[3]}, ":")
}

type stat struct {
	create, integrity, createFail, otpFail, confirmOK, confirmFail, live, die, unknown int64
}

func main() {
	n := 10
	if len(os.Args) > 1 {
		if v, err := strconv.Atoi(os.Args[1]); err == nil && v > 0 {
			n = v
		}
	}

	loaded := iosmess.LoadSharedPool([]string{datrFile})
	fmt.Printf("=== iOS Mess test — %d luồng | datr pool: %d | provider: %s ===\n", n, loaded, provider)
	if loaded == 0 {
		fmt.Println("!!! datr pool rỗng — dừng")
		return
	}

	var s stat
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3) // giới hạn 3 luồng đồng thời (tránh tempmail rate-limit)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(slot int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			runFlow(slot, &s)
		}(i)
	}
	wg.Wait()

	fmt.Printf("\n========== KẾT QUẢ (%d luồng) ==========\n", n)
	fmt.Printf("  create OK        : %d\n", atomic.LoadInt64(&s.create))
	fmt.Printf("  integrity_block  : %d  (die ở create)\n", atomic.LoadInt64(&s.integrity))
	fmt.Printf("  create fail khác : %d\n", atomic.LoadInt64(&s.createFail))
	fmt.Printf("  OTP fail         : %d\n", atomic.LoadInt64(&s.otpFail))
	fmt.Printf("  confirm OK       : %d\n", atomic.LoadInt64(&s.confirmOK))
	fmt.Printf("  confirm fail     : %d\n", atomic.LoadInt64(&s.confirmFail))
	fmt.Printf("  ----\n")
	fmt.Printf("  LIVE             : %d\n", atomic.LoadInt64(&s.live))
	fmt.Printf("  DIE              : %d\n", atomic.LoadInt64(&s.die))
	fmt.Printf("  Unknown          : %d\n", atomic.LoadInt64(&s.unknown))
}

func logf(slot int, format string, a ...interface{}) {
	fmt.Printf("[#%d] %s\n", slot, fmt.Sprintf(format, a...))
}

func runFlow(slot int, s *stat) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// STICKY: cả flow account này đi 1 IP cố định (create→add-mail→confirm cùng IP như GOOD).
	sp := stickyProxy(slot)

	// 1. tempmail (proxy xoay cho mail OK — không thuộc phiên FB)
	// Test domains: fexpost.com có thể bị blacklist — thử mailto.plus thay thế.
	svc, err := email.New(email.Options{Provider: provider, ProxyStr: proxy, TempMailDomain: "mailto.plus"})
	if err != nil {
		logf(slot, "email.New lỗi: %v", err)
		return
	}
	addr, err := svc.CreateEmail(ctx)
	if err != nil || addr == "" {
		logf(slot, "CreateEmail lỗi: %v", err)
		return
	}

	// 2. reg create (create-only) — dùng sticky proxy
	wctx, err := iosmess.NewWorkerContext(sp, "VN")
	if err != nil {
		logf(slot, "worker ctx: %v", err)
		return
	}
	defer wctx.Close()
	reg := wctx.Register(ctx, &instagram.RegInput{Email: addr, Proxy: sp, SlotIdx: slot}, func(m string) {
		if strings.Contains(m, "DIAG") || strings.Contains(m, "reg_context") {
			fmt.Printf("[#%d] %s\n", slot, m)
		}
	})
	if reg == nil || !reg.Success {
		msg := ""
		if reg != nil {
			msg = reg.Message
		}
		if strings.Contains(strings.ToLower(msg), "integrity_block") {
			atomic.AddInt64(&s.integrity, 1)
			logf(slot, "DIE integrity_block @ create | %s", msg)
		} else {
			atomic.AddInt64(&s.createFail, 1)
			logf(slot, "create fail: %s", msg)
		}
		return
	}
	atomic.AddInt64(&s.create, 1)

	uid := reg.UID
	crypted := reg.SessionlessCryptedUID
	device := reg.DeviceID
	family := reg.FamilyDeviceID
	datr := strings.TrimPrefix(reg.Cookie, "datr=")
	waterfall := reg.Srnonce
	ua := reg.UserAgent
	at := strings.SplitN(addr, "@", 2)
	local, domain := at[0], at[1]
	logf(slot, "create OK uid=%s crypted=%.10s", uid, crypted)

	// 3. verify: add-mail (trigger OTP) + screen-loads
	client, err := iosmess.NewIOSClient(sp, 60)
	if err != nil {
		logf(slot, "ios client: %v", err)
		return
	}
	// THỨ TỰ GOOD: render bottomsheet + change.email TRƯỚC (thiết lập ngữ cảnh change-email),
	// rồi mới submit contactpoint_email (add-mail) → server gửi OTP.
	for _, w := range []struct{ name, fn string }{
		{"bottomsheet", iosmess.FnBottomsheet}, {"change_email", iosmess.FnChangeMail},
	} {
		slb, _ := iosmess.BuildScreenLoadBody(w.name, device, family, datr, waterfall, uid, crypted, local, domain, reg.Phone, reg.AACJid, reg.AACcs, reg.AACts, reg.RegFlowID, reg.HeadersFlowID)
		sst, sresp, _ := iosmess.SendStep(client, slb, w.fn, device, ua)
		low := strings.ToLower(sresp)
		logf(slot, "%s RESP HTTP=%d %dB | error=%t should_show_error_true=%t CONTACTPOINT_EMAIL=%t CONFIRMATION=%t",
			w.name, sst, len(sresp),
			strings.Contains(low, "\"error\"") || strings.Contains(low, "exception"),
			strings.Contains(sresp, "should_show_error\\\":true") || strings.Contains(sresp, "should_show_error\":true"),
			strings.Contains(sresp, "CONTACTPOINT_EMAIL"), strings.Contains(sresp, "CAA_REG_CONFIRMATION"))
		_ = os.WriteFile(fmt.Sprintf("%s_%d.json", w.name, slot), []byte(sresp), 0o644)
		time.Sleep(400 * time.Millisecond)
	}
	amBody, _ := iosmess.BuildAddMailBody(device, family, datr, waterfall, uid, crypted, local, domain, reg.Phone, reg.AACJid, reg.AACcs, reg.AACts, reg.RegFlowID, reg.HeadersFlowID, reg.PassRaw, reg.PassTS)
	_ = os.WriteFile(fmt.Sprintf("amreq_%d.txt", slot), []byte(amBody), 0o644)
	logf(slot, "add-mail REQ: crypted=%t msg_prev_cp=%t email=%t aac_fresh=%t flow_fresh=%t(reg=%.8s)",
		strings.Contains(amBody, crypted[:12]), strings.Contains(amBody, strings.TrimPrefix(reg.Phone, "+")), strings.Contains(amBody, local),
		reg.AACJid != "" && !strings.Contains(amBody, "dceb25df-a981-4dd3-b4ee-5b8b5bfd4042"),
		reg.RegFlowID != "" && !strings.Contains(amBody, "175bfeb3-68ba-4d04-9332-e60c29c23c77"), reg.RegFlowID)
	amSt, amResp, _ := iosmess.SendStep(client, amBody, iosmess.FnAddMail, device, ua)
	low := strings.ToLower(amResp)
	mark := func(s string) bool { return strings.Contains(low, strings.ToLower(s)) }
	logf(slot, "add-mail HTTP=%d %dB | confirm_code=%t checkEmail=%t caa_reg_conf=%t invalid=%t",
		amSt, len(amResp), mark("confirmation_code"), mark("check your email"), mark("caa_reg_confirmation"), mark("invalid_input"))
	_ = os.WriteFile(fmt.Sprintf("am_%d.json", slot), []byte(amResp), 0o644)

	// 4. OTP
	code, err := svc.WaitForCode(ctx, 15, 4000)
	if err != nil || code == "" {
		atomic.AddInt64(&s.otpFail, 1)
		logf(slot, "OTP fail: %v", err)
		return
	}
	logf(slot, "OTP=%s", code)

	// 5. confirm
	cfBody, _ := iosmess.BuildConfirmBodyVerify(device, family, datr, waterfall, uid, crypted, local, domain, code, reg.AACJid, reg.AACcs, reg.AACts, reg.RegFlowID, reg.HeadersFlowID)
	_, cfResp, _ := iosmess.SendStep(client, cfBody, iosmess.FnConfirm, device, ua)
	if !iosmess.IsConfirmOK(cfResp) {
		atomic.AddInt64(&s.confirmFail, 1)
		snip := cfResp
		if len(snip) > 120 {
			snip = snip[:120]
		}
		logf(slot, "confirm FAIL: %s", snip)
		return
	}
	atomic.AddInt64(&s.confirmOK, 1)
	logf(slot, "confirm OK")

	// 6. live/die — login lấy token rồi check
	tok, _ := web.FetchAndroidTokenLegacyWithCookie(ctx, uid, reg.Password, datr, "vi_VN", "VN", sp, "", nil)
	status := "Unknown"
	if tok != "" {
		status = verifybase.CheckLiveDieByToken(ctx, ua, tok)
	}
	if status == "Unknown" {
		status = verifybase.CheckLiveDieByPicture(ctx, ua, uid)
	}
	switch status {
	case "Live":
		atomic.AddInt64(&s.live, 1)
	case "Die":
		atomic.AddInt64(&s.die, 1)
	default:
		atomic.AddInt64(&s.unknown, 1)
	}
	logf(slot, "LIVE/DIE = %s (token=%t)", status, tok != "")
}
