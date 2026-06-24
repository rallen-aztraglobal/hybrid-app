# ADR-0011: 首次部署的数据与图标初始化——镜像烧录 res + 首启自动注册（否决 init.sql）

- **状态**：已采纳(2026-06-24)

## 背景
线上已有 80 个渠道包，初次部署须把这批渠道 + 它们**已有的**图标/启动页一次性还原进库（手动给 80 个传图不现实）。曾提议改用预生成的 `init.sql` + 图片包、首启执行 SQL。

## 决策
采用「**镜像烧录源 + 首启自动导入/注册**」：
- `deploy/Dockerfile.api` 把 `channels/*.csv` + `app/src/channels/**/res` COPY 进 go-api 镜像的 `/app/seed`。
- 首次启动（`DB_AUTOSEED=true` 且**渠道表为空**）自动：导入 80 渠道（CSV，按 ADR-0009 派生 appId）+ 扫描 res 把每渠道 `icon/splash/res.zip` 注册进 storage（对外经 `/static`，由 nginx 或 go-api 本地的 `e.Static` 出）。
- 也可 `server import` 手动触发（dev / 重建）。
- **幂等**：非空库跳过；对象 key 按 appId 固定，重复覆盖不堆积。

## 理由
- 满足真实需求：运维首次 `docker compose up` 即得 80 渠道 + 图，**零手动上传**。
- 简单、可靠、已实测：sqlite 与真实 Docker 镜像均验证——80 渠道入库、全部 `/static` URL、图与源 md5 一致、重启幂等。

## 备选（已否决）
- **预生成 `init.sql` + 图片包，首启执行 SQL**：需保证 MySQL 方言正确、处理「GORM 建表 vs SQL 执行」的先后、写裸 SQL 多语句执行器（按 `;` 切分遇到值含 `;` 易碎）。复杂、易碎，产出与本方案**等价**。属过度设计，否决。
  > 注：`init.sql` 是最终用户提出的**建议**；经工程评估其合理性后否决，保留更简单的方案 A（用户已认可「具体实现由你确认合理性」）。

## 后果
- ✅ 首部署开箱即用。
- ➖ go-api 镜像携带 csv + res（镜像约 177MB），可接受。
- 图标对外 URL 用相对 `/static`，生产由 nginx 出、本地由 `cmd/server/main.go` 的 `e.Static("/static", local.Root())` 出。
