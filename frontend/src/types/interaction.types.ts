// interaction.types.ts — Kiểu dữ liệu cho Thiết lập tương tác
// Mapping từ WeBM frmSetup + frmVerify (configInteraction)

// === Mail provider type (union thay vì string) ===
export type MailProviderType =
  // Temp Mail (free)
  | 'moakt'
  | '@i2b.vn'
  | 'mohmal'
  | 'tempmail-lol'
  | 'mailtm'
  | 'tempmail-plus'
  | 'dropmail'
  | 'guerrillamail'
  | 'spam4me'
  | 'temp-mail.org'
  | 'mail.cx'
  | 'mailtd'
  | 'inboxes'
  | 'dismail'
  | 'mailymg'
  | 'altmails'
  | 'onesecmail'
  | 'firetempmail'
  | 'firetempmail-ctm'
  | 'firetempmail-jd'
  | 'firetempmail-offre'
  | 'fviainboxes'
  | 'byomde'
  | 'dinlaan'
  | 'cryptogmail'
  | 'buslink24'
  | 'boxmailstore'
  | 'mailermnx'
  | 'tempforward'
  | 'tempomintraccoon'
  | 'tempemail'
  | 'tmpinbox'
  | 'tenminutemail'
  | 'tempmailto'
  | 'onesecemail'
  | 'tempmail100'
  | 'tempmailso'
  | 'priyoemail'
  | 'tempmailorgpremium'
  | 'mailtempcom'
  | 'wemakemail'
  | 'mailhv'
  | 'i2b'
  | 'vietxf'
  // Temp Mail port từ NullCoreSummer (đã test tạo mail OK)
  | 'tempmailio'
  | 'anonymmail'
  | 'tempmailnow'
  | 'tempmailworld'
  | 'expressmail'
  | 'tempmail100free'
  | 'fakelegal'
  | 'tempmailbee'
  | 'tempmailapp'
  | 'tempamail'
  | 'tempmailai'
  | 'internxt'
  | 'tempmailasia'
  | 'tempemailcc'
  | 'tempmailerme'
  | 'mailwave'
  | 'tempmail10'
  | 'tempmailpro'
  | 'tempmaildigital'
  | 'tempmailx'
  | 'tempmailid'
  // Rent Mail (mua tự động)
  | 'zeus-x'
  | 'dongvanfb'
  | 'store1s'
  | 'mail30s'
  | 'muamail'
  | 'unlimitmail'
  | 'sptmail'
  | 'emailapiinfo'
  | 'otpcheap'
  | 'shopgmail9999'
  | 'rentgmail'
  | 'otpcodesms'
  | 'wmemail'
  | ''

// === Account create type ===
export type CreateType = 'spam' | 'tut' | 'normal' | ''

// === API response item types (dùng trong useMailProviderStock composable) ===
export interface ZeusXItem {
  AccountCode: string
  Name: string
  Price: number
  Instock: number
}

export interface Mail30sProduct {
  id: number
  name: string
  slug: string
  price: number
  price_display: string
  stock: number
}

export interface DvfbItem {
  id: number
  name: string
  quality: number
  price: number
}

// === VERIFY CONFIG ===
// Mapping từ frmVerify.Designer.cs controls

// === Per-platform UA config (cấu hình UA riêng theo từng API platform) ===
export interface PlatformUAConfig {
  useOriginalUA: boolean
  addVirtualSpecAndroid: boolean
  buildUA: boolean
  replaceCarrier: boolean  // chỉ dùng khi useOriginalUA=true: true=thay FBCR theo IP, false=giữ Viettel gốc
  trackingID: boolean      // chỉ dùng cho SimulatePlatformUA preview; giá trị thực từ form.trackingIDReg/Ver
  uaPoolKey?: string       // override pool riêng cho platform này (undefined = dùng global uaPoolKey)
  kind?: 'reg' | 'ver'     // pick pool FBAV split: regPlatformCfg → 'reg', verPlatformCfg → 'ver'
}

export interface SubMailConfig {
  mailType?: 'temp' | 'rent'
  provider?: MailProviderType | string
  tempMailDomain?: string
  tempMailToken?: string
}

