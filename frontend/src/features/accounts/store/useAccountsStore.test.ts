// useAccountsStore.test.ts — Unit tests cho Pinia accounts store
// Mock service layer để tránh waitForWails() 2s timeout trong test env.

import { setActivePinia, createPinia } from 'pinia'
import { useAccountsStore } from '@/features/accounts/store/useAccountsStore'
import type { Account } from '@/services/contracts'

// Hoist mock: thay thế toàn bộ @/services/client trước khi store import nó
vi.mock('@/services/client', () => ({
  getAccountService: vi.fn().mockResolvedValue({
    list: vi.fn().mockResolvedValue({ items: [] }),
    import: vi.fn().mockResolvedValue({ imported: 2, errors: [] }),
    delete: vi.fn().mockResolvedValue({ deleted: 1 }),
    get: vi.fn().mockResolvedValue(null),
  }),
}))

function makeAccount(id: number, status: Account['status'] = 'new'): Account {
  return {
    id, uid: `uid${id}`, fullData: '', password: '', twofa: '', email: '',
    passMail: '', mailRecovery: '', cookie: '', token: '', status,
    checkpoint: '', statusAds: '', bm: '', tkqc: '', chatSupport: '',
    fullName: '', location: '', avatar: '', cover: '', phone: '',
    proxy: '', userAgent: '', note: '', noteRun: '', importTime: '',
    category: '', lastRun: '', runProxy: '', activity: '', sourceCode: '',
  }
}

describe('useAccountsStore — state khởi tạo', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('state mặc định rỗng', () => {
    const store = useAccountsStore()
    expect(store.accounts).toHaveLength(0)
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
    expect(store.total).toBe(0)
    expect(store.isVerifyRunning).toBe(false)
  })

  it('stats computed: đếm đúng theo status', () => {
    const store = useAccountsStore()
    store.accounts.push(
      makeAccount(1, 'live'),
      makeAccount(2, 'die'),
      makeAccount(3, 'live'),
      makeAccount(4, 'new'),
      makeAccount(5, 'checkpoint'),
    )
    expect(store.stats.live).toBe(2)
    expect(store.stats.die).toBe(1)
    expect(store.stats.new).toBe(1)
    expect(store.stats.checkpoint).toBe(1)
    expect(store.stats.total).toBe(5)
  })
})

describe('useAccountsStore — filter', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('setFilter: merge partial, không ghi đè field khác', () => {
    const store = useAccountsStore()
    store.setFilter({ status: 'live' })
    expect(store.filter).toMatchObject({ status: 'live' })
    store.setFilter({ uid: 'abc' })
    // cả 2 field phải còn
    expect(store.filter).toMatchObject({ status: 'live', uid: 'abc' })
  })
})

describe('useAccountsStore — in-memory mutations', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('removeAccount: xóa đúng row, cập nhật accountsIndex', () => {
    const store = useAccountsStore()
    // Dùng ensureSlot để cả accounts[] lẫn accountsIndex đồng bộ
    store.ensureSlot(10)
    store.ensureSlot(11)
    store.removeAccount(10)
    expect(store.accounts.find(a => a.id === 10)).toBeUndefined()
    expect(store.accounts.find(a => a.id === 11)).toBeDefined()
    expect(store.accountsIndex.has(10)).toBe(false)
  })

  it('clearAccounts: xóa sạch list và index', () => {
    const store = useAccountsStore()
    store.ensureSlot(1)
    store.ensureSlot(2)
    store.clearAccounts()
    expect(store.accounts).toHaveLength(0)
    expect(store.accountsIndex.size).toBe(0)
  })

  it('ensureSlot: tạo placeholder khi chưa có slot', () => {
    const store = useAccountsStore()
    const acc = store.ensureSlot(99)
    expect(acc.id).toBe(99)
    expect(acc.uid).toBe('')
    expect(acc.status).toBe('new')
    expect(store.accounts.find(a => a.id === 99)).toBeDefined()
    expect(store.accountsIndex.has(99)).toBe(true)
  })

  it('ensureSlot: trả về acc hiện có nếu đã tồn tại', () => {
    const store = useAccountsStore()
    const first = store.ensureSlot(5)
    const second = store.ensureSlot(5)
    expect(first).toBe(second) // same object reference
    expect(store.accounts.filter(a => a.id === 5)).toHaveLength(1) // không duplicate
  })

  it('applySlotAssigned: cập nhật đúng field, reset stale', () => {
    const store = useAccountsStore()
    store.ensureSlot(7)
    // set một vài field cũ
    store.accountsIndex.get(7)!.email = 'old@mail.com'
    store.accountsIndex.get(7)!.runProxy = '1.2.3.4'

    store.applySlotAssigned({ slotId: 7, uid: 'user7', password: 'pw7', phone: '09xx', status: 'live' })

    const acc = store.accounts.find(a => a.id === 7)!
    expect(acc.uid).toBe('user7')
    expect(acc.password).toBe('pw7')
    expect(acc.status).toBe('live')
    // fields bị reset
    expect(acc.email).toBe('')
    expect(acc.runProxy).toBe('')
    expect(acc.activity).toBe('')
  })

  it('applySlotAssigned: tạo placeholder nếu slot chưa tồn tại (race condition)', () => {
    const store = useAccountsStore()
    // Gọi applySlotAssigned khi slot 42 chưa có
    store.applySlotAssigned({ slotId: 42, uid: 'u42', password: '', phone: '', status: 'new' })
    const acc = store.accounts.find(a => a.id === 42)
    expect(acc).toBeDefined()
    expect(acc!.uid).toBe('u42')
  })
})

describe('useAccountsStore — proxy cache', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('setRunProxy: cập nhật runProxy trên account + cache', () => {
    const store = useAccountsStore()
    const acc = store.ensureSlot(5)
    store.setRunProxy(5, '10.0.0.1/vn')
    expect(acc.runProxy).toBe('10.0.0.1/vn')
  })

  it('setDisplayProxy: cập nhật proxy trên account', () => {
    const store = useAccountsStore()
    const acc = store.ensureSlot(6)
    store.setDisplayProxy(6, 'socks5://proxy:1234')
    expect(acc.proxy).toBe('socks5://proxy:1234')
  })

  it('clearRunProxyCache: cache không ảnh hưởng sau khi clear', () => {
    const store = useAccountsStore()
    store.ensureSlot(3)
    store.setRunProxy(3, '5.5.5.5')
    store.clearRunProxyCache()
    // Sau clear, giá trị trên object vẫn còn (chỉ cache bị clear)
    // Nhưng fetchAccounts lần sau sẽ không restore từ cache
    // Test: gọi ensureSlot lần 2 và setRunProxy lại → không crash
    store.setRunProxy(3, '6.6.6.6')
    expect(store.accountsIndex.get(3)!.runProxy).toBe('6.6.6.6')
  })
})

describe('useAccountsStore — async actions (mocked service)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('fetchAccounts: loading false sau khi xong, không lỗi', async () => {
    const store = useAccountsStore()
    await store.fetchAccounts()
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
    expect(store.accounts).toHaveLength(0)
  })
})
