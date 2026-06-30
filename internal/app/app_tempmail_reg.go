package app

// app_tempmail_reg.go — TempMail reuse helpers cho RegMode=TempMail.
//
// Flow:
//   1. Spawner detect Mode=TempMail → clear phone + set Email="" (sentinel).
//   2. Inside goroutine, BEFORE reg.Register: acquireTempMailForReg() tạo mail
//      thật từ provider (zeus-x/dvfb/store1s/...) → set prof.Email + prof.EmailMeta.
//   3. After register: success → pass EmailMeta vào runner.AccountInput cho
//      verify Restore. Fail → email.ReleaseIfPossible() trả về pool.
//
// Provider-specific creds được serialize qua Snapshotter interface (xem
// internal/email/service.go). Verify side gọi RestoreIfPossible(svc, meta) →
// skip CreateEmail + skip "Add email to account" step.

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"HVRIns/internal/email"
	"HVRIns/internal/email/temp"
	"HVRIns/internal/instagram"
)

// isTempMailMode — check RegMode hiện tại có phải "TempMail" không.
func isTempMailMode(_ instagram.VerifyConfig) bool {
	// VerifyConfig là chỗ gốc có MailProvider + keys. RegMode lưu ở
	// InteractionConfig (verifyConfig.go đã merge vào cùng struct ở app side).
	// Sentinel check: RegMode field nằm ở app.InteractionConfig — caller pass.
	return false // placeholder; caller dùng helper isTempMailModeStr trực tiếp.
}

// isTempMailModeStr — string-based check cho RegMode value.
func isTempMailModeStr(regMode string) bool {
	return equalFoldTrim(regMode, "TempMail")
}

// isMailTempModeStr — check RegMode == "MailTemp" (mail-temp.com, client-side, no API key).
func isMailTempModeStr(regMode string) bool {
	return equalFoldTrim(regMode, "MailTemp")
}

func equalFoldTrim(a, s string) bool {
	a = trimSpaceLower(a)
	return a == trimSpaceLower(s)
}

func trimSpaceLower(s string) string {
	out := make([]byte, 0, len(s))
	started := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			if !started {
				continue
			}
			continue
		}
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		started = true
		out = append(out, c)
	}
	return string(out)
}

