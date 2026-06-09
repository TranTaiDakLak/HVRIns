package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	goruntime "runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"os/exec"

	"HVRIns/internal/clonehv"
	"HVRIns/internal/cookie"
	"HVRIns/internal/email"
	emailrent "HVRIns/internal/email/rent"
	emailtemp "HVRIns/internal/email/temp"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	verifybase "HVRIns/internal/instagram/verify/verifybase"
	"HVRIns/internal/fbdata"
	"HVRIns/internal/proxy"
	resultpkg "HVRIns/internal/result"
	appsettings "HVRIns/internal/settings"
	"HVRIns/internal/settings/adapter"
	"HVRIns/internal/settings/model"

	// Facebook platform implementations — named imports trigger init() registration
	// web register (named: use RandomRegInput etc.)
	// Skeleton platforms (blank import — only need init() registration)
	androidreg "HVRIns/internal/instagram/register/android"
	s399reg "HVRIns/internal/instagram/register/android/s399"
	_ "HVRIns/internal/instagram/register/chrome"
	ioshttpreg "HVRIns/internal/instagram/register/ioshttp"
	webandroidreg "HVRIns/internal/instagram/register/webandroid"

	// Verify platforms
	_ "HVRIns/internal/instagram/verify/android"
	_ "HVRIns/internal/instagram/verify/android/appmessv3"
	_ "HVRIns/internal/instagram/verify/ios/iosmess"
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
type Account struct {
	ID           int    `json:"id"`
	UID          string `json:"uid"`
	FullData     string `json:"fullData"` // Dữ liệu gốc (dòng import nguyên bản)
	Password     string `json:"password"`
	Twofa        string `json:"twofa"`
	Email        string `json:"email"`
	PassMail     string `json:"passMail"`
	MailRecovery string `json:"mailRecovery"`
	Cookie       string `json:"cookie"`
	Token        string `json:"token"`
	Status       string `json:"status"`
	Checkpoint   string `json:"checkpoint"`
	StatusAds    string `json:"statusAds"`
	BM           string `json:"bm"`
	TKQC         string `json:"tkqc"`
	ChatSupport  string `json:"chatSupport"`
	FullName     string `json:"fullName"`
	Location     string `json:"location"`
	Avatar       string `json:"avatar"`
	Cover        string `json:"cover"`
	Phone        string `json:"phone"`
	Proxy        string `json:"proxy"`
	UserAgent    string `json:"userAgent"`
	Note         string `json:"note"`
	NoteRun      string `json:"noteRun"`
	ImportTime   string `json:"importTime"`
	Category     string `json:"category"`
	LastRun      string `json:"lastRun"`
	Activity     string `json:"activity"`
	SourceCode   string `json:"sourceCode"`
	CategoryID   *int   `json:"categoryId"`
	// EmailMeta — TempMail provider creds (JSON-encoded) để verify Restore
	// và đọc OTP từ inbox đã có sẵn (skip CreateEmail + skip AddEmail step).
	// Empty cho mode Phone/Mail (giả) → verify dùng flow CreateEmail mới.
	EmailMeta string `json:"emailMeta,omitempty"`

	// iOS partial reg tokens — được lưu vào file format với prefix SRN:/SCUID:
	// để file-based verify (tab ver thủ công) dùng được thay vì chỉ inline auto-verify.
	Srnonce               string `json:"srnonce,omitempty"`
	SessionlessCryptedUID string `json:"sessionlessCryptedUID,omitempty"`
}

type AccountFilter struct {
	Keyword    string `json:"keyword"`
	Status     string `json:"status"`
	CategoryID *int   `json:"categoryId"`
	SortBy     string `json:"sortBy"`
	SortDir    string `json:"sortDir"`
}

type AccountListResult struct {
	Items []Account `json:"items"`
	Total int       `json:"total"`
}

type ImportResult struct {
	Imported int      `json:"imported"`
	Errors   []string `json:"errors"`
}

type DeleteResult struct {
	Deleted int `json:"deleted"`
}

// App struct
type App struct {
	ctx           context.Context
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

// regVersionStat — đếm thành công / thất bại của 1 version reg trong run hiện tại.
type regVersionStat struct {
	Success int
	Fail    int
}

// mailDomainStat — đếm verify / live theo domain mail trong run hiện tại.
type mailDomainStat struct {
	Veri int // tổng account đã verify (có email)
	Live int // số live
}

// MailDomainStatRow — 1 dòng trong tab "Thống kê Mail Domain".
type MailDomainStatRow struct {
	Index  int     `json:"index"`
	Domain string  `json:"domain"`
	Veri   int     `json:"veri"`
	Live   int     `json:"live"`
	Die    int     `json:"die"`
	Rate   float64 `json:"rate"` // live / veri (0–1)
}

// RegStatRow — 1 dòng trong tab "Thống kê REG" (STT, API, thành công, thất bại, tỉ lệ).
type RegStatRow struct {
	Index    int    `json:"index"`    // STT (1-based)
	Platform string `json:"platform"` // API / version (vd "s560", "android")
	Success  int    `json:"success"`
	Fail     int    `json:"fail"`
	Total    int    `json:"total"`
}

// NewApp khởi tạo App struct với slice accounts rỗng.
// Wails gọi hàm này 1 lần khi app khởi động — không nhận tham số.
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
	dir := filepath.Join(appDataDir(), "Config", "RentMail")
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
func (a *App) startup(ctx context.Context) {
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
	emailtemp.SetMailTempComDomainCachePath(filepath.Join(appDataDir(), "Config", "mailtempcom_domains.txt"))

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
	fakeinfo.LoadPhoneDatabase(filepath.Join(appDataDir(), "Config", "phone_database"))
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
func (a *App) runMemoryMaintenance(ctx context.Context) {
	// Aggressive cadence cho 24/7: GC 1 phút, watchdog 20 giây.
	// Temp mail workload tạo ~2-3 MB/s allocation → GC 1min đủ catch up.
	gcTicker := time.NewTicker(1 * time.Minute)
	defer gcTicker.Stop()
	watchdogTicker := time.NewTicker(20 * time.Second)
	defer watchdogTicker.Stop()
	// Session pool cleanup: mỗi 10 phút close idle TCP/TLS conn của tất cả
	// session trong iOS/WebAndroid pool (không remove session). Giải phóng
	// native buffer NGOÀI Go heap — GC không động tới nên cần force close.
	// Quan trọng cho run dài (12h+): nếu không clean, idle conn tích lũy
	// → RSS tăng dần dù Go heap thấp.
	sessionPoolTicker := time.NewTicker(10 * time.Minute)
	defer sessionPoolTicker.Stop()

	// Cooldown auto cleanup — tránh trigger liên tục khi RSS dao động quanh threshold
	var lastAutoCleanup time.Time
	const autoCleanupCooldown = 3 * time.Minute

	// Adaptive GOGC theo run state:
	//   active (run đang chạy): GOGC=50 — GC aggressive, ngăn drift
	//   idle (không run): GOGC=200 — GC ít hơn, tiết kiệm CPU GC khi app idle
	// Swap chỉ khi state thực sự đổi để tránh syscall không cần thiết.
	const (
		gogcActive = 50
		gogcIdle   = 200
	)
	currentGOGC := gogcActive // startup giả định active để conservative

	for {
		select {
		case <-ctx.Done():
			return
		case <-gcTicker.C:
			var before goruntime.MemStats
			goruntime.ReadMemStats(&before)
			goruntime.GC()
			debug.FreeOSMemory()
			var after goruntime.MemStats
			goruntime.ReadMemStats(&after)
			freed := int64(before.HeapInuse) - int64(after.HeapInuse)
			slog.Info("memory maintenance gc",
				"heap_before_mb", before.HeapInuse/1024/1024,
				"heap_after_mb", after.HeapInuse/1024/1024,
				"freed_mb", freed/1024/1024,
				"rss_mb", getProcessMemoryMB(),
				"transport_pool", proxy.TransportPoolStats(),
				"goroutines", goruntime.NumGoroutine())
		case <-watchdogTicker.C:
			// Adaptive GOGC: khi run active → GC aggressive (50); idle → GC nhẹ (200)
			a.verifyMu.Lock()
			verRunning := a.isRunning
			a.verifyMu.Unlock()
			a.registerMu.Lock()
			regRunning := a.registerCancel != nil
			a.registerMu.Unlock()
			anyRunning := verRunning || regRunning
			desiredGOGC := gogcIdle
			if anyRunning {
				desiredGOGC = gogcActive
			}
			if desiredGOGC != currentGOGC {
				debug.SetGCPercent(desiredGOGC)
				currentGOGC = desiredGOGC
				slog.Debug("adaptive GOGC swap",
					"new_gogc", desiredGOGC,
					"reason", map[bool]string{true: "active", false: "idle"}[anyRunning])
			}

			var m goruntime.MemStats
			goruntime.ReadMemStats(&m)
			heapMB := m.HeapInuse / 1024 / 1024
			rssMB := getProcessMemoryMB() // process RSS thật (gồm WebView2 + native)
			numGoroutine := goruntime.NumGoroutine()

			// Goroutine leak detection: nếu > 2000 goroutine thì gần chắc là leak
			// (max case: 300 luồng × 3-4 goroutine mỗi thread = ~1200). Log warning.
			if numGoroutine > 2000 {
				slog.Warn("goroutine count high — possible leak",
					"goroutines", numGoroutine,
					"heap_mb", heapMB)
			}

			// Proactive GC: nếu heap > 600 MB → force GC silent (không notify).
			// App bình thường 200-500 MB cho 400 luồng; vượt 600 MB = drift, cần cleanup.
			if heapMB > 600 {
				goruntime.GC()
				debug.FreeOSMemory()
				slog.Debug("proactive gc triggered", "heap_mb", heapMB)
			}

			// AUTO CLEANUP — khi RSS process vượt threshold động → tự gọi cleanup
			// (close idle conn + GC + FreeOSMemory). Tránh phụ thuộc user bấm
			// nút "Dọn RAM" thủ công. Cooldown 3 phút để không thrash GC.
			//
			// Threshold dùng RSS thay vì heap vì heap có thể thấp nhưng native
			// TLS/WebView2 chiếm RSS — heap-based watchdog miss case này.
			//
			// 1500 MB: cleanup nhẹ (chỉ idle conn + GC)
			// 2000 MB: cleanup AGGRESSIVE (force GC 2 lần, log warn)
			if rssMB >= 1500 && time.Since(lastAutoCleanup) > autoCleanupCooldown {
				aggressive := rssMB >= 2000
				closedIOS := 0
				closedAndroid := 0
				if ioshttpreg.SharedSessionPool != nil {
					closedIOS = ioshttpreg.SharedSessionPool.CloseIdleConnsAll()
				}
				if webandroidreg.SharedSessionPool != nil {
					closedAndroid = webandroidreg.SharedSessionPool.CloseIdleConnsAll()
				}
				bancloneTransport.CloseIdleConnections()

				goruntime.GC()
				debug.FreeOSMemory()
				if aggressive {
					// 2nd GC pass — first GC moves objects to free list, second sweeps
					goruntime.GC()
					debug.FreeOSMemory()
				}

				rssAfter := getProcessMemoryMB()
				slog.Info("auto cleanup triggered",
					"reason", "rss_threshold",
					"rss_before_mb", rssMB,
					"rss_after_mb", rssAfter,
					"freed_mb", rssMB-rssAfter,
					"aggressive", aggressive,
					"ios_closed", closedIOS,
					"android_closed", closedAndroid,
					"goroutines", numGoroutine)
				lastAutoCleanup = time.Now()
			}

			// Warning threshold — cảnh báo user restart sau batch nếu vượt 1.5 GB heap.
			if heapMB > 1500 {
				slog.Warn("memory watchdog high",
					"heap_mb", heapMB,
					"sys_mb", m.Sys/1024/1024,
					"rss_mb", rssMB,
					"goroutines", numGoroutine)
				if a.ctx != nil {
					runtime.EventsEmit(a.ctx, "system:memory-warning", map[string]interface{}{
						"heapMB": heapMB,
						"rssMB":  rssMB,
						"msg":    fmt.Sprintf("⚠️ RAM app ở mức cao (heap %d MB / RSS %.0f MB) — cân nhắc restart sau batch hiện tại", heapMB, rssMB),
					})
				}
			}
		case <-sessionPoolTicker.C:
			// Periodic close idle conn trên session pool — không remove session,
			// chỉ giải phóng idle TCP/TLS native buffer. Chạy bất kể có register
			// đang chạy hay không (idempotent: nếu pool rỗng, return 0).
			//
			// Lifecycle ownership (Task 3): đọc global pointer KHÔNG có owning;
			// chỉ acts on pool nếu non-nil. Sau khi run kết thúc, runResources.Cleanup
			// đã CloseAll + nil global → ticker thấy nil → skip. Trong khi run đang
			// chạy, ticker thấy global = pool của run → CloseIdleConnsAll an toàn
			// (chỉ đóng IDLE conn, không abort active request, không drop session).
			closedIOS := 0
			closedAndroid := 0
			if ioshttpreg.SharedSessionPool != nil {
				closedIOS = ioshttpreg.SharedSessionPool.CloseIdleConnsAll()
			}
			if webandroidreg.SharedSessionPool != nil {
				closedAndroid = webandroidreg.SharedSessionPool.CloseIdleConnsAll()
			}
			// Close idle conn của banclone push transport — sau nhiều lần push
			// idle TCP/TLS có thể tích tụ trong transport pool.
			bancloneTransport.CloseIdleConnections()
			if closedIOS+closedAndroid > 0 {
				slog.Info("session pool idle cleanup",
					"ios_sessions", closedIOS,
					"android_sessions", closedAndroid)
			}
		}
	}
}

// DebugMemory trả về snapshot resource hiện tại — frontend có thể call để xem
// trạng thái RAM/goroutine/session pool. Dùng cho debug RAM leak khi chạy lâu.
//
// Để debug sâu hơn (heap profile / goroutine dump): set env DEBUG_PPROF=1 trước
// khi start app → pprof endpoint mở tại http://127.0.0.1:6060/debug/pprof/.
// Xem `debug.go` cho chi tiết enable/disable. KHÔNG bật mặc định production.
func (a *App) DebugMemory() map[string]interface{} {
	var m goruntime.MemStats
	goruntime.ReadMemStats(&m)

	iosSessions := 0
	if ioshttpreg.SharedSessionPool != nil {
		iosSessions = ioshttpreg.SharedSessionPool.Size()
	}
	androidSessions := 0
	if webandroidreg.SharedSessionPool != nil {
		androidSessions = webandroidreg.SharedSessionPool.Size()
	}

	a.accountsMu.RLock()
	accCount := len(a.accounts)
	a.accountsMu.RUnlock()

	// Run state — frontend dùng để hiển thị "Đang chạy/Đang dừng/Idle"
	a.verifyMu.Lock()
	verRunning := a.isRunning
	a.verifyMu.Unlock()
	regState := runState(a.registerState.Load())
	regRunning := regState == runStateRunning
	regStopping := regState == runStateStopping

	// LastGC — m.LastGC là nanoseconds since epoch; convert thành seconds-ago.
	// 0 nếu chưa có GC nào (app vừa start).
	lastGCAgoSec := uint64(0)
	if m.LastGC > 0 {
		nowNs := uint64(time.Now().UnixNano())
		if nowNs > m.LastGC {
			lastGCAgoSec = (nowNs - m.LastGC) / 1_000_000_000
		}
	}
	// LastGC pause: m.PauseNs là circular buffer 256 entries; index = (NumGC+255) % 256.
	lastPauseMs := uint64(0)
	if m.NumGC > 0 {
		lastPauseMs = m.PauseNs[(m.NumGC+255)%256] / 1_000_000
	}

	return map[string]interface{}{
		// Go runtime memory stats — granular cho Task 7 (heap idle/released = giá trị
		// có thể trả về OS qua FreeOSMemory; tăng = leak thật trong heap).
		"allocMB":        m.Alloc / 1024 / 1024,        // currently allocated heap (live objects)
		"heapAllocMB":    m.HeapAlloc / 1024 / 1024,    // bytes allocated from heap (== Alloc)
		"heapInuseMB":    m.HeapInuse / 1024 / 1024,    // bytes in in-use spans (incl free)
		"heapIdleMB":     m.HeapIdle / 1024 / 1024,     // bytes in idle spans (chưa trả về OS)
		"heapReleasedMB": m.HeapReleased / 1024 / 1024, // bytes đã trả về OS (subset của HeapIdle)
		"heapSysMB":      m.HeapSys / 1024 / 1024,      // total heap obtained from OS
		"sysMB":          m.Sys / 1024 / 1024,          // total bytes obtained from OS (heap+stack+other)
		"stackInuseMB":   m.StackInuse / 1024 / 1024,
		"numGC":          m.NumGC,                    // GC cycles run
		"numForcedGC":    m.NumForcedGC,              // forced GC count (manual runtime.GC calls)
		"lastGCAgoSec":   lastGCAgoSec,               // giây từ lần GC gần nhất (0 = vừa fire)
		"lastPauseMs":    lastPauseMs,                // pause của lần GC gần nhất
		"pauseTotalMs":   m.PauseTotalNs / 1_000_000, // tổng thời gian pause vì GC
		"gcCpuFraction":  m.GCCPUFraction,            // % CPU dành cho GC (0..1)
		"mallocs":        m.Mallocs,                  // tổng allocations từ start
		"frees":          m.Frees,                    // tổng frees từ start
		// Process-level (OS view) — khác Go heap, gồm WebView2 + native buffers
		"rssMB":  getProcessMemoryMB(), // RSS thực process
		"cpuPct": getProcessCPUPercent(),
		// Goroutine + concurrency
		"goroutines": goruntime.NumGoroutine(),
		"gomaxprocs": goruntime.GOMAXPROCS(0),
		// App-specific resources (ngoài Go heap — native HTTP/TLS buffers)
		"accountsInStore": accCount,
		"iosSessions":     iosSessions,
		"androidSessions": androidSessions,
		"transportPool":   proxy.TransportPoolStats(),
		"uploadPending":   uploadPendingInMem.Load(),
		// Run state — phục vụ cả frontend status bar lẫn debug
		"registerRunning":  regRunning,
		"registerStopping": regStopping,
		"verifyRunning":    verRunning,
		"verifyStopping":   a.verifyStopping.Load(),
		"uploadStopping":   a.uploadStopping.Load(),
		// Debug instrumentation flags — frontend có thểbạn đã hiển thị badge "PPROF ON"
		"pprofEnabled": debugPProfEnabled(),
	}
}

// ForceMemoryCleanup — frontend gọi để force cleanup ngay (idle conn + GC + FreeOS).
// Dùng khi user thấy RAM cao và muốn cleanup không cần restart app.
//
// Lifecycle ownership (Task 3): đọc global pointer chỉ để CloseIdleConnsAll (KHÔNG
// CloseAll/drop session). Run đang chạy → đóng idle conn của run đó (an toàn). Run
// đã end → runResources.Cleanup đã nil global → đọc nil → skip. KHÔNG bao giờ thay
// thế lifecycle Cleanup của run.
func (a *App) ForceMemoryCleanup() map[string]interface{} {
	closedIOS := 0
	closedAndroid := 0
	if ioshttpreg.SharedSessionPool != nil {
		closedIOS = ioshttpreg.SharedSessionPool.CloseIdleConnsAll()
	}
	if webandroidreg.SharedSessionPool != nil {
		closedAndroid = webandroidreg.SharedSessionPool.CloseIdleConnsAll()
	}
	// Close idle connections của banclone push transport — sau hàng nghìn lần push,
	// idle TCP/TLS state có thể tích tụ.
	bancloneTransport.CloseIdleConnections()

	var before goruntime.MemStats
	goruntime.ReadMemStats(&before)
	goruntime.GC()
	debug.FreeOSMemory()
	var after goruntime.MemStats
	goruntime.ReadMemStats(&after)

	freedMB := int64(before.HeapInuse-after.HeapInuse) / 1024 / 1024
	slog.Info("manual memory cleanup",
		"ios_sessions", closedIOS,
		"android_sessions", closedAndroid,
		"heap_freed_mb", freedMB,
		"heap_after_mb", after.HeapInuse/1024/1024)

	return map[string]interface{}{
		"iosSessionsClosed":     closedIOS,
		"androidSessionsClosed": closedAndroid,
		"heapBeforeMB":          before.HeapInuse / 1024 / 1024,
		"heapAfterMB":           after.HeapInuse / 1024 / 1024,
		"freedMB":               freedMB,
	}
}

// removeAccountLine xóa 1 dòng khỏi file (dùng chung cho cả CloneHV và file mode)
func (a *App) removeAccountLine(filePath, lineToRemove string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	out := make([]string, 0, len(lines))
	removed := false
	target := strings.TrimSpace(lineToRemove)
	for _, l := range lines {
		if !removed && strings.TrimSpace(l) == target {
			removed = true
			continue
		}
		out = append(out, l)
	}
	if err := os.WriteFile(filePath, []byte(strings.Join(out, "\n")), 0644); err != nil {
		slog.Warn("removeAccountLine: ghi file thất bại", "file", filePath, "err", err)
	}
}

// popAccountFromFolder đọc và xóa nguyên tử dòng tài khoản đầu tiên trong thư mục.
// Giữ removeLineMu toàn bộ thao tác đọc + ghi — thread-safe khi nhiều worker gọi đồng thời.
// Trả về ("", "", nil) khi không còn tài khoản nào.
// isVerifiableAccountFile trả về true nếu filename là file chứa account ĐẦY ĐỦ
// (uid|pass|cookie|token|...) mà split-verify có thể parse + verify được.
//
// Whitelist (dùng cho verify input):
//
//	SuccessReg.txt           — account reg thành công (main source)
//	SuccessReg_*.txt         — các variant (vd SuccessReg_S23.txt nếu split theo platform)
//
// Blacklist (bỏ qua — không phải account hoặc đã fail):
//
//	SuccessNVR_Phone.txt     — chỉ chứa phone number
//	SuccessNVR_Email.txt     — chỉ chứa email
//	Blocked.txt              — reg fail, không có cookie
//	Checkpoint.txt           — reg checkpoint
//	Unknown.txt              — reg/verify unknown
//	Live.txt/Die.txt         — output của verify (tránh loop)
//	SuccessVerify*.txt       — verify output (bao gồm SuccessVerifyUG.txt)
//	FbAppVersisonSuccess.txt — counter tracking
//	errordata.txt, RemainData.txt
//
// name: basename của file (không path), ví dụ "SuccessReg.txt".
func isVerifiableAccountFile(name string) bool {
	if !strings.HasSuffix(name, ".txt") {
		return false
	}
	return name == "SuccessReg.txt" || strings.HasPrefix(name, "SuccessReg_")
}

func (a *App) popAccountFromFolder(folderPath string) (line string, filePath string, err error) {
	a.removeLineMu.Lock()
	defer a.removeLineMu.Unlock()

	files, err := filepath.Glob(filepath.Join(folderPath, "*.txt"))
	if err != nil {
		return "", "", fmt.Errorf("đọc thư mục: %w", err)
	}
	for _, fp := range files {
		// Chỉ đọc file account đầy đủ — bỏ qua Phone/Email/UA/Blocked/etc.
		if !isVerifiableAccountFile(filepath.Base(fp)) {
			continue
		}
		content, readErr := os.ReadFile(fp)
		if readErr != nil {
			continue
		}
		rawLines := strings.Split(string(content), "\n")
		for i, l := range rawLines {
			l = strings.TrimSpace(l)
			if l == "" {
				continue
			}
			// Xóa dòng này ra khỏi file
			out := make([]string, 0, len(rawLines))
			for j, raw := range rawLines {
				if j != i {
					out = append(out, raw)
				}
			}
			if writeErr := os.WriteFile(fp, []byte(strings.Join(out, "\n")), 0644); writeErr != nil {
				slog.Warn("popAccountFromFolder: ghi file thất bại", "file", fp, "err", writeErr)
			}
			return l, fp, nil
		}
	}
	return "", "", nil
}

// popFromFile — đọc và xóa nguyên tử dòng đầu tiên từ một file cụ thể.
// Dùng cho Unknown retry: chỉ đọc từ Unknown.txt, không glob toàn thư mục.
// Trả về "" khi file không tồn tại hoặc hết dòng.
func (a *App) popFromFile(filePath string) string {
	a.removeLineMu.Lock()
	defer a.removeLineMu.Unlock()
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}
	rawLines := strings.Split(string(content), "\n")
	for i, l := range rawLines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		out := make([]string, 0, len(rawLines))
		for j, raw := range rawLines {
			if j != i {
				out = append(out, raw)
			}
		}
		_ = os.WriteFile(filePath, []byte(strings.Join(out, "\n")), 0644)
		return l
	}
	return ""
}

