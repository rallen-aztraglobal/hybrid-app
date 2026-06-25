# ADR-0012: Firebase 推送（FCM）集成与按品牌项目拆分

- **状态**：已采纳(2026-06-26)
- **背景**：运营需要给已装机用户主动触达（活动/召回）。需求拆为两半：APK 集成 FCM 接收推送；Console 编辑推送内容并「选渠道包批量发送」。约束：80+ 个 applicationId（每 flavor 一个）、不改 Gradle flavor 生成逻辑（ADR-0004）、域名编译期不焊死（ADR-0002）、机密不入 git/DB/API/前端（ADR-0008）、applicationId 是唯一标识（ADR-0009）。
- **决策**：
  1. **Firebase 项目按品牌拆 3 个**（ap/bp/gp 各一）。每品牌一份**合并的 `google-services.json`**（含该品牌全部 flavor 的 `package_name`），由 CLI/构建机按品牌投放到 `app/google-services.json`，构建时 google-services 插件按当前 applicationId 自动选中对应条目。
  2. **投递走「设备 Token 注册库」**：APK 上报 `(applicationId, token)` 到公开端点，后端建 `push_device_token` 表，发送时按目标 applicationId 取活跃 token。**不使用 Topics**（拿不到精确设备数/送达统计）。
  3. **服务端发送走 FCM HTTP v1 REST**（`oauth2/google` 取 token），**不引入 `firebase.google.com/go` Admin SDK**（重依赖，违 ADR-0001 精简原则）。worker pool 逐 token 发送 + 自动剔除失效 token。
  4. **机密分级**：`google-services.json` **不是机密**（随 APK 分发），可入 DB/配置 API/CLI；**service account 私钥（3 把，发送用）是机密**，仅服务端经 env/挂载读取，照 ADR-0008 烧进 server 镜像或私有挂载，绝不入 git/DB/API/前端。
  5. **第一版支持即时 + 定时发送**：campaign 状态机 + 复用现有 robfig/cron 每分钟扫到期任务。
- **理由**：
  - 3 项目对齐现有 `BrandStrategy` 品牌隔离架构，跨品牌数据/配额不互相污染，又远少于「每渠道一项目」的 80+ 运维爆炸。
  - 合并 google-services.json + 构建期按 appId 匹配，使 flavor 生成逻辑**一行不改**，只新增插件与依赖，守住 ADR-0004。
  - Token 注册库满足「选渠道包」=「选 applicationId 集合」的精确投递与统计，符合 ADR-0009「applicationId 是唯一标识」。
  - HTTP v1 REST 守住精简二进制偏好，避免 Admin SDK 体积膨胀。
- **后果**：
  - 正面：域名不参与推送（payload 只带相对 path + palcode，APK 用 DomainResolver 拼域名，守 ADR-0002）；新增能力插件化、可被品牌策略定制。
  - 负面：需运维三套 Firebase 项目与 3 把私钥；HTTP v1 逐 token 发送随设备量增长请求数线性上升（用 worker pool + 限速兜住，量级巨大时再考虑 Topics）；新增 `POST_NOTIFICATIONS` 运行时权限（minSdk 29，Android 13+ 需动态申请）。
  - 跟进：需在 server 镜像/部署补 3 把私钥的投放（更新 docs/admin/05-deployment.md、06-release.md）；google-services.json 的生成/上传后台 UI 第一版可省略，先用 env/手工放置。
- **备选**：
  - 全局 1 个 Firebase 项目：最省事，但跨品牌数据混一起、配额共享，否决。
  - 每渠道 1 个项目：隔离彻底但 80+ 项目运维不可行，否决。
  - Topics 投递：无需 token 库，但拿不到设备数/送达统计、与「批量精确选包」统计诉求不符，否决（保留为超大规模时的演进路径）。
  - Admin SDK：API 友好但重依赖，违精简原则，否决。
