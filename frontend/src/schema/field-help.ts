// field-help.ts — Registry help text cho mọi setting field.
// Mirrors Go internal/settings/schema/registry.go nhưng tập trung vào UX (hint, detail, example).
// Dùng trong FieldHelp.vue và tooltip components.
//
// Pattern để thêm field mới:
//   1. Thêm entry với key = legacyKey hoặc model path (e.g. 'threadRequest')
//   2. Điền hint (1 dòng), detail (có thể null), example (tùy chọn)
//   3. Dùng trong template: <FieldHelp field="threadRequest" />

export interface FieldHelp {
  /** Label ngắn gọn (title case) */
  label: string
  /** Tooltip ngắn — hiển thị khi hover (≤ 80 ký tự) */
  hint: string
  /** Giải thích dài hơn — hiển thị khi mở rộng (≤ 200 ký tự, tùy chọn) */
  detail?: string
  /** Ví dụ giá trị hợp lệ */
  example?: string
  /** Link tài liệu ngoài (provider doc, API portal) */
  docUrl?: string
}

export const FIELD_HELP: Record<string, FieldHelp> = {
  // ── Runtime ──────────────────────────────────────────────────────────────────
  threadRequest: {
    label: 'Luồng gửi request',
    hint: 'Số luồng xử lý đồng thời. Cao → nhanh hơn nhưng dễ bị Facebook rate-limit.',
    detail: 'Khuyến nghị: 5–20 tài khoản thường, 1–5 tài khoản giá trị cao. Tối đa 300.',
    example: '10',
  },
  delayRequest: {
    label: 'Delay giữa request',
    hint: 'Thời gian chờ giữa mỗi request (milliseconds). 0 = không delay.',
    detail: 'Delay cao giúp tránh rate-limit tạm thời. Thông thường 500–2000ms cho tài khoản thường.',
    example: '1000',
  },
  threadCheckInfo: {
    label: 'Luồng check info',
    hint: 'Số luồng riêng dùng để kiểm tra thông tin tài khoản (ẩn, không liên quan đến request chính).',
    detail: 'Thường bằng hoặc nhỏ hơn threadRequest. 0 = không check info song song.',
    example: '5',
  },
  delayChangeIp: {
    label: 'Delay đổi IP',
    hint: 'Số giây chờ sau khi đổi IP trước khi tiếp tục gửi request.',
    detail: 'Cần thiết để IP mới được kích hoạt đầy đủ trước khi dùng. Khuyến nghị: 3–10 giây.',
    example: '5',
  },
  checkIpBeforeRun: {
    label: 'Kiểm tra IP trước khi chạy',
    hint: 'Tự động xác nhận IP hiện tại khớp với proxy đã cấu hình trước khi bắt đầu.',
    detail: 'Ngăn chạy tài khoản với IP sai (ví dụ: VPN bị ngắt, proxy lỗi).',
  },

  // ── Account source ────────────────────────────────────────────────────────────
  accountSource: {
    label: 'Nguồn tài khoản',
    hint: '"folder" = đọc từ file trên máy; "api" = mua tự động qua CloneHV khi chạy.',
  },
  cloneHvUsername: {
    label: 'Username CloneHV',
    hint: 'Tài khoản đăng nhập tại clonevn.com / clonetool.com.',
    docUrl: 'https://clonevn.com',
  },
  cloneHvPassword: {
    label: 'Password CloneHV',
    hint: 'Mật khẩu đăng nhập CloneHV. Được lưu mã hóa cục bộ.',
  },
  cloneHvProductId: {
    label: 'Product ID CloneHV',
    hint: 'ID sản phẩm tài khoản cần mua. Lấy từ trang sản phẩm CloneHV.',
    detail: 'Vào trang CloneHV → Chọn sản phẩm → Copy số ID trong URL hoặc tên sản phẩm.',
    example: '18',
    docUrl: 'https://clonevn.com/products',
  },
  cloneHvAmount: {
    label: 'Số lượng mua mỗi lần',
    hint: 'Số tài khoản mua mỗi lần tool chạy (1 batch). Không phải tổng.',
    example: '5',
  },

  // ── Proxy ─────────────────────────────────────────────────────────────────────
  ipProvider: {
    label: 'Nhà cung cấp proxy/IP',
    hint: 'Chọn provider để đổi IP. "none" = không đổi IP khi chạy.',
    detail: 'Mỗi provider yêu cầu API key riêng. Xem tài liệu từng provider để lấy key.',
  },
  proxyList: {
    label: 'Danh sách proxy',
    hint: 'Mỗi dòng 1 proxy. Định dạng: host:port hoặc host:port:user:pass.',
    example: '103.1.2.3:8080:user:pass',
  },
  proxyType: {
    label: 'Loại proxy',
    hint: 'Giao thức của proxy. HTTP/HTTPS cho web thường, SOCKS5 cho ứng dụng.',
  },
  tinsoftKeys: {
    label: 'API Key Tinsoft',
    hint: 'API key từ tinsoft.vn. Mỗi key = 1 gói dịch vụ.',
    docUrl: 'https://tinsoft.vn',
  },
  xproxyServiceUrl: {
    label: 'Service URL XProxy',
    hint: 'URL API XProxy hoặc MobiProxy được cấp sau khi đăng ký dịch vụ.',
    example: 'http://api.xproxy.vn/api/...',
  },

  // ── Captcha ───────────────────────────────────────────────────────────────────
  captchaProvider: {
    label: 'Captcha provider',
    hint: 'Dịch vụ giải captcha tự động. Chỉ hiệu lực khi Facebook hiển thị captcha.',
  },
  'captchaKeys.2captcha': {
    label: 'API Key 2captcha',
    hint: 'Key từ 2captcha.com. Thanh toán theo số lần giải captcha.',
    docUrl: 'https://2captcha.com/enterpage',
  },
  'captchaKeys.capsolver': {
    label: 'API Key CapSolver',
    hint: 'Key từ capsolver.com. Hỗ trợ nhiều loại captcha hơn 2captcha.',
    docUrl: 'https://capsolver.com',
  },
  'captchaKeys.ezcaptcha': {
    label: 'API Key EzCaptcha',
    hint: 'Key từ ez-captcha.com. Thường rẻ hơn các provider khác.',
    docUrl: 'https://ez-captcha.com',
  },
  'captchaKeys.omocaptcha': {
    label: 'API Key OmoCaptcha',
    hint: 'Key từ omocaptcha.com. Provider nội địa Việt Nam.',
    docUrl: 'https://omocaptcha.com',
  },

  // ── Verify ────────────────────────────────────────────────────────────────────
  verifyEnabled: {
    label: 'Bật verify tài khoản',
    hint: 'Khi bật, tool sẽ xác thực tài khoản qua email OTP sau khi chạy.',
  },
  mailProvider: {
    label: 'Nhà cung cấp email OTP',
    hint: 'Dịch vụ cấp email OTP để verify. Chọn "" nếu tự nhập danh sách mail.',
    detail: 'zeus-x/store1s/dongvanfb/mail30s = mua OTP tự động qua API. Còn lại = nhập thủ công.',
  },
  timeDelayCheck: {
    label: 'Delay sau submit OTP',
    hint: 'Số giây chờ sau khi submit confirm OTP rồi mới đọc response Facebook (giả lập độ trễ người dùng).',
    detail: 'Quá thấp → có thể đọc response sớm; quá cao → tốn thời gian. Khuyến nghị: 5–15 giây.',
    example: '10',
  },
  timeDelaySendCode: {
    label: 'Chờ OTP tối đa',
    hint: 'Tổng thời gian tối đa chờ OTP về mail (giây). Hết thời gian này mà chưa thấy → timeout.',
    detail: 'Khuyến nghị 30–60s. Mail tempmail ~30s, mail rent có thể chậm hơn (60s). Quá ngắn → timeout liên tục.',
    example: '30',
  },
  sendAgainCode: {
    label: 'Tự động gửi lại OTP khi timeout',
    hint: 'Khi hết "Chờ OTP tối đa" mà chưa thấy code, tool sẽ tự nhấn "Gửi lại" trên Facebook và chờ thêm 1 lần nữa.',
  },
  outputPath: {
    label: 'Thư mục lưu kết quả verify',
    hint: 'Thư mục chứa Live.txt và Die.txt sau khi chạy. Phải tồn tại trên máy.',
  },
  checkLiveDieEnabled: {
    label: 'Kiểm tra sau reg',
    hint: 'Bật lớp xác minh sau khi confirm OTP (post-verify): check pending/checkpoint để chống ghi nhầm acc chết vào SuccessVerify_No2FA.txt. KHUYẾN NGHỊ luôn bật. Step 3.5 của s273 vẫn LUÔN chạy không phụ thuộc checkbox.',
  },

  // ── Mail providers ─────────────────────────────────────────────────────────
  zeusXApiKey: {
    label: 'API Key ZeusX',
    hint: 'Key từ zeus-x.ru. Dùng để đăng ký email OTP tự động khi verify.',
    docUrl: 'https://zeus-x.ru',
  },
  store1sApiKey: {
    label: 'API Key Store1s',
    hint: 'Key từ store1s.com. Mua email Hotmail/Outlook có sẵn để verify.',
    docUrl: 'https://store1s.com',
  },
  dvfbApiKey: {
    label: 'API Key DongVanFB',
    hint: 'Key từ dongvanfb.net. Dịch vụ email OTP chất lượng cao trong nước.',
    docUrl: 'https://dongvanfb.net',
  },
  mail30sApiKey: {
    label: 'API Key Mail30s',
    hint: 'Key từ mailotp.com (tên cũ mail30s). Mua email OTP theo sản phẩm.',
    docUrl: 'https://mailotp.com',
  },

  // ── Register ──────────────────────────────────────────────────────────────────
  createEnabled: {
    label: 'Tạo tài khoản tự động',
    hint: 'Khi bật, tool sẽ tạo tài khoản Facebook mới thay vì verify tài khoản sẵn có.',
  },
  createType: {
    label: 'Kiểu tài khoản tạo',
    hint: '"spam" = tài khoản cơ bản không cần info đầy đủ; "tut" = tài khoản đầy đủ thông tin hơn.',
    detail: 'Tài khoản TUT (Tutorial) thường sống lâu hơn spam nhưng tốn nhiều tài nguyên hơn.',
  },
  createOutputPath: {
    label: 'Thư mục lưu tài khoản tạo thành công',
    hint: 'Tool sẽ ghi created_accounts.txt vào thư mục này sau mỗi batch.',
  },

  // ── Device ────────────────────────────────────────────────────────────────────
  uaIphoneList: {
    label: 'Danh sách User-Agent iPhone',
    hint: 'Mỗi dòng 1 UA string. Tool random chọn UA cho mỗi tài khoản khi chạy.',
    detail: 'UA đa dạng giúp tránh fingerprint trùng lặp. Khuyến nghị: ≥ 20 UA strings.',
    example: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) ...',
  },

  // ── Global ────────────────────────────────────────────────────────────────────
  loginPlatform: {
    label: 'Nền tảng đăng nhập',
    hint: 'Loại tài khoản đang xử lý. Ảnh hưởng đến login method và API endpoint được dùng.',
  },
  loginMethod: {
    label: 'Phương thức đăng nhập',
    hint: 'Cách tool đăng nhập vào tài khoản. Tùy platform sẽ có các option khác nhau.',
  },
  saveRunColumn: {
    label: 'Lưu kết quả vào cột Run',
    hint: 'Sau khi chạy, ghi kết quả vào cột "Run" trong grid để xem lại.',
  },
  backupDB: {
    label: 'Backup database trước khi chạy',
    hint: 'Tạo bản sao database trước mỗi lần chạy. Khuyến nghị bật để tránh mất dữ liệu.',
  },
  closeAfterDone: {
    label: 'Đóng tool sau khi xong',
    hint: 'Tự động tắt ứng dụng khi batch chạy hoàn thành. Hữu ích khi chạy theo lịch.',
  },
}

/**
 * Lấy help text cho một field.
 * @param key - key của field (e.g. 'threadRequest', 'captchaKeys.2captcha')
 */
export function getFieldHelp(key: string): FieldHelp | null {
  return FIELD_HELP[key] ?? null
}
