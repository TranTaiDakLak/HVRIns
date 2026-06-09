// settings.types.ts — Kiểu dữ liệu cho cài đặt chung
// Mapping đầy đủ từ WeBM C# frmSettings + frmFacebook.Proxy + WemakeSocial

// Nền tảng đăng nhập
export type LoginPlatform = 'facebook' | 'instagram' | 'bm'

// Nhà cung cấp captcha
export type CaptchaProvider = '2captcha' | 'omocaptcha' | 'ezcaptcha' | 'capsolver'

// Nhà cung cấp IP/Proxy — đầy đủ 13 loại theo WeBM cboChangeIP (0-12)
export type IpProvider =
  | 'none'            // 0 - Không đổi IP
  | 'fpt'             // 1 - FPT
  | 'xproxy'          // 2 - Xproxy / MobiProxy
  | 'hma'             // 3 - HMA VPN
  | 'proxy'           // 4 - Proxy (static list)
  | 'proxy_fixed'     // 5 - Proxy cố định (gắn theo account)
  | 'tinsoft'         // 6 - Tinsoft (WWProxy)
  | 'shoplike'        // 7 - ShopLike
  | 'netproxy'        // 8 - NetProxy
  | 'minproxy'        // 9 - MinProxy
  | 'netproxy_gb'     // 10 - NetProxy dung lượng (Gigabytes)
  | 'proxy_popular'   // 11 - Proxy Dân Cư
  | 'proxy_farm'      // 12 - Proxy Farm

// Kiểu chạy Xproxy
export type XproxyRunType = 'shared' | 'exclusive'

// Nguồn tài khoản chạy
export type AccountSource = 'folder' | 'file' | 'api'

// Cấu hình chung
export interface GeneralConfig {
  // CHUNG
  threadRequest: number
  delayRequest: number
  delayThread: number

  // KIỂM TRA IP API
  apiCheckIp: number

  // KIỂM TRA THÔNG TIN
  threadCheckInfo: number

  // DẠNG ĐĂNG NHẬP
  loginPlatform: LoginPlatform
  loginMethod: number

  // NGUỒN TÀI KHOẢN
  accountSource: AccountSource
  accountSourcePath: string

  // CloneHV API credentials (chỉ dùng khi accountSource === 'api')
  cloneHvUsername: string
  cloneHvPassword: string
  cloneHvProductId: string
  cloneHvAmount: number

  // KHÁC
  saveRunColumn: boolean
  backupDB: boolean
  closeAfterDone: boolean

  // Captcha
  captchaProvider: CaptchaProvider
  captchaKeys: Record<CaptchaProvider, string>

  // ĐỔI IP
  ipProvider: IpProvider
  checkIpBeforeRun: boolean
  delayChangeIp: number

  // LOCALE & DEVICE FAKE
  localeFake: 'random' | 'match-ip'
  deepFakeInApi: boolean

  // COOKIE KHỞI TẠO
  cookieUse: boolean
  cookieLimit: boolean
  cookieLimitCount: number
  cookieMode: 'in-file' | 'new' | 'ck'

  // UA TÙY CHỈNH
  uaAddSpecs: boolean
  uaBuildFile: boolean
  uaCustomType: number   // 1 | 2 | 3

  // SIM NETWORK
  simNetworkMode: 'random' | 'match-ip'
  simNetworkType: string  // 'LTE' | '3G' | '4G' | 'WIFI'

}

// Cấu hình IP theo từng nhà cung cấp
export interface IpConfig {
  // Proxy thường (case 4)
  proxyList: string       // proxy standard: host:port:user:pass
  proxyStickyList: string // proxy sticky/rotating: host:port:user_area-XX_session-xxx_life-N:pass
  proxyActiveTab: 'standard' | 'sticky' // tab đang chọn → xác định list nào được dùng khi chạy
  proxyType: 'http' | 'socks5'

  // FPT (case 1)
  fptKeys: string

  // XProxy (case 2)
  xproxyServiceUrl: string
  xproxyType: 'http' | 'socks5'
  xproxyList: string
  xproxyThreadPerIp: number
  xproxyRunType: XproxyRunType

