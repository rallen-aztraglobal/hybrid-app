import { QueryClient } from '@tanstack/react-query';

/**
 * 全局 QueryClient 单例。
 *
 * 之所以从 main.tsx 抽出来单独成模块：登录/登出时必须能拿到它把缓存整个清掉
 * （见 store/authStore.ts）。后台是**带数据权限**的（10-rbac.md），同一个标签页里
 * 换账号如果沿用上一个账号的缓存，轻则渲染错乱（品牌列表还是上一个账号的空结果，
 * 渠道卡片因为 `brand &&` 匹配不上而一张都画不出来），重则把上一个账号范围内的
 * 数据直接显示给下一个登录的人。
 */
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});