export interface VerifyConfig {
  // Verify tài khoản (groupBox2)
  verifyEnabled: boolean           // chkVerifyAccount
  mailProvider: MailProviderType   // cboVerMail — typed union thay vì string
  mailList: string                 // txtMailVerifyCustom

  // Kiểm tra live/die sau khi verify (groupBox1)
  checkLiveDieEnabled: boolean     // chkCheckLiveDie
  timeDelayCheck: number           // txtTimeDelay (giây)
  timeDelaySendCode: number        // txtTimeDelaySendCode (giây)
  sendAgainCode: boolean           // chkSendAgainCode
  outputPath: string               // txtPathOutput

  // UA type selection — quyết định đọc từ file nào trong Config/UserAgent/
  uaPoolKey: string  // 'android' | 'iphone' | 'request'

  // ZeusX Hotmail — mua email để verify
  zeusXApiKey: string
  zeusXAccountCode: string

  // DongVanFB — mua email để verify
  dvfbApiKey: string
  dvfbAccountType: string

  // Store1s — mua email để verify
  store1sApiKey: string
  store1sProductId: string

  // Mail30s (mailotp.com / mail30s.com) — mua email để verify
  mail30sApiKey: string
  mail30sProductSlug: string

  // TempMailLol (api.tempmail.lol) — email tạm miễn phí / có API key
  tempMailLolApiKey: string

  // Domain tuỳ chỉnh cho temp mail (moakt, mail1sec, tempmail-plus). Vd: "tmpbox.net"
  // Backend đọc field này — giá trị = tempMailDomains[mailProvider] của provider đang chọn.
  tempMailDomain: string

  // Map domain riêng cho từng temp mail provider. UI bind per-provider,
  // auto-save flatten sang tempMailDomain (provider hiện hành) để backend dùng.
  tempMailDomains?: Record<string, string>

  // Token/api key user nhập tay per-provider — fallback cho provider không tự lấy token.
  // UI bind per-provider map, flatten sang tempMailToken active slot khi save.
  tempMailToken?: string
  tempMailTokens?: Record<string, string>

  // MuaMail.Store
  muaMailApiKey: string
  muaMailProductId: string

  // UnlimitMail.com
  unlimitMailApiKey: string
  unlimitMailProductId: string

  // SPTMail.com
  sptMailApiKey: string
  sptMailServiceCode: string

  // EmailAPIInfo (api.emailapi.info / gmail500.com)
  emailAPIInfoApiKey: string
  emailAPIInfoProductCode: string

  // OtpCheap (api.otp.cheap)
  otpCheapApiKey: string
  otpCheapServiceId: string

  // ShopGmail9999 (shopgmail9999.com)
  shopGmail9999ApiKey: string
  shopGmail9999Service: string

  // RentGmail (rentgmail.online)
  rentGmailApiKey: string
  rentGmailPlatform: string

  // OtpCodesSms (otpcodesms.site)
  otpCodesSmsApiKey: string
  otpCodesSmsServiceId: string

  // Wmemail (www.wmemail.com)
  wmemailApiKey: string
  wmemailCommodity: string

  // PriyoEmail (free.priyo.email)
  priyoEmailApiKey: string

  // OTPHotmailPriority — nguồn đọc OTP ưu tiên cho 7 providers Hotmail OAuth2.
  // "dongvan" (default) | "unlimit". Primary fail → fallback reader còn lại.
  otpHotmailPriority: 'dongvan' | 'unlimit' | ''

  // MailPoolBatch — số email mua batch đầu khi khởi động pool (mặc định 50).
  // Các lần sau khi pool cạn, mỗi luồng tự mua 1 con độc lập.
  mailPoolBatch: number

  // Timing & delay
  waitCode: number           // giây chờ code OTP (Wait code)
  waitMail: number           // ms chờ mail đến (Wait mail)
  trySendCode: number        // số lần thử gửi code
  useMailTimes: number       // số lần dùng lại 1 mail
  delayConfirmEmail: number  // giây delay confirm email
  delayCheckLive: number     // giây delay kiểm tra live
  delayVeriReg: number       // giây delay verify khi reg
  delayDisplayResult: number // giây giữ kết quả trên UI trước khi fetch account mới
  addMailRetry: number       // số lần retry thêm khi add mail fail (0 = mặc định 2 outer attempts)
  retryUnknownNow: boolean   // sau pass 1, tự verify lại các acc Unknown/Error 1 pass nữa
  retryUnknownRelogin?: boolean  // UI NullCore: relogin acc Unknown roi verify lai (cosmetic neu backend chua co)

