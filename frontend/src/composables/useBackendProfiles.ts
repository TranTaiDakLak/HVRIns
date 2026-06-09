// useBackendProfiles.ts — Profile CRUD backed by the Go/Wails profile API.
// Drop-in replacement for useSettingsProfiles for pages that use backend settings.
// Profile config is stored server-side in AppSettings; we only track id+name here.

import { ref, onMounted, type Ref } from 'vue'
import { getProfileService } from '../bridge/client'
import type { SavedProfile } from './useSettingsProfiles'

export interface UseBackendProfilesReturn {
  profiles: Ref<SavedProfile[]>
  activeProfileId: Ref<string | null>
  /** Create new profile on backend. Returns the new profile ID or error string. */
  saveProfile: (name: string) => Promise<string>
  /** Set active profile on backend and reload settings (caller must reload form). */
  loadProfile: (id: string) => Promise<void>
  /** Clone active profile with new name on backend. */
  cloneProfile: (_srcId: string, name: string) => Promise<string>
  /** Delete profile on backend. */
  deleteProfile: (id: string) => Promise<void>
  /** Rename profile on backend. */
  renameProfile: (id: string, name: string) => Promise<void>
  /** Refresh profile list from backend. */
  refresh: () => Promise<void>
}

/**
 * Composable quản lý profile CRUD thông qua Go/Wails backend.
 * Thay thế useSettingsProfiles cho các page dùng backend settings.
 * Profile config được lưu server-side trong AppSettings; composable chỉ track id + name.
 *
 * Tự động gọi refresh() khi component mounted.
 */
export function useBackendProfiles(): UseBackendProfilesReturn {
  const profiles = ref<SavedProfile[]>([])
  const activeProfileId = ref<string | null>(null)

  /** Tải lại danh sách profile và activeProfileId từ backend. */
  async function refresh() {
    try {
      const svc = await getProfileService()
      const [list, activeId] = await Promise.all([svc.list(), svc.getActiveId()])
      const now = new Date().toISOString()
      profiles.value = list.map(p => ({
        id: p.id,
        name: p.name,
        createdAt: now,
        updatedAt: now,
        data: {},
      }))
      activeProfileId.value = activeId || null
    } catch (err) {
      console.error('[useBackendProfiles] refresh error:', err)
    }
  }

  /**
   * Tạo profile mới trên backend với settings hiện tại.
   * @param name Tên profile hiển thị. Sau khi tạo xong tự động refresh danh sách.
   * @returns ID của profile mới, hoặc string bắt đầu bằng "Lỗi:" nếu thất bại.
   */
  async function saveProfile(name: string): Promise<string> {
    try {
      const svc = await getProfileService()
      const id = await svc.create(name)
      await refresh()
      return id
    } catch (err) {
      return 'Lỗi: ' + String(err)
    }
  }

  /**
   * Đặt profile active trên backend. Caller chịu trách nhiệm reload form sau khi gọi.
   * @param id ID của profile cần activate.
   */
  async function loadProfile(id: string): Promise<void> {
    try {
      const svc = await getProfileService()
      await svc.setActive(id)
      activeProfileId.value = id
    } catch (err) {
      console.error('[useBackendProfiles] loadProfile error:', err)
    }
  }

  /**
   * Clone profile active hiện tại với tên mới trên backend.
   * @param _srcId Không dùng — backend tự clone từ active profile.
   * @param name Tên cho profile clone mới.
   * @returns ID của profile clone, hoặc string "Lỗi:" nếu thất bại.
   */
  async function cloneProfile(_srcId: string, name: string): Promise<string> {
    try {
      const svc = await getProfileService()
      const id = await svc.clone(name)
      await refresh()
      return id
    } catch (err) {
      return 'Lỗi: ' + String(err)
    }
  }

  /**
   * Xóa profile khỏi backend và refresh danh sách.
   * @param id ID của profile cần xóa.
   */
  async function deleteProfile(id: string): Promise<void> {
    try {
      const svc = await getProfileService()
      await svc.delete(id)
      await refresh()
    } catch (err) {
      console.error('[useBackendProfiles] deleteProfile error:', err)
    }
  }

  /**
   * Đổi tên profile trên backend và refresh danh sách.
   * @param id ID của profile cần đổi tên.
   * @param name Tên mới.
   */
  async function renameProfile(id: string, name: string): Promise<void> {
    try {
      const svc = await getProfileService()
      await svc.rename(id, name)
      await refresh()
    } catch (err) {
      console.error('[useBackendProfiles] renameProfile error:', err)
    }
  }

  onMounted(refresh)

  return {
    profiles,
    activeProfileId,
    saveProfile,
    loadProfile,
    cloneProfile,
    deleteProfile,
    renameProfile,
    refresh,
  }
}
