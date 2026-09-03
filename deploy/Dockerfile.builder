# ============================================================================
# deploy/Dockerfile.builder —— 服务器端 Android 构建机（ADR-0008 / docs/admin/06 §4）
#
# = 「跑在服务器上的 hybrid-pack」：JDK17 + Android SDK（cmdline-tools +
#   platforms;android-36 + build-tools）+ hybrid-pack 二进制，入口跑 build-job runner。
# 与常驻 API 进程分离，吃 CPU/内存/磁盘也不拖累 API；可起多台从队列并行取活。
#
# keystore（更新 2026-06-24，ADR-0008/0010 + docs/admin/06 §4 + 护栏 #4）：
#   运营明确「直接内置、不考虑安全」（单租户自托管 + 私有 registry，接受风险）。
#   → 本镜像【构建期烧入】签名 keystore + 写好 local.properties：
#       * COPY secrets/release.keystore → /opt/hybrid/release.keystore（固定镜像路径）。
#       * 在 /opt/hybrid/local.properties 写好 sdk.dir + KEYSTORE_FILE/KEYSTORE_PASSWORD/
#         KEY_ALIAS/KEY_PASSWORD（值由 build ARG 注入），供 Gradle signingConfigs.release
#         直接读（app/build.gradle:101-108）。
#   keystore 与口令【不再】运行时注入、不进 .env、不挂 docker secret。
#   仍不进 git / DB / 配置 API / 对象存储 / 前端。
#   注：仓库工作区是运行时卷（/workspace），entrypoint 会把烧入的 local.properties
#       放进 checkout 后的仓库根（见 builder-entrypoint.sh）。
#
# 我方 CI 构建本镜像时需提供（运维 pull 成品镜像，无需任何 args）：
#   --build-arg KEYSTORE_PASSWORD=... --build-arg KEY_ALIAS=... --build-arg KEY_PASSWORD=...
#   并把 keystore 放到构建上下文 deploy/secrets/release.keystore（gitignore，CI 注入）。
#
# 多签名 key（2026-09-01，ADR-0016）：一批已上架商店的渠道（ap01018~ap01022、gzmkt031）当年是用
#   另一把 key（gzmkt031-key.jks，证书 CN=empty-app）签的，商店按包名绑定证书不能变。Gradle 仍只认
#   上面那把默认 key（护栏 #1 不动 Gradle）；额外的 key 全部烧进本镜像，并写一份「签名 key 注册表」
#   /opt/hybrid/signing-keys.properties（<id>.file/.alias/.storePassword/.keyPassword），runner 在
#   打包完成后按渠道 signingKey（Console 下发，仅是 ID）用 apksigner 重签（v1+v2）再投递。
#   → 需额外提供：--build-arg STORE_EMPTYAPP_KEYSTORE_PASSWORD/STORE_EMPTYAPP_KEY_ALIAS/STORE_EMPTYAPP_KEY_PASSWORD
#     并把 keystore 放到 deploy/secrets/store-emptyapp.keystore（gitignore）。
#   新增一把 key = 这里多 COPY 一份 + 注册表多一段 + server 注册表（model.SigningKeys）多一条。
#
# 工具链对齐现有工程：AGP 8.12 / Gradle 8.13 / compileSdk 36 / minSdk 29 / JDK 17。
#
# 单独构建（自测，可选）：
#   docker build -f deploy/Dockerfile.builder \
#     --build-arg KEYSTORE_PASSWORD=xxx --build-arg KEY_ALIAS=xxx --build-arg KEY_PASSWORD=xxx \
#     -t hybrid-builder .
# ============================================================================

# ---- 阶段一：编译 hybrid-pack（Go CLI，全静态）----
FROM golang:1.25-bookworm AS gobuild
WORKDIR /src
COPY cli/go.mod cli/go.sum ./cli/
RUN cd cli && go mod download
COPY cli/ ./cli/
# version 通过 -ldflags 注入（root.go 的 var version）。
RUN cd cli \
    && CGO_ENABLED=0 GOOS=linux GOFLAGS=-mod=mod \
       go build -trimpath -ldflags="-s -w -X main.version=server-runner" \
       -o /out/hybrid-pack ./cmd/hybrid-pack

# ---- 阶段二：Android 构建机 ----
FROM eclipse-temurin:17-jdk-jammy

# Android SDK 版本（与 app 工程对齐；如需升级在此一处改）。
ARG ANDROID_CMDLINE_TOOLS_VERSION=11076708
ARG ANDROID_PLATFORM=android-36
# build-tools 必须匹配 AGP 8.12 实际选用的版本（= 35.0.0，与本机一致）；装错版本会在
# R8/minify 阶段尝试联网自下且因 SDK 目录只读而失败（评审 D7）。
ARG ANDROID_BUILD_TOOLS=35.0.0

ENV ANDROID_HOME=/opt/android-sdk \
    ANDROID_SDK_ROOT=/opt/android-sdk \
    GRADLE_USER_HOME=/home/builder/.gradle \
    LANG=C.UTF-8 \
    DEBIAN_FRONTEND=noninteractive

