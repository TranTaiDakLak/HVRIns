// Package runner — FIFO worker pool cho chạy verify song song
// Mapping từ WeBM frmFacebook.Core.cs Run() FIFO Scheduler
package runner

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"HVRIns/internal/cookie"
	"HVRIns/internal/igcore"
	"HVRIns/internal/instagram"
	"HVRIns/internal/instagram/fakeinfo"
	"HVRIns/internal/proxy"
)

// AccountInput dữ liệu 1 account cần verify
type AccountInput struct {
	ID             int    `json:"id"`
	UID            string `json:"uid"`
	Username       string `json:"username,omitempty"` // Instagram username (vd "falcon.3900382")
	FullName       string `json:"fullName"`
	Phone          string `json:"phone"`
	Cookie         string `json:"cookie"`
	Token          string `json:"token"`
	UserAgent      string `json:"userAgent"`
	Proxy          string `json:"proxy"`
	Password       string `json:"password"`
	InputAccount   string `json:"inputAccount"`
	DeviceID       string `json:"deviceId"`       // device_id từ lúc reg — dùng lại khi verify
	FamilyDeviceID string `json:"familyDeviceId"` // family_device_id từ lúc reg

	// iOS partial reg tokens — truyền sang verify confirm (ios562).
	Srnonce               string `json:"srnonce,omitempty"`
	SessionlessCryptedUID string `json:"sessionlessCryptedUID,omitempty"`

	// iOS Messenger verify — AAC session + flow IDs + password info từ create.
	// add-mail cần encrypted_password đúng (PassRaw+PassTS) + AAC/flow context khớp create.
	AACJid        string `json:"aacJid,omitempty"`
	AACcs         string `json:"aacCs,omitempty"`
	AACts         string `json:"aacTs,omitempty"`
	RegFlowID     string `json:"regFlowId,omitempty"`
	HeadersFlowID string `json:"headersFlowId,omitempty"`
	PassRaw       string `json:"passRaw,omitempty"`
	PassTS        int64  `json:"passTs,omitempty"`

	// Email + EmailMeta — TempMail reuse (RegMode=TempMail).
	// Email = địa chỉ mail tạm đã add vào FB account khi reg.
	// EmailMeta = JSON-encoded credentials provider-specific để verify Restore
	// và đọc OTP từ inbox đã có sẵn (skip CreateEmail + skip AddEmail step).
	// Empty cho mode Phone/Mail (giả) → verify dùng flow CreateEmail mới.
	Email     string `json:"email,omitempty"`
	EmailMeta string `json:"emailMeta,omitempty"`

	// VerifyPlatform — platform verify ĐÃ round-robin gán per-account ở precompute.
	// Nếu set → scheduler dùng giá trị này (KHÔNG re-rotate GetVerifyPlatform) để
	// UA (precompute) khớp đúng version verify. Empty → fallback GetVerifyPlatform.
	VerifyPlatform string `json:"verifyPlatform,omitempty"`
}

// RunConfig cấu hình chạy
type RunConfig struct {
	MaxThreads int `json:"maxThreads"`
	DelayMs    int `json:"delayMs"` // Nghỉ giữa các request (ms) — từ settings
	// DelayAfterResultMs — giữ status cuối của account trên UI N ms trước khi
	// fetch account mới, để user đọc được kết quả (live/die + email...)
	// Thường 3000-5000ms. 0 = không delay (bị ghi đè ngay — khó đọc).
	DelayAfterResultMs int                    `json:"delayAfterResultMs"`
	VerifyConfig       *instagram.VerifyConfig `json:"verifyConfig"`
	// GetVerifyConfig nếu set → mỗi goroutine gọi hàm này để lấy config mới nhất (realtime)
	// Cho phép user thay đổi mail provider, v.v. trong lúc đang chạy
	GetVerifyConfig func() *instagram.VerifyConfig
	OutputPath      string `json:"outputPath"`
	// WorkerCtx context riêng cho HTTP requests trong worker
	// Nếu set → worker dùng WorkerCtx (không bị cancel khi Stop, chạy hết các bước)
	// Nếu nil → dùng ctx truyền vào RunVerify/RunOneAccount (cancel khi Stop)
	WorkerCtx context.Context
	// VerifyPlatform — platform API dùng cho verify (default: "web")
	VerifyPlatform string
	// GetVerifyPlatform — nếu set, worker gọi để lấy platform mới nhất (realtime).
	// Cho phép user thay đổi API VERIFY giữa batch.
	GetVerifyPlatform func() string
	// AcquireProxy trả về (proxyStr, releaseFunc) cho mỗi goroutine.
	// workerID: chỉ số slot worker (0..maxThreads-1) — dùng để pin sticky
	// proxy per slot (KeepIPSuccess). release(success) để sticky manager biết
	// có nên giữ proxy cho account kế tiếp hay release về pool.
	AcquireProxy func(ctx context.Context, workerID int) (string, func(success bool))
	// RequireProxy — nếu true, goroutine PHẢI có proxy; trả về "" → abort account (không chạy bằng IP máy)
	RequireProxy bool
	// OnRawProxy callback ngay khi proxy được gán cho account — gửi raw proxy string
	// để hiển thị cột PROXY trên UI (trước khi CheckIP resolve xong)
	OnRawProxy func(accountID int, proxyStr string)
	// OnProxy callback sau khi CheckIP resolve xong — gửi actual IP để hiển thị cột IP CHẠY
	OnProxy func(accountID int, proxyStr string)
	// OnEmailCreated callback ngay khi temp/rent mail được tạo — gửi email string để hiển thị
	// realtime trong cột EMAIL/PHONE trên UI (trước khi verify done).
	OnEmailCreated func(accountID int, email string)
	// OnAccountDone callback ngay khi 1 account hoàn thành — realtime, trước khi RunVerify return.
	// verifyPlatform: platform API thực tế đã dùng cho account này (hỗ trợ multi-version stats).
	OnAccountDone func(accountID int, uid string, status string, message string, email string, userAgent string, twoFA string, token string, cookie string, verifyPlatform string)
	// IsRetry — nếu true, kết quả Unknown sẽ được lưu thành Die thay vì Unknown.
	// Dùng cho luồng retry từ Unknown.txt để không tạo vòng lặp vô tận.
	IsRetry bool
	// AddMailRetry — số lần retry thêm khi add mail thất bại (ngoài 2 lần mặc định).
	// Mỗi retry outer loop gọi lại GetVerifyConfig() → có thể đổi mail provider mid-run.
	// 0 = dùng mặc định 2 outer attempts.
	AddMailRetry int
	// RetryUnknownNow — sau khi pass đầu xong, tự verify lại tất cả acc Unknown.
	// Chỉ chạy 1 pass thêm (không recursion vô tận). Acc nào vẫn Unknown sau pass 2
	// được giữ nguyên Unknown. Bật từ UI: checkbox "Verify lại Unknown ngay".
	RetryUnknownNow bool
	// GetUseOriginalUA — trả true nếu dùng UA gốc cố định (bỏ qua sinh UA mới theo proxy country).
	// Dùng func để user đổi mid-run có hiệu lực ngay không cần restart.
	GetUseOriginalUA func() bool
	// GetBuildUA — trả true nếu build UA động từ Config/DeviceInfo/ + RandomFbVersion().
	// Chỉ áp dụng cho platform generic (Android/S23) — platform versioned (S55x/S56x)
	// vẫn dùng PlatformVerifyUA để đảm bảo FBAV đúng phiên bản API.
	GetBuildUA func() bool
	// GetAddVirtualSpec — trả true nếu prepend Dalvik prefix vào UA khi BuildUA=true.
	GetAddVirtualSpec func() bool
	// GetVerifyUAForPlatform — trả UA GỐC cho platform cụ thể (per-account, sau round-robin).
	// FIX multi-version + UA Gốc: trước đây UA precompute theo bản FOCUS (ApiVerifyPlatform)
	// nhưng platform verify lại round-robin → mismatch. Callback này recompute UA theo
	// đúng platform mỗi account verify. nil → giữ UA precomputed (behavior cũ).
	GetVerifyUAForPlatform func(platform, country string) string
}

