// Package facebook — Facebook session types, interfaces và login logic
// Mapping từ WeBM PlatformItems + WeSocial
package instagram

import (
	"context"
	"net/http"
	"net/http/cookiejar"

	"HVRIns/internal/email"
	"HVRIns/internal/instagram/fakeinfo"
)

// Session chứa toàn bộ trạng thái của một phiên đăng nhập Facebook.
//
// Struct này mapping từ WeBM PlatformItems — là đơn vị dữ liệu trung tâm
// được truyền xuyên suốt các bước verify B1-B5. Mọi thông tin cần thiết
// cho một account (cookie, token, proxy, client) đều nằm trong Session.
//
// Vòng đời của Session:
//  1. Khởi tạo với Cookie + Proxy + UserAgent từ file/database
//  2. LoginWithCookieMobile() điền FbDtsg, Jazoest, Lsd, Datr, UID, v.v.
//  3. Client + Jar được tạo một lần và tái dùng cho tất cả request B1-B5
//  4. Cookie được cập nhật sau mỗi bước verify (từ Set-Cookie response)
type Session struct {
	// UID là Facebook User ID của account (giá trị c_user trong cookie).
	// Được điền sau khi login thành công hoặc lấy từ database nếu đã biết.
	UID string

	// FullName là tên hiển thị của Facebook account.
	// Được extract từ response của các bước verify, không phải từ cookie.
	FullName string

	// Phone là số điện thoại liên kết với account.
	// Thường được extract từ trang profile hoặc settings trong quá trình verify.
	Phone string

	// Cookie là chuỗi cookie session dạng "name1=value1; name2=value2; ...".
	// Đây là dữ liệu đầu vào bắt buộc — LoginWithCookieMobile sẽ thất bại nếu rỗng.
	// Cookie được cập nhật sau login thành công với giá trị mới nhất từ CookieJar
	// (bao gồm các cookie được Facebook set thêm trong quá trình xác thực).
	Cookie string

	// Token là access token của Facebook account (nếu có).
	// Không được dùng trong cookie login flow — dành cho các flow khác (Graph API).
	Token string

	// UserAgent là chuỗi User-Agent gửi kèm mọi HTTP request của session.
	// Nên dùng mobile UA nhất quán với loại cookie (mobile cookie dùng mobile UA).
	// Rỗng sẽ dùng DefaultUserAgent() — iPhone/Chrome mobile.
	UserAgent string

	// Proxy là địa chỉ proxy của account, định dạng ip:port hoặc ip:port:user:pass.
	// Rỗng nếu account không dùng proxy (kết nối trực tiếp).
	Proxy string

	// === Session tokens — được extract từ HTML page của Facebook ===
	// Các token này có thời hạn sử dụng ngắn (thường theo phiên trình duyệt)
	// và cần được refresh khi session hết hạn.

	// FbDtsg là CSRF token quan trọng nhất — bắt buộc cho mọi POST request lên Facebook API.
	// Facebook thay đổi cách nhúng token nên ParseTokens dùng 4 fallback patterns để extract.
	FbDtsg string

	// Jazoest là security token phụ kèm theo fb_dtsg trong các form submission.
	// Default "21966" nếu không extract được từ HTML.
	Jazoest string

	// Lsd (Login State Data) là token chống CSRF cho login flow.
	// Default "P2XQp3VtEtx1ajSpA8Wbgw" nếu không extract được.
	Lsd string

	// Hsi là HTTP Signature Identifier — tham số __hsi trong các API request.
	// Facebook dùng để trace và validate request context.
	Hsi string

	// Rev là revision number của Facebook frontend build — tham số __rev.
	// Được gán từ SpinR sau khi parse (WeBM dùng __spin_r làm __rev).
	Rev string

	// S là session hash ngắn — tham số __s trong một số API endpoint.
	// Có thể rỗng — WeBM cũng cho phép rỗng.
	S string

	// Dyn là dynamic params string — tham số __dyn trong Bloks API request.
	// Thường rỗng trong các response từ verify flow.
	Dyn string

	// Csr là client state reference — tham số __csr.
	// Hiện chưa được extract, dành cho tương lai.
	Csr string

	// SpinR là __spin_r — revision number của resource bundle hiện tại.
	// Default "1015763339" nếu không extract được.
	SpinR string

	// SpinT là __spin_t — Unix timestamp của resource bundle.
	// Default "1723909047" nếu không extract được.
	SpinT string

	// Datr là "device authentication token" — cookie định danh thiết bị.
	// Facebook set datr khi lần đầu truy cập (trước khi có cookie account).
	// Cần gửi kèm trong API request để Facebook nhận diện đây là request từ browser hợp lệ.
	// Được extract từ CookieJar sau bước GET m.facebook.com/login/ không cookie.
	Datr string

	// === HTTP client — tái dùng xuyên suốt session ===

	// Client là http.Client được tạo một lần trong LoginWithCookieMobile và tái dùng
	// cho tất cả request trong session. Client mang theo Jar và proxy config.
	// Tương đương WeBM HttpClientRequest — một instance duy nhất cho cả luồng verify.
	// Quan hệ: Client.Jar == Jar (cùng một CookieJar instance).
	Client *http.Client

	// Jar là CookieJar dùng chung với Client — tự động capture Set-Cookie từ mọi response.
	// Cần giữ tham chiếu riêng để đọc/ghi cookie trực tiếp (extractDatrFromJar, importCookiesToJar).
	// Tương đương WeBM CookieContainer trong HttpClientRequest.
	Jar *cookiejar.Jar

	// === Dữ liệu đầu vào gốc — để lưu lại file sau verify ===

	// InputAccount là chuỗi account gốc đọc từ file input (thường là dòng text nguyên vẹn).
	// Lưu lại để ghi vào file output mà không bị biến đổi format.
	InputAccount string

	// Password là mật khẩu của account nếu có trong file input.
	// Không dùng trong cookie login flow nhưng được lưu lại để ghi ra file kết quả.
	Password string

	// DeviceID và FamilyDeviceID từ lúc reg — truyền sang verify để reg_info nhất quán.
	DeviceID       string
	FamilyDeviceID string

	// iOS partial reg tokens — xem RegResult.Srnonce / RegResult.SessionlessCryptedUID.
	Srnonce               string
	SessionlessCryptedUID string

	// AAC (Account Access Context) — token session client-mint lúc reg (iOS Messenger).
	// Set từ RegResult.AAC* để verify add-mail/confirm dùng lại đúng bộ aac của session create.
	AACJid string
	AACcs  string
	AACts  string

	// Flow-session IDs — UUID client-mint lúc reg (iOS Messenger). Set từ RegResult.RegFlowID/
	// HeadersFlowID để verify add-mail/confirm dùng lại đúng bộ flow_id của session create.
	RegFlowID     string
	HeadersFlowID string

	// PassRaw + PassTS — mật khẩu thô và Unix timestamp dùng trong #PWD_ENC:0:ts:pass.
	// Verify add-mail cần điền encrypted_password đúng vào reg_info (tránh template mismatch).
	PassRaw string
	PassTS  int64

	// Email — địa chỉ email đã dùng làm contactpoint khi register (Mode=TempMail).
	// Empty cho mode Phone hoặc Mail (giả).
	Email string

	// EmailMeta — credentials provider-specific (JSON-encoded) để verify khôi phục
	// session email tạm. Set từ RegResult.EmailMeta khi RegMode=TempMail.
	// Empty → verify dùng flow CreateEmail mới (existing behavior).
	EmailMeta string

	// Username — tên người dùng Instagram (vd "falcon.3900382").
	// Set từ AccountInput.Username khi verify IG account.
	// Dùng cho CheckLiveByCheckerCookie sau khi verify unknown.
	Username string
}