  // Tinsoft (case 6)
  tinsoftKeys: string
  tinsoftThreadPerIp: number

  // ShopLike (case 7)
  shoplikeKeys: string
  shoplikeThreadPerIp: number

  // NetProxy (case 8)
  netproxyKeys: string
  netproxyThreadPerIp: number

  // MinProxy (case 9)
  minproxyKeys: string
  minproxyThreadPerIp: number

  // NetProxy Gigabytes (case 10)
  netproxyGbKey: string

  // Proxy Dân Cư / Popular (case 11)
  proxyPopularKeys: string
  proxyPopularThreadPerIp: number
  proxyPopularAccessToken: string

  // Proxy Farm (case 12)
  proxyFarmKeys: string
  proxyFarmThreadPerIp: number
  proxyFarmAccessToken: string

  // Proxy riêng cho Reg (tách khỏi Verify)
  useVerifyProxyForReg: boolean       // true = reg dùng chung pool với verify
  regIpProvider: IpProvider           // provider riêng khi useVerifyProxyForReg=false
  regProxyList: string
  regProxyStickyList: string
  regProxyActiveTab: 'standard' | 'sticky'
  regProxyType: 'http' | 'socks5'

  // Retry & Delay khi proxy lỗi
  proxyRetry: number       // số lần retry khi gặp lỗi proxy (0 = không retry)
  proxyDelayMs: number     // delay (ms) trước khi đổi proxy sang cái tiếp theo
}

// ============================================================
// Dạng đăng nhập theo từng platform
// Mapping chính xác từ WeBM WemakeSocial.PerformLogin()
// ============================================================

export interface LoginMethodOption {
  value: number
  label: string
  disabled?: boolean
  description?: string
}

// Facebook: 7 dạng (index 0-6) — từ frmSettings.Designer.cs cboLoginWith.Items
export const LOGIN_METHODS_FACEBOOK: LoginMethodOption[] = [
  { value: 6, label: 'Cookie mobile', description: 'Đăng nhập bằng Cookie mobile' },
]

// Instagram: 4 dạng (index 0-3) — từ WemakeSocial.PerformLogin() rdoLoginInstagram
export const LOGIN_METHODS_INSTAGRAM: LoginMethodOption[] = [
  { value: 0, label: 'Tài khoản WWW', description: 'Đăng nhập bằng User/Pass trên web' },
  { value: 1, label: 'Cookie → Tài khoản WWW', description: 'Ưu tiên Cookie, fallback User/Pass' },
  { value: 2, label: 'Cookie → Android', description: 'Ưu tiên Cookie, fallback Android API' },
  { value: 3, label: 'Android', description: 'Đăng nhập qua Android API' },
]

// BM: chỉ 1 dạng — từ WemakeSocial.PerformLogin() rdoLoginBM (chỉ dùng cookie)
export const LOGIN_METHODS_BM: LoginMethodOption[] = [
  { value: 0, label: 'Cookie BM', description: 'Đăng nhập bằng Cookie BM (bắt buộc có cookie)' },
]

// Helper: lấy danh sách login methods theo platform
export function getLoginMethodsByPlatform(platform: LoginPlatform): LoginMethodOption[] {
  switch (platform) {
    case 'facebook': return LOGIN_METHODS_FACEBOOK
    case 'instagram': return LOGIN_METHODS_INSTAGRAM
    case 'bm': return LOGIN_METHODS_BM
  }
}

// Legacy export cho backward compat
export const LOGIN_METHODS = LOGIN_METHODS_FACEBOOK.map(m => m.label)

// Danh sách captcha providers
export const CAPTCHA_PROVIDERS: { value: CaptchaProvider; label: string }[] = [
  { value: '2captcha', label: '2Captcha' },
  { value: 'omocaptcha', label: 'OmoCaptcha' },
  { value: 'ezcaptcha', label: 'EZCaptcha' },
  { value: 'capsolver', label: 'CapSolver' },
]

