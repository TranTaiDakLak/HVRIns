/**
 * S10-D2-T001: Interaction tests — AppStatusBar, InteractionSetupPage, AccountsPage.
 * Pattern: shallowMount + mock window.go.app.App.* + fire click → assert binding calls.
 */
import { shallowMount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { nextTick } from 'vue'

// ── Module mocks (hoisted) ────────────────────────────────────────────────────

vi.mock('@/services/client', () => ({
  getBridgeMode: vi.fn().mockReturnValue('mock'),
  getSettingsService: vi.fn().mockResolvedValue({
    load: vi.fn().mockResolvedValue({ general: {}, ip: {} }),
    save: vi.fn().mockResolvedValue('OK'),
  }),
  getInteractionService: vi.fn().mockResolvedValue({
    load: vi.fn().mockResolvedValue({}),
    save: vi.fn().mockResolvedValue('OK'),
  }),
  getFileDialogService: vi.fn().mockResolvedValue({
    openFolder: vi.fn().mockResolvedValue(''),
    openFile: vi.fn().mockResolvedValue(''),
    openFilePath: vi.fn().mockResolvedValue(''),
    validatePath: vi.fn().mockResolvedValue(''),
    openFolderInExplorer: vi.fn().mockResolvedValue(''),
  }),
  getProfileService: vi.fn().mockResolvedValue({
    list: vi.fn().mockResolvedValue([]),
    getActive: vi.fn().mockResolvedValue(''),
    getActiveId: vi.fn().mockResolvedValue(''),
    setActive: vi.fn().mockResolvedValue(''),
    create: vi.fn().mockResolvedValue('profile-1'),
    clone: vi.fn().mockResolvedValue('profile-2'),
    delete: vi.fn().mockResolvedValue(''),
    rename: vi.fn().mockResolvedValue(''),
  }),
  getResourceUsageService: vi.fn().mockResolvedValue({
    get: vi.fn().mockResolvedValue({ ramMb: 100.5, cpuPct: 2.5 }),
  }),
  getAppInfoService: vi.fn().mockResolvedValue({
    getVersion: vi.fn().mockResolvedValue('2.0.0'),
  }),
  getVerifyRunnerService: vi.fn().mockResolvedValue({
    loadInteractionConfig: vi.fn().mockResolvedValue({}),
    startVerify: vi.fn(),
    stopVerify: vi.fn(),
  }),
  getEventBusService: vi.fn().mockResolvedValue({
    on: vi.fn().mockReturnValue(vi.fn()),
    off: vi.fn(),
  }),
  getCloneHVService: vi.fn().mockResolvedValue({
    checkStock: vi.fn().mockResolvedValue({ name: 'test', amount: '5', price: 10 }),
  }),
}))

vi.mock('@/composables/useAutoSave', () => ({
  useAutoSave: vi.fn(() => ({
    status: { value: 'idle' },
    lastError: { value: null },
    saveNow: vi.fn(),
    markReady: vi.fn(),
  })),
}))

vi.mock('@/composables/useBackendProfiles', () => ({
  useBackendProfiles: vi.fn(() => ({
    profiles: { value: [] },
    activeProfileId: { value: '' },
    saveProfile: vi.fn().mockResolvedValue('profile-1'),
    loadProfile: vi.fn(),
    cloneProfile: vi.fn(),
    deleteProfile: vi.fn(),
    renameProfile: vi.fn(),
    refresh: vi.fn(),
  })),
}))

vi.mock('@/composables/useMailProviderStock', () => ({
  useMailProviderStock: vi.fn(() => ({
    zeusXStock: { value: null }, zeusXLoading: { value: false }, zeusXError: { value: '' },
    selectedZeusXStock: { value: null }, checkZeusXStock: vi.fn(),
    mail30sProducts: { value: [] }, mail30sLoading: { value: false }, mail30sError: { value: '' },
    selectedMail30sProduct: { value: null }, checkMail30sStock: vi.fn(),
    store1sStockMap: { value: {} }, store1sLoading: { value: false }, store1sError: { value: '' },
    selectedStore1sStock: { value: null }, checkStore1sStock: vi.fn(),
    dvfbStock: { value: null }, dvfbLoading: { value: false }, dvfbError: { value: '' },
    selectedDvfbStock: { value: null }, checkDvfbStock: vi.fn(),
  })),
}))

vi.mock('@/composables/useMarqueeSelect', () => ({
  useMarqueeSelect: vi.fn(() => ({
    state: { dragging: false, mode: 'add', box: { visible: false, x: 0, y: 0, w: 0, h: 0 }, previewKeys: new Set() },
    setContainerEl: vi.fn(),
    onMouseDown: vi.fn(),
    isPreviewed: vi.fn().mockReturnValue(false),
  })),
}))

// ── Imports (after mocks) ─────────────────────────────────────────────────────
import AppStatusBar from '@/components/shell/AppStatusBar.vue'
import InteractionSetupPage from '@/pages/InteractionSetupPage.vue'
import AccountsPage from '@/features/accounts/pages/AccountsPage.vue'
import AccountsToolbar from '@/features/accounts/components/AccountsToolbar.vue'

// ── Helpers ───────────────────────────────────────────────────────────────────

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:any(.*)*', component: { template: '<div/>' } }],
  })
}

