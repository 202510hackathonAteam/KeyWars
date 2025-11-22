// import文は変更なし
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
  ],
  server: {
    host: "0.0.0.0",
    port: 5173,
    proxy: {
      "/healthz": {
        target: "http://backend:8080",
        changeOrigin: true,
        secure: false,
      },
      "/auth": {
        target: "http://backend:8080",
        changeOrigin: true,
        secure: false,
      },
      "/api": {
        target: "http://backend:8080",
        changeOrigin: true,
        secure: false,
      },

      "/api/v1/ws": {
        target: "http://backend:8080",
        ws: true,       // ❗必須
        changeOrigin: true
      }
    },
  },


  // 以下追加 //
  build: { outDir: 'dist' }, // ビルドしたファイルを出力するディレクトリ名
  base: './', // ビルド後のHTMLからの参照を相対パスにする設定
})
