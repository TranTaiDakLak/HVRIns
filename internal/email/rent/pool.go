// pool.go — Shared email credential pool với batch purchasing
// Nhiều goroutines dùng chung 1 pool → giảm số lần gọi API, tiết kiệm chi phí
package rent

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// defaultFirstBatchSize là giá trị mặc định khi caller không truyền batchSize.
const defaultFirstBatchSize = 50

// EmailCred thông tin đăng nhập 1 email đã mua
type EmailCred struct {
	Email        string
	Password     string
	RefreshToken string
	ClientId     string
}

// CredPool — shared pool email credential.
//
// Mô hình mua:
//  1. Pool còn mail → lấy từ pool, không mua thêm.
//  2. Pool rỗng + chưa mua batch đầu → mua đúng 50 con (serialize: 1 goroutine mua,
//     các goroutine khác chờ 100ms rồi lấy từ batch đó).
//  3. Pool cạn hẳn sau batch đầu → mỗi luồng tự mua 1 con độc lập.
//
// Mail trả về qua Return() được ưu tiên dùng trước (tái sử dụng, tiết kiệm chi phí).
type CredPool struct {
	mu           sync.Mutex
	creds        []EmailCred
	refilling      bool // đang mua batch (serialize guard)
	exhausted      bool // đã hết email và mua thất bại
	firstBatchSize int  // số email mua mỗi batch (cấu hình từ UI, mặc định 50)
	buyFn        func(ctx context.Context, n int) ([]EmailCred, error)
	notify       func(string)
	OnExhausted  func(err error)         // callback khi pool cạn kiệt và mua thất bại
	OnBought     func(creds []EmailCred) // callback khi MUA THÀNH CÔNG (KHÔNG fire khi Return) — dùng để lưu mail ra file
	boughtIndex  map[string]EmailCred    // MỌI mail đã mua trong run (key=email) — để cuối run phân loại used/unused
	persisted    bool                   // đã dump used/unused ra file chưa (đảm bảo 1 lần/pool)
	Provider     string                 // tên provider (mail30s/store1s/...) — để đặt tên file used/unused khi dump
}

// TryMarkPersisted trả true ĐÚNG 1 lần đầu cho mỗi pool — dùng để dump used/unused
// đúng 1 lần dù được gọi từ nhiều điểm (run-end + trước Close).
func (p *CredPool) TryMarkPersisted() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.persisted {
		return false
	}
	p.persisted = true
	return true
}

// fireBought ghi nhận batch mail vừa mua vào boughtIndex + gọi OnBought (nếu set).
// Chỉ gọi sau buyFn thành công — KHÔNG gọi trong Return (mail recycle đã được mua trước đó).
func (p *CredPool) fireBought(creds []EmailCred) {
	if len(creds) == 0 {
		return
	}
	p.mu.Lock()
	if p.boughtIndex == nil {
		p.boughtIndex = make(map[string]EmailCred)
	}
	for _, c := range creds {
		if c.Email != "" {
			p.boughtIndex[c.Email] = c
		}
	}
	p.mu.Unlock()
	if p.OnBought != nil {
		p.OnBought(creds)
	}
}

// PartitionUsedUnused phân loại MỌI mail đã mua trong run thành 2 nhóm:
//   - unused: còn nằm trong pool lúc gọi → CHƯA dùng, sạch, tái sử dụng được lần sau.
//   - used:   đã lấy khỏi pool và KHÔNG còn trong pool → đã đưa vào verify (FB có thể đã consume).
//
// Mail recycle (Get→Return→còn trong pool) được tính là unused (đúng: vẫn sạch).
// Gọi lúc KẾT THÚC run (sau khi mọi verify xong) để dump ra file.
func (p *CredPool) PartitionUsedUnused() (used, unused []EmailCred) {
	p.mu.Lock()
	defer p.mu.Unlock()
	unusedSet := make(map[string]struct{}, len(p.creds))
	for _, c := range p.creds {
		unusedSet[c.Email] = struct{}{}
		unused = append(unused, c)
	}
	for email, c := range p.boughtIndex {
		if _, ok := unusedSet[email]; !ok {
			used = append(used, c)
		}
	}
	return used, unused
}