// buildEmailOptionsForReg — build email.Options từ verify config cho register flow.
//
// Lý do dùng cùng VerifyConfig (thay vì tách field): user setup mail provider
// keys 1 lần qua "Thiết lập chạy" → cả reg và verify dùng chung. Tránh duplicate
// cấu hình.
//
// proxyStr: proxy của thread hiện tại (đã render session). Email service sẽ
// dùng proxy này để gọi provider API (giảm rủi ro IP block khi mua nhiều mail).
//
// notify: callback log status từ email service (vd "[ZeusX] Mua mail #1...").
func buildEmailOptionsForReg(cfg instagram.VerifyConfig, proxyStr string, notify func(string)) email.Options {
	// Proxy override: rent provider ưu tiên proxy_rentmail.txt nếu user bật.
	proxyOverride := ""
	if email.IsRentMailProvider(cfg.MailProvider) {
		if cfg.UseProxyGmail {
			proxyOverride = email.PickRentMailProxy()
		}
	} else if cfg.UseProxyTempMail {
		proxyOverride = email.PickTempMailProxy()
	}

	return email.Options{
		Provider:                cfg.MailProvider,
		ProxyStr:                proxyStr,
		ProxyOverride:           proxyOverride,
		OnStatus:                notify,
		Pool:                    cfg.EmailPool,
		ZeusXApiKey:             cfg.ZeusXApiKey,
		ZeusXAccountCode:        cfg.ZeusXAccountCode,
		DvfbApiKey:              cfg.DvfbApiKey,
		DvfbAccountType:         cfg.DvfbAccountType,
		Store1sApiKey:           cfg.Store1sApiKey,
		Store1sProductID:        cfg.Store1sProductID,
		Mail30sApiKey:           cfg.Mail30sApiKey,
		Mail30sProductSlug:      cfg.Mail30sProductSlug,
		TempMailLolApiKey:       cfg.TempMailLolApiKey,
		TempMailDomain:          cfg.TempMailDomain,
		MuaMailApiKey:           cfg.MuaMailApiKey,
		MuaMailProductID:        cfg.MuaMailProductID,
		UnlimitMailApiKey:       cfg.UnlimitMailApiKey,
		UnlimitMailProductID:    cfg.UnlimitMailProductID,
		SptMailApiKey:           cfg.SptMailApiKey,
		SptMailServiceCode:      cfg.SptMailServiceCode,
		EmailAPIInfoApiKey:      cfg.EmailAPIInfoApiKey,
		EmailAPIInfoProductCode: cfg.EmailAPIInfoProductCode,
		OtpCheapApiKey:          cfg.OtpCheapApiKey,
		OtpCheapServiceID:       cfg.OtpCheapServiceID,
		ShopGmail9999ApiKey:     cfg.ShopGmail9999ApiKey,
		ShopGmail9999Service:    cfg.ShopGmail9999Service,
		RentGmailApiKey:         cfg.RentGmailApiKey,
		RentGmailPlatform:       cfg.RentGmailPlatform,
		OtpCodesSmsApiKey:       cfg.OtpCodesSmsApiKey,
		OtpCodesSmsServiceID:    cfg.OtpCodesSmsServiceID,
		WmemailApiKey:           cfg.WmemailApiKey,
		WmemailCommodity:        cfg.WmemailCommodity,
		PriyoEmailApiKey:        cfg.PriyoEmailApiKey,
		TempMailToken:           cfg.TempMailToken,
	}
}

// TempMailHandle — wraps email service + creds cho 1 reg attempt. Caller giữ
// để: (a) read email.GetEmail() làm contactpoint, (b) read EmailMeta để truyền
// sang verify, (c) call Release() khi reg fail.
type TempMailHandle struct {
	Service email.Service
	Email   string // địa chỉ email
	Meta    string // serialized creds (Snapshotter blob)
}

// Close — cleanup service. Idempotent.
func (h *TempMailHandle) Close() {
	if h != nil && h.Service != nil {
		h.Service.Close()
	}
}

// ReleaseAndClose — return mail về pool (best-effort) rồi Close.
// Gọi khi reg fail/blocked → mail không cần verify nữa.
func (h *TempMailHandle) ReleaseAndClose(ctx context.Context) {
	if h == nil || h.Service == nil {
		return
	}
	email.ReleaseIfPossible(ctx, h.Service)
	h.Service.Close()
}

// acquireIOSMessVerMail — iOS Mess reg phone-only: lấy tempmail Ở BƯỚC VER (chỉ khi account
// chưa có email) để add-mail + verify. Trả handle (nil nếu không phải iosmess / đã có email /
// lỗi acquire); caller set acc.Email = h.Email, acc.EmailMeta = h.Meta rồi defer h.Close().
// Guard verifyPlatform=="iosmess" && curEmail=="" → no-op cho mọi platform/account khác.
func (a *App) acquireIOSMessVerMail(ctx context.Context, cfg instagram.VerifyConfig, proxy, verifyPlatform, curEmail string, notify func(string)) *TempMailHandle {
	if !strings.EqualFold(strings.TrimSpace(verifyPlatform), "iosmess") || strings.TrimSpace(curEmail) != "" {
		return nil
	}
	if notify != nil {
		notify("[iOS Mess] acquire tempmail để add-mail ver...")
	}
	h, err := acquireTempMailForReg(ctx, cfg, proxy, notify)
	if err != nil || h == nil {
		if err != nil && notify != nil {
			notify("[iOS Mess] acquire tempmail (ver) lỗi: " + err.Error())
		}
		return nil
	}
	if notify != nil {
		notify(fmt.Sprintf("[iOS Mess] tempmail ver: %s (meta=%dB)", h.Email, len(h.Meta)))
	}
	return h
}

