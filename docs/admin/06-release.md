# 渠道中台 · 镜像构建与发布（我方）

> 面向我方研发/发布。运维**不用看这份**——他们只 `pull + up`（见 [05](./05-deployment.md)）。这里讲我们怎么把代码变成「运维能直接跑的镜像」。
> 决策依据：[ADR-0010 交付模型](../adr/0010-delivery-prebuilt-images.md)。

---

## 1. 交付物 = 预构建镜像 + 一个发布包

运维拿到**两个发布包**（对应两套部署，见 [05](./05-deployment.md)）：
- **`hybrid-api-release`**（服务 A·1C2G）：`docker-compose.api.yml` + `.env` + `certs/`。
- **`hybrid-console-release`**（服务 B·4C8G）：`docker-compose.console.yml` + `.env` + `certs/`。（keystore 已烧进 `build-runner` 镜像。）

各含**锁定 tag** 的 compose 与 `.env` 模板。**所有代码/JDK/SDK/依赖/web 静态产物都在镜像里。**

镜像（推到我方 registry，按平台版本打 tag `vX.Y.Z`）：

| 镜像 | Dockerfile | 内含 |
| --- | --- | --- |
| `go-api` | `deploy/Dockerfile.api` | 多阶段编译的 Go 单二进制（distroless/scratch） |
| `web`（或并入 nginx） | `deploy/Dockerfile.web` | 预构建的 React `dist`，由 nginx 托管 |
| `build-runner` | `deploy/Dockerfile.builder` | JDK17 + Android SDK(cmdline-tools+platform-36+build-tools) + `hybrid-pack` 二进制 |
| `nginx` | 官方镜像 + 挂载 `nginx.conf` | 反代 + 静态(`/apks`、`/static`) |
| `mysql` | 官方 `mysql:8` | — |

> compose（`docker-compose.api.yml` / `docker-compose.console.yml`）同时写 `image:`（给运维 pull）与 `build:`（给我方 CI build）。我方 `build+push`，运维 `pull+up`，**运维永不 build**。服务 A 含 `go-api`+`mysql`+`nginx`；服务 B 含 `nginx`+`web`+`build-runner`，且 **keystore 只在 B**。

---

## 2. CI 发布流程

```bash
# 在 CI（我方），定版本号
export TAG=v1.2.0
# 构建全部镜像（用 compose 的 build: 段）
docker compose -f deploy/docker-compose.yml build
# 打 tag + 推送
docker compose -f deploy/docker-compose.yml push        # 镜像 tag 由 .env 的 TAG 注入
# 产出发布包给运维（compose + .env 模板 + certs 占位 + README）
./deploy/make-release-bundle.sh $TAG     # 产出两个包：hybrid-api-release / hybrid-console-release
```

要点：
- **镜像 tag 锁定**：发布包里的 compose 用具体 `vX.Y.Z`，不用 `latest`，保证运维拿到的可复现。
- **多架构**（若运维机型不一）：`docker buildx --platform linux/amd64,linux/arm64`。

---

## 3. 什么烧进镜像 / 什么运行时配

| 类别 | 位置 |
| --- | --- |
| 代码、JDK、Android SDK、hybrid-pack、web dist、默认 nginx.conf、**80 渠道 csv+res（首部署 seed，ADR-0011）** | **烧进镜像** |
| 域名、SSL 证书 | 运维放（`.env` + `certs/`） |
| 签名 keystore | **烧进 build-runner 镜像**（运营决定，见 §4） |
| DB 口令、`JWT_SECRET` | `.env` 预置或**首启自动生成**（`JWT_SECRET` 未设则随机），运维一般不动 |
| 初始 admin | `BOOTSTRAP_ADMIN` 默认值，首登强制改密 |

---

## 4. keystore 内置镜像（运营决定）

运营明确：**keystore 直接内置，不考虑安全问题**（单租户自托管 + 私有 registry，已接受风险）。

- **CI 构建 `build-runner` 镜像时把 keystore 烧进去**：`Dockerfile.builder` 里 `COPY release.keystore` + 写好 `local.properties`（`KEYSTORE_FILE`/`KEYSTORE_PASSWORD`/`KEY_ALIAS`/`KEY_PASSWORD`），Gradle 的 `signingConfigs.release` 直接读，无需运行时注入。
- **运维零 keystore 操作**：发布包不含 keystore，`.env` 无 keystore 项。
- 仍**不进 git / DB / 配置 API / 前端**（没必要，也不放）。
- 风险与对策：镜像 = 含签名密钥，**镜像仓库必须私有、访问受控**；轮换密钥 = 换文件后重新构建并发布 `build-runner` 镜像。

---

### 4.1 多把签名 key（ADR-0016）

- 已上架商店的老渠道（ap01018~ap01022、gzmkt031）证书是 `empty-app`（`gzmkt031-key.jks`），不能改。
- Console 渠道表单选「签名 key」（只存 ID，默认空=release-key）；manifest 带 `signingKey`；
  runner 在 assemble 后对这些 flavor 用 `apksigner sign`（v1+v2）重签并 verify 后再投递。
- 镜像多烧一把：`deploy/secrets/store-emptyapp.keystore` + `.env.release` 的
  `STORE_EMPTYAPP_KEYSTORE_PASSWORD / STORE_EMPTYAPP_KEY_ALIAS / STORE_EMPTYAPP_KEY_PASSWORD`，
  `release.sh` 会作为 build args 传入，镜像内生成 `/opt/hybrid/signing-keys.properties`。
- 构建机缺这把 key 时，需要它的渠道任务直接失败（fail-closed），默认 key 的渠道不受影响。
- 新增一把 key：server `model.SigningKeys` 加条目 → Dockerfile.builder 多 COPY 一份 + 注册表多一段 →
  `.env.release` 补口令 → 发版 → Console 里给渠道选上。

## 5. 域名配置端点抗封（我方基建）

`GET /api/app/config?appId=` 决定 APK 能否热更域名（ADR-0002/0009）。业务域名常被封，**配置端点要放在不易被封的域名/CDN**：
- 后台保存域名时生成静态 `config-<appId>.json` 推到对象存储/CDN；
- APK 编译期烧录稳定的 `configUrl`（CDN）。

这属我方平台基建，不在运维单机 compose 范围内（运维那套是自托管最简版；规模化时再接 CDN）。

---

## 6. 发布检查单

- [ ] 镜像按 `vX.Y.Z` 打 tag 并推送成功；compose 锁定该 tag。
- [ ] `go-api` / `build-runner` / `web` 三镜像本地 `pull` 后能 `up` 起来、`/healthz` 通。
- [ ] `.env` 模板默认值安全（无真实口令）；`JWT_SECRET` 留空→首启自生成。
- [ ] keystore 已烧进 `build-runner` 镜像；镜像仓库私有、访问受控。
- [ ] 发布包含 05 运维手册；版本变更写 CHANGELOG。
