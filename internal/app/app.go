package app

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"HVRIns/internal/clonehv"
	"HVRIns/internal/cookie"
	"HVRIns/internal/email"
	emailrent "HVRIns/internal/email/rent"
	emailtemp "HVRIns/internal/email/temp"
	"HVRIns/internal/fbdata"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/proxy"
	resultpkg "HVRIns/internal/result"
	appsettings "HVRIns/internal/settings"
	"HVRIns/internal/settings/model"

	// Facebook platform implementations — named imports trigger init() registration
	// web register (named: use RandomRegInput etc.)
	// Skeleton platforms (blank import — only need init() registration)
	s399reg "HVRIns/internal/instagram/register/android/s399"
	_ "HVRIns/internal/instagram/register/chrome"
	ioshttpreg "HVRIns/internal/instagram/register/ioshttp"
	webandroidreg "HVRIns/internal/instagram/register/webandroid"

	// Verify platforms
	_ "HVRIns/internal/instagram/verify/android"
	_ "HVRIns/internal/instagram/verify/android/appmessv3"
	_ "HVRIns/internal/instagram/verify/android/s23"
	_ "HVRIns/internal/instagram/verify/android/s273"
	_ "HVRIns/internal/instagram/verify/android/s399"
	_ "HVRIns/internal/instagram/verify/android/s415"
	_ "HVRIns/internal/instagram/verify/android/s425"
	_ "HVRIns/internal/instagram/verify/android/s435"
	_ "HVRIns/internal/instagram/verify/android/s445"
	_ "HVRIns/internal/instagram/verify/android/s455"
	_ "HVRIns/internal/instagram/verify/android/s550v2"
	_ "HVRIns/internal/instagram/verify/android/s551v2"
	_ "HVRIns/internal/instagram/verify/android/s552v2"
	_ "HVRIns/internal/instagram/verify/android/s553v2"
	_ "HVRIns/internal/instagram/verify/android/s554v2"
	_ "HVRIns/internal/instagram/verify/android/s555"
	_ "HVRIns/internal/instagram/verify/android/s555v2"
	_ "HVRIns/internal/instagram/verify/android/s556"
	_ "HVRIns/internal/instagram/verify/android/s556v2"
	_ "HVRIns/internal/instagram/verify/android/s557"
	_ "HVRIns/internal/instagram/verify/android/s557v2"
	_ "HVRIns/internal/instagram/verify/android/s558"
	_ "HVRIns/internal/instagram/verify/android/s558v2"
	_ "HVRIns/internal/instagram/verify/android/s559"
	_ "HVRIns/internal/instagram/verify/android/s559v2"
	_ "HVRIns/internal/instagram/verify/android/s560"
	_ "HVRIns/internal/instagram/verify/android/s560v2"
	_ "HVRIns/internal/instagram/verify/android/s560v3"
	_ "HVRIns/internal/instagram/verify/android/s561"
	_ "HVRIns/internal/instagram/verify/android/s561v2"
	_ "HVRIns/internal/instagram/verify/android/s561v3"
	_ "HVRIns/internal/instagram/verify/android/s561v4s21"
	_ "HVRIns/internal/instagram/verify/android/s561v4s23"
	_ "HVRIns/internal/instagram/verify/android/s562"
	_ "HVRIns/internal/instagram/verify/android/s562v3"
	_ "HVRIns/internal/instagram/verify/android/s562v4s21"
	_ "HVRIns/internal/instagram/verify/android/s562v4s23"
	_ "HVRIns/internal/instagram/verify/android/s563"
	_ "HVRIns/internal/instagram/verify/android/s563s21"
	_ "HVRIns/internal/instagram/verify/android/s563v2"
	_ "HVRIns/internal/instagram/verify/android/s563v3s21"
	_ "HVRIns/internal/instagram/verify/android/s563v4s21"
	_ "HVRIns/internal/instagram/verify/android/s563v4s23"
	_ "HVRIns/internal/instagram/verify/android/s563v5s21"
	_ "HVRIns/internal/instagram/verify/android/s563v5s23"
	_ "HVRIns/internal/instagram/verify/android/s563v6s21"
	_ "HVRIns/internal/instagram/verify/android/s563v6s23"
	_ "HVRIns/internal/instagram/verify/android/s564v1s21"
	_ "HVRIns/internal/instagram/verify/android/s564v1s23"
	_ "HVRIns/internal/instagram/verify/android/s564v2s21"
	_ "HVRIns/internal/instagram/verify/android/s564v2s23"
	_ "HVRIns/internal/instagram/verify/android/s564v3s21"
	_ "HVRIns/internal/instagram/verify/android/s564v3s23"
	_ "HVRIns/internal/instagram/verify/android/s565s21"
	_ "HVRIns/internal/instagram/verify/android/s565s23"
	_ "HVRIns/internal/instagram/verify/android/s565v2s21"
	_ "HVRIns/internal/instagram/verify/android/s565v2s23"
	_ "HVRIns/internal/instagram/verify/ios/ios420"
	_ "HVRIns/internal/instagram/verify/ios/ios421"
	_ "HVRIns/internal/instagram/verify/ios/ios422"
	_ "HVRIns/internal/instagram/verify/ios/ios423"
	_ "HVRIns/internal/instagram/verify/ios/ios424"
	_ "HVRIns/internal/instagram/verify/ios/ios425"
	_ "HVRIns/internal/instagram/verify/ios/ios426"
	_ "HVRIns/internal/instagram/verify/ios/ios427"
	_ "HVRIns/internal/instagram/verify/ios/ios428"
	_ "HVRIns/internal/instagram/verify/ios/ios429"
	_ "HVRIns/internal/instagram/verify/ios/ios430"
	_ "HVRIns/internal/instagram/verify/ios/ios431"
	_ "HVRIns/internal/instagram/verify/ios/ios432"
	_ "HVRIns/internal/instagram/verify/ios/ios433"
	_ "HVRIns/internal/instagram/verify/ios/ios434"
	_ "HVRIns/internal/instagram/verify/ios/ios435"
	_ "HVRIns/internal/instagram/verify/ios/ios436"
	_ "HVRIns/internal/instagram/verify/ios/ios437"
	_ "HVRIns/internal/instagram/verify/ios/ios438"
	_ "HVRIns/internal/instagram/verify/ios/ios439"
	_ "HVRIns/internal/instagram/verify/ios/ios440"
	_ "HVRIns/internal/instagram/verify/ios/ios441"
	_ "HVRIns/internal/instagram/verify/ios/ios442"
	_ "HVRIns/internal/instagram/verify/ios/ios443"
	_ "HVRIns/internal/instagram/verify/ios/ios444"
	_ "HVRIns/internal/instagram/verify/ios/ios445"
	_ "HVRIns/internal/instagram/verify/ios/ios446"
	_ "HVRIns/internal/instagram/verify/ios/ios447"
	_ "HVRIns/internal/instagram/verify/ios/ios448"
	_ "HVRIns/internal/instagram/verify/ios/ios449"
	_ "HVRIns/internal/instagram/verify/ios/ios450"
	_ "HVRIns/internal/instagram/verify/ios/ios451"
	_ "HVRIns/internal/instagram/verify/ios/ios452"
	_ "HVRIns/internal/instagram/verify/ios/ios453"
	_ "HVRIns/internal/instagram/verify/ios/ios454"
	_ "HVRIns/internal/instagram/verify/ios/ios455"
	_ "HVRIns/internal/instagram/verify/ios/ios456"
	_ "HVRIns/internal/instagram/verify/ios/ios457"
	_ "HVRIns/internal/instagram/verify/ios/ios458"
	_ "HVRIns/internal/instagram/verify/ios/ios459"
	_ "HVRIns/internal/instagram/verify/ios/ios460"
	_ "HVRIns/internal/instagram/verify/ios/ios461"
	_ "HVRIns/internal/instagram/verify/ios/ios462"
	_ "HVRIns/internal/instagram/verify/ios/ios463"
	_ "HVRIns/internal/instagram/verify/ios/ios464"
	_ "HVRIns/internal/instagram/verify/ios/ios465"
	_ "HVRIns/internal/instagram/verify/ios/ios466"
	_ "HVRIns/internal/instagram/verify/ios/ios467"
	_ "HVRIns/internal/instagram/verify/ios/ios468"
	_ "HVRIns/internal/instagram/verify/ios/ios469"
	_ "HVRIns/internal/instagram/verify/ios/ios470"
	_ "HVRIns/internal/instagram/verify/ios/ios471"
	_ "HVRIns/internal/instagram/verify/ios/ios472"
	_ "HVRIns/internal/instagram/verify/ios/ios473"
	_ "HVRIns/internal/instagram/verify/ios/ios474"
	_ "HVRIns/internal/instagram/verify/ios/ios475"
	_ "HVRIns/internal/instagram/verify/ios/ios476"
	_ "HVRIns/internal/instagram/verify/ios/ios477"
	_ "HVRIns/internal/instagram/verify/ios/ios478"
	_ "HVRIns/internal/instagram/verify/ios/ios479"
	_ "HVRIns/internal/instagram/verify/ios/ios480"
	_ "HVRIns/internal/instagram/verify/ios/ios481"
	_ "HVRIns/internal/instagram/verify/ios/ios482"
	_ "HVRIns/internal/instagram/verify/ios/ios483"
	_ "HVRIns/internal/instagram/verify/ios/ios484"
	_ "HVRIns/internal/instagram/verify/ios/ios485"
	_ "HVRIns/internal/instagram/verify/ios/ios486"
	_ "HVRIns/internal/instagram/verify/ios/ios487"
	_ "HVRIns/internal/instagram/verify/ios/ios488"
	_ "HVRIns/internal/instagram/verify/ios/ios489"
	_ "HVRIns/internal/instagram/verify/ios/ios490"
	_ "HVRIns/internal/instagram/verify/ios/ios491"
	_ "HVRIns/internal/instagram/verify/ios/ios492"
	_ "HVRIns/internal/instagram/verify/ios/ios493"
	_ "HVRIns/internal/instagram/verify/ios/ios494"
	_ "HVRIns/internal/instagram/verify/ios/ios495"
	_ "HVRIns/internal/instagram/verify/ios/ios496"
	_ "HVRIns/internal/instagram/verify/ios/ios497"
	_ "HVRIns/internal/instagram/verify/ios/ios498"
	_ "HVRIns/internal/instagram/verify/ios/ios499"
	_ "HVRIns/internal/instagram/verify/ios/ios500"
	_ "HVRIns/internal/instagram/verify/ios/ios501"
	_ "HVRIns/internal/instagram/verify/ios/ios502"
	_ "HVRIns/internal/instagram/verify/ios/ios503"
	_ "HVRIns/internal/instagram/verify/ios/ios504"
	_ "HVRIns/internal/instagram/verify/ios/ios505"
	_ "HVRIns/internal/instagram/verify/ios/ios506"
	_ "HVRIns/internal/instagram/verify/ios/ios507"
	_ "HVRIns/internal/instagram/verify/ios/ios508"
	_ "HVRIns/internal/instagram/verify/ios/ios509"
	_ "HVRIns/internal/instagram/verify/ios/ios510"
	_ "HVRIns/internal/instagram/verify/ios/ios511"
	_ "HVRIns/internal/instagram/verify/ios/ios512"
	_ "HVRIns/internal/instagram/verify/ios/ios513"
	_ "HVRIns/internal/instagram/verify/ios/ios514"
	_ "HVRIns/internal/instagram/verify/ios/ios515"
	_ "HVRIns/internal/instagram/verify/ios/ios516"
	_ "HVRIns/internal/instagram/verify/ios/ios517"
	_ "HVRIns/internal/instagram/verify/ios/ios518"
	_ "HVRIns/internal/instagram/verify/ios/ios519"
	_ "HVRIns/internal/instagram/verify/ios/ios520"
	_ "HVRIns/internal/instagram/verify/ios/ios521"
	_ "HVRIns/internal/instagram/verify/ios/ios522"
	_ "HVRIns/internal/instagram/verify/ios/ios523"
	_ "HVRIns/internal/instagram/verify/ios/ios524"
	_ "HVRIns/internal/instagram/verify/ios/ios525"
	_ "HVRIns/internal/instagram/verify/ios/ios526"
	_ "HVRIns/internal/instagram/verify/ios/ios527"
	_ "HVRIns/internal/instagram/verify/ios/ios528"
	_ "HVRIns/internal/instagram/verify/ios/ios529"
	_ "HVRIns/internal/instagram/verify/ios/ios530"
	_ "HVRIns/internal/instagram/verify/ios/ios531"
	_ "HVRIns/internal/instagram/verify/ios/ios532"
	_ "HVRIns/internal/instagram/verify/ios/ios533"
	_ "HVRIns/internal/instagram/verify/ios/ios534"
	_ "HVRIns/internal/instagram/verify/ios/ios535"
	_ "HVRIns/internal/instagram/verify/ios/ios536"
	_ "HVRIns/internal/instagram/verify/ios/ios537"
	_ "HVRIns/internal/instagram/verify/ios/ios538"
	_ "HVRIns/internal/instagram/verify/ios/ios539"
	_ "HVRIns/internal/instagram/verify/ios/ios540"
	_ "HVRIns/internal/instagram/verify/ios/ios541"
	_ "HVRIns/internal/instagram/verify/ios/ios542"
	_ "HVRIns/internal/instagram/verify/ios/ios543"
	_ "HVRIns/internal/instagram/verify/ios/ios544"
	_ "HVRIns/internal/instagram/verify/ios/ios545"
	_ "HVRIns/internal/instagram/verify/ios/ios546"
	_ "HVRIns/internal/instagram/verify/ios/ios547"
	_ "HVRIns/internal/instagram/verify/ios/ios548"
	_ "HVRIns/internal/instagram/verify/ios/ios549"
	_ "HVRIns/internal/instagram/verify/ios/ios550"
	_ "HVRIns/internal/instagram/verify/ios/ios551"
	_ "HVRIns/internal/instagram/verify/ios/ios552"
	_ "HVRIns/internal/instagram/verify/ios/ios553"
	_ "HVRIns/internal/instagram/verify/ios/ios554"
	_ "HVRIns/internal/instagram/verify/ios/ios555"
	_ "HVRIns/internal/instagram/verify/ios/ios556"
	_ "HVRIns/internal/instagram/verify/ios/ios557"
	_ "HVRIns/internal/instagram/verify/ios/ios558"
	_ "HVRIns/internal/instagram/verify/ios/ios559"
	_ "HVRIns/internal/instagram/verify/ios/ios560"
	_ "HVRIns/internal/instagram/verify/ios/ios561"
	_ "HVRIns/internal/instagram/verify/ios/ios562"
	_ "HVRIns/internal/instagram/verify/ios/ios564"
	_ "HVRIns/internal/instagram/verify/ios/iosmess"
	_ "HVRIns/internal/instagram/verify/web"
	_ "HVRIns/internal/instagram/verify/webandroid"
	_ "HVRIns/internal/instagram/verify/webrequest"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// runResources — bundle native resources owned bởi 1 run của RunRegister (Task 3).
//
// Lifecycle ownership: pool được create LOCAL trong run (constructor `newRunResources`),
// rồi assign vào các global `ioshttpreg.SharedSessionPool` / `webandroidreg.SharedSessionPool`
// để workers backward-compat đọc qua global. Spawner defer gọi `res.Cleanup()` SAU `wg.Wait()`
// — đảm bảo:
//
//  1. Đóng đúng pool của run này (qua local ref) — không phụ thuộc trạng thái global.
//  2. Sau khi đóng, nil global ref nếu nó còn trỏ vào pool của run này — tránh stale
//     pointer sau Stop (force-cleanup ticker, DebugMemory… đọc trúng pool đã chết).
//  3. Không bao giờ overwrite/clear pool của run mới (state machine ở Task 1 đã chặn
//     overlap, đây là defense-in-depth).
//
// Mở rộng: thêm field/Close() ở đây nếu run cần own thêm resource (transport, cache…).
type runResources struct {
	runID    string
	iosPool  *ioshttpreg.SessionPool
	andrPool *webandroidreg.SessionPool
}

// newRunResources — chuẩn bị resources cho run dựa theo (các) platform được chọn.
// Hỗ trợ multi-version reg: nếu list chứa cả iOS và WebAndroid thì tạo cả 2 session pool.
// Caller chịu trách nhiệm assign global pointer (giữ tách biệt giữa "create" và "publish").
func newRunResources(runID string, platforms ...string) *runResources {
	r := &runResources{runID: runID}
	for _, platform := range platforms {
		switch platform {
		case instagram.PlatformIOS:
			if r.iosPool == nil {
				r.iosPool = ioshttpreg.NewSessionPool(0)
			}
		case instagram.PlatformWebAndroid:
			if r.andrPool == nil {
				r.andrPool = webandroidreg.NewSessionPool()
			}
		}
	}
	return r
}

// publishGlobals — gán pool local vào global để workers (đọc qua global) thấy được.
// Tách thành step riêng để spawner có thể decide thời điểm publish (sau validation).
func (r *runResources) publishGlobals() {
	if r.iosPool != nil {
		ioshttpreg.SharedSessionPool = r.iosPool
		slog.Info("REG pool create", "platform", "iOS", "runID", r.runID)
	}
	if r.andrPool != nil {
		webandroidreg.SharedSessionPool = r.andrPool
		slog.Info("REG pool create", "platform", "WebAndroid", "runID", r.runID)
	}
}

