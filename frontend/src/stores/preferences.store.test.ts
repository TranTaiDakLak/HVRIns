import { setActivePinia, createPinia } from 'pinia'
import { nextTick } from 'vue'
import { usePreferencesStore } from '@/stores/preferences.store'

describe('usePreferencesStore', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  describe('trạng thái mặc định (không có localStorage cache)', () => {
    it('theme=dark, density=default, dataMasking=false', () => {
      const prefs = usePreferencesStore()
      expect(prefs.theme).toBe('dark')
      expect(prefs.density).toBe('default')
      expect(prefs.dataMasking).toBe(false)
    })

    it('lastRoute=/accounts', () => {
      const prefs = usePreferencesStore()
      expect(prefs.lastRoute).toBe('/accounts')
    })

    it('uid và status visible theo DEFAULT_VISIBLE_COLUMN_KEYS', () => {
      const prefs = usePreferencesStore()
      expect(prefs.isColumnVisible('uid')).toBe(true)
      expect(prefs.isColumnVisible('status')).toBe(true)
    })

    it('cookie KHÔNG visible theo default', () => {
      const prefs = usePreferencesStore()
      expect(prefs.isColumnVisible('cookie')).toBe(false)
    })
  })

  describe('toggleTheme', () => {
    it('dark → light → dark', () => {
      const prefs = usePreferencesStore()
      prefs.toggleTheme()
      expect(prefs.theme).toBe('light')
      prefs.toggleTheme()
      expect(prefs.theme).toBe('dark')
    })

    it('persist theme vào localStorage', async () => {
      const prefs = usePreferencesStore()
      prefs.toggleTheme()
      await nextTick()
      expect(localStorage.getItem('theme')).toBe('light')
    })
  })

  describe('setDensity', () => {
    it('compact, comfortable, default', () => {
      const prefs = usePreferencesStore()
      prefs.setDensity('compact')
      expect(prefs.density).toBe('compact')
      prefs.setDensity('comfortable')
      expect(prefs.density).toBe('comfortable')
      prefs.setDensity('default')
      expect(prefs.density).toBe('default')
    })

    it('persist density vào localStorage', async () => {
      const prefs = usePreferencesStore()
      prefs.setDensity('compact')
      await nextTick()
      expect(localStorage.getItem('density')).toBe('compact')
    })
  })

  describe('toggleMasking', () => {
    it('false → true → false', () => {
      const prefs = usePreferencesStore()
      prefs.toggleMasking()
      expect(prefs.dataMasking).toBe(true)
      prefs.toggleMasking()
      expect(prefs.dataMasking).toBe(false)
    })
  })

  describe('setLastRoute', () => {
    it('cập nhật lastRoute', () => {
      const prefs = usePreferencesStore()
      prefs.setLastRoute('/proxy-settings')
      expect(prefs.lastRoute).toBe('/proxy-settings')
    })
  })

  describe('toggleColumn', () => {
    it('ẩn cột đang visible', () => {
      const prefs = usePreferencesStore()
      prefs.toggleColumn('uid')
      expect(prefs.isColumnVisible('uid')).toBe(false)
    })

    it('hiện cột đang ẩn', () => {
      const prefs = usePreferencesStore()
      prefs.toggleColumn('cookie') // mặc định ẩn
      expect(prefs.isColumnVisible('cookie')).toBe(true)
    })

    it('toggle 2 lần: về trạng thái gốc', () => {
      const prefs = usePreferencesStore()
      prefs.toggleColumn('uid')
      prefs.toggleColumn('uid')
      expect(prefs.isColumnVisible('uid')).toBe(true)
    })

    it('persist visibleColumns vào localStorage', () => {
      const prefs = usePreferencesStore()
      prefs.toggleColumn('uid')
      const saved = JSON.parse(localStorage.getItem('visible_columns_v2') || '[]')
      expect(saved).not.toContain('uid')
    })
  })

  describe('resetColumns', () => {
    it('khôi phục uid visible sau khi đã ẩn', () => {
      const prefs = usePreferencesStore()
      prefs.toggleColumn('uid')
      expect(prefs.isColumnVisible('uid')).toBe(false)
      prefs.resetColumns()
      expect(prefs.isColumnVisible('uid')).toBe(true)
    })

    it('cookie vẫn ẩn sau reset (không có trong default)', () => {
      const prefs = usePreferencesStore()
      prefs.toggleColumn('cookie')
      prefs.resetColumns()
      expect(prefs.isColumnVisible('cookie')).toBe(false)
    })
  })
})
