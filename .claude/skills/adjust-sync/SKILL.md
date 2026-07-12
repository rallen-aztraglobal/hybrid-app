---
name: adjust-sync
description: 给渠道包批量新增/同步 Adjust 归因（建 app + Android 平台 + 6 个事件），再把 app token 与事件 token 回填到线上 Console。用于"新增了小渠道后补 Adjust""同步 Adjust""给某某渠道建 Adjust app""补 adjust token 到线上"等场景。走 Adjust 前端内部接口（无需付费 Automation），幂等可重跑。
---

# Adjust 归因批量同步

给「渠道中台」里的小渠道包在 Adjust 后台批量建 app（含 Android 平台 + 6 个转化事件），并把生成的 **app token + 事件 token** 回填到线上 Console 的 `adjustAppToken` / `adjustEvents` 字段。方案背景见 [docs/admin/08-adjust.md](../../../docs/admin/08-adjust.md) 与 [ADR-0013](../../../docs/adr/0013-adjust-attribution.md)。

## 什么时候用

- 新增了一批小渠道（`channels/*.csv` / Console 加了渠道），要给它们补上 Adjust。
- 想把线上所有 **未绑定 Adjust** 的 AP/BP 渠道一次性同步齐。
- 只想给某几个指定渠道建 Adjust app。

> 只有 **AP / BP** 需要 Adjust（GP 不接）。默认处理线上 `adjustAppToken` 为空的 AP/BP 渠道（增量），已绑定的自动跳过。

## 核心事实（务必遵守）

- **不用付费 Automation**：Adjust 前端 SPA 调的是内部接口 `https://api.adjust.com/dashboard/api/*`，**带浏览器登录态即可直接调、无 CSRF**。所以本 skill 用 chrome-devtools MCP 的 `evaluate_script` 里 `fetch(..., {credentials:'include'})` 来 replay，而不是逐个点 UI。接口细节见 [references/internal-api.md](references/internal-api.md)。
- **Adjust app 命名**：app 名 = 渠道 `flavor`（唯一，如 `ap01159` / `bpom3410`）；包名 = `applicationId`；币种 `PHP`；`no_eea_users=true`；商店 `google`（华为 `_hw` 包也统一用 google，商店只影响跳转链接、不影响 SDK 归因）。
- **6 个事件（名字写死，必须与 App 端一致）**：`AddToCart` / `CompleteRegistration` / `Login` / `OldRegPurchase` / `Purchase` / `TPFirstDeposit`。这组名字要与 APK 里 `AdjustBootstrap.LOGICAL_TO_ADJUST_NAME`（`af_login→Login`、`af_complete_registration→CompleteRegistration`，其余同名）严格对齐；改动事件集时两边一起改。
- **幂等**：脚本按渠道检查——app 不存在才建、平台没配才配、缺哪个事件补哪个、最后统一读回 token。可安全重跑、断点续跑。
- **回填前必须已发布带 Adjust 字段的新代码**：旧 Console 不认 `adjustAppToken/adjustEvents`（PUT 返回 200 但静默丢弃）。先确认线上 `GET /api/channels/:id` 返回里**有** `adjustAppToken` 键，再回填；没有就先跑 `release` skill 发布。

## 前置条件

1. **Chrome 已登录 Adjust Suite**：本机 chrome-devtools MCP 控制的 Chrome 里已登录 `suite.adjust.com`（`GET https://api.adjust.com/dashboard/api/accounts` 返回 200 即可）。没登录就 `new_page("https://suite.adjust.com")` 让用户登录。
2. **Console 管理员凭据**：`ADJUST_CONSOLE_URL`（如 `https://fortunegems-jackpot.online`）、`ADJUST_CONSOLE_USER`、`ADJUST_CONSOLE_PASS`。作为环境变量传给脚本，别写进仓库。
3. Python3、curl 可用；scratchpad 目录用于存中间结果。

## 步骤

设 `SC` = 本会话 scratchpad 目录。

### 1. 拉增量渠道清单（Console → 本地）

```bash
ADJUST_CONSOLE_URL=https://fortunegems-jackpot.online \
ADJUST_CONSOLE_USER=admin ADJUST_CONSOLE_PASS='***' \
python3 .claude/skills/adjust-sync/scripts/pull_channels.py "$SC/adjust_channels.json"
```

输出 `$SC/adjust_channels.json`：全部 AP+BP 渠道（含 `id/flavor/applicationId/palCode/appName/adjustAppToken`）+ 打印「未绑定（增量）」清单。
- 只想处理指定渠道：手工把 `delta` 里筛成你要的那几个 `[flavor, applicationId]`。

### 2. 在 Adjust 批量建 app（浏览器内部接口，幂等）

把第 1 步的**增量** `[[flavor, applicationId], ...]` 数组，**分批（每批 ≤14）** 内联进 [scripts/provision.js](scripts/provision.js) 里 `const BATCH = [...]`，用 `evaluate_script` 逐批执行。每批返回每个渠道的 `{flavor, appToken, events:{name:token}, steps}`。
- 先确认 Adjust 登录态：`evaluate_script` 跑一次 `fetch('https://api.adjust.com/dashboard/api/apps?ctv=false',{credentials:'include'}).then(r=>r.status)` 应为 200。
- 每批结果**立即落盘** `$SC/adjust_results_<n>.json`（防上下文压缩/中断丢失）。
- 全部跑完后合并去重、校验：每个渠道都要有 `appToken` + 齐 6 个事件、app token 互不相同。见 provision.js 末尾的合并/校验片段。

### 3. 回填线上 Console（本地 → Console）

先确认线上已支持字段（见「核心事实」最后一条）。然后：

```bash
ADJUST_CONSOLE_URL=... ADJUST_CONSOLE_USER=... ADJUST_CONSOLE_PASS='***' \
python3 .claude/skills/adjust-sync/scripts/fill_console.py \
  "$SC/adjust_channels.json" "$SC/adjust_final.json"
```

`adjust_final.json` = 第 2 步合并后的结果（含 `channelId`）。脚本对每个渠道 `PUT /api/channels/:id`（带 `palCode/appName/adjustAppToken/adjustEvents`），再 GET 回读校验 `adjustAppToken` 已持久化。

### 4. 触发重新打包（可选）

回填只改后台数据；要让**已有渠道**产出带新 token 的 APK，需在 Console「打包中心」或 `POST /api/build/jobs` 触发构建，CLI 会把 `adjustAppToken/adjustEvents` 渲染进 `app/adjust-tokens.json`（见 08-adjust.md §3/§5）。

## 排错

- `evaluate_script` fetch 返回 401/403：Chrome 掉登录了 → 让用户重新登录 Adjust Suite。
- 建 app 报 `Reporting currency is not included in the list`：字段名必须是 `reporting_currency`（不是 `currency`），值 `PHP`。
- 回填后回读仍为 `None`：线上还是旧代码 → 先 `release` 发布。
- 某渠道重复出现在 Adjust（重名 app）：provision.js 按 `flavor` 名查重、已存在则复用，不会重复建；若手工建过同名需先在 Adjust 删掉多余的。
- 一批太大导致 `evaluate_script` 超时：减小批量（≤10）。
