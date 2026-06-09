// Package schema — field registry và metadata cho settings UI generation.
// Phase 1: định nghĩa cấu trúc registry. Phase 3+: dùng để auto-generate UI form.
package schema

// FieldType kiểu dữ liệu của một setting field
type FieldType string

const (
	FieldTypeString   FieldType = "string"
	FieldTypeInt      FieldType = "int"
	FieldTypeBool     FieldType = "bool"
	FieldTypeSelect   FieldType = "select"
	FieldTypePassword FieldType = "password" // sensitive — future: SecretStore
	FieldTypeTextarea FieldType = "textarea"
	FieldTypeKeyMap   FieldType = "keymap"
)

// FieldDomain nhóm domain của setting field
type FieldDomain string

const (
	DomainGlobal        FieldDomain = "global"
	DomainRuntime       FieldDomain = "runtime"
	DomainAccountSource FieldDomain = "accountSource"
	DomainProxy         FieldDomain = "proxy"
	DomainVerify        FieldDomain = "verify"
	DomainRegister      FieldDomain = "register"
	DomainMail          FieldDomain = "mail"
	DomainCaptcha       FieldDomain = "captcha"
	DomainOutput        FieldDomain = "output"
	DomainDevice        FieldDomain = "device"
)

// FieldMeta metadata của một settings field
type FieldMeta struct {
	Key         string      // dotted path: "profile.runtime.threadRequest"
	Label       string      // human-readable label (Vietnamese)
	Domain      FieldDomain // nhóm domain
	Type        FieldType   // kiểu input
	Required    bool        // bắt buộc điền
	Sensitive   bool        // true = field nhạy cảm, cần SecretStore trong future
	Min         *int        // giá trị min cho số
	Max         *int        // giá trị max cho số
	Options     []string    // các option cho FieldTypeSelect
	Description string      // mô tả chi tiết
}

// intPtr helper tạo *int
func intPtr(v int) *int { return &v }

