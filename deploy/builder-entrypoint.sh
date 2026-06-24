#!/usr/bin/env bash
# ============================================================================
# deploy/builder-entrypoint.sh —— 构建机入口
#
# 职责（启动构建机容器时）：
#   1) 确认仓库工作区就绪（仓库卷 /workspace；首次为空时由 runner/运维 git clone）。
#   2) 把【镜像内烧好的】local.properties 放进仓库根（sdk.dir + 签名四项已在构建期写好，
#      值来自 build ARG；keystore 本体也已 COPY 进镜像 /opt/hybrid/release.keystore）。
#      —— 仓库是运行时卷，故文件需在 checkout 后落位；本步骤不含任何密钥/口令逻辑。
#   3) exec hybrid-pack runner：从后端 build-job 队列取活，pull→assemble→签名→产物落 /apks。
#
# keystore（ADR-0008/0010 + docs/admin/06 §4）：直接内置镜像，运行时零注入；
#   本脚本【不再】写口令、不再读 docker secret（运营决定「直接内置、不考虑安全」）。
#
# 设计：尽量「无副作用 + 失败即响亮」。所有可调项走环境变量（见 deploy/.env.console.example）。
# ============================================================================
set -euo pipefail

log() { printf '[builder] %s\n' "$*" >&2; }

WORKSPACE="${BUILDER_WORKSPACE:-/workspace}"
REPO_DIR="${BUILDER_REPO_DIR:-${WORKSPACE}/hybrid-app}"
APKS_DIR="${APKS_DIR:-/apks}"

# 镜像内烧好的 local.properties（构建期写入；含 sdk.dir + 签名四项）。
BAKED_LOCAL_PROPERTIES="/opt/hybrid/local.properties"

# ---- 仓库工作区 ----
if [ ! -d "${REPO_DIR}/.git" ]; then
  if [ -n "${BUILDER_REPO_URL:-}" ]; then
    if [ -z "${BUILDER_REPO_BRANCH:-}" ]; then
      log "警告：未设置 BUILDER_REPO_BRANCH，将 clone 默认分支（可能不是发布分支，评审 S3）；建议在 .env 指定。"
    fi
    log "仓库工作区为空，clone ${BUILDER_REPO_URL} (${BUILDER_REPO_BRANCH:-默认分支}) → ${REPO_DIR}"
    git clone --depth 1 ${BUILDER_REPO_BRANCH:+--branch "${BUILDER_REPO_BRANCH}"} \
      "${BUILDER_REPO_URL}" "${REPO_DIR}"
  else
    log "警告：${REPO_DIR} 不是 git 仓库，且未设置 BUILDER_REPO_URL。"
    log "      请把仓库 checkout 挂到仓库卷，或设置 BUILDER_REPO_URL 让 runner 自行 clone。"
  fi
fi

# ---- 放置烧入的 local.properties（供 Gradle 读 SDK 与签名）----
# 仓库是运行时卷，故把镜像内构建期写好的 local.properties 复制进仓库根。
if [ -d "${REPO_DIR}" ]; then
  if [ -f "${BAKED_LOCAL_PROPERTIES}" ]; then
    log "放置烧入的 local.properties → ${REPO_DIR}/local.properties（sdk.dir + 签名四项已内置）"
    cp "${BAKED_LOCAL_PROPERTIES}" "${REPO_DIR}/local.properties"
  else
    log "警告：未找到镜像内 ${BAKED_LOCAL_PROPERTIES}；Release 签名将失败。"
    log "      该文件应在构建本镜像时写入（见 deploy/Dockerfile.builder / docs/admin/06 §4）。"
  fi
fi

# ---- 适配环境变量名（评审 D2）----
# compose/.env 用语义化名（HYBRID_SERVER_URL / HYBRID_RUNNER_TOKEN），而 hybrid-pack runner
# 读 HYBRID_PACK_*；不桥接会导致 runner 找不到后端/令牌而「启动即退出」。在此做一次性映射。
export HYBRID_PACK_SERVER="${HYBRID_PACK_SERVER:-${HYBRID_SERVER_URL:-}}"
export HYBRID_PACK_TOKEN="${HYBRID_PACK_TOKEN:-${HYBRID_RUNNER_TOKEN:-}}"
export HYBRID_PACK_RUNNER_ID="${HYBRID_PACK_RUNNER_ID:-${HOSTNAME:-runner}}"

# ---- 启动 runner ----
# 产物目录/下载前缀（评审 D3）：runner 默认写 /var/www/apks，与 nginx 共享卷 /apks 不一致；
# 显式对齐到 ${APKS_DIR}（compose=/apks），下载前缀默认 /apks（对应 nginx 的 /apks/ 路由）。
export APKS_DIR REPO_DIR
ARTIFACT_BASE_URL="${APKS_BASE_URL:-/apks}"
# hybrid-pack 的 mustRepo 从【当前目录向上】找仓库根（settings.gradle + channels/）；
# 仓库在子目录 ${REPO_DIR}，不先 cd 进去会「向上找不到」而启动即退出（评审 D6）。
cd "${REPO_DIR}" || { log "错误：仓库目录 ${REPO_DIR} 不存在，无法构建"; exit 1; }
log "启动 hybrid-pack runner（server=${HYBRID_PACK_SERVER:-<unset>} apks=${APKS_DIR} base=${ARTIFACT_BASE_URL} repo=${REPO_DIR}）"
exec hybrid-pack runner \
  --artifact-dir "${APKS_DIR}" \
  --artifact-base-url "${ARTIFACT_BASE_URL}" \
  "$@"
