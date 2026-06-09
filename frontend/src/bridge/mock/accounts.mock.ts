// accounts.mock.ts — Mock implementation cho IAccountService
// Dùng khi chạy ngoài Wails (dev mode, unit test)

import type { IAccountService, Account, AccountFilter, AccountListResult, ImportResult, DeleteResult } from '../contracts'

const statuses = ['live', 'die', 'checkpoint', 'new', 'unknown'] as const
const cities = ['Hà Nội', 'TP.HCM', 'Đà Nẵng', 'Cần Thơ', 'Hải Phòng', '']
const checkpoints = ['', '', '', '282', '282', 'Locked', '']

function randInt(min: number, max: number, seed: number) {
  return min + (seed % (max - min + 1))
}

// Tạo 50 mock accounts với ~40 fields
function generateMockAccounts(): Account[] {
  const accounts: Account[] = []

  for (let i = 0; i < 50; i++) {
    const uid = `6157${String(8400000000 + i * 1897651).slice(0, 10)}`
    const status = statuses[i % statuses.length]
    accounts.push({
      id: i + 1,
      uid,
      fullData: `${uid}|pass_${i + 1}|2fa_${i + 1}`,
      password: `${String.fromCharCode(97 + (i % 26))}${randInt(100, 999, i * 7)}${['abc', 'xyz', 'qwe', 'rty'][i % 4]}${i}`,
      twofa: i % 3 === 0 ? 'TOTP' : i % 3 === 1 ? 'SMS' : '-',
      email: `user${i + 1}@example.com`,
      passMail: `mailpass_${i + 1}`,
      mailRecovery: i % 4 === 0 ? `recovery${i + 1}@backup.com` : '',
      cookie: `c_user=${uid}; xs=abc${i}def;`,
      token: `EAABs${i}ZCtoken${i + 100}`,
      status,
      checkpoint: checkpoints[i % checkpoints.length],
      statusAds: i % 5 === 0 ? 'Active' : i % 5 === 1 ? 'Disabled' : '',
      bm: i % 3 === 0 ? `BM${randInt(100, 999, i * 3)}` : '',
      tkqc: i % 4 === 0 ? `act_${randInt(100000, 999999, i * 11)}` : '',
      chatSupport: '',
      fullName: `Nguyen Van ${String.fromCharCode(65 + (i % 26))}`,
      location: cities[i % cities.length],
      avatar: '',
      cover: '',
      phone: i % 3 === 0 ? `09${randInt(10000000, 99999999, i * 53)}` : '',
      proxy: i % 4 === 0 ? `us.proxy.com:${12000 + i}:user${i}:pass${i}` : '',
      userAgent: i % 5 === 0 ? `Mozilla/5.0 (Windows NT 10.0; Win64; x64)` : '',
      note: i % 3 === 0 ? `Account test ${i + 1}` : '',
      noteRun: '',
      importTime: new Date(Date.now() - i * 3600000).toISOString().replace('T', ' ').slice(0, 16),
      category: `Folder ${(i % 5) + 1}`,
      lastRun: i % 6 === 0 ? new Date(Date.now() - i * 1800000).toISOString().replace('T', ' ').slice(0, 16) : '',
      runProxy: '',
      activity: '',
      sourceCode: `Import #${Math.floor(i / 10) + 1}`,
      categoryId: (i % 5) + 1,
    })
  }

  return accounts
}

let mockAccounts = generateMockAccounts()

function delay(ms = 100): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

function createEmptyAccount(id: number, uid: string): Account {
  return {
    id, uid, fullData: uid, password: '', twofa: '-', email: '', passMail: '',
    mailRecovery: '', cookie: '', token: '', status: 'new', checkpoint: '',
    statusAds: '', bm: '', tkqc: '', chatSupport: '', fullName: '', location: '',
    avatar: '', cover: '', phone: '', proxy: '',
    userAgent: '', note: '', noteRun: '',
    importTime: new Date().toISOString().replace('T', ' ').slice(0, 16),
    category: '', lastRun: '', runProxy: '', activity: '', sourceCode: `Import #${id}`,
    categoryId: undefined,
  }
}

export const accountsMock: IAccountService = {
  async list(filter: AccountFilter): Promise<AccountListResult> {
    await delay()
    let items = [...mockAccounts]

    if (filter.keyword) {
      const kw = filter.keyword.toLowerCase()
      items = items.filter(a =>
        a.uid.toLowerCase().includes(kw) ||
        a.email.toLowerCase().includes(kw) ||
        a.fullName.toLowerCase().includes(kw) ||
        a.note.toLowerCase().includes(kw)
      )
    }

    if (filter.status) {
      items = items.filter(a => a.status === filter.status)
    }

    if (filter.categoryId != null) {
      items = items.filter(a => a.categoryId === filter.categoryId)
    }

    if (filter.sortBy) {
      const dir = filter.sortDir === 'desc' ? -1 : 1
      items.sort((a, b) => {
        const va = String((a as unknown as Record<string, unknown>)[filter.sortBy!] ?? '')
        const vb = String((b as unknown as Record<string, unknown>)[filter.sortBy!] ?? '')
        return va.localeCompare(vb) * dir
      })
    }

    return { items, total: items.length }
  },

  async get(id: number): Promise<Account> {
    await delay()
    const acc = mockAccounts.find(a => a.id === id)
    if (!acc) throw { code: 'NOT_FOUND', message: `Account ID ${id} không tồn tại` }
    return { ...acc }
  },

  async import(data: string): Promise<ImportResult> {
    await delay(300)
    const lines = data.split('\n').filter(l => l.trim())
    const errors: string[] = []
    let imported = 0

    for (let i = 0; i < lines.length; i++) {
      const newId = mockAccounts.length + 1
      mockAccounts.push(createEmptyAccount(newId, lines[i].trim()))
      imported++
    }

    return { imported, errors }
  },

  async delete(ids: number[]): Promise<DeleteResult> {
    await delay()
    const idSet = new Set(ids)
    const before = mockAccounts.length
    mockAccounts = mockAccounts.filter(a => !idSet.has(a.id))
    return { deleted: before - mockAccounts.length }
  },
}