// Registry danh sách tất cả field metadata đã đăng ký
var Registry = []FieldMeta{

	// ── Global ────────────────────────────────────────────────────────────────
	{
		Key: "global.loginPlatform", Label: "Nền tảng đăng nhập", Domain: DomainGlobal,
		Type: FieldTypeSelect, Options: []string{"facebook", "instagram", "bm"},
	},
	{
		Key: "global.loginMethod", Label: "Phương thức đăng nhập", Domain: DomainGlobal,
		Type: FieldTypeInt, Min: intPtr(0),
	},
	{
		Key: "global.saveRunColumn", Label: "Lưu cột lần chạy", Domain: DomainGlobal,
		Type: FieldTypeBool, Description: "Lưu thông tin cột lần chạy vào database sau mỗi lần chạy",
	},
	{
		Key: "global.backupDB", Label: "Sao lưu database", Domain: DomainGlobal,
		Type: FieldTypeBool, Description: "Tự động sao lưu database sau mỗi phiên chạy",
	},
	{
		Key: "global.closeAfterDone", Label: "Đóng app sau khi xong", Domain: DomainGlobal,
		Type: FieldTypeBool,
	},

	// ── Runtime ───────────────────────────────────────────────────────────────
	{
		Key: "profile.runtime.threadRequest", Label: "Số luồng Request", Domain: DomainRuntime,
		Type: FieldTypeInt, Required: true, Min: intPtr(1), Max: intPtr(600),
		Description: "Số luồng chạy song song tối đa",
	},
	{
		Key: "profile.runtime.delayRequest", Label: "Nghỉ giữa các request (ms)", Domain: DomainRuntime,
		Type: FieldTypeInt, Min: intPtr(0),
	},
	{
		Key: "profile.runtime.threadCheckInfo", Label: "Số luồng kiểm tra ẩn", Domain: DomainRuntime,
		Type: FieldTypeInt, Min: intPtr(1), Max: intPtr(100),
	},
	{
		Key: "profile.runtime.checkIpBeforeRun", Label: "Kiểm tra IP trước khi chạy", Domain: DomainRuntime,
		Type: FieldTypeBool,
	},
	{
		Key: "profile.runtime.delayChangeIp", Label: "Delay đổi IP (giây)", Domain: DomainRuntime,
		Type: FieldTypeInt, Min: intPtr(0),
	},

	// ── Account Source ────────────────────────────────────────────────────────
	{
		Key: "profile.account.source", Label: "Nguồn tài khoản", Domain: DomainAccountSource,
		Type: FieldTypeSelect, Options: []string{"folder", "api"},
	},
	{
		Key: "profile.account.folderPath", Label: "Thư mục nguồn", Domain: DomainAccountSource,
		Type: FieldTypeString,
	},
	{
		Key: "profile.account.cloneHv.enabled", Label: "Bật CloneHV", Domain: DomainAccountSource,
		Type: FieldTypeBool,
	},
	{
		Key: "profile.account.cloneHv.username", Label: "CloneHV Username", Domain: DomainAccountSource,
		Type: FieldTypeString,
	},
	{
		Key: "profile.account.cloneHv.password", Label: "CloneHV Password", Domain: DomainAccountSource,
		Type: FieldTypePassword, Sensitive: true,
		Description: "TODO(secrets-phase4): sẽ chuyển sang SecretStore",
	},
	{
		Key: "profile.account.cloneHv.productId", Label: "CloneHV Product ID", Domain: DomainAccountSource,
		Type: FieldTypeString,
	},
	{
		Key: "profile.account.cloneHv.amount", Label: "Số lượng mua mỗi lần", Domain: DomainAccountSource,
		Type: FieldTypeInt, Min: intPtr(1),
		Description: "Số tài khoản mua từ CloneHV mỗi lần khởi chạy",
	},

	// ── Proxy ─────────────────────────────────────────────────────────────────
	{
		Key: "profile.proxy.provider", Label: "Nhà cung cấp IP", Domain: DomainProxy,
		Type: FieldTypeSelect,
		Options: []string{"none", "proxy", "proxy_fixed", "fpt", "xproxy", "tinsoft", "shoplike", "netproxy", "minproxy", "netproxy_gb", "proxy_popular", "proxy_farm"},
	},
	{
		Key: "profile.proxy.proxyList", Label: "Danh sách Proxy", Domain: DomainProxy,
		Type: FieldTypeTextarea, Description: "host:port:user:pass — mỗi dòng một proxy",
	},
	{
		Key: "profile.proxy.proxyType", Label: "Loại Proxy", Domain: DomainProxy,
		Type: FieldTypeSelect, Options: []string{"http", "https", "socks5", "socks4"},
	},
	{
		Key: "profile.proxy.checkIpBeforeRun", Label: "Kiểm tra IP trước khi chạy", Domain: DomainProxy,
		Type: FieldTypeBool,
	},
	{
		Key: "profile.proxy.delayChangeIp", Label: "Delay đổi IP (giây)", Domain: DomainProxy,
		Type: FieldTypeInt, Min: intPtr(0),
	},
	{
		Key: "profile.proxy.providers.tinsoft.keys", Label: "Tinsoft Keys", Domain: DomainProxy,
		Type: FieldTypePassword, Sensitive: true,
	},
	{
		Key: "profile.proxy.providers.shoplike.keys", Label: "ShopLike Keys", Domain: DomainProxy,
		Type: FieldTypePassword, Sensitive: true,
	},
	{
		Key: "profile.proxy.providers.netproxy.keys", Label: "NetProxy Keys", Domain: DomainProxy,
		Type: FieldTypePassword, Sensitive: true,
	},
	{
		Key: "profile.proxy.providers.minproxy.keys", Label: "MinProxy Keys", Domain: DomainProxy,
		Type: FieldTypePassword, Sensitive: true,
	},
	{
		Key: "profile.proxy.providers.proxyfarm.keys", Label: "Proxy Farm Keys", Domain: DomainProxy,
		Type: FieldTypePassword, Sensitive: true,
	},
	{
		Key: "profile.proxy.providers.proxyfarm.accessToken", Label: "Proxy Farm Access Token", Domain: DomainProxy,
		Type: FieldTypePassword, Sensitive: true,
	},
	{
		Key: "profile.proxy.providers.proxypopular.keys", Label: "Proxy Dân Cư Keys", Domain: DomainProxy,
		Type: FieldTypePassword, Sensitive: true,
	},
	{
		Key: "profile.proxy.providers.proxypopular.accessToken", Label: "Proxy Dân Cư Access Token", Domain: DomainProxy,
		Type: FieldTypePassword, Sensitive: true,
	},

	// ── Verify ────────────────────────────────────────────────────────────────
	{
		Key: "profile.verify.enabled", Label: "Bật verify", Domain: DomainVerify,
		Type: FieldTypeBool,
	},
	{
		Key: "profile.verify.checkLiveDie", Label: "Kiểm tra live/die sau verify", Domain: DomainVerify,
		Type: FieldTypeBool,
	},
	{
		Key: "profile.verify.timeDelayCheck", Label: "Delay kiểm tra (s)", Domain: DomainVerify,
		Type: FieldTypeInt, Min: intPtr(0),
	},
	{
		Key: "profile.verify.timeDelaySendCode", Label: "Delay gửi code (s)", Domain: DomainVerify,
		Type: FieldTypeInt, Min: intPtr(0),
	},
	{
		Key: "profile.verify.sendAgainCode", Label: "Gửi lại code nếu thất bại", Domain: DomainVerify,
		Type: FieldTypeBool,
	},

	// ── Register ──────────────────────────────────────────────────────────────
	{
		Key: "profile.register.enabled", Label: "Bật tạo tài khoản tự động", Domain: DomainRegister,
		Type: FieldTypeBool,
	},
	{
		Key: "profile.register.type", Label: "Loại tài khoản tạo", Domain: DomainRegister,
		Type: FieldTypeSelect, Options: []string{"spam", "tut"},
		Description: "spam = tài khoản spam thông thường; tut = tài khoản TUT chất lượng cao",
	},
	{
		Key: "profile.register.cookieList", Label: "Danh sách Cookie", Domain: DomainRegister,
		Type: FieldTypeTextarea, Description: "Mỗi dòng một cookie — số dòng = số tài khoản sẽ tạo",
	},
	{
		Key: "profile.register.outputPath", Label: "Thư mục lưu tài khoản tạo", Domain: DomainRegister,
		Type: FieldTypeString,
	},

	// ── Mail ─────────────────────────────────────────────────────────────────
	{
		Key: "profile.mail.provider", Label: "Mail Provider", Domain: DomainMail,
		Type: FieldTypeSelect,
		Options: []string{"@tmpbox.net", "@i2b.vn", "mohmal", "zeus-x", "dongvanfb", "store1s", "mail30s"},
	},
	{
		Key: "profile.mail.mailList", Label: "Danh sách mail", Domain: DomainMail,
		Type: FieldTypeTextarea, Description: "Dùng khi provider là @tmpbox.net, @i2b.vn hoặc mohmal",
	},
	{
		Key: "profile.mail.providers.zeusx.apiKey", Label: "ZeusX API Key", Domain: DomainMail,
		Type: FieldTypePassword, Sensitive: true,
	},
	{
		Key: "profile.mail.providers.dvfb.apiKey", Label: "DongVanFB API Key", Domain: DomainMail,
		Type: FieldTypePassword, Sensitive: true,
	},
	{
		Key: "profile.mail.providers.store1s.apiKey", Label: "Store1s API Key", Domain: DomainMail,
		Type: FieldTypePassword, Sensitive: true,
	},
	{
		Key: "profile.mail.providers.mail30s.apiKey", Label: "Mail30s API Key", Domain: DomainMail,
		Type: FieldTypePassword, Sensitive: true,
	},

	// ── Captcha ───────────────────────────────────────────────────────────────
	{
		Key: "profile.captcha.provider", Label: "Captcha Provider", Domain: DomainCaptcha,
		Type: FieldTypeSelect, Options: []string{"2captcha", "omocaptcha", "ezcaptcha", "capsolver"},
	},
	{
		Key: "profile.captcha.keys.2captcha", Label: "2Captcha API Key", Domain: DomainCaptcha,
		Type: FieldTypePassword, Sensitive: true,
		Description: "TODO(secrets-phase4): sẽ chuyển sang SecretStore",
	},
	{
		Key: "profile.captcha.keys.omocaptcha", Label: "OmoCaptcha API Key", Domain: DomainCaptcha,
		Type: FieldTypePassword, Sensitive: true,
	},
	{
		Key: "profile.captcha.keys.ezcaptcha", Label: "EZCaptcha API Key", Domain: DomainCaptcha,
		Type: FieldTypePassword, Sensitive: true,
	},
	{
		Key: "profile.captcha.keys.capsolver", Label: "CapSolver API Key", Domain: DomainCaptcha,
		Type: FieldTypePassword, Sensitive: true,
	},

	// ── Output ────────────────────────────────────────────────────────────────
	{
		Key: "profile.output.verifyPath", Label: "Thư mục output Verify", Domain: DomainOutput,
		Type: FieldTypeString,
	},
	{
		Key: "profile.output.registerPath", Label: "Thư mục output Register", Domain: DomainOutput,
		Type: FieldTypeString,
	},

	// ── Device ────────────────────────────────────────────────────────────────
	{
		Key: "profile.device.uaList", Label: "Danh sách User-Agent iPhone", Domain: DomainDevice,
		Type: FieldTypeTextarea,
		Description: "Mỗi dòng một UA — được shuffle ngẫu nhiên theo luồng khi chạy",
	},
}

// GetByDomain lọc fields theo domain
func GetByDomain(domain FieldDomain) []FieldMeta {
	var result []FieldMeta
	for _, f := range Registry {
		if f.Domain == domain {
			result = append(result, f)
		}
	}
	return result
}

// GetSensitiveFields trả về các field có Sensitive = true (để tracking future secret migration)
func GetSensitiveFields() []FieldMeta {
	var result []FieldMeta
	for _, f := range Registry {
		if f.Sensitive {
			result = append(result, f)
		}
	}
	return result
}