// Cleanup — gọi từ spawner defer SAU `wg.Wait()`. Đóng local pool + nil global nếu
// còn khớp local ref. Idempotent: an toàn khi gọi nhiều lần.
//
// Thứ tự (theo Task 3): cancel ctx (đã làm bởi state machine) → wg.Wait() (caller) →
// CloseAll (đã subsume CloseIdleConnsAll: mỗi session.client.CloseIdleConnections rồi
// drop session khỏi map) → state idle (caller).
func (r *runResources) Cleanup() {
	if r == nil {
		return
	}
	if r.iosPool != nil {
		n := r.iosPool.CloseAll()
		// Defense-in-depth: nil global ref nếu vẫn trỏ pool đã chết.
		// Race-free trong thực tế vì state machine chặn overlap, nhưng giúp DebugMemory
		// + force-cleanup ticker không thấy pool stale + báo "0 sessions" sai.
		if ioshttpreg.SharedSessionPool == r.iosPool {
			ioshttpreg.SharedSessionPool = nil
		}
		slog.Info("REG pool cleanup", "platform", "iOS", "runID", r.runID, "closedSessions", n)
		r.iosPool = nil
	}
	if r.andrPool != nil {
		n := r.andrPool.CloseAll()
		if webandroidreg.SharedSessionPool == r.andrPool {
			webandroidreg.SharedSessionPool = nil
		}
		slog.Info("REG pool cleanup", "platform", "WebAndroid", "runID", r.runID, "closedSessions", n)
		r.andrPool = nil
	}
}

// runState — state machine cho lifecycle của 1 long-running operation (register/verify).
//
// Transitions hợp lệ:
//
//	idle      → running   (Start sau khi qua validation, ngay trước khi spawn goroutine)
//	running   → stopping  (Stop user-initiated)
//	stopping  → idle      (spawner defer SAU khi wg.Wait + cleanup native resources)
//
// Mục đích chính: KHÔNG cho Start mới khi run cũ chưa thật sự dừng (workers + session pool +
// HTTP buffer vẫn còn). Hai-flag pattern cũ (registerCancel != nil + registerStopping bool)
// dễ lệch giữa StopRegister với spawner defer; state machine đơn ngữ nghĩa hơn.
type runState int32

const (
	runStateIdle     runState = 0
	runStateRunning  runState = 1
	runStateStopping runState = 2
)

// Account — cấu trúc đầy đủ mapping từ WeBM DataGridView + PlatformItems
type App struct {
	ctx           context.Context
	version       string
	accounts      []Account
	accountsMu    sync.RWMutex // bảo vệ a.accounts — RWMutex cho read-heavy (ListAccounts poll)
	activityCache sync.Map     // map[int]string — lock-free activity cache cho onStatus hot path
	removeLineMu  sync.Mutex   // serialize removeAccountLine — tránh read-modify-write race khi N workers cùng xóa

	// sourceFilePath — path file gốc user pick ở file mode (LoadAccountsFromFile).
	// OnAccountDone dùng để xóa dòng khỏi file gốc khi account verify = live.
	// Empty string = không phải file mode → không xóa gì.
	sourceFilePath     string
	sourceFilePathMu   sync.RWMutex
	verifyCancel       context.CancelFunc
	verifyWorkerCancel context.CancelFunc // cancel worker HTTP requests khi stop
	verifyMu           sync.Mutex
	verifyStopping     atomic.Bool // true sau Stop, false sau khi spawner defer xong cleanup
	registerCancel     context.CancelFunc
	registerMu         sync.Mutex
	registerGen        int // generation counter — chỉ dùng để chống event/cleanup stale (so sánh trong defer); KHÔNG dùng để gating Start
	// registerState — state machine idle/running/stopping (xem `runState`).
	// Gating Start nằm ở đây thay vì pattern cũ (registerCancel != nil + registerStopping bool).
	// Atomic để cho phép lock-free read từ helper accessors (IsRegisterRunning/IsRegisterStopping).
	// Mọi transition vẫn được serialize qua registerMu để pair với việc gán/clear registerCancel.
	registerState atomic.Int32
	isRunning     bool
	watcherCancel context.CancelFunc // folder watcher goroutine cancel
	// confirmedQuit — set true bởi RequestQuit() khi user đã confirm tắt app.
	// OnBeforeClose check flag này: false → block close + emit event để FE show dialog;
	// true → cho phép close. Tránh tắt nhầm khi nhấn X / Alt+F4 / taskbar close.
	confirmedQuit atomic.Bool
	emailPool     *email.CredPool   // shared pool mua batch email — tạo 1 lần mỗi run
	appSettings   model.AppSettings // nguồn sự thật cho settings runtime
	settingsMu    sync.RWMutex      // bảo vệ appSettings khi đọc/ghi đồng thời
	// sharedProxyMgr — proxy manager dùng cho Verify
	sharedProxyMgr        *proxy.Manager
	sharedProxyMgrKey     string       // cache key = provider+keys — recreate khi settings thay đổi
	sharedProxyMgrMu      sync.RWMutex // fast-path reader + serialize creation
	sharedProxyMgrVersion int64        // version tại lúc tạo mgr — so sánh với proxyConfigVersion
	// regProxyMgr — proxy manager riêng cho Register (có thể dùng chung với sharedProxyMgr nếu useVerifyProxyForReg=true)
	regProxyMgr        *proxy.Manager
	regProxyMgrKey     string
	regProxyMgrMu      sync.RWMutex
	regProxyMgrVersion int64
	proxyConfigVersion atomic.Int64 // tăng khi SaveSettings/SaveInteraction → invalidate cache

	// upload site runner
	uploadCh              chan string
	uploadSiteCancel      context.CancelFunc
	uploadSiteGen         int64 // generation counter — defer cleanup chỉ reset cancel nếu gen chưa đổi
	uploadSiteMu          sync.Mutex
	uploadStopping        atomic.Bool // soft-stop: loop chờ drain xong mới exit
	frontendHidden        atomic.Bool // true khi window minimize/hidden — batch emitter throttle để tiết kiệm IPC
	uploadStatsMu         sync.Mutex  // bảo vệ upload_stats.json (per-session)
	uploadLogMu           sync.Mutex  // bảo vệ upload_push_log.txt (per-session)
	uploadLogRotateCancel context.CancelFunc
	uploadSeenUIDs        sync.Map     // dedup: UID đã enqueue trong session hiện tại
	uploadRetryMu         sync.Mutex   // bảo vệ uploadRetryQueue
	uploadRetryQueue      []*retryItem // các batch fail đang chờ retry (theo retryAt)

	// currentResultPath — thư mục result của phiên đang chạy, dùng để ghi upload_push_log.txt
	currentResultPath string
	resultPathMu      sync.Mutex

	// regStats — thống kê reg theo version (API) trong run hiện tại. Reset mỗi RunRegister.
	regStatsMu sync.Mutex
	regStats   map[string]*regVersionStat
	// verifyStats — thống kê verify theo version (API). Reset mỗi RunVerify / RunRegister.
	verifyStatsMu sync.Mutex
	verifyStats   map[string]*regVersionStat
	// verifyPlatformRR — bộ đếm round-robin chia version verify theo từng account (multi-version).
	verifyPlatformRR atomic.Int64
	// mailDomainStats — thống kê verify theo domain mail. Reset mỗi RunVerify / RunRegister.
	mailDomainStatsMu sync.Mutex
	mailDomainStats   map[string]*mailDomainStat
	// buildUARegStats / buildUAVerStats — thống kê theo FBAV version (dùng chung mọi run).
	buildUARegStatsMu sync.Mutex
	buildUARegStats   map[string]*regVersionStat
	buildUAVerStatsMu sync.Mutex
	buildUAVerStats   map[string]*regVersionStat
	// poolFileSaved — số datr đã ghi vào Pool file trong run hiện tại. Reset mỗi RunRegister.
	poolFileSaved atomic.Int64
}

// accounts bắt đầu rỗng, dữ liệu thật được load từ thư mục nguồn trong startup().
func NewApp() *App {
	return &App{
		accounts: make([]Account, 0),
		uploadCh: make(chan string, 5000),
	}
}

// isAPIMode kiểm tra user đã chọn nguồn tài khoản "Mua từ API" (không load folder)
func isAPIMode(settings SettingsData) bool {
	return settings.General.AccountSource == "api"
}

// isCloneHVActive kiểm tra đã chọn API mode VÀ có đủ credentials để mua
func isCloneHVActive(settings SettingsData) bool {
	if !isAPIMode(settings) {
		return false
	}
	return settings.General.CloneHVUsername != "" && settings.General.CloneHVProductID != ""
}

// verifyNeedsEAA trả true nếu verify mode (UI ApiVerifyPlatform) yêu cầu EAAAAU (Android user
// token) và phải login Android để lấy trước khi verify.
// iOS (ios562) → false: dùng EAAAAAY trực tiếp, không cần Android login.
func verifyNeedsEAA(apiVerifyType string) bool {
	switch strings.ToLower(strings.TrimSpace(apiVerifyType)) {
	case "api android", "api token",
		"s22", "s23", "s24", "s25", "s26",
		"s415", "s425", "s435", "s445", "s455",
		"s550v2", "s551v2", "s552v2", "s553v2", "s554v2", "s555", "s555v2", "s556", "s556v2", "s557", "s557v2", "s558", "s558v2", "s559", "s559v2", "s560", "s560v2", "s560v3", "s561", "s561v2", "s561v3", "s561v99", "s561v4s21", "s561v4s23", "s562", "s562v3", "s562v4s21", "s562v4s23", "s563", "s563v2", "s563s21", "s563v3s21", "s563v4s21", "s563v4s23", "s563v5s21", "s563v5s23", "s563v6s21", "s563v6s23", "s564v1s21", "s564v1s23", "s564v2s21", "s564v2s23", "s564v3s21", "s564v3s23", "s565s21", "s565s23", "s565v2s21", "s565v2s23", "appmessv3", "s399", "s273":
		return true
	}
	return false
}

// verifyIsIOS — verify platform là FBIOS native (ios562/ios563) → cần token iOS EAAAAAY,
// KHÔNG dùng EAAAAU Android. Khi đó reg cookie-only không nên login Android lấy EAAAAU.
func verifyIsIOS(apiVerifyType string) bool {
	switch strings.ToLower(strings.TrimSpace(apiVerifyType)) {
	case "ios562", "ios563", "ios555", "ios550", "ios540", "ios530", "ios520", "ios564",
		"ios510", "ios500", "ios490", "ios480", "ios470", "ios460", "ios450",
		"ios440", "ios430", "ios420", "ios560",
		"ios421", "ios422", "ios423", "ios424", "ios425", "ios426", "ios427", "ios428", "ios429", "ios431", "ios432", "ios433", "ios434", "ios435", "ios436", "ios437", "ios438", "ios439", "ios441", "ios442", "ios443", "ios444", "ios445", "ios446", "ios447", "ios448", "ios449", "ios451", "ios452", "ios453", "ios454", "ios455", "ios456", "ios457", "ios458", "ios459", "ios461", "ios462", "ios463", "ios464", "ios465", "ios466", "ios467", "ios468", "ios469", "ios471", "ios472", "ios473", "ios474", "ios475", "ios476", "ios477", "ios478", "ios479", "ios481", "ios482", "ios483", "ios484", "ios485", "ios486", "ios487", "ios488", "ios489", "ios491", "ios492", "ios493", "ios494", "ios495", "ios496", "ios497", "ios498", "ios499", "ios501", "ios502", "ios503", "ios504", "ios505", "ios506", "ios507", "ios508", "ios509", "ios511", "ios512", "ios513", "ios514", "ios515", "ios516", "ios517", "ios518", "ios519", "ios521", "ios522", "ios523", "ios524", "ios525", "ios526", "ios527", "ios528", "ios529", "ios531", "ios532", "ios533", "ios534", "ios535", "ios536", "ios537", "ios538", "ios539", "ios541", "ios542", "ios543", "ios544", "ios545", "ios546", "ios547", "ios548", "ios549", "ios551", "ios552", "ios553", "ios554", "ios556", "ios557", "ios558", "ios559", "ios561":
		return true
	}
	return false
}

func preferUserAccessToken(current, incoming string) string {
	current = strings.TrimSpace(current)
	incoming = strings.TrimSpace(incoming)
	// Ưu tiên token MỚI hợp lệ (EAAAAU Android HOẶC EAAAAAY iOS) — KHÔNG giữ token cũ.
	// Trước đây chỉ check EAAAAU → token iOS EAAAAAY mới bị loại khi row đã có EAAAAU cũ.
	if strings.HasPrefix(incoming, "EAAAAU") || strings.HasPrefix(incoming, "EAAAAAY") {
		return incoming
	}
	if strings.HasPrefix(current, "EAAAAU") || strings.HasPrefix(current, "EAAAAAY") {
		return current
	}
	if incoming != "" {
		return incoming
	}
	return current
}

func platformNeedsAndroidLoginToken(platform string) bool {
	if isRegPlatformSxxx(platform) {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case instagram.PlatformAndroid,
		instagram.PlatformS22, instagram.PlatformS23, instagram.PlatformS24, instagram.PlatformS25, instagram.PlatformS26,
		instagram.PlatformS555, instagram.PlatformS556, instagram.PlatformS557, instagram.PlatformS558,
		instagram.PlatformS559, instagram.PlatformS559V2, instagram.PlatformS560, instagram.PlatformS560V2, instagram.PlatformS560V3,
		instagram.PlatformS561, instagram.PlatformS561V2, instagram.PlatformS561V4S21, instagram.PlatformS561V4S23, instagram.PlatformS562, instagram.PlatformS562V4S21, instagram.PlatformS562V4S23, instagram.PlatformS563, instagram.PlatformS563V2, instagram.PlatformS563S21, instagram.PlatformS563V3S21, instagram.PlatformS563V4S21, instagram.PlatformS399,
		instagram.PlatformS273:
		return true
	}
	return false
}

func applyVerifyPlatformUAConfig(cfg InteractionConfig) InteractionConfig {
	if uaCfg, ok := cfg.VerifyPlatformUA[cfg.ApiVerifyPlatform]; ok {
		cfg.BuildUA = uaCfg.BuildUA
		cfg.AddVirtualSpecAndroid = uaCfg.AddVirtualSpecAndroid
		cfg.UseOriginalUA = uaCfg.UseOriginalUA
		cfg.ReplaceCarrier = uaCfg.ReplaceCarrier
		// TrackingID là global (TrackingIDReg/TrackingIDVer), không override từ per-platform.
		if uaCfg.UaPoolKey != "" {
			cfg.UaPoolKey = uaCfg.UaPoolKey
		}
	}
	return cfg
}

func applyRegPlatformUAConfig(cfg InteractionConfig, platform string) InteractionConfig {
	if platform == "" {
		platform = cfg.ApiRegPlatform
	}
	if uaCfg, ok := cfg.RegPlatformUA[platform]; ok {
		cfg.BuildUA = uaCfg.BuildUA
		cfg.AddVirtualSpecAndroid = uaCfg.AddVirtualSpecAndroid
		cfg.UseOriginalUA = uaCfg.UseOriginalUA
		cfg.ReplaceCarrier = uaCfg.ReplaceCarrier
		// TrackingID là global (TrackingIDReg/TrackingIDVer), không override từ per-platform.
		if uaCfg.UaPoolKey != "" {
			cfg.UaPoolKey = uaCfg.UaPoolKey
		}
	}
	return cfg
}

// boughtMailMu bảo vệ ghi file mail (nhiều goroutine pool có thể mua/ghi đồng thời).
var boughtMailMu sync.Mutex

// rentMailDir trả về thư mục lưu mail rent: Config/RentMail (cạnh exe). "" nếu lỗi.
func rentMailDir() string {
	dir := filepath.Join(AppDataDir(), "Config", "RentMail")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	return dir
}

// credsToLines format mỗi mail 1 dòng: email|password|refresh_token|client_id
func credsToLines(creds []emailrent.EmailCred) string {
	var sb strings.Builder
	for _, c := range creds {
		if c.Email == "" {
			continue
		}
		sb.WriteString(c.Email + "|" + c.Password + "|" + c.RefreshToken + "|" + c.ClientId + "\n")
	}
	return sb.String()
}

