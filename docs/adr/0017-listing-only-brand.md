# ADR-0017: 品牌分两条产线——渠道 APK 品牌 vs 只做上架包的品牌

- **状态**：已采纳（2026-09-11）
- **背景**：新增大渠道 **WP / WavePlay**（主域名 `https://www.waveplay.co`），但它**只做上架包
  （listing），不做小渠道包**。此前「品牌」这个概念隐含等同于「渠道 APK 产线的一员」：
  `app/build.gradle` 的 `brandConfig`、`channels/<brand>.csv`、`package.sh`、CLI 的品牌表、
  按品牌拆的 Firebase 项目，全是围绕渠道 APK 建的。而上架包侧（ADR-0014）又确实需要品牌：
  `listing_app.brand_id` 指向品牌、默认继承品牌域名作 B 面，域名换了两条产线一起换。
  于是 WP 必须「在品牌表里存在，但不进渠道产线」。
  若不显式区分，后果不是少个 Tab 那么轻：一旦有人在 Console 给 WP 建了渠道、CLI `pull` 把它渲染进
  `channels/wp.csv`，`app/build.gradle` 会在 `brandConfig[ch.brand]` 为 null 时于**配置阶段**直接失败——
  所有渠道都打不了包。
- **决策**：
  1. **品牌加一个产线标记** `brand.supports_channels TINYINT(1) NOT NULL DEFAULT 1`（迁移 `000016`）。
     `true` = 渠道 APK 产线（ap/bp/gp）；`false` = 只做上架包（wp）。默认 1 使存量品牌加列后行为不变、无需回填；
     wp 由 `seed.EnsureBrands` 建行并置 0（该字段的唯一事实来源是 seed 的 `brands` 表，Console 不给改）。
  2. **后端 fail-closed**：`CreateChannel` 与 `seed.ImportCSV` 对 `supports_channels=false` 的品牌一律 400 /
     报错拒绝；`GET /api/brands` 照常下发该品牌并带上 `supportsChannels`，由调用方按用途取舍。
  3. **渠道产线里不登记它**：不进 `channels/*.csv`、不进 `app/build.gradle` 的 `brandConfig`、不进
     `package.sh` / CLI 的品牌表（CLI 对 wp 直接「未知品牌」）、不进 `deploy/fcm-setup.sh`
     （品牌级 Firebase 项目只服务渠道设备；上架包推送走独立项目 `hybrid-listings-51660`）。
     也**不需要** `BrandStrategy` 实现——那是渠道 APK 壳的归因插件，上架包有自己的工程。
  4. **Console 按字段过滤**：渠道管理 / 打包中心 / 构建记录 / 推送不展示它（`useScopedBrands` 与
     `CHANNEL_BRAND_ORDER` 统一口径）；**域名配置**与**上架包的「归属品牌」**照常包含它。
     前端拿不到该字段时按 `true` 兜底，老后端行为不变。
- **理由**：
  - 品牌承载的是「一套域名 + 一个身份」，产线归属是它的属性而非定义——把这个属性显式化，
    比让两条产线各自猜「这个品牌算不算我的」要稳。
  - 标记放在 DB 而不是前端常量：拦截点必须在后端（Console 写入路径、CSV 导入），
    否则护栏绕得过去，而绕过去的代价是 Gradle 全量打包失败。
  - 不选「看该品牌有没有渠道来推断」：空品牌与「禁止建渠道的品牌」语义不同，
    前者只是还没建，后者是永远不该建。
- **后果**：
  - 新增品牌时要显式声明走哪条产线（`seed.brands` 的 `Channels` 字段），漏填即默认渠道品牌。
  - `seed.ChannelBrandCodes()` 成为「渠道品牌」的单一来源，CSV/res 目录探测与批量导入都走它。
  - GORM 陷阱已在 seed 处理：带 `default:` 标签的字段零值不进 INSERT，故建 wp 后要补一次显式
    `UPDATE supports_channels=false`（列默认值不能去掉，老库加列靠它）。
  - WP 目前只有品牌与域名，尚无 listing 记录；日后在 Console 新建上架包时选 WP 即继承该域名作 B 面。
