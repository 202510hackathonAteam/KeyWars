// import文は変更なし
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [
    react(),
    // tailwindcss(),
  ],
  server: {
    host: true,
    port: 5173,
  },
  // 以下追加 //
  build: { outDir: 'dist' }, // ビルドしたファイルを出力するディレクトリ名
  base: './', // ビルド後のHTMLからの参照を相対パスにする
})