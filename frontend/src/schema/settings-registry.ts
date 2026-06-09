// settings-registry.ts — TS-side schema registry mirroring Go internal/settings/schema/registry.go
// Purpose: single source of truth for field metadata used by UI components.
// When adding a new setting field:
//   1. Add FieldMeta entry here
//   2. Add FieldHelp entry in field-help.ts (UX text)
//   3. Add Go FieldMeta in internal/settings/schema/registry.go
//   4. Add type field in types/settings.types.ts or types/interaction.types.ts

export type FieldType = 'string' | 'int' | 'bool' | 'select' | 'password' | 'textarea' | 'keymap'

export type FieldDomain =
  | 'global'
  | 'runtime'
  | 'accountSource'
  | 'proxy'
  | 'verify'
  | 'register'
  | 'mail'
  | 'captcha'
  | 'output'
  | 'device'

export interface FieldMeta {
  /** Dotted path matching Go registry key (e.g. 'profile.runtime.threadRequest') */
  key: string
  /** Vietnamese label */
  label: string
  /** Domain group */
  domain: FieldDomain
  /** Input type */
  type: FieldType
  /** Required field */
  required?: boolean
  /** Sensitive (API keys, passwords) */
  sensitive?: boolean
  /** Min value for number fields */
  min?: number
  /** Max value for number fields */
  max?: number
  /** Options for select fields */
  options?: string[]
  /** Short description */
  description?: string
  /** TS form field key (for mapping to flat form models) */
  formKey?: string
}

// ── Registry ────────────────────────────────────────────────────────────────────
// Mirrors Go internal/settings/schema/registry.go
// Keep in sync when adding/removing fields.

export const SETTINGS_REGISTRY: FieldMeta[] = [
  // ── Global ──────────────────────────────────────────
  { key: 'global.loginPlatform', label: 'Nền tảng đăng nhập', domain: 'global', type: 'select', options: ['facebook', 'instagram', 'bm'], formKey: 'loginPlatform' },
  { key: 'global.loginMethod', label: 'Phương thức đăng nhập', domain: 'global', type: 'int', min: 0, formKey: 'loginMethod' },
  { key: 'global.saveRunColumn', label: 'Lưu cột lần chạy', domain: 'global', type: 'bool', formKey: 'saveRunColumn' },
  { key: 'global.backupDB', label: 'Sao lưu database', domain: 'global', type: 'bool', formKey: 'backupDB' },
  { key: 'global.closeAfterDone', label: 'Đóng app sau khi xong', domain: 'global', type: 'bool', formKey: 'closeAfterDone' },

  // ── Runtime ─────────────────────────────────────────
  { key: 'profile.runtime.threadRequest', label: 'Số luồng Request', domain: 'runtime', type: 'int', required: true, min: 1, max: 600, formKey: 'threadRequest' },
  { key: 'profile.runtime.delayRequest', label: 'Nghỉ giữa các request (ms)', domain: 'runtime', type: 'int', min: 0, formKey: 'delayRequest' },
  { key: 'profile.runtime.threadCheckInfo', label: 'Số luồng kiểm tra ẩn', domain: 'runtime', type: 'int', min: 1, max: 100, formKey: 'threadCheckInfo' },
  { key: 'profile.runtime.checkIpBeforeRun', label: 'Kiểm tra IP trước khi chạy', domain: 'runtime', type: 'bool', formKey: 'checkIpBeforeRun' },
  { key: 'profile.runtime.delayChangeIp', label: 'Delay đổi IP (giây)', domain: 'runtime', type: 'int', min: 0, formKey: 'delayChangeIp' },

  // ── Account Source ──────────────────────────────────
  { key: 'profile.account.source', label: 'Nguồn tài khoản', domain: 'accountSource', type: 'select', options: ['folder', 'api'], formKey: 'accountSource' },
  { key: 'profile.account.cloneHv.username', label: 'CloneHV Username', domain: 'accountSource', type: 'string', formKey: 'cloneHvUsername' },
  { key: 'profile.account.cloneHv.password', label: 'CloneHV Password', domain: 'accountSource', type: 'password', sensitive: true, formKey: 'cloneHvPassword' },
  { key: 'profile.account.cloneHv.productId', label: 'CloneHV Product ID', domain: 'accountSource', type: 'string', formKey: 'cloneHvProductId' },
  { key: 'profile.account.cloneHv.amount', label: 'Số lượng mua mỗi lần', domain: 'accountSource', type: 'int', min: 1, formKey: 'cloneHvAmount' },

  // ── Proxy ───────────────────────────────────────────
  { key: 'profile.proxy.provider', label: 'Nhà cung cấp IP', domain: 'proxy', type: 'select', options: ['none', 'proxy', 'proxy_fixed', 'fpt', 'xproxy', 'tinsoft', 'shoplike', 'netproxy', 'minproxy', 'netproxy_gb', 'proxy_popular', 'proxy_farm'], formKey: 'ipProvider' },
  { key: 'profile.proxy.proxyList', label: 'Danh sách Proxy', domain: 'proxy', type: 'textarea', formKey: 'proxyList' },
  { key: 'profile.proxy.proxyType', label: 'Loại Proxy', domain: 'proxy', type: 'select', options: ['http', 'https', 'socks5', 'socks4'], formKey: 'proxyType' },

  // ── Verify ──────────────────────────────────────────
  { key: 'profile.verify.enabled', label: 'Bật verify', domain: 'verify', type: 'bool', formKey: 'verifyEnabled' },
  { key: 'profile.verify.checkLiveDie', label: 'Kiểm tra live/die', domain: 'verify', type: 'bool', formKey: 'checkLiveDieEnabled' },
  { key: 'profile.verify.timeDelayCheck', label: 'Delay kiểm tra (s)', domain: 'verify', type: 'int', min: 1, formKey: 'timeDelayCheck' },
  { key: 'profile.verify.timeDelaySendCode', label: 'Delay gửi code (s)', domain: 'verify', type: 'int', min: 1, formKey: 'timeDelaySendCode' },
  { key: 'profile.verify.sendAgainCode', label: 'Gửi lại code', domain: 'verify', type: 'bool', formKey: 'sendAgainCode' },

  // ── Register ────────────────────────────────────────
  { key: 'profile.register.enabled', label: 'Bật tạo tài khoản', domain: 'register', type: 'bool', formKey: 'createEnabled' },
  { key: 'profile.register.type', label: 'Loại tài khoản tạo', domain: 'register', type: 'select', options: ['spam', 'tut'], formKey: 'createType' },
  { key: 'profile.register.cookieList', label: 'Danh sách Cookie', domain: 'register', type: 'textarea', formKey: 'createCookieList' },
  { key: 'profile.register.outputPath', label: 'Thư mục lưu tài khoản tạo', domain: 'register', type: 'string', formKey: 'createOutputPath' },

  // ── Mail ────────────────────────────────────────────
  { key: 'profile.mail.provider', label: 'Mail Provider', domain: 'mail', type: 'select', options: ['@tmpbox.net', '@i2b.vn', 'mohmal', 'zeus-x', 'dongvanfb', 'store1s', 'mail30s'], formKey: 'mailProvider' },
  { key: 'profile.mail.mailList', label: 'Danh sách mail', domain: 'mail', type: 'textarea', formKey: 'mailList' },
  { key: 'profile.mail.providers.zeusx.apiKey', label: 'ZeusX API Key', domain: 'mail', type: 'password', sensitive: true, formKey: 'zeusXApiKey' },
  { key: 'profile.mail.providers.dvfb.apiKey', label: 'DongVanFB API Key', domain: 'mail', type: 'password', sensitive: true, formKey: 'dvfbApiKey' },
  { key: 'profile.mail.providers.store1s.apiKey', label: 'Store1s API Key', domain: 'mail', type: 'password', sensitive: true, formKey: 'store1sApiKey' },
  { key: 'profile.mail.providers.mail30s.apiKey', label: 'Mail30s API Key', domain: 'mail', type: 'password', sensitive: true, formKey: 'mail30sApiKey' },

  // ── Captcha ─────────────────────────────────────────
  { key: 'profile.captcha.provider', label: 'Captcha Provider', domain: 'captcha', type: 'select', options: ['2captcha', 'omocaptcha', 'ezcaptcha', 'capsolver'], formKey: 'captchaProvider' },
  { key: 'profile.captcha.keys.2captcha', label: '2Captcha API Key', domain: 'captcha', type: 'password', sensitive: true },
  { key: 'profile.captcha.keys.omocaptcha', label: 'OmoCaptcha API Key', domain: 'captcha', type: 'password', sensitive: true },
  { key: 'profile.captcha.keys.ezcaptcha', label: 'EZCaptcha API Key', domain: 'captcha', type: 'password', sensitive: true },
  { key: 'profile.captcha.keys.capsolver', label: 'CapSolver API Key', domain: 'captcha', type: 'password', sensitive: true },

  // ── Output ──────────────────────────────────────────
  { key: 'profile.output.verifyPath', label: 'Thư mục output Verify', domain: 'output', type: 'string', formKey: 'outputPath' },
  { key: 'profile.output.registerPath', label: 'Thư mục output Register', domain: 'output', type: 'string', formKey: 'createOutputPath' },

  // ── Device ──────────────────────────────────────────
  { key: 'profile.device.uaList', label: 'Danh sách User-Agent iPhone', domain: 'device', type: 'textarea', formKey: 'uaIphoneList' },
]

