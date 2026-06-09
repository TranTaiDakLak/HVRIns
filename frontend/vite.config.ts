import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  // Fix Vue feature flag warnings
  define: {
    __VUE_OPTIONS_API__: true,
    __VUE_PROD_DEVTOOLS__: false,
    __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: false,
  },
  // Tối ưu RAM
  build: {
    sourcemap: false,
  },
  server: {
    watch: {
      // Bỏ qua các thư mục không phải frontend để tránh rebuild thừa
      ignored: [
        '**/node_modules/**',
        '**/reg_debug/**',
        '**/Config/**',
        '**/output/**',
        '**/internal/**',
        '**/build/**',
        '**/*.go',
        '**/*.txt',
        '**/*.log',
      ],
    },
  },
  optimizeDeps: {
    // Pre-bundle chỉ deps thực sự dùng — giảm RAM scan
    include: ['vue', 'vue-router', 'pinia', 'lucide-vue-next'],
  },
})
