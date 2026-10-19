// Merges Vitest's `test` config option onto Vite's own `UserConfig`.
//
// vitest/config's own .d.ts does the same augmentation, but it does it
// from *inside* node_modules/vitest, where bare specifier "vite" resolves
// to vitest's bundled dependency, node_modules/vitest/node_modules/vite
// (vitest 2.1.9 depends on "vite": "^5.0.0"). This project's top-level
// vite is 6.4.3 — a different resolved module, so that augmentation never
// reaches the "vite" that vite.config.ts (at the project root) imports
// `defineConfig`/`UserConfig` from. Declaring the same augmentation here,
// from a project-root file, targets the correct (top-level) "vite"
// instead.
//
// `VitestUserConfig["test"]` pulls out only the test-options type, not
// vitest's whole UserConfig — so this never touches the mismatched
// Plugin/PluginOption types that caused the original attempt (importing
// `defineConfig` from "vitest/config" instead) to fail: passing a `Plugin`
// produced by @vitejs/plugin-react (typed against the top-level vite) into
// a config shaped by vitest's nested vite's `Plugin` type.
import type { UserConfig as VitestUserConfig } from "vitest/config";

declare module "vite" {
  interface UserConfig {
    test?: NonNullable<VitestUserConfig["test"]>;
  }
}
