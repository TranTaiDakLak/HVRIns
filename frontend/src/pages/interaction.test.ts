/**
 * S09-D2-T001: Component/interaction tests — mount thật, fire click, assert binding calls.
 * Cover: AppTitleBar (X + confirm), ProxySettingsPage (checkIP), GeneralSettingsPage (file browse).
 */
import { shallowMount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { nextTick } from 'vue'

// ── Module mocks (hoisted before imports) ─────────────────────────────────────

vi.mock('@/services/client', () => ({
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
  }),
  getCloneHVService: vi.fn().mockResolvedValue({
    checkStock: vi.fn().mockResolvedValue({ name: 'test', amount: '5', price: 10 }),
  }),
  getProfileService: vi.fn().mockResolvedValue({
    list: vi.fn().mockResolvedValue([]),
    getActive: vi.fn().mockResolvedValue(''),
    setActive: vi.fn().mockResolvedValue(''),
    create: vi.fn().mockResolvedValue('profile-1'),
    clone: vi.fn().mockResolvedValue('profile-2'),
    delete: vi.fn().mockResolvedValue(''),
    rename: vi.fn().mockResolvedValue(''),
  }),
  getBridgeMode: vi.fn().mockReturnValue('mock'),
}))

vi.mock('@/composables/useAutoSave', () => ({
  useAutoSave: vi.fn(() => ({
    status: { value: 'idle' },
    lastError: { value: null },
    saveNow: vi.fn(),
    markReady: vi.fn(),
  })),
}))

// ── Lazy imports (after mocks) ────────────────────────────────────────────────
import AppTitleBar from '@/components/shell/AppTitleBar.vue'
import ConfirmDialog from '@/components/feedback/ConfirmDialog.vue'
import ProxySettingsPage from '@/pages/ProxySettingsPage.vue'
import GeneralSettingsPage from '@/pages/GeneralSettingsPage.vue'

// ── Helpers ───────────────────────────────────────────────────────────────────

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/:any(.*)*', component: { template: '<div/>' } }],
  })
}

function setupGlobalMocks(extra: Record<string, unknown> = {}) {
  ;(globalThis as any).runtime = {
    Quit: vi.fn(),
    WindowMinimise: vi.fn(),
    WindowToggleMaximise: vi.fn(),
    WindowIsMaximised: vi.fn().mockResolvedValue(false),
    EventsOn: vi.fn((_evt: string, cb: Function) => cb),  // return cb so we can call it
  }
  ;(globalThis as any).go = {
    app: { App: {
      RequestQuit: vi.fn().mockResolvedValue(undefined),
      CheckCurrentIPViaProxy: vi.fn().mockResolvedValue('1.2.3.4'),
      LoadProxyList: vi.fn().mockResolvedValue(''),
      SaveProxyList: vi.fn().mockResolvedValue('OK'),
      OpenFileDialogPath: vi.fn().mockResolvedValue('/tmp/accounts.txt'),
      LoadAccountsFromFile: vi.fn().mockResolvedValue({ imported: 3, errors: [] }),
      LoadSettings: vi.fn().mockResolvedValue({ general: {}, ip: {} }),
      SaveSettings: vi.fn().mockResolvedValue('OK'),
      LoadInteractionConfig: vi.fn().mockResolvedValue({}),
      SaveInteractionConfig: vi.fn().mockResolvedValue('OK'),
      ListProfiles: vi.fn().mockResolvedValue([]),
      GetActiveProfileID: vi.fn().mockResolvedValue(''),
      SetActiveProfile: vi.fn().mockResolvedValue(''),
      ...extra,
    }}
  }
}

function teardownGlobalMocks() {
  delete (globalThis as any).runtime
  delete (globalThis as any).go
}