// LoginResult kết quả login
type LoginResult struct {
	Success bool
	Message string
	Session *Session
}

// ══════════════════════════════════════════════════════════════
// REGISTER TYPES — di chuyển từ internal/register/types.go
// ══════════════════════════════════════════════════════════════

// RegInput — dữ liệu đầu vào để tạo tài khoản Facebook
type RegInput struct {
	FirstName string // Tên, ví dụ: "Văn Hải"
	LastName  string // Họ, ví dụ: "Trần"
	Birthday  string // "DD-MM-YYYY", ví dụ: "22-05-2001"
	Gender    int    // 1=female, 2=male
	Phone     string // Số điện thoại, ví dụ: "0987247524" (không có +84)
	Email     string // Email contactpoint (Mode=Mail). Nếu set, ưu tiên hơn Phone — C# ModeReg=0.
	Password  string // Mật khẩu gốc (sẽ được encrypt trước khi gửi)
	Proxy     string // "ip:port" hoặc "ip:port:user:pass" hoặc "" (đã render session)
	ProxyKey  string // Base proxy (trước RenderSession) — dùng làm key cho session pool
	SlotIdx   int    // Index slot goroutine (1-based) — dùng làm key cho PartitionedDatrPool
	UserAgent string // Nếu rỗng dùng defaultRegUA
	DebugDir  string // Nếu set, save full response của từng bước vào thư mục này
	TutDatr   string // TUT mode: datr lấy từ cookie list, thay thế datr r.php ở B8
	// CookieInitialMethod — "file" / "new" / "ck" (C# method). Chỉ "ck" mới
	// trigger login-warm với uid|password; các mode khác chỉ extract datr.
	CookieInitialMethod string

	// EmailMeta — Snapshot blob từ email service đã CreateEmail TRƯỚC khi reg.
	// Caller (RunRegister spawner) tạo mail tạm + Snapshot + assign vào đây →
	// Register handler set RegResult.EmailMeta = input.EmailMeta để verify dùng.
	// Empty → mode Phone hoặc Mail (giả), không cần persist creds.
	EmailMeta string

	// UseOriginalUA — khi true, register handler force device/locale/SIM khớp với
	// OriginalUA của platform thay vì random. Chỉ thay FBCR theo OriginalSim để
	// match nhà mạng theo IP. Áp dụng cho s555-s559. Các platform khác bỏ qua.
	UseOriginalUA bool
	// OriginalSim — SIM dùng cho FBCR thay thế trong UA (đã set sẵn ở caller).
	// Khi UseOriginalUA=true, register handler override profile.Sim = OriginalSim
	// để HNI/MCC/MNC trong headers khớp với FBCR carrier. OperatorName rỗng → bỏ qua.
	OriginalSim fakeinfo.SimProfile

	// DelayStep — delay giữa các step (ms). Chỉ dùng cho platform step-by-step (s561v99).
	DelayStep int

	// GetOTP — callback đọc OTP từ email service (caller giữ mail handle live).
	// Dùng cho reg multi-step cần confirm OTP NGAY trong reg (Messenger appmv3reg).
	// Nil cho platform khác / mode không cần OTP trong reg.
	GetOTP func(ctx context.Context) (string, error)

	// GetNewEmail — nếu được cấp, registerer gọi callback này khi session bị flag (system_error)
	// để lấy email mới + GetOTP mới và retry toàn bộ flow với danh tính mới.
	// Signature: (ctx) → (email, getOTP, error).
	// Nil → registerer không tự retry khi SESSION_FLAGGED.
	GetNewEmail func(ctx context.Context) (email string, getOTP func(context.Context) (string, error), err error)
}

