// report.go — MappingReport: phân loại kết quả chuyển đổi legacy → new model
package adapter

import (
	"fmt"
	"strings"
)

// MappingStatus phân loại kết quả map của một field
type MappingStatus string

const (
	StatusOK          MappingStatus = "ok"          // mapped tự tin
	StatusConfirm     MappingStatus = "confirm"      // cần xác nhận (path có thể không còn hợp lệ)
	StatusSensitive   MappingStatus = "sensitive"    // field nhạy cảm — được import nhưng cần xác nhận
	StatusUnsupported MappingStatus = "unsupported"  // không hỗ trợ hoặc deprecated
)

// MappedField một field được phân tích từ legacy → new path
type MappedField struct {
	LegacyKey    string        `json:"legacyKey"`
	NewPath      string        `json:"newPath"`
	DisplayValue string        `json:"displayValue"` // giá trị hiển thị (masked nếu sensitive)
	Status       MappingStatus `json:"status"`
	Note         string        `json:"note"`
}

// MappingReport kết quả phân tích mapping từ legacy config
type MappingReport struct {
	MappedOk     []MappedField `json:"mappedOk"`
	NeedsConfirm []MappedField `json:"needsConfirm"`
	Sensitive    []MappedField `json:"sensitive"`
	Unsupported  []MappedField `json:"unsupported"`
	ParseErrors  []string      `json:"parseErrors"`
}