// startFolderWatcher khởi động goroutine poll thư mục nguồn mỗi 3 giây
// Khi tìm thấy account mới → tự động thêm vào memory + emit event lên frontend
func (a *App) startFolderWatcher(folderPath string) {
	// Dừng watcher cũ nếu có — bảo vệ bằng settingsMu
	a.settingsMu.Lock()
	if a.watcherCancel != nil {
		a.watcherCancel()
	}
	if folderPath == "" {
		a.settingsMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(a.ctx)
	a.watcherCancel = cancel
	a.settingsMu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("folder watcher panic recovered", "panic", r)
			}
		}()
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Không load từ folder khi user chọn API mode
				sett := a.LoadSettings()
				if isAPIMode(sett) {
					continue
				}
				result := a.scanAccountFolder(folderPath)
				if result.Imported > 0 {
					runtime.EventsEmit(a.ctx, "accounts:folder-updated", map[string]interface{}{
						"imported": result.Imported,
					})
				}
			}
		}
	}()
}

// === ACCOUNT SOURCE FOLDER ===

// SetAccountSourceFolder — lưu thư mục nguồn (streaming mode: không scan/watcher)
func (a *App) SetAccountSourceFolder(folderPath string) ImportResult {
	a.settingsMu.Lock()
	p := a.appSettings.GetActiveProfile()
	if p != nil {
		p.Account.FolderPath = folderPath
	}
	app := a.appSettings
	a.settingsMu.Unlock()
	if err := appsettings.Save("Config/Settings", app); err != nil {
		slog.Warn("SetAccountSourceFolder: lưu settings thất bại", "err", err)
	}

	// Sync vào general.json để LoadSettings() (dùng trong RunVerify) thấy được giá trị mới.
	const settingsDir = "Config/Settings"
	if b, err := os.ReadFile(filepath.Join(settingsDir, "general.json")); err == nil {
		var existing SettingsData
		if json.Unmarshal(b, &existing) == nil {
			existing.General.AccountSourcePath = folderPath
			if patched, err := json.MarshalIndent(existing, "", "  "); err == nil {
				_ = os.WriteFile(filepath.Join(settingsDir, "general.json"), patched, 0644)
			}
		}
	}

	// Sync vào interaction.json — VerifySourceFolderPath là alias của AccountSourcePath.
	// Đảm bảo 2 UI (Interaction Setup + Cài đặt chung) luôn hiển thị cùng 1 path.
	if b, err := os.ReadFile(filepath.Join(settingsDir, "interaction.json")); err == nil {
		var ic InteractionConfig
		if json.Unmarshal(b, &ic) == nil {
			ic.VerifySourceFolderPath = folderPath
			if patched, err := json.MarshalIndent(ic, "", "  "); err == nil {
				_ = os.WriteFile(filepath.Join(settingsDir, "interaction.json"), patched, 0644)
			}
		}
	}

	// Streaming mode: không scan folder vào grid, không start watcher.
	// Accounts sẽ được đọc lần lượt khi user bấm Chạy.
	return ImportResult{}
}

// GetAccountSourceFolder — trả về thư mục nguồn hiện tại
func (a *App) GetAppVersion() string {
	return AppVersion
}

func (a *App) GetAccountSourceFolder() string {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	if p := a.appSettings.GetActiveProfile(); p != nil {
		return p.Account.FolderPath
	}
	return ""
}

// RefreshAccountSource — scan lại thư mục nguồn, chỉ thêm account mới (dedup UID)
func (a *App) RefreshAccountSource() ImportResult {
	folderPath := a.GetAccountSourceFolder()
	if folderPath == "" {
		return ImportResult{Errors: []string{"Chưa cấu hình thư mục nguồn"}}
	}
	return a.scanAccountFolder(folderPath)
}

// scanAccountFolder đọc tất cả .txt trong thư mục, chỉ add account chưa có (dedup UID)
func (a *App) scanAccountFolder(folderPath string) ImportResult {
	files, err := filepath.Glob(filepath.Join(folderPath, "*.txt"))
	if err != nil {
		return ImportResult{Errors: []string{"Lỗi đọc thư mục: " + err.Error()}}
	}
	if len(files) == 0 {
		return ImportResult{Errors: []string{"Không tìm thấy file .txt nào trong thư mục"}}
	}

	// Build UID set để dedup — lock khi đọc
	a.accountsMu.Lock()
	existingUIDs := make(map[string]bool, len(a.accounts))
	for _, acc := range a.accounts {
		if acc.UID != "" {
			existingUIDs[acc.UID] = true
		}
	}
	a.accountsMu.Unlock()

	imported := 0
	errors := make([]string, 0)
	var newAccounts []Account

	for _, filePath := range files {
		content, err := os.ReadFile(filePath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Lỗi đọc %s: %v", filepath.Base(filePath), err))
			continue
		}
		lines := splitLines(string(content))
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			acc := autoDetectAccount(line)
			if acc.UID == "" {
				errors = append(errors, fmt.Sprintf("%s dòng %d: không nhận diện UID", filepath.Base(filePath), i+1))
				continue
			}
			if existingUIDs[acc.UID] {
				continue // skip duplicate
			}
			existingUIDs[acc.UID] = true
			acc.FullData = line
			acc.Status = "new"
			acc.ImportTime = time.Now().Format("2006/01/02 15:04")
			acc.SourceCode = filepath.Base(filePath)
			newAccounts = append(newAccounts, acc)
			imported++
		}
	}

	// Append dưới lock — re-check UID để tránh TOCTOU race với folder watcher
	if len(newAccounts) > 0 {
		a.accountsMu.Lock()
		// Re-build live UID set để catch duplicate được thêm sau khi unlock lần đầu
		liveUIDs := make(map[string]bool, len(a.accounts))
		for _, acc := range a.accounts {
			if acc.UID != "" {
				liveUIDs[acc.UID] = true
			}
		}
		base := len(a.accounts)
		added := 0
		for i := range newAccounts {
			if liveUIDs[newAccounts[i].UID] {
				continue // race: UID đã được thêm bởi goroutine khác trong lúc đọc file
			}
			newAccounts[i].ID = base + added + 1
			a.accounts = append(a.accounts, newAccounts[i])
			added++
		}
		a.accountsMu.Unlock()
		imported = added
	}

	return ImportResult{Imported: imported, Errors: errors}
}

// === FILE DIALOG ===

// OpenTextFileDialog mở dialog chọn file text, trả về nội dung file
func (a *App) OpenTextFileDialog() string {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Chọn file tài khoản",
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil || file == "" {
		return ""
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	return string(data)
}

// OpenFileDialogPath mở dialog chọn file text, trả về PATH (không đọc nội dung)
func (a *App) OpenFileDialogPath() string {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Chọn file User Agent",
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files (*.txt)", Pattern: "*.txt"},
			{DisplayName: "All Files (*.*)", Pattern: "*.*"},
		},
	})
	if err != nil {
		return ""
	}
	return file
}

// ReadTextFile đọc nội dung file text từ path, trả về nội dung hoặc "" nếu lỗi
func (a *App) ReadTextFile(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return ""
	}
	return string(data)
}

// ValidatePath kiểm tra đường dẫn có tồn tại và là thư mục không
// Trả về "" nếu hợp lệ, thông báo lỗi nếu không hợp lệ
func (a *App) ValidatePath(path string) string {
	if path == "" {
		return "Chưa chọn thư mục"
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "Thư mục không tồn tại: " + path
	}
	if err != nil {
		return "Lỗi kiểm tra đường dẫn: " + err.Error()
	}
	if !info.IsDir() {
		return "Đường dẫn không phải thư mục: " + path
	}
	return ""
}

// ValidateFilePath kiểm tra path là FILE tồn tại (ngược với ValidatePath chỉ accept folder).
// Dùng cho AccountSource="file" mode — user pick 1 file .txt cụ thể.
func (a *App) ValidateFilePath(path string) string {
	if path == "" {
		return "Chưa chọn file"
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "File không tồn tại: " + path
	}
	if err != nil {
		return "Lỗi kiểm tra file: " + err.Error()
	}
	if info.IsDir() {
		return "Đường dẫn là thư mục, cần chọn file: " + path
	}
	return ""
}

// OpenFolderDialog mở dialog chọn thư mục, trả về đường dẫn
func (a *App) OpenFolderDialog() string {
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Chọn thư mục lưu",
	})
	if err != nil {
		return ""
	}
	return folder
}

// OpenFolderInExplorer mở thư mục trong Windows Explorer.
// Sanitize path: phải là absolute, normalize, và tồn tại — tránh command injection
// qua UNC path (\\attacker\share) hoặc explorer-flag arguments (/root,/e,...).
func (a *App) OpenFolderInExplorer(path string) string {
	if path == "" {
		return "Chưa chọn thư mục"
	}
	clean := filepath.Clean(path)
	if !filepath.IsAbs(clean) {
		return "Đường dẫn phải là absolute"
	}
	// Reject UNC path (\\server\share) — không phải thư mục local an toàn để mở
	if strings.HasPrefix(clean, `\\`) {
		return "Đường dẫn UNC không được hỗ trợ"
	}
	// Reject path bắt đầu bằng "/" hoặc "-" trên Windows (có thể bị parse như flag)
	if strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "-") {
		return "Đường dẫn không hợp lệ"
	}
	// Verify path tồn tại + là directory trước khi exec
	info, err := os.Stat(clean)
	if err != nil {
		return "Thư mục không tồn tại: " + clean
	}
	if !info.IsDir() {
		return "Đường dẫn không phải là thư mục: " + clean
	}
	if err := exec.Command("explorer", clean).Start(); err != nil {
		return fmt.Errorf("không mở được thư mục: %w", err).Error()
	}
	return ""
}

// OpenConfigFolder mở thư mục Config trong Windows Explorer.
func (a *App) OpenConfigFolder() string {
	return a.OpenFolderInExplorer(filepath.Join(appDataDir(), "Config"))
}

// GetDefaultResultPath trả về đường dẫn thư mục kết quả mặc định sẽ được dùng
// nếu user chưa cấu hình ResultFolderPath qua UI.
//
// Logic fallback giống defaultResultFolder() (dùng khi Start Run):
//  1. {exe_dir}/KetQua/ — tạo nếu có quyền ghi
//  2. Fallback ~/Documents/HVR/KetQua/ nếu exe_dir read-only
//
// Gọi từ frontend lúc mount để:
//   - Hiển thị placeholder ở field "Result folder"
//   - Mở đúng folder khi user bấm nút "Mở thư mục" trước lần Start đầu tiên
//     (thay vì bật dialog chọn folder)
//
// Hàm này KHÔNG tạo thư mục — chỉ resolve path. Folder sẽ được tạo lần đầu Run.
func (a *App) GetDefaultResultPath() string {
	return defaultResultFolder()
}

// === ACCOUNT CRUD ===

// ListAccounts trả về danh sách accounts đã lọc + tổng số kết quả.
// filter.Keyword: lọc theo UID/email/tên/note (case-insensitive)
// filter.Status: lọc theo trạng thái ("live", "die", "checkpoint", "new", "")
// filter.CategoryID: lọc theo category (nil = tất cả)
// filter.SortBy / filter.SortDir: chưa implement (dành cho tương lai)
// Merge realtime activity từ activityCache (sync.Map) mà không cần lock a.accounts.
func (a *App) ListAccounts(filter AccountFilter) AccountListResult {
	filtered := make([]Account, 0)
	kw := strings.ToLower(filter.Keyword)

	a.accountsMu.RLock()
	snapshot := make([]Account, len(a.accounts))
	copy(snapshot, a.accounts)
	a.accountsMu.RUnlock()

	for _, acc := range snapshot {
		// Merge realtime activity từ lock-free cache (cập nhật bởi onStatus hot path)
		if v, ok := a.activityCache.Load(acc.ID); ok {
			if s, ok := v.(string); ok {
				acc.Activity = s
			}
		}
		if kw != "" {
			match := strings.Contains(strings.ToLower(acc.UID), kw) ||
				strings.Contains(strings.ToLower(acc.Email), kw) ||
				strings.Contains(strings.ToLower(acc.FullName), kw) ||
				strings.Contains(strings.ToLower(acc.Note), kw)
			if !match {
				continue
			}
		}
		if filter.Status != "" && acc.Status != filter.Status {
			continue
		}
		if filter.CategoryID != nil && (acc.CategoryID == nil || *acc.CategoryID != *filter.CategoryID) {
			continue
		}
		filtered = append(filtered, acc)
	}

	return AccountListResult{Items: filtered, Total: len(filtered)}
}