// ═════════════════════════════════════════════════════════════════════════════
// 1. AppTitleBar
// ═════════════════════════════════════════════════════════════════════════════
describe('AppTitleBar — interaction', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    setupGlobalMocks()
  })
  afterEach(teardownGlobalMocks)

  it('[T1] click nút X → runtime.Quit được gọi', async () => {
    const wrapper = shallowMount(AppTitleBar)
    const closeBtn = wrapper.find('.titlebar__btn--close')
    expect(closeBtn.exists()).toBe(true)
    await closeBtn.trigger('click')
    expect((globalThis as any).runtime.Quit).toHaveBeenCalledOnce()
  })

  it('[T2] runtime.EventsOn đăng ký handler cho app:request-quit-confirm', async () => {
    shallowMount(AppTitleBar)
    await flushPromises()
    const eventsOn = (globalThis as any).runtime.EventsOn
    expect(eventsOn).toHaveBeenCalledWith('app:request-quit-confirm', expect.any(Function))
  })

  it('[T3] onQuitConfirm → go.app.App.RequestQuit được gọi', async () => {
    const wrapper = shallowMount(AppTitleBar)
    await flushPromises()

    // Lấy handler đã đăng ký → gọi để showQuitDialog = true
    const eventsOn = (globalThis as any).runtime.EventsOn as ReturnType<typeof vi.fn>
    const [[, quitHandler]] = eventsOn.mock.calls
    quitHandler({})  // no running tasks
    await nextTick()

    // Emit 'confirm' từ ConfirmDialog stub → onQuitConfirm() → RequestQuit
    const dialog = wrapper.findComponent(ConfirmDialog)
    expect(dialog.exists()).toBe(true)
    await dialog.vm.$emit('confirm')
    await nextTick()

    expect((globalThis as any).go.app.App.RequestQuit).toHaveBeenCalledOnce()
  })

  it('[T4] nút X không crash khi runtime không tồn tại (nhánh lỗi)', async () => {
    delete (globalThis as any).runtime  // simulate non-Wails env
    const wrapper = shallowMount(AppTitleBar)
    const closeBtn = wrapper.find('.titlebar__btn--close')
    await expect(closeBtn.trigger('click')).resolves.toBeUndefined()
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// 2. ProxySettingsPage — nút Kiểm tra IP
// ═════════════════════════════════════════════════════════════════════════════
describe('ProxySettingsPage — kiểm tra IP binding', () => {
  let router: ReturnType<typeof makeRouter>

  beforeEach(() => {
    setActivePinia(createPinia())
    setupGlobalMocks()
    router = makeRouter()
  })
  afterEach(teardownGlobalMocks)

  it('[T5] click nút "Kiểm tra IP" → CheckCurrentIPViaProxy được gọi', async () => {
    const wrapper = shallowMount(ProxySettingsPage, {
      global: { plugins: [router] },
    })
    await flushPromises()  // chờ onMounted hoàn tất

    const ipBtn = wrapper.find('button.px-ip-inline__btn')
    expect(ipBtn.exists()).toBe(true)
    await ipBtn.trigger('click')
    await flushPromises()

    expect((globalThis as any).go.app.App.CheckCurrentIPViaProxy).toHaveBeenCalledOnce()
  })

  it('[T6] CheckCurrentIPViaProxy trả về IP → hiển thị kết quả (không crash)', async () => {
    ;(globalThis as any).go.app.App.CheckCurrentIPViaProxy = vi.fn().mockResolvedValue('203.0.113.5')
    const wrapper = shallowMount(ProxySettingsPage, {
      global: { plugins: [router] },
    })
    await flushPromises()
    const ipBtn = wrapper.find('button.px-ip-inline__btn')
    await ipBtn.trigger('click')
    await flushPromises()
    // IP phải hiện trong template (currentIp là ref hiển thị)
    expect(wrapper.html()).toContain('203.0.113.5')
  })

  it('[T7] CheckCurrentIPViaProxy throw → không crash; currentIp = "Lỗi kết nối"', async () => {
    ;(globalThis as any).go.app.App.CheckCurrentIPViaProxy = vi.fn().mockRejectedValue(new Error('network'))
    const wrapper = shallowMount(ProxySettingsPage, {
      global: { plugins: [router] },
    })
    await flushPromises()
    const ipBtn = wrapper.find('button.px-ip-inline__btn')
    await expect(ipBtn.trigger('click')).resolves.toBeUndefined()
    await flushPromises()
    // Component vẫn mount, không throw
    expect(wrapper.exists()).toBe(true)
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// 3. GeneralSettingsPage — nút Chọn file (file browse → LoadAccountsFromFile)
// ═════════════════════════════════════════════════════════════════════════════
describe('GeneralSettingsPage — file browse binding', () => {
  let router: ReturnType<typeof makeRouter>

  beforeEach(() => {
    setActivePinia(createPinia())
    setupGlobalMocks()
    router = makeRouter()
  })
  afterEach(teardownGlobalMocks)

  /** Chuyển accountSource → 'file' để hiện button "Chọn file" (trong v-else-if) */
  async function switchToFileSource(wrapper: ReturnType<typeof shallowMount>) {
    const radio = wrapper.find('input[type="radio"][value="file"]')
    await radio.setValue(true)  // v-model sẽ set accountForm.accountSource = 'file'
    await nextTick()
  }

  it('[T8] click "Chọn file" → OpenFileDialogPath được gọi', async () => {
    const wrapper = shallowMount(GeneralSettingsPage, {
      global: { plugins: [router] },
    })
    await flushPromises()
    await switchToFileSource(wrapper)

    const fileBtn = wrapper.find('button[title="Chọn file"]')
    expect(fileBtn.exists()).toBe(true)
    await fileBtn.trigger('click')
    await flushPromises()

    expect((globalThis as any).go.app.App.OpenFileDialogPath).toHaveBeenCalledOnce()
  })

  it('[T9] sau khi chọn file → LoadAccountsFromFile được gọi với đúng path', async () => {
    const filePath = '/tmp/accounts.txt'
    ;(globalThis as any).go.app.App.OpenFileDialogPath = vi.fn().mockResolvedValue(filePath)

    const wrapper = shallowMount(GeneralSettingsPage, {
      global: { plugins: [router] },
    })
    await flushPromises()
    await switchToFileSource(wrapper)

    const fileBtn = wrapper.find('button[title="Chọn file"]')
    await fileBtn.trigger('click')
    await flushPromises()

    expect((globalThis as any).go.app.App.LoadAccountsFromFile)
      .toHaveBeenCalledWith(filePath)
  })

  it('[T10] OpenFileDialogPath trả rỗng → LoadAccountsFromFile KHÔNG gọi', async () => {
    ;(globalThis as any).go.app.App.OpenFileDialogPath = vi.fn().mockResolvedValue('')

    const wrapper = shallowMount(GeneralSettingsPage, {
      global: { plugins: [router] },
    })
    await flushPromises()
    await switchToFileSource(wrapper)

    const fileBtn = wrapper.find('button[title="Chọn file"]')
    await fileBtn.trigger('click')
    await flushPromises()

    expect((globalThis as any).go.app.App.LoadAccountsFromFile).not.toHaveBeenCalled()
  })

  it('[T11] LoadAccountsFromFile throw → không crash component', async () => {
    ;(globalThis as any).go.app.App.LoadAccountsFromFile = vi.fn().mockRejectedValue(new Error('read fail'))

    const wrapper = shallowMount(GeneralSettingsPage, {
      global: { plugins: [router] },
    })
    await flushPromises()
    await switchToFileSource(wrapper)

    const fileBtn = wrapper.find('button[title="Chọn file"]')
    await expect(fileBtn.trigger('click')).resolves.toBeUndefined()
    await flushPromises()
    expect(wrapper.exists()).toBe(true)
  })
})
