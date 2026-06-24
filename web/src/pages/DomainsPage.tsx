/**
 * 域名配置页 —— 复刻原型「域名配置」视图：以**大渠道为单位**管理品牌默认域名
 * （ADR-0006：被封时整品牌一起换）。每个品牌一段可编辑域名编辑器 + 保存下发。
 * 保存 = 触发后端生成 /api/app/config 的 CDN 快照，已安装 APK 下次启动自动拉取（ADR-0002）。
 */
import { useBrands } from '@/hooks/queries';
import { BrandDomainsCard } from '@/components/BrandDomainsCard';
import { Note } from '@/components/ui';
import { InfoIcon } from '@/components/icons';

export function DomainsPage() {
  const { data: brands, isLoading } = useBrands();

  return (
    <section>
      <div className="mb-4">
        <Note>
          <InfoIcon className="w-[17px] h-[17px] flex-none mt-0.5" style={{ color: 'var(--brand)' }} />
          <div>
            域名以<b>大渠道为单位</b>统一管理（被封时整品牌一起换）。保存即生成{' '}
            <span className="mono">/api/app/config</span> 的 CDN 快照，
            <b>所有已安装 APK 下次启动自动拉取最新域名，无需重新打包</b>。个别渠道需差异化可在渠道编辑里关闭「继承」。
          </div>
        </Note>
      </div>

      {isLoading ? (
        <div className="flex flex-col gap-4">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="h-[220px] rounded-card bg-panel border border-line animate-pulse" />
          ))}
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          {(brands ?? []).map((b) => (
            <BrandDomainsCard key={b.code} brand={b} />
          ))}
        </div>
      )}
    </section>
  );
}
