# deploy/secrets/

构建机签名密钥的【挂载点】，仅作 docker secret 注入 `build-runner`。

- 把 release keystore 放成 `deploy/secrets/release.keystore`（或在 `.env` 用 `KEYSTORE_HOST_PATH` 指向别处）。
- 该目录除本 README 外**全部 gitignore**：keystore 绝不入仓库 / 镜像 / DB / 配置 API / 对象存储 / 前端（护栏 #4 / ADR-0008）。
- 口令通过 `.env` 的 `KEYSTORE_PASSWORD` / `KEY_ALIAS` / `KEY_PASSWORD` 注入（非密钥本体），由构建机入口写入仓库工作区的 `local.properties` 供 Gradle 读取。
