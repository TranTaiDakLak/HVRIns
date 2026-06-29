// app_register.go — Luồng đăng ký tài khoản (RunRegister), tách từ app.go.
// Di chuyển nguyên hàm — KHÔNG sửa logic.
package app

import (
	"HVRIns/internal/cookie"
	emailrent "HVRIns/internal/email/rent"
	"HVRIns/internal/igcore"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	uploadavatar "HVRIns/internal/instagram/interaction/android"
	android "HVRIns/internal/instagram/register/android"
	"HVRIns/internal/instagram/register/igandroid"
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
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// RunRegister — chạy đăng ký tài khoản hàng loạt
// Đọc profile từ InteractionConfig.createCookieList
// Format mỗi dòng: FirstName|LastName|DD-MM-YYYY|Gender(1/2)|Phone|Password[|Proxy]
func (a *App) RunRegister(maxThreads int) string {
	// Ngăn chạy đồng thời register + verify (cả hai dùng emailPool)
	a.verifyMu.Lock()
	verifyRunning := a.isRunning
	a.verifyMu.Unlock()
	if verifyRunning {
		return "Verify đang chạy, vui lòng dừng trước khi chạy Register"
	}

	a.registerMu.Lock()
	// State machine gating — chỉ Start được khi state == idle.
	// running/stopping đều phải reject; KHÔNG cho Start lại cho tới khi spawner defer
	// đã wg.Wait + cleanup xong và transition state về idle.
	switch runState(a.registerState.Load()) {
	case runStateRunning:
		a.registerMu.Unlock()
		return "Đang chạy đăng ký, vui lòng dừng trước"
	case runStateStopping:
		a.registerMu.Unlock()
		return "Đang dừng run cũ — vui lòng chờ hoàn tất rồi start lại"
	}
	// Snapshot generation NGAY ĐẦU run — chỉ dùng cho identity check trong defer cleanup
	// (chống event/cleanup stale từ run cũ). KHÔNG dùng để gating Start (đã có state machine).
	a.registerGen++
	myGen := a.registerGen
	// LƯU Ý: state vẫn = idle xuyên suốt validation phase. Validation early-return
	// chỉ unlock + return → không cần reset state. Transition idle → running được
	// thực hiện sau khi mọi validation pass, NGAY trước khi spawn spawner goroutine.

	if maxThreads <= 0 {
		maxThreads = 1
	}

	// Chọn platform register
	interactionCfg := a.LoadInteractionConfig()
	regModeRotateStartedAt := time.Now()

	// Override maxThreads từ InteractionConfig.RegThreads — nguồn truth thread count
	// đã chuyển từ GeneralConfig sang đây để reg/verify tự cài luồng riêng.
	// Param maxThreads từ frontend chỉ là fallback nếu RegThreads chưa set.
	if interactionCfg.RegThreads > 0 {
		maxThreads = interactionCfg.RegThreads
	}

	// Multi-version reg: list các version user chọn. regPlatform = phần tử [0] đóng vai
	// "primary" — dùng cho tên folder kết quả, session pool (iOS/WebAndroid), validation.
	// Worker spawner gán per-slot round-robin từ list này (xem vòng lặp bên dưới).
	regPlatforms := regPlatformList(interactionCfg)
	regPlatform := regPlatforms[0]
	// Áp per-platform UA config nếu user đã cấu hình cho platform này.
	interactionCfg = applyRegPlatformUAConfig(interactionCfg, regPlatform)

	// Warm proxy manager một lần — pool provider được init sẵn để goroutine đầu tiên
	// không phải chờ lần khởi tạo cache đầu.
	settings := a.LoadSettings()
	_ = a.getRegProxyManager() // warm cache

	// proxyPool chỉ dùng cho pre-check health (provider proxy list tĩnh)
	var proxyPool []string
	regProv := settings.Ip.RegIpProvider
	if settings.Ip.UseVerifyProxyForReg || regProv == "" || regProv == "none" {
		regProv = settings.General.IpProvider
	}
	if regProv == "proxy" || regProv == "proxy_fixed" {
		regList := activeRegProxyList(settings.Ip)
		if settings.Ip.UseVerifyProxyForReg {
			regList = activeProxyList(settings.Ip)
		}
		for _, p := range splitLines(regList) {
			if p = strings.TrimSpace(p); p != "" {
				proxyPool = append(proxyPool, p)
			}
		}
	}

	// Validate format proxy (nhanh, < 10ms) + warning async health check.
	// KHÔNG block spawn workers: mỗi worker tự Acquire() proxy + retry nếu bad.
	// Tránh delay 2-10s pre-check khi user chỉ muốn chạy nhanh 100 luồng.
	if len(proxyPool) > 0 {
		// Validate format instant — chỉ check parse được host:port hay không
		var validProxies []string
		for _, p := range proxyPool {
			if proxy.ShortDisplay(p) != "" && strings.Contains(proxy.ShortDisplay(p), ":") {
				validProxies = append(validProxies, p)
			} else {
				slog.Warn("proxy format sai, bỏ qua", "proxy", p)
				runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
					"index": 0, "phone": "system", "proxy": "",
					"msg": fmt.Sprintf("[ProxyCheck] ⚠️ Format sai, bỏ qua: %.40s", p),
				})
			}
		}
		if len(validProxies) == 0 {
			a.registerMu.Unlock()
			runtime.EventsEmit(a.ctx, "register:complete", map[string]interface{}{
				"total": 0,
				"error": "Không có proxy format hợp lệ — kiểm tra format proxy host:port:user:pass",
			})
			return "Không có proxy format hợp lệ"
		}
		proxyPool = validProxies

		// Async health check — chạy background, chỉ để LOG warning cho proxy die.
		// Workers đã spawn + chạy song song → không block.
		go func(pool []string) {
			healthResults := proxy.CheckProxyHealth(a.ctx, pool, 8*time.Second)
			deadCount := 0
			for _, r := range healthResults {
				if !r.Healthy {
					deadCount++
					slog.Warn("async proxy health fail",
						"proxy", proxy.ShortDisplay(r.Proxy),
						"error", r.Error)
				}
			}
			if deadCount > 0 {
				runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
					"index": 0, "phone": "system", "proxy": "",
					"msg": fmt.Sprintf("[ProxyCheck] ⚠️ %d/%d proxy có thể die (background check)", deadCount, len(pool)),
				})
			}
		}(proxyPool)
	}

	interactionCfg = a.LoadInteractionConfig()
	// Recompute multi-version list từ config mới nhất (user có thể đã đổi giữa 2 lần load).
	regPlatforms = regPlatformList(interactionCfg)
	regPlatform = regPlatforms[0]
	// Reset bảng thống kê reg cho run mới — seed sẵn các version đã chọn.
	a.resetRegStats(regPlatforms)
	a.poolFileSaved.Store(0)
	// Reset thống kê verify (RunRegister có thể tạo verify outcome qua auto-verify / split mode).
	a.resetVerifyStats(verifyPlatformKeyList(interactionCfg))
	a.resetMailDomainStats()
	a.resetBuildUAStats()

	// Validation: pool UA mode — pool phải có dữ liệu trước khi bắt đầu.
	// BuildUA=false + UseOriginalUA=false → lấy UA từ Config/UserAgent/<pool>.txt.
	// File rỗng hoặc chưa tồn tại → UA sẽ không có → reg fail ngay → báo trước.
	// IG rebrand: adapter IG tự sinh UA/device riêng (igcore) → KHÔNG cần UA pool
	// của flow Facebook cũ. Bỏ gate chặn "UA pool rỗng" để reg IG chạy được.
	if false && !interactionCfg.BuildUA && !interactionCfg.UseOriginalUA {
		effectiveCfg := applyRegPlatformUAConfig(interactionCfg, regPlatform)
		kind := uaKindFromPoolKey(effectiveCfg.UaPoolKey)
		if fakeinfo.UAPoolSize(kind) == 0 {
			a.registerMu.Unlock()
			errMsg := fmt.Sprintf("⚠️ UA pool \"%s\" rỗng — kiểm tra %s rồi bấm Reload UA Pool",
				string(kind), fakeinfo.UAOverridePath(kind))
			runtime.EventsEmit(a.ctx, "register:complete", map[string]interface{}{
				"total": 0, "error": errMsg,
			})
			return errMsg
		}
	}

	// Force luôn dùng default folder ./result/ cạnh exe (port C# FMain hardcode path).
	// User không cần chọn thư mục — app tự quản lý như C# tool gốc.
	// Mỗi lần Start tạo subfolder riêng: RegAndroid_YYYYMMDD_HHMMSS.
	baseResultPath := defaultResultFolder()
	runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
		"index": 0, "phone": "system", "proxy": "",
		"msg": fmt.Sprintf("[Result] %s", baseResultPath),
	})
	outputPath := ""
	if baseResultPath != "" {
		// Format mới: result_<type>_<datetime> — user yêu cầu đặt tên theo loại reg.
		runTag := time.Now().Format("20060102_150405")
		platformTag := "web"
		switch regPlatform {
		case instagram.PlatformIGAndroid:
			platformTag = "igandroid"
		case instagram.PlatformIGIOSBloks:
			platformTag = "ig_ios_bloks"
		default:
			platformTag = regPlatform
		}
		// Multi-version: ghép tag tất cả version (vd "s560-s561-s562"); quá dài → "multiN".
		if len(regPlatforms) > 1 {
			parts := make([]string, 0, len(regPlatforms))
			for _, p := range regPlatforms {
				parts = append(parts, strings.ToLower(strings.TrimSpace(p)))
			}
			joined := strings.Join(parts, "-")
			if len(joined) > 60 {
				platformTag = fmt.Sprintf("multi%d", len(regPlatforms))
			} else {
				platformTag = joined
			}
		}
		outputPath = filepath.Join(baseResultPath, "result_"+platformTag+"_"+runTag)
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			outputPath = baseResultPath // fallback nếu không tạo được subfolder
		}
		a.resultPathMu.Lock()
		a.currentResultPath = outputPath
		a.resultPathMu.Unlock()
		// Reset dedup UIDs + counter — session mới, không kế thừa state cũ.
		a.ResetUploadSession()
	}
	// Thông báo path thực tế đang dùng cho frontend — nút Result mở đúng thư mục này
	runtime.EventsEmit(a.ctx, "register:output-path", map[string]interface{}{
		"path": outputPath,
	})

	// Writer + CounterSet dùng chung cho tất cả reg goroutines trong session này.
	// Tạo 1 lần ở đây, pass qua closure vào worker. Start ticker auto-save 5s.
	var regWriter *resultpkg.Writer
	var regCounters *resultpkg.CounterSet
	if outputPath != "" {
		regWriter = resultpkg.NewWriter(outputPath)
		regCounters = resultpkg.NewCounterSet(regWriter)
		regCounters.Start(a.ctx, 5*time.Second)
	}

	// TUT mode: parse datr từ cookie list (static pool)
	var tutDatrPool []string
	if interactionCfg.CreateType == "tut" {
		for _, line := range splitLines(interactionCfg.CreateCookieList) {
			if datr := extractDatrFromCookieLine(line); datr != "" {
				tutDatrPool = append(tutDatrPool, datr)
			}
		}
	}
	// Dynamic TUT pool — tự động thu datr từ các tài khoản tạo thành công
	var dynamicTutPool []string
	var dynamicTutMu sync.Mutex

	// Cookie initial pool — dùng cho TẤT CẢ platforms (C#: CookieInitial checkbox).
	// Android dùng SharedPool riêng, WebAndroid/iOS HTTP dùng cookieInitialPool chung.
	//
	// Limit default = 9 (match C# UI default). C# dùng 9 ngay cả khi checkbox Limit
	// unchecked (tính năng/quirk C#) → KHÔNG nên để 9999 (vô hạn) như trước, vì đó
	// khiến 1 datr bị tái dùng hàng nghìn lần → FB flag "over-reused device" → tỷ
	// lệ success thấp hơn C#.
	var cookieInitialInlineLines []string
	var cookieInitialFilePaths []string
	// Verify iOS → reg cookie-only (WebAndroid/Web) KHÔNG login Android lấy EAAAAU.
	// Login chuyển sang lúc verify (iOS login lấy EAAAAAY). Verify Android giữ nguyên.
	// Login/token-fetch phải ở VERIFY, không phải REG: khi sẽ chạy verify, verify tự
	// lấy token (Android-family qua /auth/login, iOS qua CAA login EAAAAAY) → reg
	// cookie-only KHÔNG login lúc reg. Chỉ reg-only (không verify) mới giữ login lúc reg.
	cookieInitialLimit := 9 // default match C# UI
	if interactionCfg.LimitCookieInitialCount > 0 {
		cookieInitialLimit = interactionCfg.LimitCookieInitialCount
	}
	_ = interactionCfg.LimitCookieInitial // checkbox ở C# không thực sự ảnh hưởng; giữ field để UI không gẫy

	// Load cookie initial từ (match C# FacebookLogoutSessionUtils.LoadMachineIdsFromFile):
	//   1. datr_pool.txt → feed vào _inFilePool (C# L138-155)
	//   2. File user chọn (CookieInitialFile) — nếu set
	//   3. Config/Cookie/cookie_initial.txt — default nếu (2) rỗng
	//   4. Textarea createCookieList — paste trực tiếp trên UI
	//   Cả 2-4 sau đó feed vào _inFilePool (C# L158-172).
	//
	// C# port đúng: cả datr.txt VÀ cookie_initial.txt đều load vào pool reg.
	// Mỗi datr có max_usage_order check → khi vượt giới hạn, không được chọn nữa.
	{
		_ = cookie.EnsureDir()
		initialPath := ""

		// Load file user chọn nếu có; rỗng thì dùng cookie_initial.txt mặc định cạnh app.
		if strings.TrimSpace(interactionCfg.CookieInitialFile) != "" {
			initialPath = resolveCookieInitialPath(interactionCfg.CookieInitialFile)
		} else {
			initialPath = defaultCookieInitialPath()
			cookie.SeedInitialIfMissing(initialPath)
		}
		cookieInitialFilePaths = append(cookieInitialFilePaths, initialPath)

		// 4. Textarea paste trực tiếp
		if interactionCfg.CreateCookieList != "" {
			cookieInitialInlineLines = splitLines(interactionCfg.CreateCookieList)
		}
	}

	// persistNewDatr ghi datr mới thu được từ cookie reg vào Pool file (Pool{YYYYMMDD}_{N}.txt).
	// Chỉ active khi SaveNewDatr=true — user tích checkbox "Add new pool".
	allPlatformPools := map[string]**android.PartitionedDatrPool{}

	runPoolPath := cookie.NewRunPoolPath(defaultCookieDir())
	queuePaths := append([]string{runPoolPath}, cookieInitialFilePaths...)
	datrFileQueue := cookie.NewDatrFileQueue(queuePaths, 1500*time.Millisecond)
	defer datrFileQueue.Close()

	// cookieInitialPool là con trỏ đến sharedCookiePool, được gán sau khi pool được khởi tạo.
	// Dùng để persistNewDatr có thể cộng datr mới vào pool hiện tại (cập nhật pool count trên UI).
	var cookieInitialPool *android.PartitionedDatrPool

	// persistNewDatr luôn được định nghĩa — đọc config realtime mỗi lần gọi
	// để user bật/tắt checkbox bất cứ lúc nào có hiệu lực ngay lần reg kế tiếp.
	persistNewDatr := func(datr string) {
		cfg := a.LoadInteractionConfig()
		if !cfg.SaveNewDatr {
			return
		}
		datr = strings.TrimSpace(datr)
		if datr == "" || strings.HasPrefix(datr, "_") || strings.HasPrefix(datr, "-") {
			return
		}
		cookie.AppendDatrToPool(runPoolPath, datr)
		a.poolFileSaved.Add(1)
		// Thêm vào sharedCookiePool để pool count trên UI cập nhật realtime.
		if cookieInitialPool != nil {
			cookieInitialPool.AddDatrRawNoPersist(datr)
		}
		for _, poolPtr := range allPlatformPools {
			if poolPtr != nil && *poolPtr != nil {
				(*poolPtr).AddDatrRawNoPersist(datr)
			}
		}
	}
	removePersistedDatr := func(datr string) {
		datrFileQueue.RemoveDatrEverywhere(datr)
	}

	// Init datr pool cho TẤT CẢ 5 platforms ngay từ đầu — cho phép user hot-swap
	// API REG giữa chừng batch mà không cần stop+start. Cost RAM ~1-3MB (5 × 200KB/pool).
	// Port UX mềm hơn C# — user có thể thử nghiệm platform khác nhau trong 1 batch.
	var removingDatr sync.Map
	removeDatrEverywhere := func(datr string) {
		datr = strings.TrimSpace(datr)
		if datr == "" {
			return
		}
		if _, loaded := removingDatr.LoadOrStore(datr, struct{}{}); loaded {
			return
		}
		defer removingDatr.Delete(datr)
		removePersistedDatr(datr)
		for _, poolPtr := range allPlatformPools {
			if poolPtr != nil && *poolPtr != nil {
				(*poolPtr).RemoveDatr(datr)
			}
		}
	}
	sharedCookiePool := android.NewPartitionedPool(cookieInitialLimit)
	cookieInitialPool = sharedCookiePool // cho persistNewDatr cập nhật pool count trên UI
	loadedCookieInitialCount := 0
	cookieMethod := strings.ToLower(strings.TrimSpace(interactionCfg.CookieInitialMethod))
	if cookieMethod == "new" {
		// "Tạo mới": sinh datr nội bộ thay vì đọc từ file. Số lượng = thread count ×
		// cookieInitialLimit × 2 (heuristic — đủ để mỗi worker có buffer, sau đó
		// pool tự sinh thêm khi cạn). Tối thiểu 64 để batch nhỏ vẫn có pool.
		genN := interactionCfg.RegThreads * cookieInitialLimit * 2
		if genN < 64 {
			genN = 64
		}
		loadedCookieInitialCount += sharedCookiePool.LoadGenerated(genN)
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": fmt.Sprintf("[CookiePool] Tạo mới %d datr (random, không từ file)", loadedCookieInitialCount),
		})
	} else {
		for _, path := range cookieInitialFilePaths {
			n, err := sharedCookiePool.LoadFromFile(path)
			loadedCookieInitialCount += n
			if err != nil {
				slog.Warn("load cookie initial failed", "path", path, "error", err)
				runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
					"index": 0, "phone": "system", "proxy": "",
					"msg": fmt.Sprintf("[CookiePool] Cannot read file %s: %v", path, err),
				})
			}
		}
		if len(cookieInitialInlineLines) > 0 {
			loadedCookieInitialCount += sharedCookiePool.LoadFromLines(cookieInitialInlineLines)
		}
	}
	sharedCookiePool.SetPersistHook(persistNewDatr)
	sharedCookiePool.SetRemoveHook(removeDatrEverywhere)
	sharedCookiePool.SetPersistOnlyNewDatr(interactionCfg.KeepDatrSuccess)
	if interactionCfg.DeleteDatrCheckpoint {
		// "Xóa khi đạt giới hạn": áp dụng cho CẢ 2 ngưỡng nếu user bật.
		// - LimitCheckpoint bật → đạt LimitCheckpointCount lần checkpoint → xóa
		// - LimitCookieInitial bật → đạt LimitCookieInitialCount lần dùng → xóa
		maxCheckpoint := interactionCfg.LimitCheckpointCount
		if maxCheckpoint <= 0 {
			maxCheckpoint = 1
		}
		sharedCookiePool.SetMaxCheckpoint(maxCheckpoint)
		if interactionCfg.LimitCookieInitial {
			sharedCookiePool.SetDeleteOnUsageLimit(true)
		}
	}
	if interactionCfg.LimitDatrAge && interactionCfg.LimitDatrAgeMinutes > 0 {
		sharedCookiePool.SetMaxAgeMinutes(interactionCfg.LimitDatrAgeMinutes)
		// Background ticker: quét pool định kỳ + remove datr quá tuổi.
		// Interval = max(30s, maxAge/4) — ít nhất quét 4 lần trong 1 chu kỳ tuổi
		// để dung sai expire không quá lớn.
		interval := time.Duration(interactionCfg.LimitDatrAgeMinutes) * time.Minute / 4
		if interval < 30*time.Second {
			interval = 30 * time.Second
		}
		go func(p *android.PartitionedDatrPool, every time.Duration) {
			t := time.NewTicker(every)
			defer t.Stop()
			for {
				select {
				case <-a.ctx.Done():
					return
				case <-t.C:
					if n := p.ExpireOldDatrs(); n > 0 {
						runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
							"index": 0, "phone": "system", "proxy": "",
							"msg": fmt.Sprintf("[CookiePool] Đã xóa %d datr quá %d phút", n, interactionCfg.LimitDatrAgeMinutes),
						})
					}
				}
			}
		}(sharedCookiePool, interval)
	}
	for _, poolPtr := range allPlatformPools {
		*poolPtr = sharedCookiePool
	}
	if loadedCookieInitialCount > 0 {
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": fmt.Sprintf("[CookiePool] %d datr sẵn sàng", loadedCookieInitialCount),
		})
	}

	if loadedCookieInitialCount > 0 {
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": fmt.Sprintf("[CookieInitial] Loaded %d cookie initial (limit %d/mỗi)", loadedCookieInitialCount, cookieInitialLimit),
		})
	}

	// ── IG device pool (mid/datr/ig_did) ─────────────────────────────────────
	// Tương tự datr-pool bên FB: harvest mid/ig_did từ account live → inject
	// trước reg mới → IG thấy "thiết bị có lịch sử" → trust score cao hơn.
	// maxUses: số lần 1 mid dùng trước khi cả pool recycle (default 9 như FB; 0 = vô hạn).
	// minSize: 0 = inject ngay khi pool có ≥1 device. Lấy thẳng từ config.
	devPoolMaxUses := interactionCfg.DevicePoolMaxUses
	// "Lượt dùng" (UI COOKIE INITIAL) điều khiển maxUses mid-pool (logic recycle giữ nguyên).
	if interactionCfg.LimitCookieInitial && interactionCfg.LimitCookieInitialCount > 0 {
		devPoolMaxUses = interactionCfg.LimitCookieInitialCount
	}
	devPoolMinSize := interactionCfg.DevicePoolMinSize
	maxUsesLabel := fmt.Sprintf("%d (recycle)", devPoolMaxUses)
	if devPoolMaxUses <= 0 {
		maxUsesLabel = "vô hạn"
	}

	igDeviceFile := defaultCookieDir() + "/ig_devices.txt"
	igDevicePool := igcore.NewDevicePool(igDeviceFile, devPoolMaxUses)
	igDevicePool.SetMinSize(devPoolMinSize)
	if sz := igDevicePool.Size(); sz > 0 {
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": fmt.Sprintf("[IGDevicePool] %d device aged sẵn sàng (mid/ig_did) — maxUses %s, minSize %d", sz, maxUsesLabel, devPoolMinSize),
		})
	}
	igcore.SharedDevicePool = igDevicePool
	// KHÔNG nil pool bằng defer ở đây: RunRegister return NGAY (spawner chạy nền) →
	// defer sẽ nil pool tức thì trong khi worker còn chạy → inject/harvest thấy nil.
	// Việc nil pool dời vào spawner sau wg.Wait() (xem cuối spawner goroutine).

	// ── IG Android device pool (datr/mid/ig_did) ──────────────────────────────
	igAndroidDeviceFile := defaultCookieDir() + "/ig_android_devices.txt"
	igAndroidDevicePool := igcore.NewDevicePool(igAndroidDeviceFile, devPoolMaxUses)
	igAndroidDevicePool.SetMinSize(devPoolMinSize)
	if sz := igAndroidDevicePool.Size(); sz > 0 {
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": fmt.Sprintf("[IGAndroidPool] %d device aged sẵn sàng", sz),
		})
	}
	igandroid.SharedAndroidDevicePool = igAndroidDevicePool
	// (nil pool dời vào spawner sau wg.Wait() — xem ghi chú ở iOS pool phía trên)

	// Pre-harvest mid nếu pool rỗng — chạy qe/sync để lấy mid thật trước batch.
	// Số mid harvest = max(threads*2, 20), tối đa 10 workers song song.
	if igAndroidDevicePool.Size() == 0 && interactionCfg.RegThreads > 0 {
		harvestN := interactionCfg.RegThreads * 2
		if harvestN < 20 {
			harvestN = 20
		}
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": fmt.Sprintf("[IGAndroidPool] Pool rỗng — pre-harvest %d mid qua qe/sync...", harvestN),
		})
		harvestProxy := ""
		if len(proxyPool) > 0 {
			harvestProxy = proxyPool[0]
		}
		go func(n int, proxy string, pool *igcore.DevicePool) {
			added := igandroid.PreHarvestPool(a.ctx, proxy, n, pool, func(msg string) {
				runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
					"index": 0, "phone": "system", "proxy": "", "msg": msg,
				})
			})
			runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
				"index": 0, "phone": "system", "proxy": "",
				"msg": fmt.Sprintf("[IGAndroidPool] Pre-harvest xong: %d mid đã lưu vào pool", added),
			})
		}(harvestN, harvestProxy, igAndroidDevicePool)
	}

	// Pre-harvest iOS: MỖI lần chạy nạp thêm 1 batch mid tươi (= RegThreads) để pool
	// LỚN DẦN + luôn có mid mới (tránh tái dùng mãi 1 tập + tích mid cháy — ta chưa có
	// expiry). Chạy nền, không chặn reg. mid tươi từ qe/sync (server-token).
	// KHÔNG cap pool (user: file nhẹ, cứ để lớn dần) — mỗi lần chạy nạp thêm RegThreads mid.
	if interactionCfg.RegThreads > 0 {
		cur := igDevicePool.Size()
		iosTarget := cur + interactionCfg.RegThreads
		if cur < iosTarget {
			harvestN := iosTarget - cur
			runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
				"index": 0, "phone": "system", "proxy": "",
				"msg": fmt.Sprintf("[IGDevicePool] Pool %d < %d — pre-harvest %d mid qua qe/sync...", cur, iosTarget, harvestN),
			})
			harvestProxy := ""
			if len(proxyPool) > 0 {
				harvestProxy = proxyPool[0]
			}
			go func(nh int, proxy string, pool *igcore.DevicePool) {
				added := igandroid.PreHarvestPool(a.ctx, proxy, nh, pool, func(msg string) {
					runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
						"index": 0, "phone": "system", "proxy": "", "msg": msg,
					})
				})
				runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
					"index": 0, "phone": "system", "proxy": "",
					"msg": fmt.Sprintf("[IGDevicePool] Pre-harvest xong: +%d mid (pool ~%d)", added, pool.Size()),
				})
			}(harvestN, harvestProxy, igDevicePool)
		}
	}

	// User yêu cầu: nếu CookieInitialMethod="file" mà không có datr nào → DỪNG HOÀN TOÀN.
	// Tránh reg fake không có datr → kết quả tệ. "Tạo mới"/"Ck" thì không cần check.
	// IG (ig_ios_bloks/ig_android) dùng MID-POOL riêng (SharedDevicePool/ig_devices.txt),
	// KHÔNG dùng datr file này → bỏ qua check cho IG. Tránh chặn nhầm khi copy app sang
	// folder mới chưa có datr (datr file là cơ chế FB, không cần cho IG).
	if !strings.HasPrefix(regPlatform, "ig_") &&
		strings.EqualFold(strings.TrimSpace(interactionCfg.CookieInitialMethod), "file") &&
		loadedCookieInitialCount == 0 && len(tutDatrPool) == 0 {
		a.registerMu.Unlock()
		errMsg := "Cookie Initial: phương thức 'Từ file' nhưng không có datr — vui lòng cung cấp file cookie hoặc đổi sang 'Tạo mới'"
		runtime.EventsEmit(a.ctx, "register:complete", map[string]interface{}{
			"total": 0,
			"error": errMsg,
		})
		if regCounters != nil {
			regCounters.Stop()
		}
		return errMsg
	}
	cookieInitialInlineLines = nil

	// Session pool ownership (Task 3) — `runResources` bundle giữ LOCAL ref của pool
	// thuộc run này. Cleanup chạy trong spawner defer SAU wg.Wait + nil global nếu vẫn
	// còn khớp (xem `runResources.Cleanup`). Workers vẫn đọc qua global để giữ tương
	// thích với code internal (`if SharedSessionPool != nil { ... Acquire() }`).
	//
	// runID = myGen — generation counter sẵn có, đủ phân biệt các run trong session.
	runIDLog := fmt.Sprintf("run#%d", myGen)
	runRes := newRunResources(runIDLog, regPlatforms...)
	runRes.publishGlobals() // assign global ref để workers thấy được pool của run này

	// dateFolder dùng chung cho tất cả auto-verify trong run này
	// Verify results lưu CÙNG thư mục reg (không tạo folder riêng)
	autoVerifyDateFolder := outputPath

	// Khởi tạo email pool cho auto-verify (nếu cần)
	if interactionCfg.VerifyEnabled {
		regSysNotify := func(msg string) {
			runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
				"index": 0, "phone": "system", "proxy": "",
				"msg": "[EmailPool] " + msg,
			})
		}
		// Close pool cũ trước khi gán pool mới — giải phóng credential slice cũ.
		if a.emailPool != nil {
			persistUsedUnused(a.emailPool) // dump used/unused run trước (nếu chưa) trước khi giải phóng
			a.emailPool.Close()
		}
		a.emailPool = nil
		// poolBatch = số email mua batch đầu, lấy từ cấu hình UI (MailPoolBatch).
		// Sau batch đầu pool KHÔNG mua dư — mỗi account tự mua 1 con khi cần (xem CredPool.Get).
		poolBatch := interactionCfg.MailPoolBatch
		if poolBatch < 1 {
			poolBatch = 50
		}
		switch interactionCfg.MailProvider {
		case "zeus-x":
			accountCode := interactionCfg.ZeusXAccountCode
			if accountCode == "" {
				accountCode = "HOTMAIL"
			}
			a.emailPool = emailrent.NewZeusXPool(interactionCfg.ZeusXApiKey, accountCode, "", poolBatch, regSysNotify)
		case "dongvanfb":
			accType := interactionCfg.DvfbAccountType
			if accType == "" {
				accType = "1"
			}
			a.emailPool = emailrent.NewDongVanFBPool(interactionCfg.DvfbApiKey, accType, "", poolBatch, regSysNotify)
		case "store1s":
			productID := interactionCfg.Store1sProductID
			if productID == "" {
				productID = "40559"
			}
			a.emailPool = emailrent.NewStore1sPool(interactionCfg.Store1sApiKey, productID, "", poolBatch, regSysNotify)
		case "mail30s":
			slug := interactionCfg.Mail30sProductSlug
			if slug == "" {
				slug = "hotmail-oauth2"
			}
			a.emailPool = emailrent.NewMail30sPool(interactionCfg.Mail30sApiKey, slug, "", poolBatch, regSysNotify)
		}
		// Wire exhaustion callback — emit event + ghi log RÕ RÀNG khi email pool cạn.
		if a.emailPool != nil {
			a.emailPool.OnExhausted = func(err error) {
				runtime.EventsEmit(a.ctx, "email:pool-exhausted", map[string]interface{}{
					"provider": interactionCfg.MailProvider,
					"error":    err.Error(),
				})
				// Ghi log rõ ràng vào status row "system" để user thấy lý do.
				regSysNotify(fmt.Sprintf("⚠️ HẾT MAIL [%s]: %s — nạp tiền/đổi provider hoặc chờ có hàng lại",
					interactionCfg.MailProvider, err.Error()))
			}
			// Lưu mail mua ra file để tái dùng sau (Config/RentMail/bought_<provider>.txt).
			provider := interactionCfg.MailProvider
			a.emailPool.Provider = provider
			a.emailPool.OnBought = func(creds []emailrent.EmailCred) { appendBoughtMails(provider, creds) }
		}
	}

	ctx, cancel := context.WithCancel(a.ctx)
	a.registerCancel = cancel
	// regWorkerCtx — context riêng cho auto-verify workers (không bị cancel khi Stop).
	// Stop chỉ cancel ctx (dispatch) → ngừng nhận acc mới; nhưng các acc đã đăng ký
	// thành công vẫn chạy hết auto-verify + live/die check trước khi báo xong.
	regWorkerCtx, regWorkerCancel := context.WithCancel(a.ctx)
	// State transition idle → running. Pair với việc gán registerCancel: từ điểm này,
	// IsRegisterRunning() sẽ trả true và mọi caller Start mới sẽ bị reject.
	// Spawner defer chịu trách nhiệm transition stopping → idle (hoặc running → idle nếu
	// run kết thúc tự nhiên — hiện tại không xảy ra vì RunRegister loop forever).
	a.registerState.Store(int32(runStateRunning))
	a.registerMu.Unlock()

	// AUTO-RESTART TIMER (REAL-TIME APPLY 2026-05-16): poll config mỗi 5s → user chỉnh
	// AutoRestartEnabled/AutoRestartMinutes giữa chừng có hiệu lực NGAY (không cần Stop/Run lại).
	// Trước đây capture value lúc Run → user chỉnh giữa chừng phải đợi đến lần restart kế tiếp.
	//
	// Behavior:
	//   - User tắt AutoRestart giữa chừng → timer skip, không trigger restart
	//   - User đổi minutes (vd 60 → 5) → check ngay so với startTime; nếu đã >= 5 phút → trigger ngay
	//   - User bật AutoRestart giữa chừng → bắt đầu đếm từ thời điểm hiện tại (NOT startTime)
	go func() {
		startTime := time.Now()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		var wasEnabled bool
		var enableTime time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cfg := a.LoadInteractionConfig()
				if !cfg.AutoRestartEnabled {
					wasEnabled = false
					continue
				}
				// User vừa bật AutoRestart giữa chừng → bắt đầu đếm từ thời điểm bật.
				if !wasEnabled {
					wasEnabled = true
					enableTime = time.Now()
				}
				minutes := cfg.AutoRestartMinutes
				if minutes <= 0 {
					minutes = 60
				}
				// Reference time: nếu AutoRestart bật từ đầu → startTime; nếu bật giữa chừng → enableTime.
				refTime := startTime
				if enableTime.After(startTime) {
					refTime = enableTime
				}
				elapsed := time.Since(refTime)
				if elapsed >= time.Duration(minutes)*time.Minute {
					// Hết giờ → trigger restart
					runtime.EventsEmit(a.ctx, "register:auto-restart-trigger", map[string]interface{}{
						"minutes": minutes,
					})
					runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
						"index": 0, "phone": "system", "proxy": "",
						"msg": fmt.Sprintf("⏰ AUTO-RESTART: hết %d phút — đang dừng để chạy lại từ đầu...", minutes),
					})
					a.registerMu.Lock()
					if a.registerCancel != nil && runState(a.registerState.Load()) == runStateRunning {
						a.registerState.Store(int32(runStateStopping))
						a.registerCancel()
					}
					a.registerMu.Unlock()
					regWorkerCancel()
					return
				}
			}
		}
	}()

	// freeSlotsReg giữ slot IDs còn trống (1..maxThreads).
	// Dùng channel có ID thay vì counting semaphore để đảm bảo mỗi goroutine nhận
	// 1 slot DUY NHẤT, EXCLUSIVE — tránh 2 goroutine cùng emit về threadIdx giống nhau
	// (xảy ra khi goroutine slot 2 xong trước → release sem → dispatcher spawn goroutine
	// mới với slotIdx=1 trong khi goroutine slot 1 vẫn đang chạy VER).
	freeSlotsReg := make(chan int, maxThreads)
	for _slotInit := 1; _slotInit <= maxThreads; _slotInit++ {
		freeSlotsReg <- _slotInit
	}

	// verifySem — semaphore giới hạn SỐ VERIFY chạy đồng thời trong Split mode.
	// REG chạy maxThreads (vd 100), nhưng verify inline (RunOneAccountAt) phải acquire
	// permit này trước → giới hạn verify ≤ SplitVerifyThreads (vd 50). Nil nếu không
	// split hoặc SplitVerifyThreads <= 0 (verify chạy bằng số reg threads như cũ).
	var verifySem chan struct{}
	if interactionCfg.SplitMode && interactionCfg.SplitVerifyThreads > 0 {
		verifySem = make(chan struct{}, interactionCfg.SplitVerifyThreads)
	}

	// ───────────────────────────────────────────────────────────────────────────
	// TRUE SPLIT MODE — REG pool và VERIFY pool độc lập, giao tiếp qua channel.
	//
	// Trước đây split mode chạy verify INLINE trong reg worker (coupled): reg worker
	// đăng ký xong → tự verify luôn → mới giải phóng reg slot. Hệ quả: reg bị verify
	// block, không chạy full tốc độ.
	//
	// True split: reg worker reg xong → đẩy account vào splitVerifyCh rồi RETURN ngay
	// (giải phóng reg slot → reg tiếp tục full tốc độ). Một POOL VERIFY riêng
	// (SplitVerifyThreads goroutine) đọc từ channel, chạy verify độc lập với verify
	// slot ID riêng (cho VER panel). REG dispatch xong → close(splitVerifyCh) →
	// VER workers drain hết queue rồi thoát.
	//
	// CHỈ active khi SplitMode && VerifyEnabled. Path SplitMode==false giữ NGUYÊN
	// verify inline cũ (không đổi).
	type splitVerifyJob struct {
		acc          runner.AccountInput  // account đã reg (UID, Token, Cookie, Password, Proxy, UA, DeviceID, Email...)
		prof         instagram.RegInput   // profile gốc reg (lấy thêm field nếu cần)
		displayProxy string               // IP CHẠY hiển thị (đã CheckIP)
		regResult    *instagram.RegResult // result reg (token, password fallback)
	}

	splitModeActive := interactionCfg.SplitMode && interactionCfg.VerifyEnabled
	splitVerThreads := interactionCfg.SplitVerifyThreads
	if splitVerThreads <= 0 {
		splitVerThreads = maxThreads
	}

	var splitVerifyCh chan splitVerifyJob
	var freeSlotsVer chan int
	var splitVerWg sync.WaitGroup
	// splitWorkerCtx — context riêng cho VER pool, parent a.ctx, KHÔNG bị cancel khi
	// StopRegister. Stop chỉ cancel ctx (dispatch) → ngừng nhận reg mới; VER workers
	// vẫn drain hết queue đã đẩy vào trước khi thoát. Cancel ở spawner defer SAU khi
	// splitVerWg.Wait() xong.
	var splitWorkerCtx context.Context
	var splitWorkerCancel context.CancelFunc

	if splitModeActive {
		// Buffer lớn (maxThreads*5, tối thiểu 500) → REG đẩy account vào queue thoải mái,
		// chạy vượt xa VER mà KHÔNG bị block (true split: REG không đợi VER).
		// VER drain dần. Chỉ khi backlog > buffer mới backpressure (hiếm).
		chBuf := maxThreads * 5
		if chBuf < 500 {
			chBuf = 500
		}
		splitVerifyCh = make(chan splitVerifyJob, chBuf)
		freeSlotsVer = make(chan int, splitVerThreads)
		for v := 1; v <= splitVerThreads; v++ {
			freeSlotsVer <- v
		}
		splitWorkerCtx, splitWorkerCancel = context.WithCancel(a.ctx)
	}

	// runSplitVerify — chạy verify cho 1 job trong TRUE SPLIT mode. Tách từ block verify
	// inline cũ (chỉ giữ nhánh SplitMode). Dùng verSlot cho VER panel events thay vì
	// reg threadIdx → VER panel có slot ID riêng, không đụng REG panel slot.
	// Capture (shared): a, regWriter, regCounters, autoVerifyDateFolder, splitWorkerCtx.
	// CHỈ gọi trong split mode (splitModeActive == true).
	runSplitVerify := func(verSlot int, job splitVerifyJob) {
		result := job.regResult
		prof := job.prof
		displayProxy := job.displayProxy
		// Reload config mỗi job — user đổi mail provider / delay giữa chừng có hiệu lực ngay.
		interactionCfg := a.LoadInteractionConfig()

		if result == nil || !result.Success || result.UID == "" || !interactionCfg.VerifyEnabled {
			return
		}
		// acc copy từ job — sẽ mutate acc.UserAgent sang verify-platform UA.
		acc := job.acc
		// Cập nhật acc.UserAgent sang verify-platform UA per-account (round-robin).
		{
			verifyUAConfig := applyVerifyPlatformUAConfig(a.LoadInteractionConfig())
			// FIX multi-version: round-robin platform per-account (trước đây dùng focus
			// ApiVerifyPlatform cho MỌI account → UA + verify đều là bản focus). Set
			// acc.VerifyPlatform để scheduler dùng đúng version này (không re-rotate lệch).
			vpStr := a.nextVerifyPlatform()
			if vpStr == "" {
				vpStr = verifyPlatformFromType(verifyUAConfig.ApiVerifyPlatform)
			}
			acc.VerifyPlatform = vpStr
			if vUA := pickUAForVerifyPlatform(vpStr, acc.UserAgent, verifyUAConfig, phoneToCountryCode(acc.Phone)); vUA != "" {
				acc.UserAgent = vUA
			}
		}
		// acc.ID = verSlot → RunOneAccountAt callbacks (nếu dùng accountID) khớp VER slot.
		acc.ID = verSlot

		// Delay trước verify (DelayVeriReg). Ctx = splitWorkerCtx (drain-scoped) → Stop
		// register KHÔNG cancel sleep này (VER vẫn drain xong queue).
		if interactionCfg.DelayVeriReg > 0 {
			runtime.EventsEmit(a.ctx, "verify:batch-status", []map[string]interface{}{{
				"accountId": verSlot,
				"message":   fmt.Sprintf("Chờ %ds trước khi verify...", interactionCfg.DelayVeriReg),
			}})
			select {
			case <-splitWorkerCtx.Done():
			case <-time.After(time.Duration(interactionCfg.DelayVeriReg) * time.Second):
			}
		}

		verifyCfg := &instagram.VerifyConfig{
			UserApiLabel:  interactionCfg.ApiVerifyPlatform,
			VerifyEnabled: true,
			MailProvider:  interactionCfg.MailProvider,
			MailList:      interactionCfg.MailList,
			CheckLiveDie:  interactionCfg.CheckLiveDieEnabled,
			TimeDelayCheck: func() int {
				if interactionCfg.DelayCheckLive > 0 {
					return interactionCfg.DelayCheckLive
				}
				return interactionCfg.TimeDelayCheck
			}(),
			TimeDelaySendCode:       interactionCfg.TimeDelaySendCode,
			DelayConfirmEmail:       interactionCfg.DelayConfirmEmail,
			DelayVeriReg:            interactionCfg.DelayVeriReg,
			WaitMailMs:              interactionCfg.WaitMail * 1000, // UI = giây → ms
			SendAgainCode:           interactionCfg.SendAgainCode,
			OutputPath:              interactionCfg.OutputPath,
			UAIphoneList:            "",
			ZeusXApiKey:             interactionCfg.ZeusXApiKey,
			ZeusXAccountCode:        interactionCfg.ZeusXAccountCode,
			DvfbApiKey:              interactionCfg.DvfbApiKey,
			DvfbAccountType:         interactionCfg.DvfbAccountType,
			Store1sApiKey:           interactionCfg.Store1sApiKey,
			Store1sProductID:        interactionCfg.Store1sProductID,
			Mail30sApiKey:           interactionCfg.Mail30sApiKey,
			Mail30sProductSlug:      interactionCfg.Mail30sProductSlug,
			TempMailLolApiKey:       interactionCfg.TempMailLolApiKey,
			TempMailDomain:          interactionCfg.TempMailDomain,
			MuaMailApiKey:           interactionCfg.MuaMailApiKey,
			MuaMailProductID:        interactionCfg.MuaMailProductID,
			UnlimitMailApiKey:       interactionCfg.UnlimitMailApiKey,
			UnlimitMailProductID:    interactionCfg.UnlimitMailProductID,
			SptMailApiKey:           interactionCfg.SptMailApiKey,
			SptMailServiceCode:      interactionCfg.SptMailServiceCode,
			EmailAPIInfoApiKey:      interactionCfg.EmailAPIInfoApiKey,
			EmailAPIInfoProductCode: interactionCfg.EmailAPIInfoProductCode,
			OtpCheapApiKey:          interactionCfg.OtpCheapApiKey,
			OtpCheapServiceID:       interactionCfg.OtpCheapServiceID,
			ShopGmail9999ApiKey:     interactionCfg.ShopGmail9999ApiKey,
			ShopGmail9999Service:    interactionCfg.ShopGmail9999Service,
			RentGmailApiKey:         interactionCfg.RentGmailApiKey,
			RentGmailPlatform:       interactionCfg.RentGmailPlatform,
			OtpCodesSmsApiKey:       interactionCfg.OtpCodesSmsApiKey,
			OtpCodesSmsServiceID:    interactionCfg.OtpCodesSmsServiceID,
			WmemailApiKey:           interactionCfg.WmemailApiKey,
			WmemailCommodity:        interactionCfg.WmemailCommodity,
			PriyoEmailApiKey:        interactionCfg.PriyoEmailApiKey,
			OTPHotmailPriority:      interactionCfg.OTPHotmailPriority,
			TempMailToken:           interactionCfg.TempMailToken,
			EmailPool:               a.emailPool,
			DeepFakeLocale:          a.LoadSettings().General.DeepFakeInApi,
			ReUseEmail:              interactionCfg.ReUseEmail,
			UseEmailTime:            interactionCfg.UseEmailTime,
			FmUserTmpMail:           interactionCfg.FmUserTmpMail,
			UseProxyTempMail:        interactionCfg.UseProxyTempMail,
			UseProxyGmail:           interactionCfg.UseProxyGmail,
			Enable2FA:               interactionCfg.Enable2FA,
			AddInfo: &instagram.AddInfoConfig{
				Enabled:      interactionCfg.AddInfo,
				City:         interactionCfg.AddInfoCity,
				Hometown:     interactionCfg.AddInfoHometown,
				School:       interactionCfg.AddInfoSchool,
				College:      interactionCfg.AddInfoCollege,
				Work:         interactionCfg.AddInfoWork,
				Relationship: interactionCfg.AddInfoRelationship,
				DataDir:      interactionCfg.AddInfoDataDir,
				DelayMs:      interactionCfg.AddInfoDelayMs,
			},
		}
		getLatestVerifyCfg := func() *instagram.VerifyConfig {
			latest := a.LoadInteractionConfig()
			return &instagram.VerifyConfig{
				UserApiLabel:  latest.ApiVerifyPlatform,
				VerifyEnabled: true,
				MailProvider:  latest.MailProvider,
				MailList:      latest.MailList,
				CheckLiveDie:  latest.CheckLiveDieEnabled,
				TimeDelayCheck: func() int {
					if latest.DelayCheckLive > 0 {
						return latest.DelayCheckLive
					}
					return latest.TimeDelayCheck
				}(),
				TimeDelaySendCode:       latest.TimeDelaySendCode,
				DelayConfirmEmail:       latest.DelayConfirmEmail,
				DelayVeriReg:            latest.DelayVeriReg,
				WaitMailMs:              latest.WaitMail * 1000,
				SendAgainCode:           latest.SendAgainCode,
				OutputPath:              latest.OutputPath,
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
				EmailPool:               a.emailPool,
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
					Relationship: latest.AddInfoRelationship,
					DataDir:      latest.AddInfoDataDir,
					DelayMs:      latest.AddInfoDelayMs,
				},
			}
		}
		runCfg := runner.RunConfig{
			VerifyConfig:    verifyCfg,
			VerifyPlatform:  verifyPlatformFromType(interactionCfg.ApiVerifyPlatform),
			GetVerifyConfig: getLatestVerifyCfg,
			GetVerifyPlatform: func() string {
				return a.nextVerifyPlatform()
			},
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
			AddMailRetry: interactionCfg.AddMailRetry,
			// OnEmailCreated → VER panel EMAIL column (verSlot row).
			OnEmailCreated: func(_ int, email string) {
				runtime.EventsEmit(a.ctx, "verify:email", map[string]interface{}{
					"accountId": verSlot,
					"email":     email,
				})
			},
		}
		verifyOnStatus := func(accountID int, uid string, msg string) {
			// VER panel: verify events đi vào row verSlot.
			runtime.EventsEmit(a.ctx, "verify:batch-status", []map[string]interface{}{{
				"accountId": verSlot,
				"message":   msg,
			}})
		}
		// VER panel: tạo row mới với verSlot (uid/password/phone/status/UA/token/cookie).
		runtime.EventsEmit(a.ctx, "verify:slot-assigned", map[string]interface{}{
			"slotId":    verSlot,
			"uid":       acc.UID,
			"password":  acc.Password,
			"phone":     acc.Phone,
			"status":    "new",
			"userAgent": acc.UserAgent,
			"token":     acc.Token,
			"cookie":    acc.Cookie,
		})
		runtime.EventsEmit(a.ctx, "verify:raw-proxy", map[string]interface{}{
			"accountId": verSlot,
			"proxy":     prof.Proxy,
		})
		if displayProxy != "" {
			runtime.EventsEmit(a.ctx, "verify:proxy", map[string]interface{}{
				"accountId": verSlot,
				"proxy":     displayProxy,
			})
		}
		if prof.Email != "" {
			runtime.EventsEmit(a.ctx, "verify:email", map[string]interface{}{
				"accountId": verSlot,
				"email":     prof.Email,
			})
		}
		// iOS Mess reg phone-only → acquire tempmail Ở VER (chỉ account create OK mới tốn mail,
		// khỏi phí cho account die ngay ở create). Set acc.Email/EmailMeta → add-mail thủ công +
		// RunVerify reuse mail (Restore) y như account TempMail thường.
		if mh := a.acquireIOSMessVerMail(splitWorkerCtx, *verifyCfg, prof.Proxy, acc.VerifyPlatform, acc.Email, func(m string) { verifyOnStatus(verSlot, acc.UID, m) }); mh != nil {
			acc.Email = mh.Email
			acc.EmailMeta = mh.Meta
			defer mh.Close()
		}
		// splitWorkerCtx — KHÔNG bị cancel khi Stop → acc đã reg success chạy hết verify
		// + live/die check trước khi báo xong. Pass verSlot làm workerID → sticky cache
		// per VER slot.
		vr := runner.RunOneAccountAt(splitWorkerCtx, acc, runCfg, autoVerifyDateFolder, verSlot, verifyOnStatus)
		verifyStatus := strings.ToLower(vr.Status)
		verifyMessage := vr.Message
		if verifyStatus != "" {
			vp := vr.VerifyPlatform
			if vp == "" {
				vp = verifyPlatformFromType(interactionCfg.ApiVerifyPlatform)
			}
			a.recordVerifyOutcome(vp, verifyStatus == "live")
			a.recordBuildUAVerVersion(extractFBAV(vr.UserAgent), verifyStatus == "live")
			if vr.Email != "" {
				a.recordMailDomainOutcome(vr.Email, verifyStatus == "live")
			}
			if verifyStatus == "live" && vp == instagram.PlatformWeb && vr.UserAgent != "" {
				if fakeinfo.AppendUAToPool(fakeinfo.UAKindWebChrome, vr.UserAgent) {
					slog.Info("WebChrome pool learned new UA (auto)", "ua_prefix", vr.UserAgent[:min(len(vr.UserAgent), 50)])
				}
			}
		}
		// VER panel: status cuối (live/die/...).
		runtime.EventsEmit(a.ctx, "verify:account-done", map[string]interface{}{
			"accountId": verSlot,
			"uid":       vr.UID,
			"status":    verifyStatus,
			"message":   verifyMessage,
			"token":     vr.Token,
			"cookie":    vr.Cookie,
		})

		// Ghi kết quả verify ra file (reuse regWriter, cùng outputPath).
		if regWriter != nil && vr.UID != "" {
			verifyAcc := Account{
				UID:      vr.UID,
				Password: result.Password,
				Twofa:    vr.TwoFA,
				Cookie:   result.Cookie,
				Token: func() string {
					if vr.Token != "" {
						return vr.Token
					}
					return result.AccessToken
				}(),
				Email:     vr.Email,
				FullName:  prof.FirstName + " " + prof.LastName,
				UserAgent: vr.UserAgent,
			}
			verifyInstance := verifyPlatformFromType(interactionCfg.ApiVerifyPlatform)
			go saveVerifyOutcome(regWriter, regCounters, verifyStatus, vr.Message, verifyAcc, verifyInstance)

			if verifyStatus == "live" {
				// Auto-upload lên site sau verify live.
				if uploadCfg := a.LoadUploadSiteConfig(); uploadCfg.Ver.Enabled && uploadCfg.Code != "" && uploadCfg.ApiKey != "" {
					country := ""
					if loc := extractFBLocaleFromUA(verifyAcc.UserAgent); len(loc) >= 5 && loc[2] == '_' {
						country = loc[3:]
					}
					var uploadLine string
					if strings.TrimSpace(verifyAcc.Twofa) == "" {
						uploadLine = resultpkg.FormatReg(resultpkg.RegData{
							UID: verifyAcc.UID, Password: verifyAcc.Password,
							Cookie: verifyAcc.Cookie, Token: verifyAcc.Token,
							Email: "", Country: country,
						}, nil)
					} else {
						uploadLine = resultpkg.FormatVerify(resultpkg.VerifyData{
							UID: verifyAcc.UID, Password: verifyAcc.Password,
							TwoFA: verifyAcc.Twofa, Cookie: verifyAcc.Cookie,
							Token: verifyAcc.Token, Email: "",
							Country: country,
						}, nil)
					}
					a.ensureUploadRunning(uploadCfg)
					a.enqueueForUpload(uploadLine)
				}
			}
		}
	}

	// Spawn VER worker pool — SplitVerifyThreads goroutine đọc từ splitVerifyCh.
	// Mỗi job acquire 1 verSlot UNIQUE từ freeSlotsVer (cho VER panel), chạy verify,
	// trả slot về. Channel close → range thoát → splitVerWg.Done.
	if splitModeActive {
		for v := 0; v < splitVerThreads; v++ {
			splitVerWg.Add(1)
			go func() {
				defer splitVerWg.Done()
				defer func() {
					if r := recover(); r != nil {
						slog.Error("VER worker panic", "recovered", r, "stack", string(debug.Stack()))
					}
				}()
				for job := range splitVerifyCh {
					verSlot := <-freeSlotsVer
					func() {
						// Recover per-job → 1 job panic không giết VER worker (slot vẫn trả về).
						defer func() {
							if r := recover(); r != nil {
								slog.Error("VER job panic", "verSlot", verSlot, "recovered", r,
									"stack", string(debug.Stack()))
								runtime.EventsEmit(a.ctx, "verify:account-done", map[string]interface{}{
									"accountId": verSlot,
									"status":    "unknown",
									"message":   fmt.Sprintf("[PANIC verify] %v", r),
								})
							}
						}()
						runSplitVerify(verSlot, job)
					}()
					freeSlotsVer <- verSlot
				}
			}()
		}
	}

	var wg sync.WaitGroup
	var counter int64 // tổng tài khoản đã bắt đầu tạo

	// Batch register:status — gom updates vào 1 event/200ms (nhanh hơn 500ms để responsive).
	type regEntry struct {
		index                        int
		phone, proxy, userAgent, msg string
	}
	var regBatchCache sync.Map // key=threadIdx → regEntry

	regBatchCtx, regBatchCancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		// sentCache: key=threadIdx → hash của entry đã gửi lần cuối.
		// Chỉ emit khi msg thực sự thay đổi — tránh gửi 300 entries mỗi 200ms khi idle.
		type sentEntry struct{ phone, proxy, msg string }
		sentCache := make(map[int]sentEntry, 64)
		for {
			select {
			case <-ticker.C:
				var updates []map[string]interface{}
				regBatchCache.Range(func(k, v any) bool {
					e := v.(regEntry)
					prev := sentCache[e.index]
					if prev.msg == e.msg && prev.phone == e.phone && prev.proxy == e.proxy {
						return true // không thay đổi → skip
					}
					updates = append(updates, map[string]interface{}{
						"index": e.index, "phone": e.phone, "proxy": e.proxy, "userAgent": e.userAgent, "msg": e.msg,
					})
					sentCache[e.index] = sentEntry{phone: e.phone, proxy: e.proxy, msg: e.msg}
					return true
				})
				// Xóa entries đã hoàn thành khỏi sentCache (slot không còn trong regBatchCache)
				if len(sentCache) > 512 {
					for idx := range sentCache {
						if _, exists := regBatchCache.Load(idx); !exists {
							delete(sentCache, idx)
						}
					}
				}
				if len(updates) > 0 {
					runtime.EventsEmit(a.ctx, "register:batch-status", updates)
				}
			case <-regBatchCtx.Done():
				return
			}
		}
	}()

	runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
		"index": 0, "phone": "system", "proxy": "",
		"msg": fmt.Sprintf("Bắt đầu đăng ký liên tục với %d luồng song song...", maxThreads),
	})

	// [Diag] Theo dõi goroutine mỗi 30s — phân biệt leak (số leo) vs tải nền (số phẳng).
	// Dừng theo batch ctx. Nếu goroutine ổn định quanh ~luồng+hằng số → leak đã hết.
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				ok, se, ig, ns := igandroid.RegErrorStats()
				total := ok + se + ig + ns
				pct := func(n int64) int {
					if total == 0 {
						return 0
					}
					return int(n * 100 / total)
				}
				line := fmt.Sprintf("[RegStats] OK=%d(%d%%) system_error=%d(%d%%) integrity=%d(%d%%) no_session=%d | goroutines=%d",
					ok, pct(ok), se, pct(se), ig, pct(ig), ns, goruntime.NumGoroutine())
				runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
					"index": 0, "phone": "system", "proxy": "", "msg": line,
				})
				// Ghi ra file để đọc dễ (không cần tìm trong UI log view).
				_ = os.WriteFile(filepath.Join(AppDataDir(), "reg_stats.txt"), []byte(line+"\n"), 0644)
			}
		}
	}()
	if len(regPlatforms) > 1 {
		msg := fmt.Sprintf("[MultiReg] %d phiên bản: %s — mỗi luồng cố định 1 phiên bản (chia đều theo slot)", len(regPlatforms), strings.Join(regPlatforms, ", "))
		if len(regPlatforms) > maxThreads {
			msg += fmt.Sprintf(" ⚠️ chỉ %d phiên bản đầu được chạy (cần ≥%d luồng để chạy hết)", maxThreads, len(regPlatforms))
		}
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": msg,
		})
	}

	// Sticky proxy per slot cho reg — port C# KeepIPSuccess.
	// Success → giữ proxy cho reg kế (cùng IP, FB trust hơn).
	// Fail → thả proxy về pool → lấy fresh cho reg kế.
	regSticky := proxy.NewStickyManager(interactionCfg.KeepIPSuccess, func(c context.Context) (string, func(), error) {
		mgr := a.getRegProxyManager()
		if !mgr.IsConfigured() {
			return "", func() {}, fmt.Errorf("proxy pool chưa cấu hình")
		}
		return mgr.Acquire(c)
	})

	// Per-slot UA cache cho KeepUASuccess — success: pin UA cho acc kế, fail: UA mới.
	type regSlotUA struct {
		Platform string
		UA       string
	}
	var regUABySlot sync.Map
	// Per-slot datr cache cho KeepDatrSuccess — success: pin datr mới, fail: lấy datr khác từ pool.
	var regDatrBySlot sync.Map
	// Per-slot session cache — MỖI LUỒNG DÙNG CHUNG 1 IP SESSION xuyên suốt các account.
	// Lý do: nếu render session MỚI mỗi account, proxy phải cấp exit IP + TCP connection
	// mới mỗi lần → khi 20+ luồng burst CheckIP đồng thời, gateway proxy quá tải → CheckIP
	// rớt (đo thực tế: session-mới ~50% hit @20 luồng, session-chung 100% hit, nhanh gấp 3).
	// Reuse session per slot → proxy giữ 1 upstream/luồng, transport keep-alive reuse.
	// Cache theo base proxy: nếu base đổi (pool rotate) → render session mới.
	type regSlotSession struct{ base, rendered string }
	var regSessionBySlot sync.Map // slotIdx → regSlotSession

	// regToVerCh: DEPRECATED — channel để forward acc sang split-verify pool riêng.
	// 2026-05-15: Split Mode đã đổi thành PURE UI option (chỉ hiển thị 2 panel REG/VER
	// để dễ nhìn) — worker logic GIỐNG Normal Mode (1 worker = REG + VER inline).
	// → Giữ var = nil để các block check `regToVerCh != nil` skip hết.
	var regToVerCh chan string

	// Spawner — chạy vô hạn cho đến khi ctx bị huỷ
	go func() {
		// Recover panic ở spawner level — nếu panic sẽ log ra console + emit system event
		// để user thấy lỗi thay vì "Đang chạy" mãi với 0 luồng.
		defer func() {
			if r := recover(); r != nil {
				slog.Error("REG spawner panic", "recovered", r)
				runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
					"index": 0, "phone": "system", "proxy": "",
					"msg": fmt.Sprintf("[PANIC spawner] %v", r),
				})
			}
			wg.Wait()
			// Reg workers đã xong → giờ mới nil device pool (KHÔNG nil bằng defer trong
			// RunRegister vì hàm đó return ngay khi spawner còn chạy → sẽ nil quá sớm).
			igcore.SharedDevicePool = nil
			igandroid.SharedAndroidDevicePool = nil
			// TRUE SPLIT drain — reg workers đã xong (wg.Wait) → KHÔNG còn job đẩy vào
			// splitVerifyCh. close() báo VER workers hết job → range thoát sau khi drain
			// hết queue còn lại. splitVerWg.Wait() chờ toàn bộ VER xong (vẫn dùng
			// splitWorkerCtx CHƯA cancel → verify hiện tại chạy hết). Phải drain TRƯỚC
			// regCounters.Stop() vì runSplitVerify ghi qua regCounters (saveVerifyOutcome).
			if splitModeActive {
				close(splitVerifyCh)
				splitVerWg.Wait()
				splitWorkerCancel() // VER đã drain xong → giải phóng splitWorkerCtx
			}
			// Mọi verify đã xong → phân loại mail pool thành used/unused, lưu ra file
			// để lần sau tái dùng mail chưa dùng (Config/RentMail/unused_<provider>.txt).
			persistUsedUnused(a.emailPool)
			regWorkerCancel() // cancel sau khi mọi worker đã xong
			regBatchCancel()
			// Giải phóng batch cache ngay sau khi run xong
			regBatchCache.Range(func(k, _ any) bool { regBatchCache.Delete(k); return true })
			// Giải phóng per-slot UA cache (KeepUASuccess)
			regUABySlot.Range(func(k, _ any) bool { regUABySlot.Delete(k); return true })
			regDatrBySlot.Range(func(k, _ any) bool { regDatrBySlot.Delete(k); return true })
			regSessionBySlot.Range(func(k, _ any) bool { regSessionBySlot.Delete(k); return true })
			if regCounters != nil {
				regCounters.Stop()
			}
			// Giải phóng toàn bộ sticky proxy entries còn pin (port C# KeepIPSuccess)
			regSticky.ReleaseAll()

			// Close session pool sau khi tất cả worker đã exit (wg.Wait() đảm bảo).
			// runResources.Cleanup đóng LOCAL pool refs được capture lúc run start +
			// nil global ref nếu vẫn khớp local (defense-in-depth). Đảm bảo close ĐÚNG
			// pool của run này, KHÔNG đụng pool run mới (state machine + nil-on-equal
			// guard cộng dồn). Logging "REG pool cleanup" + closedSessions count nằm
			// bên trong Cleanup() để 1 chỗ duy nhất emit log lifecycle pool.
			runRes.Cleanup()

			a.registerMu.Lock()
			// Chỉ reset cancel nếu là run hiện tại (gen khớp). Vì state machine chặn
			// mọi Start trong khi state=stopping, không thể có run mới giành cancel slot
			// ở đây — kiểm tra gen vẫn giữ làm defense-in-depth (vd nếu future code path
			// nào đó bypass state check).
			if a.registerGen == myGen {
				a.registerCancel = nil
			}
			a.registerMu.Unlock()
			// State transition stopping → idle (hoặc running → idle nếu run kết thúc tự
			// nhiên ngoài Stop user, hiện tại không xảy ra vì spawner loop forever).
			// CUỐI CÙNG sau khi đã wg.Wait + cleanup native resources — đảm bảo Start mới
			// chỉ được phép sau khi mọi worker + session pool + transport buffer đã release.
			a.registerState.Store(int32(runStateIdle))
			runtime.EventsEmit(a.ctx, "register:complete", map[string]interface{}{
				"total": int(atomic.LoadInt64(&counter)),
			})
		}()

		for {
			// Chờ slot trống hoặc bị dừng — nhận luôn slot ID từ freeSlotsReg.
			// Đảm bảo mỗi goroutine nhận 1 slot UNIQUE, không xảy ra 2 goroutine
			// cùng slotIdx chạy song song (bug cũ với counting semaphore).
			var slotIdx int
			select {
			case <-ctx.Done():
				return
			case slotIdx = <-freeSlotsReg:
			}

			// Kiểm tra lại sau khi lấy được slot
			select {
			case <-ctx.Done():
				freeSlotsReg <- slotIdx
				return
			default:
			}

			idx := int(atomic.AddInt64(&counter, 1))
			// slotIdx đến từ freeSlotsReg (không còn tính theo idx % maxThreads)

			// Realtime reload — user thay đổi settings giữa batch có hiệu lực từ acc kế tiếp.
			// Áp dụng cho cả REG (Password/Name/Mode/Phone×Mail/FmPhoneCode) + VERIFY config.
			interactionCfg = a.LoadInteractionConfig()
			// Realtime reload REG platform — user đổi API REG mid-batch có hiệu lực ngay.
			// Tất cả datr pool đã init sẵn, nên switch platform không cần reinit.
			// Multi-version: mỗi slot được gán cố định 1 version theo round-robin
			// (slot1→list[0], slot2→list[1], ...). Slot giữ nguyên version suốt đời nó →
			// keep-ip / keep-ua / keep-datr hoạt động y hệt single-version trong từng slot.
			slotPlatforms := regPlatformList(interactionCfg)
			regPlatform = slotPlatforms[(slotIdx-1)%len(slotPlatforms)]
			// Re-apply per-platform UA override mỗi account — user đổi UA mid-run có hiệu lực ngay.
			interactionCfg = applyRegPlatformUAConfig(interactionCfg, regPlatform)
			interactionCfg = applyRegModeRotation(interactionCfg, regModeRotateStartedAt, time.Now())
			// KHÔNG pre-assign proxy từ static proxyPool (stale sau khi user đổi proxy).
			// Worker gọi a.getSharedProxyManager().Acquire() trong goroutine → realtime reload.
			// proxyPool chỉ dùng cho pre-check health ở đầu RunRegister.
			proxyStr := ""
			// Pre-generate UA để emit ngay trong register:status (hiện cột UA realtime)
			// Mỗi platform dùng UA riêng: Android API = Dalvik FB4A, WebAndroid = Chrome browser.
			// UA ở đây CHỈ để hiện UI — UA thực gửi lên FB được regenerate trong worker
			// (s23reg.RegisterAccount, android.RegisterAccount, v.v.) → dùng builder rẻ
			// tránh double-build profile (SIM + device + UUID x3) tốn thời gian init.
			fpForProfile := fakeinfo.RandomFakeProfile()
			profile := instagram.RegInput{
				Proxy:     proxyStr,
				FirstName: fpForProfile.FirstName,
				LastName:  fpForProfile.LastName,
				Birthday:  fpForProfile.Birthday,
			}
			switch regPlatform {
			case instagram.PlatformWebAndroid:
				profile.UserAgent = fakeinfo.RandomChromeAndroidProfile().UserAgent
			case instagram.PlatformIOS562, instagram.PlatformIOS563, instagram.PlatformIOS555, instagram.PlatformIOS550, instagram.PlatformIOS540, instagram.PlatformIOS530, instagram.PlatformIOS520, instagram.PlatformIOS564,
				instagram.PlatformIOS510, instagram.PlatformIOS500, instagram.PlatformIOS490, instagram.PlatformIOS480, instagram.PlatformIOS470, instagram.PlatformIOS460, instagram.PlatformIOS450,
				instagram.PlatformIOS440, instagram.PlatformIOS430, instagram.PlatformIOS420, instagram.PlatformIOS560,
				instagram.PlatformIOS421, instagram.PlatformIOS422, instagram.PlatformIOS423, instagram.PlatformIOS424, instagram.PlatformIOS425, instagram.PlatformIOS426, instagram.PlatformIOS427, instagram.PlatformIOS428, instagram.PlatformIOS429, instagram.PlatformIOS431, instagram.PlatformIOS432, instagram.PlatformIOS433, instagram.PlatformIOS434, instagram.PlatformIOS435, instagram.PlatformIOS436, instagram.PlatformIOS437, instagram.PlatformIOS438, instagram.PlatformIOS439, instagram.PlatformIOS441, instagram.PlatformIOS442, instagram.PlatformIOS443, instagram.PlatformIOS444, instagram.PlatformIOS445, instagram.PlatformIOS446, instagram.PlatformIOS447, instagram.PlatformIOS448, instagram.PlatformIOS449, instagram.PlatformIOS451, instagram.PlatformIOS452, instagram.PlatformIOS453, instagram.PlatformIOS454, instagram.PlatformIOS455, instagram.PlatformIOS456, instagram.PlatformIOS457, instagram.PlatformIOS458, instagram.PlatformIOS459, instagram.PlatformIOS461, instagram.PlatformIOS462, instagram.PlatformIOS463, instagram.PlatformIOS464, instagram.PlatformIOS465, instagram.PlatformIOS466, instagram.PlatformIOS467, instagram.PlatformIOS468, instagram.PlatformIOS469, instagram.PlatformIOS471, instagram.PlatformIOS472, instagram.PlatformIOS473, instagram.PlatformIOS474, instagram.PlatformIOS475, instagram.PlatformIOS476, instagram.PlatformIOS477, instagram.PlatformIOS478, instagram.PlatformIOS479, instagram.PlatformIOS481, instagram.PlatformIOS482, instagram.PlatformIOS483, instagram.PlatformIOS484, instagram.PlatformIOS485, instagram.PlatformIOS486, instagram.PlatformIOS487, instagram.PlatformIOS488, instagram.PlatformIOS489, instagram.PlatformIOS491, instagram.PlatformIOS492, instagram.PlatformIOS493, instagram.PlatformIOS494, instagram.PlatformIOS495, instagram.PlatformIOS496, instagram.PlatformIOS497, instagram.PlatformIOS498, instagram.PlatformIOS499, instagram.PlatformIOS501, instagram.PlatformIOS502, instagram.PlatformIOS503, instagram.PlatformIOS504, instagram.PlatformIOS505, instagram.PlatformIOS506, instagram.PlatformIOS507, instagram.PlatformIOS508, instagram.PlatformIOS509, instagram.PlatformIOS511, instagram.PlatformIOS512, instagram.PlatformIOS513, instagram.PlatformIOS514, instagram.PlatformIOS515, instagram.PlatformIOS516, instagram.PlatformIOS517, instagram.PlatformIOS518, instagram.PlatformIOS519, instagram.PlatformIOS521, instagram.PlatformIOS522, instagram.PlatformIOS523, instagram.PlatformIOS524, instagram.PlatformIOS525, instagram.PlatformIOS526, instagram.PlatformIOS527, instagram.PlatformIOS528, instagram.PlatformIOS529, instagram.PlatformIOS531, instagram.PlatformIOS532, instagram.PlatformIOS533, instagram.PlatformIOS534, instagram.PlatformIOS535, instagram.PlatformIOS536, instagram.PlatformIOS537, instagram.PlatformIOS538, instagram.PlatformIOS539, instagram.PlatformIOS541, instagram.PlatformIOS542, instagram.PlatformIOS543, instagram.PlatformIOS544, instagram.PlatformIOS545, instagram.PlatformIOS546, instagram.PlatformIOS547, instagram.PlatformIOS548, instagram.PlatformIOS549, instagram.PlatformIOS551, instagram.PlatformIOS552, instagram.PlatformIOS553, instagram.PlatformIOS554, instagram.PlatformIOS556, instagram.PlatformIOS557, instagram.PlatformIOS558, instagram.PlatformIOS559, instagram.PlatformIOS561:
				// iOS562/563 tự build UA nội bộ (FBAN/FBIOS) — không dùng Android pool.
				// Để trống; sẽ được cập nhật từ result.UserAgent sau reg.
				profile.UserAgent = ""
			default:
				// Android/S23 pre-gen UA cho UI — 4-state logic khớp với worker:
				//   BuildUA=false, AddVirtualSpec=false → pool UA thô
				//   BuildUA=false, AddVirtualSpec=true  → pool UA + Dalvik prefix
				//   BuildUA=true,  AddVirtualSpec=false → build động, KHÔNG Dalvik
				//   BuildUA=true,  AddVirtualSpec=true  → build động + Dalvik
				//
				// QUAN TRỌNG: BuildUA=true + RegMode∈{Phone,Random} → defer UA build cho đến
				// khi CheckIP biết countryCode, vì locale/carrier phải match IP. Để trống
				// UI hiển thị "" cho cột UserAgent đến khi CheckIP xong.
				modePre := strings.TrimSpace(interactionCfg.RegMode)
				needIPCountry := interactionCfg.BuildUA &&
					(strings.EqualFold(modePre, "Phone") || strings.EqualFold(modePre, "Random"))
				// UseOriginalUA: pre-gen UA bằng OriginalUA của platform để UI không flicker
				// FBAV random (RandomFbVersion picks từ Config/Fbapp/versions_and_builds.txt).
				// Worker sau đó vẫn override với FBCR-replaced version (theo IP) ở line ~7069.
				if interactionCfg.UseOriginalUA {
					if origUA := originalUAForPlatform(regPlatform, ""); origUA != "" {
						profile.UserAgent = origUA
					} else {
						profile.UserAgent = "" // platform không có OriginalUA → chờ worker
					}
				} else if needIPCountry {
					profile.UserAgent = "" // chờ CheckIP → build UA với locale/carrier theo IP
				} else if interactionCfg.BuildUA {
					dev := fakeinfo.RandomDeviceProfile()
					locale := fakeinfo.LocaleFromCountry("")
					carrier := fakeinfo.RandomCarrier()
					fbVer, fbBuild := fakeinfo.RandomFbVersionReg()
					profile.UserAgent = fakeinfo.BuildAndroidUAWithOpts(dev, locale, carrier, fbVer, fbBuild, interactionCfg.AddVirtualSpecAndroid, false)
				} else {
					kind := uaKindFromPoolKey(interactionCfg.UaPoolKey)
					ua := fakeinfo.RandomUAFromPool(kind)
					if ua == "" {
						dev := fakeinfo.RandomDeviceProfile()
						locale := fakeinfo.LocaleFromCountry("")
						carrier := fakeinfo.RandomCarrier()
						fbVer, fbBuild := fakeinfo.RandomFbVersionReg()
						ua = fakeinfo.BuildAndroidUAWithOpts(dev, locale, carrier, fbVer, fbBuild, interactionCfg.AddVirtualSpecAndroid, false)
					} else if interactionCfg.AddVirtualSpecAndroid {
						ua = fakeinfo.WrapWithDalvikPrefix(ua)
					}
					profile.UserAgent = ua
				}
			}
			// IG rebrand: igcore tự sinh iOS UA sau khi biết country code của proxy.
			// Xóa pre-gen Android UA để UI không hiện sai UA trong lúc reg chạy;
			// result.UserAgent sẽ được gán vào prof sau khi Register() trả về.
			profile.UserAgent = ""

			// Cookie Initial: bình thường lấy từ SharedPool theo slot để tránh trùng giữa luồng.
			// KeepDatrSuccess: nếu slot vừa reg thành công và có datr mới, ưu tiên dùng lại datr đó.
			if interactionCfg.KeepDatrSuccess {
				if v, ok := regDatrBySlot.Load(slotIdx); ok {
					if datr, _ := v.(string); datr != "" && !strings.HasPrefix(datr, "_") && !strings.HasPrefix(datr, "-") {
						profile.TutDatr = datr
					}
				}
			}
			if profile.TutDatr == "" && interactionCfg.CreateType == "tut" && len(tutDatrPool) > 0 {
				profile.TutDatr = tutDatrPool[(idx-1)%len(tutDatrPool)]
			} else if profile.TutDatr == "" {
				dynamicTutMu.Lock()
				if n := len(dynamicTutPool); n > 0 {
					profile.TutDatr = dynamicTutPool[(idx-1)%n]
				}
				dynamicTutMu.Unlock()
			}

			profile.SlotIdx = slotIdx
			// Pass cookie init method xuống worker — s23 dùng để gate warm login.
			profile.CookieInitialMethod = interactionCfg.CookieInitialMethod
			profile.DelayStep = interactionCfg.DelayStep
			// C#: MainFormUISettings.PasswordReg template — mỗi `*` thay bằng char random
			// vd "Fb***2025*" → "Fba7X2025y". Rỗng → RandomPassword fallback.
			if tpl := strings.TrimSpace(interactionCfg.PasswordReg); tpl != "" {
				profile.Password = fakeinfo.PasswordFromTemplate(tpl)
			}
			// C# ModeReg = "Mail" → dùng email làm contactpoint (thay vì phone).
			// LeadDomainMail split "," → random 1 domain → sinh email đầy đủ.
			if strings.EqualFold(strings.TrimSpace(interactionCfg.RegMode), "Mail") {
				profile.Email = fakeinfo.EmailFromProfile(profile.FirstName, profile.LastName, profile.Birthday, interactionCfg.LeadDomainMail)
				profile.Phone = "" // clear phone để register dùng email contactpoint
			}
			// TempMail mode: clear phone + Email → goroutine sẽ acquireTempMailForReg
			// (mua/tạo mail thật từ provider) sau khi đã có proxy. Set sentinel ở
			// đây để gen profile rộng hơn không lỡ tay set phone hoặc email giả.
			if strings.EqualFold(strings.TrimSpace(interactionCfg.RegMode), "TempMail") {
				profile.Email = "" // sentinel — goroutine sẽ acquire
				profile.Phone = ""
			}
			// MailTemp mode: giống TempMail nhưng dùng mail-temp.com (client-side, không API key).
			if strings.EqualFold(strings.TrimSpace(interactionCfg.RegMode), "MailTemp") {
				profile.Email = "" // sentinel — goroutine sẽ dùng acquireMailTempComForReg
				profile.Phone = ""
			}
			// C# NameReg = "VN" → override FirstName/LastName bằng VN name database.
			// Mặc định (US/""): giữ nguyên profile (webregister.RandomRegInput đã gen US name).
			if strings.EqualFold(strings.TrimSpace(interactionCfg.NameRegLocale), "VN") {
				fp := fakeinfo.RandomFakeProfileByLocale("VN")
				profile.FirstName = fp.FirstName
				profile.LastName = fp.LastName
			}
			// C# RandomPhonexMailMethod = "random-file" → đọc permanent phone/mail
			// từ Config/Permanent/phone.txt (hoặc mail.txt). Fallback random nếu file rỗng.
			if strings.EqualFold(strings.TrimSpace(interactionCfg.PhoneMailMode), "random-file") {
				permDir := defaultPermanentDir()
				if profile.Email != "" {
					if line := fakeinfo.RandomLineFromFile(filepath.Join(permDir, "mail.txt")); line != "" {
						profile.Email = line
					}
				} else if profile.Phone != "" {
					if line := fakeinfo.RandomLineFromFile(filepath.Join(permDir, "phone.txt")); line != "" {
						profile.Phone = line
					}
				}
			}
			// C# FmPhoneCode = true (hoặc PhoneMailMode = "fm-phone") — convert E164
			// thành local: "84912345678" → "0912345678" (theo country code của proxy).
			if profile.Phone != "" &&
				(interactionCfg.FmPhoneCode || strings.EqualFold(strings.TrimSpace(interactionCfg.PhoneMailMode), "fm-phone")) {
				profile.Phone = fakeinfo.ConvertPhoneToLocal(profile.Phone, interactionCfg.NameRegLocale)
			}
			// BuildUA + RegMode∈{Phone,Random}: clear default phone (RandomRegInput tạo +84 VN)
			// để UI không hiện +84 trước khi CheckIP. Phone thật sẽ gen sau CheckIP theo countryCode.
			{
				modeP := strings.TrimSpace(interactionCfg.RegMode)
				if interactionCfg.BuildUA &&
					(strings.EqualFold(modeP, "Phone") || strings.EqualFold(modeP, "Random")) {
					profile.Phone = ""
				}
			}
			accountCfg := interactionCfg
			accountRegPlatform := regPlatform
			wg.Add(1)
			go func(threadIdx int, prof instagram.RegInput, interactionCfg InteractionConfig, regPlatform string) {
				defer wg.Done()
				defer func() { freeSlotsReg <- threadIdx }()
				slog.Debug("REG worker spawned", "thread", threadIdx, "slot", slotIdx, "proxy_set", prof.Proxy != "")

				// Helper: contact hiện lên UI column "EMAIL / PHONE" — phone nếu có, không thì email.
				// Mode=Mail → Phone="" nên hiện email; Mode=Phone → Phone có giá trị nên hiện phone.
				pickContact := func() string {
					if prof.Phone != "" {
						return prof.Phone
					}
					return prof.Email
				}

				// Recover worker panic — log + emit status để UI hiện lỗi thay vì tắt im lặng
				defer func() {
					if r := recover(); r != nil {
						slog.Error("REG worker panic", "thread", threadIdx, "recovered", r,
							"stack", string(debug.Stack()))
						runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
							"index":       threadIdx,
							"phone":       pickContact(),
							"proxy":       "",         // không show IP chạy ở panic path
							"proxyServer": prof.Proxy, // full proxy string (hiện session token)
							"userAgent":   prof.UserAgent,
							"msg":         fmt.Sprintf("[PANIC] %v", r),
						})
					}
				}()

				// Acquire proxy từ dynamic provider (ShopLike, Tinsoft...) bên trong goroutine
				// Sticky proxy per slot — success giữ proxy, fail thả về pool.
				// Release callback lưu để gọi sau khi reg xong (biết success/fail).
				var regStickyRelease func(success bool)
				if prof.Proxy == "" {
					if p, rel := regSticky.Acquire(ctx, slotIdx); p != "" {
						prof.Proxy = p
						regStickyRelease = rel
					}
				}
				// Defer release để đảm bảo proxy slot được trả về khi goroutine exit sớm
				// (ctx.Done() inside loop) — explicit calls below handle normal flow.
				defer func() {
					if regStickyRelease != nil {
						regStickyRelease(false)
					}
				}()

				// SessionPool key = thread slot (C#: mỗi thread giữ session riêng qua nhiều lần reg)
				// Dùng slotIdx thay vì proxy → tránh session poisoning khi nhiều thread share proxy
				prof.ProxyKey = fmt.Sprintf("slot_%d", slotIdx)

				// Render session proxy — REUSE per slot: 1 luồng dùng chung 1 IP session
				// xuyên suốt các account. Tránh tạo session mới mỗi account → proxy phải cấp
				// exit IP + TCP mới mỗi lần → CheckIP rớt khi nhiều luồng burst (xem regSlotSession).
				// Cache theo base proxy: base đổi (pool rotate) → render session mới.
				{
					base := prof.Proxy
					reuse := ""
					if v, ok := regSessionBySlot.Load(slotIdx); ok {
						if e := v.(regSlotSession); e.base == base {
							reuse = e.rendered
						}
					}
					if reuse != "" {
						prof.Proxy = reuse
					} else {
						prof.Proxy = proxy.RenderSessionIfIsProxyServer(base)
						regSessionBySlot.Store(slotIdx, regSlotSession{base: base, rendered: prof.Proxy})
					}
				}

				// Emit PLACEHOLDER ngay — user thấy "Khởi chạy..." trước khi CheckIP (có thể chậm 0-6s).
				// IP CHẠY = "" tạm thời (chưa có IP thật) — KHÔNG đẩy raw proxy lên cột này.
				// PROXY full string (bao gồm user:pass + session token) → user theo dõi session rotation.
				regBatchCache.Delete(threadIdx)
				runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
					"index":       threadIdx,
					"phone":       pickContact(),
					"proxy":       "",         // CheckIP chưa xong — IP CHẠY để trống
					"proxyServer": prof.Proxy, // PROXY column: full string
					"userAgent":   prof.UserAgent,
					"msg":         "▶️ Khởi chạy luồng reg tiếp theo... (đang check IP proxy)",
					"reset":       true,
				})

				// Kiểm tra IP thực (timeout 6s hard), dùng làm display IP + country.
				// Nếu CheckIP fail → displayProxy = "" (cột IP CHẠY để trống, KHÔNG show raw proxy).
				// Goroutine+select đảm bảo hard 6s timeout bất kể http.Client.Timeout per-request:
				// fallback chain (ip-api→adspower→luna→ipify) có thể treo 4×6=24s nếu chỉ dùng context.
				generalCfg := a.LoadSettings().General
				displayProxy := ""
				countryCode := ""
				{
					type ipRes struct{ ip string }
					ipCh := make(chan ipRes, 1)
					checkCtx, checkCancel := context.WithTimeout(ctx, 6*time.Second)
					go func() {
						ip, err := proxy.CheckIP(checkCtx, prof.Proxy, generalCfg.ApiCheckIp)
						if err != nil {
							ip = ""
						}
						ipCh <- ipRes{ip}
					}()
					select {
					case res := <-ipCh:
						if res.ip != "" {
							displayProxy = res.ip
							if idx := strings.LastIndex(res.ip, "/"); idx >= 0 {
								countryCode = strings.ToLower(res.ip[idx+1:])
							}
						}
					case <-time.After(6 * time.Second):
					case <-ctx.Done():
					}
					checkCancel()
				}
				// Sinh số điện thoại theo quốc gia của IP.
				// BỎ QUA khi Mode=Mail HOẶC Mode=TempMail: contactpoint là email, không cần phone.
				// Gen phone lại sẽ hiển thị sai trên UI column "EMAIL/PHONE" + có thể lẫn lộn contactpoint.
				modeRaw := strings.TrimSpace(interactionCfg.RegMode)
				isMailMode := strings.EqualFold(modeRaw, "Mail")
				isTempMailMode := strings.EqualFold(modeRaw, "TempMail") || strings.EqualFold(modeRaw, "MailTemp")
				// BuildUA + Phone/Random: bắt buộc CheckIP phải lấy được countryCode.
				// Nếu không → abort account này (UA + phone phụ thuộc IP country).
				if interactionCfg.BuildUA && !isMailMode && !isTempMailMode && countryCode == "" {
					runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
						"index":       threadIdx,
						"phone":       "",
						"proxy":       "",
						"proxyServer": prof.Proxy,
						"userAgent":   "",
						"msg":         "❌ Bỏ qua: CheckIP thất bại — BuildUA cần country code từ IP",
					})
					if regStickyRelease != nil {
						regStickyRelease(false)
						regStickyRelease = nil
					}
					return
				}
				if countryCode != "" && !isMailMode && !isTempMailMode {
					phone, _ := fakeinfo.PhoneFromDatabase(countryCode, defaultPhoneDatabaseDir())
					if phone != "" {
						prof.Phone = phone
					} else {
						logMissingPhoneCountryCode(countryCode)
					}
				}
				// C# SimNetworkOptions: 1=random (any country), 2=match by IP.
				// simNetworkMode="random" → clear countryCode để RandomSimProfile trả random
				// thay vì filter theo proxy country (C# mode 1).
				effectiveCountryCode := countryCode
				if strings.EqualFold(strings.TrimSpace(generalCfg.SimNetworkMode), "random") {
					effectiveCountryCode = "" // random từ pool toàn cầu
				}

				// Update event với real IP sau CheckIP — frontend nhận proxy = realIP.
				// Msg hiển thị phone/email đã gen để user thấy ngay trong cột Hoạt động.
				contactInfo := prof.Phone
				if contactInfo == "" && prof.Email != "" {
					contactInfo = prof.Email
				}
				prepMsg := "▶️ IP proxy OK — chuẩn bị reg..."
				if contactInfo != "" {
					prepMsg = fmt.Sprintf("▶️ IP proxy OK — chuẩn bị reg | %s", contactInfo)
				}
				runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
					"index":       threadIdx,
					"phone":       pickContact(),
					"proxy":       displayProxy,
					"proxyServer": prof.Proxy, // proxy server string (ip:port:user:pass)
					"userAgent":   prof.UserAgent,
					"msg":         prepMsg,
				})

				onStatus := func(msg string) {
					// igcore emit "ua:<iOS UA>" ngay sau khi build profile → cập nhật sớm
					if strings.HasPrefix(msg, "ua:") {
						prof.UserAgent = strings.TrimPrefix(msg, "ua:")
						return
					}
					// Ghi vào batch cache — batch goroutine flush lên frontend mỗi 500ms
					regBatchCache.Store(threadIdx, regEntry{
						index: threadIdx, phone: pickContact(), proxy: displayProxy, userAgent: prof.UserAgent, msg: msg,
					})
				}

				threadReg, threadRegErr := instagram.NewRegisterer(regPlatform)

				// UA selection — 5 trạng thái:
				//   UseOriginalUA=true               → UA gốc cố định theo platform (s555-s559)
				//   BuildUA=false, AddVirtual=false  → UA gốc từ Config/UserAgent/{kind}_UG.txt
				//   BuildUA=false, AddVirtual=true   → UA gốc + Dalvik prefix
				//   BuildUA=true,  AddVirtual=false  → build động từ Config/DeviceInfo
				//   BuildUA=true,  AddVirtual=true   → build động + Dalvik
				if interactionCfg.UseOriginalUA {
					origCC := effectiveCountryCode
					if !interactionCfg.ReplaceCarrier {
						origCC = "" // giữ nguyên FBCR/Viettel gốc, không thay theo IP
					}
					origUA, origSim := originalUAForPlatformWithSim(regPlatform, origCC)
					if origUA != "" {
						prof.UserAgent = origUA
						// Chỉ FBCR (nhà mạng) trong OriginalUA được thay theo IP. Body/headers
						// honor flag này → override profile.Sim = origSim (HNI/MCC/MNC khớp FBCR)
						// và profile.Locale = "en_GB" (khớp FBLC trong OriginalUA).
						prof.UseOriginalUA = true
						prof.OriginalSim = origSim
					}
				} else if !interactionCfg.BuildUA {
					kind := uaKindFromPoolKey(interactionCfg.UaPoolKey)
					rawUA := fakeinfo.RandomUAFromPool(kind)
					if rawUA != "" {
						finalUA := rawUA
						if interactionCfg.AddVirtualSpecAndroid {
							finalUA = fakeinfo.WrapWithDalvikPrefix(rawUA)
						}
						prof.UserAgent = finalUA
					} else {
						// IG rebrand: UA pool rỗng KHÔNG còn là lỗi — adapter IG tự sinh UA.
						// Chỉ log, không return; prof.UserAgent để trống, igcore tự đặt.
						runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
							"index": threadIdx, "phone": pickContact(), "proxy": prof.Proxy,
							"msg": "[UA] pool rỗng — IG tự sinh UA, bỏ qua pool",
						})
					}
				}

				// KeepUASuccess: nếu slot đã có UA từ lần reg thành công trước → override.
				// Phải chạy SAU block UA selection để ghi đè cả BuildUA lẫn pool UA.
				if interactionCfg.KeepUASuccess {
					if v, ok := regUABySlot.Load(slotIdx); ok {
						if kept, ok := v.(regSlotUA); ok && kept.Platform == regPlatform && kept.UA != "" {
							keptUA := kept.UA
							prof.UserAgent = keptUA
						} else {
							regUABySlot.Delete(slotIdx)
						}
					}
				}

				// TrackingID: thêm XID/<random16>; vào UA sau khi mọi override đã xong.
				if interactionCfg.TrackingIDReg && prof.UserAgent != "" {
					prof.UserAgent = appendXIDToUA(prof.UserAgent)
				}

				// IG rebrand: igcore tự sinh iOS UA nội bộ → xóa Android UA từ pool/selection
				// NGAY TẠI ĐÂY để mọi event tiếp theo (TempMail, onStatus, result) đều
				// không carry Android FB UA ra UI. result.UserAgent sẽ set iOS UA sau reg.
				prof.UserAgent = ""

				// C# MFB: RegisterWithKeepHttpSession runs in a LOOP per thread:
				//   Success + CookieInitial → keep session → delay → reg again (same proxy/IP)
				//   Fail/Blocked → break → thread ends → new session
				// This "warm session" pattern gives higher success rate for subsequent regs.
				maxKeepSessionRegs := 1 // default: 1 reg per goroutine (no loop)

				var result *instagram.RegResult
				for regAttempt := 0; regAttempt < maxKeepSessionRegs; regAttempt++ {
					select {
					case <-ctx.Done():
						return
					default:
					}

					// Realtime reload interaction config — user thay đổi Delay/WaitCode
					// qua UI giữa chừng có hiệu lực ngay ở iteration kế tiếp.
					// Port UX C#: không cần Stop+Start để áp dụng settings mới.
					interactionCfg = applyRegPlatformUAConfig(a.LoadInteractionConfig(), regPlatform)
					interactionCfg = applyRegModeRotation(interactionCfg, regModeRotateStartedAt, time.Now())

					if regAttempt > 0 {
						// Subsequent reg on same session: generate new fake info but KEEP same proxy
						// BỎ QUA gen phone khi Mode=Mail HOẶC Mode=TempMail (contactpoint là email).
						modeRawAttempt := strings.TrimSpace(interactionCfg.RegMode)
						switch {
						case strings.EqualFold(modeRawAttempt, "Mail"):
							// Regenerate email mỗi attempt để tránh trùng + bị FB block email cũ
							prof.Email = fakeinfo.EmailFromProfile(prof.FirstName, prof.LastName, prof.Birthday, interactionCfg.LeadDomainMail)
							prof.Phone = ""
						case strings.EqualFold(modeRawAttempt, "TempMail"):
							// TempMail: clear cả 2 — goroutine sẽ acquire mail mới qua acquireTempMailForReg
							// (mỗi attempt acquire mail mới vì mail tạm TTL ngắn).
							prof.Email = ""
							prof.Phone = ""
						case strings.EqualFold(modeRawAttempt, "MailTemp"):
							// MailTemp: clear cả 2 — goroutine sẽ acquire mail mới qua acquireMailTempComForReg.
							prof.Email = ""
							prof.Phone = ""
						default:
							phone, _ := fakeinfo.PhoneFromDatabase(countryCode, defaultPhoneDatabaseDir())
							if phone != "" {
								prof.Phone = phone
							} else if countryCode != "" {
								logMissingPhoneCountryCode(countryCode)
							}
						}
						newFake := fakeinfo.RandomFakeProfile()
						prof.FirstName = newFake.FirstName
						prof.LastName = newFake.LastName
						prof.Birthday = newFake.Birthday
						prof.Gender = newFake.Gender
						// Retry KeepSession: regen pass MỚI nhưng GIỮ mẫu user nhập (PasswordReg).
						// Rỗng → PasswordFromTemplate tự fallback RandomPassword.
						prof.Password = fakeinfo.PasswordFromTemplate(interactionCfg.PasswordReg)
						onStatus(fmt.Sprintf("[KeepSession] reg attempt %d on same session...", regAttempt+1))
						// Cancellable sleep — Stop register thoát sớm.
						select {
						case <-time.After(time.Duration(interactionCfg.DelayReg) * time.Second):
						case <-ctx.Done():
							return
						}
					}

					// TempMail mode: acquire mail tạm THẬT từ provider TRƯỚC khi reg.
					// Mỗi reg attempt acquire mail mới (mail tạm TTL ngắn, attempt sau
					// dùng lại mail cũ rủi ro die). Mail + Snapshot creds gắn vào prof
					// → register handler dùng làm contactpoint → verify Restore sau.
					var tempMailHandle *TempMailHandle
					// TempMail mode: acquire mail tạm THẬT từ provider TRƯỚC khi reg. iOS Mess reg
					// phone-only KHÔNG acquire ở đây — tempmail được lấy Ở BƯỚC VER (add-mail) để
					// khỏi phí mail cho account die ngay ở create (integrity_block/checkpoint).
					if strings.EqualFold(strings.TrimSpace(interactionCfg.RegMode), "TempMail") {
						verifyCfgForMail := buildVerifyConfigFromInteraction(interactionCfg)
						onStatus(fmt.Sprintf("[TempMail] Acquiring mail từ provider=%q...", verifyCfgForMail.MailProvider))
						handle, mailErr := acquireTempMailForReg(ctx, verifyCfgForMail, prof.Proxy, onStatus)
						if mailErr != nil {
							onStatus(fmt.Sprintf("[TempMail] ❌ Acquire fail: %v — skip thread", mailErr))
							return
						}
						tempMailHandle = handle
						prof.Email = handle.Email
						prof.EmailMeta = handle.Meta
						// Emit event riêng để frontend update cột EMAIL ngay (không chờ
						// đến status emit sau — frontend overwrite full thread record).
						runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
							"index":       threadIdx,
							"phone":       handle.Email,
							"proxy":       displayProxy,
							"proxyServer": prof.Proxy,
							"userAgent":   prof.UserAgent,
							"msg":         fmt.Sprintf("✉️ TempMail: %s (meta=%dB) — bắt đầu reg...", handle.Email, len(handle.Meta)),
						})
					}

					// MailTemp mode: acquire email từ mail-temp.com (client-side, không API key).
					if strings.EqualFold(strings.TrimSpace(interactionCfg.RegMode), "MailTemp") {
						onStatus("[MailTemp] Tạo email từ mail-temp.com...")
						handle, mailErr := acquireMailTempComForReg(ctx, prof.Proxy, onStatus)
						if mailErr != nil {
							onStatus(fmt.Sprintf("[MailTemp] ❌ Tạo email thất bại: %v — skip thread", mailErr))
							return
						}
						tempMailHandle = handle
						prof.Email = handle.Email
						prof.EmailMeta = handle.Meta
						runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
							"index":       threadIdx,
							"phone":       handle.Email,
							"proxy":       displayProxy,
							"proxyServer": prof.Proxy,
							"userAgent":   prof.UserAgent,
							"msg":         fmt.Sprintf("✉️ MailTemp: %s — bắt đầu reg...", handle.Email),
						})
					}

					// appmv3reg / iosmessreg (Messenger reg): cấp GetOTP để reg TỰ confirm OTP
					// ngay trong reg (multi-step create → OTP → confirm). Mail handle còn live ở đây.
					// IG reg flow CẦN OTP NGAY trong reg (submitEmail → confirmOTP là một phần
					// của createAccount). Vì app đã chuyển sang Instagram, cấp GetOTP cho MỌI
					// platform khi đang TempMail mode (mail thật, đọc được OTP). Mail handle còn live.
					if tempMailHandle != nil && tempMailHandle.Service != nil {
						mailSvc := tempMailHandle.Service
						prof.GetOTP = func(c context.Context) (string, error) {
							// 45×2s=90s (giảm từ 180s): OTP IG về trong vài giây; quá ~90s = no-show
							// → fail nhanh, nhả slot, không giữ luồng "đứng chờ" 3 phút.
							return mailSvc.WaitForCode(c, 45, 2000)
						}
						onStatus("[IG] GetOTP wired — reg sẽ tự đọc OTP + confirm")

						// ig_ios_gql: cấp GetNewEmail để Register() tự retry với email mới khi SESSION_FLAGGED.
						if regPlatform == "ig_ios_gql" {
							prof.GetNewEmail = func(c context.Context) (string, func(context.Context) (string, error), error) {
								newAddr, err := mailSvc.CreateEmail(c)
								if err != nil {
									return "", nil, err
								}
								return newAddr, func(inner context.Context) (string, error) {
									return mailSvc.WaitForCode(inner, 45, 2000) // 90s (xem GetOTP trên)
								}, nil
							}
							onStatus("[ig_ios_gql] GetNewEmail wired — auto-retry khi SESSION_FLAGGED")
						}
					}

					// Cookie initial guard — handle pool cạn theo method:
					//
					// method="file":
					//   1. Reload file cookie_initial.txt (user có thể vừa thêm datr mới).
					//   2. Nếu vẫn cạn → đọc 100 dòng CUỐI của SuccessVerify_No2FA.txt
					//      trong output path lần chạy này (datr từ acc đã reg thành công, fresh nhất).
					//   3. Vẫn cạn hoàn toàn (activeCount==0 && exhausted==0) → skip slot.
					//
					// method="new":
					//   - Pool cạn → generate thêm 1 batch datr mới (size = RegThreads * limit * 2).
					if sharedCookiePool != nil && sharedCookiePool.Size() == 0 {
						method := strings.ToLower(strings.TrimSpace(interactionCfg.CookieInitialMethod))
						switch method {
						case "new":
							refill := interactionCfg.RegThreads * cookieInitialLimit * 2
							if refill < 32 {
								refill = 32
							}
							if n := sharedCookiePool.LoadGenerated(refill); n > 0 {
								onStatus(fmt.Sprintf("[CookieInitial] Pool cạn — sinh thêm %d datr mới", n))
							}
						default: // "file" / empty
							reloaded := 0
							for _, ciPath := range cookieInitialFilePaths {
								if n, err := sharedCookiePool.LoadFromFile(ciPath); err == nil && n > 0 {
									reloaded += n
								}
							}
							if reloaded > 0 {
								onStatus(fmt.Sprintf("[CookieInitial] Pool cạn — reload %d datr mới từ file", reloaded))
							} else if outputPath != "" {
								// Fallback bậc 2: SuccessVerify_No2FA.txt của run hiện tại.
								// Lấy 100 dòng CUỐI (datr mới nhất từ acc vừa reg+verify thành công).
								svPath := filepath.Join(outputPath, resultpkg.FileSuccessVerifyNo2FA)
								if n, err := sharedCookiePool.LoadFromFileTail(svPath, 100); err == nil && n > 0 {
									onStatus(fmt.Sprintf("[CookieInitial] Pool cạn — bổ sung %d datr từ %s (100 dòng cuối)", n, resultpkg.FileSuccessVerifyNo2FA))
								} else if sharedCookiePool.IsCompletelyEmpty() {
									onStatus("[CookieInitial] Pool hoàn toàn cạn, không tìm được datr — bỏ qua slot này")
									return
								}
							} else if sharedCookiePool.IsCompletelyEmpty() {
								onStatus("[CookieInitial] Pool hoàn toàn cạn, không tìm được datr — bỏ qua slot này")
								return
							}
						}
					}

					// S23/Android V3/WebAndroid: dùng WorkerContext.Register → reuse session + pinned profile.
					// Các platform khác vẫn dùng threadReg.Register (tạo session mới mỗi call).
					// regWorkerCtx (không bị cancel khi Stop) → HTTP request đang chạy hoàn thành
					// thay vì bị abort giữa chừng. ctx chỉ dùng để gating retry delay + dispatch.
					debugRegBreakpoint(string(regPlatform), slotIdx, threadIdx, regAttempt+1, pickContact(), prof.Proxy, prof.UserAgent)
					if threadReg == nil {
						result = &instagram.RegResult{Success: false, Message: fmt.Sprintf("platform %q không có registerer: %v", regPlatform, threadRegErr)}
					} else {
						result = threadReg.Register(regWorkerCtx, &prof, onStatus)
					}

					// igcore tự build iOS UA nội bộ → luôn ghi đè prof.UserAgent bằng
					// UA thực tế đã dùng cho reg (bỏ điều kiện == "" để không giữ lại
					// Android pre-gen UA từ giai đoạn khởi tạo).
					if result != nil && result.UserAgent != "" {
						prof.UserAgent = result.UserAgent
					}

					// TempMail cleanup sau mỗi reg attempt:
					//   - Reg fail → Release (return mail về pool) + Close
					//   - Reg success → chỉ Close (giữ mail trong account FB; verify side
					//     re-init connection riêng từ prof.EmailMeta blob)
					// Connection KHÔNG được giữ alive sang attempt sau vì mỗi attempt
					// acquire mail mới (mail tạm TTL ngắn).
					if tempMailHandle != nil {
						if result == nil || !result.Success {
							tempMailHandle.ReleaseAndClose(regWorkerCtx)
						} else {
							tempMailHandle.Close()
						}
						tempMailHandle = nil
					}

					// C# logic: only keep session on Success + CookieInitial
					needBreak := true
					if result != nil && result.Success {
						needBreak = false // success → continue loop
					}

					// Xử lý result cho lần reg này (trong loop)
					regBatchCache.Delete(threadIdx)

					// EAA token pre-fetch — phải chạy TRƯỚC khi ghi file để cả file output
					// và split-ver channel đều mang token EAA mới. Áp dụng cho cả 2 mode
					// (split & no-split). Trigger khi: reg success và verify/register platform
					// thuộc Android-family cần user access token EAAAAU. Không fallback sang cookie login cho nhóm này.
					if result != nil && result.Success && result.UID != "" && interactionCfg.VerifyEnabled &&
						(verifyNeedsEAA(interactionCfg.ApiVerifyPlatform) || platformNeedsAndroidLoginToken(string(regPlatform))) &&
						!strings.HasPrefix(result.AccessToken, "EAAAAU") {
						regPassword := result.Password
						if regPassword == "" {
							regPassword = prof.Password
						}
						if regPassword != "" {
							onStatus("[AutoVerify] Thiếu EAAAAU token → login Android để lấy...")
							tok := ""
							if strings.HasPrefix(tok, "EAAAAU") {
								result.AccessToken = tok
								onStatus(fmt.Sprintf("[AutoVerify] ✅ Lấy EAAAAU OK (len=%d)", len(tok)))
								// Register table must keep the Reg UA. The Android token request may use
								// a Verify UA internally, but that belongs to the Verify pane.
								runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
									"index":       threadIdx,
									"phone":       pickContact(),
									"proxy":       displayProxy,
									"proxyServer": prof.Proxy,
									"userAgent":   prof.UserAgent,
									"msg":         fmt.Sprintf("[Token] ✅ Lấy EAAAAU bằng UA Android (len=%d)", len(tok)),
								})
							} else if tok != "" {
								onStatus(fmt.Sprintf("[AutoVerify] ⚠ Login Android trả token không phải EAAAAU (prefix=%s) — không dùng cho verify", tok[:min(len(tok), 6)]))
							} else {
								onStatus("[AutoVerify] ⚠ Lấy EAAAAU thất bại — verify có thể fail")
							}
						}
					}

					// Emit token + cookie ngay sau pre-fetch để bảng register live cập nhật
					// realtime, không phải chờ đến register:account-done (sau khi verify xong).
					if result != nil && result.Success && result.UID != "" {
						runtime.EventsEmit(a.ctx, "register:token", map[string]interface{}{
							"index":    threadIdx,
							"uid":      result.UID,
							"token":    result.AccessToken,
							"cookie":   result.Cookie,
							"username": result.Username,
						})
					}

					// Ghi file kết quả chi tiết qua resultpkg.Writer + counters (port C# FMain.SaveFile + tracking).
					// Chạy async để không block reg loop nếu disk chậm.
					if result != nil && regWriter != nil {
						// Map RegResult → Account DTO để tái dùng saveRegOutcome helper
						accForSave := Account{
							UID:       result.UID,
							Password:  result.Password,
							Cookie:    result.Cookie,
							Token:     result.AccessToken,
							UserAgent: prof.UserAgent,
							// TempMail reuse: persist email + meta để verify (chạy ngay
							// hoặc đọc từ file sau) tự Restore + skip CreateEmail.
							Email:     prof.Email,
							EmailMeta: prof.EmailMeta,
							Username:  result.Username,
						}
						status := "unknown"
						if result.Success {
							status = "live"
						} else if strings.Contains(strings.ToLower(result.Message), "checkpoint") {
							status = "checkpoint"
						} else if strings.Contains(strings.ToLower(result.Message), "block") {
							status = "blocked"
						}
						// Reg flow chỉ có Phone (instagram.RegInput không có Email field)
						login := prof.Phone
						// File write chạy silent — user không cần thấy message "Ghi X.txt" flood UI,
						// kết quả cuối (live/die) đã hiển thị qua status column + counter.
						var successCallbacks []func(string)
						// Chỉ upload sau REG khi cả 2 điều kiện:
						//   1. KHÔNG ở split mode (regToVerCh == nil)
						//   2. KHÔNG bật Verify (VerifyEnabled == false)
						// Lý do: khi user bật REG + Verify (bất kỳ chế độ nào — split hay normal),
						// account REG live sẽ đi qua verify → chỉ upload ở bước verify live.
						// Nếu upload ngay sau REG → dedup theo UID sẽ chặn upload sau verify
						// → user vô tình upload account CHƯA VERIFY (NVR/thô) thay vì account đã verify.
						// Đồng thời tránh upload nhầm account REG live mà verify die/checkpoint.
						if a.LoadUploadSiteConfig().Reg.Enabled && regToVerCh == nil && !interactionCfg.VerifyEnabled {
							uploadCfg := a.LoadUploadSiteConfig()
							if uploadCfg.Code != "" && uploadCfg.ApiKey != "" {
								uc := uploadCfg
								a.ensureUploadRunning(uc)
								successCallbacks = append(successCallbacks, func(line string) {
									a.enqueueForUpload(line)
								})
							}
						}
						if regToVerCh != nil {
							ch := regToVerCh
							successCallbacks = append(successCallbacks, func(line string) {
								select {
								case ch <- line:
								default:
									slog.Warn("regToVerCh full — split-ver line dropped")
								}
							})
						}
						var onRegSuccess func(string)
						if len(successCallbacks) > 0 {
							fns := successCallbacks
							onRegSuccess = func(line string) {
								for _, fn := range fns {
									fn(line)
								}
							}
						}
						go func(regStatus string, acc Account, msg string, checkProxy string) {
							// 1. Ghi kết quả reg NGAY: live → SuccessReg.txt | checkpoint/blocked/unknown
							//    → file tương ứng. User thấy data thành công liền sau khi reg.
							regLine := saveRegOutcome(regWriter, regCounters, regStatus, msg, acc, string(regPlatform), login, onRegSuccess)

							// 2. Reg live → check-live NGAY bằng CheckLiveByUsername (timeout 20s/acc),
							//    KHÔNG chờ delay. live → Live.txt, còn lại (die/unknown) → Die.txt.
							//    Dùng proxy của chính luồng để phân tán request → tránh IG throttle
							//    (direct IP bị 429 khi 20+ luồng check cùng lúc → unknown hàng loạt).
							if regStatus != "live" || acc.Username == "" {
								return
							}
							checkCtx, checkCancel := context.WithTimeout(regWorkerCtx, 20*time.Second)
							result := igcore.CheckLiveByUsername(checkCtx, acc.Username, checkProxy)
							checkCancel()
							// Phân loại file: live → Live.txt | die/unknown → Die.txt (upsert theo field đầu).
							if regWriter != nil && regLine != "" {
								if result == "live" {
									_ = regWriter.Append(resultpkg.FileLive, regLine)
								} else {
									_ = regWriter.UpsertUID(resultpkg.FileDieAfterVerify, regLine)
								}
							}
							// Emit để UI cập nhật Live counter + activity.
							runtime.EventsEmit(a.ctx, "register:check-live-result", map[string]interface{}{
								"username": acc.Username,
								"result":   result,
							})
						}(status, accForSave, result.Message, prof.Proxy)
					}
					if result != nil && result.Cookie != "" {
						if datr := extractDatrFromCookieLine(result.Cookie); datr != "" {
							persistNewDatr(datr)
						} else {
							slog.Warn("[SaveDatr] extractDatr empty", "cookie_prefix", func() string {
								if len(result.Cookie) > 80 {
									return result.Cookie[:80]
								}
								return result.Cookie
							}())
						}
					}
					// Thu datr từ tài khoản thành công vào dynamic TUT pool
					if result.Success && result.Cookie != "" && interactionCfg.CreateType == "tut" {
						if datr := extractDatrFromCookieLine(result.Cookie); datr != "" {
							dynamicTutMu.Lock()
							dynamicTutPool = append(dynamicTutPool, datr)
							dynamicTutMu.Unlock()
							onStatus(fmt.Sprintf("🔗 Thêm datr vào TUT pool (size: %d)", len(dynamicTutPool)))
						}
					}
					// Tích lũy phone/email thành công vào Config/Permanent/ để dùng cho chế độ "Random File"
					if result != nil && result.Success {
						permDir := defaultPermanentDir()
						if prof.Phone != "" {
							go appendUniqueLineToPermanentFile(filepath.Join(permDir, "phone.txt"), prof.Phone)
						} else if prof.Email != "" {
							go appendUniqueLineToPermanentFile(filepath.Join(permDir, "mail.txt"), prof.Email)
						}
					}
					// Auto-verify sau khi tạo tài khoản thành công.
					// 2026-05-15: Split Mode = PURE UI option — KHÔNG còn pool verify riêng.
					// Worker logic GIỐNG Normal Mode: REG xong → VER inline trong cùng worker.
					verifyStatus := ""
					verifyEmail := ""
					verifyMessage := ""
					verifyToken := ""  // token VERIFY (EAAAAAY iOS / EAAAAU android) — để grid hiện đúng loại
					verifyCookie := "" // cookie MỚI sau login verify
					if result != nil && result.Success && result.UID != "" && interactionCfg.VerifyEnabled && interactionCfg.SplitMode {
						// TRUE SPLIT: reg worker CHỈ làm reg. Đẩy account vào splitVerifyCh →
						// VER pool riêng verify async. Reg worker KHÔNG verify inline → giải
						// phóng reg slot ngay (defer freeSlotsReg) → reg full tốc độ.
						// EAA token pre-fetch đã chạy ở block trên → result.AccessToken là token mới.
						job := splitVerifyJob{
							acc: runner.AccountInput{
								UID:                   result.UID,
								Username:              result.Username,
								FullName:              prof.FirstName + " " + prof.LastName,
								Phone:                 prof.Phone,
								Cookie:                result.Cookie,
								Token:                 result.AccessToken,
								UserAgent:             prof.UserAgent,
								Password:              result.Password,
								InputAccount:          formatRegResultLine(result, prof),
								Proxy:                 prof.Proxy,
								DeviceID:              result.DeviceID,
								FamilyDeviceID:        result.FamilyDeviceID,
								Srnonce:               result.Srnonce,
								SessionlessCryptedUID: result.SessionlessCryptedUID,
								Email:                 prof.Email,
								EmailMeta:             prof.EmailMeta,
								AACJid:                result.AACJid,
								AACcs:                 result.AACcs,
								AACts:                 result.AACts,
								RegFlowID:             result.RegFlowID,
								HeadersFlowID:         result.HeadersFlowID,
								PassRaw:               result.PassRaw,
								PassTS:                result.PassTS,
							},
							prof:         prof,
							displayProxy: displayProxy,
							regResult:    result,
						}
						// Bounded channel → block khi VER queue đầy = backpressure tự nhiên
						// (reg chờ VER bắt kịp). Ctx-aware để Stop không treo reg worker.
						select {
						case splitVerifyCh <- job:
						case <-ctx.Done():
						}
					} else if result != nil && result.Success && result.UID != "" && interactionCfg.VerifyEnabled {
						// NON-SPLIT (verify inline cũ — GIỮ NGUYÊN, không đổi).
						// EAA token pre-fetch đã chạy ở block phía trên (trước file write).
						// result.AccessToken tại đây là token EAA mới (nếu fetch thành công).
						onStatus(fmt.Sprintf("[AutoVerify] Start — UID=%s Token=%s Cookie=%s",
							result.UID, result.AccessToken[:min(len(result.AccessToken), 15)], result.Cookie[:min(len(result.Cookie), 15)]))
						acc := runner.AccountInput{
							ID:                    threadIdx,
							UID:                   result.UID,
							Username:              result.Username,
							FullName:              prof.FirstName + " " + prof.LastName,
							Phone:                 prof.Phone,
							Cookie:                result.Cookie,
							Token:                 result.AccessToken,
							UserAgent:             prof.UserAgent,
							Password:              result.Password,
							InputAccount:          formatRegResultLine(result, prof),
							Proxy:                 prof.Proxy,
							DeviceID:              result.DeviceID,
							FamilyDeviceID:        result.FamilyDeviceID,
							Srnonce:               result.Srnonce,
							SessionlessCryptedUID: result.SessionlessCryptedUID,
							// TempMail reuse: forward email + creds nếu RegMode=TempMail.
							// Verify steps detect Session.EmailMeta != "" → Restore +
							// skip CreateEmail + skip "Add email" step (mail đã add lúc reg).
							Email:     prof.Email,
							EmailMeta: prof.EmailMeta,
						}
						// Cập nhật acc.UserAgent sang verify-platform UA per-account (round-robin).
						// Phải làm trước verifyOnStatus để closure emit đúng UA verify.
						{
							verifyUAConfig := applyVerifyPlatformUAConfig(a.LoadInteractionConfig())
							// FIX multi-version: round-robin per-account (trước dùng focus → mọi acc 1 UA).
							vpStr := a.nextVerifyPlatform()
							if vpStr == "" {
								vpStr = verifyPlatformFromType(verifyUAConfig.ApiVerifyPlatform)
							}
							acc.VerifyPlatform = vpStr
							if vUA := pickUAForVerifyPlatform(vpStr, acc.UserAgent, verifyUAConfig, phoneToCountryCode(acc.Phone)); vUA != "" {
								acc.UserAgent = vUA
							}
						}
						// Delay trước verify (DelayVeriReg) — account cần "settle" trên server FB.
						// Ctx = run-scoped (RunRegister cancel) → Stop register thoát sleep ngay,
						// thay vì chờ hết DelayVeriReg (có thể 30-60s).
						if interactionCfg.DelayVeriReg > 0 {
							if interactionCfg.SplitMode {
								// Split UI: log "Chờ Xs" vào VERIFY panel (cột HOẠT ĐỘNG row mới chưa có).
								runtime.EventsEmit(a.ctx, "verify:batch-status", []map[string]interface{}{{
									"accountId": threadIdx,
									"message":   fmt.Sprintf("Chờ %ds trước khi verify...", interactionCfg.DelayVeriReg),
								}})
							} else {
								onStatus(fmt.Sprintf("[AutoVerify] Chờ %ds trước khi verify...", interactionCfg.DelayVeriReg))
							}
							select {
							case <-ctx.Done():
							case <-time.After(time.Duration(interactionCfg.DelayVeriReg) * time.Second):
							}
						}

						verifyCfg := &instagram.VerifyConfig{
							UserApiLabel:  interactionCfg.ApiVerifyPlatform,
							VerifyEnabled: true,
							MailProvider:  interactionCfg.MailProvider,
							MailList:      interactionCfg.MailList,
							CheckLiveDie:  interactionCfg.CheckLiveDieEnabled,
							TimeDelayCheck: func() int {
								if interactionCfg.DelayCheckLive > 0 {
									return interactionCfg.DelayCheckLive
								}
								return interactionCfg.TimeDelayCheck
							}(),
							TimeDelaySendCode:       interactionCfg.TimeDelaySendCode,
							DelayConfirmEmail:       interactionCfg.DelayConfirmEmail,
							DelayVeriReg:            interactionCfg.DelayVeriReg,
							WaitMailMs:              interactionCfg.WaitMail * 1000, // UI = giây → ms
							SendAgainCode:           interactionCfg.SendAgainCode,
							OutputPath:              interactionCfg.OutputPath,
							UAIphoneList:            "",
							ZeusXApiKey:             interactionCfg.ZeusXApiKey,
							ZeusXAccountCode:        interactionCfg.ZeusXAccountCode,
							DvfbApiKey:              interactionCfg.DvfbApiKey,
							DvfbAccountType:         interactionCfg.DvfbAccountType,
							Store1sApiKey:           interactionCfg.Store1sApiKey,
							Store1sProductID:        interactionCfg.Store1sProductID,
							Mail30sApiKey:           interactionCfg.Mail30sApiKey,
							Mail30sProductSlug:      interactionCfg.Mail30sProductSlug,
							TempMailLolApiKey:       interactionCfg.TempMailLolApiKey,
							TempMailDomain:          interactionCfg.TempMailDomain,
							MuaMailApiKey:           interactionCfg.MuaMailApiKey,
							MuaMailProductID:        interactionCfg.MuaMailProductID,
							UnlimitMailApiKey:       interactionCfg.UnlimitMailApiKey,
							UnlimitMailProductID:    interactionCfg.UnlimitMailProductID,
							SptMailApiKey:           interactionCfg.SptMailApiKey,
							SptMailServiceCode:      interactionCfg.SptMailServiceCode,
							EmailAPIInfoApiKey:      interactionCfg.EmailAPIInfoApiKey,
							EmailAPIInfoProductCode: interactionCfg.EmailAPIInfoProductCode,
							OtpCheapApiKey:          interactionCfg.OtpCheapApiKey,
							OtpCheapServiceID:       interactionCfg.OtpCheapServiceID,
							ShopGmail9999ApiKey:     interactionCfg.ShopGmail9999ApiKey,
							ShopGmail9999Service:    interactionCfg.ShopGmail9999Service,
							RentGmailApiKey:         interactionCfg.RentGmailApiKey,
							RentGmailPlatform:       interactionCfg.RentGmailPlatform,
							OtpCodesSmsApiKey:       interactionCfg.OtpCodesSmsApiKey,
							OtpCodesSmsServiceID:    interactionCfg.OtpCodesSmsServiceID,
							WmemailApiKey:           interactionCfg.WmemailApiKey,
							WmemailCommodity:        interactionCfg.WmemailCommodity,
							PriyoEmailApiKey:        interactionCfg.PriyoEmailApiKey,
							OTPHotmailPriority:      interactionCfg.OTPHotmailPriority,
							TempMailToken:           interactionCfg.TempMailToken,
							EmailPool:               a.emailPool,
							DeepFakeLocale:          a.LoadSettings().General.DeepFakeInApi,
							ReUseEmail:              interactionCfg.ReUseEmail,
							UseEmailTime:            interactionCfg.UseEmailTime,
							FmUserTmpMail:           interactionCfg.FmUserTmpMail,
							UseProxyTempMail:        interactionCfg.UseProxyTempMail,
							UseProxyGmail:           interactionCfg.UseProxyGmail,
							Enable2FA:               interactionCfg.Enable2FA,
							AddInfo: &instagram.AddInfoConfig{
								Enabled:      interactionCfg.AddInfo,
								City:         interactionCfg.AddInfoCity,
								Hometown:     interactionCfg.AddInfoHometown,
								School:       interactionCfg.AddInfoSchool,
								College:      interactionCfg.AddInfoCollege,
								Work:         interactionCfg.AddInfoWork,
								Relationship: interactionCfg.AddInfoRelationship,
								DataDir:      interactionCfg.AddInfoDataDir,
								DelayMs:      interactionCfg.AddInfoDelayMs,
							},
						}
						// GetVerifyConfig: reload config mỗi account để user đổi mail provider
						// giữa chừng có hiệu lực ngay (port pattern từ runCfg dòng 1487).
						getLatestVerifyCfg := func() *instagram.VerifyConfig {
							latest := a.LoadInteractionConfig()
							return &instagram.VerifyConfig{
								UserApiLabel:  latest.ApiVerifyPlatform,
								VerifyEnabled: true,
								MailProvider:  latest.MailProvider,
								MailList:      latest.MailList,
								CheckLiveDie:  latest.CheckLiveDieEnabled,
								TimeDelayCheck: func() int {
									if latest.DelayCheckLive > 0 {
										return latest.DelayCheckLive
									}
									return latest.TimeDelayCheck
								}(),
								TimeDelaySendCode:       latest.TimeDelaySendCode,
								DelayConfirmEmail:       latest.DelayConfirmEmail,
								DelayVeriReg:            latest.DelayVeriReg,
								WaitMailMs:              latest.WaitMail * 1000, // UI = giây → ms
								SendAgainCode:           latest.SendAgainCode,
								OutputPath:              latest.OutputPath,
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
								EmailPool:               a.emailPool,
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
									Relationship: latest.AddInfoRelationship,
									DataDir:      latest.AddInfoDataDir,
									DelayMs:      latest.AddInfoDelayMs,
								},
							}
						}
						runCfg := runner.RunConfig{
							VerifyConfig:    verifyCfg,
							VerifyPlatform:  verifyPlatformFromType(interactionCfg.ApiVerifyPlatform),
							GetVerifyConfig: getLatestVerifyCfg,
							GetVerifyPlatform: func() string {
								return a.nextVerifyPlatform()
							},
							GetUseOriginalUA: func() bool {
								latest := a.LoadInteractionConfig()
								if uaCfg, ok := latest.VerifyPlatformUA[latest.ApiVerifyPlatform]; ok {
									return uaCfg.UseOriginalUA
								}
								return latest.UseOriginalUA
							},
							AddMailRetry: interactionCfg.AddMailRetry,
							// OnEmailCreated: cập nhật cột EMAIL/PHONE trên bảng REG ngay khi
							// verify tạo/lấy được temp mail — không cần chờ verify:account-done.
							// Dùng register:status với phone=email để ghi đè cột hiện tại.
							OnEmailCreated: func(_ int, email string) {
								if interactionCfg.SplitMode {
									// Split UI: email update vào VERIFY panel (cột EMAIL).
									runtime.EventsEmit(a.ctx, "verify:email", map[string]interface{}{
										"accountId": threadIdx,
										"email":     email,
									})
								} else {
									// Normal mode: hiện mail đang verify ở cột EMAIL của REG panel.
									// Dùng verify:email (field verifyEmail riêng) → KHÔNG bị các status
									// verify sau (phone=pickContact) ghi đè như khi nhét vào field phone.
									runtime.EventsEmit(a.ctx, "verify:email", map[string]interface{}{
										"accountId": threadIdx,
										"email":     email,
									})
									// Activity msg riêng (UA = VER UA — FB4A).
									runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
										"index":     threadIdx,
										"proxy":     displayProxy,
										"userAgent": acc.UserAgent,
										"msg":       "[Verify] Mail: " + email,
									})
								}
							},
						}
						verifyOnStatus := func(accountID int, uid string, msg string) {
							if interactionCfg.SplitMode {
								// Split UI: verify events đi VERIFY panel (dưới), KHÔNG hiển trong REG panel (trên).
								// → REG panel chỉ thấy events reg, VERIFY panel chỉ thấy events verify.
								runtime.EventsEmit(a.ctx, "verify:batch-status", []map[string]interface{}{{
									"accountId": threadIdx,
									"message":   msg,
								}})
							} else {
								// Normal mode: verify events hiện trong cùng REG panel (1 panel chung).
								// UA: dùng acc.UserAgent (VER UA — FB4A) → cột UA hiện đúng UA verify
								// đang gửi request. Trước đây dùng prof.UserAgent (REG UA — Chrome) →
								// user thấy Chrome UA dù verify đang gọi /auth/login bằng FB4A UA.
								runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
									"index":     threadIdx,
									"phone":     pickContact(),
									"proxy":     displayProxy,
									"userAgent": acc.UserAgent,
									"msg":       "[Verify] " + msg,
								})
							}
						}
						// Split UI: KHÔNG emit register:status cho verify-start (REG panel chỉ thấy reg).
						// Normal mode: vẫn emit để REG panel hiện status verify (1 panel chung).
						if interactionCfg.SplitMode {
							// VERIFY panel: tạo row mới với uid/password/phone/status/UA/token/cookie
							runtime.EventsEmit(a.ctx, "verify:slot-assigned", map[string]interface{}{
								"slotId":    threadIdx,
								"uid":       acc.UID,
								"password":  acc.Password,
								"phone":     acc.Phone,
								"status":    "new",
								"userAgent": acc.UserAgent,
								"token":     acc.Token,
								"cookie":    acc.Cookie,
							})
							// VERIFY panel: set PROXY column (full proxy string)
							runtime.EventsEmit(a.ctx, "verify:raw-proxy", map[string]interface{}{
								"accountId": threadIdx,
								"proxy":     prof.Proxy,
							})
							// VERIFY panel: set IP CHẠY column (display proxy = IP rendered từ session)
							if displayProxy != "" {
								runtime.EventsEmit(a.ctx, "verify:proxy", map[string]interface{}{
									"accountId": threadIdx,
									"proxy":     displayProxy,
								})
							}
							// VERIFY panel: nếu có email từ reg → set EMAIL column
							if prof.Email != "" {
								runtime.EventsEmit(a.ctx, "verify:email", map[string]interface{}{
									"accountId": threadIdx,
									"email":     prof.Email,
								})
							}
						} else {
							// Normal mode: REG panel hiện verify start status (như cũ)
							runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
								"index":     threadIdx,
								"phone":     pickContact(),
								"proxy":     displayProxy,
								"userAgent": acc.UserAgent,
								"msg":       "[Verify] Bắt đầu...",
							})
						}
						// verifySem — Split mode: giới hạn verify đồng thời ≤ SplitVerifyThreads.
						// REG worker phải acquire permit trước khi verify; release sau khi xong.
						// Khi >SplitVerifyThreads worker muốn verify cùng lúc → block chờ permit.
						if verifySem != nil {
							select {
							case verifySem <- struct{}{}:
								defer func() { <-verifySem }()
							case <-regWorkerCtx.Done():
							}
						}
						// iOS Mess reg phone-only → acquire tempmail Ở VER (chỉ account create OK).
						if mh := a.acquireIOSMessVerMail(regWorkerCtx, *verifyCfg, prof.Proxy, acc.VerifyPlatform, acc.Email, func(m string) { verifyOnStatus(threadIdx, acc.UID, m) }); mh != nil {
							acc.Email = mh.Email
							acc.EmailMeta = mh.Meta
							defer mh.Close()
						}
						// regWorkerCtx — không bị cancel khi Stop; đảm bảo acc đã đăng ký thành công
						// chạy hết auto-verify + live/die check trước khi báo xong (matching tally).
						// Pass threadIdx làm workerID → sticky manager phân biệt cache per thread slot.
						vr := runner.RunOneAccountAt(regWorkerCtx, acc, runCfg, autoVerifyDateFolder, threadIdx, verifyOnStatus)
						verifyStatus = strings.ToLower(vr.Status)
						verifyEmail = vr.Email
						verifyMessage = vr.Message
						verifyToken = vr.Token
						verifyCookie = vr.Cookie
						if verifyStatus != "" {
							vp := vr.VerifyPlatform
							if vp == "" {
								vp = verifyPlatformFromType(interactionCfg.ApiVerifyPlatform)
							}
							a.recordVerifyOutcome(vp, verifyStatus == "live")
							a.recordBuildUAVerVersion(extractFBAV(vr.UserAgent), verifyStatus == "live")
							if vr.Email != "" {
								a.recordMailDomainOutcome(vr.Email, verifyStatus == "live")
							}
							// Learning loop: verify live qua PlatformWeb (api mfb) → thêm UA vào WebChrome pool.
							if verifyStatus == "live" && vp == instagram.PlatformWeb && vr.UserAgent != "" {
								if fakeinfo.AppendUAToPool(fakeinfo.UAKindWebChrome, vr.UserAgent) {
									slog.Info("WebChrome pool learned new UA (auto)", "ua_prefix", vr.UserAgent[:min(len(vr.UserAgent), 50)])
								}
							}
						}
						// Split UI: emit verify:account-done để bottom panel update status cuối (live/die/...).
						if interactionCfg.SplitMode {
							runtime.EventsEmit(a.ctx, "verify:account-done", map[string]interface{}{
								"accountId": threadIdx,
								"uid":       vr.UID,
								"status":    verifyStatus,
								"message":   verifyMessage,
								"token":     vr.Token,
								"cookie":    vr.Cookie,
							})
						}

						// FIX: Inline AutoVerify chưa save kết quả verify ra file
						// (chỉ split mode + RunVerify mới có save). Bug: SuccessVerify.txt /
						// Die.txt không được tạo khi user chạy Reg+Ver inline.
						// Reuse regWriter (cùng outputPath) để ghi verify result chung folder.
						if regWriter != nil && vr.UID != "" {
							verifyAcc := Account{
								UID:      vr.UID,
								Password: result.Password,
								Twofa:    vr.TwoFA,
								Cookie:   result.Cookie,
								// Token: ưu tiên EAA fetched trong scheduler (vr.Token),
								// fallback về reg result token.
								Token: func() string {
									if vr.Token != "" {
										return vr.Token
									}
									return result.AccessToken
								}(),
								Email:     vr.Email,
								FullName:  prof.FirstName + " " + prof.LastName,
								UserAgent: vr.UserAgent,
							}
							verifyInstance := verifyPlatformFromType(interactionCfg.ApiVerifyPlatform)
							go saveVerifyOutcome(regWriter, regCounters, verifyStatus, vr.Message, verifyAcc, verifyInstance)

							if verifyStatus == "live" {
								// Auto-upload lên site sau inline verify live
								if uploadCfg := a.LoadUploadSiteConfig(); uploadCfg.Ver.Enabled && uploadCfg.Code != "" && uploadCfg.ApiKey != "" {
									country := ""
									if loc := extractFBLocaleFromUA(verifyAcc.UserAgent); len(loc) >= 5 && loc[2] == '_' {
										country = loc[3:]
									}
									var uploadLine string
									if strings.TrimSpace(verifyAcc.Twofa) == "" {
										uploadLine = resultpkg.FormatReg(resultpkg.RegData{
											UID: verifyAcc.UID, Password: verifyAcc.Password,
											Cookie: verifyAcc.Cookie, Token: verifyAcc.Token,
											Email: "", Country: country, // user: không upload email
										}, nil)
									} else {
										uploadLine = resultpkg.FormatVerify(resultpkg.VerifyData{
											UID: verifyAcc.UID, Password: verifyAcc.Password,
											TwoFA: verifyAcc.Twofa, Cookie: verifyAcc.Cookie,
											Token: verifyAcc.Token, Email: "", // user: không upload email
											Country: country,
										}, nil)
									}
									a.ensureUploadRunning(uploadCfg)
									a.enqueueForUpload(uploadLine)
								}
							}
						}
					}
					isCheckpoint := !result.Success && strings.Contains(strings.ToLower(result.Message), "checkpoint")
					// KeepUASuccess: pin UA cho slot khi thành công, xoá khi thất bại.
					if interactionCfg.KeepUASuccess {
						if result.Success {
							regUABySlot.Store(slotIdx, regSlotUA{Platform: regPlatform, UA: prof.UserAgent})
						} else {
							regUABySlot.Delete(slotIdx)
						}
					}
					if interactionCfg.KeepDatrSuccess {
						if result.Success && result.Cookie != "" {
							if datr := extractDatrFromCookieLine(result.Cookie); datr != "" && !strings.HasPrefix(datr, "_") && !strings.HasPrefix(datr, "-") {
								regDatrBySlot.Store(slotIdx, datr)
							} else {
								regDatrBySlot.Delete(slotIdx)
							}
						} else {
							regDatrBySlot.Delete(slotIdx)
						}
					}
					// Sticky: success → giữ proxy cho reg kế (cùng slot), fail → thả về pool.
					// Silent release — không emit status msg (user không cần thấy từng lần thả/giữ).
					if regStickyRelease != nil {
						regStickyRelease(result.Success)
						regStickyRelease = nil
					}
					// Thống kê REG theo version — đếm CÙNG thời điểm với register:account-done
					// để tổng số trong tab "Thống kê REG" luôn khớp với bảng register (Live/Die).
					a.recordRegOutcome(regPlatform, result.Success)
					a.recordBuildUARegVersion(extractFBAV(prof.UserAgent), result.Success)
					runtime.EventsEmit(a.ctx, "register:account-done", map[string]interface{}{
						"index":       threadIdx,
						"phone":       pickContact(),
						"proxy":       displayProxy,
						"proxyServer": prof.Proxy,
						"userAgent":   prof.UserAgent,
						"success":     result.Success,
						"uid":         result.UID,
						"cookie": func() string {
							if verifyCookie != "" {
								return verifyCookie
							}
							return result.Cookie
						}(),
						"password": result.Password,
						"token": func() string {
							if verifyToken != "" {
								return verifyToken
							}
							return result.AccessToken
						}(),
						"message":       result.Message,
						"verifyStatus":  verifyStatus,
						"verifyMessage": verifyMessage,
						"verifyEmail":   verifyEmail,
						"checkpoint":    isCheckpoint,
					})

					if needBreak {
						break // fail/error/checkpoint → end loop → goroutine exits
					}
				} // end for regAttempt loop

				// Safety: nếu release chưa fire (loop break sớm), thả proxy về pool.
				if regStickyRelease != nil {
					regStickyRelease(false)
					regStickyRelease = nil
				}

				// C#: delay giữa mỗi lần reg. Riêng FB hard-block thì bỏ delay để slot
				// nhả ngay, lần kế lấy machine_id/datr khác từ pool thay vì đứng chờ.
				blockedReg := false
				if result != nil && !result.Success {
					msg := strings.ToLower(result.Message)
					blockedReg = strings.Contains(msg, "facebook blocked") ||
						strings.Contains(msg, "account creation denied") ||
						strings.Contains(msg, "integrity_block")
				}
				if interactionCfg.DelayReg > 0 && !blockedReg {
					onStatus(fmt.Sprintf("⏸ Delay %ds trước khi reg luồng kế...", interactionCfg.DelayReg))
					// Cancellable — Stop thoát sớm thay vì chờ DelayReg đầy đủ.
					select {
					case <-time.After(time.Duration(interactionCfg.DelayReg) * time.Second):
					case <-ctx.Done():
						return
					}
				}
			}(slotIdx, profile, accountCfg, accountRegPlatform)
		}
	}()

	// DEPRECATED 2026-05-15: Split Mode = PURE UI option (chỉ hiển thị 2 panel REG/VER).
	// Worker logic GIỐNG Normal Mode (1 worker = REG + VER inline). Block bên dưới
	// (verify pool độc lập + channel regToVerCh) ĐÃ DISABLE bằng `if false`.
	// Giữ code để reference; không xoá vì có biến local + deferred cleanup phụ thuộc.
	if false && interactionCfg.SplitMode && interactionCfg.VerifyEnabled && outputPath != "" {
		verifyThreads := interactionCfg.SplitVerifyThreads
		if verifyThreads <= 0 {
			verifyThreads = maxThreads
		}

		// Split mode: verify ghi chung folder với reg — file names đã khác nhau (SuccessVerify.txt
		// vs SuccessReg.txt, DieAfterVerify.txt vs Blocked.txt) nên không conflict.
		// popAccountFromFolder không chạy trong split mode (worker dùng acc vừa reg, không đọc folder).
		splitVerOutputPath := outputPath

		// Writer + Counters cho split-verify — ghi file SuccessVerify_No2FA.txt / Die.txt / Unknown.txt.
		// Trước đây split OnAccountDone không gọi saveVerifyOutcome → verify live không persist.
		splitVerifyWriter := resultpkg.NewWriter(splitVerOutputPath)
		splitVerifyCounters := resultpkg.NewCounterSet(splitVerifyWriter)
		splitVerifyCounters.Start(ctx, 5*time.Second)
		// KHÔNG `defer splitVerifyCounters.Stop()` ở scope RunRegister — RunRegister return
		// ngay sau khi spawn goroutines (chỉ ~ms) nên defer fire sớm, giết counter trong lúc
		// dispatcher vẫn đang chạy. Stop() được gọi trong dispatcher goroutine khi ctx.Done()
		// để đảm bảo sống cùng tuổi với split pipeline.

		// Build verify config (đọc lại từ interaction để dùng đúng settings verify)
		splitVerifyCfg := &instagram.VerifyConfig{
			UserApiLabel:  interactionCfg.ApiVerifyPlatform,
			VerifyEnabled: true,
			MailProvider:  interactionCfg.MailProvider,
			MailList:      interactionCfg.MailList,
			CheckLiveDie:  interactionCfg.CheckLiveDieEnabled,
			TimeDelayCheck: func() int {
				if interactionCfg.DelayCheckLive > 0 {
					return interactionCfg.DelayCheckLive
				}
				return interactionCfg.TimeDelayCheck
			}(),
			TimeDelaySendCode:       interactionCfg.TimeDelaySendCode,
			DelayConfirmEmail:       interactionCfg.DelayConfirmEmail,
			DelayVeriReg:            interactionCfg.DelayVeriReg,
			WaitMailMs:              interactionCfg.WaitMail * 1000, // UI = giây → ms
			SendAgainCode:           interactionCfg.SendAgainCode,
			OutputPath:              splitVerOutputPath,
			UAIphoneList:            "",
			ZeusXApiKey:             interactionCfg.ZeusXApiKey,
			ZeusXAccountCode:        interactionCfg.ZeusXAccountCode,
			DvfbApiKey:              interactionCfg.DvfbApiKey,
			DvfbAccountType:         interactionCfg.DvfbAccountType,
			Store1sApiKey:           interactionCfg.Store1sApiKey,
			Store1sProductID:        interactionCfg.Store1sProductID,
			Mail30sApiKey:           interactionCfg.Mail30sApiKey,
			Mail30sProductSlug:      interactionCfg.Mail30sProductSlug,
			TempMailLolApiKey:       interactionCfg.TempMailLolApiKey,
			TempMailDomain:          interactionCfg.TempMailDomain,
			MuaMailApiKey:           interactionCfg.MuaMailApiKey,
			MuaMailProductID:        interactionCfg.MuaMailProductID,
			UnlimitMailApiKey:       interactionCfg.UnlimitMailApiKey,
			UnlimitMailProductID:    interactionCfg.UnlimitMailProductID,
			SptMailApiKey:           interactionCfg.SptMailApiKey,
			SptMailServiceCode:      interactionCfg.SptMailServiceCode,
			EmailAPIInfoApiKey:      interactionCfg.EmailAPIInfoApiKey,
			EmailAPIInfoProductCode: interactionCfg.EmailAPIInfoProductCode,
			OtpCheapApiKey:          interactionCfg.OtpCheapApiKey,
			OtpCheapServiceID:       interactionCfg.OtpCheapServiceID,
			ShopGmail9999ApiKey:     interactionCfg.ShopGmail9999ApiKey,
			ShopGmail9999Service:    interactionCfg.ShopGmail9999Service,
			RentGmailApiKey:         interactionCfg.RentGmailApiKey,
			RentGmailPlatform:       interactionCfg.RentGmailPlatform,
			OtpCodesSmsApiKey:       interactionCfg.OtpCodesSmsApiKey,
			OtpCodesSmsServiceID:    interactionCfg.OtpCodesSmsServiceID,
			WmemailApiKey:           interactionCfg.WmemailApiKey,
			WmemailCommodity:        interactionCfg.WmemailCommodity,
			PriyoEmailApiKey:        interactionCfg.PriyoEmailApiKey,
			OTPHotmailPriority:      interactionCfg.OTPHotmailPriority,
			TempMailToken:           interactionCfg.TempMailToken,
			EmailPool:               a.emailPool,
			ReUseEmail:              interactionCfg.ReUseEmail,
			UseEmailTime:            interactionCfg.UseEmailTime,
			FmUserTmpMail:           interactionCfg.FmUserTmpMail,
			UseProxyTempMail:        interactionCfg.UseProxyTempMail,
			UseProxyGmail:           interactionCfg.UseProxyGmail,
			Enable2FA:               interactionCfg.Enable2FA,
			AddInfo: &instagram.AddInfoConfig{
				Enabled:      interactionCfg.AddInfo,
				City:         interactionCfg.AddInfoCity,
				Hometown:     interactionCfg.AddInfoHometown,
				School:       interactionCfg.AddInfoSchool,
				Relationship: interactionCfg.AddInfoRelationship,
				DataDir:      interactionCfg.AddInfoDataDir,
				DelayMs:      interactionCfg.AddInfoDelayMs,
			},
		}

		splitVerifyPlatform := verifyPlatformFromType(interactionCfg.ApiVerifyPlatform)

		// splitWorkerCtx — context riêng cho verify workers, KHÔNG bị cancel khi StopRegister.
		// feedCtx (= ctx) cancel → dừng nhận nick mới; splitWorkerCtx sống đến khi
		// verWg.Wait() xong để verify workers hoàn tất nick hiện tại + queue còn lại.
		splitWorkerCtx, splitWorkerCancel := context.WithCancel(a.ctx)

		// LEGACY (đã thay): trước dùng splitPickUA pick từ active pool, gán vào
		// displayUA nhưng KHÔNG truyền vào verify request (UserAgent="").
		// Hiện tại split verify dùng pickUAForVerifyPlatform → pick UA chuẩn theo
		// platform (S23 → Android pool, MFB → Chrome Desktop pool…) → truyền
		// thẳng vào inp.UserAgent → FB nhận đúng UA match fingerprint.

		// Tạo N slot verify trong a.accounts — reset sạch trước để tránh cộng dồn khi chạy lại
		a.accountsMu.Lock()
		a.accounts = make([]Account, 0, verifyThreads)
		for i := 0; i < verifyThreads; i++ {
			a.accounts = append(a.accounts, Account{
				ID:       i + 1,
				Status:   "waiting",
				Activity: "[Ver] Đang chờ...",
			})
		}
		verSlotBaseID := 1
		a.accountsMu.Unlock()
		runtime.EventsEmit(a.ctx, "verify:accounts-updated", nil)

		freeSlotsVer := make(chan int, verifyThreads)
		for i := 0; i < verifyThreads; i++ {
			freeSlotsVer <- verSlotBaseID + i
		}

		// Split-verify: channel queue → batch drain mỗi 100ms.
		// Channel giữ message ngay khi onStatus gọi (không bị mất dù account xong < 100ms).
		// Batch drain tránh flood WebView IPC (direct emit gây V8 heap drift khi 100+ luồng).
		type splitVerMsg struct {
			accountID int
			message   string
		}
		splitVerCh := make(chan splitVerMsg, 4000)
		// splitVerBatchCtx — parent là splitWorkerCtx (KHÔNG phải ctx) để batch emitter
		// sống qua StopRegister và vẫn emit verify:batch-status trong lúc drain+wait.
		// Khi verWg.Wait() xong → pipeline goroutine unwind → defer splitVerBatchCancel()
		// → batch emitter mới exit. Nếu dùng ctx làm parent: Stop → ctx cancel →
		// batch emitter chết ngay → UI trắng trong lúc drain.
		splitVerBatchCtx, splitVerBatchCancel := context.WithCancel(splitWorkerCtx)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("split-ver batch emitter panic recovered", "panic", r)
				}
			}()
			// Dynamic interval theo visibility — visible 300ms / hidden 2s.
			const intervalVisible = 300 * time.Millisecond
			const intervalHidden = 2 * time.Second
			ticker := time.NewTicker(intervalVisible)
			defer ticker.Stop()
			curInterval := intervalVisible
			latest := make(map[int]string, 64)
			for {
				// Drain channel trước khi tick để không bỏ sót message
				draining := true
				for draining {
					select {
					case m := <-splitVerCh:
						latest[m.accountID] = m.message
					default:
						draining = false
					}
				}
				select {
				case <-ticker.C:
					want := intervalVisible
					if a.frontendHidden.Load() {
						want = intervalHidden
					}
					if want != curInterval {
						ticker.Reset(want)
						curInterval = want
					}
					if len(latest) == 0 {
						continue
					}
					updates := make([]map[string]interface{}, 0, len(latest))
					for id, msg := range latest {
						updates = append(updates, map[string]interface{}{
							"accountId": id,
							"message":   msg,
						})
					}
					clear(latest)
					runtime.EventsEmit(a.ctx, "verify:batch-status", updates)
				case m := <-splitVerCh:
					latest[m.accountID] = m.message
				case <-splitVerBatchCtx.Done():
					return
				}
			}
		}()
		// KHÔNG `defer splitVerBatchCancel()` ở scope RunRegister — RunRegister return
		// ngay sau khi spawn goroutines → defer cancel batch emitter ngay, khiến splitVerCh
		// không còn ai drain → toàn bộ message verify:batch-status bị drop → cột HOẠT ĐỘNG
		// pane VER trắng suốt run. Cancel được chuyển vào dispatcher goroutine (ctx.Done()).

		splitVerOnStatus := func(accountID int, uid string, msg string) {
			select {
			case splitVerCh <- splitVerMsg{accountID, msg}:
			default: // drop nếu buffer đầy — tránh block goroutine verify
			}
		}

		splitVerOnAccountDone := func(accountID int, uid string, status string, message string, email string, userAgent string, twoFA string, token string, cookie string, verifyPlatform string) {
			s := strings.ToLower(status)
			if s == "" {
				s = "unknown"
			}
			if verifyPlatform == "" {
				verifyPlatform = string(splitVerifyPlatform)
			}
			a.recordVerifyOutcome(verifyPlatform, s == "live")
			a.recordBuildUAVerVersion(extractFBAV(userAgent), s == "live")
			if email != "" {
				a.recordMailDomainOutcome(email, s == "live")
			}
			// Learning loop: verify live qua PlatformWeb (api mfb) → thêm UA vào WebChrome pool.
			if s == "live" && verifyPlatform == instagram.PlatformWeb && userAgent != "" {
				if fakeinfo.AppendUAToPool(fakeinfo.UAKindWebChrome, userAgent) {
					slog.Info("WebChrome pool learned new UA (split)", "ua_prefix", userAgent[:min(len(userAgent), 50)])
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
					if cookie != "" {
						a.accounts[i].Cookie = cookie // cookie MỚI từ login verify → vào doneAcc + file
					}
					doneAcc = a.accounts[i]
					// Clear heavy fields để giải phóng RAM (file mode 1M+ accounts).
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
				"accountId": accountID, "uid": uid, "status": s, "message": message, "token": token, "cookie": cookie,
			})
			a.activityCache.Delete(accountID)

			// Ghi file SuccessVerify_No2FA.txt / Die.txt / Unknown.txt (split mode cũng cần persist).
			if doneAcc.UID != "" {
				go saveVerifyOutcome(splitVerifyWriter, splitVerifyCounters, s, message, doneAcc, string(splitVerifyPlatform))
			}
			if s == "live" {
				if interactionCfg.UploadAvatar && doneAcc.Token != "" {
					avatarDir := interactionCfg.AvatarFolderPath
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
						emitActivity("[UpAVT] Đang upload avatar...")
						// Parent = splitWorkerCtx — sống qua StopRegister để upload hoàn tất.
						avtCtx, avtCancel := context.WithTimeout(splitWorkerCtx, 60*time.Second)
						defer avtCancel()
						if err := uploadavatar.UploadAvatarS23(avtCtx, proxyForAvt, tokenForAvt, uaForAvt, avatarDir); err != nil {
							slog.Warn("UploadAvatar (split) failed", "uid", uidForAvt, "err", err)
							emitActivity("[UpAVT] Lỗi: " + err.Error())
						} else {
							slog.Info("UploadAvatar (split) OK", "uid", uidForAvt)
							emitActivity("[UpAVT] Upload avatar thành công ✓")
						}
					}()
				}

				// Auto-upload lên banclone.pro sau split-verify live — kiểm tra Ver.Enabled trong uploadsite config
				if uploadCfg := a.LoadUploadSiteConfig(); uploadCfg.Ver.Enabled && uploadCfg.Code != "" && uploadCfg.ApiKey != "" {
					{
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
						_ = filter
					}
				}
			}
		}

		// splitProxyMgr — biến này chỉ dùng cho IsConfigured check ban đầu.
		// AcquireProxy bên dưới gọi lại getSharedProxyManager() để realtime reload
		// khi user update proxy list giữa chừng run.
		splitProxyMgr := a.getSharedProxyManager()

		// Sticky proxy per worker slot cho split-verify loop (port C# KeepIPSuccess).
		splitVerifySticky := proxy.NewStickyManager(interactionCfg.KeepIPSuccess, func(c context.Context) (string, func(), error) {
			mgr := a.getSharedProxyManager()
			p, rel, err := mgr.Acquire(c)
			if err != nil {
				slog.Warn("SplitVER AcquireProxy lỗi", "err", err)
				return "", nil, err
			}
			p = proxy.RenderSessionIfIsProxyServer(p)
			return p, rel, nil
		})

		// GetVerifyConfig: reload config mỗi account để split verify pool
		// pick up mail provider mới khi user đổi giữa batch.
		splitGetVerifyCfg := func() *instagram.VerifyConfig {
			latest := a.LoadInteractionConfig()
			return &instagram.VerifyConfig{
				VerifyEnabled: true,
				MailProvider:  latest.MailProvider,
				MailList:      latest.MailList,
				CheckLiveDie:  latest.CheckLiveDieEnabled,
				TimeDelayCheck: func() int {
					if latest.DelayCheckLive > 0 {
						return latest.DelayCheckLive
					}
					return latest.TimeDelayCheck
				}(),
				TimeDelaySendCode:       latest.TimeDelaySendCode,
				DelayConfirmEmail:       latest.DelayConfirmEmail,
				DelayVeriReg:            latest.DelayVeriReg,
				WaitMailMs:              latest.WaitMail * 1000, // UI = giây → ms
				SendAgainCode:           latest.SendAgainCode,
				OutputPath:              splitVerOutputPath,
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
				EmailPool:               a.emailPool,
				ReUseEmail:              latest.ReUseEmail,
				UseEmailTime:            latest.UseEmailTime,
				FmUserTmpMail:           latest.FmUserTmpMail,
				UseProxyTempMail:        latest.UseProxyTempMail,
				UseProxyGmail:           latest.UseProxyGmail,
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
		}
		splitVerRunCfg := runner.RunConfig{
			VerifyConfig:   splitVerifyCfg,
			VerifyPlatform: splitVerifyPlatform,
			GetVerifyPlatform: func() string {
				return a.nextVerifyPlatform()
			},
			OutputPath:         splitVerOutputPath,
			WorkerCtx:          splitWorkerCtx,
			DelayAfterResultMs: interactionCfg.DelayDisplayResult * 1000,
			RequireProxy:       splitProxyMgr.IsConfigured(),
			AcquireProxy:       splitVerifySticky.Acquire,
			AddMailRetry:       interactionCfg.AddMailRetry,
			GetVerifyConfig:    splitGetVerifyCfg,
			OnRawProxy: func(accountID int, proxyStr string) {
				runtime.EventsEmit(a.ctx, "verify:raw-proxy", map[string]interface{}{
					"accountId": accountID, "proxy": proxyStr,
				})
			},
			OnProxy: func(accountID int, proxyStr string) {
				// Extract country từ proxy suffix (vd "1.2.3.4/id" → "id") → set Location cho save.
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
					"accountId": accountID, "proxy": proxyStr,
				})
			},
			// Emit email ngay khi tạo → UI cột EMAIL/PHONE hiện realtime
			OnEmailCreated: func(accountID int, email string) {
				a.accountsMu.Lock()
				for i := range a.accounts {
					if a.accounts[i].ID == accountID {
						a.accounts[i].Email = email
						break
					}
				}
				a.accountsMu.Unlock()
				runtime.EventsEmit(a.ctx, "verify:email", map[string]interface{}{
					"accountId": accountID, "email": email,
				})
			},
			OnAccountDone: splitVerOnAccountDone,
			GetUseOriginalUA: func() bool {
				latest := a.LoadInteractionConfig()
				if uaCfg, ok := latest.VerifyPlatformUA[latest.ApiVerifyPlatform]; ok {
					return uaCfg.UseOriginalUA
				}
				return latest.UseOriginalUA
			},
		}

		// retryRunCfg: giống splitVerRunCfg nhưng IsRetry=true để retry từ Unknown.txt
		// tạo/add mail mới thay vì reuse mail cũ đã fail.
		retryRunCfg := splitVerRunCfg
		retryRunCfg.IsRetry = true

		unknownFilePath := filepath.Join(splitVerOutputPath, "Unknown.txt")

		go func() {
			// Giải phóng splitWorkerCtx khi goroutine thoát (sau verWg.Wait()).
			defer splitWorkerCancel()
			defer func() {
				if r := recover(); r != nil {
					slog.Error("split-ver pipeline panic recovered", "panic", r)
					splitVerBatchCancel()
					splitVerifyCounters.Stop()
				}
			}()
			var verWg sync.WaitGroup
			totalVerRead := 0

			// Cleanup cuối đời split pipeline — chạy khi dispatcher thoát (ctx.Done()).
			// Đặt ở đây thay vì scope RunRegister để defer không fire quá sớm (RunRegister
			// return trong ~ms sau khi spawn goroutines).
			defer splitVerifySticky.ReleaseAll()
			defer splitVerBatchCancel()
			defer splitVerifyCounters.Stop()

			// unknownTicker: retry Unknown.txt định kỳ 5s (fallback, không phải hot path)
			unknownTicker := time.NewTicker(5 * time.Second)
			defer unknownTicker.Stop()

			startVerWorker := func(slotID int, line string, outPath string, cfg runner.RunConfig) {
				acc := autoDetectAccount(line)
				if acc.UID == "" {
					freeSlotsVer <- slotID
					return
				}
				totalVerRead++
				acc.ID = slotID
				acc.FullData = line
				acc.Status = "new"
				acc.ImportTime = time.Now().Format("2006/01/02 15:04")
				if cfg.IsRetry {
					acc.Email = ""
					acc.EmailMeta = ""
					acc.SourceCode = fmt.Sprintf("SplitVer Retry #%d", totalVerRead)
				} else {
					acc.SourceCode = fmt.Sprintf("SplitVer #%d", totalVerRead)
				}
				// UA chuẩn theo verify API platform (S23 → Samsung Android UA, MFB → Chrome
				// Desktop UA, Token → iPhone, etc.). KHÔNG để rỗng — FB log "no UA" thành
				// Chrome Desktop default → fingerprint mismatch.
				// Ưu tiên acc.UserAgent (nếu file import có UA), fallback pool tương ứng.
				verifyUAConfig := applyVerifyPlatformUAConfig(a.LoadInteractionConfig())
				// FIX multi-version: round-robin per-account (trước dùng focus → mọi acc 1 UA).
				currentVerifyPlatform := a.nextVerifyPlatform()
				if currentVerifyPlatform == "" {
					currentVerifyPlatform = verifyPlatformFromType(verifyUAConfig.ApiVerifyPlatform)
				}
				if currentVerifyPlatform == "" {
					currentVerifyPlatform = string(splitVerifyPlatform)
				}
				verifyUA := pickUAForVerifyPlatform(currentVerifyPlatform, acc.UserAgent, verifyUAConfig, phoneToCountryCode(acc.Phone))
				acc.UserAgent = verifyUA

				a.accountsMu.Lock()
				for i := range a.accounts {
					if a.accounts[i].ID == slotID {
						a.accounts[i] = acc
						break
					}
				}
				a.accountsMu.Unlock()

				emailMeta := resultpkg.ParseEmailMetaFromLine(line)
				if cfg.IsRetry {
					emailMeta = ""
				}
				inp := runner.AccountInput{
					ID: slotID, UID: acc.UID, Username: acc.Username, FullName: acc.FullName,
					Phone: acc.Phone, Cookie: acc.Cookie, Token: acc.Token,
					UserAgent:      verifyUA,
					Password:       acc.Password,
					InputAccount:   line,
					VerifyPlatform: currentVerifyPlatform, // scheduler dùng đúng version (UA khớp)
					// Split-ver: parse EmailMeta từ saved reg line để Restore mail.
					Email:                 acc.Email,
					EmailMeta:             emailMeta,
					Srnonce:               acc.Srnonce,
					SessionlessCryptedUID: acc.SessionlessCryptedUID,
				}
				verWg.Add(1)
				go func() {
					defer func() { verWg.Done(); freeSlotsVer <- slotID }()
					// iOS Mess reg phone-only → acquire tempmail Ở VER (account chưa có email).
					if cfg.VerifyConfig != nil {
						if mh := a.acquireIOSMessVerMail(splitWorkerCtx, *cfg.VerifyConfig, acc.Proxy, inp.VerifyPlatform, inp.Email, func(m string) { splitVerOnStatus(slotID, acc.UID, m) }); mh != nil {
							inp.Email = mh.Email
							inp.EmailMeta = mh.Meta
							defer mh.Close()
						}
					}
					// splitWorkerCtx — sống qua StopRegister (ctx cancel) đến khi verWg.Wait()
					// xong. Stop register chỉ dừng nhận nick mới; workers hiện tại + queue
					// drain tiếp đến hết rồi mới thoát.
					// Pass slotID làm workerID → sticky manager phân biệt cache per slot.
					runner.RunOneAccountAt(splitWorkerCtx, inp, cfg, outPath, slotID, splitVerOnStatus)
				}()
				// Slim event — kèm UA cho cột UA grid (giờ là verifyUA — đã gửi vào request).
				runtime.EventsEmit(a.ctx, "verify:slot-assigned", map[string]interface{}{
					"slotId":    slotID,
					"uid":       acc.UID,
					"password":  acc.Password,
					"phone":     acc.Phone,
					"status":    "new",
					"userAgent": verifyUA,
					"token":     acc.Token,
					"cookie":    acc.Cookie,
				})
			}

		mainLoop:
			for {
				select {
				case <-ctx.Done():
					break mainLoop

				case line := <-regToVerCh:
					// Chờ slot rảnh rồi spawn verify worker
					select {
					case slotID := <-freeSlotsVer:
						startVerWorker(slotID, line, splitVerOutputPath, splitVerRunCfg)
					case <-ctx.Done():
						// ctx cancel trong khi đợi slot — trả line về channel (best-effort)
						select {
						case regToVerCh <- line:
						default:
							slog.Warn("regToVerCh full on ctx cancel, line dropped")
						}
						break mainLoop
					}

				case <-unknownTicker.C:
					// Retry Unknown.txt định kỳ — non-blocking
					for {
						select {
						case slotID := <-freeSlotsVer:
							line := a.popFromFile(unknownFilePath)
							if line == "" {
								freeSlotsVer <- slotID
								goto nextUnknownTick
							}
							startVerWorker(slotID, line, splitVerOutputPath, retryRunCfg)
						default:
							goto nextUnknownTick
						}
					}
				nextUnknownTick:
				}
			}

			// Reg đã dừng — drain regToVerCh buffer rồi mới chờ verify workers xong
		drainCh:
			for {
				select {
				case line := <-regToVerCh:
					slotID := <-freeSlotsVer
					startVerWorker(slotID, line, splitVerOutputPath, splitVerRunCfg)
				default:
					break drainCh
				}
			}
			// Drain Unknown.txt một lượt khi dừng nhận reg mới. Nếu retry vẫn Unknown,
			// worker sẽ ghi lại Unknown.txt để lần chạy sau còn xử lý tiếp.
			for {
				line := a.popFromFile(unknownFilePath)
				if line == "" {
					break
				}
				slotID := <-freeSlotsVer
				startVerWorker(slotID, line, splitVerOutputPath, retryRunCfg)
			}
			verWg.Wait()
			// Tất cả verify workers đã xong — phát sự kiện để frontend cập nhật trạng thái.
			// Upload queue đã được enqueue per-account ở splitVerOnAccountDone.
			// Event này báo "hoàn tất drain" để UI unlock nút Start lại.
			runtime.EventsEmit(a.ctx, "split-verify:drained", nil)
		}()
	}

	return fmt.Sprintf("Đang chạy %d luồng đăng ký song song (liên tục đến khi dừng)...", maxThreads)
}