// writeMailFile ghi creds ra Config/RentMail/<name>.txt. appendMode=true → nối thêm,
// false → ghi đè. Thread-safe. <name> ví dụ "bought_mail30s", "used_store1s".
func writeMailFile(name string, creds []emailrent.EmailCred, appendMode bool) {
	dir := rentMailDir()
	if dir == "" || len(creds) == 0 {
		return
	}
	data := credsToLines(creds)
	if data == "" {
		return
	}
	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	boughtMailMu.Lock()
	defer boughtMailMu.Unlock()
	f, err := os.OpenFile(filepath.Join(dir, name+".txt"), flags, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(data)
}

// appendBoughtMails ghi (append) batch mail VỪA MUA ra Config/RentMail/bought_<provider>.txt.
// Crash-safe record của MỌI lần mua. Chỉ gọi từ CredPool.OnBought (mua thật).
func appendBoughtMails(provider string, creds []emailrent.EmailCred) {
	if strings.TrimSpace(provider) == "" {
		return
	}
	writeMailFile("bought_"+provider, creds, true)
}

// persistUsedUnused phân loại mail trong pool rồi ghi 2 file (provider lấy từ pool.Provider):
//   - used_<provider>.txt   (APPEND): mail đã đưa vào verify (FB có thể đã consume).
//   - unused_<provider>.txt (APPEND): mail mua dư CHƯA dùng, sạch → lần sau tái dùng để ver tiếp.
//
// Gọi ở run-end (mọi verify xong) HOẶC trước Close. Once-guard đảm bảo dump 1 lần/pool.
func persistUsedUnused(pool *emailrent.CredPool) {
	if pool == nil || strings.TrimSpace(pool.Provider) == "" {
		return
	}
	if !pool.TryMarkPersisted() {
		return // đã dump rồi (gọi từ điểm khác) → tránh ghi trùng
	}
	used, unused := pool.PartitionUsedUnused()
	writeMailFile("used_"+pool.Provider, used, true)
	writeMailFile("unused_"+pool.Provider, unused, true)
}

// verifyPlatformFromType chuyển đổi giá trị ApiVerifyTokenType (từ UI dropdown)
// thành platform string cho runner.RunConfig.VerifyPlatform.
func verifyPlatformFromType(apiVerifyType string) string {
	switch apiVerifyType {
	case "api android":
		return instagram.PlatformS23 // Android token-based (b-graph Bloks API)
	case "api mfb":
		return instagram.PlatformWeb // MFB web form (m.facebook.com/changeemail)
	case "api token":
		return instagram.PlatformAndroid // cũ: api.facebook.com/method/
	case "api web andr":
		return instagram.PlatformWebAndroid // Chrome Android web
	case "s23":
		return instagram.PlatformS23
	case "s22", "s24", "s25", "s26":
		return instagram.PlatformS23 // verify dùng chung S23 Android token flow
	case "s415":
		return instagram.PlatformS415 // Samsung S23 + FB API 415
	case "s425":
		return instagram.PlatformS425 // Samsung S23 + FB API 425
	case "s435":
		return instagram.PlatformS435 // Samsung S23 + FB API 435
	case "s445":
		return instagram.PlatformS445 // Samsung S23 + FB API 445
	case "s455":
		return instagram.PlatformS455 // Samsung S23 + FB API 455
	case "s557":
		return instagram.PlatformS557 // Samsung S23 + FB API 557
	case "s555":
		return instagram.PlatformS555 // Samsung S23 + FB API 555
	case "s555v2":
		return instagram.PlatformS555V2 // Samsung S23 + FB API 555 v2
	case "s556":
		return instagram.PlatformS556 // Samsung S23 + FB API 556
	case "s558":
		return instagram.PlatformS558 // Samsung S23 + FB API 558
	case "s558v2":
		return instagram.PlatformS558V2 // Samsung S23 + FB API 558 v2
	case "s556v2":
		return instagram.PlatformS556V2 // Samsung S23 + FB API 556 v2
	case "s557v2":
		return instagram.PlatformS557V2 // Samsung S23 + FB API 557 v2
	case "s553v2":
		return instagram.PlatformS553V2 // Samsung S23 + FB API 553 v2
	case "s554v2":
		return instagram.PlatformS554V2 // Samsung S23 + FB API 554 v2
	case "s551v2":
		return instagram.PlatformS551V2 // Samsung S23 + FB API 551 v2
	case "s552v2":
		return instagram.PlatformS552V2 // Samsung S23 + FB API 552 v2
	case "s550v2":
		return instagram.PlatformS550V2 // Samsung S23 + FB API 550 v2
	case "s559":
		return instagram.PlatformS559 // Samsung S23 + FB API 559
	case "s559v2":
		return instagram.PlatformS559V2 // Samsung S23 + FB API 559 v2
	case "s560":
		return instagram.PlatformS560 // Samsung S23 + FB API 560
	case "s560v2":
		return instagram.PlatformS560V2 // Samsung S23 + FB API 560 v2
	case "s560v3":
		return instagram.PlatformS560V3 // Samsung S23 + FB API 560 v3
	case "s561":
		return instagram.PlatformS561 // Samsung S23 + FB API 561
	case "s561v2":
		return instagram.PlatformS561V2 // Samsung S23 + FB API 561 v2
	case "s561v3":
		return instagram.PlatformS561V3 // Samsung S23 + FB API 561 v3
	case "s561v99":
		return instagram.PlatformS561V3 // s561v99 = reg-only (Type Reg 2); verify dùng chung s561v3 (cùng FBAV/561.0.0.42.67)
	case "s561v4s21":
		return instagram.PlatformS561V4S21 // Samsung Galaxy S21+ (SM-G996B) + FB API 561 v2
	case "s561v4s23":
		return instagram.PlatformS561V4S23 // Samsung Galaxy S23 (SM-S911B) + FB API 561 v2 (capture mới)
	case "s562":
		return instagram.PlatformS562 // Samsung S23 + FB API 562
	case "s562v3":
		return instagram.PlatformS562V3 // Samsung S23 + FB API 562 v3
	case "s562v4s21":
		return instagram.PlatformS562V4S21 // Samsung Galaxy S21+ + FB API 562 v4 (capture mới)
	case "s562v4s23":
		return instagram.PlatformS562V4S23 // Samsung Galaxy S23 + FB API 562 v4 (capture mới)
	case "s563":
		return instagram.PlatformS563 // Samsung S23 + FB API 563
	case "s563v2":
		return instagram.PlatformS563V2 // Samsung S23 + FB API 563 v2
	case "s563s21":
		return instagram.PlatformS563S21 // Samsung Galaxy S21+ (SM-G996B) + FB API 563
	case "s563v3s21":
		return instagram.PlatformS563V3S21 // Samsung Galaxy S21+ (SM-G996B) + FB API 563 v3 (capture mới)
	case "s563v4s21":
		return instagram.PlatformS563V4S21 // Samsung Galaxy S21+ (SM-G996B) + FB API 563 v4 (capture mới)
	case "s563v4s23":
		return instagram.PlatformS563V4S23 // Samsung Galaxy S23 (SM-S911B) + FB API 563 v4
	case "s563v5s21":
		return instagram.PlatformS563V5S21 // Samsung Galaxy S21+ (SM-G996B) + FB API 563 v5 (theme FDS only)
	case "s563v5s23":
		return instagram.PlatformS563V5S23 // Samsung Galaxy S23 (SM-S911B) + FB API 563 v5
	case "s563v6s21":
		return instagram.PlatformS563V6S21 // Samsung Galaxy S21+ (SM-G996B) + FB API 563 v6
	case "s563v6s23":
		return instagram.PlatformS563V6S23 // Samsung Galaxy S23 (SM-S911B) + FB API 563 v6
	case "s564v1s21":
		return instagram.PlatformS564V1S21 // Samsung Galaxy S21+ (SM-G996B) + FB API 564 v1 (capture mới)
	case "s564v1s23":
		return instagram.PlatformS564V1S23 // Samsung Galaxy S23 (SM-S911B) + FB API 564 v1
	case "s564v2s21":
		return instagram.PlatformS564V2S21 // Samsung Galaxy S21+ (SM-G996B) + FB API 564 v2 (theme FDS only)
	case "s564v2s23":
		return instagram.PlatformS564V2S23 // Samsung Galaxy S23 (SM-S911B) + FB API 564 v2
	case "s564v3s21":
		return instagram.PlatformS564V3S21 // Samsung Galaxy S21+ (SM-G996B) + FB API 564 v3
	case "s564v3s23":
		return instagram.PlatformS564V3S23 // Samsung Galaxy S23 (SM-S911B) + FB API 564 v3
	case "s565s21":
		return instagram.PlatformS565S21 // Samsung Galaxy S21+ (SM-G996B) + FB API 565
	case "s565s23":
		return instagram.PlatformS565S23 // Samsung Galaxy S23 (SM-S911B) + FB API 565
	case "s565v2s21":
		return instagram.PlatformS565V2S21 // Samsung Galaxy S21+ (SM-G996B) + FB API 565 v2
	case "s565v2s23":
		return instagram.PlatformS565V2S23 // Samsung Galaxy S23 (SM-S911B) + FB API 565 v2
	case "appmessv3":
		return instagram.PlatformAppMessV3 // Messenger (Orca) v529 — Ver login kiểu V3
	case "appmessv3_535":
		return instagram.PlatformAppMessV3_535 // Messenger Orca 535
	case "appmessv3_545":
		return instagram.PlatformAppMessV3_545 // Messenger Orca 545
	case "appmessv3_555":
		return instagram.PlatformAppMessV3_555 // Messenger Orca 555
	case "appmessv3_563":
		return instagram.PlatformAppMessV3_563 // Messenger Orca 563
	case "appmessv3_564":
		return instagram.PlatformAppMessV3_564 // Messenger Orca 564
	case "appmessv3_565":
		return instagram.PlatformAppMessV3_565 // Messenger Orca 565
	case "appmessv3_525":
		return instagram.PlatformAppMessV3_525 // Messenger Orca 525
	case "appmessv3_515":
		return instagram.PlatformAppMessV3_515 // Messenger Orca 515
	case "appmessv3_505":
		return instagram.PlatformAppMessV3_505 // Messenger Orca 505
	case "appmessv3_490":
		return instagram.PlatformAppMessV3_490 // Messenger Orca 490
	case "iosmess":
		return instagram.PlatformIOSMess // Messenger Lite iOS — add-mail + confirm + live/die
	case "s399":
		return instagram.PlatformS399 // Samsung S23 + FB API 399
	case "s273":
		return instagram.PlatformS273 // Vivo V2242A + FB API 273 (b-api /method/user.xxx)
	case "ios562":
		return instagram.PlatformIOS562 // iPhone + FB iOS app (FBIOS) API 562
	case "ios563":
		return instagram.PlatformIOS563 // iPhone + FB iOS app (FBIOS) API 563 — single-shot
	case "ios555":
		return instagram.PlatformIOS555 // iPhone + FB iOS app (FBIOS) API 555 (clone 562)
	case "ios564":
		return instagram.PlatformIOS564 // iPhone + FB iOS app (FBIOS) API 564 (clone 555)
	case "ios550":
		return instagram.PlatformIOS550 // iPhone + FB iOS app (FBIOS) API 550 (clone 562)
	case "ios540":
		return instagram.PlatformIOS540 // iPhone + FB iOS app (FBIOS) API 540 (clone 562)
	case "ios530":
		return instagram.PlatformIOS530 // iPhone + FB iOS app (FBIOS) API 530 (clone 562)
	case "ios520":
		return instagram.PlatformIOS520 // iPhone + FB iOS app (FBIOS) API 520 (clone 562)
	case "ios510":
		return instagram.PlatformIOS510 // iPhone + FB iOS app (FBIOS) API 510 (clone 562)
	case "ios500":
		return instagram.PlatformIOS500 // iPhone + FB iOS app (FBIOS) API 500 (clone 562)
	case "ios490":
		return instagram.PlatformIOS490 // iPhone + FB iOS app (FBIOS) API 490 (clone 562)
	case "ios480":
		return instagram.PlatformIOS480 // iPhone + FB iOS app (FBIOS) API 480 (clone 562)
	case "ios470":
		return instagram.PlatformIOS470 // iPhone + FB iOS app (FBIOS) API 470 (clone 562)
	case "ios460":
		return instagram.PlatformIOS460 // iPhone + FB iOS app (FBIOS) API 460 (clone 562)
	case "ios450":
		return instagram.PlatformIOS450 // iPhone + FB iOS app (FBIOS) API 450 (clone 562)
	case "ios440":
		return instagram.PlatformIOS440 // iPhone + FB iOS app (FBIOS) API 440 (clone 450)
	case "ios430":
		return instagram.PlatformIOS430 // iPhone + FB iOS app (FBIOS) API 430 (clone 562)
	case "ios420":
		return instagram.PlatformIOS420 // iPhone + FB iOS app (FBIOS) API 420 (clone 562)
	case "ios421":
		return instagram.PlatformIOS421 // clone ios420
	case "ios422":
		return instagram.PlatformIOS422 // clone ios420
	case "ios423":
		return instagram.PlatformIOS423 // clone ios420
	case "ios424":
		return instagram.PlatformIOS424 // clone ios420
	case "ios425":
		return instagram.PlatformIOS425 // clone ios420
	case "ios426":
		return instagram.PlatformIOS426 // clone ios420
	case "ios427":
		return instagram.PlatformIOS427 // clone ios420
	case "ios428":
		return instagram.PlatformIOS428 // clone ios420
	case "ios429":
		return instagram.PlatformIOS429 // clone ios420
	case "ios431":
		return instagram.PlatformIOS431 // clone ios430
	case "ios432":
		return instagram.PlatformIOS432 // clone ios430
	case "ios433":
		return instagram.PlatformIOS433 // clone ios430
	case "ios434":
		return instagram.PlatformIOS434 // clone ios430
	case "ios435":
		return instagram.PlatformIOS435 // clone ios430
	case "ios436":
		return instagram.PlatformIOS436 // clone ios430
	case "ios437":
		return instagram.PlatformIOS437 // clone ios430
	case "ios438":
		return instagram.PlatformIOS438 // clone ios430
	case "ios439":
		return instagram.PlatformIOS439 // clone ios430
	case "ios441":
		return instagram.PlatformIOS441 // clone ios440
	case "ios442":
		return instagram.PlatformIOS442 // clone ios440
	case "ios443":
		return instagram.PlatformIOS443 // clone ios440
	case "ios444":
		return instagram.PlatformIOS444 // clone ios440
	case "ios445":
		return instagram.PlatformIOS445 // clone ios440
	case "ios446":
		return instagram.PlatformIOS446 // clone ios440
	case "ios447":
		return instagram.PlatformIOS447 // clone ios440
	case "ios448":
		return instagram.PlatformIOS448 // clone ios440
	case "ios449":
		return instagram.PlatformIOS449 // clone ios440
	case "ios451":
		return instagram.PlatformIOS451 // clone ios450
	case "ios452":
		return instagram.PlatformIOS452 // clone ios450
	case "ios453":
		return instagram.PlatformIOS453 // clone ios450
	case "ios454":
		return instagram.PlatformIOS454 // clone ios450
	case "ios455":
		return instagram.PlatformIOS455 // clone ios450
	case "ios456":
		return instagram.PlatformIOS456 // clone ios450
	case "ios457":
		return instagram.PlatformIOS457 // clone ios450
	case "ios458":
		return instagram.PlatformIOS458 // clone ios450
	case "ios459":
		return instagram.PlatformIOS459 // clone ios450
	case "ios461":
		return instagram.PlatformIOS461 // clone ios460
	case "ios462":
		return instagram.PlatformIOS462 // clone ios460
	case "ios463":
		return instagram.PlatformIOS463 // clone ios460
	case "ios464":
		return instagram.PlatformIOS464 // clone ios460
	case "ios465":
		return instagram.PlatformIOS465 // clone ios460
	case "ios466":
		return instagram.PlatformIOS466 // clone ios460
	case "ios467":
		return instagram.PlatformIOS467 // clone ios460
	case "ios468":
		return instagram.PlatformIOS468 // clone ios460
	case "ios469":
		return instagram.PlatformIOS469 // clone ios460
	case "ios471":
		return instagram.PlatformIOS471 // clone ios470
	case "ios472":
		return instagram.PlatformIOS472 // clone ios470
	case "ios473":
		return instagram.PlatformIOS473 // clone ios470
	case "ios474":
		return instagram.PlatformIOS474 // clone ios470
	case "ios475":
		return instagram.PlatformIOS475 // clone ios470
	case "ios476":
		return instagram.PlatformIOS476 // clone ios470
	case "ios477":
		return instagram.PlatformIOS477 // clone ios470
	case "ios478":
		return instagram.PlatformIOS478 // clone ios470
	case "ios479":
		return instagram.PlatformIOS479 // clone ios470
	case "ios481":
		return instagram.PlatformIOS481 // clone ios480
	case "ios482":
		return instagram.PlatformIOS482 // clone ios480
	case "ios483":
		return instagram.PlatformIOS483 // clone ios480
	case "ios484":
		return instagram.PlatformIOS484 // clone ios480
	case "ios485":
		return instagram.PlatformIOS485 // clone ios480
	case "ios486":
		return instagram.PlatformIOS486 // clone ios480
	case "ios487":
		return instagram.PlatformIOS487 // clone ios480
	case "ios488":
		return instagram.PlatformIOS488 // clone ios480
	case "ios489":
		return instagram.PlatformIOS489 // clone ios480
	case "ios491":
		return instagram.PlatformIOS491 // clone ios490
	case "ios492":
		return instagram.PlatformIOS492 // clone ios490
	case "ios493":
		return instagram.PlatformIOS493 // clone ios490
	case "ios494":
		return instagram.PlatformIOS494 // clone ios490
	case "ios495":
		return instagram.PlatformIOS495 // clone ios490
	case "ios496":
		return instagram.PlatformIOS496 // clone ios490
	case "ios497":
		return instagram.PlatformIOS497 // clone ios490
	case "ios498":
		return instagram.PlatformIOS498 // clone ios490
	case "ios499":
		return instagram.PlatformIOS499 // clone ios490
	case "ios501":
		return instagram.PlatformIOS501 // clone ios500
	case "ios502":
		return instagram.PlatformIOS502 // clone ios500
	case "ios503":
		return instagram.PlatformIOS503 // clone ios500
	case "ios504":
		return instagram.PlatformIOS504 // clone ios500
	case "ios505":
		return instagram.PlatformIOS505 // clone ios500
	case "ios506":
		return instagram.PlatformIOS506 // clone ios500
	case "ios507":
		return instagram.PlatformIOS507 // clone ios500
	case "ios508":
		return instagram.PlatformIOS508 // clone ios500
	case "ios509":
		return instagram.PlatformIOS509 // clone ios500
	case "ios511":
		return instagram.PlatformIOS511 // clone ios510
	case "ios512":
		return instagram.PlatformIOS512 // clone ios510
	case "ios513":
		return instagram.PlatformIOS513 // clone ios510
	case "ios514":
		return instagram.PlatformIOS514 // clone ios510
	case "ios515":
		return instagram.PlatformIOS515 // clone ios510
	case "ios516":
		return instagram.PlatformIOS516 // clone ios510
	case "ios517":
		return instagram.PlatformIOS517 // clone ios510
	case "ios518":
		return instagram.PlatformIOS518 // clone ios510
	case "ios519":
		return instagram.PlatformIOS519 // clone ios510
	case "ios521":
		return instagram.PlatformIOS521 // clone ios520
	case "ios522":
		return instagram.PlatformIOS522 // clone ios520
	case "ios523":
		return instagram.PlatformIOS523 // clone ios520
	case "ios524":
		return instagram.PlatformIOS524 // clone ios520
	case "ios525":
		return instagram.PlatformIOS525 // clone ios520
	case "ios526":
		return instagram.PlatformIOS526 // clone ios520
	case "ios527":
		return instagram.PlatformIOS527 // clone ios520
	case "ios528":
		return instagram.PlatformIOS528 // clone ios520
	case "ios529":
		return instagram.PlatformIOS529 // clone ios520
	case "ios531":
		return instagram.PlatformIOS531 // clone ios530
	case "ios532":
		return instagram.PlatformIOS532 // clone ios530
	case "ios533":
		return instagram.PlatformIOS533 // clone ios530
	case "ios534":
		return instagram.PlatformIOS534 // clone ios530
	case "ios535":
		return instagram.PlatformIOS535 // clone ios530
	case "ios536":
		return instagram.PlatformIOS536 // clone ios530
	case "ios537":
		return instagram.PlatformIOS537 // clone ios530
	case "ios538":
		return instagram.PlatformIOS538 // clone ios530
	case "ios539":
		return instagram.PlatformIOS539 // clone ios530
	case "ios541":
		return instagram.PlatformIOS541 // clone ios540
	case "ios542":
		return instagram.PlatformIOS542 // clone ios540
	case "ios543":
		return instagram.PlatformIOS543 // clone ios540
	case "ios544":
		return instagram.PlatformIOS544 // clone ios540
	case "ios545":
		return instagram.PlatformIOS545 // clone ios540
	case "ios546":
		return instagram.PlatformIOS546 // clone ios540
	case "ios547":
		return instagram.PlatformIOS547 // clone ios540
	case "ios548":
		return instagram.PlatformIOS548 // clone ios540
	case "ios549":
		return instagram.PlatformIOS549 // clone ios540
	case "ios551":
		return instagram.PlatformIOS551 // clone ios550
	case "ios552":
		return instagram.PlatformIOS552 // clone ios550
	case "ios553":
		return instagram.PlatformIOS553 // clone ios550
	case "ios554":
		return instagram.PlatformIOS554 // clone ios550
	case "ios556":
		return instagram.PlatformIOS556 // clone ios555
	case "ios557":
		return instagram.PlatformIOS557 // clone ios555
	case "ios558":
		return instagram.PlatformIOS558 // clone ios555
	case "ios559":
		return instagram.PlatformIOS559 // clone ios555
	case "ios561":
		return instagram.PlatformIOS561 // clone ios560
	case "ios560":
		return instagram.PlatformIOS560 // iPhone + FB iOS app (FBIOS) API 560 (clone 555)
	default:
		return instagram.PlatformWeb
	}
}

// verifyPlatformKeyList trả về danh sách "key" verify user đã chọn (raw dropdown values
// như "s560", "api android"). Trim + bỏ rỗng + dedup, giữ thứ tự.
// Rỗng → fallback [ApiVerifyPlatform]; vẫn rỗng → ["api android"] (default cũ).
func verifyPlatformKeyList(c InteractionConfig) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(c.ApiVerifyPlatforms)+1)
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	for _, p := range c.ApiVerifyPlatforms {
		add(p)
	}
	if len(out) == 0 {
		add(c.ApiVerifyPlatform)
	}
	if len(out) == 0 {
		out = append(out, "api android")
	}
	return out
}

