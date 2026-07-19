# ADR-0014: 上架包 AB 面网关（服务端判定，客户端零内置 B 面地址）

- **状态**：提议
- **背景**：除现有 3 个品牌的 Android WebView 壳（走 `channels/*.csv` + flavor 打包）外，新增两个已正式上架应用商店的独立 App —— **ColorStack**（Flutter，Android+iOS 同包名 `com.vividnest.colorstack5821`）与 **DeckTallyPro**（原生 iOS Swift，`com.deck.tallypro`）。这两个包本身是干净的小游戏（A 面），需要一个「合规审核期展示 A 面、真实用户展示配置的 web 链接（B 面）」的开关。诉求：
  - Console 里统一管理这些「上架包」，按 Android/iOS 分类，可增删、配置域名（与现有品牌同一套 web）、接入 AF 与 Adjust；
  - 每个上架包一个「开启 AB 面」总开关；开启后按 **IP 所属国家** 与 **客户端时区** 判定是否放行 B 面；
  - 可选择允许访问的国家/时区；**中国、美国强制走 A 面**，不提供为这两国放行的选项。
- **决策**：
  1. **判定放服务端，客户端不内置任何 B 面地址**。App 启动 `POST /api/app/listing/gate`，服务端用**请求真实 IP** 查国家 + 校验时区/IP 规则，只回 `{mode:"A"}` 或 `{mode:"B", url}`。审核方拿到安装包静态扫描翻不出域名；规则改一处即时生效，已上架的包**不重新发版、不过审**。
  2. **判定顺序，任一步判 A 即短路**（唯一事实来源在 `server/internal/model/listing.go` 的 `ListingGate` 文档 + `service/listinggate.go`）：
     ① 总开关 `gate_enabled=false` → A ；② 国家 ∈ {CN, US}（硬编码）→ A ；③ 命中 IP 黑名单 → A ；④ GeoIP 解析不出国家 → A ；⑤ 国家 ∉ 白名单 → A ；⑥ 时区白名单非空且时区不在其中 → A ；⑦ IP 白名单非空且 IP 不在其中 → A ；⑧ 全部通过 → B。**条件一律 AND**，无「任一命中即放行」模式。
  3. **默认安全（fail-closed）**：任何不确定情形一律返回 A。判错成 A 只是少放一个真实用户，判错成 B 可能让审核员看到 B 面 —— 两种错误代价不对称，故所有分支向 A 收敛。新建上架包 `gate_enabled` 恒为 `false`；打开前服务端强制校验国家白名单非空。
  4. **国家白名单必填，CN/US 不可入白名单**。空白名单视为**配置无效**（拒绝保存 / 判 A），而非「不限国家」—— 否则一次误删配置即全量放行。CN/US 的强制 A 面写在**服务端判定逻辑**里（`model.ForcedACountries`），不只是前端不给选项：前端约定可被绕过，服务端闸不可。
  5. **真实 IP 提取抗伪造**（`server/internal/httpx/realip.go`）：`X-Forwarded-For` 客户端可伪造，故先判直连对端是否是可信代理（默认私有网段，同机 nginx+go-api）—— 不可信则完全忽略 XFF 用直连地址；可信则从 XFF **最右往左**扫，停在第一个不可信地址。伪造值只会落在链左侧，扫描在遇到它前已停在代理记录的真实对端。
  6. **GeoIP 用 DB-IP country-lite（CC-BY 4.0）打进镜像 + cron 月更**（`server/internal/geoip`）：免费、无需账号/凭据、URL 按月份可拼出，故能构建期烤进 `go-api` 镜像（首启即可判定）、运行期 `robfig/cron` 每月 3 号自动拉新热替换（`atomic.Pointer`，判定不中断）。国家粒度足够，零人工维护。DB-IP 署名在 Console 设置页页脚声明。
  7. **域名与品牌同一套（ADR-0006 继承语义）**：`listing_app.brand_id` + `use_brand_domains`，默认继承所属品牌域名清单，`listing_domain` 可覆盖。B 面 URL = 生效域名的主域名。
  8. **AF/Adjust A 面也初始化**：安装事件由 SDK 自动归因，进 B 面用 AF 标准事件 `af_content_view`（标准事件名，出现在游戏里不可疑）。A 面同样初始化 SDK —— 「集成了却完全不用」在审核侧比「用了标准事件」更可疑。Adjust 复用 ADR-0013 既有的 token/events 契约。
  9. **判定落流水表 `listing_gate_log`**（IP/国家/时区/A 或 B/原因），上线后排查「为什么这台设备没进 B」全靠它；量大可 `GATE_LOG_ENABLE=false` 关。**返回给客户端的响应绝不含原因/国家/命中规则**（审核方也会调此端点），原因只进日志与后台试算接口。
- **理由**：
  - 服务端判定是唯一能同时满足「客户端零泄露 + 改规则不发版」的方案；这正是相对现有 Android 渠道包（域名运行时下发、ADR-0002）的一致延伸 —— 把「下发什么域名」升级为「先判 A/B 再决定给不给域名」。
  - fail-closed 与 CN/US 硬编码是审核安全的地基：宁可漏放真用户，不可漏放审核员。
  - DB-IP 相较 MaxMind GeoLite2 唯一但决定性的优势是**无需 license key**，才能兑现「不要人工维护」。
- **后果**：
  - 正面：两个已上架包无需为改域名/改放行规则重新发版；判定逻辑是纯函数、被穷举测试；GeoIP 全自动。
  - 负面：新增 GeoIP 库（镜像 +~8MB）与一个公开不缓存端点（判定因 IP 而异，无法 CDN 强缓存，与 `/api/app/config` 相反）；时区由客户端上报可伪造，故只作叠加收紧条件、永不单独作准。
  - 跟进：新表 `listing_app/listing_domain/listing_gate/listing_gate_log`（migration 000006）；Console 新增「上架包」页；两个客户端（Flutter/iOS）接入 gate 调用与 A/B 切换；推送对上架包**强制只推 B 面设备**（见 docs/admin/07-push.md 增补）。
- **备选**：
  - **判定放客户端**（下发规则、端上判 IP/时区）：客户端能被反编译出规则与域名，审核期风险高，且改规则要发版，否决。
  - **MaxMind GeoLite2**：精度相当但需注册账号 + license key 下载，无法全自动更新，与「不要人工维护」冲突，否决。
  - **空国家白名单 = 不限国家**：语义更「方便」，但一次误删即全量放行审核员，与 fail-closed 冲突，否决（空 = 配置无效）。
  - **CN/US 仅前端不给选项**：可被绕过前端直接调 API 破坏，否决（改为服务端硬编码闸）。
