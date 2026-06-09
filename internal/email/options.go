// options.go — Cấu hình cho email.New() factory
package email

import "HVRIns/internal/email/rent"

// CredPool re-exported từ email/rent — dùng chung bởi Options.Pool, app.go, instagram.VerifyConfig.
type CredPool = rent.CredPool

// EmailCred re-exported từ email/rent.
type EmailCred = rent.EmailCred

// Options cấu hình để tạo email.Service qua factory New().
// Thay thế các flat fields đang nằm trong verify.Config.
type Options struct {
	// Provider: "moakt"/"0" | "mail1sec"/"1" | "mohmal" | "temporary-mail.net"
	//           "zeus-x" | "dongvanfb" | "store1s" | "mail30s"
	Provider string

	// ProxyStr proxy URL dùng cho temp providers (moakt, mail1sec, mohmal, guerrillamail)
	ProxyStr string

	// OnStatus callback nhận thông báo trạng thái từ provider (mua email, pool refill...)
	OnStatus func(string)

	// Pool shared credential pool — nếu set, provider sẽ pull từ pool thay vì mua đơn lẻ
	// Chỉ áp dụng cho rent providers (zeus-x, dongvanfb, store1s, mail30s)
	Pool *CredPool

	// ZeusX config
	ZeusXApiKey      string // API key cho api.zeus-x.ru
	ZeusXAccountCode string // HOTMAIL, OUTLOOK, HOTMAIL_TRUSTED_GRAPH_API, ... (default: "HOTMAIL")

	// DongVanFB config
	DvfbApiKey      string // API key cho api.dongvanfb.net
	DvfbAccountType string // account_type ID: "1"=HotMail NEW, "5"=Hotmail TRUSTED (default: "1")

	// Store1s config
	Store1sApiKey    string // API key cho store1s.com
	Store1sProductID string // product_id (vd: "40559" = Hotmail Trusted OAuth2, default)

	// Mail30s config (mailotp.com / mail30s.com)
	Mail30sApiKey      string // API key cho api.mailotp.com
	Mail30sProductSlug string // product_slug (vd: "hotmail-oauth2", default)

	// TempMailLol config (api.tempmail.lol)
	TempMailLolApiKey string // optional Bearer token — free tier để trống

	// TempMailDomain domain tuỳ chỉnh cho temp mail providers (moakt, mail1sec, tempmail-plus).
	// Ví dụ: "tmpbox.net", "i2b.vn". Để trống = dùng domain mặc định của provider.
	TempMailDomain string

	// MuaMail config (api.muamail.store)
	MuaMailApiKey   string // API key
	MuaMailProductID string // product_id

	// UnlimitMail config (unlimitmail.com)
	UnlimitMailApiKey    string // API key (token)
	UnlimitMailProductID string // product_id

	// SPTMail config (api.sptmail.com)
	SptMailApiKey      string // API key
	SptMailServiceCode string // otpServiceCode

	// EmailAPIInfo config (api.emailapi.info / gmail500.com)
	EmailAPIInfoApiKey      string // API key
	EmailAPIInfoProductCode string // product_code (vd: "gmail")

	// OtpCheap config (api.otp.cheap)
	OtpCheapApiKey    string // API key
	OtpCheapServiceID string // service_id (vd: "8" = Facebook)

	// ShopGmail9999 config (shopgmail9999.com)
	ShopGmail9999ApiKey  string // API key
	ShopGmail9999Service string // tên dịch vụ (vd: "facebook")

	// RentGmail config (rentgmail.online)
	RentGmailApiKey  string // API key (token)
	RentGmailPlatform string // nền tảng (vd: "facebook")

	// OtpCodesSms config (otpcodesms.site)
	OtpCodesSmsApiKey    string // API key
	OtpCodesSmsServiceID string // service_id

	// Wmemail config (www.wmemail.com)
	WmemailApiKey    string // token
	WmemailCommodity string // commodity_id (service id của gói hotmail)

	// PriyoEmail config (free.priyo.email) — yêu cầu API key từ v3.priyo.email
	PriyoEmailApiKey string

	// MailHV config (dulich360.com)
	MailHVApiKey string // api_token

	// VietXF config (vietxf.com)
	VietXFApiKey string // key query param

	// TempMailToken — token generic user nhập tay cho provider hiện hành.
	// Factory dùng làm fallback khi provider-specific field (TempMailLolApiKey, PriyoEmailApiKey) rỗng.
	TempMailToken string

	// CustomUsername — khi FmUserTmpMail=true, caller truyền username đã format
	// từ login info (phone/email). Providers support (moakt, mail1sec, mail.tm...)
	// sẽ dùng làm prefix email thay vì random UUID.
	// "" = dùng random như cũ.
	CustomUsername string

	// ProxyOverride — nếu UseProxyTempMail=true, caller truyền proxy riêng (từ
	// Config/Proxy/proxy_tempmail.txt) thay vì ProxyStr của account FB.
	// Providers sẽ prefer ProxyOverride > ProxyStr. "" = dùng ProxyStr.
	ProxyOverride string

	// OTPHotmailPriority — nguồn đọc OTP ưu tiên cho 7 providers Hotmail OAuth2
	// (zeus-x, dongvanfb, store1s, mail30s, muamail, unlimitmail, wmemail).
	// "dongvan" (default) → tools.dongvanfb.net/api/get_code_oauth2  ~2.6s
	// "unlimit"           → smail1s.com/get_messages?mode=oauth     ~4.5s
	// Nếu primary fail → tự fallback sang reader còn lại. "" = dongvan.
	OTPHotmailPriority string
}
