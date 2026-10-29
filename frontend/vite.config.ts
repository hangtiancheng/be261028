import { defineConfig } from 'vite';
import tailwindcss from '@tailwindcss/vite';

// be261028 前端构建配置：
// - JSX 走 @yukino.js/lit-jsx 的 automatic runtime（esbuild）
// - TailwindCSS v4 通过官方 Vite 插件接入
// - 开发期将 /api 代理到本机 Go 网关(8091)；生产由网关直接服务 dist（同源）
export default defineConfig({
  plugins: [tailwindcss()],
  esbuild: {
    jsx: 'automatic',
    jsxImportSource: '@yukino.js/lit-jsx',
  },
  server: {
    host: '127.0.0.1',
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8091',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    target: 'es2022',
  },
});