// AccountResult kết quả 1 account
type AccountResult struct {
	AccountID      int    `json:"accountId"`
	UID            string `json:"uid"`
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	Status         string `json:"status"` // Live/Die/Unknown
	Email          string `json:"email"`
	UserAgent      string `json:"userAgent"`
	TwoFA          string `json:"twoFA"`
	Token          string `json:"token"`          // EAA access token — populated sau Android login
	Cookie         string `json:"cookie"`         // cookie MỚI sau login verify (c_user/xs/fr/datr) — để UI cập nhật
	VerifyPlatform string `json:"verifyPlatform"` // platform API thực tế đã dùng cho account này
}

// StatusCallback callback cập nhật trạng thái realtime
type StatusCallback func(accountID int, uid string, message string)

// RunVerify chạy verify cho danh sách accounts song song theo kiểu FIFO worker pool.
// Mapping từ WeBM Run() FIFO Scheduler lines 1121-1400.
//
// ctx: context điều khiển vòng lặp dispatch — cancel (Stop) sẽ dừng việc đẩy accounts
// mới vào pool, nhưng KHÔNG abort worker đang chạy dở (worker dùng WorkerCtx hoặc
// ctx riêng để chạy đến hết các bước HTTP hiện tại).
//
// accounts: danh sách account cần verify, được xử lý theo thứ tự FIFO; kết quả trả
// về cùng index với input.
//
// config: cấu hình runtime — MaxThreads giới hạn số goroutine đồng thời (1–500),
// DelayMs nghỉ giữa các lần dispatch, GetVerifyConfig cho phép worker lấy config mới
// nhất (mail provider, v.v.) ngay trước khi bắt đầu — không cần restart để có hiệu lực.
//
// onStatus: callback gọi mỗi khi 1 account phát sinh message trạng thái mới (realtime),
// dùng để cập nhật UI mà không cần chờ toàn batch hoàn thành.
//
// Worker pool: semaphore channel giới hạn goroutine song song; dispatch block cho đến
// khi có slot trống (FIFO). Mỗi worker gọi runOneAccount độc lập — các account không
// phụ thuộc nhau. Sau khi ctx cancel, RunVerify chờ tối đa 10 giây để các worker đang
// chạy kịp hoàn thành trước khi return.
func RunVerify(ctx context.Context, accounts []AccountInput, config RunConfig, onStatus StatusCallback) []AccountResult {
	maxThreads := config.MaxThreads
	if maxThreads < 1 {
		maxThreads = 1
	}
	if maxThreads > 500 {
		maxThreads = 500
	}

	dateFolder := config.OutputPath

	workerCtx := ctx
	if config.WorkerCtx != nil {
		workerCtx = config.WorkerCtx
	}

	results := make([]AccountResult, len(accounts))
	var mu sync.Mutex

	delayBetween := time.Duration(config.DelayMs) * time.Millisecond

	// Work queue — tất cả accounts đẩy vào channel, N workers tự lấy
	type workItem struct {
		idx     int
		account AccountInput
	}
	workCh := make(chan workItem, len(accounts))
	for i, acc := range accounts {
		workCh <- workItem{idx: i, account: acc}
	}
	close(workCh)

	// Start N workers đồng thời — mỗi worker tự lấy account từ queue.
	// workerID = chỉ số slot ổn định (0..maxThreads-1) — dùng bởi sticky proxy
	// (KeepIPSuccess) để pin proxy theo slot, không theo goroutine ID.
	var wg sync.WaitGroup
	for w := 0; w < maxThreads; w++ {
		workerID := w
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					slog.Error("verify worker panic recovered", "worker", workerID, "panic", r)
				}
			}()
			first := true
			for work := range workCh {
				if ctx.Err() != nil {
					mu.Lock()
					results[work.idx] = AccountResult{
						AccountID: work.account.ID,
						UID:       work.account.UID,
						Message:   "Đã dừng",
					}
					mu.Unlock()
					continue
				}
				// Delay hiển thị kết quả trước khi fetch account mới
				// (cho user đọc status/email của acc vừa xong trên UI).
				if !first && config.DelayAfterResultMs > 0 {
					select {
					case <-ctx.Done():
					case <-time.After(time.Duration(config.DelayAfterResultMs) * time.Millisecond):
					}
				}
				if !first && delayBetween > 0 {
					select {
					case <-ctx.Done():
					case <-time.After(delayBetween):
					}
				}
				// Notify chuyển sang acc kế tiếp (acc đầu tiên thì bỏ qua).
				if !first && onStatus != nil {
					onStatus(work.account.ID, work.account.UID,
						fmt.Sprintf("▶️ Khởi chạy luồng verify tiếp theo (acc #%d)...", work.idx+1))
				}
				first = false

				result := runOneAccount(workerCtx, work.account, config, dateFolder, workerID, onStatus)

				mu.Lock()
				results[work.idx] = result
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// RetryUnknownNow — pass 2 cho acc Unknown nếu user bật checkbox.
	// CHỈ chạy 1 pass thêm (tránh vòng lặp vô tận). Acc nào vẫn Unknown sau pass 2
	// được giữ nguyên. Re-queue chỉ những acc kết quả status="" / "unknown" / "error"
	// (Live/Die là kết quả terminal — không retry).
	if config.RetryUnknownNow && !config.IsRetry && ctx.Err() == nil {
		var retryItems []workItem
		for i := range results {
			st := strings.ToLower(strings.TrimSpace(results[i].Status))
			if st == "" || st == "unknown" || st == "error" {
				if i < len(accounts) && accounts[i].UID != "" {
					retryItems = append(retryItems, workItem{idx: i, account: accounts[i]})
				}
			}
		}
		if len(retryItems) > 0 {
			if onStatus != nil {
				onStatus(0, "", fmt.Sprintf("🔁 [RetryUnknown] Pass 2 — verify lại %d account Unknown/Error với proxy MỚI...", len(retryItems)))
			}
			retryCh := make(chan workItem, len(retryItems))
			for _, it := range retryItems {
				retryCh <- it
			}
			close(retryCh)

			// Đánh dấu IsRetry để runOneAccount KHÔNG retry vô hạn (giữ Unknown=Unknown).
			retryCfg := config
			retryCfg.IsRetry = true
			retryCfg.RetryUnknownNow = false // tránh recursion ở pass 3

			// Offset workerID = workerID + maxThreads để sticky-proxy manager dùng SLOT MỚI
			// → force acquire proxy fresh từ pool (không reuse proxy đã fail ở pass 1).
			// Pass 1 đã `release(false)` cho mọi acc non-Live → proxy đã quay về pool,
			// nhưng nếu pool nhỏ vẫn có thể bắt trúng. Offset slot = double-insurance.
			workerIDOffset := maxThreads
			var rwg sync.WaitGroup
			for w := 0; w < maxThreads; w++ {
				retryWorkerID := w + workerIDOffset
				rwg.Add(1)
				go func() {
					defer rwg.Done()
					defer func() {
						if r := recover(); r != nil {
							slog.Error("retry-unknown worker panic", "worker", retryWorkerID, "panic", r)
						}
					}()
					for work := range retryCh {
						if ctx.Err() != nil {
							return
						}
						result := runOneAccount(workerCtx, work.account, retryCfg, dateFolder, retryWorkerID, onStatus)
						mu.Lock()
						results[work.idx] = result
						mu.Unlock()
					}
				}()
			}
			rwg.Wait()
			if onStatus != nil {
				onStatus(0, "", fmt.Sprintf("🔁 [RetryUnknown] Pass 2 xong (%d account)", len(retryItems)))
			}
		}
	}

	return results
}