# 基础依赖：unzip/curl 装 SDK；git 用于在仓库卷上做 checkout/更新（runner 需要）。
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates curl unzip git \
    && rm -rf /var/lib/apt/lists/*

# 安装 Android cmdline-tools。
# 官方要求放到 $ANDROID_HOME/cmdline-tools/latest/ 下，sdkmanager 才能找到平台目录。
RUN set -eux; \
    mkdir -p "${ANDROID_HOME}/cmdline-tools"; \
    curl -fsSL -o /tmp/cmdline-tools.zip \
        "https://dl.google.com/android/repository/commandlinetools-linux-${ANDROID_CMDLINE_TOOLS_VERSION}_latest.zip"; \
    unzip -q /tmp/cmdline-tools.zip -d /tmp/cmdline-tools; \
    mv /tmp/cmdline-tools/cmdline-tools "${ANDROID_HOME}/cmdline-tools/latest"; \
    rm -rf /tmp/cmdline-tools.zip /tmp/cmdline-tools

ENV PATH="${ANDROID_HOME}/cmdline-tools/latest/bin:${ANDROID_HOME}/platform-tools:${PATH}"

# 接受 license 并安装 platform-tools + 指定平台 + build-tools。
RUN set -eux; \
    yes | sdkmanager --licenses >/dev/null; \
    sdkmanager --install \
        "platform-tools" \
        "platforms;${ANDROID_PLATFORM}" \
        "build-tools;${ANDROID_BUILD_TOOLS}"; \
    # 清掉下载缓存，缩小镜像。
    rm -rf "${ANDROID_HOME}/.downloadIntermediates" || true

# 放入 hybrid-pack 二进制。
COPY --from=gobuild /out/hybrid-pack /usr/local/bin/hybrid-pack

# 入口脚本：把烧入的 local.properties 放进 checkout 后的仓库根 → 跑 runner。
COPY deploy/builder-entrypoint.sh /usr/local/bin/builder-entrypoint.sh
RUN chmod +x /usr/local/bin/builder-entrypoint.sh

# 非 root 运行；仓库卷与产物卷由 compose 卷挂载，属主在 entrypoint 中适配。
RUN useradd -m -u 1000 builder \
    && mkdir -p /workspace /apks /opt/hybrid "${GRADLE_USER_HOME}" \
    && chown -R builder:builder /workspace /apks /home/builder "${ANDROID_HOME}"
# 把 SDK 目录归属 builder：让 AGP 在需要时能自动补装组件（否则 root 属主只读 → 构建失败，评审 D7）。

# -------------------------------------------------------------------------
# keystore 烧入镜像（运营决定「直接内置、不考虑安全」；私有 registry）。
#   签名口令由 build ARG 注入（仅我方 CI build 时提供；运维 pull 无需 args）。
# -------------------------------------------------------------------------
ARG KEYSTORE_PASSWORD=""
ARG KEY_ALIAS=""
ARG KEY_PASSWORD=""
# 商店老 key（empty-app，ADR-0016）：口令同样由 build ARG 注入。
ARG STORE_EMPTYAPP_KEYSTORE_PASSWORD=""
ARG STORE_EMPTYAPP_KEY_ALIAS=""
ARG STORE_EMPTYAPP_KEY_PASSWORD=""

# keystore 本体：CI 把它放到 deploy/secrets/release.keystore（gitignore），COPY 进固定路径。
COPY deploy/secrets/release.keystore /opt/hybrid/release.keystore
# 第二把 key：deploy/secrets/store-emptyapp.keystore（gitignore）→ 固定路径，仅 runner 重签时用。
COPY deploy/secrets/store-emptyapp.keystore /opt/hybrid/store-emptyapp.keystore

# 写好 local.properties（含 sdk.dir + 签名四项），供 Gradle signingConfigs.release 直接读。
#   entrypoint 在仓库 checkout 后把它放到 ${REPO_DIR}/local.properties（仓库是运行时卷）。
RUN set -eux; \
    { \
      echo "sdk.dir=${ANDROID_HOME}"; \
      echo "KEYSTORE_FILE=/opt/hybrid/release.keystore"; \
      echo "KEYSTORE_PASSWORD=${KEYSTORE_PASSWORD}"; \
      echo "KEY_ALIAS=${KEY_ALIAS}"; \
      echo "KEY_PASSWORD=${KEY_PASSWORD}"; \
    } > /opt/hybrid/local.properties; \
    { \
      echo "# 签名 key 注册表（ADR-0016）：runner 按渠道 signingKey 查此表用 apksigner 重签。"; \
      echo "# 默认 key（Gradle signingConfigs.release）不在此表；此处只放「非默认」的 key。"; \
      echo "emptyapp.file=/opt/hybrid/store-emptyapp.keystore"; \
      echo "emptyapp.alias=${STORE_EMPTYAPP_KEY_ALIAS}"; \
      echo "emptyapp.storePassword=${STORE_EMPTYAPP_KEYSTORE_PASSWORD}"; \
      echo "emptyapp.keyPassword=${STORE_EMPTYAPP_KEY_PASSWORD}"; \
    } > /opt/hybrid/signing-keys.properties; \
    chown -R builder:builder /opt/hybrid; \
    chmod 0600 /opt/hybrid/release.keystore /opt/hybrid/local.properties \
               /opt/hybrid/store-emptyapp.keystore /opt/hybrid/signing-keys.properties

# runner 读注册表的路径（可被 compose 环境覆盖）。
ENV HYBRID_PACK_SIGNING_KEYS=/opt/hybrid/signing-keys.properties

WORKDIR /workspace
USER builder

# 仓库卷（hybrid-pack 在其上 pull→assemble）与产物卷。
VOLUME ["/workspace", "/apks"]

ENTRYPOINT ["/usr/local/bin/builder-entrypoint.sh"]
