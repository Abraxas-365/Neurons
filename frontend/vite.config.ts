import path from "path"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

// Other services on this machine also default to 8080, so the API target is
// overridable instead of hard-coded.
const apiTarget = process.env.VITE_API_TARGET ?? "http://localhost:8081"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(import.meta.dirname, "./src"),
    },
  },
  server: {
    port: 5173,
    proxy: {
      // The Go API and the auth endpoints both live on the same origin in
      // production, so proxying keeps cookies and relative URLs working in dev.
      "/api": { target: apiTarget, changeOrigin: true },
      "/auth": { target: apiTarget, changeOrigin: true },
    },
  },
})