// isOTPError báo cáo liệu msg có phải lỗi không nhận được OTP hay không.
//
// msg: chuỗi message từ bước poll email trong verify.go / verifybase/run.go.
// Match cả tiếng Việt ("không nhận được otp") lẫn tiếng Anh ("otp timeout",
// "no otp code received") vì verifybase/run.go return message tiếng Anh:
//   - "OTP timeout: no OTP code received after N attempts"
//   - "OTP timeout after resend: ..."
//
// Lỗi OTP KHÔNG được retry inner loop vì: email OTP đã bị gửi và hết hạn phía Facebook,
// retry login sẽ kích hoạt một mã OTP mới nhưng vẫn cần chờ email về — nếu hộp thư
// chậm/không hoạt động thì retry liên tiếp chỉ tạo thêm OTP rác và tốn thời gian.
// Thay vào đó outer loop sẽ thử lại với email mới (AddMailRetry).
func isOTPError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "không nhận được otp") ||
		strings.Contains(lower, "otp timeout") ||
		strings.Contains(lower, "no otp code received")
}

// isCookieDead báo cáo liệu msg có phải lỗi do cookie không còn hợp lệ hay không.
//
// msg: chuỗi message trả về từ bước LoginWithCookieMobile trong facebook package.
// Các string pattern được match là message cụ thể do instagram.LoginWithCookieMobile
// emit khi:
//   - "cookie không hợp lệ": Facebook trả response báo cookie bị từ chối/hết hạn.
//   - "đăng nhập bằng cookie thất bại": bước submit form login không thành công.
//   - "không có cookie": AccountInput.Cookie rỗng, không có gì để gửi.
//   - "cookie không thể đăng nhập": cookie parse được nhưng không qua được checkpoint.
//
// Cookie chết KHÔNG được retry (kể cả outer loop) vì: gửi lại cùng cookie đã chết
// sẽ cho cùng kết quả — cần cập nhật cookie mới từ bên ngoài rồi mới chạy lại.
func isCookieDead(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "cookie không hợp lệ") ||
		strings.Contains(lower, "đăng nhập bằng cookie thất bại") ||
		strings.Contains(lower, "không có cookie") ||
		strings.Contains(lower, "cookie không thể đăng nhập")
}

// isBloksCheckpoint trả true khi Facebook chặn bước verify bằng Bloks UI (checkpoint,
// xác minh danh tính, confirm thiết bị...). Không thể auto-resolve → Unknown.
func isBloksCheckpoint(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "fb_bloks_action") ||
		strings.Contains(lower, "bloks_bundle_action") ||
		strings.Contains(lower, "bloks_payload")
}

// isTokenDead trả true khi token của account đã bị FB invalidate hoàn toàn.
// HTTP 401 + "malformed access token" = token chết vĩnh viễn → Die, không retry.
func isTokenDead(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "malformed access token") ||
		(strings.Contains(lower, "http 401") && strings.Contains(lower, "access token"))
}

// isNetworkError báo cáo liệu msg có phải lỗi mạng/proxy không.
//
// Khi login gặp lỗi này, retry với cùng proxy sẽ cho cùng kết quả.
// Cần goto done ngay để tránh lãng phí maxAttempts × timeout.
func isNetworkError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "context deadline exceeded") ||
		strings.Contains(lower, "client.timeout") ||
		strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "connection reset by peer") ||
		strings.Contains(lower, "forcibly closed by the remote host") ||
		strings.Contains(lower, "wsarecv") ||
		strings.Contains(lower, "wsasend") ||
		strings.Contains(lower, "eof") ||
		strings.Contains(lower, "dial tcp")
}

