# 渠道中台 · 方案总览与架构

> 面向 hybrid-app 多渠道打包场景的后台管理平台。本文回答「整体怎么搭、后端用什么、数据怎么存、接口怎么设计」。
> 配套文档：[02-域名容灾机制](./02-domain-failover.md)、[03-打包工具与图标管线](./03-build-and-icon-pipeline.md)、[04-开发计划](./04-roadmap.md)、[UI 原型](./ui/index.html)。

---

## 1. 背景与痛点

当前工程已经把「大渠道（品牌）/ 小渠道」抽象得相当干净：

| 维度 | 现状 | 位置 |
| --- | --- | --- |
| 大渠道 | `ap` / `bp` / `gp` 三个品牌，各自的 `domain / scheme / hms` | `app/build.gradle` 的 `brandConfig` |
| 小渠道 | 一行一个，`flavor\|包名\|PAL_CODE\|应用名` | `channels/<brand>.csv` |
| 资源 | 每渠道一套 `mipmap-*/ic_launcher.png` + `splash_fullscreen.png` | `app/src/channels/<brand>/<flavor>/res` |
| 打包 | 交互式 Bash 脚本 | `package.sh` |
| URL | 编译期写入 `BuildConfig.DOMAIN` / `PAL_CODE`，启动加载 `${domain}/?palcode=${PAL_CODE}` | `WebViewActivity.kt:187` |

痛点集中在三处：

1. **域名经常变**：域名一旦写进 `BuildConfig` 就被「焊死」在 APK 里。域名被封 → 必须重新打包 + 重新分发**所有**已发布渠道，代价极高。
2. **渠道纯靠手工维护 CSV**：易错。实测 CSV 里已经出现包名重复（`ap01035 → com.arenaplus.ap01034`、`gzmarket062 → com.gamezone.gzmarket066`），这类脏数据会导致两个渠道覆盖安装、归因错乱。
3. **图标/资源靠人肉切图摆目录**：每个渠道要手动产出 5~6 档密度的 icon，没有校验、没有统一裁剪。

**目标**：做一个后台，把「渠道清单 + 图标资源 + 域名配置」收归为唯一数据源（single source of truth），本地打包脚本从后台拉取，APK 运行时也能从后台拉取域名 —— 让「改域名」从「重新打包全部」退化为「后台点一下」。

---

## 2. 整体架构

```
┌──────────────────────────────────────────────────────────────────────┐
│                          渠道中台 Admin (React 18)                      │
│   3 Tab(大渠道) · 渠道CRUD · Xcode式图标管线 · 域名配置 · 打包中心        │
└───────────────┬──────────────────────────────────────────────────────┘
                │ HTTPS / JWT
┌───────────────▼──────────────────────────────────────────────────────┐
│                      后端 API (Go · Echo · 单文件静态二进制)             │
│  ┌─────────┐ ┌──────────┐ ┌──────────┐ ┌───────────┐ ┌──────────────┐ │
│  │ 渠道服务 │ │ 图标管线  │ │ 域名服务  │ │ 构建编排   │ │ 运行时配置下发 │ │
│  │ (CRUD)  │ │(imaging) │ │(健康巡检) │ │(记录/产物) │ │ (公开,高可用)  │ │
│  └────┬────┘ └────┬─────┘ └────┬─────┘ └─────┬─────┘ └──────┬───────┘ │
└───────┼───────────┼────────────┼─────────────┼──────────────┼─────────┘
        │           │            │             │              │
   ┌────▼────┐ ┌────▼─────┐      │        ┌────▼─────┐   ┌────▼──────────┐
   │  MySQL  │ │ 对象存储  │      │        │ 本地打包  │   │  APK 客户端     │
   │ (元数据) │ │(图标/APK) │      │        │  CLI     │   │ (WebViewActivity)│
   └─────────┘ └──────────┘      │        └────┬─────┘   └────┬──────────┘
                                 │             │              │
                                 └─ 定时探测域名 │              │
                                               │              │
                       ① pull 配置+资源 → 写 CSV/res ┘          │
                       ② gradlew assemble<Flavor>Release        │
                       ③ 回传构建记录/产物                        │
                                                                │
                       启动时 GET /api/app/config?palcode=… ─────┘
                       (拿到最新主+备用域名，与编译期兜底合并)
```

三条数据流：

