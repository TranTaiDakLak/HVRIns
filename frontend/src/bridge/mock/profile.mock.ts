// profile.mock.ts — Mock bridge cho profile management
import type { IProfileService, ProfileInfo } from '../contracts'

const _profiles: ProfileInfo[] = [{ id: 'default', name: 'Default' }]
let _activeId = 'default'

export const profileMock: IProfileService = {
  async list(): Promise<ProfileInfo[]> {
    return [..._profiles]
  },
  async getActiveId(): Promise<string> {
    return _activeId
  },
  async setActive(id: string): Promise<string> {
    if (_profiles.find(p => p.id === id)) { _activeId = id; return 'OK' }
    return 'Lỗi: không tìm thấy profile'
  },
  async create(name: string): Promise<string> {
    if (!name) return 'Lỗi: tên rỗng'
    const id = `p_${Date.now()}`
    _profiles.push({ id, name })
    _activeId = id
    return id
  },
  async clone(name: string): Promise<string> {
    if (!name) return 'Lỗi: tên rỗng'
    const id = `p_${Date.now()}`
    _profiles.push({ id, name })
    _activeId = id
    return id
  },
  async rename(id: string, name: string): Promise<string> {
    const p = _profiles.find(p => p.id === id)
    if (!p) return 'Lỗi: không tìm thấy profile'
    p.name = name
    return 'OK'
  },
  async delete(id: string): Promise<string> {
    if (_profiles.length <= 1) return 'Lỗi: phải giữ ít nhất 1 profile'
    const idx = _profiles.findIndex(p => p.id === id)
    if (idx < 0) return 'Lỗi: không tìm thấy profile'
    _profiles.splice(idx, 1)
    if (_activeId === id) _activeId = _profiles[0].id
    return 'OK'
  },
}