// GetAccount trả về một account theo ID.
// id: ID nội bộ của account (auto-increment khi import).
// Thread-safe: dùng RLock vì không thay đổi slice.
// Trả về nil + error nếu ID không tồn tại.
func (a *App) GetAccount(id int) (*Account, error) {
	a.accountsMu.RLock()
	defer a.accountsMu.RUnlock()
	for _, acc := range a.accounts {
		if acc.ID == id {
			cp := acc
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("account ID %d không tồn tại", id)
}

// ImportAccounts — auto-detect fields từ dữ liệu giống WeBM frmAddAccount.btnSaveV2_Click
// Tự động nhận diện: UID, Password, Cookie (c_user=), Token (EAA), 2FA (32 chars hex),
// Email (@), PassMail, MailRecovery, Phone (digits 8-15), Client_ID (GUID), Refresh_token (M.xxx)
// LoadAccountsFromFile đọc 1 file .txt → parse accounts → thay thế store hiện tại.
// Khác với ImportAccounts (append): hàm này CLEAR store trước khi nạp.
// Dùng cho AccountSource="file" — user chọn file, load toàn bộ accounts vào grid, tick chọn.
//
// Emit event "accounts:folder-updated" sau khi load → AccountsPage auto refresh grid
// hiển thị accounts mới mà không cần user reload thủ công.
func (a *App) LoadAccountsFromFile(filePath string) ImportResult {
	// Chặn load khi verify/register đang chạy — clobber a.accounts giữa chừng sẽ làm
	// slot rows biến mất khỏi grid và worker đọc được dữ liệu sai.
	a.verifyMu.Lock()
	verifyRunning := a.isRunning
	a.verifyMu.Unlock()
	// Cấm load file khi register ở bất kỳ state nào ngoài idle (running OR stopping):
	// stopping vẫn còn workers đang quẫy → đụng vào a.accounts gây inconsistency.
	regBusy := runState(a.registerState.Load()) != runStateIdle
	if verifyRunning || regBusy {
		return ImportResult{Errors: []string{"Đang chạy verify/register — vui lòng dừng trước khi load file mới"}}
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return ImportResult{Errors: []string{fmt.Sprintf("Không đọc được file: %v", err)}}
	}
	// Clear store trước khi load — file mode là replace, không append.
	a.accountsMu.Lock()
	a.accounts = nil
	a.accountsMu.Unlock()
	// Lưu path để OnAccountDone xóa dòng khỏi file gốc khi verify = live.
	a.sourceFilePathMu.Lock()
	a.sourceFilePath = filePath
	a.sourceFilePathMu.Unlock()
	// Persist accountSourcePath cho verify flow sau nhận path từ settings.
	existing := a.LoadSettings()
	existing.General.AccountSource = "file"
	existing.General.AccountSourcePath = filePath
	_ = a.SaveSettings(existing)
	result := a.ImportAccounts(string(data))
	// Emit event để AccountsPage refresh grid realtime.
	runtime.EventsEmit(a.ctx, "accounts:folder-updated", map[string]interface{}{
		"imported": result.Imported,
		"source":   "file",
		"path":     filePath,
	})
	return result
}

func (a *App) ImportAccounts(data string) ImportResult {
	lines := splitLines(data)
	imported := 0
	errors := make([]string, 0)

	// Parse trước, không cần lock
	var newAccounts []Account
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		acc := autoDetectAccount(line)
		if acc.UID == "" {
			errors = append(errors, fmt.Sprintf("Dòng %d: không nhận diện được UID", i+1))
			continue
		}
		acc.FullData = line
		acc.Status = "new"
		acc.ImportTime = time.Now().Format("2006/01/02 15:04")
		newAccounts = append(newAccounts, acc)
	}

	// Lock chỉ khi ghi vào a.accounts
	a.accountsMu.Lock()
	for i := range newAccounts {
		newAccounts[i].ID = len(a.accounts) + 1
		newAccounts[i].SourceCode = fmt.Sprintf("Import #%d", newAccounts[i].ID)
		a.accounts = append(a.accounts, newAccounts[i])
		imported++
	}
	a.accountsMu.Unlock()

	return ImportResult{Imported: imported, Errors: errors}
}

// autoDetectAccount — tự động nhận diện fields từ chuỗi phân cách |
// Mapping từ WeBM frmAddAccount.cs btnSaveV2_Click lines 256-383
func autoDetectAccount(line string) Account {
	fields := strings.Split(line, "|")
	acc := Account{}

	hasUID := false
	hasPassword := false
	hasEmail := false
	hasRecoveryEmail := false

	for i, raw := range fields {
		f := strings.TrimSpace(raw)
		if f == "" {
			continue
		}

		// 0a. iOS partial reg tokens — SRN:<srnonce> và SCUID:<sessionlessCryptedUID>.
		// Được append bởi formatRegResultLine cho iOS accounts để file-based verify hoạt động.
		if strings.HasPrefix(f, "SRN:") {
			acc.Srnonce = f[4:]
			continue
		}
		if strings.HasPrefix(f, "SCUID:") {
			acc.SessionlessCryptedUID = f[6:]
			continue
		}

		// 0b. EmailMeta — TempMail provider creds, format "MM:<base64-json>".
		// Append làm cột cuối khi save (xem internal/result/format.go FormatReg).
		// Loader cũ skip vì không match pattern UID/email/cookie nào.
		if strings.HasPrefix(f, "MM:") {
			if meta := resultpkg.ParseEmailMetaFromLine(f); meta != "" {
				acc.EmailMeta = meta
			}
			continue
		}

		// 1. Client_ID — GUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
		if isGUID(f) {
			acc.Note = f // Tạm lưu vào note, hoặc field riêng nếu cần
			continue
		}

		// 2. Refresh_token — bắt đầu bằng "M." và dài > 50
		if strings.HasPrefix(f, "M.") && len(f) > 50 {
			continue // Refresh token — không dùng cho verify
		}

		// 3. UID — field đầu tiên, ngắn < 50 ký tự
		if i == 0 && len(f) < 50 && !hasUID {
			acc.UID = f
			hasUID = true
			continue
		}

		// 4. 2FA — 32 ký tự hex (chứa cả chữ và số, không có dấu đặc biệt)
		cleaned2fa := strings.ReplaceAll(f, " ", "")
		if len(cleaned2fa) == 32 && isAlphaNumeric(cleaned2fa) && hasLetterAndDigit(cleaned2fa) {
			acc.Twofa = cleaned2fa
			continue
		}

		// 5. Token — bắt đầu bằng "EAA"
		if strings.HasPrefix(f, "EAA") {
			acc.Token = f
			continue
		}

		// 6. Cookie — chứa "c_user=" hoặc "ds_user_id="
		if strings.Contains(f, "c_user=") || strings.Contains(f, "ds_user_id=") {
			acc.Cookie = f
			// Extract UID từ cookie nếu chưa có
			if acc.UID == "" {
				acc.UID = extractCUserFromCookie(f)
				hasUID = true
			}
			continue
		}

		// 7. Password — ngay sau UID, chưa nhận diện password
		if hasUID && !hasPassword && !strings.Contains(f, "@") && !strings.Contains(f, "c_user=") {
			acc.Password = f
			hasPassword = true
			continue
		}

		// 8. Email — chứa @ và có dạng email
		if strings.Contains(f, "@") && strings.Contains(f, ".") {
			if !hasEmail {
				acc.Email = f
				hasEmail = true
				continue
			} else if !hasRecoveryEmail && f != acc.Email {
				acc.MailRecovery = f
				hasRecoveryEmail = true
				continue
			}
		}

		// 9. Pass Email — ngay sau email, chưa nhận diện
		if hasEmail && acc.PassMail == "" && !strings.Contains(f, "@") && !strings.Contains(f, "c_user=") && !strings.HasPrefix(f, "EAA") {
			acc.PassMail = f
			continue
		}

		// 10. Phone — toàn số, 8-15 ký tự
		if isAllDigits(f) && len(f) >= 8 && len(f) <= 15 {
			acc.Phone = f
			continue
		}
	}

	return acc
}

// isGUID kiểm tra string có đúng định dạng GUID xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx không.
// s: chuỗi cần kiểm tra (ví dụ "550e8400-e29b-41d4-a716-446655440000").
// Dùng để phân biệt Client_ID (GUID) với các field khác khi auto-detect account.
func isGUID(s string) bool {
	// xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// isAlphaNumeric kiểm tra string chỉ chứa chữ cái (a-z, A-Z) và chữ số (0-9).
// s: chuỗi cần kiểm tra.
// Dùng bước đầu để lọc ứng viên 2FA key (32 ký tự alphanumeric).
func isAlphaNumeric(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	return true
}

// hasLetterAndDigit kiểm tra string có chứa ít nhất 1 chữ cái VÀ 1 chữ số không.
// s: chuỗi đã qua isAlphaNumeric (chỉ còn [a-zA-Z0-9]).
// Lý do: 2FA key phải là mix của cả chữ và số (loại bỏ các field toàn số như phone, toàn chữ như password đơn giản).
func hasLetterAndDigit(s string) bool {
	hasLetter := false
	hasDigit := false
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			hasLetter = true
		}
		if c >= '0' && c <= '9' {
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

// isAllDigits kiểm tra string chỉ chứa chữ số và không rỗng.
// s: chuỗi cần kiểm tra.
// Dùng để nhận diện số điện thoại: toàn số, độ dài 8-15 ký tự.
func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

// DeleteAccounts xóa các accounts khỏi memory theo danh sách ID.
// ids: slice các ID cần xóa (từ frontend khi user chọn hàng rồi bấm Delete).
// Thread-safe: dùng Lock vì thay đổi slice a.accounts.
// Trả về số lượng thực tế đã xóa (có thể < len(ids) nếu một số ID không tồn tại).
func (a *App) DeleteAccounts(ids []int) DeleteResult {
	idSet := make(map[int]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	a.accountsMu.Lock()
	remaining := make([]Account, 0, len(a.accounts))
	deleted := 0
	for _, acc := range a.accounts {
		if idSet[acc.ID] {
			deleted++
		} else {
			remaining = append(remaining, acc)
		}
	}
	a.accounts = remaining
	a.accountsMu.Unlock()
	return DeleteResult{Deleted: deleted}
}

// === VERIFY ===

type VerifyRunConfig struct {
	AccountIDs []int                 `json:"accountIds"`
	MaxThreads int                   `json:"maxThreads"`
	VerifyCfg  instagram.VerifyConfig `json:"verifyConfig"`
	OutputPath string                `json:"outputPath"`
	Proxy      string                `json:"proxy"` // Proxy chung từ settings
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
) {
	if writer == nil || writer.Root() == "" {
		return
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

// === SETTINGS ===

// GeneralConfig cấu hình chung — mapping từ frontend GeneralConfig
type GeneralConfig struct {
	ThreadRequest     int               `json:"threadRequest"`
	DelayRequest      int               `json:"delayRequest"`
	DelayThread       int               `json:"delayThread"`
	ApiCheckIp        int               `json:"apiCheckIp"`
	ThreadCheckInfo   int               `json:"threadCheckInfo"`
	LoginPlatform     string            `json:"loginPlatform"`
	LoginMethod       int               `json:"loginMethod"`
	SaveRunColumn     bool              `json:"saveRunColumn"`
	BackupDB          bool              `json:"backupDB"`
	CloseAfterDone    bool              `json:"closeAfterDone"`
	AccountSourcePath string            `json:"accountSourcePath"`
	AccountSource     string            `json:"accountSource"` // "folder" | "api"
	CloneHVUsername   string            `json:"cloneHvUsername"`
	CloneHVPassword   string            `json:"cloneHvPassword"`
	CloneHVProductID  string            `json:"cloneHvProductId"`
	CloneHVAmount     int               `json:"cloneHvAmount"`
	CaptchaProvider   string            `json:"captchaProvider"`
	CaptchaKeys       map[string]string `json:"captchaKeys"`
	IpProvider        string            `json:"ipProvider"`
	CheckIpBeforeRun  bool              `json:"checkIpBeforeRun"`
	DelayChangeIp     int               `json:"delayChangeIp"`
	// Locale & Device Fake
	LocaleFake    string `json:"localeFake"`
	DeepFakeInApi bool   `json:"deepFakeInApi"`
	// Cookie Initial
	CookieUse        bool   `json:"cookieUse"`
	CookieLimit      bool   `json:"cookieLimit"`
	CookieLimitCount int    `json:"cookieLimitCount"`
	CookieMode       string `json:"cookieMode"`
	// UA Custom
	UaAddSpecs   bool `json:"uaAddSpecs"`
	UaBuildFile  bool `json:"uaBuildFile"`
	UaCustomType int  `json:"uaCustomType"`
	// Sim Network
	SimNetworkMode string `json:"simNetworkMode"`
	SimNetworkType string `json:"simNetworkType"`
}

// IpConfig cấu hình IP — mapping từ frontend IpConfig
type IpConfig struct {
	ProxyList               string `json:"proxyList"`
	ProxyStickyList         string `json:"proxyStickyList"`
	ProxyActiveTab          string `json:"proxyActiveTab"` // "standard" | "sticky" — tab đang chọn trên UI
	ProxyType               string `json:"proxyType"`
	FptKeys                 string `json:"fptKeys"`
	XproxyServiceUrl        string `json:"xproxyServiceUrl"`
	XproxyType              string `json:"xproxyType"`
	XproxyList              string `json:"xproxyList"`
	XproxyThreadPerIp       int    `json:"xproxyThreadPerIp"`
	XproxyRunType           string `json:"xproxyRunType"`
	TinsoftKeys             string `json:"tinsoftKeys"`
	TinsoftThreadPerIp      int    `json:"tinsoftThreadPerIp"`
	ShoplikeKeys            string `json:"shoplikeKeys"`
	ShoplikeThreadPerIp     int    `json:"shoplikeThreadPerIp"`
	NetproxyKeys            string `json:"netproxyKeys"`
	NetproxyThreadPerIp     int    `json:"netproxyThreadPerIp"`
	MinproxyKeys            string `json:"minproxyKeys"`
	MinproxyThreadPerIp     int    `json:"minproxyThreadPerIp"`
	NetproxyGbKey           string `json:"netproxyGbKey"`
	ProxyPopularKeys        string `json:"proxyPopularKeys"`
	ProxyPopularThreadPerIp int    `json:"proxyPopularThreadPerIp"`
	ProxyPopularAccessToken string `json:"proxyPopularAccessToken"`
	ProxyFarmKeys           string `json:"proxyFarmKeys"`
	ProxyFarmThreadPerIp    int    `json:"proxyFarmThreadPerIp"`
	ProxyFarmAccessToken    string `json:"proxyFarmAccessToken"`
	// Proxy riêng cho Reg
	UseVerifyProxyForReg bool   `json:"useVerifyProxyForReg"`
	RegIpProvider        string `json:"regIpProvider"`
	RegProxyList         string `json:"regProxyList"`
	RegProxyStickyList   string `json:"regProxyStickyList"`
	RegProxyActiveTab    string `json:"regProxyActiveTab"`
	RegProxyType         string `json:"regProxyType"`
	// Retry & Delay
	ProxyRetry   int `json:"proxyRetry"`   // số lần retry khi proxy lỗi (0 = không retry)
	ProxyDelayMs int `json:"proxyDelayMs"` // delay ms trước khi đổi proxy
}

// activeProxyList trả về danh sách proxy verify.
// Session proxy (có _session-, -zone-, sid-) tự nhận diện per-line — không cần tab.
func activeProxyList(ip IpConfig) string {
	return ip.ProxyList
}

// activeRegProxyList trả về danh sách proxy reg.
func activeRegProxyList(ip IpConfig) string {
	return ip.RegProxyList
}

// SettingsData gói cả GeneralConfig + IpConfig
type SettingsData struct {
	General GeneralConfig `json:"general"`
	Ip      IpConfig      `json:"ip"`
}

// SaveSettings lưu cài đặt chung vào active profile và general.json
func (a *App) SaveSettings(data SettingsData) string {
	const settingsDir = "Config/Settings"

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "Lỗi marshal: " + err.Error()
	}
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return "Lỗi tạo thư mục: " + err.Error()
	}

	// 1. Sync vào active profile (profile-aware)
	var ls adapter.LegacySettingsData
	if err := json.Unmarshal(b, &ls); err != nil {
		slog.Warn("SaveSettings: sync appSettings thất bại", "err", err)
	}
	a.settingsMu.Lock()
	if p := a.appSettings.GetActiveProfile(); p != nil {
		adapter.ApplySettingsToProfile(p, &a.appSettings.Global, ls)
	}
	// Dừng watcher cũ nếu đang chạy
	if a.watcherCancel != nil {
		a.watcherCancel()
		a.watcherCancel = nil
	}
	app := a.appSettings
	a.settingsMu.Unlock()

	// 2. Persist AppSettings (chứa profile vừa cập nhật)
	if err := appsettings.Save(settingsDir, app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}

	// 3. Ghi general.json để backward-compat — fail ở đây nghĩa là settings không persist,
	// PHẢI return error rõ ràng để frontend không hiển thị "Đã lưu" giả.
	// 0600 vì file chứa proxy creds, captcha keys → chỉ owner đọc được.
	if err := os.WriteFile(filepath.Join(settingsDir, "general.json"), b, 0600); err != nil {
		slog.Error("SaveSettings: ghi general.json thất bại", "err", err)
		return "Lỗi ghi general.json: " + err.Error()
	}

	// 4. Sync AccountSourcePath → interaction.json VerifySourceFolderPath.
	// 2 field là 1 giá trị duy nhất — user edit ở General sẽ hiện ở Interaction.
	// Sync fail không fatal (general.json đã lưu) nhưng phải log để debug khi 2 UI hiện path khác nhau.
	interactionPath := filepath.Join(settingsDir, "interaction.json")
	if ib, err := os.ReadFile(interactionPath); err == nil {
		var ic InteractionConfig
		if uerr := json.Unmarshal(ib, &ic); uerr == nil {
			ic.VerifySourceFolderPath = data.General.AccountSourcePath
			if patched, merr := json.MarshalIndent(ic, "", "  "); merr == nil {
				if werr := os.WriteFile(interactionPath, patched, 0600); werr != nil {
					slog.Warn("SaveSettings: sync interaction.json (write) thất bại", "err", werr)
				}
			} else {
				slog.Warn("SaveSettings: sync interaction.json (marshal) thất bại", "err", merr)
			}
		} else {
			slog.Warn("SaveSettings: sync interaction.json (unmarshal) thất bại", "err", uerr)
		}
	} else if !os.IsNotExist(err) {
		slog.Warn("SaveSettings: đọc interaction.json thất bại", "err", err)
	}

	// 5. Invalidate proxy cache — settings có thể chứa IpProvider/keys → force recreate mgr
	a.InvalidateProxyCache()

	return "OK"
}

// LoadSettings đọc cài đặt chung từ general.json, fallback a.appSettings.
// First-run (file chưa tồn tại) → áp dụng full defaults (bool + string).
// Subsequent loads → chỉ fill string fields rỗng, giữ user's bool choice.
func (a *App) LoadSettings() SettingsData {
	generalPath := filepath.Join("Config/Settings", "general.json")

	// Đọc từ general.json nếu tồn tại — user đã save 1 lần
	if b, err := os.ReadFile(generalPath); err == nil {
		var data SettingsData
		if json.Unmarshal(b, &data) == nil {
			applyGeneralStringDefaults(&data.General) // chỉ fill string rỗng
			applyIpStringDefaults(&data.Ip)           // normalize proxyType rỗng → "http"
			return data
		}
	}

	// First-run: file chưa tồn tại → áp dụng FULL defaults.
	a.settingsMu.RLock()
	ls := adapter.ToLegacySettings(a.appSettings)
	a.settingsMu.RUnlock()
	var data SettingsData
	if b, err := json.Marshal(ls); err == nil {
		if err := json.Unmarshal(b, &data); err != nil {
			slog.Warn("LoadSettings fallback: unmarshal thất bại", "err", err)
		}
	}
	applyGeneralFullDefaults(&data.General) // first-run: set cả bool defaults
	return data
}

// applyIpStringDefaults fill các proxy type rỗng về mặc định "http".
// Phải gọi sau khi unmarshal để tránh radio button không được tích khi load lại.
func applyIpStringDefaults(ip *IpConfig) {
	if ip == nil {
		return
	}
	if ip.ProxyType == "" {
		ip.ProxyType = "http"
	}
	if ip.XproxyType == "" {
		ip.XproxyType = "http"
	}
	if ip.RegProxyType == "" {
		ip.RegProxyType = "http"
	}
}

// applyGeneralStringDefaults chỉ fill string fields rỗng — giữ user's bool choice.
// Dùng khi general.json đã tồn tại (user đã save 1 lần).
func applyGeneralStringDefaults(c *GeneralConfig) {
	if c == nil {
		return
	}
	if strings.TrimSpace(c.LocaleFake) == "" {
		c.LocaleFake = "match-ip"
	}
	if strings.TrimSpace(c.SimNetworkMode) == "" {
		c.SimNetworkMode = "match-ip"
	}
	if strings.TrimSpace(c.SimNetworkType) == "" {
		c.SimNetworkType = "LTE"
	}
	if strings.TrimSpace(c.LoginPlatform) == "" {
		c.LoginPlatform = "facebook"
	}
	// LoginMethod: với Facebook, 0 không match option (chỉ có value=6 "Cookie mobile").
	// Nếu platform facebook và loginMethod chưa valid → set 6.
	if c.LoginPlatform == "facebook" && c.LoginMethod == 0 {
		c.LoginMethod = 6
	}
}

// applyGeneralFullDefaults áp dụng full defaults (bool + string) cho first-run.
// Chuẩn C# defaults — user khởi động app lần đầu tick sẵn các option quan trọng.
func applyGeneralFullDefaults(c *GeneralConfig) {
	if c == nil {
		return
	}
	applyGeneralStringDefaults(c)
	c.DeepFakeInApi = true       // C#: deep locale trong API payload
	c.UaAddSpecs = true          // C#: virtual spec Android
	c.LoginPlatform = "facebook" // default platform
	c.LoginMethod = 6            // "Cookie mobile" — facebook chỉ có 1 option (value=6)
	// KeepIPSuccess, UaBuildFile giữ zero (user tuỳ chọn)
}

// getSharedProxyManager trả về proxy.Manager dùng chung giữa REG và VER.
//
// Fast path (99% case khi đang chạy batch): RLock + version check, return cached.
// Tránh LoadSettings() disk I/O mỗi worker call — 50 workers × 5ms = 250ms bottleneck.
//
// Slow path (settings thay đổi): LoadSettings() + rebuild. Trigger bằng atomic
// proxyConfigVersion++ trong SaveSettings/SaveInteractionConfig/SaveIpConfig.
func (a *App) getSharedProxyManager() *proxy.Manager {
	currentVer := a.proxyConfigVersion.Load()

	// ── Fast path: cached + version khớp ───────────────────────────────────
	a.sharedProxyMgrMu.RLock()
	if a.sharedProxyMgr != nil && a.sharedProxyMgrVersion == currentVer {
		mgr := a.sharedProxyMgr
		a.sharedProxyMgrMu.RUnlock()
		return mgr
	}
	a.sharedProxyMgrMu.RUnlock()

	// ── Slow path: cần tạo mới ─────────────────────────────────────────────
	a.sharedProxyMgrMu.Lock()
	defer a.sharedProxyMgrMu.Unlock()

	// Re-check sau khi lấy write lock (double-check pattern) — có thể goroutine
	// khác đã tạo xong trong lúc ta chờ lock.
	currentVer = a.proxyConfigVersion.Load()
	if a.sharedProxyMgr != nil && a.sharedProxyMgrVersion == currentVer {
		return a.sharedProxyMgr
	}

	s := a.LoadSettings()
	key := s.General.IpProvider + "|" + s.Ip.ShoplikeKeys + "|" + s.Ip.TinsoftKeys + "|" +
		s.Ip.NetproxyKeys + "|" + s.Ip.MinproxyKeys + "|" + s.Ip.ProxyFarmKeys + "|" + activeProxyList(s.Ip)

	// Key cũng khớp → chỉ version bump hình thức, dùng lại instance.
	if a.sharedProxyMgr != nil && a.sharedProxyMgrKey == key {
		a.sharedProxyMgrVersion = currentVer
		return a.sharedProxyMgr
	}

	slog.Info("getSharedProxyManager: tạo mới", "provider", s.General.IpProvider, "shoplikeKeys_len", len(s.Ip.ShoplikeKeys))
	a.sharedProxyMgr = proxy.NewManager(proxy.ManagerConfig{
		Provider:         s.General.IpProvider,
		ProxyList:        activeProxyList(s.Ip),
		TinsoftKeys:      s.Ip.TinsoftKeys,
		TinsoftThreads:   s.Ip.TinsoftThreadPerIp,
		ShoplikeKeys:     s.Ip.ShoplikeKeys,
		ShoplikeThreads:  s.Ip.ShoplikeThreadPerIp,
		NetproxyKeys:     s.Ip.NetproxyKeys,
		NetproxyThreads:  s.Ip.NetproxyThreadPerIp,
		MinproxyKeys:     s.Ip.MinproxyKeys,
		MinproxyThreads:  s.Ip.MinproxyThreadPerIp,
		ProxyfarmKeys:    s.Ip.ProxyFarmKeys,
		ProxyfarmThreads: s.Ip.ProxyFarmThreadPerIp,
	})
	a.sharedProxyMgrKey = key
	a.sharedProxyMgrVersion = currentVer
	return a.sharedProxyMgr
}

// getRegProxyManager trả về proxy.Manager cho luồng Register.
// Nếu useVerifyProxyForReg=true → trả về sharedProxyMgr của verify.
// Ngược lại → tạo/cache manager riêng từ reg proxy config.
func (a *App) getRegProxyManager() *proxy.Manager {
	s := a.LoadSettings()
	if s.Ip.UseVerifyProxyForReg || s.Ip.RegIpProvider == "" || s.Ip.RegIpProvider == "none" {
		return a.getSharedProxyManager()
	}

	currentVer := a.proxyConfigVersion.Load()

	a.regProxyMgrMu.RLock()
	if a.regProxyMgr != nil && a.regProxyMgrVersion == currentVer {
		mgr := a.regProxyMgr
		a.regProxyMgrMu.RUnlock()
		return mgr
	}
	a.regProxyMgrMu.RUnlock()

	a.regProxyMgrMu.Lock()
	defer a.regProxyMgrMu.Unlock()

	currentVer = a.proxyConfigVersion.Load()
	if a.regProxyMgr != nil && a.regProxyMgrVersion == currentVer {
		return a.regProxyMgr
	}

	key := s.Ip.RegIpProvider + "|" + s.Ip.ShoplikeKeys + "|" + s.Ip.TinsoftKeys + "|" +
		s.Ip.NetproxyKeys + "|" + s.Ip.MinproxyKeys + "|" + s.Ip.ProxyFarmKeys + "|" + activeRegProxyList(s.Ip)

	if a.regProxyMgr != nil && a.regProxyMgrKey == key {
		a.regProxyMgrVersion = currentVer
		return a.regProxyMgr
	}

	slog.Info("getRegProxyManager: tạo mới", "provider", s.Ip.RegIpProvider)
	a.regProxyMgr = proxy.NewManager(proxy.ManagerConfig{
		Provider:         s.Ip.RegIpProvider,
		ProxyList:        activeRegProxyList(s.Ip),
		TinsoftKeys:      s.Ip.TinsoftKeys,
		TinsoftThreads:   s.Ip.TinsoftThreadPerIp,
		ShoplikeKeys:     s.Ip.ShoplikeKeys,
		ShoplikeThreads:  s.Ip.ShoplikeThreadPerIp,
		NetproxyKeys:     s.Ip.NetproxyKeys,
		NetproxyThreads:  s.Ip.NetproxyThreadPerIp,
		MinproxyKeys:     s.Ip.MinproxyKeys,
		MinproxyThreads:  s.Ip.MinproxyThreadPerIp,
		ProxyfarmKeys:    s.Ip.ProxyFarmKeys,
		ProxyfarmThreads: s.Ip.ProxyFarmThreadPerIp,
	})
	a.regProxyMgrKey = key
	a.regProxyMgrVersion = currentVer
	return a.regProxyMgr
}

// InvalidateProxyCache bump version để lần getSharedProxyManager kế force recreate
// nếu key thay đổi. Gọi từ SaveSettings/SaveInteractionConfig/SaveIpConfig.
func (a *App) InvalidateProxyCache() {
	a.proxyConfigVersion.Add(1)
}

// === INTERACTION CONFIG (Thiết lập chạy) ===

// PlatformUAConfig — cấu hình UA riêng cho từng API platform (reg/verify).
// Cho phép mỗi platform dùng bộ UA settings độc lập thay vì dùng chung global.
type PlatformUAConfig struct {
	UseOriginalUA         bool `json:"useOriginalUA"`
	AddVirtualSpecAndroid bool `json:"addVirtualSpecAndroid"`
	BuildUA               bool `json:"buildUA"`
	// ReplaceCarrier — chỉ có hiệu lực khi UseOriginalUA=true.
	ReplaceCarrier bool `json:"replaceCarrier"`
	// TrackingID — dùng cho SimulatePlatformUA preview; giá trị thực từ InteractionConfig.TrackingIDReg/Ver.
	TrackingID bool `json:"trackingID"`
	// UaPoolKey — override pool UA riêng cho platform này ("" = dùng global).
	UaPoolKey string `json:"uaPoolKey"`
	// Kind — "reg" hoặc "ver" — dùng để pick pool FBAV split đúng nguồn khi BuildUA=true.
	// Frontend SET khi gọi SimulatePlatformUA: regPlatformCfg → "reg", verPlatformCfg → "ver".
	Kind string `json:"kind"`
}

// InteractionConfig cấu hình chạy — mapping từ frontend VerifyConfig
type InteractionConfig struct {
	VerifyEnabled       bool   `json:"verifyEnabled"`
	MailProvider        string `json:"mailProvider"`
	MailList            string `json:"mailList"`
	CheckLiveDieEnabled bool   `json:"checkLiveDieEnabled"`
	TimeDelayCheck      int    `json:"timeDelayCheck"`
	TimeDelaySendCode   int    `json:"timeDelaySendCode"`
	SendAgainCode       bool   `json:"sendAgainCode"`
	OutputPath          string `json:"outputPath"`
	UaPoolKey           string `json:"uaPoolKey,omitempty"` // loại UA: "android"|"iphone"|"request"

	// ZeusX Hotmail — mua email để verify
	ZeusXApiKey      string `json:"zeusXApiKey"`
	ZeusXAccountCode string `json:"zeusXAccountCode"`

	// DongVanFB — mua email để verify
	DvfbApiKey      string `json:"dvfbApiKey"`
	DvfbAccountType string `json:"dvfbAccountType"` // account_type ID: "1"=HotMail NEW, "5"=Hotmail TRUSTED, ...

	// Store1s — mua email để verify
	Store1sApiKey    string `json:"store1sApiKey"`
	Store1sProductID string `json:"store1sProductId"` // product_id từ store1s.com (vd: "40559", "50510")

	// Mail30s (mailotp.com / mail30s.com) — mua email để verify
	Mail30sApiKey      string `json:"mail30sApiKey"`
	Mail30sProductSlug string `json:"mail30sProductSlug"` // product_slug từ mailotp.com

	// TempMailLol (api.tempmail.lol) — email tạm miễn phí / có API key
	TempMailLolApiKey string `json:"tempMailLolApiKey"` // optional Bearer token, free tier để trống

	// TempMailDomain domain tuỳ chỉnh cho provider đang chọn (backend đọc field này mỗi call).
	// Frontend giữ map riêng per-provider (TempMailDomains) — khi đổi provider, UI ghi slot tương ứng vào đây.
	TempMailDomain string `json:"tempMailDomain"`

	// TempMailDomains map per-provider domain. Frontend bind v-model theo provider đang chọn.
	// Backend chỉ dùng TempMailDomain (slot active) — map này tồn tại để persist giữa session.
	TempMailDomains map[string]string `json:"tempMailDomains,omitempty"`

	// TempMailToken — token/api key user nhập tay cho provider hiện hành (fallback khi
	// provider-specific field rỗng — vd tempMailLolApiKey/priyoEmailApiKey).
	TempMailToken string `json:"tempMailToken,omitempty"`

	// TempMailTokens map per-provider token — persist giữa session.
	TempMailTokens map[string]string `json:"tempMailTokens,omitempty"`

	// MuaMail config (api.muamail.store)
	MuaMailApiKey    string `json:"muaMailApiKey"`
	MuaMailProductID string `json:"muaMailProductId"`

	// UnlimitMail config (unlimitmail.com)
	UnlimitMailApiKey    string `json:"unlimitMailApiKey"`
	UnlimitMailProductID string `json:"unlimitMailProductId"`

	// SPTMail config (api.sptmail.com)
	SptMailApiKey      string `json:"sptMailApiKey"`
	SptMailServiceCode string `json:"sptMailServiceCode"`

	// EmailAPIInfo config (api.emailapi.info / gmail500.com)
	EmailAPIInfoApiKey      string `json:"emailAPIInfoApiKey"`
	EmailAPIInfoProductCode string `json:"emailAPIInfoProductCode"`

	// OtpCheap config (api.otp.cheap)
	OtpCheapApiKey    string `json:"otpCheapApiKey"`
	OtpCheapServiceID string `json:"otpCheapServiceId"`

	// ShopGmail9999 config (shopgmail9999.com)
	ShopGmail9999ApiKey  string `json:"shopGmail9999ApiKey"`
	ShopGmail9999Service string `json:"shopGmail9999Service"`

	// RentGmail config (rentgmail.online)
	RentGmailApiKey   string `json:"rentGmailApiKey"`
	RentGmailPlatform string `json:"rentGmailPlatform"`

	// OtpCodesSms config (otpcodesms.site)
	OtpCodesSmsApiKey    string `json:"otpCodesSmsApiKey"`
	OtpCodesSmsServiceID string `json:"otpCodesSmsServiceId"`

	// Wmemail config (www.wmemail.com)
	WmemailApiKey    string `json:"wmemailApiKey"`
	WmemailCommodity string `json:"wmemailCommodity"`

	// PriyoEmail config (free.priyo.email)
	PriyoEmailApiKey string `json:"priyoEmailApiKey"`

	// OTPHotmailPriority — nguồn đọc OTP ưu tiên cho 7 providers Hotmail OAuth2
	// (zeus-x, dongvanfb, store1s, mail30s, muamail, unlimitmail, wmemail).
	// Giá trị: "dongvan" (default) | "unlimit". Primary fail → fallback reader còn lại.
	OTPHotmailPriority string `json:"otpHotmailPriority"`

	// MailPoolBatch — số email mua batch đầu khi khởi động pool (mặc định 50).
	// Các lần sau khi pool cạn, mỗi luồng tự mua 1 con độc lập.
	MailPoolBatch int `json:"mailPoolBatch"`

	// Timing & Delay (Verify section)
	WaitCode          int `json:"waitCode"`
	WaitMail          int `json:"waitMail"`
	TrySendCode       int `json:"trySendCode"`
	UseMailTimes      int `json:"useMailTimes"`
	DelayConfirmEmail int `json:"delayConfirmEmail"`
	DelayCheckLive    int `json:"delayCheckLive"`
	DelayVeriReg      int `json:"delayVeriReg"`
	// AddMailRetry — số lần retry thêm khi add mail fail (0 = mặc định 2 outer attempts).
	// Mỗi retry gọi lại GetVerifyConfig() → đổi mail provider mid-run nếu user đổi provider.
	AddMailRetry int `json:"addMailRetry"`
	// RetryUnknownNow — sau khi pass 1 xong, tự động verify lại các acc Unknown/Error.
	// Chỉ chạy 1 pass thêm (không recursion). Bật từ UI: checkbox "Verify lại Unknown ngay".
	RetryUnknownNow bool `json:"retryUnknownNow"`

	// API & Logic (Verify section)
	ApiVerifyPlatform string `json:"apiVerifyPlatform"` // "api android"|"api mfb"|"api token"|"api web andr"
	// ApiVerifyPlatforms — multi-version verify. Nếu set (len>0) thì mỗi account verify
	// dùng 1 version round-robin từ list này (resolve 1 lần/account, ổn định suốt account).
	// Rỗng → fallback dùng ApiVerifyPlatform như cũ.
	ApiVerifyPlatforms []string `json:"apiVerifyPlatforms,omitempty"`
	ApiVerifyTokenType string   `json:"apiVerifyTokenType"` // "adspw"|"internal"|""

	// Reg account section
	ApiRegPlatform string `json:"apiRegPlatform"`
	// ApiRegPlatforms — multi-version reg. Nếu set (len>0), mỗi worker slot được gán
	// cố định 1 version theo round-robin (slot1→[0], slot2→[1], ...) suốt đời slot →
	// keep-ip / keep-ua / keep-datr hoạt động y hệt single-version trong từng slot.
	// Rỗng → fallback dùng ApiRegPlatform như cũ.
	ApiRegPlatforms           []string `json:"apiRegPlatforms,omitempty"`
	DelayReg                  int      `json:"delayReg"`
	DelayStep                 int      `json:"delayStep"` // delay giữa các step (ms), dùng cho s561v99
	LeadDomainMail            string   `json:"leadDomainMail"`
	PasswordReg               string   `json:"passwordReg"`
	NameRegLocale             string   `json:"nameRegLocale"`
	RegMode                   string   `json:"regMode"`
	RegModeRotate             bool     `json:"regModeRotate"`
	RegModeRotateMailMinutes  int      `json:"regModeRotateMailMinutes"`
	RegModeRotatePhoneMinutes int      `json:"regModeRotatePhoneMinutes"`
	VerifyAfterReg            bool     `json:"verifyAfterReg"`
	PhoneMailMode             string   `json:"phoneMailMode"`
	FmPhoneCode               bool     `json:"fmPhoneCode"` // C# FmPhoneCode — strip country code, prefix "0"
	UseUGForVerify            bool     `json:"useUGForVerify"`
	RegForVerify              bool     `json:"regForVerify"`

	// Cookie Initial — dùng cho MỌI platform reg (Android, S23, iOS, WebAndroid, MFB...)
	CookieInitialMethod     string `json:"cookieInitialMethod"`     // "file" | "new"
	LimitCookieInitial      bool   `json:"limitCookieInitial"`      // bật giới hạn số lần dùng mỗi cookie
	LimitCookieInitialCount int    `json:"limitCookieInitialCount"` // số lần tối đa mỗi cookie được dùng
	CookieInitialFile       string `json:"cookieInitialFile"`       // đường dẫn file cookie_initial.txt

	// Giới hạn checkpoint — tự dừng reg khi số checkpoint vượt ngưỡng
	LimitCheckpoint      bool `json:"limitCheckpoint"`
	LimitCheckpointCount int  `json:"limitCheckpointCount"`
	DeleteDatrCheckpoint bool `json:"deleteDatrCheckpoint"`

	// Giới hạn tuổi datr — xóa khỏi pool sau N phút kể từ lúc nạp
	LimitDatrAge        bool `json:"limitDatrAge"`
	LimitDatrAgeMinutes int  `json:"limitDatrAgeMinutes"`

	// SaveNewDatr — nếu true, ghi datr mới thu được từ cookie reg vào cookie_initial.txt
	SaveNewDatr bool `json:"saveNewDatr"`

	// Tạo tài khoản tự động
	CreateEnabled    bool   `json:"createEnabled"`
	CreateType       string `json:"createType"`       // "spam" | "tut"
	CreateCookieList string `json:"createCookieList"` // mỗi dòng một cookie
	CreateOutputPath string `json:"createOutputPath"` // thư mục lưu file tài khoản tạo thành công

	// Thư mục kết quả chung (SuccessReg, SuccessVerify, Die...)
	ResultFolderPath string `json:"resultFolderPath"`

	// Split mode: reg và verify chạy độc lập (reg ghi file → verify đọc file)
	SplitMode          bool `json:"splitMode"`
	SplitVerifyThreads int  `json:"splitVerifyThreads"` // số luồng verify riêng (0 = bằng regThreads)

	// RegThreads — số luồng register chạy song song. Trước đây nằm ở GeneralConfig.ThreadRequest,
	// đã chuyển vào InteractionConfig để reg và verify tự cài luồng riêng.
	RegThreads int `json:"regThreads"`

	// AutoRestart — sau N phút, tự động STOP toàn bộ tiến trình + RESET counters + RUN lại từ đầu.
	// Dùng để rotate proxy/datr pool (tránh burn dài).
	AutoRestartEnabled bool `json:"autoRestartEnabled"`
	AutoRestartMinutes int  `json:"autoRestartMinutes"` // mặc định 60 phút nếu enabled mà = 0

	// VerifySourceFolderPath — thư mục chứa file .txt tài khoản cần verify (verify-only mode).
	// Nếu set, RunVerify dùng folder này thay vì settings.General.AccountSourcePath.
	// Mỗi account được pop+xóa khỏi file ngay khi bắt đầu chạy.
	VerifySourceFolderPath string `json:"verifySourceFolderPath"`

	// KeepIPSuccess — Port C# MainFormUISettings.KeepIPSuccess.
	// Sau khi 1 account verify/reg thành công, giữ nguyên IP (proxy session) cho
	// account kế tiếp chạy trên CÙNG worker slot. Fail → release + acquire fresh.
	// Giảm bandwidth proxy + giữ "IP ngon" cho nhiều account liên tiếp.
	KeepIPSuccess bool `json:"keepIpSuccess"`

	// KeepUASuccess — giữ nguyên UA cho slot sau khi reg thành công.
	// Cùng pattern với KeepIPSuccess nhưng cho User-Agent: success → pin UA cho acc kế,
	// fail → UA mới. Giúp FB nhận fingerprint quen khi reg nhiều acc liên tiếp cùng slot.
	KeepUASuccess      bool `json:"keepUaSuccess"`
	KeepDatrSuccess    bool `json:"keepDatrSuccess"`
	KeepInitialSuccess bool `json:"keepInitialSuccess"` // Keep Contact: giữ email/phone của slot sau reg thành công.

	// AddVirtualSpecAndroid — prepend Dalvik/2.1.0 prefix trong UA (C# default true).
	// false → UA chỉ là FB4A blob, không Dalvik prefix.
	AddVirtualSpecAndroid bool `json:"addVirtualSpecAndroid"`

	// BuildUA — khi true dùng AndroidUABuilder để build UA động từ Config/DeviceInfo/.
	// false → dùng pool từ Config/UserAgent/<kind>_UG.txt.
	BuildUA bool `json:"buildUA"`

	// UseOriginalUA — khi true dùng UA gốc cố định theo platform (s555-s559).
	// Loại trừ lẫn nhau với BuildUA và AddVirtualSpecAndroid.
	UseOriginalUA bool `json:"useOriginalUA"`

	// ReplaceCarrier — chỉ có hiệu lực khi UseOriginalUA=true.
	// true (default) → thay FBCR/Viettel bằng nhà mạng khớp IP.
	// false → giữ nguyên carrier gốc trong UA.
	ReplaceCarrier bool `json:"replaceCarrier"`

	// TrackingIDReg/TrackingIDVer — thêm XID/<random16>; vào cuối UA (trước ]) cho Reg / Verify.
	TrackingIDReg bool `json:"trackingIDReg"`
	TrackingIDVer bool `json:"trackingIDVer"`

	// RegPlatformUA / VerifyPlatformUA — UA config riêng theo platform.
	// Key = apiRegPlatform / apiVerifyPlatform (vd "s559", "android").
	// Nếu key tồn tại → override BuildUA/AddVirtualSpecAndroid/UseOriginalUA toàn cục.
	RegPlatformUA    map[string]PlatformUAConfig `json:"regPlatformUA,omitempty"`
	VerifyPlatformUA map[string]PlatformUAConfig `json:"verifyPlatformUA,omitempty"`

	// ═══ Advanced verify options (port C# MainFormUISettings) ═══

	// ReUseEmail — reuse email đã verify success (ArchiveEmailCollection).
	// Sau verify OK → archive email → account kế có thể dùng lại (UsedCount < UseEmailTime).
	ReUseEmail   bool `json:"reUseEmail"`
	UseEmailTime int  `json:"useEmailTime"` // số lần tái dùng tối đa (default 1)

	// FmUserTmpMail — format username tempmail theo login info (phone/email) thay vì random.
	// Port StringUtils.CreateUsernameTmpMailFromLoginInf.
	FmUserTmpMail bool `json:"fmUserTmpMail"`

	// UseProxyTempMail — khi poll temp mail, dùng proxy riêng từ Config/Proxy/proxy_tempmail.txt.
	// Tránh temp mail rate limit IP.
	UseProxyTempMail bool `json:"useProxyTempmail"`

	// UseProxyGmail — khi dùng rent mail provider hỗ trợ proxy (zeus-x, muamail, unlimitmail),
	// pick proxy từ Config/Proxy/proxy_rentmail.txt.
	UseProxyGmail bool `json:"useProxyGmail"`

	// Enable2FA — sau khi verify email thành công, bật 2FA TOTP cho account.
	// Port C# FacebookSecurityFeatureAPIAndroid.TurnOnTwofactor. Trả secret 32-char
	// để user lưu cùng account (NVR|2FA format).
	Enable2FA bool `json:"enable2fa"`

	// GetNewDatrOnLive — sau khi verify Live, dùng token + cookie + UA của account đó
	// gọi GraphQL profile-switcher để lấy datr mới → thêm vào pool + ghi vào Pool file.
	// Hiệu quả hơn button GetNewDatrFromAccounts vì chạy inline, dùng đúng UA của verify.
	GetNewDatrOnLive bool `json:"getNewDatrOnLive"`

	// UploadAvatar — sau khi verify thành công (live), upload ảnh đại diện cho account.
	// Dùng S23 rupload flow: POST rupload.facebook.com → set via Bloks NUX mutation.
	UploadAvatar bool `json:"uploadAvatar"`

	// AvatarFolderPath — thư mục chứa ảnh JPEG/PNG để upload làm avatar.
	// Mỗi account live sẽ pick 1 ảnh ngẫu nhiên từ thư mục này.
	// Mặc định "Config/Avatar" nếu để trống.
	AvatarFolderPath string `json:"avatarFolderPath"`

	// DelayDisplayResult — giây giữ status cuối của account trên UI trước khi
	// fetch account mới vào slot. 0 = không delay (ghi đè ngay, khó đọc).
	// Khuyến nghị 3-5 giây để user đọc được email + status.
	DelayDisplayResult int `json:"delayDisplayResult"`

	// AddInfo — sau verify Live, cập nhật thông tin hồ sơ account.
	AddInfo             bool   `json:"addInfo"`
	AddInfoCity         bool   `json:"addInfoCity"`
	AddInfoHometown     bool   `json:"addInfoHometown"`
	AddInfoSchool       bool   `json:"addInfoSchool"`
	AddInfoCollege      bool   `json:"addInfoCollege"`
	AddInfoWork         bool   `json:"addInfoWork"`
	AddInfoRelationship bool   `json:"addInfoRelationship"`
	AddInfoDataDir      string `json:"addInfoDataDir"`
	AddInfoDelayMs      int    `json:"addInfoDelayMs"`

	// Auto-upload sau khi reg/ver xong — đọc config từ uploadsite.json
	AutoUploadAfterReg    bool `json:"autoUploadAfterReg"`
	AutoUploadAfterVerify bool `json:"autoUploadAfterVerify"`
}

// UploadSiteSourceConfig — cấu hình cho 1 nguồn (reg hoặc ver)
type UploadSiteSourceConfig struct {
	Enabled bool `json:"enabled"`
}

// UploadSiteConfig — cấu hình đẩy tài khoản lên banclone.pro
type UploadSiteConfig struct {
	Reg                  UploadSiteSourceConfig `json:"reg"`
	Ver                  UploadSiteSourceConfig `json:"ver"`
	Code                 string                 `json:"code"`            // mã kho hàng (stock code)
	ApiKey               string                 `json:"apiKey"`          // API key admin
	AdminUsername        string                 `json:"adminUsername"`   // tài khoản đăng nhập banclone.pro
	AdminPassword        string                 `json:"adminPassword"`   // mật khẩu đăng nhập banclone.pro
	FilterDuplicate      bool                   `json:"filterDuplicate"` // true=lọc trùng UID
	DelayCheckSec        int                    `json:"delayCheckSec"`
	AccPerBatch          int                    `json:"accPerBatch"`
	DelayBetweenBatchSec int                    `json:"delayBetweenBatchSec"`
}

func defaultUploadSiteConfig() UploadSiteConfig {
	return UploadSiteConfig{
		Reg:                  UploadSiteSourceConfig{Enabled: false},
		Ver:                  UploadSiteSourceConfig{Enabled: false},
		Code:                 "69ea28f9e5e3e",
		ApiKey:               "6ddcacd6d2b59363401c516292a786aaq2Aa14OynFgKJi5lQY7tcEZhXjIvBPs0",
		FilterDuplicate:      false,
		DelayCheckSec:        25,
		AccPerBatch:          900,
		DelayBetweenBatchSec: 9,
	}
}

// SaveUploadSiteConfig lưu cấu hình đẩy tài khoản vào Config/Settings/uploadsite.json
func (a *App) SaveUploadSiteConfig(data UploadSiteConfig) string {
	const settingsDir = "Config/Settings"
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "Lỗi marshal: " + err.Error()
	}
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return "Lỗi tạo thư mục: " + err.Error()
	}
	if err := os.WriteFile(filepath.Join(settingsDir, "uploadsite.json"), b, 0644); err != nil {
		return "Lỗi ghi file: " + err.Error()
	}
	return "OK"
}

// LoadUploadSiteConfig đọc cấu hình đẩy tài khoản từ Config/Settings/uploadsite.json
func (a *App) LoadUploadSiteConfig() UploadSiteConfig {
	cfg := defaultUploadSiteConfig()
	b, err := os.ReadFile(filepath.Join("Config/Settings", "uploadsite.json"))
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(b, &cfg)
	return cfg
}

// ============================================================================
// Upload Site Runner — WAL (Write-Ahead Log) + Retry/Backoff + Soft-Stop
// ============================================================================
// Cốt lõi:
//   - In-memory queue + retry với exponential backoff (không persist cross-session).
//   - Dedup UID: cùng 1 UID không enqueue 2 lần trong cùng session reg.
//   - Stats + log per-session (lưu trong currentResultPath, không có folder Config/Upload riêng).
//   - Stop = soft-stop: chờ drain hết acc trong queue mới exit.
//   - "Acc cũ" từ session trước KHÔNG auto-upload — fresh start mỗi run.
// ============================================================================

const (
	uploadStatsFile = "upload_stats.json"
	uploadLogFile   = "upload_push_log.txt"

	uploadLogRotateInterval = 2 * time.Hour
	uploadLogMaxSize        = 2 * 1024 * 1024 // 2MB hard cap (phòng rotate chưa kịp)
	// uploadHardStopTimeout — Task 4: 10p → 60s. Soft-stop chờ in-flight push xong
	// (mỗi push max 300s từ pushTimeoutFor). Sau 60s bất kể, hard-cancel ctx →
	// goroutine push cũng exit qua ctx.Done. Pending acc còn trong pending.txt sẽ
	// được load lại khi mở lại upload. Trade-off: lose in-flight push >60s sau Stop
	// để Stop thật sự responsive.
	uploadHardStopTimeout = 60 * time.Second
)

// UploadStats — counter cho UI. Lưu per-session trong currentResultPath.
type UploadStats struct {
	TotalUploaded       int    `json:"totalUploaded"`
	TotalFailed         int    `json:"totalFailed"`
	PendingCount        int    `json:"pendingCount"` // batch + pendingRetry + uploadCh (in-memory)
	ConsecutiveFailures int    `json:"consecutiveFailures"`
	DuplicateSkipped    int    `json:"duplicateSkipped"` // số UID bị dedup bỏ qua
	LastUploadAt        string `json:"lastUploadAt"`
	LastErrorAt         string `json:"lastErrorAt"`
	LastError           string `json:"lastError"`
	LastRotateAt        string `json:"lastRotateAt"`
	StartedAt           string `json:"startedAt"`
}

// uploadInMemoryPending — count "đang chờ" gồm pendingRetry + batch + channel.
// Cập nhật từ runUploadSite qua atomic.
var uploadPendingInMem atomic.Int32

// resultDir trả về thư mục result hiện tại; rỗng = chưa có session đang chạy.
func (a *App) resultDir() string {
	a.resultPathMu.Lock()
	defer a.resultPathMu.Unlock()
	return a.currentResultPath
}

func (a *App) uploadStatsPath() string {
	d := a.resultDir()
	if d == "" {
		return ""
	}
	return filepath.Join(d, uploadStatsFile)
}

func (a *App) uploadLogPath() string {
	d := a.resultDir()
	if d == "" {
		return ""
	}
	return filepath.Join(d, uploadLogFile)
}

// ──────────────────────────────────────────────────────────────────────────
// Dedup UID — tránh push trùng cùng 1 UID trong session
// ──────────────────────────────────────────────────────────────────────────

// extractUIDFromLine lấy UID từ dòng format "uid|pass|2fa|..." (FormatVerify).
func extractUIDFromLine(line string) string {
	if i := strings.IndexByte(line, '|'); i > 0 {
		return strings.TrimSpace(line[:i])
	}
	return strings.TrimSpace(line)
}

// extractTokenFromLine lấy Facebook access token (bắt đầu bằng EAA) từ dòng pipe-separated.
func extractTokenFromLine(line string) string {
	for _, f := range strings.Split(line, "|") {
		f = strings.TrimSpace(f)
		if strings.HasPrefix(f, "EAA") && len(f) > 20 {
			return f
		}
	}
	return ""
}

// ResetUploadSession xoá toàn bộ dedup UIDs + drain queue + clear retry — gọi khi bắt đầu run mới.
// Hook vào RunVerify/RunRegister sau khi set currentResultPath.
// Đảm bảo run mới không kế thừa acc cũ chưa push xong từ session trước.
func (a *App) ResetUploadSession() {
	a.uploadSeenUIDs.Range(func(k, _ any) bool {
		a.uploadSeenUIDs.Delete(k)
		return true
	})
	// Drain channel non-blocking — vứt acc cũ còn sót sau hard-stop của session trước.
	drained := 0
drainLoop:
	for {
		select {
		case <-a.uploadCh:
			drained++
		default:
			break drainLoop
		}
	}
	// Clear retry queue cũ
	a.uploadRetryMu.Lock()
	dropped := len(a.uploadRetryQueue)
	a.uploadRetryQueue = nil
	a.uploadRetryMu.Unlock()
	if drained > 0 || dropped > 0 {
		slog.Info("ResetUploadSession: drop state cũ", "queue", drained, "retry", dropped)
	}
	uploadPendingInMem.Store(0)
}

// enqueueForUpload đẩy line vào queue, dedup theo UID.
// In-memory only — app crash = mất acc trong queue (theo yêu cầu user).
func (a *App) enqueueForUpload(line string) {
	if line == "" {
		return
	}
	uid := extractUIDFromLine(line)
	if uid != "" {
		if _, loaded := a.uploadSeenUIDs.LoadOrStore(uid, true); loaded {
			// UID đã enqueue trong session này → bỏ qua (tránh trùng).
			a.updateUploadStats(func(s *UploadStats) { s.DuplicateSkipped++ })
			slog.Debug("enqueueForUpload: skip duplicate UID", "uid", uid)
			return
		}
	}
	uploadPendingInMem.Add(1)
	// Block nếu channel đầy — đảm bảo không mất acc (channel size 5000 đủ rộng).
	a.uploadCh <- line
}

// ──────────────────────────────────────────────────────────────────────────
// Stats: lưu trong currentResultPath/upload_stats.json
// ──────────────────────────────────────────────────────────────────────────

func (a *App) loadUploadStats() UploadStats {
	a.uploadStatsMu.Lock()
	defer a.uploadStatsMu.Unlock()
	var s UploadStats
	path := a.uploadStatsPath()
	if path == "" {
		return s
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	_ = json.Unmarshal(b, &s)
	return s
}

func (a *App) saveUploadStatsLocked(s UploadStats) {
	path := a.uploadStatsPath()
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

// updateUploadStats đọc-modify-ghi atomic dưới mutex. Skip nếu chưa có session.
func (a *App) updateUploadStats(fn func(*UploadStats)) {
	path := a.uploadStatsPath()
	if path == "" {
		return // chưa có session → không lưu (theo yêu cầu user)
	}
	a.uploadStatsMu.Lock()
	defer a.uploadStatsMu.Unlock()
	var s UploadStats
	if b, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(b, &s)
	}
	fn(&s)
	a.saveUploadStatsLocked(s)
}

// GetUploadStats — frontend API hiện counter trên UI.
func (a *App) GetUploadStats() UploadStats {
	s := a.loadUploadStats()
	s.PendingCount = int(uploadPendingInMem.Load())
	return s
}

// ──────────────────────────────────────────────────────────────────────────
// Logging: emit event + ghi file (có level + rotation)
// ──────────────────────────────────────────────────────────────────────────

const (
	logLevelInfo  = "info"
	logLevelOk    = "ok"
	logLevelWarn  = "warn"
	logLevelError = "error"
)

func (a *App) emitUploadLog(source, msg string, uploaded int) {
	a.emitUploadLogL(source, msg, uploaded, "")
}

// emitUploadLogL — biến thể có level. level rỗng → tự suy từ icon đầu msg.
func (a *App) emitUploadLogL(source, msg string, uploaded int, level string) {
	if level == "" {
		switch {
		case strings.HasPrefix(msg, "❌"):
			level = logLevelError
		case strings.HasPrefix(msg, "⚠"):
			level = logLevelWarn
		case strings.HasPrefix(msg, "✅"):
			level = logLevelOk
		default:
			level = logLevelInfo
		}
	}
	runtime.EventsEmit(a.ctx, "upload-site:log", map[string]interface{}{
		"source":   source,
		"msg":      msg,
		"uploaded": uploaded,
		"level":    level,
	})

	// Ghi vào upload_push_log.txt trong currentResultPath (per-session).
	// Nếu chưa có session → chỉ emit event, không ghi file.
	path := a.uploadLogPath()
	if path == "" {
		return
	}
	a.uploadLogMu.Lock()
	defer a.uploadLogMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	// Hard cap: nếu file > uploadLogMaxSize → truncate ngay (defense ngoài rotate timer).
	if fi, err := os.Stat(path); err == nil && fi.Size() > uploadLogMaxSize {
		_ = os.WriteFile(path, []byte{}, 0600)
	}
	line := fmt.Sprintf("[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), msg)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err == nil {
		_, _ = f.WriteString(line)
		_ = f.Close()
	}
}

// runUploadLogRotator chạy nền: mỗi 2h truncate log file + emit clear UI.
func (a *App) runUploadLogRotator(ctx context.Context) {
	t := time.NewTicker(uploadLogRotateInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if path := a.uploadLogPath(); path != "" {
				a.uploadLogMu.Lock()
				_ = os.WriteFile(path, []byte{}, 0600)
				a.uploadLogMu.Unlock()
			}
			a.updateUploadStats(func(s *UploadStats) {
				s.LastRotateAt = time.Now().Format(time.RFC3339)
			})
			runtime.EventsEmit(a.ctx, "upload-site:log-cleared", nil)
		}
	}
}

// ClearUploadLog — frontend gọi để clear ngay (nút Trash).
func (a *App) ClearUploadLog() {
	if path := a.uploadLogPath(); path != "" {
		a.uploadLogMu.Lock()
		_ = os.WriteFile(path, []byte{}, 0600)
		a.uploadLogMu.Unlock()
	}
	runtime.EventsEmit(a.ctx, "upload-site:log-cleared", nil)
}

// ──────────────────────────────────────────────────────────────────────────
// Lifecycle: Start / Stop / ensureRunning
// ──────────────────────────────────────────────────────────────────────────

// StartUploadSite bắt đầu goroutine upload (manual từ UI nếu cần).
func (a *App) StartUploadSite() string {
	a.uploadSiteMu.Lock()
	defer a.uploadSiteMu.Unlock()
	if a.uploadSiteCancel != nil {
		return "ALREADY_RUNNING"
	}
	cfg := a.LoadUploadSiteConfig()
	if !cfg.Reg.Enabled && !cfg.Ver.Enabled {
		return "NO_SOURCE"
	}
	a.startUploadSiteLocked(cfg)
	return "OK"
}

// startUploadSiteLocked — caller giữ uploadSiteMu.
func (a *App) startUploadSiteLocked(cfg UploadSiteConfig) {
	a.uploadStopping.Store(false)
	// Parent = a.ctx (Wails app lifecycle). Khi app shutdown, OnShutdown cancel a.ctx
	// → cascade cancel xuống upload runner + log rotator → goroutine exit gracefully.
	// Trước đây dùng context.Background() → upload tiếp tục chạy ngầm sau khi app close window.
	parent := a.ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancel(parent)
	a.uploadSiteCancel = cancel
	// Bump gen — defer của run cũ (nếu chưa exit) sẽ thấy gen khác → KHÔNG clear cancel mới.
	a.uploadSiteGen++
	myGen := a.uploadSiteGen
	slog.Info("upload pool create", "runID", fmt.Sprintf("upload#%d", myGen))

	rotCtx, rotCancel := context.WithCancel(parent)
	a.uploadLogRotateCancel = rotCancel
	go a.runUploadLogRotator(rotCtx)

	go a.runUploadSite(ctx, cfg, myGen)
}

// StopUploadSite — soft-stop: chờ drain hết pending mới exit (bảo toàn acc).
// Block tối đa uploadHardStopTimeout; nếu site banclone die thì hard-cancel
// nhưng pending.txt vẫn còn → lần sau mở app load lại.
func (a *App) StopUploadSite() {
	a.uploadSiteMu.Lock()
	if a.uploadSiteCancel == nil {
		a.uploadSiteMu.Unlock()
		return
	}
	a.uploadStopping.Store(true)
	cancel := a.uploadSiteCancel
	a.uploadSiteMu.Unlock()

	a.emitUploadLogL("", "⏸ Đang dừng — chờ upload xong các acc còn lại...", 0, logLevelInfo)

	// Chờ runner exit tự nhiên (pending.txt rỗng) hoặc timeout.
	deadline := time.After(uploadHardStopTimeout)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			a.emitUploadLogL("", "⚠ Hard-stop sau timeout — pending.txt còn dữ liệu, mở app sau sẽ tự upload lại", 0, logLevelWarn)
			cancel()
			return
		case <-tick.C:
			a.uploadSiteMu.Lock()
			done := a.uploadSiteCancel == nil
			a.uploadSiteMu.Unlock()
			if done {
				return
			}
		}
	}
}

