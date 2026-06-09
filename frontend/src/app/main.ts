// main.ts — Entry point cho Vue app

import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'

// Styles (thứ tự quan trọng: reset → tokens → theme → app styles)
import '../styles/reset.css'
import '../styles/tokens.css'
import '../styles/light.css'

// Tắt context menu mặc định của WebView (Chromium trong Wails)
window.addEventListener('contextmenu', (e) => e.preventDefault())

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')

// Khởi động upload log listener toàn cục — tránh mất log khi chuyển tab
import { useUploadLogStore } from '../stores/uploadLog.store'
useUploadLogStore().init()
