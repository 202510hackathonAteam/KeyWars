import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0', // これでコンテナ外からアクセス可能
    port: 5173,      // 必要なら指定
  },
})