  // API & Logic
  apiVerifyPlatform: string    // nền tảng verify "focus" (UA config bên dưới áp cho version này)
  apiVerifyPlatforms?: string[] // multi-version verify: list version chạy. Mỗi account verify dùng 1 version round-robin. Rỗng → backend dùng [apiVerifyPlatform].
  apiVerifyTokenType: string   // loại token API verify: 'adspw' | 'internal' | ''

  // Verify advanced options (từ C# VerifyAccountAPIAutomation.cs)
  reUseEmail: boolean           // reuseemail — tái sử dụng email giữa các lần verify
  fmUserTmpMail: boolean        // fmusertmpmail — dùng username từ thông tin đăng nhập cho tempmail
  useProxyTempmail: boolean     // useproxytempmail — dùng proxy khi kết nối đến tempmail
  enable2fa: boolean            // enable2fa — bật xác thực 2 bước
  uploadAvatar: boolean         // uploadAvatar — upload ảnh đại diện sau khi verify live
  getNewDatrOnLive: boolean     // getNewDatrOnLive — gọi GraphQL lấy datr mới sau verify Live
  avatarFolderPath: string      // avatarFolderPath — thư mục chứa ảnh avatar (mặc định Config/Avatar)
  forceAddInfo: boolean         // forceaddinfo — bắt buộc thêm thông tin profile sau verify
  delayAddInfo: number          // delayaddinfo — giây delay trước khi thêm thông tin
  addInfo: boolean              // addInfo — bật cập nhật thông tin hồ sơ sau verify live
  addInfoCity: boolean          // addInfoCity — cập nhật thành phố hiện tại
  addInfoHometown: boolean      // addInfoHometown — cập nhật quê quán
  addInfoSchool: boolean        // addInfoSchool — cập nhật trường học
  addInfoCollege: boolean       // addInfoCollege — cập nhật trường đại học
  addInfoWork: boolean          // addInfoWork — cập nhật nơi làm việc
  addInfoRelationship: boolean  // addInfoRelationship — cập nhật tình trạng hôn nhân
  addInfoDataDir: string        // addInfoDataDir — thư mục chứa file data (mặc định Config/AddInfo)
  addInfoDelayMs: number        // addInfoDelayMs — ms delay giữa mỗi bước AddInfo
  addSubEmail: boolean          // addsubemail — thêm email phụ vào tài khoản
  subMail?: SubMailConfig       // Mail #2 (phụ) — provider RIÊNG; undefined = dùng Mail #1
  subMailStash?: SubMailConfig  // cất config Mail #2 khi tắt toggle (bật lại khôi phục)
  createAds: boolean            // createads — tạo tài khoản quảng cáo
  autoUploadAfterVerify: boolean // tự đẩy lên site sau khi verify xong

  // Register settings
  apiRegPlatform: string      // nền tảng API reg "focus" (UA config bên dưới áp cho version này)
  apiRegPlatforms?: string[]  // multi-version reg: list các version chạy. Mỗi luồng cố định 1 version (chia đều theo slot). Rỗng → backend dùng [apiRegPlatform].
  delayReg: number            // giây delay giữa các register
  delayStep: number           // ms delay giữa các step (s561v99)
  leadDomainMail: string      // domain mail chính: '@gmail.com,@yahoo.com'
  passwordReg: string         // mẫu password khi reg
  nameRegLocale: string       // locale tên: 'US' | 'VN' | 'random'
  regMode: string             // chế độ reg: 'Mail' | 'Phone' | 'TempMail' | 'Random'
                              // 'TempMail' — mua mail tạm thật từ provider, verify reuse mail (skip CreateEmail+AddEmail)
  regModeRotate: boolean      // tự động xoay giữa Mail và Phone theo thời gian (không áp dụng khi regMode=TempMail)
  regModeRotateMailMinutes: number   // số phút dùng Mode=Mail trước khi chuyển Phone (default 360 = 6h)
  regModeRotatePhoneMinutes: number  // số phút dùng Mode=Phone trước khi chuyển Mail (default 360 = 6h)
  verifyAfterReg: boolean     // verify ngay sau khi reg
  phoneMailMode: string       // 'random-normal' | 'random-file' | 'fm-phone'
  fmPhoneCode: boolean        // C# FmPhoneCode — strip country code, prefix "0"
  useUGForVerify: boolean     // dùng UG để verify/reg
  regForVerify: boolean       // reg để verify

