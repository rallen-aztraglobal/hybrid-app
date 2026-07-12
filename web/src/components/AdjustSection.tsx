/**
 * Adjust 归因区块 —— 渠道编辑抽屉「Adjust」小节（docs/admin/08-adjust.md §6 / ADR-0013）。
 *
 * 两个控件即可（对齐设计文档「有这两个就够」）：
 *  1) App Token 输入框：空 = 该渠道不启用 Adjust，打包时跳过集成、不发任何事件；
 *  2) 事件 CSV 上传：解析 Adjust 面板导出的 `token,name,unique`（首行表头、字段带引号）
 *     成 `{ name: token }`（`unique` 列丢弃），预览成 name/token 表格供核对，
 *     覆盖式保存进 adjust_events（重新上传即覆盖旧解析结果）。
 *
 * CSV 解析函数见 @/lib/adjustCsv（parseAdjustEventsCsv，含单测）。
 */
import { useRef, useState } from 'react';
import { parseAdjustEventsCsv } from '@/lib/adjustCsv';
import { cn } from '@/lib/cn';
import { InfoIcon, TrashIcon, UploadIcon } from './icons';
import { Note } from './ui';
import { Field } from './FormField';

export function AdjustSection({
  appToken,
  onAppTokenChange,
  events,
  onEventsChange,
}: {
  appToken: string;
  onAppTokenChange: (v: string) => void;
  events: Record<string, string>;
  onEventsChange: (v: Record<string, string>) => void;
}) {
  const fileRef = useRef<HTMLInputElement>(null);
  const [fileName, setFileName] = useState<string | null>(null);
  const [warnings, setWarnings] = useState<string[]>([]);
  const [dragOver, setDragOver] = useState(false);

  const rows = Object.entries(events)
    .map(([name, token]) => ({ name, token }))
    .sort((a, b) => a.name.localeCompare(b.name));

  async function handleFile(file: File) {
    const text = await file.text();
    const result = parseAdjustEventsCsv(text);
    setFileName(file.name);
    setWarnings(result.warnings);
    onEventsChange(result.events);
  }

  function clearEvents() {
    setFileName(null);
    setWarnings([]);
    onEventsChange({});
  }

  return (
    <div>
      <Field
        label="App Token"
        hint="Adjust 面板「App Token」；留空即该渠道不启用 Adjust（打包时跳过集成）"
      >
        <input
          className="field-input mono"
          placeholder="例如 abc123xyz（留空 = 不启用）"
          value={appToken}
          onChange={(e) => onAppTokenChange(e.target.value)}
        />
      </Field>

      <div className="mb-[6px] text-[12.5px] font-semibold text-ink-2">
        事件列表{' '}
        <span className="font-normal text-muted text-[11.5px]">
          · 上传 Adjust 导出的事件 CSV（token,name,unique），unique 列会被丢弃
        </span>
      </div>

      <div
        onClick={() => fileRef.current?.click()}
        onDragOver={(e) => {
          e.preventDefault();
          setDragOver(true);
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={(e) => {
          e.preventDefault();
          setDragOver(false);
          const f = e.dataTransfer.files?.[0];
          if (f) void handleFile(f);
        }}
        className={cn(
          'flex items-center gap-3 rounded-[12px] p-4 cursor-pointer border-2 border-dashed transition',
          dragOver ? 'border-brand' : 'border-[#c7d2e6] hover:border-brand',
        )}
        style={{
          background: 'repeating-linear-gradient(45deg,#fbfcfe,#fbfcfe 12px,#f6f8fc 12px,#f6f8fc 24px)',
        }}
      >
        <UploadIcon className="w-5 h-5 flex-none text-ink-2" />
        <div className="flex-1 min-w-0">
          <div className="text-[13px] font-bold">
            {fileName ? `已解析：${fileName}` : '拖入或点击选择事件 CSV'}
          </div>
          <div className="text-[12px] text-muted mt-0.5">
            {rows.length > 0
              ? `共 ${rows.length} 个事件；重新上传将覆盖当前解析结果`
              : '格式 token,name,unique，首行表头'}
          </div>
        </div>
        {rows.length > 0 && (
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              clearEvents();
            }}
            className="flex-none grid place-items-center w-8 h-8 rounded-[8px] border border-line text-ink-2 hover:bg-bg"
            title="清空已解析事件"
          >
            <TrashIcon className="w-4 h-4" />
          </button>
        )}
        <input
          ref={fileRef}
          type="file"
          accept=".csv,text/csv"
          hidden
          onChange={(e) => {
            const f = e.target.files?.[0];
            e.target.value = '';
            if (f) void handleFile(f);
          }}
        />
      </div>

      {warnings.length > 0 && (
        <div className="mt-2 text-[12px] text-down leading-[1.6]">
          {warnings.map((w, i) => (
            <div key={i}>· {w}</div>
          ))}
        </div>
      )}

      {rows.length > 0 && (
        <div className="mt-3 rounded-[10px] border border-line overflow-hidden">
          <table className="w-full text-[12.5px]">
            <thead>
              <tr className="bg-panel-2 text-ink-2">
                <th className="text-left font-semibold px-3 py-2">事件名 name</th>
                <th className="text-left font-semibold px-3 py-2">Token</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.name} className="border-t border-line-2">
                  <td className="px-3 py-[7px]">{r.name}</td>
                  <td className="px-3 py-[7px] mono text-ink-2">{r.token}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {appToken.trim() && rows.length === 0 && (
        <div className="mt-3">
          <Note>
            <InfoIcon className="w-[17px] h-[17px] flex-none mt-0.5" style={{ color: 'var(--brand)' }} />
            <div>
              已填 App Token 但未上传事件 CSV：打包后该渠道仍会集成 Adjust（install/session 由 SDK
              自动归因），但不会有自定义事件上报。可稍后补传 CSV 再重新保存。
            </div>
          </Note>
        </div>
      )}
    </div>
  );
}