// startup được Wails gọi ngay sau khi cửa sổ app khởi động.
// ctx: Wails context — dùng để emit events lên frontend.
// Thực hiện: setup logging, load AppSettings, scan thư mục nguồn (nếu folder mode), khởi động folder watcher.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// ═══ GC tuning cho chạy dài hạn (tuần/tháng) — aggressive để RAM không drift ═══
	// SetGCPercent(50) → GC trigger khi heap tăng 50% (thay default 100%) → ngăn drift.
	// GOMEMLIMIT auto-detect: 25% system RAM, cap 4GB, sàn 1GB.
	// User override qua env GOMEMLIMIT_MB=2048 (Wails set env trong launcher) hoặc GOMEMLIMIT (Go runtime).
	debug.SetGCPercent(50)

	memLimitMB := computeMemoryLimitMB()
	if memLimitMB > 0 {
		debug.SetMemoryLimit(int64(memLimitMB) << 20)
	}
	slog.Info("GC memory limit",
		"limit_mb", memLimitMB,
		"system_mb", getSystemMemoryMB(),
		"gc_percent", 50)

	// Periodic FreeOSMemory: Go GC giải phóng heap nhưng không trả về OS ngay.
	// Mỗi 5 phút force GC + trả hết freed pages về OS → RAM trong Task Manager giảm thực.
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for range t.C {
			goruntime.GC()
			debug.FreeOSMemory()
		}
	}()

	// Setup structured logging — ghi vào file %APPDATA%/HVR/logs/
	setupLogging()

	// Load AppSettings (auto-migrates từ general.json/interaction.json nếu chưa có app_settings.json)
	app, _, err := appsettings.Load("Config/Settings")
	if err != nil {
		app = appsettings.Default()
	}
	a.settingsMu.Lock()
	a.appSettings = app
	a.settingsMu.Unlock()

	// Ensure Config/Cookie/ tồn tại và warm datr pool cache ngay từ startup
	// (port từ C#: LoadMachineIdsFromFile gọi khi app start).
	if err := cookie.EnsureDir(); err != nil {
		slog.Warn("create cookie dir", "err", err)
	}
	// Seed embedded data lần đầu chạy — ship sẵn pool datr trong exe,
	// user không cần paste thủ công. Chỉ tạo nếu file chưa tồn tại.
	cookie.SeedInitialIfMissing(cookie.DefaultInitialPath())

	// Đăng ký file cache domain cho mail-temp.com (TTL 48h).
	// Khi user chọn Mode=MailTemp, CreateEmail sẽ load domain list từ file này
	// thay vì fetch lại web mỗi lần (tránh chậm + block).
	emailtemp.SetMailTempComDomainCachePath(filepath.Join(AppDataDir(), "Config", "mailtempcom_domains.txt"))

	// Ensure Config/DeviceInfo/ + 2 file split (_reg.txt + _ver.txt) auto-create.
	if err := fbdata.EnsureDir(); err != nil {
		slog.Warn("create fbapp dir", "err", err)
	}
	if err := fbdata.EnsureSplitFiles(); err != nil {
		slog.Warn("create split files", "err", err)
	}
	fbdata.Reload("")
	slog.Info("fb versions active", "count", fbdata.Size(), "reg", fbdata.SizeReg(), "ver", fbdata.SizeVer(), "override", fbdata.OverrideActive())

	// Ensure Config/UserAgent/ và load pool UA từ file.
	if err := fakeinfo.EnsureUAOverrideDir(); err != nil {
		slog.Warn("create ua dir", "err", err)
	}
	fakeinfo.ReloadUAPools()
	slog.Info("ua pools loaded",
		"android", fakeinfo.UAPoolSize(fakeinfo.UAKindAndroid),
		"ios", fakeinfo.UAPoolSize(fakeinfo.UAKindIOS),
		"request", fakeinfo.UAPoolSize(fakeinfo.UAKindRequest))

	// Tạo cấu trúc thư mục Config/ (không seed content — user tự quản lý data).
	if err := fakeinfo.SeedConfigDataIfMissing("Config"); err != nil {
		slog.Warn("seed config dirs", "err", err)
	}

	// Load phone database từ Config/phone_database/.
	fakeinfo.LoadPhoneDatabase(filepath.Join(AppDataDir(), "Config", "phone_database"))
	slog.Info("phone database loaded", "count", len(fakeinfo.PhoneCountries()))

	// Load user overrides từ Config/ (Namereg, Locales, SimNetwork).
	fakeinfo.ReloadOverrides()

	// ── Runtime optimization cho 24/7 operation ─────────────────────────────
	// Periodic GC + memory watchdog → app chạy mãi không bị leak RAM từ Go heap.
	// Phase 1 of stability optimization plan.
	go a.runMemoryMaintenance(ctx)

	// Debug instrumentation — chỉ kích hoạt khi env DEBUG_PPROF=1 hoặc DEBUG_SNAPSHOT=1.
	// Default tắt → production builds không có overhead. Xem debug.go.
	a.startDebugInstrumentation(ctx)

	// Periodic UI cleanup mỗi 12 giờ — emit event để frontend SOFT cleanup
	// (clear log buffer + cache cũ), KHÔNG reload Vue.
	//
	// Trước: 3h reload → user mất state đang nhập/xem (rối thông tin).
	// Giờ: 12h cleanup nhẹ → giữ nguyên UI, chỉ dọn buffer.
	// User có thể bấm nút "Reload UI" thủ công ở status bar nếu thực sự cần reset.
	go func() {
		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if a.ctx != nil {
					runtime.EventsEmit(a.ctx, "system:ui-reload", nil)
				}
			}
		}
	}()

	// Auto-reload accounts từ file mode nếu user đã pick file ở session trước.
	// Tránh trải nghiệm: pick file 500 acc → đóng app → mở lại → settings còn `accountSource=file`
	// nhưng a.accounts rỗng → bấm Run lỗi "Chưa load accounts nào".
	go func() {
		settings := a.LoadSettings()
		if !strings.EqualFold(settings.General.AccountSource, "file") {
			return
		}
		path := strings.TrimSpace(settings.General.AccountSourcePath)
		if path == "" {
			return
		}
		if _, err := os.Stat(path); err != nil {
			slog.Warn("startup: file mode path không tồn tại — bỏ qua auto-reload", "path", path, "err", err)
			return
		}
		result := a.LoadAccountsFromFile(path)
		slog.Info("startup: auto-reload file mode accounts",
			"path", path, "imported", result.Imported, "errors", len(result.Errors))
		// Emit event để frontend grid refresh ngay khi đã load xong.
		runtime.EventsEmit(a.ctx, "accounts:folder-updated", map[string]interface{}{
			"source":   "file",
			"imported": result.Imported,
		})
	}()
}

// runMemoryMaintenance chạy background loop gọi GC + FreeOSMemory định kỳ,
// và monitor heap để cảnh báo khi memory cao (watchdog).
//   - GC timer: 5 phút → release heap fragments, trả OS memory về hệ thống.
//   - Watchdog: 1 phút → nếu heap > 500 MB thì log warning + force GC ngay lập tức.
//   - Auto-cleanup: khi RSS > 1500 MB → tự gọi ForceMemoryCleanup-equiv (close idle
//     conn + GC + FreeOSMemory). Cooldown 3 phút để tránh thrashing.
//
// Port từ phương án Phase 1 stability: cho phép app chạy 24/7 với 100+ luồng
// mà RAM không tăng vô tội vạ.

// removeAccountLine xóa 1 dòng khỏi file (dùng chung cho cả CloneHV và file mode)
func (a *App) GetAppVersion() string {
	return a.version
}

// SetVersion injects the ldflags-set AppVersion from package main.
// Call immediately after NewApp(), before wails.Run().
func (a *App) SetVersion(v string) {
	a.version = v
	buildVersion = v
}

// OnSecondInstance restores the window when a second instance is launched.
// Wire into options.SingleInstanceLock.OnSecondInstanceLaunch.
func (a *App) OnSecondInstance() {
	if a.ctx == nil {
		return
	}
	runtime.WindowUnminimise(a.ctx)
	runtime.Show(a.ctx)
	runtime.WindowSetAlwaysOnTop(a.ctx, true)
	time.Sleep(80 * time.Millisecond)
	runtime.WindowSetAlwaysOnTop(a.ctx, false)
}

type VerifyRunConfig struct {
	AccountIDs []int                  `json:"accountIds"`
	MaxThreads int                    `json:"maxThreads"`
	VerifyCfg  instagram.VerifyConfig `json:"verifyConfig"`
	OutputPath string                 `json:"outputPath"`
	Proxy      string                 `json:"proxy"` // Proxy chung từ settings
}

// RunVerify bắt đầu verify hàng loạt accounts.
// cfgOverride: config từ frontend (AccountIDs cần chạy, MaxThreads, VerifyConfig, OutputPath).
// Hỗ trợ 2 mode:
//   - CloneHV pool mode: tự động mua accounts từ CloneHV, duy trì maxThreads luồng liên tục.
//   - File mode: chạy danh sách accounts đã chọn từ grid, mỗi account 1 goroutine.
//
// Emit các events lên frontend:
//   - "verify:status": log realtime từng account (batch 500ms)
//   - "verify:account-done": kết quả ngay khi 1 account xong
//   - "verify:complete": khi toàn bộ batch xong hoặc bị dừng
//   - "verify:results": tổng kết toàn bộ batch

// saveVerifyOutcome ghi account đã verify vào các file chi tiết + cập nhật counters.
// Port C# FMain.cs flow sau VerifyAccount().
//
// Phân loại file theo status:
//
//	live    → SuccessVerify.txt | SuccessVerify_No2FA.txt (nếu không có 2FA)
//	         → SuccessVerifyUG_<instance>.txt (UA đã verify ok)
//	die     → DieAfterVerify.txt (UPSERT theo UID)
//	         → VerifyFailed_<subStatus>.txt (nếu detect được sub-status từ message)
//	unknown → UnknownErrorCheckLiveDieApi.txt
//	         → các detail file: ChinaMail_CantGetCode.txt, LoginFbFailed_<code>.txt, v.v.
//
// Counter increments:
//
//	live → Country++, FbAppVersion++, Locale++ (port C# FbAppVersisonSuccess/CountrySuccess/FbLocalesSuccess)
//
// writer: gắn với folder run-specific (VerifyMfb_20260418_103015). nil → no-op.
// counters: auto-save ticker. nil → bỏ qua counter tracking.
// status: "live" | "die" | "unknown".
// message: Message field từ verify result — dùng để detect sub-status.
// acc: Account DTO (UID, Cookie, Token, UserAgent, FBVersion...).
// verifyInstance: VerifyInstanceName ("ApiAndroid", "ApiMfb"...) — đặt tên UA file.
//
// Lỗi I/O log qua slog, không fail flow.
func saveVerifyOutcome(
	writer *resultpkg.Writer,
	counters *resultpkg.CounterSet,
	status string,
	message string,
	acc Account,
	verifyInstance string,
) {
	if writer == nil || writer.Root() == "" {
		return
	}
	// Country: Account DTO không có field Country riêng — dùng Location (cho counter tracking).
	// Nếu Location rỗng, thử extract FBLC từ UA và lấy 2 ký tự cuối (en_US → US).
	// Fallback 2 (cho MFB/WebAndroid): extract từ cookie "locale=xx_YY;" — UA Chrome
	// không có FBLC tag nên phải đọc locale từ cookie để vẫn có country ở cuối line.
	country := strings.TrimSpace(acc.Location)
	if country == "" {
		if loc := extractFBLocaleFromUA(acc.UserAgent); len(loc) >= 5 && loc[2] == '_' {
			country = loc[3:]
		}
	}
	if country == "" {
		if loc := extractLocaleFromCookieStr(acc.Cookie); len(loc) >= 5 && loc[2] == '_' {
			country = loc[3:]
		}
	}
	line := resultpkg.FormatVerify(resultpkg.VerifyData{
		UID:      acc.UID,
		Password: acc.Password,
		TwoFA:    acc.Twofa,
		Cookie:   acc.Cookie,
		Token:    acc.Token,
		Email:    "", // user yêu cầu: không lưu email vào file tài khoản thành công
		FullName: acc.FullName,
		Country:  country,
	}, nil)

	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "live":
		filename := resultpkg.FileSuccessVerify
		// Khi acc KHÔNG có 2FA → format reg (6 field: uid|pass|cookie|token|time|country)
		// thay vì format verify (9 field) — giống layout SuccessReg.txt để user quen mắt.
		outLine := line
		if strings.TrimSpace(acc.Twofa) == "" {
			filename = resultpkg.FileSuccessVerifyNo2FA
			outLine = resultpkg.FormatReg(resultpkg.RegData{
				UID: acc.UID, Password: acc.Password,
				Cookie: acc.Cookie, Token: acc.Token,
				Email: "", Country: country, // user yêu cầu: không lưu email
			}, nil)
		}
		if err := writer.Append(filename, outLine); err != nil {
			slog.Warn("saveVerifyOutcome live", "err", err)
		}
		if verifyInstance != "" && acc.UserAgent != "" {
			_ = writer.Append(resultpkg.SuccessVerifyUGFile(verifyInstance), acc.UserAgent)
		}
		// Counter tracking: fb version
		if counters != nil {
			if fbVer := extractFBVersionFromUA(acc.UserAgent); fbVer != "" {
				counters.FbAppVersion.Incr(fbVer)
			}
		}
	case "die":
		if err := writer.UpsertUID(resultpkg.FileDieAfterVerify, line); err != nil {
			slog.Warn("saveVerifyOutcome die", "err", err)
		}
	default:
		if err := writer.Append(resultpkg.FileUnknownErrorCheckLiveDieApi, line); err != nil {
			slog.Warn("saveVerifyOutcome unknown", "err", err)
		}
	}

	// Dispatch detail files theo sub-status detect từ message
	for _, d := range resultpkg.DispatchVerifyDetails(s, message, line) {
		if d.Upsert {
			_ = writer.UpsertUID(d.File, d.Content)
		} else {
			_ = writer.Append(d.File, d.Content)
		}
	}
}

