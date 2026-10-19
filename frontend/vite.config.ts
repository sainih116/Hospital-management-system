import { fileURLToPath, URL } from "node:url";

import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// The `test` property below type-checks via the local `declare module
// "vite"` augmentation in vite.config.vitest.d.ts — see that file's
// comment for why the previous `/// <reference types="vitest/config" />`
// approach didn't actually merge (a nested-vite-version mismatch between
// this project's top-level vite and the vite bundled inside
// node_modules/vitest meant that reference augmented a different `vite`
// module instance). `node:url`/`import.meta.url` are typed by the sibling
// vite.config.types.d.ts.

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:5000",
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./tests/setup.ts"],
  },
});