// runOneAccount chạy đầy đủ flow verify cho một account, có cơ chế retry hai tầng.
// Mapping từ WeBM ExcuteOneThread() + ExecuteAction().
//
// ctx: context điều khiển HTTP requests bên trong — thường là WorkerCtx (không bị
// cancel khi Stop) để worker chạy hết các bước hiện tại trước khi dừng. Nếu cần
// dừng ngay (emergency), hủy context này.
//
// acc: dữ liệu đầu vào của một account (cookie, token, proxy, v.v.). Hàm tạo
// instagram.Session từ acc và có thể mutate các trường token trong quá trình login.
//
// config: cấu hình runtime; quan trọng nhất là GetVerifyConfig — nếu khác nil, hàm
// gọi nó ngay khi bắt đầu và mỗi outer retry để lấy verify.Config mới nhất (realtime
// config update: user có thể đổi mail provider trong lúc batch đang chạy).
//
// dateFolder: đường dẫn thư mục output của batch hiện tại (dạng "VerifyAccount20060102_150405").
// Kết quả Unknown sẽ được append vào dateFolder/Unknown.txt; Live/Die được lưu bên
// trong verify.VerifyAccount.
//
// onStatus: callback emit message trạng thái realtime cho từng bước (login, OTP, v.v.).
//
// Cơ chế retry hai tầng:
//
//   - Inner loop (maxAttempts = 3): retry toàn bộ login + verify khi gặp lỗi mạng
//     tạm thời. Mỗi lần retry reset fb_dtsg/jazoest/lsd rồi đăng nhập lại từ đầu.
//     Dừng sớm nếu: cookie chết (isCookieDead), context bị hủy, hoặc đã có
//     kết quả Live/Die rõ ràng.
//
//   - Outer loop (maxOuterAttempts = 2): chạy lại toàn bộ flow khi inner loop xong
//     mà kết quả vẫn Unknown (thường do OTP không về). Mỗi outer retry gọi lại
//     GetVerifyConfig để có thể dùng email provider khác.
func runOneAccount(ctx context.Context, acc AccountInput, config RunConfig, dateFolder string, workerID int, onStatus StatusCallback) AccountResult {
	// Nếu có GetVerifyConfig → lấy config mới nhất ngay khi goroutine bắt đầu (realtime)
	if config.GetVerifyConfig != nil {
		config.VerifyConfig = config.GetVerifyConfig()
	}
	// Inject OnEmailCreated — bridge runner-level callback (biết accountID) → verify-level
	// (không biết accountID). Closure capture acc.ID để emit đúng per-worker.
	if config.VerifyConfig != nil && config.OnEmailCreated != nil {
		accID := acc.ID
		onEmail := config.OnEmailCreated
		config.VerifyConfig.OnEmailCreated = func(email string) { onEmail(accID, email) }
	}
	// Diagnostic: emit MailProvider ngay đầu account — user thấy ngay provider
	// nào đang áp dụng thay vì phải chờ đến step "Adding email".
	if config.VerifyConfig != nil && onStatus != nil {
		onStatus(acc.ID, acc.UID, fmt.Sprintf("[Config] MailProvider=%s (reload mỗi attempt)", config.VerifyConfig.MailProvider))
	}

	// maxAttempts: số lần chạy login+verify (1 lần đầu + 1 retry = tổng 2)
	const maxAttempts = 2
	// maxOuterAttempts: 2 mặc định; nếu user nhập AddMailRetry thì dùng đúng số lượt đó.
	// Mỗi outer attempt gọi lại GetVerifyConfig() → có thể đổi mail provider/email.
	maxOuterAttempts := config.AddMailRetry
	if maxOuterAttempts < 2 {
		maxOuterAttempts = 2
	}

	result := AccountResult{
		AccountID: acc.ID,
		UID:       acc.UID,
	}

	notify := func(msg string) {
		if onStatus != nil {
			onStatus(acc.ID, acc.UID, msg)
		}
	}

	// Lấy proxy từ manager — release(success) để sticky manager biết giữ hay thả.
	// defer đọc result.Status cuối cùng để quyết định success (Live = giữ IP cho account kế).
	proxyStr := acc.Proxy // ưu tiên proxy riêng của account nếu có
	if proxyStr == "" && config.AcquireProxy != nil {
		var releaseProxy func(bool)
		proxyStr, releaseProxy = config.AcquireProxy(ctx, workerID)
		defer func() {
			releaseProxy(result.Status == "Live")
		}()
	}
	// RequireProxy: nếu proxy bắt buộc nhưng không có → abort, không dùng IP máy
	if proxyStr == "" && config.RequireProxy {
		notify("Không có proxy — bỏ qua account (proxy bắt buộc)")
		result.Status = "error"
		result.Message = "Không có proxy"
		return result
	}
	// Gửi raw proxy ngay lập tức → cập nhật cột PROXY trên UI
	if config.OnRawProxy != nil && proxyStr != "" {
		config.OnRawProxy(acc.ID, proxyStr)
	}
	// Check IP chạy background — không block login
	// CheckIP chỉ dùng để hiển thị cột IP CHẠY trên UI, không ảnh hưởng logic verify
	if config.OnProxy != nil {
		onProxy := config.OnProxy
		go func() {
			if actualIP, err := proxy.CheckIP(ctx, proxyStr, proxy.APICheckIpAuto); err == nil {
				onProxy(acc.ID, actualIP)
			}
			// Nếu CheckIP fail → KHÔNG emit gì để cột IP CHẠY trống (chứ không show raw proxy).
			// User nhìn cột HOẠT ĐỘNG để biết trạng thái thật của luồng.
		}()
	}

	// Tạo Facebook session từ account data (pointer — login sẽ mutate tokens)
	session := &instagram.Session{
		UID:                   acc.UID,
		FullName:              acc.FullName,
		Phone:                 acc.Phone,
		Cookie:                acc.Cookie,
		Token:                 acc.Token,
		UserAgent:             acc.UserAgent,
		Proxy:                 proxyStr,
		Password:              acc.Password,
		InputAccount:          acc.InputAccount,
		DeviceID:              acc.DeviceID,
		FamilyDeviceID:        acc.FamilyDeviceID,
		Srnonce:               acc.Srnonce,
		SessionlessCryptedUID: acc.SessionlessCryptedUID,
		// TempMail reuse: forward email + creds từ reg để verify steps detect
		// và skip CreateEmail/AddEmail (xem internal/facebook/verify/*/steps.go).
		Email:     acc.Email,
		EmailMeta: acc.EmailMeta,
		// iOS Messenger add-mail: cần AAC context + flow IDs + password từ create.
		AACJid:        acc.AACJid,
		AACcs:         acc.AACcs,
		AACts:         acc.AACts,
		RegFlowID:     acc.RegFlowID,
		HeadersFlowID: acc.HeadersFlowID,
		PassRaw:       acc.PassRaw,
		PassTS:        acc.PassTS,
		Username:      acc.Username,
	}

	// Country-aware UA cho S55x: extract country từ proxy suffix "/<cc>"
	// rồi gọi factory để sinh UA với carrier khớp country (vd US → T-Mobile/AT&T,
	// VN → Viettel/Vinaphone). Chỉ override khi:
	//   - platform là Android-versioned (S555/S556/S557/S558)
	//   - lấy được country từ proxy (suffix dạng /vn, /us...)
	//   - UseOriginalUA=false (nếu true: device/model cố định, FBCR đã set từ trước)
	useOrigUA := config.GetUseOriginalUA != nil && config.GetUseOriginalUA()
	useBuildUA := config.GetBuildUA != nil && config.GetBuildUA()
	addVirtualSpec := config.GetAddVirtualSpec != nil && config.GetAddVirtualSpec()

	// Resolve verify platform MỘT LẦN cho cả account này — ổn định suốt account
	// (tránh lệch giữa quyết định login-skip và lần verify thật). Hỗ trợ multi-version:
	// GetVerifyPlatform phía App round-robin chia version theo từng account.
	verifyPlatform := config.VerifyPlatform
	if acc.VerifyPlatform != "" {
		// Platform đã round-robin gán per-account ở precompute (multi-version) → dùng đúng
		// version này (KHÔNG re-rotate GetVerifyPlatform) để UA hiển thị khớp version verify.
		verifyPlatform = acc.VerifyPlatform
	} else if config.GetVerifyPlatform != nil {
		if p := config.GetVerifyPlatform(); p != "" {
			verifyPlatform = p
		}
	}
	if verifyPlatform == "" {
		verifyPlatform = instagram.PlatformWeb
	}
	result.VerifyPlatform = verifyPlatform

	if !useOrigUA {
		// Ưu tiên: platform có UA factory riêng (Android versioned + iOS) → dùng đúng UA version.
		// Fallback: BuildUA động nếu platform không có factory (generic Android/S23/Web/...).
		uaSet := false
		if country := extractCountryFromProxy(proxyStr); country != "" {
			if ua := instagram.PlatformVerifyUA(verifyPlatform, country); ua != "" {
				session.UserAgent = ua
				uaSet = true
			}
		}
		if !uaSet && useBuildUA {
			country := extractCountryFromProxy(proxyStr)
			dev := fakeinfo.RandomDeviceProfile()
			locale := fakeinfo.LocaleFromCountry(country)
			sim := fakeinfo.RandomSimProfile(country)
			carrier := sim.OperatorName
			if carrier == "" {
				carrier = fakeinfo.RandomCarrier()
			}
			fbVer, fbBuild := fakeinfo.RandomFbVersionVer()
			if ua := fakeinfo.BuildAndroidUAWithOpts(dev, locale, carrier, fbVer, fbBuild, addVirtualSpec, false); ua != "" {
				session.UserAgent = ua
			}
		}
	} else if config.GetVerifyUAForPlatform != nil {
		// UA GỐC BẬT + multi-version: recompute UA gốc theo ĐÚNG platform của account này
		// (đã round-robin), thay vì giữ UA precomputed của bản FOCUS. FIX mismatch UA↔version.
		if ua := config.GetVerifyUAForPlatform(verifyPlatform, extractCountryFromProxy(proxyStr)); ua != "" {
			session.UserAgent = ua
		}
	}

	// === OUTER RETRY LOOP: chạy lại toàn bộ flow khi kết quả unknown ===
	for outer := 0; outer < maxOuterAttempts; outer++ {
		if outer > 0 {
			if ctx.Err() != nil {
				result.Message = "Đã dừng"
				goto done
			}
			notify(fmt.Sprintf("[Retry lần %d/%d] Thử lại với email mới...", outer, maxOuterAttempts-1))
			// Reset lại config để lấy email provider mới nhất
			if config.GetVerifyConfig != nil {
				config.VerifyConfig = config.GetVerifyConfig()
				// Re-inject OnEmailCreated (GetVerifyConfig trả pointer fresh)
				if config.VerifyConfig != nil && config.OnEmailCreated != nil {
					accID := acc.ID
					onEmail := config.OnEmailCreated
					config.VerifyConfig.OnEmailCreated = func(email string) { onEmail(accID, email) }
				}
			}
			// Reset session tokens
			session.FbDtsg = ""
			session.Jazoest = ""
			session.Lsd = ""
		}

		// Reset result cho mỗi outer attempt
		result.Status = ""
		result.Message = ""
		result.Email = ""

		for attempt := 1; attempt <= maxAttempts; attempt++ {
			if ctx.Err() != nil {
				result.Message = "Đã dừng"
				goto done
			}

			if attempt > 1 {
				notify(fmt.Sprintf("[Retry %d/%d] Đăng nhập lại và thử từ đầu...", attempt, maxAttempts))
				select {
				case <-ctx.Done():
					result.Message = "Đã dừng"
					goto done
				case <-time.After(2 * time.Second):
				}
				session.FbDtsg = ""
				session.Jazoest = ""
				session.Lsd = ""
			}

			// verifyPlatform đã resolve 1 lần ở đầu account (ổn định suốt account).

			// S23/Android/S55x/S557/S558/S559/S559V2/S560/S399 verify dùng access token trực tiếp → skip cookie login
			// WebAndroid verify dùng cookie trực tiếp → skip cookie login
			// Web verify cần fb_dtsg/jazoest → phải login trước
			switch verifyPlatform {
			case instagram.PlatformS22, instagram.PlatformS23, instagram.PlatformS24, instagram.PlatformS25, instagram.PlatformS26,
				instagram.PlatformAndroid,
				instagram.PlatformS415, instagram.PlatformS425, instagram.PlatformS435, instagram.PlatformS445,
				instagram.PlatformS416, instagram.PlatformS417, instagram.PlatformS418, instagram.PlatformS419, instagram.PlatformS420, instagram.PlatformS421,
				instagram.PlatformS422, instagram.PlatformS423, instagram.PlatformS424, instagram.PlatformS426, instagram.PlatformS427, instagram.PlatformS428,
				instagram.PlatformS429, instagram.PlatformS430, instagram.PlatformS431, instagram.PlatformS432, instagram.PlatformS433, instagram.PlatformS434,
				instagram.PlatformS436, instagram.PlatformS437, instagram.PlatformS438, instagram.PlatformS439, instagram.PlatformS440, instagram.PlatformS441,
				instagram.PlatformS442, instagram.PlatformS443, instagram.PlatformS444,
				instagram.PlatformS446, instagram.PlatformS447, instagram.PlatformS448, instagram.PlatformS449, instagram.PlatformS450, instagram.PlatformS451,
				instagram.PlatformS452, instagram.PlatformS453, instagram.PlatformS454, instagram.PlatformS455,
				instagram.PlatformS456,
				instagram.PlatformS457,
				instagram.PlatformS458,
				instagram.PlatformS459,
				instagram.PlatformS460,
				instagram.PlatformS461,
				instagram.PlatformS462,
				instagram.PlatformS463,
				instagram.PlatformS464,
				instagram.PlatformS465,
				instagram.PlatformS466, instagram.PlatformS467, instagram.PlatformS468, instagram.PlatformS469, instagram.PlatformS470,
				instagram.PlatformS471, instagram.PlatformS472, instagram.PlatformS473, instagram.PlatformS474, instagram.PlatformS475, instagram.PlatformS476, instagram.PlatformS477, instagram.PlatformS478, instagram.PlatformS479,
				instagram.PlatformS480, instagram.PlatformS481, instagram.PlatformS482, instagram.PlatformS483, instagram.PlatformS484,
				instagram.PlatformS485, instagram.PlatformS486, instagram.PlatformS487, instagram.PlatformS488,
				instagram.PlatformS489, instagram.PlatformS490, instagram.PlatformS491, instagram.PlatformS492, instagram.PlatformS493,
				instagram.PlatformS494, instagram.PlatformS496, instagram.PlatformS497, instagram.PlatformS498, instagram.PlatformS499,
				instagram.PlatformS495,
				instagram.PlatformS555, instagram.PlatformS556,
				instagram.PlatformS555V2,
				instagram.PlatformS557, instagram.PlatformS558,
				instagram.PlatformS558V2,
				instagram.PlatformS556V2, instagram.PlatformS557V2,
				instagram.PlatformS553V2, instagram.PlatformS554V2,
				instagram.PlatformS551V2, instagram.PlatformS552V2,
				instagram.PlatformS550V2,
				instagram.PlatformS559, instagram.PlatformS559V2, instagram.PlatformS560, instagram.PlatformS560V2, instagram.PlatformS560V3,
				instagram.PlatformS561, instagram.PlatformS561V2, instagram.PlatformS561V3, instagram.PlatformS561V99, instagram.PlatformS561V4S21, instagram.PlatformS561V4S23,
				instagram.PlatformS562, instagram.PlatformS562V3, instagram.PlatformS562V4S21, instagram.PlatformS562V4S23, instagram.PlatformS563, instagram.PlatformS563V2, instagram.PlatformS563S21, instagram.PlatformS563V3S21, instagram.PlatformS563V4S21, instagram.PlatformS563V4S23, instagram.PlatformS563V5S21, instagram.PlatformS563V5S23, instagram.PlatformS563V6S21, instagram.PlatformS563V6S23, instagram.PlatformS564V1S21, instagram.PlatformS564V1S23, instagram.PlatformS564V2S21, instagram.PlatformS564V2S23, instagram.PlatformS564V3S21, instagram.PlatformS564V3S23, instagram.PlatformS565S21, instagram.PlatformS565S23, instagram.PlatformS565V2S21, instagram.PlatformS565V2S23, instagram.PlatformAppMessV3,
				instagram.PlatformAppMessV3_535, instagram.PlatformAppMessV3_545, instagram.PlatformAppMessV3_555, instagram.PlatformAppMessV3_563, instagram.PlatformAppMessV3_564, instagram.PlatformAppMessV3_565, instagram.PlatformAppMessV3_525, instagram.PlatformAppMessV3_515, instagram.PlatformAppMessV3_505, instagram.PlatformAppMessV3_490,
				instagram.PlatformS399,
				instagram.PlatformS273:
				// Token thiếu nhưng có UID + password → login REST /auth/login (port S399) để lấy EAA.
				// (Một số registerer như WebAndroid/iOSHttp/Web không trả token sau reg.)
				// PORT S399: REST classic API stable, KHÔNG dùng Bloks/GraphQL schema rotation.
				// UA: pass "" → auto FB4A UA (KHÔNG dùng session.UserAgent vì có thể Chrome WebAndroid).
				// Token iOS (EAAAAAY) KHÔNG hợp lệ cho Android verify (endpoint b-graph) → bỏ để login
				// lấy EAAAAU đúng loại. Đối xứng với iOS verify (chỉ nhận EAAAAAY, gặp EAAAAU thì login iOS).
				if strings.HasPrefix(session.Token, "EAAAAAY") {
					notify("Token iOS (EAAAAAY) không dùng cho Android verify → login lấy EAAAAU đúng loại...")
					session.Token = ""
				}
				needFetch := !isValidAndroidToken(session.Token) && session.UID != "" && session.Password != ""
				if needFetch {
					notify(fmt.Sprintf("[Login][Android] Token chưa có/không hợp lệ (platform=%s) → bỏ qua (REST login đã bị gỡ)", verifyPlatform))
				} else if result.Token == "" && isValidAndroidToken(session.Token) {
					// Token đã hợp lệ — không fetch mới, nhưng phải propagate
					// để OnAccountDone / file output / UI nhận được token.
					result.Token = session.Token
				}
				if session.Cookie != "" {
					result.Cookie = session.Cookie
				}
				if isValidAndroidToken(session.Token) {
					notify(fmt.Sprintf("Skip login — dùng access token trực tiếp (platform=%s)", verifyPlatform))
				} else {
					notify(fmt.Sprintf("Không có access token cho platform=%s — bỏ qua account", verifyPlatform))
					result.Message = "No valid access token for Android-family verify"
					result.Status = "unknown"
					goto done
				}
			case instagram.PlatformIOSMess:
				// Messenger Lite iOS verify: dùng APP token (SkipUserTokenCheck trong spec),
				// KHÔNG cần user token / pre-login. Delegate thẳng xuống verify (add-mail+confirm).
				notify("[iOS Mess] Verify add-mail + confirm (app token) — delegate to verify")
			case instagram.PlatformIOS562, instagram.PlatformIOS563, instagram.PlatformIOS555, instagram.PlatformIOS550, instagram.PlatformIOS540, instagram.PlatformIOS530, instagram.PlatformIOS520, instagram.PlatformIOS564,
				instagram.PlatformIOS510, instagram.PlatformIOS500, instagram.PlatformIOS490, instagram.PlatformIOS480, instagram.PlatformIOS470, instagram.PlatformIOS460, instagram.PlatformIOS450,
				instagram.PlatformIOS440, instagram.PlatformIOS430, instagram.PlatformIOS420, instagram.PlatformIOS560,
				instagram.PlatformIOS421, instagram.PlatformIOS422, instagram.PlatformIOS423, instagram.PlatformIOS424, instagram.PlatformIOS425, instagram.PlatformIOS426, instagram.PlatformIOS427, instagram.PlatformIOS428, instagram.PlatformIOS429, instagram.PlatformIOS431, instagram.PlatformIOS432, instagram.PlatformIOS433, instagram.PlatformIOS434, instagram.PlatformIOS435, instagram.PlatformIOS436, instagram.PlatformIOS437, instagram.PlatformIOS438, instagram.PlatformIOS439, instagram.PlatformIOS441, instagram.PlatformIOS442, instagram.PlatformIOS443, instagram.PlatformIOS444, instagram.PlatformIOS445, instagram.PlatformIOS446, instagram.PlatformIOS447, instagram.PlatformIOS448, instagram.PlatformIOS449, instagram.PlatformIOS451, instagram.PlatformIOS452, instagram.PlatformIOS453, instagram.PlatformIOS454, instagram.PlatformIOS455, instagram.PlatformIOS456, instagram.PlatformIOS457, instagram.PlatformIOS458, instagram.PlatformIOS459, instagram.PlatformIOS461, instagram.PlatformIOS462, instagram.PlatformIOS463, instagram.PlatformIOS464, instagram.PlatformIOS465, instagram.PlatformIOS466, instagram.PlatformIOS467, instagram.PlatformIOS468, instagram.PlatformIOS469, instagram.PlatformIOS471, instagram.PlatformIOS472, instagram.PlatformIOS473, instagram.PlatformIOS474, instagram.PlatformIOS475, instagram.PlatformIOS476, instagram.PlatformIOS477, instagram.PlatformIOS478, instagram.PlatformIOS479, instagram.PlatformIOS481, instagram.PlatformIOS482, instagram.PlatformIOS483, instagram.PlatformIOS484, instagram.PlatformIOS485, instagram.PlatformIOS486, instagram.PlatformIOS487, instagram.PlatformIOS488, instagram.PlatformIOS489, instagram.PlatformIOS491, instagram.PlatformIOS492, instagram.PlatformIOS493, instagram.PlatformIOS494, instagram.PlatformIOS495, instagram.PlatformIOS496, instagram.PlatformIOS497, instagram.PlatformIOS498, instagram.PlatformIOS499, instagram.PlatformIOS501, instagram.PlatformIOS502, instagram.PlatformIOS503, instagram.PlatformIOS504, instagram.PlatformIOS505, instagram.PlatformIOS506, instagram.PlatformIOS507, instagram.PlatformIOS508, instagram.PlatformIOS509, instagram.PlatformIOS511, instagram.PlatformIOS512, instagram.PlatformIOS513, instagram.PlatformIOS514, instagram.PlatformIOS515, instagram.PlatformIOS516, instagram.PlatformIOS517, instagram.PlatformIOS518, instagram.PlatformIOS519, instagram.PlatformIOS521, instagram.PlatformIOS522, instagram.PlatformIOS523, instagram.PlatformIOS524, instagram.PlatformIOS525, instagram.PlatformIOS526, instagram.PlatformIOS527, instagram.PlatformIOS528, instagram.PlatformIOS529, instagram.PlatformIOS531, instagram.PlatformIOS532, instagram.PlatformIOS533, instagram.PlatformIOS534, instagram.PlatformIOS535, instagram.PlatformIOS536, instagram.PlatformIOS537, instagram.PlatformIOS538, instagram.PlatformIOS539, instagram.PlatformIOS541, instagram.PlatformIOS542, instagram.PlatformIOS543, instagram.PlatformIOS544, instagram.PlatformIOS545, instagram.PlatformIOS546, instagram.PlatformIOS547, instagram.PlatformIOS548, instagram.PlatformIOS549, instagram.PlatformIOS551, instagram.PlatformIOS552, instagram.PlatformIOS553, instagram.PlatformIOS554, instagram.PlatformIOS556, instagram.PlatformIOS557, instagram.PlatformIOS558, instagram.PlatformIOS559, instagram.PlatformIOS561:
				// iOS verify (FBIOS Bloks CAA): user token EAAAAAY là điều kiện bắt buộc.
				// Nếu account chưa có EAAAAAY (reg Android / nosess / chỉ EAAAAU / chỉ UID+pass)
				// → verifybase tự login iOS lấy EAAAAAY (xem spec.FetchToken trong ios562/steps.go).
				// Scheduler KHÔNG pre-login ở đây; chỉ delegate xuống verify.
				notify(fmt.Sprintf("[iOS] Verify FBIOS (%s) — token EAAAAAY bắt buộc (login trong verify nếu thiếu)", verifyPlatform))
			case instagram.PlatformWebAndroid:
				if session.Cookie != "" {
					notify(fmt.Sprintf("[WebAndroid] Skip login — dùng cookie trực tiếp (UID=%s)", session.UID))
				} else {
					notify("[WebAndroid] Không có cookie — bỏ qua account")
					result.Message = "No cookie for WebAndroid verify"
					result.Status = "unknown"
					goto done
				}
			case instagram.PlatformWeb:
				// Web MFB (api mfb): bắt buộc login cookie m.facebook.com để parse fb_dtsg/jazoest.
				// CHỈ PlatformWeb dùng cookie login — TẤT CẢ platform Android-family phải vào
				// case ở trên (fetch EAA token qua /auth/login), KHÔNG được fall xuống đây.
				notify("Đăng nhập bằng cookie...")
				loginResult, err := instagram.LoginWithCookieMobile(ctx, session)
				if err != nil || loginResult == nil || !loginResult.Success {
					msg := "Login thất bại"
					if err != nil {
						msg = fmt.Sprintf("Login lỗi: %v", err)
					} else if loginResult != nil {
						msg = loginResult.Message
					}
					notify(msg)
					result.Message = msg
					if isCookieDead(msg) {
						goto done
					}
					// Lỗi mạng/proxy: retry cùng proxy sẽ cho cùng kết quả → unknown để retry sau
					if isNetworkError(msg) {
						result.Status = "unknown"
						goto done
					}
					if attempt == maxAttempts {
						notify(fmt.Sprintf("[Retry] Login thất bại sau %d lần", maxAttempts))
					}
					continue
				}
				notify(fmt.Sprintf("Login OK — fb_dtsg=%s... jazoest=%s lsd=%s... datr=%s...",
					truncate(session.FbDtsg, 20), session.Jazoest, truncate(session.Lsd, 15), truncate(session.Datr, 10)))
			default:
				// SAFETY NET: platform chưa được đăng ký vào case nào ở trên.
				// Bug recurring: thêm platform Android mới mà QUÊN add vào Android-family list
				// → trước đây silent fall xuống cookie login → "đăng nhập bằng cookie" sai logic.
				// Giờ explicit error để dev nhận biết ngay thay vì verify bằng cookie nhầm.
				notify(fmt.Sprintf("[FATAL] Platform '%s' chưa được handle trong scheduler switch. "+
					"Nếu là Android-family → add vào list Android case (gần PlatformS399, PlatformS273). "+
					"Nếu là Web → add vào case PlatformWeb. Bỏ qua account.", verifyPlatform))
				result.Message = fmt.Sprintf("Platform '%s' chưa được scheduler hỗ trợ (bug: quên add vào switch case)", verifyPlatform)
				result.Status = "error"
				goto done
			}

			// verifyPlatform đã resolve 1 lần ở đầu account — không re-read ở đây nữa.
			// Reload config mỗi attempt — user đổi mail provider giữa chừng có
			// hiệu lực cho retry ngay (không đợi outer loop).
			if config.GetVerifyConfig != nil {
				config.VerifyConfig = config.GetVerifyConfig()
				// Re-inject OnEmailCreated vào config mới (GetVerifyConfig trả pointer
				// fresh → mất callback injected ở đầu runOneAccount).
				if config.VerifyConfig != nil && config.OnEmailCreated != nil {
					accID := acc.ID
					onEmail := config.OnEmailCreated
					config.VerifyConfig.OnEmailCreated = func(email string) { onEmail(accID, email) }
				}
			}
			// Sync UserApiLabel = platform THẬT (round-robin) để log hiển thị đúng version
			// account này đang chạy. Trước đây UserApiLabel = focus version (single) cố định
			// → mọi account log cùng 1 tên dù round-robin chia version khác nhau.
			// config.VerifyConfig là pointer fresh per-account (từ GetVerifyConfig) → an toàn mutate.
			if config.VerifyConfig != nil && verifyPlatform != "" {
				config.VerifyConfig.UserApiLabel = verifyPlatform
			}
			ver, _ := instagram.NewVerifier(verifyPlatform)
			verifyResult := ver.Verify(ctx, session, config.VerifyConfig, dateFolder, func(uid, msg string) {
				notify(msg)
			})

			result.Success = verifyResult.Success
			result.Message = verifyResult.Message
			result.Status = verifyResult.Status
			result.Email = verifyResult.Email
			result.UserAgent = verifyResult.UserAgent
			result.TwoFA = verifyResult.TwoFA
			// Propagate token thực tế đã dùng để verify (vd iOS login đổi session.Token
			// → EAAAAAY) để cột TOKEN hiển thị đúng loại, không giữ token reg cũ (EAAAAU).
			if session.Token != "" {
				result.Token = session.Token
			}
			// Back-fill cookie MỚI sau ver.Verify (iOS: FetchToken set session.Cookie từ login).
			if session.Cookie != "" {
				result.Cookie = session.Cookie
			}

			// Live/Die = kết quả rõ ràng → dừng hẳn, không retry
			if verifyResult.Status == "Live" || verifyResult.Status == "Die" {
				goto done
			}
			// Token chết vĩnh viễn → Die ngay, retry vô ích
			if isTokenDead(result.Message) {
				notify("Token hết hạn/malformed — Die")
				result.Status = "Die"
				goto done
			}
			// Facebook Bloks checkpoint → cần thao tác thủ công, không auto-resolve được
			if isBloksCheckpoint(result.Message) {
				notify("Facebook checkpoint (Bloks) — unknown để xử lý sau")
				result.Status = "unknown"
				goto done
			}
			// Lỗi mạng/proxy ở bước verify → retry cùng proxy sẽ cho kết quả như nhau.
			// Set Unknown (không phải Error) để Unknown.txt retry lại sau, không bỏ account.
			if isNetworkError(result.Message) {
				notify("Lỗi mạng/proxy — tạm unknown để retry sau")
				result.Status = "unknown"
				goto done
			}
			// OTP timeout → email đã hết hạn, retry cùng luồng vô ích
			// Thoát inner loop ngay, outer loop sẽ cấp email mới
			if isOTPError(result.Message) {
				break
			}
			if ctx.Err() != nil {
				result.Message = "Đã dừng"
				goto done
			}
			if attempt == maxAttempts {
				notify(fmt.Sprintf("[Retry] Verify thất bại sau %d lần: %s", maxAttempts, result.Message))
			}
		}

		// Inner loop xong mà vẫn không có kết quả Live/Die → outer loop sẽ retry
		if result.Status == "Live" || result.Status == "Die" {
			break // Có kết quả rõ ràng → thoát outer loop
		}
	}

	// Post-loop: nếu vẫn unknown sau tất cả outer attempts và UID đã biết →
	// check live nhanh qua Graph API picture endpoint để giảm thất thoát.
	// Nếu UID chết → ghi Die ngay (không ghi Unknown.txt vô ích).
	// Nếu UID sống → retry verify thêm 1 lần với email provider mới nhất.
	// Chỉ chạy khi loop thoát bình thường (không phải goto done từ early exit).
	if result.Status != "Live" && result.Status != "Die" && result.UID != "" {
		notify("[CheckUID] Verify unknown — kiểm tra UID còn sống không...")
		checkCtx, checkCancel := context.WithTimeout(ctx, 12*time.Second)
		var liveStatus string
		if session.Cookie != "" {
			liveStatus = igcore.CheckLiveByCookie(checkCtx, session.Cookie, session.UserAgent, session.Proxy)
		}
		checkCancel()
		switch liveStatus {
		case "die":
			notify("[CheckUID] UID đã chết → Die")
			result.Status = "Die"
		case "live":
			notify("[CheckUID] UID còn sống — đã hết số lượt retry, giữ Unknown")
			result.Status = "unknown"
		}
	}

done:
	// Save datr vào Config/Cookie/datr_pool.txt cho MỌI outcome có cookie chứa datr.
	// Match C# SaveDatrFromCookieIfNew gọi sau reg success, reg checkpoint,
	// verify die/unknown (FMain.cs L1504, L1529, L1565, L1782, L1793).
	// Chỉ ghi file, không add pool (datr từ verify có thể đã bị flag nếu die).
	if session.Cookie != "" {
		if datr := cookie.ExtractDatr(session.Cookie); datr != "" {
			go func(d string) {
				if err := cookie.AppendDatr("", d); err != nil {
					slog.Warn("save datr failed", "err", err, "datr", d)
				}
			}(datr)
		}
	}

	// File write đã được handle qua OnAccountDone → saveVerifyOutcome (app.go)
	// → ghi Die.txt / Unknown.txt / SuccessVerify*.txt qua Writer có UpsertUID dedupe.
	// Bỏ SaveAccountToFolder cũ (ghi 2 lần, tạo file trùng Die.txt + DieAfterVerify.txt).
	_ = dateFolder // giữ signature
	// Emit realtime ngay khi account xong — không chờ hết batch
	if config.OnAccountDone != nil {
		config.OnAccountDone(result.AccountID, result.UID, result.Status, result.Message, result.Email, result.UserAgent, result.TwoFA, result.Token, result.Cookie, result.VerifyPlatform)
	}
	return result
}