// extractFBVersionFromUA rút "FBAV|FBBV" từ User-Agent string để làm key counter.
// UA chuẩn FB4A: "[FBAN/FB4A;FBAV/554.0.0.57.70;FBBV/918990560;..."
// Trả về "" nếu UA không có format FB4A (vd UA iOS hoặc UA rỗng).
func extractFBVersionFromUA(ua string) string {
	if ua == "" {
		return ""
	}
	ver := extractBetween(ua, "FBAV/", ";")
	build := extractBetween(ua, "FBBV/", ";")
	if ver == "" || build == "" {
		return ""
	}
	return ver + "|" + build
}

// extractFBLocaleFromUA rút FBLC từ UA — key cho counter locale.
func extractFBLocaleFromUA(ua string) string {
	if ua == "" {
		return ""
	}
	return extractBetween(ua, "FBLC/", ";")
}

// extractLocaleFromCookieStr extract locale từ cookie string (vd "locale=es_CL;...").
// Dùng cho MFB / WebAndroid verify — UA Chrome không có FBLC tag, locale chỉ có trong cookie.
// Trả "" nếu cookie không có field locale.
func extractLocaleFromCookieStr(cookie string) string {
	if cookie == "" {
		return ""
	}
	for _, part := range strings.Split(cookie, ";") {
		part = strings.TrimSpace(part)
		if kv := strings.SplitN(part, "=", 2); len(kv) == 2 && strings.TrimSpace(kv[0]) == "locale" {
			return strings.TrimSpace(kv[1])
		}
	}
	return ""
}

// extractBetween tìm chuỗi giữa start và end trong s. Trả "" nếu không match.
func extractBetween(s, start, end string) string {
	i := strings.Index(s, start)
	if i < 0 {
		return ""
	}
	s = s[i+len(start):]
	j := strings.Index(s, end)
	if j < 0 {
		return ""
	}
	return s[:j]
}

// saveRegOutcome ghi account đã register vào các file kết quả chi tiết.
// Port C# FMain.cs flow sau RegisterAccount():
//
//	live/success  → SuccessReg.txt (kèm "|NVR") + SuccessRegNVR_UG_<instance>.txt
//	                + SuccessNVR_Phone.txt hoặc SuccessNVR_Email.txt theo login type
//	checkpoint    → Checkpoint.txt
//	blocked       → Blocked.txt
//	unknown       → UnknownBlockType.txt
//
// writer: folder run-specific (RegAndroid_20260418_103015).
// acc: thông tin account vừa reg (UID, Cookie, Token...).
// regInstance: RegisterInstanceName ("ApiAndroid", "ApiS23"...) — để đặt UA file.
// login: chuỗi login — nếu chứa "@" thì ghi vào Email file, ngược lại vào Phone file.
func saveRegOutcome(
	writer *resultpkg.Writer,
	counters *resultpkg.CounterSet,
	status string,
	message string,
	acc Account,
	regInstance, login string,
	onSuccess func(line string),
) string {
	if writer == nil || writer.Root() == "" {
		return ""
	}
	// Country: từ Location hoặc FBLC trong UA (giống verify flow).
	country := strings.TrimSpace(acc.Location)
	if country == "" {
		if loc := extractFBLocaleFromUA(acc.UserAgent); len(loc) >= 5 && loc[2] == '_' {
			country = loc[3:]
		}
	}
	line := resultpkg.FormatReg(resultpkg.RegData{
		UID:       acc.UID,
		Username:  acc.Username, // field đầu = @handle IG (UID số vẫn còn trong cookie)
		Password:  acc.Password,
		Cookie:    acc.Cookie,
		Token:     acc.Token,
		Email:     "", // user yêu cầu: không lưu email vào file tài khoản thành công
		Country:   country,
		IsNVR:     true,          // C#: register xong = NVR (not-verified-yet)
		EmailMeta: acc.EmailMeta, // TempMail mode → persist creds vào file (cần cho split-mode verify)
	}, nil)

	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "live", "success":
		if err := writer.Append(resultpkg.FileSuccessReg, line); err != nil {
			slog.Warn("saveRegOutcome success", "err", err)
		}
		if onSuccess != nil {
			go onSuccess(line)
		}
		if regInstance != "" && acc.UserAgent != "" {
			_ = writer.Append(resultpkg.SuccessRegNVRUGFile(regInstance), acc.UserAgent)
		}
		if strings.Contains(login, "@") {
			_ = writer.Append(resultpkg.FileSuccessNVREmail, login)
		} else if login != "" {
			_ = writer.Append(resultpkg.FileSuccessNVRPhone, login)
		}
		// Counter tracking: fb version
		if counters != nil {
			if fbVer := extractFBVersionFromUA(acc.UserAgent); fbVer != "" {
				counters.FbAppVersion.Incr(fbVer)
			}
		}
	case "checkpoint":
		_ = writer.Append(resultpkg.FileCheckpoint, line)
	case "blocked", "block":
		_ = writer.Append(resultpkg.FileBlocked, line)
	default:
		_ = writer.Append(resultpkg.FileUnknownBlockType, line)
	}

	// Dispatch detail files theo sub-status từ message
	for _, d := range resultpkg.DispatchRegDetails(s, message, line) {
		if d.Upsert {
			_ = writer.UpsertUID(d.File, d.Content)
		} else {
			_ = writer.Append(d.File, d.Content)
		}
	}
	return line
}

// CloneHVStockResult kết quả kiểm tra tồn kho
type CloneHVStockResult struct {
	Name   string `json:"name"`
	Amount string `json:"amount"`
	Price  int    `json:"price"`
	Error  string `json:"error"`
}

// CheckCloneHVStock kiểm tra thông tin sản phẩm CloneHV (tồn kho, giá)
func (a *App) CheckCloneHVStock(username, password, productID string) CloneHVStockResult {
	info, err := clonehv.GetProductInfo(a.ctx, username, password, productID)
	if err != nil {
		return CloneHVStockResult{Error: err.Error()}
	}
	return CloneHVStockResult{
		Name:   info.Name,
		Amount: info.Amount,
		Price:  info.Price,
	}
}

// StopVerify dừng verify đang chạy theo kiểu DRAIN:
//   - Cancel dispatch context → vòng lặp đẩy account mới dừng ngay; account đã queue
//     trong workCh sẽ skip và đánh dấu "Đã dừng" (không chạy verify).
//   - KHÔNG cancel workerCtx → các account ĐANG chạy dở (đã vào runOneAccount)
//     được hoàn thành HTTP requests hiện tại, ghi đúng kết quả Live/Die.
//
// workerCtx vẫn sẽ tự cancel khi:
//   - goroutine spawner kết thúc (deferred workerCancel ở cuối goroutine)
//   - app shutdown (a.ctx — parent context — bị cancel bởi Wails)
//
// → Force stop (kill HTTP đang chạy) chỉ xảy ra khi user đóng app, không phải khi
// click Stop. Match với tinh thần "phải chạy hết rồi mới dừng".
func (a *App) StopVerify() string {
	a.verifyMu.Lock()
	defer a.verifyMu.Unlock()

	if !a.isRunning {
		// Defensive: clear stopping nếu lỡ kẹt
		a.verifyStopping.Store(false)
		return "Không có verify nào đang chạy"
	}
	if a.verifyStopping.Load() {
		return "Đang dừng — vui lòng chờ workers hoàn tất..."
	}
	a.verifyStopping.Store(true)
	if a.verifyCancel != nil {
		a.verifyCancel() // dừng dispatch accounts mới — workers đang chạy KHÔNG bị abort
	}
	return "Đang dừng — chờ các account đang chạy hoàn tất..."
}

// IsVerifyRunning kiểm tra xem có verify đang chạy không.
// Frontend polling mỗi vài giây để cập nhật trạng thái nút Run/Stop.
func (a *App) IsVerifyRunning() bool {
	a.verifyMu.Lock()
	defer a.verifyMu.Unlock()
	return a.isRunning
}

// IsVerifyStopping — true khi user đã bấm Stop nhưng workers chưa exit xong.
// Frontend dùng để disable nút Start tránh overlap run.
func (a *App) IsVerifyStopping() bool {
	return a.verifyStopping.Load()
}

// NotifyVisibilityChange — frontend gọi khi visibility window thay đổi (minimize/restore).
// Backend dùng để throttle batch emitter (300ms → 2s) khi hidden, tiết kiệm IPC.
// Cũng giúp giảm cost JSON serialize + Wails event dispatch khi user không xem.
func (a *App) NotifyVisibilityChange(hidden bool) {
	a.frontendHidden.Store(hidden)
}

// RequestQuit — frontend gọi sau khi user đã confirm muốn thoát.
// Set flag confirmedQuit → OnBeforeClose tiếp theo sẽ allow close → runtime.Quit() đóng app.
// Tách thành 2 bước (set flag + Quit) để Wails close pipeline chạy đầy đủ:
// goroutine emit Quit → main loop close → OnBeforeClose check flag → cleanup → exit.
func (a *App) RequestQuit() {
	a.confirmedQuit.Store(true)
	runtime.Quit(a.ctx)
}

// IsConfirmedQuit — main.go OnBeforeClose dùng để quyết định block hay allow close.
// false (default): block close + emit "app:request-quit-confirm" → FE show dialog.
// true: allow close (user đã confirm qua RequestQuit).
func (a *App) IsConfirmedQuit() bool {
	return a.confirmedQuit.Load()
}

// EmitQuitConfirm — main.go gọi từ OnBeforeClose khi cần FE show dialog xác nhận.
// Tách thành method riêng để main.go không phải import wailsRuntime trực tiếp ở handler.
func (a *App) EmitQuitConfirm() {
	regRunning := runState(a.registerState.Load()) == runStateRunning
	a.verifyMu.Lock()
	verRunning := a.isRunning
	a.verifyMu.Unlock()
	runtime.EventsEmit(a.ctx, "app:request-quit-confirm", map[string]interface{}{
		"registerRunning": regRunning,
		"verifyRunning":   verRunning,
	})
}

// IsRegisterRunning kiểm tra xem có register đang chạy không.
// Frontend dùng để restore split mode UI sau khi UI auto-reload —
// nếu vẫn đang register + splitMode trong config → render lại split layout.
func (a *App) IsRegisterRunning() bool {
	return runState(a.registerState.Load()) == runStateRunning
}

// IsRegisterStopping — true khi user đã bấm Stop nhưng workers chưa exit xong.
func (a *App) IsRegisterStopping() bool {
	return runState(a.registerState.Load()) == runStateStopping
}

// GetRunStatus — bundle status cho frontend gọi 1 lần thay vì 4 IPC riêng.
// Dùng cho restore state sau UI reload + poll định kỳ ở status bar.
func (a *App) GetRunStatus() map[string]bool {
	a.verifyMu.Lock()
	verRunning := a.isRunning
	a.verifyMu.Unlock()
	regState := runState(a.registerState.Load())
	return map[string]bool{
		"registerRunning":  regState == runStateRunning,
		"registerStopping": regState == runStateStopping,
		"verifyRunning":    verRunning,
		"verifyStopping":   a.verifyStopping.Load(),
	}
}

// AppResourceUsage thông số RAM + CPU của app — gọi từ frontend mỗi 2s
type AppResourceUsage struct {
	RAMMb  float64 `json:"ramMb"`  // RAM app đang dùng (MB)
	CPUPct float64 `json:"cpuPct"` // CPU % thực tế của process
}

// cpuTracker lưu trạng thái trước đó để tính CPU delta
var cpuTracker struct {
	mu       sync.Mutex
	lastCPU  time.Duration
	lastTime time.Time
	pct      float64
}

// GetResourceUsage trả về mức sử dụng RAM và CPU của process hiện tại.
// Frontend gọi mỗi 2 giây để hiển thị trên status bar.
// RAMMb: RAM process đang dùng tính bằng MB.
// CPUPct: CPU % thực tế, đã chia cho số CPU cores để khớp Task Manager.
func (a *App) GetResourceUsage() AppResourceUsage {
	return AppResourceUsage{
		RAMMb:  getProcessMemoryMB(),
		CPUPct: getProcessCPUPercent(),
	}
}

// getProcessCPUPercent tính CPU% của process dựa trên delta giữa 2 lần gọi liên tiếp.
// Dùng kernel+user time từ OS (getProcessCPUTime), chia cho elapsed time và số cores.
// Lần gọi đầu tiên luôn trả về 0 (chưa có điểm so sánh).
// Thread-safe: dùng cpuTracker.mu để bảo vệ lastCPU và lastTime.
func getProcessCPUPercent() float64 {
	totalCPU := getProcessCPUTime()
	now := time.Now()

	cpuTracker.mu.Lock()
	defer cpuTracker.mu.Unlock()

	if !cpuTracker.lastTime.IsZero() {
		elapsed := now.Sub(cpuTracker.lastTime)
		if elapsed > 0 {
			cpuDelta := totalCPU - cpuTracker.lastCPU
			// Chia cho số CPU cores để khớp Task Manager
			cores := getNumCPU()
			if cores < 1 {
				cores = 1
			}
			cpuTracker.pct = float64(cpuDelta) / float64(elapsed) * 100 / float64(cores)
			if cpuTracker.pct > 100 {
				cpuTracker.pct = 100
			}
			if cpuTracker.pct < 0 {
				cpuTracker.pct = 0
			}
		}
	}

	cpuTracker.lastCPU = totalCPU
	cpuTracker.lastTime = now
	return cpuTracker.pct
}

// === Register ===

// RegisterInput — input từ frontend để tạo tài khoản Facebook
type RegisterInput struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Birthday  string `json:"birthday"` // "DD-MM-YYYY"
	Gender    int    `json:"gender"`   // 1=female, 2=male
	Phone     string `json:"phone"`
	Password  string `json:"password"`
	Proxy     string `json:"proxy"`
	UserAgent string `json:"userAgent"`
}

// RegisterFacebook — tạo tài khoản Facebook (B1-B8)
// Emit event "register:status" theo từng bước
func (a *App) RegisterFacebook(input RegisterInput) string {
	regInput := &instagram.RegInput{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Birthday:  input.Birthday,
		Gender:    input.Gender,
		Phone:     input.Phone,
		Password:  input.Password,
		Proxy:     input.Proxy,
		UserAgent: input.UserAgent,
	}

	// Chọn platform register: từ InteractionConfig.ApiRegPlatform, fallback web
	regPlatform := instagram.PlatformWeb
	if ic := a.LoadInteractionConfig(); ic.ApiRegPlatform != "" {
		regPlatform = ic.ApiRegPlatform
	}
	reg, regErr := instagram.NewRegisterer(regPlatform)
	if regErr != nil {
		return "ERR|" + regErr.Error()
	}
	result := reg.Register(a.ctx, regInput, func(msg string) {
		runtime.EventsEmit(a.ctx, "register:status", msg)
	})

	if result.Success {
		return fmt.Sprintf("OK|%s|%s", result.UID, result.Message)
	}
	return "ERR|" + result.Message
}

// nextVerifyPlatform — trả về platform verify cho 1 account (resolve từ list multi-version
// theo round-robin). Đọc config mới nhất mỗi lần gọi để hỗ trợ realtime reload.
// Trả về platform string (đã qua verifyPlatformFromType), KHÔNG phải raw key.
func (a *App) nextVerifyPlatform() string {
	keys := verifyPlatformKeyList(a.LoadInteractionConfig())
	if len(keys) == 0 {
		return verifyPlatformFromType("")
	}
	n := a.verifyPlatformRR.Add(1) - 1
	if n < 0 {
		n = 0
	}
	return verifyPlatformFromType(keys[int(n)%len(keys)])
}

// formatRegResultLine — format kết quả đăng ký thành 1 dòng file
// Format: UID|password|cookies|access_token|datetime|country|NVR[|SRN:<srnonce>|SCUID:<sessionlessCryptedUID>]
// iOS accounts: cookie + token rỗng, nhưng SRN:/SCUID: được append để file-based verify dùng được.
func formatRegResultLine(result *instagram.RegResult, profile instagram.RegInput) string {
	now := time.Now().Format("02-01-2006 15:04:05")
	country := phoneCountryCode(profile.Phone)
	line := fmt.Sprintf("%s|%s|%s|%s|%s|%s|NVR",
		result.UID, result.Password, result.Cookie, result.AccessToken, now, country)
	if result.Srnonce != "" {
		line += "|SRN:" + result.Srnonce
	}
	if result.SessionlessCryptedUID != "" {
		line += "|SCUID:" + result.SessionlessCryptedUID
	}
	return line
}

