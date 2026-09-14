/**
 * 单个品牌的默认域名卡片（域名配置页用）。可编辑 + 保存下发。
 * 保存前做 §5.7 校验（主域名必填、https、去重）；不通过给红字但仍可保存（与原型/文档一致：
 * 「不通也允许保存，但红色告警」——这里前端拦截明显非法 URL，模糊健康度交后端探测）。
 */
import { useEffect, useState } from 'react';
import type { Brand, DomainEntry } from '@/lib/types';
import { useSaveBrandDomains, useUpdateBrandAdjust } from '@/hooks/queries';
import { useAuthStore } from '@/store/authStore';
import { PERM } from '@/lib/permissions';
import { friendlyNotFoundMessage } from '@/lib/api';
import { normalizeAdjustDeepLinkHost, validateDomains } from '@/lib/validation';
import { DomainEditor } from './DomainEditor';
import { Button } from './ui';

export function BrandDomainsCard({ brand }: { brand: Brand }) {
  const save = useSaveBrandDomains(brand.code);
  const canEdit = useAuthStore((s) => s.hasPerm(PERM.DOMAIN_EDIT));
  const [domains, setDomains] = useState<DomainEntry[]>(brand.domains);
  const [saved, setSaved] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  // 服务端数据变化时同步本地编辑态
  useEffect(() => {
    setDomains(brand.domains);
  }, [brand.domains]);

  const errors = validateDomains(domains);

  async function onSave() {
    setSaveError(null);
    try {
      await save.mutateAsync(domains.filter((d) => d.url.trim()));
      setSaved(true);
      setTimeout(() => setSaved(false), 2200);
    } catch (err) {
      // 品牌越界后端故意返回 404（10-rbac.md「数据权限」）：不能照搬「不存在」类文案。
      setSaveError(friendlyNotFoundMessage(err, '保存失败'));
    }
  }

  return (
    <div className="section-card" style={{ borderLeft: `3px solid ${brand.accentColor}` }}>
      <div className="flex items-start gap-2 mb-[14px]">
        <h3 className="flex items-center gap-2 text-[13px] font-bold text-ink">
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
        </h3>
        <div className="ml-auto flex items-center gap-2">
          {brand.hmsEnabled && (
            <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-[#dbeafe] text-[#1d4ed8]">
              HMS / OAID
            </span>
          )}
          {canEdit && (
            <Button
              variant="primary"
              className="text-[12px]"
              disabled={save.isPending || errors.length > 0}
              onClick={onSave}
            >
              {save.isPending ? '下发中…' : saved ? '✓ 已下发' : '保存并下发'}
            </Button>
          )}
        </div>
      </div>

      <DomainEditor domains={domains} onChange={setDomains} disabled={!canEdit} />

      {errors.length > 0 && (
        <div className="mt-2 text-[12px] text-down">
          {errors.map((e, i) => (
            <div key={i}>• {e}</div>
          ))}
        </div>
      )}

      {saveError && <div className="mt-2 text-[12px] text-down">{saveError}</div>}

      {saved && (
        <div className="mt-2 text-[12px] text-muted">已生成 CDN 配置快照，APK 下次启动生效</div>
      )}

      {brand.code === 'bp' && <AdjustDeepLinkHostRow brand={brand} canEdit={canEdit} />}
    </div>
  );
}

/**
 * Adjust 短链 host（仅 bp 品牌，BP 原始事件方案的一部分）：只存 host，如
 * `link.bingoplus.com`，不含 `https://` 或路径。权限沿用本卡片的域名编辑权限判断。
 */
function AdjustDeepLinkHostRow({ brand, canEdit }: { brand: Brand; canEdit: boolean }) {
  const save = useUpdateBrandAdjust(brand.code);
  const [draft, setDraft] = useState(brand.adjustDeepLinkHost ?? '');
  const [saved, setSaved] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  useEffect(() => {
    setDraft(brand.adjustDeepLinkHost ?? '');
  }, [brand.adjustDeepLinkHost]);

  const { value: normalized, error: formatError } = normalizeAdjustDeepLinkHost(draft);

  async function onSave() {
    if (formatError) return;
    setSaveError(null);
    try {
      await save.mutateAsync(normalized);
      setSaved(true);
      setTimeout(() => setSaved(false), 2200);
    } catch (err) {
      setSaveError(friendlyNotFoundMessage(err, '保存失败'));
    }
  }

  return (
    <div className="mt-[14px] pt-[13px] border-t border-line-2">
      <div className="text-[12.5px] font-semibold text-ink-2 mb-[6px]">
        Adjust 短链 host{' '}
        <span className="font-normal text-muted text-[11.5px]">
          · 仅 host，如 link.bingoplus.com，不含 https:// 或路径；BP 原始事件方案用
        </span>
      </div>
      <div className="flex items-center gap-2">
        <input
          className="field-input mono flex-1"
          placeholder="例如 link.bingoplus.com（留空 = 未配置）"
          value={draft}
          disabled={!canEdit}
          onChange={(e) => setDraft(e.target.value)}
        />
        {canEdit && (
          <Button
            variant="primary"
            className="text-[12px] flex-none"
            disabled={save.isPending || !!formatError}
            onClick={onSave}
          >
            {save.isPending ? '保存中…' : saved ? '✓ 已保存' : '保存'}
          </Button>
        )}
      </div>
      {formatError && <div className="mt-1 text-[12px] text-down">{formatError}</div>}
      {saveError && <div className="mt-1 text-[12px] text-down">{saveError}</div>}
      {!canEdit && (
        <div className="mt-1 text-[11.5px] text-muted">当前账号无域名编辑权限（domain:edit），此区块为只读展示。</div>
      )}
    </div>
  );
}
