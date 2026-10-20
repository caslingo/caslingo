import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// `yarn dev` serves on port 5173 and proxies /api to `go run main.go` on port 16500
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": `${import.meta.dirname}/src` },
  },
  server: {
    proxy: {
      "/api": "http://localhost:16500",
    },
  },
  build: {
    outDir: "build",
  },
});
