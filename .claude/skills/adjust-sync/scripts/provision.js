// Adjust 批量建 app —— 在 chrome-devtools MCP 的 evaluate_script 里执行。
// 前置：Chrome 已登录 suite.adjust.com（此函数用 fetch(credentials:'include') 借登录态调内部接口，无 CSRF）。
//
// 用法：把本函数整体贴进 evaluate_script 的 function 参数，先把下面 BATCH 换成本批要处理的
// [[flavor, applicationId], ...]（每批 ≤14，避免超时）。返回每个渠道的
// {flavor, appId, appToken, events:{name:token}, steps, error}。幂等：已存在的 app/平台/事件自动跳过。
//
// 每批返回后立刻落盘 scratchpad/adjust_results_<n>.json。全部批次跑完后用文末「合并/校验」片段汇总。

async () => {
  const OUR = ['AddToCart', 'CompleteRegistration', 'Login', 'OldRegPurchase', 'Purchase', 'TPFirstDeposit'];

  // ↓↓↓ 每批把这里换成本批的 [flavor, applicationId] 列表（≤14）↓↓↓
  const BATCH = [
    // ["ap01159", "com.arenaplus.ap01159"],
  ];
  // ↑↑↑ ↑↑↑

  const J = (u, m, b) => fetch(u, {
    method: m || 'GET', credentials: 'include',
    headers: (m && m !== 'GET') ? { 'Content-Type': 'application/json' } : undefined,
    body: b ? JSON.stringify(b) : undefined,
  });

  // 一次性取现有 apps，按名建索引（查重/复用）
  const lj = await (await J('https://api.adjust.com/dashboard/api/apps?page%5Bper%5D=500&page%5Btotal%5D=true')).json();
  const byName = {};
  (lj.apps || []).forEach(a => { byName[a.name] = a; });

  const out = [];
  for (const [flavor, appId] of BATCH) {
    const rec = { flavor, appId, appToken: null, events: {}, steps: [], error: null };
    try {
      // 1) app：不存在才建
      let app = byName[flavor], token;
      if (app) { token = app.token; rec.steps.push('exists'); }
      else {
        const cr = await J('https://api.adjust.com/dashboard/api/apps', 'POST',
          { name: flavor, reporting_currency: 'PHP', no_eea_users: true });
        if (!cr.ok) { rec.error = 'create ' + cr.status + ' ' + (await cr.text()).slice(0, 100); out.push(rec); continue; }
        const cj = await cr.json(); token = cj.token || (cj.app && cj.app.token); rec.steps.push('created');
      }
      rec.appToken = token;

      // 2) Android 平台：未配置才配（商店统一 google）
      const hasAndroid = app && app.platforms && app.platforms.android && app.platforms.android.configured;
      if (!hasAndroid) {
        const pb = { default_platform: 'android', platforms: {
          android: { configured: true, store: 'google', platform: 'android', app_id: appId,
            redirect_url: null, app_scheme: null, fcm_key: null, android_links_enabled: false,
            android_app_links_sha256_cert_fingerprints: [], fcm_credentials_id: null },
          ios: { configure: false }, windows: { configure: false }, 'windows-phone': { configure: false },
          'android-tv': { configure: false }, 'apple-tv': { configure: false }, 'fire-tv': { configure: false },
          'roku-os': { configure: false }, 'smart-cast': { configure: false }, tizen: { configure: false },
          webos: { configure: false }, web: { configure: false }, playstation: { configure: false },
          xbox: { configure: false }, nintendo: { configure: false } } };
        const pr = await J('https://api.adjust.com/dashboard/api/apps/' + token + '/platform_settings', 'PATCH', pb);
        rec.steps.push('plat ' + pr.status);
      } else rec.steps.push('plat skip');

      // 3) 事件：读现有，只补缺的
      let ej = (await (await J('https://api.adjust.com/dashboard/api/apps/' + token + '/event_types')).json());
      const existing = (ej.events || []).map(e => e.name);
      const missing = OUR.filter(n => !existing.includes(n));
      if (missing.length) {
        const mr = await J('https://api.adjust.com/dashboard/api/event_types', 'POST',
          { app_tokens: [token], events: missing.map(n => ({ name: n, unique: false })) });
        rec.steps.push('ev+' + missing.length + ' ' + mr.status);
        ej = (await (await J('https://api.adjust.com/dashboard/api/apps/' + token + '/event_types')).json());
      } else rec.steps.push('ev skip');

      // 4) 读回 name→token
      (ej.events || []).forEach(e => { if (OUR.includes(e.name)) rec.events[e.name] = e.token; });
    } catch (e) { rec.error = String(e); }
    out.push(rec);
  }
  return out;
}

/* ── 全部批次跑完后，本地 python 合并 + 校验（生成 adjust_final.json 供回填）──────────
import json, glob
SC = "<scratchpad>"
merged = {}
for f in sorted(glob.glob(SC + "/adjust_results_*.json")):
    for r in json.load(open(f)): merged[r["flavor"]] = r
chans = json.load(open(SC + "/adjust_channels.json"))["channels"]
EV = ["AddToCart","CompleteRegistration","Login","OldRegPurchase","Purchase","TPFirstDeposit"]
final, issues, seen = [], [], {}
for c in chans:
    r = merged.get(c["flavor"])
    if not r: continue                      # 不在本次增量内，跳过
    if not r.get("appToken"): issues.append(c["flavor"] + " 无 appToken")
    miss = [e for e in EV if e not in r.get("events", {})]
    if miss: issues.append(f"{c['flavor']} 缺事件 {miss}")
    if r["appToken"] in seen: issues.append(f"appToken 撞车 {c['flavor']} vs {seen[r['appToken']]}")
    seen[r["appToken"]] = c["flavor"]
    final.append({"channelId": c["id"], "flavor": c["flavor"],
                  "adjustAppToken": r["appToken"], "adjustEvents": r["events"]})
json.dump(final, open(SC + "/adjust_final.json", "w"), ensure_ascii=False, indent=2)
print(f"{len(final)} 条；唯一 token {len(seen)}；问题:", issues or "无")
─────────────────────────────────────────────────────────────────────────────────── */