// RunOneAccount là exported wrapper của runOneAccount, dùng cho CloneHV pool mode.
//
// Trong chế độ CloneHV, app.go quản lý pool goroutine riêng thay vì dùng RunVerify,
// do đó cần gọi trực tiếp vào logic xử lý một account mà không đi qua toàn bộ
// dispatch loop của RunVerify. RunOneAccount cung cấp điểm truy cập public đó.
//
// Tất cả tham số và hành vi giống hệt runOneAccount — xem godoc của hàm đó.
// ⚠️ DEPRECATED: gọi với workerID=0 làm sticky manager cache chung 1 entry cho TẤT CẢ
// workers → tất cả cùng proxy + cùng session → defeat mục đích session rotation.
// Prefer RunOneAccountAt(workerID) khi worker có stable slot ID (CloneHV pool, split verify).
func RunOneAccount(ctx context.Context, acc AccountInput, config RunConfig, dateFolder string, onStatus StatusCallback) AccountResult {
	return runOneAccount(ctx, acc, config, dateFolder, 0, onStatus)
}

// RunOneAccountAt chạy 1 account với workerID cụ thể — sticky manager dùng workerID
// để phân biệt cache slot. Pass slot ID ổn định (CloneHV slotID, split verify threadIdx)
// để mỗi worker có proxy session riêng, không share.
func RunOneAccountAt(ctx context.Context, acc AccountInput, config RunConfig, dateFolder string, workerID int, onStatus StatusCallback) AccountResult {
	return runOneAccount(ctx, acc, config, dateFolder, workerID, onStatus)
}

