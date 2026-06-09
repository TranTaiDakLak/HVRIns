// index.ts — Vue Router setup

import { createRouter, createWebHashHistory } from 'vue-router'
import { routes } from './routes'
import { usePreferencesStore } from '../../stores/preferences.store'

const router = createRouter({
  // Wails dùng hash history (không có server cho HTML5 history)
  history: createWebHashHistory(),
  routes,
})

// Lưu route cuối cùng vào preferences
router.afterEach((to) => {
  if (to.path !== '/') {
    const prefs = usePreferencesStore()
    prefs.setLastRoute(to.path)
  }
})

export default router