// RegSession — trạng thái phiên đăng ký (tokens + state từ server)
type RegSession struct {
	// HTTP tokens — lấy từ trang đăng ký
	FbDtsg  string
	Lsd     string
	Jazoest string
	Datr    string
	Rev     string // __rev
	Hsi     string // __hsi
	Dyn     string // __dyn
	S       string // __s — session identifier dạng "xxx:yyy:zzz"

	// Registration state — cập nhật sau mỗi bước
	WaterfallID string // UUID tạo 1 lần, dùng xuyên suốt B1-B8
	RegContext  string // Token lớn từ server, cập nhật sau mỗi bước
	RegInfo     string // JSON string của reg_info, cập nhật sau mỗi bước

	// Public key — để encrypt password ở B6
	PubKeyHex string // 32-byte Curve25519 public key dạng hex
	PubKeyID  string // key_id, ví dụ "6"
	PubKeyVer string // version, ví dụ "5"

	// HTTP config
	Proxy     string
	UserAgent string

	// Debug — nếu set thì save full response vào thư mục này (dùng cho testing)
	DebugDir string

	// Input reference
	Input *RegInput
}

// RegResult — kết quả đăng ký
type RegResult struct {
	Success        bool
	UID            string // Facebook UID của tài khoản mới
	Cookie         string // Cookie string: "c_user=...;xs=...;locale=...;fr=...;datr=..."
	AccessToken    string // Access token EAA... nếu server trả về
	Password       string // Mật khẩu gốc (không encrypt) để lưu ra file
	Message        string
	UserAgent      string // Android UA thực tế đã dùng trong request
	DeviceID       string // device_id dùng lúc reg (uuid) — truyền sang verify để reg_info nhất quán
	FamilyDeviceID string // family_device_id dùng lúc reg — truyền sang verify

	// Email — địa chỉ email đã dùng làm contactpoint khi RegMode=TempMail.
	// Empty cho mode Phone/Mail (giả).
	Email string

	// Phone — SĐT dùng làm contactpoint lúc create (iOS Mess phone-first).
	// Verify add-mail dùng làm msg_previous_cp (đổi phone→email → trigger OTP).
	Phone string

	// AAC (Account Access Context) — token session client-mint lúc reg create.
	// Verify add-mail/confirm PHẢI dùng lại y hệt bộ này (cùng session với crypted_user_id),
	// nếu không server render lại form thay vì gửi OTP. iOS Messenger only.
	AACJid string
	AACcs  string
	AACts  string

	// Flow-session IDs — UUID client-mint lúc reg, bind userid vào flow sống.
	// Verify add-mail/confirm dùng lại y hệt bộ create. iOS Messenger only.
	RegFlowID     string
	HeadersFlowID string

	// PassRaw + PassTS — mật khẩu thô và timestamp Unix dùng trong #PWD_ENC:0:ts:pass.
	// Verify add-mail cần điền lại encrypted_password đúng (không được để template placeholder).
	PassRaw string
	PassTS  int64

	// EmailMeta — credentials provider-specific (JSON-encoded) để verify khôi
	// phục mail tạm và đọc OTP từ inbox đã có sẵn (skip CreateEmail step).
	// Empty → verify dùng flow CreateEmail mới.
	//
	// Format: do provider tự quyết — caller treat như opaque blob.
	// Vd ZeusX: `{"email":"...","password":"...","refreshToken":"...","clientId":"..."}`
	// Vd Dropmail: `{"token":"...","sessId":"...","email":"..."}`
	EmailMeta string

	// iOS partial reg tokens — chỉ có giá trị với ios562 khi account vẫn còn nosess sau reg.
	// Srnonce: srnonce từ partial response cuối — dùng trong confirm server_params.
	// SessionlessCryptedUID: fb_partially_created_reg_info — dùng làm sessionless_crypted_user_id.
	Srnonce               string
	SessionlessCryptedUID string

	// Username — tên người dùng Instagram đã được tạo (vd "falcon.3900382").
	// Được sinh trong buildIGUsername() và lưu vào file kết quả với prefix "IGU:".
	Username string

	// LiveStatus — kết quả check live/die ngay sau khi reg thành công.
	// Giá trị: "live" | "checkpoint" | "suspended" | "die" | "unknown"
	// "unknown" = không xác định được (throttle / network lỗi) → coi như live.
	LiveStatus string
}

