// contracts.ts — Định nghĩa interface cho bridge layer
// Dựa trên Wails generated bindings thật (App.d.ts + models.ts)

// === Kiểu dữ liệu Account ===
// Mapping đầy đủ từ WeBM frmFacebook.Core DataGridView (~40 fields quan trọng)
export interface Account {
  id: number
  uid: string
  fullData: string           // cInputAccount - Dữ liệu gốc
  password: string
  twofa: string
  email: string
  passMail: string
  mailRecovery: string
  cookie: string
  token: string
  status: AccountStatus      // cStatus - Trạng thái
  checkpoint: string         // c282 - Checkpoint
  statusAds: string          // cStatusADS - Quảng cáo
  bm: string                 // cBM - Business Manager
  tkqc: string               // cTKQC - Tài khoản quảng cáo
  chatSupport: string        // cChatSupport
  fullName: string           // cFullName - dùng cho verify output & search
  location: string           // dùng cho country tracking trong verify/reg
  avatar: string             // cAvatar
  cover: string              // cCover
  phone: string              // cPhone
  proxy: string              // cProxy
  userAgent: string          // cUA
  note: string               // cNote - Ghi chú
  noteRun: string            // cNoteRun - Ghi chú chạy
  importTime: string         // cDateImportAccount - Ngày nhập
  category: string           // cCategory - Thư mục
  lastRun: string            // cLastRun - Chạy lần cuối
  runProxy: string           // IP thực khi chạy (realtime từ verify)
  activity: string           // cRun - Hoạt động
  sourceCode: string
  categoryId?: number
}

export type AccountStatus = 'live' | 'die' | 'checkpoint' | 'new' | 'unknown'

// === Bộ lọc accounts ===
export interface AccountFilter {
  keyword?: string
  status?: string
  categoryId?: number
  sortBy?: string
  sortDir?: 'asc' | 'desc'
}

// === Kết quả trả về ===
export interface AccountListResult {
  items: Account[]
  total: number
}

export interface ImportResult {
  imported: number
  errors: string[]
}

export interface DeleteResult {
  deleted: number
}

// === Bridge error chuẩn ===
export interface BridgeError {
  code: string
  message: string
  details?: unknown
}

// === Service interfaces ===
export interface IAccountService {
  list(filter: AccountFilter): Promise<AccountListResult>
  get(id: number): Promise<Account>
  import(data: string): Promise<ImportResult>
  delete(ids: number[]): Promise<DeleteResult>
}

export interface IFlowService {
  list(): Promise<Flow[]>
  get(id: number): Promise<Flow>
  save(flow: Flow): Promise<void>
  delete(id: number): Promise<void>
}

export interface IProxyService {
  list(): Promise<Proxy[]>
  save(proxy: Proxy): Promise<void>
  delete(id: number): Promise<void>
  test(id: number): Promise<ProxyTestResult>
}

// === Flow types ===
export interface Flow {
  id: number
  name: string
  description: string
  engineType: string
  steps: FlowStep[]
}

export interface FlowStep {
  stepNo: number
  actionKey: string
  inputText: string
  param1: string
  param2: string
  param3: string
  param4: string
  param5: string
  timeout: number
  retry: number
  enabled: boolean
}

// === Proxy types ===
export interface Proxy {
  id: number
  name: string
  host: string
  port: number
  username: string
  password: string
  type: string
  note: string
  lastTestResult: string
}

export interface ProxyTestResult {
  success: boolean
  latency: number
  ip: string
  error?: string
}

// === Verify types ===
export interface VerifyRunConfig {
  accountIds: number[]
  maxThreads: number
  verifyConfig: {
    verifyEnabled: boolean
    mailProvider: string
    mailList: string
    checkLiveDieEnabled: boolean
    timeDelayCheck: number
    timeDelaySendCode: number
    sendAgainCode: boolean
    pathLive: string
    pathDie: string
    uaIphoneList: string
  }
  outputPath: string
}

export interface VerifyStatusEvent {
  accountId: number
  uid: string
  message: string
}

export interface IVerifyService {
  run(config: VerifyRunConfig): Promise<string>
  stop(): Promise<string>
  isRunning(): Promise<boolean>
}

// === Settings types ===
export interface SettingsData {
  general: import('../types/settings.types').GeneralConfig
  ip: import('../types/settings.types').IpConfig
}

