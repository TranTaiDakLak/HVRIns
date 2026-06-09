// Package email — Interface chung cho email services
// Mapping từ WeBM MoaktMail + Mail1secMail
package email

import "context"

// Service interface chung cho tất cả email providers
type Service interface {
	// CreateEmail tạo email tạm, trả về địa chỉ email
	CreateEmail(ctx context.Context) (string, error)

	// WaitForCode poll email inbox, extract OTP code từ Facebook
	// maxRetry: số lần thử (default 12)
	// intervalMs: delay giữa mỗi lần poll (default 2000ms)
	WaitForCode(ctx context.Context, maxRetry int, intervalMs int) (string, error)

	// GetEmail trả về địa chỉ email đã tạo
	GetEmail() string

	// Close cleanup resources
	Close()
}

// ─── Optional capability interfaces (TempMail Reuse — register→verify) ────
//
// Providers implement THÊM (không bắt buộc) để hỗ trợ chức năng reuse mail
// giữa register và verify. Caller dùng type assertion để check capability:
//
//	if s, ok := svc.(Snapshotter); ok {
//	    creds, _ := s.Snapshot()
//	    // save to account.EmailMeta — verify sau dùng để Restore
//	}
//
// Provider chưa implement → fall back về flow CreateEmail mới (existing behavior).

// Snapshotter — provider hỗ trợ serialize state ra string để verify dùng lại.
//
// State include credentials provider-specific: refreshToken/clientId (OAuth),
// sessionID/cookie (free temp), order_id (rent), v.v.
//
// Serialize format: tự do (JSON khuyến nghị) — chỉ provider tự đọc lại qua Restore.
// Caller treat như opaque blob.
type Snapshotter interface {
	// Snapshot serialize internal state ra string. Trả "" + nil nếu state chưa
	// đủ để restore (vd CreateEmail chưa được gọi).
	Snapshot() (string, error)

	// Restore re-init state từ blob đã Snapshot. Sau Restore, GetEmail() phải
	// trả đúng email gốc + WaitForCode work với inbox đã có sẵn.
	//
	// Sẽ KHÔNG gọi CreateEmail sau Restore — provider chuẩn bị sẵn HTTP client,
	// auth headers, etc. để WaitForCode chạy ngay.
	Restore(creds string) error
}

// Releaser — provider hỗ trợ trả mail về pool (refund/release sau register fail).
//
// Caller gọi khi reg fail → mail không cần verify nữa → return về pool cho
// account khác dùng (tiết kiệm chi phí).
//
// No-op cho provider không hỗ trợ refund (vd public free temp mail).
type Releaser interface {
	// Release trả mail hiện tại về pool/refund. Idempotent — gọi nhiều lần OK.
	Release(ctx context.Context) error
}

// SnapshotIfPossible — helper type-assert + safe call. Trả ("", false) nếu
// provider không hỗ trợ hoặc có lỗi.
func SnapshotIfPossible(svc Service) (string, bool) {
	s, ok := svc.(Snapshotter)
	if !ok {
		return "", false
	}
	creds, err := s.Snapshot()
	if err != nil || creds == "" {
		return "", false
	}
	return creds, true
}

// RestoreIfPossible — helper type-assert + safe call. Trả false nếu provider
// không hỗ trợ Restore hoặc creds invalid.
func RestoreIfPossible(svc Service, creds string) bool {
	if creds == "" {
		return false
	}
	s, ok := svc.(Snapshotter)
	if !ok {
		return false
	}
	return s.Restore(creds) == nil
}

// ReleaseIfPossible — helper safe-call Release. Best-effort; ignore lỗi.
func ReleaseIfPossible(ctx context.Context, svc Service) {
	if r, ok := svc.(Releaser); ok {
		_ = r.Release(ctx)
	}
}
