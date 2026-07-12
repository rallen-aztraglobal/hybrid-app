/**
 * Adjust 事件 CSV 解析 —— docs/admin/08-adjust.md §6 / ADR-0013。
 *
 * Adjust 面板导出的事件 CSV 形如：
 *   token,name,unique
 *   "jrilpe","AddToCart","false"
 *   "wzb3fb","Login","false"
 *
 * 前端把它解析成 `{ name: token }`（`unique` 列丢弃，那是 Adjust 面板侧去重设置，
 * SDK 调用不需要），随渠道保存接口一起提交为 `adjust_events`。server 只存不解析
 * （见 CLAUDE.md 跨层契约）。
 *
 * 解析健壮性要求：
 *  - 字段带引号（`"jrilpe"`）与不带引号都要支持；
 *  - 引号内的转义 `""` 要还原成 `"`；
 *  - 空行 / 结尾多余换行要跳过；
 *  - 表头 token/name 列顺序不固定时按表头名定位，而非固定按位置取值；
 *  - 无法识别表头时按约定顺序 `token,name,unique` 兜底。
 */

export interface AdjustCsvRow {
  name: string;
  token: string;
}

export interface AdjustCsvParseResult {
  /** { 事件 name: token }，供 adjust_events 直接使用。 */
  events: Record<string, string>;
  /** 解析出的行（按出现顺序，去重取最后一次），供预览表格使用。 */
  rows: AdjustCsvRow[];
  /** 非致命问题提示（跳过的行 / 重复事件名 / 未识别表头等），不阻断解析。 */
  warnings: string[];
}

/**
 * 拆分单行 CSV（支持双引号包裹字段、引号内逗号、`""` 转义）。
 * 不支持字段内换行（Adjust 导出的事件 CSV 不会出现该情形）。
 */
function splitCsvLine(line: string): string[] {
  const out: string[] = [];
  let cur = '';
  let inQuotes = false;
  for (let i = 0; i < line.length; i++) {
    const ch = line[i];
    if (inQuotes) {
      if (ch === '"') {
        if (line[i + 1] === '"') {
          cur += '"';
          i++;
        } else {
          inQuotes = false;
        }
      } else {
        cur += ch;
      }
    } else if (ch === '"') {
      inQuotes = true;
    } else if (ch === ',') {
      out.push(cur);
      cur = '';
    } else {
      cur += ch;
    }
  }
  out.push(cur);
  return out.map((s) => s.trim());
}

/** 解析 Adjust 导出的事件 CSV（token,name,unique）为 { name: token }。 */
export function parseAdjustEventsCsv(text: string): AdjustCsvParseResult {
  const warnings: string[] = [];
  const lines = text
    .split(/\r\n|\r|\n/)
    .map((l) => l.trim())
    .filter((l) => l.length > 0);

  if (lines.length === 0) {
    return { events: {}, rows: [], warnings: ['CSV 内容为空'] };
  }

  const header = splitCsvLine(lines[0]).map((h) => h.toLowerCase());
  let tokenIdx = header.indexOf('token');
  let nameIdx = header.indexOf('name');
  let dataLines: string[];

  if (tokenIdx === -1 || nameIdx === -1) {
    // 未识别到表头（或字段名对不上）：按约定列顺序 token,name,unique 兜底，
    // 把第一行也当数据行解析。
    warnings.push('未识别到 token/name 表头，按默认列顺序 token,name,unique 解析');
    tokenIdx = 0;
    nameIdx = 1;
    dataLines = lines;
  } else {
    dataLines = lines.slice(1);
  }

  const events: Record<string, string> = {};
  const rows: AdjustCsvRow[] = [];

  dataLines.forEach((line, i) => {
    const cols = splitCsvLine(line);
    const token = (cols[tokenIdx] ?? '').trim();
    const name = (cols[nameIdx] ?? '').trim();
    if (!token || !name) {
      warnings.push(`第 ${i + 2} 行缺少 token 或 name，已跳过：${line}`);
      return;
    }
    if (Object.prototype.hasOwnProperty.call(events, name) && events[name] !== token) {
      warnings.push(`事件「${name}」重复出现，已用最后一次的 token 覆盖`);
    }
    events[name] = token;
    const existingIdx = rows.findIndex((r) => r.name === name);
    if (existingIdx >= 0) rows[existingIdx] = { name, token };
    else rows.push({ name, token });
  });

  if (dataLines.length > 0 && rows.length === 0) {
    warnings.push('未解析出任何有效事件，请检查 CSV 格式');
  }

  return { events, rows, warnings };
}
