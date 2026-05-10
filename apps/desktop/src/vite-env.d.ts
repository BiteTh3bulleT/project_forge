/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_FORGE_API_URL: string;
  readonly VITE_FORGE_API_TIMEOUT_MS?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
