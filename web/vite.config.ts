import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 5180,
    proxy: { '/api': 'http://127.0.0.1:8787' },
  },
  build: {
    // IconPicker's lazy chunk carries the whole Lucide set (~700 kB) and can't
    // shrink further; the warning still guards everything else.
    chunkSizeWarningLimit: 800,
  },
})
