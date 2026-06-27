package igandroid

import (
	"fmt"
	"os"
	"sync/atomic"
)

// captureSystemErrSample ghi 3 response system_error ĐẦU TIÊN ra file để phân tích
// (vì sao học jurisdiction không cứu được). Ghi vào CWD (= data dir của app).
var sysErrCaptureN int64
var noSessCaptureN int64

func captureSystemErrSample(resp string) {
	if n := atomic.AddInt64(&sysErrCaptureN, 1); n <= 3 {
		_ = os.WriteFile(fmt.Sprintf("system_error_sample_%d.txt", n), []byte(resp), 0644)
	}
}

// captureNoSessionSample ghi 3 response "no_session" đầu (không phải system_error/integrity)
// → xem có phải parser bỏ sót session không (có thể recover ~12% reg-success).
func captureNoSessionSample(resp string) {
	if n := atomic.AddInt64(&noSessCaptureN, 1); n <= 3 {
		_ = os.WriteFile(fmt.Sprintf("no_session_sample_%d.txt", n), []byte(resp), 0644)
	}
}

// Bộ đếm breakdown kết quả createAccount (atomic, thread-safe) — để ĐO loại lỗi nào
// áp đảo khi reg nhiều luồng → biết tối ưu gì:
//   - systemErr (jurisdiction lệch GeoIP) → code cứu được (retry/locale)
//   - integrity (IP bị flag/rate-limit)   → proxy / device-trust
//   - noSession (response lạ, không có session, không phải 2 loại trên)
//   - ok (createAccount thành công)
var (
	cntCreateOK  int64
	cntSystemErr int64
	cntIntegrity int64
	cntNoSession int64
)

func incCreateOK()  { atomic.AddInt64(&cntCreateOK, 1) }
func incSystemErr() { atomic.AddInt64(&cntSystemErr, 1) }
func incIntegrity() { atomic.AddInt64(&cntIntegrity, 1) }
func incNoSession() { atomic.AddInt64(&cntNoSession, 1) }

// RegErrorStats trả breakdown hiện tại (ok, system_error, integrity_block, no_session).
// Dùng từ app_register để emit lên status → user thấy loại lỗi nào nhiều.
func RegErrorStats() (ok, systemErr, integrity, noSession int64) {
	return atomic.LoadInt64(&cntCreateOK),
		atomic.LoadInt64(&cntSystemErr),
		atomic.LoadInt64(&cntIntegrity),
		atomic.LoadInt64(&cntNoSession)
}