// ══════════════════════════════════════════════════════════════
// VERIFY TYPES — di chuyển từ internal/verify/verify.go
// ══════════════════════════════════════════════════════════════

// VerifyConfig cấu hình verify — mapping từ WeBM configInteraction
type VerifyConfig struct {
	// EmailPool shared pool mua batch email — nếu set sẽ dùng pool thay vì mua đơn lẻ
	EmailPool *email.CredPool `json:"-"`
	// OnEmailCreated callback khi temp/rent mail được tạo thành công trong verify flow.
	// Dùng để frontend hiển thị email vào cột EMAIL/PHONE realtime (trước khi verify done).
	OnEmailCreated func(email string) `json:"-"`
	// UserApiLabel — tên API VER user chọn trong UI (vd "api token", "api android",
	// "api mfb", "Fb_415"). Verifybase override Spec.Tag bằng "[{UserApiLabel}]" để log
	// hiển thị đúng tên user chọn (thay vì tag hardcoded của verify package nội bộ).
	// Empty → giữ Spec.Tag default (vd "[S23 Verify]").
	UserApiLabel      string `json:"-"`
	VerifyEnabled     bool   `json:"verifyEnabled"`
	MailProvider      string `json:"mailProvider"` // "0"=moakt, "1"=mail1sec, "zeus-x"=ZeusX
	MailList          string `json:"mailList"`     // custom mail list
	CheckLiveDie      bool   `json:"checkLiveDieEnabled"`
	TimeDelayCheck    int    `json:"timeDelayCheck"`    // giây chờ trước live check
	TimeDelaySendCode int    `json:"timeDelaySendCode"` // giây tối đa chờ OTP (Check OTP sau)
	DelayConfirmEmail int    `json:"delayConfirmEmail"` // giây chờ giữa addEmail và confirm (giả lập nhập code)
	DelayVeriReg      int    `json:"delayVeriReg"`      // giây chờ trước khi bắt đầu verify sau reg
	SendAgainCode     bool   `json:"sendAgainCode"`
	WaitMailMs        int    `json:"waitMailMs"` // ms giữa các lần poll OTP (Wait mail, 0=default 2000)
	MaxResend         int    `json:"maxResend"`  // số lần resend OTP tối đa (Try Send Code, 0=default 2)
	OutputPath        string `json:"outputPath"` // thư mục chứa Live.txt / Die.txt
	UAIphoneList      string `json:"uaIphoneList"`

	// ZeusX Hotmail config
	ZeusXApiKey      string `json:"zeusXApiKey"`
	ZeusXAccountCode string `json:"zeusXAccountCode"` // HOTMAIL, OUTLOOK, HOTMAIL_TRUSTED_GRAPH_API, ...

	// DongVanFB config
	DvfbApiKey      string `json:"dvfbApiKey"`
	DvfbAccountType string `json:"dvfbAccountType"` // account_type ID: "1"=HotMail NEW, "5"=Hotmail TRUSTED, ...

	// Store1s config
	Store1sApiKey    string `json:"store1sApiKey"`
	Store1sProductID string `json:"store1sProductId"` // product_id từ store1s.com (vd: "40559", "50510")

	// Mail30s config (mailotp.com / mail30s.com)
	Mail30sApiKey      string `json:"mail30sApiKey"`
	Mail30sProductSlug string `json:"mail30sProductSlug"` // product_slug từ mailotp.com (vd: "hotmail-oauth2")

	// TempMailLol config (api.tempmail.lol)
	TempMailLolApiKey string `json:"tempMailLolApiKey"` // optional Bearer token, free tier để trống

	// TempMailDomain domain tuỳ chỉnh cho moakt/mail1sec (vd: "tmpbox.net").
	TempMailDomain string `json:"tempMailDomain"`

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
	// "dongvan" (default) | "unlimit". Primary fail → fallback reader còn lại.
	OTPHotmailPriority string `json:"otpHotmailPriority,omitempty"`

	// TempMailToken — generic token/api key user nhập tay cho provider hiện hành.
	// Fallback cho các provider-specific field (tempMailLolApiKey, priyoEmailApiKey).
	TempMailToken string `json:"tempMailToken,omitempty"`

	// ═══ Advanced verify options (port C# MainFormUISettings) ═══

	// ReUseEmail — sau verify success, archive email để reuse cho account kế.
	// Mỗi email tái dùng tối đa UseEmailTime lần (port C# ReUseEmail).
	// Tiết kiệm chi phí rent mail — 1 email cho N account.
	ReUseEmail   bool `json:"reUseEmail"`
	UseEmailTime int  `json:"useEmailTime"` // số lần tối đa mỗi email được reuse (default 1 = không reuse)

	// FmUserTmpMail — khi dùng Temp Mail, format username theo login info (phone/email)
	// thay vì UUID random. Port C# FmUserTmpMail + StringUtils.CreateUsernameTmpMailFromLoginInf.
	// Email nhìn "thật" hơn (ví dụ "nguyenvan123@tmpbox.net" thay vì "xk9dh2p4@tmpbox.net").
	FmUserTmpMail bool `json:"fmUserTmpMail"`

	// DeepFakeLocale — khi true, dùng account locale trong verify API payload (AddEmail,
	// AddPhone, ConfirmCode) thay vì "en_US" default. Port C# MainFormUISettings.DeepFakeLocale.
	// Account locale được FB tin là "real" hơn — giảm tỷ lệ Checkpoint.
	DeepFakeLocale bool `json:"deepFakeLocale"`

	// UseProxyTempMail — khi poll temp mail, dùng proxy riêng từ Config/Proxy/proxy_tempmail.txt
	// thay vì proxy của account FB. Tránh temp mail rate limit IP.
	UseProxyTempMail bool `json:"useProxyTempMail"`

	// UseProxyGmail — khi dùng rent mail provider hỗ trợ proxy (zeus-x, muamail, unlimitmail),
	// pick proxy từ Config/Proxy/proxy_rentmail.txt. Provider khác bỏ qua flag này.
	// ⚠️ Một số rent provider có thể ban account API nếu phát hiện proxy/VPN.
	UseProxyGmail bool `json:"useProxyGmail"`

	// Enable2FA — sau khi verify email thành công, bật 2FA cho account.
	// Port C# MainFormUISettings.Enable2fa + SecurityFeatureAPI.TurnOnTwofactor.
	// Trả secret key 32-char để lưu vào SuccessVerify.txt kèm 2FA field.
	Enable2FA bool `json:"enable2fa"`

	// AddInfo — sau verify Live, cập nhật thông tin hồ sơ (thành phố, quê quán, trường, tình trạng hôn nhân).
	AddInfo *AddInfoConfig `json:"addInfo,omitempty"`
}

