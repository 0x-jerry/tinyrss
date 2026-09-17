import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import UnoCSS from 'unocss/vite'

export default defineConfig({
  plugins: [vue(), UnoCSS()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    proxy: {
      // Dev only: the Go backend owns /api in production (embedded SPA).
      // `make dev` runs Go on :8087 and this Vite server on :5173.
      '/api': 'http://127.0.0.1:8087',
    },
  },
  test: {
    environment: 'node',
    include: ['test/**/*.test.ts'],
  },
})
