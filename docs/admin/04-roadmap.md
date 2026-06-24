# 开发计划与里程碑

> 工作量以「人天」估算（1 名全栈 + 0.5 名 Android）。可按团队规模并行压缩。MVP 目标：**4~5 周**跑通「后台管理 → CLI 打包 → APK 域名容灾」主链路。

---

## 里程碑总览

| 阶段 | 目标 | 工期 | 交付 |
| --- | --- | --- | --- |
| M0 脚手架 | 仓库/CI/部署骨架 | 3d | monorepo + Docker Compose 起得来 |
| M1 后端地基 | 数据模型 + 鉴权 + 渠道 CRUD API | 5d | 接口可联调 |
| M2 前端管理 | 3 Tab + 渠道列表 + 增删改 | 5d | 能管渠道 |
| M3 图标管线 | 上传/裁剪/多密度生成/九宫格 | 4d | 传 1 张出全套 |
| M4 域名 + 运行时配置 | 域名 CRUD + 校验 + `/api/app/config` + CDN 快照 | 3d | 域名可热更 |
| M5 打包 CLI | pull/build/release 跨平台 | 5d | Win/macOS 可打包 |
| M6 APK 容灾 | DomainResolver + 错误页 + 运行时拉配置 | 4d | 主→备用容灾上线 |
| M7 收尾 | 审计/构建记录/域名巡检/权限/打磨 | 4d | 可上生产 |
| **合计** | | **≈ 33 人天（含缓冲约 6~7 周）** | |

> MVP 可裁掉 M7 的巡检/审计与 M3 的单槽位覆盖，**核心链路 M0–M6 约 29 人天**。

---

## 分阶段明细

### M0 · 脚手架（3d）
- 仓库结构 + `go.work`：`server/`(Go+Echo)、`cli/`(Go+Cobra)、`web/`(React18+Vite)；`server` 与 `cli` 共享 `internal` 类型；React 的 TS 客户端由 OpenAPI 生成。
- Docker Compose：mysql8 + minio + go-api + nginx。
- CI：`go vet`/`go test` + 前端 lint/build + 交叉编译产物。
- **验收**：`docker compose up` 起得来，前后端 hello world 互通。

### M1 · 后端地基（5d）
- 用 GORM 模型 + `golang-migrate` 落地 [01 §4](./01-architecture.md) 全部表 + seed（3 个 brand）。
- JWT（`golang-jwt`）+ Echo 中间件 RBAC + `admin_user`。
- 渠道 CRUD API（[01 §5.3](./01-architecture.md)）+ DB 唯一约束 & `validator/v10` **唯一性校验**（包名/PAL_CODE/flavor）。
- 把现有 3 个 CSV 写**一次性导入命令**灌进库（顺带清洗包名重复脏数据）。
- **验收**：跑通渠道增删改查；重复包名被拒。

### M2 · 前端管理（5d）
- 应用骨架：侧边栏 + 顶栏 + 路由 + TanStack Query 封装 + 登录页。
- **3 大渠道 Tab**（需求 ①）+ 渠道列表（卡片/表格切换、搜索、筛选、分页）。
- 新增/编辑渠道抽屉：基础字段表单 + 校验联动后端。
- 删除（软删）、状态切换。
- **验收**：UI 上完成渠道全生命周期管理。

### M3 · 图标管线（4d）
- 前端 `react-easy-crop` 方形裁剪 + **九宫格槽位预览**（需求 ②，见 [03 §4](./03-build-and-icon-pipeline.md)）。
- 后端 `imaging` fan-out（纯 Go）全密度 + 圆形遮罩 + 自适应前景 + `anydpi-v26` xml + 打 res.zip 入对象存储。
- splash 单独上传。
- 单槽位覆盖（可后置）。
- **验收**：传 1 张 1024 主图 → 下载 res.zip → 解压目录与现有 `app/src/channels/.../res` 结构一致。

### M4 · 域名 + 运行时配置（3d）
- 品牌级默认域名 + 小渠道覆盖 UI/API（[01 §5.2/5.3](./01-architecture.md)）。
- **域名校验**（https/可解析/数量/去重/保存即探测，[01 §5.7](./01-architecture.md)）。
- `GET /api/app/config?palcode=…` + 保存后生成 CDN 静态快照（[02](./02-domain-failover.md)）。
- **验收**：改域名 → `curl /api/app/config?palcode=…` 立刻反映新值。