// AddInfoConfig cấu hình cập nhật thông tin hồ sơ sau verify thành công.
// DataDir chứa các file cities.txt, hometowns.txt, schools.txt theo format "pageID|name".
type AddInfoConfig struct {
	Enabled      bool   `json:"enabled"`
	City         bool   `json:"city"`
	Hometown     bool   `json:"hometown"`
	School       bool   `json:"school"`
	College      bool   `json:"college"`
	Work         bool   `json:"work"`
	Relationship bool   `json:"relationship"`
	DataDir      string `json:"dataDir"`
	DelayMs      int    `json:"delayMs"`
}

// VerifyResult kết quả verify cho 1 account
type VerifyResult struct {
	Success   bool
	Message   string
	Status    string // "Live", "Die", "Unknown"
	Email     string // email đã dùng để verify
	UserAgent string // UA thực tế dùng trong session verify
	TwoFA     string // TOTP secret nếu 2FA được bật thành công
}

// ══════════════════════════════════════════════════════════════
// FUTURE TYPES — skeleton cho android/, mfb/ platforms
// ══════════════════════════════════════════════════════════════

// FeedPost bài viết trên news feed
type FeedPost struct {
	PostID    string
	AuthorUID string
	Text      string
	Likes     int
}

// TwoFAResult kết quả bật 2FA
type TwoFAResult struct {
	Success       bool
	Secret        string // TOTP secret key
	RecoveryCodes []string
	Message       string
}