  // Cookie initial settings (từ C# MainFormUISettings — cookieinitialmethod)
  cookieInitialMethod: string   // 'file' | 'new' — phương thức lấy cookie initial (datr từ file, hoặc sinh ngẫu nhiên trong app)
  limitCookieInitial: boolean   // limitcookieinitial — giới hạn số lượng cookie initial
  limitCookieInitialCount: number // limitcookieinitialcount — số lượng tối đa
  limitCheckpoint: boolean      // dừng batch khi số checkpoint vượt ngưỡng
  limitCheckpointCount: number  // ngưỡng checkpoint tối đa trước khi dừng
  deleteDatrCheckpoint: boolean // xóa datr khỏi cookie_initial.txt khi đạt giới hạn (usage hoặc checkpoint)
  saveNewDatr: boolean          // ghi datr mới thu được từ cookie reg vào cookie_initial.txt
  limitDatrAge: boolean         // xóa datr khỏi pool sau N phút kể từ lúc nạp
  limitDatrAgeMinutes: number   // ngưỡng tuổi datr (phút)

  // Advanced reg settings (từ C# MainFormUISettings)
  buildUA: boolean              // build UA động từ Config/DeviceInfo (device/locale/carrier theo IP/density/resolution)
  addVirtualSpecAndroid: boolean // addvirtualspecandroid — thêm Dalvik prefix vào UA
  useOriginalUA: boolean        // dùng UA gốc cố định theo platform (s555-s560); loại trừ buildUA và addVirtualSpecAndroid
  androidDevicesPath: string    // androiddevicespath — đường dẫn file danh sách thiết bị Android
  keepIpSuccess: boolean        // keepipsuccess — giữ nguyên IP cho lần reg thành công
  keepUaSuccess: boolean        // keepuasuccess — giữ nguyên UA cho slot sau reg thành công
  keepDatrSuccess: boolean      // giữ datr mới của slot sau reg thành công
  keepInitialSuccess: boolean   // Keep Contact: giữ email/phone của slot sau reg thành công (chỉ Mail/Phone/Random, không TempMail)

  // Post-reg actions
  deactiveAccount: boolean      // deactiveaccount — vô hiệu hoá TK sau khi dùng
  sendToChanger: boolean        // sendtochanger — gửi kết quả sang account changer
  autoUploadAfterReg: boolean   // tự đẩy lên site sau khi reg xong

  // Verify extra
  useProxyGmail: boolean        // useproxygmail — dùng proxy khi đăng nhập Gmail để verify

  // Result folder — thư mục lưu toàn bộ kết quả (reg + verify), tương đương C# result/result_DD-MM-YYYY/
  resultFolderPath: string

  // Tạo tài khoản tự động
  createEnabled: boolean
  createType: CreateType     // 'spam' | 'tut' | 'normal' | ''
  createCookieList: string   // mỗi dòng một cookie
  cookieInitialFile: string  // đường dẫn file cookie_initial.txt
  createOutputPath: string   // thư mục lưu file tài khoản tạo thành công

  // Split mode: reg và verify chạy độc lập
  splitMode: boolean
  splitVerifyThreads: number // 0 = bằng regThreads

  // Số luồng chạy song song — trước đây ở GeneralConfig.threadRequest, đã chuyển vào đây
  // để mỗi side (reg/verify) tự cài luồng. splitVerifyThreads ở trên dùng cho verify pool
  // khi (a) chạy verify only hoặc (b) split mode bật.
  regThreads: number

  // Auto-restart sau N phút — tự động dừng + chạy lại từ đầu (rotate proxy/datr).
  autoRestartEnabled: boolean
  autoRestartMinutes: number

