# server · 渠道中台后端（Go + Echo）

渠道中台的后端 API：渠道 CRUD（含唯一性校验）、图标 fan-out 管线、域名配置（品牌默认 + 渠道覆盖）、
运行时配置下发（`/api/app/config`）、构建 manifest（供 CLI）、JWT + RBAC、域名巡检。

> 依据：根 `CLAUDE.md`、`docs/admin/01-architecture.md`、`docs/admin/03-build-and-icon-pipeline.md`、`docs/adr/0002~0006`。

## 本机零配置启动

默认用纯 Go 的 sqlite（无 cgo）+ 本地磁盘对象存储，开箱即跑：

```bash
cd server
go run ./cmd/server            # 启动，默认 :8080，自动建表 + seed 三个品牌 + 初始 admin
```

首启会创建管理员 `admin / admin12345`（可用 `BOOTSTRAP_ADMIN=user:pass` 覆盖）。

### 把现有 CSV 导入（appId 按 flavor 派生，ADR-0009）

```bash
go run ./cmd/server import      # 读取 ../channels/*.csv；applicationId 一律按 <品牌包前缀>.<flavor> 派生
```

- CSV 的 `applicationId` 列**不被信任**，仅用于与派生值比对；不一致则记为「修正」并按派生值导入。
- 历史两条 mismatch（`ap01035`→`com.arenaplus.ap01035`、`gzmarket062`→`com.gamezone.gzmarket062`）
  在派生规则下自动修正为独立渠道导入，**不再跳过**（共导入 80 条，跳过 0）。
- `pal_code` 不再全局唯一（ADR-0009），允许跨品牌/同品牌复用，不参与查重；唯一性只看 `applicationId` 与 `(brand, flavor)`。

## 自测

```bash
go vet ./...
go build ./...
go test ./...
```

## 生产配置（环境变量）

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `SERVER_ADDR` | `:8080` | 监听地址 |
| `DB_DRIVER` | `sqlite` | `mysql` / `sqlite` |
| `DB_DSN` | sqlite 文件 | mysql: `user:pass@tcp(host:3306)/db?charset=utf8mb4&parseTime=True&loc=Local` |
| `DB_AUTOMIGRATE` | `true` | 生产建议关，改用 `migrations/` 的 golang-migrate |
| `STORAGE_KIND` | `local` | `local` / `minio` |
| `STORAGE_LOCAL_DIR` | `./data/objects` | local 磁盘根 |
| `STORAGE_PUBLIC_URL` | `http://localhost:8080/static` | 对象对外前缀（minio 下指向 CDN） |
| `MINIO_ENDPOINT/ACCESS_KEY/SECRET_KEY/BUCKET/USE_SSL` | — | MinIO 连接 |
| `JWT_SECRET` | dev 占位 | **生产必须改** |
| `APP_PROBE_PATH` | `/healthz` | `/api/app/config` 返回的探测路径 |
| `APP_CONFIG_TTL_SECONDS` | `600` | 运行时配置缓存秒数 |
| `DOMAIN_PROBE_ENABLE` | `true` | 进程内 cron 域名巡检（每 5 分钟） |

## 关键端点

- 公开：`GET /healthz`（返回 `{"ok":true,"brand":"<brand>","v":1}`）、`GET /api/app/config?appId=<applicationId>`（强缓存；解析键为 applicationId，ADR-0009）。
- 鉴权：`POST /api/auth/login|refresh`。
- 管理面（JWT + RBAC）：`/api/brands*`、`/api/channels*`、`/api/channels/:id/icon|splash|res.zip`、`/api/channels/:id/latest-apk`。
- 打包（ADR-0008）：
  - `GET /api/build/manifest?brand=`（供 CLI 拉全量）。
  - `POST /api/build/jobs`（入队：`brand,flavors,versionName(X.Y.Z),testEvents,name?`；状态机 queued→running→success/failed）。
  - `GET /api/build/records[?brand=&status=&limit=]`、`GET /api/build/records/:id`（含 APK 产物）。
  - `GET /api/build/records/:id/logs?offset=`（分段/流式日志，轮询 `next` 直到 `done`）。
  - runner（构建机）：`POST /api/build/claim`（原子领取 queued）、`POST /api/build/records/:id/status`、
    `POST /api/build/records/:id/logs`（text/plain 增量追加）、`POST /api/build/records/:id/artifacts`。
- OpenAPI：`/swagger/index.html`，spec `/swagger/doc.json`（前端用 `openapi-typescript` 生成 TS 客户端）。

OpenAPI 注解改动后重新生成：

```bash
swag init -g cmd/server/main.go -o internal/docs --parseDependency --parseInternal
```

## 设计要点（护栏）

- **身份与解析键（ADR-0009）**：`applicationId` **派生**自 `<品牌包前缀>.<flavor>`（`brand.package_prefix`：ap→`com.arenaplus` / bp→`com.bingoplus` / gp→`com.gamezone`），不手填、不信任输入；运行时配置解析键改为 `appId`。
- **域名绝不编译期硬编码**：运行时 `/api/app/config?appId=` 下发 + APK 自更新缓存 + 编译期 `bootstrap.json` 兜底（ADR-0002，键由 palcode 更正为 appId）。
  保存域名时强制 https/可解析/≤4/去重 校验（ADR-0003 前提）；保存即生成 CDN 静态快照（按 appId 命名）。
- **唯一性拦截**：仅 `applicationId`（= 派生）与 `(brand, flavor)` 唯一，重复必被拒；**`pal_code` 不再全局唯一**（跨品牌可复用），仅作 `/?palcode=` URL 参数与编译期烧录。
- **keystore 永不进后台**：本服务不接收、不存储任何签名密钥；构建机产物（APK）以 nginx 静态 URL 记于 `build_artifact`，不走对象存储（ADR-0008）。
- **图标单图 fan-out**：1024² 主图 → 5 档方形/圆形/自适应前景 + `anydpi-v26` xml + 背景色 → `res.zip`，
  目录结构与 `app/src/channels/<brand>/<flavor>/res` 字节级兼容（ADR-0005）。
