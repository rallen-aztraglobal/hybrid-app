# 07 · Firebase 推送（FCM）

> 需求：APK 集成 Firebase 推送；Console 编辑推送内容并「选渠道包批量发送」。决策见 [ADR-0012](../adr/0012-push-fcm.md)。

## 一句话方案

```
Console 编辑推送 ──→ Go 后端 (campaign + 设备token库)
   (标题/正文/图/path)          │
   选渠道包(applicationId集)     │ FCM HTTP v1 REST (oauth2/google, 按品牌3把私钥)
   即时 / 定时(cron)            ▼
                          已装机 APK (FirebaseMessagingService)
                          点击 → 带 path+palcode → DomainResolver 拼域名 → WebView
```

- **Firebase 项目**：按品牌 3 个（ap/bp/gp），每品牌一份合并 `google-services.json`（含该品牌全部 flavor 的 package_name）。
- **投递**：设备 Token 注册库（APK 上报 `(applicationId, token)`），按目标 applicationId 集合精确发送 + 统计。
- **发送**：FCM HTTP v1 REST，worker pool 逐 token，失效 token 自动下线。**不引 Admin SDK**。
- **机密分级**：`google-services.json` 非机密（随 APK 走，入 DB/API/CLI）；service account 私钥（3 把，发送用）是机密，仅服务端 env/挂载，照 [ADR-0008](../adr/0008-server-side-build.md)。

## 1. 数据模型（新增 4 表，migration `000003_push`）

```sql
-- 设备 token 注册库
push_device_token(
  id, application_id, brand_code, device_token UNIQUE, pal_code,
  platform, model_info, is_active, last_seen_at, created_at,
  KEY(application_id, is_active))

-- 推送活动（内容 + 目标 + 状态机）
push_campaign(
  id, name, title, body, image_url, deeplink_path, extra_data JSON,
  status,            -- draft|scheduled|sending|done|failed
  scheduled_at, sent_at,
  total_devices, success_count, failure_count,
  created_by, created_at, KEY(status, scheduled_at))

-- 活动目标渠道（多对多：一个活动覆盖多品牌多渠道）
push_campaign_target(id, campaign_id, application_id)

-- 发送结果（按 application_id 汇总，便于复盘/重试；不存逐 token 明细以控量）
push_record(id, campaign_id, application_id, sent, failed, error_sample, finished_at)
```

要点：
- `application_id` 关联 `channel.application_id`（ADR-0009 唯一标识），**不用 palcode 做键**；palcode 仅随 token 存下、随推送 URL 透传。
- `deeplink_path` 存**相对 path**（如 `/promo/618`），**绝不存域名**——APK 端用 DomainResolver 拼当前可用域名（守 ADR-0002）。
- GORM model 加入 `model.AllModels()`；开发期 SQLite AutoMigrate，生产 golang-migrate 跑 `000003_push.up.sql`。

## 2. 后端（server/）

新增：`internal/model`(+4 model)、`internal/repo/push.go`、`internal/service/push.go`、`internal/service/fcm.go`、`internal/handler/push.go`、路由、`config`、cron。

### 2.1 FCM 发送（HTTP v1 REST，无 Admin SDK）
- `internal/service/fcm.go`：按 brand 持有 3 个 `*google.Credentials`（从 `FIREBASE_SA_AP/BP/GP` 读 JSON），用 `oauth2/google` 换 access token（自动缓存/刷新）。
- 发送：`POST https://fcm.googleapis.com/v1/projects/{projectId}/messages:send`，body 为单条 message（`token` + `notification` + `data`）。
- worker pool（如并发 20）逐 token 发；响应 `UNREGISTERED`/`INVALID_ARGUMENT` → 标记 `push_device_token.is_active=false`；按 application_id 汇总写 `push_record`。

### 2.2 Service / Handler / 路由
```
# APK 公开端点（无 JWT，按 applicationId 校验渠道存在 + 轻量防滥用）
POST /api/app/push/register-token      {appId, token, palcode, platform, model}

# 管理面（operator+）
POST   /api/push/campaigns             创建草稿（图片先 upload 到对象存储）
PUT    /api/push/campaigns/:id         编辑草稿
POST   /api/push/campaigns/:id/send    立即发送
POST   /api/push/campaigns/:id/schedule {scheduledAt}  定时
POST   /api/push/upload-image          multipart → Storage（复用图标上传管线）

# 管理面（viewer+）
GET /api/push/campaigns                列表（带状态/统计）
GET /api/push/campaigns/:id            详情 + push_record
GET /api/push/audience?appIds=...      预估目标活跃设备数（发送前展示）
```
- 复用：JWT 中间件 + `RequireRole`、`Storage` 接口（图片）、统一 Envelope 响应、swag 注解生成 OpenAPI。