// phoneCountryCode — trả về mã quốc gia 2 ký tự từ số điện thoại quốc tế.
// Nguồn: fakeinfo.FindCountryByPhonePrefix (data từ Config/phone_database/).
// KHÔNG hardcode prefix→country trong code.
// Trả "UN" nếu phone rỗng hoặc không match prefix nào.
func phoneCountryCode(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return "UN"
	}
	if p, ok := fakeinfo.FindCountryByPhonePrefix(phone); ok {
		return p.CountryCode
	}
	return "UN"
}

// appendLineToFile — append 1 dòng vào file (tạo file nếu chưa có)
func appendLineToFile(path, line string) error {
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line + "\n")
	return err
}

// StopRegister — dừng đăng ký tài khoản.
//
// Transition: running → stopping. Gọi runCancel để spawner exit loop, nhưng
// KHÔNG nil registerCancel ở đây — spawner defer sẽ tự nil sau wg.Wait() +
// cleanup pool. State chuyển stopping → idle CHỈ trong defer, sau khi mọi
// worker đã exit và native resource đã release.
//
// Frontend nhận event "register:complete" mới biết state đã idle để cho phép Start.
func (a *App) StopRegister() string {
	a.registerMu.Lock()
	defer a.registerMu.Unlock()

	switch runState(a.registerState.Load()) {
	case runStateIdle:
		// Defensive emit — frontend lỡ kẹt UI "Đang chạy" sẽ nhận event này để reset.
		runtime.EventsEmit(a.ctx, "register:complete", map[string]interface{}{"total": 0})
		return "Không có tiến trình đăng ký nào"
	case runStateStopping:
		return "Đang dừng — vui lòng chờ workers hoàn tất..."
	}

	// state == running → transition sang stopping. KHÔNG nil registerCancel:
	// spawner defer cần check `a.registerGen == myGen` rồi mới nil. Bump registerGen
	// chỉ làm gen != myGen → defer KHÔNG nil ref → cancel function leak (vì run mới
	// không thể start trong stopping nên không có cancel mới). Vì vậy cũng KHÔNG bump gen.
	a.registerState.Store(int32(runStateStopping))
	if a.registerCancel != nil {
		a.registerCancel() // báo spawner: thoát loop để defer chạy cleanup
	}
	return "Đang dừng đăng ký..."
}

// === Utility ===

// extractDatrFromCookieLine — parse datr từ 1 dòng cookie list
// Format dòng: UID|password|cookie_string|... hoặc cookie_string thẳng
// Tìm datr=VALUE trong cookie_string (kết thúc bởi ; hoặc end-of-string)
func extractDatrFromCookieLine(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	// Lấy phần cookie string: nếu có | thì field thứ 3 (index 2), không thì dùng cả dòng
	cookieStr := line
	parts := strings.SplitN(line, "|", 4)
	if len(parts) >= 3 {
		cookieStr = parts[2]
	}
	// Tìm datr=VALUE
	for _, seg := range strings.Split(cookieStr, ";") {
		seg = strings.TrimSpace(seg)
		if strings.HasPrefix(seg, "datr=") {
			return strings.TrimPrefix(seg, "datr=")
		}
	}
	return ""
}

// splitLines tách chuỗi s thành slice dòng, chuẩn hóa cả \r\n và \r về \n trước.
// s: chuỗi văn bản (có thể từ Windows CRLF hoặc Unix LF).
// Dùng để parse account list, proxy list, UA list từ textarea hoặc file.
// defaultResultFolder trả về thư mục lưu kết quả mặc định khi user chưa thiết lập.
// Ưu tiên: thư mục chứa exe → Documents/HVR/KetQua
// defaultResultFolder trả về thư mục kết quả mặc định, luôn tên "result" (đúng như C#).
//
// Port C# FMain.cs: SavePath = Application.StartupPath + "\\result\\result_" + datetimnow.
// Go giữ tên folder gốc "result" để user migrate từ C# không phải đổi thói quen.
//
// Fallback 2 tầng:
//  1. {exe_dir}/result/                         — như C#, ưu tiên cạnh exe
//  2. ~/Documents/HVR/result/         — nếu exe_dir read-only (Program Files)
//
// Luôn auto-MkdirAll — caller không cần tạo trước.
func defaultResultFolder() string {
	candidate := filepath.Join(AppDataDir(), "result")
	if mkErr := os.MkdirAll(candidate, 0755); mkErr == nil {
		return candidate
	}
	// Fallback: Documents/HVR/result
	homeDir, _ := os.UserHomeDir()
	fallback := filepath.Join(homeDir, "Documents", "HVR", "result")
	_ = os.MkdirAll(fallback, 0755)
	return fallback
}

// resolveUAPool trả về 1 UA ngẫu nhiên từ Config/UserAgent/{type}_UG.txt theo poolKey.
// Dùng nội bộ bởi pickUAForVerifyPlatform để lấy UA raw từ pool file.
func resolveUAPool(poolKey string) string {
	return fakeinfo.RandomUAFromPool(uaKindFromPoolKey(poolKey))
}

