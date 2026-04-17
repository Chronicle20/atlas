/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_ROOT_API_URL?: string;
  readonly VITE_BUILD_VERSION?: string;
  readonly VITE_ERROR_ENDPOINT?: string;
  readonly VITE_ERROR_API_KEY?: string;
  readonly VITE_ASSET_BASE_URL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