### 2.3 定时发送（cron）
- 复用 `cmd/server/main.go` 既有 `robfig/cron`：`@every 1m` 扫 `status=scheduled AND scheduled_at<=now` → 置 `sending` → 发送 → `done/failed`。
- 受 `PUSH_CRON_ENABLE` 开关控制。

### 2.4 配置（config.go 新增）
```
FIREBASE_SA_AP / FIREBASE_SA_BP / FIREBASE_SA_GP   # service account JSON 路径或内容（机密）
FIREBASE_PROJECT_AP / _BP / _GP                    # 三个 Firebase projectId
PUSH_ENABLED=true
PUSH_CRON_ENABLE=true
```

## 3. APK（app/）

> 守 ADR-0004：**不动 loadChannels/productFlavors**，只新增插件 + 依赖 + 一个 Service。

- `gradle/libs.versions.toml`：加 `firebase-bom`、`firebase-messaging`、`google-services` 插件。
- 根 `build.gradle`：`alias(libs.plugins.google.services) apply false`。
- `app/build.gradle`：apply `google-services` 插件 + `implementation platform(libs.firebase.bom)` + `firebase-messaging`。
- `app/google-services.json`：CLI/构建机按品牌投放（gitignore），含该品牌全部 package_name。
- `AndroidManifest.xml`：加 `POST_NOTIFICATIONS` 权限 + `<service>` FCM。
- 新增 `com.hybrid.android.push.HybridMessagingService : FirebaseMessagingService`：
  - `onNewToken` → POST `/api/app/push/register-token`（带 `BuildConfig.APPLICATION_ID` + `PAL_CODE`）。
  - `onMessageReceived` → 建通知；点击 Intent 进 `WebViewActivity`，携带 `deeplink_path`。
- `WebViewActivity`：收到 push path → 经 **DomainResolver** 解析当前域名后 `loadUrl("$domain$path?palcode=...")`（守 ADR-0002，域名不焊死）。
- 首次启动申请 `POST_NOTIFICATIONS`（Android 13+）+ 主动 `getToken()` 注册一次。
- 品牌差异：`BrandStrategy` 可加可选 `onPushOpen(path, host)`，BP/AP/GP 各自定制点击归因（AppsFlyer 事件）。

## 4. Console（web/）

- 导航「运营」组新增 `/push`「推送管理」。
- `PushPage`：`BrandTabs` + 子 tab（编辑 / 历史）。
- **编辑**：表单（标题/正文/图片上传/deeplink path/extra）+ 渠道多选（**直接复用 `PackPage` L111–171 全选/勾选 UI**）+ 发送方式（即时 / 定时 datetime）+ 发送前调 `/api/push/audience` 展示「预计触达 N 台设备」。
- **历史**：campaign 列表（封面/标题/渠道数/成功失败/状态/定时时间），行样式参考 `BuildsPage`。
- 新增 `lib/api.ts` 的 `pushApi`、`types.ts` 的 `Push/PushCampaign/PushInput`、`hooks` Query；图片走现有 `uploadDataUrl`。

## 5. CLI（cli/）

- `hybrid-pack pull`：除现有 CSV/res/bootstrap.json 外，额外拉取**当前品牌的 `google-services.json`** 落到 `app/google-services.json`。
- 后端提供 `GET /api/app/google-services?brand=ap`（或随构建配置下发）返回该品牌合并 json（非机密）。
- 构建机（build-runner）同样在构建前投放对应品牌 json。

## 6. 护栏对照（CLAUDE.md / ADR）

| 护栏 | 本方案如何守住 |
| --- | --- |
| 不改 Gradle flavor 逻辑（ADR-0004） | 只加 google-services 插件 + 依赖；`loadChannels`/`productFlavors` 一行不动；google-services.json 按品牌投放，构建期按 appId 自动匹配 |
| 域名编译期不焊死（ADR-0002） | 推送只带相对 path + palcode，APK 用 DomainResolver 拼当前域名 |
| 域名容灾不乱换（ADR-0003） | 推送链路不碰域名解析，原状机不受影响 |
| 机密不入 git/DB/API/前端（ADR-0008） | google-services.json 非机密可入库；service account 私钥仅服务端 env/挂载 |
| applicationId 唯一标识（ADR-0009） | token 库与目标选择均以 applicationId 为键；palcode 仅作 URL 参数 |