// pickUAForVerifyPlatform — pick UA chuẩn theo verify API platform.
//
// Mỗi verify API cần UA format riêng để match fingerprint server-side:
//   - S23/S555/S556/S557/S558/Android (api token) → Samsung Galaxy Android UA (FBAN/FB4A;FBDV/SM-S911B...)
//   - WebAndroid (api web andr) → Chrome Mobile UA (Mozilla/...; Android; ...; Chrome/...; Mobile)
//   - Web (api mfb) → Chrome Desktop UA (Mozilla/...; Windows; ...; Chrome/...)
//
// Validate accountUA theo platform để TRÁNH MISMATCH:
//   - Web cần Chrome Desktop. accountUA = FBAN/FB4A → reject, fallback pool/default.
//   - WebAndroid cần Chrome Mobile. accountUA = FBAN/FB4A → reject, fallback pool/default.
//   - Android (S23/...) cần FBAN/FB4A. accountUA = Chrome → reject, fallback pool/default.
//   - S555/S556/S557/S558 cần FBAV khớp đúng version (vì doc_id của API
//     khớp đúng phiên bản app — pool 554 cũ sẽ bị FB flag mismatch).
//
// Priority (sau khi validate):
//  1. accountUA hợp lệ với platform (đối với S55x/S558 phải khớp FBAV/<ver>).
//  2. Factory UA của platform — instagram.PlatformVerifyUA(verifyPlatform).
//     Áp dụng cho S555/S556/S557/S558 — sinh UA random device + carrier
//     với FBAV/FBBV cố định đúng phiên bản API.
//  3. Pool textarea theo platform (cài đặt chung, tab tương ứng).
//  4. Embed pool theo kind (Android_UG.txt / iOS_UG.txt).
//  5. Trả "" → verify worker tự fallback default (Chrome Pixel 8 cho WebAndroid).
func pickUAForVerifyPlatform(verifyPlatform, accountUA string, cfg InteractionConfig, countryCode string) string {
	maybeXID := func(ua string) string {
		if cfg.TrackingIDVer && ua != "" {
			return appendXIDToUA(ua)
		}
		return ua
	}
	// iOS Messenger Lite (iosmess): LUÔN dùng UA MessengerLite, KHÔNG fall through Android pool.
	// Tôn trọng 3 chế độ như platform khác: UA Gốc → pool (ios_mess) → Build (random per-account).
	if verifyPlatform == instagram.PlatformIOSMess {
		// 1. UA Gốc cố định (capture) khi tick UseOriginalUA.
		if cfg.UseOriginalUA {
			origCC := countryCode
			if !cfg.ReplaceCarrier {
				origCC = ""
			}
			if origUA := originalUAForPlatform(verifyPlatform, origCC); origUA != "" {
				return maybeXID(origUA)
			}
		}
		// 2. Pool mode (tắt Build UA): lấy từ pool ios_mess nếu file có UA.
		if !cfg.BuildUA {
			if ua := fakeinfo.RandomUAFromPool(fakeinfo.UAKindIOSMess); ua != "" {
				return maybeXID(ua)
			}
		}
		// 3. Build UA / fallback: random MessengerLite per-account (đa dạng device/version).
		if ua := instagram.PlatformVerifyUA(instagram.PlatformIOSMess, countryCode); ua != "" {
			return maybeXID(ua)
		}
	}
	// UseOriginalUA: ưu tiên tuyệt đối — dùng UA gốc cố định nếu platform hỗ trợ.
	// countryCode dùng để thay FBCR theo nhà mạng của proxy/IP.
	if cfg.UseOriginalUA {
		origCC := countryCode
		if !cfg.ReplaceCarrier {
			origCC = "" // giữ nguyên FBCR/Viettel gốc
		}
		if origUA := originalUAForPlatform(verifyPlatform, origCC); origUA != "" {
			return maybeXID(origUA)
		}
	}
	// Messenger Android (appmessv3): pool mode (tắt Build UA) → đọc Android_Mess.txt nếu file có UA Orca.
	// Nếu không sẽ rơi xuống step 2 = factory orcaUA random (Build UA / pool rỗng).
	if strings.HasPrefix(verifyPlatform, "appmessv3") && !cfg.BuildUA {
		if ua := fakeinfo.RandomUAFromPool(fakeinfo.UAKindAndroidMess); ua != "" {
			return maybeXID(ua)
		}
	}

	accountUA = strings.TrimSpace(accountUA)

	// Map verify platform → pool key + UA validator
	poolKey := "android"
	isValidUA := func(ua string) bool { return strings.Contains(ua, "FBAN/FB4A") || strings.Contains(ua, "FBAV/") }

	// expectedFBAVPrefix — non-empty với S55x/S558 để ép validate đúng phiên bản.
	// S23/Android không có ràng buộc version (pool nội bộ thường 554).
	expectedFBAVPrefix := ""

	switch verifyPlatform {
	case instagram.PlatformWeb:
		// MFB gọi m.facebook.com (mobile site) → BẮT BUỘC dùng Chrome Mobile UA.
		// Chrome Desktop UA truy cập m.facebook.com → UA mismatch → FB detect bot.
		// FIXED 2026-05-19: trước đây dùng PC pool (Chrome Desktop) sai.
		poolKey = "webchrome" // WebMobile pool = Chrome Mobile
		isValidUA = func(ua string) bool {
			return strings.Contains(ua, "Chrome") && strings.Contains(ua, "Mobile")
		}
	case instagram.PlatformWebAndroid:
		// WebAndroid (api web andr) → WebMobile pool (Chrome Mobile Android UAs).
		// Cùng endpoint m.facebook.com như MFB nên dùng chung pool.
		poolKey = "webchrome"
		isValidUA = func(ua string) bool {
			return strings.Contains(ua, "Chrome") && strings.Contains(ua, "Mobile")
		}
	case instagram.PlatformS415:
		expectedFBAVPrefix = "FBAV/415."
	case instagram.PlatformS425:
		expectedFBAVPrefix = "FBAV/425."
	case instagram.PlatformS435:
		expectedFBAVPrefix = "FBAV/435."
	case instagram.PlatformS445:
		expectedFBAVPrefix = "FBAV/445."
	case instagram.PlatformS455:
		expectedFBAVPrefix = "FBAV/455."
	case instagram.PlatformS555:
		expectedFBAVPrefix = "FBAV/555."
	case instagram.PlatformS555V2:
		expectedFBAVPrefix = "FBAV/555."
	case instagram.PlatformS556:
		expectedFBAVPrefix = "FBAV/556."
	case instagram.PlatformS557:
		expectedFBAVPrefix = "FBAV/557."
	case instagram.PlatformS558:
		expectedFBAVPrefix = "FBAV/558."
	case instagram.PlatformS558V2:
		expectedFBAVPrefix = "FBAV/558."
	case instagram.PlatformS556V2:
		expectedFBAVPrefix = "FBAV/556."
	case instagram.PlatformS557V2:
		expectedFBAVPrefix = "FBAV/557."
	case instagram.PlatformS553V2:
		expectedFBAVPrefix = "FBAV/553."
	case instagram.PlatformS554V2:
		expectedFBAVPrefix = "FBAV/554."
	case instagram.PlatformS551V2:
		expectedFBAVPrefix = "FBAV/551."
	case instagram.PlatformS552V2:
		expectedFBAVPrefix = "FBAV/552."
	case instagram.PlatformS550V2:
		expectedFBAVPrefix = "FBAV/550."
	case instagram.PlatformS559:
		expectedFBAVPrefix = "FBAV/559."
	case instagram.PlatformS559V2:
		expectedFBAVPrefix = "FBAV/559."
	case instagram.PlatformS560, instagram.PlatformS560V2, instagram.PlatformS560V3:
		expectedFBAVPrefix = "FBAV/560."
	case instagram.PlatformS561:
		expectedFBAVPrefix = "FBAV/561."
	case instagram.PlatformS561V2:
		expectedFBAVPrefix = "FBAV/561."
	case instagram.PlatformS562:
		expectedFBAVPrefix = "FBAV/562."
	case instagram.PlatformS562V3:
		expectedFBAVPrefix = "FBAV/562."
	case instagram.PlatformS563:
		expectedFBAVPrefix = "FBAV/563."
	case instagram.PlatformS563V2:
		expectedFBAVPrefix = "FBAV/563."
	case instagram.PlatformS563S21:
		expectedFBAVPrefix = "FBAV/563."
	case instagram.PlatformS563V3S21:
		expectedFBAVPrefix = "FBAV/563."
	case instagram.PlatformS563V4S21:
		expectedFBAVPrefix = "FBAV/563."
	case instagram.PlatformS563V4S23:
		expectedFBAVPrefix = "FBAV/563."
	case instagram.PlatformS563V5S21:
		expectedFBAVPrefix = "FBAV/563."
	case instagram.PlatformS563V5S23:
		expectedFBAVPrefix = "FBAV/563."
	case instagram.PlatformS563V6S21:
		expectedFBAVPrefix = "FBAV/563."
	case instagram.PlatformS563V6S23:
		expectedFBAVPrefix = "FBAV/563."
	case instagram.PlatformS564V1S21:
		expectedFBAVPrefix = "FBAV/564."
	case instagram.PlatformS564V1S23:
		expectedFBAVPrefix = "FBAV/564."
	case instagram.PlatformS564V2S21:
		expectedFBAVPrefix = "FBAV/564."
	case instagram.PlatformS564V2S23:
		expectedFBAVPrefix = "FBAV/564."
	case instagram.PlatformS564V3S21:
		expectedFBAVPrefix = "FBAV/564."
	case instagram.PlatformS564V3S23:
		expectedFBAVPrefix = "FBAV/564."
	case instagram.PlatformS565S21, instagram.PlatformS565S23:
		expectedFBAVPrefix = "FBAV/565."
	case instagram.PlatformS565V2S21, instagram.PlatformS565V2S23:
		expectedFBAVPrefix = "FBAV/565."
	case instagram.PlatformAppMessV3:
		expectedFBAVPrefix = "FBAV/530." // Messenger (Orca) v530 — UA Orca-Android (capture V4)
	case instagram.PlatformAppMessV3_535:
		expectedFBAVPrefix = "FBAV/535."
	case instagram.PlatformAppMessV3_545:
		expectedFBAVPrefix = "FBAV/545."
	case instagram.PlatformAppMessV3_555:
		expectedFBAVPrefix = "FBAV/555."
	case instagram.PlatformAppMessV3_563:
		expectedFBAVPrefix = "FBAV/563."
	case instagram.PlatformAppMessV3_564:
		expectedFBAVPrefix = "FBAV/564."
	case instagram.PlatformAppMessV3_565:
		expectedFBAVPrefix = "FBAV/565."
	case instagram.PlatformAppMessV3_525:
		expectedFBAVPrefix = "FBAV/525."
	case instagram.PlatformAppMessV3_515:
		expectedFBAVPrefix = "FBAV/515."
	case instagram.PlatformAppMessV3_505:
		expectedFBAVPrefix = "FBAV/505."
	case instagram.PlatformAppMessV3_490:
		expectedFBAVPrefix = "FBAV/490."
	case instagram.PlatformS561V4S21:
		expectedFBAVPrefix = "FBAV/561."
	case instagram.PlatformS561V4S23:
		expectedFBAVPrefix = "FBAV/561."
	case instagram.PlatformS562V4S21:
		expectedFBAVPrefix = "FBAV/562."
	case instagram.PlatformS562V4S23:
		expectedFBAVPrefix = "FBAV/562."
	case instagram.PlatformS399:
		expectedFBAVPrefix = "FBAV/399."
	case instagram.PlatformS273:
		expectedFBAVPrefix = "FBAV/273."
	case instagram.PlatformIOS562, instagram.PlatformIOS555, instagram.PlatformIOS550, instagram.PlatformIOS540, instagram.PlatformIOS530, instagram.PlatformIOS520, instagram.PlatformIOS564,
		instagram.PlatformIOS510, instagram.PlatformIOS500, instagram.PlatformIOS490, instagram.PlatformIOS480, instagram.PlatformIOS470, instagram.PlatformIOS460, instagram.PlatformIOS450,
		instagram.PlatformIOS440, instagram.PlatformIOS430, instagram.PlatformIOS420, instagram.PlatformIOS560,
		instagram.PlatformIOS421, instagram.PlatformIOS422, instagram.PlatformIOS423, instagram.PlatformIOS424, instagram.PlatformIOS425, instagram.PlatformIOS426, instagram.PlatformIOS427, instagram.PlatformIOS428, instagram.PlatformIOS429, instagram.PlatformIOS431, instagram.PlatformIOS432, instagram.PlatformIOS433, instagram.PlatformIOS434, instagram.PlatformIOS435, instagram.PlatformIOS436, instagram.PlatformIOS437, instagram.PlatformIOS438, instagram.PlatformIOS439, instagram.PlatformIOS441, instagram.PlatformIOS442, instagram.PlatformIOS443, instagram.PlatformIOS444, instagram.PlatformIOS445, instagram.PlatformIOS446, instagram.PlatformIOS447, instagram.PlatformIOS448, instagram.PlatformIOS449, instagram.PlatformIOS451, instagram.PlatformIOS452, instagram.PlatformIOS453, instagram.PlatformIOS454, instagram.PlatformIOS455, instagram.PlatformIOS456, instagram.PlatformIOS457, instagram.PlatformIOS458, instagram.PlatformIOS459, instagram.PlatformIOS461, instagram.PlatformIOS462, instagram.PlatformIOS463, instagram.PlatformIOS464, instagram.PlatformIOS465, instagram.PlatformIOS466, instagram.PlatformIOS467, instagram.PlatformIOS468, instagram.PlatformIOS469, instagram.PlatformIOS471, instagram.PlatformIOS472, instagram.PlatformIOS473, instagram.PlatformIOS474, instagram.PlatformIOS475, instagram.PlatformIOS476, instagram.PlatformIOS477, instagram.PlatformIOS478, instagram.PlatformIOS479, instagram.PlatformIOS481, instagram.PlatformIOS482, instagram.PlatformIOS483, instagram.PlatformIOS484, instagram.PlatformIOS485, instagram.PlatformIOS486, instagram.PlatformIOS487, instagram.PlatformIOS488, instagram.PlatformIOS489, instagram.PlatformIOS491, instagram.PlatformIOS492, instagram.PlatformIOS493, instagram.PlatformIOS494, instagram.PlatformIOS495, instagram.PlatformIOS496, instagram.PlatformIOS497, instagram.PlatformIOS498, instagram.PlatformIOS499, instagram.PlatformIOS501, instagram.PlatformIOS502, instagram.PlatformIOS503, instagram.PlatformIOS504, instagram.PlatformIOS505, instagram.PlatformIOS506, instagram.PlatformIOS507, instagram.PlatformIOS508, instagram.PlatformIOS509, instagram.PlatformIOS511, instagram.PlatformIOS512, instagram.PlatformIOS513, instagram.PlatformIOS514, instagram.PlatformIOS515, instagram.PlatformIOS516, instagram.PlatformIOS517, instagram.PlatformIOS518, instagram.PlatformIOS519, instagram.PlatformIOS521, instagram.PlatformIOS522, instagram.PlatformIOS523, instagram.PlatformIOS524, instagram.PlatformIOS525, instagram.PlatformIOS526, instagram.PlatformIOS527, instagram.PlatformIOS528, instagram.PlatformIOS529, instagram.PlatformIOS531, instagram.PlatformIOS532, instagram.PlatformIOS533, instagram.PlatformIOS534, instagram.PlatformIOS535, instagram.PlatformIOS536, instagram.PlatformIOS537, instagram.PlatformIOS538, instagram.PlatformIOS539, instagram.PlatformIOS541, instagram.PlatformIOS542, instagram.PlatformIOS543, instagram.PlatformIOS544, instagram.PlatformIOS545, instagram.PlatformIOS546, instagram.PlatformIOS547, instagram.PlatformIOS548, instagram.PlatformIOS549, instagram.PlatformIOS551, instagram.PlatformIOS552, instagram.PlatformIOS553, instagram.PlatformIOS554, instagram.PlatformIOS556, instagram.PlatformIOS557, instagram.PlatformIOS558, instagram.PlatformIOS559, instagram.PlatformIOS561:
		expectedFBAVPrefix = "FBAV/562."
		isValidUA = func(ua string) bool { return strings.Contains(ua, "FBAN/FBIOS") }
	default:
		// S23/Android — Samsung Galaxy Android UA
		poolKey = "android"
	}

	// Override validator cho S55x/S558: phải khớp FBAV/<ver>.
	// (iOS platforms override isValidUA trực tiếp trong case, không override lại ở đây.)
	if expectedFBAVPrefix != "" && verifyPlatform != instagram.PlatformIOS562 && verifyPlatform != instagram.PlatformIOS555 && verifyPlatform != instagram.PlatformIOS550 && verifyPlatform != instagram.PlatformIOS540 && verifyPlatform != instagram.PlatformIOS530 && verifyPlatform != instagram.PlatformIOS520 && verifyPlatform != instagram.PlatformIOS564 && verifyPlatform != instagram.PlatformIOS510 && verifyPlatform != instagram.PlatformIOS500 && verifyPlatform != instagram.PlatformIOS490 && verifyPlatform != instagram.PlatformIOS480 && verifyPlatform != instagram.PlatformIOS470 && verifyPlatform != instagram.PlatformIOS460 && verifyPlatform != instagram.PlatformIOS450 && verifyPlatform != instagram.PlatformIOS440 && verifyPlatform != instagram.PlatformIOS430 && verifyPlatform != instagram.PlatformIOS420 && verifyPlatform != instagram.PlatformIOS560 && verifyPlatform != instagram.PlatformIOS421 && verifyPlatform != instagram.PlatformIOS422 && verifyPlatform != instagram.PlatformIOS423 && verifyPlatform != instagram.PlatformIOS424 && verifyPlatform != instagram.PlatformIOS425 && verifyPlatform != instagram.PlatformIOS426 && verifyPlatform != instagram.PlatformIOS427 && verifyPlatform != instagram.PlatformIOS428 && verifyPlatform != instagram.PlatformIOS429 && verifyPlatform != instagram.PlatformIOS431 && verifyPlatform != instagram.PlatformIOS432 && verifyPlatform != instagram.PlatformIOS433 && verifyPlatform != instagram.PlatformIOS434 && verifyPlatform != instagram.PlatformIOS435 && verifyPlatform != instagram.PlatformIOS436 && verifyPlatform != instagram.PlatformIOS437 && verifyPlatform != instagram.PlatformIOS438 && verifyPlatform != instagram.PlatformIOS439 && verifyPlatform != instagram.PlatformIOS441 && verifyPlatform != instagram.PlatformIOS442 && verifyPlatform != instagram.PlatformIOS443 && verifyPlatform != instagram.PlatformIOS444 && verifyPlatform != instagram.PlatformIOS445 && verifyPlatform != instagram.PlatformIOS446 && verifyPlatform != instagram.PlatformIOS447 && verifyPlatform != instagram.PlatformIOS448 && verifyPlatform != instagram.PlatformIOS449 && verifyPlatform != instagram.PlatformIOS451 && verifyPlatform != instagram.PlatformIOS452 && verifyPlatform != instagram.PlatformIOS453 && verifyPlatform != instagram.PlatformIOS454 && verifyPlatform != instagram.PlatformIOS455 && verifyPlatform != instagram.PlatformIOS456 && verifyPlatform != instagram.PlatformIOS457 && verifyPlatform != instagram.PlatformIOS458 && verifyPlatform != instagram.PlatformIOS459 && verifyPlatform != instagram.PlatformIOS461 && verifyPlatform != instagram.PlatformIOS462 && verifyPlatform != instagram.PlatformIOS463 && verifyPlatform != instagram.PlatformIOS464 && verifyPlatform != instagram.PlatformIOS465 && verifyPlatform != instagram.PlatformIOS466 && verifyPlatform != instagram.PlatformIOS467 && verifyPlatform != instagram.PlatformIOS468 && verifyPlatform != instagram.PlatformIOS469 && verifyPlatform != instagram.PlatformIOS471 && verifyPlatform != instagram.PlatformIOS472 && verifyPlatform != instagram.PlatformIOS473 && verifyPlatform != instagram.PlatformIOS474 && verifyPlatform != instagram.PlatformIOS475 && verifyPlatform != instagram.PlatformIOS476 && verifyPlatform != instagram.PlatformIOS477 && verifyPlatform != instagram.PlatformIOS478 && verifyPlatform != instagram.PlatformIOS479 && verifyPlatform != instagram.PlatformIOS481 && verifyPlatform != instagram.PlatformIOS482 && verifyPlatform != instagram.PlatformIOS483 && verifyPlatform != instagram.PlatformIOS484 && verifyPlatform != instagram.PlatformIOS485 && verifyPlatform != instagram.PlatformIOS486 && verifyPlatform != instagram.PlatformIOS487 && verifyPlatform != instagram.PlatformIOS488 && verifyPlatform != instagram.PlatformIOS489 && verifyPlatform != instagram.PlatformIOS491 && verifyPlatform != instagram.PlatformIOS492 && verifyPlatform != instagram.PlatformIOS493 && verifyPlatform != instagram.PlatformIOS494 && verifyPlatform != instagram.PlatformIOS495 && verifyPlatform != instagram.PlatformIOS496 && verifyPlatform != instagram.PlatformIOS497 && verifyPlatform != instagram.PlatformIOS498 && verifyPlatform != instagram.PlatformIOS499 && verifyPlatform != instagram.PlatformIOS501 && verifyPlatform != instagram.PlatformIOS502 && verifyPlatform != instagram.PlatformIOS503 && verifyPlatform != instagram.PlatformIOS504 && verifyPlatform != instagram.PlatformIOS505 && verifyPlatform != instagram.PlatformIOS506 && verifyPlatform != instagram.PlatformIOS507 && verifyPlatform != instagram.PlatformIOS508 && verifyPlatform != instagram.PlatformIOS509 && verifyPlatform != instagram.PlatformIOS511 && verifyPlatform != instagram.PlatformIOS512 && verifyPlatform != instagram.PlatformIOS513 && verifyPlatform != instagram.PlatformIOS514 && verifyPlatform != instagram.PlatformIOS515 && verifyPlatform != instagram.PlatformIOS516 && verifyPlatform != instagram.PlatformIOS517 && verifyPlatform != instagram.PlatformIOS518 && verifyPlatform != instagram.PlatformIOS519 && verifyPlatform != instagram.PlatformIOS521 && verifyPlatform != instagram.PlatformIOS522 && verifyPlatform != instagram.PlatformIOS523 && verifyPlatform != instagram.PlatformIOS524 && verifyPlatform != instagram.PlatformIOS525 && verifyPlatform != instagram.PlatformIOS526 && verifyPlatform != instagram.PlatformIOS527 && verifyPlatform != instagram.PlatformIOS528 && verifyPlatform != instagram.PlatformIOS529 && verifyPlatform != instagram.PlatformIOS531 && verifyPlatform != instagram.PlatformIOS532 && verifyPlatform != instagram.PlatformIOS533 && verifyPlatform != instagram.PlatformIOS534 && verifyPlatform != instagram.PlatformIOS535 && verifyPlatform != instagram.PlatformIOS536 && verifyPlatform != instagram.PlatformIOS537 && verifyPlatform != instagram.PlatformIOS538 && verifyPlatform != instagram.PlatformIOS539 && verifyPlatform != instagram.PlatformIOS541 && verifyPlatform != instagram.PlatformIOS542 && verifyPlatform != instagram.PlatformIOS543 && verifyPlatform != instagram.PlatformIOS544 && verifyPlatform != instagram.PlatformIOS545 && verifyPlatform != instagram.PlatformIOS546 && verifyPlatform != instagram.PlatformIOS547 && verifyPlatform != instagram.PlatformIOS548 && verifyPlatform != instagram.PlatformIOS549 && verifyPlatform != instagram.PlatformIOS551 && verifyPlatform != instagram.PlatformIOS552 && verifyPlatform != instagram.PlatformIOS553 && verifyPlatform != instagram.PlatformIOS554 && verifyPlatform != instagram.PlatformIOS556 && verifyPlatform != instagram.PlatformIOS557 && verifyPlatform != instagram.PlatformIOS558 && verifyPlatform != instagram.PlatformIOS559 && verifyPlatform != instagram.PlatformIOS561 {
		want := expectedFBAVPrefix
		isValidUA = func(ua string) bool { return strings.Contains(ua, want) }
	}

	// s273 pool có nhiều FBAV (273/547/551/555) — override validator chấp nhận cả pool.
	// QUAN TRỌNG: phải KÈM Dalvik prefix — nếu không, UA reg s5xx (s547/s551/s555 native, KHÔNG Dalvik)
	// sẽ pass validator → giữ UA reg → verify dùng sai format. Fix 2026-05-28.
	if verifyPlatform == instagram.PlatformS273 {
		isValidUA = func(ua string) bool {
			if !strings.Contains(ua, "Dalvik/") {
				return false
			}
			return strings.Contains(ua, "FBAV/273.") ||
				strings.Contains(ua, "FBAV/547.") ||
				strings.Contains(ua, "FBAV/551.") ||
				strings.Contains(ua, "FBAV/555.")
		}
	}

	// 1. accountUA valid → ưu tiên dùng (bỏ qua khi BuildUA=true — user muốn build UA mới)
	if !cfg.BuildUA && accountUA != "" && isValidUA(accountUA) {
		return maybeXID(accountUA)
	}

	// 2. Factory UA của platform (S55x/S558 sinh UA đúng version, random device).
	// Truyền countryCode để factory pick FBCR + FBLC theo country của phone/proxy.
	// Scheduler có thể regenerate với country mới sau khi acquire proxy thật.
	if expectedFBAVPrefix != "" {
		if ua := instagram.PlatformVerifyUA(verifyPlatform, countryCode); ua != "" {
			return maybeXID(ua)
		}
	}

	// 3. BuildUA — build UA động cho generic Android (S23/Android).
	// Versioned (S55x/S56x/...) KHÔNG vào đây vì đã return ở bước 2 (factory UA).
	// Web/WebAndroid KHÔNG vào đây vì poolKey != "android" → điều kiện expectedFBAVPrefix=="".
	if cfg.BuildUA && expectedFBAVPrefix == "" && poolKey == "android" {
		dev := fakeinfo.RandomDeviceProfile()
		locale := fakeinfo.LocaleFromCountry(countryCode)
		sim := fakeinfo.RandomSimProfile(countryCode)
		carrier := sim.OperatorName
		if carrier == "" {
			carrier = fakeinfo.RandomCarrier()
		}
		fbVer, fbBuild := fakeinfo.RandomFbVersionVer()
		if ua := fakeinfo.BuildAndroidUAWithOpts(dev, locale, carrier, fbVer, fbBuild, cfg.AddVirtualSpecAndroid, false); ua != "" {
			return maybeXID(ua)
		}
	}

	// 4. Pool từ Config/UserAgent/{type}_UG.txt theo poolKey
	if ua := resolveUAPool(poolKey); ua != "" && isValidUA(ua) {
		return maybeXID(ua)
	}

	// 5. Per-platform fallback random UA generator (tránh return "" → worker hardcode).
	//    WebAndroid: random Chrome Android từ RandomChromeAndroidProfile (~116k combinations).
	//    Web (api mfb): random từ WebChrome pool (Config/UserAgent/WebChrome_UA.txt — user-managed).
	//    Đảm bảo MỖI VER có UA UNIQUE thay vì cùng 1 hardcoded UA → tránh FB fingerprint match.
	if verifyPlatform == instagram.PlatformWebAndroid {
		if ua := fakeinfo.RandomChromeAndroidProfile().UserAgent; ua != "" {
			return maybeXID(ua)
		}
	}
	if verifyPlatform == instagram.PlatformWeb {
		// Đọc trực tiếp từ WebChrome pool — không validate Chrome/Mobile vì user tự quản lý file.
		// Pool sẽ tự lớn dần qua learning loop (verify success → append UA).
		if ua := fakeinfo.RandomUAFromPool(fakeinfo.UAKindWebChrome); ua != "" {
			return maybeXID(ua)
		}
	}

	// 5. Trả "" → worker tự dùng default (Chrome Desktop cho web, Chrome Mobile cho webandroid)
	return ""
}

// pickRawUAFromActivePool chọn ngẫu nhiên 1 UA từ pool active.
// Priority:
//  1. uaPools[UaPoolKey] (textarea user paste) → resolveUAPool đã xử lý
//  2. Fallback sang embed pool tương ứng kind (iphone/android)
//     nếu textarea rỗng.
//
// Trả "" khi không tìm được UA nào → caller dùng UA build động.
// uaKindFromPoolKey map tab UI key → fakeinfo.UAPoolKind (→ Config/UserAgent/{type}_UG.txt).
func uaKindFromPoolKey(key string) fakeinfo.UAPoolKind {
	switch strings.TrimSpace(key) {
	case "iphone":
		return fakeinfo.UAKindIOS
	case "request":
		return fakeinfo.UAKindRequest
	case "webchrome":
		return fakeinfo.UAKindWebChrome
	case "android_mess":
		return fakeinfo.UAKindAndroidMess
	case "ios_mess":
		return fakeinfo.UAKindIOSMess
	default: // "android" hoặc rỗng
		return fakeinfo.UAKindAndroid
	}
}