// NewCredPool tạo một CredPool mới để chia sẻ giữa nhiều goroutine.
// batchSize: số email mua batch đầu (mặc định 50 nếu < 1). threshold bị bỏ qua.
func NewCredPool(batchSize, threshold int, buyFn func(context.Context, int) ([]EmailCred, error), notify func(string)) *CredPool {
	if batchSize < 1 {
		batchSize = defaultFirstBatchSize
	}
	return &CredPool{
		firstBatchSize: batchSize,
		buyFn:          buyFn,
		notify:         notify,
	}
}

func (p *CredPool) log(msg string) {
	if p.notify != nil {
		p.notify(msg)
	}
}

// Get lấy 1 EmailCred. An toàn khi gọi đồng thời từ nhiều goroutine.
//
//  1. Pool còn mail → lấy ngay, không mua thêm.
//  2. Pool rỗng → serialize: 1 goroutine mua batch (firstBatchSize con), các goroutine khác chờ.
//     Áp dụng cả lần đầu lẫn các lần sau — luôn mua theo số lượng đã cài đặt.
func (p *CredPool) Get(ctx context.Context) (EmailCred, error) {
	for {
		select {
		case <-ctx.Done():
			return EmailCred{}, ctx.Err()
		default:
		}

		p.mu.Lock()

		// 1. Pool còn mail → lấy ngay, không mua thêm.
		if len(p.creds) > 0 {
			cred := p.creds[0]
			p.creds = p.creds[1:]
			p.mu.Unlock()
			return cred, nil
		}

		// 2. Pool rỗng + goroutine khác đang mua → chờ rồi lấy từ batch đó.
		if p.refilling {
			p.mu.Unlock()
			select {
			case <-ctx.Done():
				return EmailCred{}, ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
			continue
		}

		// 3. Pool rỗng + chưa ai mua → ta mua batch.
		p.refilling = true
		p.mu.Unlock()

		p.log(fmt.Sprintf("[EmailPool] Pool hết — mua thêm %d con...", p.firstBatchSize))
		creds, err := p.buyFn(ctx, p.firstBatchSize)

		p.mu.Lock()
		p.refilling = false
		if err == nil && len(creds) > 0 {
			p.creds = append(p.creds, creds...)
			p.exhausted = false
		}
		p.mu.Unlock()

		if err == nil && len(creds) > 0 {
			p.fireBought(creds)
		}
		if err != nil {
			p.fireExhausted(err)
			return EmailCred{}, err
		}
		// quay lại đầu vòng lặp → lấy 1 con từ batch vừa mua
	}
}

// fireExhausted gọi OnExhausted ĐÚNG 1 lần khi pool cạn + mua thất bại.
func (p *CredPool) fireExhausted(err error) {
	p.mu.Lock()
	first := !p.exhausted && p.OnExhausted != nil
	if first {
		p.exhausted = true
	}
	cb := p.OnExhausted
	p.mu.Unlock()
	if first {
		cb(err)
	}
}

// Return trả 1 credential CHƯA DÙNG về pool để account khác tái sử dụng.
// Dùng khi verify fail SỚM (add mail HTTP error / account checkpoint) mà mail
// CHƯA bị FB consume → tránh mua mail mới lãng phí.
//
// Prepend vào ĐẦU pool → mail trả về được lấy ra TRƯỚC (ưu tiên reuse ngay,
// tránh mail tồn lâu trong queue có thể hết hạn token OAuth2).
//
// LƯU Ý: chỉ gọi với mail PRISTINE (chưa add thành công vào FB account nào).
// KHÔNG gọi với mail đã add OK (OTP timeout) — sẽ gây "email already used"
// khi account khác dùng lại.
func (p *CredPool) Return(cred EmailCred) {
	if cred.Email == "" {
		return
	}
	p.mu.Lock()
	p.creds = append([]EmailCred{cred}, p.creds...)
	p.exhausted = false
	p.mu.Unlock()
	p.log(fmt.Sprintf("[EmailPool] Trả mail %s về pool (reuse, tránh phí) — còn %d", cred.Email, len(p.creds)))
}

// Close giải phóng credentials slice — gọi khi run kết thúc.
func (p *CredPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.creds = nil
	p.exhausted = false
}

// Size trả về số credential hiện có trong pool — cho monitoring/debug.
func (p *CredPool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.creds)
}

func poolMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
