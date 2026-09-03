# CLAUDE.md

> 本文件每次会话自动加载，是项目的「最小必读」。详细方案见 [docs/admin/](docs/admin/)，关键决策见 [docs/adr/](docs/adr/)。

## 项目是什么

hybrid-app：3 个品牌（大渠道）的 **Android WebView 壳应用**，每个品牌下有大量小渠道包（小渠道 = 一个 Gradle product flavor）。

- 品牌：`ap`=ArenaPlus / `bp`=BingoPlus（带 HMS/OAID）/ `gp`=GameZone。
- 小渠道清单：`channels/<brand>.csv`，字段 `flavor|applicationId|palCode|appName`，`app/build.gradle` 据此**动态生成 flavor**。
- 每渠道资源：`app/src/channels/<brand>/<flavor>/res`（5 档 `mipmap-*/ic_launcher.png` + `drawable/splash_fullscreen.png`）。
- 启动加载：`${BuildConfig.DOMAIN}/?palcode=${BuildConfig.PAL_CODE}`，见 [WebViewActivity.kt](app/src/main/java/com/hybrid/android/WebViewActivity.kt)。
- 品牌差异走策略模式：`app/src/main/java/com/hybrid/android/brand/`（`BrandStrategy` / `BrandHost`）。

## 正在做什么：渠道中台（admin platform）

把「渠道清单 / 图标资源 / 域名配置」收归到一个后台统一管理，本地打包 CLI 与 APK 都从后台取配置。痛点：**域名经常变**（被封）、渠道纯手工维护 CSV 易错。完整方案见 [docs/admin/README.md](docs/admin/README.md)。

### 上架包（listing）+ AB 面网关

除渠道 APK 外，另有两个已上架商店的独立 App（`listings/colorstack`=Flutter 双端 / `listings/decktallypro`=原生 iOS）纳入 Console 管理，带「AB 面网关」：开启后**服务端**按请求真实 IP 的国家 + 时区判定，命中才放行 B 面（web，与品牌同一套域名），否则展示 App 本体（A 面）。判定客户端零内置 B 面地址、fail-closed、**CN/US 硬编码强制 A 面**。方案见 [ADR-0014](docs/adr/0014-listing-ab-gate.md) 与 [docs/admin/09-listing.md](docs/admin/09-listing.md)。

## 技术栈（已定）

| 部分 | 选型 |
| --- | --- |
| 后端 + 打包 CLI | **Go**（后端 Echo + GORM + golang-migrate + disintegration/imaging + golang-jwt + robfig/cron；CLI Cobra + charmbracelet/huh）。**不用 Node**（包太大，见 ADR-0001） |
| 前端 | React 18 + TS + Vite + shadcn/ui + Tailwind；API 客户端由 Go 的 OpenAPI 生成 |
| 存储 | MySQL（元数据）+ MinIO/OSS（图标/资源/APK） |
| Android | Kotlin（现有工程：AGP 8.12 / Kotlin 2.0.21 / compileSdk 36 / minSdk 29 / JDK 17） |
| 仓库布局 | `server/`(Go) · `cli/`(Go) · `web/`(React) · `app/`(现有 Android) · `listings/`(上架包：colorstack=Flutter / decktallypro=iOS)；`go.work` 串 server+cli |

本机已具备：Go 1.25、Node 22 + pnpm 9、JDK 17、Android SDK（`/Users/allen/Library/Android/sdk`）。

## 硬性护栏（不要违反）

1. **不改 Gradle 构建逻辑**：后台只是 `channels/*.csv` + `res/` 的上游编辑器，CLI 把后台数据**渲染回现有格式**。`app/build.gradle` 的 `loadChannels`/`productFlavors` 一行不动。（ADR-0004）
2. **域名不要编译期焊死**：APK 运行时拉取域名（`GET /api/app/config?palcode=`），成功即自更新本地缓存，编译期只烧录兜底清单。改域名 = 后台改一处、不重新打包。（ADR-0002）
3. **域名容灾不要「乱换」**：只有**确认是域名故障**才切备用；本机断网只提示「网络异常」。用中立连通性探针区分「域名问题 vs 本机网络问题」。（ADR-0003）
4. **keystore**：不进 git / DB / 配置 API / 对象存储 / 前端；本地开发用 `local.properties`；**服务器端构建直接把 keystore 烧进 `build-runner` 镜像**（运营决定「直接内置、不考虑安全」，单租户自托管 + 私有 registry、接受风险；见 ADR-0008）。运维零 keystore 操作。**多把 key**（ADR-0016）：已上架商店的老渠道（ap01018~ap01022、gzmkt031）证书是 `empty-app` 不可变，Console 只存 `signingKey` ID，构建机打包后按渠道用 apksigner 重签；非默认 key 同样烧进镜像 + 写 `/opt/hybrid/signing-keys.properties`；缺 key 时任务 fail-closed。
5. **身份与唯一性**（见 ADR-0009）：**applicationId 是唯一标识，派生自 `<品牌包前缀>.<flavor>`**（ap→`com.arenaplus` 等），表单自动填充、不手填；**PAL_CODE 不全局唯一**（跨品牌可重复），仅作 URL 参数 `/?palcode=` 与编译期烧录；**域名解析键用 applicationId（`BuildConfig.APPLICATION_ID`）而非 palcode**。唯一性以 applicationId 与 `(brand, flavor)` 为准。

## 常用命令

```bash
# Android（现状）
./package.sh                              # 交互式打包
./gradlew assemble<Flavor>Release         # 单渠道
./gradlew assembleDebug                   # 冒烟编译

# 后端（建成后）
cd server && go run ./cmd/server          # 启动
go vet ./... && go build ./...            # 自测
go test ./...

# CLI（建成后）
cd cli && go build -o bin/hybrid-pack ./cmd/hybrid-pack

# 前端（建成后）
cd web && pnpm install && pnpm dev        # 开发
pnpm build && pnpm typecheck
```

## 代码约定

- **Go**：标准布局 `cmd/` + `internal/{handler,service,repo,model}`；错误用 `fmt.Errorf("...: %w", err)` 包裹；handler 薄、service 厚；优先标准库，少引重依赖。
- **注释 / 文档 / commit 信息用中文**（与现有工程一致）。
- 改 Android 时尊重现有 `BrandStrategy`/`BrandHost` 插件架构，新增能力优先做成可被策略定制。
- 多代理协作时**各自只动自己负责的目录**（`server/` `cli/` `web/` `app/` `listings/`），共享根文件（`go.work` 等）由编排者维护。