### M5 · 打包 CLI（5d）
- `login/pull/build/release/status/doctor`（[03 §3](./03-build-and-icon-pipeline.md)）。
- `pull` 渲染 CSV/res/bootstrap.json，与现有 Gradle **零改动**兼容。
- `build` 跨平台调 `gradlew`/`gradlew.bat`，`charmbracelet/huh` 交互式多选复刻 `package.sh` 体验。
- `GOOS/GOARCH go build` 交叉编译出 Win/macOS/Linux 单文件二进制。
- **验收**：在 Windows 与 macOS 各打出同一渠道 Release APK，安装可跑。

### M6 · APK 容灾（4d，Android）
- `DomainResolver`（STEP0~3 + 实时拉取/自更新缓存/编译期兜底 + 中立探针 + 证书/业务校验，[02 §4](./02-domain-failover.md)）。
- 替换 [WebViewActivity.kt:187](../../app/src/main/java/com/hybrid/android/WebViewActivity.kt#L187) 的 `loadUrl`。
- 原生错误页（两种文案）+ 刷新 + 网络恢复自动重试。
- 运行中主框架错误防抖容灾。
- 后端加 `/healthz` 健康端点（业务站点侧）。
- **验收**：①断网→「网络异常」不换域名；②主域名 DNS 污染但有公网→自动切备用；③全封但有公网→「服务不可用」；④边加载边断网→防抖切换。

### M7 · 收尾（4d）
- 审计日志 + 构建记录页 + 域名健康巡检（`robfig/cron` 进程内定时）+ 看板红绿灯。
- 角色权限细化（viewer 只读）。
- 空态/加载/错误/响应式打磨，暗色模式（可选）。
- **验收**：生产部署 checklist 过。

---

## 并行与依赖

```
M0 ──┬─▶ M1 ──┬─▶ M2 ─▶ M3
     │        ├─▶ M4 ─────────┐
     │        └─▶ M5(依赖M4的manifest) ─┐
     └────────────────────────────────┴─▶ M6(依赖M4运行时配置) ─▶ M7
```
- M2/M3/M4 在 M1 后可并行（前端 vs 后端 vs 域名）。
- M6（Android）依赖 M4 的 `/api/app/config`，但 `DomainResolver` 骨架可用 mock 提前写。
- M5（CLI）依赖 M4 的 `/api/build/manifest`。

---

## 风险与对策

| 风险 | 影响 | 对策 |
| --- | --- | --- |
| 配置端点 `/api/app/config` 也被封 | APK 拿不到最新域名 | 部署在抗封 CDN/对象存储；烧录多个配置端点；编译期默认域名兜底 |
| 中立探针端点（gstatic）在某些地区不可达 | A/B 误判 | 多端点取「任一成功」；可后台下发探针清单 |
| 域名被劫持返回假 200 | 误判「命中」加载到广告页 | 业务特征校验 + TLS 证书域名校验（[02 §2 STEP2](./02-domain-failover.md)） |
| 现有 CSV 脏数据（包名重复） | 导入失败/覆盖安装 | M1 导入脚本清洗 + 唯一约束兜底 |
| keystore 安全 | 签名泄露 | 密钥永不进后台，仅留本地 `local.properties` |
| 大量渠道（gp 已 41 个）资源体积 | res.zip 下载慢 | CDN + 增量 pull（仅拉变更渠道） |

---

## 验收基线（Definition of Done）

1. 后台三 Tab 管理三大渠道下全部小渠道，增删改 + 唯一性校验通过。
2. 新增渠道传 1 张主图，自动生成全密度图标，CLI 拉下来目录结构与现状一致。
3. 后台改域名 → 已安装 APK 下次启动加载到新域名，**无需重新打包**。
4. APK 在「断网 / 域名被封但有网 / 全封」三种场景下文案与切换行为符合 [02 §6](./02-domain-failover.md) 表格。
5. Windows 与 macOS 均可通过 CLI 完成 `pull → build`。