export interface ISettingsService {
  save(data: SettingsData): Promise<string>
  load(): Promise<SettingsData>
}

// === Interaction Config (Thiết lập chạy) ===
export interface IInteractionService {
  save(data: import('../types/interaction.types').VerifyConfig): Promise<string>
  load(): Promise<import('../types/interaction.types').VerifyConfig>
}

// === File Dialog & Path Service ===
export interface IFileDialogService {
  openFolder(): Promise<string>
  openTextFile(): Promise<string>
  openFilePath(): Promise<string>           // mở dialog, trả về PATH (không đọc nội dung)
  readTextFile(path: string): Promise<string> // đọc nội dung file từ path
  validatePath(path: string): Promise<string>
  getAccountSourceFolder(): Promise<string>
  setAccountSourceFolder(path: string): Promise<ImportResult>
  refreshAccountSource(): Promise<ImportResult>
  openFolderInExplorer(path: string): Promise<string>
}

// === CloneHV Stock Service ===
export interface CloneHVStockResult {
  name: string
  amount: string
  price: number
  error: string
}

export interface ICloneHVService {
  checkStock(username: string, password: string, productId: string): Promise<CloneHVStockResult>
}

// === Legacy Import (Phase 4) ===
export interface LegacyMappedField {
  legacyKey: string
  newPath: string
  displayValue: string
  status: 'ok' | 'confirm' | 'sensitive' | 'unsupported'
  note: string
}

export interface LegacyMappingReport {
  mappedOk: LegacyMappedField[]
  needsConfirm: LegacyMappedField[]
  sensitive: LegacyMappedField[]
  unsupported: LegacyMappedField[]
  parseErrors: string[]
}

export interface LegacyParseResult {
  report: LegacyMappingReport
  error: string
}

export interface ILegacyImportService {
  parse(generalJSON: string, interactionJSON: string): Promise<LegacyParseResult>
  apply(generalJSON: string, interactionJSON: string): Promise<string>
}

// === Profile Management ===
export interface ProfileInfo {
  id: string
  name: string
}

export interface IProfileService {
  list(): Promise<ProfileInfo[]>
  getActiveId(): Promise<string>
  setActive(id: string): Promise<string>
  create(name: string): Promise<string>
  clone(name: string): Promise<string>
  rename(id: string, name: string): Promise<string>
  delete(id: string): Promise<string>
}

// === App Info ===
export interface IAppInfoService {
  getVersion(): Promise<string>
}

// === Resource Usage ===
export interface AppResourceUsage {
  ramMb: number
  cpuPct: number
}

export interface IResourceUsageService {
  get(): Promise<AppResourceUsage>
}

// === Verify / Register Runner ===
export interface IVerifyRunnerService {
  run(config: VerifyRunConfig): Promise<string>
  stop(): Promise<string>
  isRunning(): Promise<boolean>
  runRegister(maxThreads: number): Promise<string>
  stopRegister(): Promise<string>
  loadInteractionConfig(): Promise<import('../types/interaction.types').VerifyConfig>
  simulatePlatformUA(platform: string, cfg: import('../types/interaction.types').PlatformUAConfig): Promise<string>
}

// === Event Bus (Wails runtime events) ===
//
// Pattern khuyến nghị (Task 6):
//   const unsubs: Array<() => void> = []
//   unsubs.push(bus.on('verify:complete', handler))
//   unsubs.push(bus.on('verify:status', handler))
//   onUnmounted(() => unsubs.forEach(fn => fn()))
//
// Lý do trả unsub fn: cho phép cleanup selective per-component, tránh `bus.off(name)`
// xóa nhầm listener của component khác cùng nghe event đó.
//
// `off(...events)` GLOBAL — xóa MỌI listener của những event names. Chỉ dùng khi
// muốn reset hoàn toàn (vd reload UI). Component nên tránh dùng → ưu tiên unsub fn.
export interface IEventBusService {
  on(event: string, callback: (...args: any[]) => void): () => void
  off(...events: string[]): void
}

// === Upload Site Config ===
export interface IUploadSiteService {
  save(data: import('../types/upload-site.types').UploadSiteConfig): Promise<string>
  load(): Promise<import('../types/upload-site.types').UploadSiteConfig>
}
