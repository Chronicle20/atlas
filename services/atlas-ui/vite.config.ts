/// <reference types="vitest/config" />
import path from "path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

const ingressPort = process.env.VITE_INGRESS_PORT || "8080";
const ingressHost = process.env.VITE_INGRESS_HOST || "localhost";
const proxyTarget =
  process.env.VITE_PROXY_TARGET || `http://${ingressHost}:${ingressPort}`;

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    proxy: {
      "/api": {
        target: proxyTarget,
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test/setup.ts"],
    include: ["src/**/*.test.{ts,tsx}"],
    fakeTimers: {
      shouldAdvanceTime: true,
    },
  },
});