// phoneToCountryCode trả ISO-2 country code từ số điện thoại E.164 (vd "+84..." → "VN").
// Trả "" nếu không tìm được.
func phoneToCountryCode(phone string) string {
	if phone == "" {
		return ""
	}
	if p, ok := fakeinfo.FindCountryByPhonePrefix(phone); ok {
		return p.CountryCode
	}
	return ""
}

// originalUABaseForPlatform trả UA gốc cố định (carrier=Viettel placeholder).
// messOrcaUAGoc — UA Gốc Orca (Messenger Android) cố định SM-G996B + FBAV/FBBV theo version
// (FBCR/Viettel → thay theo country qua reFBCR như FB4A).
func messOrcaUAGoc(fbav, fbbv string) string {
	return "Dalvik/2.1.0 (Linux; U; Android 15; SM-G996B Build/AP3A.240905.015.A2) " +
		"[FBAN/Orca-Android;FBAV/" + fbav + ";FBPN/com.facebook.orca;FBLC/en_GB;FBBV/" + fbbv +
		";FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBDV/SM-G996B;FBSV/15;FBCA/arm64-v8a:null;" +
		"FBDM/{density=2.8125,width=1080,height=2400};FB_FW/1;]"
}

func originalUABaseForPlatform(platform string) string {
	if ua := originalUABaseForSxxx(platform); ua != "" {
		return ua
	}
	switch platform {
	// ── Messenger (Orca) Android — UA Gốc capture cố định SM-G996B, FBAV theo version (reg+ver) ──
	case instagram.PlatformAppMV3, instagram.PlatformAppMessV3:
		return messOrcaUAGoc("530.1.0.67.107", "814020040")
	case instagram.PlatformAppMV3_535, instagram.PlatformAppMessV3_535:
		return messOrcaUAGoc("535.0.0.101.107", "840054075")
	case instagram.PlatformAppMV3_545, instagram.PlatformAppMessV3_545:
		return messOrcaUAGoc("545.0.0.27.62", "870175947")
	case instagram.PlatformAppMV3_555, instagram.PlatformAppMessV3_555:
		return messOrcaUAGoc("555.0.0.56.66", "930834402")
	case instagram.PlatformAppMV3_563, instagram.PlatformAppMessV3_563:
		return messOrcaUAGoc("563.0.0.47.86", "979328543")
	case instagram.PlatformAppMV3_564, instagram.PlatformAppMessV3_564:
		return messOrcaUAGoc("564.0.0.42.89", "984961990")
	case instagram.PlatformAppMV3_565, instagram.PlatformAppMessV3_565:
		return messOrcaUAGoc("565.0.0.0.2", "981799924")
	case instagram.PlatformAppMV3_525, instagram.PlatformAppMessV3_525:
		return messOrcaUAGoc("525.0.0.44.108", "792260954")
	case instagram.PlatformAppMV3_515, instagram.PlatformAppMessV3_515:
		return messOrcaUAGoc("515.0.0.51.108", "763707183")
	case instagram.PlatformAppMV3_505, instagram.PlatformAppMessV3_505:
		return messOrcaUAGoc("505.0.0.62.82", "730961636")
	case instagram.PlatformAppMV3_490, instagram.PlatformAppMessV3_490:
		return messOrcaUAGoc("490.0.0.42.108", "684080902")
	// ── Messenger Lite iOS — UA Gốc cố định iPhone15,2 + FBAV 563 (reg+ver) ──
	case instagram.PlatformIOSMessReg, instagram.PlatformIOSMess:
		return "LightSpeed [FBAN/MessengerLiteForiOS;FBAV/563.0.0.27.106;FBBV/980221516;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBCR/;FBID/phone;FBLC/en_US;FBOP/0]"
	case instagram.PlatformS560V3:
		return "[FBAN/FB4A;FBAV/560.0.0.57.63;FBBV/963497253;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]"
	case instagram.PlatformS399:
		return s399reg.OriginalUA
	case instagram.PlatformS550V2:
		return "[FBAN/FB4A;FBAV/550.0.0.40.60;FBBV/900039717;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]"
	case instagram.PlatformS551V2:
		return "[FBAN/FB4A;FBAV/551.1.0.58.63;FBBV/906186219;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]"
	case instagram.PlatformS552V2:
		return "[FBAN/FB4A;FBAV/552.1.0.45.68;FBBV/911260592;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]"
	case instagram.PlatformS553V2:
		return "[FBAN/FB4A;FBAV/553.0.0.56.58;FBBV/918989583;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]"
	case instagram.PlatformS554V2:
		return "[FBAN/FB4A;FBAV/554.0.0.57.70;FBBV/926292396;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]"
	case instagram.PlatformS556V2:
		return "[FBAN/FB4A;FBAV/556.1.0.63.64;FBBV/942217461;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]"
	case instagram.PlatformS557V2:
		return "[FBAN/FB4A;FBAV/557.0.0.59.72;FBBV/953308969;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]"
		// KHÔNG có case PlatformS273 — s273 cần MỖI ACCOUNT random device khác nhau
		// (giống capture chuẩn API 273: Xiaomi M2305F1I, Realme RMX2193, Samsung SM-S906N...).
		// originalUABase return "" cho s273 → pickUAForVerifyPlatform fall through xuống
		// path PlatformVerifyUA("s273", country) → gọi s273.RandomUA() random device + FBAV 273.
	case instagram.PlatformIOS520:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/520.0.0.38.101;FBBV/756351453;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS530:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/530.0.0.59.75;FBBV/790686474;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS540:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/540.0.0.44.68;FBBV/828638047;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS550:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/550.0.0.34.65;FBBV/890804754;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS555:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/555.0.0.36.63;FBBV/923840166;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS510:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/510.0.0.38.93;FBBV/724276253;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS500:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/500.0.0.52.98;FBBV/696635672;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS490:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/490.1.0.49.107;FBBV/663124541;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS480:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/480.0.0.32.109;FBBV/638556369;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS470:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/470.1.0.43.103;FBBV/617058003;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS460:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/460.0.0.31.103;FBBV/588708950;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS450:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/450.0.0.38.108;FBBV/564431005;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS421:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/421.0.0.24.58;FBBV/489261892;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS422:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/422.0.0.24.58;FBBV/491634260;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS423:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/423.0.0.24.58;FBBV/494006628;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS424:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/424.0.0.24.58;FBBV/496378996;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS425:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/425.0.0.24.58;FBBV/498751364;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS426:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/426.0.0.24.58;FBBV/501123732;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS427:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/427.0.0.24.58;FBBV/503496100;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS428:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/428.0.0.24.58;FBBV/505868468;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS429:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/429.0.0.24.58;FBBV/508240836;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS431:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/431.0.0.33.114;FBBV/513304094;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS432:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/432.0.0.33.114;FBBV/515994984;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS433:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/433.0.0.33.114;FBBV/518685874;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS434:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/434.0.0.33.114;FBBV/521376764;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS435:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/435.0.0.33.114;FBBV/524067654;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS436:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/436.0.0.33.114;FBBV/526758544;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS437:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/437.0.0.33.114;FBBV/529449434;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS438:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/438.0.0.33.114;FBBV/532140324;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS439:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/439.0.0.33.114;FBBV/534831214;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS441:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/441.0.0.38.108;FBBV/540212995;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS442:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/442.0.0.38.108;FBBV/542903885;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS443:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/443.0.0.38.108;FBBV/545594775;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS444:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/444.0.0.38.108;FBBV/548285665;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS445:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/445.0.0.38.108;FBBV/550976555;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS446:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/446.0.0.38.108;FBBV/553667445;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS447:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/447.0.0.38.108;FBBV/556358335;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS448:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/448.0.0.38.108;FBBV/559049225;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS449:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/449.0.0.38.108;FBBV/561740115;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS451:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/451.0.0.38.108;FBBV/566858800;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS452:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/452.0.0.38.108;FBBV/569286594;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS453:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/453.0.0.38.108;FBBV/571714389;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS454:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/454.0.0.38.108;FBBV/574142183;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS455:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/455.0.0.38.108;FBBV/576569978;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS456:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/456.0.0.38.108;FBBV/578997772;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS457:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/457.0.0.38.108;FBBV/581425567;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS458:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/458.0.0.38.108;FBBV/583853361;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS459:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/459.0.0.38.108;FBBV/586281156;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS461:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/461.0.0.31.103;FBBV/591543855;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS462:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/462.0.0.31.103;FBBV/594378761;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS463:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/463.0.0.31.103;FBBV/597213666;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS464:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/464.0.0.31.103;FBBV/600048571;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS465:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/465.0.0.31.103;FBBV/602883477;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS466:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/466.0.0.31.103;FBBV/605718382;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS467:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/467.0.0.31.103;FBBV/608553287;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS468:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/468.0.0.31.103;FBBV/611388192;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS469:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/469.0.0.31.103;FBBV/614223098;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS471:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/471.1.0.43.103;FBBV/619207840;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS472:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/472.1.0.43.103;FBBV/621357676;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS473:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/473.1.0.43.103;FBBV/623507513;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS474:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/474.1.0.43.103;FBBV/625657349;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS475:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/475.1.0.43.103;FBBV/627807186;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS476:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/476.1.0.43.103;FBBV/629957023;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS477:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/477.1.0.43.103;FBBV/632106859;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS478:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/478.1.0.43.103;FBBV/634256696;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS479:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/479.1.0.43.103;FBBV/636406532;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS481:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/481.0.0.32.109;FBBV/641013186;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS482:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/482.0.0.32.109;FBBV/643470003;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS483:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/483.0.0.32.109;FBBV/645926821;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS484:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/484.0.0.32.109;FBBV/648383638;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS485:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/485.0.0.32.109;FBBV/650840455;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS486:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/486.0.0.32.109;FBBV/653297272;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS487:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/487.0.0.32.109;FBBV/655754089;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS488:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/488.0.0.32.109;FBBV/658210907;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS489:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/489.0.0.32.109;FBBV/660667724;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS491:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/491.1.0.49.107;FBBV/666475654;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS492:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/492.1.0.49.107;FBBV/669826767;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS493:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/493.1.0.49.107;FBBV/673177880;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS494:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/494.1.0.49.107;FBBV/676528993;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS495:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/495.1.0.49.107;FBBV/679880107;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS496:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/496.1.0.49.107;FBBV/683231220;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS497:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/497.1.0.49.107;FBBV/686582333;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS498:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/498.1.0.49.107;FBBV/689933446;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS499:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/499.1.0.49.107;FBBV/693284559;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS501:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/501.0.0.52.98;FBBV/699399730;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS502:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/502.0.0.52.98;FBBV/702163788;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS503:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/503.0.0.52.98;FBBV/704927846;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS504:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/504.0.0.52.98;FBBV/707691904;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS505:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/505.0.0.52.98;FBBV/710455963;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS506:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/506.0.0.52.98;FBBV/713220021;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS507:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/507.0.0.52.98;FBBV/715984079;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS508:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/508.0.0.52.98;FBBV/718748137;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS509:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/509.0.0.52.98;FBBV/721512195;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS511:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/511.0.0.38.93;FBBV/727483773;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS512:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/512.0.0.38.93;FBBV/730691293;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS513:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/513.0.0.38.93;FBBV/733898813;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS514:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/514.0.0.38.93;FBBV/737106333;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS515:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/515.0.0.38.93;FBBV/740313853;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS516:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/516.0.0.38.93;FBBV/743521373;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS517:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/517.0.0.38.93;FBBV/746728893;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS518:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/518.0.0.38.93;FBBV/749936413;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS519:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/519.0.0.38.93;FBBV/753143933;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS521:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/521.0.0.38.101;FBBV/759784955;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS522:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/522.0.0.38.101;FBBV/763218457;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS523:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/523.0.0.38.101;FBBV/766651959;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS524:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/524.0.0.38.101;FBBV/770085461;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS525:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/525.0.0.38.101;FBBV/773518964;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS526:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/526.0.0.38.101;FBBV/776952466;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS527:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/527.0.0.38.101;FBBV/780385968;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS528:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/528.0.0.38.101;FBBV/783819470;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS529:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/529.0.0.38.101;FBBV/787252972;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS531:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/531.0.0.59.75;FBBV/794481631;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS532:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/532.0.0.59.75;FBBV/798276789;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS533:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/533.0.0.59.75;FBBV/802071946;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS534:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/534.0.0.59.75;FBBV/805867103;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS535:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/535.0.0.59.75;FBBV/809662261;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS536:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/536.0.0.59.75;FBBV/813457418;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS537:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/537.0.0.59.75;FBBV/817252575;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS538:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/538.0.0.59.75;FBBV/821047732;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS539:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/539.0.0.59.75;FBBV/824842890;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS541:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/541.0.0.44.68;FBBV/834854718;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS542:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/542.0.0.44.68;FBBV/841071388;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS543:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/543.0.0.44.68;FBBV/847288059;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS544:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/544.0.0.44.68;FBBV/853504730;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS545:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/545.0.0.44.68;FBBV/859721401;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS546:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/546.0.0.44.68;FBBV/865938071;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS547:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/547.0.0.44.68;FBBV/872154742;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS548:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/548.0.0.44.68;FBBV/878371413;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS549:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/549.0.0.44.68;FBBV/884588083;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS551:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/551.0.0.34.65;FBBV/897411836;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS552:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/552.0.0.34.65;FBBV/904018919;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS553:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/553.0.0.34.65;FBBV/910626001;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS554:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/554.0.0.34.65;FBBV/917233084;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS556:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/556.0.0.36.63;FBBV/930684417;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS557:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/557.0.0.36.63;FBBV/937528668;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS558:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/558.0.0.36.63;FBBV/944372920;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS559:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/559.0.0.36.63;FBBV/951217171;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS561:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/561.0.0.36.63;FBBV/964905673;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS440:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/450.0.0.38.108;FBBV/564431005;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS430:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/430.0.0.33.114;FBBV/510613204;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS420:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/420.0.0.24.58;FBBV/486889524;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS560:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/555.0.0.36.63;FBBV/923840166;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS564:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/564.0.0.57.71;FBBV/985438427;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	case instagram.PlatformIOS562, instagram.PlatformIOS563:
		return "Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/563.0.0.67.72;FBBV/980285082;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]"
	}
	return ""
}

var reFBCR = regexp.MustCompile(`FBCR/[^;]+`)

// appendXIDToUA chèn XID/<random16>; vào cuối UA trước dấu ].
// Ví dụ: [...;FBCA/arm64-v8a;] → [...;FBCA/arm64-v8a;XID/f458iwz5cu3hkl1t;]
func appendXIDToUA(ua string) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	xid := "XID/" + string(b) + ";"
	if idx := strings.LastIndex(ua, "]"); idx >= 0 {
		sep := ""
		if idx > 0 && ua[idx-1] != ';' {
			sep = ";"
		}
		return ua[:idx] + sep + xid + "]"
	}
	return ua + xid
}

// originalUAForPlatform trả UA gốc cố định với FBCR thay theo countryCode.
// Nếu countryCode rỗng hoặc không tìm được carrier → giữ nguyên Viettel.
// Trả "" nếu platform không có UA gốc cố định.
func originalUAForPlatform(platform, countryCode string) string {
	ua, _ := originalUAForPlatformWithSim(platform, countryCode)
	return ua
}

var missingPhoneCCMu sync.Mutex

// logMissingPhoneCountryCode append countryCode vào Config/PhoneDatabase/_missing_country_codes.txt
// (dedup theo line). Gọi khi cả PhoneFromDatabase + GeneratePhoneByCountry đều trả "" cho 1 country —
// user xem file để bổ sung pattern phone cho country đó.
func logMissingPhoneCountryCode(cc string) {
	cc = strings.ToLower(strings.TrimSpace(cc))
	if cc == "" {
		return
	}
	missingPhoneCCMu.Lock()
	defer missingPhoneCCMu.Unlock()
	path := filepath.Join(defaultPhoneDatabaseDir(), "_missing_country_codes.txt")
	if data, err := os.ReadFile(path); err == nil {
		for _, line := range splitLines(string(data)) {
			if strings.EqualFold(strings.TrimSpace(line), cc) {
				return // đã log → skip
			}
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(cc + "\n")
}

// defaultPhoneDatabaseDir trả về folder chứa per-country phone pattern files.
// Path: Config/phone_database (mỗi country: AnyName=CC.locale_REGION.txt với pattern phone bên trong).
func defaultPhoneDatabaseDir() string {
	candidate := filepath.Join(AppDataDir(), "Config", "phone_database")
	if mkErr := os.MkdirAll(candidate, 0755); mkErr == nil {
		return candidate
	}
	homeDir, _ := os.UserHomeDir()
	fallback := filepath.Join(homeDir, "Documents", "HVR", "Config", "phone_database")
	_ = os.MkdirAll(fallback, 0755)
	return fallback
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Split(s, "\n")
}

// permFileMu serialize reads/writes vào Config/Permanent/*.txt từ nhiều reg goroutines.
var permFileMu sync.Mutex

// appendUniqueLineToPermanentFile thêm line vào file nếu chưa tồn tại.
// Dùng để tích lũy phone/email thành công vào file permanent qua nhiều lần chạy.
func appendUniqueLineToPermanentFile(filePath, line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	permFileMu.Lock()
	defer permFileMu.Unlock()
	if data, err := os.ReadFile(filePath); err == nil {
		for existing := range strings.SplitSeq(string(data), "\n") {
			if strings.TrimSpace(existing) == line {
				return
			}
		}
	}
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line + "\n")
}
