// app_verify.go — Luồng verify/xác thực (RunVerify), tách từ app.go.
// Di chuyển nguyên hàm — KHÔNG sửa logic.
package app

import (
	"HVRIns/internal/clonehv"
	"HVRIns/internal/cookie"
	emailrent "HVRIns/internal/email/rent"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	uploadavatar "HVRIns/internal/instagram/interaction/android"
	androidreg "HVRIns/internal/instagram/register/android"
	"HVRIns/internal/proxy"
	resultpkg "HVRIns/internal/result"
	"HVRIns/internal/runner"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	goruntime "runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) RunVerify(cfgOverride VerifyRunConfig) string {
	// Ngăn chạy đồng thời verify + register (cả hai dùng emailPool).
	// State machine cover cả running + stopping → Start verify chỉ được phép khi register idle.
	switch runState(a.registerState.Load()) {
	case runStateRunning:
		return "Register đang chạy, vui lòng dừng trước khi chạy Verify"
	case runStateStopping:
		return "Register đang dừng — vui lòng chờ hoàn tất"
	}

	a.verifyMu.Lock()
	if a.isRunning {
		a.verifyMu.Unlock()
		return "Đang chạy, vui lòng dừng trước khi chạy lại"
	}
	if a.verifyStopping.Load() {
		a.verifyMu.Unlock()
		return "Đang dừng run cũ — vui lòng chờ hoàn tất rồi start lại"
	}
	a.isRunning = true
	a.verifyMu.Unlock()

	// Đọc cài đặt chung đã lưu
	settings := a.LoadSettings()
	// Đọc thiết lập chạy (interaction config)
	interaction := a.LoadInteractionConfig()

	// Áp per-platform UA config cho verify side
	interaction = applyVerifyPlatformUAConfig(interaction)

	// === VALIDATION TRƯỚC KHI CHẠY ===
	failValidation := func(msg string) string {
		a.verifyMu.Lock()
		a.isRunning = false
		a.verifyMu.Unlock()
		// Validation fail: chưa start workers, không có gì cần stop → clear stopping
		a.verifyStopping.Store(false)
		return msg
	}

	isCloneHVCheck := isCloneHVActive(settings)

	// 1. Nguồn tài khoản
	if isCloneHVCheck {
		// API mode: phải có đủ credentials CloneHV
		if settings.General.CloneHVUsername == "" {
			return failValidation("Thiếu cấu hình: Username CloneHV chưa được nhập (Cài đặt chung)")
		}
		if settings.General.CloneHVPassword == "" {
			return failValidation("Thiếu cấu hình: Password CloneHV chưa được nhập (Cài đặt chung)")
		}
		if settings.General.CloneHVProductID == "" {
			return failValidation("Thiếu cấu hình: Product ID CloneHV chưa được nhập (Cài đặt chung)")
		}
	} else if strings.EqualFold(settings.General.AccountSource, "file") || (len(cfgOverride.AccountIDs) > 0 && len(a.accounts) > 0) {
		// FILE mode (mới): user pick 1 file .txt → load accounts vào grid → tick chọn để verify.
		// Validation: CHỈ yêu cầu có account tick chọn + account store không rỗng.
		// KHÔNG validate AccountSourcePath (có thể stale từ folder mode cũ; đã load accounts vào store rồi — path moot).
		a.accountsMu.RLock()
		accLen := len(a.accounts)
		a.accountsMu.RUnlock()
		if accLen == 0 {
			return failValidation("Chưa load accounts nào — vào Cài đặt chung → 'Từ 1 file' → pick file")
		}
		if len(cfgOverride.AccountIDs) == 0 {
			return failValidation("Chưa tick chọn account nào — tick các account muốn verify ở grid rồi bấm Chạy")
		}
	} else {
		// FOLDER mode (mặc định): streaming reads .txt files từ thư mục
		if interaction.VerifySourceFolderPath != "" {
			// Verify-only mode: dùng thư mục riêng được chọn trong thiết lập chạy
			if errMsg := a.ValidatePath(interaction.VerifySourceFolderPath); errMsg != "" {
				return failValidation("Thư mục verify tài khoản không hợp lệ: " + errMsg)
			}
		} else {
			// Fallback: nếu general.json chưa có path, lấy từ app_settings (set bởi Import dialog)
			if settings.General.AccountSourcePath == "" {
				settings.General.AccountSourcePath = a.GetAccountSourceFolder()
			}
			if settings.General.AccountSourcePath == "" {
				return failValidation("Thiếu cấu hình: Chưa chọn thư mục nguồn tài khoản (Cài đặt chung hoặc Import → Chọn thư mục)")
			}
			if errMsg := a.ValidatePath(settings.General.AccountSourcePath); errMsg != "" {
				return failValidation("Thư mục nguồn tài khoản không hợp lệ: " + errMsg)
			}
		}
	}

	// 2. Mail provider
	if interaction.VerifyEnabled {
		switch interaction.MailProvider {
		case "zeus-x":
			if interaction.ZeusXApiKey == "" {
				return failValidation("Thiếu cấu hình: API Key ZeusX chưa được nhập (Thiết lập chạy → Verify tài khoản)")
			}
		case "dongvanfb":
			if interaction.DvfbApiKey == "" {
				return failValidation("Thiếu cấu hình: API Key DongVanFB chưa được nhập (Thiết lập chạy → Verify tài khoản)")
			}
		case "store1s":
			if interaction.Store1sApiKey == "" {
				return failValidation("Thiếu cấu hình: API Key Store1s chưa được nhập (Thiết lập chạy → Verify tài khoản)")
			}
		case "mail30s":
			if interaction.Mail30sApiKey == "" {
				return failValidation("Thiếu cấu hình: API Key Mail30s chưa được nhập (Thiết lập chạy → Verify tài khoản)")
			}
		case "@tmpbox.net", "@i2b.vn", "mohmal", "moakt", "0", "mail1sec", "1",
			"tempmail-plus", "tempmail-lol", "temporary-mail.net", "mailtm",
			"dropmail", "guerrillamail", "spam4me", "inboxes", "dismail",
			"mailymg", "altmails", "onesecmail", "1secmail.com", "firetempmail",
			"fviainboxes", "byomde", "byom.de", "dinlaan", "cryptogmail",
			"buslink24", "boxmailstore", "boxmail.store", "mailermnx",
			"tempforward", "tempomintraccoon", "tempemail", "tempemail.co",
			// Providers mới thêm trong Sprint 1+2 — cũng auto-gen email, không cần MailList.
			"tmpinbox", "tenminutemail", "10minutemail",
			"tempmailto", "onesecemail", "tempmail100",
			"tempmailso", "tempmail.so", "priyoemail", "priyo",
			"tempmailorgpremium", "mailtempcom", "mail-temp.com":
			// OK — các temp mail provider, không cần MailList
		default:
			// custom mail list hoặc provider chưa wire
			if strings.TrimSpace(interaction.MailList) == "" {
				return failValidation("Thiếu cấu hình: Danh sách mail trống (Thiết lập chạy → Verify tài khoản)")
			}
		}
	}

	// 3. Thư mục lưu kết quả — validate nhẹ, KHÔNG fail nếu không tồn tại.
	// ResultFolderPath là primary (field "Result folder" chung cho cả reg và verify).
	// Fallback: cfgOverride.OutputPath → interaction.OutputPath (verify-specific cũ).
	//
	// Multi-instance safe: nếu user copy app sang máy khác có path khác (vd từ máy
	// C:\Users\PC\ → C:\Users\Admin\), path cũ trong config không tồn tại → auto-fallback
	// về defaultResultFolder() (cạnh exe hoặc ~/Documents) thay vì return error.
	outputPathToCheck := interaction.ResultFolderPath
	if outputPathToCheck == "" {
		outputPathToCheck = cfgOverride.OutputPath
	}
	if outputPathToCheck == "" {
		outputPathToCheck = interaction.OutputPath
	}
	if outputPathToCheck != "" {
		if errMsg := a.ValidatePath(outputPathToCheck); errMsg != "" {
			// Path không tồn tại/invalid → log warning, dùng defaultResultFolder fallback
			slog.Warn("ResultFolderPath invalid, dùng default fallback",
				"configured_path", outputPathToCheck, "reason", errMsg)
			// KHÔNG return error — logic sau sẽ dùng defaultResultFolder()
		}
	}

	// 4. Proxy — nếu chọn loại proxy thì phải có danh sách/key
	switch settings.General.IpProvider {
	case "proxy", "proxy_fixed":
		if strings.TrimSpace(activeProxyList(settings.Ip)) == "" {
			return failValidation("Thiếu cấu hình: Danh sách proxy trống (Proxy Settings)")
		}
	case "tinsoft":
		if strings.TrimSpace(settings.Ip.TinsoftKeys) == "" {
			return failValidation("Thiếu cấu hình: Key Tinsoft chưa nhập (Proxy Settings)")
		}
	case "shoplike":
		if strings.TrimSpace(settings.Ip.ShoplikeKeys) == "" {
			return failValidation("Thiếu cấu hình: Key ShopLike chưa nhập (Proxy Settings)")
		}
	case "netproxy":
		if strings.TrimSpace(settings.Ip.NetproxyKeys) == "" {
			return failValidation("Thiếu cấu hình: Key NetProxy chưa nhập (Proxy Settings)")
		}
	case "minproxy":
		if strings.TrimSpace(settings.Ip.MinproxyKeys) == "" {
			return failValidation("Thiếu cấu hình: Key MinProxy chưa nhập (Proxy Settings)")
		}
	}

	// Merge: InteractionConfig (verify threads) > frontend override > saved settings > defaults
	// Verify standalone dùng SplitVerifyThreads (đặt ở Section 2 - Verify); fallback regThreads.
	maxThreads := cfgOverride.MaxThreads
	if interaction.SplitVerifyThreads > 0 {
		maxThreads = interaction.SplitVerifyThreads
	} else if interaction.RegThreads > 0 {
		maxThreads = interaction.RegThreads
	} else if maxThreads == 0 {
		maxThreads = settings.General.ThreadRequest
	}
	// DelayMs giữa goroutine start: luôn = 0 để các luồng khởi động song song ngay lập tức
	// "Nghỉ giữa các request" (DelayRequest) là delay nội bộ mỗi thread — không dùng ở đây
	delayMs := 0

	// proxyManager — dùng shared instance (cùng pool với REG) để tránh duplicate API call
	// và đảm bảo cache ShopLike được chia sẻ: cùng key → cùng proxy trong 30s window.
	//
	// Lưu ý: biến này chỉ dùng cho config setup ban đầu (IsConfigured check).
	// Acquire thật sự gọi a.getSharedProxyManager() mỗi lần để realtime reload
	// khi user thêm/xóa proxy giữa chừng (port UX C#: không cần stop+start).
	proxyManager := a.getSharedProxyManager()

	// Sticky proxy per worker slot (port C# KeepIPSuccess).
	// Raw acquire: getSharedProxyManager().Acquire() + RenderSession.
	// Sticky wrap quyết định giữ proxy sau success hay release mỗi lần.
	verifySticky := proxy.NewStickyManager(interaction.KeepIPSuccess, func(ctx context.Context) (string, func(), error) {
		mgr := a.getSharedProxyManager()
		p, rel, err := mgr.Acquire(ctx)
		if err != nil {
			runtime.EventsEmit(a.ctx, "verify:status", map[string]interface{}{
				"accountId": 0, "uid": "system", "message": "[Proxy] Lỗi lấy proxy: " + err.Error(),
			})
			return "", nil, err
		}
		p = proxy.RenderSessionIfIsProxyServer(p)
		return p, rel, nil
	})
	defer verifySticky.ReleaseAll()

	// LEGACY (đã thay): trước dùng pickUA closure đọc từ textarea "UA iPhone List"
	// shared cho mọi verify API → không match fingerprint platform-specific.
	// Hiện tại verify dùng pickUAForVerifyPlatform(verifyPlatform, acc.UserAgent, cfg)
	// → pick UA từ pool tương ứng với từng API (S23 → Android pool, MFB → Chrome
	// Desktop pool, Token → iPhone pool, etc.).

	// poolCtx — kiểm soát loop mua/dispatch accounts mới
	poolCtx, poolCancel := context.WithCancel(a.ctx)
	ctx := poolCtx
	cancel := poolCancel

	// workerCtx — kiểm soát HTTP requests của worker
	// Stop sẽ cancel cả dispatch (poolCtx) và worker (workerCtx)
	workerCtx, workerCancel := context.WithCancel(a.ctx)

	// Pool path dùng cho GetNewDatrOnLive — tạo 1 lần cho cả run verify.
	// Tên file: datrNewVer{YYYYMMDD}.txt (cùng thư mục với Pool*.txt).
	verifyDatrPoolPath := ""
	if interaction.GetNewDatrOnLive {
		verifyDatrPoolPath = filepath.Join(defaultCookieDir(), "datrNewVer"+time.Now().Format("20060102")+".txt")
	}

	// Gán cancel functions trong verifyMu để StopVerify đọc/ghi đồng bộ.
	// Tránh data race: trước đây gán ngoài lock → StopVerify có thể đọc nil ngay sau khi RunVerify set isRunning=true.
	a.verifyMu.Lock()
	a.verifyCancel = poolCancel
	a.verifyWorkerCancel = workerCancel
	a.verifyMu.Unlock()

	// Tạo email pool 1 lần cho toàn bộ run — shared giữa tất cả goroutines
	sysNotify := func(msg string) {
		runtime.EventsEmit(a.ctx, "verify:status", map[string]interface{}{
			"accountId": 0, "uid": "system", "message": msg,
		})
	}
	// Close pool cũ trước khi gán pool mới — giải phóng credential slice,
	// tránh giữ token/email không còn dùng đến.
	if a.emailPool != nil {
		persistUsedUnused(a.emailPool) // dump used/unused run trước (nếu chưa) trước khi giải phóng
		a.emailPool.Close()
	}
	a.emailPool = nil
	// poolBatch = số email mua batch đầu, lấy từ cấu hình UI (MailPoolBatch).
	// Sau batch đầu, pool KHÔNG mua dư — mỗi account tự mua 1 con khi cần (xem CredPool.Get).
	poolBatch := interaction.MailPoolBatch
	if poolBatch < 1 {
		poolBatch = 50
	}
	switch interaction.MailProvider {
	case "zeus-x":
		accountCode := interaction.ZeusXAccountCode
		if accountCode == "" {
			accountCode = "HOTMAIL"
		}
		a.emailPool = emailrent.NewZeusXPool(interaction.ZeusXApiKey, accountCode, "", poolBatch, sysNotify)
	case "dongvanfb":
		accType := interaction.DvfbAccountType
		if accType == "" {
			accType = "1"
		}
		a.emailPool = emailrent.NewDongVanFBPool(interaction.DvfbApiKey, accType, "", poolBatch, sysNotify)
	case "store1s":
		productID := interaction.Store1sProductID
		if productID == "" {
			productID = "40559"
		}
		a.emailPool = emailrent.NewStore1sPool(interaction.Store1sApiKey, productID, "", poolBatch, sysNotify)
	case "mail30s":
		slug := interaction.Mail30sProductSlug
		if slug == "" {
			slug = "hotmail-oauth2"
		}
		a.emailPool = emailrent.NewMail30sPool(interaction.Mail30sApiKey, slug, "", poolBatch, sysNotify)
	}
	// Wire exhaustion callback — emit event + ghi log RÕ RÀNG khi email pool cạn.
	if a.emailPool != nil {
		a.emailPool.OnExhausted = func(err error) {
			runtime.EventsEmit(a.ctx, "email:pool-exhausted", map[string]interface{}{
				"provider": interaction.MailProvider,
				"error":    err.Error(),
			})
			sysNotify(fmt.Sprintf("⚠️ HẾT MAIL [%s]: %s — nạp tiền/đổi provider hoặc chờ có hàng lại",
				interaction.MailProvider, err.Error()))
		}
		// Lưu mail mua ra file để tái dùng sau (Config/RentMail/bought_<provider>.txt).
		provider := interaction.MailProvider
		a.emailPool.Provider = provider
		a.emailPool.OnBought = func(creds []emailrent.EmailCred) { appendBoughtMails(provider, creds) }
	}

	// Build verify config dùng chung cho cả 2 mode
	verifyCfg := instagram.VerifyConfig{
		UserApiLabel:  interaction.ApiVerifyPlatform, // log tag = tên API user chọn (vd "api token")
		VerifyEnabled: interaction.VerifyEnabled,
		MailProvider:  interaction.MailProvider,
		MailList:      interaction.MailList,
		CheckLiveDie:  interaction.CheckLiveDieEnabled,
		TimeDelayCheck: func() int {
			if interaction.DelayCheckLive > 0 {
				return interaction.DelayCheckLive
			}
			return interaction.TimeDelayCheck
		}(),
		TimeDelaySendCode: interaction.TimeDelaySendCode,
		DelayConfirmEmail: interaction.DelayConfirmEmail,
		DelayVeriReg:      interaction.DelayVeriReg,
		// WaitMail UI field là GIÂY (label "Wait mail (s)"), backend expect MILLISECONDS → * 1000.
		// Trước đây gán trực tiếp → poll mỗi 5ms thay vì 5000ms → verify poll mail 6000 lần!
		WaitMailMs:              interaction.WaitMail * 1000,
		MaxResend:               interaction.TrySendCode,
		SendAgainCode:           interaction.SendAgainCode,
		OutputPath:              interaction.OutputPath,
		UAIphoneList:            "",
		ZeusXApiKey:             interaction.ZeusXApiKey,
		ZeusXAccountCode:        interaction.ZeusXAccountCode,
		DvfbApiKey:              interaction.DvfbApiKey,
		DvfbAccountType:         interaction.DvfbAccountType,
		Store1sApiKey:           interaction.Store1sApiKey,
		Store1sProductID:        interaction.Store1sProductID,
		Mail30sApiKey:           interaction.Mail30sApiKey,
		Mail30sProductSlug:      interaction.Mail30sProductSlug,
		TempMailLolApiKey:       interaction.TempMailLolApiKey,
		TempMailDomain:          interaction.TempMailDomain,
		MuaMailApiKey:           interaction.MuaMailApiKey,
		MuaMailProductID:        interaction.MuaMailProductID,
		UnlimitMailApiKey:       interaction.UnlimitMailApiKey,
		UnlimitMailProductID:    interaction.UnlimitMailProductID,
		SptMailApiKey:           interaction.SptMailApiKey,
		SptMailServiceCode:      interaction.SptMailServiceCode,
		EmailAPIInfoApiKey:      interaction.EmailAPIInfoApiKey,
		EmailAPIInfoProductCode: interaction.EmailAPIInfoProductCode,
		OtpCheapApiKey:          interaction.OtpCheapApiKey,
		OtpCheapServiceID:       interaction.OtpCheapServiceID,
		ShopGmail9999ApiKey:     interaction.ShopGmail9999ApiKey,
		ShopGmail9999Service:    interaction.ShopGmail9999Service,
		RentGmailApiKey:         interaction.RentGmailApiKey,
		RentGmailPlatform:       interaction.RentGmailPlatform,
		OtpCodesSmsApiKey:       interaction.OtpCodesSmsApiKey,
		OtpCodesSmsServiceID:    interaction.OtpCodesSmsServiceID,
		WmemailApiKey:           interaction.WmemailApiKey,
		WmemailCommodity:        interaction.WmemailCommodity,
		PriyoEmailApiKey:        interaction.PriyoEmailApiKey,
		OTPHotmailPriority:      interaction.OTPHotmailPriority,
		TempMailToken:           interaction.TempMailToken,
		EmailPool:               a.emailPool,
		AddInfo: &instagram.AddInfoConfig{
			Enabled:      interaction.AddInfo,
			City:         interaction.AddInfoCity,
			Hometown:     interaction.AddInfoHometown,
			School:       interaction.AddInfoSchool,
			College:      interaction.AddInfoCollege,
			Work:         interaction.AddInfoWork,
			Relationship: interaction.AddInfoRelationship,
			DataDir:      interaction.AddInfoDataDir,
			DelayMs:      interaction.AddInfoDelayMs,
		},
	}
	// Force luôn dùng default folder (port C# FMain: hardcode ./result/ cạnh exe).
	// User KHÔNG cần cấu hình path — app tự quản lý như C# tool gốc.
	// interaction.ResultFolderPath / cfgOverride.OutputPath giữ lại trong DTO cho
	// backward-compat JSON nhưng bị ignore tại runtime.
	baseOutputPath := defaultResultFolder()
	runtime.EventsEmit(a.ctx, "verify:status", map[string]interface{}{
		"index": 0, "uid": "system", "proxy": "",
		"msg": fmt.Sprintf("[Result] %s", baseOutputPath),
	})

	// Tạo subfolder có timestamp: VerifyMfb_20260414_134459 (giống RegAndroid_... của reg)
	verifyPlatform := verifyPlatformFromType(interaction.ApiVerifyPlatform)
	verifyTag := "VerifyMfb"
	switch verifyPlatform {
	case instagram.PlatformS23:
		verifyTag = "VerifyS23"
	case instagram.PlatformAndroid:
		verifyTag = "VerifyAndroid"
	case instagram.PlatformWebAndroid:
		verifyTag = "VerifyWebAndroid"
	case instagram.PlatformS415:
		verifyTag = "VerifyS415"
	case instagram.PlatformS425:
		verifyTag = "VerifyS425"
	case instagram.PlatformS435:
		verifyTag = "VerifyS435"
	case instagram.PlatformS445:
		verifyTag = "VerifyS445"
	case instagram.PlatformS455:
		verifyTag = "VerifyS455"
	case instagram.PlatformS555:
		verifyTag = "VerifyS555"
	case instagram.PlatformS555V2:
		verifyTag = "VerifyS555V2"
	case instagram.PlatformS556:
		verifyTag = "VerifyS556"
	case instagram.PlatformS557:
		verifyTag = "VerifyS557"
	case instagram.PlatformS558:
		verifyTag = "VerifyS558"
	case instagram.PlatformS558V2:
		verifyTag = "VerifyS558V2"
	case instagram.PlatformS556V2:
		verifyTag = "VerifyS556V2"
	case instagram.PlatformS557V2:
		verifyTag = "VerifyS557V2"
	case instagram.PlatformS553V2:
		verifyTag = "VerifyS553V2"
	case instagram.PlatformS554V2:
		verifyTag = "VerifyS554V2"
	case instagram.PlatformS551V2:
		verifyTag = "VerifyS551V2"
	case instagram.PlatformS552V2:
		verifyTag = "VerifyS552V2"
	case instagram.PlatformS550V2:
		verifyTag = "VerifyS550V2"
	case instagram.PlatformS559:
		verifyTag = "VerifyS559"
	case instagram.PlatformS559V2:
		verifyTag = "VerifyS559V2"
	case instagram.PlatformS560:
		verifyTag = "VerifyS560"
	case instagram.PlatformS560V2:
		verifyTag = "VerifyS560V2"
	case instagram.PlatformS561:
		verifyTag = "VerifyS561"
	case instagram.PlatformS561V2:
		verifyTag = "VerifyS561V2"
	case instagram.PlatformS561V4S21:
		verifyTag = "VerifyS561V4S21"
	case instagram.PlatformS561V4S23:
		verifyTag = "VerifyS561V4S23"
	case instagram.PlatformS562:
		verifyTag = "VerifyS562"
	case instagram.PlatformS562V3:
		verifyTag = "VerifyS562V3"
	case instagram.PlatformS562V4S21:
		verifyTag = "VerifyS562V4S21"
	case instagram.PlatformS562V4S23:
		verifyTag = "VerifyS562V4S23"
	case instagram.PlatformS563:
		verifyTag = "VerifyS563"
	case instagram.PlatformS563V2:
		verifyTag = "VerifyS563V2"
	case instagram.PlatformS563S21:
		verifyTag = "VerifyS563S21"
	case instagram.PlatformS563V3S21:
		verifyTag = "VerifyS563V3S21"
	case instagram.PlatformS563V4S21:
		verifyTag = "VerifyS563V4S21"
	case instagram.PlatformS563V4S23:
		verifyTag = "VerifyS563V4S23"
	case instagram.PlatformS563V5S21:
		verifyTag = "VerifyS563V5S21"
	case instagram.PlatformS563V5S23:
		verifyTag = "VerifyS563V5S23"
	case instagram.PlatformS563V6S21:
		verifyTag = "VerifyS563V6S21"
	case instagram.PlatformS563V6S23:
		verifyTag = "VerifyS563V6S23"
	case instagram.PlatformS564V1S21:
		verifyTag = "VerifyS564V1S21"
	case instagram.PlatformS564V1S23:
		verifyTag = "VerifyS564V1S23"
	case instagram.PlatformS564V2S21:
		verifyTag = "VerifyS564V2S21"
	case instagram.PlatformS564V2S23:
		verifyTag = "VerifyS564V2S23"
	case instagram.PlatformS564V3S21:
		verifyTag = "VerifyS564V3S21"
	case instagram.PlatformS564V3S23:
		verifyTag = "VerifyS564V3S23"
	case instagram.PlatformS399:
		verifyTag = "VerifyS399"
	case instagram.PlatformIOSMess:
		verifyTag = "VerifyMessIos"
	}
	outputPath := filepath.Join(baseOutputPath, verifyTag+"_"+time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return failValidation("Không tạo được thư mục kết quả: " + err.Error())
	}
	a.resultPathMu.Lock()
	a.currentResultPath = outputPath
	a.resultPathMu.Unlock()
	// Reset dedup UIDs + counter — session mới, không kế thừa state cũ.
	a.ResetUploadSession()
	// Reset thống kê verify cho run mới (seed sẵn các version đã chọn).
	a.resetVerifyStats(verifyPlatformKeyList(interaction))
	a.resetMailDomainStats()
	a.resetBuildUAStats()
	// Thông báo path thực tế đang dùng cho frontend — nút Result mở đúng verify folder này.
	runtime.EventsEmit(a.ctx, "verify:output-path", map[string]interface{}{
		"path": outputPath,
	})

	// Capture cho OnAccountDone
	isCloneHVMode := isCloneHVActive(settings)
	sourceFolderPath := settings.General.AccountSourcePath
	// isStreamingFileMode = true khi chạy file mode mới (đọc stream từ thư mục)
	// → OnAccountDone không cố xóa file nguồn (đã xóa lúc pop)
	isStreamingFileMode := false

	// Writer ghi file kết quả chi tiết vào outputPath (port C# FMain.SaveFile/UpsertByUid).
	verifyWriter := resultpkg.NewWriter(outputPath)
	verifyInstance := verifyPlatformFromType(interaction.ApiVerifyPlatform)

	// Counter tracking: country/fb-version/locale success — auto-save mỗi 5s.
	// Port C# FMain.tmupdate_countrysuccess_Tick (Timer WinForms).
	verifyCounters := resultpkg.NewCounterSet(verifyWriter)
	verifyCounters.Start(a.ctx, 5*time.Second)
	// Dừng + flush lần cuối khi RunVerify kết thúc (defer ở cuối hàm).
	defer verifyCounters.Stop()

	runCfg := runner.RunConfig{
		MaxThreads:         maxThreads,
		DelayMs:            delayMs,
		DelayAfterResultMs: interaction.DelayDisplayResult * 1000, // giây → ms
		VerifyConfig:       &verifyCfg,
		VerifyPlatform:     verifyPlatformFromType(interaction.ApiVerifyPlatform),
		GetUseOriginalUA: func() bool {
			latest := a.LoadInteractionConfig()
			if uaCfg, ok := latest.VerifyPlatformUA[latest.ApiVerifyPlatform]; ok {
				return uaCfg.UseOriginalUA
			}
			return latest.UseOriginalUA
		},
		GetBuildUA: func() bool {
			latest := a.LoadInteractionConfig()
			if uaCfg, ok := latest.VerifyPlatformUA[latest.ApiVerifyPlatform]; ok {
				return uaCfg.BuildUA
			}
			return latest.BuildUA
		},
		GetAddVirtualSpec: func() bool {
			latest := a.LoadInteractionConfig()
			if uaCfg, ok := latest.VerifyPlatformUA[latest.ApiVerifyPlatform]; ok {
				return uaCfg.AddVirtualSpecAndroid
			}
			return latest.AddVirtualSpecAndroid
		},
		// GetVerifyUAForPlatform — FIX multi-version + UA Gốc: trả UA GỐC theo ĐÚNG platform
		// của từng account (sau round-robin), thay vì UA cố định của bản FOCUS.
		// Chỉ gọi khi UA Gốc bật (scheduler) → force UseOriginalUA=true để lấy UA gốc per-version.
		GetVerifyUAForPlatform: func(platform, country string) string {
			latest := a.LoadInteractionConfig()
			latest.UseOriginalUA = true
			// ReplaceCarrier theo per-platform config nếu có (giống pickUAForVerifyPlatform).
			if uaCfg, ok := latest.VerifyPlatformUA[platform]; ok {
				latest.ReplaceCarrier = uaCfg.ReplaceCarrier
			}
			return pickUAForVerifyPlatform(platform, "", latest, country)
		},
		// Realtime platform reload — user đổi API VERIFY mid-batch có hiệu lực.
		GetVerifyPlatform: func() string {
			return a.nextVerifyPlatform()
		},
		// GetVerifyConfig — đọc lại config mỗi khi goroutine bắt đầu → user thay đổi giữa chừng có hiệu lực ngay
		GetVerifyConfig: func() *instagram.VerifyConfig {
			latest := a.LoadInteractionConfig()
			latestOutputPath := cfgOverride.OutputPath
			if latestOutputPath == "" {
				latestOutputPath = latest.OutputPath
			}
			return &instagram.VerifyConfig{
				UserApiLabel:  latest.ApiVerifyPlatform,
				VerifyEnabled: latest.VerifyEnabled,
				MailProvider:  latest.MailProvider,
				MailList:      latest.MailList,
				CheckLiveDie:  latest.CheckLiveDieEnabled,
				TimeDelayCheck: func() int {
					if latest.DelayCheckLive > 0 {
						return latest.DelayCheckLive
					}
					return latest.TimeDelayCheck
				}(),
				TimeDelaySendCode: latest.TimeDelaySendCode,
				DelayConfirmEmail: latest.DelayConfirmEmail,
				DelayVeriReg:      latest.DelayVeriReg,
				WaitMailMs:        latest.WaitMail * 1000, // UI = giây → ms

				MaxResend:               latest.TrySendCode,
				SendAgainCode:           latest.SendAgainCode,
				OutputPath:              latestOutputPath,
				UAIphoneList:            "",
				ZeusXApiKey:             latest.ZeusXApiKey,
				ZeusXAccountCode:        latest.ZeusXAccountCode,
				DvfbApiKey:              latest.DvfbApiKey,
				DvfbAccountType:         latest.DvfbAccountType,
				Store1sApiKey:           latest.Store1sApiKey,
				Store1sProductID:        latest.Store1sProductID,
				Mail30sApiKey:           latest.Mail30sApiKey,
				Mail30sProductSlug:      latest.Mail30sProductSlug,
				TempMailLolApiKey:       latest.TempMailLolApiKey,
				TempMailDomain:          latest.TempMailDomain,
				MuaMailApiKey:           latest.MuaMailApiKey,
				MuaMailProductID:        latest.MuaMailProductID,
				UnlimitMailApiKey:       latest.UnlimitMailApiKey,
				UnlimitMailProductID:    latest.UnlimitMailProductID,
				SptMailApiKey:           latest.SptMailApiKey,
				SptMailServiceCode:      latest.SptMailServiceCode,
				EmailAPIInfoApiKey:      latest.EmailAPIInfoApiKey,
				EmailAPIInfoProductCode: latest.EmailAPIInfoProductCode,
				OtpCheapApiKey:          latest.OtpCheapApiKey,
				OtpCheapServiceID:       latest.OtpCheapServiceID,
				ShopGmail9999ApiKey:     latest.ShopGmail9999ApiKey,
				ShopGmail9999Service:    latest.ShopGmail9999Service,
				RentGmailApiKey:         latest.RentGmailApiKey,
				RentGmailPlatform:       latest.RentGmailPlatform,
				OtpCodesSmsApiKey:       latest.OtpCodesSmsApiKey,
				OtpCodesSmsServiceID:    latest.OtpCodesSmsServiceID,
				WmemailApiKey:           latest.WmemailApiKey,
				WmemailCommodity:        latest.WmemailCommodity,
				PriyoEmailApiKey:        latest.PriyoEmailApiKey,
				OTPHotmailPriority:      latest.OTPHotmailPriority,
				TempMailToken:           latest.TempMailToken,
				EmailPool:               a.emailPool, // shared pool — cùng instance suốt cả run
				DeepFakeLocale:          a.LoadSettings().General.DeepFakeInApi,
				ReUseEmail:              latest.ReUseEmail,
				UseEmailTime:            latest.UseEmailTime,
				FmUserTmpMail:           latest.FmUserTmpMail,
				UseProxyTempMail:        latest.UseProxyTempMail,
				UseProxyGmail:           latest.UseProxyGmail,
				Enable2FA:               latest.Enable2FA,
				AddInfo: &instagram.AddInfoConfig{
					Enabled:      latest.AddInfo,
					City:         latest.AddInfoCity,
					Hometown:     latest.AddInfoHometown,
					School:       latest.AddInfoSchool,
					College:      latest.AddInfoCollege,
					Work:         latest.AddInfoWork,
					Relationship: latest.AddInfoRelationship,
					DataDir:      latest.AddInfoDataDir,
					DelayMs:      latest.AddInfoDelayMs,
				},
			}
		},
		OutputPath: outputPath,
		// WorkerCtx = workerCtx → Stop sẽ cancel cả worker HTTP requests
		WorkerCtx:       workerCtx,
		RequireProxy:    proxyManager.IsConfigured(),
		AcquireProxy:    verifySticky.Acquire,
		AddMailRetry:    interaction.AddMailRetry,
		RetryUnknownNow: interaction.RetryUnknownNow,
		OnRawProxy: func(accountID int, proxyStr string) {
			// Hiển thị full proxy string (bao gồm user:pass + session token) để user theo dõi session rotation.
			runtime.EventsEmit(a.ctx, "verify:raw-proxy", map[string]interface{}{
				"accountId": accountID,
				"proxy":     proxyStr,
			})
		},
		OnProxy: func(accountID int, proxyStr string) {
			// Extract country từ proxy suffix (vd "1.2.3.4/id" → "id") + set vào a.accounts.
			// → saveVerifyOutcome đọc a.accounts[i].Location để ghi country vào file result.
			if idx := strings.LastIndex(proxyStr, "/"); idx >= 0 && idx < len(proxyStr)-1 {
				country := strings.ToUpper(proxyStr[idx+1:])
				a.accountsMu.Lock()
				for i := range a.accounts {
					if a.accounts[i].ID == accountID {
						a.accounts[i].Location = country
						break
					}
				}
				a.accountsMu.Unlock()
			}
			runtime.EventsEmit(a.ctx, "verify:proxy", map[string]interface{}{
				"accountId": accountID,
				"proxy":     proxyStr,
			})
		},
		// OnEmailCreated — emit verify:email ngay khi temp mail được tạo (trước verify done)
		// → UI hiện email vào cột EMAIL/PHONE realtime.
		OnEmailCreated: func(accountID int, email string) {
			// Update a.accounts in-place để grid lookup thấy email mới
			a.accountsMu.Lock()
			for i := range a.accounts {
				if a.accounts[i].ID == accountID {
					a.accounts[i].Email = email
					break
				}
			}
			a.accountsMu.Unlock()
			runtime.EventsEmit(a.ctx, "verify:email", map[string]interface{}{
				"accountId": accountID,
				"email":     email,
			})
		},
		// OnAccountDone — emit verify:account-done ngay khi account xong (realtime)
		OnAccountDone: func(accountID int, uid string, status string, message string, email string, userAgent string, twoFA string, token string, newCookie string, verifyPlatform string) {
			s := strings.ToLower(status)
			if s == "" {
				s = "unknown"
			}
			if verifyPlatform == "" {
				verifyPlatform = verifyInstance
			}
			a.recordVerifyOutcome(verifyPlatform, s == "live")
			a.recordBuildUAVerVersion(extractFBAV(userAgent), s == "live")
			if email != "" {
				a.recordMailDomainOutcome(email, s == "live")
			}
			// Learning loop: verify live qua PlatformWeb (api mfb) → thêm UA vào WebChrome pool.
			if s == "live" && verifyPlatform == instagram.PlatformWeb && userAgent != "" {
				if fakeinfo.AppendUAToPool(fakeinfo.UAKindWebChrome, userAgent) {
					slog.Info("WebChrome pool learned new UA", "ua_prefix", userAgent[:min(len(userAgent), 50)])
				}
			}
			a.accountsMu.Lock()
			var doneAcc Account
			for i := range a.accounts {
				if a.accounts[i].ID == accountID {
					a.accounts[i].Status = s
					a.accounts[i].Activity = message
					a.accounts[i].LastRun = time.Now().Format("2006/01/02 15:04")
					if email != "" {
						a.accounts[i].Email = email
					}
					if userAgent != "" {
						a.accounts[i].UserAgent = userAgent
					}
					if twoFA != "" {
						a.accounts[i].Twofa = twoFA
					}
					a.accounts[i].Token = preferUserAccessToken(a.accounts[i].Token, token)
					if newCookie != "" {
						a.accounts[i].Cookie = newCookie // cookie MỚI từ login verify → vào doneAcc + file
					}
					doneAcc = a.accounts[i]
					// Clear heavy fields sau khi đã copy sang doneAcc.
					// Giải phóng RAM cho file mode (1M+ account import) — chỉ giữ
					// các field cần hiển thị grid (UID, Status, Activity, Proxy, Email…).
					a.accounts[i].FullData = ""
					a.accounts[i].Cookie = doneAcc.Cookie // GIỮ cookie (cột hiển thị) — clear sẽ bị fetchAccounts refetch ra rỗng
					a.accounts[i].Token = doneAcc.Token   // GIỮ token (cột hiển thị) — clear sẽ bị fetchAccounts refetch ra rỗng
					if userAgent == "" {
						a.accounts[i].UserAgent = ""
					}
					a.accounts[i].NoteRun = ""
					a.accounts[i].SourceCode = ""
					break
				}
			}
			a.accountsMu.Unlock()

			runtime.EventsEmit(a.ctx, "verify:account-done", map[string]interface{}{
				"accountId": accountID,
				"uid":       uid,
				"status":    s,
				"message":   message,
				"token":     token,     // emit token PARAM (result.Token) — KHÔNG dùng doneAcc.Token
				"cookie":    newCookie, // (doneAcc có thể rỗng nếu row không có trong a.accounts, vd split slot)
			})

			// Xóa khỏi activityCache ngay — tránh batch timer ghi đè message cuối
			a.activityCache.Delete(accountID)

			// Ghi file kết quả chi tiết + tăng counters (port C# FMain.cs SaveFile/UpsertByUid + tracking timers).
			// Chạy async để không block flow callback.
			if doneAcc.UID != "" {
				go saveVerifyOutcome(verifyWriter, verifyCounters, s, message, doneAcc, verifyInstance)
			}

			if s == "live" {
				// Upload avatar sau verify live nếu được bật và có token OAuth
				if interaction.UploadAvatar && doneAcc.Token != "" {
					avatarDir := interaction.AvatarFolderPath
					if strings.TrimSpace(avatarDir) == "" {
						avatarDir = "Config/Avatar"
					}
					accIDForAvt := accountID
					proxyForAvt := doneAcc.Proxy
					tokenForAvt := doneAcc.Token
					uaForAvt := doneAcc.UserAgent
					uidForAvt := doneAcc.UID
					emitActivity := func(msg string) {
						runtime.EventsEmit(a.ctx, "verify:batch-status", []map[string]interface{}{
							{"accountId": accIDForAvt, "message": msg},
						})
					}
					go func() {
						defer func() {
							if r := recover(); r != nil {
								slog.Error("upload avatar panic recovered", "uid", uidForAvt, "panic", r)
							}
						}()
						emitActivity("[UpAVT] Đang upload avatar...")
						// Parent = workerCtx (run-scoped) — Stop run abort upload đang dở.
						// Avatar upload là post-processing, có thể mất khi Stop; trade-off
						// để goroutine không treo + giữ token/UA closure 60s sau Stop.
						avtCtx, avtCancel := context.WithTimeout(workerCtx, 60*time.Second)
						defer avtCancel()
						if err := uploadavatar.UploadAvatarS23(avtCtx, proxyForAvt, tokenForAvt, uaForAvt, avatarDir); err != nil {
							slog.Warn("UploadAvatar failed", "uid", uidForAvt, "err", err)
							emitActivity("[UpAVT] Lỗi: " + err.Error())
						} else {
							slog.Info("UploadAvatar OK", "uid", uidForAvt)
							emitActivity("[UpAVT] Upload avatar thành công ✓")
						}
					}()
				}

				// GetNewDatrOnLive — dùng token + cookie + UA của account vừa Live
				// gọi GraphQL profile-switcher để lấy datr mới → pool + pool file.
				if interaction.GetNewDatrOnLive && doneAcc.Token != "" && verifyDatrPoolPath != "" {
					uid4d := doneAcc.UID
					tok4d := doneAcc.Token
					coo4d := doneAcc.Cookie
					ua4d := doneAcc.UserAgent
					prx4d := doneAcc.Proxy
					accID4d := accountID
					go func() {
						dCtx, dCancel := context.WithTimeout(workerCtx, 20*time.Second)
						defer dCancel()
						datr := fetchNewDatrFromAccountUA(dCtx, uid4d, tok4d, coo4d, ua4d, prx4d)
						if datr == "" {
							slog.Warn("[GetDatrOnLive] GraphQL trả rỗng", "uid", uid4d)
							return
						}
						slog.Info("[GetDatrOnLive] datr mới", "uid", uid4d, "datr_prefix", safePrefix(datr, 6))
						cookie.AppendDatrToPool(verifyDatrPoolPath, datr)
						a.poolFileSaved.Add(1)
						if androidreg.SharedPool != nil {
							androidreg.SharedPool.AddDatrRawNoPersist(datr)
						}
						runtime.EventsEmit(a.ctx, "verify:batch-status", []map[string]interface{}{
							{"accountId": accID4d, "message": fmt.Sprintf("[GetDatr] datr mới: %s...", safePrefix(datr, 8))},
						})
					}()
				}

				// Auto-upload lên banclone.pro sau verify live — kiểm tra Ver.Enabled trong uploadsite config
				if uploadCfg := a.LoadUploadSiteConfig(); uploadCfg.Ver.Enabled && uploadCfg.Code != "" && uploadCfg.ApiKey != "" {
					country := ""
					if loc := extractFBLocaleFromUA(doneAcc.UserAgent); len(loc) >= 5 && loc[2] == '_' {
						country = loc[3:]
					}
					filter := "1"
					if !uploadCfg.FilterDuplicate {
						filter = "0"
					}
					var uploadLine string
					if strings.TrimSpace(doneAcc.Twofa) == "" {
						uploadLine = resultpkg.FormatReg(resultpkg.RegData{
							UID: doneAcc.UID, Password: doneAcc.Password,
							Cookie: doneAcc.Cookie, Token: doneAcc.Token,
							Email: "", Country: country, // user: không upload email
						}, nil)
					} else {
						uploadLine = resultpkg.FormatVerify(resultpkg.VerifyData{
							UID: doneAcc.UID, Password: doneAcc.Password,
							TwoFA: doneAcc.Twofa, Cookie: doneAcc.Cookie,
							Token: doneAcc.Token, Email: "", // user: không upload email
							Country: country,
						}, nil)
					}
					a.ensureUploadRunning(uploadCfg)
					a.enqueueForUpload(uploadLine)
					_ = filter // filter giờ đọc từ config trong runner
				}
			}

			// File mode (bulk cũ): xóa dòng khỏi file .txt nguồn trong background
			// Streaming mode đã pop+xóa trước khi chạy → bỏ qua ở đây
			if !isCloneHVMode && !isStreamingFileMode && doneAcc.UID != "" {
				if sourceFolderPath != "" && doneAcc.SourceCode != "" {
					filePath := filepath.Join(sourceFolderPath, doneAcc.SourceCode)
					fullData := doneAcc.FullData
					go func() {
						a.removeLineMu.Lock()
						defer a.removeLineMu.Unlock()
						a.removeAccountLine(filePath, fullData)
					}()
				}
			}

			// Selected-file mode (user pick 1 file .txt từ "Nguồn tài khoản → Từ 1 file"):
			// xóa dòng khỏi file gốc cho MỌI account đã verify xong (live/die/checkpoint/unknown).
			// Lý do: kết quả đã được lưu đầy đủ trong folder KetQua (Live.txt, Die.txt, Checkpoint.txt, Unknown.txt)
			// → file gốc trở thành "còn lại bao nhiêu chưa verify" → user thấy sạch, dễ quản lý.
			if doneAcc.UID != "" && doneAcc.FullData != "" {
				a.sourceFilePathMu.RLock()
				filePath := a.sourceFilePath
				a.sourceFilePathMu.RUnlock()
				if filePath != "" {
					fullData := doneAcc.FullData
					go func() {
						a.removeLineMu.Lock()
						defer a.removeLineMu.Unlock()
						a.removeAccountLine(filePath, fullData)
					}()
				}
			}
		},
	}
	// Reset activity cache cho run này
	a.activityCache.Range(func(k, _ any) bool { a.activityCache.Delete(k); return true })

	// Batch emitter: gom verify:status vào 1 event mỗi 100ms — tránh flood WebView IPC.
	// Chỉ gửi entry THỰC SỰ thay đổi kể từ lần gửi trước (dirty tracking).
	// 100ms cho phép hiển thị các bước trung gian; dirty tracking giảm payload ~80-95% khi idle.
	batchCtx, batchCancel := context.WithCancel(a.ctx)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("batch emitter panic recovered", "panic", r)
			}
		}()
		// Dynamic interval — 300ms khi visible (responsive UI), 2s khi hidden (tiết kiệm IPC)
		const intervalVisible = 300 * time.Millisecond
		const intervalHidden = 2 * time.Second
		ticker := time.NewTicker(intervalVisible)
		defer ticker.Stop()
		curInterval := intervalVisible
		// sentCache theo dõi message cuối đã gửi cho mỗi accountID.
		// Chỉ emit khi message thay đổi — bỏ qua nếu trùng.
		sentCache := make(map[int]string, 64)
		for {
			select {
			case <-ticker.C:
				// Adaptive interval: swap ticker khi visibility đổi
				want := intervalVisible
				if a.frontendHidden.Load() {
					want = intervalHidden
				}
				if want != curInterval {
					ticker.Reset(want)
					curInterval = want
				}
				var updates []map[string]interface{}
				a.activityCache.Range(func(k, v any) bool {
					id, ok1 := k.(int)
					msg, ok2 := v.(string)
					if ok1 && ok2 && sentCache[id] != msg {
						updates = append(updates, map[string]interface{}{
							"accountId": id,
							"message":   msg,
						})
						sentCache[id] = msg
					}
					return true
				})
				for id := range sentCache {
					if _, exists := a.activityCache.Load(id); !exists {
						delete(sentCache, id)
					}
				}
				if len(updates) > 0 {
					runtime.EventsEmit(a.ctx, "verify:batch-status", updates)
				}
			case <-batchCtx.Done():
				return
			}
		}
	}()

	onStatus := func(accountID int, uid string, message string) {
		// Lock-free: chỉ ghi vào activityCache — batch goroutine sẽ flush lên frontend
		a.activityCache.Store(accountID, message)
	}

	if isCloneHVActive(settings) {
		// === CHẾ ĐỘ CLONEHV POOL: Luôn duy trì maxThreads luồng song song ===
		// Con nào xong → mua bổ sung 1 con ngay lập tức → không bao giờ để luồng trống

		srcPath := a.GetAccountSourceFolder()

		// File session để append tất cả accounts mua trong lần chạy này
		sessionFilePath := ""
		if srcPath != "" {
			sessionFilePath = filepath.Join(srcPath, fmt.Sprintf("CloneHV_%s.txt", time.Now().Format("20060102_150405")))
		}

		// dateFolder = outputPath đã có timestamp subfolder (VerifyMfb_20260414_134459)
		// Tạo 1 lần cho toàn bộ run — tất cả account trong batch ghi vào cùng folder này
		dateFolder := outputPath

		// cloneNotify — system-level status message lên frontend
		cloneNotify := func(msg string) {
			runtime.EventsEmit(a.ctx, "verify:status", map[string]interface{}{
				"accountId": 0, "uid": "system", "message": msg,
			})
		}

		// removeLineFromFile — alias dùng trong CloneHV closure
		removeLineFromFile := a.removeAccountLine

		// Tạo N slot rows cố định trong a.accounts — bảng chỉ có maxThreads dòng
		// Mỗi slot sẽ được cập nhật in-place khi account mới được phân công
		a.accountsMu.Lock()
		a.accounts = make([]Account, 0, maxThreads)
		for i := 0; i < maxThreads; i++ {
			a.accounts = append(a.accounts, Account{
				ID:         i + 1,
				Status:     "waiting",
				SourceCode: fmt.Sprintf("Slot %d", i+1),
				Activity:   "Đang chờ...",
			})
		}
		slotBaseID := 1
		a.accountsMu.Unlock()
		runtime.EventsEmit(a.ctx, "verify:accounts-updated", nil)

		// === C# pattern port: Queue + SemaphoreSlim + N workers ===
		//
		// C# dùng ConcurrentQueue<string> _bResourceQueue + SemaphoreSlim _bResourceBuyLock(1,1):
		//   - N thread cùng dequeue
		//   - Queue rỗng → thread nào Wait(0) được lock sẽ bulk buy → enqueue → release
		//   - Thread khác đợi queue có data → dequeue
		//
		// Go tương đương:
		//   - accountsBuffer + bufferMu — ConcurrentQueue
		//   - buyLock (chan struct{} cap 1) — SemaphoreSlim(1,1) non-blocking
		//   - N goroutine workers chạy loop fetchOneAccount → process → lặp

		accountsBuffer := make([]string, 0, 128) // shared queue accounts đã mua
		var bufferMu sync.Mutex                  // lock cho accountsBuffer
		buyLock := make(chan struct{}, 1)        // semaphore 1 permit cho bulk buy
		var totalBoughtMu sync.Mutex             // lock cho totalBought counter
		var sessionFileMu sync.Mutex             // serialize ghi sessionFile từ N workers (tránh corrupt write trên Windows)

		// addAndBuildInput — ghi vào file + cập nhật slot row tương ứng in-place
		addAndBuildInput := func(raw string, num int, slotID int) (runner.AccountInput, bool) {
			// Ghi vào session file (giữ cho đến khi worker start) — serialize để tránh race
			// trên Windows nơi O_APPEND từ nhiều handles không atomic.
			if sessionFilePath != "" {
				sessionFileMu.Lock()
				if f, err := os.OpenFile(sessionFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600); err == nil {
					if _, werr := f.WriteString(raw + "\n"); werr != nil {
						slog.Warn("addAndBuildInput: ghi session file thất bại", "file", sessionFilePath, "err", werr)
					}
					if cerr := f.Close(); cerr != nil {
						slog.Warn("addAndBuildInput: close session file thất bại", "file", sessionFilePath, "err", cerr)
					}
				} else {
					slog.Warn("addAndBuildInput: mở session file thất bại", "file", sessionFilePath, "err", err)
				}
				sessionFileMu.Unlock()
			}

			acc := autoDetectAccount(raw)
			if acc.UID == "" {
				return runner.AccountInput{}, false
			}
			acc.ID = slotID
			acc.FullData = raw
			acc.Status = "new"
			acc.ImportTime = time.Now().Format("2006/01/02 15:04")
			acc.SourceCode = fmt.Sprintf("CloneHV #%d", num)

			// Multi-version: round-robin gán platform per-account NGAY ở precompute → UA hiển thị
			// khớp ĐÚNG version verify (trước đây dùng verifyPlatform focus cho mọi account → lệch).
			acctPlatform := a.nextVerifyPlatform()
			if acctPlatform == "" {
				acctPlatform = verifyPlatform
			}
			ua := pickUAForVerifyPlatform(acctPlatform, acc.UserAgent, interaction, phoneToCountryCode(acc.Phone))
			acc.UserAgent = ua // persist UA per-version vào account hiển thị (grid hiện đúng version)

			// Cập nhật slot row in-place — không append, bảng giữ nguyên N dòng
			a.accountsMu.Lock()
			for i := range a.accounts {
				if a.accounts[i].ID == slotID {
					a.accounts[i] = acc
					break
				}
			}
			a.accountsMu.Unlock()

			return runner.AccountInput{
				ID:       slotID,
				UID:      acc.UID,
				FullName: acc.FullName,
				Phone:    acc.Phone,
				Cookie:   acc.Cookie,
				Token:    acc.Token,
				UserAgent:    ua,
				Password:     acc.Password,
				InputAccount: raw,
				VerifyPlatform: acctPlatform, // scheduler dùng đúng platform này (không re-rotate)
				// TempMail reuse: forward email + creds nếu account đã reg
				// với Mode=TempMail. Verify steps Restore + skip CreateEmail.
				Email:                 acc.Email,
				EmailMeta:             acc.EmailMeta,
				Srnonce:               acc.Srnonce,
				SessionlessCryptedUID: acc.SessionlessCryptedUID,
			}, true
		}

		// (replenishCh cũ đã bỏ — C# pattern: worker tự fetchOneAccount trong vòng lặp,
		// không cần signal replenish từ ngoài)

		go func() {
			defer func() {
				a.verifyMu.Lock()
				a.isRunning = false
				a.verifyMu.Unlock()
				cancel()
				workerCancel()
				batchCancel()
				// Clear stopping CUỐI CÙNG sau khi đã cancel + workers exited via wg.Wait()
				// Start mới chỉ được phép sau khi resource đã release.
				a.verifyStopping.Store(false)
				runtime.EventsEmit(a.ctx, "verify:complete", "Đã dừng")
			}()

			var wg sync.WaitGroup
			totalBought := 0

			// === C# pattern port (FetchBulkAccountsFromBResource + worker dequeue) ===
			//
			// C# (FMain.cs L1236-1267):
			//   while running:
			//     ① TryDequeue queue
			//     ② nếu rỗng → Wait(0) lock → bulk buy 50 → enqueue → release
			//     ③ Parse data → run verify
			//
			// Go tương đương:
			//   takeFromBuffer — dequeue, trả ("", false) nếu rỗng
			//   tryBuyBulk     — non-blocking acquire buyLock, bulk buy, push vào buffer
			//   fetchOneAccount — loop: takeFromBuffer → nếu rỗng thì tryBuyBulk → retry

			// takeFromBuffer: dequeue 1 account từ buffer shared.
			takeFromBuffer := func() (string, bool) {
				bufferMu.Lock()
				defer bufferMu.Unlock()
				if len(accountsBuffer) == 0 {
					return "", false
				}
				raw := accountsBuffer[0]
				accountsBuffer = accountsBuffer[1:]
				return raw, true
			}

			// tryBuyBulk: non-blocking acquire buyLock + bulk buy + push vào buffer.
			// Trả số accounts đã mua (0 nếu không got lock, lỗi API, hoặc hết hàng).
			tryBuyBulk := func() int {
				// C# SemaphoreSlim.Wait(0) non-blocking → Go: select default
				select {
				case buyLock <- struct{}{}:
					defer func() { <-buyLock }()
				default:
					return 0 // thread khác đang mua — không đợi
				}

				// DoubleCheck queue sau khi got lock (có thể thread khác vừa enqueue xong)
				bufferMu.Lock()
				hasData := len(accountsBuffer) > 0
				bufferMu.Unlock()
				if hasData {
					return 0
				}

				// Read fresh credentials mỗi lần mua (realtime khi user đổi config)
				latestSett := a.LoadSettings()
				batchSize := latestSett.General.CloneHVAmount
				if batchSize <= 0 {
					batchSize = 50 // default match C# FetchBulkAccountsFromBResource(50)
				}

				cloneNotify(fmt.Sprintf("[CloneHV] Đang mua bulk %d tài khoản...", batchSize))
				rawList, err := clonehv.BuyAccounts(ctx,
					latestSett.General.CloneHVUsername,
					latestSett.General.CloneHVPassword,
					latestSett.General.CloneHVProductID,
					batchSize)
				if err != nil {
					cloneNotify(fmt.Sprintf("[CloneHV] Lỗi mua: %v", err))
					return 0
				}
				if len(rawList) == 0 {
					cloneNotify("[CloneHV] Hết tài khoản trong kho.")
					return 0
				}

				bufferMu.Lock()
				accountsBuffer = append(accountsBuffer, rawList...)
				bufferMu.Unlock()
				cloneNotify(fmt.Sprintf("[CloneHV] Đã mua %d accounts, push vào queue.", len(rawList)))
				return len(rawList)
			}

			// fetchOneAccount: worker loop — lấy 1 acc, buy bulk nếu queue rỗng, wait 2s retry.
			// Return (raw, true) nếu lấy được; ("", false) nếu ctx cancel.
			// Dùng time.NewTimer + Stop thay vì time.After để tránh timer leak — N=300 workers retry
			// đồng thời sẽ tạo 300 timers/2s nếu dùng time.After (timer chỉ GC khi fire).
			fetchOneAccount := func() (string, bool) {
				retryTimer := time.NewTimer(2 * time.Second)
				defer retryTimer.Stop()
				for ctx.Err() == nil {
					if raw, ok := takeFromBuffer(); ok {
						return raw, true
					}
					// Buffer rỗng → try bulk buy (non-blocking)
					tryBuyBulk()

					// Reset timer cho lần retry tiếp theo (idiomatic Go pattern)
					if !retryTimer.Stop() {
						select {
						case <-retryTimer.C:
						default:
						}
					}
					retryTimer.Reset(2 * time.Second)

					// Wait 2s rồi retry — C# Task.Delay(2000) khi queue rỗng
					select {
					case <-ctx.Done():
						return "", false
					case <-retryTimer.C:
					}
				}
				return "", false
			}

			// startWorkerLoop — worker chạy loop liên tục: fetch → process → fetch → ...
			// Slot ID cố định cho worker (không recycle như pattern cũ).
			// Worker exit khi ctx.Done() hoặc fetchOneAccount trả false.
			startWorkerLoop := func(slotID int) {
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() {
						if r := recover(); r != nil {
							slog.Error("CloneHV worker panic recovered — slot exits", "slot", slotID, "panic", r)
						}
					}()
					for ctx.Err() == nil {
						raw, ok := fetchOneAccount()
						if !ok {
							return
						}

						totalBoughtMu.Lock()
						totalBought++
						num := totalBought
						totalBoughtMu.Unlock()

						inp, ok := addAndBuildInput(raw, num, slotID)
						if !ok {
							// Parse fail → skip, lấy acc khác
							continue
						}
						// Slim event: chỉ update đúng 1 slot row, không trigger full fetchAccounts.
						// Frontend nhận → update in-place (uid/password/phone/status) + reset stale fields.
						runtime.EventsEmit(a.ctx, "verify:slot-assigned", map[string]interface{}{
							"slotId":    inp.ID,
							"uid":       inp.UID,
							"password":  inp.Password,
							"phone":     inp.Phone,
							"status":    "new",
							"userAgent": inp.UserAgent,
							"token":     inp.Token,
							"cookie":    inp.Cookie,
						})

						// Dùng workerCtx → Stop sẽ cancel HTTP request đang chạy thay vì
						// chờ HTTP timeout tự nhiên (30s-5p). Trade-off: account đang verify giữa
						// chừng có thể bị abort + mất kết quả. User accept để Stop responsive.
						// Pass slotID làm workerID → sticky manager phân biệt cache per slot.
						result := runner.RunOneAccountAt(workerCtx, inp, runCfg, dateFolder, slotID, onStatus)
						if sessionFilePath != "" {
							removeLineFromFile(sessionFilePath, inp.InputAccount)
						}
						// OnAccountDone callback đã ghi file kết quả + emit verify:account-done.
						// Nếu result.Status rỗng nghĩa callback chưa fire (panic, early return) → log để debug.
						if result.Status == "" {
							slog.Warn("RunOneAccount: empty status — OnAccountDone có thể không fire",
								"accountID", inp.ID, "uid", inp.UID, "msg", result.Message)
						}
					}
				}()
			}

			// === Khởi động N worker loops ===
			cloneNotify(fmt.Sprintf("[CloneHV] Khởi tạo pool %d luồng (mua bulk khi cần)...", maxThreads))
			for i := 0; i < maxThreads; i++ {
				startWorkerLoop(slotBaseID + i)
			}
			cloneNotify(fmt.Sprintf("[CloneHV] Pool %d luồng đang chạy — mua bulk theo demand (C# pattern)", maxThreads))

			// Đợi tất cả workers hoàn thành (ctx cancel → workers exit loop)
			<-ctx.Done()
			wg.Wait()
		}()

		return fmt.Sprintf("Đang chạy CloneHV pool mode — %d luồng song song liên tục...", maxThreads)
	}

	// === CHẾ ĐỘ SELECTED-FILE: accounts đã load vào a.accounts, chỉ chạy những con user tick ===
	// Trigger khi AccountSource="file" + có AccountIDs từ frontend.
	// Khác với folder streaming: không đọc từ folder mới, chạy đúng list IDs đã chọn.
	// File mode OR any selected-accounts mode: user đã tick specific accounts → chạy đúng list đó.
	// Bao quát cả case settings.AccountSource bị stale "folder" nhưng user đã load accounts + tick.
	if len(cfgOverride.AccountIDs) > 0 {
		// Build selected accounts slice (giữ order từ a.accounts, filter theo IDs)
		selectedIDs := make(map[int]bool, len(cfgOverride.AccountIDs))
		for _, id := range cfgOverride.AccountIDs {
			selectedIDs[id] = true
		}
		// Lock (KHÔNG RLock) vì ghi lại a.accounts[i].UserAgent = UA per-version (để grid hiển thị đúng).
		a.accountsMu.Lock()
		var targets []runner.AccountInput
		for i := range a.accounts {
			acc := a.accounts[i]
			if !selectedIDs[acc.ID] {
				continue
			}
			acctPlatform := a.nextVerifyPlatform()
			if acctPlatform == "" {
				acctPlatform = verifyPlatform
			}
			// UA chuẩn theo ĐÚNG version verify của account này (per-account round-robin).
			ua := pickUAForVerifyPlatform(acctPlatform, acc.UserAgent, interaction, phoneToCountryCode(acc.Phone))
			a.accounts[i].UserAgent = ua // persist → grid hiển thị đúng UA per-version (không stale 554)
			targets = append(targets, runner.AccountInput{
				ID:                    acc.ID,
				UID:                   acc.UID,
				FullName:              acc.FullName,
				Phone:                 acc.Phone,
				Cookie:                acc.Cookie,
				Token:                 acc.Token,
				UserAgent:             ua,
				Password:              acc.Password,
				InputAccount:          acc.FullData,
				VerifyPlatform:        acctPlatform,
				Email:                 acc.Email,
				EmailMeta:             acc.EmailMeta,
				Srnonce:               acc.Srnonce,
				SessionlessCryptedUID: acc.SessionlessCryptedUID,
			})
		}
		a.accountsMu.Unlock()
		// Refresh grid để cột UA hiển thị ngay UA per-version vừa ghi.
		runtime.EventsEmit(a.ctx, "verify:accounts-updated", nil)

		if len(targets) == 0 {
			batchCancel()
			a.verifyMu.Lock()
			a.isRunning = false
			a.verifyMu.Unlock()
			a.verifyStopping.Store(false)
			msg := "Không có account nào được tick chọn — hãy tick các account muốn verify trong grid"
			runtime.EventsEmit(a.ctx, "verify:complete", map[string]interface{}{"error": msg})
			return msg
		}

		fileNotify := func(msg string) {
			runtime.EventsEmit(a.ctx, "verify:status", map[string]interface{}{
				"accountId": 0, "uid": "system", "message": msg,
			})
		}
		fileNotify(fmt.Sprintf("[File] Chạy %d account được tick chọn (maxThreads=%d)", len(targets), maxThreads))

		// Run qua runner.RunVerify — scheduler dispatch song song theo maxThreads.
		go func() {
			defer func() {
				a.verifyMu.Lock()
				a.isRunning = false
				a.verifyMu.Unlock()
				cancel()
				workerCancel()
				batchCancel()
				a.verifyStopping.Store(false)
				runtime.EventsEmit(a.ctx, "verify:complete", map[string]interface{}{"total": len(targets)})
			}()
			runner.RunVerify(ctx, targets, runCfg, onStatus)
		}()

		return fmt.Sprintf("Đang chạy %d account đã chọn — %d luồng song song...", len(targets), maxThreads)
	}

	// === CHẾ ĐỘ FILE STREAMING: Đọc từng tài khoản từ thư mục, không load trước ===
	isStreamingFileMode = true
	folderPath := settings.General.AccountSourcePath
	if interaction.VerifySourceFolderPath != "" {
		folderPath = interaction.VerifySourceFolderPath
	}

	fileNotify := func(msg string) {
		runtime.EventsEmit(a.ctx, "verify:status", map[string]interface{}{
			"accountId": 0, "uid": "system", "message": msg,
		})
	}

	// Diagnostic: hiển thị source folder + file count để user biết đang đọc từ đâu.
	fileNotify(fmt.Sprintf("[File] Source folder: %s", folderPath))
	if matches, _ := filepath.Glob(filepath.Join(folderPath, "*.txt")); len(matches) > 0 {
		verifiable := 0
		for _, m := range matches {
			if isVerifiableAccountFile(filepath.Base(m)) {
				verifiable++
			}
		}
		fileNotify(fmt.Sprintf("[File] Tìm thấy %d file .txt (trong đó %d file verifiable)", len(matches), verifiable))
	} else {
		fileNotify("[File] KHÔNG tìm thấy file .txt nào trong folder!")
	}

	// Tạo N slot rows cố định trong a.accounts — giống CloneHV pool mode
	a.accountsMu.Lock()
	a.accounts = make([]Account, 0, maxThreads)
	for i := 0; i < maxThreads; i++ {
		a.accounts = append(a.accounts, Account{
			ID:       i + 1,
			Status:   "waiting",
			Activity: "Đang chờ...",
		})
	}
	slotBaseID := 1
	a.accountsMu.Unlock()
	runtime.EventsEmit(a.ctx, "verify:accounts-updated", nil)

	// freeSlots: slot ID rảnh — worker lấy trước khi chạy, trả lại khi xong
	freeSlots := make(chan int, maxThreads)
	for i := 0; i < maxThreads; i++ {
		freeSlots <- slotBaseID + i
	}

	totalRead := 0

	// replenishCh: mỗi khi 1 worker xong → signal đọc bổ sung 1 account
	replenishCh := make(chan struct{}, maxThreads)

	go func() {
		defer func() {
			a.verifyMu.Lock()
			a.isRunning = false
			a.verifyMu.Unlock()
			cancel()
			workerCancel()
			batchCancel()
			// Trả RAM về OS ngay sau khi batch xong — account heavy fields đã cleared,
			// GC + FreeOSMemory đảm bảo OS nhận lại memory thay vì Go giữ heap.
			goruntime.GC()
			debug.FreeOSMemory()
			a.verifyStopping.Store(false)
			runtime.EventsEmit(a.ctx, "verify:complete", "Đã hoàn thành")
		}()

		var wg sync.WaitGroup

		startWorker := func(inp runner.AccountInput, slotID int) {
			wg.Add(1)
			go func() {
				defer func() {
					wg.Done()
					freeSlots <- slotID
					if poolCtx.Err() == nil {
						replenishCh <- struct{}{}
					}
				}()
				// workerCtx — Stop cancel được HTTP request đang chạy (vs a.ctx app lifetime).
				// Pass slotID làm workerID → sticky manager phân biệt cache per slot.
				result := runner.RunOneAccountAt(workerCtx, inp, runCfg, outputPath, slotID, onStatus)
				// OnAccountDone callback đã ghi file + emit event. result.Status rỗng = callback không fire → log.
				if result.Status == "" {
					slog.Warn("RunOneAccount (streaming): empty status — OnAccountDone có thể không fire",
						"accountID", inp.ID, "uid", inp.UID, "msg", result.Message)
				}
			}()
		}

		// readAndStart đọc count accounts từ folder (pop + delete), parse, start workers
		readAndStart := func(count int) int {
			started := 0
			// Cap số lần invalid line liên tiếp để tránh infinite loop spin CPU 100%
			// nếu folder chứa toàn dòng không parse được. Sau N invalid liên tiếp → bỏ batch.
			const maxConsecutiveInvalid = 100
			consecutiveInvalid := 0
			for i := 0; i < count; i++ {
				if ctx.Err() != nil {
					break
				}
				line, _, popErr := a.popAccountFromFolder(folderPath)
				if popErr != nil {
					fileNotify(fmt.Sprintf("[File] Lỗi đọc thư mục: %v", popErr))
					break
				}
				if line == "" {
					break // hết tài khoản
				}
				slotID := <-freeSlots
				totalRead++
				acc := autoDetectAccount(line)
				if acc.UID == "" {
					freeSlots <- slotID
					fileNotify(fmt.Sprintf("[File] Dòng không hợp lệ (bỏ qua): %.40s", line))
					consecutiveInvalid++
					if consecutiveInvalid >= maxConsecutiveInvalid {
						fileNotify(fmt.Sprintf("[File] Bỏ batch — gặp %d dòng invalid liên tiếp, có thể file sai format", maxConsecutiveInvalid))
						break
					}
					i-- // thử đọc dòng tiếp theo
					continue
				}
				consecutiveInvalid = 0 // reset counter khi gặp dòng hợp lệ
				acc.ID = slotID
				acc.FullData = line
				acc.Status = "new"
				acc.ImportTime = time.Now().Format("2006/01/02 15:04")
				acc.SourceCode = fmt.Sprintf("File #%d", totalRead)

				// UA per-version (round-robin) — tính TRƯỚC khi ghi a.accounts để grid hiển thị đúng.
				acctPlatform := a.nextVerifyPlatform()
				if acctPlatform == "" {
					acctPlatform = verifyPlatform
				}
				ua := pickUAForVerifyPlatform(acctPlatform, acc.UserAgent, interaction, phoneToCountryCode(acc.Phone))
				acc.UserAgent = ua // persist UA per-version vào account hiển thị

				a.accountsMu.Lock()
				for j := range a.accounts {
					if a.accounts[j].ID == slotID {
						a.accounts[j] = acc
						break
					}
				}
				a.accountsMu.Unlock()

				startWorker(runner.AccountInput{
					ID:                    slotID,
					UID:                   acc.UID,
					FullName:              acc.FullName,
					Phone:                 acc.Phone,
					Cookie:                acc.Cookie,
					Token:                 acc.Token,
					UserAgent:             ua,
					Password:              acc.Password,
					InputAccount:          line,
					VerifyPlatform:        acctPlatform,
					Email:                 acc.Email,
					EmailMeta:             acc.EmailMeta,
					Srnonce:               acc.Srnonce,
					SessionlessCryptedUID: acc.SessionlessCryptedUID,
				}, slotID)
				started++
			}
			if started > 0 {
				runtime.EventsEmit(a.ctx, "verify:accounts-updated", nil)
			}
			return started
		}

		// Bước 1: Fill ban đầu — đọc maxThreads accounts
		fileNotify(fmt.Sprintf("[File] Khởi tạo pool %d luồng...", maxThreads))
		initialStarted := readAndStart(maxThreads)
		if initialStarted == 0 {
			fileNotify("[File] Không có tài khoản nào trong thư mục để chạy")
			return
		}
		fileNotify(fmt.Sprintf("[File] Pool khởi động %d luồng — đang chạy...", initialStarted))

		// Bước 2: Replenishment loop — mỗi khi worker xong → đọc thêm 1 account
		activeWorkers := initialStarted
		for {
			select {
			case <-ctx.Done():
				wg.Wait()
				return
			case <-replenishCh:
				activeWorkers--
				if ctx.Err() != nil {
					if activeWorkers == 0 {
						wg.Wait()
						return
					}
					continue
				}
				started := readAndStart(1)
				if started > 0 {
					activeWorkers++
				} else {
					// Hết tài khoản — đợi workers còn lại xong rồi thoát
					if activeWorkers == 0 {
						wg.Wait()
						return
					}
				}
			}
		}
	}()

	return fmt.Sprintf("Đang chạy file streaming — %d luồng song song...", maxThreads)
}
