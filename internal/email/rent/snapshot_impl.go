package rent

// snapshot_impl.go — Snapshotter + Releaser cho 11 rent providers còn lại
// (zeus_x.go và dongvanfb.go có inline impl trong file riêng).
//
// Providers:
//   OAuth2 (4): Mail30s, MuaMail, Store1s, UnlimitMail, Wmemail
//   EmailOnly (5): EmailAPIInfo, OtpCheap, RentGmail, ShopGmail9999, SPTMail
//   SMS (1): OtpCodesSms — cũng dùng EmailOnly pattern (token = requestID+number)
//
// Tất cả providers KHÔNG có refund API hiện tại → Release returns nil.
// Khi nhà cung cấp thêm API refund, override Release() trong file gốc.

import (
	"context"
	"encoding/json"
)

// ─── OAuth2 pattern (5 providers) ──────────────────────────────────────────

func (m *Mail30s) Snapshot() (string, error) {
	if m.emailAddr == "" {
		return "", nil
	}
	b, err := json.Marshal(CommonOAuth2Snapshot{
		Email: m.emailAddr, Password: m.password,
		RefreshToken: m.refreshToken, ClientId: m.clientID,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (m *Mail30s) Restore(creds string) error {
	var s CommonOAuth2Snapshot
	if err := json.Unmarshal([]byte(creds), &s); err != nil {
		return err
	}
	m.emailAddr, m.password, m.refreshToken, m.clientID = s.Email, s.Password, s.RefreshToken, s.ClientId
	return nil
}
func (m *Mail30s) Release(ctx context.Context) error {
	if m.pool != nil && m.emailAddr != "" {
		m.pool.Return(EmailCred{Email: m.emailAddr, Password: m.password, RefreshToken: m.refreshToken, ClientId: m.clientID})
		m.emailAddr = "" // mark consumed → tránh double-return
	}
	return nil
}

func (m *MuaMail) Snapshot() (string, error) {
	if m.emailAddr == "" {
		return "", nil
	}
	b, err := json.Marshal(CommonOAuth2Snapshot{
		Email: m.emailAddr, Password: m.password,
		RefreshToken: m.refreshToken, ClientId: m.clientID,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (m *MuaMail) Restore(creds string) error {
	var s CommonOAuth2Snapshot
	if err := json.Unmarshal([]byte(creds), &s); err != nil {
		return err
	}
	m.emailAddr, m.password, m.refreshToken, m.clientID = s.Email, s.Password, s.RefreshToken, s.ClientId
	return nil
}
func (m *MuaMail) Release(ctx context.Context) error { return nil }

func (s *Store1s) Snapshot() (string, error) {
	if s.emailAddr == "" {
		return "", nil
	}
	b, err := json.Marshal(CommonOAuth2Snapshot{
		Email: s.emailAddr, Password: s.password,
		RefreshToken: s.refreshToken, ClientId: s.clientID,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (s *Store1s) Restore(creds string) error {
	var snap CommonOAuth2Snapshot
	if err := json.Unmarshal([]byte(creds), &snap); err != nil {
		return err
	}
	s.emailAddr, s.password, s.refreshToken, s.clientID = snap.Email, snap.Password, snap.RefreshToken, snap.ClientId
	return nil
}
func (s *Store1s) Release(ctx context.Context) error {
	if s.pool != nil && s.emailAddr != "" {
		s.pool.Return(EmailCred{Email: s.emailAddr, Password: s.password, RefreshToken: s.refreshToken, ClientId: s.clientID})
		s.emailAddr = "" // mark consumed → tránh double-return
	}
	return nil
}

func (u *UnlimitMail) Snapshot() (string, error) {
	if u.emailAddr == "" {
		return "", nil
	}
	b, err := json.Marshal(CommonOAuth2Snapshot{
		Email: u.emailAddr, Password: u.password,
		RefreshToken: u.refreshToken, ClientId: u.clientID,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (u *UnlimitMail) Restore(creds string) error {
	var s CommonOAuth2Snapshot
	if err := json.Unmarshal([]byte(creds), &s); err != nil {
		return err
	}
	u.emailAddr, u.password, u.refreshToken, u.clientID = s.Email, s.Password, s.RefreshToken, s.ClientId
	return nil
}
func (u *UnlimitMail) Release(ctx context.Context) error { return nil }

func (w *Wmemail) Snapshot() (string, error) {
	if w.emailAddr == "" {
		return "", nil
	}
	b, err := json.Marshal(CommonOAuth2Snapshot{
		Email: w.emailAddr, Password: w.password,
		RefreshToken: w.refreshToken, ClientId: w.clientID,
	})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (w *Wmemail) Restore(creds string) error {
	var s CommonOAuth2Snapshot
	if err := json.Unmarshal([]byte(creds), &s); err != nil {
		return err
	}
	w.emailAddr, w.password, w.refreshToken, w.clientID = s.Email, s.Password, s.RefreshToken, s.ClientId
	return nil
}
func (w *Wmemail) Release(ctx context.Context) error { return nil }

// ─── EmailOnly pattern (5 providers) ───────────────────────────────────────

func (e *EmailAPIInfo) Snapshot() (string, error) {
	if e.emailAddr == "" {
		return "", nil
	}
	b, err := json.Marshal(EmailOnlySnapshot{Email: e.emailAddr, Token: e.orderNo})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (e *EmailAPIInfo) Restore(creds string) error {
	var s EmailOnlySnapshot
	if err := json.Unmarshal([]byte(creds), &s); err != nil {
		return err
	}
	e.emailAddr, e.orderNo = s.Email, s.Token
	return nil
}
func (e *EmailAPIInfo) Release(ctx context.Context) error { return nil }

func (o *OtpCheap) Snapshot() (string, error) {
	if o.emailAddr == "" {
		return "", nil
	}
	b, err := json.Marshal(EmailOnlySnapshot{Email: o.emailAddr, Token: o.quid})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (o *OtpCheap) Restore(creds string) error {
	var s EmailOnlySnapshot
	if err := json.Unmarshal([]byte(creds), &s); err != nil {
		return err
	}
	o.emailAddr, o.quid = s.Email, s.Token
	return nil
}
func (o *OtpCheap) Release(ctx context.Context) error { return nil }

func (r *RentGmail) Snapshot() (string, error) {
	if r.emailAddr == "" {
		return "", nil
	}
	b, err := json.Marshal(EmailOnlySnapshot{Email: r.emailAddr, Token: r.orderID})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (r *RentGmail) Restore(creds string) error {
	var s EmailOnlySnapshot
	if err := json.Unmarshal([]byte(creds), &s); err != nil {
		return err
	}
	r.emailAddr, r.orderID = s.Email, s.Token
	return nil
}
func (r *RentGmail) Release(ctx context.Context) error { return nil }

func (s *ShopGmail9999) Snapshot() (string, error) {
	if s.emailAddr == "" {
		return "", nil
	}
	b, err := json.Marshal(EmailOnlySnapshot{Email: s.emailAddr, Token: s.orderID})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (s *ShopGmail9999) Restore(creds string) error {
	var snap EmailOnlySnapshot
	if err := json.Unmarshal([]byte(creds), &snap); err != nil {
		return err
	}
	s.emailAddr, s.orderID = snap.Email, snap.Token
	return nil
}
func (s *ShopGmail9999) Release(ctx context.Context) error { return nil }

func (s *SPTMail) Snapshot() (string, error) {
	if s.emailAddr == "" {
		return "", nil
	}
	b, err := json.Marshal(EmailOnlySnapshot{Email: s.emailAddr})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (s *SPTMail) Restore(creds string) error {
	var snap EmailOnlySnapshot
	if err := json.Unmarshal([]byte(creds), &snap); err != nil {
		return err
	}
	s.emailAddr = snap.Email
	return nil
}
func (s *SPTMail) Release(ctx context.Context) error { return nil }

// ─── SMS pattern (1) ──────────────────────────────────────────────────────
// OtpCodesSms — không có email field, chỉ requestID + number. Snapshot encode
// requestID làm Token. Restore re-init để fetch OTP từ existing request.

func (o *OtpCodesSms) Snapshot() (string, error) {
	if o.requestID == "" {
		return "", nil
	}
	b, err := json.Marshal(EmailOnlySnapshot{Email: o.number, Token: o.requestID})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func (o *OtpCodesSms) Restore(creds string) error {
	var s EmailOnlySnapshot
	if err := json.Unmarshal([]byte(creds), &s); err != nil {
		return err
	}
	o.number, o.requestID = s.Email, s.Token
	return nil
}
func (o *OtpCodesSms) Release(ctx context.Context) error { return nil }
