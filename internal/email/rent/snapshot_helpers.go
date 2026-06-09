package rent

// snapshot_helpers.go — Shared types và helpers cho Snapshotter implementations
// trên 15 rent providers.
//
// Pattern chung: hầu hết rent providers (zeus-x, dvfb, store1s, mail30s, ...)
// dùng OAuth2 refresh_token + clientId pattern. Vài provider (rentgmail,
// otpcheap) chỉ cần email + token đơn giản. Vài provider (mua đơn lẻ) thậm chí
// chỉ cần email — fetch OTP qua server-side với mailbox identifier.
//
// CommonOAuth2Snapshot — dùng cho providers Hotmail/Outlook OAuth2 pattern:
//   ZeusX, DongVanFB, Store1s, Mail30s, MuaMail, UnlimitMail, EmailAPIInfo,
//   OtpCheap, ShopGmail9999, RentGmail, OtpCodesSms, Wmemail, PriyoEmail
//
// Email-only snapshot (rare): SptMail
//
// Tất cả providers KHÔNG có refund API → Release returns nil. Caller dùng
// type assertion `if r, ok := svc.(Releaser); ok` — provider nào hỗ trợ
// override Release() trả real refund call; mặc định no-op.

// CommonOAuth2Snapshot — JSON shape cho providers dùng email + OAuth2 creds.
type CommonOAuth2Snapshot struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	RefreshToken string `json:"refresh_token"`
	ClientId     string `json:"client_id"`
}

// EmailOnlySnapshot — providers chỉ cần email + token để fetch OTP.
type EmailOnlySnapshot struct {
	Email string `json:"email"`
	Token string `json:"token,omitempty"` // mailbox token / order_id / message_id
}
