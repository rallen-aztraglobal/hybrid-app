# ADR-0004: 后台为唯一数据源，CLI 渲染回现有 CSV/res，Gradle 不动

- **状态**：已采纳(2026-06-24)

## 背景
现有 Android 构建已能从 `channels/*.csv` 动态生成 flavor，资源放 `app/src/channels/<brand>/<flavor>/res`，工作良好。新增后台后，需决定后台与现有构建如何衔接：是重写 Gradle 直连后台，还是保留现有机制。

## 决策
后台是 `channels/*.csv` + `res/` 的**上游编辑器**（唯一数据源）。本地 CLI `hybrid-pack pull` 把后台数据**渲染回与现状字节级兼容的文件**（CSV 保留注释头、res 目录结构不变、外加每 flavor 的 `assets/bootstrap.json`）。`app/build.gradle` 的 `loadChannels`/`productFlavors`/`sourceSets` **一行不改**。

## 理由
- 低风险：构建逻辑零改动，出问题面小，可随时回退到「手工编辑 CSV」。
- 后台与构建解耦：后台演进不影响打包；CLI 是唯一的桥。
- 顺带用后台唯一性约束清洗现有 CSV 脏数据（包名重复 `ap01035`、`gzmarket062`）。

## 后果
- ✅ 现有打包流程稳定不动；CLI 可独立演进。
- ➖ 存在「本地文件 = 后台快照」的同步语义，需 `status` 做漂移检测、`pull` 前提示覆盖。
- Android 侧唯一改动是 `WebViewActivity` 的 `loadUrl`（接 DomainResolver），与本 ADR 无关。

## 备选
- **Gradle 直接调后台 API 拉配置**：构建期强依赖网络/后台、CI 不稳、改动现有逻辑，被否。
