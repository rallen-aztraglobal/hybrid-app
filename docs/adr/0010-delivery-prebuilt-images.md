# ADR-0010: 交付模型——预构建镜像，运维仅运行 + 配域名/SSL

- **状态**：已采纳(2026-06-24)，更正 ADR-0008 中「运维在服务器上 `docker compose up --build`」的隐含做法

## 背景
首版部署文档让运维自己 `--build`、填一大堆环境变量、管 keystore/CDN，**太复杂**。运维要的是「拿到东西直接能跑」。

## 决策
**我方 CI 预先构建并发布版本化镜像；运维只消费镜像。**
- 镜像（`go-api` / `web` / `build-runner` + 官方 `nginx`/`mysql`）由 CI 用 `deploy/Dockerfile.*` 构建，按 `vX.Y.Z` 打 tag 推到我方 registry。
- `deploy/docker-compose.yml` 同时含 `image:`（运维 pull）与 `build:`（我方 build）。**运维只 `docker compose pull && up -d`，永不 build。**
- **运维唯一要配**：域名 + SSL 证书（首次再放一次签名 keystore）。其余（DB 口令、`JWT_SECRET`）预置或首启自动生成。
- 运维手册砍到一页（[05](../../docs/admin/05-deployment.md)）；构建/发布复杂度收进我方文档（[06](../../docs/admin/06-release.md)）。

## 理由
- 运维零编译、零环境（无需 JDK/Android SDK）、零长清单 → 上手快、出错少。
- 版本可复现（tag 锁定）、可一键升级（`pull && up`）、可回滚（换 tag）。
- 复杂度留在我方 CI（本就该我方承担）。

## 后果
- ✅ 运维交接极简：放证书 → 填域名 →（首次放 keystore）→ `up`。
- ➖ 我方需维护：CI 镜像流水线 + registry + 发布包脚本。
- keystore **烧进 `build-runner` 镜像**（运营决定、接受风险）；私有 registry 控制访问，运维零 keystore 操作。
- `deploy/docker-compose.yml` 必须给每个自建服务写 `image:` tag（不能只有 `build:`），否则运维无法 pull。

## 备选
- **运维从源码 `--build`**：需 JDK/SDK/Node、慢且易错、不可复现，被否（本 ADR 即更正它）。
- **裸机 systemd 部署**：装依赖、管进程，比容器更重，被否。