// ensureUploadRunning khởi động goroutine upload nếu chưa chạy.
// Gọi từ ver/reg callback để auto-start khi cần.
func (a *App) ensureUploadRunning(cfg UploadSiteConfig) {
	a.uploadSiteMu.Lock()
	defer a.uploadSiteMu.Unlock()
	if a.uploadSiteCancel != nil {
		return
	}
	a.startUploadSiteLocked(cfg)
}

// ──────────────────────────────────────────────────────────────────────────
// Main loop — Parallel pushes + Capped batch + Retry queue
// ──────────────────────────────────────────────────────────────────────────
//
// Cốt lõi sửa "death spiral" (batch tăng vô tận khi push lỗi):
//   - Mỗi push CỐ ĐỊNH ≤ accPerBatch (không gộp pendingRetry + batch thành mega-batch).
//   - Push chạy goroutine riêng → main loop không block khi site phản hồi chậm.
//   - Concurrency limit = uploadMaxConcurrent (mặc định 3) → không hammer site.
//   - Failed batch → quay vào retryQueue với retryAt = now + backoff (10s→30s→60s→2p cap).
//   - Delay giữa batch tính TỪ "send-start" của push trước (không phải sau khi xong).
// ──────────────────────────────────────────────────────────────────────────

const uploadMaxConcurrent = 1 // số push song song tối đa

