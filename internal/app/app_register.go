// app_register.go — Luồng đăng ký tài khoản (RunRegister), tách từ app.go.
// Di chuyển nguyên hàm — KHÔNG sửa logic.
package app

import (
	"HVRIns/internal/cookie"
	emailrent "HVRIns/internal/email/rent"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	uploadavatar "HVRIns/internal/instagram/interaction/android"
	androidreg "HVRIns/internal/instagram/register/android"
	s23reg "HVRIns/internal/instagram/register/android/s23"
	s399reg "HVRIns/internal/instagram/register/android/s399"
	ios420reg "HVRIns/internal/instagram/register/ios/ios420"
	ios421reg "HVRIns/internal/instagram/register/ios/ios421"
	ios422reg "HVRIns/internal/instagram/register/ios/ios422"
	ios423reg "HVRIns/internal/instagram/register/ios/ios423"
	ios424reg "HVRIns/internal/instagram/register/ios/ios424"
	ios425reg "HVRIns/internal/instagram/register/ios/ios425"
	ios426reg "HVRIns/internal/instagram/register/ios/ios426"
	ios427reg "HVRIns/internal/instagram/register/ios/ios427"
	ios428reg "HVRIns/internal/instagram/register/ios/ios428"
	ios429reg "HVRIns/internal/instagram/register/ios/ios429"
	ios430reg "HVRIns/internal/instagram/register/ios/ios430"
	ios431reg "HVRIns/internal/instagram/register/ios/ios431"
	ios432reg "HVRIns/internal/instagram/register/ios/ios432"
	ios433reg "HVRIns/internal/instagram/register/ios/ios433"
	ios434reg "HVRIns/internal/instagram/register/ios/ios434"
	ios435reg "HVRIns/internal/instagram/register/ios/ios435"
	ios436reg "HVRIns/internal/instagram/register/ios/ios436"
	ios437reg "HVRIns/internal/instagram/register/ios/ios437"
	ios438reg "HVRIns/internal/instagram/register/ios/ios438"
	ios439reg "HVRIns/internal/instagram/register/ios/ios439"
	ios440reg "HVRIns/internal/instagram/register/ios/ios440"
	ios441reg "HVRIns/internal/instagram/register/ios/ios441"
	ios442reg "HVRIns/internal/instagram/register/ios/ios442"
	ios443reg "HVRIns/internal/instagram/register/ios/ios443"
	ios444reg "HVRIns/internal/instagram/register/ios/ios444"
	ios445reg "HVRIns/internal/instagram/register/ios/ios445"
	ios446reg "HVRIns/internal/instagram/register/ios/ios446"
	ios447reg "HVRIns/internal/instagram/register/ios/ios447"
	ios448reg "HVRIns/internal/instagram/register/ios/ios448"
	ios449reg "HVRIns/internal/instagram/register/ios/ios449"
	ios450reg "HVRIns/internal/instagram/register/ios/ios450"
	ios451reg "HVRIns/internal/instagram/register/ios/ios451"
	ios452reg "HVRIns/internal/instagram/register/ios/ios452"
	ios453reg "HVRIns/internal/instagram/register/ios/ios453"
	ios454reg "HVRIns/internal/instagram/register/ios/ios454"
	ios455reg "HVRIns/internal/instagram/register/ios/ios455"
	ios456reg "HVRIns/internal/instagram/register/ios/ios456"
	ios457reg "HVRIns/internal/instagram/register/ios/ios457"
	ios458reg "HVRIns/internal/instagram/register/ios/ios458"
	ios459reg "HVRIns/internal/instagram/register/ios/ios459"
	ios460reg "HVRIns/internal/instagram/register/ios/ios460"
	ios461reg "HVRIns/internal/instagram/register/ios/ios461"
	ios462reg "HVRIns/internal/instagram/register/ios/ios462"
	ios463reg "HVRIns/internal/instagram/register/ios/ios463"
	ios464reg "HVRIns/internal/instagram/register/ios/ios464"
	ios465reg "HVRIns/internal/instagram/register/ios/ios465"
	ios466reg "HVRIns/internal/instagram/register/ios/ios466"
	ios467reg "HVRIns/internal/instagram/register/ios/ios467"
	ios468reg "HVRIns/internal/instagram/register/ios/ios468"
	ios469reg "HVRIns/internal/instagram/register/ios/ios469"
	ios470reg "HVRIns/internal/instagram/register/ios/ios470"
	ios471reg "HVRIns/internal/instagram/register/ios/ios471"
	ios472reg "HVRIns/internal/instagram/register/ios/ios472"
	ios473reg "HVRIns/internal/instagram/register/ios/ios473"
	ios474reg "HVRIns/internal/instagram/register/ios/ios474"
	ios475reg "HVRIns/internal/instagram/register/ios/ios475"
	ios476reg "HVRIns/internal/instagram/register/ios/ios476"
	ios477reg "HVRIns/internal/instagram/register/ios/ios477"
	ios478reg "HVRIns/internal/instagram/register/ios/ios478"
	ios479reg "HVRIns/internal/instagram/register/ios/ios479"
	ios480reg "HVRIns/internal/instagram/register/ios/ios480"
	ios481reg "HVRIns/internal/instagram/register/ios/ios481"
	ios482reg "HVRIns/internal/instagram/register/ios/ios482"
	ios483reg "HVRIns/internal/instagram/register/ios/ios483"
	ios484reg "HVRIns/internal/instagram/register/ios/ios484"
	ios485reg "HVRIns/internal/instagram/register/ios/ios485"
	ios486reg "HVRIns/internal/instagram/register/ios/ios486"
	ios487reg "HVRIns/internal/instagram/register/ios/ios487"
	ios488reg "HVRIns/internal/instagram/register/ios/ios488"
	ios489reg "HVRIns/internal/instagram/register/ios/ios489"
	ios490reg "HVRIns/internal/instagram/register/ios/ios490"
	ios491reg "HVRIns/internal/instagram/register/ios/ios491"
	ios492reg "HVRIns/internal/instagram/register/ios/ios492"
	ios493reg "HVRIns/internal/instagram/register/ios/ios493"
	ios494reg "HVRIns/internal/instagram/register/ios/ios494"
	ios495reg "HVRIns/internal/instagram/register/ios/ios495"
	ios496reg "HVRIns/internal/instagram/register/ios/ios496"
	ios497reg "HVRIns/internal/instagram/register/ios/ios497"
	ios498reg "HVRIns/internal/instagram/register/ios/ios498"
	ios499reg "HVRIns/internal/instagram/register/ios/ios499"
	ios500reg "HVRIns/internal/instagram/register/ios/ios500"
	ios501reg "HVRIns/internal/instagram/register/ios/ios501"
	ios502reg "HVRIns/internal/instagram/register/ios/ios502"
	ios503reg "HVRIns/internal/instagram/register/ios/ios503"
	ios504reg "HVRIns/internal/instagram/register/ios/ios504"
	ios505reg "HVRIns/internal/instagram/register/ios/ios505"
	ios506reg "HVRIns/internal/instagram/register/ios/ios506"
	ios507reg "HVRIns/internal/instagram/register/ios/ios507"
	ios508reg "HVRIns/internal/instagram/register/ios/ios508"
	ios509reg "HVRIns/internal/instagram/register/ios/ios509"
	ios510reg "HVRIns/internal/instagram/register/ios/ios510"
	ios511reg "HVRIns/internal/instagram/register/ios/ios511"
	ios512reg "HVRIns/internal/instagram/register/ios/ios512"
	ios513reg "HVRIns/internal/instagram/register/ios/ios513"
	ios514reg "HVRIns/internal/instagram/register/ios/ios514"
	ios515reg "HVRIns/internal/instagram/register/ios/ios515"
	ios516reg "HVRIns/internal/instagram/register/ios/ios516"
	ios517reg "HVRIns/internal/instagram/register/ios/ios517"
	ios518reg "HVRIns/internal/instagram/register/ios/ios518"
	ios519reg "HVRIns/internal/instagram/register/ios/ios519"
	ios520reg "HVRIns/internal/instagram/register/ios/ios520"
	ios521reg "HVRIns/internal/instagram/register/ios/ios521"
	ios522reg "HVRIns/internal/instagram/register/ios/ios522"
	ios523reg "HVRIns/internal/instagram/register/ios/ios523"
	ios524reg "HVRIns/internal/instagram/register/ios/ios524"
	ios525reg "HVRIns/internal/instagram/register/ios/ios525"
	ios526reg "HVRIns/internal/instagram/register/ios/ios526"
	ios527reg "HVRIns/internal/instagram/register/ios/ios527"
	ios528reg "HVRIns/internal/instagram/register/ios/ios528"
	ios529reg "HVRIns/internal/instagram/register/ios/ios529"
	ios530reg "HVRIns/internal/instagram/register/ios/ios530"
	ios531reg "HVRIns/internal/instagram/register/ios/ios531"
	ios532reg "HVRIns/internal/instagram/register/ios/ios532"
	ios533reg "HVRIns/internal/instagram/register/ios/ios533"
	ios534reg "HVRIns/internal/instagram/register/ios/ios534"
	ios535reg "HVRIns/internal/instagram/register/ios/ios535"
	ios536reg "HVRIns/internal/instagram/register/ios/ios536"
	ios537reg "HVRIns/internal/instagram/register/ios/ios537"
	ios538reg "HVRIns/internal/instagram/register/ios/ios538"
	ios539reg "HVRIns/internal/instagram/register/ios/ios539"
	ios540reg "HVRIns/internal/instagram/register/ios/ios540"
	ios541reg "HVRIns/internal/instagram/register/ios/ios541"
	ios542reg "HVRIns/internal/instagram/register/ios/ios542"
	ios543reg "HVRIns/internal/instagram/register/ios/ios543"
	ios544reg "HVRIns/internal/instagram/register/ios/ios544"
	ios545reg "HVRIns/internal/instagram/register/ios/ios545"
	ios546reg "HVRIns/internal/instagram/register/ios/ios546"
	ios547reg "HVRIns/internal/instagram/register/ios/ios547"
	ios548reg "HVRIns/internal/instagram/register/ios/ios548"
	ios549reg "HVRIns/internal/instagram/register/ios/ios549"
	ios550reg "HVRIns/internal/instagram/register/ios/ios550"
	ios551reg "HVRIns/internal/instagram/register/ios/ios551"
	ios552reg "HVRIns/internal/instagram/register/ios/ios552"
	ios553reg "HVRIns/internal/instagram/register/ios/ios553"
	ios554reg "HVRIns/internal/instagram/register/ios/ios554"
	ios555reg "HVRIns/internal/instagram/register/ios/ios555"
	ios556reg "HVRIns/internal/instagram/register/ios/ios556"
	ios557reg "HVRIns/internal/instagram/register/ios/ios557"
	ios558reg "HVRIns/internal/instagram/register/ios/ios558"
	ios559reg "HVRIns/internal/instagram/register/ios/ios559"
	ios560reg "HVRIns/internal/instagram/register/ios/ios560"
	ios561reg "HVRIns/internal/instagram/register/ios/ios561"
	ios562reg "HVRIns/internal/instagram/register/ios/ios562"
	ios563reg "HVRIns/internal/instagram/register/ios/ios563"
	ios564reg "HVRIns/internal/instagram/register/ios/ios564"
	ioshttpreg "HVRIns/internal/instagram/register/ioshttp"
	webregister "HVRIns/internal/instagram/register/web"
	webandroidreg "HVRIns/internal/instagram/register/webandroid"
	"HVRIns/internal/proxy"
	resultpkg "HVRIns/internal/result"
	"HVRIns/internal/runner"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
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
		case instagram.PlatformAndroid:
			platformTag = "android"
		case instagram.PlatformWebAndroid:
			platformTag = "webandroid"
		case instagram.PlatformIOS:
			platformTag = "ios"
		case instagram.PlatformS23:
			platformTag = "s23"
		case instagram.PlatformS22:
			platformTag = "s22"
		case instagram.PlatformS24:
			platformTag = "s24"
		case instagram.PlatformS25:
			platformTag = "s25"
		case instagram.PlatformS26:
			platformTag = "s26"
		case instagram.PlatformS545, instagram.PlatformS546, instagram.PlatformS547, instagram.PlatformS548, instagram.PlatformS549,
			instagram.PlatformS550, instagram.PlatformS551, instagram.PlatformS552, instagram.PlatformS553, instagram.PlatformS554:
			platformTag = regPlatform
		case instagram.PlatformS557:
			platformTag = "s557"
		case instagram.PlatformS558:
			platformTag = "s558"
		case instagram.PlatformS555:
			platformTag = "s555"
		case instagram.PlatformS556:
			platformTag = "s556"
		case instagram.PlatformS559:
			platformTag = "s559"
		case instagram.PlatformS559V2:
			platformTag = "s559v2"
		case instagram.PlatformS560:
			platformTag = "s560"
		case instagram.PlatformS560V2:
			platformTag = "s560v2"
		case instagram.PlatformS561:
			platformTag = "s561"
		case instagram.PlatformS561V2:
			platformTag = "s561v2"
		case instagram.PlatformS561V3:
			platformTag = "s561v3"
		case instagram.PlatformS561V99:
			platformTag = "s561v99"
		case instagram.PlatformS562:
			platformTag = "s562"
		case instagram.PlatformS562V3:
			platformTag = "s562v3"
		case instagram.PlatformS563:
			platformTag = "s563"
		case instagram.PlatformS563V2:
			platformTag = "s563v2"
		case instagram.PlatformS563S21:
			platformTag = "s563s21"
		case instagram.PlatformS563V3S21:
			platformTag = "s563v3s21"
		case instagram.PlatformS563V4S21:
			platformTag = "s563v4s21"
		case instagram.PlatformS563V4S23:
			platformTag = "s563v4s23"
		case instagram.PlatformS563V5S21:
			platformTag = "s563v5s21"
		case instagram.PlatformS563V5S23:
			platformTag = "s563v5s23"
		case instagram.PlatformS563V6S21:
			platformTag = "s563v6s21"
		case instagram.PlatformS563V6S23:
			platformTag = "s563v6s23"
		case instagram.PlatformS564V1S21:
			platformTag = "s564v1s21"
		case instagram.PlatformS564V1S23:
			platformTag = "s564v1s23"
		case instagram.PlatformS564V2S21:
			platformTag = "s564v2s21"
		case instagram.PlatformS564V2S23:
			platformTag = "s564v2s23"
		case instagram.PlatformS564V3S21:
			platformTag = "s564v3s21"
		case instagram.PlatformS564V3S23:
			platformTag = "s564v3s23"
		case instagram.PlatformS561V4S21:
			platformTag = "s561v4s21"
		case instagram.PlatformS561V4S23:
			platformTag = "s561v4s23"
		case instagram.PlatformS562V4S21:
			platformTag = "s562v4s21"
		case instagram.PlatformS562V4S23:
			platformTag = "s562v4s23"
		case instagram.PlatformS399:
			platformTag = "s399"
		case instagram.PlatformIOSMessReg:
			platformTag = "iosmess"
		}
		if isRegPlatformSxxx(regPlatform) {
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
	webregister.SkipAuthLoginAtReg = interactionCfg.VerifyEnabled || verifyIsIOS(interactionCfg.ApiVerifyPlatform)

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
	allPlatformPools := map[string]**androidreg.PartitionedDatrPool{
		"Android":    &androidreg.SharedPool,
		"S23":        &s23reg.SharedPool,
		"S399":       &s399reg.SharedPool,
		"WebAndroid": &webandroidreg.SharedPool,
		"iOS HTTP":   &ioshttpreg.SharedPool,
		"Web":        &webregister.SharedPool,
		"iOS562":     &ios562reg.SharedDatrPool,
		"iOS563":     &ios563reg.SharedDatrPool,
		"iOS555":     &ios555reg.SharedDatrPool,
		"iOS564":     &ios564reg.SharedDatrPool,
		"iOS550":     &ios550reg.SharedDatrPool,
		"iOS540":     &ios540reg.SharedDatrPool,
		"iOS530":     &ios530reg.SharedDatrPool,
		"iOS520":     &ios520reg.SharedDatrPool,
		"iOS510":     &ios510reg.SharedDatrPool,
		"iOS500":     &ios500reg.SharedDatrPool,
		"iOS490":     &ios490reg.SharedDatrPool,
		"iOS480":     &ios480reg.SharedDatrPool,
		"iOS470":     &ios470reg.SharedDatrPool,
		"iOS460":     &ios460reg.SharedDatrPool,
		"iOS450":     &ios450reg.SharedDatrPool,
		"iOS440":     &ios440reg.SharedDatrPool,
		"iOS430":     &ios430reg.SharedDatrPool,
		"iOS420":     &ios420reg.SharedDatrPool,
		"iOS421":     &ios421reg.SharedDatrPool,
		"iOS422":     &ios422reg.SharedDatrPool,
		"iOS423":     &ios423reg.SharedDatrPool,
		"iOS424":     &ios424reg.SharedDatrPool,
		"iOS425":     &ios425reg.SharedDatrPool,
		"iOS426":     &ios426reg.SharedDatrPool,
		"iOS427":     &ios427reg.SharedDatrPool,
		"iOS428":     &ios428reg.SharedDatrPool,
		"iOS429":     &ios429reg.SharedDatrPool,
		"iOS431":     &ios431reg.SharedDatrPool,
		"iOS432":     &ios432reg.SharedDatrPool,
		"iOS433":     &ios433reg.SharedDatrPool,
		"iOS434":     &ios434reg.SharedDatrPool,
		"iOS435":     &ios435reg.SharedDatrPool,
		"iOS436":     &ios436reg.SharedDatrPool,
		"iOS437":     &ios437reg.SharedDatrPool,
		"iOS438":     &ios438reg.SharedDatrPool,
		"iOS439":     &ios439reg.SharedDatrPool,
		"iOS441":     &ios441reg.SharedDatrPool,
		"iOS442":     &ios442reg.SharedDatrPool,
		"iOS443":     &ios443reg.SharedDatrPool,
		"iOS444":     &ios444reg.SharedDatrPool,
		"iOS445":     &ios445reg.SharedDatrPool,
		"iOS446":     &ios446reg.SharedDatrPool,
		"iOS447":     &ios447reg.SharedDatrPool,
		"iOS448":     &ios448reg.SharedDatrPool,
		"iOS449":     &ios449reg.SharedDatrPool,
		"iOS451":     &ios451reg.SharedDatrPool,
		"iOS452":     &ios452reg.SharedDatrPool,
		"iOS453":     &ios453reg.SharedDatrPool,
		"iOS454":     &ios454reg.SharedDatrPool,
		"iOS455":     &ios455reg.SharedDatrPool,
		"iOS456":     &ios456reg.SharedDatrPool,
		"iOS457":     &ios457reg.SharedDatrPool,
		"iOS458":     &ios458reg.SharedDatrPool,
		"iOS459":     &ios459reg.SharedDatrPool,
		"iOS461":     &ios461reg.SharedDatrPool,
		"iOS462":     &ios462reg.SharedDatrPool,
		"iOS463":     &ios463reg.SharedDatrPool,
		"iOS464":     &ios464reg.SharedDatrPool,
		"iOS465":     &ios465reg.SharedDatrPool,
		"iOS466":     &ios466reg.SharedDatrPool,
		"iOS467":     &ios467reg.SharedDatrPool,
		"iOS468":     &ios468reg.SharedDatrPool,
		"iOS469":     &ios469reg.SharedDatrPool,
		"iOS471":     &ios471reg.SharedDatrPool,
		"iOS472":     &ios472reg.SharedDatrPool,
		"iOS473":     &ios473reg.SharedDatrPool,
		"iOS474":     &ios474reg.SharedDatrPool,
		"iOS475":     &ios475reg.SharedDatrPool,
		"iOS476":     &ios476reg.SharedDatrPool,
		"iOS477":     &ios477reg.SharedDatrPool,
		"iOS478":     &ios478reg.SharedDatrPool,
		"iOS479":     &ios479reg.SharedDatrPool,
		"iOS481":     &ios481reg.SharedDatrPool,
		"iOS482":     &ios482reg.SharedDatrPool,
		"iOS483":     &ios483reg.SharedDatrPool,
		"iOS484":     &ios484reg.SharedDatrPool,
		"iOS485":     &ios485reg.SharedDatrPool,
		"iOS486":     &ios486reg.SharedDatrPool,
		"iOS487":     &ios487reg.SharedDatrPool,
		"iOS488":     &ios488reg.SharedDatrPool,
		"iOS489":     &ios489reg.SharedDatrPool,
		"iOS491":     &ios491reg.SharedDatrPool,
		"iOS492":     &ios492reg.SharedDatrPool,
		"iOS493":     &ios493reg.SharedDatrPool,
		"iOS494":     &ios494reg.SharedDatrPool,
		"iOS495":     &ios495reg.SharedDatrPool,
		"iOS496":     &ios496reg.SharedDatrPool,
		"iOS497":     &ios497reg.SharedDatrPool,
		"iOS498":     &ios498reg.SharedDatrPool,
		"iOS499":     &ios499reg.SharedDatrPool,
		"iOS501":     &ios501reg.SharedDatrPool,
		"iOS502":     &ios502reg.SharedDatrPool,
		"iOS503":     &ios503reg.SharedDatrPool,
		"iOS504":     &ios504reg.SharedDatrPool,
		"iOS505":     &ios505reg.SharedDatrPool,
		"iOS506":     &ios506reg.SharedDatrPool,
		"iOS507":     &ios507reg.SharedDatrPool,
		"iOS508":     &ios508reg.SharedDatrPool,
		"iOS509":     &ios509reg.SharedDatrPool,
		"iOS511":     &ios511reg.SharedDatrPool,
		"iOS512":     &ios512reg.SharedDatrPool,
		"iOS513":     &ios513reg.SharedDatrPool,
		"iOS514":     &ios514reg.SharedDatrPool,
		"iOS515":     &ios515reg.SharedDatrPool,
		"iOS516":     &ios516reg.SharedDatrPool,
		"iOS517":     &ios517reg.SharedDatrPool,
		"iOS518":     &ios518reg.SharedDatrPool,
		"iOS519":     &ios519reg.SharedDatrPool,
		"iOS521":     &ios521reg.SharedDatrPool,
		"iOS522":     &ios522reg.SharedDatrPool,
		"iOS523":     &ios523reg.SharedDatrPool,
		"iOS524":     &ios524reg.SharedDatrPool,
		"iOS525":     &ios525reg.SharedDatrPool,
		"iOS526":     &ios526reg.SharedDatrPool,
		"iOS527":     &ios527reg.SharedDatrPool,
		"iOS528":     &ios528reg.SharedDatrPool,
		"iOS529":     &ios529reg.SharedDatrPool,
		"iOS531":     &ios531reg.SharedDatrPool,
		"iOS532":     &ios532reg.SharedDatrPool,
		"iOS533":     &ios533reg.SharedDatrPool,
		"iOS534":     &ios534reg.SharedDatrPool,
		"iOS535":     &ios535reg.SharedDatrPool,
		"iOS536":     &ios536reg.SharedDatrPool,
		"iOS537":     &ios537reg.SharedDatrPool,
		"iOS538":     &ios538reg.SharedDatrPool,
		"iOS539":     &ios539reg.SharedDatrPool,
		"iOS541":     &ios541reg.SharedDatrPool,
		"iOS542":     &ios542reg.SharedDatrPool,
		"iOS543":     &ios543reg.SharedDatrPool,
		"iOS544":     &ios544reg.SharedDatrPool,
		"iOS545":     &ios545reg.SharedDatrPool,
		"iOS546":     &ios546reg.SharedDatrPool,
		"iOS547":     &ios547reg.SharedDatrPool,
		"iOS548":     &ios548reg.SharedDatrPool,
		"iOS549":     &ios549reg.SharedDatrPool,
		"iOS551":     &ios551reg.SharedDatrPool,
		"iOS552":     &ios552reg.SharedDatrPool,
		"iOS553":     &ios553reg.SharedDatrPool,
		"iOS554":     &ios554reg.SharedDatrPool,
		"iOS556":     &ios556reg.SharedDatrPool,
		"iOS557":     &ios557reg.SharedDatrPool,
		"iOS558":     &ios558reg.SharedDatrPool,
		"iOS559":     &ios559reg.SharedDatrPool,
		"iOS561":     &ios561reg.SharedDatrPool,
		"iOS560":     &ios560reg.SharedDatrPool,
	}
	for name, poolPtr := range regSxxxPoolPointers() {
		allPlatformPools[name] = poolPtr
	}

	runPoolPath := cookie.NewRunPoolPath(defaultCookieDir())
	queuePaths := append([]string{runPoolPath}, cookieInitialFilePaths...)
	datrFileQueue := cookie.NewDatrFileQueue(queuePaths, 1500*time.Millisecond)
	defer datrFileQueue.Close()

	// cookieInitialPool là con trỏ đến sharedCookiePool, được gán sau khi pool được khởi tạo.
	// Dùng để persistNewDatr có thể cộng datr mới vào pool hiện tại (cập nhật pool count trên UI).
	var cookieInitialPool *androidreg.PartitionedDatrPool

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
	sharedCookiePool := androidreg.NewPartitionedPool(cookieInitialLimit)
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
		go func(p *androidreg.PartitionedDatrPool, every time.Duration) {
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

	if loadedCookieInitialCount > 0 &&
		regPlatform != instagram.PlatformAndroid &&
		regPlatform != instagram.PlatformS23 &&
		!isRegPlatformSxxx(regPlatform) &&
		regPlatform != instagram.PlatformWebAndroid &&
		regPlatform != instagram.PlatformIOS &&
		regPlatform != instagram.PlatformIOS562 &&
		regPlatform != instagram.PlatformIOS563 &&
		regPlatform != instagram.PlatformIOS555 &&
		regPlatform != instagram.PlatformIOS564 &&
		regPlatform != instagram.PlatformIOS550 &&
		regPlatform != instagram.PlatformIOS540 &&
		regPlatform != instagram.PlatformIOS530 &&
		regPlatform != instagram.PlatformIOS520 &&
		regPlatform != instagram.PlatformIOS510 &&
		regPlatform != instagram.PlatformIOS500 &&
		regPlatform != instagram.PlatformIOS490 &&
		regPlatform != instagram.PlatformIOS480 &&
		regPlatform != instagram.PlatformIOS470 &&
		regPlatform != instagram.PlatformIOS460 &&
		regPlatform != instagram.PlatformIOS450 &&
		regPlatform != instagram.PlatformIOS440 &&
		regPlatform != instagram.PlatformIOS430 &&
		regPlatform != instagram.PlatformIOS420 &&
		regPlatform != instagram.PlatformIOS421 &&
		regPlatform != instagram.PlatformIOS422 &&
		regPlatform != instagram.PlatformIOS423 &&
		regPlatform != instagram.PlatformIOS424 &&
		regPlatform != instagram.PlatformIOS425 &&
		regPlatform != instagram.PlatformIOS426 &&
		regPlatform != instagram.PlatformIOS427 &&
		regPlatform != instagram.PlatformIOS428 &&
		regPlatform != instagram.PlatformIOS429 &&
		regPlatform != instagram.PlatformIOS431 &&
		regPlatform != instagram.PlatformIOS432 &&
		regPlatform != instagram.PlatformIOS433 &&
		regPlatform != instagram.PlatformIOS434 &&
		regPlatform != instagram.PlatformIOS435 &&
		regPlatform != instagram.PlatformIOS436 &&
		regPlatform != instagram.PlatformIOS437 &&
		regPlatform != instagram.PlatformIOS438 &&
		regPlatform != instagram.PlatformIOS439 &&
		regPlatform != instagram.PlatformIOS441 &&
		regPlatform != instagram.PlatformIOS442 &&
		regPlatform != instagram.PlatformIOS443 &&
		regPlatform != instagram.PlatformIOS444 &&
		regPlatform != instagram.PlatformIOS445 &&
		regPlatform != instagram.PlatformIOS446 &&
		regPlatform != instagram.PlatformIOS447 &&
		regPlatform != instagram.PlatformIOS448 &&
		regPlatform != instagram.PlatformIOS449 &&
		regPlatform != instagram.PlatformIOS451 &&
		regPlatform != instagram.PlatformIOS452 &&
		regPlatform != instagram.PlatformIOS453 &&
		regPlatform != instagram.PlatformIOS454 &&
		regPlatform != instagram.PlatformIOS455 &&
		regPlatform != instagram.PlatformIOS456 &&
		regPlatform != instagram.PlatformIOS457 &&
		regPlatform != instagram.PlatformIOS458 &&
		regPlatform != instagram.PlatformIOS459 &&
		regPlatform != instagram.PlatformIOS461 &&
		regPlatform != instagram.PlatformIOS462 &&
		regPlatform != instagram.PlatformIOS463 &&
		regPlatform != instagram.PlatformIOS464 &&
		regPlatform != instagram.PlatformIOS465 &&
		regPlatform != instagram.PlatformIOS466 &&
		regPlatform != instagram.PlatformIOS467 &&
		regPlatform != instagram.PlatformIOS468 &&
		regPlatform != instagram.PlatformIOS469 &&
		regPlatform != instagram.PlatformIOS471 &&
		regPlatform != instagram.PlatformIOS472 &&
		regPlatform != instagram.PlatformIOS473 &&
		regPlatform != instagram.PlatformIOS474 &&
		regPlatform != instagram.PlatformIOS475 &&
		regPlatform != instagram.PlatformIOS476 &&
		regPlatform != instagram.PlatformIOS477 &&
		regPlatform != instagram.PlatformIOS478 &&
		regPlatform != instagram.PlatformIOS479 &&
		regPlatform != instagram.PlatformIOS481 &&
		regPlatform != instagram.PlatformIOS482 &&
		regPlatform != instagram.PlatformIOS483 &&
		regPlatform != instagram.PlatformIOS484 &&
		regPlatform != instagram.PlatformIOS485 &&
		regPlatform != instagram.PlatformIOS486 &&
		regPlatform != instagram.PlatformIOS487 &&
		regPlatform != instagram.PlatformIOS488 &&
		regPlatform != instagram.PlatformIOS489 &&
		regPlatform != instagram.PlatformIOS491 &&
		regPlatform != instagram.PlatformIOS492 &&
		regPlatform != instagram.PlatformIOS493 &&
		regPlatform != instagram.PlatformIOS494 &&
		regPlatform != instagram.PlatformIOS495 &&
		regPlatform != instagram.PlatformIOS496 &&
		regPlatform != instagram.PlatformIOS497 &&
		regPlatform != instagram.PlatformIOS498 &&
		regPlatform != instagram.PlatformIOS499 &&
		regPlatform != instagram.PlatformIOS501 &&
		regPlatform != instagram.PlatformIOS502 &&
		regPlatform != instagram.PlatformIOS503 &&
		regPlatform != instagram.PlatformIOS504 &&
		regPlatform != instagram.PlatformIOS505 &&
		regPlatform != instagram.PlatformIOS506 &&
		regPlatform != instagram.PlatformIOS507 &&
		regPlatform != instagram.PlatformIOS508 &&
		regPlatform != instagram.PlatformIOS509 &&
		regPlatform != instagram.PlatformIOS511 &&
		regPlatform != instagram.PlatformIOS512 &&
		regPlatform != instagram.PlatformIOS513 &&
		regPlatform != instagram.PlatformIOS514 &&
		regPlatform != instagram.PlatformIOS515 &&
		regPlatform != instagram.PlatformIOS516 &&
		regPlatform != instagram.PlatformIOS517 &&
		regPlatform != instagram.PlatformIOS518 &&
		regPlatform != instagram.PlatformIOS519 &&
		regPlatform != instagram.PlatformIOS521 &&
		regPlatform != instagram.PlatformIOS522 &&
		regPlatform != instagram.PlatformIOS523 &&
		regPlatform != instagram.PlatformIOS524 &&
		regPlatform != instagram.PlatformIOS525 &&
		regPlatform != instagram.PlatformIOS526 &&
		regPlatform != instagram.PlatformIOS527 &&
		regPlatform != instagram.PlatformIOS528 &&
		regPlatform != instagram.PlatformIOS529 &&
		regPlatform != instagram.PlatformIOS531 &&
		regPlatform != instagram.PlatformIOS532 &&
		regPlatform != instagram.PlatformIOS533 &&
		regPlatform != instagram.PlatformIOS534 &&
		regPlatform != instagram.PlatformIOS535 &&
		regPlatform != instagram.PlatformIOS536 &&
		regPlatform != instagram.PlatformIOS537 &&
		regPlatform != instagram.PlatformIOS538 &&
		regPlatform != instagram.PlatformIOS539 &&
		regPlatform != instagram.PlatformIOS541 &&
		regPlatform != instagram.PlatformIOS542 &&
		regPlatform != instagram.PlatformIOS543 &&
		regPlatform != instagram.PlatformIOS544 &&
		regPlatform != instagram.PlatformIOS545 &&
		regPlatform != instagram.PlatformIOS546 &&
		regPlatform != instagram.PlatformIOS547 &&
		regPlatform != instagram.PlatformIOS548 &&
		regPlatform != instagram.PlatformIOS549 &&
		regPlatform != instagram.PlatformIOS551 &&
		regPlatform != instagram.PlatformIOS552 &&
		regPlatform != instagram.PlatformIOS553 &&
		regPlatform != instagram.PlatformIOS554 &&
		regPlatform != instagram.PlatformIOS556 &&
		regPlatform != instagram.PlatformIOS557 &&
		regPlatform != instagram.PlatformIOS558 &&
		regPlatform != instagram.PlatformIOS559 &&
		regPlatform != instagram.PlatformIOS561 &&
		regPlatform != instagram.PlatformIOS560 &&
		regPlatform != instagram.PlatformWeb {
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": fmt.Sprintf("[CookieInitial] Loaded %d cookie initial (limit %d/mỗi)", loadedCookieInitialCount, cookieInitialLimit),
		})
	}

	// ── iOS562 device profile pool ────────────────────────────────────────────
	// Load DeviceID/FamilyDeviceID/MachineID từ file reg thành công trước đó.
	// Sau mỗi reg thành công, register.go tự Add() vào pool và trigger persistHook.
	ios562DeviceFile := defaultCookieDir() + "/ios562_devices.txt"
	ios562Pool := ios562reg.NewDevicePool(5)
	if n, err := ios562Pool.LoadFromFile(ios562DeviceFile); err == nil && n > 0 {
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": fmt.Sprintf("[iOS562Pool] %d device profiles sẵn sàng", n),
		})
	}
	ios562Pool.SetPersistHook(func(dp ios562reg.DeviceProfile) {
		line := dp.DeviceID + "|" + dp.FamilyDeviceID + "|" + dp.MachineID + "\n"
		if f, err := os.OpenFile(ios562DeviceFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			_, _ = f.WriteString(line)
			f.Close()
		}
	})
	ios562reg.SharedDevicePool = ios562Pool
	defer func() { ios562reg.SharedDevicePool = nil }()

	// ── iOS555 device profile pool (parity với iOS562) ────────────────────────
	ios555DeviceFile := defaultCookieDir() + "/ios555_devices.txt"
	ios555Pool := ios555reg.NewDevicePool(5)
	if n, err := ios555Pool.LoadFromFile(ios555DeviceFile); err == nil && n > 0 {
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": fmt.Sprintf("[iOS555Pool] %d device profiles sẵn sàng", n),
		})
	}
	ios555Pool.SetPersistHook(func(dp ios555reg.DeviceProfile) {
		line := dp.DeviceID + "|" + dp.FamilyDeviceID + "|" + dp.MachineID + "\n"
		if f, err := os.OpenFile(ios555DeviceFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			_, _ = f.WriteString(line)
			f.Close()
		}
	})
	ios555reg.SharedDevicePool = ios555Pool
	defer func() { ios555reg.SharedDevicePool = nil }()

	// ── iOS564 device profile pool (parity với iOS562) ────────────────────────
	ios564DeviceFile := defaultCookieDir() + "/ios564_devices.txt"
	ios564Pool := ios564reg.NewDevicePool(5)
	if n, err := ios564Pool.LoadFromFile(ios564DeviceFile); err == nil && n > 0 {
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": fmt.Sprintf("[iOS564Pool] %d device profiles sẵn sàng", n),
		})
	}
	ios564Pool.SetPersistHook(func(dp ios564reg.DeviceProfile) {
		line := dp.DeviceID + "|" + dp.FamilyDeviceID + "|" + dp.MachineID + "\n"
		if f, err := os.OpenFile(ios564DeviceFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			_, _ = f.WriteString(line)
			f.Close()
		}
	})
	ios564reg.SharedDevicePool = ios564Pool
	defer func() { ios564reg.SharedDevicePool = nil }()

	// ── iOS550 device profile pool (parity với iOS562) ────────────────────────
	ios550DeviceFile := defaultCookieDir() + "/ios550_devices.txt"
	ios550Pool := ios550reg.NewDevicePool(5)
	if n, err := ios550Pool.LoadFromFile(ios550DeviceFile); err == nil && n > 0 {
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": fmt.Sprintf("[iOS550Pool] %d device profiles sẵn sàng", n),
		})
	}
	ios550Pool.SetPersistHook(func(dp ios550reg.DeviceProfile) {
		line := dp.DeviceID + "|" + dp.FamilyDeviceID + "|" + dp.MachineID + "\n"
		if f, err := os.OpenFile(ios550DeviceFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			_, _ = f.WriteString(line)
			f.Close()
		}
	})
	ios550reg.SharedDevicePool = ios550Pool
	defer func() { ios550reg.SharedDevicePool = nil }()

	// ── iOS540 device profile pool (parity với iOS562) ────────────────────────
	ios540DeviceFile := defaultCookieDir() + "/ios540_devices.txt"
	ios540Pool := ios540reg.NewDevicePool(5)
	if n, err := ios540Pool.LoadFromFile(ios540DeviceFile); err == nil && n > 0 {
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": fmt.Sprintf("[iOS540Pool] %d device profiles sẵn sàng", n),
		})
	}
	ios540Pool.SetPersistHook(func(dp ios540reg.DeviceProfile) {
		line := dp.DeviceID + "|" + dp.FamilyDeviceID + "|" + dp.MachineID + "\n"
		if f, err := os.OpenFile(ios540DeviceFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			_, _ = f.WriteString(line)
			f.Close()
		}
	})
	ios540reg.SharedDevicePool = ios540Pool
	defer func() { ios540reg.SharedDevicePool = nil }()

	// ── iOS530 device profile pool (parity với iOS562) ────────────────────────
	ios530DeviceFile := defaultCookieDir() + "/ios530_devices.txt"
	ios530Pool := ios530reg.NewDevicePool(5)
	if n, err := ios530Pool.LoadFromFile(ios530DeviceFile); err == nil && n > 0 {
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": fmt.Sprintf("[iOS530Pool] %d device profiles sẵn sàng", n),
		})
	}
	ios530Pool.SetPersistHook(func(dp ios530reg.DeviceProfile) {
		line := dp.DeviceID + "|" + dp.FamilyDeviceID + "|" + dp.MachineID + "\n"
		if f, err := os.OpenFile(ios530DeviceFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			_, _ = f.WriteString(line)
			f.Close()
		}
	})
	ios530reg.SharedDevicePool = ios530Pool
	defer func() { ios530reg.SharedDevicePool = nil }()

	// ── iOS520 device profile pool (parity với iOS562) ────────────────────────
	ios520DeviceFile := defaultCookieDir() + "/ios520_devices.txt"
	ios520Pool := ios520reg.NewDevicePool(5)
	if n, err := ios520Pool.LoadFromFile(ios520DeviceFile); err == nil && n > 0 {
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": fmt.Sprintf("[iOS520Pool] %d device profiles sẵn sàng", n),
		})
	}
	ios520Pool.SetPersistHook(func(dp ios520reg.DeviceProfile) {
		line := dp.DeviceID + "|" + dp.FamilyDeviceID + "|" + dp.MachineID + "\n"
		if f, err := os.OpenFile(ios520DeviceFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
			_, _ = f.WriteString(line)
			f.Close()
		}
	})
	ios520reg.SharedDevicePool = ios520Pool
	defer func() { ios520reg.SharedDevicePool = nil }()

	// User yêu cầu: nếu CookieInitialMethod="file" mà không có datr nào → DỪNG HOÀN TOÀN.
	// Tránh reg fake không có datr → kết quả tệ. "Tạo mới"/"Ck" thì không cần check.
	if strings.EqualFold(strings.TrimSpace(interactionCfg.CookieInitialMethod), "file") &&
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
	if runRes.iosPool != nil {
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": "[SessionPool] iOS HTTP — keep session enabled",
		})
	}
	if runRes.andrPool != nil {
		runtime.EventsEmit(a.ctx, "register:status", map[string]interface{}{
			"index": 0, "phone": "system", "proxy": "",
			"msg": "[SessionPool] WebAndroid — keep session enabled",
		})
	}

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
		acc          runner.AccountInput // account đã reg (UID, Token, Cookie, Password, Proxy, UA, DeviceID, Email...)
		prof         instagram.RegInput   // profile gốc reg (lấy thêm field nếu cần)
		displayProxy string              // IP CHẠY hiển thị (đã CheckIP)
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
			profile := webregister.RandomRegInput("", "", proxyStr)
			switch regPlatform {
			case instagram.PlatformWebAndroid:
				profile.UserAgent = fakeinfo.RandomChromeAndroidProfile().UserAgent
			case instagram.PlatformIOS:
				profile.UserAgent = fakeinfo.RandomIPhoneProfile().UserAgent
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

				// Đăng ký slot vào PartitionedDatrPool — nhận partition riêng
				if regPlatform == instagram.PlatformAndroid && androidreg.SharedPool != nil {
					androidreg.SharedPool.Register(prof.SlotIdx)
					defer androidreg.SharedPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformS23 && s23reg.SharedPool != nil {
					s23reg.SharedPool.Register(prof.SlotIdx)
					defer s23reg.SharedPool.Unregister(prof.SlotIdx)
				}
				if pool := regPoolForSxxx(regPlatform); pool != nil {
					pool.Register(prof.SlotIdx)
					defer pool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformS399 && s399reg.SharedPool != nil {
					s399reg.SharedPool.Register(prof.SlotIdx)
					defer s399reg.SharedPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS562 && ios562reg.SharedDatrPool != nil {
					ios562reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios562reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS563 && ios563reg.SharedDatrPool != nil {
					ios563reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios563reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS555 && ios555reg.SharedDatrPool != nil {
					ios555reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios555reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS564 && ios564reg.SharedDatrPool != nil {
					ios564reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios564reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS550 && ios550reg.SharedDatrPool != nil {
					ios550reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios550reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS540 && ios540reg.SharedDatrPool != nil {
					ios540reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios540reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS530 && ios530reg.SharedDatrPool != nil {
					ios530reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios530reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS520 && ios520reg.SharedDatrPool != nil {
					ios520reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios520reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS510 && ios510reg.SharedDatrPool != nil {
					ios510reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios510reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS500 && ios500reg.SharedDatrPool != nil {
					ios500reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios500reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS490 && ios490reg.SharedDatrPool != nil {
					ios490reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios490reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS480 && ios480reg.SharedDatrPool != nil {
					ios480reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios480reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS470 && ios470reg.SharedDatrPool != nil {
					ios470reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios470reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS460 && ios460reg.SharedDatrPool != nil {
					ios460reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios460reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS450 && ios450reg.SharedDatrPool != nil {
					ios450reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios450reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS440 && ios440reg.SharedDatrPool != nil {
					ios440reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios440reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS430 && ios430reg.SharedDatrPool != nil {
					ios430reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios430reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS420 && ios420reg.SharedDatrPool != nil {
					ios420reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios420reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS421 && ios421reg.SharedDatrPool != nil {
					ios421reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios421reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS422 && ios422reg.SharedDatrPool != nil {
					ios422reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios422reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS423 && ios423reg.SharedDatrPool != nil {
					ios423reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios423reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS424 && ios424reg.SharedDatrPool != nil {
					ios424reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios424reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS425 && ios425reg.SharedDatrPool != nil {
					ios425reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios425reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS426 && ios426reg.SharedDatrPool != nil {
					ios426reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios426reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS427 && ios427reg.SharedDatrPool != nil {
					ios427reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios427reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS428 && ios428reg.SharedDatrPool != nil {
					ios428reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios428reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS429 && ios429reg.SharedDatrPool != nil {
					ios429reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios429reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS431 && ios431reg.SharedDatrPool != nil {
					ios431reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios431reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS432 && ios432reg.SharedDatrPool != nil {
					ios432reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios432reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS433 && ios433reg.SharedDatrPool != nil {
					ios433reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios433reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS434 && ios434reg.SharedDatrPool != nil {
					ios434reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios434reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS435 && ios435reg.SharedDatrPool != nil {
					ios435reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios435reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS436 && ios436reg.SharedDatrPool != nil {
					ios436reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios436reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS437 && ios437reg.SharedDatrPool != nil {
					ios437reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios437reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS438 && ios438reg.SharedDatrPool != nil {
					ios438reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios438reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS439 && ios439reg.SharedDatrPool != nil {
					ios439reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios439reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS441 && ios441reg.SharedDatrPool != nil {
					ios441reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios441reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS442 && ios442reg.SharedDatrPool != nil {
					ios442reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios442reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS443 && ios443reg.SharedDatrPool != nil {
					ios443reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios443reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS444 && ios444reg.SharedDatrPool != nil {
					ios444reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios444reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS445 && ios445reg.SharedDatrPool != nil {
					ios445reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios445reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS446 && ios446reg.SharedDatrPool != nil {
					ios446reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios446reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS447 && ios447reg.SharedDatrPool != nil {
					ios447reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios447reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS448 && ios448reg.SharedDatrPool != nil {
					ios448reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios448reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS449 && ios449reg.SharedDatrPool != nil {
					ios449reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios449reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS451 && ios451reg.SharedDatrPool != nil {
					ios451reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios451reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS452 && ios452reg.SharedDatrPool != nil {
					ios452reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios452reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS453 && ios453reg.SharedDatrPool != nil {
					ios453reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios453reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS454 && ios454reg.SharedDatrPool != nil {
					ios454reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios454reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS455 && ios455reg.SharedDatrPool != nil {
					ios455reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios455reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS456 && ios456reg.SharedDatrPool != nil {
					ios456reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios456reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS457 && ios457reg.SharedDatrPool != nil {
					ios457reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios457reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS458 && ios458reg.SharedDatrPool != nil {
					ios458reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios458reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS459 && ios459reg.SharedDatrPool != nil {
					ios459reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios459reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS461 && ios461reg.SharedDatrPool != nil {
					ios461reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios461reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS462 && ios462reg.SharedDatrPool != nil {
					ios462reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios462reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS463 && ios463reg.SharedDatrPool != nil {
					ios463reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios463reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS464 && ios464reg.SharedDatrPool != nil {
					ios464reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios464reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS465 && ios465reg.SharedDatrPool != nil {
					ios465reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios465reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS466 && ios466reg.SharedDatrPool != nil {
					ios466reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios466reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS467 && ios467reg.SharedDatrPool != nil {
					ios467reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios467reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS468 && ios468reg.SharedDatrPool != nil {
					ios468reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios468reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS469 && ios469reg.SharedDatrPool != nil {
					ios469reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios469reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS471 && ios471reg.SharedDatrPool != nil {
					ios471reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios471reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS472 && ios472reg.SharedDatrPool != nil {
					ios472reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios472reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS473 && ios473reg.SharedDatrPool != nil {
					ios473reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios473reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS474 && ios474reg.SharedDatrPool != nil {
					ios474reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios474reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS475 && ios475reg.SharedDatrPool != nil {
					ios475reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios475reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS476 && ios476reg.SharedDatrPool != nil {
					ios476reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios476reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS477 && ios477reg.SharedDatrPool != nil {
					ios477reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios477reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS478 && ios478reg.SharedDatrPool != nil {
					ios478reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios478reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS479 && ios479reg.SharedDatrPool != nil {
					ios479reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios479reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS481 && ios481reg.SharedDatrPool != nil {
					ios481reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios481reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS482 && ios482reg.SharedDatrPool != nil {
					ios482reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios482reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS483 && ios483reg.SharedDatrPool != nil {
					ios483reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios483reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS484 && ios484reg.SharedDatrPool != nil {
					ios484reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios484reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS485 && ios485reg.SharedDatrPool != nil {
					ios485reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios485reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS486 && ios486reg.SharedDatrPool != nil {
					ios486reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios486reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS487 && ios487reg.SharedDatrPool != nil {
					ios487reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios487reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS488 && ios488reg.SharedDatrPool != nil {
					ios488reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios488reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS489 && ios489reg.SharedDatrPool != nil {
					ios489reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios489reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS491 && ios491reg.SharedDatrPool != nil {
					ios491reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios491reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS492 && ios492reg.SharedDatrPool != nil {
					ios492reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios492reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS493 && ios493reg.SharedDatrPool != nil {
					ios493reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios493reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS494 && ios494reg.SharedDatrPool != nil {
					ios494reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios494reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS495 && ios495reg.SharedDatrPool != nil {
					ios495reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios495reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS496 && ios496reg.SharedDatrPool != nil {
					ios496reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios496reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS497 && ios497reg.SharedDatrPool != nil {
					ios497reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios497reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS498 && ios498reg.SharedDatrPool != nil {
					ios498reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios498reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS499 && ios499reg.SharedDatrPool != nil {
					ios499reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios499reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS501 && ios501reg.SharedDatrPool != nil {
					ios501reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios501reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS502 && ios502reg.SharedDatrPool != nil {
					ios502reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios502reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS503 && ios503reg.SharedDatrPool != nil {
					ios503reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios503reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS504 && ios504reg.SharedDatrPool != nil {
					ios504reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios504reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS505 && ios505reg.SharedDatrPool != nil {
					ios505reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios505reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS506 && ios506reg.SharedDatrPool != nil {
					ios506reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios506reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS507 && ios507reg.SharedDatrPool != nil {
					ios507reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios507reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS508 && ios508reg.SharedDatrPool != nil {
					ios508reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios508reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS509 && ios509reg.SharedDatrPool != nil {
					ios509reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios509reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS511 && ios511reg.SharedDatrPool != nil {
					ios511reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios511reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS512 && ios512reg.SharedDatrPool != nil {
					ios512reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios512reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS513 && ios513reg.SharedDatrPool != nil {
					ios513reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios513reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS514 && ios514reg.SharedDatrPool != nil {
					ios514reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios514reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS515 && ios515reg.SharedDatrPool != nil {
					ios515reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios515reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS516 && ios516reg.SharedDatrPool != nil {
					ios516reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios516reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS517 && ios517reg.SharedDatrPool != nil {
					ios517reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios517reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS518 && ios518reg.SharedDatrPool != nil {
					ios518reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios518reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS519 && ios519reg.SharedDatrPool != nil {
					ios519reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios519reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS521 && ios521reg.SharedDatrPool != nil {
					ios521reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios521reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS522 && ios522reg.SharedDatrPool != nil {
					ios522reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios522reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS523 && ios523reg.SharedDatrPool != nil {
					ios523reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios523reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS524 && ios524reg.SharedDatrPool != nil {
					ios524reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios524reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS525 && ios525reg.SharedDatrPool != nil {
					ios525reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios525reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS526 && ios526reg.SharedDatrPool != nil {
					ios526reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios526reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS527 && ios527reg.SharedDatrPool != nil {
					ios527reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios527reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS528 && ios528reg.SharedDatrPool != nil {
					ios528reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios528reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS529 && ios529reg.SharedDatrPool != nil {
					ios529reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios529reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS531 && ios531reg.SharedDatrPool != nil {
					ios531reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios531reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS532 && ios532reg.SharedDatrPool != nil {
					ios532reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios532reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS533 && ios533reg.SharedDatrPool != nil {
					ios533reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios533reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS534 && ios534reg.SharedDatrPool != nil {
					ios534reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios534reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS535 && ios535reg.SharedDatrPool != nil {
					ios535reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios535reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS536 && ios536reg.SharedDatrPool != nil {
					ios536reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios536reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS537 && ios537reg.SharedDatrPool != nil {
					ios537reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios537reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS538 && ios538reg.SharedDatrPool != nil {
					ios538reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios538reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS539 && ios539reg.SharedDatrPool != nil {
					ios539reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios539reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS541 && ios541reg.SharedDatrPool != nil {
					ios541reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios541reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS542 && ios542reg.SharedDatrPool != nil {
					ios542reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios542reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS543 && ios543reg.SharedDatrPool != nil {
					ios543reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios543reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS544 && ios544reg.SharedDatrPool != nil {
					ios544reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios544reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS545 && ios545reg.SharedDatrPool != nil {
					ios545reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios545reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS546 && ios546reg.SharedDatrPool != nil {
					ios546reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios546reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS547 && ios547reg.SharedDatrPool != nil {
					ios547reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios547reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS548 && ios548reg.SharedDatrPool != nil {
					ios548reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios548reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS549 && ios549reg.SharedDatrPool != nil {
					ios549reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios549reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS551 && ios551reg.SharedDatrPool != nil {
					ios551reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios551reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS552 && ios552reg.SharedDatrPool != nil {
					ios552reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios552reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS553 && ios553reg.SharedDatrPool != nil {
					ios553reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios553reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS554 && ios554reg.SharedDatrPool != nil {
					ios554reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios554reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS556 && ios556reg.SharedDatrPool != nil {
					ios556reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios556reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS557 && ios557reg.SharedDatrPool != nil {
					ios557reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios557reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS558 && ios558reg.SharedDatrPool != nil {
					ios558reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios558reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS559 && ios559reg.SharedDatrPool != nil {
					ios559reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios559reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS561 && ios561reg.SharedDatrPool != nil {
					ios561reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios561reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS560 && ios560reg.SharedDatrPool != nil {
					ios560reg.SharedDatrPool.Register(prof.SlotIdx)
					defer ios560reg.SharedDatrPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformWebAndroid && webandroidreg.SharedPool != nil {
					webandroidreg.SharedPool.Register(prof.SlotIdx)
					defer webandroidreg.SharedPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformIOS && ioshttpreg.SharedPool != nil {
					ioshttpreg.SharedPool.Register(prof.SlotIdx)
					defer ioshttpreg.SharedPool.Unregister(prof.SlotIdx)
				}
				if regPlatform == instagram.PlatformWeb && webregister.SharedPool != nil {
					webregister.SharedPool.Register(prof.SlotIdx)
					defer webregister.SharedPool.Unregister(prof.SlotIdx)
				}

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

				// Render session proxy TRƯỚC mọi thứ (C#: RenderSessionIfIsProxyServer)
				// Mỗi goroutine tạo session ID riêng → IP mới mỗi lần reg
				prof.Proxy = proxy.RenderSessionIfIsProxyServer(prof.Proxy)

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

				// Kiểm tra IP thực (timeout 6s), dùng làm display IP + country.
				// Nếu CheckIP fail → displayProxy = "" (cột IP CHẠY để trống, KHÔNG show raw proxy).
				generalCfg := a.LoadSettings().General
				displayProxy := ""
				countryCode := ""
				{
					// Parent = ctx (run-scoped) → Stop register cancel CheckIP đang chờ;
					// trước đây dùng a.ctx → CheckIP treo đến hết 6s ngay cả khi user Stop.
					ipCtx, ipCancel := context.WithTimeout(ctx, 6*time.Second)
					if realIP, err := proxy.CheckIP(ipCtx, prof.Proxy, generalCfg.ApiCheckIp); err == nil && realIP != "" {
						displayProxy = realIP
						// Extract country code từ "89.200.217.100/cl" → "cl"
						if idx := strings.LastIndex(realIP, "/"); idx >= 0 {
							countryCode = strings.ToLower(realIP[idx+1:])
						}
					}
					ipCancel()
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
					if phone == "" {
						phone = webregister.GeneratePhoneByCountry(countryCode)
					}
					if phone != "" { // chỉ override khi có kết quả — giữ VN default nếu cả 2 trả ""
						prof.Phone = phone
					} else {
						// Cả 2 nguồn đều thiếu pattern cho country này → log để user bổ sung.
						logMissingPhoneCountryCode(countryCode)
					}
				}
				// FALLBACK 2026-05-15: nếu phone vẫn rỗng sau mọi attempt (do phone clear
				// block + country không có phone generator + phone_database trống), force
				// generate VN phone để tránh "Thiếu contactpoint" fail toàn slot.
				// Không ảnh hưởng nếu Mail/TempMail mode (đã có Email làm contactpoint).
				if prof.Phone == "" && prof.Email == "" && !isMailMode && !isTempMailMode {
					if vnPhone := webregister.GeneratePhoneByCountry("VN"); vnPhone != "" {
						prof.Phone = vnPhone
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
					// Ghi vào batch cache — batch goroutine flush lên frontend mỗi 500ms
					regBatchCache.Store(threadIdx, regEntry{
						index: threadIdx, phone: pickContact(), proxy: displayProxy, userAgent: prof.UserAgent, msg: msg,
					})
				}

				threadReg, threadRegErr := instagram.NewRegisterer(regPlatform)

				// Keep-session context: pin device/UA/session cho cả lifetime goroutine.
				// Port C# pattern: 1 `FacebookAccountModel` + 1 `IHttpRequestClient` shared
				// cho tất cả regs trong thread → giảm device rotation, FB trust hơn.
				var s23WCtx *s23reg.WorkerContext
				var sxxxWCtx regSxxxWorkerContext
				var s399WCtx *s399reg.WorkerContext
				var androidWCtx *androidreg.WorkerContext
				var webandroidWCtx *webandroidreg.WorkerContext
				// C# LocaleFake: "random" → override locale bằng RandomLocale() (từ locales.txt).
				// "match-ip" → giữ locale đã build từ country proxy.
				overrideLocale := ""
				if strings.EqualFold(strings.TrimSpace(generalCfg.LocaleFake), "random") {
					overrideLocale = fakeinfo.RandomLocale()
				}
				// C# SimNetworkType: WIFI/mobile.LTE/cell.CTRadioAccessTechnologyHSDPA/unknown
				// Port map GUI value → Xfb_connection_type (case-insensitive).
				overrideConnType := mapSimNetworkType(generalCfg.SimNetworkType)

				if isRegPlatformSxxx(regPlatform) {
					if wctx, err := newRegSxxxWorkerContext(regPlatform, prof.Proxy, effectiveCountryCode); err == nil {
						sxxxWCtx = wctx
						defer sxxxWCtx.Close()
						sxxxWCtx.SetLocale(overrideLocale)
						sxxxWCtx.SetConnectionType(overrideConnType)
						sxxxWCtx.SetUAOptions(interactionCfg.AddVirtualSpecAndroid)
						prof.UserAgent = sxxxWCtx.UserAgent()
					}
				}

				switch regPlatform {
				case instagram.PlatformS23:
					if wctx, err := s23reg.NewWorkerContext(prof.Proxy, effectiveCountryCode); err == nil {
						s23WCtx = wctx
						defer s23WCtx.Close()
						s23WCtx.SetLocale(overrideLocale)
						s23WCtx.SetConnectionType(overrideConnType)
						s23WCtx.SetUAOptions(interactionCfg.AddVirtualSpecAndroid)
						prof.UserAgent = s23WCtx.Profile().S23UA
					}
				case instagram.PlatformAndroid:
					if wctx, err := androidreg.NewWorkerContext(prof.Proxy, effectiveCountryCode); err == nil {
						androidWCtx = wctx
						defer androidWCtx.Close()
						androidWCtx.SetLocale(overrideLocale)
						androidWCtx.SetConnectionType(overrideConnType)
						androidWCtx.SetUAOptions(interactionCfg.AddVirtualSpecAndroid)
						prof.UserAgent = androidWCtx.Profile().UserAgent
					}
				case instagram.PlatformWebAndroid:
					if wctx, err := webandroidreg.NewWorkerContext(prof.Proxy); err == nil {
						webandroidWCtx = wctx
						defer webandroidWCtx.Close()
						prof.UserAgent = webandroidWCtx.Profile().UserAgent
					}
				case instagram.PlatformS399:
					if wctx, err := s399reg.NewWorkerContext(prof.Proxy, effectiveCountryCode); err == nil {
						s399WCtx = wctx
						defer s399WCtx.Close()
						s399WCtx.SetLocale(overrideLocale)
						s399WCtx.SetConnectionType(overrideConnType)
						s399WCtx.SetUAOptions(interactionCfg.AddVirtualSpecAndroid)
						prof.UserAgent = s399WCtx.Profile().S399UA
					}
				}

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
						if s23WCtx != nil {
							s23WCtx.SetUA(origUA)
						}
						if sxxxWCtx != nil {
							sxxxWCtx.SetUA(origUA)
						}
						if s399WCtx != nil {
							s399WCtx.SetUA(origUA)
						}
						if androidWCtx != nil {
							androidWCtx.SetUA(origUA)
						}
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
						if s23WCtx != nil {
							s23WCtx.SetUA(finalUA)
						}
						if sxxxWCtx != nil {
							sxxxWCtx.SetUA(finalUA)
						}
						if s399WCtx != nil {
							s399WCtx.SetUA(finalUA)
						}
						if androidWCtx != nil {
							androidWCtx.SetUA(finalUA)
						}
						if webandroidWCtx != nil {
							webandroidWCtx.SetUA(finalUA)
						}
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
							if s23WCtx != nil {
								s23WCtx.SetUA(keptUA)
							}
							if sxxxWCtx != nil {
								sxxxWCtx.SetUA(keptUA)
							}
							if s399WCtx != nil {
								s399WCtx.SetUA(keptUA)
							}
							if androidWCtx != nil {
								androidWCtx.SetUA(keptUA)
							}
							if webandroidWCtx != nil {
								webandroidWCtx.SetUA(keptUA)
							}
						} else {
							regUABySlot.Delete(slotIdx)
						}
					}
				}

				// TrackingID: thêm XID/<random16>; vào UA sau khi mọi override đã xong.
				if interactionCfg.TrackingIDReg && prof.UserAgent != "" {
					prof.UserAgent = appendXIDToUA(prof.UserAgent)
					if s23WCtx != nil {
						s23WCtx.SetUA(prof.UserAgent)
					}
					if sxxxWCtx != nil {
						sxxxWCtx.SetUA(prof.UserAgent)
					}
					if s399WCtx != nil {
						s399WCtx.SetUA(prof.UserAgent)
					}
					if androidWCtx != nil {
						androidWCtx.SetUA(prof.UserAgent)
					}
					if webandroidWCtx != nil {
						webandroidWCtx.SetUA(prof.UserAgent)
					}
				}

				// C# MFB: RegisterWithKeepHttpSession runs in a LOOP per thread:
				//   Success + CookieInitial → keep session → delay → reg again (same proxy/IP)
				//   Fail/Blocked → break → thread ends → new session
				// This "warm session" pattern gives higher success rate for subsequent regs.
				maxKeepSessionRegs := 1 // default: 1 reg per goroutine (no loop)
				if regPlatform == instagram.PlatformIOS && loadedCookieInitialCount > 0 {
					maxKeepSessionRegs = 10 // iOS HTTP + CookieInitial: up to 10 regs per session
				}
				if regPlatform == instagram.PlatformS23 && s23reg.SharedPool != nil && s23reg.SharedPool.Size() > 0 {
					maxKeepSessionRegs = 5 // S23 + CookiePool: warm session → higher success rate
				}
				if pool := regPoolForSxxx(regPlatform); pool != nil && pool.Size() > 0 {
					maxKeepSessionRegs = 5 // Sxxx + CookiePool: warm session
				}
				if regPlatform == instagram.PlatformS399 && s399reg.SharedPool != nil && s399reg.SharedPool.Size() > 0 {
					maxKeepSessionRegs = 5 // S399 + CookiePool: warm session → higher success rate
				}
				if regPlatform == instagram.PlatformAndroid && androidreg.SharedPool != nil && androidreg.SharedPool.Size() > 0 {
					maxKeepSessionRegs = 5 // Android V3 + CookiePool: reuse session + device pin
				}
				if regPlatform == instagram.PlatformWebAndroid && loadedCookieInitialCount > 0 {
					maxKeepSessionRegs = 5 // WebAndroid + CookiePool: reuse ChromeAndroid device pin
				}

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
							if phone == "" {
								phone = webregister.GeneratePhoneByCountry(countryCode)
							}
							if phone != "" {
								prof.Phone = phone
							} else if countryCode != "" {
								logMissingPhoneCountryCode(countryCode)
							}
							// FALLBACK 2026-05-15: cùng logic ở first attempt — không để prof.Phone rỗng.
							if prof.Phone == "" && prof.Email == "" {
								if vnPhone := webregister.GeneratePhoneByCountry("VN"); vnPhone != "" {
									prof.Phone = vnPhone
								}
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
							return mailSvc.WaitForCode(c, 30, 4000)
						}
						onStatus("[IG] GetOTP wired — reg sẽ tự đọc OTP + confirm")
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
					// IG rebrand: toàn bộ platform register dùng IG adapter (threadReg).
					// WorkerContext (s23/android/webandroid/s399/sxxx) chỉ còn dùng để set
					// UA/proxy/session, KHÔNG chạy Register Facebook cũ nữa.
					_ = s23WCtx
					_ = sxxxWCtx
					_ = s399WCtx
					_ = androidWCtx
					_ = webandroidWCtx
					if threadReg == nil {
						result = &instagram.RegResult{Success: false, Message: fmt.Sprintf("platform %q không có registerer: %v", regPlatform, threadRegErr)}
					} else {
						result = threadReg.Register(regWorkerCtx, &prof, onStatus)
					}

					// Platforms tự build UA nội bộ (ios562) trả UA qua result.UserAgent.
					// Cập nhật prof để các bước sau (save, verify, emit) thấy đúng UA.
					if result != nil && result.UserAgent != "" && prof.UserAgent == "" {
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
						if regPassword != "" && !webregister.SkipAuthLoginAtReg {
							onStatus("[AutoVerify] Thiếu EAAAAU token → login Android để lấy...")
							// EAA fetch BẮT BUỘC dùng Android UA (FB4A) — không dùng prof.UserAgent
							// (có thể là Web/iOS/Chrome khi reg platform khác Android). Dùng UA verify
							// platform tương ứng (S555/S556/.../Android) → match fingerprint server-side.
							androidUAForToken := prof.UserAgent
							if !platformNeedsAndroidLoginToken(string(regPlatform)) {
								verifyUAConfig := applyVerifyPlatformUAConfig(a.LoadInteractionConfig())
								androidUAForToken = pickUAForVerifyPlatform(
									verifyPlatformFromType(verifyUAConfig.ApiVerifyPlatform),
									prof.UserAgent, verifyUAConfig, phoneToCountryCode(prof.Phone))
							}
							if androidUAForToken == "" {
								androidUAForToken = prof.UserAgent
							}
							tokenCtx, tokenCancel := context.WithTimeout(regWorkerCtx, 30*time.Second)
							// Extract datr từ cookie để pass machineID vào /auth/login.
							machineIDForLogin := extractDatrFromCookieLine(result.Cookie)
							// PORT S399 step 2: REST `/auth/login` thay vì Bloks/RSA (đều bị FB rotate schema).
							tok := webregister.FetchAndroidTokenLegacy(tokenCtx, result.UID, regPassword, machineIDForLogin, "en_US", "", prof.Proxy, androidUAForToken, func(m string) { onStatus(m) })
							tokenCancel()
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
							"index":  threadIdx,
							"uid":    result.UID,
							"token":  result.AccessToken,
							"cookie": result.Cookie,
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
						go saveRegOutcome(regWriter, regCounters, status, result.Message, accForSave, string(regPlatform), login, onRegSuccess)
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
					ID: slotID, UID: acc.UID, FullName: acc.FullName,
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
