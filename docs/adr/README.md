# 架构决策记录（ADR）

记录本项目**重大、不易逆转**的技术决策：定了什么、为什么、放弃了什么。改主意时不要删旧 ADR，新写一条标「被 ADR-XXXX 取代」。新决策用 [0000-template.md](./0000-template.md) 起草，序号递增。

| # | 决策 | 状态 |
| --- | --- | --- |
| [0001](./0001-backend-language-go.md) | 后端与打包 CLI 采用 Go | 已采纳 |
| [0002](./0002-domain-runtime-config.md) | 域名运行时下发 + 自更新缓存 + 编译期兜底 | 已采纳 |
| [0003](./0003-domain-failover-no-flapping.md) | 域名容灾：区分域名故障与本机网络，绝不乱换 | 已采纳 |
| [0004](./0004-backend-source-of-truth.md) | 后台为唯一数据源，CLI 渲染回现有 CSV/res，Gradle 不动 | 已采纳 |
| [0005](./0005-icon-pipeline.md) | 图标管线：单张主图服务端 fan-out | 已采纳 |
| [0006](./0006-domain-granularity.md) | 域名配置粒度：大渠道默认 + 小渠道覆盖 | 已采纳 |
| [0007](./0007-repo-layout.md) | 仓库布局与工具链（go.work + React + 现有 Android） | 已采纳 |
| [0008](./0008-server-side-build.md) | 服务器端构建：独立构建机/容器 + build-job 队列 | 已采纳 |
| [0009](./0009-channel-identity-appid.md) | 渠道身份：applicationId 派生且唯一；PAL_CODE 不唯一、域名解析键改 appId | 已采纳（更正 0002） |
| [0010](./0010-delivery-prebuilt-images.md) | 交付模型：预构建镜像，运维仅运行 + 配域名/SSL | 已采纳（更正 0008） |
| [0011](./0011-first-deploy-seed.md) | 首次部署初始化：烧录 res + 首启自动注册（否决 init.sql） | 已采纳 |
| [0012](./0012-push-fcm.md) | Firebase 推送（FCM）集成与按品牌项目拆分 | 已采纳 |
| [0013](./0013-adjust-attribution.md) | Adjust 归因：按 flavor 绑定即启用、未绑定则休眠 | 提议 |
