# 渠道中台 · 方案文档

> 解决「渠道包越来越多、web 域名经常变」的运营痛点：把渠道清单、图标资源、域名配置收归到一个后台统一管理；本地打包脚本与 APK 都从后台取配置 —— 让「改域名」从「重打全部包」退化为「后台点一下」。

## 一句话方案

```
React 18 后台  ──→  Go(Echo) + MySQL + imaging + 对象存储（单静态二进制）
                         │
        ┌────────────────┼─────────────────┐
        ▼                ▼                 ▼
  跨平台打包 CLI      运行时配置端点(CDN)    域名健康巡检
  (pull→build)      APK 启动拉最新域名     (监控红绿灯)
        │                │
        ▼                ▼
  现有 Gradle 零改动   WebViewActivity 容灾(主→备用, 不乱换)
```

## 文档导航

| 文档 | 内容 | 对应需求 |
| --- | --- | --- |
| [01 · 方案总览与架构](./01-architecture.md) | 整体架构、**后端选型(Go)**、域名实时下发策略、数据模型(MySQL DDL)、API 设计 | 全局 + 后端问题 |
| [02 · 域名容灾机制](./02-domain-failover.md) ⭐ | APK 端核心：区分「域名故障 vs 本机网络」的状态机、错误分类、Kotlin 草案、超时预算 | ④ |
| [03 · 打包工具与图标管线](./03-build-and-icon-pipeline.md) | 跨平台 Go CLI（Win/macOS）、Xcode 式图标九宫格、imaging 多密度生成 | ②③ |
| [04 · 开发计划](./04-roadmap.md) | 里程碑 M0–M7、工期估算、并行依赖、风险对策、验收基线 | 计划 |
| [05 · 运维部署手册（极简）](./05-deployment.md) 🛠 | 运维只做：放证书 → 填域名 → `pull && up`；日常运维/排查 | 运维 |
| [06 · 镜像构建与发布](./06-release.md) | 我方 CI 构建/推送版本化镜像、keystore 带外交付、发布包 | 我方 |
| [07 · Firebase 推送(FCM)](./07-push.md) 🔔 | APK 集成 FCM + Console 编辑推送选渠道包批量发送；按品牌 3 项目、设备 token 库、HTTP v1 发送、即时+定时 | 推送 |
| [08 · Adjust 归因](./08-adjust.md) 📊 | 按 flavor 绑定 App Token 即集成、未绑定则休眠（同 FCM gate）；后台填 token + 上传事件 CSV，事件复用 sendAFEvent fan-out | 归因 |
| [09 · 上架包管理](./09-listing.md) | ColorStack/DeckTallyPro 等独立合规应用 + AB 面放行网关 | 上架包 |
| [10 · 账号管理](./10-user-management.md) 👤 | Admin-only 账号 CRUD（无硬删除）：新建/改角色/启停用/重置密码，最后一个启用 admin 保护，禁用账号会话即时撤销 | 账号 |
| [UI 原型](./ui/index.html) 🎨 | 单文件 HTML，浏览器直接打开。3 Tab / 渠道卡片 / 新增抽屉 / 图标九宫格 / 域名配置 / 打包中心 | ①②④ |

> UI 原型已用浏览器实测渲染，截图见对话。运营版用 React 18 复刻该设计。

## 需求对照

| # | 你的需求 | 落地 |
| --- | --- | --- |
| ① | 三大渠道分 Tab | 后台顶部品牌 Tab（ArenaPlus/BingoPlus/GameZone），各带渠道计数与品牌色 |
| ② | 子渠道增删改 + 名字/icon/包名/PAL_CODE + Xcode 式图标 | 渠道 CRUD + 唯一性校验；图标传 1 张主图，前端方形裁剪、后端 imaging（纯 Go）生成 5 档×方形/圆形/自适应共 15 张，九宫格展示各尺寸位置、可单槽覆盖 |
| ③ | 本地打包拉后台配置 + 跨平台 | `hybrid-pack` Go CLI：`pull` 渲染出现有 Gradle 认识的 CSV/res，`build` 跨平台调 gradlew；交叉编译成 Win/macOS/Linux ~5–15MB 单文件二进制 |
| ④ | 主域名+3备用、容灾、不乱换 | DomainResolver 状态机：本机网络闸门 + 中立连通性探针双保险，只在确认域名故障时切换，本机断网只提示不乱换；实时拉取+自更新缓存使域名可热更 |

## 关键设计亮点

1. **域名热更**（架构核心）：APK 每次启动用不变的 `PAL_CODE` 调一次接口拉最新域名清单，**成功即更新本地兜底缓存**；域名被封时后台改一处、全网已装包下次启动生效，**无需重新打包**。三级取用（实时→缓存→编译期兜底）见 [01 §3.2](./01-architecture.md)，配置端点部署在抗封 CDN、与业务域名分离。
2. **不乱换**（你最在意的）：先用 `ConnectivityManager` 闸门挡掉「没网」，全失败时再用 `gstatic/cloudflare` 中立探针裁决「是域名问题还是本机问题」，只有前者才换域名/报「服务不可用」，后者只报「网络异常」。详见 [02](./02-domain-failover.md)。
3. **低风险接入**：后台是 `channels/*.csv` 的上游编辑器，CLI 把数据渲染回现有格式，**Gradle 构建一行不改**；Android 侧只需替换 [WebViewActivity.kt:187](../../app/src/main/java/com/hybrid/android/WebViewActivity.kt#L187) 一句 `loadUrl`。
4. **消灭脏数据**：现有 CSV 已存在包名重复（`ap01035`、`gzmarket062`），后台唯一性约束 + 导入清洗一次性解决。

## 待你拍板（默认按推荐推进）

| # | 决策 | 推荐 |
| --- | --- | --- |
| 1 | 后端语言 | **Go**（已选定，体积最优） |
| 2 | 域名下发 | **实时拉取 + 自更新缓存 + 编译期兜底**（已确认） |
| 3 | 域名粒度 | **大渠道默认 + 小渠道可覆盖** |
| 4 | 对象存储 | 自建 MinIO / 云 OSS |

详见 [01 · §8](./01-architecture.md#8-需你拍板的决策)。