// buildVerifyConfigFromInteraction — extract chỉ MailProvider + provider keys
// từ InteractionConfig sang VerifyConfig. Đủ cho acquireTempMailForReg dùng.
//
// KHÔNG include Verify-specific fields (Delay, MaxResend, AddInfo, etc.) vì
// register flow chỉ cần email service init, không chạy verify steps.
func buildVerifyConfigFromInteraction(ic InteractionConfig) instagram.VerifyConfig {
	return instagram.VerifyConfig{
		MailProvider:            ic.MailProvider,
		MailList:                ic.MailList,
		ZeusXApiKey:             ic.ZeusXApiKey,
		ZeusXAccountCode:        ic.ZeusXAccountCode,
		DvfbApiKey:              ic.DvfbApiKey,
		DvfbAccountType:         ic.DvfbAccountType,
		Store1sApiKey:           ic.Store1sApiKey,
		Store1sProductID:        ic.Store1sProductID,
		Mail30sApiKey:           ic.Mail30sApiKey,
		Mail30sProductSlug:      ic.Mail30sProductSlug,
		TempMailLolApiKey:       ic.TempMailLolApiKey,
		TempMailDomain:          ic.TempMailDomain,
		MuaMailApiKey:           ic.MuaMailApiKey,
		MuaMailProductID:        ic.MuaMailProductID,
		UnlimitMailApiKey:       ic.UnlimitMailApiKey,
		UnlimitMailProductID:    ic.UnlimitMailProductID,
		SptMailApiKey:           ic.SptMailApiKey,
		SptMailServiceCode:      ic.SptMailServiceCode,
		EmailAPIInfoApiKey:      ic.EmailAPIInfoApiKey,
		EmailAPIInfoProductCode: ic.EmailAPIInfoProductCode,
		OtpCheapApiKey:          ic.OtpCheapApiKey,
		OtpCheapServiceID:       ic.OtpCheapServiceID,
		ShopGmail9999ApiKey:     ic.ShopGmail9999ApiKey,
		ShopGmail9999Service:    ic.ShopGmail9999Service,
		RentGmailApiKey:         ic.RentGmailApiKey,
		RentGmailPlatform:       ic.RentGmailPlatform,
		OtpCodesSmsApiKey:       ic.OtpCodesSmsApiKey,
		OtpCodesSmsServiceID:    ic.OtpCodesSmsServiceID,
		WmemailApiKey:           ic.WmemailApiKey,
		WmemailCommodity:        ic.WmemailCommodity,
		PriyoEmailApiKey:        ic.PriyoEmailApiKey,
		TempMailToken:           ic.TempMailToken,
		UseProxyGmail:           ic.UseProxyGmail,
		UseProxyTempMail:        ic.UseProxyTempMail,
	}
}

