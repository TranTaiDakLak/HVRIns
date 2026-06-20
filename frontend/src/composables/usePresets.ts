// usePresets.ts — Áp dụng preset lên VerifyConfig form
// Preset KHÔNG reset toàn bộ form — chỉ ghi đè các field liên quan (patch).
// Dùng trong InteractionSetupPage.vue để cho phép user chọn template nhanh.

import { ref } from 'vue'
import type { Ref } from 'vue'
import type { VerifyConfig } from '@/types/interaction.types'
import { VERIFY_PRESETS } from '@/types/interaction.types'

export type { VerifyPreset } from '@/types/interaction.types'
export { VERIFY_PRESETS }

/**
 * Composable quản lý preset cho VerifyConfig form.
 * Preset KHÔNG reset toàn bộ form — chỉ patch đúng các field trong preset.patch.
 *
 * @param form Ref trỏ đến VerifyConfig form đang được chỉnh sửa.
 *             applyPreset/clearPreset ghi đè trực tiếp lên form.value.
 */
export function usePresets(form: Ref<VerifyConfig>) {
  const lastApplied = ref<string | null>(null)

  /**
   * Áp dụng preset lên form hiện tại.
   * Chỉ ghi đè đúng những field trong preset.patch.
   */
  function applyPreset(presetId: string) {
    const preset = VERIFY_PRESETS.find(p => p.id === presetId)
    if (!preset) return

    // Merge: giữ nguyên các field không có trong patch
    form.value = { ...form.value, ...preset.patch }
    lastApplied.value = presetId
  }

  /**
   * Reset chỉ những field đã bị preset thay đổi về default.
   */
  function clearPreset(defaultConfig: VerifyConfig) {
    if (!lastApplied.value) return
    const preset = VERIFY_PRESETS.find(p => p.id === lastApplied.value)
    if (!preset) return

    // Revert chỉ những key trong patch về default
    const reverted: Partial<VerifyConfig> = {}
    for (const key of Object.keys(preset.patch) as (keyof VerifyConfig)[]) {
      ;(reverted as any)[key] = (defaultConfig as any)[key]
    }
    form.value = { ...form.value, ...reverted }
    lastApplied.value = null
  }

  return { lastApplied, applyPreset, clearPreset, presets: VERIFY_PRESETS }
}