// isAndroidVersionedPlatform báo cáo platform có FBAV cố định theo phiên bản app
// (S555/S556/S557/S558/S559). Các platform này dùng doc_id riêng cho từng version,
// nên UA cần khớp đúng FBAV — không thể pick UA tùy ý từ pool chung.
func isValidAndroidToken(tok string) bool {
	return strings.HasPrefix(tok, "EAAAAU") || strings.HasPrefix(tok, "EAAAAAY")
}

func isAndroidVersionedPlatform(platform string) bool {
	return platform == instagram.PlatformS415 ||
		platform == instagram.PlatformS425 ||
		platform == instagram.PlatformS435 ||
		platform == instagram.PlatformS445 ||
		platform == instagram.PlatformS416 ||
		platform == instagram.PlatformS417 ||
		platform == instagram.PlatformS418 ||
		platform == instagram.PlatformS419 ||
		platform == instagram.PlatformS420 ||
		platform == instagram.PlatformS421 ||
		platform == instagram.PlatformS422 ||
		platform == instagram.PlatformS423 ||
		platform == instagram.PlatformS424 ||
		platform == instagram.PlatformS426 ||
		platform == instagram.PlatformS427 ||
		platform == instagram.PlatformS428 ||
		platform == instagram.PlatformS429 ||
		platform == instagram.PlatformS430 ||
		platform == instagram.PlatformS431 ||
		platform == instagram.PlatformS432 ||
		platform == instagram.PlatformS433 ||
		platform == instagram.PlatformS434 ||
		platform == instagram.PlatformS436 ||
		platform == instagram.PlatformS437 ||
		platform == instagram.PlatformS438 ||
		platform == instagram.PlatformS439 ||
		platform == instagram.PlatformS440 ||
		platform == instagram.PlatformS441 ||
		platform == instagram.PlatformS442 ||
		platform == instagram.PlatformS443 ||
		platform == instagram.PlatformS444 ||
		platform == instagram.PlatformS446 ||
		platform == instagram.PlatformS447 ||
		platform == instagram.PlatformS448 ||
		platform == instagram.PlatformS449 ||
		platform == instagram.PlatformS450 ||
		platform == instagram.PlatformS451 ||
		platform == instagram.PlatformS452 ||
		platform == instagram.PlatformS453 ||
		platform == instagram.PlatformS454 ||
		platform == instagram.PlatformS455 ||
		platform == instagram.PlatformS456 ||
		platform == instagram.PlatformS457 ||
		platform == instagram.PlatformS458 ||
		platform == instagram.PlatformS459 ||
		platform == instagram.PlatformS460 ||
		platform == instagram.PlatformS461 ||
		platform == instagram.PlatformS462 ||
		platform == instagram.PlatformS463 ||
		platform == instagram.PlatformS464 ||
		platform == instagram.PlatformS465 ||
		platform == instagram.PlatformS466 ||
		platform == instagram.PlatformS467 ||
		platform == instagram.PlatformS468 ||
		platform == instagram.PlatformS469 ||
		platform == instagram.PlatformS470 ||
		platform == instagram.PlatformS471 ||
		platform == instagram.PlatformS472 ||
		platform == instagram.PlatformS473 ||
		platform == instagram.PlatformS474 ||
		platform == instagram.PlatformS475 ||
		platform == instagram.PlatformS476 ||
		platform == instagram.PlatformS477 ||
		platform == instagram.PlatformS478 ||
		platform == instagram.PlatformS479 ||
		platform == instagram.PlatformS480 ||
		platform == instagram.PlatformS481 ||
		platform == instagram.PlatformS482 ||
		platform == instagram.PlatformS483 ||
		platform == instagram.PlatformS484 ||
		platform == instagram.PlatformS485 ||
		platform == instagram.PlatformS486 ||
		platform == instagram.PlatformS487 ||
		platform == instagram.PlatformS488 ||
		platform == instagram.PlatformS489 ||
		platform == instagram.PlatformS490 ||
		platform == instagram.PlatformS491 ||
		platform == instagram.PlatformS492 ||
		platform == instagram.PlatformS493 ||
		platform == instagram.PlatformS494 ||
		platform == instagram.PlatformS496 ||
		platform == instagram.PlatformS497 ||
		platform == instagram.PlatformS498 ||
		platform == instagram.PlatformS499 ||
		platform == instagram.PlatformS495 ||
		platform == instagram.PlatformS555 ||
		platform == instagram.PlatformS555V2 ||
		platform == instagram.PlatformS556 ||
		platform == instagram.PlatformS557 ||
		platform == instagram.PlatformS558 ||
		platform == instagram.PlatformS558V2 ||
		platform == instagram.PlatformS556V2 ||
		platform == instagram.PlatformS557V2 ||
		platform == instagram.PlatformS553V2 ||
		platform == instagram.PlatformS554V2 ||
		platform == instagram.PlatformS551V2 ||
		platform == instagram.PlatformS552V2 ||
		platform == instagram.PlatformS550V2 ||
		platform == instagram.PlatformS559 ||
		platform == instagram.PlatformS559V2 ||
		platform == instagram.PlatformS560 ||
		platform == instagram.PlatformS560V2 ||
		platform == instagram.PlatformS560V3 ||
		platform == instagram.PlatformS561 ||
		platform == instagram.PlatformS561V2 ||
		platform == instagram.PlatformS561V3 ||
		platform == instagram.PlatformS561V99 ||
		platform == instagram.PlatformS561V4S21 ||
		platform == instagram.PlatformS561V4S23 ||
		platform == instagram.PlatformS562 ||
		platform == instagram.PlatformS562V3 ||
		platform == instagram.PlatformS562V4S21 ||
		platform == instagram.PlatformS562V4S23 ||
		platform == instagram.PlatformS563 ||
		platform == instagram.PlatformS563V2 ||
		platform == instagram.PlatformS563S21 ||
		platform == instagram.PlatformS563V3S21 ||
		platform == instagram.PlatformS563V4S21 ||
		platform == instagram.PlatformS563V4S23 ||
		platform == instagram.PlatformS563V5S21 ||
		platform == instagram.PlatformS563V5S23 ||
		platform == instagram.PlatformS563V6S21 ||
		platform == instagram.PlatformS563V6S23 ||
		platform == instagram.PlatformS564V1S21 ||
		platform == instagram.PlatformS564V1S23 ||
		platform == instagram.PlatformS564V2S21 ||
		platform == instagram.PlatformS564V2S23 ||
		platform == instagram.PlatformS564V3S21 ||
		platform == instagram.PlatformS564V3S23
}