// uploadMaxRetryAttempts — sau N attempts vẫn fail → drop batch (log + ghi pending.txt
// đã xử lý ở caller). Tránh retry vô hạn khi site die 24/7 → retryQueue grow vô tận.
// 5 lần × backoff 10s/30s/60s/2p/2p = ~5 phút trước khi drop. Site nào bị flag thật
// thì sau 5p coi như chết hẳn, đẩy thêm cũng vô ích.
const uploadMaxRetryAttempts = 5

// uploadMaxRetryQueueLen — cap hard size của retryQueue. Nếu vượt → drop oldest.
// Defense in-depth: kể cả khi site die rất lâu, RAM không grow vô tận.
const uploadMaxRetryQueueLen = 500

// retryItem — 1 batch push lỗi đang chờ retry.
type retryItem struct {
	accs     []string
	retryAt  time.Time
	attempts int
}

// uploadBackoff trả về thời gian chờ trước khi retry batch đó.
// 10s → 30s → 60s → 2p cap. Ngắn vì retry chạy parallel, không block.
func uploadBackoff(attempts int) time.Duration {
	switch {
	case attempts <= 0:
		return 0
	case attempts == 1:
		return 10 * time.Second
	case attempts == 2:
		return 30 * time.Second
	case attempts == 3:
		return 60 * time.Second
	default:
		return 2 * time.Minute
	}
}

func (a *App) runUploadSite(ctx context.Context, cfg UploadSiteConfig, myGen int64) {
	defer func() {
		a.uploadSiteMu.Lock()
		// CHỈ clear cancel + rotate khi gen còn khớp — nếu Stop→Start nhanh, run mới
		// đã bump gen và assign cancel mới; defer của run cũ KHÔNG được phép clear nhầm.
		if a.uploadSiteGen == myGen {
			a.uploadSiteCancel = nil
			if a.uploadLogRotateCancel != nil {
				a.uploadLogRotateCancel()
				a.uploadLogRotateCancel = nil
			}
		}
		a.uploadSiteMu.Unlock()
		// uploadStopping flag chỉ reset nếu là run hiện tại — nếu run mới đã start,
		// nó tự reset stopping=false rồi (line 4002), không cần đụng.
		if a.uploadSiteGen == myGen {
			a.uploadStopping.Store(false)
		}
		// Task 4: close idle TCP/TLS conn của bancloneTransport sau run.
		// Transport là singleton (shared giữa các run) nhưng giải phóng idle conn
		// sau mỗi run giảm RSS rõ rệt khi user chạy upload ngắt quãng. Active push
		// đã exit qua wg-equivalent (inFlight + signalWake → main loop), không bị abort.
		bancloneTransport.CloseIdleConnections()
		slog.Info("upload pool cleanup", "runID", fmt.Sprintf("upload#%d", myGen))
		a.emitUploadLogL("", "🛑 Đã dừng goroutine upload.", 0, logLevelInfo)
		runtime.EventsEmit(a.ctx, "upload-site:stopped", nil)
	}()

	a.updateUploadStats(func(s *UploadStats) {
		s.StartedAt = time.Now().Format(time.RFC3339)
	})

	checkInterval := time.Duration(cfg.DelayCheckSec) * time.Second
	if checkInterval <= 0 {
		checkInterval = 30 * time.Second
	}

	a.emitUploadLogL("", fmt.Sprintf("🚀 Bắt đầu — tối đa %d acc/lần, %d push song song, delay check %ds",
		cfg.AccPerBatch, uploadMaxConcurrent, cfg.DelayCheckSec), 0, logLevelInfo)

	var (
		batch          []string     // accs đang tích lũy chờ gửi
		inFlight       atomic.Int32 // số push đang chạy
		lastFlushStart time.Time    // mốc tính delayBetweenBatch (từ start)
		lastIdleLog    time.Time
		wakeup         = make(chan struct{}, 1) // signal main loop chạy lại sớm
		hardStop       bool
	)

	latestCfg := func() UploadSiteConfig { return a.LoadUploadSiteConfig() }
	signalWake := func() {
		select {
		case wakeup <- struct{}{}:
		default:
		}
	}

	// pushAsync chạy 1 push trong goroutine. Tăng inFlight, push xong giải phóng slot.
	// Trên error: enqueue lại vào retryQueue với backoff.
	pushAsync := func(accs []string, attempts int) {
		inFlight.Add(1)
		go func() {
			defer func() {
				inFlight.Add(-1)
				signalWake()
			}()
			c := latestCfg()
			filter := "1"
			if !c.FilterDuplicate {
				filter = "0"
			}
			// ctx = runUploadSite ctx (run-scoped). Stop sẽ cancel push đang chạy
			// thay vì để TCP keep-alive treo đến hết 180s timeout.
			n, err := pushToBanclone(ctx, c.Code, c.ApiKey, filter, accs)
			if err != nil {
				newAttempts := attempts + 1
				a.updateUploadStats(func(s *UploadStats) {
					s.TotalFailed += len(accs)
					s.ConsecutiveFailures = newAttempts
					s.LastError = err.Error()
					s.LastErrorAt = time.Now().Format(time.RFC3339)
					s.PendingCount = int(uploadPendingInMem.Load())
				})

				// Skip retry nếu run đã Stop — tránh enqueue vào queue mà main loop
				// không pop nữa → retryQueue grow trong memory cho đến app close.
				// Pending acc vẫn còn trong pending.txt → lần next start sẽ load lại.
				if ctx.Err() != nil {
					slog.Info("uploadSite: skip retry — ctx cancelled", "accs", len(accs))
					a.emitUploadLogL("", fmt.Sprintf("⏹ Stop giữa chừng — bỏ qua retry %d acc (sẽ load lại lần sau)",
						len(accs)), 0, logLevelInfo)
					return
				}

				// Cap MAX retry — sau N lần vẫn fail thì drop batch, không retry tiếp.
				// Site die hẳn thì retry mãi vô ích + grow RAM. Pending.txt giữ acc đó để
				// lần start sau load lại từ disk.
				if newAttempts >= uploadMaxRetryAttempts {
					slog.Warn("uploadSite: drop batch sau max retry",
						"attempts", newAttempts, "accs", len(accs), "err", err)
					a.emitUploadLogL("", fmt.Sprintf("⛔ Bỏ qua %d acc sau %d lần retry (site die hoặc lỗi cứng): %v",
						len(accs), newAttempts, err), 0, logLevelError)
					return
				}

				delay := uploadBackoff(newAttempts)
				slog.Warn("uploadSite: push lỗi", "err", err, "attempts", newAttempts, "accs", len(accs))
				a.emitUploadLogL("", fmt.Sprintf("❌ Push lỗi (%d acc, lần %d/%d): %v — retry sau %s",
					len(accs), newAttempts, uploadMaxRetryAttempts, err, delay), 0, logLevelError)
				// Push lại vào retryQueue. Cần lock vì chạy trong goroutine khác main loop.
				a.uploadRetryMu.Lock()
				// Cap hard size — nếu queue vượt, drop oldest entry trước khi append.
				// Defense in-depth: kể cả khi site treo rất lâu, retryQueue không grow vô tận.
				if len(a.uploadRetryQueue) >= uploadMaxRetryQueueLen {
					dropped := a.uploadRetryQueue[0]
					a.uploadRetryQueue = a.uploadRetryQueue[1:]
					slog.Warn("uploadSite: retry queue full, drop oldest",
						"droppedAccs", len(dropped.accs), "queueCap", uploadMaxRetryQueueLen)
				}
				a.uploadRetryQueue = append(a.uploadRetryQueue, &retryItem{
					accs:     accs,
					retryAt:  time.Now().Add(delay),
					attempts: newAttempts,
				})
				a.uploadRetryMu.Unlock()
				return
			}
			// Success
			uploadPendingInMem.Add(-int32(len(accs)))
			if uploadPendingInMem.Load() < 0 {
				uploadPendingInMem.Store(0)
			}
			a.updateUploadStats(func(s *UploadStats) {
				s.TotalUploaded += n
				s.ConsecutiveFailures = 0
				s.LastUploadAt = time.Now().Format(time.RFC3339)
				s.PendingCount = int(uploadPendingInMem.Load())
			})
			a.emitUploadLogL("", fmt.Sprintf("✅ Tải lên thành công: %d accounts", n), n, logLevelOk)
		}()
	}

	const uploadCheckUA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/148.0.0.0 Safari/537.36"

	// uploadCheckLine check live/die bằng UID (picture endpoint) trước khi add vào batch.
	// Dùng CheckLiveDieByPicture thay vì token — tránh false positive khi token bị checkpoint
	// nhưng account vẫn live (picture endpoint không đòi token).
	uploadCheckLine := func(line string) {
		uid := extractUIDFromLine(line)
		if uid == "" {
			batch = append(batch, line)
			return
		}
		checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		result := verifybase.CheckLiveDieByPicture(checkCtx, uploadCheckUA, uid)
		if result == "Die" {
			a.emitUploadLogL(uid, "⛔ Bỏ qua upload — UID die", 0, logLevelWarn)
			a.updateUploadStats(func(s *UploadStats) { s.TotalFailed++ })
			return
		}
		batch = append(batch, line)
	}

	// drainBatch hút thêm acc từ uploadCh vào batch local (non-blocking).
	drainBatch := func() {
		for {
			select {
			case more, ok := <-a.uploadCh:
				if !ok {
					return
				}
				uploadCheckLine(more)
			default:
				return
			}
		}
	}

	// pickRetryReady lấy 1 retry item đã đến hạn (retryAt ≤ now). Nil nếu không có.
	// Lock retryMu khi truy cập queue.
	pickRetryReady := func() *retryItem {
		now := time.Now()
		a.uploadRetryMu.Lock()
		defer a.uploadRetryMu.Unlock()
		for i, item := range a.uploadRetryQueue {
			if !item.retryAt.After(now) {
				// Pop item này — giữ thứ tự còn lại.
				a.uploadRetryQueue = append(a.uploadRetryQueue[:i], a.uploadRetryQueue[i+1:]...)
				return item
			}
		}
		return nil
	}

	retryQueueLen := func() int {
		a.uploadRetryMu.Lock()
		defer a.uploadRetryMu.Unlock()
		return len(a.uploadRetryQueue)
	}

	// nextRetryAt — thời điểm sớm nhất 1 retry item sẽ đến hạn (cho timer).
	nextRetryAt := func() time.Time {
		a.uploadRetryMu.Lock()
		defer a.uploadRetryMu.Unlock()
		var earliest time.Time
		for _, item := range a.uploadRetryQueue {
			if earliest.IsZero() || item.retryAt.Before(earliest) {
				earliest = item.retryAt
			}
		}
		return earliest
	}

	// kickPushes thử khởi động các push mới (retry trước, batch sau).
	// Tôn trọng: maxBatch, maxConcurrent, delayBetween (từ send-start).
	kickPushes := func() {
		c := latestCfg()
		maxBatch := c.AccPerBatch
		if maxBatch <= 0 {
			maxBatch = 100
		}
		delayBetween := time.Duration(c.DelayBetweenBatchSec) * time.Second

		for inFlight.Load() < int32(uploadMaxConcurrent) {
			// Gate delayBetween — TỪ LÚC LASTFLUSHSTART, không phải sau khi push xong.
			if !lastFlushStart.IsZero() && delayBetween > 0 {
				if elapsed := time.Since(lastFlushStart); elapsed < delayBetween {
					return // chưa đủ delay → để tick sau wake lại
				}
			}

			// Ưu tiên retry trước (FIFO theo retryAt).
			if item := pickRetryReady(); item != nil {
				lastFlushStart = time.Now()
				a.emitUploadLogL("", fmt.Sprintf("🔁 Retry %d acc (lần %d)", len(item.accs), item.attempts), 0, logLevelInfo)
				pushAsync(item.accs, item.attempts)
				continue
			}

			// Batch mới — cap ≤ maxBatch.
			drainBatch()
			if len(batch) == 0 {
				return
			}
			n := len(batch)
			if n > maxBatch {
				n = maxBatch
			}
			toSend := batch[:n]
			batch = batch[n:]

			lastFlushStart = time.Now()
			a.emitUploadLogL("", fmt.Sprintf("⏫ Đang tải lên %d accounts...", n), 0, logLevelInfo)
			pushAsync(toSend, 0)
		}
	}

	// Soft-stop exit: stopping && batch=∅ && retryQueue=∅ && inFlight=0 && channel=∅.
	allDrained := func() bool {
		if !a.uploadStopping.Load() {
			return false
		}
		if len(batch) > 0 || retryQueueLen() > 0 || inFlight.Load() > 0 {
			return false
		}
		select {
		case line, ok := <-a.uploadCh:
			if ok {
				uploadCheckLine(line)
			}
			return false
		default:
			return true
		}
	}

	for {
		if hardStop {
			return
		}
		drainBatch()
		kickPushes()
		if allDrained() {
			return
		}

		// Tính thời gian chờ: min(checkInterval, time-to-next-retry, time-to-delay-end).
		waitDur := checkInterval
		c := latestCfg()
		if delay := time.Duration(c.DelayBetweenBatchSec) * time.Second; delay > 0 && !lastFlushStart.IsZero() {
			if rem := delay - time.Since(lastFlushStart); rem > 0 && rem < waitDur {
				waitDur = rem
			}
		}
		if t := nextRetryAt(); !t.IsZero() {
			if rem := time.Until(t); rem > 0 && rem < waitDur {
				waitDur = rem
			}
		}
		if waitDur < 100*time.Millisecond {
			waitDur = 100 * time.Millisecond
		}

		select {
		case <-ctx.Done():
			// Hard-stop từ StopUploadSite timeout. Để các push đang chạy hoàn tất background.
			a.uploadStopping.Store(true)
			hardStop = true
			return

		case <-wakeup:
			// goroutine push xong → vòng lại để kick push mới.

		case line, ok := <-a.uploadCh:
			if !ok {
				a.uploadStopping.Store(true)
				return
			}
			uploadCheckLine(line)

		case <-time.After(waitDur):
			// Idle log — không có gì làm.
			if len(batch) == 0 && retryQueueLen() == 0 && inFlight.Load() == 0 {
				if time.Since(lastIdleLog) >= 60*time.Second {
					a.emitUploadLogL("", fmt.Sprintf("⏳ Chờ acc... (check mỗi %ds)", c.DelayCheckSec), 0, logLevelInfo)
					lastIdleLog = time.Now()
				}
			}
		}
	}
}

