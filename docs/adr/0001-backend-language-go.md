# ADR-0001: 后端与打包 CLI 采用 Go

- **状态**：已采纳(2026-06-24)

## 背景
渠道中台需要一个后端（CRUD + 图标处理 + 配置下发）和一个本地打包 CLI（跨 Windows/macOS）。前端已定 React 18 + MySQL。最初推荐 NestJS（与前端同语言、`sharp` 做图标），但用户明确反馈 **Node 的包太大**：NestJS `node_modules` 常 150–300MB，CLI 用 `pkg`/`bun --compile` 打的单文件 45–90MB，分发到同事机器上笨重。

## 决策
后端与 CLI 统一用 **Go**。后端 Echo + GORM + golang-migrate + `disintegration/imaging` + golang-jwt + robfig/cron；CLI Cobra + charmbracelet/huh。

## 理由
- 后端编译成**单个静态二进制 ~10–20MB**，无运行时依赖，Docker 镜像可 ~20MB，内存 15–40MB。
- CLI **交叉编译**到三平台各 ~5–15MB、零依赖，最契合「分发给同事的跨平台打包工具」。
- 图标多密度生成用纯 Go 的 `imaging`（缩放/圆形遮罩/留边）即可，保持全静态无 cgo；`sharp` 的优势不再是决定性因素。
- `go.work` 让 server 与 CLI 共享类型。

## 后果
- ✅ 体积、部署、分发全面变轻。
- ➖ 失去与 React 前端「同语言共享类型」——用 OpenAPI（swag）从 Go 生成 spec → `openapi-typescript` 生成 TS 客户端补回。
- ➖ 对团队是新语言（团队现有 Kotlin/Android 背景）；用标准布局 + 评审降低风险。

## 备选
- **NestJS/Node**：体积过大，被否。
- **Kotlin + Spring/Ktor**：可复用 Android 团队语言，但 fat jar 需 JRE、GraalVM 原生编译折腾、CLI 跨平台分发不如 Go。
- **Rust**：二进制更小但开发慢、上手陡，CRUD 中台杀鸡用牛刀。