// Danh sách API kiểm tra IP — backend hiện dùng chain tự động (ip-api → amazonaws → adspower → ipify).
// Option 0 = default "Auto" chạy full chain, cho country code chuẩn nhất.
// Các option khác giữ để backward compat + fallback browser-side nếu backend không khả dụng.
export const API_CHECK_IP_PROVIDERS: { value: number; label: string }[] = [
  { value: 0, label: 'Auto — IP-API.com (Khuyên dùng)' },
  { value: 1, label: 'AdsPower' },
  { value: 2, label: 'Luna Proxy' },
  { value: 3, label: 'IpInfo.io' },
  { value: 4, label: 'NordVPN' },
]

// Danh sách IP providers — tất cả provider có cấu hình
export const IP_PROVIDERS: { value: IpProvider; label: string }[] = [
  { value: 'none', label: 'Không đổi IP' },
  { value: 'proxy', label: 'Proxy danh sách' },
  { value: 'proxy_fixed', label: 'Proxy cố định (theo account)' },
  { value: 'fpt', label: 'FPT' },
  { value: 'xproxy', label: 'XProxy / MobiProxy' },
  { value: 'tinsoft', label: 'Tinsoft (WWProxy)' },
  { value: 'shoplike', label: 'ShopLike' },
  { value: 'netproxy', label: 'NetProxy' },
  { value: 'minproxy', label: 'MinProxy' },
  { value: 'netproxy_gb', label: 'NetProxy dung lượng (GB)' },
  { value: 'proxy_popular', label: 'Proxy Dân Cư' },
  { value: 'proxy_farm', label: 'Proxy Farm' },
]

// Giá trị mặc định
export const DEFAULT_GENERAL_CONFIG: GeneralConfig = {
  threadRequest: 20,
  delayRequest: 1000,
  delayThread: 0,
  apiCheckIp: 0,
  threadCheckInfo: 10,
  loginPlatform: 'instagram',
  loginMethod: 0,
  accountSource: 'folder',
  accountSourcePath: '',
  cloneHvUsername: '',
  cloneHvPassword: '',
  cloneHvProductId: '',
  cloneHvAmount: 10,
  saveRunColumn: false,
  backupDB: false,
  closeAfterDone: false,
  captchaProvider: '2captcha',
  captchaKeys: { '2captcha': '', omocaptcha: '', ezcaptcha: '', capsolver: '' },
  ipProvider: 'none',
  checkIpBeforeRun: false,
  delayChangeIp: 3,
  localeFake: 'match-ip', // default: locale theo IP proxy — realistic
  deepFakeInApi: true, // default ON — deep locale trong API payload (C# trust pattern)
  cookieUse: false,
  cookieLimit: false,
  cookieLimitCount: 5,
  cookieMode: 'new',
  uaAddSpecs: true,
  uaBuildFile: false,
  uaCustomType: 1,
  simNetworkMode: 'match-ip', // default: sim network theo IP — realistic
  simNetworkType: 'LTE',
}

export const DEFAULT_IP_CONFIG: IpConfig = {
  proxyList: '',
  proxyStickyList: '',
  proxyActiveTab: 'standard',
  proxyType: 'http',
  fptKeys: '',
  xproxyServiceUrl: '',
  xproxyType: 'http',
  xproxyList: '',
  xproxyThreadPerIp: 1,
  xproxyRunType: 'shared',
  tinsoftKeys: '',
  tinsoftThreadPerIp: 1,
  shoplikeKeys: '',
  shoplikeThreadPerIp: 1,
  netproxyKeys: '',
  netproxyThreadPerIp: 1,
  minproxyKeys: '',
  minproxyThreadPerIp: 1,
  netproxyGbKey: '',
  proxyPopularKeys: '',
  proxyPopularThreadPerIp: 1,
  proxyPopularAccessToken: '',
  proxyFarmKeys: '',
  proxyFarmThreadPerIp: 1,
  proxyFarmAccessToken: '',
  useVerifyProxyForReg: true,
  regIpProvider: 'none',
  regProxyList: '',
  regProxyStickyList: '',
  regProxyActiveTab: 'standard',
  regProxyType: 'http',
  proxyRetry: 3,
  proxyDelayMs: 0,
}
