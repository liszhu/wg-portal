import { fileURLToPath, URL } from 'url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  build: {
    outDir: '../frontend-dist',
    emptyOutDir: true
  },
  // local dev api (proxy to avoid cors problems)
  server: {
    proxy: {
      "/frontend": {
        target: "http://localhost:5000",
        changeOrigin: true,
        secure: false,
        withCredentials: true,
        rewrite: (path) => path,
      },
      "/auth": {
        target: "http://localhost:5000",
        changeOrigin: true,
        secure: false,
        withCredentials: true,
        rewrite: (path) => path,
      },
    },
  },
})
