# ADR-0013: Adjust 归因集成（按 flavor 绑定即启用，未绑定则休眠）

- **状态**：提议
- **背景**：运营需要在部分渠道包接入 Adjust 归因（与现有 AppsFlyer 并行）。约束与 FCM（ADR-0012）同构：80+ 个 applicationId（每 flavor 一个）、不改 Gradle flavor 生成逻辑（ADR-0004）、域名编译期不焊死（ADR-0002）、applicationId 是唯一标识（ADR-0009）。诉求：**只有在 Console 后台给某渠道包绑定了 Adjust App Token 的包，打包时才集成并发事件；未绑定的包完全不集成、不发任何 Adjust 事件**。另外：**Adjust 账号无 Automation**，无法程序化建 app/建事件/拿 token，只能在 Adjust 面板手工建好后导出。
- **决策**：
  1. **完全复刻 FCM 的 feature gate（ADR-0012）**：Adjust SDK 依赖**无条件进 classpath**（同 firebase-messaging），靠一个独立数据文件 `app/adjust-tokens.json`（CLI 从后台渲染）+ 按 flavor 探测决定「集成 / 跳过」。`app/build.gradle` 用**自包含旁路块**（`applicationVariants.all`）按 `applicationId` 注入 `ADJUST_APP_TOKEN` / `ADJUST_EVENT_MAP` 两个 BuildConfig 字段；`loadChannels`/`productFlavors` **一行不改**（守 ADR-0004）。
  2. **运行时休眠即「不集成」**：`AdjustBootstrap.enabled = BuildConfig.ADJUST_APP_TOKEN.isNotBlank()`。为空 → `init`/`trackEvent` 全部 no-op：**SDK 不初始化、无会话、无网络、不发事件**（与 FCM 未配 `google-services.json` 时 no-op 同构）。不采用「按 flavor 物理排除依赖」，因为 `src/main` 共享代码引用 `com.adjust.sdk.*`，物理排除会导致未绑定 flavor 编译不过（需反射壳，代价过高）。
  3. **绑定键用 applicationId**（ADR-0009），非 palcode。App Token 编译期烧录（它非机密、随 APK 分发、与 Adjust app 一一对应、几乎不变，不属于「会被封的域名」，故不走运行时下发）。
  4. **事件走现有 `BrandHost.sendAFEvent` 同源分发**：在 `WebViewActivity.sendAFEvent` 加一行 fan-out 到 Adjust，各 `BrandStrategy` **零改动**。App 内置一张固定的「内部事件名 → Adjust 事件 name」适配表（仅 `af_login→Login`、`af_complete_registration→CompleteRegistration` 两条，其余同名），使后台**只需 App Token + 上传 Adjust 事件 CSV** 即可，CLI/后台不感知 App 内部事件命名。
  5. **无 Automation → 后台纯人工录入**：每个渠道包一个 Adjust 配置 = 填 App Token + 上传 Adjust 导出的事件 CSV（`token,name,unique`）。方案不做任何 Adjust 自动化 API 假设。
- **理由**：
  - 与 FCM 同一套 gate 心智，一致性最高、护栏（ADR-0002/0004/0009）天然守住，运营「配了就走、没配跳过、零改码激活」体验统一。
  - 休眠而非物理排除：SDK 体积小，休眠等价于行为上「不集成」（零初始化、零流量），且共享代码可直接引用 Adjust 类、无需反射，改动最小。
  - 事件复用 `sendAFEvent` seam：一处 fan-out 即让全部既有埋点同时打到 AppsFlyer + Adjust，策略层不动。
- **后果**：
  - 正面：Adjust 完全按 flavor 自动开关；后台绑定零代码；AppsFlyer 逻辑一字不动；install/session 由 Adjust SDK 自动归因，`Install` 自定义事件可省。
  - 负面：未绑定包 APK 里仍含 Adjust SDK 字节（但完全惰性、不运行）；每个 Adjust app 的事件 token 各不相同（无 Automation 的固有代价），需逐渠道上传 CSV；`minifyEnabled` 已开，需补 Adjust 的 ProGuard keep 规则。
  - 跟进：`channel` 表加 `adjust_app_token` / `adjust_events`（可空）；CLI `pull/build` 渲染 `app/adjust-tokens.json` 并入 `.gitignore`；Web 渠道编辑页加「Adjust」区块（App Token + 上传 CSV）。
- **备选**：
  - 按 flavor 物理排除 Adjust 依赖（同 HMS/OAID 的 `add("${flavor}Implementation", ...)`）：未绑定包真不含 Adjust 字节，但共享代码引用 SDK 类会编译失败，需把所有 Adjust 调用包进反射壳，复杂且与 FCM 模式不一致，否决。
  - App Token 走运行时下发（同域名）：Adjust app token 不像域名会被封、且 SDK init 需尽早拿到 token，运行时下发徒增复杂度，否决（编译期烧录）。
  - 后台逐渠道手工映射「App 事件 → token」：比「上传 CSV」多一层人工、易错，否决（改由 App 内置固定适配表）。