// readNewLinesFrom đọc các dòng mới trong file kể từ byte offset.
// Trả về (lines, newOffset, error). newOffset chỉ tăng khi có dòng được đọc thành công.
func readNewLinesFrom(filePath string, fromByte int64) ([]string, int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fromByte, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, fromByte, err
	}
	if info.Size() <= fromByte {
		return nil, fromByte, nil
	}

	if _, err := f.Seek(fromByte, io.SeekStart); err != nil {
		return nil, fromByte, err
	}

	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fromByte, err
	}

	raw := strings.Split(string(b), "\n")
	var lines []string
	for _, l := range raw {
		l = strings.TrimSpace(l)
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines, fromByte + int64(len(b)), nil
}

// SaveInteractionConfig lưu cấu hình thiết lập chạy vào active profile và interaction.json
func (a *App) SaveInteractionConfig(data InteractionConfig) string {
	const settingsDir = "Config/Settings"

	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "Lỗi marshal: " + err.Error()
	}
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return "Lỗi tạo thư mục: " + err.Error()
	}

	// 1. Lưu vào active profile (profile-aware)
	a.settingsMu.Lock()
	if p := a.appSettings.GetActiveProfile(); p != nil {
		p.Interaction = json.RawMessage(b)
		// Sync legacy fields để runner dùng được ngay
		var lic adapter.LegacyInteractionConfig
		if jsonErr := json.Unmarshal(b, &lic); jsonErr == nil {
			adapter.ApplyInteractionToProfile(p, lic)
		}
	}
	app := a.appSettings
	a.settingsMu.Unlock()

	// 2. Persist AppSettings (chứa profile.Interaction vừa cập nhật)
	if err := appsettings.Save(settingsDir, app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}

	// 3. Ghi interaction.json để backward-compat (runner đọc trực tiếp)
	if err := os.WriteFile(filepath.Join(settingsDir, "interaction.json"), b, 0644); err != nil {
		slog.Warn("SaveInteractionConfig: ghi interaction.json thất bại", "err", err)
	}

	// 4. Sync VerifySourceFolderPath → general.json AccountSourcePath.
	// 2 field là 1 giá trị duy nhất — user edit ở Interaction sẽ hiện ở General.
	if data.VerifySourceFolderPath != "" {
		if gb, err := os.ReadFile(filepath.Join(settingsDir, "general.json")); err == nil {
			var gs SettingsData
			if json.Unmarshal(gb, &gs) == nil {
				if gs.General.AccountSourcePath != data.VerifySourceFolderPath {
					gs.General.AccountSourcePath = data.VerifySourceFolderPath
					if patched, err := json.MarshalIndent(gs, "", "  "); err == nil {
						_ = os.WriteFile(filepath.Join(settingsDir, "general.json"), patched, 0644)
					}
				}
			}
		}
	}

	return "OK"
}

// LoadInteractionConfig đọc cấu hình thiết lập chạy.
// Thứ tự ưu tiên: active profile.Interaction → interaction.json → a.appSettings (backward compat)
// First-run (chưa có config) → full defaults. Subsequent → chỉ string defaults.
func (a *App) LoadInteractionConfig() InteractionConfig {
	// 1. Đọc từ active profile (profile-aware, nguồn chính)
	a.settingsMu.RLock()
	p := a.appSettings.GetActiveProfile()
	var profileInteraction []byte
	if p != nil && len(p.Interaction) > 0 {
		profileInteraction = []byte(p.Interaction)
	}
	a.settingsMu.RUnlock()

	if len(profileInteraction) > 0 {
		var data InteractionConfig
		if json.Unmarshal(profileInteraction, &data) == nil {
			applyInteractionStringDefaults(&data)
			a.migrateThreadCount(&data)
			return data
		}
	}

	// 2. Fallback: interaction.json (migration từ format cũ)
	if b, err := os.ReadFile(filepath.Join("Config/Settings", "interaction.json")); err == nil {
		var data InteractionConfig
		if json.Unmarshal(b, &data) == nil {
			applyInteractionStringDefaults(&data)
			a.migrateThreadCount(&data)
			return data
		}
	}

	// 3. Fallback cuối (first-run): adapter + apply full defaults (bool + string).
	a.settingsMu.RLock()
	lic := adapter.ToLegacyInteraction(a.appSettings)
	a.settingsMu.RUnlock()
	var data InteractionConfig
	if b, err := json.Marshal(lic); err == nil {
		if err := json.Unmarshal(b, &data); err != nil {
			slog.Warn("LoadInteractionConfig fallback: unmarshal thất bại", "err", err)
		}
	}
	applyInteractionFullDefaults(&data)
	a.migrateThreadCount(&data)
	return data
}

// migrateThreadCount fill RegThreads từ general.ThreadRequest khi user chưa migrate
// sang field mới. Trước đây luồng nằm ở GeneralConfig; nay đã chuyển sang InteractionConfig
// để reg và verify tự cài luồng riêng.
func (a *App) migrateThreadCount(c *InteractionConfig) {
	if c == nil || c.RegThreads > 0 {
		return
	}
	t := a.LoadSettings().General.ThreadRequest
	if t > 0 {
		c.RegThreads = t
	} else {
		c.RegThreads = 20
	}
}

// applyInteractionStringDefaults fill string/int defaults cho field rỗng.
// Dùng khi config đã tồn tại (user đã save 1 lần) — giữ bool choice của user.
func applyInteractionStringDefaults(c *InteractionConfig) {
	if c == nil {
		return
	}
	// REG string defaults
	if strings.TrimSpace(c.ApiRegPlatform) == "" {
		c.ApiRegPlatform = "s23" // khớp value option ở <select> (lowercase)
	}
	if strings.TrimSpace(c.LeadDomainMail) == "" {
		c.LeadDomainMail = "@gmail.com,@yahoo.com"
	}
	if strings.TrimSpace(c.NameRegLocale) == "" {
		c.NameRegLocale = "US"
	}
	if strings.TrimSpace(c.RegMode) == "" {
		c.RegMode = "Phone"
	}
	if c.RegModeRotateMailMinutes <= 0 {
		c.RegModeRotateMailMinutes = 360
	}
	if c.RegModeRotatePhoneMinutes <= 0 {
		c.RegModeRotatePhoneMinutes = 360
	}
	if strings.TrimSpace(c.PhoneMailMode) == "" {
		c.PhoneMailMode = "random-normal"
	}
	// VERIFY string defaults
	if strings.TrimSpace(c.ApiVerifyPlatform) == "" {
		c.ApiVerifyPlatform = "api android"
	}
	if strings.TrimSpace(c.ApiVerifyTokenType) == "" {
		c.ApiVerifyTokenType = "adspw"
	}
	if strings.TrimSpace(c.MailProvider) == "" {
		c.MailProvider = "mail1sec"
	}
	// Cookie Initial — chỉ còn 2 method: "file" (mặc định) hoặc "new" (sinh datr nội bộ).
	// Config cũ có method="ck" (đã bỏ khỏi UI) được migrate về "file".
	method := strings.ToLower(strings.TrimSpace(c.CookieInitialMethod))
	if method == "" || method == "ck" {
		c.CookieInitialMethod = "file"
	}
	// Create type
	if strings.TrimSpace(c.CreateType) == "" {
		c.CreateType = "spam"
	}
	// Timing defaults — user muốn tất cả = 1 (nhanh, đơn giản, user tự chỉnh nếu cần)
	if c.WaitCode <= 0 {
		c.WaitCode = 1
	}
	if c.WaitMail <= 0 {
		c.WaitMail = 1
	}
	if c.TrySendCode <= 0 {
		c.TrySendCode = 1
	}
	if c.UseMailTimes <= 0 {
		c.UseMailTimes = 1
	}
	if c.DelayConfirmEmail <= 0 {
		c.DelayConfirmEmail = 1
	}
	if c.DelayCheckLive <= 0 {
		c.DelayCheckLive = 1
	}
	if c.DelayVeriReg <= 0 {
		c.DelayVeriReg = 1
	}
	if c.DelayDisplayResult <= 0 {
		c.DelayDisplayResult = 1
	}
	if c.TimeDelayCheck <= 0 {
		c.TimeDelayCheck = 1
	}
	if c.TimeDelaySendCode <= 0 {
		c.TimeDelaySendCode = 1
	}
	if c.LimitCookieInitialCount <= 0 {
		c.LimitCookieInitialCount = 3
	}
	if c.LimitDatrAgeMinutes <= 0 {
		c.LimitDatrAgeMinutes = 60
	}
}

func applyRegModeRotation(c InteractionConfig, startedAt, now time.Time) InteractionConfig {
	c.RegMode = effectiveRegMode(c, startedAt, now)
	return c
}

func effectiveRegMode(c InteractionConfig, startedAt, now time.Time) string {
	base := strings.TrimSpace(c.RegMode)
	if base == "" {
		base = "Phone"
	}
	if !c.RegModeRotate {
		return base
	}

	phoneMode := strings.EqualFold(base, "Phone")
	mailMode := strings.EqualFold(base, "Mail")
	if !phoneMode && !mailMode {
		return base
	}

	phoneDur := time.Duration(c.RegModeRotatePhoneMinutes) * time.Minute
	mailDur := time.Duration(c.RegModeRotateMailMinutes) * time.Minute
	if phoneDur <= 0 {
		phoneDur = 360 * time.Minute
	}
	if mailDur <= 0 {
		mailDur = 360 * time.Minute
	}

	elapsed := now.Sub(startedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	pos := elapsed % (phoneDur + mailDur)
	if mailMode {
		if pos < mailDur {
			return "Mail"
		}
		return "Phone"
	}
	if pos < phoneDur {
		return "Phone"
	}
	return "Mail"
}

// applyInteractionFullDefaults áp dụng full defaults (bool + string) cho first-run.
// Chuẩn C# defaults — user khởi động app lần đầu tick sẵn các option quan trọng.
func applyInteractionFullDefaults(c *InteractionConfig) {
	if c == nil {
		return
	}
	applyInteractionStringDefaults(c)
	c.VerifyEnabled = true       // tick Verify panel
	c.CheckLiveDieEnabled = true // tick Kiểm tra Live/Die
	c.SendAgainCode = true       // tick Gửi lại code nếu không nhận
	c.VerifyAfterReg = true      // tự động verify sau reg thành công
	c.KeepIPSuccess = true       // C# KeepIPSuccess default — giữ proxy cho acc kế
	c.CreateEnabled = true       // Register panel ON sẵn — user muốn reg là chính
	// UA gốc là default — chỉ khi user tick thủ công mới build động với Dalvik/buildFile
	c.AddVirtualSpecAndroid = false
	c.BuildUA = false
	// ReUseEmail, FmUserTmpMail, UseProxyTempMail, FmPhoneCode, SplitMode...
	// giữ zero (false) — user tuỳ chọn, không tick sẵn.
}

// === Legacy Import ===

// LegacyFieldEntry — một field trong báo cáo mapping từ legacy config
type LegacyFieldEntry struct {
	LegacyKey    string `json:"legacyKey"`
	NewPath      string `json:"newPath"`
	DisplayValue string `json:"displayValue"`
	Status       string `json:"status"` // "ok" | "confirm" | "sensitive" | "unsupported"
	Note         string `json:"note"`
}

// LegacyMappingReport — kết quả phân tích mapping legacy config
type LegacyMappingReport struct {
	MappedOk     []LegacyFieldEntry `json:"mappedOk"`
	NeedsConfirm []LegacyFieldEntry `json:"needsConfirm"`
	Sensitive    []LegacyFieldEntry `json:"sensitive"`
	Unsupported  []LegacyFieldEntry `json:"unsupported"`
	ParseErrors  []string           `json:"parseErrors"`
}

// LegacyParseResult — kết quả trả về từ ParseLegacyConfig
type LegacyParseResult struct {
	Report LegacyMappingReport `json:"report"`
	Error  string              `json:"error"`
}

// ParseLegacyConfig phân tích cặp JSON general + interaction từ tool cũ,
// trả về MappingReport mô tả từng field — KHÔNG lưu, chỉ preview.
func (a *App) ParseLegacyConfig(generalJSON, interactionJSON string) LegacyParseResult {
	var s adapter.LegacySettingsData
	var ic adapter.LegacyInteractionConfig
	var parseErrors []string

	if generalJSON != "" {
		if err := json.Unmarshal([]byte(generalJSON), &s); err != nil {
			parseErrors = append(parseErrors, "general.json: "+err.Error())
		}
	}
	if interactionJSON != "" {
		if err := json.Unmarshal([]byte(interactionJSON), &ic); err != nil {
			parseErrors = append(parseErrors, "interaction.json: "+err.Error())
		}
	}

	if len(parseErrors) > 0 && generalJSON != "" && interactionJSON != "" {
		return LegacyParseResult{Error: strings.Join(parseErrors, "; ")}
	}

	report := adapter.BuildMappingReport(s, ic)
	report.ParseErrors = append(report.ParseErrors, parseErrors...)

	// Convert adapter.MappedField → LegacyFieldEntry
	conv := func(fields []adapter.MappedField) []LegacyFieldEntry {
		out := make([]LegacyFieldEntry, len(fields))
		for i, f := range fields {
			out[i] = LegacyFieldEntry{f.LegacyKey, f.NewPath, f.DisplayValue, string(f.Status), f.Note}
		}
		return out
	}

	return LegacyParseResult{
		Report: LegacyMappingReport{
			MappedOk:     conv(report.MappedOk),
			NeedsConfirm: conv(report.NeedsConfirm),
			Sensitive:    conv(report.Sensitive),
			Unsupported:  conv(report.Unsupported),
			ParseErrors:  report.ParseErrors,
		},
	}
}

// CheckCurrentIPViaProxy kiểm tra IP hiện tại qua proxy đầu tiên trong danh sách Proxy Settings.
// Nếu không có proxy nào cấu hình → check IP trực tiếp (IP thật của máy).
// Kết quả: "IP/country" (vd "1.2.3.4/vn") hoặc chỉ IP nếu không lấy được country.
func (a *App) CheckCurrentIPViaProxy() string {
	// Lấy proxy đầu tiên từ Proxy Settings (IpConfig.ProxyList).
	settings := a.LoadSettings()
	proxyStr := ""
	if list := strings.TrimSpace(activeProxyList(settings.Ip)); list != "" {
		for _, line := range strings.Split(list, "\n") {
			if p := strings.TrimSpace(line); p != "" {
				proxyStr = p
				break
			}
		}
	}

	// Render session ID mới (nếu proxy hỗ trợ rotating) để test IP hiện tại đúng.
	if proxyStr != "" {
		proxyStr = proxy.RenderSessionIfIsProxyServer(proxyStr)
	}

	// 20s tổng: đủ budget cho amazonaws (1-3s) + adspower (6s timeout) + ipify fallback.
	// Proxy rotating pool (iprocket KZ/EU) adspower hay block → cần fall-through.
	// Parent = a.ctx → app shutdown cancel được lookup; trước đây dùng context.Background()
	// khiến CheckIP treo đến hết 20s ngay cả khi user đóng app giữa chừng.
	parent := a.ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 20*time.Second)
	defer cancel()

	ip, err := proxy.CheckIP(ctx, proxyStr, settings.General.ApiCheckIp)
	if err != nil || ip == "" {
		if err == nil {
			return "Lỗi: không lấy được IP qua proxy"
		}
		return "Lỗi: " + err.Error()
	}
	return ip
}

