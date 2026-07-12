# Adjust 前端内部接口参考（无需付费 Automation）

Adjust Suite 是 SPA，前端调 `https://api.adjust.com/dashboard/api/*`。**带浏览器登录态即可直接调，无 CSRF**。
本 skill 通过 chrome-devtools MCP 在 `suite.adjust.com` 页面里用 `fetch(url, {credentials:'include'})` replay 这些请求。

> 这些是**未公开的内部接口**，Adjust 可能随时变更。若某请求突然 4xx，用 `initScript` 注入 fetch 拦截钩子、在 UI 里手动做一次同类操作、读 `window.__cap` 拿到最新请求形状（见下「重新抓包」）。

## 鉴权自检
```
GET https://api.adjust.com/dashboard/api/accounts        → 200 表示已登录
GET https://api.adjust.com/dashboard/api/apps?ctv=false  → {apps:[...]}
```
401/403 = Chrome 掉登录，让用户在该 Chrome 里重新登录 suite.adjust.com。

## 1. 列出所有 app（查重/复用/读平台状态）
```
GET https://api.adjust.com/dashboard/api/apps?page[per]=500&page[total]=true
```
（URL 编码：`page%5Bper%5D=500&page%5Btotal%5D=true`）
返回 `{apps:[{ id, name, token, app_token, default_store_app_id,
  platforms:{android:{configured,store,app_id}, ios:{configured}, ...}, currency:{iso_code} }]}`
- `token`（= `app_token`）就是 SDK 用的 **App Token**（12 位），也是 app 详情页 URL 里的标识码。
- 按 `name`（= 渠道 flavor）查重。

## 2. 建 app
```
POST https://api.adjust.com/dashboard/api/apps
{ "name": "<flavor>", "reporting_currency": "PHP", "no_eea_users": true }
```
- ⚠️ 字段是 `reporting_currency`（不是 `currency`），值取 `GET /dashboard/api/reporting_currencies` 里 `currencies[].name`（如 `PHP`）。
- 返回含新 app 的 `token`。

## 3. 配 Android 平台
```
PATCH https://api.adjust.com/dashboard/api/apps/<token>/platform_settings
{ "default_platform":"android",
  "platforms":{
    "android":{"configured":true,"store":"google","platform":"android","app_id":"<applicationId>",
      "redirect_url":null,"app_scheme":null,"fcm_key":null,"android_links_enabled":false,
      "android_app_links_sha256_cert_fingerprints":[],"fcm_credentials_id":null},
    "ios":{"configure":false},"windows":{"configure":false},"windows-phone":{"configure":false},
    "android-tv":{"configure":false},"apple-tv":{"configure":false},"fire-tv":{"configure":false},
    "roku-os":{"configure":false},"smart-cast":{"configure":false},"tizen":{"configure":false},
    "webos":{"configure":false},"web":{"configure":false},"playstation":{"configure":false},
    "xbox":{"configure":false},"nintendo":{"configure":false} } }
```
- `store:"google"` 用于全部（华为 `_hw` 包也用 google；商店只影响跳转链接，不影响 SDK 归因）。
- `app_id` = 渠道 `applicationId`。

## 4. 建事件（等价 UI「导入事件」，不依赖模板）
```
POST https://api.adjust.com/dashboard/api/event_types
{ "app_tokens":["<token>"],
  "events":[ {"name":"AddToCart","unique":false}, {"name":"CompleteRegistration","unique":false},
             {"name":"Login","unique":false}, {"name":"OldRegPurchase","unique":false},
             {"name":"Purchase","unique":false}, {"name":"TPFirstDeposit","unique":false} ] }
```
- 幂等做法：先 GET 现有事件名，只 POST 缺失的，避免重复。

## 5. 读回事件 token
```
GET https://api.adjust.com/dashboard/api/apps/<token>/event_types
→ { events:[ {id, name, token, unique, ...} ] }
```
`name→token` 即回填 Console `adjustEvents` 的内容。

## 固定约定
- **6 个事件名（写死，须与 APK `AdjustBootstrap.LOGICAL_TO_ADJUST_NAME` 对齐）**：
  `AddToCart` / `CompleteRegistration` / `Login` / `OldRegPurchase` / `Purchase` / `TPFirstDeposit`。
  （APK 侧 `af_login→Login`、`af_complete_registration→CompleteRegistration`，其余同名。）
- app 名 = flavor；包名 = applicationId；币种 PHP；`no_eea_users=true`；商店 google。
- 已有一个模板 app「Hybrid Template（自动化创建模版 勿删）」（本套 6 事件的来源）；纯 API 流程不需要它，但 UI「导入事件→从已有应用」会用到它，勿删。

## 重新抓包（接口变更时）
用 `navigate_page` 的 `initScript` 注入：
```js
(() => { window.__cap=[]; const of=window.fetch; window.fetch=function(){ try{
  const a0=arguments[0], url=(a0&&a0.url)||a0, opt=arguments[1]||{}, m=(opt.method||(a0&&a0.method)||'GET');
  if(String(url).includes('api.adjust.com') && m!=='GET') window.__cap.push({url:String(url),method:m,body:typeof opt.body==='string'?opt.body:JSON.stringify(opt.body)});
}catch(e){} return of.apply(this,arguments); }; })();
```
然后在 UI 里手动做一次该操作，`evaluate_script` 读 `window.__cap` 即得真实请求（URL/method/body）。