  // Thư mục tài khoản verify (chỉ dùng khi verify-only, không có register)
  verifySourceFolderPath: string

  // Tracking ID — thêm XID/<random16>; vào cuối UA (áp dụng toàn bộ, không theo platform)
  trackingIDReg: boolean
  trackingIDVer: boolean

  // UA config riêng theo platform — key = apiRegPlatform / apiVerifyPlatform
  regPlatformUA: Record<string, PlatformUAConfig>
  verifyPlatformUA: Record<string, PlatformUAConfig>
}

// === PRESETS ===
// Preset là một Partial<VerifyConfig> được áp dụng lên form hiện tại.
// KHÔNG reset toàn bộ form — chỉ thay các field liên quan.

export interface VerifyPreset {
  id: string
  name: string
  description: string
  icon: string
  /** Chỉ ghi đè các field này; phần còn lại giữ nguyên */
  patch: Partial<VerifyConfig>
}

export const VERIFY_PRESETS: VerifyPreset[] = [
  {
    id: 'verify-basic',
    name: 'Verify cơ bản',
    description: 'Kiểm tra live/die nhanh, không cần OTP. Phù hợp kiểm tra nhanh batch lớn.',
    icon: '⚡',
    patch: {
      verifyEnabled: true,
      checkLiveDieEnabled: true,
      timeDelayCheck: 5,
      timeDelaySendCode: 5,
      sendAgainCode: false,
      createEnabled: false,
    },
  },
  {
    id: 'verify-advanced',
    name: 'Verify nâng cao',
    description: 'Verify đầy đủ: live/die + OTP, tự động gửi lại nếu không nhận được code.',
    icon: '🔒',
    patch: {
      verifyEnabled: true,
      checkLiveDieEnabled: true,
      timeDelayCheck: 10,
      timeDelaySendCode: 8,
      sendAgainCode: true,
      createEnabled: false,
    },
  },
  {
    id: 'register-basic',
    name: 'Register cơ bản',
    description: 'Tạo tài khoản Spam. Không verify — phù hợp khi cần số lượng nhanh.',
    icon: '🆕',
    patch: {
      createEnabled: true,
      createType: 'spam',
      verifyEnabled: false,
      checkLiveDieEnabled: false,
    },
  },
  {
    id: 'register-with-email',
    name: 'Register + Email phụ',
    description: 'Tạo tài khoản TUT kèm verify OTP qua email. Tài khoản sống lâu hơn Spam.',
    icon: '📧',
    patch: {
      createEnabled: true,
      createType: 'tut',
      verifyEnabled: true,
      checkLiveDieEnabled: true,
      timeDelayCheck: 10,
      timeDelaySendCode: 8,
      sendAgainCode: true,
    },
  },
  {
    id: 'cookie-mode',
    name: 'Cookie mode',
    description: 'Chạy thuần cookie, không cần login password. Tắt verify, chỉ check live/die.',
    icon: '🍪',
    patch: {
      verifyEnabled: true,
      checkLiveDieEnabled: true,
      sendAgainCode: false,
      createEnabled: false,
      mailProvider: '',
    },
  },
]

// Mail providers — flat list dùng cho logic checks
export const VERIFY_MAIL_PROVIDERS: { value: MailProviderType; label: string }[] = [
  { value: 'moakt',       label: 'Moakt' },
  { value: '@i2b.vn',    label: 'Mail1sec' },
  { value: 'mohmal',     label: 'Mohmal.com' },
  { value: 'tempmail-lol', label: 'TempMail LOL' },
  { value: 'zeus-x',     label: 'ZeusX Hotmail' },
  { value: 'dongvanfb',  label: 'DongVanFB Mail' },
  { value: 'store1s',    label: 'Store1s Mail' },
  { value: 'mail30s',    label: 'Mail30s / mailotp.com' },
]