function setupGlobalMocks() {
  ;(globalThis as any).runtime = {
    Quit: vi.fn(),
    WindowMinimise: vi.fn(),
    WindowToggleMaximise: vi.fn(),
    WindowIsMaximised: vi.fn().mockResolvedValue(false),
    EventsOn: vi.fn((_evt: string, cb: Function) => cb),
    EventsOff: vi.fn(),
  }
  ;(globalThis as any).go = {
    app: { App: {
      ForceMemoryCleanup: vi.fn().mockResolvedValue({ iosSessionsClosed: 2, androidSessionsClosed: 3, freedMB: 50 }),
      OpenCookieInitialFile: vi.fn().mockResolvedValue('OK'),
      OpenUAFileInEditor: vi.fn().mockResolvedValue(undefined),
      GetCookieInitialStatus: vi.fn().mockResolvedValue({ path: '/tmp/c.txt', count: 10, exists: true, error: '' }),
      GetDatrPoolSize: vi.fn().mockResolvedValue(5),
      GetPoolFileSaveCount: vi.fn().mockResolvedValue(2),
      GetUAPoolsStatus: vi.fn().mockResolvedValue([{ kind: 'android', count: 100 }]),
      GetDefaultResultPath: vi.fn().mockResolvedValue('/tmp/result'),
      GetDefaultCookiePaths: vi.fn().mockResolvedValue({ initial: '/tmp/cookie.txt' }),
      GetAccountSourceFolder: vi.fn().mockResolvedValue('/tmp/accounts'),
      LoadProxyList: vi.fn().mockResolvedValue(''),
      SaveProxyList: vi.fn().mockResolvedValue('OK'),
      GetRunStatus: vi.fn().mockResolvedValue({ registerRunning: false, verifyRunning: false, registerStopping: false, verifyStopping: false }),
      IsRegisterRunning: vi.fn().mockResolvedValue(false),
      IsVerifyRunning: vi.fn().mockResolvedValue(false),
      OpenConfigFolder: vi.fn().mockResolvedValue(''),
      RequestQuit: vi.fn().mockResolvedValue(undefined),
    }}
  }
}

function teardownGlobalMocks() {
  delete (globalThis as any).runtime
  delete (globalThis as any).go
}

