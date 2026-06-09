// archive.go — Email archive cho ReUseEmail option.
//
// Port C# ArchiveEmailCollection + MainFormUISettings.ReUseEmail + UseEmailTime:
//   - Sau verify success → email được lưu vào archive với UsedCount++
//   - Trước verify → check archive, nếu có email UsedCount < UseEmailTime → reuse
//   - Thread-safe qua sync.Mutex
//
// Archive chỉ áp dụng cho **Rent Mail** (trả tiền) để tiết kiệm chi phí.
// Temp mail free không cần archive (tạo mới gần như miễn phí).
package email

import (
	"fmt"
	"strings"
	"sync"
)

// ArchivedEmail là email đã verify success, chờ reuse cho account kế tiếp.
type ArchivedEmail struct {
	Address   string   // "abc@hotmail.com"
	Provider  string   // "zeus-x", "dongvanfb", ... — chỉ reuse cùng provider
	UsedCount int      // số lần đã reuse
	CodesUsed []string // codes OTP đã dùng (tránh đọc lại code cũ từ inbox)
	RawCred   string   // credential raw (cookie, token, password) để reuse được mailbox
}

// EmailArchive là pool các email đã verify success.
// Thread-safe. Singleton per app session — KHÔNG persist disk (đơn giản hóa).
type EmailArchive struct {
	mu       sync.Mutex
	emails   []ArchivedEmail
	maxUsage int // giới hạn UseEmailTime — email vượt sẽ bị xóa khỏi archive
}

// NewEmailArchive tạo archive mới với giới hạn reuse.
// maxUsage <= 0 → mặc định 1 (dùng 1 lần, không reuse thực tế).
func NewEmailArchive(maxUsage int) *EmailArchive {
	if maxUsage <= 0 {
		maxUsage = 1
	}
	return &EmailArchive{maxUsage: maxUsage}
}

// SetMaxUsage cập nhật giới hạn tái dùng — gọi khi user đổi UseEmailTime giữa batch.
func (a *EmailArchive) SetMaxUsage(n int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if n > 0 {
		a.maxUsage = n
	}
}

// Acquire lấy 1 email từ archive cho provider đã chọn. Return ok=false nếu không có.
// Email được MARK used (UsedCount++) trước khi trả — lần sau sẽ thấy count mới.
// Nếu UsedCount đạt maxUsage → email bị xóa khỏi archive luôn.
func (a *EmailArchive) Acquire(provider string) (ArchivedEmail, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i := range a.emails {
		if a.emails[i].Provider != provider {
			continue
		}
		// Tìm email có quota còn lại
		if a.emails[i].UsedCount < a.maxUsage {
			a.emails[i].UsedCount++
			acquired := a.emails[i]
			// Nếu đã dùng hết quota → xóa khỏi archive
			if a.emails[i].UsedCount >= a.maxUsage {
				a.emails = append(a.emails[:i], a.emails[i+1:]...)
			}
			return acquired, true
		}
	}
	return ArchivedEmail{}, false
}

// Archive thêm email vừa verify success vào pool.
// Dedup theo Address — nếu đã có thì skip (không reset UsedCount).
func (a *EmailArchive) Archive(e ArchivedEmail) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, ex := range a.emails {
		if strings.EqualFold(ex.Address, e.Address) {
			return // đã có
		}
	}
	a.emails = append(a.emails, e)
}

// RecordCodeUsed đánh dấu 1 OTP code đã được dùng cho email này.
// Lần WaitForCode sau sẽ skip các code này.
func (a *EmailArchive) RecordCodeUsed(address, code string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.emails {
		if strings.EqualFold(a.emails[i].Address, address) {
			a.emails[i].CodesUsed = append(a.emails[i].CodesUsed, code)
			return
		}
	}
}

// Size trả số email đang trong archive.
func (a *EmailArchive) Size() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.emails)
}

// Stats trả thống kê per provider.
func (a *EmailArchive) Stats() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	counts := map[string]int{}
	for _, e := range a.emails {
		counts[e.Provider]++
	}
	parts := make([]string, 0, len(counts))
	for p, c := range counts {
		parts = append(parts, fmt.Sprintf("%s=%d", p, c))
	}
	return fmt.Sprintf("archive[total=%d, %s]", len(a.emails), strings.Join(parts, ", "))
}

// SharedArchive — singleton dùng chung qua toàn app.
// App.go gọi email.SharedArchive() để get instance, truyền vào VerifyConfig.
var (
	sharedArchive     *EmailArchive
	sharedArchiveOnce sync.Once
)

// SharedArchive trả về email archive singleton (lazy init).
func SharedArchive() *EmailArchive {
	sharedArchiveOnce.Do(func() {
		sharedArchive = NewEmailArchive(1)
	})
	return sharedArchive
}

// ─── CreateUsernameFromLogin — port C# StringUtils.CreateUsernameTmpMailFromLoginInf ───

// CreateUsernameFromLogin format login info (phone/email) thành username an toàn
// cho email address. Port chính xác C# logic:
//
//	fblogin có "@" → lấy phần trước @
//	fblogin bắt đầu "+" → bỏ dấu +
//	else → giữ nguyên
func CreateUsernameFromLogin(fblogin string) string {
	fblogin = strings.TrimSpace(fblogin)
	if fblogin == "" {
		return ""
	}
	if strings.Contains(fblogin, "@") {
		return strings.Split(fblogin, "@")[0]
	}
	if strings.HasPrefix(fblogin, "+") {
		return strings.TrimPrefix(fblogin, "+")
	}
	return fblogin
}