// LoadProxyList đọc nội dung file proxy (proxy_tempmail.txt hoặc proxy_rentmail.txt).
// kind: "tempmail" hoặc "gmail" (kind "gmail" map sang proxy_rentmail.txt để giữ back-compat API).
// Trả "" nếu file chưa tồn tại.
func (a *App) LoadProxyList(kind string) string {
	path := proxyListPath(kind)
	if path == "" {
		return ""
	}
	// Auto-migrate: nếu file mới chưa có nhưng file legacy có → rename.
	if strings.ToLower(strings.TrimSpace(kind)) == "gmail" {
		legacyPath := filepath.Join(filepath.Dir(path), "proxy_gmail.txt")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if _, err2 := os.Stat(legacyPath); err2 == nil {
				_ = os.Rename(legacyPath, path)
			}
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// SaveProxyList lưu text vào proxy list file cạnh exe (Config/Proxy/).
// kind: "tempmail" hoặc "gmail".
func (a *App) SaveProxyList(kind, content string) string {
	path := proxyListPath(kind)
	if path == "" {
		return "kind không hợp lệ (chỉ nhận tempmail/gmail)"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "Lỗi tạo thư mục: " + err.Error()
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "Lỗi lưu: " + err.Error()
	}
	return "OK"
}

// proxyListPath trả về absolute path của file proxy theo kind.
// Tự tạo folder Config/Proxy/ cạnh exe để tương thích với email.LoadTempMailProxies.
func proxyListPath(kind string) string {
	filename := ""
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "tempmail":
		filename = "proxy_tempmail.txt"
	case "gmail":
		filename = "proxy_rentmail.txt"
	default:
		return ""
	}
	return filepath.Join(appDataDir(), "Config", "Proxy", filename)
}

// GetDefaultUACounts trả về số UA embedded sẵn cho mỗi pool (iphone/android/chrome).
// Frontend hiển thị "X UA" khi textarea pool rỗng — cho user biết app có data mặc định.
func (a *App) GetDefaultUACounts() map[string]int {
	return map[string]int{
		"iphone":  fakeinfo.UAPoolSize(fakeinfo.UAKindIOS),     // FBIOS iPhone UAs
		"android": fakeinfo.UAPoolSize(fakeinfo.UAKindAndroid), // FB4A Android UAs
		"chrome":  53,                                          // Chrome versions (53 entries)
	}
}

// GetDefaultUAContent trả về toàn bộ nội dung UA pool mặc định cho 1 kind,
// join bằng "\n" — frontend fill vào textarea khi user chưa nhập/override.
// kind: "iphone" | "android" | "chrome".
// "chrome" hiện chưa có file embed riêng → trả về rỗng (fallback placeholder UI).
func (a *App) GetDefaultUAContent(kind string) string {
	var k fakeinfo.UAPoolKind
	switch kind {
	case "iphone":
		k = fakeinfo.UAKindIOS
	case "android":
		k = fakeinfo.UAKindAndroid
	default:
		return ""
	}
	list := fakeinfo.UAPoolAll(k)
	return strings.Join(list, "\n")
}

// GetDefaultCookiePaths trả về đường dẫn mặc định của cookie folder cho frontend hiển thị.
// Frontend dùng để show placeholder nếu user chưa chọn file cookie initial.
// GetDefaultCookiePaths trả về absolute paths (cạnh exe) cho cookie files.
// Tự seed data mẫu từ embedded (cookie_initial.txt + datr_pool.txt) nếu
// chưa tồn tại — user chạy máy mới là có pool datr sẵn sàng, không cần
// paste thủ công.
func (a *App) GetDefaultCookiePaths() map[string]string {
	initialPath := defaultCookieInitialPath()
	dir := filepath.Dir(initialPath)

	// Seed cả 2 file nếu chưa có — ship datr mẫu sẵn trong exe (embedded).
	cookie.SeedInitialIfMissing(initialPath)

	return map[string]string{
		"dir":     dir,
		"initial": initialPath,
	}
}

func defaultCookieInitialPath() string {
	absDir := filepath.Join(appDataDir(), cookie.DefaultDir)
	if err := os.MkdirAll(absDir, 0755); err == nil {
		return filepath.Join(absDir, cookie.InitialFilename)
	}
	return cookie.DefaultInitialPath()
}

func defaultCookieDir() string {
	absDir := filepath.Join(appDataDir(), cookie.DefaultDir)
	if err := os.MkdirAll(absDir, 0755); err == nil {
		return absDir
	}
	return cookie.DefaultDir
}

func resolveCookieInitialPath(path string) string {
	path = strings.Trim(strings.TrimSpace(path), `"'`)
	if path == "" {
		return defaultCookieInitialPath()
	}
	clean := filepath.Clean(path)
	isRootedNoVolume := filepath.VolumeName(clean) == "" &&
		(strings.HasPrefix(clean, `\`) || strings.HasPrefix(clean, `/`)) &&
		!strings.HasPrefix(clean, `\\`)
	if isRootedNoVolume {
		rel := strings.TrimLeft(clean, `\/`)
		if wd, err := os.Getwd(); err == nil {
			candidate := filepath.Join(wd, rel)
			if _, statErr := os.Stat(candidate); statErr == nil {
				return candidate
			}
		}
		return filepath.Join(appDataDir(), rel)
	}
	if filepath.IsAbs(clean) {
		return clean
	}
	if _, err := os.Stat(clean); err == nil {
		return clean
	}
	candidate := filepath.Join(appDataDir(), clean)
	if _, statErr := os.Stat(candidate); statErr == nil {
		return candidate
	}
	return clean
}

func countCookieInitialDatrLines(path string) (int, error) {
	path = resolveCookieInitialPath(path)
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		datr := cookie.ExtractDatr(line)
		if datr == "" {
			datr = line
		}
		if datr != "" && !strings.HasPrefix(datr, "_") && !strings.HasPrefix(datr, "-") {
			count++
		}
	}
	if err := sc.Err(); err != nil {
		return 0, err
	}
	return count, nil
}

func (a *App) GetCookieInitialStatus(path string) map[string]interface{} {
	resolved := defaultCookieInitialPath()
	status := map[string]interface{}{
		"path":   resolved,
		"exists": false,
		"count":  0,
		"error":  "",
	}
	info, err := os.Stat(resolved)
	if err != nil {
		status["error"] = "File không tồn tại: " + resolved
		return status
	}
	if info.IsDir() {
		status["error"] = "Đường dẫn là thư mục, cần chọn file: " + resolved
		return status
	}
	status["exists"] = true
	count, err := countCookieInitialDatrLines(resolved)
	if err != nil {
		status["error"] = err.Error()
		return status
	}
	status["count"] = count
	return status
}

// GetDatrPoolSize trả về số datr đang có trong in-memory pool (androidreg.SharedPool).
// Trả về 0 khi chưa có run nào khởi động pool.
func (a *App) GetDatrPoolSize() int {
	p := androidreg.SharedPool
	if p == nil {
		return 0
	}
	return p.Size()
}

// GetPoolFileSaveCount trả về số datr đã ghi vào Pool file trong run hiện tại.
// Reset về 0 mỗi khi bắt đầu RunRegister mới.
func (a *App) GetPoolFileSaveCount() int {
	return int(a.poolFileSaved.Load())
}

func (a *App) OpenCookieInitialFile(path string) string {
	resolved := defaultCookieInitialPath()
	if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
		return "Không tạo được thư mục: " + err.Error()
	}
	if _, err := os.Stat(resolved); os.IsNotExist(err) {
		if err := os.WriteFile(resolved, []byte(""), 0600); err != nil {
			return "Không tạo được file: " + err.Error()
		}
	}
	if err := exec.Command("notepad", resolved).Start(); err != nil {
		return "Không mở được file: " + err.Error()
	}
	return "OK"
}

// FbAppStatus — trạng thái dataset FB app version cho frontend.
// path: đường dẫn file override user đang dùng (hoặc default).
// count: số version đang active (default + override sau merge).
// overrideActive: true nếu file user có dữ liệu hợp lệ và đang override embed.
type FbAppStatus struct {
	Path           string `json:"path"`
	Count          int    `json:"count"`
	OverrideActive bool   `json:"overrideActive"`
}

// GetFbAppStatus trả về trạng thái hiện tại của FB versions dataset.
// Frontend hiển thị số version active + đường dẫn file để user biết có override chưa.
func (a *App) GetFbAppStatus() FbAppStatus {
	return FbAppStatus{
		Path:           fbdata.DefaultVersionsAndBuildsPath(),
		Count:          fbdata.Size(),
		OverrideActive: fbdata.OverrideActive(),
	}
}

// ReloadFbAppVersions ép nạp lại file override từ Config/Fbapp/versions_and_builds.txt.
// Gọi sau khi user save file qua UI hoặc edit file thủ công.
// Trả về count active sau reload — 0 nếu parse fail (fallback về default embed).
func (a *App) ReloadFbAppVersions() int {
	fbdata.Reload("")
	return fbdata.Size()
}

// SaveFbAppVersions ghi text user nhập từ UI vào Config/Fbapp/versions_and_builds.txt rồi reload.
// text: nội dung file, mỗi dòng "version|build".
// Trả về message: "OK (N versions)" nếu thành công, "Lỗi: ..." nếu fail.
func (a *App) SaveFbAppVersions(text string) string {
	if err := fbdata.EnsureDir(); err != nil {
		return "Lỗi tạo thư mục: " + err.Error()
	}
	path := fbdata.DefaultVersionsAndBuildsPath()
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		return "Lỗi ghi file: " + err.Error()
	}
	fbdata.Reload("")
	return fmt.Sprintf("OK (%d versions)", fbdata.Size())
}

// UAPoolStatus — trạng thái 1 UA pool cho frontend hiển thị.
type UAPoolStatus struct {
	Kind           string `json:"kind"`           // "android" | "ios" | "request"
	Path           string `json:"path"`           // Config/UserAgent/<kind>_UG.txt
	Count          int    `json:"count"`          // số UA active
	OverrideActive bool   `json:"overrideActive"` // đang dùng file user?
}

// GetUAPoolsStatus trả về trạng thái 3 pool cho frontend.
func (a *App) GetUAPoolsStatus() []UAPoolStatus {
	kinds := []fakeinfo.UAPoolKind{
		fakeinfo.UAKindAndroid,
		fakeinfo.UAKindIOS,
		fakeinfo.UAKindRequest,
		fakeinfo.UAKindWebChrome,
		fakeinfo.UAKindAndroidMess,
		fakeinfo.UAKindIOSMess,
	}
	out := make([]UAPoolStatus, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, UAPoolStatus{
			Kind:           string(k),
			Path:           fakeinfo.UAOverridePath(k),
			Count:          fakeinfo.UAPoolSize(k),
			OverrideActive: fakeinfo.UAPoolOverrideActive(k),
		})
	}
	return out
}

// SaveUAPool ghi user UA list vào Config/UserAgent/<kind>_UG.txt rồi reload.
// kind: "android" | "ios" | "request" | "webchrome".
// text: nội dung, mỗi dòng 1 UA.
// Trả về message "OK (N UA)" hoặc "Lỗi: ...".
func (a *App) SaveUAPool(kind string, text string) string {
	k := fakeinfo.UAPoolKind(kind)
	path := fakeinfo.UAOverridePath(k)
	if path == "" || filepath.Base(path) == "" {
		return fmt.Sprintf("Lỗi: kind %q không hợp lệ (phải là android|ios|request|webchrome|android_mess|ios_mess)", kind)
	}
	if err := fakeinfo.EnsureUAOverrideDir(); err != nil {
		return "Lỗi tạo thư mục: " + err.Error()
	}
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		return "Lỗi ghi file: " + err.Error()
	}
	fakeinfo.ReloadUAPools()
	return fmt.Sprintf("OK (%d UA)", fakeinfo.UAPoolSize(k))
}

// OpenUAFileInEditor mở file UA pool trong Notepad.
// poolKey: "android" | "iphone" | "request" — UI key khớp với form.uaPoolKey ở frontend.
func (a *App) OpenUAFileInEditor(poolKey string) string {
	k := uaKindFromPoolKey(poolKey)
	relPath := fakeinfo.UAOverridePath(k)
	if relPath == "" {
		return "pool không hợp lệ"
	}
	// Tạo file rỗng nếu chưa tồn tại để Notepad mở được ngay
	if err := fakeinfo.EnsureUAOverrideDir(); err == nil {
		if _, statErr := os.Stat(relPath); os.IsNotExist(statErr) {
			_ = os.WriteFile(relPath, []byte(""), 0644)
		}
	}
	absPath, err := filepath.Abs(relPath)
	if err != nil {
		return "không resolve được path: " + err.Error()
	}
	if err := exec.Command("notepad", absPath).Start(); err != nil {
		return "không mở được: " + err.Error()
	}
	return "OK"
}

// PhoneCountryInfo — 1 entry phone country cho frontend.
type PhoneCountryInfo struct {
	Name        string `json:"name"`
	CountryCode string `json:"countryCode"`
	PhoneCode   string `json:"phoneCode"`
	AreaCode    string `json:"areaCode"`
}

// GetPhoneCountries trả về toàn bộ danh sách phone countries.
// Dùng cho UI dropdown chọn country khi nhập số điện thoại.
func (a *App) GetPhoneCountries() []PhoneCountryInfo {
	list := fakeinfo.PhoneCountries()
	out := make([]PhoneCountryInfo, 0, len(list))
	for _, p := range list {
		out = append(out, PhoneCountryInfo{
			Name: p.Name, CountryCode: p.CountryCode,
			PhoneCode: p.PhoneCode, AreaCode: p.AreaCode,
		})
	}
	return out
}

// LookupPhoneCountry trả về info country từ ISO alpha-2 code.
// Trả về zero struct nếu không tìm thấy (Name = "" flag cho frontend).
func (a *App) LookupPhoneCountry(countryCode string) PhoneCountryInfo {
	p, ok := fakeinfo.LookupPhoneCode(countryCode)
	if !ok {
		return PhoneCountryInfo{}
	}
	return PhoneCountryInfo{
		Name: p.Name, CountryCode: p.CountryCode,
		PhoneCode: p.PhoneCode, AreaCode: p.AreaCode,
	}
}

// ImportLegacyConfig áp dụng cặp JSON general + interaction từ tool cũ vào AppSettings.
// Chỉ gọi sau khi user đã xác nhận ở wizard.
func (a *App) ImportLegacyConfig(generalJSON, interactionJSON string) string {
	var s adapter.LegacySettingsData
	var ic adapter.LegacyInteractionConfig

	if generalJSON != "" {
		if err := json.Unmarshal([]byte(generalJSON), &s); err != nil {
			return "Lỗi parse general.json: " + err.Error()
		}
	} else {
		// Giữ nguyên settings hiện tại
		a.settingsMu.RLock()
		s = adapter.ToLegacySettings(a.appSettings)
		a.settingsMu.RUnlock()
	}

	if interactionJSON != "" {
		if err := json.Unmarshal([]byte(interactionJSON), &ic); err != nil {
			return "Lỗi parse interaction.json: " + err.Error()
		}
	} else {
		// Giữ nguyên interaction hiện tại
		a.settingsMu.RLock()
		ic = adapter.ToLegacyInteraction(a.appSettings)
		a.settingsMu.RUnlock()
	}

	a.settingsMu.Lock()
	p := a.appSettings.GetActiveProfile()
	if p == nil {
		a.settingsMu.Unlock()
		return "Lỗi: không có profile active"
	}
	adapter.ApplySettingsToProfile(p, &a.appSettings.Global, s)
	adapter.ApplyInteractionToProfile(p, ic)
	app := a.appSettings
	a.settingsMu.Unlock()

	if err := appsettings.Save("Config/Settings", app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}
	return "OK"
}

// === Profile Management ===

// ProfileInfo — thông tin rút gọn của một profile trả về frontend
type ProfileInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// generateProfileID tạo unique ID cho profile mới
func generateProfileID() string {
	return fmt.Sprintf("p_%d", time.Now().UnixNano())
}

// ListProfiles trả về danh sách tất cả profiles
func (a *App) ListProfiles() []ProfileInfo {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	out := make([]ProfileInfo, len(a.appSettings.Profiles))
	for i, p := range a.appSettings.Profiles {
		out[i] = ProfileInfo{ID: p.ID, Name: p.Name}
	}
	return out
}

// GetActiveProfileID trả về ID của profile đang active
func (a *App) GetActiveProfileID() string {
	a.settingsMu.RLock()
	defer a.settingsMu.RUnlock()
	return a.appSettings.ActiveProfileID
}

// SetActiveProfile chuyển sang profile theo ID và lưu
func (a *App) SetActiveProfile(id string) string {
	a.settingsMu.Lock()
	found := false
	for _, p := range a.appSettings.Profiles {
		if p.ID == id {
			found = true
			break
		}
	}
	if !found {
		a.settingsMu.Unlock()
		return "Lỗi: profile không tồn tại"
	}
	a.appSettings.ActiveProfileID = id
	app := a.appSettings
	a.settingsMu.Unlock()
	if err := appsettings.Save("Config/Settings", app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}
	a.syncActiveProfileToFiles()
	return "OK"
}

// CreateProfile tạo profile mới từ default settings và kích hoạt nó
func (a *App) CreateProfile(name string) string {
	if name == "" {
		return "Lỗi: tên profile không được rỗng"
	}
	a.settingsMu.Lock()
	id := generateProfileID()
	p := model.DefaultProfile(id, name)
	a.appSettings.UpsertProfile(p)
	a.appSettings.ActiveProfileID = id
	app := a.appSettings
	a.settingsMu.Unlock()
	if err := appsettings.Save("Config/Settings", app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}
	a.syncActiveProfileToFiles()
	return id
}

// CloneProfile nhân bản profile đang active với tên mới và kích hoạt bản sao
func (a *App) CloneProfile(name string) string {
	if name == "" {
		return "Lỗi: tên profile không được rỗng"
	}
	a.settingsMu.Lock()
	src := a.appSettings.GetActiveProfile()
	if src == nil {
		a.settingsMu.Unlock()
		return "Lỗi: không có profile active"
	}
	cloned := *src
	cloned.ID = generateProfileID()
	cloned.Name = name
	a.appSettings.UpsertProfile(cloned)
	a.appSettings.ActiveProfileID = cloned.ID
	app := a.appSettings
	a.settingsMu.Unlock()
	if err := appsettings.Save("Config/Settings", app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}
	a.syncActiveProfileToFiles()
	return cloned.ID
}

// RenameProfile đổi tên profile theo ID
func (a *App) RenameProfile(id, name string) string {
	if name == "" {
		return "Lỗi: tên profile không được rỗng"
	}
	a.settingsMu.Lock()
	found := false
	for i := range a.appSettings.Profiles {
		if a.appSettings.Profiles[i].ID == id {
			a.appSettings.Profiles[i].Name = name
			found = true
			break
		}
	}
	if !found {
		a.settingsMu.Unlock()
		return "Lỗi: profile không tồn tại"
	}
	app := a.appSettings
	a.settingsMu.Unlock()
	if err := appsettings.Save("Config/Settings", app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}
	return "OK"
}

// DeleteProfile xóa profile theo ID — không thể xóa nếu chỉ còn 1
func (a *App) DeleteProfile(id string) string {
	a.settingsMu.Lock()
	if len(a.appSettings.Profiles) <= 1 {
		a.settingsMu.Unlock()
		return "Lỗi: phải giữ ít nhất 1 profile"
	}
	newProfiles := make([]model.Profile, 0, len(a.appSettings.Profiles)-1)
	for _, p := range a.appSettings.Profiles {
		if p.ID != id {
			newProfiles = append(newProfiles, p)
		}
	}
	if len(newProfiles) == len(a.appSettings.Profiles) {
		a.settingsMu.Unlock()
		return "Lỗi: profile không tồn tại"
	}
	a.appSettings.Profiles = newProfiles
	if a.appSettings.ActiveProfileID == id {
		a.appSettings.ActiveProfileID = newProfiles[0].ID
	}
	app := a.appSettings
	a.settingsMu.Unlock()
	if err := appsettings.Save("Config/Settings", app); err != nil {
		return "Lỗi lưu: " + err.Error()
	}
	return "OK"
}

// syncActiveProfileToFiles ghi settings + interaction config của profile đang active
// xuống general.json và interaction.json, để LoadSettings/LoadInteractionConfig
// đọc đúng dữ liệu sau khi switch profile.
// Caller phải KHÔNG giữ settingsMu khi gọi hàm này.
func (a *App) syncActiveProfileToFiles() {
	const settingsDir = "Config/Settings"
	_ = os.MkdirAll(settingsDir, 0755)

	a.settingsMu.RLock()
	ls := adapter.ToLegacySettings(a.appSettings)
	p := a.appSettings.GetActiveProfile()
	var profileInteraction []byte
	if p != nil && len(p.Interaction) > 0 {
		profileInteraction = []byte(p.Interaction)
	}
	lic := adapter.ToLegacyInteraction(a.appSettings)
	a.settingsMu.RUnlock()

	// Ghi general.json
	var settings SettingsData
	if b, err := json.Marshal(ls); err == nil {
		if err := json.Unmarshal(b, &settings); err != nil {
			slog.Warn("syncSettingsFiles: unmarshal general thất bại", "err", err)
		}
	}
	if b, err := json.MarshalIndent(settings, "", "  "); err == nil {
		if err := os.WriteFile(filepath.Join(settingsDir, "general.json"), b, 0644); err != nil {
			slog.Warn("syncSettingsFiles: ghi general.json thất bại", "err", err)
		}
	}

	// Ghi interaction.json — ưu tiên profile.Interaction, fallback adapter
	if len(profileInteraction) > 0 {
		// Profile đã có interaction riêng → ghi thẳng
		if err := os.WriteFile(filepath.Join(settingsDir, "interaction.json"), profileInteraction, 0644); err != nil {
			slog.Warn("syncSettingsFiles: ghi interaction.json thất bại", "err", err)
		}
	} else {
		// Profile chưa lưu interaction → dùng adapter (backward compat)
		var interaction InteractionConfig
		if b, err := json.Marshal(lic); err == nil {
			if err := json.Unmarshal(b, &interaction); err != nil {
				slog.Warn("syncSettingsFiles: unmarshal interaction thất bại", "err", err)
			}
		}
		if b, err := json.MarshalIndent(interaction, "", "  "); err == nil {
			if err := os.WriteFile(filepath.Join(settingsDir, "interaction.json"), b, 0644); err != nil {
				slog.Warn("syncSettingsFiles: ghi interaction.json thất bại", "err", err)
			}
		}
	}
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

// resetRegStats — khởi tạo lại bảng thống kê reg cho run mới (seed sẵn các version đã chọn
// để tab hiện đủ dòng kể cả khi version đó chưa có account nào).
func (a *App) resetRegStats(platforms []string) {
	a.regStatsMu.Lock()
	a.regStats = make(map[string]*regVersionStat, len(platforms)+1)
	for _, p := range platforms {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := a.regStats[p]; !ok {
			a.regStats[p] = &regVersionStat{}
		}
	}
	a.regStatsMu.Unlock()
}

// recordRegOutcome — ghi nhận 1 lần reg của version `platform` là thành công hay thất bại.
func (a *App) recordRegOutcome(platform string, success bool) {
	platform = strings.TrimSpace(platform)
	if platform == "" {
		return
	}
	a.regStatsMu.Lock()
	if a.regStats == nil {
		a.regStats = make(map[string]*regVersionStat)
	}
	st := a.regStats[platform]
	if st == nil {
		st = &regVersionStat{}
		a.regStats[platform] = st
	}
	if success {
		st.Success++
	} else {
		st.Fail++
	}
	a.regStatsMu.Unlock()
}

// GetRegStats — thống kê reg theo version cho run hiện tại (hoặc run gần nhất nếu đã dừng).
// Frontend poll mỗi ~10s. Sort theo platform để thứ tự dòng ổn định.
func (a *App) GetRegStats() []RegStatRow {
	a.regStatsMu.Lock()
	rows := make([]RegStatRow, 0, len(a.regStats))
	for k, st := range a.regStats {
		rows = append(rows, RegStatRow{
			Platform: k,
			Success:  st.Success,
			Fail:     st.Fail,
			Total:    st.Success + st.Fail,
		})
	}
	a.regStatsMu.Unlock()
	sort.Slice(rows, func(i, j int) bool { return rows[i].Platform < rows[j].Platform })
	for i := range rows {
		rows[i].Index = i + 1
	}
	return rows
}

// === Verify stats (song song với reg stats ở trên) ===

// verifyPlatformDisplayName chuyển internal platform ID → tên hiển thị cho thống kê.
// Đảm bảo stats luôn dùng cùng key với UI label người dùng đã chọn.
// sXXX platforms giữ nguyên vì tên internal đã khớp với label UI.
func verifyPlatformDisplayName(internal string) string {
	switch internal {
	case instagram.PlatformWeb:
		return "api mfb"
	case instagram.PlatformWebAndroid:
		return "api web andr"
	case instagram.PlatformS23:
		return "api android"
	case instagram.PlatformAndroid:
		return "api token"
	default:
		return internal
	}
}

func (a *App) resetVerifyStats(platforms []string) {
	a.verifyStatsMu.Lock()
	a.verifyStats = make(map[string]*regVersionStat, len(platforms)+1)
	for _, p := range platforms {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Resolve UI key → internal → display name để match với recordVerifyOutcome.
		// VD: "api web andr" → "webandroid" → "api web andr", "s561v99" → "s561v3".
		if resolved := verifyPlatformFromType(p); resolved != "" {
			p = verifyPlatformDisplayName(resolved)
		}
		if _, ok := a.verifyStats[p]; !ok {
			a.verifyStats[p] = &regVersionStat{}
		}
	}
	a.verifyStatsMu.Unlock()
	a.verifyPlatformRR.Store(0)
}

func (a *App) recordVerifyOutcome(platform string, success bool) {
	platform = verifyPlatformDisplayName(strings.TrimSpace(platform))
	if platform == "" {
		return
	}
	a.verifyStatsMu.Lock()
	if a.verifyStats == nil {
		a.verifyStats = make(map[string]*regVersionStat)
	}
	st := a.verifyStats[platform]
	if st == nil {
		st = &regVersionStat{}
		a.verifyStats[platform] = st
	}
	if success {
		st.Success++
	} else {
		st.Fail++
	}
	a.verifyStatsMu.Unlock()
}

// GetVerifyStats — thống kê verify theo version cho run hiện tại (hoặc run gần nhất).
func (a *App) GetVerifyStats() []RegStatRow {
	a.verifyStatsMu.Lock()
	rows := make([]RegStatRow, 0, len(a.verifyStats))
	for k, st := range a.verifyStats {
		rows = append(rows, RegStatRow{
			Platform: k,
			Success:  st.Success,
			Fail:     st.Fail,
			Total:    st.Success + st.Fail,
		})
	}
	a.verifyStatsMu.Unlock()
	sort.Slice(rows, func(i, j int) bool { return rows[i].Platform < rows[j].Platform })
	for i := range rows {
		rows[i].Index = i + 1
	}
	return rows
}

// === Mail domain stats ===

func (a *App) resetMailDomainStats() {
	a.mailDomainStatsMu.Lock()
	a.mailDomainStats = make(map[string]*mailDomainStat)
	a.mailDomainStatsMu.Unlock()
}

func (a *App) recordMailDomainOutcome(email string, isLive bool) {
	at := strings.Index(email, "@")
	if at < 0 {
		return
	}
	domain := strings.ToLower(strings.TrimSpace(email[at:]))
	if domain == "" {
		return
	}
	a.mailDomainStatsMu.Lock()
	if a.mailDomainStats == nil {
		a.mailDomainStats = make(map[string]*mailDomainStat)
	}
	st := a.mailDomainStats[domain]
	if st == nil {
		st = &mailDomainStat{}
		a.mailDomainStats[domain] = st
	}
	st.Veri++
	if isLive {
		st.Live++
	}
	a.mailDomainStatsMu.Unlock()
}

// GetMailDomainStats — thống kê verify theo domain mail, sort theo Live desc.
func (a *App) GetMailDomainStats() []MailDomainStatRow {
	a.mailDomainStatsMu.Lock()
	rows := make([]MailDomainStatRow, 0, len(a.mailDomainStats))
	for domain, st := range a.mailDomainStats {
		die := st.Veri - st.Live
		if die < 0 {
			die = 0
		}
		rate := 0.0
		if st.Veri > 0 {
			rate = float64(st.Live) / float64(st.Veri)
		}
		rows = append(rows, MailDomainStatRow{
			Domain: domain,
			Veri:   st.Veri,
			Live:   st.Live,
			Die:    die,
			Rate:   rate,
		})
	}
	a.mailDomainStatsMu.Unlock()
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Live != rows[j].Live {
			return rows[i].Live > rows[j].Live
		}
		return rows[i].Domain < rows[j].Domain
	})
	for i := range rows {
		rows[i].Index = i + 1
	}
	return rows
}

// === Build UA stats (thống kê FBAV version theo Reg / Veri) ===

func extractFBAV(ua string) string {
	idx := strings.Index(ua, "FBAV/")
	if idx == -1 {
		return ""
	}
	rest := ua[idx+5:]
	if end := strings.IndexAny(rest, ";]"); end != -1 {
		return rest[:end]
	}
	return rest
}

func (a *App) resetBuildUAStats() {
	a.buildUARegStatsMu.Lock()
	a.buildUARegStats = make(map[string]*regVersionStat)
	a.buildUARegStatsMu.Unlock()
	a.buildUAVerStatsMu.Lock()
	a.buildUAVerStats = make(map[string]*regVersionStat)
	a.buildUAVerStatsMu.Unlock()
}

func (a *App) recordBuildUARegVersion(fbav string, success bool) {
	if fbav == "" {
		return
	}
	a.buildUARegStatsMu.Lock()
	if a.buildUARegStats == nil {
		a.buildUARegStats = make(map[string]*regVersionStat)
	}
	st := a.buildUARegStats[fbav]
	if st == nil {
		st = &regVersionStat{}
		a.buildUARegStats[fbav] = st
	}
	if success {
		st.Success++
	} else {
		st.Fail++
	}
	a.buildUARegStatsMu.Unlock()
}

func (a *App) recordBuildUAVerVersion(fbav string, success bool) {
	if fbav == "" {
		return
	}
	a.buildUAVerStatsMu.Lock()
	if a.buildUAVerStats == nil {
		a.buildUAVerStats = make(map[string]*regVersionStat)
	}
	st := a.buildUAVerStats[fbav]
	if st == nil {
		st = &regVersionStat{}
		a.buildUAVerStats[fbav] = st
	}
	if success {
		st.Success++
	} else {
		st.Fail++
	}
	a.buildUAVerStatsMu.Unlock()
}

func (a *App) GetBuildUARegStats() []RegStatRow {
	a.buildUARegStatsMu.Lock()
	rows := make([]RegStatRow, 0, len(a.buildUARegStats))
	for k, st := range a.buildUARegStats {
		rows = append(rows, RegStatRow{Platform: k, Success: st.Success, Fail: st.Fail, Total: st.Success + st.Fail})
	}
	a.buildUARegStatsMu.Unlock()
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Total != rows[j].Total {
			return rows[i].Total > rows[j].Total
		}
		return rows[i].Platform < rows[j].Platform
	})
	for i := range rows {
		rows[i].Index = i + 1
	}
	return rows
}

func (a *App) GetBuildUAVerStats() []RegStatRow {
	a.buildUAVerStatsMu.Lock()
	rows := make([]RegStatRow, 0, len(a.buildUAVerStats))
	for k, st := range a.buildUAVerStats {
		rows = append(rows, RegStatRow{Platform: k, Success: st.Success, Fail: st.Fail, Total: st.Success + st.Fail})
	}
	a.buildUAVerStatsMu.Unlock()
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Total != rows[j].Total {
			return rows[i].Total > rows[j].Total
		}
		return rows[i].Platform < rows[j].Platform
	})
	for i := range rows {
		rows[i].Index = i + 1
	}
	return rows
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
	candidate := filepath.Join(appDataDir(), "result")
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

// FetchWeMakeMailDomains gọi API wemakemail.com và trả danh sách domain của tài khoản.
// Trả chuỗi JSON {"plan":"business","free":[...],"paid":[...],"all":[...]} hoặc "ERROR: ..." nếu thất bại.
func (a *App) FetchWeMakeMailDomains(apiKey string) string {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	result, err := emailtemp.FetchWeMakeMailDomains(ctx, apiKey)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	b, _ := json.Marshal(result)
	return string(b)
}

// FetchVietXFDomains gọi API vietxf.com và trả danh sách domain của tài khoản.
// Trả chuỗi JSON {"domains":[...]} hoặc "ERROR: ..." nếu thất bại.
func (a *App) FetchVietXFDomains(apiKey string) string {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	result, err := emailtemp.FetchVietXFDomains(ctx, apiKey)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	b, _ := json.Marshal(result)
	return string(b)
}

// FetchStore1sProducts gọi store1s.com products.php QUA BACKEND (tránh CORS từ webview)
// và trả JSON [{id,name,price,stock}] hoặc "ERROR: ...". Dùng cho dropdown + check tồn kho.
func (a *App) FetchStore1sProducts(apiKey string) string {
	ctx, cancel := context.WithTimeout(a.ctx, 20*time.Second)
	defer cancel()
	products, err := emailrent.FetchStore1sProducts(ctx, apiKey)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	b, _ := json.Marshal(products)
	return string(b)
}

// FetchMailHVDomains gọi API dulich360.com và trả danh sách domain của tài khoản.
// Trả chuỗi JSON {"plan":"...","free":[...],"paid":[...],"all":[...]} hoặc "ERROR: ..." nếu thất bại.
func (a *App) FetchMailHVDomains(apiKey string) string {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()
	result, err := emailtemp.FetchMailHVDomains(ctx, apiKey)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	b, _ := json.Marshal(result)
	return string(b)
}

// OpenVersionsAndBuildsFile mở file versions_and_builds_<kind>.txt trong text editor mặc định.
// kind: "reg" → versions_and_builds_reg.txt, "ver" → versions_and_builds_ver.txt, "" → versions_and_builds.txt.
// Auto-tạo file rỗng nếu chưa tồn tại. Trả "OK" hoặc "ERR|..." để frontend hiện thông báo.
func (a *App) OpenVersionsAndBuildsFile(kind string) string {
	var path string
	switch kind {
	case "reg-ios":
		path = filepath.Join("Config", "DeviceInfoIOS", "ios_app_builds_reg.txt")
	case "ver-ios":
		path = filepath.Join("Config", "DeviceInfoIOS", "ios_app_builds_ver.txt")
	case "reg":
		path = filepath.Join("Config", "DeviceInfo", "versions_and_builds_reg.txt")
	case "ver":
		path = filepath.Join("Config", "DeviceInfo", "versions_and_builds_ver.txt")
	default:
		path = filepath.Join("Config", "DeviceInfo", "versions_and_builds.txt")
	}

	// Ensure parent dir + file tồn tại (auto-tạo rỗng nếu chưa có).
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "ERR|Không tạo được thư mục: " + err.Error()
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.WriteFile(path, nil, 0644); err != nil {
			return "ERR|Không tạo được file: " + err.Error()
		}
	}

	// Absolute path để OS open chính xác (start "" relative path đôi khi resolve sai).
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}

	// Open Explorer ở folder chứa file + HIGHLIGHT file đó:
	// - Windows: explorer.exe /select,<file>
	// - macOS:   open -R <file>  (Reveal in Finder)
	// - Linux:   xdg-open <folder>  (chỉ mở folder, không highlight được)
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", "/select,"+abs)
	case "darwin":
		cmd = exec.Command("open", "-R", abs)
	default:
		cmd = exec.Command("xdg-open", filepath.Dir(abs))
	}
	if err := cmd.Start(); err != nil {
		return "ERR|Không mở được file: " + err.Error()
	}
	go func() { _ = cmd.Wait() }()
	return "OK"
}

// ExportFbVersionPool ghi danh sách FBAV được chọn (với FBBV lookup từ pool) ra file
// trong thư mục result hiện tại + mở Explorer highlight file đó.
//
// kind: "reg" / "ver" — phục vụ naming file output.
// fbavList: list FBAV string user đã chọn trong UI table (vd ["564.0.0.0.17", "563.0.0.42.67"]).
//
// File output: <resultDir>/versions_and_builds_<kind>_<timestamp>.txt với format FBAV|FBBV mỗi dòng.
// Nếu chưa có session đang chạy (resultDir rỗng) → fallback Config/DeviceInfo/.
//
// Trả "OK|<num_written>" hoặc "ERR|<msg>".
func (a *App) ExportFbVersionPool(kind string, fbavList []string) string {
	if len(fbavList) == 0 {
		return "ERR|Chưa chọn FBAV nào"
	}
	if kind != "reg" && kind != "ver" {
		return "ERR|kind phải là 'reg' hoặc 'ver'"
	}

	// Lookup FBBV cho từng FBAV.
	versionMap := a.GetFbVersionMap()
	lines := make([]string, 0, len(fbavList))
	missing := 0
	for _, fbav := range fbavList {
		fbbv, ok := versionMap[fbav]
		if !ok || fbbv == "" {
			missing++
			continue
		}
		lines = append(lines, fbav+"|"+fbbv)
	}
	if len(lines) == 0 {
		return "ERR|Không tìm thấy FBBV cho bất kỳ FBAV nào trong pool"
	}

	// Chọn target dir: ưu tiên result session hiện tại, fallback Config/DeviceInfo/.
	targetDir := a.resultDir()
	if targetDir == "" {
		targetDir = filepath.Join("Config", "DeviceInfo")
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "ERR|Không tạo được thư mục: " + err.Error()
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("versions_and_builds_%s_%s.txt", kind, timestamp)
	outPath := filepath.Join(targetDir, filename)

	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return "ERR|Không ghi được file: " + err.Error()
	}

	// Open Explorer + highlight file (giống OpenVersionsAndBuildsFile).
	abs, err := filepath.Abs(outPath)
	if err != nil {
		abs = outPath
	}
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", "/select,"+abs)
	case "darwin":
		cmd = exec.Command("open", "-R", abs)
	default:
		cmd = exec.Command("xdg-open", filepath.Dir(abs))
	}
	if err := cmd.Start(); err == nil {
		go func() { _ = cmd.Wait() }()
	}

	msg := fmt.Sprintf("OK|%d", len(lines))
	if missing > 0 {
		msg += fmt.Sprintf("|missing=%d", missing)
	}
	return msg
}

// GetFbVersionMap trả map FBAV → FBBV từ pool CHUNG (+ split _reg/_ver nếu có).
// Dùng cho UI RegStatsTable export → lookup FBBV theo FBAV đã thống kê.
// Format trả JSON-marshallable: {"564.0.0.0.17": "977893103", ...}.
func (a *App) GetFbVersionMap() map[string]string {
	out := make(map[string]string)
	// Pool chung + 2 pool split — gom hết, FBBV mới override cũ nếu trùng FBAV.
	for _, pool := range [][]fbdata.FbVersion{
		fbdata.Versions(),
		fbdata.VersionsReg(),
		fbdata.VersionsVer(),
	} {
		for _, v := range pool {
			if v.Version != "" && v.Build != "" {
				out[v.Version] = v.Build
			}
		}
	}
	return out
}

// pickFbVersionByKind pick cặp FBAV/FBBV theo kind ("reg"/"ver"/""):
//   - "reg" → versions_and_builds_reg.txt (fallback chung)
//   - "ver" → versions_and_builds_ver.txt (fallback chung)
//   - ""    → versions_and_builds.txt (chung)
func pickFbVersionByKind(kind string) (string, string) {
	switch kind {
	case "reg":
		return fakeinfo.RandomFbVersionReg()
	case "ver":
		return fakeinfo.RandomFbVersionVer()
	default:
		return fakeinfo.RandomFbVersion()
	}
}

// SimulatePlatformUA giả lập 1 UA như khi chạy thực, dùng để xem trước trong popup cài đặt.
// Trả "{note}\n{ua}" — frontend tách dòng đầu làm label, dòng sau làm code display.
// platform có thể là: UI verify string ("api mfb", "api web andr", "api android", "api token")
// HOẶC platform ID nội bộ ("web", "webandroid", "s23", "s561v99", ...).
// Resolve UI string về platform nội bộ trước khi dispatch để switch case match đúng.
func (a *App) SimulatePlatformUA(platform string, uaCfg PlatformUAConfig) string {
	// Resolve UI verify type → internal platform ID
	// CHỈ resolve khi input là UI dropdown string (có space, vd "api android" / "api token" / "api mfb" / "api web andr").
	// Platform ID nội bộ (s518, s564v1s23, webandroid, ...) giữ nguyên — verifyPlatformFromType default
	// trả PlatformWeb cho mọi case không match → sẽ dispatch nhầm sang WebChrome pool gây ra UA iPhone.
	if strings.Contains(platform, " ") {
		if resolved := verifyPlatformFromType(platform); resolved != "" {
			platform = resolved
		}
	}
	countries := []string{"VN", "US", "PH", "ID", "TH", "MY"}
	countryCode := countries[rand.Intn(len(countries))]
	maybeXID := func(ua string) string {
		if uaCfg.TrackingID && ua != "" {
			return appendXIDToUA(ua)
		}
		return ua
	}

	// iOS native (FBIOS) — UseOriginalUA ưu tiên trước: trả UA gốc cố định theo version.
	// Nếu không tick UA Gốc → random từ device pool iOS (FBAN/FBIOS).
	// Early-return ở đây để không rơi xuống nhánh BuildAndroidUA (bug hiện UA FB4A).
	if verifyIsIOS(platform) {
		var ua string
		note := "iOS FBIOS"
		if uaCfg.UseOriginalUA {
			ua = originalUAForPlatform(platform, "")
			note += " · UA Gốc"
		}
		if ua == "" {
			ua = instagram.PlatformVerifyUA(platform, countryCode)
			note += " · " + countryCode
		}
		if ua == "" {
			return "[iOS UA không khả dụng]\nPlatform \"" + platform + "\" chưa đăng ký VerifyUA (FBIOS)."
		}
		if uaCfg.Kind != "" {
			note += " · " + uaCfg.Kind
		}
		if uaCfg.TrackingID {
			note += " · +XID"
		}
		return note + "\n" + maybeXID(ua)
	}

	if uaCfg.UseOriginalUA {
		origCC := countryCode
		note := "UA Gốc · random device · FBCR theo " + countryCode
		if !uaCfg.ReplaceCarrier {
			origCC = ""
			note = "UA Gốc · random device · giữ FBCR/Viettel default"
		}
		ua := originalUAForPlatform(platform, origCC)
		if ua == "" {
			// Fallback: platform random UA per-account (vd s273) — dùng factory.
			// Truyền origCC (= "" khi user không tick "Thay nhà mạng") để factory
			// hiểu được intent: giữ default carrier thay vì random theo country.
			if factoryUA := instagram.PlatformVerifyUA(platform, origCC); factoryUA != "" {
				ua = factoryUA
			}
		}
		if ua == "" {
			return "[UA Gốc không khả dụng]\nPlatform \"" + platform + "\" không có UA gốc cố định."
		}
		if uaCfg.TrackingID {
			note += " · +XID"
		}
		return note + "\n" + maybeXID(ua)
	}

	if uaCfg.BuildUA {
		// Web-based platforms cần Chrome browser UA, không phải FBAN/FB4A.
		switch platform {
		case instagram.PlatformWeb:
			// Đọc từ WebChrome pool (Config/UserAgent/WebChrome_UA.txt) — user-managed.
			ua := fakeinfo.RandomUAFromPool(fakeinfo.UAKindWebChrome)
			size := fakeinfo.UAPoolSize(fakeinfo.UAKindWebChrome)
			if ua == "" {
				return fmt.Sprintf("[WebChrome pool rỗng]\nPaste UA vào %s rồi reload pool.", fakeinfo.UAOverridePath(fakeinfo.UAKindWebChrome))
			}
			note := fmt.Sprintf("Build UA · WebChrome pool (%d UA)", size)
			if uaCfg.TrackingID {
				note += " · +XID"
			}
			return note + "\n" + maybeXID(ua)
		case instagram.PlatformWebAndroid:
			prof := fakeinfo.RandomChromeAndroidProfile()
			note := "Build UA · Chrome Mobile · Android " + prof.AndroidOsVersion
			if uaCfg.TrackingID {
				note += " · +XID"
			}
			return note + "\n" + maybeXID(prof.UserAgent)
		}

		// Default: FB4A native Android UA cho s5xx/s4xx/android/s23/...
		dev := fakeinfo.RandomDeviceProfile()
		locale := fakeinfo.LocaleFromCountry(countryCode)
		sim := fakeinfo.RandomSimProfile(countryCode)
		carrier := sim.OperatorName
		if carrier == "" {
			carrier = fakeinfo.RandomCarrier()
		}
		fbVer, fbBuild := pickFbVersionByKind(uaCfg.Kind)
		ua := fakeinfo.BuildAndroidUAWithOpts(dev, locale, carrier, fbVer, fbBuild, uaCfg.AddVirtualSpecAndroid, false)
		note := "Build UA · " + countryCode + " · " + carrier
		if uaCfg.Kind != "" {
			note += " · pool=" + uaCfg.Kind
		}
		if uaCfg.AddVirtualSpecAndroid {
			note += " · +Dalvik"
		}
		if uaCfg.TrackingID {
			note += " · +XID"
		}
		return note + "\n" + maybeXID(ua)
	}

	// Pool UA
	// FIX: ưu tiên per-platform uaCfg.UaPoolKey (set qua applyBuildUADefault frontend
	// theo từng platform — vd ios562 → "iphone"). Chỉ fallback global cfg.UaPoolKey
	// khi per-platform rỗng (legacy hoặc chưa init).
	poolKey := uaCfg.UaPoolKey
	if poolKey == "" {
		cfg := a.LoadInteractionConfig()
		poolKey = cfg.UaPoolKey
	}
	kind := uaKindFromPoolKey(poolKey)
	ua := fakeinfo.RandomUAFromPool(kind)
	if ua == "" {
		dev := fakeinfo.RandomDeviceProfile()
		locale := fakeinfo.LocaleFromCountry(countryCode)
		carrier := fakeinfo.RandomCarrier()
		fbVer, fbBuild := pickFbVersionByKind(uaCfg.Kind)
		ua = fakeinfo.BuildAndroidUAWithOpts(dev, locale, carrier, fbVer, fbBuild, uaCfg.AddVirtualSpecAndroid, false)
		note := "Pool rỗng → build random · " + countryCode
		if uaCfg.TrackingID {
			note += " · +XID"
		}
		return note + "\n" + maybeXID(ua)
	}
	if uaCfg.AddVirtualSpecAndroid {
		ua = fakeinfo.WrapWithDalvikPrefix(ua)
	}
	note := "Pool " + string(kind)
	if uaCfg.AddVirtualSpecAndroid {
		note += " · +Dalvik"
	}
	if uaCfg.TrackingID {
		note += " · +XID"
	}
	return note + "\n" + maybeXID(ua)
}

// originalUAForPlatformWithSim trả UA gốc + SIM được dùng cho FBCR thay thế.
// Caller dùng SIM này để override profile.Sim trong register/verify → đảm bảo
// HNI/MCC/MNC trong body/headers khớp với FBCR carrier trong UA. Nếu không
// nhất quán: UA nói "Viettel" nhưng body có HNI Vinaphone → fingerprint mismatch.
func originalUAForPlatformWithSim(platform, countryCode string) (string, fakeinfo.SimProfile) {
	base := originalUABaseForPlatform(platform)
	if base == "" {
		return "", fakeinfo.SimProfile{}
	}
	var sim fakeinfo.SimProfile
	if countryCode != "" {
		sim = fakeinfo.RandomSimProfile(countryCode)
		if sim.OperatorName != "" {
			base = reFBCR.ReplaceAllString(base, "FBCR/"+sim.OperatorName)
		}
	}
	return base, sim
}

// mapSimNetworkType convert GUI value → Xfb_connection_type (port C# MainFormUISettings.SimNetworkType).
// 0=WIFI, 1=mobile.LTE, 2=cell.CTRadioAccessTechnologyHSDPA, 3=unknown.
// Input có thể là "WIFI", "LTE", "HSDPA", "unknown", "3G", "4G"... Trả "" để giữ default.
func mapSimNetworkType(t string) string {
	t = strings.TrimSpace(t)
	switch strings.ToUpper(t) {
	case "", "0":
		return "" // không override
	case "WIFI":
		return "WIFI"
	case "LTE", "4G", "MOBILE.LTE", "1":
		return "mobile.LTE"
	case "HSDPA", "3G", "CELL.CTRADIOACCESSTECHNOLOGYHSDPA", "2":
		return "cell.CTRadioAccessTechnologyHSDPA"
	case "UNKNOWN", "3":
		return "unknown"
	}
	return ""
}

// defaultPermanentDir trả về folder chứa permanent phone.txt + mail.txt.
// Port C# PathSingleton.PermanentPhonexMailFolder. Auto-create nếu chưa có.
func defaultPermanentDir() string {
	candidate := filepath.Join(appDataDir(), "Config", "Permanent")
	if mkErr := os.MkdirAll(candidate, 0755); mkErr == nil {
		return candidate
	}
	homeDir, _ := os.UserHomeDir()
	fallback := filepath.Join(homeDir, "Documents", "HVR", "Config", "Permanent")
	_ = os.MkdirAll(fallback, 0755)
	return fallback
}

// missingPhoneCCMu serialize append vào missing-country-codes log từ nhiều reg goroutines.
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
	candidate := filepath.Join(appDataDir(), "Config", "phone_database")
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

// GetPermanentFileCounts trả về số dòng trong phone.txt và mail.txt.
// Frontend dùng để hiển thị số lượng data đã tích lũy.
func (a *App) GetPermanentFileCounts() map[string]int {
	permDir := defaultPermanentDir()
	result := map[string]int{"phone": 0, "mail": 0}
	for _, key := range []string{"phone", "mail"} {
		data, err := os.ReadFile(filepath.Join(permDir, key+".txt"))
		if err != nil {
			continue
		}
		count := 0
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.TrimSpace(line) != "" {
				count++
			}
		}
		result[key] = count
	}
	return result
}

// setupLogging — cấu hình slog ghi vào thư mục logs/ cạnh file exe.
func setupLogging() {
	logDir := filepath.Join(appDataDir(), "logs")
	_ = os.MkdirAll(logDir, 0755)
	logFile := filepath.Join(logDir, "run-"+time.Now().Format("20060102")+".log")

	// Log rotation cho 24/7: nếu file > 10 MB, rotate sang .N (keep 3 backups).
	// Cap tổng log history ~40 MB thay vì grow unbounded.
	rotateIfLarge(logFile, 10<<20, 3)

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return // fallback: slog default (stderr)
	}
	handler := slog.NewTextHandler(f, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
	slog.Info("app started", "version", "0.1.0")
}

// rotateIfLarge check file size → nếu vượt maxBytes thì rotate (file → file.1, file.1 → file.2...).
// keep: số file backup tối đa. File cũ nhất bị xoá.
func rotateIfLarge(path string, maxBytes int64, keep int) {
	info, err := os.Stat(path)
	if err != nil || info.Size() < maxBytes {
		return
	}
	lastBackup := fmt.Sprintf("%s.%d", path, keep)
	_ = os.Remove(lastBackup)
	for i := keep - 1; i >= 1; i-- {
		src := fmt.Sprintf("%s.%d", path, i)
		dst := fmt.Sprintf("%s.%d", path, i+1)
		_ = os.Rename(src, dst)
	}
	_ = os.Rename(path, path+".1")
}

// extractCUserFromCookie lấy UID Facebook từ chuỗi cookie bằng cách tìm c_user=VALUE.
// cookie: chuỗi cookie dạng "name1=value1; c_user=123456789; name3=value3".
// Trả về UID (ví dụ "123456789") hoặc "" nếu không có c_user.
// Dùng trong autoDetectAccount khi account chỉ có cookie (không có UID riêng).
func extractCUserFromCookie(cookie string) string {
	for _, pair := range strings.Split(cookie, ";") {
		pair = strings.TrimSpace(pair)
		if strings.HasPrefix(pair, "c_user=") {
			return strings.TrimPrefix(pair, "c_user=")
		}
	}
	return ""
}
