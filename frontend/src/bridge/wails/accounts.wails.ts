// accounts.wails.ts — Wails implementation cho IAccountService
// Gọi trực tiếp Go bindings qua Wails generated code

import type { IAccountService, Account, AccountFilter, AccountListResult, ImportResult, DeleteResult } from '../contracts'
import { ListAccounts, GetAccount, ImportAccounts, DeleteAccounts } from '../../../wailsjs/go/main/App'
import { main } from '../../../wailsjs/go/models'

// Chuyển đổi Wails AccountFilter sang Go struct
function toWailsFilter(filter: AccountFilter): main.AccountFilter {
  return new main.AccountFilter({
    keyword: filter.keyword ?? '',
    status: filter.status ?? '',
    categoryId: filter.categoryId,
    sortBy: filter.sortBy ?? '',
    sortDir: filter.sortDir ?? '',
  })
}

// Chuyển đổi Wails Account sang bridge Account
function fromWailsAccount(wa: main.Account): Account {
  return {
    id: wa.id,
    uid: wa.uid,
    fullData: (wa as any).fullData ?? '',
    password: wa.password,
    twofa: wa.twofa,
    email: wa.email,
    passMail: wa.passMail,
    mailRecovery: wa.mailRecovery,
    cookie: wa.cookie,
    token: wa.token,
    status: wa.status as Account['status'],
    checkpoint: (wa as any).checkpoint ?? '',
    statusAds: (wa as any).statusAds ?? '',
    bm: (wa as any).bm ?? '',
    tkqc: (wa as any).tkqc ?? '',
    chatSupport: (wa as any).chatSupport ?? '',
    fullName: (wa as any).fullName ?? '',
    location: (wa as any).location ?? '',
    avatar: (wa as any).avatar ?? '',
    cover: (wa as any).cover ?? '',
    phone: (wa as any).phone ?? '',
    proxy: (wa as any).proxy ?? '',
    userAgent: (wa as any).userAgent ?? '',
    note: wa.note,
    noteRun: (wa as any).noteRun ?? '',
    importTime: wa.importTime,
    category: (wa as any).category ?? '',
    lastRun: (wa as any).lastRun ?? '',
    runProxy: (wa as any).runProxy ?? '',
    activity: (wa as any).activity ?? '',
    sourceCode: wa.sourceCode,
    categoryId: wa.categoryId,
  }
}

export const accountsWails: IAccountService = {
  async list(filter: AccountFilter): Promise<AccountListResult> {
    const result = await ListAccounts(toWailsFilter(filter))
    return {
      items: (result.items || []).map(fromWailsAccount),
      total: result.total,
    }
  },

  async get(id: number): Promise<Account> {
    const result = await GetAccount(id)
    return fromWailsAccount(result)
  },

  async import(data: string): Promise<ImportResult> {
    const result = await ImportAccounts(data)
    return { imported: result.imported, errors: result.errors || [] }
  },

  async delete(ids: number[]): Promise<DeleteResult> {
    const result = await DeleteAccounts(ids)
    return { deleted: result.deleted }
  },
}
