# ADR-0008: 服务器端构建——独立构建机/容器

- **状态**：已采纳(2026-06-24)

## 背景
运营希望在 Web「打包中心」点一下就出包，无需每人本地装 Android SDK + keystore。需决定服务器端构建在哪跑、keystore 怎么放。注意：后端进程本身**不调 Gradle**——打包逻辑在 CLI `hybrid-pack`（ADR-0004）。Linux 可 headless 构建 Android（CI 标准做法）。

## 决策
服务器端构建跑在**独立的构建机/容器**，与常驻 API 进程分离：
- **构建机** = 一个 Docker 镜像（JDK17 + Android SDK：cmdline-tools + `platforms;android-36` + build-tools + `hybrid-pack` 二进制）。
- 后端提供 **build-job 队列**：Web 触发 → 后端入队 → 构建机消费 → 在仓库 checkout 上 `hybrid-pack pull`（拉最新配置→渲染 CSV/res）+ `./gradlew assemble<Flavor>Release` + 签名 → **APK 落到 Console 同服务器的 nginx 静态目录**、日志流回后端、写 `build_record`。
- API 主机保持 1C2G；构建机 4C8G+，可起多台从队列并行取活（水平扩展）。
- **两套独立部署**（运营确认）：① **API 服务** = `go-api` + `mysql`，**1C2G·公网**，供 APK 拉域名配置(`/api/app/config`)与后台数据 API；② **Console + 打包** = web 后台 + `build-runner`，**4C8G·运营用**，**keystore 只在此**。Console 反代/调用 API 服务；build-runner 从 API 领任务、APK 落**本机** nginx `/apks` 供下载（与 Console 同机 → 共享卷）。
- **keystore 烧进 `build-runner` 镜像**（更新 2026-06-24：运营明确「直接内置、不考虑安全」，单租户自托管 + 私有 registry，接受风险）；仍不进 git / DB / 配置 API / 对象存储 / 前端。运维零 keystore 操作。
- **构建产物下载**：nginx 在 **Console 同服务器**以静态路径暴露下载，如 `/apks/<brand>/<flavor>/<versionName>/app-<flavor>-release.apk`；`build_record.apk_urls` 存这些链接，UI「打包中心 / 构建记录」直接给下载按钮。**APK 不走 MinIO / 外部 CDN**（单服务器自托管最简）。构建机与 Console 同机时用**共享卷**直接写入；分机部署时构建后用 rsync/scp 投递到该目录。
- **构建记录与单渠道下载**：每次打包 = 一条 `build_record`，含**任务名**（可填；留空用默认 `<品牌code>-<versionName>-<YYYYMMDD-HHmm>`，如 `ap-1.0.1-20260624-1430`）；每产出一个 APK 记一条 `build_artifact`（flavor / versionName / apk_url / 大小 / 时间）。下载入口两处：① **构建记录页**——按任务列出其全部 APK 下载；② **渠道卡片/详情「下载最新包」**——按 flavor 取该子渠道最近一次成功的 `build_artifact`。

## 理由
- **隔离**：构建吃 CPU/内存/磁盘，独立后不拖累常驻 API。
- **安全**：keystore 收敛到单一受控构建机，攻击面最小。
- **弹性**：构建慢，多 runner 从队列取活即可横向扩容。
- **复用**：构建机就是「跑在服务器上的 `hybrid-pack`」，与本地 CLI 同一套逻辑（ADR-0001/0004），不另造打包器。

## 后果
- ✅ 运营网页一键出包，本地零环境依赖。
- ➖ 多一个部署单元（构建机镜像 + 队列 + 日志流 + nginx 静态产物目录 `/apks`）。
- nginx 需新增静态 location `/apks/`（指向与构建机共享的产物卷），与 `/`(React)、`/api`(Go) 并列。
- ➖ keystore 烧进镜像：镜像仓库必须私有/访问受控（镜像泄露 = 签名密钥泄露）；轮换密钥 = 重建并发布 `build-runner` 镜像。
- 待办（下一轮）：后端加 build-job 队列/状态机/日志流；出 Android 构建 `Dockerfile`；UI 打包中心接服务器构建（进度/日志/下载）。
- **护栏 #4 调整**：keystore 不进 git/DB/配置 API/对象存储/前端，但**烧进 `build-runner` 镜像**（运营接受风险），运维无需处理。

## 备选
- **同机升配**（API 与构建同一台升 4C8G+）：构建抢 API 资源、keystore 与 API 同机、扩容差，被否。
- **复用现成 CI**（GitHub Actions 等）：免维护构建机，但依赖外部 CI、产物回传链路长、keystore 放第三方，自控性差；保留为「后端可选触发 CI」的后续选项，初期不选。
