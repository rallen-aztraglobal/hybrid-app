# 渠道中台 · 运维部署手册（极简）

> 给运维。**镜像由我方 CI 预先构建好，你不用编译、不用改代码。** 每套服务都只是：**放证书 → 填域名 →  `pull && up`**。
> 镜像怎么构建/发布是我方的事，见 [06 · 镜像发布（我方）](./06-release.md)、[ADR-0010](../adr/0010-delivery-prebuilt-images.md)。

---

## 方案 A（推荐 · 当前采用）：单机合并部署（A+B 同机，一台 4C8G）

最简：**一台机器、一套 compose、一个 .env、不用填后端地址**。go-api / mysql / build-runner / web / 一个 nginx 全在本机，`/api` 走内网 `go-api:8090`（固定），管理后台、打包、APK 下载都在本机。

```bash
unzip hybrid_channel_console.zip && cd hybrid_channel_console
bash images/load-images.sh                 # 无私有 registry 时离线 load 镜像（含 build-runner）
cd deploy && cp .env.allinone.example .env
vi .env                                     # 暂无域名 → DOMAIN=<本机公网IP>（如 47.80.75.106）；改口令 + RUNNER_TOKEN
docker compose -f docker-compose.allinone.yml up -d
curl -fsS http://<本机IP>/healthz           # 期望 {"ok":true,...}
# 浏览器开 http://<本机IP>/ ，admin 登录 → 立即改密。
```

要点：
- **无 `API_BASE_URL`**：单机内 go-api 地址固定，部署时不用填后端地址；`RUNNER_TOKEN` 一份，go-api 与 build-runner 共用、天然一致。
- 当前 **HTTP/IP（80 端口）**：⚠️ admin 口令**明文传输**，仅作临时/内网用；拿到域名后启用 443 TLS（见文末「升级 TLS」）。
- 首启自动导入 80 渠道 + 图标/启动页（ADR-0011）；打包由 build-runner 自动轮询本机 go-api 出包，落 `/apks`，「构建记录」下载。

> 若要把「APK 拉域名配置端点（要稳/公网）」与「运营 Console（重/内网）」分到不同机器/规格，用下面的 **方案 B**。两者镜像相同、只是编排不同。

---

## 方案 B（规模化）：分两套独立部署

| # | 服务 | 规格 | 面向 | 作用 |
| --- | --- | --- | --- | --- |
| **A** | **API 服务** | **1C2G** · 公网 | 已安装的 APK | APK 启动拉域名配置（必须稳定可达）+ 后台数据 API |
| **B** | **Console + 打包** | **4C8G** · 运营内部 | 运营人员 | 管理后台 UI + 服务器端打包 + APK 下载 |

```
 已装 APK ──/api/app/config──▶ [A] API 服务 (1C2G)  ◀── 数据API ── [B] Console+打包 (4C8G)
                                  go-api + mysql            web 后台 + build-runner
                                  公网、轻、要稳              运营用、重、出包+下载(/apks)
                                                            keystore 只在这台
```

两套各一个发布包、各自 `pull && up`，互不影响。先上 A，再上 B（B 要填 A 的地址）。

---

## 服务 A：API 服务（1C2G）

**准备**：1C2G Linux + Docker；一个 API 域名（如 `api.example.com`，解析到本机）；该域名 SSL 证书。

```bash
docker login registry.example.com            # 我方给的账号，一次性
unzip hybrid-api-release.zip && cd hybrid-api
cp fullchain.pem privkey.pem ./certs/        # 放证书
vi .env                                       # 只填：DOMAIN=api.example.com
docker compose pull && docker compose up -d   # 拉预构建镜像并启动
curl -fsS https://api.example.com/healthz     # 期望 {"ok":true,...}
```

> DB 口令、`JWT_SECRET` 已预置/首启自动生成，不用动。**首次启动会自动初始化 80 个渠道及其图标/启动页（已随镜像烧入），运维无需导入数据或手动传图**（ADR-0011）。