## 6b. 无 google-services.json 先行（feature gate）

尚未拿到 Firebase 账号时，**整套功能照常实现**，FCM 用开关门控，「在线但惰性」，拿到账号后**零改码**激活。

| 层 | 未配置时行为 | 激活方式 |
| --- | --- | --- |
| APK | `google-services` 插件**条件式 apply**（`if (file('app/google-services.json').exists())`）→ 无文件时所有渠道照常打包；`firebase-messaging` 依赖可无条件加；FCM 代码运行时探测 `FirebaseApp.getApps(ctx).isNotEmpty()`，未初始化整段跳过、不抛异常 | 丢入 `google-services.json` + 重打包 |
| 后端 | 4 表 / token 注册 / campaign CRUD / 历史 / audience **全部可跑可测**；`send` 检测 `PUSH_ENABLED`+`FIREBASE_SA_*`，缺失返回「FCM 未配置」，campaign 留 draft；提供 **dry-run**（走完取 token+统计但不真发） | 填 3 把私钥 + projectId + `PUSH_ENABLED=true` |
| Web | 读 `GET /api/push/status`，未配置挂提示条 + 发送置灰，编辑/选包/历史全可用 | 后端置位后自动解禁 |
| CLI | 拉 `google-services.json`「有则拉、无则跳过提示」，不阻断打包 | 后端有文件后自动拉取 |

激活清单：① 建 3 个 Firebase 项目、各注册本品牌全部 flavor、导出合并 json；② 上传后台/交 CLI 分发；③ 服务端填私钥+projectId+`PUSH_ENABLED=true`；④ 重打一批带 json 的包。

## 6c. gp 拆分（撞 Firebase 每项目 30 App 上限）

gp 品牌 42 个渠道 > Firebase「每项目最多 30 个 App」硬上限（`429 RESOURCE_EXHAUSTED: Too many Apps on project`，提额需付费），故 gp 拆成两个 Firebase 项目：**hybrid-gp**（30）+ **hybrid-gp2**（溢出 12）。

- **路由键**：FCM 发送不再纯按品牌，引入「路由键」ap/bp/gp/**gp2**。`fcm_routing.go` 读已上传的 `fcm/gp2/google-services.json`，把其 `client[].package_name` 建成 `applicationId → "gp2"` 索引；发送时命中索引走 gp2 项目，否则退回品牌 code（行为不变）。数据自维护，无需改库表。
- **gp2 私钥暂缺的兼容**（当前状态）：`gp2` 未配 `FIREBASE_SA_GP2/FIREBASE_PROJECT_GP2` 时，路由到 gp2 的 token 返回 `SendResult.Skipped`——**不发、不算失败、不下线 token**（push_record 记 errorSample 说明）。即「超过 30 的那批包暂不发推送」。配上 gp2 私钥后这些设备自动恢复补发，零改码。
- **存储/分发**：`fcm/gp2/google-services.json` 既供路由判定，也作 gp2 那批 flavor 的构建分发源（`validBrands` 已含 gp2）。CLI 按 flavor 路由到 gp 或 gp2 的 json 仍待做。
- **新增配置**：`FIREBASE_SA_GP2`、`FIREBASE_PROJECT_GP2`（缺失即上面 no-op 行为）。

## 7. 里程碑（建议）

| M | 内容 | 负责 agent |
| --- | --- | --- |
| P0 | 后端：4 model + migration + token 注册端点 + campaign CRUD（先不发送）；产出 OpenAPI | backend-go |
| P1 | 后端：FCM HTTP v1 发送 + worker pool + 失效下线 + cron 定时 | backend-go |
| P2 | APK：插件/依赖 + Service + token 注册 + 点击经 DomainResolver 跳转 | android-kotlin |
| P3 | Web：推送编辑（复用 PackPage 多选）+ 历史 + audience 预估 | frontend-react |
| P4 | CLI/构建机：google-services.json 按品牌投放 | cli-go |
| P5 | 联调 + code-reviewer 对照护栏审查（容灾/机密/Gradle 被动） | code-reviewer |

依赖：P0 先定 API 契约 → P2/P3 并行消费；P1 与 P2 可并行；P4 独立。