// Mail providers grouped — dùng cho <optgroup> trong select
export const VERIFY_MAIL_PROVIDER_GROUPS: {
  label: string
  providers: { value: MailProviderType; label: string }[]
}[] = [
  {
    label: '— Tự nhập —',
    providers: [{ value: '', label: '— Tự nhập danh sách mail —' }],
  },
  {
    label: 'Temp Mail (miễn phí)',
    providers: [
      { value: 'moakt',        label: 'Moakt' },
      { value: '@i2b.vn',     label: 'Mail1sec' },
      { value: 'mohmal',       label: 'Mohmal.com' },
      { value: 'tempmail-lol', label: 'TempMail LOL' },
      { value: 'mailtm',       label: 'Mail.tm' },
      { value: 'tempmail-plus', label: 'TempMail Plus' },
      { value: 'dropmail',     label: 'Dropmail.me' },
      { value: 'guerrillamail', label: 'GuerrillaMail' },
      { value: 'spam4me',      label: 'Spam4.me' },
      { value: 'inboxes',      label: 'Inboxes.com' },
      { value: 'dismail',      label: 'Dismail.top' },
      { value: 'mailymg',      label: 'Mailymg.com' },
      { value: 'altmails',     label: 'AltMails.com' },
      { value: 'onesecmail',   label: '1secmail.com' },
      { value: 'firetempmail', label: 'FireTempMail.com' },
      { value: 'fviainboxes',  label: 'FviaInboxes.com' },
      { value: 'byomde',       label: 'Byom.de' },
      { value: 'dinlaan',      label: 'Dinlaan.com' },
      { value: 'cryptogmail',  label: 'CryptoGmail.com' },
      { value: 'buslink24',    label: 'Buslink24.com' },
      { value: 'boxmailstore', label: 'BoxMail.store' },
      { value: 'mailermnx',    label: 'Mailer.mnx-family.com' },
      { value: 'tempforward',  label: 'TempForward.com' },
      { value: 'tempomintraccoon', label: 'Tempo.Mintraccoon.com' },
      { value: 'tempemail',    label: 'TempEmail.co' },
      { value: 'tmpinbox',     label: 'TmpInbox.com' },
      { value: 'tenminutemail', label: '10MinuteMail.com' },
      { value: 'tempmailto',   label: 'TempMailTo.com' },
      { value: 'onesecemail',  label: '1secemail.com' },
      { value: 'tempmail100',  label: 'TempMail100.com' },
      { value: 'tempmailso',   label: 'TempMail.so' },
      { value: 'priyoemail',   label: 'Priyo.email (cần API key)' },
      { value: 'tempmailorgpremium', label: 'Temp-Mail.org Premium' },
      { value: 'mailtempcom',  label: 'Mail-Temp.com' },
      { value: 'wemakemail',   label: 'WeMakeMail (cần API key)' },
      { value: 'mailhv',       label: 'MailHV (cần API key)' },
      { value: 'i2b',           label: 'Mail i2b.vn' },
      { value: 'vietxf',        label: 'VietXF' },
    ],
  },
  {
    label: 'Rent Mail (mua tự động)',
    providers: [
      { value: 'zeus-x',       label: 'ZeusX Hotmail' },
      { value: 'dongvanfb',    label: 'DongVanFB Mail' },
      { value: 'store1s',      label: 'Store1s Mail' },
      { value: 'mail30s',      label: 'Mail30s / mailotp.com' },
      { value: 'muamail',      label: 'MuaMail.Store' },
      { value: 'unlimitmail',  label: 'UnlimitMail.com' },
      { value: 'sptmail',      label: 'SPTMail.com' },
      { value: 'emailapiinfo', label: 'EmailAPI.info (Gmail500)' },
      { value: 'otpcheap',     label: 'OTP.cheap' },
      { value: 'shopgmail9999', label: 'ShopGmail9999' },
      { value: 'rentgmail',    label: 'RentGmail.online' },
      { value: 'otpcodesms',   label: 'OtpCodesSms.site' },
      { value: 'wmemail',      label: 'Wmemail.com' },
    ],
  },
]

// ZeusX account types
export const ZEUS_X_ACCOUNT_CODES: { value: string; label: string; price: string }[] = [
  { value: 'HOTMAIL',                    label: 'Hotmail New',                  price: '$0.002' },
  { value: 'OUTLOOK',                    label: 'Outlook New',                  price: '$0.002' },
  { value: 'HOTMAIL_TRUSTED_GRAPH_API',  label: 'Hotmail Trusted [Graph API]',  price: '$0.015' },
  { value: 'OUTLOOK_TRUSTED_GRAPH_API',  label: 'Outlook Trusted [Graph API]',  price: '$0.015' },
  { value: 'HOTMAIL_TRUSTED',            label: 'Hotmail Trusted [IMAP/POP3]',  price: '$0.015' },
  { value: 'OUTLOOK_TRUSTED',            label: 'Outlook Trusted [IMAP/POP3]',  price: '$0.015' },
]

