import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwind from "@tailwindcss/vite";
import path from "node:path";
export default defineConfig({
  base: "./",
  plugins: [react(), tailwind()],
  resolve: { alias: { "@": path.resolve(import.meta.dirname, "src") } },
  server: {
    proxy: {
      "/api": "http://127.0.0.1:8796",
      "/auth": "http://127.0.0.1:8796",
    },
  },
});
