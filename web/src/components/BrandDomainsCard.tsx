/**
 * 单个品牌的默认域名卡片（域名配置页用）。可编辑 + 保存下发。
 * 保存前做 §5.7 校验（主域名必填、https、去重）；不通过给红字但仍可保存（与原型/文档一致：
 * 「不通也允许保存，但红色告警」——这里前端拦截明显非法 URL，模糊健康度交后端探测）。
 */
import { useEffect, useState } from 'react';
import type { Brand, DomainEntry } from '@/lib/types';
import { useSaveBrandDomains } from '@/hooks/queries';
import { validateDomains } from '@/lib/validation';
import { DomainEditor } from './DomainEditor';
import { Button } from './ui';

export function BrandDomainsCard({ brand }: { brand: Brand }) {
  const save = useSaveBrandDomains(brand.code);
  const [domains, setDomains] = useState<DomainEntry[]>(brand.domains);
  const [saved, setSaved] = useState(false);

  // 服务端数据变化时同步本地编辑态
  useEffect(() => {
    setDomains(brand.domains);
  }, [brand.domains]);

  const errors = validateDomains(domains);

  async function onSave() {
    await save.mutateAsync(domains.filter((d) => d.url.trim()));
    setSaved(true);
    setTimeout(() => setSaved(false), 2200);
  }

  return (
    <div className="section-card" style={{ borderLeft: `3px solid ${brand.accentColor}` }}>
      <h3 className="flex items-center gap-2 text-[13px] font-bold text-ink mb-[14px]">
        <span
          className="grid place-items-center w-[22px] h-[22px] rounded-md text-white text-[12px]"
          style={{ background: brand.accentColor }}
        >
          {brand.name[0]}
        </span>
        {brand.name} · 品牌默认域名
        <span className="font-normal text-muted text-[11.5px] ml-1.5">
          该品牌 {brand.channelCount} 个渠道默认继承
        </span>
        {brand.hmsEnabled && (
          <span className="ml-auto text-[10px] font-semibold px-2 py-0.5 rounded-full bg-[#dbeafe] text-[#1d4ed8]">
            HMS / OAID
          </span>
        )}
      </h3>

      <DomainEditor domains={domains} onChange={setDomains} />

      {errors.length > 0 && (
        <div className="mb-2 text-[12px] text-down">
          {errors.map((e, i) => (
            <div key={i}>• {e}</div>
          ))}
        </div>
      )}

      <div className="flex gap-[10px] mt-2 items-center">
        <Button
          className="text-[12px]"
          onClick={() =>
            setDomains((d) => {
              const used = new Set(d.map((x) => x.position));
              let pos = 1;
              while (used.has(pos) && pos <= 3) pos++;
              if (pos > 3) return d;
              return [...d, { position: pos, url: '', enabled: true, health: 'unconfigured' }];
            })
          }
        >
          + 添加备用域名
        </Button>
        <Button
          variant="primary"
          className="text-[12px]"
          disabled={save.isPending || errors.length > 0}
          onClick={onSave}
        >
          {save.isPending ? '下发中…' : saved ? '✓ 已下发' : '保存并下发'}
        </Button>
        {saved && (
          <span className="text-[12px] text-muted">已生成 CDN 配置快照，APK 下次启动生效</span>
        )}
      </div>
    </div>
  );
}
