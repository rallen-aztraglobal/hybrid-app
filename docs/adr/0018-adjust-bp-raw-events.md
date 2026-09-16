# ADR-0018: Adjust「BP 原始事件」模式——按渠道编译期开关切换事件逻辑分支

- **状态**：已采纳（2026-09-11）
- **背景**：BP（BingoPlus）团队自家的马甲壳（`H5Shell`）有一套与我们不同的 Adjust 埋点：
  H5 通过自定义 scheme `adjusth5event://<action>?...` 主动触发、原生解析后上报；`customerId`
  持久化并绑定为 Adjust `externalDeviceId`；充值带 `revenue` + `deduplicationId`；每次冷启动报
  `ad_app_opened`；投放短链走 `Adjust.processDeeplink`。事件集 14 个（`ad_*` 9 个 + `Action_*` 5 个），
  与我们现有的 6 个（`AddToCart/CompleteRegistration/Login/OldRegPurchase/Purchase/TPFirstDeposit`）
  完全不同。运营要求 BP 小渠道包能**按渠道选择**走哪一套：关闭时与现在一字不差，开启时走 BP 那套。
  约束不变：不改 Gradle flavor 逻辑（ADR-0004）、applicationId 为唯一键（ADR-0009）、
  Adjust 依赖恒在 classpath 靠数据文件门控（ADR-0013）。
- **决策**：
  1. **开关是编译期 feature gate，不走运行时下发**。`channel.adjust_bp_raw_events`（bool，默认 0，
     仅 `brand.code == bp` 允许 true）沿 `GET /api/build/manifest` → CLI → `app/adjust-tokens.json`
     条目的 `bpRawEvents` → `BuildConfig.ADJUST_BP_RAW_EVENTS` 一路带下去，与 `adjustAppToken`
     同一条链路。理由：切换模式意味着 Adjust app 的事件集从 6 个变 14 个，必须重传事件 CSV、重打包
     才有 token，运行时开关没有意义，只会多一个接口。
  2. **运行时单点分叉**：`AdjustBootstrap.bpRawMode = enabled && ADJUST_BP_RAW_EVENTS`。关闭时所有
     新增代码路径不可达，WebView 设置、JS 桥、`sendAFEvent` 分发行为与加开关前完全一致。
  3. **事件归属**（14 个）：
     - 原生发 2 个：`ad_app_opened`（冷启动 initSdk 后）、`ad_deeplink_opened`（品牌短链）。
     - H5 发 6 个：`ad_registration`（`register`：先绑 customerId 再上报，原生**不自己判断注册**）、
       `ad_deposit / ad_web_deposit / ad_web_login / ad_web_pageview / ad_web_reg`，三个拦截点
       （`shouldOverrideUrlLoading`、`BingoPlusShell.openExternal`、`window.open`→`onCreateWindow`）
       汇到同一个 `handleH5Url(uri)`。action → 事件名：`register→ad_registration`、`deposit→ad_deposit`；
       其余先按原名在事件表找，再找 `ad_` + action；都没有则只消费不上报（防 WebView 加载未知 scheme
       误触发域名容灾）。
     - `ad_game_open` 与 `Action_*` 5 个 App 不处理（前者需求确认后去掉，后者 S2S 侧），
       只在 adjust-sync 建事件时照单建好。
     - `updatecustomerid` 不是事件：只更新本地缓存（空 = 退出）。**externalDeviceId 在下次冷启动
       `initSdk` 前用缓存值 `AdjustConfig.setExternalDeviceId` 设置**——已核 SDK 5.4.1 字节码，参考实现
       用的 `setExternalDeviceIdInDelay` 只在 first session delay 期间生效，否则静默丢弃，我们不开 delay。
  4. **JS 桥原样注入 `BingoPlusShell`**，只实现 H5 会调的唯一方法 `openExternal(url)`。H5 识别
     「在壳内」靠站点加载 URL 上追加的 **`appSource=mktApp`**（bpRawMode 才加），不靠探测对象，
     因此不存在「H5 调到不存在方法」的风险。值固定为 `mktApp` 而非包名：H5 有 appSource 白名单
     （lite/slim/search 三个自家马甲包名 + `mktapp`），白名单外一个事件都不发（08-adjust.md §11.3）。
  5. **金额解析比 BP 宽松**：`amount` 剔除千分位、货币符号后 `toDouble`（BP 用 `toInt`，`100.5` 会丢收入）；
     `currency` 缺失或不合法回落 `PHP`；解析失败仍照发事件，只是不设收入。
  6. **短链 host 品牌级配置**（`brand.adjust_deeplink_host`，如 `link.bingoplus.com`），CLI 只对
     bpRaw 渠道写进 `adjust-tokens.json` 的 `deepLinkHost`，Gradle 用 `androidComponents.onVariants`
     只给这些变体生成并合并一份含 App Links intent-filter 的 overlay manifest；其余变体 manifest
     不变。不写死域名，也不用占位 host（会让所有包触发无谓的 App Links 校验）。
  7. **AppsFlyer 双份**：现有拦截接口那套照旧给 AF 发老 6 个事件；本分支上报的每个 `ad_*` 事件也
     同名同参数再给 AF 一份（有收入带 `af_revenue`/`af_currency`）。bpRawMode 下老 6 个不再进 Adjust。
- **理由**：
  - 与 ADR-0013 同一套 gate 心智，后台/CLI/Gradle 各只加一个透传字段，Gradle 仍只改自包含旁路块。
  - 单点 `bpRawMode` 分叉让「关闭 = 现状」可以被代码审查直接验证，回退只需关开关重打包。
  - 注册也交给 H5 而非原生判定：运营确认 BP 侧以 H5 触发为准，避免两套判定口径不一致。
- **后果**：
  - 正面：BP 渠道可逐个灰度切换；两套逻辑互不污染；短链/事件表/开关都在 Console，零改码切换。
  - 负面：多一张事件表要维护（adjust-sync 按模式建 6 或 14 个）；H5 侧要识别 `appSource` 参数；短链 App Links 自动验证需在 Adjust 面板逐包登记证书指纹，否则退化为系统选择框；
    `link.bingoplus.com` 属 BP 自己的 Adjust 账号，我们的小渠道 app 若不在同一账号，短链不会归因到
    我们的包（配置位在后台，届时换成我们账号的链接域即可）。
  - 已知缺口：bpRawMode 下 Adjust 侧没有「仅一次」的安装类事件（老 `Install` 不再进 Adjust，
    `ad_app_opened` 每次冷启动都发），安装量以 Adjust SDK 自带的 install 归因为准；登录后的
    externalDeviceId 绑定要到下次冷启动才生效；`window.open` 的 POST 表单退化为 GET。
  - 跟进：`Action_*` 5 个事件的 S2S 发送方需要按 palcode 拿到对应渠道的 app/event token，这条链路
    目前不存在，不在本 ADR 范围。
- **备选**：
  - 运行时下发开关（同域名）：切换仍需重传 CSV 重打包，徒增接口，否决。
  - 不注入 `BingoPlusShell`、改走我们自己的 `JSBridge.call('openExternal')`：要 H5 为我们单独改一条
    桥契约；运营确认 H5 只调 `openExternal` 一个方法且靠 `appSource` 识别壳，同名注入无风险，故否决。
  - `ad_registration` 复用现有拦截 `loginAndRegisterV4` 的判定：与 H5 触发双报、口径不一，否决。
  - 原生监听 `/game` 路径上报 `ad_game_open`：需求确认后去掉，事件保留在 Adjust app 里不发。