// BuildMappingReport phân tích LegacySettingsData + LegacyInteractionConfig
// và trả về MappingReport mô tả từng field sẽ được xử lý như thế nào.
// Không thực hiện lưu — chỉ phân tích.
func BuildMappingReport(s LegacySettingsData, ic LegacyInteractionConfig) MappingReport {
	r := MappingReport{}
	g := s.General
	ip := s.Ip

	addOK := func(legacyKey, newPath, value string) {
		r.MappedOk = append(r.MappedOk, MappedField{legacyKey, newPath, value, StatusOK, ""})
	}
	addConfirm := func(legacyKey, newPath, value, note string) {
		r.NeedsConfirm = append(r.NeedsConfirm, MappedField{legacyKey, newPath, value, StatusConfirm, note})
	}
	addSensitive := func(legacyKey, newPath, note string) {
		r.Sensitive = append(r.Sensitive, MappedField{legacyKey, newPath, "***", StatusSensitive, note})
	}
	addUnsupported := func(legacyKey, value, note string) {
		r.Unsupported = append(r.Unsupported, MappedField{legacyKey, "", value, StatusUnsupported, note})
	}

	// ── Runtime ───────────────────────────────────────────────────────────────
	addOK("threadRequest", "profile.runtime.threadRequest", fmt.Sprintf("%d", g.ThreadRequest))
	addOK("delayRequest", "profile.runtime.delayRequest", fmt.Sprintf("%d ms", g.DelayRequest))
	addOK("threadCheckInfo", "profile.runtime.threadCheckInfo", fmt.Sprintf("%d", g.ThreadCheckInfo))
	addOK("checkIpBeforeRun", "profile.runtime.checkIpBeforeRun", fmt.Sprintf("%v", g.CheckIpBeforeRun))
	addOK("delayChangeIp", "profile.runtime.delayChangeIp", fmt.Sprintf("%d s", g.DelayChangeIp))

	// ── Global ────────────────────────────────────────────────────────────────
	addOK("loginPlatform", "global.loginPlatform", orDef(g.LoginPlatform, "facebook"))
	addOK("loginMethod", "global.loginMethod", fmt.Sprintf("%d", g.LoginMethod))
	addOK("saveRunColumn", "global.saveRunColumn", fmt.Sprintf("%v", g.SaveRunColumn))
	addOK("backupDB", "global.backupDB", fmt.Sprintf("%v", g.BackupDB))
	addOK("closeAfterDone", "global.closeAfterDone", fmt.Sprintf("%v", g.CloseAfterDone))

	// ── Account source ────────────────────────────────────────────────────────
	addOK("accountSource", "profile.account.source", orDef(g.AccountSource, "folder"))
	addOK("cloneHvEnabled", "profile.account.cloneHv.enabled", fmt.Sprintf("%v", g.AccountSource == "api"))
	if g.CloneHVUsername != "" {
		addOK("cloneHvUsername", "profile.account.cloneHv.username", g.CloneHVUsername)
	}
	if g.CloneHVProductID != "" {
		addOK("cloneHvProductId", "profile.account.cloneHv.productId", g.CloneHVProductID)
	}
	if g.CloneHVAmount > 0 {
		addOK("cloneHvAmount", "profile.account.cloneHv.amount", fmt.Sprintf("%d", g.CloneHVAmount))
	}

	// Folder path — needs confirmation (path may not exist)
	if g.AccountSourcePath != "" {
		addConfirm("accountSourcePath", "profile.account.folderPath", g.AccountSourcePath,
			"Kiểm tra thư mục còn tồn tại trên máy hiện tại")
	}

	// ── Proxy ─────────────────────────────────────────────────────────────────
	addOK("ipProvider", "profile.proxy.provider", orDef(g.IpProvider, "none"))
	if ip.ProxyType != "" {
		addOK("proxyType", "profile.proxy.proxyType", ip.ProxyType)
	}
	if ip.ProxyList != "" {
		lines := countLines(ip.ProxyList)
		addOK("proxyList", "profile.proxy.proxyList", fmt.Sprintf("%d proxy entries", lines))
	}

	// Provider-specific keys (mapped OK if present)
	type proxyKV struct{ key, path, val string }
	proxyEntries := []proxyKV{
		{"fptKeys", "profile.proxy.providers.fpt.keys", ip.FptKeys},
		{"xproxyServiceUrl", "profile.proxy.providers.xproxy.serviceUrl", ip.XproxyServiceUrl},
		{"tinsoftKeys", "profile.proxy.providers.tinsoft.keys", ip.TinsoftKeys},
		{"shoplikeKeys", "profile.proxy.providers.shoplike.keys", ip.ShoplikeKeys},
		{"netproxyKeys", "profile.proxy.providers.netproxy.keys", ip.NetproxyKeys},
		{"minproxyKeys", "profile.proxy.providers.minproxy.keys", ip.MinproxyKeys},
		{"netproxyGbKey", "profile.proxy.providers.netproxygb.keys", ip.NetproxyGbKey},
		{"proxyPopularKeys", "profile.proxy.providers.proxypopular.keys", ip.ProxyPopularKeys},
		{"proxyFarmKeys", "profile.proxy.providers.proxyfarm.keys", ip.ProxyFarmKeys},
	}
	for _, e := range proxyEntries {
		if e.val != "" {
			addOK(e.key, e.path, truncate(e.val, 40))
		}
	}

	// ── Verify ────────────────────────────────────────────────────────────────
	addOK("verifyEnabled", "profile.verify.enabled", fmt.Sprintf("%v", ic.VerifyEnabled))
	addOK("checkLiveDieEnabled", "profile.verify.checkLiveDie", fmt.Sprintf("%v", ic.CheckLiveDieEnabled))
	addOK("timeDelayCheck", "profile.verify.timeDelayCheck", fmt.Sprintf("%d s", ic.TimeDelayCheck))
	addOK("timeDelaySendCode", "profile.verify.timeDelaySendCode", fmt.Sprintf("%d s", ic.TimeDelaySendCode))
	addOK("sendAgainCode", "profile.verify.sendAgainCode", fmt.Sprintf("%v", ic.SendAgainCode))

	// Output paths — needs confirmation
	if ic.OutputPath != "" {
		addConfirm("outputPath", "profile.output.verifyPath", ic.OutputPath,
			"Kiểm tra thư mục lưu kết quả còn tồn tại")
	}
	if ic.CreateOutputPath != "" {
		addConfirm("createOutputPath", "profile.register.outputPath", ic.CreateOutputPath,
			"Kiểm tra thư mục lưu tài khoản tạo còn tồn tại")
	}

	// ── Mail ─────────────────────────────────────────────────────────────────
	knownProviders := map[string]bool{
		"@tmpbox.net": true, "@i2b.vn": true, "mohmal": true,
		"zeus-x": true, "dongvanfb": true, "store1s": true, "mail30s": true, "": true,
	}
	if ic.MailProvider != "" && !knownProviders[ic.MailProvider] {
		addConfirm("mailProvider", "profile.mail.provider", ic.MailProvider,
			"Provider không nằm trong danh sách hỗ trợ — sẽ được lưu nhưng cần xác nhận")
	} else if ic.MailProvider != "" {
		addOK("mailProvider", "profile.mail.provider", ic.MailProvider)
	}
	if ic.MailList != "" {
		addOK("mailList", "profile.mail.mailList", fmt.Sprintf("%d mail addresses", countLines(ic.MailList)))
	}
	if ic.ZeusXAccountCode != "" {
		addOK("zeusXAccountCode", "profile.mail.providers.zeusx.accountCode", ic.ZeusXAccountCode)
	}
	if ic.DvfbAccountType != "" {
		addOK("dvfbAccountType", "profile.mail.providers.dvfb.accountType", ic.DvfbAccountType)
	}
	if ic.Store1sProductID != "" {
		addOK("store1sProductId", "profile.mail.providers.store1s.productId", ic.Store1sProductID)
	}
	if ic.Mail30sProductSlug != "" {
		addOK("mail30sProductSlug", "profile.mail.providers.mail30s.productSlug", ic.Mail30sProductSlug)
	}

	// ── Register ──────────────────────────────────────────────────────────────
	addOK("createEnabled", "profile.register.enabled", fmt.Sprintf("%v", ic.CreateEnabled))
	if ic.CreateType != "" {
		addOK("createType", "profile.register.type", ic.CreateType)
	}
	if ic.CreateCookieList != "" {
		addOK("createCookieList", "profile.register.cookieList",
			fmt.Sprintf("%d cookie entries", countLines(ic.CreateCookieList)))
	}

	// ── Device ────────────────────────────────────────────────────────────────
	if ic.UaIphoneList != "" {
		addOK("uaIphoneList", "profile.device.uaList",
			fmt.Sprintf("%d UA strings", countLines(ic.UaIphoneList)))
	}

	// ── Captcha ───────────────────────────────────────────────────────────────
	if g.CaptchaProvider != "" {
		addOK("captchaProvider", "profile.captcha.provider", g.CaptchaProvider)
	}

	// ── Sensitive fields ──────────────────────────────────────────────────────
	if g.CloneHVPassword != "" {
		addSensitive("cloneHvPassword", "profile.account.cloneHv.password",
			"Mật khẩu CloneHV — được import, nên đổi sau")
	}
	if ic.ZeusXApiKey != "" {
		addSensitive("zeusXApiKey", "profile.mail.providers.zeusx.apiKey", "API key ZeusX")
	}
	if ic.DvfbApiKey != "" {
		addSensitive("dvfbApiKey", "profile.mail.providers.dvfb.apiKey", "API key DongVanFB")
	}
	if ic.Store1sApiKey != "" {
		addSensitive("store1sApiKey", "profile.mail.providers.store1s.apiKey", "API key Store1s")
	}
	if ic.Mail30sApiKey != "" {
		addSensitive("mail30sApiKey", "profile.mail.providers.mail30s.apiKey", "API key Mail30s")
	}
	for prov, key := range g.CaptchaKeys {
		if key != "" {
			addSensitive("captchaKeys."+prov, "profile.captcha.keys."+prov,
				"API key "+prov+" — nên xác nhận còn hiệu lực")
		}
	}

	// ── Unsupported / deprecated ──────────────────────────────────────────────
	if g.IpProvider == "hma" {
		addUnsupported("ipProvider", "hma",
			"HMA VPN không còn hỗ trợ — chọn provider khác sau khi import")
	}
	if g.LoginPlatform == "instagram" || g.LoginPlatform == "bm" {
		addUnsupported("loginPlatform", g.LoginPlatform,
			"Platform này không active trong tool mới — sẽ được lưu nhưng UI chỉ hiện Facebook")
	}

	return r
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func orDef(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// truncate rút gọn chuỗi s để hiển thị trong report: lấy dòng đầu tiên, cắt ở max ký tự,
// thêm "(+N dòng)" nếu có nhiều dòng.
// s: chuỗi gốc (thường là multi-line config value như proxy list, UA list).
// max: số ký tự tối đa cho dòng đầu.
func truncate(s string, max int) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	first := strings.TrimSpace(lines[0])
	if len(first) > max {
		first = first[:max] + "..."
	}
	if len(lines) > 1 {
		return fmt.Sprintf("%s (+%d dòng)", first, len(lines)-1)
	}
	return first
}

// countLines đếm số dòng không rỗng trong chuỗi s.
// s: chuỗi multi-line (ví dụ proxy list, UA list).
// Trả về 0 nếu s rỗng hoặc chỉ có whitespace.
func countLines(s string) int {
	if strings.TrimSpace(s) == "" {
		return 0
	}
	return len(strings.Split(strings.TrimSpace(s), "\n"))
}
