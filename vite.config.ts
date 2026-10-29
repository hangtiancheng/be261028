import { defineConfig } from "vite";
import tailwindcss from "@tailwindcss/vite";

// be261028 frontend build configuration:
// - JSX is compiled with the @yukino.js/lit-jsx automatic runtime (esbuild).
// - TailwindCSS v4 is wired in through the official Vite plugin.
// - During development /api is proxied to the local Go gateway (8091);
//   in production the gateway serves dist/ statically (same origin).
export default defineConfig({
  plugins: [tailwindcss()],
  esbuild: {
    jsx: "automatic",
    jsxImportSource: "@yukino.js/lit-jsx",
  },
  server: {
    host: "127.0.0.1",
    port: 5173,
    proxy: {
      "/api": {
        target: "http://127.0.0.1:8091",
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: "dist",
    target: "es2022",
  },
});