---

## 服务 B：Console + 打包（4C8G）

**准备**：4C8G Linux + Docker；一个 Console 域名（如 `console.example.com`）；该域名 SSL 证书。（签名 keystore 已内置进镜像，运维无需处理。）

```bash
docker login registry.example.com
unzip hybrid-console-release.zip && cd hybrid-console
cp fullchain.pem privkey.pem ./certs/        # 放证书
vi .env                                        # 填两项：
#   DOMAIN=console.example.com
#   API_BASE_URL=https://api.example.com        ← 指向服务 A
#   （keystore 已内置镜像，无需配置）
docker compose pull && docker compose up -d
# 打开 https://console.example.com ，用我方给的初始账号登录 → 立即改密码。完成。
```

首登后在「域名配置」给三个大渠道填主/备用域名并保存（之后换域名也在这里点）。

---

## 日常运维（基本零负担）

| 场景 | 操作 | 在哪台 |
| --- | --- | --- |
| **换 web 域名**（被封要换） | 后台「域名配置」改主/备用 → 保存。**已装 APK 下次启动自动生效，无需重打包。** | B（后台点） |
| **打包 / 下载 APK** | 「打包中心」选渠道 + 版本号 → 出包；「构建记录」或渠道卡片「下载最新包」下载 | B |
| **升级版本** | `docker compose pull && docker compose up -d`（我方发新镜像后） | A、B 各自 |
| **看日志** | `docker compose logs -f` | A、B |
| **备份**（重要） | 备份服务 A 的数据库卷 + 服务 B 的 `apks` 产物卷 | A、B |
| **重启** | `docker compose restart` | A、B |

---

## 就这些

- 你**不需要**：装 JDK/Android SDK、编译代码、管一堆环境变量。都在镜像里。
- **唯一env特定**的：A 配 API 域名+证书；B 配 Console 域名+证书 + API 地址。（keystore 已内置镜像，运维不碰。）
- 出问题：贴 `docker compose logs` 给我方。常见现象：

| 现象 | 处理 |
| --- | --- |
| 页面打不开 | `docker compose ps` 看是否 Up；证书/域名解析 |
| APK 装上连不上 | 后台「域名配置」确认主/备用域名可达（多半域名被封，换一个即可） |
| 打包一直转/失败 | 服务 B `docker compose logs build-runner`；多半磁盘满或依赖拉取失败 |
| 后台能开但没数据 | 服务 B 的 `API_BASE_URL` 是否指对服务 A；服务 A 是否 Up |

---

> 进阶（镜像构建、keystore 安全模型、全部环境变量、CDN 抗封）属我方职责，见 [06](./06-release.md)。运维无需关心。

---

## 升级到域名 + TLS（拿到域名后）

单机合并方案当前是 **HTTP/IP**（仅供内网调试 Console / 出包）。拿到域名后启用 HTTPS：

1. 域名解析到本机；证书 `fullchain.pem` / `privkey.pem` 放到 `deploy/certs/`。
2. `nginx.allinone.conf.template`：把当前「listen 80」单段，改回「80→301→443 + 443 ssl」两段（模板头注有说明，或复用 `nginx.api.conf.template` 的 TLS 段式）。
3. `docker-compose.allinone.yml` 的 nginx：加回 `- ./certs:/etc/nginx/certs:ro` 挂载与 `"${HTTPS_PORT:-443}:443"` 端口；`.env` 的 `DOMAIN` 改成域名。
4. `docker compose -f docker-compose.allinone.yml up -d`（我方可直接发带 TLS 的新 nginx 镜像，运维仍只 pull+up）。

> ⚠️ APK 的运行时域名配置端点 `/api/app/config` **必须 HTTPS** 才能被 APK 使用（ADR-0003 只认 https）。**正式对 APK 提供服务前务必完成本步**；在此之前 HTTP/IP 仅供内网用 Console + 出包。