// DongVanFB account types (từ API account_type)
export const DONGVANFB_ACCOUNT_TYPES: { value: string; label: string; price: string }[] = [
  { value: '1',  label: 'HotMail NEW',                price: '50đ' },
  { value: '2',  label: 'OutLook NEW',                price: '50đ' },
  { value: '3',  label: 'OutLook DOMAIN NEW',         price: '50đ' },
  { value: '5',  label: 'Hotmail TRUSTED',             price: '450đ' },
  { value: '6',  label: 'Outlook TRUSTED',             price: '450đ' },
  { value: '8',  label: 'Outlook Domain TRUSTED',      price: '450đ' },
  { value: '49', label: 'MAIL TRUSTED VERY CLONE',     price: '300đ' },
  { value: '52', label: 'Hotmail PVA kèm mail khôi phục', price: '650đ' },
  { value: '53', label: 'HOTMAIL PVA ĐÃ VERY PHONE',  price: '650đ' },
  { value: '55', label: 'Unlock Mail',                 price: '500đ' },
]

// Store1s products (hardcoded vì stable — cập nhật khi provider đổi catalog)
export const STORE1S_PRODUCTS: { value: string; label: string }[] = [
  { value: '40559', label: 'Hotmail Trusted OAuth2 IMAP/POP3 fviainboxes — 289đ' },
  { value: '24540', label: 'Hotmail Trusted OAuth2 IMAP/POP3 Smvmail — 289đ' },
  { value: '50510', label: 'Hotmail Trusted OAuth2 Graph 12-36M — 150đ' },
  { value: '43861', label: 'Hotmail Trusted OAuth2 Graph Skip7 — 160đ' },
  { value: '62225', label: 'Outlook Trusted OAuth2 Graph 12-36M — 150đ' },
  { value: '24435', label: 'Hotmail Trusted OAuth2 IMAP Skip7 — 250đ' },
]

