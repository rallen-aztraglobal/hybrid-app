import { defineConfig } from 'vitest/config';
import path from 'node:path';

// 轻量单测配置（当前仅覆盖纯函数，如 lib/adjustCsv 的 CSV 解析），环境用 node 即可，无需 jsdom。
export default defineConfig({
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
  },
});
