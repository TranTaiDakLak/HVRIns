// useSettingsProfiles.ts — Profile CRUD: save/load/clone/delete/switch named profiles
// Profiles are persisted to localStorage under a namespace key.
// Each profile is a named snapshot of a config object (GeneralConfig+IpConfig or VerifyConfig).

import { ref, computed, toRaw } from 'vue'
import type { Ref } from 'vue'

/** Deep clone that works with Vue reactive objects (unlike structuredClone) */
function deepClone<T>(obj: T): T {
  return JSON.parse(JSON.stringify(toRaw(obj)))
}

export interface SavedProfile<T = Record<string, unknown>> {
  id: string
  name: string
  createdAt: string
  updatedAt: string
  data: T
}

const STORAGE_PREFIX = 'ncs_profiles_'

function generateId(): string {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 7)
}

/**
 * Profile CRUD composable.
 * @param namespace - unique key for this profile group (e.g. 'interaction', 'general')
 * @param form - ref to the current form data
 * @param defaultConfig - default config values (used for reset)
 */
export function useSettingsProfiles<T extends Record<string, unknown>>(
  namespace: string,
  form: Ref<T>,
  defaultConfig: T,
) {
  const storageKey = STORAGE_PREFIX + namespace

  // Load profiles from localStorage
  function loadAll(): SavedProfile<T>[] {
    try {
      const raw = localStorage.getItem(storageKey)
      if (!raw) return []
      return JSON.parse(raw) as SavedProfile<T>[]
    } catch {
      return []
    }
  }

  function saveAll(profiles: SavedProfile<T>[]) {
    localStorage.setItem(storageKey, JSON.stringify(profiles))
  }

  const profiles = ref<SavedProfile<T>[]>(loadAll()) as Ref<SavedProfile<T>[]>
  const activeProfileId = ref<string | null>(null)

  const activeProfile = computed(() =>
    profiles.value.find(p => p.id === activeProfileId.value) ?? null,
  )

  /** Save current form as a new named profile */
  function saveProfile(name: string): SavedProfile<T> {
    const now = new Date().toISOString()
    const profile: SavedProfile<T> = {
      id: generateId(),
      name,
      createdAt: now,
      updatedAt: now,
      data: deepClone(form.value),
    }
    profiles.value.push(profile)
    saveAll(profiles.value)
    activeProfileId.value = profile.id
    return profile
  }

  /** Overwrite an existing profile with current form data */
  function updateProfile(id: string) {
    const idx = profiles.value.findIndex(p => p.id === id)
    if (idx === -1) return
    profiles.value[idx].data = deepClone(form.value)
    profiles.value[idx].updatedAt = new Date().toISOString()
    saveAll(profiles.value)
  }

  /** Load a profile into the form */
  function loadProfile(id: string) {
    const profile = profiles.value.find(p => p.id === id)
    if (!profile) return
    form.value = { ...defaultConfig, ...deepClone(profile.data) }
    activeProfileId.value = id
  }

  /** Clone a profile with a new name */
  function cloneProfile(id: string, newName: string): SavedProfile<T> | null {
    const source = profiles.value.find(p => p.id === id)
    if (!source) return null
    const now = new Date().toISOString()
    const cloned: SavedProfile<T> = {
      id: generateId(),
      name: newName,
      createdAt: now,
      updatedAt: now,
      data: deepClone(source.data),
    }
    profiles.value.push(cloned)
    saveAll(profiles.value)
    return cloned
  }

  /** Delete a profile */
  function deleteProfile(id: string) {
    profiles.value = profiles.value.filter(p => p.id !== id)
    saveAll(profiles.value)
    if (activeProfileId.value === id) {
      activeProfileId.value = null
    }
  }

  /** Rename a profile */
  function renameProfile(id: string, newName: string) {
    const profile = profiles.value.find(p => p.id === id)
    if (!profile) return
    profile.name = newName
    saveAll(profiles.value)
  }

  /** Export a single profile as JSON string */
  function exportProfile(id: string): string | null {
    const profile = profiles.value.find(p => p.id === id)
    if (!profile) return null
    return JSON.stringify({ name: profile.name, data: profile.data }, null, 2)
  }

  /** Import a profile from JSON string */
  function importProfile(json: string): SavedProfile<T> | null {
    try {
      const parsed = JSON.parse(json)
      if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return null
      const name = parsed.name ?? 'Imported'
      const data = parsed.data ?? parsed // support both {name, data} and flat config
      const now = new Date().toISOString()
      const profile: SavedProfile<T> = {
        id: generateId(),
        name: typeof name === 'string' ? name : 'Imported',
        createdAt: now,
        updatedAt: now,
        data: { ...defaultConfig, ...data },
      }
      profiles.value.push(profile)
      saveAll(profiles.value)
      return profile
    } catch {
      return null
    }
  }

  /** Reset form to defaults */
  function resetToDefaults() {
    form.value = deepClone(defaultConfig)
    activeProfileId.value = null
  }

  return {
    profiles,
    activeProfileId,
    activeProfile,
    saveProfile,
    updateProfile,
    loadProfile,
    cloneProfile,
    deleteProfile,
    renameProfile,
    exportProfile,
    importProfile,
    resetToDefaults,
  }
}
