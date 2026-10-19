// Ambient types for vite.config.ts only. tsconfig.node.json's "lib" is
// ES2023 with no "DOM" and no "@types/node" installed, so neither
// `import.meta.url` nor the `node:url` module have any type declarations
// in that compilation unit. Declaring exactly the two members vite.config.ts
// actually uses avoids pulling in `@types/node` (a new dependency) just for
// a single config file.
//
// This file intentionally has no top-level import/export of its own, so
// it stays a global ambient script (not a module) under
// tsconfig.node.json's "moduleDetection": "force" — see
// vite.config.vitest.d.ts for the one augmentation that does need an
// import, kept in its own file for exactly this reason.

interface ImportMeta {
  readonly url: string;
}

declare module "node:url" {
  export class URL {
    constructor(input: string, base?: string | URL);
  }
  export function fileURLToPath(url: URL | string): string;
}
