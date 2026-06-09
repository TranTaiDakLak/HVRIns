// proxies.mock.ts — Mock implementation cho IProxyService

import type { IProxyService, Proxy, ProxyTestResult } from '../contracts'

const mockProxies: Proxy[] = [
  { id: 1, name: 'US Proxy 1', host: '192.168.1.100', port: 8080, username: 'user1', password: 'pass1', type: 'HTTP', note: 'Proxy US nhanh', lastTestResult: 'OK - 120ms' },
  { id: 2, name: 'VN Proxy 1', host: '10.0.0.50', port: 1080, username: 'user2', password: 'pass2', type: 'SOCKS5', note: 'Proxy VN', lastTestResult: 'OK - 45ms' },
  { id: 3, name: 'EU Proxy 1', host: '172.16.0.25', port: 3128, username: '', password: '', type: 'HTTP', note: 'Proxy EU miễn phí', lastTestResult: 'FAIL - timeout' },
  { id: 4, name: 'SG Proxy 1', host: '103.15.20.30', port: 8888, username: 'admin', password: 'secret', type: 'HTTPS', note: '', lastTestResult: 'OK - 80ms' },
  { id: 5, name: 'JP Proxy 1', host: '45.76.100.50', port: 9090, username: 'jp_user', password: 'jp_pass', type: 'SOCKS5', note: 'Proxy Japan', lastTestResult: '' },
]

function delay(ms = 100): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

export const proxiesMock: IProxyService = {
  async list(): Promise<Proxy[]> {
    await delay()
    return mockProxies.map(p => ({ ...p }))
  },

  async save(proxy: Proxy): Promise<void> {
    await delay()
    const idx = mockProxies.findIndex(p => p.id === proxy.id)
    if (idx >= 0) {
      mockProxies[idx] = { ...proxy }
    } else {
      mockProxies.push({ ...proxy, id: mockProxies.length + 1 })
    }
  },

  async delete(id: number): Promise<void> {
    await delay()
    const idx = mockProxies.findIndex(p => p.id === id)
    if (idx >= 0) mockProxies.splice(idx, 1)
  },

  async test(id: number): Promise<ProxyTestResult> {
    await delay(500)
    const proxy = mockProxies.find(p => p.id === id)
    if (!proxy) throw { code: 'NOT_FOUND', message: `Proxy ID ${id} không tồn tại` }

    // Giả lập kết quả test ngẫu nhiên
    const success = Math.random() > 0.3
    const result: ProxyTestResult = success
      ? { success: true, latency: Math.floor(Math.random() * 200) + 20, ip: proxy.host }
      : { success: false, latency: 0, ip: '', error: 'Connection timeout' }

    proxy.lastTestResult = success
      ? `OK - ${result.latency}ms`
      : `FAIL - ${result.error}`

    return result
  },
}
