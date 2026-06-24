// factory.go — Factory chính cho email.Service
// Tất cả mail provider (temp + rent) đều KHÔNG dùng proxy account.
// Mapping từ C# gốc: UseProxy = false cho toàn bộ mail server.
package email

import (
	"fmt"

	"HVRIns/internal/email/rent"
	"HVRIns/internal/email/temp"
)

// New tạo email.Service phù hợp theo opts.Provider.
// Là factory tập trung thay thế createEmailService() trong verify.
//
// Proxy behavior (priority order):
//  1. UseProxyTempMail=true → opts.ProxyOverride = random từ Config/Proxy/proxy_tempmail.txt
//     → temp mail dùng proxy pool riêng (tách biệt với proxy verify FB)
//  2. UseProxyTempMail=false → opts.ProxyOverride = ""
//     → fallback opts.ProxyStr = proxy đang verify account
//     → mail đọc CÙNG IP với request FB → anti-fraud detection tốt hơn
//     (FB không thấy account đăng ký từ IP X, mail check từ IP Y đáng ngờ)
//  3. Nếu cả 2 rỗng → connect trực tiếp IP user (không khuyến nghị)
//
// Rent mail policy:
//   - zeus-x, muamail, unlimitmail: CÓ hỗ trợ proxy, nhận opts.ProxyOverride
//     (chỉ set khi UseProxyGmail=true — verify pick từ proxy_rentmail.txt).
//     KHÔNG fallback về opts.ProxyStr (proxy FB) để tránh dùng datacenter IP
//     mà rent provider có thể đã blacklist.
//   - dongvanfb, store1s, mail30s, sptmail, otpcheap, ...: luôn gọi trực tiếp
//     không proxy (constructor chưa accept proxy hoặc provider chính sách ban).
func New(opts Options) (Service, error) {
	// Proxy hiệu lực cho TEMP mail: ưu tiên ProxyOverride, fallback ProxyStr.
	// Mail reading dùng cùng IP với verify request → anti-fraud tốt hơn.
	effectiveProxy := opts.ProxyOverride
	if effectiveProxy == "" {
		effectiveProxy = opts.ProxyStr
	}
	// Alias cũ giữ cho back-compat với Mail1sec branch (dùng tên khác).
	mail1secProxy := effectiveProxy

	// Proxy cho RENT mail: CHỈ dùng ProxyOverride, KHÔNG fallback ProxyStr.
	// "" = rent provider gọi API trực tiếp từ IP máy user (an toàn hơn cho provider).
	rentMailProxy := opts.ProxyOverride

	switch opts.Provider {

	// ── Temp mail ────────────────────────────────────────────────────────────

	case "moakt", "0":
		mk := temp.NewMoakt(opts.TempMailDomain, effectiveProxy)
		if opts.CustomUsername != "" {
			mk.SetCustomUsername(opts.CustomUsername)
		}
		return mk, nil

	case "mail1sec", "1", "@i2b.vn":
		m1s := temp.NewMail1sec(opts.TempMailDomain, mail1secProxy)
		if opts.OnStatus != nil {
			m1s.SetOnStatus(opts.OnStatus)
		}
		if opts.CustomUsername != "" {
			m1s.SetCustomUsername(opts.CustomUsername)
		}
		return m1s, nil

	case "tempmail-lol":
		// Ưu tiên provider-specific field, fallback về generic TempMailToken (user nhập tay).
		token := opts.TempMailLolApiKey
		if token == "" {
			token = opts.TempMailToken
		}
		return temp.NewTempMailLol(token, effectiveProxy), nil

	case "mohmal":
		return temp.NewMohmal(effectiveProxy), nil

	case "temporary-mail.net":
		return temp.NewTemporaryMailNet(effectiveProxy), nil

	case "mailtm":
		return temp.NewMailTm(effectiveProxy), nil

	case "tempmail-plus":
		return temp.NewTempMailPlus(opts.TempMailDomain, effectiveProxy), nil

	case "dropmail":
		return nil, fmt.Errorf("email: dropmail — API bị disable (legacy_token_disabled), provider không khả dụng")

	case "guerrillamail":
		return temp.NewGuerrillaMail(effectiveProxy), nil

	case "spam4me":
		return temp.NewSpam4Me(effectiveProxy), nil

	case "tempmailor", "temp-mail.org":
		return nil, fmt.Errorf("email: tempmailor — temp-mail.org bị Cloudflare block, provider không khả dụng")

	case "mailcx", "mail.cx":
		// API mới: api.mail.cx/v1/ + x-api-token (tm_live_...) từ mail.cx/dashboard
		apiToken := opts.TempMailToken
		if apiToken == "" {
			return nil, fmt.Errorf("email: mailcx — cần API token tm_live_... từ mail.cx/dashboard")
		}
		return temp.NewMailCx(apiToken, effectiveProxy), nil

	case "mailtd", "mail.td":
		// mail.td — PoW SHA-256, không cần key, không Cloudflare
		return temp.NewMailTd(opts.TempMailDomain, effectiveProxy), nil

	case "inboxes":
		return temp.NewInboxes(effectiveProxy), nil

	case "dismail":
		return temp.NewDismail(effectiveProxy), nil

	case "mailymg":
		return temp.NewMailymg(effectiveProxy), nil

	case "altmails":
		return temp.NewAltMails(effectiveProxy), nil

	case "onesecmail", "1secmail.com":
		return temp.NewOneSecMail(effectiveProxy), nil

	case "firetempmail":
		return temp.NewFireTempMail(effectiveProxy), nil

	// ── Temp mail port từ NullCoreSummer (đã test tạo mail OK) ────────────────
	case "tempmailio", "temp-mail.io":
		return temp.NewTempMailIo(effectiveProxy, temp.ParseTempMailIoDomains(opts.TempMailDomain)), nil
	case "anonymmail", "anonymmail.net":
		return temp.NewAnonymMail(effectiveProxy, temp.ParseAnonymMailDomains(opts.TempMailDomain)), nil
	case "tempmailnow", "tempmail.now":
		return temp.NewTempMailNow(effectiveProxy), nil
	case "tempmailworld", "tempmail.world":
		return temp.NewTempMailWorld(effectiveProxy), nil
	case "expressmail", "expressmail.app":
		return temp.NewExpressMail(effectiveProxy), nil
	case "tempmail100free":
		return temp.NewTempMail100Free(effectiveProxy), nil
	case "fakelegal", "fake.legal":
		return temp.NewFakeLegal(effectiveProxy, temp.ParseFakeLegalDomains(opts.TempMailDomain)), nil
	case "tempmailbee":
		return temp.NewTempMailBee(effectiveProxy, temp.ParseTempMailBeeDomains(opts.TempMailDomain)), nil
	case "tempmailapp", "temp-mail.app":
		return temp.NewTempMailApp(effectiveProxy), nil
	case "tempamail", "tempamail.com":
		return temp.NewTempAmail(effectiveProxy), nil
	case "tempmailai", "temp-mail.ai":
		return temp.NewTempMailAI(effectiveProxy), nil
	case "tempemailcc", "tempemail.cc":
		return temp.NewTempEmailCC(effectiveProxy), nil
	case "tempmailerme", "temp-mailer.me":
		return temp.NewTempMailerMe(effectiveProxy), nil
	case "mailwave", "mailwave.dev":
		return temp.NewMailWave(effectiveProxy), nil
	case "tempmail10", "tempmail10.com":
		return temp.NewTempMail10(effectiveProxy), nil
	case "tempmailpro", "tempmailpro.io":
		return temp.NewTempMailPro(effectiveProxy), nil
	case "tempmaildigital", "tempmail.digital":
		return temp.NewTempMailDigital(effectiveProxy), nil
	case "tempmailx", "tempmailx.xyz":
		return temp.NewTempMailX(effectiveProxy), nil
	case "tempmailid", "temp-mail.id":
		return temp.NewTempMailId(effectiveProxy, temp.ParseTempMailIdDomains(opts.TempMailDomain)), nil

	case "firetempmail-ctm": // @ctm.edu.pl — Polish edu domain, test iOS validation
		return temp.NewFireTempMailWithDomain(effectiveProxy, "ctm.edu.pl"), nil

	case "firetempmail-jd": // @jobsdeforyou.sa.com — alternate firetempmail domain
		return temp.NewFireTempMailWithDomain(effectiveProxy, "jobsdeforyou.sa.com"), nil

	case "firetempmail-offre": // @offredaily.sa.com — another firetempmail domain, test iOS
		return temp.NewFireTempMailWithDomain(effectiveProxy, "offredaily.sa.com"), nil

	case "fviainboxes":
		return temp.NewFviaInboxes(effectiveProxy), nil

	case "byomde", "byom.de":
		return temp.NewByomDe(effectiveProxy), nil

	case "dinlaan":
		return temp.NewDinlaan(effectiveProxy), nil

	case "cryptogmail":
		return temp.NewCryptoGmail(effectiveProxy), nil

	case "buslink24":
		return temp.NewBuslink24(effectiveProxy), nil

	case "boxmailstore", "boxmail.store":
		return temp.NewBoxMailStore(effectiveProxy), nil

	case "mailermnx":
		return temp.NewMailerMnx(effectiveProxy), nil

	case "tempforward":
		return temp.NewTempForward(effectiveProxy), nil

	case "tempomintraccoon":
		return temp.NewTempoMintraccoon(effectiveProxy), nil

	case "tempemail", "tempemail.co":
		return temp.NewTempEmailCo(effectiveProxy), nil

	case "tmpinbox":
		return temp.NewTmpInbox(effectiveProxy), nil

	case "tenminutemail", "10minutemail":
		return temp.NewTenMinuteMail(effectiveProxy), nil

	case "tempmailto":
		return temp.NewTempMailTo(effectiveProxy), nil

	case "1secemail", "onesecemail":
		return temp.NewOneSecEmail(effectiveProxy), nil

	case "tempmail100":
		return temp.NewTempMail100(effectiveProxy), nil

	case "tempmailso", "tempmail.so":
		return temp.NewTempMailSo(effectiveProxy), nil

	case "priyoemail", "priyo":
		// Fallback: provider-specific field → generic TempMailToken.
		token := opts.PriyoEmailApiKey
		if token == "" {
			token = opts.TempMailToken
		}
		return temp.NewPriyoEmail(token, effectiveProxy), nil

	case "tempmailorgpremium":
		return temp.NewTempMailOrgPremium(effectiveProxy), nil

	case "mailtempcom", "mail-temp.com":
		return temp.NewMailTempCom(effectiveProxy), nil

	// ── Rent mail ────────────────────────────────────────────────────────────

	case "zeus-x":
		accountCode := opts.ZeusXAccountCode
		if accountCode == "" {
			accountCode = "HOTMAIL"
		}
		zx := rent.NewZeusX(opts.ZeusXApiKey, accountCode, rentMailProxy)
		if opts.OnStatus != nil {
			zx.SetOnStatus(opts.OnStatus)
		}
		if opts.Pool != nil {
			zx.SetPool(opts.Pool)
		}
		zx.SetOTPPriority(opts.OTPHotmailPriority)
		return zx, nil

	case "dongvanfb":
		accType := opts.DvfbAccountType
		if accType == "" {
			accType = "1"
		}
		dvfb := rent.NewDongVanFB(opts.DvfbApiKey, accType, rentMailProxy)
		if opts.OnStatus != nil {
			dvfb.SetOnStatus(opts.OnStatus)
		}
		if opts.Pool != nil {
			dvfb.SetPool(opts.Pool)
		}
		dvfb.SetOTPPriority(opts.OTPHotmailPriority)
		return dvfb, nil

	case "store1s":
		productID := opts.Store1sProductID
		if productID == "" {
			productID = "40559"
		}
		s1s := rent.NewStore1s(opts.Store1sApiKey, productID, rentMailProxy)
		if opts.OnStatus != nil {
			s1s.SetOnStatus(opts.OnStatus)
		}
		if opts.Pool != nil {
			s1s.SetPool(opts.Pool)
		}
		s1s.SetOTPPriority(opts.OTPHotmailPriority)
		return s1s, nil

	case "mail30s":
		slug := opts.Mail30sProductSlug
		if slug == "" {
			slug = "hotmail-oauth2"
		}
		m30s := rent.NewMail30s(opts.Mail30sApiKey, slug, rentMailProxy)
		if opts.OnStatus != nil {
			m30s.SetOnStatus(opts.OnStatus)
		}
		if opts.Pool != nil {
			m30s.SetPool(opts.Pool)
		}
		m30s.SetOTPPriority(opts.OTPHotmailPriority)
		return m30s, nil

	case "muamail":
		mm := rent.NewMuaMail(opts.MuaMailApiKey, opts.MuaMailProductID, rentMailProxy)
		if opts.OnStatus != nil {
			mm.SetOnStatus(opts.OnStatus)
		}
		mm.SetOTPPriority(opts.OTPHotmailPriority)
		return mm, nil

	case "unlimitmail":
		um := rent.NewUnlimitMail(opts.UnlimitMailApiKey, opts.UnlimitMailProductID, rentMailProxy)
		if opts.OnStatus != nil {
			um.SetOnStatus(opts.OnStatus)
		}
		um.SetOTPPriority(opts.OTPHotmailPriority)
		return um, nil

	case "sptmail":
		sm := rent.NewSPTMail(opts.SptMailApiKey, opts.SptMailServiceCode, "")
		if opts.OnStatus != nil {
			sm.SetOnStatus(opts.OnStatus)
		}
		return sm, nil

	case "emailapiinfo", "gmail500":
		ea := rent.NewEmailAPIInfo(opts.EmailAPIInfoApiKey, opts.EmailAPIInfoProductCode, "")
		if opts.OnStatus != nil {
			ea.SetOnStatus(opts.OnStatus)
		}
		return ea, nil

	case "otpcheap":
		oc := rent.NewOtpCheap(opts.OtpCheapApiKey, opts.OtpCheapServiceID, "")
		if opts.OnStatus != nil {
			oc.SetOnStatus(opts.OnStatus)
		}
		return oc, nil

	case "shopgmail9999":
		sg := rent.NewShopGmail9999(opts.ShopGmail9999ApiKey, opts.ShopGmail9999Service, "")
		if opts.OnStatus != nil {
			sg.SetOnStatus(opts.OnStatus)
		}
		return sg, nil

	case "rentgmail":
		rg := rent.NewRentGmail(opts.RentGmailApiKey, opts.RentGmailPlatform, "")
		if opts.OnStatus != nil {
			rg.SetOnStatus(opts.OnStatus)
		}
		return rg, nil

	case "otpcodesms":
		ocs := rent.NewOtpCodesSms(opts.OtpCodesSmsApiKey, opts.OtpCodesSmsServiceID, "")
		if opts.OnStatus != nil {
			ocs.SetOnStatus(opts.OnStatus)
		}
		return ocs, nil

	case "wmemail":
		wm := rent.NewWmemail(opts.WmemailApiKey, opts.WmemailCommodity, rentMailProxy)
		if opts.OnStatus != nil {
			wm.SetOnStatus(opts.OnStatus)
		}
		wm.SetOTPPriority(opts.OTPHotmailPriority)
		return wm, nil

	case "wemakemail":
		// API key từ TempMailToken; domain list từ TempMailDomain (comma-separated).
		return temp.NewWeMakeMail(opts.TempMailToken, effectiveProxy,
			temp.ParseWeMakeMailDomains(opts.TempMailDomain)), nil

	case "i2b", "mail.i2b.vn":
		return temp.NewI2bMail(effectiveProxy), nil

	case "vietxf":
		apiKey := opts.VietXFApiKey
		if apiKey == "" {
			apiKey = opts.TempMailToken
		}
		return temp.NewVietXF(apiKey, effectiveProxy,
			temp.ParseVietXFDomains(opts.TempMailDomain)), nil

	case "mailhv":
		// API key từ MailHVApiKey; fallback về TempMailToken.
		// Domain list từ TempMailDomain (comma-separated); rỗng = random tất cả domain account.
		apiKey := opts.MailHVApiKey
		if apiKey == "" {
			apiKey = opts.TempMailToken
		}
		return temp.NewMailHV(apiKey, effectiveProxy,
			temp.ParseMailHVDomains(opts.TempMailDomain)), nil

	default:
		return nil, fmt.Errorf("email: unknown provider %q", opts.Provider)
	}
}
