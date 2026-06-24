/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** 强制使用本地 mock（不打后端），值为 '1' 时生效 */
  readonly VITE_USE_MOCK?: string;
  /** 开发期 /api 代理目标（默认 http://localhost:8080） */
  readonly VITE_API_PROXY?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
