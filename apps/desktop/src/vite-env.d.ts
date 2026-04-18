/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_FORGE_API_URL: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
