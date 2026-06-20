// flows.mock.ts — Mock implementation cho IFlowService

import type { IFlowService, Flow } from '@/bridge/contracts'

const mockFlows: Flow[] = [
  {
    id: 1,
    name: 'Login Flow',
    description: 'Đăng nhập và kiểm tra trạng thái account',
    engineType: 'browser',
    steps: [
      { stepNo: 1, actionKey: 'open_browser', inputText: 'https://example.com', param1: '', param2: '', param3: '', param4: '', param5: '', timeout: 30, retry: 2, enabled: true },
      { stepNo: 2, actionKey: 'fill_input', inputText: '#email', param1: '{uid}', param2: '', param3: '', param4: '', param5: '', timeout: 10, retry: 1, enabled: true },
      { stepNo: 3, actionKey: 'fill_input', inputText: '#password', param1: '{password}', param2: '', param3: '', param4: '', param5: '', timeout: 10, retry: 1, enabled: true },
      { stepNo: 4, actionKey: 'click', inputText: '#login-btn', param1: '', param2: '', param3: '', param4: '', param5: '', timeout: 15, retry: 2, enabled: true },
      { stepNo: 5, actionKey: 'wait', inputText: '3000', param1: '', param2: '', param3: '', param4: '', param5: '', timeout: 5, retry: 0, enabled: true },
    ],
  },
  {
    id: 2,
    name: 'Check Profile',
    description: 'Kiểm tra thông tin profile sau khi đăng nhập',
    engineType: 'browser',
    steps: [
      { stepNo: 1, actionKey: 'navigate', inputText: '/profile', param1: '', param2: '', param3: '', param4: '', param5: '', timeout: 15, retry: 1, enabled: true },
      { stepNo: 2, actionKey: 'extract_text', inputText: '.profile-name', param1: 'fullName', param2: '', param3: '', param4: '', param5: '', timeout: 10, retry: 1, enabled: true },
      { stepNo: 3, actionKey: 'screenshot', inputText: 'profile_{uid}', param1: '', param2: '', param3: '', param4: '', param5: '', timeout: 10, retry: 0, enabled: true },
    ],
  },
  {
    id: 3,
    name: 'Mass Action',
    description: 'Thực hiện hành động hàng loạt trên accounts',
    engineType: 'api',
    steps: [
      { stepNo: 1, actionKey: 'api_call', inputText: '/api/action', param1: 'POST', param2: '{"action":"like"}', param3: '', param4: '', param5: '', timeout: 30, retry: 3, enabled: true },
      { stepNo: 2, actionKey: 'delay', inputText: '5000', param1: '', param2: '', param3: '', param4: '', param5: '', timeout: 0, retry: 0, enabled: true },
    ],
  },
]

function delay(ms = 100): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

export const flowsMock: IFlowService = {
  async list(): Promise<Flow[]> {
    await delay()
    return mockFlows.map(f => ({ ...f, steps: [...f.steps] }))
  },

  async get(id: number): Promise<Flow> {
    await delay()
    const flow = mockFlows.find(f => f.id === id)
    if (!flow) throw { code: 'NOT_FOUND', message: `Flow ID ${id} không tồn tại` }
    return { ...flow, steps: [...flow.steps] }
  },

  async save(flow: Flow): Promise<void> {
    await delay()
    const idx = mockFlows.findIndex(f => f.id === flow.id)
    if (idx >= 0) {
      mockFlows[idx] = { ...flow }
    } else {
      mockFlows.push({ ...flow, id: mockFlows.length + 1 })
    }
  },

  async delete(id: number): Promise<void> {
    await delay()
    const idx = mockFlows.findIndex(f => f.id === id)
    if (idx >= 0) mockFlows.splice(idx, 1)
  },
}