// extractCountryFromProxy đọc suffix "/<cc>" cuối proxy string để lấy ISO-2 country.
//
// Format proxy convention trong app này: "user:pass@host:port/<cc>" hoặc "host:port/<cc>".
// Ví dụ: "u:p@1.2.3.4:8080/vn" → "VN". Trả "" nếu không match (proxy không có suffix
// country, hoặc dạng URL "scheme://..." mà segment cuối không phải 2 ký tự alpha).
//
// Chỉ chấp nhận đúng 2 ký tự alpha → tránh nhầm với port hoặc path khác.
func extractCountryFromProxy(proxyStr string) string {
	idx := strings.LastIndex(proxyStr, "/")
	if idx < 0 || idx >= len(proxyStr)-1 {
		return ""
	}
	suffix := proxyStr[idx+1:]
	if len(suffix) != 2 {
		return ""
	}
	for i := 0; i < 2; i++ {
		c := suffix[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return ""
		}
	}
	return strings.ToUpper(suffix)
}

// truncate cắt ngắn s về tối đa n ký tự đầu tiên, không thêm dấu "...".
//
// s: chuỗi gốc cần cắt — thường là token dài như fb_dtsg, lsd, datr.
// n: số ký tự tối đa muốn giữ lại; nếu len(s) <= n thì trả về s nguyên vẹn.
//
// Dùng trong notify log để in phần đầu của token (đủ để nhận dạng, không lộ toàn bộ)
// mà không làm dòng log quá dài.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