- **配置下行（管理面）**：后台 → MySQL/对象存储。
- **打包链路（CI/本地）**：CLI 从后台 `pull` 最新渠道配置与资源 → 落地成现有工程认识的 `channels/*.csv` + `res/` → 调 `gradlew` 打包 → 回传记录。**关键：不改动现有 Gradle 构建逻辑**，CLI 只是「更聪明、跨平台、连后台」的 `package.sh` 替代品。
- **运行时下行（APK）**：APK 启动用不变的 `PAL_CODE` 向后台公开端点拉最新域名清单。这是解决「域名经常变」的核心 —— 详见 [02-域名容灾机制](./02-domain-failover.md)。

---

## 3. 技术选型

### 3.1 后端 & CLI：Go（按体积要求选定）

> Node 体积是硬伤：NestJS 的 `node_modules` 常 150–300MB，CLI 用 `pkg`/`bun --compile` 编译的单文件 45–90MB。改用 **Go**，针对「小体积 + 跨平台 CLI 分发」是最优解。

选 Go 的收益：

1. **后端编译成单个静态二进制 ~10–20MB**，服务器不装任何运行时，内存占用 15–40MB，Docker 镜像可做到 ~20MB（scratch/distroless 基底）。
2. **CLI 交叉编译**到 Win/macOS/Linux 各一个 ~5–15MB、零依赖二进制（`GOOS/GOARCH go build`），目标机无需装运行时——这点比 Node 更适合做分发给同事的打包工具。
3. **图标多尺寸生成用纯 Go 的 [`disintegration/imaging`](https://github.com/disintegration/imaging)**（缩放/裁剪/圆形遮罩/留边）即可满足需求 ②，保持二进制全静态、无 cgo；若要更高画质再上 `h2non/bimg`（libvips 绑定，但需 cgo、放弃全静态）。
4. **server 与 CLI 共享 Go 类型**：`go.work` 工作区里把 manifest / 域名结构体放进共用 internal 包，跨进程一份定义不漂移。

代价：对前端不再天然共享 TS 类型——用 **OpenAPI（`swag`/`oapi-codegen` 从 Go 注解生成 spec）→ `openapi-typescript` 生成 React 的 TS 客户端**补回来，类型安全跨 HTTP 依然成立。

最终技术栈：

| 层 | 选型 | 备注 |
| --- | --- | --- |
| 前端 | React 18 + TypeScript + Vite | 不变 |
| UI 库 | **shadcn/ui + Tailwind**（或 Ant Design 5） | 中台表单密集；原型用 shadcn 风格（见 UI） |
| 状态/请求 | TanStack Query + Zustand | 服务端/本地状态分离 |
| 图标裁剪(前端) | `react-easy-crop` | 客户端方形裁剪，所见即所得 |
| API 客户端 | `openapi-typescript` 从 Go 的 OpenAPI 生成 | 补回跨端类型安全 |
| **后端** | **Go + Echo**（或 Gin / Chi） | 单静态二进制；JWT/CORS 中间件齐全 |
| DB 层 | **GORM**（开发快、AutoMigrate）或 `sqlc`（原生 SQL 类型安全） | MySQL 驱动 `go-sql-driver/mysql` |
| 迁移 | `golang-migrate` | 版本化迁移 |
| 图像处理 | `disintegration/imaging`（纯 Go） | 多密度生成，全静态无 cgo |
| 鉴权 | `golang-jwt` + Echo 中间件 + RBAC | 角色 admin/operator/viewer |
| 定时巡检 | `robfig/cron`（进程内，无需 Redis） | 域名健康巡检 |
| 上传/校验 | Echo 表单 + `go-playground/validator` | 字段校验 |
| 对象存储 | MinIO Go SDK / 云 OSS·S3 SDK | 图标主图、资源 zip、APK 产物 |
| **打包 CLI** | **Go + Cobra + `charmbracelet/huh`** | 交叉编译；交互式多选/进度 |
| 部署 | Docker Compose（go-api + MySQL + MinIO + Nginx） | 镜像极小 |

### 3.2 核心架构决策：运行时远程配置 + 编译期兜底

这是整个方案最重要的一个决定，直接决定「域名经常变」是否被根治。

| 方案 | 域名改了之后… | 评价 |
| --- | --- | --- |
| **A. 纯编译期**（现状）：域名写进 BuildConfig | 必须重打 + 重发**所有** APK | ❌ 对「经常变」是灾难 |
| **B. 纯运行时**：APK 启动永远从后台拉域名 | 后台改一下，所有已装 APK 下次启动生效 | ✅ 但首启依赖网络，配置端点本身被封则全崩 |
| **C. 运行时 + 编译期兜底（推荐）** | 后台改一下即全网生效；拉取失败时用打包时烧录的默认域名兜底 | ✅✅ 兼顾「热更新」与「离线可用」 |

**推荐 C**。域名按「**实时拉取 + 自更新本地缓存 + 编译期兜底**」三级取用（与你确认的一致：启动调一次接口、有兜底、成功即更新兜底）：

- **① 实时拉取（每次启动调用一次）**：APK 启动用 `PAL_CODE` 调一次 `GET /api/app/config?palcode=…`，拿**最新**主域名 + 最多 3 个备用域名。**域名随时在后台改、随时生效**——已安装包下次启动即用新值，完全不依赖打包。
- **② 自更新本地缓存**：接口**成功返回就把结果写入本地缓存**（`SharedPreferences`/文件，覆盖上一次）。于是兜底数据始终是「最近一次成功的配置」，而非陈旧的打包快照。
- **③ 编译期兜底（仅首启）**：只有「**从未成功拉取过**」（首次安装即无网）时，才用打包时由 CLI 烧录进 `assets/bootstrap.json` 的默认清单。一旦成功拉取过一次，缓存 ② 就接管兜底角色。
- **不变量**：只有 `PAL_CODE`（渠道身份，永不变）和配置服务地址是编译期固定的；域名一律走运行时。

> 完整取用与容灾流程见 [02 · STEP 1](./02-domain-failover.md)。

> **配置服务的高可用是关键**：游戏业务域名（arenaplus.ph 等）经常被封，但**配置端点不能跟着一起死**。建议把 `GET /api/app/config` 放在**与业务域名分离的高可用基础设施**上 —— 例如对象存储/CDN 上的一份静态 JSON（后台每次保存时再生成并刷新 CDN），或一个不易被封的独立域名，甚至烧录多个配置端点轮询。这样「业务域名全被封」时，APK 仍能从配置端点拿到「最新的、换好的」业务域名。

> ⚠️ 这是对你需求 ④ 的一处**增强**（你原文只要求「APK 启动按主→备用顺序加载」）。运行时拉取让「域名经常变」从「重新打包」变成「后台点一下」，收益巨大，但引入了「配置端点高可用」这一新责任。**如果你只想要最小改动（域名仍编译期烧录，仅做主→备用容灾），也可以只取方案的容灾部分**——这一点我列在文末「需你拍板的决策」里。

---

## 4. 数据模型（MySQL）

实体关系：

```
brand 1───* channel 1───* channel_domain          admin_user 1───* audit_log
  │                              ▲                     1───* build_record
  └──* brand_domain (品牌级默认域名，渠道可继承/覆盖)
```

设计要点：

- **域名优先放在「大渠道」级别**。现实里域名被封是「整个品牌一起换」（arenaplus.ph 挂了，所有 `ap` 渠道一起换），所以默认让小渠道**继承大渠道的域名清单**，只在少数需要差异化时才在小渠道级覆盖。这让「换域名」真正变成改一处生效一片。
- **唯一性约束**消灭脏数据：`application_id`、`pal_code`、`(brand_id, flavor_name)` 全局唯一。
- **软删除**：渠道用 `status` 标记 `archived`，不物理删除，保留归因历史。

```sql
-- 大渠道（品牌）
CREATE TABLE brand (
  id           BIGINT PRIMARY KEY AUTO_INCREMENT,
  code         VARCHAR(16)  NOT NULL UNIQUE,         -- ap / bp / gp
  name         VARCHAR(64)  NOT NULL,                -- ArenaPlus
  scheme       VARCHAR(32)  NOT NULL,                -- deeplink scheme: gzone/bingo
  hms_enabled  TINYINT(1)   NOT NULL DEFAULT 0,      -- 是否集成 HMS/OAID
  accent_color VARCHAR(16)  DEFAULT NULL,            -- 后台 Tab 主题色
  sort         INT          NOT NULL DEFAULT 0,
  created_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 品牌级默认域名（小渠道默认继承）
CREATE TABLE brand_domain (
  id        BIGINT PRIMARY KEY AUTO_INCREMENT,
  brand_id  BIGINT NOT NULL,
  position  TINYINT NOT NULL,                        -- 0=主域名, 1..3=备用
  url       VARCHAR(255) NOT NULL,                   -- https://arenaplus.ph
  enabled   TINYINT(1) NOT NULL DEFAULT 1,
  UNIQUE KEY uk_brand_pos (brand_id, position),
  CONSTRAINT fk_bd_brand FOREIGN KEY (brand_id) REFERENCES brand(id)
);

-- 小渠道
CREATE TABLE channel (
  id                 BIGINT PRIMARY KEY AUTO_INCREMENT,
  brand_id           BIGINT       NOT NULL,
  flavor_name        VARCHAR(64)  NOT NULL,          -- ap01018 (gradle flavor 名)
  application_id     VARCHAR(128) NOT NULL UNIQUE,   -- com.arenaplus.ap01018
  pal_code           VARCHAR(64)  NOT NULL UNIQUE,   -- 1053259232660520961
  app_name           VARCHAR(128) NOT NULL,          -- 桌面应用名
  status             ENUM('enabled','disabled','archived') NOT NULL DEFAULT 'enabled',
  use_brand_domains  TINYINT(1)   NOT NULL DEFAULT 1, -- 是否继承品牌默认域名
  icon_master_url    VARCHAR(255) DEFAULT NULL,      -- 上传的 1024 主图
  icon_set_url       VARCHAR(255) DEFAULT NULL,      -- 生成的多密度资源 zip
  splash_url         VARCHAR(255) DEFAULT NULL,      -- splash_fullscreen 源图
  remark             VARCHAR(255) DEFAULT NULL,
  created_by         BIGINT       DEFAULT NULL,
  created_at         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at         DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_brand_flavor (brand_id, flavor_name),
  CONSTRAINT fk_ch_brand FOREIGN KEY (brand_id) REFERENCES brand(id)
);

-- 小渠道级域名覆盖（use_brand_domains=0 时生效）
CREATE TABLE channel_domain (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  channel_id BIGINT NOT NULL,
  position   TINYINT NOT NULL,                       -- 0=主域名, 1..3=备用
  url        VARCHAR(255) NOT NULL,
  enabled    TINYINT(1) NOT NULL DEFAULT 1,
  UNIQUE KEY uk_ch_pos (channel_id, position),
  CONSTRAINT fk_cd_channel FOREIGN KEY (channel_id) REFERENCES channel(id)
);

-- 域名健康巡检结果（后台定时探测，仅作监控展示，不直接决定 APK 行为）
CREATE TABLE domain_health (
  id           BIGINT PRIMARY KEY AUTO_INCREMENT,
  url          VARCHAR(255) NOT NULL,
  status       ENUM('ok','down','hijacked','unknown') NOT NULL DEFAULT 'unknown',
  http_code    INT          DEFAULT NULL,
  latency_ms   INT          DEFAULT NULL,
  checked_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_url_time (url, checked_at)
);

-- 构建记录（CLI 回传）
CREATE TABLE build_record (
  id           BIGINT PRIMARY KEY AUTO_INCREMENT,
  brand_code   VARCHAR(16)  NOT NULL,
  flavors      JSON         NOT NULL,                -- ["ap01018","ap01034"]
  test_events  TINYINT(1)   NOT NULL DEFAULT 0,
  status       ENUM('running','success','failed') NOT NULL,
  operator     VARCHAR(64)  DEFAULT NULL,
  version_name VARCHAR(32)  DEFAULT NULL,
  apk_urls     JSON         DEFAULT NULL,            -- 回传的产物地址
  log_excerpt  TEXT         DEFAULT NULL,
  started_at   DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  finished_at  DATETIME     DEFAULT NULL
);

-- 后台账号 & 审计
CREATE TABLE admin_user (
  id            BIGINT PRIMARY KEY AUTO_INCREMENT,
  username      VARCHAR(64) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  role          VARCHAR(16) NOT NULL DEFAULT 'user',  -- 只有 admin / user 两档（迁移 000010 收敛）
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE audit_log (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id    BIGINT,
  action     VARCHAR(64) NOT NULL,                   -- channel.create / domain.update ...
  target     VARCHAR(128),
  detail     JSON,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

---

## 5. API 设计（REST）

约定：除 `/api/app/*`（APK 公开消费）外，全部需要 `Authorization: Bearer <jwt>`；返回 `{ code, message, data }`。

### 5.1 鉴权
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/auth/login` | 账号密码 → access+refresh token |
| POST | `/api/auth/refresh` | 刷新 token |

### 5.2 大渠道
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/brands` | 三个大渠道列表（含渠道计数、accent 色） |
| GET | `/api/brands/:code/domains` | 品牌默认域名清单 |
| PUT | `/api/brands/:code/domains` | 更新品牌默认域名（主+最多3备用，**校验**见下） |

### 5.3 小渠道（CRUD —— 需求 ②）
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/channels?brand=ap&status=enabled&q=keyword` | 分页/搜索/筛选 |
| POST | `/api/channels` | 新增（名字/包名/PAL_CODE/flavor，**唯一性校验**） |
| GET | `/api/channels/:id` | 详情 |
| PUT | `/api/channels/:id` | 修改 |
| DELETE | `/api/channels/:id` | 软删除（置 archived） |
| PUT | `/api/channels/:id/domains` | 设置小渠道域名覆盖 / 切回继承品牌 |

### 5.4 图标管线（需求 ②）
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/channels/:id/icon` | 上传裁剪后的主图（1024²）→ `imaging` 生成全部密度 → 返回各 slot 预览 URL |
| PUT | `/api/channels/:id/icon/:slot` | 单独覆盖某一档（如只想替换 xxxhdpi） |
| POST | `/api/channels/:id/splash` | 上传 splash 源图 |
| GET | `/api/channels/:id/res.zip` | 打 zip 的整套 res 资源（CLI 下载用） |

### 5.5 打包编排（需求 ③，供 CLI 调用）
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/build/manifest?brand=ap` | 一次拉全：该品牌所有启用渠道 + 域名 + PAL_CODE + 资源 zip 地址。**CLI 据此重写 `channels/ap.csv` 与 res** |
| POST | `/api/build/records` | CLI 回传构建结果（状态/产物/日志摘要） |
| GET | `/api/build/records?brand=ap` | 构建历史 |

### 5.6 运行时配置（需求 ④，APK 公开消费，高可用）
| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/app/config?palcode=<PAL_CODE>` | 返回该渠道**最新**域名清单 + 探测端点。**强缓存 + 部署在抗封基础设施**，详见 [02](./02-domain-failover.md) |

`GET /api/app/config` 响应示例：
```json
{
  "palcode": "1053259232660520961",
  "domains": [
    "https://arenaplus.ph",
    "https://arenaplus-cdn.com",
    "https://ap-backup.net"
  ],
  "probePath": "/healthz",
  "configVersion": 42,
  "ttlSeconds": 600
}
```

### 5.7 域名校验规则（防止「乱填导致 APK 乱换」）
后端在保存域名时强制校验，这是 APK 端容灾可靠的前提：
- 必须 `https://`（`usesCleartextTraffic` 虽开着，但线上一律 https）；
- 必须是合法域名 / 可解析；
- 主域名必填，备用 0~3 个，去重；
- 保存时**实时探测一次**，给出健康提示（不通也允许保存，但红色告警）；
- 保存成功 → 触发重新生成 `/api/app/config` 的 CDN 静态快照。

---

## 6. 与现有 Android 工程的契合（低风险接入）

这套方案刻意做到**不推翻现有构建**：

- 后台是 `channels/*.csv` 的「上游编辑器」。CLI `pull` 时把后台数据**渲染成现在格式完全一致的 CSV**（甚至保留注释头），`app/build.gradle` 的 `loadChannels`/`productFlavors` 一行不用改。
- 图标资源仍然落到 `app/src/channels/<brand>/<flavor>/res`，目录结构不变。
- 唯一需要改 Android 代码的是 [WebViewActivity.kt:187](../../app/src/main/java/com/hybrid/android/WebViewActivity.kt#L187) 那一句 `loadUrl(...)` —— 换成「域名解析器 + 容灾」，且天然契合现有 `BrandStrategy`/`BrandHost` 插件架构。详见 [02](./02-domain-failover.md)。

---

## 7. 部署拓扑（起步）

```
单台 Linux (1C2G 即可) + Docker Compose
├── nginx           反代 + TLS，/api → Go API，/ → React 静态
├── go-api          后端（单静态二进制，镜像 ~20MB，scratch/distroless 基底）
├── mysql 8         元数据
└── minio           对象存储（图标/资源zip/APK）
（无需 Redis：域名健康巡检用进程内 cron）

配置端点 /api/app/config 另挂 CDN：
  后台保存 → 生成静态 config-<palcode>.json → 推 CDN/对象存储（抗封、全球可达）
```

---

## 8. 需你拍板的决策

| # | 决策 | 推荐 | 影响 |
| --- | --- | --- | --- |
| 1 | 后端语言 | **Go**（已选定） | 体积最优；放弃前端共享类型，改用 OpenAPI 生成 TS 客户端补回 |
| 2 | 域名下发模式 | **实时拉取 + 自更新缓存 + 编译期兜底**（已确认） | 域名随时改随时生效；启动调一次接口，成功即更新本地兜底，见 [02 STEP1](./02-domain-failover.md) |
| 3 | 域名配置粒度 | **大渠道级默认 + 小渠道可覆盖** | 决定 UI 域名页的层级 |
| 4 | 对象存储 | 自建 MinIO / 云 OSS | 看你是否已有云资源 |
| 5 | 打包 CLI 形态 | Go CLI（交叉编译单文件二进制） | 见 [03](./03-build-and-icon-pipeline.md) |

> 默认我按「推荐」列推进。如需调整，告诉我对应编号即可。