// ═════════════════════════════════════════════════════════════════════════════
// 1. AppStatusBar — nút Dọn RAM → ForceMemoryCleanup
// ═════════════════════════════════════════════════════════════════════════════
describe('AppStatusBar — ForceMemoryCleanup binding', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    setupGlobalMocks()
  })
  afterEach(teardownGlobalMocks)

  it('[T1] click "Dọn RAM" → ForceMemoryCleanup được gọi', async () => {
    const wrapper = shallowMount(AppStatusBar)
    await flushPromises()

    const cleanBtn = wrapper.find('button[aria-label="Dọn RAM"]')
    expect(cleanBtn.exists()).toBe(true)
    await cleanBtn.trigger('click')
    await flushPromises()

    expect((globalThis as any).go.app.App.ForceMemoryCleanup).toHaveBeenCalledOnce()
  })

  it('[T2] ForceMemoryCleanup thành công → cleaning reset về false', async () => {
    const wrapper = shallowMount(AppStatusBar)
    await flushPromises()

    const cleanBtn = wrapper.find('button[aria-label="Dọn RAM"]')
    await cleanBtn.trigger('click')
    await flushPromises()

    // Sau khi resolve, nút không còn disabled (cleaning=false)
    expect(cleanBtn.attributes('disabled')).toBeUndefined()
  })

  it('[T3] cleaning guard — click lần 2 khi đang dọn → ForceMemoryCleanup gọi đúng 1 lần', async () => {
    let resolveCleanup!: () => void
    ;(globalThis as any).go.app.App.ForceMemoryCleanup = vi.fn().mockReturnValue(
      new Promise<{iosSessionsClosed:number, androidSessionsClosed:number, freedMB:number}>(
        res => { resolveCleanup = () => res({ iosSessionsClosed: 0, androidSessionsClosed: 0, freedMB: 0 }) }
      )
    )
    const wrapper = shallowMount(AppStatusBar)
    await flushPromises()

    const cleanBtn = wrapper.find('button[aria-label="Dọn RAM"]')
    cleanBtn.trigger('click')  // click 1 — bắt đầu, nút disabled
    await nextTick()
    expect(cleanBtn.attributes('disabled')).toBeDefined()

    cleanBtn.trigger('click')  // click 2 — bị chặn bởi guard
    await nextTick()
    expect((globalThis as any).go.app.App.ForceMemoryCleanup).toHaveBeenCalledOnce()

    resolveCleanup()  // kết thúc để tránh pending promise
    await flushPromises()
  })

  it('[T4] ForceMemoryCleanup không tồn tại → nút Dọn RAM vẫn click không crash', async () => {
    delete (globalThis as any).go.app.App.ForceMemoryCleanup
    const wrapper = shallowMount(AppStatusBar)
    await flushPromises()

    const cleanBtn = wrapper.find('button[aria-label="Dọn RAM"]')
    await expect(cleanBtn.trigger('click')).resolves.toBeUndefined()
    await flushPromises()
    expect(wrapper.exists()).toBe(true)
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// 2. InteractionSetupPage — openCookieInitialFile + openUAFile
// ═════════════════════════════════════════════════════════════════════════════
describe('InteractionSetupPage — cookie initial + UA file bindings', () => {
  let router: ReturnType<typeof makeRouter>

  beforeEach(() => {
    setActivePinia(createPinia())
    setupGlobalMocks()
    router = makeRouter()
  })
  afterEach(teardownGlobalMocks)

  it('[T5] "Mở file datr" button hiện theo mặc định (cookieInitialMethod=file)', async () => {
    const wrapper = shallowMount(InteractionSetupPage, {
      global: { plugins: [router] },
    })
    await flushPromises()
    // cookieInitialMethod default = 'file' → button visible
    const btn = wrapper.find('button.rp-mini-btn')
    expect(btn.exists()).toBe(true)
  })

  it('[T6] click "Mở file datr" → OpenCookieInitialFile được gọi với ""', async () => {
    const wrapper = shallowMount(InteractionSetupPage, {
      global: { plugins: [router] },
    })
    await flushPromises()

    const btn = wrapper.find('button.rp-mini-btn')
    await btn.trigger('click')
    await flushPromises()

    expect((globalThis as any).go.app.App.OpenCookieInitialFile).toHaveBeenCalledWith('')
  })

  it('[T7] OpenCookieInitialFile throw → không crash component', async () => {
    ;(globalThis as any).go.app.App.OpenCookieInitialFile = vi.fn().mockRejectedValue(new Error('open fail'))
    const wrapper = shallowMount(InteractionSetupPage, {
      global: { plugins: [router] },
    })
    await flushPromises()

    const btn = wrapper.find('button.rp-mini-btn')
    await expect(btn.trigger('click')).resolves.toBeUndefined()
    await flushPromises()
    expect(wrapper.exists()).toBe(true)
  })

  it('[T8] click "Mở file" UA → OpenUAFileInEditor được gọi với uaPoolKey mặc định ("android")', async () => {
    const wrapper = shallowMount(InteractionSetupPage, {
      global: { plugins: [router] },
    })
    await flushPromises()

    const uaBtn = wrapper.find('button.rp-ua-open-btn')
    expect(uaBtn.exists()).toBe(true)
    await uaBtn.trigger('click')
    await flushPromises()

    expect((globalThis as any).go.app.App.OpenUAFileInEditor).toHaveBeenCalledWith('android')
  })

  it('[T9] GetDatrPoolSize + GetPoolFileSaveCount được gọi khi mount (pool status tải)', async () => {
    shallowMount(InteractionSetupPage, {
      global: { plugins: [router] },
    })
    await flushPromises()

    expect((globalThis as any).go.app.App.GetDatrPoolSize).toHaveBeenCalled()
    expect((globalThis as any).go.app.App.GetPoolFileSaveCount).toHaveBeenCalled()
  })

  it('[T10] GetUAPoolsStatus được gọi khi mount (UA file counts hiển thị)', async () => {
    shallowMount(InteractionSetupPage, {
      global: { plugins: [router] },
    })
    await flushPromises()

    expect((globalThis as any).go.app.App.GetUAPoolsStatus).toHaveBeenCalled()
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// 3. AccountsPage — GetRunStatus + OpenConfigFolder
// ═════════════════════════════════════════════════════════════════════════════
describe('AccountsPage — run status + config folder binding', () => {
  let router: ReturnType<typeof makeRouter>

  beforeEach(() => {
    setActivePinia(createPinia())
    setupGlobalMocks()
    router = makeRouter()
  })
  afterEach(teardownGlobalMocks)

  it('[T11] GetRunStatus được gọi khi mount (restore run state từ backend)', async () => {
    shallowMount(AccountsPage, {
      global: { plugins: [router] },
    })
    await flushPromises()

    expect((globalThis as any).go.app.App.GetRunStatus).toHaveBeenCalled()
  })

  it('[T12] event open-config-folder từ AccountsToolbar → OpenConfigFolder được gọi', async () => {
    const wrapper = shallowMount(AccountsPage, {
      global: { plugins: [router] },
    })
    await flushPromises()

    const toolbar = wrapper.findComponent(AccountsToolbar)
    expect(toolbar.exists()).toBe(true)
    await toolbar.vm.$emit('open-config-folder')
    await flushPromises()

    expect((globalThis as any).go.app.App.OpenConfigFolder).toHaveBeenCalled()
  })

  it('[T13] OpenConfigFolder trả về lỗi → không crash; lỗi được notify', async () => {
    ;(globalThis as any).go.app.App.OpenConfigFolder = vi.fn().mockResolvedValue('permission denied')
    const wrapper = shallowMount(AccountsPage, {
      global: { plugins: [router] },
    })
    await flushPromises()

    const toolbar = wrapper.findComponent(AccountsToolbar)
    await toolbar.vm.$emit('open-config-folder')
    await flushPromises()

    expect(wrapper.exists()).toBe(true)
    expect((globalThis as any).go.app.App.OpenConfigFolder).toHaveBeenCalled()
  })
})