// acquireTempMailForReg — tạo mail tạm thật cho 1 reg attempt.
//
// Trả nil + error nếu:
//   - cfg.MailProvider rỗng (user chưa chọn provider trong "NGUỒN XÁC THỰC")
//   - provider key không hợp lệ (email.New trả error)
//   - CreateEmail fail (provider out-of-stock, network error, etc.)
//   - CreateEmail trả "" (provider succeed nhưng không trả address — bug provider)
//
// Caller phải handle error (vd skip thread hoặc fall back về Mode=Mail random).
//
// Khi success, caller MUST call Close() hoặc ReleaseAndClose() để cleanup.
//
// Snapshot là best-effort: nếu provider chưa implement Snapshotter, Meta = ""
// và verify side sẽ fall back về CreateEmail mới (không reuse).
func acquireTempMailForReg(ctx context.Context, cfg instagram.VerifyConfig, proxyStr string, notify func(string)) (*TempMailHandle, error) {
	if strings.TrimSpace(cfg.MailProvider) == "" {
		return nil, fmt.Errorf("MailProvider chưa chọn — vào 'Thiết lập chạy' → 'NGUỒN XÁC THỰC' chọn provider")
	}
	// tryProvider: New + CreateEmail 1 lần cho provider chỉ định.
	tryProvider := func(provider string) (*TempMailHandle, error) {
		o := buildEmailOptionsForReg(cfg, proxyStr, notify)
		o.Provider = provider
		svc, err := email.New(o)
		if err != nil {
			return nil, fmt.Errorf("email.New(%q): %w", provider, err)
		}
		addr, err := svc.CreateEmail(ctx)
		if err != nil {
			svc.Close()
			return nil, fmt.Errorf("CreateEmail(%q): %w", provider, err)
		}
		if strings.TrimSpace(addr) == "" {
			svc.Close()
			return nil, fmt.Errorf("CreateEmail(%q) trả address rỗng", provider)
		}
		meta, _ := email.SnapshotIfPossible(svc) // empty nếu provider chưa support
		return &TempMailHandle{Service: svc, Email: addr, Meta: meta}, nil
	}

	// 1. Thử provider user chọn, retry tối đa 3 lần — server-side hay rate-limit khi nhiều
	//    luồng burst; retry với backoff giúp vượt qua rate-limit tạm thời.
	const maxAttempts = 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		h, err := tryProvider(cfg.MailProvider)
		if err == nil {
			return h, nil
		}
		lastErr = err
		// KHÔNG nuốt lỗi: log lý do thật (anti-bot HTML challenge / 429 rate-limit /
		// timeout) để chẩn đoán khi máy khác báo "tạo lỗi" mà không rõ vì sao.
		slog.Warn("[TempMail] create FAIL",
			"provider", cfg.MailProvider,
			"attempt", attempt,
			"max", maxAttempts,
			"err", err)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if attempt < maxAttempts {
			if notify != nil {
				// Hiện luôn lý do thật lên UI (cắt ngắn) thay vì chỉ "tạo lỗi".
				reason := err.Error()
				if len(reason) > 120 {
					reason = reason[:120] + "…"
				}
				notify(fmt.Sprintf("[TempMail] %s tạo lỗi (%d/%d): %s — thử lại...", cfg.MailProvider, attempt, maxAttempts, reason))
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
	}

	// 2. Provider chọn fail hết → fallback firetempmail (client-side, không gọi server tạo
	//    mail nên không bao giờ rate-limit) để luồng vẫn có mail thay vì bỏ.
	const fallback = "firetempmail"
	if !strings.EqualFold(strings.TrimSpace(cfg.MailProvider), fallback) {
		if notify != nil {
			notify(fmt.Sprintf("[TempMail] %s fail → fallback %s (client-side)", cfg.MailProvider, fallback))
		}
		if h, err := tryProvider(fallback); err == nil {
			return h, nil
		}
	}
	return nil, fmt.Errorf("acquireTempMail: %s fail sau %d lần + fallback fail: %w", cfg.MailProvider, maxAttempts, lastErr)
}

// acquireMailTempComForReg — tạo email từ mail-temp.com (client-side, không cần API key).
// Khác acquireTempMailForReg: không cần MailProvider config, không cần provider keys.
// Meta = "" vì mail-temp.com không hỗ trợ Snapshotter → verify side sẽ không reuse.
func acquireMailTempComForReg(ctx context.Context, proxyStr string, notify func(string)) (*TempMailHandle, error) {
	svc := temp.NewMailTempCom(proxyStr)
	addr, err := svc.CreateEmail(ctx)
	if err != nil {
		return nil, fmt.Errorf("MailTempCom.CreateEmail: %w", err)
	}
	if strings.TrimSpace(addr) == "" {
		return nil, fmt.Errorf("MailTempCom.CreateEmail trả address rỗng")
	}
	if notify != nil {
		notify(fmt.Sprintf("[MailTemp] Email: %s", addr))
	}
	return &TempMailHandle{
		Service: svc,
		Email:   addr,
		Meta:    "",
	}, nil
}
