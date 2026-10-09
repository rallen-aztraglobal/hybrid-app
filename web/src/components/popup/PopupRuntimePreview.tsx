/**
 * 运行时预览（GET /api/popups/runtime-preview）：运营核对「这个包（+国家）现在会拿到什么」。
 * payload 与 App 端 /api/app/popups 完全一致，原样展示 JSON。
 */
import { useState } from 'react';
import { useChannels, usePopupRuntimePreview } from '@/hooks/queries';
import { normalizeCountry } from '@/lib/popupMeta';
import { Button } from '../ui';
import { SearchSelect } from '../SearchSelect';
import { EyeIcon } from '../icons';

export function PopupRuntimePreview() {
  const { data: channels } = useChannels();
  const [appId, setAppId] = useState('');
  const [country, setCountry] = useState('');
  // 只在点「查询」时提交参数；改 appId / 国家本身不触发重查
  const [submitted, setSubmitted] = useState({ appId: '', country: '', nonce: 0 });
  const { data, isFetching, error } = usePopupRuntimePreview(submitted.appId, submitted.country, submitted.nonce);

  return (
    <details className="mt-4 rounded-card border border-line bg-panel shadow-sm2 px-4 py-3">
      <summary className="cursor-pointer font-bold text-[13px] flex items-center gap-2 select-none">
        <EyeIcon className="w-[15px] h-[15px] inline" /> 运行时预览 · 这个包现在会拿到什么
      </summary>
      <div className="flex items-end gap-3 flex-wrap mt-3">
        <div className="w-[300px]">
          <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">渠道包</label>
          <SearchSelect
            value={appId}
            onChange={setAppId}
            placeholder="选择渠道包（applicationId）"
            options={(channels ?? []).filter((c) => c.status !== 'archived').map((c) => ({ value: c.applicationId, label: c.appName, sub: c.applicationId }))}
          />
        </div>
        <div className="w-[120px]">
          <label className="block text-[12.5px] font-semibold text-ink-2 mb-[6px]">国家码</label>
          <input className="field-input font-mono uppercase" maxLength={2} placeholder="PH" value={country} onChange={(e) => setCountry(e.target.value)} />
        </div>
        <Button variant="primary" disabled={!appId || isFetching} onClick={() => setSubmitted((s) => ({ appId, country: normalizeCountry(country) ?? '', nonce: s.nonce + 1 }))}>
          {isFetching ? '查询中…' : '查询'}
        </Button>
        <span className="text-[11.5px] text-muted pb-2">国家码留空 = 不带地区（设了地区定向的弹窗不命中）</span>
      </div>
      {error && <div className="mt-3 text-[12.5px] text-down">{error instanceof Error ? error.message : '查询失败'}</div>}
      {data && (
        <div className="mt-3">
          <div className="text-[12px] text-ink-2 mb-2">
            共 <b>{data.popups.length}</b> 个弹窗下发 · configVersion <span className="font-mono">{data.configVersion}</span>
          </div>
          <pre className="font-mono text-[11.5px] bg-[#0c1426] text-[#cbd5e1] rounded-xl p-3 max-h-[320px] overflow-auto">{JSON.stringify(data, null, 2)}</pre>
        </div>
      )}
    </details>
  );
}
