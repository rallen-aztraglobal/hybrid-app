import { useEffect } from 'react';
import type { Brand } from '@/lib/types';
import { useUiStore } from '@/store/uiStore';
import { useBrands } from './queries';

/**
 * 渠道产线页面（渠道管理 / 打包中心 / 推送）的品牌 Tab 数据源 +
 * 「当前品牌必须落在可见品牌内」的自愈。
 *
 * 背景（10-rbac.md 数据权限）：`GET /brands` 是**按调用者数据范围过滤**的，一个只被授权
 * bp 的角色登录后 brands 只有 bp，而 uiStore.currentBrand 默认是 'ap'——三个用 BrandTabs
 * 的页面都写着 `brand && ...`，匹配不到就整块不渲染，表现为「明明配好了范围，页面却一片
 * 空白」（而且计数类按钮还照常显示，因为渠道列表本身是有数据的）。这里统一兜底：可见品牌
 * 里没有当前品牌时，自动切到第一个可见品牌。
 */
export function useScopedBrands(): { brands: Brand[] | undefined; brand: Brand | undefined } {
  const { data: allBrands } = useBrands();
  // 只做上架包的品牌（wp）不进渠道 Tab（ADR-0017）：它没有渠道、不能建渠道，也没有品牌级
  // FCM 项目（上架包推送走独立项目）。域名配置页与上架包抽屉用未过滤的 useBrands()/BRAND_ORDER。
  const brands = allBrands?.filter((b) => b.supportsChannels !== false);
  const currentBrand = useUiStore((s) => s.currentBrand);
  const setCurrentBrand = useUiStore((s) => s.setCurrentBrand);

  useEffect(() => {
    if (!brands || brands.length === 0) return;
    if (!brands.some((b) => b.code === currentBrand)) setCurrentBrand(brands[0].code);
  }, [brands, currentBrand, setCurrentBrand]);

  return { brands, brand: (brands ?? []).find((b) => b.code === currentBrand) };
}
