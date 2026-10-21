import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: "0.0.0.0",
    port: 5173,
    proxy: {
      "/api/identity": {
        target: "http://127.0.0.1:8081",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/identity/, ""),
      },
      "/api/inventory": {
        target: "http://127.0.0.1:8082",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/inventory/, ""),
      },
      "/api/lending": {
        target: "http://127.0.0.1:8083",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/lending/, ""),
      },
      "/api/notification": {
        target: "http://127.0.0.1:8084",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api\/notification/, ""),
      },
    },
  },
  preview: {
    host: "0.0.0.0",
    port: 4173,
  },
});
