# 渠道中台 · 运维部署手册（极简）

> 给运维。**镜像由我方 CI 预先构建好，你不用编译、不用改代码。** 每套服务都只是：**放证书 → 填域名 →  `pull && up`**。
> 镜像怎么构建/发布是我方的事，见 [06 · 镜像发布（我方）](./06-release.md)、[ADR-0010](../adr/0010-delivery-prebuilt-images.md)。

---

## 总览：分两套独立部署

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
