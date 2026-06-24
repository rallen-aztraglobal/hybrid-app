# hybrid-pack — 渠道中台跨平台打包 CLI

替代 `package.sh` 的跨平台打包工具（Go，单文件零依赖）。把后台统一管理的
「渠道清单 / 图标资源 / 域名配置」**渲染回现有 Gradle 输入**，再跨平台调用 `./gradlew` 打包。
绝不修改 `app/build.gradle` 的构建机制（ADR-0004）。

## 命令

| 命令 | 作用 |
| --- | --- |
| `login --server <URL>` | 登录后台，凭据存 `~/.hybrid-pack/config.json`（0600） |
| `pull [--brand] [--dry-run] [--skip-res]` | 拉 manifest → 重写 `channels/<brand>.csv` + 解压 res + 写每 flavor 的 `assets/bootstrap.json` |
| `build [--brand --channels --test-events --version X.Y.Z]` | 交互式（huh 多选，复刻 package.sh）或非交互；调 `assemble<Cap(flavor)>Release`；`--version` 透传 `-PversionName=` |
| `release --brand <b> --channels <...> [--version X.Y.Z]` | pull → build → 收集 APK → 回传 `POST /api/build/records` |
| `runner [--once --artifact-dir --artifact-base-url --poll]` | **构建机守护（ADR-0008）**：轮询 build-job 队列 → pull + `assemble<Flavor>Release`(+签名) → 落产物 → 回传状态/日志/产物 |
| `status [--brand]` | 本地 CSV 与后台的漂移检测（新增/移除/字段变化 + 本地唯一性冲突） |
| `doctor` | 预检 JDK17 / Android SDK / keystore(local.properties) / 后台连通性 |

### 版本号（`--version`）

`build --version 1.0.1` / `release --version 1.0.1` 透传 `-PversionName=1.0.1` 给 gradlew（沿用
`app/build.gradle` 的 `-PversionName` 机制，X.Y.Z 校验与其一致）。`runner` 用 `job.versionName`。
留空则用 `build.gradle` 默认版本。

### 服务器端构建 `runner`（ADR-0008）

构建机上常驻，循环：领取队列任务 → `pull`（拉最新配置渲染）→ `assemble<Flavor>Release`(+签名)
→ 把 APK 落到 `<artifact-dir>/<brand>/<flavor>/<versionName>/`（默认 `/var/www/apks`，nginx `/apks` 共享卷）
→ 回传状态/日志/产物。

鉴权与地址（优先级：命令行 > 环境 > 登录配置）：`--server`/`HYBRID_PACK_SERVER`、`--token`/`HYBRID_PACK_TOKEN`。

签名（**keystore 只作构建机 secret，绝不上传/绝不打印口令**）：四个环境变量
`HYBRID_PACK_KEYSTORE_FILE` / `_PASSWORD` / `HYBRID_PACK_KEY_ALIAS` / `HYBRID_PACK_KEY_PASSWORD`
齐全 → runner 注入进 `local.properties` 供 Gradle 签名；否则要求构建机已配好 `local.properties`，
两者皆无则拒绝启动（不打无法签名的 release 包）。

build-job 队列契约（CLI 侧，后端下一轮实现匹配端点）：
`POST /api/build/jobs/claim`、`/jobs/{id}/status`、`/jobs/{id}/logs`、`/jobs/{id}/artifacts`。

## 渲染产物（与现状字节级兼容）

```
channels/<brand>.csv                                      ← 保留注释头，flavor|applicationId|palCode|appName
app/src/channels/<brand>/<flavor>/res/...                 ← 解压 res.zip（5 档 mipmap + splash）
app/src/channels/<brand>/<flavor>/assets/bootstrap.json   ← { appId, configUrl, palcode, defaultDomains }
```

- **域名**遵循 ADR-0006：小渠道默认继承品牌默认域名，`useBrandDomains=false` 时用自身覆盖。
- **身份（ADR-0009）**：`applicationId` 派生自 `<品牌包前缀>.<flavor>`（`ap→com.arenaplus`、
  `bp→com.bingoplus`、`gp→com.gamezone`），CSV 与 `bootstrap.json` 的 appId 统一用派生值，
  从构造上杜绝 appId/flavor 不一致脏数据；唯一性以 `flavor` 与 `applicationId` 为准。
- **bootstrap.json（ADR-0002 编译期兜底 + ADR-0009 解析键）**：`appId` 是 APK 拉取域名配置的
  解析键（`GET /api/app/config?appId=`，= `BuildConfig.APPLICATION_ID`）；`palcode` **不再作解析键**，
  仅用于拼加载 URL 的 `/?palcode=` 参数（PAL_CODE 跨品牌可重复、不再全局唯一）。

## 构建与自测

```bash
cd cli
go vet ./...
go build -o bin/hybrid-pack ./cmd/hybrid-pack
GOOS=windows GOARCH=amd64 go build ./...     # 交叉编译验证
go test ./...                                 # 含 CSV 字节级兼容回归
```

## 离线演练（无后台时验证渲染）

设置 `HYBRID_PACK_MANIFEST_DIR=<目录>`，CLI 改从本地 `<brand>.json` 读取 manifest：

```bash
HYBRID_PACK_MANIFEST_DIR=./fixtures hybrid-pack pull --brand ap --dry-run
```

## 安全红线

- keystore 与任何签名口令：本地打包**只存在于 `local.properties`**；服务器端 `runner` 只从
  环境/secret 读取并注入进构建机本地 `local.properties`（ADR-0008）。**绝不写入 CLI 配置、
  绝不进任何上传路径/请求体、绝不打印口令值。**
- 构建记录与产物登记仅含品牌/渠道/状态/版本/产物**文件名 + 下载 URL + 大小 + sha256**/日志摘要，
  不含本地绝对路径、不含任何机密。
- 域名一律走运行时拉取 + 缓存 + `bootstrap.json` 编译期兜底，绝不编译期硬编码（ADR-0002）；
  解析键用 `appId`（ADR-0009）。