// ── Helpers ─────────────────────────────────────────────────────────────────────

/** Get field meta by registry key */
export function getFieldMeta(key: string): FieldMeta | undefined {
  return SETTINGS_REGISTRY.find(f => f.key === key)
}

/** Get field meta by form key (TS model field name) */
export function getFieldMetaByFormKey(formKey: string): FieldMeta | undefined {
  return SETTINGS_REGISTRY.find(f => f.formKey === formKey)
}

/** Get all fields in a domain */
export function getFieldsByDomain(domain: FieldDomain): FieldMeta[] {
  return SETTINGS_REGISTRY.filter(f => f.domain === domain)
}

/** Get all sensitive fields */
export function getSensitiveFields(): FieldMeta[] {
  return SETTINGS_REGISTRY.filter(f => f.sensitive)
}

/** Validate a numeric field value against its registry constraints */
export function validateFieldValue(formKey: string, value: unknown): string | null {
  const meta = getFieldMetaByFormKey(formKey)
  if (!meta) return null

  if (meta.required && (value === undefined || value === null || value === '')) {
    return `${meta.label} là bắt buộc`
  }

  if (meta.type === 'int' && typeof value === 'number') {
    if (meta.min !== undefined && value < meta.min) {
      return `${meta.label} tối thiểu là ${meta.min}`
    }
    if (meta.max !== undefined && value > meta.max) {
      return `${meta.label} tối đa là ${meta.max}`
    }
  }

  if (meta.type === 'select' && meta.options && typeof value === 'string') {
    if (!meta.options.includes(value)) {
      return `${meta.label}: giá trị "${value}" không hợp lệ`
    }
  }

  return null
}