// Giá trị mặc định — từ frmVerify EnsureMinValue defaults
export const DEFAULT_VERIFY_CONFIG: VerifyConfig = {
  verifyEnabled: true,
  mailProvider: 'moakt',
  mailList: '',
  checkLiveDieEnabled: true,
  timeDelayCheck: 5,
  timeDelaySendCode: 5,
  sendAgainCode: true,
  outputPath: '',
  uaPoolKey: 'android',
  zeusXApiKey: '',
  zeusXAccountCode: 'HOTMAIL_TRUSTED_GRAPH_API',
  dvfbApiKey: '',
  dvfbAccountType: '1',
  store1sApiKey: '',
  store1sProductId: '40559',
  mail30sApiKey: '',
  mail30sProductSlug: '',
  tempMailLolApiKey: '',
  tempMailDomain: '',
  tempMailDomains: {},
  tempMailToken: '',
  tempMailTokens: {},
  muaMailApiKey: '',
  muaMailProductId: '',
  unlimitMailApiKey: '',
  unlimitMailProductId: '',
  sptMailApiKey: '',
  sptMailServiceCode: '',
  emailAPIInfoApiKey: '',
  emailAPIInfoProductCode: '',
  otpCheapApiKey: '',
  otpCheapServiceId: '',
  shopGmail9999ApiKey: '',
  shopGmail9999Service: 'facebook',
  rentGmailApiKey: '',
  rentGmailPlatform: 'facebook',
  otpCodesSmsApiKey: '',
  otpCodesSmsServiceId: '',
  wmemailApiKey: '',
  wmemailCommodity: '',
  priyoEmailApiKey: '',
  otpHotmailPriority: 'dongvan',
  mailPoolBatch: 50,
  // Timing
  waitCode: 79,
  waitMail: 700,
  trySendCode: 1,
  useMailTimes: 1,
  delayConfirmEmail: 1,
  delayCheckLive: 5,
  delayVeriReg: 1,
  delayDisplayResult: 1,
  addMailRetry: 0,
  retryUnknownNow: false,
  retryUnknownRelogin: false,
  // API & Logic
  apiVerifyPlatform: 'api android',
  apiVerifyPlatforms: [],
  apiVerifyTokenType: 'adspw',
  // Verify advanced options
  reUseEmail: false,
  fmUserTmpMail: false,
  useProxyTempmail: false,
  enable2fa: false,
  uploadAvatar: false,
  getNewDatrOnLive: false,
  avatarFolderPath: '',
  forceAddInfo: false,
  delayAddInfo: 1,
  addInfo: false,
  addInfoCity: true,
  addInfoHometown: true,
  addInfoSchool: false,
  addInfoCollege: false,
  addInfoWork: false,
  addInfoRelationship: false,
  addInfoDataDir: '',
  addInfoDelayMs: 2000,
  addSubEmail: false,
  createAds: false,
  autoUploadAfterVerify: false,
  // Register settings
  apiRegPlatform: 'ANDROID',
  apiRegPlatforms: [],
  delayReg: 1,
  delayStep: 0,
  leadDomainMail: '@gmail.com,@yahoo.com',
  passwordReg: '',
  nameRegLocale: 'US',
  regMode: 'Mail',
  regModeRotate: false,
  regModeRotateMailMinutes: 360,
  regModeRotatePhoneMinutes: 360,
  verifyAfterReg: true,
  phoneMailMode: 'random-normal',
  fmPhoneCode: false,
  useUGForVerify: false,
  regForVerify: false,
  // Cookie initial settings
  cookieInitialMethod: 'file',
  limitCookieInitial: false,
  limitCookieInitialCount: 3,
  limitCheckpoint: false,
  limitCheckpointCount: 50,
  deleteDatrCheckpoint: false,
  saveNewDatr: false,
  limitDatrAge: false,
  limitDatrAgeMinutes: 60,
  // Advanced reg settings
  buildUA: false,
  addVirtualSpecAndroid: false, // default OFF — dùng UA gốc từ Config/UserAgent/
  useOriginalUA: false,
  androidDevicesPath: '',
  keepIpSuccess: true, // default ON — reg thành công giữ proxy cho acc kế (C# default)
  keepUaSuccess: false,
  keepDatrSuccess: false,
  keepInitialSuccess: false,
  // Post-reg actions
  deactiveAccount: false,
  sendToChanger: false,
  autoUploadAfterReg: false,
  // Verify extra
  useProxyGmail: false,
  // Result folder
  resultFolderPath: '',
  // Tạo tài khoản
  createEnabled: false,
  createType: 'spam',
  createCookieList: '',
  cookieInitialFile: '',
  createOutputPath: '',
  // Split mode
  splitMode: false,
  splitVerifyThreads: 0,
  regThreads: 20,
  autoRestartEnabled: false,
  autoRestartMinutes: 60,
  // Verify source folder
  verifySourceFolderPath: '',
  trackingIDReg: false,
  trackingIDVer: false,
  // Per-platform UA config maps (populated via popup khi đổi API selector)
  regPlatformUA: {},
  verifyPlatformUA: {},
}

// UA type definitions — map sang Config/UserAgent/{file}_UG.txt trên disk.
// 4 pool chuẩn hóa 2026-05:
//   - Android: FB4A native (api android, api token, s23-s563)
//   - iOS:     FBIOS native (iOS HTTP)
//   - PC:      Chrome Desktop Win/Mac (api mfb)
//   - WebMobile: Chrome Mobile Android (api web andr)
export const UA_POOLS = [
  { key: 'android',      label: 'Android',     file: 'Android_UG.txt'    },
  { key: 'iphone',       label: 'iOS',         file: 'iOS_UG.txt'        },
  { key: 'request',      label: 'PC',          file: 'PC_UG.txt'         },
  { key: 'webchrome',    label: 'WebMobile',   file: 'WebChrome_UA.txt'  },
  { key: 'android_mess', label: 'Mess Android', file: 'Android_Mess.txt' },
  { key: 'ios_mess',     label: 'Mess iOS',     file: 'iOS_Mess.txt'     },
] as const

export type UaPoolKey = typeof UA_POOLS[number]['key']
